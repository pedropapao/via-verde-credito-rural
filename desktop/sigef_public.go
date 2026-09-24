package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	sigefPublicPrimaryQueryURL  = "https://pamgia.ibama.gov.br/server/rest/services/BasesSincronizadas/lim_sigef_publico_incra_p/FeatureServer/0/query"
	sigefPublicFallbackQueryURL = "https://pamgia.ibama.gov.br/server/rest/services/01_Publicacoes_Bases/lim_imovel_sigef_publico_a/FeatureServer/10/query"
	sigefPublicSourceURL        = "https://pamgia.ibama.gov.br/server/rest/services/BasesSincronizadas/lim_sigef_publico_incra_p/FeatureServer/0"
	sigefPublicUnavailableMessage = "Base pública do SIGEF/INCRA indisponível nesta tentativa. A ausência de resultado não significa ausência de parcela certificada. Tente atualizar a análise mais tarde."
)

type SIGEFParcel struct {
	ParcelCode          string  `json:"parcel_code"`
	TechnicalRT         string  `json:"technical_rt"`
	ART                 string  `json:"art"`
	PropertySituation   string  `json:"property_situation"`
	PropertyCode        string  `json:"property_code"`
	SubmissionDate      string  `json:"submission_date"`
	ApprovalDate        string  `json:"approval_date"`
	Status              string  `json:"status"`
	AreaName            string  `json:"area_name"`
	Registry            string  `json:"registry"`
	RegistryDate        string  `json:"registry_date"`
	MunicipalityCode    string  `json:"municipality_code"`
	UFID                string  `json:"uf_id"`
	AreaHa              float64 `json:"area_ha"`
	IntersectionAreaHa  float64 `json:"intersection_area_ha"`
	ParcelInsideCARPct  float64 `json:"parcel_inside_car_pct"`
	CARInsideParcelPct  float64 `json:"car_inside_parcel_pct"`
	AreaDifferenceHa    float64 `json:"area_difference_ha"`
	AreaDifferencePct   float64 `json:"area_difference_pct"`
	ComparisonLevel     string  `json:"comparison_level"`
	ComparisonSummary   string  `json:"comparison_summary"`
	GeoJSON             string  `json:"geojson"`
}

type SIGEFPublicResult struct {
	CheckedAt              string        `json:"checked_at"`
	Available              bool          `json:"available"`
	ParcelCount            int           `json:"parcel_count"`
	RegistryCount          int           `json:"registry_count"`
	BestCARCoveragePct     float64       `json:"best_car_coverage_pct"`
	BestParcelCoveragePct  float64       `json:"best_parcel_coverage_pct"`
	BestAreaDifferenceHa   float64       `json:"best_area_difference_ha"`
	Parcels                []SIGEFParcel `json:"parcels"`
	Warnings               []string      `json:"warnings"`
	SourceURL              string        `json:"source_url"`
	Message                string        `json:"message"`
}

func (a *App) QuerySIGEFPublic(propertyID int64) (SIGEFPublicResult, error) {
	car, err := a.carForRuralCredit(propertyID)
	if err != nil {
		return SIGEFPublicResult{}, err
	}
	if !car.Found || strings.TrimSpace(car.GeoJSON) == "" {
		return SIGEFPublicResult{}, errors.New("consulte o CAR com geometria antes da análise SIGEF")
	}
	base := context.Background()
	if a != nil && a.ctx != nil {
		base = a.ctx
	}
	ctx, cancel := context.WithTimeout(base, 75*time.Second)
	defer cancel()
	return querySIGEFPublic(ctx, car.GeoJSON)
}

func querySIGEFPublic(ctx context.Context, carRaw string) (SIGEFPublicResult, error) {
	out := SIGEFPublicResult{
		CheckedAt: time.Now().Format(time.RFC3339),
		SourceURL: sigefPublicSourceURL,
	}
	if strings.TrimSpace(carRaw) == "" {
		return out, errors.New("geometria do CAR não informada")
	}
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		return out, errors.New("não foi possível calcular os limites do CAR")
	}

	var parcels []SIGEFParcel
	var sourceErrs []string
	succeeded := false
	for _, endpoint := range []string{sigefPublicPrimaryQueryURL, sigefPublicFallbackQueryURL} {
		base := url.Values{}
		base.Set("where", "1=1")
		base.Set("geometry", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f", minLon, minLat, maxLon, maxLat))
		base.Set("geometryType", "esriGeometryEnvelope")
		base.Set("inSR", "4326")
		base.Set("spatialRel", "esriSpatialRelIntersects")

		count, err := queryArcGISCount(ctx, endpoint, base, fetchEnvironmentalBody)
		if err != nil {
			sourceErrs = append(sourceErrs, err.Error())
			if ctx.Err() != nil {
				break
			}
			continue
		}
		features, err := queryArcGISGeoJSONPages(
			ctx, endpoint, base,
			"parcela_co,rt,art,situacao_i,codigo_imo,data_submi,data_aprov,status,nome_area,registro_m,registro_d,municipio_,uf_id",
			200, count, fetchEnvironmentalBody,
		)
		if err != nil {
			sourceErrs = append(sourceErrs, err.Error())
			if ctx.Err() != nil {
				break
			}
			continue
		}
		raw, marshalErr := json.Marshal(carGeoJSON{Type: "FeatureCollection", Features: features})
		if marshalErr != nil {
			sourceErrs = append(sourceErrs, marshalErr.Error())
			continue
		}
		parsed, parseErr := parseSIGEFPublicGeoJSON(raw, carRaw)
		if parseErr != nil {
			sourceErrs = append(sourceErrs, parseErr.Error())
			continue
		}
		parcels = parsed
		succeeded = true
		break
	}
	if !succeeded {
		out.Message = sigefPublicUnavailableMessage
		return out, sigefPublicUnavailableError(sourceErrs)
	}
	out.Available = true
	out.Parcels = parcels
	out.ParcelCount = len(parcels)

	registries := map[string]bool{}
	for _, p := range parcels {
		if strings.TrimSpace(p.Registry) != "" {
			registries[strings.ToUpper(strings.TrimSpace(p.Registry))] = true
		}
		if p.CARInsideParcelPct > out.BestCARCoveragePct {
			out.BestCARCoveragePct = p.CARInsideParcelPct
			out.BestParcelCoveragePct = p.ParcelInsideCARPct
			out.BestAreaDifferenceHa = p.AreaDifferenceHa
		}
	}
	out.RegistryCount = len(registries)

	switch len(parcels) {
	case 0:
		out.Message = "Nenhuma parcela SIGEF pública foi localizada por interseção espacial com este CAR."
		out.Warnings = append(out.Warnings, "Ausência de parcela SIGEF pública não prova ausência de matrícula, registro imobiliário ou imóvel no SNCR.")
	case 1:
		out.Message = "Uma parcela SIGEF pública intersecta o CAR. O vínculo é espacial e deve ser conferido com os documentos do imóvel."
	default:
		out.Message = fmt.Sprintf("%d parcelas SIGEF públicas intersectam o CAR.", len(parcels))
		out.Warnings = append(out.Warnings, "Há mais de uma parcela SIGEF pública intersectando o CAR; confira se o imóvel reúne parcelas distintas, cessões ou limites cadastrais diferentes.")
	}
	return out, nil
}

func sigefPublicUnavailableError(sourceErrs []string) error {
	if len(sourceErrs) > 0 {
		log.Printf("[SIGEF/INCRA] consulta pública indisponível: %s", strings.Join(sourceErrs, " | "))
	} else {
		log.Printf("[SIGEF/INCRA] consulta pública indisponível: nenhuma fonte respondeu")
	}
	return errors.New(sigefPublicUnavailableMessage)
}

func parseSIGEFPublicGeoJSON(body []byte, carRaw string) ([]SIGEFParcel, error) {
	var arcErr struct {
		Error *struct {
			Message string   `json:"message"`
			Details []string `json:"details"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &arcErr) == nil && arcErr.Error != nil {
		msg := strings.TrimSpace(arcErr.Error.Message)
		if len(arcErr.Error.Details) > 0 {
			msg += ": " + strings.Join(arcErr.Error.Details, " | ")
		}
		return nil, fmt.Errorf("SIGEF público: %s", msg)
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, fmt.Errorf("SIGEF público retornou GeoJSON inválido: %w", err)
	}

	carMetric, metricErr := projectAreaMetrics(carRaw)
	carAreaHa := 0.0
	if metricErr == nil {
		carAreaHa = carMetric.AreaHa
	}

	seen := map[string]bool{}
	out := make([]SIGEFParcel, 0, len(fc.Features))
	for _, feature := range fc.Features {
		raw, _ := json.Marshal(carGeoFeature{
			Type:       "Feature",
			Properties: feature.Properties,
			Geometry:   feature.Geometry,
		})
		geo := string(raw)
		metric, err := projectAreaMetrics(geo)
		if err != nil || metric.AreaHa <= 0 {
			continue
		}
		intersection, parcelInside, carInside, err := estimateGeometryOverlap(metric.GeoJSON, carRaw)
		if err != nil || intersection <= 0.0001 {
			continue
		}

		props := feature.Properties
		p := SIGEFParcel{
			ParcelCode:         carStringProp(props, "parcela_co"),
			TechnicalRT:        carStringProp(props, "rt"),
			ART:                carStringProp(props, "art"),
			PropertySituation:  carStringProp(props, "situacao_i"),
			PropertyCode:       carStringProp(props, "codigo_imo"),
			SubmissionDate:     arcGISDateString(props["data_submi"]),
			ApprovalDate:       arcGISDateString(props["data_aprov"]),
			Status:             carStringProp(props, "status"),
			AreaName:           carStringProp(props, "nome_area"),
			Registry:           carStringProp(props, "registro_m"),
			RegistryDate:       arcGISDateString(props["registro_d"]),
			MunicipalityCode:   anyString(props, "municipio_"),
			UFID:               anyString(props, "uf_id"),
			AreaHa:             metric.AreaHa,
			IntersectionAreaHa: intersection,
			ParcelInsideCARPct: safePercent(parcelInside),
			CARInsideParcelPct: safePercent(carInside),
			GeoJSON:            metric.GeoJSON,
		}
		if carAreaHa > 0 {
			p.AreaDifferenceHa = math.Abs(p.AreaHa - carAreaHa)
			p.AreaDifferencePct = p.AreaDifferenceHa / carAreaHa * 100
		}
		setSIGEFComparison(&p)

		key := strings.Join([]string{
			strings.ToUpper(strings.TrimSpace(p.ParcelCode)),
			strings.ToUpper(strings.TrimSpace(p.PropertyCode)),
			strings.ToUpper(strings.TrimSpace(p.Registry)),
		}, "|")
		if key == "||" {
			key = fmt.Sprintf("%.6f|%.6f|%.4f", p.AreaHa, p.IntersectionAreaHa, p.CARInsideParcelPct)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CARInsideParcelPct == out[j].CARInsideParcelPct {
			if out[i].ParcelInsideCARPct == out[j].ParcelInsideCARPct {
				return out[i].IntersectionAreaHa > out[j].IntersectionAreaHa
			}
			return out[i].ParcelInsideCARPct > out[j].ParcelInsideCARPct
		}
		return out[i].CARInsideParcelPct > out[j].CARInsideParcelPct
	})
	return out, nil
}

func setSIGEFComparison(p *SIGEFParcel) {
	if p == nil {
		return
	}
	switch {
	case p.CARInsideParcelPct >= 95 && p.ParcelInsideCARPct >= 95:
		p.ComparisonLevel = "high"
		p.ComparisonSummary = "Alta coincidência espacial entre CAR e parcela SIGEF pública."
	case p.CARInsideParcelPct >= 80 || p.ParcelInsideCARPct >= 80:
		p.ComparisonLevel = "partial"
		p.ComparisonSummary = "Coincidência espacial parcial; confira áreas, retificações e documentos do imóvel."
	default:
		p.ComparisonLevel = "intersection"
		p.ComparisonSummary = "A parcela intersecta o CAR, mas a coincidência espacial é limitada."
	}
}
