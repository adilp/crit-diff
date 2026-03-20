package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/adil/cr/internal/diff"
	"github.com/adil/cr/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type Mode string

const (
	ModeWorkingTree Mode = "working_tree"
	ModeSingleRef   Mode = "single_ref"
	ModeRefRange    Mode = "ref_range"
)

type Args struct {
	Mode        Mode
	RefFrom     string
	RefTo       string
	PathFilters []string
	Help        bool
}

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

	// Parse ref arguments
	if len(refArgs) == 0 {
		return result
	}

	if len(refArgs) > 1 {
		fmt.Fprintf(os.Stderr, "warning: unexpected arguments ignored: %s\n", strings.Join(refArgs[1:], " "))
	}

	ref := refArgs[0]

	if ref == "--help" || ref == "-h" {
		result.Help = true
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

Options:
  -h, --help                Show this help message
`)
}

func main() {
	args := parseArgs(os.Args[1:])

	if args.Help {
		printUsage()
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

	// Build diff args from parsed CLI args
	diffArgs := diff.DiffArgs{
		Paths: args.PathFilters,
	}
	switch args.Mode {
	case ModeRefRange:
		diffArgs.RefRange = args.RefFrom + ".." + args.RefTo
	case ModeSingleRef:
		diffArgs.RefRange = args.RefFrom
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

	// Build paired lines for the first file
	paired := diff.BuildPairedLines(files[0].Hunks)

	// Determine refs for file content fetch
	var refOld, refNew string
	switch args.Mode {
	case ModeWorkingTree:
		refOld = "HEAD"
		refNew = "" // working tree — read from disk
	case ModeSingleRef:
		refOld = args.RefFrom
		refNew = "HEAD"
	case ModeRefRange:
		refOld = args.RefFrom
		refNew = args.RefTo
	}

	// Launch TUI
	m := ui.NewModel(files, paired, 0, 0)

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
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
