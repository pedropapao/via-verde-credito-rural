package main

import (
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


func TestValidateUpdateManifest(t *testing.T) {
	base := UpdateManifest{
		Version: "2.0.0",
		DownloadURL: "https://igrxqbroklfwujcwbiwh.supabase.co/storage/v1/object/sign/via-verde-files/desktop-updates/ViaVerdeCAR-2.0.0.exe?token=abc",
		SHA256: strings.Repeat("a", 64),
		Size: 16 * 1024 * 1024,
		Channel: "stable",
	}
	if err := validateUpdateManifest(base); err != nil {
		t.Fatalf("manifesto válido rejeitado: %v", err)
	}

	badHost := base
	badHost.DownloadURL = "https://example.com/ViaVerdeCAR.exe"
	if err := validateUpdateManifest(badHost); err == nil {
		t.Fatal("host externo deveria ser rejeitado")
	}

	badHash := base
	badHash.SHA256 = "1234"
	if err := validateUpdateManifest(badHash); err == nil {
		t.Fatal("hash inválido deveria ser rejeitado")
	}

	badSize := base
	badSize.Size = 100
	if err := validateUpdateManifest(badSize); err == nil {
		t.Fatal("tamanho inválido deveria ser rejeitado")
	}
}
