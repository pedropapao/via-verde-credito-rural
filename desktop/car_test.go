package main

import (
	"encoding/json"
	"math"
	"testing"
)

func TestNormalizeCAR(t *testing.T) {
	input := "MG-1234567-ABCD.EF01.2345.6789.ABCD.EF01.2345.6789"
	got, uf, muni, err := normalizeCAR(input)
	if err != nil {
		t.Fatal(err)
	}
	if got != input {
		t.Fatalf("got %q", got)
	}
	if uf != "MG" || muni != "1234567" {
		t.Fatalf("uf/muni %s %s", uf, muni)
	}
}

func TestNormalizeCARRejectsInvalid(t *testing.T) {
	if _, _, _, err := normalizeCAR("MG-123-ABC"); err == nil {
		t.Fatal("esperava erro")
	}
}

func TestCARGeometryMetrics(t *testing.T) {
	coords := `[[[-46.0000,-20.0000],[-45.9900,-20.0000],[-45.9900,-19.9900],[-46.0000,-19.9900],[-46.0000,-20.0000]]]`
	g := carGeoJSONGeometry{Type: "Polygon", Coordinates: json.RawMessage(coords)}
	area := carGeometryAreaHa(g)
	if area < 100 || area > 130 {
		t.Fatalf("area inesperada %.2f", area)
	}
	per := carGeometryPerimeterM(g)
	if per < 4000 || per > 5000 {
		t.Fatalf("perimetro inesperado %.2f", per)
	}
	lat, lon := carGeometryCenter(g)
	if math.Abs(lat-(-19.995)) > 0.001 || math.Abs(lon-(-45.995)) > 0.001 {
		t.Fatalf("centro %.6f %.6f", lat, lon)
	}
}
