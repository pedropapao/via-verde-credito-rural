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
