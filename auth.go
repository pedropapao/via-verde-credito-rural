package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const pbkdf2Iterations = 210000

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func sha256Hex(v string) string {
	h := sha256.Sum256([]byte(v))
	return hex.EncodeToString(h[:])
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	hLen := 32
	blocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, blocks*hLen)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func HashPassword(password string) (string, error) {
	if len(password) < 10 {
		return "", errors.New("a senha deve ter pelo menos 10 caracteres")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2SHA256([]byte(password), salt, pbkdf2Iterations, 32)
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", pbkdf2Iterations, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got := pbkdf2SHA256([]byte(password), salt, iter, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

type ctxKey string

const (
	ctxUser    ctxKey = "user"
	ctxSession ctxKey = "session"
)

func userFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(ctxUser).(*User)
	return u
}
func sessionFromContext(ctx context.Context) *Session {
	s, _ := ctx.Value(ctxSession).(*Session)
	return s
}

func (a *App) createSession(ctx context.Context, userID string) (string, *Session, error) {
	raw, err := randomToken(32)
	if err != nil {
		return "", nil, err
	}
	csrf, err := randomToken(24)
	if err != nil {
		return "", nil, err
	}
	expires := time.Now().UTC().Add(time.Duration(a.cfg.SessionDays) * 24 * time.Hour).Format(time.RFC3339)
	s := Session{UserID: userID, TokenHash: sha256Hex(raw), CSRFToken: csrf, ExpiresAt: expires}
	var rows []Session
	if err := a.sb.Insert(ctx, "sessions", s, &rows); err != nil {
		return "", nil, err
	}
	if len(rows) == 0 {
		return "", nil, errors.New("não foi possível criar a sessão")
	}
	return raw, &rows[0], nil
}

func (a *App) loadAuth(r *http.Request) (*User, *Session, error) {
	c, err := r.Cookie("vv_session")
	if err != nil || c.Value == "" {
		return nil, nil, errors.New("sem sessão")
	}
	var ss []Session
	if err := a.sb.Select(r.Context(), "sessions", eq("token_hash", sha256Hex(c.Value))+"&limit=1", &ss); err != nil || len(ss) == 0 {
		return nil, nil, errors.New("sessão inválida")
	}
	s := ss[0]
	exp, err := time.Parse(time.RFC3339, s.ExpiresAt)
	if err != nil || time.Now().UTC().After(exp) {
		return nil, &s, errors.New("sessão expirada")
	}
	var us []User
	if err := a.sb.Select(r.Context(), "users", eq("id", s.UserID)+"&limit=1", &us); err != nil || len(us) == 0 {
		return nil, &s, errors.New("usuário não encontrado")
	}
	if !us[0].Active {
		return nil, &s, errors.New("usuário desativado")
	}
	return &us[0], &s, nil
}

func (a *App) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, s, err := a.loadAuth(r)
		if err != nil {
			http.Redirect(w, r, "/login?next="+urlQuery(r.URL.RequestURI()), http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUser, u)
		ctx = context.WithValue(ctx, ctxSession, s)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) ownerOnly(next http.Handler) http.Handler {
	return a.withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		if u == nil || u.Role != "owner" {
			http.Error(w, "Acesso somente para o proprietário do sistema.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (a *App) verifyCSRF(r *http.Request) bool {
	s := sessionFromContext(r.Context())
	if s == nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(r.FormValue("csrf")), []byte(s.CSRFToken)) == 1
}

func (a *App) setSessionCookie(w http.ResponseWriter, r *http.Request, raw string) {
	secure := a.cfg.AppEnv == "production" || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{Name: "vv_session", Value: raw, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: a.cfg.SessionDays * 24 * 3600})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "vv_session", Value: "", Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
}

func urlQuery(v string) string {
	r := strings.NewReplacer("%", "%25", " ", "%20", "?", "%3F", "&", "%26", "=", "%3D", "#", "%23")
	return r.Replace(v)
}
