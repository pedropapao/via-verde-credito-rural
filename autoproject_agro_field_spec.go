package main

// AutoAgroFieldSpec descreve de onde uma entrada deve vir, onde ela é gravada
// e se o AutoProjeto pode aceitá-la automaticamente ou deve pedir confirmação.
// Resultados calculados pela planilha são marcados como derived e nunca são
// tratados como entrada editável.
type AutoAgroFieldSpec struct {
	Key            string
	Group          string
	Label          string
	Sheet          string
	Cells          []string
	SourcePriority []string
	Mode           string // auto | auto_confirm | confirm | derived
	Critical       bool
	Note           string
}

func autoAgroIrrigationFieldSpecs() []AutoAgroFieldSpec {
	return []AutoAgroFieldSpec{
		{Key:"producer", Group:"Identificação", Label:"Produtor / proponente", Sheet:"01-Orçamento-Fontes", Cells:[]string{"A2"}, SourcePriority:[]string{"projeto/proposta", "cadastro", "documento pessoal"}, Mode:"auto", Critical:true, Note:"Nome deve pertencer ao proponente; fornecedor e responsável técnico não podem substituir o produtor."},
		{Key:"property", Group:"Identificação", Label:"Imóvel beneficiado", Sheet:"01-Orçamento-Fontes", Cells:[]string{"A3"}, SourcePriority:[]string{"projeto/proposta", "matrícula", "CAR", "croqui/KML"}, Mode:"auto", Critical:true},
		{Key:"municipality", Group:"Identificação", Label:"Município do imóvel", Sheet:"01-Orçamento-Fontes", Cells:[]string{"A3"}, SourcePriority:[]string{"projeto/proposta", "matrícula", "CAR"}, Mode:"auto_confirm", Critical:true, Note:"Não usar automaticamente município de outorga, endereço residencial ou sede do fornecedor como município do imóvel."},
		{Key:"line", Group:"Crédito", Label:"Linha de crédito", Sheet:"01-Orçamento-Fontes", Cells:[]string{"F2", "03 e 04-Reembolso!A7"}, SourcePriority:[]string{"projeto/proposta", "orientação do banco"}, Mode:"auto", Critical:true},
		{Key:"purpose", Group:"Crédito", Label:"Finalidade do financiamento", Sheet:"03 e 04-Reembolso", Cells:[]string{"B7"}, SourcePriority:[]string{"projeto/proposta", "orçamento"}, Mode:"auto_confirm", Critical:true},
		{Key:"project_area", Group:"Projeto", Label:"Área do projeto (ha)", Sheet:"01-Orçamento-Fontes", Cells:[]string{"G3"}, SourcePriority:[]string{"croqui/KML/Geo Mapa", "projeto/proposta", "memorial"}, Mode:"auto_confirm", Critical:true, Note:"Cruzar área geométrica com a área declarada; não substituir pela área total da matrícula."},

		{Key:"budget_items", Group:"Orçamento", Label:"Itens do investimento", Sheet:"01-Orçamento-Fontes", Cells:[]string{"A7:H15"}, SourcePriority:[]string{"orçamento/cotação", "projeto final"}, Mode:"auto_confirm", Critical:true, Note:"Descrição, unidade, quantidade, preço unitário e época devem manter vínculo com a fonte."},
		{Key:"budget_total", Group:"Orçamento", Label:"Valor total do investimento", Sheet:"01-Orçamento-Fontes", Cells:[]string{"E17", "F17", "G17"}, SourcePriority:[]string{"fórmulas da planilha"}, Mode:"derived", Critical:true, Note:"Nunca digitar total se os itens estiverem disponíveis; a planilha deve somar."},
		{Key:"interest_rate", Group:"Crédito", Label:"Taxa de juros (% a.a.)", Sheet:"03 e 04-Reembolso", Cells:[]string{"F7"}, SourcePriority:[]string{"tabela oficial Plano Safra por banco/linha/safra", "orientação do banco", "projeto"}, Mode:"auto_confirm", Critical:true, Note:"Só preencher automaticamente quando banco, linha e vigência forem compatíveis."},
		{Key:"term", Group:"Crédito", Label:"Prazo", Sheet:"03 e 04-Reembolso", Cells:nil, SourcePriority:[]string{"projeto/proposta", "orientação do banco"}, Mode:"auto_confirm", Critical:true, Note:"Usado para validar o cronograma; não criar prazo padrão."},
		{Key:"grace", Group:"Crédito", Label:"Carência", Sheet:"03 e 04-Reembolso", Cells:nil, SourcePriority:[]string{"orientação do banco", "projeto/proposta"}, Mode:"confirm", Critical:true, Note:"Se não estiver expressa na documentação, perguntar. Nunca assumir carência padrão."},

		{Key:"initial_herd", Group:"Rebanho", Label:"Rebanho inicial por categoria", Sheet:"06-Evol.Reb", Cells:[]string{"C6:C14"}, SourcePriority:[]string{"ficha sanitária", "projeto técnico", "cadastro pecuário"}, Mode:"auto_confirm", Critical:true, Note:"As categorias precisam fechar com o total de cabeças e ser coerentes com a atividade."},
		{Key:"natality", Group:"Rebanho", Label:"Natalidade (%)", Sheet:"06-Evol.Reb", Cells:autoFabioMap.NatalityCells, SourcePriority:[]string{"projeto técnico", "parâmetro técnico confirmado"}, Mode:"auto_confirm", Critical:true},
		{Key:"mortality_adult", Group:"Rebanho", Label:"Mortalidade adultos (%)", Sheet:"06-Evol.Reb", Cells:autoFabioMap.MortalityAdultCells, SourcePriority:[]string{"projeto técnico", "parâmetro técnico confirmado"}, Mode:"auto_confirm", Critical:true},
		{Key:"mortality_young", Group:"Rebanho", Label:"Mortalidade 1/2 anos (%)", Sheet:"06-Evol.Reb", Cells:autoFabioMap.MortalityYoungCells, SourcePriority:[]string{"projeto técnico", "parâmetro técnico confirmado"}, Mode:"auto_confirm", Critical:true},
		{Key:"mortality_calf", Group:"Rebanho", Label:"Mortalidade de bezerros (%)", Sheet:"06-Evol.Reb", Cells:autoFabioMap.MortalityCalfCells, SourcePriority:[]string{"projeto técnico", "parâmetro técnico confirmado"}, Mode:"auto_confirm", Critical:true},
		{Key:"cull_matrices", Group:"Rebanho", Label:"Descarte de matrizes (%)", Sheet:"06-Evol.Reb", Cells:autoFabioMap.CullMatricesCells, SourcePriority:[]string{"projeto técnico", "planejamento confirmado"}, Mode:"auto_confirm", Critical:true},
		{Key:"cull_bulls", Group:"Rebanho", Label:"Descarte de touros (%)", Sheet:"06-Evol.Reb", Cells:autoFabioMap.CullBullsCells, SourcePriority:[]string{"projeto técnico", "planejamento confirmado"}, Mode:"auto_confirm", Critical:false},
		{Key:"pasture_support", Group:"Rebanho", Label:"Suporte das pastagens (UA)", Sheet:"06-Evol.Reb", Cells:autoFabioMap.PastureSupportCells, SourcePriority:[]string{"projeto técnico", "cálculo agronômico", "parâmetro confirmado"}, Mode:"auto_confirm", Critical:true},
		{Key:"year1_finished_steer_sale", Group:"Rebanho", Label:"Venda planejada de novilhos +3 no 1º ano", Sheet:"06-Evol.Reb", Cells:[]string{autoFabioYear1FinishedSteerSaleCell}, SourcePriority:[]string{"planejamento do projeto", "informação expressa do responsável técnico"}, Mode:"confirm", Critical:true, Note:"Movimento de venda é decisão técnica; não inferir apenas porque existem animais na categoria."},

		{Key:"sale_prices", Group:"Receitas", Label:"Preços de venda por categoria", Sheet:"05-Prod_agrop", Cells:[]string{"C12:C15", "C30:C33", "C49:C52"}, SourcePriority:[]string{"projeto/orçamento de receitas", "fonte de preço atual", "responsável técnico"}, Mode:"auto_confirm", Critical:true, Note:"Preço histórico de um caso padrão não vira preço automático de novos clientes."},
		{Key:"cost_units", Group:"Custos", Label:"Custos unitários pecuários", Sheet:"07-Custeio Pec", Cells:[]string{"C10", "C14:C15", "C17:C19", "C21", "C26", "C28:C29", "C43:C44", "C47"}, SourcePriority:[]string{"projeto/orçamento de custos", "documento do cliente", "responsável técnico"}, Mode:"auto_confirm", Critical:true},
		{Key:"cost_quantities", Group:"Custos", Label:"Quantidades anuais de custeio", Sheet:"07-Custeio Pec", Cells:[]string{"D:AC nas linhas mapeadas"}, SourcePriority:[]string{"projeto técnico", "fórmulas ligadas ao rebanho", "responsável técnico"}, Mode:"auto_confirm", Critical:true, Note:"Quantidades derivadas do rebanho devem permanecer por fórmula; frequências fixas devem ter fonte/confirmacão."},
		{Key:"fixed_costs", Group:"Custos", Label:"Custos fixos anuais", Sheet:"08-Estrutura Custos", Cells:[]string{"B5:J5", "B9:J9"}, SourcePriority:[]string{"projeto técnico", "declaração/cadastro", "responsável técnico"}, Mode:"auto_confirm", Critical:false},

		{Key:"herd_projection", Group:"Resultados", Label:"Evolução do rebanho", Sheet:"06-Evol.Reb", Cells:[]string{"E:AN"}, SourcePriority:[]string{"fórmulas da planilha"}, Mode:"derived", Critical:true},
		{Key:"production_revenue", Group:"Resultados", Label:"Produção e receita", Sheet:"05-Prod_agrop", Cells:[]string{"D:N e blocos seguintes"}, SourcePriority:[]string{"fórmulas da planilha"}, Mode:"derived", Critical:true},
		{Key:"direct_cost", Group:"Resultados", Label:"Custo direto", Sheet:"07-Custeio Pec", Cells:[]string{"E54", "G54", "L54", "N54", "S54", "U54", "Z54", "AB54", "AD54"}, SourcePriority:[]string{"fórmulas da planilha"}, Mode:"derived", Critical:true},
		{Key:"cash_flow", Group:"Resultados", Label:"Fluxo de caixa / capacidade", Sheet:"09-Fluxo Caixa", Cells:[]string{"B:L"}, SourcePriority:[]string{"fórmulas da planilha"}, Mode:"derived", Critical:true, Note:"Nunca preencher diretamente: depende de orçamento, receitas, custos, juros e amortizações."},
	}
}
