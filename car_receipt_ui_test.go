package main

import (
	"strings"
	"testing"
)

func TestCentralCARFicaSimplesSemAssistenteDeRecibo(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/car.html")
	if err != nil {
		t.Fatalf("não foi possível ler template CAR: %v", err)
	}
	page := string(body)
	for _, want := range []string{
		"Gerar e baixar KML",
		"Baixar demonstrativo técnico PDF",
		"Abrir demonstrativo oficial do SICAR",
		"Abrir Consulta Pública do CAR",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("Central CAR não contém %q", want)
		}
	}
	for _, removed := range []string{
		"RECIBO OFICIAL DO CAR",
		"Gerenciar Vínculos",
		"Vincular Representante",
		"data-receipt-helper",
		"data-open-whatsapp",
		"car.meioambiente@goias.gov.br",
		"data-state-receipt-email",
	} {
		if strings.Contains(page, removed) {
			t.Fatalf("Central CAR ainda contém conteúdo removido %q", removed)
		}
	}
}

func TestAppJSNaoTemMaisAssistenteDeRecibo(t *testing.T) {
	body, err := webFS.ReadFile("web/static/app.js")
	if err != nil {
		t.Fatalf("não foi possível ler app.js: %v", err)
	}
	js := string(body)
	for _, removed := range []string{
		"data-receipt-helper",
		"data-build-receipt-request",
		"https://wa.me/?text=",
		"Solicitação de cópia do Recibo de Inscrição do CAR",
	} {
		if strings.Contains(js, removed) {
			t.Fatalf("app.js ainda contém lógica removida %q", removed)
		}
	}
}
