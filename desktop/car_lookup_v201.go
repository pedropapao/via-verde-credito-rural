package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// carPublicLookupMeta separates "not found" from transport/service failures.
// The public SICAR GeoServer is useful but occasionally unstable, so callers
// must never translate a timeout or malformed response into "zero result".
type carPublicLookupMeta struct {
	Status    string
	Detail    string
	Version   string
	Endpoint  string
	Responded bool
	Attempts  int
}

func lookupCARPublicDetailed(ctx context.Context, car, uf string) (*carGeoFeature, carPublicLookupMeta, error) {
	// Current live SICAR data publishes cod_imovel without dots. Query the
	// compact form first, then keep the formatted representation as a fallback
	// for compatibility with older deployments.
	codes := []string{strings.ReplaceAll(car, ".", "")}
	if codes[0] != car {
		codes = append(codes, car)
	}

	// WFS 1.0.0 has historically been the least fragile for exact CAR lookup.
	// WFS 2.0/1.1 remain as fallbacks.
	versions := []struct {
		version string
		typeKey string
		limitKey string
		limit string
	}{
		{"1.0.0", "typeName", "maxFeatures", "2"},
		{"2.0.0", "typeNames", "count", "2"},
		{"1.1.0", "typeName", "maxFeatures", "2"},
	}
	endpoints := []string{carWFSURL, carWFSFallback}

	meta := carPublicLookupMeta{Status: "unavailable"}
	var failures []string
	successfulResponses := 0

	for _, code := range codes {
		for _, v := range versions {
			for _, endpoint := range endpoints {
				meta.Attempts++
				params := url.Values{}
				params.Set("service", "WFS")
				params.Set("version", v.version)
				params.Set("request", "GetFeature")
				params.Set(v.typeKey, "sicar:sicar_imoveis_"+carLayerUF(uf))
				params.Set("outputFormat", "application/json")
				params.Set("srsName", "EPSG:4326")
				params.Set(v.limitKey, v.limit)
				params.Set("CQL_FILTER", "cod_imovel='"+strings.ReplaceAll(code, "'", "''")+"'")

				body, err := fetchCARBody(ctx, endpoint+"?"+params.Encode())
				if err != nil {
					failures = append(failures, fmt.Sprintf("%s WFS %s: %v", shortCAREndpoint(endpoint), v.version, err))
					continue
				}
				var fc carGeoJSON
				if err := json.Unmarshal(body, &fc); err != nil {
					failures = append(failures, fmt.Sprintf("%s WFS %s retornou JSON inesperado", shortCAREndpoint(endpoint), v.version))
					continue
				}
				successfulResponses++
				meta.Responded = true
				if len(fc.Features) == 0 {
					continue
				}
				meta.Status = "found"
				meta.Version = v.version
				meta.Endpoint = endpoint
				meta.Detail = fmt.Sprintf("CAR localizado no SICAR público via WFS %s.", v.version)
				return &fc.Features[0], meta, nil
			}
		}
	}

	if successfulResponses > 0 {
		meta.Status = "not_found"
		meta.Detail = fmt.Sprintf("O SICAR público respondeu em %d tentativa(s), mas não retornou este código de CAR.", successfulResponses)
		return nil, meta, nil
	}

	if len(failures) == 0 {
		failures = append(failures, "nenhuma resposta utilizável foi recebida")
	}
	meta.Status = "unavailable"
	meta.Detail = compactLookupErrors(failures)
	return nil, meta, errors.New(meta.Detail)
}

func shortCAREndpoint(v string) string {
	if strings.HasSuffix(strings.TrimRight(v, "/"), "/wfs") {
		return "endpoint /wfs"
	}
	return "endpoint /ows"
}

func compactLookupErrors(in []string) string {
	if len(in) == 0 {
		return "base pública SICAR indisponível"
	}
	seen := map[string]bool{}
	out := make([]string, 0, 3)
	for _, x := range in {
		x = strings.TrimSpace(x)
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
		if len(out) == 3 {
			break
		}
	}
	return "base pública SICAR indisponível nesta tentativa: " + strings.Join(out, " | ")
}

// carLookupFallback returns only a previously saved result for the same CAR.
// It is intentionally not treated as a current public confirmation.
func (a *App) carLookupFallback(propertyID int64, car string) (CARResult, bool) {
	if a == nil {
		return CARResult{}, false
	}
	if propertyID > 0 && a.db != nil {
		if previous, err := a.GetLatestCAR(propertyID); err == nil &&
			strings.EqualFold(strings.TrimSpace(previous.CAR), strings.TrimSpace(car)) &&
			previous.Found && previous.HasGeometry && strings.TrimSpace(previous.GeoJSON) != "" {
			return previous, true
		}
	}
	if cache, err := a.GetLastCARSession(); err == nil &&
		strings.EqualFold(strings.TrimSpace(cache.Result.CAR), strings.TrimSpace(car)) &&
		cache.Result.Found && cache.Result.HasGeometry && strings.TrimSpace(cache.Result.GeoJSON) != "" {
		return cache.Result, true
	}
	return CARResult{}, false
}
