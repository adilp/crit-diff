package diff

// SideHint indicates which pane a comment is attached to.
type SideHint string

const (
	SideNew SideHint = "new"
	SideOld SideHint = "old"
)

// CommentInfo holds the data needed to display a comment row.
type CommentInfo struct {
	ID   string
	Body string
	Line int
}

// PairedLine represents a single row in the side-by-side view.
// Left and Right are zipped together with nil padding on the shorter side.
type PairedLine struct {
	Left        *DiffLine // nil = blank padding row
	Right       *DiffLine // nil = blank padding row
	IsSeparator bool      // true = hunk separator row (expandable context)
	IsComment   bool      // true = comment display row
	CommentID   string    // comment ID (non-empty for comment rows)
	CommentBody string    // comment body text (non-empty for comment rows)
	HunkIndex   int       // which hunk this line belongs to
}

// BuildPairedLines takes parsed hunks and produces a flat slice of PairedLine
// for side-by-side rendering. Contiguous delete/add groups are zipped together
// with nil padding on the shorter side. Separator rows are inserted between hunks.
func BuildPairedLines(hunks []Hunk) []PairedLine {
	if len(hunks) == 0 {
		return nil
	}

	var pairs []PairedLine

	for i, h := range hunks {
		if i > 0 {
			pairs = append(pairs, PairedLine{
				IsSeparator: true,
				HunkIndex:   i,
			})
		}

		pairs = append(pairs, pairHunkLines(h.Lines, i)...)
	}

	return pairs
}

// pairHunkLines processes a single hunk's lines, grouping contiguous
// deletes and adds together and zipping them with padding.
func pairHunkLines(lines []DiffLine, hunkIndex int) []PairedLine {
	var pairs []PairedLine
	var dels []DiffLine
	var adds []DiffLine

	flush := func() {
		pairs = append(pairs, zipDeletesAdds(dels, adds, hunkIndex)...)
		dels = nil
		adds = nil
	}

	for _, line := range lines {
		switch line.Type {
		case LineDelete:
			if len(adds) > 0 {
				// We had adds accumulating but now see a delete — flush first
				flush()
			}
			dels = append(dels, line)
		case LineAdd:
			adds = append(adds, line)
		case LineContext:
			flush()
			left := line
			right := line
			pairs = append(pairs, PairedLine{
				Left:      &left,
				Right:     &right,
				HunkIndex: hunkIndex,
			})
		}
	}
	flush()

	return pairs
}

// InsertCommentRows inserts comment display rows into paired lines after matching code lines.
// comments maps line numbers to CommentInfo. side determines which side to try first
// (SideNew matches Right.NewNum, SideOld matches Left.OldNum). If the primary side
// doesn't match, the other side is also checked as fallback.
func InsertCommentRows(pairs []PairedLine, comments map[int]CommentInfo, side SideHint) []PairedLine {
	if len(comments) == 0 {
		return pairs
	}

	matched := make(map[string]bool) // track matched comment IDs to avoid duplicates

	var result []PairedLine
	for _, p := range pairs {
		result = append(result, p)

		if p.IsSeparator || p.IsComment {
			continue
		}

		// Try primary side first, then fallback to other side
		lineNums := []int{}
		if side == SideNew {
			if p.Right != nil {
				lineNums = append(lineNums, p.Right.NewNum)
			}
			if p.Left != nil {
				lineNums = append(lineNums, p.Left.OldNum)
			}
		} else {
			if p.Left != nil {
				lineNums = append(lineNums, p.Left.OldNum)
			}
			if p.Right != nil {
				lineNums = append(lineNums, p.Right.NewNum)
			}
		}

		for _, lineNum := range lineNums {
			if lineNum == 0 {
				continue
			}
			if ci, ok := comments[lineNum]; ok && !matched[ci.ID] {
				matched[ci.ID] = true
				result = append(result, PairedLine{
					IsComment:   true,
					CommentID:   ci.ID,
					CommentBody: ci.Body,
					HunkIndex:   p.HunkIndex,
				})
				break
			}
		}
	}
	return result
}

// zipDeletesAdds zips contiguous deletes and adds, padding the shorter side with nil.
func zipDeletesAdds(dels, adds []DiffLine, hunkIndex int) []PairedLine {
	n := len(dels)
	if len(adds) > n {
		n = len(adds)
	}
	if n == 0 {
		return nil
	}

	pairs := make([]PairedLine, n)
	for i := range n {
		p := PairedLine{HunkIndex: hunkIndex}
		if i < len(dels) {
			d := dels[i]
			p.Left = &d
		}
		if i < len(adds) {
			a := adds[i]
			p.Right = &a
		}
		pairs[i] = p
	}
	return pairs
}
