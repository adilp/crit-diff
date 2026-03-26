package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NoConfigFile(t *testing.T) {
	// When no config file exists, all defaults should apply
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "nonexistent", "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Display defaults
	if cfg.Display.Theme != "auto" {
		t.Errorf("Theme: got %q, want %q", cfg.Display.Theme, "auto")
	}
	if cfg.Display.WordDiff != true {
		t.Error("WordDiff: got false, want true")
	}
	if cfg.Display.Wrap != false {
		t.Error("Wrap: got true, want false")
	}
	if cfg.Display.TabWidth != 4 {
		t.Errorf("TabWidth: got %d, want %d", cfg.Display.TabWidth, 4)
	}
	if cfg.Display.MinWidth != 100 {
		t.Errorf("MinWidth: got %d, want %d", cfg.Display.MinWidth, 100)
	}

	// Key defaults - leader
	if cfg.Keys.Leader != "Space" {
		t.Errorf("Leader: got %q, want %q", cfg.Keys.Leader, "Space")
	}

	// Normal keybindings
	normalDefaults := map[string]string{
		"scroll_down":      "j",
		"scroll_up":        "k",
		"pane_left":        "h",
		"pane_right":       "l",
		"half_page_down":   "Ctrl-d",
		"half_page_up":     "Ctrl-u",
		"top":              "gg",
		"bottom":           "G",
		"next_hunk":        "]c",
		"prev_hunk":        "[c",
		"next_file":        "]f",
		"prev_file":        "[f",
		"next_comment":     "]m",
		"prev_comment":     "[m",
		"expand_below":     "]e",
		"expand_above":     "[e",
		"toggle_tree":      "Tab",
		"fuzzy_files":      "Space f",
		"search_diffs":     "Space s",
		"toggle_word_diff": "w",
		"toggle_wrap":      "z",
		"search":           "/",
		"help":             "?",
		"quit":             "q",
	}
	for action, want := range normalDefaults {
		got, ok := cfg.Keys.Normal[action]
		if !ok {
			t.Errorf("Normal[%q]: missing", action)
			continue
		}
		if got != want {
			t.Errorf("Normal[%q]: got %q, want %q", action, got, want)
		}
	}

	// Comment keybindings
	commentDefaults := map[string]string{
		"add":    "c",
		"visual": "V",
		"edit":   "e",
		"delete": "dc",
	}
	for action, want := range commentDefaults {
		got, ok := cfg.Keys.Comment[action]
		if !ok {
			t.Errorf("Comment[%q]: missing", action)
			continue
		}
		if got != want {
			t.Errorf("Comment[%q]: got %q, want %q", action, got, want)
		}
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[display]
tab_width = 2
theme = "dark"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overridden values
	if cfg.Display.TabWidth != 2 {
		t.Errorf("TabWidth: got %d, want %d", cfg.Display.TabWidth, 2)
	}
	if cfg.Display.Theme != "dark" {
		t.Errorf("Theme: got %q, want %q", cfg.Display.Theme, "dark")
	}

	// Default values still intact
	if cfg.Display.MinWidth != 100 {
		t.Errorf("MinWidth: got %d, want %d", cfg.Display.MinWidth, 100)
	}
	if cfg.Display.WordDiff != true {
		t.Error("WordDiff: got false, want true")
	}

	// Keybindings should all be defaults
	if cfg.Keys.Normal["scroll_down"] != "j" {
		t.Errorf("Normal[scroll_down]: got %q, want %q", cfg.Keys.Normal["scroll_down"], "j")
	}
}

func TestLoad_CustomKeybindings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[keys.normal]
scroll_down = "n"
scroll_up = "p"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overridden
	if cfg.Keys.Normal["scroll_down"] != "n" {
		t.Errorf("Normal[scroll_down]: got %q, want %q", cfg.Keys.Normal["scroll_down"], "n")
	}
	if cfg.Keys.Normal["scroll_up"] != "p" {
		t.Errorf("Normal[scroll_up]: got %q, want %q", cfg.Keys.Normal["scroll_up"], "p")
	}

	// Non-overridden defaults still intact
	if cfg.Keys.Normal["quit"] != "q" {
		t.Errorf("Normal[quit]: got %q, want %q", cfg.Keys.Normal["quit"], "q")
	}
	if cfg.Keys.Normal["next_hunk"] != "]c" {
		t.Errorf("Normal[next_hunk]: got %q, want %q", cfg.Keys.Normal["next_hunk"], "]c")
	}
}

func TestLoad_LeaderKeySubstitution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[keys]
leader = "Ctrl-a"

[keys.normal]
fuzzy_files = "<leader>f"
search_diffs = "<leader>s"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Keys.Leader != "Ctrl-a" {
		t.Errorf("Leader: got %q, want %q", cfg.Keys.Leader, "Ctrl-a")
	}
	if cfg.Keys.Normal["fuzzy_files"] != "Ctrl-a f" {
		t.Errorf("Normal[fuzzy_files]: got %q, want %q", cfg.Keys.Normal["fuzzy_files"], "Ctrl-a f")
	}
	if cfg.Keys.Normal["search_diffs"] != "Ctrl-a s" {
		t.Errorf("Normal[search_diffs]: got %q, want %q", cfg.Keys.Normal["search_diffs"], "Ctrl-a s")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `this is [not valid toml !!!`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v (should warn and use defaults)", err)
	}

	// Should fall back to all defaults
	if cfg.Display.Theme != "auto" {
		t.Errorf("Theme: got %q, want %q (should use defaults on invalid TOML)", cfg.Display.Theme, "auto")
	}
	if cfg.Keys.Normal["scroll_down"] != "j" {
		t.Errorf("Normal[scroll_down]: got %q, want %q", cfg.Keys.Normal["scroll_down"], "j")
	}
}

func TestLoad_FullConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[keys]
leader = "Comma"

[keys.normal]
scroll_down = "n"
scroll_up = "p"
pane_left = "b"
pane_right = "f"
half_page_down = "Ctrl-f"
half_page_up = "Ctrl-b"
top = "gg"
bottom = "G"
next_hunk = "]h"
prev_hunk = "[h"
next_file = "]n"
prev_file = "[n"
next_comment = "]m"
prev_comment = "[m"
expand_below = "]e"
expand_above = "[e"
toggle_tree = "t"
fuzzy_files = "<leader>f"
search_diffs = "<leader>s"
toggle_word_diff = "W"
toggle_wrap = "Z"
search = "/"
help = "?"
quit = "Q"

[keys.comment]
add = "a"
visual = "v"
edit = "E"
delete = "dd"

[display]
theme = "light"
word_diff = true
wrap = true
tab_width = 8
min_width = 120
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Display
	if cfg.Display.Theme != "light" {
		t.Errorf("Theme: got %q, want %q", cfg.Display.Theme, "light")
	}
	if cfg.Display.WordDiff != true {
		t.Error("WordDiff: got false, want true")
	}
	if cfg.Display.Wrap != true {
		t.Error("Wrap: got false, want true")
	}
	if cfg.Display.TabWidth != 8 {
		t.Errorf("TabWidth: got %d, want %d", cfg.Display.TabWidth, 8)
	}
	if cfg.Display.MinWidth != 120 {
		t.Errorf("MinWidth: got %d, want %d", cfg.Display.MinWidth, 120)
	}

	// Leader resolution
	if cfg.Keys.Normal["fuzzy_files"] != "Comma f" {
		t.Errorf("Normal[fuzzy_files]: got %q, want %q", cfg.Keys.Normal["fuzzy_files"], "Comma f")
	}
	if cfg.Keys.Normal["search_diffs"] != "Comma s" {
		t.Errorf("Normal[search_diffs]: got %q, want %q", cfg.Keys.Normal["search_diffs"], "Comma s")
	}

	// Custom keys
	if cfg.Keys.Normal["scroll_down"] != "n" {
		t.Errorf("Normal[scroll_down]: got %q, want %q", cfg.Keys.Normal["scroll_down"], "n")
	}
	if cfg.Keys.Normal["quit"] != "Q" {
		t.Errorf("Normal[quit]: got %q, want %q", cfg.Keys.Normal["quit"], "Q")
	}

	// Comment keys
	if cfg.Keys.Comment["add"] != "a" {
		t.Errorf("Comment[add]: got %q, want %q", cfg.Keys.Comment["add"], "a")
	}
	if cfg.Keys.Comment["delete"] != "dd" {
		t.Errorf("Comment[delete]: got %q, want %q", cfg.Keys.Comment["delete"], "dd")
	}
}

func TestLoad_DefaultColors(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := Load(filepath.Join(dir, "nonexistent.toml"))

	// Verify color defaults exist
	if cfg.Colors.AddBg == "" {
		t.Error("Colors.AddBg should have a default value")
	}
	if cfg.Colors.DeleteBg == "" {
		t.Error("Colors.DeleteBg should have a default value")
	}
	if cfg.Colors.EmphasisAdd == "" {
		t.Error("Colors.EmphasisAdd should have a default value")
	}
	if cfg.Colors.EmphasisDelete == "" {
		t.Error("Colors.EmphasisDelete should have a default value")
	}
	if cfg.Colors.CursorActive == "" {
		t.Error("Colors.CursorActive should have a default value")
	}
}

func TestLoad_CustomColors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[colors]
add_bg = "#003300"
delete_bg = "#330000"
emphasis_add = "#005500"
emphasis_delete = "#550000"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Colors.AddBg != "#003300" {
		t.Errorf("Colors.AddBg: got %q, want %q", cfg.Colors.AddBg, "#003300")
	}
	if cfg.Colors.DeleteBg != "#330000" {
		t.Errorf("Colors.DeleteBg: got %q, want %q", cfg.Colors.DeleteBg, "#330000")
	}
	if cfg.Colors.EmphasisAdd != "#005500" {
		t.Errorf("Colors.EmphasisAdd: got %q, want %q", cfg.Colors.EmphasisAdd, "#005500")
	}
	if cfg.Colors.EmphasisDelete != "#550000" {
		t.Errorf("Colors.EmphasisDelete: got %q, want %q", cfg.Colors.EmphasisDelete, "#550000")
	}

	// Non-overridden colors should retain defaults
	if cfg.Colors.CursorActive == "" {
		t.Error("Colors.CursorActive should retain default")
	}
}

func TestLoad_CustomLeaderResolvesDefaultBindings(t *testing.T) {
	// When only the leader is changed, default <leader> bindings should resolve
	// with the new leader key.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `[keys]
leader = "Ctrl-a"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Keys.Normal["fuzzy_files"] != "Ctrl-a f" {
		t.Errorf("Normal[fuzzy_files]: got %q, want %q", cfg.Keys.Normal["fuzzy_files"], "Ctrl-a f")
	}
	if cfg.Keys.Normal["search_diffs"] != "Ctrl-a s" {
		t.Errorf("Normal[search_diffs]: got %q, want %q", cfg.Keys.Normal["search_diffs"], "Ctrl-a s")
	}
}
