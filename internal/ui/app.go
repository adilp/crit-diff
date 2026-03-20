package ui

import (
	"fmt"
	"strings"

	"github.com/adil/cr/internal/diff"
	"github.com/adil/cr/internal/render"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Side represents which pane is active.
type Side string

const (
	SideOld Side = "old"
	SideNew Side = "new"
)

// InputMode represents the current TUI input mode.
type InputMode string

const (
	InputModeNormal InputMode = "normal"
)

// Style constants for terminal colors.
const (
	colorAdd            = "2"   // green foreground for added lines
	colorDelete         = "1"   // red foreground for deleted lines
	colorCursorActive   = "237" // brighter background for cursor on active pane
	colorCursorInactive = "235" // dimmer background for cursor on inactive pane
	colorPaddingBg      = "233" // subtle background for blank padding rows
	colorPaddingFg      = "239" // foreground for padding rows
	colorSeparator      = "240" // foreground for hunk separator lines
	colorEmphasisAdd    = "22"  // dark green background for word-level add emphasis
	colorEmphasisDelete = "52"  // dark red background for word-level delete emphasis
)

// Model is the Bubble Tea model for the cr TUI.
type Model struct {
	mode           InputMode
	files          []diff.DiffFile
	activeFile     int
	activeSide     Side
	cursorRow      int
	paired         []diff.PairedLine
	width          int
	height         int
	yOffset        int
	wordDiff       bool
	renderer       *render.Renderer
	oldHighlighted []render.HighlightedLine // highlighted lines for old side of active file
	newHighlighted []render.HighlightedLine // highlighted lines for new side of active file
}

// NewModel creates a new TUI model with the given diff data and terminal dimensions.
func NewModel(files []diff.DiffFile, paired []diff.PairedLine, width, height int) Model {
	return Model{
		mode:       InputModeNormal,
		files:      files,
		activeSide: SideNew,
		paired:     paired,
		width:      width,
		height:     height,
		renderer:   render.NewRenderer(),
	}
}

// SetHighlighting sets the highlighted lines for the current active file.
// oldContent and newContent are the full file contents for old and new sides.
// The filename is used for language detection.
func (m *Model) SetHighlighting(filename, oldContent, newContent string) {
	m.oldHighlighted = m.renderer.HighlightFileWithKey("old:"+filename, filename, oldContent)
	m.newHighlighted = m.renderer.HighlightFileWithKey("new:"+filename, filename, newContent)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// visibleRows returns the number of content rows visible in the viewport.
// Reserves 2 rows: 1 for status bar, 1 for help bar.
func (m Model) visibleRows() int {
	v := m.height - 2
	if v < 1 {
		return 1
	}
	return v
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampCursor()
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlD:
		m.cursorRow += m.visibleRows() / 2
		m.clampCursor()
		m.scrollToCursor()
		return m, nil
	case tea.KeyCtrlU:
		m.cursorRow -= m.visibleRows() / 2
		m.clampCursor()
		m.scrollToCursor()
		return m, nil
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q":
			return m, tea.Quit
		case "j":
			if m.cursorRow < len(m.paired)-1 {
				m.cursorRow++
			}
			m.scrollToCursor()
			return m, nil
		case "k":
			if m.cursorRow > 0 {
				m.cursorRow--
			}
			m.scrollToCursor()
			return m, nil
		case "h":
			m.activeSide = SideOld
			return m, nil
		case "l":
			m.activeSide = SideNew
			return m, nil
		case "w":
			m.wordDiff = !m.wordDiff
			return m, nil
		}
	}
	return m, nil
}

// clampCursor ensures cursor stays within valid range.
func (m *Model) clampCursor() {
	maxRow := len(m.paired) - 1
	if maxRow < 0 {
		maxRow = 0
	}
	if m.cursorRow < 0 {
		m.cursorRow = 0
	}
	if m.cursorRow > maxRow {
		m.cursorRow = maxRow
	}
}

// scrollToCursor adjusts yOffset to keep cursor visible.
func (m *Model) scrollToCursor() {
	if m.cursorRow < m.yOffset {
		m.yOffset = m.cursorRow
	}
	vis := m.visibleRows()
	if m.cursorRow >= m.yOffset+vis {
		m.yOffset = m.cursorRow - vis + 1
	}
}

// View implements tea.Model.
func (m Model) View() string {
	if len(m.paired) == 0 {
		return "No diff content to display.\n"
	}

	paneWidth := (m.width - 1) / 2
	lineNumWidth := 5 // 4 digits + separator

	var rows []string
	vis := m.visibleRows()
	end := m.yOffset + vis
	if end > len(m.paired) {
		end = len(m.paired)
	}

	for i := m.yOffset; i < end; i++ {
		p := m.paired[i]
		isCursor := i == m.cursorRow

		if p.IsSeparator {
			rows = append(rows, m.renderSeparator(paneWidth))
			continue
		}

		// Prepare highlighted content for each side
		leftHL, rightHL := m.highlightPair(p)

		leftStr := m.renderPane(p.Left, SideOld, paneWidth, lineNumWidth, isCursor, m.activeSide == SideOld, leftHL)
		rightStr := m.renderPane(p.Right, SideNew, paneWidth, lineNumWidth, isCursor, m.activeSide == SideNew, rightHL)
		row := lipgloss.JoinHorizontal(lipgloss.Top, leftStr, "│", rightStr)
		rows = append(rows, row)
	}

	// Reserve status bar line (placeholder — actual bar in CR-009)
	statusBar := m.renderStatusPlaceholder()
	// Reserve help bar line (placeholder — actual bar in CR-009)
	helpBar := ""

	return strings.Join(rows, "\n") + "\n" + statusBar + "\n" + helpBar
}

// highlightPair returns rendered highlighted content for both sides of a paired line.
// Returns empty strings if highlighting is not available.
func (m Model) highlightPair(p diff.PairedLine) (string, string) {
	var leftHL, rightHL string

	// Look up highlighted lines by line number (1-based → 0-indexed)
	if p.Left != nil && len(m.oldHighlighted) > 0 {
		idx := p.Left.OldNum - 1
		if idx >= 0 && idx < len(m.oldHighlighted) {
			hl := m.oldHighlighted[idx]
			// Apply word-level emphasis if enabled and this is a delete paired with an add
			if m.wordDiff && p.Left.Type == diff.LineDelete && p.Right != nil && p.Right.Type == diff.LineAdd {
				oldMask, _ := render.ComputeWordDiff(p.Left.Content, p.Right.Content)
				hl = render.ApplyEmphasis(hl, oldMask, colorEmphasisDelete)
			}
			leftHL = render.RenderLine(hl)
		}
	}

	if p.Right != nil && len(m.newHighlighted) > 0 {
		idx := p.Right.NewNum - 1
		if idx >= 0 && idx < len(m.newHighlighted) {
			hl := m.newHighlighted[idx]
			// Apply word-level emphasis if enabled and this is an add paired with a delete
			if m.wordDiff && p.Right.Type == diff.LineAdd && p.Left != nil && p.Left.Type == diff.LineDelete {
				_, newMask := render.ComputeWordDiff(p.Left.Content, p.Right.Content)
				hl = render.ApplyEmphasis(hl, newMask, colorEmphasisAdd)
			}
			rightHL = render.RenderLine(hl)
		}
	}

	return leftHL, rightHL
}

// renderPane renders one side of a paired line.
// highlightedContent, if non-empty, is used instead of raw line content (contains ANSI styling).
func (m Model) renderPane(line *diff.DiffLine, side Side, paneWidth, lineNumWidth int, isCursor, isActiveSide bool, highlightedContent string) string {
	if line == nil {
		return m.renderBlankPadding(paneWidth, isCursor, isActiveSide)
	}

	// Line number: left pane shows OldNum, right pane shows NewNum for context lines
	var numStr string
	switch line.Type {
	case diff.LineDelete:
		numStr = fmt.Sprintf("%4d", line.OldNum)
	case diff.LineAdd:
		numStr = fmt.Sprintf("%4d", line.NewNum)
	default:
		if side == SideOld {
			numStr = fmt.Sprintf("%4d", line.OldNum)
		} else {
			numStr = fmt.Sprintf("%4d", line.NewNum)
		}
	}

	contentWidth := paneWidth - lineNumWidth
	if contentWidth < 0 {
		contentWidth = 0
	}

	// Use highlighted content if available, otherwise use raw content
	content := highlightedContent
	useHighlighted := content != ""
	if !useHighlighted {
		content = line.Content
	}

	// Clip content at pane width using display-width-aware truncation
	displayWidth := ansi.StringWidth(content)
	if displayWidth > contentWidth {
		content = ansi.Truncate(content, contentWidth, "")
		displayWidth = ansi.StringWidth(content)
	}

	// Pad content to fill pane width
	if displayWidth < contentWidth {
		content = content + strings.Repeat(" ", contentWidth-displayWidth)
	}

	lineStr := numStr + "│" + content

	// Apply styling — skip foreground color when using highlighted content
	// (syntax colors are already embedded in the ANSI content)
	style := lipgloss.NewStyle()
	if !useHighlighted {
		switch line.Type {
		case diff.LineAdd:
			style = style.Foreground(lipgloss.Color(colorAdd))
		case diff.LineDelete:
			style = style.Foreground(lipgloss.Color(colorDelete))
		}
	}

	if isCursor {
		if isActiveSide {
			style = style.Background(lipgloss.Color(colorCursorActive))
		} else {
			style = style.Background(lipgloss.Color(colorCursorInactive))
		}
	}

	return style.Render(lineStr)
}

// renderBlankPadding renders an empty padding row for nil sides.
func (m Model) renderBlankPadding(paneWidth int, isCursor, isActiveSide bool) string {
	blank := strings.Repeat(" ", paneWidth)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPaddingFg))
	if isCursor {
		if isActiveSide {
			style = style.Background(lipgloss.Color(colorCursorActive))
		} else {
			style = style.Background(lipgloss.Color(colorCursorInactive))
		}
	} else {
		style = style.Background(lipgloss.Color(colorPaddingBg))
	}
	return style.Render(blank)
}

// renderSeparator renders a hunk separator row.
func (m Model) renderSeparator(paneWidth int) string {
	totalWidth := paneWidth*2 + 1
	sep := strings.Repeat("─", totalWidth)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSeparator))
	return style.Render(sep)
}

// renderStatusPlaceholder renders a placeholder status bar.
func (m Model) renderStatusPlaceholder() string {
	var side string
	if m.activeSide == SideOld {
		side = "old"
	} else {
		side = "new"
	}
	fileName := ""
	if len(m.files) > 0 && m.activeFile < len(m.files) {
		fileName = m.files[m.activeFile].NewName
	}
	return fmt.Sprintf(" %s  [%s]  row %d/%d", fileName, side, m.cursorRow+1, len(m.paired))
}
