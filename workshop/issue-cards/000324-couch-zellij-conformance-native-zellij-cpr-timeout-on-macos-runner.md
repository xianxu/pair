---
id: 000324
status: codecomplete
created: 2026-09-24
updated: 2026-10-10
estimate_hours:
github_issue:
started: 2026-10-10T17:22:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 1.44
tracker:
    version: 1
    completion:
        token: close-076eea886a2a
        repository: github.com/xianxu/pair
        reviewed_head: c17bc6b33cc376eb4da79cc1960476c870a88d5a
        evidence_commit: 530793a2c06f892f8349dc031ff6594917d7a14a
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
