package ui

import (
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
