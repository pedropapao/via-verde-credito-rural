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
	bcbRuralDataURL          = "https://dadosabertos.bcb.gov.br/dataset/matrizdadoscreditorural"
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
		Municipality: municipality, UF: uf, SourceURL: bcbRuralDataURL,
	}
	endpoints := []struct{ name, kind string }{
		{"CusteioMunicipioProduto", "Custeio"},
		{"InvestMunicipioProduto", "Investimento"},
	}
	for _, ep := range endpoints {
		rows, err := queryBCBAdaptiveEndpoint(ctx, ep.name, ep.kind, municipality, uf, municipalityCode)
		if err != nil {
			continue
		}
		out.Rows = append(out.Rows, rows...)
	}
	if len(out.Rows) == 0 {
		out.Message = "A API pública agregada do BCB não retornou linhas municipais compatíveis nesta tentativa."
		return out, nil
	}
	sort.SliceStable(out.Rows, func(i, j int) bool {
		if out.Rows[i].Year == out.Rows[j].Year {
			return out.Rows[i].Value > out.Rows[j].Value
		}
		return out.Rows[i].Year > out.Rows[j].Year
	})
	if len(out.Rows) > 20 {
		out.Rows = out.Rows[:20]
	}
	for _, row := range out.Rows {
		out.Contracts += row.Contracts
		out.Value += row.Value
	}
	out.Available = true
	out.Message = "Contexto agregado do município na Matriz de Dados do Crédito Rural do BCB. Não representa operações específicas deste CAR."
	return out, nil
}

func queryBCBAdaptiveEndpoint(ctx context.Context, endpoint, kind, municipality, uf, municipalityCode string) ([]BCBRuralCreditRow, error) {
	sampleURL := bcbSicorODataBase + endpoint + "?%24top=1&%24format=json"
	sample, err := getODataRows(ctx, sampleURL)
	if err != nil || len(sample) == 0 {
		return nil, err
	}
	fields := detectBCBFields(sample[0])
	munField := fields["municipality"]
	ufField := fields["uf"]
	if munField == "" {
		return nil, errors.New("campo municipal não identificado no recurso " + endpoint)
	}
	filterParts := []string{}
	sampleValue := sample[0][munField]
	if municipalityCode != "" && (strings.Contains(strings.ToLower(munField), "cod") || strings.Contains(strings.ToLower(munField), "ibge")) {
		if _, ok := sampleValue.(float64); ok {
			if n, e := strconv.ParseFloat(municipalityCode, 64); e == nil {
				filterParts = append(filterParts, fmt.Sprintf("%s eq %.0f", munField, n))
			}
		} else {
			filterParts = append(filterParts, fmt.Sprintf("%s eq '%s'", munField, odataEscape(municipalityCode)))
		}
	}
	if len(filterParts) == 0 {
		filterParts = append(filterParts, fmt.Sprintf("%s eq '%s'", munField, odataEscape(strings.ToUpper(strings.TrimSpace(municipality)))))
	}
	if ufField != "" && strings.TrimSpace(uf) != "" {
		filterParts = append(filterParts, fmt.Sprintf("%s eq '%s'", ufField, odataEscape(strings.ToUpper(strings.TrimSpace(uf)))))
	}
	q := url.Values{}
	q.Set("$top", "80")
	q.Set("$format", "json")
	q.Set("$filter", strings.Join(filterParts, " and "))
	target := bcbSicorODataBase + endpoint + "?" + q.Encode()
	raw, err := getODataRows(ctx, target)
	if err != nil {
		// Alguns recursos gravam o nome municipal com capitalização diferente.
		q.Set("$filter", fmt.Sprintf("contains(toupper(%s),'%s')", munField, odataEscape(strings.ToUpper(strings.TrimSpace(municipality)))))
		target = bcbSicorODataBase + endpoint + "?" + q.Encode()
		raw, err = getODataRows(ctx, target)
	}
	if err != nil {
		return nil, err
	}
	var out []BCBRuralCreditRow
	for _, r := range raw {
		row := BCBRuralCreditRow{Kind: kind}
		row.Year = firstStringMapValue(r, fields["year"])
		row.Product = firstStringMapValue(r, fields["product"])
		row.Contracts = numberMapValue(r, fields["contracts"])
		row.Value = numberMapValue(r, fields["value"])
		if row.Product == "" {
			row.Product = "Produto não identificado"
		}
		row.RawLabel = endpoint
		out = append(out, row)
	}
	return out, nil
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

func detectBCBFields(sample map[string]any) map[string]string {
	out := map[string]string{}
	for key := range sample {
		k := strings.ToLower(key)
		switch {
		case out["municipality"] == "" && (strings.Contains(k, "municip") || strings.Contains(k, "ibge")):
			out["municipality"] = key
		case out["uf"] == "" && (k == "uf" || strings.Contains(k, "siglauf") || strings.Contains(k, "estado")):
			out["uf"] = key
		case out["year"] == "" && strings.Contains(k, "ano"):
			out["year"] = key
		case out["product"] == "" && (strings.Contains(k, "produto") || strings.Contains(k, "atividade")):
			out["product"] = key
		case out["contracts"] == "" && (strings.Contains(k, "qtd") || strings.Contains(k, "quant")) && strings.Contains(k, "contr"):
			out["contracts"] = key
		case out["value"] == "" && (strings.Contains(k, "valor") || strings.HasPrefix(k, "vl")):
			out["value"] = key
		}
	}
	return out
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
