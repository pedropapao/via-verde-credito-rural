package main

import "testing"

func hasAutoKey(cs []autoCandidate, key string) bool {
	for _, c := range cs {
		if c.Key == key {
			return true
		}
	}
	return false
}

func autoValueFor(cs []autoCandidate, key string) string {
	for _, c := range cs {
		if c.Key == key {
			return c.Value
		}
	}
	return ""
}

func TestSmartExtractionIgnoresSupplierIdentity(t *testing.T) {
	text := "ORÇAMENTO ASPERSÃO CNPJ: 11.222.333/0001-81 Município: Franca Valor: R$ 180.000,00"
	got := extractAutoFieldsSmart(text, "ORÇAMENTO ASPERSAO.pdf")
	if hasAutoKey(got, "cpf") || hasAutoKey(got, "municipality") || hasAutoKey(got, "amount") {
		t.Fatalf("orçamento de fornecedor não deve definir dados mestres: %#v", got)
	}
}

func TestSmartExtractionIgnoresSanitaryCPFAndActivity(t *testing.T) {
	text := "FICHA SANITÁRIA CPF 529.982.247-25 atividade pecuária Município: São Sebastião do Paraíso"
	got := extractAutoFieldsSmart(text, "FichaSanitariaAnalitica-4584.pdf")
	if len(got) != 0 {
		t.Fatalf("ficha sanitária deve ser documento de apoio, recebeu %#v", got)
	}
}

func TestSmartExtractionRejectsLegalTextAsMunicipality(t *testing.T) {
	text := "Município: São Sebastião do Maranhão Cronologia de Captação Mês Jan Fev Mar Abr Mai Jun Vazão L/s 27,6627"
	got := extractAutoFieldsSmart(text, "PORTARIA DE OUTORGA.pdf")
	if hasAutoKey(got, "municipality") {
		t.Fatalf("texto legal não pode virar município: %#v", got)
	}
}

func TestSmartExtractionKeepsPlausiblePropertyMunicipality(t *testing.T) {
	text := "Município: São Sebastião do Paraíso\nNome da propriedade: Fazenda Barra Mansa\nMatrícula: 33931\nÁrea total: 45,20 ha"
	got := extractAutoFieldsSmart(text, "CCIR fabio.pdf")
	if autoValueFor(got, "municipality") != "São Sebastião do Paraíso" {
		t.Fatalf("município válido não extraído: %#v", got)
	}
	if !hasAutoKey(got, "property") || !hasAutoKey(got, "registry") || !hasAutoKey(got, "area") {
		t.Fatalf("dados fundiários esperados ausentes: %#v", got)
	}
}

func TestValidCPF(t *testing.T) {
	if !validCPFOrCNPJ("52998224725") {
		t.Fatal("CPF de teste válido foi rejeitado")
	}
	if validCPFOrCNPJ("00006038156") {
		t.Fatal("sequência inválida não deve ser aceita como CPF")
	}
}
