package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
)

type RuralClientDetailView struct {
	Client        Client
	Projects      []Project
	Properties    []Property
	ProjectCount  int
	ActiveCount   int
	PropertyCount int
	Archived      bool
	DisplayNotes  string
}

func normalizePropertyUF(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	if len(v) != 2 {
		return ""
	}
	for _, r := range v {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return v
}

func (a *App) managerClientDetailRural(w http.ResponseWriter, r *http.Request) {
	c, ok := a.getManagerClient(r, r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}

	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", eq("client_id", c.ID)+"&"+order("updated_at", true), &projects)
	a.enrichProjects(r, projects)

	var properties []Property
	_ = a.sb.Select(r.Context(), "properties", eq("client_id", c.ID)+"&"+order("name", false), &properties)

	active := 0
	for _, p := range projects {
		if !viewerProjectClosed(p.Status) {
			active++
		}
	}

	a.render(w, r, "client_detail", ViewData{
		Title: c.Name,
		Data: RuralClientDetailView{
			Client:        c,
			Projects:      projects,
			Properties:    properties,
			ProjectCount:  len(projects),
			ActiveCount:   active,
			PropertyCount: len(properties),
			Archived:      clientArchived(c.Notes),
			DisplayNotes:  clientDisplayNotes(c.Notes),
		},
	})
}

func (a *App) managerPropertyCreate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", http.StatusForbidden)
		return
	}
	clientID := r.PathValue("id")
	c, ok := a.getManagerClient(r, clientID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if clientArchived(c.Notes) {
		http.Error(w, "Reative o cliente antes de cadastrar um imóvel.", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Nome do imóvel é obrigatório.", http.StatusBadRequest)
		return
	}
	stateRaw := strings.TrimSpace(r.FormValue("state"))
	state := ""
	if stateRaw != "" {
		state = normalizePropertyUF(stateRaw)
		if state == "" {
			http.Error(w, "Informe a UF com duas letras.", http.StatusBadRequest)
			return
		}
	}

	row := map[string]any{
		"client_id": clientID,
		"name":      name,
		"city":      strings.TrimSpace(r.FormValue("city")),
		"state":     state,
		"area_ha":   parseFloat(r.FormValue("area_ha")),
		"registry":  strings.TrimSpace(r.FormValue("registry")),
		"car":       strings.TrimSpace(r.FormValue("car")),
		"ccir":      strings.TrimSpace(r.FormValue("ccir")),
		"itr":       strings.TrimSpace(r.FormValue("itr")),
		"tenure":    strings.TrimSpace(r.FormValue("tenure")),
		"notes":     strings.TrimSpace(r.FormValue("notes")),
	}
	var out []Property
	if err := a.sb.Insert(r.Context(), "properties", row, &out); err != nil || len(out) == 0 {
		if err != nil {
			log.Printf("criar imóvel cliente=%s: %v", clientID, err)
		}
		http.Error(w, "Não foi possível cadastrar o imóvel.", http.StatusInternalServerError)
		return
	}

	a.audit(r.Context(), userFromContext(r.Context()), "create", "property", out[0].ID, name)
	http.Redirect(w, r, "/clients/"+clientID+"?ok="+url.QueryEscape("Imóvel cadastrado na central documental.")+"#documentos-imovel", http.StatusSeeOther)
}

func (a *App) getClientProperty(r *http.Request, clientID, propertyID string) (Property, bool) {
	var rows []Property
	query := eq("id", propertyID) + "&" + eq("client_id", clientID) + "&limit=1"
	if err := a.sb.Select(r.Context(), "properties", query, &rows); err != nil || len(rows) == 0 {
		return Property{}, false
	}
	return rows[0], true
}

func (a *App) managerPropertyEdit(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", http.StatusForbidden)
		return
	}
	clientID := r.PathValue("id")
	propertyID := r.PathValue("property")
	if _, ok := a.getClientProperty(r, clientID, propertyID); !ok {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Nome do imóvel é obrigatório.", http.StatusBadRequest)
		return
	}
	stateRaw := strings.TrimSpace(r.FormValue("state"))
	state := ""
	if stateRaw != "" {
		state = normalizePropertyUF(stateRaw)
		if state == "" {
			http.Error(w, "Informe a UF com duas letras.", http.StatusBadRequest)
			return
		}
	}

	row := map[string]any{
		"name":     name,
		"city":     strings.TrimSpace(r.FormValue("city")),
		"state":    state,
		"area_ha":  parseFloat(r.FormValue("area_ha")),
		"registry": strings.TrimSpace(r.FormValue("registry")),
		"car":      strings.TrimSpace(r.FormValue("car")),
		"ccir":     strings.TrimSpace(r.FormValue("ccir")),
		"itr":      strings.TrimSpace(r.FormValue("itr")),
		"tenure":   strings.TrimSpace(r.FormValue("tenure")),
		"notes":    strings.TrimSpace(r.FormValue("notes")),
	}
	if err := a.sb.Update(r.Context(), "properties", eq("id", propertyID)+"&"+eq("client_id", clientID), row, nil); err != nil {
		log.Printf("atualizar imóvel cliente=%s imóvel=%s: %v", clientID, propertyID, err)
		http.Error(w, "Não foi possível atualizar o imóvel.", http.StatusInternalServerError)
		return
	}

	a.audit(r.Context(), userFromContext(r.Context()), "update", "property", propertyID, name)
	http.Redirect(w, r, "/clients/"+clientID+"?ok="+url.QueryEscape("Dados documentais do imóvel atualizados.")+"#documentos-imovel", http.StatusSeeOther)
}
