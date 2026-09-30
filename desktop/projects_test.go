package main

import (
	"path/filepath"
	"testing"
)

func newProjectTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	db, err := openSQLiteDatabase(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	app := &App{db: db, dataDir: dir}
	if err := app.migrate(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func(){ _ = db.Close() })
	return app
}

func TestRuralProjectCRUD220(t *testing.T) {
	app := newProjectTestApp(t)
	client, err := app.SaveClient(Client{Name:"Produtor Teste"})
	if err != nil { t.Fatal(err) }
	property, err := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda Teste", Municipality:"Teste", UF:"MG", DeclaredAreaHa:100})
	if err != nil { t.Fatal(err) }

	project, err := app.SaveRuralProject(RuralProject{
		PropertyID:property.ID, Name:"Custeio Café", Bank:"Banco Teste", CreditLine:"Custeio",
		OperationType:"custeio", Activity:"Café", RequestedAmount:150000, TermMonths:12,
		InterestRatePct:8.5, AreaHa:20, Status:"draft",
	})
	if err != nil { t.Fatal(err) }
	if project.ID <= 0 { t.Fatal("projeto não recebeu ID") }

	list, err := app.ListRuralProjects(property.ID)
	if err != nil { t.Fatal(err) }
	if len(list) != 1 || list[0].Name != "Custeio Café" { t.Fatalf("lista inesperada: %#v", list) }

	project.Status = "review"
	project.RequestedAmount = 175000
	project, err = app.SaveRuralProject(project)
	if err != nil { t.Fatal(err) }
	if project.Status != "review" || project.RequestedAmount != 175000 { t.Fatal("edição do projeto não foi preservada") }

	prep, err := app.PrepareRuralProject(project.ID)
	if err != nil { t.Fatal(err) }
	if len(prep.Checks) == 0 { t.Fatal("preparação automática não retornou verificações") }
	if prep.Scope == "" { t.Fatal("preparação precisa declarar seu escopo") }

	if err := app.DeleteRuralProject(project.ID); err != nil { t.Fatal(err) }
	if _, err := app.GetProperty(property.ID); err != nil { t.Fatalf("excluir projeto não pode apagar imóvel: %v", err) }
	list, err = app.ListRuralProjects(property.ID)
	if err != nil { t.Fatal(err) }
	if len(list) != 0 { t.Fatal("projeto deveria ter sido excluído") }
}

func TestRuralProjectRejectsAreaFromAnotherProperty220(t *testing.T) {
	app := newProjectTestApp(t)
	client, err := app.SaveClient(Client{Name:"Produtor"})
	if err != nil { t.Fatal(err) }
	p1, err := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda A", UF:"MG"})
	if err != nil { t.Fatal(err) }
	p2, err := app.SaveProperty(Property{ClientID:client.ID, Name:"Fazenda B", UF:"MG"})
	if err != nil { t.Fatal(err) }

	res, err := app.db.Exec("INSERT INTO project_areas(property_id,name,purpose,area_ha,perimeter_m,center_lat,center_lon,inside_car_pct,geojson,kml_path,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,datetime('now'),datetime('now'))",
		p2.ID, "Gleba B", "", 10, 0, 0, 0, 100, "{}", "")
	if err != nil { t.Fatal(err) }
	areaID, _ := res.LastInsertId()

	_, err = app.SaveRuralProject(RuralProject{PropertyID:p1.ID, Name:"Projeto inválido", AreaID:areaID})
	if err == nil { t.Fatal("projeto não pode usar gleba de outro imóvel") }
}

func TestSchemaVersion220(t *testing.T) {
	app := newProjectTestApp(t)
	var version int
	if err := app.db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version); err != nil { t.Fatal(err) }
	if version != 9 { t.Fatalf("schema esperado 9, obtido %d", version) }
}
