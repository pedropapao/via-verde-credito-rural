package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
)

type geoPoint struct{ Lat, Lon float64 }

type kmlStats struct {
	Points []geoPoint
	AreaHa float64
	PerimeterM float64
	SVG string
}

func parseKML360(data []byte) kmlStats {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var polygons [][]geoPoint
	for {
		tok, err := dec.Token()
		if err != nil { break }
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "coordinates" { continue }
		var raw string
		if err := dec.DecodeElement(&raw, &se); err != nil { continue }
		pts := parseCoordinateBlock(raw)
		if len(pts) >= 3 { polygons = append(polygons, pts) }
	}
	if len(polygons) == 0 { return kmlStats{} }
	best := polygons[0]
	bestArea := math.Abs(projectedArea(best))
	for _, p := range polygons[1:] {
		a := math.Abs(projectedArea(p))
		if a > bestArea { best, bestArea = p, a }
	}
	areaM2 := math.Abs(projectedArea(best))
	return kmlStats{Points: best, AreaHa: areaM2 / 10000.0, PerimeterM: projectedPerimeter(best), SVG: buildMapSVG(best)}
}

func parseCoordinateBlock(raw string) []geoPoint {
	fields := strings.Fields(strings.TrimSpace(raw))
	out := make([]geoPoint, 0, len(fields))
	for _, f := range fields {
		parts := strings.Split(f, ",")
		if len(parts) < 2 { continue }
		lon, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lat, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if e1 == nil && e2 == nil { out = append(out, geoPoint{Lat: lat, Lon: lon}) }
	}
	if len(out) > 1 && out[0] == out[len(out)-1] { out = out[:len(out)-1] }
	return out
}

func xyMeters(pts []geoPoint) [][2]float64 {
	if len(pts) == 0 { return nil }
	lat0 := 0.0
	for _, p := range pts { lat0 += p.Lat }
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

func projectedArea(pts []geoPoint) float64 {
	xy := xyMeters(pts)
	if len(xy) < 3 { return 0 }
	s := 0.0
	for i := range xy {
		j := (i + 1) % len(xy)
		s += xy[i][0]*xy[j][1] - xy[j][0]*xy[i][1]
	}
	return s / 2
}

func projectedPerimeter(pts []geoPoint) float64 {
	xy := xyMeters(pts)
	if len(xy) < 2 { return 0 }
	t := 0.0
	for i := range xy {
		j := (i + 1) % len(xy)
		t += math.Hypot(xy[j][0]-xy[i][0], xy[j][1]-xy[i][1])
	}
	return t
}

func buildMapSVG(pts []geoPoint) string {
	if len(pts) < 3 { return "" }
	xy := xyMeters(pts)
	minX, maxX := xy[0][0], xy[0][0]
	minY, maxY := xy[0][1], xy[0][1]
	for _, p := range xy {
		minX, maxX = math.Min(minX,p[0]), math.Max(maxX,p[0])
		minY, maxY = math.Min(minY,p[1]), math.Max(maxY,p[1])
	}
	w, h, pad := 720.0, 360.0, 24.0
	dx, dy := maxX-minX, maxY-minY
	if dx == 0 { dx = 1 }; if dy == 0 { dy = 1 }
	scale := math.Min((w-2*pad)/dx, (h-2*pad)/dy)
	var b strings.Builder
	for i, p := range xy {
		x := pad + (p[0]-minX)*scale
		y := h - pad - (p[1]-minY)*scale
		if i > 0 { b.WriteByte(' ') }
		fmt.Fprintf(&b, "%.1f,%.1f", x, y)
	}
	label := html.EscapeString(fmt.Sprintf("%d pontos", len(pts)))
	return fmt.Sprintf(`<svg class="kml-map" viewBox="0 0 720 360" role="img" aria-label="Mapa do perímetro"><rect width="720" height="360" rx="18" fill="#f3f8f4"/><g stroke="#d7e7dc" stroke-width="1"><path d="M0 90H720M0 180H720M0 270H720M180 0V360M360 0V360M540 0V360"/></g><polygon points="%s" fill="rgba(37,135,81,.18)" stroke="#19774a" stroke-width="4" stroke-linejoin="round"/><circle cx="24" cy="24" r="5" fill="#55b949"/><text x="40" y="30" font-size="14" fill="#365a49">%s</text></svg>`, b.String(), label)
}
