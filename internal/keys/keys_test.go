package keys

import (
	"testing"

	"github.com/adil/cr/internal/config"
)

func defaultCfg() config.Config {
	cfg, _ := config.Load("/nonexistent/path")
	return cfg
}

func TestResolve(t *testing.T) {
	cfg := defaultCfg()

	tests := []struct {
		name       string
		keyStr     string
		pendingKey string
		wantAction Action
	}{
		// Single-key actions
		{name: "j → scroll down", keyStr: "j", wantAction: ActionScrollDown},
		{name: "k → scroll up", keyStr: "k", wantAction: ActionScrollUp},
		{name: "h → pane left", keyStr: "h", wantAction: ActionPaneLeft},
		{name: "l → pane right", keyStr: "l", wantAction: ActionPaneRight},
		{name: "G → bottom", keyStr: "G", wantAction: ActionBottom},
		{name: "q → quit", keyStr: "q", wantAction: ActionQuit},
		{name: "w → toggle word diff", keyStr: "w", wantAction: ActionToggleWordDiff},
		{name: "c → comment add", keyStr: "c", wantAction: ActionCommentAdd},
		{name: "e → comment edit", keyStr: "e", wantAction: ActionCommentEdit},
		{name: "V → visual select", keyStr: "V", wantAction: ActionVisualSelect},

		// Prefix keys with no pending → ActionNone (they wait for next key)
		{name: "g alone → none", keyStr: "g", wantAction: ActionNone},
		{name: "] alone → none", keyStr: "]", wantAction: ActionNone},
		{name: "[ alone → none", keyStr: "[", wantAction: ActionNone},
		{name: "d alone → none", keyStr: "d", wantAction: ActionNone},

		// Two-key sequences (pending key set)
		{name: "gg → top", keyStr: "g", pendingKey: "g", wantAction: ActionTop},
		{name: "]c → next hunk", keyStr: "c", pendingKey: "]", wantAction: ActionNextHunk},
		{name: "[c → prev hunk", keyStr: "c", pendingKey: "[", wantAction: ActionPrevHunk},
		{name: "]f → next file", keyStr: "f", pendingKey: "]", wantAction: ActionNextFile},
		{name: "[f → prev file", keyStr: "f", pendingKey: "[", wantAction: ActionPrevFile},
		{name: "]m → next comment", keyStr: "m", pendingKey: "]", wantAction: ActionNextComment},
		{name: "[m → prev comment", keyStr: "m", pendingKey: "[", wantAction: ActionPrevComment},
		{name: "]e → expand below", keyStr: "e", pendingKey: "]", wantAction: ActionExpandBelow},
		{name: "[e → expand above", keyStr: "e", pendingKey: "[", wantAction: ActionExpandAbove},
		{name: "dc → comment delete", keyStr: "c", pendingKey: "d", wantAction: ActionCommentDelete},

		// Leader sequences
		{name: "Space f → fuzzy files", keyStr: "f", pendingKey: "Space", wantAction: ActionFuzzyFiles},
		{name: "Space s → search diffs", keyStr: "s", pendingKey: "Space", wantAction: ActionSearchDiffs},

		// Invalid completions → discard
		{name: "gx → discard", keyStr: "x", pendingKey: "g", wantAction: ActionDiscard},
		{name: "]x → discard", keyStr: "x", pendingKey: "]", wantAction: ActionDiscard},
		{name: "[x → discard", keyStr: "x", pendingKey: "[", wantAction: ActionDiscard},
		{name: "dx → discard", keyStr: "x", pendingKey: "d", wantAction: ActionDiscard},
		{name: "Space x → discard", keyStr: "x", pendingKey: "Space", wantAction: ActionDiscard},

		// Unknown single key → none
		{name: "unknown key x → none", keyStr: "x", wantAction: ActionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(cfg, tt.keyStr, tt.pendingKey)
			if got != tt.wantAction {
				t.Errorf("Resolve(%q, %q) = %v, want %v", tt.keyStr, tt.pendingKey, got, tt.wantAction)
			}
		})
	}
}

func TestIsPrefix(t *testing.T) {
	cfg := defaultCfg()

	tests := []struct {
		name   string
		keyStr string
		want   bool
	}{
		{name: "g is prefix", keyStr: "g", want: true},
		{name: "] is prefix", keyStr: "]", want: true},
		{name: "[ is prefix", keyStr: "[", want: true},
		{name: "d is prefix", keyStr: "d", want: true},
		{name: "Space is prefix", keyStr: "Space", want: true},
		{name: "j is not prefix", keyStr: "j", want: false},
		{name: "k is not prefix", keyStr: "k", want: false},
		{name: "q is not prefix", keyStr: "q", want: false},
		{name: "G is not prefix", keyStr: "G", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPrefix(cfg, tt.keyStr)
			if got != tt.want {
				t.Errorf("IsPrefix(%q) = %v, want %v", tt.keyStr, got, tt.want)
			}
		})
	}
}

func TestResolveSpecialKeys(t *testing.T) {
	cfg := defaultCfg()

	tests := []struct {
		name       string
		keyStr     string
		wantAction Action
	}{
		{name: "Ctrl-d → half page down", keyStr: "Ctrl-d", wantAction: ActionHalfPageDown},
		{name: "Ctrl-u → half page up", keyStr: "Ctrl-u", wantAction: ActionHalfPageUp},
		{name: "Tab → toggle tree", keyStr: "Tab", wantAction: ActionToggleTree},
		{name: "/ → search", keyStr: "/", wantAction: ActionSearch},
		{name: "? → help", keyStr: "?", wantAction: ActionHelp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(cfg, tt.keyStr, "")
			if got != tt.wantAction {
				t.Errorf("Resolve(%q, \"\") = %v, want %v", tt.keyStr, got, tt.wantAction)
			}
		})
	}
}
