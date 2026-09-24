package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	worldCoverImageServer = "https://tiledimageservices.arcgis.com/P3ePLMYs2RVChkJx/arcgis/rest/services/European_Space_Agency_WorldCover_2021_Land_Cover_WGS84_7/ImageServer"
	worldCoverSourceURL    = "https://esa-worldcover.org/en/data-access"
	worldCoverYear         = 2021
)

type EnvironmentalLandCoverClass struct {
	Code       int     `json:"code"`
	Label      string  `json:"label"`
	Samples    int     `json:"samples"`
	Percent    float64 `json:"percent"`
	AreaHa     float64 `json:"area_ha"`
}

type EnvironmentalProfile struct {
	Biome             string                        `json:"biome"`
	BiomeSource       string                        `json:"biome_source"`
	BiomeAvailable    bool                          `json:"biome_available"`
	LandCoverAvailable bool                         `json:"land_cover_available"`
	LandCoverSource   string                        `json:"land_cover_source"`
	LandCoverYear     int                           `json:"land_cover_year"`
	LandCoverSamples  int                           `json:"land_cover_samples"`
	DominantLandCover string                        `json:"dominant_land_cover"`
	LandCoverClasses  []EnvironmentalLandCoverClass `json:"land_cover_classes"`
	Fire              EnvironmentalFireProfile       `json:"fire"`
	Hydrology         EnvironmentalHydrologyProfile  `json:"hydrology"`
	Warnings          []string                      `json:"warnings"`
}

type mapBiomasPointInformationResponse struct {
	Data struct {
		PointInformation struct {
			Territories []struct {
				Name         string  `json:"name"`
				CategoryName string  `json:"categoryName"`
				Code         string  `json:"code"`
				AreaHa       float64 `json:"areaHa"`
			} `json:"territories"`
		} `json:"pointInformation"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type arcGISWorldCoverSampleResponse struct {
	Samples []struct {
		Value string `json:"value"`
	} `json:"samples"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Details []string `json:"details"`
	} `json:"error"`
}

func (a *App) buildEnvironmentalProfile(ctx context.Context, car CARResult) EnvironmentalProfile {
	out := EnvironmentalProfile{
		BiomeSource:     "MapBiomas Alerta — territórios",
		LandCoverSource: "ESA WorldCover 2021 v200 — serviço público ArcGIS",
		LandCoverYear:   worldCoverYear,
		Fire:            newEnvironmentalFireProfile(),
		Hydrology:       newEnvironmentalHydrologyProfile(),
	}
	if strings.TrimSpace(car.GeoJSON) == "" {
		out.Warnings = append(out.Warnings, "Perfil ambiental: geometria do CAR indisponível.")
		return out
	}

	type biomeResult struct {
		name string
		err  error
	}
	type coverResult struct {
		classes []EnvironmentalLandCoverClass
		samples int
		err     error
	}
	type fireResult struct {
		profile EnvironmentalFireProfile
		err     error
	}
	type hydrologyResult struct {
		profile EnvironmentalHydrologyProfile
		err     error
	}
	biomeCh := make(chan biomeResult, 1)
	coverCh := make(chan coverResult, 1)
	fireCh := make(chan fireResult, 1)
	hydrologyCh := make(chan hydrologyResult, 1)

	go func() {
		name, err := a.queryBiomeAtCAR(ctx, car)
		biomeCh <- biomeResult{name: name, err: err}
	}()
	go func() {
		classes, samples, err := queryWorldCoverForCAR(ctx, car)
		coverCh <- coverResult{classes: classes, samples: samples, err: err}
	}()
	go func() {
		profile, err := queryFireProfile(ctx, car.GeoJSON)
		fireCh <- fireResult{profile: profile, err: err}
	}()
	go func() {
		profile, err := queryHydrologyProfile(ctx, car.GeoJSON)
		hydrologyCh <- hydrologyResult{profile: profile, err: err}
	}()

	b := <-biomeCh
	if b.err != nil {
		out.Warnings = append(out.Warnings, "Bioma: "+b.err.Error())
	} else if strings.TrimSpace(b.name) != "" {
		out.Biome = b.name
		out.BiomeAvailable = true
	}

	lc := <-coverCh
	if lc.err != nil {
		out.Warnings = append(out.Warnings, "Cobertura do solo: "+lc.err.Error())
	} else if lc.samples > 0 {
		out.LandCoverAvailable = true
		out.LandCoverSamples = lc.samples
		out.LandCoverClasses = lc.classes
		if len(lc.classes) > 0 {
			out.DominantLandCover = lc.classes[0].Label
		}
	}

	fire := <-fireCh
	out.Fire = fire.profile
	if fire.err != nil {
		out.Warnings = append(out.Warnings, "Focos de calor: "+fire.err.Error())
	}

	hydrology := <-hydrologyCh
	out.Hydrology = hydrology.profile
	if hydrology.err != nil {
		out.Warnings = append(out.Warnings, "Hidrografia: "+hydrology.err.Error())
	} else if strings.TrimSpace(hydrology.profile.Warning) != "" {
		out.Warnings = append(out.Warnings, "Hidrografia: "+hydrology.profile.Warning)
	}
	return out
}

func (a *App) queryBiomeAtCAR(ctx context.Context, car CARResult) (string, error) {
	token := strings.TrimSpace(a.getSetting("mapbiomas_alert_token"))
	if token == "" {
		return "", errors.New("MapBiomas Alerta não conectado")
	}
	lat, lon := carGeometryCenterFromGeoJSON(car.GeoJSON)
	if lat == 0 && lon == 0 {
		lat, lon = car.CenterLat, car.CenterLon
	}
	if lat == 0 && lon == 0 {
		return "", errors.New("centróide do CAR indisponível")
	}
	const d = 0.00005
	query := `query pointInformation($boundingBox: BoundingBoxInput) {
		pointInformation(boundingBox: $boundingBox) {
			territories { name categoryName code areaHa }
		}
	}`
	var resp mapBiomasPointInformationResponse
	errCh := make(chan error, 1)
	go func() {
		errCh <- mapBiomasGraphQL(token, graphQLRequest{
			Query: query,
			Variables: map[string]any{
				"boundingBox": map[string]float64{
					"swLat": lat - d, "swLng": lon - d,
					"neLat": lat + d, "neLng": lon + d,
				},
			},
		}, &resp)
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	}
	if len(resp.Errors) > 0 {
		return "", errors.New(joinGraphQLErrors(resp.Errors))
	}
	if name := biomeFromMapBiomasTerritories(resp.Data.PointInformation.Territories); name != "" {
		return name, nil
	}
	return "", errors.New("bioma não retornado para o ponto central do CAR")
}

func biomeFromMapBiomasTerritories(items []struct {
	Name         string  `json:"name"`
	CategoryName string  `json:"categoryName"`
	Code         string  `json:"code"`
	AreaHa       float64 `json:"areaHa"`
}) string {
	for _, t := range items {
		cat := strings.ToLower(strings.TrimSpace(t.CategoryName))
		if strings.Contains(cat, "bioma") {
			return strings.TrimSpace(t.Name)
		}
	}
	return ""
}

func worldCoverClassesFromCounts(counts map[int]int, valid int, areaHa float64) []EnvironmentalLandCoverClass {
	if valid <= 0 {
		return nil
	}
	classes := make([]EnvironmentalLandCoverClass, 0, len(counts))
	for code, n := range counts {
		label, ok := worldCoverLabel(code)
		if !ok || n <= 0 {
			continue
		}
		pct := float64(n) / float64(valid) * 100
		classes = append(classes, EnvironmentalLandCoverClass{
			Code: code, Label: label, Samples: n,
			Percent: pct, AreaHa: areaHa * pct / 100,
		})
	}
	sort.SliceStable(classes, func(i, j int) bool {
		if classes[i].Samples == classes[j].Samples {
			return classes[i].Code < classes[j].Code
		}
		return classes[i].Samples > classes[j].Samples
	})
	return classes
}

func carGeometryCenterFromGeoJSON(raw string) (lat, lon float64) {
	var f carGeoFeature
	if json.Unmarshal([]byte(raw), &f) != nil {
		return 0, 0
	}
	return carGeometryCenter(f.Geometry)
}

func queryWorldCoverForCAR(ctx context.Context, car CARResult) ([]EnvironmentalLandCoverClass, int, error) {
	points, err := worldCoverSamplePoints(car.GeoJSON, 900)
	if err != nil {
		return nil, 0, err
	}
	if len(points) < 20 {
		return nil, 0, errors.New("poucos pontos internos para estimar a cobertura")
	}
	counts := map[int]int{}
	valid := 0
	const chunkSize = 170
	for start := 0; start < len(points); start += chunkSize {
		end := start + chunkSize
		if end > len(points) {
			end = len(points)
		}
		vals, err := queryWorldCoverSamples(ctx, points[start:end])
		if err != nil {
			return nil, valid, err
		}
		for _, code := range vals {
			if _, ok := worldCoverLabel(code); !ok {
				continue
			}
			counts[code]++
			valid++
		}
	}
	if valid == 0 {
		return nil, 0, errors.New("serviço WorldCover não retornou amostras válidas")
	}
	areaHa := firstPositive(car.GeometryAreaHa, car.AreaHa)
	return worldCoverClassesFromCounts(counts, valid, areaHa), valid, nil
}

func worldCoverSamplePoints(raw string, limit int) ([][2]float64, error) {
	if limit < 50 {
		limit = 50
	}
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(raw)
	if !ok || maxLon <= minLon || maxLat <= minLat {
		return nil, errors.New("limites do CAR inválidos")
	}
	lat0 := sharedLatitude(raw)
	mp, err := geoJSONToPlanar(raw, lat0)
	if err != nil || len(mp) == 0 {
		return nil, errors.New("geometria do CAR não pôde ser preparada para amostragem")
	}
	const earthR = 6371008.8
	cos0 := math.Cos(lat0 * math.Pi / 180)
	grid := 52
	all := make([][2]float64, 0, grid*grid)
	for iy := 0; iy < grid; iy++ {
		lat := minLat + (float64(iy)+0.5)/float64(grid)*(maxLat-minLat)
		for ix := 0; ix < grid; ix++ {
			lon := minLon + (float64(ix)+0.5)/float64(grid)*(maxLon-minLon)
			p := planarPoint{
				X: earthR * lon * math.Pi / 180 * cos0,
				Y: earthR * lat * math.Pi / 180,
			}
			if pointInMultiPolygon(p, mp) {
				all = append(all, [2]float64{lon, lat})
			}
		}
	}
	if len(all) <= limit {
		return all, nil
	}
	out := make([][2]float64, 0, limit)
	step := float64(len(all)) / float64(limit)
	for i := 0; i < limit; i++ {
		idx := int(math.Floor(float64(i) * step))
		if idx >= len(all) {
			idx = len(all) - 1
		}
		out = append(out, all[idx])
	}
	return out, nil
}

func queryWorldCoverSamples(ctx context.Context, points [][2]float64) ([]int, error) {
	if len(points) == 0 {
		return nil, nil
	}
	geometry := struct {
		Points [][2]float64 `json:"points"`
		SpatialReference map[string]int `json:"spatialReference"`
	}{
		Points: points,
		SpatialReference: map[string]int{"wkid": 4326},
	}
	gb, _ := json.Marshal(geometry)
	q := url.Values{}
	q.Set("geometryType", "esriGeometryMultipoint")
	q.Set("geometry", string(gb))
	q.Set("interpolation", "RSP_NearestNeighbor")
	q.Set("returnFirstValueOnly", "true")
	q.Set("f", "json")
	target := worldCoverImageServer + "/getSamples"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(q.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	resp, err := (&http.Client{Timeout: 25 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WorldCover respondeu HTTP %d", resp.StatusCode)
	}
	var data arcGISWorldCoverSampleResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.Error != nil {
		msg := strings.TrimSpace(data.Error.Message)
		if len(data.Error.Details) > 0 {
			msg += ": " + strings.Join(data.Error.Details, "; ")
		}
		return nil, fmt.Errorf("WorldCover/ArcGIS: %s", strings.TrimSpace(msg))
	}
	out := make([]int, 0, len(data.Samples))
	for _, s := range data.Samples {
		raw := strings.TrimSpace(strings.Split(s.Value, ",")[0])
		if raw == "" || strings.EqualFold(raw, "NoData") {
			continue
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		out = append(out, int(math.Round(v)))
	}
	return out, nil
}

func worldCoverLabel(code int) (string, bool) {
	labels := map[int]string{
		10:  "Cobertura arbórea",
		20:  "Arbustos",
		30:  "Vegetação herbácea (inclui pastagens)",
		40:  "Cultivos anuais",
		50:  "Área construída",
		60:  "Solo exposto ou vegetação esparsa",
		70:  "Neve e gelo",
		80:  "Água permanente",
		90:  "Área úmida herbácea",
		95:  "Manguezal",
		100: "Musgos e líquens",
	}
	v, ok := labels[code]
	return v, ok
}
