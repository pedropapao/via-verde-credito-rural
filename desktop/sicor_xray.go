package main

import (
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	sicorRawBaseURL = "https://www.bcb.gov.br/htms/sicor/DadosBrutos/"
	sicorDomainBase = "https://www.bcb.gov.br/htms/sicor/"
	sicorXRayStartYear = 2013
)

type SICORPublicOperation struct {
	RefBacen          string       `json:"ref_bacen"`
	Order             string       `json:"order"`
	IssueDate         string       `json:"issue_date"`
	DueDate           string       `json:"due_date"`
	Year              int          `json:"year"`
	UF                string       `json:"uf"`
	InstitutionCode   string       `json:"institution_code"`
	InstitutionName   string       `json:"institution_name"`
	InstitutionType   string       `json:"institution_type"`
	ProgramCode       string       `json:"program_code"`
	ProgramName       string       `json:"program_name"`
	SubprogramCode    string       `json:"subprogram_code"`
	SubprogramName    string       `json:"subprogram_name"`
	ResourceCode      string       `json:"resource_code"`
	ResourceName      string       `json:"resource_name"`
	EnterpriseCode    string       `json:"enterprise_code"`
	Purpose           string       `json:"purpose"`
	Activity          string       `json:"activity"`
	Modality          string       `json:"modality"`
	Product           string       `json:"product"`
	Variety           string       `json:"variety"`
	CreditValue       float64      `json:"credit_value"`
	OwnResources      float64      `json:"own_resources"`
	FinancedAreaHa    float64      `json:"financed_area_ha"`
	InformedAreaHa    float64      `json:"informed_area_ha"`
	InterestRatePct       float64      `json:"interest_rate_pct"`
	PostFixedInterestPct  float64      `json:"post_fixed_interest_pct"`
	EffectiveCostPct      float64      `json:"effective_cost_pct"`
	InvestmentInstallment float64      `json:"investment_installment"`
	ExpectedProduction    float64      `json:"expected_production"`
	Quantity              float64      `json:"quantity"`
	ExpectedGrossRevenue  float64      `json:"expected_gross_revenue"`
	ObtainedProductivity  float64      `json:"obtained_productivity"`
	ProagroRatePct        float64      `json:"proagro_rate_pct"`
	STNRiskPct            float64      `json:"stn_risk_pct"`
	FundRiskPct           float64      `json:"fund_risk_pct"`
	ServiceOwnResources   float64      `json:"service_own_resources"`
	BonusCARPct           float64      `json:"bonus_car_pct"`
	ContractSTN           string       `json:"contract_stn"`
	RegistrantCNPJ        string       `json:"registrant_cnpj"`
	SoilCode              string       `json:"soil_code"`
	SoilName              string       `json:"soil_name"`
	CycleCode             string       `json:"cycle_code"`
	CycleName             string       `json:"cycle_name"`
	InsuranceCode         string       `json:"insurance_code"`
	InsuranceName         string       `json:"insurance_name"`
	InstrumentCode        string       `json:"instrument_code"`
	InstrumentName        string       `json:"instrument_name"`
	IrrigationCode        string       `json:"irrigation_code"`
	IrrigationName        string       `json:"irrigation_name"`
	AgricultureCode       string       `json:"agriculture_code"`
	AgricultureName       string       `json:"agriculture_name"`
	CultivationCode       string       `json:"cultivation_code"`
	CultivationName       string       `json:"cultivation_name"`
	IntegrationCode       string       `json:"integration_code"`
	IntegrationName       string       `json:"integration_name"`
	GrainSeedCode         string       `json:"grain_seed_code"`
	GrainSeedName         string       `json:"grain_seed_name"`
	ProductionPhaseCode   string       `json:"production_phase_code"`
	ProductionPhaseName   string       `json:"production_phase_name"`
	PlantingStart         string       `json:"planting_start"`
	PlantingEnd           string       `json:"planting_end"`
	HarvestStart          string       `json:"harvest_start"`
	HarvestEnd            string       `json:"harvest_end"`
	Intelligence          SICOROperationIntelligence `json:"intelligence"`
	Glebas                []SICORGleba `json:"glebas"`
}

type SICORGleba struct {
	RefBacen             string  `json:"ref_bacen"`
	Order                string  `json:"order"`
	Index                string  `json:"index"`
	AreaHa               float64 `json:"area_ha"`
	InsideCARPct         float64 `json:"inside_car_pct"`
	ProjectOverlapPct    float64 `json:"project_overlap_pct"`
	ProjectOverlapName   string  `json:"project_overlap_name"`
	GeoJSON              string  `json:"geojson"`
}

type SICORXRayResult struct {
	CAR                    string                 `json:"car"`
	FromYear               int                    `json:"from_year"`
	ToYear                 int                    `json:"to_year"`
	GeneratedAt            string                 `json:"generated_at"`
	CacheUntil             string                 `json:"cache_until"`
	PropertyReferences     int                    `json:"property_references"`
	UnresolvedReferences   int                    `json:"unresolved_references"`
	OperationCount         int                    `json:"operation_count"`
	DestinationCount       int                    `json:"destination_count"`
	GlebaCount             int                    `json:"gleba_count"`
	TotalCreditValue       float64                `json:"total_credit_value"`
	TotalFinancedAreaHa    float64                `json:"total_financed_area_ha"`
	TotalGlebaAreaHa       float64                `json:"total_gleba_area_ha"`
	LatestBalanceTotal      float64                `json:"latest_balance_total"`
	ReleasedTotal           float64                `json:"released_total"`
	ProagroPaidTotal        float64                `json:"proagro_paid_total"`
	RenegotiatedOperations  int                    `json:"renegotiated_operations"`
	Operations             []SICORPublicOperation `json:"operations"`
	Warnings               []string               `json:"warnings"`
	SourceURL              string                 `json:"source_url"`
	Scope                  string                 `json:"scope"`
	UsedCache              bool                   `json:"used_cache"`
}

type sicorRef struct {
	RefBacen string
	Order    string
}

type sicorEnterprise struct {
	Purpose  string
	Activity string
	Modality string
	Product  string
	Variety  string
}

type sicorDomains struct {
	Institutions map[string][2]string
	Programs     map[string]string
	Subprograms  map[string]string
	Resources    map[string]string
	Enterprises  map[string]sicorEnterprise
}

func (a *App) BuildSICORPropertyXRay(propertyID int64, force bool) (SICORXRayResult, error) {
	car, err := a.carForRuralCredit(propertyID)
	if err != nil {
		return SICORXRayResult{}, err
	}
	if strings.TrimSpace(car.CAR) == "" {
		return SICORXRayResult{}, errors.New("CAR não informado")
	}

	cacheDir := filepath.Join(a.dataDir, "cache", "sicor_xray")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return SICORXRayResult{}, err
	}
	cachePath := filepath.Join(cacheDir, safeFilePart(car.CAR)+".json")
	if !force {
		if cached, ok := loadSICORXRayCache(cachePath, 14*24*time.Hour); ok {
			cached.UsedCache = true
			return cached, nil
		}
	}

	baseCtx := context.Background()
	if a.ctx != nil {
		baseCtx = a.ctx
	}
	ctx, cancel := context.WithTimeout(baseCtx, 25*time.Minute)
	defer cancel()

	sourceDir := filepath.Join(a.dataDir, "cache", "sicor_source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return SICORXRayResult{}, err
	}
	pruneSICORSourceCache(sourceDir, 72*time.Hour)

	propertiesPath, err := a.ensureSICORSourceFile(ctx,
		sicorRawBaseURL+"SICOR_PROPRIEDADES.gz",
		filepath.Join(sourceDir, "SICOR_PROPRIEDADES.gz"),
		7*24*time.Hour,
	)
	if err != nil {
		return SICORXRayResult{}, fmt.Errorf("não foi possível sincronizar o índice de propriedades do SICOR: %w", err)
	}
	refs, err := scanSICORPropertyRefs(propertiesPath, car.CAR)
	if err != nil {
		return SICORXRayResult{}, err
	}

	result := SICORXRayResult{
		CAR: car.CAR,
		FromYear: sicorXRayStartYear,
		ToYear: time.Now().Year(),
		GeneratedAt: time.Now().Format(time.RFC3339),
		CacheUntil: time.Now().Add(14*24*time.Hour).Format(time.RFC3339),
		PropertyReferences: len(refs),
		SourceURL: "https://www.bcb.gov.br/estabilidadefinanceira/tabelas-credito-rural-proagro",
		Scope: "Microdados públicos do SICOR vinculados ao CAR declarado. Operações registradas/contratadas no SICOR desde 2013; glebas somente quando publicadas pelo Banco Central. REF BACEN é contado como operação e NU_ORDEM como destinação.",
	}
	if len(refs) == 0 {
		result.Warnings = append(result.Warnings,
			"Nenhuma referência pública do SICOR foi localizada para este CAR. Isso não prova inexistência de financiamento ou crédito privado.")
		_ = saveSICORXRayCache(cachePath, result)
		return result, nil
	}

	domains, domainWarnings := a.loadSICORDomains(ctx, sourceDir)
	result.Warnings = append(result.Warnings, domainWarnings...)

	refSet := make(map[string]sicorRef, len(refs))
	for _, ref := range refs {
		refSet[sicorOperationKey(ref.RefBacen, ref.Order)] = ref
	}
	found := map[string]bool{}
	operationsByKey := map[string]*SICORPublicOperation{}
	yearKeys := map[int]map[string]bool{}

	for year := time.Now().Year(); year >= sicorXRayStartYear; year-- {
		if len(found) == len(refSet) {
			break
		}
		name := fmt.Sprintf("SICOR_OPERACAO_BASICA_ESTADO_%d.gz", year)
		path := filepath.Join(sourceDir, name)
		path, dlErr := a.ensureSICORSourceFile(ctx, sicorRawBaseURL+name, path, 36*time.Hour)
		if dlErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%d: operação básica não pôde ser lida (%v)", year, dlErr))
			continue
		}
		rows, scanErr := scanSICOROperations(path, refSet, year)
		if scanErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%d: falha ao filtrar operações (%v)", year, scanErr))
			continue
		}
		if len(rows) == 0 {
			continue
		}
		if yearKeys[year] == nil {
			yearKeys[year] = map[string]bool{}
		}
		for i := range rows {
			op := rows[i]
			key := sicorOperationKey(op.RefBacen, op.Order)
			enrichSICOROperation(&op, domains)
			operationsByKey[key] = &op
			found[key] = true
			yearKeys[year][key] = true
		}
	}

	var projectAreas []ProjectArea
	if propertyID > 0 {
		projectAreas, _ = a.ListProjectAreas(propertyID)
	} else if temp, tempErr := a.GetTemporaryProjectArea(); tempErr == nil {
		projectAreas = []ProjectArea{temp}
	}

	for year, keys := range yearKeys {
		path, dlErr := a.ensureSICORGlebaYearFile(ctx, sourceDir, year)
		if dlErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%d: glebas públicas não puderam ser obtidas (%v)", year, dlErr))
			continue
		}
		glebas, glebaErr := scanSICORGlebas(path, keys, car.GeoJSON, projectAreas)
		if glebaErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%d: falha ao processar glebas (%v)", year, glebaErr))
			continue
		}
		for _, g := range glebas {
			key := sicorOperationKey(g.RefBacen, g.Order)
			if op := operationsByKey[key]; op != nil {
				op.Glebas = append(op.Glebas, g)
			}
		}
	}

	uniqueRefs := map[string]bool{}
	for _, op := range operationsByKey {
		sort.SliceStable(op.Glebas, func(i, j int) bool { return op.Glebas[i].Index < op.Glebas[j].Index })
		result.Operations = append(result.Operations, *op)
		uniqueRefs[strings.TrimSpace(op.RefBacen)] = true
		result.TotalCreditValue += op.CreditValue
		result.TotalFinancedAreaHa += op.FinancedAreaHa
		result.GlebaCount += len(op.Glebas)
		for _, g := range op.Glebas {
			result.TotalGlebaAreaHa += g.AreaHa
		}
	}
	sort.SliceStable(result.Operations, func(i, j int) bool {
		if result.Operations[i].IssueDate == result.Operations[j].IssueDate {
			if result.Operations[i].RefBacen == result.Operations[j].RefBacen {
				return result.Operations[i].Order < result.Operations[j].Order
			}
			return result.Operations[i].RefBacen > result.Operations[j].RefBacen
		}
		return result.Operations[i].IssueDate > result.Operations[j].IssueDate
	})
	result.DestinationCount = len(result.Operations)
	result.OperationCount = len(uniqueRefs)
	result.UnresolvedReferences = len(refSet) - len(found)
	if result.UnresolvedReferences > 0 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("%d referência(s) do arquivo de propriedades não foram localizadas nos arquivos anuais de operações desde 2013.", result.UnresolvedReferences))
	}
	if result.OperationCount == 0 {
		result.Warnings = append(result.Warnings,
			"O CAR aparece no índice público de propriedades do SICOR, mas nenhuma operação compatível foi localizada nos arquivos anuais processados.")
	}


	if err := saveSICORXRayCache(cachePath, result); err != nil {
		result.Warnings = append(result.Warnings, "Não foi possível gravar o cache local do Raio X: "+err.Error())
	}
	return result, nil
}


func (a *App) ensureSICORGlebaYearFile(ctx context.Context, sourceDir string, year int) (string, error) {
	localName := fmt.Sprintf("SICOR_GLEBAS_WKT_%d.gz", year)
	path := filepath.Join(sourceDir, localName)
	if st, err := os.Stat(path); err == nil && st.Size() > 0 && time.Since(st.ModTime()) <= 36*time.Hour {
		return path, nil
	}
	var errs []string
	for _, remoteName := range []string{
		fmt.Sprintf("SICOR_GLEBAS_WKT_%d.gz", year),
		fmt.Sprintf("sicor_glebas_wkt_%d.gz", year),
	} {
		got, err := a.ensureSICORSourceFile(ctx, sicorRawBaseURL+remoteName, path, 36*time.Hour)
		if err == nil {
			return got, nil
		}
		errs = append(errs, remoteName+": "+err.Error())
		_ = os.Remove(path)
	}
	return "", errors.New(strings.Join(errs, " | "))
}

func (a *App) ClearSICORXRayCache(car string) error {
	if a == nil || a.dataDir == "" {
		return nil
	}
	if strings.TrimSpace(car) == "" {
		return errors.New("CAR não informado")
	}
	path := filepath.Join(a.dataDir, "cache", "sicor_xray", safeFilePart(car)+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func loadSICORXRayCache(path string, maxAge time.Duration) (SICORXRayResult, bool) {
	st, err := os.Stat(path)
	if err != nil || time.Since(st.ModTime()) > maxAge {
		return SICORXRayResult{}, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return SICORXRayResult{}, false
	}
	var out SICORXRayResult
	if json.Unmarshal(b, &out) != nil || strings.TrimSpace(out.CAR) == "" {
		return SICORXRayResult{}, false
	}
	return out, true
}

func saveSICORXRayCache(path string, result SICORXRayResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (a *App) ensureSICORSourceFile(ctx context.Context, targetURL, path string, maxAge time.Duration) (string, error) {
	if st, err := os.Stat(path); err == nil && st.Size() > 0 && time.Since(st.ModTime()) <= maxAge {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".part"
	_ = os.Remove(tmp)
	if err := downloadLargeFile(ctx, targetURL, tmp); err != nil {
		_ = os.Remove(tmp)
		if curlErr := downloadLargeFileCurl(ctx, targetURL, tmp); curlErr != nil {
			_ = os.Remove(tmp)
			return "", fmt.Errorf("download nativo: %v; fallback: %v", err, curlErr)
		}
	}
	if st, err := os.Stat(tmp); err != nil || st.Size() == 0 {
		_ = os.Remove(tmp)
		return "", errors.New("arquivo baixado vazio")
	}
	// No Windows, os.Rename não substitui com segurança um destino existente.
	// O arquivo novo já está completo em .part, então removemos apenas a cópia
	// antiga imediatamente antes da troca.
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return path, nil
}

func downloadLargeFile(ctx context.Context, targetURL, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 ViaVerdeCAR/"+AppVersion)
	req.Header.Set("Accept", "application/gzip,application/octet-stream,text/csv,*/*")
	client := &http.Client{Timeout: 12 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func downloadLargeFileCurl(ctx context.Context, targetURL, path string) error {
	name := "curl"
	if _, err := exec.LookPath("curl.exe"); err == nil {
		name = "curl.exe"
	} else if _, err := exec.LookPath("curl"); err != nil {
		return errors.New("curl não encontrado")
	}
	cmd := exec.CommandContext(ctx, name, "-L", "--fail", "--silent", "--show-error", "--retry", "2", "--connect-timeout", "20", "-o", path, targetURL)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(msg)
	}
	return nil
}

func pruneSICORSourceCache(dir string, maxAge time.Duration) {
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if entry.IsDir() || strings.EqualFold(entry.Name(), "SICOR_PROPRIEDADES.gz") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if st, err := entry.Info(); err == nil && time.Since(st.ModTime()) > maxAge {
			_ = os.Remove(path)
		}
	}
}

func scanSICORPropertyRefs(path, car string) ([]sicorRef, error) {
	target := normalizeSICORCARKey(car)
	seen := map[string]bool{}
	var out []sicorRef
	err := readGzipCSV(path, ';', func(headers map[string]int, row []string) error {
		carValue := fieldCSV(headers, row, "CD_CAR")
		if normalizeSICORCARKey(carValue) != target {
			return nil
		}
		ref := strings.TrimSpace(fieldCSV(headers, row, "REF_BACEN"))
		order := strings.TrimSpace(fieldCSV(headers, row, "NU_ORDEM"))
		if ref == "" || order == "" {
			return nil
		}
		key := sicorOperationKey(ref, order)
		if !seen[key] {
			seen[key] = true
			out = append(out, sicorRef{RefBacen: ref, Order: order})
		}
		return nil
	})
	return out, err
}

func scanSICOROperations(path string, targets map[string]sicorRef, year int) ([]SICORPublicOperation, error) {
	var out []SICORPublicOperation
	err := readGzipCSV(path, ';', func(headers map[string]int, row []string) error {
		ref := strings.TrimSpace(fieldCSV(headers, row, "REF_BACEN"))
		order := strings.TrimSpace(fieldCSV(headers, row, "NU_ORDEM"))
		if _, ok := targets[sicorOperationKey(ref, order)]; !ok {
			return nil
		}
		out = append(out, SICORPublicOperation{
			RefBacen: ref,
			Order: order,
			IssueDate: strings.TrimSpace(fieldCSV(headers, row, "DT_EMISSAO")),
			DueDate: strings.TrimSpace(fieldCSV(headers, row, "DT_VENCIMENTO")),
			Year: year,
			UF: strings.TrimSpace(fieldCSV(headers, row, "CD_ESTADO")),
			InstitutionCode: strings.TrimSpace(fieldCSV(headers, row, "CNPJ_IF")),
			ProgramCode: strings.TrimSpace(fieldCSV(headers, row, "CD_PROGRAMA")),
			SubprogramCode: strings.TrimSpace(fieldCSV(headers, row, "CD_SUBPROGRAMA")),
			ResourceCode: strings.TrimSpace(fieldCSV(headers, row, "CD_FONTE_RECURSO")),
			EnterpriseCode: strings.TrimSpace(fieldCSV(headers, row, "CD_EMPREENDIMENTO")),
			CreditValue: parseSICORNumber(fieldCSV(headers, row, "VL_PARC_CREDITO")),
			OwnResources: parseSICORNumber(fieldCSV(headers, row, "VL_REC_PROPRIO")),
			FinancedAreaHa: parseSICORNumber(fieldCSV(headers, row, "VL_AREA_FINANC")),
			InformedAreaHa: parseSICORNumber(fieldCSV(headers, row, "VL_AREA_INFORMADA")),
			InterestRatePct: parseSICORNumber(fieldCSV(headers, row, "VL_JUROS")),
			PostFixedInterestPct: parseSICORNumber(fieldCSV(headers, row, "VL_JUROS_ENC_FINAN_POSFIX")),
			EffectiveCostPct: parseSICORNumber(fieldCSV(headers, row, "VL_PERC_CUSTO_EFET_TOTAL")),
			InvestmentInstallment: parseSICORNumber(fieldCSV(headers, row, "VL_PRESTACAO_INVESTIMENTO")),
			ExpectedProduction: parseSICORNumber(fieldCSV(headers, row, "VL_PREV_PROD")),
			Quantity: parseSICORNumber(fieldCSV(headers, row, "VL_QUANTIDADE")),
			ExpectedGrossRevenue: parseSICORNumber(fieldCSV(headers, row, "VL_RECEITA_BRUTA_ESPERADA")),
			ObtainedProductivity: parseSICORNumber(fieldCSV(headers, row, "VL_PRODUTIV_OBTIDA")),
			ProagroRatePct: parseSICORNumber(fieldCSV(headers, row, "VL_ALIQ_PROAGRO")),
			STNRiskPct: parseSICORNumber(fieldCSV(headers, row, "VL_PERC_RISCO_STN")),
			FundRiskPct: parseSICORNumber(fieldCSV(headers, row, "VL_PERC_RISCO_FUNDO_CONST")),
			ServiceOwnResources: parseSICORNumber(fieldCSV(headers, row, "VL_REC_PROPRIO_SRV")),
			BonusCARPct: parseSICORNumber(fieldCSV(headers, row, "PC_BONUS_CAR")),
			ContractSTN: strings.TrimSpace(fieldCSV(headers, row, "CD_CONTRATO_STN")),
			RegistrantCNPJ: strings.TrimSpace(fieldCSV(headers, row, "CD_CNPJ_CADASTRANTE")),
			SoilCode: strings.TrimSpace(fieldCSV(headers, row, "CD_TIPO_SOLO")),
			CycleCode: strings.TrimSpace(fieldCSV(headers, row, "CD_CICLO_CULTIVAR")),
			InsuranceCode: strings.TrimSpace(fieldCSV(headers, row, "CD_TIPO_SEGURO")),
			InstrumentCode: strings.TrimSpace(fieldCSV(headers, row, "CD_INST_CREDITO")),
			IrrigationCode: strings.TrimSpace(fieldCSV(headers, row, "CD_TIPO_IRRIGACAO")),
			AgricultureCode: strings.TrimSpace(fieldCSV(headers, row, "CD_TIPO_AGRICULTURA")),
			CultivationCode: strings.TrimSpace(fieldCSV(headers, row, "CD_TIPO_CULTIVO")),
			IntegrationCode: strings.TrimSpace(fieldCSV(headers, row, "CD_TIPO_INTGR_CONSOR")),
			GrainSeedCode: strings.TrimSpace(fieldCSV(headers, row, "CD_TIPO_GRAO_SEMENTE")),
			ProductionPhaseCode: strings.TrimSpace(fieldCSV(headers, row, "CD_FASE_CICLO_PRODUCAO")),
			PlantingStart: strings.TrimSpace(fieldCSV(headers, row, "DT_INIC_PLANTIO")),
			PlantingEnd: strings.TrimSpace(fieldCSV(headers, row, "DT_FIM_PLANTIO")),
			HarvestStart: strings.TrimSpace(fieldCSV(headers, row, "DT_INIC_COLHEITA")),
			HarvestEnd: strings.TrimSpace(fieldCSV(headers, row, "DT_FIM_COLHEITA")),
		})
		return nil
	})
	return out, err
}

func scanSICORGlebas(path string, targets map[string]bool, carGeoJSON string, projectAreas []ProjectArea) ([]SICORGleba, error) {
	var out []SICORGleba
	err := readGzipCSV(path, ';', func(headers map[string]int, row []string) error {
		ref := strings.TrimSpace(fieldCSV(headers, row, "REF_BACEN"))
		order := strings.TrimSpace(fieldCSV(headers, row, "NU_ORDEM"))
		if !targets[sicorOperationKey(ref, order)] {
			return nil
		}
		rawWKT := strings.TrimSpace(fieldCSV(headers, row, "GT_GEOMETRIA"))
		if rawWKT == "" {
			return nil
		}
		geo, err := sicorWKTToGeoJSON(rawWKT)
		if err != nil {
			return nil
		}
		metric, err := projectAreaMetrics(geo)
		if err != nil {
			return nil
		}
		g := SICORGleba{
			RefBacen: ref,
			Order: order,
			Index: strings.TrimSpace(fieldCSV(headers, row, "NU_INDICE")),
			AreaHa: metric.AreaHa,
			GeoJSON: metric.GeoJSON,
		}
		if strings.TrimSpace(carGeoJSON) != "" {
			_, inside, _, cmpErr := estimateGeometryOverlap(metric.GeoJSON, carGeoJSON)
			if cmpErr == nil {
				g.InsideCARPct = inside
			}
		}
		for _, area := range projectAreas {
			if strings.TrimSpace(area.GeoJSON) == "" {
				continue
			}
			_, pct, _, cmpErr := estimateGeometryOverlap(area.GeoJSON, metric.GeoJSON)
			if cmpErr == nil && pct > g.ProjectOverlapPct {
				g.ProjectOverlapPct = pct
				g.ProjectOverlapName = area.Name
			}
		}
		out = append(out, g)
		return nil
	})
	return out, err
}

func readGzipCSV(path string, delimiter rune, fn func(map[string]int, []string) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	r := csv.NewReader(gz)
	r.Comma = delimiter
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	headers := make(map[string]int, len(header))
	for i, h := range header {
		headers[normalizeCSVHeader(h)] = i
	}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if err := fn(headers, row); err != nil {
			return err
		}
	}
	return nil
}

func normalizeCSVHeader(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(s, "\ufeff"))
	s = strings.TrimPrefix(s, "#")
	return strings.ToUpper(s)
}

func fieldCSV(headers map[string]int, row []string, name string) string {
	i, ok := headers[strings.ToUpper(strings.TrimPrefix(name, "#"))]
	if !ok || i < 0 || i >= len(row) {
		return ""
	}
	return decodeSICORText(row[i])
}

func normalizeSICORCARKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(s)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func sicorOperationKey(ref, order string) string {
	return strings.TrimSpace(ref) + "|" + strings.TrimSpace(order)
}

func parseSICORNumber(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	}
	n, _ := strconv.ParseFloat(s, 64)
	return n
}

func decodeSICORText(s string) string {
	if strings.ToValidUTF8(s, "") == s {
		return strings.TrimSpace(s)
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x80 {
			b.WriteByte(c)
			continue
		}
		if c >= 0xA0 {
			b.WriteRune(rune(c))
			continue
		}
		table := map[byte]rune{
			0x80:'€',0x82:'‚',0x83:'ƒ',0x84:'„',0x85:'…',0x86:'†',0x87:'‡',
			0x88:'ˆ',0x89:'‰',0x8A:'Š',0x8B:'‹',0x8C:'Œ',0x8E:'Ž',
			0x91:'‘',0x92:'’',0x93:'“',0x94:'”',0x95:'•',0x96:'–',0x97:'—',
			0x98:'˜',0x99:'™',0x9A:'š',0x9B:'›',0x9C:'œ',0x9E:'ž',0x9F:'Ÿ',
		}
		if r, ok := table[c]; ok {
			b.WriteRune(r)
		} else {
			b.WriteRune('�')
		}
	}
	return strings.TrimSpace(b.String())
}

func enrichSICOROperation(op *SICORPublicOperation, d sicorDomains) {
	if op == nil {
		return
	}
	if v, ok := d.Institutions[normalizeDomainCode(op.InstitutionCode)]; ok {
		op.InstitutionName = v[0]
		op.InstitutionType = v[1]
	}
	op.ProgramName = d.Programs[normalizeDomainCode(op.ProgramCode)]
	op.SubprogramName = d.Subprograms[normalizeDomainCode(op.SubprogramCode)]
	op.ResourceName = d.Resources[normalizeDomainCode(op.ResourceCode)]
	if e, ok := d.Enterprises[normalizeDomainCode(op.EnterpriseCode)]; ok {
		op.Purpose = e.Purpose
		op.Activity = e.Activity
		op.Modality = e.Modality
		op.Product = e.Product
		op.Variety = e.Variety
	}
}

func normalizeDomainCode(s string) string {
	return strings.TrimLeft(strings.TrimSpace(s), "0")
}

func (a *App) loadSICORDomains(ctx context.Context, sourceDir string) (sicorDomains, []string) {
	out := sicorDomains{
		Institutions: map[string][2]string{},
		Programs: map[string]string{},
		Subprograms: map[string]string{},
		Resources: map[string]string{},
		Enterprises: map[string]sicorEnterprise{},
	}
	var warnings []string
	type source struct {
		name, url, file string
	}
	sources := []source{
		{"instituições", sicorRawBaseURL+"SICOR_LISTA_IFS.csv", "SICOR_LISTA_IFS.csv"},
		{"programas", sicorDomainBase+"Programa.csv", "Programa.csv"},
		{"subprogramas", sicorDomainBase+"Subprogramas.csv", "Subprogramas.csv"},
		{"fontes", sicorDomainBase+"FonteRecursos.csv", "FonteRecursos.csv"},
		{"empreendimentos", sicorDomainBase+"Empreendimento.csv", "Empreendimento.csv"},
	}
	for _, src := range sources {
		path, err := a.ensureSICORSourceFile(ctx, src.url, filepath.Join(sourceDir, src.file), 30*24*time.Hour)
		if err != nil {
			warnings = append(warnings, src.name+": "+err.Error())
			continue
		}
		rows, err := readPlainCSVMaps(path)
		if err != nil {
			warnings = append(warnings, src.name+": "+err.Error())
			continue
		}
		switch src.name {
		case "instituições":
			for _, r := range rows {
				code := normalizeDomainCode(firstMapValue(r, "CNPJ_IF", "#CNPJ_IF"))
				if code == "" { continue }
				out.Institutions[code] = [2]string{
					firstMapValue(r, "NOME_IF", "NOME", "DESCRICAO"),
					firstMapValue(r, "SEGMENTO_IF", "SEGMENTO"),
				}
			}
		case "programas":
			for _, r := range rows {
				code := normalizeDomainCode(firstMapValue(r, "CODIGO", "#CODIGO"))
				if code != "" { out.Programs[code] = firstMapValue(r, "DESCRICAO", "PROGRAMA") }
			}
		case "subprogramas":
			for _, r := range rows {
				code := normalizeDomainCode(firstMapValue(r, "CODIGO_SUBPROGRAMA", "#CODIGO_SUBPROGRAMA", "CODIGO", "#CODIGO", "CD_SUBPROGRAMA"))
				if code != "" { out.Subprograms[code] = firstMapValue(r, "DESCRICAO_SUBPROGRAMA", "DESCRICAO", "SUBPROGRAMA") }
			}
		case "fontes":
			for _, r := range rows {
				code := normalizeDomainCode(firstMapValue(r, "CODIGO", "#CODIGO"))
				if code != "" { out.Resources[code] = firstMapValue(r, "DESCRICAO", "FONTE_RECURSO") }
			}
		case "empreendimentos":
			for _, r := range rows {
				code := normalizeDomainCode(firstMapValue(r, "CODIGO", "#CODIGO"))
				if code == "" { continue }
				out.Enterprises[code] = sicorEnterprise{
					Purpose: firstMapValue(r, "FINALIDADE"),
					Activity: firstMapValue(r, "ATIVIDADE"),
					Modality: firstMapValue(r, "MODALIDADE"),
					Product: firstMapValue(r, "PRODUTO"),
					Variety: firstMapValue(r, "VARIEDADE"),
				}
			}
		}
	}
	return out, warnings
}

func readPlainCSVMaps(path string) ([]map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := decodeSICORText(string(b))
	first := text
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		first = text[:i]
	}
	delim := ','
	if strings.Count(first, ";") > strings.Count(first, ",") {
		delim = ';'
	}
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = delim
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	var out []map[string]string
	for {
		row, err := r.Read()
		if err == io.EOF { break }
		if err != nil { continue }
		m := map[string]string{}
		for i, value := range row {
			if i >= len(header) { break }
			m[normalizeCSVHeader(header[i])] = strings.TrimSpace(value)
		}
		out = append(out, m)
	}
	return out, nil
}

func firstMapValue(m map[string]string, names ...string) string {
	for _, name := range names {
		key := normalizeCSVHeader(name)
		if v := strings.TrimSpace(m[key]); v != "" {
			return v
		}
	}
	return ""
}

func (r SICORXRayResult) TotalProjectConflictGlebas() int {
	n := 0
	for _, op := range r.Operations {
		for _, g := range op.Glebas {
			if g.ProjectOverlapPct > 0.5 {
				n++
			}
		}
	}
	return n
}

func safePercent(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 { return 0 }
	if v > 100 { return 100 }
	return v
}


func (a *App) ExportSICORXRayCSV(car string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	car = strings.TrimSpace(car)
	if car == "" {
		return "", errors.New("CAR não informado")
	}
	cachePath := filepath.Join(a.dataDir, "cache", "sicor_xray", safeFilePart(car)+".json")
	result, ok := loadSICORXRayCache(cachePath, 365*24*time.Hour)
	if !ok {
		return "", errors.New("monte o Raio X do SICOR antes de exportar")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Salvar operações públicas do SICOR",
		DefaultFilename: "Raio_X_SICOR_" + safeFilePart(car) + ".csv",
		Filters: []runtime.FileFilter{{DisplayName: "CSV", Pattern: "*.csv"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Comma = ';'
	defer w.Flush()
	_ = w.Write([]string{
		"REF_BACEN","ORDEM","EMISSAO","VENCIMENTO","INSTITUICAO","SEGMENTO_IF",
		"PROGRAMA","SUBPROGRAMA","FONTE_RECURSO","FINALIDADE","ATIVIDADE","MODALIDADE","PRODUTO",
		"VALOR_CREDITO","AREA_FINANCIADA_HA","GLEBAS","AREA_GLEBAS_HA","CONFLITO_PROJETO_MAX_PCT",
	})
	for _, op := range result.Operations {
		glebaArea := 0.0
		maxConflict := 0.0
		for _, g := range op.Glebas {
			glebaArea += g.AreaHa
			if g.ProjectOverlapPct > maxConflict {
				maxConflict = g.ProjectOverlapPct
			}
		}
		_ = w.Write([]string{
			op.RefBacen, op.Order, op.IssueDate, op.DueDate,
			op.InstitutionName, op.InstitutionType, op.ProgramName, op.SubprogramName,
			op.ResourceName, op.Purpose, op.Activity, op.Modality, op.Product,
			strconv.FormatFloat(op.CreditValue, 'f', 2, 64),
			strconv.FormatFloat(op.FinancedAreaHa, 'f', 4, 64),
			strconv.Itoa(len(op.Glebas)),
			strconv.FormatFloat(glebaArea, 'f', 4, 64),
			strconv.FormatFloat(maxConflict, 'f', 2, 64),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return path, nil
}
