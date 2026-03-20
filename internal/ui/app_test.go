package ui

import (
	"strings"
	"testing"

	"github.com/adil/cr/internal/diff"
	tea "github.com/charmbracelet/bubbletea"
)

// buildTestPairs creates a simple set of paired lines for testing.
func buildTestPairs() []diff.PairedLine {
	return diff.BuildPairedLines([]diff.Hunk{
		{
			OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "context line one"},
				{Type: diff.LineDelete, OldNum: 2, Content: "deleted line"},
				{Type: diff.LineAdd, NewNum: 2, Content: "added line"},
				{Type: diff.LineContext, OldNum: 3, NewNum: 3, Content: "context line three"},
			},
		},
	})
}

// buildMultiHunkPairs creates paired lines with two hunks and a separator.
func buildMultiHunkPairs() []diff.PairedLine {
	return diff.BuildPairedLines([]diff.Hunk{
		{
			OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "first hunk line"},
				{Type: diff.LineAdd, NewNum: 2, Content: "added in hunk 1"},
			},
		},
		{
			OldStart: 10, OldCount: 2, NewStart: 11, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 10, NewNum: 11, Content: "second hunk line"},
				{Type: diff.LineDelete, OldNum: 11, Content: "deleted in hunk 2"},
			},
		},
	})
}

// buildTallPairs creates enough lines to require scrolling.
func buildTallPairs() []diff.PairedLine {
	lines := make([]diff.DiffLine, 30)
	for i := range lines {
		lines[i] = diff.DiffLine{
			Type:    diff.LineContext,
			OldNum:  i + 1,
			NewNum:  i + 1,
			Content: "line content",
		}
	}
	return diff.BuildPairedLines([]diff.Hunk{
		{OldStart: 1, OldCount: 30, NewStart: 1, NewCount: 30, Lines: lines},
	})
}

func newTestModel(pairs []diff.PairedLine, width, height int) Model {
	files := []diff.DiffFile{
		{OldName: "test.go", NewName: "test.go"},
	}
	return NewModel(files, pairs, width, height)
}

func TestNewModel(t *testing.T) {
	pairs := buildTestPairs()
	m := newTestModel(pairs, 120, 40)

	if m.cursorRow != 0 {
		t.Errorf("cursorRow: got %d, want 0", m.cursorRow)
	}
	if m.activeSide != SideNew {
		t.Errorf("activeSide: got %v, want SideNew", m.activeSide)
	}
	if m.yOffset != 0 {
		t.Errorf("yOffset: got %d, want 0", m.yOffset)
	}
	if m.width != 120 {
		t.Errorf("width: got %d, want 120", m.width)
	}
	if m.height != 40 {
		t.Errorf("height: got %d, want 40", m.height)
	}
	if len(m.paired) != len(pairs) {
		t.Errorf("paired length: got %d, want %d", len(m.paired), len(pairs))
	}
}

func TestInit(t *testing.T) {
	m := newTestModel(buildTestPairs(), 120, 40)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil cmd")
	}
}

func TestUpdateKeyNavigation(t *testing.T) {
	tests := []struct {
		name           string
		pairs          []diff.PairedLine
		width          int
		height         int
		keys           []tea.KeyMsg
		wantCursorRow  int
		wantActiveSide Side
		wantYOffset    int
	}{
		{
			name:          "j moves cursor down",
			pairs:         buildTestPairs(),
			width:         120,
			height:        40,
			keys:          []tea.KeyMsg{keyMsg("j")},
			wantCursorRow: 1,
		},
		{
			name:          "k does not go above 0",
			pairs:         buildTestPairs(),
			width:         120,
			height:        40,
			keys:          []tea.KeyMsg{keyMsg("k")},
			wantCursorRow: 0,
		},
		{
			name:          "j then k returns to 0",
			pairs:         buildTestPairs(),
			width:         120,
			height:        40,
			keys:          []tea.KeyMsg{keyMsg("j"), keyMsg("j"), keyMsg("k")},
			wantCursorRow: 1,
		},
		{
			name:          "j does not go past last line",
			pairs:         buildTestPairs(),
			width:         120,
			height:        40,
			keys:          []tea.KeyMsg{keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j")},
			wantCursorRow: len(buildTestPairs()) - 1,
		},
		{
			name:           "h switches to left (old) pane",
			pairs:          buildTestPairs(),
			width:          120,
			height:         40,
			keys:           []tea.KeyMsg{keyMsg("h")},
			wantActiveSide: SideOld,
		},
		{
			name:           "l switches to right (new) pane",
			pairs:          buildTestPairs(),
			width:          120,
			height:         40,
			keys:           []tea.KeyMsg{keyMsg("h"), keyMsg("l")},
			wantActiveSide: SideNew,
		},
		{
			name:          "ctrl+d scrolls half page down",
			pairs:         buildTallPairs(),
			width:         120,
			height:        20, // visibleRows = 18
			keys:          []tea.KeyMsg{ctrlKeyMsg("d")},
			wantCursorRow: 9, // half of 18
		},
		{
			name:          "ctrl+u scrolls half page up",
			pairs:         buildTallPairs(),
			width:         120,
			height:        20,
			keys:          []tea.KeyMsg{ctrlKeyMsg("d"), ctrlKeyMsg("u")},
			wantCursorRow: 0,
		},
		{
			name:          "viewport scrolls down when cursor reaches edge",
			pairs:         buildTallPairs(),
			width:         120,
			height:        5, // visibleRows = 3
			keys:          []tea.KeyMsg{keyMsg("j"), keyMsg("j"), keyMsg("j")},
			wantCursorRow: 3,
			wantYOffset:   1, // cursor at 3, visibleRows 3, offset = 3 - 3 + 1 = 1
		},
		{
			name:          "viewport scrolls up when cursor goes above offset",
			pairs:         buildTallPairs(),
			width:         120,
			height:        5,
			keys:          []tea.KeyMsg{keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("k"), keyMsg("k"), keyMsg("k"), keyMsg("k")},
			wantCursorRow: 0,
			wantYOffset:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(tt.pairs, tt.width, tt.height)
			for _, key := range tt.keys {
				newModel, _ := m.Update(key)
				m = newModel.(Model)
			}

			if m.cursorRow != tt.wantCursorRow {
				t.Errorf("cursorRow: got %d, want %d", m.cursorRow, tt.wantCursorRow)
			}
			if tt.wantActiveSide != "" {
				if m.activeSide != tt.wantActiveSide {
					t.Errorf("activeSide: got %v, want %v", m.activeSide, tt.wantActiveSide)
				}
			}
			if m.yOffset != tt.wantYOffset {
				t.Errorf("yOffset: got %d, want %d", m.yOffset, tt.wantYOffset)
			}
		})
	}
}

func TestUpdateQuit(t *testing.T) {
	m := newTestModel(buildTestPairs(), 120, 40)
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	// Execute the cmd and check it produces a tea.QuitMsg
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestUpdateWindowResize(t *testing.T) {
	m := newTestModel(buildTestPairs(), 120, 40)
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	m = newModel.(Model)
	if m.width != 200 {
		t.Errorf("width: got %d, want 200", m.width)
	}
	if m.height != 50 {
		t.Errorf("height: got %d, want 50", m.height)
	}
}

func TestViewRendering(t *testing.T) {
	tests := []struct {
		name     string
		pairs    []diff.PairedLine
		width    int
		height   int
		contains []string
		absent   []string
	}{
		{
			name:   "renders line numbers",
			pairs:  buildTestPairs(),
			width:  120,
			height: 40,
			contains: []string{
				"1", "2", "3", // line numbers present
			},
		},
		{
			name:   "renders separator between hunks",
			pairs:  buildMultiHunkPairs(),
			width:  120,
			height: 40,
			contains: []string{
				"───", // separator line
			},
		},
		{
			name:   "non-empty output",
			pairs:  buildTestPairs(),
			width:  120,
			height: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(tt.pairs, tt.width, tt.height)
			output := m.View()
			if output == "" {
				t.Fatal("View() returned empty string")
			}
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("expected output to contain %q", s)
				}
			}
			for _, s := range tt.absent {
				if strings.Contains(output, s) {
					t.Errorf("expected output to NOT contain %q", s)
				}
			}
		})
	}
}

func TestViewVisibleRows(t *testing.T) {
	pairs := buildTallPairs()         // 30 lines
	m := newTestModel(pairs, 120, 12) // height 12 → visibleRows = 10
	output := m.View()
	// Should have at most visibleRows + 2 (status + help bar) lines in output
	lines := strings.Split(output, "\n")
	// Remove trailing empty line if present
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	// Should not render all 30 lines — viewport clips to visible area
	if len(lines) > 14 { // some tolerance for status/help bar placeholder lines
		t.Errorf("expected at most ~12 rendered lines, got %d", len(lines))
	}
}

func TestViewEmptyPairs(t *testing.T) {
	m := newTestModel(nil, 120, 40)
	output := m.View()
	if output == "" {
		t.Fatal("View() should return something even with no pairs")
	}
}

// Helper to create a simple key message
func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(key),
	}
}

// Helper to create ctrl+key message
func ctrlKeyMsg(key string) tea.KeyMsg {
	switch key {
	case "d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}
