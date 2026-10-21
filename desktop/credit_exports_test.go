package main

import (
	"strings"
	"testing"
)

func TestBuildSICORGlebasKML231(t *testing.T) {
	result := CreditIntelligenceResult{
		CAR: "MG-TESTE",
		Operations: []CreditOperationIntelligence{{
			RefBacen: "REF123", Order: "1", Institution: "Banco Teste",
			Program: "Programa Teste", Product: "Café", CreditValue: 100000,
			FinancedAreaHa: 5,
			Glebas: []SICORGleba{{
				RefBacen: "REF123", Order: "1", Index: "1", AreaHa: 4.75, InsideCARPct: 100,
				GeoJSON: `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.0,-20.0],[-45.9,-20.0],[-45.9,-20.1],[-46.0,-20.1],[-46.0,-20.0]]]}}`,
			}},
		}},
	}
	kml, glebas, ops := buildSICORGlebasKML(result)
	if glebas != 1 || ops != 1 {
		t.Fatalf("contagem inesperada: glebas=%d ops=%d", glebas, ops)
	}
	s := string(kml)
	for _, marker := range []string{"<kml", "REF123", "Gleba 1", "Banco Teste", "area_gleba_ha", "-46"} {
		if !strings.Contains(s, marker) {
			t.Fatalf("KML SICOR sem marcador %q", marker)
		}
	}
}

func TestBuildSICORGlebasKML231IgnoresMissingGeometry(t *testing.T) {
	result := CreditIntelligenceResult{
		CAR: "MG-TESTE",
		Operations: []CreditOperationIntelligence{{
			RefBacen:"REF", Order:"1", Glebas:[]SICORGleba{{Index:"1",AreaHa:10}},
		}},
	}
	_, glebas, ops := buildSICORGlebasKML(result)
	if glebas != 0 || ops != 0 {
		t.Fatalf("gleba sem geometria não deve entrar no KML: glebas=%d ops=%d", glebas, ops)
	}
}

func TestBuildCreditIntelligencePDF231(t *testing.T) {
	p := Property{Name:"Fazenda Teste",ClientName:"Produtor Teste",Municipality:"Jacuí",UF:"MG",Registry:"12345"}
	result := CreditIntelligenceResult{
		CAR:"MG-TESTE",Municipality:"Jacuí",UF:"MG",GeneratedAt:"2026-10-21T14:00:00-03:00",
		Scope:"Informações públicas SICOR/BCB.",
		Totals:CreditIntelligenceTotals{Operations:1,Contracted:100000,Released:80000,WithBalance:1},
		BCBSourceURL:"https://www.bcb.gov.br/",MCRSourceURL:"https://www3.bcb.gov.br/mcr/completo",ZARCSourceURL:"https://dados.agricultura.gov.br/",
		Operations:[]CreditOperationIntelligence{{
			RefBacen:"REF123",EffectiveRefBacen:"REF-EF",Order:"1",Year:2026,
			Institution:"Banco Teste",Program:"Programa",Subprogram:"Sub",Resource:"Recurso",
			Purpose:"Investimento",Activity:"Agrícola",Modality:"Investimento",Product:"Café",Variety:"Catuaí",
			IssueDate:"2026-01-10",DueDate:"2030-01-10",CreditValue:100000,OwnResources:10000,
			FinancedAreaHa:5,InterestRatePct:8.5,TotalReleased:80000,
			Balance:CreditBalance{Found:true,Year:2026,Month:9,LastDay:60000,Status:"NORMAL"},
			Releases:[]CreditRelease{{Date:"2026-02-01",Value:80000}},
			Disbursements:[]CreditDisbursement{{Date:"2026-02-01",Value:80000}},
			Proagro:CreditProagro{HasCOP:true,COPDate:"2026-03-01",COPStatus:"COMUNICADO"},
			ZARC:ZARCCheck{Status:"Indicação localizada",Safra:"2025/2026",Culture:"Café",Message:"Teste"},
			Glebas:[]SICORGleba{{Index:"1",AreaHa:4.75,InsideCARPct:100}},
		}},
	}
	pdf := buildCreditIntelligencePDF(p,result)
	if len(pdf) < 2500 || !strings.HasPrefix(string(pdf), "%PDF-1.4") {
		t.Fatalf("PDF de crédito inválido, tamanho=%d", len(pdf))
	}
	raw := string(pdf)
	for _, marker := range []string{"REF123", "Banco Teste", "Catua", "SICOR/BCB"} {
		if !strings.Contains(raw, marker) {
			t.Fatalf("PDF completo sem marcador %q", marker)
		}
	}
}
