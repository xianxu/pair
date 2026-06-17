# Changelog

All notable user-facing changes to `pair` land here. Each release is also
tagged in git (`vN.M`) and tracked in the homebrew formula at
[xianxu/homebrew-pair](https://github.com/xianxu/homebrew-pair).

## v1.23 — 2026-06-17

### Change log (`Alt+l`) — new
- **Distilled session change log.** A new read-only viewer (`Alt+l`) shows an
  LLM-summarized log of the session's milestones and decisions — the distilled
  counterpart to the raw scrollback (`Alt+/`). It opens instantly on the existing
  log and refreshes in the background via a **detached distiller that survives
  closing the viewer**, streaming batches in with a spinner. Entries are dated by
  real change-time (captured from the TTY stream), and the log is keyed per agent
  session so a resume reuses it and a fresh session starts clean.
- **`Alt+q` in the change log.** Drop a 🤖 question/comment on a line or selection,
  exactly like the scrollback viewer; on quit it ships to the draft tagged
  `[change log]` so the agent sees what you're asking about. (The marker machinery
  is now shared between both viewers.)

### Session continuation
- **`pair continue [slug]`.** Distill a *rendered* session into a portable,
  durable markdown doc and resume from it across time, machines, or agent stacks
  — a human-readable cousin of `pair resume`. Bare lists continuations; a slug
  seeds a fresh session to pick the work back up; an agent arg ports it to a
  different stack.
- **`Alt+Shift+C` — in-session compaction.** Park the current scrollback as a
  recovery net, then reincarnate the session under the same tag with a clean
  conversation seeded from a continuation.
- On `Alt+x`, pair now offers to **park** a session's scrollback so a later
  session can distill it; the prompt auto-defaults to "preserve nothing" after 5s
  so an unattended quit never blocks.

### Draft & agent pane
- **`Alt+Backspace` always deletes the current `+N` queued prompt** (normal and
  insert mode), instead of working in only one mode.
- Draft polish: smooth scrolling in the draft pane; the first `z=` spell popup no
  longer flashes shut; as-you-type completion menus dropped the `⌥N` numbering
  (unified with the spell menu); autopair next-char gate + a normal-mode position
  marker.

### Misc
- **`:PairTTYRawPath`** (`_G.PairTTYRawPath()`) prints the live session's raw
  scrollback path and copies it to the `+` register — handy for grabbing the byte
  stream mid-session.

## v1.22 — 2026-06-08

### Draft input
- **As-you-type spell-fix typeahead** — while typing, a misspelled word
  (alphabetic, ≥4 chars) now pops a menu of likely corrections from nvim's
  `spellsuggest`, pickable like any completion (CR / Tab / arrows). It fires
  only as the last-resort completer, so it never crowds out real path/word
  completions.
- **Ctrl+C in the draft forwards to the agent as ESC** (normal + insert
  mode) — the reflexive interrupt chord now stops the agent's stream
  without disrupting your draft (insert mode stays in insert).

### Agent pane
- **Alt+Backspace → Ctrl+U (kill-to-line-start)** in the agent pane,
  matching the agent's Cmd+Delete and the draft pane's Alt+Delete. Recognizes
  both the legacy `ESC DEL` and Kitty-keyboard `ESC [127;3u` protocol forms.

## v1.21 — 2026-06-06

### Startup picker
- Inactive ("no live session") tag rows now show an amber **`[⏎ N queued]`
  badge** when that session has prompts parked in its queue — a reminder of
  queued-up work before you resume an old session. Shown only when N > 0.

### Homebrew
- **`python@3` (and the vendored `pyte` / `wcwidth` resources + private
  venv) dropped from the formula** — the v1.20 soak window is over and the
  runtime is Go-only (`pair-scrollback-render`). `brew install pair` no
  longer pulls python.

## v1.20 — 2026-06-06

### Multi-agent: Codex & Antigravity (agy) parity
- **Antigravity (`agy`) brought to full capability parity** with claude
  (#39): plain Return → newline / Alt+Return → send remap (#32),
  orientation-slug support via its JSONL transcript (#38), permission-
  picker detection that suspends the Enter remap while the picker is open
  (#42), and a fix for stdin transcript pollution during slug generation
  (#41).
- **Codex fully supported**: Enter remap suspends while any Codex picker
  (question / quota / image) is open (#25, #31, #34, #37); Codex sync
  output is filtered out of autocomplete (#30); Codex is forced into
  `--no-alt-screen` so mouse-wheel and Alt+/ scrollback work; slug
  generation falls back to Codex CLI subscription auth and recovers the
  live rollout transcript when the per-tag config is missing (#35).
- **Deprecated gemini-cli support removed** (#40) — superseded by `agy`.

### Auto-orientation slug (#27)
- The draft's first line is now an **auto-generated `=== <branch> | <focus>
  ===` orientation slug**, refreshed each turn by a small model so line 1
  always reflects where the session is. Agent-agnostic: triggered from
  `pair-wrap` at turn-end (no claude-specific Stop hook), with per-agent
  transcript parsing. The left half comes from the git branch; the right
  focus is widened to 4–8 words. Sticky and edit-safe — your edits to
  line 1 are respected and fed back as the next turn's baseline.

### Tag management
- **`pair rename <old> <new>`** offline tag-rename primitive plus an
  in-flow **Ctrl+Alt+n → (R)ename** (#22).
- Picker now **surfaces historical tags from the current directory**
  (active in the last week) (#24), shows an inactive base-tag session,
  defaults the tag to the cwd basename, and titles the pane by binary name.

### Draft editor
- **Alt+Shift+Return** — append the buffer to the agent's composer
  followed by a newline, *without* submitting (logs + clears like
  Alt+Return).
- **Markdown blockquotes render as a vertical bar** in the draft; the
  pinned `===` winbar header is tinted the diff "added" green.
- **Alt+Backspace kills to line start** in insert mode (macOS Cmd+Delete
  convention).
- Navigation: **Alt+j** toggles focus between the agent and draft panes;
  **Alt+b** opens the scrollback viewer on the previous prompt;
  **Shift+Alt+←/→** stops at three landmarks (history / `*` / queue).
- **Spell popup** (`z=`): bare-digit `1`…`9` pick with clean normal-mode
  handoff (#49).
- **Completion popup**: LeftMouse click selects and confirms an item
  (#43, #44).

### Scrollback viewer (Alt+/)
- **Smart-case search** — `/foo` is case-insensitive, `/Foo`
  case-sensitive.
- **Overall-comment affordance** at the end of the viewer for a standalone
  summary not tied to any line (#21).
- **Alt+b / Alt+B** jump to the previous / next prompt boundary.
- **Re-entrancy guard** — pressing Alt+/ while the viewer is already open
  no longer stacks a second nvim on top.
- Scrollback cap raised to **2000 rows** (in sync with zellij); assorted
  rendering fixes (hidden `~` end-of-buffer marker, no phantom rows past
  EOF, larger wrapped-quote prompt window).

### Diagnostics & runtime
- **`pair-doctor` / `:PairDoctor`** — agent-agnostic health check with
  color-coded table output (#48), an emitter-health probe that flags stale
  built binaries (#47), and an adaptation flight recorder
  (`adapt-<tag>.jsonl`) recording every harness-adaptation trigger (#45).
- **`pair-dev`** — dev-mode entrypoint that rebuilds the Go binaries from
  source on launch and on every restart (#46).
- **Python dropped from the runtime path** (#19) — scrollback rendering is
  the static Go `pair-scrollback-render`. (The brew formula still vendors
  python + pyte as a fallback during the soak window.)

## v1.19 — 2026-05-19

### cmux workspace title
- New activity-emoji prefix, refreshed every 60s from the most recent
  agent-session-file / nvim-draft mtime. Heat-ramp buckets:
  `🔴 <1d`, `🟠 <3d`, `🟡 <10d`, `🔵 <21d`, no prefix (cold).
- Personal-display substitutions baked in: `brain` → 🧠, `book` → 📗,
  `pair` → ♋.
- Fix race on `pair <tag>` (create path) where the title poller exited
  before zellij had finished creating the session, leaving workspaces
  without their activity prefix.

### Scrollback viewer (Alt+/)
- **New floating-window prompt** for adding and editing 🤖[] markers.
  Markdown-blockquote header shows the line (bare marker) or selection
  (scoped marker) being annotated; single-line input below; Return to
  accept, Esc to cancel. Replaces the cmdline-based input that leaked
  Option+Delete byte sequences into the underlying buffer on macOS.
- **Edit-in-place**: Alt+q with the cursor on an existing 🤖[…] or
  🤖<…>[…] opens the edit prompt prefilled with the comment.
- **Clear-and-Enter deletes**: scoped markers restore their wrapped
  selection on delete; bare markers remove cleanly with adjacent-space
  collapse.
- **Baseline filter**: snapshot the 🤖[…] tokens present at buffer
  load and emit only user-added markers on quit. The viewer no longer
  re-ships markers that were already in the agent's captured
  transcript.
- **Confirm-on-exit** prompt on Esc *only* when there are pending
  markers to ship, with their count. Passive reads exit instantly.
- **Instant marker pickup** in the draft pane: a libuv `fs_event`
  watches the scrollback-pending sidecar and routes into the existing
  landing logic within milliseconds of the rename. Previously the
  pickup waited for a `FocusGained` event, which could lag 5–10 s
  after the floating pane closed.
- Drop empty `🤖[]` markers from the sidecar block — unfinished
  marker syntax no longer ships a quote-only line.
- Fix phantom-space rendering after wide-grapheme cells in the Go
  renderer (`🔴` / `🧠` / `📗` used to land as `🔴 ` with a stray
  column of whitespace).
- **Open at the agent pane's actual current viewport** — including
  any in-zellij scroll-back the user has navigated to. The renderer
  writes a `<ansi>.viewport` sidecar with the *emulator's* visible-
  buffer top; pair-scrollback-open then overlays zellij's live
  scroll position by dumping the agent pane (`zellij action
  dump-screen --pane-id terminal_<n>`) and multi-line-matching the
  dump back into the `.ansi` to recover the line number you're
  currently looking at. Match requires ≥ 50% of non-blank dump
  lines to land consecutively, so short-line false positives don't
  steal the hypothesis. The viewer reads the resolved `.viewport`
  on `BufReadPost` and pulls that line to the top of the window
  (`zt`). Falls back to the emulator viewport, then to file-end,
  in that order. Net effect: Alt+/ opens exactly where you were
  looking — whether at the bottom or pageup'd to the top of
  zellij's buffer.

### Autocomplete
- `pair-wrap`: preserve word boundaries when the agent paints a colored
  span cell-by-cell with cursor-positioning escapes between glyphs. The
  inline-code span `make nous-install` stops merging into the unusable
  completion candidate `makenous-install`. Fixed in both the Go binary
  (the primary path shipped via brew) and the Python fallback.

### Tests (developer-facing)
- Eight new Go test files (`extractFG`, `splitBytes`, `updateAgentOutput`,
  `translateStdin` pipeline + 30ms flush timer, OSC dispatch, keymap
  registry incl. gemini, `serializeRow` incl. wide-grapheme, `render`
  end-to-end, `parseEvents`, `feedSegments` clamp-beyond-EOF,
  `initialSize`). Test count went from 1 file to 8.
- `make test` / `make test-race` targets.
- `render()` waits for its drainer goroutine before `em.Close()` —
  race-detector-clean cleanup ordering (this also formalises a
  correctness ordering that existed only implicitly before).

### Misc
- Vendored refreshes from ariadne: `.tart/` toolchain (rsync-mirror of
  `~/repo`, prepended `~/repo/bin` on the VM PATH, GPG_TTY self-heal)
  and `apply-gitignore-entries.sh`.
