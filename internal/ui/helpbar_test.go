package ui

import (
	"strings"
	"testing"
)

func TestRenderHelpBar(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		mode     InputMode
		contains []string
	}{
		{
			name:  "normal mode shows navigation hints",
			width: 120,
			mode:  InputModeNormal,
			contains: []string{
				"j/k", "scroll",
				"h/l", "pane",
				"Tab", "tree",
				"c", "comment",
				"/", "search",
				"q", "quit",
			},
		},
		{
			name:  "non-empty output",
			width: 80,
			mode:  InputModeNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderHelpBar(tt.width, tt.mode)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected output to contain %q, got: %s", s, result)
				}
			}

			if result == "" {
				t.Error("RenderHelpBar returned empty string")
			}
		})
	}
}
