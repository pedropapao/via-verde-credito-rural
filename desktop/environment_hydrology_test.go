package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestFinalizeHydrologyFound110(t *testing.T) {
	car := hydrologyTestCAR(t)
	rivers := []carGeoFeature{
		hydrologyTestLine(t, [][]float64{{-46.20, -20.00}, {-45.80, -20.00}}, map[string]any{
			"COCURSODAG": "123",
			"NORIOCOMP":  "Rio Teste",
			"DEDOMINIAL": "Federal",
		}),
		hydrologyTestLine(t, [][]float64{{-47.00, -20.00}, {-46.80, -20.00}}, map[string]any{
			"COCURSODAG": "999",
			"NORIOCOMP":  "Rio Fora",
			"DEDOMINIAL": "Estadual",
		}),
	}
	water := hydrologyQueryResult{
		Complete: true,
		Features: []carGeoFeature{
			hydrologyTestPolygon(t, [][][]float64{{
				{-46.04, -20.04}, {-45.96, -20.04}, {-45.96, -19.96},
				{-46.04, -19.96}, {-46.04, -20.04},
			}}, map[string]any{"NOORIGINAL": "Lago Teste"}),
		},
	}

	got, err := finalizeHydrologyProfile(car, rivers, true, water, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != hydrologyStatusFound || !got.Checked || !got.Available {
		t.Fatalf("estado inesperado: %+v", got)
	}
	if got.RiverReachCount != 1 || got.WaterBodyCount != 1 {
		t.Fatalf("contagens inesperadas: rios=%d massas=%d", got.RiverReachCount, got.WaterBodyCount)
	}
	if got.FederalReachCount != 1 || got.StateReachCount != 0 {
		t.Fatalf("domínio inesperado: federal=%d estadual=%d", got.FederalReachCount, got.StateReachCount)
	}
	if len(got.NamedRivers) != 1 || got.NamedRivers[0] != "Rio Teste" {
		t.Fatalf("rios nomeados inesperados: %#v", got.NamedRivers)
	}
	if got.WaterBodyAreaHa <= 0 {
		t.Fatalf("área de massa d'água deveria ser positiva: %f", got.WaterBodyAreaHa)
	}
	if got.RiverGeoJSON == "" || got.WaterGeoJSON == "" {
		t.Fatal("geometrias encontradas devem ser preservadas para etapas futuras de mapa")
	}
}

func TestFinalizeHydrologyZeroRequiresCompleteSources110(t *testing.T) {
	car := hydrologyTestCAR(t)
	got, err := finalizeHydrologyProfile(car, nil, true, hydrologyQueryResult{Complete: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != hydrologyStatusNone || !got.Available || !got.Checked {
		t.Fatalf("zero confirmado deveria ser sem ocorrência: %+v", got)
	}
	if got.RiverReachCount != 0 || got.WaterBodyCount != 0 {
		t.Fatalf("esperava zero ocorrências: %+v", got)
	}
}

func TestFinalizeHydrologyUnavailableIsNotZero110(t *testing.T) {
	car := hydrologyTestCAR(t)
	got, err := finalizeHydrologyProfile(
		car,
		nil,
		false,
		hydrologyQueryResult{Err: errors.New("context deadline exceeded")},
		[]string{"timeout na parte 3"},
	)
	if err == nil {
		t.Fatal("base incompleta sem ocorrência deveria retornar erro de indisponibilidade")
	}
	if got.Status != hydrologyStatusUnavailable || got.Available || !got.Checked {
		t.Fatalf("estado indisponível inesperado: %+v", got)
	}
	if !strings.Contains(got.Warning, "não significa ausência") {
		t.Fatalf("aviso deveria impedir interpretação de falso zero: %q", got.Warning)
	}
	for _, technical := range []string{"context deadline", "timeout na parte 3", "https://"} {
		if strings.Contains(strings.ToLower(err.Error()), strings.ToLower(technical)) {
			t.Fatalf("erro amigável expôs detalhe técnico %q: %s", technical, err)
		}
	}
}

func TestFinalizeHydrologyPartialOccurrencePreserved110(t *testing.T) {
	car := hydrologyTestCAR(t)
	rivers := []carGeoFeature{
		hydrologyTestLine(t, [][]float64{{-46.20, -20.00}, {-45.80, -20.00}}, map[string]any{
			"NORIOCOMP":  "Córrego Parcial",
			"DEDOMINIAL": "Estadual",
		}),
	}
	got, err := finalizeHydrologyProfile(
		car,
		rivers,
		false,
		hydrologyQueryResult{Err: errors.New("serviço de massa indisponível")},
		[]string{"uma parte da BHO não respondeu"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != hydrologyStatusFound || got.RiverReachCount != 1 {
		t.Fatalf("ocorrência parcial válida deveria ser preservada: %+v", got)
	}
	if !strings.Contains(strings.ToLower(got.Warning), "incompleto") {
		t.Fatalf("resultado parcial precisa avisar que pode estar incompleto: %q", got.Warning)
	}
}

func TestHydrologyLineCrossesCARWithoutVertexInside110(t *testing.T) {
	car := hydrologyTestCAR(t)
	line := hydrologyTestLine(t,
		[][]float64{{-46.20, -20.00}, {-45.80, -20.00}},
		map[string]any{},
	)
	if !hydrologyLineIntersectsCAR(line.Geometry, car) {
		t.Fatal("linha que atravessa o CAR deveria ser identificada mesmo sem vértice interno")
	}

	outside := hydrologyTestLine(t,
		[][]float64{{-47.00, -20.00}, {-46.80, -20.00}},
		map[string]any{},
	)
	if hydrologyLineIntersectsCAR(outside.Geometry, car) {
		t.Fatal("linha externa não deveria intersectar o CAR")
	}
}

func TestEnvironmentalCacheRejectsPartialHydrology110(t *testing.T) {
	base := EnvironmentalIntelligenceResult{
		Profile: EnvironmentalProfile{
			BiomeAvailable: true,
			Biome:          "Mata Atlântica",
			Fire: EnvironmentalFireProfile{
				Status: fireStatusNone, Checked: true, Available: true,
			},
			Hydrology: EnvironmentalHydrologyProfile{
				Status: hydrologyStatusFound, Checked: true, Available: true,
				Warning: "resultado pode estar incompleto",
			},
		},
		MapBiomas: MapBiomasCARSummary{Connected: true, Available: true},
	}
	if environmentalIntelligenceCacheUsable(base) {
		t.Fatal("hidrografia parcial não deve ficar congelada no cache")
	}
	base.Profile.Hydrology.Warning = ""
	if !environmentalIntelligenceCacheUsable(base) {
		t.Fatal("hidrografia completa deveria permitir cache")
	}
}

func hydrologyTestCAR(t *testing.T) string {
	t.Helper()
	f := carGeoFeature{
		Type:       "Feature",
		Properties: map[string]any{},
		Geometry: carGeoJSONGeometry{
			Type: "Polygon",
			Coordinates: json.RawMessage(`[[[-46.1,-20.1],[-45.9,-20.1],[-45.9,-19.9],[-46.1,-19.9],[-46.1,-20.1]]]`),
		},
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func hydrologyTestLine(t *testing.T, coords [][]float64, props map[string]any) carGeoFeature {
	t.Helper()
	raw, err := json.Marshal(coords)
	if err != nil {
		t.Fatal(err)
	}
	return carGeoFeature{
		Type:       "Feature",
		Properties: props,
		Geometry:   carGeoJSONGeometry{Type: "LineString", Coordinates: raw},
	}
}

func hydrologyTestPolygon(t *testing.T, coords [][][]float64, props map[string]any) carGeoFeature {
	t.Helper()
	raw, err := json.Marshal(coords)
	if err != nil {
		t.Fatal(err)
	}
	return carGeoFeature{
		Type:       "Feature",
		Properties: props,
		Geometry:   carGeoJSONGeometry{Type: "Polygon", Coordinates: raw},
	}
}
