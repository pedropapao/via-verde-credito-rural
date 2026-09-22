package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	mcrOfficialURL = "https://www3.bcb.gov.br/mcr/completo"
	zarcOfficialURL = "https://www.gov.br/agricultura/pt-br/assuntos/riscos-seguro/programa-nacional-de-zoneamento-agricola-de-risco-climatico"
	zarcDatasetURL  = "https://dados.agricultura.gov.br/dataset/tabua-de-risco-zoneamento-agricola-de-risco-climatico"
	zarcCKANPackageURL = "https://dados.agricultura.gov.br/api/3/action/package_show?id=6d3d141c-885e-41a4-ab7f-dc8ff323b96f"
)

type SICORBalanceSnapshot struct {
	BaseYear       int     `json:"base_year"`
	BaseMonth      int     `json:"base_month"`
	SituationCode  string  `json:"situation_code"`
	SituationName  string  `json:"situation_name"`
	AverageDaily   float64 `json:"average_daily"`
	AverageDue     float64 `json:"average_due"`
	LastDayBalance float64 `json:"last_day_balance"`
}

type SICORRelease struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type SICORDisbursement struct {
	ExpectedDate string  `json:"expected_date"`
	Value        float64 `json:"value"`
}

type SICORDisqualification struct {
	Date       string  `json:"date"`
	ReasonCode string  `json:"reason_code"`
	ReasonName string  `json:"reason_name"`
	Value      float64 `json:"value"`
	Type       string  `json:"type"`
}

type SICORProagroCOP struct {
	StatusCode        string  `json:"status_code"`
	StatusName        string  `json:"status_name"`
	EventCode         string  `json:"event_code"`
	EventName         string  `json:"event_name"`
	SoilCode          string  `json:"soil_code"`
	SoilName          string  `json:"soil_name"`
	CycleCode         string  `json:"cycle_code"`
	CycleName         string  `json:"cycle_name"`
	CommunicationDate string  `json:"communication_date"`
	PlantingStart     string  `json:"planting_start"`
	PlantingEnd       string  `json:"planting_end"`
	HarvestStart      string  `json:"harvest_start"`
	HarvestEnd        string  `json:"harvest_end"`
}

type SICORProagroRCP struct {
	StatusCode       string  `json:"status_code"`
	EventCode        string  `json:"event_code"`
	TypeCode         string  `json:"type_code"`
	DeliveryDate     string  `json:"delivery_date"`
	VisitDate        string  `json:"visit_date"`
	EventStart       string  `json:"event_start"`
	EventEnd         string  `json:"event_end"`
	AreaHa           float64 `json:"area_ha"`
	ExpectedProd     float64 `json:"expected_production"`
	ExpectedRevenue  float64 `json:"expected_revenue"`
}

type SICORProagroJudgment struct {
	StatusCode        string  `json:"status_code"`
	DecisionCode      string  `json:"decision_code"`
	DecisionDate      string  `json:"decision_date"`
	BaseDate          string  `json:"base_date"`
	CoverageCredit    float64 `json:"coverage_credit"`
	CoverageOwn       float64 `json:"coverage_own_resources"`
	UncoveredLosses   float64 `json:"uncovered_losses"`
	FinancialCharges  float64 `json:"financial_charges"`
	RevenueConsidered float64 `json:"revenue_considered"`
}

type SICORProagroPayment struct {
	StatusCode string  `json:"status_code"`
	BaseDate   string  `json:"base_date"`
	PaidDate   string  `json:"paid_date"`
	BaseValue  float64 `json:"base_value"`
	Current    float64 `json:"current_value"`
	Paid       float64 `json:"paid_value"`
}

type SICORGenericRecord struct {
	Fields map[string]string `json:"fields"`
}

type SICORMCRContext struct {
	SourceURL       string  `json:"source_url"`
	Program         string  `json:"program"`
	Subprogram      string  `json:"subprogram"`
	RegisteredRate  float64 `json:"registered_rate_pct"`
	Message         string  `json:"message"`
}

type SICORZARCContext struct {
	SourceURL      string `json:"source_url"`
	DatasetURL     string `json:"dataset_url"`
	Product        string `json:"product"`
	PlantingStart  string `json:"planting_start"`
	PlantingEnd    string `json:"planting_end"`
	Irrigation     string `json:"irrigation"`
	Cultivation    string `json:"cultivation"`
	Message        string `json:"message"`
}

type SICORZARCAutomaticCheck struct {
	Attempted         bool     `json:"attempted"`
	Available         bool     `json:"available"`
	Matched           bool     `json:"matched"`
	Safra             string   `json:"safra"`
	Culture           string   `json:"culture"`
	Municipality      string   `json:"municipality"`
	UF                string   `json:"uf"`
	SoilCode          string   `json:"soil_code"`
	SoilName          string   `json:"soil_name"`
	CycleCode         string   `json:"cycle_code"`
	CycleName         string   `json:"cycle_name"`
	PlantingStart     string   `json:"planting_start"`
	PlantingEnd       string   `json:"planting_end"`
	PlantingDecendios []int    `json:"planting_decendios"`
	RiskLevels        []int    `json:"risk_levels"`
	Portarias         []string `json:"portarias"`
	Manejos           []string `json:"manejos"`
	Message           string   `json:"message"`
	SourceURL         string   `json:"source_url"`
	DatasetURL        string   `json:"dataset_url"`
}

type SICOROperationIntelligence struct {
	LatestBalance         *SICORBalanceSnapshot   `json:"latest_balance,omitempty"`
	Releases              []SICORRelease           `json:"releases"`
	ReleasedTotal         float64                  `json:"released_total"`
	Disbursements         []SICORDisbursement      `json:"disbursements"`
	ScheduledTotal        float64                  `json:"scheduled_total"`
	Disqualifications     []SICORDisqualification `json:"disqualifications"`
	ProagroCOP            []SICORProagroCOP        `json:"proagro_cop"`
	ProagroRCP            []SICORProagroRCP        `json:"proagro_rcp"`
	ProagroJudgments      []SICORProagroJudgment   `json:"proagro_judgments"`
	ProagroPayments       []SICORProagroPayment    `json:"proagro_payments"`
	ProagroPaidTotal      float64                  `json:"proagro_paid_total"`
	Renegotiations        []SICORGenericRecord     `json:"renegotiations"`
	SourceChanges         []SICORGenericRecord     `json:"source_changes"`
	MCR                   SICORMCRContext          `json:"mcr"`
	ZARC                  SICORZARCContext         `json:"zarc"`
	ZARCCheck             SICORZARCAutomaticCheck  `json:"zarc_check"`
}

type SICORFinancialIntelligenceResult struct {
	CAR                    string                 `json:"car"`
	GeneratedAt            string                 `json:"generated_at"`
	UsedCache              bool                   `json:"used_cache"`
	LatestBalanceTotal     float64                `json:"latest_balance_total"`
	ReleasedTotal          float64                `json:"released_total"`
	ProagroPaidTotal       float64                `json:"proagro_paid_total"`
	RenegotiatedOperations int                    `json:"renegotiated_operations"`
	Operations             []SICORPublicOperation `json:"operations"`
	Warnings               []string               `json:"warnings"`
	MCRSourceURL           string                 `json:"mcr_source_url"`
	ZARCSourceURL          string                 `json:"zarc_source_url"`
	ZARCDatasetURL         string                 `json:"zarc_dataset_url"`
}

func (a *App) BuildSICORFinancialIntelligence(propertyID int64, force bool) (SICORFinancialIntelligenceResult, error) {
	car, err := a.carForRuralCredit(propertyID)
	if err != nil {
		return SICORFinancialIntelligenceResult{}, err
	}
	if strings.TrimSpace(car.CAR) == "" {
		return SICORFinancialIntelligenceResult{}, errors.New("CAR não informado")
	}
	cacheDir := filepath.Join(a.dataDir, "cache", "credit_intelligence")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return SICORFinancialIntelligenceResult{}, err
	}
	cachePath := filepath.Join(cacheDir, safeFilePart(car.CAR)+".json")
	if !force {
		if st, statErr := os.Stat(cachePath); statErr == nil && time.Since(st.ModTime()) <= 7*24*time.Hour {
			if b, readErr := os.ReadFile(cachePath); readErr == nil {
				var cached SICORFinancialIntelligenceResult
				if json.Unmarshal(b, &cached) == nil && strings.TrimSpace(cached.CAR) != "" {
					cached.UsedCache = true
					return cached, nil
				}
			}
		}
	}

	base, err := a.BuildSICORPropertyXRay(propertyID, true)
	if err != nil {
		return SICORFinancialIntelligenceResult{}, err
	}
	if len(base.Operations) == 0 {
		return SICORFinancialIntelligenceResult{
			CAR: car.CAR, GeneratedAt: time.Now().Format(time.RFC3339),
			Operations: base.Operations, Warnings: base.Warnings,
			MCRSourceURL: mcrOfficialURL, ZARCSourceURL: zarcOfficialURL, ZARCDatasetURL: zarcDatasetURL,
		}, nil
	}

	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	ctx, cancel := context.WithTimeout(ctx, 35*time.Minute)
	defer cancel()
	sourceDir := filepath.Join(a.dataDir, "cache", "sicor_source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return SICORFinancialIntelligenceResult{}, err
	}
	base.Warnings = append(base.Warnings, a.enrichSICORFinancialIntelligence(ctx, sourceDir, car.MunicipalityCode, car.Municipality, car.UF, &base)...)
	out := SICORFinancialIntelligenceResult{
		CAR: car.CAR,
		GeneratedAt: time.Now().Format(time.RFC3339),
		LatestBalanceTotal: base.LatestBalanceTotal,
		ReleasedTotal: base.ReleasedTotal,
		ProagroPaidTotal: base.ProagroPaidTotal,
		RenegotiatedOperations: base.RenegotiatedOperations,
		Operations: base.Operations,
		Warnings: base.Warnings,
		MCRSourceURL: mcrOfficialURL,
		ZARCSourceURL: zarcOfficialURL,
		ZARCDatasetURL: zarcDatasetURL,
	}
	if b, marshalErr := json.Marshal(out); marshalErr == nil {
		_ = os.WriteFile(cachePath, b, 0o644)
	}
	return out, nil
}

func (a *App) enrichSICORFinancialIntelligence(ctx context.Context, sourceDir, municipalityCode, municipality, uf string, result *SICORXRayResult) []string {
	if result == nil || len(result.Operations) == 0 {
		return nil
	}
	var warnings []string
	insuranceDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "TipoGarantiaEmpreendimento.csv", "CD_TIPO_SEGURO", "DESCRICAO")
	instrumentDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "instrumentoCredito.csv", "CD_INST_CREDITO", "DESCRICAO")
	irrigationDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "TipoIrrigacao.csv", "CD_TIPO_IRRIGACAO", "DESCRICAO")
	agricultureDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "TipoAgropecuaria.csv", "CD_TIPO_AGRICULTURA", "DESCRICAO")
	cultivationDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "TipoCultivo.csv", "CD_TIPO_CULTIVO", "DESCRICAO")
	integrationDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "TipoIntegracao.csv", "CD_TIPO_INTGR_CONSOR", "DESCRICAO")
	grainDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "GraoSemente.csv", "CD_TIPO_GRAO_SEMENTE", "DESCRICAO")
	phaseDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "FaseCicloProducao.csv", "CD_FASE_CICLO_PRODUCAO", "DESCRICAO")
	soilDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "TipoSolo.csv", "CD_TIPO_SOLO", "DESCRICAO")
	cycleDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "CicloCultivar.csv", "CD_CICLO_CULTIVAR", "DESCRICAO_CICLO")
	targets := make(map[string]bool, len(result.Operations))
	refOnly := make(map[string]bool, len(result.Operations))
	minYear := time.Now().Year()
	for i := range result.Operations {
		op := &result.Operations[i]
		targets[sicorOperationKey(op.RefBacen, op.Order)] = true
		refOnly[strings.TrimSpace(op.RefBacen)] = true
		if op.Year > 0 && op.Year < minYear {
			minYear = op.Year
		}
		op.InsuranceName = insuranceDomain[normalizeDomainCode(op.InsuranceCode)]
		op.InstrumentName = instrumentDomain[normalizeDomainCode(op.InstrumentCode)]
		op.IrrigationName = irrigationDomain[normalizeDomainCode(op.IrrigationCode)]
		op.AgricultureName = agricultureDomain[normalizeDomainCode(op.AgricultureCode)]
		op.CultivationName = cultivationDomain[normalizeDomainCode(op.CultivationCode)]
		op.IntegrationName = integrationDomain[normalizeDomainCode(op.IntegrationCode)]
		op.GrainSeedName = grainDomain[normalizeDomainCode(op.GrainSeedCode)]
		op.ProductionPhaseName = phaseDomain[normalizeDomainCode(op.ProductionPhaseCode)]
		op.SoilName = soilDomain[normalizeDomainCode(op.SoilCode)]
		op.CycleName = cycleDomain[normalizeDomainCode(op.CycleCode)]
		op.Intelligence.MCR = SICORMCRContext{
			SourceURL: mcrOfficialURL,
			Program: op.ProgramName,
			Subprogram: op.SubprogramName,
			RegisteredRate: op.InterestRatePct,
			Message: "Taxa e enquadramento exibidos são os registrados na operação pública. Regras vigentes devem ser conferidas no MCR oficial.",
		}
		op.Intelligence.ZARC = SICORZARCContext{
			SourceURL: zarcOfficialURL,
			DatasetURL: zarcDatasetURL,
			Product: op.Product,
			PlantingStart: op.PlantingStart,
			PlantingEnd: op.PlantingEnd,
			Irrigation: op.IrrigationName,
			Cultivation: op.CultivationName,
			Message: "Use os dados da operação como referência e confirme a janela vigente no ZARC oficial para município, safra, solo, grupo/ciclo e manejo.",
		}
	}
	if minYear < sicorXRayStartYear {
		minYear = sicorXRayStartYear
	}

	situationDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "SituacaoOperacao.csv", "CD_SITUACAO_OPERACAO", "DESCRICAO")
	reasonDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "motivoDesclassificacao.csv", "CD_MOTIVO_DESC", "DESCRICAO")
	statusCOP, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "StatusCOPProagro.csv", "CD_STATUS", "DESCRICAO")
	eventDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "EventoProagro.csv", "CD_EVENTO", "NOME_EVENTO")
	cycleCOPDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "CicloCultivarProagro.csv", "CD_CICLO_CULTIVAR", "DESCRICAO_CICLO")
	soilCOPDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "TipoSoloProagro.csv", "CD_TIPO_SOLO", "DESCRICAO_TIPO_SOLO")

	balances := map[string]SICORBalanceSnapshot{}
	pending := make(map[string]bool, len(targets))
	for k := range targets { pending[k] = true }
	for year := time.Now().Year(); year >= minYear && len(pending) > 0; year-- {
		names := []string{fmt.Sprintf("SICOR_SALDOS_%d.gz", year)}
		if year == 2013 || year == 2014 {
			names = []string{fmt.Sprintf("SICOR_SALDOS_%d_02.gz", year), fmt.Sprintf("SICOR_SALDOS_%d_01.gz", year)}
		}
		for _, name := range names {
			path, err := a.ensureSICORSourceFile(ctx, sicorRawBaseURL+name, filepath.Join(sourceDir, name), 36*time.Hour)
			if err != nil {
				warnings = append(warnings, "saldos "+name+": "+err.Error())
				continue
			}
			if err := scanSICORBalances(path, pending, balances, situationDomain); err != nil {
				warnings = append(warnings, "saldos "+name+": "+err.Error())
			}
		}
		for k := range pending {
			if _, ok := balances[k]; ok { delete(pending, k) }
		}
	}

	type fileScan struct {
		name string
		age time.Duration
		run func(string) error
	}
	scans := []fileScan{
		{"SICOR_LIBERACAO_RECURSOS.gz", 36*time.Hour, func(path string) error { return scanSICORReleases(path, targets, result) }},
		{"SICOR_PARCELAS_DESEMBOLSO.gz", 36*time.Hour, func(path string) error { return scanSICORDisbursements(path, targets, result) }},
		{"SICOR_DESCLASSIFICACAO.gz", 72*time.Hour, func(path string) error { return scanSICORDisqualifications(path, targets, result, reasonDomain) }},
		{"SICOR_COP_BASICO.gz", 72*time.Hour, func(path string) error { return scanSICORCOP(path, targets, result, statusCOP, eventDomain, soilCOPDomain, cycleCOPDomain) }},
		{"SICOR_RCP_BASICO.gz", 72*time.Hour, func(path string) error { return scanSICORRCP(path, targets, result) }},
		{"SICOR_SUMULA_JULGAMENTO.gz", 72*time.Hour, func(path string) error { return scanSICORJudgments(path, targets, result) }},
		{"SICOR_PARCELAS_PROAGRO.gz", 72*time.Hour, func(path string) error { return scanSICORProagroPayments(path, targets, result) }},
	}
	for _, s := range scans {
		path, err := a.ensureSICORSourceFile(ctx, sicorRawBaseURL+s.name, filepath.Join(sourceDir, s.name), s.age)
		if err != nil {
			warnings = append(warnings, strings.TrimSuffix(s.name, ".gz")+": "+err.Error())
			continue
		}
		if err := s.run(path); err != nil {
			warnings = append(warnings, strings.TrimSuffix(s.name, ".gz")+": "+err.Error())
		}
	}

	for i := range result.Operations {
		key := sicorOperationKey(result.Operations[i].RefBacen, result.Operations[i].Order)
		if b, ok := balances[key]; ok {
			x := b
			result.Operations[i].Intelligence.LatestBalance = &x
			result.LatestBalanceTotal += b.LastDayBalance
		}
		result.ReleasedTotal += result.Operations[i].Intelligence.ReleasedTotal
		result.ProagroPaidTotal += result.Operations[i].Intelligence.ProagroPaidTotal
		if len(result.Operations[i].Intelligence.Renegotiations) > 0 {
			result.RenegotiatedOperations++
		}
	}

	for _, generic := range []struct{
		name string
		dst func(int, SICORGenericRecord)
	}{
		{"SICOR_LISTA_RENEGOCIACAO.gz", func(i int, r SICORGenericRecord){ result.Operations[i].Intelligence.Renegotiations = append(result.Operations[i].Intelligence.Renegotiations, r) }},
		{"SICOR_OPERACAO_BASICA_RENEGOCIACAO.gz", func(i int, r SICORGenericRecord){ result.Operations[i].Intelligence.Renegotiations = append(result.Operations[i].Intelligence.Renegotiations, r) }},
		{"SICOR_LISTA_ALTERACAO_FONTE.gz", func(i int, r SICORGenericRecord){ result.Operations[i].Intelligence.SourceChanges = append(result.Operations[i].Intelligence.SourceChanges, r) }},
	} {
		path, err := a.ensureSICORSourceFile(ctx, sicorRawBaseURL+generic.name, filepath.Join(sourceDir, generic.name), 72*time.Hour)
		if err != nil {
			warnings = append(warnings, strings.TrimSuffix(generic.name, ".gz")+": "+err.Error())
			continue
		}
		records, err := scanSICORGenericByRef(path, refOnly)
		if err != nil {
			warnings = append(warnings, strings.TrimSuffix(generic.name, ".gz")+": "+err.Error())
			continue
		}
		for i := range result.Operations {
			ref := strings.TrimSpace(result.Operations[i].RefBacen)
			for _, rec := range records {
				if genericRecordMentionsRef(rec, ref) { generic.dst(i, rec) }
			}
		}
	}
	result.RenegotiatedOperations = 0
	for i := range result.Operations {
		if len(result.Operations[i].Intelligence.Renegotiations) > 0 { result.RenegotiatedOperations++ }
	}
	if zarcWarnings := a.enrichZARCAutomaticChecks(ctx, sourceDir, municipalityCode, municipality, uf, result); len(zarcWarnings) > 0 {
		warnings = append(warnings, zarcWarnings...)
	}
	return warnings
}

func (a *App) loadSICORSimpleDomain(ctx context.Context, sourceDir, filename, keyName, valueName string) (map[string]string, error) {
	path, err := a.ensureSICORSourceFile(ctx, sicorDomainBase+filename, filepath.Join(sourceDir, filename), 30*24*time.Hour)
	if err != nil { return nil, err }
	rows, err := readPlainCSVMaps(path)
	if err != nil { return nil, err }
	out := map[string]string{}
	for _, r := range rows {
		k := normalizeDomainCode(firstMapValue(r, keyName))
		v := firstMapValue(r, valueName, "DESCRICAO")
		if k != "" && v != "" { out[k] = v }
	}
	return out, nil
}

func opIndex(result *SICORXRayResult) map[string]int {
	out := map[string]int{}
	for i := range result.Operations {
		out[sicorOperationKey(result.Operations[i].RefBacen, result.Operations[i].Order)] = i
	}
	return out
}

func scanSICORBalances(path string, targets map[string]bool, out map[string]SICORBalanceSnapshot, domain map[string]string) error {
	return readGzipCSV(path, ';', func(headers map[string]int, row []string) error {
		key := sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"), fieldCSV(headers,row,"NU_ORDEM"))
		if !targets[key] { return nil }
		year := parseIntLoose(fieldCSV(headers,row,"ANO_BASE"))
		month := parseIntLoose(fieldCSV(headers,row,"MES_BASE"))
		old, ok := out[key]
		if ok && old.BaseYear*100+old.BaseMonth >= year*100+month { return nil }
		code := strings.TrimSpace(fieldCSV(headers,row,"CD_SITUACAO_OPERACAO"))
		out[key] = SICORBalanceSnapshot{
			BaseYear:year, BaseMonth:month, SituationCode:code,
			SituationName:domain[normalizeDomainCode(code)],
			AverageDaily:parseSICORNumber(fieldCSV(headers,row,"VL_MEDIO_DIARIO")),
			AverageDue:parseSICORNumber(fieldCSV(headers,row,"VL_MEDIO_DIARIO_VINCENDO")),
			LastDayBalance:parseSICORNumber(fieldCSV(headers,row,"VL_ULTIMO_DIA")),
		}
		return nil
	})
}

func scanSICORReleases(path string, targets map[string]bool, result *SICORXRayResult) error {
	idx := opIndex(result)
	err := readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		key:=sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"),fieldCSV(headers,row,"NU_ORDEM")); if !targets[key]{return nil}
		i,ok:=idx[key]; if !ok{return nil}
		x:=SICORRelease{Date:strings.TrimSpace(fieldCSV(headers,row,"DT_LIBERACAO")),Value:parseSICORNumber(fieldCSV(headers,row,"VL_LIBERADO"))}
		result.Operations[i].Intelligence.Releases=append(result.Operations[i].Intelligence.Releases,x)
		result.Operations[i].Intelligence.ReleasedTotal+=x.Value
		return nil
	})
	for i:=range result.Operations { sort.SliceStable(result.Operations[i].Intelligence.Releases,func(a,b int)bool{return result.Operations[i].Intelligence.Releases[a].Date<result.Operations[i].Intelligence.Releases[b].Date}) }
	return err
}

func scanSICORDisbursements(path string, targets map[string]bool, result *SICORXRayResult) error {
	idx:=opIndex(result)
	err:=readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		key:=sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"),fieldCSV(headers,row,"NU_ORDEM")); if !targets[key]{return nil}
		i,ok:=idx[key]; if !ok{return nil}
		x:=SICORDisbursement{ExpectedDate:strings.TrimSpace(fieldCSV(headers,row,"DT_PREV_PAGAMENTO")),Value:parseSICORNumber(fieldCSV(headers,row,"VALOR_PARCELA"))}
		result.Operations[i].Intelligence.Disbursements=append(result.Operations[i].Intelligence.Disbursements,x)
		result.Operations[i].Intelligence.ScheduledTotal+=x.Value
		return nil
	})
	for i:=range result.Operations { sort.SliceStable(result.Operations[i].Intelligence.Disbursements,func(a,b int)bool{return result.Operations[i].Intelligence.Disbursements[a].ExpectedDate<result.Operations[i].Intelligence.Disbursements[b].ExpectedDate}) }
	return err
}

func scanSICORDisqualifications(path string, targets map[string]bool, result *SICORXRayResult, reasons map[string]string) error {
	idx:=opIndex(result)
	return readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		key:=sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"),fieldCSV(headers,row,"NU_ORDEM")); if !targets[key]{return nil}
		i,ok:=idx[key]; if !ok{return nil}; code:=strings.TrimSpace(fieldCSV(headers,row,"CD_MOTIVO_DESC"))
		result.Operations[i].Intelligence.Disqualifications=append(result.Operations[i].Intelligence.Disqualifications,SICORDisqualification{
			Date:strings.TrimSpace(fieldCSV(headers,row,"DT_DESC")),ReasonCode:code,ReasonName:reasons[normalizeDomainCode(code)],
			Value:parseSICORNumber(fieldCSV(headers,row,"VL_DESC")),Type:strings.TrimSpace(fieldCSV(headers,row,"TIPO_DESC")),
		}); return nil
	})
}

func scanSICORCOP(path string, targets map[string]bool, result *SICORXRayResult, statuses,eventNames,soilNames,cycleNames map[string]string) error {
	idx:=opIndex(result)
	return readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		key:=sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"),fieldCSV(headers,row,"NU_ORDEM")); if !targets[key]{return nil}
		i,ok:=idx[key]; if !ok{return nil}; sc:=strings.TrimSpace(fieldCSV(headers,row,"CD_STATUS")); ec:=strings.TrimSpace(fieldCSV(headers,row,"CD_EVENTO"))
		result.Operations[i].Intelligence.ProagroCOP=append(result.Operations[i].Intelligence.ProagroCOP,SICORProagroCOP{
			StatusCode:sc,StatusName:statuses[normalizeDomainCode(sc)],EventCode:ec,EventName:eventNames[normalizeDomainCode(ec)],
			SoilCode:strings.TrimSpace(fieldCSV(headers,row,"CD_TIPO_SOLO")),SoilName:soilNames[normalizeDomainCode(fieldCSV(headers,row,"CD_TIPO_SOLO"))],
			CycleCode:strings.TrimSpace(fieldCSV(headers,row,"CD_CICLO_CULTIVAR")),CycleName:cycleNames[normalizeDomainCode(fieldCSV(headers,row,"CD_CICLO_CULTIVAR"))],
			CommunicationDate:strings.TrimSpace(fieldCSV(headers,row,"DT_COMUNICACAO")),PlantingStart:strings.TrimSpace(fieldCSV(headers,row,"DT_INICIO_PLANTIO")),
			PlantingEnd:strings.TrimSpace(fieldCSV(headers,row,"DT_FIM_PLANTIO")),HarvestStart:strings.TrimSpace(fieldCSV(headers,row,"DT_INICIO_COLHEITA")),HarvestEnd:strings.TrimSpace(fieldCSV(headers,row,"DT_FIM_COLHEITA")),
		}); return nil
	})
}

func scanSICORRCP(path string, targets map[string]bool, result *SICORXRayResult) error {
	idx:=opIndex(result)
	return readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		key:=sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"),fieldCSV(headers,row,"NU_ORDEM")); if !targets[key]{return nil}
		i,ok:=idx[key]; if !ok{return nil}
		result.Operations[i].Intelligence.ProagroRCP=append(result.Operations[i].Intelligence.ProagroRCP,SICORProagroRCP{
			StatusCode:strings.TrimSpace(fieldCSV(headers,row,"CD_STATUS")),EventCode:strings.TrimSpace(fieldCSV(headers,row,"CD_EVENTO")),TypeCode:strings.TrimSpace(fieldCSV(headers,row,"CD_TIPO")),
			DeliveryDate:strings.TrimSpace(fieldCSV(headers,row,"DT_ENTREGA")),VisitDate:strings.TrimSpace(fieldCSV(headers,row,"DT_VISITA")),EventStart:strings.TrimSpace(fieldCSV(headers,row,"DT_INICIO_EVENTO")),EventEnd:strings.TrimSpace(fieldCSV(headers,row,"DT_FIM_EVENTO")),
			AreaHa:parseSICORNumber(fieldCSV(headers,row,"VL_AREA")),ExpectedProd:parseSICORNumber(fieldCSV(headers,row,"VL_PREV_PROD")),ExpectedRevenue:parseSICORNumber(fieldCSV(headers,row,"VL_REC_PREV")),
		}); return nil
	})
}

func scanSICORJudgments(path string, targets map[string]bool, result *SICORXRayResult) error {
	idx:=opIndex(result)
	return readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		key:=sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"),fieldCSV(headers,row,"NU_ORDEM")); if !targets[key]{return nil}
		i,ok:=idx[key]; if !ok{return nil}
		result.Operations[i].Intelligence.ProagroJudgments=append(result.Operations[i].Intelligence.ProagroJudgments,SICORProagroJudgment{
			StatusCode:strings.TrimSpace(fieldCSV(headers,row,"CD_STATUS")),DecisionCode:strings.TrimSpace(fieldCSV(headers,row,"CD_DECISAO")),DecisionDate:strings.TrimSpace(fieldCSV(headers,row,"DT_DECISAO")),BaseDate:strings.TrimSpace(fieldCSV(headers,row,"DT_BASE")),
			CoverageCredit:parseSICORNumber(fieldCSV(headers,row,"VL_COBERTURA_ANT_CREDITO_CUSTEIO")),CoverageOwn:parseSICORNumber(fieldCSV(headers,row,"VL_COBERTURA_ANT_REC_PROPRIOS")),
			UncoveredLosses:parseSICORNumber(fieldCSV(headers,row,"VL_PERDAS_NAO_AMPARADAS")),FinancialCharges:parseSICORNumber(fieldCSV(headers,row,"VL_ENCARGOS_SOB_CREDITO")),RevenueConsidered:parseSICORNumber(fieldCSV(headers,row,"VL_RECEITAS_CONSIDERADAS")),
		}); return nil
	})
}

func scanSICORProagroPayments(path string, targets map[string]bool, result *SICORXRayResult) error {
	idx:=opIndex(result)
	return readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		key:=sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"),fieldCSV(headers,row,"NU_ORDEM")); if !targets[key]{return nil}
		i,ok:=idx[key]; if !ok{return nil}
		x:=SICORProagroPayment{StatusCode:strings.TrimSpace(fieldCSV(headers,row,"CD_STATUS")),BaseDate:strings.TrimSpace(fieldCSV(headers,row,"DT_BASE")),PaidDate:strings.TrimSpace(fieldCSV(headers,row,"DT_PAGAMENTO")),BaseValue:parseSICORNumber(fieldCSV(headers,row,"VL_BASE")),Current:parseSICORNumber(fieldCSV(headers,row,"VL_ATUAL")),Paid:parseSICORNumber(fieldCSV(headers,row,"VL_PAGO"))}
		result.Operations[i].Intelligence.ProagroPayments=append(result.Operations[i].Intelligence.ProagroPayments,x)
		result.Operations[i].Intelligence.ProagroPaidTotal+=x.Paid
		return nil
	})
}

func scanSICORGenericByRef(path string, refs map[string]bool) ([]SICORGenericRecord,error) {
	var out []SICORGenericRecord
	err:=readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		matched:=false
		for h,i:=range headers {
			if !strings.Contains(h,"REF_BACEN") || i<0 || i>=len(row){continue}
			if refs[strings.TrimSpace(decodeSICORText(row[i]))]{matched=true;break}
		}
		if !matched{return nil}
		fields:=map[string]string{}
		keys:=make([]string,0,len(headers)); for h:=range headers{keys=append(keys,h)}; sort.Strings(keys)
		for _,h:=range keys {
			i:=headers[h]; if i<0||i>=len(row){continue}; v:=strings.TrimSpace(decodeSICORText(row[i])); if v==""{continue}
			if strings.Contains(h,"REF_BACEN")||h=="NU_ORDEM"||strings.HasPrefix(h,"DT_")||strings.HasPrefix(h,"VL_")||strings.Contains(h,"BASE_LEGAL")||strings.Contains(h,"FONTE")||strings.Contains(h,"TIPO"){
				fields[h]=v
			}
		}
		if len(fields)>0{out=append(out,SICORGenericRecord{Fields:fields})}
		return nil
	})
	return out,err
}

func genericRecordMentionsRef(rec SICORGenericRecord, ref string) bool {
	for k,v:=range rec.Fields { if strings.Contains(k,"REF_BACEN")&&strings.TrimSpace(v)==strings.TrimSpace(ref){return true} }
	return false
}


type zarcResource struct {
	Name   string
	URL    string
	Format string
}

func (a *App) fetchZARCResources(ctx context.Context) ([]zarcResource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zarcCKANPackageURL, nil)
	if err != nil { return nil, err }
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("MAPA/ZARC respondeu HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Success bool \`json:"success"\`
		Result struct {
			Resources []struct {
				Name string \`json:"name"\`
				URL string \`json:"url"\`
				Format string \`json:"format"\`
			} \`json:"resources"\`
		} \`json:"result"\`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil { return nil, err }
	if !payload.Success { return nil, errors.New("catálogo ZARC não retornou sucesso") }
	var out []zarcResource
	for _, r := range payload.Result.Resources {
		if strings.TrimSpace(r.URL)=="" { continue }
		if !strings.EqualFold(strings.TrimSpace(r.Format),"CSV") && !strings.Contains(strings.ToLower(r.URL),".csv") { continue }
		out=append(out,zarcResource{Name:strings.TrimSpace(r.Name),URL:strings.TrimSpace(r.URL),Format:r.Format})
	}
	if len(out)==0 { return nil, errors.New("catálogo ZARC sem recursos CSV") }
	return out,nil
}

func (a *App) enrichZARCAutomaticChecks(ctx context.Context, sourceDir, municipalityCode, municipality, uf string, result *SICORXRayResult) []string {
	if result==nil || len(result.Operations)==0 { return nil }
	needs:=false
	for i:=range result.Operations {
		op:=&result.Operations[i]
		if !strings.Contains(normalizeZARCText(op.Purpose),"custeio") || !strings.Contains(normalizeZARCText(op.Activity),"agric") { continue }
		if strings.TrimSpace(op.Product)=="" { continue }
		if len(op.Intelligence.ProagroCOP)>0 || strings.TrimSpace(op.PlantingStart)!="" || strings.TrimSpace(op.PlantingEnd)!="" {
			needs=true
			break
		}
	}
	if !needs { return nil }
	resources,err:=a.fetchZARCResources(ctx)
	if err!=nil {
		for i:=range result.Operations {
			result.Operations[i].Intelligence.ZARCCheck.Message="Conferência automática ZARC indisponível nesta tentativa: "+err.Error()
			result.Operations[i].Intelligence.ZARCCheck.SourceURL=zarcOfficialURL
			result.Operations[i].Intelligence.ZARCCheck.DatasetURL=zarcDatasetURL
		}
		return []string{"ZARC automático: "+err.Error()}
	}
	zarcDir:=filepath.Join(sourceDir,"zarc")
	_ = os.MkdirAll(zarcDir,0o755)
	cache:=map[string]string{}
	var warnings []string
	for i:=range result.Operations {
		op:=&result.Operations[i]
		check:=SICORZARCAutomaticCheck{SourceURL:zarcOfficialURL,DatasetURL:zarcDatasetURL,Municipality:municipality,UF:uf,Culture:op.Product}
		if !strings.Contains(normalizeZARCText(op.Purpose),"custeio") || !strings.Contains(normalizeZARCText(op.Activity),"agric") {
			check.Message="Conferência ZARC automática não se aplica a esta destinação porque ela não foi identificada como custeio agrícola."
			op.Intelligence.ZARCCheck=check
			continue
		}
		soilCode,soilName,cycleCode,cycleName:=op.SoilCode,op.SoilName,op.CycleCode,op.CycleName
		var start,end string
		if len(op.Intelligence.ProagroCOP)>0 {
			cop:=op.Intelligence.ProagroCOP[0]
			soilCode,soilName=cop.SoilCode,cop.SoilName
			cycleCode,cycleName=cop.CycleCode,cop.CycleName
			start,end=cop.PlantingStart,cop.PlantingEnd
		}
		if start=="" { start=op.PlantingStart }
		if end=="" { end=op.PlantingEnd }
		check.SoilCode,check.SoilName=soilCode,soilName
		check.CycleCode,check.CycleName=cycleCode,cycleName
		check.PlantingStart,check.PlantingEnd=start,end
		check.Attempted=true
		if strings.TrimSpace(municipalityCode)=="" || strings.TrimSpace(op.Product)=="" {
			check.Message="ZARC automático sem dados suficientes: falta geocódigo do município ou cultura/produto."
			op.Intelligence.ZARCCheck=check
			continue
		}
		seasonCandidates:=zarcSeasonCandidates(start,end,op.IssueDate)
		check.PlantingDecendios=plantingDecendios(start,end)
		if len(seasonCandidates)==0 {
			check.Message="ZARC automático sem data suficiente para determinar a safra. A fonte oficial continua disponível."
			op.Intelligence.ZARCCheck=check
			continue
		}
		var best SICORZARCAutomaticCheck
		best=check
		var attemptErrs []string
		for _,season:=range seasonCandidates {
			resource,ok:=findZARCResource(resources,season)
			if !ok { continue }
			path:=cache[resource.URL]
			if path=="" {
				name:="zarc_"+strings.ReplaceAll(season,"/","_")+".csv"
				path=filepath.Join(zarcDir,name)
				if st,statErr:=os.Stat(path); statErr!=nil || st.Size()==0 || time.Since(st.ModTime())>7*24*time.Hour {
					tmp:=path+".part"; _=os.Remove(tmp)
					if dlErr:=downloadLargeFile(ctx,resource.URL,tmp); dlErr!=nil {
						_ = os.Remove(tmp); attemptErrs=append(attemptErrs,season+": "+dlErr.Error()); continue
					}
					_ = os.Remove(path)
					if rnErr:=os.Rename(tmp,path); rnErr!=nil { _=os.Remove(tmp); attemptErrs=append(attemptErrs,season+": "+rnErr.Error()); continue }
				}
				cache[resource.URL]=path
			}
			got,scanErr:=scanZARCForOperation(path,season,municipalityCode,municipality,uf,op.Product,soilCode,soilName,cycleCode,cycleName,start,end)
			if scanErr!=nil { attemptErrs=append(attemptErrs,season+": "+scanErr.Error()); continue }
			got.SourceURL=zarcOfficialURL; got.DatasetURL=resource.URL
			if got.Available {
				best=got
				if got.Matched { break }
			}
		}
		if !best.Available && len(attemptErrs)>0 {
			best.Message="A base ZARC foi localizada, mas a conferência automática não pôde ser concluída nesta tentativa."
			warnings=append(warnings,"ZARC "+op.RefBacen+"/"+op.Order+": "+strings.Join(attemptErrs," | "))
		}
		op.Intelligence.ZARCCheck=best
	}
	return warnings
}

func zarcSeasonCandidates(start,end,issue string) []string {
	raw:=start
	if raw=="" { raw=end }
	if raw=="" { raw=issue }
	t,err:=time.Parse("2006-01-02",strings.TrimSpace(raw))
	if err!=nil { return nil }
	var first,second string
	if int(t.Month())>=7 {
		first=fmt.Sprintf("%d/%d",t.Year(),t.Year()+1)
		second=fmt.Sprintf("%d/%d",t.Year()-1,t.Year())
	} else {
		first=fmt.Sprintf("%d/%d",t.Year()-1,t.Year())
		second=fmt.Sprintf("%d/%d",t.Year(),t.Year()+1)
	}
	return []string{first,second}
}

func findZARCResource(resources []zarcResource, season string) (zarcResource,bool) {
	want:=strings.ReplaceAll(strings.ToLower(season)," ","")
	for _,r:=range resources {
		n:=strings.ReplaceAll(strings.ToLower(r.Name)," ","")
		if strings.Contains(n,want) { return r,true }
	}
	return zarcResource{},false
}

func scanZARCForOperation(path,season,municipalityCode,municipality,uf,culture,soilCode,soilName,cycleCode,cycleName,start,end string) (SICORZARCAutomaticCheck,error) {
	out:=SICORZARCAutomaticCheck{
		Attempted:true,Safra:season,Culture:culture,Municipality:municipality,UF:uf,
		SoilCode:soilCode,SoilName:soilName,CycleCode:cycleCode,CycleName:cycleName,
		PlantingStart:start,PlantingEnd:end,PlantingDecendios:plantingDecendios(start,end),
		SourceURL:zarcOfficialURL,DatasetURL:zarcDatasetURL,
	}
	f,err:=os.Open(path); if err!=nil{return out,err}; defer f.Close()
	br:=bufio.NewReader(f)
	first,err:=br.ReadString('\n'); if err!=nil && err!=io.EOF{return out,err}
	delim:=','
	if strings.Count(first,";")>strings.Count(first,","){delim=';'}
	reader:=csv.NewReader(io.MultiReader(strings.NewReader(first),br))
	reader.Comma=delim; reader.LazyQuotes=true; reader.FieldsPerRecord=-1
	header,err:=reader.Read(); if err!=nil{return out,err}
	headers:=map[string]int{}
	for i,h:=range header{headers[normalizeCSVHeader(h)]=i}
	wantCode:=normalizeDigits(municipalityCode)
	wantUF:=strings.ToUpper(strings.TrimSpace(uf))
	wantCulture:=normalizeZARCText(culture)
	riskSet:=map[int]bool{}; portSet:=map[string]bool{}; manejoSet:=map[string]bool{}
	baseRows:=0; exactRows:=0
	for{
		row,e:=reader.Read(); if e==io.EOF{break}; if e!=nil{continue}
		code:=normalizeDigits(zarcField(headers,row,"GEOCODIGO","COD_IBGE","COD_MUNIC"))
		rowUF:=strings.ToUpper(strings.TrimSpace(zarcField(headers,row,"UF")))
		if wantCode!=""&&code!=wantCode{continue}
		if wantUF!=""&&rowUF!=""&&rowUF!=wantUF{continue}
		rowCulture:=normalizeZARCText(zarcField(headers,row,"NOME_CULTURA","CULTURA"))
		if !zarcCultureMatch(wantCulture,rowCulture){continue}
		baseRows++
		rowSoil:=strings.TrimSpace(zarcField(headers,row,"COD_SOLO"))
		if strings.TrimSpace(soilCode)!="" && normalizeDomainCode(rowSoil)!=normalizeDomainCode(soilCode){continue}
		rowCycle:=strings.TrimSpace(zarcField(headers,row,"COD_CICLO","GRUPO"))
		if strings.TrimSpace(cycleCode)!="" || strings.TrimSpace(cycleName)!="" {
			if !zarcCycleMatch(rowCycle,cycleCode,cycleName){continue}
		}
		exactRows++
		if p:=strings.TrimSpace(zarcField(headers,row,"PORTARIA"));p!=""{portSet[p]=true}
		if m:=strings.TrimSpace(zarcField(headers,row,"NOME_OUTROS_MANEJOS","OUTROS_MANEJOS"));m!=""{manejoSet[m]=true}
		for _,d:=range out.PlantingDecendios {
			v:=strings.TrimSpace(zarcField(headers,row,fmt.Sprintf("DEC%d",d),fmt.Sprintf("DECENDIO_%d",d)))
			if v==""{continue}
			n,_:=strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(v,"%")))
			if n>0{riskSet[n]=true}
		}
	}
	out.Available=baseRows>0
	for n:=range riskSet{out.RiskLevels=append(out.RiskLevels,n)}
	sort.Ints(out.RiskLevels)
	for p:=range portSet{out.Portarias=append(out.Portarias,p)}; sort.Strings(out.Portarias)
	for m:=range manejoSet{out.Manejos=append(out.Manejos,m)}; sort.Strings(out.Manejos)
	if baseRows==0 {
		out.Message="Nenhum registro ZARC foi localizado para município e cultura na safra selecionada."
		return out,nil
	}
	if exactRows==0 {
		out.Message="ZARC localizado para município e cultura, mas não houve correspondência segura de solo/ciclo com os dados públicos da operação."
		return out,nil
	}
	if len(out.PlantingDecendios)==0 {
		out.Message="ZARC localizado para município, cultura, solo/ciclo, mas faltam datas de plantio para conferir os decêndios."
		return out,nil
	}
	out.Matched=len(out.RiskLevels)>0
	if out.Matched {
		out.Message="Há indicação ZARC publicada para os dados e decêndios localizados. Confira a portaria e o manejo antes de usar como conclusão de enquadramento."
	} else {
		out.Message="Não foi localizada indicação ZARC nos decêndios informados para a combinação encontrada. Confira a portaria oficial antes de concluir impedimento."
	}
	return out,nil
}

func zarcField(headers map[string]int,row []string,names ...string) string{
	for _,name:=range names{
		i,ok:=headers[normalizeCSVHeader(name)]
		if ok&&i>=0&&i<len(row){if v:=strings.TrimSpace(row[i]);v!=""{return v}}
	}
	return ""
}

func plantingDecendios(start,end string) []int {
	s,e:=parseZARCDate(start),parseZARCDate(end)
	if s.IsZero()&&e.IsZero(){return nil}
	if s.IsZero(){s=e}; if e.IsZero(){e=s}
	if e.Before(s){s,e=e,s}
	if e.Sub(s)>45*24*time.Hour{e=s}
	set:=map[int]bool{}
	for d:=s; !d.After(e); d=d.AddDate(0,0,1){
		part:=1; if d.Day()>20{part=3}else if d.Day()>10{part=2}
		set[(int(d.Month())-1)*3+part]=true
	}
	var out []int; for d:=range set{out=append(out,d)}; sort.Ints(out); return out
}

func parseZARCDate(v string) time.Time {
	v=strings.TrimSpace(v)
	for _,layout:=range []string{"2006-01-02","02/01/2006","2006/01/02"}{
		if t,err:=time.Parse(layout,v);err==nil{return t}
	}
	return time.Time{}
}

func normalizeDigits(v string) string {
	var b strings.Builder
	for _,r:=range strings.TrimSpace(v){if r>='0'&&r<='9'{b.WriteRune(r)}}
	s:=strings.TrimLeft(b.String(),"0"); if s==""&&b.Len()>0{return "0"}; return s
}

func normalizeZARCText(v string) string {
	r:=strings.NewReplacer("á","a","à","a","ã","a","â","a","ä","a","é","e","ê","e","è","e","ë","e","í","i","ì","i","î","i","ï","i","ó","o","ô","o","õ","o","ò","o","ö","o","ú","u","ù","u","û","u","ü","u","ç","c",
		"Á","a","À","a","Ã","a","Â","a","Ä","a","É","e","Ê","e","È","e","Ë","e","Í","i","Ì","i","Î","i","Ï","i","Ó","o","Ô","o","Õ","o","Ò","o","Ö","o","Ú","u","Ù","u","Û","u","Ü","u","Ç","c")
	v=strings.ToLower(r.Replace(v))
	var b strings.Builder
	lastSpace:=false
	for _,ch:=range v{
		if (ch>='a'&&ch<='z')||(ch>='0'&&ch<='9'){b.WriteRune(ch);lastSpace=false}else if !lastSpace{b.WriteByte(' ');lastSpace=true}
	}
	return strings.TrimSpace(b.String())
}

func zarcCultureMatch(want,got string) bool {
	if want==""||got==""{return false}
	if want==got{return true}
	return (len(want)>=4&&strings.Contains(got,want))||(len(got)>=4&&strings.Contains(want,got))
}

func zarcCycleMatch(rowCycle,code,name string) bool {
	r:=normalizeZARCText(rowCycle); c:=normalizeZARCText(code); n:=normalizeZARCText(name)
	if c!=""&&(r==c||strings.Contains(r,c)){return true}
	if n!=""&&(r==n||strings.Contains(r,n)||strings.Contains(n,r)){return true}
	roman:=map[string]string{"i":"1","ii":"2","iii":"3","iv":"4","v":"5","vi":"6"}
	for k,v:=range roman{
		if strings.Contains(n,"grupo "+k)&&normalizeDomainCode(r)==v{return true}
	}
	return c==""&&n==""
}


func parseIntLoose(v string) int { n,_:=strconv.Atoi(strings.TrimSpace(v)); return n }
