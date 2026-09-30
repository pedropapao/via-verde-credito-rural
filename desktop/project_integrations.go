package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	agritecBaseURL = "https://api.cnptia.embrapa.br/agritec/v2"
	satvegBaseURL  = "https://api.cnptia.embrapa.br/satveg/v2"
	anaBaseURL     = "https://www.ana.gov.br/hidrowebservice/EstacoesTelemetricas"
	projectHTTPClient = &http.Client{Timeout: 25 * time.Second}
)

type ProjectIntegrationSettings struct {
	EmbrapaConfigured bool   `json:"embrapa_configured"`
	ANAConfigured     bool   `json:"ana_configured"`
	ANAUsername       string `json:"ana_username"`
}

type ProjectIntegrationSettingsInput struct {
	EmbrapaToken string `json:"embrapa_token"`
	ANAUsername  string `json:"ana_username"`
	ANAPassword  string `json:"ana_password"`
	ClearEmbrapa bool   `json:"clear_embrapa"`
	ClearANA     bool   `json:"clear_ana"`
}

func (a *App) settingValue(key string) string {
	if a == nil || a.db == nil {
		return ""
	}
	var v string
	if err := a.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v); err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func (a *App) setSettingValue(key, value string) error {
	if a == nil || a.db == nil {
		return errors.New("banco local indisponível")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		_, err := a.db.Exec(`DELETE FROM settings WHERE key=?`, key)
		return err
	}
	_, err := a.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (a *App) GetProjectIntegrationSettings() ProjectIntegrationSettings {
	embrapa := a.settingValue("project.embrapa_token")
	anaUser := a.settingValue("project.ana_username")
	anaPass := a.settingValue("project.ana_password")
	return ProjectIntegrationSettings{
		EmbrapaConfigured: embrapa != "",
		ANAConfigured: anaUser != "" && anaPass != "",
		ANAUsername: anaUser,
	}
}

func (a *App) SaveProjectIntegrationSettings(in ProjectIntegrationSettingsInput) (ProjectIntegrationSettings, error) {
	if in.ClearEmbrapa {
		if err := a.setSettingValue("project.embrapa_token", ""); err != nil { return ProjectIntegrationSettings{}, err }
	} else if strings.TrimSpace(in.EmbrapaToken) != "" {
		if err := a.setSettingValue("project.embrapa_token", in.EmbrapaToken); err != nil { return ProjectIntegrationSettings{}, err }
	}
	if in.ClearANA {
		if err := a.setSettingValue("project.ana_username", ""); err != nil { return ProjectIntegrationSettings{}, err }
		if err := a.setSettingValue("project.ana_password", ""); err != nil { return ProjectIntegrationSettings{}, err }
	} else {
		if strings.TrimSpace(in.ANAUsername) != "" {
			if err := a.setSettingValue("project.ana_username", in.ANAUsername); err != nil { return ProjectIntegrationSettings{}, err }
		}
		if strings.TrimSpace(in.ANAPassword) != "" {
			if err := a.setSettingValue("project.ana_password", in.ANAPassword); err != nil { return ProjectIntegrationSettings{}, err }
		}
	}
	return a.GetProjectIntegrationSettings(), nil
}

type AgritecCulture struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	HasZoneamento   bool   `json:"has_zoneamento"`
	HasCultivar     bool   `json:"has_cultivar"`
	HasProductivity bool   `json:"has_productivity"`
}

type AgritecProjectResult struct {
	Status             string           `json:"status"`
	Detail             string           `json:"detail"`
	SourceURL          string           `json:"source_url"`
	Culture            AgritecCulture   `json:"culture"`
	CultureCandidates  int              `json:"culture_candidates"`
	Cultivars          []string         `json:"cultivars"`
	ProductivityRan    bool             `json:"productivity_ran"`
	ProductivityLatest float64          `json:"productivity_latest"`
	ProductivityMean   float64          `json:"productivity_mean"`
	UpdatedAt          string           `json:"updated_at"`
}

type agritecCulturePayload struct {
	Data []struct {
		ID              int64  `json:"id"`
		Name            string `json:"nome"`
		FullName        string `json:"nomeCompleto"`
		HasZoneamento   bool   `json:"hasZoneamento"`
		HasCultivar     bool   `json:"hasCultivar"`
		HasProductivity bool   `json:"hasProdutividade"`
	} `json:"data"`
}

func bearerRequest(ctx context.Context, method, target, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil { return nil, err }
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	if body != nil { req.Header.Set("Content-Type", "application/json") }
	if strings.TrimSpace(token) != "" { req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token)) }
	return req, nil
}

func decodeHTTPJSON(resp *http.Response, dest any) error {
	if resp == nil { return errors.New("resposta vazia") }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		msg := strings.TrimSpace(string(b))
		if len(msg) > 240 { msg = msg[:240] }
		if msg == "" { return fmt.Errorf("HTTP %d", resp.StatusCode) }
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(dest)
}

func agritecSeason(v string) string {
	v = strings.TrimSpace(strings.ReplaceAll(v, "/", "-"))
	if len(v) == 9 && v[4] == '-' { return v }
	return ""
}

func normalizeProjectCulture(v string) string {
	v = projectSignal(v)
	v = strings.ReplaceAll(v, "-", " ")
	return strings.Join(strings.Fields(v), " ")
}

func selectAgritecCulture(items []struct {
	ID              int64  `json:"id"`
	Name            string `json:"nome"`
	FullName        string `json:"nomeCompleto"`
	HasZoneamento   bool   `json:"hasZoneamento"`
	HasCultivar     bool   `json:"hasCultivar"`
	HasProductivity bool   `json:"hasProdutividade"`
}, wanted string, explicitID int64) (AgritecCulture, int) {
	if explicitID > 0 {
		for _, item := range items {
			if item.ID == explicitID {
				return AgritecCulture{ID:item.ID,Name:item.Name,FullName:item.FullName,HasZoneamento:item.HasZoneamento,HasCultivar:item.HasCultivar,HasProductivity:item.HasProductivity}, 1
			}
		}
	}
	w := normalizeProjectCulture(wanted)
	var matches []AgritecCulture
	for _, item := range items {
		n := normalizeProjectCulture(item.Name+" "+item.FullName)
		if w != "" && (n == w || strings.HasPrefix(n, w+" ") || strings.Contains(n, w)) {
			matches = append(matches, AgritecCulture{ID:item.ID,Name:item.Name,FullName:item.FullName,HasZoneamento:item.HasZoneamento,HasCultivar:item.HasCultivar,HasProductivity:item.HasProductivity})
		}
	}
	if len(matches) == 0 { return AgritecCulture{}, 0 }
	return matches[0], len(matches)
}

func (a *App) runAgritecProject(ctx context.Context, car CARResult, tech ProjectTechnicalData) AgritecProjectResult {
	out := AgritecProjectResult{Status:"not_run", SourceURL:agritecBaseURL, UpdatedAt:time.Now().Format(time.RFC3339)}
	token := a.settingValue("project.embrapa_token")
	if token == "" {
		out.Status = "not_configured"
		out.Detail = "Token da AgroAPI/Embrapa não configurado."
		return out
	}
	if strings.TrimSpace(car.MunicipalityCode) == "" || strings.TrimSpace(tech.Culture) == "" {
		out.Status = "pending"
		out.Detail = "Faltam código IBGE do município ou cultura para consultar a Agritec."
		return out
	}
	target := fmt.Sprintf("%s/municipios/%s/culturas", agritecBaseURL, url.PathEscape(strings.TrimLeft(car.MunicipalityCode, "0")))
	req, err := bearerRequest(ctx, http.MethodGet, target, token, nil)
	if err != nil { out.Status="error"; out.Detail=err.Error(); return out }
	resp, err := projectHTTPClient.Do(req)
	if err != nil { out.Status="source_unavailable"; out.Detail="Agritec indisponível nesta execução: "+err.Error(); return out }
	var payload agritecCulturePayload
	if err := decodeHTTPJSON(resp, &payload); err != nil {
		if strings.Contains(err.Error(),"401") || strings.Contains(err.Error(),"403") { out.Status="not_configured" } else { out.Status="source_unavailable" }
		out.Detail = "Consulta Agritec não concluída: "+err.Error()
		return out
	}
	culture, candidates := selectAgritecCulture(payload.Data, tech.Culture, tech.AgritecCultureID)
	out.CultureCandidates = candidates
	if culture.ID == 0 {
		out.Status = "review"
		out.Detail = "A cultura informada não foi associada com segurança a uma cultura Agritec do município."
		return out
	}
	out.Culture = culture
	if candidates > 1 && tech.AgritecCultureID == 0 {
		out.Status = "review"
		out.Detail = fmt.Sprintf("%d variações da cultura foram encontradas; informe o ID Agritec nos Dados Técnicos para fixar a seleção.", candidates)
	} else {
		out.Status = "ready"
		out.Detail = "Cultura reconhecida pela Agritec."
	}
	season := agritecSeason(tech.CropSeason)
	if culture.HasCultivar && season != "" && strings.TrimSpace(car.UF) != "" {
		q := url.Values{}
		q.Set("safra", season)
		q.Set("idCultura", strconv.FormatInt(culture.ID,10))
		q.Set("uf", strings.ToUpper(strings.TrimSpace(car.UF)))
		req, _ := bearerRequest(ctx,http.MethodGet,agritecBaseURL+"/cultivares?"+q.Encode(),token,nil)
		if resp, err := projectHTTPClient.Do(req); err == nil {
			var cp struct{ Data []struct{ ID int64 `json:"idCultivar"`; Name string `json:"cultivar"` } `json:"data"` }
			if decodeHTTPJSON(resp,&cp)==nil {
				for i,item := range cp.Data {
					if i >= 20 { break }
					out.Cultivars = append(out.Cultivars, fmt.Sprintf("%d • %s", item.ID, item.Name))
				}
			}
		}
	}
	if culture.HasProductivity && tech.AgritecCultivarID > 0 && tech.SoilCAD > 0 && tech.PlantingStart != "" && tech.ExpectedProductivity > 0 {
		expect, ok := productivityTonsHa(tech.ExpectedProductivity, tech.ProductivityUnit)
		if ok {
			q:=url.Values{}
			q.Set("idCultura",strconv.FormatInt(culture.ID,10))
			q.Set("idCultivar",strconv.FormatInt(tech.AgritecCultivarID,10))
			q.Set("codigoIBGE",strings.TrimLeft(car.MunicipalityCode,"0"))
			q.Set("cad",strconv.Itoa(tech.SoilCAD))
			q.Set("dataPlantio",tech.PlantingStart)
			q.Set("expectativaProdutividade",strconv.FormatFloat(expect,'f',4,64))
			if car.CenterLat != 0 || car.CenterLon != 0 {
				q.Set("latitude",strconv.FormatFloat(car.CenterLat,'f',6,64))
				q.Set("longitude",strconv.FormatFloat(car.CenterLon,'f',6,64))
			}
			req,_:=bearerRequest(ctx,http.MethodGet,agritecBaseURL+"/produtividade?"+q.Encode(),token,nil)
			if resp,err:=projectHTTPClient.Do(req);err==nil {
				var pp struct{ Data struct{ Values []float64 `json:"produtividadeAlmejada"` } `json:"data"` }
				if decodeHTTPJSON(resp,&pp)==nil && len(pp.Data.Values)>0 {
					out.ProductivityRan=true
					out.ProductivityLatest=pp.Data.Values[len(pp.Data.Values)-1]
					sum:=0.0; for _,v:=range pp.Data.Values { sum+=v }
					out.ProductivityMean=sum/float64(len(pp.Data.Values))
				}
			}
		}
	}
	return out
}

func productivityTonsHa(v float64, unit string) (float64,bool) {
	u:=normalizeProjectCulture(unit)
	switch u {
	case "t/ha","ton/ha","tonelada/ha","toneladas/ha":
		return v,true
	case "kg/ha","kg ha":
		return v/1000,true
	default:
		return 0,false
	}
}

type SATVegProjectResult struct {
	Status      string    `json:"status"`
	Detail      string    `json:"detail"`
	SourceURL   string    `json:"source_url"`
	Index       string    `json:"index"`
	Count       int       `json:"count"`
	FirstDate   string    `json:"first_date"`
	LastDate    string    `json:"last_date"`
	LatestValue float64   `json:"latest_value"`
	MeanValue   float64   `json:"mean_value"`
	Dates       []string  `json:"dates"`
	Values      []float64 `json:"values"`
	UpdatedAt   string    `json:"updated_at"`
}

func projectAreaSATVegPolygon(raw string) (string,error) {
	for _, feature := range geoJSONFeatures(raw) {
		polys, err := carGeometryPolygons(feature.Geometry)
		if err != nil || len(polys)==0 || len(polys[0])==0 || len(polys[0][0])<4 { continue }
		ring:=polys[0][0]
		step:=1
		if len(ring)>180 { step=int(math.Ceil(float64(len(ring))/180)) }
		var pts []string
		for i:=0;i<len(ring);i+=step {
			if len(ring[i])<2 { continue }
			pts=append(pts,fmt.Sprintf("%.7f %.7f",ring[i][0],ring[i][1]))
		}
		if len(pts)<3 { continue }
		first:=strings.Fields(pts[0]); last:=strings.Fields(pts[len(pts)-1])
		if len(first)==2 && len(last)==2 && pts[0]!=pts[len(pts)-1] { pts=append(pts,pts[0]) }
		return strings.Join(pts,","),nil
	}
	return "",errors.New("polígono da gleba inválido para SATVeg")
}

func (a *App) runSATVegProject(ctx context.Context, p RuralProject) SATVegProjectResult {
	out:=SATVegProjectResult{Status:"not_run",SourceURL:satvegBaseURL,Index:"NDVI",UpdatedAt:time.Now().Format(time.RFC3339)}
	token:=a.settingValue("project.embrapa_token")
	if token=="" { out.Status="not_configured"; out.Detail="Token da AgroAPI/Embrapa não configurado."; return out }
	if p.AreaID<=0 { out.Status="pending"; out.Detail="Vincule uma gleba ao projeto para executar o SATVeg no polígono financiado."; return out }
	var raw string
	if err:=a.db.QueryRow(`SELECT geojson FROM project_areas WHERE id=? AND property_id=?`,p.AreaID,p.PropertyID).Scan(&raw);err!=nil {
		out.Status="pending"; out.Detail="Gleba vinculada não encontrada."; return out
	}
	polygon,err:=projectAreaSATVegPolygon(raw)
	if err!=nil { out.Status="review"; out.Detail=err.Error(); return out }
	body,_:=json.Marshal(map[string]any{"tipoPerfil":"ndvi","satelite":"comb","preFiltro":3,"filtro":"sav","parametroFiltro":4,"poligono":polygon,"todasEstatisticas":false})
	req,err:=bearerRequest(ctx,http.MethodPost,satvegBaseURL+"/seriespoligono",token,bytes.NewReader(body))
	if err!=nil { out.Status="error";out.Detail=err.Error();return out }
	resp,err:=projectHTTPClient.Do(req)
	if err!=nil { out.Status="source_unavailable";out.Detail="SATVeg indisponível nesta execução: "+err.Error();return out }
	var payload struct{ Values []float64 `json:"listaSerie"`; Dates []string `json:"listaDatas"` }
	if err:=decodeHTTPJSON(resp,&payload);err!=nil {
		if strings.Contains(err.Error(),"401")||strings.Contains(err.Error(),"403"){out.Status="not_configured"}else{out.Status="source_unavailable"}
		out.Detail="Consulta SATVeg não concluída: "+err.Error();return out
	}
	n:=len(payload.Values); if len(payload.Dates)<n { n=len(payload.Dates) }
	if n==0 { out.Status="review";out.Detail="SATVeg respondeu sem série temporal para a gleba.";return out }
	if n>500 { payload.Values=payload.Values[len(payload.Values)-500:]; payload.Dates=payload.Dates[len(payload.Dates)-500:]; n=500 }
	out.Values=payload.Values[:n];out.Dates=payload.Dates[:n];out.Count=n
	out.FirstDate=out.Dates[0];out.LastDate=out.Dates[n-1];out.LatestValue=out.Values[n-1]
	sum:=0.0;for _,v:=range out.Values{sum+=v};out.MeanValue=sum/float64(n)
	out.Status="ready";out.Detail=fmt.Sprintf("Série NDVI da gleba disponível com %d observações.",n)
	return out
}

type ANAHydroStation struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	DistanceKM float64 `json:"distance_km"`
	River      string  `json:"river"`
}

type ANAHydroProjectResult struct {
	Status        string            `json:"status"`
	Detail        string            `json:"detail"`
	SourceURL     string            `json:"source_url"`
	Stations      []ANAHydroStation `json:"stations"`
	RainValueCount int              `json:"rain_value_count"`
	RainTotalMM    float64          `json:"rain_total_mm"`
	FlowValueCount int              `json:"flow_value_count"`
	FlowMeanM3S    float64          `json:"flow_mean_m3s"`
	PeriodStart    string            `json:"period_start"`
	PeriodEnd      string            `json:"period_end"`
	UpdatedAt      string            `json:"updated_at"`
}

func (a *App) anaToken(ctx context.Context) (string,error) {
	user:=a.settingValue("project.ana_username"); pass:=a.settingValue("project.ana_password")
	if user==""||pass=="" { return "",errors.New("credencial ANA não configurada") }
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,anaBaseURL+"/OAUth/v1",nil)
	if err!=nil{return "",err}
	req.Header.Set("accept","*/*");req.Header.Set("Identificador",user);req.Header.Set("Senha",pass);req.Header.Set("User-Agent","ViaVerdeCAR/"+AppVersion)
	resp,err:=projectHTTPClient.Do(req);if err!=nil{return "",err}
	var payload struct{ Status string `json:"status"`; Items struct{ Success bool `json:"sucesso"`; Token string `json:"tokenautenticacao"` } `json:"items"` }
	if err:=decodeHTTPJSON(resp,&payload);err!=nil{return "",err}
	if !strings.EqualFold(payload.Status,"OK")||!payload.Items.Success||payload.Items.Token=="" { return "",errors.New("ANA não autenticou a credencial") }
	return payload.Items.Token,nil
}

func anaGet(ctx context.Context, token,target string,dest any) error {
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,target,nil);if err!=nil{return err}
	req.Header.Set("Accept","application/json");req.Header.Set("Authorization","Bearer "+token);req.Header.Set("User-Agent","ViaVerdeCAR/"+AppVersion)
	resp,err:=projectHTTPClient.Do(req);if err!=nil{return err}
	return decodeHTTPJSON(resp,dest)
}

func projectReferencePoint(a *App,p RuralProject,car CARResult)(float64,float64){
	if p.AreaID>0 {
		var lat,lon float64
		if a.db.QueryRow(`SELECT center_lat,center_lon FROM project_areas WHERE id=?`,p.AreaID).Scan(&lat,&lon)==nil && (lat!=0||lon!=0){return lat,lon}
	}
	return car.CenterLat,car.CenterLon
}

func numericString(v any) float64 {
	switch x:=v.(type){
	case string: f,_:=strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(x),",","."),64);return f
	case float64:return x
	case json.Number:f,_:=x.Float64();return f
	}
	return 0
}

func (a *App) runANAProject(ctx context.Context,p RuralProject,car CARResult) ANAHydroProjectResult {
	out:=ANAHydroProjectResult{Status:"not_run",SourceURL:anaBaseURL,UpdatedAt:time.Now().Format(time.RFC3339)}
	if a.settingValue("project.ana_username")==""||a.settingValue("project.ana_password")=="" {
		out.Status="not_configured";out.Detail="Credencial do HidroWebService/ANA não configurada.";return out
	}
	token,err:=a.anaToken(ctx);if err!=nil{out.Status="source_unavailable";out.Detail="Autenticação ANA não concluída: "+err.Error();return out}
	q:=url.Values{};q.Set("Unidade Federativa",strings.ToUpper(strings.TrimSpace(car.UF)))
	var inv struct{ Status string `json:"status"`; Items []map[string]any `json:"items"` }
	if err:=anaGet(ctx,token,anaBaseURL+"/HidroInventarioEstacoes/v1?"+q.Encode(),&inv);err!=nil {
		out.Status="source_unavailable";out.Detail="Inventário ANA indisponível nesta execução: "+err.Error();return out
	}
	lat0,lon0:=projectReferencePoint(a,p,car)
	for _,item:=range inv.Items {
		lat:=numericString(item["Latitude"]);lon:=numericString(item["Longitude"])
		code:=fmt.Sprint(item["codigoestacao"]);name:=fmt.Sprint(item["Estacao_Nome"]);typ:=fmt.Sprint(item["Tipo_Estacao"]);river:=fmt.Sprint(item["Bacia_Nome"])
		if code==""||code=="<nil>"||lat==0||lon==0{continue}
		d:=0.0;if lat0!=0||lon0!=0{d=carHaversineM(lat0,lon0,lat,lon)/1000}
		out.Stations=append(out.Stations,ANAHydroStation{Code:code,Name:name,Type:typ,Latitude:lat,Longitude:lon,DistanceKM:d,River:river})
	}
	for i:=0;i<len(out.Stations);i++{for j:=i+1;j<len(out.Stations);j++{if out.Stations[j].DistanceKM<out.Stations[i].DistanceKM{out.Stations[i],out.Stations[j]=out.Stations[j],out.Stations[i]}}}
	if len(out.Stations)>5{out.Stations=out.Stations[:5]}
	if len(out.Stations)==0{out.Status="review";out.Detail="ANA respondeu, mas nenhuma estação com coordenadas foi localizada para a UF.";return out}
	end:=time.Now();start:=end.AddDate(-1,0,0);out.PeriodStart=start.Format("2006-01-02");out.PeriodEnd=end.Format("2006-01-02")
	station:=out.Stations[0]
	out.RainValueCount,out.RainTotalMM=anaSeriesSummary(ctx,token,station.Code,"HidroSerieChuva/v1","Chuva_",start,end,true)
	out.FlowValueCount,out.FlowMeanM3S=anaSeriesSummary(ctx,token,station.Code,"HidroSerieVazao/v1","Vazao_",start,end,false)
	out.Status="ready";out.Detail=fmt.Sprintf("Estação ANA mais próxima: %s (%s), %.1f km.",station.Name,station.Code,station.DistanceKM)
	return out
}

func anaSeriesSummary(ctx context.Context,token,station,endpoint,prefix string,start,end time.Time,sumMode bool)(int,float64){
	q:=url.Values{}
	q.Set("Código da Estação",station);q.Set("Tipo Filtro Data","DATA_LEITURA");q.Set("Data Inicial (yyyy-MM-dd)",start.Format("2006-01-02"));q.Set("Data Final (yyyy-MM-dd)",end.Format("2006-01-02"))
	var payload struct{ Items []map[string]any `json:"items"` }
	if anaGet(ctx,token,anaBaseURL+"/"+endpoint+"?"+q.Encode(),&payload)!=nil{return 0,0}
	count:=0;total:=0.0
	for _,item:=range payload.Items {
		for k,v:=range item {
			if !strings.HasPrefix(k,prefix){continue}
			s:=fmt.Sprint(v);if s==""||s=="<nil>"{continue}
			f,err:=strconv.ParseFloat(strings.ReplaceAll(s,",","."),64);if err!=nil{continue}
			count++;total+=f
		}
	}
	if count>0&&!sumMode{total/=float64(count)}
	return count,total
}
