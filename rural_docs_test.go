package main

import (
	"strings"
	"testing"
)

func TestNormalizePropertyUF(t *testing.T) {
	cases := map[string]string{
		"mg":  "MG",
		" GO ": "GO",
		"M":   "",
		"123": "",
	}
	for in, want := range cases {
		if got := normalizePropertyUF(in); got != want {
			t.Fatalf("normalizePropertyUF(%q) = %q; esperado %q", in, got, want)
		}
	}
}

func TestClientDetailTemCentralDocumental(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/client_detail.html")
	if err != nil {
		t.Fatalf("não foi possível ler client_detail.html: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		"CENTRAL DOCUMENTAL",
		"Pesquisa Nacional de Bens",
		"https://www.ridigital.org.br/PO/DefaultPO.aspx?from=menu",
		"https://sncr.serpro.gov.br/ccir/emissao",
		"Certidão ITR",
		"/car?number=",
		"/clients/{{$.Data.Client.ID}}/properties/{{.ID}}",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("central documental não contém %q", want)
		}
	}
}

func TestCentralDocumentalNaoPrometeAcessoAutenticado(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/client_detail.html")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(body))
	for _, forbidden := range []string{"senha gov.br", "capturar token", "burlar captcha"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("texto não deve sugerir coleta ou bypass de autenticação: %q", forbidden)
		}
	}
}
