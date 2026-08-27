package main

import "testing"

func TestViewerProjectClosedIncluiAprovadoEReprovado(t *testing.T) {
	for _, status := range []string{"Aprovado", "Reprovado", "Negado", "Contratado", "Concluído"} {
		if !viewerProjectClosed(status) {
			t.Fatalf("status %q deveria ser considerado encerrado", status)
		}
	}
	for _, status := range []string{"Em preparação", "Aguardando documentos", "Enviado ao banco", "Em análise"} {
		if viewerProjectClosed(status) {
			t.Fatalf("status %q não deveria ser considerado encerrado", status)
		}
	}
}

func TestProjetoEncerradoNaoCarregaPendenciasOperacionais(t *testing.T) {
	p := Project{ID: "p1", Status: "Aprovado", Title: "Projeto teste"}
	checklist := []ChecklistItem{{ProjectID: "p1", Required: true, Status: "Pendente", DocumentName: "Matrícula"}}
	tasks := []ProjectTask{{ProjectID: "p1", Status: "Pendente", DueAt: "2000-01-01", Title: "Cobrar documento"}}
	rows := buildViewerProjectRows([]Project{p}, checklist, tasks, nil)
	if len(rows) != 1 {
		t.Fatalf("esperava 1 projeto, recebeu %d", len(rows))
	}
	row := rows[0]
	if row.PendingDocs != 0 {
		t.Fatalf("projeto encerrado não deve ter documentos pendentes operacionais: %d", row.PendingDocs)
	}
	if row.OverdueTasks != 0 {
		t.Fatalf("projeto encerrado não deve ter tarefas vencidas operacionais: %d", row.OverdueTasks)
	}
	if row.Signal != "gray" || row.SignalLabel != "Encerrado" {
		t.Fatalf("sinal inesperado para projeto encerrado: %q / %q", row.Signal, row.SignalLabel)
	}
}
