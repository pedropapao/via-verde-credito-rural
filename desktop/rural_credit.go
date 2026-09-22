package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	mapBiomasRuralCreditURL = "https://plataforma.creditorural.mapbiomas.org/"
	bcbRuralDataURL          = "https://www.bcb.gov.br/estabilidadefinanceira/micrrural/"
	bcbSicorODataBase        = "https://olinda.bcb.gov.br/olinda/servico/SICOR/versao/v2/odata/"
)

type BCBRuralCreditRow struct {
	Kind      string  `json:"kind"`
	Year      string  `json:"year"`
	Product   string  `json:"product"`
	Contracts float64 `json:"contracts"`
	Value     float64 `json:"value"`
	RawLabel  string  `json:"raw_label"`
}

type BCBRuralCreditContext struct {
	Available    bool                `json:"available"`
	ExternalOnly bool                `json:"external_only"`
	Municipality string              `json:"municipality"`
	UF           string              `json:"uf"`
	Rows         []BCBRuralCreditRow `json:"rows"`
	Contracts    float64             `json:"contracts"`
	Value        float64             `json:"value"`
	Message      string              `json:"message"`
	SourceURL    string              `json:"source_url"`
}

type RuralCreditOverview struct {
	CAR                    string                `json:"car"`
	Municipality           string                `json:"municipality"`
	UF                     string                `json:"uf"`
	MapBiomasMonitorURL    string                `json:"mapbiomas_monitor_url"`
	MapBiomasAlertProperty string                `json:"mapbiomas_alert_property_url"`
	BCBSourceURL           string                `json:"bcb_source_url"`
	BCB                    BCBRuralCreditContext `json:"bcb"`
	Alerts                 MapBiomasCARSummary   `json:"alerts"`
	Warnings               []string              `json:"warnings"`
	CheckedAt              string                `json:"checked_at"`
}

func (a *App) GetRuralCreditOverview(propertyID int64) (RuralCreditOverview, error) {
	car, err := a.carForRuralCredit(propertyID)
	if err != nil {
		return RuralCreditOverview{}, err
	}
	out := RuralCreditOverview{
		CAR: car.CAR, Municipality: car.Municipality, UF: car.UF,
		MapBiomasMonitorURL: mapBiomasRuralCreditURL,
		MapBiomasAlertProperty: "https://plataforma.alerta.mapbiomas.org/imovel/" + url.PathEscape(car.CAR),
		BCBSourceURL: bcbRuralDataURL,
		CheckedAt: time.Now().Format(time.RFC3339),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
	defer cancel()
	type bcbResult struct {
		v   BCBRuralCreditContext
		err error
	}
	type alertResult struct {
		v   MapBiomasCARSummary
		err error
	}
	bcbCh := make(chan bcbResult, 1)
	alertCh := make(chan alertResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				bcbCh <- bcbResult{err: fmt.Errorf("falha interna isolada na consulta BCB/SICOR: %v", r)}
			}
		}()
		v, e := queryBCBRuralMunicipality(ctx, car.Municipality, car.UF, car.MunicipalityCode)
		bcbCh <- bcbResult{v: v, err: e}
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				alertCh <- alertResult{err: fmt.Errorf("falha interna isolada na consulta MapBiomas: %v", r)}
			}
		}()
		v, e := a.QueryMapBiomasCAR(car.CAR)
		alertCh <- alertResult{v: v, err: e}
	}()
	b := <-bcbCh
	m := <-alertCh
	if b.err != nil {
		out.Warnings = append(out.Warnings, "Contexto municipal BCB/SICOR indisponível: "+b.err.Error())
		out.BCB = BCBRuralCreditContext{
			Municipality: car.Municipality, UF: car.UF, SourceURL: bcbRuralDataURL,
			Message: "Não foi possível consultar o contexto agregado do município nesta tentativa.",
		}
	} else {
		out.BCB = b.v
	}
	if m.err != nil {
		out.Warnings = append(out.Warnings, "MapBiomas Alerta: "+m.err.Error())
		out.Alerts = MapBiomasCARSummary{Connected: a.GetMapBiomasAlertStatus().Connected, Message: m.err.Error()}
	} else {
		out.Alerts = m.v
	}
	return out, nil
}

func (a *App) carForRuralCredit(propertyID int64) (CARResult, error) {
	if propertyID > 0 {
		r, err := a.GetLatestCAR(propertyID)
		if err != nil || strings.TrimSpace(r.CAR) == "" {
			return CARResult{}, errors.New("consulte o CAR deste imóvel antes de abrir Crédito Rural")
		}
		return r, nil
	}
	cache, err := a.GetLastCARSession()
	if err != nil || strings.TrimSpace(cache.Result.CAR) == "" {
		return CARResult{}, errors.New("consulte um CAR antes de abrir Crédito Rural")
	}
	return cache.Result, nil
}

func queryBCBRuralMunicipality(ctx context.Context, municipality, uf, municipalityCode string) (BCBRuralCreditContext, error) {
	out := BCBRuralCreditContext{
		Municipality: municipality,
		UF:           uf,
		SourceURL:    "https://dadosabertos.bcb.gov.br/dataset/matrizdadoscreditorural",
	}
	years := []int{time.Now().Year(), time.Now().Year() - 1}
	var raw []map[string]any
	var failures []string
	successfulYears := 0

	for _, year := range years {
		rows, err := queryBCBMunicipalityAggregate(ctx, municipality, uf, municipalityCode, year)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%d: %v", year, err))
			continue
		}
		successfulYears++
		raw = append(raw, rows...)
	}
	if len(raw) > 0 {
		out.Rows = aggregateBCBMunicipalityRows(raw)
		sort.SliceStable(out.Rows, func(i, j int) bool {
			if out.Rows[i].Year == out.Rows[j].Year {
				if out.Rows[i].Kind == out.Rows[j].Kind {
					return out.Rows[i].Value > out.Rows[j].Value
				}
				return out.Rows[i].Kind < out.Rows[j].Kind
			}
			return out.Rows[i].Year > out.Rows[j].Year
		})
		for _, row := range out.Rows {
			out.Contracts += row.Contracts
			out.Value += row.Value
		}
		out.Available = true
		out.ExternalOnly = false
		out.Message = "Consulta automática ao SICOR/MDCR concluída para o município do CAR, usando os dois anos mais recentes disponíveis na API pública."
		return out, nil
	}

	if successfulYears > 0 && len(failures) == 0 {
		out.Available = true
		out.ExternalOnly = false
		out.Message = "A consulta automática ao SICOR/MDCR foi concluída, mas não retornou registros para este município nos dois anos consultados."
		return out, nil
	}

	out.ExternalOnly = true
	if len(failures) > 0 {
		out.Message = "A consulta automática ao SICOR/MDCR foi tentada, mas a API pública não aceitou ou não concluiu o filtro municipal nesta tentativa. A fonte oficial continua disponível como contingência."
		return out, errors.New(strings.Join(failures, " | "))
	}
	out.Message = "A consulta automática ao SICOR/MDCR não retornou resposta utilizável. A fonte oficial continua disponível como contingência."
	return out, nil
}

func queryBCBMunicipalityAggregate(ctx context.Context, municipality, uf, municipalityCode string, year int) ([]map[string]any, error) {
	const endpoint = "CusteioInvestimentoComercialIndustrialSemFiltros"
	yearText := strconv.Itoa(year)
	name := strings.ToUpper(strings.TrimSpace(municipality))
	uf = strings.ToUpper(strings.TrimSpace(uf))
	code := strings.TrimSpace(municipalityCode)

	var filters []string
	if code != "" {
		filters = append(filters,
			fmt.Sprintf("codMunicIbge eq '%s' and AnoEmissao eq '%s'", odataEscape(code), yearText),
		)
		if _, err := strconv.ParseInt(code, 10, 64); err == nil {
			filters = append(filters,
				fmt.Sprintf("codMunicIbge eq %s and AnoEmissao eq '%s'", code, yearText),
			)
		}
	}
	if name != "" && uf != "" {
		filters = append(filters,
			fmt.Sprintf("Municipio eq '%s' and nomeUF eq '%s' and AnoEmissao eq '%s'", odataEscape(name), odataEscape(uf), yearText),
		)
	}
	if name != "" {
		filters = append(filters,
			fmt.Sprintf("Municipio eq '%s' and AnoEmissao eq '%s'", odataEscape(name), yearText),
		)
	}
	if len(filters) == 0 {
		return nil, errors.New("município sem código ou nome para consulta")
	}

	var lastErr error
	sawEmptyFilteredResponse := false
	sawIgnoredFilter := false
	for _, filter := range filters {
		q := url.Values{}
		q.Set("$format", "json")
		q.Set("$top", "5000")
		q.Set("$filter", filter)
		target := bcbSicorODataBase + endpoint + "?" + q.Encode()
		raw, err := getODataRows(ctx, target)
		if err != nil {
			lastErr = err
			continue
		}
		if len(raw) == 0 {
			// Uma resposta vazia com HTTP 200 é compatível com filtro aceito e
			// município sem registros naquele ano. Tentamos os filtros seguintes
			// para contornar diferenças de tipagem/capitalização do serviço.
			sawEmptyFilteredResponse = true
			continue
		}
		matched := filterBCBMunicipalityRows(raw, municipality, uf, municipalityCode, yearText)
		if len(matched) > 0 {
			return matched, nil
		}
		// Se a API devolveu linhas, mas nenhuma pertence ao município/ano
		// solicitado, o deployment provavelmente ignorou o $filter. Nunca
		// tratamos essas linhas como dados do município.
		sawIgnoredFilter = true
	}
	if sawEmptyFilteredResponse && !sawIgnoredFilter {
		return nil, nil
	}
	if sawIgnoredFilter {
		return nil, errors.New("a API do SICOR retornou dados de outros municípios e não respeitou o filtro solicitado")
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("consulta municipal do BCB/SICOR sem resposta")
}

func filterBCBMunicipalityRows(rows []map[string]any, municipality, uf, municipalityCode, year string) []map[string]any {
	wantName := strings.ToUpper(strings.TrimSpace(municipality))
	wantUF := strings.ToUpper(strings.TrimSpace(uf))
	wantCode := strings.TrimLeft(strings.TrimSpace(municipalityCode), "0")
	if wantCode == "" && strings.TrimSpace(municipalityCode) != "" {
		wantCode = "0"
	}
	var out []map[string]any
	for _, r := range rows {
		if y := strings.TrimSpace(firstStringMapValue(r, "AnoEmissao")); year != "" && y != year {
			continue
		}
		rowCode := strings.TrimLeft(strings.TrimSpace(firstNonEmptyStringMapValue(r, "codMunicIbge", "codIbge", "CD_IBGE_MUNICIPIO")), "0")
		if rowCode == "" && firstNonEmptyStringMapValue(r, "codMunicIbge", "codIbge", "CD_IBGE_MUNICIPIO") != "" {
			rowCode = "0"
		}
		rowName := strings.ToUpper(strings.TrimSpace(firstNonEmptyStringMapValue(r, "Municipio", "municipio")))
		rowUF := strings.ToUpper(strings.TrimSpace(firstNonEmptyStringMapValue(r, "nomeUF", "UF", "uf")))
		codeMatch := wantCode != "" && rowCode != "" && rowCode == wantCode
		nameMatch := wantName != "" && rowName == wantName && (wantUF == "" || rowUF == "" || rowUF == wantUF)
		if codeMatch || nameMatch {
			out = append(out, r)
		}
	}
	return out
}

func aggregateBCBMunicipalityRows(rows []map[string]any) []BCBRuralCreditRow {
	type key struct {
		Kind, Year, Activity string
	}
	grouped := map[key]*BCBRuralCreditRow{}
	for _, r := range rows {
		year := firstNonEmptyStringMapValue(r, "AnoEmissao", "ano")
		activityCode := firstNonEmptyStringMapValue(r, "Atividade", "atividade")
		activity := bcbActivityLabel(activityCode)

		for _, spec := range []struct {
			kind, qty, value string
		}{
			{"Custeio", "QtdCusteio", "VlCusteio"},
			{"Investimento", "QtdInvestimento", "VlInvestimento"},
		} {
			qty := numberMapValue(r, spec.qty)
			value := numberMapValue(r, spec.value)
			if qty == 0 && value == 0 {
				continue
			}
			k := key{Kind: spec.kind, Year: year, Activity: activity}
			if grouped[k] == nil {
				grouped[k] = &BCBRuralCreditRow{
					Kind: spec.kind, Year: year, Product: activity,
					RawLabel: "CusteioInvestimentoComercialIndustrialSemFiltros",
				}
			}
			grouped[k].Contracts += qty
			grouped[k].Value += value
		}
	}
	out := make([]BCBRuralCreditRow, 0, len(grouped))
	for _, row := range grouped {
		out = append(out, *row)
	}
	return out
}

func bcbActivityLabel(code string) string {
	switch strings.TrimSpace(code) {
	case "1":
		return "Atividade agrícola"
	case "2":
		return "Atividade pecuária"
	case "":
		return "Atividade não informada"
	default:
		return "Atividade " + strings.TrimSpace(code)
	}
}

func firstNonEmptyStringMapValue(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := firstStringMapValue(m, key); strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func getODataRows(ctx context.Context, target string) ([]map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	resp, err := (&http.Client{Timeout: 18 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("BCB/SICOR respondeu HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Value, nil
}

func firstStringMapValue(m map[string]any, key string) string {
	if key == "" {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprint(x)
	}
}

func numberMapValue(m map[string]any, key string) float64 {
	if key == "" {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return x
	case string:
		x = strings.ReplaceAll(x, ".", "")
		x = strings.ReplaceAll(x, ",", ".")
		n, _ := strconv.ParseFloat(x, 64)
		return n
	default:
		n, _ := strconv.ParseFloat(fmt.Sprint(x), 64)
		return n
	}
}

func odataEscape(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}
