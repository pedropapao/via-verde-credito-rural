# AutoProjeto — Caso padrão Fábio Irrigação

Este documento registra o primeiro caso de validação estrutural do motor nativo Agro/Irrigação.

## Objetivo
Reproduzir a lógica da planilha de investimento agropecuário/irrigação preservando a planilha-base, suas fórmulas e suas ligações internas. O AutoProjeto deve escrever apenas dados de entrada e deixar a própria planilha calcular orçamento, cronograma, reembolso, evolução do rebanho, produção, custos e fluxo de caixa.

## Estrutura validada
- 01-Orçamento-Fontes
- 02-Cronograma Exe.
- 03 e 04-Reembolso
- 05-Prod_agrop
- 06-Evol.Reb
- 07-Custeio Pec
- 08-Estrutura Custos
- 09-Fluxo Caixa

## Dependências principais
1. Orçamento alimenta cronograma, reembolso e fluxo de caixa.
2. Evolução do rebanho é calculada pelas fórmulas originais do modelo a partir do rebanho inicial, coeficientes técnicos e movimentos.
3. Produção agropecuária lê vendas da evolução do rebanho e aplica preços por categoria.
4. Custeio pecuário lê quantidades do rebanho e aplica custos unitários.
5. Estrutura de custos consolida o custeio.
6. Fluxo de caixa consolida receitas, custos, juros e amortizações.

## Regra de engenharia
Não substituir por cálculos genéricos aquilo que o modelo já calcula corretamente. O motor deve preservar as fórmulas oficiais do arquivo e preencher somente células de entrada.

## Caso padrão numérico anonimizado
O caso de referência usa, entre outros parâmetros: área de projeto 9,96 ha, investimento total de R$ 299.316,00, taxa de 9% a.a., rebanho inicial por categoria, coeficientes anuais de natalidade/mortalidade/descarte e custos operacionais. O nome do produtor e demais dados pessoais não fazem parte do fixture de teste.

## Critério de aprovação
O motor é aprovado quando a base limpa mantém as mesmas fórmulas críticas do caso validado e, após o preenchimento das entradas, o Excel reproduz a cadeia de cálculo sem #REF!, #DIV/0!, #VALUE!, #NAME? ou #N/A.
