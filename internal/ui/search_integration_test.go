package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/adil/cr/internal/diff"
)

func newSearchTestModel() Model {
	hunks := []diff.Hunk{
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
	}
	files := []diff.DiffFile{
		{OldName: "main.go", NewName: "main.go", Hunks: hunks},
	}
	pairs := diff.BuildPairedLines(hunks)
	return NewModel(files, pairs, 120, 40)
}

// typeSearch is a helper that enters search mode, types a query, and submits.
func typeSearch(m Model, query string) Model {
	result, _ := m.Update(keyMsg("/"))
	model := result.(Model)
	for _, r := range query {
		result, _ = model.Update(keyMsg(string(r)))
		model = result.(Model)
	}
	result, _ = model.Update(enterKeyMsg())
	model = result.(Model)
	return model
}

func TestSearchMode_SlashOpensInput(t *testing.T) {
	m := newSearchTestModel()

	result, _ := m.Update(keyMsg("/"))
	model := result.(Model)

	if model.mode != InputModeSearch {
		t.Errorf("mode: got %v, want InputModeSearch", model.mode)
	}
	if !model.search.Input.Focused() {
		t.Error("search input should be focused")
	}
}

func TestSearchMode_EscCancels(t *testing.T) {
	m := newSearchTestModel()

	result, _ := m.Update(keyMsg("/"))
	model := result.(Model)

	result, _ = model.Update(escKeyMsg())
	model = result.(Model)

	if model.mode != InputModeNormal {
		t.Errorf("mode: got %v, want InputModeNormal", model.mode)
	}
	if model.search.Active {
		t.Error("search should not be active after Esc")
	}
	if model.search.Query != "" {
		t.Errorf("query should be cleared, got %q", model.search.Query)
	}
}

func TestSearchMode_EnterSubmits(t *testing.T) {
	m := newSearchTestModel()
	model := typeSearch(m, "hello")

	if model.mode != InputModeNormal {
		t.Errorf("mode: got %v, want InputModeNormal", model.mode)
	}
	if !model.search.Active {
		t.Error("search should be active after Enter (highlights visible)")
	}
	if model.search.Query != "hello" {
		t.Errorf("query: got %q, want %q", model.search.Query, "hello")
	}
	if len(model.search.Matches) != 4 {
		t.Errorf("matches: got %d, want 4", len(model.search.Matches))
	}
}

func TestSearchMode_EnterJumpsToFirstMatch(t *testing.T) {
	m := newSearchTestModel()
	m.cursorRow = 4

	model := typeSearch(m, "package")

	if model.cursorRow != 0 {
		t.Errorf("cursorRow: got %d, want 0", model.cursorRow)
	}
}

func TestSearchMode_NNextMatch(t *testing.T) {
	m := newSearchTestModel()
	model := typeSearch(m, "hello")

	// After Enter, Current should be 0 (first match)
	if model.search.Current != 0 {
		t.Fatalf("initial Current: got %d, want 0", model.search.Current)
	}

	// Press n — should advance to match 1
	result, _ := model.Update(keyMsg("n"))
	model = result.(Model)

	if model.search.Current != 1 {
		t.Errorf("after n: got Current=%d, want 1", model.search.Current)
	}
}

func TestSearchMode_NPrevMatch(t *testing.T) {
	m := newSearchTestModel()
	model := typeSearch(m, "hello")

	// Press N — should wrap to last match
	result, _ := model.Update(keyMsg("N"))
	model = result.(Model)

	wantCurrent := len(model.search.Matches) - 1
	if model.search.Current != wantCurrent {
		t.Errorf("after N: got Current=%d, want %d", model.search.Current, wantCurrent)
	}
}

func TestSearchMode_EscClearsActiveSearch(t *testing.T) {
	m := newSearchTestModel()
	model := typeSearch(m, "hello")

	if !model.search.Active {
		t.Fatal("search should be active after Enter")
	}

	result, _ := model.Update(escKeyMsg())
	model = result.(Model)

	if model.search.Active {
		t.Error("Esc in normal mode should clear active search")
	}
}

func TestSearchMode_NoMatches(t *testing.T) {
	m := newSearchTestModel()
	model := typeSearch(m, "zzzzz")

	if len(model.search.Matches) != 0 {
		t.Errorf("should have 0 matches, got %d", len(model.search.Matches))
	}
	if !model.search.Active {
		t.Error("search should remain active even with no matches")
	}
}

func TestSearchMode_HelpBar(t *testing.T) {
	bar := RenderHelpBar(120, InputModeSearch)
	if !strings.Contains(bar, "Enter") {
		t.Error("search mode help bar should contain Enter")
	}
	if !strings.Contains(bar, "Esc") {
		t.Error("search mode help bar should contain Esc")
	}
}

func TestSearchMode_RenderSearchPrompt(t *testing.T) {
	m := newSearchTestModel()

	result, _ := m.Update(keyMsg("/"))
	model := result.(Model)

	view := model.View()
	if !strings.Contains(view, "/") {
		t.Error("view should contain search prompt with /")
	}
}

func TestSearchMode_MatchCountDisplay(t *testing.T) {
	m := newSearchTestModel()
	model := typeSearch(m, "hello")

	view := model.View()
	matchCount := len(model.search.Matches)
	if matchCount == 0 {
		t.Fatal("should have matches for this test")
	}

	// View should show the match count indicator [1/N]
	expected := fmt.Sprintf("[1/%d]", matchCount)
	if !strings.Contains(view, expected) {
		t.Errorf("view should contain match count %q", expected)
	}
	// View should also show the query
	if !strings.Contains(view, "/hello") {
		t.Error("view should show /hello in search info bar")
	}
}

func TestSearchMode_NoMatchesDisplay(t *testing.T) {
	m := newSearchTestModel()
	model := typeSearch(m, "zzzzz")

	view := model.View()
	if !strings.Contains(view, "no matches") {
		t.Error("view should show 'no matches' indicator")
	}
}

func TestSearchMode_RealTimeMatching(t *testing.T) {
	m := newSearchTestModel()

	// Enter search mode
	result, _ := m.Update(keyMsg("/"))
	model := result.(Model)

	// Type "hello" — matches should update in real-time
	for _, r := range "hello" {
		result, _ = model.Update(keyMsg(string(r)))
		model = result.(Model)
	}

	// Before pressing Enter, matches should already be computed
	if len(model.search.Matches) == 0 {
		t.Error("matches should be computed in real-time while typing")
	}
	if model.search.Query != "hello" {
		t.Errorf("query should update in real-time, got %q", model.search.Query)
	}
}

func TestSearchMode_FileSwitchRecomputesMatches(t *testing.T) {
	// Create a model with two files
	hunks1 := []diff.Hunk{{
		OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
		Lines: []diff.DiffLine{
			{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "alpha beta"},
		},
	}}
	hunks2 := []diff.Hunk{{
		OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
		Lines: []diff.DiffLine{
			{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "gamma delta"},
		},
	}}
	files := []diff.DiffFile{
		{OldName: "a.go", NewName: "a.go", Hunks: hunks1},
		{OldName: "b.go", NewName: "b.go", Hunks: hunks2},
	}
	pairs1 := diff.BuildPairedLines(hunks1)
	pairs2 := diff.BuildPairedLines(hunks2)
	m := NewModel(files, pairs1, 120, 40)
	m.allPaired = [][]diff.PairedLine{pairs1, pairs2}

	// Search for "alpha" — should find matches in file 1
	model := typeSearch(m, "alpha")
	if len(model.search.Matches) == 0 {
		t.Fatal("should have matches for 'alpha' in file 1")
	}

	// Switch to file 2 via ]f
	result, _ := model.Update(keyMsg("]"))
	model = result.(Model)
	result, _ = model.Update(keyMsg("f"))
	model = result.(Model)

	// After file switch, matches should be recomputed for new file
	// "alpha" should not match in file 2 (which has "gamma delta")
	if len(model.search.Matches) != 0 {
		t.Errorf("after file switch, should have 0 matches for 'alpha' in file 2, got %d",
			len(model.search.Matches))
	}
}

func TestSearchMode_NWorksOnlyWhenSearchActive(t *testing.T) {
	m := newSearchTestModel()
	startRow := m.cursorRow

	result, _ := m.Update(keyMsg("n"))
	model := result.(Model)

	if model.cursorRow != startRow {
		t.Errorf("n without active search should not change cursor, got %d want %d",
			model.cursorRow, startRow)
	}
}
