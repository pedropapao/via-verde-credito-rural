package main

import (
	"os"
	"strings"
	"testing"
)

func TestVersionGreater(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.0.2", "1.0.1", true},
		{"1.1.0", "1.0.9", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.1", "1.0.1", false},
		{"1.0.0", "1.0.1", false},
	}
	for _, tc := range cases {
		if got := versionGreater(tc.a, tc.b); got != tc.want {
			t.Fatalf("versionGreater(%q,%q)=%v want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestWindowsUpdaterUsesTargetPid(t *testing.T) {
	s := buildWindowsUpdateScript()
	if !strings.Contains(s, "$TargetPid") {
		t.Fatal("script precisa usar $TargetPid")
	}
	if strings.Contains(s, "Get-Process -Id $Pid") {
		t.Fatal("script não pode usar $Pid: PID é variável automática do PowerShell")
	}
}

func TestSafeVersionFilename(t *testing.T) {
	if got := safeVersionFilename("1.2.3 beta/rc"); got != "1.2.3_beta_rc" {
		t.Fatalf("safeVersionFilename=%q", got)
	}
}


func TestValidateOfficialUpdateManifest196(t *testing.T) {
	valid := UpdateManifest{
		Version: "1.9.6",
		DownloadURL: "https://igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/sign/via-verde-files/desktop-updates/ViaVerdeCAR-1.9.6.exe?token=abc",
		SHA256: strings.Repeat("a", 64),
		Size: 16 * 1024 * 1024,
		Channel: "stable",
	}
	if err := validateOfficialUpdateManifest(valid); err != nil {
		t.Fatalf("manifesto oficial válido foi rejeitado: %v", err)
	}

	bad := valid
	bad.DownloadURL = "https://example.com/ViaVerdeCAR-1.9.6.exe"
	if err := validateOfficialUpdateManifest(bad); err == nil {
		t.Fatal("host externo não autorizado deveria ser rejeitado")
	}

	bad = valid
	bad.DownloadURL = "https://igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/sign/via-verde-files/outros/ViaVerdeCAR-1.9.6.exe?token=abc"
	if err := validateOfficialUpdateManifest(bad); err == nil {
		t.Fatal("caminho fora de desktop-updates deveria ser rejeitado")
	}

	bad = valid
	bad.SHA256 = strings.Repeat("z", 64)
	if err := validateOfficialUpdateManifest(bad); err == nil {
		t.Fatal("SHA-256 não hexadecimal deveria ser rejeitado")
	}

	bad = valid
	bad.Size = 100
	if err := validateOfficialUpdateManifest(bad); err == nil {
		t.Fatal("arquivo pequeno demais deveria ser rejeitado")
	}

	bad = valid
	bad.Channel = "beta"
	if err := validateOfficialUpdateManifest(bad); err == nil {
		t.Fatal("canal não estável deveria ser rejeitado")
	}

	bad = valid
	bad.Version = "1.9.6-beta"
	if err := validateOfficialUpdateManifest(bad); err == nil {
		t.Fatal("versão fora do formato x.y.z deveria ser rejeitada")
	}

	bad = valid
	bad.DownloadURL = "https://igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/sign/via-verde-files/desktop-updates/ViaVerdeCAR-9.9.9.exe?token=abc"
	if err := validateOfficialUpdateManifest(bad); err == nil {
		t.Fatal("arquivo incompatível com a versão do manifesto deveria ser rejeitado")
	}
}

func TestValidateOfficialUpdateURL196(t *testing.T) {
	allowed := []string{
		"https://igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/sign/via-verde-files/desktop-updates/ViaVerdeCAR-1.9.6.exe?token=abc",
		"https://igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/public/via-verde-files/desktop-updates/ViaVerdeCAR-1.9.6.exe",
	}
	for _, raw := range allowed {
		if err := validateOfficialUpdateURL(raw); err != nil {
			t.Fatalf("URL oficial rejeitada %q: %v", raw, err)
		}
	}
	rejected := []string{
		"http://igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/sign/via-verde-files/desktop-updates/x.exe",
		"https://evil.example/storage/v1/object/sign/via-verde-files/desktop-updates/x.exe",
		"https://igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/sign/via-verde-files/desktop-updates/x.zip",
		"https://user:pass@igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/sign/via-verde-files/desktop-updates/x.exe",
		"https://igrxqbroklfwujcwbiwh.supabase.co:8443/storage/v1/object/sign/via-verde-files/desktop-updates/x.exe",
	}
	for _, raw := range rejected {
		if err := validateOfficialUpdateURL(raw); err == nil {
			t.Fatalf("URL não confiável deveria ser rejeitada: %q", raw)
		}
	}
}

func TestLeafletIsBundledLocally196(t *testing.T) {
	html, err := os.ReadFile("frontend/dist/index.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(html)
	if strings.Contains(text, "unpkg.com/leaflet") ||
		strings.Contains(text, "cdn.jsdelivr.net/npm/leaflet") {
		t.Fatal("index.html ainda carrega Leaflet remoto")
	}
	for _, want := range []string{
		`href="vendor/leaflet/leaflet.css"`,
		`src="vendor/leaflet/leaflet.js"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("index.html não contém %s", want)
		}
	}
	for _, path := range []string{
		"frontend/dist/vendor/leaflet/leaflet.js",
		"frontend/dist/vendor/leaflet/leaflet.css",
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Leaflet local ausente em %s: %v", path, err)
		}
		if info.Size() < 10*1024 {
			t.Fatalf("arquivo Leaflet local suspeitamente pequeno: %s (%d bytes)", path, info.Size())
		}
	}
}
