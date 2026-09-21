package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type RouteStep struct {
	Instruction string  `json:"instruction"`
	Road        string  `json:"road"`
	DistanceKm  float64 `json:"distance_km"`
	DurationMin float64 `json:"duration_min"`
}

type AccessRoute struct {
	PropertyID                  int64       `json:"property_id"`
	ReferenceLabel              string      `json:"reference_label"`
	ReferenceLat                float64     `json:"reference_lat"`
	ReferenceLon                float64     `json:"reference_lon"`
	EntranceLat                 float64     `json:"entrance_lat"`
	EntranceLon                 float64     `json:"entrance_lon"`
	HeadquartersLat             float64     `json:"headquarters_lat"`
	HeadquartersLon             float64     `json:"headquarters_lon"`
	ReferenceToEntranceKm       float64     `json:"reference_to_entrance_km"`
	EntranceToHeadquartersKm    float64     `json:"entrance_to_headquarters_km"`
	RouteDistanceKm             float64     `json:"route_distance_km"`
	RouteDurationMin            float64     `json:"route_duration_min"`
	RouteGeoJSON                string      `json:"route_geojson"`
	Steps                       []RouteStep `json:"steps"`
	RouteSource                 string      `json:"route_source"`
	Automatic                   bool        `json:"automatic"`
	GeneratedAt                 string      `json:"generated_at"`
	Notes                       string      `json:"notes"`
	Text                        string      `json:"text"`
	GoogleMapsURL               string      `json:"google_maps_url"`
	UpdatedAt                   string      `json:"updated_at"`
}

func (a *App) GetAccessRoute(propertyID int64) (AccessRoute, error) {
	if a.db == nil {
		return AccessRoute{}, errors.New("banco local indisponível")
	}
	if propertyID <= 0 {
		return AccessRoute{}, errors.New("imóvel inválido")
	}
	// A rota automática passa a ser a referência principal. Se ainda não existir,
	// mantemos compatibilidade com o roteiro manual da 1.0.9.
	if auto, err := a.loadAutomaticRoute(propertyID); err == nil && auto.EntranceLat != 0 {
		return auto, nil
	}
	var x AccessRoute
	err := a.db.QueryRow(`SELECT property_id,reference_label,reference_lat,reference_lon,entrance_lat,entrance_lon,headquarters_lat,headquarters_lon,reference_to_entrance_km,entrance_to_headquarters_km,notes,updated_at
		FROM access_routes WHERE property_id=?`, propertyID).
		Scan(&x.PropertyID, &x.ReferenceLabel, &x.ReferenceLat, &x.ReferenceLon, &x.EntranceLat, &x.EntranceLon, &x.HeadquartersLat, &x.HeadquartersLon, &x.ReferenceToEntranceKm, &x.EntranceToHeadquartersKm, &x.Notes, &x.UpdatedAt)
	if err != nil {
		return AccessRoute{PropertyID: propertyID}, nil
	}
	x.Text = buildAccessRouteText(x)
	x.GoogleMapsURL = accessRouteMapsURL(x)
	return x, nil
}

func (a *App) SaveAccessRoute(x AccessRoute) (AccessRoute, error) {
	if a.db == nil {
		return AccessRoute{}, errors.New("banco local indisponível")
	}
	if x.PropertyID <= 0 {
		return AccessRoute{}, errors.New("selecione um imóvel")
	}
	if !validCoordinatePair(x.EntranceLat, x.EntranceLon) {
		return AccessRoute{}, errors.New("marque pelo menos a entrada da propriedade no mapa")
	}
	x.ReferenceLabel = strings.TrimSpace(x.ReferenceLabel)
	x.Notes = strings.TrimSpace(x.Notes)
	now := time.Now().Format(time.RFC3339)
	_, err := a.db.Exec(`INSERT INTO access_routes(property_id,reference_label,reference_lat,reference_lon,entrance_lat,entrance_lon,headquarters_lat,headquarters_lon,reference_to_entrance_km,entrance_to_headquarters_km,notes,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(property_id) DO UPDATE SET reference_label=excluded.reference_label,reference_lat=excluded.reference_lat,reference_lon=excluded.reference_lon,entrance_lat=excluded.entrance_lat,entrance_lon=excluded.entrance_lon,headquarters_lat=excluded.headquarters_lat,headquarters_lon=excluded.headquarters_lon,reference_to_entrance_km=excluded.reference_to_entrance_km,entrance_to_headquarters_km=excluded.entrance_to_headquarters_km,notes=excluded.notes,updated_at=excluded.updated_at`,
		x.PropertyID, x.ReferenceLabel, x.ReferenceLat, x.ReferenceLon, x.EntranceLat, x.EntranceLon, x.HeadquartersLat, x.HeadquartersLon, x.ReferenceToEntranceKm, x.EntranceToHeadquartersKm, x.Notes, now)
	if err != nil {
		return AccessRoute{}, err
	}
	x.UpdatedAt = now
	x.Text = buildAccessRouteText(x)
	x.GoogleMapsURL = accessRouteMapsURL(x)
	return x, nil
}

func (a *App) ExportAccessRouteTXT(propertyID int64) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	x, err := a.GetAccessRoute(propertyID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(x.Text) == "" {
		return "", errors.New("cadastre o roteiro de acesso antes de exportar")
	}
	p, _ := a.GetProperty(propertyID)
	name := "Roteiro_de_Acesso_" + safeFilePart(p.Name) + ".txt"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Salvar roteiro de acesso", DefaultFilename: name,
		Filters: []runtime.FileFilter{{DisplayName: "Arquivo de texto", Pattern: "*.txt"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	content := "VIA VERDE CAR — ROTEIRO DE ACESSO\r\n\r\n" + x.Text + "\r\n"
	if x.RouteDistanceKm > 0 {
		content += fmt.Sprintf("\r\nDistância pela rota: %.2f km\r\nTempo estimado: %.0f min\r\n", x.RouteDistanceKm, x.RouteDurationMin)
	}
	if len(x.Steps) > 0 {
		content += "\r\nINSTRUÇÕES DA ROTA\r\n"
		for i, step := range x.Steps {
			content += fmt.Sprintf("%d. %s", i+1, step.Instruction)
			if step.Road != "" {
				content += " — " + step.Road
			}
			if step.DistanceKm > 0 {
				content += fmt.Sprintf(" (%.2f km)", step.DistanceKm)
			}
			content += "\r\n"
		}
	}
	if x.RouteSource != "" {
		content += "\r\nFonte de roteamento: " + x.RouteSource + "\r\n"
	}
	if x.Notes != "" {
		content += "\r\nObservações: " + x.Notes + "\r\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func buildAccessRouteText(x AccessRoute) string {
	if !validCoordinatePair(x.EntranceLat, x.EntranceLon) {
		return ""
	}
	start := "Partindo do ponto de referência informado"
	if strings.TrimSpace(x.ReferenceLabel) != "" {
		start = "Partindo de " + strings.TrimSpace(x.ReferenceLabel)
	}
	var b strings.Builder
	b.WriteString(start)
	if validCoordinatePair(x.ReferenceLat, x.ReferenceLon) {
		fmt.Fprintf(&b, " (%.6f, %.6f)", x.ReferenceLat, x.ReferenceLon)
	}
	fmt.Fprintf(&b, ", seguir até a entrada da propriedade, localizada nas coordenadas %.6f, %.6f", x.EntranceLat, x.EntranceLon)

	dist1 := x.ReferenceToEntranceKm
	if dist1 <= 0 && validCoordinatePair(x.ReferenceLat, x.ReferenceLon) {
		dist1 = carHaversineM(x.ReferenceLat, x.ReferenceLon, x.EntranceLat, x.EntranceLon) / 1000
		if dist1 > 0 {
			fmt.Fprintf(&b, " (distância aproximada em linha reta de %.2f km)", dist1)
		}
	} else if dist1 > 0 {
		fmt.Fprintf(&b, " (aproximadamente %.2f km pelo acesso informado)", dist1)
	}
	b.WriteString(".")

	if validCoordinatePair(x.HeadquartersLat, x.HeadquartersLon) {
		fmt.Fprintf(&b, " A sede/ponto principal do imóvel está nas coordenadas %.6f, %.6f", x.HeadquartersLat, x.HeadquartersLon)
		dist2 := x.EntranceToHeadquartersKm
		if dist2 <= 0 {
			dist2 = carHaversineM(x.EntranceLat, x.EntranceLon, x.HeadquartersLat, x.HeadquartersLon) / 1000
			if dist2 > 0 {
				fmt.Fprintf(&b, ", a aproximadamente %.2f km em linha reta da entrada", dist2)
			}
		} else {
			fmt.Fprintf(&b, ", a aproximadamente %.2f km pelo acesso interno informado", dist2)
		}
		b.WriteString(".")
	}
	return b.String()
}

func accessRouteMapsURL(x AccessRoute) string {
	lat, lon := x.HeadquartersLat, x.HeadquartersLon
	if !validCoordinatePair(lat, lon) {
		lat, lon = x.EntranceLat, x.EntranceLon
	}
	if !validCoordinatePair(lat, lon) {
		return ""
	}
	return fmt.Sprintf("https://www.google.com/maps?q=%.8f,%.8f", lat, lon)
}

func validCoordinatePair(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 && (lat != 0 || lon != 0)
}
