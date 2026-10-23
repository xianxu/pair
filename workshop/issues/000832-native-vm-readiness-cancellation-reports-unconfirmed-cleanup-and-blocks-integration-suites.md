---
id: 000832
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a310033c9b13aeaf8292d80ade9faa1f84cfb536' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T01:02:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/vmcmd.TestNativeReadinessRunCancellationRetainsCleanupAuthority]
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
- [ ] Instrument the existing process owner to distinguish Wait blockage from residual group membership; report the demonstrated cause before fixing, preserving the existing timeout and ownership proof.
- [ ] Fix the demonstrated cause, pin its ordering regression, pass at least100 repeated native cancellations and meaningful process/descendant tests, then recorded verification, close and priority landing.

## Log

Failed recorded suite attempt04cb95c189fa4fe5b8267c7641d54b14 on b3afa544d6645e9f04d2c03179afb257dc159269, exit2/11m53s, raw log SHA256 prefix96f529c34edf; retained /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/04cb95c189fa4fe5b8267c7641d54b14.log. Failure driver_darwin_test.go:517 after10.03s: execution cancellation discarded cleanup authority: context canceled / VM subprocess cleanup unconfirmed / context deadline exceeded. SDLC's diagnostic main baseline54a02d7aa3971f50987bcef321de6534d8d566a9 passed the full suite10m52s; that is a different-head diagnostic, not same-head flake proof. Isolated unchanged-head command go test ./cmd/internal/vmcmd -run '^TestNativeReadinessRunCancellationRetainsCleanupAuthority$' -count=1 failed10.265s with the same assertion after10.12s. Go1.27.3 darwin/arm64. No retry, registration, bypass or source change made. Source originated in landed #743; cleanupScope and runOwned must be inspected with ownedProcess.Finish rather than assuming the assertion identifies the cause.


### 2026-10-23 — Claimed priority triage; main reproduces without #789

Authenticated hub5c051ce1 dispatches classification ahead of#728; authenticated6b4dd173 confirms root-cause fix priority, no timeout increase, cause report before fixing, and at least100 repeats. Claimed#832 and started its own branch from main; #728 remains pushed at d64d87347e061c0cf107232303edd964c8e9e155, with its independently verified detached check069c4c72a8a630a0faff10606cc4367d21d02354 retained on runner74025/lease860972511. Switching this checkout cannot alter that queued snapshot. No production fix yet.

Fresh read-only worktrees: main1cc10fdcc1020bed5400255841785ba5d093267f and #789 b3afa544d6645e9f04d2c03179afb257dc159269. Git reports identical vmcmd/go.mod/go.sum. Go1.27.3 darwin/arm64, GOFLAGS=-p=3 -trimpath, default SSD GOCACHE. First isolated exact test passed on main0.05s (raw SHA256fbc17d272927db8c99dcdf0491dc6f777ef37ca1a830af74e3f0263c7472ac59) and #7890.04s (8aeced708795d9aa38ab967fdb5bfc7c22e184a453a4b1301b4cdec4d48f46ba).

Repeated30 main:23PASS/7FAIL, total71.631s, raw SHA2566c4efd1ed0d9ab484fe47d5868de8285504e8685897ca45f114cde8095cc270d. Repeated30 #789:26PASS/4FAIL, total41.433s, a10ae971d0157ee1b4604f8a372e8446f934d5ab7ade73a71a6f8cfe345004da. Failures have the same assertion after about10.04–10.06s: context canceled / VM subprocess cleanup unconfirmed / context deadline exceeded. Main failures were observed before starting the #789 repeat; those repeat runs later overlapped, so counts are reproduction evidence, not comparative rates. This establishes intermittent main failure, not a #789-specific defect; it does not attribute the cause to Go's version. Reported before fixing, submitted hub receipt107be92c-4555-48d5-9dd2-0bfc6e27670e, followed by final counts a155fecc-43b1-4fef-a417-1e24787084bc.

Raw files live in /Users/xianxu/.local/share/pair/evidence/000832/triage.6UUKdV: main-go1273.log, integrated789-go1273.log, main-30-go1273.log and integrated789-30-go1273.log. Preserve these alongside the prior failed registered suite04cb95c189fa4fe5b8267c7641d54b14 and pair:4's isolated failure on the same integrated head.

Authenticated peer message b4e442f3-f8b0-43d0-847e-f8b4b31ba659 reports registration completed under TLc7fa55d6 using that suite/isolated failure plus our exact-head/main passes; evidence text was not stored on the card. Peer retained targeted resume parent04cb95c1, childb6b44350aa8e680033121f71d09ce86b, lease1117603018, FIFO6; no source/timeout changes, and it will stop if the failure repeats. This Log records the fail/pass provenance requested by that peer; no duplicate registration or quarantine performed here.

Read-only cleanup trace: runOwned selects the existing detached cleanupScope when the boot context is cancelled, so the10s failure does not alone establish authority loss. Finish can fail either while joining cmd.Wait or while awaiting group absence. The fake removes its readiness barrier before spawning sleep30; group cancellation can therefore encounter different descendant states. Next use a disposable test overlay to observe which Finish phase expires and the exact group members, without changing production behavior (ARCH-ORDER, ARCH-SECURE).
