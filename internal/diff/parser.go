package diff

import (
	"fmt"
	"os"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// LineType indicates whether a diff line is context, added, or deleted.
type LineType int

const (
	LineContext LineType = iota
	LineAdd
	LineDelete
)

// DiffLine represents a single line within a diff hunk.
type DiffLine struct {
	Type    LineType
	OldNum  int
	NewNum  int
	Content string
	NoEOL   bool
}

// Hunk represents a contiguous block of changes within a file diff.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// DiffFile represents a single file's diff with its metadata and hunks.
type DiffFile struct {
	OldName     string
	NewName     string
	IsRename    bool
	IsBinary    bool
	IsNew       bool
	IsDeleted   bool
	IsSubmodule bool
	OldMode     os.FileMode
	NewMode     os.FileMode
	Hunks       []Hunk
}

// Parse takes raw unified diff output and returns structured DiffFile data.
func Parse(rawDiff string) ([]DiffFile, error) {
	if rawDiff == "" {
		return nil, nil
	}

	// Pre-scan for "Binary files ... differ" lines that go-gitdiff doesn't detect
	binaryFiles := detectBinaryFiles(rawDiff)

	parsed, _, err := gitdiff.Parse(strings.NewReader(rawDiff))
	if err != nil {
		return nil, fmt.Errorf("parse diff: %w", err)
	}

	files := make([]DiffFile, 0, len(parsed))
	for _, f := range parsed {
		df := DiffFile{
			OldName:   f.OldName,
			NewName:   f.NewName,
			IsRename:  f.IsRename,
			IsBinary:  f.IsBinary,
			IsNew:     f.IsNew,
			IsDeleted: f.IsDelete,
			OldMode:   f.OldMode,
			NewMode:   f.NewMode,
		}

		// Check pre-scanned binary markers
		if !df.IsBinary && binaryFiles[f.NewName] {
			df.IsBinary = true
		}

		for _, frag := range f.TextFragments {
			h := Hunk{
				OldStart: int(frag.OldPosition),
				OldCount: int(frag.OldLines),
				NewStart: int(frag.NewPosition),
				NewCount: int(frag.NewLines),
			}

			oldNum := int(frag.OldPosition)
			newNum := int(frag.NewPosition)

			for _, line := range frag.Lines {
				dl := DiffLine{
					Content: strings.TrimSuffix(line.Line, "\n"),
					NoEOL:   line.NoEOL(),
				}

				switch line.Op {
				case gitdiff.OpContext:
					dl.Type = LineContext
					dl.OldNum = oldNum
					dl.NewNum = newNum
					oldNum++
					newNum++
				case gitdiff.OpAdd:
					dl.Type = LineAdd
					dl.NewNum = newNum
					newNum++
				case gitdiff.OpDelete:
					dl.Type = LineDelete
					dl.OldNum = oldNum
					oldNum++
				}

				h.Lines = append(h.Lines, dl)
			}

			df.Hunks = append(df.Hunks, h)
		}

		// Post-process: detect submodule diffs
		for _, h := range df.Hunks {
			for _, l := range h.Lines {
				if strings.HasPrefix(l.Content, "Subproject commit ") {
					df.IsSubmodule = true
					break
				}
			}
			if df.IsSubmodule {
				break
			}
		}

		files = append(files, df)
	}

	return files, nil
}

// detectBinaryFiles scans raw diff for "Binary files ... differ" lines
// that go-gitdiff does not detect as binary.
func detectBinaryFiles(rawDiff string) map[string]bool {
	result := make(map[string]bool)
	for _, line := range strings.Split(rawDiff, "\n") {
		if strings.HasPrefix(line, "Binary files ") && strings.HasSuffix(line, " differ") {
			// Format: "Binary files a/path and b/path differ"
			trimmed := strings.TrimPrefix(line, "Binary files ")
			trimmed = strings.TrimSuffix(trimmed, " differ")
			parts := strings.SplitN(trimmed, " and ", 2)
			if len(parts) == 2 {
				// Strip "b/" prefix from new path
				newPath := strings.TrimPrefix(parts[1], "b/")
				result[newPath] = true
			}
		}
	}
	return result
}