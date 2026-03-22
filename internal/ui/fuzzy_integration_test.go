package ui

import (
	"strings"
	"testing"

	"github.com/adil/cr/internal/diff"
	tea "github.com/charmbracelet/bubbletea"
)

func newFuzzyTestModel() Model {
	files := []diff.DiffFile{
		{OldName: "src/auth.go", NewName: "src/auth.go",
			Hunks: []diff.Hunk{{
				OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
				Lines: []diff.DiffLine{
					{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "package auth"},
					{Type: diff.LineAdd, NewNum: 2, Content: "func Login()"},
				},
			}},
		},
		{OldName: "src/helper.go", NewName: "src/helper.go",
			Hunks: []diff.Hunk{{
				OldStart: 5, OldCount: 2, NewStart: 5, NewCount: 2,
				Lines: []diff.DiffLine{
					{Type: diff.LineContext, OldNum: 5, NewNum: 5, Content: "package helper"},
					{Type: diff.LineDelete, OldNum: 6, Content: "old helper"},
				},
			}},
		},
	}
	allPaired := make([][]diff.PairedLine, len(files))
	for i, f := range files {
		allPaired[i] = diff.BuildPairedLines(f.Hunks)
	}
	m := NewModel(files, allPaired[0], 120, 40)
	m.allPaired = allPaired
	return m
}

func TestFuzzyFilesOpensOverlay(t *testing.T) {
	m := newFuzzyTestModel()

	// Simulate Space f (fzf not in path — uses built-in)
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	if m.mode != InputModeFuzzy {
		t.Errorf("mode: got %v, want InputModeFuzzy", m.mode)
	}
	if m.fuzzy.Mode != FuzzyModeFiles {
		t.Errorf("fuzzy.Mode: got %v, want FuzzyModeFiles", m.fuzzy.Mode)
	}
	if len(m.fuzzy.Items) != 2 {
		t.Errorf("fuzzy items: got %d, want 2", len(m.fuzzy.Items))
	}
}

func TestFuzzySearchDiffsOpensOverlay(t *testing.T) {
	m := newFuzzyTestModel()

	// Simulate Space s
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("s"))
	m = newM.(Model)

	if m.mode != InputModeFuzzy {
		t.Errorf("mode: got %v, want InputModeFuzzy", m.mode)
	}
	if m.fuzzy.Mode != FuzzyModeContent {
		t.Errorf("fuzzy.Mode: got %v, want FuzzyModeContent", m.fuzzy.Mode)
	}
	// 2 files x 2 lines each = 4 content lines
	if len(m.fuzzy.Items) != 4 {
		t.Errorf("fuzzy items: got %d, want 4", len(m.fuzzy.Items))
	}
}

func TestFuzzyEscCancels(t *testing.T) {
	m := newFuzzyTestModel()

	// Open fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	if m.mode != InputModeFuzzy {
		t.Fatal("should be in fuzzy mode")
	}

	// Press Esc
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newM.(Model)

	if m.mode != InputModeNormal {
		t.Errorf("mode after Esc: got %v, want InputModeNormal", m.mode)
	}
}

func TestFuzzyFileSelectionNavigates(t *testing.T) {
	m := newFuzzyTestModel()

	// Open file fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	// Move to second file with arrow key
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(Model)

	// Select it with Enter
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.mode != InputModeNormal {
		t.Errorf("mode after Enter: got %v, want InputModeNormal", m.mode)
	}
	if m.activeFile != 1 {
		t.Errorf("activeFile: got %d, want 1", m.activeFile)
	}
}

func TestFuzzyArrowNavigation(t *testing.T) {
	m := newFuzzyTestModel()

	// Open file fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	if m.fuzzy.Cursor != 0 {
		t.Fatalf("initial cursor: got %d, want 0", m.fuzzy.Cursor)
	}

	// Down arrow moves down
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(Model)
	if m.fuzzy.Cursor != 1 {
		t.Errorf("after Down: got %d, want 1", m.fuzzy.Cursor)
	}

	// Up arrow moves up
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newM.(Model)
	if m.fuzzy.Cursor != 0 {
		t.Errorf("after Up: got %d, want 0", m.fuzzy.Cursor)
	}
}

func TestFuzzyCtrlJKNavigation(t *testing.T) {
	m := newFuzzyTestModel()

	// Open file fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	// Ctrl-J moves down
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m = newM.(Model)
	if m.fuzzy.Cursor != 1 {
		t.Errorf("after Ctrl-J: got %d, want 1", m.fuzzy.Cursor)
	}

	// Ctrl-K moves up
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = newM.(Model)
	if m.fuzzy.Cursor != 0 {
		t.Errorf("after Ctrl-K: got %d, want 0", m.fuzzy.Cursor)
	}
}

func TestFuzzyLiveFiltering(t *testing.T) {
	m := newFuzzyTestModel()

	// Open file fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	if len(m.fuzzy.Filtered) != 2 {
		t.Fatalf("initial filtered: got %d, want 2", len(m.fuzzy.Filtered))
	}

	// Type "auth" to filter - use rune input
	for _, ch := range "auth" {
		newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = newM.(Model)
	}

	if len(m.fuzzy.Filtered) != 1 {
		t.Errorf("after typing 'auth': got %d filtered, want 1", len(m.fuzzy.Filtered))
	}
}

func TestFuzzyTypingJKGoesToInput(t *testing.T) {
	m := newFuzzyTestModel()

	// Open file fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	// Type "j" — should go to input, not navigate
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newM.(Model)

	if m.fuzzy.Input.Value() != "j" {
		t.Errorf("input value: got %q, want %q", m.fuzzy.Input.Value(), "j")
	}
	// Cursor should remain at 0 (not navigated)
	if m.fuzzy.Cursor != 0 {
		t.Errorf("cursor should be 0 after typing j, got %d", m.fuzzy.Cursor)
	}
}

func TestFuzzyHelpBar(t *testing.T) {
	m := newFuzzyTestModel()

	// Open fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	helpBar := RenderHelpBar(120, m.mode)
	if !strings.Contains(helpBar, "↑/↓") {
		t.Error("help bar should contain ↑/↓ hint")
	}
	if !strings.Contains(helpBar, "Enter") {
		t.Error("help bar should contain Enter hint")
	}
	if !strings.Contains(helpBar, "Esc") {
		t.Error("help bar should contain Esc hint")
	}
}

func TestFuzzyViewRendering(t *testing.T) {
	m := newFuzzyTestModel()

	// Open fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	view := m.View()
	if !strings.Contains(view, ">") {
		t.Error("view should contain prompt '>'")
	}
}

func TestFuzzyContentSelectionNavigates(t *testing.T) {
	m := newFuzzyTestModel()

	// Open content fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("s"))
	m = newM.(Model)

	// Navigate to a line in the second file (helper.go lines start at index 2)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(Model)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(Model)

	// Select it
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.mode != InputModeNormal {
		t.Errorf("mode: got %v, want InputModeNormal", m.mode)
	}
	// Should have navigated to second file (index 1)
	if m.activeFile != 1 {
		t.Errorf("activeFile: got %d, want 1", m.activeFile)
	}
}

func TestFuzzyCtrlCQuits(t *testing.T) {
	m := newFuzzyTestModel()

	// Open fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	// Ctrl-C should quit the whole app
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Ctrl-C in fuzzy mode should return quit command")
	}
}

func TestFuzzyScrollTracking(t *testing.T) {
	// Create a model with many files to test scroll
	files := make([]diff.DiffFile, 50)
	for i := range files {
		name := strings.Repeat("a", i+1) + ".go"
		files[i] = diff.DiffFile{OldName: name, NewName: name,
			Hunks: []diff.Hunk{{
				OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
				Lines: []diff.DiffLine{
					{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "ctx"},
				},
			}},
		}
	}
	allPaired := make([][]diff.PairedLine, len(files))
	for i, f := range files {
		allPaired[i] = diff.BuildPairedLines(f.Hunks)
	}
	m := NewModel(files, allPaired[0], 120, 10) // small height
	m.allPaired = allPaired

	// Open fuzzy
	newM, _ := m.Update(keyMsg("Space"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("f"))
	m = newM.(Model)

	// Navigate down past visible area
	for i := 0; i < 8; i++ {
		newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = newM.(Model)
	}

	if m.fuzzy.Cursor != 8 {
		t.Errorf("cursor: got %d, want 8", m.fuzzy.Cursor)
	}
	// YOffset should have scrolled to keep cursor visible
	if m.fuzzy.YOffset == 0 {
		t.Error("YOffset should have scrolled away from 0")
	}
}
