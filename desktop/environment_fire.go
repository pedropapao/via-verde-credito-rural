package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	inpeFireWFSURL        = "https://terrabrasilis.dpi.inpe.br/queimadas/geoserver/ows"
	fireStatusFound       = "ocorrencia_encontrada"
	fireStatusNone        = "sem_ocorrencia"
	fireStatusUnavailable = "base_indisponivel"
	fireStatusNotRun      = "consulta_nao_realizada"
)

type EnvironmentalFireProfile struct {
	Checked        bool     `json:"checked"`
	Available      bool     `json:"available"`
	Status         string   `json:"status"`
	FeatureCount   int      `json:"feature_count"`
	LastDetectedAt string   `json:"last_detected_at"`
	Satellites     []string `json:"satellites"`
	MaxRisk        float64  `json:"max_risk"`
	MaxFRP         float64  `json:"max_frp"`
	SourceLayer    string   `json:"source_layer"`
	SourceURL      string   `json:"source_url"`
	WindowLabel    string   `json:"window_label"`
	GeoJSON        string   `json:"geojson"`
	Warning        string   `json:"warning"`
}

type fireWFSCapabilities struct {
	FeatureTypes []struct {
		Name string `xml:"Name"`
	} `xml:"FeatureTypeList>FeatureType"`
}

func newEnvironmentalFireProfile() EnvironmentalFireProfile {
	return EnvironmentalFireProfile{
		Status:      fireStatusNotRun,
		SourceURL:   inpeFireWFSURL,
		WindowLabel: "focos ativos/recentes retornados pela camada pública do Programa Queimadas/INPE",
	}
}

func queryFireProfile(ctx context.Context, carRaw string) (EnvironmentalFireProfile, error) {
	out := newEnvironmentalFireProfile()
	if strings.TrimSpace(carRaw) == "" {
		out.Warning = "geometria do CAR indisponível"
		return out, errors.New(out.Warning)
	}
	if _, _, _, _, ok := geoJSONBounds(carRaw); !ok {
		out.Warning = "geometria do CAR inválida para consulta de focos"
		return out, errors.New(out.Warning)
	}

	layers, err := getINPEFireWFSLayers(ctx)
	if err != nil {
		out.Status = fireStatusUnavailable
		out.Warning = err.Error()
		return out, err
	}
	layer := chooseINPEFireLayer(layers)
	if layer == "" {
		out.Status = fireStatusUnavailable
		out.Warning = "camada pública de focos ativos não localizada no WFS do INPE"
		return out, errors.New(out.Warning)
	}
	out.SourceLayer = layer
	out.WindowLabel = fireWindowLabel(layer)

	fc, err := queryINPEFireFeatures(ctx, layer, carRaw)
	if err != nil {
		out.Status = fireStatusUnavailable
		out.Warning = err.Error()
		return out, err
	}
	return summarizeINPEFireFeatures(carRaw, layer, fc)
}

func getINPEFireWFSLayers(ctx context.Context) ([]string, error) {
	q := url.Values{}
	q.Set("service", "WFS")
	q.Set("version", "2.0.0")
	q.Set("request", "GetCapabilities")
	body, err := fetchEnvironmentalBody(ctx, inpeFireWFSURL+"?"+q.Encode(), "application/xml,text/xml,*/*")
	if err != nil {
		return nil, fmt.Errorf("INPE Queimadas/WFS indisponível: %w", err)
	}
	var caps fireWFSCapabilities
	if err := xml.Unmarshal(body, &caps); err != nil {
		return nil, fmt.Errorf("INPE Queimadas retornou capabilities inválido: %w", err)
	}
	out := make([]string, 0, len(caps.FeatureTypes))
	for _, ft := range caps.FeatureTypes {
		if name := strings.TrimSpace(ft.Name); name != "" {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("INPE Queimadas não retornou camadas WFS")
	}
	return out, nil
}

func chooseINPEFireLayer(layers []string) string {
	best, bestScore := "", -1
	for _, layer := range layers {
		raw := strings.TrimSpace(layer)
		if raw == "" {
			continue
		}
		s := strings.ToLower(raw)
		if strings.Contains(s, "estatistic") ||
			strings.Contains(s, "nfocos") ||
			strings.Contains(s, "municip") ||
			strings.Contains(s, "estado") ||
			strings.Contains(s, "paises") ||
			strings.Contains(s, "pais_") ||
			strings.Contains(s, "grid") ||
			strings.Contains(s, "grd") ||
			strings.Contains(s, "previsao") ||
			strings.Contains(s, "risco") {
			continue
		}
		score := 0
		switch {
		case strings.Contains(s, "focos_48h"):
			score += 120
		case strings.Contains(s, "focos_24h"):
			score += 110
		case strings.Contains(s, "active-fire") || strings.Contains(s, "active_fire"):
			score += 100
		case strings.Contains(s, "focos"):
			score += 60
		case strings.Contains(s, "fire"):
			score += 50
		default:
			continue
		}
		if strings.Contains(s, "48h") {
			score += 20
		}
		if strings.Contains(s, "24h") {
			score += 15
		}
		if strings.Contains(s, "ponto") || strings.Contains(s, "point") {
			score += 10
		}
		if score > bestScore {
			best, bestScore = raw, score
		}
	}
	return best
}

func fireWindowLabel(layer string) string {
	s := strings.ToLower(layer)
	switch {
	case strings.Contains(s, "48h"):
		return "focos de fogo ativo das últimas 48 horas retornados pelo Programa Queimadas/INPE"
	case strings.Contains(s, "24h"):
		return "focos de fogo ativo das últimas 24 horas retornados pelo Programa Queimadas/INPE"
	default:
		return "focos ativos/recentes retornados pela camada pública do Programa Queimadas/INPE"
	}
}

func queryINPEFireFeatures(ctx context.Context, layer, carRaw string) (carGeoJSON, error) {
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		return carGeoJSON{}, errors.New("limites do CAR indisponíveis")
	}
	q := url.Values{}
	q.Set("service", "WFS")
	q.Set("version", "2.0.0")
	q.Set("request", "GetFeature")
	q.Set("typeNames", layer)
	q.Set("outputFormat", "application/json")
	q.Set("srsName", "EPSG:4326")
	q.Set("count", "5000")
	q.Set("bbox", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f,EPSG:4326", minLon, minLat, maxLon, maxLat))
	body, err := fetchEnvironmentalBody(ctx, inpeFireWFSURL+"?"+q.Encode(), "application/json,application/geo+json,*/*")
	if err != nil {
		return carGeoJSON{}, fmt.Errorf("INPE Queimadas/WFS: %w", err)
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return carGeoJSON{}, fmt.Errorf("INPE Queimadas retornou GeoJSON inválido: %w", err)
	}
	return fc, nil
}

func summarizeINPEFireFeatures(carRaw, layer string, fc carGeoJSON) (EnvironmentalFireProfile, error) {
	out := newEnvironmentalFireProfile()
	out.Checked = true
	out.Available = true
	out.SourceLayer = layer
	out.WindowLabel = fireWindowLabel(layer)

	lat0 := sharedLatitude(carRaw)
	mp, err := geoJSONToPlanar(carRaw, lat0)
	if err != nil || len(mp) == 0 {
		out.Checked = false
		out.Available = false
		out.Status = fireStatusNotRun
		out.Warning = "geometria do CAR não pôde ser preparada para cruzamento dos focos"
		if err != nil {
			return out, fmt.Errorf("%s: %w", out.Warning, err)
		}
		return out, errors.New(out.Warning)
	}

	satelliteSet := map[string]struct{}{}
	kept := make([]carGeoFeature, 0)
	var latest time.Time
	for _, f := range fc.Features {
		lat, lon, ok := fireGeometryPoint(f.Geometry)
		if !ok || !pointInMultiPolygon(fireProjectGeoPoint(lon, lat, lat0), mp) {
			continue
		}
		kept = append(kept, f)
		if t := fireFeatureDate(f.Properties); !t.IsZero() && t.After(latest) {
			latest = t
		}
		if sat := firePropertyString(f.Properties, "satelite", "satellite", "sensor"); sat != "" {
			satelliteSet[sat] = struct{}{}
		}
		out.MaxRisk = math.Max(out.MaxRisk, firePropertyFloat(f.Properties, "riscofogo", "risco_fogo", "risk", "fire_risk"))
		out.MaxFRP = math.Max(out.MaxFRP, firePropertyFloat(f.Properties, "frp", "FRP"))
	}
	out.FeatureCount = len(kept)
	out.GeoJSON = fireFeatureCollectionJSON(kept)
	if !latest.IsZero() {
		out.LastDetectedAt = latest.UTC().Format(time.RFC3339)
	}
	for sat := range satelliteSet {
		out.Satellites = append(out.Satellites, sat)
	}
	sort.Strings(out.Satellites)
	if out.FeatureCount > 0 {
		out.Status = fireStatusFound
	} else {
		out.Status = fireStatusNone
	}
	return out, nil
}

func fireGeometryPoint(g carGeoJSONGeometry) (lat, lon float64, ok bool) {
	switch strings.ToLower(strings.TrimSpace(g.Type)) {
	case "point":
		var p []float64
		if json.Unmarshal(g.Coordinates, &p) == nil && len(p) >= 2 {
			return p[1], p[0], true
		}
	case "multipoint":
		var points [][]float64
		if json.Unmarshal(g.Coordinates, &points) == nil && len(points) > 0 && len(points[0]) >= 2 {
			return points[0][1], points[0][0], true
		}
	}
	return 0, 0, false
}

func fireProjectGeoPoint(lon, lat, lat0 float64) planarPoint {
	const earthR = 6371008.8
	cos0 := math.Cos(lat0 * math.Pi / 180)
	return planarPoint{
		X: earthR * lon * math.Pi / 180 * cos0,
		Y: earthR * lat * math.Pi / 180,
	}
}

func firePropertyString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := anyString(m, key); v != "" {
			return v
		}
	}
	return ""
}

func firePropertyFloat(m map[string]any, keys ...string) float64 {
	for _, key := range keys {
		for k, v := range m {
			if !strings.EqualFold(k, key) || v == nil {
				continue
			}
			switch x := v.(type) {
			case float64:
				return x
			case float32:
				return float64(x)
			case int:
				return float64(x)
			case int64:
				return float64(x)
			case json.Number:
				n, _ := x.Float64()
				return n
			default:
				s := strings.ReplaceAll(strings.TrimSpace(fmt.Sprint(v)), ",", ".")
				n, err := strconv.ParseFloat(s, 64)
				if err == nil {
					return n
				}
			}
		}
	}
	return 0
}

func fireFeatureDate(m map[string]any) time.Time {
	raw := firePropertyString(m,
		"view_date",
		"datahora",
		"data_hora_gmt",
		"data_pas",
		"date",
		"data",
	)
	if raw == "" {
		return time.Time{}
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func fireFeatureCollectionJSON(features []carGeoFeature) string {
	fc := carGeoJSON{Type: "FeatureCollection", Features: features}
	b, err := json.Marshal(fc)
	if err != nil {
		return ""
	}
	return string(b)
}
