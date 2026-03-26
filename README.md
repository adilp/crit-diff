# cr

A full-screen, side-by-side diff viewer with vim-native navigation that writes comments to [crit](https://github.com/kevindutra/crit)-compatible `.crit/` format.

`cr` is the *lens* — crit is the *notebook*. Review diffs in a GitHub-style TUI, leave inline comments, and let Claude Code address your feedback automatically.

## Install

### Claude Code Plugin Marketplace (recommended)

cr is available as a Claude Code plugin. Add the marketplace and install:

```
/plugin marketplace add adil/cr
/plugin install cr
```

Then use `/cr:review <ref>` to open the TUI. After you close it, Claude reads your comments and edits the code to address them.

### From source

```bash
go install github.com/adil/cr@latest
```

Make sure `$GOPATH/bin` (defaults to `~/go/bin`) is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Manual skill install

If you prefer not to use the plugin, you can install the Claude Code skill directly:

```bash
cr setup-claude            # Install globally (~/.claude/skills/)
cr setup-claude --project  # Install for current project only
cr setup-claude --force    # Overwrite existing installation
```

Then use `/cr-review <ref>` in Claude Code.

## Requirements

- **Go 1.21+** for building from source
- **git** — cr is a git tool, it reads diffs from your repo
- **tmux** — required for the Claude Code integration. cr opens the review TUI in a tmux split pane next to Claude Code.
- **fzf** (optional) — for fuzzy file/content search. Falls back to a built-in filter if not installed.

### Starting a tmux session

If you're not already in tmux, start one before launching Claude Code:

```bash
tmux new -s work
# Now launch Claude Code inside this tmux session
claude
```

If you forget, cr will tell you — but the split-pane review won't work outside of tmux.

## Usage

```bash
# Review working tree changes (staged + unstaged)
cr

# Review changes against a branch
cr main

# Review last N commits
cr -1
cr -3

# Review a commit range
cr HEAD~5..HEAD
cr v1.0..v1.1

# Review only staged changes
cr --staged

# Filter to specific paths
cr main -- src/services/

# Output review comments as JSON
cr status
```

## Claude Code Integration

```
/cr-review main            # review changes against main
/cr-review -3              # review last 3 commits
/cr-review --staged        # review staged changes
```

**Workflow:**

1. Claude opens cr in a tmux pane alongside your conversation
2. You review the diff side-by-side, leave comments with `c`, quit with `q`
3. Claude reads your comments via `cr status` and addresses them automatically
4. Claude offers to re-open cr for you to verify the changes

### tmux split pane mode

When running inside tmux, you can open the TUI in a side-by-side split pane:

```bash
# Open review in a tmux split and return immediately
cr --detach main

# Open review in a tmux split and block until it closes
cr --detach --wait main
```

This is how the Claude Code skill invokes cr — `--detach --wait` is a single blocking call that opens the TUI next to Claude Code and waits for you to finish reviewing.

## Keybindings

All keybindings are configurable via `~/.config/cr/config.toml`.

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll down / up |
| `h` / `l` | Switch to left (old) / right (new) pane |
| `Ctrl-d` / `Ctrl-u` | Half page down / up |
| `gg` / `G` | Jump to top / bottom of file |
| `]c` / `[c` | Next / prev hunk |
| `]f` / `[f` | Next / prev file |
| `]m` / `[m` | Next / prev comment |
| `]e` / `[e` | Expand context below / above (10 lines) |

### File switching

| Key | Action |
|-----|--------|
| `Tab` | Toggle file tree sidebar |
| `Space f` | Fuzzy file search |
| `Space s` | Fuzzy content search |

### Comments

| Key | Action |
|-----|--------|
| `c` | Comment on current line |
| `V` | Visual select mode (then `j`/`k` to extend, `c` to comment on range) |
| `e` | Edit comment (cursor must be on comment row) |
| `dc` | Delete comment (cursor must be on comment row) |

### View

| Key | Action |
|-----|--------|
| `w` | Toggle word-level diff highlighting |
| `z` | Toggle line wrapping |
| `/` | Search within current file |
| `?` | Help overlay |
| `q` | Quit (comments are auto-saved) |

## How It Works

```
git repo ──git diff──> cr (TUI) ──.crit/ YAML──> crit / Claude
```

1. `cr <ref>` parses the git diff and renders it side-by-side with syntax highlighting
2. You navigate with vim motions and leave inline comments
3. Comments are auto-saved to `.crit/reviews/` as YAML — the exact same format crit uses
4. `cr status` outputs all comments as JSON for Claude (or any tool) to consume
5. Claude reads the comments, edits your code, and you can re-review with `cr`

### Interop with crit

cr and crit share the same `.crit/` storage format:

- Comments left in cr can be read by `crit status <file>`
- Comments left in crit can be seen in cr
- The session manifest at `.crit/code-review.yaml` is readable by both tools
- You can use cr for visual diff review and crit for document review — they don't conflict

## Configuration

`~/.config/cr/config.toml` — only override what you need:

```toml
[keys]
leader = "Space"

[keys.normal]
scroll_down = "j"
scroll_up = "k"
quit = "q"
# ... see cr --help or press ? in the TUI for all bindings

[display]
theme = "auto"       # auto | dark | light
word_diff = true     # word-level diff on by default
wrap = false         # line wrapping off by default
tab_width = 4
min_width = 100      # below this, show "too narrow" message

[colors]
add_bg = "#0d2818"           # line background for added lines
delete_bg = "#2d1117"        # line background for deleted lines
emphasis_add = "#1a4d2e"     # word-level add highlight
emphasis_delete = "#5c1a1a"  # word-level delete highlight
cursor_active = "237"        # cursor on active pane
cursor_inactive = "235"      # cursor on inactive pane
```

## Development

```bash
go test ./...
go build ./...
go vet ./...
```

## License

MIT
