package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseVersion203(t *testing.T) {
	if AppVersion != "2.0.3" {
		t.Fatalf("AppVersion inesperada: %s", AppVersion)
	}
	b, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"productVersion": "2.0.3"`) {
		t.Fatal("wails.json não está alinhado com a versão 2.0.3")
	}
}

func TestReleaseFirePipeline191(t *testing.T) {
	if fireStatusFound == "" || fireStatusNone == "" || fireStatusUnavailable == "" || fireStatusNotRun == "" {
		t.Fatal("estados da consulta de focos não podem ficar vazios")
	}
}


func TestRelease201AssetsRemainVersioned(t *testing.T) {
	for _, path := range []string{"frontend/dist/ui201.css", "frontend/dist/ui201.js"} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("asset legado 2.0.1 ausente: %s", path)
		}
	}
}

func TestRelease202LoadsSVGInterface(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/index.html")
	if err != nil { t.Fatal(err) }
	html := string(b)
	for _, asset := range []string{"ui202.css", "ui202.js"} {
		if !strings.Contains(html, asset) { t.Fatalf("interface 2.0.2 não carrega %s", asset) }
	}
	js, err := os.ReadFile("frontend/dist/ui202.js")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(js), "<svg class=\"vv202-icon") {
		t.Fatal("interface 2.0.2 precisa usar ícones SVG embutidos")
	}
}


func TestRelease202DoesNotLoadSupersededShells(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/index.html")
	if err != nil { t.Fatal(err) }
	html := string(b)
	for _, asset := range []string{"ui200.js", "ui200.css", "ui201.js", "ui201.css"} {
		if strings.Contains(html, asset) {
			t.Fatalf("shell legado não deve ser carregado na 2.0.2: %s", asset)
		}
	}
	for _, asset := range []string{"ui202.js", "ui202.css"} {
		if !strings.Contains(html, asset) {
			t.Fatalf("shell 2.0.2 obrigatório não carregado: %s", asset)
		}
	}
}


func TestRelease203IncludesBCBOpenDataModule(t *testing.T) {
	for _, path := range []string{"bcb_public.go", "bcb_public_test.go"} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("módulo BCB 2.0.3 ausente: %s", path)
		}
	}
	b, err := os.ReadFile("frontend/dist/ui202.js")
	if err != nil { t.Fatal(err) }
	js := string(b)
	for _, marker := range []string{"BANCO CENTRAL • DADOS ABERTOS", "MDCR / SICOR", "IFDATA", "SCR.DATA", "TAXAS POR INSTITUIÇÃO"} {
		if !strings.Contains(js, marker) {
			t.Fatalf("painel BCB sem marcador %q", marker)
		}
	}
}
