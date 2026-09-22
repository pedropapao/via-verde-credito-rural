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
	Code          string  `json:"code"`
	Label         string  `json:"label"`
	AreaHa        float64 `json:"area_ha"`
	FeatureCount  int     `json:"feature_count"`
	Available     bool    `json:"available"`
	SourceFile    string  `json:"source_file"`
	Method        string  `json:"method"`
	GeoJSON       string  `json:"geojson"`
	CacheStatus   string  `json:"cache_status"`
	CacheAgeHours float64 `json:"cache_age_hours"`
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

func (a *App) RetrySICARThemes(propertyID int64) (SICARThemesSummary, error) {
	var car CARResult
	var err error
	if propertyID > 0 {
		car, err = a.GetLatestCAR(propertyID)
	} else {
		var cache CARSessionCache
		cache, err = a.GetLastCARSession()
		if err == nil {
			car = cache.Result
		}
	}
	if err != nil || !car.Found || strings.TrimSpace(car.GeoJSON) == "" {
		return SICARThemesSummary{}, errors.New("consulte um CAR com geometria antes de tentar novamente os temas SICAR")
	}

	// A consulta inicial permanece curta para não travar o fluxo principal.
	// A tentativa manual pode esperar mais, pois os pacotes municipais do SICAR
	// — especialmente em MG — podem demorar significativamente.
	timeout := 4 * time.Minute
	if strings.EqualFold(strings.TrimSpace(car.UF), "MG") {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	themes := a.analyzeSICARThemes(ctx, car.CAR, car.UF, car.MunicipalityCode, car.GeoJSON, car.AreaHa)
	car.Themes = themes

	// Atualiza somente o resultado da sessão existente para não perder gleba
	// temporária, mapa ou outras informações da consulta avulsa.
	if cache, cacheErr := a.GetLastCARSession(); cacheErr == nil && strings.EqualFold(cache.Result.CAR, car.CAR) {
		_ = a.updateLastCARSessionResult(car)
	}
	return themes, nil
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
	zipPath, cacheStatus, cacheAge, err := a.ensureSICARThemeZipResilient(ctx, strings.ToUpper(uf), municipalityCode, theme)
	if err != nil {
		return metric, err
	}
	metric.SourceFile = zipPath
	metric.CacheStatus = cacheStatus
	metric.CacheAgeHours = cacheAge
	if cacheStatus == "cache_stale" {
		metric.Method += " (último pacote válido em cache)"
	}
	if cacheStatus == "manual" {
		metric.Method += " (pacote oficial importado manualmente)"
	}
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
	path, _, _, err := a.ensureSICARThemeZipResilient(ctx, uf, municipalityCode, theme)
	return path, err
}

func (a *App) ensureSICARThemeZipResilient(ctx context.Context, uf, municipalityCode, theme string) (string, string, float64, error) {
	cacheDir := filepath.Join(a.dataDir, "cache", "sicar", municipalityCode)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", "", 0, err
	}
	path := filepath.Join(cacheDir, strings.ToLower(theme)+".zip")
	manualMarker := path + ".manual"

	if st, err := os.Stat(path); err == nil && st.Size() > 1024 && time.Since(st.ModTime()) < 7*24*time.Hour {
		if err := validSICARThemeZip(path); err == nil {
			status := "cache_fresh"
			if _, err := os.Stat(manualMarker); err == nil {
				status = "manual"
			}
			return path, status, time.Since(st.ModTime()).Hours(), nil
		}
	}

	var downloadErr error
	remoteThemes := sicarThemeRemoteCandidates(theme)
	for _, remoteTheme := range remoteThemes {
		for attempt := 1; attempt <= 2; attempt++ {
			err := a.downloadSICARThemeZipAs(ctx, uf, municipalityCode, remoteTheme, path)
			if err == nil {
				st, _ := os.Stat(path)
				_ = os.Remove(manualMarker)
				age := 0.0
				if st != nil {
					age = time.Since(st.ModTime()).Hours()
				}
				return path, "online", age, nil
			}
			downloadErr = err
			if ctx.Err() != nil {
				break
			}
			if attempt < 2 {
				select {
				case <-ctx.Done():
				case <-time.After(1200 * time.Millisecond):
				}
			}
		}
		if ctx.Err() != nil {
			break
		}
	}

	// Se o serviço público estiver instável, o último pacote válido continua útil
	// como contingência. O status e a idade são devolvidos para a interface deixar
	// explícito que se trata de cache, não de uma consulta online atual.
	if st, err := os.Stat(path); err == nil && st.Size() > 1024 {
		if err := validSICARThemeZip(path); err == nil {
			status := "cache_stale"
			if _, err := os.Stat(manualMarker); err == nil {
				status = "manual"
			}
			return path, status, time.Since(st.ModTime()).Hours(), nil
		}
	}
	if downloadErr == nil {
		downloadErr = errors.New("pacote público do SICAR indisponível")
	}
	return "", "", 0, downloadErr
}

func sicarThemeRemoteCandidates(theme string) []string {
	switch strings.ToUpper(strings.TrimSpace(theme)) {
	case "APP":
		// As interfaces públicas do SICAR já expuseram as duas grafias.
		// Mantemos ambas para compatibilidade sem alterar o código canônico usado
		// pelo restante do Via Verde.
		return []string{"APP", "APPS"}
	default:
		return []string{strings.ToUpper(strings.TrimSpace(theme))}
	}
}

func (a *App) downloadSICARThemeZip(ctx context.Context, uf, municipalityCode, theme, path string) error {
	return a.downloadSICARThemeZipAs(ctx, uf, municipalityCode, theme, path)
}

func (a *App) downloadSICARThemeZipAs(ctx context.Context, uf, municipalityCode, remoteTheme, path string) error {
	params := url.Values{}
	params.Set("municipio", municipalityCode)
	params.Set("tema", remoteTheme)
	params.Set("servico", "SHP")
	target := sicarGeoServicesBase + "/" + url.PathEscape(uf) + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/zip, application/octet-stream, */*")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Referer", "https://consulta.car.gov.br/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 ViaVerdeCAR/"+AppVersion)
	// O contexto da consulta inicial continua limitando a espera a ~75 s.
	// Em uma tentativa manual o cliente pode aguardar mais pelo pacote municipal.
	resp, err := (&http.Client{Timeout: 210 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GeoServices respondeu HTTP %d", resp.StatusCode)
	}

	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, 120<<20))
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	if n < 512 {
		_ = os.Remove(tmp)
		return errors.New("pacote retornado está vazio")
	}
	if n >= 120<<20 {
		_ = os.Remove(tmp)
		return errors.New("pacote municipal excedeu 120 MB")
	}
	if err := validSICARThemeZip(tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}

func validSICARThemeZip(path string) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return errors.New("GeoServices não retornou um ZIP válido")
	}
	defer zr.Close()
	hasShp := false
	for _, zf := range zr.File {
		if strings.HasSuffix(strings.ToLower(zf.Name), ".shp") {
			hasShp = true
			break
		}
	}
	if !hasShp {
		return errors.New("pacote não contém shapefile")
	}
	return nil
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
