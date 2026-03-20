package diff

// PairedLine represents a single row in the side-by-side view.
// Left and Right are zipped together with nil padding on the shorter side.
type PairedLine struct {
	Left        *DiffLine // nil = blank padding row
	Right       *DiffLine // nil = blank padding row
	IsSeparator bool      // true = hunk separator row (expandable context)
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
