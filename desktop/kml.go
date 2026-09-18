package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type KMLResult struct {
	Path       string     `json:"path"`
	Name       string     `json:"name"`
	AreaHa     float64    `json:"area_ha"`
	PerimeterM float64    `json:"perimeter_m"`
	CenterLat  float64    `json:"center_lat"`
	CenterLon  float64    `json:"center_lon"`
	Points     int        `json:"points"`
	GeoJSON    string     `json:"geojson"`
	Warnings   []string   `json:"warnings"`
	Sample     []GeoPoint `json:"sample"`
}

type GeometryComparison struct {
	AreaDifferenceHa  float64 `json:"area_difference_ha"`
	AreaDifferencePct float64 `json:"area_difference_pct"`
	CenterDistanceM   float64 `json:"center_distance_m"`
	Level             string  `json:"level"`
	Summary           string  `json:"summary"`
}

type parsedKML struct {
	Rings [][]GeoPoint
}

func (a *App) AttachKML(propertyID int64) (KMLResult, error) {
	if propertyID <= 0 {
		return KMLResult{}, errors.New("salve o imóvel antes de anexar o KML")
	}
	if a.ctx == nil {
		return KMLResult{}, errors.New("aplicativo ainda não inicializado")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Selecionar KML do imóvel",
		Filters: []runtime.FileFilter{{DisplayName: "Google Earth KML", Pattern: "*.kml"}},
	})
	if err != nil {
		return KMLResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return KMLResult{}, errors.New("importação cancelada")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return KMLResult{}, err
	}
	if len(data) > 25<<20 {
		return KMLResult{}, errors.New("KML maior que 25 MB")
	}
	result, err := analyzeKML(data)
	if err != nil {
		return KMLResult{}, err
	}
	dir := filepath.Join(a.dataDir, "properties", fmt.Sprint(propertyID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return KMLResult{}, err
	}
	dest := filepath.Join(dir, "imovel.kml")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return KMLResult{}, err
	}
	result.Path = dest
	result.Name = filepath.Base(path)
	if a.db != nil {
		_, _ = a.db.Exec(`UPDATE properties SET kml_path=?,updated_at=? WHERE id=?`, dest, time.Now().Format(time.RFC3339), propertyID)
	}
	return result, nil
}

func (a *App) LoadPropertyKML(propertyID int64) (KMLResult, error) {
	p, err := a.GetProperty(propertyID)
	if err != nil {
		return KMLResult{}, err
	}
	if strings.TrimSpace(p.KMLPath) == "" {
		return KMLResult{}, errors.New("este imóvel ainda não possui KML")
	}
	data, err := os.ReadFile(p.KMLPath)
	if err != nil {
		return KMLResult{}, err
	}
	r, err := analyzeKML(data)
	if err != nil {
		return KMLResult{}, err
	}
	r.Path = p.KMLPath
	r.Name = filepath.Base(p.KMLPath)
	return r, nil
}

func analyzeKML(data []byte) (KMLResult, error) {
	parsed, err := parseKML(data)
	if err != nil {
		return KMLResult{}, err
	}
	if len(parsed.Rings) == 0 {
		return KMLResult{}, errors.New("nenhum polígono válido encontrado no KML")
	}
	best := parsed.Rings[0]
	bestArea := math.Abs(projectedArea(best))
	for _, ring := range parsed.Rings[1:] {
		a := math.Abs(projectedArea(ring))
		if a > bestArea {
			best, bestArea = ring, a
		}
	}
	centerLat, centerLon := ringCenter(best)
	geo := ringGeoJSON(best)
	res := KMLResult{AreaHa: bestArea / 10000, PerimeterM: projectedPerimeter(best), CenterLat: centerLat, CenterLon: centerLon, Points: len(best), GeoJSON: geo}
	if len(parsed.Rings) > 1 {
		res.Warnings = append(res.Warnings, fmt.Sprintf("O arquivo contém %d polígonos. A análise principal usa o maior perímetro.", len(parsed.Rings)))
	}
	if len(best) > 5000 {
		res.Warnings = append(res.Warnings, "O KML possui muitos vértices; o mapa pode levar alguns segundos para desenhar.")
	}
	limit := len(best)
	if limit > 12 {
		limit = 12
	}
	res.Sample = append(res.Sample, best[:limit]...)
	return res, nil
}

func parseKML(data []byte) (parsedKML, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out parsedKML
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "coordinates" {
			continue
		}
		var raw string
		if err := dec.DecodeElement(&raw, &se); err != nil {
			continue
		}
		pts := parseCoordinateBlock(raw)
		if len(pts) >= 3 {
			out.Rings = append(out.Rings, pts)
		}
	}
	return out, nil
}

func parseCoordinateBlock(raw string) []GeoPoint {
	fields := strings.Fields(strings.TrimSpace(raw))
	out := make([]GeoPoint, 0, len(fields))
	for _, f := range fields {
		parts := strings.Split(f, ",")
		if len(parts) < 2 {
			continue
		}
		lon, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lat, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if e1 == nil && e2 == nil && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 {
			out = append(out, GeoPoint{Lat: lat, Lon: lon})
		}
	}
	if len(out) > 1 && almostEqual(out[0].Lat, out[len(out)-1].Lat) && almostEqual(out[0].Lon, out[len(out)-1].Lon) {
		out = out[:len(out)-1]
	}
	return out
}
func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-10 }

func xyMeters(pts []GeoPoint) [][2]float64 {
	if len(pts) == 0 {
		return nil
	}
	lat0 := 0.0
	for _, p := range pts {
		lat0 += p.Lat
	}
	lat0 /= float64(len(pts))
	const r = 6371008.8
	cos0 := math.Cos(lat0 * math.Pi / 180)
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		out[i][0] = r * p.Lon * math.Pi / 180 * cos0
		out[i][1] = r * p.Lat * math.Pi / 180
	}
	return out
}
func projectedArea(pts []GeoPoint) float64 {
	xy := xyMeters(pts)
	if len(xy) < 3 {
		return 0
	}
	s := 0.0
	for i := range xy {
		j := (i + 1) % len(xy)
		s += xy[i][0]*xy[j][1] - xy[j][0]*xy[i][1]
	}
	return s / 2
}
func projectedPerimeter(pts []GeoPoint) float64 {
	xy := xyMeters(pts)
	if len(xy) < 2 {
		return 0
	}
	t := 0.0
	for i := range xy {
		j := (i + 1) % len(xy)
		t += math.Hypot(xy[j][0]-xy[i][0], xy[j][1]-xy[i][1])
	}
	return t
}
func ringCenter(pts []GeoPoint) (float64, float64) {
	if len(pts) == 0 {
		return 0, 0
	}
	minLat, maxLat := pts[0].Lat, pts[0].Lat
	minLon, maxLon := pts[0].Lon, pts[0].Lon
	for _, p := range pts[1:] {
		minLat = math.Min(minLat, p.Lat)
		maxLat = math.Max(maxLat, p.Lat)
		minLon = math.Min(minLon, p.Lon)
		maxLon = math.Max(maxLon, p.Lon)
	}
	return (minLat + maxLat) / 2, (minLon + maxLon) / 2
}

func ringGeoJSON(pts []GeoPoint) string {
	coords := make([][]float64, 0, len(pts)+1)
	for _, p := range pts {
		coords = append(coords, []float64{p.Lon, p.Lat})
	}
	if len(pts) > 0 {
		coords = append(coords, []float64{pts[0].Lon, pts[0].Lat})
	}
	feature := map[string]any{"type": "Feature", "properties": map[string]any{"source": "KML local"}, "geometry": map[string]any{"type": "Polygon", "coordinates": [][][]float64{coords}}}
	b, _ := json.Marshal(feature)
	return string(b)
}

func (a *App) CompareKMLWithCAR(kml KMLResult, car CARResult) GeometryComparison {
	out := GeometryComparison{Level: "info", Summary: "Comparação disponível após carregar CAR e KML."}
	if kml.AreaHa <= 0 || car.AreaHa <= 0 {
		return out
	}
	out.AreaDifferenceHa = math.Abs(kml.AreaHa - car.AreaHa)
	out.AreaDifferencePct = out.AreaDifferenceHa / car.AreaHa * 100
	if kml.CenterLat != 0 && car.CenterLat != 0 {
		out.CenterDistanceM = carHaversineM(kml.CenterLat, kml.CenterLon, car.CenterLat, car.CenterLon)
	}
	switch {
	case out.AreaDifferencePct <= 1 && out.CenterDistanceM <= 150:
		out.Level = "ok"
		out.Summary = "KML e CAR apresentam áreas e centros muito próximos."
	case out.AreaDifferencePct <= 3 && out.CenterDistanceM <= 500:
		out.Level = "warning"
		out.Summary = "Há pequena diferença entre KML e CAR; vale conferir os limites no mapa."
	default:
		out.Level = "error"
		out.Summary = "KML e CAR apresentam diferença relevante. Confira se os arquivos pertencem ao mesmo imóvel e se houve retificação."
	}
	return out
}
