# Boundary Review — pair#489 (whole-issue close)

| field | value |
|-------|-------|
| issue | 489 — couch: relaunch/resume fails under load (pane not born in 15 s) and leaves the thread parked; make startup resilient |
| repo | pair |
| issue file | workshop/issues/000489-couch-relaunch-resume-fails-under-load-pane-not-born-in-15-s-and-leaves-the-thread-parked-make-startup-resilient.md |
| boundary | whole-issue close |
| milestone | — |
| window | 618ba11e4ace65761025cb35aa3119ab5ea40226..67c844bfc6cb6945fd039cdb94c3985635fa2aab |
| command | sdlc close --issue 489 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T11:40:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I'm checking how the helper process group behaves, so I can tell whether the new exit watch could end a healthy launch too early.

I'm confirming how long the title poller runs, since it shares the helper's process group.

Almost done: I'm checking whether the post-exit "final look" has side effects, then I'll write up the findings.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The diff fixes the root cause it measured. The fixed 15 s registration window is now a 2-minute hang ceiling. The wait also ends early when the helper's process group exits, and a final look afterwards still adopts a registration that was already written. Couch-launched Pair no longer runs `dev_rebuild` inside that window. Relaunch's freshness check now also applies under PAIR_DEV. The CLI's status poll asks again after a timed-out read instead of reporting the result as uncertain. Every failure still goes through the same post-acknowledge cleanup.

What I ran: the full `couchcore` suite passes with `go test -race` (444 s). The targeted `couchcmd` and `launcher` tests also pass. To run those two I had to add placeholder runtime assets, because they are generated files missing from this checkout; I removed them afterwards.

One thing should be fixed before shipping. The README and atlas promise that a dead launch "fails at once". In production, the Couch-launched Pair puts its title-poller sidecar in the helper's process group, and that sidecar waits up to 30 s for the pane to appear. So a launch that dies after the poller starts holds the wait open for about 30 s. For the #288 dead-at-birth cold resume, that is about twice as long as the old 15 s. The fake runner doesn't model this, so the new test can't catch it.

1. **Strengths**
   - **Exit watch and deadline order** (`launch_existing.go:223-227`): the exit watch sits under the deadline, so an injected deadline still cancels synchronously. The watch's goroutine is joined by `stop` and leaves nothing running. This order was found the hard way and is recorded in the Log.
   - **Final look after exit** (`launch_existing.go:241-250`): an exit that ends the wait leads to one bounded, read-only look. A registration already written still counts, so the existing test `TestResumeRunsAsTheNewLiveOwner` keeps passing.
   - **One failure path:** every failure still goes through `failTrackedPostAckStart`, which kills the helper's process group and puts the thread back to parked. `TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering` checks both the error message and that the thread is left parked.
   - **Dead flag removed:** `BinaryFacts.DevRebuild` is gone, along with its consumer, its case in `DecideBinaryFreshness` and its test row. The probe test now checks that PAIR_DEV changes nothing.
   - **Status-poll retry is narrow:** only timeouts are retried (the read is idempotent). A dial error, an unavailable caller or an unknown receipt still ends the CLI as uncertain at once. There is a test for each side.

2. **Critical findings**
   - None.

3. **Important findings**
   - **I1 – "fails at once" is not what production does (ARCH-MOCK, ARCH-CONSTRAINTS).**
     - **Cause:** `whileHelperAlive` tests the whole process group (`ownedProcessGroupAlive`). But a Couch-launched Pair deliberately keeps its sidecars in that group (`launcher/osruntime.go:458-462`). `SpawnTitlePoller` runs at `createflow.go:859`, before `launchWatched`, and the poller waits up to `StartupGrace` (30 s, `titlepoller/run.go:77-78,127`) for the pane.
     - **Effect:** if Pair dies after that point (e.g. the birth watch kills a hung client at about 10 s and Pair exits 1), the group stays alive until about 30 s. That is slower than the old 15 s.
     - **Why the test misses it:** in the fake, `SetExited` makes the whole group dead.
     - **Fix sketch:**
       - End the watch on the *direct child's* exit. `execHandle.done` already exists, and `ptyHandle` has `child.Wait`. Expose that as something like `Exited() <-chan struct{}` on `Handle` and keep group liveness for cleanup only.
       - Make the fake able to model a group member that outlives the child, and add a test where the child exits but a sidecar is still alive.
   - **I2 – stale atlas text (docs gate, ARCH-PURPOSE shadow sweep).** `atlas/couch.md:2101` still says "Cold resume. Its wait still runs to its 15 s deadline even after the helper has exited at about 10 s". That is no longer true, and I1 is what actually happens now. Rewrite it once I1 is settled.

4. **Minor findings**
   - `TestRegistrationCeilingClearsAStartSlowedByLoad` checks a constant, not behaviour; the late-start test is the real evidence.
   - `TestColdResumeAdoptsALateStart` depends on wall-clock time (`time.Since(since) > 5*helperExitPoll`), so it only ever samples one interleaving (ARCH-ORDER). Passing in a clock would make it deterministic.
   - After timed-out polls, the progress line reports the last status it actually received, without saying that recent polls timed out. "still running (Couch slow to answer)" would make that clear.
   - Spec item 4 ("build once per sweep") is delivered as a manual `make build`, enforced by the freshness refusal. That is acceptable and documented in the README and atlas, but it is the operator's job, not automated.

5. **Test coverage notes**
   - Covered:
     - a late start is adopted
     - an exit gives a fast, named failure and leaves the thread parked
     - DevRebuild is skipped only when Couch-hosted
     - PAIR_DEV doesn't change the binary facts
     - a status-poll timeout is retried, and persistent timeouts end uncertain at the budget
     - the progress line appears twice over about 40 s
   - Not covered: a group member outliving the direct child (I1).
   - I did not revert each fix to watch its test fail. Reading the assertions, each new test would fail without its fix (for example, `devRebuilt` would be true without the `CouchHosted` guard).

6. **Architectural notes for upcoming work**
   - **ARCH-DRY: pass.** `await` is a single closure used by both the main wait and the final look.
   - **ARCH-PURE: pass.** The watch is a thin IO layer, and the freshness decision is still a pure function.
   - **ARCH-PURPOSE: flag (I2).** Spec items 1–4 are delivered. Item 5 (pacing) was declined with a reason: each CLI call already waits for its launch.
   - **ARCH-MOCK: flag (I1).** The fake handle can't represent a child that has exited while its group is still alive.
   - **ARCH-CONSTRAINTS: flag (I1).** The 2-minute ceiling is justified against measured startup times, and the bounded-work caveat is kept. The actual bound on a dead launch is about 30 s, not "at once".
   - **ARCH-SECURE: pass.** There is no new untrusted input and no credentials.
   - **ARCH-ORDER: pass with the note above.** The watch's lifetime is joined, cancellation causes are kept separate, and the exit-then-look order is written down. The only gap is the wall-clock test.
   - **ARCH-FUNERAL: pass.** Nothing durable is created, and the goroutine is joined by `stop`.

7. **Plan revision recommendations**
   - "2026-10-12: The exit watch must key on the helper's direct child, not its process group. Couch-launched sidecars (the title poller) share the group and keep it alive up to `StartupGrace` (30 s). Until then, 'fails at once' means 'fails when the last group member exits'."

```findings
findings:
  - id: new
    severity: Important
    family: fake-diverges-from-real-seam
    title: |
      Exit watch tests process-group liveness, but the Couch-launched title poller shares the group for up to 30 s, so a dead launch does not "fail at once"
    detail: |
      whileHelperAlive (launch_existing.go:461) uses ownedProcessGroupAlive. Couch-hosted sidecars stay in the helper's group (osruntime.go:458-462); SpawnTitlePoller runs before launchWatched (createflow.go:859) and waits StartupGrace=30s (titlepoller/run.go:77,127). A Pair that dies at birth (the #288 case, about 10 s) therefore holds the wait until about 30 s, longer than the old 15 s. The fake's SetExited kills the whole group, so TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering can't see it. Fix: watch the direct child's exit (execHandle.done or ptyHandle child wait, exposed on Handle) and keep group liveness for cleanup; teach the fake a surviving group member and add that test.
  - id: new
    severity: Important
    family: docs-restate-replaced-behavior
    title: |
      atlas/couch.md:2101 still says a cold resume runs to its 15 s deadline after the helper exits
    detail: |
      The registration wait is now a 2-minute ceiling ended by the helper exiting. Rewrite the pair#288 cold-resume bullet to describe what really happens (it depends on I1).
  - id: new
    severity: Minor
    family: test-pins-constant-not-behavior
    title: |
      TestRegistrationCeilingClearsAStartSlowedByLoad only checks that a constant is at least 1 minute
  - id: new
    severity: Minor
    family: wall-clock-single-interleaving-test
    title: |
      TestColdResumeAdoptsALateStart decides timing with time.Since on the real clock, so it only ever runs one interleaving
    detail: |
      Pass a controllable clock into the sidecar hook so the late start becomes a reproducible ordering (ARCH-ORDER).
  - id: new
    severity: Minor
    family: progress-reports-stale-status
    title: |
      After timed-out status polls, the "still <status>" line reports the last status received without saying the polls timed out
```

---

## Re-review — 2026-10-12T13:18:26-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 489 — couch: relaunch/resume fails under load (pane not born in 15 s) and leaves the thread parked; make startup resilient |
| repo | pair |
| issue file | workshop/issues/000489-couch-relaunch-resume-fails-under-load-pane-not-born-in-15-s-and-leaves-the-thread-parked-make-startup-resilient.md |
| boundary | whole-issue close |
| milestone | — |
| window | 373e763635f8f50a9a6e842efccb8a93cc0ca1de..74a69f606eeff07c76b9ea2b0cd9e13751a6b1dd |
| command | sdlc close --issue 489 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T13:18:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've worked through the diff and am now checking what process Couch actually launches directly, since the BR-1 fix depends on it.

Tests pass. Now confirming the BR-1 regression test fails when the fix is reverted to group-liveness polling.

Checking whether any test covers the "one final look after exit" path by disabling it and running the couchcore suite.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All five prior findings are fixed, and I checked each against the code and tests. The main one, BR-1, now waits on Pair's own process exiting rather than on its whole process group. The `Handle.Exited()` change touches every runner: exec, pty, fake, the acknowledged-handle wrapper and stubs. To prove the test guards it, I put back the old process-group check in a scratch copy; `TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering` then hung until the 30 s test timeout. With the fix in place, couchcore, couchcmd, launcher and couchtty all pass. These packages only build after `make runtimebundle-generate`, because the bundled runtime files are generated, not committed.

Three things stand between this and SHIP, none blocking:
- **Stale descriptions of the exit signal.** The BR-1 fix moved the signal from process group to direct child, but four places still describe the old one.
- **Untested final look.** The "one last registration check after exit" (commit e8d31bfe3) changes behaviour, but turning it off breaks no test.
- **Leftover stale-status case.** The BR-5 fix only covers part of the progress line's misleading status, and the "gave up" message has the same problem.

### 1. Strengths
- `whileHelperAlive` (`launch_existing.go:458-479`) is a small, self-contained watcher. Its stop function waits for the watch goroutine to finish (`<-done`), and its result comes from the cancellation cause rather than a separate flag, so nothing outlives the wait.
- The fake keeps the process and its group as two separate states: `endChild` / `SetChildExited` (`runner_fake.go:606-622`). Every existing exit path also ends the child, so earlier tests keep their meaning.
- A status poll that times out is retried until the 5-minute poll budget runs out (`messages.go:287-292`). That is correct because reading status has no side effects. Two tests pin it: retry-then-succeed, and keep-timing-out-until-uncertain.
- The pacing decision is sound. There is no load-average check; each CLI call waits for its own launch, so a sequential sweep paces itself by real launch time. The `make build` that ran on every Couch launch is gone, behind `env.CouchHosted()`, and the freshness gate now applies under `PAIR_DEV` too.
- I confirmed the helper `syscall.Exec`s into Pair (`launchhelper.go:109`). So the "direct child" really is Pair, and `Exited()` watches the right process.

### 2. Critical findings
None.

### 3. Important findings
- **I-1: Four places still say the wait ends when Pair's "process group" exits** (family `docs-restate-replaced-behavior`, ARCH-DRY / ARCH-PURPOSE).
  - The sites: `couch.go:396-397` ("caught by its process group exiting"), `birthwatch.go:24-25` ("ends early when Pair's process group exits"), `atlas/architecture.md:351` ("ends early only when the launch's process group exits") and the issue's own Plan line 47.
  - This is the opposite of the new `lessons.md` entry and of `atlas/couch.md:2046-2059`.
  - **This is the 2nd finding in family `docs-restate-replaced-behavior`.** Don't just patch these four.
  - The rule: describe the exit signal once, in the `whileHelperAlive` doc comment and the atlas/couch.md "Registration waits on Pair" section. Everywhere else should name it ("Couch's registration wait; see `whileHelperAlive`") and not repeat the mechanism.
  - The sweep for this round: `rg -n "process group exit|group exits|15 s registration|15s registration"` across `cmd/`, `atlas/`, `README.md` and the issue file. Then add a Plan `## Revisions` entry.
- **I-2: Nothing tests the final registration check after an exit** (new family `behavior-fix-without-red-test`, ARCH-ORDER).
  - With `if false && await(final) == nil` at `launch_existing.go:286`, the full couchcore suite still passes (212 s).
  - The ordering it guards — Pair writes its registration, then exits before the wait first looks — is real, but no test sets it up.
  - Fix: add a fake-driven test. In `AfterAcknowledge`, write the registration (`SetPaneSidecar` + `SetPairSession`) and call `SetChildExited`. Use `PaneSidecarsHook` to hold the first wait poll until `h.Exited()` has closed, so the exit always wins the race. Assert that the resume succeeds and the thread is occupied.

### 4. Minor findings
- **M-1: The "(last answer before N timed-out status polls)" note can be wrong after a fresh answer** (family `progress-reports-stale-status`).
  - `timeouts` resets only when a progress line prints (`messages.go:279`), not when a poll succeeds (`messages.go:297`).
  - Example: two polls time out at 1 s, then 28 fresh "running" answers arrive. The 15 s line still says the status predates the timeouts.
  - The same class: when the budget runs out at `messages.go:270`, the message says "still running after 5m" even if every poll in the last minutes timed out.
  - **This is the 2nd finding in family `progress-reports-stale-status`.** The rule: every status line derives its staleness note from the time of the last answer received, not from a counter of events since the last print. Track `lastAnswer := clock.now()` on success. Have the progress line and the "gave up" message both say "(last answer Ns ago)" when that time is older than one poll interval.
  - Test: timeouts, then success, then a progress line — the note must be absent.
- **M-2: No live check that the real runners behave like the fake when only the child exits** (family `fake-diverges-from-real-seam`, ARCH-MOCK).
  - The fake now models "child exited, group still alive". No live test shows the real `execHandle` / `ptyHandle` `Exited()` closing while a group member survives. `conformance_live_test.go` doesn't call `Exited()`.
  - **This is the 2nd finding in family `fake-diverges-from-real-seam`.** The rule: every state transition added to `FakeRunner` gets a matching case in the live runner conformance test, in the same change.
  - Here: start `sh -c 'sleep 5 & exit 0'` and assert that `Exited()` closes while `Alive()` stays true.
- **M-3: `stubHandle.Exited()` returns nil** (`console_relaunch_chord_test.go:655`). Waiting on that channel blocks forever. It's harmless today because only `ID()` is used, but a closed or never-closing channel created on purpose would make the intent clear.

### 5. Test coverage notes
- The BR-1 regression test fails when the fix is reverted (it hung until the 30 s timeout), and passes at the pinned head.
- `TestColdResumeAdoptsALateStart` now drives the late start by poll count, not the wall clock. It checks the deadline the real launch path set up (`couch.go:200`), so it tests behaviour rather than a constant.
- The fake's 10 ms ticker means "past the old window" is shown by the ≥ 1 min deadline, not by elapsed time. That's acceptable, but the Plan's "late registration past the old window (injected delay)" wording overstates it.
- Gap: I-2, the race between the final check and the exit.

### 6. Architectural notes
- **ARCH-DRY:** flagged (I-1).
- **ARCH-PURE:** pass. The decision logic is small and lives in the wait; the IO is behind `Handle`.
- **ARCH-PURPOSE:** pass. The Spec items (cause measured, hang ceiling, late start adopted, dead launch fails fast, one build per sweep, progress line) are all delivered. The synthetic-load substitution is recorded in Revisions.
- **ARCH-MOCK:** flagged (M-2).
- **ARCH-CONSTRAINTS:** pass.
  - The 2-minute ceiling sits inside the CLI's 5-minute poll budget.
  - The final check is capped at 1 s.
  - Retries cost at most one 2 s timeout plus a 0.5 s sleep, all inside the budget.
- **ARCH-SECURE:** N/A. Nothing new is read from outside the process, and no credentials are involved.
- **ARCH-ORDER:** flagged (I-2).
  - The exit-versus-registration race is handled, but no test controls the ordering.
  - The watcher's lifetime is bounded: stop joins the goroutine.
- **ARCH-FUNERAL:** pass. The only new resource is the watcher goroutine, and stop joins it. Nothing durable is created.

### 7. Plan revision recommendations
- Add a `## Revisions` entry: "close review BR-1: the registration wait ends on the helper's DIRECT child exiting (`Handle.Exited`), not on its process group. A Couch-launched Pair's title poller stays in the group for up to 30 s. Plan line 47 is superseded."
- Add a `## Revisions` entry: "the late-start test proves adoption by poll count plus a ≥ 1 min deadline, not by elapsed time past 15 s."

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      whileHelperAlive keys on Handle.Exited (launch_existing.go:461); I swapped back a group-Alive poll in a scratch copy and TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering hung to the 30 s timeout (red), and it passes at head.
  - id: BR-2
    disposition: addressed
    note: |
      atlas/couch.md:2111 now says the wait ends when the helper's direct child exits; stale group wording elsewhere is raised as a new family instance.
  - id: BR-3
    disposition: addressed
    note: |
      The constant-only test is gone; TestColdResumeAdoptsALateStart checks the deadline the real launch path arms (default from couch.go:200).
  - id: BR-4
    disposition: addressed
    note: |
      The late start is now driven by poll count through PaneSidecarsHook (50th look), not time.Since on the real clock.
  - id: BR-5
    disposition: addressed
    note: |
      The progress line names timed-out polls (messages.go:276) and TestSlotOperationCLIProgressNamesTimedOutPolls pins it; a leftover case is raised separately.
findings:
  - id: new
    severity: Important
    family: docs-restate-replaced-behavior
    title: |
      couch.go:396, birthwatch.go:25, atlas/architecture.md:351 and Plan line 47 still say the wait ends on the process group exiting
    detail: |
      2nd instance of this family. Rule: describe the exit signal once (whileHelperAlive doc + atlas/couch.md pair-489 section) and make every other site name it rather than restate it. Sweep with rg for process-group-exit and old 15 s registration wording across cmd/, atlas/, README.md and the issue file in the same round.
  - id: new
    severity: Important
    family: behavior-fix-without-red-test
    title: |
      The final registration check after the helper exits has no test that fails without it
    detail: |
      With the check disabled (if false && await(final)) the full couchcore suite still passes. Add a fake test: AfterAcknowledge writes the registration and calls SetChildExited, and PaneSidecarsHook holds the first poll until Exited() has closed; assert the resume succeeds (ARCH-ORDER).
  - id: new
    severity: Minor
    family: progress-reports-stale-status
    title: |
      The timed-out-polls note persists after fresh answers, and the budget-exhausted message hides timeouts
    detail: |
      2nd instance. timeouts resets only when a line prints (messages.go:279), not on a successful poll (messages.go:297). Rule: every status line derives staleness from the time of the last answer received (lastAnswer), not from a counter of events since the last print; apply it to both the progress line and the budget-exhausted message.
  - id: new
    severity: Minor
    family: fake-diverges-from-real-seam
    title: |
      No live conformance check that the real Exited() closes while a group member survives, as the fake now models
    detail: |
      2nd instance. Rule: every state transition added to FakeRunner gets a matching live runner-conformance case in the same change; here, sh -c 'sleep 5 & exit 0' must give Exited closed while Alive stays true.
  - id: new
    severity: Minor
    family: test-double-unsafe-default
    title: |
      stubHandle.Exited returns a nil channel, which blocks forever if ever awaited
```

---

## Re-review — 2026-10-12T14:29:41-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 489 — couch: relaunch/resume fails under load (pane not born in 15 s) and leaves the thread parked; make startup resilient |
| repo | pair |
| issue file | workshop/issues/000489-couch-relaunch-resume-fails-under-load-pane-not-born-in-15-s-and-leaves-the-thread-parked-make-startup-resilient.md |
| boundary | whole-issue close |
| milestone | — |
| window | 397bb75290cba221c4337242c349c7b1e28e6a34..646773eb662bd25738bbc0f143cce4ef02d3aa18 |
| command | sdlc close --issue 489 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T14:29:41-07:00 |
| verdict | SHIP |

## Review

Sweep and inspection done; now running the touched packages' tests (vet hit an unrelated missing embed asset).

All touched packages pass. Now reverting the final-look branch to confirm BR-7's test goes red.

```verdict
verdict: SHIP
confidence: high
```

All five open findings (BR-6 to BR-10) are fixed, and I checked each one against the code and tests rather than the Log. I ran the touched packages (couchcore, couchcmd, launcher, couchtty) after `make runtimebundle-generate`, and they all pass. The checkout lacks the generated embed assets, so tests fail to build until that step runs. That is an environment issue, not part of this diff. For BR-7, I turned off the final look (`if false && await(final) == nil`). `TestColdResumeRegistrationWrittenBeforeTheExitStillCounts` then went red with "Pair exited before registering", and went green again once I restored the line. The live check `TestRealRunnersExitedPrecedesTheGroup` runs on every test run, with no gate, and passes on both the exec and pty runners. I found only two new Minor issues, so nothing blocks shipping.

Done-when: current — the Revisions entry dated 10:35 explicitly swaps the "synthetic load ~200" item for an injected-delay test plus the 10:27 evidence. Both remaining items are covered: the late start is adopted (`TestColdResumeAdoptsALateStart`), and a dead launch fails at once and rolls back cleanly (`TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering`).

**1. Strengths**
- `whileHelperAlive` (`cmd/internal/couchcore/launch_existing.go:442`) watches the direct child's exit, not process-group liveness. Its goroutine is limited to the call and joined by `stop`, and it returns whether the exit is what ended the wait.
- The ordering seam in `registration_liveness_test.go:672-693` captures the watch context through `registrationDeadline`. The pane stays unobservable until the exit has cancelled the wait, so the test reaches the final-look branch every time instead of depending on timing.
- `exchangeSlotOperation` (`cmd/internal/couchcmd/messages.go:265-202`) now works out staleness from `lastAnswer`, which both the progress line and the budget message use. The test checks that the note appears during timeouts and is gone from the next line once Couch answers.
- The fake's `SetChildExited` and `endChild` sit next to a live check (`ptyrunner_test.go:526`) that confirms real runners behave the same way: `Exited` closes while `Alive` stays true.
- The dev-rebuild skip goes through `env.CouchHosted()` in one place (`createflow.go:860`). The freshness gate's PAIR_DEV exemption was removed along with its field, its test row and its doc.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- **Wrong cause in the dead-launch error** (`launch_existing.go:404-424`). The error from the first `await` is kept, so it reads "context canceled [ownership poll consumed the budget …] (Pair exited before registering …)". Nothing consumed a budget here. Fix: when `exited`, wrap `context.Cause` (the exit error) and leave out the phase bracket.
- **Ambiguous sentence in `atlas/couch.md:2069-2072`.** "the wait ends at registration, when the helper's DIRECT child exits" can be read as saying registration happens when the child exits. Splitting it into a three-item list (registration / direct child exits / 2-min ceiling) fixes it.
- **Nit:** the final look also runs when `err == nil && exited`. It's harmless, but `if exited && err != nil` says what's meant.

**5. Test coverage notes:** The adopt-late, exit-fails-fast, registered-then-exited, CLI-progress, staleness, retry-on-timeout and rebuild-skip paths each have a test. Only BR-7's final-look test was reverted by hand this round, and it is a proper regression test.

**6. Architecture**
- **ARCH-DRY:** pass. The closed-channel `Exited` pattern appears twice in test doubles, which is too trivial to share.
- **ARCH-PURE:** pass. `DecideBinaryFreshness` stays pure, and the watch is a thin wrapper around IO.
- **ARCH-PURPOSE:** pass. Spec items 1–4 are delivered, and item 5 (pacing) was declined with a stated reason.
- **ARCH-MOCK:** pass. The fake's new transition has a live check that runs on every test run.
- **ARCH-CONSTRAINTS:** pass. The 2-min ceiling fits inside the CLI's 5-min poll budget, with a progress line every 15 s.
- **ARCH-SECURE:** not applicable. The change reads no new untrusted input and touches no secrets.
- **ARCH-ORDER:** pass. Exit vs. registration ordering is tested through the seam, and the watcher's lifetime ends when `stop` joins it.
- **ARCH-FUNERAL:** pass. Nothing durable is created; the only new thing is a goroutine, and `stop` joins it.

**7. Plan revisions:** none needed. The Plan and Revisions match the code.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      couch.go:396, birthwatch.go:25 and architecture.md:351 now point at whileHelperAlive or the atlas section; the Plan line and Revisions now say direct child. An rg sweep found no remaining process-group-exit or 15 s registration wording.
  - id: BR-7
    disposition: addressed
    note: |
      With the final look disabled (if false && ...), TestColdResumeRegistrationWrittenBeforeTheExitStillCounts FAILS ("Pair exited before registering"); it passes once restored.
  - id: BR-8
    disposition: addressed
    note: |
      stale() derives from lastAnswer, which only a successful poll updates; it covers the progress and budget messages, and TestSlotOperationCLIProgressNamesTimedOutPolls asserts the note clears.
  - id: BR-9
    disposition: addressed
    note: |
      TestRealRunnersExitedPrecedesTheGroup runs ungated on the exec and pty runners and passes.
  - id: BR-10
    disposition: addressed
    note: |
      The couchtty stubHandle.Exited returns a closed channel.
findings:
  - id: new
    severity: Minor
    family: failure-text-names-wrong-cause
    title: |
      Dead-launch error still says "ownership poll consumed the budget" and "context canceled"
    detail: |
      launch_existing.go:404-424 keeps the first await's registrationWaitError, so the text blames a budget the exit ended. When exited, wrap context.Cause (the exit error) and leave out the phase bracket.
  - id: new
    severity: Minor
    family: doc-ambiguous-wording
    title: |
      atlas/couch.md pair#489 sentence reads as if registration happens when the child exits
    detail: |
      "the wait ends at registration, when the helper's DIRECT child exits ..., or at ..." should list three separate end conditions.
```
