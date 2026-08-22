// Nome: root_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Valida a montagem do comando raiz da CLI e garante que os comandos
// esperados sejam registrados sem depender de execução real das regras de negócio
// ou de detalhes internos do engine.
package main

import (
	"bytes"
	"testing"
)

func TestRootCommandKeepsPositionalGreetingCompatibility(t *testing.T) {
	t.Parallel()

	command := newRootCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"Kevin"})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := stdout.String(), "Hello Kevin, It's show time!\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestGreetCommandUsesDefaultName(t *testing.T) {
	t.Parallel()

	command := newRootCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"greet"})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := stdout.String(), "Hello World, It's show time!\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
