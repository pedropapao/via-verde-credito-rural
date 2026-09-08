package main

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestAutoFieldKeyFromLabel(t *testing.T) {
	cases := map[string]string{
		"Nome do Proponente": "producer",
		"CPF/CNPJ":           "cpf",
		"Município":          "municipality",
		"Área a financiar":   "area",
		"Valor pretendido":   "amount",
	}
	for in, want := range cases {
		if got := autoFieldKeyFromLabel(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestAutoFillWorkbookOnlyBlankSafeCells(t *testing.T) {
	f := excelize.NewFile()
	s := f.GetSheetName(0)
	_ = f.SetCellValue(s, "A1", "Nome do Proponente")
	_ = f.SetCellValue(s, "A2", "CPF/CNPJ")
	_ = f.SetCellValue(s, "A3", "Valor pretendido")
	_ = f.SetCellFormula(s, "B4", "=1+1")
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	filled, mapped, err := autoFillWorkbook(buf.Bytes(), map[string]string{
		"producer": "JOAO TESTE",
		"cpf":      "123.456.789-09",
		"amount":   "R$ 30.000,00",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mapped) < 3 {
		t.Fatalf("expected at least 3 mappings, got %v", mapped)
	}
	g, err := excelize.OpenReader(bytes.NewReader(filled))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got, _ := g.GetCellValue(s, "B1"); got != "JOAO TESTE" {
		t.Fatalf("producer not filled: %q", got)
	}
	if got, _ := g.GetCellValue(s, "B2"); got != "123.456.789-09" {
		t.Fatalf("cpf not filled: %q", got)
	}
	if got, _ := g.GetCellValue(s, "B3"); got == "" {
		t.Fatal("amount not filled")
	}
}

func TestMakeSimpleDOCXIsZipWithDocument(t *testing.T) {
	b := makeSimpleDOCX("Teste", []string{"Linha 1"})
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			found = true
		}
	}
	if !found {
		t.Fatal("word/document.xml missing")
	}
}
