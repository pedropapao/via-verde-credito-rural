package main

import (
	"errors"
	"fmt"
	"strings"
)

type ConferenceItem struct {
	Level  string `json:"level"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type ConferenceSummary struct {
	Status           string           `json:"status"`
	Label            string           `json:"label"`
	OKCount          int              `json:"ok_count"`
	AttentionCount   int              `json:"attention_count"`
	UnavailableCount int              `json:"unavailable_count"`
	Items            []ConferenceItem `json:"items"`
	ManualActions    []string         `json:"manual_actions"`
}

func (a *App) GetPropertyConference(propertyID int64) (ConferenceSummary, error) {
	if propertyID <= 0 {
		return ConferenceSummary{}, errors.New("imóvel inválido")
	}
	car, err := a.GetLatestCAR(propertyID)
	if err != nil {
		return ConferenceSummary{
			Status: "incomplete",
			Label:  "Sem consulta do CAR",
			Items: []ConferenceItem{{
				Level: "info", Title: "CAR", Detail: "Este imóvel ainda não possui uma consulta salva para conferência.",
			}},
			UnavailableCount: 1,
			ManualActions: []string{"Realizar a primeira consulta do CAR deste imóvel."},
		}, nil
	}

	out := buildConferenceSummary(car)
	p, pErr := a.GetProperty(propertyID)
	if pErr == nil && strings.TrimSpace(p.KMLPath) != "" {
		if kml, kErr := a.LoadPropertyKML(propertyID); kErr == nil {
			cmp := a.CompareKMLWithCAR(kml, car)
			level := cmp.Level
			if level == "error" {
				level = "warning"
			}
			out.Items = append(out.Items, ConferenceItem{
				Level: level,
				Title: "KML externo × CAR",
				Detail: cmp.Summary,
			})
			if cmp.Level == "ok" {
				out.OKCount++
			} else {
				out.AttentionCount++
				out.ManualActions = appendUniqueString(out.ManualActions, "Conferir visualmente os limites do KML externo e do CAR.")
			}
		}
	} else {
		out.Items = append(out.Items, ConferenceItem{
			Level: "info",
			Title: "KML externo",
			Detail: "Não carregado. O perímetro público do SICAR continua disponível; um KML externo é opcional para confronto independente.",
		})
		out.UnavailableCount++
	}

	if a.db != nil {
		var areas int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM project_areas WHERE property_id=?`, propertyID).Scan(&areas)
		if areas > 0 {
			out.Items = append(out.Items, ConferenceItem{
				Level: "ok", Title: "Áreas do projeto", Detail: fmt.Sprintf("%d gleba(s)/área(s) cadastrada(s) para este imóvel.", areas),
			})
			out.OKCount++
		}
		var routes int
		_ = a.db.QueryRow(`SELECT (SELECT COUNT(*) FROM access_routes WHERE property_id=?) + (SELECT COUNT(*) FROM automatic_routes WHERE property_id=?)`, propertyID, propertyID).Scan(&routes)
		if routes > 0 {
			detail := "Roteiro de acesso disponível."
			if r, rErr := a.GetAccessRoute(propertyID); rErr == nil && r.Automatic && r.RouteDistanceKm > 0 {
				detail = fmt.Sprintf("Roteiro automático calculado a partir de %s: %.2f km / %.0f min até o acesso viário estimado.", r.ReferenceLabel, r.RouteDistanceKm, r.RouteDurationMin)
			}
			out.Items = append(out.Items, ConferenceItem{
				Level: "ok", Title: "Roteiro de acesso", Detail: detail,
			})
			out.OKCount++
		}
	}

	finalizeConference(&out)
	return out, nil
}

func buildConferenceSummary(car CARResult) ConferenceSummary {
	out := ConferenceSummary{Status: "ok", Label: "Conferência concluída"}

	seen := map[string]bool{}
	for _, c := range car.Checks {
		key := strings.ToLower(strings.TrimSpace(c.Title + "|" + c.Detail))
		if seen[key] {
			continue
		}
		seen[key] = true
		level := normalizeConferenceLevel(c.Level, c.Title, c.Detail)
		out.Items = append(out.Items, ConferenceItem{Level: level, Title: c.Title, Detail: c.Detail})
		countConferenceLevel(&out, level)
	}

	env := car.Environment
	if car.HasGeometry {
		appendSourceConference(&out, "IBAMA / Embargos", env.IBAMAChecked, env.IBAMAEmbargoCount,
			"Consulta concluída sem interseção de embargo na geometria analisada.",
			fmt.Sprintf("%d ocorrência(s) espacial(is) exigem conferência na fonte oficial.", env.IBAMAEmbargoCount))
		appendSourceConference(&out, "FUNAI / Terras Indígenas", env.FUNAIChecked, env.IndigenousCount,
			"Consulta concluída sem interseção identificada.",
			fmt.Sprintf("%d interseção(ões) espacial(is) exigem conferência na fonte oficial.", env.IndigenousCount))
		appendSourceConference(&out, "ICMBio / UCs federais", env.ICMBioChecked, env.FederalUCCount,
			"Consulta concluída sem interseção identificada.",
			fmt.Sprintf("%d interseção(ões) espacial(is) exigem conferência na fonte oficial.", env.FederalUCCount))
		if env.MCRChecked {
			if env.MCRListed {
				out.Items = append(out.Items, ConferenceItem{Level: "warning", Title: "MMA / MCR-PRODES", Detail: "CAR localizado na lista pública consultada; confirmar o registro e a documentação aplicável."})
				out.AttentionCount++
				out.ManualActions = appendUniqueString(out.ManualActions, "Conferir o registro MMA/MCR-PRODES na fonte oficial.")
			} else {
				out.Items = append(out.Items, ConferenceItem{Level: "ok", Title: "MMA / MCR-PRODES", Detail: "Consulta concluída; CAR não localizado na lista pública consultada."})
				out.OKCount++
			}
		} else {
			out.Items = append(out.Items, ConferenceItem{Level: "info", Title: "MMA / MCR-PRODES", Detail: "Fonte não pôde ser confirmada nesta consulta."})
			out.UnavailableCount++
		}
	}

	availableThemes := 0
	for _, m := range car.Themes.Themes {
		if m.Available {
			availableThemes++
		}
	}
	switch {
	case car.Themes.Complete:
		out.Items = append(out.Items, ConferenceItem{Level: "ok", Title: "Temas SICAR", Detail: "APP, Reserva Legal, vegetação, área consolidada, uso restrito e servidão foram processados."})
		out.OKCount++
	case availableThemes > 0:
		out.Items = append(out.Items, ConferenceItem{Level: "info", Title: "Temas SICAR", Detail: fmt.Sprintf("%d de 6 temas foram processados; os demais permanecem indisponíveis nesta consulta.", availableThemes)})
		out.UnavailableCount++
	default:
		out.Items = append(out.Items, ConferenceItem{Level: "info", Title: "Temas SICAR", Detail: "Os pacotes públicos detalhados do SICAR não ficaram disponíveis nesta consulta."})
		out.UnavailableCount++
	}

	if env.IBAMAEmbargoCount > 0 || env.IndigenousCount > 0 || env.FederalUCCount > 0 || env.MCRListed {
		out.ManualActions = appendUniqueString(out.ManualActions, "Confirmar ocorrências territoriais diretamente nas respectivas fontes oficiais.")
	}
	if out.UnavailableCount > 0 {
		out.ManualActions = appendUniqueString(out.ManualActions, "Repetir as fontes indisponíveis antes de concluir uma análise que dependa delas.")
	}
	finalizeConference(&out)
	return out
}

func appendSourceConference(out *ConferenceSummary, title string, checked bool, count int, okText, warningText string) {
	if !checked {
		out.Items = append(out.Items, ConferenceItem{Level: "info", Title: title, Detail: "Fonte não pôde ser confirmada nesta consulta."})
		out.UnavailableCount++
		return
	}
	if count > 0 {
		out.Items = append(out.Items, ConferenceItem{Level: "warning", Title: title, Detail: warningText})
		out.AttentionCount++
		return
	}
	out.Items = append(out.Items, ConferenceItem{Level: "ok", Title: title, Detail: okText})
	out.OKCount++
}

func normalizeConferenceLevel(level, title, detail string) string {
	l := strings.ToLower(strings.TrimSpace(level))
	txt := strings.ToLower(title + " " + detail)
	if strings.Contains(txt, "indispon") || strings.Contains(txt, "não respondeu") || strings.Contains(txt, "nao respondeu") {
		return "info"
	}
	switch l {
	case "ok":
		return "ok"
	case "warning", "error":
		return "warning"
	default:
		return "info"
	}
}

func countConferenceLevel(out *ConferenceSummary, level string) {
	switch level {
	case "ok":
		out.OKCount++
	case "warning":
		out.AttentionCount++
	default:
		out.UnavailableCount++
	}
}

func finalizeConference(out *ConferenceSummary) {
	switch {
	case out.AttentionCount > 0:
		out.Status = "attention"
		out.Label = "Conferência requer atenção"
	case out.UnavailableCount > 0:
		out.Status = "incomplete"
		out.Label = "Conferência parcial"
	default:
		out.Status = "ok"
		out.Label = "Conferência concluída"
	}
}

func appendUniqueString(items []string, value string) []string {
	for _, x := range items {
		if x == value {
			return items
		}
	}
	return append(items, value)
}
