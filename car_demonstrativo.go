package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode"
)

type carDemonstrativoData struct {
	CAR           string
	UF            string
	Municipio     string
	AreaHa        float64
	Modulos       float64
	Status        string
	Condicao      string
	Tipo          string
	DataCadastro  string
	DataAtualiza  string
	PerimetroM    float64
	CentroLat     float64
	CentroLon     float64
	Geometry      carGeoJSONGeometry
	ConsultadoEm  string
}

// carDemonstrativoPDF gera, sem API paga e sem armazenar arquivos, um PDF tecnico
// a partir da camada publica do SICAR. Ele e propositalmente identificado como
// documento tecnico Via Verde: o Demonstrativo oficial continua sendo emitido
// pelo proprio SICAR.
func (a *App) carDemonstrativoPDF(w http.ResponseWriter, r *http.Request) {
	car, uf, _, err := normalizeCAR(r.URL.Query().Get("number"))
	if err != nil {
		http.Error(w, "Numero do CAR invalido.", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 22*time.Second)
	defer cancel()
	feature, err := lookupCARPublic(ctx, car, uf)
	if err != nil {
		http.Error(w, "A base publica do SICAR nao respondeu. Tente novamente em instantes.", http.StatusBadGateway)
		return
	}
	if feature == nil {
		http.Error(w, "CAR nao localizado na camada publica consultada.", http.StatusNotFound)
		return
	}

	data := carDemonstrativoFromFeature(car, uf, *feature)
	pdf, err := buildCARTechnicalPDF(data)
	if err != nil {
		http.Error(w, "Nao foi possivel gerar o demonstrativo.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="Demonstrativo_CAR_%s.pdf"`, safeCARFilename(car)))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(pdf)
}

func (a *App) carDemonstrativoJSON(w http.ResponseWriter, r *http.Request) {
	car, uf, _, err := normalizeCAR(r.URL.Query().Get("number"))
	if err != nil {
		http.Error(w, "Numero do CAR invalido.", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 22*time.Second)
	defer cancel()
	feature, err := lookupCARPublic(ctx, car, uf)
	if err != nil {
		http.Error(w, "A base publica do SICAR nao respondeu.", http.StatusBadGateway)
		return
	}
	if feature == nil {
		http.Error(w, "CAR nao localizado.", http.StatusNotFound)
		return
	}
	data := carDemonstrativoFromFeature(car, uf, *feature)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, `{"car":%q,"uf":%q,"municipio":%q,"area_ha":%.4f,"modulos_fiscais":%.4f,"situacao":%q,"condicao":%q,"tipo":%q,"data_cadastro":%q,"data_atualizacao":%q,"perimetro_m":%.2f,"centro_lat":%.8f,"centro_lon":%.8f}`,
		data.CAR, data.UF, data.Municipio, data.AreaHa, data.Modulos, data.Status, data.Condicao, data.Tipo, data.DataCadastro, data.DataAtualiza, data.PerimetroM, data.CentroLat, data.CentroLon)
}

func carDemonstrativoFromFeature(car, uf string, feature carGeoFeature) carDemonstrativoData {
	area := carFloatProp(feature.Properties, "num_area", "area_ha", "area")
	if area <= 0 && carGeometryUsable(feature.Geometry) {
		area = carGeometryAreaHa(feature.Geometry)
	}
	lat, lon := carGeometryCenter(feature.Geometry)
	return carDemonstrativoData{
		CAR:          car,
		UF:           uf,
		Municipio:    carStringProp(feature.Properties, "municipio", "nom_munici", "nom_municipio", "nm_muni", "nome_municipio"),
		AreaHa:       area,
		Modulos:      carFloatProp(feature.Properties, "mod_fiscal", "modulos_fiscais"),
		Status:       carStatusLabel(carStringProp(feature.Properties, "ind_status", "situacao", "status")),
		Condicao:     carStringProp(feature.Properties, "des_condic", "condicao", "descricao_condicao"),
		Tipo:         carPropertyTypeLabel(carStringProp(feature.Properties, "ind_tipo", "tipo_imove", "tipo_imovel", "des_tipo", "tipo")),
		DataCadastro: carStringProp(feature.Properties, "dat_criaca", "data_cadastro", "data_criacao"),
		DataAtualiza: carStringProp(feature.Properties, "dat_atuali", "data_atualizacao", "data_ultima_atualizacao"),
		PerimetroM:   carGeometryPerimeterM(feature.Geometry),
		CentroLat:    lat,
		CentroLon:    lon,
		Geometry:     feature.Geometry,
		ConsultadoEm: time.Now().Format("02/01/2006 15:04"),
	}
}

func carGeometryCenter(g carGeoJSONGeometry) (lat, lon float64) {
	polys, err := carGeometryPolygons(g)
	if err != nil {
		return 0, 0
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range polys {
		for _, ring := range poly {
			for _, p := range ring {
				if len(p) < 2 {
					continue
				}
				if p[0] < minX { minX = p[0] }
				if p[0] > maxX { maxX = p[0] }
				if p[1] < minY { minY = p[1] }
				if p[1] > maxY { maxY = p[1] }
			}
		}
	}
	if math.IsInf(minX, 1) {
		return 0, 0
	}
	return (minY + maxY) / 2, (minX + maxX) / 2
}

func carGeometryPerimeterM(g carGeoJSONGeometry) float64 {
	polys, err := carGeometryPolygons(g)
	if err != nil {
		return 0
	}
	total := 0.0
	for _, poly := range polys {
		for _, ring := range poly {
			for i := 1; i < len(ring); i++ {
				if len(ring[i-1]) < 2 || len(ring[i]) < 2 { continue }
				total += carHaversineM(ring[i-1][1], ring[i-1][0], ring[i][1], ring[i][0])
			}
		}
	}
	return total
}

func carHaversineM(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371008.8
	p1, p2 := lat1*math.Pi/180, lat2*math.Pi/180
	dp := (lat2-lat1)*math.Pi/180
	dl := (lon2-lon1)*math.Pi/180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

type carPDFCanvas struct {
	b strings.Builder
}

func (c *carPDFCanvas) text(x, y, size float64, bold bool, value string) {
	font := "F1"
	if bold { font = "F2" }
	fmt.Fprintf(&c.b, "BT /%s %.1f Tf %.2f %.2f Td (%s) Tj ET\n", font, size, x, y, pdfEscapeWin1252(value))
}

func (c *carPDFCanvas) line(x1, y1, x2, y2 float64) {
	fmt.Fprintf(&c.b, "%.2f %.2f m %.2f %.2f l S\n", x1, y1, x2, y2)
}

func (c *carPDFCanvas) rect(x, y, w, h float64, fill bool) {
	op := "S"
	if fill { op = "f" }
	fmt.Fprintf(&c.b, "%.2f %.2f %.2f %.2f re %s\n", x, y, w, h, op)
}

func (c *carPDFCanvas) wrappedText(x, y, size float64, bold bool, value string, widthChars int, leading float64) float64 {
	for _, line := range pdfWrap(value, widthChars) {
		c.text(x, y, size, bold, line)
		y -= leading
	}
	return y
}

func buildCARTechnicalPDF(d carDemonstrativoData) ([]byte, error) {
	var c carPDFCanvas
	// Cabecalho verde.
	c.b.WriteString("0.055 0.42 0.29 rg\n")
	c.rect(0, 760, 595, 82, true)
	c.b.WriteString("1 1 1 rg\n")
	c.text(38, 808, 18, true, "DEMONSTRATIVO TECNICO DO CAR")
	c.text(38, 786, 10, false, "Via Verde - consulta de dados publicos do SICAR")

	c.b.WriteString("0.10 0.18 0.14 rg\n")
	c.text(38, 730, 8, true, "CADASTRO AMBIENTAL RURAL")
	c.text(38, 708, 12, true, d.CAR)
	c.b.WriteString("0.82 0.87 0.84 RG 0.7 w\n")
	c.line(38, 693, 557, 693)

	y := 667.0
	row := func(label, value string) {
		if strings.TrimSpace(value) == "" { value = "Nao informado na camada publica" }
		c.b.WriteString("0.35 0.42 0.39 rg\n")
		c.text(40, y, 8, true, label)
		c.b.WriteString("0.08 0.15 0.12 rg\n")
		y = c.wrappedText(172, y, 9, false, value, 63, 12)
		y -= 5
	}
	row("Municipio / UF", strings.TrimSpace(d.Municipio+" / "+d.UF))
	row("Situacao do cadastro", d.Status)
	row("Condicao de analise", d.Condicao)
	row("Tipo do imovel", d.Tipo)
	row("Area declarada", formatCARNumber(d.AreaHa, " ha"))
	row("Modulos fiscais", formatCARNumber(d.Modulos, ""))
	row("Data de cadastro", d.DataCadastro)
	row("Ultima atualizacao", d.DataAtualiza)
	row("Perimetro calculado", formatCARNumber(d.PerimetroM/1000, " km"))
	if d.CentroLat != 0 || d.CentroLon != 0 {
		row("Centro aproximado", fmt.Sprintf("%.6f, %.6f", d.CentroLat, d.CentroLon))
	}

	mapTop := math.Min(y-8, 450)
	mapBottom := 205.0
	if mapTop > mapBottom+80 && carGeometryUsable(d.Geometry) {
		c.b.WriteString("0.08 0.30 0.21 RG 1 w\n")
		c.rect(40, mapBottom, 515, mapTop-mapBottom, false)
		c.b.WriteString("0.30 0.38 0.34 rg\n")
		c.text(48, mapTop-18, 8, true, "PERIMETRO DO IMOVEL - GEOMETRIA PUBLICA SICAR")
		carDrawGeometryPDF(&c, d.Geometry, 54, mapBottom+18, 487, mapTop-mapBottom-48)
	}

	noteY := 165.0
	c.b.WriteString("0.94 0.97 0.95 rg\n")
	c.rect(38, 64, 519, 92, true)
	c.b.WriteString("0.08 0.30 0.21 rg\n")
	c.text(48, noteY-22, 8, true, "IMPORTANTE")
	c.b.WriteString("0.20 0.27 0.23 rg\n")
	n := "Este PDF e um demonstrativo tecnico gerado pelo Via Verde a partir da camada publica do SICAR. Nao substitui o Demonstrativo oficial emitido pelo SICAR. Para o documento oficial, utilize www.car.gov.br/#/consultar."
	c.wrappedText(48, noteY-39, 8, false, n, 102, 11)
	c.text(48, 76, 7, false, "Consulta realizada em "+d.ConsultadoEm+". Nenhum arquivo e armazenado pelo Via Verde.")

	return assembleSimplePDF(c.b.String()), nil
}

func formatCARNumber(v float64, suffix string) string {
	if v <= 0 { return "Nao informado na camada publica" }
	s := fmt.Sprintf("%.2f", v)
	s = strings.ReplaceAll(s, ".", ",")
	return s + suffix
}

func carDrawGeometryPDF(c *carPDFCanvas, g carGeoJSONGeometry, x, y, w, h float64) {
	polys, err := carGeometryPolygons(g)
	if err != nil { return }
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range polys {
		for _, ring := range poly {
			for _, p := range ring {
				if len(p) < 2 { continue }
				minX = math.Min(minX, p[0]); maxX = math.Max(maxX, p[0])
				minY = math.Min(minY, p[1]); maxY = math.Max(maxY, p[1])
			}
		}
	}
	if math.IsInf(minX, 1) || maxX <= minX || maxY <= minY { return }
	sx, sy := w/(maxX-minX), h/(maxY-minY)
	scale := math.Min(sx, sy)
	offX := x + (w-(maxX-minX)*scale)/2
	offY := y + (h-(maxY-minY)*scale)/2
	c.b.WriteString("0.055 0.42 0.29 RG 1.2 w\n")
	for _, poly := range polys {
		for _, ring := range poly {
			if len(ring) < 2 { continue }
			step := 1
			if len(ring) > 900 { step = int(math.Ceil(float64(len(ring))/900.0)) }
			started := false
			for i := 0; i < len(ring); i += step {
				p := ring[i]
				if len(p) < 2 { continue }
				px := offX + (p[0]-minX)*scale
				py := offY + (p[1]-minY)*scale
				if !started {
					fmt.Fprintf(&c.b, "%.2f %.2f m\n", px, py)
					started = true
				} else {
					fmt.Fprintf(&c.b, "%.2f %.2f l\n", px, py)
				}
			}
			if started { c.b.WriteString("h S\n") }
		}
	}
}

func pdfWrap(s string, max int) []string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" { return []string{""} }
	words := strings.Fields(s)
	lines := []string{}
	line := ""
	for _, word := range words {
		candidate := word
		if line != "" { candidate = line + " " + word }
		if len([]rune(candidate)) <= max || line == "" {
			line = candidate
			continue
		}
		lines = append(lines, line)
		line = word
	}
	if line != "" { lines = append(lines, line) }
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
			case '–', '—': by = '-'
			case '“', '”': by = '"'
			case '‘', '’': by = '\''
			case '•': by = 149
			default:
				if unicode.IsSpace(r) { by = ' ' } else { by = '?' }
			}
		}
		switch by {
		case '\\', '(', ')':
			b.WriteByte('\\'); b.WriteByte(by)
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
	writeObj := func(n int, body string) {
		offsets[n] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", n, body)
	}
	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObj(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >>")
	writeObj(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	writeObj(5, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	offsets[6] = out.Len()
	fmt.Fprintf(&out, "6 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len([]byte(content)), content)
	xref := out.Len()
	out.WriteString("xref\n0 7\n0000000000 65535 f \n")
	for i := 1; i <= 6; i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size 7 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xref)
	return out.Bytes()
}
