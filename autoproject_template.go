package main

import (
	"fmt"
	"html/template"
	"strings"
)

func (a *App) installAutoProjectTemplate() {
	funcs := template.FuncMap{
		"money":       formatMoney,
		"dateBR":      dateBR,
		"statusClass": statusClass,
		"roleLabel": func(v string) string {
			if v == "owner" {
				return "Proprietário"
			}
			return "Visualização"
		},
		"bytes":     formatBytes,
		"pct":       func(v float64) string { return strings.ReplaceAll(fmt.Sprintf("%.2f%%", v), ".", ",") },
		"safe":      func(v string) template.HTML { return template.HTML(v) },
		"hasPrefix": strings.HasPrefix,
		"list":      func(v ...string) []string { return v },
	}
	t, err := template.New("base").Funcs(funcs).ParseFS(webFS, "web/templates/base.html", "web/templates/autoproject.html")
	if err != nil {
		panic(err)
	}
	a.templates["autoproject"] = t
}
