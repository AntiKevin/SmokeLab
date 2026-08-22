# CONTRIBUTING.md

## Padrao de commits

Este projeto usa o padrao Conventional Commits para manter o historico legivel e facilitar automacoes futuras.

Formato:

```text
<tipo>(<escopo>): <descricao curta>
```

O escopo e opcional, mas deve ser usado sempre que a mudanca estiver limitada a um modulo especifico.

Exemplos:

```text
feat(engine): adiciona validacao de entrada
fix(gui): corrige exibicao da saudacao
docs: documenta comandos de desenvolvimento
chore(cli): ajusta argumentos do comando principal
```

## Tipos permitidos

- `feat`: adiciona uma nova funcionalidade.
- `fix`: corrige um bug.
- `docs`: altera apenas documentacao.
- `style`: altera formatacao, espacos, nomes ou organizacao visual sem mudar comportamento.
- `refactor`: altera a estrutura do codigo sem adicionar funcionalidade ou corrigir bug.
- `test`: adiciona ou ajusta testes.
- `build`: altera build, empacotamento ou dependencias.
- `chore`: tarefas de manutencao sem impacto direto no comportamento da aplicacao.
- `wip`: tarefas ainda em andamento mas com historico salvo no git.

## Escopos recomendados

- `engine`: regras de negocio e motor interno do aplicativo.
- `gui`: interface visual.
- `cli`: interface de linha de comando.
- `deps`: dependencias do projeto.
- `build`: scripts, empacotamento e geracao de artefatos.

## Regras para mensagens

- Use descricao curta no imperativo ou infinitivo, sem ponto final.
- Escreva a descricao em minusculas quando possivel.
- Prefira commits pequenos e focados em uma unica mudanca.
- Nao misture mudancas de `engine`, `gui` e `cli` no mesmo commit quando elas puderem ser separadas.
- Quando a mudanca afetar regra de negocio compartilhada, o commit deve usar o escopo `engine`.

## Exemplos por modulo

```text
feat(engine): adiciona calculo de mistura
fix(engine): evita resultado invalido sem insumos
feat(gui): adiciona tela de composicao
fix(gui): corrige estado inicial do formulario
feat(cli): adiciona comando para listar receitas
docs: adiciona padrao de commits
build(deps): atualiza dependencias do wails
```
