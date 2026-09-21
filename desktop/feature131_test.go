package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearLastCARSessionRemovesOnlyTemporaryWorkspace(t *testing.T) {
	dir := t.TempDir()
	app := &App{dataDir: dir}
	cacheDir := filepath.Join(dir, "cache")
	tempDir := filepath.Join(cacheDir, "temporary")
	sicarDir := filepath.Join(cacheDir, "sicar", "1234567")
	if err := os.MkdirAll(tempDir, 0o755); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(sicarDir, 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(cacheDir, "last_car_session.json"), []byte(`{"saved_at":"2026-09-21T10:00:00-03:00"}`), 0o644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(tempDir, "GLEBA_TEMP.kml"), []byte("temp"), 0o644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(sicarDir, "app.zip"), []byte("cache"), 0o644); err != nil { t.Fatal(err) }

	if err := app.ClearLastCARSession(); err != nil { t.Fatal(err) }

	if _, err := os.Stat(filepath.Join(cacheDir, "last_car_session.json")); !os.IsNotExist(err) {
		t.Fatalf("sessão temporária deveria ter sido removida: %v", err)
	}
	entries, err := os.ReadDir(tempDir)
	if err != nil { t.Fatal(err) }
	if len(entries) != 0 {
		t.Fatalf("pasta temporária deveria estar vazia: %d item(ns)", len(entries))
	}
	if _, err := os.Stat(filepath.Join(sicarDir, "app.zip")); err != nil {
		t.Fatalf("cache SICAR permanente não deveria ser apagado: %v", err)
	}
}
