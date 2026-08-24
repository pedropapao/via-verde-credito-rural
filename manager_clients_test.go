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
	tpl := a.templates["documents"]
	if tpl == nil {
		t.Fatal("template usado pela tela de clientes não foi carregado")
	}
	view := ManagerClientsView{Clients: []ManagerClientRow{{Client: Client{Name: "Cliente Teste", Document: "000.000.000-00"}, ProjectCount: 2, ActiveCount: 1}}, Total: 1}
	var b bytes.Buffer
	err := tpl.ExecuteTemplate(&b, "base", ViewData{Title: "Clientes", User: &User{Name: "Pedro", Role: "owner"}, CSRF: "teste", CurrentPath: "/clients", Version: "teste", Data: view})
	if err != nil {
		t.Fatalf("tela de clientes falhou ao executar: %v", err)
	}
	out := b.String()
	for _, want := range []string{"Adicionar produtor", `action="/clients"`, "Cliente Teste", "2</b> projeto(s)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("tela de clientes não contém %q", want)
		}
	}
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
	r := httptest.NewRequest(http.MethodGet, "/clients", nil)
	w := httptest.NewRecorder()
	a.routesManagerCAR().ServeHTTP(w, r)
	if w.Code == http.StatusNotFound {
		t.Fatal("rota /clients não foi registrada")
	}
}
