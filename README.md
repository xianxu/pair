# pair

A small launcher that gives any TUI coding agent (Claude Code, Codex, Gemini CLI) a real input field — backed by Neovim — and decouples the input scroll from the output scroll.

## What it does

Launches a zellij session split into two panes:

- **Top** — the agent. Owns the *output* affordance: streams responses, renders tool calls and diffs. Fills the rest of the screen.
- **Bottom (10 rows)** — Neovim on a persistent draft file. Owns the *input* affordance: full editing power, persistent undo, prompt history. Toggle to fullscreen with `Alt+u` when you need more room.

You compose prompts with full editor power, scroll the agent output independently, and never lose draft text again.

## Keybindings

| Key | Scope | Action |
|---|---|---|
| **Alt+Return** | nvim (normal/insert) | Send buffer to agent + Enter, log to history, clear draft, drop into insert mode |
| **Alt+u** | any pane | Toggle the nvim pane to fullscreen (works from either pane) |
| **Alt+d** | any pane | Detach from the current session |
| **Alt+n** | any pane | Pull clipboard contents into nvim cursor — paragraph-reflowed and prefixed with `> ` (markdown quote) |
| **Alt+i** | nvim (normal/insert) | Attach clipboard image to the agent and insert `[Image #N]` reference at cursor. If cursor is on an existing `[Image #N]`, sync the internal counter to N (manual-correction path). |
| `<leader>cs` | nvim | Send only the section between `---` markers |
| `<leader>cp` | nvim | Paste-and-reflow at cursor (raw, no quoting) |

## Dependencies

**Required**

| Tool | Purpose |
|---|---|
| [`zellij`](https://zellij.dev/) | terminal multiplexer; pair runs as a zellij session |
| [`nvim`](https://neovim.io/) | the input/drafting pane |
| [`fzf`](https://github.com/junegunn/fzf) | session picker (falls back to a numbered prompt if missing) |
| an agent | `claude`, `codex`, `gemini`, or any TUI agent you want to drive |

**Optional**

| Tool | Purpose |
|---|---|
| [`par`](https://www.nicemice.net/par/) | paragraph reflow for `Alt+n` and `<leader>cp` (clipboard content passes through unchanged if missing) |

macOS install:
```sh
brew install zellij neovim fzf par
```

Debian/Ubuntu:
```sh
sudo apt install neovim fzf par
# zellij: see https://zellij.dev/documentation/installation.html
```

## Install

Manual symlink for now — packaging is deferred to v2.

```sh
git clone <repo> ~/workspace/pair
ln -s ~/workspace/pair/bin/pair /usr/local/bin/pair
```

## Usage

```sh
pair                          # default: claude
pair <agent>                  # claude / codex / gemini
pair <agent> <variant>        # independent session, e.g. `pair claude work`
pair -h, --help               # show full help
```

When `pair` runs and any detached `pair-*` session exists, it shows an `fzf` picker over the detached sessions plus a `+ new <agent> session` sentinel. Picking attaches; picking the sentinel falls through to the create flow. **No silent auto-attach** — every reattach is explicit.

When the create flow runs, it prompts for the session name with the auto-suggested name as the default:

```
Session name [pair-claude]: <Enter to accept, or type a custom name>
```

Custom names like `bugfix`, `pair-blogging`, or `claude-research` are allowed (chars: `A-Z a-z 0-9 - _`). Sessions named without an agent prefix (`pair-blogging`) live outside any agent family and reappear under the picker on any future `pair` run.

To detach mid-session: `Alt+d`. To re-attach: run `pair` again and pick from the list.

## Image paste

`Alt+i` is the integrated path: put an image on the OS clipboard first, then press `Alt+i` from inside nvim. `pair` types `Ctrl+V` into the agent pane (so claude attaches the image as a chip) *and* inserts a `[Image #N]` reference at your nvim cursor. If the local counter drifts from claude's actual count, edit the number in nvim and press `Alt+i` while on the corrected token to resync.

Putting an image on the clipboard:

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
bin/clipboard-to-pane.sh     # helper for Alt+n (clipboard → nvim cursor as quote)
nvim/init.lua                # bundled nvim config (loaded via `nvim -u`)
zellij/config.kdl            # zellij keybinds: Alt+u, Alt+d, Alt+n, unbind Alt+i
zellij/layouts/main.kdl      # the split (agent on top, nvim 10 rows on bottom)
```

Drafts and prompt history live in `~/scratch/`, keyed by *tag* (the agent name, or `<agent>-<variant>`, or your custom session name):

- `pair-draft-<tag>.md` — the active draft (cleared on send, persists across launches)
- `pair-log-<tag>.md` — appended on every send with timestamp; your grep-able prompt history

## Design notes

See [the pensive that motivated this](../brain/docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md) (sibling repo) and `atlas/architecture.md` for the architecture map.

Highlights:

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit and lets each side specialize.
- **Self-contained, doesn't touch your config.** v1 uses `zellij --config-dir` and `nvim -u` to fully isolate from your normal `~/.config/{zellij,nvim}`. Try it without commitment.
- **Prompt history is just a markdown file.** Every send appends to `~/scratch/pair-log-<tag>.md`. Grep, diff, copy from. Your conversations are searchable forever.
- **Explicit reattach.** The picker fires whenever any detached session exists, so you always see what you're connecting to. Long-lived agent sessions make silent attach surprising.
- **Agent-agnostic.** The same zellij+nvim plumbing works for any TUI agent. Switching from `pair claude` to `pair codex` is one keystroke.
