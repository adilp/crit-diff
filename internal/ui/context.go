package ui

import (
	"github.com/adil/cr/internal/diff"
)

// SeparatorState tracks the expansion state for a separator between two hunks.
type SeparatorState struct {
	HunkIndex    int // between hunk HunkIndex-1 and hunk HunkIndex
	ExpandedUp   int // lines already expanded upward (from hunk below)
	ExpandedDown int // lines already expanded downward (from hunk above)
	MaxGap       int // total lines available between the two hunks
}

// expandLinesPerPress is the number of context lines added per ]e or [e press.
const expandLinesPerPress = 10

// ComputeMaxGap calculates the number of lines between hunk ending at oldEndLine
// and hunk starting at oldStartLine. These are the "hidden" context lines between hunks.
func ComputeMaxGap(oldEndLine, oldStartLine int) int {
	gap := oldStartLine - oldEndLine - 1
	if gap < 0 {
		return 0
	}
	return gap
}

// BuildSeparatorStates creates initial SeparatorState entries for all separators
// in a file's hunks. Each separator sits between hunk[i-1] and hunk[i].
func BuildSeparatorStates(hunks []diff.Hunk) []SeparatorState {
	if len(hunks) < 2 {
		return nil
	}
	states := make([]SeparatorState, 0, len(hunks)-1)
	for i := 1; i < len(hunks); i++ {
		prevEnd := hunks[i-1].OldStart + hunks[i-1].OldCount - 1
		nextStart := hunks[i].OldStart
		states = append(states, SeparatorState{
			HunkIndex: i,
			MaxGap:    ComputeMaxGap(prevEnd, nextStart),
		})
	}
	return states
}

// FindNearestSeparatorBelow finds the index into paired lines of the separator
// at or below cursorRow. Returns -1 if no separator found.
func FindNearestSeparatorBelow(paired []diff.PairedLine, cursorRow int) int {
	for i := cursorRow; i < len(paired); i++ {
		if paired[i].IsSeparator {
			return i
		}
	}
	return -1
}

// FindNearestSeparatorAbove finds the index into paired lines of the separator
// at or above cursorRow. Returns -1 if no separator found.
func FindNearestSeparatorAbove(paired []diff.PairedLine, cursorRow int) int {
	if cursorRow >= len(paired) {
		cursorRow = len(paired) - 1
	}
	for i := cursorRow; i >= 0; i-- {
		if paired[i].IsSeparator {
			return i
		}
	}
	return -1
}

// IsSeparatorFullyExpanded returns true if all context between two hunks has been revealed.
func IsSeparatorFullyExpanded(state SeparatorState) bool {
	return state.ExpandedDown+state.ExpandedUp >= state.MaxGap
}

// ExpandedContextLines returns context PairedLines for an expansion.
// oldLines and newLines are the full file content (0-indexed slices of line strings).
// direction is "down" or "up".
// Returns the new context paired lines to insert and updates the state.
// Syntax highlighting is applied automatically by the render pipeline via line numbers.
func ExpandedContextLines(
	state *SeparatorState,
	hunkAbove, hunkBelow diff.Hunk,
	oldLines, newLines []string,
	direction string,
) []diff.PairedLine {
	if IsSeparatorFullyExpanded(*state) {
		return nil
	}

	remaining := state.MaxGap - state.ExpandedDown - state.ExpandedUp
	count := expandLinesPerPress
	if count > remaining {
		count = remaining
	}

	// The gap sits between:
	//   old line: hunkAbove.OldStart + hunkAbove.OldCount - 1 (last line of hunk above)
	//   old line: hunkBelow.OldStart (first line of hunk below)
	// The new side has a parallel gap computed from NewStart/NewCount.
	oldGapStart := hunkAbove.OldStart + hunkAbove.OldCount // first gap line (1-based)
	newGapStart := hunkAbove.NewStart + hunkAbove.NewCount

	var startOld, startNew int
	switch direction {
	case "down":
		startOld = oldGapStart + state.ExpandedDown
		startNew = newGapStart + state.ExpandedDown
		state.ExpandedDown += count
	case "up":
		// Expand upward from the bottom of the gap
		// The bottom of the gap is hunkBelow.OldStart - 1
		startOld = hunkBelow.OldStart - state.ExpandedUp - count
		startNew = hunkBelow.NewStart - state.ExpandedUp - count
		state.ExpandedUp += count
	default:
		return nil
	}

	pairs := make([]diff.PairedLine, count)
	for i := range count {
		oldIdx := startOld + i - 1 // 1-based to 0-indexed
		newIdx := startNew + i - 1

		oldContent := ""
		if oldIdx >= 0 && oldIdx < len(oldLines) {
			oldContent = oldLines[oldIdx]
		}
		newContent := ""
		if newIdx >= 0 && newIdx < len(newLines) {
			newContent = newLines[newIdx]
		}

		left := diff.DiffLine{
			Type:    diff.LineContext,
			OldNum:  startOld + i,
			NewNum:  startNew + i,
			Content: oldContent,
		}
		right := diff.DiffLine{
			Type:    diff.LineContext,
			OldNum:  startOld + i,
			NewNum:  startNew + i,
			Content: newContent,
		}

		pairs[i] = diff.PairedLine{
			Left:      &left,
			Right:     &right,
			HunkIndex: state.HunkIndex,
		}
	}

	return pairs
}
