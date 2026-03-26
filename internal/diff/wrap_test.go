package diff

import (
	"testing"
)

func TestVisualRowCount(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		paneWidth int
		want      int
	}{
		{
			name:      "short content fits in one row",
			content:   "hello",
			paneWidth: 20,
			want:      1,
		},
		{
			name:      "exact fit is one row",
			content:   "12345",
			paneWidth: 5,
			want:      1,
		},
		{
			name:      "content wraps to two rows",
			content:   "1234567890",
			paneWidth: 6,
			want:      2,
		},
		{
			name:      "content wraps to three rows",
			content:   "123456789012345",
			paneWidth: 6,
			want:      3,
		},
		{
			name:      "empty content is one row",
			content:   "",
			paneWidth: 10,
			want:      1,
		},
		{
			name:      "zero pane width returns one row",
			content:   "hello",
			paneWidth: 0,
			want:      1,
		},
		{
			name:      "negative pane width returns one row",
			content:   "hello",
			paneWidth: -5,
			want:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VisualRowCount(tt.content, tt.paneWidth)
			if got != tt.want {
				t.Errorf("VisualRowCount(%q, %d) = %d, want %d", tt.content, tt.paneWidth, got, tt.want)
			}
		})
	}
}

func TestBuildWrappedPairedLines(t *testing.T) {
	tests := []struct {
		name      string
		hunks     []Hunk
		paneWidth int
		check     func(t *testing.T, pairs []PairedLine)
	}{
		{
			name:      "empty hunks returns nil",
			hunks:     nil,
			paneWidth: 40,
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 0 {
					t.Errorf("expected 0 pairs, got %d", len(pairs))
				}
			},
		},
		{
			name: "short lines produce same result as non-wrapped",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 1, NewNum: 1, Content: "short"},
						{Type: LineContext, OldNum: 2, NewNum: 2, Content: "also short"},
					},
				},
			},
			paneWidth: 40,
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 2 {
					t.Fatalf("expected 2 pairs, got %d", len(pairs))
				}
				// Short lines — no extra visual rows
				if pairs[0].Left == nil || pairs[0].Right == nil {
					t.Error("context line should have both sides")
				}
			},
		},
		{
			name: "long left line expands with padding on right",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
					Lines: []DiffLine{
						{Type: LineDelete, OldNum: 1, Content: "this is a very long line that should wrap"},
						{Type: LineAdd, NewNum: 1, Content: "short"},
					},
				},
			},
			paneWidth: 10, // "this is a very long line that should wrap" is 41 chars → 5 visual rows
			check: func(t *testing.T, pairs []PairedLine) {
				// Left side needs ceil(41/10)=5 visual rows, right needs 1
				// Max = 5, so we should get 5 rows for this paired line
				if len(pairs) < 5 {
					t.Fatalf("expected at least 5 rows for wrapped content, got %d", len(pairs))
				}
				// First row should have both left and right content
				if pairs[0].Left == nil {
					t.Error("first row should have left content")
				}
				if pairs[0].Right == nil {
					t.Error("first row should have right content")
				}
				// Subsequent rows should be continuation (IsWrapContinuation)
				for i := 1; i < 5; i++ {
					if !pairs[i].IsWrapContinuation {
						t.Errorf("row %d should be wrap continuation", i)
					}
				}
			},
		},
		{
			name: "separator rows preserved in wrapped output",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 1, NewNum: 1, Content: "short"},
					},
				},
				{
					OldStart: 10, OldCount: 1, NewStart: 10, NewCount: 1,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 10, NewNum: 10, Content: "also short"},
					},
				},
			},
			paneWidth: 40,
			check: func(t *testing.T, pairs []PairedLine) {
				hasSep := false
				for _, p := range pairs {
					if p.IsSeparator {
						hasSep = true
						break
					}
				}
				if !hasSep {
					t.Error("expected separator between hunks in wrapped output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs := BuildWrappedPairedLines(tt.hunks, tt.paneWidth)
			tt.check(t, pairs)
		})
	}
}
