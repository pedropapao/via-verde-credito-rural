package main

import (
	"strings"
	"testing"
)

func TestChooseWFSLayer190(t *testing.T) {
	types := []wfsFeatureType{
		{Name:"prodes:municipality", Title:"Municipality"},
		{Name:"prodes:yearly_deforestation", Title:"Yearly deforestation"},
		{Name:"prodes:grid", Title:"Grid"},
	}
	got := chooseWFSLayer(types, []string{"yearly_deforestation","deforestation","increment"}, []string{"municip","grid"})
	if got != "prodes:yearly_deforestation" {
		t.Fatalf("camada PRODES inesperada: %s", got)
	}
}

func TestNearestWorldCoverClass190(t *testing.T) {
	code, ok := nearestWorldCoverClass(0, 100, 0)
	if !ok || code != 10 {
		t.Fatalf("cobertura arbórea não reconhecida: code=%d ok=%v", code, ok)
	}
	_, ok = nearestWorldCoverClass(3, 3, 3)
	if ok {
		t.Fatal("pixel muito distante da paleta não deveria ser forçado para uma classe")
	}
}

func TestFeatureYear190(t *testing.T) {
	if got := featureYear(map[string]any{"year": 2024.0}); got != 2024 {
		t.Fatalf("ano numérico inesperado: %d", got)
	}
	if got := featureYear(map[string]any{"class_name": "desmatamento_2022"}); got != 2022 {
		t.Fatalf("ano no texto inesperado: %d", got)
	}
}

func TestNearbySourceStatus190(t *testing.T) {
	p := NearbyEnvironmentalProfile{
		Available:true, SearchRadiusKm:50,
		EmbargoChecked:true, EmbargoFound:false,
		IndigenousChecked:false,
		UCChecked:true, UCFound:true, NearestUCKm:12.3,
	}
	if !p.EmbargoChecked || p.IndigenousChecked || !p.UCChecked {
		t.Fatalf("flags de consulta inesperadas: %+v", p)
	}
	if !p.UCFound || p.NearestUCKm != 12.3 {
		t.Fatalf("ocorrência próxima inesperada: %+v", p)
	}
}

func TestEnvironmentalProfileSources190(t *testing.T) {
	p := EnvironmentalProfile{
		Terrain: TerrainMetric{Available:true},
		DETER: DeforestationProfile{Available:true, Applicable:false, Warning:"fora da cobertura"},
		Nearby: NearbyEnvironmentalProfile{Available:true},
	}
	sources := buildEnvironmentalProfileSources(p)
	if len(sources) < 8 {
		t.Fatalf("esperava matriz ampla de fontes, obteve %d", len(sources))
	}
	foundNonApplicable := false
	for _, s := range sources {
		if s.Key=="deter" {
			foundNonApplicable = true
			if s.Status!="não aplicável" {
				t.Fatalf("DETER deveria ser não aplicável: %+v", s)
			}
		}
	}
	if !foundNonApplicable {
		t.Fatal("fonte DETER ausente")
	}
}

func TestWorldCoverWarningIsApproximate190(t *testing.T) {
	p := LandCoverProfile{
		Approximate:true,
		Source:"ESA WorldCover 2021 v200 / Terrascope WMS (amostragem cartográfica automática)",
		Warning:"Percentuais representam a participação dos pontos amostrados na imagem WMS RGB; não substitui análise raster.",
	}
	if !p.Approximate || !strings.Contains(strings.ToLower(p.Warning),"amostr") {
		t.Fatalf("WorldCover precisa permanecer identificado como aproximado: %+v", p)
	}
}
