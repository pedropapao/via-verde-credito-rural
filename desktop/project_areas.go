package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ProjectArea struct {
	ID             int64   `json:"id"`
	PropertyID     int64   `json:"property_id"`
	Name           string  `json:"name"`
	Purpose        string  `json:"purpose"`
	AreaHa         float64 `json:"area_ha"`
	PerimeterM     float64 `json:"perimeter_m"`
	CenterLat      float64 `json:"center_lat"`
	CenterLon      float64 `json:"center_lon"`
	InsideCARPct   float64 `json:"inside_car_pct"`
	GeoJSON        string  `json:"geojson"`
	KMLPath        string  `json:"kml_path"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string        `json:"updated_at"`
	Terrain        TerrainMetric `json:"terrain"`
}

func (a *App) ListProjectAreas(propertyID int64) ([]ProjectArea, error) {
	if a.db == nil {
		return nil, errors.New("banco local indisponível")
	}
	if propertyID <= 0 {
		return nil, errors.New("imóvel inválido")
	}
	rows, err := a.db.Query(`SELECT id,property_id,name,purpose,area_ha,perimeter_m,center_lat,center_lon,inside_car_pct,geojson,kml_path,created_at,updated_at
		FROM project_areas WHERE property_id=? ORDER BY updated_at DESC,id DESC`, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectArea
	for rows.Next() {
		var x ProjectArea
		if err := rows.Scan(&x.ID, &x.PropertyID, &x.Name, &x.Purpose, &x.AreaHa, &x.PerimeterM, &x.CenterLat, &x.CenterLon, &x.InsideCARPct, &x.GeoJSON, &x.KMLPath, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		x.Terrain = a.loadProjectAreaTerrain(x.ID)
		out = append(out, x)
	}
	return out, rows.Err()
}

func (a *App) SaveProjectArea(area ProjectArea) (ProjectArea, error) {
	if a.db == nil {
		return ProjectArea{}, errors.New("banco local indisponível")
	}
	if area.PropertyID <= 0 {
		return ProjectArea{}, errors.New("selecione um imóvel")
	}
	area.Name = strings.TrimSpace(area.Name)
	area.Purpose = strings.TrimSpace(area.Purpose)
	if area.Name == "" {
		return ProjectArea{}, errors.New("informe o nome da área/gleba")
	}
	metric, err := projectAreaMetrics(area.GeoJSON)
	if err != nil {
		return ProjectArea{}, err
	}
	area.AreaHa = metric.AreaHa
	area.PerimeterM = metric.PerimeterM
	area.CenterLat = metric.CenterLat
	area.CenterLon = metric.CenterLon
	area.GeoJSON = metric.GeoJSON

	if latest, err := a.GetLatestCAR(area.PropertyID); err == nil && strings.TrimSpace(latest.GeoJSON) != "" {
		_, inside, _, cmpErr := estimateGeometryOverlap(area.GeoJSON, latest.GeoJSON)
		if cmpErr == nil {
			area.InsideCARPct = inside
		}
	}

	now := time.Now().Format(time.RFC3339)
	if area.ID == 0 {
		res, err := a.db.Exec(`INSERT INTO project_areas(property_id,name,purpose,area_ha,perimeter_m,center_lat,center_lon,inside_car_pct,geojson,kml_path,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			area.PropertyID, area.Name, area.Purpose, area.AreaHa, area.PerimeterM, area.CenterLat, area.CenterLon, area.InsideCARPct, area.GeoJSON, "", now, now)
		if err != nil {
			return ProjectArea{}, err
		}
		area.ID, _ = res.LastInsertId()
		area.CreatedAt = now
	} else {
		_, err := a.db.Exec(`UPDATE project_areas SET name=?,purpose=?,area_ha=?,perimeter_m=?,center_lat=?,center_lon=?,inside_car_pct=?,geojson=?,updated_at=? WHERE id=? AND property_id=?`,
			area.Name, area.Purpose, area.AreaHa, area.PerimeterM, area.CenterLat, area.CenterLon, area.InsideCARPct, area.GeoJSON, now, area.ID, area.PropertyID)
		if err != nil {
			return ProjectArea{}, err
		}
	}
	area.UpdatedAt = now
	if path, err := a.writeProjectAreaKML(area); err == nil {
		area.KMLPath = path
		_, _ = a.db.Exec(`UPDATE project_areas SET kml_path=? WHERE id=?`, path, area.ID)
	}
	return area, nil
}

func (a *App) ImportProjectAreaKML(propertyID int64, name, purpose string) (ProjectArea, error) {
	if a.ctx == nil {
		return ProjectArea{}, errors.New("aplicativo ainda não inicializado")
	}
	if propertyID <= 0 {
		return ProjectArea{}, errors.New("selecione um imóvel")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Selecionar KML da área do projeto",
		Filters: []runtime.FileFilter{{DisplayName: "Google Earth KML", Pattern: "*.kml"}},
	})
	if err != nil {
		return ProjectArea{}, err
	}
	if strings.TrimSpace(path) == "" {
		return ProjectArea{}, errors.New("importação cancelada")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectArea{}, err
	}
	if len(data) > 25<<20 {
		return ProjectArea{}, errors.New("KML maior que 25 MB")
	}
	kml, err := analyzeKML(data)
	if err != nil {
		return ProjectArea{}, err
	}
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return a.SaveProjectArea(ProjectArea{
		PropertyID: propertyID,
		Name:       name,
		Purpose:    purpose,
		GeoJSON:    kml.GeoJSON,
	})
}

func (a *App) DeleteProjectArea(id int64) error {
	if a.db == nil {
		return errors.New("banco local indisponível")
	}
	if id <= 0 {
		return errors.New("área inválida")
	}
	var path string
	_ = a.db.QueryRow(`SELECT kml_path FROM project_areas WHERE id=?`, id).Scan(&path)
	if _, err := a.db.Exec(`DELETE FROM project_areas WHERE id=?`, id); err != nil {
		return err
	}
	if strings.TrimSpace(path) != "" {
		_ = os.Remove(path)
	}
	return nil
}

func (a *App) ExportProjectAreaKML(id int64) (string, error) {
	if a.db == nil {
		return "", errors.New("banco local indisponível")
	}
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	var area ProjectArea
	err := a.db.QueryRow(`SELECT id,property_id,name,purpose,area_ha,perimeter_m,center_lat,center_lon,inside_car_pct,geojson,kml_path,created_at,updated_at FROM project_areas WHERE id=?`, id).
		Scan(&area.ID, &area.PropertyID, &area.Name, &area.Purpose, &area.AreaHa, &area.PerimeterM, &area.CenterLat, &area.CenterLon, &area.InsideCARPct, &area.GeoJSON, &area.KMLPath, &area.CreatedAt, &area.UpdatedAt)
	if err != nil {
		return "", err
	}
	var f carGeoFeature
	if err := json.Unmarshal([]byte(area.GeoJSON), &f); err != nil || !carGeometryUsable(f.Geometry) {
		return "", errors.New("geometria da área inválida")
	}
	data, err := carGeometryKML(area.Name, f.Geometry)
	if err != nil {
		return "", err
	}
	name := "GLEBA_" + safeFilePart(area.Name) + ".kml"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Salvar KML da área do projeto", DefaultFilename: name,
		Filters: []runtime.FileFilter{{DisplayName: "Google Earth KML", Pattern: "*.kml"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func projectAreaMetrics(raw string) (ProjectArea, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ProjectArea{}, errors.New("desenhe ou importe uma geometria para a área do projeto")
	}
	var f carGeoFeature
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return ProjectArea{}, errors.New("geometria da área do projeto inválida")
	}
	if !carGeometryUsable(f.Geometry) {
		return ProjectArea{}, errors.New("a área do projeto precisa ser um polígono válido")
	}
	area := carGeometryAreaHa(f.Geometry)
	if area <= 0 {
		return ProjectArea{}, errors.New("não foi possível calcular a área da gleba")
	}
	b, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: map[string]any{"source": "Via Verde CAR - área do projeto"}, Geometry: f.Geometry})
	lat, lon := carGeometryCenter(f.Geometry)
	return ProjectArea{
		AreaHa:     area,
		PerimeterM: carGeometryPerimeterM(f.Geometry),
		CenterLat:  lat,
		CenterLon:  lon,
		GeoJSON:    string(b),
	}, nil
}

func (a *App) writeProjectAreaKML(area ProjectArea) (string, error) {
	var f carGeoFeature
	if err := json.Unmarshal([]byte(area.GeoJSON), &f); err != nil {
		return "", err
	}
	p, err := a.GetProperty(area.PropertyID)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(a.dataDir, "properties", fmt.Sprint(area.PropertyID), "areas")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := "GLEBA_" + safeFilePart(p.Name) + "_" + safeFilePart(area.Name) + ".kml"
	path := filepath.Join(dir, name)
	data, err := carGeometryKML(area.Name, f.Geometry)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
