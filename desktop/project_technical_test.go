package main

import "testing"

func TestProjectTechnicalDataCRUD222(t *testing.T) {
	app := newProjectTestApp(t)
	client, err := app.SaveClient(Client{Name:"Produtor Técnico"})
	if err != nil { t.Fatal(err) }
	property, err := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda Técnica", UF:"MG"})
	if err != nil { t.Fatal(err) }
	project, err := app.SaveRuralProject(RuralProject{
		PropertyID:property.ID, Name:"Investimento Café", Bank:"Banco",
		OperationType:"investimento", Activity:"Café", RequestedAmount:50000, AreaHa:10, Status:"draft",
	})
	if err != nil { t.Fatal(err) }

	saved, err := app.SaveRuralProjectTechnicalData(ProjectTechnicalData{
		ProjectID:project.ID,
		Culture:"Café",
		BenefitedAreaHa:9.5,
		Items:[]ProjectTechnicalItem{
			{Description:"Secador", Quantity:1, Unit:"un", UnitValue:30000, Supplier:"Fornecedor A"},
			{Description:"Terreiro", Quantity:1, Unit:"un", UnitValue:20000, Supplier:"Fornecedor B"},
		},
		TechnicalPurpose:"Melhoria da pós-colheita",
	})
	if err != nil { t.Fatal(err) }
	if len(saved.Items) != 2 { t.Fatalf("itens esperados=2, obtido=%d", len(saved.Items)) }

	got, err := app.GetRuralProjectTechnicalData(project.ID)
	if err != nil { t.Fatal(err) }
	if got.Culture != "Café" || got.BenefitedAreaHa != 9.5 || len(got.Items) != 2 {
		t.Fatalf("dados técnicos não preservados: %#v", got)
	}
	if total := projectTechnicalBudgetTotal(got); total != 50000 {
		t.Fatalf("total técnico esperado=50000, obtido=%v", total)
	}

	if err := app.DeleteRuralProject(project.ID); err != nil { t.Fatal(err) }
	var count int
	if err := app.db.QueryRow("SELECT COUNT(*) FROM rural_project_technical_data WHERE project_id=?", project.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 { t.Fatal("dados técnicos deveriam ser removidos junto com o projeto") }
}

func TestProjectTechnicalChecksCoffeeInvestment222(t *testing.T) {
	p := RuralProject{
		Name:"Investimento Café", Activity:"Café", OperationType:"investimento",
		RequestedAmount:50000, AreaHa:10,
	}
	profile := buildProjectSmartProfile(p)
	data := ProjectTechnicalData{
		Culture:"Café", BenefitedAreaHa:9.5,
		Items:[]ProjectTechnicalItem{
			{Description:"Secador", Quantity:1, Unit:"un", UnitValue:30000},
			{Description:"Terreiro", Quantity:1, Unit:"un", UnitValue:20000},
		},
	}
	checks := projectTechnicalChecks(p, profile, data)
	agri, ok := findPreparationCheck(checks, "technical_agriculture")
	if !ok || agri.Status != "ready" {
		t.Fatalf("dados agrícolas deveriam estar prontos: %#v", agri)
	}
	items, ok := findPreparationCheck(checks, "technical_items")
	if !ok || items.Status != "ready" {
		t.Fatalf("itens do investimento deveriam estar prontos: %#v", items)
	}
}

func TestProjectTechnicalChecksDetectAreaAndValueMismatch222(t *testing.T) {
	p := RuralProject{
		Name:"Investimento Café", Activity:"Café", OperationType:"investimento",
		RequestedAmount:50000, AreaHa:9.61,
	}
	profile := buildProjectSmartProfile(p)
	data := ProjectTechnicalData{
		Culture:"Café", BenefitedAreaHa:10,
		Items:[]ProjectTechnicalItem{{Description:"Equipamento", Quantity:1, Unit:"un", UnitValue:45000}},
	}
	checks := projectTechnicalChecks(p, profile, data)
	agri, _ := findPreparationCheck(checks, "technical_agriculture")
	if agri.Status != "pending" {
		t.Fatalf("área beneficiada maior que a área do projeto deveria ser pendência: %#v", agri)
	}
	items, _ := findPreparationCheck(checks, "technical_items")
	if items.Status != "review" {
		t.Fatalf("composição divergente do valor solicitado deveria exigir conferência: %#v", items)
	}
}

func TestReadinessDoesNotDoubleCountDocumentSummary222(t *testing.T) {
	app := newProjectTestApp(t)
	client, err := app.SaveClient(Client{Name:"Produtor"})
	if err != nil { t.Fatal(err) }
	property, err := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda", UF:"MG"})
	if err != nil { t.Fatal(err) }
	project, err := app.SaveRuralProject(RuralProject{
		PropertyID:property.ID, Name:"Investimento Café", Bank:"Banco",
		OperationType:"investimento", Activity:"Café", RequestedAmount:50000, AreaHa:5, Status:"draft",
	})
	if err != nil { t.Fatal(err) }

	result, err := app.PrepareRuralProject(project.ID)
	if err != nil { t.Fatal(err) }

	docSummary, ok := findPreparationCheck(result.Checks, "documents")
	if !ok { t.Fatal("resumo documental ausente") }
	if !docSummary.Informational {
		t.Fatal("resumo geral de documentos deve ser informativo para não contar duas vezes")
	}

	countedReady, countedPending, countedReview := 0, 0, 0
	for _, check := range result.Checks {
		if check.Informational { continue }
		switch check.Status {
		case "ready": countedReady++
		case "pending": countedPending++
		case "review": countedReview++
		}
	}
	if result.Ready != countedReady || result.Pending != countedPending || result.Review != countedReview {
		t.Fatalf("contadores incluem item informativo: result=%d/%d/%d contado=%d/%d/%d",
			result.Ready,result.Pending,result.Review,countedReady,countedPending,countedReview)
	}
}

func TestProjectTechnicalRejectsNegativeValues222(t *testing.T) {
	app := newProjectTestApp(t)
	client, _ := app.SaveClient(Client{Name:"Produtor"})
	property, _ := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda", UF:"MG"})
	project, _ := app.SaveRuralProject(RuralProject{PropertyID:property.ID, Name:"Projeto", Status:"draft"})

	_, err := app.SaveRuralProjectTechnicalData(ProjectTechnicalData{ProjectID:project.ID, AnimalCount:-1})
	if err == nil { t.Fatal("valor técnico negativo deveria ser rejeitado") }
}
