package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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

	fmt.Printf("cr: mode=%s", args.Mode)
	if args.RefFrom != "" {
		fmt.Printf(" ref=%s..%s", args.RefFrom, args.RefTo)
	}
	if len(args.PathFilters) > 0 {
		fmt.Printf(" paths=%v", args.PathFilters)
	}
	fmt.Println()
}
