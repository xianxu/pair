---
id: 000324
status: open
deps: []
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
---

# couch-zellij-conformance: native zellij CPR timeout on macOS runner

## Problem

`couch-zellij-conformance` fails on almost every run: 94 of the last 100 at
filing time, with the only recent pass on 2026-09-15 (run 34941751536). The
workflow triggers on most couch/terminal paths, so nearly every couch PR gets a
failure email. Every sampled failure (2026-09-19 through 2026-09-25) is the same:

```
TestNativeConsoleWrapperZellij/direct-zellij-baseline
  terminal_native_test.go:166: timed out waiting for wrapped native process and CPR
TestNativeConsoleWrapperZellij/wrapped-zellij   (same)
```

`wrapper.log` and `receipts` never appear in the evidence dir, so the child never
ran and zellij never answered the cursor-position report (CPR) within 15s.

Evidence that this is the runner environment, not couch code:
- The **direct-zellij-baseline** case fails too, and it does not involve pair's
  wrapper.
- Locally, with the same pinned zellij 0.45.1, that phase completes in about 1.6s.
  (The local run then stops on a missing `@xterm/headless` test oracle module,
  which CI installs.)
- The passing (09-15) and failing (09-25) runs used the same runner image
  (Hosted Compute Agent 20260828.587).

## Spec

Find out why zellij on the macOS runner does not start the child and answer CPR,
then either fix the harness or bound the check so it stops red-flagging PRs:

- Capture zellij's own startup evidence on the runner (its log dir, stderr, and
  the raw parent/child byte prefixes already printed) for the direct baseline,
  and compare it with a local run. Candidates: first-run zellij setup on a fresh
  HOME (config, layout or plugin download prompts), missing terminfo, TTY sizing,
  or a slower cold start than 15s.
- Fix the root cause in the harness (e.g. pre-seed zellij config, or wait on a
  readiness signal rather than a fixed 15s). Do not just raise the timeout
  unless the evidence shows a slow but healthy start.
- Until it is green, consider making the job non-blocking on PRs
  (schedule/dispatch only, or `continue-on-error`), so real regressions in its
  other checks are not buried under a known red. That is a stopgap with an
  owner (this issue), not the fix.

## Done when

- `couch-zellij-conformance` passes on a PR and on the weekly schedule, with
  `TestNativeConsoleWrapperZellij` green in both baseline and wrapped modes.
- The root cause is recorded in `## Log` with the runner evidence that proves it.
- Any stopgap (non-blocking job) is removed in the same issue.

## Plan

- [ ]

## Log

### 2026-09-24

- Filed from the #321 ship session. Diagnosis via `gh run list/view` across
  runs 36088997227, 36066248730, 35966937282, 35914082673, 35453653409, plus a
  local run of `TestNativeConsoleWrapperZellij` with `PAIR_LIVE_COUCH_NATIVE=1`.
