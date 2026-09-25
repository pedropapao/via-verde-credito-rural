package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportEnvironmentalEvidencePDF gera um caderno em PDF para conferência e
// rastreabilidade. O JSON estruturado continua disponível na aba Documentos.
func (a *App) ExportEnvironmentalEvidencePDF(propertyID int64, force bool) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	if propertyID <= 0 {
		return "", errors.New("salve ou selecione um imóvel antes de gerar as evidências ambientais")
	}
	p, err := a.GetProperty(propertyID)
	if err != nil {
		return "", err
	}
	car, err := a.carForEnvironmental(propertyID)
	if err != nil {
		return "", err
	}
	intel, err := a.GetEnvironmentalIntelligence(propertyID, force)
	if err != nil {
		return "", err
	}
	pdf := buildEnvironmentalEvidencePDF(p, car, intel)
	name := "Caderno_Evidencias_Ambientais_" + safeCARFilename(car.CAR) + ".pdf"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvar caderno de evidências ambientais",
		DefaultFilename: name,
		Filters:         []runtime.FileFilter{{DisplayName: "PDF", Pattern: "*.pdf"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ExportPropertyTechnicalDossierPDF gera um PDF consolidado. Não substitui o
// ZIP técnico já existente: o ZIP permanece disponível na aba Documentos.
func (a *App) ExportPropertyTechnicalDossierPDF(propertyID int64, force bool) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	if propertyID <= 0 {
		return "", errors.New("salve ou selecione um imóvel antes de gerar o dossiê")
	}
	p, err := a.GetProperty(propertyID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(p.CARNumber) == "" {
		return "", errors.New("o imóvel precisa ter um CAR vinculado para gerar o dossiê técnico")
	}
	result, err := a.RunCARAutomation(p.CARNumber, propertyID, force)
	if err != nil {
		return "", err
	}
	pdf := buildPropertyTechnicalDossierPDF(p, result)
	name := "Dossie_Tecnico_" + safeFilePart(p.Name) + "_" + safeCARFilename(result.CAR.CAR) + ".pdf"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salvar dossiê técnico do imóvel",
		DefaultFilename: name,
		Filters:         []runtime.FileFilter{{DisplayName: "PDF", Pattern: "*.pdf"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func proHeader(c *pdfCanvas, title, subtitle string, page int) {
	c.b.WriteString("0.035 0.31 0.20 rg\n")
	c.rect(0, 760, 595, 82, true)
	c.b.WriteString("0.08 0.50 0.25 rg\n")
	c.rect(0, 760, 8, 82, true)
	c.b.WriteString("1 1 1 rg\n")
	c.text(38, 810, 17, true, "VIA VERDE")
	c.text(38, 790, 8.5, true, "CONSULTORIA AGRICOLA")
	c.text(210, 810, 11.5, true, title)
	c.text(210, 790, 7.2, false, subtitle)
	c.text(525, 775, 6.2, false, fmt.Sprintf("p. %d", page))
}

func proFooter(c *pdfCanvas, page int) {
	c.b.WriteString("0.82 0.87 0.84 RG 0.5 w\n")
	c.line(38, 48, 557, 48)
	c.b.WriteString("0.34 0.42 0.38 rg\n")
	c.text(38, 32, 6.2, false, "Via Verde Consultoria Agricola - documento tecnico auxiliar - gerado em "+time.Now().Format("02/01/2006 15:04"))
	c.text(524, 32, 6.2, false, fmt.Sprintf("p. %d", page))
}

func proSection(c *pdfCanvas, y *float64, title, subtitle string) {
	c.b.WriteString("0.06 0.34 0.22 rg\n")
	c.text(40, *y, 9, true, title)
	if strings.TrimSpace(subtitle) != "" {
		c.b.WriteString("0.38 0.46 0.41 rg\n")
		c.text(235, *y, 6.4, false, subtitle)
	}
	c.b.WriteString("0.82 0.87 0.84 RG 0.6 w\n")
	c.line(40, *y-6, 555, *y-6)
	*y -= 22
}

func proKV(c *pdfCanvas, y *float64, label, value string) {
	if strings.TrimSpace(value) == "" {
		value = "Não informado"
	}
	c.b.WriteString("0.38 0.45 0.41 rg\n")
	c.text(44, *y, 7.2, true, label)
	c.b.WriteString("0.10 0.18 0.14 rg\n")
	ny := c.wrapped(175, *y, 8.2, false, value, 65, 10)
	*y = ny - 5
}

func proMetric(c *pdfCanvas, x, y, w, h float64, label, value, detail, tone string) {
	switch tone {
	case "danger":
		c.b.WriteString("1.00 0.95 0.94 rg\n")
	case "warn":
		c.b.WriteString("1.00 0.98 0.91 rg\n")
	case "purple":
		c.b.WriteString("0.97 0.95 1.00 rg\n")
	case "blue":
		c.b.WriteString("0.95 0.98 1.00 rg\n")
	default:
		c.b.WriteString("0.95 0.98 0.96 rg\n")
	}
	c.rect(x, y, w, h, true)
	c.b.WriteString("0.80 0.86 0.82 RG 0.6 w\n")
	c.rect(x, y, w, h, false)
	c.b.WriteString("0.35 0.43 0.38 rg\n")
	c.text(x+9, y+h-15, 6.3, true, label)
	c.b.WriteString("0.06 0.31 0.20 rg\n")
	c.text(x+9, y+h-34, 11, true, value)
	c.b.WriteString("0.38 0.45 0.41 rg\n")
	for i, line := range pdfWrap(detail, 34) {
		if i >= 2 {
			break
		}
		c.text(x+9, y+10+float64(1-i)*8, 5.7, false, line)
	}
}

func proParagraph(c *pdfCanvas, y *float64, text string) {
	c.b.WriteString("0.18 0.26 0.21 rg\n")
	*y = c.wrapped(44, *y, 7.8, false, text, 100, 10.5) - 5
}

func proNotice(c *pdfCanvas, y *float64, title, text, tone string) {
	switch tone {
	case "danger":
		c.b.WriteString("1.00 0.94 0.93 rg\n")
	case "warn":
		c.b.WriteString("1.00 0.98 0.90 rg\n")
	default:
		c.b.WriteString("0.94 0.98 0.95 rg\n")
	}
	lines := pdfWrap(text, 96)
	h := 30.0 + float64(len(lines))*9
	if h > 92 {
		h = 92
	}
	c.rect(40, *y-h+10, 515, h, true)
	c.b.WriteString("0.06 0.31 0.20 rg\n")
	c.text(50, *y-8, 7.5, true, title)
	c.b.WriteString("0.24 0.32 0.27 rg\n")
	yy := *y - 22
	for i, line := range lines {
		if i >= 7 {
			break
		}
		c.text(50, yy, 6.8, false, line)
		yy -= 9
	}
	*y -= h + 8
}

func reportLookupLabel(car CARResult) string {
	switch strings.ToLower(strings.TrimSpace(car.LookupStatus)) {
	case "cached":
		return "Cache local - fonte indisponível no momento"
	case "partial":
		return "Consulta pública parcial"
	case "unavailable":
		return "Base pública indisponível"
	case "not_found":
		return "CAR não localizado"
	default:
		if strings.TrimSpace(car.Status) != "" {
			return car.Status
		}
		if car.Found || car.HasGeometry {
			return "Localizado"
		}
		return "Não confirmado"
	}
}

func reportLookupTone(car CARResult) string {
	switch strings.ToLower(strings.TrimSpace(car.LookupStatus)) {
	case "unavailable", "not_found":
		return "danger"
	case "partial", "cached":
		return "warn"
	default:
		return "blue"
	}
}

func buildCARProfessionalPDF(p Property, car CARResult, kml KMLResult, cmp GeometryComparison) []byte {
	pages := []string{
		carProfessionalSummaryPage(p, car, kml, cmp, 1),
		carProfessionalMapPage(p, car, kml, cmp, 2),
	}
	return assembleMultiPagePDF(pages)
}

func carProfessionalSummaryPage(p Property, car CARResult, kml KMLResult, cmp GeometryComparison, page int) string {
	var c pdfCanvas
	proHeader(&c, "DEMONSTRATIVO TECNICO DO CAR", "Cadastro, geometria e conferência do imóvel rural", page)
	y := 724.0

	c.b.WriteString("0.06 0.31 0.20 rg\n")
	c.text(40, y, 8, true, "IMÓVEL ANALISADO")
	c.b.WriteString("0.10 0.18 0.14 rg\n")
	c.text(40, y-23, 16, true, firstNonEmptyText(p.Name, car.PropertyName, "Imóvel rural"))
	c.b.WriteString("0.38 0.45 0.41 rg\n")
	c.text(40, y-40, 7.3, false, firstNonEmptyText(p.ClientName, "Cliente não informado")+" - "+firstNonEmptyText(car.Municipality, p.Municipality)+" / "+firstNonEmptyText(car.UF, p.UF))
	y -= 70

	status := reportLookupLabel(car)
	area := car.AreaHa
	if area == 0 {
		area = car.GeometryAreaHa
	}
	kmlStatus := "Não carregado"
	if kml.AreaHa > 0 {
		kmlStatus = fmtBR(kml.AreaHa, 4) + " ha"
	} else if car.AutoKMLPath != "" {
		kmlStatus = "SICAR gerado"
	}
	proMetric(&c, 40, y-72, 122, 64, "SITUAÇÃO SICAR", status, firstNonEmptyText(car.Condition, car.LookupDetail, "Consulta pública"), reportLookupTone(car))
	proMetric(&c, 171, y-72, 122, 64, "ÁREA", fmtBR(area, 4)+" ha", fmtBR(car.FiscalModules, 2)+" módulo(s) fiscal(is)", "blue")
	proMetric(&c, 302, y-72, 122, 64, "GEOMETRIA", fmtBR(car.PerimeterM/1000, 3)+" km", "perímetro público calculado", "")
	proMetric(&c, 433, y-72, 122, 64, "KML", kmlStatus, "arquivo técnico para conferência", "")
	y -= 94

	proSection(&c, &y, "IDENTIFICAÇÃO E CADASTRO", "dados locais + fonte pública consultada")
	proKV(&c, &y, "Cliente", p.ClientName)
	proKV(&c, &y, "Nome do imóvel", firstNonEmptyText(p.Name, car.PropertyName))
	proKV(&c, &y, "CAR", car.CAR)
	proKV(&c, &y, "Município / UF", strings.Trim(strings.TrimSpace(firstNonEmptyText(car.Municipality, p.Municipality)+" / "+firstNonEmptyText(car.UF, p.UF)), " /"))
	proKV(&c, &y, "Matrícula / registro local", p.Registry)
	proKV(&c, &y, "Tipo do imóvel", car.PropertyType)
	proKV(&c, &y, "Inscrição no CAR", dateBR(car.DataCadastro))
	proKV(&c, &y, "Última atualização pública", dateBR(car.DataAtualizacao))

	if y > 230 {
		proSection(&c, &y, "RESULTADO DA CONSULTA", "não confundir indisponibilidade com ausência de ocorrência")
		proKV(&c, &y, "Situação", status)
		proKV(&c, &y, "Condição", car.Condition)
		proKV(&c, &y, "Área declarada SICAR", fmtBR(car.AreaHa, 4)+" ha")
		proKV(&c, &y, "Área geométrica calculada", fmtBR(car.GeometryAreaHa, 4)+" ha")
		proKV(&c, &y, "Centro aproximado", fmt.Sprintf("%.6f, %.6f", car.CenterLat, car.CenterLon))
	}

	detail := firstNonEmptyText(car.LookupDetail, "A ficha pública e a geometria foram organizadas pelo ViaVerdeCAR a partir das fontes disponíveis na consulta.")
	if car.LookupStatus == "partial" || car.LookupStatus == "cached" || car.LookupStatus == "unavailable" {
		proNotice(&c, &y, "ATENÇÃO À FONTE PÚBLICA", detail, "warn")
	} else {
		proNotice(&c, &y, "SÍNTESE", detail, "")
	}
	proFooter(&c, page)
	return c.b.String()
}

func carProfessionalMapPage(p Property, car CARResult, kml KMLResult, cmp GeometryComparison, page int) string {
	var c pdfCanvas
	proHeader(&c, "DEMONSTRATIVO TECNICO DO CAR", "Mapa vetorial e conferência geométrica", page)
	y := 720.0

	proSection(&c, &y, "MAPA DE CONFERÊNCIA", "CAR público e KML externo quando disponível")
	c.b.WriteString("0.97 0.98 0.97 rg\n")
	c.rect(40, 350, 515, 330, true)
	c.b.WriteString("0.80 0.86 0.82 RG 0.7 w\n")
	c.rect(40, 350, 515, 330, false)
	drawDossierGeometry(&c, car, kml, 53, 382, 489, 270)
	y = 320

	proSection(&c, &y, "MÉTRICAS GEOMÉTRICAS", "")
	proMetric(&c, 40, y-70, 122, 60, "ÁREA SICAR", fmtBR(car.GeometryAreaHa, 4)+" ha", "cálculo a partir da geometria", "blue")
	proMetric(&c, 171, y-70, 122, 60, "PERÍMETRO", fmtBR(car.PerimeterM/1000, 3)+" km", "perímetro geométrico", "")
	proMetric(&c, 302, y-70, 122, 60, "KML EXTERNO", func() string {
		if kml.AreaHa > 0 {
			return fmtBR(kml.AreaHa, 4) + " ha"
		}
		return "Não carregado"
	}(), "geometria independente", "")
	proMetric(&c, 433, y-70, 122, 60, "SOBREPOSIÇÃO", func() string {
		if cmp.OverlapMethod != "" {
			return fmtBR(minFloat(cmp.KMLInsideCARPct, cmp.CARInsideKMLPct), 1) + "%"
		}
		return "Não calculada"
	}(), "menor cobertura estimada", func() string {
		if cmp.Level == "error" {
			return "danger"
		}
		if cmp.Level == "warning" {
			return "warn"
		}
		return ""
	}())
	y -= 94

	if kml.AreaHa > 0 {
		proSection(&c, &y, "COMPARAÇÃO KML x CAR", "")
		proKV(&c, &y, "Síntese", cmp.Summary)
		proKV(&c, &y, "Diferença de área", fmtBR(cmp.AreaDifferenceHa, 4)+" ha ("+fmtBR(cmp.AreaDifferencePct, 2)+"%)")
		proKV(&c, &y, "Distância entre centros", fmtBR(cmp.CenterDistanceM, 0)+" m")
		if cmp.OverlapMethod != "" {
			proKV(&c, &y, "Interseção estimada", fmtBR(cmp.IntersectionAreaHa, 4)+" ha")
			proKV(&c, &y, "KML dentro do CAR", fmtBR(cmp.KMLInsideCARPct, 1)+"%")
			proKV(&c, &y, "CAR dentro do KML", fmtBR(cmp.CARInsideKMLPct, 1)+"%")
		}
	} else {
		proNotice(&c, &y, "KML EXTERNO NÃO INFORMADO", "O perímetro público do SICAR está disponível, mas não existe KML externo do cliente nesta análise. A ausência de KML independente impede a conferência CAR x levantamento externo.", "warn")
	}

	proNotice(&c, &y, "LIMITAÇÃO TÉCNICA", "O mapa vetorial é uma representação auxiliar para conferência. Não substitui planta, memorial descritivo, levantamento topográfico, certificação fundiária ou documento oficial emitido pelo órgão competente.", "")
	proFooter(&c, page)
	return c.b.String()
}

func buildEnvironmentalEvidencePDF(p Property, car CARResult, intel EnvironmentalIntelligenceResult) []byte {
	car.Environment = intel.Environment
	return assembleMultiPagePDF([]string{
		environmentalEvidenceSummaryPage(p, car, intel, 1),
		environmentalEvidenceMapAndAlertsPage(p, car, intel, 2),
		environmentalEvidenceSourcesPage(p, car, intel, 3),
	})
}

func environmentalEvidenceSummaryPage(p Property, car CARResult, intel EnvironmentalIntelligenceResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "CADERNO DE EVIDENCIAS AMBIENTAIS", "Rastreabilidade das bases e resultados consultados", page)
	y := 720.0

	proSection(&c, &y, "IDENTIFICAÇÃO", "")
	proKV(&c, &y, "Imóvel", firstNonEmptyText(p.Name, car.PropertyName))
	proKV(&c, &y, "Cliente", p.ClientName)
	proKV(&c, &y, "CAR", car.CAR)
	proKV(&c, &y, "Município / UF", firstNonEmptyText(car.Municipality, p.Municipality)+" / "+firstNonEmptyText(car.UF, p.UF))
	proKV(&c, &y, "Área", fmtBR(firstPositive(car.AreaHa, car.GeometryAreaHa), 4)+" ha")
	y -= 4

	proSection(&c, &y, "RESUMO DAS EVIDÊNCIAS", "")
	s := intel.Summary
	proMetric(&c, 40, y-72, 122, 64, "MAPBIOMAS", fmt.Sprintf("%d alerta(s)", s.Alerts), fmtBR(s.AlertAreaInCARHa, 2)+" ha estimados no CAR", func() string {
		if s.Alerts > 0 {
			return "warn"
		}
		return ""
	}())
	proMetric(&c, 171, y-72, 122, 64, "ALTA ATENÇÃO", fmt.Sprintf("%d", s.HighAttentionAlerts), "alertas com prioridade elevada", func() string {
		if s.HighAttentionAlerts > 0 {
			return "danger"
		}
		return ""
	}())
	proMetric(&c, 302, y-72, 122, 64, "IBAMA", fmt.Sprintf("%d", intel.Environment.IBAMAEmbargoCount), sourceState(intel.Environment.IBAMAChecked, intel.Environment.IBAMAEmbargoCount), func() string {
		if intel.Environment.IBAMAEmbargoCount > 0 {
			return "danger"
		}
		return ""
	}())
	proMetric(&c, 433, y-72, 122, 64, "FOCOS DE CALOR", fmt.Sprintf("%d", intel.Profile.Fire.FeatureCount), fireReportText(intel.Profile.Fire), func() string {
		if intel.Profile.Fire.FeatureCount > 0 {
			return "warn"
		}
		return ""
	}())
	y -= 95

	proSection(&c, &y, "STATUS DAS BASES", "resultado da fonte, não inferência")
	mbState, mbDetail := mapBiomasReportState(intel.MapBiomas)
	proKV(&c, &y, "MapBiomas Alerta", mbState+" - "+mbDetail)
	proKV(&c, &y, "IBAMA / PAMGIA", sourceState(intel.Environment.IBAMAChecked, intel.Environment.IBAMAEmbargoCount))
	proKV(&c, &y, "FUNAI", sourceState(intel.Environment.FUNAIChecked, intel.Environment.IndigenousCount))
	proKV(&c, &y, "ICMBio", sourceState(intel.Environment.ICMBioChecked, intel.Environment.FederalUCCount))
	proKV(&c, &y, "MMA / MCR-PRODES", mcrSourceSummary(intel.Environment))
	proKV(&c, &y, "Temas SICAR", themeSourceSummary(intel.Themes))
	proKV(&c, &y, "INPE / Programa Queimadas", fireReportText(intel.Profile.Fire))

	if intel.Profile.BiomeAvailable || intel.Profile.LandCoverAvailable {
		proSection(&c, &y, "PERFIL TERRITORIAL", "")
		if intel.Profile.BiomeAvailable {
			proKV(&c, &y, "Bioma", intel.Profile.Biome)
		}
		if intel.Profile.LandCoverAvailable {
			proKV(&c, &y, "Cobertura dominante", fmt.Sprintf("%s - ESA WorldCover %d", intel.Profile.DominantLandCover, intel.Profile.LandCoverYear))
		}
	}

	conclusion := environmentalConclusionText(intel, intel.Alerts)
	proNotice(&c, &y, "LEITURA TÉCNICA AUTOMÁTICA", conclusion, func() string {
		if s.HighAttentionAlerts > 0 || intel.Environment.IBAMAEmbargoCount > 0 {
			return "warn"
		}
		return ""
	}())
	proFooter(&c, page)
	return c.b.String()
}

func environmentalEvidenceMapAndAlertsPage(p Property, car CARResult, intel EnvironmentalIntelligenceResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "CADERNO DE EVIDENCIAS AMBIENTAIS", "Mapa e registros detalhados", page)
	y := 720.0
	proSection(&c, &y, "MAPA DE EVIDÊNCIAS", "CAR, alertas MapBiomas e embargos com geometria")
	c.b.WriteString("0.97 0.98 0.97 rg\n")
	c.rect(40, 400, 515, 275, true)
	c.b.WriteString("0.80 0.86 0.82 RG 0.7 w\n")
	c.rect(40, 400, 515, 275, false)
	drawEnvironmentalEvidenceMap(&c, car, intel.Alerts, 52, 430, 490, 225)
	y = 370

	proSection(&c, &y, "REGISTROS MAPBIOMAS", "")
	if len(intel.Alerts) == 0 {
		proParagraph(&c, &y, func() string {
			if intel.MapBiomas.Available {
				return "A consulta foi concluída sem alerta MapBiomas vinculado ao CAR nesta execução."
			}
			return "A base MapBiomas Alerta não forneceu resultado utilizável nesta execução. Isso não deve ser interpretado como ausência de alertas."
		}())
	} else {
		for i, a := range intel.Alerts {
			if i >= 6 || y < 115 {
				break
			}
			title := firstNonEmptyText(a.AlertCode, fmt.Sprintf("Alerta %d", i+1))
			detail := fmt.Sprintf("%s ha no alerta; %s ha estimados dentro do CAR; detecção %s; prioridade %s",
				fmtBR(a.AreaHa, 2), fmtBR(a.AlertAreaInCAR, 2), firstNonEmptyText(dateBR(a.DetectedAt), "não informada"), firstNonEmptyText(a.AttentionLevel, "revisão"))
			proKV(&c, &y, title, detail)
		}
	}

	if y > 105 {
		proSection(&c, &y, "CRUZAMENTOS SENSÍVEIS", "")
		proKV(&c, &y, "Alertas sobre APP", fmt.Sprintf("%d alerta(s); %s ha estimados", intel.Summary.AlertsOverAPP, fmtBR(intel.Summary.APPOverlapHa, 2)))
		proKV(&c, &y, "Alertas sobre Reserva Legal", fmt.Sprintf("%d alerta(s); %s ha estimados", intel.Summary.AlertsOverRL, fmtBR(intel.Summary.RLOverlapHa, 2)))
		proKV(&c, &y, "Alertas x embargo IBAMA", fmt.Sprintf("%d", intel.Summary.AlertsOverIBAMA))
		proKV(&c, &y, "Alertas x Terra Indígena", fmt.Sprintf("%d", intel.Summary.AlertsOverIndigenousLand))
		proKV(&c, &y, "Alertas x UC federal", fmt.Sprintf("%d", intel.Summary.AlertsOverFederalUC))
	}
	proFooter(&c, page)
	return c.b.String()
}

func environmentalEvidenceSourcesPage(p Property, car CARResult, intel EnvironmentalIntelligenceResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "CADERNO DE EVIDENCIAS AMBIENTAIS", "Fontes, rastreabilidade e limitações", page)
	y := 720.0
	proSection(&c, &y, "FONTES CONSULTADAS", "")
	rows := []struct{ label, value string }{
		{"SICAR", carWFSURL},
		{"MapBiomas Alerta", "https://plataforma.alerta.mapbiomas.org/api/v2/graphql"},
		{"IBAMA / PAMGIA", "https://pamgia.ibama.gov.br/"},
		{"FUNAI", funaiGeoURL},
		{"ICMBio", icmbioGeoURL},
		{"MMA / MCR", environmentMCRURL},
		{"INPE / Programa Queimadas", inpeFireWFSURL},
		{"ESA WorldCover", worldCoverSourceURL},
	}
	for _, row := range rows {
		proKV(&c, &y, row.label, row.value)
	}

	proSection(&c, &y, "AVISOS DA EXECUÇÃO", "")
	if len(intel.Warnings) == 0 {
		proParagraph(&c, &y, "Nenhum aviso técnico adicional foi registrado nesta execução.")
	} else {
		for i, w := range intel.Warnings {
			if i >= 10 || y < 190 {
				break
			}
			proParagraph(&c, &y, "- "+w)
		}
	}

	proNotice(&c, &y, "LIMITAÇÕES", "Este caderno registra evidências públicas e resultados automáticos para rastreabilidade. Não constitui certidão de regularidade ambiental, auto de infração, parecer jurídico, licenciamento ou decisão de crédito. Ocorrências devem ser confirmadas na fonte oficial; fonte indisponível deve permanecer identificada como indisponível.", "")
	proFooter(&c, page)
	return c.b.String()
}

func buildPropertyTechnicalDossierPDF(p Property, r CARAutomationResult) []byte {
	pages := []string{}
	page := 1
	pages = append(pages, dossierCoverPage(p, r, page)); page++
	pages = append(pages, dossierCARPage(p, r, page)); page++
	pages = append(pages, dossierEnvironmentalPage(p, r, page)); page++
	pages = append(pages, dossierLandPage(p, r, page)); page++
	pages = append(pages, dossierCreditPage(p, r, page)); page++
	pages = append(pages, dossierBCBPage(p, r, page)); page++
	pages = append(pages, dossierSourcesPage(p, r, page))
	return assembleMultiPagePDF(pages)
}

func dossierCoverPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIE TECNICO DO IMOVEL", "CAR, ambiente, fundiário e crédito rural em um único documento", page)
	y := 706.0
	c.b.WriteString("0.94 0.98 0.95 rg\n")
	c.rect(40, 560, 515, 126, true)
	c.b.WriteString("0.06 0.31 0.20 rg\n")
	c.text(58, 650, 8, true, "IMÓVEL")
	c.text(58, 620, 19, true, firstNonEmptyText(p.Name, r.CAR.PropertyName, "Imóvel rural"))
	c.b.WriteString("0.32 0.42 0.36 rg\n")
	c.text(58, 596, 8, false, firstNonEmptyText(p.ClientName, "Cliente não informado"))
	c.text(58, 579, 7.5, false, firstNonEmptyText(r.CAR.Municipality, p.Municipality)+" / "+firstNonEmptyText(r.CAR.UF, p.UF))
	y = 525

	proMetric(&c, 40, y-72, 122, 64, "CAR", reportLookupLabel(r.CAR), fmtBR(firstPositive(r.CAR.AreaHa, r.CAR.GeometryAreaHa), 2)+" ha", reportLookupTone(r.CAR))
	proMetric(&c, 171, y-72, 122, 64, "AMBIENTAL", fmt.Sprintf("%d alerta(s)", r.Environmental.Summary.Alerts), fmt.Sprintf("%d alta atenção", r.Environmental.Summary.HighAttentionAlerts), func() string {
		if r.Environmental.Summary.HighAttentionAlerts > 0 {
			return "warn"
		}
		return ""
	}())
	proMetric(&c, 302, y-72, 122, 64, "FUNDIÁRIO", fmt.Sprintf("%d parcela(s)", r.XRay.SIGEF.ParcelCount), func() string {
		if r.XRay.SIGEF.BestCARCoveragePct > 0 {
			return fmtBR(r.XRay.SIGEF.BestCARCoveragePct, 1) + "% melhor cobertura"
		}
		return "SIGEF / INCRA"
	}(), "")
	proMetric(&c, 433, y-72, 122, 64, "CRÉDITO RURAL", fmt.Sprintf("%d operação(ões)", r.XRay.SICOR.OperationCount), func() string {
		if r.XRay.SICOR.TotalCreditValue > 0 {
			return moneyBR(r.XRay.SICOR.TotalCreditValue)
		}
		return "SICOR / BCB"
	}(), "purple")
	y -= 102

	proSection(&c, &y, "SÍNTESE EXECUTIVA", "")
	proParagraph(&c, &y, firstNonEmptyText(r.ExecutiveSummary, "O ViaVerdeCAR consolidou as fontes públicas disponíveis para o imóvel."))
	if len(r.Warnings) > 0 {
		proNotice(&c, &y, "PONTOS PARA CONFERÊNCIA", fmt.Sprintf("%d aviso(s) técnico(s) foram registrados. As páginas seguintes detalham bases indisponíveis, consultas parciais e ocorrências localizadas.", len(r.Warnings)), "warn")
	} else {
		proNotice(&c, &y, "STATUS DA EXECUÇÃO", "A execução não registrou avisos técnicos adicionais. Isso não substitui a conferência documental e profissional do imóvel.", "")
	}

	proSection(&c, &y, "IDENTIFICAÇÃO", "")
	proKV(&c, &y, "CAR", r.CAR.CAR)
	proKV(&c, &y, "Matrícula / registro local", p.Registry)
	proKV(&c, &y, "Área cadastrada", fmtBR(firstPositive(r.CAR.AreaHa, p.DeclaredAreaHa), 4)+" ha")
	proKV(&c, &y, "Gerado em", time.Now().Format("02/01/2006 15:04"))
	proFooter(&c, page)
	return c.b.String()
}

func dossierCARPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIE TECNICO DO IMOVEL", "CAR e representação geométrica", page)
	y := 720.0
	proSection(&c, &y, "CADASTRO AMBIENTAL RURAL", "")
	proKV(&c, &y, "CAR", r.CAR.CAR)
	proKV(&c, &y, "Situação / condição", strings.Trim(strings.TrimSpace(reportLookupLabel(r.CAR)+" / "+r.CAR.Condition), " /"))
	proKV(&c, &y, "Município / UF", r.CAR.Municipality+" / "+r.CAR.UF)
	proKV(&c, &y, "Área declarada", fmtBR(r.CAR.AreaHa, 4)+" ha")
	proKV(&c, &y, "Área geométrica", fmtBR(r.CAR.GeometryAreaHa, 4)+" ha")
	proKV(&c, &y, "Perímetro", fmtBR(r.CAR.PerimeterM/1000, 3)+" km")
	proKV(&c, &y, "Módulos fiscais", fmtBR(r.CAR.FiscalModules, 2))
	proKV(&c, &y, "Inscrição / atualização", dateBR(r.CAR.DataCadastro)+" / "+dateBR(r.CAR.DataAtualizacao))

	proSection(&c, &y, "MAPA VETORIAL DO IMÓVEL", "temas e ocorrências disponíveis")
	c.b.WriteString("0.97 0.98 0.97 rg\n")
	c.rect(40, 110, 515, y-125, true)
	c.b.WriteString("0.80 0.86 0.82 RG 0.7 w\n")
	c.rect(40, 110, 515, y-125, false)
	drawDossierGeometry(&c, r.CAR, KMLResult{}, 54, 137, 487, y-175)
	proFooter(&c, page)
	return c.b.String()
}

func dossierEnvironmentalPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIE TECNICO DO IMOVEL", "Raio X ambiental", page)
	y := 720.0
	proSection(&c, &y, "RESUMO AMBIENTAL", "")
	s := r.Environmental.Summary
	proMetric(&c, 40, y-72, 122, 64, "MAPBIOMAS", fmt.Sprintf("%d alerta(s)", s.Alerts), fmtBR(s.AlertAreaInCARHa, 2)+" ha no CAR", func() string { if s.Alerts>0{return "warn"};return "" }())
	proMetric(&c, 171, y-72, 122, 64, "IBAMA", fmt.Sprintf("%d", r.Environmental.Environment.IBAMAEmbargoCount), sourceState(r.Environmental.Environment.IBAMAChecked, r.Environmental.Environment.IBAMAEmbargoCount), func() string { if r.Environmental.Environment.IBAMAEmbargoCount>0{return "danger"};return "" }())
	proMetric(&c, 302, y-72, 122, 64, "FUNAI", fmt.Sprintf("%d", r.Environmental.Environment.IndigenousCount), sourceState(r.Environmental.Environment.FUNAIChecked, r.Environmental.Environment.IndigenousCount), "")
	proMetric(&c, 433, y-72, 122, 64, "ICMBio", fmt.Sprintf("%d", r.Environmental.Environment.FederalUCCount), sourceState(r.Environmental.Environment.ICMBioChecked, r.Environmental.Environment.FederalUCCount), "")
	y -= 95

	proSection(&c, &y, "FONTES E RESULTADOS", "")
	mbState, mbDetail := mapBiomasReportState(r.Environmental.MapBiomas)
	proKV(&c, &y, "MapBiomas Alerta", mbState+" - "+mbDetail)
	proKV(&c, &y, "INPE / Focos de calor", fireReportText(r.Environmental.Profile.Fire))
	proKV(&c, &y, "MMA / MCR-PRODES", mcrSourceSummary(r.Environmental.Environment))
	proKV(&c, &y, "Temas SICAR", themeSourceSummary(r.Environmental.Themes))
	if r.Environmental.Profile.BiomeAvailable {
		proKV(&c, &y, "Bioma", r.Environmental.Profile.Biome)
	}
	if r.Environmental.Profile.LandCoverAvailable {
		proKV(&c, &y, "Cobertura dominante", fmt.Sprintf("%s - ESA WorldCover %d", r.Environmental.Profile.DominantLandCover, r.Environmental.Profile.LandCoverYear))
	}

	proSection(&c, &y, "MAPA DE EVIDÊNCIAS", "")
	c.b.WriteString("0.97 0.98 0.97 rg\n")
	h := y - 120
	if h < 120 { h = 120 }
	c.rect(40, 88, 515, h, true)
	c.b.WriteString("0.80 0.86 0.82 RG 0.7 w\n")
	c.rect(40, 88, 515, h, false)
	drawEnvironmentalEvidenceMap(&c, r.CAR, r.Environmental.Alerts, 53, 114, 489, h-43)
	proFooter(&c, page)
	return c.b.String()
}

func dossierLandPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIE TECNICO DO IMOVEL", "Raio X fundiário - SIGEF / INCRA", page)
	y := 720.0
	s := r.XRay.SIGEF
	proMetric(&c, 40, y-72, 156, 64, "PARCELAS SIGEF", fmt.Sprintf("%d", s.ParcelCount), "parcelas públicas retornadas", "")
	proMetric(&c, 219, y-72, 156, 64, "REGISTROS", fmt.Sprintf("%d", s.RegistryCount), "matrículas / registros retornados", "")
	proMetric(&c, 398, y-72, 157, 64, "MELHOR COBERTURA", func() string {if s.BestCARCoveragePct>0{return fmtBR(s.BestCARCoveragePct,1)+"%"};return "—"}(), "CAR coberto pela melhor parcela", "")
	y -= 95

	proSection(&c, &y, "RESULTADO DA FONTE", "")
	proKV(&c, &y, "Disponibilidade", func() string {if s.Available{return "Consulta realizada"};return "Base indisponível / sem resultado utilizável"}())
	proKV(&c, &y, "Mensagem", s.Message)
	proKV(&c, &y, "Fonte", s.SourceURL)

	proSection(&c, &y, "PARCELAS MAIS RELEVANTES", "")
	if len(s.Parcels) == 0 {
		proParagraph(&c, &y, "Nenhuma parcela SIGEF foi relacionada ao perímetro nesta execução.")
	} else {
		for i, parcel := range s.Parcels {
			if i >= 7 || y < 150 { break }
			label := firstNonEmptyText(parcel.AreaName, parcel.ParcelCode, fmt.Sprintf("Parcela %d", i+1))
			detail := fmt.Sprintf("Área %s ha; cobertura CAR %s%%; matrícula %s; situação %s",
				fmtBR(parcel.AreaHa, 2), fmtBR(parcel.CARInsideParcelPct, 1), firstNonEmptyText(parcel.Registry, "não informada"), firstNonEmptyText(parcel.Status, parcel.PropertySituation))
			proKV(&c, &y, label, detail)
		}
	}

	proNotice(&c, &y, "LIMITAÇÃO", "O cruzamento SIGEF é uma conferência espacial auxiliar. Não comprova domínio, cadeia dominial, validade registral, certificação atual ou inexistência de outros direitos sobre a área.", "")
	proFooter(&c, page)
	return c.b.String()
}

func dossierCreditPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIE TECNICO DO IMOVEL", "Operações públicas de crédito rural - SICOR", page)
	y := 720.0
	s := r.XRay.SICOR
	proMetric(&c, 40, y-72, 156, 64, "OPERAÇÕES", fmt.Sprintf("%d", s.OperationCount), fmt.Sprintf("%d gleba(s)", s.GlebaCount), "purple")
	proMetric(&c, 219, y-72, 156, 64, "VALOR TOTAL", moneyBR(s.TotalCreditValue), "operações públicas retornadas", "purple")
	proMetric(&c, 398, y-72, 157, 64, "ÁREA FINANCIADA", fmtBR(s.TotalFinancedAreaHa, 2)+" ha", "soma informada nas operações", "")
	y -= 95

	proSection(&c, &y, "OPERAÇÕES ENCONTRADAS", "")
	if len(s.Operations) == 0 {
		proParagraph(&c, &y, "Nenhuma operação pública SICOR vinculada ao contexto espacial do imóvel foi localizada nesta execução.")
	} else {
		for i, op := range s.Operations {
			if i >= 9 || y < 125 { break }
			label := firstNonEmptyText(op.Purpose, op.Activity, op.Product, fmt.Sprintf("Operação %d", i+1))
			detail := strings.Join(v2NonEmpty([]string{
				firstNonEmptyText(op.InstitutionName, op.InstitutionCode),
				firstNonEmptyText(op.ProgramName, op.SubprogramName),
				func() string {if op.CreditValue>0{return moneyBR(op.CreditValue)};return ""}(),
				func() string {if op.Year>0{return fmt.Sprintf("%d", op.Year)};return ""}(),
			}), " - ")
			proKV(&c, &y, label, detail)
		}
	}
	proNotice(&c, &y, "INTERPRETAÇÃO", "Operação pública SICOR localizada no contexto do imóvel não equivale a dívida atual, saldo devedor, limite disponível, aprovação futura ou obrigação atribuída automaticamente ao proprietário cadastrado localmente.", "")
	proFooter(&c, page)
	return c.b.String()
}

func dossierBCBPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIE TECNICO DO IMOVEL", "Banco Central - contexto público e agregado", page)
	y := 720.0
	b := r.BCB
	proSection(&c, &y, "CONTEXTO DE MERCADO", "dados agregados, não análise individual")
	proKV(&c, &y, "Município / UF", b.Municipality+" / "+b.UF)
	proKV(&c, &y, "MDCR / SICOR", b.Market.Message)
	proKV(&c, &y, "Séries SGS", b.Series.Message)
	proKV(&c, &y, "Entidades supervisionadas", b.Institutions.Message)
	proKV(&c, &y, "IFData", b.IFData.Message)
	proKV(&c, &y, "Taxas por instituição", b.InstitutionRates.Message)

	proSection(&c, &y, "TAXAS RURAIS AGREGADAS", "")
	rateCount := 0
	for _, m := range b.Series.Metrics {
		if m.Measure != "Taxa de juros" || m.Status != "available" {
			continue
		}
		if rateCount >= 6 || y < 250 { break }
		proKV(&c, &y, m.Label, fmt.Sprintf("%s %s - ref. %s - SGS %d", fmtBR(m.LatestValue, 2), m.Unit, m.LatestDate, m.SGSCode))
		rateCount++
	}
	if rateCount == 0 {
		proParagraph(&c, &y, "As séries de taxas rurais não retornaram valores utilizáveis nesta execução.")
	}

	proSection(&c, &y, "MERCADO MUNICIPAL", "")
	for i, item := range b.Market.MunicipalProducts {
		if i >= 6 || y < 130 { break }
		proKV(&c, &y, firstNonEmptyText(item.Label, item.Kind), fmt.Sprintf("%s - %s contrato(s) - %s", item.Year, fmtBR(item.Contracts, 0), moneyBR(item.Value)))
	}
	proNotice(&c, &y, "SALVAGUARDA", "Os dados do Banco Central são públicos e agregados. Não representam taxa garantida, aprovação, limite, dívida, inadimplência ou risco individual do produtor.", "")
	proFooter(&c, page)
	return c.b.String()
}

func dossierSourcesPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIE TECNICO DO IMOVEL", "Fontes consultadas, avisos e limitações", page)
	y := 720.0
	proSection(&c, &y, "FONTES DA EXECUÇÃO", "")
	for i, s := range r.Sources {
		if i >= 18 || y < 300 { break }
		value := reportAutomationStatus(s.Status)
		if s.Count > 0 {
			value += fmt.Sprintf(" - %d registro(s)", s.Count)
		}
		if strings.TrimSpace(s.Detail) != "" {
			value += " - " + s.Detail
		}
		proKV(&c, &y, s.Label, value)
	}

	proSection(&c, &y, "AVISOS TÉCNICOS", "")
	if len(r.Warnings) == 0 {
		proParagraph(&c, &y, "Nenhum aviso técnico adicional foi registrado nesta execução.")
	} else {
		for i, w := range r.Warnings {
			if i >= 12 || y < 150 { break }
			proParagraph(&c, &y, "- "+w)
		}
	}
	proNotice(&c, &y, "USO DO DOCUMENTO", "Este dossiê consolida consultas públicas e dados locais para apoio ao trabalho técnico. Ele não substitui documentos oficiais do SICAR, registro de imóveis, certificação SIGEF/INCRA, licenciamento, auto de infração, consulta bancária privada ou decisão de crédito. Toda ocorrência relevante deve ser confirmada na fonte oficial e revisada pelo profissional responsável.", "")
	proFooter(&c, page)
	return c.b.String()
}

func fireReportText(f EnvironmentalFireProfile) string {
	status, value, detail := environmentalFireReportStatus(f)
	parts := v2NonEmpty([]string{status, value, detail})
	return strings.Join(parts, " - ")
}

func reportAutomationStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "ok":
		return "Sem ocorrência"
	case "hit":
		return "Ocorrência encontrada"
	case "available":
		return "Disponível"
	case "empty":
		return "Sem dados no recorte"
	case "on_demand":
		return "Sob demanda"
	case "partial":
		return "Consulta parcial"
	case "cached":
		return "Cache local"
	case "not_found":
		return "Não localizado"
	case "not_saved":
		return "Não salvo"
	case "not_configured":
		return "Não configurado"
	case "unavailable":
		return "Base indisponível"
	default:
		return "Não consultado"
	}
}

func moneyBR(v float64) string {
	if v == 0 {
		return "R$ 0,00"
	}
	raw := fmt.Sprintf("%.2f", v)
	parts := strings.Split(raw, ".")
	intPart := parts[0]
	for i := len(intPart) - 3; i > 0; i -= 3 {
		intPart = intPart[:i] + "." + intPart[i:]
	}
	return "R$ " + intPart + "," + parts[1]
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
