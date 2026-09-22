package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func sicorWKTToGeoJSON(wkt string) (string, error) {
	wkt = strings.TrimSpace(wkt)
	if wkt == "" {
		return "", errors.New("WKT vazio")
	}
	upper := strings.ToUpper(wkt)
	switch {
	case strings.HasPrefix(upper, "POLYGON"):
		coords, err := parseWKTPolygonCoordinates(wkt[len("POLYGON"):])
		if err != nil {
			return "", err
		}
		raw, _ := json.Marshal(coords)
		f := carGeoFeature{
			Type: "Feature",
			Properties: map[string]any{"source": "SICOR/BCB - gleba financiada"},
			Geometry: carGeoJSONGeometry{Type: "Polygon", Coordinates: raw},
		}
		b, _ := json.Marshal(f)
		return string(b), nil
	case strings.HasPrefix(upper, "MULTIPOLYGON"):
		polys, err := parseWKTMultiPolygonCoordinates(wkt[len("MULTIPOLYGON"):])
		if err != nil {
			return "", err
		}
		raw, _ := json.Marshal(polys)
		f := carGeoFeature{
			Type: "Feature",
			Properties: map[string]any{"source": "SICOR/BCB - gleba financiada"},
			Geometry: carGeoJSONGeometry{Type: "MultiPolygon", Coordinates: raw},
		}
		b, _ := json.Marshal(f)
		return string(b), nil
	default:
		return "", fmt.Errorf("tipo WKT não suportado: %s", firstWKTWord(wkt))
	}
}

func firstWKTWord(wkt string) string {
	wkt = strings.TrimSpace(wkt)
	if i := strings.IndexAny(wkt, " ("); i >= 0 {
		return wkt[:i]
	}
	return wkt
}

func parseWKTPolygonCoordinates(body string) ([][][]float64, error) {
	body = strings.TrimSpace(body)
	body = trimOuterPair(body)
	if body == "" {
		return nil, errors.New("POLYGON sem coordenadas")
	}
	ringTexts := splitTopLevelGroups(body)
	if len(ringTexts) == 0 {
		ringTexts = []string{body}
	}
	var rings [][][]float64
	for _, ringText := range ringTexts {
		ringText = trimOuterPair(strings.TrimSpace(ringText))
		ring, err := parseWKTRing(ringText)
		if err != nil {
			return nil, err
		}
		rings = append(rings, ring)
	}
	return rings, nil
}

func parseWKTMultiPolygonCoordinates(body string) ([][][][]float64, error) {
	body = strings.TrimSpace(body)
	body = trimOuterPair(body)
	if body == "" {
		return nil, errors.New("MULTIPOLYGON sem coordenadas")
	}
	polyTexts := splitTopLevelGroups(body)
	if len(polyTexts) == 0 {
		return nil, errors.New("MULTIPOLYGON inválido")
	}
	var polys [][][][]float64
	for _, polyText := range polyTexts {
		polyText = trimOuterPair(strings.TrimSpace(polyText))
		ringTexts := splitTopLevelGroups(polyText)
		if len(ringTexts) == 0 {
			ringTexts = []string{polyText}
		}
		var rings [][][]float64
		for _, ringText := range ringTexts {
			ring, err := parseWKTRing(trimOuterPair(strings.TrimSpace(ringText)))
			if err != nil {
				return nil, err
			}
			rings = append(rings, ring)
		}
		polys = append(polys, rings)
	}
	return polys, nil
}

func parseWKTRing(text string) ([][]float64, error) {
	parts := strings.Split(text, ",")
	if len(parts) < 3 {
		return nil, errors.New("anel WKT com poucos pontos")
	}
	out := make([][]float64, 0, len(parts)+1)
	for _, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) < 2 {
			return nil, fmt.Errorf("coordenada WKT inválida: %q", part)
		}
		lon, err1 := strconv.ParseFloat(fields[0], 64)
		lat, err2 := strconv.ParseFloat(fields[1], 64)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("coordenada WKT inválida: %q", part)
		}
		out = append(out, []float64{lon, lat})
	}
	if len(out) >= 3 {
		first := out[0]
		last := out[len(out)-1]
		if first[0] != last[0] || first[1] != last[1] {
			out = append(out, []float64{first[0], first[1]})
		}
	}
	if len(out) < 4 {
		return nil, errors.New("anel WKT inválido")
	}
	return out, nil
}

func trimOuterPair(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return s
	}
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 && i != len(s)-1 {
				return s
			}
		}
	}
	if depth == 0 {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

func splitTopLevelGroups(s string) []string {
	s = strings.TrimSpace(s)
	var out []string
	depth := 0
	start := -1
	for i, r := range s {
		switch r {
		case '(':
			if depth == 0 {
				start = i
			}
			depth++
		case ')':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					out = append(out, s[start:i+1])
					start = -1
				}
			}
		}
	}
	return out
}
