package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type dossierKMLLayer struct {
	Name    string
	StyleID string
	GeoJSON string
}

func buildEnvironmentalKML(car CARResult, kml KMLResult) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
`)
	b.WriteString(`<kml xmlns="http://www.opengis.net/kml/2.2"><Document>
`)
	b.WriteString("<name>" + kmlXMLEscape("Via Verde CAR - Dossiê Ambiental") + "</name>\n")
	b.WriteString(`<Style id="car"><LineStyle><color>ff317a2c</color><width>3</width></LineStyle><PolyStyle><color>22317a2c</color></PolyStyle></Style>
<Style id="kml"><LineStyle><color>ff2279d7</color><width>3</width></LineStyle><PolyStyle><color>142279d7</color></PolyStyle></Style>
<Style id="app"><LineStyle><color>fff6823b</color><width>2</width></LineStyle><PolyStyle><color>334aa3ff</color></PolyStyle></Style>
<Style id="rl"><LineStyle><color>ff346516</color><width>2</width></LineStyle><PolyStyle><color>3322c55e</color></PolyStyle></Style>
<Style id="vn"><LineStyle><color>ff2d5314</color><width>2</width></LineStyle><PolyStyle><color>2b15803d</color></PolyStyle></Style>
<Style id="ac"><LineStyle><color>ff0c41c2</color><width>2</width></LineStyle><PolyStyle><color>28fb923c</color></PolyStyle></Style>
<Style id="ur"><LineStyle><color>ffed3a7c</color><width>2</width></LineStyle><PolyStyle><color>22a78bfa</color></PolyStyle></Style>
<Style id="sa"><LineStyle><color>ff5b5b52</color><width>2</width></LineStyle><PolyStyle><color>18a1a1aa</color></PolyStyle></Style>
<Style id="ibama"><LineStyle><color>ff1c1cb9</color><width>3</width></LineStyle><PolyStyle><color>3a4444ef</color></PolyStyle></Style>
<Style id="funai"><LineStyle><color>ffce227e</color><width>3</width></LineStyle><PolyStyle><color>2ba855f7</color></PolyStyle></Style>
<Style id="uc"><LineStyle><color>ffa16903</color><width>3</width></LineStyle><PolyStyle><color>2638bdf8</color></PolyStyle></Style>
`)

	layers := []dossierKMLLayer{
		{Name: "Perímetro público do SICAR", StyleID: "car", GeoJSON: car.GeoJSON},
	}
	if strings.TrimSpace(kml.GeoJSON) != "" {
		layers = append(layers, dossierKMLLayer{Name: "KML do cliente", StyleID: "kml", GeoJSON: kml.GeoJSON})
	}
	themeDefs := []struct {
		Code, Name, Style string
	}{
		{"APP", "APP declarada no SICAR", "app"},
		{"RESERVA_LEGAL", "Reserva Legal declarada no SICAR", "rl"},
		{"VEGETACAO_NATIVA", "Vegetação nativa declarada no SICAR", "vn"},
		{"AREA_CONSOLIDADA", "Área consolidada declarada no SICAR", "ac"},
		{"USO_RESTRITO", "Área de uso restrito declarada no SICAR", "ur"},
		{"SERVIDAO_ADMINISTRATIVA", "Servidão administrativa declarada no SICAR", "sa"},
	}
	for _, def := range themeDefs {
		if m, ok := car.Themes.Themes[def.Code]; ok && strings.TrimSpace(m.GeoJSON) != "" {
			layers = append(layers, dossierKMLLayer{Name: def.Name, StyleID: def.Style, GeoJSON: m.GeoJSON})
		}
	}
	for i, x := range car.Environment.IBAMAEmbargos {
		layers = append(layers, dossierKMLLayer{
			Name:    fmt.Sprintf("Embargo IBAMA %d - %s", i+1, firstNonEmpty(x.Number, x.Situation)),
			StyleID: "ibama",
			GeoJSON: x.GeoJSON,
		})
	}
	for i, x := range car.Environment.IndigenousFindings {
		layers = append(layers, dossierKMLLayer{
			Name:    fmt.Sprintf("Terra Indígena %d - %s", i+1, firstNonEmpty(x.Name, x.Phase)),
			StyleID: "funai",
			GeoJSON: x.GeoJSON,
		})
	}
	for i, x := range car.Environment.FederalUCFindings {
		layers = append(layers, dossierKMLLayer{
			Name:    fmt.Sprintf("UC Federal %d - %s", i+1, firstNonEmpty(x.Name, x.Category)),
			StyleID: "uc",
			GeoJSON: x.GeoJSON,
		})
	}

	for _, layer := range layers {
		writeGeoJSONKMLFolder(&b, layer)
	}
	b.WriteString("</Document></kml>\n")
	return []byte(b.String())
}

func writeGeoJSONKMLFolder(b *strings.Builder, layer dossierKMLLayer) {
	features := geoJSONFeatures(layer.GeoJSON)
	if len(features) == 0 {
		return
	}
	b.WriteString("<Folder><name>" + kmlXMLEscape(layer.Name) + "</name>\n")
	for i, feature := range features {
		polys, err := carGeometryPolygons(feature.Geometry)
		if err != nil || len(polys) == 0 {
			continue
		}
		name := layer.Name
		if len(features) > 1 {
			name = fmt.Sprintf("%s - feição %d", layer.Name, i+1)
		}
		b.WriteString("<Placemark><name>" + kmlXMLEscape(name) + "</name><styleUrl>#" + layer.StyleID + "</styleUrl><MultiGeometry>\n")
		for _, poly := range polys {
			writeKMLPolygon(b, poly)
		}
		b.WriteString("</MultiGeometry></Placemark>\n")
	}
	b.WriteString("</Folder>\n")
}

func geoJSONFeatures(raw string) []carGeoFeature {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var feature carGeoFeature
	if json.Unmarshal([]byte(raw), &feature) == nil && strings.EqualFold(feature.Type, "Feature") && feature.Geometry.Type != "" {
		return []carGeoFeature{feature}
	}
	var fc carGeoJSON
	if json.Unmarshal([]byte(raw), &fc) == nil && len(fc.Features) > 0 {
		return fc.Features
	}
	return nil
}

func writeKMLPolygon(b *strings.Builder, poly [][][]float64) {
	if len(poly) == 0 {
		return
	}
	b.WriteString("<Polygon><tessellate>1</tessellate>")
	writeKMLBoundary(b, "outerBoundaryIs", poly[0])
	for _, hole := range poly[1:] {
		writeKMLBoundary(b, "innerBoundaryIs", hole)
	}
	b.WriteString("</Polygon>\n")
}

func writeKMLBoundary(b *strings.Builder, tag string, ring [][]float64) {
	if len(ring) < 3 {
		return
	}
	b.WriteString("<" + tag + "><LinearRing><coordinates>")
	for _, p := range ring {
		if len(p) < 2 {
			continue
		}
		b.WriteString(strconv.FormatFloat(p[0], 'f', 8, 64))
		b.WriteByte(',')
		b.WriteString(strconv.FormatFloat(p[1], 'f', 8, 64))
		b.WriteString(",0 ")
	}
	first, last := ring[0], ring[len(ring)-1]
	if len(first) >= 2 && len(last) >= 2 && (first[0] != last[0] || first[1] != last[1]) {
		b.WriteString(strconv.FormatFloat(first[0], 'f', 8, 64))
		b.WriteByte(',')
		b.WriteString(strconv.FormatFloat(first[1], 'f', 8, 64))
		b.WriteString(",0 ")
	}
	b.WriteString("</coordinates></LinearRing></" + tag + ">")
}

func kmlXMLEscape(v string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(v))
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return "área identificada"
}

func sumEmbargoOverlap(items []EmbargoFinding) float64 {
	var total float64
	for _, x := range items {
		total += x.OverlapAreaHa
	}
	return total
}

func sumTerritoryOverlap(items []TerritoryFinding) float64 {
	var total float64
	for _, x := range items {
		total += x.OverlapAreaHa
	}
	return total
}

func sumUCOverlap(items []UCFindings) float64 {
	var total float64
	for _, x := range items {
		total += x.OverlapAreaHa
	}
	return total
}

func buildDossierSourcesText(car CARResult) []byte {
	var b strings.Builder
	b.WriteString("VIA VERDE CAR — FONTES, DATA DA CONSULTA E AVISOS\r\n\r\n")
	fmt.Fprintf(&b, "CAR: %s\r\n", car.CAR)
	fmt.Fprintf(&b, "Consulta principal: %s\r\n", firstNonEmpty(car.CheckedAt, "não informada"))
	fmt.Fprintf(&b, "SICAR público: %s\r\n", firstNonEmpty(car.OfficialURL, car.Source))
	fmt.Fprintf(&b, "Temas SICAR: %s\r\n", firstNonEmpty(car.Themes.SourceURL, "não consultado"))
	fmt.Fprintf(&b, "IBAMA/SISCOM: %s\r\n", firstNonEmpty(car.Environment.IBAMASourceURL, "não consultado"))
	fmt.Fprintf(&b, "FUNAI: %s\r\n", firstNonEmpty(car.Environment.FUNAISourceURL, "não consultado"))
	fmt.Fprintf(&b, "ICMBio/INDE: %s\r\n", firstNonEmpty(car.Environment.ICMBioSourceURL, "não consultado"))
	fmt.Fprintf(&b, "MMA/MCR-PRODES: %s\r\n", firstNonEmpty(car.Environment.MCRSourceURL, "não consultado"))
	if car.Environment.MCRDataUpdated != "" {
		fmt.Fprintf(&b, "Atualização informada da base MMA/MCR: %s\r\n", car.Environment.MCRDataUpdated)
	}
	b.WriteString("\r\nRESULTADOS RESUMIDOS\r\n")
	fmt.Fprintf(&b, "Embargos IBAMA: %d ocorrência(s), %.4f ha de sobreposição estimada somada.\r\n", car.Environment.IBAMAEmbargoCount, sumEmbargoOverlap(car.Environment.IBAMAEmbargos))
	fmt.Fprintf(&b, "Terras Indígenas FUNAI: %d ocorrência(s), %.4f ha de sobreposição estimada somada.\r\n", car.Environment.IndigenousCount, sumTerritoryOverlap(car.Environment.IndigenousFindings))
	fmt.Fprintf(&b, "UCs federais ICMBio: %d ocorrência(s), %.4f ha de sobreposição estimada somada.\r\n", car.Environment.FederalUCCount, sumUCOverlap(car.Environment.FederalUCFindings))
	if car.Environment.MCRChecked {
		if car.Environment.MCRListed {
			b.WriteString("MMA/MCR-PRODES: CAR localizado na lista pública consultada.\r\n")
		} else {
			b.WriteString("MMA/MCR-PRODES: CAR não localizado na lista pública consultada.\r\n")
		}
	}
	if len(car.Environment.Warnings) > 0 || len(car.Themes.Warnings) > 0 {
		b.WriteString("\r\nAVISOS DE CONSULTA\r\n")
		for _, w := range car.Environment.Warnings {
			b.WriteString("- " + w + "\r\n")
		}
		for _, w := range car.Themes.Warnings {
			b.WriteString("- " + w + "\r\n")
		}
	}
	b.WriteString("\r\nIMPORTANTE\r\n")
	b.WriteString("Sobreposição espacial é uma triagem auxiliar e, isoladamente, não determina irregularidade. Confirme ocorrências, limites, datas, situação jurídica e documentos diretamente nas fontes oficiais antes de concluir análise de crédito, ambiental ou fundiária.\r\n")
	return []byte(b.String())
}
