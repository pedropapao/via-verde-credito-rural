package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ProjectSmartProfile struct {
	ActivityClass string   `json:"activity_class"`
	ActivityLabel string   `json:"activity_label"`
	OperationType string   `json:"operation_type"`
	OperationLabel string  `json:"operation_label"`
	Agriculture   bool     `json:"agriculture"`
	Coffee        bool     `json:"coffee"`
	Livestock     bool     `json:"livestock"`
	Irrigation    bool     `json:"irrigation"`
	Ambiguous     bool     `json:"ambiguous"`
	Rules         []string `json:"rules"`
	Message       string   `json:"message"`
}

func projectSignal(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	r := strings.NewReplacer(
		"á","a","à","a","â","a","ã","a","ä","a",
		"é","e","è","e","ê","e","ë","e",
		"í","i","ì","i","î","i","ï","i",
		"ó","o","ò","o","ô","o","õ","o","ö","o",
		"ú","u","ù","u","û","u","ü","u",
		"ç","c",
	)
	return r.Replace(v)
}

func projectHasAny(s string, values ...string) bool {
	for _, v := range values {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}

func projectOperationLabel(v string) string {
	switch strings.TrimSpace(v) {
	case "custeio":
		return "Custeio"
	case "investimento":
		return "Investimento"
	case "aquisicao":
		return "Aquisição"
	case "renegociacao":
		return "Renegociação"
	case "outro":
		return "Outro"
	default:
		return "Não informado"
	}
}

func buildProjectSmartProfile(p RuralProject) ProjectSmartProfile {
	s := projectSignal(strings.Join([]string{p.Name, p.Activity, p.CreditLine}, " "))
	out := ProjectSmartProfile{
		OperationType: p.OperationType,
		OperationLabel: projectOperationLabel(p.OperationType),
	}
	out.Irrigation = projectHasAny(s, "irrig", "pivo", "gotej", "aspers", "microaspers")
	out.Coffee = projectHasAny(s, "cafe", "cafeic")
	out.Livestock = projectHasAny(s, "pecuar", "bovin", "gado", "leite", "confin", "animal", "bezer", "novilh", "vaca")
	out.Agriculture = out.Coffee || projectHasAny(s, "agric", "soja", "milho", "sorgo", "trigo", "feij", "arroz", "algod", "cana", "lavour", "cultivo", "hort", "frut")

	switch {
	case out.Irrigation && out.Coffee:
		out.ActivityClass, out.ActivityLabel = "irrigated_coffee", "Café / irrigação"
	case out.Irrigation && out.Agriculture:
		out.ActivityClass, out.ActivityLabel = "irrigated_agriculture", "Agricultura irrigada"
	case out.Livestock && out.Agriculture:
		out.ActivityClass, out.ActivityLabel = "mixed", "Atividade mista"
	case out.Irrigation:
		out.ActivityClass, out.ActivityLabel = "irrigation", "Irrigação"
	case out.Coffee:
		out.ActivityClass, out.ActivityLabel = "coffee", "Café"
	case out.Livestock:
		out.ActivityClass, out.ActivityLabel = "livestock", "Pecuária"
	case out.Agriculture:
		out.ActivityClass, out.ActivityLabel = "agriculture", "Agricultura"
	default:
		out.ActivityClass, out.ActivityLabel = "unknown", "Atividade não classificada"
		out.Ambiguous = true
	}

	out.Rules = append(out.Rules, "Documentação base do produtor e do imóvel")
	if out.Agriculture && p.OperationType == "custeio" {
		out.Rules = append(out.Rules, "ZARC aplicável ao custeio agrícola quando houver parâmetros técnicos suficientes")
	}
	if out.Irrigation {
		out.Rules = append(out.Rules, "Regularização hídrica e relevo da gleba")
	}
	if out.Livestock {
		out.Rules = append(out.Rules, "Trânsito animal conforme a natureza da operação")
	}
	if p.OperationType == "investimento" || p.OperationType == "aquisicao" {
		out.Rules = append(out.Rules, "Orçamento da aquisição/investimento e conferência documental da execução")
	}
	if p.OperationType == "renegociacao" {
		out.Rules = append(out.Rules, "Justificativa/laudo técnico deve ser conferido para a renegociação")
	}
	if out.Ambiguous {
		out.Message = "A atividade não pôde ser classificada com segurança. Somente regras gerais foram aplicadas; confira a descrição da atividade."
	} else {
		out.Message = "Perfil detectado automaticamente a partir da atividade, nome, linha e tipo de operação. Regras específicas foram aplicadas sem alterar documentos ou dados do imóvel."
	}
	return out
}

func projectDocumentByType(center *PropertyDocumentCenter, docType string) *DocumentChecklistItem {
	if center == nil {
		return nil
	}
	for i := range center.Items {
		if center.Items[i].DocType == docType {
			return &center.Items[i]
		}
	}
	return nil
}

func projectSmartDocumentCheck(center *PropertyDocumentCenter, docType, label, requirement, reason string) ProjectPreparationCheck {
	item := projectDocumentByType(center, docType)
	source := "Central de Documentos"
	if item == nil {
		if center == nil {
			return smartPrepCheck("doc_"+docType, label, "review", "Central de Documentos indisponível para confirmar esta exigência. "+reason, source)
		}
		return smartPrepCheck("doc_"+docType, label, "review", "Tipo documental não localizado no checklist atual. "+reason, source)
	}
	if item.Current != nil && item.Status != "expired" {
		return smartPrepCheck("doc_"+docType, label, "ready", "Documento atual disponível: "+item.Current.OriginalName+". "+reason, source)
	}
	if item.Status == "expired" {
		return smartPrepCheck("doc_"+docType, label, "pending", "Documento existente está vencido/desatualizado. "+reason, source)
	}
	if item.Status == "not_applicable" {
		if requirement == "required" {
			return smartPrepCheck("doc_"+docType, label, "review", "A Central de Documentos está marcada como “não se aplica”, mas a regra deste projeto indica necessidade de conferência. "+reason, source)
		}
		return smartPrepCheck("doc_"+docType, label, "ready", "Marcado como não aplicável na Central de Documentos. "+reason, source)
	}
	if requirement == "required" {
		return smartPrepCheck("doc_"+docType, label, "pending", "Documento necessário para este perfil de projeto ainda não está disponível. "+reason, source)
	}
	return smartPrepCheck("doc_"+docType, label, "review", "Aplicabilidade/conteúdo deve ser conferido para este projeto. "+reason, source)
}

func smartPrepCheck(key, label, status, detail, source string) ProjectPreparationCheck {
	return ProjectPreparationCheck{Key:key, Label:label, Status:status, Detail:detail, Source:source, Group:"smart"}
}

func projectTerrainCheck(a *App, p RuralProject) ProjectPreparationCheck {
	if p.AreaID <= 0 {
		return smartPrepCheck("terrain", "Relevo da gleba", "review", "Análise de relevo aplicável ao perfil de irrigação, mas nenhuma gleba está vinculada ao projeto.", "Terrain Tiles / área do projeto")
	}
	var raw string
	err := a.db.QueryRow(`SELECT terrain_json FROM project_area_terrain WHERE area_id=?`, p.AreaID).Scan(&raw)
	if err != nil || strings.TrimSpace(raw) == "" {
		return smartPrepCheck("terrain", "Relevo da gleba", "review", "Consulta de relevo ainda não realizada para a gleba vinculada.", "Terrain Tiles / área do projeto")
	}
	var metric TerrainMetric
	if json.Unmarshal([]byte(raw), &metric) != nil || !metric.Available {
		return smartPrepCheck("terrain", "Relevo da gleba", "review", "Resultado de relevo armazenado não está disponível para uso técnico; atualize a análise da gleba.", "Terrain Tiles / área do projeto")
	}
	detail := fmt.Sprintf("Relevo disponível: altitude média %.0f m, declividade média %.1f%%, confiança %s.", metric.ElevationMeanM, metric.MeanSlopePct, metric.Confidence)
	return smartPrepCheck("terrain", "Relevo da gleba", "ready", detail, metric.Source)
}

func (a *App) projectSmartChecks(p RuralProject, center *PropertyDocumentCenter) (ProjectSmartProfile, []ProjectPreparationCheck) {
	profile := buildProjectSmartProfile(p)
	checks := []ProjectPreparationCheck{
		projectSmartDocumentCheck(center, "producer_id", "Documentos do produtor", "required", "Base documental do dossiê do projeto."),
		projectSmartDocumentCheck(center, "registry", "Matrícula / certidão do imóvel", "required", "Conferência do vínculo e da área do imóvel."),
		projectSmartDocumentCheck(center, "ccir", "CCIR", "required", "Conferência cadastral rural do imóvel."),
		projectSmartDocumentCheck(center, "itr", "ITR / DITR / recibo", "required", "Conferência fiscal do imóvel rural."),
	}

	if center != nil {
		switch strings.TrimSpace(center.Context.Tenure) {
		case "arrendado", "cessao", "comodato", "misto":
			checks = append(checks, projectSmartDocumentCheck(center, "lease", "Instrumento de uso/posse", "required", "O contexto documental informa uso por "+center.Context.Tenure+"."))
		}
	}

	if profile.Irrigation {
		checks = append(checks,
			projectSmartDocumentCheck(center, "outorga", "Outorga / regularização hídrica", "required", "A atividade indica irrigação ou uso de água; a situação deve ser conferida antes da conclusão técnica."),
			projectTerrainCheck(a, p),
		)
	}

	if profile.Livestock {
		req := "review"
		reason := "A atividade pecuária pode envolver trânsito animal; confirme a necessidade de GTA."
		if p.OperationType == "aquisicao" {
			req = "required"
			reason = "Aquisição pecuária indica necessidade de conferir documentação de trânsito animal/GTA."
		}
		checks = append(checks, projectSmartDocumentCheck(center, "gta", "GTA / trânsito animal", req, reason))
	}

	switch p.OperationType {
	case "investimento", "aquisicao":
		checks = append(checks,
			projectSmartDocumentCheck(center, "budget", "Orçamento", "required", "Investimento/aquisição requer orçamento para montagem e conferência da proposta."),
			projectSmartDocumentCheck(center, "invoice", "Nota fiscal", "review", "A exigência da nota fiscal depende da etapa da operação e da instituição financeira."),
			projectSmartDocumentCheck(center, "technical_report", "Laudo / documento técnico", "review", "Confira se o investimento exige especificação, laudo ou responsabilidade técnica."),
		)
	case "custeio":
		checks = append(checks, projectSmartDocumentCheck(center, "budget", "Orçamento / base de custos", "review", "Confira orçamentos ou memória de custos utilizados na proposta de custeio."))
	case "renegociacao":
		checks = append(checks, projectSmartDocumentCheck(center, "technical_report", "Laudo / justificativa técnica", "review", "Renegociação/prorrogação pode depender de comprovação e justificativa técnica; confirmar a exigência do agente financeiro."))
	}

	if profile.Agriculture && p.OperationType == "custeio" {
		checks = append(checks, smartPrepCheck(
			"zarc_new_project", "ZARC do novo projeto", "review",
			"O perfil indica custeio agrícola. A consulta ZARC do novo projeto ainda não foi executada porque o cadastro atual não possui todos os parâmetros técnicos de plantio (data/ciclo/solo) necessários para uma conclusão automática.",
			"MAPA / ZARC",
		))
	}
	if profile.Ambiguous {
		checks = append(checks, smartPrepCheck(
			"rule_context", "Classificação da atividade", "review",
			"Atividade não reconhecida com segurança. Informe uma atividade mais específica, por exemplo Café, Agricultura, Pecuária ou Irrigação, para aplicar regras direcionadas.",
			"Motor local de regras",
		))
	}
	return profile, checks
}
