package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelaClientesExecutaEPermiteNovoCadastro(t *testing.T) {
	a := NewApp(Config{}, nil)
	tpl := a.templates["client_manager"]
	if tpl == nil {
		t.Fatal("template da tela de clientes não foi carregado")
	}
	view := ManagerClientsView{Clients: []ManagerClientRow{{Client: Client{ID:"cliente-1", Name: "Cliente Teste", Document: "000.000.000-00"}, ProjectCount: 2, ActiveCount: 1}}, Total: 1}
	var b bytes.Buffer
	err := tpl.ExecuteTemplate(&b, "base", ViewData{Title: "Clientes", User: &User{Name: "Pedro", Role: "owner"}, CSRF: "teste", CurrentPath: "/clients", Version: "teste", Data: view})
	if err != nil {
		t.Fatalf("tela de clientes falhou ao executar: %v", err)
	}
	out := b.String()
	for _, want := range []string{"Adicionar produtor", `action="/clients"`, "Cliente Teste", "2</b> projeto(s)", `/clients/cliente-1`, `/projects/new?client_id=cliente-1`} {
		if !strings.Contains(out, want) {
			t.Fatalf("tela de clientes não contém %q", want)
		}
	}
}

func TestFichaClienteEProjetoNovoCarregam(t *testing.T) {
	a := NewApp(Config{}, nil)
	for _, page := range []string{"client_detail", "manager_project_new"} {
		if a.templates[page] == nil {
			t.Fatalf("template %s não foi carregado", page)
		}
	}
	view := ManagerClientDetailView{Client:Client{ID:"cliente-1",Name:"Cliente Teste"}, Projects:[]Project{{ID:"projeto-1",Title:"Custeio milho",Status:"Em preparação"}}, ProjectCount:1, ActiveCount:1}
	var b bytes.Buffer
	if err:=a.templates["client_detail"].ExecuteTemplate(&b,"base",ViewData{Title:"Cliente Teste",User:&User{Name:"Pedro",Role:"owner"},CSRF:"teste",CurrentPath:"/clients/cliente-1",Version:"teste",Data:view}); err!=nil {
		t.Fatalf("ficha do cliente falhou: %v",err)
	}
	out:=b.String()
	for _, want:=range []string{"Atualizar cadastro","Arquivar cliente","Custeio milho","/projects/new?client_id=cliente-1"} {
		if !strings.Contains(out,want) { t.Fatalf("ficha não contém %q",want) }
	}
}

func TestArquivamentoClientePreservaObservacao(t *testing.T) {
	original := "Cliente prefere contato por WhatsApp"
	stored := clientStoredNotes(original, true)
	if !clientArchived(stored) { t.Fatal("cliente deveria estar marcado como arquivado") }
	if got:=clientDisplayNotes(stored); got!=original { t.Fatalf("observação mudou: %q",got) }
	if got:=clientStoredNotes(stored,false); got!=original { t.Fatalf("reativação deveria remover só o marcador: %q",got) }
}

func TestMenuTemAcessoAClientes(t *testing.T) {
	body, err := webFS.ReadFile("web/templates/base.html")
	if err != nil {
		t.Fatalf("não foi possível ler menu: %v", err)
	}
	if !strings.Contains(string(body), `href="/clients"`) {
		t.Fatal("menu principal precisa ter acesso à área de clientes")
	}
}

func TestRotaClientesExisteNoGerenciador(t *testing.T) {
	a := NewApp(Config{}, nil)
	for _, path := range []string{"/clients", "/clients/abc", "/projects/new"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		a.routesManagerCAR().ServeHTTP(w, r)
		if w.Code == http.StatusNotFound {
			t.Fatalf("rota %s não foi registrada", path)
		}
	}
}
