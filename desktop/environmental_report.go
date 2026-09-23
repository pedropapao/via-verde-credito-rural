package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ExportEnvironmentalTechnicalReport(propertyID int64, force bool) (string,error) {
	if a.ctx==nil{return "",errors.New("aplicativo ainda não inicializado")}
	if propertyID<=0{return "",errors.New("salve ou selecione um imóvel antes de gerar o laudo ambiental")}
	p,err:=a.GetProperty(propertyID);if err!=nil{return "",err}
	car,err:=a.carForEnvironmental(propertyID);if err!=nil{return "",err}
	intel,err:=a.GetEnvironmentalIntelligence(propertyID,force);if err!=nil{return "",err}
	pdf:=buildEnvironmentalTechnicalPDF(p,car,intel,"")
	name:="Laudo_Tecnico_Ambiental_"+safeCARFilename(car.CAR)+".pdf"
	path,err:=runtime.SaveFileDialog(a.ctx,runtime.SaveDialogOptions{Title:"Salvar laudo técnico ambiental",DefaultFilename:name,Filters:[]runtime.FileFilter{{DisplayName:"PDF",Pattern:"*.pdf"}}})
	if err!=nil{return "",err}
	if strings.TrimSpace(path)==""{return "",errors.New("exportação cancelada")}
	if err:=os.WriteFile(path,pdf,0o644);err!=nil{return "",err}
	return path,nil
}

func (a *App) ExportMapBiomasAlertTechnicalReport(propertyID int64, alertCode string, force bool)(string,error){
	if a.ctx==nil{return "",errors.New("aplicativo ainda não inicializado")}
	if propertyID<=0{return "",errors.New("salve ou selecione um imóvel antes de gerar o laudo do alerta")}
	p,err:=a.GetProperty(propertyID);if err!=nil{return "",err}
	car,err:=a.carForEnvironmental(propertyID);if err!=nil{return "",err}
	intel,err:=a.GetEnvironmentalIntelligence(propertyID,force);if err!=nil{return "",err}
	if _,ok:=findEnvironmentalAlert(intel.Alerts,alertCode);!ok{return "",errors.New("alerta MapBiomas não localizado na análise atual")}
	pdf:=buildEnvironmentalTechnicalPDF(p,car,intel,strings.TrimSpace(alertCode))
	name:="Laudo_Alerta_MapBiomas_"+safeFilePart(alertCode)+"_"+safeCARFilename(car.CAR)+".pdf"
	path,err:=runtime.SaveFileDialog(a.ctx,runtime.SaveDialogOptions{Title:"Salvar laudo técnico do alerta MapBiomas",DefaultFilename:name,Filters:[]runtime.FileFilter{{DisplayName:"PDF",Pattern:"*.pdf"}}})
	if err!=nil{return "",err}
	if strings.TrimSpace(path)==""{return "",errors.New("exportação cancelada")}
	if err:=os.WriteFile(path,pdf,0o644);err!=nil{return "",err}
	return path,nil
}

func (a *App) ExportEnvironmentalEvidenceJSON(propertyID int64, force bool)(string,error){
	if a.ctx==nil{return "",errors.New("aplicativo ainda não inicializado")}
	if propertyID<=0{return "",errors.New("salve ou selecione um imóvel antes de exportar as evidências")}
	intel,err:=a.GetEnvironmentalIntelligence(propertyID,force);if err!=nil{return "",err}
	b,err:=json.MarshalIndent(intel,"","  ");if err!=nil{return "",err}
	name:="Evidencias_Ambientais_"+safeCARFilename(intel.CAR)+".json"
	path,err:=runtime.SaveFileDialog(a.ctx,runtime.SaveDialogOptions{Title:"Salvar evidências ambientais",DefaultFilename:name,Filters:[]runtime.FileFilter{{DisplayName:"JSON",Pattern:"*.json"}}})
	if err!=nil{return "",err}
	if strings.TrimSpace(path)==""{return "",errors.New("exportação cancelada")}
	if err:=os.WriteFile(path,b,0o644);err!=nil{return "",err}
	return path,nil
}

func buildEnvironmentalTechnicalPDF(p Property,car CARResult,intel EnvironmentalIntelligenceResult,alertFilter string)[]byte{
	// O relatório usa apenas fontes obtidas automaticamente nesta execução.
	car.Themes = SICARThemesSummary{}
	car.Environment = intel.Environment
	alerts:=intel.Alerts
	title:="LAUDO TÉCNICO DE TRIAGEM AMBIENTAL"
	subtitle:="CAR, MapBiomas Alerta e cruzamentos territoriais públicos"
	if alertFilter!=""{
		title="LAUDO TÉCNICO DE ALERTA MAPBIOMAS"
		subtitle="Análise de evidências do alerta "+alertFilter+" no imóvel consultado"
		if a,ok:=findEnvironmentalAlert(alerts,alertFilter);ok{alerts=[]EnvironmentalAlertDetail{a}}
	}
	var pages []string
	pages=append(pages,environmentalCoverPage(p,car,intel,title,subtitle,alerts))
	pages=append(pages,environmentalMapPage(p,car,intel,title,alerts))
	for _,a:=range alerts{
		pages=append(pages,environmentalAlertPage(p,car,intel,a,title))
	}
	pages=append(pages,environmentalSourcesPage(p,car,intel,title,alertFilter))
	return assembleMultiPagePDF(pages)
}

func environmentalCoverPage(p Property,car CARResult,intel EnvironmentalIntelligenceResult,title,subtitle string,alerts []EnvironmentalAlertDetail)string{
	var c pdfCanvas
	envReportHeader(&c,title,subtitle,1)
	y:=716.0
	envSection(&c,&y,"IDENTIFICAÇÃO DO IMÓVEL")
	envRow(&c,&y,"Cliente",p.ClientName)
	envRow(&c,&y,"Imóvel",p.Name)
	envRow(&c,&y,"Município / UF",strings.Trim(strings.TrimSpace(firstNonEmptyText(p.Municipality,car.Municipality)+" / "+firstNonEmptyText(p.UF,car.UF))," /"))
	envRow(&c,&y,"CAR",car.CAR)
	envRow(&c,&y,"Área do imóvel",fmtBR(firstPositive(car.GeometryAreaHa,car.AreaHa),4)+" ha")
	envRow(&c,&y,"Situação SICAR",firstNonEmptyText(car.Status,"Não informada"))

	y-=5
	envSection(&c,&y,"SÍNTESE EXECUTIVA")
	s:=intel.Summary
	if len(alerts)!=len(intel.Alerts){
		s=summarizeEnvironmentalEvidence(alerts)
	}
	envMetricBox(&c,40,y-60,122,54,"Alertas MapBiomas",fmt.Sprintf("%d",s.Alerts),"vinculados ao CAR")
	envMetricBox(&c,172,y-60,122,54,"Área dos alertas",fmtBR(s.AlertAreaInCARHa,2)+" ha","estimada dentro do CAR")
	envMetricBox(&c,304,y-60,122,54,"Alta prioridade",fmt.Sprintf("%d",s.HighAttentionAlerts),"alerta(s) para conferência")
	envMetricBox(&c,436,y-60,119,54,"Última detecção",dateBR(s.LatestDetection),"MapBiomas Alerta")
	y-=76
	envMetricBox(&c,40,y-60,122,54,"Embargo IBAMA",fmt.Sprintf("%d",s.AlertsOverIBAMA),"alerta(s) com interseção")
	envMetricBox(&c,172,y-60,122,54,"Terra Indígena",fmt.Sprintf("%d",s.AlertsOverIndigenousLand),"alerta(s) com interseção")
	envMetricBox(&c,304,y-60,122,54,"UC Federal",fmt.Sprintf("%d",s.AlertsOverFederalUC),"alerta(s) com interseção")
	mcr:="Não listado"
	if intel.Environment.MCRListed{mcr="LISTADO"}
	if !intel.Environment.MCRChecked{mcr="Não verificado"}
	envMetricBox(&c,436,y-60,119,54,"MMA / MCR",mcr,"lista pública PRODES")
	y-=82

	envSection(&c,&y,"CONCLUSÃO DA TRIAGEM")
	conclusion:=environmentalConclusionText(intel,alerts)
	c.b.WriteString("0.10 0.18 0.14 rg\n")
	y=c.wrapped(42,y,9,false,conclusion,104,13)
	y-=9
	c.b.WriteString("0.94 0.97 0.95 rg\n")
	c.rect(40,y-72,515,70,true)
	c.b.WriteString("0.08 0.30 0.21 rg\n")
	c.text(50,y-18,8,true,"NATUREZA DO DOCUMENTO")
	c.b.WriteString("0.20 0.27 0.23 rg\n")
	note:="Laudo técnico auxiliar de triagem e organização de evidências públicas. Não substitui laudo oficial do MapBiomas, licença, autorização, certidão, vistoria de campo, parecer do órgão ambiental, perícia, ART/TRT ou manifestação de profissional habilitado quando exigidos."
	c.wrapped(50,y-34,7.8,false,note,100,10)
	envReportFooter(&c,1)
	return c.b.String()
}

func environmentalMapPage(p Property,car CARResult,intel EnvironmentalIntelligenceResult,title string,alerts []EnvironmentalAlertDetail)string{
	var c pdfCanvas
	envReportHeader(&c,title,"Mapa técnico e matriz de fontes",2)
	c.b.WriteString("0.10 0.18 0.14 rg\n")
	c.text(40,720,10,true,"MAPA ESQUEMÁTICO DE EVIDÊNCIAS")
	c.text(40,704,7.5,false,"Perímetro do CAR, alertas MapBiomas e camadas públicas disponíveis. Sem base cartográfica; uso para conferência espacial.")
	c.b.WriteString("0.80 0.86 0.82 RG 0.8 w\n")
	c.rect(40,378,515,306,false)
	drawEnvironmentalEvidenceMap(&c,car,alerts,52,400,491,260)

	y:=350.0
	envSection(&c,&y,"MATRIZ DE FONTES CONSULTADAS")
	envSourceRow(&c,&y,"MapBiomas Alerta",sourceState(intel.MapBiomas.Connected,intel.MapBiomas.TotalAlerts),fmt.Sprintf("%d alerta(s); %s ha somados",intel.MapBiomas.TotalAlerts,fmtBR(intel.MapBiomas.TotalAreaHa,2)))
	envSourceRow(&c,&y,"IBAMA / PAMGIA",sourceState(intel.Environment.IBAMAChecked,intel.Environment.IBAMAEmbargoCount),fmt.Sprintf("%d interseção(ões) com embargo no CAR",intel.Environment.IBAMAEmbargoCount))
	envSourceRow(&c,&y,"FUNAI",sourceState(intel.Environment.FUNAIChecked,intel.Environment.IndigenousCount),fmt.Sprintf("%d interseção(ões) com Terra Indígena no CAR",intel.Environment.IndigenousCount))
	envSourceRow(&c,&y,"ICMBio",sourceState(intel.Environment.ICMBioChecked,intel.Environment.FederalUCCount),fmt.Sprintf("%d interseção(ões) com UC federal no CAR",intel.Environment.FederalUCCount))
	envSourceRow(&c,&y,"MMA / MCR-PRODES",sourceState(intel.Environment.MCRChecked,boolInt(intel.Environment.MCRListed)),mcrSourceSummary(intel.Environment))
	envReportFooter(&c,2)
	return c.b.String()
}

func environmentalAlertPage(p Property,car CARResult,intel EnvironmentalIntelligenceResult,a EnvironmentalAlertDetail,title string)string{
	var c pdfCanvas
	envReportHeader(&c,title,"Alerta "+a.AlertCode+" — evidências e cruzamentos",3)
	y:=718.0
	envSection(&c,&y,"DADOS DO ALERTA")
	envRow(&c,&y,"Código",a.AlertCode)
	envRow(&c,&y,"Área do alerta",fmtBR(a.AreaHa,4)+" ha")
	envRow(&c,&y,"Área estimada dentro do CAR",fmtBR(a.AlertAreaInCAR,4)+" ha ("+fmtBR(a.AlertPctOfCAR,2)+"% do imóvel)")
	envRow(&c,&y,"Detecção / publicação",dateBR(a.DetectedAt)+" / "+dateBR(a.PublishedAt))
	envRow(&c,&y,"Imagens antes / depois",dateBR(a.ImageBeforeAt)+" / "+dateBR(a.ImageAfterAt))
	envRow(&c,&y,"Fonte(s)",strings.Join(a.Sources,", "))
	envRow(&c,&y,"Bioma(s)",strings.Join(a.Biomes,", "))
	envRow(&c,&y,"Município(s)",strings.Join(a.Cities,", "))
	envRow(&c,&y,"Classe(s)",strings.Join(a.DeforestationClasses,", "))
	envRow(&c,&y,"Velocidade informada",a.DeforestationSpeed)
	envRow(&c,&y,"Status na plataforma",strings.TrimSpace(a.StatusName+" "+dateBR(a.StatusAt)))

	y-=5
	envSection(&c,&y,"CRUZAMENTOS NO IMÓVEL")
	envCompactMetricState(&c,&y,"Embargo IBAMA",a.IBAMAOverlapHa,intel.Environment.IBAMAChecked,"PAMGIA × geometria do alerta")
	envCompactMetricState(&c,&y,"Terra Indígena",a.IndigenousOverlapHa,intel.Environment.FUNAIChecked,"FUNAI × geometria do alerta")
	envCompactMetricState(&c,&y,"UC federal",a.FederalUCOverlapHa,intel.Environment.ICMBioChecked,"ICMBio × geometria do alerta")

	y-=4
	envSection(&c,&y,"CRUZAMENTOS REPORTADOS PELO MAPBIOMAS")
	envCompactMetric(&c,&y,"Autorização de supressão",a.MapBiomasAuthorizedAreaHa,"cruzamento informado pela API")
	envCompactMetric(&c,&y,"Manejo florestal",a.MapBiomasForestManagementAreaHa,"cruzamento informado pela API")
	envCompactMetric(&c,&y,"Embargo / imóvel rural",a.MapBiomasEmbargoAreaHa,"cruzamento informado pela API")
	envCompactMetric(&c,&y,"Reserva Legal",a.MapBiomasLegalReserveAreaHa,"cruzamento informado pela API")
	envCompactMetric(&c,&y,"APP",a.MapBiomasPermanentProtectedAreaHa,"cruzamento informado pela API")

	y-=4
	envSection(&c,&y,"PRIORIDADE DE CONFERÊNCIA")
	c.b.WriteString("0.08 0.30 0.21 rg\n")
	c.text(42,y,9,true,firstNonEmptyText(a.AttentionLevel,"Conferir"))
	y-=15
	c.b.WriteString("0.20 0.27 0.23 rg\n")
	if len(a.AttentionReasons)==0{
		y=c.wrapped(42,y,7.7,false,"O alerta deve ser confrontado com imagens, documentos e autorizações aplicáveis. A existência do alerta não é declaração de ilegalidade.",104,10)
	}else{
		for _,r:=range a.AttentionReasons{
			y=c.wrapped(46,y,7.5,false,"• "+r,100,10)
		}
	}
	envReportFooter(&c,3)
	return c.b.String()
}

func environmentalSourcesPage(p Property,car CARResult,intel EnvironmentalIntelligenceResult,title,alertFilter string)string{
	var c pdfCanvas
	envReportHeader(&c,title,"Metodologia, rastreabilidade e limitações",4)
	y:=718.0
	envSection(&c,&y,"METODOLOGIA")
	method:="1) identificação do imóvel pelo CAR e geometria pública do SICAR; 2) consulta autenticada à API V2 do MapBiomas Alerta para alertas vinculados ao CAR; 3) leitura das geometrias e atributos retornados, inclusive cruzamentos territoriais informados pela própria API; 4) cruzamento espacial local do alerta com embargos IBAMA/PAMGIA, Terras Indígenas FUNAI e UCs federais ICMBio; 5) conferência da lista pública MMA/MCR-PRODES; 6) organização dos achados em relatório rastreável, sem inferir autoria ou regularidade jurídica."
	c.b.WriteString("0.15 0.23 0.19 rg\n")
	y=c.wrapped(42,y,8,false,method,104,11)
	y-=10

	envSection(&c,&y,"FONTES")
	for _,row:=range []struct{n,u string}{
		{"MapBiomas Alerta — API V2","https://plataforma.alerta.mapbiomas.org/api/v2/graphql"},
		{"MapBiomas Alerta — metodologia",mapBiomasMethodologyURL},
		{"SICAR — perímetro público do CAR",carWFSURL},
		{"IBAMA / PAMGIA","https://pamgia.ibama.gov.br/"},
		{"FUNAI — dados geoespaciais",funaiGeoURL},
		{"ICMBio — dados geoespaciais",icmbioGeoURL},
		{"MMA — atendimento ao Manual de Crédito Rural",environmentMCRURL},
	}{
		c.b.WriteString("0.08 0.30 0.21 rg\n");c.text(42,y,7.6,true,row.n)
		c.b.WriteString("0.30 0.38 0.34 rg\n");y=c.wrapped(178,y,6.6,false,row.u,66,9);y-=6
	}
	y-=4

	envSection(&c,&y,"LIMITAÇÕES E SALVAGUARDAS")
	limits:=[]string{
		"A área somada de alertas pode conter sobreposição temporal/espacial entre eventos; não deve ser tratada automaticamente como área única desmatada.",
		"Quando o MapBiomas informa cruzamentos com APP, Reserva Legal ou outras categorias, esses valores são exibidos como atributos da própria API e devem ser conferidos na fonte.",
		"Cruzamentos geométricos do ViaVerdeCAR são auxiliares e dependem da precisão e atualidade das geometrias das fontes.",
		"Alertas do MapBiomas são evidências de mudança de cobertura validadas pela metodologia da plataforma, mas não constituem por si só decisão administrativa, constatação de autoria ou juízo de legalidade.",
		"Autorizações, licenças, termos, embargos, datas e situação cadastral devem ser conferidos no documento e órgão competente antes de qualquer conclusão técnica ou financeira.",
	}
	for _,v:=range limits{y=c.wrapped(46,y,7.4,false,"• "+v,100,10);y-=3}

	if len(intel.Warnings)>0 && y>145{
		envSection(&c,&y,"OCORRÊNCIAS DA CONSULTA")
		for _,v:=range intel.Warnings{
			if y<105{break}
			y=c.wrapped(46,y,6.8,false,"• "+v,102,9);y-=2
		}
	}
	envReportFooter(&c,4)
	return c.b.String()
}

func environmentalConclusionText(intel EnvironmentalIntelligenceResult,alerts []EnvironmentalAlertDetail)string{
	s:=summarizeEnvironmentalEvidence(alerts)
	if len(alerts)==0{
		if !intel.MapBiomas.Connected{
			return "A consulta MapBiomas Alerta não estava autenticada nesta execução. As demais camadas públicas permanecem como triagem auxiliar. Recomenda-se conectar a API e atualizar a análise antes da emissão final."
		}
		return "A consulta não retornou alerta MapBiomas vinculado ao CAR no momento da análise. Este resultado descreve a base consultada e não constitui certificado de regularidade ambiental nem substitui a verificação documental."
	}
	text:=fmt.Sprintf("Foram identificados %d alerta(s) MapBiomas vinculados ao CAR, com %.4f ha somados como área estimada dentro do imóvel.",s.Alerts,s.AlertAreaInCARHa)
	if s.AlertsOverIBAMA>0||s.AlertsOverIndigenousLand>0||s.AlertsOverFederalUC>0{
		text+=fmt.Sprintf(" Há alertas com interseção espacial estimada em camadas sensíveis: embargo IBAMA (%d), Terra Indígena (%d) e UC federal (%d).",s.AlertsOverIBAMA,s.AlertsOverIndigenousLand,s.AlertsOverFederalUC)
	}
	if s.HighAttentionAlerts>0{
		text+=fmt.Sprintf(" %d alerta(s) receberam alta prioridade de conferência pelas regras objetivas descritas no relatório.",s.HighAttentionAlerts)
	}
	text+=" A conclusão jurídica ou de elegibilidade para crédito depende da conferência de autorizações, documentos, datas, situação dos registros e análise competente."
	return text
}

func envReportHeader(c *pdfCanvas,title,subtitle string,page int){
	c.b.WriteString("0.055 0.42 0.29 rg\n");c.rect(0,760,595,82,true)
	c.b.WriteString("1 1 1 rg\n");c.text(38,808,15,true,"VIA VERDE CAR")
	c.text(38,788,9,true,title);c.text(38,774,7,false,subtitle)
	c.b.WriteString("0.10 0.18 0.14 rg\n")
	c.text(520,744,6.5,false,fmt.Sprintf("p. %d",page))
}

func envReportFooter(c *pdfCanvas,page int){
	c.b.WriteString("0.82 0.87 0.84 RG 0.5 w\n");c.line(40,45,555,45)
	c.b.WriteString("0.35 0.42 0.39 rg\n")
	c.text(40,30,6.3,false,"Gerado em "+time.Now().Format("02/01/2006 15:04")+" • Via Verde CAR v"+AppVersion+" • relatório técnico auxiliar")
	c.text(520,30,6.3,false,fmt.Sprintf("%d",page))
}

func envSection(c *pdfCanvas,y *float64,title string){
	c.b.WriteString("0.08 0.30 0.21 rg\n");c.text(40,*y,8,true,title)
	c.b.WriteString("0.82 0.87 0.84 RG 0.5 w\n");c.line(40,*y-5,555,*y-5);*y-=20
}

func envRow(c *pdfCanvas,y *float64,label,value string){
	value=strings.TrimSpace(value);if value==""{value="Não informado"}
	c.b.WriteString("0.35 0.42 0.39 rg\n");c.text(42,*y,7.2,true,label)
	c.b.WriteString("0.08 0.15 0.12 rg\n");ny:=c.wrapped(174,*y,8,false,value,66,10);*y=ny-5
}

func envMetricBox(c *pdfCanvas,x,y,w,h float64,label,value,detail string){
	c.b.WriteString("0.96 0.98 0.97 rg\n");c.rect(x,y,w,h,true)
	c.b.WriteString("0.80 0.86 0.82 RG 0.6 w\n");c.rect(x,y,w,h,false)
	c.b.WriteString("0.35 0.42 0.39 rg\n");c.text(x+8,y+h-14,6.2,true,label)
	c.b.WriteString("0.08 0.30 0.21 rg\n");c.text(x+8,y+h-31,10,true,value)
	c.b.WriteString("0.35 0.42 0.39 rg\n");c.text(x+8,y+8,5.7,false,detail)
}

func envCompactMetric(c *pdfCanvas,y *float64,label string,value float64,source string){
	envCompactMetricState(c,y,label,value,true,source)
}

func envCompactMetricState(c *pdfCanvas,y *float64,label string,value float64,available bool,source string){
	c.b.WriteString("0.35 0.42 0.39 rg\n");c.text(42,*y,7.1,true,label)
	c.b.WriteString("0.08 0.15 0.12 rg\n")
	valueText := fmtBR(value,4)+" ha"
	if !available {
		valueText = "Indisponível"
		source = source+" • fonte não obtida"
	}
	c.text(230,*y,7.4,false,valueText)
	c.b.WriteString("0.40 0.46 0.42 rg\n");c.text(334,*y,6.2,false,source);*y-=13
}

func envSourceRow(c *pdfCanvas,y *float64,name,status,detail string){
	c.b.WriteString("0.08 0.30 0.21 rg\n");c.text(42,*y,7.2,true,name)
	c.b.WriteString("0.15 0.23 0.19 rg\n");c.text(210,*y,7,false,status)
	c.b.WriteString("0.35 0.42 0.39 rg\n");ny:=c.wrapped(328,*y,6.5,false,detail,40,8.5);*y=ny-5
}

func sourceState(checked bool,count int)string{
	if !checked{return "Indisponível / não consultado"}
	if count>0{return "Ocorrência localizada"}
	return "Consulta concluída"
}
func themeSourceSummary(t SICARThemesSummary)string{
	available:=availableSICARThemeCount(t)
	if available==0 {
		return "0/6 temas disponíveis; fonte indisponível nesta execução, sem assumir área zero"
	}
	stale:=0
	for _,m:=range t.Themes{if m.Available && (m.CacheStatus=="cache_stale" || m.CacheStatus=="manual"){stale++}}
	if stale>0 {
		return fmt.Sprintf("%d/6 tema(s) disponível(is); %d por cache/importação; dados declarados do SICAR",available,stale)
	}
	return fmt.Sprintf("%d/6 tema(s) disponível(is); dados declarados do SICAR",available)
}

func availableSICARThemeCount(t SICARThemesSummary)int{
	n:=0
	for _,m:=range t.Themes{if m.Available{n++}}
	return n
}

func sicarThemeAvailable(t SICARThemesSummary,code string)bool{
	m,ok:=t.Themes[code]
	return ok&&m.Available
}

func environmentalThemeOverlapLabel(t SICARThemesSummary,code string,value float64)(string,string){
	m,ok:=t.Themes[code]
	if !ok||!m.Available {
		return "Indisponível","fonte SICAR não obtida"
	}
	detail:="interseção calculada"
	switch m.CacheStatus {
	case "cache_stale":
		detail=fmt.Sprintf("cache anterior • %.0f h",m.CacheAgeHours)
	case "manual":
		detail="pacote oficial importado"
	case "cache_fresh":
		detail="cache recente do SICAR"
	}
	return fmtBR(value,2)+" ha",detail
}
func mcrSourceSummary(e EnvironmentalSummary)string{
	if !e.MCRChecked{return "lista não consultada nesta execução"}
	if e.MCRListed{return "CAR localizado na lista pública MMA/MCR-PRODES"}
	return "CAR não localizado na lista pública consultada"
}
func boolInt(v bool)int{if v{return 1};return 0}
func firstNonEmptyText(v ...string)string{for _,x:=range v{if strings.TrimSpace(x)!=""{return strings.TrimSpace(x)}};return ""}
func dateBR(v string)string{
	v=strings.TrimSpace(v);if v==""{return "—"}
	for _,layout:=range []string{"2006-01-02",time.RFC3339}{
		if t,err:=time.Parse(layout,v);err==nil{return t.Format("02/01/2006")}
	}
	return v
}

func drawEnvironmentalEvidenceMap(c *pdfCanvas,car CARResult,alerts []EnvironmentalAlertDetail,x,y,w,h float64){
	type layer struct{label,geo,rgb string;width float64;bound bool}
	layers:=[]layer{{"CAR",car.GeoJSON,"0.055 0.42 0.29",1.8,true}}
	for _,a:=range alerts{if strings.TrimSpace(a.GeometryGeoJSON)!=""{layers=append(layers,layer{"Alerta "+a.AlertCode,a.GeometryGeoJSON,"0.78 0.16 0.12",1.2,false})}}
	for _,e:=range car.Environment.IBAMAEmbargos{if strings.TrimSpace(e.GeoJSON)!=""{layers=append(layers,layer{"Embargo IBAMA",e.GeoJSON,"0.58 0.12 0.12",1,false})}}
	type parsedLayer struct{label,rgb string;width float64;polys [][][][]float64;bound bool}
	var parsed []parsedLayer
	minX,minY:=math.Inf(1),math.Inf(1);maxX,maxY:=math.Inf(-1),math.Inf(-1)
	addBounds:=func(polys [][][][]float64){for _,poly:=range polys{for _,ring:=range poly{for _,p:=range ring{if len(p)<2{continue};minX=math.Min(minX,p[0]);maxX=math.Max(maxX,p[0]);minY=math.Min(minY,p[1]);maxY=math.Max(maxY,p[1])}}}}
	for _,l:=range layers{
		if strings.TrimSpace(l.geo)==""{continue}
		var all [][][][]float64
		for _,f:=range geoJSONFeatures(l.geo){if ps,err:=carGeometryPolygons(f.Geometry);err==nil{all=append(all,ps...)}}
		if len(all)==0{continue}
		if l.bound{addBounds(all)}
		parsed=append(parsed,parsedLayer{l.label,l.rgb,l.width,all,l.bound})
	}
	if math.IsInf(minX,1)||maxX<=minX||maxY<=minY{for _,l:=range parsed{addBounds(l.polys)}}
	if math.IsInf(minX,1)||maxX<=minX||maxY<=minY{return}
	scale:=math.Min(w/(maxX-minX),h/(maxY-minY));offX:=x+(w-(maxX-minX)*scale)/2;offY:=y+(h-(maxY-minY)*scale)/2
	c.b.WriteString("q\n");fmt.Fprintf(&c.b,"%.2f %.2f %.2f %.2f re W n\n",x,y,w,h)
	for _,l:=range parsed{
		c.b.WriteString(l.rgb+" RG\n");fmt.Fprintf(&c.b,"%.1f w\n",l.width)
		for _,poly:=range l.polys{for _,ring:=range poly{
			if len(ring)<2{continue};step:=1;if len(ring)>600{step=int(math.Ceil(float64(len(ring))/600))}
			started:=false
			for i:=0;i<len(ring);i+=step{p:=ring[i];if len(p)<2{continue};px:=offX+(p[0]-minX)*scale;py:=offY+(p[1]-minY)*scale;if !started{fmt.Fprintf(&c.b,"%.2f %.2f m\n",px,py);started=true}else{fmt.Fprintf(&c.b,"%.2f %.2f l\n",px,py)}}
			if started{c.b.WriteString("h S\n")}
		}}
	}
	c.b.WriteString("Q\n")
	c.b.WriteString("0.08 0.18 0.14 rg\n");c.text(x+w-18,y+h-14,8,true,"N");c.line(x+w-14,y+h-32,x+w-14,y+h-17)
	seen:=map[string]bool{};lx,ly:=x+4,y-13;col:=0
	for _,l:=range parsed{
		if seen[l.label]{continue};seen[l.label]=true
		c.b.WriteString(l.rgb+" rg\n");c.rect(lx+float64(col)*96,ly+3,8,3,true)
		c.b.WriteString("0.20 0.27 0.23 rg\n");c.text(lx+11+float64(col)*96,ly,5.2,false,l.label)
		col++;if col>=5{break}
	}
}

func assembleMultiPagePDF(contents []string)[]byte{
	if len(contents)==0{return assembleSimplePDF("")}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%âãÏÓ\n")
	pageCount:=len(contents)
	totalObjects:=4+pageCount*2
	offsets:=make([]int,totalObjects+1)
	writeObj:=func(n int,body string){offsets[n]=out.Len();fmt.Fprintf(&out,"%d 0 obj\n%s\nendobj\n",n,body)}
	writeObj(1,"<< /Type /Catalog /Pages 2 0 R >>")
	var kids strings.Builder
	for i:=0;i<pageCount;i++{fmt.Fprintf(&kids,"%d 0 R ",5+i*2)}
	writeObj(2,fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>",kids.String(),pageCount))
	writeObj(3,"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	writeObj(4,"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	for i,content:=range contents{
		pageObj:=5+i*2;streamObj:=pageObj+1
		writeObj(pageObj,fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>",streamObj))
		writeObj(streamObj,fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream",len(content),content))
	}
	xref:=out.Len();fmt.Fprintf(&out,"xref\n0 %d\n0000000000 65535 f \n",totalObjects+1)
	for i:=1;i<=totalObjects;i++{fmt.Fprintf(&out,"%010d 00000 n \n",offsets[i])}
	fmt.Fprintf(&out,"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",totalObjects+1,xref)
	return out.Bytes()
}
