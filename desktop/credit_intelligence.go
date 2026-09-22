package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	creditIntelligenceBCBURL = "https://www.bcb.gov.br/estabilidadefinanceira/tabelas-credito-rural-proagro"
	creditIntelligenceMCRURL = "https://www3.bcb.gov.br/mcr/completo"
	creditIntelligenceZARCURL = "https://dados.agricultura.gov.br/dataset/tabua-de-risco-zoneamento-agricola-de-risco-climatico"
)

type CreditRelease struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type CreditDisbursement struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type CreditBalance struct {
	Found          bool    `json:"found"`
	Year           int     `json:"year"`
	Month          int     `json:"month"`
	LastDay        float64 `json:"last_day"`
	DailyAverage   float64 `json:"daily_average"`
	DueDailyAverage float64 `json:"due_daily_average"`
	StatusCode     string  `json:"status_code"`
	Status         string  `json:"status"`
}

type CreditDeclassification struct {
	Found      bool    `json:"found"`
	Date       string  `json:"date"`
	ReasonCode string  `json:"reason_code"`
	Reason     string  `json:"reason"`
	Type       string  `json:"type"`
	Value      float64 `json:"value"`
}

type CreditRenegotiation struct {
	Found          bool    `json:"found"`
	RelatedRef     string  `json:"related_ref"`
	RelatedOrder   string  `json:"related_order"`
	Value          float64 `json:"value"`
	LegalBasisCode string  `json:"legal_basis_code"`
	LegalBasis     string  `json:"legal_basis"`
	Direction      string  `json:"direction"`
}

type CreditProagro struct {
	HasCOP          bool    `json:"has_cop"`
	COPDate         string  `json:"cop_date"`
	COPStatusCode   string  `json:"cop_status_code"`
	COPStatus       string  `json:"cop_status"`
	EventCode       string  `json:"event_code"`
	Event           string  `json:"event"`
	CycleCode       string  `json:"cycle_code"`
	Cycle           string  `json:"cycle"`
	SoilCode        string  `json:"soil_code"`
	Soil            string  `json:"soil"`
	PlantingStart   string  `json:"planting_start"`
	PlantingEnd     string  `json:"planting_end"`
	HarvestStart    string  `json:"harvest_start"`
	HarvestEnd      string  `json:"harvest_end"`
	HasRCP          bool    `json:"has_rcp"`
	RCPDate         string  `json:"rcp_date"`
	RCPStatusCode   string  `json:"rcp_status_code"`
	RCPAreaHa       float64 `json:"rcp_area_ha"`
	RCPForecastProd float64 `json:"rcp_forecast_prod"`
	RCPForecastRev  float64 `json:"rcp_forecast_revenue"`
	HasJudgment     bool    `json:"has_judgment"`
	JudgmentDate    string  `json:"judgment_date"`
	DecisionCode    string  `json:"decision_code"`
	CoverageCredit  float64 `json:"coverage_credit"`
	CoverageOwn     float64 `json:"coverage_own"`
	LossesUncovered float64 `json:"losses_uncovered"`
	PaidValue       float64 `json:"paid_value"`
}

type CreditOperationIntelligence struct {
	RefBacen               string                 `json:"ref_bacen"`
	Order                  string                 `json:"order"`
	Year                   int                    `json:"year"`
	Institution            string                 `json:"institution"`
	Program                string                 `json:"program"`
	Subprogram             string                 `json:"subprogram"`
	Resource               string                 `json:"resource"`
	Purpose                string                 `json:"purpose"`
	Activity               string                 `json:"activity"`
	Modality               string                 `json:"modality"`
	Product                string                 `json:"product"`
	Variety                string                 `json:"variety"`
	IssueDate               string                 `json:"issue_date"`
	DueDate                 string                 `json:"due_date"`
	CreditValue             float64                `json:"credit_value"`
	OwnResources            float64                `json:"own_resources"`
	FinancedAreaHa          float64                `json:"financed_area_ha"`
	InformedAreaHa          float64                `json:"informed_area_ha"`
	InterestRatePct         float64                `json:"interest_rate_pct"`
	PostFixedInterestPct    float64                `json:"post_fixed_interest_pct"`
	EffectiveCostPct        float64                `json:"effective_cost_pct"`
	InvestmentInstallment   float64                `json:"investment_installment"`
	ForecastProduction      float64                `json:"forecast_production"`
	Quantity                float64                `json:"quantity"`
	ExpectedRevenue         float64                `json:"expected_revenue"`
	ProductivityObtained    float64                `json:"productivity_obtained"`
	ProagroRatePct          float64                `json:"proagro_rate_pct"`
	InsuranceCode           string                 `json:"insurance_code"`
	Insurance               string                 `json:"insurance"`
	InstrumentCode          string                 `json:"instrument_code"`
	Instrument              string                 `json:"instrument"`
	IssuerCategoryCode      string                 `json:"issuer_category_code"`
	IssuerCategory          string                 `json:"issuer_category"`
	IrrigationCode          string                 `json:"irrigation_code"`
	Irrigation              string                 `json:"irrigation"`
	AgricultureCode         string                 `json:"agriculture_code"`
	Agriculture             string                 `json:"agriculture"`
	CropTypeCode            string                 `json:"crop_type_code"`
	CropType                string                 `json:"crop_type"`
	IntegrationCode         string                 `json:"integration_code"`
	Integration             string                 `json:"integration"`
	SeedTypeCode            string                 `json:"seed_type_code"`
	SeedType                string                 `json:"seed_type"`
	ProductionPhaseCode     string                 `json:"production_phase_code"`
	ProductionPhase         string                 `json:"production_phase"`
	CultivarCycleCode       string                 `json:"cultivar_cycle_code"`
	CultivarCycle           string                 `json:"cultivar_cycle"`
	SoilCode                string                 `json:"soil_code"`
	Soil                    string                 `json:"soil"`
	PlantingStart           string                 `json:"planting_start"`
	PlantingEnd             string                 `json:"planting_end"`
	HarvestStart            string                 `json:"harvest_start"`
	HarvestEnd              string                 `json:"harvest_end"`
	Releases                []CreditRelease        `json:"releases"`
	TotalReleased           float64                `json:"total_released"`
	LastReleaseDate         string                 `json:"last_release_date"`
	Disbursements           []CreditDisbursement   `json:"disbursements"`
	PlannedDisbursement     float64                `json:"planned_disbursement"`
	Balance                 CreditBalance          `json:"balance"`
	Declassification        CreditDeclassification `json:"declassification"`
	Renegotiations          []CreditRenegotiation  `json:"renegotiations"`
	Proagro                 CreditProagro           `json:"proagro"`
	ZARC                    ZARCCheck               `json:"zarc"`
}

type CreditIntelligenceTotals struct {
	Operations          int     `json:"operations"`
	Contracted          float64 `json:"contracted"`
	OwnResources        float64 `json:"own_resources"`
	Released            float64 `json:"released"`
	LatestBalances      float64 `json:"latest_balances"`
	PlannedDisbursement float64 `json:"planned_disbursement"`
	Declassified        float64 `json:"declassified"`
	ProagroPaid         float64 `json:"proagro_paid"`
	WithBalance         int     `json:"with_balance"`
	WithRenegotiation   int     `json:"with_renegotiation"`
	WithProagro         int     `json:"with_proagro"`
}

type CreditIntelligenceResult struct {
	CAR          string                        `json:"car"`
	Municipality string                        `json:"municipality"`
	UF           string                        `json:"uf"`
	GeneratedAt  string                        `json:"generated_at"`
	Totals       CreditIntelligenceTotals      `json:"totals"`
	Operations   []CreditOperationIntelligence `json:"operations"`
	Warnings     []string                      `json:"warnings"`
	BCBSourceURL string                        `json:"bcb_source_url"`
	MCRSourceURL string                        `json:"mcr_source_url"`
	ZARCSourceURL string                       `json:"zarc_source_url"`
	Market       CreditMarketContext          `json:"market"`
	Scope        string                        `json:"scope"`
}

type creditDomains struct {
	Situation map[string]string
	Insurance map[string]string
	Instrument map[string]string
	IssuerCategory map[string]string
	Irrigation map[string]string
	Agriculture map[string]string
	CropType map[string]string
	Integration map[string]string
	SeedType map[string]string
	ProductionPhase map[string]string
	CultivarCycle map[string]string
	Soil map[string]string
	DeclassReason map[string]string
	COPStatus map[string]string
	Event map[string]string
	LegalBasis map[string]string
}

func (a *App) GetCreditIntelligence(propertyID int64, force bool) (CreditIntelligenceResult, error) {
	car, err := a.carForRuralCredit(propertyID)
	if err != nil {
		return CreditIntelligenceResult{}, err
	}
	xray, err := a.BuildSICORPropertyXRay(propertyID, force)
	if err != nil {
		return CreditIntelligenceResult{}, err
	}
	out := CreditIntelligenceResult{
		CAR: car.CAR, Municipality: car.Municipality, UF: car.UF,
		GeneratedAt: time.Now().Format(time.RFC3339),
		BCBSourceURL: creditIntelligenceBCBURL,
		MCRSourceURL: creditIntelligenceMCRURL,
		ZARCSourceURL: creditIntelligenceZARCURL,
		Scope: "Informações públicas localizadas no SICOR/BCB para operações vinculadas ao CAR. Valores identificados não representam necessariamente toda a dívida, todo o crédito privado ou toda a exposição financeira do produtor.",
	}
	if len(xray.Operations) == 0 {
		out.Warnings = append(out.Warnings, "Nenhuma operação pública vinculada ao CAR foi localizada para enriquecer.")
		return out, nil
	}
	sourceDir := filepath.Join(a.dataDir, "cache", "sicor_credit_intelligence")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return out, err
	}
	baseCtx := context.Background()
	if a.ctx != nil {
		baseCtx = a.ctx
	}
	ctx, cancel := context.WithTimeout(baseCtx, 25*time.Minute)
	defer cancel()

	dom, domainWarnings := a.loadCreditDomains(ctx, sourceDir)
	out.Warnings = append(out.Warnings, domainWarnings...)

	targets := map[string]bool{}
	byKey := map[string]*CreditOperationIntelligence{}
	years := map[int]bool{}
	for _, old := range xray.Operations {
		key := sicorOperationKey(old.RefBacen, old.Order)
		targets[key] = true
		years[old.Year] = true
		op := &CreditOperationIntelligence{
			RefBacen: old.RefBacen, Order: old.Order, Year: old.Year,
			Institution: old.InstitutionName, Program: old.ProgramName,
			Subprogram: old.SubprogramName, Resource: old.ResourceName,
			Purpose: old.Purpose, Activity: old.Activity, Modality: old.Modality,
			Product: old.Product, Variety: old.Variety, IssueDate: old.IssueDate,
			DueDate: old.DueDate, CreditValue: old.CreditValue,
			OwnResources: old.OwnResources, FinancedAreaHa: old.FinancedAreaHa,
			InformedAreaHa: old.InformedAreaHa, InterestRatePct: old.InterestRatePct,
			InsuranceCode: old.InsuranceCode,
		}
		byKey[key] = op
	}

	for year := range years {
		name := fmt.Sprintf("SICOR_OPERACAO_BASICA_ESTADO_%d.gz", year)
		path, e := a.ensureSICORSourceFile(ctx, sicorRawBaseURL+name, filepath.Join(sourceDir, name), 36*time.Hour)
		if e != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%d: detalhes financeiros da operação indisponíveis (%v)", year, e))
			continue
		}
		if e = scanCreditOperationDetails(path, byKey, dom); e != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%d: falha ao ler detalhes financeiros (%v)", year, e))
		}
	}

	type dataset struct{ file, label string; scan func(string) error }
	datasets := []dataset{
		{"SICOR_LIBERACAO_RECURSOS.gz", "liberações", func(p string) error { return scanCreditReleases(p, byKey) }},
		{"SICOR_PARCELAS_DESEMBOLSO.gz", "cronograma de desembolso", func(p string) error { return scanCreditDisbursements(p, byKey) }},
		{"SICOR_DESCLASSIFICACAO.gz", "desclassificações", func(p string) error { return scanCreditDeclassification(p, byKey, dom) }},
		{"SICOR_LISTA_RENEGOCIACAO.gz", "renegociações", func(p string) error { return scanCreditRenegotiations(p, byKey, dom) }},
	}
	for _, ds := range datasets {
		path, e := a.ensureSICORSourceFile(ctx, sicorRawBaseURL+ds.file, filepath.Join(sourceDir, ds.file), 36*time.Hour)
		if e != nil {
			out.Warnings = append(out.Warnings, ds.label+": "+e.Error())
			continue
		}
		if e = ds.scan(path); e != nil {
			out.Warnings = append(out.Warnings, ds.label+": "+e.Error())
		}
	}

	a.enrichCreditBalances(ctx, sourceDir, byKey, dom, &out)
	if hasProagroCandidate(byKey) {
		a.enrichCreditProagro(ctx, sourceDir, byKey, dom, &out)
	}
	a.enrichCreditZARC(ctx, sourceDir, car, byKey, &out)
	if market, marketErr := queryCreditMarketContext(ctx, car.Municipality, car.UF, car.MunicipalityCode); marketErr == nil {
		out.Market = market
	} else {
		out.Market = market
		out.Warnings = append(out.Warnings, "Contexto de mercado MDCR: "+marketErr.Error())
	}

	for _, op := range byKey {
		sort.SliceStable(op.Releases, func(i, j int) bool { return op.Releases[i].Date < op.Releases[j].Date })
		sort.SliceStable(op.Disbursements, func(i, j int) bool { return op.Disbursements[i].Date < op.Disbursements[j].Date })
		out.Operations = append(out.Operations, *op)
		out.Totals.Contracted += op.CreditValue
		out.Totals.OwnResources += op.OwnResources
		out.Totals.Released += op.TotalReleased
		out.Totals.PlannedDisbursement += op.PlannedDisbursement
		out.Totals.Declassified += op.Declassification.Value
		out.Totals.ProagroPaid += op.Proagro.PaidValue
		if op.Balance.Found {
			out.Totals.WithBalance++
			out.Totals.LatestBalances += op.Balance.LastDay
		}
		if len(op.Renegotiations) > 0 || strings.Contains(strings.ToUpper(op.Balance.Status), "RENEGOCI") || strings.Contains(strings.ToUpper(op.Balance.Status), "PRORROG") {
			out.Totals.WithRenegotiation++
		}
		if op.Proagro.HasCOP || op.Proagro.HasRCP || op.Proagro.HasJudgment || op.Proagro.PaidValue > 0 {
			out.Totals.WithProagro++
		}
	}
	out.Totals.Operations = len(out.Operations)
	sort.SliceStable(out.Operations, func(i, j int) bool {
		if out.Operations[i].IssueDate == out.Operations[j].IssueDate {
			return out.Operations[i].RefBacen > out.Operations[j].RefBacen
		}
		return out.Operations[i].IssueDate > out.Operations[j].IssueDate
	})
	return out, nil
}

func scanCreditOperationDetails(path string, byKey map[string]*CreditOperationIntelligence, dom creditDomains) error {
	return readGzipCSV(path, ';', func(h map[string]int, row []string) error {
		ref, order := strings.TrimSpace(fieldCSV(h,row,"REF_BACEN")), strings.TrimSpace(fieldCSV(h,row,"NU_ORDEM"))
		op := byKey[sicorOperationKey(ref,order)]
		if op == nil { return nil }
		op.OwnResources = parseSICORNumber(fieldCSV(h,row,"VL_REC_PROPRIO"))
		op.InformedAreaHa = parseSICORNumber(fieldCSV(h,row,"VL_AREA_INFORMADA"))
		op.InterestRatePct = parseSICORNumber(fieldCSV(h,row,"VL_JUROS"))
		op.PostFixedInterestPct = parseSICORNumber(fieldCSV(h,row,"VL_JUROS_ENC_FINAN_POSFIX"))
		op.EffectiveCostPct = parseSICORNumber(fieldCSV(h,row,"VL_PERC_CUSTO_EFET_TOTAL"))
		op.InvestmentInstallment = parseSICORNumber(fieldCSV(h,row,"VL_PRESTACAO_INVESTIMENTO"))
		op.ForecastProduction = parseSICORNumber(fieldCSV(h,row,"VL_PREV_PROD"))
		op.Quantity = parseSICORNumber(fieldCSV(h,row,"VL_QUANTIDADE"))
		op.ExpectedRevenue = parseSICORNumber(fieldCSV(h,row,"VL_RECEITA_BRUTA_ESPERADA"))
		op.ProductivityObtained = parseSICORNumber(fieldCSV(h,row,"VL_PRODUTIV_OBTIDA"))
		op.ProagroRatePct = parseSICORNumber(fieldCSV(h,row,"VL_ALIQ_PROAGRO"))
		op.InsuranceCode = fieldCSV(h,row,"CD_TIPO_SEGURO")
		op.Insurance = domainValue(dom.Insurance,op.InsuranceCode)
		op.InstrumentCode = fieldCSV(h,row,"CD_INST_CREDITO"); op.Instrument = domainValue(dom.Instrument,op.InstrumentCode)
		op.IssuerCategoryCode = fieldCSV(h,row,"CD_CATEG_EMITENTE"); op.IssuerCategory = domainValue(dom.IssuerCategory,op.IssuerCategoryCode)
		op.IrrigationCode = fieldCSV(h,row,"CD_TIPO_IRRIGACAO"); op.Irrigation = domainValue(dom.Irrigation,op.IrrigationCode)
		op.AgricultureCode = fieldCSV(h,row,"CD_TIPO_AGRICULTURA"); op.Agriculture = domainValue(dom.Agriculture,op.AgricultureCode)
		op.CropTypeCode = fieldCSV(h,row,"CD_TIPO_CULTIVO"); op.CropType = domainValue(dom.CropType,op.CropTypeCode)
		op.IntegrationCode = fieldCSV(h,row,"CD_TIPO_INTGR_CONSOR"); op.Integration = domainValue(dom.Integration,op.IntegrationCode)
		op.SeedTypeCode = fieldCSV(h,row,"CD_TIPO_GRAO_SEMENTE"); op.SeedType = domainValue(dom.SeedType,op.SeedTypeCode)
		op.ProductionPhaseCode = fieldCSV(h,row,"CD_FASE_CICLO_PRODUCAO"); op.ProductionPhase = domainValue(dom.ProductionPhase,op.ProductionPhaseCode)
		op.CultivarCycleCode = fieldCSV(h,row,"CD_CICLO_CULTIVAR"); op.CultivarCycle = domainValue(dom.CultivarCycle,op.CultivarCycleCode)
		op.SoilCode = fieldCSV(h,row,"CD_TIPO_SOLO"); op.Soil = domainValue(dom.Soil,op.SoilCode)
		op.PlantingStart = fieldCSV(h,row,"DT_INIC_PLANTIO")
		op.PlantingEnd = fieldCSV(h,row,"DT_FIM_PLANTIO")
		op.HarvestStart = fieldCSV(h,row,"DT_INIC_COLHEITA")
		op.HarvestEnd = fieldCSV(h,row,"DT_FIM_COLHEITA")
		return nil
	})
}

func scanCreditReleases(path string, byKey map[string]*CreditOperationIntelligence) error {
	return readGzipCSV(path,';',func(h map[string]int,row []string) error {
		op := byKey[sicorOperationKey(fieldCSV(h,row,"REF_BACEN"),fieldCSV(h,row,"NU_ORDEM"))]
		if op==nil{return nil}
		x:=CreditRelease{Date:fieldCSV(h,row,"DT_LIBERACAO"),Value:parseSICORNumber(fieldCSV(h,row,"VL_LIBERADO"))}
		op.Releases=append(op.Releases,x);op.TotalReleased+=x.Value
		if x.Date>op.LastReleaseDate{op.LastReleaseDate=x.Date}
		return nil
	})
}

func scanCreditDisbursements(path string, byKey map[string]*CreditOperationIntelligence) error {
	return readGzipCSV(path,';',func(h map[string]int,row []string) error {
		op := byKey[sicorOperationKey(fieldCSV(h,row,"REF_BACEN"),fieldCSV(h,row,"NU_ORDEM"))]
		if op==nil{return nil}
		x:=CreditDisbursement{Date:fieldCSV(h,row,"DT_PREV_PAGAMENTO"),Value:parseSICORNumber(fieldCSV(h,row,"VALOR_PARCELA"))}
		op.Disbursements=append(op.Disbursements,x);op.PlannedDisbursement+=x.Value
		return nil
	})
}

func scanCreditDeclassification(path string, byKey map[string]*CreditOperationIntelligence, dom creditDomains) error {
	return readGzipCSV(path,';',func(h map[string]int,row []string) error {
		op:=byKey[sicorOperationKey(fieldCSV(h,row,"REF_BACEN"),fieldCSV(h,row,"NU_ORDEM"))]
		if op==nil{return nil}
		d:=CreditDeclassification{Found:true,Date:fieldCSV(h,row,"DT_DESC"),ReasonCode:fieldCSV(h,row,"CD_MOTIVO_DESC"),Type:fieldCSV(h,row,"TIPO_DESC"),Value:parseSICORNumber(fieldCSV(h,row,"VL_DESC"))}
		d.Reason=domainValue(dom.DeclassReason,d.ReasonCode)
		if !op.Declassification.Found || d.Date>=op.Declassification.Date { op.Declassification=d }
		return nil
	})
}

func scanCreditRenegotiations(path string, byKey map[string]*CreditOperationIntelligence, dom creditDomains) error {
	targetRefs:=map[string][]*CreditOperationIntelligence{}
	for _,op:=range byKey{targetRefs[strings.TrimSpace(op.RefBacen)]=append(targetRefs[strings.TrimSpace(op.RefBacen)],op)}
	return readGzipCSV(path,';',func(h map[string]int,row []string) error {
		refFields:=[]struct{name,value string}{}
		for name,idx:=range h{
			if idx<0||idx>=len(row){continue}
			up:=strings.ToUpper(name)
			if strings.Contains(up,"REF_BACEN")||strings.Contains(up,"REFBACEN")||strings.Contains(up,"NUMREFBC"){
				refFields=append(refFields,struct{name,value string}{name,strings.TrimSpace(decodeSICORText(row[idx]))})
			}
		}
		for _,rf:=range refFields{
			ops:=targetRefs[rf.value];if len(ops)==0{continue}
			var related string
			for _,other:=range refFields{if other.value!=""&&other.value!=rf.value{related=other.value;break}}
			for _,op:=range ops{
				r:=CreditRenegotiation{Found:true,RelatedRef:related,Direction:rf.name}
				r.RelatedOrder=firstExistingCSV(h,row,"NU_ORDEM_RENEGC","NUMORDEMDESTCRENEGC","NU_ORDEM")
				r.Value=parseSICORNumber(firstExistingCSV(h,row,"VLR_RENEGC","VL_RENEGC","VALOR_RENEGOCIADO"))
				r.LegalBasisCode=firstExistingCSV(h,row,"CD_BASE_LEGAL_RENEGC","CODBASELEGALRENEGC","CD_BASE_LEGAL")
				r.LegalBasis=domainValue(dom.LegalBasis,r.LegalBasisCode)
				op.Renegotiations=append(op.Renegotiations,r)
			}
		}
		return nil
	})
}

func (a *App) enrichCreditBalances(ctx context.Context, sourceDir string, byKey map[string]*CreditOperationIntelligence, dom creditDomains, out *CreditIntelligenceResult) {
	pending:=map[string]bool{}
	minYear:=time.Now().Year()
	for k,op:=range byKey{pending[k]=true;if op.Year>0&&op.Year<minYear{minYear=op.Year}}
	years:=[]int{}
	now:=time.Now().Year()
	for y:=now;y>=minYear;y--{years=append(years,y)}
	for _,year:=range years{
		if len(pending)==0{break}
		name:=fmt.Sprintf("SICOR_SALDOS_%d.gz",year)
		path,e:=a.ensureSICORSourceFile(ctx,sicorRawBaseURL+name,filepath.Join(sourceDir,name),36*time.Hour)
		if e!=nil{
			if year>=now-1 { out.Warnings=append(out.Warnings,fmt.Sprintf("saldos %d: %v",year,e)) }
			continue
		}
		_ = readGzipCSV(path,';',func(h map[string]int,row []string) error{
			key:=sicorOperationKey(fieldCSV(h,row,"REF_BACEN"),fieldCSV(h,row,"NU_ORDEM"))
			op:=byKey[key];if op==nil{return nil}
			y,_:=strconv.Atoi(strings.TrimSpace(fieldCSV(h,row,"ANO_BASE")));m,_:=strconv.Atoi(strings.TrimSpace(fieldCSV(h,row,"MES_BASE")))
			b:=CreditBalance{Found:true,Year:y,Month:m,LastDay:parseSICORNumber(fieldCSV(h,row,"VL_ULTIMO_DIA")),DailyAverage:parseSICORNumber(fieldCSV(h,row,"VL_MEDIO_DIARIO")),DueDailyAverage:parseSICORNumber(fieldCSV(h,row,"VL_MEDIO_DIARIO_VINCENDO")),StatusCode:fieldCSV(h,row,"CD_SITUACAO_OPERACAO")}
			b.Status=domainValue(dom.Situation,b.StatusCode)
			if !op.Balance.Found || y>op.Balance.Year || (y==op.Balance.Year&&m>op.Balance.Month){op.Balance=b}
			delete(pending,key)
			return nil
		})
	}
	if len(pending)>0{out.Warnings=append(out.Warnings,fmt.Sprintf("%d operação(ões) sem saldo/situação localizado nos arquivos anuais processados.",len(pending)))}
}

func hasProagroCandidate(byKey map[string]*CreditOperationIntelligence) bool {
	for _,op:=range byKey{if op.ProagroRatePct>0||strings.TrimSpace(op.InsuranceCode)!=""{return true}}
	return false
}

func (a *App) enrichCreditProagro(ctx context.Context, sourceDir string, byKey map[string]*CreditOperationIntelligence, dom creditDomains, out *CreditIntelligenceResult) {
	type ds struct{file,label string; fn func(string)error}
	list:=[]ds{
		{"SICOR_COP_BASICO.gz","Proagro/COP",func(p string)error{return scanCreditCOP(p,byKey,dom)}},
		{"SICOR_RCP_BASICO.gz","Proagro/RCP",func(p string)error{return scanCreditRCP(p,byKey)}},
		{"SICOR_PARCELAS_PROAGRO.gz","Proagro/parcelas",func(p string)error{return scanCreditProagroParcels(p,byKey)}},
		{"SICOR_SUMULA_JULGAMENTO.gz","Proagro/julgamento",func(p string)error{return scanCreditProagroJudgment(p,byKey)}},
	}
	for _,d:=range list{
		p,e:=a.ensureSICORSourceFile(ctx,sicorRawBaseURL+d.file,filepath.Join(sourceDir,d.file),36*time.Hour)
		if e!=nil{out.Warnings=append(out.Warnings,d.label+": "+e.Error());continue}
		if e=d.fn(p);e!=nil{out.Warnings=append(out.Warnings,d.label+": "+e.Error())}
	}
}

func scanCreditCOP(path string,byKey map[string]*CreditOperationIntelligence,dom creditDomains)error{
	return readGzipCSV(path,';',func(h map[string]int,row []string)error{
		op:=byKey[sicorOperationKey(fieldCSV(h,row,"REF_BACEN"),fieldCSV(h,row,"NU_ORDEM"))];if op==nil{return nil}
		p:=&op.Proagro;p.HasCOP=true;p.COPDate=fieldCSV(h,row,"DT_COMUNICACAO");p.COPStatusCode=fieldCSV(h,row,"CD_STATUS");p.COPStatus=domainValue(dom.COPStatus,p.COPStatusCode);p.EventCode=fieldCSV(h,row,"CD_EVENTO");p.Event=domainValue(dom.Event,p.EventCode);p.CycleCode=fieldCSV(h,row,"CD_CICLO_CULTIVAR");p.Cycle=domainValue(dom.CultivarCycle,p.CycleCode);p.SoilCode=fieldCSV(h,row,"CD_TIPO_SOLO");p.Soil=domainValue(dom.Soil,p.SoilCode);p.PlantingStart=fieldCSV(h,row,"DT_INICIO_PLANTIO");p.PlantingEnd=fieldCSV(h,row,"DT_FIM_PLANTIO");p.HarvestStart=fieldCSV(h,row,"DT_INICIO_COLHEITA");p.HarvestEnd=fieldCSV(h,row,"DT_FIM_COLHEITA");return nil
	})
}
func scanCreditRCP(path string,byKey map[string]*CreditOperationIntelligence)error{
	return readGzipCSV(path,';',func(h map[string]int,row []string)error{
		op:=byKey[sicorOperationKey(fieldCSV(h,row,"REF_BACEN"),fieldCSV(h,row,"NU_ORDEM"))];if op==nil{return nil}
		p:=&op.Proagro;p.HasRCP=true;p.RCPDate=fieldCSV(h,row,"DT_ENTREGA");p.RCPStatusCode=fieldCSV(h,row,"CD_STATUS");p.RCPAreaHa=parseSICORNumber(fieldCSV(h,row,"VL_AREA"));p.RCPForecastProd=parseSICORNumber(fieldCSV(h,row,"VL_PREV_PROD"));p.RCPForecastRev=parseSICORNumber(fieldCSV(h,row,"VL_REC_PREV"));return nil
	})
}
func scanCreditProagroParcels(path string,byKey map[string]*CreditOperationIntelligence)error{
	return readGzipCSV(path,';',func(h map[string]int,row []string)error{
		op:=byKey[sicorOperationKey(fieldCSV(h,row,"REF_BACEN"),fieldCSV(h,row,"NU_ORDEM"))];if op==nil{return nil}
		op.Proagro.PaidValue+=parseSICORNumber(fieldCSV(h,row,"VL_PAGO"));return nil
	})
}
func scanCreditProagroJudgment(path string,byKey map[string]*CreditOperationIntelligence)error{
	return readGzipCSV(path,';',func(h map[string]int,row []string)error{
		op:=byKey[sicorOperationKey(fieldCSV(h,row,"REF_BACEN"),fieldCSV(h,row,"NU_ORDEM"))];if op==nil{return nil}
		p:=&op.Proagro;p.HasJudgment=true;p.JudgmentDate=fieldCSV(h,row,"DT_DECISAO");p.DecisionCode=fieldCSV(h,row,"CD_DECISAO");p.CoverageCredit=parseSICORNumber(fieldCSV(h,row,"VL_COBERTURA_ANT_CREDITO_CUSTEIO"));p.CoverageOwn=parseSICORNumber(fieldCSV(h,row,"VL_COBERTURA_ANT_REC_PROPRIOS"));p.LossesUncovered=parseSICORNumber(fieldCSV(h,row,"VL_PERDAS_NAO_AMPARADAS"));return nil
	})
}

func (a *App) loadCreditDomains(ctx context.Context, dir string)(creditDomains,[]string){
	d:=creditDomains{Situation:map[string]string{},Insurance:map[string]string{},Instrument:map[string]string{},IssuerCategory:map[string]string{},Irrigation:map[string]string{},Agriculture:map[string]string{},CropType:map[string]string{},Integration:map[string]string{},SeedType:map[string]string{},ProductionPhase:map[string]string{},CultivarCycle:map[string]string{},Soil:map[string]string{},DeclassReason:map[string]string{},COPStatus:map[string]string{},Event:map[string]string{},LegalBasis:map[string]string{}}
	var warnings []string
	specs:=[]struct{name string; candidates []string; target map[string]string}{
		{"situação",[]string{"SituacaoOperacao.csv"},d.Situation},
		{"seguro",[]string{"TipoGarantiaEmpreendimento.csv"},d.Insurance},
		{"instrumento",[]string{"instrumentoCredito.csv","InstrumentoCredito.csv"},d.Instrument},
		{"categoria",[]string{"CategoriaEmitente.csv"},d.IssuerCategory},
		{"irrigação",[]string{"TipoIrrigacao.csv"},d.Irrigation},
		{"agricultura",[]string{"TipoAgropecuaria.csv","TipoAgricultura.csv"},d.Agriculture},
		{"cultivo",[]string{"TipoCultivo.csv"},d.CropType},
		{"integração",[]string{"TipoIntegracao.csv"},d.Integration},
		{"grão/semente",[]string{"GraoSemente.csv"},d.SeedType},
		{"fase produtiva",[]string{"FaseCicloProducao.csv"},d.ProductionPhase},
		{"ciclo cultivar",[]string{"CicloCultivarProagro.csv"},d.CultivarCycle},
		{"solo",[]string{"TipoSoloProagro.csv"},d.Soil},
		{"motivo desclassificação",[]string{"motivoDesclassificacao.csv","MotivoDesclassificacao.csv"},d.DeclassReason},
		{"status COP",[]string{"StatusCOPProagro.csv"},d.COPStatus},
		{"evento Proagro",[]string{"EventoProagro.csv"},d.Event},
		{"base legal renegociação",[]string{"BaseLegalRenegociacao.csv","BaseLegalRenegociação.csv"},d.LegalBasis},
	}
	for _,s:=range specs{
		ok:=false
		for _,file:=range s.candidates{
			p,e:=a.ensureSICORSourceFile(ctx,sicorDomainBase+file,filepath.Join(dir,file),30*24*time.Hour);if e!=nil{continue}
			rows,e:=readPlainCSVMaps(p);if e!=nil{continue}
			for _,r:=range rows{
				code:=firstMapValue(r,"CODIGO","CD_SITUACAO_OPERACAO","CD_TIPO_SEGURO","CD_INST_CREDITO","CD_CATEG_EMITENTE","CD_TIPO_IRRIGACAO","CD_TIPO_AGRICULTURA","CD_TIPO_CULTIVO","CD_TIPO_INTGR_CONSOR","CD_TIPO_GRAO_SEMENTE","CD_FASE_CICLO_PRODUCAO","CD_CICLO_CULTIVAR","CD_TIPO_SOLO","CD_MOTIVO_DESC","CD_STATUS","CD_EVENTO","CD_BASE_LEGAL")
				desc:=firstMapValue(r,"DESCRICAO","DESCRICAO_CICLO","DESCRICAO_TIPO_SOLO","SIGLA")
				if code!="" { s.target[normalizeDomainCode(code)]=desc }
			}
			ok=true;break
		}
		if !ok { warnings=append(warnings,"domínio "+s.name+" não pôde ser atualizado; códigos brutos continuam disponíveis.") }
	}
	if len(d.Situation)==0{
		for k,v:=range map[string]string{"1":"Em curso original","2":"Em atraso","3":"Prorrogada","4":"Renegociada sem nova operação","5":"Renegociada parcialmente com nova operação","6":"Renegociada totalmente com nova operação","7":"Liquidada","8":"Desclassificada totalmente","9":"Baixada como prejuízo","10":"Excluída","11":"Inscrita em Dívida Ativa da União","12":"Inadimplente","13":"Desclassificada parcialmente"}{d.Situation[k]=v}
	}
	return d,warnings
}

func domainValue(m map[string]string,code string)string{
	code=strings.TrimSpace(code);if code==""{return ""}
	if v:=strings.TrimSpace(m[normalizeDomainCode(code)]);v!=""{return v}
	return "Código "+code
}
func firstExistingCSV(h map[string]int,row []string,names ...string)string{
	for _,n:=range names{if v:=strings.TrimSpace(fieldCSV(h,row,n));v!=""{return v}}
	return ""
}

func (a *App) ClearCreditIntelligenceCache() error {
	if a==nil||a.dataDir==""{return nil}
	p:=filepath.Join(a.dataDir,"cache","sicor_credit_intelligence")
	if err:=os.RemoveAll(p);err!=nil{return err}
	return nil
}

func creditReleaseRatio(op CreditOperationIntelligence) float64 {
	if op.CreditValue<=0{return 0}
	v:=op.TotalReleased/op.CreditValue*100
	if v<0{return 0};if v>999{return 999};return v
}

func validateCreditIntelligenceOperation(op CreditOperationIntelligence) error {
	if strings.TrimSpace(op.RefBacen)==""||strings.TrimSpace(op.Order)==""{return errors.New("operação sem REF BACEN/ordem")}
	if op.TotalReleased<0||op.CreditValue<0{return errors.New("valores negativos inesperados")}
	_ = creditReleaseRatio(op)
	return nil
}
