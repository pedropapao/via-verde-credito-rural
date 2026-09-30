package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ProjectTechnicalItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitValue   float64 `json:"unit_value"`
	Supplier    string  `json:"supplier"`
}

type ProjectTechnicalData struct {
	ProjectID            int64                  `json:"project_id"`
	Culture              string                 `json:"culture"`
	CropSeason           string                 `json:"crop_season"`
	PlantingStart        string                 `json:"planting_start"`
	PlantingEnd          string                 `json:"planting_end"`
	ExpectedProductivity float64                `json:"expected_productivity"`
	ProductivityUnit     string                 `json:"productivity_unit"`
	Soil                 string                 `json:"soil"`
	Cycle                string                 `json:"cycle"`
	BenefitedAreaHa      float64                `json:"benefited_area_ha"`

	Items []ProjectTechnicalItem `json:"items"`

	AnimalCount      int     `json:"animal_count"`
	AnimalCategory   string  `json:"animal_category"`
	AverageWeightKg  float64 `json:"average_weight_kg"`
	DailyGainKg      float64 `json:"daily_gain_kg"`
	CycleDays        int     `json:"cycle_days"`
	TargetArroba     float64 `json:"target_arroba"`

	IrrigatedAreaHa  float64 `json:"irrigated_area_ha"`
	IrrigationSystem string  `json:"irrigation_system"`
	FlowM3H          float64 `json:"flow_m3h"`
	WaterSource      string  `json:"water_source"`

	TechnicalPurpose string `json:"technical_purpose"`
	Notes            string `json:"notes"`
	UpdatedAt        string `json:"updated_at"`
}

func normalizeProjectTechnicalData(v ProjectTechnicalData) (ProjectTechnicalData, error) {
	v.Culture = strings.TrimSpace(v.Culture)
	v.CropSeason = strings.TrimSpace(v.CropSeason)
	v.PlantingStart = strings.TrimSpace(v.PlantingStart)
	v.PlantingEnd = strings.TrimSpace(v.PlantingEnd)
	v.ProductivityUnit = strings.TrimSpace(v.ProductivityUnit)
	v.Soil = strings.TrimSpace(v.Soil)
	v.Cycle = strings.TrimSpace(v.Cycle)
	v.AnimalCategory = strings.TrimSpace(v.AnimalCategory)
	v.IrrigationSystem = strings.TrimSpace(v.IrrigationSystem)
	v.WaterSource = strings.TrimSpace(v.WaterSource)
	v.TechnicalPurpose = strings.TrimSpace(v.TechnicalPurpose)
	v.Notes = strings.TrimSpace(v.Notes)

	if v.ProjectID <= 0 {
		return ProjectTechnicalData{}, errors.New("projeto inválido")
	}
	if v.ExpectedProductivity < 0 || v.BenefitedAreaHa < 0 || v.AverageWeightKg < 0 ||
		v.DailyGainKg < 0 || v.TargetArroba < 0 || v.IrrigatedAreaHa < 0 || v.FlowM3H < 0 ||
		v.AnimalCount < 0 || v.CycleDays < 0 {
		return ProjectTechnicalData{}, errors.New("dados técnicos não podem conter valores negativos")
	}
	if len(v.Items) > 100 {
		return ProjectTechnicalData{}, errors.New("quantidade de itens técnicos acima do limite")
	}
	clean := make([]ProjectTechnicalItem, 0, len(v.Items))
	for _, item := range v.Items {
		item.Description = strings.TrimSpace(item.Description)
		item.Unit = strings.TrimSpace(item.Unit)
		item.Supplier = strings.TrimSpace(item.Supplier)
		if item.Quantity < 0 || item.UnitValue < 0 {
			return ProjectTechnicalData{}, errors.New("itens técnicos não podem conter valores negativos")
		}
		if item.Description == "" && item.Quantity == 0 && item.UnitValue == 0 && item.Supplier == "" {
			continue
		}
		clean = append(clean, item)
	}
	v.Items = clean
	return v, nil
}

func (a *App) SaveRuralProjectTechnicalData(v ProjectTechnicalData) (ProjectTechnicalData, error) {
	if a == nil || a.db == nil {
		return ProjectTechnicalData{}, errors.New("banco local indisponível")
	}
	var err error
	v, err = normalizeProjectTechnicalData(v)
	if err != nil {
		return ProjectTechnicalData{}, err
	}
	if _, err := a.GetRuralProject(v.ProjectID); err != nil {
		return ProjectTechnicalData{}, err
	}
	v.UpdatedAt = time.Now().Format(time.RFC3339)
	raw, err := json.Marshal(v)
	if err != nil {
		return ProjectTechnicalData{}, err
	}
	_, err = a.db.Exec(`INSERT INTO rural_project_technical_data(project_id,data_json,updated_at)
		VALUES(?,?,?)
		ON CONFLICT(project_id) DO UPDATE SET data_json=excluded.data_json,updated_at=excluded.updated_at`,
		v.ProjectID, string(raw), v.UpdatedAt)
	if err != nil {
		return ProjectTechnicalData{}, err
	}
	return v, nil
}

func (a *App) GetRuralProjectTechnicalData(projectID int64) (ProjectTechnicalData, error) {
	if a == nil || a.db == nil {
		return ProjectTechnicalData{}, errors.New("banco local indisponível")
	}
	if projectID <= 0 {
		return ProjectTechnicalData{}, errors.New("projeto inválido")
	}
	if _, err := a.GetRuralProject(projectID); err != nil {
		return ProjectTechnicalData{}, err
	}
	var raw, updated string
	err := a.db.QueryRow(`SELECT data_json,updated_at FROM rural_project_technical_data WHERE project_id=?`, projectID).Scan(&raw, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return ProjectTechnicalData{ProjectID: projectID}, nil
	}
	if err != nil {
		return ProjectTechnicalData{}, err
	}
	var out ProjectTechnicalData
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return ProjectTechnicalData{}, errors.New("dados técnicos do projeto estão inválidos")
		}
	}
	out.ProjectID = projectID
	if strings.TrimSpace(out.UpdatedAt) == "" {
		out.UpdatedAt = updated
	}
	return out, nil
}

func projectTechnicalBudgetTotal(v ProjectTechnicalData) float64 {
	total := 0.0
	for _, item := range v.Items {
		total += item.Quantity * item.UnitValue
	}
	return total
}

func projectTechnicalChecks(p RuralProject, profile ProjectSmartProfile, data ProjectTechnicalData) []ProjectPreparationCheck {
	var out []ProjectPreparationCheck

	if profile.Agriculture {
		missing := []string{}
		if data.Culture == "" {
			missing = append(missing, "cultura")
		}
		if data.BenefitedAreaHa <= 0 {
			missing = append(missing, "área beneficiada")
		}
		if p.OperationType == "custeio" {
			if data.CropSeason == "" {
				missing = append(missing, "safra")
			}
			if data.PlantingStart == "" {
				missing = append(missing, "início de plantio")
			}
			if data.ExpectedProductivity <= 0 {
				missing = append(missing, "produtividade esperada")
			} else if data.ProductivityUnit == "" {
				missing = append(missing, "unidade da produtividade")
			}
		}
		if p.AreaHa > 0 && data.BenefitedAreaHa > p.AreaHa+0.01 {
			out = append(out, technicalPrepCheck("technical_agriculture", "Dados agrícolas", "pending",
				fmt.Sprintf("Área beneficiada (%.2f ha) supera a área informada do projeto (%.2f ha).", data.BenefitedAreaHa, p.AreaHa)))
		} else if len(missing) == 0 {
			out = append(out, technicalPrepCheck("technical_agriculture", "Dados agrícolas", "ready",
				fmt.Sprintf("Cultura %s, área beneficiada %.2f ha e parâmetros principais preenchidos.", data.Culture, data.BenefitedAreaHa)))
		} else {
			out = append(out, technicalPrepCheck("technical_agriculture", "Dados agrícolas", "pending",
				"Falta informar: "+strings.Join(missing, ", ") + "."))
		}

		if p.OperationType == "custeio" {
			if data.Culture != "" && data.PlantingStart != "" && data.Soil != "" && data.Cycle != "" {
				out = append(out, technicalPrepCheck("technical_zarc_input", "Parâmetros para ZARC", "ready",
					"Dados mínimos de cultura, plantio, solo e ciclo estão preenchidos para a próxima etapa de consulta ZARC."))
			} else {
				var zMissing []string
				if data.Culture == "" { zMissing = append(zMissing, "cultura") }
				if data.PlantingStart == "" { zMissing = append(zMissing, "plantio") }
				if data.Soil == "" { zMissing = append(zMissing, "solo") }
				if data.Cycle == "" { zMissing = append(zMissing, "ciclo") }
				out = append(out, technicalPrepCheck("technical_zarc_input", "Parâmetros para ZARC", "review",
					"Para automatizar o ZARC ainda falta: "+strings.Join(zMissing, ", ") + "."))
			}
		}
	}

	if p.OperationType == "investimento" || p.OperationType == "aquisicao" {
		total := projectTechnicalBudgetTotal(data)
		if len(data.Items) == 0 {
			out = append(out, technicalPrepCheck("technical_items", "Itens do investimento/aquisição", "pending",
				"Cadastre ao menos um item com descrição, quantidade e valor para estruturar o investimento."))
		} else {
			incomplete := 0
			for _, item := range data.Items {
				if item.Description == "" || item.Quantity <= 0 || item.UnitValue <= 0 {
					incomplete++
				}
			}
			if incomplete > 0 {
				out = append(out, technicalPrepCheck("technical_items", "Itens do investimento/aquisição", "review",
					fmt.Sprintf("%d item(ns) cadastrado(s), mas %d precisa(m) de descrição, quantidade ou valor.", len(data.Items), incomplete)))
			} else {
				detail := fmt.Sprintf("%d item(ns) técnico(s), total estimado R$ %.2f.", len(data.Items), total)
				status := "ready"
				if p.RequestedAmount > 0 && total > 0 {
					diff := total - p.RequestedAmount
					if diff < 0 { diff = -diff }
					if diff > 1 {
						status = "review"
						detail += fmt.Sprintf(" O total difere em R$ %.2f do valor solicitado; conferir composição.", diff)
					}
				}
				out = append(out, technicalPrepCheck("technical_items", "Itens do investimento/aquisição", status, detail))
			}
		}
	}

	if profile.Livestock {
		var missing []string
		if data.AnimalCount <= 0 { missing = append(missing, "quantidade de animais") }
		if data.AnimalCategory == "" { missing = append(missing, "categoria animal") }
		if p.OperationType == "aquisicao" && data.AverageWeightKg <= 0 { missing = append(missing, "peso médio") }
		if len(missing) == 0 {
			out = append(out, technicalPrepCheck("technical_livestock", "Dados pecuários", "ready",
				fmt.Sprintf("%d animal(is) • %s.", data.AnimalCount, data.AnimalCategory)))
		} else {
			out = append(out, technicalPrepCheck("technical_livestock", "Dados pecuários", "pending",
				"Falta informar: "+strings.Join(missing, ", ") + "."))
		}
	}

	if profile.Irrigation {
		var missing []string
		if data.IrrigatedAreaHa <= 0 { missing = append(missing, "área irrigada") }
		if data.IrrigationSystem == "" { missing = append(missing, "sistema de irrigação") }
		if data.WaterSource == "" { missing = append(missing, "fonte de água") }
		if len(missing) == 0 {
			detail := fmt.Sprintf("Área irrigada %.2f ha • sistema %s • fonte %s.", data.IrrigatedAreaHa, data.IrrigationSystem, data.WaterSource)
			if data.FlowM3H > 0 {
				detail += fmt.Sprintf(" Vazão informada %.2f m³/h.", data.FlowM3H)
			}
			out = append(out, technicalPrepCheck("technical_irrigation", "Dados de irrigação", "ready", detail))
		} else {
			out = append(out, technicalPrepCheck("technical_irrigation", "Dados de irrigação", "pending",
				"Falta informar: "+strings.Join(missing, ", ") + "."))
		}
	}

	return out
}

func technicalPrepCheck(key, label, status, detail string) ProjectPreparationCheck {
	return ProjectPreparationCheck{
		Key: key, Label: label, Status: status, Detail: detail,
		Source: "Dados técnicos do projeto", Group: "technical",
	}
}
