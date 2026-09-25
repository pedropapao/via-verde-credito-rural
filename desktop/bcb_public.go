package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	bcbOpenDataURL          = "https://dadosabertos.bcb.gov.br/"
	bcbMDCRDatasetURL       = "https://dadosabertos.bcb.gov.br/dataset/matrizdadoscreditorural"
	bcbEntitiesDatasetURL   = "https://dadosabertos.bcb.gov.br/dataset/dados-cadastrais-de-entidades-autorizadas"
	bcbIFDataDatasetURL     = "https://dadosabertos.bcb.gov.br/dataset/ifdata---dados-selecionados-de-instituies-financeiras"
	bcbSCRDatasetURL        = "https://dadosabertos.bcb.gov.br/dataset/scr_data"
	bcbSGSBaseURL           = "https://api.bcb.gov.br/dados/serie/bcdata.sgs."
	bcbBcBaseODataBase      = "https://olinda.bcb.gov.br/olinda/servico/BcBase/versao/v2/odata/"
	bcbIFDataODataBase      = "https://olinda.bcb.gov.br/olinda/servico/IFDATA/versao/v1/odata/"
	bcbPublicCacheAge       = 8 * time.Hour
)

type BCBSeriesPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type BCBSeriesMetric struct {
	Key           string           `json:"key"`
	Label         string           `json:"label"`
	PersonType    string           `json:"person_type"`
	Measure       string           `json:"measure"`
	Unit          string           `json:"unit"`
	SGSCode       int              `json:"sgs_code"`
	Status        string           `json:"status"`
	LatestDate    string           `json:"latest_date"`
	LatestValue   float64          `json:"latest_value"`
	PreviousValue float64          `json:"previous_value"`
	ChangePct     float64          `json:"change_pct"`
	SourceURL     string           `json:"source_url"`
	Points        []BCBSeriesPoint `json:"points"`
	Warning       string           `json:"warning"`
}

type BCBSeriesContext struct {
	Available bool              `json:"available"`
	Metrics   []BCBSeriesMetric `json:"metrics"`
	Message   string            `json:"message"`
	SourceURL string            `json:"source_url"`
	Warnings  []string          `json:"warnings"`
}

type BCBInstitutionRecord struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	CNPJ         string `json:"cnpj"`
	Type         string `json:"type"`
	Segment      string `json:"segment"`
	Situation    string `json:"situation"`
	UF           string `json:"uf"`
	Municipality string `json:"municipality"`
}

type BCBInstitutionContext struct {
	Available    bool                   `json:"available"`
	Reference    string                 `json:"reference"`
	Institutions []BCBInstitutionRecord `json:"institutions"`
	Message      string                 `json:"message"`
	SourceURL    string                 `json:"source_url"`
	Warnings     []string               `json:"warnings"`
}

type BCBIFDataRecord struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Reference    string `json:"reference"`
	Segment      string `json:"segment"`
	Activity     string `json:"activity"`
	Situation    string `json:"situation"`
	UF           string `json:"uf"`
	Municipality string `json:"municipality"`
	LeaderCNPJ   string `json:"leader_cnpj"`
}

type BCBIFDataContext struct {
	Available    bool              `json:"available"`
	Reference    string            `json:"reference"`
	Institutions []BCBIFDataRecord `json:"institutions"`
	Message      string            `json:"message"`
	SourceURL    string            `json:"source_url"`
	Warnings     []string          `json:"warnings"`
}

type BCBOpenDataSource struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
	SourceURL string `json:"source_url"`
	Automatic bool  `json:"automatic"`
}

type BCBPublicContext struct {
	Available     bool                  `json:"available"`
	UsedCache     bool                  `json:"used_cache"`
	GeneratedAt   string                `json:"generated_at"`
	Municipality  string                `json:"municipality"`
	UF            string                `json:"uf"`
	Market        CreditMarketContext   `json:"market"`
	Series        BCBSeriesContext      `json:"series"`
	Institutions  BCBInstitutionContext `json:"institutions"`
	IFData        BCBIFDataContext      `json:"ifdata"`
	Sources       []BCBOpenDataSource   `json:"sources"`
	Warnings      []string              `json:"warnings"`
	SourceURL     string                `json:"source_url"`
	Interpretation string               `json:"interpretation"`
}

type bcbSeriesSpec struct {
	Key, Label, PersonType, Measure, Unit string
	Code                                  int
}

var bcbRuralSeriesSpecs = []bcbSeriesSpec{
	{Key: "pf_rate_total_annual", Label: "Taxa média rural PF - total", PersonType: "PF", Measure: "Taxa de juros", Unit: "% a.a.", Code: 20771},
	{Key: "pj_rate_total_annual", Label: "Taxa média rural PJ - total", PersonType: "PJ", Measure: "Taxa de juros", Unit: "% a.a.", Code: 20760},
	{Key: "pf_rate_market_monthly", Label: "Taxa rural PF - recursos livres/mercado", PersonType: "PF", Measure: "Taxa de juros", Unit: "% a.m.", Code: 25494},
	{Key: "pf_rate_regulated_monthly", Label: "Taxa rural PF - recursos regulados", PersonType: "PF", Measure: "Taxa de juros", Unit: "% a.m.", Code: 25495},
	{Key: "pj_rate_market_monthly", Label: "Taxa rural PJ - recursos livres/mercado", PersonType: "PJ", Measure: "Taxa de juros", Unit: "% a.m.", Code: 25483},
	{Key: "pj_rate_regulated_monthly", Label: "Taxa rural PJ - recursos regulados", PersonType: "PJ", Measure: "Taxa de juros", Unit: "% a.m.", Code: 25484},
	{Key: "pf_balance", Label: "Saldo de crédito rural PF", PersonType: "PF", Measure: "Saldo", Unit: "R$ milhões", Code: 20609},
	{Key: "pj_balance", Label: "Saldo de crédito rural PJ", PersonType: "PJ", Measure: "Saldo", Unit: "R$ milhões", Code: 20597},
	{Key: "pf_concessions", Label: "Concessões de crédito rural PF", PersonType: "PF", Measure: "Concessões", Unit: "R$ milhões", Code: 20701},
	{Key: "pj_concessions", Label: "Concessões de crédito rural PJ", PersonType: "PJ", Measure: "Concessões", Unit: "R$ milhões", Code: 20689},
	{Key: "pf_delinquency", Label: "Inadimplência do crédito rural PF", PersonType: "PF", Measure: "Inadimplência", Unit: "%", Code: 21148},
	{Key: "pj_delinquency", Label: "Inadimplência do crédito rural PJ", PersonType: "PJ", Measure: "Inadimplência", Unit: "%", Code: 21136},
}

// GetBCBPublicContext consolidates only public/open BCB datasets. It does not
// query individual debt, SCR reports by CPF/CNPJ or any protected banking data.
func (a *App) GetBCBPublicContext(propertyID int64, force bool) (BCBPublicContext, error) {
	car, err := a.carForRuralCredit(propertyID)
	if err != nil {
		return BCBPublicContext{}, err
	}
	cachePath := ""
	if a != nil && strings.TrimSpace(a.dataDir) != "" {
		cachePath = filepath.Join(a.dataDir, "cache", "bcb_public", safeFilePart(car.UF+"_"+car.Municipality+"_"+car.MunicipalityCode)+".json")
		if !force {
			if cached, ok := loadBCBPublicCache(cachePath, bcbPublicCacheAge); ok {
				cached.UsedCache = true
				return cached, nil
			}
		}
	}

	out := BCBPublicContext{
		GeneratedAt:  time.Now().Format(time.RFC3339),
		Municipality: car.Municipality,
		UF:           car.UF,
		SourceURL:    bcbOpenDataURL,
		Interpretation: "Contexto público e agregado do Banco Central. Os dados ajudam a conferir mercado, taxas e instituições, mas não representam limite, aprovação, dívida ou risco individual do produtor.",
		Sources: []BCBOpenDataSource{
			{Key: "mdcr", Label: "MDCR / SICOR", Status: "pending", Detail: "Mercado agregado de crédito rural por município e UF.", SourceURL: bcbMDCRDatasetURL, Automatic: true},
			{Key: "sgs", Label: "SGS / Séries rurais", Status: "pending", Detail: "Taxas, saldos, concessões e inadimplência agregadas.", SourceURL: "https://www.bcb.gov.br/estatisticas/indecoreestruturacao", Automatic: true},
			{Key: "entities", Label: "Entidades supervisionadas", Status: "pending", Detail: "Cadastro oficial de instituições autorizadas pelo BCB.", SourceURL: bcbEntitiesDatasetURL, Automatic: true},
			{Key: "ifdata", Label: "IFData", Status: "pending", Detail: "Cadastro trimestral e contexto das instituições financeiras.", SourceURL: bcbIFDataDatasetURL, Automatic: true},
			{Key: "scr", Label: "SCR.data", Status: "on_demand", Detail: "Base mensal agregada por UF. Não é baixada automaticamente no Raio X porque os arquivos são volumosos e não permitem consulta de dívida individual por CPF/CNPJ.", SourceURL: bcbSCRDatasetURL, Automatic: false},
		},
	}

	baseCtx := context.Background()
	if a != nil && a.ctx != nil {
		baseCtx = a.ctx
	}
	ctx, cancel := context.WithTimeout(baseCtx, 65*time.Second)
	defer cancel()

	type marketResult struct {
		v CreditMarketContext
		e error
	}
	type seriesResult struct {
		v BCBSeriesContext
		e error
	}
	marketCh := make(chan marketResult, 1)
	seriesCh := make(chan seriesResult, 1)
	go func() {
		v, e := queryCreditMarketContext(ctx, car.Municipality, car.UF, car.MunicipalityCode)
		marketCh <- marketResult{v: v, e: e}
	}()
	go func() {
		v, e := queryBCBRuralSeries(ctx)
		seriesCh <- seriesResult{v: v, e: e}
	}()

	mr, sr := <-marketCh, <-seriesCh
	out.Market = mr.v
	out.Series = sr.v
	if mr.e != nil {
		out.Warnings = append(out.Warnings, "MDCR/SICOR: "+mr.e.Error())
		setBCBSourceStatus(&out, "mdcr", "unavailable", mr.v.Message)
	} else if mr.v.Available {
		setBCBSourceStatus(&out, "mdcr", "available", mr.v.Message)
	} else {
		setBCBSourceStatus(&out, "mdcr", "empty", mr.v.Message)
	}
	if sr.e != nil {
		out.Warnings = append(out.Warnings, "Séries BCB: "+sr.e.Error())
	}
	if sr.v.Available {
		setBCBSourceStatus(&out, "sgs", "available", sr.v.Message)
	} else {
		setBCBSourceStatus(&out, "sgs", "unavailable", sr.v.Message)
	}

	targetNames := marketInstitutionNames(out.Market)
	type institutionResult struct {
		v BCBInstitutionContext
		e error
	}
	type ifdataResult struct {
		v BCBIFDataContext
		e error
	}
	instCh := make(chan institutionResult, 1)
	ifdCh := make(chan ifdataResult, 1)
	go func() {
		v, e := queryBCBSupervisedInstitutions(ctx, targetNames, car.UF)
		instCh <- institutionResult{v: v, e: e}
	}()
	go func() {
		v, e := queryBCBIFData(ctx, targetNames, car.UF)
		ifdCh <- ifdataResult{v: v, e: e}
	}()
	ir, fr := <-instCh, <-ifdCh
	out.Institutions, out.IFData = ir.v, fr.v
	if ir.e != nil {
		out.Warnings = append(out.Warnings, "Entidades supervisionadas BCB: "+ir.e.Error())
	}
	if ir.v.Available {
		setBCBSourceStatus(&out, "entities", "available", ir.v.Message)
	} else {
		setBCBSourceStatus(&out, "entities", "unavailable", ir.v.Message)
	}
	if fr.e != nil {
		out.Warnings = append(out.Warnings, "IFData: "+fr.e.Error())
	}
	if fr.v.Available {
		setBCBSourceStatus(&out, "ifdata", "available", fr.v.Message)
	} else {
		setBCBSourceStatus(&out, "ifdata", "unavailable", fr.v.Message)
	}

	out.Warnings = v2UniqueNonEmpty(out.Warnings)
	out.Available = out.Market.Available || out.Series.Available || out.Institutions.Available || out.IFData.Available
	if cachePath != "" {
		if err := saveBCBPublicCache(cachePath, out); err != nil {
			out.Warnings = append(out.Warnings, "Cache BCB: "+err.Error())
		}
	}
	return out, nil
}

func setBCBSourceStatus(out *BCBPublicContext, key, status, detail string) {
	if out == nil {
		return
	}
	for i := range out.Sources {
		if out.Sources[i].Key == key {
			out.Sources[i].Status = status
			if strings.TrimSpace(detail) != "" {
				out.Sources[i].Detail = strings.TrimSpace(detail)
			}
			return
		}
	}
}

func queryBCBRuralSeries(ctx context.Context) (BCBSeriesContext, error) {
	out := BCBSeriesContext{
		SourceURL: "https://www.bcb.gov.br/estatisticas/",
		Metrics: make([]BCBSeriesMetric, len(bcbRuralSeriesSpecs)),
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, spec := range bcbRuralSeriesSpecs {
		i, spec := i, spec
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				out.Metrics[i] = seriesMetricUnavailable(spec, ctx.Err())
				return
			}
			out.Metrics[i] = queryBCBSeriesMetric(ctx, spec)
		}()
	}
	wg.Wait()

	okCount := 0
	for _, m := range out.Metrics {
		if m.Status == "available" {
			okCount++
		}
		if m.Status == "unavailable" && m.Warning != "" {
			out.Warnings = append(out.Warnings, m.Label+": "+m.Warning)
		}
	}
	out.Available = okCount > 0
	if out.Available {
		out.Message = fmt.Sprintf("%d de %d séries públicas rurais do BCB foram atualizadas nesta consulta.", okCount, len(out.Metrics))
		if okCount < len(out.Metrics) {
			out.Message += " As séries indisponíveis permaneceram identificadas separadamente."
		}
	} else {
		out.Message = "As séries públicas rurais do BCB não responderam nesta consulta."
	}
	if !out.Available && len(out.Warnings) > 0 {
		return out, fmt.Errorf("%s", strings.Join(out.Warnings, " | "))
	}
	return out, nil
}

func queryBCBSeriesMetric(ctx context.Context, spec bcbSeriesSpec) BCBSeriesMetric {
	m := BCBSeriesMetric{
		Key: spec.Key, Label: spec.Label, PersonType: spec.PersonType,
		Measure: spec.Measure, Unit: spec.Unit, SGSCode: spec.Code,
		SourceURL: fmt.Sprintf("https://api.bcb.gov.br/dados/serie/bcdata.sgs.%d/dados", spec.Code),
	}
	end := time.Now()
	start := end.AddDate(-2, 0, 0)
	q := url.Values{}
	q.Set("formato", "json")
	q.Set("dataInicial", start.Format("02/01/2006"))
	q.Set("dataFinal", end.Format("02/01/2006"))
	target := fmt.Sprintf("%s%d/dados?%s", bcbSGSBaseURL, spec.Code, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return seriesMetricUnavailable(spec, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return seriesMetricUnavailable(spec, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return seriesMetricUnavailable(spec, fmt.Errorf("HTTP %d", resp.StatusCode))
	}
	var raw []struct {
		Date  string `json:"data"`
		Value string `json:"valor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return seriesMetricUnavailable(spec, err)
	}
	for _, x := range raw {
		v, err := parseBCBDecimal(x.Value)
		if err != nil {
			continue
		}
		m.Points = append(m.Points, BCBSeriesPoint{Date: x.Date, Value: v})
	}
	if len(m.Points) == 0 {
		m.Status = "empty"
		m.Warning = "a série respondeu sem valores utilizáveis no período consultado"
		return m
	}
	m.Status = "available"
	last := m.Points[len(m.Points)-1]
	m.LatestDate, m.LatestValue = last.Date, last.Value
	if len(m.Points) > 1 {
		m.PreviousValue = m.Points[len(m.Points)-2].Value
		if m.PreviousValue != 0 {
			m.ChangePct = (m.LatestValue - m.PreviousValue) / m.PreviousValue * 100
		}
	}
	return m
}

func seriesMetricUnavailable(spec bcbSeriesSpec, err error) BCBSeriesMetric {
	m := BCBSeriesMetric{
		Key: spec.Key, Label: spec.Label, PersonType: spec.PersonType,
		Measure: spec.Measure, Unit: spec.Unit, SGSCode: spec.Code,
		Status: "unavailable",
		SourceURL: fmt.Sprintf("https://api.bcb.gov.br/dados/serie/bcdata.sgs.%d/dados", spec.Code),
	}
	if err != nil {
		m.Warning = err.Error()
	}
	return m
}

func parseBCBDecimal(v string) (float64, error) {
	v = strings.TrimSpace(v)
	if strings.Contains(v, ",") {
		v = strings.ReplaceAll(v, ".", "")
		v = strings.ReplaceAll(v, ",", ".")
	}
	return strconv.ParseFloat(v, 64)
}

func marketInstitutionNames(m CreditMarketContext) []string {
	out := make([]string, 0, len(m.StateInstitutions))
	seen := map[string]bool{}
	for _, item := range m.StateInstitutions {
		name := strings.TrimSpace(item.Label)
		if name == "" {
			continue
		}
		n := v2NormalizedSearchText(name)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, name)
	}
	return out
}

func queryBCBSupervisedInstitutions(ctx context.Context, targetNames []string, uf string) (BCBInstitutionContext, error) {
	out := BCBInstitutionContext{SourceURL: bcbEntitiesDatasetURL}
	dates := []time.Time{time.Now(), time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, -7)}
	var rows []map[string]any
	var lastErr error
	for _, d := range dates {
		q := url.Values{}
		q.Set("@dataBase", "'"+d.Format("01-02-2006")+"'")
		q.Set("$format", "json")
		q.Set("$top", "10000")
		target := bcbBcBaseODataBase + "EntidadesSupervisionadas(dataBase=@dataBase)?" + q.Encode()
		got, err := getODataRows(ctx, target)
		if err != nil {
			lastErr = err
			continue
		}
		rows = got
		out.Reference = d.Format("2006-01-02")
		break
	}
	if rows == nil {
		out.Message = "Cadastro de entidades supervisionadas indisponível nesta tentativa."
		if lastErr != nil {
			return out, lastErr
		}
		return out, nil
	}

	matches := make([]BCBInstitutionRecord, 0, 20)
	for _, r := range rows {
		rec := BCBInstitutionRecord{
			Code: firstNonEmptyStringMapValue(r, "CodigoSisbacen", "CodInst", "Codigo", "CodigoEntidade", "CodEntidade"),
			Name: firstNonEmptyStringMapValue(r, "Nome", "NomeEntidade", "NomeInstituicao", "RazaoSocial", "NomeReduzido", "NomeFantasia"),
			CNPJ: firstNonEmptyStringMapValue(r, "Cnpj", "CNPJ", "CnpjInstituicao", "CNPJInstituicao"),
			Type: firstNonEmptyStringMapValue(r, "TipoEntidadeSupervisionada", "NomeTipoEntidadeSupervisionada", "TipoEntidade", "Tipo"),
			Segment: firstNonEmptyStringMapValue(r, "Segmento", "NomeSegmento", "SegmentoTb"),
			Situation: firstNonEmptyStringMapValue(r, "Situacao", "Situação", "NomeSituacao", "SituacaoPessoaJuridica"),
			UF: strings.ToUpper(firstNonEmptyStringMapValue(r, "UF", "Uf", "uf")),
			Municipality: firstNonEmptyStringMapValue(r, "Municipio", "Município", "NomeMunicipio"),
		}
		if strings.TrimSpace(rec.Name) == "" {
			continue
		}
		if institutionMatchesTargets(rec.Name, targetNames) {
			matches = append(matches, rec)
			continue
		}
		if len(targetNames) == 0 && strings.EqualFold(rec.UF, strings.TrimSpace(uf)) && len(matches) < 20 {
			matches = append(matches, rec)
		}
	}
	matches = dedupeBCBInstitutions(matches)
	if len(matches) > 25 {
		matches = matches[:25]
	}
	out.Institutions = matches
	out.Available = true
	if len(matches) > 0 {
		out.Message = fmt.Sprintf("Cadastro oficial consultado; %d instituição(ões) relevante(s) foram relacionadas ao contexto do crédito rural.", len(matches))
	} else {
		out.Message = "Cadastro oficial consultado, sem correspondência segura entre os nomes do contexto MDCR e as entidades supervisionadas."
	}
	return out, nil
}

func institutionMatchesTargets(name string, targets []string) bool {
	n := normalizedInstitutionName(name)
	if n == "" {
		return false
	}
	for _, t := range targets {
		x := normalizedInstitutionName(t)
		if x == "" {
			continue
		}
		if strings.Contains(n, x) || strings.Contains(x, n) {
			return true
		}
		nt := significantInstitutionTokens(n)
		xt := significantInstitutionTokens(x)
		common := 0
		for token := range xt {
			if nt[token] {
				common++
			}
		}
		if common >= 2 || (common == 1 && len(xt) == 1) {
			return true
		}
	}
	return false
}

func normalizedInstitutionName(v string) string {
	v = v2NormalizedSearchText(v)
	repl := strings.NewReplacer(
		"s.a.", " ", "sa", " ", "s/a", " ", "ltda", " ", "limitada", " ",
		"cooperativa de credito", " ", "cooperativa", " ", "banco", " ",
	)
	return strings.Join(strings.Fields(repl.Replace(v)), " ")
}

func significantInstitutionTokens(v string) map[string]bool {
	stop := map[string]bool{"do": true, "da": true, "de": true, "dos": true, "das": true, "e": true, "credito": true, "financeira": true, "financeiro": true}
	out := map[string]bool{}
	for _, token := range strings.Fields(v) {
		if len(token) < 3 || stop[token] {
			continue
		}
		out[token] = true
	}
	return out
}

func dedupeBCBInstitutions(in []BCBInstitutionRecord) []BCBInstitutionRecord {
	seen := map[string]bool{}
	out := make([]BCBInstitutionRecord, 0, len(in))
	for _, x := range in {
		key := strings.TrimSpace(x.Code) + "|" + v2NormalizedSearchText(x.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, x)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func queryBCBIFData(ctx context.Context, targetNames []string, uf string) (BCBIFDataContext, error) {
	out := BCBIFDataContext{SourceURL: bcbIFDataDatasetURL}
	periods := recentIFDataPeriods(time.Now(), 8)
	var rows []map[string]any
	var lastErr error
	for _, period := range periods {
		q := url.Values{}
		q.Set("@AnoMes", period)
		q.Set("$format", "json")
		q.Set("$top", "10000")
		target := bcbIFDataODataBase + "IfDataCadastro(AnoMes=@AnoMes)?" + q.Encode()
		got, err := getODataRows(ctx, target)
		if err != nil {
			lastErr = err
			continue
		}
		if len(got) == 0 {
			continue
		}
		rows = got
		out.Reference = period
		break
	}
	if rows == nil {
		out.Message = "IFData não retornou uma data-base trimestral utilizável nesta tentativa."
		if lastErr != nil {
			return out, lastErr
		}
		return out, nil
	}

	for _, r := range rows {
		rec := BCBIFDataRecord{
			Code: firstNonEmptyStringMapValue(r, "CodInst", "CodigoInstituicao"),
			Name: firstNonEmptyStringMapValue(r, "NomeInstituicao", "Nome", "Instituicao"),
			Reference: firstNonEmptyStringMapValue(r, "Data", "AnoMes"),
			Segment: firstNonEmptyStringMapValue(r, "SegmentoTb", "Tcb", "Td", "Tc", "Sr"),
			Activity: firstNonEmptyStringMapValue(r, "Atividade"),
			Situation: firstNonEmptyStringMapValue(r, "Situacao"),
			UF: strings.ToUpper(firstNonEmptyStringMapValue(r, "Uf", "UF")),
			Municipality: firstNonEmptyStringMapValue(r, "Municipio"),
			LeaderCNPJ: firstNonEmptyStringMapValue(r, "CnpjInstituicaoLider", "CNPJInstituicaoLider"),
		}
		if rec.Name == "" {
			continue
		}
		if institutionMatchesTargets(rec.Name, targetNames) || (len(targetNames) == 0 && strings.EqualFold(rec.UF, strings.TrimSpace(uf))) {
			out.Institutions = append(out.Institutions, rec)
		}
	}
	out.Institutions = dedupeIFData(out.Institutions)
	if len(out.Institutions) > 25 {
		out.Institutions = out.Institutions[:25]
	}
	out.Available = true
	if len(out.Institutions) > 0 {
		out.Message = fmt.Sprintf("IFData %s consultado; %d instituição(ões) relacionada(s) ao contexto foram identificadas.", out.Reference, len(out.Institutions))
	} else {
		out.Message = "IFData consultado, sem correspondência segura para as instituições do contexto MDCR."
	}
	return out, nil
}

func recentIFDataPeriods(now time.Time, n int) []string {
	if n <= 0 {
		return nil
	}
	// IFData é trimestral e divulgado com defasagem. Começamos no trimestre
	// encerrado há pelo menos 60 dias e recuamos até encontrar uma data-base.
	cutoff := now.AddDate(0, 0, -65)
	year, month := cutoff.Year(), int(cutoff.Month())
	qmonth := 3
	switch {
	case month >= 12:
		qmonth = 12
	case month >= 9:
		qmonth = 9
	case month >= 6:
		qmonth = 6
	default:
		qmonth = 3
	}
	out := make([]string, 0, n)
	for len(out) < n {
		out = append(out, fmt.Sprintf("%04d%02d", year, qmonth))
		qmonth -= 3
		if qmonth <= 0 {
			year--
			qmonth = 12
		}
	}
	return out
}

func dedupeIFData(in []BCBIFDataRecord) []BCBIFDataRecord {
	seen := map[string]bool{}
	out := make([]BCBIFDataRecord, 0, len(in))
	for _, x := range in {
		key := strings.TrimSpace(x.Code) + "|" + v2NormalizedSearchText(x.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, x)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func loadBCBPublicCache(path string, maxAge time.Duration) (BCBPublicContext, bool) {
	if strings.TrimSpace(path) == "" {
		return BCBPublicContext{}, false
	}
	st, err := os.Stat(path)
	if err != nil || time.Since(st.ModTime()) > maxAge {
		return BCBPublicContext{}, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return BCBPublicContext{}, false
	}
	var out BCBPublicContext
	if json.Unmarshal(b, &out) != nil || strings.TrimSpace(out.GeneratedAt) == "" {
		return BCBPublicContext{}, false
	}
	return out, true
}

func saveBCBPublicCache(path string, value BCBPublicContext) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
