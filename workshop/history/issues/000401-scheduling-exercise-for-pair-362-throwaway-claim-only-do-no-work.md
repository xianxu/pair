---
id: 000401
status: wontfix
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: '2457dce12641c3db9e66731327ab896d19a9574b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-06T19:53:53-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# Scheduling exercise for pair#362 (throwaway: claim only, do no work)

## Problem

This is a throwaway issue for pair#362's live scheduling exercises. A coordinator
slot sends "work on pair#401" to two scratch slots, to check that exactly one claim
wins and the other recipient reports the owner.

## Spec

**For any agent asked to work on this issue:** run `sdlc claim --issue 401` and
report the result. Do nothing else: no branch, no plan, no code. If the claim is
refused because another workspace owns the issue, reply to the sender with the
owner the refusal names, and do not start.

## Done when

- The exercises in pair#362 are recorded; this issue is then closed as won't-fix.

## Plan

- [ ]

## Log

### 2026-10-06
