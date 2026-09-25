package main

import (
	"strings"
	"testing"
)

func TestProfessionalCARPDF205(t *testing.T) {
	p := Property{Name: "Fazenda Teste", ClientName: "Cliente Teste", Municipality: "Jacuí", UF: "MG", Registry: "4020"}
	car := CARResult{
		CAR: "MG-0000000-AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA",
		Municipality: "Jacuí", UF: "MG", AreaHa: 97.80, GeometryAreaHa: 97.79,
		PerimeterM: 4200, FiscalModules: 1.09, Status: "Ativo", Condition: "Regular",
		Found: true, HasGeometry: true, LookupStatus: "ok", LookupDetail: "Consulta pública concluída.",
	}
	pdf := buildCARProfessionalPDF(p, car, KMLResult{}, GeometryComparison{})
	s := string(pdf)
	if !strings.HasPrefix(s, "%PDF-1.4") {
		t.Fatal("demonstrativo CAR não gerou PDF")
	}
	if !strings.Contains(s, "/Count 2") {
		t.Fatal("demonstrativo CAR profissional deve ter 2 páginas")
	}
	if !strings.Contains(s, "DEMONSTRATIVO TECNICO DO CAR") {
		t.Fatal("título profissional ausente")
	}
}

func TestEnvironmentalEvidencePDF205(t *testing.T) {
	p := Property{Name: "Fazenda Teste", ClientName: "Cliente Teste"}
	car := CARResult{CAR: "MG-0000000-AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA", AreaHa: 20, Found: true}
	intel := EnvironmentalIntelligenceResult{
		CAR: car.CAR, PropertyAreaHa: 20,
		MapBiomas: MapBiomasCARSummary{Available: true, Connected: true},
		Summary: EnvironmentalEvidenceSummary{},
	}
	pdf := buildEnvironmentalEvidencePDF(p, car, intel)
	s := string(pdf)
	if !strings.HasPrefix(s, "%PDF-1.4") {
		t.Fatal("caderno de evidências não gerou PDF")
	}
	if !strings.Contains(s, "/Count 3") {
		t.Fatal("caderno de evidências deve ter 3 páginas")
	}
	if !strings.Contains(s, "CADERNO DE EVIDENCIAS AMBIENTAIS") {
		t.Fatal("título do caderno de evidências ausente")
	}
}

func TestPropertyTechnicalDossierPDF205(t *testing.T) {
	p := Property{Name: "Fazenda Teste", ClientName: "Cliente Teste", Municipality: "Jacuí", UF: "MG"}
	r := CARAutomationResult{
		OverallStatus: "complete",
		ExecutiveSummary: "Resumo técnico de teste.",
		CAR: CARResult{
			CAR: "MG-0000000-AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA",
			Municipality: "Jacuí", UF: "MG", AreaHa: 97.8, Found: true, Status: "Ativo",
		},
		XRay: PropertyXRay{
			SICOR: SICORXRayResult{OperationCount: 1, TotalCreditValue: 150000},
			SIGEF: SIGEFPublicResult{Available: true, ParcelCount: 1, BestCARCoveragePct: 99.8},
		},
		Environmental: EnvironmentalIntelligenceResult{
			Summary: EnvironmentalEvidenceSummary{},
			MapBiomas: MapBiomasCARSummary{Available: true, Connected: true},
		},
		BCB: BCBPublicContext{Available: true, Municipality: "Jacuí", UF: "MG"},
	}
	pdf := buildPropertyTechnicalDossierPDF(p, r, KMLResult{}, GeometryComparison{})
	s := string(pdf)
	if !strings.HasPrefix(s, "%PDF-1.4") {
		t.Fatal("dossiê técnico não gerou PDF")
	}
	if !strings.Contains(s, "/Count 7") {
		t.Fatal("dossiê técnico deve ter 7 páginas")
	}
	for _, marker := range []string{"DOSSIE TECNICO DO IMOVEL", "Raio X ambiental", "SIGEF / INCRA", "Banco Central"} {
		if !strings.Contains(s, marker) {
			t.Fatalf("dossiê sem seção %q", marker)
		}
	}
}

func TestReportAutomationStatus205(t *testing.T) {
	cases := map[string]string{
		"ok": "Sem ocorrência",
		"hit": "Ocorrência encontrada",
		"unavailable": "Base indisponível",
		"partial": "Consulta parcial",
	}
	for in, want := range cases {
		if got := reportAutomationStatus(in); got != want {
			t.Fatalf("%s: esperado %q, obtido %q", in, want, got)
		}
	}
}
