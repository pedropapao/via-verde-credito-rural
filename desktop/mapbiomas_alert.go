package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const mapBiomasAlertGraphQL = "https://plataforma.alerta.mapbiomas.org/api/v2/graphql"

type MapBiomasAlertStatus struct {
	Connected bool   `json:"connected"`
	Email     string `json:"email"`
	Message   string `json:"message"`
}

type MapBiomasCARAlert struct {
	AlertCode   string   `json:"alert_code"`
	AreaHa      float64  `json:"area_ha"`
	DetectedAt  string   `json:"detected_at"`
	PublishedAt string   `json:"published_at"`
	Sources     []string `json:"sources"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	ReportURL   string   `json:"report_url"`
}

type MapBiomasCARSummary struct {
	Connected    bool                `json:"connected"`
	Found        bool                `json:"found"`
	PropertyCode string              `json:"property_code"`
	AreaHa       float64             `json:"area_ha"`
	State        string              `json:"state"`
	StateAcronym string              `json:"state_acronym"`
	CARUpdatedAt string              `json:"car_updated_at"`
	Alerts       []MapBiomasCARAlert `json:"alerts"`
	TotalAlerts  int                 `json:"total_alerts"`
	TotalAreaHa  float64             `json:"total_area_ha"`
	Message      string              `json:"message"`
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

func (a *App) getSetting(key string) string {
	if a == nil || a.db == nil {
		return ""
	}
	var value string
	_ = a.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&value)
	return value
}

func (a *App) setSetting(key, value string) error {
	if a == nil || a.db == nil {
		return errors.New("banco local indisponível")
	}
	_, err := a.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (a *App) deleteSetting(key string) error {
	if a == nil || a.db == nil {
		return errors.New("banco local indisponível")
	}
	_, err := a.db.Exec(`DELETE FROM settings WHERE key=?`, key)
	return err
}

func (a *App) GetMapBiomasAlertStatus() MapBiomasAlertStatus {
	email := strings.TrimSpace(a.getSetting("mapbiomas_alert_email"))
	token := strings.TrimSpace(a.getSetting("mapbiomas_alert_token"))
	if token == "" {
		return MapBiomasAlertStatus{Connected: false, Email: email, Message: "Conta MapBiomas Alerta não conectada."}
	}
	return MapBiomasAlertStatus{Connected: true, Email: email, Message: "MapBiomas Alerta conectado."}
}

func (a *App) MapBiomasAlertLogin(email, password string) (MapBiomasAlertStatus, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return MapBiomasAlertStatus{}, errors.New("informe e-mail e senha da sua conta MapBiomas Alerta")
	}
	query := `mutation signIn($email: String!, $password: String!) {
		signIn(email: $email, password: $password) { token }
	}`
	var resp struct {
		Data struct {
			SignIn struct {
				Token string `json:"token"`
			} `json:"signIn"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}
	if err := mapBiomasGraphQL("", graphQLRequest{
		Query: query,
		Variables: map[string]any{"email": email, "password": password},
	}, &resp); err != nil {
		return MapBiomasAlertStatus{}, err
	}
	if len(resp.Errors) > 0 {
		return MapBiomasAlertStatus{}, errors.New(resp.Errors[0].Message)
	}
	token := strings.TrimSpace(resp.Data.SignIn.Token)
	if token == "" {
		return MapBiomasAlertStatus{}, errors.New("MapBiomas Alerta não retornou token de acesso")
	}
	if err := a.setSetting("mapbiomas_alert_token", token); err != nil {
		return MapBiomasAlertStatus{}, err
	}
	if err := a.setSetting("mapbiomas_alert_email", email); err != nil {
		return MapBiomasAlertStatus{}, err
	}
	// A senha é usada apenas nesta chamada e nunca é gravada no banco local.
	return MapBiomasAlertStatus{Connected: true, Email: email, Message: "Conta MapBiomas Alerta conectada."}, nil
}

func (a *App) DisconnectMapBiomasAlert() error {
	_ = a.deleteSetting("mapbiomas_alert_token")
	_ = a.deleteSetting("mapbiomas_alert_email")
	return nil
}

type mapBiomasRuralPropertyResponse struct {
	Data struct {
		RuralProperty *struct {
			PropertyCode string  `json:"propertyCode"`
			AreaHa       float64 `json:"areaHa"`
			State        string  `json:"state"`
			StateAcronym string  `json:"stateAcronym"`
			CARUpdatedAt string  `json:"carUpdatedAt"`
			Alerts       []struct {
				AlertCode   any      `json:"alertCode"`
				AreaHa      float64  `json:"areaHa"`
				DetectedAt  string   `json:"detectedAt"`
				PublishedAt string   `json:"publishedAt"`
				Sources     []string `json:"sources"`
				Coordinates struct {
					Latitude  float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"coordinates"`
				Coordenates struct {
					Latitude  float64 `json:"latitude"`
					Longitude float64 `json:"longitude"`
				} `json:"coordenates"`
			} `json:"alerts"`
		} `json:"ruralProperty"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

func (a *App) QueryMapBiomasCAR(car string) (MapBiomasCARSummary, error) {
	car = strings.TrimSpace(car)
	if car == "" {
		return MapBiomasCARSummary{}, errors.New("CAR não informado")
	}
	token := strings.TrimSpace(a.getSetting("mapbiomas_alert_token"))
	if token == "" {
		return MapBiomasCARSummary{
			Connected: false,
			Message:   "Conecte uma conta MapBiomas Alerta em Configurações para consultar alertas automaticamente.",
		}, nil
	}

	resp, err := queryMapBiomasRuralProperty(token, car, "coordinates")
	if err != nil {
		return MapBiomasCARSummary{}, err
	}
	if hasGraphQLErrorContaining(resp.Errors, "coordinates", "AlertData") {
		// A documentação pública da V2 descreve ruralProperty.alerts como
		// RuralPropertyAlert (coordinates), mas algumas respostas do servidor
		// expõem AlertData, cujo campo histórico possui a grafia "coordenates".
		// O fallback mantém compatibilidade com os dois esquemas sem afetar a
		// consulta principal nem exigir intervenção do usuário.
		resp, err = queryMapBiomasRuralProperty(token, car, "coordenates")
		if err != nil {
			return MapBiomasCARSummary{}, err
		}
	}
	if len(resp.Errors) > 0 {
		return MapBiomasCARSummary{}, errors.New(joinGraphQLErrors(resp.Errors))
	}

	out := MapBiomasCARSummary{Connected: true}
	if resp.Data.RuralProperty == nil {
		out.Message = "CAR não localizado entre os imóveis cruzados com alertas na base consultada."
		return out, nil
	}
	p := resp.Data.RuralProperty
	out.Found = true
	out.PropertyCode = p.PropertyCode
	out.AreaHa = p.AreaHa
	out.State = p.State
	out.StateAcronym = p.StateAcronym
	out.CARUpdatedAt = p.CARUpdatedAt
	for _, x := range p.Alerts {
		code := graphQLScalarString(x.AlertCode)
		lat, lon := x.Coordinates.Latitude, x.Coordinates.Longitude
		if lat == 0 && lon == 0 {
			lat, lon = x.Coordenates.Latitude, x.Coordenates.Longitude
		}
		item := MapBiomasCARAlert{
			AlertCode: code, AreaHa: x.AreaHa,
			DetectedAt: x.DetectedAt, PublishedAt: x.PublishedAt,
			Sources: x.Sources, Latitude: lat, Longitude: lon,
		}
		if code != "" {
			item.ReportURL = "https://plataforma.alerta.mapbiomas.org/alerta/" + code
		}
		out.Alerts = append(out.Alerts, item)
		out.TotalAreaHa += x.AreaHa
	}
	out.TotalAlerts = len(out.Alerts)
	if out.TotalAlerts == 0 {
		out.Message = "Imóvel localizado na base MapBiomas Alerta sem alertas vinculados retornados pela consulta."
	} else {
		out.Message = fmt.Sprintf("%d alerta(s) vinculado(s) ao imóvel na base MapBiomas Alerta.", out.TotalAlerts)
	}
	return out, nil
}

func queryMapBiomasRuralProperty(token, car, coordinateField string) (mapBiomasRuralPropertyResponse, error) {
	if coordinateField != "coordinates" && coordinateField != "coordenates" {
		return mapBiomasRuralPropertyResponse{}, errors.New("campo de coordenadas MapBiomas inválido")
	}
	query := fmt.Sprintf(`query ruralProperty($carCode: String!) {
		ruralProperty(carCode: $carCode) {
			propertyCode
			areaHa
			state
			stateAcronym
			carUpdatedAt
			alerts {
				alertCode
				areaHa
				detectedAt
				publishedAt
				sources
				%s { latitude longitude }
			}
		}
	}`, coordinateField)
	var resp mapBiomasRuralPropertyResponse
	err := mapBiomasGraphQL(token, graphQLRequest{
		Query: query, Variables: map[string]any{"carCode": car},
	}, &resp)
	return resp, err
}

func hasGraphQLErrorContaining(errs []graphQLError, terms ...string) bool {
	if len(errs) == 0 {
		return false
	}
	for _, e := range errs {
		msg := strings.ToLower(e.Message)
		match := true
		for _, term := range terms {
			if !strings.Contains(msg, strings.ToLower(term)) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func joinGraphQLErrors(errs []graphQLError) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		if msg := strings.TrimSpace(e.Message); msg != "" {
			parts = append(parts, msg)
		}
	}
	if len(parts) == 0 {
		return "MapBiomas Alerta retornou erro sem detalhes"
	}
	return strings.Join(parts, " • ")
}

func graphQLScalarString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%.0f", x)
		}
		return fmt.Sprintf("%v", x)
	case json.Number:
		return x.String()
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func mapBiomasGraphQL(token string, reqBody graphQLRequest, dst any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, mapBiomasAlertGraphQL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("sessão MapBiomas Alerta inválida ou expirada; conecte novamente em Configurações")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("MapBiomas Alerta respondeu HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
