package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseVersion191(t *testing.T) {
	if AppVersion != "1.9.1" {
		t.Fatalf("AppVersion inesperada: %s", AppVersion)
	}
	b, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"productVersion": "1.9.1"`) {
		t.Fatal("wails.json não está alinhado com a versão 1.9.1")
	}
}

func TestReleaseFirePipeline191(t *testing.T) {
	if fireStatusFound == "" || fireStatusNone == "" || fireStatusUnavailable == "" || fireStatusNotRun == "" {
		t.Fatal("estados da consulta de focos não podem ficar vazios")
	}
}
