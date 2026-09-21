package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type SICARThemeImportResult struct {
	Theme   string `json:"theme"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (a *App) ImportSICARThemeZIP(propertyID int64, theme string) (SICARThemeImportResult, error) {
	if a.ctx == nil {
		return SICARThemeImportResult{}, errors.New("aplicativo ainda não inicializado")
	}
	if propertyID <= 0 {
		return SICARThemeImportResult{}, errors.New("selecione um imóvel")
	}
	theme = strings.ToUpper(strings.TrimSpace(theme))
	validTheme := false
	for _, def := range sicarThemeDefinitions {
		if def.Code == theme {
			validTheme = true
			break
		}
	}
	if !validTheme {
		return SICARThemeImportResult{}, errors.New("tema SICAR inválido")
	}
	car, err := a.GetLatestCAR(propertyID)
	if err != nil {
		return SICARThemeImportResult{}, errors.New("consulte o CAR do imóvel antes de importar um pacote de tema")
	}
	if car.MunicipalityCode == "" {
		return SICARThemeImportResult{}, errors.New("código municipal do CAR indisponível")
	}

	src, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Selecionar pacote ZIP oficial do SICAR",
		Filters: []runtime.FileFilter{{DisplayName: "Pacote ZIP", Pattern: "*.zip"}},
	})
	if err != nil {
		return SICARThemeImportResult{}, err
	}
	if strings.TrimSpace(src) == "" {
		return SICARThemeImportResult{}, errors.New("importação cancelada")
	}
	if err := validSICARThemeZip(src); err != nil {
		return SICARThemeImportResult{}, err
	}
	st, err := os.Stat(src)
	if err != nil {
		return SICARThemeImportResult{}, err
	}
	if st.Size() > 120<<20 {
		return SICARThemeImportResult{}, errors.New("pacote maior que 120 MB")
	}

	cacheDir := filepath.Join(a.dataDir, "cache", "sicar", car.MunicipalityCode)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return SICARThemeImportResult{}, err
	}
	dst := filepath.Join(cacheDir, strings.ToLower(theme)+".zip")
	data, err := os.ReadFile(src)
	if err != nil {
		return SICARThemeImportResult{}, err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return SICARThemeImportResult{}, err
	}
	_ = os.WriteFile(dst+".manual", []byte(time.Now().Format(time.RFC3339)), 0o644)
	return SICARThemeImportResult{
		Theme: theme,
		Path: dst,
		Message: "Pacote oficial importado. Consulte novamente o CAR para recalcular este tema.",
	}, nil
}
