package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestEnvironmentalAttention180(t *testing.T) {
	item := EnvironmentalAlertDetail{
		AlertCode:    "123",
		AreaHa:       10,
		APPOverlapHa: 1.25,
	}
	level, reasons := environmentalAttention(item, false)
	if level != "Alta prioridade de conferência" {
		t.Fatalf("nível inesperado: %s", level)
	}
	if len(reasons) == 0 || !strings.Contains(strings.ToLower(reasons[0]), "app") {
		t.Fatalf("motivo de APP não registrado: %#v", reasons)
	}

	item = EnvironmentalAlertDetail{AlertCode: "456", AreaHa: 4, AlertAreaInCAR: 3}
	level, reasons = environmentalAttention(item, false)
	if level != "Conferir" {
		t.Fatalf("alerta simples deveria exigir conferência, obteve %s", level)
	}
	if len(reasons) == 0 {
		t.Fatal("esperava motivo de conferência")
	}

	item = EnvironmentalAlertDetail{AlertCode: "789", AreaHa: 2}
	level, _ = environmentalAttention(item, true)
	if level != "Alta prioridade de conferência" {
		t.Fatalf("CAR na lista MCR deveria elevar prioridade, obteve %s", level)
	}
}

func TestSummarizeEnvironmentalEvidence180(t *testing.T) {
	items := []EnvironmentalAlertDetail{
		{
			AlertCode:"1", AreaHa:10, AlertAreaInCAR:8,
			APPOverlapHa:1.5, RLOverlapHa:2,
			IBAMAOverlapHa:.2, AttentionLevel:"Alta prioridade de conferência",
			DetectedAt:"2026-08-10",
		},
		{
			AlertCode:"2", AreaHa:4, AlertAreaInCAR:4,
			RLOverlapHa:.5, FederalUCOverlapHa:.1,
			AttentionLevel:"Conferir", DetectedAt:"2026-09-01",
		},
	}
	got := summarizeEnvironmentalEvidence(items)
	if got.Alerts != 2 || got.AlertAreaTotalHa != 14 || got.AlertAreaInCARHa != 12 {
		t.Fatalf("totais inesperados: %+v", got)
	}
	if got.AlertsOverAPP != 1 || got.APPOverlapHa != 1.5 {
		t.Fatalf("APP inesperada: %+v", got)
	}
	if got.AlertsOverRL != 2 || got.RLOverlapHa != 2.5 {
		t.Fatalf("RL inesperada: %+v", got)
	}
	if got.AlertsOverIBAMA != 1 || got.AlertsOverFederalUC != 1 {
		t.Fatalf("camadas sensíveis inesperadas: %+v", got)
	}
	if got.HighAttentionAlerts != 1 || got.ReviewAlerts != 1 {
		t.Fatalf("prioridades inesperadas: %+v", got)
	}
	if got.LatestDetection != "2026-09-01" {
		t.Fatalf("última detecção inesperada: %s", got.LatestDetection)
	}
}

func TestStripWKTSpatialReference180(t *testing.T) {
	got := stripWKTSpatialReference("SRID=4674;POLYGON((-46 -20,-45 -20,-45 -19,-46 -20))")
	if !strings.HasPrefix(got, "POLYGON(") {
		t.Fatalf("SRID não removido: %s", got)
	}
	plain := "POLYGON((-46 -20,-45 -20,-45 -19,-46 -20))"
	if stripWKTSpatialReference(plain) != plain {
		t.Fatal("WKT simples foi alterado")
	}
}

func TestEnvironmentalReportPDF180(t *testing.T) {
	p := Property{
		ID:1, Name:"Fazenda Teste", ClientName:"Produtor Teste",
		Municipality:"Goiatuba", UF:"GO",
	}
	car := CARResult{
		CAR:"GO-5209101-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Municipality:"Goiatuba", UF:"GO", AreaHa:100, GeometryAreaHa:99.5,
		Status:"Ativo",
	}
	alert := EnvironmentalAlertDetail{
		AlertCode:"12345", AreaHa:5, AlertAreaInCAR:4.5, AlertPctOfCAR:4.52,
		DetectedAt:"2026-07-10", PublishedAt:"2026-07-20",
		Sources:[]string{"DETER"}, AttentionLevel:"Conferir",
		AttentionReasons:[]string{"alerta MapBiomas vinculado ao imóvel"},
	}
	intel := EnvironmentalIntelligenceResult{
		CAR:car.CAR, Municipality:"Goiatuba", UF:"GO", PropertyAreaHa:99.5,
		MapBiomas:MapBiomasCARSummary{Connected:true, TotalAlerts:1, TotalAreaHa:5},
		Alerts:[]EnvironmentalAlertDetail{alert},
		Summary:summarizeEnvironmentalEvidence([]EnvironmentalAlertDetail{alert}),
		Interpretation:"Triagem técnica auxiliar.",
	}
	pdf := buildEnvironmentalTechnicalPDF(p,car,intel,"")
	if len(pdf) < 1000 {
		t.Fatalf("PDF muito pequeno: %d bytes", len(pdf))
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatal("cabeçalho PDF inválido")
	}
	if !bytes.Contains(pdf, []byte("/Count 4")) {
		t.Fatal("esperava 4 páginas no laudo consolidado")
	}

	single := buildEnvironmentalTechnicalPDF(p,car,intel,"12345")
	if !bytes.Contains(single, []byte("/Count 4")) {
		t.Fatal("laudo individual deveria manter capa, mapa, alerta e metodologia")
	}
}

func TestEnvironmentalCacheRoundTrip180(t *testing.T) {
	path := t.TempDir()+"/env.json"
	in := EnvironmentalIntelligenceResult{
		CAR:"MG-0000000-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Municipality:"Teste", UF:"MG",
		Alerts:[]EnvironmentalAlertDetail{{AlertCode:"10",AreaHa:2}},
	}
	if err:=saveEnvironmentalIntelligenceCache(path,in);err!=nil{t.Fatal(err)}
	got,ok:=loadEnvironmentalIntelligenceCache(path,environmentalIntelligenceCacheAge)
	if !ok || got.CAR!=in.CAR || len(got.Alerts)!=1 {
		t.Fatalf("cache inesperado: ok=%v got=%+v",ok,got)
	}
}
