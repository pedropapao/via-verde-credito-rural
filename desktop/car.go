package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	carWFSURL       = "https://geoserver.car.gov.br/geoserver/sicar/ows"
	carWFSFallback = "https://geoserver.car.gov.br/geoserver/sicar/wfs"
	carPublicURL    = "https://www.car.gov.br/#/consultar"
	carMeuImovelURL = "https://meuimovelrural.sistema.gov.br/#/"
)

var (
	carMunicipalityPattern = regexp.MustCompile(`^[0-9]{7}$`)
	carHexPattern          = regexp.MustCompile(`^[A-F0-9]{32}$`)
)

type CARResult struct {
	CAR              string         `json:"car"`
	UF               string         `json:"uf"`
	MunicipalityCode string         `json:"municipality_code"`
	Municipality     string         `json:"municipality"`
	AreaHa           float64        `json:"area_ha"`
	GeometryAreaHa   float64        `json:"geometry_area_ha"`
	PerimeterM       float64        `json:"perimeter_m"`
	CenterLat        float64        `json:"center_lat"`
	CenterLon        float64        `json:"center_lon"`
	Status           string         `json:"status"`
	Condition        string         `json:"condition"`
	PropertyType     string         `json:"property_type"`
	FiscalModules    float64        `json:"fiscal_modules"`
	DataCadastro     string         `json:"data_cadastro"`
	DataAtualizacao  string         `json:"data_atualizacao"`
	Found            bool           `json:"found"`
	HasGeometry      bool           `json:"has_geometry"`
	GeoJSON          string         `json:"geojson"`
	CheckedAt        string         `json:"checked_at"`
	Source           string         `json:"source"`
	OfficialURL      string         `json:"official_url"`
	MeuImovelURL     string         `json:"meu_imovel_url"`
	GoogleMapsURL    string         `json:"google_maps_url"`
	Checks           []QualityCheck `json:"checks"`
}

type QualityCheck struct {
	Level  string `json:"level"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
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

func (a *App) LookupCAR(number string) (CARResult, error) {
	return a.analyzeCAR(0, number)
}

func (a *App) AnalyzePropertyCAR(propertyID int64, number string) (CARResult, error) {
	return a.analyzeCAR(propertyID, number)
}

func (a *App) analyzeCAR(propertyID int64, number string) (CARResult, error) {
	car, uf, muniCode, err := normalizeCAR(number)
	if err != nil {
		return CARResult{}, err
	}
	result := CARResult{
		CAR: car, UF: uf, MunicipalityCode: muniCode,
		CheckedAt: time.Now().Format(time.RFC3339),
		Source:    "Camada pública SICAR (WFS)", OfficialURL: carPublicURL, MeuImovelURL: carMeuImovelURL,
		Checks: []QualityCheck{{Level: "ok", Title: "Número do CAR", Detail: "Formato do código validado."}},
	}
	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	ctx, cancel := context.WithTimeout(ctx, 22*time.Second)
	defer cancel()
	feature, err := lookupCARPublic(ctx, car, uf)
	if err != nil {
		result.Checks = append(result.Checks, QualityCheck{Level: "error", Title: "Consulta pública indisponível", Detail: err.Error()})
		return result, fmt.Errorf("a base pública do SICAR não respondeu: %w", err)
	}
	if feature == nil {
		result.Checks = append(result.Checks, QualityCheck{Level: "warning", Title: "CAR não localizado", Detail: "O código é válido, mas não apareceu na camada pública consultada."})
		return result, nil
	}
	result.Found = true
	result.Municipality = carStringProp(feature.Properties, "nom_munici", "nom_municipio", "municipio", "nm_muni", "nome_municipio")
	result.Status = carStatusLabel(carStringProp(feature.Properties, "ind_status", "situacao", "status"))
	result.Condition = carStringProp(feature.Properties, "des_condic", "condicao", "descricao_condicao")
	result.PropertyType = carPropertyTypeLabel(carStringProp(feature.Properties, "ind_tipo", "tipo_imove", "tipo_imovel", "des_tipo", "tipo"))
	result.AreaHa = carFloatProp(feature.Properties, "num_area", "area_ha", "area")
	result.FiscalModules = carFloatProp(feature.Properties, "mod_fiscal", "modulos_fiscais")
	result.DataCadastro = carStringProp(feature.Properties, "dat_criaca", "data_cadastro", "data_criacao")
	result.DataAtualizacao = carStringProp(feature.Properties, "dat_atuali", "data_atualizacao", "data_ultima_atualizacao")
	result.HasGeometry = carGeometryUsable(feature.Geometry)
	if result.HasGeometry {
		result.GeometryAreaHa = carGeometryAreaHa(feature.Geometry)
		result.PerimeterM = carGeometryPerimeterM(feature.Geometry)
		result.CenterLat, result.CenterLon = carGeometryCenter(feature.Geometry)
		if result.AreaHa <= 0 {
			result.AreaHa = result.GeometryAreaHa
		}
		b, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: feature.Properties, Geometry: feature.Geometry})
		result.GeoJSON = string(b)
		if result.CenterLat != 0 || result.CenterLon != 0 {
			result.GoogleMapsURL = fmt.Sprintf("https://www.google.com/maps?q=%.8f,%.8f", result.CenterLat, result.CenterLon)
		}
		result.Checks = append(result.Checks, QualityCheck{Level: "ok", Title: "Geometria pública", Detail: "Perímetro encontrado e pronto para mapa/KML."})
	} else {
		result.Checks = append(result.Checks, QualityCheck{Level: "warning", Title: "Sem geometria pública", Detail: "O cadastro foi localizado, mas a camada consultada não retornou um polígono utilizável."})
	}
	if result.Status != "" {
		level := "ok"
		detail := "Situação informada pela camada pública: " + result.Status + "."
		if !strings.EqualFold(result.Status, "Ativo") {
			level = "warning"
		}
		result.Checks = append(result.Checks, QualityCheck{Level: level, Title: "Situação cadastral", Detail: detail})
	}
	if result.AreaHa > 0 && result.GeometryAreaHa > 0 {
		diff := math.Abs(result.AreaHa - result.GeometryAreaHa)
		pct := diff / result.AreaHa * 100
		level := "ok"
		if pct > 2 {
			level = "warning"
		}
		result.Checks = append(result.Checks, QualityCheck{Level: level, Title: "Área declarada × geometria", Detail: fmt.Sprintf("Diferença de %.2f ha (%.2f%%).", diff, pct)})
	}

	if propertyID > 0 && a.db != nil {
		p, pErr := a.GetProperty(propertyID)
		if pErr == nil {
			if p.Municipality != "" && result.Municipality != "" && !samePlace(p.Municipality, result.Municipality) {
				result.Checks = append(result.Checks, QualityCheck{Level: "warning", Title: "Município divergente", Detail: "Imóvel local: " + p.Municipality + "; SICAR: " + result.Municipality + "."})
			}
			if p.DeclaredAreaHa > 0 && result.AreaHa > 0 {
				diff := math.Abs(p.DeclaredAreaHa - result.AreaHa)
				pct := diff / p.DeclaredAreaHa * 100
				level := "ok"
				if pct > 2 {
					level = "warning"
				}
				result.Checks = append(result.Checks, QualityCheck{Level: level, Title: "Área cadastrada × SICAR", Detail: fmt.Sprintf("Local %.4f ha; SICAR %.4f ha; diferença %.4f ha (%.2f%%).", p.DeclaredAreaHa, result.AreaHa, diff, pct)})
			}
		}
		var duplicates int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM properties WHERE car_number=? AND id<>?`, car, propertyID).Scan(&duplicates)
		if duplicates > 0 {
			result.Checks = append(result.Checks, QualityCheck{Level: "warning", Title: "CAR já usado", Detail: "Este número também está vinculado a outro imóvel no cadastro local."})
		}
		blob := marshalJSON(result)
		_, _ = a.db.Exec(`UPDATE properties SET car_number=?,last_car_json=?,updated_at=? WHERE id=?`, car, blob, time.Now().Format(time.RFC3339), propertyID)
		_, _ = a.db.Exec(`INSERT INTO car_checks(property_id,car_number,checked_at,status,condition_text,area_ha,municipality,geometry_json,result_json) VALUES(?,?,?,?,?,?,?,?,?)`, propertyID, car, result.CheckedAt, result.Status, result.Condition, result.AreaHa, result.Municipality, result.GeoJSON, blob)
	}
	return result, nil
}

func samePlace(a, b string) bool {
	norm := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		r := strings.NewReplacer("á", "a", "à", "a", "ã", "a", "â", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c")
		return r.Replace(s)
	}
	return norm(a) == norm(b)
}

func (a *App) ExportCARKML(number string) (string, error) {
	car, uf, _, err := normalizeCAR(number)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	ctx, cancel := context.WithTimeout(ctx, 22*time.Second)
	defer cancel()
	feature, err := lookupCARPublic(ctx, car, uf)
	if err != nil {
		return "", err
	}
	if feature == nil || !carGeometryUsable(feature.Geometry) {
		return "", errors.New("geometria pública não localizada para este CAR")
	}
	data, err := carGeometryKML(car, feature.Geometry)
	if err != nil {
		return "", err
	}
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "Salvar KML do CAR", DefaultFilename: "CAR_" + safeCARFilename(car) + ".kml", Filters: []runtime.FileFilter{{DisplayName: "Google Earth KML", Pattern: "*.kml"}}})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func normalizeCAR(v string) (car, uf, municipality string, err error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "_", "-")
	parts := strings.SplitN(v, "-", 3)
	if len(parts) != 3 || !validUF(parts[0]) || !carMunicipalityPattern.MatchString(parts[1]) {
		return "", "", "", errors.New("digite o CAR completo no formato UF-0000000-XXXX.XXXX.XXXX.XXXX.XXXX.XXXX.XXXX.XXXX")
	}
	compact := strings.ReplaceAll(parts[2], ".", "")
	if !carHexPattern.MatchString(compact) {
		return "", "", "", errors.New("o identificador final do CAR deve possuir 32 caracteres hexadecimais")
	}
	groups := make([]string, 0, 8)
	for i := 0; i < len(compact); i += 4 {
		groups = append(groups, compact[i:i+4])
	}
	return parts[0] + "-" + parts[1] + "-" + strings.Join(groups, "."), parts[0], parts[1], nil
}

func validUF(v string) bool {
	_, ok := map[string]bool{"AC": true, "AL": true, "AP": true, "AM": true, "BA": true, "CE": true, "DF": true, "ES": true, "GO": true, "MA": true, "MT": true, "MS": true, "MG": true, "PA": true, "PB": true, "PR": true, "PE": true, "PI": true, "RJ": true, "RN": true, "RS": true, "RO": true, "RR": true, "SC": true, "SP": true, "SE": true, "TO": true}[strings.ToUpper(v)]
	return ok
}

func carLookupCodes(car string) []string {
	out := []string{car}
	plain := strings.ReplaceAll(car, ".", "")
	if plain != car {
		out = append(out, plain)
	}
	return out
}
func carLayerUF(uf string) string {
	if strings.EqualFold(uf, "DF") {
		return "DF"
	}
	return strings.ToLower(uf)
}

func lookupCARPublic(ctx context.Context, car, uf string) (*carGeoFeature, error) {
	versions := []struct{ version, typeKey string }{{"2.0.0", "typeNames"}, {"1.1.0", "typeName"}, {"1.0.0", "typeName"}}
	var lastErr error
	for _, code := range carLookupCodes(car) {
		for _, v := range versions {
			f, err := lookupCARPublicVersion(ctx, code, uf, v.version, v.typeKey)
			if err == nil && f != nil {
				return f, nil
			}
			if err != nil {
				lastErr = err
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

func lookupCARPublicVersion(ctx context.Context, car, uf, version, typeKey string) (*carGeoFeature, error) {
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", version)
	params.Set("request", "GetFeature")
	params.Set(typeKey, "sicar:sicar_imoveis_"+carLayerUF(uf))
	params.Set("outputFormat", "application/json")
	params.Set("srsName", "EPSG:4326")
	if version == "2.0.0" {
		params.Set("count", "2")
	} else {
		params.Set("maxFeatures", "2")
	}
	params.Set("CQL_FILTER", "cod_imovel='"+strings.ReplaceAll(car, "'", "''")+"'")

	var lastErr error
	for _, endpoint := range []string{carWFSURL, carWFSFallback} {
		body, err := fetchCARBody(ctx, endpoint+"?"+params.Encode())
		if err != nil {
			lastErr = err
			continue
		}
		var fc carGeoJSON
		if err := json.Unmarshal(body, &fc); err != nil {
			lastErr = err
			continue
		}
		if len(fc.Features) == 0 {
			return nil, nil
		}
		return &fc.Features[0], nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("consulta SICAR sem resposta")
}

// fetchCARBody tenta primeiro o cliente HTTP nativo. Alguns servidores do SICAR
// recusam esporadicamente o handshake TLS do Go/Windows embora funcionem no
// navegador. Nessa situação usamos o curl.exe do próprio Windows (Schannel)
// como fallback, sem desabilitar validação TLS.
func fetchCARBody(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 ViaVerdeCAR/1.0.1")
	resp, directErr := (&http.Client{Timeout: 18 * time.Second}).Do(req)
	if directErr == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		}
		directErr = fmt.Errorf("SICAR respondeu HTTP %d", resp.StatusCode)
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	}

	body, fallbackErr := fetchCARWithCurl(ctx, target)
	if fallbackErr == nil {
		return body, nil
	}
	return nil, fmt.Errorf("conexão HTTPS direta falhou (%v); fallback do Windows também falhou (%v)", directErr, fallbackErr)
}

func fetchCARWithCurl(ctx context.Context, target string) ([]byte, error) {
	bin, err := exec.LookPath("curl.exe")
	if err != nil {
		bin, err = exec.LookPath("curl")
	}
	if err != nil {
		return nil, errors.New("curl do sistema não encontrado")
	}
	cmd := exec.CommandContext(ctx, bin,
		"--location",
		"--silent",
		"--show-error",
		"--fail-with-body",
		"--connect-timeout", "10",
		"--max-time", "20",
		"--header", "Accept: application/json",
		"--user-agent", "Mozilla/5.0 ViaVerdeCAR/1.0.1",
		target,
	)
	body, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			msg := strings.TrimSpace(string(ee.Stderr))
			if len(msg) > 260 {
				msg = msg[len(msg)-260:]
			}
			if msg != "" {
				return nil, fmt.Errorf("curl: %s", msg)
			}
		}
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("curl retornou resposta vazia")
	}
	if len(body) > 8<<20 {
		return nil, errors.New("resposta SICAR excedeu 8 MB")
	}
	return body, nil
}

func carStatusLabel(v string) string {
	s := strings.ToUpper(strings.TrimSpace(v))
	switch s {
	case "AT":
		return "Ativo"
	case "PE":
		return "Pendente"
	case "SU":
		return "Suspenso"
	case "CA":
		return "Cancelado"
	case "RE":
		return "Retificado"
	}
	return strings.TrimSpace(v)
}
func carPropertyTypeLabel(v string) string {
	s := strings.ToUpper(strings.TrimSpace(v))
	switch s {
	case "IRU":
		return "Imóvel Rural"
	case "AST":
		return "Assentamento da Reforma Agrária"
	case "PCT":
		return "Povos e Comunidades Tradicionais"
	}
	return strings.TrimSpace(v)
}
func carStringProp(props map[string]any, keys ...string) string {
	for _, k := range keys {
		for pk, pv := range props {
			if !strings.EqualFold(pk, k) || pv == nil {
				continue
			}
			s := strings.TrimSpace(fmt.Sprint(pv))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}
func carFloatProp(props map[string]any, keys ...string) float64 {
	for _, k := range keys {
		for pk, pv := range props {
			if !strings.EqualFold(pk, k) || pv == nil {
				continue
			}
			switch x := pv.(type) {
			case float64:
				return x
			case json.Number:
				f, _ := x.Float64()
				return f
			default:
				s := strings.ReplaceAll(strings.TrimSpace(fmt.Sprint(x)), ",", ".")
				f, _ := strconv.ParseFloat(s, 64)
				if f != 0 {
					return f
				}
			}
		}
	}
	return 0
}
func carGeometryUsable(g carGeoJSONGeometry) bool {
	return len(g.Coordinates) > 0 && (strings.EqualFold(g.Type, "Polygon") || strings.EqualFold(g.Type, "MultiPolygon"))
}

func carGeometryPolygons(g carGeoJSONGeometry) ([][][][]float64, error) {
	switch strings.ToLower(g.Type) {
	case "polygon":
		var p [][][]float64
		if err := json.Unmarshal(g.Coordinates, &p); err != nil {
			return nil, err
		}
		return [][][][]float64{p}, nil
	case "multipolygon":
		var mp [][][][]float64
		if err := json.Unmarshal(g.Coordinates, &mp); err != nil {
			return nil, err
		}
		return mp, nil
	default:
		return nil, errors.New("geometria não suportada")
	}
}

func carGeometryKML(car string, g carGeoJSONGeometry) ([]byte, error) {
	polys, err := carGeometryPolygons(g)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<kml xmlns=\"http://www.opengis.net/kml/2.2\"><Document><name>")
	_ = xml.EscapeText(&b, []byte("CAR "+car))
	b.WriteString("</name><Style id=\"car\"><LineStyle><color>ff3d7f37</color><width>3</width></LineStyle><PolyStyle><color>3374b96a</color></PolyStyle></Style><Placemark><name>")
	_ = xml.EscapeText(&b, []byte(car))
	b.WriteString("</name><styleUrl>#car</styleUrl>")
	if len(polys) > 1 {
		b.WriteString("<MultiGeometry>")
	}
	for _, poly := range polys {
		if len(poly) == 0 {
			continue
		}
		b.WriteString("<Polygon><tessellate>1</tessellate>")
		for ri, ring := range poly {
			if len(ring) < 3 {
				continue
			}
			if ri == 0 {
				b.WriteString("<outerBoundaryIs>")
			} else {
				b.WriteString("<innerBoundaryIs>")
			}
			b.WriteString("<LinearRing><coordinates>")
			for _, c := range ring {
				if len(c) < 2 {
					continue
				}
				fmt.Fprintf(&b, "%.8f,%.8f,0 ", c[0], c[1])
			}
			b.WriteString("</coordinates></LinearRing>")
			if ri == 0 {
				b.WriteString("</outerBoundaryIs>")
			} else {
				b.WriteString("</innerBoundaryIs>")
			}
		}
		b.WriteString("</Polygon>")
	}
	if len(polys) > 1 {
		b.WriteString("</MultiGeometry>")
	}
	b.WriteString("</Placemark></Document></kml>")
	return b.Bytes(), nil
}

func carGeometryAreaHa(g carGeoJSONGeometry) float64 {
	polys, err := carGeometryPolygons(g)
	if err != nil {
		return 0
	}
	area := 0.0
	for _, poly := range polys {
		for ri, ring := range poly {
			a := math.Abs(carRingAreaM2(ring))
			if ri == 0 {
				area += a
			} else {
				area -= a
			}
		}
	}
	if area < 0 {
		area = 0
	}
	return area / 10000
}
func carRingAreaM2(ring [][]float64) float64 {
	if len(ring) < 3 {
		return 0
	}
	lat0 := 0.0
	n := 0.0
	for _, p := range ring {
		if len(p) >= 2 {
			lat0 += p[1]
			n++
		}
	}
	if n == 0 {
		return 0
	}
	lat0 /= n
	const r = 6371008.8
	cos0 := math.Cos(lat0 * math.Pi / 180)
	xys := make([][2]float64, 0, len(ring))
	for _, p := range ring {
		if len(p) < 2 {
			continue
		}
		xys = append(xys, [2]float64{r * p[0] * math.Pi / 180 * cos0, r * p[1] * math.Pi / 180})
	}
	s := 0.0
	for i := range xys {
		j := (i + 1) % len(xys)
		s += xys[i][0]*xys[j][1] - xys[j][0]*xys[i][1]
	}
	return s / 2
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
				if len(ring[i-1]) < 2 || len(ring[i]) < 2 {
					continue
				}
				total += carHaversineM(ring[i-1][1], ring[i-1][0], ring[i][1], ring[i][0])
			}
		}
	}
	return total
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
				minX = math.Min(minX, p[0])
				maxX = math.Max(maxX, p[0])
				minY = math.Min(minY, p[1])
				maxY = math.Max(maxY, p[1])
			}
		}
	}
	if math.IsInf(minX, 1) {
		return 0, 0
	}
	return (minY + maxY) / 2, (minX + maxX) / 2
}
func carHaversineM(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371008.8
	p1, p2 := lat1*math.Pi/180, lat2*math.Pi/180
	dp := (lat2 - lat1) * math.Pi / 180
	dl := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
func safeCARFilename(v string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9_-]+`)
	return strings.Trim(re.ReplaceAllString(strings.ReplaceAll(v, ".", "-"), "_"), "_")
}
