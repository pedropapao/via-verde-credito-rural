package main

import (
	"os"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestArcGISPaginationReadsAllPages195(t *testing.T) {
	base := url.Values{"where": {"1=1"}}
	fetch := func(ctx context.Context, target, accept string) ([]byte, error) {
		u, err := url.Parse(target)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		if q.Get("returnCountOnly") == "true" {
			return []byte("{\\\"count\\\":450}"), nil
		}
		offset, _ := strconv.Atoi(q.Get("resultOffset"))
		limit, _ := strconv.Atoi(q.Get("resultRecordCount"))
		end := offset + limit
		if end > 450 {
			end = 450
		}
		features := make([]carGeoFeature, 0, end-offset)
		for i := offset; i < end; i++ {
			features = append(features, paginationTestFeature(i))
		}
		return paginationFeatureCollection(features)
	}

	count, err := queryArcGISCount(context.Background(), "https://example.test/query", base, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if count != 450 {
		t.Fatalf("contagem inesperada: %d", count)
	}
	features, err := queryArcGISGeoJSONPages(context.Background(), "https://example.test/query", base, "id", 200, count, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if len(features) != 450 {
		t.Fatalf("esperava 450 registros, recebeu %d", len(features))
	}
	if paginationFeatureID(features[0]) != 0 || paginationFeatureID(features[449]) != 449 {
		t.Fatalf("ordem/paginação incorreta: primeiro=%d último=%d", paginationFeatureID(features[0]), paginationFeatureID(features[449]))
	}
}

func TestArcGISPaginationFailureDoesNotBecomePartialZero195(t *testing.T) {
	base := url.Values{"where": {"1=1"}}
	fetch := func(ctx context.Context, target, accept string) ([]byte, error) {
		u, _ := url.Parse(target)
		q := u.Query()
		if q.Get("resultOffset") == "200" {
			return nil, errors.New("segunda página indisponível")
		}
		features := make([]carGeoFeature, 200)
		for i := range features {
			features[i] = paginationTestFeature(i)
		}
		return paginationFeatureCollection(features)
	}
	features, err := queryArcGISGeoJSONPages(context.Background(), "https://example.test/query", base, "id", 200, 350, fetch)
	if err == nil {
		t.Fatalf("consulta parcial não pode ser aceita; recebeu %d registros", len(features))
	}
	if !strings.Contains(err.Error(), "200") {
		t.Fatalf("erro deveria identificar a página interrompida: %v", err)
	}
}

func TestWFSPaginationReadsAllPages195(t *testing.T) {
	base := url.Values{"service": {"WFS"}, "version": {"1.1.0"}}
	fetch := func(ctx context.Context, target string) ([]byte, error) {
		u, err := url.Parse(target)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		offset, _ := strconv.Atoi(q.Get("startIndex"))
		limit, _ := strconv.Atoi(q.Get("maxFeatures"))
		end := offset + limit
		if end > 205 {
			end = 205
		}
		features := make([]carGeoFeature, 0)
		for i := offset; i < end; i++ {
			features = append(features, paginationTestFeature(i))
		}
		return paginationFeatureCollection(features)
	}
	features, err := queryWFSGeoJSONPages(context.Background(), "https://example.test/wfs", base, 100, 10, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if len(features) != 205 {
		t.Fatalf("esperava 205 registros, recebeu %d", len(features))
	}
	if paginationFeatureID(features[204]) != 204 {
		t.Fatalf("último registro incorreto: %d", paginationFeatureID(features[204]))
	}
}

func TestWFSPaginationRejectsRepeatedFullPage195(t *testing.T) {
	base := url.Values{"service": {"WFS"}}
	fetch := func(ctx context.Context, target string) ([]byte, error) {
		features := make([]carGeoFeature, 100)
		for i := range features {
			features[i] = paginationTestFeature(i)
		}
		return paginationFeatureCollection(features)
	}
	features, err := queryWFSGeoJSONPages(context.Background(), "https://example.test/wfs", base, 100, 5, fetch)
	if err == nil {
		t.Fatalf("página repetida não pode ser aceita; recebeu %d registros", len(features))
	}
	if !strings.Contains(strings.ToLower(err.Error()), "repetiu") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestPaginationIsWiredIntoAllFourSources195(t *testing.T) {
	checks := map[string][]string{
		"environment.go": {
			"queryArcGISGeoJSONPages(ctx, endpoint, base, outFields, 200, count, fetchEnvironmentalBody)",
			"queryWFSGeoJSONPages(ctx, funaiWFSURL, params, 100, 50, fetchCARBody)",
			"queryWFSGeoJSONPages(ctx, icmbioWFSURL, params, 100, 50, fetchCARBody)",
		},
		"sigef_public.go": {
			"queryArcGISCount(ctx, endpoint, base, fetchEnvironmentalBody)",
			"queryArcGISGeoJSONPages(",
		},
	}
	for file, wants := range checks {
		b, err := osReadFilePaginationTest(file)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, want := range wants {
			if !strings.Contains(s, want) {
				t.Fatalf("%s não contém paginação esperada %q", file, want)
			}
		}
	}
}

func paginationTestFeature(id int) carGeoFeature {
	coords, _ := json.Marshal([][][]float64{{
		{float64(id), 0}, {float64(id) + 0.1, 0}, {float64(id) + 0.1, 0.1}, {float64(id), 0},
	}})
	return carGeoFeature{
		Type: "Feature",
		Properties: map[string]any{"id": id},
		Geometry: carGeoJSONGeometry{Type: "Polygon", Coordinates: coords},
	}
}

func paginationFeatureCollection(features []carGeoFeature) ([]byte, error) {
	return json.Marshal(carGeoJSON{Type: "FeatureCollection", Features: features})
}

func paginationFeatureID(feature carGeoFeature) int {
	switch v := feature.Properties["id"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return -1
	}
}

func osReadFilePaginationTest(name string) ([]byte, error) {
	return os.ReadFile(name)
}
