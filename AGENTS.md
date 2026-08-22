# AGENTS.md

## Regras do projeto

Este projeto deve manter a arquitetura modular baseada em tres pacotes principais:

- `packages/engine`: contem apenas as execucoes de regras de negocio e o motor interno do aplicativo.
- `packages/gui`: contem a interface visual do aplicativo.
- `packages/cli`: contem a interface de linha de comando.

## Arquitetura dos modulos

O `engine` e a base funcional do projeto. Toda regra de negocio, validacao, fluxo interno, processamento e comportamento reutilizavel deve ficar neste modulo. O `engine` nao deve depender de detalhes da `gui` nem da `cli`.

A `gui` deve atuar como camada de apresentacao visual. Ela pode chamar servicos e funcoes do `engine`, mas nao deve implementar regra de negocio propria quando essa regra puder ser compartilhada com outras interfaces.

A `cli` deve atuar como camada de apresentacao por terminal. Ela pode interpretar argumentos, formatar saidas e chamar o `engine`, mas a logica de negocio deve permanecer no `engine`.

## Diretrizes para alteracoes

- Ao criar uma nova funcionalidade, implemente primeiro a regra de negocio no `engine`.
- Exponha a funcionalidade na `gui` e/ou na `cli` apenas como adaptadores de interface.
- Evite duplicar regras entre `gui` e `cli`; compartilhe pelo `engine`.
- Mantenha dependencias de interface fora do `engine`.
- Quando uma regra precisar ser usada por mais de uma interface, ela deve obrigatoriamente estar no `engine`.
- Mudancas estruturais devem preservar a separacao entre motor interno, interface visual e interface de linha de comando.

## Contribuindo

- Antes de contribuir no projeto leia o arquivo de [CONTRIBUTING](CONTRIBUTING.md)
