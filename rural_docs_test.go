package main

import (
	"strings"
	"testing"
)

func TestConsultaRuralNaoApareceNoMenu(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/base.html")
	if err != nil {
		t.Fatalf("não foi possível ler base.html: %v", err)
	}
	text := string(body)
	for _, forbidden := range []string{"Consulta Rural", `href="/documents"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("menu não deve conter %q", forbidden)
		}
	}
}

func TestFichaClienteNaoContemConsultaRural(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/client_detail.html")
	if err != nil {
		t.Fatalf("não foi possível ler client_detail.html: %v", err)
	}
	text := string(body)
	for _, forbidden := range []string{"CENTRAL DOCUMENTAL", "documentos-imovel", "Consulta Rural", "/properties/"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("ficha de cliente não deve conter consulta rural: %q", forbidden)
		}
	}
}
