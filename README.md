# pair

A small launcher that gives any TUI coding agent (Claude Code, Codex, Gemini CLI) a real input field — backed by Neovim. `pair` glues several tools together: `zellij`, `nvim`, `fzf` etc.

All [AI created](https://xianxu.dev/2026/05/a-saturday-coding-session/).

## What it does

Launches a `zellij` session split into two panes:

- **Top** — the coding agent, `Claude`, `Codex`, `Gemini`, ...
- **Bottom (~12 rows)** — Neovim on a persistent draft file. 

You compose prompts with full editor power, scroll the agent output independently. When you are done, `Alt+Return` to send your text to the agent.

Works on Mac, probably on Linux, but haven't tested.

## What do you get

**Full `nvim` support in draft**

For example:

1. mouse support
2. syntax color, spelling check etc.
3. typeahead and search local file path, just type `./` and then continue
4. typeahead from highlighted terms in agent's response

**Much better scrollback**

For example:

1. search in the scrollback buffer and / or ? to search
2. comment (select then Alt+q) on agent's response, which is inserted upon exit from scrollback buffer

**Prompt history & draft queue** 

You can use `Alt+←` and `Alt+→` to move among history of prompts you issued. You can `Alt+q` to enqueue a prompt to be parked on the side. You can use `===` at start of a line, to write comments to remind you about what this prompt is for (comment in draft window's not sent to agent). 

I use `Alt+q` extensively to park small things I notice while I work with coding agent, but not yet to the level that I want to create an issue to track separately. Then, when I finish current task, I'd go pick up from the queue and work in the same session. 

`===` is also sticky, so it stays on after you submit a prompt, thus create a stable area for sticky notes.

All of those prompts are persisted on disk, keyed on the session's name. So next time you start up `pair` with same tag name, you recover all prompt history and future queue. 

**Copy on select from Claude, and insert into draft nvim as quotation**

Select something with mouse on agent's pane, the selection is inserted at current mouse location in nvim, like: 

```
> Copid text from agent's window, reflowed to remove extra line breaks
```
## Keybindings

| Key | Scope | Action |
|---|---|---|
| **Alt+h** | any pane | Pop up the full keybind help in a floating pane (press `q` to dismiss). |
| **Alt+Return** | nvim (normal/insert) | Send buffer to agent. Note for consistency, claude's keybinding also changed to Alt+return as send, and return as newline |
| **Alt+←** / **Alt+→** | nvim (normal/insert) | Walk through prompt history (`-N`) and queued prompts (`+N`) one slot at a time. |
| **Alt+↑** / **Alt+↓** | any pane | Step the nvim pane along a `minimized` ↔ `12 lines` ↔ `1/2` ladder one rung at a time. When minimized, claude pane always have focus |
| **Alt+i** | nvim (normal/insert) | Attach clipboard image to the agent and insert anchor text at cursor location |
| **Shift+Alt+←** / **Shift+Alt+→** | nvim (normal/insert) | Jump to the next region boundary: oldest-history, newest-history, `*`, front-of-queue, back-of-queue. |
| **Alt+q** | nvim (normal/insert) | Push current buffer to the front of the queue (`+1`). From `*` clears the draft; from `+N` it's move-to-front. |
| **Alt+/** | any pane | Enter into scrollback viewer, at same view port of current mouse scroll state of claude pane |
| **Alt+q** | scrollback viewer | Insert comment for the line, or selection |
| **Alt+Backspace** | nvim (normal/insert), at `+N` | Delete the current queued prompt. |
| **Shift+Alt+Backspace** | nvim (normal/insert) | Erase history, draft, and queue for this session to "start anew". |
| **Alt+d** | any pane | Detach from the current session (re-attach later via `pair`). |
| **Alt+x** | any pane | Full quit — kill the session and all processes inside. Pair captures the agent's session id alongside the launch args, so the session is resumable later via `pair resume <tag>`. |
| **Alt+n** (or **Ctrl+Alt+n**) | any pane | Reload pair — kill the session and re-launch with the same tag, agent, args, AND agent session. Ctrl+Alt+n is the macOS-friendly alias — adding Ctrl defeats the Option+n dead-tilde composer on newer macOS / terminal combos that ignore the Option-as-Meta setting. Press Alt+n twice works as well. |
| **Shift+Alt+N** | any pane | Restart with a fresh agent conversation — same tag, agent, and args as Alt+n, but the saved per-(tag,agent) config is dropped before relaunch so the agent starts a brand-new session. |

## Prompt history & queue

The nvim pane is a virtual cursor over `[ ... -2 -1 ] * [ +1 +2 ... ]`.

**Status line:**

```
Alt: <- history 17 < * [q=queue] > 3 queued -> 
Alt: <- history 17 < -2 [q=queue] > 3 queued -> 
Alt: <- history 17 < -2* [q=queue] > 3 queued -> 
Alt: <- history 17 < +1 [⌫=del] > 3 queued -> 
```

`17` and `3` are total history and queue counts. `pos` is one of `*`, `-N`, `+N`. The flanking `<-` and `->` hint the Alt+← / Alt+→ navigation. The `[key=action]` hint inside the brackets is contextual: `[q=queue]` on `*`/`-N`, `[⌫=del]` on `+N`. A trailing `*` on `-N` means you've edited that history entry and have an unsent fork.

History is immutable. If you edit a `-N` slot, the position label shows the dirty mark (`-2*`) and navigating away pops a single-line prompt:

```
(S)end, (Q)ueue, (D)iscard, [S]tay:
```

- `s/S` — append the fork to history and return to `*`.
- `q/Q` — push to queue front and return to `*`.
- `d/D` — drop the edit and continue navigating.
- Enter / ESC / anything else — stay where you are.

queue `+N` and draft `*` are mutable: edits autosave to disk on navigate-away or focus loss, no prompt. `Alt+q` from draft `*` parks the current draft for later; from history `-N` it forks the history entry into the queue; from `+N` it bumps the item to the front. `Alt+Backspace` deletes the current `+N` (no-op anywhere else). When you mouse-select text in the agent pane, the selection always goes to the OS clipboard, but the auto-quote-into-nvim only fires when nvim is in **insert mode** — so browsing history in normal mode doesn't get its buffer overwritten.

## Draft comments (`===`)

Lines starting with `===` (leading whitespace allowed) are **stripped from the prompt at send time** but **kept in draft, queue, and log files**. Useful for "remember what this is for" notes that travel with a queued prompt and survive history navigation.

```
=== queued for after the build passes — re-check Auth.tsx imports
fix the token-rotation bug in src/auth/session.ts
```

Only the second line reaches the agent.

- Whole-line only — mid-line `===` is unaffected (`a === b` ships as-is).
- A prompt that's all comments is a no-op send (no log entry, no queue item consumed, no flash).
- Comment-only edits to a `-N` history entry **autosave back into the log** — annotating an old prompt isn't a fork (the agent's view is unchanged), so it doesn't trigger the dirty prompt and the note is preserved across navigation and nvim restarts.

## Mouse

- **Click-and-drag in agent pane** → starts selecting immediately. 
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
| an agent | `claude`, `codex`, `gemini`, or any TUI agent you want to drive |

## Terminal setup

Pair leans on `Alt+<key>` chords for almost every action — `Alt+Return` to send, `Alt+x/d/n/N` to quit/detach/restart, `Alt+↑/↓` for layout, `Alt+i` for image attach, `Alt+/` for scrollback, `Alt+q` for marker comments. macOS terminals don't all forward Option as a meta-prefix by default, so the chords silently insert macOS special characters (`Alt+x` → `≈`, `Alt+e` → ``` ` ```, etc.) instead of reaching pair. One-time per-terminal setup:

| Terminal | Setting | Default | Required |
|---|---|---|---|
| **Ghostty** | `macos-option-as-alt = true` | already `true` | nothing — works out of the box |
| **iTerm2** | Settings → Profiles → Keys → General → Left/Right Option Key → **Esc+** | Normal | flip both Option-key dropdowns to **Esc+** |
| **Terminal.app** | Settings → Profiles → Keyboard → **Use Option as Meta key** | unchecked | check the box |

Symptom when not configured: `Alt+Return` may still send (since that chord doesn't have a macOS special character), but `Alt+x` prints `≈` in nvim, `Alt+n` prints `˜`, `Alt+d` prints `∂`, etc. — the literal Unicode insertions tell you the chord was eaten by macOS before reaching pair.

Newer MacOS `Alt+n` sends dead-tilda. You can use [Ukelele](https://software.sil.org/ukelele/) to create a Mac keyboard configuration without those dead-letter.

## Install

**Homebrew (recommended).** 

```sh
# install
brew tap xianxu/pair: brew install pair

# upgrade
brew update; brew upgrade pair
```

That installs `zellij`, `neovim`, `fzf`, `jq`, and `par` if they aren't already present. The agent (`claude`, `codex`, `gemini`) you install separately. Then:

## Command Usage

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

## Notifications

Pair forwards "agent needs attention" signals to your outer terminal automatically — useful for outer wrappers like [cmux](https://github.com/saharNooby/cmux) that surface badges per session.

For readers interested in details in design rationale and architecture, see [the original pensive](docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md) and [`atlas/architecture.md`](atlas/architecture.md).
