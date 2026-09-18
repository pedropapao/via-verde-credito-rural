package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	ibamaEmbargoLayerURL = "https://pamgia.ibama.gov.br/server/rest/services/01_Publicacoes_Bases/embargos_siscom_brasil/FeatureServer/2/query"
	funaiWFSURL           = "https://geoserver.funai.gov.br/geoserver/Funai/ows"
	environmentMCRURL     = "https://www.gov.br/mma/pt-br/assuntos/controle-ao-desmatamento-queimadas-e-ordenamento-ambiental-territorial/controle-do-desmatamento-1/atendimento-ao-manual-de-credito-rural"
	icmbioGeoURL          = "https://www.gov.br/icmbio/pt-br/dados-icmbio/dados_geoespaciais"
	funaiGeoURL           = "https://www.gov.br/funai/pt-br/atuacao/terras-indigenas/geoprocessamento-e-mapas"
)

type EnvironmentalSummary struct {
	CheckedAt          string             `json:"checked_at"`
	IBAMAChecked       bool               `json:"ibama_checked"`
	IBAMAEmbargoCount  int                `json:"ibama_embargo_count"`
	IBAMAEmbargos      []EmbargoFinding   `json:"ibama_embargos"`
	FUNAIChecked       bool               `json:"funai_checked"`
	IndigenousCount    int                `json:"indigenous_count"`
	IndigenousFindings []TerritoryFinding `json:"indigenous_findings"`
	Warnings           []string           `json:"warnings"`
	IBAMASourceURL     string             `json:"ibama_source_url"`
	FUNAISourceURL     string             `json:"funai_source_url"`
	ICMBioSourceURL    string             `json:"icmbio_source_url"`
	MCRSourceURL       string             `json:"mcr_source_url"`
}

type EmbargoFinding struct {
	Number       string `json:"number"`
	Date         string `json:"date"`
	Status       string `json:"status"`
	Situation    string `json:"situation"`
	Area         string `json:"area"`
	Municipality string `json:"municipality"`
	Agency       string `json:"agency"`
	Infraction   string `json:"infraction"`
}

type TerritoryFinding struct {
	Name           string  `json:"name"`
	Phase          string  `json:"phase"`
	OverlapAreaHa  float64 `json:"overlap_area_ha"`
	OverlapCARPct  float64 `json:"overlap_car_pct"`
	Source         string  `json:"source"`
}

func screenEnvironment(ctx context.Context, carGeoJSON string) EnvironmentalSummary {
	out := EnvironmentalSummary{
		CheckedAt:      time.Now().Format(time.RFC3339),
		IBAMASourceURL: "https://pamgia.ibama.gov.br/server/rest/services/01_Publicacoes_Bases/embargos_siscom_brasil/FeatureServer/2",
		FUNAISourceURL: funaiGeoURL,
		ICMBioSourceURL: icmbioGeoURL,
		MCRSourceURL:   environmentMCRURL,
	}
	if strings.TrimSpace(carGeoJSON) == "" {
		out.Warnings = append(out.Warnings, "Sem geometria do CAR para cruzamentos territoriais.")
		return out
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(2)

	go func() {
		defer wg.Done()
		findings, err := queryIBAMAEmbargos(ctx, carGeoJSON)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			out.Warnings = append(out.Warnings, "IBAMA: "+err.Error())
			return
		}
		out.IBAMAChecked = true
		out.IBAMAEmbargos = findings
		out.IBAMAEmbargoCount = len(findings)
	}()

	go func() {
		defer wg.Done()
		findings, err := queryFUNAITerritories(ctx, carGeoJSON)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			out.Warnings = append(out.Warnings, "FUNAI: "+err.Error())
			return
		}
		out.FUNAIChecked = true
		out.IndigenousFindings = findings
		out.IndigenousCount = len(findings)
	}()

	wg.Wait()
	return out
}

func queryIBAMAEmbargos(ctx context.Context, carGeoJSON string) ([]EmbargoFinding, error) {
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carGeoJSON)
	if !ok {
		return nil, fmt.Errorf("limites do CAR indisponíveis")
	}
	params := url.Values{}
	params.Set("where", "1=1")
	params.Set("geometry", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f", minLon, minLat, maxLon, maxLat))
	params.Set("geometryType", "esriGeometryEnvelope")
	params.Set("inSR", "4326")
	params.Set("spatialRel", "esriSpatialRelIntersects")
	params.Set("outFields", "numero_tad,data_tad,status_tad,sit_embarg,qtd_area_d,nom_munici,orgao,des_infrac")
	params.Set("returnGeometry", "true")
	params.Set("outSR", "4326")
	params.Set("resultRecordCount", "200")
	params.Set("f", "geojson")

	body, err := fetchCARBody(ctx, ibamaEmbargoLayerURL+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, err
	}
	out := make([]EmbargoFinding, 0, len(fc.Features))
	for _, feature := range fc.Features {
		raw, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: feature.Properties, Geometry: feature.Geometry})
		intersection, _, _, err := estimateGeometryOverlap(carGeoJSON, string(raw))
		if err != nil || intersection <= 0.0001 {
			continue
		}
		a := feature.Properties
		out = append(out, EmbargoFinding{
			Number:       anyString(a, "numero_tad"),
			Date:         arcGISDateString(a["data_tad"]),
			Status:       anyString(a, "status_tad"),
			Situation:    anyString(a, "sit_embarg"),
			Area:         anyString(a, "qtd_area_d"),
			Municipality: anyString(a, "nom_munici"),
			Agency:       anyString(a, "orgao"),
			Infraction:   anyString(a, "des_infrac"),
		})
	}
	return out, nil
}

func queryFUNAITerritories(ctx context.Context, carGeoJSON string) ([]TerritoryFinding, error) {
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carGeoJSON)
	if !ok {
		return nil, fmt.Errorf("limites do CAR indisponíveis")
	}
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "1.0.0")
	params.Set("request", "GetFeature")
	params.Set("typeName", "Funai:tis_poligonais")
	params.Set("outputFormat", "application/json")
	params.Set("srsName", "EPSG:4326")
	params.Set("maxFeatures", "100")
	params.Set("bbox", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f,EPSG:4326", minLon, minLat, maxLon, maxLat))
	body, err := fetchCARBody(ctx, funaiWFSURL+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, err
	}
	out := []TerritoryFinding{}
	for _, feature := range fc.Features {
		raw, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: feature.Properties, Geometry: feature.Geometry})
		intersection, _, carPct, err := estimateGeometryOverlap(carGeoJSON, string(raw))
		if err != nil || intersection <= 0.0001 {
			continue
		}
		name := carStringProp(feature.Properties, "terrai_nom", "terra_nome", "nome", "nom_ti", "terrai")
		phase := carStringProp(feature.Properties, "fase_ti", "fase", "etapa", "situacao")
		out = append(out, TerritoryFinding{
			Name:          name,
			Phase:         phase,
			OverlapAreaHa: intersection,
			OverlapCARPct: carPct,
			Source:        "FUNAI",
		})
	}
	return out, nil
}

func geoJSONToEsriPolygon(raw string) (map[string]any, error) {
	var f carGeoFeature
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return nil, err
	}
	polys, err := carGeometryPolygons(f.Geometry)
	if err != nil {
		return nil, err
	}
	rings := make([][][]float64, 0)
	for _, poly := range polys {
		for _, ring := range poly {
			if len(ring) < 3 {
				continue
			}
			copied := make([][]float64, 0, len(ring)+1)
			for _, p := range ring {
				if len(p) >= 2 {
					copied = append(copied, []float64{p[0], p[1]})
				}
			}
			if len(copied) >= 3 {
				first, last := copied[0], copied[len(copied)-1]
				if first[0] != last[0] || first[1] != last[1] {
					copied = append(copied, []float64{first[0], first[1]})
				}
				rings = append(rings, copied)
			}
		}
	}
	if len(rings) == 0 {
		return nil, fmt.Errorf("geometria do CAR vazia")
	}
	return map[string]any{"rings": rings, "spatialReference": map[string]any{"wkid": 4326}}, nil
}

func geoJSONBounds(raw string) (minLon, minLat, maxLon, maxLat float64, ok bool) {
	var f carGeoFeature
	if json.Unmarshal([]byte(raw), &f) != nil {
		return
	}
	polys, err := carGeometryPolygons(f.Geometry)
	if err != nil {
		return
	}
	minLon, minLat = math.Inf(1), math.Inf(1)
	maxLon, maxLat = math.Inf(-1), math.Inf(-1)
	for _, poly := range polys {
		for _, ring := range poly {
			for _, p := range ring {
				if len(p) < 2 {
					continue
				}
				minLon, minLat = math.Min(minLon, p[0]), math.Min(minLat, p[1])
				maxLon, maxLat = math.Max(maxLon, p[0]), math.Max(maxLat, p[1])
				ok = true
			}
		}
	}
	return
}

func anyString(m map[string]any, key string) string {
	for k, v := range m {
		if strings.EqualFold(k, key) && v != nil {
			return strings.TrimSpace(fmt.Sprint(v))
		}
	}
	return ""
}

func arcGISDateString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case float64:
		if x <= 0 {
			return ""
		}
		return time.UnixMilli(int64(x)).Format("02/01/2006")
	case int64:
		if x <= 0 {
			return ""
		}
		return time.UnixMilli(x).Format("02/01/2006")
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
