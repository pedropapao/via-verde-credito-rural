package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

func nowBR() time.Time {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil { return time.Now() }
	return time.Now().In(loc)
}

func (a *App) dashboard360(w http.ResponseWriter, r *http.Request) {
	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &projects)
	a.enrichProjects(r, projects)
	var tasks []ProjectTask
	_ = a.sb.Select(r.Context(), "project_tasks", "select=*&status=neq.Concluída&"+order("due_at", false), &tasks)
	var reports []DailyReport
	_ = a.sb.Select(r.Context(), "daily_reports", "select=*&"+order("date", true), &reports)
	stats := map[string]int{"elaboration":0,"docs":0,"bank":0,"approved":0,"rejected":0,"contracted":0}
	total, contractedValue := 0.0, 0.0
	for _, p := range projects {
		total += p.FinancedValue
		s := strings.ToLower(p.Status+" "+p.Phase)
		switch {
		case strings.Contains(s,"reprov") || strings.Contains(s,"negad"): stats["rejected"]++
		case strings.Contains(s,"contrat"): stats["contracted"]++; contractedValue += p.FinancedValue
		case strings.Contains(s,"aprov"): stats["approved"]++
		case strings.Contains(s,"enviado") || strings.Contains(s,"banco") || strings.Contains(s,"análise"): stats["bank"]++
		case strings.Contains(s,"document") || strings.Contains(s,"pend"): stats["docs"]++
		default: stats["elaboration"]++
		}
	}
	today := nowBR().Format("2006-01-02")
	todayCount, todayValue := 0, 0.0
	for _, d := range reports { if strings.HasPrefix(d.Date,today) { todayCount++; todayValue += d.Value } }
	due := make([]ProjectTask,0,10); overdue:=0
	for _, t := range tasks {
		if t.DueAt != "" && t.DueAt < today { overdue++ }
		if len(due)<10 { due=append(due,t) }
	}
	data := map[string]any{"Projects":firstProjects(projects,8),"TotalFinanced":total,"ContractedValue":contractedValue,"Stats":stats,"Tasks":due,"Overdue":overdue,"TodayCount":todayCount,"TodayValue":todayValue,"RecentReports":firstReports(reports,6)}
	a.render(w,r,"dashboard",ViewData{Title:"Visão geral",Data:data})
}
func firstProjects(in []Project,n int) []Project { if len(in)<=n{return in}; return in[:n] }

func (a *App) projects360(w http.ResponseWriter, r *http.Request) {
	var rows []Project
	_ = a.sb.Select(r.Context(),"projects","select=*&"+order("updated_at",true),&rows)
	a.enrichProjects(r,rows)
	q:=strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status:=strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	bank:=strings.ToLower(strings.TrimSpace(r.URL.Query().Get("bank")))
	mod:=strings.ToLower(strings.TrimSpace(r.URL.Query().Get("modality")))
	out:=rows[:0]
	for _,p:=range rows{
		hay:=strings.ToLower(p.Title+" "+p.Activity+" "+p.ClientName+" "+p.PropertyName+" "+p.Bank+" "+p.Modality+" "+p.Status)
		if q!=""&&!strings.Contains(hay,q){continue}; if status!=""&&!strings.Contains(strings.ToLower(p.Status),status){continue}; if bank!=""&&!strings.Contains(strings.ToLower(p.Bank),bank){continue}; if mod!=""&&!strings.Contains(strings.ToLower(p.Modality),mod){continue}; out=append(out,p)
	}
	a.render(w,r,"projects",ViewData{Title:"Projetos",Data:map[string]any{"Projects":out,"Q":r.URL.Query().Get("q"),"Status":r.URL.Query().Get("status"),"Bank":r.URL.Query().Get("bank"),"Modality":r.URL.Query().Get("modality")}})
}

func (a *App) project360(w http.ResponseWriter, r *http.Request) {
	p,err:=a.getProject(r,r.PathValue("id")); if err!=nil{http.NotFound(w,r);return}
	v:=Project360View{Project:p,ViabilityLabel:"A calcular"}
	var cs []Client; _=a.sb.Select(r.Context(),"clients",eq("id",p.ClientID)+"&limit=1",&cs); if len(cs)>0{v.Client=cs[0]}
	if p.PropertyID!="" { var ps []Property360; _=a.sb.Select(r.Context(),"properties",eq("id",p.PropertyID)+"&limit=1",&ps); if len(ps)>0{v.Property=ps[0]} }
	migrationReady:=true
	if err:=a.sb.Select(r.Context(),"project_budget_items",eq("project_id",p.ID)+"&"+order("created_at",false),&v.Budget);err!=nil{migrationReady=false}
	_ = a.sb.Select(r.Context(),"project_revenues",eq("project_id",p.ID)+"&"+order("created_at",false),&v.Revenues)
	var ms []ProjectMetrics; _=a.sb.Select(r.Context(),"project_metrics",eq("project_id",p.ID)+"&limit=1",&ms); if len(ms)>0{v.Metrics=ms[0]}
	_ = a.sb.Select(r.Context(),"project_tasks",eq("project_id",p.ID)+"&"+order("due_at",false),&v.Tasks)
	_ = a.sb.Select(r.Context(),"project_history",eq("project_id",p.ID)+"&"+order("event_date",true),&v.History)
	_ = a.sb.Select(r.Context(),"project_checklist",eq("project_id",p.ID)+"&"+order("created_at",false),&v.Checklist)
	_ = a.sb.Select(r.Context(),"project_sections",eq("project_id",p.ID)+"&"+order("section_key",false),&v.Sections)
	_ = a.sb.Select(r.Context(),"project_files","select=id,project_id,name,storage_path,content_type,size_bytes,uploaded_by,created_at,document_type,version_label,valid_until,is_current&"+eq("project_id",p.ID)+"&"+order("created_at",true),&v.Files)
	if migrationReady && len(v.Checklist)==0 { a.seedChecklist360(r,p); _=a.sb.Select(r.Context(),"project_checklist",eq("project_id",p.ID)+"&"+order("created_at",false),&v.Checklist) }
	calc360(&v)
	if svg,area,perim:=a.kmlPreview(r,v.Files); svg!=""{v.MapSVG=svg;v.KMLAreaHa=area;v.KMLPerimeterM=perim}
	a.render(w,r,"project_detail",ViewData{Title:p.Title,Data:map[string]any{"V":v,"MigrationReady":migrationReady}})
}

func calc360(v *Project360View){
	for _,b:=range v.Budget { total:=b.TotalValue;if total==0{total=b.Quantity*b.UnitValue}; v.BudgetTotal+=total; v.BudgetOwn+=b.OwnResources; fin:=b.FinancedValue;if fin==0{fin=math.Max(0,total-b.OwnResources)};v.BudgetFinanced+=fin }
	for _,x:=range v.Revenues { total:=x.TotalValue;if total==0{total=x.Quantity*x.UnitPrice};v.RevenueTotal+=total }
	v.OperatingMargin=v.RevenueTotal-v.BudgetTotal+v.Metrics.OtherIncome-v.Metrics.OtherDebts
	pay:=v.Metrics.AnnualPayment;if pay<=0 && v.Project.FinancedValue>0 && v.Project.TermMonths>0 { years:=math.Max(1,float64(v.Project.TermMonths)/12); pay=(v.Project.FinancedValue/years)+(v.Project.FinancedValue*v.Project.InterestRate/100) }
	if pay>0{v.DebtCoverage=v.OperatingMargin/pay}
	if v.Metrics.AreaHa>0{v.CostPerHa=v.BudgetTotal/v.Metrics.AreaHa;v.RevenuePerHa=v.RevenueTotal/v.Metrics.AreaHa}
	if v.Metrics.AnimalCount>0{v.CostPerAnimal=v.BudgetTotal/float64(v.Metrics.AnimalCount);v.RevenuePerAnimal=v.RevenueTotal/float64(v.Metrics.AnimalCount); if v.Metrics.FinalWeightKg>0 { v.ProjectedArrobas=(v.Metrics.FinalWeightKg*v.Metrics.CarcassYieldPct/100/15)*float64(v.Metrics.AnimalCount) }}
	if v.DebtCoverage>=1.30{v.ViabilityLabel="VIÁVEL"}else if v.DebtCoverage>=1{v.ViabilityLabel="ATENÇÃO"}else if pay>0{v.ViabilityLabel="INVIÁVEL / REVISAR"}
	completedDocs:=0; requiredDocs:=0; for _,c:=range v.Checklist{if c.Required{requiredDocs++;if strings.EqualFold(c.Status,"Concluído")||strings.EqualFold(c.Status,"OK")||strings.EqualFold(c.Status,"Recebido"){completedDocs++}}}
	if requiredDocs>0{v.DocCompleteness=completedDocs*100/requiredDocs}else if len(v.Files)>0{v.DocCompleteness=50}
	techPoints:=0;if v.Project.TechnicalSummary!=""{techPoints++};if v.Metrics.AreaHa>0||v.Metrics.AnimalCount>0{techPoints++};if len(v.Sections)>0{techPoints++};if v.Property.ID!=""{techPoints++};v.TechCompleteness=techPoints*25
	finPoints:=0;if len(v.Budget)>0{finPoints++};if len(v.Revenues)>0{finPoints++};if v.Project.FinancedValue>0{finPoints++};if v.DebtCoverage>0{finPoints++};v.FinCompleteness=finPoints*25
	v.Completeness=(v.DocCompleteness+v.TechCompleteness+v.FinCompleteness)/3
	today:=nowBR().Format("2006-01-02");for _,t:=range v.Tasks{if !strings.EqualFold(t.Status,"Concluída"){v.PendingCount++;if t.DueAt!=""&&t.DueAt<today{v.OverdueCount++}}}
}

func (a *App) seedChecklist360(r *http.Request,p Project){
	items:=defaultChecklist360(p); for _,c:=range items{c.ProjectID=p.ID;_ = a.sb.Insert(r.Context(),"project_checklist",c,nil)}
}
func defaultChecklist360(p Project) []ChecklistItem{
	base:=[]ChecklistItem{{Code:"DOC_ID",DocumentName:"Documento de identificação / CPF ou CNPJ",Required:true,Status:"Pendente",SourceRule:"Cadastro"},{Code:"CAR",DocumentName:"Recibo do Cadastro Ambiental Rural (CAR)",Required:true,Status:"Pendente",SourceRule:"MCR 2-1"},{Code:"AREA",DocumentName:"Coordenadas geodésicas / KML da área financiada",Required:true,Status:"Pendente",SourceRule:"MCR 2-1"},{Code:"ORC",DocumentName:"Orçamento detalhado e fontes de preços",Required:true,Status:"Pendente",SourceRule:"MCR 2-2"},{Code:"MATRICULA",DocumentName:"Matrícula / comprovação da relação com o imóvel",Required:true,Status:"Pendente",SourceRule:"MCR 1-2 e Roteiro B"},{Code:"CCIR",DocumentName:"CCIR",Required:true,Status:"Pendente",SourceRule:"Dossiê cadastral"},{Code:"ITR",DocumentName:"ITR",Required:true,Status:"Pendente",SourceRule:"Dossiê cadastral"}}
	x:=strings.ToLower(p.Modality+" "+p.Activity)
	if strings.Contains(x,"agr")||strings.Contains(x,"soja")||strings.Contains(x,"milho")||strings.Contains(x,"sorgo")||strings.Contains(x,"café"){base=append(base,ChecklistItem{Code:"ZARC",DocumentName:"Conferência ZARC / janela de plantio",Required:true,Status:"Pendente",SourceRule:"MCR 2-1"},ChecklistItem{Code:"SOLO",DocumentName:"Análise de solo / recomendação técnica quando aplicável",Required:false,Status:"Pendente",SourceRule:"Roteiro B 4.4.2"})}
	if strings.Contains(x,"pec")||strings.Contains(x,"bovin")||strings.Contains(x,"gado")||strings.Contains(x,"animal"){base=append(base,ChecklistItem{Code:"SANITARIA",DocumentName:"Ficha sanitária do rebanho",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-20"});if strings.Contains(x,"aquisição")||strings.Contains(x,"aquisicao"){base=append(base,ChecklistItem{Code:"NF",DocumentName:"Nota fiscal de aquisição",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-20"},ChecklistItem{Code:"GTA",DocumentName:"Guia de Trânsito Animal (GTA)",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-20"},ChecklistItem{Code:"OFERTA",DocumentName:"Carta de oferta / identificação do vendedor",Required:true,Status:"Pendente",SourceRule:"Procedimento operacional"})}}
	return base
}

func (a *App) budgetAdd360(w http.ResponseWriter,r *http.Request){ if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");q:=parseFloat(r.FormValue("quantity"));u:=parseFloat(r.FormValue("unit_value"));tot:=q*u;own:=parseFloat(r.FormValue("own_resources"));b:=BudgetItem{ProjectID:id,Category:r.FormValue("category"),Item:strings.TrimSpace(r.FormValue("item")),AcquisitionStart:r.FormValue("acquisition_start"),AcquisitionEnd:r.FormValue("acquisition_end"),UseStart:r.FormValue("use_start"),UseEnd:r.FormValue("use_end"),Unit:r.FormValue("unit"),Quantity:q,UnitValue:u,TotalValue:tot,OwnResources:own,FinancedValue:math.Max(0,tot-own),PriceSource:r.FormValue("price_source"),Notes:r.FormValue("notes")};if b.Item==""{http.Error(w,"Informe o item",400);return};if err:=a.sb.Insert(r.Context(),"project_budget_items",b,nil);err!=nil{http.Error(w,"Execute supabase/03_upgrade_dossie_360.sql antes de usar o orçamento.",500);return};a.addHistory360(r,id,"Orçamento","Item incluído",b.Item);redirectProject(w,r,id,"Item de orçamento adicionado.") }
func (a *App) budgetDelete360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");_ = a.sb.Delete(r.Context(),"project_budget_items",eq("id",r.PathValue("item")));redirectProject(w,r,id,"Item removido.")}
func (a *App) revenueAdd360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");q:=parseFloat(r.FormValue("quantity"));p:=parseFloat(r.FormValue("unit_price"));x:=ProjectRevenue{ProjectID:id,Product:r.FormValue("product"),Period:r.FormValue("period"),Unit:r.FormValue("unit"),Quantity:q,UnitPrice:p,TotalValue:q*p,Source:r.FormValue("source"),Notes:r.FormValue("notes")};if x.Product==""{http.Error(w,"Informe o produto",400);return};if err:=a.sb.Insert(r.Context(),"project_revenues",x,nil);err!=nil{http.Error(w,"Migração 03 ainda não executada.",500);return};a.addHistory360(r,id,"Receita","Receita incluída",x.Product);redirectProject(w,r,id,"Receita adicionada.")}
func (a *App) revenueDelete360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");_ = a.sb.Delete(r.Context(),"project_revenues",eq("id",r.PathValue("item")));redirectProject(w,r,id,"Receita removida.")}

func (a *App) metricsSave360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");m:=ProjectMetrics{ProjectID:id,ActivityType:r.FormValue("activity_type"),AreaHa:parseFloat(r.FormValue("area_ha")),Productivity:parseFloat(r.FormValue("productivity")),ProductivityUnit:r.FormValue("productivity_unit"),AnimalCount:parseInt(r.FormValue("animal_count")),InitialWeightKg:parseFloat(r.FormValue("initial_weight_kg")),FinalWeightKg:parseFloat(r.FormValue("final_weight_kg")),GMDKgDay:parseFloat(r.FormValue("gmd_kg_day")),CycleDays:parseInt(r.FormValue("cycle_days")),CarcassYieldPct:parseFloat(r.FormValue("carcass_yield_pct")),MortalityPct:parseFloat(r.FormValue("mortality_pct")),StockingRate:parseFloat(r.FormValue("stocking_rate")),OtherDebts:parseFloat(r.FormValue("other_debts")),OtherIncome:parseFloat(r.FormValue("other_income")),AnnualPayment:parseFloat(r.FormValue("annual_payment")),Notes:r.FormValue("notes"),UpdatedAt:time.Now().UTC().Format(time.RFC3339)};var old []ProjectMetrics;_ = a.sb.Select(r.Context(),"project_metrics",eq("project_id",id)+"&limit=1",&old);var err error;if len(old)>0{err=a.sb.Update(r.Context(),"project_metrics",eq("project_id",id),m,nil)}else{err=a.sb.Insert(r.Context(),"project_metrics",m,nil)};if err!=nil{http.Error(w,"Migração 03 ainda não executada.",500);return};a.addHistory360(r,id,"Técnico","Indicadores atualizados",m.ActivityType);redirectProject(w,r,id,"Indicadores atualizados.")}

func (a *App) taskAdd360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");t:=ProjectTask{ProjectID:id,Title:strings.TrimSpace(r.FormValue("title")),Category:r.FormValue("category"),Responsible:r.FormValue("responsible"),RequestedAt:r.FormValue("requested_at"),DueAt:r.FormValue("due_at"),Status:"Pendente",Priority:defaultString(r.FormValue("priority"),"Normal"),Notes:r.FormValue("notes")};if t.Title==""{http.Error(w,"Informe a pendência",400);return};if err:=a.sb.Insert(r.Context(),"project_tasks",t,nil);err!=nil{http.Error(w,"Migração 03 ainda não executada.",500);return};a.addHistory360(r,id,"Pendência","Pendência criada",t.Title);redirectProject(w,r,id,"Pendência criada.")}
func (a *App) taskToggle360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");vals:=map[string]any{"status":"Concluída","completed_at":time.Now().UTC().Format(time.RFC3339)};_ = a.sb.Update(r.Context(),"project_tasks",eq("id",r.PathValue("task")),vals,nil);a.addHistory360(r,id,"Pendência","Pendência concluída","");redirectProject(w,r,id,"Pendência concluída.")}

func (a *App) checklistUpdate360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");vals:=map[string]any{"status":r.FormValue("status"),"valid_until":nullIfEmpty(r.FormValue("valid_until")),"notes":r.FormValue("notes")};_ = a.sb.Update(r.Context(),"project_checklist",eq("id",r.PathValue("check")),vals,nil);redirectProject(w,r,id,"Checklist atualizado.")}
func (a *App) sectionSave360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");key:=r.FormValue("section_key");title:=r.FormValue("section_title");content:=r.FormValue("content");status:=r.FormValue("status");var old []ProjectSection;_ = a.sb.Select(r.Context(),"project_sections",eq("project_id",id)+"&"+eq("section_key",key)+"&limit=1",&old);row:=ProjectSection{ProjectID:id,SectionKey:key,SectionTitle:title,Content:content,Status:status,UpdatedAt:time.Now().UTC().Format(time.RFC3339)};if len(old)>0{_ = a.sb.Update(r.Context(),"project_sections",eq("id",old[0].ID),row,nil)}else{_ = a.sb.Insert(r.Context(),"project_sections",row,nil)};a.addHistory360(r,id,"Roteiro B","Seção atualizada",title);redirectProject(w,r,id,"Memória técnica salva.")}
func (a *App) historyAdd360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");a.addHistory360(r,id,r.FormValue("event_type"),r.FormValue("title"),r.FormValue("details"));redirectProject(w,r,id,"Evento registrado.")}
func (a *App) addHistory360(r *http.Request,id,typ,title,details string){u:=userFromContext(r.Context());uid:="";if u!=nil{uid=u.ID};_ = a.sb.Insert(r.Context(),"project_history",ProjectHistory{ProjectID:id,EventType:typ,Title:defaultString(title,"Atualização"),Details:details,UserID:uid},nil)}

func (a *App) statusUpdate360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");p,err:=a.getProject(r,id);if err!=nil{http.NotFound(w,r);return};newStatus:=r.FormValue("status");newPhase:=r.FormValue("phase");_ = a.sb.Update(r.Context(),"projects",eq("id",id),map[string]any{"status":newStatus,"phase":newPhase,"updated_at":time.Now().UTC().Format(time.RFC3339)},nil);a.addHistory360(r,id,"Status","Status: "+newStatus,newPhase);if shouldCreateDaily(p.Status,newStatus){_ = a.sb.Insert(r.Context(),"daily_reports",DailyReport{Date:nowBR().Format("2006-01-02"),Value:p.FinancedValue,Bank:p.Bank,Producer:p.ClientName,ProjectType:p.Modality+" · "+p.Activity,Situation:newStatus},nil)};redirectProject(w,r,id,"Status atualizado e histórico registrado.")}
func shouldCreateDaily(old,new string)bool{o:=strings.ToLower(old);n:=strings.ToLower(new);if o==n{return false};for _,x:=range []string{"enviado","aprov","contrat","reprov"}{if strings.Contains(n,x){return true}};return false}

func (a *App) fileUpload360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");r.Body=http.MaxBytesReader(w,r.Body,45<<20);if err:=r.ParseMultipartForm(45<<20);err!=nil{http.Error(w,"Arquivo muito grande",400);return};f,h,err:=r.FormFile("file");if err!=nil{http.Error(w,"Selecione o arquivo",400);return};defer f.Close();name:=safeFilename(h.Filename);ct:=h.Header.Get("Content-Type");if ct==""{ct="application/octet-stream"};obj:=fmt.Sprintf("projects/%s/%d-%s",id,time.Now().UnixNano(),name);if err:=a.sb.Upload(r.Context(),a.cfg.StorageBucket,obj,ct,io.LimitReader(f,45<<20));err!=nil{http.Error(w,"Falha no upload: "+err.Error(),500);return};u:=userFromContext(r.Context());row:=ProjectFile360{ProjectID:id,Name:name,StoragePath:obj,ContentType:ct,SizeBytes:h.Size,UploadedBy:u.ID,DocumentType:r.FormValue("document_type"),VersionLabel:r.FormValue("version_label"),ValidUntil:r.FormValue("valid_until"),IsCurrent:true};if err:=a.sb.Insert(r.Context(),"project_files",row,nil);err!=nil{_ = a.sb.DeleteObject(r.Context(),a.cfg.StorageBucket,obj);http.Error(w,"Migração 03 ainda não executada.",500);return};a.addHistory360(r,id,"Documento","Documento anexado",name);redirectProject(w,r,id,"Documento anexado e versionado.")}

func redirectProject(w http.ResponseWriter,r *http.Request,id,msg string){http.Redirect(w,r,"/projects/"+id+"?ok="+url.QueryEscape(msg),http.StatusSeeOther)}

func (a *App) exportProject360(w http.ResponseWriter,r *http.Request){id:=r.PathValue("id");p,err:=a.getProject(r,id);if err!=nil{http.NotFound(w,r);return};var files []ProjectFile360;_ = a.sb.Select(r.Context(),"project_files","select=*&"+eq("project_id",id),&files);var budget []BudgetItem;_ = a.sb.Select(r.Context(),"project_budget_items",eq("project_id",id),&budget);var rev []ProjectRevenue;_ = a.sb.Select(r.Context(),"project_revenues",eq("project_id",id),&rev);var tasks []ProjectTask;_ = a.sb.Select(r.Context(),"project_tasks",eq("project_id",id),&tasks);var checks []ChecklistItem;_ = a.sb.Select(r.Context(),"project_checklist",eq("project_id",id),&checks);var hist []ProjectHistory;_ = a.sb.Select(r.Context(),"project_history",eq("project_id",id),&hist);w.Header().Set("Content-Type","application/zip");w.Header().Set("Content-Disposition",`attachment; filename="dossie-via-verde.zip"`);zw:=zip.NewWriter(w);defer zw.Close();manifest:=map[string]any{"project":p,"budget":budget,"revenues":rev,"tasks":tasks,"checklist":checks,"history":hist,"exported_at":time.Now().UTC()};b,_:=json.MarshalIndent(manifest,"","  ");f,_:=zw.Create("dados-projeto.json");_,_=f.Write(b);csvf,_:=zw.Create("orcamento.csv");cw:=csv.NewWriter(csvf);_ = cw.Write([]string{"Categoria","Item","Unidade","Quantidade","Valor unitário","Total","Recursos próprios","Financiado","Fonte"});for _,x:=range budget{_ = cw.Write([]string{x.Category,x.Item,x.Unit,fmt.Sprint(x.Quantity),fmt.Sprint(x.UnitValue),fmt.Sprint(x.TotalValue),fmt.Sprint(x.OwnResources),fmt.Sprint(x.FinancedValue),x.PriceSource})};cw.Flush();for _,pf:=range files{data,_,e:=a.sb.Download(r.Context(),a.cfg.StorageBucket,pf.StoragePath);if e!=nil{continue};zf,e:=zw.Create("documentos/"+safeFilename(pf.Name));if e==nil{_,_=zf.Write(data)}}}

func (a *App) kmlPreview(r *http.Request,files []ProjectFile360)(string,float64,float64){var target *ProjectFile360;for i:=range files{if strings.HasSuffix(strings.ToLower(files[i].Name),".kml")&&files[i].IsCurrent{target=&files[i];break}};if target==nil{return "",0,0};b,_,err:=a.sb.Download(r.Context(),a.cfg.StorageBucket,target.StoragePath);if err!=nil{return "",0,0};pts:=extractKMLCoords(b);if len(pts)<3{return "",0,0};area,perim:=geoAreaPerimeter(pts);return svgPolygon(pts),area,perim}
type geoPoint struct{Lon,Lat float64}
func extractKMLCoords(b []byte)[]geoPoint{dec:=xml.NewDecoder(bytes.NewReader(b));var out []geoPoint;for{t,err:=dec.Token();if err!=nil{break};se,ok:=t.(xml.StartElement);if !ok||se.Name.Local!="coordinates"{continue};var s string;if dec.DecodeElement(&s,&se)!=nil{continue};for _,part:=range strings.Fields(s){v:=strings.Split(part,",");if len(v)<2{continue};lon,e1:=strconv.ParseFloat(v[0],64);lat,e2:=strconv.ParseFloat(v[1],64);if e1==nil&&e2==nil{out=append(out,geoPoint{lon,lat})}}};return out}
func geoAreaPerimeter(p []geoPoint)(float64,float64){lat0:=0.0;for _,x:=range p{lat0+=x.Lat};lat0/=float64(len(p));cos:=math.Cos(lat0*math.Pi/180);xy:=make([][2]float64,len(p));for i,x:=range p{xy[i]=[2]float64{x.Lon*111320*cos,x.Lat*110540}};area:=0.0;per:=0.0;for i:=range xy{j:=(i+1)%len(xy);area+=xy[i][0]*xy[j][1]-xy[j][0]*xy[i][1];dx:=xy[j][0]-xy[i][0];dy:=xy[j][1]-xy[i][1];per+=math.Hypot(dx,dy)};return math.Abs(area)/2/10000,per}
func svgPolygon(p []geoPoint)string{minx,maxx,miny,maxy:=p[0].Lon,p[0].Lon,p[0].Lat,p[0].Lat;for _,x:=range p{minx=math.Min(minx,x.Lon);maxx=math.Max(maxx,x.Lon);miny=math.Min(miny,x.Lat);maxy=math.Max(maxy,x.Lat)};dx:=maxx-minx;dy:=maxy-miny;if dx==0{dx=1};if dy==0{dy=1};var sb strings.Builder;for _,x:=range p{px:=20+(x.Lon-minx)/dx*560;py:=320-(x.Lat-miny)/dy*280;fmt.Fprintf(&sb,"%.1f,%.1f ",px,py)};return `<svg viewBox="0 0 600 340" role="img" aria-label="Prévia do perímetro KML"><rect width="600" height="340" rx="18" fill="#eef4ef"/><polygon points="`+sb.String()+`" fill="#18865722" stroke="#188657" stroke-width="4"/><text x="22" y="32" fill="#315448" font-size="15">Prévia geométrica do KML · não substitui conferência cartográfica</text></svg>`}

func (a *App) propertyUpdate360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");vals:=map[string]any{"latitude":parseFloat(r.FormValue("latitude")),"longitude":parseFloat(r.FormValue("longitude")),"water_info":r.FormValue("water_info"),"environmental_info":r.FormValue("environmental_info"),"access_info":r.FormValue("access_info"),"owner_name":r.FormValue("owner_name"),"car_status":r.FormValue("car_status")};if err:=a.sb.Update(r.Context(),"properties",eq("id",id),vals,nil);err!=nil{http.Error(w,"Migração 03 ainda não executada.",500);return};http.Redirect(w,r,"/properties?ok="+url.QueryEscape("Propriedade atualizada."),303)}

func sortTasks360(ts []ProjectTask){sort.SliceStable(ts,func(i,j int)bool{return ts[i].DueAt<ts[j].DueAt})}
