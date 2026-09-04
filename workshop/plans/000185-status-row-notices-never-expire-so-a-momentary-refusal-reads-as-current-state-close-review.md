# Boundary Review — pair#185 (whole-issue close)

| field | value |
|-------|-------|
| issue | 185 — Status-row notices never expire, so a momentary refusal reads as current state |
| repo | pair |
| issue file | workshop/issues/000185-status-row-notices-never-expire-so-a-momentary-refusal-reads-as-current-state.md |
| boundary | whole-issue close |
| milestone | — |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..319a3f9f49293479fffec62d78d5c69efb1b3c41 |
| command | sdlc close --issue 185 |
| reviewer | claude |
| timestamp | 2026-09-04T15:33:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The expiry mechanism itself is well built and genuinely test-pinned: I verified by mutation that removing the Run-loop arming turns `TestAnIdleConsoleRepaintsWhenItsNoticeExpires` red (times out), and that removing the expiry-map prune turns `TestExpiryBookkeepingStaysWithinCapacity` red (50 entries for a capacity-2 feed). The Control obligation/event split is the right axis, `Row()`'s tail-walk is correct, the clock+lifetime pair really is load-bearing, and `paintNow` reads the feed under `c.mu` so `-race` is clean. What blocks a plain SHIP is the other half of the same rule: the row now repaints when a notice *goes* but still nothing repaints when one *arrives*, so on an idle console a ctrl+backspace refusal is pushed, armed, and then erased 12s later without ever having been on screen — I confirmed this with a throwaway probe (since removed): the only paint the refusal ever caused was the one that removed it. Separately, the dedup guard is documented — in the code comment, the Spec, the atlas and the commit message — as the thing that stops the deadline being pushed out forever, and that claim is false; deleting the guard leaves the whole package green and the deadline intact, because `Standing` shrinks.

**1. Strengths**

- `cmd/internal/couchtty/notice.go:128` — `Row()` walking from the tail with "an older transient is never a better answer" is the correct rule, and `TestAnExpiredNoticeUncoversOnlyWhatIsStillTrue` pins both directions (control uncovered, older transient not).
- `cmd/internal/couchtty/notice.go:104` — keying expiry on `Message.Kind`, which *is* `couchcore.Enqueue`'s collapse identity, means a replaced notice inherits the slot. I checked `couchcore/mailbox.go:32`: collapse is unconditional on Kind, so no queue can ever hold a control and a transient sharing a key — the aliasing hazard this design could have had doesn't exist.
- `console_notice_expiry_test.go:41` — driving ctrl+backspace as *bytes* rather than calling `setNotice` is the right instinct, and it is what makes the arming reachable. Same lesson as pair#182's BR-16, correctly applied.
- The prune-to-retained loop (`notice.go:120`) is a real invariant, not decoration — verified red.
- `atlas/couch.md` extended in the same commit with both the standing rule and the repaint obligation (ARCH docs gate satisfied for the new surface).

**2. Critical findings**

- `cmd/internal/couchtty/console.go:650` (and `console.go:1650` `setNotice`) — **a transient notice is now guaranteed to expire unseen on an idle console.** `setNotice` does not repaint, and the only paint triggers are child row-dirty/bell output, resize, switch, exit, and the new expiry timer. On a pane sitting at a prompt with no output, pressing ctrl+backspace with nowhere to return pushes the notice, arms a 12s timer, and the timer's repaint renders the row *without* it. Probe output on a fresh console after the keystroke and one lifetime: `\x1b[1;23r\x1b7\x1b[24;1H\x1b[2K[brain]\x1b8` — the actor label, no sentence, ever. This is precisely the outcome `reportPrevious`'s own comment (`console.go:1258`) says must not happen ("would make the key silently do nothing — which is exactly what the operator would report as the bug"). The no-paint-on-push half is pre-existing, but the expiry converts it from "shown late" into "never shown", which is why it lands here rather than as a deferral (ARCH-PURPOSE: the row must be an honest projection of the feed *when it changes*, in both directions; the diff delivers one direction). Note the diff's own console test compensates for this by calling `con.repaint()` before testing expiry — a test working around the defect. Fix sketch: have `syncNoticeExpiry` (already running at the bottom of every iteration, already reading the row under the lock) remember the last row body and `c.repaint()` when it changed; that covers every producer, keeps the paint on the Run goroutine, and is symmetric with the expiry repaint. Pin it with a console test that writes `previousByte` and asserts the sentence reaches `host.Written()` with **no** explicit `repaint()` — that test is red today.

**3. Important findings**

- `cmd/internal/couchtty/console.go:658` — **the dedup guard's stated rationale is false.** The comment ("Without this an event-heavy console would re-arm on every iteration and the notice would never actually go"), the Spec's consequence 3, the atlas paragraph and the commit message all assert the guard is what preserves the deadline. It isn't: the re-arm uses `row.Standing`, which is `expires - now` and shrinks, so re-arming every iteration still fires at the same absolute instant. I verified by deleting the guard — the whole `couchtty` package stays green (only the environment-blocked PTY conformance test fails), i.e. the guard is also entirely untested. It is a worthwhile *efficiency* guard (no Stop/Reset churn per event); it is not a correctness one. Fix: restate the comment, the Spec and the atlas as "avoids per-event timer churn", so the codebase map stops recording an invariant the code does not depend on.

**4. Minor findings**

- `console.go:646` — `syncNoticeExpiry` is the fourth near-identical arm-a-timer closure inside `Run` (`armInputEscape`, `armPanelEscape`, `syncSpinner`). Following the established shape was the right local call, but a shared `arm(&timer, &ch, d)` helper is now clearly owed (ARCH-DRY).
- `notice.go:93` — `NewFeed`'s `now == nil` and `lifetime <= 0` defaults are unreachable from all three production/test call sites and untested; a future caller asking for "no expiry" with `0` silently gets 12s instead.
- `console.go:702` — the `case <-noticeC` branch clears `noticeC` but leaves `noticeExpiry` stale, relying on the bottom-of-loop sync to zero it. Correct today (an expired row always reports zero `Expires`), but the invariant is implicit.
- `README.md:~305` documents that couch reserves the bottom row for a status line; one sentence that transient notices retire after ~12s while an exit notice stands would close the user-visible-timing gap. Not a new command/flag/keybinding, so the README gate otherwise passes.

**5. Test coverage notes**

- Four pure policy rules each have a test, and two of them I confirmed red under mutation. The fifth behaviour in the Run loop — the arming — is pinned by the console test, also confirmed red.
- Gap: nothing covers a notice **arriving** on screen (the Critical above). Gap: nothing covers the dedup (removing it is green).
- Suite state: `go test ./...` fails only with `operation not permitted` from pty allocation across `couch`, `couchcmd`, `couchcore`, `couchtty`, `hostty`, `keyscmd`, `ptychild`, `termcmd`, `wrapcmd`, `pair-go` — the known environment restriction on spawning pty children, unrelated to this diff (`notification_pty_test.go` is untouched). `go test -race ./cmd/internal/couchtty/` passes apart from that one. The Done-when's `-race` claim should be re-confirmed by the operator on a shell that can allocate a pty.

**6. Architectural notes**

- ARCH-DRY — flag (Minor, above): fourth timer-arm closure.
- ARCH-PURE — pass. `Feed` is genuinely pure by injection; its tests hand-advance a fake clock and touch no IO. The repaint half correctly lives in the `Run` shell, which is the thin seam.
- ARCH-PURPOSE — flag (Critical, above). Shadow-sweep of the notice surfaces: the status row (`Feed`) and the panel (`MenuState.Notice`). The panel surface already retires on the operator's next gesture (`menu.go:299 clearsPreviousNotice`), a deliberately different rule, so the class "a notice with no retirement" is now closed. The class that remains open is "a notice with no *appearance*", which the fix should sweep for both producers (`setNotice` and the exit `feed.Push`), not just for the previous-hotkey site.
- ARCH-MOCK — pass. The clock is the external dependency and is behind an injected seam; the console test runs the production timer against a shortened lifetime, which is the live-conformance half rather than a stateless mock.
- ARCH-CONSTRAINTS — pass. One timer per row, dedup'd; O(capacity) prune per push on a capacity-8 queue; nothing added to the keystroke path. The 12s figure is an operator choice and is documented as one.
- ARCH-SECURE — pass. Notice bodies reach the terminal through `sanitize()` (`reserve.go:121`), so an escape sequence in an error string cannot drive the row; no credentials, no untrusted persisted input, no new external boundary.

**7. Plan revision recommendations**

- `## Revisions` entry on the issue: correct consequence 3 of the Spec — the deadline-identity dedup prevents per-event timer churn, not deadline extension; arming from `Standing` already preserves the absolute deadline. Mirror the correction in `atlas/couch.md` and the `console.go` comment.
- `## Revisions` entry: promote the Log's deliberately-unanswered question ("whether `setNotice` should repaint") from an open note to either a Done-when item in this issue or a filed follow-up issue. The diff makes the answer load-bearing — with an expiry in place, "no paint on push" is no longer a latency question but a visibility one.

```findings
findings:
  - id: new
    severity: Critical
    family: row-must-track-the-feed
    title: |
      A transient notice now expires unseen on an idle console — nothing repaints when one is pushed
    detail: |
      setNotice does not repaint and no other paint trigger is guaranteed, so on
      an idle pane a ctrl+backspace refusal is pushed, the 12s timer is armed, and
      the timer's own repaint renders the row WITHOUT the sentence — the operator
      never sees it. Probe on a fresh console after one lifetime painted only
      "[brain]". This is the outcome reportPrevious's comment at console.go:1258
      says must not happen. Fix: have syncNoticeExpiry repaint when the row body
      changed (it already reads the row under the lock every iteration), and pin it
      with a console test that writes previousByte and asserts the sentence reaches
      the host with NO explicit repaint() — red today. ARCH-PURPOSE: the retirement
      half shipped, the appearance half did not.
  - id: new
    severity: Important
    family: overstated-guard-rationale
    title: |
      The timer dedup is documented as correctness-critical in code, Spec, atlas and commit message; it is not
    detail: |
      "Without this an event-heavy console would re-arm on every iteration and the
      notice would never actually go" is false: the re-arm uses row.Standing
      (expires - now), which shrinks, so the absolute deadline is preserved either
      way. Verified by deleting the guard — the whole couchtty package stays green,
      so it is also untested. It is a real churn-avoidance optimisation; restate it
      as one in console.go:658, the Spec's consequence 3, and atlas/couch.md so the
      codebase map stops recording an invariant the code does not rely on.
  - id: new
    severity: Minor
    family: repeated-timer-arm-shape
    title: |
      syncNoticeExpiry is the fourth near-identical arm-a-timer closure in Run
    detail: |
      armInputEscape, armPanelEscape, syncSpinner and now syncNoticeExpiry all
      repeat stop/reset/assign-channel. A shared arm(&timer, &ch, d) helper is owed
      (ARCH-DRY). Following the established shape was the right local call.
  - id: new
    severity: Minor
    family: unreachable-constructor-default
    title: |
      NewFeed's nil-clock and lifetime<=0 defaults are unreachable and untested
    detail: |
      All three call sites pass a real clock and a positive lifetime. A future
      caller passing 0 to mean "no expiry" silently gets 12s.
  - id: new
    severity: Minor
    family: implicit-timer-state-invariant
    title: |
      The noticeC fire branch leaves noticeExpiry stale, relying on the bottom-of-loop sync
    detail: |
      console.go:702 clears noticeC but not noticeExpiry. Correct today because an
      expired row always reports a zero Expires, but the invariant is implicit.
  - id: new
    severity: Minor
    family: readme-behavior-timing
    title: |
      README documents the reserved status row but not that transient notices now retire
    detail: |
      One sentence at README.md:~305 — transients go after ~12s, an exit notice
      stands — would document the new user-visible timing. No new command, flag or
      keybinding, so the README gate otherwise passes.
```
