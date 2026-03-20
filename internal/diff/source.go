package diff

import (
	"fmt"
	"os/exec"
	"strings"
)

// DiffArgs holds parsed CLI arguments for generating a git diff.
type DiffArgs struct {
	RefRange string   // "" for working tree, "main" for single ref, "a..b" for range
	Paths    []string // path filters after "--"
}

// GetDiff executes the appropriate git diff command and returns the raw unified diff output.
// The dir parameter specifies the git repository directory.
func GetDiff(args DiffArgs, dir string) (string, error) {
	gitArgs := []string{"-C", dir, "diff"}

	switch {
	case args.RefRange == "":
		// Working tree: git diff HEAD
		gitArgs = append(gitArgs, "HEAD")
	case strings.Contains(args.RefRange, ".."):
		// Ref range: git diff refA..refB
		gitArgs = append(gitArgs, args.RefRange)
	default:
		// Single ref: git diff ref..HEAD
		gitArgs = append(gitArgs, args.RefRange+"..HEAD")
	}

	if len(args.Paths) > 0 {
		gitArgs = append(gitArgs, "--")
		gitArgs = append(gitArgs, args.Paths...)
	}

	cmd := exec.Command("git", gitArgs...)
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git diff failed: %s: %w", strings.TrimSpace(string(exitErr.Stderr)), err)
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	return string(stdout), nil
}

// GetRepoRoot returns the absolute path to the git repository root.
func GetRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("not a git repository: %s: %w", strings.TrimSpace(string(exitErr.Stderr)), err)
		}
		return "", fmt.Errorf("not a git repository: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
