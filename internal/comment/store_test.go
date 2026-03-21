package comment

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestAddComment(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		line    int
		endLine int
		snippet string
		body    string
		check   func(t *testing.T, s *Store, err error)
	}{
		{
			name:    "adds single-line comment and persists to disk",
			file:    "src/main.go",
			line:    42,
			endLine: 0,
			snippet: "func main() {",
			body:    "Consider renaming this",
			check: func(t *testing.T, s *Store, err error) {
				if err != nil {
					t.Fatalf("AddComment: %v", err)
				}
				comments := s.Comments("src/main.go")
				if len(comments) != 1 {
					t.Fatalf("expected 1 comment, got %d", len(comments))
				}
				c := comments[0]
				if len(c.ID) != 8 {
					t.Errorf("ID length: got %d, want 8", len(c.ID))
				}
				if c.Line != 42 {
					t.Errorf("Line: got %d, want 42", c.Line)
				}
				if c.EndLine != 0 {
					t.Errorf("EndLine: got %d, want 0", c.EndLine)
				}
				if c.ContentSnippet != "func main() {" {
					t.Errorf("ContentSnippet: got %q, want %q", c.ContentSnippet, "func main() {")
				}
				if c.Body != "Consider renaming this" {
					t.Errorf("Body: got %q, want %q", c.Body, "Consider renaming this")
				}
				if c.CreatedAt.IsZero() {
					t.Error("CreatedAt is zero")
				}
				// Verify persisted to disk
				path := reviewPath(s.repoRoot, "src/main.go")
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Error("review YAML file not created on disk")
				}
			},
		},
		{
			name:    "adds range comment with end_line",
			file:    "src/main.go",
			line:    10,
			endLine: 15,
			snippet: "if err != nil {",
			body:    "This block needs refactoring",
			check: func(t *testing.T, s *Store, err error) {
				if err != nil {
					t.Fatalf("AddComment: %v", err)
				}
				comments := s.Comments("src/main.go")
				if len(comments) != 1 {
					t.Fatalf("expected 1 comment, got %d", len(comments))
				}
				if comments[0].EndLine != 15 {
					t.Errorf("EndLine: got %d, want 15", comments[0].EndLine)
				}
			},
		},
		{
			name:    "adds multiple comments to same file",
			file:    "src/main.go",
			line:    1,
			endLine: 0,
			snippet: "package main",
			body:    "first",
			check: func(t *testing.T, s *Store, err error) {
				if err != nil {
					t.Fatalf("AddComment: %v", err)
				}
				// Add a second comment
				err = s.AddComment("src/main.go", 5, 0, "import", "second")
				if err != nil {
					t.Fatalf("AddComment second: %v", err)
				}
				comments := s.Comments("src/main.go")
				if len(comments) != 2 {
					t.Fatalf("expected 2 comments, got %d", len(comments))
				}
				if comments[0].Body != "first" {
					t.Errorf("first comment Body: got %q, want %q", comments[0].Body, "first")
				}
				if comments[1].Body != "second" {
					t.Errorf("second comment Body: got %q, want %q", comments[1].Body, "second")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			s := NewStore(dir)
			err := s.AddComment(tt.file, tt.line, tt.endLine, tt.snippet, tt.body)
			tt.check(t, s, err)
		})
	}
}

func TestEditComment(t *testing.T) {
	tests := []struct {
		name  string
		check func(t *testing.T, s *Store)
	}{
		{
			name: "edits existing comment body and persists",
			check: func(t *testing.T, s *Store) {
				err := s.AddComment("f.go", 1, 0, "line", "original")
				if err != nil {
					t.Fatalf("AddComment: %v", err)
				}
				id := s.Comments("f.go")[0].ID
				err = s.EditComment(id, "updated")
				if err != nil {
					t.Fatalf("EditComment: %v", err)
				}
				comments := s.Comments("f.go")
				if len(comments) != 1 {
					t.Fatalf("expected 1 comment, got %d", len(comments))
				}
				if comments[0].Body != "updated" {
					t.Errorf("Body: got %q, want %q", comments[0].Body, "updated")
				}
			},
		},
		{
			name: "returns ErrNotFound for nonexistent comment ID",
			check: func(t *testing.T, s *Store) {
				err := s.EditComment("nonexist", "new body")
				if !errors.Is(err, ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "edit persists to disk and survives round-trip",
			check: func(t *testing.T, s *Store) {
				_ = s.AddComment("f.go", 10, 0, "code", "original")
				id := s.Comments("f.go")[0].ID
				_ = s.EditComment(id, "edited body")

				s2 := NewStore(s.repoRoot)
				_ = s2.LoadComments("f.go")
				loaded := s2.Comments("f.go")
				if len(loaded) != 1 {
					t.Fatalf("expected 1 comment, got %d", len(loaded))
				}
				if loaded[0].Body != "edited body" {
					t.Errorf("Body after round-trip: got %q, want %q", loaded[0].Body, "edited body")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			s := NewStore(dir)
			tt.check(t, s)
		})
	}
}

func TestDeleteComment(t *testing.T) {
	tests := []struct {
		name  string
		check func(t *testing.T, s *Store)
	}{
		{
			name: "deletes existing comment and persists",
			check: func(t *testing.T, s *Store) {
				err := s.AddComment("f.go", 1, 0, "line", "to delete")
				if err != nil {
					t.Fatalf("AddComment: %v", err)
				}
				id := s.Comments("f.go")[0].ID
				err = s.DeleteComment(id)
				if err != nil {
					t.Fatalf("DeleteComment: %v", err)
				}
				comments := s.Comments("f.go")
				if len(comments) != 0 {
					t.Errorf("expected 0 comments, got %d", len(comments))
				}
			},
		},
		{
			name: "returns ErrNotFound for nonexistent comment ID",
			check: func(t *testing.T, s *Store) {
				err := s.DeleteComment("nonexist")
				if !errors.Is(err, ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "delete persists to disk and survives round-trip",
			check: func(t *testing.T, s *Store) {
				_ = s.AddComment("f.go", 1, 0, "a", "keep")
				_ = s.AddComment("f.go", 2, 0, "b", "remove")
				id := s.Comments("f.go")[1].ID
				_ = s.DeleteComment(id)

				s2 := NewStore(s.repoRoot)
				_ = s2.LoadComments("f.go")
				loaded := s2.Comments("f.go")
				if len(loaded) != 1 {
					t.Fatalf("expected 1 comment, got %d", len(loaded))
				}
				if loaded[0].Body != "keep" {
					t.Errorf("Body after round-trip: got %q, want %q", loaded[0].Body, "keep")
				}
			},
		},
		{
			name: "deletes correct comment when multiple exist",
			check: func(t *testing.T, s *Store) {
				_ = s.AddComment("f.go", 1, 0, "a", "first")
				_ = s.AddComment("f.go", 2, 0, "b", "second")
				_ = s.AddComment("f.go", 3, 0, "c", "third")
				id := s.Comments("f.go")[1].ID // delete "second"
				err := s.DeleteComment(id)
				if err != nil {
					t.Fatalf("DeleteComment: %v", err)
				}
				comments := s.Comments("f.go")
				if len(comments) != 2 {
					t.Fatalf("expected 2 comments, got %d", len(comments))
				}
				if comments[0].Body != "first" {
					t.Errorf("comments[0].Body: got %q, want %q", comments[0].Body, "first")
				}
				if comments[1].Body != "third" {
					t.Errorf("comments[1].Body: got %q, want %q", comments[1].Body, "third")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			s := NewStore(dir)
			tt.check(t, s)
		})
	}
}

func TestComments(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// No comments for a file returns empty slice
	comments := s.Comments("nonexist.go")
	if comments == nil {
		t.Error("Comments for unknown file should return empty slice, not nil")
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}

func TestLoadComments(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
		file  string
		check func(t *testing.T, s *Store, err error)
	}{
		{
			name: "loads existing crit-format YAML file",
			setup: func(t *testing.T, dir string) {
				reviewDir := filepath.Join(dir, ".crit", "reviews")
				if err := os.MkdirAll(reviewDir, 0o755); err != nil {
					t.Fatal(err)
				}
				state := ReviewState{
					File: "src/app.go",
					Comments: []Comment{
						{
							ID:             "a1b2c3d4",
							Line:           10,
							EndLine:        0,
							ContentSnippet: "func foo() {",
							Body:           "Why is this public?",
							CreatedAt:      time.Date(2026, 3, 19, 14, 30, 0, 0, time.UTC),
						},
					},
				}
				data, _ := yaml.Marshal(&state)
				path := reviewPath(dir, "src/app.go")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
			},
			file: "src/app.go",
			check: func(t *testing.T, s *Store, err error) {
				if err != nil {
					t.Fatalf("LoadComments: %v", err)
				}
				comments := s.Comments("src/app.go")
				if len(comments) != 1 {
					t.Fatalf("expected 1 comment, got %d", len(comments))
				}
				c := comments[0]
				if c.ID != "a1b2c3d4" {
					t.Errorf("ID: got %q, want %q", c.ID, "a1b2c3d4")
				}
				if c.Body != "Why is this public?" {
					t.Errorf("Body: got %q, want %q", c.Body, "Why is this public?")
				}
			},
		},
		{
			name:  "no file returns no error and no comments",
			setup: func(t *testing.T, dir string) {},
			file:  "missing.go",
			check: func(t *testing.T, s *Store, err error) {
				if err != nil {
					t.Fatalf("LoadComments: %v", err)
				}
				comments := s.Comments("missing.go")
				if len(comments) != 0 {
					t.Errorf("expected 0 comments, got %d", len(comments))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			s := NewStore(dir)
			err := s.LoadComments(tt.file)
			tt.check(t, s, err)
		})
	}
}

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()

	// Pre-create a review file for one of the files
	reviewDir := filepath.Join(dir, ".crit", "reviews")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	state := ReviewState{
		File: "a.go",
		Comments: []Comment{
			{ID: "11111111", Line: 1, Body: "hello", CreatedAt: time.Now()},
		},
	}
	data, _ := yaml.Marshal(&state)
	path := reviewPath(dir, "a.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewStore(dir)
	err := s.LoadAll([]string{"a.go", "b.go"})
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	if len(s.Comments("a.go")) != 1 {
		t.Errorf("expected 1 comment for a.go, got %d", len(s.Comments("a.go")))
	}
	if len(s.Comments("b.go")) != 0 {
		t.Errorf("expected 0 comments for b.go, got %d", len(s.Comments("b.go")))
	}
}

func TestReviewPathHashing(t *testing.T) {
	// reviewPath should use sha256 of absolute path
	repoRoot := "/home/user/project"
	path1 := reviewPath(repoRoot, "src/main.go")
	path2 := reviewPath(repoRoot, "src/other.go")

	if path1 == path2 {
		t.Error("different files should produce different review paths")
	}

	// Same file should always produce same path
	path1a := reviewPath(repoRoot, "src/main.go")
	if path1 != path1a {
		t.Error("same file should produce same review path")
	}

	// Path should be under .crit/reviews/
	if !filepath.IsAbs(path1) || filepath.Dir(path1) != filepath.Join(repoRoot, ".crit", "reviews") {
		t.Errorf("unexpected review path: %s", path1)
	}

	// Path should end in .yaml
	if filepath.Ext(path1) != ".yaml" {
		t.Errorf("review path should end in .yaml: %s", path1)
	}
}

func TestYAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// Add a comment
	err := s.AddComment("round.go", 42, 15, "content here", "my comment body")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	original := s.Comments("round.go")[0]

	// Load into a new store
	s2 := NewStore(dir)
	err = s2.LoadComments("round.go")
	if err != nil {
		t.Fatalf("LoadComments: %v", err)
	}

	loaded := s2.Comments("round.go")
	if len(loaded) != 1 {
		t.Fatalf("expected 1 comment after round-trip, got %d", len(loaded))
	}

	c := loaded[0]
	if c.ID != original.ID {
		t.Errorf("ID: got %q, want %q", c.ID, original.ID)
	}
	if c.Line != 42 {
		t.Errorf("Line: got %d, want 42", c.Line)
	}
	if c.EndLine != 15 {
		t.Errorf("EndLine: got %d, want 15", c.EndLine)
	}
	if c.ContentSnippet != "content here" {
		t.Errorf("ContentSnippet: got %q, want %q", c.ContentSnippet, "content here")
	}
	if c.Body != "my comment body" {
		t.Errorf("Body: got %q, want %q", c.Body, "my comment body")
	}
}

func TestCritDirectorySetup(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// Adding a comment should create .crit/ and .crit/.gitignore
	err := s.AddComment("test.go", 1, 0, "x", "y")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	gitignorePath := filepath.Join(dir, ".crit", ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if string(data) != "*\n" {
		t.Errorf(".gitignore content: got %q, want %q", string(data), "*\n")
	}
}

func TestCritDirectorySetupIdempotent(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// Create .crit/.gitignore beforehand
	critDir := filepath.Join(dir, ".crit")
	if err := os.MkdirAll(critDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(critDir, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should not error
	err := s.AddComment("test.go", 1, 0, "x", "y")
	if err != nil {
		t.Fatalf("AddComment with existing .crit: %v", err)
	}
}

func TestYAMLFormat(t *testing.T) {
	// Verify the YAML output matches crit's exact format
	dir := t.TempDir()
	s := NewStore(dir)

	err := s.AddComment("src/handler.ts", 440, 0, "if ($('[data]').length > 0", "Why did we switch to DOM check here?")
	if err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	path := reviewPath(dir, "src/handler.ts")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading review file: %v", err)
	}

	// Parse back and verify structure
	var state ReviewState
	if err := yaml.Unmarshal(data, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if state.File != "src/handler.ts" {
		t.Errorf("File: got %q, want %q", state.File, "src/handler.ts")
	}
	if len(state.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(state.Comments))
	}
	c := state.Comments[0]
	if c.Line != 440 {
		t.Errorf("Line: got %d, want 440", c.Line)
	}
	if c.Body != "Why did we switch to DOM check here?" {
		t.Errorf("Body: got %q", c.Body)
	}

	// end_line should be omitted when 0 (omitempty)
	if c.EndLine != 0 {
		t.Errorf("EndLine: got %d, want 0", c.EndLine)
	}
	// Verify end_line is not present in raw YAML when 0
	if strings.Contains(string(data), "end_line") {
		t.Error("end_line should be omitted from YAML when 0")
	}
}
