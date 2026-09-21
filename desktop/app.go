package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	_ "modernc.org/sqlite"
)

const AppVersion = "1.0.9"

type App struct {
	ctx     context.Context
	db      *sql.DB
	dataDir string
	mu      sync.Mutex
}

type Client struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CPFCNPJ   string `json:"cpf_cnpj"`
	Phone     string `json:"phone"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Property struct {
	ID             int64   `json:"id"`
	ClientID       int64   `json:"client_id"`
	ClientName     string  `json:"client_name"`
	Name           string  `json:"name"`
	Municipality   string  `json:"municipality"`
	UF             string  `json:"uf"`
	Registry       string  `json:"registry"`
	CARNumber      string  `json:"car_number"`
	DeclaredAreaHa float64 `json:"declared_area_ha"`
	KMLPath        string  `json:"kml_path"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type DashboardSummary struct {
	Clients           int `json:"clients"`
	Properties        int `json:"properties"`
	PropertiesWithCAR int `json:"properties_with_car"`
	PropertiesWithKML int `json:"properties_with_kml"`
	ChecksLast30Days  int `json:"checks_last_30_days"`
}

type BackupResult struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type AppInfo struct {
	Version string `json:"version"`
	DataDir string `json:"data_dir"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.openDatabase(); err != nil {
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Via Verde CAR",
			Message: "Não foi possível abrir a base local: " + err.Error(),
		})
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
}

func (a *App) openDatabase() error {
	root, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	a.dataDir = filepath.Join(root, "ViaVerdeCAR")
	for _, dir := range []string{"properties", "backups", "updates", "cache"} {
		if err := os.MkdirAll(filepath.Join(a.dataDir, dir), 0o755); err != nil {
			return err
		}
	}
	dbPath := filepath.Join(a.dataDir, "viaverde.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;`); err != nil {
		db.Close()
		return err
	}
	a.db = db
	return a.migrate()
}

func (a *App) migrate() error {
	if a.db == nil {
		return errors.New("banco local indisponível")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL);`,
		`INSERT INTO schema_version(version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_version);`,
		`CREATE TABLE IF NOT EXISTS clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			cpf_cnpj TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			notes TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS properties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			municipality TEXT NOT NULL DEFAULT '',
			uf TEXT NOT NULL DEFAULT '',
			registry TEXT NOT NULL DEFAULT '',
			car_number TEXT NOT NULL DEFAULT '',
			declared_area_ha REAL NOT NULL DEFAULT 0,
			kml_path TEXT NOT NULL DEFAULT '',
			last_car_json TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(client_id) REFERENCES clients(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_properties_client ON properties(client_id);`,
		`CREATE INDEX IF NOT EXISTS idx_properties_car ON properties(car_number);`,
		`CREATE TABLE IF NOT EXISTS car_checks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			property_id INTEGER NOT NULL,
			car_number TEXT NOT NULL,
			checked_at TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT '',
			condition_text TEXT NOT NULL DEFAULT '',
			area_ha REAL NOT NULL DEFAULT 0,
			municipality TEXT NOT NULL DEFAULT '',
			geometry_json TEXT NOT NULL DEFAULT '',
			result_json TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(property_id) REFERENCES properties(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_car_checks_property ON car_checks(property_id, checked_at DESC);`,
		`CREATE TABLE IF NOT EXISTS project_areas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			property_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			purpose TEXT NOT NULL DEFAULT '',
			area_ha REAL NOT NULL DEFAULT 0,
			perimeter_m REAL NOT NULL DEFAULT 0,
			center_lat REAL NOT NULL DEFAULT 0,
			center_lon REAL NOT NULL DEFAULT 0,
			inside_car_pct REAL NOT NULL DEFAULT 0,
			geojson TEXT NOT NULL DEFAULT '',
			kml_path TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(property_id) REFERENCES properties(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_project_areas_property ON project_areas(property_id, updated_at DESC);`,
		`CREATE TABLE IF NOT EXISTS access_routes (
			property_id INTEGER PRIMARY KEY,
			reference_label TEXT NOT NULL DEFAULT '',
			reference_lat REAL NOT NULL DEFAULT 0,
			reference_lon REAL NOT NULL DEFAULT 0,
			entrance_lat REAL NOT NULL DEFAULT 0,
			entrance_lon REAL NOT NULL DEFAULT 0,
			headquarters_lat REAL NOT NULL DEFAULT 0,
			headquarters_lon REAL NOT NULL DEFAULT 0,
			reference_to_entrance_km REAL NOT NULL DEFAULT 0,
			entrance_to_headquarters_km REAL NOT NULL DEFAULT 0,
			notes TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			FOREIGN KEY(property_id) REFERENCES properties(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
		`UPDATE schema_version SET version=2 WHERE version < 2;`,
	}
	for _, stmt := range stmts {
		if _, err := a.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) GetAppInfo() AppInfo {
	return AppInfo{Version: AppVersion, DataDir: a.dataDir}
}

func (a *App) GetDashboard() DashboardSummary {
	var out DashboardSummary
	if a.db == nil {
		return out
	}
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM clients`).Scan(&out.Clients)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM properties`).Scan(&out.Properties)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM properties WHERE TRIM(car_number) <> ''`).Scan(&out.PropertiesWithCAR)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM properties WHERE TRIM(kml_path) <> ''`).Scan(&out.PropertiesWithKML)
	cut := time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM car_checks WHERE checked_at >= ?`, cut).Scan(&out.ChecksLast30Days)
	return out
}

func (a *App) ListClients() ([]Client, error) {
	if a.db == nil {
		return nil, errors.New("banco local indisponível")
	}
	rows, err := a.db.Query(`SELECT id,name,cpf_cnpj,phone,notes,created_at,updated_at FROM clients ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.Name, &c.CPFCNPJ, &c.Phone, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (a *App) SaveClient(c Client) (Client, error) {
	if a.db == nil {
		return Client{}, errors.New("banco local indisponível")
	}
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return Client{}, errors.New("informe o nome do cliente")
	}
	now := time.Now().Format(time.RFC3339)
	if c.ID == 0 {
		res, err := a.db.Exec(`INSERT INTO clients(name,cpf_cnpj,phone,notes,created_at,updated_at) VALUES(?,?,?,?,?,?)`, c.Name, strings.TrimSpace(c.CPFCNPJ), strings.TrimSpace(c.Phone), strings.TrimSpace(c.Notes), now, now)
		if err != nil {
			return Client{}, err
		}
		c.ID, _ = res.LastInsertId()
		c.CreatedAt, c.UpdatedAt = now, now
		return c, nil
	}
	_, err := a.db.Exec(`UPDATE clients SET name=?,cpf_cnpj=?,phone=?,notes=?,updated_at=? WHERE id=?`, c.Name, strings.TrimSpace(c.CPFCNPJ), strings.TrimSpace(c.Phone), strings.TrimSpace(c.Notes), now, c.ID)
	if err != nil {
		return Client{}, err
	}
	c.UpdatedAt = now
	return c, nil
}

func (a *App) DeleteClient(id int64) error {
	if a.db == nil {
		return errors.New("banco local indisponível")
	}
	if id <= 0 {
		return errors.New("cliente inválido")
	}
	_, err := a.db.Exec(`DELETE FROM clients WHERE id=?`, id)
	return err
}

func (a *App) ListProperties(clientID int64) ([]Property, error) {
	if a.db == nil {
		return nil, errors.New("banco local indisponível")
	}
	q := `SELECT p.id,p.client_id,c.name,p.name,p.municipality,p.uf,p.registry,p.car_number,p.declared_area_ha,p.kml_path,p.created_at,p.updated_at
		FROM properties p JOIN clients c ON c.id=p.client_id`
	var rows *sql.Rows
	var err error
	if clientID > 0 {
		rows, err = a.db.Query(q+` WHERE p.client_id=? ORDER BY p.updated_at DESC`, clientID)
	} else {
		rows, err = a.db.Query(q + ` ORDER BY p.updated_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Property
	for rows.Next() {
		var p Property
		if err := rows.Scan(&p.ID, &p.ClientID, &p.ClientName, &p.Name, &p.Municipality, &p.UF, &p.Registry, &p.CARNumber, &p.DeclaredAreaHa, &p.KMLPath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (a *App) GetProperty(id int64) (Property, error) {
	if a.db == nil {
		return Property{}, errors.New("banco local indisponível")
	}
	var p Property
	err := a.db.QueryRow(`SELECT p.id,p.client_id,c.name,p.name,p.municipality,p.uf,p.registry,p.car_number,p.declared_area_ha,p.kml_path,p.created_at,p.updated_at
		FROM properties p JOIN clients c ON c.id=p.client_id WHERE p.id=?`, id).
		Scan(&p.ID, &p.ClientID, &p.ClientName, &p.Name, &p.Municipality, &p.UF, &p.Registry, &p.CARNumber, &p.DeclaredAreaHa, &p.KMLPath, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (a *App) SaveProperty(p Property) (Property, error) {
	if a.db == nil {
		return Property{}, errors.New("banco local indisponível")
	}
	p.Name = strings.TrimSpace(p.Name)
	p.UF = strings.ToUpper(strings.TrimSpace(p.UF))
	if p.ClientID <= 0 {
		return Property{}, errors.New("selecione o cliente")
	}
	if p.Name == "" {
		return Property{}, errors.New("informe o nome do imóvel")
	}
	if p.UF != "" && !validUF(p.UF) {
		return Property{}, errors.New("UF inválida")
	}
	if strings.TrimSpace(p.CARNumber) != "" {
		car, _, _, err := normalizeCAR(p.CARNumber)
		if err != nil {
			return Property{}, err
		}
		p.CARNumber = car
	}
	now := time.Now().Format(time.RFC3339)
	if p.ID == 0 {
		res, err := a.db.Exec(`INSERT INTO properties(client_id,name,municipality,uf,registry,car_number,declared_area_ha,kml_path,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			p.ClientID, p.Name, strings.TrimSpace(p.Municipality), p.UF, strings.TrimSpace(p.Registry), p.CARNumber, p.DeclaredAreaHa, strings.TrimSpace(p.KMLPath), now, now)
		if err != nil {
			return Property{}, err
		}
		p.ID, _ = res.LastInsertId()
		p.CreatedAt, p.UpdatedAt = now, now
		_ = os.MkdirAll(filepath.Join(a.dataDir, "properties", fmt.Sprint(p.ID)), 0o755)
		return p, nil
	}
	_, err := a.db.Exec(`UPDATE properties SET client_id=?,name=?,municipality=?,uf=?,registry=?,car_number=?,declared_area_ha=?,updated_at=? WHERE id=?`,
		p.ClientID, p.Name, strings.TrimSpace(p.Municipality), p.UF, strings.TrimSpace(p.Registry), p.CARNumber, p.DeclaredAreaHa, now, p.ID)
	if err != nil {
		return Property{}, err
	}
	p.UpdatedAt = now
	return p, nil
}

func (a *App) DeleteProperty(id int64) error {
	if a.db == nil {
		return errors.New("banco local indisponível")
	}
	if id <= 0 {
		return errors.New("imóvel inválido")
	}
	_, err := a.db.Exec(`DELETE FROM properties WHERE id=?`, id)
	if err == nil {
		_ = os.RemoveAll(filepath.Join(a.dataDir, "properties", fmt.Sprint(id)))
	}
	return err
}

func (a *App) BackupData() (BackupResult, error) {
	if a.ctx == nil {
		return BackupResult{}, errors.New("aplicativo ainda não inicializado")
	}
	defaultName := "ViaVerdeCAR_Backup_" + time.Now().Format("20060102_150405") + ".zip"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "Salvar backup do Via Verde CAR", DefaultFilename: defaultName, Filters: []runtime.FileFilter{{DisplayName: "Arquivo ZIP", Pattern: "*.zip"}}})
	if err != nil {
		return BackupResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return BackupResult{}, errors.New("backup cancelado")
	}
	if err := a.writeBackup(path); err != nil {
		return BackupResult{}, err
	}
	return BackupResult{Path: path, Message: "Backup criado com sucesso."}, nil
}

func (a *App) writeBackup(target string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.db != nil {
		_, _ = a.db.Exec(`PRAGMA wal_checkpoint(FULL);`)
	}
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	return filepath.Walk(a.dataDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(a.dataDir, path)
		if relErr != nil {
			return relErr
		}
		if info.IsDir() {
			if rel == "backups" || rel == "updates" || rel == "cache" {
				return filepath.SkipDir
			}
			return nil
		}
		absTarget, _ := filepath.Abs(target)
		absPath, _ := filepath.Abs(path)
		if absTarget == absPath {
			return nil
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, r)
		r.Close()
		return copyErr
	})
}

func (a *App) OpenDataFolder() error {
	if a.ctx == nil {
		return errors.New("aplicativo ainda não inicializado")
	}
	runtime.BrowserOpenURL(a.ctx, "file:///"+filepath.ToSlash(a.dataDir))
	return nil
}

func (a *App) SaveSetting(key, value string) error {
	if a.db == nil {
		return errors.New("banco local indisponível")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("chave inválida")
	}
	_, err := a.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (a *App) GetSetting(key string) string {
	if a.db == nil {
		return ""
	}
	var v string
	_ = a.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	return v
}

func marshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
