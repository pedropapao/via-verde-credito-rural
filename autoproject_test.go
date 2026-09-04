package main

import (
	"math"
	"testing"
)

func TestAutoProjectExtractsCoreFields(t *testing.T) {
	text := `Agente Financeiro: Banco do Brasil
N.º Agência: 0408-1
Proponente: Valdecir José de Souza
CPF/CNPJ: 930.225.576-04
Nome da Propriedade: Sítio Água Vermelha
Matrícula: 37.498 e 38.566
Município: São Sebastião do Paraíso
Cultura: Café
Área à financiar (ha): 1,77 ha
Valor Pretendido (R$): R$ 30.000,00
Nome do Técnico: Carlos Alberto Ohara
Registro no CREA/UF: 0685013027/SP
PRONAF`
	fields := extractAutoFields(text, "teste.docx")
	got := map[string]string{}
	for _, f := range fields {
		got[f.Key] = f.Value
	}
	for _, key := range []string{"producer", "cpf", "property", "municipality", "bank", "line", "activity", "area", "amount", "technician", "crea"} {
		if got[key] == "" {
			t.Fatalf("campo %s não extraído: %#v", key, got)
		}
	}
}

func TestAutoProjectAreaTolerance(t *testing.T) {
	docs := []autoDocument{
		{Name: "proposta.docx", Fields: []autoCandidate{{Key: "area", Label: "Área financiada", Value: "1,77 ha", Source: "proposta.docx"}}},
		{Name: "kml.kml", Fields: []autoCandidate{{Key: "area", Label: "Área financiada", Value: "1,79 ha", Source: "kml.kml"}}},
	}
	master, findings := buildAutoMaster(docs, nil)
	status := ""
	for _, m := range master {
		if m.Key == "area" {
			status = m.Status
		}
	}
	if status != "compatible" {
		t.Fatalf("esperava área compatível, obtive %q (%#v)", status, findings)
	}
}

func TestAutoProjectDetectsRealAreaDivergence(t *testing.T) {
	docs := []autoDocument{
		{Name: "proposta.docx", Fields: []autoCandidate{{Key: "area", Label: "Área financiada", Value: "1,77 ha", Source: "proposta.docx"}}},
		{Name: "planilha.xlsx", Fields: []autoCandidate{{Key: "area", Label: "Área financiada", Value: "2,10 ha", Source: "planilha.xlsx"}}},
	}
	_, findings := buildAutoMaster(docs, nil)
	found := false
	for _, f := range findings {
		if f.Level == "error" {
			found = true
		}
	}
	if !found {
		t.Fatal("divergência real de área não foi sinalizada")
	}
}

func TestAutoProjectBudgetCloses(t *testing.T) {
	text := `Insumos | Unidade | Quantidade/ha | Total | Preço unitário | Valor/ha | Valor Total
Fertilizante Orgânico | Ton | 4 | 7,08 | R$ 400,00 | R$ 1.600,00 | R$ 2.832,00
Fertilizante 19-04-19 | Ton | 0,8 | 1,416 | R$ 3.899,99 | R$ 3.119,99 | R$ 5.522,39
Calcário | Ton | 1 | 1,77 | R$ 306,66 | R$ 306,66 | R$ 542,79
Nitrato | Ton | 0,3 | 0,531 | R$ 3.370,00 | R$ 1.011,00 | R$ 1.789,47
Foliar | l | 12 | 21,24 | R$ 22,00 | R$ 264,00 | R$ 467,28
Herbicida | L | 6 | 10,62 | R$ 33,81 | R$ 202,86 | R$ 359,06
Fungicida | l | 4,5 | 7,965 | R$ 80,00 | R$ 360,00 | R$ 637,20
Fungicida solo | kg | 1 | 1,77 | R$ 280,00 | R$ 280,00 | R$ 495,60
Inseticida | L | 2 | 3,54 | R$ 120,00 | R$ 240,00 | R$ 424,80
Serviços | Hr. Máq. | 15 | 26,55 | R$ 170,00 | R$ 2.550,00 | R$ 4.513,50
Desbrotas | dia/hom | 4 | 7,08 | R$ 133,66 | R$ 534,64 | R$ 946,31
Colheita | Hr. Máq. | 9 | 15,93 | R$ 400,00 | R$ 3.600,00 | R$ 6.372,00
Secagem e Benefício | sc | 36 | 63,72 | R$ 80,00 | R$ 2.880,00 | R$ 5.097,60
TÉCNICO RESPONSÁVEL PELO LAUDO`
	total, ok, _ := detectAutoBudget(text)
	if !ok {
		t.Fatal("orçamento não foi reconhecido")
	}
	if math.Abs(total-30000) > 0.02 {
		t.Fatalf("esperava 30000, obtive %.2f", total)
	}
}
