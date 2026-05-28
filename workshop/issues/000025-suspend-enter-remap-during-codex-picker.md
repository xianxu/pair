---
id: 000025
status: working
deps: []
created: 2026-05-28
updated: 2026-05-28
related: [cmd/pair-wrap/main.go]
---

# Suspend Enter remap while a codex picker is open

## Problem

`pair-wrap` remaps the user's plain Enter (`\r`) to `\n` for the `codex`
agent (`cmd/pair-wrap/main.go:130-138`). That's correct in codex's
chat textarea — codex treats `\n` as "insert newline" / Shift+Enter
and `\r` as "send". The remap keeps Enter consistent across panes
(pair's nvim draft uses Enter = newline, Alt+Enter = send).

But codex also has **blocking pickers** (e.g. the resume-cwd prompt
that appears when the resumed session's cwd differs from the current
cwd; the "Choose working directory to resume this session" overlay).
In picker context, codex reads `\n` as "down" and `\r` as "select".
With the remap engaged, pressing Enter navigates down instead of
confirming the highlighted option. The user has to know to press
Alt+Enter to select — a confusing UX regression vs. running codex
outside pair.

Mirror of the claude story in #000023: same root cause (per-textarea
remap engaged during a non-textarea overlay), different agent.

## Spec

Suspend the codex Enter remap whenever codex is in a picker, restore
it when the picker closes.

**Open signal — TBD.** Claude emits an OSC 777 body that pair-wrap
hooks (`pickerOpenOSCBody` in main.go:360). Codex doesn't emit such a
signal as far as observed. Options to investigate:

1. **Watch for a textual marker in codex's output stream.** Codex
   renders `Press enter to continue` at the bottom of the resume-cwd
   picker; other pickers likely share a similar bottom-line cue. Set
   `pickerActive` when seen, clear on the first stdin event. Fragile:
   tied to UI text that may change between codex versions.
2. **Watch for a specific control sequence.** If codex toggles a DEC
   private mode (e.g. enables/disables KKP differently) or emits a
   distinctive cursor positioning while a picker is open, that's a
   more stable signal.
3. **Detect via `\x1b[?…h/l` sequences.** If codex's picker overlay
   sets/unsets a known terminal mode (e.g. cursor visibility,
   bracketed paste, or a custom DEC private mode) at open/close,
   that's a clean, byte-level hook.
4. **Probe codex source/CLI for a "tell me about overlays" option.**
   Some agents have a hidden notify channel; worth checking codex's
   binary strings or release notes.

Prefer (2) or (3) over (1) if a stable byte signal exists. Need to
probe with `PAIR_WRAP_LOG=…` to see what codex actually emits when
the picker opens/closes.

**Close signal — same as claude.** Clear `pickerActive` after the
first stdin Enter while it's set: that Enter goes through as `\r` so
the overlay confirms, and the next Enter (back in the textarea) gets
the normal `\n` remap.

**Acceptance.**
- Reproducer: a) resume codex from a session whose cwd differs from
  cwd (e.g. via Alt+n after the cwd has moved), b) wait for the
  "Choose working directory" picker, c) press Enter — option 1
  confirms (today: navigates to option 2).
- Claude's overlay handling continues to work unchanged.
- No regression when codex isn't in a picker: textarea Enter still
  inserts newline.

## Plan

1. [x] Probe codex's wire output around picker open/close. Capture
   `PAIR_WRAP_LOG=/tmp/codex.log PAIR_WRAP_REMAP_RETURN=0 pair codex`
   and trigger the resume-cwd picker (or any other picker codex
   shows). Diff the bytes between non-picker and picker states.
2. [x] Pick the most stable signal from the probe. If nothing
   byte-level works, fall back to a text marker watched in the
   output stream.
3. [x] Generalize the existing `pickerActive` plumbing in
   `cmd/pair-wrap/main.go` so claude and codex share the suspend-
   on-overlay machinery. Likely shape:
   - Per-agent `overlayDetector func([]byte) bool` registered next
     to `sendKeymapByAgent`.
   - `checkOSCForOverlayOpen` becomes a generic
     `checkOverlaySignal(data []byte)` called from `handleChunk`.
4. [x] Test: extend `osc_test.go` (or a new `overlay_test.go`) with
   table-driven cases for both agents' open detection.
5. [ ] Manual verify: trigger the codex resume-cwd picker, confirm
   Enter selects option 1 instead of navigating.

## Log

### 2026-05-28 09:57 PDT

- Marked issue working and started implementation pass.

### 2026-05-28 10:06 PDT

- Probed the locally installed Codex CLI (`codex-cli 0.134.0`) bundle with `strings`; the resume-cwd picker exposes `Use session directory (` and `Use current directory (` labels, but no dedicated overlay OSC was visible.
- Implemented per-agent overlay detectors in `cmd/pair-wrap/main.go`: Claude still uses the OSC 777 permission body; Codex strips terminal controls from newly arrived output plus a short text carryover and detects the resume-cwd labels plus `Press enter to continue`.
- Added tests in `cmd/pair-wrap/overlay_test.go` and updated the existing picker overlay tests for the generic `checkOverlayOpen` path.
- Self-review caught a stale rolling-tail edge case where old Codex picker text could re-arm `pickerActive` after the confirming Enter; fixed by clearing the Codex text carryover on the consumed Enter and added a regression test.
- Verification: `GOCACHE=/private/tmp/pair-go-cache go test ./cmd/pair-wrap -count=1`, `GOCACHE=/private/tmp/pair-go-cache go test ./... -count=1`, and `GOCACHE=/private/tmp/pair-go-cache go test -count=1 -race ./cmd/pair-wrap/` all pass.
- Rebuilt `bin/pair-wrap`; the final build was run with permission so Go could update its module stat cache cleanly.
- Updated `atlas/architecture.md` with the per-agent overlay detector behavior and Codex text-marker fallback.

## Workarounds today

- Press **Alt+Enter** to confirm in any codex picker.
- Or set `PAIR_WRAP_REMAP_RETURN=0` in the env to disable the remap
  entirely (loses the textarea Enter = newline convenience).
