package main

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// AutoAgroIrrigationInputs representa somente entradas do modelo.
// Resultados (produção, evolução, custos, fluxo e capacidade) continuam sendo
// calculados pelas fórmulas da planilha nativa, e não por valores inventados no Go.
type AutoAgroIrrigationInputs struct {
	Producer     string
	Property     string
	Municipality string
	Line         string
	Purpose      string
	Area         float64
	InterestRate float64
	StartDate    any
	EndDate      any

	Budget []autoNativeBudgetItem
	BudgetMonths []string

	InitialHerd map[string]float64
	Year1FinishedSteerSale float64

	Natality        [9]float64
	MortalityAdult  [9]float64
	MortalityYoung  [9]float64
	MortalityCalf   [9]float64
	CullMatrices    [9]float64
	CullBulls       [9]float64
	PastureSupport  [9]float64

	SalePrice map[string]float64
	CostUnit  map[string]float64
	CostQty   map[string][9]float64
	FixedCost map[string][9]float64
}

func autoRepairAgroIrrigationFormulaModel(f *excelize.File) error {
	// O início de cada ano é sempre o fim do ano anterior para todas as categorias.
	for i, startCol := range autoFabioMap.FutureStartCols {
		prev := autoFabioMap.PreviousEndCols[i]
		for row := 6; row <= 14; row++ {
			if err := f.SetCellFormula("06-Evol.Reb", fmt.Sprintf("%s%d", startCol, row), fmt.Sprintf("=%s%d", prev, row)); err != nil {
				return err
			}
	}

	// Totais do movimento anual. Alguns arquivos históricos tinham valores digitados.
	for _, col := range []string{"C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "Y", "Z", "AA", "AB", "AC", "AD", "AE", "AF", "AG", "AH", "AI", "AJ", "AK", "AL", "AM", "AN"} {
		if err := f.SetCellFormula("06-Evol.Reb", col+"29", "=SUM("+col+"20:"+col+"28)"); err != nil {
			return err
		}
	}

	// Totais da estrutura de custos.
	for _, col := range []string{"B", "C", "D", "E", "F", "G", "H", "I", "J"} {
		if err := f.SetCellFormula("08-Estrutura Custos", col+"22", "=SUM("+col+"5:"+col+"21)"); err != nil {
			return err
		}
	}

	// Os quatro blocos de reembolso usam os mesmos dados contratuais.
	for _, row := range []int{13, 19, 25} {
		for _, col := range []string{"A", "B", "C", "E", "F"} {
			if err := f.SetCellFormula("03 e 04-Reembolso", fmt.Sprintf("%s%d", col, row), fmt.Sprintf("=%s$7", col)); err != nil {
				return err
			}
	}

	// Descrições/unidades/preços de venda se repetem nos três blocos de produção.
	for sourceRow, targets := range map[int][]int{12: {30, 49}, 13: {31, 50}, 14: {32, 51}, 15: {33, 52}} {
		for _, row := range targets {
			for _, col := range []string{"A", "B"} {
				if err := f.SetCellFormula("05-Prod_agrop", fmt.Sprintf("%s%d", col, row), fmt.Sprintf("=%s$%d", col, sourceRow)); err != nil {
					return err
				}
			}
			if err := f.SetCellFormula("05-Prod_agrop", fmt.Sprintf("C%d", row), fmt.Sprintf("=$C$%d", sourceRow)); err != nil {
				return err
			}
		}
	}
	return nil
}

func autoApplyAgroIrrigationInputs(f *excelize.File, in AutoAgroIrrigationInputs) ([]string, error) {
	if err := autoRepairAgroIrrigationFormulaModel(f); err != nil {
		return nil, err
	}
	var notes []string
	set := func(sheet, cell string, value any) error {
		if value == nil {
			return nil
		}
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			return err
		}
		notes = append(notes, sheet+"!"+cell)
		return nil
	}

	place := strings.Trim(strings.TrimSpace(in.Property+" - "+in.Municipality), " -")
	if in.Producer != "" {
		if err := set("01-Orçamento-Fontes", "A2", "PROPONENTE: "+in.Producer); err != nil { return nil, err }
	}
	if in.Line != "" {
		if err := set("01-Orçamento-Fontes", "F2", "LINHA PRETENDIDA: "+in.Line); err != nil { return nil, err }
	}
	if place != "" {
		for _, target := range [][2]string{{"01-Orçamento-Fontes", "A3"}, {"05-Prod_agrop", "A3"}, {"06-Evol.Reb", "A2"}, {"06-Evol.Reb", "W2"}} {
			prefix := "IMÓVEL: "
			if err := set(target[0], target[1], prefix+place); err != nil { return nil, err }
		}
	}
	if in.Area > 0 {
		if err := set("01-Orçamento-Fontes", "G3", in.Area); err != nil { return nil, err }
	}

	for i, item := range in.Budget {
		if i >= autoFabioMap.BudgetMaxItems { break }
		row := autoFabioMap.BudgetStartRow + i
		if item.Description != "" { if err := set("01-Orçamento-Fontes", fmt.Sprintf("A%d", row), item.Description); err != nil { return nil, err } }
		if item.Unit != "" { if err := set("01-Orçamento-Fontes", fmt.Sprintf("B%d", row), item.Unit); err != nil { return nil, err } }
		if item.Quantity > 0 { if err := set("01-Orçamento-Fontes", fmt.Sprintf("C%d", row), item.Quantity); err != nil { return nil, err } }
		if item.UnitValue > 0 { if err := set("01-Orçamento-Fontes", fmt.Sprintf("D%d", row), item.UnitValue); err != nil { return nil, err } }
		if i < len(in.BudgetMonths) && strings.TrimSpace(in.BudgetMonths[i]) != "" {
			if err := set("01-Orçamento-Fontes", fmt.Sprintf("H%d", row), in.BudgetMonths[i]); err != nil { return nil, err }
		}
	}

	if in.Line != "" { if err := set("03 e 04-Reembolso", "A7", in.Line); err != nil { return nil, err } }
	if in.Purpose != "" { if err := set("03 e 04-Reembolso", "B7", in.Purpose); err != nil { return nil, err } }
	if in.StartDate != nil { if err := set("03 e 04-Reembolso", "C7", in.StartDate); err != nil { return nil, err } }
	if in.EndDate != nil { if err := set("03 e 04-Reembolso", "E7", in.EndDate); err != nil { return nil, err } }
	if in.InterestRate > 0 { if err := set("03 e 04-Reembolso", "F7", in.InterestRate); err != nil { return nil, err } }

	for key, cell := range autoFabioMap.InitialHerd {
		if v := in.InitialHerd[key]; v >= 0 {
			if _, ok := in.InitialHerd[key]; ok { if err := set("06-Evol.Reb", cell, v); err != nil { return nil, err } }
		}
	}
	if in.Year1FinishedSteerSale > 0 {
		if err := set("06-Evol.Reb", autoFabioYear1FinishedSteerSaleCell, in.Year1FinishedSteerSale); err != nil { return nil, err }
	}

	arrays := []struct{ cells []string; vals [9]float64 }{
		{autoFabioMap.NatalityCells, in.Natality},
		{autoFabioMap.MortalityAdultCells, in.MortalityAdult},
		{autoFabioMap.MortalityYoungCells, in.MortalityYoung},
		{autoFabioMap.MortalityCalfCells, in.MortalityCalf},
		{autoFabioMap.CullMatricesCells, in.CullMatrices},
		{autoFabioMap.CullBullsCells, in.CullBulls},
		{autoFabioMap.PastureSupportCells, in.PastureSupport},
	}
	for _, a := range arrays {
		for i, cell := range a.cells {
			if a.vals[i] != 0 { if err := set("06-Evol.Reb", cell, a.vals[i]); err != nil { return nil, err } }
		}
	}

	for key, cells := range autoFabioMap.SalePriceCells {
		v := in.SalePrice[key]
		if v <= 0 { continue }
		for _, cell := range cells { if err := set("05-Prod_agrop", cell, v); err != nil { return nil, err } }
	}
	for key, cell := range autoFabioMap.CostUnitCells {
		if v := in.CostUnit[key]; v > 0 { if err := set("07-Custeio Pec", cell, v); err != nil { return nil, err } }
	}
	for key, cells := range autoFabioMap.CostQtyCells {
		vals, ok := in.CostQty[key]; if !ok { continue }
		for i, cell := range cells { if vals[i] != 0 { if err := set("07-Custeio Pec", cell, vals[i]); err != nil { return nil, err } } }
	}
	for key, cells := range autoFabioMap.FixedCostCells {
		vals, ok := in.FixedCost[key]; if !ok { continue }
		for i, cell := range cells { if vals[i] != 0 { if err := set("08-Estrutura Custos", cell, vals[i]); err != nil { return nil, err } } }
	}
	return notes, nil
}
