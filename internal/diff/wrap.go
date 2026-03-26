package diff

import "github.com/charmbracelet/x/ansi"

// VisualRowCount returns how many visual rows a line occupies at the given pane width.
// Uses ANSI-aware string width to handle styled content correctly.
func VisualRowCount(content string, paneWidth int) int {
	if paneWidth <= 0 {
		return 1
	}
	displayWidth := ansi.StringWidth(content)
	if displayWidth <= paneWidth {
		return 1
	}
	return (displayWidth + paneWidth - 1) / paneWidth
}

// BuildWrappedPairedLines takes parsed hunks and a pane width, producing a flat slice
// of PairedLine with long lines expanded into multiple visual rows.
// For each logical paired line, both sides expand to max(leftRows, rightRows) visual rows,
// with nil padding on the shorter side's extra rows.
func BuildWrappedPairedLines(hunks []Hunk, paneWidth int) []PairedLine {
	logical := BuildPairedLines(hunks)
	if len(logical) == 0 {
		return nil
	}

	// Content width excludes the line number column (5 chars: "1234│")
	contentWidth := paneWidth - 5
	if contentWidth < 1 {
		contentWidth = 1
	}

	var result []PairedLine
	for _, p := range logical {
		if p.IsSeparator || p.IsComment {
			result = append(result, p)
			continue
		}

		leftRows := 1
		rightRows := 1
		if p.Left != nil {
			leftRows = VisualRowCount(p.Left.Content, contentWidth)
		}
		if p.Right != nil {
			rightRows = VisualRowCount(p.Right.Content, contentWidth)
		}

		maxRows := leftRows
		if rightRows > maxRows {
			maxRows = rightRows
		}

		// First row keeps the original paired line data
		result = append(result, p)

		// Additional rows are wrap continuation rows
		for i := 1; i < maxRows; i++ {
			result = append(result, PairedLine{
				IsWrapContinuation: true,
				WrapSourceLeft:     p.Left,
				WrapSourceRight:    p.Right,
				WrapRow:            i,
				HunkIndex:          p.HunkIndex,
			})
		}
	}

	return result
}
