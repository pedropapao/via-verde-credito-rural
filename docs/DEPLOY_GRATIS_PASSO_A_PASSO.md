# Publicar a Via Verde na internet — passo a passo

A configuração recomendada para esta versão usa **Supabase Free** para banco/arquivos e **Render Free** para executar o programa Go.

## Resultado final

Você terá um endereço parecido com:

`https://via-verde-credito-rural.onrender.com`

Login principal:

- Nome: Pedro Massoli
- Usuário: `pedro.massoli`
- Senha: definida por você na variável `ADMIN_PASSWORD`

Não existe cadastro público. Pedro gera links de convite na aba **Acessos** e os convidados entram com perfil **Visualização**, sem permissão para editar.

---

## 1. Criar o banco gratuito no Supabase

1. Entre em https://supabase.com e crie uma conta.
2. Crie um projeto chamado, por exemplo, `via-verde-credito-rural`.
3. Aguarde o banco ficar pronto.
4. Abra **SQL Editor**.
5. Copie todo o conteúdo de `supabase/01_schema.sql`, cole e execute.
6. Depois copie `supabase/02_seed_via_verde.sql`, cole e execute.

A segunda carga já cria os produtores/projetos recuperados do controle diário e os lançamentos históricos do relatório.

### Guardar duas informações do Supabase

No painel do Supabase, copie:

- Project URL -> será `SUPABASE_URL`
- Service role key / Secret key do servidor -> será `SUPABASE_SERVICE_ROLE_KEY`

**Nunca** coloque a service role key no código, em prints ou no GitHub.

O aplicativo cria automaticamente o bucket privado `via-verde-files` no primeiro início.

---

## 2. Colocar o código no GitHub

A pasta já contém `PUBLICAR_GITHUB.bat`.

Antes:

1. Crie um repositório privado no GitHub, por exemplo `via-verde-credito-rural`.
2. Copie a URL do repositório.
3. Na pasta do projeto, dê dois cliques em `PUBLICAR_GITHUB.bat`.
4. Cole a URL quando solicitado.

O `.gitignore` impede o envio do arquivo `.env`.

---

## 3. Criar o site gratuito no Render

1. Entre em https://render.com.
2. Crie sua conta e conecte o GitHub.
3. Escolha **New -> Blueprint** ou **New -> Web Service**.
4. Selecione o repositório da Via Verde.
5. O arquivo `render.yaml` já contém build, start, health check e plano Free.

Preencha as variáveis pedidas:

- `APP_BASE_URL` = URL que o Render fornecer ao site, por exemplo `https://via-verde-credito-rural.onrender.com`
- `SUPABASE_URL` = URL copiada do Supabase
- `SUPABASE_SERVICE_ROLE_KEY` = chave secreta do servidor
- `ADMIN_PASSWORD` = sua senha inicial forte

As outras variáveis já estão no `render.yaml`:

- `ADMIN_NAME=Pedro Massoli`
- `ADMIN_USERNAME=pedro.massoli`
- `SUPABASE_STORAGE_BUCKET=via-verde-files`

Clique em **Deploy**.

No primeiro início, se a tabela `users` estiver vazia, o programa cria automaticamente o perfil proprietário Pedro Massoli usando `ADMIN_PASSWORD`.

---

## 4. Entrar no site

Abra a URL do Render.

Usuário: `pedro.massoli`

Senha: a que você definiu em `ADMIN_PASSWORD`.

Depois vá em **Meu perfil** e troque a senha se desejar.

---

## 5. Dar acesso de visualização a outra pessoa

1. Entre como Pedro.
2. Abra **Acessos**.
3. Informe nome/e-mail e validade do convite.
4. Clique em **Gerar link de convite**.
5. Copie o link exibido e envie à pessoa.
6. Ela escolhe usuário e senha.

O perfil criado é `viewer`: pode consultar projetos, documentos e relatórios, mas os endpoints de alteração ficam bloqueados no backend.

---

## 6. Limitação do plano gratuito

O Render pode desligar um Web Service gratuito depois de um período sem requisições. Quando isso ocorrer, a primeira abertura depois da inatividade pode demorar mais que o normal. Para uso pessoal e testes isso costuma ser aceitável; se o sistema se tornar crítico para uma equipe, avalie posteriormente uma instância paga.

Os limites do Supabase Free também podem mudar. Confira a página oficial de preços antes de armazenar grandes volumes de PDFs, planilhas e imagens.
