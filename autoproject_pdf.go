package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	pdfreader "github.com/ledongthuc/pdf"
)

const autoPDFMaxBytes int64 = 40 << 20

// autoPDFText extrai a camada textual de PDFs digitais. PDFs compostos apenas
// por imagens continuam sendo recebidos, mas são marcados para conferência em
// vez de alimentar a ficha mestre com informação inventada.
func autoPDFText(data []byte) (text string, err error) {
	defer func() {
		if v := recover(); v != nil {
			text = ""
			err = fmt.Errorf("falha segura ao interpretar PDF: %v", v)
		}
	}()

	tmp, err := os.CreateTemp("", "viaverde-autoprojeto-*.pdf")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	defer os.Remove(name)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	f, r, err := pdfreader.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	plain, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if _, err := io.Copy(&out, io.LimitReader(plain, 8<<20)); err != nil {
		return "", err
	}
	text = cleanupAutoText(out.String())
	if len(strings.TrimSpace(text)) < 20 {
		return "", fmt.Errorf("PDF sem camada de texto pesquisável")
	}
	return text, nil
}

func analyzeAutoFileV2(fh *multipart.FileHeader) (AutoProjectFile, autoDocument) {
	kind := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	if kind != "pdf" {
		return analyzeAutoFile(fh)
	}

	info := AutoProjectFile{Name: filepath.Base(fh.Filename), Kind: kind, SizeLabel: autoSize(fh.Size)}
	doc := autoDocument{Name: info.Name, Kind: kind}
	if fh.Size > autoPDFMaxBytes {
		info.Note = "PDF maior que 40 MB; não foi processado nesta análise."
		return info, doc
	}
	f, err := fh.Open()
	if err != nil {
		info.Note = "Não foi possível abrir o PDF."
		return info, doc
	}
	defer f.Close()

	b, err := io.ReadAll(io.LimitReader(f, autoPDFMaxBytes))
	if err != nil {
		info.Note = "Falha durante a leitura do PDF."
		return info, doc
	}
	text, err := autoPDFText(b)
	if err != nil || strings.TrimSpace(text) == "" {
		info.Note = "PDF recebido, mas sem texto pesquisável. Pode ser documento escaneado; os dados dele não foram usados automaticamente."
		return info, doc
	}

	info.Extracted = true
	info.Note = "PDF lido, interpretado e comparado."
	doc.Text = text
	return info, doc
}

func (a *App) autoProjectPageV3(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Data: AutoProjectCompleteView{}})
}

func (a *App) autoProjectAnalyzeV2(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida. Atualize a página e tente novamente.", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Error: "Não foi possível ler os arquivos enviados.", Data: AutoProjectCompleteView{}})
		return
	}

	view := AutoProjectView{Analyzed: true, BankHint: strings.TrimSpace(r.FormValue("bank_hint")), LineHint: strings.TrimSpace(r.FormValue("line_hint"))}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Error: "Envie pelo menos um documento ou planilha.", Data: AutoProjectCompleteView{AutoProjectView: view}})
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
		if finfo.Kind == "pdf" && !finfo.Extracted {
			view.Findings = append(view.Findings, AutoFinding{Level: "warning", Title: "PDF sem texto pesquisável", Detail: finfo.Name + ": o arquivo parece escaneado ou não possui camada textual. Ele será preservado no pacote, mas não define dados da ficha mestre."})
		}
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

	complete := enrichAutoProjectView(view, files, docs)
	a.render(w, r, "autoproject", ViewData{Title: "AutoProjeto inteligente", Data: complete})
}
