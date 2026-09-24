package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type pagedHTTPFetcher func(context.Context, string, string) ([]byte, error)
type pagedSimpleFetcher func(context.Context, string) ([]byte, error)

type arcGISQueryError struct {
	Code    int
	Message string
	Details []string
}

type arcGISQueryErrorEnvelope struct {
	Error *arcGISQueryError
}

func queryArcGISCount(ctx context.Context, endpoint string, base url.Values, fetch pagedHTTPFetcher) (int, error) {
	if fetch == nil {
		return 0, errors.New("cliente HTTP indisponível")
	}
	params := cloneURLValues(base)
	params.Set("returnCountOnly", "true")
	params.Set("f", "json")
	body, err := fetch(ctx, endpoint+"?"+params.Encode(), "application/json")
	if err != nil {
		return 0, err
	}
	var resp struct {
		Count int
		Error *arcGISQueryError
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("contagem ArcGIS inválida: %w", err)
	}
	if resp.Error != nil {
		return 0, arcGISQueryError(resp.Error.Code, resp.Error.Message, resp.Error.Details)
	}
	if resp.Count < 0 {
		return 0, errors.New("contagem ArcGIS negativa")
	}
	return resp.Count, nil
}

func queryArcGISGeoJSONPages(ctx context.Context, endpoint string, base url.Values, outFields string,
	pageSize, expectedCount int, fetch pagedHTTPFetcher) ([]carGeoFeature, error) {
	if fetch == nil {
		return nil, errors.New("cliente HTTP indisponível")
	}
	if pageSize <= 0 {
		pageSize = 200
	}
	if expectedCount < 0 {
		return nil, errors.New("contagem esperada inválida")
	}
	if expectedCount == 0 {
		return []carGeoFeature{}, nil
	}

	out := make([]carGeoFeature, 0, expectedCount)
	for offset := 0; offset < expectedCount; offset += pageSize {
		params := cloneURLValues(base)
		params.Set("outFields", outFields)
		params.Set("returnGeometry", "true")
		params.Set("outSR", "4326")
		params.Set("resultOffset", strconv.Itoa(offset))
		params.Set("resultRecordCount", strconv.Itoa(pageSize))
		params.Set("f", "geojson")

		body, err := fetch(ctx, endpoint+"?"+params.Encode(), "application/geo+json,application/json")
		if err != nil {
			return nil, fmt.Errorf("página ArcGIS a partir de %d falhou: %w", offset, err)
		}
		fc, err := parseGeoJSONFeaturePage(body)
		if err != nil {
			return nil, fmt.Errorf("página ArcGIS a partir de %d inválida: %w", offset, err)
		}
		if len(fc.Features) == 0 {
			return nil, fmt.Errorf("paginação ArcGIS terminou antes da contagem esperada (%d de %d registros)", len(out), expectedCount)
		}
		out = append(out, fc.Features...)
	}
	if len(out) != expectedCount {
		return nil, fmt.Errorf("paginação ArcGIS inconsistente: recebeu %d de %d registros esperados", len(out), expectedCount)
	}
	return out, nil
}

func queryWFSGeoJSONPages(ctx context.Context, endpoint string, base url.Values, pageSize, maxPages int,
	fetch pagedSimpleFetcher) ([]carGeoFeature, error) {
	if fetch == nil {
		return nil, errors.New("cliente HTTP indisponível")
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	if maxPages <= 0 {
		maxPages = 50
	}

	out := make([]carGeoFeature, 0)
	seen := map[string]struct{}{}
	for page := 0; page < maxPages; page++ {
		offset := page * pageSize
		params := cloneURLValues(base)
		params.Set("startIndex", strconv.Itoa(offset))
		params.Set("maxFeatures", strconv.Itoa(pageSize))

		body, err := fetch(ctx, endpoint+"?"+params.Encode())
		if err != nil {
			return nil, fmt.Errorf("página WFS a partir de %d falhou: %w", offset, err)
		}
		fc, err := parseGeoJSONFeaturePage(body)
		if err != nil {
			return nil, fmt.Errorf("página WFS a partir de %d inválida: %w", offset, err)
		}
		if len(fc.Features) == 0 {
			return out, nil
		}

		newOnPage := 0
		for _, feature := range fc.Features {
			key := geoJSONFeaturePageKey(feature)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, feature)
			newOnPage++
		}
		if len(fc.Features) < pageSize {
			return out, nil
		}
		if newOnPage == 0 {
			return nil, errors.New("servidor WFS repetiu uma página completa; não é seguro classificar a consulta como completa")
		}
	}
	return nil, fmt.Errorf("paginação WFS excedeu o limite de segurança de %d páginas", maxPages)
}

func parseGeoJSONFeaturePage(body []byte) (carGeoJSON, error) {
	var arcErr arcGISQueryErrorEnvelope
	if json.Unmarshal(body, &arcErr) == nil && arcErr.Error != nil {
		return carGeoJSON{}, arcGISQueryError(arcErr.Error.Code, arcErr.Error.Message, arcErr.Error.Details)
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return carGeoJSON{}, err
	}
	if strings.TrimSpace(fc.Type) != "" && !strings.EqualFold(fc.Type, "FeatureCollection") {
		return carGeoJSON{}, fmt.Errorf("GeoJSON inesperado: %s", fc.Type)
	}
	return fc, nil
}

func arcGISQueryError(code int, message string, details []string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "erro ArcGIS"
	}
	if len(details) > 0 {
		message += ": " + strings.Join(details, " | ")
	}
	if code != 0 {
		return fmt.Errorf("%s (código %d)", message, code)
	}
	return errors.New(message)
}

func geoJSONFeaturePageKey(feature carGeoFeature) string {
	b, err := json.Marshal(feature)
	if err != nil {
		return fmt.Sprintf("%v|%s", feature.Properties, string(feature.Geometry.Coordinates))
	}
	return string(b)
}
