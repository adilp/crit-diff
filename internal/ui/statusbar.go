package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// RenderStatusBar renders the top status bar line.
// fileIdx is 0-based; displayed as 1-based [N/M].
// ref is the CLI ref arg; empty string means working tree.
func RenderStatusBar(width int, ref string, fileIdx, fileCount int, filePath string, adds, dels, commentCount int, side Side) string {
	refDisplay := ref
	if refDisplay == "" {
		refDisplay = "(working tree)"
	}

	sideStr := string(side)

	left := fmt.Sprintf(" cr: %s  [%d/%d] %s", refDisplay, fileIdx+1, fileCount, filePath)
	right := fmt.Sprintf("+%d -%d  C:%d  %s ", adds, dels, commentCount, sideStr)

	// Pad middle to fill width
	leftWidth := ansi.StringWidth(left)
	rightWidth := ansi.StringWidth(right)
	padding := width - leftWidth - rightWidth
	if padding < 1 {
		padding = 1
	}

	bar := left + strings.Repeat(" ", padding) + right

	style := lipgloss.NewStyle().
		Background(lipgloss.Color(colorBarBg)).
		Foreground(lipgloss.Color(colorBarFg))

	return style.Render(bar)
}
