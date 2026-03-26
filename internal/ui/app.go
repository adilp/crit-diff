package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/adil/cr/internal/comment"
	"github.com/adil/cr/internal/config"
	"github.com/adil/cr/internal/diff"
	"github.com/adil/cr/internal/keys"
	"github.com/adil/cr/internal/render"
	"github.com/charmbracelet/bubbles/textinput"
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
	InputModeVisual  InputMode = "visual"
	InputModeSearch  InputMode = "search"
	InputModeFuzzy   InputMode = "fuzzy"
	InputModeHelp    InputMode = "help"
)

// Style constants for terminal colors.
const (
	colorAdd            = "2"   // green foreground for added lines
	colorDelete         = "1"   // red foreground for deleted lines
	colorPaddingBg  = "233" // subtle background for blank padding rows
	colorPaddingFg  = "239" // foreground for padding rows
	colorSeparator  = "240" // foreground for hunk separator lines
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
	colorSearchMatch    = "226" // bright yellow background for search match highlighting
)

// Model is the Bubble Tea model for the cr TUI.
type Model struct {
	mode            InputMode
	files           []diff.DiffFile
	activeFile      int
	activeSide      Side
	cursorRow       int
	paired          []diff.PairedLine
	allPaired       [][]diff.PairedLine // paired lines for each file
	width           int
	height          int
	yOffset         int
	wordDiff        bool
	pendingKey      string
	ref             string // CLI ref arg; empty means working tree
	tree            TreeState
	config          config.Config
	renderer        *render.Renderer
	oldHighlighted  []render.HighlightedLine // highlighted lines for old side of active file
	newHighlighted  []render.HighlightedLine // highlighted lines for new side of active file
	store           *comment.Store           // comment store for .crit/ persistence
	overlay         CommentOverlay           // comment input overlay modal
	visualStart     int                      // row index where V was pressed (visual mode anchor)
	search          SearchState              // in-file search state
	fuzzy           FuzzyState               // built-in fuzzy overlay state
	separatorStates []SeparatorState         // expansion state per separator
	oldFileLines    []string                 // full old file content (0-indexed line strings)
	newFileLines    []string                 // full new file content (0-indexed line strings)
	wrapMode        bool                     // true = lines wrap within pane, false = clipped
	helpYOffset     int                      // scroll offset for help overlay
	helpEntries     []HelpEntry              // cached help entries (built on mode entry)
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
		wordDiff:   cfg.Display.WordDiff,
		wrapMode:   cfg.Display.Wrap,
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

// SetFileContent stores the full file content for context expansion and initializes
// separator states for the active file's hunks.
func (m *Model) SetFileContent(oldContent, newContent string) {
	if oldContent != "" {
		m.oldFileLines = strings.Split(oldContent, "\n")
	} else {
		m.oldFileLines = nil
	}
	if newContent != "" {
		m.newFileLines = strings.Split(newContent, "\n")
	} else {
		m.newFileLines = nil
	}
	if len(m.files) > 0 && m.activeFile < len(m.files) {
		m.separatorStates = BuildSeparatorStates(m.files[m.activeFile].Hunks)
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
		if m.wrapMode {
			m.recomputeWrap()
		}
		m.clampCursor()
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	// Help mode has its own key handling
	if m.mode == InputModeHelp {
		return m.handleHelpKey(msg)
	}

	// Fuzzy mode has its own key handling
	if m.mode == InputModeFuzzy {
		return m.handleFuzzyKey(msg)
	}

	// Search input mode has its own key handling
	if m.mode == InputModeSearch {
		return m.handleSearchKey(msg)
	}

	// Comment input mode has its own key handling
	if m.mode == InputModeComment {
		return m.handleCommentKey(msg)
	}

	// Visual mode has its own key handling
	if m.mode == InputModeVisual {
		return m.handleVisualKey(msg)
	}

	// Tree mode has its own key handling
	if m.mode == InputModeTree {
		return m.handleTreeKey(msg)
	}

	// Handle search-related keys in normal mode (only when no pending prefix key)
	if m.search.Active && m.pendingKey == "" {
		if msg.Type == tea.KeyEsc {
			m.search = SearchState{}
			return m, nil
		}
		if msg.Type == tea.KeyRunes {
			switch string(msg.Runes) {
			case "n":
				m.searchNext()
				return m, nil
			case "N":
				m.searchPrev()
				return m, nil
			}
		}
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
	case keys.ActionExpandBelow:
		m.expandContext("down")
	case keys.ActionExpandAbove:
		m.expandContext("up")
	case keys.ActionVisualSelect:
		m.enterVisualMode()
	case keys.ActionCommentAdd:
		m.openCommentOverlay()
	case keys.ActionToggleTree:
		m.tree.Open = true
		m.mode = InputModeTree
	case keys.ActionFuzzyFiles:
		m.openFuzzyFiles()
	case keys.ActionSearchDiffs:
		m.openFuzzyContent()
	case keys.ActionSearch:
		m.openSearch()
	case keys.ActionHelp:
		m.mode = InputModeHelp
		m.helpYOffset = 0
		m.helpEntries = BuildHelpEntries(m.config)
	case keys.ActionToggleWrap:
		m.toggleWrap()
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

// expandContext expands context lines around a separator.
// direction is "down" (]e) or "up" ([e).
func (m *Model) expandContext(direction string) {
	if len(m.separatorStates) == 0 || len(m.files) == 0 || m.activeFile >= len(m.files) {
		return
	}

	// Find the nearest separator in paired lines
	var sepRow int
	if direction == "down" {
		sepRow = FindNearestSeparatorBelow(m.paired, m.cursorRow)
	} else {
		sepRow = FindNearestSeparatorAbove(m.paired, m.cursorRow)
	}
	if sepRow < 0 {
		return
	}

	// Find which separator state corresponds to this separator row
	sepP := m.paired[sepRow]
	var state *SeparatorState
	for i := range m.separatorStates {
		if m.separatorStates[i].HunkIndex == sepP.HunkIndex {
			state = &m.separatorStates[i]
			break
		}
	}
	if state == nil || IsSeparatorFullyExpanded(*state) {
		return
	}

	hunks := m.files[m.activeFile].Hunks
	if state.HunkIndex < 1 || state.HunkIndex >= len(hunks) {
		return
	}
	hunkAbove := hunks[state.HunkIndex-1]
	hunkBelow := hunks[state.HunkIndex]

	newLines := ExpandedContextLines(
		state, hunkAbove, hunkBelow,
		m.oldFileLines, m.newFileLines,
		direction,
	)
	if len(newLines) == 0 {
		return
	}

	// Insert the expanded lines into paired lines at the separator position
	var result []diff.PairedLine
	if IsSeparatorFullyExpanded(*state) {
		// Replace separator with expanded lines
		result = append(result, m.paired[:sepRow]...)
		result = append(result, newLines...)
		result = append(result, m.paired[sepRow+1:]...)
	} else if direction == "down" {
		// Insert expanded lines after separator
		result = append(result, m.paired[:sepRow]...)
		result = append(result, newLines...)
		result = append(result, m.paired[sepRow:]...)
	} else {
		// Insert expanded lines before separator
		result = append(result, m.paired[:sepRow]...)
		result = append(result, m.paired[sepRow])
		result = append(result, newLines...)
		result = append(result, m.paired[sepRow+1:]...)
	}

	m.paired = result
	if m.allPaired != nil && m.activeFile < len(m.allPaired) {
		m.allPaired[m.activeFile] = result
	}
	m.clampCursor()
	m.scrollToCursor()
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
	m.resetFileState()
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
	m.resetFileState()
}

// resetFileState resets per-file state when switching to a new file.
// Reinitializes separator states and recomputes search matches.
func (m *Model) resetFileState() {
	if m.activeFile < len(m.files) {
		m.separatorStates = BuildSeparatorStates(m.files[m.activeFile].Hunks)
	}
	// Clear file content lines — they need to be re-fetched for the new file
	m.oldFileLines = nil
	m.newFileLines = nil
	m.recomputeSearch()
}

// recomputeSearch updates search matches for the current paired lines.
// Called after file switches to keep search state consistent.
func (m *Model) recomputeSearch() {
	if !m.search.Active || m.search.Query == "" {
		return
	}
	m.search.Matches = FindMatches(m.paired, m.search.Query)
	m.search.Current = 0
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

// toggleWrap toggles line wrapping mode and recomputes paired lines.
func (m *Model) toggleWrap() {
	m.wrapMode = !m.wrapMode
	m.recomputeWrap()
}

// recomputeWrap rebuilds paired lines for the active file based on wrap mode.
func (m *Model) recomputeWrap() {
	if len(m.files) == 0 || m.activeFile >= len(m.files) {
		return
	}
	if m.wrapMode {
		// Calculate pane content width
		treeWidth := 0
		dividers := 1
		if m.tree.Open {
			treeWidth = TreeWidth(m.width)
			dividers = 2
		}
		paneWidth := (m.width - treeWidth - dividers) / 2
		m.paired = diff.BuildWrappedPairedLines(m.files[m.activeFile].Hunks, paneWidth)
	} else {
		m.paired = diff.BuildPairedLines(m.files[m.activeFile].Hunks)
	}
	if m.allPaired != nil && m.activeFile < len(m.allPaired) {
		m.allPaired[m.activeFile] = m.paired
	}
	m.clampCursor()
	m.scrollToCursor()
}

// View implements tea.Model.
func (m Model) View() string {
	// Check for narrow terminal
	minWidth := m.config.Display.MinWidth
	if minWidth <= 0 {
		minWidth = 100
	}
	if m.width < minWidth {
		msg := fmt.Sprintf("Terminal too narrow — resize to %d+ cols", minWidth)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	// Check for special file types that show centered messages instead of diff
	if len(m.files) > 0 && m.activeFile < len(m.files) {
		f := m.files[m.activeFile]
		if msg := m.specialFileMessage(f); msg != "" {
			return m.renderSpecialFileView(f, msg)
		}
	}

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

	// Compute visual selection range
	visualMin, visualMax := -1, -1
	if m.mode == InputModeVisual {
		visualMin, visualMax = m.visualStart, m.cursorRow
		if visualMin > visualMax {
			visualMin, visualMax = visualMax, visualMin
		}
	}

	for i := m.yOffset; i < end; i++ {
		p := m.paired[i]
		isCursor := i == m.cursorRow
		isSelected := m.mode == InputModeVisual && i >= visualMin && i <= visualMax && !p.IsSeparator && !p.IsComment

		if p.IsSeparator {
			rows = append(rows, m.renderSeparator(paneWidth))
			continue
		}

		if p.IsComment {
			rows = append(rows, m.renderCommentRow(p, paneWidth, isCursor))
			continue
		}

		if p.IsWrapContinuation {
			leftStr := m.renderWrapContinuation(p.WrapSourceLeft, SideOld, paneWidth, lineNumWidth, p.WrapRow, isCursor, m.activeSide == SideOld)
			rightStr := m.renderWrapContinuation(p.WrapSourceRight, SideNew, paneWidth, lineNumWidth, p.WrapRow, isCursor, m.activeSide == SideNew)
			row := lipgloss.JoinHorizontal(lipgloss.Top, leftStr, "│", rightStr)
			rows = append(rows, row)
			continue
		}

		// Prepare highlighted content for each side
		leftHL, rightHL := m.highlightPair(p)

		leftStr := m.renderPane(p.Left, SideOld, paneWidth, lineNumWidth, isCursor || (isSelected && m.activeSide == SideOld), m.activeSide == SideOld, leftHL)
		rightStr := m.renderPane(p.Right, SideNew, paneWidth, lineNumWidth, isCursor || (isSelected && m.activeSide == SideNew), m.activeSide == SideNew, rightHL)
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
	var oldMode, newMode os.FileMode
	if len(m.files) > 0 && m.activeFile < len(m.files) {
		f := m.files[m.activeFile]
		filePath = fileDisplayPath(f)
		oldMode = f.OldMode
		newMode = f.NewMode
	}
	statusBar := RenderStatusBarWithMode(m.width, m.ref, m.activeFile, len(m.files), filePath, adds, dels, m.commentCount(), m.activeSide, oldMode, newMode)

	var helpBar string
	if m.mode == InputModeSearch {
		helpBar = m.renderSearchPrompt()
	} else if m.search.Active {
		helpBar = m.renderSearchInfo()
	} else {
		helpBar = RenderHelpBar(m.width, m.mode)
	}

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

	// Render fuzzy overlay when active
	if m.mode == InputModeFuzzy && m.fuzzy.Active {
		result = RenderFuzzyOverlay(&m.fuzzy, m.width, m.height, m.config.Colors.CursorActive)
	}

	// Render help overlay when active
	if m.mode == InputModeHelp {
		helpBox := RenderHelpOverlay(m.helpEntries, m.width, m.height, m.helpYOffset)
		result = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpBox)
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
				hl = render.ApplyEmphasis(hl, oldMask, m.config.Colors.EmphasisDelete)
			}
			// Apply search highlight (active search or live preview while typing)
			if m.search.Query != "" {
				searchMask := BuildSearchMask(p.Left.Content, m.search.Query)
				if searchMask != nil {
					hl = render.ApplyEmphasis(hl, searchMask, colorSearchMatch)
				}
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
				hl = render.ApplyEmphasis(hl, newMask, m.config.Colors.EmphasisAdd)
			}
			// Apply search highlight (active search or live preview while typing)
			if m.search.Query != "" {
				searchMask := BuildSearchMask(p.Right.Content, m.search.Query)
				if searchMask != nil {
					hl = render.ApplyEmphasis(hl, searchMask, colorSearchMatch)
				}
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

	// Gutter marker: +/- for changes, space for context
	var gutter string
	switch line.Type {
	case diff.LineAdd:
		gutter = "+"
	case diff.LineDelete:
		gutter = "-"
	default:
		gutter = " "
	}

	contentWidth := paneWidth - lineNumWidth - 1 // -1 for gutter marker
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

	lineStr := numStr + "│" + gutter + content

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

	// Background: cursor overrides line tint, otherwise changed lines get subtle color
	if isCursor {
		if isActiveSide {
			style = style.Background(lipgloss.Color(m.config.Colors.CursorActive))
		} else {
			style = style.Background(lipgloss.Color(m.config.Colors.CursorInactive))
		}
	} else {
		switch line.Type {
		case diff.LineAdd:
			style = style.Background(lipgloss.Color(m.config.Colors.AddBg))
		case diff.LineDelete:
			style = style.Background(lipgloss.Color(m.config.Colors.DeleteBg))
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
			style = style.Background(lipgloss.Color(m.config.Colors.CursorActive))
		} else {
			style = style.Background(lipgloss.Color(m.config.Colors.CursorInactive))
		}
	} else {
		style = style.Background(lipgloss.Color(colorPaddingBg))
	}
	return style.Render(blank)
}

// renderWrapContinuation renders a continuation row for a wrapped line.
// wrapRow is the 1-based visual row index within the wrapped line.
// Uses ANSI-aware slicing to correctly handle syntax-highlighted content.
func (m Model) renderWrapContinuation(source *diff.DiffLine, side Side, paneWidth, lineNumWidth, wrapRow int, isCursor, isActiveSide bool) string {
	if source == nil {
		return m.renderBlankPadding(paneWidth, isCursor, isActiveSide)
	}

	contentWidth := paneWidth - lineNumWidth - 1 // -1 for gutter
	if contentWidth < 1 {
		contentWidth = 1
	}

	content := source.Content
	// Use ANSI-aware Cut to extract the correct visual slice
	left := wrapRow * contentWidth
	right := left + contentWidth
	totalWidth := ansi.StringWidth(content)

	if left >= totalWidth {
		// This row is beyond the content — render as blank padding within the line
		numStr := "    │ "
		padded := numStr + strings.Repeat(" ", contentWidth)
		style := lipgloss.NewStyle()
		if isCursor {
			if isActiveSide {
				style = style.Background(lipgloss.Color(m.config.Colors.CursorActive))
			} else {
				style = style.Background(lipgloss.Color(m.config.Colors.CursorInactive))
			}
		}
		return style.Render(padded)
	}

	if right > totalWidth {
		right = totalWidth
	}
	slice := ansi.Cut(content, left, right)

	// Pad to fill content width
	displayWidth := ansi.StringWidth(slice)
	if displayWidth < contentWidth {
		slice = slice + strings.Repeat(" ", contentWidth-displayWidth)
	}

	// Continuation rows have blank line number area + space gutter
	numStr := "    │ "
	lineStr := numStr + slice

	style := lipgloss.NewStyle()
	switch source.Type {
	case diff.LineAdd:
		style = style.Foreground(lipgloss.Color(colorAdd))
	case diff.LineDelete:
		style = style.Foreground(lipgloss.Color(colorDelete))
	}

	if isCursor {
		if isActiveSide {
			style = style.Background(lipgloss.Color(m.config.Colors.CursorActive))
		} else {
			style = style.Background(lipgloss.Color(m.config.Colors.CursorInactive))
		}
	} else {
		switch source.Type {
		case diff.LineAdd:
			style = style.Background(lipgloss.Color(m.config.Colors.AddBg))
		case diff.LineDelete:
			style = style.Background(lipgloss.Color(m.config.Colors.DeleteBg))
		}
	}

	return style.Render(lineStr)
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
		style = style.Background(lipgloss.Color(m.config.Colors.CursorActive))
	}

	return style.Render(text)
}

// renderSeparator renders a hunk separator row.
func (m Model) renderSeparator(paneWidth int) string {
	totalWidth := paneWidth*2 + 1
	hint := " expand context (]e / [e) "
	hintWidth := ansi.StringWidth(hint)
	if totalWidth < hintWidth+4 {
		// Too narrow for hint, just dashes
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSeparator))
		return style.Render(strings.Repeat("─", totalWidth))
	}
	leftDashes := (totalWidth - hintWidth) / 2
	rightDashes := totalWidth - hintWidth - leftDashes
	sep := strings.Repeat("─", leftDashes) + hint + strings.Repeat("─", rightDashes)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSeparator))
	return style.Render(sep)
}

// fileDisplayPath returns the display path for the status bar.
// Renamed files show "old → new", otherwise just the file path.
func fileDisplayPath(f diff.DiffFile) string {
	if f.IsRename {
		return f.OldName + " → " + f.NewName
	}
	path := f.NewName
	if path == "" {
		path = f.OldName
	}
	return path
}

// specialFileMessage returns a centered message for special file types, or empty string for normal files.
func (m Model) specialFileMessage(f diff.DiffFile) string {
	if f.IsBinary {
		return "Binary file changed — cannot display diff"
	}
	if f.IsSubmodule {
		oldSHA, newSHA := extractSubmoduleSHAs(f)
		return fmt.Sprintf("Submodule updated: %s → %s", oldSHA, newSHA)
	}
	if f.IsRename && len(f.Hunks) == 0 {
		return "File renamed — no content changes"
	}
	if f.OldMode != 0 && f.NewMode != 0 && f.OldMode != f.NewMode && len(f.Hunks) == 0 {
		return fmt.Sprintf("File mode changed: %04o → %04o", f.OldMode, f.NewMode)
	}
	return ""
}

// extractSubmoduleSHAs extracts old and new commit SHAs from submodule hunk lines.
// Returns short SHAs (7 chars) for readability.
func extractSubmoduleSHAs(f diff.DiffFile) (string, string) {
	var oldSHA, newSHA string
	for _, h := range f.Hunks {
		for _, l := range h.Lines {
			if strings.HasPrefix(l.Content, "Subproject commit ") {
				sha := strings.TrimPrefix(l.Content, "Subproject commit ")
				if len(sha) > 7 {
					sha = sha[:7]
				}
				if l.Type == diff.LineDelete {
					oldSHA = sha
				} else if l.Type == diff.LineAdd {
					newSHA = sha
				}
			}
		}
	}
	if oldSHA == "" {
		oldSHA = "(none)"
	}
	if newSHA == "" {
		newSHA = "(none)"
	}
	return oldSHA, newSHA
}

// renderSpecialFileView renders the full view for a special file (binary, submodule, etc.)
// with a centered message instead of diff content.
func (m Model) renderSpecialFileView(f diff.DiffFile, msg string) string {
	vis := m.visibleRows()

	// Center the message in the viewport
	diffContent := lipgloss.Place(m.width, vis, lipgloss.Center, lipgloss.Center, msg)

	// Status bar
	filePath := fileDisplayPath(f)

	adds, dels := m.countFileChanges()
	statusBar := RenderStatusBarWithMode(m.width, m.ref, m.activeFile, len(m.files), filePath, adds, dels, m.commentCount(), m.activeSide, f.OldMode, f.NewMode)

	helpBar := RenderHelpBar(m.width, m.mode)

	return statusBar + "\n" + diffContent + "\n" + helpBar
}

// openFuzzyFiles opens the built-in fuzzy overlay for file search.
func (m *Model) openFuzzyFiles() {
	items := BuildFileList(m.files)
	m.fuzzy = NewFuzzyState(FuzzyModeFiles, items)
	m.mode = InputModeFuzzy
}

// openFuzzyContent opens the built-in fuzzy overlay for content search.
func (m *Model) openFuzzyContent() {
	contentItems := BuildContentList(m.files)
	items := make([]string, len(contentItems))
	for i, ci := range contentItems {
		items[i] = ci.Display
	}
	m.fuzzy = NewFuzzyState(FuzzyModeContent, items)
	m.fuzzy.ContentItems = contentItems
	m.mode = InputModeFuzzy
}

// handleFuzzyKey handles key input when in fuzzy overlay mode.
// Navigation uses arrow keys and Ctrl-j/Ctrl-k only; all other keys go to the text input.
// handleHelpKey handles key input when the help overlay is open.
// Only Esc, ?, and j/k (scroll) are active; all other keys are ignored.
func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = InputModeNormal
		return m, nil
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "?":
			m.mode = InputModeNormal
			return m, nil
		case "j":
			m.helpYOffset++
			// Upper bound is clamped inside RenderHelpOverlay
			return m, nil
		case "k":
			if m.helpYOffset > 0 {
				m.helpYOffset--
			}
			return m, nil
		}
	}
	// All other keys are ignored in help mode
	return m, nil
}

func (m Model) handleFuzzyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.fuzzy = FuzzyState{}
		m.mode = InputModeNormal
		return m, nil
	case tea.KeyEnter:
		return m.fuzzySelect()
	case tea.KeyUp, tea.KeyCtrlK:
		m.fuzzy.CursorUp()
		m.fuzzy.EnsureCursorVisible(m.fuzzyMaxItems())
		return m, nil
	case tea.KeyDown, tea.KeyCtrlJ:
		m.fuzzy.CursorDown()
		m.fuzzy.EnsureCursorVisible(m.fuzzyMaxItems())
		return m, nil
	default:
		// Forward all other keys to text input for filtering
		m.fuzzy.Input, _ = m.fuzzy.Input.Update(msg)
		m.fuzzy.UpdateFilter(m.fuzzy.Input.Value())
		return m, nil
	}
}

// fuzzyMaxItems returns the number of visible items in the fuzzy overlay.
func (m Model) fuzzyMaxItems() int {
	maxItems := m.height - 4
	if maxItems < 1 {
		maxItems = 1
	}
	return maxItems
}

// fuzzySelect handles Enter in fuzzy mode — navigates to the selected item.
func (m Model) fuzzySelect() (tea.Model, tea.Cmd) {
	if m.fuzzy.Mode == FuzzyModeFiles {
		item, ok := m.fuzzy.SelectedItem()
		if !ok {
			m.fuzzy = FuzzyState{}
			m.mode = InputModeNormal
			return m, nil
		}
		idx := FindFileIndex(m.files, item)
		if idx >= 0 {
			m.activeFile = idx
			m.cursorRow = 0
			m.yOffset = 0
			if m.allPaired != nil && idx < len(m.allPaired) {
				m.paired = m.allPaired[idx]
			} else {
				m.paired = diff.BuildPairedLines(m.files[idx].Hunks)
			}
			m.recomputeSearch()
		}
	} else {
		ci, ok := m.fuzzy.SelectedContentItem()
		if !ok {
			m.fuzzy = FuzzyState{}
			m.mode = InputModeNormal
			return m, nil
		}
		idx := ci.FileIndex
		if idx >= 0 && idx < len(m.files) {
			m.activeFile = idx
			if m.allPaired != nil && idx < len(m.allPaired) {
				m.paired = m.allPaired[idx]
			} else {
				m.paired = diff.BuildPairedLines(m.files[idx].Hunks)
			}
			// Determine side from line type: adds are on new side, deletes on old side
			side := m.activeSide
			switch ci.LineType {
			case diff.LineAdd:
				side = SideNew
			case diff.LineDelete:
				side = SideOld
			}
			m.cursorRow = FindRowForLine(m.paired, ci.LineNum, side)
			m.yOffset = 0
			m.scrollToCursor()
			m.recomputeSearch()
		}
	}
	m.fuzzy = FuzzyState{}
	m.mode = InputModeNormal
	return m, nil
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
	m.recomputeSearch()
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
				endLine := m.overlay.EndLine
				snippet := m.overlay.rangeSnippet
				if snippet == "" {
					// Single-line comment: get snippet from cursor line
					p := m.paired[m.overlay.RowIndex]
					line := m.activeSideLine(p)
					if line != nil {
						snippet = strings.TrimSpace(line.Content)
					}
				}
				_ = m.store.AddComment(filePath, m.overlay.Line, endLine, snippet, body)
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

// enterVisualMode starts visual selection at the current cursor position.
// No-op on separator or comment rows.
func (m *Model) enterVisualMode() {
	if len(m.paired) == 0 || m.cursorRow >= len(m.paired) {
		return
	}
	p := m.paired[m.cursorRow]
	if p.IsSeparator || p.IsComment {
		return
	}
	m.visualStart = m.cursorRow
	m.mode = InputModeVisual
}

// handleVisualKey handles key input when in visual select mode.
func (m Model) handleVisualKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = InputModeNormal
		return m, nil
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			if m.cursorRow < len(m.paired)-1 {
				m.cursorRow++
			}
			m.scrollToCursor()
		case "k":
			if m.cursorRow > 0 {
				m.cursorRow--
			}
			m.scrollToCursor()
		case "h":
			m.activeSide = SideOld
		case "l":
			m.activeSide = SideNew
		case "c":
			m.openRangeCommentOverlay()
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// openRangeCommentOverlay opens the comment overlay for the visual selection range.
func (m *Model) openRangeCommentOverlay() {
	if m.store == nil {
		return
	}

	// Determine the selected range
	startRow := m.visualStart
	endRow := m.cursorRow
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}

	// Find the line numbers from code lines in the range (skip separator/comment rows)
	var startLine, endLine int
	var snippet string
	for i := startRow; i <= endRow; i++ {
		p := m.paired[i]
		if p.IsSeparator || p.IsComment {
			continue
		}
		line := m.activeSideLine(p)
		if line == nil {
			continue
		}
		lineNum := m.activeSideLineNum(line)
		if startLine == 0 || lineNum < startLine {
			startLine = lineNum
			snippet = strings.TrimSpace(line.Content)
		}
		if lineNum > endLine {
			endLine = lineNum
		}
	}

	if startLine == 0 {
		return // no valid code lines in selection
	}

	// If start and end are the same, treat as single-line comment
	if startLine == endLine {
		endLine = 0
	}

	m.overlay = NewCommentOverlay(startLine, m.activeSide, m.cursorRow)
	m.overlay.EndLine = endLine
	m.overlay.rangeSnippet = snippet
	m.mode = InputModeComment
}

// renderSearchPrompt renders the search input prompt (replaces help bar during search input).
func (m Model) renderSearchPrompt() string {
	prompt := "/" + m.search.Input.View()

	contentWidth := ansi.StringWidth(prompt)
	padding := m.width - contentWidth
	if padding < 0 {
		padding = 0
	}
	prompt = prompt + strings.Repeat(" ", padding)

	style := lipgloss.NewStyle().
		Background(lipgloss.Color(colorBarBg))

	return style.Render(prompt)
}

// renderSearchInfo renders the search results bar (replaces help bar when search is active).
func (m Model) renderSearchInfo() string {
	var info string
	if len(m.search.Matches) == 0 {
		info = fmt.Sprintf(" /%s  no matches", m.search.Query)
	} else {
		info = fmt.Sprintf(" /%s  [%d/%d]", m.search.Query, m.search.Current+1, len(m.search.Matches))
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorBarFg))
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorBarDim))
	hints := keyStyle.Render("[n]") + " " + actionStyle.Render("next") + "  " +
		keyStyle.Render("[N]") + " " + actionStyle.Render("prev") + "  " +
		keyStyle.Render("[Esc]") + " " + actionStyle.Render("clear")

	content := info + "  " + hints

	contentWidth := ansi.StringWidth(content)
	padding := m.width - contentWidth
	if padding < 0 {
		padding = 0
	}
	content = content + strings.Repeat(" ", padding)

	style := lipgloss.NewStyle().
		Background(lipgloss.Color(colorBarBg))

	return style.Render(content)
}

// openSearch enters search input mode.
func (m *Model) openSearch() {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 0
	ti.Placeholder = "search..."
	m.search = SearchState{
		Input: ti,
	}
	m.mode = InputModeSearch
}

// handleSearchKey handles key input when in search input mode.
func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.search = SearchState{}
		m.mode = InputModeNormal
		return m, nil
	case tea.KeyEnter:
		query := m.search.Input.Value()
		matches := FindMatches(m.paired, query)
		m.search.Active = true
		m.search.Query = query
		m.search.Matches = matches
		m.search.Current = 0
		m.mode = InputModeNormal

		// Jump to first match after current cursor
		if len(matches) > 0 {
			idx := m.search.FirstMatchAfter(m.cursorRow)
			m.search.Current = idx
			m.cursorRow = matches[idx].Row
			m.scrollToCursor()
		}

		return m, nil
	default:
		var cmd tea.Cmd
		m.search.Input, cmd = m.search.Input.Update(msg)
		// Update matches in real-time as user types
		query := m.search.Input.Value()
		m.search.Query = query
		m.search.Matches = FindMatches(m.paired, query)
		m.search.Current = 0
		return m, cmd
	}
}

// searchNext jumps to the next search match.
func (m *Model) searchNext() {
	if len(m.search.Matches) == 0 {
		return
	}
	m.search.Next()
	m.cursorRow = m.search.Matches[m.search.Current].Row
	m.scrollToCursor()
}

// searchPrev jumps to the previous search match.
func (m *Model) searchPrev() {
	if len(m.search.Matches) == 0 {
		return
	}
	m.search.Prev()
	m.cursorRow = m.search.Matches[m.search.Current].Row
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
