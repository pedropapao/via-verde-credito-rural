package main

import "testing"

func techFieldByKey(t *testing.T, r AutoTechnicalReview, key string) AutoTechnicalField {
	t.Helper()
	for _, g := range r.Groups {
		for _, f := range g.Fields {
			if f.Key == key { return f }
		}
	}
	t.Fatalf("campo técnico %q não encontrado", key)
	return AutoTechnicalField{}
}

func TestAgroTechnicalReviewDoesNotInventDefaults(t *testing.T) {
	r := autoBuildAgroTechnicalReview(nil, nil)
	for _, key := range []string{"interest", "term", "grace", "matrizes", "natality", "pasture_support", "labor_monthly"} {
		f := techFieldByKey(t, r, key)
		if f.Value != "" {
			t.Fatalf("%s recebeu valor sem documento: %q", key, f.Value)
		}
		if f.Status != "missing" {
			t.Fatalf("%s deveria estar missing, veio %s", key, f.Status)
		}
	}
	if r.CriticalMissing == 0 {
		t.Fatal("esperava bloqueio por entradas críticas ausentes")
	}
}

func TestAgroTechnicalReviewReadsFabioStyleHerdTable(t *testing.T) {
	docs := []autoDocument{{Name:"Projeto_Fabio.pdf", Text:`
PRAZO: 8 anos
ANEXO 06 - Composição, Evolução e Dinâmica do Rebanho Bovino:
CATEGORIA UA INÍCIO FIM
Matrizes 1,00 74 74 74 74
Novilhas 2/3 anos 0,75 0 0 48 36
Novilhas 1/2 anos 0,50 48 24 15 8
Bezerras 0,25 15 4 0 0
Bezerros 0,25 0 0 0 0
Novilhos 1/2 anos 0,50 0 0 0 0
Novilhos 2/3 anos 0,75 81 61 0 0
Novilhos + 3 anos 1,00 0 0 0 0
Touros 1,50 6 9 6 9
- Natalidade (%) 80 80 83
- Mortalidade adultos (%) 1 1 1
- Mortalidade 1/2 anos (%) 3 3 3
- Mortalidade bezerros (%) 5 5 5
- Desc. matrizes (%) 0 10 10
- Desc. touros (%) 0 0 0
Suporte pastagens (UA) 180 180 200 200
`}}
	master := []AutoMasterField{
		{Key:"area", Value:"9,96", Status:"ok", Sources:[]string{"Projeto_Fabio.pdf"}},
		{Key:"line", Value:"PRONAMP INVESTIMENTO", Status:"ok", Sources:[]string{"Projeto_Fabio.pdf"}},
		{Key:"activity", Value:"Irrigação em pastagem", Status:"ok", Sources:[]string{"Projeto_Fabio.pdf"}},
	}
	r := autoBuildAgroTechnicalReview(docs, master)
	checks := map[string]string{
		"term":"96", "matrizes":"74", "novilhas_23":"0", "novilhas_12":"48", "bezerras":"15",
		"novilhos_23":"81", "touros":"6", "natality":"80", "mortality_adult":"1",
		"mortality_young":"3", "mortality_calf":"5", "cull_matrices":"0", "pasture_support":"180",
	}
	for key, want := range checks {
		got := techFieldByKey(t, r, key)
		if got.Value != want {
			t.Fatalf("%s=%q, esperava %q", key, got.Value, want)
		}
		if got.Status != "compatible" {
			t.Fatalf("%s deveria exigir confirmação, veio %s", key, got.Status)
		}
	}
	if got := techFieldByKey(t, r, "year1_finished_sale"); got.Value != "" || got.Status != "missing" {
		t.Fatalf("venda não pode ser inferida do estoque: %+v", got)
	}
}

func TestAgroTechnicalReviewReadsExplicitSaleAndCosts(t *testing.T) {
	docs := []autoDocument{{Name:"Projeto_Fabio.pdf", Text:`
Venda planejada de novilhos +3: 81 cabeças no primeiro ano.
2.1.1 Assist. veterinaria visitas R$ 800,00 2 R$ 1.600,00
2.1.2 Assist. agronomica visitas R$ 1.200,00 2 R$ 2.400,00
Peoes mes R$ 2.000,00 12 R$ 24.000,00
Energia eletrica mes R$ 1.617,25 12 R$ 19.407,00
Currais unidade R$ 2.000,00 1 R$ 2.000,00
Manutencao sistema de irrigacao ano R$ 5.010,00 0 R$ 0,00
`}}
	r := autoBuildAgroTechnicalReview(docs, nil)
	checks := map[string]string{
		"year1_finished_sale":"81", "vet_cost":"800", "agronomy_cost":"1200", "labor_monthly":"2000",
		"energy_monthly":"1617,25", "corral_annual":"2000", "irrigation_annual":"5010",
	}
	for key, want := range checks {
		got := techFieldByKey(t, r, key)
		if got.Value != want { t.Fatalf("%s=%q, esperava %q", key, got.Value, want) }
	}
}
