---
id: 000450
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '528f9bd31d63111f5d90aa3d39b54956b6087ffc' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T13:19:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
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

- [x] Bisect `test-review`: no landing broke it. The 8 failures reproduce at #436's merge (80cc80af) and pass there unsandboxed. `review.lua`'s restore controller (#341, 09-28) calls `serverstart`, the agent sandbox refuses the unix socket, the error aborts VimEnter, and `start_review` never runs. #436 was verified unsandboxed; today's workers ran sandboxed.
- [x] Fix: a refused restore socket warns and publishes the open state without an endpoint (restore_client already reads that as a legacy pane), so review still works on such a host. Unit test in `restore_controller_test.lua`.
- [ ] Preflight: `make test` first probes that the host can bind a unix socket and open a PTY; if not, it fails at once naming the cause, so a sandboxed run cannot be mistaken for ~150 code failures.
- [ ] `go test ./...` triage. Unsandboxed, raw `go test ./...` fails 6: launcher `TestLaunchNative*`/`TestCheckpoint*` and couchcmd `TestContinuationWriter…`, all from the Couch shell's `COUCH_PAIR_DATA_DIR` (read since #366) that their hand-pinned env lists miss. Pin it. Under `make test`'s scrub they already pass. pair#423 passed 20/20 isolated and in the full run; it stays under its own issue, no quarantine.
- [x] Coverage decision: `make test` already ends with `go test ./... -count=1` (Makefile.local), so it covers every package and is the one gate. The raw command inherits the session env; the tests pin what they read.

## Log

### 2026-10-11
