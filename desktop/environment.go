package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	ibamaEmbargoLayerURL = "https://pamgia.ibama.gov.br/server/rest/services/01_Publicacoes_Bases/adm_embargos_ibama_a/FeatureServer/0/query"
	ibamaEmbargoLegacyURL = "https://pamgia.ibama.gov.br/server/rest/services/01_Publicacoes_Bases/embargos_siscom_brasil/FeatureServer/2/query"
	funaiWFSURL           = "https://geoserver.funai.gov.br/geoserver/Funai/ows"
	icmbioWFSURL          = "https://geoservicos.inde.gov.br/geoserver/ICMBio/ows"
	icmbioUCLayer         = "ICMBio:limiteucsfederais_a"
	environmentMCRURL     = "https://www.gov.br/mma/pt-br/assuntos/controle-ao-desmatamento-queimadas-e-ordenamento-ambiental-territorial/controle-do-desmatamento-1/atendimento-ao-manual-de-credito-rural"
	mmaMCRZipURL          = "https://www.gov.br/mma/pt-br/assuntos/controle-ao-desmatamento-queimadas-e-ordenamento-ambiental-territorial/controle-do-desmatamento-1/atendimento-ao-manual-de-credito-rural/car_lista_mcr_acima4mf_toleranc.zip"
	icmbioGeoURL          = "https://www.gov.br/icmbio/pt-br/dados-icmbio/dados_geoespaciais"
	funaiGeoURL           = "https://www.gov.br/funai/pt-br/atuacao/terras-indigenas/geoprocessamento-e-mapas"
)

type EnvironmentalSummary struct {
	CheckedAt          string             `json:"checked_at"`
	IBAMAChecked       bool               `json:"ibama_checked"`
	IBAMAEmbargoCount  int                `json:"ibama_embargo_count"`
	IBAMAEmbargos      []EmbargoFinding   `json:"ibama_embargos"`
	FUNAIChecked       bool               `json:"funai_checked"`
	IndigenousCount    int                `json:"indigenous_count"`
	IndigenousFindings []TerritoryFinding `json:"indigenous_findings"`
	ICMBioChecked      bool               `json:"icmbio_checked"`
	FederalUCCount     int                `json:"federal_uc_count"`
	FederalUCFindings  []UCFindings       `json:"federal_uc_findings"`
	MCRChecked         bool               `json:"mcr_checked"`
	MCRListed          bool               `json:"mcr_listed"`
	MCRDataUpdated     string             `json:"mcr_data_updated"`
	MCRFields          map[string]string  `json:"mcr_fields"`
	Warnings           []string           `json:"warnings"`
	IBAMASourceURL     string             `json:"ibama_source_url"`
	FUNAISourceURL     string             `json:"funai_source_url"`
	ICMBioSourceURL    string             `json:"icmbio_source_url"`
	MCRSourceURL       string             `json:"mcr_source_url"`
}

type EmbargoFinding struct {
	Number        string  `json:"number"`
	Date          string  `json:"date"`
	Status        string  `json:"status"`
	Situation     string  `json:"situation"`
	Area          string  `json:"area"`
	Municipality  string  `json:"municipality"`
	Agency        string  `json:"agency"`
	Infraction    string  `json:"infraction"`
	OverlapAreaHa float64 `json:"overlap_area_ha"`
	OverlapCARPct float64 `json:"overlap_car_pct"`
	GeoJSON       string  `json:"geojson"`
}

type TerritoryFinding struct {
	Name           string  `json:"name"`
	Phase          string  `json:"phase"`
	OverlapAreaHa  float64 `json:"overlap_area_ha"`
	OverlapCARPct  float64 `json:"overlap_car_pct"`
	Source         string  `json:"source"`
	GeoJSON        string  `json:"geojson"`
}

type UCFindings struct {
	Name          string  `json:"name"`
	Category      string  `json:"category"`
	Group         string  `json:"group"`
	UF            string  `json:"uf"`
	OverlapAreaHa float64 `json:"overlap_area_ha"`
	OverlapCARPct float64 `json:"overlap_car_pct"`
	Source        string  `json:"source"`
	GeoJSON       string  `json:"geojson"`
}

func screenEnvironment(ctx context.Context, carGeoJSON string) EnvironmentalSummary {
	out := EnvironmentalSummary{
		CheckedAt:      time.Now().Format(time.RFC3339),
		IBAMASourceURL: "https://pamgia.ibama.gov.br/server/rest/services/01_Publicacoes_Bases/adm_embargos_ibama_a/FeatureServer/0",
		FUNAISourceURL: funaiGeoURL,
		ICMBioSourceURL: icmbioGeoURL,
		MCRSourceURL:   environmentMCRURL,
	}
	if strings.TrimSpace(carGeoJSON) == "" {
		out.Warnings = append(out.Warnings, "Sem geometria do CAR para cruzamentos territoriais.")
		return out
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Add(3)

	go func() {
		defer wg.Done()
		findings, err := queryIBAMAEmbargos(ctx, carGeoJSON)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			out.Warnings = append(out.Warnings, "IBAMA: "+err.Error())
			return
		}
		out.IBAMAChecked = true
		out.IBAMAEmbargos = findings
		out.IBAMAEmbargoCount = len(findings)
	}()

	go func() {
		defer wg.Done()
		findings, err := queryFUNAITerritories(ctx, carGeoJSON)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			out.Warnings = append(out.Warnings, "FUNAI: "+err.Error())
			return
		}
		out.FUNAIChecked = true
		out.IndigenousFindings = findings
		out.IndigenousCount = len(findings)
	}()

	go func() {
		defer wg.Done()
		findings, err := queryICMBioFederalUCs(ctx, carGeoJSON)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			out.Warnings = append(out.Warnings, "ICMBio: "+err.Error())
			return
		}
		out.ICMBioChecked = true
		out.FederalUCFindings = findings
		out.FederalUCCount = len(findings)
	}()

	wg.Wait()
	return out
}

func queryIBAMAEmbargos(ctx context.Context, carRaw string) ([]EmbargoFinding, error) {
	// A base adm_embargos_ibama_a é a publicação atual do PAMGIA e tem
	// atualização diária. Mantemos embargos_siscom_brasil apenas como fallback.
	type source struct {
		URL     string
		Current bool
	}
	var errs []string
	for _, src := range []source{
		{URL: ibamaEmbargoLayerURL, Current: true},
		{URL: ibamaEmbargoLegacyURL, Current: false},
	} {
		findings, err := queryIBAMAEmbargosSource(ctx, carRaw, src.URL, src.Current)
		if err == nil {
			return findings, nil
		}
		errs = append(errs, err.Error())
		if ctx.Err() != nil {
			break
		}
	}
	if len(errs) == 0 {
		return nil, errors.New("fonte IBAMA não respondeu")
	}
	return nil, errors.New(strings.Join(errs, " | "))
}

func queryIBAMAEmbargosSource(ctx context.Context, carRaw, endpoint string, current bool) ([]EmbargoFinding, error) {
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		return nil, fmt.Errorf("limites do CAR indisponíveis")
	}
	base := url.Values{}
	base.Set("where", "1=1")
	base.Set("geometry", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f", minLon, minLat, maxLon, maxLat))
	base.Set("geometryType", "esriGeometryEnvelope")
	base.Set("inSR", "4326")
	base.Set("spatialRel", "esriSpatialRelIntersects")
	base.Set("f", "json")

	// Primeiro pedimos somente a contagem. Para a situação mais comum (zero
	// interseções) isto evita transferir geometrias e torna a triagem muito mais
	// leve que a consulta anterior.
	countParams := cloneURLValues(base)
	countParams.Set("returnCountOnly", "true")
	countBody, err := fetchEnvironmentalBody(ctx, endpoint+"?"+countParams.Encode(), "application/json")
	if err != nil {
		return nil, fmt.Errorf("PAMGIA %s: %w", ibamaSourceLabel(current), err)
	}
	var countResp struct {
		Count int `json:"count"`
		Error *struct {
			Message string `json:"message"`
			Details []string `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(countBody, &countResp); err != nil {
		return nil, fmt.Errorf("PAMGIA %s retornou contagem inválida: %w", ibamaSourceLabel(current), err)
	}
	if countResp.Error != nil {
		return nil, fmt.Errorf("PAMGIA %s: %s", ibamaSourceLabel(current), strings.TrimSpace(countResp.Error.Message))
	}
	if countResp.Count == 0 {
		return []EmbargoFinding{}, nil
	}

	params := cloneURLValues(base)
	if current {
		params.Set("outFields", "num_tad,dat_embargo,sit_desmatamento,tipo_area,qtd_area_embargada,municipio,origem_geom,des_infracao")
	} else {
		params.Set("outFields", "numero_tad,data_tad,status_tad,sit_embarg,qtd_area_d,nom_munici,orgao,des_infrac")
	}
	params.Set("returnGeometry", "true")
	params.Set("outSR", "4326")
	params.Set("resultRecordCount", "200")
	params.Set("f", "geojson")

	body, err := fetchEnvironmentalBody(ctx, endpoint+"?"+params.Encode(), "application/geo+json,application/json")
	if err != nil {
		return nil, fmt.Errorf("PAMGIA %s: %w", ibamaSourceLabel(current), err)
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, fmt.Errorf("PAMGIA %s retornou GeoJSON inválido: %w", ibamaSourceLabel(current), err)
	}
	out := make([]EmbargoFinding, 0, len(fc.Features))
	for _, feature := range fc.Features {
		raw, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: feature.Properties, Geometry: feature.Geometry})
		intersection, carPct, _, err := estimateGeometryOverlap(carRaw, string(raw))
		if err != nil || intersection <= 0.0001 {
			continue
		}
		a := feature.Properties
		number := anyString(a, "num_tad")
		date := arcGISDateString(a["dat_embargo"])
		status := anyString(a, "sit_desmatamento")
		situation := anyString(a, "tipo_area")
		area := anyString(a, "qtd_area_embargada")
		municipality := anyString(a, "municipio")
		agency := "IBAMA"
		if !current {
			number = anyString(a, "numero_tad")
			date = arcGISDateString(a["data_tad"])
			status = anyString(a, "status_tad")
			situation = anyString(a, "sit_embarg")
			area = anyString(a, "qtd_area_d")
			municipality = anyString(a, "nom_munici")
			if x := anyString(a, "orgao"); x != "" {
				agency = x
			}
		}
		out = append(out, EmbargoFinding{
			Number: number, Date: date, Status: status, Situation: situation,
			Area: area, Municipality: municipality, Agency: agency,
			Infraction: anyString(a, "des_infrac"),
			OverlapAreaHa: intersection, OverlapCARPct: carPct, GeoJSON: string(raw),
		})
	}
	return out, nil
}

func ibamaSourceLabel(current bool) string {
	if current {
		return "embargos IBAMA atual"
	}
	return "embargos SISCOM legado"
}

func cloneURLValues(in url.Values) url.Values {
	out := url.Values{}
	for k, values := range in {
		for _, v := range values {
			out.Add(k, v)
		}
	}
	return out
}

func fetchEnvironmentalBody(ctx context.Context, target, accept string) ([]byte, error) {
	// Deixe o Go negociar HTTP/2/HTTP/1.1 normalmente. O PAMGIA pode responder
	// com frames HTTP/2 mesmo quando a conexão é forçada para HTTP/1.1; isso
	// resulta em "malformed HTTP response" no Windows.
	client := &http.Client{Timeout: 28 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 ViaVerdeCAR/"+AppVersion)

	var directErr error
	resp, err := client.Do(req)
	if err == nil {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
		resp.Body.Close()
		if readErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}
		if readErr != nil {
			directErr = readErr
		} else {
			directErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		}
	} else {
		directErr = err
	}

	// O curl.exe do Windows usa a pilha TLS/HTTP do sistema e já é usado como
	// contingência na consulta principal do CAR. Ir para ele imediatamente após
	// a primeira falha evita esperar dois timeouts seguidos do PAMGIA.
	if body, curlErr := fetchCARWithCurl(ctx, target); curlErr == nil {
		return body, nil
	} else if directErr != nil {
		return nil, fmt.Errorf("cliente nativo falhou (%v); fallback do Windows falhou (%v)", directErr, curlErr)
	} else {
		return nil, fmt.Errorf("fallback do Windows falhou: %v", curlErr)
	}
}

func queryFUNAITerritories(ctx context.Context, carRaw string) ([]TerritoryFinding, error) {
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		return nil, fmt.Errorf("limites do CAR indisponíveis")
	}
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "1.0.0")
	params.Set("request", "GetFeature")
	params.Set("typeName", "Funai:tis_poligonais")
	params.Set("outputFormat", "application/json")
	params.Set("srsName", "EPSG:4326")
	params.Set("maxFeatures", "100")
	params.Set("bbox", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f,EPSG:4326", minLon, minLat, maxLon, maxLat))
	body, err := fetchCARBody(ctx, funaiWFSURL+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, err
	}
	out := []TerritoryFinding{}
	for _, feature := range fc.Features {
		raw, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: feature.Properties, Geometry: feature.Geometry})
		intersection, carPct, _, err := estimateGeometryOverlap(carRaw, string(raw))
		if err != nil || intersection <= 0.0001 {
			continue
		}
		name := carStringProp(feature.Properties, "terrai_nom", "terra_nome", "nome", "nom_ti", "terrai")
		phase := carStringProp(feature.Properties, "fase_ti", "fase", "etapa", "situacao")
		out = append(out, TerritoryFinding{
			Name:          name,
			Phase:         phase,
			OverlapAreaHa: intersection,
			OverlapCARPct: carPct,
			Source:        "FUNAI",
			GeoJSON:       string(raw),
		})
	}
	return out, nil
}


func queryICMBioFederalUCs(ctx context.Context, carRaw string) ([]UCFindings, error) {
	minLon, minLat, maxLon, maxLat, ok := geoJSONBounds(carRaw)
	if !ok {
		return nil, fmt.Errorf("limites do CAR indisponíveis")
	}
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "1.1.0")
	params.Set("request", "GetFeature")
	params.Set("typeName", icmbioUCLayer)
	params.Set("outputFormat", "application/json")
	params.Set("srsName", "EPSG:4326")
	params.Set("maxFeatures", "100")
	params.Set("bbox", fmt.Sprintf("%.8f,%.8f,%.8f,%.8f,EPSG:4326", minLon, minLat, maxLon, maxLat))
	body, err := fetchCARBody(ctx, icmbioWFSURL+"?"+params.Encode())
	if err != nil {
		return nil, err
	}
	var fc carGeoJSON
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, err
	}
	out := []UCFindings{}
	for _, feature := range fc.Features {
		raw, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: feature.Properties, Geometry: feature.Geometry})
		intersection, carPct, _, err := estimateGeometryOverlap(carRaw, string(raw))
		if err != nil || intersection <= 0.0001 {
			continue
		}
		out = append(out, UCFindings{
			Name:          carStringProp(feature.Properties, "nome", "nome_uc", "nom_uc", "nm_uc"),
			Category:      carStringProp(feature.Properties, "categoria", "categoria_uc", "cat_uc"),
			Group:         carStringProp(feature.Properties, "grupo", "grupo_uc"),
			UF:            carStringProp(feature.Properties, "uf", "sigla_uf"),
			OverlapAreaHa: intersection,
			OverlapCARPct: carPct,
			Source:        "ICMBio/INDE",
			GeoJSON:       string(raw),
		})
	}
	return out, nil
}

func (a *App) enrichMCRScreening(ctx context.Context, car string, env *EnvironmentalSummary) {
	if env == nil || strings.TrimSpace(car) == "" {
		return
	}
	listed, fields, updated, err := a.lookupMCRList(ctx, car)
	if err != nil {
		env.Warnings = append(env.Warnings, "MMA/MCR: "+err.Error())
		return
	}
	env.MCRChecked = true
	env.MCRListed = listed
	env.MCRFields = fields
	env.MCRDataUpdated = updated
}

func (a *App) lookupMCRList(ctx context.Context, car string) (bool, map[string]string, string, error) {
	cacheDir := filepath.Join(a.dataDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return false, nil, "", err
	}
	zipPath := filepath.Join(cacheDir, "mma_mcr_lista.zip")
	metaPath := filepath.Join(cacheDir, "mma_mcr_meta.json")
	updated := ""
	needDownload := true
	if st, err := os.Stat(zipPath); err == nil && st.Size() > 1024 && time.Since(st.ModTime()) < 24*time.Hour {
		needDownload = false
		if raw, err := os.ReadFile(metaPath); err == nil {
			var meta map[string]string
			if json.Unmarshal(raw, &meta) == nil {
				updated = meta["last_modified"]
			}
		}
	}
	if needDownload {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, mmaMCRZipURL, nil)
		if err != nil {
			return false, nil, "", err
		}
		req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
		resp, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
		if err != nil {
			return false, nil, "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, nil, "", fmt.Errorf("download da lista retornou HTTP %d", resp.StatusCode)
		}
		tmp := zipPath + ".part"
		f, err := os.Create(tmp)
		if err != nil {
			return false, nil, "", err
		}
		n, copyErr := io.Copy(f, io.LimitReader(resp.Body, 250<<20))
		closeErr := f.Close()
		if copyErr != nil || closeErr != nil || n < 1024 {
			_ = os.Remove(tmp)
			if copyErr != nil {
				return false, nil, "", copyErr
			}
			if closeErr != nil {
				return false, nil, "", closeErr
			}
			return false, nil, "", fmt.Errorf("arquivo MMA/MCR inválido")
		}
		_ = os.Remove(zipPath)
		if err := os.Rename(tmp, zipPath); err != nil {
			return false, nil, "", err
		}
		updated = strings.TrimSpace(resp.Header.Get("Last-Modified"))
		meta, _ := json.Marshal(map[string]string{"last_modified": updated, "downloaded_at": time.Now().Format(time.RFC3339)})
		_ = os.WriteFile(metaPath, meta, 0o644)
	}
	return scanMCRZip(zipPath, car, updated)
}

func scanMCRZip(zipPath, car, updated string) (bool, map[string]string, string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return false, nil, updated, err
	}
	defer zr.Close()
	targetFull := strings.ToUpper(strings.TrimSpace(car))
	targetCompact := strings.ReplaceAll(targetFull, ".", "")
	supportedFiles := 0

	for _, zf := range zr.File {
		name := strings.ToLower(zf.Name)
		switch {
		case strings.HasSuffix(name, ".csv") || strings.HasSuffix(name, ".txt"):
			supportedFiles++
			r, err := zf.Open()
			if err != nil {
				continue
			}
			scanner := bufio.NewScanner(r)
			scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
			var header string
			if scanner.Scan() {
				header = scanner.Text()
			}
			for scanner.Scan() {
				line := scanner.Text()
				upper := strings.ToUpper(line)
				compact := strings.ReplaceAll(upper, ".", "")
				if !strings.Contains(upper, targetFull) && !strings.Contains(compact, targetCompact) {
					continue
				}
				fields := rowToMap(header, line)
				r.Close()
				return true, fields, updated, nil
			}
			r.Close()

		case strings.HasSuffix(name, ".dbf"):
			supportedFiles++
			r, err := zf.Open()
			if err != nil {
				continue
			}
			found, fields, scanErr := scanDBFForCAR(r, targetFull, targetCompact)
			r.Close()
			if scanErr != nil {
				return false, nil, updated, scanErr
			}
			if found {
				return true, fields, updated, nil
			}
		}
	}
	if supportedFiles == 0 {
		return false, nil, updated, errors.New("pacote MMA/MCR sem CSV, TXT ou DBF pesquisável; resultado não pode ser classificado como não listado")
	}
	return false, map[string]string{}, updated, nil
}

type dbfField struct {
	Name   string
	Length int
}

func scanDBFForCAR(r io.Reader, targetFull, targetCompact string) (bool, map[string]string, error) {
	header := make([]byte, 32)
	if _, err := io.ReadFull(r, header); err != nil {
		return false, nil, err
	}
	numRecords := int(binary.LittleEndian.Uint32(header[4:8]))
	headerLen := int(binary.LittleEndian.Uint16(header[8:10]))
	recordLen := int(binary.LittleEndian.Uint16(header[10:12]))
	if numRecords < 0 || headerLen < 33 || recordLen < 2 || recordLen > 1<<20 {
		return false, nil, errors.New("DBF inválido no pacote MMA/MCR")
	}
	fieldBytes := headerLen - 33
	if fieldBytes < 0 || fieldBytes%32 != 0 {
		return false, nil, errors.New("cabeçalho DBF inválido no pacote MMA/MCR")
	}
	fields := make([]dbfField, 0, fieldBytes/32)
	for i := 0; i < fieldBytes/32; i++ {
		desc := make([]byte, 32)
		if _, err := io.ReadFull(r, desc); err != nil {
			return false, nil, err
		}
		nameBytes := desc[:11]
		if idx := bytes.IndexByte(nameBytes, 0); idx >= 0 {
			nameBytes = nameBytes[:idx]
		}
		name := strings.TrimSpace(string(nameBytes))
		if name == "" {
			name = fmt.Sprintf("campo_%d", i+1)
		}
		fields = append(fields, dbfField{Name: name, Length: int(desc[16])})
	}
	terminator := make([]byte, 1)
	if _, err := io.ReadFull(r, terminator); err != nil {
		return false, nil, err
	}
	if terminator[0] != 0x0D {
		return false, nil, errors.New("terminador de cabeçalho DBF ausente")
	}

	record := make([]byte, recordLen)
	for row := 0; row < numRecords; row++ {
		if _, err := io.ReadFull(r, record); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return false, nil, nil
			}
			return false, nil, err
		}
		if record[0] == '*' {
			continue
		}
		lineUpper := strings.ToUpper(string(record[1:]))
		lineCompact := strings.ReplaceAll(lineUpper, ".", "")
		if !strings.Contains(lineUpper, targetFull) && !strings.Contains(lineCompact, targetCompact) {
			continue
		}
		out := map[string]string{}
		offset := 1
		for _, field := range fields {
			if offset+field.Length > len(record) {
				break
			}
			out[field.Name] = strings.TrimSpace(string(record[offset : offset+field.Length]))
			offset += field.Length
		}
		return true, out, nil
	}
	return false, map[string]string{}, nil
}

func rowToMap(header, line string) map[string]string {
	delimiter := ";"
	if strings.Count(header, ",") > strings.Count(header, ";") {
		delimiter = ","
	}
	heads := strings.Split(header, delimiter)
	vals := strings.Split(line, delimiter)
	out := map[string]string{}
	for i, h := range heads {
		h = strings.Trim(strings.TrimSpace(h), "\"")
		if h == "" {
			h = fmt.Sprintf("campo_%d", i+1)
		}
		v := ""
		if i < len(vals) {
			v = strings.Trim(strings.TrimSpace(vals[i]), "\"")
		}
		out[h] = v
	}
	return out
}

func geoJSONToEsriPolygon(raw string) (map[string]any, error) {
	var f carGeoFeature
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return nil, err
	}
	polys, err := carGeometryPolygons(f.Geometry)
	if err != nil {
		return nil, err
	}
	rings := make([][][]float64, 0)
	for _, poly := range polys {
		for _, ring := range poly {
			if len(ring) < 3 {
				continue
			}
			copied := make([][]float64, 0, len(ring)+1)
			for _, p := range ring {
				if len(p) >= 2 {
					copied = append(copied, []float64{p[0], p[1]})
				}
			}
			if len(copied) >= 3 {
				first, last := copied[0], copied[len(copied)-1]
				if first[0] != last[0] || first[1] != last[1] {
					copied = append(copied, []float64{first[0], first[1]})
				}
				rings = append(rings, copied)
			}
		}
	}
	if len(rings) == 0 {
		return nil, fmt.Errorf("geometria do CAR vazia")
	}
	return map[string]any{"rings": rings, "spatialReference": map[string]any{"wkid": 4326}}, nil
}

func geoJSONBounds(raw string) (minLon, minLat, maxLon, maxLat float64, ok bool) {
	var f carGeoFeature
	if json.Unmarshal([]byte(raw), &f) != nil {
		return
	}
	polys, err := carGeometryPolygons(f.Geometry)
	if err != nil {
		return
	}
	minLon, minLat = math.Inf(1), math.Inf(1)
	maxLon, maxLat = math.Inf(-1), math.Inf(-1)
	for _, poly := range polys {
		for _, ring := range poly {
			for _, p := range ring {
				if len(p) < 2 {
					continue
				}
				minLon, minLat = math.Min(minLon, p[0]), math.Min(minLat, p[1])
				maxLon, maxLat = math.Max(maxLon, p[0]), math.Max(maxLat, p[1])
				ok = true
			}
		}
	}
	return
}

func anyString(m map[string]any, key string) string {
	for k, v := range m {
		if strings.EqualFold(k, key) && v != nil {
			return strings.TrimSpace(fmt.Sprint(v))
		}
	}
	return ""
}

func arcGISDateString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case float64:
		if x <= 0 {
			return ""
		}
		return time.UnixMilli(int64(x)).Format("02/01/2006")
	case int64:
		if x <= 0 {
			return ""
		}
		return time.UnixMilli(x).Format("02/01/2006")
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
