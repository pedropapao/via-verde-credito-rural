package main

import (
	"encoding/json"
	"math"
	"testing"
)

func TestWorldCoverLabels190(t *testing.T) {
	cases := map[int]string{
		10:"Cobertura arbórea",
		30:"Vegetação herbácea (inclui pastagens)",
		40:"Cultivos anuais",
		80:"Água permanente",
	}
	for code,want:=range cases {
		got,ok:=worldCoverLabel(code)
		if !ok || got!=want {
			t.Fatalf("classe %d: got=%q ok=%v want=%q",code,got,ok,want)
		}
	}
	if _,ok:=worldCoverLabel(999);ok {
		t.Fatal("classe inexistente não deveria ser aceita")
	}
}

func TestWorldCoverClassesFromCounts190(t *testing.T) {
	got:=worldCoverClassesFromCounts(map[int]int{10:60,40:30,80:10},100,200)
	if len(got)!=3 {t.Fatalf("esperava 3 classes, obteve %d",len(got))}
	if got[0].Code!=10 || math.Abs(got[0].Percent-60)>0.001 || math.Abs(got[0].AreaHa-120)>0.001 {
		t.Fatalf("classe dominante inesperada: %+v",got[0])
	}
	sumPct,sumHa:=0.0,0.0
	for _,x:=range got {sumPct+=x.Percent;sumHa+=x.AreaHa}
	if math.Abs(sumPct-100)>0.001 || math.Abs(sumHa-200)>0.001 {
		t.Fatalf("totais inconsistentes: pct=%f ha=%f",sumPct,sumHa)
	}
}

func TestWorldCoverSamplePointsInsideCAR190(t *testing.T) {
	feature:=carGeoFeature{
		Type:"Feature",
		Properties:map[string]any{},
		Geometry:carGeoJSONGeometry{
			Type:"Polygon",
			Coordinates:json.RawMessage(`[[[-47.01,-20.93],[-46.99,-20.93],[-46.99,-20.91],[-47.01,-20.91],[-47.01,-20.93]]]`),
		},
	}
	raw,_:=json.Marshal(feature)
	points,err:=worldCoverSamplePoints(string(raw),300)
	if err!=nil {t.Fatal(err)}
	if len(points)<50 || len(points)>300 {
		t.Fatalf("quantidade de pontos inesperada: %d",len(points))
	}
	for _,p:=range points {
		if p[0] <= -47.01 || p[0] >= -46.99 || p[1] <= -20.93 || p[1] >= -20.91 {
			t.Fatalf("ponto fora do CAR: %+v",p)
		}
	}
}

func TestBiomeFromMapBiomasTerritories190(t *testing.T) {
	items:=[]struct {
		Name string `json:"name"`
		CategoryName string `json:"categoryName"`
		Code string `json:"code"`
		AreaHa float64 `json:"areaHa"`
	}{
		{Name:"Minas Gerais",CategoryName:"Estado"},
		{Name:"Mata Atlântica",CategoryName:"Bioma"},
	}
	if got:=biomeFromMapBiomasTerritories(items);got!="Mata Atlântica" {
		t.Fatalf("bioma inesperado: %q",got)
	}
}

func TestEnvironmentalProfileHasData190(t *testing.T) {
	if environmentalProfileHasData(EnvironmentalProfile{}) {
		t.Fatal("perfil vazio não deveria ser considerado preenchido")
	}
	if environmentalProfileHasData(EnvironmentalProfile{BiomeAvailable:true,Biome:"Cerrado"}) {
		t.Fatal("cache antigo sem estado dos focos não deve ser reutilizado")
	}
	if environmentalProfileHasData(EnvironmentalProfile{LandCoverAvailable:true,DominantLandCover:"Cultivos anuais"}) {
		t.Fatal("cache antigo do WorldCover sem focos não deve ser reutilizado")
	}
	if environmentalProfileHasData(EnvironmentalProfile{
		BiomeAvailable:true,
		Biome:"Cerrado",
		Fire:EnvironmentalFireProfile{Status:fireStatusNone,Checked:true,Available:true},
	}) {
		t.Fatal("cache anterior à hidrografia não deve ser reutilizado")
	}
	if !environmentalProfileHasData(EnvironmentalProfile{
		BiomeAvailable:true,
		Biome:"Cerrado",
		Fire:EnvironmentalFireProfile{Status:fireStatusNone,Checked:true,Available:true},
		Hydrology:EnvironmentalHydrologyProfile{Status:hydrologyStatusNone,Checked:true,Available:true},
	}) {
		t.Fatal("perfil atual com focos e hidrografia concluídos deveria ser válido")
	}
	if !environmentalProfileHasData(EnvironmentalProfile{
		LandCoverAvailable:true,
		DominantLandCover:"Cultivos anuais",
		Fire:EnvironmentalFireProfile{Status:fireStatusUnavailable},
		Hydrology:EnvironmentalHydrologyProfile{Status:hydrologyStatusUnavailable,Checked:true},
	}) {
		t.Fatal("falhas registradas ainda devem identificar cache da versão atual")
	}
}
