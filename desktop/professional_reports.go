package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Os quatro métodos abaixo geram os PDFs diretamente do resultado que já está
// na tela. Isso permite relatório de consulta avulsa sem criar cliente, imóvel,
// histórico ou qualquer vínculo implícito no banco local.
func (a *App) ExportCARAutomationPDF(result CARAutomationResult) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	if strings.TrimSpace(result.CAR.CAR) == "" {
		return "", errors.New("execute a análise de um CAR antes de gerar o demonstrativo")
	}
	p := a.propertyForAutomationReport(result)
	kml, cmp := a.savedKMLForAutomationReport(result, p)
	pdf := buildCARProfessionalPDF(p, result.CAR, kml, cmp)
	name := "Demonstrativo_CAR_" + safeCARFilename(result.CAR.CAR) + ".pdf"
	return a.saveProfessionalPDF("Salvar demonstrativo técnico do CAR", name, pdf)
}

func (a *App) ExportEnvironmentalAutomationPDF(result CARAutomationResult) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	if strings.TrimSpace(result.CAR.CAR) == "" {
		return "", errors.New("execute a análise de um CAR antes de gerar o laudo ambiental")
	}
	p := a.propertyForAutomationReport(result)
	pdf := buildEnvironmentalTechnicalPDF(p, result.CAR, result.Environmental, "")
	name := "Laudo_Tecnico_Ambiental_" + safeCARFilename(result.CAR.CAR) + ".pdf"
	return a.saveProfessionalPDF("Salvar laudo técnico ambiental", name, pdf)
}

func (a *App) ExportEnvironmentalAutomationEvidencePDF(result CARAutomationResult) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	if strings.TrimSpace(result.CAR.CAR) == "" {
		return "", errors.New("execute a análise de um CAR antes de gerar as evidências")
	}
	p := a.propertyForAutomationReport(result)
	pdf := buildEnvironmentalEvidencePDF(p, result.CAR, result.Environmental)
	name := "Caderno_Evidencias_Ambientais_" + safeCARFilename(result.CAR.CAR) + ".pdf"
	return a.saveProfessionalPDF("Salvar caderno de evidências ambientais", name, pdf)
}

func (a *App) ExportPropertyAutomationDossierPDF(result CARAutomationResult) (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicativo ainda não inicializado")
	}
	if strings.TrimSpace(result.CAR.CAR) == "" {
		return "", errors.New("execute a análise de um CAR antes de gerar o dossiê")
	}
	p := a.propertyForAutomationReport(result)
	kml, cmp := a.savedKMLForAutomationReport(result, p)
	docs := dossierDocumentCenterForResult(result)
	if result.PropertyID > 0 {
		if savedDocs, docsErr := a.GetPropertyDocumentCenter(result.PropertyID); docsErr == nil {
			docs = savedDocs
		}
	}
	pdf := buildPropertyTechnicalDossierPDF(p, result, kml, cmp, docs)
	name := "Dossie_Tecnico_" + safeFilePart(firstNonEmptyText(p.Name, result.CAR.PropertyName, "Imovel")) + "_" + safeCARFilename(result.CAR.CAR) + ".pdf"
	return a.saveProfessionalPDF("Salvar dossiê técnico do imóvel", name, pdf)
}

func (a *App) propertyForAutomationReport(result CARAutomationResult) Property {
	if result.PropertyID > 0 {
		if p, err := a.GetProperty(result.PropertyID); err == nil {
			return p
		}
	}
	area := firstPositive(result.CAR.AreaHa, result.CAR.GeometryAreaHa)
	return Property{
		Name:           firstNonEmptyText(result.CAR.PropertyName, "Imóvel rural"),
		Municipality:   result.CAR.Municipality,
		UF:             result.CAR.UF,
		CARNumber:      result.CAR.CAR,
		DeclaredAreaHa: area,
	}
}

func (a *App) savedKMLForAutomationReport(result CARAutomationResult, p Property) (KMLResult, GeometryComparison) {
	var kml KMLResult
	var cmp GeometryComparison
	if result.PropertyID <= 0 || strings.TrimSpace(p.KMLPath) == "" {
		return kml, cmp
	}
	if saved, err := a.LoadPropertyKML(result.PropertyID); err == nil {
		kml = saved
		cmp = a.CompareKMLWithCAR(kml, result.CAR)
	}
	return kml, cmp
}

func (a *App) saveProfessionalPDF(title, defaultName string, pdf []byte) (string, error) {
	if len(pdf) == 0 {
		return "", errors.New("o PDF não pôde ser montado")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultName,
		Filters:         []runtime.FileFilter{{DisplayName: "PDF", Pattern: "*.pdf"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("exportação cancelada")
	}
	if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
		path += ".pdf"
	}
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

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
	var kml KMLResult
	var comparison GeometryComparison
	if strings.TrimSpace(p.KMLPath) != "" {
		if savedKML, loadErr := a.LoadPropertyKML(propertyID); loadErr == nil {
			kml = savedKML
			comparison = a.CompareKMLWithCAR(kml, result.CAR)
		}
	}
	docs := dossierDocumentCenterForResult(result)
	if savedDocs, docsErr := a.GetPropertyDocumentCenter(propertyID); docsErr == nil {
		docs = savedDocs
	}
	pdf := buildPropertyTechnicalDossierPDF(p, result, kml, comparison, docs)
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
	c.text(38, 790, 8.5, true, "CONSULTORIA AGRÍCOLA")
	c.text(210, 810, 11.5, true, title)
	c.text(210, 790, 7.2, false, subtitle)
	c.text(505, 775, 6.2, false, fmt.Sprintf("Página %d", page))
}

func proFooter(c *pdfCanvas, page int) {
	c.b.WriteString("0.82 0.87 0.84 RG 0.5 w\n")
	c.line(38, 48, 557, 48)
	c.b.WriteString("0.34 0.42 0.38 rg\n")
	c.text(38, 32, 5.8, false, "Via Verde Consultoria Agrícola • ViaVerdeCAR v"+AppVersion+" • documento técnico auxiliar • "+time.Now().Format("02/01/2006 15:04"))
	c.text(522, 32, 5.8, false, fmt.Sprintf("Página %d", page))
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
	proHeader(&c, "DEMONSTRATIVO TÉCNICO DO CAR", "Cadastro, geometria e conferência do imóvel rural", page)
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
	proHeader(&c, "DEMONSTRATIVO TÉCNICO DO CAR", "Mapa vetorial e conferência geométrica", page)
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
	proHeader(&c, "CADERNO DE EVIDÊNCIAS AMBIENTAIS", "Rastreabilidade das bases e resultados consultados", page)
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
	proMetric(&c, 302, y-72, 122, 64, "IBAMA", reportCheckedCount(intel.Environment.IBAMAChecked, intel.Environment.IBAMAEmbargoCount), sourceState(intel.Environment.IBAMAChecked, intel.Environment.IBAMAEmbargoCount), func() string {
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
	proHeader(&c, "CADERNO DE EVIDÊNCIAS AMBIENTAIS", "Mapa e registros detalhados", page)
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
	proHeader(&c, "CADERNO DE EVIDÊNCIAS AMBIENTAIS", "Fontes, rastreabilidade e limitações", page)
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

	proSection(&c, &y, "PONTOS DE ATENÇÃO DA EXECUÇÃO", "")
	reportWarnings := professionalReportWarnings(intel.Warnings)
	if len(reportWarnings) == 0 {
		proParagraph(&c, &y, "Nenhum ponto de atenção adicional foi registrado nesta execução.")
	} else {
		for i, w := range reportWarnings {
			if i >= 8 || y < 190 {
				break
			}
			proParagraph(&c, &y, "• "+w)
		}
	}

	proNotice(&c, &y, "LIMITAÇÕES", "Este caderno organiza evidências públicas e o estado de cada consulta para fins de rastreabilidade. Não constitui certidão de regularidade ambiental, auto de infração, parecer jurídico, licenciamento ou decisão de crédito. Ocorrências relevantes devem ser confirmadas na fonte oficial; fontes indisponíveis permanecem expressamente identificadas como indisponíveis.", "")
	proFooter(&c, page)
	return c.b.String()
}

func buildPropertyTechnicalDossierPDF(p Property, r CARAutomationResult, kml KMLResult, cmp GeometryComparison, docs PropertyDocumentCenter) []byte {
	pages := []string{}
	page := 1
	pages = append(pages, dossierCoverPage(p, r, page)); page++
	pages = append(pages, dossierCARPage(p, r, kml, cmp, page)); page++
	pages = append(pages, dossierEnvironmentalPage(p, r, page)); page++
	pages = append(pages, dossierLandPage(p, r, page)); page++
	pages = append(pages, dossierCreditPage(p, r, page)); page++
	pages = append(pages, dossierBCBPage(p, r, page)); page++
	pages = append(pages, dossierSourcesPage(p, r, page)); page++
	pages = append(pages, dossierDocumentsPage(p, r, docs, page))
	return assembleMultiPagePDF(pages)
}

func dossierCoverPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIÊ TÉCNICO DO IMÓVEL", "CAR, ambiente, fundiário e crédito rural em um único documento", page)
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
	reportWarnings := professionalReportWarnings(r.Warnings)
	if len(reportWarnings) > 0 {
		proNotice(&c, &y, "PONTOS PARA CONFERÊNCIA", fmt.Sprintf("%d ponto(s) de atenção foram registrados. As páginas seguintes identificam bases indisponíveis, consultas parciais e ocorrências que exigem conferência.", len(reportWarnings)), "warn")
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

func dossierCARPage(p Property, r CARAutomationResult, kml KMLResult, cmp GeometryComparison, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIÊ TÉCNICO DO IMÓVEL", "CAR e representação geométrica", page)
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
	drawDossierGeometry(&c, r.CAR, kml, 54, 137, 487, y-175)
	if kml.AreaHa > 0 {
		c.b.WriteString("0.94 0.98 0.95 rg\n")
		c.rect(40, 62, 515, 38, true)
		c.b.WriteString("0.16 0.27 0.20 rg\n")
		c.text(50, 84, 6.7, true, "KML EXTERNO")
		c.text(132, 84, 6.7, false, fmtBR(kml.AreaHa, 4)+" ha; "+firstNonEmptyText(cmp.Summary, "comparação geométrica calculada"))
	}
	proFooter(&c, page)
	return c.b.String()
}

func dossierEnvironmentalPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIÊ TÉCNICO DO IMÓVEL", "Raio X ambiental", page)
	y := 720.0
	proSection(&c, &y, "RESUMO AMBIENTAL", "")
	s := r.Environmental.Summary
	proMetric(&c, 40, y-72, 122, 64, "MAPBIOMAS", fmt.Sprintf("%d alerta(s)", s.Alerts), fmtBR(s.AlertAreaInCARHa, 2)+" ha no CAR", func() string { if s.Alerts>0{return "warn"};return "" }())
	proMetric(&c, 171, y-72, 122, 64, "IBAMA", reportCheckedCount(r.Environmental.Environment.IBAMAChecked, r.Environmental.Environment.IBAMAEmbargoCount), sourceState(r.Environmental.Environment.IBAMAChecked, r.Environmental.Environment.IBAMAEmbargoCount), func() string { if r.Environmental.Environment.IBAMAEmbargoCount>0{return "danger"};return "" }())
	proMetric(&c, 302, y-72, 122, 64, "FUNAI", reportCheckedCount(r.Environmental.Environment.FUNAIChecked, r.Environmental.Environment.IndigenousCount), sourceState(r.Environmental.Environment.FUNAIChecked, r.Environmental.Environment.IndigenousCount), "")
	proMetric(&c, 433, y-72, 122, 64, "ICMBio", reportCheckedCount(r.Environmental.Environment.ICMBioChecked, r.Environmental.Environment.FederalUCCount), sourceState(r.Environmental.Environment.ICMBioChecked, r.Environmental.Environment.FederalUCCount), "")
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
	proHeader(&c, "DOSSIÊ TÉCNICO DO IMÓVEL", "Raio X fundiário - SIGEF / INCRA", page)
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
	proHeader(&c, "DOSSIÊ TÉCNICO DO IMÓVEL", "Operações públicas de crédito rural - SICOR", page)
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
	proHeader(&c, "DOSSIÊ TÉCNICO DO IMÓVEL", "Banco Central - contexto público e agregado", page)
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

	proSection(&c, &y, "INDICADORES DE CRÉDITO RURAL", "")
	macroCount := 0
	for _, m := range b.Series.Metrics {
		if m.Measure == "Taxa de juros" || m.Status != "available" {
			continue
		}
		if macroCount >= 4 || y < 235 { break }
		proKV(&c, &y, m.Label, fmt.Sprintf("%s %s • ref. %s • SGS %d", fmtBR(m.LatestValue, 2), m.Unit, m.LatestDate, m.SGSCode))
		macroCount++
	}
	if macroCount == 0 {
		proParagraph(&c, &y, "Os indicadores agregados de saldo, concessões e inadimplência não retornaram valores utilizáveis nesta execução.")
	}

	if y > 185 {
		proSection(&c, &y, "MERCADO MUNICIPAL", "")
		shown := 0
		for _, item := range b.Market.MunicipalProducts {
			if shown >= 4 || y < 120 { break }
			proKV(&c, &y, firstNonEmptyText(item.Label, item.Kind), fmt.Sprintf("%s • %s contrato(s) • %s", item.Year, fmtBR(item.Contracts, 0), moneyBR(item.Value)))
			shown++
		}
		if shown == 0 {
			proParagraph(&c, &y, "Nenhum produto municipal foi retornado no recorte disponível desta execução.")
		}
	}
	proNotice(&c, &y, "SALVAGUARDA", "Os dados do Banco Central são públicos e agregados. Não representam taxa garantida, aprovação, limite, dívida, inadimplência ou risco individual do produtor.", "")
	proFooter(&c, page)
	return c.b.String()
}

func dossierDocumentCenterForResult(r CARAutomationResult) PropertyDocumentCenter {
	out := PropertyDocumentCenter{PropertyID: r.PropertyID, UpdatedAt: time.Now().Format(time.RFC3339)}
	carStatus := "pending"
	carDetail := "CAR ainda não confirmado."
	if strings.TrimSpace(r.CAR.CAR) != "" {
		carStatus = "received"
		carDetail = r.CAR.CAR
	}
	kmlStatus := "pending"
	kmlDetail := "KML automático não confirmado."
	if strings.TrimSpace(r.CAR.AutoKMLPath) != "" {
		kmlStatus = "received"
		kmlDetail = "Perímetro SICAR gerado automaticamente."
	} else if r.CAR.HasGeometry {
		kmlStatus = "review"
		kmlDetail = "Geometria disponível; KML automático não persistido nesta consulta."
	}
	out.Items = []DocumentChecklistItem{
		{DocType: "car", Label: "Cadastro Ambiental Rural (CAR)", Group: "Imóvel e cadastro", SourceLabel: "SICAR", Automatic: true, Status: carStatus, StatusLabel: documentStatusLabel(carStatus), Detail: carDetail},
		{DocType: "kml_sicar", Label: "KML SICAR", Group: "Imóvel e cadastro", SourceLabel: "SICAR", Automatic: true, Status: kmlStatus, StatusLabel: documentStatusLabel(kmlStatus), Detail: kmlDetail},
	}
	for _, item := range out.Items {
		switch item.Status {
		case "received":
			out.Summary.Received++
		case "pending":
			out.Summary.Pending++
		case "review":
			out.Summary.Review++
		}
	}
	return out
}

func dossierDocumentsPage(p Property, r CARAutomationResult, docs PropertyDocumentCenter, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIÊ TÉCNICO DO IMÓVEL", "Documentos, pendências e estado do dossiê", page)
	y := 720.0

	proSection(&c, &y, "RESUMO DOCUMENTAL", "checklist do imóvel e documentos condicionais ao projeto")
	proMetric(&c, 40, y-72, 96, 60, "RECEBIDOS", fmt.Sprintf("%d", docs.Summary.Received), "documentos disponíveis", "")
	proMetric(&c, 145, y-72, 96, 60, "PENDENTES", fmt.Sprintf("%d", docs.Summary.Pending), "itens ainda necessários", func() string { if docs.Summary.Pending > 0 { return "danger" }; return "" }())
	proMetric(&c, 250, y-72, 96, 60, "CONFERIR", fmt.Sprintf("%d", docs.Summary.Review), "aplicabilidade ou conteúdo", func() string { if docs.Summary.Review > 0 { return "warn" }; return "" }())
	proMetric(&c, 355, y-72, 96, 60, "VENCIDOS", fmt.Sprintf("%d", docs.Summary.Expired), "validade objetiva expirada", func() string { if docs.Summary.Expired > 0 { return "danger" }; return "" }())
	proMetric(&c, 460, y-72, 95, 60, "NÃO SE APLICA", fmt.Sprintf("%d", docs.Summary.NotApplicable), "dispensados no contexto", "")
	y -= 92

	proSection(&c, &y, "CHECKLIST", "")
	if len(docs.Items) == 0 {
		proParagraph(&c, &y, "A consulta foi realizada sem imóvel salvo. O CAR e os relatórios podem ser gerados normalmente, mas a Central de Documentos exige vínculo com um imóvel local para armazenar arquivos e acompanhar pendências.")
	} else {
		for i, item := range docs.Items {
			if i >= 15 || y < 150 {
				break
			}
			value := item.StatusLabel
			if item.Status == "received" && item.Current != nil && strings.TrimSpace(item.Current.OriginalName) != "" {
				value += " • " + item.Current.OriginalName
			} else if strings.TrimSpace(item.Detail) != "" {
				value += " • " + item.Detail
			}
			if item.Conditional && item.Status == "review" {
				value += " • verificar necessidade conforme a finalidade do projeto"
			}
			proKV(&c, &y, item.Label, value)
		}
	}

	if r.PropertyID <= 0 {
		proNotice(&c, &y, "CONSULTA AVULSA", "Nenhum cliente ou imóvel foi criado automaticamente para gerar este dossiê. Para guardar matrícula, CCIR, ITR, orçamentos e demais documentos, vincule o CAR a um imóvel na Central de Documentos.", "warn")
	} else if docs.Summary.Pending+docs.Summary.Expired > 0 {
		proNotice(&c, &y, "PENDÊNCIAS DOCUMENTAIS", "Existem documentos pendentes ou vencidos no checklist. O ViaVerdeCAR registra o estado documental, mas a exigência final depende da finalidade do financiamento e da instituição financeira.", "warn")
	} else {
		proNotice(&c, &y, "ESTADO DOCUMENTAL", "O checklist não apresenta documento básico pendente ou vencido neste momento. Itens marcados como “Conferir” continuam dependendo da finalidade do projeto e da exigência da instituição financeira.", "")
	}
	proFooter(&c, page)
	return c.b.String()
}

func dossierSourcesPage(p Property, r CARAutomationResult, page int) string {
	var c pdfCanvas
	proHeader(&c, "DOSSIÊ TÉCNICO DO IMÓVEL", "Fontes consultadas, avisos e limitações", page)
	y := 720.0
	proSection(&c, &y, "FONTES DA EXECUÇÃO", "")
	for i, s := range r.Sources {
		if i >= 18 || y < 300 { break }
		value := reportAutomationSourceStatus(s)
		if s.Count > 0 {
			value += fmt.Sprintf(" • %d registro(s)", s.Count)
		}
		if detail := professionalSourceDetail(s); detail != "" {
			value += " • " + detail
		}
		proKV(&c, &y, s.Label, value)
	}

	proSection(&c, &y, "PONTOS DE ATENÇÃO", "")
	reportWarnings := professionalReportWarnings(r.Warnings)
	if len(reportWarnings) == 0 {
		proParagraph(&c, &y, "Nenhum ponto de atenção adicional foi registrado nesta execução.")
	} else {
		for i, w := range reportWarnings {
			if i >= 10 || y < 150 { break }
			proParagraph(&c, &y, "• "+w)
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
		return "Consulta concluída"
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

func reportAutomationSourceStatus(s AutomationSourceStatus) string {
	status := strings.TrimSpace(s.Status)
	switch s.Key {
	case "sicar":
		switch status {
		case "ok", "available":
			return "CAR localizado"
		case "cached":
			return "CAR recuperado do cache local"
		case "partial":
			return "CAR localizado com consulta parcial"
		case "not_found":
			return "CAR não localizado"
		case "unavailable":
			return "Base pública indisponível"
		}
	case "kml":
		switch status {
		case "ok", "available":
			return "Disponível"
		case "not_saved":
			return "Geometria disponível; arquivo não salvo"
		case "unavailable":
			return "Geração não confirmada"
		}
	case "sicor":
		if status == "hit" {
			return "Operação pública localizada"
		}
		if status == "ok" {
			return "Consulta concluída sem operação localizada"
		}
	case "sigef":
		if status == "hit" {
			return "Parcela pública localizada"
		}
		if status == "ok" {
			return "Consulta concluída sem parcela localizada"
		}
	case "mcr":
		if status == "hit" {
			return "CAR listado na base pública"
		}
		if status == "ok" {
			return "CAR não listado na base consultada"
		}
	}
	if strings.HasPrefix(s.Key, "bcb_") {
		return reportAutomationStatus(status)
	}
	if status == "ok" {
		return "Consulta concluída sem ocorrência"
	}
	return reportAutomationStatus(status)
}

func professionalSourceDetail(s AutomationSourceStatus) string {
	detail := strings.TrimSpace(s.Detail)
	if detail == "" {
		return ""
	}
	if strings.TrimSpace(s.Status) == "unavailable" || strings.TrimSpace(s.Status) == "not_configured" {
		switch s.Key {
		case "funai":
			return "Base geoespacial indisponível nesta execução."
		case "inpe_fire":
			return "Serviço de focos de calor indisponível nesta execução."
		case "sigef":
			return "Consulta SIGEF/INCRA indisponível nesta execução."
		default:
			if strings.HasPrefix(s.Key, "bcb_") {
				return "Consulta do Banco Central indisponível nesta execução."
			}
		}
	}
	if reportContainsInternalError(detail) {
		return "Fonte externa indisponível nesta execução."
	}
	return detail
}

func professionalReportWarnings(warnings []string) []string {
	out := make([]string, 0, len(warnings))
	seen := map[string]bool{}
	for _, warning := range warnings {
		clean := professionalReportWarning(warning)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	return out
}

func professionalReportWarning(warning string) string {
	w := strings.TrimSpace(warning)
	if w == "" {
		return ""
	}
	lower := strings.ToLower(w)
	switch {
	case strings.Contains(lower, "worldcover"):
		return "ESA WorldCover: base de cobertura do solo indisponível nesta execução."
	case strings.HasPrefix(lower, "funai:") || strings.Contains(lower, " funai:"):
		return "FUNAI: base geoespacial indisponível nesta execução."
	case strings.Contains(lower, "taxas por instituição bcb"):
		return "Banco Central: taxas por instituição indisponíveis nesta execução."
	case strings.Contains(lower, "entidades supervisionadas bcb"):
		return "Banco Central: cadastro de entidades supervisionadas indisponível nesta execução."
	case strings.HasPrefix(lower, "ifdata:") || strings.Contains(lower, " ifdata:"):
		return "Banco Central: IFData indisponível nesta execução."
	case strings.Contains(lower, "context deadline exceeded"),
		strings.Contains(lower, "client.timeout"),
		strings.Contains(lower, "invalid url"),
		strings.Contains(lower, "invalid character"),
		strings.Contains(lower, "unexpected end of json"),
		strings.Contains(lower, "connection reset"),
		strings.Contains(lower, "no such host"):
		return "Uma fonte externa ficou indisponível durante a consulta; o resultado foi mantido como indisponível, sem assumir ausência de ocorrência."
	default:
		return w
	}
}

func reportContainsInternalError(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"context deadline exceeded",
		"client.timeout",
		"invalid url",
		"invalid character",
		"unexpected end of json",
		"connection reset",
		"no such host",
		"http 5",
		"http 429",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
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
