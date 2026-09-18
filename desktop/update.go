package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type UpdateManifest struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Notes       string `json:"notes"`
	PublishedAt string `json:"published_at"`
}

type UpdateInfo struct {
	CurrentVersion   string `json:"current_version"`
	AvailableVersion string `json:"available_version"`
	Available        bool   `json:"available"`
	DownloadURL      string `json:"download_url"`
	Notes            string `json:"notes"`
	Message          string `json:"message"`
}

func (a *App) CheckUpdates() UpdateInfo {
	out := UpdateInfo{CurrentVersion: AppVersion, Message: "Você está usando a versão " + AppVersion + "."}
	manifestURL := strings.TrimSpace(a.GetSetting("update_manifest_url"))
	if manifestURL == "" {
		out.Message = "Atualização automática preparada. O canal estável será ativado quando publicarmos o primeiro servidor de versões."
		return out
	}
	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		out.Message = err.Error()
		return out
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		out.Message = "Não foi possível verificar atualizações agora."
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		out.Message = fmt.Sprintf("Servidor de atualização respondeu HTTP %d.", resp.StatusCode)
		return out
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		out.Message = err.Error()
		return out
	}
	var m UpdateManifest
	if err := json.Unmarshal(body, &m); err != nil {
		out.Message = "Manifesto de atualização inválido."
		return out
	}
	out.AvailableVersion = m.Version
	out.DownloadURL = m.DownloadURL
	out.Notes = m.Notes
	out.Available = versionGreater(m.Version, AppVersion)
	if out.Available {
		out.Message = "Nova versão " + m.Version + " disponível."
	} else {
		out.Message = "O Via Verde CAR está atualizado."
	}
	return out
}

func (a *App) OpenUpdateDownload(url string) error {
	if a.ctx == nil {
		return errors.New("aplicativo ainda não inicializado")
	}
	url = strings.TrimSpace(url)
	if !(strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")) {
		return errors.New("endereço de atualização inválido")
	}
	runtime.BrowserOpenURL(a.ctx, url)
	return nil
}

func versionGreater(a, b string) bool {
	pa := parseVersion(a)
	pb := parseVersion(b)
	for i := 0; i < 3; i++ {
		if pa[i] > pb[i] {
			return true
		}
		if pa[i] < pb[i] {
			return false
		}
	}
	return false
}
func parseVersion(v string) [3]int {
	v = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(v), "v"))
	var out [3]int
	fmt.Sscanf(v, "%d.%d.%d", &out[0], &out[1], &out[2])
	return out
}
