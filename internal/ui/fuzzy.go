package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/adilp/crit-diff/internal/diff"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// FuzzyMode distinguishes file search from content search.
type FuzzyMode int

const (
	FuzzyModeFiles FuzzyMode = iota
	FuzzyModeContent
)

// ContentItem holds a content search line with navigation metadata.
type ContentItem struct {
	Display   string        // formatted line: "file.go:42 [+]: content"
	FileIndex int           // index into Model.files
	LineNum   int           // line number in the file
	LineType  diff.LineType // type of the diff line (add, delete, context)
}

// FuzzyState holds the state for the built-in fuzzy overlay.
type FuzzyState struct {
	Active       bool
	Mode         FuzzyMode
	Items        []string      // all items (display strings)
	ContentItems []ContentItem // populated for FuzzyModeContent
	Filtered     []int         // indices into Items that match the filter
	Cursor       int           // cursor position in Filtered
	Input        textinput.Model
	YOffset      int // scroll offset for the filtered list
}

// NewFuzzyState creates a new fuzzy state with the given items.
func NewFuzzyState(mode FuzzyMode, items []string) FuzzyState {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Focus()
	ti.CharLimit = 0

	filtered := make([]int, len(items))
	for i := range items {
		filtered[i] = i
	}

	return FuzzyState{
		Active:   true,
		Mode:     mode,
		Items:    items,
		Filtered: filtered,
		Input:    ti,
	}
}

// CursorDown moves the cursor down in the filtered list.
func (fs *FuzzyState) CursorDown() {
	if fs.Cursor < len(fs.Filtered)-1 {
		fs.Cursor++
	}
}

// CursorUp moves the cursor up in the filtered list.
func (fs *FuzzyState) CursorUp() {
	if fs.Cursor > 0 {
		fs.Cursor--
	}
}

// EnsureCursorVisible adjusts YOffset so the cursor is within the visible window.
func (fs *FuzzyState) EnsureCursorVisible(maxItems int) {
	if fs.Cursor < fs.YOffset {
		fs.YOffset = fs.Cursor
	}
	if fs.Cursor >= fs.YOffset+maxItems {
		fs.YOffset = fs.Cursor - maxItems + 1
	}
}

// UpdateFilter re-filters the items based on the query and resets cursor.
func (fs *FuzzyState) UpdateFilter(query string) {
	fs.Filtered = FuzzyFilterIndices(fs.Items, query)
	fs.Cursor = 0
	fs.YOffset = 0
}

// SelectedItem returns the currently selected item string.
func (fs *FuzzyState) SelectedItem() (string, bool) {
	if len(fs.Filtered) == 0 {
		return "", false
	}
	if fs.Cursor >= len(fs.Filtered) {
		return "", false
	}
	return fs.Items[fs.Filtered[fs.Cursor]], true
}

// SelectedContentItem returns the currently selected ContentItem (for content mode).
func (fs *FuzzyState) SelectedContentItem() (ContentItem, bool) {
	if len(fs.Filtered) == 0 || fs.Cursor >= len(fs.Filtered) {
		return ContentItem{}, false
	}
	idx := fs.Filtered[fs.Cursor]
	if idx >= len(fs.ContentItems) {
		return ContentItem{}, false
	}
	return fs.ContentItems[idx], true
}

// BuildFileList returns a list of file paths from diff files.
func BuildFileList(files []diff.DiffFile) []string {
	result := make([]string, len(files))
	for i, f := range files {
		name := f.NewName
		if name == "" {
			name = f.OldName
		}
		result[i] = name
	}
	return result
}

// BuildContentList returns content items for all diff lines across all files.
func BuildContentList(files []diff.DiffFile) []ContentItem {
	var items []ContentItem
	for fi, f := range files {
		name := f.NewName
		if name == "" {
			name = f.OldName
		}
		for _, h := range f.Hunks {
			for _, line := range h.Lines {
				var marker string
				var lineNum int
				switch line.Type {
				case diff.LineAdd:
					marker = "[+]"
					lineNum = line.NewNum
				case diff.LineDelete:
					marker = "[-]"
					lineNum = line.OldNum
				default:
					marker = "[ ]"
					lineNum = line.NewNum
					if lineNum == 0 {
						lineNum = line.OldNum
					}
				}
				display := fmt.Sprintf("%s:%d %s: %s", name, lineNum, marker, line.Content)
				items = append(items, ContentItem{
					Display:   display,
					FileIndex: fi,
					LineNum:   lineNum,
					LineType:  line.Type,
				})
			}
		}
	}
	return items
}

// FuzzyFilterIndices returns indices of items matching query (case-insensitive substring).
func FuzzyFilterIndices(items []string, query string) []int {
	if query == "" {
		result := make([]int, len(items))
		for i := range items {
			result[i] = i
		}
		return result
	}
	lowerQuery := strings.ToLower(query)
	var result []int
	for i, item := range items {
		if strings.Contains(strings.ToLower(item), lowerQuery) {
			result = append(result, i)
		}
	}
	return result
}

// ParseContentSelection parses a content search selection line.
// Format: "file.go:42 [+]: content"
// Returns file path, line number, and success flag.
func ParseContentSelection(selection string) (string, int, bool) {
	// Find the first space after "file:line"
	spaceIdx := strings.Index(selection, " ")
	if spaceIdx < 0 {
		return "", 0, false
	}
	fileLine := selection[:spaceIdx]
	colonIdx := strings.LastIndex(fileLine, ":")
	if colonIdx < 0 {
		return "", 0, false
	}
	file := fileLine[:colonIdx]
	lineStr := fileLine[colonIdx+1:]
	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return "", 0, false
	}
	return file, line, true
}

// FindFileIndex finds the index of a file by path in the file list.
func FindFileIndex(files []diff.DiffFile, path string) int {
	for i, f := range files {
		if f.NewName == path || f.OldName == path {
			return i
		}
	}
	return -1
}

// FindRowForLine finds the row index in paired lines for a given line number and side.
func FindRowForLine(pairs []diff.PairedLine, lineNum int, side Side) int {
	for i, p := range pairs {
		if p.IsSeparator || p.IsComment {
			continue
		}
		if side == SideNew && p.Right != nil && p.Right.NewNum == lineNum {
			return i
		}
		if side == SideOld && p.Left != nil && p.Left.OldNum == lineNum {
			return i
		}
	}
	return 0
}

// RenderFuzzyOverlay renders the built-in fuzzy search overlay.
func RenderFuzzyOverlay(fs *FuzzyState, width, height int, cursorColor string) string {
	maxItems := height - 4
	if maxItems < 1 {
		maxItems = 1
	}

	// Input prompt
	prompt := fs.Input.View()

	// Build filtered list
	var listLines []string
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color(cursorColor))
	normalStyle := lipgloss.NewStyle()

	end := fs.YOffset + maxItems
	if end > len(fs.Filtered) {
		end = len(fs.Filtered)
	}

	for i := fs.YOffset; i < end; i++ {
		item := fs.Items[fs.Filtered[i]]
		contentWidth := width - 4
		if contentWidth < 1 {
			contentWidth = 1
		}
		displayWidth := ansi.StringWidth(item)
		if displayWidth > contentWidth {
			item = ansi.Truncate(item, contentWidth, "")
			displayWidth = ansi.StringWidth(item)
		}
		if displayWidth < contentWidth {
			item = item + strings.Repeat(" ", contentWidth-displayWidth)
		}

		if i == fs.Cursor {
			listLines = append(listLines, cursorStyle.Render("  "+item))
		} else {
			listLines = append(listLines, normalStyle.Render("  "+item))
		}
	}

	if len(fs.Filtered) == 0 {
		listLines = append(listLines, normalStyle.Render("  no matches"))
	}

	// Count line
	countLine := fmt.Sprintf("  %d/%d", len(fs.Filtered), len(fs.Items))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorBarDim))

	parts := []string{prompt, countStyle.Render(countLine)}
	parts = append(parts, listLines...)

	return strings.Join(parts, "\n")
}
