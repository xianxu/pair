# pair

A small launcher that gives any TUI coding agent (Claude Code, Codex, Gemini CLI) a real input field — backed by Neovim. `pair` glues several tools together: `zellij`, `nvim`, `fzf`, `par`. 

## What it does

Launches a zellij session split into two panes:

- **Top** — the coding agent, `Claude`, `Codex`, `Gemini`, ...
- **Bottom (~10 rows)** — Neovim on a persistent draft file. 

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
| `python3` (any 3.x) | runs `bin/pair-wrap`, the PTY proxy that translates agent OSC notifications to outer-terminal OSC 9. Stdlib-only — no `pip install` needed. Usually already present via Xcode CLT |
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
pair -h, --help                  # show full help
```

Use `--` to separate pair's positional from agent flags. Without it, pair only takes `<agent>` as a positional and everything else is rejected.

Agent args (after `--`) are appended to the agent command line on **create**. Reattaching to an existing session does not re-launch the agent, so the args don't apply on attach. (The picker connects you to whatever's already running.)

Caveat: agent args are word-split by the shell, so quoted args containing spaces (`pair claude -- --system "hello world"`) get split into multiple args. Most CLI flags don't have spaces, but worth knowing.

When `pair` runs and any detached `pair-*` session exists, it shows an `fzf` picker over the detached sessions plus a `+ new <agent> session` sentinel.

When the create flow runs, it prompts for the session name with the auto-suggested name as the default:

```
Session name [claude]: <Enter to accept, or type a custom name>
```

Custom names like `bugfix`, `blogging`, or `research` are allowed (chars: `A-Z a-z 0-9 - _`). 

To detach mid-session: `Alt+d`. To re-attach: run `pair` again and pick from the list. To fully quit (no resurrect entry): `Alt+x`.

## Image paste

`Alt+i` is the integrated path: put an image on the OS clipboard first, then press `Alt+i` from inside nvim. `pair` types `Ctrl+V` into the agent pane (so claude attaches the image as a chip) *and* inserts a `[Image #N]` reference at your nvim cursor. If the local counter drifts from claude's actual count, edit the number in nvim and press `Alt+i` while on the corrected token to resync.

Caveat: the reference [Image #N] doesn't work with Genimi. Though user can manually paste the reference from Gemini over.

## Notifications via outer-PTY passthrough

Zellij filters most OSC escapes (parses them for its virtual screen and drops anything it doesn't recognize). That means OSC 9 / OSC 777 emitted from inside a pane never reach an outer wrapper that watches the PTY stream — e.g. cmux, which uses these escapes to surface "agent needs attention" badges. BEL is forwarded (so terminal-emulator bells work fine) but cmux watches OSC, not BEL.

Pair works around this in two layers:

1. **Outer-TTY capture.** On every attach, `bin/pair` records its controlling TTY (the outer PTY's path) into `${XDG_DATA_HOME:-~/.local/share}/pair/outer-tty-<tag>`. Anything that wants to talk to the outer terminal directly reads this file.
2. **Transparent agent wrapper.** The agent runs under `pair-wrap`, a small PTY proxy that watches the agent's output stream. It parses every OSC sequence and translates *actionable* notifications into an OSC 9 written directly to the recorded outer-TTY path — bypassing zellij. The filter:
   - **Forward:** OSC 777 (`notify;<title>;<msg>`), OSC 9 with text body.
   - **Skip:** OSC 0/1/2 (window-title sets — claude updates these every second with a spinner; without filtering this single-handedly causes a notification storm), OSC 9;4;... (iTerm progress codes — fire on every tool-call cycle).
   - Bare BEL → forwarded as a fallback when no OSC framing is found.

For Claude Code specifically: vanilla claude emits `OSC 777;notify;Claude Code;Claude is waiting for your input` after ~60s of idle waiting. That signal flows through `pair-wrap` → outer terminal → cmux badge with no extra config.

If you want richer signals (semantic events like `Notification` vs `Stop`, custom messages, faster turnaround than 60s), use the bundled `pair-notify` helper from a Claude Code hook. Sample `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Notification": [{
      "hooks": [{ "type": "command", "command": "pair-notify 'Claude needs input'" }]
    }],
    "Stop": [{
      "hooks": [{ "type": "command", "command": "pair-notify 'Claude is done'" }]
    }]
  }
}
```

Use `pair-notify --osc 777 "msg"` for the urxvt-style variant. Outside a pair session both `pair-wrap` and `pair-notify` are graceful: the wrapper just acts as a transparent proxy, and the helper exits cleanly with a stderr warning. Safe to leave configured globally.

### Debugging an agent's notification protocol

When pairing with a new agent (codex, gemini, etc.) it's not always obvious what OSC family — if any — the agent uses for "I want attention." `pair-wrap` has an opt-in forensic log that records every OSC and BEL it sees, with timestamps and what action it took. Use it to discover the agent's protocol the first time, then update `is_actionable_osc()` in `bin/pair-wrap` if the agent uses something the current filter doesn't recognize.

```sh
PAIR_WRAP_LOG=~/pair-wrap.log pair codex
# use the agent normally; let it idle, finish tasks, etc.
# detach with Alt+d when done
cat ~/pair-wrap.log
```

Log lines you'll see:

| Line | Meaning |
|---|---|
| `OSC<N>: b'<body>'` | OSC `<N>` recognized as actionable; emit fired |
| `OSC<N>-skip: b'<body>'` | OSC `<N>` recognized but filtered (title set, progress, etc.) |
| `BEL: b'<context>'` | bare BEL fallback fired (no OSC framing seen) |
| `EMIT: 'wrote OSC 9 to <path>'` | successful write to outer TTY (cmux should have badged) |
| `EMIT-skip: 'rate-limited (...)'` | within 0.5s of last emit; collapsed |
| `EMIT-skip: 'no outer-tty file...'` | not running under pair, or `record_outer_tty` failed |
| `EMIT-fail: '<path>: ...'` | tried to write but the recorded path is gone or unwritable |

Reading strategy:
1. Look for *any* `OSC` or `BEL` lines that fired around moments where the agent was waiting for you. That's the actionable signal.
2. Look at the gaps. Most agents wait some seconds before emitting an attention signal — note the cadence so you know what to expect.
3. If only `-skip` lines appear, the agent's signals are all things we filter out. Either (a) the agent doesn't have an attention notification protocol and you'll need a hook-based path (`pair-notify`), or (b) the agent uses an OSC family `is_actionable_osc()` doesn't yet recognize — extend the filter.

## Files

```
bin/pair                     # launcher
bin/clipboard-to-pane.sh     # reflow + > -prefix + write into nvim pane
bin/copy-on-select.sh        # zellij copy_command — wraps clipboard-to-pane on mouse-up
bin/pair-notify              # emit OSC 9/777 to the outer terminal, bypassing zellij
bin/pair-wrap                # transparent PTY proxy — translates agent BEL/OSC to outer terminal
bin/pair-quit.sh             # Alt+x — marks session for cleanup, kills it
nvim/init.lua                # bundled nvim config (loaded via `nvim -u`)
zellij/config.kdl            # mouse_click_through, copy_command, keybinds (Alt+u/d/i/x)
zellij/layouts/main.kdl      # the split (agent on top, nvim 10 rows on bottom)
```

Drafts and prompt history live under `${XDG_DATA_HOME:-~/.local/share}/pair/`, keyed by *tag* (the agent name, or your custom session name):

- `draft-<tag>.md` — the active draft (cleared on send, persists across launches)
- `log-<tag>.md` — appended on every send with timestamp; your grep-able prompt history

Quit-marker file (used internally by Alt+x): `~/.cache/pair/quit-<session>` — touched by `pair-quit.sh`, removed by `bin/pair` when it cleans up the session.

Outer-TTY file (used internally by `pair-notify`): `${XDG_DATA_HOME:-~/.local/share}/pair/outer-tty-<tag>` — written by `bin/pair` on every attach with the path of pair's controlling TTY before zellij takes over.

## Design notes

See [the pensive that motivated this](docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md) and `atlas/architecture.md` for the architecture map.

Design choices:

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit and lets each side specialize.
- **Self-contained, doesn't touch your config.** Uses `zellij --config-dir` and `nvim -u` to fully isolate from your normal `~/.config/{zellij,nvim}`. Try it without commitment.
- **Prompt history is just a markdown file.** Every send appends to `${XDG_DATA_HOME:-~/.local/share}/pair/log-<tag>.md`. Grep, diff, copy from. Your conversations are searchable forever.
- **Explicit reattach.** The picker fires whenever any detached session exists, so you always see what you're connecting to. 
- **One detach, one quit.** Alt+d detaches (session keeps running, claude/nvim stay alive, re-attach via `pair`). Alt+x fully quits and cleans up the resurrect entry. Zellij's default Ctrl+q is unbound — it would otherwise leave a confusing half-state.
- **Selection is the gesture.** Click-and-drag in the agent pane, mouse up — the quote is in nvim, ready for your reaction. No keystroke between.
- **Agent-agnostic.** The same zellij+nvim plumbing works for any TUI agent. Switching from `pair claude` to `pair codex` is one keystroke.
