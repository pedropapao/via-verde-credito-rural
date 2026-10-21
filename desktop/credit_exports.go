package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ExportSICORGlebasKML(propertyID int64, force bool) (string, error) {
	if a == nil || a.db == nil {
		return "", errors.New("banco local indisponível")
	}
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	result, err := a.GetCreditIntelligence(propertyID, force)
	if err != nil {
		return "", err
	}
	kml, glebaCount, operationCount := buildSICORGlebasKML(result)
	if glebaCount == 0 {
		return "", errors.New("nenhuma geometria de gleba financiada foi publicada/localizada no SICOR para este CAR")
	}
	name := "Glebas_SICOR_" + safeCARFilename(result.CAR) + ".kml"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvar KML das glebas financiadas no SICOR",
		DefaultFilename: name,
		Filters:         []runtime.FileFilter{{DisplayName: "KML", Pattern: "*.kml"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".kml") {
		path += ".kml"
	}
	if err := os.WriteFile(path, kml, 0o644); err != nil {
		return "", err
	}
	_ = operationCount
	return path, nil
}

func buildSICORGlebasKML(result CreditIntelligenceResult) ([]byte, int, int) {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<kml xmlns=\"http://www.opengis.net/kml/2.2\"><Document>\n")
	b.WriteString("<name>" + kmlXMLEscape("ViaVerdeCAR - Glebas SICOR - "+result.CAR) + "</name>\n")
	b.WriteString("<description>" + kmlXMLEscape("Glebas publicadas nos microdados SICOR/BCB para operações vinculadas ao CAR. A ausência de geometria não prova ausência de financiamento.") + "</description>\n")
	b.WriteString("<Style id=\"sicor_gleba\"><LineStyle><color>ff7c3aed</color><width>3</width></LineStyle><PolyStyle><color>338b5cf6</color></PolyStyle></Style>\n")

	glebaCount := 0
	operationCount := 0
	seen := map[string]bool{}
	for _, op := range result.Operations {
		valid := make([]SICORGleba, 0, len(op.Glebas))
		for _, g := range op.Glebas {
			if strings.TrimSpace(g.GeoJSON) == "" {
				continue
			}
			key := strings.TrimSpace(op.RefBacen) + "|" + strings.TrimSpace(op.Order) + "|" + strings.TrimSpace(g.Index) + "|" + strings.TrimSpace(g.GeoJSON)
			if seen[key] {
				continue
			}
			seen[key] = true
			valid = append(valid, g)
		}
		if len(valid) == 0 {
			continue
		}
		operationCount++
		folderName := fmt.Sprintf("REF %s - ordem %s - %s", firstNonEmptyText(op.RefBacen, "não informada"), firstNonEmptyText(op.Order, "não informada"), firstNonEmptyText(op.Product, op.Purpose, "Crédito rural"))
		b.WriteString("<Folder><name>" + kmlXMLEscape(folderName) + "</name>\n")
		for i, g := range valid {
			features := geoJSONFeatures(g.GeoJSON)
			if len(features) == 0 {
				continue
			}
			glebaName := fmt.Sprintf("Gleba %s", firstNonEmptyText(g.Index, fmt.Sprintf("%d", i+1)))
			desc := []string{
				"REF BACEN: " + firstNonEmptyText(op.RefBacen, "não informado"),
				"Ordem/destinação: " + firstNonEmptyText(op.Order, "não informado"),
				"Instituição: " + firstNonEmptyText(op.Institution, "não informada"),
				"Programa: " + firstNonEmptyText(op.Program, "não informado"),
				"Produto: " + firstNonEmptyText(op.Product, "não informado"),
				"Valor contratado: " + moneyBR(op.CreditValue),
				"Área financiada da operação: " + fmtBR(op.FinancedAreaHa, 4) + " ha",
				"Área geométrica da gleba: " + fmtBR(g.AreaHa, 4) + " ha",
				"Percentual da gleba dentro do CAR: " + fmtBR(g.InsideCARPct, 2) + "%",
			}
			if strings.TrimSpace(g.ProjectOverlapName) != "" {
				desc = append(desc, "Sobreposição com área local "+g.ProjectOverlapName+": "+fmtBR(g.ProjectOverlapPct, 2)+"%")
			}
			for fi, feature := range features {
				polys, err := carGeometryPolygons(feature.Geometry)
				if err != nil || len(polys) == 0 {
					continue
				}
				name := glebaName
				if len(features) > 1 {
					name += fmt.Sprintf(" - feição %d", fi+1)
				}
				b.WriteString("<Placemark><name>" + kmlXMLEscape(name) + "</name>")
				b.WriteString("<description>" + kmlXMLEscape(strings.Join(desc, "\n")) + "</description>")
				b.WriteString("<styleUrl>#sicor_gleba</styleUrl><ExtendedData>")
				writeSICORKMLData(&b, "ref_bacen", op.RefBacen)
				writeSICORKMLData(&b, "ordem", op.Order)
				writeSICORKMLData(&b, "instituicao", op.Institution)
				writeSICORKMLData(&b, "programa", op.Program)
				writeSICORKMLData(&b, "produto", op.Product)
				writeSICORKMLData(&b, "valor_credito", fmt.Sprintf("%.2f", op.CreditValue))
				writeSICORKMLData(&b, "area_gleba_ha", fmt.Sprintf("%.6f", g.AreaHa))
				writeSICORKMLData(&b, "inside_car_pct", fmt.Sprintf("%.4f", g.InsideCARPct))
				b.WriteString("</ExtendedData><MultiGeometry>")
				for _, poly := range polys {
					writeKMLPolygon(&b, poly)
				}
				b.WriteString("</MultiGeometry></Placemark>\n")
			}
			glebaCount++
		}
		b.WriteString("</Folder>\n")
	}
	b.WriteString("</Document></kml>\n")
	return []byte(b.String()), glebaCount, operationCount
}

func writeSICORKMLData(b *strings.Builder, name, value string) {
	if b == nil || strings.TrimSpace(value) == "" {
		return
	}
	b.WriteString("<Data name=\"" + kmlXMLEscape(name) + "\"><value>" + kmlXMLEscape(value) + "</value></Data>")
}

func (a *App) ExportCreditIntelligencePDF(propertyID int64, force bool) (string, error) {
	if a == nil || a.db == nil {
		return "", errors.New("banco local indisponível")
	}
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	var property Property
	if propertyID > 0 {
		if saved, getErr := a.GetProperty(propertyID); getErr == nil {
			property = saved
		}
	}
	result, err := a.GetCreditIntelligence(propertyID, force)
	if err != nil {
		return "", err
	}
	pdf := buildCreditIntelligencePDF(property, result)
	name := "Credito_Rural_SICOR_" + safeCARFilename(result.CAR) + ".pdf"
	return a.saveProfessionalPDF("Salvar relatório completo de crédito rural", name, pdf)
}

func buildCreditIntelligencePDF(property Property, result CreditIntelligenceResult) []byte {
	pages := []string{}
	page := 1
	pages = append(pages, creditReportSummaryPage(property, result, page))
	page++
	for i := range result.Operations {
		op := result.Operations[i]
		pages = append(pages, creditReportOperationMainPage(property, result, op, i+1, page))
		page++
		pages = append(pages, creditReportOperationDetailPage(property, result, op, i+1, page))
		page++
		pages = append(pages, creditReportOperationRiskPage(property, result, op, i+1, page))
		page++
	}
	if len(result.Warnings) > 0 {
		pages = append(pages, creditReportWarningsPage(property, result, page))
	}
	return assembleMultiPagePDF(pages)
}

func creditReportSummaryPage(property Property, result CreditIntelligenceResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "CRÉDITO RURAL • SICOR/BCB", "Relatório consolidado das operações públicas vinculadas ao CAR", page)
	y := 716.0
	proSection(&c, &y, "IDENTIFICAÇÃO", "microdados públicos do Sistema de Operações do Crédito Rural")
	proKV(&c, &y, "Cliente", property.ClientName)
	proKV(&c, &y, "Imóvel", property.Name)
	proKV(&c, &y, "CAR", result.CAR)
	proKV(&c, &y, "Município / UF", strings.Trim(strings.TrimSpace(result.Municipality+" / "+result.UF), " /"))
	proKV(&c, &y, "Matrícula / registro local", property.Registry)
	proKV(&c, &y, "Consulta detalhada", dateBR(result.GeneratedAt))
	y -= 4

	proSection(&c, &y, "RESUMO FINANCEIRO", "")
	proMetric(&c, 40, y-72, 122, 64, "OPERAÇÕES", fmt.Sprintf("%d", result.Totals.Operations), "destinações detalhadas localizadas", "purple")
	proMetric(&c, 171, y-72, 122, 64, "CONTRATADO", moneyBR(result.Totals.Contracted), "soma dos valores públicos localizados", "purple")
	proMetric(&c, 302, y-72, 122, 64, "LIBERADO", moneyBR(result.Totals.Released), "recursos liberados publicados", "blue")
	proMetric(&c, 433, y-72, 122, 64, "ÚLTIMOS SALDOS", moneyBR(result.Totals.LatestBalances), fmt.Sprintf("%d operação(ões) com saldo", result.Totals.WithBalance), "")
	y -= 94
	proMetric(&c, 40, y-72, 122, 64, "RECURSOS PRÓPRIOS", moneyBR(result.Totals.OwnResources), "contrapartida publicada", "")
	proMetric(&c, 171, y-72, 122, 64, "DESEMBOLSO PREVISTO", moneyBR(result.Totals.PlannedDisbursement), "cronograma publicado", "")
	proMetric(&c, 302, y-72, 122, 64, "RENEG./PRORROG.", fmt.Sprintf("%d", result.Totals.WithRenegotiation), "operações com vínculo localizado", "warn")
	proMetric(&c, 433, y-72, 122, 64, "PROAGRO", fmt.Sprintf("%d", result.Totals.WithProagro), moneyBR(result.Totals.ProagroPaid)+" pagos publicados", "")
	y -= 92

	proSection(&c, &y, "OPERAÇÕES LOCALIZADAS", "")
	if len(result.Operations) == 0 {
		proNotice(&c, &y, "SEM OPERAÇÃO DETALHADA LOCALIZADA", "Nenhuma operação pública vinculada ao CAR foi localizada nesta execução. Isso não prova inexistência de financiamento, dívida ou operação privada.", "warn")
	} else {
		for i, op := range result.Operations {
			if y < 115 {
				break
			}
			label := fmt.Sprintf("%02d • REF %s • ordem %s", i+1, firstNonEmptyText(op.RefBacen, "—"), firstNonEmptyText(op.Order, "—"))
			value := fmt.Sprintf("%s • %s • %s • %s", firstNonEmptyText(op.Institution, "instituição não informada"), firstNonEmptyText(op.Product, op.Purpose, "crédito rural"), moneyBR(op.CreditValue), firstNonEmptyText(dateBR(op.IssueDate), fmt.Sprintf("ano %d", op.Year)))
			proKV(&c, &y, label, value)
		}
	}
	proNotice(&c, &y, "LEITURA CORRETA", result.Scope, "")
	proFooter(&c, page)
	return c.b.String()
}

func creditReportOperationMainPage(property Property, result CreditIntelligenceResult, op CreditOperationIntelligence, index, page int) string {
	var c pdfCanvas
	proHeader(&c, "CRÉDITO RURAL • SICOR/BCB", fmt.Sprintf("Operação %d de %d • identificação, enquadramento e financeiro", index, len(result.Operations)), page)
	y := 716.0
	proSection(&c, &y, "IDENTIFICAÇÃO DA OPERAÇÃO", "")
	creditKV(&c, &y, "REF BACEN", op.RefBacen)
	creditKV(&c, &y, "REF BACEN efetivo", op.EffectiveRefBacen)
	creditKV(&c, &y, "Ordem / destinação", op.Order)
	creditKV(&c, &y, "Ano", fmt.Sprint(op.Year))
	creditKV(&c, &y, "Instituição financeira", op.Institution)
	creditKV(&c, &y, "Agência", op.AgencyCode)
	creditKV(&c, &y, "Município IBGE da operação", op.MunicipalityCode)
	creditKV(&c, &y, "Instituição cadastrante", op.RegisteringInstitution)
	creditKV(&c, &y, "Programa", op.Program)
	creditKV(&c, &y, "Subprograma", op.Subprogram)
	creditKV(&c, &y, "Fonte de recursos", op.Resource)
	creditKV(&c, &y, "Instrumento", firstNonEmptyText(op.Instrument, op.InstrumentCode))
	creditKV(&c, &y, "Categoria do emitente", firstNonEmptyText(op.IssuerCategory, op.IssuerCategoryCode))
	creditKV(&c, &y, "Seguro / garantia", firstNonEmptyText(op.Insurance, op.InsuranceCode))
	creditKV(&c, &y, "Emissão", dateBR(op.IssueDate))
	creditKV(&c, &y, "Vencimento", dateBR(op.DueDate))

	if y > 240 {
		proSection(&c, &y, "FINANCEIRO", "")
		creditKV(&c, &y, "Valor contratado", moneyBR(op.CreditValue))
		creditKV(&c, &y, "Recursos próprios", moneyBR(op.OwnResources))
		creditKV(&c, &y, "Parcela de crédito", moneyBR(op.CreditInstallment))
		creditKV(&c, &y, "Prestação de investimento", moneyBR(op.InvestmentInstallment))
		creditKV(&c, &y, "Taxa de juros", percentBR(op.InterestRatePct))
		creditKV(&c, &y, "Juros pós-fixados", percentBR(op.PostFixedInterestPct))
		creditKV(&c, &y, "Custo efetivo total", percentBR(op.EffectiveCostPct))
		creditKV(&c, &y, "Risco STN", percentBR(op.STNRiskPct))
		creditKV(&c, &y, "Risco fundo constitucional", percentBR(op.ConstitutionalFundRiskPct))
		creditKV(&c, &y, "Garantia de renda mínima", moneyBR(op.MinimumIncomeGuarantee))
		creditKV(&c, &y, "Contrato STN", op.STNContractCode)
	}
	proFooter(&c, page)
	return c.b.String()
}

func creditReportOperationDetailPage(property Property, result CreditIntelligenceResult, op CreditOperationIntelligence, index, page int) string {
	var c pdfCanvas
	proHeader(&c, "CRÉDITO RURAL • SICOR/BCB", fmt.Sprintf("Operação %d de %d • produção, liberações e situação", index, len(result.Operations)), page)
	y := 716.0
	proSection(&c, &y, "EMPREENDIMENTO E PRODUÇÃO", "")
	for _, row := range []struct{ l, v string }{
		{"Finalidade", op.Purpose},
		{"Atividade", op.Activity},
		{"Modalidade", op.Modality},
		{"Produto", op.Product},
		{"Variedade", op.Variety},
		{"Área financiada", hectareBR(op.FinancedAreaHa)},
		{"Área informada", hectareBR(op.InformedAreaHa)},
		{"Previsão de produção", numberBR(op.ForecastProduction)},
		{"Quantidade", numberBR(op.Quantity)},
		{"Receita bruta esperada", moneyBR(op.ExpectedRevenue)},
		{"Produtividade obtida", numberBR(op.ProductivityObtained)},
		{"Irrigação", firstNonEmptyText(op.Irrigation, op.IrrigationCode)},
		{"Agricultura", firstNonEmptyText(op.Agriculture, op.AgricultureCode)},
		{"Tipo de cultivo", firstNonEmptyText(op.CropType, op.CropTypeCode)},
		{"Integração / consórcio", firstNonEmptyText(op.Integration, op.IntegrationCode)},
		{"Grão / semente", firstNonEmptyText(op.SeedType, op.SeedTypeCode)},
		{"Fase produtiva", firstNonEmptyText(op.ProductionPhase, op.ProductionPhaseCode)},
		{"Ciclo da cultivar", firstNonEmptyText(op.CultivarCycle, op.CultivarCycleCode)},
		{"Solo", firstNonEmptyText(op.Soil, op.SoilCode)},
		{"Plantio", dateRangeBR(op.PlantingStart, op.PlantingEnd)},
		{"Colheita", dateRangeBR(op.HarvestStart, op.HarvestEnd)},
	} {
		creditKV(&c, &y, row.l, row.v)
	}

	proSection(&c, &y, "LIBERAÇÕES E DESEMBOLSOS", "")
	creditKV(&c, &y, "Total liberado", moneyBR(op.TotalReleased))
	creditKV(&c, &y, "Última liberação", dateBR(op.LastReleaseDate))
	creditKV(&c, &y, "Liberações publicadas", creditReleaseText(op.Releases))
	creditKV(&c, &y, "Desembolso previsto", moneyBR(op.PlannedDisbursement))
	creditKV(&c, &y, "Cronograma de desembolso", creditDisbursementText(op.Disbursements))

	if y > 120 {
		proSection(&c, &y, "SALDO E SITUAÇÃO", "")
		if op.Balance.Found {
			creditKV(&c, &y, "Situação", firstNonEmptyText(op.Balance.Status, op.Balance.StatusCode))
			creditKV(&c, &y, "Data-base do saldo", fmt.Sprintf("%02d/%d", op.Balance.Month, op.Balance.Year))
			creditKV(&c, &y, "Saldo no último dia", moneyBR(op.Balance.LastDay))
			creditKV(&c, &y, "Saldo médio diário", moneyBR(op.Balance.DailyAverage))
			creditKV(&c, &y, "Saldo médio vincendo", moneyBR(op.Balance.DueDailyAverage))
		} else {
			creditKV(&c, &y, "Situação / saldo", "Não localizado nos arquivos públicos processados.")
		}
		if op.Declassification.Found {
			creditKV(&c, &y, "Desclassificação", strings.TrimSpace(dateBR(op.Declassification.Date)+" • "+firstNonEmptyText(op.Declassification.Reason, op.Declassification.ReasonCode)+" • "+op.Declassification.Type+" • "+moneyBR(op.Declassification.Value)))
		}
	}
	proFooter(&c, page)
	return c.b.String()
}

func creditReportOperationRiskPage(property Property, result CreditIntelligenceResult, op CreditOperationIntelligence, index, page int) string {
	var c pdfCanvas
	proHeader(&c, "CRÉDITO RURAL • SICOR/BCB", fmt.Sprintf("Operação %d de %d • renegociação, Proagro, ZARC e glebas", index, len(result.Operations)), page)
	y := 716.0
	proSection(&c, &y, "RENEGOCIAÇÕES / PRORROGAÇÕES", "")
	if len(op.Renegotiations) == 0 {
		creditKV(&c, &y, "Resultado", "Nenhum vínculo público de renegociação/prorrogação foi localizado para esta destinação.")
	} else {
		for i, ren := range op.Renegotiations {
			creditKV(&c, &y, fmt.Sprintf("Vínculo %d", i+1), strings.TrimSpace(firstNonEmptyText(ren.Direction, "relação SICOR")+" • REF "+ren.RelatedRef+" / "+ren.RelatedOrder+" • "+firstNonEmptyText(ren.LegalBasis, ren.LegalBasisCode)+" • "+moneyBR(ren.Value)))
		}
	}

	proSection(&c, &y, "PROAGRO", "")
	p := op.Proagro
	creditKV(&c, &y, "Alíquota Proagro", percentBR(op.ProagroRatePct))
	creditKV(&c, &y, "COP", boolRecordText(p.HasCOP, p.COPDate, firstNonEmptyText(p.COPStatus, p.COPStatusCode)))
	creditKV(&c, &y, "Evento", firstNonEmptyText(p.Event, p.EventCode))
	creditKV(&c, &y, "Ciclo", firstNonEmptyText(p.Cycle, p.CycleCode))
	creditKV(&c, &y, "Solo", firstNonEmptyText(p.Soil, p.SoilCode))
	creditKV(&c, &y, "Período plantio", dateRangeBR(p.PlantingStart, p.PlantingEnd))
	creditKV(&c, &y, "Período colheita", dateRangeBR(p.HarvestStart, p.HarvestEnd))
	creditKV(&c, &y, "RCP", boolRecordText(p.HasRCP, p.RCPDate, p.RCPStatusCode))
	creditKV(&c, &y, "Área RCP", hectareBR(p.RCPAreaHa))
	creditKV(&c, &y, "Produção prevista RCP", numberBR(p.RCPForecastProd))
	creditKV(&c, &y, "Receita prevista RCP", moneyBR(p.RCPForecastRev))
	creditKV(&c, &y, "Julgamento", boolRecordText(p.HasJudgment, p.JudgmentDate, p.DecisionCode))
	creditKV(&c, &y, "Cobertura do crédito", moneyBR(p.CoverageCredit))
	creditKV(&c, &y, "Cobertura recursos próprios", moneyBR(p.CoverageOwn))
	creditKV(&c, &y, "Perdas não amparadas", moneyBR(p.LossesUncovered))
	creditKV(&c, &y, "Valor pago", moneyBR(p.PaidValue))

	proSection(&c, &y, "ZARC", "")
	z := op.ZARC
	creditKV(&c, &y, "Resultado", z.Status)
	creditKV(&c, &y, "Safra", z.Safra)
	creditKV(&c, &y, "Cultura", firstNonEmptyText(z.Culture, op.Product))
	creditKV(&c, &y, "Ciclo", firstNonEmptyText(z.Cycle, op.CultivarCycle))
	creditKV(&c, &y, "Solo", firstNonEmptyText(z.Soil, op.Soil))
	creditKV(&c, &y, "Manejo", z.Management)
	creditKV(&c, &y, "Portaria", z.Portaria)
	creditKV(&c, &y, "Decêndios", zarcPeriodsText(z.Periods))
	creditKV(&c, &y, "Observação ZARC", z.Message)

	proSection(&c, &y, "GLEBAS FINANCIADAS PUBLICADAS", "")
	if len(op.Glebas) == 0 {
		creditKV(&c, &y, "Resultado", "Nenhuma geometria de gleba foi publicada/localizada para esta destinação.")
	} else {
		for i, g := range op.Glebas {
			creditKV(&c, &y, fmt.Sprintf("Gleba %s", firstNonEmptyText(g.Index, fmt.Sprintf("%d", i+1))), fmt.Sprintf("%s ha • %s%% dentro do CAR%s", fmtBR(g.AreaHa, 4), fmtBR(g.InsideCARPct, 2), func() string {
				if strings.TrimSpace(g.ProjectOverlapName) == "" {
					return ""
				}
				return " • " + g.ProjectOverlapName + " " + fmtBR(g.ProjectOverlapPct, 2) + "%"
			}()))
		}
	}
	proFooter(&c, page)
	return c.b.String()
}

func creditReportWarningsPage(property Property, result CreditIntelligenceResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "CRÉDITO RURAL • SICOR/BCB", "Fontes, limitações e avisos da consulta", page)
	y := 716.0
	proSection(&c, &y, "FONTES", "")
	creditKV(&c, &y, "Microdados SICOR / BCB", result.BCBSourceURL)
	creditKV(&c, &y, "Manual de Crédito Rural", result.MCRSourceURL)
	creditKV(&c, &y, "MAPA / ZARC", result.ZARCSourceURL)
	proSection(&c, &y, "AVISOS", "")
	for i, warning := range result.Warnings {
		if y < 120 {
			break
		}
		creditKV(&c, &y, fmt.Sprintf("Aviso %d", i+1), warning)
	}
	proNotice(&c, &y, "ESCOPO", result.Scope, "warn")
	proFooter(&c, page)
	return c.b.String()
}

func creditKV(c *pdfCanvas, y *float64, label, value string) {
	if c == nil || y == nil || strings.TrimSpace(value) == "" || value == "R$ 0,00" || value == "0" || value == "0,00 ha" || value == "0,00%" {
		return
	}
	c.b.WriteString("0.39 0.45 0.41 rg\n")
	c.text(44, *y, 6.6, true, label)
	c.b.WriteString("0.10 0.18 0.14 rg\n")
	ny := c.wrapped(172, *y, 7.3, false, value, 68, 8.7)
	*y = ny - 3.5
}

func percentBR(v float64) string {
	if v == 0 {
		return ""
	}
	return fmtBR(v, 4) + "%"
}

func hectareBR(v float64) string {
	if v == 0 {
		return ""
	}
	return fmtBR(v, 4) + " ha"
}

func numberBR(v float64) string {
	if v == 0 {
		return ""
	}
	return fmtBR(v, 4)
}

func dateRangeBR(start, end string) string {
	a, b := dateBR(start), dateBR(end)
	if strings.TrimSpace(a) == "" && strings.TrimSpace(b) == "" {
		return ""
	}
	if strings.TrimSpace(b) == "" {
		return a
	}
	if strings.TrimSpace(a) == "" {
		return b
	}
	return a + " a " + b
}

func creditReleaseText(rows []CreditRelease) string {
	if len(rows) == 0 {
		return ""
	}
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, dateBR(row.Date)+" • "+moneyBR(row.Value))
	}
	return strings.Join(parts, " | ")
}

func creditDisbursementText(rows []CreditDisbursement) string {
	if len(rows) == 0 {
		return ""
	}
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, dateBR(row.Date)+" • "+moneyBR(row.Value))
	}
	return strings.Join(parts, " | ")
}

func boolRecordText(found bool, date, detail string) string {
	if !found && strings.TrimSpace(date) == "" && strings.TrimSpace(detail) == "" {
		return ""
	}
	if found {
		return strings.TrimSpace(firstNonEmptyText(dateBR(date), "registro localizado") + " • " + detail)
	}
	return detail
}

func zarcPeriodsText(rows []ZARCPeriod) string {
	if len(rows) == 0 {
		return ""
	}
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		status := "sem indicação"
		if row.Indicated {
			if row.RiskPct > 0 {
				status = fmt.Sprintf("%d%%", row.RiskPct)
			} else {
				status = firstNonEmptyText(row.Raw, "indicado")
			}
		}
		parts = append(parts, fmt.Sprintf("D%d: %s", row.Decendio, status))
	}
	return strings.Join(parts, " | ")
}

func initCreditExportGeneratedAt() string {
	return time.Now().Format(time.RFC3339)
}
