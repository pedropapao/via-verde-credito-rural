# Via Verde CAR Desktop 1.0

Primeira versão desktop do ecossistema Via Verde, com foco no Cadastro Ambiental Rural.

## O que a V1 entrega

- clientes e imóveis em banco local SQLite;
- consulta do número do CAR na camada pública do SICAR via WFS;
- situação, condição, município, área, módulos fiscais, geometria, perímetro e centro aproximado;
- mapa online com base OpenStreetMap e imagem de satélite;
- importação de KML e cópia local por imóvel;
- cálculo local de área, perímetro e centro do KML;
- comparação KML × CAR por área e distância entre centros;
- alertas de divergência de área, município e CAR duplicado no cadastro local;
- exportação da geometria pública SICAR em KML;
- demonstrativo técnico em PDF;
- histórico das consultas por imóvel;
- backup ZIP da base local e dos KMLs;
- estrutura para atualização pela internet sem misturar os dados do usuário com o executável.

## Dados locais

O banco e os arquivos ficam em `%AppData%/ViaVerdeCAR` no Windows. O executável pode ser substituído/atualizado sem apagar essa pasta.

## Compilar no Windows

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
cd desktop
wails build -platform windows/amd64 -webview2 download -o ViaVerdeCAR.exe
```

Para gerar instalador, instale NSIS e use `wails build -nsis`.

## Observação jurídica/técnica

O demonstrativo gerado pelo aplicativo é técnico e auxiliar. Não substitui o Demonstrativo oficial do SICAR, certidões, análise ambiental, georreferenciamento ou documentos emitidos pelo órgão competente.

## ViaVerdeCAR 1.7.0 — Inteligência de Crédito Rural

O Raio X passa a oferecer, sob demanda, uma camada financeira complementar baseada em microdados públicos do SICOR/BCB: situação e saldos, liberações, cronograma de desembolso, desclassificações, vínculos de renegociação, Proagro e detalhes produtivos/financeiros das operações vinculadas ao CAR.

A leitura é deliberadamente conservadora: valores localizados são apresentados como **identificados nas bases públicas** e não como dívida total do produtor. Programa, taxa e dados de plantio auxiliam a conferência, mas não substituem o MCR vigente nem a Tábua/Portaria ZARC aplicável.
