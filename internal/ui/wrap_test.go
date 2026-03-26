package ui

import (
	"strings"
	"testing"

	"github.com/adil/cr/internal/diff"
	tea "github.com/charmbracelet/bubbletea"
)

func TestToggleWrap(t *testing.T) {
	m := newTestModel(buildTestPairs(), 120, 40)

	if m.wrapMode {
		t.Fatal("wrap should be off by default")
	}

	// Press z to toggle wrap on
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = result.(Model)

	if !m.wrapMode {
		t.Error("wrap should be on after z press")
	}

	// Press z again to toggle off
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = result.(Model)

	if m.wrapMode {
		t.Error("wrap should be off after second z press")
	}
}

func TestToggleWrapPreservesCursorPosition(t *testing.T) {
	hunks := []diff.Hunk{
		{
			OldStart: 1, OldCount: 4, NewStart: 1, NewCount: 4,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "line one"},
				{Type: diff.LineContext, OldNum: 2, NewNum: 2, Content: "line two"},
				{Type: diff.LineContext, OldNum: 3, NewNum: 3, Content: "line three"},
				{Type: diff.LineContext, OldNum: 4, NewNum: 4, Content: "line four"},
			},
		},
	}
	pairs := diff.BuildPairedLines(hunks)
	files := []diff.DiffFile{
		{OldName: "test.go", NewName: "test.go", Hunks: hunks},
	}
	m := NewModel(files, pairs, 120, 40)
	m.cursorRow = 2

	// Toggle wrap on
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = result.(Model)

	// Cursor should still be valid (not lost)
	if m.cursorRow < 0 || m.cursorRow >= len(m.paired) {
		t.Errorf("cursor out of range after wrap toggle: %d (paired len: %d)", m.cursorRow, len(m.paired))
	}
}

func TestNarrowTerminalShowsMessage(t *testing.T) {
	m := newTestModel(buildTestPairs(), 80, 40) // 80 < default min_width 100

	view := m.View()

	if !strings.Contains(view, "Terminal too narrow") {
		t.Error("expected narrow terminal message, got normal diff view")
	}
}

func TestNarrowTerminalUsesConfigMinWidth(t *testing.T) {
	m := newTestModel(buildTestPairs(), 80, 40)
	m.config.Display.MinWidth = 60 // lower min_width to 60

	view := m.View()

	// 80 >= 60, so should NOT show narrow message
	if strings.Contains(view, "Terminal too narrow") {
		t.Error("should not show narrow terminal message when width >= min_width")
	}
}

func TestResizeAboveMinWidthRestoresDiff(t *testing.T) {
	m := newTestModel(buildTestPairs(), 80, 40) // starts narrow

	view := m.View()
	if !strings.Contains(view, "Terminal too narrow") {
		t.Fatal("expected narrow terminal message at width 80")
	}

	// Resize to above min_width
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = result.(Model)

	view = m.View()
	if strings.Contains(view, "Terminal too narrow") {
		t.Error("should show diff after resizing above min_width")
	}
}

func TestWrapModeRecomputesPairedLines(t *testing.T) {
	// Create a file with a long line that will wrap
	hunks := []diff.Hunk{
		{
			OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: strings.Repeat("a", 200)},
			},
		},
	}
	pairs := diff.BuildPairedLines(hunks)
	files := []diff.DiffFile{
		{OldName: "test.go", NewName: "test.go", Hunks: hunks},
	}
	m := NewModel(files, pairs, 120, 40)

	originalLen := len(m.paired)

	// Toggle wrap on — long line should expand to multiple visual rows
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = result.(Model)

	if len(m.paired) <= originalLen {
		t.Errorf("wrap mode should produce more paired lines; got %d, original was %d", len(m.paired), originalLen)
	}
}

func TestNarrowTerminalNoInteractionWithSpecialFiles(t *testing.T) {
	// Binary file at narrow width should still show narrow message, not binary message
	files := []diff.DiffFile{
		{OldName: "image.png", NewName: "image.png", IsBinary: true},
	}
	m := NewModel(files, nil, 80, 40)

	view := m.View()
	if !strings.Contains(view, "Terminal too narrow") {
		t.Error("narrow terminal should take priority over special file messages")
	}
}

func TestWrapModeRendersWithoutCrash(t *testing.T) {
	hunks := []diff.Hunk{
		{
			OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: strings.Repeat("x", 200)},
				{Type: diff.LineDelete, OldNum: 2, Content: strings.Repeat("d", 150)},
				{Type: diff.LineAdd, NewNum: 2, Content: "short add"},
			},
		},
	}
	pairs := diff.BuildPairedLines(hunks)
	files := []diff.DiffFile{
		{OldName: "test.go", NewName: "test.go", Hunks: hunks},
	}
	m := NewModel(files, pairs, 120, 40)

	// Toggle wrap
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = result.(Model)

	// View should not panic
	view := m.View()
	if len(view) == 0 {
		t.Error("expected non-empty view in wrap mode")
	}
}

func TestWrapModeResizeRecomputes(t *testing.T) {
	hunks := []diff.Hunk{
		{
			OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: strings.Repeat("a", 100)},
			},
		},
	}
	pairs := diff.BuildPairedLines(hunks)
	files := []diff.DiffFile{
		{OldName: "test.go", NewName: "test.go", Hunks: hunks},
	}
	m := NewModel(files, pairs, 120, 40)

	// Toggle wrap on
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = result.(Model)
	lenBefore := len(m.paired)

	// Resize to narrower terminal — should recompute wrap
	result, _ = m.Update(tea.WindowSizeMsg{Width: 110, Height: 40})
	m = result.(Model)
	lenAfter := len(m.paired)

	// Narrower panes should mean more wrap rows (or same if still fits)
	if lenAfter < lenBefore {
		t.Errorf("narrower terminal should not reduce wrapped line count: before=%d, after=%d", lenBefore, lenAfter)
	}
}

