package ui

import (
	"strings"
	"testing"

	"github.com/adilp/crit-diff/internal/diff"
)

func TestViewContainsStatusBar(t *testing.T) {
	pairs := buildTestPairs()
	files := []diff.DiffFile{
		{
			OldName: "test.go", NewName: "test.go",
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
	m := NewModel(files, pairs, 120, 40)
	m.ref = "main..HEAD"

	output := m.View()

	// Status bar should contain ref, file position, file path
	if !strings.Contains(output, "cr: main..HEAD") {
		t.Errorf("View() should contain ref in status bar, got:\n%s", output)
	}
	if !strings.Contains(output, "[1/1]") {
		t.Errorf("View() should contain file position [1/1], got:\n%s", output)
	}
	if !strings.Contains(output, "test.go") {
		t.Errorf("View() should contain file name, got:\n%s", output)
	}
}

func TestViewContainsHelpBar(t *testing.T) {
	pairs := buildTestPairs()
	m := newTestModel(pairs, 120, 40)

	output := m.View()

	// Help bar should contain key hints for normal mode
	if !strings.Contains(output, "scroll") {
		t.Errorf("View() should contain help bar with 'scroll' hint, got:\n%s", output)
	}
	if !strings.Contains(output, "quit") {
		t.Errorf("View() should contain help bar with 'quit' hint, got:\n%s", output)
	}
}

func TestViewWorkingTreeRef(t *testing.T) {
	pairs := buildTestPairs()
	m := newTestModel(pairs, 120, 40)
	// ref is empty by default (working tree)

	output := m.View()

	if !strings.Contains(output, "(working tree)") {
		t.Errorf("View() should show '(working tree)' when ref is empty, got:\n%s", output)
	}
}

func TestViewInsertionDeletionCounts(t *testing.T) {
	files := []diff.DiffFile{
		{
			OldName: "test.go", NewName: "test.go",
			Hunks: []diff.Hunk{{
				OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 4,
				Lines: []diff.DiffLine{
					{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "ctx"},
					{Type: diff.LineDelete, OldNum: 2, Content: "old1"},
					{Type: diff.LineDelete, OldNum: 3, Content: "old2"},
					{Type: diff.LineAdd, NewNum: 2, Content: "new1"},
					{Type: diff.LineAdd, NewNum: 3, Content: "new2"},
					{Type: diff.LineAdd, NewNum: 4, Content: "new3"},
				},
			}},
		},
	}
	pairs := diff.BuildPairedLines(files[0].Hunks)
	m := NewModel(files, pairs, 120, 40)

	output := m.View()

	// 3 adds, 2 deletes
	if !strings.Contains(output, "+3") {
		t.Errorf("View() should show +3 additions, got:\n%s", output)
	}
	if !strings.Contains(output, "-2") {
		t.Errorf("View() should show -2 deletions, got:\n%s", output)
	}
}

func TestViewActiveSideInStatusBar(t *testing.T) {
	pairs := buildTestPairs()
	m := newTestModel(pairs, 120, 40)

	// Default is SideNew
	output := m.View()
	if !strings.Contains(output, "new") {
		t.Errorf("View() should show 'new' as active side, got:\n%s", output)
	}

	// Switch to old side
	newModel, _ := m.Update(keyMsg("h"))
	m = newModel.(Model)
	output = m.View()
	if !strings.Contains(output, "old") {
		t.Errorf("View() should show 'old' after pressing h, got:\n%s", output)
	}
}

func TestViewCommentCountZero(t *testing.T) {
	pairs := buildTestPairs()
	m := newTestModel(pairs, 120, 40)

	output := m.View()

	if !strings.Contains(output, "C:0") {
		t.Errorf("View() should show C:0 when no comments, got:\n%s", output)
	}
}
