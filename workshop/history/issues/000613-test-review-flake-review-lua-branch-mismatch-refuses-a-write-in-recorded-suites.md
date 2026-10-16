---
id: 000613
status: done
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: medium
card_mirror: '6954ce5e374f7eff40ace496afdefffd2a4f2104' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T06:41:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.37
---

# test-review flake: review.lua branch mismatch refuses a write in recorded suites

## Problem

`make test-review` intermittently fails in recorded suites with `review.lua:881: review: branch mismatch; write refused`, raised from a BufWritePre autocommand when the test drives `nvim --server ... --remote-expr 'luaeval(vim.cmd("silent write!"))'` (CalledProcessError exit 2; test `test_positive_control_retains_wanted_fresh_and_request_order`, ObservationFixtureDiagnosticsTest). It did not reproduce on rerun either time.

Sightings (2026-10-16, pair:2, `sdlc local-check run --name suite`, each in a fresh detached worktree):
- #605 branch at 78d6bf93: log `.git/sdlc/test-stats/failed/960e87ecd1ecb38b79d24b6f51a3d99e.log` (lines ~721-748); rerun green.
- #595 branch at c1a5e8b9: log `.git/sdlc/test-stats/failed/c91173c6cd31485579f1133bf945547b.log` (lines ~723-763); the next run failed elsewhere.

Both runs were under heavy machine load (load average ~25-30, several slots running suites). The detached worktree has no branch, so the review buffer's branch check may race a git query, or read ambient state.

## Spec

Find why review.lua sees a branch mismatch in the suite's detached checkout and make the test deterministic: it must not depend on load or on ambient git state.

## Done when

- The cause is measured, not guessed.
- The test passes repeatedly under concurrent load, with a regression for the cause.

## Plan

- [ ]

## Log

### 2026-10-16
- 2026-10-16: closed — Recorded suite green on 0a487724. Cause measured from the second sighting own observation report: guard resolve returned invalid "cannot read review history: exec: WaitDelay expired before I/O complete" from reviewcmd GitContext WaitDelay 50ms. CPU load alone did not reproduce (0/20 at load 18 and at load 58). Fix: WaitDelay 1s; same class in two couchcmd zellij queries; BufWritePre refusal names the guard reason. TestOSGitContextToleratesBrieflyHeldStdout RED on main with the exact error, GREEN with the fix; reviewcmd, couchcmd and make test-review green. no-atlas: bugfix, no new surface.; review verdict: SHIP

### 2026-10-16 — measured cause and fix (pair:2)

- Not CPU load alone: tests/review-observation-test.sh passed 0/20 failures at load 18 and 0/20 at load 58 (eight niced burners, removed after); the resolver chain the test uses (python wrapper exec real pair) measured median 0.10 s, max 0.20 s.
- The second sighting's own observation report (review observation failure JSON in the suite log) records the guard's synchronous resolve (call 39) returning status invalid with "cannot read review history: exec: WaitDelay expired before I/O complete"; every earlier resolve was resolved. reviewcmd.OSRuntime.GitContext ran git with WaitDelay 50 ms: after git exits, the stdout copy (lagging under load, or held briefly by a descendant git starts) got only 50 ms before exec closed the pipes and failed the complete read. The review pane's BufWritePre then refused the write with a misleading "branch mismatch".
- Fix: GitContext WaitDelay 1 s (still bounded, within the guard's 2.5 s); the same class fixed in couchcmd's zellij list-panes and list-clients queries (100 ms to 1 s, context still bounds). BufWritePre now says "review: write refused: <guard reason>", so a resolver failure is no longer reported as a branch change.
- Regression: TestOSGitContextToleratesBrieflyHeldStdout uses a git alias whose shell leaves a 0.3 s background holder of stdout; RED on main with the exact sighting error, GREEN with the fix.
