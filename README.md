# crit-diff

A full-screen, side-by-side diff viewer with vim-native navigation. A companion to [crit](https://github.com/kevindutra/crit) — the excellent terminal review tool for markdown documents.

Where crit handles document review (plans, specs, markdown), cr handles **git diff review** (code changes across files) with a GitHub-style side-by-side TUI. Both tools share the same `.crit/` comment format, so they work together seamlessly as part of the same review ecosystem.

We recommend installing both for the full workflow — crit for reviewing what Claude writes, cr for reviewing what Claude changes.

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
go install github.com/adilp/crit-diff@latest
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

## Better Together: cr + crit

cr was built to complement [crit](https://github.com/kevindutra/crit), not replace it. They cover different parts of the review workflow and share the same `.crit/` storage format:

| Tool | Reviews | Think of it as |
|------|---------|----------------|
| **crit** | Markdown documents (plans, specs, designs) | "Review what Claude writes" |
| **cr** | Git diffs (code changes across files) | "Review what Claude changes" |

### Shared storage

Both tools read and write to `.crit/reviews/` using the same YAML schema:
- Comments left in cr show up in `crit status <file>`
- Comments left in crit are visible in cr
- The session manifest at `.crit/code-review.yaml` is readable by both

### The full ecosystem

```bash
# Install both
go install github.com/kevindutra/crit/cmd/crit@latest
go install github.com/adilp/crit-diff@latest

# Claude writes a plan → review the document with crit
crit review docs/plan.md

# Claude implements the plan → review the code changes with cr
cr main

# Both leave comments in .crit/ → Claude addresses them
claude "address the comments in .crit/"
```

### In Claude Code

```
/crit:review docs/plan.md   → crit opens to review the document
/cr-review main              → cr opens to review the diff
```

Both skills follow the same pattern: open TUI in a tmux pane, user reviews and comments, Claude reads the comments and acts on them.

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
