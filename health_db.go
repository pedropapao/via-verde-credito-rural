package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func databaseStatus(err error) string {
	if err == nil {
		return "ok"
	}
	s := strings.ToLower(err.Error())
	switch {
	case strings.Contains(s, "status 401"), strings.Contains(s, "status 403"), strings.Contains(s, "invalid api key"), strings.Contains(s, "invalid jwt"):
		return "auth_error"
	case strings.Contains(s, "status 404"):
		return "api_not_found"
	case strings.Contains(s, "timeout"), strings.Contains(s, "deadline exceeded"):
		return "timeout"
	case strings.Contains(s, "status 429"):
		return "rate_limited"
	default:
		return "unavailable"
	}
}

func (a *App) healthDB(w http.ResponseWriter, r *http.Request) {
	result := map[string]any{
		"ok":                  true,
		"version":             AppVersion,
		"database_configured": a.sb != nil,
		"database_ok":         false,
		"database_status":     "not_configured",
	}

	if a.sb != nil {
		var rows []struct {
			ID string `json:"id"`
		}
		err := a.sb.Select(r.Context(), "users", "select=id&limit=1", &rows)
		result["database_ok"] = err == nil
		result["database_status"] = databaseStatus(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(result)
}
