package main

import (
	"net/http"
	"net/url"
	"strings"
)

func (a *App) managerClientCreate(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida.", http.StatusForbidden)
		return
	}
	c := Client{
		Name:     strings.TrimSpace(r.FormValue("name")),
		Document: strings.TrimSpace(r.FormValue("document")),
		Phone:    strings.TrimSpace(r.FormValue("phone")),
		Email:    strings.TrimSpace(r.FormValue("email")),
		Notes:    strings.TrimSpace(r.FormValue("notes")),
	}
	if c.Name == "" {
		http.Error(w, "Nome obrigatório.", http.StatusBadRequest)
		return
	}
	var out []Client
	if err := a.sb.Insert(r.Context(), "clients", c, &out); err != nil || len(out) == 0 {
		http.Error(w, "Não foi possível cadastrar o cliente.", http.StatusInternalServerError)
		return
	}
	a.audit(r.Context(), userFromContext(r.Context()), "create", "client", out[0].ID, c.Name)
	// Abre imediatamente a ficha recém-criada, onde o botão Novo projeto já leva
	// o client_id correto. Isso evita qualquer dúvida sobre qual cliente vincular.
	http.Redirect(w, r, "/clients/"+out[0].ID+"?ok="+url.QueryEscape("Cliente cadastrado. Você já pode criar o primeiro projeto."), http.StatusSeeOther)
}
