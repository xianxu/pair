---
gate: boundary-review
issue: 185
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-04T15:33:30-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: A transient notice now expires unseen on an idle console — nothing repaints when one is pushed
          detail: |-
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
          family: row-must-track-the-feed
          round: 1
        - id: BR-2
          severity: Important
          title: The timer dedup is documented as correctness-critical in code, Spec, atlas and commit message; it is not
          detail: |-
            "Without this an event-heavy console would re-arm on every iteration and the
            notice would never actually go" is false: the re-arm uses row.Standing
            (expires - now), which shrinks, so the absolute deadline is preserved either
            way. Verified by deleting the guard — the whole couchtty package stays green,
            so it is also untested. It is a real churn-avoidance optimisation; restate it
            as one in console.go:658, the Spec's consequence 3, and atlas/couch.md so the
            codebase map stops recording an invariant the code does not rely on.
          family: overstated-guard-rationale
          round: 1
        - id: BR-3
          severity: Minor
          title: syncNoticeExpiry is the fourth near-identical arm-a-timer closure in Run
          detail: |-
            armInputEscape, armPanelEscape, syncSpinner and now syncNoticeExpiry all
            repeat stop/reset/assign-channel. A shared arm(&timer, &ch, d) helper is owed
            (ARCH-DRY). Following the established shape was the right local call.
          family: repeated-timer-arm-shape
          round: 1
        - id: BR-4
          severity: Minor
          title: NewFeed's nil-clock and lifetime<=0 defaults are unreachable and untested
          detail: |-
            All three call sites pass a real clock and a positive lifetime. A future
            caller passing 0 to mean "no expiry" silently gets 12s.
          family: unreachable-constructor-default
          round: 1
        - id: BR-5
          severity: Minor
          title: The noticeC fire branch leaves noticeExpiry stale, relying on the bottom-of-loop sync
          detail: |-
            console.go:702 clears noticeC but not noticeExpiry. Correct today because an
            expired row always reports a zero Expires, but the invariant is implicit.
          family: implicit-timer-state-invariant
          round: 1
        - id: BR-6
          severity: Minor
          title: README documents the reserved status row but not that transient notices now retire
          detail: |-
            One sentence at README.md:~305 — transients go after ~12s, an exit notice
            stands — would document the new user-visible timing. No new command, flag or
            keybinding, so the README gate otherwise passes.
          family: readme-behavior-timing
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-04T15:52:27-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified by reversion: removing publishNotice''s repaint turns both console tests red (timeouts at :46 and :77).'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Restated as an optimisation in console.go:665, atlas/couch.md:922, the Spec's consequence 3, and the commit body.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: No shared arm helper; syncNoticeExpiry still repeats stop/reset/assign. Minor, does not block.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: notice.go:94-99 still defaults a nil clock and lifetime<=0; all three call sites still pass real values.
          round: 2
        - id: BR-5
          disposition: addressed
          note: console.go:722 now clears noticeExpiry in the fire branch; untestable by construction, as the finding itself stated.
          round: 2
        - id: BR-6
          disposition: addressed
          note: README.md:300-303 documents that transients go after ~12s while an exit stands.
          round: 2
      findings:
        - id: BR-7
          severity: Important
          title: The package's -race suite fails 3 runs in 10, so the issue's "go test -race passes" Done-when is not a check that passes
          detail: |-
            TestConsoleRunOrdinarySwitchAdvancesPrevious (console_run_menu_test.go:390) waits for
            len(Frames) > 0, but the fixture's inventory refresh already built a root frame
            (menu.go:1267), so the wait returns BEFORE ctrl+space is processed. The test's direct
            write of menu.Frames[0].SelectedAddress is then reset by onHotkey reopening the
            switcher on the thread being left; instrumented failures report SelectedAddress=c1.
            Not a pair#185 regression (4 of 6 failures at the pre-185 commit, and the test does
            not exist at the window base), but it falsifies the verification this close records.
            Waiting on f.con.focus.IsPanel() instead makes it 10 of 10 green under -race; verified.
          family: test-writes-state-the-loop-rebuilds
          round: 2
        - id: BR-8
          severity: Minor
          title: publishNotice owns the paint half of the row-tracks-the-feed rule; the Run loop still owns the deadline half
          detail: |-
            This is the 2nd finding in family row-must-track-the-feed. Do NOT fix the instance.
            The rule: one seam owns push, paint, AND expiry re-sync. Today publishNotice
            (console.go:1689) pushes and paints, while syncNoticeExpiry re-arms only at the bottom
            of the Run select (console.go:758) -- so a notice published off the Run goroutine paints
            but arms no timer and never retires. Measured prevalence: 0 live instances (all 6
            setNotice sites and onExit are Run-goroutine; SetInputTrace publishes pre-Run with
            started == false), which is why this is Minor rather than a live defect. Same seam:
            c.started (console.go:517) is set but never cleared at teardown.
          family: row-must-track-the-feed
          round: 2
        - id: BR-9
          severity: Minor
          title: The new console expiry test resets the host buffer after a poll loop that can outlast the 60ms lifetime
          detail: |-
            console_notice_expiry_test.go:52 calls host.Reset() after waiting for the sentence to
            appear. On a loaded machine that wait can exceed testLifetime, so the expiry repaint
            lands before the Reset and the second waitFor hangs to its 3s timeout. Reproduced by
            injecting time.Sleep(3 * testLifetime) before the Reset in a scratch copy -> FAIL.
            Assert that the LAST painted row lacks the sentence instead of resetting the buffer.
            Latent, not live: 40 -race runs clean as written.
          family: test-discards-evidence-it-awaits
          round: 2
        - id: BR-10
          severity: Minor
          title: Feed.Push's ok return is discarded at its only production call site
          detail: |-
            console.go:1691 ignores the bool. Enqueue's own doc says a full mailbox should be "a
            loud bug signal rather than flow control" and gave the function a fallible signature
            deliberately; nothing consumes it. Pre-existing behaviour, but publishNotice is the new
            line that drops it.
          family: discarded-invariant-signal
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-04T16:11:18-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: not-addressed
          note: Still no shared arm helper; console.go:648 remains the fourth stop/reset/assign closure in Run.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: notice.go:94-99 unchanged; notice.go:100 is the only Feed literal in the tree, so both defaults stay unreachable.
          round: 3
        - id: BR-7
          disposition: addressed
          note: Verified by reversion — reverted wait fails 9/20 -race runs here; fixed wait is 100/100, full package 5/5.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: Only the started sub-item was touched, and its guard is TOCTOU-shaped; the class rule (one seam owns push, paint and deadline re-sync) is unchanged.
          round: 3
        - id: BR-9
          disposition: addressed
          note: lastPaintedRow needs no window and the rewrite keeps the pin — removing syncNoticeExpiry's arming still turns the test red; 50/50 green under -race.
          round: 3
        - id: BR-10
          disposition: withdrawn
          note: Counter-argument accepted — for a rolling feed a capacity drop is the policy, so ok=false carries no signal this caller could act on.
          round: 3
      findings:
        - id: BR-11
          severity: Minor
          title: Two comments added this round assert guarantees the code does not deliver
          detail: |-
            This is the 2nd finding in family overstated-guard-rationale. Do NOT fix the
            instances. The rule: a comment that states an absolute must name the condition
            under which it does not hold, and the guarantee must be pinned by a test that
            fails without it. Enumeration in this diff, prevalence 2 of 2 new absolute
            claims. (a) console.go:781 says "nothing may paint into it again", but
            publishNotice (console.go:1694-1700) reads started under c.mu, releases the
            lock, then paints -- so a publish that passes the check before teardown runs
            still writes into the restored shell. Prevalence 0 live, because every
            publisher is the Run goroutine that also runs teardown; the hole opens exactly
            when BR-8's class opens. (b) console.go:1693 describes publishNotice as the one
            way a notice reaches the operator, but writeOwn (console.go:973) declines while
            hostScan.MidSequence() and that debt is paid only by the next child chunk
            (console.go:1079) -- on the idle console this issue is about, there is none.
            Trigger is pathological and takeOverScreen resets hostScan, so 0 live there
            too. Neither is a behavioural defect today; both are the same habit BR-2 named.
          family: overstated-guard-rationale
          round: 3
      blocked: false
---

# Gate ledger — pair#185 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T15:33:30-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `row-must-track-the-feed` A transient notice now expires unseen on an idle console — nothing repaints when one is pushed
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
- **BR-2** [Important] `overstated-guard-rationale` The timer dedup is documented as correctness-critical in code, Spec, atlas and commit message; it is not
  "Without this an event-heavy console would re-arm on every iteration and the
  notice would never actually go" is false: the re-arm uses row.Standing
  (expires - now), which shrinks, so the absolute deadline is preserved either
  way. Verified by deleting the guard — the whole couchtty package stays green,
  so it is also untested. It is a real churn-avoidance optimisation; restate it
  as one in console.go:658, the Spec's consequence 3, and atlas/couch.md so the
  codebase map stops recording an invariant the code does not rely on.
- **BR-3** [Minor] `repeated-timer-arm-shape` syncNoticeExpiry is the fourth near-identical arm-a-timer closure in Run
  armInputEscape, armPanelEscape, syncSpinner and now syncNoticeExpiry all
  repeat stop/reset/assign-channel. A shared arm(&timer, &ch, d) helper is owed
  (ARCH-DRY). Following the established shape was the right local call.
- **BR-4** [Minor] `unreachable-constructor-default` NewFeed's nil-clock and lifetime<=0 defaults are unreachable and untested
  All three call sites pass a real clock and a positive lifetime. A future
  caller passing 0 to mean "no expiry" silently gets 12s.
- **BR-5** [Minor] `implicit-timer-state-invariant` The noticeC fire branch leaves noticeExpiry stale, relying on the bottom-of-loop sync
  console.go:702 clears noticeC but not noticeExpiry. Correct today because an
  expired row always reports a zero Expires, but the invariant is implicit.
- **BR-6** [Minor] `readme-behavior-timing` README documents the reserved status row but not that transient notices now retire
  One sentence at README.md:~305 — transients go after ~12s, an exit notice
  stands — would document the new user-visible timing. No new command, flag or
  keybinding, so the README gate otherwise passes.

## Round 2 — 2026-09-04T15:52:27-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Verified by reversion: removing publishNotice's repaint turns both console tests red (timeouts at :46 and :77).
- BR-2 — addressed — Restated as an optimisation in console.go:665, atlas/couch.md:922, the Spec's consequence 3, and the commit body.
- BR-3 — not-addressed — No shared arm helper; syncNoticeExpiry still repeats stop/reset/assign. Minor, does not block.
- BR-4 — not-addressed — notice.go:94-99 still defaults a nil clock and lifetime<=0; all three call sites still pass real values.
- BR-5 — addressed — console.go:722 now clears noticeExpiry in the fire branch; untestable by construction, as the finding itself stated.
- BR-6 — addressed — README.md:300-303 documents that transients go after ~12s while an exit stands.

### Raised

- **BR-7** [Important] `test-writes-state-the-loop-rebuilds` The package's -race suite fails 3 runs in 10, so the issue's "go test -race passes" Done-when is not a check that passes
  TestConsoleRunOrdinarySwitchAdvancesPrevious (console_run_menu_test.go:390) waits for
  len(Frames) > 0, but the fixture's inventory refresh already built a root frame
  (menu.go:1267), so the wait returns BEFORE ctrl+space is processed. The test's direct
  write of menu.Frames[0].SelectedAddress is then reset by onHotkey reopening the
  switcher on the thread being left; instrumented failures report SelectedAddress=c1.
  Not a pair#185 regression (4 of 6 failures at the pre-185 commit, and the test does
  not exist at the window base), but it falsifies the verification this close records.
  Waiting on f.con.focus.IsPanel() instead makes it 10 of 10 green under -race; verified.
- **BR-8** [Minor] `row-must-track-the-feed` publishNotice owns the paint half of the row-tracks-the-feed rule; the Run loop still owns the deadline half
  This is the 2nd finding in family row-must-track-the-feed. Do NOT fix the instance.
  The rule: one seam owns push, paint, AND expiry re-sync. Today publishNotice
  (console.go:1689) pushes and paints, while syncNoticeExpiry re-arms only at the bottom
  of the Run select (console.go:758) -- so a notice published off the Run goroutine paints
  but arms no timer and never retires. Measured prevalence: 0 live instances (all 6
  setNotice sites and onExit are Run-goroutine; SetInputTrace publishes pre-Run with
  started == false), which is why this is Minor rather than a live defect. Same seam:
  c.started (console.go:517) is set but never cleared at teardown.
- **BR-9** [Minor] `test-discards-evidence-it-awaits` The new console expiry test resets the host buffer after a poll loop that can outlast the 60ms lifetime
  console_notice_expiry_test.go:52 calls host.Reset() after waiting for the sentence to
  appear. On a loaded machine that wait can exceed testLifetime, so the expiry repaint
  lands before the Reset and the second waitFor hangs to its 3s timeout. Reproduced by
  injecting time.Sleep(3 * testLifetime) before the Reset in a scratch copy -> FAIL.
  Assert that the LAST painted row lacks the sentence instead of resetting the buffer.
  Latent, not live: 40 -race runs clean as written.
- **BR-10** [Minor] `discarded-invariant-signal` Feed.Push's ok return is discarded at its only production call site
  console.go:1691 ignores the bool. Enqueue's own doc says a full mailbox should be "a
  loud bug signal rather than flow control" and gave the function a fallible signature
  deliberately; nothing consumes it. Pre-existing behaviour, but publishNotice is the new
  line that drops it.

## Round 3 — 2026-09-04T16:11:18-07:00 (claude) — passed

### Disposed

- BR-3 — not-addressed — Still no shared arm helper; console.go:648 remains the fourth stop/reset/assign closure in Run.
- BR-4 — not-addressed — notice.go:94-99 unchanged; notice.go:100 is the only Feed literal in the tree, so both defaults stay unreachable.
- BR-7 — addressed — Verified by reversion — reverted wait fails 9/20 -race runs here; fixed wait is 100/100, full package 5/5.
- BR-8 — not-addressed — Only the started sub-item was touched, and its guard is TOCTOU-shaped; the class rule (one seam owns push, paint and deadline re-sync) is unchanged.
- BR-9 — addressed — lastPaintedRow needs no window and the rewrite keeps the pin — removing syncNoticeExpiry's arming still turns the test red; 50/50 green under -race.
- BR-10 — withdrawn — Counter-argument accepted — for a rolling feed a capacity drop is the policy, so ok=false carries no signal this caller could act on.

### Raised

- **BR-11** [Minor] `overstated-guard-rationale` Two comments added this round assert guarantees the code does not deliver
  This is the 2nd finding in family overstated-guard-rationale. Do NOT fix the
  instances. The rule: a comment that states an absolute must name the condition
  under which it does not hold, and the guarantee must be pinned by a test that
  fails without it. Enumeration in this diff, prevalence 2 of 2 new absolute
  claims. (a) console.go:781 says "nothing may paint into it again", but
  publishNotice (console.go:1694-1700) reads started under c.mu, releases the
  lock, then paints -- so a publish that passes the check before teardown runs
  still writes into the restored shell. Prevalence 0 live, because every
  publisher is the Run goroutine that also runs teardown; the hole opens exactly
  when BR-8's class opens. (b) console.go:1693 describes publishNotice as the one
  way a notice reaches the operator, but writeOwn (console.go:973) declines while
  hostScan.MidSequence() and that debt is paid only by the next child chunk
  (console.go:1079) -- on the idle console this issue is about, there is none.
  Trigger is pathological and takeOverScreen resets hostScan, so 0 live there
  too. Neither is a behavioural defect today; both are the same habit BR-2 named.

## Open findings

- **BR-3** [Minor] `repeated-timer-arm-shape` syncNoticeExpiry is the fourth near-identical arm-a-timer closure in Run
- **BR-4** [Minor] `unreachable-constructor-default` NewFeed's nil-clock and lifetime<=0 defaults are unreachable and untested
- **BR-8** [Minor] `row-must-track-the-feed` publishNotice owns the paint half of the row-tracks-the-feed rule; the Run loop still owns the deadline half
- **BR-11** [Minor] `overstated-guard-rationale` Two comments added this round assert guarantees the code does not deliver
