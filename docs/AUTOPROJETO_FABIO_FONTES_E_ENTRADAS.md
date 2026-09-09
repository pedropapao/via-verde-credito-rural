# AutoProjeto — Fábio Irrigação: fontes, entradas e confirmações

Este documento transforma o caso Fábio em especificação de engenharia para o motor Agro/Irrigação.

## Princípio

O AutoProjeto deve separar quatro classes de informação:

1. **AUTO** — dado objetivo encontrado em fonte compatível e sem conflito.
2. **AUTO + CONFIRMAR** — dado técnico encontrado, mas que deve aparecer para conferência antes de gerar.
3. **CONFIRMAR** — decisão técnica/contratual que não pode ser inferida.
4. **DERIVADO** — resultado calculado pela planilha; nunca deve ser digitado pelo sistema.

## Caso padrão validado

- Proponente: Fábio Garcia dos Prazeres.
- Imóvel: Fazenda Barra Mansa do São Félix.
- Matrícula: 6804.
- Área total: 87,6292 ha.
- Área do projeto: 9,96 ha.
- Atividade: bovinocultura de corte.
- Finalidade: implantação de irrigação por aspersão em pastagem.
- Linha: PRONAMP INVESTIMENTO.
- Prazo: 8 anos.
- Valor do investimento/financiamento na versão padrão-ouro: R$ 299.316,00.
- Taxa do Investimento Pronamp BB no Plano Safra usado como referência: 9% a.a.

Existem versões históricas do projeto com R$ 278.017,50. Elas não devem ser misturadas ao fixture de R$ 299.316,00.

## Orçamento padrão-ouro

| Item | Unidade | Quantidade | Unitário | Total | Época |
|---|---:|---:|---:|---:|---|
| Sistema completo de irrigação por aspersão | conjunto | 1 | R$ 250.500,00 | R$ 250.500,00 | out/26 |
| Postes de madeira | un | 150 | R$ 127,44 | R$ 19.116,00 | nov/26 |
| Escavação mecanizada de valetas | hora | 110 | R$ 270,00 | R$ 29.700,00 | nov/26 |
| **Total** | | | | **R$ 299.316,00** | |

O total é **DERIVADO** pelas fórmulas da planilha.

## Rebanho inicial

| Categoria | Cabeças |
|---|---:|
| Matrizes | 74 |
| Novilhas 2/3 | 0 |
| Novilhas 1/2 | 48 |
| Bezerras | 15 |
| Bezerros | 0 |
| Novilhos 1/2 | 0 |
| Novilhos 2/3 | 81 |
| Novilhos +3 | 0 |
| Touros | 6 |
| **Total** | **224** |

Destino: `06-Evol.Reb!C6:C14`.

### Parâmetros técnicos

- Natalidade: 80%, 80%, depois 83%.
- Mortalidade adultos: 1%.
- Mortalidade 1/2 anos: 3%.
- Mortalidade bezerros: 5%.
- Descarte matrizes: 0%, 10%, 10%, depois 20% nos anos seguintes do modelo validado.
- Descarte touros: 0%.
- Suporte de pastagem: 180 UA no primeiro ano e 200 UA nos anos seguintes.

Esses parâmetros são **AUTO + CONFIRMAR** quando encontrados em projeto técnico. Não são defaults universais.

### Movimento técnico especial

No caso padrão há venda planejada de **81 novilhos +3 no primeiro ano** (`06-Evol.Reb!F27`).

Regra: **CONFIRMAR**, salvo quando a venda estiver expressa documentalmente. A existência de animais em uma categoria não autoriza o AutoProjeto a inventar a venda.

## Preços de venda do caso padrão

- Bezerros: R$ 3.500/cabeça.
- Vacas: R$ 3.400/cabeça.
- Novilhos +3: R$ 4.500/cabeça.
- Bezerras: R$ 2.500/cabeça.

Para novos projetos: **AUTO + CONFIRMAR** com fonte atual. Nunca reutilizar silenciosamente o preço do caso Fábio.

## Custos — células conferidas no arquivo real

| Custo | Custo unitário | Quantidade ano 1 |
|---|---|---|
| Mineral | `C10` | conforme rebanho/modelo |
| Veterinária | `C14` | `D14` |
| Agronômica | `C15` | `D15` |
| Aftosa | `C17` | conforme modelo |
| Brucelose | `C18` | conforme modelo |
| Raiva | `C19` | conforme modelo |
| Vermífugo | `C21` | conforme modelo |
| Mão de obra | `C26` | `D26` |
| Energia elétrica | `C28` | `D28` |
| Combustível | `C29` | `D29` |
| Currais | `C43` | `D43` |
| Manutenção irrigação | `C44` | `D44` |
| Seguridade social | `C47` | `D47` |

O mapeamento anterior de mão de obra/energia/combustível/curral/irrigação estava deslocado para linhas erradas e foi corrigido após conferência do workbook real.

Valores do caso Fábio:

- mineral R$ 5,40/kg;
- veterinária R$ 800/visita, 2 visitas/ano;
- agronômica R$ 1.200/visita, 2 visitas/ano;
- raiva R$ 2/dose;
- vermífugo R$ 3/dose;
- mão de obra R$ 2.000/mês, 12 meses;
- energia R$ 1.617,25/mês, 12 meses;
- combustível R$ 3.000/ano;
- currais R$ 2.000/ano;
- irrigação R$ 5.010/ano, sem lançamento no primeiro ano no caso padrão;
- seguridade social R$ 400/mês, 12 meses.

## Ambiental / outorga

A outorga é uma fonte de apoio para validar:

- titular/CPF;
- empreendimento;
- finalidade irrigação;
- curso d’água;
- vazão autorizada;
- vigência.

Ela **não deve definir sozinha o município cadastral do imóvel**, pois o documento de outorga pode usar município/contexto hidrográfico diferente do projeto, matrícula ou cadastro.

## Carência

No projeto padrão consultado há prazo de 8 anos, mas a carência não aparece expressa como campo textual confiável.

Regra: **CONFIRMAR**. Não usar 1, 2 ou qualquer outro número como default silencioso.

## Cadeia de cálculo que deve permanecer intacta

`Orçamento → Cronograma/Reembolso → Fluxo de caixa`

`Rebanho inicial + coeficientes + movimentos → Evolução do rebanho`

`Evolução do rebanho + preços → Produção/receitas`

`Evolução do rebanho + custos unitários → Custeio pecuário`

`Custeio → Estrutura de custos → Fluxo de caixa/capacidade`

O AutoProjeto escreve entradas. O Excel calcula resultados.
