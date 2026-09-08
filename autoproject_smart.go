package main

import (
	"regexp"
	"strings"
	"unicode"
)

type autoDocRole string

const (
	autoRolePrimary  autoDocRole = "primary"
	autoRoleProperty autoDocRole = "property"
	autoRoleCroqui   autoDocRole = "croqui"
	autoRoleQuote    autoDocRole = "quote"
	autoRolePermit   autoDocRole = "permit"
	autoRoleSanitary autoDocRole = "sanitary"
	autoRoleOther    autoDocRole = "other"
)

func autoFold(v string) string {
	r := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"é", "e", "ê", "e", "ë", "e",
		"í", "i", "ï", "i",
		"ó", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ü", "u", "ç", "c",
	)
	return r.Replace(strings.ToLower(v))
}

func classifyAutoDoc(source, text string) autoDocRole {
	name := autoFold(source)
	sample := autoFold(text)
	contains := func(parts ...string) bool {
		for _, p := range parts {
			if strings.Contains(name, p) {
				return true
			}
		}
		return false
	}
	if contains("proposta", "projeto", "laudo", "recomendacao", "financiamento", "cadastro", "caf", "dap") {
		return autoRolePrimary
	}
	if contains("croqui") {
		return autoRoleCroqui
	}
	if contains("ccir", "itr", "inteiro teor", "matricula", "escritura", "certidao") {
		return autoRoleProperty
	}
	if contains("orcamento", "proposta comercial", "cotacao") {
		return autoRoleQuote
	}
	if contains("outorga", "portaria") {
		return autoRolePermit
	}
	if contains("sanitaria", "sanitario", "sanidade") {
		return autoRoleSanitary
	}
	if strings.Contains(sample, "proponente") && strings.Contains(sample, "financi") {
		return autoRolePrimary
	}
	return autoRoleOther
}

func autoRoleAllows(role autoDocRole, key string) bool {
	switch role {
	case autoRolePrimary:
		return true
	case autoRoleCroqui:
		return map[string]bool{"producer": true, "cpf": true, "property": true, "registry": true, "municipality": true, "area": true, "technician": true, "crea": true}[key]
	case autoRoleProperty:
		return map[string]bool{"property": true, "registry": true, "municipality": true, "possession": true, "area": true}[key]
	case autoRolePermit:
		return map[string]bool{"property": true, "municipality": true}[key]
	case autoRoleQuote:
		// Orçamento serve para conferir custo, mas não deve definir sozinho
		// CPF, município ou valor de financiamento do produtor.
		return false
	case autoRoleSanitary:
		// Ficha sanitária pode conter titulares, estabelecimentos e CPFs que
		// não são necessariamente o proponente do financiamento.
		return false
	default:
		return false
	}
}

func extractAutoFieldsSmart(text, source string) []autoCandidate {
	role := classifyAutoDoc(source, text)
	base := extractAutoFields(text, source)
	var out []autoCandidate
	seen := map[string]bool{}
	for _, c := range base {
		if !autoRoleAllows(role, c.Key) || !autoCandidatePlausible(c) {
			continue
		}
		k := c.Key + "|" + normalizeAutoByKey(c.Key, c.Value)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, c)
	}
	return out
}

func autoCandidatePlausible(c autoCandidate) bool {
	v := strings.TrimSpace(c.Value)
	if v == "" {
		return false
	}
	switch c.Key {
	case "cpf":
		d := regexp.MustCompile(`\D`).ReplaceAllString(v, "")
		return validCPFOrCNPJ(d)
	case "municipality":
		return plausibleMunicipality(v)
	case "producer", "technician":
		return plausiblePersonName(v)
	case "activity":
		return plausibleActivity(v)
	case "property":
		return plausibleProperty(v)
	case "registry":
		return len(v) <= 80 && regexp.MustCompile(`\d`).MatchString(v)
	case "area":
		f := parseAutoDecimal(v)
		return f > 0 && f < 1000000
	case "amount":
		f := parseAutoMoney(v)
		return f > 0 && f < 1000000000
	}
	return len(v) <= 120
}

func plausibleMunicipality(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) < 2 || len(v) > 70 || regexp.MustCompile(`\d`).MatchString(v) {
		return false
	}
	low := autoFold(v)
	for _, bad := range []string{"cronologia", "captacao", "vazao", "latitude", "longitude", "art.", " artigo ", "estado:", "localidade:", "confrontante", "testemun", "mes/", "l/s", "m3", "coordenad"} {
		if strings.Contains(low, bad) {
			return false
		}
	}
	words := strings.Fields(v)
	if len(words) > 7 {
		return false
	}
	letters := 0
	other := 0
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsSpace(r) || r == '-' || r == '/' {
			letters++
		} else {
			other++
		}
	}
	return letters > other*4
}

func plausiblePersonName(v string) bool {
	if len(v) < 5 || len(v) > 90 || regexp.MustCompile(`\d`).MatchString(v) {
		return false
	}
	low := autoFold(v)
	for _, bad := range []string{"municipio", "propriedade", "financiamento", "cronologia", "portaria", "art.", "cpf", "cnpj", "agencia"} {
		if strings.Contains(low, bad) {
			return false
		}
	}
	return len(strings.Fields(v)) >= 2
}

func plausibleActivity(v string) bool {
	low := autoFold(v)
	if len(v) > 40 {
		return false
	}
	for _, ok := range []string{"cafe", "milho", "soja", "sorgo", "pecuaria", "bovino", "leite", "corte", "irrigacao", "pastagem"} {
		if strings.Contains(low, ok) {
			return true
		}
	}
	return false
}

func plausibleProperty(v string) bool {
	if len(v) < 3 || len(v) > 100 {
		return false
	}
	low := autoFold(v)
	for _, bad := range []string{"cronologia", "captacao", "vazao", "portaria", "art.", "coordenad", "testemun"} {
		if strings.Contains(low, bad) {
			return false
		}
	}
	return true
}

func validCPFOrCNPJ(d string) bool {
	if len(d) == 11 {
		return validCPF(d)
	}
	if len(d) == 14 {
		return validCNPJ(d)
	}
	return false
}

func allDigitsEqual(d string) bool {
	for i := 1; i < len(d); i++ {
		if d[i] != d[0] {
			return false
		}
	}
	return true
}

func validCPF(d string) bool {
	if len(d) != 11 || allDigitsEqual(d) {
		return false
	}
	calc := func(n int) int {
		sum := 0
		weight := n + 1
		for i := 0; i < n; i++ {
			sum += int(d[i]-'0') * (weight - i)
		}
		r := (sum * 10) % 11
		if r == 10 {
			r = 0
		}
		return r
	}
	return calc(9) == int(d[9]-'0') && calc(10) == int(d[10]-'0')
}

func validCNPJ(d string) bool {
	if len(d) != 14 || allDigitsEqual(d) {
		return false
	}
	calc := func(n int, weights []int) int {
		sum := 0
		for i := 0; i < n; i++ {
			sum += int(d[i]-'0') * weights[i]
		}
		r := sum % 11
		if r < 2 {
			return 0
		}
		return 11 - r
	}
	w1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	w2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	return calc(12, w1) == int(d[12]-'0') && calc(13, w2) == int(d[13]-'0')
}

func autoSupportDocumentFinding(source, text string) *AutoFinding {
	role := classifyAutoDoc(source, text)
	switch role {
	case autoRoleQuote:
		f := AutoFinding{Level: "info", Title: "Orçamento tratado como documento de apoio", Detail: source + ": valores, CNPJ e endereço do fornecedor não são usados para definir automaticamente o proponente ou o município do projeto."}
		return &f
	case autoRoleSanitary:
		f := AutoFinding{Level: "info", Title: "Ficha sanitária tratada como documento de apoio", Detail: source + ": CPF, nome e estabelecimento encontrados nela não são assumidos como dados do proponente sem confirmação em documento principal."}
		return &f
	case autoRolePermit:
		f := AutoFinding{Level: "info", Title: "Outorga tratada com filtro reforçado", Detail: source + ": textos legais, vazões e cronologias não são aceitos como município, atividade ou dados cadastrais."}
		return &f
	}
	return nil
}
