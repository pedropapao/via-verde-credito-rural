package main

import "testing"

func TestAnalyzeKML(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><kml xmlns="http://www.opengis.net/kml/2.2"><Placemark><Polygon><outerBoundaryIs><LinearRing><coordinates>-46,-20,0 -45.99,-20,0 -45.99,-19.99,0 -46,-19.99,0 -46,-20,0</coordinates></LinearRing></outerBoundaryIs></Polygon></Placemark></kml>`)
	r, err := analyzeKML(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.AreaHa < 100 || r.AreaHa > 130 {
		t.Fatalf("area %.2f", r.AreaHa)
	}
	if r.Points != 4 {
		t.Fatalf("pontos %d", r.Points)
	}
	if r.GeoJSON == "" {
		t.Fatal("geojson vazio")
	}
}

func TestCompareKMLWithCAR(t *testing.T) {
	k := KMLResult{AreaHa: 100, CenterLat: -20, CenterLon: -46}
	c := CARResult{AreaHa: 100.5, CenterLat: -20.0002, CenterLon: -46.0002}
	r := (&App{}).CompareKMLWithCAR(k, c)
	if r.Level != "ok" {
		t.Fatalf("esperava ok: %+v", r)
	}
}
