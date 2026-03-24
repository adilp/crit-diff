package comment

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
)

// reviewPath returns the .crit/reviews/<sha256>.yaml path for a given file.
func reviewPath(repoRoot, filePath string) string {
	abs := filepath.Join(repoRoot, filePath)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(abs)))
	return filepath.Join(repoRoot, ".crit", "reviews", hash+".yaml")
}

// ensureCritDir creates .crit/ and .crit/.gitignore if they don't exist.
func ensureCritDir(repoRoot string) error {
	reviewDir := filepath.Join(repoRoot, ".crit", "reviews")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		return fmt.Errorf("create .crit/reviews: %w", err)
	}

	gitignorePath := filepath.Join(repoRoot, ".crit", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, []byte("*\n"), 0o644); err != nil {
			return fmt.Errorf("create .crit/.gitignore: %w", err)
		}
	}

	return nil
}

// writeReviewFile writes a ReviewState to disk with file locking and atomic rename.
func writeReviewFile(path string, state *ReviewState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create review dir: %w", err)
	}

	lock := flock.New(path + ".lock")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ok, err := lock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	if !ok {
		return fmt.Errorf("could not acquire lock for %s", path)
	}
	defer lock.Unlock()

	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal review state: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// CodeReviewSession is the session manifest written to .crit/code-review.yaml on launch.
type CodeReviewSession struct {
	Files     []string  `yaml:"files"`
	DiffBase  string    `yaml:"diff_base"`
	CreatedAt time.Time `yaml:"created_at"`
}

// WriteSessionManifest writes the session manifest to .crit/code-review.yaml.
func WriteSessionManifest(repoRoot, diffBase string, files []string) error {
	if err := ensureCritDir(repoRoot); err != nil {
		return fmt.Errorf("write session manifest: %w", err)
	}

	session := CodeReviewSession{
		Files:     files,
		DiffBase:  diffBase,
		CreatedAt: time.Now().UTC(),
	}

	data, err := yaml.Marshal(&session)
	if err != nil {
		return fmt.Errorf("marshal session manifest: %w", err)
	}

	path := filepath.Join(repoRoot, ".crit", "code-review.yaml")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write session manifest temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename session manifest: %w", err)
	}

	return nil
}

// readReviewFile reads a ReviewState from disk. Returns nil if the file does not exist.
func readReviewFile(path string) (*ReviewState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read review file: %w", err)
	}

	var state ReviewState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal review file: %w", err)
	}

	return &state, nil
}
