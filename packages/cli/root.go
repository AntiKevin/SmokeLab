// Nome: root.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Define o comando raiz da CLI, registra os subcomandos disponíveis e
// mantém a composição da interface de terminal separada das regras reutilizáveis
// executadas pelos pacotes do engine.
package main

import (
	"fmt"

	"SmokeLab/packages/engine"
	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "smokelab",
		Short:         "A modular toolbox for developer workflows",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE:          showWelcome,
	}

	root.AddCommand(newGreetCommand())
	root.AddCommand(newLogsCommand())
	return root
}

func newGreetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "greet [name]",
		Short: "Print a greeting",
		Args:  cobra.MaximumNArgs(1),
		RunE:  greet,
	}
}

func greet(cmd *cobra.Command, args []string) error {
	name := "World"
	if len(args) == 1 {
		name = args[0]
	}

	_, err := fmt.Fprintln(cmd.OutOrStdout(), engine.NewGreetingService().Greet(name))
	return err
}
