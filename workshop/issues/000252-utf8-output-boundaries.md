---
id: 000252
status: open
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
---

# Preserve UTF-8 across terminal output chunks

## Problem

Operator screenshots show replacement glyphs in Codex transcript separators and
the surrounding Zellij pane border, plus stray dots in the composer. The source
Codex raw capture is valid UTF-8; the saved ANSI transcript has no replacement
characters. Corruption therefore occurs downstream of that capture.

Confirmed defect: `ptychild.Screen.MidSequence` tracks incomplete escape
sequences but not incomplete UTF-8 characters. Couch's `writeChild` uses that
answer to append keyboard-disambiguation controls after each output chunk. A
chunk ending inside the three-byte `─` character is treated as complete; the
injected control sequence separates its bytes. `SafeToPaint` also incorrectly
returns true at this boundary. This affects the whole Zellij output, explaining
why both agent content and pane borders can be affected.

Whether the stray composer dots share this cause remains unconfirmed.

## Spec

Proposed fix: extend the shared output-boundary model to account for incomplete
UTF-8 as well as incomplete terminal escape sequences. Consumers must defer
injected controls and paint until a safe boundary without altering child bytes.
Keep decoder state bounded; malformed input must not indefinitely block control
writes. Audit shared callers, including Couch and terminal panes, rather than
adding a separate Couch-only UTF-8 parser (ARCH-DRY, ARCH-PURPOSE).

## Done when

- Splitting valid two-, three-, and four-byte characters at every byte boundary
  cannot let injected controls corrupt their rendering.
- Real terminal-emulator tests cover Couch's keyboard-control injection and
  interleaved paint; escape-sequence framing regressions continue to pass.
- Malformed bytes, ASCII recovery, screen reset, and successive partial reads
  have defined bounded behavior.
- Verify the operator-visible separators after installing the fix; investigate
  stray dots separately if they persist.

## Plan

- [ ] Design the shared safe-boundary contract and malformed-input recovery.
- [ ] Add regression tests and implement the boundary fix.
- [ ] Verify affected consumers, update documentation, and close through SDLC.

## Log

### 2026-09-14

Read-only reproduction used a Go test overlay (no source changes): feed the
first one or two bytes of `─` into `ptychild.Screen`, observe both
`MidSequence=false` and `SafeToPaint=true`, append the actual
`hostty.EnableKeyboardDisambiguation` bytes, then append the remaining UTF-8
bytes. The existing VT emulator renders a replacement glyph and leaked control
text rather than `─`. Both split positions reproduce the defect. Temporary
overlay: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-utf8-repro-o6l7alu9/overlay.json`.

Inspection links frequent injection to commit `9ec88703` (#251, maintaining
Couch notification key encoding). The shared boundary-model omission predates
that trigger. This is separate from ongoing #249/#250 continuation work.


### 2026-09-14 — Screenshot-local TTY evidence

Operator reported flashing around the SDLC sync tool output in current Codex
thread couch-79469b7a163cc37d. Located exact screenshot text in raw capture at
byte 343091506 (Syncing issue changes), with matching wrap events around
15:13:44.427725–15:13:45.002679 local time. The displayed cyan arrow is ASCII
`==>` emitted by SDLC (font renders it as an arrow); branch marks are valid
UTF-8 U+2502/U+2514 and ellipsis is U+2026. A 20 KB surrounding sample is valid
UTF-8 and contains no replacement codepoints. Preserved sample and event rows
at /tmp/pair-display-sync-window.raw and /tmp/pair-display-sync-events.json.

Sample includes 19 synchronized-update begin/end pairs. Actual wrap events
confirm standalone ESC[?2026l chunks were stripped (stdout_len=0, filtered=true).
The source also repeatedly moves the cursor and clears/repaints lines. Removing
frame boundaries can expose partial redraws and is a plausible flashing cause,
but this log does not prove it caused the operator's observed flashing. The
separate confirmed UTF-8 injection defect above remains applicable downstream;
raw capture is upstream of Couch and cannot show Couch-added corruption.
No filter, terminal mode, or live session changed during this inspection.
