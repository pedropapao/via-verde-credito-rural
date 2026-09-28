package main

import (
	"strings"
	"testing"
)

func TestProfessionalCARPDF207(t *testing.T) {
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
	if !strings.Contains(s, "DEMONSTRATIVO T") || !strings.Contains(s, "DO CAR") {
		t.Fatal("título profissional ausente")
	}
}

func TestEnvironmentalEvidencePDF207(t *testing.T) {
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
	if !strings.Contains(s, "CADERNO DE EVID") || !strings.Contains(s, "AMBIENTAIS") {
		t.Fatal("título do caderno de evidências ausente")
	}
}

func TestPropertyTechnicalDossierPDF208(t *testing.T) {
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
	pdf := buildPropertyTechnicalDossierPDF(p, r, KMLResult{}, GeometryComparison{}, PropertyDocumentCenter{})
	s := string(pdf)
	if !strings.HasPrefix(s, "%PDF-1.4") {
		t.Fatal("dossiê técnico não gerou PDF")
	}
	if !strings.Contains(s, "/Count 8") {
		t.Fatal("dossiê técnico deve ter 8 páginas com Documentos e Pendências")
	}
	for _, marker := range []string{"DOSSI", "Raio X ambiental", "SIGEF / INCRA", "Banco Central", "Documentos"} {
		if !strings.Contains(s, marker) {
			t.Fatalf("dossiê sem seção %q", marker)
		}
	}
}

func TestReportAutomationStatus207(t *testing.T) {
	cases := map[string]string{
		"ok": "Consulta concluída",
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


func TestProfessionalWarningsSanitized207(t *testing.T) {
	input := []string{
		"Cobertura do solo: WorldCover/ArcGIS: Invalid URL: Invalid URL",
		"FUNAI: página WFS a partir de 0 inválida: invalid character '<' looking for beginning of value",
		"Taxas por instituição BCB: Get https://olinda.bcb.gov.br/: context deadline exceeded",
		"Nenhum alerta foi retornado para o CAR nesta consulta. Isso não equivale a certificado de regularidade ambiental.",
	}
	got := professionalReportWarnings(input)
	if len(got) != 4 {
		t.Fatalf("esperava 4 avisos profissionais, obteve %d: %#v", len(got), got)
	}
	joined := strings.Join(got, " | ")
	for _, raw := range []string{"Invalid URL", "invalid character", "context deadline exceeded", "https://olinda"} {
		if strings.Contains(joined, raw) {
			t.Fatalf("erro interno vazou para o relatório: %q", raw)
		}
	}
	for _, want := range []string{"ESA WorldCover", "FUNAI", "Banco Central"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("aviso profissional deveria preservar contexto %q: %s", want, joined)
		}
	}
}

func TestAutomationSourceStatusIsContextAware207(t *testing.T) {
	cases := []struct {
		s AutomationSourceStatus
		want string
	}{
		{AutomationSourceStatus{Key: "sicar", Status: "ok"}, "CAR localizado"},
		{AutomationSourceStatus{Key: "kml", Status: "ok"}, "Disponível"},
		{AutomationSourceStatus{Key: "ibama", Status: "ok"}, "Consulta concluída sem ocorrência"},
		{AutomationSourceStatus{Key: "sigef", Status: "ok"}, "Consulta concluída sem parcela localizada"},
		{AutomationSourceStatus{Key: "sicor", Status: "hit"}, "Operação pública localizada"},
	}
	for _, tc := range cases {
		if got := reportAutomationSourceStatus(tc.s); got != tc.want {
			t.Fatalf("%s/%s: esperado %q, obtido %q", tc.s.Key, tc.s.Status, tc.want, got)
		}
	}
}

func TestEnvironmentalReportSkipsSparseWorldCoverPage207(t *testing.T) {
	p := Property{Name: "Fazenda Teste"}
	car := CARResult{CAR: "MG-0000000-AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA", AreaHa: 20, Found: true}
	intel := EnvironmentalIntelligenceResult{
		CAR: car.CAR,
		PropertyAreaHa: 20,
		MapBiomas: MapBiomasCARSummary{Available: true, Connected: true},
		Profile: EnvironmentalProfile{Biome: "Mata Atlântica", BiomeAvailable: true, LandCoverAvailable: false},
	}
	pdf := buildEnvironmentalTechnicalPDF(p, car, intel, "")
	s := string(pdf)
	if !strings.Contains(s, "/Count 4") {
		t.Fatalf("sem WorldCover utilizável, o laudo deve ser condensado para 4 páginas")
	}
}

func TestProfessionalPDFDoesNotExposeRawTransportErrors207(t *testing.T) {
	p := Property{Name: "Fazenda Teste"}
	car := CARResult{CAR: "MG-0000000-AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA", AreaHa: 20, Found: true}
	intel := EnvironmentalIntelligenceResult{
		CAR: car.CAR,
		PropertyAreaHa: 20,
		MapBiomas: MapBiomasCARSummary{Available: true, Connected: true},
		Warnings: []string{
			"FUNAI: invalid character '<' looking for beginning of value",
			"Cobertura do solo: WorldCover/ArcGIS: Invalid URL: Invalid URL",
		},
	}
	pdf := buildEnvironmentalEvidencePDF(p, car, intel)
	s := string(pdf)
	for _, raw := range []string{"invalid character", "Invalid URL"} {
		if strings.Contains(s, raw) {
			t.Fatalf("PDF profissional contém erro interno: %q", raw)
		}
	}
}
