package ui

import (
	"strings"
	"testing"

	"github.com/adil/cr/internal/config"
	"github.com/adil/cr/internal/diff"
	tea "github.com/charmbracelet/bubbletea"
)

func TestBuildHelpEntries(t *testing.T) {
	cfg, _ := config.Load("")

	entries := BuildHelpEntries(cfg)

	// Must have 4 categories
	categories := make(map[string]bool)
	for _, e := range entries {
		if e.Category != "" {
			categories[e.Category] = true
		}
	}
	wantCategories := []string{"Navigation", "File Switching", "Comments", "View"}
	for _, cat := range wantCategories {
		if !categories[cat] {
			t.Errorf("missing category %q", cat)
		}
	}

	// Must include j/k for scroll
	found := false
	for _, e := range entries {
		if strings.Contains(e.Key, "j") && strings.Contains(e.Key, "k") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected j/k scroll entry")
	}
}

func TestBuildHelpEntriesCustomKeys(t *testing.T) {
	cfg, _ := config.Load("")
	cfg.Keys.Normal["scroll_down"] = "n"
	cfg.Keys.Normal["scroll_up"] = "p"

	entries := BuildHelpEntries(cfg)

	found := false
	for _, e := range entries {
		if strings.Contains(e.Key, "n") && strings.Contains(e.Key, "p") && strings.Contains(e.Action, "scroll") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected custom n/p scroll entry in help")
	}
}

func TestRenderHelpOverlay(t *testing.T) {
	cfg, _ := config.Load("")
	entries := BuildHelpEntries(cfg)
	output := RenderHelpOverlay(entries, 120, 40, 0)

	if !strings.Contains(output, "Help") {
		t.Error("overlay should contain Help title")
	}
	if !strings.Contains(output, "Navigation") {
		t.Error("overlay should contain Navigation category")
	}
	if !strings.Contains(output, "quit") {
		t.Error("overlay should contain quit action")
	}
}

func TestRenderHelpOverlayScrollable(t *testing.T) {
	cfg, _ := config.Load("")
	entries := BuildHelpEntries(cfg)
	// Very short terminal — overlay should not crash
	output := RenderHelpOverlay(entries, 80, 5, 0)
	if output == "" {
		t.Error("overlay should render even with tiny terminal")
	}
}

func TestHelpOverlayQuestionMarkOpens(t *testing.T) {
	pairs := buildTestPairs()
	m := NewModel(
		[]diff.DiffFile{{OldName: "test.go", NewName: "test.go"}},
		pairs, 120, 40,
	)

	updated, _ := m.Update(keyMsg("?"))
	result := updated.(Model)

	if result.mode != InputModeHelp {
		t.Errorf("expected InputModeHelp, got %v", result.mode)
	}
}

func TestHelpOverlayEscCloses(t *testing.T) {
	pairs := buildTestPairs()
	m := NewModel(
		[]diff.DiffFile{{OldName: "test.go", NewName: "test.go"}},
		pairs, 120, 40,
	)

	// Open help
	updated, _ := m.Update(keyMsg("?"))
	result := updated.(Model)
	// Close with Esc
	updated, _ = result.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result = updated.(Model)

	if result.mode != InputModeNormal {
		t.Errorf("expected InputModeNormal after Esc, got %v", result.mode)
	}
}

func TestHelpOverlayQuestionMarkCloses(t *testing.T) {
	pairs := buildTestPairs()
	m := NewModel(
		[]diff.DiffFile{{OldName: "test.go", NewName: "test.go"}},
		pairs, 120, 40,
	)

	// Open help
	updated, _ := m.Update(keyMsg("?"))
	result := updated.(Model)
	// Close with ?
	updated, _ = result.Update(keyMsg("?"))
	result = updated.(Model)

	if result.mode != InputModeNormal {
		t.Errorf("expected InputModeNormal after second ?, got %v", result.mode)
	}
}

func TestHelpOverlayHelpBarHint(t *testing.T) {
	output := RenderHelpBar(120, InputModeHelp)
	if !strings.Contains(output, "Esc") {
		t.Error("help mode help bar should mention Esc")
	}
	if !strings.Contains(output, "?") {
		t.Error("help mode help bar should mention ?")
	}
	if !strings.Contains(output, "close") {
		t.Error("help mode help bar should mention close")
	}
}

func TestHelpOverlayRendersInView(t *testing.T) {
	pairs := buildTestPairs()
	m := NewModel(
		[]diff.DiffFile{{OldName: "test.go", NewName: "test.go"}},
		pairs, 120, 40,
	)

	// Open help
	updated, _ := m.Update(keyMsg("?"))
	result := updated.(Model)
	view := result.View()

	if !strings.Contains(view, "Help") {
		t.Error("View should contain Help overlay when in help mode")
	}
	if !strings.Contains(view, "Navigation") {
		t.Error("View should contain Navigation category in help overlay")
	}
}

func TestHelpOverlayScrollJK(t *testing.T) {
	pairs := buildTestPairs()
	m := NewModel(
		[]diff.DiffFile{{OldName: "test.go", NewName: "test.go"}},
		pairs, 120, 10, // short terminal to need scrolling
	)

	// Open help
	updated, _ := m.Update(keyMsg("?"))
	result := updated.(Model)

	// j should scroll (not change mode)
	updated, _ = result.Update(keyMsg("j"))
	result = updated.(Model)
	if result.mode != InputModeHelp {
		t.Errorf("j in help mode should stay in help, got %v", result.mode)
	}
	if result.helpYOffset != 1 {
		t.Errorf("expected helpYOffset=1 after j, got %d", result.helpYOffset)
	}

	// k should scroll back
	updated, _ = result.Update(keyMsg("k"))
	result = updated.(Model)
	if result.helpYOffset != 0 {
		t.Errorf("expected helpYOffset=0 after k, got %d", result.helpYOffset)
	}

	// k at 0 should stay at 0
	updated, _ = result.Update(keyMsg("k"))
	result = updated.(Model)
	if result.helpYOffset != 0 {
		t.Errorf("expected helpYOffset=0 after k at top, got %d", result.helpYOffset)
	}
}

func TestHelpOverlayQuitStillWorks(t *testing.T) {
	pairs := buildTestPairs()
	m := NewModel(
		[]diff.DiffFile{{OldName: "test.go", NewName: "test.go"}},
		pairs, 120, 40,
	)

	// Open help
	updated, _ := m.Update(keyMsg("?"))
	result := updated.(Model)

	// q should close help (not quit the app) — or quit, depending on design
	// Per ticket: Esc or ? closes. Other keys ignored or used for scrolling.
	// q should NOT quit the app while help is open
	updated, cmd := result.Update(keyMsg("q"))
	result = updated.(Model)
	if cmd != nil {
		// If cmd is tea.Quit, q quit the app — that's wrong in help mode
		t.Error("q in help mode should not quit the app")
	}
}
