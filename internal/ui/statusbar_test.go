package ui

import (
	"os"
	"strings"
	"testing"
)

func TestRenderStatusBar(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		ref          string
		fileIdx      int
		fileCount    int
		filePath     string
		adds         int
		dels         int
		commentCount int
		side         Side
		contains     []string
	}{
		{
			name:         "shows ref range",
			width:        120,
			ref:          "main..HEAD",
			fileIdx:      2,
			fileCount:    17,
			filePath:     "src/services/auth-handler.ts",
			adds:         42,
			dels:         8,
			commentCount: 1,
			side:         SideNew,
			contains:     []string{"cr: main..HEAD", "[3/17]", "src/services/auth-handler.ts", "+42", "-8", "C:1", "new"},
		},
		{
			name:         "working tree shows (working tree)",
			width:        120,
			ref:          "",
			fileIdx:      0,
			fileCount:    1,
			filePath:     "main.go",
			adds:         5,
			dels:         0,
			commentCount: 0,
			side:         SideNew,
			contains:     []string{"cr: (working tree)", "[1/1]", "main.go", "+5", "-0", "C:0", "new"},
		},
		{
			name:         "old side indicator",
			width:        120,
			ref:          "main",
			fileIdx:      0,
			fileCount:    3,
			filePath:     "config.go",
			adds:         0,
			dels:         10,
			commentCount: 0,
			side:         SideOld,
			contains:     []string{"old"},
		},
		{
			name:         "file position updates correctly",
			width:        120,
			ref:          "main..HEAD",
			fileIdx:      9,
			fileCount:    20,
			filePath:     "deep/nested/file.go",
			adds:         100,
			dels:         50,
			commentCount: 3,
			side:         SideNew,
			contains:     []string{"[10/20]", "+100", "-50", "C:3"},
		},
		{
			name:         "output fills full width",
			width:        80,
			ref:          "main",
			fileIdx:      0,
			fileCount:    1,
			filePath:     "a.go",
			adds:         1,
			dels:         1,
			commentCount: 0,
			side:         SideNew,
		},
		{
			name:         "rename path display",
			width:        120,
			ref:          "main..HEAD",
			fileIdx:      0,
			fileCount:    5,
			filePath:     "old/path.ts → new/path.ts",
			adds:         10,
			dels:         5,
			commentCount: 0,
			side:         SideNew,
			contains:     []string{"old/path.ts → new/path.ts"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderStatusBar(tt.width, tt.ref, tt.fileIdx, tt.fileCount, tt.filePath, tt.adds, tt.dels, tt.commentCount, tt.side)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected output to contain %q, got: %s", s, result)
				}
			}

			if result == "" {
				t.Error("RenderStatusBar returned empty string")
			}
		})
	}
}

func TestRenderStatusBarWithMode(t *testing.T) {
	tests := []struct {
		name     string
		oldMode  os.FileMode
		newMode  os.FileMode
		contains []string
		absent   []string
	}{
		{
			name:     "shows mode change when different",
			oldMode:  0644,
			newMode:  0755,
			contains: []string{"0644", "0755"},
		},
		{
			name:    "hides mode change when same",
			oldMode: 0644,
			newMode: 0644,
			absent:  []string{"0644"},
		},
		{
			name:    "hides mode change when old is zero",
			oldMode: 0,
			newMode: 0644,
			absent:  []string{"→"},
		},
		{
			name:    "hides mode change when new is zero",
			oldMode: 0644,
			newMode: 0,
			absent:  []string{"→"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderStatusBarWithMode(120, "main..HEAD", 0, 5, "script.sh", 0, 0, 0, SideNew, tt.oldMode, tt.newMode)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected output to contain %q, got: %s", s, result)
				}
			}
			for _, s := range tt.absent {
				if strings.Contains(result, s) {
					t.Errorf("expected output to NOT contain %q, got: %s", s, result)
				}
			}
		})
	}
}
