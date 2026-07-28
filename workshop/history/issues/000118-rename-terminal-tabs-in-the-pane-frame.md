---
id: 000118
status: done
deps: [117]
github_issue:
created: 2026-07-24
updated: 2026-07-28
estimate_hours: 2.48
started: 2026-07-24T17:04:26-07:00
actual_hours: 2.61
---

# Rename terminal tabs in the pane frame

## Problem

Alt+r currently calls a raw prompt rendered into the active child terminal's
content area. It feels acceptable at an idle shell prompt but visually corrupts
or competes with full-screen applications such as Neovim. Zellij's native
rename-tab editor cannot be reused because the right-side tabs are Pair's own
PTY multiplexer tabs inside one Zellij floating pane.

## Spec

- Alt+r in `pair term` enters a Pair-owned rename mode for the active internal
  terminal tab. It never forwards the chord or editing bytes to the child PTY.
- Rename mode edits the existing tab name, initially placing the cursor at the
  end. The Zellij pane frame is the editor surface: on each edit its title shows
  the tab inventory with the active entry rendered as an explicit rename field
  and cursor marker. The child application's viewport is not cleared, scrolled,
  or written to.
- Printable UTF-8 inserts at the cursor. Left/Right move one rune;
  Home/End move to the boundary; Backspace/Delete remove one rune on the
  corresponding side.
- A streaming decoder separates bytes from pure editor events. Recognized
  escape-sequence prefixes and incomplete UTF-8 remain buffered across reads.
  A bare Escape cancels only after the existing 50ms terminal-prefix timeout;
  a completed Left/Right/Home/End/Delete sequence acts immediately. Malformed
  or unknown escape sequences, invalid UTF-8, Pair shortcuts, mouse reports,
  and bracketed-paste delimiters plus payload are consumed without editing or
  reaching the child.
- Enter trims surrounding whitespace and commits a non-empty name. An empty
  result retains the previous name. Escape cancels and restores the previous
  name. Either exit restores the ordinary tab-inventory frame title. If a
  single stdin read contains commit/cancel followed by more bytes, the entire
  remainder of that read is considered rename-mode input and consumed; only a
  subsequent read may resume child forwarding.
- While rename mode is active, unrelated bytes and Pair shortcuts are consumed
  rather than reaching the child. Terminal resize and child output continue to
  be processed normally; child output must not erase the rename title.
- The editor transition is a pure rune/cursor state machine (`ARCH-PURE`).
  A separately tested streaming decoder owns sequence/UTF-8 buffering and its
  50ms Escape flush boundary. The existing
  `renamePane`/Zellij runtime is the single title-output boundary
  (`ARCH-DRY`). No Zellij-native tab or pane rename mode is entered.
- Frame-title IO failures are observable and deterministic. Failure to render
  the initial rename title consumes Alt+r, reports the error, and does not enter
  rename mode. A refresh failure after an edit reports the error but retains
  both mode and edited state. Commit changes the tab name and exits even if
  restoring the ordinary title fails; cancel preserves the old name and exits
  even if restoration fails. All failures consume their input and never write
  to the child.
- Confirmation-global focus routing remains owned by #117 and is not coupled
  to rename mode.

## Done when

- Alt+r edits the active Pair terminal tab name in the pane frame while a shell,
  Neovim, or another full-screen child has focus.
- Enter commits and Escape cancels without any prompt or edit bytes appearing
  in the child application.
- Unicode insertion, cursor movement, Backspace/Delete, empty-name handling,
  and fragmented escape sequences and UTF-8 have table-driven tests. Every
  supported sequence is split at every byte boundary.
- Production-stream tests cover edit+Enter+same-read suffix, Escape timeout
  followed by subsequent-read child forwarding, shortcuts, mouse input,
  bracketed paste, and injected title failures, proving that rename-mode bytes
  never reach the child PTY.
- Existing tab create/close/switch behavior and pane-title inventory remain
  unchanged outside rename mode.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. The calibration source is marked stale, so
the estimate is provisional.

```estimate
model: estimate-logic-v3.1
familiarity: 0.90
item: issue-spec design=0.20 impl=0.08
item: tui-screen design=0.40 impl=0.40
item: smaller-go-module design=0.06 impl=0.16
item: api-integration design=0.20 impl=0.20
item: api-integration design=0.16 impl=0.08
item: atlas-docs design=0.05 impl=0.05
item: milestone-review design=0.08 impl=0.12
total: 2.48
```

## Revisions

### 2026-07-24 — Reconcile estimate vocabulary

The implementation gate rejected the non-canonical `ux-iteration` estimate
label. Classify the same live terminal/title seam and revision risk under the
closed-vocabulary `api-integration` primitive. Hours and scope are unchanged.

### 2026-07-27 — Clarify Escape suffix production behavior

Bare Escape cancellation is timeout-driven, so same-read Escape suffix bytes are
rename-mode input rather than child input. The production contract is: Escape
times out and cancels, then only a subsequent stdin read resumes child
forwarding. Keep the decoder-level flush test for suffix consumption and pin
the production boundary with `TestPumpStdinRenameEscapeTimeoutThenNextReadForwards`.

## Plan

- [x] Write and approve a durable implementation plan.
- [x] Implement the pure rename editor and terminal input mode test-first.
- [x] Verify the terminal integration suite and live Neovim-child behavior.

## Log

### 2026-07-24

- Confirmed the right-side tabs are Pair-owned PTYs, so Zellij's native
  rename-tab editor targets the wrong abstraction. Selected a Pair-owned
  frame-title editor that works independently of the foreground child.
- The implementation gate passed with INFO after replacing the unsupported
  estimate primitive. Its only plan suggestion was to name the authoritative
  shortcut source; Task 2 now points directly at
  `workbenchshortcut.FindChord`/`IsChordPrefix`.
- RED/GREEN editor tests established rune-indexed insert/move/delete,
  whitespace-trimmed commit, empty-name retention, and cancel semantics.
- RED/GREEN decoder matrices split every supported navigation sequence,
  representative UTF-8 rune, shortcut/mouse input, and bracketed-paste input
  across read boundaries. Invalid bytes and same-read exit suffixes are
  consumed without child leakage.
- Replaced the content-area prompt with one lifetime stdin reader and an
  injected 50ms Escape timer. Production-stream tests cover frame previews,
  commit/cancel, timeout versus completed controls, suffix consumption, and
  initial/refresh/finish title failures.
- Full verification passed: `go test ./... -count=1`, `make test-lua`, terminal
  shortcut and review-toggle shell suites, runtime-bundle drift check, Zellij
  config validation, and `git diff --check`.
- Added KKP Super+Backspace (`ESC[127;9u`) as delete-to-start during rename.
  RED/GREEN tests cover Unicode suffix preservation, every sequence split, the
  production stdin path, and child non-leakage. The full Go suite, terminal
  shortcut suite, runtime drift check, and whitespace check pass.
- Live smoke still produced no delete-to-start event. The installed Zellij
  exposes KKP as a restart-required session option and Pair had not pinned it.
  Added `support_kitty_keyboard_protocol true` without changing any mouse
  option; the configuration regression and Zellij parser pass.

### 2026-07-27
- 2026-07-27: closed — go test ./cmd/internal/termcmd -count=1; go test ./... -count=1; make test-lua; bash tests/term-pane-shortcuts-test.sh; bash tests/review-toggle-test.sh; zellij --config-dir zellij setup --check; git diff --check; live temporary ./bin/pair term smoke verified Alt+t new tab clears old-tab marker residue before child output; review verdict: SHIP
- 2026-07-27: closed — go test ./... -count=1; make test-lua; bash tests/term-pane-shortcuts-test.sh; bash tests/review-toggle-test.sh; make runtimebundle-drift-check; zellij --config-dir zellij setup --check; git diff --check; live layout-3 smoke in temporary ./bin/pair term pane with Neovim child verified pane-id-targeted frame-title edit/cancel/commit and no child viewport bytes; review verdict: FIX-THEN-SHIP

- Whole-issue close review returned `REWORK`: the required live Neovim-child
  smoke was still unchecked, and production-stream tests did not pin child
  output during active rename. Added
  `TestTerminalMuxChildOutputDoesNotRestoreTitleDuringRename`.
- Live smoke in the active layout-3 session exposed a real title-targeting bug:
  `rename-pane` without `--pane-id` renamed the draft pane when Pair's floating
  terminal and draft pane both appeared focused. `terminalMux` now passes its
  own `$ZELLIJ_PANE_ID` to every frame-title update, with
  `TestTerminalMuxSetPaneTitleTargetsOwnPane` pinning the action shape.
- Fresh temporary `./bin/pair term` pane smoke passed against Neovim: Alt+r
  entered frame-title rename on the terminal pane, Unicode insertion plus
  Left/Backspace/Delete updated only the frame title, Escape canceled and
  restored `[terminal 1]`, Enter committed `[terminal 1smok]`, and the Neovim
  viewport remained unchanged throughout.
- Addressed the second close review's decoder finding: unknown CSI/SS3 escape
  sequences now consume their final terminator instead of preserving it as
  printable rename text. Added regression coverage for `ESC[1;5D`,
  `ESC[999~`, and `ESCOX`, and changed rename shortcut tests to iterate the
  authoritative `workbenchshortcut.ChordSequences()` registry.
- Addressed the close review's `FIX-THEN-SHIP` traceability finding: same-read
  bare-Escape suffix forwarding is not a production behavior because Escape
  cancels only after the 50ms timeout. Added
  `TestPumpStdinRenameEscapeTimeoutThenNextReadForwards` and clarified the
  Done-when wording.
- User reported `Alt+t` left old-tab residue in the terminal where it was
  pressed. Live repro in a temporary `./bin/pair term` pane showed `newTab`
  switched the active tab and let new child output arrive without first clearing
  the pane viewport, so the old tab contents remained behind the new tab's shell
  startup output. Added `TestTerminalMuxNewTabClearsPreviousTabViewport` and
  redraw the new tab's empty buffer immediately after `renamePane`.
- Post-fix re-close review found rename commit/cancel still addressed the
  currently active tab at finish time. Captured the tab ID at rename entry and
  pass it through refresh/finish so a tab exit during rename cannot rename the
  replacement active tab. Added
  `TestTerminalMuxRenameCommitDoesNotRenameReplacementActiveTab`.
- Re-close review found tab lifecycle redraw still bypassed active rename mode:
  a PTY exit could restore the ordinary pane title and redraw the active
  viewport while stdin kept consuming rename input. `terminalMux` now tracks the
  active rename preview, preserves that frame title across tab removal, and
  suppresses lifecycle viewport redraw until finish/cancel clears rename mode.
  Added `TestTerminalMuxBackgroundExitPreservesRenameTitleAndViewport`.
- Re-close review found malformed SGR-mouse-like escape sequences with a
  non-mouse terminator could stay buffered indefinitely. Generalized unknown
  CSI/SS3 buffering so incomplete sequences are held across reads, but sequences
  with any final terminator are consumed as one malformed control. Extended
  `TestDecodeRenameInputConsumesUnknownEscapeTerminators` across every split for
  `ESC[<0;12;4X`.
- Defined target-tab removal during rename: if the renamed tab exits, the mux
  keeps an explicit detached `[rename: ...]` field visible until Enter/Escape
  clears rename mode, and commit still does not rename the replacement active
  tab.
- Re-close review found malformed sizing still used a narrower terminator set
  than incomplete detection, so `ESC[@z` and `ESCO@z` consumed the following
  printable `z`. Added split-boundary coverage for both and centralized the
  terminal final-byte predicate.

## Revisions

### 2026-07-24 — Reconcile first spec review

Define the 50ms Escape-prefix disambiguation boundary, fragmented UTF-8 policy,
same-read exit consumption, explicit handling for shortcuts/mouse/bracketed
paste, and deterministic title-IO failure semantics. Expand Done criteria to
prove every supported sequence split and every no-leak production boundary.

### 2026-07-24 — Add KKP Cmd+Delete editing

During rename mode, Kitty keyboard protocol Super+Backspace
(`ESC [ 127 ; 9 u`) deletes all runes before the cursor and preserves the
suffix. Keep this KKP-only: do not alias Ctrl+U or change Pair's mouse/terminal
protocol configuration. Split-boundary decoder and production-stream tests
must prove the sequence never reaches the child PTY.

### 2026-07-24 — Enable KKP for the Pair session

Live smoke showed the decoder never receives Super+Backspace. Explicitly enable
Zellij's `support_kitty_keyboard_protocol` session option so a supporting host
terminal can emit the already-decoded `ESC[127;9u`. Keep mouse settings
unchanged and require a completely fresh Pair/Zellij session for verification.

### 2026-07-27 — Preserve rename ownership during tab lifecycle events

Tab removal while rename mode is active must not restore the ordinary frame
title or redraw the active child viewport. `terminalMux` keeps the active rename
preview as mux state and only clears it through explicit finish/cancel, so
background PTY exits do not disturb the frame-title editor.

### 2026-07-27 — Consume malformed SGR-like controls

Malformed SGR-mouse-looking controls such as `ESC[<0;12;4X` are unknown control
sequences, not incomplete mouse reports. The decoder now buffers only CSI/SS3
sequences without a final byte; once any final byte arrives, unsupported
sequences are consumed and following printable text remains editable.

### 2026-07-27 — Share escape final-byte semantics

Unknown-control buffering and malformed-control sizing must use the same final
byte predicate. Non-letter CSI/SS3 finals such as `@` now consume only the
control sequence and preserve following printable rename input.
