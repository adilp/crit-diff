package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// RenderStatusBar renders the top status bar line.
// fileIdx is 0-based; displayed as 1-based [N/M].
// ref is the CLI ref arg; empty string means working tree.
func RenderStatusBar(width int, ref string, fileIdx, fileCount int, filePath string, adds, dels, commentCount int, side Side) string {
	return RenderStatusBarWithMode(width, ref, fileIdx, fileCount, filePath, adds, dels, commentCount, side, 0, 0)
}

// RenderStatusBarWithMode renders the status bar with optional mode change display.
// When oldMode and newMode are both non-zero and different, appends mode change info.
func RenderStatusBarWithMode(width int, ref string, fileIdx, fileCount int, filePath string, adds, dels, commentCount int, side Side, oldMode, newMode os.FileMode) string {
	modeStr := ""
	if oldMode != 0 && newMode != 0 && oldMode != newMode {
		modeStr = fmt.Sprintf("  %04o → %04o", oldMode, newMode)
	}

	refDisplay := ref
	if refDisplay == "" {
		refDisplay = "(working tree)"
	}

	sideStr := string(side)

	left := fmt.Sprintf(" cr: %s  [%d/%d] %s", refDisplay, fileIdx+1, fileCount, filePath)
	right := fmt.Sprintf("+%d -%d  C:%d%s  %s ", adds, dels, commentCount, modeStr, sideStr)

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
