package main

import "testing"

func TestTemplatesDoDossie360Compilam(t *testing.T) {
	a := NewApp(Config{}, nil)
	if len(a.templates) < 10 {
		t.Fatalf("templates carregados insuficientes: %d", len(a.templates))
	}
	for _, name := range []string{"dashboard", "projects", "project_form", "project_detail", "clients", "properties", "daily"} {
		if a.templates[name] == nil {
			t.Fatalf("template %s não carregado", name)
		}
	}
}
