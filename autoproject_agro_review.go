package main

import (
	"fmt"
	"regexp"
	"strings"
)

// AutoTechnicalField é uma entrada técnica exibida para conferência antes da
// geração do projeto. Status segue as classes já usadas pela Ficha Mestre:
// ok = encontrado objetivo; compatible = encontrado, mas requer conferência;
// missing = não encontrado; info = calculado pelo Excel.
type AutoTechnicalField struct {
	Key       string
	FormName  string
	Group     string
	Label     string
	Value     string
	Unit      string
	Status    string
	Badge     string
	Source    string
	Critical  bool
	Editable  bool
	Note      string
}

type AutoTechnicalGroup struct {
	Name   string
	Fields []AutoTechnicalField
}

type AutoTechnicalReview struct {
	Groups     []AutoTechnicalGroup
	Found      int
	Review     int
	Missing    int
	CriticalMissing int
}

func autoTechFmt(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return strings.ReplaceAll(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), "."), ".", ",")
}

func autoMasterValue(master []AutoMasterField, key string) (string, string, bool) {
	for _, f := range master {
		if f.Key != key || strings.TrimSpace(f.Value) == "" {
			continue
		}
		return strings.TrimSpace(f.Value), strings.Join(f.Sources, " · "), true
	}
	return "", "", false
}

func autoDocPattern(docs []autoDocument, pattern string) (float64, string, bool) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0, "", false
	}
	for _, d := range docs {
		if strings.TrimSpace(d.Text) == "" {
			continue
		}
		m := re.FindStringSubmatch(d.Text)
		if len(m) < 2 {
			continue
		}
		return parseAutoDecimal(m[1]), d.Name, true
	}
	return 0, "", false
}

func autoDocMoney(docs []autoDocument, labels []string) (float64, string, bool) {
	for _, d := range docs {
		if strings.TrimSpace(d.Text) == "" {
			continue
		}
		if v := autoFindNamedMoney(d.Text, labels); v > 0 {
			return v, d.Name, true
		}
	}
	return 0, "", false
}

// Tabelas de evolução normalmente trazem: categoria, coeficiente UA, cabeças,
// UA e demais anos. A expressão abaixo pula o coeficiente UA quando ele está
// presente e captura a primeira quantidade de cabeças, inclusive zero.
func autoDocHerdCount(docs []autoDocument, labels []string) (float64, string, bool) {
	for _, d := range docs {
		if strings.TrimSpace(d.Text) == "" {
			continue
		}
		for _, label := range labels {
			q := regexp.QuoteMeta(label)
			patterns := []string{
				`(?im)^\s*` + q + `\s+(?:[01](?:[\.,][0-9]{1,2})\s+)?([0-9]{1,5})(?:\s|$)`,
				`(?im)^\s*` + q + `[^0-9\n]{0,25}([0-9]{1,5})(?:\s*cab|\s*cabe[cç]as?)?\b`,
			}
			for _, p := range patterns {
				re := regexp.MustCompile(p)
				if m := re.FindStringSubmatch(d.Text); len(m) > 1 {
					return parseAutoDecimal(m[1]), d.Name, true
				}
			}
		}
	}
	return 0, "", false
}

func autoTechnicalField(group, key, formName, label, unit, mode, value, source string, critical bool, note string) AutoTechnicalField {
	f := AutoTechnicalField{Group: group, Key: key, FormName: formName, Label: label, Unit: unit, Value: strings.TrimSpace(value), Source: strings.TrimSpace(source), Critical: critical, Editable: mode != "derived", Note: note}
	if mode == "derived" {
		f.Status, f.Badge, f.Editable = "info", "Calculado pelo Excel", false
		if f.Value == "" { f.Value = "Automático" }
		return f
	}
	if f.Value == "" {
		f.Status, f.Badge = "missing", "Faltando"
		return f
	}
	if mode == "auto" {
		f.Status, f.Badge = "ok", "Encontrado"
	} else {
		f.Status, f.Badge = "compatible", "Confirmar"
	}
	return f
}

func autoBuildAgroTechnicalReview(docs []autoDocument, master []AutoMasterField) AutoTechnicalReview {
	var all []AutoTechnicalField
	add := func(f AutoTechnicalField) { all = append(all, f) }

	area, areaSource, _ := autoMasterValue(master, "area")
	line, lineSource, _ := autoMasterValue(master, "line")
	activity, activitySource, _ := autoMasterValue(master, "activity")
	amount, amountSource, _ := autoMasterValue(master, "amount")
	add(autoTechnicalField("Projeto e crédito", "project_area", "tech_crop_area", "Área do projeto", "ha", "review", area, areaSource, true, "Conferir com KML/Geo Mapa; não confundir com área total da matrícula."))
	add(autoTechnicalField("Projeto e crédito", "line", "tech_line", "Linha de crédito", "", "auto", line, lineSource, true, "A linha precisa ser compatível com banco, porte e vigência."))
	add(autoTechnicalField("Projeto e crédito", "purpose", "tech_purpose", "Atividade / finalidade", "", "review", activity, activitySource, true, "Confirmar se a finalidade descreve exatamente o objeto financiado."))
	add(autoTechnicalField("Projeto e crédito", "amount", "tech_amount", "Valor pretendido", "R$", "review", amount, amountSource, true, "O total final deve ser calculado a partir dos itens do orçamento."))

	if v, src, ok := autoDocPattern(docs, `(?i)(?:taxa\s+de\s+juros|juros)[^0-9]{0,35}([0-9]+(?:[\.,][0-9]+)?)\s*%`); ok {
		add(autoTechnicalField("Projeto e crédito", "interest", "tech_interest", "Taxa de juros", "% a.a.", "review", autoTechFmt(v), src, true, "Validar banco, linha e safra antes de aceitar."))
	} else {
		add(autoTechnicalField("Projeto e crédito", "interest", "tech_interest", "Taxa de juros", "% a.a.", "review", "", "", true, "Buscar em regra oficial do banco/Plano Safra ou confirmar."))
	}
	termVal, termSrc := "", ""
	if v, src, ok := autoDocPattern(docs, `(?i)(?:prazo)[^0-9]{0,30}([0-9]{1,3})\s*(?:anos?|ano)\b`); ok {
		termVal, termSrc = autoTechFmt(v*12), src
	} else if v, src, ok := autoDocPattern(docs, `(?i)(?:prazo)[^0-9]{0,30}([0-9]{1,3})\s*(?:mes|meses|mês|m[eê]ses)\b`); ok {
		termVal, termSrc = autoTechFmt(v), src
	}
	add(autoTechnicalField("Projeto e crédito", "term", "tech_term", "Prazo", "meses", "review", termVal, termSrc, true, "Não criar prazo padrão quando não houver fonte."))
	graceVal, graceSrc := "", ""
	if v, src, ok := autoDocPattern(docs, `(?i)(?:car[eê]ncia)[^0-9]{0,30}([0-9]{1,3})\s*(?:anos?|ano)\b`); ok {
		graceVal, graceSrc = autoTechFmt(v*12), src
	} else if v, src, ok := autoDocPattern(docs, `(?i)(?:car[eê]ncia)[^0-9]{0,30}([0-9]{1,3})\s*(?:mes|meses|mês|m[eê]ses)\b`); ok {
		graceVal, graceSrc = autoTechFmt(v), src
	}
	add(autoTechnicalField("Projeto e crédito", "grace", "tech_grace", "Carência", "meses", "confirm", graceVal, graceSrc, true, "Se não estiver expressa na documentação, perguntar; nunca assumir."))

	herd := []struct{key, form, label string; labels []string; critical bool}{
		{"matrizes", "tech_cows_lactating", "Matrizes", []string{"Matrizes", "Vacas adultas", "Vacas"}, true},
		{"novilhas_23", "tech_heifers23", "Novilhas 2/3 anos", []string{"Novilhas 2/3 anos", "Novilhas 2 a 3 anos", "Novilhas 2/3"}, true},
		{"novilhas_12", "tech_heifers12", "Novilhas 1/2 anos", []string{"Novilhas 1/2 anos", "Novilhas 1 a 2 anos", "Novilhas 1/2"}, true},
		{"bezerras", "tech_female_calves", "Bezerras", []string{"Bezerras", "Bezerras 0 a 12 meses"}, true},
		{"bezerros", "tech_male_calves", "Bezerros", []string{"Bezerros", "Bezerros 0 a 12 meses"}, false},
		{"novilhos_12", "tech_steers12", "Novilhos 1/2 anos", []string{"Novilhos 1/2 anos", "Novilhos 1 a 2 anos", "Novilhos 12 a 24 meses"}, false},
		{"novilhos_23", "tech_steers23", "Novilhos 2/3 anos", []string{"Novilhos 2/3 anos", "Novilhos 2 a 3 anos", "Novilhos 25 a 36 meses"}, true},
		{"novilhos_mais_3", "tech_finished_steer", "Novilhos +3 anos", []string{"Novilhos + 3 anos", "Novilhos +3 anos", "Novilhos mais 3", "Bois gordos"}, false},
		{"touros", "tech_bulls", "Touros", []string{"Touros"}, true},
	}
	for _, h := range herd {
		val, src := "", ""
		if v, s, ok := autoDocHerdCount(docs, h.labels); ok { val, src = autoTechFmt(v), s }
		add(autoTechnicalField("Rebanho inicial", h.key, h.form, h.label, "cabeças", "review", val, src, h.critical, "Conferir com ficha sanitária/cadastro e fechamento do total do rebanho."))
	}

	rates := []struct{key, form, label, pattern string; critical bool}{
		{"natality", "tech_natality", "Natalidade", `(?i)(?:taxa\s+de\s+natalidade|natalidade)\s*(?:\(%\))?[^0-9]{0,20}([0-9]+(?:[\.,][0-9]+)?)`, true},
		{"mortality_adult", "tech_mortality_adults", "Mortalidade adultos", `(?i)mortalidade\s+(?:de\s+)?(?:adultos?|vacas)[^0-9]{0,25}([0-9]+(?:[\.,][0-9]+)?)`, true},
		{"mortality_young", "tech_mortality_young", "Mortalidade 1/2 anos", `(?i)mortalidade\s+(?:de\s+)?(?:1\s*/?\s*2|novilhas|novilhos)[^0-9]{0,25}([0-9]+(?:[\.,][0-9]+)?)`, true},
		{"mortality_calf", "tech_mortality_calves", "Mortalidade bezerros", `(?i)mortalidade\s+(?:de\s+)?bezerros?[^0-9]{0,25}([0-9]+(?:[\.,][0-9]+)?)`, true},
		{"cull_matrices", "tech_cull_matrices", "Descarte de matrizes", `(?i)(?:desc(?:arte)?\.?\s+matrizes|descarte\s+(?:de\s+)?matrizes)[^0-9]{0,25}([0-9]+(?:[\.,][0-9]+)?)`, true},
		{"cull_bulls", "tech_cull_bulls", "Descarte de touros", `(?i)(?:desc(?:arte)?\.?\s+touros|descarte\s+(?:de\s+)?touros)[^0-9]{0,25}([0-9]+(?:[\.,][0-9]+)?)`, false},
		{"pasture_support", "tech_pasture_support", "Suporte das pastagens", `(?i)(?:suporte\s+(?:das|de)?\s*pastagens?|capacidade\s+de\s+suporte)[^0-9]{0,30}([0-9]+(?:[\.,][0-9]+)?)`, true},
	}
	for _, x := range rates {
		val, src := "", ""
		if v, s, ok := autoDocPattern(docs, x.pattern); ok { val, src = autoTechFmt(v), s }
		unit := "%"
		if x.key == "pasture_support" { unit = "UA" }
		add(autoTechnicalField("Coeficientes técnicos", x.key, x.form, x.label, unit, "review", val, src, x.critical, "Parâmetro técnico encontrado não vira padrão universal; precisa ser conferido."))
	}

	// Movimento de venda é propositalmente conservador: só aparece quando há texto
	// explícito de venda/planejamento próximo da categoria. A mera existência do
	// animal no rebanho não autoriza inferir a venda.
	saleVal, saleSrc := "", ""
	if v, src, ok := autoDocPattern(docs, `(?i)(?:venda|vendas|vendidos|comercializa[cç][aã]o)[^\n]{0,80}(?:novilhos?\s*(?:\+|mais)\s*3|bois?\s+gordos?)[^0-9]{0,25}([0-9]{1,5})`); ok {
		saleVal, saleSrc = autoTechFmt(v), src
	}
	add(autoTechnicalField("Movimentos e receitas", "year1_finished_sale", "tech_year1_finished_sale", "Venda planejada de novilhos +3 no 1º ano", "cabeças", "confirm", saleVal, saleSrc, true, "Decisão técnica: não inferir apenas a partir do estoque inicial."))

	costs := []struct{key, form, label, unit string; labels []string; critical bool}{
		{"vet_cost", "tech_vet_cost", "Assistência veterinária", "R$/visita", []string{"assistencia veterinaria", "assist. veterinaria", "assistência veterinária", "visita veterinaria"}, true},
		{"agronomy_cost", "tech_agronomy_cost", "Assistência agronômica", "R$/visita", []string{"assistencia agronomica", "assist. agronomica", "assistência agronômica", "visita agronomica"}, true},
		{"labor_monthly", "tech_labor_monthly", "Mão de obra", "R$/mês", []string{"peoes", "peões", "mao de obra", "mão de obra"}, true},
		{"energy_monthly", "tech_energy_monthly", "Energia elétrica", "R$/mês", []string{"energia eletrica", "energia elétrica"}, true},
		{"corral_annual", "tech_corral_annual", "Manutenção de currais", "R$/ano", []string{"manutencao curral", "manutenção curral", "currais"}, false},
		{"irrigation_annual", "tech_irrigation_annual", "Manutenção da irrigação", "R$/ano", []string{"manutencao sistema de irrigacao", "manutenção sistema de irrigação", "manutencao irrigacao", "manutenção irrigação"}, true},
	}
	for _, c := range costs {
		val, src := "", ""
		if v, s, ok := autoDocMoney(docs, c.labels); ok { val, src = autoTechFmt(v), s }
		add(autoTechnicalField("Custos operacionais", c.key, c.form, c.label, c.unit, "review", val, src, c.critical, "Usar o custo documentado do cliente/projeto; não reutilizar custo de caso antigo."))
	}

	add(autoTechnicalField("Resultados do modelo", "herd_projection", "", "Evolução do rebanho", "", "derived", "", "Fórmulas do modelo nativo", true, "Calculada a partir do rebanho inicial, coeficientes e movimentos confirmados."))
	add(autoTechnicalField("Resultados do modelo", "production_revenue", "", "Produção e receitas", "", "derived", "", "Fórmulas do modelo nativo", true, "Não é entrada manual."))
	add(autoTechnicalField("Resultados do modelo", "cash_flow", "", "Fluxo de caixa / capacidade", "", "derived", "", "Fórmulas do modelo nativo", true, "Saída final; nunca deve ser digitada pelo AutoProjeto."))

	order := []string{"Projeto e crédito", "Rebanho inicial", "Coeficientes técnicos", "Movimentos e receitas", "Custos operacionais", "Resultados do modelo"}
	review := AutoTechnicalReview{}
	for _, groupName := range order {
		g := AutoTechnicalGroup{Name: groupName}
		for _, f := range all {
			if f.Group != groupName { continue }
			g.Fields = append(g.Fields, f)
			switch f.Status {
			case "ok": review.Found++
			case "compatible": review.Review++
			case "missing":
				review.Missing++
				if f.Critical { review.CriticalMissing++ }
			}
		}
		if len(g.Fields) > 0 { review.Groups = append(review.Groups, g) }
	}
	return review
}
