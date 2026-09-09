package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AutoProjectV6View acrescenta ao fluxo V4 a seleção do modelo nativo e a
// Ficha Técnica auditável. A geração continua isolada até o motor nativo estar
// completamente validado, evitando que uma tela nova altere um pacote antigo.
type AutoProjectV6View struct {
	AutoProjectV4View
	NativeModels    []AutoNativeTemplateInfo
	NativeSelected  string
	NativeSuggested string
	Technical       AutoTechnicalReview
}

func (a *App) autoProjectPageV6(w http.ResponseWriter, r *http.Request) {
	view := AutoProjectV6View{NativeModels: autoNativeModels(), NativeSelected: "auto"}
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

func (a *App) autoProjectAnalyzeV6(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida. Atualize a página e tente novamente.", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Error: "Não foi possível ler os arquivos enviados.", Data: AutoProjectV6View{NativeModels: autoNativeModels(), NativeSelected: "auto"}})
		return
	}

	activityHint := strings.TrimSpace(r.FormValue("activity_hint"))
	nativeChoice := strings.TrimSpace(r.FormValue("native_model"))
	if nativeChoice == "" { nativeChoice = "auto" }
	view := AutoProjectView{Analyzed: true, BankHint: strings.TrimSpace(r.FormValue("bank_hint")), LineHint: strings.TrimSpace(r.FormValue("line_hint"))}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		out := AutoProjectV6View{AutoProjectV4View: AutoProjectV4View{AutoProjectCompleteView: AutoProjectCompleteView{AutoProjectView: view}, ActivityHint: activityHint}, NativeModels: autoNativeModels(), NativeSelected: nativeChoice}
		a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Error: "Envie pelo menos um documento ou planilha.", Data: out})
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
				if !view.Files[i].Extracted { view.ReadCount++ }
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
			view.Findings = append(view.Findings, AutoFinding{Level: "info", Title: "OCR concluído", Detail: item.Name + ": o texto do documento escaneado foi recuperado no próprio navegador e incluído na conferência."})
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
		case "error": view.Errors++
		case "warning": view.Warnings++
		default: view.Infos++
		}
	}
	for _, m := range view.Master {
		if m.Status == "ok" || m.Status == "compatible" { view.Confirmed++ }
	}

	complete := enrichAutoProjectView(view, files, docs)
	values := autoMasterMap(view.Master)
	suggested := autoSuggestNativeModel(values, docs, nativeChoice)
	technical := AutoTechnicalReview{}
	if suggested == "agro_irrigacao" {
		technical = autoBuildAgroTechnicalReview(docs, view.Master)
	}

	out := AutoProjectV6View{
		AutoProjectV4View: AutoProjectV4View{AutoProjectCompleteView: complete, ActivityHint: activityHint},
		NativeModels: autoNativeModels(), NativeSelected: nativeChoice, NativeSuggested: suggested, Technical: technical,
	}
	ctx, cancel := contextWithTimeout(15 * time.Second)
	defer cancel()
	if templates, err := a.autoListProjectTemplates(ctx); err == nil {
		out.Templates = templates
	} else {
		out.LibraryError = "Não foi possível carregar a Biblioteca de Modelos agora."
	}
	a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Data: out})
}
