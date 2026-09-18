package main

import (
	"strings"
	"testing"
)

func testSquareFeature() string {
	return `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.0,-20.0],[-45.99,-20.0],[-45.99,-19.99],[-46.0,-19.99],[-46.0,-20.0]]]}}`
}

func TestBuildEnvironmentalKMLIncludesMainLayers(t *testing.T) {
	car := CARResult{
		CAR:     "MG-0000000-AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA",
		GeoJSON: testSquareFeature(),
		Themes: SICARThemesSummary{Themes: map[string]SICARThemeMetric{
			"APP": {Code: "APP", Available: true, GeoJSON: testSquareFeature()},
		}},
		Environment: EnvironmentalSummary{
			IBAMAEmbargoCount: 1,
			IBAMAEmbargos: []EmbargoFinding{{
				Number: "123", GeoJSON: testSquareFeature(), OverlapAreaHa: 1.2, OverlapCARPct: 3.4,
			}},
		},
	}
	kml := KMLResult{GeoJSON: testSquareFeature()}
	got := string(buildEnvironmentalKML(car, kml))
	for _, want := range []string{
		"Via Verde CAR - Dossiê Ambiental",
		"Perímetro público do SICAR",
		"KML do cliente",
		"APP declarada no SICAR",
		"Embargo IBAMA",
		"<Polygon>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("KML ambiental não contém %q", want)
		}
	}
}

func TestBuildDossierSourcesTextIncludesOverlapAndWarning(t *testing.T) {
	car := CARResult{
		CAR: "MG-0000000-AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA",
		Environment: EnvironmentalSummary{
			IBAMAEmbargoCount: 1,
			IBAMAEmbargos: []EmbargoFinding{{OverlapAreaHa: 2.5}},
			Warnings: []string{"fonte de teste indisponível"},
		},
	}
	got := string(buildDossierSourcesText(car))
	for _, want := range []string{"Embargos IBAMA: 1", "2.5000 ha", "fonte de teste indisponível", "Sobreposição espacial"} {
		if !strings.Contains(got, want) {
			t.Fatalf("fontes/avisos não contém %q", want)
		}
	}
}
