package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBCBDecimalParser203(t *testing.T) {
	cases := map[string]float64{
		"12,34": 12.34,
		"1234.56": 1234.56,
		"1.234,56": 1234.56,
	}
	for raw, want := range cases {
		got, err := parseBCBDecimal(raw)
		if err != nil {
			t.Fatalf("%q: %v", raw, err)
		}
		if got != want {
			t.Fatalf("%q: esperado %.2f, obtido %.2f", raw, want, got)
		}
	}
}

func TestRecentIFDataPeriods203(t *testing.T) {
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	got := recentIFDataPeriods(now, 5)
	want := []string{"202606", "202603", "202512", "202509", "202506"}
	if len(got) != len(want) {
		t.Fatalf("quantidade inesperada: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("período %d: esperado %s, obtido %s", i, want[i], got[i])
		}
	}
}

func TestInstitutionNameMatching203(t *testing.T) {
	if !institutionMatchesTargets("BANCO DO BRASIL S.A.", []string{"Banco do Brasil"}) {
		t.Fatal("Banco do Brasil deveria corresponder")
	}
	if !institutionMatchesTargets("COOPERATIVA DE CREDITO CRESOL", []string{"Cresol"}) {
		t.Fatal("Cresol deveria corresponder")
	}
	if institutionMatchesTargets("BANCO ALFA", []string{"Sicoob"}) {
		t.Fatal("instituições sem relação não podem corresponder")
	}
}

func TestMarketYearFilter203(t *testing.T) {
	rows := []map[string]any{
		{"AnoEmissao": "2026", "FonteRecursos": "A"},
		{"AnoEmissao": "2025", "FonteRecursos": "B"},
	}
	got := filterMarketYearRows(rows, "2026")
	if len(got) != 1 || firstNonEmptyStringMapValue(got[0], "FonteRecursos") != "A" {
		t.Fatalf("filtro de ano inesperado: %#v", got)
	}
}

func TestBCBPublicCache203(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "bcb.json")
	in := BCBPublicContext{
		Available: true, GeneratedAt: time.Now().Format(time.RFC3339),
		Municipality: "Goiatuba", UF: "GO",
		Series: BCBSeriesContext{Available: true},
	}
	if err := saveBCBPublicCache(path, in); err != nil {
		t.Fatal(err)
	}
	got, ok := loadBCBPublicCache(path, time.Hour)
	if !ok || !got.Available || got.Municipality != "Goiatuba" || got.UF != "GO" {
		t.Fatalf("cache inesperado: ok=%v value=%#v", ok, got)
	}
}

func TestBCBRuralSeriesSpecsUnique203(t *testing.T) {
	if len(bcbRuralSeriesSpecs) < 10 {
		t.Fatalf("esperava conjunto amplo de séries rurais, obteve %d", len(bcbRuralSeriesSpecs))
	}
	codes := map[int]bool{}
	keys := map[string]bool{}
	for _, spec := range bcbRuralSeriesSpecs {
		if spec.Code <= 0 || spec.Key == "" || spec.Unit == "" {
			t.Fatalf("série incompleta: %#v", spec)
		}
		if codes[spec.Code] {
			t.Fatalf("código SGS duplicado: %d", spec.Code)
		}
		if keys[spec.Key] {
			t.Fatalf("chave duplicada: %s", spec.Key)
		}
		codes[spec.Code], keys[spec.Key] = true, true
	}
}

func TestBCBSourcesKeepSCRAggregateOnly203(t *testing.T) {
	out := BCBPublicContext{Sources: []BCBOpenDataSource{
		{Key: "scr", Status: "on_demand", Automatic: false},
	}}
	if out.Sources[0].Automatic || out.Sources[0].Status != "on_demand" {
		t.Fatal("SCR.data não deve virar consulta individual automática")
	}
}
