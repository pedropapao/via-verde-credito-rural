package main

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
)

const clientArchiveMarker = "[[VV_CLIENT_ARCHIVED]]"

type ManagerClientRow struct {
	Client
	ProjectCount int
	ActiveCount  int
	Archived     bool
	DisplayNotes string
}

type ManagerClientsView struct {
	Clients       []ManagerClientRow
	Query         string
	Total         int
	ArchivedCount int
	ShowArchived  bool
}

type ManagerClientDetailView struct {
	Client       Client
	Projects     []Project
	ProjectCount int
	ActiveCount  int
	Archived     bool
	DisplayNotes string
}

type ManagerProjectNewView struct {
	Project Project
	Clients []Client
}

func clientArchived(notes string) bool {
	return strings.Contains(notes, clientArchiveMarker)
}

func clientDisplayNotes(notes string) string {
	return strings.TrimSpace(strings.ReplaceAll(notes, clientArchiveMarker, ""))
}

func clientStoredNotes(notes string, archived bool) string {
	notes = strings.TrimSpace(strings.ReplaceAll(notes, clientArchiveMarker, ""))
	if !archived {
		return notes
	}
	if notes == "" {
		return clientArchiveMarker
	}
	return notes + "\n" + clientArchiveMarker
}

func (a *App) managerClientsList(w http.ResponseWriter, r *http.Request) {
	var clients []Client
	_ = a.sb.Select(r.Context(), "clients", "select=*&"+order("name", false), &clients)

	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", "select=id,client_id,status", &projects)

	counts := map[string]int{}
	active := map[string]int{}
	for _, p := range projects {
		if p.ClientID == "" {
			continue
		}
		counts[p.ClientID]++
		if !viewerProjectClosed(p.Status) {
			active[p.ClientID]++
		}
	}

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	showArchived := r.URL.Query().Get("archived") == "1"
	rows := make([]ManagerClientRow, 0, len(clients))
	activeTotal := 0
	archivedTotal := 0
	for _, c := range clients {
		archived := clientArchived(c.Notes)
		displayNotes := clientDisplayNotes(c.Notes)
		if archived {
			archivedTotal++
		} else {
			activeTotal++
		}
		if archived != showArchived {
			continue
		}
		if q != "" {
			hay := strings.ToLower(strings.Join([]string{c.Name, c.Document, c.Phone, c.Email, displayNotes}, " "))
			if !strings.Contains(hay, q) {
				continue
			}
		}
		rows = append(rows, ManagerClientRow{Client: c, ProjectCount: counts[c.ID], ActiveCount: active[c.ID], Archived: archived, DisplayNotes: displayNotes})
	}
	sort.SliceStable(rows, func(i, j int) bool { return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name) })

	a.render(w, r, "client_manager", ViewData{
		Title: "Clientes",
		Data: ManagerClientsView{Clients: rows, Query: r.URL.Query().Get("q"), Total: activeTotal, ArchivedCount: archivedTotal, ShowArchived: showArchived},
	})
}

func (a *App) getManagerClient(r *http.Request, id string) (Client, bool) {
	var rows []Client
	if err := a.sb.Select(r.Context(), "clients", eq("id", id)+"&limit=1", &rows); err != nil || len(rows) == 0 {
		return Client{}, false
	}
	return rows[0], true
}

func (a *App) managerClientDetail(w http.ResponseWriter, r *http.Request) {
	c, ok := a.getManagerClient(r, r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", eq("client_id", c.ID)+"&"+order("updated_at", true), &projects)
	a.enrichProjects(r, projects)
	active := 0
	for _, p := range projects {
		if !viewerProjectClosed(p.Status) {
			active++
		}
	}
	a.render(w, r, "client_detail", ViewData{
		Title: c.Name,
		Data: ManagerClientDetailView{Client: c, Projects: projects, ProjectCount: len(projects), ActiveCount: active, Archived: clientArchived(c.Notes), DisplayNotes: clientDisplayNotes(c.Notes)},
	})
}

func (a *App) managerClientEdit(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", http.StatusForbidden)
		return
	}
	c, ok := a.getManagerClient(r, r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Nome obrigatório.", http.StatusBadRequest)
		return
	}
	vals := map[string]any{
		"name": name,
		"document": strings.TrimSpace(r.FormValue("document")),
		"phone": strings.TrimSpace(r.FormValue("phone")),
		"email": strings.TrimSpace(r.FormValue("email")),
		"notes": clientStoredNotes(r.FormValue("notes"), clientArchived(c.Notes)),
	}
	if err := a.sb.Update(r.Context(), "clients", eq("id", c.ID), vals, nil); err != nil {
		http.Error(w, "Não foi possível atualizar o cliente.", http.StatusInternalServerError)
		return
	}
	a.audit(r.Context(), userFromContext(r.Context()), "update", "client", c.ID, name)
	http.Redirect(w, r, "/clients/"+c.ID+"?ok="+url.QueryEscape("Cliente atualizado."), http.StatusSeeOther)
}

func (a *App) managerClientArchiveToggle(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", http.StatusForbidden)
		return
	}
	c, ok := a.getManagerClient(r, r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	wasArchived := clientArchived(c.Notes)
	if err := a.sb.Update(r.Context(), "clients", eq("id", c.ID), map[string]any{"notes": clientStoredNotes(clientDisplayNotes(c.Notes), !wasArchived)}, nil); err != nil {
		http.Error(w, "Não foi possível alterar o arquivamento.", http.StatusInternalServerError)
		return
	}
	action := "archive"
	message := "Cliente arquivado. O histórico de projetos foi preservado."
	if wasArchived {
		action = "unarchive"
		message = "Cliente reativado."
	}
	a.audit(r.Context(), userFromContext(r.Context()), action, "client", c.ID, c.Name)
	http.Redirect(w, r, "/clients/"+c.ID+"?ok="+url.QueryEscape(message), http.StatusSeeOther)
}

func (a *App) managerProjectNewGet(w http.ResponseWriter, r *http.Request) {
	p := Project{ClientID: strings.TrimSpace(r.URL.Query().Get("client_id")), Status: "Em preparação"}
	a.managerProjectNewForm(w, r, p, "")
}

func (a *App) managerProjectNewForm(w http.ResponseWriter, r *http.Request, p Project, errMsg string) {
	var all []Client
	_ = a.sb.Select(r.Context(), "clients", "select=*&"+order("name", false), &all)
	clients := make([]Client, 0, len(all))
	selectedExists := false
	for _, c := range all {
		if clientArchived(c.Notes) {
			continue
		}
		clients = append(clients, c)
		if c.ID == p.ClientID {
			selectedExists = true
		}
	}
	if p.ClientID != "" && !selectedExists {
		p.ClientID = ""
	}
	a.render(w, r, "manager_project_new", ViewData{Title: "Novo projeto", Error: errMsg, Data: ManagerProjectNewView{Project: p, Clients: clients}})
}

func (a *App) managerProjectNewPost(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", http.StatusForbidden)
		return
	}
	p := Project{
		ClientID: strings.TrimSpace(r.FormValue("client_id")),
		Title: strings.TrimSpace(r.FormValue("title")),
		Modality: strings.TrimSpace(r.FormValue("modality")),
		Activity: strings.TrimSpace(r.FormValue("activity")),
		Bank: strings.TrimSpace(r.FormValue("bank")),
		FinancedValue: parseFloat(r.FormValue("financed_value")),
		Status: defaultString(strings.TrimSpace(r.FormValue("status")), "Em preparação"),
		Phase: strings.TrimSpace(r.FormValue("phase")),
		Responsible: strings.TrimSpace(r.FormValue("responsible")),
	}
	if err := validateProject(p); err != nil {
		a.managerProjectNewForm(w, r, p, err.Error())
		return
	}
	if c, ok := a.getManagerClient(r, p.ClientID); !ok || clientArchived(c.Notes) {
		a.managerProjectNewForm(w, r, p, "Selecione um cliente ativo.")
		return
	}
	var out []Project
	if err := a.sb.Insert(r.Context(), "projects", p, &out); err != nil || len(out) == 0 {
		a.managerProjectNewForm(w, r, p, "Não foi possível criar o projeto.")
		return
	}
	_ = a.ensureProjectDefaults360(r, out[0])
	_ = a.addHistory360(r, out[0].ID, "Projeto", "Projeto criado", "Projeto criado pelo painel de acompanhamento")
	a.audit(r.Context(), userFromContext(r.Context()), "create", "project", out[0].ID, p.Title)
	http.Redirect(w, r, "/projects/"+out[0].ID+"?ok="+url.QueryEscape("Projeto criado e vinculado ao cliente."), http.StatusSeeOther)
}
