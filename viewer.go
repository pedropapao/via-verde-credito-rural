package main

import (
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type ViewerDashboard struct {
	Projects      []Project
	TotalProjects int
	Draft         int
	WaitingDocs   int
	Sent          int
	Analysis      int
	Approved      int
	Contracted    int
	Rejected      int
	TodayTasks    []ViewerTask
	TodayPending  int
	TodayDone     int
	Overdue       int
}

type ViewerTask struct {
	ProjectTask
	ProjectTitle string
	Producer     string
}

type ViewerProject struct {
	Project         Project
	Client          Client
	Property        Property
	Checklist       []ChecklistItem
	History         []ProjectHistory
	Tasks           []ViewerTask
	DocCompletion   int
	PendingDocs     int
	CompletedDocs   int
	RequiredDocs    int
}

type DailyChecklistView struct {
	Date       string
	Tasks      []ViewerTask
	Projects   []Project
	Done       int
	Pending    int
	Overdue    int
	Total      int
}

func (a *App) routesViewer() http.Handler {
	mux := http.NewServeMux()
	staticFS, _ := fs.Sub(webFS, "web/static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /login", a.loginGet)
	mux.HandleFunc("POST /login", a.loginPost)
	mux.HandleFunc("GET /invite/{token}", a.inviteGet)
	mux.HandleFunc("POST /invite/{token}", a.invitePost)

	mux.Handle("GET /", a.withAuth(http.HandlerFunc(a.viewerDashboard)))
	mux.Handle("POST /logout", a.withAuth(http.HandlerFunc(a.logout)))

	mux.Handle("GET /projects", a.withAuth(http.HandlerFunc(a.viewerProjects)))
	mux.Handle("GET /projects/{id}", a.withAuth(http.HandlerFunc(a.viewerProjectDetail)))
	mux.Handle("POST /projects/{id}/checklist/{item}", a.ownerOnly(http.HandlerFunc(a.viewerChecklistUpdate)))

	mux.Handle("GET /reports/daily", a.withAuth(http.HandlerFunc(a.viewerDailyChecklist)))
	mux.Handle("POST /reports/daily/tasks", a.ownerOnly(http.HandlerFunc(a.viewerDailyTaskCreate)))
	mux.Handle("POST /reports/daily/tasks/{item}/toggle", a.ownerOnly(http.HandlerFunc(a.viewerDailyTaskToggle)))
	mux.Handle("POST /reports/daily/tasks/{item}/delete", a.ownerOnly(http.HandlerFunc(a.viewerDailyTaskDelete)))

	mux.Handle("GET /users", a.ownerOnly(http.HandlerFunc(a.usersList)))
	mux.Handle("POST /users/invite", a.ownerOnly(http.HandlerFunc(a.userInvite)))
	mux.Handle("POST /users/{id}/toggle", a.ownerOnly(http.HandlerFunc(a.userToggle)))
	mux.Handle("GET /settings", a.withAuth(http.HandlerFunc(a.settingsGet)))
	mux.Handle("POST /settings/password", a.withAuth(http.HandlerFunc(a.settingsPassword)))

	return securityHeaders(recoverer(logger(mux)))
}

func (a *App) viewerDashboard(w http.ResponseWriter, r *http.Request) {
	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &projects)
	a.enrichProjects(r, projects)

	d := ViewerDashboard{Projects: firstProjectsViewer(projects, 8), TotalProjects: len(projects)}
	for _, p := range projects {
		s := strings.ToLower(strings.TrimSpace(p.Status))
		phase := strings.ToLower(strings.TrimSpace(p.Phase))
		switch {
		case strings.Contains(s, "reprov") || strings.Contains(s, "negad"):
			d.Rejected++
		case strings.Contains(s, "contrat") || strings.Contains(s, "conclu"):
			d.Contracted++
		case strings.Contains(s, "aprov"):
			d.Approved++
		case strings.Contains(s, "análise") || strings.Contains(s, "analise"):
			d.Analysis++
		case strings.Contains(s, "enviado"):
			d.Sent++
		case strings.Contains(phase, "document") || strings.Contains(phase, "pend"):
			d.WaitingDocs++
		default:
			d.Draft++
		}
	}

	var tasks []ProjectTask
	_ = a.sb.Select(r.Context(), "project_tasks", "select=*&"+order("due_at", false), &tasks)
	projectMap := make(map[string]Project, len(projects))
	for _, p := range projects { projectMap[p.ID] = p }
	today := nowViewer().Format("2006-01-02")
	for _, t := range tasks {
		if t.DueAt == "" { continue }
		row := makeViewerTask(t, projectMap)
		if t.DueAt < today && !viewerTaskDone(t.Status) {
			d.Overdue++
			if len(d.TodayTasks) < 10 { d.TodayTasks = append(d.TodayTasks, row) }
			continue
		}
		if t.DueAt == today {
			if viewerTaskDone(t.Status) { d.TodayDone++ } else { d.TodayPending++ }
			if len(d.TodayTasks) < 10 { d.TodayTasks = append(d.TodayTasks, row) }
		}
	}
	a.render(w, r, "dashboard", ViewData{Title: "Painel de projetos", Data: d})
}

func firstProjectsViewer(in []Project, n int) []Project {
	if len(in) <= n { return in }
	return in[:n]
}

func (a *App) viewerProjects(w http.ResponseWriter, r *http.Request) {
	var rows []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &rows)
	a.enrichProjects(r, rows)
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	bank := strings.TrimSpace(r.URL.Query().Get("bank"))
	filtered := make([]Project, 0, len(rows))
	banks := map[string]bool{}
	statuses := map[string]bool{}
	for _, p := range rows {
		if p.Bank != "" { banks[p.Bank] = true }
		if p.Status != "" { statuses[p.Status] = true }
		hay := strings.ToLower(p.Title+" "+p.ClientName+" "+p.PropertyName+" "+p.Activity+" "+p.Bank+" "+p.Status+" "+p.Phase)
		if q != "" && !strings.Contains(hay, q) { continue }
		if status != "" && p.Status != status { continue }
		if bank != "" && p.Bank != bank { continue }
		filtered = append(filtered, p)
	}
	a.render(w, r, "projects", ViewData{Title: "Projetos", Data: map[string]any{
		"Projects": filtered, "Q": r.URL.Query().Get("q"), "Status": status, "Bank": bank,
		"Banks": sortedViewerKeys(banks), "Statuses": sortedViewerKeys(statuses),
	}})
}

func sortedViewerKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m { out = append(out, k) }
	sort.Strings(out)
	return out
}

func (a *App) viewerProjectDetail(w http.ResponseWriter, r *http.Request) {
	p, err := a.getProject(r, r.PathValue("id"))
	if err != nil { http.NotFound(w, r); return }
	v := ViewerProject{Project: p}
	var clients []Client
	_ = a.sb.Select(r.Context(), "clients", "select=*&"+eq("id", p.ClientID)+"&limit=1", &clients)
	if len(clients) > 0 { v.Client = clients[0] }
	if p.PropertyID != "" {
		var props []Property
		_ = a.sb.Select(r.Context(), "properties", "select=*&"+eq("id", p.PropertyID)+"&limit=1", &props)
		if len(props) > 0 { v.Property = props[0] }
	}
	checkErr := a.sb.Select(r.Context(), "project_checklist", "select=*&"+eq("project_id", p.ID)+"&"+order("created_at", false), &v.Checklist)
	if checkErr == nil && len(v.Checklist) == 0 {
		for _, item := range defaultChecklist360(p) {
			item.ProjectID = p.ID
			_ = a.sb.Insert(r.Context(), "project_checklist", item, nil)
		}
		_ = a.sb.Select(r.Context(), "project_checklist", "select=*&"+eq("project_id", p.ID)+"&"+order("created_at", false), &v.Checklist)
	}
	_ = a.sb.Select(r.Context(), "project_history", "select=*&"+eq("project_id", p.ID)+"&"+order("event_date", true), &v.History)
	var tasks []ProjectTask
	_ = a.sb.Select(r.Context(), "project_tasks", "select=*&"+eq("project_id", p.ID)+"&"+order("due_at", true), &tasks)
	pm := map[string]Project{p.ID:p}
	for _, t := range tasks { v.Tasks = append(v.Tasks, makeViewerTask(t, pm)) }
	for _, c := range v.Checklist {
		if !c.Required { continue }
		v.RequiredDocs++
		if viewerChecklistDone(c.Status) { v.CompletedDocs++ } else { v.PendingDocs++ }
	}
	if v.RequiredDocs > 0 { v.DocCompletion = v.CompletedDocs * 100 / v.RequiredDocs }
	a.render(w, r, "project_detail", ViewData{Title: p.Title, Data: v})
}

func viewerChecklistDone(v string) bool {
	x := strings.ToLower(strings.TrimSpace(v))
	return strings.Contains(x, "recebid") || strings.Contains(x, "concl") || strings.Contains(x, "ok") || strings.Contains(x, "dispens")
}

func (a *App) viewerChecklistUpdate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) { http.Error(w, "Sessão inválida.", 403); return }
	projectID := r.PathValue("id")
	itemID := r.PathValue("item")
	status := strings.TrimSpace(r.FormValue("status"))
	if status == "" { status = "Pendente" }
	var rows []ChecklistItem
	_ = a.sb.Select(r.Context(), "project_checklist", "select=*&"+eq("id", itemID)+"&limit=1", &rows)
	if err := a.sb.Update(r.Context(), "project_checklist", eq("id", itemID), map[string]any{"status": status}, nil); err != nil {
		http.Error(w, "Não foi possível atualizar o checklist.", 500); return
	}
	doc := "Documento"
	if len(rows) > 0 { doc = rows[0].DocumentName }
	a.viewerHistoryAdd(r, projectID, "Checklist documental", doc, "Status alterado para "+status)
	http.Redirect(w, r, "/projects/"+projectID+"?ok="+url.QueryEscape("Checklist atualizado."), http.StatusSeeOther)
}

func (a *App) viewerDailyChecklist(w http.ResponseWriter, r *http.Request) {
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" { date = nowViewer().Format("2006-01-02") }
	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &projects)
	a.enrichProjects(r, projects)
	pm := make(map[string]Project, len(projects))
	for _, p := range projects { pm[p.ID] = p }
	var tasks []ProjectTask
	_ = a.sb.Select(r.Context(), "project_tasks", "select=*&"+order("due_at", false), &tasks)
	v := DailyChecklistView{Date: date, Projects: projects}
	for _, t := range tasks {
		show := t.DueAt == date
		if date == nowViewer().Format("2006-01-02") && t.DueAt != "" && t.DueAt < date && !viewerTaskDone(t.Status) { show = true }
		if !show { continue }
		v.Total++
		if viewerTaskDone(t.Status) { v.Done++ } else { v.Pending++ }
		if t.DueAt != "" && t.DueAt < date && !viewerTaskDone(t.Status) { v.Overdue++ }
		v.Tasks = append(v.Tasks, makeViewerTask(t, pm))
	}
	a.render(w, r, "daily", ViewData{Title: "Checklist diário", Data: v})
}

func (a *App) viewerDailyTaskCreate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) { http.Error(w, "Sessão inválida.", 403); return }
	projectID := strings.TrimSpace(r.FormValue("project_id"))
	title := strings.TrimSpace(r.FormValue("title"))
	if projectID == "" || title == "" { http.Error(w, "Projeto e item são obrigatórios.", 400); return }
	due := strings.TrimSpace(r.FormValue("due_at")); if due == "" { due = nowViewer().Format("2006-01-02") }
	t := ProjectTask{ProjectID: projectID, Title: title, Category: "Checklist diário", Responsible: strings.TrimSpace(r.FormValue("responsible")), RequestedAt: nowViewer().Format("2006-01-02"), DueAt: due, Status: "Pendente", Priority: defaultString(strings.TrimSpace(r.FormValue("priority")), "Normal")}
	if err := a.sb.Insert(r.Context(), "project_tasks", t, nil); err != nil { http.Error(w, "Não foi possível adicionar o item.", 500); return }
	a.viewerHistoryAdd(r, projectID, "Checklist diário", "Item criado", title)
	http.Redirect(w, r, "/reports/daily?date="+url.QueryEscape(due)+"&ok="+url.QueryEscape("Item adicionado."), http.StatusSeeOther)
}

func (a *App) viewerDailyTaskToggle(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) { http.Error(w, "Sessão inválida.", 403); return }
	id := r.PathValue("item")
	var rows []ProjectTask
	if err := a.sb.Select(r.Context(), "project_tasks", "select=*&"+eq("id", id)+"&limit=1", &rows); err != nil || len(rows) == 0 { http.NotFound(w, r); return }
	t := rows[0]
	vals := map[string]any{}
	newStatus := "Concluída"
	if viewerTaskDone(t.Status) {
		newStatus = "Pendente"
		vals["status"] = newStatus
		vals["completed_at"] = nil
	} else {
		vals["status"] = newStatus
		vals["completed_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	if err := a.sb.Update(r.Context(), "project_tasks", eq("id", id), vals, nil); err != nil { http.Error(w, "Não foi possível atualizar o item.", 500); return }
	a.viewerHistoryAdd(r, t.ProjectID, "Checklist diário", t.Title, "Status alterado para "+newStatus)
	date := t.DueAt; if date == "" { date = nowViewer().Format("2006-01-02") }
	http.Redirect(w, r, "/reports/daily?date="+url.QueryEscape(date), http.StatusSeeOther)
}

func (a *App) viewerDailyTaskDelete(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) { http.Error(w, "Sessão inválida.", 403); return }
	id := r.PathValue("item")
	var rows []ProjectTask
	_ = a.sb.Select(r.Context(), "project_tasks", "select=*&"+eq("id", id)+"&limit=1", &rows)
	_ = a.sb.Delete(r.Context(), "project_tasks", eq("id", id))
	date := nowViewer().Format("2006-01-02")
	if len(rows) > 0 {
		date = defaultString(rows[0].DueAt, date)
		a.viewerHistoryAdd(r, rows[0].ProjectID, "Checklist diário", "Item removido", rows[0].Title)
	}
	http.Redirect(w, r, "/reports/daily?date="+url.QueryEscape(date)+"&ok="+url.QueryEscape("Item removido."), http.StatusSeeOther)
}

func (a *App) viewerHistoryAdd(r *http.Request, projectID, eventType, title, details string) {
	if projectID == "" { return }
	u := userFromContext(r.Context())
	userID := ""; if u != nil { userID = u.ID }
	_ = a.sb.Insert(r.Context(), "project_history", ProjectHistory{ProjectID: projectID, EventType: eventType, Title: title, Details: details, UserID: userID}, nil)
}

func makeViewerTask(t ProjectTask, pm map[string]Project) ViewerTask {
	row := ViewerTask{ProjectTask: t}
	if p, ok := pm[t.ProjectID]; ok { row.ProjectTitle = p.Title; row.Producer = p.ClientName }
	return row
}

func viewerTaskDone(v string) bool {
	x := strings.ToLower(strings.TrimSpace(v))
	return strings.Contains(x, "concl") || strings.Contains(x, "feito") || strings.Contains(x, "resolvido")
}

func nowViewer() time.Time {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil { return time.Now() }
	return time.Now().In(loc)
}
