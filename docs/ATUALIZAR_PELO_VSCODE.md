# Atualizar o site pelo VS Code

A partir desta versão, o fluxo de atualização fica mais simples que no aplicativo Windows.

## Abrir o projeto

Dê dois cliques em:

`ABRIR_NO_VSCODE.bat`

ou no VS Code use **File -> Open Folder** e escolha a pasta do projeto.

## Testar antes de publicar

No terminal do VS Code:

```powershell
go test ./...
```

Para executar localmente, copie `.env.example` para `.env`, coloque as credenciais do mesmo Supabase e rode:

```powershell
go run .
```

Abra `http://localhost:8080`.

## Publicar uma atualização

Depois de editar e testar, execute:

`ATUALIZAR_SITE.bat`

O script:

1. executa `go test ./...`;
2. se houver erro, para sem publicar;
3. cria um commit Git;
4. envia para o GitHub.

O Render detecta o novo commit e publica a nova versão automaticamente.

## O banco não é apagado a cada atualização

O código fica no GitHub/Render.

Os clientes, projetos, relatórios e usuários ficam no Supabase Postgres.

Os PDFs, planilhas e outros anexos ficam no Supabase Storage.

Portanto, atualizar o código do site não apaga a carteira.
