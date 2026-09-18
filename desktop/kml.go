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
	AreaDifferenceHa       float64 `json:"area_difference_ha"`
	AreaDifferencePct      float64 `json:"area_difference_pct"`
	CenterDistanceM        float64 `json:"center_distance_m"`
	PerimeterDifferencePct float64 `json:"perimeter_difference_pct"`
	IntersectionAreaHa     float64 `json:"intersection_area_ha"`
	KMLInsideCARPct        float64 `json:"kml_inside_car_pct"`
	CARInsideKMLPct        float64 `json:"car_inside_kml_pct"`
	OverlapMethod          string  `json:"overlap_method"`
	Level                  string  `json:"level"`
	Summary                string  `json:"summary"`
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
	if kml.PerimeterM > 0 && car.PerimeterM > 0 {
		out.PerimeterDifferencePct = math.Abs(kml.PerimeterM-car.PerimeterM) / car.PerimeterM * 100
	}

	if kml.GeoJSON != "" && car.GeoJSON != "" {
		if area, kpct, cpct, err := estimateGeometryOverlap(kml.GeoJSON, car.GeoJSON); err == nil {
			out.IntersectionAreaHa = area
			out.KMLInsideCARPct = kpct
			out.CARInsideKMLPct = cpct
			out.OverlapMethod = "estimativa espacial por malha de alta resolução"
		}
	}

	// A classificação é uma conferência geométrica auxiliar, não uma conclusão
	// ambiental ou cadastral oficial. Os limites são explícitos e conservadores.
	if out.OverlapMethod != "" {
		minOverlap := math.Min(out.KMLInsideCARPct, out.CARInsideKMLPct)
		switch {
		case minOverlap >= 95 && out.AreaDifferencePct <= 2 && out.CenterDistanceM <= 250:
			out.Level = "ok"
			out.Summary = "Geometrias muito compatíveis: forte sobreposição, área próxima e centros coerentes."
		case minOverlap >= 85 && out.AreaDifferencePct <= 5 && out.CenterDistanceM <= 750:
			out.Level = "warning"
			out.Summary = "Geometrias parcialmente compatíveis. Confira visualmente limites, retificações e origem do KML."
		default:
			out.Level = "error"
			out.Summary = "Divergência geométrica relevante entre KML e CAR. Confira se os arquivos representam o mesmo imóvel e a mesma versão cadastral."
		}
		return out
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

type planarPoint struct{ X, Y float64 }
type planarRing []planarPoint
type planarPolygon []planarRing
type planarMultiPolygon []planarPolygon

func estimateGeometryOverlap(kmlGeoJSON, carGeoJSON string) (intersectionHa, kmlInsidePct, carInsidePct float64, err error) {
	kmlPolys, err := geoJSONToPlanar(kmlGeoJSON, 0)
	if err != nil || len(kmlPolys) == 0 {
		return 0, 0, 0, errors.New("KML sem geometria comparável")
	}
	carPolys, err := geoJSONToPlanar(carGeoJSON, sharedLatitude(kmlGeoJSON, carGeoJSON))
	if err != nil || len(carPolys) == 0 {
		return 0, 0, 0, errors.New("CAR sem geometria comparável")
	}
	// Reprojeta também o KML com a mesma latitude de referência do CAR para que
	// a malha espacial use o mesmo sistema local em metros.
	lat0 := sharedLatitude(kmlGeoJSON, carGeoJSON)
	kmlPolys, err = geoJSONToPlanar(kmlGeoJSON, lat0)
	if err != nil {
		return 0, 0, 0, err
	}
	carPolys, err = geoJSONToPlanar(carGeoJSON, lat0)
	if err != nil {
		return 0, 0, 0, err
	}

	kx0, ky0, kx1, ky1, okK := planarBounds(kmlPolys)
	cx0, cy0, cx1, cy1, okC := planarBounds(carPolys)
	if !okK || !okC {
		return 0, 0, 0, errors.New("limites geométricos inválidos")
	}
	x0, y0 := math.Max(kx0, cx0), math.Max(ky0, cy0)
	x1, y1 := math.Min(kx1, cx1), math.Min(ky1, cy1)
	if x1 <= x0 || y1 <= y0 {
		return 0, 0, 0, nil
	}

	const grid = 320
	dx, dy := (x1-x0)/grid, (y1-y0)/grid
	insideBoth := 0
	for iy := 0; iy < grid; iy++ {
		y := y0 + (float64(iy)+0.5)*dy
		for ix := 0; ix < grid; ix++ {
			x := x0 + (float64(ix)+0.5)*dx
			p := planarPoint{X: x, Y: y}
			if pointInMultiPolygon(p, kmlPolys) && pointInMultiPolygon(p, carPolys) {
				insideBoth++
			}
		}
	}
	intersectionM2 := float64(insideBoth) * dx * dy
	intersectionHa = intersectionM2 / 10000
	kAreaM2 := planarMultiArea(kmlPolys)
	cAreaM2 := planarMultiArea(carPolys)
	if kAreaM2 > 0 {
		kmlInsidePct = math.Min(100, intersectionM2/kAreaM2*100)
	}
	if cAreaM2 > 0 {
		carInsidePct = math.Min(100, intersectionM2/cAreaM2*100)
	}
	return intersectionHa, kmlInsidePct, carInsidePct, nil
}

func sharedLatitude(raws ...string) float64 {
	total, count := 0.0, 0
	for _, raw := range raws {
		var f carGeoFeature
		if json.Unmarshal([]byte(raw), &f) != nil {
			continue
		}
		polys, err := carGeometryPolygons(f.Geometry)
		if err != nil {
			continue
		}
		for _, poly := range polys {
			for _, ring := range poly {
				for _, p := range ring {
					if len(p) >= 2 {
						total += p[1]
						count++
					}
				}
			}
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func geoJSONToPlanar(raw string, lat0 float64) (planarMultiPolygon, error) {
	var f carGeoFeature
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return nil, err
	}
	polys, err := carGeometryPolygons(f.Geometry)
	if err != nil {
		return nil, err
	}
	if lat0 == 0 {
		lat0 = sharedLatitude(raw)
	}
	const earthR = 6371008.8
	cos0 := math.Cos(lat0 * math.Pi / 180)
	out := make(planarMultiPolygon, 0, len(polys))
	for _, poly := range polys {
		pp := make(planarPolygon, 0, len(poly))
		for _, ring := range poly {
			pr := make(planarRing, 0, len(ring))
			for _, p := range ring {
				if len(p) < 2 {
					continue
				}
				pr = append(pr, planarPoint{
					X: earthR * p[0] * math.Pi / 180 * cos0,
					Y: earthR * p[1] * math.Pi / 180,
				})
			}
			if len(pr) >= 3 {
				pp = append(pp, pr)
			}
		}
		if len(pp) > 0 {
			out = append(out, pp)
		}
	}
	return out, nil
}

func planarBounds(mp planarMultiPolygon) (minX, minY, maxX, maxY float64, ok bool) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, poly := range mp {
		for _, ring := range poly {
			for _, p := range ring {
				minX, minY = math.Min(minX, p.X), math.Min(minY, p.Y)
				maxX, maxY = math.Max(maxX, p.X), math.Max(maxY, p.Y)
				ok = true
			}
		}
	}
	return
}

func planarRingArea(r planarRing) float64 {
	if len(r) < 3 {
		return 0
	}
	s := 0.0
	for i := range r {
		j := (i + 1) % len(r)
		s += r[i].X*r[j].Y - r[j].X*r[i].Y
	}
	return math.Abs(s / 2)
}

func planarMultiArea(mp planarMultiPolygon) float64 {
	total := 0.0
	for _, poly := range mp {
		for i, ring := range poly {
			a := planarRingArea(ring)
			if i == 0 {
				total += a
			} else {
				total -= a
			}
		}
	}
	if total < 0 {
		return 0
	}
	return total
}

func pointInMultiPolygon(p planarPoint, mp planarMultiPolygon) bool {
	for _, poly := range mp {
		if len(poly) == 0 || !pointInRing(p, poly[0]) {
			continue
		}
		inHole := false
		for _, hole := range poly[1:] {
			if pointInRing(p, hole) {
				inHole = true
				break
			}
		}
		if !inHole {
			return true
		}
	}
	return false
}

func pointInRing(p planarPoint, ring planarRing) bool {
	inside := false
	j := len(ring) - 1
	for i := 0; i < len(ring); i++ {
		pi, pj := ring[i], ring[j]
		crosses := (pi.Y > p.Y) != (pj.Y > p.Y)
		if crosses {
			x := (pj.X-pi.X)*(p.Y-pi.Y)/(pj.Y-pi.Y) + pi.X
			if p.X < x {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}
