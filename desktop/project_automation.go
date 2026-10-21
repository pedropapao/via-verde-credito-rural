package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const projectAutomationCacheVersion = 1
const projectAutomationTTL = 12 * time.Hour

type ProjectPendingAction struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
}

type ProjectZARCResult struct {
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	SourceURL string    `json:"source_url"`
	Check     ZARCCheck `json:"check"`
	UpdatedAt string    `json:"updated_at"`
}

type RuralProjectAutomationResult struct {
	CacheVersion int                      `json:"cache_version"`
	UsedCache    bool                     `json:"used_cache"`
	ProjectID    int64                    `json:"project_id"`
	CheckedAt    string                   `json:"checked_at"`
	Preparation  ProjectPreparationResult `json:"preparation"`
	ZARC         ProjectZARCResult        `json:"zarc"`
	Agritec      AgritecProjectResult     `json:"agritec"`
	ANA          ANAHydroProjectResult    `json:"ana"`
	SATVeg       SATVegProjectResult      `json:"satveg"`
	Pending      []ProjectPendingAction   `json:"pending"`
	Scope        string                   `json:"scope"`
}

func automationPrepCheck(key,label,status,detail,source string, informational bool) ProjectPreparationCheck {
	return ProjectPreparationCheck{
		Key:key,Label:label,Status:status,Detail:detail,Source:source,
		Group:"automation",Informational:informational,
	}
}

func (a *App) GetRuralProjectAutomation(projectID int64) (RuralProjectAutomationResult,error) {
	if a==nil || a.db==nil { return RuralProjectAutomationResult{},errors.New("banco local indisponível") }
	if projectID<=0 { return RuralProjectAutomationResult{},errors.New("projeto inválido") }
	var raw,checked string
	err:=a.db.QueryRow(`SELECT result_json,checked_at FROM rural_project_automation WHERE project_id=?`,projectID).Scan(&raw,&checked)
	if err!=nil { return RuralProjectAutomationResult{},err }
	var out RuralProjectAutomationResult
	if json.Unmarshal([]byte(raw),&out)!=nil || out.ProjectID!=projectID {
		return RuralProjectAutomationResult{},errors.New("última automação do projeto está inválida")
	}
	if out.CheckedAt=="" { out.CheckedAt=checked }
	return out,nil
}

func (a *App) saveRuralProjectAutomation(out RuralProjectAutomationResult) error {
	raw,err:=json.Marshal(out);if err!=nil{return err}
	_,err=a.db.Exec(`INSERT INTO rural_project_automation(project_id,checked_at,result_json)
		VALUES(?,?,?)
		ON CONFLICT(project_id) DO UPDATE SET checked_at=excluded.checked_at,result_json=excluded.result_json`,
		out.ProjectID,out.CheckedAt,string(raw))
	return err
}

func recentProjectAutomation(out RuralProjectAutomationResult) bool {
	if out.CacheVersion!=projectAutomationCacheVersion || out.ProjectID<=0 || out.CheckedAt=="" { return false }
	t,err:=time.Parse(time.RFC3339,out.CheckedAt)
	return err==nil && time.Since(t)>=0 && time.Since(t)<=projectAutomationTTL
}

func (a *App) RunRuralProjectAutomation(projectID int64, force bool) (RuralProjectAutomationResult,error) {
	if a==nil || a.db==nil { return RuralProjectAutomationResult{},errors.New("banco local indisponível") }
	if !force {
		if cached,err:=a.GetRuralProjectAutomation(projectID);err==nil && recentProjectAutomation(cached) {
			cached.UsedCache=true
			return cached,nil
		}
	}
	project,err:=a.GetRuralProject(projectID);if err!=nil{return RuralProjectAutomationResult{},err}
	property,err:=a.GetProperty(project.PropertyID);if err!=nil{return RuralProjectAutomationResult{},err}
	prep,err:=a.PrepareRuralProject(projectID);if err!=nil{return RuralProjectAutomationResult{},err}
	tech:=prep.TechnicalData
	profile:=prep.SmartProfile

	var car CARResult
	carAvailable:=false
	if latest,e:=a.GetLatestCAR(property.ID);e==nil {
		car=latest
		carAvailable=latest.Found
	}

	out:=RuralProjectAutomationResult{
		CacheVersion:projectAutomationCacheVersion,ProjectID:projectID,
		CheckedAt:time.Now().Format(time.RFC3339),Preparation:prep,
		Scope:"Automação técnica auxiliar do projeto. Cada fonte é independente. Fonte indisponível ou integração não configurada não equivale a ausência de ocorrência nem substitui parecer técnico, enquadramento bancário ou documento oficial.",
	}

	ctxZ,cancelZ:=context.WithTimeout(context.Background(),45*time.Second)
	if profile.Agriculture && project.OperationType=="custeio" {
		out.ZARC=a.runProjectZARC(ctxZ,car,tech,carAvailable)
	} else {
		out.ZARC=ProjectZARCResult{Status:"not_applicable",Detail:"ZARC automático desta etapa é aplicado a custeio agrícola.",SourceURL:creditIntelligenceZARCURL,UpdatedAt:time.Now().Format(time.RFC3339)}
	}
	cancelZ()

	ctxA,cancelA:=context.WithTimeout(context.Background(),30*time.Second)
	if profile.Agriculture && carAvailable {
		out.Agritec=a.runAgritecProject(ctxA,car,tech)
	} else if profile.Agriculture {
		out.Agritec=AgritecProjectResult{Status:"pending",Detail:"CAR público com código IBGE ainda não disponível para a Agritec.",SourceURL:agritecBaseURL,UpdatedAt:time.Now().Format(time.RFC3339)}
	} else {
		out.Agritec=AgritecProjectResult{Status:"not_applicable",Detail:"Agritec não aplicada ao perfil atual.",SourceURL:agritecBaseURL,UpdatedAt:time.Now().Format(time.RFC3339)}
	}
	cancelA()

	ctxH,cancelH:=context.WithTimeout(context.Background(),35*time.Second)
	if profile.Irrigation && carAvailable {
		out.ANA=a.runANAProject(ctxH,project,car)
	} else if profile.Irrigation {
		out.ANA=ANAHydroProjectResult{Status:"pending",Detail:"CAR público ainda não disponível para localizar estações ANA.",SourceURL:anaBaseURL,UpdatedAt:time.Now().Format(time.RFC3339)}
	} else {
		out.ANA=ANAHydroProjectResult{Status:"not_applicable",Detail:"Análise HidroWebService ativada para projetos com perfil de irrigação.",SourceURL:anaBaseURL,UpdatedAt:time.Now().Format(time.RFC3339)}
	}
	cancelH()

	ctxS,cancelS:=context.WithTimeout(context.Background(),35*time.Second)
	if project.AreaID>0 && (profile.Agriculture||profile.Livestock) {
		out.SATVeg=a.runSATVegProject(ctxS,project)
	} else if profile.Agriculture||profile.Livestock {
		out.SATVeg=SATVegProjectResult{Status:"pending",Detail:"Vincule a gleba do projeto para executar o SATVeg no polígono correto.",SourceURL:satvegBaseURL,Index:"NDVI",UpdatedAt:time.Now().Format(time.RFC3339)}
	} else {
		out.SATVeg=SATVegProjectResult{Status:"not_applicable",Detail:"SATVeg não aplicado ao perfil atual.",SourceURL:satvegBaseURL,Index:"NDVI",UpdatedAt:time.Now().Format(time.RFC3339)}
	}
	cancelS()

	a.applyAutomationToPreparation(&out)
	out.Pending=projectPendingActions(out.Preparation.Checks,out)
	if err:=a.saveRuralProjectAutomation(out);err!=nil{return RuralProjectAutomationResult{},err}
	return out,nil
}

func (a *App) runProjectZARC(ctx context.Context,car CARResult,tech ProjectTechnicalData,carAvailable bool) ProjectZARCResult {
	out:=ProjectZARCResult{Status:"not_run",SourceURL:creditIntelligenceZARCURL,UpdatedAt:time.Now().Format(time.RFC3339)}
	if !carAvailable || strings.TrimSpace(car.MunicipalityCode)=="" {
		out.Status="pending";out.Detail="CAR público com município/código IBGE ainda não está disponível.";return out
	}
	var missing []string
	if tech.Culture==""{missing=append(missing,"cultura")}
	if tech.PlantingStart==""{missing=append(missing,"início de plantio")}
	if tech.Soil==""{missing=append(missing,"solo")}
	if tech.Cycle==""{missing=append(missing,"ciclo")}
	if len(missing)>0 {
		out.Status="pending";out.Detail="Dados insuficientes para ZARC automático: "+strings.Join(missing,", ")+"."
		return out
	}
	start,err:=parseCreditDate(tech.PlantingStart)
	if err!=nil{out.Status="review";out.Detail="Data de plantio inválida para ZARC.";return out}
	si,sf:=zarcCropSeason(start)
	sourceDir:=filepath.Join(a.dataDir,"cache","zarc_project")
	if err:=os.MkdirAll(sourceDir,0o755);err!=nil{out.Status="error";out.Detail=err.Error();return out}
	res,err:=resolveZARCResource(ctx,si,sf)
	if err!=nil{out.Status="source_unavailable";out.Detail="Catálogo MAPA/ZARC indisponível nesta execução: "+err.Error();return out}
	path:=filepath.Join(sourceDir,fmt.Sprintf("zarc_%d_%d.csv",si,sf))
	if st,statErr:=os.Stat(path);statErr!=nil||st.Size()==0||time.Since(st.ModTime())>7*24*time.Hour {
		tmp:=path+".part";_ = os.Remove(tmp)
		if err:=downloadLargeFile(ctx,res.URL,tmp);err!=nil{_ = os.Remove(tmp);out.Status="source_unavailable";out.Detail="Tabela ZARC não pôde ser baixada: "+err.Error();return out}
		_ = os.Remove(path)
		if err:=os.Rename(tmp,path);err!=nil{_ = os.Remove(tmp);out.Status="error";out.Detail=err.Error();return out}
	}
	op:=CreditOperationIntelligence{
		Product:tech.Culture,PlantingStart:tech.PlantingStart,PlantingEnd:tech.PlantingEnd,
		Soil:tech.Soil,CultivarCycle:tech.Cycle,
	}
	check,err:=scanZARCForOperation(path,res,car,op,si,sf)
	if err!=nil{out.Status="review";out.Detail="ZARC não concluído: "+err.Error();out.Check=check;return out}
	out.Check=check
	switch check.Status {
	case "Indicação localizada":
		out.Status="ready"
	case "Indicação parcial","Sem indicação no período","Sem combinação exata":
		out.Status="review"
	default:
		out.Status="review"
	}
	out.Detail=check.Status+". "+check.Message
	return out
}

func statusToPreparation(status string)(string,bool){
	switch status {
	case "ready":return "ready",false
	case "pending":return "pending",false
	case "review":return "review",false
	case "source_unavailable","not_configured","not_applicable","not_run":
		return "review",true
	default:return "review",true
	}
}

func (a *App) applyAutomationToPreparation(out *RuralProjectAutomationResult) {
	if out==nil{return}
	for i:=range out.Preparation.Checks {
		if out.Preparation.Checks[i].Key=="zarc_new_project" {
			out.Preparation.Checks[i].Informational=true
		}
	}
	if out.ZARC.Status!="not_applicable" {
		s,info:=statusToPreparation(out.ZARC.Status)
		out.Preparation.Checks=append(out.Preparation.Checks,automationPrepCheck("auto_zarc","ZARC automático",s,out.ZARC.Detail,"MAPA / ZARC",info))
	}
	if out.Agritec.Status!="not_applicable" {
		s,info:=statusToPreparation(out.Agritec.Status)
		out.Preparation.Checks=append(out.Preparation.Checks,automationPrepCheck("auto_agritec","Agritec / Embrapa",s,out.Agritec.Detail,"Embrapa AgroAPI",info))
	}
	if out.ANA.Status!="not_applicable" {
		s,info:=statusToPreparation(out.ANA.Status)
		out.Preparation.Checks=append(out.Preparation.Checks,automationPrepCheck("auto_ana","ANA / HidroWebService",s,out.ANA.Detail,"ANA",info))
	}
	if out.SATVeg.Status!="not_applicable" {
		s,info:=statusToPreparation(out.SATVeg.Status)
		out.Preparation.Checks=append(out.Preparation.Checks,automationPrepCheck("auto_satveg","SATVeg / NDVI",s,out.SATVeg.Detail,"Embrapa SATVeg",info))
	}
	recalculateProjectPreparation(&out.Preparation)
}

func recalculateProjectPreparation(prep *ProjectPreparationResult) {
	if prep==nil{return}
	prep.Ready,prep.Pending,prep.Review,prep.ReadinessPct=0,0,0,0
	total:=0
	for _,c:=range prep.Checks {
		if c.Informational{continue}
		switch c.Status {
		case "ready":prep.Ready++;total++
		case "pending":prep.Pending++;total++
		case "review":prep.Review++;total++
		}
	}
	if total>0{prep.ReadinessPct=int(float64(prep.Ready)/float64(total)*100+0.5)}
}

func projectPendingActions(checks []ProjectPreparationCheck,out RuralProjectAutomationResult) []ProjectPendingAction {
	seen:=map[string]bool{}
	var actions []ProjectPendingAction
	for _,c:=range checks {
		if c.Informational || (c.Status!="pending"&&c.Status!="review"){continue}
		if seen[c.Key]{continue};seen[c.Key]=true
		severity:="attention";if c.Status=="pending"{severity="pending"}
		actions=append(actions,ProjectPendingAction{Key:c.Key,Label:c.Label,Detail:c.Detail,Severity:severity,Source:c.Source})
	}
	for _,x:=range []struct{key,label,status,detail,source string}{
		{"integration_agritec","Configurar Agritec",out.Agritec.Status,out.Agritec.Detail,"Embrapa AgroAPI"},
		{"integration_ana","Configurar ANA",out.ANA.Status,out.ANA.Detail,"ANA HidroWebService"},
		{"integration_satveg","Configurar SATVeg",out.SATVeg.Status,out.SATVeg.Detail,"Embrapa SATVeg"},
	}{
		if x.status!="not_configured"{continue}
		actions=append(actions,ProjectPendingAction{Key:x.key,Label:x.label,Detail:x.detail,Severity:"integration",Source:x.source})
	}
	return actions
}
