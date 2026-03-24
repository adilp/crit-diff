package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestWriteSessionManifest(t *testing.T) {
	tests := []struct {
		name     string
		diffBase string
		files    []string
		check    func(t *testing.T, dir string, err error)
	}{
		{
			name:     "creates manifest with diff_base and files",
			diffBase: "main",
			files:    []string{"src/app.go", "src/handler.ts"},
			check: func(t *testing.T, dir string, err error) {
				if err != nil {
					t.Fatalf("WriteSessionManifest: %v", err)
				}
				path := filepath.Join(dir, ".crit", "code-review.yaml")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading manifest: %v", err)
				}
				var session CodeReviewSession
				if err := yaml.Unmarshal(data, &session); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if session.DiffBase != "main" {
					t.Errorf("DiffBase: got %q, want %q", session.DiffBase, "main")
				}
				if len(session.Files) != 2 {
					t.Fatalf("Files: got %d, want 2", len(session.Files))
				}
				if session.Files[0] != "src/app.go" {
					t.Errorf("Files[0]: got %q, want %q", session.Files[0], "src/app.go")
				}
				if session.Files[1] != "src/handler.ts" {
					t.Errorf("Files[1]: got %q, want %q", session.Files[1], "src/handler.ts")
				}
				if session.CreatedAt.IsZero() {
					t.Error("CreatedAt is zero")
				}
			},
		},
		{
			name:     "no args uses HEAD as diff_base",
			diffBase: "HEAD",
			files:    []string{"main.go"},
			check: func(t *testing.T, dir string, err error) {
				if err != nil {
					t.Fatalf("WriteSessionManifest: %v", err)
				}
				path := filepath.Join(dir, ".crit", "code-review.yaml")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading manifest: %v", err)
				}
				var session CodeReviewSession
				if err := yaml.Unmarshal(data, &session); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if session.DiffBase != "HEAD" {
					t.Errorf("DiffBase: got %q, want %q", session.DiffBase, "HEAD")
				}
			},
		},
		{
			name:     "ref range uses full range as diff_base",
			diffBase: "abc..def",
			files:    []string{"file.go"},
			check: func(t *testing.T, dir string, err error) {
				if err != nil {
					t.Fatalf("WriteSessionManifest: %v", err)
				}
				path := filepath.Join(dir, ".crit", "code-review.yaml")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading manifest: %v", err)
				}
				var session CodeReviewSession
				if err := yaml.Unmarshal(data, &session); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if session.DiffBase != "abc..def" {
					t.Errorf("DiffBase: got %q, want %q", session.DiffBase, "abc..def")
				}
			},
		},
		{
			name:     "overwrites existing manifest",
			diffBase: "new-base",
			files:    []string{"new.go"},
			check: func(t *testing.T, dir string, err error) {
				if err != nil {
					t.Fatalf("WriteSessionManifest: %v", err)
				}
				// Read back and verify it was overwritten
				path := filepath.Join(dir, ".crit", "code-review.yaml")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading manifest: %v", err)
				}
				var session CodeReviewSession
				if err := yaml.Unmarshal(data, &session); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if session.DiffBase != "new-base" {
					t.Errorf("DiffBase: got %q, want %q", session.DiffBase, "new-base")
				}
				if len(session.Files) != 1 {
					t.Fatalf("Files: got %d, want 1", len(session.Files))
				}
				if session.Files[0] != "new.go" {
					t.Errorf("Files[0]: got %q, want %q", session.Files[0], "new.go")
				}
			},
		},
		{
			name:     "creates crit directory if not exists",
			diffBase: "main",
			files:    []string{"a.go"},
			check: func(t *testing.T, dir string, err error) {
				if err != nil {
					t.Fatalf("WriteSessionManifest: %v", err)
				}
				critDir := filepath.Join(dir, ".crit")
				info, err := os.Stat(critDir)
				if err != nil {
					t.Fatalf(".crit dir not created: %v", err)
				}
				if !info.IsDir() {
					t.Error(".crit is not a directory")
				}
			},
		},
		{
			name:     "created_at is RFC3339 in YAML output",
			diffBase: "main",
			files:    []string{"a.go"},
			check: func(t *testing.T, dir string, err error) {
				if err != nil {
					t.Fatalf("WriteSessionManifest: %v", err)
				}
				path := filepath.Join(dir, ".crit", "code-review.yaml")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading manifest: %v", err)
				}
				raw := string(data)
				// Extract raw created_at value and verify RFC3339 format
				for _, line := range strings.Split(raw, "\n") {
					trimmed := strings.TrimSpace(line)
					if strings.HasPrefix(trimmed, "created_at:") {
						ts := strings.TrimSpace(strings.TrimPrefix(trimmed, "created_at:"))
						if _, parseErr := time.Parse(time.RFC3339, ts); parseErr != nil {
							t.Errorf("created_at not RFC3339: %q: %v", ts, parseErr)
						}
					}
				}
				// Parse and verify it's a valid recent time
				var session CodeReviewSession
				if err := yaml.Unmarshal(data, &session); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if time.Since(session.CreatedAt) > time.Minute {
					t.Errorf("CreatedAt too old: %v", session.CreatedAt)
				}
			},
		},
		{
			name:     "manifest file paths match diff paths",
			diffBase: "main",
			files:    []string{"src/services/auth.ts", "src/helpers/app.rb", "controllers/smart.rb"},
			check: func(t *testing.T, dir string, err error) {
				if err != nil {
					t.Fatalf("WriteSessionManifest: %v", err)
				}
				path := filepath.Join(dir, ".crit", "code-review.yaml")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("reading manifest: %v", err)
				}
				var session CodeReviewSession
				if err := yaml.Unmarshal(data, &session); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				want := []string{"src/services/auth.ts", "src/helpers/app.rb", "controllers/smart.rb"}
				if len(session.Files) != len(want) {
					t.Fatalf("Files: got %d, want %d", len(session.Files), len(want))
				}
				for i, w := range want {
					if session.Files[i] != w {
						t.Errorf("Files[%d]: got %q, want %q", i, session.Files[i], w)
					}
				}
			},
		},
		{
			name:     "returns error for invalid repo root",
			diffBase: "main",
			files:    []string{"a.go"},
			check: func(t *testing.T, dir string, err error) {
				if err == nil {
					t.Error("expected error for invalid repo root, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			// For the error test, use a non-writable path
			if tt.name == "returns error for invalid repo root" {
				dir = "/dev/null/invalid"
			}
			// For the overwrite test, pre-create a manifest
			if tt.name == "overwrites existing manifest" {
				critDir := filepath.Join(dir, ".crit")
				if err := os.MkdirAll(critDir, 0o755); err != nil {
					t.Fatal(err)
				}
				old := CodeReviewSession{
					Files:     []string{"old.go"},
					DiffBase:  "old-base",
					CreatedAt: time.Now().Add(-time.Hour),
				}
				data, _ := yaml.Marshal(&old)
				if err := os.WriteFile(filepath.Join(critDir, "code-review.yaml"), data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := WriteSessionManifest(dir, tt.diffBase, tt.files)
			tt.check(t, dir, err)
		})
	}
}
