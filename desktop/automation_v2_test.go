package main

import "testing"

func TestClassifyUnifiedQueryCPF(t *testing.T) {
	mode, normalized, valid := v2ClassifyUnifiedQuery("529.982.247-25")
	if mode != "cpf" || normalized != "52998224725" || !valid {
		t.Fatalf("classificação CPF inesperada: %s %s %v", mode, normalized, valid)
	}
}

func TestClassifyUnifiedQueryCNPJ(t *testing.T) {
	mode, normalized, valid := v2ClassifyUnifiedQuery("11.222.333/0001-81")
	if mode != "cnpj" || normalized != "11222333000181" || !valid {
		t.Fatalf("classificação CNPJ inesperada: %s %s %v", mode, normalized, valid)
	}
}

func TestInvalidTaxIDs(t *testing.T) {
	if v2ValidCPF("11111111111") {
		t.Fatal("CPF repetido não pode ser válido")
	}
	if v2ValidCNPJ("00000000000000") {
		t.Fatal("CNPJ repetido não pode ser válido")
	}
}

func TestNormalizedSearchText(t *testing.T) {
	got := v2NormalizedSearchText("  São Sebastião   do Paraíso ")
	if got != "sao sebastiao do paraiso" {
		t.Fatalf("normalização inesperada: %q", got)
	}
}
