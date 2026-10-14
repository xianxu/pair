---
id: 000551
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'aea372223acc999409c7944adf860eedd6c0d4a9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T08:51:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# locks: close raw diagnostic and notification descriptors on exec

## Problem

The #545 lock-open sweep found raw descriptor opens without O_CLOEXEC in
cmd/internal/diagnosticlog/writer.go openRegular and
cmd/internal/notifytransport/address.go lockNamespace (and readPID in the same file).
Go os.NewFile does not add the atomic close-on-exec flag. These descriptors may
therefore survive unrelated execs. Both lock paths explicitly LOCK_UN, so this
is separate descriptor hygiene, not #545's pre-exec creation-lease cause.

## Spec

Add atomic close-on-exec on the raw opens in these two modules, sweeping sibling
raw opens in the same boundary. Preserve current validation and lock semantics.
No broader creation-lease or retry changes. TL requested this separate follow-up
in receipt cc82ffc3-c4f5-4c1c-9e9a-5b2fb14f5340.

## Done when

- Raw lock/binding opens in the two modules use O_CLOEXEC atomically.
- Targeted tests verify FD_CLOEXEC and existing lock/publication behavior.
- Full suite completes and the small change passes SDLC close/landing.

## Plan

- [ ] Inspect the raw open family, add regression tests, fix flags, verify and land.

## Log

### 2026-10-14

Filed from #545 investigation; independent follow-up, not started.
