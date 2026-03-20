package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// RenderHelpBar renders the bottom help bar line with context-sensitive hints.
func RenderHelpBar(width int, mode InputMode) string {
	type hint struct {
		key    string
		action string
	}

	var hints []hint
	switch mode {
	case InputModeNormal:
		hints = []hint{
			{"j/k", "scroll"},
			{"h/l", "pane"},
			{"Tab", "tree"},
			{"c", "comment"},
			{"/", "search"},
			{"q", "quit"},
		}
	default:
		hints = []hint{
			{"Esc", "back"},
		}
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorBarFg))
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorBarDim))

	var parts []string
	for _, h := range hints {
		part := keyStyle.Render("["+h.key+"]") + " " + actionStyle.Render(h.action)
		parts = append(parts, part)
	}

	content := " " + strings.Join(parts, "  ")

	// Pad to fill width
	contentWidth := ansi.StringWidth(content)
	padding := width - contentWidth
	if padding < 0 {
		padding = 0
	}
	content = content + strings.Repeat(" ", padding)

	style := lipgloss.NewStyle().
		Background(lipgloss.Color(colorBarBg))

	return style.Render(content)
}
