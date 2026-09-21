package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"net/http"
	"sync"
	"time"
)

var terrainTileURLTemplate = "https://s3.amazonaws.com/elevation-tiles-prod/terrarium/%d/%d/%d.png"

const terrainTileZoom = 13

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

type indexedTerrainPoint struct {
	Candidate int
	Point     GeoPoint
}

type terrainTileKey struct{ Z, X, Y int }

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

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	return "Terrain Tiles / AWS Open Data (mosaico SRTM, GMTED2010, ETOPO1 e outras fontes)", nil
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
	type sample struct {
		key terrainTileKey
		px  int
		py  int
	}
	samples := make([]sample, len(indexed))
	keys := map[terrainTileKey]bool{}
	for i, item := range indexed {
		x, y, px, py := terrainTilePixel(item.Point.Lat, item.Point.Lon, terrainTileZoom)
		k := terrainTileKey{Z: terrainTileZoom, X: x, Y: y}
		samples[i] = sample{key: k, px: px, py: py}
		keys[k] = true
	}

	tiles := map[terrainTileKey]image.Image{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	var firstErr error
	for k := range keys {
		k := k
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			img, err := fetchTerrainTile(ctx, k)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			tiles[k] = img
		}()
	}
	wg.Wait()
	if len(tiles) == 0 {
		if firstErr == nil {
			firstErr = errors.New("nenhum tile de relevo disponível")
		}
		return nil, firstErr
	}

	out := make([]float64, len(samples))
	for i, s := range samples {
		img := tiles[s.key]
		if img == nil {
			out[i] = math.NaN()
			continue
		}
		r16, g16, b16, a16 := img.At(s.px, s.py).RGBA()
		if a16 == 0 {
			out[i] = math.NaN()
			continue
		}
		r := float64(r16 >> 8)
		g := float64(g16 >> 8)
		b := float64(b16 >> 8)
		z := (r*256 + g + b/256) - 32768
		if z < -500 || z > 9000 {
			out[i] = math.NaN()
			continue
		}
		out[i] = z
	}
	return out, nil
}

func fetchTerrainTile(ctx context.Context, k terrainTileKey) (image.Image, error) {
	target := fmt.Sprintf(terrainTileURLTemplate, k.Z, k.X, k.Y)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tile de relevo respondeu HTTP %d", resp.StatusCode)
	}
	img, _, err := image.Decode(resp.Body)
	return img, err
}

func terrainTilePixel(lat, lon float64, z int) (tileX, tileY, px, py int) {
	lat = math.Max(-85.05112878, math.Min(85.05112878, lat))
	n := math.Exp2(float64(z))
	xf := (lon + 180) / 360 * n
	latRad := lat * math.Pi / 180
	yf := (1 - math.Asinh(math.Tan(latRad))/math.Pi) / 2 * n
	tileX = int(math.Floor(xf))
	tileY = int(math.Floor(yf))
	px = int(math.Floor((xf - float64(tileX)) * 256))
	py = int(math.Floor((yf - float64(tileY)) * 256))
	if px < 0 {
		px = 0
	}
	if px > 255 {
		px = 255
	}
	if py < 0 {
		py = 0
	}
	if py > 255 {
		py = 255
	}
	return
}

func calculateTerrainMetric(points []GeoPoint, elev []float64, areaHa float64) TerrainMetric {
	out := TerrainMetric{
		ResolutionM: 30,
		Source:      "Terrain Tiles / AWS Open Data; fontes incluem SRTM, GMTED2010 e ETOPO1",
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
			if d < 12 || d >= bestD {
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
	case areaHa < 2:
		out.Confidence = "baixa"
		out.Warning = "gleba pequena em relação à resolução do modelo; use o relevo apenas como indicação geral"
	case valid < 7:
		out.Confidence = "baixa"
		out.Warning = "poucos pontos de elevação válidos"
	case areaHa < 15:
		out.Confidence = "média"
	default:
		out.Confidence = "boa"
	}
	return out
}
