---
id: 000726
status: working
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T07:54:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis

## Problem

After the 10-20 07:49 Couch restart (supervisor on pair:0 staged f3c597f8, which includes #714 and #721), slot
lifecycle operations from the hub CLI end in a client-side timeout with no diagnosis:

- `couch --resume pair:6` → "context deadline exceeded … outcome uncertain" (4 times since 10-19 23:00, across
  three supervisor builds). `--resume-candidates pair:6` says: "Previous screen unavailable: lstat
  …/scrollback-1-pair-59-codex.raw: no such file", "native transcript incomplete: session inventory read exceeds
  limit", "Candidate scan incomplete", yet still offers "Resume from disk (confirmed)". `--reboot pair:6 --confirm`
  also timed out (10-19 23:05).
- `couch --relaunch parley.nvim:4 --confirm` (idle, load ~16) → "context deadline exceeded"; the slot was NOT
  relaunched (wrapper Start still 10-19 20:12).

The CLI's deadline expires while the supervisor-side operation is presumably still working (or wedged), and the user
gets "verify with couch --recover-plan-from-sdlc", which itself fails to resolve the fleet from a non-git cwd.
