package main

import (
	"testing"
)

func TestDetectBCBFields(t *testing.T) {
	sample := map[string]any{
		"AnoEmissao":        2026.0,
		"Municipio":         "ITABIRA",
		"UF":                "MG",
		"Produto":           "BOVINOS",
		"QtdContratos":      14.0,
		"ValorContratado":   1250000.0,
	}
	f := detectBCBFields(sample)
	for key, want := range map[string]string{
		"year":"AnoEmissao",
		"municipality":"Municipio",
		"uf":"UF",
		"product":"Produto",
		"contracts":"QtdContratos",
		"value":"ValorContratado",
	} {
		if f[key] != want {
			t.Fatalf("%s: esperado %s, obteve %s", key, want, f[key])
		}
	}
}

func TestBCBNumberMapValue(t *testing.T) {
	m := map[string]any{"n": 1234.5, "s": "1234,50"}
	if got := numberMapValue(m,"n"); got != 1234.5 {
		t.Fatalf("número JSON inesperado: %f", got)
	}
	if got := numberMapValue(m,"s"); got != 1234.5 {
		t.Fatalf("número textual inesperado: %f", got)
	}
}

func TestMapBiomasAlertStatusDisconnected(t *testing.T) {
	app := &App{}
	s := app.GetMapBiomasAlertStatus()
	if s.Connected {
		t.Fatal("app sem banco/token não pode aparecer conectado")
	}
}
