// Nome: main.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Contém o ponto de entrada da interface de linha de comando do SmokeLab,
// inicializando o comando raiz e centralizando a execução do processo CLI sem
// implementar regras de negócio que pertencem ao engine.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCommand().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
