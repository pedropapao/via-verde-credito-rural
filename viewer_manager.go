package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type ManagerProjectRow struct {
	ViewerProjectRow
	StageDays  int
	Dependency string
}

type KanbanColumn struct {
	Key      string
	Title    string
	Projects []ManagerProjectRow
}

type KanbanView struct {
	Columns []KanbanColumn
	Total   int
}

type AttentionView struct {
	WaitingDocs []ManagerProjectRow
	Stale       []ManagerProjectRow
	Overdue     []ManagerProjectRow
	TotalUnique int
}

type DailyProjectSummary struct {
	Project          Project
	Producer         string
	Activities       []string
	PendingTasks     []string
	PendingDocuments []string
	PendingDocs      int
	Dependency       string
	NextStep         string
}

type DailyPendingSummary struct {
	ProjectID    string
	ProjectTitle string
	Producer     string
	Text         string
	Responsible string
}

type DailySummaryView struct {
	Date            string
	Events          []ViewerHistoryEntry
	CompletedTasks  []ViewerTask
	ProjectSummaries []DailyProjectSummary
	PendingNext     []DailyPendingSummary
	ProjectsMoved   int
	ProjectsAdvanced int
	TotalActions    int
	DocsReceived    int
	TasksDone       int
	ProgressUpdates int
	Sent            int
	Approved        int
	Contracted      int
	ShortText       string
	FullText        string
}

func (a *App) routesManager() http.Handler {
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
	mux.Handle("POST /projects/{id}/progress", a.ownerOnly(http.HandlerFunc(a.viewerProgressUpdate)))

	mux.Handle("GET /kanban", a.withAuth(http.HandlerFunc(a.viewerKanban)))
	mux.Handle("GET /attention", a.withAuth(http.HandlerFunc(a.viewerAttention)))

	mux.Handle("GET /reports/daily", a.withAuth(http.HandlerFunc(a.viewerDailyChecklist)))
	mux.Handle("POST /reports/daily/tasks", a.ownerOnly(http.HandlerFunc(a.viewerDailyTaskCreate)))
	mux.Handle("POST /reports/daily/tasks/{item}/toggle", a.ownerOnly(http.HandlerFunc(a.viewerDailyTaskToggle)))
	mux.Handle("POST /reports/daily/tasks/{item}/delete", a.ownerOnly(http.HandlerFunc(a.viewerDailyTaskDelete)))
	mux.Handle("GET /reports/summary", a.withAuth(http.HandlerFunc(a.viewerDailySummary)))

	mux.Handle("GET /users", a.ownerOnly(http.HandlerFunc(a.usersList)))
	mux.Handle("POST /users/invite", a.ownerOnly(http.HandlerFunc(a.userInvite)))
	mux.Handle("POST /users/{id}/toggle", a.ownerOnly(http.HandlerFunc(a.userToggle)))
	mux.Handle("GET /settings", a.withAuth(http.HandlerFunc(a.settingsGet)))
	mux.Handle("POST /settings/password", a.withAuth(http.HandlerFunc(a.settingsPassword)))

	return securityHeaders(recoverer(logger(mux)))
}

func (a *App) loadManagerRows(r *http.Request) ([]ManagerProjectRow, []Project, []ProjectTask, []ChecklistItem, []ProjectHistory) {
	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &projects)
	a.enrichProjects(r, projects)
	var tasks []ProjectTask
	_ = a.sb.Select(r.Context(), "project_tasks", "select=*&"+order("due_at", false), &tasks)
	var checklist []ChecklistItem
	_ = a.sb.Select(r.Context(), "project_checklist", "select=*&"+order("created_at", false), &checklist)
	var history []ProjectHistory
	_ = a.sb.Select(r.Context(), "project_history", "select=*&"+order("event_date", true), &history)
	base := buildViewerProjectRows(projects, checklist, tasks, history)
	rows := make([]ManagerProjectRow, 0, len(base))
	for _, row := range base {
		rows = append(rows, ManagerProjectRow{
			ViewerProjectRow: row,
			StageDays:        managerStageDays(row.Project, history),
			Dependency:       managerDependency(row.Project.Responsible),
		})
	}
	return rows, projects, tasks, checklist, history
}

func (a *App) viewerKanban(w http.ResponseWriter, r *http.Request) {
	rows, _, _, _, _ := a.loadManagerRows(r)
	columns := []KanbanColumn{
		{Key: "preparacao", Title: "Em preparação"},
		{Key: "documentos", Title: "Aguardando documentos"},
		{Key: "enviado", Title: "Enviado ao banco"},
		{Key: "analise", Title: "Em análise"},
		{Key: "aprovado", Title: "Aprovado"},
		{Key: "contratado", Title: "Contratado"},
		{Key: "reprovado", Title: "Reprovado"},
	}
	idx := map[string]int{}
	for i := range columns { idx[columns[i].Key] = i }
	for _, row := range rows {
		key := managerStageKey(row.Project)
		columns[idx[key]].Projects = append(columns[idx[key]].Projects, row)
	}
	a.render(w, r, "clients", ViewData{Title: "Pipeline de projetos", Data: KanbanView{Columns: columns, Total: len(rows)}})
}

func (a *App) viewerAttention(w http.ResponseWriter, r *http.Request) {
	rows, _, _, _, _ := a.loadManagerRows(r)
	v := AttentionView{}
	unique := map[string]bool{}
	for _, row := range rows {
		if viewerProjectClosed(row.Status) { continue }
		if row.PendingDocs > 0 || managerStageKey(row.Project) == "documentos" {
			v.WaitingDocs = append(v.WaitingDocs, row); unique[row.ID] = true
		}
		if row.StaleDays >= 5 {
			v.Stale = append(v.Stale, row); unique[row.ID] = true
		}
		if row.OverdueTasks > 0 {
			v.Overdue = append(v.Overdue, row); unique[row.ID] = true
		}
	}
	v.TotalUnique = len(unique)
	a.render(w, r, "properties", ViewData{Title: "Projetos que exigem atenção", Data: v})
}

func (a *App) viewerProgressUpdate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) { http.Error(w, "Sessão inválida.", http.StatusForbidden); return }
	id := r.PathValue("id")
	p, err := a.getProject(r, id)
	if err != nil { http.NotFound(w, r); return }

	status := managerNormalizeStatus(strings.TrimSpace(r.FormValue("status")), p.Status)
	phase := strings.TrimSpace(r.FormValue("phase"))
	dependency := strings.TrimSpace(r.FormValue("responsible"))
	alerts := strings.TrimSpace(r.FormValue("alerts"))
	if phase == "" { phase = p.Phase }
	if dependency == "" { dependency = p.Responsible }

	changed := status != p.Status || phase != p.Phase || dependency != p.Responsible || alerts != p.Alerts
	if !changed {
		http.Redirect(w, r, "/projects/"+id+"?ok="+url.QueryEscape("Nenhuma alteração no andamento."), http.StatusSeeOther)
		return
	}
	vals := map[string]any{
		"status": status, "phase": phase, "responsible": dependency, "alerts": alerts,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err := a.sb.Update(r.Context(), "projects", eq("id", id), vals, nil); err != nil {
		http.Error(w, "Não foi possível atualizar o andamento do projeto.", http.StatusInternalServerError); return
	}
	parts := []string{}
	if status != p.Status { parts = append(parts, "Status: "+p.Status+" → "+status) }
	if phase != p.Phase { parts = append(parts, "Fase: "+defaultManager(p.Phase, "—")+" → "+phase) }
	if dependency != p.Responsible { parts = append(parts, "Dependência: "+defaultManager(p.Responsible, "—")+" → "+dependency) }
	if alerts != p.Alerts { parts = append(parts, "Observação atualizada") }
	a.viewerHistoryAdd(r, id, "Andamento", "Andamento atualizado", strings.Join(parts, " · "))
	http.Redirect(w, r, "/projects/"+id+"?ok="+url.QueryEscape("Andamento atualizado e registrado no histórico."), http.StatusSeeOther)
}

func (a *App) viewerDailySummary(w http.ResponseWriter, r *http.Request) {
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" { date = nowViewer().Format("2006-01-02") }
	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &projects)
	a.enrichProjects(r, projects)
	pm := map[string]Project{}
	for _, p := range projects { pm[p.ID] = p }
	var history []ProjectHistory
	_ = a.sb.Select(r.Context(), "project_history", "select=*&"+order("event_date", true), &history)
	var tasks []ProjectTask
	_ = a.sb.Select(r.Context(), "project_tasks", "select=*&"+order("due_at", false), &tasks)
	var checklist []ChecklistItem
	_ = a.sb.Select(r.Context(), "project_checklist", "select=*&"+order("created_at", false), &checklist)

	v := DailySummaryView{Date: date}
	moved := map[string]bool{}
	advanced := map[string]bool{}
	activities := map[string][]string{}
	completedByProject := map[string][]string{}
	openTasks := map[string][]ProjectTask{}
	pendingDocs := map[string][]string{}

	for _, c := range checklist {
		if c.ProjectID == "" || !c.Required || viewerChecklistDone(c.Status) { continue }
		pendingDocs[c.ProjectID] = appendUniqueManager(pendingDocs[c.ProjectID], c.DocumentName)
	}

	for _, t := range tasks {
		if viewerTaskDone(t.Status) {
			key := managerDateKey(t.CompletedAt)
			if key == "" { key = t.DueAt }
			if key != date { continue }
			v.TasksDone++
			v.CompletedTasks = append(v.CompletedTasks, makeViewerTask(t, pm))
			moved[t.ProjectID] = true
			line := "Atividade concluída: "+t.Title
			completedByProject[t.ProjectID] = appendUniqueManager(completedByProject[t.ProjectID], line)
			continue
		}
		if t.ProjectID != "" { openTasks[t.ProjectID] = append(openTasks[t.ProjectID], t) }
	}
	for id := range openTasks {
		sort.SliceStable(openTasks[id], func(i, j int) bool {
			ai, aj := openTasks[id][i].DueAt, openTasks[id][j].DueAt
			if ai == "" { return false }
			if aj == "" { return true }
			return ai < aj
		})
	}

	for _, h := range history {
		if managerDateKey(h.EventDate) != date { continue }
		entry := ViewerHistoryEntry{ProjectHistory: h}
		if p, ok := pm[h.ProjectID]; ok { entry.ProjectTitle = p.Title; entry.Producer = p.ClientName }
		v.Events = append(v.Events, entry)
		if h.ProjectID != "" { moved[h.ProjectID] = true }
		low := strings.ToLower(h.EventType+" "+h.Title+" "+h.Details)
		if strings.Contains(low, "checklist documental") && (strings.Contains(low, "recebido") || strings.Contains(low, "dispensado")) { v.DocsReceived++ }
		if strings.Contains(low, "andamento") { v.ProgressUpdates++ }
		if strings.Contains(low, "status:") || strings.Contains(low, "→ enviado") || strings.Contains(low, "→ em análise") || strings.Contains(low, "→ em analise") || strings.Contains(low, "→ aprovado") || strings.Contains(low, "→ contratado") { advanced[h.ProjectID] = true }
		if strings.Contains(low, "enviado ao banco") || strings.Contains(low, "→ enviado") { v.Sent++ }
		if strings.Contains(low, "aprovado") { v.Approved++ }
		if strings.Contains(low, "contratado") { v.Contracted++ }
		if line := managerHistoryActivity(h); line != "" {
			activities[h.ProjectID] = appendUniqueManager(activities[h.ProjectID], line)
		}
	}

	for projectID, lines := range completedByProject {
		for _, line := range lines { activities[projectID] = appendUniqueManager(activities[projectID], line) }
	}

	ids := make([]string, 0, len(moved))
	for id := range moved { if id != "" { ids = append(ids, id) } }
	sort.SliceStable(ids, func(i, j int) bool {
		pi, iok := pm[ids[i]]; pj, jok := pm[ids[j]]
		if !iok { return false }; if !jok { return true }
		return strings.ToLower(pi.ClientName+pi.Title) < strings.ToLower(pj.ClientName+pj.Title)
	})

	for _, id := range ids {
		p, ok := pm[id]; if !ok { continue }
		ps := DailyProjectSummary{
			Project: p,
			Producer: defaultManager(p.ClientName, "Produtor a confirmar"),
			Activities: activities[id],
			Dependency: managerDependency(p.Responsible),
			PendingDocuments: pendingDocs[id],
			PendingDocs: len(pendingDocs[id]),
		}
		for _, t := range openTasks[id] {
			ps.PendingTasks = append(ps.PendingTasks, t.Title)
			if len(ps.PendingTasks) >= 3 { break }
		}
		ps.NextStep = managerNextStep(p, openTasks[id], len(pendingDocs[id]))
		if len(ps.Activities) == 0 { ps.Activities = []string{"Projeto movimentado no sistema"} }
		v.TotalActions += len(ps.Activities)
		v.ProjectSummaries = append(v.ProjectSummaries, ps)
		if !viewerProjectClosed(p.Status) && ps.NextStep != "" {
			v.PendingNext = append(v.PendingNext, DailyPendingSummary{ProjectID:id, ProjectTitle:p.Title, Producer:ps.Producer, Text:ps.NextStep, Responsible:ps.Dependency})
		}
	}
	v.ProjectsMoved = len(v.ProjectSummaries)
	v.ProjectsAdvanced = len(advanced)
	v.ShortText = managerDailyShortText(v)
	v.FullText = managerDailyFullText(v)
	a.render(w, r, "rules", ViewData{Title: "Relatório do dia", Data: v})
}

func managerHistoryActivity(h ProjectHistory) string {
	kind := strings.ToLower(strings.TrimSpace(h.EventType))
	details := strings.TrimSpace(h.Details)
	title := strings.TrimSpace(h.Title)
	low := strings.ToLower(kind+" "+title+" "+details)
	if strings.Contains(kind, "checklist diário") || strings.Contains(kind, "checklist diario") {
		if strings.Contains(low, "status alterado para conclu") || strings.Contains(low, "item criado") || strings.Contains(low, "item removido") { return "" }
	}
	if strings.Contains(kind, "checklist documental") {
		if strings.Contains(low, "recebid") { return "Documento recebido: "+title }
		if strings.Contains(low, "dispens") { return "Documento dispensado: "+title }
		if strings.Contains(low, "pendente") { return "Documento marcado como pendente: "+title }
	}
	if strings.Contains(kind, "andamento") {
		if details != "" { return "Andamento atualizado: "+details }
		return "Andamento atualizado"
	}
	if title == "" { title = defaultManager(h.EventType, "Movimentação registrada") }
	if details != "" { return title+": "+details }
	return title
}

func managerNextStep(p Project, tasks []ProjectTask, pendingDocs int) string {
	if strings.TrimSpace(p.Alerts) != "" { return strings.TrimSpace(p.Alerts) }
	if len(tasks) > 0 && strings.TrimSpace(tasks[0].Title) != "" { return strings.TrimSpace(tasks[0].Title) }
	if pendingDocs > 0 { return "Receber e conferir os documentos pendentes" }
	dep := strings.ToLower(strings.TrimSpace(p.Responsible))
	stage := managerStageKey(p)
	if strings.Contains(dep, "banco") || stage == "enviado" || stage == "analise" { return "Aguardar retorno do banco e acompanhar a proposta" }
	if stage == "aprovado" { return "Acompanhar formalização e contratação" }
	return "Acompanhar o próximo avanço do projeto"
}

func appendUniqueManager(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" { return items }
	for _, item := range items { if strings.EqualFold(strings.TrimSpace(item), value) { return items } }
	return append(items, value)
}

func managerStageKey(p Project) string {
	s := strings.ToLower(strings.TrimSpace(p.Status))
	phase := strings.ToLower(strings.TrimSpace(p.Phase))
	switch {
	case strings.Contains(s, "reprov") || strings.Contains(s, "negad"):
		return "reprovado"
	case strings.Contains(s, "contrat") || strings.Contains(s, "conclu"):
		return "contratado"
	case strings.Contains(s, "aprov"):
		return "aprovado"
	case strings.Contains(s, "análise") || strings.Contains(s, "analise"):
		return "analise"
	case strings.Contains(s, "enviado") || strings.Contains(s, "banco"):
		return "enviado"
	case strings.Contains(s, "aguard") && strings.Contains(s, "doc"), strings.Contains(phase, "document"), strings.Contains(phase, "pend"):
		return "documentos"
	default:
		return "preparacao"
	}
}

func managerStageDays(p Project, history []ProjectHistory) int {
	base := ""
	for _, h := range history {
		if h.ProjectID != p.ID { continue }
		low := strings.ToLower(h.EventType+" "+h.Title)
		if strings.Contains(low, "andamento") || strings.Contains(low, "status") || strings.Contains(low, "fase") {
			base = h.EventDate; break
		}
	}
	if base == "" { base = p.UpdatedAt }
	if base == "" { base = p.CreatedAt }
	t, ok := managerParseTime(base)
	if !ok { return 0 }
	d := int(nowViewer().Sub(t.In(nowViewer().Location())).Hours() / 24)
	if d < 0 { return 0 }
	return d
}

func managerParseTime(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" { return time.Time{}, false }
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, v); err == nil { return t, true }
	}
	return time.Time{}, false
}

func managerDateKey(v string) string {
	if t, ok := managerParseTime(v); ok { return t.In(nowViewer().Location()).Format("2006-01-02") }
	if len(v) >= 10 { return v[:10] }
	return ""
}

func managerDependency(v string) string {
	v = strings.TrimSpace(v)
	if v == "" { return "A definir" }
	low := strings.ToLower(v)
	if strings.HasPrefix(low, "com ") { return v }
	return "Com "+v
}

func managerNormalizeStatus(v, fallback string) string {
	allowed := map[string]string{
		"em preparação":"Em preparação", "em preparacao":"Em preparação",
		"aguardando documentos":"Aguardando documentos", "enviado ao banco":"Enviado ao banco",
		"em análise":"Em análise", "em analise":"Em análise", "aprovado":"Aprovado",
		"contratado":"Contratado", "reprovado":"Reprovado",
	}
	if x, ok := allowed[strings.ToLower(v)]; ok { return x }
	if strings.TrimSpace(v) != "" { return v }
	return fallback
}

func managerBRDate(v string) string {
	if t, ok := managerParseTime(v); ok { return t.In(nowViewer().Location()).Format("02/01/2006") }
	return v
}

func managerDailyShortText(v DailySummaryView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Via Verde — Relatório de %s\n", managerBRDate(v.Date))
	if v.ProjectsMoved == 0 {
		b.WriteString("Sem movimentações registradas neste dia.")
		return b.String()
	}
	fmt.Fprintf(&b, "%d projeto(s) trabalhado(s) · %d ação(ões) registrada(s) · %d documento(s) recebido(s)/dispensado(s) · %d projeto(s) avançaram de etapa.", v.ProjectsMoved, v.TotalActions, v.DocsReceived, v.ProjectsAdvanced)
	if v.Sent+v.Approved+v.Contracted > 0 { fmt.Fprintf(&b, " Marcos: %d enviado(s), %d aprovado(s), %d contratado(s).", v.Sent, v.Approved, v.Contracted) }
	if len(v.PendingNext) > 0 {
		b.WriteString("\n\nPróximos acompanhamentos:")
		limit := len(v.PendingNext); if limit > 4 { limit = 4 }
		for i := 0; i < limit; i++ {
			p := v.PendingNext[i]
			fmt.Fprintf(&b, "\n• %s — %s", defaultManager(p.Producer, p.ProjectTitle), p.Text)
		}
	}
	return b.String()
}

func managerDailyFullText(v DailySummaryView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "RELATÓRIO DO DIA — %s\n", managerBRDate(v.Date))
	fmt.Fprintf(&b, "Via Verde Consultoria Agropecuária\n\nRESUMO\n%d projeto(s) trabalhado(s) · %d ação(ões) registrada(s) · %d atividade(s) concluída(s) · %d documento(s) recebido(s)/dispensado(s) · %d projeto(s) avançaram de etapa.\n", v.ProjectsMoved, v.TotalActions, v.TasksDone, v.DocsReceived, v.ProjectsAdvanced)
	if v.Sent+v.Approved+v.Contracted > 0 { fmt.Fprintf(&b, "Marcos do dia: %d enviado(s) ao banco · %d aprovado(s) · %d contratado(s).\n", v.Sent, v.Approved, v.Contracted) }
	if len(v.ProjectSummaries) == 0 {
		b.WriteString("\nSem movimentações registradas neste dia.")
		return b.String()
	}
	for _, ps := range v.ProjectSummaries {
		fmt.Fprintf(&b, "\n%s — %s", ps.Producer, ps.Project.Title)
		if strings.TrimSpace(ps.Project.Bank) != "" { fmt.Fprintf(&b, " — %s", ps.Project.Bank) }
		b.WriteString("\n")
		for _, action := range ps.Activities { fmt.Fprintf(&b, "✓ %s\n", action) }
		fmt.Fprintf(&b, "Status: %s\n", defaultManager(ps.Project.Status, "A confirmar"))
		fmt.Fprintf(&b, "Fase: %s\n", defaultManager(ps.Project.Phase, "A definir"))
		fmt.Fprintf(&b, "Depende agora: %s\n", ps.Dependency)
		if ps.PendingDocs > 0 { fmt.Fprintf(&b, "Documentos pendentes: %d\n", ps.PendingDocs) }
		fmt.Fprintf(&b, "Próximo passo: %s\n", ps.NextStep)
	}
	if len(v.PendingNext) > 0 {
		b.WriteString("\nPENDÊNCIAS PARA O PRÓXIMO ACOMPANHAMENTO\n")
		for _, p := range v.PendingNext { fmt.Fprintf(&b, "• %s — %s (%s)\n", defaultManager(p.Producer, p.ProjectTitle), p.Text, p.Responsible) }
	}
	return strings.TrimSpace(b.String())
}

func defaultManager(v, fallback string) string {
	if strings.TrimSpace(v) == "" { return fallback }
	return v
}

func sortManagerRows(rows []ManagerProjectRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Signal != rows[j].Signal {
			rank := map[string]int{"red":0,"yellow":1,"green":2,"gray":3}
			return rank[rows[i].Signal] < rank[rows[j].Signal]
		}
		return rows[i].StaleDays > rows[j].StaleDays
	})
}
