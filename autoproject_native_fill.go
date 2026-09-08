package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func autoFillNativeTemplate(id string, values map[string]string, t AutoNativeTechnical, docs []autoDocument) ([]byte, string, []string, error) {
	var raw []byte
	var name string
	var err error
	switch id {
	case "agro_irrigacao":
		raw, err = autoDecodeBuiltinTemplate(autoBuiltinIrrigacaoB64)
		name = "Projeto_Investimento_Agropecuario_Irrigacao_Preenchido.xlsx"
	case "pecuario_completo":
		raw, err = autoDecodeBuiltinTemplate(autoBuiltinPecuarioB64)
		name = "Projeto_Investimento_Pecuario_Preenchido.xlsx"
	default:
		return nil, "", nil, fmt.Errorf("modelo nativo inválido")
	}
	if err != nil {
		return nil, "", nil, err
	}
	if id == "agro_irrigacao" {
		out, notes, err := autoFillNativeAgro(raw, values, t, docs)
		return out, name, notes, err
	}
	out, notes, err := autoFillNativePecuario(raw, values, t, docs)
	return out, name, notes, err
}

func autoDecodeBuiltinTemplate(raw string) ([]byte, error) {
	raw = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			return -1
		}
		return r
	}, raw)
	return base64.StdEncoding.DecodeString(raw)
}

func autoFillNativePecuario(data []byte, values map[string]string, t AutoNativeTechnical, docs []autoDocument) ([]byte, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var notes []string
	set := func(sheet, cell string, value any) {
		_ = f.SetCellValue(sheet, cell, value)
		notes = append(notes, sheet+"!"+cell)
	}
	setFormula := func(sheet, cell, formula string) { _ = f.SetCellFormula(sheet, cell, formula) }

	producer := values["producer"]
	property := values["property"]
	municipality := values["municipality"]
	activity := values["activity"]
	if activity == "" {
		activity = "Pecuária"
	}
	area := parseAutoDecimal(values["area"])
	amount := parseAutoMoney(values["amount"])
	place := strings.Trim(strings.TrimSpace(property+" - "+municipality), " -")

	set("Projeto", "C11", producer)
	set("Projeto", "G11", values["cpf"])
	set("Projeto", "C15", values["bank"])
	set("Projeto", "C19", property)
	set("Projeto", "G19", values["registry"])
	set("Projeto", "C20", municipality)
	set("Projeto", "C24", producer)
	if area > 0 {
		set("Projeto", "C25", area)
	}
	set("Projeto", "C71", activity)
	set("Projeto", "B82", "Investimento em "+activity+" conforme orçamento, capacidade produtiva e documentos apresentados.")
	if amount > 0 {
		set("Projeto", "C84", amount)
	}
	set("Projeto", "C85", values["line"])
	if t.InterestRate > 0 {
		set("Projeto", "G85", t.InterestRate)
	}
	if t.TermMonths > 0 {
		set("Projeto", "C86", t.TermMonths)
	}
	if t.GraceMonths > 0 {
		set("Projeto", "G86", t.GraceMonths)
	}
	set("Capa", "C24", municipality)
	set("Capa", "C37", values["bank"])

	for _, sh := range []string{"01-Orçamento-Fontes", "02-Reembolso", "03-Evolução Rebanho", "04-Prod_agrop", "06-Estrutura Custos", "08-Fluxo Caixa"} {
		if place != "" {
			cell := "A2"
			if sh == "04-Prod_agrop" {
				cell = "A3"
			}
			set(sh, cell, place)
		}
	}
	set("03-Evolução Rebanho", "A3", "Evolução de rebanho - "+activity)
	set("05- Custeio Pecuario", "A2", "ATIVIDADE: "+activity)
	set("05- Custeio Pecuario", "L2", "ATIVIDADE: "+activity)

	items := autoNativeBudgetItems(docs, amount, activity)
	for i := 0; i < len(items) && i < 5; i++ {
		row := 5 + i
		it := items[i]
		set("01-Orçamento-Fontes", fmt.Sprintf("A%d", row), it.Description)
		set("01-Orçamento-Fontes", fmt.Sprintf("B%d", row), valueOrDefault(it.Unit, "un"))
		set("01-Orçamento-Fontes", fmt.Sprintf("C%d", row), nonZeroOr(it.Quantity, 1))
		set("01-Orçamento-Fontes", fmt.Sprintf("D%d", row), it.UnitValue)
		setFormula("01-Orçamento-Fontes", fmt.Sprintf("E%d", row), fmt.Sprintf("=C%d*D%d", row, row))
		setFormula("01-Orçamento-Fontes", fmt.Sprintf("F%d", row), fmt.Sprintf("=E%d", row))
	}
	for _, col := range []string{"E", "F", "G", "H"} {
		setFormula("01-Orçamento-Fontes", col+"10", "=SUM("+col+"5:"+col+"9)")
	}

	autoFillPecuarioReembolso(f, t, amount)
	projection := autoProjectSimpleHerd(t, 6, activity)
	autoFillPecuarioHerd(f, t, projection)
	autoFillPecuarioProduction(f, t, projection, activity)
	autoFillPecuarioCosts(f, t, projection)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), notes, nil
}

func autoFillNativeAgro(data []byte, values map[string]string, t AutoNativeTechnical, docs []autoDocument) ([]byte, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var notes []string
	set := func(sheet, cell string, value any) {
		_ = f.SetCellValue(sheet, cell, value)
		notes = append(notes, sheet+"!"+cell)
	}
	setFormula := func(sheet, cell, formula string) { _ = f.SetCellFormula(sheet, cell, formula) }

	producer, property, municipality := values["producer"], values["property"], values["municipality"]
	line := values["line"]
	activity := values["activity"]
	if t.CropName == "" && !autoLooksLivestock(activity) {
		t.CropName = activity
	}
	if t.CropUnit == "" && t.CropName != "" {
		t.CropUnit = "sc"
	}
	amount := parseAutoMoney(values["amount"])
	place := strings.Trim(strings.TrimSpace(property+" - "+municipality), " -")

	set("01-Orçamento-Fontes", "A2", "PROPONENTE: "+producer)
	set("01-Orçamento-Fontes", "F2", "LINHA PRETENDIDA: "+line)
	set("01-Orçamento-Fontes", "A3", "IMÓVEL: "+place)
	if t.CropArea > 0 {
		set("01-Orçamento-Fontes", "G3", t.CropArea)
	} else if a := parseAutoDecimal(values["area"]); a > 0 {
		set("01-Orçamento-Fontes", "G3", a)
	}
	set("05-Prod_agrop", "A3", "IMÓVEL: "+place)
	set("06-Evol.Reb", "A2", "IMÓVEL: "+place)
	set("06-Evol.Reb", "W2", "IMÓVEL: "+place)
	for _, c := range []string{"A3", "H3", "O3", "V3"} {
		set("07-Custeio Pec", c, "ATIVIDADE: "+activity)
	}

	autoFillAgroYears(f, t.StartYear)
	items := autoNativeBudgetItems(docs, amount, activity)
	for i := 0; i < len(items) && i < 9; i++ {
		row := 7 + i
		it := items[i]
		set("01-Orçamento-Fontes", fmt.Sprintf("A%d", row), it.Description)
		set("01-Orçamento-Fontes", fmt.Sprintf("B%d", row), valueOrDefault(it.Unit, "un"))
		set("01-Orçamento-Fontes", fmt.Sprintf("C%d", row), nonZeroOr(it.Quantity, 1))
		set("01-Orçamento-Fontes", fmt.Sprintf("D%d", row), it.UnitValue)
		setFormula("01-Orçamento-Fontes", fmt.Sprintf("E%d", row), fmt.Sprintf("=C%d*D%d", row, row))
		setFormula("01-Orçamento-Fontes", fmt.Sprintf("F%d", row), fmt.Sprintf("=E%d", row))
		setFormula("01-Orçamento-Fontes", fmt.Sprintf("G%d", row), fmt.Sprintf("=E%d-F%d", row, row))
	}

	autoFillAgroReembolso(f, t, amount)
	autoFillAgroCrop(f, t)
	projection := autoProjectSimpleHerd(t, 9, activity)
	autoFillAgroHerdInputs(f, t)
	autoFillAgroLivestockRevenue(f, t, projection, activity)
	autoFillAgroCosts(f, t, projection)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), notes, nil
}

func autoFillPecuarioReembolso(f *excelize.File, t AutoNativeTechnical, amount float64) {
	if t.StartYear == 0 {
		t.StartYear = time.Now().Year()
	}
	if t.TermMonths == 0 {
		t.TermMonths = 84
	}
	annualYears := int(math.Ceil(float64(t.TermMonths) / 12))
	if annualYears < 1 {
		annualYears = 1
	}
	graceYears := int(math.Ceil(float64(t.GraceMonths) / 12))
	principalYears := annualYears - graceYears
	if principalYears < 1 {
		principalYears = 1
	}
	principal := amount / float64(principalYears)
	balance := amount
	interestCells := []string{"G8", "I8", "K8", "G15", "I15", "K15", "G22"}
	amortCells := []string{"H8", "J8", "L8", "H15", "J15", "L15", "H22"}
	for i := 0; i < len(interestCells); i++ {
		year := i + 1
		interest := balance * t.InterestRate / 100
		amort := 0.0
		if year > graceYears && balance > 0 {
			amort = math.Min(principal, balance)
		}
		_ = f.SetCellValue("02-Reembolso", interestCells[i], interest)
		_ = f.SetCellValue("02-Reembolso", amortCells[i], amort)
		balance -= amort
	}
}

func autoFillAgroReembolso(f *excelize.File, t AutoNativeTechnical, amount float64) {
	if t.TermMonths == 0 {
		t.TermMonths = 96
	}
	annualYears := int(math.Ceil(float64(t.TermMonths) / 12))
	if annualYears < 1 {
		annualYears = 1
	}
	graceYears := int(math.Ceil(float64(t.GraceMonths) / 12))
	principalYears := annualYears - graceYears
	if principalYears < 1 {
		principalYears = 1
	}
	principal := amount / float64(principalYears)
	balance := amount
	interestCells := []string{"G8", "I8", "G14", "I14", "G20", "I20", "G26", "I26"}
	amortCells := []string{"H8", "J8", "H14", "J14", "H20", "J20", "H26", "J26"}
	_ = f.SetCellValue("03 e 04-Reembolso", "D7", amount)
	_ = f.SetCellValue("03 e 04-Reembolso", "F7", t.InterestRate)
	for i := range interestCells {
		year := i + 1
		interest := balance * t.InterestRate / 100
		amort := 0.0
		if year > graceYears && balance > 0 {
			amort = math.Min(principal, balance)
		}
		_ = f.SetCellValue("03 e 04-Reembolso", interestCells[i], interest)
		_ = f.SetCellValue("03 e 04-Reembolso", amortCells[i], amort)
		balance -= amort
	}
}

func autoFillAgroYears(f *excelize.File, start int) {
	if start < 2020 {
		start = time.Now().Year()
	}
	for i, cell := range []string{"D4", "H4", "L4", "D22", "H22", "L22", "D41", "H41", "L41"} {
		_ = f.SetCellValue("05-Prod_agrop", cell, start+i)
	}
	for i, cell := range []string{"C3", "G3", "K3", "O3", "S3", "Y3", "AC3", "AG3", "AK3"} {
		_ = f.SetCellValue("06-Evol.Reb", cell, start+i)
	}
	for i, cell := range []string{"D6", "F6", "K6", "M6", "R6", "T6", "Y6", "AA6", "AC6"} {
		_ = f.SetCellValue("07-Custeio Pec", cell, start+i)
	}
	for i, cell := range []string{"B3", "C3", "D3", "E3", "F3", "G3", "H3", "I3", "J3"} {
		_ = f.SetCellValue("08-Estrutura Custos", cell, start+i)
	}
	for i, cell := range []string{"B3", "C3", "D3", "F3", "G3", "H3", "J3", "K3", "L3"} {
		_ = f.SetCellValue("09-Fluxo Caixa", cell, start+i)
	}
}

func autoFillAgroCrop(f *excelize.File, t AutoNativeTechnical) {
	if t.CropName == "" || t.CropArea <= 0 || t.CropProductivity <= 0 {
		return
	}
	unit := valueOrDefault(t.CropUnit, "sc")
	blocks := []struct {
		row    int
		starts []string
	}{
		{9, []string{"D", "H", "L"}}, {27, []string{"D", "H", "L"}}, {46, []string{"D", "H", "L"}},
	}
	for _, b := range blocks {
		_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("A%d", b.row), t.CropName)
		_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("B%d", b.row), unit)
		_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("C%d", b.row), t.CropPrice)
		for _, c := range b.starts {
			areaCol := c
			prodCol := string(rune(c[0] + 1))
			qtyCol := string(rune(c[0] + 2))
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("%s%d", areaCol, b.row), t.CropArea)
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("%s%d", prodCol, b.row), t.CropProductivity)
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("%s%d", qtyCol, b.row), t.CropArea*t.CropProductivity)
		}
	}
}

func autoFillPecuarioProduction(f *excelize.File, t AutoNativeTechnical, proj []autoHerdYear, activity string) {
	if t.CropName != "" && t.CropArea > 0 && t.CropProductivity > 0 {
		_ = f.SetCellValue("04-Prod_agrop", "A9", t.CropName)
		_ = f.SetCellValue("04-Prod_agrop", "B9", valueOrDefault(t.CropUnit, "sc"))
		_ = f.SetCellValue("04-Prod_agrop", "C9", t.CropPrice)
		for _, c := range []string{"D", "F", "H", "J", "L", "N", "P", "R", "T", "V"} {
			_ = f.SetCellValue("04-Prod_agrop", c+"9", t.CropArea*t.CropProductivity)
			revCol := string(rune(c[0] + 1))
			_ = f.SetCellFormula("04-Prod_agrop", revCol+"9", "="+c+"9*$C$9")
		}
	}
	isDairy := strings.Contains(autoFold(activity), "leite")
	if isDairy && t.MilkLitersCowDay > 0 && t.MilkPriceLiter > 0 {
		_ = f.SetCellValue("04-Prod_agrop", "A12", "Leite")
		_ = f.SetCellValue("04-Prod_agrop", "B12", "litro")
		_ = f.SetCellValue("04-Prod_agrop", "C12", t.MilkPriceLiter)
		for i, c := range []string{"D", "F", "H", "J", "L", "N"} {
			cows := t.CowsLactating
			if i < len(proj) {
				cows = proj[i].End.CowsLactating
			}
			_ = f.SetCellValue("04-Prod_agrop", c+"12", cows*t.MilkLitersCowDay*365)
		}
	} else if t.SalePriceHead > 0 {
		_ = f.SetCellValue("04-Prod_agrop", "A12", "Venda de bovinos")
		_ = f.SetCellValue("04-Prod_agrop", "B12", "cabeça")
		_ = f.SetCellValue("04-Prod_agrop", "C12", t.SalePriceHead)
		for i, c := range []string{"D", "F", "H", "J", "L", "N"} {
			if i >= len(proj) {
				break
			}
			_ = f.SetCellValue("04-Prod_agrop", c+"12", herdSalesTotal(proj[i].Sales))
		}
	}
}

func autoFillAgroLivestockRevenue(f *excelize.File, t AutoNativeTechnical, proj []autoHerdYear, activity string) {
	isDairy := strings.Contains(autoFold(activity), "leite")
	blocks := []struct {
		row  int
		cols []string
	}{
		{12, []string{"F", "J", "N"}}, {30, []string{"F", "J", "N"}}, {49, []string{"F", "J", "N"}},
	}
	if isDairy && t.MilkLitersCowDay > 0 && t.MilkPriceLiter > 0 {
		for bi, b := range blocks {
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("A%d", b.row), "Leite")
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("B%d", b.row), "litro")
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("C%d", b.row), t.MilkPriceLiter)
			for j, c := range b.cols {
				i := bi*3 + j
				cows := t.CowsLactating
				if i < len(proj) {
					cows = proj[i].End.CowsLactating
				}
				_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("%s%d", c, b.row), cows*t.MilkLitersCowDay*365)
			}
		}
	} else if t.SalePriceHead > 0 {
		for bi, b := range blocks {
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("A%d", b.row), "Venda de bovinos")
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("B%d", b.row), "cabeça")
			_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("C%d", b.row), t.SalePriceHead)
			for j, c := range b.cols {
				i := bi*3 + j
				if i < len(proj) {
					_ = f.SetCellValue("05-Prod_agrop", fmt.Sprintf("%s%d", c, b.row), herdSalesTotal(proj[i].Sales))
				}
			}
		}
	}
}

func autoFillAgroHerdInputs(f *excelize.File, t AutoNativeTechnical) {
	initial := []float64{t.CowsLactating + t.CowsDry, t.Heifers23, t.Heifers12, t.FemaleCalves, t.MaleCalves, t.Steers12, t.Steers23, t.FinishedSteer, t.Bulls}
	for i, v := range initial {
		if v > 0 {
			_ = f.SetCellValue("06-Evol.Reb", fmt.Sprintf("C%d", 6+i), v)
		}
	}
	for _, cell := range []string{"D31", "H31", "L31", "P31", "T31", "Z31", "AD31", "AH31", "AL31"} {
		_ = f.SetCellValue("06-Evol.Reb", cell, t.Natality)
	}
	for _, cell := range []string{"D32", "H32", "L32", "P32", "T32", "Z32", "AD32", "AH32", "AL32"} {
		_ = f.SetCellValue("06-Evol.Reb", cell, t.MortalityAdults)
	}
	for _, cell := range []string{"D33", "H33", "L33", "P33", "T33", "Z33", "AD33", "AH33", "AL33"} {
		_ = f.SetCellValue("06-Evol.Reb", cell, t.MortalityYoung)
	}
	for _, cell := range []string{"D34", "H34", "L34", "P34", "T34", "Z34", "AD34", "AH34", "AL34"} {
		_ = f.SetCellValue("06-Evol.Reb", cell, t.MortalityCalves)
	}
	for _, cell := range []string{"D35", "H35", "L35", "P35", "T35", "Z35", "AD35", "AH35", "AL35"} {
		_ = f.SetCellValue("06-Evol.Reb", cell, t.CullMatrices)
	}
	for _, cell := range []string{"D36", "H36", "L36", "P36", "T36", "Z36", "AD36", "AH36", "AL36"} {
		_ = f.SetCellValue("06-Evol.Reb", cell, t.CullBulls)
	}
	if t.PastureSupportUA > 0 {
		for _, cell := range []string{"D17", "F17", "H17", "J17", "L17", "N17", "P17", "R17", "T17", "V17", "Z17", "AB17", "AD17", "AF17", "AH17", "AJ17", "AL17", "AN17"} {
			_ = f.SetCellValue("06-Evol.Reb", cell, t.PastureSupportUA)
		}
	}
}
