package main

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

func TestEnvironmentalUnavailableDoesNotCreateFalseHistory194(t *testing.T) {
	oldR := CARResult{
		CAR: "MG-3100000-AAAA.BBBB.CCCC.DDDD.EEEE.FFFF.0000.1111",
		Status: "Ativo",
		AreaHa: 50,
		Environment: EnvironmentalSummary{
			IBAMAChecked:      true,
			IBAMAEmbargoCount: 1,
			FUNAIChecked:      true,
			IndigenousCount:   0,
			ICMBioChecked:     true,
			FederalUCCount:    0,
			MCRChecked:        true,
			MCRListed:         false,
		},
	}
	current := oldR
	current.Environment.IBAMAChecked = false
	current.Environment.IBAMAEmbargoCount = 0

	if carResultMateriallyChanged(marshalJSON(oldR), current) {
		t.Fatal("indisponibilidade do IBAMA não pode virar mudança material 1 → 0")
	}
	changes := strings.Join(compareCARHistoryJSON(marshalJSON(oldR), marshalJSON(current)), " | ")
	if strings.Contains(changes, "Embargos IBAMA") {
		t.Fatalf("histórico registrou falso desaparecimento do embargo: %s", changes)
	}

	current = oldR
	current.Environment.IBAMAEmbargoCount = 0
	if !carResultMateriallyChanged(marshalJSON(oldR), current) {
		t.Fatal("mudança 1 → 0 com IBAMA consultado nas duas execuções deve ser material")
	}
	changes = strings.Join(compareCARHistoryJSON(marshalJSON(oldR), marshalJSON(current)), " | ")
	if !strings.Contains(changes, "Embargos IBAMA: 1 → 0") {
		t.Fatalf("mudança ambiental confirmada não apareceu no histórico: %s", changes)
	}
}

func TestConfigureSQLiteKeepsForeignKeysAndSingleConnection194(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/integrity.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := configureSQLite(db); err != nil {
		t.Fatal(err)
	}
	if got := db.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("esperava uma única conexão SQLite, recebeu %d", got)
	}

	var foreignKeys, busyTimeout int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys deveria estar ativo, recebeu %d", foreignKeys)
	}
	if busyTimeout < 5000 {
		t.Fatalf("busy_timeout deveria ser >= 5000 ms, recebeu %d", busyTimeout)
	}

	if _, err := db.Exec(`
		CREATE TABLE parent(id INTEGER PRIMARY KEY);
		CREATE TABLE child(
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL,
			FOREIGN KEY(parent_id) REFERENCES parent(id) ON DELETE CASCADE
		);
		INSERT INTO parent(id) VALUES(1);
		INSERT INTO child(id,parent_id) VALUES(1,1);
		DELETE FROM parent WHERE id=1;
	`); err != nil {
		t.Fatal(err)
	}
	var children int
	if err := db.QueryRow("SELECT COUNT(*) FROM child").Scan(&children); err != nil {
		t.Fatal(err)
	}
	if children != 0 {
		t.Fatalf("ON DELETE CASCADE não funcionou; sobraram %d registro(s)", children)
	}
}

func TestPersistCARAnalysisRollsBackWhenHistoryInsertFails194(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/rollback.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := configureSQLite(db); err != nil {
		t.Fatal(err)
	}

	// car_checks é propositalmente incompleta: a leitura do histórico funciona,
	// mas o INSERT completo falha depois do UPDATE de properties.
	if _, err := db.Exec(`
		CREATE TABLE properties(
			id INTEGER PRIMARY KEY,
			car_number TEXT NOT NULL DEFAULT '',
			last_car_json TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE car_checks(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			property_id INTEGER NOT NULL,
			checked_at TEXT NOT NULL DEFAULT '',
			result_json TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO properties(id,car_number,last_car_json,updated_at)
		VALUES(1,'CAR-ANTIGO','ORIGINAL','2026-09-24T00:00:00Z');
	`); err != nil {
		t.Fatal(err)
	}

	app := &App{db: db}
	result := CARResult{
		CAR:       "MG-3100000-AAAA.BBBB.CCCC.DDDD.EEEE.FFFF.0000.1111",
		Found:     true,
		Status:    "Ativo",
		CheckedAt: time.Now().Format(time.RFC3339),
	}
	err = app.persistCARAnalysis(1, result.CAR, &result)
	if err == nil {
		t.Fatal("esperava falha no INSERT do histórico")
	}

	var carNumber, lastRaw string
	if err := db.QueryRow("SELECT car_number,last_car_json FROM properties WHERE id=1").Scan(&carNumber, &lastRaw); err != nil {
		t.Fatal(err)
	}
	if carNumber != "CAR-ANTIGO" || lastRaw != "ORIGINAL" {
		t.Fatalf("UPDATE deveria ter sido revertido; car=%q last=%q", carNumber, lastRaw)
	}
}

func TestPersistCARAnalysisCommitsPropertyAndHistoryTogether194(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/commit.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := configureSQLite(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE properties(
			id INTEGER PRIMARY KEY,
			car_number TEXT NOT NULL DEFAULT '',
			last_car_json TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE car_checks(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			property_id INTEGER NOT NULL,
			car_number TEXT NOT NULL,
			checked_at TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT '',
			condition_text TEXT NOT NULL DEFAULT '',
			area_ha REAL NOT NULL DEFAULT 0,
			municipality TEXT NOT NULL DEFAULT '',
			geometry_json TEXT NOT NULL DEFAULT '',
			result_json TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO properties(id) VALUES(1);
	`); err != nil {
		t.Fatal(err)
	}

	app := &App{db: db}
	result := CARResult{
		CAR:       "MG-3100000-AAAA.BBBB.CCCC.DDDD.EEEE.FFFF.0000.1111",
		Found:     true,
		Status:    "Ativo",
		AreaHa:    42.5,
		CheckedAt: time.Now().Format(time.RFC3339),
	}
	if err := app.persistCARAnalysis(1, result.CAR, &result); err != nil {
		t.Fatal(err)
	}
	if !result.SnapshotSaved {
		t.Fatal("primeira consulta deveria criar snapshot")
	}

	var propertyRaw string
	var checks int
	if err := db.QueryRow("SELECT last_car_json FROM properties WHERE id=1").Scan(&propertyRaw); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM car_checks WHERE property_id=1").Scan(&checks); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(propertyRaw) == "" || checks != 1 {
		t.Fatalf("gravação transacional incompleta: last_car_json=%q checks=%d", propertyRaw, checks)
	}
}
