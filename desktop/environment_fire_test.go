package main

import (
	"encoding/json"
	"math"
	"testing"
)

func TestChooseINPEFireLayerPrefersRecentPointLayer191(t *testing.T) {
	layers := []string{
		"estatisticas:nfocos_municipios_br_todosat_48h",
		"focos:focos",
		"focos:focos_48h",
	}
	if got := chooseINPEFireLayer(layers); got != "focos:focos_48h" {
		t.Fatalf("camada escolhida=%q; esperava focos:focos_48h", got)
	}
}

func TestSummarizeINPEFireFeaturesInsideCAR191(t *testing.T) {
	carRaw := fireTestCAR(t)
	fc := carGeoJSON{
		Type: "FeatureCollection",
		Features: []carGeoFeature{
			fireTestPoint(t, -46.0, -20.0, map[string]any{
				"view_date": "2026-09-23T14:00:00Z",
				"satelite":  "AQUA",
				"riscofogo": 0.75,
				"frp":       12.5,
			}),
			fireTestPoint(t, -47.0, -20.0, map[string]any{
				"view_date": "2026-09-23T15:00:00Z",
				"satelite":  "NOAA-20",
			}),
		},
	}

	got, err := summarizeINPEFireFeatures(carRaw, "focos:focos_48h", fc)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Checked || !got.Available {
		t.Fatalf("fonte deveria estar marcada como consultada: %+v", got)
	}
	if got.Status != fireStatusFound {
		t.Fatalf("status=%q; esperava %q", got.Status, fireStatusFound)
	}
	if got.FeatureCount != 1 {
		t.Fatalf("esperava 1 foco dentro do CAR; obteve %d", got.FeatureCount)
	}
	if got.LastDetectedAt != "2026-09-23T14:00:00Z" {
		t.Fatalf("data mais recente inesperada: %q", got.LastDetectedAt)
	}
	if len(got.Satellites) != 1 || got.Satellites[0] != "AQUA" {
		t.Fatalf("satélites inesperados: %+v", got.Satellites)
	}
	if math.Abs(got.MaxRisk-0.75) > 0.0001 || math.Abs(got.MaxFRP-12.5) > 0.0001 {
		t.Fatalf("métricas inesperadas: risco=%f frp=%f", got.MaxRisk, got.MaxFRP)
	}
	if got.GeoJSON == "" {
		t.Fatal("GeoJSON dos focos internos deveria ser preservado para a futura camada do mapa")
	}
}

func TestSummarizeINPEFireFeaturesZeroIsNotUnavailable191(t *testing.T) {
	carRaw := fireTestCAR(t)
	fc := carGeoJSON{
		Type: "FeatureCollection",
		Features: []carGeoFeature{
			fireTestPoint(t, -47.0, -20.0, map[string]any{
				"view_date": "2026-09-23T15:00:00Z",
				"satelite":  "AQUA",
			}),
		},
	}

	got, err := summarizeINPEFireFeatures(carRaw, "focos:focos_48h", fc)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != fireStatusNone {
		t.Fatalf("status=%q; zero focos deve ser %q", got.Status, fireStatusNone)
	}
	if !got.Checked || !got.Available || got.FeatureCount != 0 {
		t.Fatalf("zero focos não pode virar base indisponível: %+v", got)
	}
}

func TestSummarizeINPEFireFeaturesInvalidCARIsNotRun191(t *testing.T) {
	got, err := summarizeINPEFireFeatures(`{"type":"Feature","geometry":null}`, "focos:focos_48h", carGeoJSON{})
	if err == nil {
		t.Fatal("geometria inválida deveria retornar erro")
	}
	if got.Status != fireStatusNotRun || got.Checked || got.Available {
		t.Fatalf("geometria inválida deveria resultar em consulta não realizada: %+v", got)
	}
}

func fireTestCAR(t *testing.T) string {
	t.Helper()
	feature := carGeoFeature{
		Type:       "Feature",
		Properties: map[string]any{},
		Geometry: carGeoJSONGeometry{
			Type: "Polygon",
			Coordinates: json.RawMessage(`[[[-46.1,-20.1],[-45.9,-20.1],[-45.9,-19.9],[-46.1,-19.9],[-46.1,-20.1]]]`),
		},
	}
	b, err := json.Marshal(feature)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fireTestPoint(t *testing.T, lon, lat float64, props map[string]any) carGeoFeature {
	t.Helper()
	coords, err := json.Marshal([]float64{lon, lat})
	if err != nil {
		t.Fatal(err)
	}
	return carGeoFeature{
		Type:       "Feature",
		Properties: props,
		Geometry: carGeoJSONGeometry{
			Type:        "Point",
			Coordinates: coords,
		},
	}
}
