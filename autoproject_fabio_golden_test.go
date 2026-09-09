package main

import "testing"

func TestFabioMapCarriesFutureHerdStarts(t *testing.T) {
	if len(autoFabioMap.FutureStartCols) != len(autoFabioMap.PreviousEndCols) {
		t.Fatalf("mapa de início/fim anual inconsistente")
	}
	if len(autoFabioMap.FutureStartCols) != 8 {
		t.Fatalf("esperava 8 transições anuais, recebeu %d", len(autoFabioMap.FutureStartCols))
	}
	if autoFabioMap.InitialHerd["matrizes"] != "C6" || autoFabioMap.InitialHerd["novilhos_23"] != "C12" {
		t.Fatalf("mapa do rebanho inicial alterado")
	}
}

func TestFabioTechnicalArraysCoverNineYears(t *testing.T) {
	sets := map[string][]string{
		"natalidade": autoFabioMap.NatalityCells,
		"mortalidade_adultos": autoFabioMap.MortalityAdultCells,
		"mortalidade_jovens": autoFabioMap.MortalityYoungCells,
		"mortalidade_bezerros": autoFabioMap.MortalityCalfCells,
		"descarte_matrizes": autoFabioMap.CullMatricesCells,
		"descarte_touros": autoFabioMap.CullBullsCells,
		"suporte_pastagens": autoFabioMap.PastureSupportCells,
	}
	for name, cells := range sets {
		if len(cells) != 9 {
			t.Fatalf("%s deveria ter 9 anos, recebeu %d", name, len(cells))
		}
	}
}

func TestFabioManualSaleIsExplicitInput(t *testing.T) {
	if autoFabioYear1FinishedSteerSaleCell != "F27" {
		t.Fatalf("venda planejada do primeiro ano deve permanecer entrada explícita")
	}
}
