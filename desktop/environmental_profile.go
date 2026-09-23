package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ibgeBiomeWFSURL       = "https://geoservicos.ibge.gov.br/geoserver/CREN/ows"
	ibgeBiomeLayer        = "CREN:lm_bioma_250"
	terraBrasilisBaseURL  = "https://terrabrasilis.dpi.inpe.br/geoserver"
	inpeFireWFSURL        = "https://terrabrasilis.dpi.inpe.br/queimadas/geoserver/ows"
	anaHydroURL           = "https://portal1.snirh.gov.br/server/rest/services/dados_abertos/Hidrografia/MapServer/0/query"
	anaWaterBodyURL       = "https://portal1.snirh.gov.br/arcgis/rest/services/DADOSABERTOS/Massa_d%C3%A1gua/FeatureServer/0/query"
	anaTelemetryURL       = "https://portal1.snirh.gov.br/server/rest/services/dados_abertos/Estacao_Telemetrica/FeatureServer/0/query"
	worldCoverWMSURL      = "https://mapproxy.terrascope.be/mapproxy/service"
	worldCoverLayer       = "esa-worldcover-map-10m-2021-v2_map"
)

type EnvironmentalProfile struct {
	CheckedAt  string                       `json:"checked_at"`
	Terrain    TerrainMetric                `json:"terrain"`
	Biome      BiomeProfile                 `json:"biome"`
	Hydrology  HydrologyProfile             `json:"hydrology"`
	PRODES     DeforestationProfile         `json:"prodes"`
	DETER      DeforestationProfile         `json:"deter"`
	Fire       FireProfile                  `json:"fire"`
	LandCover  LandCoverProfile             `json:"land_cover"`
	Nearby     NearbyEnvironmentalProfile    `json:"nearby"`
	Sources    []EnvironmentalProfileSource `json:"sources"`
	Warnings   []string                     `json:"warnings"`
}

type EnvironmentalProfileSource struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	Available  bool   `json:"available"`
	Applicable bool   `json:"applicable"`
	SourceURL  string `json:"source_url"`
	Detail     string `json:"detail"`
}

type BiomeProfile struct {
	Available  bool            `json:"available"`
	Items      []BiomePresence `json:"items"`
	Dominant   string          `json:"dominant"`
	Source     string          `json:"source"`
	GeoJSON    string          `json:"geojson"`
	Warning    string          `json:"warning"`
}

type BiomePresence struct {
	Name      string  `json:"name"`
	AreaHa    float64 `json:"area_ha"`
	CARPct    float64 `json:"car_pct"`
}

type HydrologyProfile struct {
	Available          bool     `json:"available"`
	RiverReachCount    int      `json:"river_reach_count"`
	NamedRivers        []string `json:"named_rivers"`
	FederalReachCount  int      `json:"federal_reach_count"`
	StateReachCount    int      `json:"state_reach_count"`
	WaterBodyCount     int      `json:"water_body_count"`
	WaterBodyAreaHa    float64  `json:"water_body_area_ha"`
	NearestStationName string   `json:"nearest_station_name"`
	NearestStationKm   float64  `json:"nearest_station_km"`
	RiverGeoJSON       string   `json:"river_geojson"`
	WaterGeoJSON       string   `json:"water_geojson"`
	Source              string   `json:"source"`
	Warning             string   `json:"warning"`
}

type DeforestationProfile struct {
	Available     bool       `json:"available"`
	Applicable    bool       `json:"applicable"`
	Program       string     `json:"program"`
	FeatureCount  int        `json:"feature_count"`
	AreaInCARHa   float64    `json:"area_in_car_ha"`
	LatestYear    int        `json:"latest_year"`
	LatestDate    string     `json:"latest_date"`
	Years         []YearArea `json:"years"`
	SourceLayer   string     `json:"source_layer"`
	SourceURL     string     `json:"source_url"`
	GeoJSON       string     `json:"geojson"`
	Warning       string     `json:"warning"`
}

type YearArea struct {
	Year   int     `json:"year"`
	AreaHa float64 `json:"area_ha"`
	Count  int     `json:"count"`
}

type FireProfile struct {
	Available      bool    `json:"available"`
	FeatureCount   int     `json:"feature_count"`
	LastDetectedAt string  `json:"last_detected_at"`
	MaxRisk        float64 `json:"max_risk"`
	MaxFRP         float64 `json:"max_frp"`
	SourceLayer    string  `json:"source_layer"`
	SourceURL      string  `json:"source_url"`
	WindowLabel    string  `json:"window_label"`
	GeoJSON        string  `json:"geojson"`
	Warning        string  `json:"warning"`
}

type LandCoverProfile struct {
	Available     bool             `json:"available"`
	Approximate   bool             `json:"approximate"`
	ReferenceYear int              `json:"reference_year"`
	SampleCount   int              `json:"sample_count"`
	DominantClass string           `json:"dominant_class"`
	DominantPct   float64          `json:"dominant_pct"`
	Classes       []LandCoverClass `json:"classes"`
	Source        string           `json:"source"`
	Warning       string           `json:"warning"`
}

type LandCoverClass struct {
	Code      int     `json:"code"`
	Label     string  `json:"label"`
	SamplePct float64 `json:"sample_pct"`
	Samples   int     `json:"samples"`
}

type NearbyEnvironmentalProfile struct {
	Available              bool    `json:"available"`
	SearchRadiusKm         float64 `json:"search_radius_km"`
	EmbargoFound           bool    `json:"embargo_found"`
	NearestEmbargoKm       float64 `json:"nearest_embargo_km"`
	NearestEmbargoLabel    string  `json:"nearest_embargo_label"`
	IndigenousFound        bool    `json:"indigenous_found"`
	NearestIndigenousKm    float64 `json:"nearest_indigenous_km"`
	NearestIndigenousLabel string  `json:"nearest_indigenous_label"`
	UCFound                bool    `json:"uc_found"`
	NearestUCKm            float64 `json:"nearest_uc_km"`
	NearestUCLabel         string  `json:"nearest_uc_label"`
	Warning                string  `json:"warning"`
}

type wfsCapabilities struct {
	FeatureTypes []wfsFeatureType `xml:"FeatureTypeList>FeatureType"`
}

type wfsFeatureType struct {
	Name  string `xml:"Name"`
	Title string `xml:"Title"`
}

func buildEnvironmentalProfile(parent context.Context, car CARResult) EnvironmentalProfile {
	out := EnvironmentalProfile{CheckedAt: time.Now().Format(time.RFC3339)}
	if strings.TrimSpace(car.GeoJSON) == "" {
		out.Warnings = append(out.Warnings, "Perfil ambiental: geometria do CAR indisponível.")
		return out
	}
	ctx, cancel := context.WithTimeout(parent, 52*time.Second)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	run := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}

	// Fontes independentes começam imediatamente. Bioma é consultado em
	// paralelo e serve apenas para selecionar os workspaces PRODES/DETER.
	run(func() {
		m, err := terrainMetricForCAR(ctx, car.GeoJSON, firstPositive(car.GeometryAreaHa, car.AreaHa))
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			m.Warning = err.Error()
			out.Warnings = append(out.Warnings, "Relevo: "+err.Error())
		}
		out.Terrain = m
	})
	run(func() {
		h, err := queryHydrologyProfile(ctx, car)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			h.Warning = err.Error()
			out.Warnings = append(out.Warnings, "ANA/Hidrografia: "+err.Error())
		}
		out.Hydrology = h
	})
	run(func() {
		l, err := queryWorldCoverProfile(ctx, car.GeoJSON)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			l.Warning = err.Error()
			out.Warnings = append(out.Warnings, "ESA WorldCover: "+err.Error())
		}
		out.LandCover = l
	})
	run(func() {
		f, err := queryFireProfile(ctx, car.GeoJSON)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			f.Warning = err.Error()
			out.Warnings = append(out.Warnings, "INPE/Queimadas: "+err.Error())
		}
		out.Fire = f
	})
	run(func() {
		n, err := queryNearbyEnvironmental(ctx, car)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			n.Warning = err.Error()
			out.Warnings = append(out.Warnings, "Proximidade territorial: "+err.Error())
		}
		out.Nearby = n
	})

	biome, biomeErr := queryBiomeProfile(ctx, car.GeoJSON, firstPositive(car.GeometryAreaHa, car.AreaHa))
	mu.Lock()
	if biomeErr != nil {
		biome.Warning = biomeErr.Error()
		out.Warnings = append(out.Warnings, "Bioma/IBGE: "+biomeErr.Error())
	}
	out.Biome = biome
	mu.Unlock()

	biomeName := biome.Dominant
	run(func() {
		p, err := queryPRODESProfile(ctx, car.GeoJSON, biomeName)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			p.Warning = err.Error()
			out.Warnings = append(out.Warnings, "INPE/PRODES: "+err.Error())
		}
		out.PRODES = p
	})
	run(func() {
		d, err := queryDETERProfile(ctx, car.GeoJSON, biomeName)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			d.Warning = err.Error()
			out.Warnings = append(out.Warnings, "INPE/DETER: "+err.Error())
		}
		out.DETER = d
	})

	wg.Wait()
	out.Sources = buildEnvironmentalProfileSources(out)
	out.Warnings = uniqueStrings(out.Warnings)
	return out
}

func terrainMetricForCAR(ctx context.Context, raw string, areaHa float64) (TerrainMetric, error) {
	points := terrainSamplePoints(raw, 24)
	if len(points) < 2 {
		return TerrainMetric{}, errors.New("pontos internos insuficientes para amostrar o relevo")
	}
	indexed := make([]indexedTerrainPoint, 0, len(points))
	for _, p := range points {
		indexed = append(indexed, indexedTerrainPoint{Candidate: 0, Point: p})
	}
	elev, err := fetchTerrainElevations(ctx, indexed)
	if err != nil {
		return TerrainMetric{}, err
	}
	metric := calculateTerrainMetric(points, elev, areaHa)
	if !metric.Available {
		if metric.Warning != "" {
			return metric, errors.New(metric.Warning)
		}
		return metric, errors.New("relevo indisponível")
	}
	return metric, nil
}

func queryBiomeProfile(ctx context.Context, carRaw string, carAreaHa float64) (BiomeProfile, error) {
	out := BiomeProfile{Source: "IBGE — Limites dos Biomas 1:250.000"}
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		return out, errors.New("limites do CAR indisponíveis")
	}
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "1.0.0")
	params.Set("request", "GetFeature")
	params.Set("typeName", ibgeBiomeLayer)
	params.Set("outputFormat", "application/json")
	params.Set("srsName", "EPSG:4326")
	params.Set("maxFeatures", "20")
	params.Set("bbox", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f,EPSG:4326", minLon, minLat, maxLon, maxLat))
	body, err := fetchEnvironmentalBody(ctx, ibgeBiomeWFSURL+"?"+params.Encode(), "application/json,application/geo+json")
	if err != nil {
		return out, err
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return out, fmt.Errorf("IBGE retornou GeoJSON inválido: %w", err)
	}
	var kept []carGeoFeature
	for _, f := range fc.Features {
		raw := featureJSON(f)
		ha, _, carPct, overlapErr := estimateGeometryOverlap(raw, carRaw)
		if overlapErr != nil || ha <= 0.0001 {
			continue
		}
		name := carStringProp(f.Properties, "bioma", "nome", "nm_bioma", "nome_bioma", "Bioma", "NOME")
		if name == "" {
			name = "Bioma não identificado"
		}
		if carAreaHa > 0 && carPct <= 0 {
			carPct = math.Min(100, ha/carAreaHa*100)
		}
		out.Items = append(out.Items, BiomePresence{Name: name, AreaHa: ha, CARPct: carPct})
		kept = append(kept, f)
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].AreaHa > out.Items[j].AreaHa })
	if len(out.Items) > 0 {
		out.Dominant = out.Items[0].Name
		out.Available = true
		out.GeoJSON = featureCollectionJSON(kept)
		return out, nil
	}
	return out, errors.New("nenhum limite de bioma interceptou o CAR")
}

func queryHydrologyProfile(ctx context.Context, car CARResult) (HydrologyProfile, error) {
	out := HydrologyProfile{Source: "ANA/SNIRH — Base Hidrográfica Ottocodificada e massas d'água"}
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(car.GeoJSON)
	if !ok {
		return out, errors.New("limites do CAR indisponíveis")
	}
	riverSeen := map[string]bool{}
	nameSeen := map[string]bool{}
	var riverFeatures []carGeoFeature
	var errs []string
	hydroOK := false
	fc, hydroErr := queryArcGISByEnvelope(ctx, anaHydroURL, minLon, minLat, maxLon, maxLat,
		"COCURSODAG,COBACIA,NORIOCOMP,DEDOMINIAL,OBJECTID", 1000)
	if hydroErr != nil {
		errs = append(errs, "hidrografia: "+hydroErr.Error())
	} else {
		hydroOK = true
		for _, f := range fc.Features {
			if !lineGeometryIntersectsCAR(f.Geometry, car.GeoJSON) {
				continue
			}
			key := carStringProp(f.Properties, "COCURSODAG", "cocursodag", "OBJECTID", "objectid")
			if key == "" {
				key = featureGeometryKey(f.Geometry)
			}
			if riverSeen[key] {
				continue
			}
			riverSeen[key] = true
			out.RiverReachCount++
			name := carStringProp(f.Properties, "NORIOCOMP", "noriocomp", "nome", "NOME")
			if name != "" && !nameSeen[name] {
				nameSeen[name] = true
				out.NamedRivers = append(out.NamedRivers, name)
			}
			domain := strings.ToLower(carStringProp(f.Properties, "DEDOMINIAL", "dedominial"))
			if strings.Contains(domain, "federal") {
				out.FederalReachCount++
			} else if strings.Contains(domain, "estad") {
				out.StateReachCount++
			}
			riverFeatures = append(riverFeatures, f)
		}
	}
	sort.Strings(out.NamedRivers)
	if len(out.NamedRivers) > 8 {
		out.NamedRivers = out.NamedRivers[:8]
	}
	out.RiverGeoJSON = featureCollectionJSON(riverFeatures)

	waterFC, waterErr := queryArcGISByEnvelope(ctx, anaWaterBodyURL, minLon, minLat, maxLon, maxLat, "*", 500)
	var waterFeatures []carGeoFeature
	if waterErr == nil {
		for _, f := range waterFC.Features {
			raw := featureJSON(f)
			ha, _, _, overlapErr := estimateGeometryOverlap(raw, car.GeoJSON)
			if overlapErr != nil || ha <= 0.0001 {
				continue
			}
			out.WaterBodyCount++
			out.WaterBodyAreaHa += ha
			waterFeatures = append(waterFeatures, f)
		}
		out.WaterGeoJSON = featureCollectionJSON(waterFeatures)
	} else {
		errs = append(errs, "massas d'água: "+waterErr.Error())
	}

	station, stationKm, stationErr := queryNearestTelemetryStation(ctx, car.CenterLat, car.CenterLon)
	if stationErr == nil {
		out.NearestStationName = station
		out.NearestStationKm = stationKm
	} else {
		errs = append(errs, "estação telemétrica: "+stationErr.Error())
	}

	out.Available = hydroOK || waterErr == nil
	if !out.Available {
		return out, errors.New(strings.Join(errs, " | "))
	}
	if len(errs) > 0 {
		out.Warning = strings.Join(errs, " | ")
	}
	return out, nil
}

func queryArcGISByEnvelope(ctx context.Context, endpoint string, minLon, minLat, maxLon, maxLat float64, fields string, limit int) (carGeoJSON, error) {
	params := url.Values{}
	params.Set("where", "1=1")
	params.Set("geometry", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f", minLon, minLat, maxLon, maxLat))
	params.Set("geometryType", "esriGeometryEnvelope")
	params.Set("inSR", "4326")
	params.Set("spatialRel", "esriSpatialRelIntersects")
	params.Set("outFields", fields)
	params.Set("returnGeometry", "true")
	params.Set("outSR", "4326")
	params.Set("resultRecordCount", strconv.Itoa(limit))
	params.Set("f", "geojson")
	body, err := fetchEnvironmentalBody(ctx, endpoint+"?"+params.Encode(), "application/geo+json,application/json")
	if err != nil {
		return carGeoJSON{}, err
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return carGeoJSON{}, err
	}
	return fc, nil
}

func queryNearestTelemetryStation(ctx context.Context, lat, lon float64) (string, float64, error) {
	if lat == 0 && lon == 0 {
		return "", 0, errors.New("centroide do CAR indisponível")
	}
	span := 0.75
	fc, err := queryArcGISByEnvelope(ctx, anaTelemetryURL, lon-span, lat-span, lon+span, lat+span, "*", 500)
	if err != nil {
		return "", 0, err
	}
	best := math.Inf(1)
	label := ""
	for _, f := range fc.Features {
		pLat, pLon, ok := geometryFirstPoint(f.Geometry)
		if !ok {
			continue
		}
		d := carHaversineM(lat, lon, pLat, pLon) / 1000
		if d >= best {
			continue
		}
		best = d
		label = carStringProp(f.Properties, "Nome", "NOME", "nome", "NomeEstacao", "nomeestacao", "Estacao", "estacao", "Codigo", "codigo")
		if label == "" {
			label = "Estação telemétrica"
		}
	}
	if math.IsInf(best, 1) {
		return "", 0, errors.New("nenhuma estação retornada em raio aproximado de 80 km")
	}
	return label, best, nil
}

func queryPRODESProfile(ctx context.Context, carRaw, biome string) (DeforestationProfile, error) {
	out := DeforestationProfile{Program: "PRODES", Applicable: true}
	workspace := prodesWorkspaceForBiome(biome)
	if workspace == "" {
		out.Applicable = false
		return out, errors.New("bioma não identificado para selecionar o conjunto PRODES")
	}
	endpoint := terraBrasilisBaseURL + "/" + workspace + "/wfs"
	types, err := getWFSFeatureTypes(ctx, endpoint)
	if err != nil {
		return out, err
	}
	layer := chooseWFSLayer(types, []string{"yearly_deforestation", "deforestation", "desmatamento", "increment"}, []string{"municip", "state", "grid", "rate", "mask"})
	if layer == "" {
		return out, errors.New("camada vetorial de desmatamento não localizada no workspace "+workspace)
	}
	fc, err := queryWFSByCARBBox(ctx, endpoint, layer, carRaw, 800, "")
	if err != nil {
		return out, err
	}
	out.Available = true
	out.SourceLayer = layer
	out.SourceURL = endpoint
	var kept []carGeoFeature
	yearMap := map[int]*YearArea{}
	for _, f := range fc.Features {
		raw := featureJSON(f)
		ha, _, _, overlapErr := estimateGeometryOverlap(raw, carRaw)
		if overlapErr != nil || ha <= 0.0001 {
			continue
		}
		out.FeatureCount++
		out.AreaInCARHa += ha
		year := featureYear(f.Properties)
		if year > out.LatestYear {
			out.LatestYear = year
		}
		if year > 0 {
			y := yearMap[year]
			if y == nil {
				y = &YearArea{Year: year}
				yearMap[year] = y
			}
			y.AreaHa += ha
			y.Count++
		}
		kept = append(kept, f)
	}
	for _, y := range yearMap {
		out.Years = append(out.Years, *y)
	}
	sort.Slice(out.Years, func(i, j int) bool { return out.Years[i].Year > out.Years[j].Year })
	out.GeoJSON = featureCollectionJSON(kept)
	return out, nil
}

func queryDETERProfile(ctx context.Context, carRaw, biome string) (DeforestationProfile, error) {
	out := DeforestationProfile{Program: "DETER"}
	key := normalizeBiomeKey(biome)
	workspace := ""
	switch key {
	case "amazonia":
		workspace = "deter-amz"
	case "cerrado":
		workspace = "deter-cerrado-nb"
	case "pantanal":
		workspace = "deter-pantanal"
	default:
		out.Available = true
		out.Applicable = false
		out.Warning = "DETER público não possui cobertura operacional equivalente para este bioma; o PRODES permanece como referência anual."
		return out, nil
	}
	out.Applicable = true
	endpoint := terraBrasilisBaseURL + "/" + workspace + "/wfs"
	types, err := getWFSFeatureTypes(ctx, endpoint)
	if err != nil {
		return out, err
	}
	layer := chooseWFSLayer(types, []string{"deter_public", "deter", "alert"}, []string{"municip", "state", "grid", "mask"})
	if layer == "" {
		return out, errors.New("camada pública DETER não localizada em "+workspace)
	}
	fc, err := queryWFSByCARBBox(ctx, endpoint, layer, carRaw, 800, "")
	if err != nil {
		return out, err
	}
	out.Available = true
	out.SourceLayer = layer
	out.SourceURL = endpoint
	var kept []carGeoFeature
	yearMap := map[int]*YearArea{}
	cut := time.Now().AddDate(-1, 0, 0)
	for _, f := range fc.Features {
		date := featureDate(f.Properties)
		if !date.IsZero() && date.Before(cut) {
			continue
		}
		raw := featureJSON(f)
		ha, _, _, overlapErr := estimateGeometryOverlap(raw, carRaw)
		if overlapErr != nil || ha <= 0.0001 {
			continue
		}
		out.FeatureCount++
		out.AreaInCARHa += ha
		if !date.IsZero() {
			ds := date.Format("2006-01-02")
			if ds > out.LatestDate {
				out.LatestDate = ds
			}
			y := yearMap[date.Year()]
			if y == nil {
				y = &YearArea{Year: date.Year()}
				yearMap[date.Year()] = y
			}
			y.AreaHa += ha
			y.Count++
		}
		kept = append(kept, f)
	}
	for _, y := range yearMap {
		out.Years = append(out.Years, *y)
	}
	sort.Slice(out.Years, func(i, j int) bool { return out.Years[i].Year > out.Years[j].Year })
	out.GeoJSON = featureCollectionJSON(kept)
	return out, nil
}

func prodesWorkspaceForBiome(name string) string {
	switch normalizeBiomeKey(name) {
	case "amazonia":
		return "prodes-amazon-nb"
	case "cerrado":
		return "prodes-cerrado-nb"
	case "caatinga":
		return "prodes-caatinga-nb"
	case "mata_atlantica":
		return "prodes-mata-atlantica-nb"
	case "pampa":
		return "prodes-pampa-nb"
	case "pantanal":
		return "prodes-pantanal-nb"
	default:
		return ""
	}
}

func normalizeBiomeKey(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	r := strings.NewReplacer("á", "a", "ã", "a", "â", "a", "à", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c", " ", "_", "-", "_")
	s = r.Replace(s)
	switch {
	case strings.Contains(s, "amazon"):
		return "amazonia"
	case strings.Contains(s, "cerrado"):
		return "cerrado"
	case strings.Contains(s, "caatinga"):
		return "caatinga"
	case strings.Contains(s, "mata") && strings.Contains(s, "atlant"):
		return "mata_atlantica"
	case strings.Contains(s, "pampa"):
		return "pampa"
	case strings.Contains(s, "pantanal"):
		return "pantanal"
	}
	return s
}

func getWFSFeatureTypes(ctx context.Context, endpoint string) ([]wfsFeatureType, error) {
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "1.0.0")
	params.Set("request", "GetCapabilities")
	body, err := fetchEnvironmentalBody(ctx, endpoint+"?"+params.Encode(), "application/xml,text/xml")
	if err != nil {
		return nil, err
	}
	var caps wfsCapabilities
	if err := xml.Unmarshal(body, &caps); err != nil {
		return nil, err
	}
	if len(caps.FeatureTypes) == 0 {
		return nil, errors.New("GetCapabilities não retornou camadas WFS")
	}
	return caps.FeatureTypes, nil
}

func chooseWFSLayer(types []wfsFeatureType, positives, negatives []string) string {
	best := ""
	bestScore := -999
	for _, ft := range types {
		text := strings.ToLower(ft.Name + " " + ft.Title)
		score := 0
		for i, p := range positives {
			if strings.Contains(text, strings.ToLower(p)) {
				score += (len(positives)-i)*5 + 5
			}
		}
		for _, n := range negatives {
			if strings.Contains(text, strings.ToLower(n)) {
				score -= 15
			}
		}
		if score > bestScore {
			bestScore = score
			best = ft.Name
		}
	}
	if bestScore <= 0 {
		return ""
	}
	return best
}

func queryWFSByCARBBox(ctx context.Context, endpoint, layer, carRaw string, maxFeatures int, cql string) (carGeoJSON, error) {
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		return carGeoJSON{}, errors.New("limites do CAR indisponíveis")
	}
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "1.0.0")
	params.Set("request", "GetFeature")
	params.Set("typeName", layer)
	params.Set("outputFormat", "application/json")
	params.Set("srsName", "EPSG:4326")
	params.Set("maxFeatures", strconv.Itoa(maxFeatures))
	params.Set("bbox", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f,EPSG:4326", minLon, minLat, maxLon, maxLat))
	if strings.TrimSpace(cql) != "" {
		params.Set("CQL_FILTER", cql)
	}
	body, err := fetchEnvironmentalBody(ctx, endpoint+"?"+params.Encode(), "application/json,application/geo+json")
	if err != nil {
		return carGeoJSON{}, err
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return carGeoJSON{}, fmt.Errorf("WFS retornou GeoJSON inválido: %w", err)
	}
	return fc, nil
}

func queryFireProfile(ctx context.Context, carRaw string) (FireProfile, error) {
	out := FireProfile{SourceURL: inpeFireWFSURL, WindowLabel: "focos ativos/recentes retornados pela camada pública do INPE"}
	types, err := getWFSFeatureTypes(ctx, inpeFireWFSURL)
	if err != nil {
		return out, err
	}
	layer := chooseWFSLayer(types,
		[]string{"active-fire", "focos_48h", "focos", "active_fire", "fire"},
		[]string{"grid", "municip", "state", "pais", "estatistic", "previsao", "risco"})
	if layer == "" {
		return out, errors.New("camada pública de focos ativos não localizada")
	}
	fc, err := queryWFSByCARBBox(ctx, inpeFireWFSURL, layer, carRaw, 1000, "")
	if err != nil {
		return out, err
	}
	out.Available = true
	out.SourceLayer = layer
	var kept []carGeoFeature
	for _, f := range fc.Features {
		lat, lon, ok := geometryFirstPoint(f.Geometry)
		if !ok || !pointInsideCAR(lat, lon, carRaw) {
			continue
		}
		out.FeatureCount++
		d := featureDate(f.Properties)
		if !d.IsZero() {
			ds := d.Format(time.RFC3339)
			if ds > out.LastDetectedAt {
				out.LastDetectedAt = ds
			}
		}
		out.MaxRisk = math.Max(out.MaxRisk, propFloat(f.Properties, "riscofogo", "risco_fogo", "risk", "fire_risk"))
		out.MaxFRP = math.Max(out.MaxFRP, propFloat(f.Properties, "frp", "FRP"))
		kept = append(kept, f)
	}
	out.GeoJSON = featureCollectionJSON(kept)
	return out, nil
}

type worldCoverPaletteEntry struct {
	Code  int
	Label string
	R, G, B int
}

var worldCoverPalette = []worldCoverPaletteEntry{
	{10, "Cobertura arbórea", 0, 100, 0},
	{20, "Arbustos", 255, 187, 34},
	{30, "Vegetação herbácea", 255, 255, 76},
	{40, "Lavoura", 240, 150, 255},
	{50, "Área construída", 250, 0, 0},
	{60, "Solo exposto / vegetação esparsa", 180, 180, 180},
	{70, "Neve e gelo", 240, 240, 240},
	{80, "Água permanente", 0, 100, 200},
	{90, "Área úmida herbácea", 0, 150, 160},
	{95, "Manguezal", 0, 207, 117},
	{100, "Musgos e líquens", 250, 230, 160},
}

func queryWorldCoverProfile(ctx context.Context, carRaw string) (LandCoverProfile, error) {
	out := LandCoverProfile{
		Approximate: true, ReferenceYear: 2021,
		Source: "ESA WorldCover 2021 v200 / Terrascope WMS (amostragem cartográfica automática)",
		Warning: "Percentuais representam a participação dos pontos amostrados na imagem WMS RGB; a ESA informa que o WMS é adequado à visualização e não substitui análise raster do produto COG.",
	}
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		return out, errors.New("limites do CAR indisponíveis")
	}
	if maxLon-minLon < 0.001 {
		pad := 0.001
		minLon -= pad
		maxLon += pad
	}
	if maxLat-minLat < 0.001 {
		pad := 0.001
		minLat -= pad
		maxLat += pad
	}
	const size = 256
	params := url.Values{}
	params.Set("service", "WMS")
	params.Set("version", "1.1.1")
	params.Set("request", "GetMap")
	params.Set("layers", worldCoverLayer)
	params.Set("styles", "")
	params.Set("srs", "EPSG:4326")
	params.Set("bbox", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f", minLon, minLat, maxLon, maxLat))
	params.Set("width", strconv.Itoa(size))
	params.Set("height", strconv.Itoa(size))
	params.Set("format", "image/png")
	params.Set("transparent", "false")
	body, err := fetchEnvironmentalBody(ctx, worldCoverWMSURL+"?"+params.Encode(), "image/png,image/*")
	if err != nil {
		return out, err
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return out, fmt.Errorf("WorldCover retornou imagem inválida: %w", err)
	}
	mp, err := geoJSONToPlanar(carRaw, sharedLatitude(carRaw))
	if err != nil {
		return out, err
	}
	lat0 := sharedLatitude(carRaw)
	counts := map[int]int{}
	total := 0
	step := 3
	for py := 1; py < size; py += step {
		lat := maxLat - (float64(py)+0.5)/size*(maxLat-minLat)
		for px := 1; px < size; px += step {
			lon := minLon + (float64(px)+0.5)/size*(maxLon-minLon)
			if !pointInMultiPolygon(projectGeoPoint(lon, lat, lat0), mp) {
				continue
			}
			r16, g16, b16, a16 := img.At(px, py).RGBA()
			if a16 == 0 {
				continue
			}
			code, ok := nearestWorldCoverClass(int(r16>>8), int(g16>>8), int(b16>>8))
			if !ok {
				continue
			}
			counts[code]++
			total++
		}
	}
	if total < 8 {
		return out, errors.New("amostragem WorldCover insuficiente dentro do CAR")
	}
	for _, p := range worldCoverPalette {
		n := counts[p.Code]
		if n == 0 {
			continue
		}
		out.Classes = append(out.Classes, LandCoverClass{Code: p.Code, Label: p.Label, Samples: n, SamplePct: float64(n) / float64(total) * 100})
	}
	sort.Slice(out.Classes, func(i, j int) bool { return out.Classes[i].Samples > out.Classes[j].Samples })
	out.Available = true
	out.SampleCount = total
	if len(out.Classes) > 0 {
		out.DominantClass = out.Classes[0].Label
		out.DominantPct = out.Classes[0].SamplePct
	}
	return out, nil
}

func nearestWorldCoverClass(r, g, b int) (int, bool) {
	bestCode, best := 0, math.Inf(1)
	for _, p := range worldCoverPalette {
		d := float64((r-p.R)*(r-p.R) + (g-p.G)*(g-p.G) + (b-p.B)*(b-p.B))
		if d < best {
			best = d
			bestCode = p.Code
		}
	}
	// Tolerância deliberadamente conservadora: o WMS é RGB e não é produto
	// analítico. Pixels muito reamostrados/escuros são descartados em vez de
	// serem forçados para uma classe de cobertura.
	return bestCode, best <= 45*45*3
}

func queryNearbyEnvironmental(ctx context.Context, car CARResult) (NearbyEnvironmentalProfile, error) {
	out := NearbyEnvironmentalProfile{SearchRadiusKm: 50}
	if car.CenterLat == 0 && car.CenterLon == 0 {
		return out, errors.New("centroide do CAR indisponível")
	}
	latSpan := out.SearchRadiusKm / 111.0
	lonSpan := latSpan / math.Max(0.2, math.Cos(car.CenterLat*math.Pi/180))
	minLon, maxLon := car.CenterLon-lonSpan, car.CenterLon+lonSpan
	minLat, maxLat := car.CenterLat-latSpan, car.CenterLat+latSpan
	success := 0
	var errs []string

	// IBAMA
	if fc, err := queryArcGISByEnvelope(ctx, strings.TrimSuffix(ibamaEmbargoLayerURL, "/query")+"/query", minLon, minLat, maxLon, maxLat, "num_tad,municipio,OBJECTID", 300); err == nil {
		success++
		best := math.Inf(1)
		for _, f := range fc.Features {
			d := minGeometryVertexDistanceKm(f.Geometry, car.CenterLat, car.CenterLon)
			if d < best {
				best = d
				out.NearestEmbargoLabel = firstNonEmptyText(carStringProp(f.Properties, "num_tad", "numero_tad"), "Embargo IBAMA")
			}
		}
		if !math.IsInf(best, 1) && best <= out.SearchRadiusKm*1.2 {
			out.EmbargoFound = true
			out.NearestEmbargoKm = best
		}
	} else {
		errs = append(errs, "IBAMA: "+err.Error())
	}

	// FUNAI
	if fc, err := querySimpleWFSBBox(ctx, funaiWFSURL, "Funai:tis_poligonais", minLon, minLat, maxLon, maxLat, 100); err == nil {
		success++
		best := math.Inf(1)
		for _, f := range fc.Features {
			d := minGeometryVertexDistanceKm(f.Geometry, car.CenterLat, car.CenterLon)
			if d < best {
				best = d
				out.NearestIndigenousLabel = firstNonEmptyText(carStringProp(f.Properties, "terrai_nom", "terra_nome", "nome", "nom_ti"), "Terra Indígena")
			}
		}
		if !math.IsInf(best, 1) && best <= out.SearchRadiusKm*1.2 {
			out.IndigenousFound = true
			out.NearestIndigenousKm = best
		}
	} else {
		errs = append(errs, "FUNAI: "+err.Error())
	}

	// ICMBio
	if fc, err := querySimpleWFSBBox(ctx, icmbioWFSURL, icmbioUCLayer, minLon, minLat, maxLon, maxLat, 100); err == nil {
		success++
		best := math.Inf(1)
		for _, f := range fc.Features {
			d := minGeometryVertexDistanceKm(f.Geometry, car.CenterLat, car.CenterLon)
			if d < best {
				best = d
				out.NearestUCLabel = firstNonEmptyText(carStringProp(f.Properties, "nome", "nome_uc", "nom_uc", "nm_uc"), "Unidade de Conservação federal")
			}
		}
		if !math.IsInf(best, 1) && best <= out.SearchRadiusKm*1.2 {
			out.UCFound = true
			out.NearestUCKm = best
		}
	} else {
		errs = append(errs, "ICMBio: "+err.Error())
	}

	out.Available = success > 0
	if success == 0 {
		return out, errors.New(strings.Join(errs, " | "))
	}
	if len(errs) > 0 {
		out.Warning = strings.Join(errs, " | ")
	}
	return out, nil
}

func querySimpleWFSBBox(ctx context.Context, endpoint, layer string, minLon, minLat, maxLon, maxLat float64, maxFeatures int) (carGeoJSON, error) {
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "1.0.0")
	params.Set("request", "GetFeature")
	params.Set("typeName", layer)
	params.Set("outputFormat", "application/json")
	params.Set("srsName", "EPSG:4326")
	params.Set("maxFeatures", strconv.Itoa(maxFeatures))
	params.Set("bbox", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f,EPSG:4326", minLon, minLat, maxLon, maxLat))
	body, err := fetchEnvironmentalBody(ctx, endpoint+"?"+params.Encode(), "application/json,application/geo+json")
	if err != nil {
		return carGeoJSON{}, err
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return carGeoJSON{}, err
	}
	return fc, nil
}

func buildEnvironmentalProfileSources(p EnvironmentalProfile) []EnvironmentalProfileSource {
	s := []EnvironmentalProfileSource{
		{Key: "terrain", Label: "Relevo / AWS Terrain Tiles", Available: p.Terrain.Available, Applicable: true, SourceURL: "https://registry.opendata.aws/terrain-tiles/", Detail: p.Terrain.Warning},
		{Key: "biome", Label: "IBGE — Biomas", Available: p.Biome.Available, Applicable: true, SourceURL: ibgeBiomeWFSURL, Detail: p.Biome.Warning},
		{Key: "hydrology", Label: "ANA / SNIRH — Hidrografia", Available: p.Hydrology.Available, Applicable: true, SourceURL: "https://portal1.snirh.gov.br/server/rest/services/dados_abertos", Detail: p.Hydrology.Warning},
		{Key: "prodes", Label: "INPE / PRODES", Available: p.PRODES.Available, Applicable: p.PRODES.Applicable, SourceURL: p.PRODES.SourceURL, Detail: p.PRODES.Warning},
		{Key: "deter", Label: "INPE / DETER", Available: p.DETER.Available, Applicable: p.DETER.Applicable, SourceURL: p.DETER.SourceURL, Detail: p.DETER.Warning},
		{Key: "fire", Label: "INPE / Queimadas", Available: p.Fire.Available, Applicable: true, SourceURL: p.Fire.SourceURL, Detail: p.Fire.Warning},
		{Key: "worldcover", Label: "ESA WorldCover 2021", Available: p.LandCover.Available, Applicable: true, SourceURL: worldCoverWMSURL, Detail: p.LandCover.Warning},
		{Key: "nearby", Label: "Proximidade territorial", Available: p.Nearby.Available, Applicable: true, SourceURL: "", Detail: p.Nearby.Warning},
	}
	for i := range s {
		switch {
		case !s[i].Applicable:
			s[i].Status = "não aplicável"
		case s[i].Available:
			s[i].Status = "consultado"
		default:
			s[i].Status = "indisponível"
		}
	}
	return s
}

func featureJSON(f carGeoFeature) string {
	b, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: f.Properties, Geometry: f.Geometry})
	return string(b)
}

func featureCollectionJSON(features []carGeoFeature) string {
	if len(features) == 0 {
		return ""
	}
	b, _ := json.Marshal(carGeoJSON{Type: "FeatureCollection", Features: features})
	return string(b)
}

func featureGeometryKey(g carGeoJSONGeometry) string {
	b, _ := json.Marshal(g)
	if len(b) > 160 {
		b = b[:160]
	}
	return string(b)
}

func featureYear(props map[string]any) int {
	for _, key := range []string{"year", "ano", "YEAR", "ANO"} {
		if v, ok := props[key]; ok {
			if n := int(math.Round(anyFloat(v))); n >= 1980 && n <= time.Now().Year()+1 {
				return n
			}
		}
	}
	for _, key := range []string{"class_name", "classname", "main_class", "classe", "CLASS_NAME"} {
		s := carStringProp(props, key)
		for i := 0; i+4 <= len(s); i++ {
			n, _ := strconv.Atoi(s[i : i+4])
			if n >= 1980 && n <= time.Now().Year()+1 {
				return n
			}
		}
	}
	return 0
}

func featureDate(props map[string]any) time.Time {
	keys := []string{"date", "view_date", "image_date", "publish_date", "pub_date", "data", "datahora", "data_hora", "DataHora"}
	for _, key := range keys {
		v, ok := props[key]
		if !ok || v == nil {
			continue
		}
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			continue
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02", "2006/01/02", "02/01/2006", "20060102", "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t
			}
		}
		if ms, err := strconv.ParseInt(strings.TrimSuffix(s, ".0"), 10, 64); err == nil && ms > 1000000000 {
			if ms > 100000000000 {
				return time.UnixMilli(ms)
			}
			return time.Unix(ms, 0)
		}
	}
	return time.Time{}
}

func propFloat(props map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := props[k]; ok {
			if f := anyFloat(v); !math.IsNaN(f) {
				return f
			}
		}
	}
	return 0
}

func anyFloat(v any) float64 {
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
		f, _ := x.Float64()
		return f
	case string:
		s := strings.ReplaceAll(strings.TrimSpace(x), ",", ".")
		f, _ := strconv.ParseFloat(s, 64)
		return f
	default:
		f, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(v)), 64)
		return f
	}
}

func geometryFirstPoint(g carGeoJSONGeometry) (lat, lon float64, ok bool) {
	var raw any
	if json.Unmarshal(g.Coordinates, &raw) != nil {
		return 0, 0, false
	}
	var walk func(any) (float64, float64, bool)
	walk = func(v any) (float64, float64, bool) {
		arr, ok := v.([]any)
		if !ok || len(arr) == 0 {
			return 0, 0, false
		}
		if len(arr) >= 2 {
			x, xok := arr[0].(float64)
			y, yok := arr[1].(float64)
			if xok && yok {
				return y, x, true
			}
		}
		for _, child := range arr {
			if la, lo, found := walk(child); found {
				return la, lo, true
			}
		}
		return 0, 0, false
	}
	return walk(raw)
}

func minGeometryVertexDistanceKm(g carGeoJSONGeometry, lat, lon float64) float64 {
	var raw any
	if json.Unmarshal(g.Coordinates, &raw) != nil {
		return math.Inf(1)
	}
	best := math.Inf(1)
	var walk func(any)
	walk = func(v any) {
		arr, ok := v.([]any)
		if !ok || len(arr) == 0 {
			return
		}
		if len(arr) >= 2 {
			x, xok := arr[0].(float64)
			y, yok := arr[1].(float64)
			if xok && yok {
				best = math.Min(best, carHaversineM(lat, lon, y, x)/1000)
				return
			}
		}
		for _, child := range arr {
			walk(child)
		}
	}
	walk(raw)
	return best
}

func pointInsideCAR(lat, lon float64, carRaw string) bool {
	lat0 := sharedLatitude(carRaw)
	mp, err := geoJSONToPlanar(carRaw, lat0)
	if err != nil {
		return false
	}
	return pointInMultiPolygon(projectGeoPoint(lon, lat, lat0), mp)
}

func projectGeoPoint(lon, lat, lat0 float64) planarPoint {
	const earthR = 6371008.8
	return planarPoint{
		X: earthR * lon * math.Pi / 180 * math.Cos(lat0*math.Pi/180),
		Y: earthR * lat * math.Pi / 180,
	}
}

func lineGeometryIntersectsCAR(g carGeoJSONGeometry, carRaw string) bool {
	lat0 := sharedLatitude(carRaw)
	mp, err := geoJSONToPlanar(carRaw, lat0)
	if err != nil || len(mp) == 0 {
		return false
	}
	lines := geometryLines(g)
	for _, line := range lines {
		var prev planarPoint
		hasPrev := false
		for _, p := range line {
			if len(p) < 2 {
				continue
			}
			cur := projectGeoPoint(p[0], p[1], lat0)
			if pointInMultiPolygon(cur, mp) {
				return true
			}
			if hasPrev && segmentIntersectsMultiPolygon(prev, cur, mp) {
				return true
			}
			prev, hasPrev = cur, true
		}
	}
	return false
}

func geometryLines(g carGeoJSONGeometry) [][][]float64 {
	switch strings.ToLower(g.Type) {
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

func segmentIntersectsMultiPolygon(a, b planarPoint, mp planarMultiPolygon) bool {
	mid := planarPoint{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}
	if pointInMultiPolygon(mid, mp) {
		return true
	}
	for _, poly := range mp {
		for _, ring := range poly {
			for i := 0; i < len(ring); i++ {
				j := (i + 1) % len(ring)
				if segmentsIntersect(a, b, ring[i], ring[j]) {
					return true
				}
			}
		}
	}
	return false
}

func segmentsIntersect(a, b, c, d planarPoint) bool {
	orient := func(p, q, r planarPoint) float64 {
		return (q.X-p.X)*(r.Y-p.Y) - (q.Y-p.Y)*(r.X-p.X)
	}
	o1, o2, o3, o4 := orient(a, b, c), orient(a, b, d), orient(c, d, a), orient(c, d, b)
	return (o1 == 0 || o2 == 0 || (o1 < 0) != (o2 < 0)) &&
		(o3 == 0 || o4 == 0 || (o3 < 0) != (o4 < 0))
}
