package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AutoProjectV4View struct {
	AutoProjectCompleteView
	Templates    []AutoProjectTemplate
	LibraryError string
	ActivityHint string
}

type autoOCRItem struct {
	Name  string `json:"name"`
	Text  string `json:"text"`
	Pages int    `json:"pages,omitempty"`
}

func parseAutoOCRPayload(raw string) []autoOCRItem {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 6<<20 {
		return nil
	}
	var in []autoOCRItem
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return nil
	}
	out := make([]autoOCRItem, 0, len(in))
	total := 0
	for _, item := range in {
		item.Name = strings.TrimSpace(item.Name)
		item.Text = strings.TrimSpace(item.Text)
		if item.Name == "" || item.Text == "" {
			continue
		}
		if len(item.Text) > 2<<20 {
			item.Text = item.Text[:2<<20]
		}
		total += len(item.Text)
		if total > 5<<20 {
			break
		}
		out = append(out, item)
	}
	return out
}

func (a *App) autoProjectPageV4(w http.ResponseWriter, r *http.Request) {
	view := AutoProjectV4View{}
	ctx, cancel := contextWithTimeout(15 * time.Second)
	defer cancel()
	templates, err := a.autoListProjectTemplates(ctx)
	if err != nil {
		view.LibraryError = "Não foi possível carregar a Biblioteca de Modelos agora."
	} else {
		view.Templates = templates
	}
	a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Data: view})
}

func (a *App) autoProjectAnalyzeV4(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida. Atualize a página e tente novamente.", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Error: "Não foi possível ler os arquivos enviados.", Data: AutoProjectV4View{}})
		return
	}

	activityHint := strings.TrimSpace(r.FormValue("activity_hint"))
	view := AutoProjectView{Analyzed: true, BankHint: strings.TrimSpace(r.FormValue("bank_hint")), LineHint: strings.TrimSpace(r.FormValue("line_hint"))}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Error: "Envie pelo menos um documento ou planilha.", Data: AutoProjectV4View{AutoProjectCompleteView: AutoProjectCompleteView{AutoProjectView: view}, ActivityHint: activityHint}})
		return
	}
	if len(files) > 20 {
		files = files[:20]
		view.Findings = append(view.Findings, AutoFinding{Level: "warning", Title: "Limite de arquivos", Detail: "Foram analisados os primeiros 20 arquivos deste envio."})
	}

	var docs []autoDocument
	for _, fh := range files {
		view.FileCount++
		finfo, doc := analyzeAutoFileV2(fh)
		view.Files = append(view.Files, finfo)
		if doc.Text != "" {
			view.ReadCount++
			doc.Fields = extractAutoFieldsSmart(doc.Text, doc.Name)
			doc.BudgetSum, doc.BudgetOK, doc.BudgetNote = detectAutoBudget(doc.Text)
			if f := autoSupportDocumentFinding(doc.Name, doc.Text); f != nil {
				view.Findings = append(view.Findings, *f)
			}
			docs = append(docs, doc)
		}
	}

	// PDFs escaneados podem chegar com texto recuperado pelo OCR que roda localmente
	// no navegador. O arquivo em si continua sendo enviado normalmente e guardado no pacote.
	for _, item := range parseAutoOCRPayload(r.FormValue("ocr_payload")) {
		source := item.Name + " (OCR)"
		doc := autoDocument{Name: source, Kind: "pdf_ocr", Text: item.Text}
		doc.Fields = extractAutoFieldsSmart(doc.Text, doc.Name)
		doc.BudgetSum, doc.BudgetOK, doc.BudgetNote = detectAutoBudget(doc.Text)
		docs = append(docs, doc)

		matched := false
		for i := range view.Files {
			if strings.EqualFold(strings.TrimSpace(view.Files[i].Name), item.Name) {
				matched = true
				if !view.Files[i].Extracted {
					view.ReadCount++
				}
				view.Files[i].Extracted = true
				if item.Pages > 0 {
					view.Files[i].Note = "Texto recuperado por OCR local em " + strconv.Itoa(item.Pages) + " página(s)"
				} else {
					view.Files[i].Note = "Texto recuperado por OCR local"
				}
				break
			}
		}
		if matched {
			detail := item.Name + ": o texto do documento escaneado foi recuperado no próprio navegador e incluído na conferência."
			view.Findings = append(view.Findings, AutoFinding{Level: "info", Title: "OCR concluído", Detail: detail})
		}
	}

	for _, finfo := range view.Files {
		if finfo.Kind == "pdf" && !finfo.Extracted {
			view.Findings = append(view.Findings, AutoFinding{Level: "warning", Title: "PDF ainda sem texto", Detail: finfo.Name + ": não foi possível obter texto pesquisável nem concluir OCR. O original será preservado e este arquivo deverá ser conferido manualmente."})
		}
	}

	if view.BankHint != "" {
		docs = append(docs, autoDocument{Name: "Informado no formulário", Fields: []autoCandidate{{Key: "bank", Label: "Banco", Value: view.BankHint, Source: "Informado no formulário"}}})
	}
	if view.LineHint != "" {
		docs = append(docs, autoDocument{Name: "Informado no formulário", Fields: []autoCandidate{{Key: "line", Label: "Linha / enquadramento", Value: view.LineHint, Source: "Informado no formulário"}}})
	}
	if activityHint != "" {
		docs = append(docs, autoDocument{Name: "Informado no formulário", Fields: []autoCandidate{{Key: "activity", Label: "Atividade / cultura", Value: activityHint, Source: "Informado no formulário"}}})
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

	complete := enrichAutoProjectView(view, files, docs)
	out := AutoProjectV4View{AutoProjectCompleteView: complete, ActivityHint: activityHint}
	ctx, cancel := contextWithTimeout(15 * time.Second)
	defer cancel()
	if templates, err := a.autoListProjectTemplates(ctx); err == nil {
		out.Templates = templates
	} else {
		out.LibraryError = "Não foi possível carregar a Biblioteca de Modelos agora."
	}
	a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Data: out})
}

func (a *App) autoProjectBuildV4(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida. Atualize a página e tente novamente.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Não foi possível ler os dados confirmados.", http.StatusBadRequest)
		return
	}
	draft, ok := loadAutoDraft(r.FormValue("token"))
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
		if sf.Kind != "xlsx" && sf.Kind != "xlsm" && sf.Kind != "docx" {
			continue
		}
		switch sf.Kind {
		case "xlsx", "xlsm":
			filled, mappings, err := autoFillWorkbook(sf.Data, values)
			if err != nil {
				fillReport = append(fillReport, "Lote atual — "+sf.Name+": falha ao preencher ("+err.Error()+")")
				continue
			}
			if len(mappings) == 0 {
				fillReport = append(fillReport, "Lote atual — "+sf.Name+": nenhuma célula segura foi localizada.")
				continue
			}
			addZipBytes("Modelos_Preenchidos/Lote_Atual/"+safeZipName(sf.Name), filled)
			fillReport = append(fillReport, "Lote atual — "+sf.Name+": "+strings.Join(mappings, "; "))
		case "docx":
			filled, mappings, err := autoFillDOCX(sf.Data, values)
			if err != nil {
				fillReport = append(fillReport, "Lote atual — "+sf.Name+": falha ao preencher DOCX ("+err.Error()+")")
				continue
			}
			if len(mappings) == 0 {
				fillReport = append(fillReport, "Lote atual — "+sf.Name+": nenhum campo seguro foi localizado.")
				continue
			}
			addZipBytes("Modelos_Preenchidos/Lote_Atual/"+safeZipName(sf.Name), filled)
			fillReport = append(fillReport, "Lote atual — "+sf.Name+": "+strings.Join(mappings, "; "))
		}
	}

	ctx, cancel := contextWithTimeout(60 * time.Second)
	libraryFiles, libraryNotes := a.autoLoadMatchingTemplates(ctx, values)
	cancel()
	fillReport = append(fillReport, libraryNotes...)
	for _, sf := range libraryFiles {
		switch sf.Kind {
		case "xlsx", "xlsm":
			filled, mappings, err := autoFillWorkbook(sf.Data, values)
			if err != nil {
				fillReport = append(fillReport, "Biblioteca — "+sf.Name+": falha ao preencher ("+err.Error()+")")
				continue
			}
			if len(mappings) == 0 {
				fillReport = append(fillReport, "Biblioteca — "+sf.Name+": nenhuma célula segura foi localizada; o original não foi alterado.")
				continue
			}
			addZipBytes("Modelos_Preenchidos/Biblioteca/"+safeZipName(sf.Name), filled)
			fillReport = append(fillReport, "Biblioteca — "+sf.Name+": "+strings.Join(mappings, "; "))
		case "docx":
			filled, mappings, err := autoFillDOCX(sf.Data, values)
			if err != nil {
				fillReport = append(fillReport, "Biblioteca — "+sf.Name+": falha ao preencher DOCX ("+err.Error()+")")
				continue
			}
			if len(mappings) == 0 {
				fillReport = append(fillReport, "Biblioteca — "+sf.Name+": nenhum campo seguro foi localizado; o original não foi alterado.")
				continue
			}
			addZipBytes("Modelos_Preenchidos/Biblioteca/"+safeZipName(sf.Name), filled)
			fillReport = append(fillReport, "Biblioteca — "+sf.Name+": "+strings.Join(mappings, "; "))
		}
	}
	if len(fillReport) == 0 {
		fillReport = append(fillReport, "Nenhum modelo de planilha/documento foi encontrado neste lote ou na Biblioteca.")
	}
	addZipText("08_Relatorio_Preenchimento_Modelos.txt", strings.Join(fillReport, "\n"))

	for _, sf := range draft.Files {
		addZipBytes("Documentos_Originais/"+safeZipName(sf.Name), sf.Data)
	}
	addZipText("LEIA-ME.txt", autoPackageReadmeV4(values, draft, fillReport, len(libraryFiles)))
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

func autoPackageReadmeV4(v map[string]string, d *autoDraft, fillReport []string, libraryCount int) string {
	var b strings.Builder
	b.WriteString("VIA VERDE AUTOPROJETO — PACOTE GERADO AUTOMATICAMENTE\n\n")
	b.WriteString("Proponente: " + valueOrPending(v["producer"]) + "\n")
	b.WriteString("Banco/Linha: " + valueOrPending(v["bank"]) + " / " + valueOrPending(v["line"]) + "\n")
	b.WriteString("Atividade: " + valueOrPending(v["activity"]) + "\n")
	b.WriteString("Área: " + valueOrPending(v["area"]) + "\n")
	b.WriteString("Valor: " + valueOrPending(v["amount"]) + "\n")
	b.WriteString("Modelos compatíveis carregados da Biblioteca: " + fmtInt(libraryCount) + "\n\n")
	b.WriteString("O pacote contém ficha mestre, projeto técnico, proposta, laudo, relatório de divergências, checklist, cálculos, modelos preenchidos e cópias dos documentos originais.\n\n")
	b.WriteString("LEITURA DE PDF ESCANEADO\n")
	b.WriteString("Quando o PDF não possui texto pesquisável, o AutoProjeto tenta OCR local no navegador antes do envio final da análise. O texto recuperado participa dos mesmos cruzamentos e da ficha mestre. Se o OCR não puder ser concluído, o sistema mantém o original e sinaliza a conferência manual.\n\n")
	b.WriteString("PREENCHIMENTO DE MODELOS\n")
	for _, s := range fillReport {
		b.WriteString("- " + s + "\n")
	}
	b.WriteString("\nIMPORTANTE\n")
	b.WriteString("O AutoProjeto só usa dados encontrados ou confirmados. Campos sem fonte confiável ficam como A CONFIRMAR; dados duvidosos não são inventados.\n")
	return b.String()
}

func fmtInt(v int) string {
	return strconv.Itoa(v)
}
