---
id: 000464
status: done
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '733b0bd290e3efcc5208345ddc81b254441a8b83' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T00:15:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "87d3ca31", done: "28fc0160"}
actual_hours: 0.41
---

# couch: messages to an idle slot expire 'waiting for picker to close' when no picker is open

## Problem

2026-10-12 00:0x–00:5x: three `couch --send-to pair:3` messages from ops:0 expired with `delivery deadline elapsed: waiting for picker to close (1m30s)` (last probe: 6a381c3b-90c1-473f-a6c4-d32a3684f0b4). pair:3 (Claude Code 2.1.296) was idle at its prompt with no picker visible in `couch --peek`. Its prompt showed Claude Code's dim prompt suggestion (ghost text: "The operator reviewed the format; proceed to close and merge"), and earlier a background shell with the footer "1 shell · ↓ to manage". Other slots received messages normally in the same period. The hub couldn't reach the slot at all; the operator had to type into it by hand.

## Spec

Find what Couch's delivery reads as an open picker on that screen (the dim prompt suggestion, the background-task footer, or stale state from when one was shown) and fix the detection at the root: a picker is an actual selection UI, not ghost text or a footer hint. A message that can't be delivered says why in terms the sender can act on. Add a screen fixture from this case to the tests.

## Done when

- A fixture of an idle Claude Code prompt with a dim suggestion (and one with the background-task footer) reads as deliverable; real pickers still block.
- A live check: a message to an idle slot showing a prompt suggestion is submitted.

## Plan

Root cause (measured, below): `pickerActive` is a latch that only a plain Enter
through the wrapper clears. A picker that closes any other way (Claude Code's
unattended auto-deny, Esc, a digit key) leaves it armed for good, and peer
delivery waits "for picker to close" until someone presses Enter in the pane.

- [x] Profile flag `overlayHidesComposer` (claude): its pickers replace the
      composer. While the latch is armed, each terminal feed reads the
      recognizer; once the composer has been seen hidden, its return means the
      picker closed, and the latch clears (same consume as Enter: tails reset).
- [x] Tests: the recorded pair:3 screens (permission prompt with auto-deny
      countdown → idle composer with a faint suggestion) clear the latch and
      read deliverable; a picker still on screen stays armed; a composer that
      never went hidden (OSC ahead of the paint) doesn't clear; agents without
      the flag keep the Enter-only rule.
- [ ] Live check: a peer message to an idle slot showing a suggestion submits.
      Deferred to the TL (ops:0, 2026-10-12): after the next batched pair:0
      build, Couch restart and slot relaunch, the TL messages a relaunched slot
      that had a picker auto-denied, or the closest repro. The replayed-screen
      tests are this close's verification.

## Log

### 2026-10-12
- 2026-10-12: closed — Replayed-screen tests (cmd/internal/wrapcmd/peer_picker_test.go) over screens re-rendered from pair:3's recording plus a recorded question picker: the auto-denied permission prompt closes the latch, a picker on screen keeps it, an OSC ahead of the paint keeps it, a question picker's free-text row keeps it, profiles without overlayHidesComposer keep Enter-only, and a peer message to the idle suggestion screen after the auto-deny is submitted. Each part of the fix mutation-checked. Full make -k test green (exit 0, 242 packages ok). Plan's live-check item left open on purpose: deferred to the TL (ops:0) per their instruction, after the next pair:0 build + Couch restart + slot relaunch. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- Measured on pair:3's recording (`scrollback-1-pair-7-claude.raw`): one
  `777;notify;Claude Code;Claude needs your permission` at 23:45 for a Bash
  prompt ("Dangerous rm operation…"), whose screen read "Claude Code will
  automatically deny this request in 1:59, to avoid blocking progress on an
  unattended session". It auto-denied with no Enter, so the latch stayed
  armed. The dim suggestion was a red herring: `peerPlaceholder` already reads
  faint ghost text as an empty composer.
- Fix (`wrap.go`): `observePickerScreen` runs on every terminal feed while the
  latch is armed, and once at arming. The picker's frame lands before its
  OSC, so without the arming read a picker closing in the very next frame
  went unseen. The end-to-end test caught that. `consumeOverlayLocked` is now
  the one way the overlay ends, for the Enter and the screen alike (ARCH-DRY).
- Fixtures: four screens re-rendered from the pair:3 recording, the #438 way
  (README). Round trip checked: each re-render reads the same text, cursor,
  recognizer and composer state as the live replay.
- Mutation-checked (`peer_picker_test.go`): dropping the feed hook, the arming
  read, the hidden-first rule or the profile flag each fails a test.
- Close review round 1 (FIX-THEN-SHIP): BR-1 atlas still described the
  Enter-only edge for every harness and called the two layers independent.
  Fixed: the edge is now per profile, and the one coupling is named with its
  test. BR-2 (sender-actionable reason) descoped to #465, see Revisions. The
  Minor on the coupling got a fixture: a recorded question picker whose
  free-text row (`❯ 4. Type something.`) sits at column 0 between rules. The
  recognizer declines it, and the latch holds over it. The Minor on tracking
  the live check is answered by the TL's deferral, recorded in the Plan.

## Revisions

### 2026-10-12, at close
- **The Spec's "a message that can't be delivered says why in terms the
  sender can act on" moves to #465.** #465, filed the same night, owns
  exactly that: receipts reporting `blocked by a picker: <title>` and a hub
  notice with the picker's text, built on this fix's picker state. Rewording
  the detail here would leave a second, interim wording for #465 to replace.
  What this issue delivers on that clause is the cause: the expiry no longer
  names a picker that isn't on screen.
