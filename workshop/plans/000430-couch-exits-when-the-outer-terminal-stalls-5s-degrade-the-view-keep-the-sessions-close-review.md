# Boundary Review — pair#430 (whole-issue close)

| field | value |
|-------|-------|
| issue | 430 — couch exits when the outer terminal stalls 5s: degrade the view, keep the sessions |
| repo | pair |
| issue file | workshop/issues/000430-couch-exits-when-the-outer-terminal-stalls-5s-degrade-the-view-keep-the-sessions.md |
| boundary | whole-issue close |
| milestone | — |
| window | c39ed7243b1765b44d173f6063abe37f98d93998..72ef8f20ec226e12958cf339ef618c4a5ac32aa3 |
| command | sdlc close --issue 430 |
| reviewer | claude |
| timestamp | 2026-10-10T18:40:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking whether Couch's production callers pass their own deadlines (which would bypass the new stall path), and who consumes `Stalls()`.

Two real defects found so far (dropped non-writing effects; probe in the effect path skips the repaint). Running the terminal tests to confirm the suite state.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Verdict: fix two things, then ship.** The core change is sound. `WriteFailure.Expired` cleanly separates "the presenter's own 5s write budget ran out" (a slow terminal) from EIO, EPIPE or a caller's deadline (a gone terminal). Couch's production callers pass `c.lifetime`, which has no deadline, so a real stall does reach the new path. The fake-writer test covers the hardest part of the Done-when: frames are coalesced, one full repaint follows, the result is checked against a fresh presentation by emulator replay, and the kitty-keyboard stack gets no second push.

Two defects come from routing effects and copies through `writeUnlessStalled`:
- **Effects lost during a stall.** Effects that write nothing to the terminal are now dropped, so Couch loses attention marks and directory updates. This is control-plane state the Spec says keeps running.
- **No repaint after recovery.** If an effect or copy is the first thing to reach the drained terminal, the screen is left in its mid-frame state.

Both are cheap to fix. Neither path has a test.

Verification: at head 72ef8f20, the fake-writer tests pass (`TestPresenterStallPastWriteTimeoutDegradesThenRepaints`, `TestPresenterFailsWhenTheParentIsGone`). The PTY-backed tests (`TestPresenterSurvivesARealStallPastWriteTimeout` and the existing ones in `presenter_stall_test.go` and `exitreason_test.go`) failed with "operation not permitted" because the sandbox blocks PTYs. They need an unsandboxed run before close.

### Strengths
- `presenter.go:236`: `Expired` is computed as `DeadlineExceeded && caller.Err() == nil`. This keeps the #383/#409 tests that use a caller deadline failing the presenter, as they should.
- `stall()` (`presenter.go:261`) forgets everything known about the outer terminal (previous frame, history, modes) in one place, so the next paint that gets through is a full one.
- The retry timer (`presenter.go:191`) means recovery doesn't wait for new output. `repaint` is factored out of `Refresh` (`tap.go:41`) instead of copied.
- `parentModeDelta`'s new `keyboardOwned` parameter avoids a second push onto the keyboard stack. The test pins exactly one `keyboardPush`.
- The test compares end states by replaying both byte streams through Pair's emulator, not by asserting on bytes.

### Critical
None.

### Important
1. **`presenter.go:834`, `EmitEffects`: while stalled, effects that write nothing are dropped, along with the rest of the batch.**
   - A background slot's bell sets `s == ""` (`policy.Bell` is false). So does `DirectoryEffect`, and any effect whose policy is off.
   - These still go through `writeUnlessStalled`. When the probe fails, it returns `ok=false`, and `return err` (nil) exits the loop.
   - That effect and every later effect in the batch are never recorded in `p.effects` or `delivered`.
   - `console.go:1259-1266` builds attention marks from `delivered`: a bell on an inactive slot, or a `pair` notification. Under load, exactly when agents finish, those marks are silently lost.
   - Fix:
     - When `s == ""`, mark the effect delivered without touching the outer terminal.
     - When a write is dropped because of a stall, still advance the sequence and `continue` rather than `return`.
     - Decide explicitly whether a dropped notification still counts as delivered for attention. Couch reads it that way.
   - Add a test: a stalled presenter given a batch with an inactive-slot bell and a notification must still return them as delivered.
2. **`presenter.go:294`, `writeUnlessStalled`: a probe that succeeds there clears `stalled` but never schedules the full repaint.**
   - `armRetry` only arms while `p.stalled` is true, and `previous` and `modesKnown` stay cleared.
   - If a copy or effect is the first operation after the terminal drains, it writes its control and then nothing repaints until the child produces output.
   - Meanwhile the outer terminal shows the abandoned mid-frame bytes, with modes (mouse, paste, focus) not re-established. An idle slot can sit like that indefinitely, which defeats the retry's stated purpose ("recovery doesn't wait for new output").
   - Fix: in `writeUnlessStalled`, drop while stalled without probing, leaving recovery to the paint and retry paths. Alternatively, keep a `repaintDue` flag that keeps the retry armed until a paint succeeds.
   - Add a test: call `Copy` after `close(gate)` but before the retry fires, then assert that the frame text and mode setup still arrive.

### Minor
- **No notice or diagnostic for a stall (ARCH-PURPOSE).** The Spec says "The stall is recorded as a notice or diagnostic". Only `Stalls()` exists, and nothing in production calls it. The Log defers the notice, and the Done-when doesn't require it. Note it as a follow-up: without it, the live check after rollout can't tell a survived stall from no stall.
- **Spec wording drift.** The Spec says to keep the write "blocked, with no deadline"; the code abandons the write after 5s and probes. The Plan describes the code correctly. A `## Revisions` line on the Spec would remove the contradiction.
- **Stall state is a flag, not a modeled state (ARCH-ORDER).** `stalled` sits alongside `modesKnown` and `previous` outside the `ViewEvent` state machine. That's acceptable at this size, but a candidate for an explicit state if more conditions accrue.
- **Hard-to-read atlas sentence.** `atlas/couch.md:2193`: "The stall wording below is left for a write bounded by a caller's deadline" is hard to parse.

### Test coverage
- **Covered:**
  - Stall then drain on the fake writer: coalescing, a single keyboard push, end state equal to a fresh presentation, and release after recovery.
  - EIO and EPIPE still end the presenter with an exit reason.
  - A real PTY stall (pending an unsandboxed run).
- **Not covered:**
  - Effects during a stall (Important 1).
  - A probe succeeding through an effect or copy (Important 2).
  - The Done-when's "endpoints stay alive" is only implied by the presenter staying `Ready`. Nothing asserts on endpoint liveness or input reaching the child during a stall.

### Architecture
- **ARCH-DRY: pass.** `repaint` is shared by `Refresh` and the retry.
- **ARCH-PURE: pass.** The classification is a small predicate; the shell stays thin.
- **ARCH-PURPOSE: minor flag.** The diagnostic is missing; effect loss (Important 1) erodes "the control plane keeps running".
- **ARCH-MOCK: pass.** Stateful `ttyio.Fake` with blocking steps, plus a real PTY check.
- **ARCH-CONSTRAINTS: pass.**
  - The first stall still blocks the presenter for 5s (documented).
  - Each probe is capped at 100 ms per painting call.
  - Retry runs every 250 ms, only while stalled.
  - Admission is bounded per caller (Log).
- **ARCH-SECURE: N/A.** No new untrusted input or secrets; probe bytes are a fixed constant.
- **ARCH-ORDER: minor flag.** The flag constellation noted above. Tests drive ordering with a gate channel, which is good; the effect and copy orderings around recovery are untested.
- **ARCH-FUNERAL: pass.** Nothing durable is created; the retry timer lives with the run loop.

### Plan revisions
- Add a `## Revisions` entry noting that the Spec's "keep the in-flight write blocked, with no deadline" became "abandon after `WriteTimeout`, then probe with `CAN ST syncEnd` every 250 ms", as the Plan states.
- Add a `## Revisions` entry noting that the Spec's "stall is recorded as a notice" is reduced to the `Stalls()` counter, with the notice deferred.

```findings
findings:
  - id: new
    severity: Important
    family: stall-drop-loses-control-plane-state
    title: |
      EmitEffects drops non-writing effects and the rest of the batch while stalled, losing attention marks and directory updates
    detail: |
      presenter.go:834 sends s=="" effects (inactive-slot bell, DirectoryEffect, policy-off) through writeUnlessStalled; a failed probe returns ok=false and the loop returns, so p.effects/delivered miss them and couchtty console.go:1259 never marks attention. Skip the outer terminal for empty writes, continue past dropped writes, and add a stalled-batch test.
  - id: new
    severity: Important
    family: stall-recovery-must-repaint
    title: |
      A probe succeeding inside writeUnlessStalled clears stalled without scheduling the full repaint
    detail: |
      After the outer terminal drains, a Copy or effect that probes first sets stalled=false, so armRetry no longer fires; the screen keeps the abandoned mid-frame bytes and unset modes until new child output. Drop without probing while stalled, or keep a repaint-due flag until a paint succeeds; test Copy-after-drain.
  - id: new
    severity: Minor
    family: spec-diagnostic-deferred
    title: |
      Spec's stall notice/diagnostic is reduced to an unconsumed Stalls() counter
    detail: |
      Nothing in production calls Stalls(), so the rollout live check cannot distinguish survived stalls; add a Revisions entry and a follow-up.
  - id: new
    severity: Minor
    family: state-flag-constellation
    title: |
      stalled/modesKnown/previous form a flag constellation outside the ViewEvent state machine
```

---

## Re-review — 2026-10-10T18:57:23-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 430 — couch exits when the outer terminal stalls 5s: degrade the view, keep the sessions |
| repo | pair |
| issue file | workshop/issues/000430-couch-exits-when-the-outer-terminal-stalls-5s-degrade-the-view-keep-the-sessions.md |
| boundary | whole-issue close |
| milestone | — |
| window | c39ed7243b1765b44d173f6063abe37f98d93998..2017402e181f25f85af9c992ff8edae5e6794ee9 |
| command | sdlc close --issue 430 |
| reviewer | claude |
| timestamp | 2026-10-10T18:57:23-07:00 |
| verdict | SHIP |

## Review

Prior fixes look sound in the code. Now I'm checking the tests and running the suite.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** Both Important findings from round 1 are fixed in code and covered by a regression test. The later fixes are also sound: a failed presenter no longer re-arms the retry, a stalled terminal that then returns EIO now fails, and the timeout is classified by which deadline was binding. I ran the full `cmd/internal/terminal` suite outside the sandbox, because the sandbox blocks PTY tests, and it passed (`ok … 46.5s`). That run includes the real-PTY stall test and the earlier tests in `presenter_stall_test.go` and `exitreason_test.go`. One minor gap remains, described under Minor findings. Nothing blocks shipping.

### 1. Strengths
- **BR-1 fixed.** In `EmitEffects`, empty writes (`s == ""`) no longer touch the outer terminal, and a stalled write returns nil (`presenter.go:846-852`). Sequence numbers and the `delivered` list always advance. The test checks that a stalled batch with a no-write bell and a dropped title returns both effects as delivered, and the old early-return would fail it.
- **BR-2 fixed by the general rule.** `writeUnlessStalled` drops its bytes without probing while stalled (`presenter.go:303-311`), and `stalled` is cleared only at `painted := err == nil` (`presenter.go:484-486`). The test calls `Copy` right after `close(gate)` and asserts that recovery starts with `stallProbe`. With the old behaviour the copy's OSC 52 bytes would come first, so the test would fail.
- **The probe tells slow from gone.** Only its own 100 ms deadline means "still stalled" (`presenter.go:292-294`); any other error goes through `p.fail`. `TestPresenterStalledThenGoneFails` pins this.
- **The timeout classification no longer races.** `own := !bounded || deadline.After(budget)` (`presenter.go:227-229`) replaces the racy `ctx.Err()` check, and the #383/#409 tests that set a caller deadline keep failing the presenter, as they should.
- **Lessons recorded.** The three entries in `workshop/lessons.md` state the rules (only the repairing operation clears a degraded flag; drop the bytes, not the event; classify by the binding deadline), not just the sites where they broke.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **A title change during a stall stays lost.** A `TitleEffect` emitted while stalled is dropped and marked delivered, and the full repaint does not re-emit the current title, so the outer window keeps a stale title until the next title effect. This is the 2nd finding in family `stall-recovery-must-repaint`. The rule covering both instances: **recovery must restore everything the presenter knows the outer terminal should show**, meaning screen, modes, keyboard stack, alt-screen push and title. That is the same parent-screen value the Log defers to a refactor (BR-4 and the round-2 alt-screen/kitty-push window). Measured prevalence: two instances so far (alt-screen push, title). The other dropped effects (bell, notification, clipboard) are one-shot events, not state, so they don't belong to this family.
- **Hard-to-read atlas sentence.** `atlas/couch.md:2193-2196`: "The stall wording below is left for a write bounded by a caller's deadline" still reads awkwardly.

### 5. Test coverage notes
- **Covered:** stall then drain on the fake writer, the stalled batch of effects, the copy right after the drain, a single keyboard push, the recovered screen matching a fresh presentation by emulator replay, release after recovery, EIO and EPIPE, a stall followed by EIO, and the real-PTY stall.
- **Not covered:** recovery while a panel (rather than an endpoint) is selected, through the `repaint` → `paint(panel)` branch. It shares `paintPublication`, so the risk is low.
- **Unchanged from round 1:** "endpoints stay alive" is still inferred from the presenter staying `Ready`; no test asserts on it directly.

### 6. Architectural notes
- **ARCH-DRY: pass.** `repaint` is shared by `Refresh` and the retry. `writeUnlessStalled` is the one path for one-shot controls (effects and copy).
- **ARCH-PURE: pass.** The classification is a small predicate (`stall`, plus `Expired` set in `write`), and the IO stays in the thin presenter loop.
- **ARCH-PURPOSE: pass.** The diagnostic was reduced to the `Stalls()` counter, and that is now explicitly recorded in Revisions and the Done-when.
- **ARCH-MOCK: pass.** A stateful `ttyio.Fake` with blocking steps, plus a real PTY.
- **ARCH-CONSTRAINTS: pass.** The first stall still costs 5s. Each probe is capped at 100 ms per painting call, retries run every 250 ms only while stalled and not `Failed`, and admission is bounded per caller.
- **ARCH-SECURE: N/A.** No new untrusted input and no secrets; the probe bytes are a constant.
- **ARCH-ORDER: minor flag (BR-4, still open).** `stalled`, `modesKnown`, `previous`, `history` and `keyboardOwned` are still loose fields. The tests do inject ordering through a gate channel, which is good.
- **ARCH-FUNERAL: pass.** Nothing durable is created; the retry timer lives inside the run loop.

### 7. Plan revision recommendations
- The Spec still says to keep the in-flight write "blocked, with no deadline", but the code abandons the write after `WriteTimeout` and then probes. A one-line `## Revisions` entry would remove the contradiction; the Plan already describes the actual behaviour.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      s=="" skips the terminal, a stalled write returns nil and bookkeeping proceeds (presenter.go:846-852); the stalled-batch test asserts 2 delivered effects.
  - id: BR-2
    disposition: addressed
    note: |
      writeUnlessStalled drops without probing; only a completed paint clears stalled (presenter.go:484-486); Copy-after-drain test asserts recovery begins with stallProbe.
  - id: BR-3
    disposition: addressed
    note: |
      Revisions entry plus Done-when line scope the diagnostic to Presenter.Stalls() and name the follow-up (trace or crash dir).
  - id: BR-4
    disposition: not-addressed
    note: |
      Still loose fields; the Log explicitly defers moving parent-screen knowledge into one value; minor, non-blocking.
findings:
  - id: new
    severity: Minor
    family: stall-recovery-must-repaint
    title: |
      A TitleEffect dropped during a stall is never re-emitted on recovery, so the outer title stays stale
    detail: |
      2nd instance in this family (after the alt-screen kitty-push window). Rule: recovery must restore everything the presenter knows the outer terminal should show (screen, modes, keyboard stack, alt push, title), which is the deferred single parent-screen value that stall() resets whole. Measured prevalence: 2 instances (alt push, title); bell, notification and clipboard are one-shot events, not state.
```
