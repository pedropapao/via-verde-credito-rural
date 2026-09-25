package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
)

// UniversalSearchHit represents a safe, local match. CPF/CNPJ searches never
// attempt to bypass protected SICAR ownership data: they only match records
// already stored locally by the user.
type UniversalSearchHit struct {
	Kind         string  `json:"kind"`
	Title        string  `json:"title"`
	Subtitle     string  `json:"subtitle"`
	ClientID     int64   `json:"client_id"`
	PropertyID   int64   `json:"property_id"`
	ClientName   string  `json:"client_name"`
	PropertyName string  `json:"property_name"`
	CPFCNPJ      string  `json:"cpf_cnpj"`
	CAR          string  `json:"car"`
	Registry     string  `json:"registry"`
	Municipality string  `json:"municipality"`
	UF           string  `json:"uf"`
	AreaHa       float64 `json:"area_ha"`
}

type DocumentSourceOption struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
	URL       string `json:"url"`
	Automatic bool   `json:"automatic"`
}

type UniversalSearchResult struct {
	Query           string               `json:"query"`
	Mode            string               `json:"mode"`
	Normalized      string               `json:"normalized"`
	Valid           bool                 `json:"valid"`
	CanAnalyzeCAR   bool                 `json:"can_analyze_car"`
	Hits            []UniversalSearchHit `json:"hits"`
	Message         string               `json:"message"`
	PrivacyNotice   string               `json:"privacy_notice"`
	OfficialOptions []DocumentSourceOption `json:"official_options"`
}

type AutomationSourceStatus struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
	Count     int    `json:"count"`
	SourceURL string `json:"source_url"`
}

type CARAutomationResult struct {
	GeneratedAt       string                         `json:"generated_at"`
	Input             string                         `json:"input"`
	PropertyID        int64                          `json:"property_id"`
	MatchedExisting   bool                           `json:"matched_existing"`
	LocalDataEnriched bool                           `json:"local_data_enriched"`
	OverallStatus     string                         `json:"overall_status"`
	ExecutiveSummary  string                         `json:"executive_summary"`
	CAR               CARResult                      `json:"car"`
	XRay              PropertyXRay                   `json:"xray"`
	Environmental     EnvironmentalIntelligenceResult `json:"environmental"`
	Sources           []AutomationSourceStatus       `json:"sources"`
	Warnings          []string                       `json:"warnings"`
}

// SearchEverything searches local clients/properties using one box. A valid CAR
// also becomes directly analyzable even when it has not been saved locally.
func (a *App) SearchEverything(query string) (UniversalSearchResult, error) {
	q := strings.TrimSpace(query)
	out := UniversalSearchResult{Query: q, Valid: true}
	if q == "" {
		out.Mode = "empty"
		out.Message = "Digite CAR, CPF, CNPJ, cliente, imóvel, matrícula ou município."
		return out, nil
	}

	mode, normalized, valid := v2ClassifyUnifiedQuery(q)
	out.Mode, out.Normalized, out.Valid = mode, normalized, valid
	if mode == "car" && valid {
		out.CanAnalyzeCAR = true
		out.Message = "CAR válido: a análise automática pode ser iniciada diretamente."
	} else if (mode == "cpf" || mode == "cnpj") && !valid {
		out.Message = strings.ToUpper(mode) + " com dígitos verificadores inválidos."
	} else if mode == "cpf" || mode == "cnpj" {
		out.Message = "Busca por " + strings.ToUpper(mode) + ": primeiro o ViaVerdeCAR cruza sua base local e apresenta os caminhos oficiais disponíveis."
		out.PrivacyNotice = "O SICAR público não oferece pesquisa de imóvel por CPF/CNPJ. Vínculos de titularidade só são mostrados quando já cadastrados localmente ou quando uma fonte oficial consultada comprovar a relação."
		out.OfficialOptions = documentOfficialOptions(mode)
	} else {
		out.Message = "Busca local por cliente, imóvel, matrícula, município e CAR."
	}

	if a == nil || a.db == nil {
		if out.CanAnalyzeCAR {
			return out, nil
		}
		return out, errors.New("banco local indisponível")
	}

	clients, err := a.ListClients()
	if err != nil {
		return out, err
	}
	properties, err := a.ListProperties(0)
	if err != nil {
		return out, err
	}

	clientByID := make(map[int64]Client, len(clients))
	for _, c := range clients {
		clientByID[c.ID] = c
	}

	needle := v2NormalizedSearchText(q)
	docNeedle := v2DigitsOnly(q)
	seenProperty := map[int64]bool{}
	seenClient := map[int64]bool{}

	for _, p := range properties {
		c := clientByID[p.ClientID]
		searchable := v2NormalizedSearchText(strings.Join([]string{
			c.Name, c.CPFCNPJ, c.Phone, p.Name, p.Municipality, p.UF,
			p.Registry, p.CARNumber,
		}, " "))
		match := strings.Contains(searchable, needle)
		if docNeedle != "" && (mode == "cpf" || mode == "cnpj") {
			match = strings.Contains(v2DigitsOnly(c.CPFCNPJ), docNeedle)
		}
		if mode == "car" && valid {
			car, _, _, _ := normalizeCAR(q)
			match = strings.EqualFold(strings.TrimSpace(p.CARNumber), car)
		}
		if !match {
			continue
		}
		out.Hits = append(out.Hits, UniversalSearchHit{
			Kind: "property", Title: v2FirstNonEmpty(p.Name, "Imóvel sem nome"),
			Subtitle: strings.TrimSpace(strings.Join(v2NonEmpty([]string{c.Name, v2JoinPlace(p.Municipality, p.UF)}), " • ")),
			ClientID: p.ClientID, PropertyID: p.ID, ClientName: c.Name, PropertyName: p.Name,
			CPFCNPJ: c.CPFCNPJ, CAR: p.CARNumber, Registry: p.Registry,
			Municipality: p.Municipality, UF: p.UF, AreaHa: p.DeclaredAreaHa,
		})
		seenProperty[p.ID] = true
		seenClient[c.ID] = true
		if len(out.Hits) >= 30 {
			break
		}
	}

	if len(out.Hits) < 30 {
		for _, c := range clients {
			if seenClient[c.ID] {
				continue
			}
			searchable := v2NormalizedSearchText(strings.Join([]string{c.Name, c.CPFCNPJ, c.Phone}, " "))
			match := strings.Contains(searchable, needle)
			if docNeedle != "" && (mode == "cpf" || mode == "cnpj") {
				match = strings.Contains(v2DigitsOnly(c.CPFCNPJ), docNeedle)
			}
			if !match {
				continue
			}
			out.Hits = append(out.Hits, UniversalSearchHit{
				Kind: "client", Title: c.Name, Subtitle: v2FirstNonEmpty(c.CPFCNPJ, "Cliente local"),
				ClientID: c.ID, ClientName: c.Name, CPFCNPJ: c.CPFCNPJ,
			})
			if len(out.Hits) >= 30 {
				break
			}
		}
	}

	if mode == "car" && valid && len(out.Hits) == 0 {
		car, _, _, _ := normalizeCAR(q)
		out.Hits = append(out.Hits, UniversalSearchHit{
			Kind: "car_public", Title: "Consultar CAR público", Subtitle: car, CAR: car,
		})
	}
	return out, nil
}

// RunCARAutomation coordinates the existing reliable modules instead of
// duplicating them. One CAR drives cadastral, KML, environmental, fundiary and
// rural-credit analyses. Failures remain isolated and are returned explicitly.
func (a *App) RunCARAutomation(input string, propertyID int64, force bool) (CARAutomationResult, error) {
	started := time.Now()
	out := CARAutomationResult{Input: strings.TrimSpace(input), GeneratedAt: time.Now().Format(time.RFC3339)}
	car, _, _, err := normalizeCAR(input)
	if err != nil {
		return out, err
	}

	if propertyID <= 0 {
		if p, ok := a.findPropertyByCAR(car); ok {
			propertyID = p.ID
			out.MatchedExisting = true
		}
	}
	out.PropertyID = propertyID

	carResult, err := a.AnalyzePropertyCAR(propertyID, car)
	out.CAR = carResult
	if err != nil {
		out.OverallStatus = "error"
		out.ExecutiveSummary = "A consulta do SICAR não foi concluída; nenhuma conclusão de ausência de ocorrência deve ser inferida."
		out.Warnings = append(out.Warnings, err.Error())
		out.Sources = automationSources(out)
		return out, err
	}
	if !carResult.Found {
		out.OverallStatus = "not_found"
		out.ExecutiveSummary = "O código informado tem formato válido, mas não foi localizado na camada pública SICAR consultada."
		out.Sources = automationSources(out)
		return out, nil
	}
	if propertyID > 0 {
		enriched, enrichErr := a.enrichPropertyFromCAR(propertyID, carResult)
		out.LocalDataEnriched = enriched
		if enrichErr != nil {
			out.Warnings = append(out.Warnings, "Cadastro local: "+enrichErr.Error())
		}
	}

	if !carResult.HasGeometry {
		out.OverallStatus = "partial"
		out.ExecutiveSummary = "O CAR foi localizado, porém sem geometria pública utilizável; os cruzamentos espaciais automáticos ficaram limitados."
		out.Sources = automationSources(out)
		return out, nil
	}

	type xrayResp struct {
		v PropertyXRay
		e error
	}
	type envResp struct {
		v EnvironmentalIntelligenceResult
		e error
	}
	xrayCh := make(chan xrayResp, 1)
	envCh := make(chan envResp, 1)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				xrayCh <- xrayResp{e: fmt.Errorf("falha isolada no Raio X territorial: %v", r)}
			}
		}()
		v, e := a.BuildPropertyXRay(propertyID, force)
		xrayCh <- xrayResp{v: v, e: e}
	}()
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				envCh <- envResp{e: fmt.Errorf("falha isolada na inteligência ambiental: %v", r)}
			}
		}()
		v, e := a.GetEnvironmentalIntelligence(propertyID, force)
		envCh <- envResp{v: v, e: e}
	}()
	wg.Wait()

	xr, ev := <-xrayCh, <-envCh
	if xr.e != nil {
		out.Warnings = append(out.Warnings, "Raio X: "+xr.e.Error())
	} else {
		out.XRay = xr.v
		out.Warnings = append(out.Warnings, xr.v.Warnings...)
	}
	if ev.e != nil {
		out.Warnings = append(out.Warnings, "Ambiental: "+ev.e.Error())
	} else {
		out.Environmental = ev.v
		out.Warnings = append(out.Warnings, ev.v.Warnings...)
	}

	out.Sources = automationSources(out)
	out.OverallStatus = automationOverallStatus(out)
	out.ExecutiveSummary = automationExecutiveSummary(out, time.Since(started))
	out.GeneratedAt = time.Now().Format(time.RFC3339)
	out.Warnings = v2UniqueNonEmpty(out.Warnings)
	return out, nil
}

// SaveAnalyzedCARToClient turns an ad-hoc CAR consultation into a permanent
// property without repeating the public query when the current session matches.
// It never infers ownership: the user explicitly chooses the local client.
func (a *App) SaveAnalyzedCARToClient(clientID int64, car string) (Property, error) {
	if a == nil || a.db == nil {
		return Property{}, errors.New("banco local indisponível")
	}
	if clientID <= 0 {
		return Property{}, errors.New("selecione um cliente para salvar o imóvel")
	}
	normalized, _, _, err := normalizeCAR(car)
	if err != nil {
		return Property{}, err
	}
	if existing, ok := a.findPropertyByCAR(normalized); ok {
		return existing, nil
	}

	var clientExists int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM clients WHERE id=?`, clientID).Scan(&clientExists); err != nil || clientExists == 0 {
		return Property{}, errors.New("cliente não localizado na base local")
	}

	var result CARResult
	if cache, cacheErr := a.GetLastCARSession(); cacheErr == nil && strings.EqualFold(strings.TrimSpace(cache.Result.CAR), normalized) {
		result = cache.Result
	} else {
		result, err = a.AnalyzePropertyCAR(0, normalized)
		if err != nil {
			return Property{}, err
		}
	}
	if !result.Found {
		return Property{}, errors.New("o CAR não foi localizado na camada pública; nada foi salvo")
	}

	name := strings.TrimSpace(result.PropertyName)
	if name == "" && strings.TrimSpace(result.Municipality) != "" {
		name = "Imóvel - " + strings.TrimSpace(result.Municipality)
	}
	if name == "" {
		name = "Imóvel CAR"
	}
	area := result.AreaHa
	if area <= 0 {
		area = result.GeometryAreaHa
	}

	saved, err := a.SaveProperty(Property{
		ClientID:       clientID,
		Name:           name,
		Municipality:   strings.TrimSpace(result.Municipality),
		UF:             strings.ToUpper(strings.TrimSpace(result.UF)),
		CARNumber:      normalized,
		DeclaredAreaHa: area,
	})
	if err != nil {
		return Property{}, err
	}

	// Persist the already obtained geometry/KML and snapshot. If KML writing
	// fails, the property remains valid and the failure is recorded in checks.
	if result.HasGeometry && strings.TrimSpace(result.GeoJSON) != "" {
		var feature carGeoFeature
		if jsonErr := json.Unmarshal([]byte(result.GeoJSON), &feature); jsonErr == nil && carGeometryUsable(feature.Geometry) {
			if kmlPath, kmlErr := a.saveAutomaticCARKML(saved, result, feature.Geometry); kmlErr == nil {
				// AutoKMLPath é o perímetro público SICAR. KMLPath continua
				// reservado ao KML externo/do cliente para não comparar o CAR
				// contra ele mesmo.
				result.AutoKMLPath = kmlPath
			} else {
				result.Checks = append(result.Checks, QualityCheck{Level: "warning", Title: "KML automático", Detail: "O imóvel foi salvo, mas o KML não pôde ser gravado: " + kmlErr.Error()})
			}
		}
	}
	if err := a.persistCARAnalysis(saved.ID, normalized, &result); err != nil {
		return Property{}, fmt.Errorf("imóvel salvo, mas o histórico do CAR não pôde ser registrado: %w", err)
	}
	a.saveLastCARSession(result)
	return a.GetProperty(saved.ID)
}

func (a *App) findPropertyByCAR(car string) (Property, bool) {
	if a == nil || a.db == nil || strings.TrimSpace(car) == "" {
		return Property{}, false
	}
	var p Property
	err := a.db.QueryRow(`
		SELECT p.id,p.client_id,c.name,p.name,p.municipality,p.uf,p.registry,
		       p.car_number,p.declared_area_ha,p.kml_path,p.created_at,p.updated_at
		FROM properties p JOIN clients c ON c.id=p.client_id
		WHERE UPPER(TRIM(p.car_number))=UPPER(TRIM(?))
		ORDER BY p.updated_at DESC LIMIT 1
	`, car).Scan(&p.ID, &p.ClientID, &p.ClientName, &p.Name, &p.Municipality, &p.UF,
		&p.Registry, &p.CARNumber, &p.DeclaredAreaHa, &p.KMLPath, &p.CreatedAt, &p.UpdatedAt)
	return p, err == nil
}

func (a *App) enrichPropertyFromCAR(propertyID int64, car CARResult) (bool, error) {
	if a == nil || a.db == nil || propertyID <= 0 {
		return false, nil
	}
	p, err := a.GetProperty(propertyID)
	if err != nil {
		return false, err
	}
	name := p.Name
	municipality := p.Municipality
	uf := p.UF
	area := p.DeclaredAreaHa
	changed := false
	if strings.TrimSpace(name) == "" && strings.TrimSpace(car.PropertyName) != "" {
		name = strings.TrimSpace(car.PropertyName)
		changed = true
	}
	if strings.TrimSpace(municipality) == "" && strings.TrimSpace(car.Municipality) != "" {
		municipality = strings.TrimSpace(car.Municipality)
		changed = true
	}
	if strings.TrimSpace(uf) == "" && strings.TrimSpace(car.UF) != "" {
		uf = strings.ToUpper(strings.TrimSpace(car.UF))
		changed = true
	}
	if area <= 0 && car.AreaHa > 0 {
		area = car.AreaHa
		changed = true
	}
	if !changed {
		return false, nil
	}
	_, err = a.db.Exec(`
		UPDATE properties SET name=?,municipality=?,uf=?,declared_area_ha=?,updated_at=? WHERE id=?
	`, name, municipality, uf, area, time.Now().Format(time.RFC3339), propertyID)
	return err == nil, err
}

func automationSources(out CARAutomationResult) []AutomationSourceStatus {
	sources := []AutomationSourceStatus{{
		Key: "sicar", Label: "SICAR", SourceURL: out.CAR.OfficialURL,
	}}
	switch strings.ToLower(strings.TrimSpace(out.CAR.LookupStatus)) {
	case "cached":
		sources[0].Status = "cached"
		sources[0].Detail = v2FirstNonEmpty(out.CAR.LookupDetail, "SICAR indisponível; última geometria pública salva foi reaproveitada.")
	case "partial":
		sources[0].Status = "partial"
		sources[0].Detail = v2FirstNonEmpty(out.CAR.LookupDetail, "CAR e geometria localizados, com ficha cadastral parcial.")
	case "unavailable":
		sources[0].Status = "unavailable"
		sources[0].Detail = v2FirstNonEmpty(out.CAR.LookupDetail, "Base pública SICAR indisponível.")
	case "not_found":
		sources[0].Status = "not_found"
		sources[0].Detail = v2FirstNonEmpty(out.CAR.LookupDetail, "CAR não localizado na camada pública consultada.")
	default:
		if out.CAR.Found {
			sources[0].Status = "ok"
			sources[0].Detail = "CAR localizado na camada pública."
		} else {
			sources[0].Status = "not_found"
			sources[0].Detail = "CAR não localizado na camada pública consultada."
		}
	}
	if !out.CAR.Found {
		return sources
	}

	kmlStatus := AutomationSourceStatus{Key: "kml", Label: "KML automático"}
	if strings.TrimSpace(out.CAR.AutoKMLPath) != "" {
		kmlStatus.Status, kmlStatus.Detail = "ok", "Perímetro SICAR salvo automaticamente no imóvel."
	} else if out.PropertyID <= 0 {
		kmlStatus.Status, kmlStatus.Detail = "not_saved", "Consulta avulsa: geometria pronta, mas sem imóvel local para armazenamento automático."
	} else {
		kmlStatus.Status, kmlStatus.Detail = "unavailable", "Geometria encontrada, mas o KML automático não foi confirmado."
	}
	sources = append(sources, kmlStatus)

	env := out.Environmental.Environment
	if !env.IBAMAChecked && !env.FUNAIChecked && !env.ICMBioChecked && !env.MCRChecked {
		env = out.CAR.Environment
	}
	sources = append(sources,
		envSource("ibama", "IBAMA • Embargos", env.IBAMAChecked, env.IBAMAEmbargoCount, env.IBAMASourceURL),
		envSource("funai", "FUNAI • Terras Indígenas", env.FUNAIChecked, env.IndigenousCount, env.FUNAISourceURL),
		envSource("icmbio", "ICMBio • UCs Federais", env.ICMBioChecked, env.FederalUCCount, env.ICMBioSourceURL),
	)
	mcr := AutomationSourceStatus{Key: "mcr", Label: "MMA • MCR/PRODES", SourceURL: env.MCRSourceURL}
	if env.MCRChecked {
		if env.MCRListed {
			mcr.Status, mcr.Count, mcr.Detail = "hit", 1, "CAR localizado na lista pública consultada; requer conferência técnica."
		} else {
			mcr.Status, mcr.Detail = "ok", "CAR não localizado na lista pública consultada."
		}
	} else {
		mcr.Status, mcr.Detail = "unavailable", "Base não confirmada nesta execução."
	}
	sources = append(sources, mcr)

	mb := out.Environmental.MapBiomas
	if !mb.Connected && !mb.Available && strings.TrimSpace(mb.Message) == "" {
		mb = out.XRay.MapBiomas
	}
	mbs := AutomationSourceStatus{Key: "mapbiomas", Label: "MapBiomas Alerta", Count: mb.TotalAlerts}
	if !mb.Connected {
		mbs.Status, mbs.Detail = "not_configured", v2FirstNonEmpty(mb.Message, "Conta MapBiomas Alerta não conectada.")
	} else if !mb.Available {
		mbs.Status, mbs.Detail = "unavailable", v2FirstNonEmpty(mb.Message, "Base indisponível nesta execução.")
	} else if mb.TotalAlerts > 0 {
		mbs.Status, mbs.Detail = "hit", fmt.Sprintf("%d alerta(s) retornado(s).", mb.TotalAlerts)
	} else {
		mbs.Status, mbs.Detail = "ok", "Nenhum alerta retornado na consulta atual."
	}
	sources = append(sources, mbs)

	fire := out.Environmental.Profile.Fire
	fs := AutomationSourceStatus{Key: "inpe_fire", Label: "INPE • Focos de calor", Count: fire.FeatureCount, SourceURL: fire.SourceURL}
	if !fire.Checked || !fire.Available {
		fs.Status, fs.Detail = "unavailable", v2FirstNonEmpty(fire.Warning, "Base não confirmada nesta execução.")
	} else if fire.FeatureCount > 0 {
		fs.Status, fs.Detail = "hit", fmt.Sprintf("%d foco(s) encontrado(s) na janela consultada.", fire.FeatureCount)
	} else {
		fs.Status, fs.Detail = "ok", "Nenhum foco retornado na janela consultada."
	}
	sources = append(sources, fs)

	sigef := out.XRay.SIGEF
	ss := AutomationSourceStatus{Key: "sigef", Label: "SIGEF / INCRA", Count: sigef.ParcelCount, SourceURL: sigef.SourceURL}
	if !sigef.Available {
		ss.Status, ss.Detail = "unavailable", v2FirstNonEmpty(sigef.Message, "Base não confirmada nesta execução.")
	} else if sigef.ParcelCount > 0 {
		ss.Status, ss.Detail = "hit", fmt.Sprintf("%d parcela(s) pública(s) com relação espacial analisada.", sigef.ParcelCount)
	} else {
		ss.Status, ss.Detail = "ok", "Nenhuma parcela retornada na consulta atual."
	}
	sources = append(sources, ss)

	sicor := out.XRay.SICOR
	cs := AutomationSourceStatus{Key: "sicor", Label: "SICOR • Crédito Rural", Count: sicor.OperationCount, SourceURL: sicor.SourceURL}
	if strings.TrimSpace(sicor.GeneratedAt) == "" {
		cs.Status, cs.Detail = "unavailable", "Consulta ficou parcial; confira os avisos da base."
	} else if sicor.OperationCount > 0 {
		cs.Status, cs.Detail = "hit", fmt.Sprintf("%d operação(ões) pública(s) associada(s) à análise territorial.", sicor.OperationCount)
	} else {
		cs.Status, cs.Detail = "ok", "Nenhuma operação retornada no recorte analisado."
	}
	sources = append(sources, cs)
	return sources
}

func envSource(key, label string, checked bool, count int, sourceURL string) AutomationSourceStatus {
	s := AutomationSourceStatus{Key: key, Label: label, Count: count, SourceURL: sourceURL}
	if !checked {
		s.Status, s.Detail = "unavailable", "Base não confirmada nesta execução."
	} else if count > 0 {
		s.Status, s.Detail = "hit", fmt.Sprintf("%d ocorrência(s) espacial(is) encontrada(s).", count)
	} else {
		s.Status, s.Detail = "ok", "Nenhuma ocorrência espacial retornada."
	}
	return s
}

func automationOverallStatus(out CARAutomationResult) string {
	if !out.CAR.Found {
		return "not_found"
	}
	partial := false
	for _, s := range out.Sources {
		if s.Status == "unavailable" || s.Status == "not_configured" || s.Status == "partial" || s.Status == "cached" {
			partial = true
		}
	}
	if partial {
		return "partial"
	}
	return "complete"
}

func automationExecutiveSummary(out CARAutomationResult, elapsed time.Duration) string {
	car := out.CAR
	parts := []string{}
	if car.Found {
		parts = append(parts, fmt.Sprintf("CAR %s localizado", car.CAR))
	}
	if car.Municipality != "" {
		parts = append(parts, v2JoinPlace(car.Municipality, car.UF))
	}
	if car.AreaHa > 0 {
		parts = append(parts, fmt.Sprintf("%.2f ha", car.AreaHa))
	}
	hits := 0
	unavailable := 0
	for _, s := range out.Sources {
		if s.Status == "hit" {
			hits += s.Count
			if s.Count == 0 {
				hits++
			}
		}
		if s.Status == "unavailable" || s.Status == "not_configured" {
			unavailable++
		}
	}
	summary := strings.Join(parts, " • ")
	if summary != "" {
		summary += ". "
	}
	if strings.EqualFold(car.LookupStatus, "cached") {
		summary += "SICAR não confirmou a ficha nesta tentativa; foi usada a última geometria pública salva para manter os cruzamentos. "
	} else if strings.EqualFold(car.LookupStatus, "partial") {
		summary += "SICAR confirmou o CAR/geometria, mas a ficha veio parcial. "
	}
	summary += fmt.Sprintf("A análise automática consolidou %d fonte(s); %d item(ns) de ocorrência/registro foram sinalizados para conferência", len(out.Sources), hits)
	if unavailable > 0 {
		summary += fmt.Sprintf(" e %d fonte(s) ficaram indisponíveis ou não configuradas", unavailable)
	}
	summary += fmt.Sprintf(". Tempo aproximado: %.0f s. Resultados são triagem técnica e devem ser conferidos na fonte oficial quando houver ocorrência.", elapsed.Seconds())
	return summary
}

func documentOfficialOptions(mode string) []DocumentSourceOption {
	local := DocumentSourceOption{
		Key: "local", Label: "ViaVerdeCAR • vínculos locais", Status: "ready", Automatic: true,
		Detail: "Pesquisa clientes, imóveis e CARs já vinculados no banco local.",
	}
	sncr := DocumentSourceOption{
		Key: "sncr", Label: "INCRA • Consulta Pública SNCR", Status: "manual_official",
		Detail: "A consulta pública do SNCR disponibiliza nome do titular, imóvel, município, área e condição. Não fornece uma API pública por CPF; use como conferência de candidatos por nome/localidade.",
		URL: "https://sncr.serpro.gov.br/sncr-web/consultaPublica.jsf",
	}
	if mode == "cpf" {
		return []DocumentSourceOption{
			local,
			{Key: "cpf_serpro", Label: "Receita Federal / SERPRO • Consulta CPF v3", Status: "credentials_required",
				Detail: "Fonte oficial automatizável mediante contratação/credenciais. A versão v3 exige CPF e data de nascimento; o ViaVerdeCAR não realiza consulta sem autorização configurada.",
				URL: "https://www.gov.br/pt-br/servicos/obter-solucao-digital-de-consulta-de-dados-de-cadastro-de-pessoa-fisica-cpf"},
			sncr,
		}
	}
	return []DocumentSourceOption{
		local,
		{Key: "cnpj_conecta", Label: "Receita Federal • Consulta CNPJ", Status: "credentials_required",
			Detail: "A API oficial de CNPJ pode retornar nome empresarial, situação e outros dados, mas exige adesão/credenciais do serviço oficial.",
			URL: "https://www.gov.br/conecta/catalogo/apis/consulta-cnpj"},
		{Key: "cnpj_open", Label: "Receita Federal • Dados Abertos CNPJ", Status: "bulk_official",
			Detail: "Base oficial em dados abertos para processamento em lote. Não é usada silenciosamente como consulta online por exigir sincronização local volumosa.",
			URL: "https://www.gov.br/receitafederal/pt-br/acesso-a-informacao/dados-abertos/cadastros"},
		sncr,
	}
}

func v2ClassifyUnifiedQuery(v string) (mode, normalized string, valid bool) {
	if car, _, _, err := normalizeCAR(v); err == nil {
		return "car", car, true
	}
	digits := v2DigitsOnly(v)
	switch len(digits) {
	case 11:
		return "cpf", digits, v2ValidCPF(digits)
	case 14:
		return "cnpj", digits, v2ValidCNPJ(digits)
	default:
		return "text", v2NormalizedSearchText(v), true
	}
}

func v2DigitsOnly(v string) string {
	var b strings.Builder
	for _, r := range v {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func v2ValidCPF(v string) bool {
	v = v2DigitsOnly(v)
	if len(v) != 11 || v2AllSame(v) {
		return false
	}
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(v[i]-'0') * (10 - i)
	}
	d := (sum * 10) % 11
	if d == 10 {
		d = 0
	}
	if d != int(v[9]-'0') {
		return false
	}
	sum = 0
	for i := 0; i < 10; i++ {
		sum += int(v[i]-'0') * (11 - i)
	}
	d = (sum * 10) % 11
	if d == 10 {
		d = 0
	}
	return d == int(v[10]-'0')
}

func v2ValidCNPJ(v string) bool {
	v = v2DigitsOnly(v)
	if len(v) != 14 || v2AllSame(v) {
		return false
	}
	calc := func(base string, weights []int) int {
		sum := 0
		for i, w := range weights {
			sum += int(base[i]-'0') * w
		}
		r := sum % 11
		if r < 2 {
			return 0
		}
		return 11 - r
	}
	d1 := calc(v[:12], []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	if d1 != int(v[12]-'0') {
		return false
	}
	d2 := calc(v[:13], []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2})
	return d2 == int(v[13]-'0')
}

func v2AllSame(v string) bool {
	if len(v) < 2 {
		return true
	}
	for i := 1; i < len(v); i++ {
		if v[i] != v[0] {
			return false
		}
	}
	return true
}

func v2NormalizedSearchText(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	repl := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "õ", "o", "ô", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c",
	)
	return strings.Join(strings.Fields(repl.Replace(v)), " ")
}

func v2FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func v2NonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

func v2JoinPlace(city, uf string) string {
	if strings.TrimSpace(city) == "" {
		return strings.TrimSpace(uf)
	}
	if strings.TrimSpace(uf) == "" {
		return strings.TrimSpace(city)
	}
	return strings.TrimSpace(city) + " / " + strings.ToUpper(strings.TrimSpace(uf))
}

func v2UniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
