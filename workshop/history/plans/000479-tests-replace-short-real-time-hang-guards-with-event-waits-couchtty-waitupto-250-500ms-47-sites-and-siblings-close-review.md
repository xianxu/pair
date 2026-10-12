# Boundary Review — pair#479 (whole-issue close)

| field | value |
|-------|-------|
| issue | 479 — tests: replace short real-time hang guards with event waits (couchtty waitUpTo 250/500ms, ~47 sites, and siblings) |
| repo | pair |
| issue file | workshop/issues/000479-tests-replace-short-real-time-hang-guards-with-event-waits-couchtty-waitupto-250-500ms-47-sites-and-siblings.md |
| boundary | whole-issue close |
| milestone | — |
| window | 97478ad961f162c25c10eb2f1018f122c5c77441..2f1a6b57d1c150102735b890fa06f216aa505476 |
| command | sdlc close --issue 479 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T08:07:54-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing: the listed sites are converted, and I'm now checking for sibling guards that were left behind and testing the helper.

I'm writing up the verdict now. The targeted tests pass (launcher handoff ×5 under `-race`, couchsingleton ×3, testwait). couchtty and wrapcmd would not build in this checkout because a generated embed asset is missing, so I could not run them here.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The issue asked for event waits instead of short real-time windows, and the listed sites get them. The new `cmd/internal/testwait` helper (`After`, `Wait`, `Until`) is small and pure at its core: `giveUpAt` is tested without a real clock. couchtty's three pollers become one `waitFor`, and no short positive guard is left in couchtty. The ceiling sites now check the outcome instead of a stopwatch: `ErrThreadStoreBusy`, "lock timed out", "timed out", `context.DeadlineExceeded`, and the signal that ended the child. The launcher change also fixed a real test race: the cancel could land before the TERM trap was set.

Two things are left. The touched files still hold the same kind of 1–3 s fatal `time.After` guards next to the converted lines. The Log says both notifytransport ceilings now check an outcome, but one only drops its stopwatch. Neither blocks: the TL limited Done-when to the listed sites. Converting the leftover guards is a mechanical change.

I tried to verify everything else at full depth but couldn't build couchtty or wrapcmd here, so I confirmed those by reading the diff, not by running them.

**1. Strengths**
- `cmd/internal/testwait/testwait.go:55-65`: the margin calculation is a pure function, unit-tested with fixed times, including the case where the deadline has already passed.
- `cmd/internal/launcher/osruntime_launch_test.go:38-60`: the test now checks *how* the child ended (SIGTERM vs SIGKILL) and only cancels once the child has set its trap. This exposed a race that the old 1.5 s ceiling hid. Ran 5× under `-race`: passes.
- `cmd/internal/couchtty/console_test.go:20-26`: `waitFor`, `waitLong` and `waitUpTo` are now one wrapper over `testwait.Until`. `grep waitUpTo|waitLong` finds nothing left.
- `console_slotgit_test.go:166-181`: `settle` now reads the schedule's idle state on the loop that owns it, instead of waiting for a 20 ms quiet spell.
- The atlas entry and the lessons entry both describe the new surface and the "check the outcome, not the elapsed time" rule.

**2. Critical:** none.

**3. Important**
- **ARCH-PURPOSE, siblings left in touched files.** These files were edited in this window but still have positive guards of 1–3 s that fail the test (`t.Fatal`). They are the same class the issue title calls "siblings":
  - `wrapcmd/terminal_model_test.go`: lines 424, 778, 789, 854, 891, 943, 952, 1021
  - `terminalqualify/candidate_test.go`: 103, 108, 154
  - `wrapcmd/peer_input_test.go`: 190, 348
  - `notifytransport/transport_test.go`: 54, 136, 149, 437
  - `wrapcmd/orientation_replies_test.go`: 30, 129, 137, 143, 152
  - Fix: replace each with `case <-testwait.After(t):`.
  - Many more exist in files this window did not touch (broadcast, webcast, termcmd, couchmessage, couchcmd …). Those should go to a follow-up issue, not this one.

**4. Minor**
- `notifytransport/transport_test.go:319-340` (`TestSendToUnavailableReaderIsBounded`): the stopwatch was removed but no outcome check replaced it. The test only checks `sendErr != nil`, yet the Log claims "notifytransport ×2" check outcomes. Fix: check the write-deadline error where the platform produces it, or correct the Log entry.
- `testwait_test.go`: the give-up path (`Wait` returning false, `Until` failing the test) is not tested. Neither is the no-deadline case where `After` returns nil. Only the arithmetic and the success path are covered.
- Thin wrappers remain: wrapcmd `waitFor` → `testwait.Wait`, `waitForStdoutBatch` → `testwait.Until`. Keeping them limits churn, but callers could use `testwait` directly.
- Design cost to note: one lost event now uses up almost the whole binary timeout. Every test after it then gets "half the remaining time" (about 5 s), so one real failure can look like a cascade of failures. This was agreed with the TL; it may be worth a line in the atlas entry.

**5. Test coverage**
- The probe that adds a temporary product sleep is the right evidence that each site was a window. It is recorded in the Log; I did not re-run it.
- couchtty and wrapcmd could not be built here (the `runtimebundle` embed asset is missing), so those converted tests were not run in this review.
- The out-of-list soak failures are assigned to #484 and #485.

**6. Architecture**
- ARCH-DRY: pass. The three couchtty pollers, the wrapcmd pollers and the supervisorlease loop all go through one helper now. The thin wrappers are noted as Minor.
- ARCH-PURE: pass. `giveUpAt` is pure, and reading the clock and `t.Deadline` stays in a thin outer layer.
- ARCH-PURPOSE: flagged. The siblings in the touched files listed under Important.
- A follow-up issue should sweep the rest of the tree with `testwait.After`.

**7. Plan revisions**
- Add a Revisions entry: either the notifytransport `Send` ceiling now checks the deadline error, or the "notifytransport ×2" claim becomes ×1, with `Send` relying on the binary timeout.
- Record the in-file sibling guards: either converted in this round, or explicitly deferred to a new issue.

```findings
findings:
  - id: new
    severity: Important
    family: real-time-window-as-event-guard
    title: |
      Touched files keep fatal 1-3 s time.After guards of the same class as the converted sites
    detail: |
      wrapcmd/terminal_model_test.go:424,778,789,854,891,943,952,1021; terminalqualify/candidate_test.go:103,108,154; wrapcmd/peer_input_test.go:190,348; notifytransport/transport_test.go:54,136,149,437; wrapcmd/orientation_replies_test.go:30,129,137,143,152. Convert each to testwait.After(t); file a follow-up issue for the untouched packages (broadcast, webcast, termcmd, couchmessage, couchcmd, couchcore, ...).
  - id: new
    severity: Minor
    family: ceiling-replaced-by-outcome
    title: |
      TestSendToUnavailableReaderIsBounded dropped its stopwatch without asserting the bounding outcome
    detail: |
      notifytransport/transport_test.go:319-340 only checks sendErr != nil; the Log claims notifytransport x2 assert outcomes. Either assert the write-deadline error (where the platform produces it) or correct the Log/Revisions entry.
  - id: new
    severity: Minor
    family: helper-failure-path-untested
    title: |
      testwait unit tests cover only the success path and the margin arithmetic
    detail: |
      Neither the Wait give-up (false) path nor the After nil-channel case with no deadline is exercised; injecting the deadline into Wait would make the give-up path testable.
```

---

## Re-review — 2026-10-12T08:49:39-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 479 — tests: replace short real-time hang guards with event waits (couchtty waitUpTo 250/500ms, ~47 sites, and siblings) |
| repo | pair |
| issue file | workshop/issues/000479-tests-replace-short-real-time-hang-guards-with-event-waits-couchtty-waitupto-250-500ms-47-sites-and-siblings.md |
| boundary | whole-issue close |
| milestone | — |
| window | 97478ad961f162c25c10eb2f1018f122c5c77441..b05acbdac9fa742ac1aaf11e6627d3e8cf2959ec |
| command | sdlc close --issue 479 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T08:49:39-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still checking the remaining wait guards that the round-1 fix may have missed.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** I ran the stat and name-status recipes, then read the targeted diffs, the `testwait` package and its tests, and the issue file. I ran `go test` on `testwait` and `notifytransport` and both passed. The design holds up: one shared helper bounds every wait by the test's deadline minus a margin, both `waitFor` wrappers now delegate to it, and the stopwatch "ceilings" now check the outcome that proves the bound instead of elapsed time. The round-1 Minors (BR-2, BR-3) are fixed and have tests. BR-1 is only partly fixed. Its sweep converted guards whose `t.Fatal`/`t.Error` comes right after the `case`. It missed three kinds: guards that run cleanup before failing, guards built as `deadline := time.After(N)` + `case <-deadline:`, and at least one plain single-line guard. So the Log's claim that "every positive select guard … in all touched packages now uses `testwait.After(t)`" is wrong. It's cheap to fix, so this is FIX-THEN-SHIP.

1. **Strengths**
   - `cmd/internal/testwait/testwait.go`: the API is small (`After`/`Wait`/`Until`). The time math lives in pure functions (`giveUpAt`, `waitUntil`, `afterAt`), which can be tested without a real clock.
   - `couchtty/console_test.go:23` and `wrapcmd/translate_stdin_test.go:48`: the old `waitFor`/`waitLong`/`waitUpTo` variants now share one source of truth (ARCH-DRY pass).
   - `notifytransport/transport_test.go:345-350,407`: the stopwatch checks became outcome checks (ENOBUFS or the write deadline; "lock timed out"). Same approach in launcher, couchsingleton and launchhelper.
   - The soak found three races that the old stopwatches had been hiding (SIGTERM trap timing, the readiness deadline on ready frames, paste ordering). Each is fixed by ordering on an event, not by a bigger number.

2. **Critical:** none.

3. **Important**
   - BR-1 is still open (disposed as not-addressed below). These positive guards remain in touched packages. Fatal, event-awaiting, multi-line body or `deadline :=` form:
     - `wrapcmd/agent_restart_test.go:109` (3 s)
     - `wrapcmd/shortcut_passthrough_test.go:208` (1 s, in the same file as converted sites)
     - `couchcmd/capture_test.go:235` (30 s, same file)
     - `couchtty/terminal_pressure_test.go:252` (5 s teardown join, `t.Error`)
     - `couchtty/console_messages_test.go:85/92` (3 s)
     - `couchtty/console_pointer_test.go:279/283` (10 s)
     - `couchcore/park_test.go:887` (5 s, documented "liveness backstop", ends in `t.Fatalf`; the issue's own rule says to use `testwait.After`)
     - `couchcore/park_lifecycle_live_test.go:193` (5 s, returns an error; this is a live test, so convert it or document why it's exempt)
     - `couchtty/menu_perf_test.go:266` (1 s): a perf test whose latency may be the subject. Either exempt it explicitly or convert it.

     **The rule:** a select arm on a real-time timer whose body (any statement, not only the next line) ends in `t.Fatal*`/`t.Error*`/an error return, and whose sibling arm receives the awaited event, becomes `testwait.After(t)`. The same goes for a `deadline := time.After(N)` hoisted out of a loop. Arms with an empty body (absence windows) stay. Re-run the sweep with a body-aware scan, not a next-line grep.

4. **Minor**
   - The issue Log entry for close round 1 overstates the sweep (as described above). Correct it once the residual sites are converted.

5. **Test coverage notes**
   - BR-3 is covered by `testwait_test.go:41-60`. If the `bounded && !Before(at)` check were reverted, the give-up test would hang until the binary timeout, so it does fail without the fix.
   - BR-2's assertion passes on darwin. The `os.ErrDeadlineExceeded` branch isn't exercised on this platform, which is acceptable.

6. **Architectural notes**
   - ARCH-DRY: pass. Both package-local `waitFor`s are thin wrappers over `testwait`, and I found no duplicated deadline logic.
   - ARCH-PURE: pass. Deadline math is pure and unit-tested; only `After`/`Wait` touch `t.Deadline()` and the clock.
   - ARCH-PURPOSE: flag. This is the BR-1 residual. The class sweep is the deliverable, and the enumeration is incomplete.
   - Untouched packages are tracked in #485.

7. **Plan revisions:** none needed beyond correcting the round-1 Log claim after the sweep.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Sweep matched only guards whose next line is t.Fatal/t.Error; residual positive guards remain in touched packages: wrapcmd/agent_restart_test.go:109 (3s), wrapcmd/shortcut_passthrough_test.go:208 (1s), couchcmd/capture_test.go:235 (30s), couchtty/terminal_pressure_test.go:252 (5s), couchtty/console_messages_test.go:85 (3s deadline var), couchtty/console_pointer_test.go:279 (10s deadline var), couchcore/park_test.go:887 (5s), couchcore/park_lifecycle_live_test.go:193 (5s); couchtty/menu_perf_test.go:266 needs convert-or-exempt. Rule: any timer arm whose body (any statement) fails the test while a sibling arm awaits the event, including hoisted deadline := time.After(N), becomes testwait.After(t). The Log's "every positive select guard" claim is false until then.
  - id: BR-2
    disposition: addressed
    note: |
      notifytransport/transport_test.go:345-350 now asserts ENOBUFS or os.ErrDeadlineExceeded; test passes on darwin.
  - id: BR-3
    disposition: addressed
    note: |
      testwait_test.go:41-60 covers waitUntil past give-up (returns false after one poll), afterAt unbounded (nil channel) and afterAt past time (fires); dropping the bounded check would hang the test.
```

---

## Re-review — 2026-10-12T09:12:02-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 479 — tests: replace short real-time hang guards with event waits (couchtty waitUpTo 250/500ms, ~47 sites, and siblings) |
| repo | pair |
| issue file | workshop/issues/000479-tests-replace-short-real-time-hang-guards-with-event-waits-couchtty-waitupto-250-500ms-47-sites-and-siblings.md |
| boundary | whole-issue close |
| milestone | — |
| window | 97478ad961f162c25c10eb2f1018f122c5c77441..d4377b9bb164f2978688a2244ea5e90d9ac3ad57 |
| command | sdlc close --issue 479 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T09:12:02-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Now I'll check whether any fatal timer arms remain in the touched packages.

Those are absence windows, which the spec allows. Next I'll list all 37 remaining timer sites.

Most remaining sites are absence windows or fake clocks. Five need a closer look: couch_test.go:1112, leave_test.go:135, pane_birth_test.go:181, the harness_tty_live timers, and terminal_pressure:89.

BR-1's select-guard class looks clean, and harness reap is tracked in #485. Next I'll check sibling forms: polling deadlines and context timeouts.

A second form of the same class shows up: polling loops with `time.Now().Add(N)` deadlines. I'll sample a few, including files this window touched.

warm_reattach_test.go:208 is a hand-rolled copy of `testwait.Until` in a touched file. I'll check what the window changed in those files.

The remaining sites fall into two forms: polling deadlines and stopwatch ceilings. I'll count both across the touched packages.

I'll confirm the packages compile and that testwait's tests pass.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

BR-1 is addressed as written. In the touched packages, no timer arm still fails a test while a sibling arm waits for the event; all 37 remaining `time.After`/`NewTimer` sites are absence windows, fake clocks, or bounds that are the subject. But the same class survives in a second form that both close rounds missed: polling loops with a `deadline := time.Now().Add(N)` wall-clock bound that fail the test. Two of them hand-roll `testwait.Until` with a 1–3 s window, in files this window touched. A few stopwatch ceilings (`time.Since(start) > N`) also remain in touched files. Neither blocks the Done-when as the TL scoped it (listed sites only), but both belong to the families in play, so the fix should apply a rule rather than patch sites.

**1. Strengths**
- `cmd/internal/testwait/testwait.go` is small, and its arithmetic is pure and tested directly. `giveUpAt` and `waitUntil`/`afterAt` take explicit times, so the give-up paths are tested without real waiting, and `go test ./cmd/internal/testwait/` passes.
- Round 2 really did convert the irregular shapes the round-1 regex missed: hoisted deadlines (`console_messages_test.go:86`, `console_pointer_test.go:280`, `zellij_test.go:97`), multi-statement bodies (`agent_restart_test.go:109`, `shortcut_passthrough_test.go:208`), and `t` threaded into the park driver (`park_lifecycle_live_test.go:103,377`).
- Removing the 5 s safety from the fake launcher (`createflow_test.go:300`) is consistent with the spec: a launch that is never released hangs, and the binary's timeout names it.
- The bounds kept on purpose are justified: the harness `StartupTimeout` (its "startup timed out" error is what tests assert) and the provisioning fake's simulated deadline.

**2. Critical findings**
None.

**3. Important findings**
- **Polling deadlines are the same class as the converted timer arms.** **This is the 2nd finding in family `real-time-window-as-event-guard`.** The rule is: every wall-clock bound in a test whose expiry fails the test while it waits for an event becomes `testwait.After`/`Until`/`Wait`, unless the bound is the product's subject. That covers a select timer arm, a hoisted `time.After`, a `time.Now().Add(N)` polling deadline, and a `context.WithTimeout` whose expiry fails the test.

  Prevalence in the touched packages: about 32 polling deadlines that end in a failure. In files this window touched:
  - `wrapcmd/codex_working_test.go:186`: 1 s, a verbatim copy of `testwait.Until`, so also an ARCH-DRY finding
  - `couchcmd/warm_reattach_test.go:208`: 3 s, also a copy of `Until`
  - `wrapcmd/shortcut_passthrough_test.go:222`: 1 s
  - `wrapcmd/agent_restart_test.go:87`
  - `wrapcmd/detach_acceptance_test.go:235`
  - `couchtty/console_broadcast_test.go:150`
  - `couchtty/console_pointer_test.go:72,221`
  - `couchtty/notification_pty_test.go:29`
  - `couchtty/terminal_soak_test.go:200`
  - `couchcore/couch_test.go:148,1030,1748`

  In untouched files of touched packages:
  - `couchcmd/message_deliveries_test.go:49`
  - `couchcmd/message_lifecycle_test.go:170`
  - `couchcmd/message_service_test.go:257,339,378,652`
  - `couchcmd/wait_notice_test.go:272,324,465`
  - `couchcore/provision_lock_test.go:117`
  - `couchcore/ptyrunner_test.go:17`
  - `couchsingleton/subprocess_test.go:64`
  - `wrapcmd/launch_watcher_test.go:139`
  - the `*_live_test.go` files

  `shortcut_passthrough:222` waits out a product timeout, so check it before converting. Also add "polling deadlines" to #485's sweep for the untouched packages.

**4. Minor findings**
- **Stopwatch ceilings remain in touched files.** **This is the 2nd finding in family `ceiling-replaced-by-outcome`.** The rule is: a `time.Since(start) > N` failure is replaced by the outcome that proves the bound. The instances:
  - `couchtty/console_broadcast_test.go:357`: 3 s, right after the converted guard. The subject is `broadcastShutdownWait`, so the outcome is that `f.done` fired while `CloseBlock` was still unreleased, which the select already proves. The ceiling can simply go.
  - `couchcore/provision_subprocess_test.go:54`: 5 s. `DeadlineExceeded` is already asserted. Assert how the child ended instead.
  - `launcher/session_reap_test.go:175`: 2 s, in an untouched file.
- The harness teardown's `grace` reap window (50 ms) still covers listed site `harness_tty_live_test.go:362`: an unreaped zombie can make `processExists` true. #485's "Harness reap" item tracks this; cross-reference that site there.

**5. Test coverage notes**
- `go test ./cmd/internal/testwait/` passes.
- `go vet` passes for couchcore, couchsingleton, launcher, notifytransport and terminalqualify.
- couchcmd, couchtty and wrapcmd could not compile in this checkout: the embed assets at `runtimebundle/embed.go:8` are generated and missing here. This is environmental, not the diff, but those three packages were not compile-checked by this review.
- I did not re-run the soak.

**6. Architectural notes**
- ARCH-DRY: **flag.** `waitForCodexCouch` and `waitWarmAcceptance` are copies of `testwait.Until` with a latency guess. The other per-package wait helpers (`serviceRig`, the `couch_test` pid poll) should also delegate to testwait.
- ARCH-PURE: **pass.** testwait's IO (clock and sleep) is a thin shell over pure `giveUpAt`.
- ARCH-PURPOSE: **flag** (the Important finding above). The class sweep covered select arms only, not the polling form of the same guard.

**7. Plan revision recommendations**
- Revisions entry: "Close round 3: the guard class includes `time.Now().Add` polling deadlines and stopwatch ceilings, not only select timer arms. Converted in touched files; untouched packages are added to #485's sweep."

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      A scan of all 37 time.After/NewTimer sites left in the touched packages finds only absence windows, fake clocks or subject bounds; no fatal timer arm remains.
findings:
  - id: new
    severity: Important
    family: real-time-window-as-event-guard
    title: |
      Polling deadlines (time.Now().Add(N) + Fatal) in touched files still guard events with 1-3 s windows
    detail: |
      2nd finding in this family. Rule: every wall-clock bound whose expiry fails the test while it awaits an event (select arm, hoisted After, polling deadline, ctx timeout) becomes testwait unless the bound is the subject. Touched files: wrapcmd/codex_working_test.go:186 (1 s, verbatim testwait.Until copy), couchcmd/warm_reattach_test.go:208 (also an Until copy), wrapcmd/shortcut_passthrough_test.go:222, wrapcmd/agent_restart_test.go:87, wrapcmd/detach_acceptance_test.go:235, couchtty/console_broadcast_test.go:150, couchtty/console_pointer_test.go:72,221, couchtty/notification_pty_test.go:29, couchtty/terminal_soak_test.go:200, couchcore/couch_test.go:148,1030,1748. Untouched files in touched packages: couchcmd message_deliveries:49, message_lifecycle:170, message_service:257,339,378,652, wait_notice:272,324,465; couchcore provision_lock:117, ptyrunner:17; couchsingleton subprocess:64; wrapcmd launch_watcher:139. About 32 in total. Add polling deadlines to the #485 sweep.
  - id: new
    severity: Minor
    family: ceiling-replaced-by-outcome
    title: |
      Stopwatch ceilings remain in touched files (console_broadcast 3 s, provision_subprocess 5 s)
    detail: |
      2nd finding in this family. Rule: a time.Since(start) > N failure is replaced by the outcome that proves the bound. Instances: couchtty/console_broadcast_test.go:357 (the select on f.done with CloseBlock unreleased already proves the bound; delete the ceiling), couchcore/provision_subprocess_test.go:54 (DeadlineExceeded already asserted; assert how the child ended), launcher/session_reap_test.go:175 (untouched file).
```

---

## Re-review — 2026-10-12T09:24:03-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 479 — tests: replace short real-time hang guards with event waits (couchtty waitUpTo 250/500ms, ~47 sites, and siblings) |
| repo | pair |
| issue file | workshop/issues/000479-tests-replace-short-real-time-hang-guards-with-event-waits-couchtty-waitupto-250-500ms-47-sites-and-siblings.md |
| boundary | whole-issue close |
| milestone | — |
| window | 97478ad961f162c25c10eb2f1018f122c5c77441..d3e54be349829f1b38e0b8ec71d6c0901530b6a6 |
| command | sdlc close --issue 479 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T09:24:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Round 3's polling-deadline conversions look right. Now checking the remaining context-timeout bounds in touched files, which BR-4's rule also named.

Checking whether the provision-cancellation test actually catches a lost SIGKILL escalation, by temporarily removing it in this disposable checkout.

Confirmed: the test passes (3.1 s) with the escalation removed, so its new comment is false. Source restored. Now compile-checking the touched packages.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 3 fixed every site BR-4 listed. All 14 polling deadlines in touched files now use `time.Now().Add(testwait.Budget(t))`, and the untouched files are recorded in #485. `Budget` has a unit test, and the console_broadcast stopwatch is gone with its select proving the bound. Two things are still open. First, BR-4's own rule names four forms: select arm, hoisted After, polling deadline and **ctx timeout**. Round 3 swept only the polling form. At least seven `context.WithTimeout(…, 2–8s)` bounds in touched files still fail the test while it waits for an event, and one of them is a select guard arm (`case <-ctx.Done(): t.Fatal`), the form round 2 said it had swept. Second, the new comment in `provision_subprocess_test.go` says something the test does not prove. I checked this: I removed the SIGKILL escalation from `provision_io.go:83` and the test still passed in 3.1 s.

1. **Strengths**
   - `testwait.Budget`/`budgetAt` (`cmd/internal/testwait/testwait.go:32-45`) follows the package's split between pure logic and the IO shell, and `budgetAt` is tested with a fixed `now`.
   - `realDescendantRunner.wait` (`couchcore/couch_test.go:85,123`) passes the test's budget into the fake through a field rather than a global, so the helper's 2 s guess is gone.
   - `console_broadcast_test.go:350-355`: the ceiling was removed because the unreleased `CloseBlock` plus the select on `f.done` already proves the bound. That is the correct proof.
   - The #485 hand-off lists each deferred site by line instead of saying "and others".

2. **Critical:** none.

3. **Important**
   - **BR-4 is not fully addressed: the ctx-timeout form remains.** This is the 3rd finding in family `real-time-window-as-event-guard`; see the BR-4 disposition note for the rule that covers all of them. Instances in touched files, each of which fails the test when the timeout expires:
     - `couchtty/terminal_exit_test.go:178` (3 s; `case <-ctx.Done(): t.Fatal` at :195 and :201, the select-arm form round 2 claimed to have swept)
     - `couchcore/couch_test.go:753` (2 s; awaitThreadRegistration, then Fatal)
     - `couchtty/console_relaunch_chord_test.go:65` (3 s; runTerminalCommand, then Fatal)
     - `couchtty/terminal_soak_test.go:217` (5 s; Flush, then Fatal)
     - `wrapcmd/detach_acceptance_test.go:211` (8 s; polling loop that Fatals on `ctx.Err()`)
     - `couchcore/recovery_execute_test.go:607` (5 s; Await, then "never settled" Fatal)
     - `couchcmd/continuation_acceptance_test.go:250` and `couchcore/couch_test.go:500` (20 s, which is generous)

     Leave these alone: `couch_test.go:721` (pre-cancelled), `parkworker_test.go:254` and `recovery_execute_test.go:568` (goroutine cleanup that never fails), `retention_test.go:593` and `terminal_pressure_test.go:165,543` (the bound is the subject), and the zellij cleanup/detach kill contexts.

     Fix: add `testwait.Context(t)`, which returns a context with deadline `time.Now().Add(Budget(t))`, or call `context.WithTimeout(ctx, testwait.Budget(t))` inline. Then sweep the classified list above.

4. **Minor**
   - **BR-5 is not fully addressed** (`couchcore/provision_subprocess_test.go:49-51`). The comment says "a lost escalation hangs into the binary timeout". It doesn't: `cmd.WaitDelay` (`provision_io.go:72`) plus the group kill after `ErrWaitDelay` (`provision_io.go:99-100`) return `DeadlineExceeded` without the escalation. I removed the SIGKILL at `provision_io.go:83` and the test passed in 3.10 s. Either assert how the child ended, as BR-5 asked (for example, the group is gone or ended by SIGKILL, distinguishable from the WaitDelay backstop), or correct the comment to say the test checks only that the error is a deadline error. The family's other instances were the console_broadcast ceiling (fixed correctly) and `launcher/session_reap_test.go:175` (deferred to #485).
   - The atlas line at `atlas/index.md:28` is now longer than its neighbours; reflow it.

5. **Test coverage**
   - `TestBudgetRunsToTheGiveUpTime` covers the bounded and unbounded branches.
   - Nothing pins `Budget` at or past the give-up time. A negative budget is harmless, because the loops exit on their first check.
   - `go test ./cmd/internal/testwait` passes.
   - `go vet` on the touched packages fails only on a missing embedded asset (`runtimebundle/embed.go:8`, a checkout artifact unrelated to this diff); couchcore compiled and ran.

6. **Architecture**
   - **ARCH-DRY: pass.** Round 3 kept hand-rolled `Until` copies (`waitWarmAcceptance`, `waitForCodexCouch`, `waitDetachCondition`, the soak `wait`) and changed only their bound. Those could collapse into `testwait.Until`, but this is not a blocker.
   - **ARCH-PURE: pass.** The `*At` functions are pure, and `Budget`/`After`/`Wait` are thin clock shells.
   - **ARCH-PURPOSE: flag**, under BR-4 above: the sweep went by syntax (polling form) rather than by the rule's semantics (all four forms).

7. **Plan revisions**
   - Change the close-round-3 log entry to say ctx-timeout bounds were swept, or explicitly deferred with sites listed.
   - Change the provision_subprocess sentence ("a lost escalation hangs into the binary timeout") so it no longer claims what the WaitDelay backstop disproves.

```findings
dispose:
  - id: BR-4
    disposition: not-addressed
    note: |
      Polling sites all converted, but the rule's 4th form (ctx timeout) was not swept. The rule: every wall-clock bound in a test whose expiry fails the test while it awaits an event, whatever its syntax (select arm, hoisted After, time.Now().Add deadline, context.WithTimeout/ctx.Done arm), takes testwait's budget unless the bound is the subject. Sweep by classifying every WithTimeout/After/Now().Add/NewTimer in touched files, not by regex on one form. Remaining: couchtty/terminal_exit_test.go:178 (ctx.Done Fatal arms :195,:201), couchcore/couch_test.go:753, couchtty/console_relaunch_chord_test.go:65, couchtty/terminal_soak_test.go:217, wrapcmd/detach_acceptance_test.go:211, couchcore/recovery_execute_test.go:607, couchcmd/continuation_acceptance_test.go:250, couchcore/couch_test.go:500. Suggest a testwait.Context(t) helper. 3rd finding in family real-time-window-as-event-guard.
  - id: BR-5
    disposition: not-addressed
    note: |
      console_broadcast fixed correctly. provision_subprocess_test.go:49 claims a lost escalation hangs into the binary timeout, but WaitDelay plus the ErrWaitDelay group kill (provision_io.go:72,99) bound it anyway: with the SIGKILL at provision_io.go:83 removed, the test passed in 3.10s. Assert how the child ended, or correct the claim. session_reap_test:175 is deferred to #485.
```
