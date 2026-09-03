package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Supabase struct {
	BaseURL string
	Key     string
	Client  *http.Client
}

func NewSupabase(baseURL, key string) *Supabase {
	return &Supabase{BaseURL: strings.TrimRight(baseURL, "/"), Key: key, Client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *Supabase) setAuthHeaders(req *http.Request) {
	req.Header.Set("apikey", s.Key)
	// As chaves novas sb_secret_* são opacas e devem ser enviadas no header apikey.
	// A service_role legada é um JWT e continua compatível com Authorization: Bearer.
	if !strings.HasPrefix(s.Key, "sb_secret_") {
		req.Header.Set("Authorization", "Bearer "+s.Key)
	}
}

func (s *Supabase) do(ctx context.Context, method, endpoint string, body io.Reader, contentType string, extra map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.BaseURL+endpoint, body)
	if err != nil {
		return nil, 0, err
	}
	s.setAuthHeaders(req)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return b, resp.StatusCode, fmt.Errorf("supabase %s %s: status %d: %s", method, endpoint, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return b, resp.StatusCode, nil
}

func shouldRetrySupabaseRead(status int) bool {
	return status == 0 || status == http.StatusTooManyRequests || status >= 500
}

func (s *Supabase) Select(ctx context.Context, table, query string, out any) error {
	endpoint := "/rest/v1/" + url.PathEscape(table)
	if query != "" {
		endpoint += "?" + query
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		b, status, err := s.do(ctx, http.MethodGet, endpoint, nil, "", nil)
		if err == nil {
			return json.Unmarshal(b, out)
		}
		lastErr = err
		if !shouldRetrySupabaseRead(status) || attempt == 2 {
			return err
		}
		wait := time.Duration(attempt+1) * 250 * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return lastErr
}

func (s *Supabase) Insert(ctx context.Context, table string, value any, out any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	b, _, err := s.do(ctx, http.MethodPost, "/rest/v1/"+url.PathEscape(table), bytes.NewReader(payload), "application/json", map[string]string{"Prefer": "return=representation"})
	if err != nil {
		return err
	}
	if out != nil {
		return json.Unmarshal(b, out)
	}
	return nil
}

func (s *Supabase) Update(ctx context.Context, table, query string, value any, out any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	endpoint := "/rest/v1/" + url.PathEscape(table)
	if query != "" {
		endpoint += "?" + query
	}
	b, _, err := s.do(ctx, http.MethodPatch, endpoint, bytes.NewReader(payload), "application/json", map[string]string{"Prefer": "return=representation"})
	if err != nil {
		return err
	}
	if out != nil {
		return json.Unmarshal(b, out)
	}
	return nil
}

func (s *Supabase) Delete(ctx context.Context, table, query string) error {
	endpoint := "/rest/v1/" + url.PathEscape(table)
	if query != "" {
		endpoint += "?" + query
	}
	_, _, err := s.do(ctx, http.MethodDelete, endpoint, nil, "", nil)
	return err
}

func eq(field, value string) string { return url.QueryEscape(field) + "=eq." + url.QueryEscape(value) }
func order(field string, desc bool) string {
	d := ".asc"
	if desc {
		d = ".desc"
	}
	return "order=" + url.QueryEscape(field+d)
}

func storageEscape(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func (s *Supabase) EnsureBucket(ctx context.Context, bucket string) error {
	payload, _ := json.Marshal(map[string]any{"id": bucket, "name": bucket, "public": false, "file_size_limit": 52428800})
	_, status, err := s.do(ctx, http.MethodPost, "/storage/v1/bucket", bytes.NewReader(payload), "application/json", nil)
	if err != nil && (status == http.StatusConflict || strings.Contains(err.Error(), "already exists")) {
		return nil
	}
	return err
}

func (s *Supabase) Upload(ctx context.Context, bucket, objectPath, contentType string, r io.Reader) error {
	endpoint := "/storage/v1/object/" + url.PathEscape(bucket) + "/" + storageEscape(objectPath)
	_, _, err := s.do(ctx, http.MethodPost, endpoint, r, contentType, map[string]string{"x-upsert": "false"})
	return err
}

func (s *Supabase) Download(ctx context.Context, bucket, objectPath string) ([]byte, string, error) {
	endpoint := "/storage/v1/object/" + url.PathEscape(bucket) + "/" + storageEscape(objectPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	s.setAuthHeaders(req)
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("download status %d: %s", resp.StatusCode, string(b))
	}
	return b, resp.Header.Get("Content-Type"), nil
}

func (s *Supabase) DeleteObject(ctx context.Context, bucket, objectPath string) error {
	payload, _ := json.Marshal(map[string]any{"prefixes": []string{objectPath}})
	_, _, err := s.do(ctx, http.MethodDelete, "/storage/v1/object/"+url.PathEscape(bucket), bytes.NewReader(payload), "application/json", nil)
	return err
}

// kept for future direct multipart integrations; currently uploads are streamed to Storage.
func multipartBuffer(field, filename string, data []byte) (*bytes.Buffer, string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	p, err := w.CreateFormFile(field, path.Base(filename))
	if err != nil {
		return nil, "", err
	}
	if _, err = p.Write(data); err != nil {
		return nil, "", err
	}
	_ = w.Close()
	return &b, w.FormDataContentType(), nil
}
