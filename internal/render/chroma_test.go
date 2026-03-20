package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// reconstructText joins all segment texts in a HighlightedLine.
func reconstructText(t *testing.T, hl HighlightedLine) string {
	t.Helper()
	var b strings.Builder
	for _, seg := range hl.Segments {
		b.WriteString(seg.Text)
	}
	return b.String()
}

func TestHighlightFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		check    func(t *testing.T, lines []HighlightedLine)
	}{
		{
			name:     "Go source produces multiple segments",
			filename: "main.go",
			content:  "package main\n\nfunc hello() string {\n\treturn \"world\"\n}\n",
			check: func(t *testing.T, lines []HighlightedLine) {
				t.Helper()
				if len(lines) < 5 {
					t.Fatalf("want at least 5 lines, got %d", len(lines))
				}
				text := reconstructText(t, lines[0])
				if text != "package main" {
					t.Errorf("line 0 text = %q, want %q", text, "package main")
				}
				if len(lines[0].Segments) < 2 {
					t.Errorf("expected multiple segments for Go syntax, got %d", len(lines[0].Segments))
				}
			},
		},
		{
			name:     "unrecognized file type falls back to plain text",
			filename: "file.xyzunknown999",
			content:  "some random content\nanother line\n",
			check: func(t *testing.T, lines []HighlightedLine) {
				t.Helper()
				if len(lines) < 2 {
					t.Fatalf("want at least 2 lines, got %d", len(lines))
				}
				text := reconstructText(t, lines[0])
				if text != "some random content" {
					t.Errorf("line 0 text = %q, want %q", text, "some random content")
				}
			},
		},
		{
			name:     "empty content returns nil",
			filename: "main.go",
			content:  "",
			check: func(t *testing.T, lines []HighlightedLine) {
				t.Helper()
				if len(lines) != 0 {
					t.Errorf("expected 0 lines for empty content, got %d", len(lines))
				}
			},
		},
		{
			name:     "new file old side empty",
			filename: "new_file.go",
			content:  "package main\n",
			check: func(t *testing.T, lines []HighlightedLine) {
				t.Helper()
				if len(lines) < 1 {
					t.Fatalf("expected at least 1 line, got %d", len(lines))
				}
				text := reconstructText(t, lines[0])
				if text != "package main" {
					t.Errorf("line 0 text = %q, want %q", text, "package main")
				}
			},
		},
		{
			name:     "deleted file content highlighted normally",
			filename: "deleted.go",
			content:  "func old() {}\n",
			check: func(t *testing.T, lines []HighlightedLine) {
				t.Helper()
				if len(lines) < 1 {
					t.Fatalf("expected at least 1 line, got %d", len(lines))
				}
				text := reconstructText(t, lines[0])
				if text != "func old() {}" {
					t.Errorf("line 0 text = %q, want %q", text, "func old() {}")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRenderer()
			lines := r.HighlightFile(tt.filename, tt.content)
			tt.check(t, lines)
		})
	}
}

func TestHighlightFile_Cache(t *testing.T) {
	r := NewRenderer()
	content := "package main\n"
	lines1 := r.HighlightFile("main.go", content)
	lines2 := r.HighlightFile("main.go", content)

	if len(lines1) != len(lines2) {
		t.Fatalf("cache mismatch: %d vs %d lines", len(lines1), len(lines2))
	}

	for i := range lines1 {
		t1 := reconstructText(t, lines1[i])
		t2 := reconstructText(t, lines2[i])
		if t1 != t2 {
			t.Errorf("cache mismatch at line %d: %q vs %q", i, t1, t2)
		}
	}
}

func TestHighlightFileWithKey_SeparateCache(t *testing.T) {
	r := NewRenderer()
	oldContent := "package old\n"
	newContent := "package new\n"

	oldLines := r.HighlightFileWithKey("old:main.go", "main.go", oldContent)
	newLines := r.HighlightFileWithKey("new:main.go", "main.go", newContent)

	oldText := reconstructText(t, oldLines[0])
	newText := reconstructText(t, newLines[0])

	if oldText == newText {
		t.Errorf("expected different cache entries, both returned %q", oldText)
	}
	if oldText != "package old" {
		t.Errorf("old text = %q, want %q", oldText, "package old")
	}
	if newText != "package new" {
		t.Errorf("new text = %q, want %q", newText, "package new")
	}
}

func TestComputeWordDiff(t *testing.T) {
	tests := []struct {
		name    string
		oldLine string
		newLine string
		wantOld []bool
		wantNew []bool
	}{
		{
			name:    "changed first char",
			oldLine: "hello",
			newLine: "jello",
			wantOld: []bool{true, false, false, false, false},
			wantNew: []bool{true, false, false, false, false},
		},
		{
			name:    "identical lines",
			oldLine: "hello world",
			newLine: "hello world",
			wantOld: []bool{false, false, false, false, false, false, false, false, false, false, false},
			wantNew: []bool{false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			name:    "empty old",
			oldLine: "",
			newLine: "new content",
			wantOld: []bool{},
			wantNew: []bool{true, true, true, true, true, true, true, true, true, true, true},
		},
		{
			name:    "empty new",
			oldLine: "old content",
			newLine: "",
			wantOld: []bool{true, true, true, true, true, true, true, true, true, true, true},
			wantNew: []bool{},
		},
		{
			name:    "both empty",
			oldLine: "",
			newLine: "",
			wantOld: []bool{},
			wantNew: []bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldMask, newMask := ComputeWordDiff(tt.oldLine, tt.newLine)

			if len(oldMask) != len(tt.wantOld) {
				t.Fatalf("oldMask length = %d, want %d", len(oldMask), len(tt.wantOld))
			}
			if len(newMask) != len(tt.wantNew) {
				t.Fatalf("newMask length = %d, want %d", len(newMask), len(tt.wantNew))
			}

			for i, want := range tt.wantOld {
				if oldMask[i] != want {
					t.Errorf("oldMask[%d] = %v, want %v", i, oldMask[i], want)
				}
			}
			for i, want := range tt.wantNew {
				if newMask[i] != want {
					t.Errorf("newMask[%d] = %v, want %v", i, newMask[i], want)
				}
			}
		})
	}
}

func TestApplyEmphasis(t *testing.T) {
	tests := []struct {
		name         string
		line         HighlightedLine
		mask         []bool
		emphasisBg   string
		wantSegments int
		wantText     string
		check        func(t *testing.T, result HighlightedLine)
	}{
		{
			name: "full token emphasized",
			line: HighlightedLine{
				Segments: []Segment{
					{Text: "rate", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("2"))},
				},
			},
			mask:         []bool{true, true, true, true},
			emphasisBg:   "52",
			wantSegments: 1,
			wantText:     "rate",
			check: func(t *testing.T, result HighlightedLine) {
				t.Helper()
				expectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Background(lipgloss.Color("52"))
				got := result.Segments[0].Style.Render("test")
				want := expectedStyle.Render("test")
				if got != want {
					t.Errorf("emphasis style render = %q, want %q", got, want)
				}
			},
		},
		{
			name: "partial token emphasized",
			line: HighlightedLine{
				Segments: []Segment{
					{Text: "helloworld", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("2"))},
				},
			},
			mask:         []bool{false, false, false, false, false, true, true, true, true, true},
			emphasisBg:   "52",
			wantSegments: 2,
			wantText:     "helloworld",
			check: func(t *testing.T, result HighlightedLine) {
				t.Helper()
				if result.Segments[0].Text != "hello" {
					t.Errorf("segment 0 text = %q, want %q", result.Segments[0].Text, "hello")
				}
				if result.Segments[1].Text != "world" {
					t.Errorf("segment 1 text = %q, want %q", result.Segments[1].Text, "world")
				}
			},
		},
		{
			name: "no emphasis preserves segments",
			line: HighlightedLine{
				Segments: []Segment{
					{Text: "hello", Style: lipgloss.NewStyle()},
					{Text: " world", Style: lipgloss.NewStyle()},
				},
			},
			mask:         []bool{false, false, false, false, false, false, false, false, false, false, false},
			emphasisBg:   "52",
			wantSegments: 2,
			wantText:     "hello world",
		},
		{
			name: "empty mask returns line unchanged",
			line: HighlightedLine{
				Segments: []Segment{
					{Text: "hello", Style: lipgloss.NewStyle()},
				},
			},
			mask:         nil,
			emphasisBg:   "52",
			wantSegments: 1,
			wantText:     "hello",
		},
		{
			name: "multiple segments with targeted emphasis",
			line: HighlightedLine{
				Segments: []Segment{
					{Text: "const", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("4"))},
					{Text: " ", Style: lipgloss.NewStyle()},
					{Text: "rate", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("2"))},
					{Text: " = ", Style: lipgloss.NewStyle()},
					{Text: "1", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("3"))},
				},
			},
			// Emphasize "rate" (positions 6-9): "const"=5, " "=1, "rate" starts at 6
			mask:       []bool{false, false, false, false, false, false, true, true, true, true, false, false, false, false},
			emphasisBg: "52",
			wantText:   "const rate = 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyEmphasis(tt.line, tt.mask, tt.emphasisBg)

			text := reconstructText(t, result)
			if text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}

			if tt.wantSegments > 0 && len(result.Segments) != tt.wantSegments {
				t.Errorf("segments = %d, want %d", len(result.Segments), tt.wantSegments)
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestRenderLine(t *testing.T) {
	tests := []struct {
		name      string
		line      HighlightedLine
		contains  []string
		wantEmpty bool
	}{
		{
			name: "renders styled segments",
			line: HighlightedLine{
				Segments: []Segment{
					{Text: "hello", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("2"))},
					{Text: " world", Style: lipgloss.NewStyle()},
				},
			},
			contains: []string{"hello", "world"},
		},
		{
			name:      "empty line renders empty",
			line:      HighlightedLine{},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderLine(tt.line)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}

			if result == "" {
				t.Error("expected non-empty rendered output")
			}
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("rendered output should contain %q: %q", s, result)
				}
			}
		})
	}
}
