package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// CommentOverlay holds the state for the comment input modal.
type CommentOverlay struct {
	Active       bool
	Input        textinput.Model
	Line         int    // file line number being commented
	EndLine      int    // end line for range comments (0 = single-line)
	Side         Side   // old or new
	RowIndex     int    // row in paired lines (for positioning)
	editingID    string // non-empty when editing an existing comment (stores comment ID)
	rangeSnippet string // content snippet for range comments (first line of range)
}

// NewCommentOverlay creates a new active comment overlay for the given line.
func NewCommentOverlay(line int, side Side, rowIndex int) CommentOverlay {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 0 // unlimited
	ti.Placeholder = "Type your comment..."
	return CommentOverlay{
		Active:   true,
		Input:    ti,
		Line:     line,
		Side:     side,
		RowIndex: rowIndex,
	}
}

// Value returns the current text input value.
func (o *CommentOverlay) Value() string {
	return o.Input.Value()
}

// ShouldFlipAbove returns true if the overlay should appear above the cursor line
// (when cursor is in the bottom half of the viewport).
func (o *CommentOverlay) ShouldFlipAbove(yOffset, visibleRows int) bool {
	relativeRow := o.RowIndex - yOffset
	return relativeRow > visibleRows/2
}

// Render renders the overlay box at the given terminal width.
func (o *CommentOverlay) Render(width int) string {
	overlayWidth := width * 60 / 100
	if overlayWidth < 30 {
		overlayWidth = 30
	}
	if overlayWidth > width {
		overlayWidth = width
	}

	var header string
	if o.EndLine > 0 {
		header = fmt.Sprintf(" Comment on L%d-L%d (%s):", o.Line, o.EndLine, string(o.Side))
	} else {
		header = fmt.Sprintf(" Comment on L%d (%s):", o.Line, string(o.Side))
	}

	// Set input width to overlay width minus border (2) and left padding (1)
	o.Input.Width = overlayWidth - 3
	inputView := " " + o.Input.View()

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBarDim)).
		Width(overlayWidth - 2) // account for border

	content := header + "\n" + inputView
	return style.Render(content)
}
