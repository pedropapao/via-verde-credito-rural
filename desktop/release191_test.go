package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseVersion202(t *testing.T) {
	if AppVersion != "2.0.2" {
		t.Fatalf("AppVersion inesperada: %s", AppVersion)
	}
	b, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"productVersion": "2.0.2"`) {
		t.Fatal("wails.json não está alinhado com a versão 2.0.2")
	}
}

func TestReleaseFirePipeline191(t *testing.T) {
	if fireStatusFound == "" || fireStatusNone == "" || fireStatusUnavailable == "" || fireStatusNotRun == "" {
		t.Fatal("estados da consulta de focos não podem ficar vazios")
	}
}


func TestRelease201LoadsNewInterface(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	for _, asset := range []string{"ui201.css", "ui201.js"} {
		if !strings.Contains(html, asset) {
			t.Fatalf("interface 2.0.1 não carrega %s", asset)
		}
	}
	for _, path := range []string{"frontend/dist/ui201.css", "frontend/dist/ui201.js"} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("asset 2.0.1 ausente ou vazio: %s", path)
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
