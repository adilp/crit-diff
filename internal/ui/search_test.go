package ui

import (
	"testing"

	"github.com/adilp/crit-diff/internal/diff"
)

func buildSearchPairs() []diff.PairedLine {
	return diff.BuildPairedLines([]diff.Hunk{
		{
			OldStart: 1, OldCount: 5, NewStart: 1, NewCount: 5,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "package main"},
				{Type: diff.LineContext, OldNum: 2, NewNum: 2, Content: "func hello() {"},
				{Type: diff.LineDelete, OldNum: 3, Content: "old hello world"},
				{Type: diff.LineAdd, NewNum: 3, Content: "new Hello World"},
				{Type: diff.LineContext, OldNum: 4, NewNum: 4, Content: "return nil"},
				{Type: diff.LineContext, OldNum: 5, NewNum: 5, Content: "}"},
			},
		},
	})
}

func TestFindMatches(t *testing.T) {
	pairs := buildSearchPairs()

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{"empty query returns no matches", "", 0},
		{"case insensitive match", "hello", 4}, // "func hello" left+right, "old hello world" left, "new Hello World" right
		{"exact substring", "package", 2},      // context line appears on both sides
		{"no matches", "zzzznothere", 0},
		{"matches in delete lines", "old hello", 1},
		{"matches in add lines", "new Hello", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := FindMatches(pairs, tt.query)
			if len(matches) != tt.wantCount {
				t.Errorf("FindMatches(%q): got %d matches, want %d", tt.query, len(matches), tt.wantCount)
				for i, m := range matches {
					t.Logf("  match[%d]: row=%d side=%v col=%d", i, m.Row, m.Side, m.Col)
				}
			}
		})
	}
}

func TestFindMatches_ContextBothSides(t *testing.T) {
	// Context lines appear on both left and right — matches should be found on both sides
	pairs := buildSearchPairs()
	matches := FindMatches(pairs, "return nil")
	// "return nil" is a context line — appears on both left and right
	if len(matches) != 2 {
		t.Errorf("got %d matches for 'return nil', want 2 (left+right)", len(matches))
		for i, m := range matches {
			t.Logf("  match[%d]: row=%d side=%v col=%d", i, m.Row, m.Side, m.Col)
		}
	}
}

func TestFindMatches_MultipleMatchesPerLine(t *testing.T) {
	// If a line has "aa aa", searching for "aa" should find 2 matches on that side
	pairs := diff.BuildPairedLines([]diff.Hunk{
		{
			OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "aa bb aa"},
			},
		},
	})
	matches := FindMatches(pairs, "aa")
	// "aa bb aa" on left side: 2 matches; right side: 2 matches = 4 total
	if len(matches) != 4 {
		t.Errorf("got %d matches, want 4", len(matches))
	}
}

func TestFindMatches_ColumnOffset(t *testing.T) {
	pairs := diff.BuildPairedLines([]diff.Hunk{
		{
			OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "prefix hello suffix"},
			},
		},
	})
	matches := FindMatches(pairs, "hello")
	if len(matches) < 1 {
		t.Fatal("expected at least 1 match")
	}
	// First match should have col = 7 (index of "hello" in "prefix hello suffix")
	if matches[0].Col != 7 {
		t.Errorf("match col: got %d, want 7", matches[0].Col)
	}
}

func TestSearchState_Navigation(t *testing.T) {
	matches := []SearchMatch{
		{Row: 0, Side: SideOld, Col: 0},
		{Row: 2, Side: SideOld, Col: 4},
		{Row: 3, Side: SideNew, Col: 4},
	}

	s := SearchState{
		Active:  true,
		Query:   "hello",
		Matches: matches,
		Current: 0,
	}

	// Next wraps around
	s.Next()
	if s.Current != 1 {
		t.Errorf("after Next: got Current=%d, want 1", s.Current)
	}
	s.Next()
	if s.Current != 2 {
		t.Errorf("after Next: got Current=%d, want 2", s.Current)
	}
	s.Next()
	if s.Current != 0 {
		t.Errorf("after Next wrap: got Current=%d, want 0", s.Current)
	}

	// Prev wraps around
	s.Prev()
	if s.Current != 2 {
		t.Errorf("after Prev wrap: got Current=%d, want 2", s.Current)
	}
	s.Prev()
	if s.Current != 1 {
		t.Errorf("after Prev: got Current=%d, want 1", s.Current)
	}
}

func TestSearchState_NavigationEmpty(t *testing.T) {
	s := SearchState{Active: true, Query: "x", Matches: nil, Current: 0}
	s.Next() // should not panic
	s.Prev() // should not panic
	if s.Current != 0 {
		t.Errorf("Current should stay 0 with no matches, got %d", s.Current)
	}
}

func TestSearchState_FirstMatchAfter(t *testing.T) {
	matches := []SearchMatch{
		{Row: 1, Side: SideOld, Col: 0},
		{Row: 3, Side: SideNew, Col: 0},
		{Row: 5, Side: SideOld, Col: 0},
	}

	tests := []struct {
		name      string
		cursorRow int
		wantIdx   int
	}{
		{"cursor before all matches", 0, 0},
		{"cursor at first match", 1, 0},
		{"cursor between matches", 2, 1},
		{"cursor after all matches wraps", 6, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SearchState{Active: true, Query: "x", Matches: matches, Current: 0}
			idx := s.FirstMatchAfter(tt.cursorRow)
			if idx != tt.wantIdx {
				t.Errorf("FirstMatchAfter(%d): got %d, want %d", tt.cursorRow, idx, tt.wantIdx)
			}
		})
	}
}

func TestBuildSearchMask(t *testing.T) {
	content := "hello world hello"
	mask := BuildSearchMask(content, "hello")

	if len(mask) != len(content) {
		t.Fatalf("mask length: got %d, want %d", len(mask), len(content))
	}

	// Positions 0-4 should be true ("hello")
	for i := 0; i < 5; i++ {
		if !mask[i] {
			t.Errorf("mask[%d] should be true", i)
		}
	}
	// Positions 5-11 should be false (" world ")
	for i := 5; i < 12; i++ {
		if mask[i] {
			t.Errorf("mask[%d] should be false", i)
		}
	}
	// Positions 12-16 should be true ("hello")
	for i := 12; i < 17; i++ {
		if !mask[i] {
			t.Errorf("mask[%d] should be true", i)
		}
	}
}

func TestBuildSearchMask_CaseInsensitive(t *testing.T) {
	content := "Hello HELLO"
	mask := BuildSearchMask(content, "hello")

	// Both "Hello" (0-4) and "HELLO" (6-10) should be masked
	for i := 0; i < 5; i++ {
		if !mask[i] {
			t.Errorf("mask[%d] should be true", i)
		}
	}
	if mask[5] {
		t.Error("mask[5] (space) should be false")
	}
	for i := 6; i < 11; i++ {
		if !mask[i] {
			t.Errorf("mask[%d] should be true", i)
		}
	}
}

func TestBuildSearchMask_EmptyQuery(t *testing.T) {
	mask := BuildSearchMask("hello", "")
	if mask != nil {
		t.Error("empty query should return nil mask")
	}
}
