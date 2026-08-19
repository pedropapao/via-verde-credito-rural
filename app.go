package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed web/templates/*.html web/static/*
var webFS embed.FS

const AppVersion = "2.0.0-web"

type App struct {
	cfg        Config
	sb         *Supabase
	templates  map[string]*template.Template
	startupErr error
	mu         sync.RWMutex
}

type ViewData struct {
	Title       string
	User        *User
	CSRF        string
	CurrentPath string
	Flash       string
	Error       string
	Version     string
	Data        any
}

func NewApp(cfg Config, cfgErr error) *App {
	a := &App{cfg: cfg, templates: map[string]*template.Template{}, startupErr: cfgErr}
	if cfg.SupabaseURL != "" && cfg.SupabaseServiceKey != "" {
		a.sb = NewSupabase(cfg.SupabaseURL, cfg.SupabaseServiceKey)
	}
	a.parseTemplates()
	if a.sb != nil {
		ctx, cancel := contextWithTimeout(20 * time.Second)
		defer cancel()
		if err := a.ensureBootstrap(ctx); err != nil {
			log.Printf("bootstrap pendente: %v", err)
			if a.startupErr == nil {
				a.startupErr = err
			}
		} else if err := a.sb.EnsureBucket(ctx, cfg.StorageBucket); err != nil {
			log.Printf("storage bucket: %v", err)
		}
	}
	return a
}

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

func (a *App) parseTemplates() {
	funcs := template.FuncMap{
		"money":       formatMoney,
		"dateBR":      dateBR,
		"statusClass": statusClass,
		"roleLabel": func(v string) string {
			if v == "owner" {
				return "Proprietário"
			}
			return "Visualização"
		},
		"bytes":     formatBytes,
		"pct":       func(v float64) string { return strings.ReplaceAll(fmt.Sprintf("%.2f%%", v), ".", ",") },
		"safe":      func(v string) template.HTML { return template.HTML(v) },
		"hasPrefix": strings.HasPrefix,
		"list":      func(v ...string) []string { return v },
	}
	pages := []string{"login", "dashboard", "projects", "project_form", "project_detail", "clients", "properties", "daily", "users", "invite", "rules", "settings", "documents", "setup", "car"}
	for _, page := range pages {
		t, err := template.New("base").Funcs(funcs).ParseFS(webFS, "web/templates/base.html", "web/templates/"+page+".html")
		if err != nil {
			panic(err)
		}
		a.templates[page] = t
	}
}

func (a *App) render(w http.ResponseWriter, r *http.Request, page string, data ViewData) {
	data.Version = AppVersion
	if data.User == nil {
		data.User = userFromContext(r.Context())
	}
	if data.CSRF == "" {
		if s := sessionFromContext(r.Context()); s != nil {
			data.CSRF = s.CSRFToken
		}
	}
	data.CurrentPath = r.URL.Path
	if data.Flash == "" {
		data.Flash = r.URL.Query().Get("ok")
	}
	t := a.templates[page]
	if t == nil {
		http.Error(w, "template não encontrado", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("template %s: %v", page, err)
	}
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	staticFS, _ := fs.Sub(webFS, "web/static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /login", a.loginGet)
	mux.HandleFunc("POST /login", a.loginPost)
	mux.HandleFunc("GET /invite/{token}", a.inviteGet)
	mux.HandleFunc("POST /invite/{token}", a.invitePost)

	mux.Handle("GET /", a.withAuth(http.HandlerFunc(a.dashboard)))
	mux.Handle("POST /logout", a.withAuth(http.HandlerFunc(a.logout)))

	mux.Handle("GET /projects", a.withAuth(http.HandlerFunc(a.projectsList)))
	mux.Handle("GET /projects/new", a.ownerOnly(http.HandlerFunc(a.projectNewGet)))
	mux.Handle("POST /projects/new", a.ownerOnly(http.HandlerFunc(a.projectNewPost)))
	mux.Handle("GET /projects/{id}", a.withAuth(http.HandlerFunc(a.projectDetail)))
	mux.Handle("GET /projects/{id}/edit", a.ownerOnly(http.HandlerFunc(a.projectEditGet)))
	mux.Handle("POST /projects/{id}/edit", a.ownerOnly(http.HandlerFunc(a.projectEditPost)))
	mux.Handle("POST /projects/{id}/files", a.ownerOnly(http.HandlerFunc(a.projectFileUpload)))
	mux.Handle("GET /files/{id}/download", a.withAuth(http.HandlerFunc(a.fileDownload)))
	mux.Handle("POST /files/{id}/delete", a.ownerOnly(http.HandlerFunc(a.fileDelete)))

	mux.Handle("GET /clients", a.withAuth(http.HandlerFunc(a.clientsList)))
	mux.Handle("POST /clients", a.ownerOnly(http.HandlerFunc(a.clientCreate)))
	mux.Handle("GET /properties", a.withAuth(http.HandlerFunc(a.propertiesList)))
	mux.Handle("POST /properties", a.ownerOnly(http.HandlerFunc(a.propertyCreate)))
	mux.Handle("GET /documents", a.withAuth(http.HandlerFunc(a.documentsList)))

	mux.Handle("GET /reports/daily", a.withAuth(http.HandlerFunc(a.dailyList)))
	mux.Handle("POST /reports/daily", a.ownerOnly(http.HandlerFunc(a.dailyCreate)))
	mux.Handle("GET /reports/daily/export.csv", a.withAuth(http.HandlerFunc(a.dailyExportCSV)))

	mux.Handle("GET /rules", a.withAuth(http.HandlerFunc(a.rulesList)))
	mux.Handle("POST /rules", a.ownerOnly(http.HandlerFunc(a.ruleCreate)))

	mux.Handle("GET /users", a.ownerOnly(http.HandlerFunc(a.usersList)))
	mux.Handle("POST /users/invite", a.ownerOnly(http.HandlerFunc(a.userInvite)))
	mux.Handle("POST /users/{id}/toggle", a.ownerOnly(http.HandlerFunc(a.userToggle)))

	mux.Handle("GET /settings", a.withAuth(http.HandlerFunc(a.settingsGet)))
	mux.Handle("POST /settings/password", a.withAuth(http.HandlerFunc(a.settingsPassword)))

	return securityHeaders(recoverer(logger(mux)))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'self'")
		next.ServeHTTP(w, r)
	})
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Printf("panic: %v", v)
				http.Error(w, "Ocorreu um erro interno. Tente novamente.", 500)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "version": AppVersion, "database_configured": a.sb != nil})
}

func (a *App) ensureBootstrap(ctx context.Context) error {
	if a.sb == nil {
		return errors.New("Supabase não configurado")
	}
	var users []User
	if err := a.sb.Select(ctx, "users", "select=id,username&limit=1", &users); err != nil {
		return fmt.Errorf("execute primeiro supabase/01_schema.sql: %w", err)
	}
	if len(users) > 0 {
		return nil
	}
	if a.cfg.AdminPassword == "" {
		return errors.New("defina ADMIN_PASSWORD no ambiente para criar o perfil Pedro Massoli")
	}
	hash, err := HashPassword(a.cfg.AdminPassword)
	if err != nil {
		return err
	}
	row := User{Name: a.cfg.AdminName, Username: a.cfg.AdminUsername, PasswordHash: hash, Role: "owner", Active: true}
	var out []User
	if err := a.sb.Insert(ctx, "users", row, &out); err != nil {
		return err
	}
	log.Printf("perfil proprietário criado: %s (%s)", a.cfg.AdminName, a.cfg.AdminUsername)
	return nil
}

func (a *App) audit(ctx context.Context, u *User, action, entity, entityID, details string) {
	if a.sb == nil || u == nil {
		return
	}
	_ = a.sb.Insert(ctx, "audit_logs", AuditLog{UserID: u.ID, Action: action, Entity: entity, EntityID: entityID, Details: details}, nil)
}

func formatMoney(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	parts := strings.Split(s, ".")
	intp := parts[0]
	neg := ""
	if strings.HasPrefix(intp, "-") {
		neg = "-"
		intp = strings.TrimPrefix(intp, "-")
	}
	for i := len(intp) - 3; i > 0; i -= 3 {
		intp = intp[:i] + "." + intp[i:]
	}
	return neg + "R$ " + intp + "," + parts[1]
}
func dateBR(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t.Format("02/01/2006")
		}
	}
	return v
}
func statusClass(v string) string {
	x := strings.ToLower(v)
	switch {
	case strings.Contains(x, "aprov"), strings.Contains(x, "contrat"):
		return "ok"
	case strings.Contains(x, "reprov"), strings.Contains(x, "negad"):
		return "bad"
	case strings.Contains(x, "andamento"), strings.Contains(x, "enviado"), strings.Contains(x, "análise"):
		return "warn"
	default:
		return "neutral"
	}
}
func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}
func parseFloat(v string) float64 {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "R$", "")
	v = strings.ReplaceAll(v, " ", "")
	if strings.Contains(v, ",") {
		v = strings.ReplaceAll(v, ".", "")
		v = strings.ReplaceAll(v, ",", ".")
	}
	f, _ := strconv.ParseFloat(v, 64)
	return f
}
func parseInt(v string) int { i, _ := strconv.Atoi(strings.TrimSpace(v)); return i }
func baseURL(r *http.Request, cfg Config) string {
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") != "" {
		scheme = r.Header.Get("X-Forwarded-Proto")
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
func safeFilename(v string) string {
	v = filepath.Base(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, "\\", "_")
	return v
}
