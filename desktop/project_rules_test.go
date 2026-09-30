package main

import (
	"strings"
	"testing"
)

func findPreparationCheck(checks []ProjectPreparationCheck, key string) (ProjectPreparationCheck, bool) {
	for _, c := range checks {
		if c.Key == key {
			return c, true
		}
	}
	return ProjectPreparationCheck{}, false
}

func TestProjectSmartProfileCoffeeCusteio221(t *testing.T) {
	p := RuralProject{Name:"Custeio de Café 2026/27", Activity:"Café", OperationType:"custeio"}
	profile := buildProjectSmartProfile(p)
	if !profile.Coffee || !profile.Agriculture || profile.Ambiguous {
		t.Fatalf("perfil de café incorreto: %#v", profile)
	}
	if profile.ActivityClass != "coffee" {
		t.Fatalf("classe esperada coffee, obtida %q", profile.ActivityClass)
	}
	foundZARC := false
	for _, r := range profile.Rules {
		if strings.Contains(r, "ZARC") { foundZARC = true }
	}
	if !foundZARC { t.Fatal("custeio de café deve ativar regra ZARC") }
}

func TestProjectSmartRulesLivestockAcquisition221(t *testing.T) {
	app := newProjectTestApp(t)
	client, err := app.SaveClient(Client{Name:"Produtor Pecuária"})
	if err != nil { t.Fatal(err) }
	property, err := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda Pecuária", UF:"MG"})
	if err != nil { t.Fatal(err) }
	center, err := app.GetPropertyDocumentCenter(property.ID)
	if err != nil { t.Fatal(err) }

	p := RuralProject{Name:"Aquisição de bovinos", PropertyID:property.ID, Activity:"Pecuária bovina", OperationType:"aquisicao"}
	profile, checks := app.projectSmartChecks(p, &center)
	if !profile.Livestock { t.Fatal("atividade pecuária não detectada") }

	gta, ok := findPreparationCheck(checks, "doc_gta")
	if !ok { t.Fatal("regra GTA ausente") }
	if gta.Status != "pending" {
		t.Fatalf("GTA deveria ficar pendente sem documento na aquisição pecuária, obtido %q", gta.Status)
	}
	budget, ok := findPreparationCheck(checks, "doc_budget")
	if !ok || budget.Status != "pending" {
		t.Fatalf("orçamento deveria ser exigido em aquisição: %#v", budget)
	}
}

func TestProjectSmartRulesIrrigationInvestment221(t *testing.T) {
	app := newProjectTestApp(t)
	client, err := app.SaveClient(Client{Name:"Produtor Irrigação"})
	if err != nil { t.Fatal(err) }
	property, err := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda Irrigada", UF:"MG"})
	if err != nil { t.Fatal(err) }
	center, err := app.GetPropertyDocumentCenter(property.ID)
	if err != nil { t.Fatal(err) }

	p := RuralProject{Name:"Irrigação por pivô", PropertyID:property.ID, Activity:"Irrigação", OperationType:"investimento"}
	profile, checks := app.projectSmartChecks(p, &center)
	if !profile.Irrigation { t.Fatal("atividade de irrigação não detectada") }

	outorga, ok := findPreparationCheck(checks, "doc_outorga")
	if !ok || outorga.Status != "pending" {
		t.Fatalf("outorga deveria ser exigida para irrigação: %#v", outorga)
	}
	terrain, ok := findPreparationCheck(checks, "terrain")
	if !ok || terrain.Status != "review" {
		t.Fatalf("relevo deveria ficar para conferir sem gleba: %#v", terrain)
	}
}

func TestProjectSmartRulesUnknownActivity221(t *testing.T) {
	p := RuralProject{Name:"Projeto genérico", Activity:"Outra atividade", OperationType:"outro"}
	profile := buildProjectSmartProfile(p)
	if !profile.Ambiguous || profile.ActivityClass != "unknown" {
		t.Fatalf("atividade desconhecida deveria exigir conferência: %#v", profile)
	}
}

func TestPrepareRuralProjectReturnsSmartProfile221(t *testing.T) {
	app := newProjectTestApp(t)
	client, err := app.SaveClient(Client{Name:"Produtor Café"})
	if err != nil { t.Fatal(err) }
	property, err := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda Café", UF:"MG"})
	if err != nil { t.Fatal(err) }
	project, err := app.SaveRuralProject(RuralProject{
		PropertyID:property.ID, Name:"Custeio Café", Bank:"Banco", OperationType:"custeio",
		Activity:"Café", RequestedAmount:50000, AreaHa:5, Status:"draft",
	})
	if err != nil { t.Fatal(err) }

	result, err := app.PrepareRuralProject(project.ID)
	if err != nil { t.Fatal(err) }
	if !result.SmartProfile.Coffee {
		t.Fatalf("perfil inteligente não retornado: %#v", result.SmartProfile)
	}
	if _, ok := findPreparationCheck(result.Checks, "zarc_new_project"); !ok {
		t.Fatal("preparação deveria retornar a checagem ZARC aplicável ao custeio agrícola")
	}
	if result.ReadinessPct < 0 || result.ReadinessPct > 100 {
		t.Fatalf("prontidão inválida: %d", result.ReadinessPct)
	}
}
