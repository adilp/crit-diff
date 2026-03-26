package main

import (
	"strings"
	"testing"

	"github.com/adil/cr/internal/comment"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantMode    Mode
		wantRefFrom string
		wantRefTo   string
		wantPaths   []string
		wantHelp    bool
		wantSubcmd  string
		wantDetach  bool
		wantWait    bool
	}{
		{
			name:     "no args",
			args:     []string{},
			wantMode: ModeWorkingTree,
		},
		{
			name:        "single ref",
			args:        []string{"main"},
			wantMode:    ModeSingleRef,
			wantRefFrom: "main",
			wantRefTo:   "HEAD",
		},
		{
			name:        "ref range",
			args:        []string{"a1b2..c3d4"},
			wantMode:    ModeRefRange,
			wantRefFrom: "a1b2",
			wantRefTo:   "c3d4",
		},
		{
			name:        "single ref with path filters",
			args:        []string{"main", "--", "src/", "lib/"},
			wantMode:    ModeSingleRef,
			wantRefFrom: "main",
			wantRefTo:   "HEAD",
			wantPaths:   []string{"src/", "lib/"},
		},
		{
			name:      "only path filters",
			args:      []string{"--", "src/"},
			wantMode:  ModeWorkingTree,
			wantPaths: []string{"src/"},
		},
		{
			name:     "--help flag",
			args:     []string{"--help"},
			wantHelp: true,
		},
		{
			name:     "-h flag",
			args:     []string{"-h"},
			wantHelp: true,
		},
		{
			name:        "ref range with path filters",
			args:        []string{"a1b2..c3d4", "--", "src/"},
			wantMode:    ModeRefRange,
			wantRefFrom: "a1b2",
			wantRefTo:   "c3d4",
			wantPaths:   []string{"src/"},
		},
		{
			name:        "numeric shortcut -1",
			args:        []string{"-1"},
			wantMode:    ModeRefRange,
			wantRefFrom: "HEAD~1",
			wantRefTo:   "HEAD",
		},
		{
			name:        "numeric shortcut -3",
			args:        []string{"-3"},
			wantMode:    ModeRefRange,
			wantRefFrom: "HEAD~3",
			wantRefTo:   "HEAD",
		},
		{
			name:        "numeric shortcut -10",
			args:        []string{"-10"},
			wantMode:    ModeRefRange,
			wantRefFrom: "HEAD~10",
			wantRefTo:   "HEAD",
		},
		{
			name:        "numeric shortcut with path filters",
			args:        []string{"-2", "--", "src/"},
			wantMode:    ModeRefRange,
			wantRefFrom: "HEAD~2",
			wantRefTo:   "HEAD",
			wantPaths:   []string{"src/"},
		},
		{
			name:     "--staged flag",
			args:     []string{"--staged"},
			wantMode: ModeStaged,
		},
		{
			name:      "--staged with path filters",
			args:      []string{"--staged", "--", "src/"},
			wantMode:  ModeStaged,
			wantPaths: []string{"src/"},
		},
		{
			name:     "--cached alias for --staged",
			args:     []string{"--cached"},
			wantMode: ModeStaged,
		},
		{
			name:       "status subcommand",
			args:       []string{"status"},
			wantMode:   ModeWorkingTree,
			wantSubcmd: "status",
		},
		{
			name:        "--detach flag with ref",
			args:        []string{"--detach", "main"},
			wantMode:    ModeSingleRef,
			wantRefFrom: "main",
			wantRefTo:   "HEAD",
			wantDetach:  true,
		},
		{
			name:        "--detach --wait with ref range",
			args:        []string{"--detach", "--wait", "a..b"},
			wantMode:    ModeRefRange,
			wantRefFrom: "a",
			wantRefTo:   "b",
			wantDetach:  true,
			wantWait:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseArgs(tt.args)

			if tt.wantHelp {
				if !got.Help {
					t.Error("expected Help to be true")
				}
				return
			}

			if got.Mode != tt.wantMode {
				t.Errorf("Mode: got %q, want %q", got.Mode, tt.wantMode)
			}
			if got.RefFrom != tt.wantRefFrom {
				t.Errorf("RefFrom: got %q, want %q", got.RefFrom, tt.wantRefFrom)
			}
			if got.RefTo != tt.wantRefTo {
				t.Errorf("RefTo: got %q, want %q", got.RefTo, tt.wantRefTo)
			}
			if len(got.PathFilters) != len(tt.wantPaths) {
				t.Fatalf("PathFilters length: got %d, want %d", len(got.PathFilters), len(tt.wantPaths))
			}
			for i, p := range tt.wantPaths {
				if got.PathFilters[i] != p {
					t.Errorf("PathFilters[%d]: got %q, want %q", i, got.PathFilters[i], p)
				}
			}
			if got.Subcmd != tt.wantSubcmd {
				t.Errorf("Subcmd: got %q, want %q", got.Subcmd, tt.wantSubcmd)
			}
			if got.Detach != tt.wantDetach {
				t.Errorf("Detach: got %v, want %v", got.Detach, tt.wantDetach)
			}
			if got.Wait != tt.wantWait {
				t.Errorf("Wait: got %v, want %v", got.Wait, tt.wantWait)
			}
		})
	}
}

func TestRunStatus(t *testing.T) {
	dir := t.TempDir()

	// Write session manifest
	if err := comment.WriteSessionManifest(dir, "HEAD", []string{"test.go", "main.go"}); err != nil {
		t.Fatal(err)
	}

	// Create comments
	store := comment.NewStore(dir)
	if err := store.AddComment("test.go", 10, 0, "some code", "fix this bug"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddComment("main.go", 5, 0, "other code", "rename this"); err != nil {
		t.Fatal(err)
	}

	output, err := runStatus(dir)
	if err != nil {
		t.Fatalf("runStatus error: %v", err)
	}

	// Should contain both comments as JSON
	if !strings.Contains(output, "fix this bug") {
		t.Error("expected output to contain 'fix this bug'")
	}
	if !strings.Contains(output, "rename this") {
		t.Error("expected output to contain 'rename this'")
	}
	if !strings.Contains(output, "test.go") {
		t.Error("expected output to contain file path 'test.go'")
	}
}

func TestRunStatusNoComments(t *testing.T) {
	dir := t.TempDir()

	output, err := runStatus(dir)
	if err != nil {
		t.Fatalf("runStatus error: %v", err)
	}

	// Should output valid JSON with empty comments
	if !strings.Contains(output, "[]") && !strings.Contains(output, "comments") {
		// At minimum should be parseable
		if output == "" {
			t.Error("expected non-empty output")
		}
	}
}

func TestCheckGitRepo_NotARepo(t *testing.T) {
	err := checkGitRepo("/tmp/definitely-not-a-git-repo-" + t.Name())
	if err == nil {
		t.Error("expected error for non-git directory, got nil")
	}
}

func TestCheckGitRepo_ValidRepo(t *testing.T) {
	dir := t.TempDir()
	cmd := execCommand("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init git repo: %s: %v", out, err)
	}
	if err := checkGitRepo(dir); err != nil {
		t.Errorf("expected no error for valid git repo, got: %v", err)
	}
}
