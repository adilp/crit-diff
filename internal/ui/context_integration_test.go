package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/adilp/crit-diff/internal/diff"
)

// buildExpandableHunks creates two hunks with a gap for context expansion.
// Hunk 0: old lines 1-3, Hunk 1: old lines 10-12.
// Gap: old lines 4-9 (6 lines).
func buildExpandableHunks() []diff.Hunk {
	return []diff.Hunk{
		{
			OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "line 1"},
				{Type: diff.LineDelete, OldNum: 2, Content: "old line 2"},
				{Type: diff.LineAdd, NewNum: 2, Content: "new line 2"},
				{Type: diff.LineContext, OldNum: 3, NewNum: 3, Content: "line 3"},
			},
		},
		{
			OldStart: 10, OldCount: 3, NewStart: 10, NewCount: 3,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 10, NewNum: 10, Content: "line 10"},
				{Type: diff.LineDelete, OldNum: 11, Content: "old line 11"},
				{Type: diff.LineAdd, NewNum: 11, Content: "new line 11"},
				{Type: diff.LineContext, OldNum: 12, NewNum: 12, Content: "line 12"},
			},
		},
	}
}

// buildExpandableModel creates a model with expandable hunks and file content.
func buildExpandableModel() Model {
	hunks := buildExpandableHunks()
	files := []diff.DiffFile{
		{OldName: "test.go", NewName: "test.go", Hunks: hunks},
	}
	paired := diff.BuildPairedLines(hunks)
	m := NewModel(files, paired, 120, 40)
	m.allPaired = [][]diff.PairedLine{paired}

	// Build file content for 20 lines
	oldLines := make([]string, 20)
	newLines := make([]string, 20)
	for i := range 20 {
		oldLines[i] = fmt.Sprintf("line %d", i+1)
		newLines[i] = fmt.Sprintf("line %d", i+1)
	}
	m.oldFileLines = oldLines
	m.newFileLines = newLines

	// Initialize separator states
	m.separatorStates = BuildSeparatorStates(hunks)

	return m
}

func TestExpandContextDown(t *testing.T) {
	m := buildExpandableModel()

	// paired: [0]=ctx1, [1]=del/add, [2]=ctx3, [3]=separator, [4]=ctx10, [5]=del/add, [6]=ctx12
	// Verify separator is at index 3
	if !m.paired[3].IsSeparator {
		t.Fatal("expected separator at index 3")
	}

	initialLen := len(m.paired)

	// Press ]e to expand context below
	newM, _ := m.Update(keyMsg("]"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("e"))
	m = newM.(Model)

	// Should have inserted 6 context lines (the full gap: lines 4-9)
	expectedLen := initialLen + 6 - 1 // +6 lines, -1 separator removed (fully expanded)
	if len(m.paired) != expectedLen {
		t.Errorf("after expand: len(paired) = %d, want %d", len(m.paired), expectedLen)
	}

	// The expanded lines should contain the gap content
	// After expanding down 6 lines (full gap), separator should be gone
	hasSeparator := false
	for _, p := range m.paired {
		if p.IsSeparator {
			hasSeparator = true
			break
		}
	}
	if hasSeparator {
		t.Error("separator should be removed after full expansion")
	}

	// Verify expanded context lines have correct content
	// Index 3 should now be context line 4 (first expanded line)
	if m.paired[3].Left == nil || m.paired[3].Left.OldNum != 4 {
		left := "nil"
		if m.paired[3].Left != nil {
			left = fmt.Sprintf("OldNum=%d", m.paired[3].Left.OldNum)
		}
		t.Errorf("expanded line at idx 3: Left = %s, want OldNum=4", left)
	}
}

func TestExpandContextUp(t *testing.T) {
	m := buildExpandableModel()

	// Move cursor to after the separator (on hunk 2)
	m.cursorRow = 4

	// Press [e to expand context above
	newM, _ := m.Update(keyMsg("["))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("e"))
	m = newM.(Model)

	// Full gap is 6 lines. Should expand all 6 and remove separator
	hasSeparator := false
	for _, p := range m.paired {
		if p.IsSeparator {
			hasSeparator = true
			break
		}
	}
	if hasSeparator {
		t.Error("separator should be removed after full expansion via [e")
	}
}

func TestExpandContextIncremental(t *testing.T) {
	// Create a model with a larger gap (15 lines between hunks)
	hunks := []diff.Hunk{
		{
			OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "line 1"},
				{Type: diff.LineContext, OldNum: 2, NewNum: 2, Content: "line 2"},
			},
		},
		{
			OldStart: 18, OldCount: 2, NewStart: 18, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 18, NewNum: 18, Content: "line 18"},
				{Type: diff.LineContext, OldNum: 19, NewNum: 19, Content: "line 19"},
			},
		},
	}
	files := []diff.DiffFile{{OldName: "test.go", NewName: "test.go", Hunks: hunks}}
	paired := diff.BuildPairedLines(hunks)
	m := NewModel(files, paired, 120, 40)
	m.allPaired = [][]diff.PairedLine{paired}

	oldLines := make([]string, 25)
	newLines := make([]string, 25)
	for i := range 25 {
		oldLines[i] = fmt.Sprintf("line %d", i+1)
		newLines[i] = fmt.Sprintf("line %d", i+1)
	}
	m.oldFileLines = oldLines
	m.newFileLines = newLines
	m.separatorStates = BuildSeparatorStates(hunks)

	// First ]e: should expand 10 lines, separator still present (15-10 = 5 remaining)
	newM, _ := m.Update(keyMsg("]"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("e"))
	m = newM.(Model)

	// Separator should still exist (5 lines remaining)
	hasSeparator := false
	for _, p := range m.paired {
		if p.IsSeparator {
			hasSeparator = true
			break
		}
	}
	if !hasSeparator {
		t.Error("separator should still exist after partial expansion (10 of 15)")
	}

	// Second ]e: should expand remaining 5 lines, separator disappears
	newM, _ = m.Update(keyMsg("]"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("e"))
	m = newM.(Model)

	hasSeparator = false
	for _, p := range m.paired {
		if p.IsSeparator {
			hasSeparator = true
			break
		}
	}
	if hasSeparator {
		t.Error("separator should be removed after second expansion (15 of 15)")
	}
}

func TestExpandContextNoSeparator(t *testing.T) {
	// Single hunk, no separator — ]e should be no-op
	m := newTestModel(buildTestPairs(), 120, 40)
	initialLen := len(m.paired)

	newM, _ := m.Update(keyMsg("]"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("e"))
	m = newM.(Model)

	if len(m.paired) != initialLen {
		t.Errorf("expected no change, len went from %d to %d", initialLen, len(m.paired))
	}
}

func TestExpandContextSeparatorText(t *testing.T) {
	m := buildExpandableModel()

	// Render the view and check separator text contains expand hint
	view := m.View()
	// Strip ANSI escape codes for reliable text matching
	stripped := stripAnsi(view)
	if !strings.Contains(stripped, "expand context") || !strings.Contains(stripped, "]e") || !strings.Contains(stripped, "[e") {
		t.Errorf("separator should show expand hint with ]e / [e, got:\n%s", stripped)
	}
}

// stripAnsi removes ANSI escape sequences from a string for test assertions.
func stripAnsi(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until 'm' or end of string
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++ // skip 'm'
			}
			i = j
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

func TestExpandContextCursorNotOnSeparator(t *testing.T) {
	m := buildExpandableModel()

	// Cursor on row 0 (not on separator at row 3)
	// ]e should find the separator below and expand it
	m.cursorRow = 0

	newM, _ := m.Update(keyMsg("]"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("e"))
	m = newM.(Model)

	// Should still have expanded (found separator below)
	hasSeparator := false
	for _, p := range m.paired {
		if p.IsSeparator {
			hasSeparator = true
			break
		}
	}
	if hasSeparator {
		t.Error("separator should be removed — ]e should find and expand nearest separator below cursor")
	}
}

func TestExpandContextAlreadyFullyExpanded(t *testing.T) {
	m := buildExpandableModel()

	// Expand fully first
	newM, _ := m.Update(keyMsg("]"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("e"))
	m = newM.(Model)

	lenAfterFull := len(m.paired)

	// Try expanding again — should be no-op
	newM, _ = m.Update(keyMsg("]"))
	m = newM.(Model)
	newM, _ = m.Update(keyMsg("e"))
	m = newM.(Model)

	if len(m.paired) != lenAfterFull {
		t.Errorf("expected no change after full expansion, len went from %d to %d", lenAfterFull, len(m.paired))
	}
}
