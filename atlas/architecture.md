# Architecture

## What pair is

A launcher that starts a zellij session with a fixed two-pane split. The top pane runs a TUI coding agent; the bottom pane runs Neovim on a persistent draft file. Keystrokes — and mouse-up after a selection — drive bidirectional flow between the panes via `zellij action write-chars` and `zellij action focus-pane-id`.

The whole thing is deliberately small — a handful of shell scripts, one nvim init, and two zellij KDL files. Required deps: `zellij`, `nvim`, `fzf`, `jq`, `par`, plus the agent itself.

## Pieces

```
bin/pair                     # entry point (launcher)
bin/clipboard-to-pane.sh     # read clipboard, hand off to nvim's PairPasteQuote
bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
nvim/init.lua                # bundled nvim config (loaded via -u)
zellij/config.kdl            # mouse, copy_command, keybinds
zellij/layouts/main.kdl      # the split + agent/draft commands
```

### `bin/pair` — launcher

Resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, sets `$PAIR_TAG` to the agent name (custom names come from the create-flow prompt), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:

**Decision tree.** Finds *all* detached pair-* sessions on the machine (any agent, any naming). Then:

- 0 detached → run create flow directly (validate agent, prompt for name, create).
- ≥1 detached → fzf picker over the detached sessions plus a `+ new <agent> session` sentinel. Pick a session → attach. Pick the sentinel → fall through to create.

The agent argument doesn't filter the picker — reattach is agent-agnostic (the existing session already runs whatever it runs). The agent argument only matters for the create path: it labels the sentinel, drives the auto-suggested default name, and is the binary that gets exec'd in the new session.

There is **no silent auto-attach**. Every reattach goes through the picker so the user explicitly sees what they're connecting to.

Detection of attached-vs-detached uses `zellij --session NAME action list-clients`, which prints a header plus one row per connected client. Zero rows = detached.

**Naming prompt.** When the create flow runs, the launcher prompts the user with the auto-suggested tag as the default (`Session name [claude-2]:`). The `pair-` prefix is implicit — the prompt shows just the tag since `pair-` is always prepended. Pressing Enter accepts; typing a custom name (`bugfix`, or `pair-bugfix` — leading `pair-` is stripped) overrides it.

**Agent validation deferred.** `command -v "$AGENT"` runs only inside the create branch, not at startup, so attaching to a custom-named session whose tag isn't a real binary still works.

**Title.** The launcher emits an OSC 0 escape sequence right before invoking zellij, so the terminal title shows the session name on both create and attach paths (zellij itself only sets it on create).

**Cleanup on quit.** zellij is run as a child (not `exec`) so the launcher resumes when zellij exits. On resume it checks for `~/.cache/pair/quit-<session>` (the marker that `pair-quit.sh` writes when Alt+x fires) and, if present, runs `zellij delete-session --force <session>` to clear the resurrect entry. No marker → leave the session as zellij left it (running if Alt+d detached).

### `zellij/layouts/main.kdl` — pane split

Horizontal split. Top pane runs `$PAIR_AGENT $PAIR_AGENT_ARGS` (auto-fills remaining height). Bottom pane is a fixed 11 rows running `nvim -u $PAIR_HOME/nvim/init.lua` on the per-tag draft file.

Both panes wrap their command in `sh -c "..."` so the shell expands `$PAIR_AGENT`, `$PAIR_AGENT_ARGS`, `$PAIR_TAG`, and `$PAIR_HOME` at exec time — zellij itself does not interpolate env vars in `command`/`args` fields.

`$PAIR_AGENT_ARGS` is appended on the agent pane command line as a single space-separated string; the shell word-splits it. Args containing spaces are *not* preserved (rare for CLI flags; documented in README).

The bottom pane has `focus=true` (drafting pane gets focus on launch) and a `name=` set to the help string (`Alt: ⏎=send  u=max  i=img  d=detach  x=quit`) so zellij renders that as the pane's frame title.

### `zellij/config.kdl` — mouse, copy, keybinds

Top-level config:

- `mouse_click_through true` — first click on an unfocused pane goes through to the pane (so click-and-drag selects in one motion) instead of being consumed by zellij just to change focus.
- `copy_command "copy-on-select.sh"` — on every selection finalize (mouse-up after drag), zellij pipes the selected text to this script. `copy_command` replaces zellij's default OS-clipboard write, so the script does that part too. Resolved by PATH (which `bin/pair` populated).

Keybinds added on top of zellij defaults (`clear-defaults=false`):

- `unbind "Alt i"` — release Alt+i (zellij's default binds it to MoveTab; we want nvim to see it for image attach).
- `Alt+d` — `Detach` — detach from the session.
- `Alt+u` — `MoveFocus Down; ToggleFocusFullscreen` — toggle nvim pane fullscreen, regardless of which pane has focus.
- `Alt+x` — `Run "pair-quit.sh"` — full quit (writes marker, kills session).

Alt+n (clipboard → nvim quote) used to be a manual keybind here too, but became redundant once `copy_command` started auto-firing on mouse-up. Removed.

### `bin/clipboard-to-pane.sh` — clipboard read + hand off to nvim

Read OS clipboard (`pbpaste` / `wl-paste` / `xclip`). Stage the raw body to `$PAIR_DATA_DIR/quote-<tag>`. All formatting decisions (par reflow, `> ` prefix) live in nvim now, conditional on cursor position — the shell is just a transport.

Find the nvim pane via `zellij action list-panes --json`, looking for the pane whose `terminal_command` contains `nvim`. Focus it via `zellij action focus-pane-id <id>` — this is critical because the script runs inside a transient `Run` pane (when invoked directly) or as a child of the zellij server (when invoked via `copy_command`), and we cannot rely on positional `move-focus` to land on nvim.

Once nvim is focused, send `Ctrl-\ Ctrl-N` (force normal) followed by `:lua PairPasteQuote()` + CR. `PairPasteQuote` reads the staged body and dispatches on cursor column — see the `nvim/init.lua` section below.

Why force normal via Ctrl-\ Ctrl-N rather than Esc: Esc + a literal char is the terminal encoding for Alt+`<char>`, which would fire `<M-...>` keymaps spuriously (e.g. `<M-i>` → `attach_image` inserting a stray `[Image #N]`). Ctrl-\ Ctrl-N has no Alt-encoding ambiguity.

Diagnostic log lives at `${XDG_CACHE_HOME:-~/.cache}/pair/clipboard-debug.log` (overwritten each invocation).

### `bin/copy-on-select.sh` — zellij copy_command wrapper

Receives selected text on stdin from zellij. Mirrors the text to the OS clipboard (zellij's default clipboard write is bypassed when `copy_command` is set, so this is mandatory). Then checks if the focused pane (where the selection happened) is the nvim draft pane; if so, exits without further action (selecting in nvim shouldn't loop back). Otherwise:

- Flashes the focused pane's background via `zellij action set-pane-color --pane-id <id> --bg <color>`, then backgrounds a delayed `--reset` (the bash subshell outlives the script's `exec` below). Default flash color `#5a4a00`, duration 100ms (intentionally shorter than the 500ms nvim-side flash — the source flash is a quick "fired" pulse, the destination flash lingers to orient the user on where the text landed); override via `$PAIR_FLASH_BG` and `$PAIR_FLASH_MS`. Best-effort: `set-pane-color` only affects the pane's default bg, so cells the agent has actively painted won't change. Visible-or-not depends on the agent.
- Execs `clipboard-to-pane.sh` to hand the selection off to nvim.

Pane detection: parse `list-panes --json --command`, find the focused pane, check if its `title` or `terminal_command` matches `nvim|draft`.

### `bin/pair-quit.sh` — Alt+x handler

Touches the marker file `~/.cache/pair/quit-$ZELLIJ_SESSION_NAME`, truncates the per-tag draft file (`$PAIR_DATA_DIR/draft-<tag>.md`) so the next session starts on a blank buffer, then `exec zellij kill-session $ZELLIJ_SESSION_NAME`. The kill terminates the session including the script itself; on the launcher side, `bin/pair` resumes, sees the marker, and runs `delete-session --force` to clean up the resurrect entry.

The draft is truncated rather than removed so its persistent-undo entry under `~/.local/share/pair/undo/` (keyed by file path) stays addressable — if you full-quit by accident you can still re-launch and `u` your way back to recent state.

### Outer-TTY capture and notification routing — `bin/pair-wrap`, `bin/pair-notify`

**Why.** Zellij parses every escape on the way out for its virtual-screen reconstruction and drops sequences it doesn't recognize. OSC 9 and OSC 777 (the notification escapes outer wrappers like cmux watch for) fall in that bucket and never reach the host terminal. BEL is forwarded since zellij 0.44, but cmux specifically watches OSC, not BEL — so BEL forwarding doesn't help that integration. Filed as #000011.

**Mechanism, in two layers:**

1. **Outer-TTY capture (in `bin/pair`).** Before invoking zellij, on every attach (both create and reattach branches), pair calls `tty(1)`. The result is the path of pair's controlling TTY — which is precisely the outer PTY (the one allocated by whatever wraps pair: cmux, a terminal emulator, etc.). That path gets written to `$DATA_DIR/outer-tty-<tag>`. Refreshed on every attach because the outer PTY changes across detach/reattach, while pane-shell env stays frozen at zellij session-creation time (env-var approaches would go stale).

2. **Two consumers** of the captured path:

   - **`bin/pair-wrap`** (Python). Transparent PTY proxy. The zellij agent pane runs `pair-wrap $PAIR_AGENT $PAIR_AGENT_ARGS` instead of the agent directly (wired in `zellij/layouts/main.kdl`). The wrapper allocates a fresh PTY for the agent, forwards stdin/stdout transparently with SIGWINCH propagation, and watches the agent's output stream for OSC notifications. On detection it writes OSC 9 directly to the recorded outer-TTY path — bypassing zellij.

     **Stdin raw mode.** The wrapper switches its stdin (zellij's pane PTY) into termios raw mode for the duration. Without this the kernel's line discipline does local echo + canonical buffering on the bytes flowing toward the wrapped TUI, which double-echoes keystrokes and corrupts terminal-response sequences. Saved/restored in a `finally` block.

     **OSC filter (`is_actionable_osc`).** Parsing every OSC `<Ps>;<body>` and discriminating is essential — naive "any BEL → emit" over-fires constantly because claude (and similar agents) update OSC 0 (window title) every second with a spinner, and every title set's BEL terminator looks like a "lone bell." The filter:
     - **Skip** OSC 0/1/2 (title sets), OSC 9;4;... (iTerm progress codes — fire on every tool-call cycle).
     - **Forward** OSC 777;... (urxvt-style `Notify`) and OSC 9;`<text>` (iTerm-style notification with content).
     - Bare BEL (no OSC framing in the rolling buffer) → **logged but not forwarded by default**; set `PAIR_WRAP_BELL_FALLBACK=1` to re-enable forwarding (issue #000014).

     Rate-limited to one emit per 0.5s. Empirically: claude emits `OSC 777;notify;Claude Code;Claude is waiting for your input` after ~60s of idle waiting — that's the actionable signal that gets through.

     **Why bare BEL is opt-in.** When an OSC sequence's terminating `\x07` arrives in a read whose preceding bytes (the `\x1b]<ps>;` opener) were already consumed by a prior match, `OSC_RE` can't reconstruct the boundary, and the trailing `\x07` looks like a standalone BEL. Live data from a single 2hr Claude Code session showed 76 emits, only 8 legitimate (all OSC 777); the other 68 were BEL fallback firing on tails of OSC 8 hyperlinks (claude renders file references as clickable links) and OSC 0 spinner title sets. Modern TUI agents signal attention via OSC 9/777 explicitly — the BEL fallback's defensive value never materialized. The detection branch still runs (so `PAIR_WRAP_LOG` shows `BEL-skip` lines), it just doesn't write to the outer TTY unless the env flag is set.

     **Debug log.** `PAIR_WRAP_LOG=<path>` enables a per-detection forensic trail (timestamp, OSC/BEL match, emit/skip outcome). Off by default. Used to discover an unfamiliar agent's notification protocol the first time, then update `is_actionable_osc()` if the agent uses a family the current filter doesn't recognize.

     ```sh
     PAIR_WRAP_LOG=~/pair-wrap.log pair codex
     # use the agent normally; let it idle, finish tasks, etc.
     # detach with Alt+d when done
     cat ~/pair-wrap.log
     ```

     Log lines:

     | Line | Meaning |
     |---|---|
     | `OSC<N>: b'<body>'` | OSC `<N>` recognized as actionable; emit fired |
     | `OSC<N>-skip: b'<body>'` | OSC `<N>` recognized but filtered (title set, progress, etc.) |
     | `BEL: b'<context>'` | bare BEL fallback fired (only with `PAIR_WRAP_BELL_FALLBACK=1`) |
     | `BEL-skip: b'<context>'` | bare BEL detected but not forwarded (default) |
     | `EMIT: 'wrote OSC 9 to <path>'` | successful write to outer TTY (cmux should have badged) |
     | `EMIT-skip: 'rate-limited (...)'` | within 0.5s of last emit; collapsed |
     | `EMIT-skip: 'no outer-tty file...'` | not running under pair, or `record_outer_tty` failed |
     | `EMIT-fail: '<path>: ...'` | tried to write but the recorded path is gone or unwritable |

     Reading strategy: look for `OSC` or `BEL` lines that fired around moments where the agent was waiting — that's the actionable signal. If only `-skip` lines appear, either (a) the agent has no attention notification protocol and you'll need a hook-based path (`pair-notify`), or (b) the agent uses an OSC family `is_actionable_osc()` doesn't yet recognize — extend the filter.

   - **`bin/pair-notify`** (bash). Hook-driven helper for richer signals. `pair-notify [--osc 9|777] "msg"` reads the same outer-TTY file and writes the OSC. Intended for Claude Code `Notification`/`Stop` hooks where you want semantic events with custom message text rather than relying on the agent's native OSC stream.

**Failure mode.** Both are designed to never block the agent. `pair-wrap` swallows exceptions in the detection/emission path and keeps proxying. `pair-notify` exits 0 with a stderr warning when `PAIR_TAG` is unset, the file is missing, or the recorded path isn't writable.

### `nvim/init.lua` — drafting buffer config

Loaded via `nvim -u`, fully isolated from the user's main nvim config. Provides:

- Drafting-friendly defaults: no line numbers, wrap, linebreak, breakindent, spell, persistent undo under `~/.local/share/pair/undo/`, `cmdheight=0` to keep the cmdline out of the way, custom statusline (see "prompt history & queue" below).
- `<M-CR>` (Alt+Return, normal+insert) — `send_and_clear`: append buffer to log, send to agent pane via `zellij action focus-pane-id` + `write-chars` + Enter, clear `*` (only when source was `*`), save, drop into insert mode.
- `<M-Left>` / `<M-Right>` — navigate the prompt-history / queue position (see below).
- `<M-q>` — push the current buffer to the front of the queue. From `*` also clears `*`; from `+N` it's move-to-front (removes the source queue file).
- `<M-BS>` — delete the current `+N` queue item without sending; "stay-near" behavior (items behind shift down, position label keeps its number, so the next item is now under the cursor for repeat-delete). No-op from `*` or `-N`.
- `<M-i>` (Alt+i, normal+insert) — `attach_image`: increment per-session counter, send Ctrl+V to the agent pane (claude reads OS clipboard, attaches image), insert `[Image #N]` at cursor. If cursor is on an existing `[Image #N]`, sync the counter to N instead.
- `PairPasteQuote()` (global, called from `bin/clipboard-to-pane.sh` via `:lua PairPasteQuote()`): reads the raw selection from `$PAIR_DATA_DIR/quote-<tag>` and dispatches on cursor column.
  - **col == 0 (`paste_as_quote`)**: par-reflow with width 1000, prefix every line with `> `; if the cursor's line is empty, replace it, else insert above (existing line slides down); scroll first inserted line to top via `zt`; cursor on a single empty line directly below the block in insert mode; flash the quoted lines with `IncSearch` (full-line, per-line `nvim_buf_add_highlight`).
  - **col > 0 (`paste_inline`)**: par-reflow (so hard-wrapped sources collapse to one continuous run, paragraph breaks preserved), insert at the cursor via `nvim_buf_set_text` (handles multi-line splits); cursor at the end of the inserted span in insert mode; no scroll; flash the inserted span with a single multi-line extmark.
  - In both modes the highlight is cleared 500ms later via `vim.defer_fn`. Selection-finalize visual cue (issue #12).
- Autosave on `BufLeave`, `FocusLost`, `InsertLeave` so disk and buffer agree.
- As-you-type fuzzy path completion (issue #13). `TextChangedI`/`TextChangedP` autocmd splits the trailing path token on the last `/` into `<dir>` + `<filter>`, lists `<dir>` via `getcompletion`, fuzzy-filters with built-in `matchfuzzy`, hands the result to `vim.fn.complete()`. Triggers only when the token contains `/` or starts with `~` (plain words stay quiet). `<Tab>`/`<S-Tab>` cycle, `<CR>` accepts when an item is selected (else newline). Plugin-free.
- All autocmds live in the `pair` augroup (`clear=true`), so iterating via `:luafile $PAIR_HOME/nvim/init.lua` reloads cleanly without duplicating handlers.

### Prompt history & queue (issue #000015)

The nvim buffer is a virtual cursor over a sequence of slots:

```
[ -N ... -2  -1 ]   *   [ +1  +2 ... +M ]
   history (log)    draft     queue (future)
```

The status line shows position state: ` H < pos[*] > Q ` — `H` = log entry count, `Q` = queue size, `pos` = `*` | `-N` | `+N`. A trailing `*` on `-N` means the buffer differs from the loaded baseline (a pending fork).

**Slot mutability is the central distinction:**

| Slot | Storage | Mutable? | Edit autosave? |
|---|---|---|---|
| `*` | `draft-<tag>.md` | yes | yes (existing autocmd) |
| `+N` | `queue-<tag>/NNNNNN.md` | yes | yes (same autocmd) |
| `-N` | parsed from `log-<tag>.md` | **no — immutable** | no; edit becomes a pending fork |

**Navigation (Alt+←/→):** on navigate-away from a mutable slot, the buffer is autosaved to its underlying file. On navigate-away from a dirty `-N`, a single-line prompt fires:

```
(S)end, (Q)ueue, (D)iscard, [S]tay:
```

- **s/S** — Send the fork (append to log), return to `*`.
- **q/Q** — push to queue front (`+1`), return to `*`.
- **d/D** — drop the edit, proceed with the navigation.
- **anything else (Enter, ESC, ...)** — Stay; cancel the navigation.

`*` is preserved across navigation: when leaving `*`, its content is autosaved, so navigating into history/queue and back never destroys the draft. Sending from `-N` or `+N` also preserves `*` — the "clear the draft" semantic of `Alt+Return` only fires when the source slot was `*`.

**Queue store:** `queue-<tag>/` directory of one file per queued prompt. Filenames are 6-digit zero-padded sortable keys; sort order = display order (`+1` is the lowest key). New keys at `push_front` decrement the current min; `push_back` increments the current max. Initial midpoint at `500000` to leave room either way.

Implementation in `nvim/init.lua`: see helpers grouped under `is_dirty_history_slot`, `autosave_current_slot`, `leave_dirty_history`, `go_to`, `nav_left`/`nav_right`, `queue_current`, plus the `queue_*` file ops. State lives in module-local `nav = { pos, baseline }` — `pos` is `'*'` or `{ kind='history'|'queue', n=N }`.

## Quit semantics

Two ways to end a session, with different aftermath:

- **Alt+d** — detach. The session keeps running (claude/nvim processes alive); `pair` surfaces it in the picker for re-attach.
- **Alt+x** — full quit. Kills the session AND removes the resurrect entry. After Alt+x, the session is fully gone.

Zellij's default `Ctrl+q` (Quit with resurrect) is **unbound** in pair's config — it would otherwise leave a half-state where the processes inside die but the session record stays as a "resurrect candidate," which is confusing for pair's long-lived-agent model. Alt+x is the only quit path.

## Data layout

Drafts and prompt history live under `${XDG_DATA_HOME:-~/.local/share}/pair/` (per XDG Base Directory spec), keyed by tag (the agent name, or a custom name from the create-flow prompt):

- `draft-<tag>.md` — the active draft file (the `*` slot). Cleared by `send_and_clear` only when sending from `*`, persists across launches and navigation.
- `log-<tag>.md` — append-only log of every send, with timestamp. Doubles as the source for the `-N` history slots (parsed at navigation time). Searchable via `rg`.
- `queue-<tag>/NNNNNN.md` — one file per queued prompt (the `+N` slots). Filenames sort to display order (lowest = `+1`). Created lazily by `Alt+q` or auto-front-push from a dirty-`-N` "Queue" choice. Removed when the corresponding queue item is sent.
- `quote-<tag>` — transient hand-off file written by `bin/clipboard-to-pane.sh` and read by nvim's `PairPasteQuote()`. Overwritten on every selection.

The launcher exports `$PAIR_DATA_DIR` so `nvim/init.lua` can compute the same path without re-deriving the XDG fallback chain.

Per-tag files mean `pair claude`, `pair codex`, and a custom-named `pair-bugfix` (entered at the prompt) all have independent draft state.

Internal: `~/.cache/pair/quit-<session>` — marker file used to communicate "user asked for full quit" between `pair-quit.sh` and the launcher. Touched on Alt+x, removed by the launcher after delete-session.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/outer-tty-<tag>` — single-line file containing the path to pair's controlling TTY at attach time. Read by `pair-notify` to emit OSC escapes that reach the outer terminal/wrapper. Rewritten on every attach (create or reattach); removed on full quit.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.

**Migration from v1:** the launcher detects old `~/scratch/pair-{draft,log}-*.md` files on startup and moves them to the new XDG location, stripping the redundant `pair-` prefix from filenames.

## Path resolution

`bin/pair` prepends `$PAIR_HOME/bin` to `$PATH` before exec'ing zellij. zellij and all its child processes (panes, copy_command, Run actions) inherit the PATH and can resolve `clipboard-to-pane.sh`, `copy-on-select.sh`, `pair-quit.sh` by bare name. This lets the zellij KDL configs reference scripts without `sh -c` env-var quoting hacks.

## Design intent

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit — agent owns *output*, nvim owns *input* — and lets each side specialize.
- **Selection is the gesture.** Click-and-drag in the agent pane, mouse up — the quote is in nvim, ready for your reaction. No keystroke between.
- **Self-contained.** Uses `--config-dir` and `nvim -u` to fully isolate from the user's normal configs. No invasive install.
- **Agent-agnostic.** Same plumbing works for any TUI agent that accepts typed input. Switching is one keystroke.
- **Prompt history is just a markdown file.** Aligns with the "data into central location, shell-ed agent runs free" pattern: every send appends to a grep-able log.

## Future work

Tracked in workshop issues. v2 candidates include a real nvim plugin (for users who want LSP/snippets/telescope inside the input pane) and per-agent reference templates for Alt+i (gemini and codex use different image-attach naming schemes than claude's `[Image #N]`).
