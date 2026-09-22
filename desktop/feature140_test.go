package main

import (
	"testing"
)

func TestFilterBCBMunicipalityRowsByIBGE(t *testing.T) {
	rows := []map[string]any{
		{"codMunicIbge":"3107109","Municipio":"BOM JESUS DO AMPARO","nomeUF":"MG","AnoEmissao":"2026","Atividade":"1"},
		{"codMunicIbge":"3550308","Municipio":"SAO PAULO","nomeUF":"SP","AnoEmissao":"2026","Atividade":"1"},
		{"codMunicIbge":"3107109","Municipio":"BOM JESUS DO AMPARO","nomeUF":"MG","AnoEmissao":"2025","Atividade":"2"},
	}
	got := filterBCBMunicipalityRows(rows, "Bom Jesus do Amparo", "MG", "3107109", "2026")
	if len(got) != 1 {
		t.Fatalf("esperava 1 linha do município/ano, obteve %d: %#v", len(got), got)
	}
	if firstStringMapValue(got[0], "codMunicIbge") != "3107109" {
		t.Fatalf("linha errada: %#v", got[0])
	}
}

func TestAggregateBCBMunicipalityRows(t *testing.T) {
	rows := []map[string]any{
		{
			"AnoEmissao":"2026","Atividade":"1",
			"QtdCusteio":3.0,"VlCusteio":100000.0,
			"QtdInvestimento":2.0,"VlInvestimento":250000.0,
		},
		{
			"AnoEmissao":"2026","Atividade":"1",
			"QtdCusteio":1.0,"VlCusteio":50000.0,
			"QtdInvestimento":0.0,"VlInvestimento":0.0,
		},
	}
	got := aggregateBCBMunicipalityRows(rows)
	if len(got) != 2 {
		t.Fatalf("esperava custeio e investimento, obteve %d: %#v", len(got), got)
	}
	var custeio, investimento *BCBRuralCreditRow
	for i := range got {
		switch got[i].Kind {
		case "Custeio":
			custeio = &got[i]
		case "Investimento":
			investimento = &got[i]
		}
	}
	if custeio == nil || custeio.Contracts != 4 || custeio.Value != 150000 || custeio.Product != "Atividade agrícola" {
		t.Fatalf("custeio agregado incorreto: %#v", custeio)
	}
	if investimento == nil || investimento.Contracts != 2 || investimento.Value != 250000 {
		t.Fatalf("investimento agregado incorreto: %#v", investimento)
	}
}

func TestBCBNumberMapValue(t *testing.T) {
	m := map[string]any{"n": 1234.5, "s": "1234,50"}
	if got := numberMapValue(m,"n"); got != 1234.5 {
		t.Fatalf("número JSON inesperado: %f", got)
	}
	if got := numberMapValue(m,"s"); got != 1234.5 {
		t.Fatalf("número textual inesperado: %f", got)
	}
}

func TestMapBiomasAlertStatusDisconnected(t *testing.T) {
	app := &App{}
	s := app.GetMapBiomasAlertStatus()
	if s.Connected {
		t.Fatal("app sem banco/token não pode aparecer conectado")
	}
}


func TestMapBiomasGraphQLErrorSchemaFallbackDetection(t *testing.T) {
	errs := []graphQLError{{Message: "Field 'coordinates' doesn't exist on type 'AlertData'"}}
	if !hasGraphQLErrorContaining(errs, "coordinates", "AlertData") {
		t.Fatal("erro de schema deveria acionar fallback de coordenadas")
	}
}

func TestGraphQLScalarStringAcceptsStringAndNumber(t *testing.T) {
	if got := graphQLScalarString("12345"); got != "12345" {
		t.Fatalf("string inesperada: %s", got)
	}
	if got := graphQLScalarString(float64(12345)); got != "12345" {
		t.Fatalf("número inesperado: %s", got)
	}
}


func TestSICARThemeRemoteCandidatesAPP(t *testing.T) {
	got := sicarThemeRemoteCandidates("APP")
	if len(got) != 2 || got[0] != "APP" || got[1] != "APPS" {
		t.Fatalf("aliases APP inesperados: %#v", got)
	}
	got = sicarThemeRemoteCandidates("RESERVA_LEGAL")
	if len(got) != 1 || got[0] != "RESERVA_LEGAL" {
		t.Fatalf("tema sem alias alterado indevidamente: %#v", got)
	}
}


func TestBCBNumberMapValueDecimalFormats(t *testing.T) {
	m := map[string]any{
		"dot": "1234.50",
		"br":  "1.234,50",
	}
	if got := numberMapValue(m, "dot"); got != 1234.5 {
		t.Fatalf("decimal com ponto inesperado: %f", got)
	}
	if got := numberMapValue(m, "br"); got != 1234.5 {
		t.Fatalf("decimal pt-BR inesperado: %f", got)
	}
}

func TestFilterBCBMunicipalityRowsRejectsOtherMunicipality(t *testing.T) {
	rows := []map[string]any{
		{"codMunicIbge":"3550308","Municipio":"SAO PAULO","nomeUF":"SP","AnoEmissao":"2026"},
	}
	got := filterBCBMunicipalityRows(rows, "Bom Jesus do Amparo", "MG", "3107703", "2026")
	if len(got) != 0 {
		t.Fatalf("não deveria aceitar linha de outro município: %#v", got)
	}
}


func TestAggregateBCBMunicipalityRowsIncludesAllFourPurposes(t *testing.T) {
	rows := []map[string]any{{
		"AnoEmissao":"2026","Atividade":"2",
		"QtdCusteio":1.0,"VlCusteio":100.0,
		"QtdInvestimento":2.0,"VlInvestimento":200.0,
		"QtdComercializacao":3.0,"VlComercializacao":300.0,
		"QtdIndustrializacao":4.0,"VlIndustrializacao":400.0,
	}}
	got := aggregateBCBMunicipalityRows(rows)
	if len(got) != 4 {
		t.Fatalf("esperava as quatro finalidades do crédito rural, obteve %d: %#v", len(got), got)
	}
	totalQty, totalValue := 0.0, 0.0
	kinds := map[string]bool{}
	for _, row := range got {
		kinds[row.Kind] = true
		totalQty += row.Contracts
		totalValue += row.Value
	}
	for _, kind := range []string{"Custeio","Investimento","Comercialização","Industrialização"} {
		if !kinds[kind] { t.Fatalf("finalidade ausente: %s", kind) }
	}
	if totalQty != 10 || totalValue != 1000 {
		t.Fatalf("totais incorretos: quantidade=%v valor=%v", totalQty, totalValue)
	}
}
