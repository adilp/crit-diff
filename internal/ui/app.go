package ui

import (
	"fmt"
	"strings"

	"github.com/adil/cr/internal/config"
	"github.com/adil/cr/internal/diff"
	"github.com/adil/cr/internal/keys"
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
	allPaired      [][]diff.PairedLine // paired lines for each file
	width          int
	height         int
	yOffset        int
	wordDiff       bool
	pendingKey     string
	config         config.Config
	renderer       *render.Renderer
	oldHighlighted []render.HighlightedLine // highlighted lines for old side of active file
	newHighlighted []render.HighlightedLine // highlighted lines for new side of active file
}

// NewModel creates a new TUI model with the given diff data and terminal dimensions.
func NewModel(files []diff.DiffFile, paired []diff.PairedLine, width, height int) Model {
	cfg, _ := config.Load("")
	return Model{
		mode:       InputModeNormal,
		files:      files,
		activeSide: SideNew,
		paired:     paired,
		width:      width,
		height:     height,
		config:     cfg,
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
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// Convert tea.KeyMsg to a string for the keys package
	keyStr := teaKeyToString(msg)

	// If we have a pending key, resolve the two-key sequence
	if m.pendingKey != "" {
		action := keys.Resolve(m.config, keyStr, m.pendingKey)
		m.pendingKey = ""
		return m.handleAction(action)
	}

	// Check if this key is a prefix (starts a sequence)
	if keys.IsPrefix(m.config, keyStr) {
		m.pendingKey = keyStr
		return m, nil
	}

	// Single-key action
	action := keys.Resolve(m.config, keyStr, "")
	return m.handleAction(action)
}

// teaKeyToString converts a Bubble Tea key message to a string for keys.Resolve.
func teaKeyToString(msg tea.KeyMsg) string {
	switch msg.Type {
	case tea.KeyCtrlD:
		return "Ctrl-d"
	case tea.KeyCtrlU:
		return "Ctrl-u"
	case tea.KeyTab:
		return "Tab"
	case tea.KeyEsc:
		return "Esc"
	case tea.KeySpace:
		return "Space"
	case tea.KeyRunes:
		return string(msg.Runes)
	default:
		return msg.String()
	}
}

func (m Model) handleAction(action keys.Action) (tea.Model, tea.Cmd) {
	switch action {
	case keys.ActionQuit:
		return m, tea.Quit
	case keys.ActionScrollDown:
		if m.cursorRow < len(m.paired)-1 {
			m.cursorRow++
		}
		m.scrollToCursor()
	case keys.ActionScrollUp:
		if m.cursorRow > 0 {
			m.cursorRow--
		}
		m.scrollToCursor()
	case keys.ActionPaneLeft:
		m.activeSide = SideOld
	case keys.ActionPaneRight:
		m.activeSide = SideNew
	case keys.ActionHalfPageDown:
		m.cursorRow += m.visibleRows() / 2
		m.clampCursor()
		m.scrollToCursor()
	case keys.ActionHalfPageUp:
		m.cursorRow -= m.visibleRows() / 2
		m.clampCursor()
		m.scrollToCursor()
	case keys.ActionToggleWordDiff:
		m.wordDiff = !m.wordDiff
	case keys.ActionTop:
		m.cursorRow = 0
		m.scrollToCursor()
	case keys.ActionBottom:
		m.cursorRow = len(m.paired) - 1
		if m.cursorRow < 0 {
			m.cursorRow = 0
		}
		m.scrollToCursor()
	case keys.ActionNextHunk:
		m.jumpToNextHunk()
	case keys.ActionPrevHunk:
		m.jumpToPrevHunk()
	case keys.ActionNextFile:
		m.switchToNextFile()
	case keys.ActionPrevFile:
		m.switchToPrevFile()
	// Recognized but no-op until later tickets
	case keys.ActionNextComment, keys.ActionPrevComment:
	case keys.ActionExpandBelow, keys.ActionExpandAbove:
	case keys.ActionCommentAdd, keys.ActionCommentEdit, keys.ActionCommentDelete:
	case keys.ActionVisualSelect:
	case keys.ActionToggleTree:
	case keys.ActionFuzzyFiles, keys.ActionSearchDiffs:
	case keys.ActionSearch, keys.ActionHelp:
	case keys.ActionToggleWrap:
	case keys.ActionDiscard, keys.ActionNone:
	}
	return m, nil
}

// jumpToNextHunk finds the next separator row after cursor and positions on the line after it.
func (m *Model) jumpToNextHunk() {
	for i := m.cursorRow + 1; i < len(m.paired); i++ {
		if m.paired[i].IsSeparator && i+1 < len(m.paired) {
			m.cursorRow = i + 1
			m.scrollToCursor()
			return
		}
	}
}

// jumpToPrevHunk finds the previous separator row before cursor and positions on the first line of that hunk.
func (m *Model) jumpToPrevHunk() {
	// Find the separator before the current position
	for i := m.cursorRow - 1; i >= 0; i-- {
		if m.paired[i].IsSeparator {
			// Find the start of the hunk before this separator
			// The hunk starts at the beginning or after a previous separator
			start := 0
			for j := i - 1; j >= 0; j-- {
				if m.paired[j].IsSeparator {
					start = j + 1
					break
				}
			}
			m.cursorRow = start
			m.scrollToCursor()
			return
		}
	}
}

// switchToNextFile switches to the next file in the file list.
func (m *Model) switchToNextFile() {
	if m.activeFile >= len(m.files)-1 {
		return
	}
	m.activeFile++
	m.cursorRow = 0
	m.yOffset = 0
	if m.allPaired != nil && m.activeFile < len(m.allPaired) {
		m.paired = m.allPaired[m.activeFile]
	} else {
		m.paired = diff.BuildPairedLines(m.files[m.activeFile].Hunks)
	}
}

// switchToPrevFile switches to the previous file in the file list.
func (m *Model) switchToPrevFile() {
	if m.activeFile <= 0 {
		return
	}
	m.activeFile--
	m.cursorRow = 0
	m.yOffset = 0
	if m.allPaired != nil && m.activeFile < len(m.allPaired) {
		m.paired = m.allPaired[m.activeFile]
	} else {
		m.paired = diff.BuildPairedLines(m.files[m.activeFile].Hunks)
	}
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
