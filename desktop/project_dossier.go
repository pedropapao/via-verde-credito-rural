package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ExportRuralProjectDossierPDF(projectID int64, force bool) (string,error) {
	if a==nil || a.ctx==nil { return "",errors.New("aplicativo ainda não inicializado") }
	project,err:=a.GetRuralProject(projectID);if err!=nil{return "",err}
	property,err:=a.GetProperty(project.PropertyID);if err!=nil{return "",err}
	auto,err:=a.RunRuralProjectAutomation(projectID,force);if err!=nil{return "",err}
	pdf:=buildRuralProjectDossierPDF(property,project,auto)
	name:="Dossie_Projeto_"+safeFilePart(project.Name)+"_"+time.Now().Format("20060102")+".pdf"
	path,err:=runtime.SaveFileDialog(a.ctx,runtime.SaveDialogOptions{
		Title:"Salvar dossiê técnico do projeto",DefaultFilename:name,
		Filters:[]runtime.FileFilter{{DisplayName:"PDF",Pattern:"*.pdf"}},
	})
	if err!=nil{return "",err}
	if strings.TrimSpace(path)==""{return "",errors.New("exportação cancelada")}
	if !strings.HasSuffix(strings.ToLower(path),".pdf"){path+=".pdf"}
	if err:=os.WriteFile(path,pdf,0o644);err!=nil{return "",err}
	return path,nil
}

func dossierStatusLabel(v string) string {
	switch v {
	case "ready":return "Concluído"
	case "pending":return "Pendente"
	case "review":return "Conferir"
	case "source_unavailable":return "Fonte indisponível"
	case "not_configured":return "Integração não configurada"
	case "not_applicable":return "Não se aplica"
	case "not_run":return "Não consultado"
	case "error":return "Erro"
	default:return firstNonEmptyText(v,"Não informado")
	}
}

func buildRuralProjectDossierPDF(property Property,project RuralProject,auto RuralProjectAutomationResult) []byte {
	tech:=auto.Preparation.TechnicalData
	page1:=newProjectDossierPage("DOSSIÊ TÉCNICO DO PROJETO",project.Name)
	page1.section("IDENTIFICAÇÃO")
	page1.row("Cliente",property.ClientName)
	page1.row("Imóvel",property.Name)
	page1.row("Município / UF",strings.Trim(strings.TrimSpace(property.Municipality+" / "+property.UF)," /"))
	page1.row("CAR",property.CARNumber)
	page1.row("Banco / cooperativa",project.Bank)
	page1.row("Linha / programa",project.CreditLine)
	page1.row("Operação",projectOperationLabel(project.OperationType))
	page1.row("Atividade",project.Activity)
	page1.row("Valor solicitado",fmt.Sprintf("R$ %s",fmtBR(project.RequestedAmount,2)))
	page1.row("Área do projeto",fmtBR(project.AreaHa,2)+" ha")
	page1.row("Prontidão operacional",fmt.Sprintf("%d%% • %d pronto(s) • %d pendente(s) • %d para conferir",auto.Preparation.ReadinessPct,auto.Preparation.Ready,auto.Preparation.Pending,auto.Preparation.Review))

	page1.section("DADOS TÉCNICOS")
	if tech.Culture!="" { page1.row("Cultura",tech.Culture) }
	if tech.CropSeason!="" { page1.row("Safra",tech.CropSeason) }
	if tech.PlantingStart!="" { page1.row("Plantio",strings.Trim(strings.TrimSpace(tech.PlantingStart+" a "+tech.PlantingEnd)," a")) }
	if tech.ExpectedProductivity>0 { page1.row("Produtividade esperada",fmtBR(tech.ExpectedProductivity,2)+" "+tech.ProductivityUnit) }
	if tech.Soil!="" || tech.Cycle!="" { page1.row("Solo / ciclo",strings.Trim(strings.TrimSpace(tech.Soil+" / "+tech.Cycle)," /")) }
	if tech.BenefitedAreaHa>0 { page1.row("Área beneficiada",fmtBR(tech.BenefitedAreaHa,2)+" ha") }
	if len(tech.Items)>0 {
		page1.row("Itens investimento",fmt.Sprintf("%d item(ns) • total estimado R$ %s",len(tech.Items),fmtBR(projectTechnicalBudgetTotal(tech),2)))
	}
	if tech.AnimalCount>0 { page1.row("Pecuária",fmt.Sprintf("%d animal(is) • %s • peso médio %s kg",tech.AnimalCount,tech.AnimalCategory,fmtBR(tech.AverageWeightKg,1))) }
	if tech.IrrigatedAreaHa>0 { page1.row("Irrigação",fmt.Sprintf("%s ha • %s • fonte %s",fmtBR(tech.IrrigatedAreaHa,2),tech.IrrigationSystem,tech.WaterSource)) }
	if tech.TechnicalPurpose!="" { page1.row("Finalidade técnica",tech.TechnicalPurpose) }
	if tech.Notes!="" { page1.row("Observações técnicas",tech.Notes) }
	page1.footer("Gerado pelo ViaVerdeCAR "+AppVersion+" em "+time.Now().Format("02/01/2006 15:04"))

	page2:=newProjectDossierPage("AUTOMAÇÃO TÉCNICA",project.Name)
	page2.section("FONTES E ANÁLISES")
	page2.row("ZARC / MAPA",dossierStatusLabel(auto.ZARC.Status)+" — "+auto.ZARC.Detail)
	page2.row("Agritec / Embrapa",dossierStatusLabel(auto.Agritec.Status)+" — "+auto.Agritec.Detail)
	if auto.Agritec.Culture.ID>0 {
		page2.row("Cultura Agritec",fmt.Sprintf("%d • %s",auto.Agritec.Culture.ID,firstNonEmptyText(auto.Agritec.Culture.FullName,auto.Agritec.Culture.Name)))
	}
	if len(auto.Agritec.Cultivars)>0 {
		max:=len(auto.Agritec.Cultivars);if max>8{max=8}
		page2.row("Cultivares localizadas",strings.Join(auto.Agritec.Cultivars[:max],"; "))
	}
	page2.row("ANA / HidroWebService",dossierStatusLabel(auto.ANA.Status)+" — "+auto.ANA.Detail)
	if len(auto.ANA.Stations)>0 {
		s:=auto.ANA.Stations[0]
		page2.row("Estação ANA referência",fmt.Sprintf("%s (%s) • %.1f km",s.Name,s.Code,s.DistanceKM))
	}
	if auto.ANA.RainValueCount>0 { page2.row("Chuva — período consultado",fmt.Sprintf("%s a %s • %d valor(es) • soma bruta %.1f mm",auto.ANA.PeriodStart,auto.ANA.PeriodEnd,auto.ANA.RainValueCount,auto.ANA.RainTotalMM)) }
	if auto.ANA.FlowValueCount>0 { page2.row("Vazão — período consultado",fmt.Sprintf("%s a %s • %d valor(es) • média %.2f m³/s",auto.ANA.PeriodStart,auto.ANA.PeriodEnd,auto.ANA.FlowValueCount,auto.ANA.FlowMeanM3S)) }
	page2.row("SATVeg / NDVI",dossierStatusLabel(auto.SATVeg.Status)+" — "+auto.SATVeg.Detail)
	if auto.SATVeg.Count>0 { page2.row("Série vegetativa",fmt.Sprintf("%s a %s • %d observações • NDVI atual %.3f • média %.3f",auto.SATVeg.FirstDate,auto.SATVeg.LastDate,auto.SATVeg.Count,auto.SATVeg.LatestValue,auto.SATVeg.MeanValue)) }

	page2.section("PENDÊNCIAS E AÇÕES")
	if len(auto.Pending)==0 {
		page2.row("Situação","Nenhuma pendência operacional detectada nas verificações contabilizadas.")
	} else {
		max:=len(auto.Pending);if max>14{max=14}
		for _,p:=range auto.Pending[:max] {
			page2.row(p.Label,strings.ToUpper(p.Severity)+" — "+p.Detail)
		}
		if len(auto.Pending)>max { page2.row("Outras pendências",fmt.Sprintf("%d item(ns) adicionais. Consulte a Central de Projetos.",len(auto.Pending)-max)) }
	}
	page2.note("IMPORTANTE",auto.Scope)
	page2.footer("Este dossiê consolida dados auxiliares. Confirme ocorrências, documentos, ZARC, séries hidrológicas e requisitos bancários nas fontes oficiais antes da conclusão técnica.")

	return assembleProjectDossierPDF([]string{page1.content(),page2.content()})
}

type projectDossierPage struct {
	c pdfCanvas
	y float64
}

func newProjectDossierPage(title,subtitle string)*projectDossierPage{
	p:=&projectDossierPage{y:746}
	p.c.b.WriteString("0.055 0.42 0.29 rg\n");p.c.rect(0,772,595,70,true)
	p.c.b.WriteString("1 1 1 rg\n");p.c.text(38,814,16,true,"VIA VERDE CAR");p.c.text(38,793,9,false,title)
	p.c.b.WriteString("0.08 0.16 0.12 rg\n");p.c.text(38,754,13,true,firstNonEmptyText(subtitle,"Projeto rural"))
	p.y=724
	return p
}

func (p *projectDossierPage)section(v string){
	if p.y<100{return}
	p.c.b.WriteString("0.91 0.96 0.93 rg\n");p.c.rect(38,p.y-4,519,22,true)
	p.c.b.WriteString("0.06 0.34 0.22 rg\n");p.c.text(46,p.y+3,8,true,v)
	p.y-=33
}

func (p *projectDossierPage)row(label,value string){
	if p.y<90{return}
	value=firstNonEmptyText(strings.TrimSpace(value),"Não informado")
	p.c.b.WriteString("0.36 0.43 0.39 rg\n");p.c.text(40,p.y,7.5,true,label)
	p.c.b.WriteString("0.10 0.17 0.13 rg\n")
	lines:=pdfWrap(value,72)
	if len(lines)>4{lines=append(lines[:4],"...")}
	for _,line:=range lines{p.c.text(175,p.y,8,false,line);p.y-=10}
	p.y-=5
}

func (p *projectDossierPage)note(title,value string){
	if p.y<130{return}
	h:=75.0
	p.c.b.WriteString("0.95 0.97 0.95 rg\n");p.c.rect(38,p.y-h+10,519,h,true)
	p.c.b.WriteString("0.08 0.30 0.21 rg\n");p.c.text(48,p.y-6,8,true,title)
	p.c.b.WriteString("0.22 0.29 0.25 rg\n")
	y:=p.y-22
	for _,line:=range pdfWrap(value,100){p.c.text(48,y,7.5,false,line);y-=10;if y<p.y-h+18{break}}
	p.y-=h+4
}

func (p *projectDossierPage)footer(v string){
	p.c.b.WriteString("0.42 0.48 0.44 rg\n");p.c.text(38,42,6.5,false,v)
}
func (p *projectDossierPage)content()string{return p.c.b.String()}

func assembleProjectDossierPDF(contents []string) []byte {
	if len(contents)==0{return nil}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	n:=len(contents)
	font1:=3+n
	font2:=font1+1
	firstContent:=font2+1
	size:=firstContent+n
	offsets:=make([]int,size)
	writeObj:=func(id int,body string){offsets[id]=out.Len();fmt.Fprintf(&out,"%d 0 obj\n%s\nendobj\n",id,body)}
	writeObj(1,"<< /Type /Catalog /Pages 2 0 R >>")
	kids:=make([]string,n)
	for i:=0;i<n;i++{kids[i]=fmt.Sprintf("%d 0 R",3+i)}
	writeObj(2,fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>",strings.Join(kids," "),n))
	for i:=0;i<n;i++ {
		writeObj(3+i,fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 %d 0 R /F2 %d 0 R >> >> /Contents %d 0 R >>",font1,font2,firstContent+i))
	}
	writeObj(font1,"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	writeObj(font2,"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	for i,content:=range contents{writeObj(firstContent+i,fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream",len(content),content))}
	xref:=out.Len()
	fmt.Fprintf(&out,"xref\n0 %d\n0000000000 65535 f \n",size)
	for i:=1;i<size;i++{fmt.Fprintf(&out,"%010d 00000 n \n",offsets[i])}
	fmt.Fprintf(&out,"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",size,xref)
	return out.Bytes()
}
