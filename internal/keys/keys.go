package keys

import (
	"strings"
	"unicode/utf8"

	"github.com/adil/cr/internal/config"
)

// Action represents a resolved keybinding action.
type Action int

const (
	ActionNone Action = iota
	ActionDiscard
	ActionScrollDown
	ActionScrollUp
	ActionPaneLeft
	ActionPaneRight
	ActionHalfPageDown
	ActionHalfPageUp
	ActionTop
	ActionBottom
	ActionNextHunk
	ActionPrevHunk
	ActionNextFile
	ActionPrevFile
	ActionNextComment
	ActionPrevComment
	ActionExpandBelow
	ActionExpandAbove
	ActionToggleTree
	ActionFuzzyFiles
	ActionSearchDiffs
	ActionToggleWordDiff
	ActionToggleWrap
	ActionSearch
	ActionHelp
	ActionQuit
	ActionCommentAdd
	ActionVisualSelect
	ActionCommentEdit
	ActionCommentDelete
)

// actionNames maps config action names to Action constants.
var actionNames = map[string]Action{
	"scroll_down":      ActionScrollDown,
	"scroll_up":        ActionScrollUp,
	"pane_left":        ActionPaneLeft,
	"pane_right":       ActionPaneRight,
	"half_page_down":   ActionHalfPageDown,
	"half_page_up":     ActionHalfPageUp,
	"top":              ActionTop,
	"bottom":           ActionBottom,
	"next_hunk":        ActionNextHunk,
	"prev_hunk":        ActionPrevHunk,
	"next_file":        ActionNextFile,
	"prev_file":        ActionPrevFile,
	"next_comment":     ActionNextComment,
	"prev_comment":     ActionPrevComment,
	"expand_below":     ActionExpandBelow,
	"expand_above":     ActionExpandAbove,
	"toggle_tree":      ActionToggleTree,
	"fuzzy_files":      ActionFuzzyFiles,
	"search_diffs":     ActionSearchDiffs,
	"toggle_word_diff": ActionToggleWordDiff,
	"toggle_wrap":      ActionToggleWrap,
	"search":           ActionSearch,
	"help":             ActionHelp,
	"quit":             ActionQuit,
	"add":              ActionCommentAdd,
	"visual":           ActionVisualSelect,
	"edit":             ActionCommentEdit,
	"delete":           ActionCommentDelete,
}

// Resolve maps a key string and optional pending key prefix to an Action
// using the config's keybinding maps.
func Resolve(cfg config.Config, keyStr string, pendingKey string) Action {
	if pendingKey != "" {
		seq := pendingKey + " " + keyStr
		if action, ok := lookupSequence(cfg, seq); ok {
			return action
		}
		return ActionDiscard
	}

	if IsPrefix(cfg, keyStr) {
		return ActionNone
	}

	return lookupSingle(cfg, keyStr)
}

// IsPrefix returns true if keyStr is a prefix key that should set pendingKey.
func IsPrefix(cfg config.Config, keyStr string) bool {
	for _, key := range cfg.Keys.Normal {
		if p, ok := splitPrefix(key); ok && p == keyStr {
			return true
		}
	}
	for _, key := range cfg.Keys.Comment {
		if p, ok := splitPrefix(key); ok && p == keyStr {
			return true
		}
	}
	return false
}

// splitPrefix splits a multi-key binding into its prefix and suffix.
// Returns ("", false) for single-key bindings.
func splitPrefix(key string) (string, bool) {
	if utf8.RuneCountInString(key) < 2 {
		return "", false
	}
	// Handle "Space X" style (space-separated)
	if i := strings.IndexByte(key, ' '); i > 0 {
		return key[:i], true
	}
	// Handle "gg", "]c", "[c", "dc" style (two runes, no separator)
	if utf8.RuneCountInString(key) == 2 {
		r, size := utf8.DecodeRuneInString(key)
		_ = size
		return string(r), true
	}
	// Handle "Ctrl-d" etc — not a prefix sequence
	return "", false
}

// lookupSequence checks if a "prefix suffix" sequence maps to an action.
func lookupSequence(cfg config.Config, seq string) (Action, bool) {
	for action, key := range cfg.Keys.Normal {
		if normalizeBinding(key) == seq {
			if a, ok := actionNames[action]; ok {
				return a, true
			}
		}
	}
	for action, key := range cfg.Keys.Comment {
		if normalizeBinding(key) == seq {
			if a, ok := actionNames[action]; ok {
				return a, true
			}
		}
	}
	return ActionNone, false
}

// normalizeBinding converts a key binding to "prefix suffix" format.
// "gg" → "g g", "]c" → "] c", "dc" → "d c", "Space f" → "Space f"
func normalizeBinding(key string) string {
	// Already space-separated
	if strings.ContainsRune(key, ' ') {
		return key
	}
	// Two-rune binding without space
	if utf8.RuneCountInString(key) == 2 {
		r, size := utf8.DecodeRuneInString(key)
		return string(r) + " " + key[size:]
	}
	return key
}

// lookupSingle checks if a single key maps to an action.
func lookupSingle(cfg config.Config, keyStr string) Action {
	for action, key := range cfg.Keys.Normal {
		if key == keyStr {
			if a, ok := actionNames[action]; ok {
				return a
			}
		}
	}
	for action, key := range cfg.Keys.Comment {
		if key == keyStr {
			if a, ok := actionNames[action]; ok {
				return a
			}
		}
	}
	return ActionNone
}
