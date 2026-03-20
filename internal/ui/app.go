package ui

import (
	"fmt"
	"strings"

	"github.com/adil/cr/internal/diff"
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
)

// Model is the Bubble Tea model for the cr TUI.
type Model struct {
	mode       InputMode
	files      []diff.DiffFile
	activeFile int
	activeSide Side
	cursorRow  int
	paired     []diff.PairedLine
	width      int
	height     int
	yOffset    int
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
	}
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

		leftStr := m.renderPane(p.Left, SideOld, paneWidth, lineNumWidth, isCursor, m.activeSide == SideOld)
		rightStr := m.renderPane(p.Right, SideNew, paneWidth, lineNumWidth, isCursor, m.activeSide == SideNew)
		row := lipgloss.JoinHorizontal(lipgloss.Top, leftStr, "│", rightStr)
		rows = append(rows, row)
	}

	// Reserve status bar line (placeholder — actual bar in CR-009)
	statusBar := m.renderStatusPlaceholder()
	// Reserve help bar line (placeholder — actual bar in CR-009)
	helpBar := ""

	return strings.Join(rows, "\n") + "\n" + statusBar + "\n" + helpBar
}

// renderPane renders one side of a paired line.
func (m Model) renderPane(line *diff.DiffLine, side Side, paneWidth, lineNumWidth int, isCursor, isActiveSide bool) string {
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

	content := line.Content
	contentWidth := paneWidth - lineNumWidth
	if contentWidth < 0 {
		contentWidth = 0
	}

	// Clip content at pane width using display-width-aware truncation
	displayWidth := ansi.StringWidth(content)
	if displayWidth > contentWidth {
		content = ansi.Truncate(content, contentWidth, "")
		displayWidth = contentWidth
	}

	// Pad content to fill pane width
	if displayWidth < contentWidth {
		content = content + strings.Repeat(" ", contentWidth-displayWidth)
	}

	lineStr := numStr + "│" + content

	// Apply styling
	style := lipgloss.NewStyle()
	switch line.Type {
	case diff.LineAdd:
		style = style.Foreground(lipgloss.Color(colorAdd))
	case diff.LineDelete:
		style = style.Foreground(lipgloss.Color(colorDelete))
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
