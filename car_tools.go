package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	carPublicURL    = "https://consulta.car.gov.br/geoservices"
	carMeuImovelURL = "https://meuimovelrural.sistema.gov.br/#/"
	carWFSURL       = "https://geoserver.car.gov.br/geoserver/sicar/ows"
)

var carPattern = regexp.MustCompile(`^([A-Z]{2})-([0-9]{7})-([A-F0-9]{4}(?:\.[A-F0-9]{4}){7})$`)

type CARView struct {
	Query            string
	CAR              string
	Valid            bool
	LookedUp         bool
	Found            bool
	Error            string
	UF               string
	MunicipalityCode string
	Municipality     string
	AreaHa           float64
	Status           string
	Condition        string
	PropertyType     string
	FiscalModules    float64
	HasGeometry      bool
	MapSVG           template.HTML
	PublicURL        string
	MeuImovelURL     string
	KMLURL           string
}

type carGeoJSON struct {
	Type     string          `json:"type"`
	Features []carGeoFeature `json:"features"`
}

type carGeoFeature struct {
	Type       string             `json:"type"`
	Properties map[string]any     `json:"properties"`
	Geometry   carGeoJSONGeometry `json:"geometry"`
}

type carGeoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func (a *App) carPage(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("number"))
	v := CARView{Query: q, PublicURL: carPublicURL, MeuImovelURL: carMeuImovelURL}
	if q == "" {
		a.render(w, r, "car", ViewData{Title: "Consulta CAR", Data: v})
		return
	}

	car, uf, muni, err := normalizeCAR(q)
	v.LookedUp = true
	if err != nil {
		v.Error = err.Error()
		a.render(w, r, "car", ViewData{Title: "Consulta CAR", Data: v})
		return
	}
	v.Valid = true
	v.CAR = car
	v.Query = car
	v.UF = uf
	v.MunicipalityCode = muni
	v.KMLURL = "/car/kml?number=" + url.QueryEscape(car)

	ctx, cancel := context.WithTimeout(r.Context(), 18*time.Second)
	defer cancel()
	feature, err := lookupCARPublic(ctx, car, uf)
	if err != nil {
		v.Error = "O número é válido, mas a consulta automática da base pública do SICAR não respondeu agora. Você ainda pode abrir a Consulta Pública ou o Meu Imóvel Rural pelos botões abaixo."
		a.render(w, r, "car", ViewData{Title: "Consulta CAR", Data: v})
		return
	}
	if feature == nil {
		v.Error = "O número é válido, mas o imóvel não foi localizado na camada pública consultada. Confira o código ou use a Consulta Pública oficial."
		a.render(w, r, "car", ViewData{Title: "Consulta CAR", Data: v})
		return
	}

	v.Found = true
	v.Municipality = carStringProp(feature.Properties, "nom_munici", "nom_municipio", "municipio", "nm_muni", "nome_municipio")
	v.Status = carStringProp(feature.Properties, "ind_status", "situacao", "status")
	v.Condition = carStringProp(feature.Properties, "des_condic", "condicao", "descricao_condicao")
	v.PropertyType = carStringProp(feature.Properties, "tipo_imove", "tipo_imovel", "des_tipo", "tipo")
	v.AreaHa = carFloatProp(feature.Properties, "num_area", "area_ha", "area")
	v.FiscalModules = carFloatProp(feature.Properties, "mod_fiscal", "modulos_fiscais")
	v.HasGeometry = carGeometryUsable(feature.Geometry)
	if v.AreaHa <= 0 && v.HasGeometry {
		v.AreaHa = carGeometryAreaHa(feature.Geometry)
	}
	if v.HasGeometry {
		v.MapSVG = template.HTML(carGeometrySVG(feature.Geometry))
	}
	a.render(w, r, "car", ViewData{Title: "Consulta CAR", Data: v})
}

func (a *App) carKML(w http.ResponseWriter, r *http.Request) {
	car, uf, _, err := normalizeCAR(r.URL.Query().Get("number"))
	if err != nil {
		http.Error(w, "Número do CAR inválido.", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 22*time.Second)
	defer cancel()
	feature, err := lookupCARPublic(ctx, car, uf)
	if err != nil {
		http.Error(w, "A base pública do SICAR não respondeu. Tente novamente ou use a Consulta Pública oficial.", http.StatusBadGateway)
		return
	}
	if feature == nil || !carGeometryUsable(feature.Geometry) {
		http.Error(w, "Geometria pública não localizada para este CAR.", http.StatusNotFound)
		return
	}
	kml, err := carGeometryKML(car, feature.Geometry)
	if err != nil {
		http.Error(w, "Não foi possível gerar o KML.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.google-earth.kml+xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="CAR_%s.kml"`, safeCARFilename(car)))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(kml)
}

func normalizeCAR(v string) (car, uf, municipality string, err error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "_", "-")
	m := carPattern.FindStringSubmatch(v)
	if len(m) != 4 {
		return "", "", "", errors.New("Digite o número completo do CAR no formato UF-0000000-XXXX.XXXX.XXXX.XXXX.XXXX.XXXX.XXXX.XXXX.")
	}
	if !validUF(m[1]) {
		return "", "", "", errors.New("A UF informada no número do CAR não é válida.")
	}
	return v, m[1], m[2], nil
}

func validUF(v string) bool {
	_, ok := map[string]bool{"AC":true,"AL":true,"AP":true,"AM":true,"BA":true,"CE":true,"DF":true,"ES":true,"GO":true,"MA":true,"MT":true,"MS":true,"MG":true,"PA":true,"PB":true,"PR":true,"PE":true,"PI":true,"RJ":true,"RN":true,"RS":true,"RO":true,"RR":true,"SC":true,"SP":true,"SE":true,"TO":true}[v]
	return ok
}

func lookupCARPublic(ctx context.Context, car, uf string) (*carGeoFeature, error) {
	versions := []struct{ version, typeKey string }{{"2.0.0", "typeNames"}, {"1.1.0", "typeName"}}
	var lastErr error
	for _, v := range versions {
		feature, err := lookupCARPublicVersion(ctx, car, uf, v.version, v.typeKey)
		if err == nil {
			return feature, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func lookupCARPublicVersion(ctx context.Context, car, uf, version, typeKey string) (*carGeoFeature, error) {
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", version)
	params.Set("request", "GetFeature")
	params.Set(typeKey, "sicar:sicar_imoveis_"+strings.ToLower(uf))
	params.Set("outputFormat", "application/json")
	params.Set("srsName", "EPSG:4326")
	if version == "2.0.0" {
		params.Set("count", "2")
	} else {
		params.Set("maxFeatures", "2")
	}
	params.Set("CQL_FILTER", "cod_imovel='"+strings.ReplaceAll(car, "'", "''")+"'")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, carWFSURL+"?"+params.Encode(), nil)
	if err != nil { return nil, err }
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerde-CreditoRural/1.0")

	client := &http.Client{Timeout: 18 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("SICAR respondeu HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil { return nil, err }
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil { return nil, err }
	if len(fc.Features) == 0 { return nil, nil }
	return &fc.Features[0], nil
}

func carStringProp(props map[string]any, keys ...string) string {
	for _, k := range keys {
		for pk, pv := range props {
			if !strings.EqualFold(pk, k) || pv == nil { continue }
			s := strings.TrimSpace(fmt.Sprint(pv))
			if s != "" && s != "<nil>" { return s }
		}
	}
	return ""
}

func carFloatProp(props map[string]any, keys ...string) float64 {
	for _, k := range keys {
		for pk, pv := range props {
			if !strings.EqualFold(pk, k) || pv == nil { continue }
			switch x := pv.(type) {
			case float64:
				return x
			case json.Number:
				f, _ := x.Float64()
				return f
			default:
				s := strings.ReplaceAll(strings.TrimSpace(fmt.Sprint(x)), ",", ".")
				f, _ := strconv.ParseFloat(s, 64)
				if f != 0 { return f }
			}
		}
	}
	return 0
}

func carGeometryUsable(g carGeoJSONGeometry) bool {
	if len(g.Coordinates) == 0 { return false }
	return strings.EqualFold(g.Type, "Polygon") || strings.EqualFold(g.Type, "MultiPolygon")
}

func carGeometryPolygons(g carGeoJSONGeometry) ([][][][]float64, error) {
	switch strings.ToLower(g.Type) {
	case "polygon":
		var p [][][]float64
		if err := json.Unmarshal(g.Coordinates, &p); err != nil { return nil, err }
		return [][][][]float64{p}, nil
	case "multipolygon":
		var mp [][][][]float64
		if err := json.Unmarshal(g.Coordinates, &mp); err != nil { return nil, err }
		return mp, nil
	default:
		return nil, errors.New("geometria não suportada")
	}
}

func carGeometryKML(car string, g carGeoJSONGeometry) ([]byte, error) {
	polys, err := carGeometryPolygons(g)
	if err != nil { return nil, err }
	var b bytes.Buffer
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<kml xmlns=\"http://www.opengis.net/kml/2.2\"><Document><name>")
	_ = xml.EscapeText(&b, []byte("CAR "+car))
	b.WriteString("</name><Style id=\"car\"><LineStyle><width>2</width></LineStyle><PolyStyle><fill>0</fill></PolyStyle></Style><Placemark><name>")
	_ = xml.EscapeText(&b, []byte(car))
	b.WriteString("</name><styleUrl>#car</styleUrl>")
	if len(polys) > 1 { b.WriteString("<MultiGeometry>") }
	for _, poly := range polys {
		if len(poly) == 0 { continue }
		b.WriteString("<Polygon><tessellate>1</tessellate>")
		for ri, ring := range poly {
			if len(ring) < 3 { continue }
			if ri == 0 { b.WriteString("<outerBoundaryIs>") } else { b.WriteString("<innerBoundaryIs>") }
			b.WriteString("<LinearRing><coordinates>")
			for _, c := range ring {
				if len(c) < 2 { continue }
				fmt.Fprintf(&b, "%.8f,%.8f,0 ", c[0], c[1])
			}
			b.WriteString("</coordinates></LinearRing>")
			if ri == 0 { b.WriteString("</outerBoundaryIs>") } else { b.WriteString("</innerBoundaryIs>") }
		}
		b.WriteString("</Polygon>")
	}
	if len(polys) > 1 { b.WriteString("</MultiGeometry>") }
	b.WriteString("</Placemark></Document></kml>")
	return b.Bytes(), nil
}

func carGeometryAreaHa(g carGeoJSONGeometry) float64 {
	polys, err := carGeometryPolygons(g)
	if err != nil { return 0 }
	area := 0.0
	for _, poly := range polys {
		for ri, ring := range poly {
			a := math.Abs(carRingAreaM2(ring))
			if ri == 0 { area += a } else { area -= a }
		}
	}
	if area < 0 { area = 0 }
	return area / 10000
}

func carRingAreaM2(ring [][]float64) float64 {
	if len(ring) < 3 { return 0 }
	lat0 := 0.0
	for _, p := range ring { if len(p) >= 2 { lat0 += p[1] } }
	lat0 /= float64(len(ring))
	const r = 6378137.0
	cosLat := math.Cos(lat0 * math.Pi / 180)
	area := 0.0
	for i := 0; i < len(ring); i++ {
		j := (i + 1) % len(ring)
		if len(ring[i]) < 2 || len(ring[j]) < 2 { continue }
		x1 := r * ring[i][0] * math.Pi / 180 * cosLat
		y1 := r * ring[i][1] * math.Pi / 180
		x2 := r * ring[j][0] * math.Pi / 180 * cosLat
		y2 := r * ring[j][1] * math.Pi / 180
		area += x1*y2 - x2*y1
	}
	return area / 2
}

func carGeometrySVG(g carGeoJSONGeometry) string {
	polys, err := carGeometryPolygons(g)
	if err != nil { return "" }
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range polys {
		for _, ring := range poly {
			for _, p := range ring {
				if len(p) < 2 { continue }
				if p[0] < minX { minX = p[0] }
				if p[0] > maxX { maxX = p[0] }
				if p[1] < minY { minY = p[1] }
				if p[1] > maxY { maxY = p[1] }
			}
		}
	}
	if math.IsInf(minX, 1) || maxX == minX || maxY == minY { return "" }
	const w, h, pad = 760.0, 360.0, 20.0
	sx := (w - 2*pad) / (maxX - minX)
	sy := (h - 2*pad) / (maxY - minY)
	scale := math.Min(sx, sy)
	cx := (minX + maxX) / 2
	cy := (minY + maxY) / 2
	var b strings.Builder
	b.WriteString(`<svg viewBox="0 0 760 360" role="img" aria-label="Prévia do perímetro do CAR"><rect x="0" y="0" width="760" height="360" rx="16" fill="#eef5f1"/>`)
	for _, poly := range polys {
		for ri, ring := range poly {
			if len(ring) < 3 { continue }
			b.WriteString(`<polygon points="`)
			for _, p := range ring {
				if len(p) < 2 { continue }
				x := w/2 + (p[0]-cx)*scale
				y := h/2 - (p[1]-cy)*scale
				fmt.Fprintf(&b, "%.2f,%.2f ", x, y)
			}
			if ri == 0 {
				b.WriteString(`" fill="#d9ece2" stroke="#16744f" stroke-width="2"/>`)
			} else {
				b.WriteString(`" fill="#eef5f1" stroke="#16744f" stroke-width="1.5"/>`)
			}
		}
	}
	b.WriteString(`</svg>`)
	return strings.ReplaceAll(b.String(), `\"`, `"`)
}

func safeCARFilename(v string) string {
	r := strings.NewReplacer("-", "_", ".", "_")
	return r.Replace(v)
}
