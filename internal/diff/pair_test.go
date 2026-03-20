package diff

import (
	"testing"
)

func TestBuildPairedLines(t *testing.T) {
	tests := []struct {
		name  string
		hunks []Hunk
		check func(t *testing.T, pairs []PairedLine)
	}{
		{
			name:  "empty hunks returns empty slice",
			hunks: nil,
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 0 {
					t.Errorf("expected 0 pairs, got %d", len(pairs))
				}
			},
		},
		{
			name: "context lines appear on both sides",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 1, NewNum: 1, Content: "hello"},
						{Type: LineContext, OldNum: 2, NewNum: 2, Content: "world"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 2 {
					t.Fatalf("expected 2 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "hello", "hello")
				assertPair(t, pairs[1], "world", "world")
			},
		},
		{
			name: "equal deletes and adds are zipped with no padding",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
					Lines: []DiffLine{
						{Type: LineDelete, OldNum: 1, Content: "old1"},
						{Type: LineDelete, OldNum: 2, Content: "old2"},
						{Type: LineAdd, NewNum: 1, Content: "new1"},
						{Type: LineAdd, NewNum: 2, Content: "new2"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 2 {
					t.Fatalf("expected 2 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "old1", "new1")
				assertPair(t, pairs[1], "old2", "new2")
			},
		},
		{
			name: "more adds than deletes pads left with nil",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 3,
					Lines: []DiffLine{
						{Type: LineDelete, OldNum: 1, Content: "old"},
						{Type: LineAdd, NewNum: 1, Content: "new1"},
						{Type: LineAdd, NewNum: 2, Content: "new2"},
						{Type: LineAdd, NewNum: 3, Content: "new3"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 3 {
					t.Fatalf("expected 3 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "old", "new1")
				assertPairLeftNil(t, pairs[1], "new2")
				assertPairLeftNil(t, pairs[2], "new3")
			},
		},
		{
			name: "more deletes than adds pads right with nil",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 1,
					Lines: []DiffLine{
						{Type: LineDelete, OldNum: 1, Content: "old1"},
						{Type: LineDelete, OldNum: 2, Content: "old2"},
						{Type: LineDelete, OldNum: 3, Content: "old3"},
						{Type: LineAdd, NewNum: 1, Content: "new"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 3 {
					t.Fatalf("expected 3 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "old1", "new")
				assertPairRightNil(t, pairs[1], "old2")
				assertPairRightNil(t, pairs[2], "old3")
			},
		},
		{
			name: "3 deletions 5 additions produces 5 paired rows",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 5,
					Lines: []DiffLine{
						{Type: LineDelete, OldNum: 1, Content: "d1"},
						{Type: LineDelete, OldNum: 2, Content: "d2"},
						{Type: LineDelete, OldNum: 3, Content: "d3"},
						{Type: LineAdd, NewNum: 1, Content: "a1"},
						{Type: LineAdd, NewNum: 2, Content: "a2"},
						{Type: LineAdd, NewNum: 3, Content: "a3"},
						{Type: LineAdd, NewNum: 4, Content: "a4"},
						{Type: LineAdd, NewNum: 5, Content: "a5"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 5 {
					t.Fatalf("expected 5 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "d1", "a1")
				assertPair(t, pairs[1], "d2", "a2")
				assertPair(t, pairs[2], "d3", "a3")
				assertPairLeftNil(t, pairs[3], "a4")
				assertPairLeftNil(t, pairs[4], "a5")
			},
		},
		{
			name: "new file has all left sides nil",
			hunks: []Hunk{
				{
					OldStart: 0, OldCount: 0, NewStart: 1, NewCount: 3,
					Lines: []DiffLine{
						{Type: LineAdd, NewNum: 1, Content: "line1"},
						{Type: LineAdd, NewNum: 2, Content: "line2"},
						{Type: LineAdd, NewNum: 3, Content: "line3"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 3 {
					t.Fatalf("expected 3 pairs, got %d", len(pairs))
				}
				for i, p := range pairs {
					if p.Left != nil {
						t.Errorf("pair[%d]: expected Left=nil", i)
					}
					if p.Right == nil {
						t.Errorf("pair[%d]: expected Right non-nil", i)
					}
				}
			},
		},
		{
			name: "deleted file has all right sides nil",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 2, NewStart: 0, NewCount: 0,
					Lines: []DiffLine{
						{Type: LineDelete, OldNum: 1, Content: "line1"},
						{Type: LineDelete, OldNum: 2, Content: "line2"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 2 {
					t.Fatalf("expected 2 pairs, got %d", len(pairs))
				}
				for i, p := range pairs {
					if p.Left == nil {
						t.Errorf("pair[%d]: expected Left non-nil", i)
					}
					if p.Right != nil {
						t.Errorf("pair[%d]: expected Right=nil", i)
					}
				}
			},
		},
		{
			name: "multiple hunks have separator between them",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 1, NewNum: 1, Content: "aaa"},
					},
				},
				{
					OldStart: 10, OldCount: 1, NewStart: 10, NewCount: 1,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 10, NewNum: 10, Content: "bbb"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				// hunk0 lines + separator + hunk1 lines = 1 + 1 + 1 = 3
				if len(pairs) != 3 {
					t.Fatalf("expected 3 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "aaa", "aaa")
				if !pairs[1].IsSeparator {
					t.Error("expected separator between hunks")
				}
				if pairs[1].HunkIndex != 1 {
					t.Errorf("separator HunkIndex: got %d, want 1", pairs[1].HunkIndex)
				}
				assertPair(t, pairs[2], "bbb", "bbb")
			},
		},
		{
			name: "interleaved changes within a hunk",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 4, NewStart: 1, NewCount: 5,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 1, NewNum: 1, Content: "line A"},
						{Type: LineDelete, OldNum: 2, Content: "line B"},
						{Type: LineDelete, OldNum: 3, Content: "line C"},
						{Type: LineAdd, NewNum: 2, Content: "line D"},
						{Type: LineAdd, NewNum: 3, Content: "line E"},
						{Type: LineAdd, NewNum: 4, Content: "line F"},
						{Type: LineContext, OldNum: 4, NewNum: 5, Content: "line G"},
						{Type: LineDelete, OldNum: 5, Content: "line H"},
						{Type: LineAdd, NewNum: 6, Content: "line I"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 6 {
					t.Fatalf("expected 6 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "line A", "line A") // context
				assertPair(t, pairs[1], "line B", "line D") // del/add zipped
				assertPair(t, pairs[2], "line C", "line E") // del/add zipped
				assertPairLeftNil(t, pairs[3], "line F")           // padding, adds > dels
				assertPair(t, pairs[4], "line G", "line G") // context
				assertPair(t, pairs[5], "line H", "line I") // second group
			},
		},
		{
			name: "hunk index is set correctly on paired lines",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 1, NewNum: 1, Content: "x"},
					},
				},
				{
					OldStart: 20, OldCount: 1, NewStart: 20, NewCount: 1,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 20, NewNum: 20, Content: "y"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if pairs[0].HunkIndex != 0 {
					t.Errorf("first line HunkIndex: got %d, want 0", pairs[0].HunkIndex)
				}
				if pairs[2].HunkIndex != 1 {
					t.Errorf("third line HunkIndex: got %d, want 1", pairs[2].HunkIndex)
				}
			},
		},
		{
			name: "only deletes no adds in group",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 4, NewStart: 1, NewCount: 2,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 1, NewNum: 1, Content: "keep"},
						{Type: LineDelete, OldNum: 2, Content: "removed1"},
						{Type: LineDelete, OldNum: 3, Content: "removed2"},
						{Type: LineContext, OldNum: 4, NewNum: 2, Content: "also keep"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 4 {
					t.Fatalf("expected 4 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "keep", "keep")
				assertPairRightNil(t, pairs[1], "removed1")
				assertPairRightNil(t, pairs[2], "removed2")
				assertPair(t, pairs[3], "also keep", "also keep")
			},
		},
		{
			name: "only adds no deletes in group",
			hunks: []Hunk{
				{
					OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 4,
					Lines: []DiffLine{
						{Type: LineContext, OldNum: 1, NewNum: 1, Content: "keep"},
						{Type: LineAdd, NewNum: 2, Content: "inserted1"},
						{Type: LineAdd, NewNum: 3, Content: "inserted2"},
						{Type: LineContext, OldNum: 2, NewNum: 4, Content: "also keep"},
					},
				},
			},
			check: func(t *testing.T, pairs []PairedLine) {
				if len(pairs) != 4 {
					t.Fatalf("expected 4 pairs, got %d", len(pairs))
				}
				assertPair(t, pairs[0], "keep", "keep")
				assertPairLeftNil(t, pairs[1], "inserted1")
				assertPairLeftNil(t, pairs[2], "inserted2")
				assertPair(t, pairs[3], "also keep", "also keep")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs := BuildPairedLines(tt.hunks)
			tt.check(t, pairs)
		})
	}
}

func assertPair(t *testing.T, p PairedLine, wantLeft, wantRight string) {
	t.Helper()
	if p.IsSeparator {
		t.Error("IsSeparator: got true, want false")
	}
	if p.Left == nil {
		t.Errorf("Left is nil, want content %q", wantLeft)
	} else if p.Left.Content != wantLeft {
		t.Errorf("Left.Content: got %q, want %q", p.Left.Content, wantLeft)
	}
	if p.Right == nil {
		t.Errorf("Right is nil, want content %q", wantRight)
	} else if p.Right.Content != wantRight {
		t.Errorf("Right.Content: got %q, want %q", p.Right.Content, wantRight)
	}
}

func assertPairLeftNil(t *testing.T, p PairedLine, wantRight string) {
	t.Helper()
	if p.Left != nil {
		t.Errorf("Left: got %q, want nil", p.Left.Content)
	}
	if p.Right == nil {
		t.Errorf("Right is nil, want content %q", wantRight)
	} else if p.Right.Content != wantRight {
		t.Errorf("Right.Content: got %q, want %q", p.Right.Content, wantRight)
	}
}

func assertPairRightNil(t *testing.T, p PairedLine, wantLeft string) {
	t.Helper()
	if p.Left == nil {
		t.Errorf("Left is nil, want content %q", wantLeft)
	} else if p.Left.Content != wantLeft {
		t.Errorf("Left.Content: got %q, want %q", p.Left.Content, wantLeft)
	}
	if p.Right != nil {
		t.Errorf("Right: got %q, want nil", p.Right.Content)
	}
}
