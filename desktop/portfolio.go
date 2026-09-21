package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

type PortfolioAuditItem struct {
	PropertyID   int64    `json:"property_id"`
	ClientName   string   `json:"client_name"`
	PropertyName string   `json:"property_name"`
	CAR          string   `json:"car"`
	Status       string   `json:"status"`
	Result       string   `json:"result"`
	Changed      bool     `json:"changed"`
	Changes      []string `json:"changes"`
	Error        string   `json:"error"`
	CheckedAt    string   `json:"checked_at"`
}

type PortfolioAuditResult struct {
	Mode        string               `json:"mode"`
	Checked     int                  `json:"checked"`
	Unchanged   int                  `json:"unchanged"`
	Changed     int                  `json:"changed"`
	NeedsReview int                  `json:"needs_review"`
	Failed      int                  `json:"failed"`
	Items       []PortfolioAuditItem `json:"items"`
	StartedAt   string               `json:"started_at"`
	FinishedAt  string               `json:"finished_at"`
}

func (a *App) AuditCARPortfolio(full bool) (PortfolioAuditResult, error) {
	if a.db == nil {
		return PortfolioAuditResult{}, errors.New("banco local indisponível")
	}
	props, err := a.ListProperties(0)
	if err != nil {
		return PortfolioAuditResult{}, err
	}
	targets := make([]Property, 0, len(props))
	for _, p := range props {
		if strings.TrimSpace(p.CARNumber) != "" {
			targets = append(targets, p)
		}
	}
	out := PortfolioAuditResult{Mode: "rápida", StartedAt: time.Now().Format(time.RFC3339)}
	if full {
		out.Mode = "completa"
	}
	if len(targets) == 0 {
		out.FinishedAt = time.Now().Format(time.RFC3339)
		return out, nil
	}

	if full {
		for _, p := range targets {
			item := PortfolioAuditItem{PropertyID: p.ID, ClientName: p.ClientName, PropertyName: p.Name, CAR: p.CARNumber, CheckedAt: time.Now().Format(time.RFC3339)}
			beforeRaw := ""
			_ = a.db.QueryRow(`SELECT last_car_json FROM properties WHERE id=?`, p.ID).Scan(&beforeRaw)
			r, err := a.analyzeCAR(p.ID, p.CARNumber)
			if err != nil {
				item.Error = err.Error()
				item.Result = "Falha na consulta"
				out.Failed++
			} else {
				item.Status = r.Status
				item.Changes = compareCARCoreRaw(beforeRaw, r)
				item.Changed = len(item.Changes) > 0 && strings.TrimSpace(beforeRaw) != ""
				if item.Changed {
					item.Result = "Alteração detectada"
					out.Changed++
				} else {
					item.Result = "Sem alteração material"
					out.Unchanged++
				}
				if hasCARReviewFinding(r) {
					out.NeedsReview++
				}
			}
			out.Items = append(out.Items, item)
			out.Checked++
		}
		out.FinishedAt = time.Now().Format(time.RFC3339)
		return out, nil
	}

	type indexed struct {
		i    int
		item PortfolioAuditItem
		core CARResult
		raw  string
	}
	jobs := make(chan int)
	results := make(chan indexed, len(targets))
	workers := 3
	if len(targets) < workers {
		workers = len(targets)
	}
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				p := targets[i]
				item := PortfolioAuditItem{PropertyID: p.ID, ClientName: p.ClientName, PropertyName: p.Name, CAR: p.CARNumber, CheckedAt: time.Now().Format(time.RFC3339)}
				var previousRaw string
				_ = a.db.QueryRow(`SELECT last_car_json FROM properties WHERE id=?`, p.ID).Scan(&previousRaw)
				core, qErr := quickCARLookup(p.CARNumber)
				if qErr != nil {
					item.Error = qErr.Error()
					item.Result = "Falha na consulta"
					results <- indexed{i: i, item: item, raw: previousRaw}
					continue
				}
				item.Status = core.Status
				item.Changes = compareCARCoreRaw(previousRaw, core)
				item.Changed = strings.TrimSpace(previousRaw) != "" && len(item.Changes) > 0
				switch {
				case strings.TrimSpace(previousRaw) == "":
					item.Result = "Sem linha de base"
				case item.Changed:
					item.Result = "Alteração detectada"
				default:
					item.Result = "Sem alteração material"
				}
				results <- indexed{i: i, item: item, core: core, raw: previousRaw}
			}
		}()
	}
	go func() {
		for i := range targets {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	ordered := make([]indexed, len(targets))
	for x := range results {
		ordered[x.i] = x
	}
	for i, x := range ordered {
		item := x.item
		if item.Error != "" {
			out.Failed++
			out.Items = append(out.Items, item)
			out.Checked++
			continue
		}
		if strings.TrimSpace(x.raw) == "" || item.Changed {
			// Só faz a triagem completa quando é necessário criar a linha de base
			// ou quando a consulta rápida identificou mudança material.
			fullResult, err := a.analyzeCAR(targets[i].ID, targets[i].CARNumber)
			if err != nil {
				item.Error = err.Error()
				item.Result = "Mudança detectada, mas a análise completa falhou"
				out.Failed++
			} else {
				item.Status = fullResult.Status
				if strings.TrimSpace(x.raw) == "" {
					item.Result = "Linha de base criada"
					item.Changed = false
					item.Changes = nil
					out.Unchanged++
				} else {
					item.Changes = compareCARCoreRaw(x.raw, fullResult)
					item.Changed = len(item.Changes) > 0
					if item.Changed {
						item.Result = "Alteração confirmada"
						out.Changed++
					} else {
						item.Result = "Sem alteração material"
						out.Unchanged++
					}
				}
				if hasCARReviewFinding(fullResult) {
					out.NeedsReview++
				}
			}
		} else {
			out.Unchanged++
		}
		out.Items = append(out.Items, item)
		out.Checked++
	}
	out.FinishedAt = time.Now().Format(time.RFC3339)
	return out, nil
}

func quickCARLookup(number string) (CARResult, error) {
	car, uf, muni, err := normalizeCAR(number)
	if err != nil {
		return CARResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	f, err := lookupCARPublic(ctx, car, uf)
	if err != nil {
		return CARResult{}, err
	}
	if f == nil {
		return CARResult{CAR: car, UF: uf, MunicipalityCode: muni, CheckedAt: time.Now().Format(time.RFC3339)}, nil
	}
	out := CARResult{
		CAR: car, UF: uf, MunicipalityCode: muni, Found: true, CheckedAt: time.Now().Format(time.RFC3339),
		Municipality: carStringProp(f.Properties, "nom_munici", "nom_municipio", "municipio", "nm_muni", "nome_municipio"),
		PropertyName: carStringProp(f.Properties, "nom_imovel", "nome_imovel", "imovel"),
		Status: carStatusLabel(carStringProp(f.Properties, "status_imovel", "ind_status", "situacao", "status")),
		Condition: carStringProp(f.Properties, "des_condic", "condicao", "descricao_condicao"),
		PropertyType: carPropertyTypeLabel(carStringProp(f.Properties, "tipo_imovel", "ind_tipo_i", "ind_tipo", "tipo_imove", "des_tipo", "tipo")),
		AreaHa: carFloatProp(f.Properties, "num_area", "num_area_i", "area_imove", "area_ha", "area"),
		FiscalModules: carFloatProp(f.Properties, "m_fiscal", "num_modulo", "mod_fiscal", "modulos_fiscais"),
		DataCadastro: carStringProp(f.Properties, "dat_criacao", "dat_criaca", "data_cadastro", "data_criacao"),
		DataAtualizacao: carStringProp(f.Properties, "data_atualizacao", "dat_atuali", "data_ultima_atualizacao"),
		HasGeometry: carGeometryUsable(f.Geometry),
	}
	if out.HasGeometry {
		out.GeometryAreaHa = carGeometryAreaHa(f.Geometry)
		out.PerimeterM = carGeometryPerimeterM(f.Geometry)
		out.CenterLat, out.CenterLon = carGeometryCenter(f.Geometry)
		b, _ := json.Marshal(carGeoFeature{Type: "Feature", Properties: f.Properties, Geometry: f.Geometry})
		out.GeoJSON = string(b)
	}
	return out, nil
}

func compareCARCoreRaw(previousRaw string, current CARResult) []string {
	if strings.TrimSpace(previousRaw) == "" {
		return nil
	}
	var old CARResult
	if json.Unmarshal([]byte(previousRaw), &old) != nil {
		return []string{"A linha de base anterior não pôde ser lida"}
	}
	var changes []string
	if strings.TrimSpace(old.Status) != strings.TrimSpace(current.Status) {
		changes = append(changes, fmt.Sprintf("Situação: %s → %s", emptyDash(old.Status), emptyDash(current.Status)))
	}
	if strings.TrimSpace(old.Condition) != strings.TrimSpace(current.Condition) {
		changes = append(changes, fmt.Sprintf("Condição: %s → %s", emptyDash(old.Condition), emptyDash(current.Condition)))
	}
	if strings.TrimSpace(old.Municipality) != strings.TrimSpace(current.Municipality) {
		changes = append(changes, fmt.Sprintf("Município: %s → %s", emptyDash(old.Municipality), emptyDash(current.Municipality)))
	}
	if math.Abs(old.AreaHa-current.AreaHa) > 0.0001 {
		changes = append(changes, fmt.Sprintf("Área: %.4f ha → %.4f ha", old.AreaHa, current.AreaHa))
	}
	if geometryFingerprint(old.GeoJSON) != geometryFingerprint(current.GeoJSON) {
		changes = append(changes, "Geometria pública alterada")
	}
	if strings.TrimSpace(old.DataAtualizacao) != strings.TrimSpace(current.DataAtualizacao) && strings.TrimSpace(current.DataAtualizacao) != "" {
		changes = append(changes, fmt.Sprintf("Data de atualização SICAR: %s → %s", emptyDash(old.DataAtualizacao), emptyDash(current.DataAtualizacao)))
	}
	return changes
}

func hasCARReviewFinding(r CARResult) bool {
	return r.Environment.IBAMAEmbargoCount > 0 ||
		r.Environment.IndigenousCount > 0 ||
		r.Environment.FederalUCCount > 0 ||
		r.Environment.MCRListed ||
		!strings.EqualFold(strings.TrimSpace(r.Status), "Ativo")
}
