package ui

import (
	"fmt"
	"testing"

	"github.com/adil/cr/internal/diff"
)

func TestComputeMaxGap(t *testing.T) {
	tests := []struct {
		name         string
		oldEndLine   int
		oldStartLine int
		want         int
	}{
		{name: "gap of 5", oldEndLine: 5, oldStartLine: 11, want: 5},
		{name: "adjacent lines no gap", oldEndLine: 5, oldStartLine: 6, want: 0},
		{name: "overlapping returns 0", oldEndLine: 10, oldStartLine: 8, want: 0},
		{name: "gap of 1", oldEndLine: 5, oldStartLine: 7, want: 1},
		{name: "large gap", oldEndLine: 10, oldStartLine: 31, want: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeMaxGap(tt.oldEndLine, tt.oldStartLine)
			if got != tt.want {
				t.Errorf("ComputeMaxGap(%d, %d) = %d, want %d", tt.oldEndLine, tt.oldStartLine, got, tt.want)
			}
		})
	}
}

func TestBuildSeparatorStates(t *testing.T) {
	tests := []struct {
		name  string
		hunks []diff.Hunk
		want  int // number of separator states
		check func(t *testing.T, states []SeparatorState)
	}{
		{
			name:  "no hunks returns empty",
			hunks: nil,
			want:  0,
		},
		{
			name: "single hunk returns empty",
			hunks: []diff.Hunk{
				{OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3},
			},
			want: 0,
		},
		{
			name: "two hunks returns one state",
			hunks: []diff.Hunk{
				{OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3},
				{OldStart: 10, OldCount: 2, NewStart: 10, NewCount: 2},
			},
			want: 1,
			check: func(t *testing.T, states []SeparatorState) {
				s := states[0]
				if s.HunkIndex != 1 {
					t.Errorf("HunkIndex = %d, want 1", s.HunkIndex)
				}
				// hunk0 ends at OldStart+OldCount-1 = 3, hunk1 starts at OldStart = 10
				// gap = 10 - 3 - 1 = 6
				if s.MaxGap != 6 {
					t.Errorf("MaxGap = %d, want 6", s.MaxGap)
				}
				if s.ExpandedUp != 0 || s.ExpandedDown != 0 {
					t.Errorf("Expected zero expansion, got up=%d down=%d", s.ExpandedUp, s.ExpandedDown)
				}
			},
		},
		{
			name: "three hunks returns two states",
			hunks: []diff.Hunk{
				{OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2},
				{OldStart: 10, OldCount: 3, NewStart: 10, NewCount: 3},
				{OldStart: 20, OldCount: 2, NewStart: 20, NewCount: 2},
			},
			want: 2,
			check: func(t *testing.T, states []SeparatorState) {
				// Between hunk 0 (ends at 2) and hunk 1 (starts at 10): gap = 10-2-1 = 7
				if states[0].MaxGap != 7 {
					t.Errorf("states[0].MaxGap = %d, want 7", states[0].MaxGap)
				}
				// Between hunk 1 (ends at 12) and hunk 2 (starts at 20): gap = 20-12-1 = 7
				if states[1].MaxGap != 7 {
					t.Errorf("states[1].MaxGap = %d, want 7", states[1].MaxGap)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			states := BuildSeparatorStates(tt.hunks)
			if len(states) != tt.want {
				t.Fatalf("len(states) = %d, want %d", len(states), tt.want)
			}
			if tt.check != nil {
				tt.check(t, states)
			}
		})
	}
}

func TestFindNearestSeparatorBelow(t *testing.T) {
	// Build paired lines with two hunks and a separator at index 2
	pairs := diff.BuildPairedLines([]diff.Hunk{
		{
			OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "a"},
				{Type: diff.LineContext, OldNum: 2, NewNum: 2, Content: "b"},
			},
		},
		{
			OldStart: 10, OldCount: 2, NewStart: 10, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 10, NewNum: 10, Content: "c"},
				{Type: diff.LineContext, OldNum: 11, NewNum: 11, Content: "d"},
			},
		},
	})
	// pairs: [0]=a, [1]=b, [2]=separator, [3]=c, [4]=d

	tests := []struct {
		name      string
		cursorRow int
		want      int
	}{
		{name: "cursor on separator", cursorRow: 2, want: 2},
		{name: "cursor above separator", cursorRow: 0, want: 2},
		{name: "cursor below separator", cursorRow: 3, want: -1},
		{name: "cursor just before separator", cursorRow: 1, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindNearestSeparatorBelow(pairs, tt.cursorRow)
			if got != tt.want {
				t.Errorf("FindNearestSeparatorBelow(cursor=%d) = %d, want %d", tt.cursorRow, got, tt.want)
			}
		})
	}
}

func TestFindNearestSeparatorAbove(t *testing.T) {
	pairs := diff.BuildPairedLines([]diff.Hunk{
		{
			OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 1, NewNum: 1, Content: "a"},
				{Type: diff.LineContext, OldNum: 2, NewNum: 2, Content: "b"},
			},
		},
		{
			OldStart: 10, OldCount: 2, NewStart: 10, NewCount: 2,
			Lines: []diff.DiffLine{
				{Type: diff.LineContext, OldNum: 10, NewNum: 10, Content: "c"},
				{Type: diff.LineContext, OldNum: 11, NewNum: 11, Content: "d"},
			},
		},
	})
	// pairs: [0]=a, [1]=b, [2]=separator, [3]=c, [4]=d

	tests := []struct {
		name      string
		cursorRow int
		want      int
	}{
		{name: "cursor on separator", cursorRow: 2, want: 2},
		{name: "cursor below separator", cursorRow: 4, want: 2},
		{name: "cursor above separator", cursorRow: 0, want: -1},
		{name: "cursor just after separator", cursorRow: 3, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindNearestSeparatorAbove(pairs, tt.cursorRow)
			if got != tt.want {
				t.Errorf("FindNearestSeparatorAbove(cursor=%d) = %d, want %d", tt.cursorRow, got, tt.want)
			}
		})
	}
}

func TestIsSeparatorFullyExpanded(t *testing.T) {
	tests := []struct {
		name  string
		state SeparatorState
		want  bool
	}{
		{
			name:  "not expanded",
			state: SeparatorState{MaxGap: 10, ExpandedUp: 0, ExpandedDown: 0},
			want:  false,
		},
		{
			name:  "partially expanded",
			state: SeparatorState{MaxGap: 10, ExpandedUp: 3, ExpandedDown: 5},
			want:  false,
		},
		{
			name:  "fully expanded exact",
			state: SeparatorState{MaxGap: 10, ExpandedUp: 5, ExpandedDown: 5},
			want:  true,
		},
		{
			name:  "over expanded",
			state: SeparatorState{MaxGap: 10, ExpandedUp: 6, ExpandedDown: 6},
			want:  true,
		},
		{
			name:  "zero gap always fully expanded",
			state: SeparatorState{MaxGap: 0, ExpandedUp: 0, ExpandedDown: 0},
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSeparatorFullyExpanded(tt.state)
			if got != tt.want {
				t.Errorf("IsSeparatorFullyExpanded(%+v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestExpandedContextLines(t *testing.T) {
	// Simulate a file with 20 lines. Hunk 0 ends at old line 3, hunk 1 starts at old line 10.
	// So gap is lines 4-9 (6 lines). Old lines and new lines are same (no changes in gap).
	oldLines := make([]string, 20)
	newLines := make([]string, 20)
	for i := range 20 {
		oldLines[i] = fmt.Sprintf("line %d", i+1)
		newLines[i] = fmt.Sprintf("line %d", i+1)
	}

	hunkAbove := diff.Hunk{OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3}
	hunkBelow := diff.Hunk{OldStart: 10, OldCount: 2, NewStart: 10, NewCount: 2}

	t.Run("expand down first press", func(t *testing.T) {
		state := &SeparatorState{HunkIndex: 1, MaxGap: 6}
		lines := ExpandedContextLines(state, hunkAbove, hunkBelow, oldLines, newLines, "down")
		if len(lines) != 6 { // only 6 lines available, not 10
			t.Fatalf("expected 6 lines, got %d", len(lines))
		}
		// First expanded line should be old line 4 (0-indexed: 3)
		if lines[0].Left == nil || lines[0].Left.OldNum != 4 {
			t.Errorf("first line OldNum = %v, want 4", lines[0].Left)
		}
		if lines[0].Left.Content != "line 4" {
			t.Errorf("first line content = %q, want %q", lines[0].Left.Content, "line 4")
		}
		// Last expanded line should be old line 9
		if lines[5].Left == nil || lines[5].Left.OldNum != 9 {
			t.Errorf("last line OldNum = %v, want 9", lines[5].Left)
		}
		// State should be updated
		if state.ExpandedDown != 6 {
			t.Errorf("ExpandedDown = %d, want 6", state.ExpandedDown)
		}
		// Context lines should have both Left and Right
		for i, l := range lines {
			if l.Left == nil || l.Right == nil {
				t.Errorf("line %d: expected both Left and Right", i)
			}
			if l.Left.Type != diff.LineContext {
				t.Errorf("line %d: expected LineContext, got %d", i, l.Left.Type)
			}
		}
	})

	t.Run("expand up first press", func(t *testing.T) {
		state := &SeparatorState{HunkIndex: 1, MaxGap: 6}
		lines := ExpandedContextLines(state, hunkAbove, hunkBelow, oldLines, newLines, "up")
		if len(lines) != 6 { // only 6 lines available
			t.Fatalf("expected 6 lines, got %d", len(lines))
		}
		// Lines should be old lines 4-9 (expanding up from hunk below)
		if lines[0].Left == nil || lines[0].Left.OldNum != 4 {
			t.Errorf("first line OldNum = %v, want 4", lines[0].Left)
		}
		if state.ExpandedUp != 6 {
			t.Errorf("ExpandedUp = %d, want 6", state.ExpandedUp)
		}
	})

	t.Run("expand down with larger gap", func(t *testing.T) {
		// Hunks with 15-line gap
		hAbove := diff.Hunk{OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2}
		hBelow := diff.Hunk{OldStart: 18, OldCount: 2, NewStart: 18, NewCount: 2}
		state := &SeparatorState{HunkIndex: 1, MaxGap: 15}
		lines := ExpandedContextLines(state, hAbove, hBelow, oldLines, newLines, "down")
		if len(lines) != 10 { // capped at 10 per press
			t.Fatalf("expected 10 lines, got %d", len(lines))
		}
		if state.ExpandedDown != 10 {
			t.Errorf("ExpandedDown = %d, want 10", state.ExpandedDown)
		}
	})

	t.Run("incremental expand down", func(t *testing.T) {
		hAbove := diff.Hunk{OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2}
		hBelow := diff.Hunk{OldStart: 18, OldCount: 2, NewStart: 18, NewCount: 2}
		state := &SeparatorState{HunkIndex: 1, MaxGap: 15, ExpandedDown: 10}
		lines := ExpandedContextLines(state, hAbove, hBelow, oldLines, newLines, "down")
		if len(lines) != 5 { // only 5 remaining
			t.Fatalf("expected 5 lines, got %d", len(lines))
		}
		// Should start from old line 13 (2+10+1=13)
		if lines[0].Left == nil || lines[0].Left.OldNum != 13 {
			t.Errorf("first line OldNum = %v, want 13", lines[0].Left)
		}
		if state.ExpandedDown != 15 {
			t.Errorf("ExpandedDown = %d, want 15", state.ExpandedDown)
		}
	})

	t.Run("expand when already fully expanded returns nil", func(t *testing.T) {
		state := &SeparatorState{HunkIndex: 1, MaxGap: 6, ExpandedDown: 3, ExpandedUp: 3}
		lines := ExpandedContextLines(state, hunkAbove, hunkBelow, oldLines, newLines, "down")
		if len(lines) != 0 {
			t.Errorf("expected 0 lines when fully expanded, got %d", len(lines))
		}
	})
}
