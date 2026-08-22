// Nome: root_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Valida a montagem do comando raiz da CLI e garante que os comandos
// esperados sejam registrados sem depender de execução real das regras de negócio
// ou de detalhes internos do engine.
package main

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRootCommandShowsBannerAndHelp(t *testing.T) {
	t.Parallel()

	command := newRootCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	got := stdout.String()
	bannerLines := strings.Split(strings.TrimRight(smokeLabBanner, "\r\n"), "\n")
	firstLinePadding := widestLine(bannerLines) - utf8.RuneCountInString(bannerLines[0]) + welcomeColumnGap
	wantPrefix := bannerLines[0] + strings.Repeat(" ", firstLinePadding) + "SmokeLab"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("stdout does not start with the banner and side panel")
	}
	for _, want := range []string{"A modular toolbox for developer workflows", "Usage:", "Commands:", "logs"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout does not contain %q", want)
		}
	}
}

func TestRenderWelcomeFallsBackToVerticalLayout(t *testing.T) {
	t.Parallel()

	got := renderWelcome("AA\nB", "Usage:\n command", 8)
	want := "AA\nB\n\nUsage:\n command\n"
	if got != want {
		t.Fatalf("renderWelcome() = %q, want %q", got, want)
	}
}

func TestGreetCommandUsesProvidedName(t *testing.T) {
	t.Parallel()

	command := newRootCommand()
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"greet", "Kevin"})

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
