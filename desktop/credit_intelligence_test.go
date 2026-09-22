package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeCreditFixture170(t *testing.T, name, header, row string) string {
	t.Helper()
	return writeGzipFixture150(t, name, header+"\n"+row+"\n")
}

func TestCreditReleaseAndDisbursementScanners170(t *testing.T) {
	key := sicorOperationKey("517256416","1")
	ops := map[string]*CreditOperationIntelligence{key:{RefBacen:"517256416",Order:"1",CreditValue:300000}}
	release := writeCreditFixture170(t,"release.gz","#REF_BACEN;NU_ORDEM;VL_LIBERADO;DT_LIBERACAO","517256416;1;125000,50;2026-09-01")
	if err:=scanCreditReleases(release,ops);err!=nil{t.Fatal(err)}
	if ops[key].TotalReleased!=125000.50 || len(ops[key].Releases)!=1{t.Fatalf("liberação inesperada: %+v",ops[key])}
	disb := writeCreditFixture170(t,"disb.gz","#REF_BACEN;NU_ORDEM;DT_PREV_PAGAMENTO;VALOR_PARCELA","517256416;1;2026-10-01;300000")
	if err:=scanCreditDisbursements(disb,ops);err!=nil{t.Fatal(err)}
	if ops[key].PlannedDisbursement!=300000{t.Fatalf("cronograma inesperado: %+v",ops[key])}
}

func TestCreditDeclassificationScanner170(t *testing.T) {
	key:=sicorOperationKey("10","2")
	ops:=map[string]*CreditOperationIntelligence{key:{RefBacen:"10",Order:"2"}}
	path:=writeCreditFixture170(t,"desc.gz","#REF_BACEN;NU_ORDEM;DT_DESC;CD_MOTIVO_DESC;VL_DESC;TIPO_DESC","10;2;2026-08-11;7;25000;P")
	d:=creditDomains{DeclassReason:map[string]string{"7":"Motivo teste"}}
	if err:=scanCreditDeclassification(path,ops,d);err!=nil{t.Fatal(err)}
	got:=ops[key].Declassification
	if !got.Found||got.Value!=25000||got.Reason!="Motivo teste"{t.Fatalf("desclassificação inesperada: %+v",got)}
}

func TestCreditOperationDetails170(t *testing.T) {
	key:=sicorOperationKey("99","1")
	ops:=map[string]*CreditOperationIntelligence{key:{RefBacen:"99",Order:"1"}}
	header:="#REF_BACEN;NU_ORDEM;VL_REC_PROPRIO;VL_AREA_INFORMADA;VL_JUROS;VL_JUROS_ENC_FINAN_POSFIX;VL_PERC_CUSTO_EFET_TOTAL;VL_PRESTACAO_INVESTIMENTO;VL_PREV_PROD;VL_QUANTIDADE;VL_RECEITA_BRUTA_ESPERADA;VL_PRODUTIV_OBTIDA;VL_ALIQ_PROAGRO;CD_TIPO_SEGURO;CD_INST_CREDITO;CD_CATEG_EMITENTE;CD_TIPO_IRRIGACAO;CD_TIPO_AGRICULTURA;CD_TIPO_CULTIVO;CD_TIPO_INTGR_CONSOR;CD_TIPO_GRAO_SEMENTE;CD_FASE_CICLO_PRODUCAO;CD_CICLO_CULTIVAR;CD_TIPO_SOLO;DT_INIC_PLANTIO;DT_FIM_PLANTIO"
	row:="99;1;20000;12,5;8,5;1,2;9,1;50000;100;90;450000;80;2,0;1;2;3;4;5;6;7;8;9;10;11;2026-10-01;2026-11-10"
	path:=writeCreditFixture170(t,"op.gz",header,row)
	d:=creditDomains{
		Insurance:map[string]string{"1":"Proagro"},
		Instrument:map[string]string{"2":"Cédula"},
		IssuerCategory:map[string]string{"3":"Produtor"},
		Irrigation:map[string]string{"4":"Irrigado"},
		Agriculture:map[string]string{"5":"Convencional"},
		CropType:map[string]string{"6":"Temporário"},
		Integration:map[string]string{"7":"Consórcio"},
		SeedType:map[string]string{"8":"Semente"},
		ProductionPhase:map[string]string{"9":"Formação"},
		CultivarCycle:map[string]string{"10":"Médio"},
		Soil:map[string]string{"11":"Tipo 3"},
	}
	if err:=scanCreditOperationDetails(path,ops,d);err!=nil{t.Fatal(err)}
	got:=ops[key]
	if got.OwnResources!=20000||got.InterestRatePct!=8.5||got.ExpectedRevenue!=450000||got.Irrigation!="Irrigado"||got.Soil!="Tipo 3"{t.Fatalf("detalhes inesperados: %+v",got)}
}

func TestCreditProagroScanners170(t *testing.T) {
	key:=sicorOperationKey("7","1")
	ops:=map[string]*CreditOperationIntelligence{key:{RefBacen:"7",Order:"1"}}
	d:=creditDomains{COPStatus:map[string]string{"1":"Ativa"},Event:map[string]string{"3":"Seca"},CultivarCycle:map[string]string{"2":"Médio"},Soil:map[string]string{"3":"Tipo 3"}}
	cop:=writeCreditFixture170(t,"cop.gz","#REF_BACEN;NU_ORDEM;DT_COMUNICACAO;CD_STATUS;CD_EVENTO;CD_CICLO_CULTIVAR;CD_TIPO_SOLO;DT_INICIO_PLANTIO;DT_FIM_PLANTIO","7;1;2026-07-01;1;3;2;3;2026-01-10;2026-02-20")
	if err:=scanCreditCOP(cop,ops,d);err!=nil{t.Fatal(err)}
	parc:=writeCreditFixture170(t,"parc.gz","#REF_BACEN;NU_ORDEM;VL_PAGO","7;1;12500")
	if err:=scanCreditProagroParcels(parc,ops);err!=nil{t.Fatal(err)}
	if !ops[key].Proagro.HasCOP||ops[key].Proagro.Event!="Seca"||ops[key].Proagro.PaidValue!=12500{t.Fatalf("Proagro inesperado: %+v",ops[key].Proagro)}
}

func TestCreditReleaseRatio170(t *testing.T){
	op:=CreditOperationIntelligence{CreditValue:200000,TotalReleased:150000}
	if got:=creditReleaseRatio(op);got!=75{t.Fatalf("esperava 75, obteve %f",got)}
}

func TestCreditIntelligenceCacheClear170(t *testing.T){
	a:=&App{dataDir:t.TempDir()}
	p:=filepath.Join(a.dataDir,"cache","sicor_credit_intelligence")
	if err:=os.MkdirAll(p,0o755);err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(filepath.Join(p,"x"),[]byte("x"),0o644);err!=nil{t.Fatal(err)}
	if err:=a.ClearCreditIntelligenceCache();err!=nil{t.Fatal(err)}
	if _,err:=os.Stat(p);!os.IsNotExist(err){t.Fatalf("cache deveria ter sido removido: %v",err)}
}


func TestZARCSeasonAndCodes170(t *testing.T) {
	d, err := time.Parse("2006-01-02","2026-10-15")
	if err != nil { t.Fatal(err) }
	a,b := zarcCropSeason(d)
	if a!=2026 || b!=2027 { t.Fatalf("safra inesperada: %d/%d",a,b) }
	if zarcDecendio(d)!=29 { t.Fatalf("decêndio inesperado: %d",zarcDecendio(d)) }
	if zarcCycleCode("Grupo II","")!="21" { t.Fatal("Grupo II deveria mapear para 21") }
	if zarcSoilCode("AD3","")!="13" { t.Fatal("AD3 deveria mapear para 13") }
}

func TestScanZARCForOperation170(t *testing.T) {
	path:=filepath.Join(t.TempDir(),"zarc.csv")
	header:="Nome_cultura,SafraIni,SafraFin,Cod_Cultura,Cod_Ciclo,Cod_Solo,geocodigo,UF,municipio,Cod_Outros_Manejos,Nome_Outros_Manejos,Cod_Clima,Nome_Clima,Cod_Munic,Portaria"
	for i:=1;i<=36;i++ { header += fmt.Sprintf(",dec%d",i) }
	vals:=make([]string,36)
	vals[28]="20"
	vals[29]="30"
	row:="Soja,2026,2027,1,21,13,5209101,GO,Goiatuba,1,Sequeiro,0,Nao se aplica,1234,Portaria teste"
	for _,v:=range vals { row += ","+v }
	if err:=os.WriteFile(path,[]byte(header+"\n"+row+"\n"),0o644);err!=nil{t.Fatal(err)}
	car:=CARResult{Municipality:"Goiatuba",UF:"GO",MunicipalityCode:"5209101"}
	op:=CreditOperationIntelligence{
		Product:"Soja em grao",CultivarCycle:"Grupo II",Soil:"AD3",Agriculture:"Sequeiro",
		PlantingStart:"2026-10-15",PlantingEnd:"2026-10-25",
	}
	got,err:=scanZARCForOperation(path,zarcResource{Name:"Tábua Safra 2026/2027",URL:"https://example.invalid/zarc.csv"},car,op,2026,2027)
	if err!=nil{t.Fatal(err)}
	if !got.Matched || got.Status!="Indicação localizada" { t.Fatalf("ZARC inesperado: %+v",got) }
	if len(got.Periods)!=2 || got.Periods[0].RiskPct!=20 || got.Periods[1].RiskPct!=30 {
		t.Fatalf("decêndios inesperados: %+v",got.Periods)
	}
}

func TestScanZARCDoesNotClaimOutsideWhenNoExactCombination170(t *testing.T) {
	path:=filepath.Join(t.TempDir(),"zarc.csv")
	header:="Nome_cultura,Cod_Ciclo,Cod_Solo,geocodigo,UF,municipio,Cod_Outros_Manejos,Nome_Outros_Manejos,Portaria,dec29"
	row:="Milho,21,13,5209101,GO,Goiatuba,1,Sequeiro,Portaria teste,20"
	if err:=os.WriteFile(path,[]byte(header+"\n"+row+"\n"),0o644);err!=nil{t.Fatal(err)}
	car:=CARResult{Municipality:"Goiatuba",UF:"GO",MunicipalityCode:"5209101"}
	op:=CreditOperationIntelligence{Product:"Soja",CultivarCycle:"Grupo II",Soil:"AD3",PlantingStart:"2026-10-15"}
	got,err:=scanZARCForOperation(path,zarcResource{URL:"https://example.invalid"},car,op,2026,2027)
	if err!=nil{t.Fatal(err)}
	if got.Matched || got.Status!="Sem combinação exata" { t.Fatalf("não deveria declarar fora do ZARC: %+v",got) }
}


func TestMarketAggregation170(t *testing.T) {
	rows:=[]map[string]any{
		{"AnoEmissao":"2026","Produto":"Soja","QtdCusteio":3.0,"VlCusteio":150000.0},
		{"AnoEmissao":"2026","Produto":"Soja","QtdCusteio":2.0,"VlCusteio":100000.0},
		{"AnoEmissao":"2026","Produto":"Milho","QtdCusteio":1.0,"VlCusteio":50000.0},
	}
	got:=aggregateMarketRows(rows,"Município","Custeio","2026","product")
	if len(got)!=2 { t.Fatalf("esperava 2 produtos, obteve %d: %#v",len(got),got) }
	var soja *CreditMarketItem
	for i:=range got { if got[i].Label=="Soja" { soja=&got[i] } }
	if soja==nil || soja.Contracts!=5 || soja.Value!=250000 { t.Fatalf("agregação de soja inesperada: %#v",soja) }
}

func TestMarketMunicipalityFilterRejectsOtherCity170(t *testing.T) {
	rows:=[]map[string]any{
		{"AnoEmissao":"2026","Municipio":"GOIATUBA","nomeUF":"GO","codMunicIbge":"5209101","Produto":"Soja"},
		{"AnoEmissao":"2026","Municipio":"GOIANIA","nomeUF":"GO","codMunicIbge":"5208707","Produto":"Milho"},
	}
	got:=filterMarketMunicipalityRows(rows,"Goiatuba","GO","5209101","2026")
	if len(got)!=1 || firstNonEmptyStringMapValue(got[0],"Produto")!="Soja" { t.Fatalf("filtro municipal inesperado: %#v",got) }
}

func TestMarketTopItemsPrefersNewestAndLargest170(t *testing.T) {
	items:=[]CreditMarketItem{
		{Year:"2025",Label:"A",Value:900},
		{Year:"2026",Label:"B",Value:100},
		{Year:"2026",Label:"C",Value:500},
	}
	got:=topMarketItems(items,2)
	if len(got)!=2 || got[0].Label!="C" || got[1].Label!="B" { t.Fatalf("ordenação inesperada: %#v",got) }
}
