package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type AreaAlternative struct {
	Key                 string             `json:"key"`
	Rank                int                `json:"rank"`
	Label               string             `json:"label"`
	Score               float64            `json:"score"`
	AreaHa              float64            `json:"area_ha"`
	PerimeterM          float64            `json:"perimeter_m"`
	CenterLat           float64            `json:"center_lat"`
	CenterLon           float64            `json:"center_lon"`
	InsideCARPct        float64            `json:"inside_car_pct"`
	CompactnessPct      float64            `json:"compactness_pct"`
	DistanceToAccessM   float64            `json:"distance_to_access_m"`
	ExistingOverlapPct  float64            `json:"existing_overlap_pct"`
	ThemeOverlapHa      map[string]float64 `json:"theme_overlap_ha"`
	ThemeOverlapPct     map[string]float64 `json:"theme_overlap_pct"`
	Terrain              TerrainMetric      `json:"terrain"`
	Flags               []string           `json:"flags"`
	Explanation         string             `json:"explanation"`
	GeoJSON             string             `json:"geojson"`
}

type AreaAlternativesResult struct {
	RequestedAreaHa float64           `json:"requested_area_ha"`
	Candidates      []AreaAlternative `json:"candidates"`
	AccessUsed      bool              `json:"access_used"`
	ThemesUsed      []string          `json:"themes_used"`
	TerrainSource   string            `json:"terrain_source"`
	Warnings        []string          `json:"warnings"`
	Method          string            `json:"method"`
}

func (a *App) GenerateProjectAreaAlternatives(propertyID int64, name, purpose string, targetAreaHa float64) (AreaAlternativesResult, error) {
	if targetAreaHa <= 0 {
		return AreaAlternativesResult{}, errors.New("informe a área desejada em hectares")
	}
	car, err := a.carForAreaIntelligence(propertyID)
	if err != nil {
		return AreaAlternativesResult{}, err
	}
	if car.GeometryAreaHa > 0 && targetAreaHa > car.GeometryAreaHa {
		return AreaAlternativesResult{}, fmt.Errorf("a área solicitada (%.2f ha) é maior que a geometria do CAR (%.2f ha)", targetAreaHa, car.GeometryAreaHa)
	}

	result := AreaAlternativesResult{
		RequestedAreaHa: targetAreaHa,
		Method: "comparação preliminar por geometria, proximidade do acesso, relevo, conflito com áreas já usadas e temas declarados do SICAR quando disponíveis",
	}
	route := car.AutoRoute
	if propertyID > 0 {
		if r, rErr := a.GetAccessRoute(propertyID); rErr == nil && validCoordinatePair(r.EntranceLat, r.EntranceLon) {
			route = r
		}
	}
	result.AccessUsed = validCoordinatePair(route.EntranceLat, route.EntranceLon)

	seeds, err := intelligentAreaSeeds(car.GeoJSON, car.CenterLat, car.CenterLon, route.EntranceLat, route.EntranceLon, targetAreaHa)
	if err != nil {
		return result, err
	}

	var existing []ProjectArea
	if propertyID > 0 {
		existing, _ = a.ListProjectAreas(propertyID)
	}
	themeGeo := map[string]string{}
	for _, code := range []string{"APP", "RESERVA_LEGAL", "VEGETACAO_NATIVA", "USO_RESTRITO", "SERVIDAO_ADMINISTRATIVA", "AREA_CONSOLIDADA"} {
		if m, ok := car.Themes.Themes[code]; ok && m.Available && strings.TrimSpace(m.GeoJSON) != "" {
			themeGeo[code] = m.GeoJSON
			result.ThemesUsed = append(result.ThemesUsed, code)
		}
	}
	if len(result.ThemesUsed) == 0 {
		result.Warnings = append(result.Warnings, "Temas detalhados do SICAR não estavam disponíveis; as opções foram comparadas por geometria, acesso e áreas já utilizadas.")
	}

	var candidates []AreaAlternative
	minCenterSeparation := math.Max(35, math.Sqrt(targetAreaHa*10000)*0.38)
	for i, seed := range seeds {
		geo, gErr := buildAutomaticCompactArea(car.GeoJSON, targetAreaHa, seed.Lat, seed.Lon)
		if gErr != nil {
			continue
		}
		metric, mErr := projectAreaMetrics(geo)
		if mErr != nil {
			continue
		}
		duplicate := false
		for _, c := range candidates {
			if carHaversineM(metric.CenterLat, metric.CenterLon, c.CenterLat, c.CenterLon) < minCenterSeparation {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		alt := AreaAlternative{
			Key:             fmt.Sprintf("alt-%02d", i+1),
			AreaHa:          metric.AreaHa,
			PerimeterM:      metric.PerimeterM,
			CenterLat:       metric.CenterLat,
			CenterLon:       metric.CenterLon,
			GeoJSON:         metric.GeoJSON,
			ThemeOverlapHa:  map[string]float64{},
			ThemeOverlapPct: map[string]float64{},
		}
		if _, inside, _, e := estimateGeometryOverlap(metric.GeoJSON, car.GeoJSON); e == nil {
			alt.InsideCARPct = inside
		}
		if alt.InsideCARPct == 0 {
			alt.InsideCARPct = 100
		}
		areaM2 := metric.AreaHa * 10000
		if metric.PerimeterM > 0 && areaM2 > 0 {
			alt.CompactnessPct = math.Min(100, 4*math.Pi*areaM2/(metric.PerimeterM*metric.PerimeterM)*100)
		}
		if result.AccessUsed {
			alt.DistanceToAccessM = carHaversineM(metric.CenterLat, metric.CenterLon, route.EntranceLat, route.EntranceLon)
		}

		for _, used := range existing {
			if strings.TrimSpace(used.GeoJSON) == "" {
				continue
			}
			_, pct, _, e := estimateGeometryOverlap(metric.GeoJSON, used.GeoJSON)
			if e == nil {
				alt.ExistingOverlapPct += pct
			}
		}
		if alt.ExistingOverlapPct > 100 {
			alt.ExistingOverlapPct = 100
		}

		for code, raw := range themeGeo {
			ha, pct, _, e := estimateGeometryOverlap(metric.GeoJSON, raw)
			if e == nil {
				alt.ThemeOverlapHa[code] = ha
				alt.ThemeOverlapPct[code] = pct
			}
		}
		candidates = append(candidates, alt)
		if len(candidates) >= 8 {
			break
		}
	}
	if len(candidates) == 0 {
		return result, errors.New("não foi possível formar opções compactas com a área solicitada dentro do CAR")
	}

	if source, terrainErr := enrichAreaAlternativesTerrain(candidates); terrainErr == nil {
		result.TerrainSource = source
	} else {
		result.Warnings = append(result.Warnings, "Relevo/declividade indisponível nesta análise: "+terrainErr.Error())
	}
	for i := range candidates {
		scoreAreaAlternative(&candidates[i], result.AccessUsed)
	}

	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	for i := range candidates {
		candidates[i].Rank = i + 1
		candidates[i].Label = fmt.Sprintf("Opção %c", 'A'+rune(i))
		candidates[i].Explanation = explainAreaAlternative(candidates[i], result.AccessUsed, len(result.ThemesUsed) > 0)
	}
	result.Candidates = candidates
	return result, nil
}

func (a *App) ChooseProjectAreaAlternative(propertyID int64, name, purpose string, alt AreaAlternative) (ProjectArea, error) {
	if strings.TrimSpace(alt.GeoJSON) == "" {
		return ProjectArea{}, errors.New("opção de área sem geometria")
	}
	if strings.TrimSpace(name) == "" {
		name = alt.Label
		if name == "" {
			name = fmt.Sprintf("Área selecionada %.2f ha", alt.AreaHa)
		}
	}
	if propertyID > 0 {
		saved, err := a.SaveProjectArea(ProjectArea{
			PropertyID: propertyID,
			Name:       name,
			Purpose:    purpose,
			GeoJSON:    alt.GeoJSON,
		})
		if err != nil {
			return ProjectArea{}, err
		}
		saved.Terrain = alt.Terrain
		if err := a.saveProjectAreaTerrain(saved.ID, alt.Terrain); err != nil {
			return ProjectArea{}, err
		}
		return saved, nil
	}
	metric, err := projectAreaMetrics(alt.GeoJSON)
	if err != nil {
		return ProjectArea{}, err
	}
	metric.Name = name
	metric.Purpose = strings.TrimSpace(purpose)
	metric.PropertyID = 0
	metric.InsideCARPct = alt.InsideCARPct
	metric.Terrain = alt.Terrain
	metric.CreatedAt = time.Now().Format(time.RFC3339)
	metric.UpdatedAt = metric.CreatedAt
	if path, err := a.writeTemporaryProjectAreaKML(metric); err == nil {
		metric.KMLPath = path
	}
	if err := a.saveTemporaryProjectArea(metric); err != nil {
		return ProjectArea{}, err
	}
	return metric, nil
}

func (a *App) carForAreaIntelligence(propertyID int64) (CARResult, error) {
	if propertyID > 0 {
		r, err := a.GetLatestCAR(propertyID)
		if err != nil || strings.TrimSpace(r.GeoJSON) == "" {
			return CARResult{}, errors.New("consulte o CAR deste imóvel antes de analisar opções de área")
		}
		return r, nil
	}
	cache, err := a.GetLastCARSession()
	if err != nil || strings.TrimSpace(cache.Result.GeoJSON) == "" {
		return CARResult{}, errors.New("consulte um CAR antes de analisar opções de área")
	}
	return cache.Result, nil
}

func intelligentAreaSeeds(carGeoJSON string, centerLat, centerLon, accessLat, accessLon, targetAreaHa float64) ([]GeoPoint, error) {
	lat0 := sharedLatitude(carGeoJSON)
	mp, err := geoJSONToPlanar(carGeoJSON, lat0)
	if err != nil || len(mp) == 0 {
		return nil, errors.New("geometria do CAR inválida para análise territorial")
	}
	minX, minY, maxX, maxY, ok := planarBounds(mp)
	if !ok {
		return nil, errors.New("limites do CAR inválidos")
	}
	const earthR = 6371008.8
	cos0 := math.Cos(lat0 * math.Pi / 180)
	toGeo := func(p planarPoint) GeoPoint {
		return GeoPoint{
			Lat: p.Y / earthR * 180 / math.Pi,
			Lon: p.X / (earthR * cos0) * 180 / math.Pi,
		}
	}
	toPlanar := func(lat, lon float64) planarPoint {
		return planarPoint{
			X: earthR * lon * math.Pi / 180 * cos0,
			Y: earthR * lat * math.Pi / 180,
		}
	}

	var seeds []GeoPoint
	add := func(p GeoPoint) {
		pp := toPlanar(p.Lat, p.Lon)
		if !pointInMultiPolygon(pp, mp) {
			return
		}
		for _, x := range seeds {
			if carHaversineM(p.Lat, p.Lon, x.Lat, x.Lon) < 30 {
				return
			}
		}
		seeds = append(seeds, p)
	}
	if validCoordinatePair(accessLat, accessLon) {
		// O snap da estrada pode cair fora do CAR. Projetamos a ideia do acesso
		// usando também pontos internos próximos encontrados na grade abaixo.
		add(GeoPoint{Lat: accessLat, Lon: accessLon})
	}
	if validCoordinatePair(centerLat, centerLon) {
		add(GeoPoint{Lat: centerLat, Lon: centerLon})
	}

	fractions := []float64{0.14, 0.30, 0.46, 0.62, 0.78, 0.90}
	type scored struct {
		p GeoPoint
		d float64
	}
	var internal []scored
	for _, fy := range fractions {
		for _, fx := range fractions {
			pp := planarPoint{X: minX + (maxX-minX)*fx, Y: minY + (maxY-minY)*fy}
			if !pointInMultiPolygon(pp, mp) {
				continue
			}
			g := toGeo(pp)
			d := 0.0
			if validCoordinatePair(accessLat, accessLon) {
				d = carHaversineM(g.Lat, g.Lon, accessLat, accessLon)
			} else if validCoordinatePair(centerLat, centerLon) {
				d = carHaversineM(g.Lat, g.Lon, centerLat, centerLon)
			}
			internal = append(internal, scored{p: g, d: d})
		}
	}
	sort.Slice(internal, func(i, j int) bool { return internal[i].d < internal[j].d })
	// Metade das sementes privilegia acesso; a outra metade distribui as opções
	// pelo imóvel para evitar devolver três glebas praticamente iguais.
	for i := 0; i < len(internal) && i < 6; i++ {
		add(internal[i].p)
	}
	if len(internal) > 6 {
		step := int(math.Max(1, float64(len(internal)-1)/5))
		for i := len(internal) - 1; i >= 0 && len(seeds) < 14; i -= step {
			add(internal[i].p)
		}
	}
	if len(seeds) == 0 {
		return nil, errors.New("não foram encontrados pontos internos para iniciar as alternativas")
	}
	return seeds, nil
}

func scoreAreaAlternative(a *AreaAlternative, hasAccess bool) {
	insideScore := math.Max(0, math.Min(1, a.InsideCARPct/100))
	compactScore := math.Max(0, math.Min(1, a.CompactnessPct/100))
	accessScore := 0.5
	if hasAccess {
		accessScore = math.Exp(-a.DistanceToAccessM / 2200)
	}
	existingScore := 1 - math.Min(1, a.ExistingOverlapPct/100)

	attention := 0.0
	weights := map[string]float64{
		"APP": 1.00, "VEGETACAO_NATIVA": 0.85, "RESERVA_LEGAL": 0.70,
		"USO_RESTRITO": 0.60, "SERVIDAO_ADMINISTRATIVA": 0.60,
	}
	weightTotal := 0.0
	for code, w := range weights {
		if pct, ok := a.ThemeOverlapPct[code]; ok {
			attention += w * math.Min(1, pct/100)
			weightTotal += w
		}
	}
	attentionScore := 1.0
	if weightTotal > 0 {
		attentionScore = 1 - math.Min(1, attention/weightTotal)
	}

	consolidatedBonus := 0.0
	if pct, ok := a.ThemeOverlapPct["AREA_CONSOLIDADA"]; ok {
		consolidatedBonus = math.Min(5, pct/100*5)
	}

	terrainScore := 0.5
	if a.Terrain.Available {
		terrainScore = math.Max(0, math.Min(1, a.Terrain.OperationalScore/100))
	}
	a.Score = insideScore*25 + compactScore*15 + accessScore*20 + existingScore*20 + attentionScore*10 + terrainScore*10 + consolidatedBonus
	if a.Score > 100 {
		a.Score = 100
	}
	if a.ExistingOverlapPct > 2 {
		a.Flags = append(a.Flags, fmt.Sprintf("sobrepõe %.1f%% de área de projeto já registrada", a.ExistingOverlapPct))
	}
	for _, code := range []string{"APP", "VEGETACAO_NATIVA", "RESERVA_LEGAL", "USO_RESTRITO", "SERVIDAO_ADMINISTRATIVA"} {
		if pct := a.ThemeOverlapPct[code]; pct > 0.5 {
			a.Flags = append(a.Flags, fmt.Sprintf("%s declarado: %.1f%% da gleba", areaThemeShortLabel(code), pct))
		}
	}
	if a.Terrain.Available {
		if a.Terrain.MeanSlopePct >= 15 {
			a.Flags = append(a.Flags, fmt.Sprintf("relevo mais acentuado: inclinação média estimada %.1f%%", a.Terrain.MeanSlopePct))
		}
		if a.Terrain.Warning != "" {
			a.Flags = append(a.Flags, "relevo: "+a.Terrain.Warning)
		}
	}
}

func explainAreaAlternative(a AreaAlternative, hasAccess, hasThemes bool) string {
	parts := []string{
		fmt.Sprintf("%.2f ha", a.AreaHa),
		fmt.Sprintf("%.1f%% dentro do CAR", a.InsideCARPct),
		fmt.Sprintf("compacidade %.0f%%", a.CompactnessPct),
	}
	if hasAccess {
		parts = append(parts, fmt.Sprintf("%.0f m do acesso estimado", a.DistanceToAccessM))
	}
	if a.ExistingOverlapPct <= 0.5 {
		parts = append(parts, "sem conflito relevante com áreas de projeto já salvas")
	}
	if hasThemes {
		maxAttention := 0.0
		for _, code := range []string{"APP", "VEGETACAO_NATIVA", "RESERVA_LEGAL", "USO_RESTRITO", "SERVIDAO_ADMINISTRATIVA"} {
			if a.ThemeOverlapPct[code] > maxAttention {
				maxAttention = a.ThemeOverlapPct[code]
			}
		}
		if maxAttention <= 0.5 {
			parts = append(parts, "sem sobreposição relevante nos temas de atenção disponíveis")
		}
		if pct := a.ThemeOverlapPct["AREA_CONSOLIDADA"]; pct > 0.5 {
			parts = append(parts, fmt.Sprintf("%.0f%% em área consolidada declarada", pct))
		}
	}
	if a.Terrain.Available {
		parts = append(parts, fmt.Sprintf("altitude média %.0f m", a.Terrain.ElevationMeanM))
		parts = append(parts, fmt.Sprintf("inclinação média estimada %.1f%%", a.Terrain.MeanSlopePct))
	}
	return strings.Join(parts, " • ")
}

func areaThemeShortLabel(code string) string {
	switch code {
	case "APP":
		return "APP"
	case "RESERVA_LEGAL":
		return "Reserva Legal"
	case "VEGETACAO_NATIVA":
		return "Vegetação Nativa"
	case "USO_RESTRITO":
		return "Uso Restrito"
	case "SERVIDAO_ADMINISTRATIVA":
		return "Servidão"
	case "AREA_CONSOLIDADA":
		return "Área Consolidada"
	default:
		return code
	}
}
