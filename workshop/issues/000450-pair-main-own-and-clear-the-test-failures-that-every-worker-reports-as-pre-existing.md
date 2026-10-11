---
id: 000450
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'b60729638cecd6c3dcbbadf5be6dbfc8a26ffb1d' # card fields mirrored from issue-cards; edit via sdlc
---

# pair main: own and clear the test failures that every worker reports as pre-existing

## Problem

On 10-11 four workers each reported the same failures on pair main as "pre-existing, not mine" and moved on: `make test`'s `test-review` (8 failures), and in `go test ./...` the launcher `TestLaunchNative*`/`TestCheckpoint*`, couchcmd (`TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins`, pair#423), and a continuation-writer test. pair#436 made `make test` green at 02:54, so `test-review` regressed during the day. How it gets in: pair's `sdlc close` does not run the suite; it records the worker's `--verified` text. A worker that sees a failure also fail on main labels it pre-existing, so a regression that lands once is then everyone's background noise. `go test ./...` covers packages `make test` does not, so failures there are never gated at all.

## Spec

- Find which landing broke `test-review` (bisect from pair#436's merge), and fix or revert it.
- For each `go test ./...` failure: fix, or quarantine with a named issue (as ariadne#330 did with `WF_TEST_QUARANTINED`), so a clean run is the norm.
- Decide whether `make test` should cover every package `go test ./...` does (the operator's intent: one suite that is the gate).
- The gate itself (the suite runs at landing, not as typed evidence) is ariadne's `sdlc merge` integration issue; link it, don't build it here.

## Done when

- `make test` passes on pair main, and `go test ./...` passes or its exceptions are quarantined under named issues.
- The Log names the commit that broke `test-review`.

## Plan

- [ ] Bisect `test-review`; fix.
- [ ] Triage each `go test ./...` failure: fix or quarantine with an issue.
- [ ] Decide `make test` coverage; record in the Log.

## Log

### 2026-10-11
