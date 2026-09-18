package main

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type CARHistoryEntry struct {
	ID            int64    `json:"id"`
	PropertyID    int64    `json:"property_id"`
	CAR           string   `json:"car"`
	CheckedAt     string   `json:"checked_at"`
	Status        string   `json:"status"`
	Condition     string   `json:"condition"`
	AreaHa        float64  `json:"area_ha"`
	Municipality  string   `json:"municipality"`
	Changes       []string `json:"changes"`
	HasGeometry   bool     `json:"has_geometry"`
}

type PropertyPackageResult struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (a *App) GetLatestCAR(propertyID int64) (CARResult, error) {
	if a.db == nil {
		return CARResult{}, errors.New("banco local indisponível")
	}
	if propertyID <= 0 {
		return CARResult{}, errors.New("imóvel inválido")
	}
	var raw string
	err := a.db.QueryRow(`SELECT result_json FROM car_checks WHERE property_id=? ORDER BY checked_at DESC,id DESC LIMIT 1`, propertyID).Scan(&raw)
	if err != nil {
		return CARResult{}, err
	}
	var out CARResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return CARResult{}, err
	}
	return out, nil
}

func (a *App) GetCARHistory(propertyID int64) ([]CARHistoryEntry, error) {
	if a.db == nil {
		return nil, errors.New("banco local indisponível")
	}
	if propertyID <= 0 {
		return nil, errors.New("imóvel inválido")
	}
	rows, err := a.db.Query(`SELECT id,property_id,car_number,checked_at,status,condition_text,area_ha,municipality,geometry_json,result_json
		FROM car_checks WHERE property_id=? ORDER BY checked_at DESC,id DESC LIMIT 50`, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rowData struct {
		entry CARHistoryEntry
		raw   string
	}
	var tmp []rowData
	for rows.Next() {
		var e CARHistoryEntry
		var geometry, raw string
		if err := rows.Scan(&e.ID, &e.PropertyID, &e.CAR, &e.CheckedAt, &e.Status, &e.Condition, &e.AreaHa, &e.Municipality, &geometry, &raw); err != nil {
			return nil, err
		}
		e.HasGeometry = strings.TrimSpace(geometry) != ""
		tmp = append(tmp, rowData{entry: e, raw: raw})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range tmp {
		if i+1 >= len(tmp) {
			continue
		}
		tmp[i].entry.Changes = compareCARHistoryJSON(tmp[i+1].raw, tmp[i].raw)
	}
	out := make([]CARHistoryEntry, 0, len(tmp))
	for _, r := range tmp {
		out = append(out, r.entry)
	}
	return out, nil
}

func compareCARHistoryJSON(olderRaw, newerRaw string) []string {
	var oldR, newR CARResult
	if json.Unmarshal([]byte(olderRaw), &oldR) != nil || json.Unmarshal([]byte(newerRaw), &newR) != nil {
		return nil
	}
	var changes []string
	if strings.TrimSpace(oldR.Status) != strings.TrimSpace(newR.Status) {
		changes = append(changes, fmt.Sprintf("Situação: %s → %s", emptyDash(oldR.Status), emptyDash(newR.Status)))
	}
	if strings.TrimSpace(oldR.Condition) != strings.TrimSpace(newR.Condition) {
		changes = append(changes, fmt.Sprintf("Condição: %s → %s", emptyDash(oldR.Condition), emptyDash(newR.Condition)))
	}
	if strings.TrimSpace(oldR.Municipality) != strings.TrimSpace(newR.Municipality) {
		changes = append(changes, fmt.Sprintf("Município: %s → %s", emptyDash(oldR.Municipality), emptyDash(newR.Municipality)))
	}
	if oldR.AreaHa > 0 || newR.AreaHa > 0 {
		if absFloat(oldR.AreaHa-newR.AreaHa) > 0.0001 {
			changes = append(changes, fmt.Sprintf("Área: %.4f ha → %.4f ha", oldR.AreaHa, newR.AreaHa))
		}
	}
	if oldR.GeometryAreaHa > 0 || newR.GeometryAreaHa > 0 {
		if absFloat(oldR.GeometryAreaHa-newR.GeometryAreaHa) > 0.0001 {
			changes = append(changes, fmt.Sprintf("Área geométrica: %.4f ha → %.4f ha", oldR.GeometryAreaHa, newR.GeometryAreaHa))
		}
	}
	if oldR.GeoJSON != "" && newR.GeoJSON != "" && oldR.GeoJSON != newR.GeoJSON {
		changes = append(changes, "Geometria pública alterada")
	}
	if oldR.Environment.IBAMAEmbargoCount != newR.Environment.IBAMAEmbargoCount {
		changes = append(changes, fmt.Sprintf("Embargos IBAMA: %d → %d", oldR.Environment.IBAMAEmbargoCount, newR.Environment.IBAMAEmbargoCount))
	}
	if oldR.Environment.IndigenousCount != newR.Environment.IndigenousCount {
		changes = append(changes, fmt.Sprintf("Interseções FUNAI: %d → %d", oldR.Environment.IndigenousCount, newR.Environment.IndigenousCount))
	}
	if oldR.Environment.FederalUCCount != newR.Environment.FederalUCCount {
		changes = append(changes, fmt.Sprintf("UCs federais: %d → %d", oldR.Environment.FederalUCCount, newR.Environment.FederalUCCount))
	}
	if oldR.Environment.MCRListed != newR.Environment.MCRListed {
		changes = append(changes, fmt.Sprintf("Lista MMA/MCR: %t → %t", oldR.Environment.MCRListed, newR.Environment.MCRListed))
	}
	for _, code := range []string{"APP", "RESERVA_LEGAL", "VEGETACAO_NATIVA", "AREA_CONSOLIDADA"} {
		oldM, oldOK := oldR.Themes.Themes[code]
		newM, newOK := newR.Themes.Themes[code]
		if oldOK && newOK && absFloat(oldM.AreaHa-newM.AreaHa) > 0.01 {
			changes = append(changes, fmt.Sprintf("%s: %.2f ha → %.2f ha", code, oldM.AreaHa, newM.AreaHa))
		}
	}
	return changes
}

func emptyDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "—"
	}
	return v
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func (a *App) ExportPropertyPackage(propertyID int64, car CARResult, kml KMLResult, comparison GeometryComparison) (PropertyPackageResult, error) {
	if a.ctx == nil {
		return PropertyPackageResult{}, errors.New("aplicativo ainda não inicializado")
	}
	p, err := a.GetProperty(propertyID)
	if err != nil {
		return PropertyPackageResult{}, err
	}
	history, _ := a.GetCARHistory(propertyID)

	defaultName := "Dossie_CAR_" + safeCARFilename(car.CAR) + "_" + time.Now().Format("20060102") + ".zip"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvar dossiê técnico do imóvel",
		DefaultFilename: defaultName,
		Filters:         []runtime.FileFilter{{DisplayName: "Arquivo ZIP", Pattern: "*.zip"}},
	})
	if err != nil {
		return PropertyPackageResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return PropertyPackageResult{}, errors.New("exportação cancelada")
	}

	f, err := os.Create(path)
	if err != nil {
		return PropertyPackageResult{}, err
	}
	zw := zip.NewWriter(f)
	closeWithError := func(e error) (PropertyPackageResult, error) {
		_ = zw.Close()
		_ = f.Close()
		if e != nil {
			_ = os.Remove(path)
		}
		return PropertyPackageResult{}, e
	}

	meta := map[string]any{
		"gerado_em":  time.Now().Format(time.RFC3339),
		"aplicativo": "Via Verde CAR",
		"versao":     AppVersion,
		"imovel":     p,
		"car":        car,
		"kml":        kml,
		"comparacao": comparison,
		"historico":  history,
		"aviso":      "Dossiê técnico auxiliar. Não substitui documento oficial do SICAR, certificação, georreferenciamento ou análise ambiental do órgão competente.",
	}
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	if err := zipWriteBytes(zw, "01_Dados_Tecnicos.json", metaJSON); err != nil {
		return closeWithError(err)
	}

	pdf := buildCARTechnicalPDF(p, car, kml, comparison)
	if err := zipWriteBytes(zw, "02_Demonstrativo_Tecnico.pdf", pdf); err != nil {
		return closeWithError(err)
	}

	if car.HasGeometry && strings.TrimSpace(car.GeoJSON) != "" {
		var fcar carGeoFeature
		if json.Unmarshal([]byte(car.GeoJSON), &fcar) == nil {
			if data, err := carGeometryKML(car.CAR, fcar.Geometry); err == nil {
				if err := zipWriteBytes(zw, "03_CAR_SICAR.kml", data); err != nil {
					return closeWithError(err)
				}
			}
		}
	}
	if strings.TrimSpace(p.KMLPath) != "" {
		if data, err := os.ReadFile(p.KMLPath); err == nil {
			if err := zipWriteBytes(zw, "04_KML_Cliente.kml", data); err != nil {
				return closeWithError(err)
			}
		}
	}

	var csvBuf strings.Builder
	cw := csv.NewWriter(&csvBuf)
	_ = cw.Write([]string{"Data da consulta", "CAR", "Situação", "Condição", "Área (ha)", "Município", "Alterações detectadas"})
	for _, h := range history {
		_ = cw.Write([]string{h.CheckedAt, h.CAR, h.Status, h.Condition, strconv.FormatFloat(h.AreaHa, 'f', 4, 64), h.Municipality, strings.Join(h.Changes, " | ")})
	}
	cw.Flush()
	if err := zipWriteBytes(zw, "05_Historico_CAR.csv", []byte(csvBuf.String())); err != nil {
		return closeWithError(err)
	}

	var themeBuf strings.Builder
	tw := csv.NewWriter(&themeBuf)
	_ = tw.Write([]string{"Tema", "Descrição", "Área estimada (ha)", "Feições intersectadas", "Disponível", "Método"})
	for _, code := range []string{"APP", "RESERVA_LEGAL", "VEGETACAO_NATIVA", "AREA_CONSOLIDADA", "USO_RESTRITO", "SERVIDAO_ADMINISTRATIVA"} {
		m, ok := car.Themes.Themes[code]
		if !ok {
			continue
		}
		_ = tw.Write([]string{m.Code, m.Label, strconv.FormatFloat(m.AreaHa, 'f', 4, 64), strconv.Itoa(m.FeatureCount), strconv.FormatBool(m.Available), m.Method})
	}
	tw.Flush()
	if err := zipWriteBytes(zw, "06_Temas_SICAR.csv", []byte(themeBuf.String())); err != nil {
		return closeWithError(err)
	}

	envJSON, _ := json.MarshalIndent(car.Environment, "", "  ")
	if err := zipWriteBytes(zw, "07_Triagem_Socioambiental.json", envJSON); err != nil {
		return closeWithError(err)
	}

	readme := "VIA VERDE CAR — DOSSIÊ TÉCNICO E SOCIOAMBIENTAL\r\n\r\n" +
		"Este pacote reúne a conferência do imóvel, geometria pública do SICAR, KML, temas declarados do CAR quando disponíveis, histórico e triagens espaciais em bases públicas oficiais.\r\n\r\n" +
		"Os temas APP, Reserva Legal, vegetação nativa e demais camadas são estimativas por interseção espacial com pacotes municipais públicos do SICAR. As ocorrências em IBAMA, FUNAI, ICMBio e MMA/MCR exigem confirmação na fonte oficial e análise do contexto jurídico e documental.\r\n\r\n" +
		"IMPORTANTE: este material é auxiliar e não substitui o Demonstrativo oficial do CAR, certidões, memorial descritivo, georreferenciamento, análise ambiental ou documento emitido pelo órgão competente.\r\n"
	if err := zipWriteBytes(zw, "LEIA-ME.txt", []byte(readme)); err != nil {
		return closeWithError(err)
	}

	if err := zw.Close(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return PropertyPackageResult{}, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return PropertyPackageResult{}, err
	}
	return PropertyPackageResult{Path: filepath.Clean(path), Message: "Dossiê técnico criado com sucesso."}, nil
}

func zipWriteBytes(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
