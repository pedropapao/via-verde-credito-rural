package main

import "strings"

func defaultChecklist360(p Project) []ChecklistItem {
	items := []ChecklistItem{
		{Code:"DOC_ID", DocumentName:"Documento de identificação / CPF ou CNPJ", Required:true, Status:"Pendente"},
		{Code:"CAR", DocumentName:"Cadastro Ambiental Rural (CAR)", Required:true, Status:"Pendente"},
		{Code:"MATRICULA", DocumentName:"Matrícula ou documento de vínculo com o imóvel", Required:true, Status:"Pendente"},
		{Code:"CCIR", DocumentName:"CCIR", Required:true, Status:"Pendente"},
		{Code:"ITR", DocumentName:"ITR", Required:true, Status:"Pendente"},
		{Code:"AREA", DocumentName:"KML / coordenadas da área do projeto", Required:true, Status:"Pendente"},
		{Code:"ORCAMENTO", DocumentName:"Orçamento / proposta comercial", Required:true, Status:"Pendente"},
	}
	x := strings.ToLower(p.Modality + " " + p.Activity + " " + p.Title)
	if strings.Contains(x,"agrícola") || strings.Contains(x,"agricola") || strings.Contains(x,"soja") || strings.Contains(x,"milho") || strings.Contains(x,"sorgo") || strings.Contains(x,"café") || strings.Contains(x,"cafe") {
		items = append(items, ChecklistItem{Code:"ZARC", DocumentName:"Conferência de ZARC / janela de plantio", Required:true, Status:"Pendente"})
	}
	if strings.Contains(x,"pecu") || strings.Contains(x,"bovin") || strings.Contains(x,"gado") || strings.Contains(x,"animal") {
		items = append(items, ChecklistItem{Code:"SANITARIA", DocumentName:"Ficha sanitária / comprovação do rebanho", Required:true, Status:"Pendente"})
		if strings.Contains(x,"aquisição") || strings.Contains(x,"aquisicao") {
			items = append(items,
				ChecklistItem{Code:"OFERTA", DocumentName:"Carta de oferta / dados do vendedor", Required:true, Status:"Pendente"},
				ChecklistItem{Code:"NF", DocumentName:"Nota fiscal de aquisição", Required:true, Status:"Pendente"},
				ChecklistItem{Code:"GTA", DocumentName:"Guia de Trânsito Animal (GTA)", Required:true, Status:"Pendente"},
			)
		}
	}
	return items
}
