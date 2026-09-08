package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type AutoProjectTemplate struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Bank        string `json:"bank"`
	Line        string `json:"line"`
	Activity    string `json:"activity"`
	StoragePath string `json:"storage_path"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Active      bool   `json:"active"`
	CreatedBy   string `json:"created_by,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

func (a *App) autoListProjectTemplates(ctx context.Context) ([]AutoProjectTemplate, error) {
	if a.sb == nil {
		return nil, fmt.Errorf("Supabase não configurado")
	}
	var out []AutoProjectTemplate
	q := "select=id,name,bank,line,activity,storage_path,content_type,size_bytes,active,created_by,created_at&active=eq.true&order=created_at.desc"
	if err := a.sb.Select(ctx, "auto_project_templates", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *App) autoProjectTemplateUpload(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida. Atualize a página e tente novamente.", http.StatusForbidden)
		return
	}
	if a.sb == nil {
		http.Error(w, "Banco de dados não configurado.", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseMultipartForm(36 << 20); err != nil {
		http.Error(w, "Não foi possível receber o modelo.", http.StatusBadRequest)
		return
	}
	file, fh, err := r.FormFile("template")
	if err != nil {
		http.Error(w, "Selecione uma planilha ou documento modelo.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".xlsx" && ext != ".xlsm" && ext != ".docx" {
		http.Error(w, "A biblioteca aceita XLSX, XLSM e DOCX.", http.StatusBadRequest)
		return
	}
	if fh.Size <= 0 || fh.Size > 30<<20 {
		http.Error(w, "O modelo deve ter até 30 MB.", http.StatusBadRequest)
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, 30<<20+1))
	if err != nil || len(data) == 0 || int64(len(data)) > 30<<20 {
		http.Error(w, "Não foi possível ler o modelo.", http.StatusBadRequest)
		return
	}

	bank := strings.TrimSpace(r.FormValue("template_bank"))
	line := strings.TrimSpace(r.FormValue("template_line"))
	activity := strings.TrimSpace(r.FormValue("template_activity"))
	if bank == "" || line == "" {
		http.Error(w, "Informe banco e linha para o sistema saber quando usar este modelo.", http.StatusBadRequest)
		return
	}

	objectPath := "autoproject/templates/" + newAutoDraftToken() + "-" + safeZipName(fh.Filename)
	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		switch ext {
		case ".xlsx":
			contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case ".xlsm":
			contentType = "application/vnd.ms-excel.sheet.macroEnabled.12"
		case ".docx":
			contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		}
	}

	ctx, cancel := contextWithTimeout(40 * time.Second)
	defer cancel()
	if err := a.sb.Upload(ctx, a.cfg.StorageBucket, objectPath, contentType, bytes.NewReader(data)); err != nil {
		http.Error(w, "Falha ao guardar o modelo no armazenamento.", http.StatusBadGateway)
		return
	}

	createdBy := ""
	if u := userFromContext(r.Context()); u != nil {
		createdBy = u.ID
	}
	payload := map[string]any{
		"name":         filepath.Base(fh.Filename),
		"bank":         bank,
		"line":         line,
		"activity":     activity,
		"storage_path": objectPath,
		"content_type": contentType,
		"size_bytes":   len(data),
		"active":       true,
	}
	if createdBy != "" {
		payload["created_by"] = createdBy
	}
	if err := a.sb.Insert(ctx, "auto_project_templates", payload, nil); err != nil {
		_ = a.sb.DeleteObject(ctx, a.cfg.StorageBucket, objectPath)
		http.Error(w, "O arquivo foi recebido, mas não foi possível cadastrar o modelo.", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/autoproject?ok="+url.QueryEscape("Modelo salvo na Biblioteca do AutoProjeto."), http.StatusSeeOther)
}

func (a *App) autoProjectTemplateDelete(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", http.StatusForbidden)
		return
	}
	if a.sb == nil {
		http.Error(w, "Banco de dados não configurado.", http.StatusServiceUnavailable)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "Modelo inválido.", http.StatusBadRequest)
		return
	}
	ctx, cancel := contextWithTimeout(20 * time.Second)
	defer cancel()
	var rows []AutoProjectTemplate
	q := "select=id,storage_path&" + eq("id", id) + "&limit=1"
	if err := a.sb.Select(ctx, "auto_project_templates", q, &rows); err != nil || len(rows) == 0 {
		http.Error(w, "Modelo não encontrado.", http.StatusNotFound)
		return
	}
	_ = a.sb.DeleteObject(ctx, a.cfg.StorageBucket, rows[0].StoragePath)
	if err := a.sb.Update(ctx, "auto_project_templates", eq("id", id), map[string]any{"active": false}, nil); err != nil {
		http.Error(w, "Não foi possível desativar o modelo.", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/autoproject?ok="+url.QueryEscape("Modelo removido da biblioteca."), http.StatusSeeOther)
}

func autoTemplateMatches(t AutoProjectTemplate, values map[string]string) bool {
	match := func(rule, got string) bool {
		rule = autoFold(strings.TrimSpace(rule))
		got = autoFold(strings.TrimSpace(got))
		if rule == "" {
			return true
		}
		if got == "" {
			return false
		}
		return rule == got || strings.Contains(rule, got) || strings.Contains(got, rule)
	}
	return match(t.Bank, values["bank"]) && match(t.Line, values["line"]) && match(t.Activity, values["activity"])
}

func (a *App) autoLoadMatchingTemplates(ctx context.Context, values map[string]string) ([]autoStoredFile, []string) {
	templates, err := a.autoListProjectTemplates(ctx)
	if err != nil {
		return nil, []string{"Biblioteca: não foi possível consultar os modelos salvos: " + err.Error()}
	}
	var files []autoStoredFile
	var notes []string
	for _, t := range templates {
		if !autoTemplateMatches(t, values) {
			continue
		}
		b, _, err := a.sb.Download(ctx, a.cfg.StorageBucket, t.StoragePath)
		if err != nil {
			notes = append(notes, "Biblioteca: falha ao baixar "+t.Name+".")
			continue
		}
		kind := strings.TrimPrefix(strings.ToLower(filepath.Ext(t.Name)), ".")
		files = append(files, autoStoredFile{Name: t.Name, Kind: kind, Data: b})
		notes = append(notes, "Biblioteca: modelo compatível carregado — "+t.Name+" ["+t.Bank+" / "+t.Line+" / "+valueOrGlobal(t.Activity)+"]")
	}
	if len(files) == 0 && strings.TrimSpace(values["bank"]) != "" && strings.TrimSpace(values["line"]) != "" {
		notes = append(notes, "Biblioteca: nenhum modelo salvo corresponde a "+values["bank"]+" / "+values["line"]+" / "+valueOrGlobal(values["activity"])+".")
	}
	return files, notes
}

func valueOrGlobal(v string) string {
	if strings.TrimSpace(v) == "" {
		return "geral"
	}
	return strings.TrimSpace(v)
}

func autoFillDOCX(data []byte, values map[string]string) ([]byte, []string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, err
	}
	files := map[string][]byte{}
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			return nil, nil, err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, err
		}
		files[zf.Name] = b
	}
	xmlData, ok := files["word/document.xml"]
	if !ok {
		return nil, nil, fmt.Errorf("document.xml ausente")
	}

	updated, mappings := autoFillDOCXPlaceholders(string(xmlData), values)
	updated, tableMappings := autoFillDOCXXML(updated, values)
	mappings = append(mappings, tableMappings...)
	if len(mappings) == 0 {
		return data, nil, nil
	}
	files["word/document.xml"] = []byte(updated)

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, zf := range zr.File {
		h := zf.FileHeader
		w, err := zw.CreateHeader(&h)
		if err != nil {
			_ = zw.Close()
			return nil, nil, err
		}
		if _, err := w.Write(files[zf.Name]); err != nil {
			_ = zw.Close()
			return nil, nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, nil, err
	}
	return out.Bytes(), mappings, nil
}

func autoFillDOCXPlaceholders(xmlText string, values map[string]string) (string, []string) {
	var mappings []string
	seen := map[string]bool{}
	for key, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		tokens := []string{"{{" + key + "}}", "[[" + key + "]]", "<<" + key + ">>"}
		for _, token := range tokens {
			if !strings.Contains(xmlText, token) {
				continue
			}
			xmlText = strings.ReplaceAll(xmlText, token, xmlAutoEscape(value))
			m := "DOCX:placeholder=" + key
			if !seen[m] {
				seen[m] = true
				mappings = append(mappings, m)
			}
		}
	}
	return xmlText, mappings
}

func autoFillDOCXXML(xmlText string, values map[string]string) (string, []string) {
	rowRe := regexp.MustCompile(`(?s)<w:tr\b[^>]*>.*?</w:tr>`)
	cellRe := regexp.MustCompile(`(?s)<w:tc\b[^>]*>.*?</w:tc>`)
	tagRe := regexp.MustCompile(`<[^>]+>`)
	textRe := regexp.MustCompile(`(?s)<w:t\b[^>]*>(.*?)</w:t>`)
	var mappings []string
	seen := map[string]bool{}

	updated := rowRe.ReplaceAllStringFunc(xmlText, func(row string) string {
		locs := cellRe.FindAllStringIndex(row, -1)
		if len(locs) < 2 {
			return row
		}
		cells := make([]string, len(locs))
		for i, loc := range locs {
			cells[i] = row[loc[0]:loc[1]]
		}
		changed := false
		for i := 0; i+1 < len(cells); i++ {
			var parts []string
			for _, m := range textRe.FindAllStringSubmatch(cells[i], -1) {
				if len(m) > 1 {
					parts = append(parts, html.UnescapeString(tagRe.ReplaceAllString(m[1], "")))
				}
			}
			label := strings.TrimSpace(strings.Join(parts, " "))
			key := autoFieldKeyFromLabel(label)
			value := strings.TrimSpace(values[key])
			if key == "" || value == "" {
				continue
			}
			var targetParts []string
			for _, m := range textRe.FindAllStringSubmatch(cells[i+1], -1) {
				if len(m) > 1 {
					targetParts = append(targetParts, html.UnescapeString(tagRe.ReplaceAllString(m[1], "")))
				}
			}
			targetText := strings.TrimSpace(strings.Join(targetParts, " "))
			if targetText != "" && targetText != "-" && targetText != "___" && !strings.Contains(strings.ToLower(targetText), "preencher") {
				continue
			}
			insert := `<w:p><w:r><w:t xml:space="preserve">` + xmlAutoEscape(value) + `</w:t></w:r></w:p>`
			cells[i+1] = strings.Replace(cells[i+1], "</w:tc>", insert+"</w:tc>", 1)
			changed = true
			m := "DOCX:" + label + "=" + key
			if !seen[m] {
				seen[m] = true
				mappings = append(mappings, m)
			}
		}
		if !changed {
			return row
		}
		var b strings.Builder
		last := 0
		for i, loc := range locs {
			b.WriteString(row[last:loc[0]])
			b.WriteString(cells[i])
			last = loc[1]
		}
		b.WriteString(row[last:])
		return b.String()
	})
	return updated, mappings
}
