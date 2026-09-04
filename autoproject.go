package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type AutoProjectFile struct {
	Name      string
	Kind      string
	SizeLabel string
	Extracted bool
	Note      string
}

type AutoMasterField struct {
	Key     string
	Label   string
	Value   string
	Status  string
	Sources []string
}

type AutoFinding struct {
	Level  string
	Title  string
	Detail string
}

type AutoProjectView struct {
	Analyzed   bool
	Score      int
	Files      []AutoProjectFile
	Master     []AutoMasterField
	Findings   []AutoFinding
	Missing    []string
	Errors     int
	Warnings   int
	Infos      int
	Confirmed  int
	BankHint   string
	LineHint   string
	FileCount  int
	ReadCount  int
}

type autoCandidate struct {
	Key    string
	Label  string
	Value  string
	Source string
}

type autoDocument struct {
	Name       string
	Kind       string
	Text       string
	Fields     []autoCandidate
	BudgetSum  float64
	BudgetOK   bool
	BudgetNote string
}

var (
	autoCPFRe   = regexp.MustCompile(`\b\d{3}\.?\d{3}\.?\d{3}[-/]?\d{2}\b|\b\d{2}\.?\d{3}\.?\d{3}/\d{4}-?\d{2}\b`)
	autoMoneyRe = regexp.MustCompile(`(?i)R\$\s*[0-9\.]+(?:,[0-9]{1,2})?`)
	autoAreaRe  = regexp.MustCompile(`(?i)([0-9]+(?:[\.,][0-9]+)?)\s*ha\b`)
	autoCREARe  = regexp.MustCompile(`(?i)\b[0-9]{7,12}/[A-Z]{2}\b`)
)

var autoFieldOrder = []struct {
	Key   string
	Label string
}{
	{"producer", "Produtor / Proponente"},
	{"cpf", "CPF / CNPJ"},
	{"property", "Propriedade"},
	{"registry", "Matrícula"},
	{"municipality", "Município"},
	{"possession", "Condição de posse"},
	{"bank", "Banco"},
	{"agency", "Agência"},
	{"line", "Linha / enquadramento"},
	{"activity", "Atividade / cultura"},
	{"area", "Área financiada"},
	{"amount", "Valor pretendido"},
	{"technician", "Técnico responsável"},
	{"crea", "CREA"},
}

func (a *App) autoProjectPage(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Data: AutoProjectView{}})
}

func (a *App) autoProjectAnalyze(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida. Atualize a página e tente novamente.", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Error: "Não foi possível ler os arquivos enviados."})
		return
	}

	view := AutoProjectView{Analyzed: true, BankHint: strings.TrimSpace(r.FormValue("bank_hint")), LineHint: strings.TrimSpace(r.FormValue("line_hint"))}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Error: "Envie pelo menos um documento ou planilha.", Data: view})
		return
	}
	if len(files) > 20 {
		files = files[:20]
		view.Findings = append(view.Findings, AutoFinding{Level: "warning", Title: "Limite de arquivos", Detail: "Foram analisados os primeiros 20 arquivos deste envio."})
	}

	var docs []autoDocument
	for _, fh := range files {
		view.FileCount++
		finfo, doc := analyzeAutoFile(fh)
		view.Files = append(view.Files, finfo)
		if doc.Text != "" {
			view.ReadCount++
			doc.Fields = extractAutoFields(doc.Text, doc.Name)
			doc.BudgetSum, doc.BudgetOK, doc.BudgetNote = detectAutoBudget(doc.Text)
			docs = append(docs, doc)
		}
	}

	if view.BankHint != "" {
		docs = append(docs, autoDocument{Name: "Informado no formulário", Fields: []autoCandidate{{Key: "bank", Label: "Banco", Value: view.BankHint, Source: "Informado no formulário"}}})
	}
	if view.LineHint != "" {
		docs = append(docs, autoDocument{Name: "Informado no formulário", Fields: []autoCandidate{{Key: "line", Label: "Linha / enquadramento", Value: view.LineHint, Source: "Informado no formulário"}}})
	}

	view.Master, view.Findings = buildAutoMaster(docs, view.Findings)
	view.Findings = append(view.Findings, detectFilenameBankMismatch(docs)...)
	view.Findings = append(view.Findings, detectBudgetFindings(docs)...)
	view.Missing = autoMissingFields(view.Master)
	view.Score = autoProjectScore(view.Master, view.Findings)
	for _, f := range view.Findings {
		switch f.Level {
		case "error":
			view.Errors++
		case "warning":
			view.Warnings++
		default:
			view.Infos++
		}
	}
	for _, m := range view.Master {
		if m.Status == "ok" || m.Status == "compatible" {
			view.Confirmed++
		}
	}

	a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Data: view})
}

func analyzeAutoFile(fh *multipart.FileHeader) (AutoProjectFile, autoDocument) {
	info := AutoProjectFile{Name: filepath.Base(fh.Filename), Kind: strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), "."), SizeLabel: autoSize(fh.Size)}
	doc := autoDocument{Name: info.Name, Kind: info.Kind}
	if fh.Size > 18<<20 {
		info.Note = "Arquivo maior que 18 MB; não foi processado nesta versão."
		return info, doc
	}
	f, err := fh.Open()
	if err != nil {
		info.Note = "Não foi possível abrir o arquivo."
		return info, doc
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 18<<20))
	if err != nil {
		info.Note = "Falha durante a leitura."
		return info, doc
	}

	var text string
	switch info.Kind {
	case "docx":
		text, err = autoDOCXText(b)
	case "xlsx", "xlsm":
		text, err = autoXLSXText(b)
	case "txt", "csv":
		text = string(b)
	case "kml":
		text = string(b)
		if area := autoKMLArea(text); area > 0 {
			text += fmt.Sprintf("\nÁrea KML calculada: %.4f ha\n", area)
		}
	case "pdf":
		info.Note = "PDF recebido. A leitura profunda de PDF ficará na próxima etapa; use também DOCX/XLSX quando houver."
		return info, doc
	default:
		info.Note = "Formato ainda não interpretado automaticamente."
		return info, doc
	}
	if err != nil || strings.TrimSpace(text) == "" {
		info.Note = "O arquivo foi recebido, mas o conteúdo não pôde ser extraído."
		return info, doc
	}
	info.Extracted = true
	info.Note = "Conteúdo lido e comparado."
	doc.Text = text
	return info, doc
}

func autoDOCXText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	for _, zf := range zr.File {
		if zf.Name != "word/document.xml" {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		dec := xml.NewDecoder(rc)
		var out strings.Builder
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				if t.Name.Local == "tab" {
					out.WriteString(" ")
				}
			case xml.CharData:
				s := strings.TrimSpace(string(t))
				if s != "" {
					if out.Len() > 0 {
						out.WriteString(" ")
					}
					out.WriteString(s)
				}
			case xml.EndElement:
				switch t.Name.Local {
				case "tc":
					out.WriteString(" | ")
				case "tr", "p":
					out.WriteString("\n")
				}
			}
		}
		return cleanupAutoText(out.String()), nil
	}
	return "", fmt.Errorf("word/document.xml ausente")
}

type autoSharedStrings struct {
	Items []struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

type autoSheet struct {
	Rows []struct {
		Cells []struct {
			Ref string `xml:"r,attr"`
			T   string `xml:"t,attr"`
			V   string `xml:"v"`
			IS  struct {
				T string `xml:"t"`
			} `xml:"is"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

func autoXLSXText(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	var shared []string
	for _, zf := range zr.File {
		if zf.Name != "xl/sharedStrings.xml" {
			continue
		}
		b, err := readAutoZip(zf)
		if err != nil {
			return "", err
		}
		var ss autoSharedStrings
		if err := xml.Unmarshal(b, &ss); err == nil {
			for _, si := range ss.Items {
				v := si.T
				for _, r := range si.R {
					v += r.T
				}
				shared = append(shared, strings.TrimSpace(v))
			}
		}
	}
	var out strings.Builder
	for _, zf := range zr.File {
		if !strings.HasPrefix(zf.Name, "xl/worksheets/") || !strings.HasSuffix(zf.Name, ".xml") {
			continue
		}
		b, err := readAutoZip(zf)
		if err != nil {
			continue
		}
		var sh autoSheet
		if err := xml.Unmarshal(b, &sh); err != nil {
			continue
		}
		for _, row := range sh.Rows {
			var vals []string
			for _, c := range row.Cells {
				v := strings.TrimSpace(c.V)
				switch c.T {
				case "s":
					i, _ := strconv.Atoi(v)
					if i >= 0 && i < len(shared) {
						v = shared[i]
					}
				case "inlineStr":
					v = strings.TrimSpace(c.IS.T)
				}
				if v != "" {
					vals = append(vals, v)
				}
			}
			if len(vals) > 0 {
				out.WriteString(strings.Join(vals, " | "))
				out.WriteByte('\n')
			}
		}
	}
	if out.Len() == 0 {
		return "", fmt.Errorf("planilha sem células legíveis")
	}
	return cleanupAutoText(out.String()), nil
}

func readAutoZip(zf *zip.File) ([]byte, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func cleanupAutoText(s string) string {
	s = strings.ReplaceAll(s, "\r", "\n")
	s = regexp.MustCompile(`[ \t]+`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func extractAutoFields(text, source string) []autoCandidate {
	lines := autoLines(text)
	var out []autoCandidate
	add := func(key, label, value string) {
		value = cleanAutoValue(value)
		if value == "" {
			return
		}
		out = append(out, autoCandidate{Key: key, Label: label, Value: value, Source: source})
	}

	producer := autoAfterLabels(lines, []string{"nome do proponente", "proponente"})
	if producer == "" {
		producer = autoProducerAfterSection(lines)
	}
	add("producer", "Produtor / Proponente", producer)

	if m := autoCPFRe.FindString(text); m != "" {
		add("cpf", "CPF / CNPJ", m)
	}
	add("property", "Propriedade", autoAfterLabels(lines, []string{"nome da propriedade", "imóvel beneficiado"}))
	add("registry", "Matrícula", autoAfterLabels(lines, []string{"matrícula"}))
	add("municipality", "Município", autoAfterLabels(lines, []string{"município"}))
	add("agency", "Agência", autoAfterLabels(lines, []string{"n.º agência", "nº agência", "agência"}))
	add("technician", "Técnico responsável", autoAfterLabels(lines, []string{"nome do técnico"}))
	if m := autoCREARe.FindString(text); m != "" {
		add("crea", "CREA", strings.ToUpper(m))
	}

	low := strings.ToLower(text)
	for _, bank := range []string{"Banco do Brasil", "Sicoob", "Cresol", "Sicredi", "Credicom"} {
		if strings.Contains(low, strings.ToLower(bank)) {
			add("bank", "Banco", bank)
			break
		}
	}
	for _, line := range []string{"PRONAF", "PRONAMP", "RenovAgro", "Moderfrota"} {
		if strings.Contains(low, strings.ToLower(line)) {
			add("line", "Linha / enquadramento", line)
			break
		}
	}
	activity := autoAfterLabels(lines, []string{"cultura", "atividade"})
	if activity == "" {
		for _, v := range []string{"Café", "Milho", "Soja", "Sorgo", "Pecuária", "Bovinos"} {
			if strings.Contains(low, strings.ToLower(v)) {
				activity = v
				break
			}
		}
	}
	add("activity", "Atividade / cultura", activity)

	area := autoNumberAfterLabels(lines, []string{"área à financiar", "área a financiar", "área beneficiada", "área kml calculada", "área total"}, autoAreaRe)
	if area == "" {
		if m := autoAreaRe.FindStringSubmatch(text); len(m) > 1 {
			area = normalizeDecimalDisplay(m[1]) + " ha"
		}
	}
	add("area", "Área financiada", area)

	amount := autoMoneyAfterLabels(lines, []string{"valor pretendido", "financiamento pretendido", "valor financiado"})
	if amount == "" {
		if m := autoMoneyRe.FindString(text); m != "" {
			amount = moneyDisplay(parseAutoMoney(m))
		}
	}
	add("amount", "Valor pretendido", amount)

	possession := ""
	for _, v := range []string{"Proprietário", "Arrendatário", "Comodatário", "Parceiro"} {
		if regexp.MustCompile(`(?i)\(\s*[xX]\s*\)\s*` + regexp.QuoteMeta(v)).MatchString(text) {
			possession = v
			break
		}
	}
	add("possession", "Condição de posse", possession)
	return out
}

func autoLines(text string) []string {
	var lines []string
	for _, raw := range strings.Split(text, "\n") {
		line := strings.Trim(strings.TrimSpace(raw), "|")
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func autoAfterLabels(lines, labels []string) string {
	for i, line := range lines {
		low := strings.ToLower(line)
		for _, label := range labels {
			idx := strings.Index(low, label)
			if idx < 0 {
				continue
			}
			rest := strings.TrimSpace(line[idx+len(label):])
			rest = strings.TrimLeft(rest, ": -|_")
			if rest != "" && !strings.EqualFold(rest, "Nome") {
				return firstAutoCell(rest)
			}
			if i+1 < len(lines) {
				n := firstAutoCell(lines[i+1])
				nlow := strings.ToLower(n)
				for _, prefix := range []string{"nome:", "valor:", "cpf/cnpj:"} {
					if strings.HasPrefix(nlow, prefix) {
						n = strings.TrimSpace(n[len(prefix):])
					}
				}
				if n != "" {
					return n
				}
			}
		}
	}
	return ""
}

func autoProducerAfterSection(lines []string) string {
	for i, line := range lines {
		if strings.EqualFold(strings.Trim(strings.TrimSpace(line), ":"), "Produtor") && i+1 < len(lines) {
			n := lines[i+1]
			if idx := strings.Index(strings.ToLower(n), "nome:"); idx >= 0 {
				return strings.TrimSpace(n[idx+5:])
			}
		}
	}
	return ""
}

func firstAutoCell(v string) string {
	if idx := strings.Index(v, "|"); idx >= 0 {
		return strings.TrimSpace(v[:idx])
	}
	return strings.TrimSpace(v)
}

func autoNumberAfterLabels(lines, labels []string, re *regexp.Regexp) string {
	for i, line := range lines {
		low := strings.ToLower(line)
		matched := false
		for _, label := range labels {
			if strings.Contains(low, label) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		for j := i; j <= i+1 && j < len(lines); j++ {
			if m := re.FindStringSubmatch(lines[j]); len(m) > 1 {
				return normalizeDecimalDisplay(m[1]) + " ha"
			}
		}
	}
	return ""
}

func autoMoneyAfterLabels(lines, labels []string) string {
	for i, line := range lines {
		low := strings.ToLower(line)
		matched := false
		for _, label := range labels {
			if strings.Contains(low, label) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		for j := i; j <= i+2 && j < len(lines); j++ {
			if m := autoMoneyRe.FindString(lines[j]); m != "" {
				return moneyDisplay(parseAutoMoney(m))
			}
		}
	}
	return ""
}

func buildAutoMaster(docs []autoDocument, findings []AutoFinding) ([]AutoMasterField, []AutoFinding) {
	byKey := map[string][]autoCandidate{}
	for _, d := range docs {
		for _, c := range d.Fields {
			byKey[c.Key] = append(byKey[c.Key], c)
		}
	}
	var master []AutoMasterField
	for _, spec := range autoFieldOrder {
		cs := byKey[spec.Key]
		if len(cs) == 0 {
			master = append(master, AutoMasterField{Key: spec.Key, Label: spec.Label, Status: "missing"})
			continue
		}
		groups := map[string][]autoCandidate{}
		for _, c := range cs {
			groups[normalizeAutoByKey(spec.Key, c.Value)] = append(groups[normalizeAutoByKey(spec.Key, c.Value)], c)
		}
		keys := make([]string, 0, len(groups))
		for k := range groups {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return len(groups[keys[i]]) > len(groups[keys[j]]) })
		winner := groups[keys[0]][0]
		sources := uniqueAutoSources(groups[keys[0]])
		status := "ok"
		if len(groups) > 1 {
			if autoGroupsCompatible(spec.Key, groups) {
				status = "compatible"
				findings = append(findings, AutoFinding{Level: "info", Title: spec.Label + " compatível", Detail: autoGroupDetail(spec.Label, groups) + ". As diferenças estão dentro da tolerância automática."})
			} else {
				status = "error"
				findings = append(findings, AutoFinding{Level: "error", Title: "Divergência em " + spec.Label, Detail: autoGroupDetail(spec.Label, groups) + ". Confirme qual informação deve prevalecer antes de gerar a versão final."})
			}
		}
		master = append(master, AutoMasterField{Key: spec.Key, Label: spec.Label, Value: winner.Value, Status: status, Sources: sources})
	}
	return master, findings
}

func normalizeAutoByKey(key, v string) string {
	switch key {
	case "cpf":
		return regexp.MustCompile(`\D`).ReplaceAllString(v, "")
	case "area":
		return fmt.Sprintf("%.4f", parseAutoDecimal(v))
	case "amount":
		return fmt.Sprintf("%.2f", parseAutoMoney(v))
	default:
		v = strings.ToLower(strings.TrimSpace(v))
		v = strings.Trim(v, ".,;:-_")
		return strings.Join(strings.Fields(v), " ")
	}
}

func autoGroupsCompatible(key string, groups map[string][]autoCandidate) bool {
	if key != "area" && key != "amount" {
		return false
	}
	var vals []float64
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		if key == "area" {
			vals = append(vals, parseAutoDecimal(g[0].Value))
		} else {
			vals = append(vals, parseAutoMoney(g[0].Value))
		}
	}
	if len(vals) < 2 {
		return true
	}
	minv, maxv := vals[0], vals[0]
	for _, v := range vals[1:] {
		minv = math.Min(minv, v)
		maxv = math.Max(maxv, v)
	}
	if key == "area" {
		return maxv-minv <= 0.03
	}
	return maxv-minv <= 1.0
}

func autoGroupDetail(label string, groups map[string][]autoCandidate) string {
	var parts []string
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		parts = append(parts, g[0].Value+" em "+strings.Join(uniqueAutoSources(g), ", "))
	}
	sort.Strings(parts)
	return label + ": " + strings.Join(parts, " | ")
}

func uniqueAutoSources(cs []autoCandidate) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range cs {
		if !seen[c.Source] {
			seen[c.Source] = true
			out = append(out, c.Source)
		}
	}
	return out
}

func detectFilenameBankMismatch(docs []autoDocument) []AutoFinding {
	var out []AutoFinding
	banks := []string{"Banco do Brasil", "Sicoob", "Cresol", "Sicredi", "Credicom"}
	for _, d := range docs {
		nameLow := strings.ToLower(d.Name)
		fileBank := ""
		for _, b := range banks {
			needle := strings.ToLower(b)
			if b == "Banco do Brasil" {
				if strings.Contains(nameLow, "bb") || strings.Contains(nameLow, "banco do brasil") {
					fileBank = b
				}
			} else if strings.Contains(nameLow, needle) {
				fileBank = b
			}
		}
		if fileBank == "" {
			continue
		}
		for _, c := range d.Fields {
			if c.Key == "bank" && !strings.EqualFold(c.Value, fileBank) {
				out = append(out, AutoFinding{Level: "warning", Title: "Banco no nome do arquivo não confere", Detail: d.Name + " sugere " + fileBank + ", mas o conteúdo menciona " + c.Value + ". Verifique se o modelo correto foi usado."})
			}
		}
	}
	return out
}

func detectAutoBudget(text string) (float64, bool, string) {
	lines := autoLines(text)
	start := -1
	end := len(lines)
	for i, line := range lines {
		low := strings.ToLower(line)
		if start < 0 && strings.Contains(low, "insumos") && strings.Contains(low, "valor") {
			start = i + 1
			continue
		}
		if start >= 0 && (strings.Contains(low, "técnico responsável") || strings.Contains(low, "tecnico responsavel")) {
			end = i
			break
		}
	}
	if start < 0 || start >= end {
		return 0, false, ""
	}
	var total float64
	rows := 0
	for _, line := range lines[start:end] {
		ms := autoMoneyRe.FindAllString(line, -1)
		if len(ms) >= 2 {
			total += parseAutoMoney(ms[len(ms)-1])
			rows++
		}
	}
	if rows < 3 || total <= 0 {
		return 0, false, ""
	}
	return total, true, fmt.Sprintf("%d itens somados", rows)
}

func detectBudgetFindings(docs []autoDocument) []AutoFinding {
	var out []AutoFinding
	for _, d := range docs {
		if !d.BudgetOK {
			continue
		}
		amount := 0.0
		for _, c := range d.Fields {
			if c.Key == "amount" {
				amount = parseAutoMoney(c.Value)
				break
			}
		}
		if amount <= 0 {
			continue
		}
		diff := math.Abs(d.BudgetSum - amount)
		if diff <= 1.0 {
			out = append(out, AutoFinding{Level: "info", Title: "Orçamento fecha com o valor pretendido", Detail: d.Name + ": soma automática " + moneyDisplay(d.BudgetSum) + " e valor pretendido " + moneyDisplay(amount) + "."})
		} else {
			out = append(out, AutoFinding{Level: "error", Title: "Orçamento não fecha", Detail: d.Name + ": soma dos itens " + moneyDisplay(d.BudgetSum) + ", valor pretendido " + moneyDisplay(amount) + ", diferença " + moneyDisplay(diff) + "."})
		}
	}
	return out
}

func autoMissingFields(master []AutoMasterField) []string {
	required := map[string]bool{"producer": true, "cpf": true, "property": true, "municipality": true, "bank": true, "line": true, "activity": true, "area": true, "amount": true}
	var out []string
	for _, m := range master {
		if required[m.Key] && strings.TrimSpace(m.Value) == "" {
			out = append(out, m.Label)
		}
	}
	return out
}

func autoProjectScore(master []AutoMasterField, findings []AutoFinding) int {
	required := map[string]bool{"producer": true, "cpf": true, "property": true, "municipality": true, "bank": true, "line": true, "activity": true, "area": true, "amount": true}
	filled := 0
	for _, m := range master {
		if required[m.Key] && m.Value != "" {
			filled++
		}
	}
	score := int(math.Round(float64(filled) / float64(len(required)) * 80))
	errors := 0
	warnings := 0
	for _, f := range findings {
		if f.Level == "error" {
			errors++
		} else if f.Level == "warning" {
			warnings++
		}
	}
	score += 20
	score -= errors * 12
	score -= warnings * 4
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func autoSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}

func cleanAutoValue(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "|;:_")
	v = strings.TrimSpace(v)
	if len(v) > 160 {
		v = v[:160]
	}
	return v
}

func parseAutoMoney(v string) float64 {
	v = strings.ReplaceAll(v, "R$", "")
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, " ", "")
	if strings.Contains(v, ",") {
		v = strings.ReplaceAll(v, ".", "")
		v = strings.ReplaceAll(v, ",", ".")
	}
	f, _ := strconv.ParseFloat(regexp.MustCompile(`[^0-9.-]`).ReplaceAllString(v, ""), 64)
	return f
}

func parseAutoDecimal(v string) float64 {
	v = regexp.MustCompile(`[^0-9,.-]`).ReplaceAllString(v, "")
	if strings.Contains(v, ",") {
		v = strings.ReplaceAll(v, ".", "")
		v = strings.ReplaceAll(v, ",", ".")
	}
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

func moneyDisplay(v float64) string {
	if v <= 0 {
		return ""
	}
	s := fmt.Sprintf("%.2f", v)
	p := strings.Split(s, ".")
	intp := p[0]
	for i := len(intp) - 3; i > 0; i -= 3 {
		intp = intp[:i] + "." + intp[i:]
	}
	return "R$ " + intp + "," + p[1]
}

func normalizeDecimalDisplay(v string) string {
	f := parseAutoDecimal(v)
	return strings.ReplaceAll(strconv.FormatFloat(f, 'f', -1, 64), ".", ",")
}

func autoKMLArea(text string) float64 {
	re := regexp.MustCompile(`(?is)<coordinates[^>]*>(.*?)</coordinates>`)
	blocks := re.FindAllStringSubmatch(text, -1)
	var total float64
	for _, b := range blocks {
		if len(b) < 2 {
			continue
		}
		var pts [][2]float64
		for _, token := range strings.Fields(strings.TrimSpace(b[1])) {
			parts := strings.Split(token, ",")
			if len(parts) < 2 {
				continue
			}
			lon, e1 := strconv.ParseFloat(parts[0], 64)
			lat, e2 := strconv.ParseFloat(parts[1], 64)
			if e1 == nil && e2 == nil {
				pts = append(pts, [2]float64{lon, lat})
			}
		}
		if len(pts) < 3 {
			continue
		}
		lat0 := 0.0
		for _, p := range pts {
			lat0 += p[1]
		}
		lat0 = lat0 / float64(len(pts)) * math.Pi / 180
		var area float64
		for i := range pts {
			j := (i + 1) % len(pts)
			x1 := pts[i][0] * 111320 * math.Cos(lat0)
			y1 := pts[i][1] * 110540
			x2 := pts[j][0] * 111320 * math.Cos(lat0)
			y2 := pts[j][1] * 110540
			area += x1*y2 - x2*y1
		}
		total += math.Abs(area) / 2 / 10000
	}
	return total
}
