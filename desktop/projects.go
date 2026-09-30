package main

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type RuralProject struct {
	ID int64 `json:"id"`
	PropertyID int64 `json:"property_id"`
	PropertyName string `json:"property_name"`
	ClientID int64 `json:"client_id"`
	ClientName string `json:"client_name"`
	Name string `json:"name"`
	Bank string `json:"bank"`
	CreditLine string `json:"credit_line"`
	OperationType string `json:"operation_type"`
	Activity string `json:"activity"`
	RequestedAmount float64 `json:"requested_amount"`
	TermMonths int `json:"term_months"`
	InterestRatePct float64 `json:"interest_rate_pct"`
	AreaHa float64 `json:"area_ha"`
	AreaID int64 `json:"area_id"`
	Status string `json:"status"`
	Notes string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ProjectPreparationCheck struct {
	Key string `json:"key"`
	Label string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Source string `json:"source"`
	Group string `json:"group"`
}

type ProjectPreparationResult struct {
	Project RuralProject `json:"project"`
	Property Property `json:"property"`
	GeneratedAt string `json:"generated_at"`
	ReadinessPct int `json:"readiness_pct"`
	Ready int `json:"ready"`
	Pending int `json:"pending"`
	Review int `json:"review"`
	Checks []ProjectPreparationCheck `json:"checks"`
	DocumentSummary PropertyDocumentSummary `json:"document_summary"`
	PublicCreditOps int `json:"public_credit_operations"`
	PublicCreditValue float64 `json:"public_credit_value"`
	EnvironmentalHits int `json:"environmental_hits"`
	SmartProfile ProjectSmartProfile `json:"smart_profile"`
	Scope string `json:"scope"`
}

func validRuralProjectStatus(v string) bool {
	switch strings.TrimSpace(v) {
	case "draft","review","sent","bank_pending","approved","contracted","released","rejected","cancelled":
		return true
	default:
		return false
	}
}

func validRuralProjectOperation(v string) bool {
	switch strings.TrimSpace(v) {
	case "","custeio","investimento","aquisicao","renegociacao","outro":
		return true
	default:
		return false
	}
}

func normalizeRuralProject(p RuralProject) (RuralProject,error) {
	p.Name=strings.TrimSpace(p.Name)
	p.Bank=strings.TrimSpace(p.Bank)
	p.CreditLine=strings.TrimSpace(p.CreditLine)
	p.OperationType=strings.ToLower(strings.TrimSpace(p.OperationType))
	p.Activity=strings.TrimSpace(p.Activity)
	p.Status=strings.TrimSpace(p.Status)
	p.Notes=strings.TrimSpace(p.Notes)
	if p.PropertyID<=0 { return RuralProject{},errors.New("selecione o imóvel do projeto") }
	if p.Name=="" { return RuralProject{},errors.New("informe o nome do projeto") }
	if p.Status=="" { p.Status="draft" }
	if !validRuralProjectStatus(p.Status) { return RuralProject{},errors.New("status do projeto inválido") }
	if !validRuralProjectOperation(p.OperationType) { return RuralProject{},errors.New("tipo de operação inválido") }
	if p.RequestedAmount<0 || p.InterestRatePct<0 || p.AreaHa<0 || p.TermMonths<0 {
		return RuralProject{},errors.New("valores do projeto não podem ser negativos")
	}
	return p,nil
}

func (a *App) SaveRuralProject(p RuralProject) (RuralProject,error) {
	if a==nil || a.db==nil { return RuralProject{},errors.New("banco local indisponível") }
	var err error
	p,err=normalizeRuralProject(p); if err!=nil { return RuralProject{},err }
	property,err:=a.GetProperty(p.PropertyID); if err!=nil { return RuralProject{},err }
	if p.AreaID>0 {
		var owner int64
		if err:=a.db.QueryRow(`SELECT property_id FROM project_areas WHERE id=?`,p.AreaID).Scan(&owner); err!=nil {
			return RuralProject{},errors.New("área/gleba selecionada não foi localizada")
		}
		if owner!=p.PropertyID { return RuralProject{},errors.New("a área/gleba selecionada pertence a outro imóvel") }
	}
	now:=time.Now().Format(time.RFC3339)
	if p.ID==0 {
		res,err:=a.db.Exec(`INSERT INTO rural_projects(
			property_id,name,bank,credit_line,operation_type,activity,requested_amount,term_months,
			interest_rate_pct,area_ha,area_id,status,notes,created_at,updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			p.PropertyID,p.Name,p.Bank,p.CreditLine,p.OperationType,p.Activity,p.RequestedAmount,p.TermMonths,
			p.InterestRatePct,p.AreaHa,p.AreaID,p.Status,p.Notes,now,now)
		if err!=nil { return RuralProject{},err }
		p.ID,_=res.LastInsertId(); p.CreatedAt=now
	} else {
		var existing int64
		if err:=a.db.QueryRow(`SELECT property_id FROM rural_projects WHERE id=?`,p.ID).Scan(&existing); err!=nil {
			if errors.Is(err,sql.ErrNoRows) { return RuralProject{},errors.New("projeto não localizado") }
			return RuralProject{},err
		}
		_,err=a.db.Exec(`UPDATE rural_projects SET property_id=?,name=?,bank=?,credit_line=?,operation_type=?,activity=?,
			requested_amount=?,term_months=?,interest_rate_pct=?,area_ha=?,area_id=?,status=?,notes=?,updated_at=? WHERE id=?`,
			p.PropertyID,p.Name,p.Bank,p.CreditLine,p.OperationType,p.Activity,p.RequestedAmount,p.TermMonths,
			p.InterestRatePct,p.AreaHa,p.AreaID,p.Status,p.Notes,now,p.ID)
		if err!=nil { return RuralProject{},err }
	}
	p.PropertyName=property.Name; p.ClientID=property.ClientID; p.ClientName=property.ClientName; p.UpdatedAt=now
	return p,nil
}

func (a *App) ListRuralProjects(propertyID int64) ([]RuralProject,error) {
	if a==nil || a.db==nil { return nil,errors.New("banco local indisponível") }
	q:=`SELECT rp.id,rp.property_id,p.name,p.client_id,c.name,rp.name,rp.bank,rp.credit_line,
		rp.operation_type,rp.activity,rp.requested_amount,rp.term_months,rp.interest_rate_pct,rp.area_ha,
		rp.area_id,rp.status,rp.notes,rp.created_at,rp.updated_at
		FROM rural_projects rp JOIN properties p ON p.id=rp.property_id JOIN clients c ON c.id=p.client_id`
	var rows *sql.Rows
	var err error
	if propertyID>0 { rows,err=a.db.Query(q+` WHERE rp.property_id=? ORDER BY rp.updated_at DESC,rp.id DESC`,propertyID) } else {
		rows,err=a.db.Query(q+` ORDER BY rp.updated_at DESC,rp.id DESC`)
	}
	if err!=nil { return nil,err }; defer rows.Close()
	var out []RuralProject
	for rows.Next() {
		var p RuralProject
		if err:=rows.Scan(&p.ID,&p.PropertyID,&p.PropertyName,&p.ClientID,&p.ClientName,&p.Name,&p.Bank,&p.CreditLine,
			&p.OperationType,&p.Activity,&p.RequestedAmount,&p.TermMonths,&p.InterestRatePct,&p.AreaHa,&p.AreaID,
			&p.Status,&p.Notes,&p.CreatedAt,&p.UpdatedAt); err!=nil { return nil,err }
		out=append(out,p)
	}
	return out,rows.Err()
}

func (a *App) GetRuralProject(id int64) (RuralProject,error) {
	if a==nil || a.db==nil { return RuralProject{},errors.New("banco local indisponível") }
	if id<=0 { return RuralProject{},errors.New("projeto inválido") }
	var p RuralProject
	err:=a.db.QueryRow(`SELECT rp.id,rp.property_id,pr.name,pr.client_id,c.name,rp.name,rp.bank,rp.credit_line,
		rp.operation_type,rp.activity,rp.requested_amount,rp.term_months,rp.interest_rate_pct,rp.area_ha,
		rp.area_id,rp.status,rp.notes,rp.created_at,rp.updated_at
		FROM rural_projects rp JOIN properties pr ON pr.id=rp.property_id JOIN clients c ON c.id=pr.client_id
		WHERE rp.id=?`,id).Scan(&p.ID,&p.PropertyID,&p.PropertyName,&p.ClientID,&p.ClientName,&p.Name,&p.Bank,&p.CreditLine,
		&p.OperationType,&p.Activity,&p.RequestedAmount,&p.TermMonths,&p.InterestRatePct,&p.AreaHa,&p.AreaID,
		&p.Status,&p.Notes,&p.CreatedAt,&p.UpdatedAt)
	if errors.Is(err,sql.ErrNoRows) { return RuralProject{},errors.New("projeto não localizado") }
	return p,err
}

func (a *App) DeleteRuralProject(id int64) error {
	if a==nil || a.db==nil { return errors.New("banco local indisponível") }
	if id<=0 { return errors.New("projeto inválido") }
	res,err:=a.db.Exec(`DELETE FROM rural_projects WHERE id=?`,id); if err!=nil { return err }
	n,_:=res.RowsAffected(); if n==0 { return errors.New("projeto não localizado") }
	return nil
}

func prepCheck(key,label,status,detail,source string) ProjectPreparationCheck {
	return ProjectPreparationCheck{Key:key,Label:label,Status:status,Detail:detail,Source:source,Group:"base"}
}

func projectFieldsReady(p RuralProject) (bool,string) {
	var missing []string
	if strings.TrimSpace(p.Bank)=="" { missing=append(missing,"banco") }
	if strings.TrimSpace(p.OperationType)=="" { missing=append(missing,"tipo de operação") }
	if strings.TrimSpace(p.Activity)=="" { missing=append(missing,"atividade") }
	if p.RequestedAmount<=0 { missing=append(missing,"valor") }
	if len(missing)==0 { return true,"Dados essenciais do projeto preenchidos." }
	return false,"Falta informar: "+strings.Join(missing,", ")+"."
}

func (a *App) PrepareRuralProject(id int64) (ProjectPreparationResult,error) {
	project,err:=a.GetRuralProject(id); if err!=nil { return ProjectPreparationResult{},err }
	property,err:=a.GetProperty(project.PropertyID); if err!=nil { return ProjectPreparationResult{},err }
	out:=ProjectPreparationResult{Project:project,Property:property,GeneratedAt:time.Now().Format(time.RFC3339),
		Scope:"Prontidão operacional do cadastro local. Não representa aprovação bancária, limite de crédito, enquadramento definitivo ou parecer técnico."}

	if ok,d:=projectFieldsReady(project); ok { out.Checks=append(out.Checks,prepCheck("project_data","Dados do projeto","ready",d,"Cadastro local")) } else {
		out.Checks=append(out.Checks,prepCheck("project_data","Dados do projeto","pending",d,"Cadastro local"))
	}

	var car CARResult
	carLoaded:=false
	if strings.TrimSpace(property.CARNumber)=="" {
		out.Checks=append(out.Checks,prepCheck("car","CAR do imóvel","pending","Imóvel sem CAR vinculado.","Cadastro local"))
	} else if latest,e:=a.GetLatestCAR(property.ID); e==nil {
		car=latest; carLoaded=true
		if latest.Found { out.Checks=append(out.Checks,prepCheck("car","CAR do imóvel","ready","Última consulta pública do CAR está armazenada.","SICAR")) } else {
			out.Checks=append(out.Checks,prepCheck("car","CAR do imóvel","review","Existe CAR vinculado, mas a última consulta não confirmou o cadastro.","SICAR"))
		}
	} else {
		out.Checks=append(out.Checks,prepCheck("car","CAR do imóvel","review","CAR cadastrado, mas ainda sem consulta pública salva.","SICAR"))
	}

	if carLoaded && car.HasGeometry && strings.TrimSpace(car.GeoJSON)!="" {
		out.Checks=append(out.Checks,prepCheck("geometry","Geometria","ready","Perímetro público disponível para mapa, cruzamentos e KML.","SICAR"))
	} else {
		out.Checks=append(out.Checks,prepCheck("geometry","Geometria","review","Geometria pública ainda não está disponível no cadastro local.","SICAR"))
	}

	if project.AreaID>0 {
		var name string; var area,inside float64
		if e:=a.db.QueryRow(`SELECT name,area_ha,inside_car_pct FROM project_areas WHERE id=? AND property_id=?`,project.AreaID,project.PropertyID).Scan(&name,&area,&inside); e==nil {
			status:="ready"; detail:=fmt.Sprintf("Gleba %q vinculada com %.2f ha.",name,area)
			if inside>0 && inside<99 { status="review"; detail+=fmt.Sprintf(" %.1f%% da gleba está dentro do CAR; conferir limite.",inside) }
			out.Checks=append(out.Checks,prepCheck("area","Área do projeto",status,detail,"Área/gleba local"))
		} else { out.Checks=append(out.Checks,prepCheck("area","Área do projeto","pending","A gleba vinculada não foi localizada.","Área/gleba local")) }
	} else if project.AreaHa>0 {
		status:="review"; detail:=fmt.Sprintf("Área informada: %.2f ha. Ainda não há polígono de gleba vinculado.",project.AreaHa)
		if carLoaded && car.AreaHa>0 && project.AreaHa>car.AreaHa+0.01 { status="pending"; detail=fmt.Sprintf("Área do projeto (%.2f ha) supera a área do CAR (%.2f ha).",project.AreaHa,car.AreaHa) }
		out.Checks=append(out.Checks,prepCheck("area","Área do projeto",status,detail,"Cadastro local"))
	} else { out.Checks=append(out.Checks,prepCheck("area","Área do projeto","pending","Informe a área ou vincule uma gleba ao projeto.","Cadastro local")) }

	if carLoaded {
		env:=car.Environment
		out.EnvironmentalHits=env.IBAMAEmbargoCount+env.IndigenousCount+env.FederalUCCount
		if env.MCRListed { out.EnvironmentalHits++ }
		checked:=env.IBAMAChecked||env.FUNAIChecked||env.ICMBioChecked||env.MCRChecked
		if !checked { out.Checks=append(out.Checks,prepCheck("environment","Análise ambiental","review","Não há resultado ambiental armazenado na última consulta do CAR.","Bases ambientais")) } else if out.EnvironmentalHits>0 {
			out.Checks=append(out.Checks,prepCheck("environment","Análise ambiental","review",fmt.Sprintf("%d ocorrência(s) ou sinal(is) exigem conferência técnica.",out.EnvironmentalHits),"IBAMA/FUNAI/ICMBio/MMA"))
		} else { out.Checks=append(out.Checks,prepCheck("environment","Análise ambiental","ready","Nenhuma ocorrência foi registrada nas bases que responderam na última consulta.","IBAMA/FUNAI/ICMBio/MMA")) }
	} else { out.Checks=append(out.Checks,prepCheck("environment","Análise ambiental","review","Consulte o CAR para executar e armazenar os cruzamentos ambientais.","Bases ambientais")) }

	var documentCenter *PropertyDocumentCenter
	if center,e:=a.GetPropertyDocumentCenter(project.PropertyID); e==nil {
		documentCenter=&center
		out.DocumentSummary=center.Summary
		pending:=center.Summary.Pending+center.Summary.Expired
		if pending>0 { out.Checks=append(out.Checks,prepCheck("documents","Documentos","pending",fmt.Sprintf("%d pendência(s) documental(is) e %d item(ns) para conferir.",pending,center.Summary.Review),"Central de Documentos")) } else if center.Summary.Review>0 {
			out.Checks=append(out.Checks,prepCheck("documents","Documentos","review",fmt.Sprintf("%d item(ns) precisam de conferência.",center.Summary.Review),"Central de Documentos"))
		} else { out.Checks=append(out.Checks,prepCheck("documents","Documentos","ready",fmt.Sprintf("%d documento(s) recebido(s); nenhuma pendência ativa no checklist atual.",center.Summary.Received),"Central de Documentos")) }
	} else { out.Checks=append(out.Checks,prepCheck("documents","Documentos","review","Não foi possível carregar a Central de Documentos nesta execução.","Central de Documentos")) }

	if strings.TrimSpace(property.CARNumber)!="" && a.dataDir!="" {
		cachePath:=filepath.Join(a.dataDir,"cache","sicor_xray",safeFilePart(property.CARNumber)+".json")
		if x,ok:=loadSICORXRayCache(cachePath,30*24*time.Hour); ok {
			out.PublicCreditOps=x.OperationCount; out.PublicCreditValue=x.TotalCreditValue
			out.Checks=append(out.Checks,prepCheck("credit","Crédito rural anterior","ready",fmt.Sprintf("%d operação(ões) pública(s) localizada(s), total R$ %.2f.",x.OperationCount,x.TotalCreditValue),"SICOR/BCB"))
		} else { out.Checks=append(out.Checks,prepCheck("credit","Crédito rural anterior","review","Não há Raio X SICOR recente em cache. Atualize a análise de Crédito Rural quando necessário.","SICOR/BCB")) }
	} else { out.Checks=append(out.Checks,prepCheck("credit","Crédito rural anterior","review","CAR ainda não disponível para cruzamento com o SICOR.","SICOR/BCB")) }

	profile,smartChecks:=a.projectSmartChecks(project,documentCenter)
	out.SmartProfile=profile
	out.Checks=append(out.Checks,smartChecks...)

	total:=0
	for _,c:=range out.Checks {
		switch c.Status {
		case "ready": out.Ready++; total++
		case "pending": out.Pending++; total++
		case "review": out.Review++; total++
		}
	}
	if total>0 { out.ReadinessPct=int(float64(out.Ready)/float64(total)*100+0.5) }
	return out,nil
}
