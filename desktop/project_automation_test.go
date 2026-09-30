package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProjectIntegrationSettings230(t *testing.T) {
	app:=newProjectTestApp(t)
	s,err:=app.SaveProjectIntegrationSettings(ProjectIntegrationSettingsInput{
		EmbrapaToken:"token-teste",ANAUsername:"123",ANAPassword:"senha",
	})
	if err!=nil{t.Fatal(err)}
	if !s.EmbrapaConfigured||!s.ANAConfigured||s.ANAUsername!="123"{t.Fatalf("configuração inesperada: %#v",s)}
	if app.settingValue("project.embrapa_token")!="token-teste"{t.Fatal("token Embrapa não persistido")}
	if _,err:=app.SaveProjectIntegrationSettings(ProjectIntegrationSettingsInput{ClearEmbrapa:true,ClearANA:true});err!=nil{t.Fatal(err)}
	s=app.GetProjectIntegrationSettings()
	if s.EmbrapaConfigured||s.ANAConfigured{t.Fatalf("credenciais deveriam estar limpas: %#v",s)}
}

func TestRuralProjectAutomation230WithoutCredentials(t *testing.T) {
	app:=newProjectTestApp(t)
	client,_:=app.SaveClient(Client{Name:"Produtor"})
	property,_:=app.SaveProperty(Property{ClientID:client.ID,Name:"Fazenda",Municipality:"Jacuí",UF:"MG"})
	project,err:=app.SaveRuralProject(RuralProject{
		PropertyID:property.ID,Name:"Investimento Café",Bank:"Banco",OperationType:"investimento",
		Activity:"Café",RequestedAmount:50000,AreaHa:5,Status:"draft",
	})
	if err!=nil{t.Fatal(err)}
	if _,err:=app.SaveRuralProjectTechnicalData(ProjectTechnicalData{ProjectID:project.ID,Culture:"Café",BenefitedAreaHa:5});err!=nil{t.Fatal(err)}

	out,err:=app.RunRuralProjectAutomation(project.ID,true)
	if err!=nil{t.Fatal(err)}
	if out.ProjectID!=project.ID||out.CacheVersion!=projectAutomationCacheVersion{t.Fatalf("automação inválida: %#v",out)}
	if out.Agritec.Status!="pending" {
		t.Fatalf("sem CAR salvo a Agritec deve aguardar dados cadastrais, obtido %q",out.Agritec.Status)
	}
	if out.ZARC.Status!="not_applicable"{t.Fatalf("investimento não deve executar ZARC de custeio: %#v",out.ZARC)}
	if _,err:=app.GetRuralProjectAutomation(project.ID);err!=nil{t.Fatalf("última automação não foi persistida: %v",err)}
	cached,err:=app.RunRuralProjectAutomation(project.ID,false)
	if err!=nil{t.Fatal(err)}
	if !cached.UsedCache{t.Fatal("automação recente deveria usar cache")}
}

func TestSATVegPolygon230(t *testing.T) {
	raw:=`{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46.0,-20.0],[-45.9,-20.0],[-45.9,-20.1],[-46.0,-20.1],[-46.0,-20.0]]]}}`
	p,err:=projectAreaSATVegPolygon(raw)
	if err!=nil{t.Fatal(err)}
	if !strings.Contains(p,"-46.0000000 -20.0000000")||!strings.Contains(p,","){t.Fatalf("polígono SATVeg inesperado: %s",p)}
}

func TestAgritecConnector230(t *testing.T) {
	oldURL,oldClient:=agritecBaseURL,projectHTTPClient
	defer func(){agritecBaseURL=oldURL;projectHTTPClient=oldClient}()
	srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.Header.Get("Authorization")!="Bearer token"{t.Errorf("Bearer ausente: %q",r.Header.Get("Authorization"))}
		if r.URL.Path!="/municipios/3106200/culturas"{t.Errorf("rota inesperada: %s",r.URL.Path)}
		w.Header().Set("Content-Type","application/json")
		_,_=w.Write([]byte(`{"data":[{"id":900,"nome":"CAFÉ ARÁBICA","nomeCompleto":"CAFÉ ARÁBICA","hasZoneamento":true,"hasCultivar":false,"hasProdutividade":false}]}`))
	}))
	defer srv.Close()
	agritecBaseURL=srv.URL
	projectHTTPClient=srv.Client()

	app:=newProjectTestApp(t)
	_ = app.setSettingValue("project.embrapa_token","token")
	out:=app.runAgritecProject(t.Context(),CARResult{MunicipalityCode:"3106200",UF:"MG"},ProjectTechnicalData{Culture:"Café"})
	if out.Status!="ready"||out.Culture.ID!=900{t.Fatalf("Agritec não reconheceu cultura: %#v",out)}
}

func TestSATVegConnector230(t *testing.T) {
	oldURL,oldClient:=satvegBaseURL,projectHTTPClient
	defer func(){satvegBaseURL=oldURL;projectHTTPClient=oldClient}()
	srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.URL.Path!="/seriespoligono"{t.Errorf("rota inesperada: %s",r.URL.Path)}
		if r.Method!="POST"{t.Errorf("método inesperado: %s",r.Method)}
		w.Header().Set("Content-Type","application/json")
		_,_=w.Write([]byte(`{"listaSerie":[0.4,0.6],"listaDatas":["2026-01-01","2026-01-17"]}`))
	}))
	defer srv.Close()
	satvegBaseURL=srv.URL
	projectHTTPClient=srv.Client()

	app:=newProjectTestApp(t)
	_ = app.setSettingValue("project.embrapa_token","token")
	client,_:=app.SaveClient(Client{Name:"Produtor"})
	property,_:=app.SaveProperty(Property{ClientID:client.ID,Name:"Fazenda",UF:"MG"})
	res,err:=app.db.Exec(`INSERT INTO project_areas(property_id,name,purpose,area_ha,perimeter_m,center_lat,center_lon,inside_car_pct,geojson,kml_path,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,datetime('now'),datetime('now'))`,property.ID,"Gleba","",10,0,-20,-46,100,
		`{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-46,-20],[-45.9,-20],[-45.9,-20.1],[-46,-20.1],[-46,-20]]]}}`,"")
	if err!=nil{t.Fatal(err)}
	areaID,_:=res.LastInsertId()
	project,_:=app.SaveRuralProject(RuralProject{PropertyID:property.ID,Name:"Custeio Soja",OperationType:"custeio",Activity:"Soja",AreaID:areaID,Status:"draft"})
	out:=app.runSATVegProject(t.Context(),project)
	if out.Status!="ready"||out.Count!=2||out.LatestValue!=0.6{t.Fatalf("SATVeg inesperado: %#v",out)}
}

func TestProjectDossierPDF230(t *testing.T) {
	p:=Property{Name:"Fazenda",ClientName:"Produtor",Municipality:"Jacuí",UF:"MG",CARNumber:"MG-TESTE"}
	project:=RuralProject{Name:"Custeio Café",Bank:"Banco",OperationType:"custeio",Activity:"Café",RequestedAmount:50000,AreaHa:5}
	auto:=RuralProjectAutomationResult{
		Preparation:ProjectPreparationResult{ReadinessPct:50,Ready:2,Pending:1,Review:1,TechnicalData:ProjectTechnicalData{Culture:"Café",BenefitedAreaHa:5}},
		ZARC:ProjectZARCResult{Status:"review",Detail:"Conferir ZARC."},
		Agritec:AgritecProjectResult{Status:"not_configured",Detail:"Token não configurado."},
		ANA:ANAHydroProjectResult{Status:"not_applicable",Detail:"Não se aplica."},
		SATVeg:SATVegProjectResult{Status:"not_configured",Detail:"Token não configurado."},
		Scope:"Escopo de teste.",
		Pending:[]ProjectPendingAction{{Label:"Documento",Detail:"Pendente",Severity:"pending",Source:"Teste"}},
	}
	pdf:=buildRuralProjectDossierPDF(p,project,auto)
	if len(pdf)<500||!strings.HasPrefix(string(pdf),"%PDF-1.4"){t.Fatalf("PDF de projeto inválido, tamanho=%d",len(pdf))}
}
