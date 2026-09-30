package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseVersion212(t *testing.T) {
	if AppVersion != "2.1.2" {
		t.Fatalf("AppVersion inesperada: %s", AppVersion)
	}
	b, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"productVersion": "2.1.2"`) {
		t.Fatal("wails.json não está alinhado com a versão 2.1.2")
	}
}

func TestReleaseFirePipeline191(t *testing.T) {
	if fireStatusFound == "" || fireStatusNone == "" || fireStatusUnavailable == "" || fireStatusNotRun == "" {
		t.Fatal("estados da consulta de focos não podem ficar vazios")
	}
}

func TestReleaseLegacyShellAssetsRemainVersioned(t *testing.T) {
	for _, path := range []string{
		"frontend/dist/ui201.css", "frontend/dist/ui201.js",
		"frontend/dist/ui202.css", "frontend/dist/ui202.js",
	} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("asset legado ausente: %s", path)
		}
	}
}

func TestRelease204LoadsOnlyOperationalShell(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	for _, asset := range []string{"ui200.js", "ui200.css", "ui201.js", "ui201.css", "ui202.js", "ui202.css"} {
		if strings.Contains(html, asset) {
			t.Fatalf("shell legado não deve ser carregado na 2.0.4: %s", asset)
		}
	}
	for _, asset := range []string{"ui204.js", "ui204.css"} {
		if !strings.Contains(html, asset) {
			t.Fatalf("shell 2.0.4 obrigatório não carregado: %s", asset)
		}
	}
}

func TestRelease204UsesSVGAndSingleScreenTabs(t *testing.T) {
	jsb, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(jsb)
	for _, marker := range []string{
		"<svg class=\"vv204-icon",
		"Resumo",
		"Ambiental",
		"Fundiário",
		"Crédito Rural",
		"Mapa",
		"Documentos",
		"Relatórios",
		"BANCO CENTRAL",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface 2.0.4 sem marcador %q", marker)
		}
	}
	cssb, err := os.ReadFile("frontend/dist/ui204.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(cssb)
	if !strings.Contains(css, "body.vv204 .sidebar{display:none!important}") {
		t.Fatal("interface 2.0.4 deve remover a sidebar fixa da experiência principal")
	}
}

func TestRelease203BCBModulePreserved(t *testing.T) {
	for _, path := range []string{"bcb_public.go", "bcb_public_test.go"} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("módulo BCB ausente: %s", path)
		}
	}
}


func TestRelease205ProfessionalPDFsPreserved(t *testing.T) {
	if st, err := os.Stat("professional_reports.go"); err != nil || st.Size() == 0 {
		t.Fatal("módulo de PDFs profissionais ausente")
	}
	b, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil { t.Fatal(err) }
	js := string(b)
	for _, marker := range []string{
		"Gerar Demonstrativo PDF",
		"Gerar Laudo PDF",
		"Gerar Evidências PDF",
		"Gerar Dossiê PDF",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface sem saída PDF %q", marker)
		}
	}
}


func TestRelease206PDFsDoNotRequireSavedProperty(t *testing.T) {
	b, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil { t.Fatal(err) }
	js := string(b)
	for _, marker := range []string{
		"ExportCARAutomationPDF(r)",
		"ExportEnvironmentalAutomationPDF(r)",
		"ExportEnvironmentalAutomationEvidencePDF(r)",
		"ExportPropertyAutomationDossierPDF(r)",
		"Salvar/vincular o imóvel é opcional",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("consulta avulsa sem integração PDF %q", marker)
		}
	}
	if strings.Contains(js, "const can=!!r.property_id;") {
		t.Fatal("relatórios 2.0.6 não podem depender de property_id")
	}
}


func TestRelease207KeepsInternalErrorsOutOfProfessionalPDFs(t *testing.T) {
	b, err := os.ReadFile("professional_reports.go")
	if err != nil { t.Fatal(err) }
	src := string(b)
	for _, marker := range []string{
		"professionalReportWarnings",
		"professionalSourceDetail",
		"reportAutomationSourceStatus",
		"Fonte externa indisponível nesta execução",
	} {
		if !strings.Contains(src, marker) {
			t.Fatalf("acabamento PDF 2.0.7 sem marcador %q", marker)
		}
	}
}


func TestRelease208DocumentCenterAndPendingTab(t *testing.T) {
	if st, err := os.Stat("documents.go"); err != nil || st.Size() == 0 {
		t.Fatal("módulo de documentos 2.0.8 ausente")
	}
	jsb, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil { t.Fatal(err) }
	js := string(jsb)
	for _, marker := range []string{
		"Central de documentos do imóvel",
		"Pendências",
		"GetPropertyDocumentCenter",
		"AddPropertyDocument",
		"SetPropertyDocumentStatus",
		"ListPropertyDocumentHistory",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface 2.0.8 sem marcador %q", marker)
		}
	}
	appb, err := os.ReadFile("app.go")
	if err != nil { t.Fatal(err) }
	app := string(appb)
	for _, marker := range []string{"property_documents", "property_document_status", "property_document_context", "property_document_automation", "version=8"} {
		if !strings.Contains(app, marker) {
			t.Fatalf("migração documental 2.0.8 sem marcador %q", marker)
		}
	}
}

func TestRelease208DossierIncludesDocumentsPage(t *testing.T) {
	b, err := os.ReadFile("professional_reports.go")
	if err != nil { t.Fatal(err) }
	src := string(b)
	for _, marker := range []string{
		"dossierDocumentsPage",
		"Documentos, pendências e estado do dossiê",
		"GetPropertyDocumentCenter",
	} {
		if !strings.Contains(src, marker) {
			t.Fatalf("dossiê 2.0.8 sem integração documental %q", marker)
		}
	}
}


func TestRelease209SmartPending(t *testing.T) {
	jsb, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil { t.Fatal(err) }
	js := string(jsb)
	for _, marker := range []string{
		"Pendências inteligentes",
		"SavePropertyDocumentContext",
		"OBRIGATÓRIO PELO CONTEXTO",
		"Automático / não sei",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface 2.0.9 sem marcador %q", marker)
		}
	}
	db, err := os.ReadFile("documents.go")
	if err != nil { t.Fatal(err) }
	src := string(db)
	for _, marker := range []string{
		"DocumentProjectContext",
		"documentRequirements",
		"requirementByTriState",
		"RequirementReason",
	} {
		if !strings.Contains(src, marker) {
			t.Fatalf("backend 2.0.9 sem marcador %q", marker)
		}
	}
}


func TestRelease210DocumentAutomation(t *testing.T) {
	jsb, err := os.ReadFile("frontend/dist/ui204.js")
	if err != nil { t.Fatal(err) }
	js := string(jsb)
	for _, marker := range []string{
		"AUTOMAÇÃO DOCUMENTAL • 2.1.0",
		"RunPropertyDocumentAutomation",
		"Executar automação",
		"Fontes indisponíveis",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface 2.1.0 sem marcador %q", marker)
		}
	}
	db, err := os.ReadFile("documents.go")
	if err != nil { t.Fatal(err) }
	src := string(db)
	for _, marker := range []string{
		"PropertyDocumentAutomationResult",
		"probeOfficialDocumentSource",
		"GetLastPropertyDocumentAutomation",
		"extractDocumentReferenceYear",
		"Fonte oficial indisponível nesta execução",
	} {
		if !strings.Contains(src, marker) {
			t.Fatalf("backend 2.1.0 sem marcador %q", marker)
		}
	}
	pdf, err := os.ReadFile("professional_reports.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(pdf), "Última automação documental") {
		t.Fatal("dossiê 2.1.0 sem resumo da automação documental")
	}
}


func TestRelease211CreditOperationDetails(t *testing.T) {
	for _, path := range []string{"frontend/dist/ui211.js", "frontend/dist/ui211.css"} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("asset 2.1.1 ausente: %s", path)
		}
	}
	htmlb, err := os.ReadFile("frontend/dist/index.html")
	if err != nil { t.Fatal(err) }
	html := string(htmlb)
	for _, asset := range []string{"ui211.js", "ui211.css"} {
		if !strings.Contains(html, asset) {
			t.Fatalf("index não carrega %s", asset)
		}
	}

	jsb, err := os.ReadFile("frontend/dist/ui211.js")
	if err != nil { t.Fatal(err) }
	js := string(jsb)
	for _, marker := range []string{
		"FICHA COMPLETA • SICOR/BCB",
		"Ver operação completa",
		"GetCreditIntelligenceProgress",
		"Área financiada / glebas",
		"Fontes oficiais e rastreabilidade",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface 2.1.1 sem marcador %q", marker)
		}
	}

	srcb, err := os.ReadFile("credit_intelligence.go")
	if err != nil { t.Fatal(err) }
	src := string(srcb)
	for _, marker := range []string{
		"creditIntelligenceResultCachePath",
		"scanCreditOperationComplement",
		"SICOR_COMPLEMENTO_OPERACAO_BASICA.gz",
		"VL_PARC_CREDITO",
		"VL_PERC_RISCO_STN",
		"REF_BACEN_EFETIVO",
		"Glebas",
	} {
		if !strings.Contains(src, marker) {
			t.Fatalf("backend 2.1.1 sem marcador %q", marker)
		}
	}
}


func TestRelease212DarkOperationalUI(t *testing.T) {
	for _, path := range []string{"frontend/dist/ui212.js", "frontend/dist/ui212.css"} {
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			t.Fatalf("asset visual 2.1.2 ausente: %s", path)
		}
	}
	htmlb, err := os.ReadFile("frontend/dist/index.html")
	if err != nil { t.Fatal(err) }
	html := string(htmlb)
	for _, asset := range []string{"ui211.css", "ui212.css", "ui211.js", "ui212.js"} {
		if !strings.Contains(html, asset) {
			t.Fatalf("index não carrega %s", asset)
		}
	}
	if strings.Index(html, "ui212.css") < strings.Index(html, "ui211.css") {
		t.Fatal("ui212.css deve ser carregado depois do ui211.css")
	}
	if strings.Index(html, "ui212.js") < strings.Index(html, "ui211.js") {
		t.Fatal("ui212.js deve ser carregado depois do ui211.js")
	}

	jsb, err := os.ReadFile("frontend/dist/ui212.js")
	if err != nil { t.Fatal(err) }
	js := string(jsb)
	for _, marker := range []string{
		"interface escura operacional",
		"Crédito Rural",
		"Documentos",
		"Pendências",
		"GetPropertyDocumentCenter",
		"Gerar Dossiê",
		"connectionLabel",
		"versionLabel",
	} {
		if !strings.Contains(js, marker) {
			t.Fatalf("interface 2.1.2 sem marcador %q", marker)
		}
	}

	cssb, err := os.ReadFile("frontend/dist/ui212.css")
	if err != nil { t.Fatal(err) }
	css := string(cssb)
	for _, marker := range []string{
		"body.vv204.vv212-dark",
		"grid-template-columns:214px",
		".vv212-summary-mode",
		".vv212-summary-side",
		".vv212-dossier",
		".vv211-modal-card",
	} {
		if !strings.Contains(css, marker) {
			t.Fatalf("tema 2.1.2 sem marcador %q", marker)
		}
	}
}
