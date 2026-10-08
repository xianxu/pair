# Boundary Review — pair#412 (milestone M3)

| field | value |
|-------|-------|
| issue | 412 — Couch broadcast: remote pointer link (tap and draw fading marks on the operator's screen) |
| repo | pair |
| issue file | workshop/issues/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | e8b6d73dbaf176db9ccfec0530db78e51353e25d..f663b220c2fe6f3d0d59edc744ad337f1ec6dcc5 |
| command | sdlc milestone-close --issue 412 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-08T10:13:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: fix then ship.** M3 does what the plan says. While live, the status row reads `LIVE ⏸ 👆 👽` at the fixed column the hub's pointer check reads. Clicks on `👆` toggle pointing and copy the link, and a right-click copies it again. Points travel through the Run loop, which checks the pointer phase, the broadcast and the switcher again before drawing. Turning pointing off, the watch firing and the broadcast ending all clear the marks. The M2 commit in this window adds a generation number and a stopped flag, which fixes BR-9.

I checked the generation fix rather than taking the commit message's word for it. `ArmPointer` and `DisarmPointer` wait for the hub's goroutine to finish (`hub.do`), so reading the generation when the watch fires can't pick up a later re-enable.

The pointer and status-row tests pass under `-race` at HEAD. I removed the switcher check (`!c.focus.IsPanel()`) in a scratch copy, and `TestPointerDroppedWhileSwitcherOpen` failed as it should.

Nothing blocks this milestone. Before crossing it:
- **Important:** README.md doesn't mention the new `👆`/`👽` controls.
- **Minor:** two lifecycle details and one plan-prose item, listed below.

1. **Strengths**
   - `console_pointer.go:11-19`: the phase is a three-value enum, and its header comment holds the transition table. That matches ARCH-ORDER.
   - The lock layout follows the plan. `pmarks.mu` is a leaf: `refreshMarks` calls `Presenter.Refresh` before taking it, and `clearMarks` releases it first. A stress test exercises this under `-race`.
   - `applyPoints` (`console_pointer.go:141`) checks again on the loop that pointing is on, the broadcast is live and no panel is open. The late-batch test and the mutation check above both catch a missing check.
   - `RenderStatusRow` returns empty spans when the controls are clipped, and `TestStatusRowPointerCells` shows `👆` clips before `LIVE ⏸` and that `PointerShown` accepts only the active marker.
   - The pointer link is copied but never drawn in a notice, and a test checks this.
2. **Critical:** none.
3. **Important**
   - README.md (near line 812) still describes only `LIVE ⏸`. The new mouse controls are user-facing: left-click `👆` toggles pointing and copies the link, right-click copies it again, and `👽` is inert. No step in the plan (including Task 4.2) updates the README.
4. **Minor**
   - `startBroadcast` passes the Console's `OnPoints`/`OnPointerOff` without tying them to the session that triggered them. A stale batch or watch report from an old broadcast could reach a new one if it ran after the operator stopped, restarted and re-enabled pointing. The effect is a short-lived mark, or a spurious "off" that fails safe.
   - `endBroadcastForShutdown`, and `broadcastStarted` when `SetTap` fails, don't go through `detachBroadcastScreen`. On those paths the fade timer isn't stopped and the overlay stays installed, though the plan's lifetime notes say the timer stops at Console teardown. It's harmless today: the overlay draws nothing without marks, and the timer's callback fails quietly once the Console stops.
   - BR-1 is still open: plan Task 3.2 still lists test cases in prose.
5. **Test coverage**
   - Good end-to-end coverage: the operator's screen and the viewer emulator both show the mark, no byte reaches the child, the fade runs to nothing, the toggle keeps the same link, the watch fires on a clipped `👆`, and marks clear when the broadcast ends.
   - Of the two orders the plan asks for, the late batch after off has a direct test. Batch then off is covered only by the toggle test's mark clearing, which is acceptable.
   - Not tested here: a stroke over `LIVE ⏸ 👆` at the Console level. Marks drop the status row in `Add` and `Overlay` (M1), and the session drops points against a stale grid, so the risk is low.
6. **Architecture**
   - ARCH-DRY: pass. The drawer and the checker share `PointerLabel`/`PointerSGR`, and `detachBroadcastScreen` merges the two end paths.
   - ARCH-PURE: pass. `RenderStatusRow` stays pure and the Console is thin glue.
   - ARCH-PURPOSE: pass for M3, apart from the README gap.
   - ARCH-MOCK: pass. Tests use `FakeTunnel` and the real HTTP server.
   - ARCH-CONSTRAINTS: pass. Refreshes are bounded by the 30/s rate limit, and the timer runs only while marks live.
   - ARCH-SECURE: pass. Coordinates reach only `Marks`, the link never reaches a notice, and the Console's checks repeat the session's.
   - ARCH-ORDER: pass, with the session-identity note above.
   - ARCH-FUNERAL: flagged as Minor, for the shutdown and failed-`SetTap` paths above.
7. **Plan revisions**
   - Add a README step for the status-row controls to M4 (or do it now).
   - Make Task 3.2 a one-line pointer to the Test strategy section (BR-1).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan Task 3.2 still enumerates console test cases in prose; the plan file is unchanged in this window.
findings:
  - id: new
    severity: Important
    family: readme-covers-new-surface
    title: |
      README.md doesn't document the new status-row controls (left-click 👆 toggles pointing and copies the link, right-click re-copies, 👽 inert)
    detail: |
      README.md:808-819 describes only LIVE ⏸. M3 adds mouse controls an operator clicks, and no plan task (including 4.2) updates the README. Add a short paragraph now, or add an explicit M4 README step via a plan Revisions entry.
  - id: new
    severity: Minor
    family: stale-observation-acts-on-new-generation
    title: |
      The Console's OnPoints and OnPointerOff aren't tied to the session that triggered them, so a late callback from an old broadcast can act on the next one
    detail: |
      This is the 2nd finding in this family; BR-9 fixed the session-layer instance. The rule: every deferred report (watch fire, point batch, timer) carries the generation it observed, and its consumer compares that with the current generation before acting. Sweep: startBroadcast binds cfg.OnPoints/OnPointerOff to the Console unconditionally (console_broadcast.go:117-122). Bind a closure capturing the attempt and check c.bcast.attempt in applyPoints and pointerOffByWatch, as broadcastEnded already does with s. Reaching it needs a request goroutine descheduled across an operator stop, start and enable; the impact is a transient mark or a fail-safe 'off'.
  - id: new
    severity: Minor
    family: teardown-path-skips-detach
    title: |
      endBroadcastForShutdown and a failed SetTap in broadcastStarted bypass detachBroadcastScreen, leaving the fade timer armed and the overlay installed
    detail: |
      ARCH-FUNERAL. The plan's lifetimes section says the fade timer stops at Console teardown, but only stopBroadcast and broadcastEnded call resetPointer and SetOverlay(nil). Every path that ends or abandons a broadcast should go through detachBroadcastScreen. It's harmless today (no marks means the overlay does nothing, and the timer's command fails once stopped), but the contract isn't met as written.
```

---

## Re-review — 2026-10-08T10:14:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 412 — Couch broadcast: remote pointer link (tap and draw fading marks on the operator's screen) |
| repo | pair |
| issue file | workshop/issues/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | e8b6d73dbaf176db9ccfec0530db78e51353e25d..768c4d702f2d29ba09873656a1dc564c9905e5d5 |
| command | sdlc milestone-close --issue 412 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-08T10:14:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Since the last round, the window has one change: the README commit `ed2e412e`. The other commit is the review ledger. That README paragraph closes the only Important finding, BR-10. I checked what it says against the code. The overlay skips private frames (`console_pointer.go` `markOverlay`). `Marks.Add` and `Marks.Overlay` both leave out the last row (`marks.go:42`, `:185-188`). The right-click re-copies the link. Clicking `👽` only shows a notice. Turning pointing off and on again keeps the same link (`TestPointerToggleOffAndOnKeepsTheLink`). The three Minor findings still open (BR-1, BR-11, BR-12) are unchanged in this window. They are not-addressed, but none of them blocks the gate. I found nothing new.

1. **Strengths**
   - `README.md:840-851` matches how the code behaves. It also states the safety guarantees: marks never send input to any program, never cover the status row, and the link is never drawn on screen.
   - `detachBroadcastScreen` (`console_broadcast.go:301`) is now the one place where both the operator stop and an unexpected end detach the screen. Because it is shared, closing BR-12 only means calling it from two more places.
   - The two pointer generation guards, `offIfGeneration` and `stopped` (`pointer.go:100-148`), each have a test that targets them directly.
2. **Critical:** none.
3. **Important:** none.
4. **Minor:** BR-1, BR-11 and BR-12 carry over unchanged (see the dispositions below).
5. **Test coverage:** no new code or tests in this window. The earlier round's coverage notes still apply. A README wording change needs no test of its own.
6. **Architecture:**
   - ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK, ARCH-CONSTRAINTS and ARCH-SECURE: pass. The README now covers the surface ARCH-PURPOSE requires.
   - ARCH-ORDER: pass, with BR-11 still open.
   - ARCH-FUNERAL: flagged, with BR-12 still open.
   - Fixing BR-11 and BR-12 is cheap, and doing it before M4 adds the viewer-side surface would keep those two families from coming back.
7. **Plan revisions:**
   - Make plan Task 3.2 (and Tasks 1.3, 2.2 and 2.3) a one-line pointer to the Test strategy section (BR-1).
   - Optionally, add a Revisions note recording that the README step was done in M3 rather than M4.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan Task 3.2 still enumerates test cases in prose; the plan file is unchanged in this window.
  - id: BR-10
    disposition: addressed
    note: |
      README.md:840-851 (ed2e412e) documents the three click behaviours of 👆 and 👽 and the end-of-broadcast rule; the claims match markOverlay, Marks.Add/Overlay and the existing pointer tests.
  - id: BR-11
    disposition: not-addressed
    note: |
      startBroadcast still binds c.onPoints and c.pointerOffByWatch unconditionally (console_broadcast.go:116-122); no attempt check in applyPoints or pointerOffByWatch.
  - id: BR-12
    disposition: not-addressed
    note: |
      endBroadcastForShutdown and the failed-SetTap branch of broadcastStarted still skip detachBroadcastScreen; unchanged in this window.
```
