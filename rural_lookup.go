package main

import (
	"net/http"
	"strings"
)

type RuralLookupView struct {
	Document string
	Registry string
	SNCR     string
	CIB      string
	CAR      string
	HasAny   bool
}

func trimRuralLookup(v string, max int) string {
	v = strings.TrimSpace(v)
	if len(v) > max {
		v = v[:max]
	}
	return v
}

func (a *App) ruralLookupPage(w http.ResponseWriter, r *http.Request) {
	v := RuralLookupView{
		Document: trimRuralLookup(r.URL.Query().Get("document"), 40),
		Registry: trimRuralLookup(r.URL.Query().Get("registry"), 120),
		SNCR:     trimRuralLookup(r.URL.Query().Get("sncr"), 80),
		CIB:      trimRuralLookup(r.URL.Query().Get("cib"), 80),
		CAR:      trimRuralLookup(r.URL.Query().Get("car"), 90),
	}
	v.HasAny = v.Document != "" || v.Registry != "" || v.SNCR != "" || v.CIB != "" || v.CAR != ""
	a.render(w, r, "documents", ViewData{Title: "Consulta Rural", Data: v})
}
