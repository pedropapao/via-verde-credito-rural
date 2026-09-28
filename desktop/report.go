package main

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ExportCARReport(propertyID int64, car CARResult, kml KMLResult, comparison GeometryComparison) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	p, err := a.GetProperty(propertyID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(kml.GeoJSON) == "" && strings.TrimSpace(p.KMLPath) != "" {
		if savedKML, loadErr := a.LoadPropertyKML(propertyID); loadErr == nil {
			kml = savedKML
		}
	}
	if kml.AreaHa > 0 && car.AreaHa > 0 && strings.TrimSpace(comparison.Summary) == "" {
		comparison = a.CompareKMLWithCAR(kml, car)
	}
	pdf := buildCARProfessionalPDF(p, car, kml, comparison)
	name := "Demonstrativo_CAR_" + safeCARFilename(car.CAR) + ".pdf"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "Salvar demonstrativo técnico", DefaultFilename: name, Filters: []runtime.FileFilter{{DisplayName: "PDF", Pattern: "*.pdf"}}})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

type pdfCanvas struct{ b strings.Builder }

func (c *pdfCanvas) text(x, y, size float64, bold bool, value string) {
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(&c.b, "BT /%s %.1f Tf %.2f %.2f Td (%s) Tj ET\n", font, size, x, y, pdfEscapeWin1252(value))
}
func (c *pdfCanvas) line(x1, y1, x2, y2 float64) {
	fmt.Fprintf(&c.b, "%.2f %.2f m %.2f %.2f l S\n", x1, y1, x2, y2)
}
func (c *pdfCanvas) rect(x, y, w, h float64, fill bool) {
	op := "S"
	if fill {
		op = "f"
	}
	fmt.Fprintf(&c.b, "%.2f %.2f %.2f %.2f re %s\n", x, y, w, h, op)
}
func (c *pdfCanvas) wrapped(x, y, size float64, bold bool, value string, width int, leading float64) float64 {
	for _, line := range pdfWrap(value, width) {
		c.text(x, y, size, bold, line)
		y -= leading
	}
	return y
}

func buildCARTechnicalPDF(p Property, car CARResult, kml KMLResult, cmp GeometryComparison) []byte {
	var c pdfCanvas
	c.b.WriteString("0.055 0.42 0.29 rg\n")
	c.rect(0, 760, 595, 82, true)
	c.b.WriteString("1 1 1 rg\n")
	c.text(38, 808, 18, true, "VIA VERDE CAR")
	c.text(38, 786, 10, false, "Demonstrativo tecnico de conferencia cadastral e geometrica")
	c.b.WriteString("0.10 0.18 0.14 rg\n")
	c.text(38, 730, 8, true, "IMOVEL")
	c.text(38, 708, 13, true, p.Name)
	c.b.WriteString("0.82 0.87 0.84 RG 0.7 w\n")
	c.line(38, 693, 557, 693)
	y := 668.0
	row := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			value = "Nao informado"
		}
		c.b.WriteString("0.35 0.42 0.39 rg\n")
		c.text(40, y, 8, true, label)
		c.b.WriteString("0.08 0.15 0.12 rg\n")
		y = c.wrapped(172, y, 9, false, value, 61, 12)
		y -= 5
	}
	row("Cliente", p.ClientName)
	row("Municipio / UF", strings.Trim(strings.TrimSpace(p.Municipality+" / "+p.UF), " /"))
	row("Matricula", p.Registry)
	row("CAR", car.CAR)
	row("Situacao SICAR", car.Status)
	row("Condicao", car.Condition)
	row("Tipo do imovel", car.PropertyType)
	row("Area SICAR", fmtBR(car.AreaHa, 4)+" ha")
	row("Area geometria SICAR", fmtBR(car.GeometryAreaHa, 4)+" ha")
	row("Perimetro SICAR", fmtBR(car.PerimeterM/1000, 3)+" km")
	if car.AutoKMLPath != "" {
		row("KML SICAR", "Gerado e salvo automaticamente pelo aplicativo")
	}
	if car.Environment.IBAMAChecked {
		row("Triagem IBAMA", fmt.Sprintf("%d intersecao(oes) espacial(is) com areas de embargo SISCOM/IBAMA; soma bruta das sobreposicoes estimadas: %s ha", car.Environment.IBAMAEmbargoCount, fmtBR(sumEmbargoOverlap(car.Environment.IBAMAEmbargos), 4)))
	}
	if car.Environment.FUNAIChecked {
		row("Triagem FUNAI", fmt.Sprintf("%d intersecao(oes) espacial(is) com Terras Indigenas; soma bruta das sobreposicoes estimadas: %s ha", car.Environment.IndigenousCount, fmtBR(sumTerritoryOverlap(car.Environment.IndigenousFindings), 4)))
	}
	if car.Environment.ICMBioChecked {
		row("Triagem ICMBio", fmt.Sprintf("%d intersecao(oes) espacial(is) com Unidades de Conservacao federais; soma bruta das sobreposicoes estimadas: %s ha", car.Environment.FederalUCCount, fmtBR(sumUCOverlap(car.Environment.FederalUCFindings), 4)))
	}
	if car.Environment.MCRChecked {
		statusMCR := "CAR nao localizado na lista publica MMA/MCR consultada"
		if car.Environment.MCRListed {
			statusMCR = "CAR localizado na lista publica MMA/MCR vinculada a verificacoes PRODES"
		}
		row("MMA / MCR-PRODES", statusMCR)
	}
	if car.CenterLat != 0 || car.CenterLon != 0 {
		row("Centro aproximado", fmt.Sprintf("%.6f, %.6f", car.CenterLat, car.CenterLon))
	}
	if len(car.Themes.Themes) > 0 {
		parts := []string{}
		for _, code := range []string{"APP", "RESERVA_LEGAL", "VEGETACAO_NATIVA", "AREA_CONSOLIDADA"} {
			if m, ok := car.Themes.Themes[code]; ok && m.Available {
				parts = append(parts, m.Code+" "+fmtBR(m.AreaHa, 4)+" ha")
			}
		}
		if len(parts) > 0 {
			row("Temas SICAR", strings.Join(parts, "; "))
		}
	}

	if kml.AreaHa > 0 {
		row("KML local", fmtBR(kml.AreaHa, 4)+" ha; "+fmtBR(kml.PerimeterM/1000, 3)+" km de perimetro")
		comparisonText := cmp.Summary+" Diferenca de area: "+fmtBR(cmp.AreaDifferenceHa, 4)+" ha ("+fmtBR(cmp.AreaDifferencePct, 2)+"%). Distancia entre centros: "+fmtBR(cmp.CenterDistanceM, 0)+" m."
		if cmp.OverlapMethod != "" {
			comparisonText += " Intersecao estimada: "+fmtBR(cmp.IntersectionAreaHa, 4)+" ha; KML dentro do CAR: "+fmtBR(cmp.KMLInsideCARPct, 1)+"%; CAR dentro do KML: "+fmtBR(cmp.CARInsideKMLPct, 1)+"%."
		}
		row("Comparacao KML x CAR", comparisonText)
	}
	mapTop := math.Min(y-5, 390.0)
	mapBottom := 205.0
	if mapTop > mapBottom+70 && car.GeoJSON != "" {
		c.b.WriteString("0.08 0.30 0.21 RG 1 w\n")
		c.rect(40, mapBottom, 515, mapTop-mapBottom, false)
		c.b.WriteString("0.30 0.38 0.34 rg\n")
		c.text(48, mapTop-18, 8, true, "PERIMETRO PUBLICO DO SICAR")
		drawDossierGeometry(&c, car, kml, 54, mapBottom+18, 487, mapTop-mapBottom-48)
	}
	c.b.WriteString("0.94 0.97 0.95 rg\n")
	c.rect(38, 64, 519, 112, true)
	c.b.WriteString("0.08 0.30 0.21 rg\n")
	c.text(48, 154, 8, true, "IMPORTANTE")
	c.b.WriteString("0.20 0.27 0.23 rg\n")
	note := "Documento tecnico gerado pelo Via Verde CAR a partir de dados locais e da camada publica consultada do SICAR. Nao substitui o Demonstrativo oficial do CAR, certidao, analise ambiental ou documento emitido pelo orgao competente."
	c.wrapped(48, 137, 8, false, note, 102, 11)
	c.text(48, 79, 7, false, "Gerado em "+time.Now().Format("02/01/2006 15:04")+" - Via Verde CAR v"+AppVersion)
	return assembleSimplePDF(c.b.String())
}

func drawDossierGeometry(c *pdfCanvas, car CARResult, kml KMLResult, x, y, w, h float64) {
	type mapLayer struct {
		label string
		raw   string
		rgb   string
		width float64
		bound bool
	}
	layers := []mapLayer{
		{label: "CAR SICAR", raw: car.GeoJSON, rgb: "0.055 0.42 0.29", width: 1.7, bound: true},
	}
	if strings.TrimSpace(kml.GeoJSON) != "" {
		layers = append(layers, mapLayer{label: "KML cliente", raw: kml.GeoJSON, rgb: "0.84 0.40 0.08", width: 1.2, bound: true})
	}
	themeDefs := []struct {
		code, label, rgb string
	}{
		{"APP", "APP", "0.23 0.51 0.96"},
		{"RESERVA_LEGAL", "Reserva Legal", "0.09 0.40 0.20"},
		{"VEGETACAO_NATIVA", "Vegetacao", "0.08 0.33 0.18"},
		{"AREA_CONSOLIDADA", "Area consolidada", "0.76 0.25 0.05"},
	}
	for _, def := range themeDefs {
		if m, ok := car.Themes.Themes[def.code]; ok && strings.TrimSpace(m.GeoJSON) != "" {
			layers = append(layers, mapLayer{label: def.label, raw: m.GeoJSON, rgb: def.rgb, width: 0.8})
		}
	}
	for _, item := range car.Environment.IBAMAEmbargos {
		if strings.TrimSpace(item.GeoJSON) != "" {
			layers = append(layers, mapLayer{label: "IBAMA", raw: item.GeoJSON, rgb: "0.72 0.11 0.11", width: 1.1})
		}
	}
	for _, item := range car.Environment.IndigenousFindings {
		if strings.TrimSpace(item.GeoJSON) != "" {
			layers = append(layers, mapLayer{label: "FUNAI", raw: item.GeoJSON, rgb: "0.49 0.13 0.81", width: 1.0})
		}
	}
	for _, item := range car.Environment.FederalUCFindings {
		if strings.TrimSpace(item.GeoJSON) != "" {
			layers = append(layers, mapLayer{label: "UC federal", raw: item.GeoJSON, rgb: "0.01 0.41 0.63", width: 1.0})
		}
	}

	type parsedLayer struct {
		label string
		polys [][][][]float64
		rgb   string
		width float64
		bound bool
	}
	var parsed []parsedLayer
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)

	addBounds := func(polys [][][][]float64) {
		for _, poly := range polys {
			for _, ring := range poly {
				for _, p := range ring {
					if len(p) < 2 {
						continue
					}
					minX = math.Min(minX, p[0])
					maxX = math.Max(maxX, p[0])
					minY = math.Min(minY, p[1])
					maxY = math.Max(maxY, p[1])
				}
			}
		}
	}

	for _, layer := range layers {
		if strings.TrimSpace(layer.raw) == "" {
			continue
		}
		var all [][][][]float64
		for _, feature := range geoJSONFeatures(layer.raw) {
			polys, err := carGeometryPolygons(feature.Geometry)
			if err == nil {
				all = append(all, polys...)
			}
		}
		if len(all) == 0 {
			continue
		}
		if layer.bound {
			addBounds(all)
		}
		parsed = append(parsed, parsedLayer{label: layer.label, polys: all, rgb: layer.rgb, width: layer.width, bound: layer.bound})
	}
	if len(parsed) == 0 {
		return
	}
	if math.IsInf(minX, 1) || maxX <= minX || maxY <= minY {
		for _, item := range parsed {
			addBounds(item.polys)
		}
	}
	if math.IsInf(minX, 1) || maxX <= minX || maxY <= minY {
		return
	}

	scale := math.Min(w/(maxX-minX), h/(maxY-minY))
	offX := x + (w-(maxX-minX)*scale)/2
	offY := y + (h-(maxY-minY)*scale)/2

	// Recorta camadas externas ao quadro do imóvel para evitar que grandes
	// polígonos de TI, UC ou embargo alterem a escala principal do mapa.
	c.b.WriteString("q\n")
	fmt.Fprintf(&c.b, "%.2f %.2f %.2f %.2f re W n\n", x, y, w, h)
	for _, item := range parsed {
		c.b.WriteString(item.rgb + " RG\n")
		fmt.Fprintf(&c.b, "%.1f w\n", item.width)
		for _, poly := range item.polys {
			for _, ring := range poly {
				if len(ring) < 2 {
					continue
				}
				step := 1
				if len(ring) > 700 {
					step = int(math.Ceil(float64(len(ring)) / 700))
				}
				started := false
				for i := 0; i < len(ring); i += step {
					p := ring[i]
					if len(p) < 2 {
						continue
					}
					px := offX + (p[0]-minX)*scale
					py := offY + (p[1]-minY)*scale
					if !started {
						fmt.Fprintf(&c.b, "%.2f %.2f m\n", px, py)
						started = true
					} else {
						fmt.Fprintf(&c.b, "%.2f %.2f l\n", px, py)
					}
				}
				if started {
					c.b.WriteString("h S\n")
				}
			}
		}
	}
	c.b.WriteString("Q\n")

	// Legenda compacta, sem repetir entradas iguais.
	seen := map[string]bool{}
	legendX, legendY := x+5.0, y+3.0
	colW := 92.0
	col := 0
	row := 0
	for _, item := range parsed {
		if seen[item.label] {
			continue
		}
		seen[item.label] = true
		lx := legendX + float64(col)*colW
		ly := legendY + float64(row)*10
		c.b.WriteString(item.rgb + " rg\n")
		c.rect(lx, ly+2, 7, 3, true)
		c.b.WriteString("0.18 0.25 0.21 rg\n")
		c.text(lx+10, ly, 5.5, false, item.label)
		col++
		if col >= 5 {
			col = 0
			row++
		}
		if row >= 2 {
			break
		}
	}
}

func fmtBR(v float64, dec int) string {
	if v == 0 {
		return "0"
	}
	return strings.ReplaceAll(fmt.Sprintf("%.*f", dec, v), ".", ",")
}
func pdfWrap(s string, max int) []string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return []string{""}
	}
	words := strings.Fields(s)
	lines := []string{}
	line := ""
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if len([]rune(candidate)) <= max || line == "" {
			line = candidate
			continue
		}
		lines = append(lines, line)
		line = word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
func pdfEscapeWin1252(s string) string {
	var b strings.Builder
	for _, r := range s {
		var by byte
		switch {
		case r >= 32 && r <= 126:
			by = byte(r)
		case r >= 160 && r <= 255:
			by = byte(r)
		default:
			switch r {
			case '–', '—':
				by = '-'
			case '“', '”':
				by = '"'
			case '‘', '’':
				by = '\''
			case '•':
				by = 149
			default:
				if unicode.IsSpace(r) {
					by = ' '
				} else {
					by = '?'
				}
			}
		}
		switch by {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteByte(by)
		default:
			b.WriteByte(by)
		}
	}
	return b.String()
}
func assembleSimplePDF(content string) []byte {
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, 7)
	writeObj := func(n int, body string) { offsets[n] = out.Len(); fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", n, body) }
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >>")
	writeObj(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	writeObj(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	writeObj(6, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))
	xref := out.Len()
	out.WriteString("xref\n0 7\n0000000000 65535 f \n")
	for i := 1; i <= 6; i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size 7 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xref)
	return out.Bytes()
}
