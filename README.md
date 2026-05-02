# pair

A small launcher that gives any TUI coding agent (Claude Code, Codex, Gemini CLI) a real input field — backed by Neovim — and decouples the input scroll from the output scroll.

## What it does

Launches a zellij session split into two panes:

- **Top (~65%)** — the agent. Owns the *output* affordance: streams responses, renders tool calls and diffs.
- **Bottom (~35%)** — Neovim on a persistent draft file. Owns the *input* affordance: full editing power, persistent undo, prompt history.

You compose prompts with full editor power, scroll the agent output independently, and never lose draft text again.

## Keybindings

| Key | Scope | Action |
|---|---|---|
| **Alt+Return** | nvim (normal/insert) | Send buffer to agent, log to history, clear draft, drop into insert mode |
| **Alt+u** | any pane | Toggle the nvim pane to fullscreen (works from either pane) |
| **Alt+n** | any pane | Pull clipboard contents into nvim cursor — paragraph-reflowed and prefixed with `> ` (markdown quote) |
| `<leader>cs` | nvim | Send only the section between `---` markers |
| `<leader>cp` | nvim | Paste-and-reflow at cursor (raw, no quoting) |

## Install

Manual symlink for now — packaging is deferred to v2.

```sh
git clone <repo> ~/workspace/pair
ln -s ~/workspace/pair/bin/pair /usr/local/bin/pair
```

Recommended: install `par` for paragraph reflow (used by `Alt+n` and `<leader>cp`):

```sh
brew install par              # macOS
sudo apt install par          # Debian/Ubuntu
```

If `par` is missing the scripts pass clipboard content through unchanged.

You also need [`zellij`](https://zellij.dev/) and [`nvim`](https://neovim.io/) on your PATH, plus whichever agent you want to drive (`claude`, `codex`, `gemini`).

## Usage

```sh
pair             # default: claude
pair claude
pair codex
pair gemini
```

Each agent runs in its own zellij session (`pair-claude`, `pair-codex`, ...), so multiple can run simultaneously without conflicting.

To detach without closing the session: `Ctrl+p d` (zellij default). Re-attach by running `pair <agent>` again.

## Image paste

`Ctrl+V` in the agent pane reads the OS clipboard directly. Put an image on the clipboard first:

**macOS:**
```sh
osascript -e 'set the clipboard to (read (POSIX file "/path/to/img.png") as «class PNGf»)'
```

**Linux (Wayland):**
```sh
wl-copy --type image/png < /path/to/img.png
```

**Linux (X11):**
```sh
xclip -selection clipboard -t image/png -i /path/to/img.png
```

A simpler path that skips the clipboard entirely: type `@/abs/path/to/img.png` into the draft. Most TUI agents recognize that as a file attachment.

## Files

```
bin/pair                     # launcher
bin/clipboard-to-pane.sh     # helper for Alt+n
nvim/init.lua                # bundled nvim config (loaded via -u)
zellij/config.kdl            # zellij keybinds for Alt+u and Alt+n
zellij/layouts/main.kdl      # the 65/35 split
```

Drafts and prompt history live in `~/scratch/`:

- `pair-draft-<agent>.md` — the active draft (cleared on send, persists across launches)
- `pair-log-<agent>.md` — appended on every send with timestamp; your grep-able prompt history

## Design notes

See [the pensive that motivated this](../brain/docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md) (sibling repo).

Highlights:

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit and lets each side specialize.
- **Self-contained, doesn't touch your config.** v1 uses `zellij --config-dir` and `nvim -u` to fully isolate from your normal `~/.config/{zellij,nvim}`. Try it without commitment.
- **Prompt history is just a markdown file.** Every send appends to `~/scratch/pair-log-<agent>.md`. Grep, diff, copy from. Your conversations are searchable forever.
- **Agent-agnostic.** The same zellij+nvim plumbing works for any TUI agent. Switching from `pair claude` to `pair codex` is one keystroke.
