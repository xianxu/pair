# Changelog

All notable user-facing changes to `pair` land here. Each release is also
tagged in git (`vN.M`) and tracked in the homebrew formula at
[xianxu/homebrew-pair](https://github.com/xianxu/homebrew-pair).

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
