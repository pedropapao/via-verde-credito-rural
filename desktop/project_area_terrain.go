package main

import (
	"database/sql"
	"encoding/json"
)

func (a *App) saveProjectAreaTerrain(areaID int64, terrain TerrainMetric) error {
	if a == nil || a.db == nil || areaID <= 0 || !terrain.Available {
		return nil
	}
	b, err := json.Marshal(terrain)
	if err != nil {
		return err
	}
	_, err = a.db.Exec(`INSERT INTO project_area_terrain(area_id,terrain_json) VALUES(?,?)
		ON CONFLICT(area_id) DO UPDATE SET terrain_json=excluded.terrain_json`, areaID, string(b))
	return err
}

func (a *App) loadProjectAreaTerrain(areaID int64) TerrainMetric {
	if a == nil || a.db == nil || areaID <= 0 {
		return TerrainMetric{}
	}
	var raw string
	if err := a.db.QueryRow(`SELECT terrain_json FROM project_area_terrain WHERE area_id=?`, areaID).Scan(&raw); err != nil {
		if err != sql.ErrNoRows {
			return TerrainMetric{}
		}
		return TerrainMetric{}
	}
	var out TerrainMetric
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
