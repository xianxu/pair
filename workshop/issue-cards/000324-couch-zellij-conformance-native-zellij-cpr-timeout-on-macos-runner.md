---
id: 000324
status: open
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
github_issue:
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
