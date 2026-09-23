package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
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


func TestSICARThemePayloadDetection181(t *testing.T) {
	if !hasZIPSignature([]byte{'P','K',3,4,0,0}) {
		t.Fatal("assinatura ZIP válida não reconhecida")
	}
	if hasZIPSignature([]byte("<html><body>erro</body></html>")) {
		t.Fatal("HTML não pode ser reconhecido como ZIP")
	}
	desc := describeSICARThemePayload([]byte("<!doctype html><html>serviço indisponível</html>"), "text/html; charset=utf-8")
	if !strings.Contains(strings.ToLower(desc), "html") {
		t.Fatalf("diagnóstico de HTML inesperado: %s", desc)
	}
	desc = describeSICARThemePayload([]byte(`{"erro":"temporario"}`), "application/json")
	if !strings.Contains(strings.ToLower(desc), "json") {
		t.Fatalf("diagnóstico de JSON inesperado: %s", desc)
	}
}

func TestEnvironmentalThemeUnavailableIsNotZero181(t *testing.T) {
	themes := SICARThemesSummary{Themes: map[string]SICARThemeMetric{
		"APP": {Code:"APP", Label:"Área de Preservação Permanente", Available:false, Status:"unavailable", Error:"fonte indisponível"},
		"RESERVA_LEGAL": {Code:"RESERVA_LEGAL", Label:"Reserva Legal", Available:true, AreaHa:0, Status:"online", CacheStatus:"online"},
	}}
	value, detail := environmentalThemeOverlapLabel(themes, "APP", 0)
	if value != "Indisponível" {
		t.Fatalf("fonte indisponível foi apresentada como %q", value)
	}
	if !strings.Contains(strings.ToLower(detail), "não obtida") {
		t.Fatalf("detalhe inesperado: %s", detail)
	}
	value, _ = environmentalThemeOverlapLabel(themes, "RESERVA_LEGAL", 0)
	if value == "Indisponível" {
		t.Fatal("zero confirmado em camada disponível foi confundido com indisponibilidade")
	}
	if !strings.Contains(value, "0") {
		t.Fatalf("zero confirmado não foi mantido: %s", value)
	}
}

func TestThemeSourceSummary181(t *testing.T) {
	allUnavailable := SICARThemesSummary{Themes: map[string]SICARThemeMetric{
		"APP": {Available:false},
		"RESERVA_LEGAL": {Available:false},
	}}
	got := themeSourceSummary(allUnavailable)
	if !strings.Contains(got, "0/6") || !strings.Contains(strings.ToLower(got), "sem assumir área zero") {
		t.Fatalf("resumo indisponível inadequado: %s", got)
	}

	mixed := SICARThemesSummary{Themes: map[string]SICARThemeMetric{
		"APP": {Available:true, CacheStatus:"cache_stale"},
		"RESERVA_LEGAL": {Available:true, CacheStatus:"online"},
	}}
	got = themeSourceSummary(mixed)
	if !strings.Contains(got, "2/6") || !strings.Contains(strings.ToLower(got), "cache") {
		t.Fatalf("resumo de cache inadequado: %s", got)
	}
}


func TestSICARHTMLRequiresManualValidation183(t *testing.T) {
	html := []byte("<!doctype html><html><body>Base de Downloads</body></html>")
	if !sicarThemePayloadRequiresHumanValidation(html, "text/html; charset=UTF-8") {
		t.Fatal("HTML do portal deveria exigir validação humana")
	}
	if sicarThemePayloadRequiresHumanValidation([]byte(`{"erro":"temporario"}`), "application/json") {
		t.Fatal("JSON de erro não deve ser confundido automaticamente com CAPTCHA")
	}
}

func TestRebuildSICARWarningsCollapsesManualRequirement183(t *testing.T) {
	s := SICARThemesSummary{Themes: map[string]SICARThemeMetric{
		"APP": {Code:"APP", Status:"manual_required", Available:false},
		"RESERVA_LEGAL": {Code:"RESERVA_LEGAL", Status:"manual_required", Available:false},
		"AREA_CONSOLIDADA": {Code:"AREA_CONSOLIDADA", Status:"manual_required", Available:false},
	}}
	got := rebuildSICARThemeWarnings(s)
	if len(got) != 1 {
		t.Fatalf("esperava um aviso agrupado, obteve %#v", got)
	}
	if !strings.Contains(strings.ToLower(got[0]), "captcha") {
		t.Fatalf("aviso deveria orientar sobre CAPTCHA: %s", got[0])
	}
}

func TestSICARThemeZipCompatibility183(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tema.zip")
	f, err := os.Create(path)
	if err != nil { t.Fatal(err) }
	zw := zip.NewWriter(f)
	w, err := zw.Create("MG_APP.shp")
	if err != nil { t.Fatal(err) }
	_, _ = w.Write([]byte("teste"))
	if err := zw.Close(); err != nil { t.Fatal(err) }
	if err := f.Close(); err != nil { t.Fatal(err) }

	if !sicarThemeZipLooksCompatible(path, "APP") {
		t.Fatal("ZIP de APP não reconhecido")
	}
	if sicarThemeZipLooksCompatible(path, "RESERVA_LEGAL") {
		t.Fatal("ZIP de APP não pode ser aceito como Reserva Legal")
	}
}


func TestNormalizeLegacySICARThemes184(t *testing.T) {
	in := SICARThemesSummary{
		Themes: map[string]SICARThemeMetric{
			"APP": {
				Code: "APP", Label: "Área de Preservação Permanente",
				Status: "unavailable", Available: false,
				Error: "GeoServices não entregou pacote utilizável (HTTP nativo: GeoServices respondeu conteúdo que não é ZIP (página HTML/erro do serviço; text/html; charset=UTF-8))",
			},
			"RESERVA_LEGAL": {
				Code: "RESERVA_LEGAL", Label: "Reserva Legal",
				Status: "unavailable", Available: false,
				Error: "falha de rede sem resposta HTTP",
			},
		},
		Warnings: []string{
			"Área de Preservação Permanente: GeoServices não entregou pacote utilizável (text/html)",
			"Reserva Legal: falha de rede sem resposta HTTP",
		},
	}
	got := normalizeLegacySICARThemes(in)
	if got.Themes["APP"].Status != "manual_required" {
		t.Fatalf("APP legada deveria migrar para manual_required: %+v", got.Themes["APP"])
	}
	if got.Themes["RESERVA_LEGAL"].Status != "unavailable" {
		t.Fatalf("falha de rede genérica não deve ser rotulada como CAPTCHA: %+v", got.Themes["RESERVA_LEGAL"])
	}
}

func TestNormalizeLegacyEnvironmentalCache184(t *testing.T) {
	in := EnvironmentalIntelligenceResult{
		CAR: "MG-0000000-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Themes: SICARThemesSummary{
			Themes: map[string]SICARThemeMetric{
				"APP": {Code:"APP", Label:"Área de Preservação Permanente", Status:"unavailable", Error:"GeoServices respondeu conteúdo que não é ZIP (página HTML/erro do serviço; text/html)"},
			},
			Warnings: []string{"Área de Preservação Permanente: GeoServices respondeu conteúdo que não é ZIP (text/html)"},
		},
		Warnings: []string{
			"Nenhum alerta foi retornado para o CAR nesta consulta. Isso não equivale a certificado de regularidade ambiental.",
			"Área de Preservação Permanente: GeoServices respondeu conteúdo que não é ZIP (text/html)",
		},
	}
	got := normalizeLegacyEnvironmentalCache(in)
	if got.Themes.Themes["APP"].Status != "manual_required" {
		t.Fatal("cache antigo não foi migrado")
	}
	if len(got.Warnings) != 2 {
		t.Fatalf("esperava aviso MapBiomas + aviso SICAR agrupado, obteve %#v", got.Warnings)
	}
	for _, w := range got.Warnings {
		if strings.Contains(strings.ToLower(w), "conteúdo que não é zip") {
			t.Fatalf("erro técnico legado não deveria permanecer: %s", w)
		}
	}
}
