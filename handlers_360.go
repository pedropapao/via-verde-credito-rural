package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"math"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func nowBR360() time.Time {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil { return time.Now() }
	return time.Now().In(loc)
}

func (a *App) routes360() http.Handler {
	mux := http.NewServeMux()
	staticFS, _ := fs.Sub(webFS, "web/static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /login", a.loginGet)
	mux.HandleFunc("POST /login", a.loginPost)
	mux.HandleFunc("GET /invite/{token}", a.inviteGet)
	mux.HandleFunc("POST /invite/{token}", a.invitePost)

	mux.Handle("GET /", a.withAuth(http.HandlerFunc(a.dashboard360)))
	mux.Handle("POST /logout", a.withAuth(http.HandlerFunc(a.logout)))

	mux.Handle("GET /projects", a.withAuth(http.HandlerFunc(a.projectsList360)))
	mux.Handle("GET /projects/new", a.ownerOnly(http.HandlerFunc(a.projectNewGet)))
	mux.Handle("POST /projects/new", a.ownerOnly(http.HandlerFunc(a.projectNewPost360)))
	mux.Handle("GET /projects/{id}", a.withAuth(http.HandlerFunc(a.projectDetail360)))
	mux.Handle("GET /projects/{id}/edit", a.ownerOnly(http.HandlerFunc(a.projectEditGet)))
	mux.Handle("POST /projects/{id}/edit", a.ownerOnly(http.HandlerFunc(a.projectEditPost360)))
	mux.Handle("POST /projects/{id}/files", a.ownerOnly(http.HandlerFunc(a.projectFileUpload360)))
	mux.Handle("POST /files/{id}/delete", a.ownerOnly(http.HandlerFunc(a.fileDelete360)))
	mux.Handle("GET /files/{id}/download", a.withAuth(http.HandlerFunc(a.fileDownload)))
	mux.Handle("GET /projects/{id}/export.zip", a.withAuth(http.HandlerFunc(a.projectExport360)))

	mux.Handle("POST /projects/{id}/budget", a.ownerOnly(http.HandlerFunc(a.budgetCreate360)))
	mux.Handle("POST /projects/{id}/budget/{item}/delete", a.ownerOnly(http.HandlerFunc(a.budgetDelete360)))
	mux.Handle("POST /projects/{id}/revenues", a.ownerOnly(http.HandlerFunc(a.revenueCreate360)))
	mux.Handle("POST /projects/{id}/revenues/{item}/delete", a.ownerOnly(http.HandlerFunc(a.revenueDelete360)))
	mux.Handle("POST /projects/{id}/metrics", a.ownerOnly(http.HandlerFunc(a.metricsSave360)))
	mux.Handle("POST /projects/{id}/tasks", a.ownerOnly(http.HandlerFunc(a.taskCreate360)))
	mux.Handle("POST /projects/{id}/tasks/{item}/toggle", a.ownerOnly(http.HandlerFunc(a.taskToggle360)))
	mux.Handle("POST /projects/{id}/tasks/{item}/delete", a.ownerOnly(http.HandlerFunc(a.taskDelete360)))
	mux.Handle("POST /projects/{id}/checklist/{item}", a.ownerOnly(http.HandlerFunc(a.checklistSave360)))
	mux.Handle("POST /projects/{id}/sections/{key}", a.ownerOnly(http.HandlerFunc(a.sectionSave360)))
	mux.Handle("POST /projects/{id}/history", a.ownerOnly(http.HandlerFunc(a.historyCreate360)))

	mux.Handle("GET /clients", a.withAuth(http.HandlerFunc(a.clientsList)))
	mux.Handle("POST /clients", a.ownerOnly(http.HandlerFunc(a.clientCreate)))
	mux.Handle("GET /properties", a.withAuth(http.HandlerFunc(a.propertiesList360)))
	mux.Handle("POST /properties", a.ownerOnly(http.HandlerFunc(a.propertyCreate360)))
	mux.Handle("POST /properties/{id}/files", a.ownerOnly(http.HandlerFunc(a.propertyFileUpload360)))
	mux.Handle("GET /property-files/{id}/download", a.withAuth(http.HandlerFunc(a.propertyFileDownload360)))
	mux.Handle("POST /property-files/{id}/delete", a.ownerOnly(http.HandlerFunc(a.propertyFileDelete360)))
	mux.Handle("GET /documents", a.withAuth(http.HandlerFunc(a.documentsList)))

	mux.Handle("GET /reports/daily", a.withAuth(http.HandlerFunc(a.dailyList)))
	mux.Handle("POST /reports/daily", a.ownerOnly(http.HandlerFunc(a.dailyCreate)))
	mux.Handle("GET /reports/daily/export.csv", a.withAuth(http.HandlerFunc(a.dailyExportCSV)))
	mux.Handle("GET /rules", a.withAuth(http.HandlerFunc(a.rulesList)))
	mux.Handle("POST /rules", a.ownerOnly(http.HandlerFunc(a.ruleCreate)))
	mux.Handle("GET /users", a.ownerOnly(http.HandlerFunc(a.usersList)))
	mux.Handle("POST /users/invite", a.ownerOnly(http.HandlerFunc(a.userInvite)))
	mux.Handle("POST /users/{id}/toggle", a.ownerOnly(http.HandlerFunc(a.userToggle)))
	mux.Handle("GET /settings", a.withAuth(http.HandlerFunc(a.settingsGet)))
	mux.Handle("POST /settings/password", a.withAuth(http.HandlerFunc(a.settingsPassword)))
	return securityHeaders(recoverer(logger(mux)))
}

func (a *App) dashboard360(w http.ResponseWriter, r *http.Request) {
	var projects []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &projects)
	a.enrichProjects(r, projects)
	var reports []DailyReport
	_ = a.sb.Select(r.Context(), "daily_reports", "select=*&"+order("date", true), &reports)
	var tasks []ProjectTask
	_ = a.sb.Select(r.Context(), "project_tasks", "select=*&"+order("due_at", false), &tasks)
	d := Dashboard360{Projects: firstProjects360(projects, 8), RecentReports: firstReports(reports, 6)}
	today := nowBR360().Format("2006-01-02")
	for _, p := range projects {
		d.TotalFinanced += p.FinancedValue
		s := strings.ToLower(p.Status)
		phase := strings.ToLower(p.Phase)
		if !strings.Contains(s, "reprov") && !strings.Contains(s, "conclu") { d.Active++ }
		if strings.Contains(s, "elabora") { d.Draft++ }
		if strings.Contains(phase, "document") || strings.Contains(phase, "pend") { d.WaitingDocs++ }
		if strings.Contains(s, "enviado") || strings.Contains(s, "análise") { d.Sent++ }
		if strings.Contains(s, "aprov") { d.Approved++ }
		if strings.Contains(s, "reprov") || strings.Contains(s, "negad") { d.Rejected++ }
		if strings.Contains(s, "contrat") { d.Contracted++; d.TotalContracted += p.FinancedValue }
	}
	for _, x := range reports {
		if strings.HasPrefix(x.Date, today) { d.TodayCount++; d.TodayValue += x.Value }
		d.ExpectedCommission += x.Commission
	}
	for _, t := range tasks {
		if taskDone360(t.Status) { continue }
		d.PendingTasks++
		if t.DueAt != "" && t.DueAt < today { d.OverdueTasks++ }
		if len(d.DueTasks) < 8 { d.DueTasks = append(d.DueTasks, t) }
	}
	a.render(w, r, "dashboard", ViewData{Title: "Visão geral", Data: d})
}

func firstProjects360(in []Project, n int) []Project { if len(in) <= n { return in }; return in[:n] }
func taskDone360(v string) bool { x:=strings.ToLower(v); return strings.Contains(x,"concl")||strings.Contains(x,"feito")||strings.Contains(x,"resolvido") }

func (a *App) projectsList360(w http.ResponseWriter, r *http.Request) {
	var rows []Project
	_ = a.sb.Select(r.Context(), "projects", "select=*&"+order("updated_at", true), &rows)
	a.enrichProjects(r, rows)
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status := strings.TrimSpace(r.URL.Query().Get("status")); bank := strings.TrimSpace(r.URL.Query().Get("bank")); modality := strings.TrimSpace(r.URL.Query().Get("modality"))
	filtered := make([]Project,0,len(rows)); banks:=map[string]bool{}; mods:=map[string]bool{}; statuses:=map[string]bool{}
	for _, p := range rows {
		if p.Bank!="" { banks[p.Bank]=true }; if p.Modality!="" { mods[p.Modality]=true }; if p.Status!="" { statuses[p.Status]=true }
		hay := strings.ToLower(p.Title+" "+p.ClientName+" "+p.PropertyName+" "+p.Activity+" "+p.Bank+" "+p.Modality+" "+p.Status)
		if q!="" && !strings.Contains(hay,q) { continue }
		if status!="" && p.Status!=status { continue }; if bank!="" && p.Bank!=bank { continue }; if modality!="" && p.Modality!=modality { continue }
		filtered=append(filtered,p)
	}
	data:=map[string]any{"Projects":filtered,"Q":r.URL.Query().Get("q"),"Status":status,"Bank":bank,"Modality":modality,"Banks":sortedKeys360(banks),"Modalities":sortedKeys360(mods),"Statuses":sortedKeys360(statuses)}
	a.render(w,r,"projects",ViewData{Title:"Projetos",Data:data})
}
func sortedKeys360(m map[string]bool) []string { out:=make([]string,0,len(m)); for k:=range m { out=append(out,k) }; sort.Strings(out); return out }

func projectFromForm360(r *http.Request) Project {
	return Project{ClientID:r.FormValue("client_id"),PropertyID:r.FormValue("property_id"),Title:strings.TrimSpace(r.FormValue("title")),Modality:r.FormValue("modality"),Activity:strings.TrimSpace(r.FormValue("activity")),Bank:strings.TrimSpace(r.FormValue("bank")),Program:strings.TrimSpace(r.FormValue("program")),TotalValue:parseFloat(r.FormValue("total_value")),FinancedValue:parseFloat(r.FormValue("financed_value")),OwnResources:parseFloat(r.FormValue("own_resources")),InterestRate:parseFloat(r.FormValue("interest_rate")),TermMonths:parseInt(r.FormValue("term_months")),GraceMonths:parseInt(r.FormValue("grace_months")),PaymentFrequency:strings.TrimSpace(r.FormValue("payment_frequency")),Status:r.FormValue("status"),Phase:strings.TrimSpace(r.FormValue("phase")),Responsible:strings.TrimSpace(r.FormValue("responsible")),SentAt:r.FormValue("sent_at"),ContractedAt:r.FormValue("contracted_at"),TechnicalSummary:strings.TrimSpace(r.FormValue("technical_summary")),Alerts:strings.TrimSpace(r.FormValue("alerts")),SensitiveNotes:strings.TrimSpace(r.FormValue("sensitive_notes"))}
}

func (a *App) projectNewPost360(w http.ResponseWriter,r *http.Request) {
	if !a.verifyCSRF(r) { http.Error(w,"Sessão inválida.",403); return }
	p:=projectFromForm360(r); if err:=validateProject(p); err!=nil { a.projectForm(w,r,p,false,err.Error()); return }; p.Status=defaultString(p.Status,"Em andamento")
	var out []Project; if err:=a.sb.Insert(r.Context(),"projects",p,&out); err!=nil||len(out)==0 { a.projectForm(w,r,p,false,"Não foi possível salvar o projeto. Se os campos avançados já estiverem preenchidos, execute a migração 03 no Supabase."); return }
	_ = a.ensureProjectDefaults360(r,out[0]); _ = a.addHistory360(r,out[0].ID,"Projeto","Projeto criado","Cadastro inicial do projeto")
	a.audit(r.Context(),userFromContext(r.Context()),"create","project",out[0].ID,p.Title)
	http.Redirect(w,r,"/projects/"+out[0].ID+"?ok="+url.QueryEscape("Projeto criado e checklist inicial preparado."),303)
}

func (a *App) projectEditPost360(w http.ResponseWriter,r *http.Request) {
	if !a.verifyCSRF(r) { http.Error(w,"Sessão inválida.",403); return }
	old,err:=a.getProject(r,r.PathValue("id")); if err!=nil {http.NotFound(w,r);return}
	p:=projectFromForm360(r); p.ID=old.ID; if err:=validateProject(p); err!=nil {a.projectForm(w,r,p,true,err.Error());return}
	vals:=map[string]any{"client_id":p.ClientID,"property_id":nullIfEmpty(p.PropertyID),"title":p.Title,"modality":p.Modality,"activity":p.Activity,"bank":p.Bank,"program":p.Program,"total_value":p.TotalValue,"financed_value":p.FinancedValue,"own_resources":p.OwnResources,"interest_rate":p.InterestRate,"term_months":p.TermMonths,"grace_months":p.GraceMonths,"payment_frequency":p.PaymentFrequency,"status":p.Status,"phase":p.Phase,"responsible":p.Responsible,"sent_at":nullIfEmpty(p.SentAt),"contracted_at":nullIfEmpty(p.ContractedAt),"technical_summary":p.TechnicalSummary,"alerts":p.Alerts,"sensitive_notes":p.SensitiveNotes,"updated_at":time.Now().UTC().Format(time.RFC3339)}
	if err:=a.sb.Update(r.Context(),"projects",eq("id",p.ID),vals,nil); err!=nil { a.projectForm(w,r,p,true,"Não foi possível atualizar. Execute supabase/03_upgrade_dossie_360.sql se ainda não fez isso."); return }
	p.ClientName=old.ClientName
	if old.Status!=p.Status { _ = a.addHistory360(r,p.ID,"Status","Status alterado",old.Status+" → "+p.Status); a.autoDailyReport360(r,p) }
	_ = a.ensureProjectDefaults360(r,p); a.audit(r.Context(),userFromContext(r.Context()),"update","project",p.ID,p.Title)
	http.Redirect(w,r,"/projects/"+p.ID+"?ok="+url.QueryEscape("Projeto atualizado."),303)
}

func (a *App) autoDailyReport360(r *http.Request,p Project) {
	s:=strings.ToLower(p.Status); if !(strings.Contains(s,"enviado")||strings.Contains(s,"aprov")||strings.Contains(s,"contrat")||strings.Contains(s,"reprov")) {return}
	producer:=p.ClientName; if producer=="" { var cs []Client; _=a.sb.Select(r.Context(),"clients",eq("id",p.ClientID)+"&limit=1",&cs); if len(cs)>0 {producer=cs[0].Name} }
	d:=DailyReport{ProjectID:p.ID,EventType:"status",Date:nowBR360().Format("2006-01-02"),Value:p.FinancedValue,Bank:p.Bank,Producer:producer,ProjectType:p.Modality+" — "+p.Activity,Situation:p.Status,CommissionRate:0.5,Commission:p.FinancedValue*0.005,Notes:"Lançamento automático por mudança de status"}
	if strings.Contains(s,"enviado") { d.SentDate=d.Date }; _ = a.sb.Insert(r.Context(),"daily_reports",d,nil)
}

func (a *App) projectDetail360(w http.ResponseWriter,r *http.Request) {
	p,err:=a.getProject(r,r.PathValue("id")); if err!=nil {http.NotFound(w,r);return}
	v:=Project360View{Project:p,ActiveTab:defaultString(r.URL.Query().Get("tab"),"resumo"),UpgradeReady:true}
	var cs []Client; _=a.sb.Select(r.Context(),"clients",eq("id",p.ClientID)+"&limit=1",&cs); if len(cs)>0 {v.Client=cs[0]}
	if p.PropertyID!="" {var ps []Property; _=a.sb.Select(r.Context(),"properties",eq("id",p.PropertyID)+"&limit=1",&ps); if len(ps)>0 {v.Property=ps[0]}}
	_ = a.sb.Select(r.Context(),"project_files",eq("project_id",p.ID)+"&"+order("created_at",true),&v.Files)
	if err:=a.sb.Select(r.Context(),"project_budget_items",eq("project_id",p.ID)+"&"+order("created_at",false),&v.Budget); err!=nil {v.UpgradeReady=false;v.UpgradeError="O código do Dossiê 360 já está publicado, mas falta executar supabase/03_upgrade_dossie_360.sql uma única vez no SQL Editor do Supabase."}
	if v.UpgradeReady {
		_ = a.ensureProjectDefaults360(r,p)
		_ = a.sb.Select(r.Context(),"project_revenues",eq("project_id",p.ID)+"&"+order("created_at",false),&v.Revenues)
		var ms []ProjectMetrics; _=a.sb.Select(r.Context(),"project_metrics",eq("project_id",p.ID)+"&limit=1",&ms); if len(ms)>0 {v.Metrics=ms[0]}
		_ = a.sb.Select(r.Context(),"project_tasks",eq("project_id",p.ID)+"&"+order("due_at",false),&v.Tasks)
		_ = a.sb.Select(r.Context(),"project_history",eq("project_id",p.ID)+"&"+order("event_date",true),&v.History)
		_ = a.sb.Select(r.Context(),"project_checklist",eq("project_id",p.ID)+"&"+order("document_name",false),&v.Checklist)
		_ = a.sb.Select(r.Context(),"project_sections",eq("project_id",p.ID)+"&"+order("section_title",false),&v.Sections)
		a.syncChecklist360(r,&v)
	}
	a.calculateProject360(&v)
	for _,f:=range v.Files { if f.IsCurrent && (strings.EqualFold(f.DocumentType,"KML")||strings.HasSuffix(strings.ToLower(f.Name),".kml")) { if b,_,e:=a.sb.Download(r.Context(),a.cfg.StorageBucket,f.StoragePath);e==nil {s:=parseKML360(b);v.MapSVG=s.SVG;v.KMLAreaHa=s.AreaHa;v.KMLPerimeterM=s.PerimeterM;break} } }
	a.render(w,r,"project_detail",ViewData{Title:p.Title,Data:v})
}

func (a *App) calculateProject360(v *Project360View) {
	for _,x:=range v.Budget {v.BudgetTotal+=x.TotalValue;v.BudgetOwn+=x.OwnResources;v.BudgetFinanced+=x.FinancedValue}
	for _,x:=range v.Revenues {v.RevenueTotal+=x.TotalValue}
	v.OperatingMargin=v.RevenueTotal+v.Metrics.OtherIncome-v.BudgetTotal
	v.CashBeforeDebt=v.OperatingMargin-v.Metrics.OtherDebts
	v.NetAfterDebt=v.CashBeforeDebt-v.Metrics.AnnualPayment
	if v.Metrics.AnnualPayment>0 {v.DebtCoverage=v.CashBeforeDebt/v.Metrics.AnnualPayment}
	if v.Metrics.AreaHa>0 {v.CostPerHa=v.BudgetTotal/v.Metrics.AreaHa;v.RevenuePerHa=v.RevenueTotal/v.Metrics.AreaHa}
	if v.Metrics.AnimalCount>0 {v.CostPerAnimal=v.BudgetTotal/float64(v.Metrics.AnimalCount);v.RevenuePerAnimal=v.RevenueTotal/float64(v.Metrics.AnimalCount); if v.Metrics.FinalWeightKg>0&&v.Metrics.CarcassYieldPct>0 {v.ProjectedArrobas=float64(v.Metrics.AnimalCount)*v.Metrics.FinalWeightKg*(v.Metrics.CarcassYieldPct/100)/15}}
	if v.Metrics.AnnualPayment<=0 {v.ViabilityLabel="Parcela anual não informada";v.ViabilityClass="neutral"} else if v.DebtCoverage>=1.25 {v.ViabilityLabel="Viável — cobertura confortável";v.ViabilityClass="ok"} else if v.DebtCoverage>=1 {v.ViabilityLabel="Atenção — cobertura apertada";v.ViabilityClass="warn"} else {v.ViabilityLabel="Insuficiente — revisar premissas";v.ViabilityClass="bad"}
	req,done:=0,0; for _,c:=range v.Checklist {if c.Required {req++; if strings.Contains(strings.ToLower(c.Status),"concl")||strings.Contains(strings.ToLower(c.Status),"ok")||strings.Contains(strings.ToLower(c.Status),"receb") {done++}}}; if req>0 {v.DocCompleteness=int(float64(done)/float64(req)*100)}
	techTotal,techDone:=2,0; if v.Project.TechnicalSummary!="" {techDone++}; if v.Metrics.AreaHa>0||v.Metrics.AnimalCount>0 {techDone++}; for _,s:=range v.Sections {techTotal++;if strings.Contains(strings.ToLower(s.Status),"preench")||strings.Contains(strings.ToLower(s.Status),"concl") {techDone++}}; if techTotal>0 {v.TechCompleteness=int(float64(techDone)/float64(techTotal)*100)}
	finDone:=0; if len(v.Budget)>0 {finDone++};if len(v.Revenues)>0 {finDone++};if v.Metrics.AnnualPayment>0 {finDone++};v.FinCompleteness=finDone*100/3
	v.Completeness=(v.DocCompleteness*40+v.TechCompleteness*30+v.FinCompleteness*30)/100
	today:=nowBR360().Format("2006-01-02");for _,t:=range v.Tasks {if taskDone360(t.Status){continue};v.PendingCount++;if t.DueAt!=""&&t.DueAt<today{v.OverdueCount++}}
}

func (a *App) ensureProjectDefaults360(r *http.Request,p Project) error {
	var existing []ChecklistItem; if err:=a.sb.Select(r.Context(),"project_checklist",eq("project_id",p.ID),&existing);err!=nil{return err}; seen:=map[string]bool{};for _,x:=range existing{seen[x.Code]=true}
	for _,x:=range checklistSpecs360(p) {if !seen[x.Code] {x.ProjectID=p.ID; _=a.sb.Insert(r.Context(),"project_checklist",x,nil)}}
	var sections []ProjectSection; _=a.sb.Select(r.Context(),"project_sections",eq("project_id",p.ID),&sections); ss:=map[string]bool{};for _,s:=range sections{ss[s.SectionKey]=true}
	for _,sp:=range sectionSpecs360(){if !ss[sp.SectionKey]{sp.ProjectID=p.ID;_=a.sb.Insert(r.Context(),"project_sections",sp,nil)}}
	return nil
}

func checklistSpecs360(p Project) []ChecklistItem {
	x:=[]ChecklistItem{{Code:"IDENT",DocumentName:"Identificação do proponente",Required:true,Status:"Pendente",SourceRule:"Cadastro / instituição financeira"},{Code:"TERRA",DocumentName:"Matrícula, contrato ou carta de anuência",Required:true,Status:"Pendente",SourceRule:"MCR 1-2-8 e 1-2-9"},{Code:"CAR",DocumentName:"Recibo de inscrição no CAR",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-12"},{Code:"COORD",DocumentName:"KML / coordenadas geodésicas da área financiada",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-2"},{Code:"CCIR",DocumentName:"CCIR",Required:false,Status:"Pendente",SourceRule:"Conforme instituição"},{Code:"ITR",DocumentName:"ITR / documentação fiscal do imóvel",Required:false,Status:"Pendente",SourceRule:"Conforme instituição"},{Code:"ORC",DocumentName:"Orçamento / memória de cálculo",Required:true,Status:"Pendente",SourceRule:"MCR 2-2-1 a 2-2-4"}}
	m:=strings.ToLower(p.Modality+" "+p.Activity)
	if strings.Contains(m,"agrícola")||strings.Contains(m,"agricola")||strings.Contains(m,"soja")||strings.Contains(m,"milho")||strings.Contains(m,"café")||strings.Contains(m,"cafe")||strings.Contains(m,"sorgo") {x=append(x,ChecklistItem{Code:"ZARC",DocumentName:"Conferência ZARC / janela de plantio",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-1"},ChecklistItem{Code:"SOLO",DocumentName:"Análise de solo / recomendação técnica",Required:false,Status:"Pendente",SourceRule:"Roteiro B 4.4.2"})}
	if strings.Contains(m,"pecu")||strings.Contains(m,"bov")||strings.Contains(m,"gado")||strings.Contains(m,"leite") {x=append(x,ChecklistItem{Code:"SANIT",DocumentName:"Ficha sanitária do rebanho",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-20-b"})}
	if strings.Contains(m,"aquisi")||strings.Contains(m,"animal") {x=append(x,ChecklistItem{Code:"NF",DocumentName:"Nota fiscal dos animais",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-20-a-I"},ChecklistItem{Code:"GTA",DocumentName:"Guia de Trânsito Animal (GTA)",Required:true,Status:"Pendente",SourceRule:"MCR 2-1-20-a-II"},ChecklistItem{Code:"OFERTA",DocumentName:"Carta de oferta / inventário do vendedor",Required:false,Status:"Pendente",SourceRule:"Conforme exigência da instituição"})}
	if strings.Contains(m,"invest")||strings.Contains(m,"reforma") {x=append(x,ChecklistItem{Code:"FORN",DocumentName:"Orçamentos atualizados de fornecedores",Required:true,Status:"Pendente",SourceRule:"Roteiro B 4.2.3"},ChecklistItem{Code:"AMB",DocumentName:"Licença, dispensa ou regularidade ambiental aplicável",Required:false,Status:"Pendente",SourceRule:"Roteiro B 4.3.2 / legislação aplicável"})}
	return x
}

func sectionSpecs360() []ProjectSection {
	names:=[][2]string{{"identificacao","01. Identificação e proponente"},{"patrimonio","02. Levantamento patrimonial e obrigações"},{"infraestrutura","03. Infraestrutura, clima e localização"},{"recursos_hidricos","04. Recursos hídricos"},{"meio_ambiente","05. Meio ambiente e conformidade"},{"producao_historica","06. Receitas e produção histórica"},{"mercado","07. Mercado e comercialização"},{"finalidade","08. Finalidade do projeto"},{"financiamento","09. Financiamento, usos e fontes"},{"tecnologia","10. Administração e tecnologia"},{"impactos","11. Impactos socioambientais"},{"engenharia","12. Engenharia e dimensionamentos"},{"receitas","13. Previsão de receitas"},{"custos","14. Estrutura de custos"},{"fluxo","15. Fluxo de caixa e capacidade de pagamento"},{"conclusao","16. Conclusão técnica"}}
	out:=make([]ProjectSection,0,len(names));for _,n:=range names{out=append(out,ProjectSection{SectionKey:n[0],SectionTitle:n[1],Status:"Pendente"})};return out
}

func (a *App) syncChecklist360(r *http.Request,v *Project360View) {
	statusByCode:=map[string]bool{}
	if v.Property.CAR!="" {statusByCode["CAR"]=true};if v.Property.CCIR!=""{statusByCode["CCIR"]=true};if v.Property.ITR!=""{statusByCode["ITR"]=true};if v.Property.Registry!=""||strings.Contains(strings.ToLower(v.Property.Tenure),"arrend")||strings.Contains(strings.ToLower(v.Property.Tenure),"comod") {statusByCode["TERRA"]=true}
	for _,f:=range v.Files {if !f.IsCurrent&&f.DocumentType!=""{continue}; s:=strings.ToLower(f.DocumentType+" "+f.Name);switch{case strings.Contains(s,"kml")||strings.Contains(s,"coorden"):statusByCode["COORD"]=true;case strings.Contains(s,"gta"):statusByCode["GTA"]=true;case strings.Contains(s,"nota fiscal")||strings.Contains(s," nf"):statusByCode["NF"]=true;case strings.Contains(s,"sanit"):statusByCode["SANIT"]=true;case strings.Contains(s,"zarc"):statusByCode["ZARC"]=true;case strings.Contains(s,"solo"):statusByCode["SOLO"]=true;case strings.Contains(s,"orçamento")||strings.Contains(s,"orcamento"):statusByCode["ORC"]=true;case strings.Contains(s,"oferta")||strings.Contains(s,"inventário")||strings.Contains(s,"inventario"):statusByCode["OFERTA"]=true}}
	for i:=range v.Checklist {if statusByCode[v.Checklist[i].Code] && !strings.Contains(strings.ToLower(v.Checklist[i].Status),"concl") {v.Checklist[i].Status="Concluído";_=a.sb.Update(r.Context(),"project_checklist",eq("id",v.Checklist[i].ID),map[string]any{"status":"Concluído"},nil)}}
}

func (a *App) requireUpgrade360(w http.ResponseWriter,r *http.Request) bool {var x []BudgetItem;if err:=a.sb.Select(r.Context(),"project_budget_items","select=id&limit=1",&x);err!=nil{http.Error(w,"Dossiê 360 ainda não ativado no banco. Execute uma única vez o arquivo supabase/03_upgrade_dossie_360.sql no SQL Editor do Supabase.",503);return false};return true}
func redirectProject360(w http.ResponseWriter,r *http.Request,id,tab,msg string){http.Redirect(w,r,"/projects/"+id+"?tab="+url.QueryEscape(tab)+"&ok="+url.QueryEscape(msg),303)}

func (a *App) budgetCreate360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r)||!a.requireUpgrade360(w,r){return};id:=r.PathValue("id");q:=parseFloat(r.FormValue("quantity"));uv:=parseFloat(r.FormValue("unit_value"));own:=parseFloat(r.FormValue("own_resources"));total:=parseFloat(r.FormValue("total_value"));if total==0{total=q*uv};item:=BudgetItem{ProjectID:id,Category:r.FormValue("category"),Item:strings.TrimSpace(r.FormValue("item")),AcquisitionStart:r.FormValue("acquisition_start"),AcquisitionEnd:r.FormValue("acquisition_end"),UseStart:r.FormValue("use_start"),UseEnd:r.FormValue("use_end"),Unit:r.FormValue("unit"),Quantity:q,UnitValue:uv,TotalValue:total,OwnResources:own,FinancedValue:math.Max(total-own,0),PriceSource:strings.TrimSpace(r.FormValue("price_source")),Notes:strings.TrimSpace(r.FormValue("notes"))};if item.Item==""{http.Error(w,"Informe o item.",400);return};if err:=a.sb.Insert(r.Context(),"project_budget_items",item,nil);err!=nil{http.Error(w,err.Error(),500);return};_=a.addHistory360(r,id,"Orçamento","Item incluído",item.Item+" — "+formatMoney(item.TotalValue));redirectProject360(w,r,id,"orcamento","Item incluído no orçamento.")}
func (a *App) budgetDelete360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};_=a.sb.Delete(r.Context(),"project_budget_items",eq("id",r.PathValue("item")));redirectProject360(w,r,r.PathValue("id"),"orcamento","Item removido.")}
func (a *App) revenueCreate360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r)||!a.requireUpgrade360(w,r){return};id:=r.PathValue("id");q:=parseFloat(r.FormValue("quantity"));p:=parseFloat(r.FormValue("unit_price"));rv:=ProjectRevenue{ProjectID:id,Product:strings.TrimSpace(r.FormValue("product")),Period:r.FormValue("period"),Unit:r.FormValue("unit"),Quantity:q,UnitPrice:p,TotalValue:q*p,Source:r.FormValue("source"),Notes:r.FormValue("notes")};if rv.Product==""{http.Error(w,"Informe o produto.",400);return};if err:=a.sb.Insert(r.Context(),"project_revenues",rv,nil);err!=nil{http.Error(w,err.Error(),500);return};_=a.addHistory360(r,id,"Receita","Receita incluída",rv.Product+" — "+formatMoney(rv.TotalValue));redirectProject360(w,r,id,"receitas","Receita incluída.")}
func (a *App) revenueDelete360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};_=a.sb.Delete(r.Context(),"project_revenues",eq("id",r.PathValue("item")));redirectProject360(w,r,r.PathValue("id"),"receitas","Receita removida.")}

func (a *App) metricsSave360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r)||!a.requireUpgrade360(w,r){return};id:=r.PathValue("id");m:=ProjectMetrics{ProjectID:id,ActivityType:r.FormValue("activity_type"),AreaHa:parseFloat(r.FormValue("area_ha")),Productivity:parseFloat(r.FormValue("productivity")),ProductivityUnit:r.FormValue("productivity_unit"),AnimalCount:parseInt(r.FormValue("animal_count")),InitialWeightKg:parseFloat(r.FormValue("initial_weight_kg")),FinalWeightKg:parseFloat(r.FormValue("final_weight_kg")),GMDKgDay:parseFloat(r.FormValue("gmd_kg_day")),CycleDays:parseInt(r.FormValue("cycle_days")),CarcassYieldPct:parseFloat(r.FormValue("carcass_yield_pct")),MortalityPct:parseFloat(r.FormValue("mortality_pct")),StockingRate:parseFloat(r.FormValue("stocking_rate")),OtherDebts:parseFloat(r.FormValue("other_debts")),OtherIncome:parseFloat(r.FormValue("other_income")),AnnualPayment:parseFloat(r.FormValue("annual_payment")),Notes:r.FormValue("notes")};var old []ProjectMetrics;_=a.sb.Select(r.Context(),"project_metrics",eq("project_id",id)+"&limit=1",&old);if len(old)>0{m.ID=old[0].ID;vals:=map[string]any{"activity_type":m.ActivityType,"area_ha":m.AreaHa,"productivity":m.Productivity,"productivity_unit":m.ProductivityUnit,"animal_count":m.AnimalCount,"initial_weight_kg":m.InitialWeightKg,"final_weight_kg":m.FinalWeightKg,"gmd_kg_day":m.GMDKgDay,"cycle_days":m.CycleDays,"carcass_yield_pct":m.CarcassYieldPct,"mortality_pct":m.MortalityPct,"stocking_rate":m.StockingRate,"other_debts":m.OtherDebts,"other_income":m.OtherIncome,"annual_payment":m.AnnualPayment,"notes":m.Notes,"updated_at":time.Now().UTC().Format(time.RFC3339)};_=a.sb.Update(r.Context(),"project_metrics",eq("id",m.ID),vals,nil)}else{_=a.sb.Insert(r.Context(),"project_metrics",m,nil)};_=a.addHistory360(r,id,"Técnico","Indicadores atualizados","Produção e capacidade de pagamento revisadas");redirectProject360(w,r,id,"capacidade","Indicadores salvos.")}

func (a *App) taskCreate360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r)||!a.requireUpgrade360(w,r){return};id:=r.PathValue("id");t:=ProjectTask{ProjectID:id,Title:strings.TrimSpace(r.FormValue("title")),Category:r.FormValue("category"),Responsible:r.FormValue("responsible"),RequestedAt:r.FormValue("requested_at"),DueAt:r.FormValue("due_at"),Status:"Pendente",Priority:defaultString(r.FormValue("priority"),"Normal"),Notes:r.FormValue("notes")};if t.Title==""{http.Error(w,"Informe a pendência.",400);return};_=a.sb.Insert(r.Context(),"project_tasks",t,nil);_=a.addHistory360(r,id,"Pendência","Pendência criada",t.Title);redirectProject360(w,r,id,"pendencias","Pendência registrada.")}
func (a *App) taskToggle360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};var rows []ProjectTask;_=a.sb.Select(r.Context(),"project_tasks",eq("id",r.PathValue("item"))+"&limit=1",&rows);if len(rows)>0{done:=taskDone360(rows[0].Status);vals:=map[string]any{"status":"Concluído","completed_at":time.Now().UTC().Format(time.RFC3339)};if done{vals=map[string]any{"status":"Pendente","completed_at":nil}};_=a.sb.Update(r.Context(),"project_tasks",eq("id",rows[0].ID),vals,nil)};redirectProject360(w,r,r.PathValue("id"),"pendencias","Pendência atualizada.")}
func (a *App) taskDelete360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};_=a.sb.Delete(r.Context(),"project_tasks",eq("id",r.PathValue("item")));redirectProject360(w,r,r.PathValue("id"),"pendencias","Pendência removida.")}
func (a *App) checklistSave360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};vals:=map[string]any{"status":r.FormValue("status"),"valid_until":nullIfEmpty(r.FormValue("valid_until")),"notes":r.FormValue("notes")};_=a.sb.Update(r.Context(),"project_checklist",eq("id",r.PathValue("item")),vals,nil);redirectProject360(w,r,r.PathValue("id"),"documentos","Checklist atualizado.")}
func (a *App) sectionSave360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");vals:=map[string]any{"content":r.FormValue("content"),"status":r.FormValue("status"),"updated_at":time.Now().UTC().Format(time.RFC3339)};_=a.sb.Update(r.Context(),"project_sections",eq("project_id",id)+"&"+eq("section_key",r.PathValue("key")),vals,nil);redirectProject360(w,r,id,"tecnico","Memória técnica salva.")}
func (a *App) historyCreate360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");_=a.addHistory360(r,id,r.FormValue("event_type"),strings.TrimSpace(r.FormValue("title")),r.FormValue("details"));redirectProject360(w,r,id,"historico","Evento registrado.")}
func (a *App) addHistory360(r *http.Request,id,typ,title,details string) error {if title==""{return nil};u:=userFromContext(r.Context());uid:="";if u!=nil{uid=u.ID};return a.sb.Insert(r.Context(),"project_history",ProjectHistory{ProjectID:id,EventType:typ,Title:title,Details:details,UserID:uid},nil)}

func (a *App) projectFileUpload360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};id:=r.PathValue("id");r.Body=http.MaxBytesReader(w,r.Body,45<<20);if err:=r.ParseMultipartForm(45<<20);err!=nil{http.Error(w,"Arquivo muito grande. Limite: 45 MB.",400);return};f,h,err:=r.FormFile("file");if err!=nil{http.Error(w,"Selecione um arquivo.",400);return};defer f.Close();name:=safeFilename(h.Filename);ct:=h.Header.Get("Content-Type");if ct==""{ct=mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))};if ct==""{ct="application/octet-stream"};docType:=strings.TrimSpace(r.FormValue("document_type"));version:=strings.TrimSpace(r.FormValue("version_label"));valid:=r.FormValue("valid_until");if docType!=""{_=a.sb.Update(r.Context(),"project_files",eq("project_id",id)+"&"+eq("document_type",docType),map[string]any{"is_current":false},nil)};objectPath:=fmt.Sprintf("projects/%s/%d-%s",id,time.Now().UnixNano(),name);if err:=a.sb.Upload(r.Context(),a.cfg.StorageBucket,objectPath,ct,io.LimitReader(f,45<<20));err!=nil{http.Error(w,"Falha no upload: "+err.Error(),500);return};u:=userFromContext(r.Context());meta:=ProjectFile{ProjectID:id,Name:name,StoragePath:objectPath,ContentType:ct,SizeBytes:h.Size,UploadedBy:u.ID,DocumentType:docType,VersionLabel:version,ValidUntil:valid,IsCurrent:true};if err:=a.sb.Insert(r.Context(),"project_files",meta,nil);err!=nil{_=a.sb.DeleteObject(r.Context(),a.cfg.StorageBucket,objectPath);http.Error(w,"Falha ao registrar arquivo. Rode a migração 03 se necessário.",500);return};_=a.addHistory360(r,id,"Documento","Documento enviado",docType+" · "+name);redirectProject360(w,r,id,"documentos","Documento enviado e versionado.")}
func (a *App) fileDelete360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};var rows []ProjectFile;_=a.sb.Select(r.Context(),"project_files",eq("id",r.PathValue("id"))+"&limit=1",&rows);if len(rows)==0{http.NotFound(w,r);return};f:=rows[0];_=a.sb.DeleteObject(r.Context(),a.cfg.StorageBucket,f.StoragePath);_=a.sb.Delete(r.Context(),"project_files",eq("id",f.ID));_=a.addHistory360(r,f.ProjectID,"Documento","Documento excluído",f.Name);redirectProject360(w,r,f.ProjectID,"documentos","Documento excluído.")}

func (a *App) propertyCreate360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};p:=Property{ClientID:r.FormValue("client_id"),Name:strings.TrimSpace(r.FormValue("name")),City:r.FormValue("city"),State:r.FormValue("state"),AreaHa:parseFloat(r.FormValue("area_ha")),Registry:r.FormValue("registry"),CAR:r.FormValue("car"),CCIR:r.FormValue("ccir"),ITR:r.FormValue("itr"),Tenure:r.FormValue("tenure"),Latitude:parseFloat(r.FormValue("latitude")),Longitude:parseFloat(r.FormValue("longitude")),WaterInfo:r.FormValue("water_info"),EnvironmentalInfo:r.FormValue("environmental_info"),AccessInfo:r.FormValue("access_info"),OwnerName:r.FormValue("owner_name"),RegistryDate:r.FormValue("registry_date"),CARStatus:r.FormValue("car_status"),Notes:r.FormValue("notes")};if p.ClientID==""||p.Name==""{http.Error(w,"Produtor e nome são obrigatórios.",400);return};if err:=a.sb.Insert(r.Context(),"properties",p,nil);err!=nil{http.Error(w,"Falha ao salvar. Rode a migração 03 para habilitar os campos completos.",500);return};http.Redirect(w,r,"/properties?ok="+url.QueryEscape("Propriedade cadastrada."),303)}
func (a *App) propertiesList360(w http.ResponseWriter,r *http.Request){var rows []Property;_ = a.sb.Select(r.Context(),"properties","select=*&"+order("name",false),&rows);var cs []Client;_=a.sb.Select(r.Context(),"clients","select=id,name&"+order("name",false),&cs);filesBy:=map[string][]PropertyFile360{};maps:=map[string]string{};stats:=map[string]string{};upgrade:=true;var fsx []PropertyFile360;if e:=a.sb.Select(r.Context(),"property_files","select=*&"+order("created_at",true),&fsx);e!=nil{upgrade=false}else{for _,f:=range fsx{filesBy[f.PropertyID]=append(filesBy[f.PropertyID],f)};for _,p:=range rows{for _,f:=range filesBy[p.ID]{if f.IsCurrent&&(strings.EqualFold(f.DocumentType,"KML")||strings.HasSuffix(strings.ToLower(f.Name),".kml")){if b,_,e:=a.sb.Download(r.Context(),a.cfg.StorageBucket,f.StoragePath);e==nil{s:=parseKML360(b);maps[p.ID]=s.SVG;stats[p.ID]=fmt.Sprintf("%.4f ha · %.0f m",s.AreaHa,s.PerimeterM);break}}}}};a.render(w,r,"properties",ViewData{Title:"Propriedades",Data:map[string]any{"Properties":rows,"Clients":cs,"FilesByProperty":filesBy,"Maps":maps,"MapStats":stats,"UpgradeReady":upgrade}})}
func (a *App) propertyFileUpload360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r)||!a.requireUpgrade360(w,r){return};id:=r.PathValue("id");r.Body=http.MaxBytesReader(w,r.Body,45<<20);if err:=r.ParseMultipartForm(45<<20);err!=nil{http.Error(w,"Arquivo muito grande.",400);return};f,h,err:=r.FormFile("file");if err!=nil{http.Error(w,"Selecione um arquivo.",400);return};defer f.Close();name:=safeFilename(h.Filename);ct:=h.Header.Get("Content-Type");if ct==""{ct=mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))};if ct==""{ct="application/octet-stream"};typ:=strings.TrimSpace(r.FormValue("document_type"));if typ!=""{_=a.sb.Update(r.Context(),"property_files",eq("property_id",id)+"&"+eq("document_type",typ),map[string]any{"is_current":false},nil)};path:=fmt.Sprintf("properties/%s/%d-%s",id,time.Now().UnixNano(),name);if err:=a.sb.Upload(r.Context(),a.cfg.StorageBucket,path,ct,io.LimitReader(f,45<<20));err!=nil{http.Error(w,err.Error(),500);return};u:=userFromContext(r.Context());row:=PropertyFile360{PropertyID:id,DocumentType:typ,Name:name,VersionLabel:r.FormValue("version_label"),ValidUntil:r.FormValue("valid_until"),StoragePath:path,ContentType:ct,SizeBytes:h.Size,UploadedBy:u.ID,IsCurrent:true};if err:=a.sb.Insert(r.Context(),"property_files",row,nil);err!=nil{_=a.sb.DeleteObject(r.Context(),a.cfg.StorageBucket,path);http.Error(w,err.Error(),500);return};http.Redirect(w,r,"/properties?ok="+url.QueryEscape("Documento da propriedade enviado."),303)}
func (a *App) propertyFileDownload360(w http.ResponseWriter,r *http.Request){var rows []PropertyFile360;_=a.sb.Select(r.Context(),"property_files",eq("id",r.PathValue("id"))+"&limit=1",&rows);if len(rows)==0{http.NotFound(w,r);return};b,ct,err:=a.sb.Download(r.Context(),a.cfg.StorageBucket,rows[0].StoragePath);if err!=nil{http.Error(w,"Não foi possível abrir.",500);return};if ct==""{ct=rows[0].ContentType};w.Header().Set("Content-Type",ct);w.Header().Set("Content-Disposition",fmt.Sprintf(`inline; filename="%s"`,strings.ReplaceAll(rows[0].Name,"\"","")));_,_=w.Write(b)}
func (a *App) propertyFileDelete360(w http.ResponseWriter,r *http.Request){if !a.verifyCSRF(r){http.Error(w,"Sessão inválida",403);return};var rows []PropertyFile360;_=a.sb.Select(r.Context(),"property_files",eq("id",r.PathValue("id"))+"&limit=1",&rows);if len(rows)>0{_=a.sb.DeleteObject(r.Context(),a.cfg.StorageBucket,rows[0].StoragePath);_=a.sb.Delete(r.Context(),"property_files",eq("id",rows[0].ID))};http.Redirect(w,r,"/properties?ok="+url.QueryEscape("Documento removido."),303)}

func (a *App) projectExport360(w http.ResponseWriter,r *http.Request){p,err:=a.getProject(r,r.PathValue("id"));if err!=nil{http.NotFound(w,r);return};v:=Project360View{Project:p,UpgradeReady:true};_ = a.sb.Select(r.Context(),"project_budget_items",eq("project_id",p.ID),&v.Budget);_ = a.sb.Select(r.Context(),"project_revenues",eq("project_id",p.ID),&v.Revenues);_ = a.sb.Select(r.Context(),"project_tasks",eq("project_id",p.ID),&v.Tasks);_ = a.sb.Select(r.Context(),"project_history",eq("project_id",p.ID),&v.History);_ = a.sb.Select(r.Context(),"project_checklist",eq("project_id",p.ID),&v.Checklist);_ = a.sb.Select(r.Context(),"project_files",eq("project_id",p.ID),&v.Files);var ms []ProjectMetrics;_=a.sb.Select(r.Context(),"project_metrics",eq("project_id",p.ID)+"&limit=1",&ms);if len(ms)>0{v.Metrics=ms[0]};a.calculateProject360(&v);var buf bytes.Buffer;zw:=zip.NewWriter(&buf);writeZipText360(zw,"00_RESUMO.txt",projectSummaryText360(v));writeBudgetCSV360(zw,v.Budget);writeRevenueCSV360(zw,v.Revenues);writeTaskCSV360(zw,v.Tasks);writeChecklistCSV360(zw,v.Checklist);for _,f:=range v.Files{b,_,e:=a.sb.Download(r.Context(),a.cfg.StorageBucket,f.StoragePath);if e==nil{z,_:=zw.Create("documentos/"+safeFilename(f.Name));_,_=z.Write(b)}};_ = zw.Close();w.Header().Set("Content-Type","application/zip");w.Header().Set("Content-Disposition",fmt.Sprintf(`attachment; filename="dossie-%s.zip"`,slug360(p.Title)));w.Header().Set("Content-Length",strconv.Itoa(buf.Len()));_,_=w.Write(buf.Bytes())}
func writeZipText360(z *zip.Writer,name,txt string){w,_:=z.Create(name);_,_=w.Write([]byte(txt))}
func projectSummaryText360(v Project360View)string{return fmt.Sprintf("VIA VERDE — DOSSIÊ DO PROJETO\n\nProjeto: %s\nProdutor: %s\nBanco: %s\nModalidade: %s\nAtividade: %s\nStatus: %s\nValor total: %s\nFinanciamento: %s\nTaxa: %.2f%% a.a.\nPrazo: %d meses\n\nOrçamento: %s\nReceitas: %s\nMargem operacional: %s\nServiço anual da dívida informado: %s\nSaldo após dívida: %s\nCobertura: %.2fx\nResultado preliminar: %s\nCompletude: %d%%\n\nObservação: análise econômico-financeira interna. A decisão de crédito cabe à instituição financeira.\n",v.Project.Title,v.Project.ClientName,v.Project.Bank,v.Project.Modality,v.Project.Activity,v.Project.Status,formatMoney(v.Project.TotalValue),formatMoney(v.Project.FinancedValue),v.Project.InterestRate,v.Project.TermMonths,formatMoney(v.BudgetTotal),formatMoney(v.RevenueTotal),formatMoney(v.OperatingMargin),formatMoney(v.Metrics.AnnualPayment),formatMoney(v.NetAfterDebt),v.DebtCoverage,v.ViabilityLabel,v.Completeness)}
func csvToZip360(z *zip.Writer,name string,header []string,rows [][]string){w,_:=z.Create(name);c:=csv.NewWriter(w);_ = c.Write(header);_ = c.WriteAll(rows);c.Flush()}
func writeBudgetCSV360(z *zip.Writer,in []BudgetItem){rows:=[][]string{};for _,x:=range in{rows=append(rows,[]string{x.Category,x.Item,x.Unit,fmt.Sprint(x.Quantity),fmt.Sprintf("%.2f",x.UnitValue),fmt.Sprintf("%.2f",x.TotalValue),fmt.Sprintf("%.2f",x.OwnResources),fmt.Sprintf("%.2f",x.FinancedValue),x.PriceSource,x.Notes})};csvToZip360(z,"01_ORCAMENTO.csv",[]string{"Categoria","Item","Unidade","Quantidade","Valor unitário","Total","Recursos próprios","Financiado","Fonte","Observações"},rows)}
func writeRevenueCSV360(z *zip.Writer,in []ProjectRevenue){rows:=[][]string{};for _,x:=range in{rows=append(rows,[]string{x.Product,x.Period,x.Unit,fmt.Sprint(x.Quantity),fmt.Sprintf("%.2f",x.UnitPrice),fmt.Sprintf("%.2f",x.TotalValue),x.Source,x.Notes})};csvToZip360(z,"02_RECEITAS.csv",[]string{"Produto","Período","Unidade","Quantidade","Preço","Total","Fonte","Observações"},rows)}
func writeTaskCSV360(z *zip.Writer,in []ProjectTask){rows:=[][]string{};for _,x:=range in{rows=append(rows,[]string{x.Title,x.Category,x.Responsible,x.RequestedAt,x.DueAt,x.Status,x.Priority,x.Notes})};csvToZip360(z,"03_PENDENCIAS.csv",[]string{"Pendência","Categoria","Responsável","Solicitada","Prazo","Status","Prioridade","Observações"},rows)}
func writeChecklistCSV360(z *zip.Writer,in []ChecklistItem){rows:=[][]string{};for _,x:=range in{rows=append(rows,[]string{x.DocumentName,fmt.Sprint(x.Required),x.Status,x.ValidUntil,x.SourceRule,x.Notes})};csvToZip360(z,"04_CHECKLIST.csv",[]string{"Documento","Obrigatório","Status","Validade","Referência","Observações"},rows)}
func slug360(s string)string{s=strings.ToLower(strings.TrimSpace(s));r:=strings.NewReplacer(" ","-","/","-","\\","-","á","a","ã","a","â","a","é","e","ê","e","í","i","ó","o","õ","o","ô","o","ú","u","ç","c");s=r.Replace(s);return s}
