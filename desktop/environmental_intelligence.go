package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	environmentalIntelligenceCacheAge = 24 * time.Hour
	mapBiomasMethodologyURL = "https://alerta.mapbiomas.org/metodo-mapbiomas-alerta/"
	mapBiomasAPIURL = "https://plataforma.alerta.mapbiomas.org/api/docs/index.html"
)

type EnvironmentalAlertDetail struct {
	AlertCode       string   `json:"alert_code"`
	AreaHa          float64  `json:"area_ha"`
	AlertAreaInCAR  float64  `json:"alert_area_in_car_ha"`
	AlertPctOfCAR   float64  `json:"alert_pct_of_car"`
	DetectedAt      string   `json:"detected_at"`
	PublishedAt     string   `json:"published_at"`
	StatusName      string   `json:"status_name"`
	StatusAt        string   `json:"status_at"`
	Sources         []string `json:"sources"`
	Biomes          []string `json:"biomes"`
	Cities          []string `json:"cities"`
	DeforestationClasses []string `json:"deforestation_classes"`
	DeforestationSpeed string `json:"deforestation_speed"`
	ImageBeforeAt    string `json:"image_before_at"`
	ImageAfterAt     string `json:"image_after_at"`
	ConservationUnits []string `json:"conservation_units"`
	IndigenousLands   []string `json:"indigenous_lands"`
	Quilombos         []string `json:"quilombos"`
	Settlements       []string `json:"settlements"`
	SpecialTerritories []string `json:"special_territories"`
	DeforestationAuthorizations []string `json:"deforestation_authorizations"`
	ForestManagements []string `json:"forest_managements"`
	MapBiomasConservationAreaHa float64 `json:"mapbiomas_conservation_area_ha"`
	MapBiomasIndigenousAreaHa float64 `json:"mapbiomas_indigenous_area_ha"`
	MapBiomasAuthorizedAreaHa float64 `json:"mapbiomas_authorized_area_ha"`
	MapBiomasForestManagementAreaHa float64 `json:"mapbiomas_forest_management_area_ha"`
	MapBiomasEmbargoAreaHa float64 `json:"mapbiomas_embargo_area_ha"`
	MapBiomasLegalReserveAreaHa float64 `json:"mapbiomas_legal_reserve_area_ha"`
	MapBiomasPermanentProtectedAreaHa float64 `json:"mapbiomas_permanent_protected_area_ha"`
	MapBiomasRiverSourcesAreaHa float64 `json:"mapbiomas_river_sources_area_ha"`
	APPOverlapHa       float64 `json:"app_overlap_ha"`
	RLOverlapHa        float64 `json:"rl_overlap_ha"`
	NativeOverlapHa    float64 `json:"native_overlap_ha"`
	ConsolidatedOverlapHa float64 `json:"consolidated_overlap_ha"`
	IBAMAOverlapHa     float64 `json:"ibama_overlap_ha"`
	IndigenousOverlapHa float64 `json:"indigenous_overlap_ha"`
	FederalUCOverlapHa float64 `json:"federal_uc_overlap_ha"`
	GeometryGeoJSON   string   `json:"geometry_geojson"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	ReportURL         string   `json:"report_url"`
	AttentionLevel    string   `json:"attention_level"`
	AttentionReasons  []string `json:"attention_reasons"`
}

type EnvironmentalEvidenceSummary struct {
	Alerts                    int     `json:"alerts"`
	AlertAreaTotalHa          float64 `json:"alert_area_total_ha"`
	AlertAreaInCARHa          float64 `json:"alert_area_in_car_ha"`
	AlertsOverAPP             int     `json:"alerts_over_app"`
	APPOverlapHa              float64 `json:"app_overlap_ha"`
	AlertsOverRL              int     `json:"alerts_over_rl"`
	RLOverlapHa               float64 `json:"rl_overlap_ha"`
	AlertsOverIBAMA           int     `json:"alerts_over_ibama"`
	AlertsOverIndigenousLand  int     `json:"alerts_over_indigenous_land"`
	AlertsOverFederalUC       int     `json:"alerts_over_federal_uc"`
	HighAttentionAlerts       int     `json:"high_attention_alerts"`
	ReviewAlerts              int     `json:"review_alerts"`
	LatestDetection           string  `json:"latest_detection"`
}

type EnvironmentalIntelligenceResult struct {
	CAR             string                       `json:"car"`
	Municipality    string                       `json:"municipality"`
	UF              string                       `json:"uf"`
	PropertyAreaHa  float64                      `json:"property_area_ha"`
	GeneratedAt     string                       `json:"generated_at"`
	UsedCache       bool                         `json:"used_cache"`
	MapBiomas       MapBiomasCARSummary          `json:"mapbiomas"`
	Alerts          []EnvironmentalAlertDetail   `json:"alerts"`
	Summary         EnvironmentalEvidenceSummary `json:"summary"`
	Environment     EnvironmentalSummary         `json:"environment"`
	Themes          SICARThemesSummary           `json:"themes"`
	Warnings        []string                     `json:"warnings"`
	Interpretation  string                       `json:"interpretation"`
	MapBiomasMethodURL string                    `json:"mapbiomas_method_url"`
	MapBiomasAPIURL    string                    `json:"mapbiomas_api_url"`
}

type mapBiomasEnvironmentalResponse struct {
	Data struct {
		Alerts struct {
			Collection []struct {
				AlertCode any `json:"alertCode"`
				AreaHa float64 `json:"areaHa"`
				DetectedAt string `json:"detectedAt"`
				PublishedAt string `json:"publishedAt"`
				StatusName string `json:"statusName"`
				StatusAt string `json:"statusAt"`
				Sources []string `json:"sources"`
				CrossedBiomes []string `json:"crossedBiomes"`
				CrossedCities []string `json:"crossedCities"`
				CrossedConservationUnits []string `json:"crossedConservationUnits"`
				CrossedConservationUnitsArea float64 `json:"crossedConservationUnitsArea"`
				CrossedIndigenousLands []string `json:"crossedIndigenousLands"`
				CrossedIndigenousLandsArea float64 `json:"crossedIndigenousLandsArea"`
				CrossedQuilombos []string `json:"crossedQuilombos"`
				CrossedSettlements []string `json:"crossedSettlements"`
				CrossedSpecialTerritories []string `json:"crossedSpecialTerritories"`
				CrossedDeforestationAuthorizationsActivities []string `json:"crossedDeforestationAuthorizationsActivities"`
				CrossedDeforestationAuthorizationsArea float64 `json:"crossedDeforestationAuthorizationsArea"`
				CrossedForestManagementsActivities []string `json:"crossedForestManagementsActivities"`
				CrossedForestManagementsArea float64 `json:"crossedForestManagementsArea"`
				CrossedEmbargoesRuralPropertiesArea float64 `json:"crossedEmbargoesRuralPropertiesArea"`
				CrossedLegalReserveRuralPropertiesArea float64 `json:"crossedLegalReserveRuralPropertiesArea"`
				CrossedPermanentProtectedRuralPropertiesArea float64 `json:"crossedPermanentProtectedRuralPropertiesArea"`
				CrossedRiverSourcesArea float64 `json:"crossedRiverSourcesArea"`
				DeforestationClasses []string `json:"deforestationClasses"`
				DeforestationSpeed string `json:"deforestationSpeed"`
				ImageAcquiredBeforeAt string `json:"imageAcquiredBeforeAt"`
				ImageAcquiredAfterAt string `json:"imageAcquiredAfterAt"`
				GeometryWKT string `json:"geometryWkt"`
				Coordenates struct {
					Latitude float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"coordenates"`
				CrossedRuralProperties []struct {
					Code string `json:"code"`
					AreaHa float64 `json:"areaHa"`
					AlertAreaInCAR float64 `json:"alertAreaInCar"`
					Type string `json:"type"`
				} `json:"crossedRuralProperties"`
			} `json:"collection"`
		} `json:"alerts"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

func (a *App) GetEnvironmentalIntelligence(propertyID int64, force bool) (EnvironmentalIntelligenceResult, error) {
	car, err := a.carForEnvironmental(propertyID)
	if err != nil {
		return EnvironmentalIntelligenceResult{}, err
	}
	cachePath := ""
	if a != nil && a.dataDir != "" {
		cachePath = filepath.Join(a.dataDir, "cache", "environmental_intelligence", safeFilePart(car.CAR)+".json")
		if !force {
			if cached, ok := loadEnvironmentalIntelligenceCache(cachePath, environmentalIntelligenceCacheAge); ok {
				cached.UsedCache = true
				return cached, nil
			}
		}
	}

	out := EnvironmentalIntelligenceResult{
		CAR: car.CAR,
		Municipality: car.Municipality,
		UF: car.UF,
		PropertyAreaHa: firstPositive(car.GeometryAreaHa, car.AreaHa),
		GeneratedAt: time.Now().Format(time.RFC3339),
		Environment: car.Environment,
		Themes: car.Themes,
		MapBiomasMethodURL: mapBiomasMethodologyURL,
		MapBiomasAPIURL: mapBiomasAPIURL,
		Interpretation: "Triagem técnica auxiliar baseada em fontes públicas e cruzamentos espaciais. A presença de alerta ou sobreposição não determina, por si só, infração, autoria, responsabilidade ou impedimento de crédito; exige conferência documental e, quando aplicável, análise por profissional habilitado e pelo órgão competente.",
	}

	mb, err := a.QueryMapBiomasCAR(car.CAR)
	if err != nil {
		out.Warnings = append(out.Warnings, "MapBiomas Alerta: "+err.Error())
		mb = MapBiomasCARSummary{Connected:a.GetMapBiomasAlertStatus().Connected, Message:err.Error()}
	}
	out.MapBiomas = mb

	if mb.Connected && mb.TotalAlerts > 0 {
		details, detailErr := a.queryMapBiomasEnvironmentalDetails(car.CAR)
		if detailErr != nil {
			out.Warnings = append(out.Warnings, "Detalhamento MapBiomas: "+detailErr.Error())
		}
		if len(details) > 0 {
			out.Alerts = details
		} else {
			out.Alerts = alertsFromMapBiomasSummary(mb)
		}
	} else if mb.TotalAlerts > 0 {
		out.Alerts = alertsFromMapBiomasSummary(mb)
	}

	for i := range out.Alerts {
		a.enrichEnvironmentalAlert(&out.Alerts[i], car)
	}
	sort.SliceStable(out.Alerts, func(i,j int) bool {
		if out.Alerts[i].DetectedAt == out.Alerts[j].DetectedAt {
			return out.Alerts[i].AlertCode > out.Alerts[j].AlertCode
		}
		return out.Alerts[i].DetectedAt > out.Alerts[j].DetectedAt
	})
	out.Summary = summarizeEnvironmentalEvidence(out.Alerts)

	if !mb.Connected {
		out.Warnings = append(out.Warnings, "MapBiomas Alerta não está conectado; a análise profissional de alertas fica limitada às demais camadas públicas já carregadas.")
	}
	if len(out.Alerts) == 0 && mb.Connected {
		out.Warnings = append(out.Warnings, "Nenhum alerta foi retornado para o CAR nesta consulta. Isso não equivale a certificado de regularidade ambiental.")
	}
	out.Warnings = append(out.Warnings, car.Environment.Warnings...)
	out.Warnings = append(out.Warnings, car.Themes.Warnings...)
	out.Warnings = uniqueStrings(out.Warnings)

	if cachePath != "" {
		if err := saveEnvironmentalIntelligenceCache(cachePath, out); err != nil {
			out.Warnings = append(out.Warnings, "Não foi possível salvar o cache ambiental: "+err.Error())
		}
	}
	return out, nil
}

func (a *App) carForEnvironmental(propertyID int64) (CARResult, error) {
	if propertyID > 0 {
		car, err := a.GetLatestCAR(propertyID)
		if err != nil || strings.TrimSpace(car.CAR) == "" {
			return CARResult{}, errors.New("consulte o CAR deste imóvel antes da análise ambiental")
		}
		return car, nil
	}
	cache, err := a.GetLastCARSession()
	if err != nil || strings.TrimSpace(cache.Result.CAR) == "" {
		return CARResult{}, errors.New("consulte um CAR antes da análise ambiental")
	}
	return cache.Result, nil
}

func (a *App) queryMapBiomasEnvironmentalDetails(car string) ([]EnvironmentalAlertDetail, error) {
	token := strings.TrimSpace(a.getSetting("mapbiomas_alert_token"))
	if token == "" {
		return nil, errors.New("conta MapBiomas Alerta não conectada")
	}
	query := `query environmentalAlerts($carCodes: [ID!], $carCode: String!) {
		alerts(carCodes: $carCodes, limit: 100) {
			collection {
				alertCode
				areaHa
				detectedAt
				publishedAt
				statusName
				statusAt
				sources
				crossedBiomes
				crossedCities
				crossedConservationUnits
				crossedConservationUnitsArea
				crossedIndigenousLands
				crossedIndigenousLandsArea
				crossedQuilombos
				crossedSettlements
				crossedSpecialTerritories
				crossedDeforestationAuthorizationsActivities
				crossedDeforestationAuthorizationsArea
				crossedForestManagementsActivities
				crossedForestManagementsArea
				crossedEmbargoesRuralPropertiesArea
				crossedLegalReserveRuralPropertiesArea
				crossedPermanentProtectedRuralPropertiesArea
				crossedRiverSourcesArea
				deforestationClasses
				deforestationSpeed
				imageAcquiredBeforeAt
				imageAcquiredAfterAt
				geometryWkt
				coordenates { latitude longitude }
				crossedRuralProperties(carCode: $carCode) {
					code
					areaHa
					alertAreaInCar
					type
				}
			}
		}
	}`
	var resp mapBiomasEnvironmentalResponse
	err := mapBiomasGraphQL(token, graphQLRequest{
		Query: query,
		Variables: map[string]any{"carCodes":[]string{car}, "carCode":car},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, errors.New(joinGraphQLErrors(resp.Errors))
	}
	out := make([]EnvironmentalAlertDetail,0,len(resp.Data.Alerts.Collection))
	for _, x := range resp.Data.Alerts.Collection {
		item := EnvironmentalAlertDetail{
			AlertCode:graphQLScalarString(x.AlertCode),
			AreaHa:x.AreaHa,
			DetectedAt:x.DetectedAt,
			PublishedAt:x.PublishedAt,
			StatusName:x.StatusName,
			StatusAt:x.StatusAt,
			Sources:x.Sources,
			Biomes:x.CrossedBiomes,
			Cities:x.CrossedCities,
			DeforestationClasses:x.DeforestationClasses,
			DeforestationSpeed:x.DeforestationSpeed,
			ImageBeforeAt:x.ImageAcquiredBeforeAt,
			ImageAfterAt:x.ImageAcquiredAfterAt,
			ConservationUnits:x.CrossedConservationUnits,
			IndigenousLands:x.CrossedIndigenousLands,
			Quilombos:x.CrossedQuilombos,
			Settlements:x.CrossedSettlements,
			SpecialTerritories:x.CrossedSpecialTerritories,
			DeforestationAuthorizations:x.CrossedDeforestationAuthorizationsActivities,
			ForestManagements:x.CrossedForestManagementsActivities,
			MapBiomasConservationAreaHa:x.CrossedConservationUnitsArea,
			MapBiomasIndigenousAreaHa:x.CrossedIndigenousLandsArea,
			MapBiomasAuthorizedAreaHa:x.CrossedDeforestationAuthorizationsArea,
			MapBiomasForestManagementAreaHa:x.CrossedForestManagementsArea,
			MapBiomasEmbargoAreaHa:x.CrossedEmbargoesRuralPropertiesArea,
			MapBiomasLegalReserveAreaHa:x.CrossedLegalReserveRuralPropertiesArea,
			MapBiomasPermanentProtectedAreaHa:x.CrossedPermanentProtectedRuralPropertiesArea,
			MapBiomasRiverSourcesAreaHa:x.CrossedRiverSourcesArea,
			Latitude:x.Coordenates.Latitude,
			Longitude:x.Coordenates.Longitude,
		}
		for _, rp := range x.CrossedRuralProperties {
			if strings.EqualFold(strings.TrimSpace(rp.Code), strings.TrimSpace(car)) {
				if rp.AlertAreaInCAR > item.AlertAreaInCAR {
					item.AlertAreaInCAR = rp.AlertAreaInCAR
				}
			}
		}
		if item.AlertCode != "" {
			item.ReportURL = "https://plataforma.alerta.mapbiomas.org/alerta/"+item.AlertCode
		}
		if strings.TrimSpace(x.GeometryWKT) != "" {
			wkt := stripWKTSpatialReference(x.GeometryWKT)
			if geo, geoErr := sicorWKTToGeoJSON(wkt); geoErr == nil {
				item.GeometryGeoJSON = geo
			}
		}
		out = append(out,item)
	}
	return out,nil
}

func alertsFromMapBiomasSummary(s MapBiomasCARSummary) []EnvironmentalAlertDetail {
	out := make([]EnvironmentalAlertDetail,0,len(s.Alerts))
	for _, x := range s.Alerts {
		out = append(out,EnvironmentalAlertDetail{
			AlertCode:x.AlertCode, AreaHa:x.AreaHa, DetectedAt:x.DetectedAt,
			PublishedAt:x.PublishedAt, Sources:x.Sources, Latitude:x.Latitude,
			Longitude:x.Longitude, ReportURL:x.ReportURL,
		})
	}
	return out
}

func (a *App) enrichEnvironmentalAlert(item *EnvironmentalAlertDetail, car CARResult) {
	if item == nil { return }
	carArea := firstPositive(car.GeometryAreaHa,car.AreaHa)
	if item.GeometryGeoJSON != "" && car.GeoJSON != "" {
		if intersection, _, _, err := estimateGeometryOverlap(item.GeometryGeoJSON,car.GeoJSON); err == nil && intersection > 0 {
			item.AlertAreaInCAR = intersection
		}
	}
	if item.AlertAreaInCAR <= 0 && item.AreaHa > 0 {
		item.AlertAreaInCAR = item.AreaHa
	}
	if carArea > 0 {
		item.AlertPctOfCAR = item.AlertAreaInCAR/carArea*100
	}

	themeOverlap := func(code string) float64 {
		m,ok := car.Themes.Themes[code]
		if !ok || strings.TrimSpace(m.GeoJSON)=="" || item.GeometryGeoJSON=="" { return 0 }
		area,_,_,err := estimateGeometryOverlap(item.GeometryGeoJSON,m.GeoJSON)
		if err != nil || area < 0 { return 0 }
		return area
	}
	item.APPOverlapHa = themeOverlap("APP")
	item.RLOverlapHa = themeOverlap("RESERVA_LEGAL")
	item.NativeOverlapHa = themeOverlap("VEGETACAO_NATIVA")
	item.ConsolidatedOverlapHa = themeOverlap("AREA_CONSOLIDADA")

	if item.GeometryGeoJSON != "" {
		for _, finding := range car.Environment.IBAMAEmbargos {
			item.IBAMAOverlapHa += environmentalLayerOverlap(item.GeometryGeoJSON,finding.GeoJSON)
		}
		for _, finding := range car.Environment.IndigenousFindings {
			item.IndigenousOverlapHa += environmentalLayerOverlap(item.GeometryGeoJSON,finding.GeoJSON)
		}
		for _, finding := range car.Environment.FederalUCFindings {
			item.FederalUCOverlapHa += environmentalLayerOverlap(item.GeometryGeoJSON,finding.GeoJSON)
		}
	}
	item.AttentionLevel,item.AttentionReasons = environmentalAttention(*item,car.Environment.MCRListed)
}

func environmentalLayerOverlap(alertGeo, layerGeo string) float64 {
	if strings.TrimSpace(alertGeo)=="" || strings.TrimSpace(layerGeo)=="" { return 0 }
	area,_,_,err := estimateGeometryOverlap(alertGeo,layerGeo)
	if err != nil || area < 0 { return 0 }
	return area
}

func environmentalAttention(a EnvironmentalAlertDetail,mcrListed bool)(string,[]string){
	var reasons []string
	high := false
	if a.APPOverlapHa > 0 { high=true; reasons=append(reasons,fmt.Sprintf("interseção estimada com APP declarada: %.4f ha",a.APPOverlapHa)) }
	if a.RLOverlapHa > 0 { high=true; reasons=append(reasons,fmt.Sprintf("interseção estimada com Reserva Legal declarada: %.4f ha",a.RLOverlapHa)) }
	if a.IBAMAOverlapHa > 0 { high=true; reasons=append(reasons,fmt.Sprintf("interseção estimada com embargo IBAMA: %.4f ha",a.IBAMAOverlapHa)) }
	if a.IndigenousOverlapHa > 0 { high=true; reasons=append(reasons,fmt.Sprintf("interseção estimada com Terra Indígena: %.4f ha",a.IndigenousOverlapHa)) }
	if a.FederalUCOverlapHa > 0 { high=true; reasons=append(reasons,fmt.Sprintf("interseção estimada com UC federal: %.4f ha",a.FederalUCOverlapHa)) }
	if a.MapBiomasAuthorizedAreaHa > 0 { reasons=append(reasons,"o MapBiomas reporta cruzamento com área de autorização de supressão; conferir documento, vigência e polígono") }
	if a.MapBiomasForestManagementAreaHa > 0 { reasons=append(reasons,"o MapBiomas reporta cruzamento com área de manejo florestal; conferir documentação") }
	if mcrListed { high=true; reasons=append(reasons,"o CAR aparece na lista pública MMA/MCR-PRODES consultada") }
	if high { return "Alta prioridade de conferência",reasons }
	if a.AlertAreaInCAR > 0 || a.AreaHa > 0 {
		if len(reasons)==0 { reasons=append(reasons,"alerta MapBiomas vinculado ao imóvel; conferir data, geometria, imagens e documentos") }
		return "Conferir",reasons
	}
	return "Informativo",reasons
}

func summarizeEnvironmentalEvidence(items []EnvironmentalAlertDetail) EnvironmentalEvidenceSummary {
	var s EnvironmentalEvidenceSummary
	s.Alerts=len(items)
	for _,a:=range items {
		s.AlertAreaTotalHa+=a.AreaHa
		s.AlertAreaInCARHa+=a.AlertAreaInCAR
		s.APPOverlapHa+=a.APPOverlapHa
		s.RLOverlapHa+=a.RLOverlapHa
		if a.APPOverlapHa>0{s.AlertsOverAPP++}
		if a.RLOverlapHa>0{s.AlertsOverRL++}
		if a.IBAMAOverlapHa>0{s.AlertsOverIBAMA++}
		if a.IndigenousOverlapHa>0{s.AlertsOverIndigenousLand++}
		if a.FederalUCOverlapHa>0{s.AlertsOverFederalUC++}
		if a.AttentionLevel=="Alta prioridade de conferência"{s.HighAttentionAlerts++}
		if a.AttentionLevel=="Conferir"{s.ReviewAlerts++}
		if a.DetectedAt>s.LatestDetection{s.LatestDetection=a.DetectedAt}
	}
	return s
}

func stripWKTSpatialReference(v string) string {
	v=strings.TrimSpace(v)
	if i:=strings.Index(v,";"); i>0 && strings.HasPrefix(strings.ToUpper(v),"SRID=") {
		return strings.TrimSpace(v[i+1:])
	}
	return v
}

func firstPositive(values ...float64) float64 {
	for _,v:=range values { if v>0{return v} }
	return 0
}

func uniqueStrings(in []string) []string {
	seen:=map[string]bool{}
	out:=make([]string,0,len(in))
	for _,v:=range in {
		v=strings.TrimSpace(v)
		if v==""||seen[v]{continue}
		seen[v]=true;out=append(out,v)
	}
	return out
}

func loadEnvironmentalIntelligenceCache(path string,maxAge time.Duration)(EnvironmentalIntelligenceResult,bool){
	st,err:=os.Stat(path)
	if err!=nil||st.Size()==0||time.Since(st.ModTime())>maxAge{return EnvironmentalIntelligenceResult{},false}
	b,err:=os.ReadFile(path);if err!=nil{return EnvironmentalIntelligenceResult{},false}
	var out EnvironmentalIntelligenceResult
	if json.Unmarshal(b,&out)!=nil||strings.TrimSpace(out.CAR)==""{return EnvironmentalIntelligenceResult{},false}
	return out,true
}

func saveEnvironmentalIntelligenceCache(path string,out EnvironmentalIntelligenceResult) error {
	if err:=os.MkdirAll(filepath.Dir(path),0o755);err!=nil{return err}
	b,err:=json.MarshalIndent(out,"","  ");if err!=nil{return err}
	return os.WriteFile(path,b,0o644)
}

func (a *App) ClearEnvironmentalIntelligenceCache(car string) error {
	if a==nil||a.dataDir==""{return nil}
	car=strings.TrimSpace(car)
	if car==""{return errors.New("CAR não informado")}
	path:=filepath.Join(a.dataDir,"cache","environmental_intelligence",safeFilePart(car)+".json")
	if err:=os.Remove(path);err!=nil&&!os.IsNotExist(err){return err}
	return nil
}

func findEnvironmentalAlert(items []EnvironmentalAlertDetail, code string)(EnvironmentalAlertDetail,bool){
	code=strings.TrimSpace(code)
	for _,x:=range items{if strings.EqualFold(strings.TrimSpace(x.AlertCode),code){return x,true}}
	return EnvironmentalAlertDetail{},false
}

func mapBiomasAlertCodeInt(code string)(int,error){
	n,err:=strconv.Atoi(strings.TrimSpace(code))
	if err!=nil{return 0,fmt.Errorf("código de alerta MapBiomas inválido: %s",code)}
	return n,nil
}
