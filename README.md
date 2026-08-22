# SmokeLab

SmokeLab e uma aplicacao desktop feita com Go, Wails, React, Vite e TypeScript.
O repositorio tambem possui uma entrada de linha de comando para exercitar as
regras compartilhadas do pacote `engine`.

## Estrutura do projeto

```text
.
├── Makefile
├── go.mod
├── build/
└── packages/
    ├── engine/  # regras de negocio e codigo compartilhado
    ├── cli/   # entrada de linha de comando
    └── gui/   # aplicacao Wails + React + Vite
```

## Pre-requisitos

Instale antes de rodar o projeto:

- Go compativel com o `go.mod` (`go 1.25.0` ou superior).
- Node.js e npm para o frontend.
- Wails CLI v2 para rodar e empacotar a aplicacao desktop.

Para conferir o ambiente:

```bash
go version
node --version
npm --version
wails version
wails doctor
```

Se o Wails CLI ainda nao estiver instalado, use a versao do Wails usada pelo
projeto:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
```

Em Linux, o Wails tambem depende das bibliotecas nativas de GTK/WebKit. Use
`wails doctor` para ver exatamente o que falta na sua distribuicao.

## Instalacao

Na raiz do repositorio, baixe as dependencias de Go:

```bash
go mod download
```

Instale as dependencias do frontend:

```bash
cd packages/gui
npm ci
```

Depois volte para a raiz para usar os comandos do `Makefile`:

```bash
cd ../..
```

## Rodando a aplicacao desktop

Use o alvo principal de desenvolvimento:

```bash
make dev
```

Esse comando entra em `packages/gui` e executa `wails dev`, iniciando a janela
desktop com o frontend Vite em modo de desenvolvimento.

`make run` e um alias para `make dev`:

```bash
make run
```

Voce pode repassar argumentos para o Wails usando `ARGS`:

```bash
make dev ARGS="-debug"
```

## Rodando apenas o frontend

Para trabalhar somente no React/Vite:

```bash
cd packages/gui
npm run dev
```

Isso sobe apenas o servidor do Vite. As chamadas ao backend expostas pelo Wails
em `window.go...` so funcionam dentro da aplicacao Wails; para testar o fluxo
completo, use `make dev`.

Para gerar e visualizar o build estatico do frontend:

```bash
cd packages/gui
npm run build
npm run preview
```

O build do frontend e gerado em `packages/gui/dist` e tambem e usado pelo Wails
no empacotamento da aplicacao.

## Rodando a CLI

A CLI usa o pacote `engine` diretamente:

```bash
make cli
```

Por padrao ela usa `World` como nome. Para passar outro valor:

```bash
make cli ARGS="Kevin"
```

O comando equivalente, sem `Makefile`, e:

```bash
go run ./packages/cli Kevin
```

### Ingestao de logs estruturados

A CLI tambem pode ingerir logs estruturados emitidos como NDJSON, ou seja, um
objeto JSON completo por linha:

```bash
go run ./packages/cli logs ingest --stdin
```

Tambem e possivel ler um arquivo uma vez:

```bash
go run ./packages/cli logs ingest --file app.ndjson
```

Ou acompanhar novas linhas adicionadas ao arquivo:

```bash
go run ./packages/cli logs ingest --file app.ndjson --follow
```

Por padrao, os logs sao gravados no banco local da aplicacao dentro do diretorio
de configuracao do usuario. Para escolher outro arquivo:

```bash
go run ./packages/cli logs ingest --file app.ndjson --db /tmp/smokelab.db
```

Cada linha deve conter os campos fixos obrigatorios `timestamp`, `level` e
`message`. Todos os outros campos do objeto, como `context`, `service_name` e
`global_event_name`, sao armazenados em `params` como JSON.

Exemplo aceito:

```json
{"context":{"className":"NewRelicMetricsReporter","lockKey":"newrelic-payments-metrics","method":"report"},"global_event_name":"CRON_LOCK_ACQUIRED","level":"info","message":"[CronWithLock] Lock acquired (key: newrelic-payments-metrics). Executing routine...","service_name":"service-payments-undefined","timestamp":"2026-08-21T19:38:00.011Z"}
```

O formato colorido de `console.log`/pretty-print com ANSI, chaves sem aspas e
strings concatenadas nao faz parte deste primeiro corte. Se houver linhas
invalidas, o comportamento padrao e falhar; use `--on-invalid skip` para
continuar e contabilizar as linhas rejeitadas.

## Testes

Rode todos os testes Go:

```bash
make test
```

Tambem da para repassar argumentos do `go test`:

```bash
make test ARGS="-v"
```

## Build

Para gerar o binario desktop com Wails:

```bash
make build
```

O binario gerado fica em `build/bin/`.

Para compilar apenas o frontend:

```bash
make gui-build
```

## Pacotes

### `packages/engine`

Contem as regras de negocio e codigo compartilhado entre as interfaces do
projeto. A CLI e a GUI devem consumir esse pacote para evitar duplicacao de
comportamento.

### `packages/cli`

Contem a entrada de linha de comando da aplicacao. Hoje ela le um nome pelos
argumentos e imprime a saudacao retornada pelo `engine`.

### `packages/gui`

Contem a aplicacao Wails atual, incluindo:

- backend Go da janela desktop;
- frontend React/Vite/TypeScript;
- assets;
- codigo gerado em `wailsjs`;
- configuracao em `wails.json`.

## Arquivos gerados

Alguns diretorios sao resultado de build ou geracao de codigo:

- `packages/gui/dist`: build estatico do frontend usado pelo Wails.
- `packages/gui/wailsjs`: bindings gerados pelo Wails para o frontend.
- `build/bin`: binarios desktop gerados por `make build`.
