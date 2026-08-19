package main

import (
	"strings"
	"testing"
)

func TestCentralCARTemFluxoAutorizadoParaRecibo(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/car.html")
	if err != nil {
		t.Fatalf("não foi possível ler template CAR: %v", err)
	}
	page := string(body)
	for _, want := range []string{
		"RECIBO OFICIAL DO CAR",
		"Gerenciar Vínculos",
		"Vincular Representante",
		"https://www.car.gov.br/#/central/acesso",
		"https://www.car.gov.br/#/consultar",
		"data-receipt-helper",
		"data-open-whatsapp",
		"car.meioambiente@goias.gov.br",
		"data-state-receipt-email",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("Central CAR não contém %q", want)
		}
	}
}

func TestAssistenteReciboNaoSolicitaSenhaDoProdutor(t *testing.T) {
	body, err := webFS.ReadFile("web/static/app.js")
	if err != nil {
		t.Fatalf("não foi possível ler app.js: %v", err)
	}
	js := string(body)
	if !strings.Contains(js, "não preciso da sua senha GOV.BR") {
		t.Fatal("mensagem de vínculo precisa deixar explícito que a senha do produtor não é necessária")
	}
	if !strings.Contains(js, "https://wa.me/?text=") {
		t.Fatal("assistente precisa permitir abrir a mensagem no WhatsApp")
	}
	if !strings.Contains(js, "Solicitação de cópia do Recibo de Inscrição do CAR") {
		t.Fatal("assistente precisa preparar solicitação oficial estadual do Recibo CAR")
	}
}
