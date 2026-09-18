package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const updateManifestURL = "https://igrxqbroklfwujcwbiwh.supabase.co/functions/v1/via-verde-desktop-update?action=manifest"

type UpdateManifest struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	Notes       string `json:"notes"`
	PublishedAt string `json:"published_at"`
	Channel     string `json:"channel"`
}

type UpdateInfo struct {
	CurrentVersion   string `json:"current_version"`
	AvailableVersion string `json:"available_version"`
	Available        bool   `json:"available"`
	DownloadURL      string `json:"download_url"`
	SHA256           string `json:"sha256"`
	Size             int64  `json:"size"`
	Notes            string `json:"notes"`
	PublishedAt      string `json:"published_at"`
	Message          string `json:"message"`
}

type UpdateInstallResult struct {
	Started bool   `json:"started"`
	Message string `json:"message"`
}

func (a *App) CheckUpdates() UpdateInfo {
	out := UpdateInfo{CurrentVersion: AppVersion, Message: "Você está usando a versão " + AppVersion + "."}
	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateManifestURL, nil)
	if err != nil {
		out.Message = err.Error()
		return out
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		out.Message = "Não foi possível verificar atualizações agora."
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		out.Message = fmt.Sprintf("Servidor de atualização respondeu HTTP %d.", resp.StatusCode)
		return out
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		out.Message = err.Error()
		return out
	}
	var m UpdateManifest
	if err := json.Unmarshal(body, &m); err != nil {
		out.Message = "Manifesto de atualização inválido."
		return out
	}
	if strings.TrimSpace(m.Version) == "" || strings.TrimSpace(m.DownloadURL) == "" {
		out.Message = "Nenhuma atualização publicada no canal estável."
		return out
	}
	out.AvailableVersion = m.Version
	out.DownloadURL = m.DownloadURL
	out.SHA256 = strings.ToLower(strings.TrimSpace(m.SHA256))
	out.Size = m.Size
	out.Notes = m.Notes
	out.PublishedAt = m.PublishedAt
	out.Available = versionGreater(m.Version, AppVersion)
	if out.Available {
		out.Message = "Nova versão " + m.Version + " disponível."
	} else {
		out.Message = "O Via Verde CAR está atualizado."
	}
	return out
}

// InstallUpdate baixa a nova versão, valida SHA-256, cria backup dos dados,
// prepara um script auxiliar e reinicia o aplicativo. O PowerShell é usado
// apenas para trocar o executável depois que o processo atual encerrar.
func (a *App) InstallUpdate(info UpdateInfo) (UpdateInstallResult, error) {
	if goruntime.GOOS != "windows" {
		return UpdateInstallResult{}, errors.New("atualização automática está disponível somente no Windows nesta versão")
	}
	if a.ctx == nil {
		return UpdateInstallResult{}, errors.New("aplicativo ainda não inicializado")
	}
	if !versionGreater(info.AvailableVersion, AppVersion) {
		return UpdateInstallResult{}, errors.New("não há versão mais nova para instalar")
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(info.DownloadURL)), "https://") {
		return UpdateInstallResult{}, errors.New("endereço de atualização inválido")
	}
	if len(strings.TrimSpace(info.SHA256)) != 64 {
		return UpdateInstallResult{}, errors.New("assinatura SHA-256 da atualização ausente")
	}

	updateDir := filepath.Join(a.dataDir, "updates")
	backupDir := filepath.Join(a.dataDir, "backups")
	if err := os.MkdirAll(updateDir, 0o755); err != nil {
		return UpdateInstallResult{}, err
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return UpdateInstallResult{}, err
	}

	backupPath := filepath.Join(backupDir, "Antes_da_atualizacao_"+time.Now().Format("20060102_150405")+".zip")
	if err := a.writeBackup(backupPath); err != nil {
		return UpdateInstallResult{}, fmt.Errorf("não foi possível criar o backup automático: %w", err)
	}

	staged := filepath.Join(updateDir, "ViaVerdeCAR-"+safeVersionFilename(info.AvailableVersion)+".exe")
	if err := downloadUpdateFile(a.ctx, info.DownloadURL, staged, info.Size); err != nil {
		return UpdateInstallResult{}, err
	}
	gotSHA, err := fileSHA256(staged)
	if err != nil {
		return UpdateInstallResult{}, err
	}
	if !strings.EqualFold(gotSHA, info.SHA256) {
		_ = os.Remove(staged)
		return UpdateInstallResult{}, errors.New("a atualização baixada falhou na verificação de integridade SHA-256")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return UpdateInstallResult{}, err
	}
	currentExe, _ = filepath.Abs(currentExe)
	staged, _ = filepath.Abs(staged)
	oldExe := filepath.Join(updateDir, "ViaVerdeCAR-versao-anterior.exe")
	scriptPath := filepath.Join(updateDir, "aplicar-atualizacao.ps1")
	logPath := filepath.Join(updateDir, "ultima-atualizacao.log")

	if err := os.WriteFile(scriptPath, []byte(buildWindowsUpdateScript()), 0o644); err != nil {
		return UpdateInstallResult{}, err
	}
	ps, err := exec.LookPath("powershell.exe")
	if err != nil {
		return UpdateInstallResult{}, errors.New("PowerShell do Windows não encontrado")
	}
	cmd := exec.Command(ps,
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-WindowStyle", "Hidden",
		"-File", scriptPath,
		"-TargetPid", fmt.Sprint(os.Getpid()),
		"-CurrentExe", currentExe,
		"-NewExe", staged,
		"-OldExe", oldExe,
		"-LogFile", logPath,
	)
	if err := cmd.Start(); err != nil {
		return UpdateInstallResult{}, fmt.Errorf("não foi possível iniciar o atualizador: %w", err)
	}

	go func() {
		time.Sleep(1200 * time.Millisecond)
		wailsruntime.Quit(a.ctx)
	}()
	return UpdateInstallResult{Started: true, Message: "Atualização " + info.AvailableVersion + " validada. O Via Verde será reiniciado automaticamente."}, nil
}

func downloadUpdateFile(ctx context.Context, url, target string, expectedSize int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return fmt.Errorf("falha ao baixar atualização: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("servidor de atualização respondeu HTTP %d", resp.StatusCode)
	}
	tmp := target + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, 80<<20))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n < 1024*1024 {
		_ = os.Remove(tmp)
		return errors.New("arquivo de atualização recebido é pequeno demais")
	}
	if expectedSize > 0 && n != expectedSize {
		_ = os.Remove(tmp)
		return fmt.Errorf("tamanho da atualização divergente: recebido %d bytes, esperado %d", n, expectedSize)
	}
	_ = os.Remove(target)
	return os.Rename(tmp, target)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func safeVersionFilename(v string) string {
	v = strings.TrimSpace(v)
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return r.Replace(v)
}

func buildWindowsUpdateScript() string {
	return `param(
  [int]$TargetPid,
  [string]$CurrentExe,
  [string]$NewExe,
  [string]$OldExe,
  [string]$LogFile
)
$ErrorActionPreference = "Stop"
function Log([string]$m) {
  $line = (Get-Date -Format "yyyy-MM-dd HH:mm:ss") + " " + $m
  Add-Content -LiteralPath $LogFile -Value $line -Encoding UTF8
}
try {
  Log "Aguardando Via Verde encerrar (PID $TargetPid)."
  for ($i = 0; $i -lt 120; $i++) {
    if (-not (Get-Process -Id $TargetPid -ErrorAction SilentlyContinue)) { break }
    Start-Sleep -Milliseconds 250
  }
  if (Get-Process -Id $TargetPid -ErrorAction SilentlyContinue) {
    throw "O processo antigo não encerrou a tempo."
  }

  if (Test-Path -LiteralPath $OldExe) { Remove-Item -LiteralPath $OldExe -Force }
  if (Test-Path -LiteralPath $CurrentExe) { Copy-Item -LiteralPath $CurrentExe -Destination $OldExe -Force }
  Copy-Item -LiteralPath $NewExe -Destination $CurrentExe -Force
  Log "Executável atualizado com sucesso."
  Start-Process -FilePath $CurrentExe
  Start-Sleep -Seconds 2
  Remove-Item -LiteralPath $NewExe -Force -ErrorAction SilentlyContinue
  Log "Nova versão iniciada."
} catch {
  Log ("ERRO: " + $_.Exception.Message)
  try {
    if (Test-Path -LiteralPath $OldExe) {
      Copy-Item -LiteralPath $OldExe -Destination $CurrentExe -Force
      Start-Process -FilePath $CurrentExe
      Log "Versão anterior restaurada."
    }
  } catch {
    Log ("ERRO AO RESTAURAR: " + $_.Exception.Message)
  }
}
`
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
