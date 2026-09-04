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

---

## Re-review — 2026-09-04T15:52:27-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 185 — Status-row notices never expire, so a momentary refusal reads as current state |
| repo | pair |
| issue file | workshop/issues/000185-status-row-notices-never-expire-so-a-momentary-refusal-reads-as-current-state.md |
| boundary | whole-issue close |
| milestone | — |
| window | 88fe1de011b4c6be58e5a8b20eed89dfa4000f5d..5b08b7a7b9d4c4dfa031ad866d6494ba366545c5 |
| command | sdlc close --issue 185 |
| reviewer | claude |
| timestamp | 2026-09-04T15:52:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Critical from round 1 is genuinely fixed and genuinely pinned: I reverted `publishNotice`'s repaint in a scratch copy and both console tests go red (`console_notice_expiry_test.go:46` and `:77`, 3s timeouts), so the test is not written from the same mental model as the fix — it fails without it. BR-2's false rationale is corrected in all four places it lived, and the four Spec policy rules each have a pure test on a hand-advanced clock. What blocks SHIP is one Important: the issue's own Done-when asserts `go test -race ./cmd/internal/couchtty/` passes, and it doesn't — `TestConsoleRunOrdinarySwitchAdvancesPrevious` fails 3 of 10 race runs. Not a #185 regression (it fails 4 of 6 at the pre-#185 commit and the test doesn't exist at the window base), but it falsifies the verification this close will record, and I confirmed both the mechanism and a 3-line fix that makes it 10/10.

## 1. Strengths

- **The BR-1 fix is verified by reversion, not by assertion.** Scratch revert of `console.go:1689-1697` → both `console_notice_expiry_test.go` tests time out. The removed hand-`repaint()` in the test was correctly identified as the tell, and its removal is what makes the test load-bearing.
- **`publishNotice` swept the whole class, not the one site the finding named.** The two paths that bypassed `setNotice` entirely — `SetInputTrace`'s trace failure (`console.go:210`) and `onExit`'s exit notice (`console.go:856`) — now go through it. And the exit push was correctly moved *out* of the `c.mu` critical section, which is mandatory: `repaint`→`paintNow` takes the same mutex, so leaving it inside would have self-deadlocked. I checked all six `setNotice` callers; none holds the lock.
- **BR-2's correction is complete rather than cosmetic.** Restated at `console.go:665-671`, `atlas/couch.md:922-931`, the Spec's consequence 3, and the commit body, with a `workshop/lessons.md` rule ("never document a line as correctness-critical without testing the claim").
- **Pure/IO split is right** (ARCH-PURE): `Feed.Row()`/`Push()` are pure policy tested on a fake clock (`notice_test.go:44-127`); the timer and paint live in `Run`/`publishNotice`. The insight that injecting *only* the clock produced a console test that proved nothing is the best thing in this diff, and it's captured as a lesson.
- **Untrusted text is already contained** (ARCH-SECURE): thread labels from persisted records and raw `err.Error()` bodies reach the reserved row only through `sanitize` (`reserve.go:121,130`), which strips escape sequences then C0/DEL.

## 2. Critical findings

None.

## 3. Important findings

**`cmd/internal/couchtty/console_run_menu_test.go:390` — the package's race suite is flaky, so #185's Done-when is not a check that passes.**

Measured at HEAD: 3 failures in 10 `-race` runs of `TestConsoleRunOrdinarySwitchAdvancesPrevious`; the full-package `-race` run failed on it too. Non-race is stable (3/3 clean). Mechanism, confirmed by instrumenting a scratch copy: on failure the frame reports `SelectedAddress={legacy c1}`, not `c2`. The fixture's inventory refresh already creates a root frame via `reconcileMenuFrames` (`menu.go:1267-1271`), so the test's `waitUpTo(..., "the switcher", len(Frames) > 0)` is satisfied **before** ctrl-space is processed. The test then writes `menu.Frames[0].SelectedAddress = two`, `onHotkey` runs afterwards and reopens the switcher focused on the thread being left (`c1`), and `\r` selects `c1`.

Fix sketch — wait for the switcher to actually own focus instead of for a frame to exist:

```go
waitUpTo(t, 250*time.Millisecond, "the switcher", func() bool {
    f.con.mu.Lock()
    defer f.con.mu.Unlock()
    return f.con.focus.IsPanel()
})
```

Verified: 10/10 green under `-race` with that change. (The sibling `TestConsoleRunNotificationHopThenPreviousReturnsHome` is immune because it waits on `SelectedAddress == two`, which only becomes true after the key is processed.)

## 4. Minor findings

- `console.go:1689` — **2nd finding in family `row-must-track-the-feed`.** `publishNotice` owns the paint half of "the row must track the feed"; the Run loop owns the timer half (`syncNoticeExpiry` at `console.go:758`). They coincide only because every post-start publisher happens to *be* the Run goroutine. Don't patch the instance — state the rule: *one seam owns push + paint + deadline re-sync*, e.g. `publishNotice` doing a non-blocking send on a `noticeChanged` channel the select drains. Enumeration today: 6 `setNotice` sites (all Run-goroutine), `onExit` (Run), `SetInputTrace` (pre-Run, `started == false`) — so 0 live instances, which is why this is Minor. Same seam: `started` (`console.go:517`) is never cleared at teardown, so a post-`Run` publish would paint into a restored terminal.
- `cmd/internal/couchtty/console_notice_expiry_test.go:52` — the new test discards the evidence it is waiting for. `host.Reset()` runs after a poll loop; if that loop takes longer than the 60ms `testLifetime` on a loaded box, the expiry paint lands before the Reset and the test hangs to its 3s timeout. Reproduced by injecting `time.Sleep(3 * testLifetime)` before the Reset in a scratch copy → FAIL. Assert the final painted row lacks the sentence rather than resetting the buffer. (40 `-race` runs were clean as written, so it's latent, not live.)
- `console.go:1691` — `f.feed.Push(n)`'s `bool` is discarded at its only production call site, though `Enqueue`'s own doc says a full mailbox should be "a loud bug signal rather than flow control" and deliberately gave it a fallible signature. Pre-existing, but `publishNotice` is the new line that drops it.

## 5. Test coverage notes

- All five behavioral Done-whens have tests: four pure (`notice_test.go:44,90,110,122`) on a hand-advanced clock, two console-level on a real `time.Timer`. I stress-ran the two console tests 40× under `-race`: clean.
- `NewFeed`'s defaults (BR-4) remain unreachable and untested — three call sites, all passing a real clock and positive lifetime.
- Package-wide: `go build ./...` clean; non-race `./cmd/...` failures are all `operation not permitted` from `ptychild` (the known pty environment limit here), none from this diff.

## 6. Architectural notes

- **ARCH-DRY** — flag, but it's BR-3 unchanged: `syncNoticeExpiry` is the fourth stop/reset/assign-channel closure in `Run`. Still owed as a shared `arm(&timer, &ch, d)`; nothing new added to the pile.
- **ARCH-PURE** — pass. See Strengths.
- **ARCH-PURPOSE** — pass this round. I ran the shadow-sweep on the class "a transient message must retire": the only other surface that shows refusals is the switcher banner, and it already retires them on the next keystroke via `clearsPreviousNotice` (`menu.go:299`). No hand-maintained sibling left behind.
- **ARCH-MOCK** — pass. No external binary; `hostty.NewFakeHost` + `ptychild.NewFakeChild` are the seams, and the console test drives a *real* `time.Timer` through them rather than faking the mechanism under test.
- **ARCH-CONSTRAINTS** — pass. Keystroke path: `publishNotice` adds one synchronous one-line row write; `syncNoticeExpiry` adds one mutex acquisition plus a ≤8-element walk per Run-loop iteration, which is on the child-output hot path but O(capacity). The dedup is now honestly labelled churn-avoidance.
- **ARCH-SECURE** — pass. See Strengths. `Feed` has no untrusted persisted state of its own.

## 7. Plan revision recommendations

None for the Plan itself — all three checkboxes are delivered as written, and the Spec matches the code including the two corrections. One `## Revisions` entry is owed to the **Done when** list once the Important is disposed: either fix the flake and keep `go test -race ./cmd/internal/couchtty/ passes`, or restate what was actually verified. Leaving it as-is records a check that fails 3 runs in 10.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified by reversion: removing publishNotice's repaint turns both console tests red (timeouts at :46 and :77).
  - id: BR-2
    disposition: addressed
    note: |
      Restated as an optimisation in console.go:665, atlas/couch.md:922, the Spec's consequence 3, and the commit body.
  - id: BR-3
    disposition: not-addressed
    note: |
      No shared arm helper; syncNoticeExpiry still repeats stop/reset/assign. Minor, does not block.
  - id: BR-4
    disposition: not-addressed
    note: |
      notice.go:94-99 still defaults a nil clock and lifetime<=0; all three call sites still pass real values.
  - id: BR-5
    disposition: addressed
    note: |
      console.go:722 now clears noticeExpiry in the fire branch; untestable by construction, as the finding itself stated.
  - id: BR-6
    disposition: addressed
    note: |
      README.md:300-303 documents that transients go after ~12s while an exit stands.
findings:
  - id: new
    severity: Important
    family: test-writes-state-the-loop-rebuilds
    title: |
      The package's -race suite fails 3 runs in 10, so the issue's "go test -race passes" Done-when is not a check that passes
    detail: |
      TestConsoleRunOrdinarySwitchAdvancesPrevious (console_run_menu_test.go:390) waits for
      len(Frames) > 0, but the fixture's inventory refresh already built a root frame
      (menu.go:1267), so the wait returns BEFORE ctrl+space is processed. The test's direct
      write of menu.Frames[0].SelectedAddress is then reset by onHotkey reopening the
      switcher on the thread being left; instrumented failures report SelectedAddress=c1.
      Not a pair#185 regression (4 of 6 failures at the pre-185 commit, and the test does
      not exist at the window base), but it falsifies the verification this close records.
      Waiting on f.con.focus.IsPanel() instead makes it 10 of 10 green under -race; verified.
  - id: new
    severity: Minor
    family: row-must-track-the-feed
    title: |
      publishNotice owns the paint half of the row-tracks-the-feed rule; the Run loop still owns the deadline half
    detail: |
      This is the 2nd finding in family row-must-track-the-feed. Do NOT fix the instance.
      The rule: one seam owns push, paint, AND expiry re-sync. Today publishNotice
      (console.go:1689) pushes and paints, while syncNoticeExpiry re-arms only at the bottom
      of the Run select (console.go:758) -- so a notice published off the Run goroutine paints
      but arms no timer and never retires. Measured prevalence: 0 live instances (all 6
      setNotice sites and onExit are Run-goroutine; SetInputTrace publishes pre-Run with
      started == false), which is why this is Minor rather than a live defect. Same seam:
      c.started (console.go:517) is set but never cleared at teardown.
  - id: new
    severity: Minor
    family: test-discards-evidence-it-awaits
    title: |
      The new console expiry test resets the host buffer after a poll loop that can outlast the 60ms lifetime
    detail: |
      console_notice_expiry_test.go:52 calls host.Reset() after waiting for the sentence to
      appear. On a loaded machine that wait can exceed testLifetime, so the expiry repaint
      lands before the Reset and the second waitFor hangs to its 3s timeout. Reproduced by
      injecting time.Sleep(3 * testLifetime) before the Reset in a scratch copy -> FAIL.
      Assert that the LAST painted row lacks the sentence instead of resetting the buffer.
      Latent, not live: 40 -race runs clean as written.
  - id: new
    severity: Minor
    family: discarded-invariant-signal
    title: |
      Feed.Push's ok return is discarded at its only production call site
    detail: |
      console.go:1691 ignores the bool. Enqueue's own doc says a full mailbox should be "a
      loud bug signal rather than flow control" and gave the function a fallible signature
      deliberately; nothing consumes it. Pre-existing behaviour, but publishNotice is the new
      line that drops it.
```
