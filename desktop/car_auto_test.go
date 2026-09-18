package main

import (
	"strings"
	"testing"
)

func TestCARResultMateriallyChangedIgnoresRepeatQuery(t *testing.T) {
	raw := testSquare(-46.00, -20.00, -45.99, -19.99)
	oldR := CARResult{
		CAR: "MG-3100000-AAAA.BBBB.CCCC.DDDD.EEEE.FFFF.0000.1111",
		Status: "Ativo", Condition: "Aguardando análise", Municipality: "Teste",
		AreaHa: 50, GeometryAreaHa: 50.1, FiscalModules: 2.3, GeoJSON: raw,
		CheckedAt: "2026-09-18T10:00:00-03:00", AutoKMLPath: "antigo.kml",
	}
	newR := oldR
	newR.CheckedAt = "2026-09-18T14:00:00-03:00"
	newR.AutoKMLPath = "novo.kml"
	if carResultMateriallyChanged(marshalJSON(oldR), newR) {
		t.Fatal("consulta repetida não deve gerar novo snapshot")
	}
	newR.AreaHa = 50.5
	if !carResultMateriallyChanged(marshalJSON(oldR), newR) {
		t.Fatal("mudança de área deve gerar novo snapshot")
	}
}

func TestSafeFilePartForProducer(t *testing.T) {
	got := safeFilePart("José da Conceição / Fazenda Boa Fé")
	if strings.ContainsAny(got, "/\\:") {
		t.Fatalf("nome de arquivo inseguro: %q", got)
	}
	if !strings.Contains(got, "Jose") {
		t.Fatalf("acentos não normalizados: %q", got)
	}
}

func TestGeoJSONBoundsForEnvironmentalScreening(t *testing.T) {
	raw := testSquare(-46.00, -20.00, -45.99, -19.99)
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(raw)
	if !ok {
		t.Fatal("limites não encontrados")
	}
	if minLon != -46.00 || maxLon != -45.99 || minLat != -20.00 || maxLat != -19.99 {
		t.Fatalf("bounds inesperados: %f %f %f %f", minLon, minLat, maxLon, maxLat)
	}
}
