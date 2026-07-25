---
id: 000118
status: working
deps: [117]
github_issue:
created: 2026-07-24
updated: 2026-07-24
estimate_hours: 2.48
started: 2026-07-24T17:04:26-07:00
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
- Production-stream tests cover edit+Enter+suffix, edit+Escape+suffix,
  shortcuts, mouse input, bracketed paste, and injected title failures, proving
  that rename-mode bytes never reach the child PTY.
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

## Plan

- [x] Write and approve a durable implementation plan.
- [x] Implement the pure rename editor and terminal input mode test-first.
- [ ] Verify the terminal integration suite and live Neovim-child behavior.

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

## Revisions

### 2026-07-24 — Reconcile first spec review

Define the 50ms Escape-prefix disambiguation boundary, fragmented UTF-8 policy,
same-read exit consumption, explicit handling for shortcuts/mouse/bracketed
paste, and deterministic title-IO failure semantics. Expand Done criteria to
prove every supported sequence split and every no-leak production boundary.
