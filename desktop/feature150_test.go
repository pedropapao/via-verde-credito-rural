package main

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func writeGzipFixture150(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSICORWKTPolygonToGeoJSON(t *testing.T) {
	raw := "POLYGON ((-46.6100 -20.9100 0,-46.6000 -20.9100 0,-46.6000 -20.9000 0,-46.6100 -20.9000 0,-46.6100 -20.9100 0))"
	geo, err := sicorWKTToGeoJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	metric, err := projectAreaMetrics(geo)
	if err != nil {
		t.Fatal(err)
	}
	if metric.AreaHa <= 0 {
		t.Fatalf("área inválida: %.4f", metric.AreaHa)
	}
}



func TestSICORWKTPolygonZToGeoJSON(t *testing.T) {
	raw := "POLYGON Z ((-46.6100 -20.9100 0,-46.6000 -20.9100 0,-46.6000 -20.9000 0,-46.6100 -20.9000 0,-46.6100 -20.9100 0))"
	geo, err := sicorWKTToGeoJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projectAreaMetrics(geo); err != nil {
		t.Fatal(err)
	}
}

func TestSICORWKTMultiPolygonToGeoJSON(t *testing.T) {
	raw := "MULTIPOLYGON (((-46.61 -20.91 0,-46.60 -20.91 0,-46.60 -20.90 0,-46.61 -20.90 0,-46.61 -20.91 0)),((-46.59 -20.91 0,-46.58 -20.91 0,-46.58 -20.90 0,-46.59 -20.90 0,-46.59 -20.91 0)))"
	geo, err := sicorWKTToGeoJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	metric, err := projectAreaMetrics(geo)
	if err != nil {
		t.Fatal(err)
	}
	if metric.AreaHa <= 0 {
		t.Fatalf("área inválida: %.4f", metric.AreaHa)
	}
}

func TestScanSICORPropertyRefsNormalizesCAR(t *testing.T) {
	path := writeGzipFixture150(t, "properties.gz",
		"#REF_BACEN;NU_ORDEM;CD_CNPJ_CPF;CD_SNCR;CD_CIB;CD_CAR\n"+
			"517256416;1;00000000000;;;MG-3107703-7186.E27C.4DE7.4C13.A051.ACE1.4DAF.E97C\n"+
			"517256416;2;00000000000;;;MG-3107703-7186E27C4DE74C13A051ACE14DAFE97C\n"+
			"999999999;1;00000000000;;;MG-3550308-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\n")
	refs, err := scanSICORPropertyRefs(path, "MG-3107703-7186.E27C.4DE7.4C13.A051.ACE1.4DAF.E97C")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("esperava 2 destinações vinculadas, obteve %d: %#v", len(refs), refs)
	}
}

func TestScanSICOROperationsFiltersOnlyTarget(t *testing.T) {
	path := writeGzipFixture150(t, "ops.gz",
		"#REF_BACEN;NU_ORDEM;CNPJ_IF;DT_EMISSAO;DT_VENCIMENTO;CD_FONTE_RECURSO;CD_ESTADO;CD_TIPO_SEGURO;CD_EMPREENDIMENTO;CD_PROGRAMA;VL_JUROS;VL_PARC_CREDITO;VL_REC_PROPRIO;VL_AREA_FINANC;CD_SUBPROGRAMA;VL_AREA_INFORMADA\n"+
			"517256416;1;07237373;2024-01-02;2027-01-02;0502;MG;9;13105760989408;0001;0.50;6000.00;100.00;3.00;0056;3.00\n"+
			"999999999;1;00000000;2024-01-02;2025-01-02;0001;SP;0;10000000000000;0002;1.00;999.00;0;1;0001;1\n")
	targets := map[string]sicorRef{
		sicorOperationKey("517256416", "1"): {RefBacen: "517256416", Order: "1"},
	}
	got, err := scanSICOROperations(path, targets, 2024)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("esperava 1 operação/destinação, obteve %d: %#v", len(got), got)
	}
	if got[0].CreditValue != 6000 || got[0].FinancedAreaHa != 3 {
		t.Fatalf("valores inesperados: %+v", got[0])
	}
}

func TestScanSICORGlebasAndProjectOverlap(t *testing.T) {
	car := `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.62,-20.92],[-46.58,-20.92],[-46.58,-20.88],[-46.62,-20.88],[-46.62,-20.92]]]}}`
	project := ProjectArea{
		Name: "Projeto atual",
		GeoJSON: `{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.61,-20.91],[-46.60,-20.91],[-46.60,-20.90],[-46.61,-20.90],[-46.61,-20.91]]]}}`,
	}
	path := writeGzipFixture150(t, "glebas.gz",
		"#REF_BACEN;NU_ORDEM;NU_INDICE;GT_GEOMETRIA\n"+
			"517256416;1;0;POLYGON ((-46.61 -20.91 0,-46.60 -20.91 0,-46.60 -20.90 0,-46.61 -20.90 0,-46.61 -20.91 0))\n")
	got, err := scanSICORGlebas(path, map[string]bool{sicorOperationKey("517256416", "1"): true}, car, []ProjectArea{project})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("esperava 1 gleba, obteve %d", len(got))
	}
	if got[0].InsideCARPct < 99 || got[0].ProjectOverlapPct < 99 {
		t.Fatalf("cruzamento inesperado: %+v", got[0])
	}
}

func TestParseSICORNumber(t *testing.T) {
	for input, want := range map[string]float64{
		"6000.00": 6000,
		"1.234,50": 1234.5,
		"3,25": 3.25,
	} {
		if got := parseSICORNumber(input); got != want {
			t.Fatalf("%q: esperado %.2f, obteve %.2f", input, want, got)
		}
	}
}
