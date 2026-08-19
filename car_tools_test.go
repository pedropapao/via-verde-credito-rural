package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeCAR(t *testing.T) {
	in := "mg-3164605-6f3f.1d39.a7a5.4752.b116.1c3f.b236.e981"
	got, uf, muni, err := normalizeCAR(in)
	if err != nil { t.Fatalf("CAR válido rejeitado: %v", err) }
	if got != "MG-3164605-6F3F.1D39.A7A5.4752.B116.1C3F.B236.E981" { t.Fatalf("normalização inesperada: %s", got) }
	if uf != "MG" || muni != "3164605" { t.Fatalf("UF/município inesperados: %s %s", uf, muni) }
}

func TestNormalizeCARSemPontos(t *testing.T) {
	in := "MG-3164605-6F3F1D39A7A54752B1161C3FB236E981"
	got, _, _, err := normalizeCAR(in)
	if err != nil { t.Fatalf("CAR sem pontos deveria ser aceito: %v", err) }
	if got != "MG-3164605-6F3F.1D39.A7A5.4752.B116.1C3F.B236.E981" { t.Fatalf("normalização inesperada: %s", got) }
}

func TestNormalizeCARInvalido(t *testing.T) {
	if _, _, _, err := normalizeCAR("MG-123"); err == nil { t.Fatal("esperava erro para CAR inválido") }
}

func TestCARLookupCodes(t *testing.T) {
	codes := carLookupCodes("MG-3164605-6F3F.1D39.A7A5.4752.B116.1C3F.B236.E981")
	if len(codes) != 2 { t.Fatalf("esperava duas variantes de busca, recebeu %d", len(codes)) }
	if strings.Contains(codes[1], ".") { t.Fatalf("segunda variante deveria estar sem pontos: %s", codes[1]) }
}

func TestCARLayerUF(t *testing.T) {
	if carLayerUF("MG") != "mg" { t.Fatalf("MG deveria gerar layer mg") }
	if carLayerUF("DF") != "DF" { t.Fatalf("DF deveria preservar caixa alta") }
}

func TestCARLabels(t *testing.T) {
	if carStatusLabel("AT") != "Ativo" { t.Fatalf("status AT não traduzido") }
	if carStatusLabel("SU") != "Suspenso" { t.Fatalf("status SU não traduzido") }
	if carPropertyTypeLabel("IRU") != "Imóvel Rural" { t.Fatalf("tipo IRU não traduzido") }
	if carPropertyTypeLabel("AST") != "Assentamento da Reforma Agrária" { t.Fatalf("tipo AST não traduzido") }
}

func TestCARGeometryKML(t *testing.T) {
	g := carGeoJSONGeometry{
		Type: "Polygon",
		Coordinates: json.RawMessage(`[[[-45.0,-20.0],[-44.9,-20.0],[-44.9,-19.9],[-45.0,-19.9],[-45.0,-20.0]]]`),
	}
	out, err := carGeometryKML("MG-3164605-6F3F.1D39.A7A5.4752.B116.1C3F.B236.E981", g)
	if err != nil { t.Fatalf("KML falhou: %v", err) }
	if !bytes.Contains(out, []byte(`<kml xmlns="http://www.opengis.net/kml/2.2">`)) { t.Fatalf("KML sem namespace esperado: %s", string(out)) }
	if !bytes.Contains(out, []byte("-45.00000000,-20.00000000,0")) { t.Fatalf("KML sem coordenada esperada: %s", string(out)) }
}

func TestCARGeometrySVG(t *testing.T) {
	g := carGeoJSONGeometry{
		Type: "Polygon",
		Coordinates: json.RawMessage(`[[[-45.0,-20.0],[-44.9,-20.0],[-44.9,-19.9],[-45.0,-19.9],[-45.0,-20.0]]]`),
	}
	svg := carGeometrySVG(g)
	if !strings.Contains(svg, `<svg viewBox="0 0 760 360"`) { t.Fatalf("SVG inválido: %s", svg) }
	if strings.Contains(svg, `\"`) { t.Fatalf("SVG contém barras de escape indevidas: %s", svg) }
}

func TestCARAreaPositiva(t *testing.T) {
	g := carGeoJSONGeometry{
		Type: "Polygon",
		Coordinates: json.RawMessage(`[[[-45.0,-20.0],[-44.99,-20.0],[-44.99,-19.99],[-45.0,-19.99],[-45.0,-20.0]]]`),
	}
	if area := carGeometryAreaHa(g); area <= 0 { t.Fatalf("área deveria ser positiva, recebeu %.4f", area) }
}
