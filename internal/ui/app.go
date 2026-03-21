package ui

import (
	"fmt"
	"strings"

	"github.com/adil/cr/internal/comment"
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
	InputModeNormal  InputMode = "normal"
	InputModeTree    InputMode = "tree"
	InputModeComment InputMode = "comment"
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
	colorBarBg          = "236" // dark gray background for status/help bars
	colorBarFg          = "252" // light foreground for bar text
	colorBarDim         = "245" // dimmer foreground for help bar action labels
	colorTreeCursor     = "237" // background for tree cursor (matches colorCursorActive)
	colorTreeDir        = "245" // dimmer foreground for directories (matches colorBarDim)
	colorTreeFile       = "252" // brighter foreground for files (matches colorBarFg)
	colorTreeActive     = "2"   // green for active file indicator (matches colorAdd)
	colorTreeDim        = "240" // dim text for tree indicators
	colorCommentBg      = "235" // dimmed background for comment display rows
	colorCommentFg      = "245" // muted foreground for comment body text
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
	ref            string // CLI ref arg; empty means working tree
	tree           TreeState
	config         config.Config
	renderer       *render.Renderer
	oldHighlighted []render.HighlightedLine // highlighted lines for old side of active file
	newHighlighted []render.HighlightedLine // highlighted lines for new side of active file
	store          *comment.Store           // comment store for .crit/ persistence
	overlay        CommentOverlay           // comment input overlay modal
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
		tree:       NewTreeState(files),
		config:     cfg,
		renderer:   render.NewRenderer(),
	}
}

// SetRef sets the ref display string for the status bar.
// Pass the CLI ref arg (e.g., "main..HEAD"); empty string means working tree.
func (m *Model) SetRef(ref string) {
	m.ref = ref
}

// SetStore sets the comment store for .crit/ persistence.
func (m *Model) SetStore(s *comment.Store) {
	m.store = s
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

	// Comment input mode has its own key handling
	if m.mode == InputModeComment {
		return m.handleCommentKey(msg)
	}

	// Tree mode has its own key handling
	if m.mode == InputModeTree {
		return m.handleTreeKey(msg)
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
	case keys.ActionNextComment:
		m.jumpToNextComment()
	case keys.ActionPrevComment:
		m.jumpToPrevComment()
	case keys.ActionCommentEdit:
		m.editComment()
	case keys.ActionCommentDelete:
		m.deleteComment()
	// Recognized but no-op until later tickets
	case keys.ActionExpandBelow, keys.ActionExpandAbove:
	case keys.ActionVisualSelect:
	case keys.ActionCommentAdd:
		m.openCommentOverlay()
	case keys.ActionToggleTree:
		m.tree.Open = true
		m.mode = InputModeTree
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

	// Calculate pane widths based on whether tree is open
	treeWidth := 0
	dividers := 1 // center divider between panes
	if m.tree.Open {
		treeWidth = TreeWidth(m.width)
		dividers = 2 // tree divider + center divider
	}
	paneWidth := (m.width - treeWidth - dividers) / 2
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

		if p.IsComment {
			rows = append(rows, m.renderCommentRow(p, paneWidth, isCursor))
			continue
		}

		// Prepare highlighted content for each side
		leftHL, rightHL := m.highlightPair(p)

		leftStr := m.renderPane(p.Left, SideOld, paneWidth, lineNumWidth, isCursor, m.activeSide == SideOld, leftHL)
		rightStr := m.renderPane(p.Right, SideNew, paneWidth, lineNumWidth, isCursor, m.activeSide == SideNew, rightHL)
		row := lipgloss.JoinHorizontal(lipgloss.Top, leftStr, "│", rightStr)
		rows = append(rows, row)
	}

	diffContent := strings.Join(rows, "\n")

	// If tree is open, render tree panel alongside diff
	if m.tree.Open {
		treeOutput := RenderTree(&m.tree, treeWidth, vis, m.activeFile, m.treeCommentCounts())
		diffContent = lipgloss.JoinHorizontal(lipgloss.Top, treeOutput, "│", diffContent)
	}

	// Count insertions/deletions for the active file
	adds, dels := m.countFileChanges()

	// Status bar (top) and help bar (bottom)
	filePath := ""
	if len(m.files) > 0 && m.activeFile < len(m.files) {
		filePath = m.files[m.activeFile].NewName
		if filePath == "" {
			filePath = m.files[m.activeFile].OldName
		}
	}
	statusBar := RenderStatusBar(m.width, m.ref, m.activeFile, len(m.files), filePath, adds, dels, m.commentCount(), m.activeSide)
	helpBar := RenderHelpBar(m.width, m.mode)

	result := statusBar + "\n" + diffContent + "\n" + helpBar

	// Overlay the comment input modal when active
	if m.overlay.Active {
		overlayBox := m.overlay.Render(m.width)
		flipAbove := m.overlay.ShouldFlipAbove(m.yOffset, vis)
		// Position overlay vertically within the diff content area
		overlayRow := m.overlay.RowIndex - m.yOffset + 1 // +1 for status bar
		if flipAbove {
			overlayRow -= 4 // place above cursor (overlay is ~3 rows + gap)
		} else {
			overlayRow += 1 // place below cursor
		}
		if overlayRow < 1 {
			overlayRow = 1
		}
		if overlayRow > vis {
			overlayRow = vis - 3
		}
		result = lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top,
			result,
			lipgloss.WithWhitespaceChars(" "),
		)
		// Insert the overlay at the computed position using lipgloss overlay
		resultLines := strings.Split(result, "\n")
		overlayLines := strings.Split(overlayBox, "\n")
		xOffset := (m.width - ansi.StringWidth(overlayLines[0])) / 2
		for i, ol := range overlayLines {
			row := overlayRow + i
			if row >= 0 && row < len(resultLines) {
				resultLines[row] = placeOverlayLine(resultLines[row], ol, xOffset, m.width)
			}
		}
		result = strings.Join(resultLines, "\n")
	}

	return result
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

// renderCommentRow renders a comment display row spanning both panes.
func (m Model) renderCommentRow(p diff.PairedLine, paneWidth int, isCursor bool) string {
	totalWidth := paneWidth*2 + 1
	text := "💬 " + p.CommentBody

	// Clip or pad to total width
	displayWidth := ansi.StringWidth(text)
	if displayWidth > totalWidth {
		text = ansi.Truncate(text, totalWidth, "")
		displayWidth = ansi.StringWidth(text)
	}
	if displayWidth < totalWidth {
		text = text + strings.Repeat(" ", totalWidth-displayWidth)
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorCommentFg)).
		Background(lipgloss.Color(colorCommentBg))

	if isCursor {
		style = style.Background(lipgloss.Color(colorCursorActive))
	}

	return style.Render(text)
}

// renderSeparator renders a hunk separator row.
func (m Model) renderSeparator(paneWidth int) string {
	totalWidth := paneWidth*2 + 1
	sep := strings.Repeat("─", totalWidth)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSeparator))
	return style.Render(sep)
}

// handleTreeKey handles key input when in tree mode.
func (m Model) handleTreeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyEsc:
		m.tree.Open = false
		m.mode = InputModeNormal
		return m, nil
	case tea.KeyEnter:
		return m.treeOpenSelected()
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			m.tree.CursorDown()
		case "k":
			m.tree.CursorUp()
		case "l":
			return m.treeOpenSelected()
		case "h":
			if entry, ok := m.tree.SelectedEntry(); ok && entry.IsDir {
				m.tree.ToggleCollapse(entry.FullPath)
			}
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// treeOpenSelected opens the file under the tree cursor and closes the tree.
func (m Model) treeOpenSelected() (tea.Model, tea.Cmd) {
	idx := m.tree.SelectedFileIndex()
	if idx < 0 || idx >= len(m.files) {
		// On a directory — expand/collapse instead
		if entry, ok := m.tree.SelectedEntry(); ok && entry.IsDir {
			m.tree.ToggleCollapse(entry.FullPath)
		}
		return m, nil
	}

	m.activeFile = idx
	m.cursorRow = 0
	m.yOffset = 0
	if m.allPaired != nil && idx < len(m.allPaired) {
		m.paired = m.allPaired[idx]
	} else {
		m.paired = diff.BuildPairedLines(m.files[idx].Hunks)
	}
	m.tree.Open = false
	m.mode = InputModeNormal
	return m, nil
}

// openCommentOverlay opens the comment input overlay on the current cursor line.
// Guards: no-op on separator, blank padding, comment rows, or lines with existing comments.
func (m *Model) openCommentOverlay() {
	// Guard: need a store to save comments
	if m.store == nil {
		return
	}

	if len(m.paired) == 0 || m.cursorRow >= len(m.paired) {
		return
	}
	p := m.paired[m.cursorRow]

	// Guard: separator row
	if p.IsSeparator {
		return
	}

	// Guard: comment display row
	if p.IsComment {
		return
	}

	// Guard: get the line on the active side
	line := m.activeSideLine(p)
	if line == nil {
		return
	}

	// Get the line number for the active side
	lineNum := m.activeSideLineNum(line)

	// Guard: one comment per line
	filePath := m.activeFilePath()
	if m.store.HasComment(filePath, lineNum) {
		return
	}

	m.overlay = NewCommentOverlay(lineNum, m.activeSide, m.cursorRow)
	m.mode = InputModeComment
}

// handleCommentKey handles key input when in comment input mode.
func (m Model) handleCommentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.overlay = CommentOverlay{}
		m.mode = InputModeNormal
		return m, nil
	case tea.KeyEnter:
		body := strings.TrimSpace(m.overlay.Value())
		if body != "" && m.store != nil {
			if m.overlay.editingID != "" {
				// Edit existing comment
				_ = m.store.EditComment(m.overlay.editingID, body)
			} else {
				// Add new comment
				filePath := m.activeFilePath()
				p := m.paired[m.overlay.RowIndex]
				line := m.activeSideLine(p)
				snippet := ""
				if line != nil {
					snippet = strings.TrimSpace(line.Content)
				}
				_ = m.store.AddComment(filePath, m.overlay.Line, 0, snippet, body)
			}
			m.rebuildPairedLines()
		}
		m.overlay = CommentOverlay{}
		m.mode = InputModeNormal
		return m, nil
	default:
		// Forward all other keys to the text input
		var cmd tea.Cmd
		m.overlay.Input, cmd = m.overlay.Input.Update(msg)
		return m, cmd
	}
}

// activeSideLine returns the DiffLine on the active side of a paired line.
func (m Model) activeSideLine(p diff.PairedLine) *diff.DiffLine {
	if m.activeSide == SideNew {
		return p.Right
	}
	return p.Left
}

// activeSideLineNum returns the line number for the active side.
func (m Model) activeSideLineNum(line *diff.DiffLine) int {
	if m.activeSide == SideNew {
		return line.NewNum
	}
	return line.OldNum
}

// activeFilePath returns the file path for the active file.
func (m Model) activeFilePath() string {
	if len(m.files) == 0 || m.activeFile >= len(m.files) {
		return ""
	}
	f := m.files[m.activeFile]
	if f.NewName != "" {
		return f.NewName
	}
	return f.OldName
}

// commentCount returns the number of comments for the active file.
func (m Model) commentCount() int {
	if m.store == nil {
		return 0
	}
	return len(m.store.Comments(m.activeFilePath()))
}

// placeOverlayLine replaces a horizontal span in a background line with an overlay line.
// Uses display-width-aware positioning to handle ANSI-styled content.
func placeOverlayLine(bgLine, overlayLine string, xOffset, totalWidth int) string {
	bgWidth := ansi.StringWidth(bgLine)

	// Pad background to totalWidth if needed
	if bgWidth < totalWidth {
		bgLine = bgLine + strings.Repeat(" ", totalWidth-bgWidth)
	}

	// Build: left portion + overlay + right portion
	left := ansi.Truncate(bgLine, xOffset, "")
	leftWidth := ansi.StringWidth(left)

	// Pad left to exact offset if truncation was short
	if leftWidth < xOffset {
		left = left + strings.Repeat(" ", xOffset-leftWidth)
	}

	overlayWidth := ansi.StringWidth(overlayLine)
	rightStart := xOffset + overlayWidth

	// Extract right portion from background
	right := ""
	if rightStart < totalWidth {
		// Cut the background after the overlay ends
		cut := ansi.Truncate(bgLine, rightStart, "")
		cutWidth := ansi.StringWidth(cut)
		if cutWidth < rightStart {
			right = strings.Repeat(" ", totalWidth-rightStart)
		} else {
			// Get the remaining portion
			fullBg := bgLine
			right = ""
			if rightStart < bgWidth {
				// We need the part of bgLine starting at display position rightStart
				right = strings.Repeat(" ", totalWidth-rightStart)
			}
			_ = fullBg
		}
	}

	return left + overlayLine + right
}

// treeCommentCounts returns a map of file path → comment count for tree rendering.
func (m Model) treeCommentCounts() map[string]int {
	if m.store == nil {
		return nil
	}
	counts := make(map[string]int)
	for _, f := range m.files {
		path := f.NewName
		if path == "" {
			path = f.OldName
		}
		n := len(m.store.Comments(path))
		if n > 0 {
			counts[path] = n
		}
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

// rebuildPairedLines rebuilds the paired lines for the active file,
// inserting comment display rows from the store.
func (m *Model) rebuildPairedLines() {
	if len(m.files) == 0 || m.activeFile >= len(m.files) {
		return
	}
	base := diff.BuildPairedLines(m.files[m.activeFile].Hunks)

	if m.store != nil {
		filePath := m.activeFilePath()
		comments := m.store.Comments(filePath)
		if len(comments) > 0 {
			// Build the line→comment map for the active side
			cm := make(map[int]diff.CommentInfo, len(comments))
			for _, c := range comments {
				cm[c.Line] = diff.CommentInfo{ID: c.ID, Body: c.Body, Line: c.Line}
			}
			side := diff.SideNew
			if m.activeSide == SideOld {
				side = diff.SideOld
			}
			base = diff.InsertCommentRows(base, cm, side)
		}
	}

	m.paired = base
	// Also update the allPaired cache for this file
	if m.allPaired != nil && m.activeFile < len(m.allPaired) {
		m.allPaired[m.activeFile] = base
	}
}

// jumpToNextComment scans forward from cursor for the next comment row.
func (m *Model) jumpToNextComment() {
	for i := m.cursorRow + 1; i < len(m.paired); i++ {
		if m.paired[i].IsComment {
			m.cursorRow = i
			m.scrollToCursor()
			return
		}
	}
}

// jumpToPrevComment scans backward from cursor for the previous comment row.
func (m *Model) jumpToPrevComment() {
	for i := m.cursorRow - 1; i >= 0; i-- {
		if m.paired[i].IsComment {
			m.cursorRow = i
			m.scrollToCursor()
			return
		}
	}
}

// editComment opens the comment overlay pre-filled with the existing comment body.
func (m *Model) editComment() {
	if len(m.paired) == 0 || m.cursorRow >= len(m.paired) {
		return
	}
	p := m.paired[m.cursorRow]
	if !p.IsComment {
		return
	}

	// Look up the comment's line number from the store
	commentLine := 0
	if m.store != nil {
		for _, c := range m.store.Comments(m.activeFilePath()) {
			if c.ID == p.CommentID {
				commentLine = c.Line
				break
			}
		}
	}

	m.overlay = NewCommentOverlay(commentLine, m.activeSide, m.cursorRow)
	m.overlay.Input.SetValue(p.CommentBody)
	m.overlay.editingID = p.CommentID
	m.mode = InputModeComment
}

// deleteComment removes the comment under the cursor and rebuilds paired lines.
func (m *Model) deleteComment() {
	if len(m.paired) == 0 || m.cursorRow >= len(m.paired) {
		return
	}
	p := m.paired[m.cursorRow]
	if !p.IsComment {
		return
	}
	if m.store == nil {
		return
	}

	_ = m.store.DeleteComment(p.CommentID)

	// Move cursor up to the code line above
	if m.cursorRow > 0 {
		m.cursorRow--
	}

	m.rebuildPairedLines()
	m.clampCursor()
	m.scrollToCursor()
}

// countFileChanges counts insertions and deletions in the active file's hunks.
func (m Model) countFileChanges() (int, int) {
	if len(m.files) == 0 || m.activeFile >= len(m.files) {
		return 0, 0
	}
	var adds, dels int
	for _, h := range m.files[m.activeFile].Hunks {
		for _, l := range h.Lines {
			switch l.Type {
			case diff.LineAdd:
				adds++
			case diff.LineDelete:
				dels++
			}
		}
	}
	return adds, dels
}
