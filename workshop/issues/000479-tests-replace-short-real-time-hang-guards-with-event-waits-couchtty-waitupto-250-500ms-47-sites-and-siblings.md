---
id: 000479
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '0f2e387208cb04a1a6022d18d6d2b1ece40f2ee2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T07:00:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "f9d1aacd", done: "e78a039e"}
actual_hours: 2.24
---

# tests: replace short real-time hang guards with event waits (couchtty waitUpTo 250/500ms, ~47 sites, and siblings)

## Problem

Found by the #476 sweep (fixed real-time windows asserted under load, the #473/#476 flake family). Ranks are a reading of the code, not measured failures. Paths are under `cmd/internal/`.

These tests wait for an event with a short fixed bound (100 ms to 1.5 s), and fail if the event is late. The event is right; the bound is a performance assertion that a loaded `-race` suite can miss:
- couchtty `waitUpTo(t, 250ms/500ms, ...)`, about 47 sites: console_run_menu (~30), console_menu_operation, console_newest_page, console_menu, console_preview, console_completion, console_slotgit:116. Highest volume.
- couchtty select guards at 250/500 ms: console_run_menu_test.go:274,279,304,481,489,550,555,583; console_menu_test.go:54,65,71,119,131,136; console_preview_test.go:78,84,89,96,130,162,167; console_menu_operation_test.go:227,239; console_attach_transaction_test.go:76,81; console_mouse_test.go:184 (`timeoutAfter` 400 ms).
- couchcore/supervisorlease_subprocess_test.go:97 (750 ms poll on a re-exec'd subprocess; high), actionableinventory_test.go:352,360 (250 ms), launchhelper_test.go:114 (1 s).
- couchcmd/run_test.go:1299 (250 ms).
- wrapcmd: harness_tty_live_test.go:362 (1 s, kill and reap of a re-exec'd binary), overlay_test.go:72, terminal_model_test.go:910,918,926, composer_recognizers_test.go:608, stdout_batch_test.go:96,102,142, translate_stdin_test.go:101,121,157,188 and picker_overlay_test.go:218,228 (`waitFor(200ms)`).
- launcher/osruntime_launch_test.go:27,44 (1 s / 1.5 s on a real `sh` child).
- terminalqualify/candidate_test.go:182, couchsingleton/evidence_test.go:107, notifytransport/transport_test.go:342,402, couchtty/terminal_pressure_test.go:183 (redundant 1 s total next to its select).

## Spec

As #476 did for the retention lock test: where the event is the assertion, wait on it with no real-time window, and let the test binary's timeout catch a hang (its goroutine dump names the blocked call). Where a bound is part of the subject, keep it and inject the clock. Don't just raise the constants. couchtty's `waitUpTo` helper is one change for most sites. Before changing a site, prove it is a window: widen it with a temporary sleep in the product and watch the test fail.

Design (agreed with the TL, 2026-10-12): one shared helper, `cmd/internal/testwait`, bounds every wait by the test's own deadline minus a margin (10 s, or half the time left), not by a guess at event latency. A late event passes; a lost one fails with a message naming what the test waited for, before the binary's timeout. `testwait.After(t)` is the guard case of a select, `Until`/`Wait` poll. Sites that asserted a wall-clock ceiling (`elapsed > 1s`) assert the outcome that proves the bound instead: the product's own timeout error (`ErrThreadStoreBusy`, "lock timed out", "timed out") or how a child ended (SIGTERM vs SIGKILL). The binary timeout catches a lost bound. Absence windows (`case <-time.After(...):` with an empty body, "nothing happens in N ms") stay: under load they get less sensitive but never fail falsely.

## Done when

- No test in the list fails because an awaited event arrived late; each site either waits on its event or injects the clock it asserts.
- The listed sites' tests pass 200 runs under `-race` and parallel load (scoped; see Revisions).

## Revisions

- 2026-10-12: Done-when scoped to the listed sites, by the TL's call. The soak's second line ("touched packages pass 200 runs") isn't met by tests outside the list, which belong to other families. couchcmd, couchcore, couchsingleton, launcher, terminalqualify and testwait passed 200/200. wrapcmd passed 197/200. couchtty passed 120/128 before the soak was stopped. None of the remaining failures is a listed site or a late event: a lost notice repaint (`TestEndpointChrome*`, a product bug) → #484; real 50/100 ms product timeouts in the wrapcmd harness tests, plus ordering/state asserts (mouse parent-capture, the soak's shutdown "presenter unavailable", orientation overlay-after-paste) → #485. The second Done-when line now reads: the listed sites' tests pass the soak.

## Plan

- [x] Prove the window with a temporary product sleep: couchtty `paintNow` +300 ms fails 23 tests; wrapcmd stdin read +300 ms fails 3 `waitFor(200ms)` tests. The ceiling sites are windows by construction.
- [x] `cmd/internal/testwait` (`After`, `Wait`, `Until`) plus its unit tests.
- [x] couchtty: `waitFor`/`waitLong`/`waitUpTo` become one deadline-bounded `waitFor` (107 calls); 81 positive select guards use `testwait.After(t)`; the slot-git `settle` reads the schedule's idle state on the loop instead of waiting for a 20 ms quiet spell (the probe exposed it).
- [x] Other listed select guards: couchcore actionableinventory, couchcmd run, wrapcmd overlay/terminal_model/composer, terminalqualify candidate.
- [x] wrapcmd `waitFor`/`waitForStdoutBatch`: deadline-bounded; couchcore supervisorlease: `testwait.Wait`.
- [x] Ceilings: launchhelper, harness_tty_live, notifytransport ×2, couchsingleton evidence, terminal_pressure, launcher ×2, each asserting the outcome instead.
- [x] 200 `-race` runs of the touched tests under parallel load; full `make -k test`. The listed sites are clean; out-of-list failures went to #484/#485 (see Revisions).

## Log

### 2026-10-12
- 2026-10-12: closed — testwait bounds waits by t.Deadline()-margin (After/Until/Wait/Budget); window proofs (couchtty paint +300ms failed 23 tests, wrapcmd stdin +300ms failed 3; green after); every failing timer arm and polling deadline in touched files converted (rounds 1-3, scan-verified) except bounds that are the subject; ceilings assert outcomes; soak 200x -race under load: couchcmd/couchcore/couchsingleton/launcher/terminalqualify/testwait 200/200; out-of-list fails + untouched-file sites filed #484/#485 (TL-scoped Done-when, see Revisions); make -k test green (headless nvim failed only under soak load, passes unloaded); touched packages pass after round 3; remaining -race-only failures reproduce on origin/main (in #485); review verdict: FIX-THEN-SHIP

- TL's assignment note read "give each a bounded wait with a clear failure", against the spec's "let the binary timeout catch a hang". Asked the TL; agreed reconciliation: no performance window, bounded only by the test's deadline minus a margin, so the failure names the event (`testwait`).
- Window proofs: couchtty `paintNow` +300 ms failed 23 menu/preview/completion tests at the 250/500 ms waits; after conversion the same probe passes. wrapcmd stdin read +300 ms failed 3 `waitFor(200ms)` tests on the old code, none on the new. The probe also exposed a sibling window in `TestSlotGitTriggers`: `settle` read "done" from a 20 ms quiet spell; it now reads the slot-git schedule's idle state (`Running`, `Dirty`, buffered requests/results) on the loop.
- Ceilings, where the bound is the subject, now assert the outcome that proves it instead of a stopwatch: `ErrThreadStoreBusy` (couchsingleton, "refused, didn't wait out the 1 s publication context"), "lock timed out" (notifytransport Start), the product's "timed out" errors (launchhelper, harness_tty_live), and the ending signal (launcher). No product clock seam was needed: each product bound already leaves an observable outcome.
- Absence windows (`case <-time.After(N):` with an empty body) are left alone: load makes them less sensitive, never falsely red.
- The soak found three races the stopwatches had hidden. (1) `TestCancelledHandoffKillsAChildThatIgnoresSIGTERM` cancelled at 50 ms, before `sh` set its TERM trap under load, so the child died by SIGTERM and the escalation went untested (the old 1.5 s ceiling passed it). It now cancels once the trap has been set. (2) `TestOrientationLiveStartupFrames` armed a real 20 ms readiness deadline for ready frames too; ready frames lost to it under load (25 of 40 early soak runs). Only not-ready frames arm it now. (3) `TestOrientationUnsolicitedReplyAndReplyWithInputCancel` painted the composer before the stdin reader admitted the input, so the paste could win. It now paints once the input has reached the child.
- Close round 1 (FIX-THEN-SHIP). BR-1, Important: the touched files still held fatal 1-3 s `time.After` guards of the same class. Fixed the class, not just the files the review named: every positive select guard (`case <-time.After(N):` followed by `t.Fatal`/`t.Error`) in all touched packages now uses `testwait.After(t)` (79 more across wrapcmd, terminalqualify, notifytransport, couchcore, couchcmd, matched by a regex on the next line). Untouched packages (broadcast, webcast, termcmd, couchmessage, …) are filed in #485. Minor: `TestSendToUnavailableReaderIsBounded` asserted no outcome after its stopwatch went; it now asserts the refusal (darwin answers ENOBUFS at once, measured; a blocking queue would hit the write deadline). Minor: testwait's give-up paths (`waitUntil` past its time, `afterAt` unbounded or past) now have unit tests.
- Close round 2: BR-1 stayed open because the regex missed guards whose body starts with another statement (`con.Stop()`, `_ = writer.Close()`, a log read) and hoisted `deadline := time.After(N)` variables. The rule, applied by a scan of every timer arm in the touched packages: any timer arm that fails the test while a sibling arm awaits the event becomes `testwait.After(t)`. Converted by hand: wrapcmd agent_restart, shortcut_passthrough; couchcmd capture; couchtty terminal_pressure (its "shared five-second deadline" was a test budget, not a product one), console_messages and console_pointer (hoisted), menu_perf (its correlated-frame wait; the perf numbers it measures are #481's); couchcore park_test and park_lifecycle_live (the driver now carries its `t`); launcher zellij_test. The launcher fake's 5 s release safety went: an unreleased launch hangs, and the binary timeout names it. Kept, because the bound is the subject: the wrapcmd harness's `StartupTimeout` timer (its "startup timed out" error is what tests assert), and the provisioning fake's simulated command deadline.
- Close round 3 (BR-4, the same class in a third form): polling loops with `deadline := time.Now().Add(N)` and a Fatal. Added `testwait.Budget(t)` (time left until give-up, tested) so each becomes `time.Now().Add(testwait.Budget(t))`. Converted in touched files: couchcmd warm_reattach (a hand-rolled Until); couchcore couch_test ×4 (the descendant runner now carries the test's budget); couchtty console_broadcast `until`, console_pointer ×2, notification_pty, terminal_soak; wrapcmd agent_restart, detach_acceptance, codex_working (another Until copy), shortcut_passthrough ×2. Left alone: cleanup courtesy waits that never fail (notification_pty:155, couch_test's 50/100 ms cleanup probes) and menu_perf's measurement loop (#481). Untouched files went to #485. Stopwatch minors: console_broadcast's 3 s exit ceiling is gone (the select on `f.done`, with CloseBlock unreleased, is the proof). provision_subprocess's child now sleeps 3600 s instead of 30, so a lost escalation hangs into the binary timeout rather than needing a 5 s ceiling; the error still names the deadline.

