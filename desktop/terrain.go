package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var terrainElevationEndpoint = "https://api.open-meteo.com/v1/elevation"

type TerrainMetric struct {
	Available          bool    `json:"available"`
	ElevationMinM      float64 `json:"elevation_min_m"`
	ElevationMaxM      float64 `json:"elevation_max_m"`
	ElevationMeanM     float64 `json:"elevation_mean_m"`
	ReliefM            float64 `json:"relief_m"`
	MeanSlopePct       float64 `json:"mean_slope_pct"`
	MaxSlopePct        float64 `json:"max_slope_pct"`
	OperationalScore  float64 `json:"operational_score"`
	SampleCount        int     `json:"sample_count"`
	ResolutionM        int     `json:"resolution_m"`
	Confidence         string  `json:"confidence"`
	Source             string  `json:"source"`
	Warning            string  `json:"warning"`
}

type terrainElevationResponse struct {
	Elevation []*float64 `json:"elevation"`
}

type indexedTerrainPoint struct {
	Candidate int
	Point     GeoPoint
}

func enrichAreaAlternativesTerrain(candidates []AreaAlternative) (string, error) {
	if len(candidates) == 0 {
		return "", nil
	}
	var indexed []indexedTerrainPoint
	for i := range candidates {
		pts := terrainSamplePoints(candidates[i].GeoJSON, 12)
		for _, p := range pts {
			if len(indexed) >= 100 {
				break
			}
			indexed = append(indexed, indexedTerrainPoint{Candidate: i, Point: p})
		}
	}
	if len(indexed) == 0 {
		return "", errors.New("não foi possível amostrar pontos internos para relevo")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	elev, err := fetchTerrainElevations(ctx, indexed)
	if err != nil {
		return "", err
	}

	pointsByCandidate := make([][]GeoPoint, len(candidates))
	elevByCandidate := make([][]float64, len(candidates))
	for i, item := range indexed {
		if i >= len(elev) || math.IsNaN(elev[i]) {
			continue
		}
		pointsByCandidate[item.Candidate] = append(pointsByCandidate[item.Candidate], item.Point)
		elevByCandidate[item.Candidate] = append(elevByCandidate[item.Candidate], elev[i])
	}
	for i := range candidates {
		candidates[i].Terrain = calculateTerrainMetric(pointsByCandidate[i], elevByCandidate[i], candidates[i].AreaHa)
	}
	return "Copernicus DEM GLO-90 (90 m) via Open-Meteo Elevation API", nil
}

func terrainSamplePoints(raw string, maxPoints int) []GeoPoint {
	if maxPoints < 4 {
		maxPoints = 4
	}
	lat0 := sharedLatitude(raw)
	mp, err := geoJSONToPlanar(raw, lat0)
	if err != nil || len(mp) == 0 {
		return nil
	}
	minX, minY, maxX, maxY, ok := planarBounds(mp)
	if !ok {
		return nil
	}
	const earthR = 6371008.8
	cos0 := math.Cos(lat0 * math.Pi / 180)
	toGeo := func(p planarPoint) GeoPoint {
		return GeoPoint{Lat: p.Y / earthR * 180 / math.Pi, Lon: p.X / (earthR * cos0) * 180 / math.Pi}
	}
	var out []GeoPoint
	// Grade regular: ajuda a estimar relevo e inclinação sem depender dos vértices.
	const grid = 6
	for iy := 0; iy < grid && len(out) < maxPoints; iy++ {
		fy := (float64(iy) + 0.5) / grid
		for ix := 0; ix < grid && len(out) < maxPoints; ix++ {
			fx := (float64(ix) + 0.5) / grid
			p := planarPoint{X: minX + (maxX-minX)*fx, Y: minY + (maxY-minY)*fy}
			if pointInMultiPolygon(p, mp) {
				out = append(out, toGeo(p))
			}
		}
	}
	if len(out) < 4 {
		// Fallback para o centro aproximado.
		p := planarPoint{X: (minX + maxX) / 2, Y: (minY + maxY) / 2}
		if pointInMultiPolygon(p, mp) {
			out = append(out, toGeo(p))
		}
	}
	return out
}

func fetchTerrainElevations(ctx context.Context, indexed []indexedTerrainPoint) ([]float64, error) {
	if len(indexed) == 0 {
		return nil, errors.New("nenhum ponto para elevação")
	}
	if len(indexed) > 100 {
		indexed = indexed[:100]
	}
	lat := make([]string, 0, len(indexed))
	lon := make([]string, 0, len(indexed))
	for _, p := range indexed {
		lat = append(lat, strconv.FormatFloat(p.Point.Lat, 'f', 7, 64))
		lon = append(lon, strconv.FormatFloat(p.Point.Lon, 'f', 7, 64))
	}
	q := url.Values{}
	q.Set("latitude", strings.Join(lat, ","))
	q.Set("longitude", strings.Join(lon, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, terrainElevationEndpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 18 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("serviço de elevação respondeu HTTP %d", resp.StatusCode)
	}
	var data terrainElevationResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Elevation) != len(indexed) {
		return nil, errors.New("serviço de elevação retornou quantidade inesperada de pontos")
	}
	out := make([]float64, len(data.Elevation))
	for i, v := range data.Elevation {
		if v == nil {
			out[i] = math.NaN()
			continue
		}
		out[i] = *v
	}
	return out, nil
}

func calculateTerrainMetric(points []GeoPoint, elev []float64, areaHa float64) TerrainMetric {
	out := TerrainMetric{
		ResolutionM: 90,
		Source:      "Copernicus DEM GLO-90 via Open-Meteo",
	}
	if len(points) == 0 || len(points) != len(elev) {
		out.Warning = "amostragem de relevo insuficiente"
		return out
	}
	minE, maxE, sum := math.Inf(1), math.Inf(-1), 0.0
	valid := 0
	for _, z := range elev {
		if math.IsNaN(z) {
			continue
		}
		minE, maxE = math.Min(minE, z), math.Max(maxE, z)
		sum += z
		valid++
	}
	if valid < 2 {
		out.Warning = "elevação indisponível em pontos suficientes"
		return out
	}
	out.Available = true
	out.SampleCount = valid
	out.ElevationMinM = minE
	out.ElevationMaxM = maxE
	out.ElevationMeanM = sum / float64(valid)
	out.ReliefM = maxE - minE

	var slopes []float64
	for i := range points {
		if i >= len(elev) || math.IsNaN(elev[i]) {
			continue
		}
		bestD := math.Inf(1)
		bestSlope := 0.0
		for j := range points {
			if i == j || j >= len(elev) || math.IsNaN(elev[j]) {
				continue
			}
			d := carHaversineM(points[i].Lat, points[i].Lon, points[j].Lat, points[j].Lon)
			if d < 15 || d >= bestD {
				continue
			}
			bestD = d
			bestSlope = math.Abs(elev[i]-elev[j]) / d * 100
		}
		if !math.IsInf(bestD, 1) {
			slopes = append(slopes, bestSlope)
		}
	}
	if len(slopes) > 0 {
		total := 0.0
		for _, s := range slopes {
			total += s
			out.MaxSlopePct = math.Max(out.MaxSlopePct, s)
		}
		out.MeanSlopePct = total / float64(len(slopes))
	}

	side := math.Sqrt(math.Max(1, areaHa*10000))
	slopeScore := 100 / (1 + out.MeanSlopePct/15)
	reliefScore := 100 / (1 + out.ReliefM/(side*0.40+10))
	out.OperationalScore = math.Max(0, math.Min(100, slopeScore*0.72+reliefScore*0.28))

	switch {
	case areaHa < 3:
		out.Confidence = "baixa"
		out.Warning = "gleba pequena para um DEM de 90 m; use o relevo apenas como indicação geral"
	case valid < 7:
		out.Confidence = "baixa"
		out.Warning = "poucos pontos de elevação válidos"
	case areaHa < 20:
		out.Confidence = "média"
	default:
		out.Confidence = "boa"
	}
	return out
}
