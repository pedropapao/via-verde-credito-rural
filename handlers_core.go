package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &projects)
	var reports []DailyReport
	_ = a.sb.Select(r.Context(), "daily_reports", "select=*&"+order("date", true), &reports)
	total := 0.0
	active := 0
	approved := 0
	for _, p := range projects {
		total += p.FinancedValue
		x := strings.ToLower(p.Status)
		if !strings.Contains(x, "reprov") && !strings.Contains(x, "conclu") {
			active++
		}
		if strings.Contains(x, "aprov") || strings.Contains(x, "contrat") {
			approved++
		}
	}
	today := time.Now().Format("2006-01-02")
	todayCount := 0
	todayValue := 0.0
	for _, d := range reports {
		if strings.HasPrefix(d.Date, today) {
			todayCount++
			todayValue += d.Value
		}
	}
	data := map[string]any{"Projects": projects, "TotalFinanced": total, "Active": active, "Approved": approved, "TodayCount": todayCount, "TodayValue": todayValue, "RecentReports": firstReports(reports, 6)}
	a.render(w, r, "dashboard", ViewData{Title: "Visão geral", Data: data})
}
func firstReports(in []DailyReport, n int) []DailyReport {
	if len(in) < n {
		return in
	}
	return in[:n]
}

func (a *App) projectsList(w http.ResponseWriter, r *http.Request) {
	var rows []Project
	q := "select=*&" + order("updated_at", true)
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		q += "&" + eq("status", status)
	}
	_ = a.sb.Select(r.Context(), "projects", q, &rows)
	a.enrichProjects(r, rows)
	a.render(w, r, "projects", ViewData{Title: "Projetos", Data: map[string]any{"Projects": rows, "Status": r.URL.Query().Get("status")}})
}

func (a *App) enrichProjects(r *http.Request, ps []Project) {
	var cs []Client
	_ = a.sb.Select(r.Context(), "clients", "select=id,name", &cs)
	cm := map[string]string{}
	for _, c := range cs {
		cm[c.ID] = c.Name
	}
	var props []Property
	_ = a.sb.Select(r.Context(), "properties", "select=id,name", &props)
	pm := map[string]string{}
	for _, p := range props {
		pm[p.ID] = p.Name
	}
	for i := range ps {
		ps[i].ClientName = cm[ps[i].ClientID]
		ps[i].PropertyName = pm[ps[i].PropertyID]
	}
}

func (a *App) projectNewGet(w http.ResponseWriter, r *http.Request) {
	a.projectForm(w, r, Project{}, false, "")
}
func (a *App) projectEditGet(w http.ResponseWriter, r *http.Request) {
	p, err := a.getProject(r, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.projectForm(w, r, p, true, "")
}
func (a *App) projectForm(w http.ResponseWriter, r *http.Request, p Project, edit bool, errMsg string) {
	var cs []Client
	_ = a.sb.Select(r.Context(), "clients", "select=*&"+order("name", false), &cs)
	var props []Property
	_ = a.sb.Select(r.Context(), "properties", "select=*&"+order("name", false), &props)
	a.render(w, r, "project_form", ViewData{Title: map[bool]string{true: "Editar projeto", false: "Novo projeto"}[edit], Error: errMsg, Data: map[string]any{"Project": p, "Clients": cs, "Properties": props, "Edit": edit}})
}

func projectFromForm(r *http.Request) Project {
	return Project{ClientID: r.FormValue("client_id"), PropertyID: r.FormValue("property_id"), Title: strings.TrimSpace(r.FormValue("title")), Modality: r.FormValue("modality"), Activity: strings.TrimSpace(r.FormValue("activity")), Bank: strings.TrimSpace(r.FormValue("bank")), Program: strings.TrimSpace(r.FormValue("program")), TotalValue: parseFloat(r.FormValue("total_value")), FinancedValue: parseFloat(r.FormValue("financed_value")), InterestRate: parseFloat(r.FormValue("interest_rate")), TermMonths: parseInt(r.FormValue("term_months")), Status: r.FormValue("status"), Phase: strings.TrimSpace(r.FormValue("phase")), TechnicalSummary: strings.TrimSpace(r.FormValue("technical_summary")), Alerts: strings.TrimSpace(r.FormValue("alerts"))}
}
func validateProject(p Project) error {
	if p.ClientID == "" {
		return fmt.Errorf("selecione o produtor")
	}
	if p.Title == "" {
		return fmt.Errorf("informe o título do projeto")
	}
	if p.Modality == "" {
		return fmt.Errorf("selecione a modalidade")
	}
	return nil
}

func (a *App) projectNewPost(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	p := projectFromForm(r)
	if err := validateProject(p); err != nil {
		a.projectForm(w, r, p, false, err.Error())
		return
	}
	p.Status = defaultString(p.Status, "Em andamento")
	var out []Project
	if err := a.sb.Insert(r.Context(), "projects", p, &out); err != nil || len(out) == 0 {
		a.projectForm(w, r, p, false, "Não foi possível salvar o projeto.")
		return
	}
	a.audit(r.Context(), userFromContext(r.Context()), "create", "project", out[0].ID, p.Title)
	http.Redirect(w, r, "/projects/"+out[0].ID+"?ok="+url.QueryEscape("Projeto criado."), 303)
}
func (a *App) projectEditPost(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	p := projectFromForm(r)
	p.ID = r.PathValue("id")
	if err := validateProject(p); err != nil {
		a.projectForm(w, r, p, true, err.Error())
		return
	}
	vals := map[string]any{"client_id": p.ClientID, "property_id": nullIfEmpty(p.PropertyID), "title": p.Title, "modality": p.Modality, "activity": p.Activity, "bank": p.Bank, "program": p.Program, "total_value": p.TotalValue, "financed_value": p.FinancedValue, "interest_rate": p.InterestRate, "term_months": p.TermMonths, "status": p.Status, "phase": p.Phase, "technical_summary": p.TechnicalSummary, "alerts": p.Alerts, "updated_at": time.Now().UTC().Format(time.RFC3339)}
	if err := a.sb.Update(r.Context(), "projects", eq("id", p.ID), vals, nil); err != nil {
		a.projectForm(w, r, p, true, "Não foi possível atualizar o projeto.")
		return
	}
	a.audit(r.Context(), userFromContext(r.Context()), "update", "project", p.ID, p.Title)
	http.Redirect(w, r, "/projects/"+p.ID+"?ok="+url.QueryEscape("Projeto atualizado."), 303)
}
func defaultString(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}
func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func (a *App) getProject(r *http.Request, id string) (Project, error) {
	var rows []Project
	if err := a.sb.Select(r.Context(), "projects", eq("id", id)+"&limit=1", &rows); err != nil || len(rows) == 0 {
		return Project{}, fmt.Errorf("not found")
	}
	a.enrichProjects(r, rows)
	return rows[0], nil
}
func (a *App) projectDetail(w http.ResponseWriter, r *http.Request) {
	p, err := a.getProject(r, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var files []ProjectFile
	_ = a.sb.Select(r.Context(), "project_files", eq("project_id", p.ID)+"&"+order("created_at", true), &files)
	a.render(w, r, "project_detail", ViewData{Title: p.Title, Data: map[string]any{"Project": p, "Files": files}})
}

func (a *App) projectFileUpload(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	id := r.PathValue("id")
	if _, err := a.getProject(r, id); err != nil {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 45<<20)
	if err := r.ParseMultipartForm(45 << 20); err != nil {
		http.Error(w, "Arquivo muito grande. Limite: 45 MB.", 400)
		return
	}
	f, h, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Selecione um arquivo.", 400)
		return
	}
	defer f.Close()
	name := safeFilename(h.Filename)
	ct := h.Header.Get("Content-Type")
	if ct == "" {
		ct = mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	objectPath := fmt.Sprintf("projects/%s/%d-%s", id, time.Now().UnixNano(), name)
	if err := a.sb.Upload(r.Context(), a.cfg.StorageBucket, objectPath, ct, io.LimitReader(f, 45<<20)); err != nil {
		http.Error(w, "Falha no upload: "+err.Error(), 500)
		return
	}
	u := userFromContext(r.Context())
	var out []ProjectFile
	meta := ProjectFile{ProjectID: id, Name: name, StoragePath: objectPath, ContentType: ct, SizeBytes: h.Size, UploadedBy: u.ID}
	if err := a.sb.Insert(r.Context(), "project_files", meta, &out); err != nil {
		_ = a.sb.DeleteObject(r.Context(), a.cfg.StorageBucket, objectPath)
		http.Error(w, "Falha ao registrar arquivo.", 500)
		return
	}
	a.audit(r.Context(), u, "upload", "file", firstFileID(out), name)
	http.Redirect(w, r, "/projects/"+id+"?ok="+url.QueryEscape("Arquivo enviado."), 303)
}
func firstFileID(v []ProjectFile) string {
	if len(v) > 0 {
		return v[0].ID
	}
	return ""
}
func (a *App) fileDownload(w http.ResponseWriter, r *http.Request) {
	var rows []ProjectFile
	if err := a.sb.Select(r.Context(), "project_files", eq("id", r.PathValue("id"))+"&limit=1", &rows); err != nil || len(rows) == 0 {
		http.NotFound(w, r)
		return
	}
	b, ct, err := a.sb.Download(r.Context(), a.cfg.StorageBucket, rows[0].StoragePath)
	if err != nil {
		http.Error(w, "Não foi possível abrir o arquivo.", 500)
		return
	}
	if ct == "" {
		ct = rows[0].ContentType
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, strings.ReplaceAll(rows[0].Name, "\"", "")))
	w.Write(b)
}
func (a *App) fileDelete(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	var rows []ProjectFile
	if err := a.sb.Select(r.Context(), "project_files", eq("id", r.PathValue("id"))+"&limit=1", &rows); err != nil || len(rows) == 0 {
		http.NotFound(w, r)
		return
	}
	_ = a.sb.DeleteObject(r.Context(), a.cfg.StorageBucket, rows[0].StoragePath)
	_ = a.sb.Delete(r.Context(), "project_files", eq("id", rows[0].ID))
	a.audit(r.Context(), userFromContext(r.Context()), "delete", "file", rows[0].ID, rows[0].Name)
	http.Redirect(w, r, "/projects/"+rows[0].ProjectID+"?ok="+url.QueryEscape("Arquivo excluído."), 303)
}

func (a *App) clientsList(w http.ResponseWriter, r *http.Request) {
	var rows []Client
	_ = a.sb.Select(r.Context(), "clients", "select=*&"+order("name", false), &rows)
	a.render(w, r, "clients", ViewData{Title: "Produtores / clientes", Data: rows})
}
func (a *App) clientCreate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	c := Client{Name: strings.TrimSpace(r.FormValue("name")), Document: strings.TrimSpace(r.FormValue("document")), Phone: strings.TrimSpace(r.FormValue("phone")), Email: strings.TrimSpace(r.FormValue("email")), RBA: parseFloat(r.FormValue("rba")), Classification: r.FormValue("classification"), Notes: strings.TrimSpace(r.FormValue("notes"))}
	if c.Name == "" {
		http.Error(w, "Nome obrigatório.", 400)
		return
	}
	var out []Client
	if err := a.sb.Insert(r.Context(), "clients", c, &out); err != nil {
		http.Error(w, "Falha ao salvar cliente.", 500)
		return
	}
	a.audit(r.Context(), userFromContext(r.Context()), "create", "client", firstClientID(out), c.Name)
	http.Redirect(w, r, "/clients?ok="+url.QueryEscape("Cliente cadastrado."), 303)
}
func firstClientID(v []Client) string {
	if len(v) > 0 {
		return v[0].ID
	}
	return ""
}

func (a *App) propertiesList(w http.ResponseWriter, r *http.Request) {
	var rows []Property
	_ = a.sb.Select(r.Context(), "properties", "select=*&"+order("name", false), &rows)
	var cs []Client
	_ = a.sb.Select(r.Context(), "clients", "select=id,name&"+order("name", false), &cs)
	a.render(w, r, "properties", ViewData{Title: "Propriedades", Data: map[string]any{"Properties": rows, "Clients": cs}})
}
func (a *App) propertyCreate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	p := Property{ClientID: r.FormValue("client_id"), Name: strings.TrimSpace(r.FormValue("name")), City: strings.TrimSpace(r.FormValue("city")), State: strings.ToUpper(strings.TrimSpace(r.FormValue("state"))), AreaHa: parseFloat(r.FormValue("area_ha")), Registry: strings.TrimSpace(r.FormValue("registry")), CAR: strings.TrimSpace(r.FormValue("car")), CCIR: strings.TrimSpace(r.FormValue("ccir")), ITR: strings.TrimSpace(r.FormValue("itr")), Tenure: r.FormValue("tenure"), Notes: strings.TrimSpace(r.FormValue("notes"))}
	if p.ClientID == "" || p.Name == "" {
		http.Error(w, "Cliente e propriedade são obrigatórios.", 400)
		return
	}
	var out []Property
	if err := a.sb.Insert(r.Context(), "properties", p, &out); err != nil {
		http.Error(w, "Falha ao salvar propriedade.", 500)
		return
	}
	a.audit(r.Context(), userFromContext(r.Context()), "create", "property", firstPropID(out), p.Name)
	http.Redirect(w, r, "/properties?ok="+url.QueryEscape("Propriedade cadastrada."), 303)
}
func firstPropID(v []Property) string {
	if len(v) > 0 {
		return v[0].ID
	}
	return ""
}

func (a *App) documentsList(w http.ResponseWriter, r *http.Request) {
	var files []ProjectFile
	_ = a.sb.Select(r.Context(), "project_files", "select=*&"+order("created_at", true), &files)
	var ps []Project
	_ = a.sb.Select(r.Context(), "projects", "select=id,title", &ps)
	pm := map[string]string{}
	for _, p := range ps {
		pm[p.ID] = p.Title
	}
	a.render(w, r, "documents", ViewData{Title: "Documentos", Data: map[string]any{"Files": files, "ProjectNames": pm}})
}

func (a *App) dailyList(w http.ResponseWriter, r *http.Request) {
	var rows []DailyReport
	q := "select=*&" + order("date", true)
	if d := r.URL.Query().Get("date"); d != "" {
		q = "select=*&" + eq("date", d) + "&" + order("number", false)
	}
	_ = a.sb.Select(r.Context(), "daily_reports", q, &rows)
	total := 0.0
	comm := 0.0
	for _, x := range rows {
		total += x.Value
		comm += x.Commission
	}
	a.render(w, r, "daily", ViewData{Title: "Relatório do dia", Data: map[string]any{"Rows": rows, "Total": total, "Commission": comm, "Date": r.URL.Query().Get("date")}})
}
func (a *App) dailyCreate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	rate := parseFloat(r.FormValue("commission_rate"))
	if rate == 0 {
		rate = 0.5
	}
	value := parseFloat(r.FormValue("value"))
	row := DailyReport{Number: parseInt(r.FormValue("number")), Date: r.FormValue("date"), Value: value, Bank: strings.TrimSpace(r.FormValue("bank")), Referral: strings.TrimSpace(r.FormValue("referral")), Producer: strings.TrimSpace(r.FormValue("producer")), ProjectType: strings.TrimSpace(r.FormValue("project_type")), SentDate: r.FormValue("sent_date"), Situation: strings.TrimSpace(r.FormValue("situation")), CommissionRate: rate, Commission: value * rate / 100, PedroPercent: parseFloat(r.FormValue("pedro_percent")), PaidBy: strings.TrimSpace(r.FormValue("paid_by")), Notes: strings.TrimSpace(r.FormValue("notes"))}
	if row.Date == "" {
		row.Date = time.Now().Format("2006-01-02")
	}
	if row.Producer == "" {
		http.Error(w, "Produtor obrigatório.", 400)
		return
	}
	var out []DailyReport
	if err := a.sb.Insert(r.Context(), "daily_reports", row, &out); err != nil {
		http.Error(w, "Falha ao salvar relatório.", 500)
		return
	}
	a.audit(r.Context(), userFromContext(r.Context()), "create", "daily_report", firstReportID(out), row.Producer)
	http.Redirect(w, r, "/reports/daily?date="+url.QueryEscape(row.Date)+"&ok="+url.QueryEscape("Lançamento incluído."), 303)
}
func firstReportID(v []DailyReport) string {
	if len(v) > 0 {
		return v[0].ID
	}
	return ""
}
func (a *App) dailyExportCSV(w http.ResponseWriter, r *http.Request) {
	var rows []DailyReport
	q := "select=*&" + order("date", false)
	if d := r.URL.Query().Get("date"); d != "" {
		q = "select=*&" + eq("date", d) + "&" + order("number", false)
	}
	_ = a.sb.Select(r.Context(), "daily_reports", q, &rows)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=relatorio_diario_via_verde.csv")
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	defer cw.Flush()
	_ = cw.Write([]string{"N°", "DATA", "VALOR R$", "BANCO", "INDICAÇÃO", "PRODUTOR", "TIPO", "DATA DE ENVIO", "SITUAÇÃO", "COMISSÃO %", "COMISSÃO R$", "% PEDRO", "QUEM PAGOU", "OBSERVAÇÕES"})
	for _, x := range rows {
		_ = cw.Write([]string{strconv.Itoa(x.Number), dateBR(x.Date), fmt.Sprintf("%.2f", x.Value), x.Bank, x.Referral, x.Producer, x.ProjectType, dateBR(x.SentDate), x.Situation, fmt.Sprintf("%.2f", x.CommissionRate), fmt.Sprintf("%.2f", x.Commission), fmt.Sprintf("%.2f", x.PedroPercent), x.PaidBy, x.Notes})
	}
}

func (a *App) rulesList(w http.ResponseWriter, r *http.Request) {
	var rows []Rule
	_ = a.sb.Select(r.Context(), "rules", "select=*&active=eq.true&"+order("category", false), &rows)
	a.render(w, r, "rules", ViewData{Title: "Regras e referências", Data: rows})
}
func (a *App) ruleCreate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	row := Rule{Category: r.FormValue("category"), Title: r.FormValue("title"), Reference: r.FormValue("reference"), Summary: r.FormValue("summary"), SourceURL: r.FormValue("source_url"), VerifiedAt: r.FormValue("verified_at"), Active: true}
	if row.Title == "" {
		http.Error(w, "Título obrigatório.", 400)
		return
	}
	if err := a.sb.Insert(r.Context(), "rules", row, nil); err != nil {
		http.Error(w, "Falha ao salvar regra.", 500)
		return
	}
	http.Redirect(w, r, "/rules?ok="+url.QueryEscape("Referência adicionada."), 303)
}

func (a *App) usersList(w http.ResponseWriter, r *http.Request) {
	var us []User
	_ = a.sb.Select(r.Context(), "users", "select=id,name,username,email,role,active,created_at,last_login_at&"+order("created_at", false), &us)
	var inv []Invite
	_ = a.sb.Select(r.Context(), "invites", "select=*&used_at=is.null&"+order("created_at", true), &inv)
	a.render(w, r, "users", ViewData{Title: "Usuários e acessos", Data: map[string]any{"Users": us, "Invites": inv}})
}
func (a *App) userInvite(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	raw, err := randomToken(32)
	if err != nil {
		http.Error(w, "Falha ao criar convite.", 500)
		return
	}
	days := parseInt(r.FormValue("days"))
	if days < 1 {
		days = 7
	}
	u := userFromContext(r.Context())
	row := Invite{TokenHash: sha256Hex(raw), Name: strings.TrimSpace(r.FormValue("name")), Email: strings.TrimSpace(r.FormValue("email")), Role: "viewer", ExpiresAt: time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Format(time.RFC3339), CreatedBy: u.ID}
	if err := a.sb.Insert(r.Context(), "invites", row, nil); err != nil {
		http.Error(w, "Falha ao criar convite.", 500)
		return
	}
	link := baseURL(r, a.cfg) + "/invite/" + raw
	http.Redirect(w, r, "/users?ok="+url.QueryEscape("Convite criado: "+link), 303)
}
func (a *App) userToggle(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	id := r.PathValue("id")
	if id == userFromContext(r.Context()).ID {
		http.Error(w, "Você não pode desativar o próprio usuário.", 400)
		return
	}
	var us []User
	if err := a.sb.Select(r.Context(), "users", eq("id", id)+"&limit=1", &us); err != nil || len(us) == 0 {
		http.NotFound(w, r)
		return
	}
	_ = a.sb.Update(r.Context(), "users", eq("id", id), map[string]any{"active": !us[0].Active}, nil)
	a.audit(r.Context(), userFromContext(r.Context()), "toggle", "user", id, us[0].Username)
	http.Redirect(w, r, "/users?ok="+url.QueryEscape("Acesso atualizado."), 303)
}

func (a *App) settingsGet(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "settings", ViewData{Title: "Meu perfil"})
}
func (a *App) settingsPassword(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", 403)
		return
	}
	u := userFromContext(r.Context())
	if !VerifyPassword(r.FormValue("current_password"), u.PasswordHash) {
		a.render(w, r, "settings", ViewData{Title: "Meu perfil", Error: "Senha atual incorreta."})
		return
	}
	if r.FormValue("new_password") != r.FormValue("confirm_password") {
		a.render(w, r, "settings", ViewData{Title: "Meu perfil", Error: "As novas senhas não coincidem."})
		return
	}
	hash, err := HashPassword(r.FormValue("new_password"))
	if err != nil {
		a.render(w, r, "settings", ViewData{Title: "Meu perfil", Error: err.Error()})
		return
	}
	if err := a.sb.Update(r.Context(), "users", eq("id", u.ID), map[string]any{"password_hash": hash}, nil); err != nil {
		a.render(w, r, "settings", ViewData{Title: "Meu perfil", Error: "Não foi possível alterar a senha."})
		return
	}
	_ = a.sb.Delete(r.Context(), "sessions", "user_id=eq."+url.QueryEscape(u.ID))
	clearSessionCookie(w)
	http.Redirect(w, r, "/login?ok="+url.QueryEscape("Senha alterada. Entre novamente."), 303)
}

func (a *App) sortedClientNames(r *http.Request) []string {
	var rows []Client
	_ = a.sb.Select(r.Context(), "clients", "select=name", &rows)
	out := make([]string, 0, len(rows))
	for _, c := range rows {
		out = append(out, c.Name)
	}
	sort.Strings(out)
	return out
}
