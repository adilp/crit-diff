package ui

import (
	"strings"
	"testing"

	"github.com/adil/cr/internal/comment"
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

func TestSetHighlighting(t *testing.T) {
	pairs := buildTestPairs()
	m := newTestModel(pairs, 120, 40)

	if len(m.oldHighlighted) != 0 {
		t.Error("oldHighlighted should be empty initially")
	}
	if len(m.newHighlighted) != 0 {
		t.Error("newHighlighted should be empty initially")
	}

	// Set highlighting with Go-like content
	m.SetHighlighting("test.go", "context line one\ndeleted line\ncontext line three\n", "context line one\nadded line\ncontext line three\n")

	if len(m.oldHighlighted) == 0 {
		t.Error("oldHighlighted should not be empty after SetHighlighting")
	}
	if len(m.newHighlighted) == 0 {
		t.Error("newHighlighted should not be empty after SetHighlighting")
	}

	// View should still work (no crashes)
	output := m.View()
	if output == "" {
		t.Error("View() should return non-empty output with highlighting")
	}
}

func TestViewWithHighlightingAndWordDiff(t *testing.T) {
	pairs := buildTestPairs()
	m := newTestModel(pairs, 120, 40)
	m.SetHighlighting("test.go", "context line one\ndeleted line\ncontext line three\n", "context line one\nadded line\ncontext line three\n")

	// View without word diff
	output1 := m.View()

	// Toggle word diff
	newModel, _ := m.Update(keyMsg("w"))
	m = newModel.(Model)
	m.SetHighlighting("test.go", "context line one\ndeleted line\ncontext line three\n", "context line one\nadded line\ncontext line three\n")

	// View with word diff
	output2 := m.View()

	// Both should produce non-empty output
	if output1 == "" || output2 == "" {
		t.Error("View() should return non-empty output")
	}

	// Content text should still be present
	if !strings.Contains(output1, "context line one") {
		t.Error("output should contain line content")
	}
}

func TestWordDiffToggle(t *testing.T) {
	m := newTestModel(buildTestPairs(), 120, 40)

	// Initially wordDiff should be false
	if m.wordDiff {
		t.Error("wordDiff should be false initially")
	}

	// Press 'w' to toggle on
	newModel, _ := m.Update(keyMsg("w"))
	m = newModel.(Model)
	if !m.wordDiff {
		t.Error("wordDiff should be true after pressing 'w'")
	}

	// Press 'w' again to toggle off
	newModel, _ = m.Update(keyMsg("w"))
	m = newModel.(Model)
	if m.wordDiff {
		t.Error("wordDiff should be false after pressing 'w' again")
	}
}

// buildMultiFilePairs creates paired lines for two files (used for ]f/[f tests).
func buildMultiFilePairs() ([]diff.DiffFile, [][]diff.PairedLine) {
	files := []diff.DiffFile{
		{OldName: "file1.go", NewName: "file1.go",
			Hunks: []diff.Hunk{{
				OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
				Lines: []diff.DiffLine{
					{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "file1 line1"},
					{Type: diff.LineAdd, NewNum: 2, Content: "file1 added"},
				},
			}},
		},
		{OldName: "file2.go", NewName: "file2.go",
			Hunks: []diff.Hunk{{
				OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
				Lines: []diff.DiffLine{
					{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "file2 line1"},
					{Type: diff.LineDelete, OldNum: 2, Content: "file2 deleted"},
				},
			}},
		},
	}
	allPaired := make([][]diff.PairedLine, len(files))
	for i, f := range files {
		allPaired[i] = diff.BuildPairedLines(f.Hunks)
	}
	return files, allPaired
}

func newMultiFileTestModel(width, height int) Model {
	files, allPaired := buildMultiFilePairs()
	m := NewModel(files, allPaired[0], width, height)
	m.allPaired = allPaired
	return m
}

func TestKeySequenceNavigation(t *testing.T) {
	tests := []struct {
		name            string
		setupModel      func() Model
		keys            []tea.KeyMsg
		wantCursorRow   int
		checkActiveFile bool
		wantActiveFile  int
		wantPendingKey  string
	}{
		{
			name: "gg moves to top",
			setupModel: func() Model {
				m := newTestModel(buildTallPairs(), 120, 40)
				// Move cursor down first
				for i := 0; i < 15; i++ {
					newM, _ := m.Update(keyMsg("j"))
					m = newM.(Model)
				}
				return m
			},
			keys:          []tea.KeyMsg{keyMsg("g"), keyMsg("g")},
			wantCursorRow: 0,
		},
		{
			name: "G moves to bottom",
			setupModel: func() Model {
				return newTestModel(buildTallPairs(), 120, 40)
			},
			keys:          []tea.KeyMsg{keyMsg("G")},
			wantCursorRow: 29, // buildTallPairs creates 30 lines (0-indexed)
		},
		{
			name: "]c jumps to next hunk",
			setupModel: func() Model {
				return newTestModel(buildMultiHunkPairs(), 120, 40)
			},
			keys:          []tea.KeyMsg{keyMsg("]"), keyMsg("c")},
			wantCursorRow: 3, // separator at index 2, first line of hunk 2 at index 3
		},
		{
			name: "[c jumps to prev hunk",
			setupModel: func() Model {
				m := newTestModel(buildMultiHunkPairs(), 120, 40)
				m.cursorRow = 4 // in hunk 2
				return m
			},
			keys:          []tea.KeyMsg{keyMsg("["), keyMsg("c")},
			wantCursorRow: 0, // first line of hunk 1
		},
		{
			name: "]c at last hunk does nothing",
			setupModel: func() Model {
				m := newTestModel(buildMultiHunkPairs(), 120, 40)
				m.cursorRow = 4 // in last hunk
				return m
			},
			keys:          []tea.KeyMsg{keyMsg("]"), keyMsg("c")},
			wantCursorRow: 4,
		},
		{
			name: "[c at first hunk does nothing",
			setupModel: func() Model {
				return newTestModel(buildMultiHunkPairs(), 120, 40)
			},
			keys:          []tea.KeyMsg{keyMsg("["), keyMsg("c")},
			wantCursorRow: 0,
		},
		{
			name: "]f switches to next file",
			setupModel: func() Model {
				return newMultiFileTestModel(120, 40)
			},
			keys:            []tea.KeyMsg{keyMsg("]"), keyMsg("f")},
			checkActiveFile: true,
			wantActiveFile:  1,
			wantCursorRow:   0,
		},
		{
			name: "[f switches to prev file",
			setupModel: func() Model {
				m := newMultiFileTestModel(120, 40)
				// Move to file 2 first
				newM, _ := m.Update(keyMsg("]"))
				m = newM.(Model)
				newM, _ = m.Update(keyMsg("f"))
				m = newM.(Model)
				return m
			},
			keys:            []tea.KeyMsg{keyMsg("["), keyMsg("f")},
			checkActiveFile: true,
			wantActiveFile:  0,
			wantCursorRow:   0,
		},
		{
			name: "]f at last file does nothing",
			setupModel: func() Model {
				m := newMultiFileTestModel(120, 40)
				// Move to last file
				newM, _ := m.Update(keyMsg("]"))
				m = newM.(Model)
				newM, _ = m.Update(keyMsg("f"))
				m = newM.(Model)
				return m
			},
			keys:            []tea.KeyMsg{keyMsg("]"), keyMsg("f")},
			checkActiveFile: true,
			wantActiveFile:  1,
		},
		{
			name: "[f at first file does nothing",
			setupModel: func() Model {
				return newMultiFileTestModel(120, 40)
			},
			keys:            []tea.KeyMsg{keyMsg("["), keyMsg("f")},
			checkActiveFile: true,
			wantActiveFile:  0,
		},
		{
			name: "g then x discards both keys",
			setupModel: func() Model {
				return newTestModel(buildTestPairs(), 120, 40)
			},
			keys:          []tea.KeyMsg{keyMsg("g"), keyMsg("x")},
			wantCursorRow: 0,
		},
		{
			name: "] then x discards both keys",
			setupModel: func() Model {
				return newTestModel(buildTestPairs(), 120, 40)
			},
			keys:          []tea.KeyMsg{keyMsg("]"), keyMsg("x")},
			wantCursorRow: 0,
		},
		{
			name: "[ then x discards both keys",
			setupModel: func() Model {
				return newTestModel(buildTestPairs(), 120, 40)
			},
			keys:          []tea.KeyMsg{keyMsg("["), keyMsg("x")},
			wantCursorRow: 0,
		},
		{
			name: "d then x discards both keys",
			setupModel: func() Model {
				return newTestModel(buildTestPairs(), 120, 40)
			},
			keys:          []tea.KeyMsg{keyMsg("d"), keyMsg("x")},
			wantCursorRow: 0,
		},
		{
			name: "g sets pending key",
			setupModel: func() Model {
				return newTestModel(buildTestPairs(), 120, 40)
			},
			keys:           []tea.KeyMsg{keyMsg("g")},
			wantPendingKey: "g",
		},
		{
			name: "] sets pending key",
			setupModel: func() Model {
				return newTestModel(buildTestPairs(), 120, 40)
			},
			keys:           []tea.KeyMsg{keyMsg("]")},
			wantPendingKey: "]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			for _, key := range tt.keys {
				newModel, _ := m.Update(key)
				m = newModel.(Model)
			}

			if m.cursorRow != tt.wantCursorRow {
				t.Errorf("cursorRow: got %d, want %d", m.cursorRow, tt.wantCursorRow)
			}
			if tt.checkActiveFile {
				if m.activeFile != tt.wantActiveFile {
					t.Errorf("activeFile: got %d, want %d", m.activeFile, tt.wantActiveFile)
				}
			}
			if m.pendingKey != tt.wantPendingKey {
				t.Errorf("pendingKey: got %q, want %q", m.pendingKey, tt.wantPendingKey)
			}
		})
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

// Helper to create Tab key message
func tabKeyMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyTab}
}

// Helper to create Enter key message
func enterKeyMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

// Helper to create Esc key message
func escKeyMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEsc}
}

// newTestModelWithStore creates a test model wired with a comment store (using temp dir).
func newTestModelWithStore(t *testing.T, pairs []diff.PairedLine, width, height int) Model {
	t.Helper()
	files := []diff.DiffFile{
		{OldName: "test.go", NewName: "test.go",
			Hunks: []diff.Hunk{{
				OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3,
				Lines: []diff.DiffLine{
					{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "context line one"},
					{Type: diff.LineDelete, OldNum: 2, Content: "deleted line"},
					{Type: diff.LineAdd, NewNum: 2, Content: "added line"},
					{Type: diff.LineContext, OldNum: 3, NewNum: 3, Content: "context line three"},
				},
			}},
		},
	}
	m := NewModel(files, pairs, width, height)
	m.SetStore(comment.NewStore(t.TempDir()))
	return m
}

func TestCommentCreation(t *testing.T) {
	tests := []struct {
		name         string
		setupModel   func(t *testing.T) Model
		keys         []tea.KeyMsg
		wantMode     InputMode
		wantOverlay  bool
		wantComments int
	}{
		{
			name: "c on code line opens comment overlay",
			setupModel: func(t *testing.T) Model {
				return newTestModelWithStore(t, buildTestPairs(), 120, 40)
			},
			keys:        []tea.KeyMsg{keyMsg("c")},
			wantMode:    InputModeComment,
			wantOverlay: true,
		},
		{
			name: "Esc cancels overlay and returns to normal",
			setupModel: func(t *testing.T) Model {
				return newTestModelWithStore(t, buildTestPairs(), 120, 40)
			},
			keys:         []tea.KeyMsg{keyMsg("c"), escKeyMsg()},
			wantMode:     InputModeNormal,
			wantOverlay:  false,
			wantComments: 0,
		},
		{
			name: "c on separator row does nothing",
			setupModel: func(t *testing.T) Model {
				m := newTestModelWithStore(t, buildMultiHunkPairs(), 120, 40)
				m.cursorRow = 2 // separator row
				return m
			},
			keys:        []tea.KeyMsg{keyMsg("c")},
			wantMode:    InputModeNormal,
			wantOverlay: false,
		},
		{
			name: "c on blank padding row does nothing",
			setupModel: func(t *testing.T) Model {
				// Create pairs with a nil side on active pane
				pairs := []diff.PairedLine{
					{Left: &diff.DiffLine{Type: diff.LineDelete, OldNum: 1, Content: "deleted"}, Right: nil},
				}
				m := newTestModelWithStore(t, pairs, 120, 40)
				m.activeSide = SideNew // Right is nil
				return m
			},
			keys:        []tea.KeyMsg{keyMsg("c")},
			wantMode:    InputModeNormal,
			wantOverlay: false,
		},
		{
			name: "c on line with existing comment does nothing",
			setupModel: func(t *testing.T) Model {
				m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
				// Pre-add a comment on line 1 (the NewNum of cursor row 0)
				_ = m.store.AddComment("test.go", 1, 0, "context line one", "existing")
				return m
			},
			keys:        []tea.KeyMsg{keyMsg("c")},
			wantMode:    InputModeNormal,
			wantOverlay: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel(t)
			for _, key := range tt.keys {
				newModel, _ := m.Update(key)
				m = newModel.(Model)
			}

			if m.mode != tt.wantMode {
				t.Errorf("mode: got %v, want %v", m.mode, tt.wantMode)
			}
			if m.overlay.Active != tt.wantOverlay {
				t.Errorf("overlay.Active: got %v, want %v", m.overlay.Active, tt.wantOverlay)
			}
			if tt.wantComments > 0 && m.store != nil {
				comments := m.store.Comments("test.go")
				if len(comments) != tt.wantComments {
					t.Errorf("comments: got %d, want %d", len(comments), tt.wantComments)
				}
			}
		})
	}
}

func TestCommentSubmission(t *testing.T) {
	m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
	// Press c to open overlay
	newModel, _ := m.Update(keyMsg("c"))
	m = newModel.(Model)

	if m.mode != InputModeComment {
		t.Fatalf("expected InputModeComment, got %v", m.mode)
	}

	// Type a comment body by sending rune keys to the text input
	for _, r := range "test comment body" {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	// Press Enter to submit
	newModel, _ = m.Update(enterKeyMsg())
	m = newModel.(Model)

	// Should return to normal mode
	if m.mode != InputModeNormal {
		t.Errorf("mode after submit: got %v, want InputModeNormal", m.mode)
	}
	if m.overlay.Active {
		t.Error("overlay should be inactive after submit")
	}

	// Comment should be saved in the store
	comments := m.store.Comments("test.go")
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Body != "test comment body" {
		t.Errorf("comment body: got %q, want %q", comments[0].Body, "test comment body")
	}
	if comments[0].Line != 1 {
		t.Errorf("comment line: got %d, want 1", comments[0].Line)
	}
}

func TestCommentSubmissionEmptyBody(t *testing.T) {
	m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
	// Press c to open overlay
	newModel, _ := m.Update(keyMsg("c"))
	m = newModel.(Model)

	// Press Enter with empty body — should not create comment
	newModel, _ = m.Update(enterKeyMsg())
	m = newModel.(Model)

	if m.mode != InputModeNormal {
		t.Errorf("mode: got %v, want InputModeNormal", m.mode)
	}
	comments := m.store.Comments("test.go")
	if len(comments) != 0 {
		t.Errorf("expected 0 comments for empty body, got %d", len(comments))
	}
}

func TestCommentHelpBar(t *testing.T) {
	output := RenderHelpBar(120, InputModeComment)
	if !strings.Contains(output, "Enter") {
		t.Error("comment mode help bar should mention Enter")
	}
	if !strings.Contains(output, "submit") {
		t.Error("comment mode help bar should mention submit")
	}
	if !strings.Contains(output, "Esc") {
		t.Error("comment mode help bar should mention Esc")
	}
	if !strings.Contains(output, "cancel") {
		t.Error("comment mode help bar should mention cancel")
	}
}

func TestCommentCountInStatusBar(t *testing.T) {
	m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
	// Add a comment via the store
	_ = m.store.AddComment("test.go", 1, 0, "context line one", "a comment")

	output := m.View()
	if !strings.Contains(output, "C:1") {
		t.Error("status bar should show C:1 after adding a comment")
	}
}

// --- CR-013: Comment Display, Navigation, Edit, Delete ---

// newTestModelWithComments creates a model with a comment already added and paired lines rebuilt.
func newTestModelWithComments(t *testing.T, width, height int) Model {
	t.Helper()
	pairs := buildTestPairs()
	m := newTestModelWithStore(t, pairs, width, height)
	// Add a comment on line 1 (NewNum of first context line)
	_ = m.store.AddComment("test.go", 1, 0, "context line one", "my test comment")
	m.rebuildPairedLines()
	return m
}

func TestCommentRowRendering(t *testing.T) {
	m := newTestModelWithComments(t, 120, 40)

	output := m.View()
	if !strings.Contains(output, "💬") {
		t.Error("expected 💬 prefix in rendered comment row")
	}
	if !strings.Contains(output, "my test comment") {
		t.Error("expected comment body text in rendered output")
	}
}

func TestCommentRowsInPairedLines(t *testing.T) {
	m := newTestModelWithComments(t, 120, 40)

	// Should have original pairs + 1 comment row
	foundComment := false
	for _, p := range m.paired {
		if p.IsComment {
			foundComment = true
			if p.CommentBody != "my test comment" {
				t.Errorf("CommentBody: got %q, want %q", p.CommentBody, "my test comment")
			}
		}
	}
	if !foundComment {
		t.Error("expected at least one comment row in paired lines")
	}
}

func TestCommentRowScrollsWithDiff(t *testing.T) {
	m := newTestModelWithComments(t, 120, 40)

	// Cursor can land on a comment row via j/k
	// Find where the comment row is
	commentIdx := -1
	for i, p := range m.paired {
		if p.IsComment {
			commentIdx = i
			break
		}
	}
	if commentIdx < 0 {
		t.Fatal("no comment row found")
	}

	// Navigate to comment row
	for i := 0; i < commentIdx; i++ {
		newModel, _ := m.Update(keyMsg("j"))
		m = newModel.(Model)
	}
	if m.cursorRow != commentIdx {
		t.Errorf("cursorRow: got %d, want %d", m.cursorRow, commentIdx)
	}
}

func TestCommentNavigationNextPrev(t *testing.T) {
	t.Run("]m jumps to next comment row", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		// Start at row 0, comment should be at row 1
		newModel, _ := m.Update(keyMsg("]"))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("m"))
		m = newModel.(Model)

		if !m.paired[m.cursorRow].IsComment {
			t.Errorf("expected cursor on comment row, got cursorRow=%d IsComment=%v",
				m.cursorRow, m.paired[m.cursorRow].IsComment)
		}
	})

	t.Run("[m jumps to prev comment row", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		// Move past the comment
		for i := 0; i < 4; i++ {
			newModel, _ := m.Update(keyMsg("j"))
			m = newModel.(Model)
		}
		// Jump back to prev comment
		newModel, _ := m.Update(keyMsg("["))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("m"))
		m = newModel.(Model)

		if !m.paired[m.cursorRow].IsComment {
			t.Errorf("expected cursor on comment row, got cursorRow=%d", m.cursorRow)
		}
	})

	t.Run("[m with no previous comments does nothing", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		// Cursor at row 0, before the comment row
		prevRow := m.cursorRow

		newModel, _ := m.Update(keyMsg("["))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("m"))
		m = newModel.(Model)

		if m.cursorRow != prevRow {
			t.Errorf("cursorRow should stay at %d, got %d", prevRow, m.cursorRow)
		}
	})

	t.Run("]m with no more comments does nothing", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		// Move to comment row first
		newModel, _ := m.Update(keyMsg("]"))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("m"))
		m = newModel.(Model)
		prevRow := m.cursorRow

		// Try to jump to next — no more comments
		newModel, _ = m.Update(keyMsg("]"))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("m"))
		m = newModel.(Model)

		if m.cursorRow != prevRow {
			t.Errorf("cursorRow should stay at %d, got %d", prevRow, m.cursorRow)
		}
	})
}

func TestCommentEdit(t *testing.T) {
	t.Run("e on comment row opens overlay pre-filled", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		// Navigate to the comment row
		commentIdx := -1
		for i, p := range m.paired {
			if p.IsComment {
				commentIdx = i
				break
			}
		}
		for i := 0; i < commentIdx; i++ {
			newModel, _ := m.Update(keyMsg("j"))
			m = newModel.(Model)
		}

		// Press e to edit
		newModel, _ := m.Update(keyMsg("e"))
		m = newModel.(Model)

		if m.mode != InputModeComment {
			t.Errorf("mode: got %v, want InputModeComment", m.mode)
		}
		if !m.overlay.Active {
			t.Error("overlay should be active")
		}
		if m.overlay.Value() != "my test comment" {
			t.Errorf("overlay value: got %q, want %q", m.overlay.Value(), "my test comment")
		}
	})

	t.Run("e on non-comment row does nothing", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		// Cursor on row 0 (code line)
		newModel, _ := m.Update(keyMsg("e"))
		m = newModel.(Model)

		if m.mode != InputModeNormal {
			t.Errorf("mode: got %v, want InputModeNormal", m.mode)
		}
		if m.overlay.Active {
			t.Error("overlay should not be active on non-comment row")
		}
	})

	t.Run("edit submit updates comment body", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		// Navigate to comment row
		commentIdx := -1
		for i, p := range m.paired {
			if p.IsComment {
				commentIdx = i
				break
			}
		}
		for i := 0; i < commentIdx; i++ {
			newModel, _ := m.Update(keyMsg("j"))
			m = newModel.(Model)
		}

		// Press e, clear and type new body, submit
		newModel, _ := m.Update(keyMsg("e"))
		m = newModel.(Model)

		// Select all and clear the pre-filled text
		// Use Ctrl+A to select all, then type replacement
		// Actually simpler: just use the textinput API directly for test
		m.overlay.Input.SetValue("updated comment")

		// Submit
		newModel, _ = m.Update(enterKeyMsg())
		m = newModel.(Model)

		if m.mode != InputModeNormal {
			t.Errorf("mode after edit submit: got %v, want InputModeNormal", m.mode)
		}

		comments := m.store.Comments("test.go")
		if len(comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(comments))
		}
		if comments[0].Body != "updated comment" {
			t.Errorf("comment body: got %q, want %q", comments[0].Body, "updated comment")
		}
	})
}

func TestCommentDelete(t *testing.T) {
	t.Run("dc on comment row deletes it", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		pairsBeforeDelete := len(m.paired)

		// Navigate to the comment row
		commentIdx := -1
		for i, p := range m.paired {
			if p.IsComment {
				commentIdx = i
				break
			}
		}
		for i := 0; i < commentIdx; i++ {
			newModel, _ := m.Update(keyMsg("j"))
			m = newModel.(Model)
		}

		// Press dc to delete
		newModel, _ := m.Update(keyMsg("d"))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("c"))
		m = newModel.(Model)

		// Comment should be gone from store
		comments := m.store.Comments("test.go")
		if len(comments) != 0 {
			t.Errorf("expected 0 comments after delete, got %d", len(comments))
		}

		// Paired lines should be shorter (comment row removed)
		if len(m.paired) >= pairsBeforeDelete {
			t.Errorf("paired lines should have fewer entries after delete, got %d (was %d)", len(m.paired), pairsBeforeDelete)
		}

		// No IsComment rows should remain
		for _, p := range m.paired {
			if p.IsComment {
				t.Error("no comment rows should remain after deletion")
			}
		}
	})

	t.Run("dc on non-comment row does nothing", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		commentsBefore := len(m.store.Comments("test.go"))

		// Cursor on row 0 (code line)
		newModel, _ := m.Update(keyMsg("d"))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("c"))
		m = newModel.(Model)

		commentsAfter := len(m.store.Comments("test.go"))
		if commentsAfter != commentsBefore {
			t.Errorf("comments: got %d, want %d (no change)", commentsAfter, commentsBefore)
		}
	})

	t.Run("cursor moves up after delete", func(t *testing.T) {
		m := newTestModelWithComments(t, 120, 40)
		commentIdx := -1
		for i, p := range m.paired {
			if p.IsComment {
				commentIdx = i
				break
			}
		}
		for i := 0; i < commentIdx; i++ {
			newModel, _ := m.Update(keyMsg("j"))
			m = newModel.(Model)
		}

		// Delete the comment
		newModel, _ := m.Update(keyMsg("d"))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("c"))
		m = newModel.(Model)

		// Cursor should be on the code line above (commentIdx - 1)
		expectedRow := commentIdx - 1
		if expectedRow < 0 {
			expectedRow = 0
		}
		if m.cursorRow != expectedRow {
			t.Errorf("cursorRow after delete: got %d, want %d", m.cursorRow, expectedRow)
		}
	})
}

func TestCommentAddIsCommentGuard(t *testing.T) {
	m := newTestModelWithComments(t, 120, 40)
	// Navigate to comment row
	commentIdx := -1
	for i, p := range m.paired {
		if p.IsComment {
			commentIdx = i
			break
		}
	}
	for i := 0; i < commentIdx; i++ {
		newModel, _ := m.Update(keyMsg("j"))
		m = newModel.(Model)
	}

	// Press c on a comment row — should NOT open overlay
	newModel, _ := m.Update(keyMsg("c"))
	m = newModel.(Model)

	if m.mode != InputModeNormal {
		t.Errorf("mode: got %v, want InputModeNormal (c on comment row should be no-op)", m.mode)
	}
	if m.overlay.Active {
		t.Error("overlay should not open on comment row")
	}
}

func TestCommentSubmissionRebuildsPairedLines(t *testing.T) {
	m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
	pairsBefore := len(m.paired)

	// Press c to open overlay
	newModel, _ := m.Update(keyMsg("c"))
	m = newModel.(Model)

	// Type body and submit
	for _, r := range "new comment" {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}
	newModel, _ = m.Update(enterKeyMsg())
	m = newModel.(Model)

	// Paired lines should now include a comment row
	if len(m.paired) <= pairsBefore {
		t.Errorf("paired lines should grow after comment submission, got %d (was %d)", len(m.paired), pairsBefore)
	}

	foundComment := false
	for _, p := range m.paired {
		if p.IsComment && p.CommentBody == "new comment" {
			foundComment = true
			break
		}
	}
	if !foundComment {
		t.Error("expected comment row with 'new comment' body after submission")
	}
}

// --- CR-014: Visual Select Mode and Range Comments ---

func TestVisualSelectMode(t *testing.T) {
	tests := []struct {
		name            string
		setupModel      func(t *testing.T) Model
		keys            []tea.KeyMsg
		wantMode        InputMode
		wantVisualStart int
		wantCursorRow   int
	}{
		{
			name: "V enters visual mode and sets anchor",
			setupModel: func(t *testing.T) Model {
				m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
				m.cursorRow = 1
				return m
			},
			keys:            []tea.KeyMsg{keyMsg("V")},
			wantMode:        InputModeVisual,
			wantVisualStart: 1,
			wantCursorRow:   1,
		},
		{
			name: "j extends selection downward",
			setupModel: func(t *testing.T) Model {
				return newTestModelWithStore(t, buildTestPairs(), 120, 40)
			},
			keys:            []tea.KeyMsg{keyMsg("V"), keyMsg("j"), keyMsg("j")},
			wantMode:        InputModeVisual,
			wantVisualStart: 0,
			wantCursorRow:   2,
		},
		{
			name: "k extends selection upward",
			setupModel: func(t *testing.T) Model {
				m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
				m.cursorRow = 2
				return m
			},
			keys:            []tea.KeyMsg{keyMsg("V"), keyMsg("k")},
			wantMode:        InputModeVisual,
			wantVisualStart: 2,
			wantCursorRow:   1,
		},
		{
			name: "Esc cancels visual mode",
			setupModel: func(t *testing.T) Model {
				return newTestModelWithStore(t, buildTestPairs(), 120, 40)
			},
			keys:          []tea.KeyMsg{keyMsg("V"), keyMsg("j"), escKeyMsg()},
			wantMode:      InputModeNormal,
			wantCursorRow: 1,
		},
		{
			name: "V on separator row does nothing",
			setupModel: func(t *testing.T) Model {
				m := newTestModelWithStore(t, buildMultiHunkPairs(), 120, 40)
				m.cursorRow = 2 // separator row
				return m
			},
			keys:          []tea.KeyMsg{keyMsg("V")},
			wantMode:      InputModeNormal,
			wantCursorRow: 2,
		},
		{
			name: "V on comment row does nothing",
			setupModel: func(t *testing.T) Model {
				m := newTestModelWithComments(t, 120, 40)
				// Navigate to comment row
				commentIdx := -1
				for i, p := range m.paired {
					if p.IsComment {
						commentIdx = i
						break
					}
				}
				m.cursorRow = commentIdx
				return m
			},
			keys:          []tea.KeyMsg{keyMsg("V")},
			wantMode:      InputModeNormal,
			wantCursorRow: 1, // comment row is at index 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel(t)
			for _, key := range tt.keys {
				newModel, _ := m.Update(key)
				m = newModel.(Model)
			}

			if m.mode != tt.wantMode {
				t.Errorf("mode: got %v, want %v", m.mode, tt.wantMode)
			}
			if tt.wantMode == InputModeVisual && m.visualStart != tt.wantVisualStart {
				t.Errorf("visualStart: got %d, want %d", m.visualStart, tt.wantVisualStart)
			}
			if m.cursorRow != tt.wantCursorRow {
				t.Errorf("cursorRow: got %d, want %d", m.cursorRow, tt.wantCursorRow)
			}
		})
	}
}

func TestVisualSelectPaneSwitching(t *testing.T) {
	m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
	// Enter visual mode
	newModel, _ := m.Update(keyMsg("V"))
	m = newModel.(Model)

	if m.activeSide != SideNew {
		t.Fatalf("expected SideNew initially, got %v", m.activeSide)
	}

	// Press h to switch to old pane
	newModel, _ = m.Update(keyMsg("h"))
	m = newModel.(Model)

	if m.mode != InputModeVisual {
		t.Errorf("should stay in visual mode after h, got %v", m.mode)
	}
	if m.activeSide != SideOld {
		t.Errorf("activeSide: got %v, want SideOld", m.activeSide)
	}

	// Press l to switch back to new pane
	newModel, _ = m.Update(keyMsg("l"))
	m = newModel.(Model)

	if m.activeSide != SideNew {
		t.Errorf("activeSide: got %v, want SideNew", m.activeSide)
	}
}

func TestVisualSelectCommentCreation(t *testing.T) {
	t.Run("c in visual mode opens overlay with range header", func(t *testing.T) {
		m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
		// V on row 0, j to extend to row 1, then c to comment
		newModel, _ := m.Update(keyMsg("V"))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("j"))
		m = newModel.(Model)
		newModel, _ = m.Update(keyMsg("c"))
		m = newModel.(Model)

		if m.mode != InputModeComment {
			t.Errorf("mode: got %v, want InputModeComment", m.mode)
		}
		if !m.overlay.Active {
			t.Error("overlay should be active")
		}

		// Overlay should show range in the render
		rendered := m.overlay.Render(120)
		if !strings.Contains(rendered, "-L") {
			t.Errorf("overlay should show range header with -L, got: %s", rendered)
		}
	})

	t.Run("submitting range comment sets line and end_line", func(t *testing.T) {
		m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
		// V on row 0 (NewNum=1), j to row 1 (delete line, no NewNum — skip), then to row 2 (add line NewNum=2)
		// Actually buildTestPairs: row0=context(1,1), row1=delete(old2), row2=add(new2), row3=context(3,3)
		// On SideNew: row0 NewNum=1, row1 Right=nil (delete), row2 NewNum=2, row3 NewNum=3
		// Select row 0 and row 3 (both have NewNum on new side)
		newModel, _ := m.Update(keyMsg("V"))
		m = newModel.(Model)
		for i := 0; i < 3; i++ {
			newModel, _ = m.Update(keyMsg("j"))
			m = newModel.(Model)
		}
		// Now visualStart=0, cursorRow=3
		newModel, _ = m.Update(keyMsg("c"))
		m = newModel.(Model)

		// Type comment and submit
		for _, r := range "range comment" {
			newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = newModel.(Model)
		}
		newModel, _ = m.Update(enterKeyMsg())
		m = newModel.(Model)

		if m.mode != InputModeNormal {
			t.Errorf("mode after submit: got %v, want InputModeNormal", m.mode)
		}

		comments := m.store.Comments("test.go")
		if len(comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(comments))
		}
		c := comments[0]
		if c.Body != "range comment" {
			t.Errorf("body: got %q, want %q", c.Body, "range comment")
		}
		if c.Line >= c.EndLine {
			t.Errorf("expected Line < EndLine, got Line=%d EndLine=%d", c.Line, c.EndLine)
		}
		if c.EndLine == 0 {
			t.Errorf("EndLine should be non-zero for range comment, got 0")
		}
	})

	t.Run("range comment snippet is first line of range", func(t *testing.T) {
		m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
		// Select rows 0-3 (context line one through context line three)
		newModel, _ := m.Update(keyMsg("V"))
		m = newModel.(Model)
		for i := 0; i < 3; i++ {
			newModel, _ = m.Update(keyMsg("j"))
			m = newModel.(Model)
		}
		newModel, _ = m.Update(keyMsg("c"))
		m = newModel.(Model)

		for _, r := range "snippet test" {
			newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = newModel.(Model)
		}
		newModel, _ = m.Update(enterKeyMsg())
		m = newModel.(Model)

		comments := m.store.Comments("test.go")
		if len(comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(comments))
		}
		if comments[0].ContentSnippet != "context line one" {
			t.Errorf("snippet: got %q, want %q", comments[0].ContentSnippet, "context line one")
		}
	})
}

func TestVisualSelectHelpBar(t *testing.T) {
	output := RenderHelpBar(120, InputModeVisual)
	if !strings.Contains(output, "j/k") {
		t.Error("visual mode help bar should mention j/k")
	}
	if !strings.Contains(output, "extend") {
		t.Error("visual mode help bar should mention extend")
	}
	if !strings.Contains(output, "comment") {
		t.Error("visual mode help bar should mention comment")
	}
	if !strings.Contains(output, "Esc") {
		t.Error("visual mode help bar should mention Esc")
	}
}

func TestVisualSelectRendering(t *testing.T) {
	m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
	// Enter visual mode and extend selection
	newModel, _ := m.Update(keyMsg("V"))
	m = newModel.(Model)
	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(Model)

	// View should render without crashing
	output := m.View()
	if output == "" {
		t.Error("View() should return non-empty output in visual mode")
	}
}

func TestVisualSelectQuit(t *testing.T) {
	m := newTestModelWithStore(t, buildTestPairs(), 120, 40)
	newModel, _ := m.Update(keyMsg("V"))
	m = newModel.(Model)

	// q should still quit from visual mode
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Error("expected quit command from visual mode")
	}
}
