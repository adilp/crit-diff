package ui

import (
	"fmt"
	"strings"

	"github.com/adilp/crit-diff/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// HelpEntry represents a single keybinding entry in the help overlay.
type HelpEntry struct {
	Category string // non-empty for the first entry in a category
	Key      string
	Action   string
}

// helpBinding defines a keybinding to display in help, with paired action names.
type helpBinding struct {
	actionA string // config action name for first key
	actionB string // config action name for second key (empty if single)
	label   string // display label
}

// helpCategory defines a category with its bindings for the help overlay.
type helpCategory struct {
	name     string
	bindings []helpBinding
}

var helpCategories = []helpCategory{
	{
		name: "Navigation",
		bindings: []helpBinding{
			{"scroll_down", "scroll_up", "scroll down/up"},
			{"pane_left", "pane_right", "switch pane"},
			{"half_page_down", "half_page_up", "half-page down/up"},
			{"top", "bottom", "top/bottom of file"},
			{"next_hunk", "prev_hunk", "next/prev hunk"},
			{"next_file", "prev_file", "next/prev file"},
			{"next_comment", "prev_comment", "next/prev comment"},
			{"expand_below", "expand_above", "expand context"},
		},
	},
	{
		name: "File Switching",
		bindings: []helpBinding{
			{"toggle_tree", "", "toggle tree"},
			{"fuzzy_files", "", "fuzzy file search"},
			{"search_diffs", "", "search diff content"},
		},
	},
	{
		name: "Comments",
		bindings: []helpBinding{
			{"add", "", "add comment"},
			{"visual", "", "visual select"},
			{"edit", "", "edit comment"},
			{"delete", "", "delete comment"},
		},
	},
	{
		name: "View",
		bindings: []helpBinding{
			{"toggle_word_diff", "", "word-level diff"},
			{"toggle_wrap", "", "toggle wrapping"},
			{"search", "", "search in file"},
			{"help", "", "this help"},
			{"quit", "", "quit"},
		},
	},
}

// BuildHelpEntries builds the list of help entries from the given config.
// If the user remapped keys, the help shows the remapped keys.
func BuildHelpEntries(cfg config.Config) []HelpEntry {
	var entries []HelpEntry

	for _, cat := range helpCategories {
		first := true
		for _, b := range cat.bindings {
			keyA := lookupKey(cfg, b.actionA)
			var keyStr string
			if b.actionB != "" {
				keyB := lookupKey(cfg, b.actionB)
				keyStr = fmt.Sprintf("%s/%s", keyA, keyB)
			} else {
				keyStr = keyA
			}

			entry := HelpEntry{
				Key:    keyStr,
				Action: b.label,
			}
			if first {
				entry.Category = cat.name
				first = false
			}
			entries = append(entries, entry)
		}
	}

	return entries
}

// lookupKey finds the key string for a given action name from the config.
func lookupKey(cfg config.Config, actionName string) string {
	if key, ok := cfg.Keys.Normal[actionName]; ok {
		return key
	}
	if key, ok := cfg.Keys.Comment[actionName]; ok {
		return key
	}
	return actionName
}

// RenderHelpOverlay renders the help overlay box at the given terminal dimensions.
// yOffset controls vertical scrolling within the overlay content.
func RenderHelpOverlay(entries []HelpEntry, width, height, yOffset int) string {
	overlayWidth := 58
	if overlayWidth > width-4 {
		overlayWidth = width - 4
	}
	if overlayWidth < 20 {
		overlayWidth = 20
	}

	keyColWidth := 14

	var lines []string

	for _, e := range entries {
		if e.Category != "" {
			if len(lines) > 0 {
				lines = append(lines, "") // blank line before category
			}
			catStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorBarFg))
			lines = append(lines, catStyle.Render(e.Category))
		}

		keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorBarFg))
		actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorBarDim))

		keyStr := keyStyle.Render(e.Key)
		// Pad key column
		keyVisWidth := lipgloss.Width(keyStr)
		padding := keyColWidth - keyVisWidth
		if padding < 1 {
			padding = 1
		}

		actionStr := actionStyle.Render(e.Action)
		line := "  " + keyStr + strings.Repeat(" ", padding) + actionStr
		lines = append(lines, line)
	}

	// Apply scroll offset
	maxVisible := height - 6 // account for border, padding, title
	if maxVisible < 3 {
		maxVisible = 3
	}
	if yOffset > len(lines)-maxVisible {
		yOffset = len(lines) - maxVisible
	}
	if yOffset < 0 {
		yOffset = 0
	}
	if yOffset > 0 && yOffset < len(lines) {
		lines = lines[yOffset:]
	}
	if len(lines) > maxVisible {
		lines = lines[:maxVisible]
	}

	content := strings.Join(lines, "\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBarDim)).
		Width(overlayWidth-2). // account for border
		Padding(1, 1).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true)

	// Custom top border with title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colorBarFg))
	title := titleStyle.Render(" Help ")

	box := style.Render(content)

	// Insert title into top border
	boxLines := strings.Split(box, "\n")
	if len(boxLines) > 0 {
		topBorder := boxLines[0]
		// Replace middle of top border with title
		titleWidth := lipgloss.Width(title)
		borderWidth := lipgloss.Width(topBorder)
		if borderWidth > titleWidth+4 {
			insertAt := 3 // after "╭──"
			// Build new top line: border start + title + border rest
			runes := []rune(topBorder)
			if insertAt+titleWidth < len(runes) {
				newTop := string(runes[:insertAt]) + title + string(runes[insertAt+titleWidth:])
				boxLines[0] = newTop
			}
		}
		box = strings.Join(boxLines, "\n")
	}

	return box
}
