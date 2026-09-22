package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type PropertyXRaySummary struct {
	PublicCreditOperations int     `json:"public_credit_operations"`
	PublicCreditValue      float64 `json:"public_credit_value"`
	FinancedGlebas         int     `json:"financed_glebas"`
	GlebasWithProjectHit   int     `json:"glebas_with_project_hit"`
	MapBiomasAlerts        int     `json:"mapbiomas_alerts"`
	EnvironmentalHits      int     `json:"environmental_hits"`
	AvailableSICARThemes   int     `json:"available_sicar_themes"`
	ProjectAreas           int     `json:"project_areas"`
	SIGEFParcels           int     `json:"sigef_parcels"`
	SIGEFRegistries        int     `json:"sigef_registries"`
	SIGEFBestCARCoverage   float64 `json:"sigef_best_car_coverage"`
}

type PropertyXRay struct {
	GeneratedAt string                 `json:"generated_at"`
	CAR         CARResult              `json:"car"`
	SICOR       SICORXRayResult        `json:"sicor"`
	SIGEF       SIGEFPublicResult       `json:"sigef"`
	MapBiomas   MapBiomasCARSummary    `json:"mapbiomas"`
	Areas       []ProjectArea          `json:"areas"`
	Route       AccessRoute            `json:"route"`
	Summary     PropertyXRaySummary     `json:"summary"`
	Warnings    []string               `json:"warnings"`
}

func (a *App) BuildPropertyXRay(propertyID int64, force bool) (PropertyXRay, error) {
	car, err := a.carForRuralCredit(propertyID)
	if err != nil {
		return PropertyXRay{}, err
	}
	if !car.Found || strings.TrimSpace(car.GeoJSON) == "" {
		return PropertyXRay{}, errors.New("consulte o CAR com geometria antes de montar o Raio X")
	}

	out := PropertyXRay{GeneratedAt: time.Now().Format(time.RFC3339), CAR: car}

	if propertyID > 0 {
		out.Areas, _ = a.ListProjectAreas(propertyID)
		if route, routeErr := a.GetAccessRoute(propertyID); routeErr == nil {
			out.Route = route
		}
	} else {
		if area, areaErr := a.GetTemporaryProjectArea(); areaErr == nil {
			out.Areas = []ProjectArea{area}
		}
		out.Route = car.AutoRoute
	}

	type sicorResult struct {
		v SICORXRayResult
		e error
	}
	type alertResult struct {
		v MapBiomasCARSummary
		e error
	}
	type sigefResult struct {
		v SIGEFPublicResult
		e error
	}
	sicorCh := make(chan sicorResult, 1)
	alertCh := make(chan alertResult, 1)
	sigefCh := make(chan sigefResult, 1)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				sicorCh <- sicorResult{e: fmt.Errorf("falha isolada no SICOR: %v", r)}
			}
		}()
		v, e := a.BuildSICORPropertyXRay(propertyID, force)
		sicorCh <- sicorResult{v: v, e: e}
	}()
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				alertCh <- alertResult{e: fmt.Errorf("falha isolada no MapBiomas: %v", r)}
			}
		}()
		v, e := a.QueryMapBiomasCAR(car.CAR)
		alertCh <- alertResult{v: v, e: e}
	}()
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				sigefCh <- sigefResult{e: fmt.Errorf("falha isolada no SIGEF público: %v", r)}
			}
		}()
		v, e := a.QuerySIGEFPublic(propertyID)
		sigefCh <- sigefResult{v: v, e: e}
	}()

	wg.Wait()
	sicor := <-sicorCh
	alerts := <-alertCh
	sigef := <-sigefCh
	if sicor.e != nil {
		out.Warnings = append(out.Warnings, "SICOR: "+sicor.e.Error())
	} else {
		out.SICOR = sicor.v
	}
	if alerts.e != nil {
		out.Warnings = append(out.Warnings, "MapBiomas Alerta: "+alerts.e.Error())
		out.MapBiomas = MapBiomasCARSummary{Connected: a.GetMapBiomasAlertStatus().Connected, Message: alerts.e.Error()}
	} else {
		out.MapBiomas = alerts.v
	}
	if sigef.e != nil {
		out.Warnings = append(out.Warnings, "SIGEF/INCRA: "+sigef.e.Error())
		out.SIGEF = SIGEFPublicResult{SourceURL: sigefPublicSourceURL, Message: "Fonte pública do SIGEF indisponível nesta tentativa."}
	} else {
		out.SIGEF = sigef.v
	}

	out.Summary.PublicCreditOperations = out.SICOR.OperationCount
	out.Summary.PublicCreditValue = out.SICOR.TotalCreditValue
	out.Summary.FinancedGlebas = out.SICOR.GlebaCount
	out.Summary.MapBiomasAlerts = out.MapBiomas.TotalAlerts
	out.Summary.ProjectAreas = len(out.Areas)
	out.Summary.SIGEFParcels = out.SIGEF.ParcelCount
	out.Summary.SIGEFRegistries = out.SIGEF.RegistryCount
	out.Summary.SIGEFBestCARCoverage = out.SIGEF.BestCARCoveragePct

	for _, op := range out.SICOR.Operations {
		for _, g := range op.Glebas {
			if g.ProjectOverlapPct > 0.5 {
				out.Summary.GlebasWithProjectHit++
			}
		}
	}
	env := car.Environment
	out.Summary.EnvironmentalHits =
		env.IBAMAEmbargoCount + env.IndigenousCount + env.FederalUCCount
	if env.MCRListed {
		out.Summary.EnvironmentalHits++
	}
	for _, theme := range car.Themes.Themes {
		if theme.Available {
			out.Summary.AvailableSICARThemes++
		}
	}

	out.Warnings = append(out.Warnings, out.SICOR.Warnings...)
	out.Warnings = append(out.Warnings, out.SIGEF.Warnings...)
	return out, nil
}
