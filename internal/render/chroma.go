package render

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Segment represents a piece of text with a lipgloss style.
type Segment struct {
	Text  string
	Style lipgloss.Style
}

// HighlightedLine represents a single line of syntax-highlighted text as segments.
type HighlightedLine struct {
	Segments []Segment
}

// Renderer provides syntax highlighting and word-level diffs.
// It caches highlighted lines per cache key (filename or ref:path).
type Renderer struct {
	cache map[string][]HighlightedLine
}

// NewRenderer creates a new Renderer with an empty cache.
func NewRenderer() *Renderer {
	return &Renderer{
		cache: make(map[string][]HighlightedLine),
	}
}

// HighlightFile tokenizes file content with chroma and returns highlighted lines.
// The filename is used for both language detection and cache key.
// If the file type is unrecognized, falls back to plain text.
// Empty content returns nil.
func (r *Renderer) HighlightFile(filename, content string) []HighlightedLine {
	return r.HighlightFileWithKey(filename, filename, content)
}

// HighlightFileWithKey tokenizes file content with chroma using a separate cache key.
// Use this when the same filename has different content for old/new sides.
func (r *Renderer) HighlightFileWithKey(cacheKey, filename, content string) []HighlightedLine {
	if content == "" {
		return nil
	}

	if cached, ok := r.cache[cacheKey]; ok {
		return cached
	}

	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		lines := plainTextLines(content)
		r.cache[cacheKey] = lines
		return lines
	}

	var lines []HighlightedLine
	var currentLine HighlightedLine

	for _, token := range iterator.Tokens() {
		style := tokenStyle(token.Type)

		parts := strings.Split(token.Value, "\n")
		for i, part := range parts {
			if i > 0 {
				lines = append(lines, currentLine)
				currentLine = HighlightedLine{}
			}
			if part != "" {
				currentLine.Segments = append(currentLine.Segments, Segment{
					Text:  part,
					Style: style,
				})
			}
		}
	}

	// Append last line only if it has segments
	if len(currentLine.Segments) > 0 {
		lines = append(lines, currentLine)
	}

	r.cache[cacheKey] = lines
	return lines
}

// ComputeWordDiff computes byte-position emphasis masks for a pair of changed lines.
// Returns masks where true means the byte at that position was changed.
func ComputeWordDiff(oldLine, newLine string) ([]bool, []bool) {
	oldMask := make([]bool, len(oldLine))
	newMask := make([]bool, len(newLine))

	if oldLine == newLine {
		return oldMask, newMask
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldLine, newLine, false)

	oldIdx := 0
	newIdx := 0
	for _, d := range diffs {
		n := len(d.Text)
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			oldIdx += n
			newIdx += n
		case diffmatchpatch.DiffDelete:
			for i := 0; i < n; i++ {
				oldMask[oldIdx+i] = true
			}
			oldIdx += n
		case diffmatchpatch.DiffInsert:
			for i := 0; i < n; i++ {
				newMask[newIdx+i] = true
			}
			newIdx += n
		}
	}

	return oldMask, newMask
}

// ApplyEmphasis merges an emphasis mask into highlighted segments.
// Bytes with mask[i]=true get an additional background color.
// The emphasisBg parameter is a lipgloss color string for the emphasis background.
func ApplyEmphasis(line HighlightedLine, mask []bool, emphasisBg string) HighlightedLine {
	if len(mask) == 0 {
		return line
	}

	var result HighlightedLine
	pos := 0

	for _, seg := range line.Segments {
		segLen := len(seg.Text)

		if pos+segLen > len(mask) {
			result.Segments = append(result.Segments, seg)
			pos += segLen
			continue
		}

		segMask := mask[pos : pos+segLen]

		// Find contiguous runs of same emphasis value
		i := 0
		for i < segLen {
			emphasized := segMask[i]
			j := i
			for j < segLen && segMask[j] == emphasized {
				j++
			}

			subText := seg.Text[i:j]
			style := seg.Style
			if emphasized {
				style = style.Background(lipgloss.Color(emphasisBg))
			}
			result.Segments = append(result.Segments, Segment{
				Text:  subText,
				Style: style,
			})
			i = j
		}
		pos += segLen
	}

	return result
}

// RenderLine converts a highlighted line to a styled string.
func RenderLine(line HighlightedLine) string {
	var b strings.Builder
	for _, seg := range line.Segments {
		b.WriteString(seg.Style.Render(seg.Text))
	}
	return b.String()
}

// plainTextLines splits content into lines with no syntax styling.
// Empty lines are preserved as HighlightedLine with no segments.
func plainTextLines(content string) []HighlightedLine {
	raw := strings.Split(content, "\n")
	// Trim trailing empty string from final newline
	if len(raw) > 0 && raw[len(raw)-1] == "" {
		raw = raw[:len(raw)-1]
	}
	lines := make([]HighlightedLine, len(raw))
	for i, r := range raw {
		if r != "" {
			lines[i] = HighlightedLine{
				Segments: []Segment{{Text: r, Style: lipgloss.NewStyle()}},
			}
		}
	}
	return lines
}

// tokenStyle maps a chroma token type to a lipgloss style.
func tokenStyle(tt chroma.TokenType) lipgloss.Style {
	s := lipgloss.NewStyle()

	switch {
	// Keywords
	case tt == chroma.Keyword || tt == chroma.KeywordConstant ||
		tt == chroma.KeywordDeclaration || tt == chroma.KeywordNamespace ||
		tt == chroma.KeywordPseudo || tt == chroma.KeywordReserved ||
		tt == chroma.KeywordType:
		s = s.Foreground(lipgloss.Color("#FF7733"))

	// Strings
	case tt == chroma.LiteralString || tt == chroma.LiteralStringAffix ||
		tt == chroma.LiteralStringBacktick || tt == chroma.LiteralStringChar ||
		tt == chroma.LiteralStringDelimiter || tt == chroma.LiteralStringDoc ||
		tt == chroma.LiteralStringDouble || tt == chroma.LiteralStringEscape ||
		tt == chroma.LiteralStringHeredoc || tt == chroma.LiteralStringInterpol ||
		tt == chroma.LiteralStringOther || tt == chroma.LiteralStringRegex ||
		tt == chroma.LiteralStringSingle || tt == chroma.LiteralStringSymbol:
		s = s.Foreground(lipgloss.Color("#98C379"))

	// Numbers
	case tt == chroma.LiteralNumber || tt == chroma.LiteralNumberBin ||
		tt == chroma.LiteralNumberFloat || tt == chroma.LiteralNumberHex ||
		tt == chroma.LiteralNumberInteger || tt == chroma.LiteralNumberIntegerLong ||
		tt == chroma.LiteralNumberOct:
		s = s.Foreground(lipgloss.Color("#D19A66"))

	// Comments
	case tt == chroma.Comment || tt == chroma.CommentMultiline ||
		tt == chroma.CommentPreproc || tt == chroma.CommentPreprocFile ||
		tt == chroma.CommentSingle || tt == chroma.CommentSpecial:
		s = s.Foreground(lipgloss.Color("#5C6370"))

	// Functions / Names
	case tt == chroma.NameFunction || tt == chroma.NameFunctionMagic ||
		tt == chroma.NameBuiltin || tt == chroma.NameBuiltinPseudo:
		s = s.Foreground(lipgloss.Color("#61AFEF"))

	// Operators
	case tt == chroma.Operator || tt == chroma.OperatorWord:
		s = s.Foreground(lipgloss.Color("#56B6C2"))

	// Punctuation
	case tt == chroma.Punctuation:
		s = s.Foreground(lipgloss.Color("#ABB2BF"))

	// Types / Classes
	case tt == chroma.NameClass || tt == chroma.NameException ||
		tt == chroma.NameDecorator:
		s = s.Foreground(lipgloss.Color("#E5C07B"))
	}

	return s
}
