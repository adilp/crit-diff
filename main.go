package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adilp/crit-diff/internal/comment"
	"github.com/adilp/crit-diff/internal/diff"
	"github.com/adilp/crit-diff/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

//go:embed skill/cr-review/SKILL.md
var skillContent embed.FS

type Mode string

const (
	ModeWorkingTree Mode = "working_tree"
	ModeSingleRef   Mode = "single_ref"
	ModeRefRange    Mode = "ref_range"
	ModeStaged      Mode = "staged"
)

type Args struct {
	Mode        Mode
	RefFrom     string
	RefTo       string
	PathFilters []string
	Help        bool
	Subcmd      string // "status" subcommand
	Detach      bool   // --detach: open in tmux split pane
	Wait        bool   // --wait: block until detached review completes
}

// numRe matches numeric shortcuts like -1, -3, -10.
var numRe = regexp.MustCompile(`^-\d+$`)

// execCommand wraps exec.Command to allow substitution in tests.
var execCommand = exec.Command

func parseArgs(args []string) Args {
	result := Args{
		Mode: ModeWorkingTree,
	}

	// Find the "--" separator
	dashIdx := -1
	for i, a := range args {
		if a == "--" {
			dashIdx = i
			break
		}
	}

	// Extract path filters (everything after --)
	var refArgs []string
	if dashIdx >= 0 {
		result.PathFilters = args[dashIdx+1:]
		refArgs = args[:dashIdx]
	} else {
		refArgs = args
	}

	// Extract flags and positional args
	var positional []string
	for _, a := range refArgs {
		switch a {
		case "--help", "-h":
			result.Help = true
			return result
		case "--detach":
			result.Detach = true
		case "--wait":
			result.Wait = true
		case "--staged", "--cached":
			result.Mode = ModeStaged
		case "--force", "--project":
			// Handled directly in main() for setup-claude
		default:
			positional = append(positional, a)
		}
	}

	// Check for subcommands
	if len(positional) > 0 {
		switch positional[0] {
		case "status", "setup-claude":
			result.Subcmd = positional[0]
			return result
		}
	}

	// Parse ref arguments
	if len(positional) == 0 {
		return result
	}

	if len(positional) > 1 {
		fmt.Fprintf(os.Stderr, "warning: unexpected arguments ignored: %s\n", strings.Join(positional[1:], " "))
	}

	ref := positional[0]

	// Numeric shortcut: -N → HEAD~N..HEAD
	if numRe.MatchString(ref) {
		n := ref[1:] // strip leading "-"
		result.Mode = ModeRefRange
		result.RefFrom = "HEAD~" + n
		result.RefTo = "HEAD"
		return result
	}

	if strings.Contains(ref, "..") {
		parts := strings.SplitN(ref, "..", 2)
		result.Mode = ModeRefRange
		result.RefFrom = parts[0]
		result.RefTo = parts[1]
	} else {
		result.Mode = ModeSingleRef
		result.RefFrom = ref
		result.RefTo = "HEAD"
	}

	return result
}

func checkGitRepo(dir string) error {
	cmd := execCommand("git", "-C", dir, "rev-parse", "--show-toplevel")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("not a git repository (git: %s): %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: cr [<ref>] [<refA>..<refB>] [-- <path>...]

Modes:
  cr                        Review working tree changes (staged + unstaged)
  cr <ref>                  Review changes from <ref> to HEAD
  cr <refA>..<refB>         Review changes between two refs
  cr <ref> -- <path>...     Review changes filtered to specific paths

Shortcuts:
  cr -1                     Last commit
  cr -3                     Last 3 commits
  cr --staged               Only staged changes (same as --cached)

Commands:
  cr status                 Output all review comments as JSON (reads .crit/)
  cr setup-claude           Install /cr-review skill for Claude Code
  cr setup-claude --project Install skill for current project only
  cr setup-claude --force   Overwrite existing skill

Integration:
  cr --detach <ref>         Open in a tmux split pane
  cr --detach --wait <ref>  Open in tmux and block until review completes

Options:
  -h, --help                Show this help message
`)
}

// statusComment is the JSON output format for cr status.
type statusComment struct {
	File           string `json:"file"`
	ID             string `json:"id"`
	Line           int    `json:"line"`
	EndLine        int    `json:"end_line,omitempty"`
	ContentSnippet string `json:"content_snippet"`
	Body           string `json:"body"`
}

// runStatus reads all .crit/ comments and outputs them as JSON.
func runStatus(repoRoot string) (string, error) {
	manifest, err := os.ReadFile(filepath.Join(repoRoot, ".crit", "code-review.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return "[]", nil
		}
		return "", fmt.Errorf("read session manifest: %w", err)
	}

	// Parse manifest to get file list
	var session comment.CodeReviewSession
	if err := yaml.Unmarshal(manifest, &session); err != nil {
		return "", fmt.Errorf("parse session manifest: %w", err)
	}

	store := comment.NewStore(repoRoot)
	if err := store.LoadAll(session.Files); err != nil {
		return "", fmt.Errorf("load comments: %w", err)
	}

	var result []statusComment
	for _, file := range session.Files {
		for _, c := range store.Comments(file) {
			result = append(result, statusComment{
				File:           file,
				ID:             c.ID,
				Line:           c.Line,
				EndLine:        c.EndLine,
				ContentSnippet: c.ContentSnippet,
				Body:           c.Body,
			})
		}
	}

	if result == nil {
		result = []statusComment{}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal comments: %w", err)
	}
	return string(data), nil
}

// shellEscape escapes a string for safe embedding in a POSIX shell command.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// runDetached opens cr in a tmux split pane. If wait is true, blocks until the pane closes.
func runDetached(crArgs []string, wait bool) error {
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("--detach requires a tmux session (TMUX not set)")
	}

	tmuxBin, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	crBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve cr binary: %w", err)
	}
	crBin, err = filepath.EvalSymlinks(crBin)
	if err != nil {
		return fmt.Errorf("resolve cr symlink: %w", err)
	}

	channel := fmt.Sprintf("cr-review-%d", os.Getpid())

	// Build the command to run in the tmux pane
	var argStr string
	for _, a := range crArgs {
		argStr += " " + shellEscape(a)
	}
	paneCmd := fmt.Sprintf("%s%s ; tmux wait-for -S %s", shellEscape(crBin), argStr, channel)

	splitCmd := exec.Command(tmuxBin, "split-window", "-h", "-p", "70", paneCmd)
	if err := splitCmd.Run(); err != nil {
		// Retry without -p flag — percentage sizing fails when parent pane
		// size isn't available (e.g. invoked from a subprocess like Claude Code)
		splitCmd = exec.Command(tmuxBin, "split-window", "-h", paneCmd)
		if err := splitCmd.Run(); err != nil {
			return fmt.Errorf("failed to open tmux pane: %w", err)
		}
	}

	fmt.Fprintln(os.Stderr, "Opened cr in tmux pane")

	if wait {
		waitCmd := exec.Command(tmuxBin, "wait-for", channel)
		if err := waitCmd.Run(); err != nil {
			return fmt.Errorf("review pane terminated abnormally")
		}
		fmt.Fprintln(os.Stderr, "Review complete.")
	}

	return nil
}

// runSetupClaude installs the cr-review skill to ~/.claude/skills/ (or a custom home dir for testing).
func runSetupClaude(homeDir string, force bool) error {
	return installSkill(filepath.Join(homeDir, ".claude", "skills", "cr-review"), force)
}

// runSetupClaudeProject installs the cr-review skill to .claude/skills/ in the given directory.
func runSetupClaudeProject(projectDir string, force bool) error {
	return installSkill(filepath.Join(projectDir, ".claude", "skills", "cr-review"), force)
}

func installSkill(targetDir string, force bool) error {
	targetPath := filepath.Join(targetDir, "SKILL.md")

	if !force {
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("skill already exists at %s (use --force to overwrite)", targetPath)
		}
	}

	content, err := skillContent.ReadFile("skill/cr-review/SKILL.md")
	if err != nil {
		return fmt.Errorf("reading embedded skill: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", targetDir, err)
	}

	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		return fmt.Errorf("writing skill file: %w", err)
	}

	return nil
}

func main() {
	args := parseArgs(os.Args[1:])

	if args.Help {
		printUsage()
		os.Exit(0)
	}

	// Handle setup-claude (doesn't need git repo)
	if args.Subcmd == "setup-claude" {
		force := false
		project := false
		for _, a := range os.Args[1:] {
			if a == "--force" {
				force = true
			}
			if a == "--project" {
				project = true
			}
		}

		if project {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			if err := runSetupClaudeProject(cwd, force); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Installed /cr-review skill for this project\n")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			if err := runSetupClaude(home, force); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Installed /cr-review skill globally\n")
		}
		fmt.Println("You can now use /cr-review <ref> in Claude Code.")
		os.Exit(0)
	}

	// Check we're in a git repo
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := checkGitRepo(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Handle status subcommand
	if args.Subcmd == "status" {
		repoRoot, err := diff.GetRepoRoot(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		output, err := runStatus(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(output)
		os.Exit(0)
	}

	// Handle --detach mode (open in tmux split pane)
	if args.Detach {
		// Rebuild args without --detach and --wait for the child process
		var childArgs []string
		for _, a := range os.Args[1:] {
			if a != "--detach" && a != "--wait" {
				childArgs = append(childArgs, a)
			}
		}
		if err := runDetached(childArgs, args.Wait); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Build diff args from parsed CLI args
	diffArgs := diff.DiffArgs{
		Paths: args.PathFilters,
	}
	switch args.Mode {
	case ModeRefRange:
		diffArgs.RefRange = args.RefFrom + ".." + args.RefTo
	case ModeSingleRef:
		diffArgs.RefRange = args.RefFrom
	case ModeStaged:
		diffArgs.Staged = true
	}

	// Get raw diff
	rawDiff, err := diff.GetDiff(diffArgs, cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if rawDiff == "" {
		fmt.Println("no changes")
		os.Exit(0)
	}

	// Parse diff
	files, err := diff.Parse(rawDiff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("no changes")
		os.Exit(0)
	}

	// Write session manifest
	var diffBase string
	switch args.Mode {
	case ModeWorkingTree:
		diffBase = "HEAD"
	case ModeStaged:
		diffBase = "HEAD (staged)"
	case ModeSingleRef:
		diffBase = args.RefFrom
	case ModeRefRange:
		diffBase = args.RefFrom + ".." + args.RefTo
	}
	var filePaths []string
	for _, f := range files {
		name := f.NewName
		if name == "" {
			name = f.OldName
		}
		filePaths = append(filePaths, name)
	}
	repoRoot, err := diff.GetRepoRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := comment.WriteSessionManifest(repoRoot, diffBase, filePaths); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write session manifest: %v\n", err)
	}

	// Build paired lines for the first file
	paired := diff.BuildPairedLines(files[0].Hunks)

	// Determine refs for file content fetch
	var refOld, refNew string
	switch args.Mode {
	case ModeWorkingTree:
		refOld = "HEAD"
		refNew = "" // working tree — read from disk
	case ModeStaged:
		refOld = "HEAD"
		refNew = "" // staged — read from index (git show :path)
	case ModeSingleRef:
		refOld = args.RefFrom
		refNew = "HEAD"
	case ModeRefRange:
		refOld = args.RefFrom
		refNew = args.RefTo
	}

	// Create comment store, load existing comments
	store := comment.NewStore(repoRoot)
	if err := store.LoadAll(filePaths); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load comments: %v\n", err)
	}

	// Launch TUI
	m := ui.NewModel(files, paired, 0, 0)
	m.SetStore(store)

	// Set ref for status bar display
	switch args.Mode {
	case ModeRefRange:
		m.SetRef(args.RefFrom + ".." + args.RefTo)
	case ModeSingleRef:
		m.SetRef(args.RefFrom + "..HEAD")
	case ModeStaged:
		m.SetRef("(staged)")
	}

	// Lazy-fetch and highlight the first file
	f := files[0]
	if !f.IsBinary {
		filename := f.NewName
		if filename == "" {
			filename = f.OldName
		}

		var oldContent, newContent string

		// Fetch old side content
		if !f.IsNew {
			oldContent, _ = diff.GetFileContent(refOld, f.OldName, cwd)
		}

		// Fetch new side content
		if !f.IsDeleted {
			if refNew == "" {
				// Working tree — read from disk
				data, err := os.ReadFile(f.NewName)
				if err == nil {
					newContent = string(data)
				}
			} else {
				newContent, _ = diff.GetFileContent(refNew, f.NewName, cwd)
			}
		}

		m.SetHighlighting(filename, oldContent, newContent)
		m.SetFileContent(oldContent, newContent)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
