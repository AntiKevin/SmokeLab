// Nome: banner.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descricao: Renderiza a identidade visual da CLI na tela inicial sem adicionar
// saida aos subcomandos usados em pipelines ou automacoes.
package main

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

//go:embed assets/banner.txt
var smokeLabBanner string

const (
	welcomeColumnGap        = 4
	welcomeDescriptionWidth = 24
)

func showWelcome(cmd *cobra.Command, _ []string) error {
	banner := strings.TrimRight(smokeLabBanner, "\r\n")
	details := buildWelcomeDetails(cmd)
	_, err := fmt.Fprint(cmd.OutOrStdout(), renderWelcome(banner, details, terminalWidth(cmd.OutOrStdout())))
	return err
}

func buildWelcomeDetails(cmd *cobra.Command) string {
	commands := cmd.Commands()
	nameWidth := 0
	for _, child := range commands {
		if child.IsAvailableCommand() && len(child.Name()) > nameWidth {
			nameWidth = len(child.Name())
		}
	}

	lines := []string{
		"SmokeLab",
		cmd.Short,
		"",
		"Usage:",
		fmt.Sprintf("  %s <command> [flags]", cmd.CommandPath()),
		"",
		"Commands:",
	}
	for _, child := range commands {
		if !child.IsAvailableCommand() {
			continue
		}
		lines = append(lines, fmt.Sprintf(
			"  %-*s  %s",
			nameWidth,
			child.Name(),
			truncateText(child.Short, welcomeDescriptionWidth),
		))
	}
	lines = append(lines,
		"",
		"Help:",
		fmt.Sprintf("  %s <command> --help", cmd.CommandPath()),
	)
	return strings.Join(lines, "\n")
}

func renderWelcome(banner string, details string, width int) string {
	bannerLines := strings.Split(banner, "\n")
	detailLines := strings.Split(details, "\n")
	bannerWidth := widestLine(bannerLines)
	detailWidth := widestLine(detailLines)
	requiredWidth := bannerWidth + welcomeColumnGap + detailWidth
	if width > 0 && width < requiredWidth {
		return banner + "\n\n" + details + "\n"
	}

	lineCount := max(len(bannerLines), len(detailLines))
	var output strings.Builder
	for index := 0; index < lineCount; index++ {
		left := ""
		if index < len(bannerLines) {
			left = bannerLines[index]
		}
		output.WriteString(left)

		if index < len(detailLines) {
			padding := bannerWidth - utf8.RuneCountInString(left) + welcomeColumnGap
			output.WriteString(strings.Repeat(" ", padding))
			output.WriteString(detailLines[index])
		}
		output.WriteByte('\n')
	}
	return output.String()
}

func widestLine(lines []string) int {
	width := 0
	for _, line := range lines {
		width = max(width, utf8.RuneCountInString(line))
	}
	return width
}

func truncateText(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width-3]) + "..."
}
