package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const zarcCKANPackageURL = "https://dados.agricultura.gov.br/api/3/action/package_show?id=tabua-de-risco-zoneamento-agricola-de-risco-climatico"

type ZARCPeriod struct {
	Decendio  int    `json:"decendio"`
	Raw       string `json:"raw"`
	RiskPct   int    `json:"risk_pct"`
	Indicated bool   `json:"indicated"`
}

type ZARCCheck struct {
	Checked       bool         `json:"checked"`
	Available     bool         `json:"available"`
	Matched       bool         `json:"matched"`
	Status        string       `json:"status"`
	Safra         string       `json:"safra"`
	Culture       string       `json:"culture"`
	Cycle         string       `json:"cycle"`
	Soil          string       `json:"soil"`
	Management    string       `json:"management"`
	Municipality  string       `json:"municipality"`
	UF            string       `json:"uf"`
	Portaria      string       `json:"portaria"`
	PlantingStart string       `json:"planting_start"`
	PlantingEnd   string       `json:"planting_end"`
	Candidates    int          `json:"candidates"`
	Periods       []ZARCPeriod `json:"periods"`
	Message       string       `json:"message"`
	SourceURL     string       `json:"source_url"`
	UpdatedAt     string       `json:"updated_at"`
}

type zarcResource struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Format       string `json:"format"`
	LastModified string `json:"last_modified"`
}

func (a *App) enrichCreditZARC(ctx context.Context, sourceDir string, car CARResult, byKey map[string]*CreditOperationIntelligence, out *CreditIntelligenceResult) {
	type resourceCache struct {
		resource zarcResource
		path     string
		err      error
	}
	cache := map[string]resourceCache{}
	for _, op := range byKey {
		if strings.TrimSpace(op.PlantingStart) == "" || strings.TrimSpace(op.Product) == "" {
			op.ZARC = ZARCCheck{
				Checked: false,
				Status: "Dados insuficientes",
				Message: "A operação não possui data de plantio e produto suficientes para cruzamento automático com a Tábua de Risco do ZARC.",
				SourceURL: creditIntelligenceZARCURL,
			}
			continue
		}
		start, err := parseCreditDate(op.PlantingStart)
		if err != nil {
			op.ZARC = ZARCCheck{Status:"Data de plantio inválida", Message:"Não foi possível interpretar a data de início de plantio publicada na operação.", SourceURL:creditIntelligenceZARCURL}
			continue
		}
		safraIni, safraFin := zarcCropSeason(start)
		safraKey := fmt.Sprintf("%d/%d", safraIni, safraFin)
		rc, ok := cache[safraKey]
		if !ok {
			res, err := resolveZARCResource(ctx, safraIni, safraFin)
			if err != nil {
				rc = resourceCache{err: err}
			} else {
				name := fmt.Sprintf("zarc_%d_%d.csv", safraIni, safraFin)
				path := filepath.Join(sourceDir, name)
				if st, statErr := os.Stat(path); statErr != nil || st.Size() == 0 || time.Since(st.ModTime()) > 7*24*time.Hour {
					tmp := path + ".part"
					_ = os.Remove(tmp)
					if dlErr := downloadLargeFile(ctx, res.URL, tmp); dlErr != nil {
						_ = os.Remove(tmp)
						rc = resourceCache{resource:res, err:dlErr}
					} else {
						_ = os.Remove(path)
						if rnErr := os.Rename(tmp, path); rnErr != nil {
							_ = os.Remove(tmp)
							rc = resourceCache{resource:res, err:rnErr}
						} else {
							rc = resourceCache{resource:res, path:path}
						}
					}
				} else {
					rc = resourceCache{resource:res, path:path}
				}
			}
			cache[safraKey] = rc
		}
		if rc.err != nil {
			op.ZARC = ZARCCheck{
				Checked:true, Available:false, Status:"Fonte indisponível", Safra:safraKey,
				Message:"A Tábua de Risco oficial não pôde ser sincronizada nesta consulta: "+rc.err.Error(),
				SourceURL:creditIntelligenceZARCURL,
			}
			out.Warnings = append(out.Warnings, "ZARC "+safraKey+": "+rc.err.Error())
			continue
		}
		check, err := scanZARCForOperation(rc.path, rc.resource, car, *op, safraIni, safraFin)
		if err != nil {
			check.Checked = true
			check.Available = true
			check.Status = "Não concluído"
			check.Message = err.Error()
			if check.SourceURL == "" { check.SourceURL = rc.resource.URL }
			op.ZARC = check
			out.Warnings = append(out.Warnings, "ZARC "+op.RefBacen+"/"+op.Order+": "+err.Error())
			continue
		}
		op.ZARC = check
	}
}

func resolveZARCResource(ctx context.Context, safraIni, safraFin int) (zarcResource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zarcCKANPackageURL, nil)
	if err != nil { return zarcResource{}, err }
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ViaVerdeCAR/"+AppVersion)
	resp, err := (&http.Client{Timeout:20*time.Second}).Do(req)
	if err != nil { return zarcResource{}, err }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zarcResource{}, fmt.Errorf("MAPA/CKAN respondeu HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Success bool `json:"success"`
		Result struct {
			Resources []zarcResource `json:"resources"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil { return zarcResource{}, err }
	if !payload.Success { return zarcResource{}, errors.New("MAPA/CKAN não confirmou a consulta") }
	want := fmt.Sprintf("%d/%d", safraIni, safraFin)
	for _, r := range payload.Result.Resources {
		name := strings.ToUpper(strings.TrimSpace(r.Name))
		if strings.EqualFold(strings.TrimSpace(r.Format), "CSV") && strings.Contains(name, strings.ToUpper(want)) && strings.Contains(name, "SAFRA") {
			if strings.TrimSpace(r.URL) == "" { continue }
			return r, nil
		}
	}
	return zarcResource{}, fmt.Errorf("recurso CSV da safra %s não localizado no catálogo oficial", want)
}

func scanZARCForOperation(path string, resource zarcResource, car CARResult, op CreditOperationIntelligence, safraIni, safraFin int) (ZARCCheck, error) {
	start, err := parseCreditDate(op.PlantingStart)
	if err != nil { return ZARCCheck{}, err }
	end := start
	if strings.TrimSpace(op.PlantingEnd) != "" {
		if parsed, e := parseCreditDate(op.PlantingEnd); e == nil { end = parsed }
	}
	if end.Before(start) { end = start }
	wantedPeriods := decendiosBetween(start, end)
	if len(wantedPeriods) == 0 { wantedPeriods = []int{zarcDecendio(start)} }

	out := ZARCCheck{
		Checked:true, Available:true, Safra:fmt.Sprintf("%d/%d",safraIni,safraFin),
		Culture:op.Product, Cycle:op.CultivarCycle, Soil:op.Soil, Management:zarcManagementFromOperation(op),
		Municipality:car.Municipality, UF:car.UF, PlantingStart:op.PlantingStart, PlantingEnd:op.PlantingEnd,
		SourceURL:resource.URL, UpdatedAt:resource.LastModified,
	}
	f, err := os.Open(path)
	if err != nil { return out, err }
	defer f.Close()
	br := bufio.NewReader(f)
	first, err := br.ReadString('\n')
	if err != nil && err != io.EOF { return out, err }
	delim := ','
	if strings.Count(first, ";") > strings.Count(first, ",") { delim = ';' }
	if _, err := f.Seek(0, 0); err != nil { return out, err }
	cr := csv.NewReader(f)
	cr.Comma = delim
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	headers, err := cr.Read()
	if err != nil { return out, err }
	h := map[string]int{}
	for i, name := range headers { h[zarcKey(name)] = i }

	required := []string{"nome_cultura","uf","municipio","cod_ciclo","cod_solo"}
	for _, key := range required {
		if _, ok := h[key]; !ok { return out, fmt.Errorf("Tábua ZARC sem coluna esperada %s", key) }
	}
	wantCode := strings.TrimLeft(strings.TrimSpace(car.MunicipalityCode), "0")
	wantMunicipality := normalizeZARCText(car.Municipality)
	wantUF := strings.ToUpper(strings.TrimSpace(car.UF))
	wantCulture := canonicalZARCCulture(op.Product)
	wantCycle := zarcCycleCode(op.CultivarCycle, op.CultivarCycleCode)
	wantSoil := zarcSoilCode(op.Soil, op.SoilCode)
	wantManagement := zarcManagementCode(op)

	type candidate struct {
		portaria string
		values map[int]string
		culture string
		cycle string
		soil string
		management string
	}
	var candidates []candidate
	for {
		row, e := cr.Read()
		if e == io.EOF { break }
		if e != nil { continue }
		get := func(name string) string {
			i, ok := h[zarcKey(name)]
			if !ok || i < 0 || i >= len(row) { return "" }
			return strings.TrimSpace(row[i])
		}
		rowUF := strings.ToUpper(get("UF"))
		if wantUF != "" && rowUF != "" && rowUF != wantUF { continue }
		rowGeo := strings.TrimLeft(get("geocodigo"), "0")
		rowMun := normalizeZARCText(get("municipio"))
		municipalityOK := (wantCode != "" && rowGeo != "" && rowGeo == wantCode) || (wantMunicipality != "" && rowMun == wantMunicipality)
		if !municipalityOK { continue }
		rowCulture := canonicalZARCCulture(get("Nome_cultura"))
		if !zarcCultureCompatible(wantCulture, rowCulture) { continue }
		rowCycle := normalizeZARCCode(get("Cod_Ciclo"))
		if wantCycle != "" && rowCycle != "" && rowCycle != wantCycle { continue }
		rowSoil := normalizeZARCCode(get("Cod_Solo"))
		if wantSoil != "" && rowSoil != "" && rowSoil != wantSoil { continue }
		rowManagement := normalizeZARCCode(get("Cod_Outros_Manejos"))
		if wantManagement != "" && rowManagement != "" && rowManagement != wantManagement { continue }
		c := candidate{
			portaria:get("Portaria"), values:map[int]string{}, culture:get("Nome_cultura"),
			cycle:rowCycle, soil:rowSoil, management:get("Nome_Outros_Manejos"),
		}
		for _, d := range wantedPeriods { c.values[d] = get(fmt.Sprintf("dec%d",d)) }
		candidates = append(candidates,c)
	}
	out.Candidates = len(candidates)
	if len(candidates) == 0 {
		out.Matched = false
		out.Status = "Sem combinação exata"
		out.Message = "Nenhuma linha da Tábua de Risco coincidiu com município, cultura e os filtros técnicos disponíveis na operação. Isso não deve ser interpretado automaticamente como fora do ZARC; confira cultura, ciclo, solo e manejo."
		return out, nil
	}
	out.Matched = true
	if out.Management == "" { out.Management = candidates[0].management }
	if out.Portaria == "" { out.Portaria = candidates[0].portaria }
	if out.Culture == "" { out.Culture = candidates[0].culture }

	allIndicated := true
	anyIndicated := false
	for _, d := range wantedPeriods {
		bestRaw := ""
		bestRisk := 0
		indicated := false
		for _, c := range candidates {
			raw := strings.TrimSpace(c.values[d])
			if !zarcRawIndicated(raw) { continue }
			indicated = true
			risk := zarcRiskPercent(raw)
			if bestRaw == "" || (risk > 0 && (bestRisk == 0 || risk < bestRisk)) {
				bestRaw, bestRisk = raw, risk
			}
		}
		if !indicated { allIndicated = false } else { anyIndicated = true }
		out.Periods = append(out.Periods, ZARCPeriod{Decendio:d,Raw:bestRaw,RiskPct:bestRisk,Indicated:indicated})
	}
	switch {
	case allIndicated:
		out.Status = "Indicação localizada"
		out.Message = "Há indicação publicada para todos os decêndios do período de plantio informado, considerando as linhas compatíveis com município, cultura, ciclo, solo e manejo disponíveis. Confirme a Portaria e eventuais filtros de clima antes de concluir o enquadramento."
	case anyIndicated:
		out.Status = "Indicação parcial"
		out.Message = "Parte do período de plantio informado possui indicação publicada e parte não possui nas linhas compatíveis encontradas. Confira os decêndios e a Portaria vigente."
	default:
		out.Status = "Sem indicação no período"
		out.Message = "Foram localizadas linhas compatíveis para município, cultura e filtros técnicos, mas nenhum dos decêndios do período informado trouxe indicação. Confira a Portaria antes de concluir que o plantio está fora do ZARC."
	}
	return out, nil
}

func parseCreditDate(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	for _, layout := range []string{"2006-01-02", time.RFC3339, "02/01/2006"} {
		if t, err := time.Parse(layout, v); err == nil { return t, nil }
	}
	return time.Time{}, fmt.Errorf("data não reconhecida: %s", v)
}

func zarcCropSeason(t time.Time) (int,int) {
	if t.Month() >= time.July { return t.Year(), t.Year()+1 }
	return t.Year()-1, t.Year()
}

func zarcDecendio(t time.Time) int {
	part := 1
	if t.Day() > 20 { part = 3 } else if t.Day() > 10 { part = 2 }
	return (int(t.Month())-1)*3 + part
}

func decendiosBetween(start,end time.Time) []int {
	if end.Before(start) { end = start }
	seen := map[int]bool{}
	var out []int
	cur := start
	for !cur.After(end) {
		d := zarcDecendio(cur)
		if !seen[d] { seen[d]=true; out=append(out,d) }
		cur = cur.AddDate(0,0,1)
		if len(out) >= 36 { break }
	}
	return out
}

func zarcCycleCode(label, raw string) string {
	n := normalizeZARCText(label)
	switch {
	case strings.Contains(n,"GRUPO VI"): return "26"
	case strings.Contains(n,"GRUPO V"): return "25"
	case strings.Contains(n,"GRUPO IV"): return "24"
	case strings.Contains(n,"GRUPO III"): return "22"
	case strings.Contains(n,"GRUPO II"): return "21"
	case strings.Contains(n,"GRUPO I"): return "20"
	case strings.Contains(n,"SEMIPERENE"): return "19"
	case strings.Contains(n,"PERENE"): return "13"
	}
	r := normalizeZARCCode(raw)
	if r=="13"||r=="19"||r=="20"||r=="21"||r=="22"||r=="24"||r=="25"||r=="26" { return r }
	return ""
}

func zarcSoilCode(label, raw string) string {
	n := normalizeZARCText(label)
	for i:=1;i<=6;i++ {
		if strings.Contains(n,fmt.Sprintf("AD%d",i)) { return strconv.Itoa(10+i) }
	}
	switch {
	case strings.Contains(n,"ARENOS"): return "1"
	case strings.Contains(n,"TEXTURA MEDIA")||strings.Contains(n,"MEDIO"): return "2"
	case strings.Contains(n,"ARGIL"): return "3"
	}
	r := normalizeZARCCode(raw)
	if r=="1"||r=="2"||r=="3"||(len(r)==2&&r>="11"&&r<="16") { return r }
	return ""
}

func zarcManagementCode(op CreditOperationIntelligence) string {
	n := normalizeZARCText(op.Irrigation+" "+op.Agriculture)
	if strings.Contains(n,"CONTROLE DE GEADA") { return "3" }
	if strings.Contains(n,"IRRIG") { return "2" }
	if n != "" && !strings.Contains(n,"CODIGO") { return "1" }
	return ""
}

func zarcManagementFromOperation(op CreditOperationIntelligence) string {
	switch zarcManagementCode(op) {
	case "1": return "Sequeiro"
	case "2": return "Irrigado"
	case "3": return "Irrigado com controle de geada"
	default: return ""
	}
}

func zarcRawIndicated(v string) bool {
	n := strings.ToUpper(strings.TrimSpace(v))
	return n != "" && n != "-" && n != "0" && n != "NA" && n != "N/A" && n != "NULL"
}

func zarcRiskPercent(v string) int {
	n := strings.TrimSpace(strings.TrimSuffix(v,"%"))
	f, err := strconv.ParseFloat(strings.ReplaceAll(n,",","."),64)
	if err != nil { return 0 }
	i := int(f+0.5)
	switch i { case 20,30,40,50: return i }
	return 0
}

func zarcCultureCompatible(a,b string) bool {
	if a==""||b=="" { return false }
	if a==b { return true }
	if len(a)>=4&&strings.Contains(b,a) { return true }
	if len(b)>=4&&strings.Contains(a,b) { return true }
	return false
}

func canonicalZARCCulture(v string) string {
	n := normalizeZARCText(v)
	repl := []string{" EM GRAO"," GRAO"," GRAOS"," SEMENTE"," SEMENTES"," CULTURA DE "," CULTIVO DE "}
	for _,x:=range repl { n=strings.ReplaceAll(n,x,"") }
	return strings.TrimSpace(n)
}

func normalizeZARCCode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimLeft(v,"0")
	if v=="" { return "0" }
	return v
}

func zarcKey(v string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(v,"\ufeff")))
}

func normalizeZARCText(v string) string {
	v = strings.ToUpper(strings.TrimSpace(v))
	r := strings.NewReplacer(
		"Á","A","À","A","Â","A","Ã","A","Ä","A",
		"É","E","È","E","Ê","E","Ë","E",
		"Í","I","Ì","I","Î","I","Ï","I",
		"Ó","O","Ò","O","Ô","O","Õ","O","Ö","O",
		"Ú","U","Ù","U","Û","U","Ü","U",
		"Ç","C","-"," ","_"," ","/"," ",
	)
	v = r.Replace(v)
	return strings.Join(strings.Fields(v)," ")
}
