---
id: 000726
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5ee473e18eeb3cf55e84f0203c08e407aef661cc' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

- Find where the lifecycle operation spends its time (pair:6: likely the native inventory read that "exceeds limit";
  parley.nvim:4: unknown) and bound it inside the supervisor, so the CLI gets a typed outcome (refused with reason,
  or running with an operation id to poll) instead of a client deadline.
- `--resume-candidates` must not offer "Resume from disk (confirmed)" when the resume path will fail on the same
  oversized inventory; either the resume path tolerates it (the binding is established) or the menu says why not.
- `--recover-plan-from-sdlc` must work when run from a non-repo cwd (or say which directory to run it from).

## Done when

- A fixture with an oversized native inventory: resume of an established binding completes or refuses with a reason
  within the CLI budget (test). A relaunch of an idle slot completes within the budget (test with a stateful fake).
- pair:6 resumes (or refuses with a reason) live after the fix.

## Plan

- [ ]

## Log

### 2026-10-20
