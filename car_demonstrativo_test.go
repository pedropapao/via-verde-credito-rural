package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildCARTechnicalPDF(t *testing.T) {
	g := carGeoJSONGeometry{
		Type: "Polygon",
		Coordinates: json.RawMessage(`[[[-49.10,-17.80],[-49.00,-17.80],[-49.00,-17.70],[-49.10,-17.70],[-49.10,-17.80]]]`),
	}
	d := carDemonstrativoData{
		CAR: "GO-5209101-D25F.EA6E.3C2A.400E.B07E.4787.3A6C.E563",
		UF: "GO", Municipio: "Teste", AreaHa: 120.50, Modulos: 2.5,
		Status: "Ativo", Condicao: "Aguardando analise", Tipo: "Imovel Rural",
		DataCadastro: "01/01/2024", DataAtualiza: "02/02/2026",
		PerimetroM: carGeometryPerimeterM(g), CentroLat: -17.75, CentroLon: -49.05,
		Geometry: g, ConsultadoEm: "19/08/2026 16:00",
	}
	pdf, err := buildCARTechnicalPDF(d)
	if err != nil { t.Fatalf("geracao do PDF falhou: %v", err) }
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) { t.Fatalf("arquivo nao inicia como PDF: %q", pdf[:min(12, len(pdf))]) }
	if !bytes.Contains(pdf, []byte(d.CAR)) { t.Fatal("PDF deve conter o numero do CAR") }
	if !bytes.Contains(pdf, []byte("DEMONSTRATIVO TECNICO DO CAR")) { t.Fatal("PDF deve identificar sua natureza tecnica") }
}

func TestCarGeometryCenterAndPerimeter(t *testing.T) {
	g := carGeoJSONGeometry{
		Type: "Polygon",
		Coordinates: json.RawMessage(`[[[-45.0,-20.0],[-44.99,-20.0],[-44.99,-19.99],[-45.0,-19.99],[-45.0,-20.0]]]`),
	}
	lat, lon := carGeometryCenter(g)
	if lat == 0 || lon == 0 { t.Fatalf("centro inesperado: %.6f %.6f", lat, lon) }
	if p := carGeometryPerimeterM(g); p <= 0 { t.Fatalf("perimetro deveria ser positivo: %.2f", p) }
}

func TestCentralCARExibeDemonstrativoTecnico(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/car.html")
	if err != nil { t.Fatalf("nao foi possivel ler template: %v", err) }
	page := string(body)
	for _, want := range []string{
		"/car/demonstrativo?number={{.CAR}}",
		"Baixar demonstrativo técnico PDF",
		"Abrir demonstrativo oficial do SICAR",
		"não substitui o Demonstrativo oficial",
	} {
		if !strings.Contains(page, want) { t.Fatalf("template CAR nao contem %q", want) }
	}
}
