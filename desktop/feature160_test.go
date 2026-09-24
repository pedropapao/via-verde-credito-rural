package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseSIGEFPublicGeoJSON(t *testing.T) {
	car := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.62,-20.92],[-46.58,-20.92],[-46.58,-20.88],[-46.62,-20.88],[-46.62,-20.92]]]}}`
	body := []byte(`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{"parcela_co":"PARC-001","rt":"1234","art":"ART-99","situacao_i":"CERTIFICADO","codigo_imo":"1234567890123","data_submi":1704067200000,"data_aprov":1706745600000,"status":"ATIVO","nome_area":"FAZENDA TESTE","registro_m":"MATRICULA 4020","registro_d":1709251200000,"municipio_":3107703,"uf_id":31},"geometry":{"type":"Polygon","coordinates":[[[-46.615,-20.915],[-46.585,-20.915],[-46.585,-20.885],[-46.615,-20.885],[-46.615,-20.915]]]}}]}`)
	got, err := parseSIGEFPublicGeoJSON(body, car)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("esperava 1 parcela, obteve %d", len(got))
	}
	p := got[0]
	if p.ParcelCode != "PARC-001" || p.Registry != "MATRICULA 4020" || p.PropertyCode != "1234567890123" {
		t.Fatalf("atributos SIGEF incorretos: %+v", p)
	}
	if p.AreaHa <= 0 || p.IntersectionAreaHa <= 0 || p.CARInsideParcelPct <= 0 || p.ParcelInsideCARPct <= 0 {
		t.Fatalf("métricas espaciais inválidas: %+v", p)
	}
	if p.ApprovalDate == "" || p.RegistryDate == "" {
		t.Fatalf("datas públicas não foram convertidas: %+v", p)
	}
}

func TestParseSIGEFPublicRejectsArcGISError(t *testing.T) {
	_, err := parseSIGEFPublicGeoJSON([]byte(`{"error":{"message":"Invalid query","details":["bad geometry"]}}`), `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[]}}`)
	if err == nil || !strings.Contains(err.Error(), "Invalid query") {
		t.Fatalf("esperava erro ArcGIS explícito, obteve %v", err)
	}
}

func TestSIGEFComparisonLevels(t *testing.T) {
	p := SIGEFParcel{CARInsideParcelPct: 98, ParcelInsideCARPct: 97}
	setSIGEFComparison(&p)
	if p.ComparisonLevel != "high" {
		t.Fatalf("esperava alta coincidência, obteve %+v", p)
	}

	p = SIGEFParcel{CARInsideParcelPct: 84, ParcelInsideCARPct: 60}
	setSIGEFComparison(&p)
	if p.ComparisonLevel != "partial" {
		t.Fatalf("esperava coincidência parcial, obteve %+v", p)
	}

	p = SIGEFParcel{CARInsideParcelPct: 25, ParcelInsideCARPct: 20}
	setSIGEFComparison(&p)
	if p.ComparisonLevel != "intersection" {
		t.Fatalf("esperava interseção limitada, obteve %+v", p)
	}
}


func TestSIGEFUnavailableErrorIsUserFriendly192(t *testing.T) {
	err := sigefPublicUnavailableError([]string{
		`cliente nativo falhou (Get "https://pamgia.ibama.gov.br/server/rest/services/...": context deadline exceeded)`,
		`fallback do Windows falhou (curl: (28) Operation timed out after 20015 milliseconds)`,
	})
	if err == nil {
		t.Fatal("esperava erro amigável de indisponibilidade")
	}
	got := err.Error()
	if got != sigefPublicUnavailableMessage {
		t.Fatalf("mensagem inesperada: %q", got)
	}
	for _, technical := range []string{"https://", "context deadline exceeded", "curl", "20015"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(technical)) {
			t.Fatalf("mensagem do usuário expôs detalhe técnico %q: %s", technical, got)
		}
	}
}

func TestSIGEFUnavailableUIIsNotZero192(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/ui150.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	required := []string{
		"const sigefAvailable=sigef.available===true;",
		"sigefAvailable?(s.sigef_parcels||0):'Indisponível'",
		"sigefAvailable?(sigef.parcel_count||0):'Indisponível'",
		"fonte pública sem resposta",
	}
	for _, want := range required {
		if !strings.Contains(js, want) {
			t.Fatalf("ui150.js não contém proteção SIGEF %q", want)
		}
	}
}
