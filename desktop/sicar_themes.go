package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const sicarGeoServicesBase = "https://consulta.car.gov.br/geoservices/estadosCSV"

type SICARThemeMetric struct {
	Code         string  `json:"code"`
	Label        string  `json:"label"`
	AreaHa       float64 `json:"area_ha"`
	FeatureCount int     `json:"feature_count"`
	Available    bool    `json:"available"`
	SourceFile   string  `json:"source_file"`
	Method       string  `json:"method"`
	GeoJSON      string  `json:"geojson"`
}

type SICARThemesSummary struct {
	CheckedAt  string                      `json:"checked_at"`
	SourceURL  string                      `json:"source_url"`
	Municipio  string                      `json:"municipio"`
	UF         string                      `json:"uf"`
	Themes     map[string]SICARThemeMetric `json:"themes"`
	Warnings   []string                    `json:"warnings"`
	Complete   bool                        `json:"complete"`
}

var sicarThemeDefinitions = []struct {
	Code  string
	Label string
}{
	{"APP", "Área de Preservação Permanente"},
	{"RESERVA_LEGAL", "Reserva Legal"},
	{"VEGETACAO_NATIVA", "Remanescente de Vegetação Nativa"},
	{"AREA_CONSOLIDADA", "Área Consolidada"},
	{"USO_RESTRITO", "Área de Uso Restrito"},
	{"SERVIDAO_ADMINISTRATIVA", "Servidão Administrativa"},
}

func (a *App) analyzeSICARThemes(ctx context.Context, car, uf, municipalityCode, carGeoJSON string, carAreaHa float64) SICARThemesSummary {
	out := SICARThemesSummary{
		CheckedAt: time.Now().Format(time.RFC3339),
		SourceURL: "https://consulta.car.gov.br/geoservices",
		Municipio: municipalityCode,
		UF: strings.ToUpper(strings.TrimSpace(uf)),
		Themes: map[string]SICARThemeMetric{},
	}
	if strings.TrimSpace(carGeoJSON) == "" || municipalityCode == "" || uf == "" {
		out.Warnings = append(out.Warnings, "Geometria, UF ou código municipal insuficiente para consultar temas detalhados do SICAR.")
		return out
	}

	type result struct {
		metric SICARThemeMetric
		err    error
	}
	ch := make(chan result, len(sicarThemeDefinitions))
	var wg sync.WaitGroup
	// Limita downloads simultâneos para não sobrecarregar o GeoServices público.
	sem := make(chan struct{}, 2)
	for _, def := range sicarThemeDefinitions {
		def := def
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				ch <- result{metric: SICARThemeMetric{Code: def.Code, Label: def.Label}, err: ctx.Err()}
				return
			}
			m, err := a.analyzeSingleSICARTheme(ctx, car, uf, municipalityCode, carGeoJSON, carAreaHa, def.Code, def.Label)
			ch <- result{metric: m, err: err}
		}()
	}
	wg.Wait()
	close(ch)

	success := 0
	for r := range ch {
		if r.err != nil {
			out.Warnings = append(out.Warnings, r.metric.Label+": "+r.err.Error())
			out.Themes[r.metric.Code] = r.metric
			continue
		}
		success++
		out.Themes[r.metric.Code] = r.metric
	}
	out.Complete = success == len(sicarThemeDefinitions)
	return out
}

func (a *App) analyzeSingleSICARTheme(ctx context.Context, car, uf, municipalityCode, carGeoJSON string, carAreaHa float64, theme, label string) (SICARThemeMetric, error) {
	metric := SICARThemeMetric{Code: theme, Label: label, Method: "interseção espacial aproximada com pacote municipal público do SICAR"}
	zipPath, err := a.ensureSICARThemeZip(ctx, strings.ToUpper(uf), municipalityCode, theme)
	if err != nil {
		return metric, err
	}
	metric.SourceFile = zipPath
	area, count, geojson, err := intersectThemeZipWithCAR(zipPath, carGeoJSON, carAreaHa)
	if err != nil {
		return metric, err
	}
	metric.AreaHa = area
	metric.FeatureCount = count
	metric.GeoJSON = geojson
	metric.Available = true
	return metric, nil
}

func (a *App) ensureSICARThemeZip(ctx context.Context, uf, municipalityCode, theme string) (string, error) {
	cacheDir := filepath.Join(a.dataDir, "cache", "sicar", municipalityCode)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(cacheDir, strings.ToLower(theme)+".zip")
	if st, err := os.Stat(path); err == nil && st.Size() > 1024 && time.Since(st.ModTime()) < 7*24*time.Hour {
		return path, nil
	}

	params := url.Values{}
	params.Set("municipio", municipalityCode)
	params.Set("tema", theme)
	params.Set("servico", "SHP")
	target := sicarGeoServicesBase + "/" + url.PathEscape(uf) + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/zip, application/octet-stream, */*")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	resp, err := (&http.Client{Timeout: 75 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GeoServices respondeu HTTP %d", resp.StatusCode)
	}
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, 120<<20))
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		if copyErr != nil {
			return "", copyErr
		}
		return "", closeErr
	}
	if n < 512 {
		_ = os.Remove(tmp)
		return "", errors.New("pacote retornado está vazio")
	}
	if n >= 120<<20 {
		_ = os.Remove(tmp)
		return "", errors.New("pacote municipal excedeu 120 MB")
	}
	zr, err := zip.OpenReader(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return "", errors.New("GeoServices não retornou um ZIP válido")
	}
	hasShp := false
	for _, zf := range zr.File {
		if strings.HasSuffix(strings.ToLower(zf.Name), ".shp") {
			hasShp = true
			break
		}
	}
	_ = zr.Close()
	if !hasShp {
		_ = os.Remove(tmp)
		return "", errors.New("pacote não contém shapefile")
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

func intersectThemeZipWithCAR(zipPath, carGeoJSON string, carAreaHa float64) (float64, int, string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, 0, "", err
	}
	defer zr.Close()

	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carGeoJSON)
	if !ok {
		return 0, 0, "", errors.New("limites do CAR inválidos")
	}

	total := 0.0
	count := 0
	foundShape := false
	combined := make([][][][]float64, 0)
	for _, zf := range zr.File {
		if !strings.HasSuffix(strings.ToLower(zf.Name), ".shp") {
			continue
		}
		foundShape = true
		r, err := zf.Open()
		if err != nil {
			continue
		}
		err = scanShapefilePolygons(r, func(bbox [4]float64, geom carGeoJSONGeometry) bool {
			if bbox[2] < minLon || bbox[0] > maxLon || bbox[3] < minLat || bbox[1] > maxLat {
				return true
			}
			raw, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: map[string]any{}, Geometry: geom})
			intersection, _, _, err := estimateGeometryOverlap(carGeoJSON, string(raw))
			if err == nil && intersection > 0.0001 {
				total += intersection
				count++
				if len(combined) < 300 {
					var mp [][][][]float64
					if json.Unmarshal(geom.Coordinates, &mp) == nil {
						combined = append(combined, mp...)
					}
				}
			}
			return true
		})
		_ = r.Close()
		if err != nil {
			return 0, count, "", err
		}
	}
	if !foundShape {
		return 0, 0, "", errors.New("nenhum .shp encontrado")
	}
	if carAreaHa > 0 && total > carAreaHa {
		// Evita apresentar mais de 100% do imóvel em casos de feições municipais
		// sobrepostas. O relatório mantém o método como estimativa espacial.
		total = carAreaHa
	}
	geojson := ""
	if len(combined) > 0 {
		coords, _ := json.Marshal(combined)
		feature := carGeoFeature{
			Type:       "Feature",
			Properties: map[string]any{"source": "SICAR GeoServices"},
			Geometry:   carGeoJSONGeometry{Type: "MultiPolygon", Coordinates: coords},
		}
		b, _ := json.Marshal(feature)
		geojson = string(b)
	}
	return total, count, geojson, nil
}

func scanShapefilePolygons(r io.Reader, visit func(bbox [4]float64, geom carGeoJSONGeometry) bool) error {
	header := make([]byte, 100)
	if _, err := io.ReadFull(r, header); err != nil {
		return err
	}
	for {
		recHeader := make([]byte, 8)
		if _, err := io.ReadFull(r, recHeader); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}
		contentWords := binary.BigEndian.Uint32(recHeader[4:8])
		contentBytes := int(contentWords) * 2
		if contentBytes < 4 || contentBytes > 64<<20 {
			return errors.New("registro shapefile inválido")
		}
		payload := make([]byte, contentBytes)
		if _, err := io.ReadFull(r, payload); err != nil {
			return err
		}
		shapeType := int32(binary.LittleEndian.Uint32(payload[:4]))
		if shapeType == 0 {
			continue
		}
		if shapeType != 5 && shapeType != 15 && shapeType != 25 {
			continue
		}
		bbox, geom, err := parseShapefilePolygonPayload(payload)
		if err != nil {
			continue
		}
		if !visit(bbox, geom) {
			return nil
		}
	}
}

func parseShapefilePolygonPayload(payload []byte) ([4]float64, carGeoJSONGeometry, error) {
	var bbox [4]float64
	if len(payload) < 48 {
		return bbox, carGeoJSONGeometry{}, errors.New("polígono shapefile curto")
	}
	rd := bytes.NewReader(payload[4:])
	for i := range bbox {
		if err := binary.Read(rd, binary.LittleEndian, &bbox[i]); err != nil {
			return bbox, carGeoJSONGeometry{}, err
		}
	}
	var numParts, numPoints int32
	if err := binary.Read(rd, binary.LittleEndian, &numParts); err != nil {
		return bbox, carGeoJSONGeometry{}, err
	}
	if err := binary.Read(rd, binary.LittleEndian, &numPoints); err != nil {
		return bbox, carGeoJSONGeometry{}, err
	}
	if numParts <= 0 || numPoints < 3 || numParts > 20000 || numPoints > 2_000_000 {
		return bbox, carGeoJSONGeometry{}, errors.New("contagem de pontos inválida")
	}
	parts := make([]int32, numParts)
	if err := binary.Read(rd, binary.LittleEndian, &parts); err != nil {
		return bbox, carGeoJSONGeometry{}, err
	}
	points := make([][2]float64, numPoints)
	for i := range points {
		if err := binary.Read(rd, binary.LittleEndian, &points[i][0]); err != nil {
			return bbox, carGeoJSONGeometry{}, err
		}
		if err := binary.Read(rd, binary.LittleEndian, &points[i][1]); err != nil {
			return bbox, carGeoJSONGeometry{}, err
		}
	}
	// Cada parte é tratada como polígono independente para a triagem aproximada.
	// A origem municipal continua preservada no ZIP em cache para auditoria.
	multi := make([][][][]float64, 0, numParts)
	for i := 0; i < int(numParts); i++ {
		start := int(parts[i])
		end := len(points)
		if i+1 < int(numParts) {
			end = int(parts[i+1])
		}
		if start < 0 || start >= end || end > len(points) || end-start < 3 {
			continue
		}
		ring := make([][]float64, 0, end-start+1)
		step := 1
		if end-start > 250 {
			step = int(math.Ceil(float64(end-start) / 250.0))
		}
		for p := start; p < end; p += step {
			ring = append(ring, []float64{points[p][0], points[p][1]})
		}
		if len(ring) >= 3 {
			first := ring[0]
			last := ring[len(ring)-1]
			if first[0] != last[0] || first[1] != last[1] {
				ring = append(ring, []float64{first[0], first[1]})
			}
			multi = append(multi, [][][]float64{ring})
		}
	}
	if len(multi) == 0 {
		return bbox, carGeoJSONGeometry{}, errors.New("polígono sem anéis válidos")
	}
	raw, _ := json.Marshal(multi)
	return bbox, carGeoJSONGeometry{Type: "MultiPolygon", Coordinates: raw}, nil
}
