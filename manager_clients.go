package main

import (
	"net/http"
	"sort"
	"strings"
)

type ManagerClientRow struct {
	Client
	ProjectCount int
	ActiveCount  int
}

type ManagerClientsView struct {
	Clients []ManagerClientRow
	Query   string
	Total   int
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
	rows := make([]ManagerClientRow, 0, len(clients))
	for _, c := range clients {
		if q != "" {
			hay := strings.ToLower(strings.Join([]string{c.Name, c.Document, c.Phone, c.Email}, " "))
			if !strings.Contains(hay, q) {
				continue
			}
		}
		rows = append(rows, ManagerClientRow{Client: c, ProjectCount: counts[c.ID], ActiveCount: active[c.ID]})
	}
	sort.SliceStable(rows, func(i, j int) bool { return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name) })

	a.render(w, r, "client_manager", ViewData{
		Title: "Clientes",
		Data: ManagerClientsView{Clients: rows, Query: r.URL.Query().Get("q"), Total: len(clients)},
	})
}
