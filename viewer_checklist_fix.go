package main

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// routesManagerReliable mantém todas as rotas atuais e intercepta apenas a
// atualização do checklist documental para usar a versão com confirmação de persistência.
func (a *App) routesManagerReliable() http.Handler {
	base := a.routesManager()
	fixedChecklist := securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.viewerChecklistUpdateReliable)))))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) == 4 && parts[0] == "projects" && parts[2] == "checklist" && parts[1] != "" && parts[3] != "" {
				r.SetPathValue("id", parts[1])
				r.SetPathValue("item", parts[3])
				fixedChecklist.ServeHTTP(w, r)
				return
			}
		}
		base.ServeHTTP(w, r)
	})
}

func canonicalChecklistStatus(v string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "pendente":
		return "Pendente", true
	case "recebido", "recebida":
		return "Recebido", true
	case "dispensado", "dispensada":
		return "Dispensado", true
	default:
		return "", false
	}
}

// viewerChecklistUpdateReliable só informa sucesso depois que o Supabase
// devolve a linha alterada e uma nova leitura confirma o status salvo.
func (a *App) viewerChecklistUpdateReliable(w http.ResponseWriter, r *http.Request) {
	if !a.verifyCSRF(r) {
		http.Error(w, "Sessão inválida. Atualize a página e tente novamente.", http.StatusForbidden)
		return
	}

	projectID := strings.TrimSpace(r.PathValue("id"))
	itemID := strings.TrimSpace(r.PathValue("item"))
	status, ok := canonicalChecklistStatus(r.FormValue("status"))
	if projectID == "" || itemID == "" || !ok {
		http.Error(w, "Dados do checklist inválidos.", http.StatusBadRequest)
		return
	}

	var current []ChecklistItem
	query := "select=*&" + eq("id", itemID) + "&" + eq("project_id", projectID) + "&limit=1"
	if err := a.sb.Select(r.Context(), "project_checklist", query, &current); err != nil {
		http.Error(w, "Não foi possível localizar o documento no checklist.", http.StatusInternalServerError)
		return
	}
	if len(current) == 0 {
		http.Error(w, "Documento não encontrado neste projeto. Recarregue a página.", http.StatusNotFound)
		return
	}

	item := current[0]
	if item.Status == status {
		http.Redirect(w, r, "/projects/"+projectID+"?ok="+url.QueryEscape(item.DocumentName+" já estava marcado como "+status+"."), http.StatusSeeOther)
		return
	}

	var updated []ChecklistItem
	updateQuery := eq("id", itemID) + "&" + eq("project_id", projectID)
	if err := a.sb.Update(r.Context(), "project_checklist", updateQuery, map[string]any{"status": status}, &updated); err != nil {
		http.Error(w, "Não foi possível salvar o status do documento.", http.StatusInternalServerError)
		return
	}

	// PostgREST pode responder sucesso HTTP sem linha alterada quando o filtro não encontra
	// registro. Nesse caso não exibimos um falso 'salvo'.
	if len(updated) == 0 {
		http.Error(w, "O banco não confirmou a alteração. Recarregue a página e tente novamente.", http.StatusConflict)
		return
	}

	var confirmed []ChecklistItem
	if err := a.sb.Select(r.Context(), "project_checklist", query, &confirmed); err != nil || len(confirmed) == 0 || confirmed[0].Status != status {
		http.Error(w, "A alteração não pôde ser confirmada no banco de dados.", http.StatusConflict)
		return
	}

	// Atualiza a data geral do projeto para o painel refletir a movimentação.
	_ = a.sb.Update(r.Context(), "projects", eq("id", projectID), map[string]any{"updated_at": time.Now().UTC().Format(time.RFC3339)}, nil)
	a.viewerHistoryAdd(r, projectID, "Checklist documental", item.DocumentName, "Status alterado de "+defaultString(item.Status, "Pendente")+" para "+status)

	http.Redirect(w, r, "/projects/"+projectID+"?ok="+url.QueryEscape(item.DocumentName+" marcado como "+status+" e salvo."), http.StatusSeeOther)
}
