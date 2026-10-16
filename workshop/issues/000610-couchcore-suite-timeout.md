---
id: 000610
status: codecomplete
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: high
card_mirror: '8efa678437e68f3779884c86a6a5a030e39fd063' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T03:54:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "2ad97ea0", done: "2e9ba8a8"}
actual_hours: 0.28
---

# couchcore package takes ~10 min, at the go test timeout

## Problem

Couchcore takes nearly Go's default ten-minute per-package test timeout. This blocks otherwise verified pair landings under fleet load, including #591 and #589. In #591's recorded integration suite abe0a5ed883f06ffe58305cb45a29c09, couchcore timed out at 600.245 s; every other recipe/package passed. The automatic main comparison 2064bde22de3d40f5d15091fe41ef443 passed, but couchcore alone took 580.585 s. This is a pair item in ops/test-economy; the hub owns that project.

## Spec

Apply the hub's explicit temporary unblock: give the full suite's broad Go invocation an explicit 20m per-package timeout in Makefile.local, with this issue reference and a temporary-budget explanation beside the recipe. Preserve the complete package selection, cache-disabled execution, assertions and fixture deadlines. Profile/split/shard work is a separate unassigned follow-up. No local-gates configuration changes or retries are part of this fix.

## Done when

- [ ] The suite invokes go test with an explicit 20m timeout and retains ./... and -count=1; the Makefile explains the temporary budget with this issue reference.
- [ ] An unassigned test-economy follow-up records profiling, package splitting or sharding, and removal of the temporary allowance after a measured execution budget is restored.
- [ ] Recipe verification and the recorded full suite pass; close review is SHIP and the unblock lands before #591 and #589 resume.

## Plan

- [x] From main, change only the broad suite recipe and its temporary-budget comment; verify actual Make argv before/after.
- [x] File the performance follow-up; run build and the recorded suite, close, and land before resuming #591 unchanged.

## Log

### 2026-10-16 — approved fleet unblock
- 2026-10-16: closed — Explicit suite argv red/green; make-test-env conformance PASS (215 knobs, overrides, nested Make); clean make -k -B build on 6b7831ec5bd5d772efe0e20828e22aaae40e4fbf; recorded suite2cd9dc07fa29172d19df49c8be76332f PASS every recipe/package, 14m26, couchcore453.524s, log SHA2567e6e89a956a53b4bdc071c0a379ce4705d2077caadde65444d43ff65b03310d5; actual test binary20m. #611 filed unassigned for profiling/split/shard/removal. Atlas not applicable: no new architectural surface; ops/test-economy project is TL-owned. Budget explicitly approved in hub decision173f498b; fixture deadlines/assertions untouched.; review verdict: SHIP

Hub decision 173f498b-9796-4fd6-a777-07e28ce4107c explicitly authorizes 20m as a deliberate temporary budget. #591 remains clean at 47a221c079675acc1415c1a85db3ac559fcb1722 and is not rerun yet. Integrated suite failed only the package-wide timeout; automatic main comparison passed in 946.514 s with couchcore at 580.585 s. The fixed duration is visible at the suite owner (ARCH-DRY), not injected through shared GOFLAGS or hidden retry machinery. Production behavior is unchanged.

### 2026-10-16 — minimal implementation and recipe verification

Started independently from main a00d0ceee25e3ca0f8703e52531b20c2d112f818. Filed #611 unassigned for the measured performance remedy and eventual allowance removal. A direct Make dry-run argv check failed before the change (only go test ./... -count=1), then passed with exactly go test ./... -count=1 -timeout 20m. This configuration-only edit adds two explanatory comments and one explicit flag; no runtime code or fixture assertions/deadlines change. Existing tests/make-test-env-test.sh passed both outer Make variants, all 215 derived knobs, explicit overrides and nested Make. make -k -B build and git diff --check passed. The required recorded suite will verify the real compiler/test invocation; the outer suite budget remains 30m.

### 2026-10-16 — full verification

Recorded suite 2cd9dc07fa29172d19df49c8be76332f passed every recipe/package on 6b7831ec5bd5d772efe0e20828e22aaae40e4fbf, exit 0 in 865.892 s (14m26), queue 0.091 s. Couchcore passed in 453.524 s. The log shows the real broad invocation with -timeout 20m and the running test binary independently showed -test.timeout=20m0s. Log SHA256 independently verified: 7e6e89a956a53b4bdc071c0a379ce4705d2077caadde65444d43ff65b03310d5. A clean make -k -B build reported the same vcs.revision with vcs.modified=false. No speedup is claimed by the budget change; #611 owns the performance remedy. Atlas is waived because this changes no architectural surface; ops/test-economy project updates are left to its TL owner.
