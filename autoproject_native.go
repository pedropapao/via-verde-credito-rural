package main

import (
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AutoNativeTemplateInfo struct {
	ID          string
	Name        string
	Description string
	Focus       string
}

type AutoNativeTechnical struct {
	StartYear        int
	InterestRate     float64
	TermMonths       int
	GraceMonths      int
	CropName         string
	CropUnit         string
	CropArea         float64
	CropProductivity float64
	CropPrice        float64

	CowsLactating float64
	CowsDry       float64
	Heifers23     float64
	Heifers12     float64
	FemaleCalves  float64
	MaleCalves    float64
	Steers12      float64
	Steers23      float64
	FinishedSteer float64
	Bulls         float64

	Natality         float64
	MortalityAdults  float64
	MortalityYoung   float64
	MortalityCalves  float64
	CullMatrices     float64
	CullBulls        float64
	PastureSupportUA float64

	MilkLitersCowDay float64
	MilkPriceLiter   float64
	SalePriceHead    float64

	VetVisitCost      float64
	AgronomyVisitCost float64
	LaborMonthly      float64
	EnergyMonthly     float64
	CorralAnnual      float64
	IrrigationAnnual  float64
}

type autoNativeBudgetItem struct {
	Description string
	Unit        string
	Quantity    float64
	UnitValue   float64
}

type autoHerdState struct {
	CowsLactating float64
	CowsDry       float64
	Heifers23     float64
	Heifers12     float64
	FemaleCalves  float64
	MaleCalves    float64
	Steers12      float64
	Steers23      float64
	FinishedSteer float64
	Bulls         float64
}

type autoHerdYear struct {
	End    autoHerdState
	Sales  autoHerdState
	Deaths autoHerdState
}

func autoNativeModels() []AutoNativeTemplateInfo {
	return []AutoNativeTemplateInfo{
		{ID: "pecuario_completo", Name: "Investimento Pecuário — modelo completo", Description: "Base com capa, projeto, orçamento, reembolso, evolução do rebanho, produção, custeio, custos e fluxo de caixa.", Focus: "Pecuária de leite, corte e investimento pecuário"},
		{ID: "agro_irrigacao", Name: "Investimento Agropecuário / Irrigação", Description: "Base usada em projetos de irrigação e investimento, com orçamento, cronograma, reembolso, produção agrícola, rebanho, custos e capacidade de pagamento.", Focus: "Irrigação, café, lavouras e projetos agropecuários"},
	}
}

func autoSuggestNativeModel(values map[string]string, docs []autoDocument, selected string) string {
	selected = strings.TrimSpace(selected)
	if selected != "" && selected != "auto" {
		return selected
	}
	var b strings.Builder
	for _, k := range []string{"activity", "line", "bank"} {
		b.WriteString(" ")
		b.WriteString(values[k])
	}
	for _, d := range docs {
		b.WriteString(" ")
		b.WriteString(d.Name)
		b.WriteString(" ")
		if len(d.Text) > 6000 {
			b.WriteString(d.Text[:6000])
		} else {
			b.WriteString(d.Text)
		}
	}
	x := autoFold(b.String())
	if strings.Contains(x, "irrig") || strings.Contains(x, "aspers") || strings.Contains(x, "pivo") || strings.Contains(x, "caf") || strings.Contains(x, "soja") || strings.Contains(x, "milho") || strings.Contains(x, "sorgo") || strings.Contains(x, "lavour") || strings.Contains(x, "agric") {
		return "agro_irrigacao"
	}
	if strings.Contains(x, "pecuar") || strings.Contains(x, "bovin") || strings.Contains(x, "gado") || strings.Contains(x, "rebanho") || strings.Contains(x, "leite") {
		return "pecuario_completo"
	}
	return "pecuario_completo"
}

func autoExtractNativeTechnical(docs []autoDocument, master []AutoMasterField) AutoNativeTechnical {
	v := autoMasterMap(master)
	t := AutoNativeTechnical{
		StartYear:        time.Now().Year(),
		CropArea:         parseAutoDecimal(v["area"]),
		Natality:         85,
		MortalityAdults:  1,
		MortalityYoung:   3,
		MortalityCalves:  5,
		CullMatrices:     15,
		CullBulls:        0,
		PastureSupportUA: 200,
	}
	activity := strings.TrimSpace(v["activity"])
	if !autoLooksLivestock(activity) {
		t.CropName = activity
	}
	if t.CropName != "" {
		t.CropUnit = "sc"
	}

	var all strings.Builder
	for _, d := range docs {
		if d.Text == "" {
			continue
		}
		all.WriteString("\n")
		all.WriteString(d.Text)
	}
	text := all.String()
	fold := autoFold(text)

	if n := autoFindNumber(text, `(?i)(?:taxa\s+de\s+juros|juros)[^0-9]{0,30}([0-9]+(?:[\.,][0-9]+)?)\s*%`); n > 0 {
		t.InterestRate = n
	}
	if n := autoFindNumber(text, `(?i)(?:prazo)[^0-9]{0,30}([0-9]{1,3})\s*(?:mes|meses)`); n > 0 {
		t.TermMonths = int(math.Round(n))
	}
	if n := autoFindNumber(text, `(?i)(?:car[eê]ncia)[^0-9]{0,30}([0-9]{1,3})\s*(?:mes|meses)`); n > 0 {
		t.GraceMonths = int(math.Round(n))
	}
	if y := autoFindYear(text); y >= 2020 && y <= 2100 {
		t.StartYear = y
	}
	if n := autoFindNumber(text, `(?i)(?:produtividade|rendimento)[^0-9]{0,40}([0-9]+(?:[\.,][0-9]+)?)\s*(?:sc|sacas|kg|t)?\s*/?\s*ha`); n > 0 {
		t.CropProductivity = n
	}
	if n := autoFindNumber(text, `(?i)(?:pre[cç]o(?:\s+m[eé]dio)?|valor\s+unit[aá]rio)[^0-9R$]{0,25}(?:R\$\s*)?([0-9\.]+(?:,[0-9]+)?)`); n > 0 {
		t.CropPrice = n
	}

	if t.CropName == "" {
		for _, crop := range []string{"café", "soja", "milho", "sorgo", "cana", "feijão", "trigo", "laranja"} {
			if strings.Contains(fold, autoFold(crop)) {
				t.CropName = crop
				break
			}
		}
	}
	if t.CropName != "" && t.CropUnit == "" {
		t.CropUnit = "sc"
	}

	t.CowsLactating = autoFindCount(text, []string{"vacas paridas", "vacas em lactacao", "vacas em lactação", "vacas lactantes"})
	t.CowsDry = autoFindCount(text, []string{"vacas secas"})
	if t.CowsLactating == 0 && t.CowsDry == 0 {
		t.CowsLactating = autoFindCount(text, []string{"matrizes", "vacas adultas", "vacas"})
	}
	t.Heifers23 = autoFindCount(text, []string{"novilhas 2 a 3", "novilhas 2/3", "novilhas de 2 a 3"})
	t.Heifers12 = autoFindCount(text, []string{"novilhas 1 a 2", "novilhas 1/2", "novilhas de 1 a 2"})
	t.FemaleCalves = autoFindCount(text, []string{"bezerras"})
	t.MaleCalves = autoFindCount(text, []string{"bezerros"})
	t.Steers12 = autoFindCount(text, []string{"novilhos 1 a 2", "novilhos 1/2"})
	t.Steers23 = autoFindCount(text, []string{"novilhos 2 a 3", "novilhos 2/3"})
	t.FinishedSteer = autoFindCount(text, []string{"bois gordos", "bois gordo", "novilhos + 3", "novilhos mais 3"})
	t.Bulls = autoFindCount(text, []string{"touros"})

	if n := autoFindNumber(text, `(?i)(?:taxa\s+de\s+natalidade|natalidade)[^0-9]{0,30}([0-9]+(?:[\.,][0-9]+)?)\s*%`); n > 0 {
		t.Natality = n
	}
	if n := autoFindNumber(text, `(?i)(?:mortalidade\s+(?:de\s+)?adultos?|mortalidade\s+vacas)[^0-9]{0,30}([0-9]+(?:[\.,][0-9]+)?)\s*%`); n > 0 {
		t.MortalityAdults = n
	}
	if n := autoFindNumber(text, `(?i)(?:mortalidade\s+(?:de\s+)?(?:novilhas|novilhos|1\s*a\s*2|2\s*a\s*3))[^0-9]{0,30}([0-9]+(?:[\.,][0-9]+)?)\s*%`); n > 0 {
		t.MortalityYoung = n
	}
	if n := autoFindNumber(text, `(?i)(?:mortalidade\s+(?:de\s+)?bezerros?)[^0-9]{0,30}([0-9]+(?:[\.,][0-9]+)?)\s*%`); n > 0 {
		t.MortalityCalves = n
	}
	if n := autoFindNumber(text, `(?i)(?:descarte\s+(?:de\s+)?matrizes?)[^0-9]{0,30}([0-9]+(?:[\.,][0-9]+)?)\s*%`); n > 0 {
		t.CullMatrices = n
	}
	if n := autoFindNumber(text, `(?i)(?:suporte\s+(?:de\s+)?pastagens?|capacidade\s+de\s+suporte)[^0-9]{0,40}([0-9]+(?:[\.,][0-9]+)?)\s*ua`); n > 0 {
		t.PastureSupportUA = n
	}
	if n := autoFindNumber(text, `(?i)(?:litros?\s*/?\s*vaca\s*/?\s*dia|litros?\s+por\s+vaca)[^0-9]{0,20}([0-9]+(?:[\.,][0-9]+)?)`); n > 0 {
		t.MilkLitersCowDay = n
	} else if n := autoFindNumber(text, `(?i)([0-9]+(?:[\.,][0-9]+)?)\s*litros?\s*/?\s*vaca\s*/?\s*dia`); n > 0 {
		t.MilkLitersCowDay = n
	}
	if n := autoFindNumber(text, `(?i)(?:pre[cç]o\s+(?:do\s+)?leite)[^0-9R$]{0,20}(?:R\$\s*)?([0-9]+(?:[\.,][0-9]+)?)`); n > 0 {
		t.MilkPriceLiter = n
	}
	if n := autoFindNumber(text, `(?i)(?:pre[cç]o\s+(?:por\s+)?(?:animal|cabe[cç]a)|valor\s+(?:por\s+)?(?:animal|cabe[cç]a))[^0-9R$]{0,25}(?:R\$\s*)?([0-9\.]+(?:,[0-9]+)?)`); n > 0 {
		t.SalePriceHead = n
	}

	t.VetVisitCost = autoFindNamedMoney(text, []string{"assistencia veterinaria", "assist. veterinaria", "visita veterinaria"})
	t.AgronomyVisitCost = autoFindNamedMoney(text, []string{"assistencia agronomica", "assist. agronomica", "visita agronomica"})
	t.LaborMonthly = autoFindNamedMoney(text, []string{"peoes", "peão", "mao de obra", "mão de obra"})
	t.EnergyMonthly = autoFindNamedMoney(text, []string{"energia eletrica", "energia elétrica"})
	t.CorralAnnual = autoFindNamedMoney(text, []string{"manutencao curral", "manutenção curral", "currais"})
	t.IrrigationAnnual = autoFindNamedMoney(text, []string{"manutencao sistema de irrigacao", "manutenção sistema de irrigação", "manutencao irrigacao", "manutenção irrigação"})

	return t
}

func autoNativeTechnicalFromForm(r *http.Request, fallback AutoNativeTechnical) AutoNativeTechnical {
	getF := func(name string, current float64) float64 {
		v := strings.TrimSpace(r.FormValue(name))
		if v == "" {
			return current
		}
		return parseAutoDecimal(v)
	}
	getI := func(name string, current int) int {
		v := strings.TrimSpace(r.FormValue(name))
		if v == "" {
			return current
		}
		n, _ := strconv.Atoi(regexp.MustCompile(`[^0-9]`).ReplaceAllString(v, ""))
		if n == 0 {
			return current
		}
		return n
	}
	fallback.StartYear = getI("tech_start_year", fallback.StartYear)
	fallback.InterestRate = getF("tech_interest", fallback.InterestRate)
	fallback.TermMonths = getI("tech_term", fallback.TermMonths)
	fallback.GraceMonths = getI("tech_grace", fallback.GraceMonths)
	if v := strings.TrimSpace(r.FormValue("tech_crop_name")); v != "" {
		fallback.CropName = v
	}
	if v := strings.TrimSpace(r.FormValue("tech_crop_unit")); v != "" {
		fallback.CropUnit = v
	}
	fallback.CropArea = getF("tech_crop_area", fallback.CropArea)
	fallback.CropProductivity = getF("tech_crop_productivity", fallback.CropProductivity)
	fallback.CropPrice = getF("tech_crop_price", fallback.CropPrice)
	fallback.CowsLactating = getF("tech_cows_lactating", fallback.CowsLactating)
	fallback.CowsDry = getF("tech_cows_dry", fallback.CowsDry)
	fallback.Heifers23 = getF("tech_heifers23", fallback.Heifers23)
	fallback.Heifers12 = getF("tech_heifers12", fallback.Heifers12)
	fallback.FemaleCalves = getF("tech_female_calves", fallback.FemaleCalves)
	fallback.MaleCalves = getF("tech_male_calves", fallback.MaleCalves)
	fallback.Steers12 = getF("tech_steers12", fallback.Steers12)
	fallback.Steers23 = getF("tech_steers23", fallback.Steers23)
	fallback.FinishedSteer = getF("tech_finished", fallback.FinishedSteer)
	fallback.Bulls = getF("tech_bulls", fallback.Bulls)
	fallback.Natality = getF("tech_natality", fallback.Natality)
	fallback.MortalityAdults = getF("tech_mort_adults", fallback.MortalityAdults)
	fallback.MortalityYoung = getF("tech_mort_young", fallback.MortalityYoung)
	fallback.MortalityCalves = getF("tech_mort_calves", fallback.MortalityCalves)
	fallback.CullMatrices = getF("tech_cull_matrices", fallback.CullMatrices)
	fallback.CullBulls = getF("tech_cull_bulls", fallback.CullBulls)
	fallback.PastureSupportUA = getF("tech_support_ua", fallback.PastureSupportUA)
	fallback.MilkLitersCowDay = getF("tech_milk_liters", fallback.MilkLitersCowDay)
	fallback.MilkPriceLiter = getF("tech_milk_price", fallback.MilkPriceLiter)
	fallback.SalePriceHead = getF("tech_sale_price_head", fallback.SalePriceHead)
	fallback.VetVisitCost = getF("tech_vet_cost", fallback.VetVisitCost)
	fallback.AgronomyVisitCost = getF("tech_agronomy_cost", fallback.AgronomyVisitCost)
	fallback.LaborMonthly = getF("tech_labor_monthly", fallback.LaborMonthly)
	fallback.EnergyMonthly = getF("tech_energy_monthly", fallback.EnergyMonthly)
	fallback.CorralAnnual = getF("tech_corral_annual", fallback.CorralAnnual)
	fallback.IrrigationAnnual = getF("tech_irrigation_annual", fallback.IrrigationAnnual)
	return fallback
}

func autoFindNumber(text, pattern string) float64 {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0
	}
	return parseAutoDecimal(m[1])
}
func autoFindYear(text string) int {
	re := regexp.MustCompile(`\b(20[2-9][0-9])\b`)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}
func autoFindCount(text string, labels []string) float64 {
	for _, label := range labels {
		q := regexp.QuoteMeta(label)
		if n := autoFindNumber(text, `(?im)^\s*`+q+`[^0-9\n]{0,35}([0-9]{1,5})(?:\s*cab|\s*cabe[cç]as?)?\b`); n > 0 {
			return n
		}
		if n := autoFindNumber(text, `(?im)^\s*([0-9]{1,5})\s*(?:cab|cabe[cç]as?)?[^\n]{0,20}`+q+`\b`); n > 0 {
			return n
		}
	}
	return 0
}
func autoFindNamedMoney(text string, labels []string) float64 {
	for _, label := range labels {
		parts := strings.Fields(label)
		if len(parts) == 0 {
			continue
		}
		var pat []string
		for _, part := range parts {
			pat = append(pat, regexp.QuoteMeta(part))
		}
		re := regexp.MustCompile(`(?is)` + strings.Join(pat, `\s+`) + `.{0,180}?R\$\s*([0-9\.]+(?:,[0-9]{1,2})?)`)
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			return parseAutoMoney(m[0])
		}
	}
	return 0
}
func autoLooksLivestock(v string) bool {
	x := autoFold(v)
	return strings.Contains(x, "pecuar") || strings.Contains(x, "bovin") || strings.Contains(x, "gado") || strings.Contains(x, "leite") || strings.Contains(x, "rebanho")
}
func valueOrDefault(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return strings.TrimSpace(v)
}
func nonZeroOr(v, d float64) float64 {
	if v == 0 {
		return d
	}
	return v
}
