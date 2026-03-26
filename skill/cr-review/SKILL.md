---
name: cr-review
description: Launch cr's side-by-side diff review TUI. User reviews code changes, leaves comments in .crit/ format, then Claude addresses the feedback. Works standalone or alongside crit.
allowed-tools: Bash(cr *), Bash(crit *), Bash(tmux *), Read, Edit, Grep, Glob
argument-hint: <ref> (e.g., main, HEAD~3, a1b2..c3d4, -1, -3, --staged)
---

# Code Review with cr

Launch cr's side-by-side diff TUI for the user to review code changes and leave comments. Comments are stored in `.crit/` YAML format (compatible with crit).

## Step 1: Launch the TUI

Determine the ref argument from `$ARGUMENTS`. If empty, default to reviewing working tree changes.

Check if `$TMUX` is set:

**If in tmux**, run with a **timeout of 600000** (10 minutes) since it blocks until the user finishes:
```bash
cr --detach --wait $ARGUMENTS
```

If that fails (e.g., tmux error), fall back to opening the pane manually and blocking with wait-for:
```bash
tmux split-window -h "cr $ARGUMENTS ; tmux wait-for -S cr-review-done"
```
Then block until the user quits:
```bash
tmux wait-for cr-review-done
```

**If not in tmux**, ask the user to run it manually:

> Please run this in your terminal, review the diff, leave comments, and let me know when you're done:
>
> ```
> cr $ARGUMENTS
> ```

Wait for the user to confirm before proceeding.

## Step 2: Read the comments

After the review is complete, read all comments:

```bash
cr status
```

This outputs JSON with all comments across all files in the review session:
```json
[
  {
    "file": "src/auth.go",
    "id": "a1b2c3d4",
    "line": 42,
    "content_snippet": "if token == \"\" {",
    "body": "Should validate token format, not just emptiness"
  }
]
```

If the output is `[]` (no comments), tell the user: "No comments found. The code looks good!" and stop.

## Step 3: Address comments

For each comment in the JSON array:

1. Read the file at the `file` path
2. Use `line` and `content_snippet` to locate the exact code
3. Read the `body` for what the reviewer wants changed
4. Edit the file to address the comment

After addressing ALL comments, summarize what you changed.

## Step 4: Re-review (optional)

After making changes, ask the user:

> "I've addressed all N comments. Want to review the changes? I'll open cr again."

If yes, go back to Step 1 (reviewing the working tree changes this time). If no, done.

## Key bindings reference (for helping the user)

| Key | Action |
|-----|--------|
| `j/k` | Scroll up/down |
| `h/l` | Switch left/right pane |
| `]f/[f` | Next/prev file |
| `c` | Comment on current line |
| `V` then `c` | Comment on selected range |
| `e` | Edit comment (on comment row) |
| `dc` | Delete comment (on comment row) |
| `/` | Search in file |
| `Tab` | Toggle file tree |
| `w` | Toggle word-level diff |
| `q` | Quit (comments auto-saved) |

## Important notes

- Comments are auto-saved to `.crit/` on every add/edit/delete — no save step needed
- `cr` writes comments in crit's exact YAML format — `crit status` can also read them
- Do NOT modify code while the TUI is open — only edit after it exits
- The `content_snippet` field helps locate code even if line numbers shift
