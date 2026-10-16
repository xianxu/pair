---
id: 000613
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: medium
---

# test-review flake: review.lua branch mismatch refuses a write in recorded suites

## Problem

`make test-review` intermittently fails in recorded suites with `review.lua:881: review: branch mismatch; write refused`, raised from a BufWritePre autocommand when the test drives `nvim --server ... --remote-expr 'luaeval(vim.cmd("silent write!"))'` (CalledProcessError exit 2; test `test_positive_control_retains_wanted_fresh_and_request_order`, ObservationFixtureDiagnosticsTest). It did not reproduce on rerun either time.

Sightings (2026-10-16, pair:2, `sdlc local-check run --name suite`, each in a fresh detached worktree):
- #605 branch at 78d6bf93: log `.git/sdlc/test-stats/failed/960e87ecd1ecb38b79d24b6f51a3d99e.log` (lines ~721-748); rerun green.
- #595 branch at c1a5e8b9: log `.git/sdlc/test-stats/failed/c91173c6cd31485579f1133bf945547b.log` (lines ~723-763); the next run failed elsewhere.

Both runs were under heavy machine load (load average ~25-30, several slots running suites). The detached worktree has no branch, so the review buffer's branch check may race a git query, or read ambient state.
