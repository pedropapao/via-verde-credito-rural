package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateAutomaticProjectAreaFromTemporarySession(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	app := &App{dataDir: dir}
	carGeo := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.6100,-20.9100],[-46.5950,-20.9100],[-46.5950,-20.9000],[-46.6100,-20.9000],[-46.6100,-20.9100]]]}}`
	app.saveLastCARSession(CARResult{
		CAR: "MG-0000000-0000.0000.0000.0000.0000.0000.0000.0000",
		Found: true, HasGeometry: true, GeoJSON: carGeo,
		GeometryAreaHa: 150, CenterLat: -20.905, CenterLon: -46.6025,
		AutoRoute: AccessRoute{EntranceLat: -20.905, EntranceLon: -46.6025},
	})
	area, err := app.GenerateAutomaticProjectArea(0, "Café", "Plantio", 1)
	if err != nil {
		t.Fatal(err)
	}
	if area.PropertyID != 0 {
		t.Fatalf("gleba temporária não deve exigir imóvel salvo: %+v", area)
	}
	if area.AreaHa < .95 || area.AreaHa > 1.05 {
		t.Fatalf("área inesperada: %.4f", area.AreaHa)
	}
	if area.KMLPath == "" {
		t.Fatal("KML temporário não foi criado")
	}
	cache, err := app.GetLastCARSession()
	if err != nil {
		t.Fatal(err)
	}
	if cache.TemporaryArea == nil || cache.TemporaryArea.Name != "Café" {
		t.Fatalf("gleba temporária não persistiu na sessão: %+v", cache.TemporaryArea)
	}
}
