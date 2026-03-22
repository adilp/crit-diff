package ui

import (
	"strings"
	"testing"

	"github.com/adil/cr/internal/diff"
)

func TestBuildFileList(t *testing.T) {
	files := []diff.DiffFile{
		{OldName: "src/auth.go", NewName: "src/auth.go"},
		{OldName: "src/helper.go", NewName: "src/helper.go"},
		{OldName: "", NewName: "src/new.go", IsNew: true},
		{OldName: "src/old.go", NewName: "", IsDeleted: true},
	}

	got := BuildFileList(files)

	if len(got) != 4 {
		t.Fatalf("BuildFileList: got %d items, want 4", len(got))
	}

	// Uses NewName when available
	if got[0] != "src/auth.go" {
		t.Errorf("got[0]: got %q, want %q", got[0], "src/auth.go")
	}
	// New file uses NewName
	if got[2] != "src/new.go" {
		t.Errorf("got[2]: got %q, want %q", got[2], "src/new.go")
	}
	// Deleted file uses OldName
	if got[3] != "src/old.go" {
		t.Errorf("got[3]: got %q, want %q", got[3], "src/old.go")
	}
}

func TestBuildContentList(t *testing.T) {
	files := []diff.DiffFile{
		{OldName: "src/auth.go", NewName: "src/auth.go",
			Hunks: []diff.Hunk{{
				OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3,
				Lines: []diff.DiffLine{
					{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "package main"},
					{Type: diff.LineDelete, OldNum: 2, Content: "old line"},
					{Type: diff.LineAdd, NewNum: 2, Content: "new line"},
				},
			}},
		},
	}

	got := BuildContentList(files)

	if len(got) != 3 {
		t.Fatalf("BuildContentList: got %d items, want 3", len(got))
	}

	// Context line
	if !strings.Contains(got[0].Display, "[ ]") {
		t.Errorf("context line should have [ ], got %q", got[0].Display)
	}
	if !strings.Contains(got[0].Display, "src/auth.go:1") {
		t.Errorf("context line should have file:line, got %q", got[0].Display)
	}

	// Delete line
	if !strings.Contains(got[1].Display, "[-]") {
		t.Errorf("delete line should have [-], got %q", got[1].Display)
	}

	// Add line
	if !strings.Contains(got[2].Display, "[+]") {
		t.Errorf("add line should have [+], got %q", got[2].Display)
	}

	// Check navigation info
	if got[2].FileIndex != 0 {
		t.Errorf("add line FileIndex: got %d, want 0", got[2].FileIndex)
	}
	if got[2].LineNum != 2 {
		t.Errorf("add line LineNum: got %d, want 2", got[2].LineNum)
	}
}

func TestFuzzyFilterIndices(t *testing.T) {
	items := []string{
		"src/auth.go",
		"src/helper.go",
		"README.md",
	}

	tests := []struct {
		name  string
		query string
		want  int
	}{
		{name: "empty query returns all", query: "", want: 3},
		{name: "matching query", query: "auth", want: 1},
		{name: "case insensitive", query: "README", want: 1},
		{name: "no match", query: "zzz", want: 0},
		{name: "partial match", query: "src", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FuzzyFilterIndices(items, tt.query)
			if len(got) != tt.want {
				t.Errorf("FuzzyFilterIndices(%q): got %d items, want %d", tt.query, len(got), tt.want)
			}
		})
	}
}

func TestFuzzyStateNavigation(t *testing.T) {
	fs := NewFuzzyState(FuzzyModeFiles, []string{"a.go", "b.go", "c.go"})

	if fs.Cursor != 0 {
		t.Errorf("initial cursor: got %d, want 0", fs.Cursor)
	}

	fs.CursorDown()
	if fs.Cursor != 1 {
		t.Errorf("after CursorDown: got %d, want 1", fs.Cursor)
	}

	fs.CursorDown()
	fs.CursorDown() // should clamp
	if fs.Cursor != 2 {
		t.Errorf("after CursorDown x3: got %d, want 2", fs.Cursor)
	}

	fs.CursorUp()
	if fs.Cursor != 1 {
		t.Errorf("after CursorUp: got %d, want 1", fs.Cursor)
	}

	fs.CursorUp()
	fs.CursorUp() // should clamp
	if fs.Cursor != 0 {
		t.Errorf("after CursorUp x3: got %d, want 0", fs.Cursor)
	}
}

func TestFuzzyStateLiveFilter(t *testing.T) {
	fs := NewFuzzyState(FuzzyModeFiles, []string{"auth.go", "helper.go", "README.md"})

	fs.UpdateFilter("auth")
	if len(fs.Filtered) != 1 {
		t.Errorf("after filter 'auth': got %d items, want 1", len(fs.Filtered))
	}
	if fs.Cursor != 0 {
		t.Errorf("cursor should reset to 0 after filter, got %d", fs.Cursor)
	}

	fs.UpdateFilter("")
	if len(fs.Filtered) != 3 {
		t.Errorf("after clearing filter: got %d items, want 3", len(fs.Filtered))
	}
}

func TestFuzzyStateEnsureCursorVisible(t *testing.T) {
	items := make([]string, 20)
	for i := range items {
		items[i] = "item"
	}
	fs := NewFuzzyState(FuzzyModeFiles, items)

	// Move cursor beyond visible window (maxItems=5)
	fs.Cursor = 7
	fs.EnsureCursorVisible(5)
	if fs.YOffset != 3 { // 7 - 5 + 1 = 3
		t.Errorf("YOffset after scroll down: got %d, want 3", fs.YOffset)
	}

	// Move cursor back above visible window
	fs.Cursor = 1
	fs.EnsureCursorVisible(5)
	if fs.YOffset != 1 {
		t.Errorf("YOffset after scroll up: got %d, want 1", fs.YOffset)
	}
}

func TestFuzzyStateSelectedItem(t *testing.T) {
	fs := NewFuzzyState(FuzzyModeFiles, []string{"a.go", "b.go", "c.go"})

	got, ok := fs.SelectedItem()
	if !ok {
		t.Fatal("SelectedItem should return true")
	}
	if got != "a.go" {
		t.Errorf("SelectedItem: got %q, want %q", got, "a.go")
	}

	fs.CursorDown()
	got, ok = fs.SelectedItem()
	if !ok || got != "b.go" {
		t.Errorf("SelectedItem after down: got %q, want %q", got, "b.go")
	}

	// No items
	empty := NewFuzzyState(FuzzyModeFiles, []string{})
	_, ok = empty.SelectedItem()
	if ok {
		t.Error("SelectedItem on empty should return false")
	}
}

func TestParseContentSelection(t *testing.T) {
	tests := []struct {
		name      string
		selection string
		wantFile  string
		wantLine  int
		wantOk    bool
	}{
		{
			name:      "valid add line",
			selection: "src/auth.go:440 [+]: if check",
			wantFile:  "src/auth.go",
			wantLine:  440,
			wantOk:    true,
		},
		{
			name:      "valid delete line",
			selection: "src/auth.go:23 [-]: old code",
			wantFile:  "src/auth.go",
			wantLine:  23,
			wantOk:    true,
		},
		{
			name:      "valid context line",
			selection: "README.md:1 [ ]: # Title",
			wantFile:  "README.md",
			wantLine:  1,
			wantOk:    true,
		},
		{
			name:      "invalid format",
			selection: "not a valid line",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, line, ok := ParseContentSelection(tt.selection)
			if ok != tt.wantOk {
				t.Fatalf("ok: got %v, want %v", ok, tt.wantOk)
			}
			if !tt.wantOk {
				return
			}
			if file != tt.wantFile {
				t.Errorf("file: got %q, want %q", file, tt.wantFile)
			}
			if line != tt.wantLine {
				t.Errorf("line: got %d, want %d", line, tt.wantLine)
			}
		})
	}
}

func TestFindFileIndex(t *testing.T) {
	files := []diff.DiffFile{
		{OldName: "a.go", NewName: "a.go"},
		{OldName: "b.go", NewName: "b.go"},
		{OldName: "old.go", NewName: "", IsDeleted: true},
	}

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "first file", path: "a.go", want: 0},
		{name: "second file", path: "b.go", want: 1},
		{name: "deleted file by OldName", path: "old.go", want: 2},
		{name: "not found", path: "z.go", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindFileIndex(files, tt.path)
			if got != tt.want {
				t.Errorf("FindFileIndex(%q): got %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

func TestFindRowForLine(t *testing.T) {
	pairs := diff.BuildPairedLines([]diff.Hunk{
		{
			OldStart: 10, OldCount: 3, NewStart: 10, NewCount: 3,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 10, NewNum: 10, Content: "ctx"},
				{Type: diff.LineDelete, OldNum: 11, Content: "del"},
				{Type: diff.LineAdd, NewNum: 11, Content: "add"},
				{Type: diff.LineContext, OldNum: 12, NewNum: 12, Content: "ctx2"},
			},
		},
	})

	tests := []struct {
		name    string
		lineNum int
		side    Side
		want    int
	}{
		{name: "context new side", lineNum: 10, side: SideNew, want: 0},
		{name: "add new side", lineNum: 11, side: SideNew, want: 1},
		{name: "delete old side", lineNum: 11, side: SideOld, want: 1},
		{name: "not found", lineNum: 999, side: SideNew, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindRowForLine(pairs, tt.lineNum, tt.side)
			if got != tt.want {
				t.Errorf("FindRowForLine(%d, %v): got %d, want %d", tt.lineNum, tt.side, got, tt.want)
			}
		})
	}
}
