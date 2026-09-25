package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseVersion206(t *testing.T) {
	if AppVersion != "2.0.6" {
		t.Fatalf("AppVersion inesperada: %s", AppVersion)
	}
	b, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"productVersion": "2.0.6"`) {
		t.Fatal("wails.json não está alinhado com a versão 2.0.6")
	}
}

func TestReleaseFirePipeline191(t *testing.T) {
	if fireStatusFound == "" || fireStatusNone == "" || fireStatusUnavailable == "" || fireStatusNotRun == "" {
		t.Fatal("estados da consulta de focos não podem ficar vazios")
	}
}

func TestReleaseLegacyShellAssetsRemainVersioned(t *testing.T) {
	for _, path := range []string{
		"frontend/dist/ui201.css", "frontend/dist/ui201.js",
		"frontend/dist/ui202.css", "frontend/dist/ui202.js",
	} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("asset legado ausente: %s", path)
		}
	}
}

func TestRelease204LoadsOnlyOperationalShell(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	for _, asset := range []string{"ui200.js", "ui200.css", "ui201.js", "ui201.css", "ui202.js", "ui202.css"} {
		if strings.Contains(html, asset) {
			t.Fatalf("shell legado não deve ser carregado na 2.0.4: %s", asset)
		}
	}
	for _, asset := range []string{"ui204.js", "ui204.css"} {
		if !strings.Contains(html, asset) {
			t.Fatalf("shell 2.0.4 obrigatório não carregado: %s", asset)
		}
	}
}

func TestRelease204UsesSVGAndSingleScreenTabs(t *testing.T) {
	jsb, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(jsb)
	for _, marker := range []string{
		"<svg class=\"vv204-icon",
		"Resumo",
		"Ambiental",
		"Fundiário",
		"Crédito Rural",
		"Mapa",
		"Documentos",
		"Relatórios",
		"BANCO CENTRAL",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface 2.0.4 sem marcador %q", marker)
		}
	}
	cssb, err := os.ReadFile("frontend/dist/ui204.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(cssb)
	if !strings.Contains(css, "body.vv204 .sidebar{display:none!important}") {
		t.Fatal("interface 2.0.4 deve remover a sidebar fixa da experiência principal")
	}
}

func TestRelease203BCBModulePreserved(t *testing.T) {
	for _, path := range []string{"bcb_public.go", "bcb_public_test.go"} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("módulo BCB ausente: %s", path)
		}
	}
}


func TestRelease205ProfessionalPDFsPreserved(t *testing.T) {
	if st, err := os.Stat("professional_reports.go"); err != nil || st.Size() == 0 {
		t.Fatal("módulo de PDFs profissionais ausente")
	}
	b, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil { t.Fatal(err) }
	js := string(b)
	for _, marker := range []string{
		"Gerar Demonstrativo PDF",
		"Gerar Laudo PDF",
		"Gerar Evidências PDF",
		"Gerar Dossiê PDF",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface sem saída PDF %q", marker)
		}
	}
}


func TestRelease206PDFsDoNotRequireSavedProperty(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil { t.Fatal(err) }
	js := string(b)
	for _, marker := range []string{
		"ExportCARAutomationPDF(r)",
		"ExportEnvironmentalAutomationPDF(r)",
		"ExportEnvironmentalAutomationEvidencePDF(r)",
		"ExportPropertyAutomationDossierPDF(r)",
		"Salvar/vincular o imóvel é opcional",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("consulta avulsa sem integração PDF %q", marker)
		}
	}
	if strings.Contains(js, "const can=!!r.property_id;") {
		t.Fatal("relatórios 2.0.6 não podem depender de property_id")
	}
}
