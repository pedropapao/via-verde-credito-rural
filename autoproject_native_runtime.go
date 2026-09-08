package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

func autoNativeBudgetItems(docs []autoDocument, amount float64, activity string) []autoNativeBudgetItem {
	var out []autoNativeBudgetItem
	seen := map[string]bool{}
	lineRE := regexp.MustCompile(`(?i)^\s*(.{3,90}?)\s+(un|und|unid|unidade|kg|t|ton|l|litro|m|m2|m²|m3|m³|ha|h|hr|hora|cab|cabe[cç]a|serv|servi[cç]o)s?\s+([0-9]+(?:[\.,][0-9]+)?)\s+(?:R\$\s*)?([0-9\.]+(?:,[0-9]{1,2})?)`)
	for _, d := range docs {
		for _, raw := range strings.Split(strings.ReplaceAll(d.Text, "\r", ""), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || len(line) > 220 {
				continue
			}
			m := lineRE.FindStringSubmatch(line)
			if len(m) != 5 {
				continue
			}
			desc := strings.TrimSpace(m[1])
			key := autoFold(desc)
			if seen[key] || strings.Contains(key, "total") || strings.Contains(key, "subtotal") {
				continue
			}
			q := parseAutoDecimal(m[3])
			u := parseAutoDecimal(m[4])
			if q <= 0 || u <= 0 {
				continue
			}
			seen[key] = true
			out = append(out, autoNativeBudgetItem{Description: desc, Unit: strings.TrimSpace(m[2]), Quantity: q, UnitValue: u})
			if len(out) >= 12 {
				return out
			}
		}
	}
	if len(out) == 0 && amount > 0 {
		desc := "Investimento proposto"
		if strings.TrimSpace(activity) != "" {
			desc = "Investimento em " + strings.TrimSpace(activity)
		}
		out = append(out, autoNativeBudgetItem{Description: desc, Unit: "un", Quantity: 1, UnitValue: amount})
	}
	return out
}

func herdTotal(h autoHerdState) float64 {
	return h.CowsLactating + h.CowsDry + h.Heifers23 + h.Heifers12 + h.FemaleCalves + h.MaleCalves + h.Steers12 + h.Steers23 + h.FinishedSteer + h.Bulls
}

func herdSalesTotal(h autoHerdState) float64 { return herdTotal(h) }

func herdRounded(v float64) float64 {
	if v < 0 {
		return 0
	}
	return math.Round(v)
}

func autoProjectSimpleHerd(t AutoNativeTechnical, years int, activity string) []autoHerdYear {
	if years < 1 {
		return nil
	}
	current := autoHerdState{
		CowsLactating: herdRounded(t.CowsLactating), CowsDry: herdRounded(t.CowsDry),
		Heifers23: herdRounded(t.Heifers23), Heifers12: herdRounded(t.Heifers12),
		FemaleCalves: herdRounded(t.FemaleCalves), MaleCalves: herdRounded(t.MaleCalves),
		Steers12: herdRounded(t.Steers12), Steers23: herdRounded(t.Steers23),
		FinishedSteer: herdRounded(t.FinishedSteer), Bulls: herdRounded(t.Bulls),
	}
	adult := current.CowsLactating + current.CowsDry
	lactShare := .8
	if adult > 0 && current.CowsLactating > 0 {
		lactShare = current.CowsLactating / adult
	}
	isDairy := strings.Contains(autoFold(activity), "leite")
	out := make([]autoHerdYear, 0, years)
	for y := 0; y < years; y++ {
		births := (current.CowsLactating + current.CowsDry) * t.Natality / 100
		femaleBirths := births / 2
		maleBirths := births / 2
		deaths := autoHerdState{
			CowsLactating: current.CowsLactating * t.MortalityAdults / 100,
			CowsDry: current.CowsDry * t.MortalityAdults / 100,
			Heifers23: current.Heifers23 * t.MortalityYoung / 100,
			Heifers12: current.Heifers12 * t.MortalityYoung / 100,
			FemaleCalves: (current.FemaleCalves + femaleBirths) * t.MortalityCalves / 100,
			MaleCalves: (current.MaleCalves + maleBirths) * t.MortalityCalves / 100,
			Steers12: current.Steers12 * t.MortalityYoung / 100,
			Steers23: current.Steers23 * t.MortalityYoung / 100,
			FinishedSteer: current.FinishedSteer * t.MortalityAdults / 100,
			Bulls: current.Bulls * t.MortalityAdults / 100,
		}
		sales := autoHerdState{}
		matrixCull := (current.CowsLactating + current.CowsDry) * t.CullMatrices / 100
		sales.CowsLactating = matrixCull * lactShare
		sales.CowsDry = matrixCull * (1 - lactShare)
		sales.Bulls = current.Bulls * t.CullBulls / 100
		if isDairy {
			sales.MaleCalves = math.Max(0, maleBirths-deaths.MaleCalves)
			sales.FinishedSteer = math.Max(0, current.FinishedSteer-deaths.FinishedSteer)
		}

		matureHeifers := math.Max(0, current.Heifers23-deaths.Heifers23)
		adultEnd := math.Max(0, current.CowsLactating+current.CowsDry-deaths.CowsLactating-deaths.CowsDry-sales.CowsLactating-sales.CowsDry) + matureHeifers
		next := autoHerdState{}
		next.CowsLactating = herdRounded(adultEnd * lactShare)
		next.CowsDry = herdRounded(adultEnd - next.CowsLactating)
		next.Heifers23 = herdRounded(math.Max(0, current.Heifers12-deaths.Heifers12))
		next.Heifers12 = herdRounded(math.Max(0, current.FemaleCalves+femaleBirths-deaths.FemaleCalves))
		next.FemaleCalves = herdRounded(math.Max(0, femaleBirths*(1-t.MortalityCalves/100)))
		if isDairy {
			next.MaleCalves = 0
			next.Steers12 = herdRounded(math.Max(0, current.Steers12-deaths.Steers12))
			next.Steers23 = herdRounded(math.Max(0, current.Steers23-deaths.Steers23))
			next.FinishedSteer = 0
		} else {
			next.MaleCalves = herdRounded(math.Max(0, maleBirths*(1-t.MortalityCalves/100)))
			next.Steers12 = herdRounded(math.Max(0, current.MaleCalves+maleBirths-deaths.MaleCalves))
			next.Steers23 = herdRounded(math.Max(0, current.Steers12-deaths.Steers12))
			sales.FinishedSteer = herdRounded(math.Max(0, current.FinishedSteer-deaths.FinishedSteer))
			next.FinishedSteer = herdRounded(math.Max(0, current.Steers23-deaths.Steers23))
		}
		next.Bulls = herdRounded(math.Max(0, current.Bulls-deaths.Bulls-sales.Bulls))
		out = append(out, autoHerdYear{End: next, Sales: sales, Deaths: deaths})
		current = next
	}
	return out
}

func autoFillPecuarioHerd(f *excelize.File, t AutoNativeTechnical, proj []autoHerdYear) {
	initial := []float64{t.CowsLactating, t.CowsDry, t.Heifers23, t.Heifers12, t.FemaleCalves, t.MaleCalves, t.Steers12, t.Steers23, t.FinishedSteer, t.Bulls}
	for i, v := range initial {
		_ = f.SetCellValue("03-Evolução Rebanho", fmt.Sprintf("C%d", 18+i), herdRounded(v))
	}
	_ = f.SetCellValue("03-Evolução Rebanho", "D4", t.MortalityCalves/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D5", t.MortalityYoung/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D6", t.MortalityYoung/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D7", t.MortalityAdults/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D8", t.MortalityAdults/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D9", t.MortalityAdults/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D10", t.MortalityAdults/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D11", t.MortalityAdults/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D12", t.Natality/100)
	_ = f.SetCellValue("03-Evolução Rebanho", "D13", t.CullMatrices/100)

	cols := []string{"D", "E", "F", "G", "H", "I"}
	for y, yr := range proj {
		if y >= len(cols) {
			break
		}
		c := cols[y]
		end := []float64{yr.End.CowsLactating, yr.End.CowsDry, yr.End.Heifers23, yr.End.Heifers12, yr.End.FemaleCalves, yr.End.MaleCalves, yr.End.Steers12, yr.End.Steers23, yr.End.FinishedSteer, yr.End.Bulls}
		sales := []float64{yr.Sales.CowsLactating + yr.Sales.CowsDry, yr.Sales.Heifers23, yr.Sales.Heifers12, yr.Sales.FemaleCalves, yr.Sales.MaleCalves, yr.Sales.Steers12, yr.Sales.Steers23, yr.Sales.FinishedSteer, yr.Sales.Bulls}
		deaths := []float64{yr.Deaths.CowsLactating + yr.Deaths.CowsDry, yr.Deaths.Heifers23, yr.Deaths.Heifers12, yr.Deaths.FemaleCalves, yr.Deaths.MaleCalves, yr.Deaths.Steers12, yr.Deaths.Steers23, yr.Deaths.FinishedSteer, yr.Deaths.Bulls}
		for i, v := range end {
			_ = f.SetCellValue("03-Evolução Rebanho", fmt.Sprintf("%s%d", c, 18+i), herdRounded(v))
		}
		for i, v := range sales {
			_ = f.SetCellValue("03-Evolução Rebanho", fmt.Sprintf("%s%d", c, 30+i), herdRounded(v))
		}
		for i, v := range deaths {
			_ = f.SetCellValue("03-Evolução Rebanho", fmt.Sprintf("%s%d", c, 41+i), herdRounded(v))
		}
		_ = f.SetCellFormula("03-Evolução Rebanho", c+"28", fmt.Sprintf("=SUM(%s18:%s27)", c, c))
	}
}

func autoFillPecuarioCosts(f *excelize.File, t AutoNativeTechnical, proj []autoHerdYear) {
	colsQty := []string{"D", "F", "H", "J", "M", "O"}
	colsVal := []string{"E", "G", "I", "K", "N", "P"}
	for i := 0; i < len(colsQty); i++ {
		head := t.CowsLactating + t.CowsDry + t.Heifers23 + t.Heifers12 + t.FemaleCalves + t.MaleCalves + t.Steers12 + t.Steers23 + t.FinishedSteer + t.Bulls
		if i < len(proj) {
			head = herdTotal(proj[i].End)
		}
		q, v := colsQty[i], colsVal[i]
		if t.EnergyMonthly > 0 {
			_ = f.SetCellValue("05- Custeio Pecuario", "A27", "C - Energia Elétrica")
			_ = f.SetCellValue("05- Custeio Pecuario", "B27", "mês")
			_ = f.SetCellValue("05- Custeio Pecuario", "C27", t.EnergyMonthly)
			_ = f.SetCellValue("05- Custeio Pecuario", q+"27", 12)
			_ = f.SetCellFormula("05- Custeio Pecuario", v+"27", "="+q+"27*$C27")
		}
		if t.VetVisitCost > 0 {
			_ = f.SetCellValue("05- Custeio Pecuario", "A15", "Assistência veterinária")
			_ = f.SetCellValue("05- Custeio Pecuario", "B15", "visita")
			_ = f.SetCellValue("05- Custeio Pecuario", "C15", t.VetVisitCost)
			_ = f.SetCellValue("05- Custeio Pecuario", q+"15", 6)
			_ = f.SetCellFormula("05- Custeio Pecuario", v+"15", "="+q+"15*$C15")
		}
		if head > 0 {
			_ = f.SetCellValue("05- Custeio Pecuario", "A9", "Custeio operacional por cabeça")
			_ = f.SetCellValue("05- Custeio Pecuario", "B9", "cabeça")
			if cell, _ := f.GetCellValue("05- Custeio Pecuario", "C9"); strings.TrimSpace(cell) == "" {
				_ = f.SetCellValue("05- Custeio Pecuario", "C9", 0)
			}
			_ = f.SetCellValue("05- Custeio Pecuario", q+"9", herdRounded(head))
		}
	}

	for i, col := range []string{"B", "C", "D", "E", "F", "G"} {
		annual := t.CorralAnnual
		if t.LaborMonthly > 0 {
			annual += 12 * t.LaborMonthly
		}
		if t.AgronomyVisitCost > 0 {
			annual += 6 * t.AgronomyVisitCost
		}
		if annual > 0 {
			_ = f.SetCellValue("06-Estrutura Custos", col+"5", annual)
		}
		_ = i
	}
}

func autoFillAgroCosts(f *excelize.File, t AutoNativeTechnical, proj []autoHerdYear) {
	blocks := []struct{ unit, q1, v1, q2, v2 string }{
		{"C", "D", "E", "F", "G"}, {"J", "K", "L", "M", "N"}, {"Q", "R", "S", "T", "U"}, {"X", "Y", "Z", "AA", "AB"},
	}
	for _, b := range blocks {
		if t.VetVisitCost > 0 {
			_ = f.SetCellValue("07-Custeio Pec", b.unit+"14", t.VetVisitCost)
			_ = f.SetCellValue("07-Custeio Pec", b.q1+"14", 6)
			_ = f.SetCellFormula("07-Custeio Pec", b.v1+"14", "="+b.q1+"14*"+b.unit+"14")
			_ = f.SetCellValue("07-Custeio Pec", b.q2+"14", 6)
			_ = f.SetCellFormula("07-Custeio Pec", b.v2+"14", "="+b.q2+"14*"+b.unit+"14")
		}
		if t.AgronomyVisitCost > 0 {
			_ = f.SetCellValue("07-Custeio Pec", b.unit+"15", t.AgronomyVisitCost)
			_ = f.SetCellValue("07-Custeio Pec", b.q1+"15", 6)
			_ = f.SetCellFormula("07-Custeio Pec", b.v1+"15", "="+b.q1+"15*"+b.unit+"15")
			_ = f.SetCellValue("07-Custeio Pec", b.q2+"15", 6)
			_ = f.SetCellFormula("07-Custeio Pec", b.v2+"15", "="+b.q2+"15*"+b.unit+"15")
		}
		if t.LaborMonthly > 0 {
			_ = f.SetCellValue("07-Custeio Pec", b.unit+"26", t.LaborMonthly)
			_ = f.SetCellValue("07-Custeio Pec", b.q1+"26", 12)
			_ = f.SetCellFormula("07-Custeio Pec", b.v1+"26", "="+b.q1+"26*"+b.unit+"26")
			_ = f.SetCellValue("07-Custeio Pec", b.q2+"26", 12)
			_ = f.SetCellFormula("07-Custeio Pec", b.v2+"26", "="+b.q2+"26*"+b.unit+"26")
		}
		if t.EnergyMonthly > 0 {
			_ = f.SetCellValue("07-Custeio Pec", b.unit+"28", t.EnergyMonthly)
			_ = f.SetCellValue("07-Custeio Pec", b.q1+"28", 12)
			_ = f.SetCellFormula("07-Custeio Pec", b.v1+"28", "="+b.q1+"28*"+b.unit+"28")
			_ = f.SetCellValue("07-Custeio Pec", b.q2+"28", 12)
			_ = f.SetCellFormula("07-Custeio Pec", b.v2+"28", "="+b.q2+"28*"+b.unit+"28")
		}
		if t.CorralAnnual > 0 {
			_ = f.SetCellValue("07-Custeio Pec", b.unit+"43", t.CorralAnnual)
			_ = f.SetCellValue("07-Custeio Pec", b.q1+"43", 1)
			_ = f.SetCellFormula("07-Custeio Pec", b.v1+"43", "="+b.q1+"43*"+b.unit+"43")
			_ = f.SetCellValue("07-Custeio Pec", b.q2+"43", 1)
			_ = f.SetCellFormula("07-Custeio Pec", b.v2+"43", "="+b.q2+"43*"+b.unit+"43")
		}
		if t.IrrigationAnnual > 0 {
			_ = f.SetCellValue("07-Custeio Pec", b.unit+"44", t.IrrigationAnnual)
			_ = f.SetCellValue("07-Custeio Pec", b.q1+"44", 1)
			_ = f.SetCellFormula("07-Custeio Pec", b.v1+"44", "="+b.q1+"44*"+b.unit+"44")
			_ = f.SetCellValue("07-Custeio Pec", b.q2+"44", 1)
			_ = f.SetCellFormula("07-Custeio Pec", b.v2+"44", "="+b.q2+"44*"+b.unit+"44")
		}
	}

	// Fix the cost structure to reference the detailed cost sheet for all 9 years.
	for i, col := range []string{"B", "C", "D", "E", "F", "G", "H", "I", "J"} {
		src := []string{"E54", "G54", "L54", "N54", "S54", "U54", "Z54", "AB54", "AD54"}[i]
		_ = f.SetCellFormula("08-Estrutura Custos", col+"16", "='07-Custeio Pec'!"+src)
	}
}
