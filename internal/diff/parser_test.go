package diff

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFiles  int
		wantErr    bool
		checkFiles func(t *testing.T, files []DiffFile)
	}{
		{
			name:      "empty input returns empty slice",
			input:     "",
			wantFiles: 0,
		},
		{
			name: "single file with additions and deletions",
			input: `diff --git a/file.txt b/file.txt
index 1234567..abcdefg 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 hello
-world
+universe
+new line
 end
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				f := files[0]
				if f.OldName != "file.txt" || f.NewName != "file.txt" {
					t.Errorf("names: got old=%q new=%q, want file.txt/file.txt", f.OldName, f.NewName)
				}
				if f.IsRename || f.IsBinary || f.IsNew || f.IsDeleted || f.IsSubmodule {
					t.Error("unexpected flags set")
				}
				if len(f.Hunks) != 1 {
					t.Fatalf("expected 1 hunk, got %d", len(f.Hunks))
				}
				h := f.Hunks[0]
				if h.OldStart != 1 || h.OldCount != 3 || h.NewStart != 1 || h.NewCount != 4 {
					t.Errorf("hunk ranges: got old=%d,%d new=%d,%d", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
				}
				// Lines: context("hello"), delete("world"), add("universe"), add("new line"), context("end")
				if len(h.Lines) != 5 {
					t.Fatalf("expected 5 lines, got %d", len(h.Lines))
				}
				// Check context line
				assertLine(t, h.Lines[0], LineContext, 1, 1, "hello")
				// Check delete line
				assertLine(t, h.Lines[1], LineDelete, 2, 0, "world")
				// Check add lines
				assertLine(t, h.Lines[2], LineAdd, 0, 2, "universe")
				assertLine(t, h.Lines[3], LineAdd, 0, 3, "new line")
				// Check trailing context
				assertLine(t, h.Lines[4], LineContext, 3, 4, "end")
			},
		},
		{
			name: "multiple files",
			input: `diff --git a/a.txt b/a.txt
index 1234567..abcdefg 100644
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-old
+new
diff --git a/b.txt b/b.txt
index 1234567..abcdefg 100644
--- a/b.txt
+++ b/b.txt
@@ -1 +1,2 @@
 keep
+added
`,
			wantFiles: 2,
			checkFiles: func(t *testing.T, files []DiffFile) {
				if files[0].NewName != "a.txt" {
					t.Errorf("first file: got %q, want a.txt", files[0].NewName)
				}
				if files[1].NewName != "b.txt" {
					t.Errorf("second file: got %q, want b.txt", files[1].NewName)
				}
			},
		},
		{
			name: "renamed file",
			input: `diff --git a/old.txt b/new.txt
similarity index 100%
rename from old.txt
rename to new.txt
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				f := files[0]
				if !f.IsRename {
					t.Error("expected IsRename=true")
				}
				if f.OldName != "old.txt" {
					t.Errorf("OldName: got %q, want old.txt", f.OldName)
				}
				if f.NewName != "new.txt" {
					t.Errorf("NewName: got %q, want new.txt", f.NewName)
				}
			},
		},
		{
			name: "binary file",
			input: `diff --git a/image.png b/image.png
index 1234567..abcdefg 100644
Binary files a/image.png and b/image.png differ
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				f := files[0]
				if !f.IsBinary {
					t.Error("expected IsBinary=true")
				}
				if f.NewName != "image.png" {
					t.Errorf("NewName: got %q, want image.png", f.NewName)
				}
				if len(f.Hunks) != 0 {
					t.Errorf("expected 0 hunks for binary, got %d", len(f.Hunks))
				}
			},
		},
		{
			name: "new file",
			input: `diff --git a/new.txt b/new.txt
new file mode 100644
index 0000000..abcdefg
--- /dev/null
+++ b/new.txt
@@ -0,0 +1,2 @@
+line one
+line two
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				f := files[0]
				if !f.IsNew {
					t.Error("expected IsNew=true")
				}
				if f.NewName != "new.txt" {
					t.Errorf("NewName: got %q, want new.txt", f.NewName)
				}
				if len(f.Hunks) != 1 {
					t.Fatalf("expected 1 hunk, got %d", len(f.Hunks))
				}
				if len(f.Hunks[0].Lines) != 2 {
					t.Fatalf("expected 2 lines, got %d", len(f.Hunks[0].Lines))
				}
				assertLine(t, f.Hunks[0].Lines[0], LineAdd, 0, 1, "line one")
				assertLine(t, f.Hunks[0].Lines[1], LineAdd, 0, 2, "line two")
			},
		},
		{
			name: "deleted file",
			input: `diff --git a/gone.txt b/gone.txt
deleted file mode 100644
index abcdefg..0000000
--- a/gone.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line one
-line two
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				f := files[0]
				if !f.IsDeleted {
					t.Error("expected IsDeleted=true")
				}
				if f.OldName != "gone.txt" {
					t.Errorf("OldName: got %q, want gone.txt", f.OldName)
				}
				if len(f.Hunks[0].Lines) != 2 {
					t.Fatalf("expected 2 lines, got %d", len(f.Hunks[0].Lines))
				}
				assertLine(t, f.Hunks[0].Lines[0], LineDelete, 1, 0, "line one")
				assertLine(t, f.Hunks[0].Lines[1], LineDelete, 2, 0, "line two")
			},
		},
		{
			name: "mode change",
			input: `diff --git a/script.sh b/script.sh
old mode 100644
new mode 100755
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				f := files[0]
				// go-gitdiff preserves full git mode (100644/100755)
				if f.OldMode != 0100644 {
					t.Errorf("OldMode: got %o, want 100644", f.OldMode)
				}
				if f.NewMode != 0100755 {
					t.Errorf("NewMode: got %o, want 100755", f.NewMode)
				}
			},
		},
		{
			name: "no newline at EOF",
			input: `diff --git a/file.txt b/file.txt
index 1234567..abcdefg 100644
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-old
\ No newline at end of file
+new
\ No newline at end of file
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				h := files[0].Hunks[0]
				if len(h.Lines) != 2 {
					t.Fatalf("expected 2 lines, got %d", len(h.Lines))
				}
				if !h.Lines[0].NoEOL {
					t.Error("expected NoEOL=true on deleted line")
				}
				if !h.Lines[1].NoEOL {
					t.Error("expected NoEOL=true on added line")
				}
			},
		},
		{
			name: "submodule diff",
			input: `diff --git a/vendor/lib b/vendor/lib
index abc1234..def5678 160000
--- a/vendor/lib
+++ b/vendor/lib
@@ -1 +1 @@
-Subproject commit abc1234567890
+Subproject commit def5678901234
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				f := files[0]
				if !f.IsSubmodule {
					t.Error("expected IsSubmodule=true")
				}
				if f.NewName != "vendor/lib" {
					t.Errorf("NewName: got %q, want vendor/lib", f.NewName)
				}
			},
		},
		{
			name: "multiple hunks in one file",
			input: `diff --git a/file.txt b/file.txt
index 1234567..abcdefg 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 aaa
-bbb
+BBB
 ccc
@@ -10,3 +10,3 @@
 xxx
-yyy
+YYY
 zzz
`,
			wantFiles: 1,
			checkFiles: func(t *testing.T, files []DiffFile) {
				if len(files[0].Hunks) != 2 {
					t.Fatalf("expected 2 hunks, got %d", len(files[0].Hunks))
				}
				h1 := files[0].Hunks[0]
				h2 := files[0].Hunks[1]
				if h1.OldStart != 1 || h1.NewStart != 1 {
					t.Errorf("hunk1: old=%d new=%d", h1.OldStart, h1.NewStart)
				}
				if h2.OldStart != 10 || h2.NewStart != 10 {
					t.Errorf("hunk2: old=%d new=%d", h2.OldStart, h2.NewStart)
				}
				// Check line numbers in second hunk
				assertLine(t, h2.Lines[0], LineContext, 10, 10, "xxx")
				assertLine(t, h2.Lines[1], LineDelete, 11, 0, "yyy")
				assertLine(t, h2.Lines[2], LineAdd, 0, 11, "YYY")
				assertLine(t, h2.Lines[3], LineContext, 12, 12, "zzz")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(files) != tt.wantFiles {
				t.Fatalf("expected %d files, got %d", tt.wantFiles, len(files))
			}
			if tt.checkFiles != nil {
				tt.checkFiles(t, files)
			}
		})
	}
}

func assertLine(t *testing.T, line DiffLine, wantType LineType, wantOld, wantNew int, wantContent string) {
	t.Helper()
	if line.Type != wantType {
		t.Errorf("line type: got %d, want %d", line.Type, wantType)
	}
	if line.OldNum != wantOld {
		t.Errorf("OldNum: got %d, want %d (content=%q)", line.OldNum, wantOld, line.Content)
	}
	if line.NewNum != wantNew {
		t.Errorf("NewNum: got %d, want %d (content=%q)", line.NewNum, wantNew, line.Content)
	}
	if line.Content != wantContent {
		t.Errorf("Content: got %q, want %q", line.Content, wantContent)
	}
}
