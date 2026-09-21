package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	nominatimReverseURL = "https://nominatim.openstreetmap.org/reverse"
	nominatimSearchURL  = "https://nominatim.openstreetmap.org/search"
	osrmBaseURL         = "https://router.project-osrm.org"
)

type nominatimReverseResponse struct {
	Lat         string            `json:"lat"`
	Lon         string            `json:"lon"`
	DisplayName string            `json:"display_name"`
	Address     map[string]string `json:"address"`
}

type nominatimSearchItem struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
}

type osrmWaypoint struct {
	Name     string    `json:"name"`
	Location []float64 `json:"location"`
	Distance float64   `json:"distance"`
}

type osrmTableResponse struct {
	Code         string         `json:"code"`
	Durations    [][]*float64   `json:"durations"`
	Distances    [][]*float64   `json:"distances"`
	Destinations []osrmWaypoint `json:"destinations"`
}

type osrmRouteResponse struct {
	Code      string         `json:"code"`
	Waypoints []osrmWaypoint `json:"waypoints"`
	Routes    []struct {
		Distance float64         `json:"distance"`
		Duration float64         `json:"duration"`
		Geometry map[string]any  `json:"geometry"`
		Legs     []struct {
			Steps []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Name     string  `json:"name"`
				Maneuver struct {
					Type     string `json:"type"`
					Modifier string `json:"modifier"`
					Exit     int    `json:"exit"`
				} `json:"maneuver"`
			} `json:"steps"`
		} `json:"legs"`
	} `json:"routes"`
}

func (a *App) RegenerateAutomaticAccessRoute(propertyID int64) (AccessRoute, error) {
	if propertyID <= 0 {
		return AccessRoute{}, errors.New("selecione um imóvel")
	}
	car, err := a.GetLatestCAR(propertyID)
	if err != nil {
		return AccessRoute{}, errors.New("consulte o CAR antes de gerar o roteiro automático")
	}
	return a.generateAutomaticAccessRoute(propertyID, car, true)
}

func (a *App) generateAutomaticAccessRoute(propertyID int64, car CARResult, force bool) (AccessRoute, error) {
	if !car.HasGeometry || strings.TrimSpace(car.GeoJSON) == "" {
		return AccessRoute{}, errors.New("geometria do CAR indisponível para calcular acesso")
	}
	if propertyID > 0 && !force {
		if cached, err := a.loadAutomaticRoute(propertyID); err == nil && cached.GeneratedAt != "" {
			if t, parseErr := time.Parse(time.RFC3339, cached.GeneratedAt); parseErr == nil && time.Since(t) < 30*24*time.Hour {
				return cached, nil
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 38*time.Second)
	defer cancel()

	label, cityLat, cityLon, err := a.nearestCityReference(ctx, car)
	if err != nil {
		return AccessRoute{}, err
	}
	candidates, err := routeBoundaryCandidates(car.GeoJSON, 18)
	if err != nil || len(candidates) == 0 {
		return AccessRoute{}, errors.New("não foi possível montar pontos de acesso no perímetro do CAR")
	}

	best, err := chooseRoadAccessCandidate(ctx, cityLat, cityLon, candidates)
	if err != nil {
		return AccessRoute{}, err
	}
	route, err := fetchOSRMRoute(ctx, cityLat, cityLon, best.Lat, best.Lon)
	if err != nil {
		return AccessRoute{}, err
	}

	out := AccessRoute{
		PropertyID:       propertyID,
		ReferenceLabel:   label,
		ReferenceLat:     cityLat,
		ReferenceLon:     cityLon,
		EntranceLat:      best.Lat,
		EntranceLon:      best.Lon,
		HeadquartersLat:  car.CenterLat,
		HeadquartersLon:  car.CenterLon,
		RouteDistanceKm:  route.RouteDistanceKm,
		RouteDurationMin: route.RouteDurationMin,
		RouteGeoJSON:     route.RouteGeoJSON,
		Steps:            route.Steps,
		RouteSource:      "OpenStreetMap/Nominatim + OSRM (rota viária estimada)",
		Automatic:        true,
		GeneratedAt:      time.Now().Format(time.RFC3339),
		UpdatedAt:        time.Now().Format(time.RFC3339),
	}
	// A coordenada efetivamente usada pelo roteador pode ser um ponto da via
	// alguns metros fora do limite amostrado. Ela é mais útil como acesso.
	if validCoordinatePair(route.EntranceLat, route.EntranceLon) {
		out.EntranceLat = route.EntranceLat
		out.EntranceLon = route.EntranceLon
	}
	out.ReferenceToEntranceKm = out.RouteDistanceKm
	out.Text = buildAutomaticRouteText(out)
	out.GoogleMapsURL = fmt.Sprintf(
		"https://www.google.com/maps/dir/?api=1&origin=%.8f,%.8f&destination=%.8f,%.8f&travelmode=driving",
		out.ReferenceLat, out.ReferenceLon, out.EntranceLat, out.EntranceLon,
	)

	if propertyID > 0 && a.db != nil {
		if err := a.saveAutomaticRoute(out, car); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (a *App) nearestCityReference(ctx context.Context, car CARResult) (string, float64, float64, error) {
	cacheKey := strings.ToUpper(strings.TrimSpace(car.Municipality + "|" + car.UF))
	if a.db != nil && cacheKey != "|" {
		var label string
		var lat, lon float64
		var updated string
		err := a.db.QueryRow(`SELECT label,lat,lon,updated_at FROM route_city_cache WHERE cache_key=?`, cacheKey).Scan(&label, &lat, &lon, &updated)
		if err == nil && validCoordinatePair(lat, lon) {
			if t, e := time.Parse(time.RFC3339, updated); e == nil && time.Since(t) < 180*24*time.Hour {
				return label, lat, lon, nil
			}
		}
	}

	// Primeiro pedimos ao geocodificador um núcleo urbano/administrativo próximo
	// ao centro do CAR. Isso costuma ser melhor que assumir a sede municipal.
	params := url.Values{}
	params.Set("format", "jsonv2")
	params.Set("lat", strconv.FormatFloat(car.CenterLat, 'f', 8, 64))
	params.Set("lon", strconv.FormatFloat(car.CenterLon, 'f', 8, 64))
	params.Set("zoom", "10")
	params.Set("addressdetails", "1")
	var rev nominatimReverseResponse
	err := fetchJSONWithUA(ctx, nominatimReverseURL+"?"+params.Encode(), &rev)
	if err == nil {
		lat, _ := strconv.ParseFloat(rev.Lat, 64)
		lon, _ := strconv.ParseFloat(rev.Lon, 64)
		label := firstNonEmpty(
			rev.Address["city"],
			rev.Address["town"],
			rev.Address["village"],
			rev.Address["municipality"],
			car.Municipality,
		)
		if validCoordinatePair(lat, lon) && label != "" {
			label = label + " / " + car.UF
			a.cacheCityReference(cacheKey, label, lat, lon)
			return label, lat, lon, nil
		}
	}

	// Fallback: busca a sede do município informado pelo próprio CAR.
	if strings.TrimSpace(car.Municipality) == "" {
		return "", 0, 0, errors.New("não foi possível determinar a cidade de referência")
	}
	select {
	case <-ctx.Done():
		return "", 0, 0, ctx.Err()
	case <-time.After(1100 * time.Millisecond):
	}
	q := url.Values{}
	q.Set("format", "jsonv2")
	q.Set("limit", "1")
	q.Set("countrycodes", "br")
	q.Set("q", car.Municipality+", "+car.UF+", Brasil")
	var items []nominatimSearchItem
	if err := fetchJSONWithUA(ctx, nominatimSearchURL+"?"+q.Encode(), &items); err != nil || len(items) == 0 {
		return "", 0, 0, errors.New("cidade de referência não localizada automaticamente")
	}
	lat, _ := strconv.ParseFloat(items[0].Lat, 64)
	lon, _ := strconv.ParseFloat(items[0].Lon, 64)
	if !validCoordinatePair(lat, lon) {
		return "", 0, 0, errors.New("coordenadas da cidade de referência inválidas")
	}
	label := car.Municipality + " / " + car.UF
	a.cacheCityReference(cacheKey, label, lat, lon)
	return label, lat, lon, nil
}

func (a *App) cacheCityReference(key, label string, lat, lon float64) {
	if a == nil || a.db == nil || strings.TrimSpace(key) == "" {
		return
	}
	_, _ = a.db.Exec(`INSERT INTO route_city_cache(cache_key,label,lat,lon,updated_at) VALUES(?,?,?,?,?)
		ON CONFLICT(cache_key) DO UPDATE SET label=excluded.label,lat=excluded.lat,lon=excluded.lon,updated_at=excluded.updated_at`,
		key, label, lat, lon, time.Now().Format(time.RFC3339))
}

func fetchJSONWithUA(ctx context.Context, target string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion+" desktop-rural-management")
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Timeout: 18 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("serviço de localização respondeu HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

func routeBoundaryCandidates(raw string, maxPoints int) ([]GeoPoint, error) {
	var f carGeoFeature
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return nil, err
	}
	polys, err := carGeometryPolygons(f.Geometry)
	if err != nil || len(polys) == 0 || len(polys[0]) == 0 {
		return nil, errors.New("perímetro do CAR inválido")
	}
	// Escolhe o maior anel externo para evitar direcionar o acesso para uma ilha
	// pequena de um eventual MultiPolygon.
	best := polys[0][0]
	bestArea := ringAreaLonLat(best)
	for _, poly := range polys[1:] {
		if len(poly) == 0 {
			continue
		}
		a := ringAreaLonLat(poly[0])
		if a > bestArea {
			best, bestArea = poly[0], a
		}
	}
	if len(best) < 3 {
		return nil, errors.New("perímetro insuficiente")
	}
	if maxPoints < 6 {
		maxPoints = 6
	}
	step := int(math.Ceil(float64(len(best)) / float64(maxPoints)))
	if step < 1 {
		step = 1
	}
	out := make([]GeoPoint, 0, maxPoints)
	for i := 0; i < len(best) && len(out) < maxPoints; i += step {
		if len(best[i]) >= 2 {
			out = append(out, GeoPoint{Lat: best[i][1], Lon: best[i][0]})
		}
	}
	return out, nil
}

func ringAreaLonLat(ring [][]float64) float64 {
	if len(ring) < 3 {
		return 0
	}
	pts := make([]GeoPoint, 0, len(ring))
	for _, p := range ring {
		if len(p) >= 2 {
			pts = append(pts, GeoPoint{Lat: p[1], Lon: p[0]})
		}
	}
	return math.Abs(projectedArea(pts))
}

func chooseRoadAccessCandidate(ctx context.Context, cityLat, cityLon float64, candidates []GeoPoint) (GeoPoint, error) {
	coords := []string{fmt.Sprintf("%.8f,%.8f", cityLon, cityLat)}
	for _, p := range candidates {
		coords = append(coords, fmt.Sprintf("%.8f,%.8f", p.Lon, p.Lat))
	}
	dest := make([]string, len(candidates))
	for i := range candidates {
		dest[i] = strconv.Itoa(i + 1)
	}
	target := osrmBaseURL + "/table/v1/driving/" + strings.Join(coords, ";") +
		"?sources=0&destinations=" + strings.Join(dest, ";") + "&annotations=duration,distance"
	var table osrmTableResponse
	if err := fetchJSONWithUA(ctx, target, &table); err != nil {
		return GeoPoint{}, err
	}
	if table.Code != "Ok" || len(table.Durations) == 0 {
		return GeoPoint{}, errors.New("roteador não encontrou acesso viário ao imóvel")
	}
	bestIdx := -1
	bestScore := math.Inf(1)
	for i := range candidates {
		if i >= len(table.Durations[0]) || table.Durations[0][i] == nil {
			continue
		}
		duration := *table.Durations[0][i]
		snap := 0.0
		if i < len(table.Destinations) {
			snap = table.Destinations[i].Distance
		}
		// Penaliza fortemente pontos que precisam ser deslocados muitos metros
		// até uma via; entre acessos equivalentes, escolhe a rota mais rápida.
		score := duration + math.Min(snap, 5000)*8
		if score < bestScore {
			bestScore, bestIdx = score, i
		}
	}
	if bestIdx < 0 {
		return GeoPoint{}, errors.New("nenhum ponto do perímetro foi conectado à malha viária")
	}
	return candidates[bestIdx], nil
}

func fetchOSRMRoute(ctx context.Context, fromLat, fromLon, toLat, toLon float64) (AccessRoute, error) {
	target := fmt.Sprintf(
		"%s/route/v1/driving/%.8f,%.8f;%.8f,%.8f?steps=true&geometries=geojson&overview=full&alternatives=false",
		osrmBaseURL, fromLon, fromLat, toLon, toLat,
	)
	var rr osrmRouteResponse
	if err := fetchJSONWithUA(ctx, target, &rr); err != nil {
		return AccessRoute{}, err
	}
	if rr.Code != "Ok" || len(rr.Routes) == 0 {
		return AccessRoute{}, errors.New("rota viária não encontrada")
	}
	r := rr.Routes[0]
	out := AccessRoute{
		RouteDistanceKm:  r.Distance / 1000,
		RouteDurationMin: r.Duration / 60,
	}
	if len(rr.Waypoints) > 1 && len(rr.Waypoints[1].Location) >= 2 {
		out.EntranceLon = rr.Waypoints[1].Location[0]
		out.EntranceLat = rr.Waypoints[1].Location[1]
	}
	feature := map[string]any{
		"type":       "Feature",
		"properties": map[string]any{"source": "OSRM/OpenStreetMap"},
		"geometry":   r.Geometry,
	}
	b, _ := json.Marshal(feature)
	out.RouteGeoJSON = string(b)
	for _, leg := range r.Legs {
		for _, step := range leg.Steps {
			if step.Distance < 3 && step.Maneuver.Type != "arrive" && step.Maneuver.Type != "depart" {
				continue
			}
			out.Steps = append(out.Steps, RouteStep{
				Instruction: translateOSRMManeuver(step.Maneuver.Type, step.Maneuver.Modifier, step.Maneuver.Exit),
				Road:        strings.TrimSpace(step.Name),
				DistanceKm:  step.Distance / 1000,
				DurationMin: step.Duration / 60,
			})
		}
	}
	return out, nil
}

func translateOSRMManeuver(kind, modifier string, exit int) string {
	dir := map[string]string{
		"left": "à esquerda", "right": "à direita",
		"slight left": "levemente à esquerda", "slight right": "levemente à direita",
		"sharp left": "acentuadamente à esquerda", "sharp right": "acentuadamente à direita",
		"straight": "em frente", "uturn": "fazendo retorno",
	}[modifier]
	switch kind {
	case "depart":
		return "Saia do ponto de referência e inicie a rota"
	case "arrive":
		return "Chegue ao acesso viário estimado da propriedade"
	case "turn":
		if dir != "" {
			return "Vire " + dir
		}
		return "Faça a conversão indicada"
	case "continue":
		if dir != "" {
			return "Continue " + dir
		}
		return "Continue pela via"
	case "new name":
		return "Continue pela via"
	case "merge":
		if dir != "" {
			return "Entre na via " + dir
		}
		return "Entre na via indicada"
	case "fork":
		if dir != "" {
			return "Na bifurcação, mantenha-se " + dir
		}
		return "Siga pela bifurcação indicada"
	case "roundabout", "rotary":
		if exit > 0 {
			return fmt.Sprintf("Entre na rotatória e utilize a saída %d", exit)
		}
		return "Entre na rotatória e siga pela saída indicada"
	case "exit roundabout", "exit rotary":
		return "Saia da rotatória"
	default:
		if dir != "" {
			return "Siga " + dir
		}
		return "Siga pela rota indicada"
	}
}

func buildAutomaticRouteText(x AccessRoute) string {
	if !validCoordinatePair(x.ReferenceLat, x.ReferenceLon) || !validCoordinatePair(x.EntranceLat, x.EntranceLon) {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Referência automática: %s, coordenadas %.6f, %.6f. ", firstNonEmpty(x.ReferenceLabel, "núcleo urbano próximo"), x.ReferenceLat, x.ReferenceLon)
	fmt.Fprintf(&b, "O melhor acesso viário estimado ao perímetro consultado foi localizado em %.6f, %.6f. ", x.EntranceLat, x.EntranceLon)
	if x.RouteDistanceKm > 0 {
		fmt.Fprintf(&b, "A rota estimada possui %.2f km", x.RouteDistanceKm)
		if x.RouteDurationMin > 0 {
			fmt.Fprintf(&b, " e tempo aproximado de %.0f minutos", x.RouteDurationMin)
		}
		b.WriteString(". ")
	}
	if validCoordinatePair(x.HeadquartersLat, x.HeadquartersLon) {
		fmt.Fprintf(&b, "O centro geométrico aproximado do CAR está em %.6f, %.6f. ", x.HeadquartersLat, x.HeadquartersLon)
	}
	b.WriteString("A entrada é estimada automaticamente a partir da malha viária pública e de pontos do perímetro do CAR; confirme porteira, estrada particular e condições reais de acesso em campo.")
	return b.String()
}

func (a *App) saveAutomaticRoute(x AccessRoute, car CARResult) error {
	if a.db == nil {
		return nil
	}
	steps, _ := json.Marshal(x.Steps)
	_, err := a.db.Exec(`INSERT INTO automatic_routes(property_id,car_number,reference_label,reference_lat,reference_lon,entrance_lat,entrance_lon,center_lat,center_lon,route_distance_km,route_duration_min,route_geojson,steps_json,route_source,generated_at,car_fingerprint)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(property_id) DO UPDATE SET car_number=excluded.car_number,reference_label=excluded.reference_label,reference_lat=excluded.reference_lat,reference_lon=excluded.reference_lon,entrance_lat=excluded.entrance_lat,entrance_lon=excluded.entrance_lon,center_lat=excluded.center_lat,center_lon=excluded.center_lon,route_distance_km=excluded.route_distance_km,route_duration_min=excluded.route_duration_min,route_geojson=excluded.route_geojson,steps_json=excluded.steps_json,route_source=excluded.route_source,generated_at=excluded.generated_at,car_fingerprint=excluded.car_fingerprint`,
		x.PropertyID, car.CAR, x.ReferenceLabel, x.ReferenceLat, x.ReferenceLon, x.EntranceLat, x.EntranceLon, x.HeadquartersLat, x.HeadquartersLon, x.RouteDistanceKm, x.RouteDurationMin, x.RouteGeoJSON, string(steps), x.RouteSource, x.GeneratedAt, geometryFingerprint(car.GeoJSON))
	return err
}

func (a *App) loadAutomaticRoute(propertyID int64) (AccessRoute, error) {
	if a.db == nil {
		return AccessRoute{}, sql.ErrNoRows
	}
	var x AccessRoute
	var stepsRaw string
	err := a.db.QueryRow(`SELECT property_id,reference_label,reference_lat,reference_lon,entrance_lat,entrance_lon,center_lat,center_lon,route_distance_km,route_duration_min,route_geojson,steps_json,route_source,generated_at
		FROM automatic_routes WHERE property_id=?`, propertyID).
		Scan(&x.PropertyID, &x.ReferenceLabel, &x.ReferenceLat, &x.ReferenceLon, &x.EntranceLat, &x.EntranceLon, &x.HeadquartersLat, &x.HeadquartersLon, &x.RouteDistanceKm, &x.RouteDurationMin, &x.RouteGeoJSON, &stepsRaw, &x.RouteSource, &x.GeneratedAt)
	if err != nil {
		return AccessRoute{}, err
	}
	_ = json.Unmarshal([]byte(stepsRaw), &x.Steps)
	x.Automatic = true
	x.ReferenceToEntranceKm = x.RouteDistanceKm
	x.Text = buildAutomaticRouteText(x)
	x.GoogleMapsURL = fmt.Sprintf("https://www.google.com/maps/dir/?api=1&origin=%.8f,%.8f&destination=%.8f,%.8f&travelmode=driving", x.ReferenceLat, x.ReferenceLon, x.EntranceLat, x.EntranceLon)
	x.UpdatedAt = x.GeneratedAt
	return x, nil
}
