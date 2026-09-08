package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

type AutoProjectCompleteView struct {
	AutoProjectView
	Token             string
	BuildReady        bool
	ScannedPDFs       int
	SpreadsheetModels int
}

type autoStoredFile struct {
	Name string
	Kind string
	Data []byte
}

type autoDraft struct {
	Created  time.Time
	Files    []autoStoredFile
	Master   []AutoMasterField
	Findings []AutoFinding
	Docs     []autoDocument
}

var autoDraftCache sync.Map

func newAutoDraftToken() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func cloneAutoMaster(in []AutoMasterField) []AutoMasterField {
	out := make([]AutoMasterField, len(in))
	copy(out, in)
	for i := range out {
		out[i].Sources = append([]string(nil), out[i].Sources...)
	}
	return out
}

func storeAutoDraft(files []*multipart.FileHeader, master []AutoMasterField, findings []AutoFinding, docs []autoDocument) string {
	token := newAutoDraftToken()
	d := &autoDraft{Created: time.Now(), Master: cloneAutoMaster(master), Findings: append([]AutoFinding(nil), findings...), Docs: append([]autoDocument(nil), docs...)}
	var total int64
	const totalMax int64 = 120 << 20
	for _, fh := range files {
		if fh == nil || fh.Size <= 0 || fh.Size > autoPDFMaxBytes || total+fh.Size > totalMax {
			continue
		}
		f, err := fh.Open()
		if err != nil {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(f, fh.Size+1))
		f.Close()
		if err != nil || int64(len(b)) > fh.Size+1 {
			continue
		}
		total += int64(len(b))
		d.Files = append(d.Files, autoStoredFile{Name: filepath.Base(fh.Filename), Kind: strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), "."), Data: b})
	}
	autoDraftCache.Store(token, d)
	cleanupAutoDrafts()
	return token
}

func cleanupAutoDrafts() {
	cut := time.Now().Add(-45 * time.Minute)
	autoDraftCache.Range(func(k, v any) bool {
		if d, ok := v.(*autoDraft); !ok || d.Created.Before(cut) {
			autoDraftCache.Delete(k)
		}
		return true
	})
}

func loadAutoDraft(token string) (*autoDraft, bool) {
	v, ok := autoDraftCache.Load(strings.TrimSpace(token))
	if !ok {
		return nil, false
	}
	d, ok := v.(*autoDraft)
	if !ok || time.Since(d.Created) > 45*time.Minute {
		autoDraftCache.Delete(token)
		return nil, false
	}
	return d, true
}

func enrichAutoProjectView(view AutoProjectView, files []*multipart.FileHeader, docs []autoDocument) AutoProjectCompleteView {
	cv := AutoProjectCompleteView{AutoProjectView: view}
	for _, f := range view.Files {
		if f.Kind == "pdf" && !f.Extracted {
			cv.ScannedPDFs++
		}
	}
	for _, fh := range files {
		kind := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
		if kind == "xlsx" || kind == "xlsm" {
			cv.SpreadsheetModels++
		}
	}
	cv.Token = storeAutoDraft(files, view.Master, view.Findings, docs)
	cv.BuildReady = cv.Token != ""
	return cv
}

func (a *App) autoProjectBuild(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida. Atualize a página e tente novamente.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Não foi possível ler os dados confirmados.", http.StatusBadRequest)
		return
	}
	token := r.FormValue("token")
	draft, ok := loadAutoDraft(token)
	if !ok {
		http.Error(w, "Esta análise expirou. Envie os arquivos novamente para gerar o pacote.", http.StatusGone)
		return
	}

	master := cloneAutoMaster(draft.Master)
	for i := range master {
		if v := strings.TrimSpace(r.FormValue("field_" + master[i].Key)); v != "" {
			master[i].Value = v
			master[i].Status = "ok"
			master[i].Sources = []string{"Confirmado no AutoProjeto"}
		}
	}
	values := autoMasterMap(master)
	filename := "Pacote_AutoProjeto_" + safeAutoFilename(values["producer"]) + ".zip"
	if values["producer"] == "" {
		filename = "Pacote_AutoProjeto.zip"
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	addZipText := func(name, text string) {
		f, err := zw.Create(name)
		if err == nil {
			_, _ = io.WriteString(f, text)
		}
	}
	addZipBytes := func(name string, data []byte) {
		f, err := zw.Create(name)
		if err == nil {
			_, _ = f.Write(data)
		}
	}

	addZipBytes("01_Ficha_Mestre.csv", autoMasterCSV(master))
	addZipBytes("02_Projeto_Tecnico.docx", makeAutoProjectDOCX(values, draft.Findings))
	addZipBytes("03_Proposta.docx", makeAutoProposalDOCX(values))
	addZipBytes("04_Laudo_de_Vistoria.docx", makeAutoInspectionDOCX(values, draft.Findings))
	addZipBytes("05_Relatorio_de_Divergencias.docx", makeAutoFindingsDOCX(values, draft.Findings))
	addZipText("06_Checklist.txt", autoChecklistText(values, draft))
	addZipText("07_Resumo_de_Calculos.txt", autoCalculationSummary(values, draft.Docs))

	var fillReport []string
	for _, sf := range draft.Files {
		if sf.Kind != "xlsx" && sf.Kind != "xlsm" {
			continue
		}
		filled, mappings, err := autoFillWorkbook(sf.Data, values)
		if err != nil {
			fillReport = append(fillReport, sf.Name+": não foi possível preencher automaticamente ("+err.Error()+")")
			continue
		}
		if len(mappings) == 0 {
			fillReport = append(fillReport, sf.Name+": nenhuma célula segura foi identificada para preenchimento automático.")
			continue
		}
		addZipBytes("Modelos_Preenchidos/"+safeZipName(sf.Name), filled)
		fillReport = append(fillReport, sf.Name+": "+strings.Join(mappings, "; "))
	}
	if len(fillReport) == 0 {
		fillReport = append(fillReport, "Nenhuma planilha XLSX/XLSM foi enviada neste lote.")
	}
	addZipText("08_Relatorio_Preenchimento_Planilhas.txt", strings.Join(fillReport, "\n"))

	for _, sf := range draft.Files {
		if sf.Kind == "pdf" || sf.Kind == "kml" {
			addZipBytes("Documentos_Originais/"+safeZipName(sf.Name), sf.Data)
		}
	}
	addZipText("LEIA-ME.txt", autoPackageReadme(values, draft, fillReport))
	if err := zw.Close(); err != nil {
		http.Error(w, "Falha ao finalizar o pacote.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func autoMasterMap(master []AutoMasterField) map[string]string {
	out := map[string]string{}
	for _, m := range master {
		out[m.Key] = strings.TrimSpace(m.Value)
	}
	return out
}

func safeAutoFilename(v string) string {
	v = autoFold(strings.TrimSpace(v))
	v = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(v, "_")
	v = strings.Trim(v, "_")
	if len(v) > 55 {
		v = v[:55]
	}
	if v == "" {
		return "projeto"
	}
	return v
}

func safeZipName(v string) string {
	v = filepath.Base(v)
	v = strings.ReplaceAll(v, "\\", "_")
	v = strings.ReplaceAll(v, "/", "_")
	return v
}

func autoMasterCSV(master []AutoMasterField) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	_ = w.Write([]string{"Campo", "Valor", "Status", "Fonte"})
	for _, m := range master {
		_ = w.Write([]string{m.Label, m.Value, m.Status, strings.Join(m.Sources, " | ")})
	}
	w.Flush()
	return buf.Bytes()
}

func autoFieldKeyFromLabel(label string) string {
	v := autoFold(strings.TrimSpace(label))
	v = regexp.MustCompile(`\s+`).ReplaceAllString(v, " ")
	patterns := []struct {
		key string
		ss  []string
	}{
		{"producer", []string{"nome do proponente", "nome proponente", "nome do produtor", "produtor", "proponente"}},
		{"cpf", []string{"cpf/cnpj", "cpf / cnpj", "cpf", "cnpj"}},
		{"property", []string{"nome da propriedade", "imovel beneficiado", "propriedade", "imovel rural"}},
		{"registry", []string{"matricula", "matricula do imovel"}},
		{"municipality", []string{"municipio", "cidade"}},
		{"possession", []string{"condicao de posse", "posse"}},
		{"bank", []string{"banco", "instituicao financeira"}},
		{"agency", []string{"agencia", "nº agencia", "n. agencia"}},
		{"line", []string{"linha de credito", "linha / enquadramento", "enquadramento", "programa"}},
		{"activity", []string{"atividade / cultura", "atividade", "cultura", "exploracao"}},
		{"area", []string{"area a financiar", "area à financiar", "area financiada", "area beneficiada", "area total"}},
		{"amount", []string{"valor pretendido", "valor financiado", "financiamento pretendido", "valor do financiamento"}},
		{"technician", []string{"tecnico responsavel", "nome do tecnico", "responsavel tecnico"}},
		{"crea", []string{"crea", "registro crea"}},
	}
	for _, p := range patterns {
		for _, s := range p.ss {
			if v == s || strings.HasPrefix(v, s+":") {
				return p.key
			}
		}
	}
	return ""
}

func autoFillWorkbook(data []byte, values map[string]string) ([]byte, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var mapped []string
	seen := map[string]bool{}
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for r, row := range rows {
			for c, cell := range row {
				key := autoFieldKeyFromLabel(cell)
				value := strings.TrimSpace(values[key])
				if key == "" || value == "" {
					continue
				}
				candidates := [][2]int{{c + 2, r + 1}, {c + 3, r + 1}, {c + 1, r + 2}}
				for _, rc := range candidates {
					if rc[0] < 1 || rc[0] > 16384 || rc[1] < 1 || rc[1] > 1048576 {
						continue
					}
					axis, _ := excelize.CoordinatesToCellName(rc[0], rc[1])
					formula, _ := f.GetCellFormula(sheet, axis)
					if strings.TrimSpace(formula) != "" {
						continue
					}
					existing, _ := f.GetCellValue(sheet, axis)
					if strings.TrimSpace(existing) != "" {
						continue
					}
					if key == "amount" {
						if n := parseAutoMoney(value); n > 0 {
							_ = f.SetCellValue(sheet, axis, n)
						} else {
							_ = f.SetCellValue(sheet, axis, value)
						}
					} else if key == "area" {
						if n := parseAutoDecimal(value); n > 0 {
							_ = f.SetCellValue(sheet, axis, n)
						} else {
							_ = f.SetCellValue(sheet, axis, value)
						}
					} else {
						_ = f.SetCellValue(sheet, axis, value)
					}
					mapKey := sheet + "!" + axis + "=" + key
					if !seen[mapKey] {
						seen[mapKey] = true
						mapped = append(mapped, mapKey)
					}
					break
				}
			}
		}
	}
	if len(mapped) == 0 {
		return data, mapped, nil
	}
	out, err := f.WriteToBuffer()
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(mapped)
	return out.Bytes(), mapped, nil
}

type autoDocLine struct {
	Label string
	Value string
}

func makeAutoProjectDOCX(v map[string]string, findings []AutoFinding) []byte {
	lines := []autoDocLine{
		{"Proponente", valueOrPending(v["producer"])}, {"CPF/CNPJ", valueOrPending(v["cpf"])},
		{"Imóvel", valueOrPending(v["property"])}, {"Matrícula", valueOrPending(v["registry"])}, {"Município", valueOrPending(v["municipality"])},
		{"Banco", valueOrPending(v["bank"])}, {"Linha / enquadramento", valueOrPending(v["line"])}, {"Atividade", valueOrPending(v["activity"])},
		{"Área financiada", valueOrPending(v["area"])}, {"Valor pretendido", valueOrPending(v["amount"])}, {"Técnico responsável", valueOrPending(v["technician"])}, {"CREA", valueOrPending(v["crea"])},
	}
	body := []string{"DOCUMENTO GERADO AUTOMATICAMENTE PELO VIA VERDE AUTOPROJETO", "", "1. IDENTIFICAÇÃO DO PROPONENTE"}
	body = append(body, linesToParagraphs(lines[:2])...)
	body = append(body, "", "2. IDENTIFICAÇÃO DO IMÓVEL")
	body = append(body, linesToParagraphs(lines[2:5])...)
	body = append(body, "", "3. OPERAÇÃO DE CRÉDITO")
	body = append(body, linesToParagraphs(lines[5:10])...)
	body = append(body, "", "4. RESPONSABILIDADE TÉCNICA")
	body = append(body, linesToParagraphs(lines[10:])...)
	body = append(body, "", "5. CONFERÊNCIA AUTOMÁTICA")
	if len(findings) == 0 {
		body = append(body, "Nenhuma divergência foi registrada nos documentos lidos.")
	} else {
		for _, f := range findings {
			body = append(body, strings.ToUpper(f.Level)+" — "+f.Title+": "+f.Detail)
		}
	}
	body = append(body, "", "Observação: campos marcados como A CONFIRMAR não foram encontrados com segurança nos documentos enviados e não foram inventados pelo sistema.")
	return makeSimpleDOCX("PROJETO TÉCNICO RURAL — AUTOPROJETO", body)
}

func makeAutoProposalDOCX(v map[string]string) []byte {
	body := []string{
		"Proponente: " + valueOrPending(v["producer"]),
		"CPF/CNPJ: " + valueOrPending(v["cpf"]),
		"Imóvel: " + valueOrPending(v["property"]),
		"Município: " + valueOrPending(v["municipality"]),
		"Banco: " + valueOrPending(v["bank"]),
		"Linha: " + valueOrPending(v["line"]),
		"Finalidade / atividade: " + valueOrPending(v["activity"]),
		"Área financiada: " + valueOrPending(v["area"]),
		"Valor pretendido: " + valueOrPending(v["amount"]),
		"Garantia / condição de posse: " + valueOrPending(v["possession"]),
		"",
		"Esta proposta foi montada automaticamente a partir da ficha mestre consolidada pelo AutoProjeto. Os dados devem ser conferidos antes da assinatura e do protocolo bancário.",
	}
	return makeSimpleDOCX("PROPOSTA DE FINANCIAMENTO RURAL", body)
}

func makeAutoInspectionDOCX(v map[string]string, findings []AutoFinding) []byte {
	body := []string{
		"Proponente: " + valueOrPending(v["producer"]),
		"CPF/CNPJ: " + valueOrPending(v["cpf"]),
		"Imóvel vistoriado: " + valueOrPending(v["property"]),
		"Matrícula: " + valueOrPending(v["registry"]),
		"Município: " + valueOrPending(v["municipality"]),
		"Atividade / cultura: " + valueOrPending(v["activity"]),
		"Área beneficiada: " + valueOrPending(v["area"]),
		"Valor da operação: " + valueOrPending(v["amount"]),
		"Técnico responsável: " + valueOrPending(v["technician"]),
		"CREA: " + valueOrPending(v["crea"]),
		"",
		"SITUAÇÃO AUTOMÁTICA DA DOCUMENTAÇÃO",
	}
	for _, f := range findings {
		if f.Level == "error" || f.Level == "warning" {
			body = append(body, strings.ToUpper(f.Level)+" — "+f.Title+": "+f.Detail)
		}
	}
	body = append(body, "", "As condições de campo, estágio da cultura, estado fitossanitário e demais constatações que não constarem dos documentos enviados permanecem como A CONFIRMAR e exigem validação do responsável técnico.")
	return makeSimpleDOCX("LAUDO DE VISTORIA — RASCUNHO AUTOMÁTICO", body)
}

func makeAutoFindingsDOCX(v map[string]string, findings []AutoFinding) []byte {
	body := []string{"Proponente: " + valueOrPending(v["producer"]), "Imóvel: " + valueOrPending(v["property"]), ""}
	if len(findings) == 0 {
		body = append(body, "Nenhuma divergência detectada entre os documentos lidos.")
	} else {
		for i, f := range findings {
			body = append(body, fmt.Sprintf("%d. [%s] %s", i+1, strings.ToUpper(f.Level), f.Title), f.Detail, "")
		}
	}
	return makeSimpleDOCX("RELATÓRIO DE DIVERGÊNCIAS E ATENÇÕES", body)
}

func valueOrPending(v string) string {
	if strings.TrimSpace(v) == "" {
		return "A CONFIRMAR"
	}
	return strings.TrimSpace(v)
}

func linesToParagraphs(lines []autoDocLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.Label+": "+l.Value)
	}
	return out
}

func makeSimpleDOCX(title string, paragraphs []string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name, s string) {
		f, err := zw.Create(name)
		if err == nil {
			_, _ = io.WriteString(f, s)
		}
	}
	write("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`)
	write("_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`)
	var body strings.Builder
	body.WriteString(`<w:p><w:r><w:rPr><w:b/><w:sz w:val="30"/></w:rPr><w:t>` + xmlAutoEscape(title) + `</w:t></w:r></w:p>`)
	for _, p := range paragraphs {
		if p == "" {
			body.WriteString(`<w:p/>`)
			continue
		}
		body.WriteString(`<w:p><w:r><w:t xml:space="preserve">` + xmlAutoEscape(p) + `</w:t></w:r></w:p>`)
	}
	doc := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` + body.String() + `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr></w:body></w:document>`
	write("word/document.xml", doc)
	_ = zw.Close()
	return buf.Bytes()
}

func xmlAutoEscape(v string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(v)
}

func autoChecklistText(v map[string]string, d *autoDraft) string {
	roles := map[autoDocRole]int{}
	hasKML := false
	scanned := 0
	for _, sf := range d.Files {
		if sf.Kind == "kml" {
			hasKML = true
		}
		if sf.Kind == "pdf" {
			text, err := autoPDFText(sf.Data)
			if err != nil || strings.TrimSpace(text) == "" {
				scanned++
				continue
			}
			roles[classifyAutoDoc(sf.Name, text)]++
		}
	}
	var b strings.Builder
	b.WriteString("CHECKLIST AUTOMÁTICO — VIA VERDE\n\n")
	checks := []struct {
		label string
		ok    bool
	}{
		{"Documento principal (projeto/proposta/laudo/cadastro)", roles[autoRolePrimary] > 0},
		{"Documento fundiário (CCIR/ITR/matrícula/certidão)", roles[autoRoleProperty] > 0},
		{"Orçamento/cotação", roles[autoRoleQuote] > 0},
		{"KML/croqui geográfico", hasKML},
	}
	activity := autoFold(v["activity"])
	if strings.Contains(activity, "pecu") || strings.Contains(activity, "bovin") || strings.Contains(activity, "leite") || strings.Contains(activity, "corte") {
		checks = append(checks, struct {
			label string
			ok    bool
		}{"Ficha sanitária/pecuária", roles[autoRoleSanitary] > 0})
	}
	if strings.Contains(activity, "irriga") {
		checks = append(checks, struct {
			label string
			ok    bool
		}{"Outorga/autorização hídrica", roles[autoRolePermit] > 0})
	}
	for _, c := range checks {
		status := "[FALTA]"
		if c.ok {
			status = "[OK]"
		}
		b.WriteString(status + " " + c.label + "\n")
	}
	for _, spec := range autoFieldOrder {
		if strings.TrimSpace(v[spec.Key]) == "" {
			b.WriteString("[FALTA] Campo mestre: " + spec.Label + "\n")
		}
	}
	if scanned > 0 {
		b.WriteString(fmt.Sprintf("[ATENÇÃO] %d PDF(s) sem camada de texto pesquisável; foram preservados no pacote, mas seus dados não foram usados automaticamente.\n", scanned))
	}
	return b.String()
}

func autoCalculationSummary(v map[string]string, docs []autoDocument) string {
	area := parseAutoDecimal(v["area"])
	amount := parseAutoMoney(v["amount"])
	var b strings.Builder
	b.WriteString("RESUMO AUTOMÁTICO DE CÁLCULOS\n\n")
	b.WriteString("Área informada: " + valueOrPending(v["area"]) + "\n")
	b.WriteString("Valor pretendido: " + valueOrPending(v["amount"]) + "\n")
	if area > 0 && amount > 0 {
		b.WriteString("Valor financiado por hectare: " + moneyDisplay(amount/area) + "/ha\n")
	}
	for _, d := range docs {
		if d.BudgetOK {
			b.WriteString("Orçamento detectado em " + d.Name + ": " + moneyDisplay(d.BudgetSum) + " (" + d.BudgetNote + ")\n")
			if amount > 0 {
				diff := d.BudgetSum - amount
				b.WriteString("Diferença para o valor pretendido: " + moneyDisplayAbs(diff) + "\n")
			}
		}
	}
	return b.String()
}

func moneyDisplayAbs(v float64) string {
	if v < 0 {
		return "-" + moneyDisplay(-v)
	}
	return moneyDisplay(v)
}

func autoPackageReadme(v map[string]string, d *autoDraft, fillReport []string) string {
	var b strings.Builder
	b.WriteString("VIA VERDE AUTOPROJETO — PACOTE GERADO AUTOMATICAMENTE\n\n")
	b.WriteString("Proponente: " + valueOrPending(v["producer"]) + "\n")
	b.WriteString("Banco/Linha: " + valueOrPending(v["bank"]) + " / " + valueOrPending(v["line"]) + "\n")
	b.WriteString("Atividade: " + valueOrPending(v["activity"]) + "\n")
	b.WriteString("Área: " + valueOrPending(v["area"]) + "\n")
	b.WriteString("Valor: " + valueOrPending(v["amount"]) + "\n\n")
	b.WriteString("O pacote contém ficha mestre, projeto técnico, proposta, laudo, relatório de divergências, checklist, cálculos, modelos XLSX/XLSM preenchidos quando foi possível localizar células seguras e cópias dos PDFs/KML originais.\n\n")
	b.WriteString("PLANILHAS\n")
	for _, s := range fillReport {
		b.WriteString("- " + s + "\n")
	}
	b.WriteString("\nIMPORTANTE\n")
	b.WriteString("O AutoProjeto não inventa dados ausentes. Campos sem fonte confiável ficam como A CONFIRMAR. PDFs escaneados sem camada de texto são preservados, mas exigem leitura visual/OCR externo antes de seus dados entrarem na ficha mestre.\n")
	return b.String()
}
