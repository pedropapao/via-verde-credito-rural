package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (a *App) saveAutomaticCARKML(p Property, result CARResult, geometry carGeoJSONGeometry) (string, error) {
	if !result.HasGeometry {
		return "", errors.New("geometria do SICAR indisponível")
	}
	data, err := carGeometryKML(result.CAR, geometry)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(a.dataDir, "properties", fmt.Sprint(p.ID), "car")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	owner := safeFilePart(p.ClientName)
	property := safeFilePart(p.Name)
	if owner == "" {
		owner = "Produtor"
	}
	if property == "" {
		property = "Imovel"
	}
	name := "CAR_" + owner + "_" + property + "_" + safeCARFilename(result.CAR) + ".kml"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func safeFilePart(v string) string {
	v = strings.TrimSpace(v)
	repl := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"Á", "A", "À", "A", "Ã", "A", "Â", "A", "Ä", "A",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"É", "E", "È", "E", "Ê", "E", "Ë", "E",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
		"ó", "o", "ò", "o", "õ", "o", "ô", "o", "ö", "o",
		"Ó", "O", "Ò", "O", "Õ", "O", "Ô", "O", "Ö", "O",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
		"ç", "c", "Ç", "C",
	)
	v = repl.Replace(v)
	v = regexp.MustCompile(`[^A-Za-z0-9._-]+`).ReplaceAllString(v, "_")
	v = strings.Trim(v, "._-")
	if len(v) > 60 {
		v = v[:60]
	}
	return v
}

func carResultMateriallyChanged(previousRaw string, current CARResult) bool {
	var previous CARResult
	if err := json.Unmarshal([]byte(previousRaw), &previous); err != nil {
		return true
	}
	if strings.TrimSpace(previous.CAR) != strings.TrimSpace(current.CAR) ||
		strings.TrimSpace(previous.Status) != strings.TrimSpace(current.Status) ||
		strings.TrimSpace(previous.Condition) != strings.TrimSpace(current.Condition) ||
		strings.TrimSpace(previous.Municipality) != strings.TrimSpace(current.Municipality) ||
		strings.TrimSpace(previous.PropertyType) != strings.TrimSpace(current.PropertyType) ||
		strings.TrimSpace(previous.PropertyName) != strings.TrimSpace(current.PropertyName) ||
		strings.TrimSpace(previous.DataAtualizacao) != strings.TrimSpace(current.DataAtualizacao) {
		return true
	}
	if absFloat(previous.AreaHa-current.AreaHa) > 0.0001 ||
		absFloat(previous.GeometryAreaHa-current.GeometryAreaHa) > 0.0001 ||
		absFloat(previous.FiscalModules-current.FiscalModules) > 0.0001 {
		return true
	}
	// GeoJSON contém também atributos, então comparamos a geometria normalizada.
	if geometryFingerprint(previous.GeoJSON) != geometryFingerprint(current.GeoJSON) {
		return true
	}
	if previous.Environment.IBAMAEmbargoCount != current.Environment.IBAMAEmbargoCount ||
		previous.Environment.IndigenousCount != current.Environment.IndigenousCount ||
		previous.Environment.FederalUCCount != current.Environment.FederalUCCount ||
		previous.Environment.MCRListed != current.Environment.MCRListed {
		return true
	}
	for _, code := range []string{"APP", "RESERVA_LEGAL", "VEGETACAO_NATIVA", "AREA_CONSOLIDADA", "USO_RESTRITO", "SERVIDAO_ADMINISTRATIVA"} {
		oldM, oldOK := previous.Themes.Themes[code]
		newM, newOK := current.Themes.Themes[code]
		if oldOK != newOK || oldM.Available != newM.Available || absFloat(oldM.AreaHa-newM.AreaHa) > 0.01 {
			return true
		}
	}
	return false
}

func geometryFingerprint(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var feature carGeoFeature
	if json.Unmarshal([]byte(raw), &feature) != nil {
		return strings.TrimSpace(raw)
	}
	b, _ := json.Marshal(feature.Geometry)
	return string(b)
}
