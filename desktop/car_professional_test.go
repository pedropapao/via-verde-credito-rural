package main

import (
	"strings"
	"testing"
)

func testSquare(lon0, lat0, lon1, lat1 float64) string {
	return ringGeoJSON([]GeoPoint{
		{Lat: lat0, Lon: lon0},
		{Lat: lat0, Lon: lon1},
		{Lat: lat1, Lon: lon1},
		{Lat: lat1, Lon: lon0},
	})
}

func TestProfessionalOverlapIdentical(t *testing.T) {
	raw := testSquare(-46.00, -20.00, -45.99, -19.99)
	area, kpct, cpct, err := estimateGeometryOverlap(raw, raw)
	if err != nil {
		t.Fatal(err)
	}
	if area <= 0 {
		t.Fatalf("interseção inválida: %.4f", area)
	}
	if kpct < 99 || cpct < 99 {
		t.Fatalf("sobreposição idêntica baixa: kml=%.2f car=%.2f", kpct, cpct)
	}
}

func TestProfessionalOverlapPartial(t *testing.T) {
	a := testSquare(-46.00, -20.00, -45.99, -19.99)
	b := testSquare(-45.995, -20.00, -45.985, -19.99)
	_, kpct, cpct, err := estimateGeometryOverlap(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if kpct < 42 || kpct > 58 || cpct < 42 || cpct > 58 {
		t.Fatalf("esperava sobreposição próxima de 50%%: kml=%.2f car=%.2f", kpct, cpct)
	}
}

func TestCompareKMLWithCARProfessionalClassification(t *testing.T) {
	raw := testSquare(-46.00, -20.00, -45.99, -19.99)
	k := KMLResult{AreaHa: 116.0, PerimeterM: 4300, CenterLat: -19.995, CenterLon: -45.995, GeoJSON: raw}
	c := CARResult{AreaHa: 116.4, GeometryAreaHa: 116.0, PerimeterM: 4310, CenterLat: -19.995, CenterLon: -45.995, GeoJSON: raw}
	got := (&App{}).CompareKMLWithCAR(k, c)
	if got.Level != "ok" {
		t.Fatalf("esperava compatível, recebeu %+v", got)
	}
	if got.OverlapMethod == "" || got.KMLInsideCARPct < 99 {
		t.Fatalf("métrica de sobreposição ausente: %+v", got)
	}
}

func TestCARHistoryDetectsRelevantChanges(t *testing.T) {
	oldR := CARResult{Status: "Ativo", Condition: "Aguardando análise", Municipality: "Município A", AreaHa: 50, GeometryAreaHa: 50.1, GeoJSON: "A"}
	newR := CARResult{Status: "Pendente", Condition: "Em análise", Municipality: "Município A", AreaHa: 50.5, GeometryAreaHa: 50.6, GeoJSON: "B"}
	changes := compareCARHistoryJSON(marshalJSON(oldR), marshalJSON(newR))
	joined := strings.Join(changes, " | ")
	for _, want := range []string{"Situação:", "Condição:", "Área:", "Área geométrica:", "Geometria pública alterada"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("mudança %q não detectada: %s", want, joined)
		}
	}
}
