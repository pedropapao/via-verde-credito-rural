package main

import (
	"bytes"
	"testing"
)

func TestTemplatesDoPainelDeVisualizacaoCompilamEExecutam(t *testing.T) {
	a := NewApp(Config{}, nil)
	if len(a.templates) < 10 {
		t.Fatalf("templates carregados insuficientes: %d", len(a.templates))
	}
	u := &User{Name:"Pedro", Role:"owner"}
	cases := []struct{name string; data any}{
		{"dashboard", ViewerDashboard{}},
		{"projects", map[string]any{"Projects":[]ViewerProjectRow{},"Q":"","Status":"","Bank":"","Banks":[]string{},"Statuses":[]string{}}},
		{"project_form", map[string]any{"Project":Project{},"Clients":[]Client{},"Properties":[]Property{},"Edit":false}},
		{"project_detail", ViewerProject{Project:Project{ID:"00000000-0000-0000-0000-000000000001",Title:"Teste",Status:"Em andamento"},Signal:"green",SignalLabel:"Em dia"}},
		{"clients", []Client{}},
		{"properties", map[string]any{"Properties":[]Property{},"Clients":[]Client{},"FilesByProperty":map[string][]PropertyFile360{},"Maps":map[string]string{},"MapStats":map[string]string{},"UpgradeReady":true}},
		{"daily", DailyChecklistView{Projects:[]Project{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T){
			tpl := a.templates[tc.name]
			if tpl == nil { t.Fatalf("template %s não carregado", tc.name) }
			var b bytes.Buffer
			err := tpl.ExecuteTemplate(&b,"base",ViewData{Title:"Teste",User:u,CSRF:"teste",CurrentPath:"/",Version:"teste",Data:tc.data})
			if err != nil { t.Fatalf("template %s falhou ao executar: %v",tc.name,err) }
			if b.Len()==0 { t.Fatalf("template %s gerou saída vazia",tc.name) }
		})
	}
}

func TestSemaforoDeProjetos(t *testing.T) {
	p := Project{Status:"Em análise", Phase:"Análise de crédito"}
	if s, _ := viewerProjectSignal(p, 0, 1, 0); s != "red" { t.Fatalf("esperava vermelho para tarefa vencida, recebeu %s", s) }
	if s, _ := viewerProjectSignal(p, 1, 0, 1); s != "yellow" { t.Fatalf("esperava amarelo para documento pendente, recebeu %s", s) }
	if s, _ := viewerProjectSignal(p, 0, 0, 0); s != "green" { t.Fatalf("esperava verde para projeto em dia, recebeu %s", s) }
}
