package ui

import (
	"strings"

	"github.com/adil/cr/internal/diff"
	"github.com/charmbracelet/bubbles/textinput"
)

// SearchMatch represents a single match location in the paired lines.
type SearchMatch struct {
	Row  int  // index in paired lines
	Side Side // which pane the match is in
	Col  int  // character offset in the line content
}

// SearchState holds the state for in-file search.
type SearchState struct {
	Active  bool
	Query   string
	Input   textinput.Model
	Matches []SearchMatch
	Current int // index into Matches
}

// Next advances to the next match, wrapping to the beginning.
func (s *SearchState) Next() {
	if len(s.Matches) == 0 {
		return
	}
	s.Current = (s.Current + 1) % len(s.Matches)
}

// Prev moves to the previous match, wrapping to the end.
func (s *SearchState) Prev() {
	if len(s.Matches) == 0 {
		return
	}
	s.Current--
	if s.Current < 0 {
		s.Current = len(s.Matches) - 1
	}
}

// FirstMatchAfter returns the index of the first match at or after cursorRow.
// If no match is found after cursorRow, wraps to the first match (index 0).
func (s *SearchState) FirstMatchAfter(cursorRow int) int {
	for i, m := range s.Matches {
		if m.Row >= cursorRow {
			return i
		}
	}
	return 0
}

// FindMatches finds all occurrences of query (case-insensitive) in the paired lines.
// Searches both Left and Right content of each pair.
func FindMatches(pairs []diff.PairedLine, query string) []SearchMatch {
	if query == "" {
		return nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []SearchMatch

	for i, p := range pairs {
		if p.IsSeparator || p.IsComment {
			continue
		}

		// Search left side
		if p.Left != nil {
			matches = append(matches, findInContent(i, SideOld, p.Left.Content, lowerQuery)...)
		}

		// Search right side
		if p.Right != nil {
			matches = append(matches, findInContent(i, SideNew, p.Right.Content, lowerQuery)...)
		}
	}

	return matches
}

// findInContent finds all occurrences of lowerQuery in content (case-insensitive).
func findInContent(row int, side Side, content, lowerQuery string) []SearchMatch {
	lowerContent := strings.ToLower(content)
	var matches []SearchMatch
	start := 0
	for {
		idx := strings.Index(lowerContent[start:], lowerQuery)
		if idx < 0 {
			break
		}
		matches = append(matches, SearchMatch{
			Row:  row,
			Side: side,
			Col:  start + idx,
		})
		start += idx + len(lowerQuery)
	}
	return matches
}

// BuildSearchMask builds a boolean mask for highlighting search matches in a line.
// Returns nil for empty query. Matching is case-insensitive.
func BuildSearchMask(content, query string) []bool {
	if query == "" {
		return nil
	}

	mask := make([]bool, len(content))
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	start := 0
	for {
		idx := strings.Index(lowerContent[start:], lowerQuery)
		if idx < 0 {
			break
		}
		for j := 0; j < len(lowerQuery); j++ {
			mask[start+idx+j] = true
		}
		start += idx + len(lowerQuery)
	}

	return mask
}
