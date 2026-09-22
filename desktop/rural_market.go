package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type BCBMarketItem struct {
	Scope     string  `json:"scope"`
	Kind      string  `json:"kind"`
	Year      string  `json:"year"`
	Name      string  `json:"name"`
	Detail    string  `json:"detail"`
	Contracts float64 `json:"contracts"`
	Value     float64 `json:"value"`
}

type BCBRuralMarketIntelligence struct {
	Available     bool                `json:"available"`
	Municipality  string              `json:"municipality"`
	UF            string              `json:"uf"`
	FromYear      int                 `json:"from_year"`
	ToYear        int                 `json:"to_year"`
	Trend         []BCBRuralCreditRow `json:"trend"`
	Products      []BCBMarketItem     `json:"products"`
	Institutions  []BCBMarketItem     `json:"institutions"`
	Programs      []BCBMarketItem     `json:"programs"`
	Message       string              `json:"message"`
	Warnings      []string            `json:"warnings"`
	SourceURL     string              `json:"source_url"`
	CheckedAt     string              `json:"checked_at"`
}

func (a *App) GetRuralCreditMarket(propertyID int64) (BCBRuralMarketIntelligence, error) {
	car, err := a.carForRuralCredit(propertyID)
	if err != nil {
		return BCBRuralMarketIntelligence{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()
	return queryBCBRuralMarket(ctx, car.Municipality, car.UF, car.MunicipalityCode)
}

func queryBCBRuralMarket(ctx context.Context, municipality, uf, municipalityCode string) (BCBRuralMarketIntelligence, error) {
	nowYear := time.Now().Year()
	out := BCBRuralMarketIntelligence{
		Municipality: municipality, UF: strings.ToUpper(strings.TrimSpace(uf)),
		FromYear: nowYear - 4, ToYear: nowYear,
		SourceURL: "https://dadosabertos.bcb.gov.br/dataset/matrizdadoscreditorural",
		CheckedAt: time.Now().Format(time.RFC3339),
	}

	type trendResult struct {
		rows []map[string]any
		err error
		year int
	}
	trendCh := make(chan trendResult, 5)
	for year := out.FromYear; year <= out.ToYear; year++ {
		y := year
		go func() {
			rows, e := queryBCBMunicipalityAggregate(ctx, municipality, uf, municipalityCode, y)
			trendCh <- trendResult{rows: rows, err: e, year: y}
		}()
	}
	var trendRaw []map[string]any
	for i := 0; i < 5; i++ {
		r := <-trendCh
		if r.err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("evolução %d: %v", r.year, r.err))
			continue
		}
		trendRaw = append(trendRaw, r.rows...)
	}
	out.Trend = aggregateBCBMunicipalityRows(trendRaw)
	sort.SliceStable(out.Trend, func(i,j int) bool {
		if out.Trend[i].Year == out.Trend[j].Year {
			return out.Trend[i].Value > out.Trend[j].Value
		}
		return out.Trend[i].Year > out.Trend[j].Year
	})

	// Produtos no município: os endpoints oficiais publicam recorte municipal
	// para Custeio e Investimento. Não extrapolamos Comercialização/Industrialização
	// para município quando a API não oferece esse mesmo detalhamento.
	type itemResult struct {
		items []BCBMarketItem
		err error
		label string
	}
	itemCh := make(chan itemResult, 6)
	for _, spec := range []struct{
		endpoint, kind string
	}{
		{"CusteioMunicipioProduto","Custeio"},
		{"InvestMunicipioProduto","Investimento"},
	} {
		s := spec
		go func() {
			items, e := queryBCBMunicipalityProduct(ctx, s.endpoint, s.kind, municipality, uf, municipalityCode, nowYear)
			if e == nil && len(items)==0 {
				items, e = queryBCBMunicipalityProduct(ctx, s.endpoint, s.kind, municipality, uf, municipalityCode, nowYear-1)
			}
			itemCh <- itemResult{items:items,err:e,label:s.kind+" por produto"}
		}()
	}
	go func(){
		items,e:=queryBCBUFMarket(ctx,"SegmentoIFRegiaoUF","Instituição",uf,nowYear)
		if e==nil&&len(items)==0{items,e=queryBCBUFMarket(ctx,"SegmentoIFRegiaoUF","Instituição",uf,nowYear-1)}
		itemCh<-itemResult{items:items,err:e,label:"instituições na UF"}
	}()
	go func(){
		items,e:=queryBCBUFMarket(ctx,"ProgramaSubprogramaRegiaoUF","Programa",uf,nowYear)
		if e==nil&&len(items)==0{items,e=queryBCBUFMarket(ctx,"ProgramaSubprogramaRegiaoUF","Programa",uf,nowYear-1)}
		itemCh<-itemResult{items:items,err:e,label:"programas na UF"}
	}()

	for i:=0;i<4;i++ {
		r:=<-itemCh
		if r.err!=nil {
			out.Warnings=append(out.Warnings,r.label+": "+r.err.Error())
			continue
		}
		switch {
		case strings.Contains(r.label,"produto"):
			out.Products=append(out.Products,r.items...)
		case strings.Contains(r.label,"institui"):
			out.Institutions=r.items
		case strings.Contains(r.label,"program"):
			out.Programs=r.items
		}
	}
	out.Products=aggregateMarketItems(out.Products,12)
	out.Institutions=aggregateMarketItems(out.Institutions,12)
	out.Programs=aggregateMarketItems(out.Programs,12)

	out.Available=len(out.Trend)>0||len(out.Products)>0||len(out.Institutions)>0||len(out.Programs)>0
	if out.Available {
		out.Message="Inteligência pública do MDCR montada. Evolução e produtos usam o município do CAR; instituições e programas são contexto da UF e não identificam o banco ou o programa do produtor."
	} else {
		out.Message="A API pública do MDCR não retornou contexto de mercado utilizável nesta tentativa."
	}
	if !out.Available && len(out.Warnings)>0 {
		return out, errors.New(strings.Join(out.Warnings," | "))
	}
	return out,nil
}

func queryBCBMunicipalityProduct(ctx context.Context, endpoint, kind, municipality, uf, municipalityCode string, year int) ([]BCBMarketItem,error) {
	rows,err:=queryBCBODataMunicipalityEndpoint(ctx,endpoint,municipality,uf,municipalityCode,year)
	if err!=nil{return nil,err}
	group:=map[string]*BCBMarketItem{}
	for _,r:=range rows{
		name:=firstNonEmptyStringMapValue(r,"Produto","produto","NomeProduto","nomeProduto","Nome_Produto","ProdutoEmpreendimento","DESCRICAO_PRODUTO")
		if name==""{name=findStringFieldByTokens(r,[]string{"PRODUTO"})}
		if name==""{continue}
		qty:=numberMapValueCandidates(r,
			[]string{"QtdCusteio","QtdInvestimento","QuantidadeContratos","QtdContratos","Quantidade"},
			[]string{"QTD","QUANT"})
		val:=numberMapValueCandidates(r,
			[]string{"VlCusteio","VlInvestimento","ValorContratos","Valor","VL_CONTRATOS"},
			[]string{"VL","VALOR"})
		k:=kind+"|"+strings.ToUpper(strings.TrimSpace(name))
		if group[k]==nil{group[k]=&BCBMarketItem{Scope:"Município",Kind:kind,Year:strconv.Itoa(year),Name:name}}
		group[k].Contracts+=qty;group[k].Value+=val
	}
	var out []BCBMarketItem
	for _,x:=range group{if x.Contracts!=0||x.Value!=0{out=append(out,*x)}}
	sort.SliceStable(out,func(i,j int)bool{return out[i].Value>out[j].Value})
	return out,nil
}

func queryBCBODataMunicipalityEndpoint(ctx context.Context, endpoint, municipality, uf, municipalityCode string, year int) ([]map[string]any,error) {
	yearText:=strconv.Itoa(year)
	name:=strings.ToUpper(strings.TrimSpace(municipality))
	code:=strings.TrimSpace(municipalityCode)
	uf=strings.ToUpper(strings.TrimSpace(uf))
	var filters []string
	if code!="" {
		filters=append(filters,
			fmt.Sprintf("codMunicIbge eq '%s' and AnoEmissao eq '%s'",odataEscape(code),yearText),
			fmt.Sprintf("codIbge eq '%s' and AnoEmissao eq '%s'",odataEscape(code),yearText),
		)
	}
	if name!="" {
		if uf!="" {
			filters=append(filters,fmt.Sprintf("Municipio eq '%s' and nomeUF eq '%s' and AnoEmissao eq '%s'",odataEscape(name),odataEscape(uf),yearText))
		}
		filters=append(filters,fmt.Sprintf("Municipio eq '%s' and AnoEmissao eq '%s'",odataEscape(name),yearText))
	}
	return queryBCBODataValidated(ctx,endpoint,filters,func(rows []map[string]any)[]map[string]any{
		return filterBCBMunicipalityRows(rows,municipality,uf,municipalityCode,yearText)
	})
}

func queryBCBUFMarket(ctx context.Context, endpoint, kind, uf string, year int) ([]BCBMarketItem,error) {
	abbr:=strings.ToUpper(strings.TrimSpace(uf))
	full:=brazilUFName(abbr)
	yearText:=strconv.Itoa(year)
	var filters []string
	for _,u:=range []string{full,abbr} {
		if u==""{continue}
		filters=append(filters,
			fmt.Sprintf("nomeUF eq '%s' and AnoEmissao eq '%s'",odataEscape(u),yearText),
			fmt.Sprintf("UF eq '%s' and AnoEmissao eq '%s'",odataEscape(u),yearText),
		)
	}
	rows,err:=queryBCBODataValidated(ctx,endpoint,filters,func(rows []map[string]any)[]map[string]any{
		return filterBCBUFYearRows(rows,abbr,full,yearText)
	})
	if err!=nil{return nil,err}
	var out []BCBMarketItem
	for _,r:=range rows{
		var name,detail string
		if kind=="Instituição" {
			name=firstNonEmptyStringMapValue(r,"NomeIF","nomeIF","IF","InstituicaoFinanceira","Instituição Financeira","NOME_IF")
			if name==""{name=findStringFieldByTokens(r,[]string{"NOME","IF"})}
			if name==""{name=findStringFieldByTokens(r,[]string{"INSTITU"})}
			detail=firstNonEmptyStringMapValue(r,"Segmento","segmento","NomeSegmento","NOME_SEGMENTO")
		}else{
			name=firstNonEmptyStringMapValue(r,"Programa","programa","NomePrograma","nomePrograma","NOME_PROGRAMA")
			if name==""{name=findStringFieldByTokens(r,[]string{"PROGRAMA"})}
			detail=firstNonEmptyStringMapValue(r,"Subprograma","subprograma","NomeSubprograma","nomeSubprograma","NOME_SUBPROGRAMA")
		}
		if name==""{continue}
		qty:=numberMapValueCandidates(r,[]string{"QtdContratos","QuantidadeContratos","Qtd","Quantidade"},[]string{"QTD","QUANT"})
		val:=numberMapValueCandidates(r,[]string{"VlContratos","ValorContratos","Valor","VL_CREDITO"},[]string{"VL","VALOR"})
		if qty==0&&val==0{continue}
		out=append(out,BCBMarketItem{Scope:"UF "+abbr,Kind:kind,Year:yearText,Name:name,Detail:detail,Contracts:qty,Value:val})
	}
	return aggregateMarketItems(out,12),nil
}

func queryBCBODataValidated(ctx context.Context, endpoint string, filters []string, validate func([]map[string]any)[]map[string]any)([]map[string]any,error){
	var lastErr error
	sawEmpty:=false
	sawIgnored:=false
	for _,filter:=range filters{
		q:=url.Values{};q.Set("$format","json");q.Set("$top","5000");q.Set("$filter",filter)
		rows,err:=getODataRows(ctx,bcbSicorODataBase+endpoint+"?"+q.Encode())
		if err!=nil{lastErr=err;continue}
		if len(rows)==0{sawEmpty=true;continue}
		matched:=validate(rows)
		if len(matched)>0{return matched,nil}
		sawIgnored=true
	}
	if sawEmpty&&!sawIgnored{return nil,nil}
	if sawIgnored{return nil,errors.New("a API retornou linhas fora do filtro solicitado")}
	if lastErr!=nil{return nil,lastErr}
	return nil,errors.New("consulta OData sem resposta utilizável")
}

func filterBCBUFYearRows(rows []map[string]any, abbr, full, year string)[]map[string]any{
	abbr=strings.ToUpper(strings.TrimSpace(abbr));full=strings.ToUpper(strings.TrimSpace(full))
	var out []map[string]any
	for _,r:=range rows{
		y:=strings.TrimSpace(firstNonEmptyStringMapValue(r,"AnoEmissao","ano","Ano"))
		if year!=""&&(y==""||y!=year){continue}
		rowUF:=strings.ToUpper(strings.TrimSpace(firstNonEmptyStringMapValue(r,"nomeUF","UF","uf","NomeUF")))
		if rowUF==""|| (rowUF!=abbr&&rowUF!=full){continue}
		out=append(out,r)
	}
	return out
}

func aggregateMarketItems(items []BCBMarketItem,limit int)[]BCBMarketItem{
	type key struct{scope,kind,year,name,detail string}
	m:=map[key]*BCBMarketItem{}
	for _,x:=range items{
		k:=key{x.Scope,x.Kind,x.Year,strings.ToUpper(strings.TrimSpace(x.Name)),strings.ToUpper(strings.TrimSpace(x.Detail))}
		if m[k]==nil{copy:=x;m[k]=&copy}else{m[k].Contracts+=x.Contracts;m[k].Value+=x.Value}
	}
	out:=make([]BCBMarketItem,0,len(m))
	for _,x:=range m{out=append(out,*x)}
	sort.SliceStable(out,func(i,j int)bool{
		if out[i].Value==out[j].Value{return out[i].Contracts>out[j].Contracts}
		return out[i].Value>out[j].Value
	})
	if limit>0&&len(out)>limit{out=out[:limit]}
	return out
}

func numberMapValueCandidates(m map[string]any,exact,tokens []string)float64{
	for _,k:=range exact{if v:=numberMapValue(m,k);v!=0{return v}}
	type candidate struct{k string;v float64}
	var cs []candidate
	for k:=range m{
		up:=strings.ToUpper(k);ok:=false
		for _,t:=range tokens{if strings.Contains(up,strings.ToUpper(t)){ok=true;break}}
		if !ok{continue}
		if v:=numberMapValue(m,k);v!=0{cs=append(cs,candidate{k,v})}
	}
	sort.SliceStable(cs,func(i,j int)bool{return len(cs[i].k)<len(cs[j].k)})
	if len(cs)>0{return cs[0].v}
	return 0
}

func findStringFieldByTokens(m map[string]any,tokens []string)string{
	keys:=make([]string,0,len(m));for k:=range m{keys=append(keys,k)};sort.Strings(keys)
	for _,k:=range keys{
		up:=strings.ToUpper(k);ok:=true
		for _,t:=range tokens{if !strings.Contains(up,strings.ToUpper(t)){ok=false;break}}
		if !ok{continue}
		if v:=firstStringMapValue(m,k);v!=""{return v}
	}
	return ""
}

func brazilUFName(uf string)string{
	return map[string]string{
		"AC":"ACRE","AL":"ALAGOAS","AP":"AMAPÁ","AM":"AMAZONAS","BA":"BAHIA","CE":"CEARÁ","DF":"DISTRITO FEDERAL",
		"ES":"ESPÍRITO SANTO","GO":"GOIÁS","MA":"MARANHÃO","MT":"MATO GROSSO","MS":"MATO GROSSO DO SUL",
		"MG":"MINAS GERAIS","PA":"PARÁ","PB":"PARAÍBA","PR":"PARANÁ","PE":"PERNAMBUCO","PI":"PIAUÍ",
		"RJ":"RIO DE JANEIRO","RN":"RIO GRANDE DO NORTE","RS":"RIO GRANDE DO SUL","RO":"RONDÔNIA",
		"RR":"RORAIMA","SC":"SANTA CATARINA","SP":"SÃO PAULO","SE":"SERGIPE","TO":"TOCANTINS",
	}[strings.ToUpper(strings.TrimSpace(uf))]
}
