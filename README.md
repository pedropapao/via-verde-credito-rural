# Via Verde Crédito Rural — Web v2.0

Sistema web em Go para gestão de projetos de crédito rural da Via Verde, com login privado, perfil proprietário, usuários somente leitura, carteira de projetos, produtores, propriedades, documentos em nuvem, relatório diário e base de referências técnicas.

## Arquitetura escolhida

- Aplicação: Go (biblioteca padrão, sem framework obrigatório)
- Hospedagem: Render Web Service
- Banco de dados: Supabase Postgres
- Arquivos: Supabase Storage privado
- Login: autenticação própria no backend, sessões armazenadas no banco
- Perfil inicial: Pedro Massoli (`pedro.massoli`)
- Visualizadores: somente por convite gerado pelo proprietário
- Atualizações: VS Code -> Git -> GitHub -> Render faz novo deploy automaticamente

## Segurança

A chave `SUPABASE_SERVICE_ROLE_KEY` fica somente no backend/Render. Ela nunca é enviada ao navegador. As tabelas têm Row Level Security habilitado sem políticas públicas. O aplicativo valida papel `owner`/`viewer` antes de operações de escrita.

## Primeira implantação

Veja `docs/DEPLOY_GRATIS_PASSO_A_PASSO.md`.

## Desenvolvimento no VS Code

Veja `docs/ATUALIZAR_PELO_VSCODE.md`.

## Banco e carga inicial

1. `supabase/01_schema.sql` — cria as tabelas e índices.
2. `supabase/02_seed_via_verde.sql` — importa a carteira inicial e o controle diário disponibilizado em agosto de 2026.

## Observação normativa

O módulo "Regras / MCR" é um índice de conferência. Regras financeiras, taxas, limites e enquadramentos devem ser validados na fonte oficial vigente antes de orientar ou finalizar um projeto.
