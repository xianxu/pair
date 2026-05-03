# pair

A small launcher that gives any TUI coding agent (Claude Code, Codex, Gemini CLI) a real input field — backed by Neovim. `pair` glues several tools together: `zellij`, `nvim`, `fzf`, `par`. 

All [AI created](https://xianxu.dev/2026/05/a-saturday-coding-session/).

## What it does

Launches a zellij session split into two panes:

- **Top** — the coding agent, `Claude`, `Codex`, `Gemini`, ...
- **Bottom (~10 rows)** — Neovim on a persistent draft file. 

You compose prompts with full editor power, scroll the agent output independently. When you are done, `Alt+Return` to send your text to the agent.

Works on Mac, probably on Linux as well, as it's just bash.

## What do you get

Full nvim support, for example:

1. mouse support
2. search in the input box
3. typeahead and search local file path, just type `./` and then continue. 

Easy to access prompt history: Alt+<- Alt+-> to navigate. You can also queue up command with Alt+q for quick ideas but not to interrupt current working of an agent, like a stack of command.

## Keybindings

| Key | Scope | Action |
|---|---|---|
| **Alt+Return** | nvim (normal/insert) | Send buffer to agent |
| **Alt+←** / **Alt+→** | nvim (normal/insert) | Walk through prompt history (`-N`) and queued prompts (`+N`) one slot at a time. Status line shows `H < pos > Q`. |
| **Shift+Alt+←** / **Shift+Alt+→** | nvim (normal/insert) | Jump to the next region boundary: oldest-history, newest-history, `*`, front-of-queue, back-of-queue. Lets you skip over long histories or queues quickly. |
| **Alt+q** | nvim (normal/insert) | Push current buffer to the front of the queue (`+1`). From `*` clears the draft; from `+N` it's move-to-front. |
| **Alt+Backspace** | nvim (normal/insert), at `+N` | Delete the current queued prompt without sending. Items behind shift down so you can delete a run by tapping repeatedly. |
| **Alt+u** | any pane | Toggle the nvim pane to fullscreen (works from either pane) |
| **Alt+i** | nvim (normal/insert) | Attach clipboard image to the agent and insert `[Image #N]` reference at cursor. |
| **Alt+i** | when inside [Image tag] | Sync the internal counter to N (manual-correction path), allowing user to edit if the cursor between nvim and agent gets out of sync. |
| **Alt+d** | any pane | Detach from the current session (re-attach later via `pair`) |
| **Alt+x** | any pane | Full quit — kill the session AND all processes running inside. |

### Prompt history & queue

The nvim pane is a virtual cursor over `[ ... -2 -1 ] * [ +1 +2 ... ]`. The status line shows `H < pos > Q` (history count, current position, queue count). `Alt+←` walks toward older history; `Alt+→` walks toward the future queue.

History is immutable. If you edit a `-N` slot, the position label shows a dirty mark (`-2*`) and navigating away pops a single-line prompt:

```
(S)end, (Q)ueue, (D)iscard, [S]tay:
```

- `s/S` — append the fork to history and return to `*`.
- `q/Q` — push to queue front and return to `*`.
- `d/D` — drop the edit and continue navigating.
- Enter / ESC / anything else — stay where you are.

`+N` and `*` are mutable: edits autosave to disk on navigate-away or focus loss, no prompt. `Alt+q` from `*` parks the current draft for later; from `-N` it forks the history entry into the queue; from `+N` it bumps the item to the front.

## Mouse

- **Click-and-drag in any pane** → starts selecting immediately. 
- **Release mouse to finish the selection** → the selection is inserted as quotes into `nvim`: 

1. If cursor is at beginning of line, insert with format `> and selected text`.
2. Otherwise, just insert selected text. 

Visual feedback is provided on inserted text.

## Dependencies

**Required**

Automatically installed with `homebrew`.

| Tool | Purpose |
|---|---|
| [`zellij`](https://zellij.dev/) | terminal multiplexer hosting the two-pane session |
| [`nvim`](https://neovim.io/) | the input/drafting pane |
| [`fzf`](https://github.com/junegunn/fzf) | session picker |
| [`jq`](https://jqlang.github.io/jq/) | JSON parsing for pane targeting |
| [`par`](https://www.nicemice.net/par/) | paragraph reflow when pasting from the agent pane |
| `python3` (any 3.x) | runs the notification-forwarding helper. Stdlib-only — already present on macOS |
| an agent | `claude`, `codex`, `gemini`, or any TUI agent you want to drive |

## Install

**Homebrew (recommended).** 

```sh
# install
brew tap xianxu/pair
brew install pair

# upgrade
brew update; brew upgrade pair
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
pair list                        # list pair-* sessions and attach state
pair -h, --help                  # show full help
```

Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.

Agent args (after `--`) are appended to the agent command line on **create**. Reattaching to an existing session does not re-launch the agent, so the args don't apply on attach. (The picker connects you to whatever's already running.)

When `pair` runs and any detached `pair-*` session exists, it shows an `fzf` picker over the detached sessions plus a `+ new <agent> session` sentinel.

When the create flow runs, it prompts for the session name with the auto-suggested name as the default:

```
Session name [claude]: <Enter to accept, or type a custom name>
```

Custom names like `bugfix`, `blogging`, or `research` are allowed (chars: `A-Z a-z 0-9 - _`). 

To detach mid-session: `Alt+d`. To re-attach: run `pair` again and pick from the list. To fully quit (no resurrect entry): `Alt+x`.

## Image paste

`Alt+i` is the integrated path: put an image on the OS clipboard first, then press `Alt+i` from inside nvim. `pair` types `Ctrl+V` into the agent pane (so claude attaches the image as a chip) *and* inserts a `[Image #N]` reference at your nvim cursor. If the local counter drifts from claude's actual count, edit the number in nvim and press `Alt+i` while on the corrected token to resync.

Caveat: the `[Image #N]` reference syntax works in Claude Code and Codex. With Gemini, the image still attaches via `Ctrl+V`, but you'll need to mention it inline however Gemini expects.

## Notifications

Pair forwards "agent needs attention" signals to your outer terminal automatically — useful for outer wrappers like [cmux](https://github.com/saharNooby/cmux) that surface badges per session.

## Why pair

- **Real editor in the input box.** Modal editing, undo, search, paste, file path completion — all the things web chat boxes don't have.
- **Doesn't touch your config.** `zellij --config-dir` and `nvim -u` isolate everything from your `~/.config`. Try it, walk away, no cleanup.
- **Prompt history is a markdown file.** Every send appends to `${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md`. Grep it.
- **Selection is the gesture.** Drag in the agent pane, release — quoted into nvim, ready to react. No keystroke between.
- **Agent-agnostic.** `pair claude`, `pair codex`, `pair gemini` — same plumbing.

For readers interested in details in design rationale and architecture, see [the original pensive](docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md) and [`atlas/architecture.md`](atlas/architecture.md).
