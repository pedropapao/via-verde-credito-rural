package main

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CreditMarketItem struct {
	Scope     string  `json:"scope"`
	Kind      string  `json:"kind"`
	Year      string  `json:"year"`
	Label     string  `json:"label"`
	Detail    string  `json:"detail"`
	Contracts float64 `json:"contracts"`
	Value     float64 `json:"value"`
}

type CreditMarketContext struct {
	Available          bool               `json:"available"`
	Municipality       string             `json:"municipality"`
	UF                 string             `json:"uf"`
	MunicipalProducts  []CreditMarketItem `json:"municipal_products"`
	StateInstitutions  []CreditMarketItem `json:"state_institutions"`
	StatePrograms      []CreditMarketItem `json:"state_programs"`
	NationalSources    []CreditMarketItem `json:"national_sources"`
	SourceURL          string             `json:"source_url"`
	Message            string             `json:"message"`
	Warnings           []string           `json:"warnings"`
}

func queryCreditMarketContext(ctx context.Context, municipality, uf, municipalityCode string) (CreditMarketContext, error) {
	out := CreditMarketContext{
		Municipality: municipality,
		UF: strings.ToUpper(strings.TrimSpace(uf)),
		SourceURL: "https://dadosabertos.bcb.gov.br/dataset/matrizdadoscreditorural",
	}
	years := []int{time.Now().Year(), time.Now().Year()-1}
	var anySuccess bool
	for _, year := range years {
		for _, spec := range []struct{
			endpoint, kind string
		}{
			{"CusteioMunicipioProduto","Custeio"},
			{"InvestMunicipioProduto","Investimento"},
		} {
			rows, err := queryMarketRows(ctx, spec.endpoint, marketMunicipalityFilters(municipality, uf, municipalityCode, year))
			if err != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("%s %d: %v", spec.kind, year, err))
				continue
			}
			anySuccess = true
			rows = filterMarketMunicipalityRows(rows, municipality, uf, municipalityCode, strconv.Itoa(year))
			out.MunicipalProducts = append(out.MunicipalProducts, aggregateMarketRows(rows, "Município", spec.kind, strconv.Itoa(year), "product")...)
		}

		rows, err := queryMarketRows(ctx, "SegmentoIFRegiaoUF", marketUFFilters(uf, year))
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("instituições %d: %v", year, err))
		} else {
			anySuccess = true
			rows = filterMarketUFRows(rows, uf, strconv.Itoa(year))
			out.StateInstitutions = append(out.StateInstitutions, aggregateMarketRows(rows, "UF", "Instituição financeira", strconv.Itoa(year), "institution")...)
		}

		rows, err = queryMarketRows(ctx, "ProgramaSubprogramaRegiaoUF", marketUFFilters(uf, year))
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("programas %d: %v", year, err))
		} else {
			anySuccess = true
			rows = filterMarketUFRows(rows, uf, strconv.Itoa(year))
			out.StatePrograms = append(out.StatePrograms, aggregateMarketRows(rows, "UF", "Programa", strconv.Itoa(year), "program")...)
		}

		rows, err = queryMarketRows(ctx, "FonteRecursos", []string{
			fmt.Sprintf("AnoEmissao eq '%s'", strconv.Itoa(year)),
			"",
		})
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("fontes de recursos %d: %v", year, err))
		} else {
			anySuccess = true
			rows = filterMarketYearRows(rows, strconv.Itoa(year))
			out.NationalSources = append(out.NationalSources, aggregateMarketRows(rows, "Brasil", "Fonte de recursos", strconv.Itoa(year), "resource")...)
		}
	}
	out.MunicipalProducts = topMarketItems(out.MunicipalProducts, 12)
	out.StateInstitutions = topMarketItems(out.StateInstitutions, 10)
	out.StatePrograms = topMarketItems(out.StatePrograms, 10)
	out.NationalSources = topMarketItems(out.NationalSources, 10)
	out.Available = anySuccess
	if anySuccess {
		out.Message = "Contexto agregado da MDCR. Produtos são filtrados pelo município; instituições e programas usam as visões públicas por UF. Esses dados descrevem o mercado, não o risco de crédito individual do produtor."
		return out, nil
	}
	out.Message = "As visões complementares da MDCR não puderam ser consultadas nesta tentativa."
	if len(out.Warnings)>0 {
		return out, fmt.Errorf("%s", strings.Join(out.Warnings, " | "))
	}
	return out, nil
}

func marketMunicipalityFilters(municipality, uf, code string, year int) []string {
	y := strconv.Itoa(year)
	name := strings.ToUpper(strings.TrimSpace(municipality))
	uf = strings.ToUpper(strings.TrimSpace(uf))
	code = strings.TrimSpace(code)
	var filters []string
	if code != "" {
		filters = append(filters,
			fmt.Sprintf("codMunicIbge eq '%s' and AnoEmissao eq '%s'",odataEscape(code),y),
			fmt.Sprintf("CodMunicIBGE eq '%s' and AnoEmissao eq '%s'",odataEscape(code),y),
		)
	}
	if name != "" && uf != "" {
		filters = append(filters,
			fmt.Sprintf("Municipio eq '%s' and nomeUF eq '%s' and AnoEmissao eq '%s'",odataEscape(name),odataEscape(uf),y),
			fmt.Sprintf("Municipio eq '%s' and UF eq '%s' and AnoEmissao eq '%s'",odataEscape(name),odataEscape(uf),y),
		)
	}
	if name != "" {
		filters = append(filters,fmt.Sprintf("Municipio eq '%s' and AnoEmissao eq '%s'",odataEscape(name),y))
	}
	filters = append(filters, "")
	return filters
}

func marketUFFilters(uf string, year int) []string {
	y:=strconv.Itoa(year)
	uf=strings.ToUpper(strings.TrimSpace(uf))
	if uf=="" { return []string{fmt.Sprintf("AnoEmissao eq '%s'",y)} }
	return []string{
		fmt.Sprintf("nomeUF eq '%s' and AnoEmissao eq '%s'",odataEscape(uf),y),
		fmt.Sprintf("UF eq '%s' and AnoEmissao eq '%s'",odataEscape(uf),y),
		fmt.Sprintf("AnoEmissao eq '%s'",y),
		"",
	}
}

func queryMarketRows(ctx context.Context, endpoint string, filters []string) ([]map[string]any,error) {
	var lastErr error
	for _, filter := range filters {
		q:=url.Values{}
		q.Set("$format","json")
		q.Set("$top","5000")
		if strings.TrimSpace(filter)!="" { q.Set("$filter",filter) }
		target:=bcbSicorODataBase+endpoint+"?"+q.Encode()
		rows,err:=getODataRows(ctx,target)
		if err!=nil { lastErr=err; continue }
		if len(rows)>0 { return rows,nil }
	}
	if lastErr!=nil { return nil,lastErr }
	return nil,nil
}

func filterMarketMunicipalityRows(rows []map[string]any, municipality, uf, municipalityCode, year string) []map[string]any {
	wantName:=strings.ToUpper(strings.TrimSpace(municipality))
	wantUF:=strings.ToUpper(strings.TrimSpace(uf))
	wantCode:=strings.TrimLeft(strings.TrimSpace(municipalityCode),"0")
	var out []map[string]any
	for _,r:=range rows{
		if y:=marketYear(r);year!=""&&y!=""&&y!=year{continue}
		rowUF:=strings.ToUpper(strings.TrimSpace(firstNonEmptyStringMapValue(r,"nomeUF","UF","uf")))
		if wantUF!=""&&rowUF!=""&&rowUF!=wantUF{continue}
		rowName:=strings.ToUpper(strings.TrimSpace(firstNonEmptyStringMapValue(r,"Municipio","municipio","NomeMunicipio","nomeMunicipio")))
		rowCode:=strings.TrimLeft(strings.TrimSpace(firstNonEmptyStringMapValue(r,"codMunicIbge","CodMunicIBGE","codIbge","geocodigo")),"0")
		codeMatch:=wantCode!=""&&rowCode!=""&&wantCode==rowCode
		nameMatch:=wantName!=""&&rowName!=""&&wantName==rowName
		if wantCode!=""||wantName!="" {
			if !codeMatch&&!nameMatch{continue}
		}
		out=append(out,r)
	}
	return out
}

func filterMarketUFRows(rows []map[string]any, uf, year string) []map[string]any {
	wantUF:=strings.ToUpper(strings.TrimSpace(uf))
	var out []map[string]any
	for _,r:=range rows{
		if y:=marketYear(r);year!=""&&y!=""&&y!=year{continue}
		rowUF:=strings.ToUpper(strings.TrimSpace(firstNonEmptyStringMapValue(r,"nomeUF","UF","uf","Estado")))
		if wantUF!=""&&rowUF!=""&&rowUF!=wantUF{continue}
		out=append(out,r)
	}
	return out
}

func filterMarketYearRows(rows []map[string]any, year string) []map[string]any {
	var out []map[string]any
	for _, r := range rows {
		y := marketYear(r)
		if year != "" && y != "" && y != year {
			continue
		}
		out = append(out, r)
	}
	return out
}

func aggregateMarketRows(rows []map[string]any, scope, kind, year, mode string) []CreditMarketItem {
	type key struct{ label,detail string }
	grouped:=map[key]*CreditMarketItem{}
	for _,r:=range rows{
		label,detail:=marketLabels(r,mode)
		if strings.TrimSpace(label)==""{continue}
		k:=key{label:label,detail:detail}
		if grouped[k]==nil{
			grouped[k]=&CreditMarketItem{Scope:scope,Kind:kind,Year:year,Label:label,Detail:detail}
		}
		grouped[k].Contracts+=marketContracts(r)
		grouped[k].Value+=marketValue(r)
	}
	out:=make([]CreditMarketItem,0,len(grouped))
	for _,x:=range grouped{out=append(out,*x)}
	return out
}

func marketLabels(r map[string]any, mode string)(string,string){
	switch mode{
	case "product":
		return firstNonEmptyStringMapValue(r,"Produto","produto","nomeProduto","NomeProduto","ProdutoNome","DESCRICAO_PRODUTO"), firstNonEmptyStringMapValue(r,"Atividade","atividade","Finalidade")
	case "institution":
		return firstNonEmptyStringMapValue(r,"nomeIF","NomeIF","IF","InstituicaoFinanceira","Instituição Financeira","NomeInstituicao","NomeInstituição"), firstNonEmptyStringMapValue(r,"Segmento","segmento","nomeSegmento","NomeSegmento")
	case "program":
		return firstNonEmptyStringMapValue(r,"Programa","programa","nomePrograma","NomePrograma","DESCRICAO_PROGRAMA"), firstNonEmptyStringMapValue(r,"Subprograma","subprograma","nomeSubprograma","NomeSubprograma","DESCRICAO_SUBPROGRAMA")
	case "resource":
		return firstNonEmptyStringMapValue(r,"FonteRecursos","FonteRecurso","fonteRecursos","nomeFonteRecursos","NomeFonteRecursos","DESCRICAO_FONTE_RECURSOS"), firstNonEmptyStringMapValue(r,"Finalidade","Atividade","Segmento")
	default:
		return "",""
	}
}

func marketContracts(r map[string]any) float64 {
	for _,k:=range []string{"QtdContratos","QtdContrato","QuantidadeContratos","QtdCusteio","QtdInvestimento","QtdComercializacao","QtdIndustrializacao","Quantidade"}{
		if v:=numberMapValue(r,k);v!=0{return v}
	}
	for k:=range r{
		u:=strings.ToUpper(k)
		if strings.Contains(u,"QTD")||strings.Contains(u,"QUANT") {
			if v:=numberMapValue(r,k);v!=0{return v}
		}
	}
	return 0
}

func marketValue(r map[string]any) float64 {
	for _,k:=range []string{"VlContratos","VlContrato","ValorContratos","VlCusteio","VlInvestimento","VlComercializacao","VlIndustrializacao","Valor"}{
		if v:=numberMapValue(r,k);v!=0{return v}
	}
	for k:=range r{
		u:=strings.ToUpper(k)
		if strings.HasPrefix(u,"VL")||strings.Contains(u,"VALOR"){
			if v:=numberMapValue(r,k);v!=0{return v}
		}
	}
	return 0
}

func marketYear(r map[string]any) string {
	return strings.TrimSpace(firstNonEmptyStringMapValue(r,"AnoEmissao","Ano","ano","AnoContrato","ANO_EMISSAO"))
}

func topMarketItems(items []CreditMarketItem, n int) []CreditMarketItem {
	sort.SliceStable(items,func(i,j int)bool{
		if items[i].Year==items[j].Year {
			if items[i].Value==items[j].Value{return items[i].Contracts>items[j].Contracts}
			return items[i].Value>items[j].Value
		}
		return items[i].Year>items[j].Year
	})
	if len(items)>n { items=items[:n] }
	return items
}
