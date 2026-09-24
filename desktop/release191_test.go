package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseVersion200(t *testing.T) {
	if AppVersion != "2.0.0" {
		t.Fatalf("AppVersion inesperada: %s", AppVersion)
	}
	b, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"productVersion": "2.0.0"`) {
		t.Fatal("wails.json não está alinhado com a versão 2.0.0")
	}
}

func TestReleaseFirePipeline191(t *testing.T) {
	if fireStatusFound == "" || fireStatusNone == "" || fireStatusUnavailable == "" || fireStatusNotRun == "" {
		t.Fatal("estados da consulta de focos não podem ficar vazios")
	}
}
