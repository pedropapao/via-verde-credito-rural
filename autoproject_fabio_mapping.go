package main

// AutoProjeto Fábio/Irrigação: este arquivo concentra o mapa das células de entrada
// do modelo nativo. O princípio é simples: a planilha original é o motor de cálculo;
// o Go escreve somente entradas e preserva as fórmulas do arquivo.

type autoAgroIrrigationInputMap struct {
	BudgetStartRow int
	BudgetMaxItems int

	InitialHerd map[string]string
	FutureStartCols []string
	PreviousEndCols []string

	NatalityCells        []string
	MortalityAdultCells  []string
	MortalityYoungCells  []string
	MortalityCalfCells   []string
	CullMatricesCells    []string
	CullBullsCells       []string
	PastureSupportCells  []string

	SalePriceCells map[string][]string
	CostUnitCells  map[string]string
	CostQtyCells   map[string][]string
	FixedCostCells map[string][]string
}

var autoFabioMap = autoAgroIrrigationInputMap{
	BudgetStartRow: 7,
	BudgetMaxItems:  9,
	InitialHerd: map[string]string{
		"matrizes": "C6",
		"novilhas_23": "C7",
		"novilhas_12": "C8",
		"bezerras": "C9",
		"bezerros": "C10",
		"novilhos_12": "C11",
		"novilhos_23": "C12",
		"novilhos_mais_3": "C13",
		"touros": "C14",
	},
	// A partir do segundo ano, o INÍCIO de cada categoria deve ser igual ao FIM
	// do ano anterior. No arquivo histórico isso estava digitado manualmente;
	// a base nativa passa a usar fórmulas para eliminar uma fonte de erro.
	FutureStartCols: []string{"G", "K", "O", "S", "Y", "AC", "AG", "AK"},
	PreviousEndCols: []string{"E", "I", "M", "Q", "U", "AA", "AE", "AI"},
	NatalityCells:       []string{"D31", "H31", "L31", "P31", "T31", "Z31", "AD31", "AH31", "AL31"},
	MortalityAdultCells: []string{"D32", "H32", "L32", "P32", "T32", "Z32", "AD32", "AH32", "AL32"},
	MortalityYoungCells: []string{"D33", "H33", "L33", "P33", "T33", "Z33", "AD33", "AH33", "AL33"},
	MortalityCalfCells:  []string{"D34", "H34", "L34", "P34", "T34", "Z34", "AD34", "AH34", "AL34"},
	CullMatricesCells:   []string{"D35", "H35", "L35", "P35", "T35", "Z35", "AD35", "AH35", "AL35"},
	CullBullsCells:      []string{"D36", "H36", "L36", "P36", "T36", "Z36", "AD36", "AH36", "AL36"},
	PastureSupportCells: []string{"D17", "H17", "L17", "P17", "T17", "Z17", "AD17", "AH17", "AL17"},
	SalePriceCells: map[string][]string{
		"bezerros":        {"C12", "C30", "C49"},
		"vacas":           {"C13", "C31", "C50"},
		"novilhos_mais_3": {"C14", "C32", "C51"},
		"bezerras":        {"C15", "C33", "C52"},
	},
	// Linhas conferidas diretamente no modelo Fábio. Estes endereços são células
	// de CUSTO UNITÁRIO; quantidades anuais ficam em CostQtyCells.
	CostUnitCells: map[string]string{
		"mineral":            "C10",
		"veterinaria":        "C14",
		"agronomica":         "C15",
		"aftosa":             "C17",
		"brucelose":          "C18",
		"raiva":              "C19",
		"vermifugo":          "C21",
		"mao_obra":           "C26",
		"energia":            "C28",
		"combustivel":        "C29",
		"curral":             "C43",
		"irrigacao":          "C44",
		"seguridade_social":  "C47",
	},
	CostQtyCells: map[string][]string{
		"veterinaria":       {"D14", "F14", "K14", "M14", "R14", "T14", "Y14", "AA14", "AC14"},
		"agronomica":        {"D15", "F15", "K15", "M15", "R15", "T15", "Y15", "AA15", "AC15"},
		"mao_obra":          {"D26", "F26", "K26", "M26", "R26", "T26", "Y26", "AA26", "AC26"},
		"energia":           {"D28", "F28", "K28", "M28", "R28", "T28", "Y28", "AA28", "AC28"},
		"combustivel":       {"D29", "F29", "K29", "M29", "R29", "T29", "Y29", "AA29", "AC29"},
		"curral":            {"D43", "F43", "K43", "M43", "R43", "T43", "Y43", "AA43", "AC43"},
		"irrigacao":         {"D44", "F44", "K44", "M44", "R44", "T44", "Y44", "AA44", "AC44"},
		"seguridade_social": {"D47", "F47", "K47", "M47", "R47", "T47", "Y47", "AA47", "AC47"},
	},
	FixedCostCells: map[string][]string{
		"administracao": {"B5", "C5", "D5", "E5", "F5", "G5", "H5", "I5", "J5"},
		"impostos":      {"B9", "C9", "D9", "E9", "F9", "G9", "H9", "I9", "J9"},
	},
}

// No caso Fábio havia venda planejada de 81 novilhos +3 no primeiro ano.
// Este valor NÃO é inferido pelo sistema para novos projetos; ele é um movimento
// técnico que precisa vir dos documentos ou ser confirmado pelo usuário.
const autoFabioYear1FinishedSteerSaleCell = "F27"
