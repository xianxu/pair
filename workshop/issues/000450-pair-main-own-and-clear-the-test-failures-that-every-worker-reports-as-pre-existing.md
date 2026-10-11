---
id: 000450
status: codecomplete
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '7dd32d151b00f9fedc0203ab397aae0c6cb9730f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T13:19:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "31818637", done: "1b0aa7ea"}
actual_hours: 1.04
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
- [x] Preflight: `make test` first probes that the host can bind a unix socket and open a PTY; if not, it fails at once naming the cause, so a sandboxed run cannot be mistaken for ~150 code failures.
- [x] `go test ./...` triage. Unsandboxed, raw `go test ./...` fails 6: launcher `TestLaunchNative*`/`TestCheckpoint*` and couchcmd `TestContinuationWriter…`, all from the Couch shell's `COUCH_PAIR_DATA_DIR` (read since #366) that their hand-pinned env lists miss. Pin it. Under `make test`'s scrub they already pass. pair#423 passed 20/20 isolated and in the full run; it stays under its own issue, no quarantine.
- [x] Coverage decision: `make test` already ends with `go test ./... -count=1` (Makefile.local), so it covers every package and is the one gate. The raw command inherits the session env; the tests pin what they read.

## Log

### 2026-10-11
- 2026-10-11: closed — At b049190c: unsandboxed make -k test exit 0 (host-preflight ok incl. exec ps probe, review-window-test ok, go test ./... -count=1 inside it green); raw go test ./... -count=1 from a live Couch shell (COUCH_PAIR_DATA_DIR set) exit 0, was 6 failures. Sandboxed: restore_controller_test.lua, restore_client_test.lua (new unserved case, red before fix) and tests/review-window-test.sh pass (was 8 FAIL); host-preflight names socket+pty+exec-ps refusals; PATH without python3 reports python3 missing separately. Bisect: test-review fails identically at #436 merge 80cc80af sandboxed, passes unsandboxed -> environment, not a landing. pair#423 20/20 isolated. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

- What broke `test-review`: no landing. It fails identically at #436's merge (80cc80af) when run sandboxed and passes there unsandboxed. The sandbox-sensitive code is #341's restore controller (e5dd82c4, 09-28: `serverstart` at VimEnter). #436 was verified outside the sandbox; the four 10-11 workers ran inside it. Baseline unsandboxed `make -k test` on unmodified main exits 0.
- Sandboxed `go test ./...` fails ~150 tests across 14 packages (unix sockets, PTYs: couchmessage, notifytransport, ptychild, wrapcmd, terminal, ...). These are the host, not the code; `tests/host-preflight.sh` now says so first.
- Unsandboxed raw `go test ./...` in a Couch shell: 6 failures, all `COUCH_PAIR_DATA_DIR` leaking (read since #366; launcher's `LaunchNative` refuses a `PAIR_DATA_DIR` outside the selected root). Fixed: launcher `TestMain` unsets it; the continuation acceptance child env pins it empty. pair#423 (`TestColdResume…`) passed 20/20 isolated and in the full run: no quarantine, stays under #423.
- Coverage decision: `make test` already ends in `go test ./... -count=1`, so it covers every package and is the single gate. No new target.
- Follow-up (ariadne): the machine-wide full-suite lock (`Makefile.workflow`, #319) is taken at parse time, so the preflight runs only after the lock wait. A base-layer pre-lock hook would let a sandboxed run fail before queueing. Not built here (base layer).

## Revisions

### 2026-10-11 — close review round 1
- Plan item 3 overstated the preflight ("cannot be mistaken"). The review sandbox also refuses exec of `ps` (`TestPublicPairCommandFamiliesIgnoreCouchStore`: `fork/exec /bin/ps: operation not permitted`), which socket+PTY probes missed. Delta: the preflight now probes socket, PTY and `exec ps`; it reports a missing python3 separately; the atlas and script header name the probed capabilities and say an unprobed refusal still fails as an ordinary test.
- A pane published without an endpoint now gets its own message ("no restore socket … host refused it") instead of the generic "finish and close it before reopening", which a reopen on the same host could not satisfy.
- Makefile comment notes that `make -k test` runs past the preflight; the lesson now covers the child-env pin as well as `TestMain`.

