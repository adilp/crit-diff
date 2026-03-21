package ui

import (
	"strings"
	"testing"

	"github.com/adil/cr/internal/diff"
	tea "github.com/charmbracelet/bubbletea"
)

func TestBuildTree(t *testing.T) {
	tests := []struct {
		name       string
		files      []diff.DiffFile
		wantPaths  []string // display paths in order
		wantIsDir  []bool
		wantDepths []int
	}{
		{
			name:       "single file at root",
			files:      []diff.DiffFile{{NewName: "main.go"}},
			wantPaths:  []string{"main.go"},
			wantIsDir:  []bool{false},
			wantDepths: []int{0},
		},
		{
			name: "files in same directory",
			files: []diff.DiffFile{
				{NewName: "src/foo.go"},
				{NewName: "src/bar.go"},
			},
			wantPaths:  []string{"src/", "foo.go", "bar.go"},
			wantIsDir:  []bool{true, false, false},
			wantDepths: []int{0, 1, 1},
		},
		{
			name: "compact single-child directory chains",
			files: []diff.DiffFile{
				{NewName: "src/services/auth/handler.go"},
			},
			wantPaths:  []string{"src/services/auth/", "handler.go"},
			wantIsDir:  []bool{true, false},
			wantDepths: []int{0, 1},
		},
		{
			name: "no compact when dir has multiple children",
			files: []diff.DiffFile{
				{NewName: "src/foo.go"},
				{NewName: "src/services/bar.go"},
			},
			wantPaths:  []string{"src/", "foo.go", "services/", "bar.go"},
			wantIsDir:  []bool{true, false, true, false},
			wantDepths: []int{0, 1, 1, 2},
		},
		{
			name: "preserves diff order",
			files: []diff.DiffFile{
				{NewName: "b.go"},
				{NewName: "a.go"},
			},
			wantPaths:  []string{"b.go", "a.go"},
			wantIsDir:  []bool{false, false},
			wantDepths: []int{0, 0},
		},
		{
			name: "binary file indicator",
			files: []diff.DiffFile{
				{NewName: "image.png", IsBinary: true},
			},
			wantPaths: []string{"image.png"},
			wantIsDir: []bool{false},
		},
		{
			name: "rename indicator",
			files: []diff.DiffFile{
				{OldName: "old.go", NewName: "new.go", IsRename: true},
			},
			wantPaths: []string{"new.go"},
			wantIsDir: []bool{false},
		},
		{
			name: "deleted file uses OldName",
			files: []diff.DiffFile{
				{OldName: "removed.go", NewName: "", IsDeleted: true},
			},
			wantPaths: []string{"removed.go"},
			wantIsDir: []bool{false},
		},
		{
			name:       "empty file list",
			files:      nil,
			wantPaths:  nil,
			wantIsDir:  nil,
			wantDepths: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTreeState(tt.files)
			entries := ts.Entries

			if len(entries) != len(tt.wantPaths) {
				t.Fatalf("entry count: got %d, want %d", len(entries), len(tt.wantPaths))
			}

			for i, e := range entries {
				if e.Path != tt.wantPaths[i] {
					t.Errorf("entries[%d].Path: got %q, want %q", i, e.Path, tt.wantPaths[i])
				}
				if e.IsDir != tt.wantIsDir[i] {
					t.Errorf("entries[%d].IsDir: got %v, want %v", i, e.IsDir, tt.wantIsDir[i])
				}
				if tt.wantDepths != nil && e.Depth != tt.wantDepths[i] {
					t.Errorf("entries[%d].Depth: got %d, want %d", i, e.Depth, tt.wantDepths[i])
				}
			}
		})
	}
}

func TestBuildTreeFileIndex(t *testing.T) {
	files := []diff.DiffFile{
		{NewName: "src/foo.go"},
		{NewName: "src/bar.go"},
	}
	ts := NewTreeState(files)

	// Directories should have FileIndex -1
	for _, e := range ts.Entries {
		if e.IsDir && e.FileIndex != -1 {
			t.Errorf("dir entry %q should have FileIndex -1, got %d", e.Path, e.FileIndex)
		}
	}

	// File entries should map to correct file indices
	fileEntries := []TreeEntry{}
	for _, e := range ts.Entries {
		if !e.IsDir {
			fileEntries = append(fileEntries, e)
		}
	}
	if len(fileEntries) != 2 {
		t.Fatalf("expected 2 file entries, got %d", len(fileEntries))
	}
	if fileEntries[0].FileIndex != 0 {
		t.Errorf("first file FileIndex: got %d, want 0", fileEntries[0].FileIndex)
	}
	if fileEntries[1].FileIndex != 1 {
		t.Errorf("second file FileIndex: got %d, want 1", fileEntries[1].FileIndex)
	}
}

func TestBuildTreeBinaryAndRenameFlags(t *testing.T) {
	files := []diff.DiffFile{
		{NewName: "image.png", IsBinary: true},
		{OldName: "old.go", NewName: "new.go", IsRename: true},
	}
	ts := NewTreeState(files)

	fileEntries := []TreeEntry{}
	for _, e := range ts.Entries {
		if !e.IsDir {
			fileEntries = append(fileEntries, e)
		}
	}
	if len(fileEntries) != 2 {
		t.Fatalf("expected 2 file entries, got %d", len(fileEntries))
	}
	if !fileEntries[0].IsBinary {
		t.Error("image.png should be marked as binary")
	}
	if !fileEntries[1].IsRename {
		t.Error("new.go should be marked as rename")
	}
}

func TestTreeCollapse(t *testing.T) {
	files := []diff.DiffFile{
		{NewName: "src/foo.go"},
		{NewName: "src/bar.go"},
	}
	ts := NewTreeState(files)

	// Initially not collapsed
	visible := ts.VisibleEntries()
	if len(visible) != 3 { // src/, foo.go, bar.go
		t.Fatalf("expected 3 visible entries, got %d", len(visible))
	}

	// Collapse the directory (fullPath is "src" for top-level dir)
	ts.ToggleCollapse("src")
	visible = ts.VisibleEntries()
	if len(visible) != 1 { // just src/
		t.Fatalf("after collapse: expected 1 visible entry, got %d", len(visible))
	}

	// Expand again
	ts.ToggleCollapse("src")
	visible = ts.VisibleEntries()
	if len(visible) != 3 {
		t.Fatalf("after expand: expected 3 visible entries, got %d", len(visible))
	}
}

func TestTreeCursorNavigation(t *testing.T) {
	files := []diff.DiffFile{
		{NewName: "a.go"},
		{NewName: "b.go"},
		{NewName: "c.go"},
	}
	ts := NewTreeState(files)

	if ts.Cursor != 0 {
		t.Errorf("initial cursor: got %d, want 0", ts.Cursor)
	}

	ts.CursorDown()
	if ts.Cursor != 1 {
		t.Errorf("after CursorDown: got %d, want 1", ts.Cursor)
	}

	ts.CursorDown()
	ts.CursorDown() // should clamp at last entry
	if ts.Cursor != 2 {
		t.Errorf("cursor should clamp at 2, got %d", ts.Cursor)
	}

	ts.CursorUp()
	if ts.Cursor != 1 {
		t.Errorf("after CursorUp: got %d, want 1", ts.Cursor)
	}

	ts.CursorUp()
	ts.CursorUp() // should clamp at 0
	if ts.Cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", ts.Cursor)
	}
}

func TestTreeSelectedFileIndex(t *testing.T) {
	files := []diff.DiffFile{
		{NewName: "src/foo.go"},
		{NewName: "src/bar.go"},
	}
	ts := NewTreeState(files)
	// entries: src/, foo.go, bar.go
	// cursor 0 = src/ (dir, no file)
	if ts.SelectedFileIndex() != -1 {
		t.Errorf("dir entry should return -1, got %d", ts.SelectedFileIndex())
	}

	ts.CursorDown() // cursor 1 = foo.go
	if ts.SelectedFileIndex() != 0 {
		t.Errorf("foo.go should have FileIndex 0, got %d", ts.SelectedFileIndex())
	}

	ts.CursorDown() // cursor 2 = bar.go
	if ts.SelectedFileIndex() != 1 {
		t.Errorf("bar.go should have FileIndex 1, got %d", ts.SelectedFileIndex())
	}
}

func TestTreeWidthMinimum(t *testing.T) {
	if w := TreeWidth(30); w < 10 {
		t.Errorf("TreeWidth(30) = %d, want >= 10", w)
	}
	if w := TreeWidth(200); w != 40 {
		t.Errorf("TreeWidth(200) = %d, want 40", w)
	}
}

func TestTreeCollapseDistinctPaths(t *testing.T) {
	// Two directories with same name at different locations should not collide
	files := []diff.DiffFile{
		{NewName: "a/src/foo.go"},
		{NewName: "b/src/bar.go"},
	}
	ts := NewTreeState(files)

	// Find the "a/src/" entry and collapse it
	for _, e := range ts.Entries {
		if e.IsDir && strings.HasPrefix(e.FullPath, "a") {
			ts.ToggleCollapse(e.FullPath)
			break
		}
	}

	// b/src/bar.go should still be visible
	visible := ts.VisibleEntries()
	found := false
	for _, e := range visible {
		if !e.IsDir && e.Path == "bar.go" {
			found = true
		}
	}
	if !found {
		t.Error("collapsing a/src/ should not hide b/src/bar.go")
	}
}

func TestRenderTree(t *testing.T) {
	tests := []struct {
		name       string
		files      []diff.DiffFile
		activeFile int
		width      int
		height     int
		contains   []string
		absent     []string
	}{
		{
			name: "renders file names",
			files: []diff.DiffFile{
				{NewName: "foo.go"},
				{NewName: "bar.go"},
			},
			activeFile: 0,
			width:      20,
			height:     10,
			contains:   []string{"foo.go", "bar.go"},
		},
		{
			name: "shows active file indicator",
			files: []diff.DiffFile{
				{NewName: "foo.go"},
				{NewName: "bar.go"},
			},
			activeFile: 0,
			width:      20,
			height:     10,
			contains:   []string{"●"},
		},
		{
			name: "shows binary indicator",
			files: []diff.DiffFile{
				{NewName: "image.png", IsBinary: true},
			},
			activeFile: 0,
			width:      30,
			height:     10,
			contains:   []string{"[bin]"},
		},
		{
			name: "shows rename indicator",
			files: []diff.DiffFile{
				{OldName: "old.go", NewName: "new.go", IsRename: true},
			},
			activeFile: 0,
			width:      30,
			height:     10,
			contains:   []string{"→"},
		},
		{
			name: "shows directory with trailing slash",
			files: []diff.DiffFile{
				{NewName: "src/foo.go"},
			},
			activeFile: 0,
			width:      30,
			height:     10,
			contains:   []string{"src/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTreeState(tt.files)
			output := RenderTree(&ts, tt.width, tt.height, tt.activeFile)
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("expected output to contain %q, got:\n%s", s, output)
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

func TestRenderTreeScrolling(t *testing.T) {
	files := make([]diff.DiffFile, 20)
	for i := range files {
		files[i] = diff.DiffFile{NewName: "file" + string(rune('a'+i)) + ".go"}
	}
	ts := NewTreeState(files)
	// Move cursor to entry 15
	for i := 0; i < 15; i++ {
		ts.CursorDown()
	}

	output := RenderTree(&ts, 20, 5, 0)
	// The cursor entry should be visible
	if !strings.Contains(output, files[15].NewName) {
		t.Errorf("scrolled tree should show cursor entry %q", files[15].NewName)
	}
}

// Integration tests: tree mode in app.go

func TestTabTogglesTree(t *testing.T) {
	m := newMultiFileTestModel(120, 40)

	// Initially tree is closed, mode is normal
	if m.mode != InputModeNormal {
		t.Errorf("initial mode: got %v, want normal", m.mode)
	}
	if m.tree.Open {
		t.Error("tree should be closed initially")
	}

	// Press Tab to open tree
	newModel, _ := m.Update(tabKeyMsg())
	m = newModel.(Model)
	if m.mode != InputModeTree {
		t.Errorf("after Tab: mode got %v, want tree", m.mode)
	}
	if !m.tree.Open {
		t.Error("after Tab: tree should be open")
	}

	// Press Tab again to close tree
	newModel, _ = m.Update(tabKeyMsg())
	m = newModel.(Model)
	if m.mode != InputModeNormal {
		t.Errorf("after 2nd Tab: mode got %v, want normal", m.mode)
	}
	if m.tree.Open {
		t.Error("after 2nd Tab: tree should be closed")
	}
}

func TestTreeModeJKNavigation(t *testing.T) {
	m := newMultiFileTestModel(120, 40)

	// Open tree
	newModel, _ := m.Update(tabKeyMsg())
	m = newModel.(Model)

	if m.tree.Cursor != 0 {
		t.Errorf("tree cursor initial: got %d, want 0", m.tree.Cursor)
	}

	// j moves tree cursor down
	newModel, _ = m.Update(keyMsg("j"))
	m = newModel.(Model)
	if m.tree.Cursor != 1 {
		t.Errorf("tree cursor after j: got %d, want 1", m.tree.Cursor)
	}

	// k moves tree cursor up
	newModel, _ = m.Update(keyMsg("k"))
	m = newModel.(Model)
	if m.tree.Cursor != 0 {
		t.Errorf("tree cursor after k: got %d, want 0", m.tree.Cursor)
	}
}

func TestTreeModeLOpensFile(t *testing.T) {
	m := newMultiFileTestModel(120, 40)

	// Open tree
	newModel, _ := m.Update(tabKeyMsg())
	m = newModel.(Model)

	// Move to second file entry (file1.go is at index 0, file2.go at index 1)
	m.tree.Cursor = 1

	// Press l to open
	newModel, _ = m.Update(keyMsg("l"))
	m = newModel.(Model)

	// Should have switched to file2 and closed tree
	if m.tree.Open {
		t.Error("tree should be closed after opening file")
	}
	if m.mode != InputModeNormal {
		t.Errorf("mode should be normal after opening file, got %v", m.mode)
	}
	if m.activeFile != 1 {
		t.Errorf("activeFile: got %d, want 1", m.activeFile)
	}
}

func TestTreeModeEnterOpensFile(t *testing.T) {
	m := newMultiFileTestModel(120, 40)

	// Open tree, move to file2
	newModel, _ := m.Update(tabKeyMsg())
	m = newModel.(Model)
	m.tree.Cursor = 1

	// Press Enter
	newModel, _ = m.Update(enterKeyMsg())
	m = newModel.(Model)

	if m.tree.Open {
		t.Error("tree should be closed after Enter")
	}
	if m.activeFile != 1 {
		t.Errorf("activeFile: got %d, want 1", m.activeFile)
	}
}

func TestTreeModeEscClosesTree(t *testing.T) {
	m := newMultiFileTestModel(120, 40)

	// Open tree
	newModel, _ := m.Update(tabKeyMsg())
	m = newModel.(Model)

	// Press Esc
	newModel, _ = m.Update(escKeyMsg())
	m = newModel.(Model)

	if m.tree.Open {
		t.Error("tree should be closed after Esc")
	}
	if m.mode != InputModeNormal {
		t.Errorf("mode should be normal after Esc, got %v", m.mode)
	}
}

func TestTreeModeHCollapsesDirectory(t *testing.T) {
	files := []diff.DiffFile{
		{NewName: "src/foo.go"},
		{NewName: "src/bar.go"},
	}
	allPaired := make([][]diff.PairedLine, len(files))
	for i, f := range files {
		allPaired[i] = diff.BuildPairedLines(f.Hunks)
	}
	m := NewModel(files, allPaired[0], 120, 40)
	m.allPaired = allPaired

	// Open tree
	newModel, _ := m.Update(tabKeyMsg())
	m = newModel.(Model)

	// Cursor is on src/ (dir), press h to collapse
	newModel, _ = m.Update(keyMsg("h"))
	m = newModel.(Model)

	visible := m.tree.VisibleEntries()
	if len(visible) != 1 {
		t.Errorf("after collapse: expected 1 visible entry, got %d", len(visible))
	}
}

func TestTreeModeHelpBar(t *testing.T) {
	helpBar := RenderHelpBar(80, InputModeTree)
	if !strings.Contains(helpBar, "j/k") {
		t.Error("tree mode help bar should contain j/k")
	}
	if !strings.Contains(helpBar, "tree") {
		t.Error("tree mode help bar should contain 'tree'")
	}
	if !strings.Contains(helpBar, "open") {
		t.Error("tree mode help bar should contain 'open'")
	}
	if !strings.Contains(helpBar, "Tab") {
		t.Error("tree mode help bar should contain Tab")
	}
}

func TestTreeViewRendering(t *testing.T) {
	m := newMultiFileTestModel(120, 40)

	// Open tree
	newModel, _ := m.Update(tabKeyMsg())
	m = newModel.(Model)

	output := m.View()
	if output == "" {
		t.Fatal("View() should not be empty with tree open")
	}
	// Should contain file names from tree
	if !strings.Contains(output, "file1.go") {
		t.Error("view should contain file1.go from tree")
	}
}

func TestTreeModeQQuits(t *testing.T) {
	m := newMultiFileTestModel(120, 40)

	// Open tree
	newModel, _ := m.Update(tabKeyMsg())
	m = newModel.(Model)

	// q should still quit
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q in tree mode should produce quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}
