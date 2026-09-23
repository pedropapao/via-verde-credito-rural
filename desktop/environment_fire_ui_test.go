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

	if strings.Contains(js, "p?.fire?.geojson") || strings.Contains(js, "p.fire.geojson") {
		t.Fatal("Parte 3B não deve adicionar focos ao mapa")
	}
}
