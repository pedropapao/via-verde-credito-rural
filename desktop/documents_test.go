package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newDocumentTestProperty(t *testing.T) (*App, Property) {
	t.Helper()
	a := newV2AutomationTestApp(t)
	client, err := a.SaveClient(Client{Name: "Cliente Documentos", CPFCNPJ: "529.982.247-25"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.SaveProperty(Property{
		ClientID: client.ID,
		Name: "Fazenda Documental",
		Municipality: "Jacuí",
		UF: "MG",
		Registry: "12345",
		CARNumber: "MG-3106200-ABCD.ABCD.ABCD.ABCD.ABCD.ABCD.ABCD.ABCD",
		DeclaredAreaHa: 25.4,
	})
	if err != nil {
		t.Fatal(err)
	}
	return a, p
}

func TestSchemaVersion209(t *testing.T) {
	a := newV2AutomationTestApp(t)
	var version int
	if err := a.db.QueryRow(`SELECT version FROM schema_version LIMIT 1`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 7 {
		t.Fatalf("schema esperado=7, obtido=%d", version)
	}
	for _, table := range []string{"property_documents", "property_document_status", "property_document_context"} {
		var name string
		if err := a.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
			t.Fatalf("tabela %s ausente: %v", table, err)
		}
	}
}

func TestDocumentCenterAutomaticAndConditionalStates208(t *testing.T) {
	a, p := newDocumentTestProperty(t)
	kmlPath := filepath.Join(a.dataDir, "properties", "autokml-test.kml")
	if err := os.WriteFile(kmlPath, []byte("<kml></kml>"), 0o644); err != nil {
		t.Fatal(err)
	}
	car := CARResult{
		CAR: p.CARNumber,
		Found: true,
		HasGeometry: true,
		AutoKMLPath: kmlPath,
	}
	if _, err := a.db.Exec(`UPDATE properties SET last_car_json=? WHERE id=?`, marshalJSON(car), p.ID); err != nil {
		t.Fatal(err)
	}

	center, err := a.GetPropertyDocumentCenter(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertDocStatus := func(docType, want string) {
		t.Helper()
		for _, item := range center.Items {
			if item.DocType == docType {
				if item.Status != want {
					t.Fatalf("%s: esperado %s, obtido %s", docType, want, item.Status)
				}
				return
			}
		}
		t.Fatalf("item %s ausente", docType)
	}
	assertDocStatus("car", "received")
	assertDocStatus("kml_sicar", "received")
	assertDocStatus("registry", "pending")
	assertDocStatus("ccir", "pending")
	assertDocStatus("outorga", "review")
	assertDocStatus("gta", "review")

	if err := a.SetPropertyDocumentStatus(p.ID, "outorga", "not_applicable", "Projeto sem uso de água."); err != nil {
		t.Fatal(err)
	}
	center, err = a.GetPropertyDocumentCenter(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range center.Items {
		if item.DocType == "outorga" && item.Status != "not_applicable" {
			t.Fatalf("outorga deveria ficar não aplicável: %#v", item)
		}
	}
}

func TestExpiredDocumentOverridesReceived208(t *testing.T) {
	a, p := newDocumentTestProperty(t)
	dir := filepath.Join(a.dataDir, "properties", "docs-test")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ccir.pdf")
	if err := os.WriteFile(path, []byte("arquivo teste"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Format(time.RFC3339)
	past := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	res, err := a.db.Exec(`INSERT INTO property_documents(
		property_id,doc_type,title,original_name,stored_path,source,issue_date,expiry_date,reference_year,notes,sha256,size_bytes,is_current,created_at,updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,1,?,?)`,
		p.ID, "ccir", "CCIR", "ccir.pdf", path, "INCRA / SNCR", "", past, "2025", "", "abc", 13, now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	if err := a.SetPropertyDocumentStatus(p.ID, "ccir", "received", ""); err != nil {
		t.Fatal(err)
	}

	center, err := a.GetPropertyDocumentCenter(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range center.Items {
		if item.DocType == "ccir" {
			found = true
			if item.Status != "expired" {
				t.Fatalf("documento vencido não pode ficar recebido: %#v", item)
			}
			if item.Current == nil || item.Current.ID != id {
				t.Fatalf("documento atual inesperado: %#v", item.Current)
			}
		}
	}
	if !found {
		t.Fatal("CCIR ausente do checklist")
	}
}

func TestArchiveDocumentPreservesFileAndHistory208(t *testing.T) {
	a, p := newDocumentTestProperty(t)
	dir := filepath.Join(a.dataDir, "properties", "docs-archive")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "matricula.pdf")
	if err := os.WriteFile(path, []byte("matricula"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Format(time.RFC3339)
	res, err := a.db.Exec(`INSERT INTO property_documents(
		property_id,doc_type,title,original_name,stored_path,source,issue_date,expiry_date,reference_year,notes,sha256,size_bytes,is_current,created_at,updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,1,?,?)`,
		p.ID, "registry", "Matrícula / certidão do imóvel", "matricula.pdf", path, "RI Digital / Cartório", "", "", "", "", "def", 9, now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	if err := a.SetPropertyDocumentStatus(p.ID, "registry", "received", ""); err != nil {
		t.Fatal(err)
	}
	if err := a.ArchivePropertyDocument(id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("arquivar não pode apagar o arquivo: %v", err)
	}
	history, err := a.ListPropertyDocumentHistory(p.ID, "registry")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].IsCurrent {
		t.Fatalf("histórico inesperado após arquivar: %#v", history)
	}
	center, err := a.GetPropertyDocumentCenter(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range center.Items {
		if item.DocType == "registry" && item.Status != "pending" {
			t.Fatalf("matrícula arquivada deveria voltar a pendente: %#v", item)
		}
	}
}

func TestSafePropertyDocumentFilename208(t *testing.T) {
	got := safePropertyDocumentFilename("  Matrícula João 2026.pdf ")
	if got == "" || filepath.Ext(got) != ".pdf" {
		t.Fatalf("nome seguro inesperado: %q", got)
	}
	if got == "Matrícula João 2026.pdf" {
		t.Fatalf("nome deve ser normalizado para armazenamento: %q", got)
	}
}


func TestSmartDocumentContext209(t *testing.T) {
	a, p := newDocumentTestProperty(t)
	ctx, err := a.SavePropertyDocumentContext(DocumentProjectContext{
		PropertyID: p.ID,
		Activity: "irrigacao",
		OperationType: "investimento",
		Tenure: "arrendado",
		WaterUse: "auto",
		AnimalTransit: "no",
		SupplierPurchase: "auto",
		TechnicalReport: "yes",
		Notes: "Irrigação em área arrendada.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ctx.UpdatedAt == "" {
		t.Fatal("contexto documental deve registrar atualização")
	}

	center, err := a.GetPropertyDocumentCenter(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"lease": "pending",
		"outorga": "pending",
		"gta": "not_applicable",
		"budget": "pending",
		"technical_report": "pending",
	}
	for typ, status := range want {
		found := false
		for _, item := range center.Items {
			if item.DocType != typ {
				continue
			}
			found = true
			if item.Status != status {
				t.Fatalf("%s esperado=%s obtido=%s item=%#v", typ, status, item.Status, item)
			}
			if status == "pending" && !item.Required {
				t.Fatalf("%s deveria estar marcado como obrigatório pelo contexto", typ)
			}
		}
		if !found {
			t.Fatalf("item %s ausente", typ)
		}
	}
}

func TestManualDocumentStatusOverridesInference209(t *testing.T) {
	a, p := newDocumentTestProperty(t)
	if _, err := a.SavePropertyDocumentContext(DocumentProjectContext{
		PropertyID: p.ID,
		Activity: "irrigacao",
		WaterUse: "yes",
		AnimalTransit: "auto",
		SupplierPurchase: "auto",
		TechnicalReport: "auto",
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.SetPropertyDocumentStatus(p.ID, "outorga", "not_applicable", "Dispensado após conferência manual."); err != nil {
		t.Fatal(err)
	}
	center, err := a.GetPropertyDocumentCenter(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range center.Items {
		if item.DocType == "outorga" {
			if item.Status != "not_applicable" {
				t.Fatalf("status manual deve prevalecer: %#v", item)
			}
			return
		}
	}
	t.Fatal("outorga ausente")
}
