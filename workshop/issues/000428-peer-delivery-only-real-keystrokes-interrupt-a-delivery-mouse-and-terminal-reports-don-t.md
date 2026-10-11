---
id: 000428
status: working
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '303722dba6405c2d631130af81dfed0de8971ef4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T13:16:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "c3a43590", done: "6429af50"}
---

# peer delivery: only real keystrokes interrupt a delivery; mouse and terminal reports don't

## Problem

A peer message to ops:0 came back `cancelled | operator input interrupted
delivery`. Its text sat unsubmitted in the composer until the operator
pressed Return (10-10; ops learnings, kink 47). The operator was probably
typing in the slot's right-pane nvim, not in the agent composer.

Analysis (pair:3, after pair#427):
- **Typing in the nvim pane cannot reach the agent.** Each zellij pane has its
  own PTY, and the wrapper only sees bytes sent to the agent pane.
- **What does reach it:** `peerDelivery.admitInput` counts any stdin byte as
  operator input, except focus in/out and terminal replies the wrapper asked
  for (`wrapcmd/orientation_replies.go` `operatorDataWithFocus`). Two
  candidates are left. One is mouse events: Claude turns on mouse tracking,
  so moving or scrolling over the agent pane on the way to nvim sends SGR
  mouse bytes. The other is an unsolicited terminal report, such as a
  color-scheme notice. This is unconfirmed without that slot's wrap-events
  log.
- **After pair#427** such an interrupt ends `indeterminate` (`uncertain:
  operator input interrupted delivery after the paste/submit`), not
  `cancelled`. The report is honest, but the text still sits unsubmitted.

## Spec

Pair classifies its **own input stream**, never the agent's screen. That
keeps pair#427's "pair renders, never classifies" principle.
1. Only real keystrokes interrupt a delivery or count as operator activity
   for delivery gating: printable text, editing and control keys, and
   bracketed paste from the operator. Mouse reports (X10/SGR/urxvt), focus
   events and terminal reports (DSR/DECRPM/OSC replies, color-scheme
   notices, and the like) do not, whether solicited or not.
2. Non-keystroke bytes are still forwarded to the agent unchanged. Only the
   delivery's interpretation changes.
3. First confirm which bytes caused kink 47, from the slot's wrap-events or
   PEER debug log if it was kept, or by reproducing it. Then encode that
   case as a test.

## Done when

- A delivery in progress survives mouse motion, clicks and scrolls over the
  agent pane, and unsolicited terminal reports. A table test covers each
  byte class through `admitInput` and `dispatchPeer`.
- A real keystroke during a delivery still interrupts it, as before.
- The kink 47 byte sequence (once identified) is a regression test.
- `atlas/couch.md` names what counts as operator input for delivery.
- **Other panes never block (operator, 10-10):** typing in the slot's draft nvim (or any non-agent pane) never delays or interrupts a delivery to the agent pane. Test: drive draft-pane keystrokes during a delivery; it submits.
- **The composer must be visible:** while the operator has the agent pane scrolled up (the input box off screen), a delivery waits, and goes as soon as the composer is visible again. A delivery must never yank the viewport away from scrollback the operator is reading. Test: scrolled-up state holds the delivery; scrolling back releases it.
- **Mouse that leaves the composer visible doesn't block:** motion, clicks and wheel events that don't scroll the composer out of view don't delay a delivery.
- **A short mutual lock with the draft pane:** between paste and submit of a robot delivery, a draft-pane send waits (about a second) instead of interleaving, and a robot delivery waits for an in-flight draft-pane send. Test: concurrent draft send and peer delivery both submit, in order, unmixed.

## Plan

Durable plan: `workshop/plans/000428-peer-delivery-only-real-keystrokes-interrupt-a-delivery-mouse-and-terminal-reports-don-t-plan.md`.

- [x] keystroke classifier (peer mode) + table tests + kink 47 regression
- [x] lone-ESC flush resolves the partial frame + test
- [x] viewport probe (zellij dump-screen aligned to the model; cursor row visible) gates the paste + tests
- [x] draft-paste hold during Delivering (≤1 s) with ordered, gap-preserving release + tests
- [x] atlas/couch.md: operator input for delivery, viewport and draft-hold rules
- [x] full make test (see Log); the live smoke (scroll up → held, scroll back → delivered; draft send mid-delivery) is handed to the ops TL with the landing report, since it needs relaunched wrappers

## Log

### 2026-10-10

- **Kink 47 measured (pair:2).** In ops:0's wrap-events (stdin is hashed), during delivery `4d5ff954` (queued 13:05:35.69): `ESC[I` at 13:05:40.853, **`ESC[13;3u` (kitty Alt+Enter) at 13:05:46.656**, and `ESC[O` at 13:05:47.364. Claude logged the message 26 ms after the Alt+Enter. So the "interrupt" was the operator's own submit after the pre-#427 stall. That pane had no mouse reports all day: its Claude never enables mouse tracking (output modes `?2004 ?1004 ?2031 ?2026 ?25` only), so zellij scrolls natively. `?2031h` does invite unsolicited color-scheme reports (`ESC[?997;Nn`), which the classifier now covers.
- Live alignment check: zellij's `dump-screen` of this slot's Claude pane, matched against a vt model fed its scrollback raw, aligns at shift 0. Slicing `dump-screen --full` at k = 1…300 rows gives consistent shifts, with a constant offset from output that landed between the two captures. Whitespace normalization absorbs the NBSP and padding differences.
- Full `go test -race` on wrapcmd: the only failures are sandbox `/tmp` denials and a race inside `codexCouchRendered`'s test helper, unrelated to this change.
- zellij 0.45 `dump-screen -p` returns the *scrolled* viewport, while `list-panes` cursor coordinates don't change with scroll. Output while scrolled does not snap zellij's viewport (probed in a background session).

## Revisions

### 2026-10-10 — operator scope fold (via the ops TL)
- **Why:** the operator types and watches in ops:0 more than any other slot, so over-eager "operator input" gating delays the TL's incoming reports most. A live test showed a delivery submitting while the agent pane was scrolled up, which snapped the viewport to the bottom and would interrupt reading.
- **Delta:** four Done-when items added: other panes never block; the composer must be visible (wait while scrolled up); mouse that keeps the composer visible doesn't block; a short mutual lock with the draft pane's send.

