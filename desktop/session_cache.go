package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CARSessionCache struct {
	SavedAt       string       `json:"saved_at"`
	Result        CARResult    `json:"result"`
	TemporaryArea *ProjectArea `json:"temporary_area,omitempty"`
}

func (a *App) saveLastCARSession(result CARResult) {
	if a == nil || a.dataDir == "" || !result.Found || result.GeoJSON == "" {
		return
	}
	path := filepath.Join(a.dataDir, "cache", "last_car_session.json")
	cache := CARSessionCache{SavedAt: time.Now().Format(time.RFC3339), Result: result}
	b, err := json.Marshal(cache)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o644)
}

func (a *App) GetLastCARSession() (CARSessionCache, error) {
	if a == nil || a.dataDir == "" {
		return CARSessionCache{}, errors.New("cache local indisponível")
	}
	path := filepath.Join(a.dataDir, "cache", "last_car_session.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return CARSessionCache{}, err
	}
	var out CARSessionCache
	if err := json.Unmarshal(b, &out); err != nil {
		return CARSessionCache{}, err
	}
	t, err := time.Parse(time.RFC3339, out.SavedAt)
	if err != nil || time.Since(t) > 24*time.Hour {
		_ = os.Remove(path)
		return CARSessionCache{}, errors.New("sessão temporária expirada")
	}
	return out, nil
}

func (a *App) saveTemporaryProjectArea(area ProjectArea) error {
	if a == nil || a.dataDir == "" {
		return errors.New("cache local indisponível")
	}
	cache, err := a.GetLastCARSession()
	if err != nil {
		return err
	}
	cache.SavedAt = time.Now().Format(time.RFC3339)
	cache.TemporaryArea = &area
	path := filepath.Join(a.dataDir, "cache", "last_car_session.json")
	b, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (a *App) GetTemporaryProjectArea() (ProjectArea, error) {
	cache, err := a.GetLastCARSession()
	if err != nil {
		return ProjectArea{}, err
	}
	if cache.TemporaryArea == nil || strings.TrimSpace(cache.TemporaryArea.GeoJSON) == "" {
		return ProjectArea{}, errors.New("nenhuma gleba temporária nesta consulta")
	}
	return *cache.TemporaryArea, nil
}

func (a *App) ClearLastCARSession() error {
	if a == nil || a.dataDir == "" {
		return nil
	}
	path := filepath.Join(a.dataDir, "cache", "last_car_session.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
