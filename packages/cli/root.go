package main

import (
	"fmt"

	"SmokeLab/packages/engine"
	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "smokelab [name]",
		Short:         "SmokeLab command-line interface",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return greet(cmd, args)
		},
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
