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
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	updateManifestURL       = "https://igrxqbroklfwujcwbiwh.supabase.co/functions/v1/via-verde-desktop-update?action=manifest"
	updateDownloadHost      = "igrxqbroklfwujcwbiwh.supabase.co"
	updateDownloadPathRoot  = "/storage/v1/object/sign/via-verde-files/desktop-updates/"
	updatePublicPathRoot    = "/storage/v1/object/public/via-verde-files/desktop-updates/"
	updateMaxDownloadSize   = 80 << 20
	updateMinDownloadSize   = 1 << 20
)

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

	m, err := fetchOfficialUpdateManifest(ctx)
	if err != nil {
		out.Message = "Não foi possível verificar atualizações agora."
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

func fetchOfficialUpdateManifest(ctx context.Context) (UpdateManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateManifestURL, nil)
	if err != nil {
		return UpdateManifest{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return UpdateManifest{}, fmt.Errorf("manifesto de atualização indisponível: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return UpdateManifest{}, fmt.Errorf("servidor de atualização respondeu HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return UpdateManifest{}, err
	}
	var m UpdateManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return UpdateManifest{}, errors.New("manifesto de atualização inválido")
	}
	if err := validateOfficialUpdateManifest(m); err != nil {
		return UpdateManifest{}, err
	}
	m.SHA256 = strings.ToLower(strings.TrimSpace(m.SHA256))
	m.Version = strings.TrimSpace(m.Version)
	m.DownloadURL = strings.TrimSpace(m.DownloadURL)
	m.Channel = strings.TrimSpace(m.Channel)
	return m, nil
}

func validateOfficialUpdateManifest(m UpdateManifest) error {
	if !isStrictReleaseVersion(m.Version) {
		return errors.New("manifesto de atualização com versão inválida")
	}
	sha := strings.ToLower(strings.TrimSpace(m.SHA256))
	decoded, err := hex.DecodeString(sha)
	if err != nil || len(decoded) != sha256.Size {
		return errors.New("manifesto de atualização com SHA-256 inválido")
	}
	if m.Size < updateMinDownloadSize || m.Size > updateMaxDownloadSize {
		return errors.New("manifesto de atualização com tamanho inválido")
	}
	channel := strings.ToLower(strings.TrimSpace(m.Channel))
	if channel != "" && channel != "stable" {
		return errors.New("manifesto de atualização fora do canal estável")
	}
	if err := validateOfficialUpdateURL(m.DownloadURL); err != nil {
		return err
	}
	u, _ := neturl.Parse(strings.TrimSpace(m.DownloadURL))
	expectedName := "ViaVerdeCAR-" + safeVersionFilename(strings.TrimSpace(m.Version)) + ".exe"
	actualName := u.Path
	if idx := strings.LastIndex(actualName, "/"); idx >= 0 {
		actualName = actualName[idx+1:]
	}
	if actualName != expectedName {
		return errors.New("arquivo do manifesto não corresponde à versão declarada")
	}
	return nil
}

func validateOfficialUpdateURL(raw string) error {
	u, err := neturl.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.User != nil {
		return errors.New("endereço de atualização inválido")
	}
	if !strings.EqualFold(u.Hostname(), updateDownloadHost) {
		return errors.New("origem da atualização não autorizada")
	}
	if port := u.Port(); port != "" && port != "443" {
		return errors.New("porta da atualização não autorizada")
	}
	path := u.EscapedPath()
	if !strings.HasPrefix(path, updateDownloadPathRoot) &&
		!strings.HasPrefix(path, updatePublicPathRoot) {
		return errors.New("caminho da atualização não autorizado")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".exe") {
		return errors.New("arquivo de atualização inválido")
	}
	return nil
}

func isStrictReleaseVersion(v string) bool {
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || strconv.Itoa(n) != part {
			return false
		}
	}
	return true
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

	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	manifest, err := fetchOfficialUpdateManifest(ctx)
	if err != nil {
		return UpdateInstallResult{}, fmt.Errorf("não foi possível validar a atualização no canal oficial: %w", err)
	}
	if !versionGreater(manifest.Version, AppVersion) {
		return UpdateInstallResult{}, errors.New("não há versão mais nova para instalar")
	}
	if requested := strings.TrimSpace(info.AvailableVersion); requested != "" && requested != manifest.Version {
		return UpdateInstallResult{}, errors.New("a versão disponível mudou; verifique as atualizações novamente")
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

	staged := filepath.Join(updateDir, "ViaVerdeCAR-"+safeVersionFilename(manifest.Version)+".exe")
	if err := downloadUpdateFile(a.ctx, manifest.DownloadURL, staged, manifest.Size); err != nil {
		return UpdateInstallResult{}, err
	}
	gotSHA, err := fileSHA256(staged)
	if err != nil {
		return UpdateInstallResult{}, err
	}
	if !strings.EqualFold(gotSHA, manifest.SHA256) {
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
	hideExternalProcessWindow(cmd)
	if err := cmd.Start(); err != nil {
		return UpdateInstallResult{}, fmt.Errorf("não foi possível iniciar o atualizador: %w", err)
	}

	go func() {
		time.Sleep(1200 * time.Millisecond)
		wailsruntime.Quit(a.ctx)
	}()
	return UpdateInstallResult{Started: true, Message: "Atualização " + manifest.Version + " validada. O Via Verde será reiniciado automaticamente."}, nil
}

func downloadUpdateFile(ctx context.Context, url, target string, expectedSize int64) error {
	if err := validateOfficialUpdateURL(url); err != nil {
		return err
	}
	if expectedSize < updateMinDownloadSize || expectedSize > updateMaxDownloadSize {
		return errors.New("tamanho esperado da atualização inválido")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	client := &http.Client{
		Timeout: 5 * time.Minute,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			return validateOfficialUpdateURL(req.URL.String())
		},
	}
	resp, err := client.Do(req)
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
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, updateMaxDownloadSize+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n < updateMinDownloadSize || n > updateMaxDownloadSize {
		_ = os.Remove(tmp)
		return errors.New("tamanho do arquivo de atualização fora do limite permitido")
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
