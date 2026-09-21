package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateProjectAreaAlternativesAvulsa(t *testing.T) {
	oldEndpoint := terrainElevationEndpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := len(strings.Split(r.URL.Query().Get("latitude"), ","))
		values := make([]float64, n)
		for i := range values {
			values[i] = 800 + float64(i%7)*3
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"elevation": values})
	}))
	defer server.Close()
	terrainElevationEndpoint = server.URL
	defer func(){ terrainElevationEndpoint = oldEndpoint }()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := &App{dataDir: dir}
	carGeo := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.6200,-20.9200],[-46.5900,-20.9200],[-46.5900,-20.8900],[-46.6200,-20.8900],[-46.6200,-20.9200]]]}}`
	car := CARResult{
		CAR: "MG-0000000-0000.0000.0000.0000.0000.0000.0000.0000",
		Found: true, HasGeometry: true, GeoJSON: carGeo,
		GeometryAreaHa: 900, CenterLat: -20.905, CenterLon: -46.605,
		AutoRoute: AccessRoute{EntranceLat: -20.918, EntranceLon: -46.606},
		Themes: SICARThemesSummary{Themes: map[string]SICARThemeMetric{}},
	}
	app.saveLastCARSession(car)

	r, err := app.GenerateProjectAreaAlternatives(0, "Café", "Plantio", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Candidates) < 2 {
		t.Fatalf("esperava alternativas distintas, obteve %d", len(r.Candidates))
	}
	if r.Candidates[0].Score < r.Candidates[len(r.Candidates)-1].Score {
		t.Fatalf("alternativas não ordenadas: %#v", r.Candidates)
	}
	for _, c := range r.Candidates {
		if c.AreaHa < 4.8 || c.AreaHa > 5.2 {
			t.Fatalf("área fora da tolerância: %.4f", c.AreaHa)
		}
		if c.InsideCARPct < 95 {
			t.Fatalf("alternativa pouco contida no CAR: %.2f", c.InsideCARPct)
		}
		if !c.Terrain.Available {
			t.Fatalf("relevo deveria estar disponível: %+v", c.Terrain)
		}
	}

	chosen, err := app.ChooseProjectAreaAlternative(0, "Café escolhido", "Plantio", r.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	if chosen.KMLPath == "" {
		t.Fatal("opção escolhida não gerou KML temporário")
	}
	cache, err := app.GetLastCARSession()
	if err != nil {
		t.Fatal(err)
	}
	if cache.TemporaryArea == nil || cache.TemporaryArea.Name != "Café escolhido" {
		t.Fatalf("opção não foi preservada na sessão: %+v", cache.TemporaryArea)
	}
}

func TestAreaAlternativeScorePenalizesConflict(t *testing.T) {
	a := AreaAlternative{
		InsideCARPct: 100, CompactnessPct: 80, DistanceToAccessM: 300,
		ExistingOverlapPct: 0, ThemeOverlapPct: map[string]float64{},
	}
	b := a
	b.ExistingOverlapPct = 80
	scoreAreaAlternative(&a, true)
	scoreAreaAlternative(&b, true)
	if a.Score <= b.Score {
		t.Fatalf("conflito com área existente deveria reduzir pontuação: %.2f vs %.2f", a.Score, b.Score)
	}
}
