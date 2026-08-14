package main

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port               string
	AppEnv             string
	BaseURL            string
	SupabaseURL        string
	SupabaseServiceKey string
	StorageBucket      string
	AdminName          string
	AdminUsername      string
	AdminPassword      string
	SessionDays        int
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func LoadConfig() (Config, error) {
	loadDotEnv(".env")
	days, _ := strconv.Atoi(env("SESSION_DAYS", "14"))
	if days < 1 {
		days = 14
	}
	c := Config{
		Port:               env("PORT", "8080"),
		AppEnv:             env("APP_ENV", "development"),
		BaseURL:            strings.TrimRight(env("APP_BASE_URL", ""), "/"),
		SupabaseURL:        strings.TrimRight(env("SUPABASE_URL", ""), "/"),
		SupabaseServiceKey: env("SUPABASE_SERVICE_ROLE_KEY", ""),
		StorageBucket:      env("SUPABASE_STORAGE_BUCKET", "via-verde-files"),
		AdminName:          env("ADMIN_NAME", "Pedro Massoli"),
		AdminUsername:      env("ADMIN_USERNAME", "pedro.massoli"),
		AdminPassword:      env("ADMIN_PASSWORD", ""),
		SessionDays:        days,
	}
	if c.SupabaseURL == "" || c.SupabaseServiceKey == "" {
		return c, errors.New("SUPABASE_URL e SUPABASE_SERVICE_ROLE_KEY ainda não foram configurados")
	}
	return c, nil
}
