package main

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func newFabioEngineTestWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	_ = f.SetSheetName("Sheet1", "01-Orçamento-Fontes")
	for _, s := range []string{"03 e 04-Reembolso", "05-Prod_agrop", "06-Evol.Reb", "07-Custeio Pec", "08-Estrutura Custos"} {
		if _, err := f.NewSheet(s); err != nil { t.Fatal(err) }
	}
	return f
}

func TestAgroFormulaModelRepairsCarryForward(t *testing.T) {
	f := newFabioEngineTestWorkbook(t)
	defer f.Close()
	if err := autoRepairAgroIrrigationFormulaModel(f); err != nil { t.Fatal(err) }
	checks := map[string]string{
		"G7": "=E7", "K7": "=I7", "O14": "=M14", "AK9": "=AI9",
	}
	for cell, want := range checks {
		got, err := f.GetCellFormula("06-Evol.Reb", cell)
		if err != nil { t.Fatal(err) }
		if got != want { t.Fatalf("%s: fórmula %q, esperava %q", cell, got, want) }
	}
	if got, _ := f.GetCellFormula("08-Estrutura Custos", "J22"); got != "=SUM(J5:J21)" {
		t.Fatalf("total de custos não reparado: %q", got)
	}
}

func TestAgroFormulaModelWritesOnlyExplicitTechnicalInputs(t *testing.T) {
	f := newFabioEngineTestWorkbook(t)
	defer f.Close()
	in := AutoAgroIrrigationInputs{
		Producer: "PRODUTOR TESTE", Property: "FAZENDA TESTE", Municipality: "MUNICIPIO TESTE",
		Line: "PRONAMP INVESTIMENTO", Purpose: "Sistema de irrigação por aspersão", Area: 9.96, InterestRate: 9,
		Budget: []autoNativeBudgetItem{{Description:"Sistema de irrigação", Unit:"Conjunto", Quantity:1, UnitValue:250500}},
		BudgetMonths: []string{"out/26"},
		InitialHerd: map[string]float64{"matrizes":74, "novilhas_12":48, "bezerras":15, "novilhos_23":81, "touros":6},
		Year1FinishedSteerSale: 81,
		SalePrice: map[string]float64{"bezerros":3500, "vacas":3400, "novilhos_mais_3":4500, "bezerras":2500},
	}
	in.Natality = [9]float64{80,80,83,83,83,83,83,83,83}
	in.PastureSupport = [9]float64{180,200,200,200,200,200,200,200,200}
	if _, err := autoApplyAgroIrrigationInputs(f, in); err != nil { t.Fatal(err) }
	if got, _ := f.GetCellValue("06-Evol.Reb", "C6"); got != "74" { t.Fatalf("matrizes: %q", got) }
	if got, _ := f.GetCellValue("06-Evol.Reb", "F27"); got != "81" { t.Fatalf("venda planejada: %q", got) }
	if got, _ := f.GetCellFormula("06-Evol.Reb", "G7"); got != "=E7" { t.Fatalf("carry forward perdido: %q", got) }
	if got, _ := f.GetCellValue("06-Evol.Reb", "D31"); got != "80" { t.Fatalf("natalidade ano 1: %q", got) }
	if got, _ := f.GetCellValue("06-Evol.Reb", "H17"); got != "200" { t.Fatalf("suporte ano 2: %q", got) }
}

func TestAgroFormulaModelUsesRealFabioCostRows(t *testing.T) {
	f := newFabioEngineTestWorkbook(t)
	defer f.Close()
	in := AutoAgroIrrigationInputs{
		CostUnit: map[string]float64{
			"vermifugo": 3,
			"mao_obra": 2000,
			"energia": 1617.25,
			"combustivel": 3000,
			"curral": 2000,
			"irrigacao": 5010,
			"seguridade_social": 400,
		},
		CostQty: map[string][9]float64{
			"mao_obra": {12,12,12,12,12,12,12,12,12},
			"energia": {12,12,12,12,12,12,12,12,12},
			"combustivel": {1,1,1,1,1,1,1,1,1},
			"curral": {1,1,1,1,1,1,1,1,1},
			"irrigacao": {0,1,1,1,1,1,1,1,1},
			"seguridade_social": {12,12,12,12,12,12,12,12,12},
		},
	}
	if _, err := autoApplyAgroIrrigationInputs(f, in); err != nil { t.Fatal(err) }

	unitChecks := map[string]string{
		"C21":"3", "C26":"2000", "C28":"1617.25", "C29":"3000",
		"C43":"2000", "C44":"5010", "C47":"400",
	}
	for cell, want := range unitChecks {
		got, err := f.GetCellValue("07-Custeio Pec", cell)
		if err != nil { t.Fatal(err) }
		if got != want { t.Fatalf("custo %s=%q, esperava %q", cell, got, want) }
	}
	qtyChecks := map[string]string{
		"D26":"12", "D28":"12", "D29":"1", "D43":"1", "F44":"1", "D47":"12",
	}
	for cell, want := range qtyChecks {
		got, err := f.GetCellValue("07-Custeio Pec", cell)
		if err != nil { t.Fatal(err) }
		if got != want { t.Fatalf("quantidade %s=%q, esperava %q", cell, got, want) }
	}
	if got, _ := f.GetCellValue("07-Custeio Pec", "D44"); got != "" {
		t.Fatalf("manutenção de irrigação no 1º ano não deve ser inventada: %q", got)
	}
}
