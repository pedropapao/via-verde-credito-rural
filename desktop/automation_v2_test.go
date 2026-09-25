package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClassifyUnifiedQueryCPF(t *testing.T) {
	mode, normalized, valid := v2ClassifyUnifiedQuery("529.982.247-25")
	if mode != "cpf" || normalized != "52998224725" || !valid {
		t.Fatalf("classificação CPF inesperada: %s %s %v", mode, normalized, valid)
	}
}

func TestClassifyUnifiedQueryCNPJ(t *testing.T) {
	mode, normalized, valid := v2ClassifyUnifiedQuery("11.222.333/0001-81")
	if mode != "cnpj" || normalized != "11222333000181" || !valid {
		t.Fatalf("classificação CNPJ inesperada: %s %s %v", mode, normalized, valid)
	}
}

func TestInvalidTaxIDs(t *testing.T) {
	if v2ValidCPF("11111111111") {
		t.Fatal("CPF repetido não pode ser válido")
	}
	if v2ValidCNPJ("00000000000000") {
		t.Fatal("CNPJ repetido não pode ser válido")
	}
}

func TestNormalizedSearchText(t *testing.T) {
	got := v2NormalizedSearchText("  São Sebastião   do Paraíso ")
	if got != "sao sebastiao do paraiso" {
		t.Fatalf("normalização inesperada: %q", got)
	}
}


func newV2AutomationTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	db, err := openSQLiteDatabase(filepath.Join(dir, "viaverde-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	a := &App{db: db, dataDir: dir}
	for _, name := range []string{"properties", "cache", "backups", "updates"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.migrate(); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestSearchEverythingFindsLocalCPFAndProperty(t *testing.T) {
	a := newV2AutomationTestApp(t)
	client, err := a.SaveClient(Client{Name: "Produtor Teste", CPFCNPJ: "529.982.247-25"})
	if err != nil {
		t.Fatal(err)
	}
	car := "MG-3106200-AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA.AAAA"
	_, err = a.SaveProperty(Property{
		ClientID: client.ID, Name: "Fazenda Teste", Municipality: "Belo Horizonte",
		UF: "MG", Registry: "12345", CARNumber: car, DeclaredAreaHa: 12.5,
	})
	if err != nil {
		t.Fatal(err)
	}

	byCPF, err := a.SearchEverything("529.982.247-25")
	if err != nil {
		t.Fatal(err)
	}
	if len(byCPF.Hits) == 0 || byCPF.Hits[0].PropertyName != "Fazenda Teste" {
		t.Fatalf("busca por CPF não encontrou imóvel: %#v", byCPF.Hits)
	}
	byRegistry, err := a.SearchEverything("12345")
	if err != nil {
		t.Fatal(err)
	}
	if len(byRegistry.Hits) == 0 || byRegistry.Hits[0].CAR != car {
		t.Fatalf("busca por matrícula não encontrou CAR: %#v", byRegistry.Hits)
	}
}

func TestSaveAnalyzedCARToClientReusesSessionAndCreatesKML(t *testing.T) {
	a := newV2AutomationTestApp(t)
	client, err := a.SaveClient(Client{Name: "Cliente CAR"})
	if err != nil {
		t.Fatal(err)
	}
	car := "MG-3106200-BBBB.BBBB.BBBB.BBBB.BBBB.BBBB.BBBB.BBBB"
	geo := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.0,-20.0],[-45.99,-20.0],[-45.99,-19.99],[-46.0,-19.99],[-46.0,-20.0]]]}}`
	result := CARResult{
		CAR: car, UF: "MG", Municipality: "Município Teste", PropertyName: "Fazenda Automática",
		AreaHa: 10.25, GeometryAreaHa: 10.25, Found: true, HasGeometry: true, GeoJSON: geo,
		CheckedAt: time.Now().Format(time.RFC3339),
	}
	a.saveLastCARSession(result)

	p, err := a.SaveAnalyzedCARToClient(client.ID, car)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Fazenda Automática" || p.CARNumber != car {
		t.Fatalf("imóvel automático inesperado: %#v", p)
	}
	if p.KMLPath != "" {
		t.Fatal("KML SICAR automático não pode ocupar o campo de KML externo")
	}
	latest, err := a.GetLatestCAR(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.AutoKMLPath == "" {
		t.Fatal("KML SICAR automático não foi registrado no resultado do CAR")
	}
	if _, err := os.Stat(latest.AutoKMLPath); err != nil {
		t.Fatalf("KML automático não existe: %v", err)
	}
	history, err := a.GetCARHistory(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("histórico esperado=1, obtido=%d", len(history))
	}

	again, err := a.SaveAnalyzedCARToClient(client.ID, car)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != p.ID {
		t.Fatalf("CAR duplicou imóvel: primeiro=%d segundo=%d", p.ID, again.ID)
	}
}


func TestSearchEverythingDocumentShowsOfficialPaths(t *testing.T) {
	a := newV2AutomationTestApp(t)
	r, err := a.SearchEverything("529.982.247-25")
	if err != nil {
		t.Fatal(err)
	}
	if r.Mode != "cpf" || len(r.OfficialOptions) < 2 {
		t.Fatalf("caminhos oficiais CPF ausentes: %#v", r.OfficialOptions)
	}
	if r.PrivacyNotice == "" {
		t.Fatal("pesquisa por CPF precisa explicar o limite de titularidade pública")
	}

	cnpj, err := a.SearchEverything("11.222.333/0001-81")
	if err != nil {
		t.Fatal(err)
	}
	if cnpj.Mode != "cnpj" || len(cnpj.OfficialOptions) < 3 {
		t.Fatalf("caminhos oficiais CNPJ ausentes: %#v", cnpj.OfficialOptions)
	}
}

func TestCARLookupFallbackUsesOnlySameSavedCAR(t *testing.T) {
	a := newV2AutomationTestApp(t)
	client, err := a.SaveClient(Client{Name: "Produtor Fallback"})
	if err != nil {
		t.Fatal(err)
	}
	car := "MG-3106200-CCCC.CCCC.CCCC.CCCC.CCCC.CCCC.CCCC.CCCC"
	p, err := a.SaveProperty(Property{ClientID: client.ID, Name: "Fazenda Fallback", UF: "MG", CARNumber: car})
	if err != nil {
		t.Fatal(err)
	}
	geo := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.0,-20.0],[-45.99,-20.0],[-45.99,-19.99],[-46.0,-19.99],[-46.0,-20.0]]]}}`
	r := CARResult{
		CAR: car, UF: "MG", Found: true, HasGeometry: true, GeoJSON: geo,
		LookupStatus: "found", PublicConfirmed: true, CheckedAt: time.Now().Format(time.RFC3339),
	}
	if err := a.persistCARAnalysis(p.ID, car, &r); err != nil {
		t.Fatal(err)
	}
	got, ok := a.carLookupFallback(p.ID, car)
	if !ok || got.CAR != car || !got.HasGeometry {
		t.Fatalf("fallback válido não recuperado: ok=%v result=%#v", ok, got)
	}
	if _, ok := a.carLookupFallback(p.ID, "MG-3106200-DDDD.DDDD.DDDD.DDDD.DDDD.DDDD.DDDD.DDDD"); ok {
		t.Fatal("fallback não pode reutilizar CAR diferente")
	}
}

func TestAutomationSourcesDistinguishesSICARCache(t *testing.T) {
	out := CARAutomationResult{CAR: CARResult{
		CAR: "MG-3106200-EEEE.EEEE.EEEE.EEEE.EEEE.EEEE.EEEE.EEEE",
		Found: true, HasGeometry: true, LookupStatus: "cached",
		LookupDetail: "SICAR indisponível; cache local reaproveitado.",
	}}
	sources := automationSources(out)
	if len(sources) == 0 || sources[0].Status != "cached" {
		t.Fatalf("status SICAR deveria ser cached: %#v", sources)
	}
	if automationOverallStatus(out) == "complete" {
		t.Fatal("resultado com SICAR em cache não pode ser marcado como completo")
	}
}
