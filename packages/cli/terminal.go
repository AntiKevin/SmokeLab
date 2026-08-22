// Nome: terminal.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descricao: Centraliza a deteccao de terminal usada pelas apresentacoes da CLI.
package main

import (
	"io"
	"os"

	"golang.org/x/term"
)

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func terminalWidth(writer io.Writer) int {
	file, ok := writer.(*os.File)
	if !ok {
		return 0
	}

	width, _, err := term.GetSize(int(file.Fd()))
	if err != nil {
		return 0
	}
	return width
}
