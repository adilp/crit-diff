package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds all cr configuration.
type Config struct {
	Keys    KeysConfig
	Display DisplayConfig
}

// KeysConfig holds keybinding configuration.
type KeysConfig struct {
	Leader  string
	Normal  map[string]string
	Comment map[string]string
}

// DisplayConfig holds display preferences.
type DisplayConfig struct {
	Theme    string
	WordDiff bool
	Wrap     bool
	TabWidth int
	MinWidth int
}

// tomlConfig mirrors the TOML file structure for parsing.
type tomlConfig struct {
	Keys    tomlKeysConfig    `toml:"keys"`
	Display tomlDisplayConfig `toml:"display"`
}

type tomlKeysConfig struct {
	Leader  string            `toml:"leader"`
	Normal  map[string]string `toml:"normal"`
	Comment map[string]string `toml:"comment"`
}

type tomlDisplayConfig struct {
	Theme    *string `toml:"theme"`
	WordDiff *bool   `toml:"word_diff"`
	Wrap     *bool   `toml:"wrap"`
	TabWidth *int    `toml:"tab_width"`
	MinWidth *int    `toml:"min_width"`
}

func defaultConfig() Config {
	return Config{
		Keys: KeysConfig{
			Leader: "Space",
			Normal: map[string]string{
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
				"fuzzy_files":      "<leader>f",
				"search_diffs":     "<leader>s",
				"toggle_word_diff": "w",
				"toggle_wrap":      "z",
				"search":           "/",
				"help":             "?",
				"quit":             "q",
			},
			Comment: map[string]string{
				"add":    "c",
				"visual": "V",
				"edit":   "e",
				"delete": "dc",
			},
		},
		Display: DisplayConfig{
			Theme:    "auto",
			WordDiff: false,
			Wrap:     false,
			TabWidth: 4,
			MinWidth: 100,
		},
	}
}

// Load reads configuration from the given path, merging with defaults.
// If the file doesn't exist, defaults are returned.
// If the file has a parse error, a warning is printed to stderr and defaults are returned.
func Load(path string) (Config, error) {
	cfg := defaultConfig()
	defer resolveLeader(&cfg)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		fmt.Fprintf(os.Stderr, "warning: failed to read %s: %v (using defaults)\n", path, err)
		return cfg, nil
	}

	var parsed tomlConfig
	if err := toml.Unmarshal(data, &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to parse %s: %v (using defaults)\n", path, err)
		return cfg, nil
	}

	// Merge keys
	if parsed.Keys.Leader != "" {
		cfg.Keys.Leader = parsed.Keys.Leader
	}
	for action, key := range parsed.Keys.Normal {
		cfg.Keys.Normal[action] = key
	}
	for action, key := range parsed.Keys.Comment {
		cfg.Keys.Comment[action] = key
	}

	// Merge display
	if parsed.Display.Theme != nil {
		cfg.Display.Theme = *parsed.Display.Theme
	}
	if parsed.Display.WordDiff != nil {
		cfg.Display.WordDiff = *parsed.Display.WordDiff
	}
	if parsed.Display.Wrap != nil {
		cfg.Display.Wrap = *parsed.Display.Wrap
	}
	if parsed.Display.TabWidth != nil {
		cfg.Display.TabWidth = *parsed.Display.TabWidth
	}
	if parsed.Display.MinWidth != nil {
		cfg.Display.MinWidth = *parsed.Display.MinWidth
	}

	return cfg, nil
}

// resolveLeader replaces "<leader>X" patterns with the actual leader key + " " + X.
func resolveLeader(cfg *Config) {
	for action, key := range cfg.Keys.Normal {
		if strings.HasPrefix(key, "<leader>") {
			suffix := strings.TrimPrefix(key, "<leader>")
			cfg.Keys.Normal[action] = cfg.Keys.Leader + " " + suffix
		}
	}
	for action, key := range cfg.Keys.Comment {
		if strings.HasPrefix(key, "<leader>") {
			suffix := strings.TrimPrefix(key, "<leader>")
			cfg.Keys.Comment[action] = cfg.Keys.Leader + " " + suffix
		}
	}
}
