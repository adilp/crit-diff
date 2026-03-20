package diff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temp git repo with known commits for testing.
// Returns the repo path. Structure:
//   - initial commit: file.txt ("hello\n"), src/app.go ("package main\n")
//   - second commit (on main): file.txt ("hello\nworld\n"), src/app.go ("package app\n")
//   - working tree: file.txt has unstaged change ("hello\nworld\nlocal\n")
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s: %v", args, out, err)
		}
	}

	run("git", "init", "-b", "main")

	// Initial commit
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "app.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-m", "initial commit")

	// Tag the initial commit so we have a ref to diff against
	run("git", "tag", "v0")

	// Second commit: modify both files
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "app.go"), []byte("package app\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-m", "add world line and rename package")

	// Unstaged working tree change
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\nworld\nlocal\n"), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestGetDiff(t *testing.T) {
	tests := []struct {
		name      string
		args      DiffArgs
		wantErr   bool
		contains  []string // substrings expected in output
		excludes  []string // substrings that must NOT appear
		wantEmpty bool
	}{
		{
			name:     "working tree (no args) shows unstaged changes",
			args:     DiffArgs{},
			contains: []string{"+local"},
		},
		{
			name:     "single ref diffs ref..HEAD",
			args:     DiffArgs{RefRange: "v0"},
			contains: []string{"+world"},
		},
		{
			name:     "ref range diffs refA..refB",
			args:     DiffArgs{RefRange: "v0..HEAD"},
			contains: []string{"+world"},
		},
		{
			name:     "path filter excludes non-matching files",
			args:     DiffArgs{RefRange: "v0", Paths: []string{"src/"}},
			contains: []string{"app.go"},
			excludes: []string{"file.txt"},
		},
		{
			name:     "path filter includes matching changes",
			args:     DiffArgs{RefRange: "v0", Paths: []string{"file.txt"}},
			contains: []string{"+world", "file.txt"},
			excludes: []string{"app.go"},
		},
		{
			name:     "working tree with path filter",
			args:     DiffArgs{Paths: []string{"file.txt"}},
			contains: []string{"+local"},
		},
		{
			name:    "invalid ref returns error",
			args:    DiffArgs{RefRange: "nonexistent_ref_xyz"},
			wantErr: true,
		},
		{
			name:      "empty diff returns empty string",
			args:      DiffArgs{RefRange: "HEAD..HEAD"},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTestRepo(t)

			got, err := GetDiff(tt.args, dir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("expected empty string, got: %q", got)
				}
				return
			}

			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("expected output to contain %q, got:\n%s", s, got)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(got, s) {
					t.Errorf("expected output to NOT contain %q, got:\n%s", s, got)
				}
			}
		})
	}
}

func TestGetRepoRoot(t *testing.T) {
	dir := setupTestRepo(t)

	root, err := GetRepoRoot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resolve symlinks for macOS /private/tmp vs /tmp
	wantResolved, _ := filepath.EvalSymlinks(dir)
	gotResolved, _ := filepath.EvalSymlinks(root)

	if gotResolved != wantResolved {
		t.Errorf("GetRepoRoot: got %q, want %q", gotResolved, wantResolved)
	}
}

func TestGetFileContent(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "file at HEAD",
			ref:  "HEAD",
			path: "file.txt",
			want: "hello\nworld\n",
		},
		{
			name: "file at tag",
			ref:  "v0",
			path: "file.txt",
			want: "hello\n",
		},
		{
			name: "go file at HEAD",
			ref:  "HEAD",
			path: "src/app.go",
			want: "package app\n",
		},
		{
			name:    "nonexistent file returns error",
			ref:     "HEAD",
			path:    "nosuchfile.txt",
			wantErr: true,
		},
		{
			name:    "invalid ref returns error",
			ref:     "nonexistent_ref_xyz",
			path:    "file.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTestRepo(t)

			got, err := GetFileContent(tt.ref, tt.path, dir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("GetFileContent = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetRepoRoot_NotARepo(t *testing.T) {
	dir := t.TempDir()

	_, err := GetRepoRoot(dir)
	if err == nil {
		t.Error("expected error for non-git directory, got nil")
	}
}
