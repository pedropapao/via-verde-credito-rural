package main

import (
	"strings"
	"testing"
)

func TestAutoTemplateMatchesByBankLineActivity(t *testing.T) {
	tpl := AutoProjectTemplate{Bank: "Banco do Brasil", Line: "PRONAF", Activity: "Café"}
	values := map[string]string{"bank": "Banco do Brasil", "line": "Pronaf", "activity": "cafe"}
	if !autoTemplateMatches(tpl, values) {
		t.Fatal("modelo compatível deveria ser selecionado")
	}
	values["activity"] = "soja"
	if autoTemplateMatches(tpl, values) {
		t.Fatal("modelo específico de café não deve casar com soja")
	}
}

func TestAutoTemplateGlobalActivity(t *testing.T) {
	tpl := AutoProjectTemplate{Bank: "Sicoob", Line: "PRONAMP", Activity: ""}
	values := map[string]string{"bank": "Sicoob", "line": "PRONAMP", "activity": "pecuária de leite"}
	if !autoTemplateMatches(tpl, values) {
		t.Fatal("modelo geral deveria aceitar qualquer atividade")
	}
}

func TestAutoFillDOCXPlaceholders(t *testing.T) {
	xml := `<w:document><w:body><w:p><w:r><w:t>Produtor: {{producer}} - Banco: [[bank]]</w:t></w:r></w:p></w:body></w:document>`
	values := map[string]string{"producer": "José & Filhos", "bank": "Banco do Brasil"}
	out, mappings := autoFillDOCXPlaceholders(xml, values)
	if strings.Contains(out, "{{producer}}") || strings.Contains(out, "[[bank]]") {
		t.Fatal("placeholders deveriam ter sido substituídos")
	}
	if !strings.Contains(out, "José &amp; Filhos") || !strings.Contains(out, "Banco do Brasil") {
		t.Fatalf("conteúdo preenchido inesperado: %s", out)
	}
	if len(mappings) != 2 {
		t.Fatalf("esperava 2 mapeamentos, obteve %d", len(mappings))
	}
}
