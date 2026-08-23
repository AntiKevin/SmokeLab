// Nome: root.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Define o comando raiz da CLI, registra os subcomandos disponíveis e
// mantém a composição da interface de terminal separada das regras reutilizáveis
// executadas pelos pacotes do engine.
package main

import "github.com/spf13/cobra"

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "smokelab",
		Short:         "A modular toolbox for developer workflows",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE:          showWelcome,
	}

	root.AddCommand(newLogsCommand())
	return root
}
