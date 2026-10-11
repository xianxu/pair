---
id: 000430
status: working
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '647e9d6e13bb6f03f2abd8823d9ee23944bacca3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T17:47:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# couch exits when the outer terminal stalls 5s: degrade the view, keep the sessions

## Problem

Couch keeps shutting itself down under heavy load. Each time, Ghostty stopped
reading Couch's output for 5 seconds. #409 deliberately kept that exit
("failing loudly is how this problem was found"). It added a crash record and
row-diff output. The crash records now show the exit recurring at small write
sizes, so output volume is no longer the cause:

| run started (UTC) | record |
|---|---|
| 2026-10-09 18:09 | wrote 122 of 14086 bytes (before #409 M2) |
| 2026-10-10 04:10 | wrote 45 of 605 bytes |
| 2026-10-10 23:21 (exit ~17:35 local) | wrote 373 of 688 bytes |

(`~/.local/share/pair/couch/crash/*.crash`; the running binary was main
4c147e5d, which includes #409 M2.)

At the 17:35 exit (pair:4 investigation for ops:0, 2026-10-10):

- **Load:** the 12-core host had load averages of 72/87/63.
  - Two ariadne `sdlc.test` runs at about 140% CPU each.
  - Spotlight and FSEvents.
  - About 90 pair processes, plus nvim.
- **Ghostty:** its main thread kept logging through 17:35–17:36 (unified
  log), so the app was alive. Only its pty reader fell behind.
- **Memory:** no jetsam and no OOM. Free memory was 83%.

Ghostty took more than 5s to drain a 600-byte write: the terminal was slow,
not gone.

The policy turns a transient stall in the presentation layer into the loss of
the coordinator, along with all ~10 agent PTYs it hosts (`endpoint-end` for
every `couch-pty-*`, per #409's capture). The restore write then also times
out, which leaves the operator's terminal in a broken mode state.

Mechanism: `terminal.WriteTimeout = 5s` (`cmd/internal/terminal/profile.go`).
`Presenter.write` turns an expired deadline into
`WriteFailure{Op: "parent output"}`, then `Presenter.fail` runs and Couch
exits (`cmd/internal/terminal/presenter.go`).

## Spec

Separate two cases: a slow parent and a gone parent.

- **Gone parent:** EIO, EPIPE or hangup on the parent tty means the terminal
  is gone. Exiting stays correct.
- **Slow parent:** an expired deadline means the terminal is slow. The
  presenter should stop painting and keep the in-flight write blocked, with no
  deadline. Frames coalesce into the dirty state the presenter already keeps,
  so they are skipped rather than queued. Sessions, endpoints and the control
  plane keep running.
  - When the write completes, the presenter repaints the full screen once
    (`reset`) and resumes.
  - The stall is recorded as a notice or diagnostic, not as an exit.
- **Escape hatch:** if a deadline is still wanted, use a much longer one
  (minutes), or an operator action (detach, then reattach). Never 5s of
  scheduler starvation.

This also covers the last Done-when line of #420 ("Couch stays connected
through a multi-second terminal stall"). #420's polling work is a load
reducer; this issue is the resilience fix. Ship it independently.

## Done when

The stall's cause is unsettled (see Log), so the bar is cause-agnostic: a
parent write blocked well past `WriteTimeout`, then the terminal drains.

- A test with a fake parent writer that blocks for longer than `WriteTimeout`
  and then drains shows:
  - Couch stays up.
  - The endpoints stay alive.
  - No frames are queued during the stall: one coalesced full repaint follows
    the drain.
  - The end state matches a fresh presentation of the final frame. The test
    replays both byte streams through Pair's emulator; the xterm oracle isn't
    installed in this checkout.
  - Effects emitted during the stall are still handled and delivered, so
    attention marks survive; only their bytes are dropped.
  - A one-shot write (copy) right after the drain does not end the stall
    without the full repaint.
- The same holds over a real PTY host that stops reading past `WriteTimeout`
  and then drains.
- A stall is counted (`Presenter.Stalls()`). A Couch-visible notice is out of
  scope (see Revisions).
- A test where the parent writer returns EIO or EPIPE still ends Couch with a
  recorded exit reason (the #409 path).
- Live (TL rollout): Couch survives load and background stalls that used to
  end it. The operator watches for exits.

## Plan

- [x] `terminal.Presenter`: a parent write whose own `WriteTimeout` expires
  (`WriteFailure.Expired`) enters a stall, not a failure. The stall forgets
  the parent's screen (previous frame, history state, modes) and keeps
  admitting frames logically.
- [x] While stalled, paints probe with `CAN ST syncEnd` (100 ms budget) and
  write nothing else. A retry every 250 ms repaints in full once a probe gets
  through, so recovery doesn't wait for new output.
- [x] Effects and clipboard copies are dropped while stalled, never queued and
  never an error, so no caller ends Couch over them.
- [x] Recovery re-establishes modes without a second kitty-keyboard push.
- [x] A caller's own deadline, EIO and EPIPE still fail as before.
- [x] Tests: `presenter_degrade_test.go` (fake-writer stall then drain, real PTY
  stall then drain, EPIPE/EIO).

## Log

### 2026-10-10
- 2026-10-10: closed — make -k test green except two env-leak targets (test-pair-embedded-runtime, test-changelog: PAIR_DATA_DIR scope conflict from the hosting pair session), both PASS under env -i. go test ./... (env -i, -p 2): terminal incl. presenter_degrade_test (stall past WriteTimeout then drain: stays up, no queued frames, stalled effect batch delivered, copy right after drain still gets the full repaint, end screen equals a fresh presentation, one keyboard push; real-PTY stall then drain; EPIPE/EIO fail with exit reason; stalled-then-EIO fails) PASS; BR-2 and stalled-then-gone mutations fail their tests. couchtty/termcmd/crashreport/couch PASS. couchcore PASS with -timeout 20m (1052s under load ~30). Not this change: couchsingleton TestSelectionSizeContract* and gcruntime TestCouchReferencesLocalArchiveLocatorRoundTrip fail on an origin/main export too; artifactpath/diagnosticlog do not depend on cmd/internal/terminal; couchcmd TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins is the #423 flake (main export fails 3/5 and 1/5). Live: TL rollout + operator watch.; review verdict: SHIP
- 2026-10-10: flow upgraded quick → full — 149 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Filed by pair:4 at the request of ops:0 (TL), during a load diagnosis. Not implemented.
- Claimed by pair:4 for implementation (TL, high priority). This issue owns
  #420's last Done-when ("Couch stays connected through a multi-second
  terminal stall"); ops:0 and pair:3 (#420) have been told.
- **Two hypotheses for the stall, both open.** The fix doesn't depend on
  which one is right.
  1. *Load starvation:* load was ~80 on 12 cores and Ghostty's pty reader got
     no CPU for 5s.
  2. *A backgrounded or blocked Ghostty:* the operator's challenge. Couch
     paints only the selected slot, and the stalled writes were under 700
     bytes, so load alone is a weak explanation.
     - #409's exits also happened while Couch was in the background behind a
       browser, and WebKit was at 75% this time.
     - A partial write (373/688) means the pty buffer was already full, so
       Ghostty's reader was not reading at all, not just slow. A plausible
       cause is its IO thread waiting on the terminal-state lock held by a
       stuck renderer (Metal/WindowServer; there were 41 MTLCompilerService
       processes). App Nap or lowered QoS on a non-frontmost app could
       contribute.
- **Background repro: not run.** It needs another app frontmost while a slot
  prints. The agent sandbox can't drive GUI focus (osascript and `open` are
  blocked), so it's left to the operator. The live check after rollout covers
  both causes.
- **`sample <terminal pid>` capture on a stall (ops:0, optional): skipped.**
  The parent terminal's pid isn't cheap to find reliably. Couch's ancestry is
  zsh → login → ghostty here, but under zellij, ssh or tmux the outer terminal
  isn't a local ancestor at all, and a heuristic walk would guess. If the
  stalls persist after rollout, a follow-up can take the terminal pid
  explicitly (env or flag) and run `sample` once per stall, rate-limited,
  next to the crash directory.
- Design: the presenter distinguishes its own budget from a caller's deadline
  (`WriteFailure.Expired`). The #383/#409 tests that use a 300 ms caller
  deadline keep failing the presenter, which is correct: the caller asked for
  a bound. `Presenter.Stalls()` counts stalls; wiring a visible notice is left
  out of scope.
- Considered: the stall-inducing write still holds the presenter for its 5s
  budget, and probes cost at most 100 ms per painting call. Admission can't
  overflow into `ErrBackpressure`, which Couch treats as fatal: `p.call` blocks
  each caller until done, so each calling goroutine holds at most one queued
  request, and `Present` (endpoint output) is a flag, not a request.
- Close review round 1 (FIX-THEN-SHIP) found two Important issues, both fixed:
  - BR-1: a stalled `EmitEffects` returned early and lost non-writing effects
    (attention marks). Now only the bytes are dropped; every effect is handled
    and delivered.
  - BR-2: a probe inside a copy or effect cleared the stall without the
    repaint. Now only a completed full paint clears `stalled`, and one-shot
    writes drop while stalled without probing. The test now copies right after
    the drain; a mutation check reintroducing the bug fails it.
  - Also fixed: `Expired` raced `ctx.Err()` (a poll can see the deadline
    first), so it now compares the caller's deadline with the write's budget.
- Minor dispositions:
  - `Stalls()` is a test-facing counter only; see Revisions.
  - Flag constellation: `stalled` is the one new flag. `modesKnown` and
    `previous` predate it and are reset together by `stall()`, so it isn't a
    new state machine. Moving parent-screen knowledge into `View` belongs in
    a separate refactor.

- Close review round 2 (SHIP), minor findings:
  - Fixed: a presenter that failed while stalled re-armed the retry forever;
    `armRetry` now skips a Failed view.
  - Disposed: a stall landing exactly between the alt-screen enter and its
    kitty push leaves the alt screen without the push after recovery (#279's
    legacy keys). Both are tiny consecutive control packets, so the window is
    one packet boundary. The general fix is the reviewer's rule: make
    everything known about the outer terminal one value that `stall()`
    resets whole. That is the parent-screen refactor noted above, not this
    narrow fix.

- Close review round 3 (FIX-THEN-SHIP), fixed: a terminal that stalled and
  then went away (EIO on the probe) counted as still slow, so it never failed.
  Now only the probe's own deadline means "still stalled", and any other error
  fails with its exit reason. `TestPresenterStalledThenGoneFails` covers it,
  and a mutation check (the probe swallowing errors) fails that test.

## Revisions

- 2026-10-10: the Spec's "stall recorded as a notice or diagnostic" is reduced
  to `Presenter.Stalls()`, which only tests read; no Couch notice yet. Reason:
  keep the high-priority fix narrow. The rollout's live check watches for
  exits, which is the property that matters. If stalls need to be visible, a
  follow-up can trace each stall (with duration) through Couch's trace or
  crash directory, together with the optional `sample` capture noted above.
