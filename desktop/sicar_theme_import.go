package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type SICARThemeImportResult struct {
	Theme     string  `json:"theme"`
	Path      string  `json:"path"`
	Message   string  `json:"message"`
	Available bool    `json:"available"`
	AreaHa    float64 `json:"area_ha"`
	Features  int     `json:"features"`
}

func (a *App) ImportSICARThemeZIP(propertyID int64, theme string) (SICARThemeImportResult, error) {
	if a == nil || a.ctx == nil {
		return SICARThemeImportResult{}, errors.New("aplicativo ainda não inicializado")
	}
	theme = strings.ToUpper(strings.TrimSpace(theme))
	label := ""
	for _, def := range sicarThemeDefinitions {
		if def.Code == theme {
			label = def.Label
			break
		}
	}
	if label == "" {
		return SICARThemeImportResult{}, errors.New("tema SICAR inválido")
	}

	var car CARResult
	var err error
	if propertyID > 0 {
		car, err = a.GetLatestCAR(propertyID)
	} else {
		var session CARSessionCache
		session, err = a.GetLastCARSession()
		if err == nil {
			car = session.Result
		}
	}
	if err != nil || !car.Found || strings.TrimSpace(car.GeoJSON) == "" {
		return SICARThemeImportResult{}, errors.New("consulte um CAR com geometria antes de importar um pacote de tema")
	}
	if car.MunicipalityCode == "" {
		return SICARThemeImportResult{}, errors.New("código municipal do CAR indisponível")
	}

	src, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Selecionar ZIP oficial — " + label,
		Filters: []runtime.FileFilter{{DisplayName: "Pacote ZIP do SICAR", Pattern: "*.zip"}},
	})
	if err != nil {
		return SICARThemeImportResult{}, err
	}
	if strings.TrimSpace(src) == "" {
		return SICARThemeImportResult{}, errors.New("importação cancelada")
	}
	if err := validSICARThemeZip(src); err != nil {
		return SICARThemeImportResult{}, fmt.Errorf("ZIP inválido: %w", err)
	}
	if !sicarThemeZipLooksCompatible(src, theme) {
		return SICARThemeImportResult{}, fmt.Errorf("o ZIP selecionado não parece corresponder a %s; baixe esse tema na Base de Downloads do SICAR", label)
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
	tmp := dst + ".import.part"
	if err := copySICARThemeFile(src, tmp); err != nil {
		return SICARThemeImportResult{}, err
	}
	if err := validSICARThemeZip(tmp); err != nil {
		_ = os.Remove(tmp)
		return SICARThemeImportResult{}, err
	}
	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return SICARThemeImportResult{}, err
	}
	marker := fmt.Sprintf("imported_at=%s\nsource=%s\ntheme=%s\n", time.Now().Format(time.RFC3339), filepath.Base(src), theme)
	_ = os.WriteFile(dst+".manual", []byte(marker), 0o644)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	metric, err := a.analyzeSingleSICARTheme(ctx, car.CAR, car.UF, car.MunicipalityCode, car.GeoJSON, car.AreaHa, theme, label)
	if err != nil {
		return SICARThemeImportResult{}, err
	}
	if car.Themes.Themes == nil {
		car.Themes = SICARThemesSummary{
			CheckedAt: time.Now().Format(time.RFC3339),
			SourceURL: "https://consulta.car.gov.br/geoservices",
			Municipio: car.MunicipalityCode,
			UF: strings.ToUpper(strings.TrimSpace(car.UF)),
			Themes: map[string]SICARThemeMetric{},
		}
	}
	car.Themes.Themes[theme] = metric
	car.Themes.CheckedAt = time.Now().Format(time.RFC3339)
	car.Themes.Complete = true
	for _, def := range sicarThemeDefinitions {
		m, ok := car.Themes.Themes[def.Code]
		if !ok || !m.Available {
			car.Themes.Complete = false
			break
		}
	}
	car.Themes.Warnings = rebuildSICARThemeWarnings(car.Themes)

	if propertyID > 0 && a.db != nil {
		_, _ = a.db.Exec(`UPDATE properties SET last_car_json=?,updated_at=? WHERE id=?`, marshalJSON(car), time.Now().Format(time.RFC3339), propertyID)
	}
	if session, sessionErr := a.GetLastCARSession(); sessionErr == nil && strings.EqualFold(session.Result.CAR, car.CAR) {
		_ = a.updateLastCARSessionResult(car)
	}
	_ = a.ClearEnvironmentalIntelligenceCache(car.CAR)

	return SICARThemeImportResult{
		Theme: theme,
		Path: dst,
		Message: fmt.Sprintf("%s importada e cruzada com o CAR: %.4f ha em %d feição(ões).", label, metric.AreaHa, metric.FeatureCount),
		Available: metric.Available,
		AreaHa: metric.AreaHa,
		Features: metric.FeatureCount,
	}, nil
}

func copySICARThemeFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(out, io.LimitReader(in, (120<<20)+1))
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}
	if n > 120<<20 {
		_ = os.Remove(dst)
		return errors.New("pacote maior que 120 MB")
	}
	return nil
}
