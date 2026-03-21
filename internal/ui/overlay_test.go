package ui

import (
	"strings"
	"testing"
)

func TestNewCommentOverlay(t *testing.T) {
	o := NewCommentOverlay(440, SideNew, 10)
	if !o.Active {
		t.Error("overlay should be active on creation")
	}
	if o.Line != 440 {
		t.Errorf("Line: got %d, want 440", o.Line)
	}
	if o.Side != SideNew {
		t.Errorf("Side: got %v, want SideNew", o.Side)
	}
	if o.RowIndex != 10 {
		t.Errorf("RowIndex: got %d, want 10", o.RowIndex)
	}
	// Input should be focused
	if !o.Input.Focused() {
		t.Error("text input should be focused on creation")
	}
}

func TestCommentOverlayRender(t *testing.T) {
	tests := []struct {
		name     string
		line     int
		side     Side
		width    int
		contains []string
	}{
		{
			name:  "shows header with line number and side",
			line:  440,
			side:  SideNew,
			width: 80,
			contains: []string{
				"Comment on L440 (new):",
			},
		},
		{
			name:  "shows header for old side",
			line:  123,
			side:  SideOld,
			width: 80,
			contains: []string{
				"Comment on L123 (old):",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewCommentOverlay(tt.line, tt.side, 5)
			output := o.Render(tt.width)
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("expected output to contain %q, got:\n%s", s, output)
				}
			}
		})
	}
}

func TestCommentOverlayPosition(t *testing.T) {
	tests := []struct {
		name       string
		rowIndex   int
		yOffset    int
		visibleRow int
		wantAbove  bool // true = overlay above the line, false = below
	}{
		{
			name:       "cursor in top half places overlay below",
			rowIndex:   2,
			yOffset:    0,
			visibleRow: 20,
			wantAbove:  false,
		},
		{
			name:       "cursor in bottom half places overlay above",
			rowIndex:   15,
			yOffset:    0,
			visibleRow: 20,
			wantAbove:  true,
		},
		{
			name:       "cursor exactly at midpoint places below",
			rowIndex:   10,
			yOffset:    0,
			visibleRow: 20,
			wantAbove:  false,
		},
		{
			name:       "cursor with yOffset in bottom half places above",
			rowIndex:   25,
			yOffset:    10,
			visibleRow: 20,
			wantAbove:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewCommentOverlay(100, SideNew, tt.rowIndex)
			got := o.ShouldFlipAbove(tt.yOffset, tt.visibleRow)
			if got != tt.wantAbove {
				t.Errorf("ShouldFlipAbove: got %v, want %v", got, tt.wantAbove)
			}
		})
	}
}

func TestCommentOverlayValue(t *testing.T) {
	o := NewCommentOverlay(440, SideNew, 10)
	// Initially empty
	if o.Value() != "" {
		t.Errorf("Value: got %q, want empty string", o.Value())
	}
}
