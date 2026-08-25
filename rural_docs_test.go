package main

import (
	"strings"
	"testing"
)

func TestConsultaRuralEhIndependenteDosClientes(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/documents.html")
	if err != nil {
		t.Fatalf("não foi possível ler documents.html: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		"CONSULTA RURAL",
		"action=\"/documents\"",
		"Pesquisa Nacional de Bens",
		"SNCR / CCIR",
		"CIB / NIRF",
		"Consulta CAR Via Verde",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("consulta rural não contém %q", want)
		}
	}

	clientBody, err := webFS.ReadFile("web/templates/client_detail.html")
	if err != nil {
		t.Fatalf("não foi possível ler client_detail.html: %v", err)
	}
	clientText := string(clientBody)
	for _, forbidden := range []string{"CENTRAL DOCUMENTAL", "documentos-imovel", "/properties/"} {
		if strings.Contains(clientText, forbidden) {
			t.Fatalf("ficha de cliente não deve conter a consulta rural: %q", forbidden)
		}
	}
}

func TestConsultaRuralNaoSalvaNemPrometeAcessoProtegido(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/documents.html")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(body))
	if !strings.Contains(text, "não salva os dados") {
		t.Fatal("consulta rural deve informar que não salva os dados")
	}
	for _, forbidden := range []string{"senha gov.br", "capturar token", "burlar captcha"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("texto não deve sugerir coleta ou bypass de autenticação: %q", forbidden)
		}
	}
}

func TestTrimRuralLookup(t *testing.T) {
	if got := trimRuralLookup("  123  ", 10); got != "123" {
		t.Fatalf("trim inesperado: %q", got)
	}
	if got := trimRuralLookup("123456", 4); got != "1234" {
		t.Fatalf("limite inesperado: %q", got)
	}
}
