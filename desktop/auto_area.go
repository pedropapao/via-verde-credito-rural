package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

type autoGridCell struct{ X, Y int }
type autoGridVertex struct{ X, Y int }

func (a *App) GenerateAutomaticProjectArea(propertyID int64, name, purpose string, targetAreaHa float64) (ProjectArea, error) {
	if propertyID <= 0 {
		return ProjectArea{}, errors.New("selecione um imóvel")
	}
	if targetAreaHa <= 0 {
		return ProjectArea{}, errors.New("informe a área desejada em hectares")
	}
	car, err := a.GetLatestCAR(propertyID)
	if err != nil || strings.TrimSpace(car.GeoJSON) == "" {
		return ProjectArea{}, errors.New("consulte o CAR do imóvel antes de gerar a área automática")
	}
	if car.GeometryAreaHa > 0 && targetAreaHa > car.GeometryAreaHa {
		return ProjectArea{}, fmt.Errorf("a área solicitada (%.2f ha) é maior que a geometria do CAR (%.2f ha)", targetAreaHa, car.GeometryAreaHa)
	}
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Área automática %.2f ha", targetAreaHa)
	}

	seedLat, seedLon := car.CenterLat, car.CenterLon
	if route, rErr := a.GetAccessRoute(propertyID); rErr == nil && validCoordinatePair(route.EntranceLat, route.EntranceLon) {
		seedLat, seedLon = route.EntranceLat, route.EntranceLon
	}
	geo, err := buildAutomaticCompactArea(car.GeoJSON, targetAreaHa, seedLat, seedLon)
	if err != nil {
		return ProjectArea{}, err
	}
	area, err := a.SaveProjectArea(ProjectArea{
		PropertyID: propertyID,
		Name:       name,
		Purpose:    purpose,
		GeoJSON:    geo,
	})
	if err != nil {
		return ProjectArea{}, err
	}
	return area, nil
}

func buildAutomaticCompactArea(carGeoJSON string, targetAreaHa, seedLat, seedLon float64) (string, error) {
	if targetAreaHa <= 0 {
		return "", errors.New("área desejada inválida")
	}
	lat0 := sharedLatitude(carGeoJSON)
	polys, err := geoJSONToPlanar(carGeoJSON, lat0)
	if err != nil || len(polys) == 0 {
		return "", errors.New("geometria do CAR não pôde ser preparada para delimitação")
	}
	minX, minY, maxX, maxY, ok := planarBounds(polys)
	if !ok {
		return "", errors.New("limites do CAR inválidos")
	}

	const cellsWanted = 1000
	targetM2 := targetAreaHa * 10000
	cellArea := targetM2 / cellsWanted
	side := math.Sqrt(cellArea)
	if side < 0.20 {
		side = 0.20
		cellArea = side * side
	}
	originX, originY := minX-side*2, minY-side*2

	const earthR = 6371008.8
	cos0 := math.Cos(lat0 * math.Pi / 180)
	toPlanar := func(lat, lon float64) planarPoint {
		return planarPoint{
			X: earthR * lon * math.Pi / 180 * cos0,
			Y: earthR * lat * math.Pi / 180,
		}
	}
	toCell := func(p planarPoint) autoGridCell {
		return autoGridCell{
			X: int(math.Floor((p.X - originX) / side)),
			Y: int(math.Floor((p.Y - originY) / side)),
		}
	}
	cellPoint := func(c autoGridCell, fx, fy float64) planarPoint {
		return planarPoint{X: originX + (float64(c.X)+fx)*side, Y: originY + (float64(c.Y)+fy)*side}
	}
	cellValid := func(c autoGridCell) bool {
		pts := []planarPoint{
			cellPoint(c, .5, .5),
			cellPoint(c, .08, .08),
			cellPoint(c, .92, .08),
			cellPoint(c, .92, .92),
			cellPoint(c, .08, .92),
		}
		for _, p := range pts {
			if p.X < minX || p.X > maxX || p.Y < minY || p.Y > maxY || !pointInMultiPolygon(p, polys) {
				return false
			}
		}
		return true
	}
	findSeed := func(initial autoGridCell, maxRadius int) (autoGridCell, bool) {
		if cellValid(initial) {
			return initial, true
		}
		for r := 1; r <= maxRadius; r++ {
			for dx := -r; dx <= r; dx++ {
				for _, dy := range []int{-r, r} {
					c := autoGridCell{initial.X + dx, initial.Y + dy}
					if cellValid(c) {
						return c, true
					}
				}
			}
			for dy := -r + 1; dy <= r-1; dy++ {
				for _, dx := range []int{-r, r} {
					c := autoGridCell{initial.X + dx, initial.Y + dy}
					if cellValid(c) {
						return c, true
					}
				}
			}
		}
		return autoGridCell{}, false
	}

	seed := toCell(toPlanar(seedLat, seedLon))
	start, found := findSeed(seed, 100)
	if !found {
		// O centro da caixa é um fallback melhor quando o ponto de acesso viário
		// ficou fora do polígono por causa do snap da estrada.
		start, found = findSeed(toCell(planarPoint{X: (minX + maxX) / 2, Y: (minY + maxY) / 2}), 180)
	}
	if !found {
		return "", errors.New("não foi encontrado espaço interno suficiente para iniciar a delimitação automática")
	}

	selected := map[autoGridCell]bool{}
	seen := map[autoGridCell]bool{start: true}
	queue := []autoGridCell{start}
	dirs := []autoGridCell{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}
	maxVisited := 500000
	visited := 0
	for len(queue) > 0 && len(selected) < cellsWanted && visited < maxVisited {
		c := queue[0]
		queue = queue[1:]
		visited++
		if !cellValid(c) {
			continue
		}
		selected[c] = true
		for _, d := range dirs {
			n := autoGridCell{c.X + d.X, c.Y + d.Y}
			if !seen[n] {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	if len(selected) < cellsWanted {
		achieved := float64(len(selected)) * cellArea / 10000
		return "", fmt.Errorf("a região interna contínua encontrada comportou aproximadamente %.2f ha; não foi possível delimitar %.2f ha de forma compacta", achieved, targetAreaHa)
	}

	loops := traceSelectedGridBoundary(selected)
	if len(loops) == 0 {
		return "", errors.New("não foi possível construir o contorno da área automática")
	}

	// Se houver mais de um laço, o de maior área positiva é o limite externo;
	// laços negativos são buracos internos.
	outerIdx := -1
	outerArea := 0.0
	for i, loop := range loops {
		a := signedGridArea(loop)
		if a > outerArea {
			outerArea, outerIdx = a, i
		}
	}
	if outerIdx < 0 {
		return "", errors.New("contorno externo da área automática inválido")
	}
	ordered := [][]autoGridVertex{simplifyGridLoop(loops[outerIdx])}
	for i, loop := range loops {
		if i == outerIdx {
			continue
		}
		if signedGridArea(loop) < 0 {
			ordered = append(ordered, simplifyGridLoop(loop))
		}
	}

	rings := make([][][]float64, 0, len(ordered))
	for _, loop := range ordered {
		ring := make([][]float64, 0, len(loop)+1)
		for _, v := range loop {
			x := originX + float64(v.X)*side
			y := originY + float64(v.Y)*side
			lon := x / (earthR * cos0) * 180 / math.Pi
			lat := y / earthR * 180 / math.Pi
			ring = append(ring, []float64{lon, lat})
		}
		if len(ring) > 0 {
			first := ring[0]
			last := ring[len(ring)-1]
			if first[0] != last[0] || first[1] != last[1] {
				ring = append(ring, []float64{first[0], first[1]})
			}
		}
		rings = append(rings, ring)
	}
	feature := map[string]any{
		"type": "Feature",
		"properties": map[string]any{
			"source":         "Via Verde CAR - delimitação automática",
			"target_area_ha": targetAreaHa,
			"method":         "malha interna compacta a partir do acesso/centro do CAR",
		},
		"geometry": map[string]any{"type": "Polygon", "coordinates": rings},
	}
	b, _ := json.Marshal(feature)
	return string(b), nil
}

func traceSelectedGridBoundary(selected map[autoGridCell]bool) [][]autoGridVertex {
	adj := map[autoGridVertex][]autoGridVertex{}
	add := func(a, b autoGridVertex) { adj[a] = append(adj[a], b) }
	for c := range selected {
		if !selected[autoGridCell{c.X, c.Y - 1}] {
			add(autoGridVertex{c.X, c.Y}, autoGridVertex{c.X + 1, c.Y})
		}
		if !selected[autoGridCell{c.X + 1, c.Y}] {
			add(autoGridVertex{c.X + 1, c.Y}, autoGridVertex{c.X + 1, c.Y + 1})
		}
		if !selected[autoGridCell{c.X, c.Y + 1}] {
			add(autoGridVertex{c.X + 1, c.Y + 1}, autoGridVertex{c.X, c.Y + 1})
		}
		if !selected[autoGridCell{c.X - 1, c.Y}] {
			add(autoGridVertex{c.X, c.Y + 1}, autoGridVertex{c.X, c.Y})
		}
	}
	var loops [][]autoGridVertex
	for {
		var start autoGridVertex
		found := false
		for k, outs := range adj {
			if len(outs) > 0 {
				start, found = k, true
				break
			}
		}
		if !found {
			break
		}
		cur := start
		loop := []autoGridVertex{start}
		for guard := 0; guard < 200000; guard++ {
			outs := adj[cur]
			if len(outs) == 0 {
				break
			}
			next := outs[len(outs)-1]
			adj[cur] = outs[:len(outs)-1]
			cur = next
			if cur == start {
				break
			}
			loop = append(loop, cur)
		}
		if len(loop) >= 4 {
			loops = append(loops, loop)
		}
	}
	return loops
}

func signedGridArea(loop []autoGridVertex) float64 {
	if len(loop) < 3 {
		return 0
	}
	s := 0.0
	for i := range loop {
		j := (i + 1) % len(loop)
		s += float64(loop[i].X*loop[j].Y - loop[j].X*loop[i].Y)
	}
	return s / 2
}

func simplifyGridLoop(loop []autoGridVertex) []autoGridVertex {
	if len(loop) < 4 {
		return loop
	}
	out := make([]autoGridVertex, 0, len(loop))
	n := len(loop)
	for i := 0; i < n; i++ {
		prev := loop[(i-1+n)%n]
		cur := loop[i]
		next := loop[(i+1)%n]
		dx1, dy1 := cur.X-prev.X, cur.Y-prev.Y
		dx2, dy2 := next.X-cur.X, next.Y-cur.Y
		if dx1*dy2 == dy1*dx2 {
			continue
		}
		out = append(out, cur)
	}
	return out
}
