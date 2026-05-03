# pair

A small launcher that gives any TUI coding agent (Claude Code, Codex, Gemini CLI) a real input field — backed by Neovim. `pair` glues several tools together: `zellij`, `nvim`, `fzf`, `par`. 

## What it does

Launches a zellij session split into two panes:

- **Top** — the coding agent, `Claude`, `Codex`, `Gemini`, ...
- **Bottom (~8 rows)** — Neovim on a persistent draft file. 

You compose prompts with full editor power, scroll the agent output independently. When you are done, `Alt+Return` to send your text to the agent.

Works on Mac, probably on Linux as well, as it's just bash.

## Keybindings

| Key | Scope | Action |
|---|---|---|
| **Alt+Return** | nvim (normal/insert) | Send buffer to agent |
| **Alt+u** | any pane | Toggle the nvim pane to fullscreen (works from either pane) |
| **Alt+i** | nvim (normal/insert) | Attach clipboard image to the agent and insert `[Image #N]` reference at cursor. |
| **Alt+i** | when inside [Image tag] | Sync the internal counter to N (manual-correction path), allowing user to edit if the cursor between nvim and agent gets out of sync. |
| **Alt+d** | any pane | Detach from the current session (re-attach later via `pair`) |
| **Alt+x** | any pane | Full quit — kill the session AND all processes running inside. |

## Mouse

- **Click-and-drag in any pane** → starts selecting immediately. 
- **Release mouse to finish the selection** → the selection is inserted as quotes into `nvim` with format `> and selected text`.

## Dependencies

**Required**

Automatically installed with `homebrew`.

| Tool | Purpose |
|---|---|
| [`zellij`](https://zellij.dev/) | terminal multiplexer; `pair` runs as a zellij session |
| [`nvim`](https://neovim.io/) | the input/drafting pane |
| [`fzf`](https://github.com/junegunn/fzf) | session picker (falls back to a numbered prompt if missing) |
| [`jq`](https://jqlang.github.io/jq/) | parses `zellij action list-panes --json` to target the nvim pane explicitly |
| [`par`](https://www.nicemice.net/par/) | paragraph reflow on copy-from-agent (without it, lines stay wrapped at terminal width) |
| an agent | `claude`, `codex`, `gemini`, or any TUI agent you want to drive |

## Install

**Homebrew (recommended).** 

```sh
brew tap xianxu/pair
brew install pair
```

That installs `zellij`, `neovim`, `fzf`, `jq`, and `par` if they aren't already present. The agent (`claude`, `codex`, `gemini`) you install separately. Then:

## Usage

```sh
pair                             # default: claude
pair <agent>                     # claude / codex / gemini
pair [<agent>] -- <args...>      # forward args to agent on create
                                 # e.g. pair claude -- --resume
                                 #      pair -- --dangerously-skip-permissions
                                 #      pair codex -- -p "say hi"
pair -h, --help                  # show full help
```

Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.

Agent args (after `--`) are appended to the agent command line on **create**. Reattaching to an existing session does not re-launch the agent, so the args don't apply on attach. (The picker connects you to whatever's already running.)

Caveat: agent args are word-split by the shell, so quoted args containing spaces (`pair claude -- --system "hello world"`) get split into multiple args. Most CLI flags don't have spaces, but worth knowing.

When `pair` runs and any detached `pair-*` session exists, it shows an `fzf` picker over the detached sessions plus a `+ new <agent> session` sentinel.

When the create flow runs, it prompts for the session name with the auto-suggested name as the default:

```
Session name [pair-claude]: <Enter to accept, or type a custom name>
```

Custom names like `bugfix`, `blogging`, or `research` are allowed (chars: `A-Z a-z 0-9 - _`). 

To detach mid-session: `Alt+d`. To re-attach: run `pair` again and pick from the list. To fully quit (no resurrect entry): `Alt+x`.

## Image paste

`Alt+i` is the integrated path: put an image on the OS clipboard first, then press `Alt+i` from inside nvim. `pair` types `Ctrl+V` into the agent pane (so claude attaches the image as a chip) *and* inserts a `[Image #N]` reference at your nvim cursor. If the local counter drifts from claude's actual count, edit the number in nvim and press `Alt+i` while on the corrected token to resync.

Caveat: the reference [Image #N] doesn't work with Genimi. Though user can manually paste the reference from Gemini over.

## Files

```
bin/pair                     # launcher
bin/clipboard-to-pane.sh     # reflow + > -prefix + write into nvim pane
bin/copy-on-select.sh        # zellij copy_command — wraps clipboard-to-pane on mouse-up
bin/pair-quit.sh             # Alt+x — marks session for cleanup, kills it
nvim/init.lua                # bundled nvim config (loaded via `nvim -u`)
zellij/config.kdl            # mouse_click_through, copy_command, keybinds (Alt+u/d/i/x)
zellij/layouts/main.kdl      # the split (agent on top, nvim 10 rows on bottom)
```

Drafts and prompt history live under `${XDG_DATA_HOME:-~/.local/share}/pair/`, keyed by *tag* (the agent name, or your custom session name):

- `draft-<tag>.md` — the active draft (cleared on send, persists across launches)
- `log-<tag>.md` — appended on every send with timestamp; your grep-able prompt history

Quit-marker file (used internally by Alt+x): `~/.cache/pair/quit-<session>` — touched by `pair-quit.sh`, removed by `bin/pair` when it cleans up the session.

## Design notes

See [the pensive that motivated this](../brain/docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md) (sibling repo) and `atlas/architecture.md` for the architecture map.

Highlights:

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit and lets each side specialize.
- **Self-contained, doesn't touch your config.** Uses `zellij --config-dir` and `nvim -u` to fully isolate from your normal `~/.config/{zellij,nvim}`. Try it without commitment.
- **Prompt history is just a markdown file.** Every send appends to `~/scratch/pair-log-<tag>.md`. Grep, diff, copy from. Your conversations are searchable forever.
- **Explicit reattach.** The picker fires whenever any detached session exists, so you always see what you're connecting to. Long-lived agent sessions make silent attach surprising.
- **Two quit semantics.** Ctrl+q keeps the session as a resurrect candidate (zellij default). Alt+x is full quit — session is gone. Both are useful.
- **Selection is the gesture.** Click-and-drag in the agent pane, mouse up — the quote is in nvim, ready for your reaction. No keystroke between.
- **Agent-agnostic.** The same zellij+nvim plumbing works for any TUI agent. Switching from `pair claude` to `pair codex` is one keystroke.
