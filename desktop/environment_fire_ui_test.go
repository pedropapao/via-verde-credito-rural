package main

import (
	"os"
	"strings"
	"testing"
)

func TestFireCardUI191(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/ui180.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)

	required := []string{
		"function fireCard180",
		"Focos de calor",
		"Programa Queimadas/INPE",
		"ocorrencia_encontrada",
		"sem_ocorrencia",
		"base_indisponivel",
		"0 focos encontrados",
		"Base indisponível",
		"Consulta não realizada",
	}
	for _, want := range required {
		if !strings.Contains(js, want) {
			t.Fatalf("ui180.js não contém %q", want)
		}
	}

	requiredMap := []string{
		"profileHasMap180(p)",
		"p?.fire?.geojson",
		"Foco de calor — INPE",
		"Focos de calor — INPE",
		"s180.fireLayer",
	}
	for _, want := range requiredMap {
		if !strings.Contains(js, want) {
			t.Fatalf("Parte 3C não contém integração de mapa %q", want)
		}
	}
}
