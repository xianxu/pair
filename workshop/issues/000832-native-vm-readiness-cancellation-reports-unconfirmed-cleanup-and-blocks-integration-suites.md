---
id: 000832
status: codecomplete
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '6eb41af7c124932a3830ca85659cb1fca641a6eb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T01:02:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/vmcmd.TestNativeReadinessRunCancellationRetainsCleanupAuthority]
flow: {kind: quick, provenance: inferred, spec: "4d5cfaeb", done: "b9daf772"}
actual_hours: 0.48
---

# Native VM readiness cancellation reports unconfirmed cleanup and blocks integration suites

## Problem

The #789 integration suite is blocked by vmcmd.TestNativeReadinessRunCancellationRetainsCleanupAuthority. It failed in the full suite and reproduced on the same unchanged head with an isolated targeted run. This is not established as a flake. #789 changes no vmcmd source relative to its integrated main.

## Spec

Investigate the native fake readiness cancellation and owned subprocess cleanup lifetime. Preserve cancellation reporting, detached shared cleanup authority, process-group ownership and confirmed reap. Fix the cause without increasing deadlines or weakening the cleanup proof. The fixture removes hold-first-probe then sleeps30s; cancellation must clean that process tree within the existing cleanup scope.

## Done when

- Root cause of the unconfirmed cleanup is demonstrated with a regression through the native cancellation path.
- The existing readiness cancellation test and meaningful process/descendant lifecycle regressions pass without a timeout bump or weakening ownership.
- Record exact-head verification and unblock the dependent integration suite through normal SDLC gates.

## Plan

- [x] Compare the named native test on fresh main and the exact #789 integrated head under Go1.27.3; retain raw fail/pass evidence.
- [x] Instrument the existing process owner to distinguish Wait blockage from residual group membership; report the demonstrated cause before fixing, preserving the existing timeout and ownership proof.
- [x] Fix the demonstrated cause, pin its ordering regression, and pass at least100 repeated native cancellations and meaningful process/descendant tests; recorded verification, close and priority landing follow through their gates.

## Log

Failed recorded suite attempt04cb95c189fa4fe5b8267c7641d54b14 on b3afa544d6645e9f04d2c03179afb257dc159269, exit2/11m53s, raw log SHA256 prefix96f529c34edf; retained /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/04cb95c189fa4fe5b8267c7641d54b14.log. Failure driver_darwin_test.go:517 after10.03s: execution cancellation discarded cleanup authority: context canceled / VM subprocess cleanup unconfirmed / context deadline exceeded. SDLC's diagnostic main baseline54a02d7aa3971f50987bcef321de6534d8d566a9 passed the full suite10m52s; that is a different-head diagnostic, not same-head flake proof. Isolated unchanged-head command go test ./cmd/internal/vmcmd -run '^TestNativeReadinessRunCancellationRetainsCleanupAuthority$' -count=1 failed10.265s with the same assertion after10.12s. Go1.27.3 darwin/arm64. No retry, registration, bypass or source change made. Source originated in landed #743; cleanupScope and runOwned must be inspected with ownedProcess.Finish rather than assuming the assertion identifies the cause.


### 2026-10-23 — Claimed priority triage; main reproduces without #789
- 2026-10-23: closed — Pinned terminal barrier regression failed before the fix and passes for cancellation and writer failure; unknown terminal proof refuses. Native readiness cancellation 150/150 PASS7.500s, race owner/descendant/native count10 PASS3.836s, complete vmcmd PASS42.432s; audited reader inventory PASS immediately before close. Recorded affected check is queued/running and must pass on final source before landing; existing cleanup timeout, positive absence and no-post-reap signaling retained.; review verdict: SHIP

Authenticated hub5c051ce1 dispatches classification ahead of#728; authenticated6b4dd173 confirms root-cause fix priority, no timeout increase, cause report before fixing, and at least100 repeats. Claimed#832 and started its own branch from main; #728 remains pushed at d64d87347e061c0cf107232303edd964c8e9e155, with its independently verified detached check069c4c72a8a630a0faff10606cc4367d21d02354 retained on runner74025/lease860972511. Switching this checkout cannot alter that queued snapshot. No production fix yet.

Fresh read-only worktrees: main1cc10fdcc1020bed5400255841785ba5d093267f and #789 b3afa544d6645e9f04d2c03179afb257dc159269. Git reports identical vmcmd/go.mod/go.sum. Go1.27.3 darwin/arm64, GOFLAGS=-p=3 -trimpath, default SSD GOCACHE. First isolated exact test passed on main0.05s (raw SHA256fbc17d272927db8c99dcdf0491dc6f777ef37ca1a830af74e3f0263c7472ac59) and #7890.04s (8aeced708795d9aa38ab967fdb5bfc7c22e184a453a4b1301b4cdec4d48f46ba).

Repeated30 main:23PASS/7FAIL, total71.631s, raw SHA2566c4efd1ed0d9ab484fe47d5868de8285504e8685897ca45f114cde8095cc270d. Repeated30 #789:26PASS/4FAIL, total41.433s, a10ae971d0157ee1b4604f8a372e8446f934d5ab7ade73a71a6f8cfe345004da. Failures have the same assertion after about10.04–10.06s: context canceled / VM subprocess cleanup unconfirmed / context deadline exceeded. Main failures were observed before starting the #789 repeat; those repeat runs later overlapped, so counts are reproduction evidence, not comparative rates. This establishes intermittent main failure, not a #789-specific defect; it does not attribute the cause to Go's version. Reported before fixing, submitted hub receipt107be92c-4555-48d5-9dd2-0bfc6e27670e, followed by final counts a155fecc-43b1-4fef-a417-1e24787084bc.

Raw files live in /Users/xianxu/.local/share/pair/evidence/000832/triage.6UUKdV: main-go1273.log, integrated789-go1273.log, main-30-go1273.log and integrated789-30-go1273.log. Preserve these alongside the prior failed registered suite04cb95c189fa4fe5b8267c7641d54b14 and pair:4's isolated failure on the same integrated head.

Authenticated peer message b4e442f3-f8b0-43d0-847e-f8b4b31ba659 reports registration completed under TLc7fa55d6 using that suite/isolated failure plus our exact-head/main passes; evidence text was not stored on the card. Peer retained targeted resume parent04cb95c1, childb6b44350aa8e680033121f71d09ce86b, lease1117603018, FIFO6; no source/timeout changes, and it will stop if the failure repeats. This Log records the fail/pass provenance requested by that peer; no duplicate registration or quarantine performed here.

Read-only cleanup trace: runOwned selects the existing detached cleanupScope when the boot context is cancelled, so the10s failure does not alone establish authority loss. Finish can fail either while joining cmd.Wait or while awaiting group absence. The fake removes its readiness barrier before spawning sleep30; group cancellation can therefore encounter different descendant states. Next use a disposable test overlay to observe which Finish phase expires and the exact group members, without changing production behavior (ARCH-ORDER, ARCH-SECURE).

### 2026-10-23 — Demonstrated Wait-phase late-fork race

Error-path-only overlay on unchanged main: 92PASS/8FAIL in100, all eight failures in cmd.Wait join and none in group-absence polling; raw main-phase-instrumented-100.log SHA256 e657734f15dd7c7ea9bbd9eb279988503a296beed747fdedc027d2102b6c8215 under the retained triage directory. A subsequent kernel capture at Wait expiry shows a live orphan, e.g parent19744 already reaped; PID19755 PPID1 PGID19744 state2. Cancellation signals twice rapidly then retires signaling and reaps without observing parent terminal; a concurrently published shell child survives and holds output pipes. This is not a zombie-only-group finding. Cause report submitted to hub, receipt3b022576-4473-416c-9ab1-fc847ba3e1b2.

Correction: preserve the direct-child PID pin through a terminal observation under the existing cleanup context before final group signal/retirement/Wait. Output failure cannot short-circuit that cleanup barrier. Keep positive final group absence, unknown-proof refusal and existing timeouts. Pin order through the existing owner seams, then run100+ real native cancellations. ARCH-ORDER and ARCH-SECURE require the barrier before surrendering signal authority; no new state, authority or timeout policy.

### 2026-10-23 — Ordering correction and targeted proof

The ordering regression first failed on unmodified production for cancellation and prior writer failure (late descendant retained), and the unknown-terminal regression first failed because cleanup incorrectly accepted nil. Fix adds the pinned terminal barrier before final signal/retire/reap; cleanup observation ignores prior output error while preserving execution's original error behavior. Existing final absence and retirement tests pass. This is an ordering protocol across ownership phases, not a syntactic lint pattern; the stateful ordering regression is the mechanical guard, with the lesson recorded in workshop/lessons.md.

Fixed native cancellation:150PASS/0FAIL,7.500s, raw fixed-native-150.log SHA256125e7cb30719a934445b2be85401a9931da22b9c4fbada6a02611bd437839ad7. Owner, descendant, positive-absence and native cancellation tests with race detector, count10:PASS3.836s (fixed-owner-race-10.log). The final baseline Wait-member overlay30 had23PASS/7FAIL, raw main-wait-instrumented-30.log SHA25660ff6b6196d539faf661b5e1d7a9edc337d96aeef24296816d60f5e3a9a866a3. All raw files retained under the previously named triage directory.

Reader audit: process_owner_unix_test.go inputs remain literal /bin/sh scripts, real native child observations and injected local lifecycle functions; the additions read no repository fixtures or external mutable files. Refreshed only that local_or_literal_fixture certificate through the inventory's exact printed command. Full vmcmd package PASS42.432s (fixed-vmcmd-package.log); final make test-affected-inventory PASS. Recorded verification and close remain pending before landing.

### 2026-10-23 — SHIP; recorded verification retained through infrastructure repair

Close runner5557 exited0: SHIP/high confidence, zero findings, reviewed source04d8ff3f22cbd21d66a4c35cc393cca76e0d706b; reviewer independently verified regression failures against base and repeated native150/race10. Close evidence d590c690d2bdce645e569ecb2c0311a1cb58e3b4; card codecomplete,0.48h measured/adopted. PR554 CI green. Merge runner11796 exited1 pending CI and the recorded affected/full-suite record; no merge runner remains active.

Recorded affected chose full suite, attempt cb2106650ab119dd5675503a440daf5e, runner33650, lease-3568798822, immutable snapshot /Volumes/TestScratch/local-checks/sdlc-local-185600369. Preserve it until joined; do not duplicate/requeue. It tests the reviewed executable source; later close/log commits are prose/evidence only. #728 runner74025/lease-860972511 remains independently queued and unchanged.

Infrastructure (not test evidence): no suite holders but mutually contradictory next-in-line output exposed old/new helper admission deadlock. Report b5856049-bc15-49c1-9770-d51fbed601b4. Authenticated hub c2f37c08 confirmed old tickets32–36 deferred to #832 grant while newer aging helper deferred that grant to the old tickets; revoked grant and suites started at once. Correct issue is ariadne#570 per authenticated correction9dbbb1da. #832 keeps its normal FIFO place. No helper changes, queue edits or cancellation by this worker.

Next: join recorded runner33650; on genuine pass update evidence and re-enter sdlc merge --yes, retaining its runner through landing/archive verification. On a failure classify the exact test before retrying. #728 waits behind this fix, preserving its retained suite.

### 2026-10-23 — Recorded full suite PASS; landing

Original retained runner33650/lease3568798822 joined EXIT0: actual full suite PASS11m45s on reviewed executable source04d8ff3f22cbd21d66a4c35cc393cca76e0d706b. Saved /Users/xianxu/workspace/pair/.git/sdlc/local-checks/04d8ff3f22cbd21d66a4c35cc393cca76e0d706b/suite.log SHA256ea27ac1c262e5b496fc49f67161206e3c4c14101f7106c7b6922c04d67467a5d. Zero Go failure markers and zero known-flake reruns; independently read back exact-source GitHub sdlc-local/suite SUCCESS. vmcmd25.153s, couchcore185.604s, couchcmd58.624s, termcmd2.987s, wrapcmd61.240s passed. Optional upstream affected-config qualification skipped for unavailable source, not claimed passed; required inventory and actual full suite passed.

Removed only the two owned read-only triage worktrees after retaining raw logs/overlays on SSD. No source/private cache or machine-setting changes. #728 used this wait for its integrated narrow re-close: SHIP/no findings, pushed; it waits behind this priority landing. Returning to sdlc merge --yes now; retain that runner through integration and archive verification, then report PR554 landing and return to#728 PR551.
