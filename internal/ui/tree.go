package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/adil/cr/internal/diff"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// TreeEntry represents a single entry in the file tree (file or directory).
type TreeEntry struct {
	Path        string // display path (compact, e.g. "src/services/auth/")
	FullPath    string // original full path for the file
	IsDir       bool
	Depth       int
	FileIndex   int // index into Model.files (-1 for dirs)
	IsBinary    bool
	IsRename    bool
	IsSubmodule bool
}

// TreeState holds the state of the file tree sidebar.
type TreeState struct {
	Open      bool
	Cursor    int
	Entries   []TreeEntry
	Collapsed map[string]bool // collapsed directory full paths
	YOffset   int             // scroll offset for tree
}

// dirNode is an internal tree node used during tree construction.
type dirNode struct {
	name        string
	children    []*dirNode // ordered children
	isDir       bool
	fileIdx     int // -1 for dirs
	isBinary    bool
	isRename    bool
	isSubmodule bool
	fullPath    string // full path from root
}

// NewTreeState builds a TreeState from the list of diff files.
func NewTreeState(files []diff.DiffFile) TreeState {
	if len(files) == 0 {
		return TreeState{Collapsed: make(map[string]bool)}
	}

	// Build a tree of dirNodes
	root := &dirNode{name: "", isDir: true, fileIdx: -1}

	for i, f := range files {
		path := f.NewName
		if path == "" {
			path = f.OldName
		}
		parts := strings.Split(filepath.ToSlash(path), "/")
		insertPath(root, parts, i, f.IsBinary, f.IsRename, f.IsSubmodule, "")
	}

	// Compact single-child directory chains
	compactTree(root)

	// Flatten tree into entries
	var entries []TreeEntry
	for _, child := range root.children {
		flattenNode(child, 0, &entries)
	}

	return TreeState{
		Entries:   entries,
		Collapsed: make(map[string]bool),
	}
}

// insertPath inserts a file path into the tree, creating intermediate directory nodes.
// parentPath accumulates the full path from root (e.g., "src/services/").
func insertPath(node *dirNode, parts []string, fileIdx int, isBinary, isRename, isSubmodule bool, parentPath string) {
	fullPath := parentPath + parts[0]
	if len(parts) == 1 {
		// Leaf file node
		node.children = append(node.children, &dirNode{
			name:        parts[0],
			isDir:       false,
			fileIdx:     fileIdx,
			isBinary:    isBinary,
			isRename:    isRename,
			isSubmodule: isSubmodule,
			fullPath:    fullPath,
		})
		return
	}

	// Find or create the directory child
	dirName := parts[0]
	var child *dirNode
	for _, c := range node.children {
		if c.isDir && c.name == dirName {
			child = c
			break
		}
	}
	if child == nil {
		child = &dirNode{name: dirName, isDir: true, fileIdx: -1, fullPath: fullPath}
		node.children = append(node.children, child)
	}

	insertPath(child, parts[1:], fileIdx, isBinary, isRename, isSubmodule, fullPath+"/")
}

// compactTree merges single-child directory chains.
// e.g., src -> services -> auth -> handler.go becomes "src/services/auth" -> handler.go
func compactTree(node *dirNode) {
	for _, child := range node.children {
		compactTree(child)
	}

	// Merge: if this dir has exactly one child and that child is also a dir
	for i, child := range node.children {
		if child.isDir {
			for len(child.children) == 1 && child.children[0].isDir {
				grandchild := child.children[0]
				child.name = child.name + "/" + grandchild.name
				child.fullPath = grandchild.fullPath
				child.children = grandchild.children
			}
			node.children[i] = child
		}
	}
}

// flattenNode recursively flattens a dirNode into a list of TreeEntry.
func flattenNode(node *dirNode, depth int, entries *[]TreeEntry) {
	if node.isDir {
		*entries = append(*entries, TreeEntry{
			Path:      node.name + "/",
			FullPath:  node.fullPath,
			IsDir:     true,
			Depth:     depth,
			FileIndex: -1,
		})
		for _, child := range node.children {
			flattenNode(child, depth+1, entries)
		}
	} else {
		*entries = append(*entries, TreeEntry{
			Path:        node.name,
			FullPath:    node.fullPath,
			IsDir:       false,
			Depth:       depth,
			FileIndex:   node.fileIdx,
			IsBinary:    node.isBinary,
			IsRename:    node.isRename,
			IsSubmodule: node.isSubmodule,
		})
	}
}

// VisibleEntries returns entries that are visible (not hidden by collapsed parents).
func (ts *TreeState) VisibleEntries() []TreeEntry {
	if len(ts.Entries) == 0 {
		return nil
	}

	var visible []TreeEntry
	skipDepth := -1 // skip children deeper than this

	for _, e := range ts.Entries {
		if skipDepth >= 0 && e.Depth > skipDepth {
			continue
		}
		skipDepth = -1
		visible = append(visible, e)

		if e.IsDir && ts.Collapsed[e.FullPath] {
			skipDepth = e.Depth
		}
	}
	return visible
}

// ToggleCollapse toggles the collapsed state of a directory.
func (ts *TreeState) ToggleCollapse(fullPath string) {
	if ts.Collapsed[fullPath] {
		delete(ts.Collapsed, fullPath)
	} else {
		ts.Collapsed[fullPath] = true
	}
}

// CursorDown moves the tree cursor down within visible entries.
func (ts *TreeState) CursorDown() {
	visible := ts.VisibleEntries()
	if ts.Cursor < len(visible)-1 {
		ts.Cursor++
	}
}

// CursorUp moves the tree cursor up within visible entries.
func (ts *TreeState) CursorUp() {
	if ts.Cursor > 0 {
		ts.Cursor--
	}
}

// SelectedFileIndex returns the FileIndex of the entry under the cursor,
// or -1 if the cursor is on a directory.
func (ts *TreeState) SelectedFileIndex() int {
	visible := ts.VisibleEntries()
	if ts.Cursor < 0 || ts.Cursor >= len(visible) {
		return -1
	}
	return visible[ts.Cursor].FileIndex
}

// SelectedEntry returns the TreeEntry under the cursor in visible entries.
// Returns the entry and true if found, or zero value and false if cursor is out of range.
func (ts *TreeState) SelectedEntry() (TreeEntry, bool) {
	visible := ts.VisibleEntries()
	if ts.Cursor < 0 || ts.Cursor >= len(visible) {
		return TreeEntry{}, false
	}
	return visible[ts.Cursor], true
}

// TreeWidth returns the width of the tree panel (20% of terminal width).
func TreeWidth(termWidth int) int {
	w := termWidth * 20 / 100
	if w < 10 {
		w = 10
	}
	return w
}

// RenderTree renders the file tree sidebar.
// commentCounts maps file paths to the number of comments on that file (may be nil).
func RenderTree(ts *TreeState, width, height, activeFile int, commentCounts map[string]int) string {
	visible := ts.VisibleEntries()
	if len(visible) == 0 {
		return ""
	}

	// Handle tree scrolling
	if ts.Cursor < ts.YOffset {
		ts.YOffset = ts.Cursor
	}
	if ts.Cursor >= ts.YOffset+height {
		ts.YOffset = ts.Cursor - height + 1
	}

	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorTreeDir))
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorTreeFile))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorTreeActive))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorTreeDim))
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color(colorTreeCursor))

	var rows []string
	end := ts.YOffset + height
	if end > len(visible) {
		end = len(visible)
	}

	for i := ts.YOffset; i < end; i++ {
		e := visible[i]
		isCursor := i == ts.Cursor

		indent := strings.Repeat("  ", e.Depth)
		var line string

		if e.IsDir {
			line = indent + dirStyle.Render(e.Path)
		} else {
			// File entry
			prefix := "  "
			if e.FileIndex == activeFile {
				prefix = activeStyle.Render("● ")
			}

			suffix := ""
			if e.IsRename {
				suffix = " " + dimStyle.Render("→")
			}
			if e.IsBinary {
				suffix = " " + dimStyle.Render("[bin]")
			}
			if e.IsSubmodule {
				suffix = " " + dimStyle.Render("[sub]")
			}

			// Show comment count indicator (appended, not replacing type indicator)
			if commentCounts != nil {
				if n, ok := commentCounts[e.FullPath]; ok && n > 0 {
					suffix += " " + dimStyle.Render(fmt.Sprintf("💬 (%d)", n))
				}
			}

			line = indent + prefix + fileStyle.Render(e.Path) + suffix
		}

		// Clip to width
		displayWidth := ansi.StringWidth(line)
		if displayWidth > width {
			line = ansi.Truncate(line, width, "")
			displayWidth = ansi.StringWidth(line)
		}
		// Pad to fill width
		if displayWidth < width {
			line = line + strings.Repeat(" ", width-displayWidth)
		}

		if isCursor {
			line = cursorStyle.Render(line)
		}

		rows = append(rows, line)
	}

	// Pad remaining rows if tree has fewer entries than height
	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", width))
	}

	return strings.Join(rows, "\n")
}
