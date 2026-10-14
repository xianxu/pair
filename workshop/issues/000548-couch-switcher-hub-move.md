---
id: 000548
status: open
deps: [pair#546]
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'bc36ed35cbf90e1172e460dcf468e15c5d726ad1' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T00:00:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# couch: move slots between hubs from the switcher

## Problem

#546 provides CLI-first hub moves. The operator may later want the same action directly in the switcher; it was explicitly cut from #546 to keep its initial scope small.

## Spec

Add a non-hub row action to choose an active target hub and optionally persist the move (default ephemeral), invoking #546’s existing move operation and authority model. Preserve pending/failure feedback and current ownership until acknowledgment. Do not introduce a second ownership or persistence path.

## Done when

- The switcher can move an eligible slot to an active hub through the existing operation.
- Persistence is optional and off by default; errors retain the current projection and offer retry.
- Hub rows cannot be moved; stale completions cannot override newer ownership.
- Reducer and integration tests cover success, failure and optional persistence.

## Plan

- [ ]

## Log

### 2026-10-14

Filed on ops:0 instruction while approving #546, receipt 676cf232-af54-4067-b179-874215cf0fba. Depends on #546 landing; no implementation now.
