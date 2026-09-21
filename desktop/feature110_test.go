package main

import (
	"strings"
	"testing"
)

func TestBuildAutomaticCompactAreaFiveHa(t *testing.T) {
	car := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.6100,-20.9100],[-46.5950,-20.9100],[-46.5950,-20.9000],[-46.6100,-20.9000],[-46.6100,-20.9100]]]}}`
	geo, err := buildAutomaticCompactArea(car, 5.0, -20.9050, -46.6025)
	if err != nil {
		t.Fatal(err)
	}
	m, err := projectAreaMetrics(geo)
	if err != nil {
		t.Fatal(err)
	}
	if m.AreaHa < 4.90 || m.AreaHa > 5.10 {
		t.Fatalf("área automática fora da tolerância: %.4f ha", m.AreaHa)
	}
	_, inside, _, err := estimateGeometryOverlap(geo, car)
	if err != nil {
		t.Fatal(err)
	}
	if inside < 97 {
		t.Fatalf("gleba deveria estar contida no CAR, estimativa %.2f%%", inside)
	}
}

func TestRouteBoundaryCandidates(t *testing.T) {
	car := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.61,-20.91],[-46.59,-20.91],[-46.59,-20.89],[-46.61,-20.89],[-46.61,-20.91]]]}}`
	c, err := routeBoundaryCandidates(car, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(c) < 4 || len(c) > 12 {
		t.Fatalf("quantidade inesperada de candidatos: %d", len(c))
	}
}

func TestTranslateOSRMManeuver(t *testing.T) {
	if got := translateOSRMManeuver("turn", "left", 0); !strings.Contains(got, "esquerda") {
		t.Fatalf("tradução inesperada: %s", got)
	}
	if got := translateOSRMManeuver("roundabout", "", 3); !strings.Contains(got, "3") {
		t.Fatalf("rotatória sem saída: %s", got)
	}
}

func TestBuildAutomaticRouteText(t *testing.T) {
	x := AccessRoute{
		ReferenceLabel: "Jacuí / MG",
		ReferenceLat: -21.01, ReferenceLon: -46.74,
		EntranceLat: -20.99, EntranceLon: -46.70,
		HeadquartersLat: -20.98, HeadquartersLon: -46.69,
		RouteDistanceKm: 12.4, RouteDurationMin: 24,
	}
	txt := buildAutomaticRouteText(x)
	for _, want := range []string{"Jacuí", "12.40 km", "24 minutos", "entrada"} {
		if !strings.Contains(txt, want) {
			t.Fatalf("texto não contém %q: %s", want, txt)
		}
	}
}
