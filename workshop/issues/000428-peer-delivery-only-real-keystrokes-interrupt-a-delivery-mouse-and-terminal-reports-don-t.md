---
id: 000428
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'a7d13edc39cf720736ae1e4f33f714ecaaa393e3' # card fields mirrored from issue-cards; edit via sdlc
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

## Plan

- [ ]

## Log

### 2026-10-10
