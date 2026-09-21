package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectAreaMetrics(t *testing.T) {
	raw := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.60,-20.90],[-46.59,-20.90],[-46.59,-20.89],[-46.60,-20.89],[-46.60,-20.90]]]}}`
	x, err := projectAreaMetrics(raw)
	if err != nil {
		t.Fatal(err)
	}
	if x.AreaHa <= 0 || x.PerimeterM <= 0 {
		t.Fatalf("métricas inválidas: area=%f perimetro=%f", x.AreaHa, x.PerimeterM)
	}
	if x.CenterLat == 0 || x.CenterLon == 0 {
		t.Fatalf("centro não calculado: %+v", x)
	}
}

func TestBuildAccessRouteText(t *testing.T) {
	x := AccessRoute{
		ReferenceLabel: "Praça Central",
		ReferenceLat: -20.916, ReferenceLon: -46.991,
		EntranceLat: -20.920, EntranceLon: -46.980,
		HeadquartersLat: -20.925, HeadquartersLon: -46.975,
	}
	text := buildAccessRouteText(x)
	for _, want := range []string{"Praça Central", "entrada da propriedade", "sede/ponto principal"} {
		if !strings.Contains(text, want) {
			t.Fatalf("roteiro não contém %q: %s", want, text)
		}
	}
}

func TestBuildConferenceSummary(t *testing.T) {
	r := CARResult{
		Found: true, HasGeometry: true, Status: "Ativo",
		Checks: []QualityCheck{{Level: "ok", Title: "Situação cadastral", Detail: "Ativo"}},
		Environment: EnvironmentalSummary{
			IBAMAChecked: true, FUNAIChecked: true, ICMBioChecked: true, MCRChecked: true,
		},
		Themes: SICARThemesSummary{Complete: false, Themes: map[string]SICARThemeMetric{}},
	}
	c := buildConferenceSummary(r)
	if c.OKCount < 5 {
		t.Fatalf("esperava fontes consultadas como OK: %+v", c)
	}
	if c.UnavailableCount == 0 {
		t.Fatalf("esperava Temas SICAR como indisponível: %+v", c)
	}
	if c.Status != "incomplete" {
		t.Fatalf("status inesperado: %s", c.Status)
	}
}

func TestCompareCARCoreRaw(t *testing.T) {
	old := marshalJSON(CARResult{Status: "Ativo", AreaHa: 10, Municipality: "Jacuí"})
	now := CARResult{Status: "Ativo", AreaHa: 11, Municipality: "Jacuí"}
	changes := compareCARCoreRaw(old, now)
	if len(changes) != 1 || !strings.Contains(changes[0], "Área") {
		t.Fatalf("mudanças inesperadas: %#v", changes)
	}
}

func TestValidSICARThemeZip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tema.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("tema.shp")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("shp"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := validSICARThemeZip(path); err != nil {
		t.Fatal(err)
	}
}
