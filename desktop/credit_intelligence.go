package main

import (
	"context"
	"fmt"
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
	CycleCode         string  `json:"cycle_code"`
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
}

func (a *App) enrichSICORFinancialIntelligence(ctx context.Context, sourceDir string, result *SICORXRayResult) []string {
	if result == nil || len(result.Operations) == 0 {
		return nil
	}
	var warnings []string
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
	eventDomain, _ := a.loadSICORSimpleDomain(ctx, sourceDir, "EventoProagro.csv", "CD_EVENTO", "DESCRICAO")

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
		{"SICOR_COP_BASICO.gz", 72*time.Hour, func(path string) error { return scanSICORCOP(path, targets, result, statusCOP, eventDomain) }},
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

func scanSICORCOP(path string, targets map[string]bool, result *SICORXRayResult, statuses,eventNames map[string]string) error {
	idx:=opIndex(result)
	return readGzipCSV(path,';',func(headers map[string]int,row []string) error{
		key:=sicorOperationKey(fieldCSV(headers,row,"REF_BACEN"),fieldCSV(headers,row,"NU_ORDEM")); if !targets[key]{return nil}
		i,ok:=idx[key]; if !ok{return nil}; sc:=strings.TrimSpace(fieldCSV(headers,row,"CD_STATUS")); ec:=strings.TrimSpace(fieldCSV(headers,row,"CD_EVENTO"))
		result.Operations[i].Intelligence.ProagroCOP=append(result.Operations[i].Intelligence.ProagroCOP,SICORProagroCOP{
			StatusCode:sc,StatusName:statuses[normalizeDomainCode(sc)],EventCode:ec,EventName:eventNames[normalizeDomainCode(ec)],
			SoilCode:strings.TrimSpace(fieldCSV(headers,row,"CD_TIPO_SOLO")),CycleCode:strings.TrimSpace(fieldCSV(headers,row,"CD_CICLO_CULTIVAR")),
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

func parseIntLoose(v string) int { n,_:=strconv.Atoi(strings.TrimSpace(v)); return n }
