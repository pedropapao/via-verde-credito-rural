package main

import (
	"os"
	"path/filepath"
	"testing"
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
