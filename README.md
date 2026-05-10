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
4. typeahead from highlighted terms in agent's response.

**Prompt history & a future queue, in-buffer.** 

You can use `Alt+←` and `Alt+→` to move among history of prompts you issued. You can `Alt+q` to enqueue a prompt to be parked on the side. You can use `===` at start of a line, to write comments to remind you about what this prompt is for (comment's not sent to agent). 

I use `Alt+q` extensively to park small things I notice while I work with coding agent, but not yet to the level that I want to create an issue to track it. Then, when I finish current task, I'd go pick up from the queue and work in the same session. 

`===` is also sticky, so it stays on after you submit a prompt, thus create a stable area for sticky notes.

All of those prompts are persisted on disk, keyed on the session's name.

**Copy on select, and insert into nvim buffer**

Select something on agent's pane, the selection is inserted at current mouse location if cursor is at start of line.

```
> Copid text from agent's window, reflowed to remove extra line breaks
```

Otherwise without `>`. Focus' automatically put at the likely position you want to type, such as the next line after `> quoted text line`

## Keybindings

| Key | Scope | Action |
|---|---|---|
| **Alt+Return** | nvim (normal/insert) | Send buffer to agent |
| **Alt+←** / **Alt+→** | nvim (normal/insert) | Walk through prompt history (`-N`) and queued prompts (`+N`) one slot at a time. Status line: `Alt: <- history H < pos[*] [hint] > Q queued ->`. |
| **Shift+Alt+←** / **Shift+Alt+→** | nvim (normal/insert) | Jump to the next region boundary: oldest-history, newest-history, `*`, front-of-queue, back-of-queue. Lets you skip over long histories or queues quickly. |
| **Alt+q** | nvim (normal/insert) | Push current buffer to the front of the queue (`+1`). From `*` clears the draft; from `+N` it's move-to-front. |
| **Alt+Backspace** | nvim (normal/insert), at `+N` | Delete the current queued prompt without sending. Items behind shift down so you can delete a run by tapping repeatedly. |
| **Shift+Alt+Backspace** | nvim (normal/insert) | Erase history, draft, and queue for this session — "start anew". Confirmation prompt defaults to No. Hard delete (no archive). |
| **Alt+↑** / **Alt+↓** | any pane | Step the nvim pane along a `minimized` ↔ `10 lines` ↔ `1/2` ladder one rung at a time (works from either pane). `minimized` collapses nvim to a single statusline row showing `Alt+↑ for pair input box`; confirm-requiring keys (Alt+x/d/n) auto-grow out of minimized so the modal prompt is visible. |
| **Alt+i** | nvim (normal/insert) | Attach clipboard image to the agent and insert `[Image #N]` reference at cursor. |
| **Alt+i** | when inside [Image tag] | Sync the internal counter to N (manual-correction path), allowing user to edit if the cursor between nvim and agent gets out of sync. |
| **Alt+h** | any pane | Pop up the full keybind help in a floating pane (press `q` to dismiss). |
| **Alt+d** | any pane | Detach from the current session (re-attach later via `pair`). Confirms first. |
| **Alt+x** | any pane | Full quit — kill the session and all processes inside. Confirms first. Pair captures the agent's session id alongside the launch args, so the session is resumable later via `pair resume <tag>`. |
| **Alt+n** | any pane | Restart in place — kill the session and re-launch with the same tag, agent, and agent args, but a fresh agent conversation (the saved per-(tag,agent) config is dropped before relaunch). Confirms first. |

### Prompt history & queue

The nvim pane is a virtual cursor over `[ ... -2 -1 ] * [ +1 +2 ... ]`.

**Status line:**

```
Alt: <- history 17 < * [q=queue] > 3 queued -> 
Alt: <- history 17 < -2 [q=queue] > 3 queued -> 
Alt: <- history 17 < -2* [q=queue] > 3 queued -> 
Alt: <- history 17 < +1 [⌫=del] > 3 queued -> 
```

`H` and `Q` are total history and queue counts. `pos` is one of `*`, `-N`, `+N`. The flanking `<-` and `->` hint the Alt+← / Alt+→ navigation. The `[key=action]` hint inside the brackets is contextual: `[q=queue]` on `*`/`-N`, `[⌫=del]` on `+N`. A trailing `*` on `-N` means you've edited that history entry and have an unsent fork.

History is immutable. If you edit a `-N` slot, the position label shows the dirty mark (`-2*`) and navigating away pops a single-line prompt:

```
(S)end, (Q)ueue, (D)iscard, [S]tay:
```

- `s/S` — append the fork to history and return to `*`.
- `q/Q` — push to queue front and return to `*`.
- `d/D` — drop the edit and continue navigating.
- Enter / ESC / anything else — stay where you are.

`+N` and `*` are mutable: edits autosave to disk on navigate-away or focus loss, no prompt. `Alt+q` from `*` parks the current draft for later; from `-N` it forks the history entry into the queue; from `+N` it bumps the item to the front. `Alt+Backspace` deletes the current `+N` (no-op anywhere else). When you mouse-select text in the agent pane, the selection always goes to the OS clipboard, but the auto-quote-into-nvim only fires when nvim is in **insert mode** — so browsing history in normal mode doesn't get its buffer overwritten.

### Draft comments (`===`)

Lines starting with `===` (leading whitespace allowed) are **stripped from the prompt at send time** but **kept in draft, queue, and log files**. Useful for "remember what this is for" notes that travel with a queued prompt and survive history navigation.

```
=== queued for after the build passes — re-check Auth.tsx imports
fix the token-rotation bug in src/auth/session.ts
```

Only the second line reaches the agent; the comment stays attached when you scroll back through history or browse the queue.

- Whole-line only — mid-line `===` is unaffected (`a === b` ships as-is).
- A prompt that's all comments is a no-op send (no log entry, no queue item consumed, no flash).
- Stripping is line-based and **not fence-aware**: a `===` line inside a fenced code block also gets stripped. Use `# H1` headings or `<!-- ... -->` if you need literal `===` in a sent prompt.
- Comment-only edits to a `-N` history entry **autosave back into the log** — annotating an old prompt isn't a fork (the agent's view is unchanged), so it doesn't trigger the dirty prompt and the note is preserved across navigation and nvim restarts.

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

**Optional**

| Tool | Purpose |
|---|---|
| [`pyte`](https://github.com/selectel/pyte) | Python terminal emulator used by `Alt+/` to render full scrollback with colors and matching line numbers. Without it, `Alt+/` prints a hint; everything else works. Install with `make pair-bootstrap` (or `pip3 install --user pyte`). |

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
pair resume <tag>                # restart a tag with its saved config
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

## Resume a session by tag

Pair captures each new session's startup args plus the agent's own session id, keyed by tag. After `Alt+x` you'll see:

```
pair: saved session config for tag "pair-bugfix" (claude).
      resume with: pair resume pair-bugfix
```

Run that command and the picker + name prompt are skipped. Pair offers three options for what to do with the saved config:

```
saved config for tag 'bugfix' (claude)
  1) use params + session
       args=[--dangerously-skip-permissions]
       resume=8d745d08-4ecc-4474-969a-53c98a6fa5f0
  2) use params
       args=[--dangerously-skip-permissions]
       fresh session
  3) use none
       args=[<whatever you passed on this command>]
       fresh session
```

- **use params + session** replays the original launch args *and* points the agent at its previous session id (claude's `--resume`, codex's `resume <id>` subcommand, gemini's `--resume`). The "session" option only appears if the agent's transcript file is still on disk.
- **use params** replays the args but starts a fresh agent session.
- **use none** ignores the saved config (and deletes it); a new config is captured the next time the watcher sees a session id appear.

The agent (claude / codex / gemini) is inferred from saved state, so `pair resume <tag>` is enough on its own — no need to repeat the agent positional. If `pair-<tag>` is still a running zellij session (e.g. you only `Alt+d` detached), `pair resume <tag>` re-attaches without prompting.

Saved configs live at `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json`.

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
