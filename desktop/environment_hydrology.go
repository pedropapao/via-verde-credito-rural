package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	hydrologyStatusFound       = "ocorrencia_encontrada"
	hydrologyStatusNone        = "sem_ocorrencia"
	hydrologyStatusUnavailable = "base_indisponivel"
	hydrologyStatusNotRun      = "consulta_nao_realizada"

	anaHydrologySource = "ANA/SNIRH — Base Hidrográfica Ottocodificada (BHO) e massas d'água"
	anaHydrologyUnifiedURL = "https://portal1.snirh.gov.br/server/rest/services/dados_abertos/Hidrografia/MapServer/0/query"
	anaWaterBodyURL = "https://portal1.snirh.gov.br/arcgis/rest/services/DADOSABERTOS/Massa_d%C3%A1gua/FeatureServer/0/query"
)

var anaHydrologyPartURLs = []string{
	"https://portal1.snirh.gov.br/server/rest/services/dados_abertos/Hidrografia_Parte_1/FeatureServer/0/query",
	"https://portal1.snirh.gov.br/server/rest/services/dados_abertos/Hidrografia_Parte_2/FeatureServer/0/query",
	"https://portal1.snirh.gov.br/server/rest/services/dados_abertos/Hidrografia_Parte_3/FeatureServer/0/query",
	"https://portal1.snirh.gov.br/server/rest/services/dados_abertos/Hidrografia_Parte_4/FeatureServer/0/query",
	"https://portal1.snirh.gov.br/server/rest/services/dados_abertos/Hidrografia_Parte_5/FeatureServer/0/query",
}

type EnvironmentalHydrologyProfile struct {
	Checked           bool     `json:"checked"`
	Available         bool     `json:"available"`
	Status            string   `json:"status"`
	RiverReachCount   int      `json:"river_reach_count"`
	NamedRivers       []string `json:"named_rivers"`
	FederalReachCount int      `json:"federal_reach_count"`
	StateReachCount   int      `json:"state_reach_count"`
	WaterBodyCount    int      `json:"water_body_count"`
	WaterBodyAreaHa   float64  `json:"water_body_area_ha"`
	RiverGeoJSON      string   `json:"river_geojson"`
	WaterGeoJSON      string   `json:"water_geojson"`
	Source            string   `json:"source"`
	RiverSourceURL    string   `json:"river_source_url"`
	WaterSourceURL    string   `json:"water_source_url"`
	Warning           string   `json:"warning"`
}

type hydrologyQueryResult struct {
	Features []carGeoFeature
	Complete bool
	Err      error
	Source   string
}

type hydrologyArcGISErrorEnvelope struct {
	Error *struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Details []string `json:"details"`
	} `json:"error"`
}

func newEnvironmentalHydrologyProfile() EnvironmentalHydrologyProfile {
	return EnvironmentalHydrologyProfile{
		Status:         hydrologyStatusNotRun,
		Source:         anaHydrologySource,
		RiverSourceURL: strings.TrimSuffix(anaHydrologyUnifiedURL, "/query"),
		WaterSourceURL: strings.TrimSuffix(anaWaterBodyURL, "/query"),
	}
}

func queryHydrologyProfile(ctx context.Context, carRaw string) (EnvironmentalHydrologyProfile, error) {
	out := newEnvironmentalHydrologyProfile()
	if strings.TrimSpace(carRaw) == "" {
		out.Warning = "geometria do CAR indisponível para consulta hidrográfica"
		return out, errors.New(out.Warning)
	}
	if _, err := hydrologyCARPolygons(carRaw); err != nil {
		out.Warning = "geometria do CAR inválida para consulta hidrográfica"
		return out, errors.New(out.Warning)
	}
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		out.Warning = "limites do CAR indisponíveis para consulta hidrográfica"
		return out, errors.New(out.Warning)
	}

	var wg sync.WaitGroup
	partResults := make([]hydrologyQueryResult, len(anaHydrologyPartURLs))
	wg.Add(len(anaHydrologyPartURLs) + 1)
	for i, endpoint := range anaHydrologyPartURLs {
		go func(i int, endpoint string) {
			defer wg.Done()
			features, complete, err := queryHydrologyArcGISPaged(ctx, endpoint, minLon, minLat, maxLon, maxLat,
				"COCURSODAG,COBACIA,NORIOCOMP,DEDOMINIAL", 1000, 10)
			partResults[i] = hydrologyQueryResult{Features: features, Complete: complete, Err: err, Source: endpoint}
		}(i, endpoint)
	}
	var waterResult hydrologyQueryResult
	go func() {
		defer wg.Done()
		features, complete, err := queryHydrologyArcGISPaged(ctx, anaWaterBodyURL, minLon, minLat, maxLon, maxLat,
			"NOORIGINAL", 1000, 10)
		waterResult = hydrologyQueryResult{Features: features, Complete: complete, Err: err, Source: anaWaterBodyURL}
	}()
	wg.Wait()

	riverComplete := true
	var partErrors []string
	var riverCandidates []carGeoFeature
	for _, r := range partResults {
		if r.Err != nil || !r.Complete {
			riverComplete = false
			if r.Err != nil {
				partErrors = append(partErrors, r.Err.Error())
			} else {
				partErrors = append(partErrors, "consulta parcial por limite de registros")
			}
			continue
		}
		riverCandidates = append(riverCandidates, r.Features...)
	}

	// Se alguma parte falhou, tenta a camada unificada uma única vez.
	// Se ela responder, temos cobertura nacional completa e descartamos as partes.
	if !riverComplete && ctx.Err() == nil {
		features, complete, err := queryHydrologyArcGISPaged(ctx, anaHydrologyUnifiedURL,
			minLon, minLat, maxLon, maxLat, "COCURSODAG,COBACIA,NORIOCOMP,DEDOMINIAL", 1000, 10)
		if err == nil && complete {
			riverCandidates = features
			riverComplete = true
			partErrors = nil
		} else if err != nil {
			partErrors = append(partErrors, err.Error())
		}
	}

	return finalizeHydrologyProfile(carRaw, riverCandidates, riverComplete, waterResult, partErrors)
}

func finalizeHydrologyProfile(carRaw string, riverCandidates []carGeoFeature, riverComplete bool,
	waterResult hydrologyQueryResult, partErrors []string) (EnvironmentalHydrologyProfile, error) {
	out := newEnvironmentalHydrologyProfile()
	riverFeatures := make([]carGeoFeature, 0)
	riverSeen := map[string]struct{}{}
	nameSeen := map[string]struct{}{}
	for _, feature := range riverCandidates {
		if !hydrologyLineIntersectsCAR(feature.Geometry, carRaw) {
			continue
		}
		key := hydrologyFeatureKey(feature)
		if _, exists := riverSeen[key]; exists {
			continue
		}
		riverSeen[key] = struct{}{}
		riverFeatures = append(riverFeatures, feature)
		out.RiverReachCount++

		name := hydrologyStringProp(feature.Properties, "NORIOCOMP", "noriocomp")
		if name != "" {
			if _, exists := nameSeen[strings.ToLower(name)]; !exists {
				nameSeen[strings.ToLower(name)] = struct{}{}
				out.NamedRivers = append(out.NamedRivers, name)
			}
		}
		domain := strings.ToLower(hydrologyStringProp(feature.Properties, "DEDOMINIAL", "dedominial"))
		switch {
		case strings.Contains(domain, "federal"):
			out.FederalReachCount++
		case strings.Contains(domain, "estad"):
			out.StateReachCount++
		}
	}
	sort.Strings(out.NamedRivers)
	if len(out.NamedRivers) > 12 {
		out.NamedRivers = out.NamedRivers[:12]
	}
	out.RiverGeoJSON = hydrologyFeatureCollectionJSON(riverFeatures)

	waterComplete := waterResult.Err == nil && waterResult.Complete
	waterFeatures := make([]carGeoFeature, 0)
	if waterComplete {
		for _, feature := range waterResult.Features {
			raw := hydrologyFeatureJSON(feature)
			ha, _, _, err := estimateGeometryOverlap(raw, carRaw)
			if err != nil || ha <= 0.0001 {
				continue
			}
			out.WaterBodyCount++
			out.WaterBodyAreaHa += ha
			waterFeatures = append(waterFeatures, feature)
		}
		out.WaterGeoJSON = hydrologyFeatureCollectionJSON(waterFeatures)
	}

	found := out.RiverReachCount > 0 || out.WaterBodyCount > 0
	fullCoverage := riverComplete && waterComplete
	out.Checked = true

	switch {
	case found:
		out.Available = true
		out.Status = hydrologyStatusFound
		if !fullCoverage {
			out.Warning = "Parte da base ANA/SNIRH ficou indisponível; as ocorrências encontradas foram preservadas, mas o resultado pode estar incompleto."
			hydrologyLogUnavailable(partErrors, waterResult)
		}
		return out, nil
	case fullCoverage:
		out.Available = true
		out.Status = hydrologyStatusNone
		return out, nil
	default:
		out.Available = false
		out.Status = hydrologyStatusUnavailable
		out.Warning = "Base ANA/SNIRH indisponível nesta tentativa. A ausência de resultado não significa ausência de cursos ou massas d'água no imóvel. Tente atualizar a análise mais tarde."
		hydrologyLogUnavailable(partErrors, waterResult)
		return out, errors.New(out.Warning)
	}
}

func hydrologyLogUnavailable(partErrors []string, water hydrologyQueryResult) {
	var details []string
	details = append(details, partErrors...)
	if water.Err != nil {
		details = append(details, "massa d'água: "+water.Err.Error())
	} else if !water.Complete {
		details = append(details, "massa d'água: consulta parcial por limite de registros")
	}
	if len(details) > 0 {
		log.Printf("[ANA/SNIRH] consulta hidrográfica parcial/indisponível: %s", strings.Join(details, " | "))
	}
}

func queryHydrologyArcGISPaged(ctx context.Context, endpoint string, minLon, minLat, maxLon, maxLat float64,
	fields string, pageSize, maxPages int) ([]carGeoFeature, bool, error) {
	if pageSize <= 0 {
		pageSize = 1000
	}
	if maxPages <= 0 {
		maxPages = 1
	}
	out := make([]carGeoFeature, 0)
	for page := 0; page < maxPages; page++ {
		params := url.Values{}
		params.Set("where", "1=1")
		params.Set("geometry", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f", minLon, minLat, maxLon, maxLat))
		params.Set("geometryType", "esriGeometryEnvelope")
		params.Set("inSR", "4326")
		params.Set("spatialRel", "esriSpatialRelIntersects")
		params.Set("outFields", fields)
		params.Set("returnGeometry", "true")
		params.Set("outSR", "4326")
		params.Set("resultOffset", strconv.Itoa(page*pageSize))
		params.Set("resultRecordCount", strconv.Itoa(pageSize))
		params.Set("f", "geojson")

		body, err := fetchEnvironmentalBody(ctx, endpoint+"?"+params.Encode(), "application/geo+json,application/json,*/*")
		if err != nil {
			return out, false, err
		}
		var apiErr hydrologyArcGISErrorEnvelope
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != nil {
			msg := strings.TrimSpace(apiErr.Error.Message)
			if msg == "" {
				msg = fmt.Sprintf("ArcGIS retornou erro %d", apiErr.Error.Code)
			}
			return out, false, errors.New(msg)
		}
		var fc carGeoJSON
		if err := json.Unmarshal(body, &fc); err != nil {
			return out, false, fmt.Errorf("ANA/SNIRH retornou GeoJSON inválido: %w", err)
		}
		out = append(out, fc.Features...)
		if len(fc.Features) < pageSize {
			return out, true, nil
		}
	}
	return out, false, nil
}

func hydrologyStringProp(props map[string]any, keys ...string) string {
	for _, wanted := range keys {
		for key, value := range props {
			if !strings.EqualFold(key, wanted) || value == nil {
				continue
			}
			s := strings.TrimSpace(fmt.Sprint(value))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func hydrologyFeatureKey(feature carGeoFeature) string {
	code := hydrologyStringProp(feature.Properties, "COCURSODAG", "cocursodag")
	geometry, _ := json.Marshal(feature.Geometry)
	if code != "" {
		return code + "|" + string(geometry)
	}
	return string(geometry)
}

func hydrologyFeatureJSON(feature carGeoFeature) string {
	b, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: feature.Properties, Geometry: feature.Geometry})
	return string(b)
}

func hydrologyFeatureCollectionJSON(features []carGeoFeature) string {
	if len(features) == 0 {
		return ""
	}
	b, _ := json.Marshal(carGeoJSON{Type: "FeatureCollection", Features: features})
	return string(b)
}

func hydrologyLineIntersectsCAR(g carGeoJSONGeometry, carRaw string) bool {
	polygons, err := hydrologyCARPolygons(carRaw)
	if err != nil || len(polygons) == 0 {
		return false
	}
	for _, line := range hydrologyGeometryLines(g) {
		for i, point := range line {
			if len(point) < 2 {
				continue
			}
			if hydrologyPointInPolygons(point[0], point[1], polygons) {
				return true
			}
			if i == 0 || len(line[i-1]) < 2 {
				continue
			}
			a, b := line[i-1], point
			for _, polygon := range polygons {
				for _, ring := range polygon {
					for j := 1; j < len(ring); j++ {
						if hydrologySegmentsIntersect(a[0], a[1], b[0], b[1],
							ring[j-1][0], ring[j-1][1], ring[j][0], ring[j][1]) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func hydrologyGeometryLines(g carGeoJSONGeometry) [][][]float64 {
	switch strings.ToLower(strings.TrimSpace(g.Type)) {
	case "linestring":
		var line [][]float64
		if json.Unmarshal(g.Coordinates, &line) == nil {
			return [][][]float64{line}
		}
	case "multilinestring":
		var lines [][][]float64
		if json.Unmarshal(g.Coordinates, &lines) == nil {
			return lines
		}
	}
	return nil
}

func hydrologyCARPolygons(raw string) ([][][][]float64, error) {
	var feature carGeoFeature
	if json.Unmarshal([]byte(raw), &feature) == nil && strings.TrimSpace(feature.Geometry.Type) != "" {
		return hydrologyGeometryPolygons(feature.Geometry)
	}
	var fc carGeoJSON
	if json.Unmarshal([]byte(raw), &fc) == nil {
		var out [][][][]float64
		for _, item := range fc.Features {
			polys, err := hydrologyGeometryPolygons(item.Geometry)
			if err == nil {
				out = append(out, polys...)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	return nil, errors.New("GeoJSON do CAR sem polígono utilizável")
}

func hydrologyGeometryPolygons(g carGeoJSONGeometry) ([][][][]float64, error) {
	switch strings.ToLower(strings.TrimSpace(g.Type)) {
	case "polygon":
		var polygon [][][]float64
		if err := json.Unmarshal(g.Coordinates, &polygon); err != nil || len(polygon) == 0 {
			return nil, errors.New("Polygon inválido")
		}
		return [][][][]float64{polygon}, nil
	case "multipolygon":
		var polygons [][][][]float64
		if err := json.Unmarshal(g.Coordinates, &polygons); err != nil || len(polygons) == 0 {
			return nil, errors.New("MultiPolygon inválido")
		}
		return polygons, nil
	default:
		return nil, fmt.Errorf("geometria %s não é poligonal", g.Type)
	}
}

func hydrologyPointInPolygons(lon, lat float64, polygons [][][][]float64) bool {
	for _, polygon := range polygons {
		if len(polygon) == 0 || !hydrologyPointInRing(lon, lat, polygon[0]) {
			continue
		}
		inHole := false
		for i := 1; i < len(polygon); i++ {
			if hydrologyPointInRing(lon, lat, polygon[i]) {
				inHole = true
				break
			}
		}
		if !inHole {
			return true
		}
	}
	return false
}

func hydrologyPointInRing(x, y float64, ring [][]float64) bool {
	inside := false
	if len(ring) < 3 {
		return false
	}
	j := len(ring) - 1
	for i := 0; i < len(ring); i++ {
		if len(ring[i]) < 2 || len(ring[j]) < 2 {
			j = i
			continue
		}
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]
		if ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}

func hydrologySegmentsIntersect(ax, ay, bx, by, cx, cy, dx, dy float64) bool {
	const eps = 1e-12
	orient := func(px, py, qx, qy, rx, ry float64) float64 {
		return (qx-px)*(ry-py) - (qy-py)*(rx-px)
	}
	onSegment := func(px, py, qx, qy, rx, ry float64) bool {
		return qx >= minHydrology(px, rx)-eps && qx <= maxHydrology(px, rx)+eps &&
			qy >= minHydrology(py, ry)-eps && qy <= maxHydrology(py, ry)+eps
	}
	o1 := orient(ax, ay, bx, by, cx, cy)
	o2 := orient(ax, ay, bx, by, dx, dy)
	o3 := orient(cx, cy, dx, dy, ax, ay)
	o4 := orient(cx, cy, dx, dy, bx, by)
	if ((o1 > eps && o2 < -eps) || (o1 < -eps && o2 > eps)) &&
		((o3 > eps && o4 < -eps) || (o3 < -eps && o4 > eps)) {
		return true
	}
	if absHydrology(o1) <= eps && onSegment(ax, ay, cx, cy, bx, by) {
		return true
	}
	if absHydrology(o2) <= eps && onSegment(ax, ay, dx, dy, bx, by) {
		return true
	}
	if absHydrology(o3) <= eps && onSegment(cx, cy, ax, ay, dx, dy) {
		return true
	}
	return absHydrology(o4) <= eps && onSegment(cx, cy, bx, by, dx, dy)
}

func minHydrology(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxHydrology(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func absHydrology(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
