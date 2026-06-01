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
bin/pair-restart.sh          # invoked by Alt+n / Shift+Alt+N — marks (quit + restart) + kills session
bin/pair-session-watch.sh    # captures codex/gemini session id at create time (#000016, #000020)
bin/pair-wrap                # PTY proxy: OSC translation + scrollback capture
bin/pair-notify              # hook-driven OSC notifier (e.g. claude Notification)
bin/pair-scrollback-render   # raw PTY capture → ANSI-colored line dump (#000017)
bin/pair-scrollback-open     # Alt+/ orchestrator: render + open viewer
nvim/init.lua                # bundled nvim config (loaded via -u)
nvim/scrollback.lua          # read-only ANSI viewer for the scrollback dump
zellij/config.kdl            # mouse, copy_command, keybinds, pane frames
zellij/layouts/main.kdl      # the split + agent/draft commands + swap layouts
```

### `bin/pair` — launcher

Resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, parses argv — first positional is `$PAIR_AGENT` (default `claude`), everything after `--` is joined into `$PAIR_AGENT_ARGS`, extra positionals before `--` are an error with a usage hint, defaults `$PAIR_TAG` to the cwd basename (the create-flow prompt or `pair resume <tag>` overrides it), resolves `$PAIR_DATA_DIR` to `${XDG_DATA_HOME:-$HOME/.local/share}/pair`, runs a one-time migration of any old `~/scratch/pair-{draft,log}-*` files, and dispatches:

A leading `pair resume <tag>` is recognized as a subcommand verb (alongside `list` / `help`): it skips both the picker and the name prompt, attaches if `pair-<tag>` already exists in any state, otherwise creates with that tag. When `resume` is in play, the agent is inferred from saved state on disk (`agent-<tag>` for live/recently-detached sessions; the agent embedded in the `config-<tag>-<agent>.json` filename otherwise) — so a single tag is enough to restart, regardless of which agent was originally paired with it. See "Tag-restart" below.

**Decision tree.** Finds *all* detached pair-* sessions on the machine (any agent, any naming). It also surfaces **historical tags from this cwd** (#000024): tags named `<cwd-base>` or `<cwd-base>-<subproject>` whose `draft-` / `log-` sidecars in `$DATA_DIR/` were touched within the last `$PAIR_HISTORY_DAYS` (default 14) but no longer have a live session. Convention-only — operators are expected to name sessions `<cwd-base>-<subproject>` so they appear in the right cwd's picker. Then:

- 0 detached + 0 historical → run create flow directly (validate agent, prompt for name, create).
- ≥1 detached or ≥1 historical → fzf picker over the detached sessions, then historical rows annotated `(Nd ago, no live session)`, then a `+ new <agent> session` sentinel. Pick a detached row → attach. Pick a historical row → create-by-name (same path as `pair resume <tag>`, which re-uses any saved `draft-<tag>.md` / `config-<tag>-<agent>.json`). Pick the sentinel → fall through to create with `free_slot_tag`. `PAIR_DEBUG_HISTORY=1 pair` exits early printing the scan results — useful for sanity-checking the cwd-prefix convention on a given data dir.

The agent argument doesn't filter the picker — reattach is agent-agnostic (the existing session already runs whatever it runs). The agent argument only matters for the create path: it labels the sentinel, drives the auto-suggested default name, and is the binary that gets exec'd in the new session.

There is **no silent auto-attach**. Every reattach goes through the picker so the user explicitly sees what they're connecting to.

Detection of attached-vs-detached uses `zellij --session NAME action list-clients`, which prints a header plus one row per connected client. Zero rows = detached.

**Naming prompt.** When the create flow runs, the launcher prompts the user with the auto-suggested tag as the default — the cwd basename, sanitized (so `~/workspace/pair` → `Session name: pair`). The prompt is editable inline (delegated to zsh's `vared` since bash 3.2 has no `read -i`). The `pair-` prefix is implicit — the prompt shows just the tag, since `pair-` is always prepended. Pressing Enter accepts; typing a custom name (`bugfix`, or `pair-bugfix` — leading `pair-` is stripped) overrides it. `pair resume <tag>` skips this prompt entirely.

**Agent validation deferred.** `command -v "$AGENT"` runs only inside the create branch, not at startup, so attaching to a custom-named session whose tag isn't a real binary still works.

**Title.** The launcher emits an OSC 0 escape sequence right before invoking zellij, so the terminal title shows the session name on both create and attach paths (zellij itself only sets it on create).

**Cleanup on quit.** zellij is run as a child (not `exec`) so the launcher resumes when zellij exits. On resume it checks for `~/.cache/pair/quit-<session>` (the marker that `pair-quit.sh` writes when Alt+x fires) and, if present, runs `zellij delete-session --force <session>` to clear the resurrect entry. It then SIGKILLs any leftover children that didn't follow the session down: a lingering `zellij --server` (rare but seen), and `nvim --embed` orphans (every `nvim FILE` is internally TUI parent + embed child; the embed sometimes survives RPC-pipe EOF and gets reparented to launchd). The embed reap is two-layered — primary path reads `nvim-pid-<tag>-{draft,scrollback}` files written by VimEnter autocmds inside `nvim/init.lua` and `nvim/scrollback.lua` (so the embed pid is known deterministically); fallback is a tag-scoped `pkill -f`. If a `config-<tag>-<agent>.json` was captured during the session, it also prints a one-liner naming the resume command (`pair resume <session>`) so the user can pick the work back up later. No marker → leave the session as zellij left it (running if Alt+d detached).

**Startup orphan sweep.** The Alt+x reaper only runs when the user quit through pair. External terminations (`zellij kill-session`, host reboot during a session, pair upgrade mid-session) leave the embed orphaned with no marker. `sweep_orphan_nvim` runs once per `pair` invocation, just after the live pair-session list is computed: it collects candidate tags from both pidfiles and the argv of every running `nvim --embed` referencing `$DATA_DIR/`, then calls `reap_nvim_for_tag` on any tag whose `pair-<tag>` session is no longer alive. The argv walk is what catches embeds with no pidfile (autocmd errored before VimEnter, or panes that predate the autocmd). The same `reap_nvim_for_tag` is shared with `cleanup_quit_marker`, so there's exactly one reaper definition; adding a new nvim surface in pair means routing it through `$PAIR_NVIM_PID_FILE` and naming it under `$DATA_DIR/{draft,scrollback}-<tag>...`, not extending the reaper.

**Reload / restart in place (Alt+n, Shift+Alt+N).** A second marker, `~/.cache/pair/restart-<session>`, is written alongside `quit-` by `bin/pair-restart.sh`, carrying the agent name + a `new_session` flag. After cleanup_quit_marker tears the session down, `handle_restart_marker` reads the marker and `exec`s pair on itself with `PAIR_FORCE_TAG=<same-tag>` set in the env (pins the new run to the killed session's tag, skipping both the picker and the name prompt). The flag controls what happens to the saved config:

- `new_session=0` (Alt+n) — keep `config-<tag>-<agent>.json`. Append the agent-appropriate resume token to the re-exec'd argv: `--resume <id>` for claude/gemini, `resume <id>` for codex. Result: pure pair reload — same tag, same draft, same agent conversation. Useful after a binary or config rebuild.
- `new_session=1` (Shift+Alt+N) — drop `config-<tag>-<agent>.json` so the next launch's claude `--session-id` injection (or the codex/gemini watcher) writes a brand-new entry. Result: fresh agent conversation, same tag and draft.

The picker is bypassed in either flavor — Alt+n's argv carries an explicit resume token, and Shift+Alt+N has no saved config to pick against.

### `zellij/layouts/main.kdl` — pane split + swap-layout ladder

Horizontal split. Top pane runs `$PAIR_AGENT $PAIR_AGENT_ARGS` (auto-fills remaining height). Bottom pane is `size=12` (fixed 12 rows) running `nvim -u $PAIR_HOME/nvim/init.lua` on the per-tag draft file. Integer sizes are FIXED in zellij (refusing the `resize` action), but pair drives all rung changes through swap layouts, not resize, so FIXED is harmless.

Both panes wrap their command in `sh -c "..."` so the shell expands `$PAIR_AGENT`, `$PAIR_AGENT_ARGS`, `$PAIR_TAG`, and `$PAIR_HOME` at exec time — zellij itself does not interpolate env vars in `command`/`args` fields.

`$PAIR_AGENT_ARGS` is appended on the agent pane command line as a single space-separated string; the shell word-splits it. Args containing spaces are *not* preserved (rare for CLI flags; documented in README).

The bottom pane has `focus=true` (drafting pane gets focus on launch), `borderless=true` (so the `minimized` rung can collapse to 1 row — see "pane frame asymmetry" below), and `name="draft"` — used by zellij in the OSC 0 terminal title (`pair-<tag>: draft`) which propagates to the user's terminal/multiplexer tab title. The draft is borderless so it has no frame title slot; the keybind cheatsheet that used to live in the frame title lives in nvim's statusline (right-aligned, see `nvim/init.lua`).

**Pane frame asymmetry.** `pane_frames true` is set globally in `zellij/config.kdl` so the **agent pane** renders a frame — the value is the scroll-position indicator zellij draws in the top-right of a framed pane (e.g. `500/540`), which is the only way to see scrollback position (zellij doesn't expose scroll offset to plugins or the CLI). The **draft pane** opts out via `borderless=true` in every layout (default + both swap layouts), because a framed pane has a ~3-row minimum and the `minimized` rung needs `size=1`. Cost: the agent pane loses 2 rows + 2 cols to the frame chrome.

**Swap layouts.** Two `swap_tiled_layout` entries — `minimized` (draft `size=1`) and `half` (draft `size="50%"`) — sit alongside the default layout above. Each is gated by `exact_panes=2` so it only applies when the current pane structure matches what pair builds. `nvim/init.lua` drives them via `zellij action next-swap-layout` / `previous-swap-layout`, which re-tile the existing agent + nvim panes onto the target layout positionally — running pane processes (`pair-wrap`, `nvim`) survive each swap. Cycle from default(small) is `[minimized, half]`: `next-swap-layout` from small → minimized, from minimized → half, from half → wraps to small. The lua side maps Alt+Down to next-swap (smaller rung) and Alt+Up to prev-swap (bigger rung), with a state-machine clamp at the rung extremes.

### `zellij/config.kdl` — mouse, copy, keybinds

Top-level config:

- `mouse_click_through true` — first click on an unfocused pane goes through to the pane (so click-and-drag selects in one motion) instead of being consumed by zellij just to change focus.
- `copy_command "copy-on-select.sh"` — on every selection finalize (mouse-up after drag), zellij pipes the selected text to this script. `copy_command` replaces zellij's default OS-clipboard write, so the script does that part too. Resolved by PATH (which `bin/pair` populated).
- `pane_frames true` — frames are enabled globally so the agent pane shows zellij's scroll-position indicator (top-right of the frame) when scrolled. The draft pane opts out via `borderless=true` in `zellij/layouts/main.kdl` so the `minimized` rung can still collapse to 1 row (a framed pane's minimum is ~3 rows). The cheatsheet still renders in nvim's statusline rather than a frame title — the draft has no frame to hold one.

Keybinds added on top of zellij defaults (`clear-defaults=false`):

- `unbind "Alt i"` — release Alt+i (zellij's default binds it to MoveTab; we want nvim to see it for image attach).
- `unbind "Alt n"` — release Alt+n (zellij's default `NewPane` would break pair's two-pane invariant; we rebind it below for restart).
- Mode-locking — every default chord that would switch zellij modes (`Ctrl+g/p/t/n/h/s/o/b`) is unbound, and `Ctrl+q` (zellij's resurrect-leaving Quit) is unbound too — Alt+x is the only quit path.
- `Alt+d` — routed through nvim to `:lua PairConfirmDetach()` — Y/N modal then detach.
- `Alt+x` — routed through nvim to `:lua PairConfirmQuit()` — Y/N modal then `pair-quit.sh` (full quit).
- `Alt+n` — routed through nvim to `:lua PairConfirmRestart()` — Y/N modal then `pair-restart.sh` (reload pair, keep agent session).
- `Shift+Alt+N` — routed through nvim to `:lua PairConfirmRestartNewSession()` — Y/N modal then `pair-restart.sh --new-session` (restart with a fresh agent conversation). See "Reload / restart in place" under `bin/pair`.
- `Alt+h` — `Run "pair-help" { floating true; close_on_exit true; ... }` — pops a floating pane running `pair -h | less`.
- `Alt+↑` / `Alt+↓` — route to nvim's `PairLayoutBigger` / `PairLayoutSmaller` — step the nvim pane along the swap-layout ladder (`minimized ↔ small (12 rows) ↔ half`).
- `Alt+j` — `FocusNextPane` — toggle focus between the agent and draft panes. Works from either pane because it's a global zellij bind, intercepted before the focused pane sees the key. Overrides zellij's default Alt+j (`MoveFocus "Down"`), which only reached the draft and was a dead key once you were already there; the two-pane invariant makes `FocusNextPane` a clean toggle with no direction to track.

The Alt+x/d/n confirms route through nvim rather than running directly so a single fat-finger doesn't tear the session down (Alt+x in particular is unrecoverable). The lua side also auto-grows out of `minimized` before showing the modal, since otherwise the prompt would land on a 1-row pane where nothing is visible.

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

Touches the marker file `~/.cache/pair/quit-$ZELLIJ_SESSION_NAME`, then `exec zellij kill-session $ZELLIJ_SESSION_NAME`. The kill terminates the session including the script itself; on the launcher side, `bin/pair` resumes, sees the marker, and runs `delete-session --force` to clean up the resurrect entry.

Alt+x leaves the draft, queue, and history intact — the next session resumes them. Use Shift+Alt+Backspace (`forget_all`) for the destructive "start anew" path.

### Outer-TTY capture and notification routing — `bin/pair-wrap`, `bin/pair-notify`

**Why.** Zellij parses every escape on the way out for its virtual-screen reconstruction and drops sequences it doesn't recognize. OSC 9 and OSC 777 (the notification escapes outer wrappers like cmux watch for) fall in that bucket and never reach the host terminal. BEL is forwarded since zellij 0.44, but cmux specifically watches OSC, not BEL — so BEL forwarding doesn't help that integration. Filed as #000011.

**Mechanism, in two layers:**

1. **Outer-TTY capture (in `bin/pair`).** Before invoking zellij, on every attach (both create and reattach branches), pair calls `tty(1)`. The result is the path of pair's controlling TTY — which is precisely the outer PTY (the one allocated by whatever wraps pair: cmux, a terminal emulator, etc.). That path gets written to `$DATA_DIR/outer-tty-<tag>`. Refreshed on every attach because the outer PTY changes across detach/reattach, while pane-shell env stays frozen at zellij session-creation time (env-var approaches would go stale).

2. **Two consumers** of the captured path:

   - **`bin/pair-wrap`** (Go, `cmd/pair-wrap`). Transparent PTY proxy. The zellij agent pane runs `pair-wrap $PAIR_AGENT $PAIR_AGENT_ARGS` instead of the agent directly (wired in `zellij/layouts/main.kdl`). The wrapper allocates a fresh PTY for the agent, forwards stdin/stdout transparently with SIGWINCH propagation, and watches the agent's output stream for OSC notifications. On detection it writes OSC 9 directly to the recorded outer-TTY path — bypassing zellij.

     **Stdin raw mode.** The wrapper switches its stdin (zellij's pane PTY) into termios raw mode for the duration. Without this the kernel's line discipline does local echo + canonical buffering on the bytes flowing toward the wrapped TUI, which double-echoes keystrokes and corrupts terminal-response sequences. Saved/restored in a `finally` block.

     **Stdin Enter remap (per-agent).** `sendKeymapByAgent` (`cmd/pair-wrap/main.go`) translates the user's Enter / Alt+Enter to per-agent send/newline bytes so the convention matches pair's nvim draft pane (Enter = newline, Alt+Enter = send). For `claude` the user's plain Enter becomes `\<CR>` (claude's portable "insert newline" sequence); Alt+Enter becomes a bare `\r` (send). Codex / Gemini use the textbook `Enter=send, Shift+Enter=newline`. Opt out with `PAIR_WRAP_REMAP_RETURN=0`.

     **Stdout filtering (Codex).** Codex inline mode emits DEC synchronized-output markers (`ESC[?2026h` / `ESC[?2026l`) around frequent redraw batches. `pair-wrap` strips those markers from the stdout stream sent to zellij, because zellij scrollback/mouse scrolling can behave poorly while a pane is in synchronized-output mode during generation. The raw scrollback log remains unfiltered so forensic replay still captures the agent's original PTY stream.

     **Overlay-aware suspension (per-agent).** Textarea Enter remaps are wrong while a blocking overlay / picker has focus: the overlay needs a bare `\r` to confirm the highlighted option. pair-wrap registers per-agent overlay detectors in `overlayDetectorByAgent`, sets `pickerActive` when one fires, and emits a bare `\r` for the next plain Enter only. The flag clears after that one Enter, so normal textarea remapping resumes for the following keystroke. Claude uses the stable `OSC 777;notify;Claude Code;Claude needs your permission` body. Codex question prompts use `OSC 9;Plan mode prompt:...`; other Codex pickers fall back to stripped visible output plus a short text carryover watching for labels such as `Use session directory (` / `Use current directory (`, `Press enter to continue`, and `Press enter to confirm or esc to go back`. The carryover is cleared when the confirming Enter is consumed so stale picker text cannot re-arm the flag. Known edge inherited from the one-shot design: dismissing an overlay without Enter leaves the flag set until the next plain Enter.

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

### Colored scrollback dump — `pair-wrap`, `pair-scrollback-render`, `pair-scrollback-open`, `nvim/scrollback.lua`

**Why.** zellij now renders a frame on the agent pane, which surfaces a scroll-position indicator (e.g. `500/540`) in the top-right. Knowing the position is half the value — the other half is being able to *jump back* to a remembered line. zellij's built-in `EditScrollback` strips ANSI styles when dumping (its scrollback is a styled cell grid internally, but the dump is plain text) and opens in a new tiled pane that breaks pair's two-pane invariant. Filed as #000017.

**Capture (in `pair-wrap`).** When invoked with `--scrollback-log <path>`, pair-wrap opens `<path>` (truncated) and tees every chunk read from the agent's master PTY into it. Alongside it, `<path-without-.raw>.events.jsonl` collects one JSON line per resize:

```
{"type":"resize","offset":<bytes>,"cols":N,"rows":N}
```

The existing `set_winsize()` is the single entry point for both the initial PTY size (called once after `pty.fork`) and every SIGWINCH (the registered handler). Threading `log_scrollback_event()` through it covers both. `SCROLLBACK_BYTES` is bumped after each successful write to the raw fd, so the offset on each resize event demarcates "from this byte onward, apply these new (cols, rows)" — which is what the renderer needs to replay each segment at its correct width. Failure mode is unchanged: any tee or sidecar write error is `debug()`-logged and swallowed; the proxy never blocks the agent on a logging hiccup. `zellij/layouts/main.kdl` passes the flag by default, so capture runs automatically for every pair session.

**Replay (`bin/pair-scrollback-render`, Go).** Reads `<raw>` and `<events.jsonl>`, feeds the bytes to a `charmbracelet/x/vt` emulator in segments delimited by resize events. The emulator runs the same VT100 interpretation zellij does live (width-based wrap, alternate-screen flips, scroll regions), so its row count matches what the user saw in zellij's indicator. After feeding, the renderer walks the scrolled-out history followed by the visible buffer, and emits one ANSI-decorated line per row to `<out.ansi>`: full-reset SGR + per-row attrs + the row's characters + `\x1b[0m`. Built into `bin/pair-scrollback-render` via `make pair-scrollback-render`; single static binary, no runtime dep.

**Viewer (`nvim/scrollback.lua`).** Plugin-free init loaded via `nvim -u`. On `BufReadPost`, an SGR state machine walks each line: peels every `\x1b[...m` escape, mutates a running state (fg/bg/bold/italic/underline/reverse/strike/blink), and emits an extmark span for each contiguous run of visible bytes under a single state. Color resolution: 30-37/90-97 fg + 40-47/100-107 bg map through an xterm-default palette; `38;5;n` indexed maps via the standard 256-color formula (16 anchored to the same palette, 16-231 = 6×6×6 cube, 232-255 = greyscale ramp); `38;2;r;g;b` uses RGB directly. State→hl-group cache is keyed by stringified attrs and uses an explicit counter (not `#hl_cache` — that's 0 on string-keyed tables, a bug caught during the test pass). Buffer is locked read-only (`modifiable = false`, `buftype = nofile`, no swapfile); only `<Esc>` quits via `<cmd>qa<CR>` — `q`, `ZZ`, `ZQ` are deliberately shadowed so a fat-fingered `q` (instead of `Alt+q` for the marker comment) can't slam the viewer shut and drop pending markers.

**Open (`bin/pair-scrollback-open`, POSIX sh).** Validates `PAIR_DATA_DIR` / `PAIR_TAG` / `PAIR_AGENT`, runs the renderer, then `exec`s `nvim -u $PAIR_HOME/nvim/scrollback.lua $ANSI`. Errors print and `sleep` briefly so the message is readable before the floating pane self-closes. Bound in `zellij/config.kdl` to `Alt+/` as a 100% × 100% floating pane with `close_on_exit=true` — the user's `:q` in the viewer dismisses the pane and returns to pair's two-pane layout untouched.

**Jump-on-open shortcut — draft `Alt+b` = "Alt+/ then Alt+b".** `pair-scrollback-open` takes an optional `--jump prev|next`; it exports `PAIR_SCROLLBACK_JUMP` before exec'ing nvim, and `scrollback.lua` calls `jump_to_prompt()` right after its normal viewport positioning — so the viewer opens already sitting on the previous (or next) user prompt, behaviourally identical to opening with Alt+/ and then pressing Alt+b. The draft pane's `Alt+b` (`nvim/init.lua`, `pair_scrollback_prev_prompt`) is the one-key trigger: it opens the same floating pane via `zellij run --floating … -- pair-scrollback-open --jump prev` (geometry mirrored from the `Alt+/` bind). Env-scoped rather than a sentinel file, so there's no staleness across plain `Alt+/` opens.

**Comment markers — `Alt+q` in viewer → draft pickup (#000018).** While reading scrollback, `Alt+q` drops a parley-style `🤖[]` marker at the cursor (or `🤖<selection>[…]` in visual mode). The buffer is read-only, so the keymap lifts `modifiable`/`readonly` for the insert and re-locks immediately. On viewer exit (`VimLeavePre`), `nvim/scrollback.lua` walks every line, parses each `🤖<X>?[Y]` marker by literal-byte scan (Lua patterns aren't UTF-8 aware), and writes a formatted block to `$PAIR_DATA_DIR/scrollback-pending-<tag>.md`:

```
> <X | line stripped of all markers>
<Y>
```

The draft pane's `nvim/init.lua` registers a `FocusGained` autocmd that picks up the sidecar: on the `*` slot, it appends the block directly into the buffer and `:write`s (going through nvim_buf_set_lines, not an autoread + checktime dance, sidesteps the sub-second mtime resolution issue). Off-slot (`-N` / `+N`), it appends to `draft-<tag>.md` so the next nav-to-`*` reads it from disk. Sidecar is removed in both cases, and a `vim.notify` flashes "🤖 picked up N scrollback comment(s)". Round-trip: read scrollback → `Alt+q` to mark → `:q` → focus the draft → see the formatted block ready to send via `Alt+Return`.

**Overall comment affordance (#000021).** After inline annotations, users often want a standalone summary not tied to any line. `BufReadPost` appends one trailing row — `For overall comment, Alt+q on this line.` — rendered in default Normal color (not dimmed; the affordance is positional, not visual). `nvim`'s `virt_lines` aren't cursor-navigable, so this is a real line. `Alt+q` on that row routes to `add_footer_comment` (no inline-quote context) and stores the text in a module-local `footer_text_by_buf[bufnr]`; the visible row becomes `Overall comment: <text>` and edits via the same chord. Empty submit clears, restoring the hint. `emit_pending` strips the affordance row from the marker scan and appends the stored text as a trailing standalone block (no `> quote` prefix) in the sidecar. The Esc exit-confirm folds it into the prompt ("3 🤖[] markers + overall comment will be sent").

### `nvim/init.lua` — drafting buffer config

Loaded via `nvim -u`, fully isolated from the user's main nvim config. Provides:

- Drafting-friendly defaults: no line numbers, wrap, linebreak, breakindent, spell, persistent undo under `~/.local/share/pair/undo/`, `cmdheight=0` to keep the cmdline out of the way, custom statusline (see "prompt history & queue" below).
- `<M-CR>` (Alt+Return, normal+insert) — `send_and_clear`: append buffer to log, send to agent pane via `zellij action focus-pane-id` + `write-chars` + Enter, clear `*` (when source was `*`, or when a send from `+N` parked a non-empty draft into the queue — see "Prompt history & queue"), save, drop into insert mode.
- `<M-Left>` / `<M-Right>` — navigate the prompt-history / queue position one slot at a time (see below).
- `<S-M-Left>` / `<S-M-Right>` — jump to the next region boundary (oldest history, newest history, `*`, front-of-queue, back-of-queue). Lets the user skip over long histories or queues without N taps.
- `<M-b>` — `pair_scrollback_prev_prompt`: open the scrollback viewer already positioned on the previous agent-conversation prompt — a one-key shortcut for `Alt+/` then `Alt+b`. Shells out `zellij run --floating … -- pair-scrollback-open --jump prev`. See the scrollback section's "Jump-on-open shortcut".
- `<M-q>` — push the current buffer to the front of the queue. From `*` also clears `*`; from `+N` it's move-to-front (removes the source queue file).
- `<M-BS>` — delete the current `+N` queue item without sending; "stay-near" behavior (items behind shift down, position label keeps its number, so the next item is now under the cursor for repeat-delete). No-op from `*` or `-N`.
- `<M-i>` (Alt+i, normal+insert) — `attach_image`: capture-driven image attach. 1) Verify the OS clipboard holds image data (macOS: AppleScript `clipboard info` enumerates `PNGf`/`TIFF`/etc.; Linux: `wl-paste --list-types` or `xclip -t TARGETS`) — if not, flash `[no image in clipboard]` as inline virt_text for 1s and bail. 2) Read pair-wrap's pid from `$DATA_DIR/pair-wrap-pid-<tag>` (notify+abort if missing/dead, since pair-wrap is the whole agent I/O path). 3) `kill -USR1 <pid>` to arm a ~200ms capture window in pair-wrap, then `zellij action write 22` to send Ctrl+V to the agent pane. 4) Poll `image-capture-<tag>.done` (20ms cadence, 600ms cap); on hit, read `image-capture-<tag>`, strip ANSI, regex `%[Image[ #][^%]]+%]` (matches both claude's `[Image #N]` and gemini's `[Image N-M]`) and insert the captured marker verbatim at cursor. The agent is the source of truth for the marker text — no local counter, no per-agent format hardcoded.
- `PairPasteQuote()` (global, called from `bin/clipboard-to-pane.sh` via `:lua PairPasteQuote()`): reads the raw selection from `$PAIR_DATA_DIR/quote-<tag>` and dispatches on cursor column.
  - **col == 0 (`paste_as_quote`)**: par-reflow with width 1000, prefix every line with `> `; if the cursor's line is empty, replace it, else insert above (existing line slides down); scroll first inserted line to top via `zt`; cursor on a single empty line directly below the block in insert mode; flash the quoted lines with `IncSearch` (full-line, per-line `nvim_buf_add_highlight`).
  - **col > 0 (`paste_inline`)**: par-reflow (so hard-wrapped sources collapse to one continuous run, paragraph breaks preserved), insert at the cursor via `nvim_buf_set_text` (handles multi-line splits); cursor at the end of the inserted span in insert mode; no scroll; flash the inserted span with a single multi-line extmark.
  - In both modes the highlight is cleared 500ms later via `vim.defer_fn`. Selection-finalize visual cue (issue #12).
- Autosave on `BufLeave`, `FocusLost`, `InsertLeave` so disk and buffer agree.
- As-you-type fuzzy path completion (issue #13). `TextChangedI`/`TextChangedP` autocmd splits the trailing path token on the last `/` into `<dir>` + `<filter>`, lists `<dir>` via `getcompletion`, fuzzy-filters with built-in `matchfuzzy`, hands the result to `vim.fn.complete()`. Triggers only when the token contains `/` or starts with `~` (plain words stay quiet). `<Tab>`/`<S-Tab>` cycle, `<CR>` accepts when an item is selected (else newline). Plugin-free.
- All autocmds live in the `pair` augroup (`clear=true`), so iterating via `:luafile $PAIR_HOME/nvim/init.lua` reloads cleanly without duplicating handlers.
- **Layout ladder** — `PairLayoutBigger` / `PairLayoutSmaller` derive the current rung from `vim.o.lines` (the kdl pins each rung to an exact size — 1 / 12 / 50% — so nvim's pane height is ground truth) and call `zellij action next-swap-layout` / `previous-swap-layout` accordingly. Reading actual height makes drift self-correcting: a silently-rejected swap can't desync state, since the next press recomputes from reality rather than a counter that was incremented optimistically. `pair_layout_state` mirrors the rung in-memory for callers like `pair_spinner_start` and `pair_ensure_visible_then` to check without re-reading; an on-disk copy at `${XDG_DATA_HOME:-~/.local/share}/pair/layout-mode-<tag>` is purely diagnostic. Landing in `minimized` also `MoveFocus`es up to the agent pane (the draft is unusable at 1 row) and the focus-grab spinner suppresses itself when `pair_layout_state == 'minimized'`.
- **Statusline cheatsheet (right-aligned, progressive disclosure).** `PAIR_CHEATS` lists `Alt+h help`, `Alt+⏎ send`, `Alt+q queue`, `Alt+x quit`, `Alt+d detach` in priority order. `pair_compose_statusline` measures the variable left segment (history/queue/position cluster), reserves a 6-cell minimum gap, and accumulates as many cheat entries as fit in the remaining columns — Alt+h is always the last entry to drop. Spinner takes the right slot when active (vim only honors a single `%=` per statusline). The minimized rung shows a standalone "Alt+↑ for pair input box" hint instead, with 4 leading spaces so the terminal cursor (which lands on the statusline row when the buffer has zero visible lines) sits on whitespace rather than the hint text.
- **Alt+x / Alt+d / Alt+n / Shift+Alt+N confirm modals.** `PairConfirmQuit` / `PairConfirmDetach` / `PairConfirmRestart` / `PairConfirmRestartNewSession` shell out to `pair-quit.sh` / `zellij action detach` / `pair-restart.sh` / `pair-restart.sh --new-session` after a Y/N modal that defaults to No. All four are wrapped in `pair_ensure_visible_then`, which auto-grows out of `minimized` (calls `PairLayoutBigger` and defers the modal 100ms) so the prompt renders on visible rows. The two restart modals share a single `pair_confirm_restart_impl(new_session)` helper.

### Prompt history & queue (issue #000015)

The nvim buffer is a virtual cursor over a sequence of slots:

```
[ -N ... -2  -1 ]   *   [ +1  +2 ... +M ]
   history (log)    draft     queue (future)
```

The status line shows position state:

```
 Alt: <- history H < pos[*][ (⌫=del)] > Q queued -> 
```

- `H` / `Q` = total counts of history / queue entries.
- `pos` = `*` | `-N` | `+N`.
- Trailing `*` on `-N` means the buffer differs from the loaded baseline (a pending fork awaiting `Send` / `Queue` / `Discard`).
- A contextual `[key=action]` hint appears inside the brackets — `[q=queue]` on `*`/`-N` (Alt+q parks/forks to queue front), `[⌫=del]` on `+N` (Alt+BS deletes the item). Bracket convention: TUI status-bar "key badge" idiom (`[Esc] cancel` etc.). Distinct from the prompt convention `( ) [ ]` for access-key-vs-default, which only applies to interactive dialogs.
- The flanking `<-` / `->` text and the `Alt:` prefix make the navigation gesture self-documenting (Alt+← / Alt+→).
- Highlight is linked to `Comment` rather than the default inverted `StatusLine` so the bar reads as muted secondary info; reapplied on `ColorScheme`.

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

`*` is preserved across navigation: when leaving `*`, its content is autosaved, so navigating into history/queue and back never destroys the draft. Sending from `-N` preserves `*` (the "clear the draft" semantic of `Alt+Return` only fires when the source slot was `*`). **Sending from `+N` while `*` holds an in-progress draft parks that draft as a queue item (`push_front`) before shipping the selected item** — so `*` ends up empty (sent item's stickies + a fresh line) and the WIP survives as the new `+1`, rather than dangling at `*`. The selected item is resolved by its filename **key captured before** the enqueue, never by the display index: the `push_front` shifts every index by one, and removing by a stale index is what previously left the sent item in *both* `+N` and `-1` (duplication). Regression-guarded by `tests/queue-send-test.sh` (`make test-queue`). Empty / comment-only drafts have nothing to park, so that case is unchanged.

**Queue store:** `queue-<tag>/` directory of one file per queued prompt. Filenames are 6-digit zero-padded sortable keys; sort order = display order (`+1` is the lowest key). New keys at `push_front` decrement the current min; `push_back` increments the current max. Initial midpoint at `500000` to leave room either way.

**Forget-all (Shift+Alt+BS):** wipes `log-<tag>.md`, `draft-<tag>.md`, and every file in `queue-<tag>/` after a confirmation prompt that defaults to No. Hard delete, not an archive — symmetric with the per-item `Alt+BS` queue delete. The confirm-default-No is the safety: a stray Shift+Alt+BS doesn't nuke the session.

**Comments (`=== ...`):** whole lines matching `^%s*===` are stripped from the body at send time only. Draft, queue, and log files store the unstripped text so annotations survive history navigation. A comment-only prompt is a silent no-op (no queue consumption, no log append). Stripping is line-based and not fence-aware. Implementation: `strip_comments` in `nvim/init.lua`, called from `send_and_clear` and `ship_buffer_and_reset`.

**Comment-only edits to history are persisted.** `is_dirty_history_slot` compares `strip_comments(buffer)` against `strip_comments(baseline)`, so adding/removing comments on a `-N` slot doesn't read as a fork. `autosave_current_slot` then rewrites the corresponding log entry's body in place (preserving its timestamp header) via `write_history_entry`. Real forks — anything that changes the stripped body — are still left unsaved so the next `go_to` raises the Send/Queue/Discard/Stay prompt.

Implementation in `nvim/init.lua`: see helpers grouped under `is_dirty_history_slot`, `autosave_current_slot`, `leave_dirty_history`, `go_to`, `nav_left`/`nav_right`, `nav_boundary` (Shift+Alt jumps), `queue_current`, `delete_current_queue_item`, `forget_all`, plus the `queue_*` file ops. State lives in module-local `nav = { pos, baseline }` — `pos` is `'*'` or `{ kind='history'|'queue', n=N }`.

**Insert-mode-only auto-insert from mouse selection.** `bin/copy-on-select.sh` mirrors any selection to the OS clipboard; for selections outside the nvim pane it then triggers `PairPasteQuote` by sending Ctrl-_ (ASCII 31) to the focused nvim pane. The `<C-_>` keymap is bound **only in insert mode**, which is structurally the gate: when the user is in normal mode (e.g. browsing prompt history with Alt+←/→), Ctrl-_ hits nvim's near-no-op default and the buffer isn't mutated. The selection is still on the OS clipboard for manual paste. No mode-probing files or shell-side state needed.

### Auto-orientation slug — `cmd/pair-slug`, pair-wrap trigger, nvim winbar (issue #000027)

When juggling several pair tabs, the `=== comment ===` on draft line 1 (pinned
to the winbar by `pair_pin_header`) tells you what a tab is about — but only if
you remember to type it. This feature auto-maintains it. A **propose / dispose**
split keeps the model out of the live buffer:

- **Trigger** — `pair-wrap`'s turn-end detection (`emitOuter`, the agent-agnostic
  notify sink: marker-regex for claude, idle/native OSC for codex/gemini). On
  turn-end it spawns `pair-slug` in the background (debounced `slugDebounceS`,
  `PAIR_AGENT` set, repo cwd inherited). This is agent-agnostic by design — *not*
  a claude `Stop` hook — so the slug works for every agent and needs no
  `~/.claude` config (pair-wrap wraps every session).
- **Propose** — `cmd/pair-slug` (Go binary). Resolves its own transcript from
  `config-<tag>-<agent>.json` (session_id) + the per-agent path, and parses each
  **native format** into `{role,text}` turns: claude jsonl, codex rollout
  (`response_item`/`payload.message`), gemini json (`messages[]`). Derives the
  left from the git branch (`git -C <cwd>`); asks a small model (`$PAIR_SLUG_MODEL`,
  default `claude-haiku-4-5` via `claude -p`, or `gpt-5.4-mini` when
  `PAIR_AGENT=codex`) for the `<focus>` right over a **user-biased**
  window (`selectWindow` extends back past tool-only turns to include real user
  prompts). Codex uses the direct OpenAI Responses API when `OPENAI_API_KEY` is
  exported; otherwise it shells through `codex exec` so subscription-authenticated
  Codex CLI sessions still work. It writes a validated `=== <branch> | <focus> ===`
  to `slug-proposed-<tag>`. Gates: KEEP keeps the focus but refreshes the left,
  validate-or-keep-last, left always stomped with the authoritative branch.
  `PAIR_SLUG_NESTED` breaks any recursion. Failures are non-fatal.
- **Dispose** — nvim (`nvim/slug.lua`) watches `slug-proposed-<tag>` and applies
  it to draft line 1 only when safe (never touches the prompt below, not
  mid-compose, freeform no-pipe stays manual). An empty draft is an initialization
  case: nvim inserts the slug on line 1, adds a blank line 2, and moves the
  cursor there so composition continues below the header. nvim mirrors the
  effective line 1 back into `slug-<tag>` — the `prev` the proposer reads next
  turn (so a user edit reaches the model, soft policy). Single writer per file
  (proposer→`slug-proposed`, nvim→`slug-<tag>`) makes the channel race-free.

Pure cores are tested: `cmd/pair-slug/slug.go` (normalize/parse/decide) via
`go test`, the nvim decision via `nvim -l` (`make test-lua`). Per-agent parsers
validated against real codex/gemini transcripts.

## Quit / restart semantics

Four ways to end (or refresh) a session, with different aftermath:

- **Alt+d** — detach. The session keeps running (claude/nvim processes alive); `pair` surfaces it in the picker for re-attach.
- **Alt+x** — full quit. Kills the session AND removes the resurrect entry. After Alt+x, the session is fully gone (but the `config-<tag>-<agent>.json` survives, so `pair resume <tag>` later replays the saved launch args + agent session id).
- **Alt+n** — reload pair. Kills the session AND keeps the saved `config-<tag>-<agent>.json` AND re-launches pair on the same tag with the same agent + args + agent session: the conversation resumes via `--resume <id>` (claude/gemini) or `resume <id>` (codex). Pair itself is the only thing that restarts — useful after a binary or config rebuild.
- **Shift+Alt+N** — restart with a fresh agent conversation. Same as Alt+n but drops `config-<tag>-<agent>.json` first, so the relaunched agent starts a brand-new session.

Mechanically Alt+n and Shift+Alt+N share two markers (`quit-` + `restart-`) plus a `PAIR_FORCE_TAG` env var on re-exec; the restart marker carries a `new_session` flag that selects the keep-vs-drop branch. See the launcher's "Reload / restart in place" section.

All three route through a Y/N confirm modal in nvim before firing, so a single fat-finger Alt-key can't tear the session down. The lua side auto-grows the nvim pane out of the `minimized` rung first, so the modal lands on visible rows.

Zellij's default `Ctrl+q` (Quit with resurrect) is **unbound** in pair's config — it would otherwise leave a half-state where the processes inside die but the session record stays as a "resurrect candidate," which is confusing for pair's long-lived-agent model. Alt+x is the only full-quit path.

## Tag-restart (issue #000016)

A pair *tag* is a durable identity for a coding session: it survives Alt+d (detach) trivially, and survives Alt+x because pair captures both the original launch args and the agent's own session id to disk, keyed by `(tag, agent)`. After Alt+x, the user sees a one-liner naming the resume command; running it short-circuits the picker and replays the saved configuration.

**Discovery — two layers.** The session id needs to be on disk by Alt+x time so `pair resume <tag>` can replay it. Two mechanisms, picked by agent and launch shape:

1. **Pre-write at launch (`bin/pair`).** Two paths:
   - `--resume <id>` / `resume <id>` explicit on argv: pair writes `config-<tag>-<agent>.json` directly with that id, before zellij launch.
   - **Claude fresh launch (issue #000020):** claude supports `--session-id <uuid>`, so on the new-session path pair generates a v4 UUID, injects the flag into the agent argv, and writes the config synchronously *before* spawning the watcher. The id is deterministic from the launcher's perspective, so the watcher is a no-op for claude — and the cross-tag race that existed when two pair sessions shared a cwd is structurally eliminated.
2. **Watcher (`bin/pair-session-watch.sh`, codex/gemini only).** Spawned in the background by `bin/pair` on the create path, right before the zellij launch. Two discovery paths:
   - **PID-bound (preferred).** Reads `$PAIR_DATA_DIR/agent-pid-<tag>` (written by pair-wrap right after `pty.Start`) and inspects open files in that PID's process tree via `lsof -p <pid> -Fn`. Race-free across concurrent pair sessions because lsof output is scoped to specific PIDs. Falls back internally to a birth-time-filtered directory walk if the agent doesn't keep its session file open: candidates are files with `stat -f %B >= agent_start_epoch`, and only a *single* candidate is accepted (multiple = concurrent race, refuse rather than guess).
   - **Legacy snapshot-diff (fallback).** Used when the pidfile doesn't appear within 2s — i.e., when the installed pair-wrap binary predates #000020 and doesn't publish the pidfile. Behaves identically to pre-#000020: snapshots the watch dir at start, picks the first new file. Cross-tag races re-emerge in this path, so the proper resolution is to rebuild pair-wrap.

   Times out after 60s in either path.

Known gap: `/clear` rotates claude's session id mid-session, allocating a new jsonl that neither layer above sees. The launch-time `--session-id` is captured at create time, the watcher's 60s window is long gone by then, and there is no Alt+x trigger anymore. After a `/clear` + Alt+x, `pair resume <tag>` will replay the pre-clear conversation. (Pair previously sent a `bye\n` to the agent on Alt+x specifically to refresh the saved id past a `/clear`; that layer was retired because it polluted the conversation log and the rotation case is rare in practice. `/compact` doesn't rotate.)

Per-agent surface:

| Agent | Path | Id source | Capture mechanism |
|---|---|---|---|
| claude | `~/.claude/projects/<encoded-cwd>/<id>.jsonl` | filename | `--session-id` pre-injected by `bin/pair` (deterministic) |
| codex | `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<id>.jsonl` | trailing UUID in filename (regex) | `lsof -p <pid>` against agent PID + `ps`-discovered descendants, birth-time fallback |
| gemini | `~/.gemini/tmp/<project>/chats/session-<ts>-<short>.json` | `.sessionId` in JSON body (filename only carries an 8-char prefix) | `lsof -p <pid>` against agent PID, birth-time fallback |

Gemini in particular can write the file before the JSON body is flushed; `extract_id` returns empty in that case and the outer loop retries on the next tick.

**Stored shape.** `$PAIR_DATA_DIR/config-<tag>-<agent>.json`:

```json
{ "agent": "claude", "args": ["--dangerously-skip-permissions"], "session_id": "8d745d08-..." }
```

Single write path: jq + mktemp + rename, only after the id is in hand. So a concurrent reader either sees a complete prior config or a complete new one — never a partial. Keyed by `(tag, agent)` because the same tag can hold separate configs for different agents.

**Create-flow prompt (`bin/pair`).** When the create path commits a tag, pair reads `config-<tag>-<agent>.json`. If present, it runs the per-agent stale-id check (claude: `[ -f .../<id>.jsonl ]`; codex: `find ~/.codex/sessions -name "*<id>*"`; gemini: `grep -rl '"sessionId":"<id>"' ~/.gemini/tmp`) and fzf-prompts the user with up to three options:

```
1) use params + session   args=[...]   resume=<id>
2) use params             args=[...]   fresh session
3) use none               args=[<current>]   fresh session
```

fzf renders each option multi-line via `--read0` so long args / full session ids stay visible without truncation. ESC aborts the create flow. Option 3 deletes the saved config before proceeding so the watcher writes a fresh one cleanly.

**Resume composition.** "use params + session" is per-agent because the resume surface differs:

- claude / gemini — flag style. Strip any pre-existing `--resume <X>` from saved args, then append `--resume <session_id>`.
- codex — subcommand. `codex resume <id>` is the syntax, so prepend `resume <id>` ahead of any saved flags. The strip phase also drops a leading `resume <X>` at args[0..1] from saved args (the codex case where the user originally launched with `codex resume <foo>`).

The shape `compose = saved_args (stripped of any prior resume tokens) + agent's resume invocation` keeps the composed line idempotent under repeated restarts.

**Post-Alt+x hint.** `cleanup_quit_marker` reads `agent-<tag>` *before* clearing it (so the hint names the right binary even though that file is about to disappear), then prints:

```
pair: saved session config for tag "pair-2" (claude).
      resume with: pair resume pair-2
```

`SESSION` rather than `PAIR_TAG` is shown — that's what the user just saw in the UI tab. `pair resume <tag>` accepts both forms (it strips a leading `pair-`).

## Tag rename (issue #000022)

A tag is durable but historically frozen-at-create. `pair rename <old> <new>` lifts that: every tag-scoped file in `$PAIR_DATA_DIR` is renamed in one transactional pass, so the agent's saved session, draft buffer, scrollback artefacts, log, queue, and per-pane pidfiles all follow the new name. Renaming is offline-only — zellij has no live-rename for a session, so the inside-session UX wraps quit → rename → re-exec around this primitive: Ctrl+Alt+n's confirm offers `&Yes / &No / &Rename`, and the (R) path prompts for a new tag, pre-validates via `pair rename --restart-check`, then triggers the restart with `--rename-to <new>`. Orthogonal to Shift+Alt+N's `--new-session` — rename + fresh agent is one gesture.

**File-family enumeration is the canonical place to look up "what is scoped to a tag."** The launcher walks two shapes:

1. **Tag-only families** (filename is `<prefix>-<tag>[<ext>]`, no further structure): `agent`, `agent-pid`, `agent-output`, `agent-picks`, `outer-tty`, `pair-wrap-pid`, `cmux-title-pid`, `layout-mode`, `queue` (dir), `quote`, `image-capture` + `.done`, `draft-<tag>.md`, `log-<tag>.md`, `nvim-pid-<tag>-{draft,scrollback}`.
2. **Per-(tag, agent) families** anchored on `config-<tag>-<agent>.json` — also `scrollback-<tag>-<agent>.{ansi,raw,viewport,events.jsonl}` and the per-agent draft `draft-<tag>-<agent>.md`. The set of agent suffixes is hardcoded (`claude codex gemini`) — adding a new agent to pair requires updating that list in lockstep.

**Substring safety is enforced by construction**, never by filtering. The enumerator computes exact filenames like `$DD/config-$old-claude.json`; it never globs `$DD/config-$old-*.json`. This is why `pair rename brain newname` cannot accidentally pick up `brain-2`'s files — the `brain-2`'s filenames are never constructed.

**Atomicity.** The full `(src, dst)` plan is written to `$PAIR_DATA_DIR/.rename-<old>-to-<new>.journal` before any `mv` runs. On mid-flight failure, the renamer reads the first N journal lines, swaps columns, and `mv`s the completed renames back to their original paths. The journal is cleared on success and retained on rollback failure as a forensic breadcrumb (M3 will add crash-recovery: a stale journal on startup gets finished or rolled back automatically).

**Refusals.** The CLI refuses upfront when: (a) `pair-<old>` or `pair-<new>` is in `zellij list-sessions` (live, detached, or resurrectable), (b) any file matching the `<new>` family exists, (c) `<old>` has no files. Tested via `tests/pair-rename.sh`. `--restart-check` skips (a) for `pair-<old>` only (the inside-flow case: `pair-<old>` is the current session, about to be killed) and exits without touching disk.

**Inside-flow choreography.** `nvim/init.lua`'s `pair_confirm_restart_impl` shells out `pair rename --restart-check` after the user enters a new tag, re-prompting on each rejection. On accept it execs `pair-restart.sh --rename-to <new>`. `pair-restart.sh` writes `rename_to=<new>` into the restart marker (`~/.cache/pair/restart-<SESSION>`) alongside the existing `tag`, `agent`, `new_session` fields. `handle_restart_marker` in `bin/pair` runs after `cleanup_quit_marker` (so the zj delete-session has cleared the live-old gate) and if `rename_to` is set, invokes `"$0" rename <old> <new>` — full check. On success, the working tag for the re-exec is swapped to `<new>` (so `config-<new>-<agent>.json`, the just-renamed file, is what gets resumed). On failure, a 2-second visible stderr warning is printed and the restart continues with the original tag — the user is never stranded.

## Data layout

Drafts and prompt history live under `${XDG_DATA_HOME:-~/.local/share}/pair/` (per XDG Base Directory spec), keyed by tag (the agent name, or a custom name from the create-flow prompt):

- `draft-<tag>.md` — the active draft file (the `*` slot). Cleared by `send_and_clear` only when sending from `*`, persists across launches and navigation.
- `log-<tag>.md` — append-only log of every send, with timestamp. Doubles as the source for the `-N` history slots (parsed at navigation time). Searchable via `rg`.
- `queue-<tag>/NNNNNN.md` — one file per queued prompt (the `+N` slots). Filenames sort to display order (lowest = `+1`). Created lazily by `Alt+q` or auto-front-push from a dirty-`-N` "Queue" choice. Removed when the corresponding queue item is sent.
- `quote-<tag>` — transient hand-off file written by `bin/clipboard-to-pane.sh` and read by nvim's `PairPasteQuote()`. Overwritten on every selection.
- `scrollback-<tag>-<agent>.raw` / `.events.jsonl` / `.ansi` — pair-wrap's raw PTY capture, the resize sidecar, and the rendered viewer file (#000017). The .raw + .events are written live during the session (truncated on each launch); the .ansi is regenerated on every `Alt+/` press. Per (tag, agent) so multiple agents on the same tag don't clobber each other.

The launcher exports `$PAIR_DATA_DIR` so `nvim/init.lua` can compute the same path without re-deriving the XDG fallback chain.

Per-tag files mean `pair claude`, `pair codex`, and a custom-named `pair-bugfix` (entered at the prompt) all have independent draft state.

Internal: `~/.cache/pair/quit-<session>` — marker file used to communicate "user asked for full quit" between `pair-quit.sh` (or `pair-restart.sh`) and the launcher. Touched on Alt+x, Alt+n, and Shift+Alt+N; removed by the launcher after delete-session.

Internal: `~/.cache/pair/restart-<session>` — marker written alongside `quit-` by `pair-restart.sh` (Alt+n / Shift+Alt+N). Holds `tag`, `agent`, and `new_session` (0 = keep config and resume, 1 = drop config and start fresh) as `key=value` lines so the launcher can reconstruct the relaunch params after `cleanup_quit_marker` has wiped `agent-<tag>`. Removed by `handle_restart_marker` immediately before `exec`-ing pair on itself.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/outer-tty-<tag>` — single-line file containing the path to pair's controlling TTY at attach time. Read by `pair-notify` to emit OSC escapes that reach the outer terminal/wrapper. Rewritten on every attach (create or reattach); removed on full quit.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-<tag>` — single-line file recording which agent binary was launched in the session (`claude`, `codex`, ...). Written once at session create; read by `pair list` to display the agent column, and by `bin/pair`'s tag-restart agent-inference. Removed on full quit. The agent isn't otherwise recoverable post-create — env vars are frozen in pane shells, and custom session names (e.g. `pair-bugfix`) don't carry the agent in the name.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/config-<tag>-<agent>.json` — saved restart configuration for `(tag, agent)` (issue #000016, #000020). `{ agent, args, session_id }`. For claude, written synchronously by `bin/pair` before zellij launch (`--session-id` is deterministic). For codex/gemini, written by `bin/pair-session-watch.sh` once the agent's session file is discovered via lsof. Read by `bin/pair`'s create-flow prompt and by the post-Alt+x hint. Survives Alt+x (unlike `agent-<tag>`, which is cleared) — that's the whole point: it's the bridge between two pair launches against the same tag.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/agent-pid-<tag>` — child agent PID written by `cmd/pair-wrap` immediately after `pty.Start`, removed on shutdown. Consumed by `bin/pair-session-watch.sh` to scope `lsof` discovery to a specific process tree (issue #000020). Mtime is also used as the agent-start epoch in the watcher's birth-time fallback.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/nvim-pid-<tag>-{draft,scrollback}` — single-line file containing the pid of an `nvim --embed` server child. Written at VimEnter by `nvim/init.lua` (for the draft pane) and `nvim/scrollback.lua` (for the Alt+/ floating viewer) when `$PAIR_NVIM_PID_FILE` is set; the launch sites (`zellij/layouts/main.kdl` for draft, `bin/pair-scrollback-open` for scrollback) export the env var pointing at a tag-scoped path. Read and removed by `cleanup_quit_marker` on Alt+x to SIGKILL the embed deterministically — without this, the embed sometimes survives zellij's pane teardown and accumulates as a PPID=1 orphan, dragging the host into memory pressure across many quits.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/pair-wrap-pid-<tag>` — single-line file containing pair-wrap's pid, written at startup by `bin/pair-wrap` if `PAIR_TAG` is set. Read by nvim's Alt+i (`attach_image`) so it can `kill -USR1 <pid>` to arm an image-capture window. Removed by pair-wrap on exit (the `finally` block in `main()`) and by `cleanup_quit_marker` as belt-and-suspenders on Alt+x.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/image-capture-<tag>` + `image-capture-<tag>.done` — paired files driving the Alt+i image-marker pickup. On SIGUSR1, pair-wrap buffers bytes from the agent's PTY for `PAIR_WRAP_CAPTURE_S` seconds (default 0.2), then writes the buffer to the first file and touches the `.done` sentinel. nvim polls the sentinel (20ms cadence, 600ms cap), reads the buffer, strips ANSI, regex-matches the agent's image marker (claude `[Image #N]`, gemini `[Image N-M]`), and inserts it at cursor. Both files are removed by nvim after the pickup and by `cleanup_quit_marker` on Alt+x.

Internal: `${XDG_DATA_HOME:-~/.local/share}/pair/slug-proposed-<tag>` and `slug-<tag>` — the orientation-slug channel (issue #000027). `pair-slug` (spawned by pair-wrap at turn-end) writes the proposed `=== <branch> | <focus> ===` to `slug-proposed-<tag>` (atomic temp+rename); nvim applies it to draft line 1 and writes the effective line back to `slug-<tag>`, which is the `prev` the proposer reads next turn. For Codex, if `config-<tag>-codex.json` is missing, `pair-slug` can recover the live rollout by reading `agent-pid-<tag>`, walking descendants via `ps`, and checking their `lsof` paths for `~/.codex/sessions/.../rollout-*.jsonl`. Codex model auth is API-key first, then Codex CLI subscription auth via `codex exec`. Single writer each, so the channel is race-free.

**Migration from v1:** the launcher detects old `~/scratch/pair-{draft,log}-*.md` files on startup and moves them to the new XDG location, stripping the redundant `pair-` prefix from filenames.

## Path resolution

`bin/pair` prepends `$PAIR_HOME/bin` to `$PATH` before exec'ing zellij. zellij and all its child processes (panes, copy_command, Run actions) inherit the PATH and can resolve `clipboard-to-pane.sh`, `copy-on-select.sh`, `pair-quit.sh` by bare name. This lets the zellij KDL configs reference scripts without `sh -c` env-var quoting hacks.

## Adjacent: `pair-scribe`

`cmd/pair-scribe` is a `script(1)` replacement that lives in the pair repo for build-system convenience but is not part of pair's runtime — it's user shell tooling, typically wired at the top of `~/.zshrc` to swap for `script -q -F`. The user's preexec/precmd hooks send `SIGUSR1`/`SIGUSR2` to pause/resume the on-disk typescript around commands whose output (e.g. TUI redraws) shouldn't be captured, enabling a clean "capture last command output" flow that pair can read back from `$_ZSH_SCRIPT_LOG`. Lives at `~/.local/bin/pair-scribe` after `make install`. Full design notes and the zshrc snippet: `cmd/pair-scribe/README.md`.

## Design intent

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit — agent owns *output*, nvim owns *input* — and lets each side specialize.
- **Selection is the gesture.** Click-and-drag in the agent pane, mouse up — the quote is in nvim, ready for your reaction. No keystroke between.
- **Self-contained.** Uses `--config-dir` and `nvim -u` to fully isolate from the user's normal configs. No invasive install.
- **Agent-agnostic.** Same plumbing works for any TUI agent that accepts typed input. Switching is one keystroke.
- **Prompt history is just a markdown file.** Aligns with the "data into central location, shell-ed agent runs free" pattern: every send appends to a grep-able log.

## Future work

Tracked in workshop issues. v2 candidates include a real nvim plugin (for users who want LSP/snippets/telescope inside the input pane).
