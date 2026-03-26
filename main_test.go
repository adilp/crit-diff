package main

import (
	"testing"
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
		})
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
