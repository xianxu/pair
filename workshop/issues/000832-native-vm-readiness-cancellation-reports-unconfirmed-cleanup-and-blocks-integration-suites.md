---
id: 000832
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '8d5e243717b12ec22c273fc4132d0ff91d8f22ae' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

Failed recorded suite attempt04cb95c189fa4fe5b8267c7641d54b14 on b3afa544d6645e9f04d2c03179afb257dc159269, exit2/11m53s, raw log SHA256 prefix96f529c34edf; retained /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/04cb95c189fa4fe5b8267c7641d54b14.log. Failure driver_darwin_test.go:517 after10.03s: execution cancellation discarded cleanup authority: context canceled / VM subprocess cleanup unconfirmed / context deadline exceeded. SDLC's diagnostic main baseline54a02d7aa3971f50987bcef321de6534d8d566a9 passed the full suite10m52s; that is a different-head diagnostic, not same-head flake proof. Isolated unchanged-head command go test ./cmd/internal/vmcmd -run '^TestNativeReadinessRunCancellationRetainsCleanupAuthority$' -count=1 failed10.265s with the same assertion after10.12s. Go1.27.3 darwin/arm64. No retry, registration, bypass or source change made. Source originated in landed #743; cleanupScope and runOwned must be inspected with ownedProcess.Finish rather than assuming the assertion identifies the cause.
