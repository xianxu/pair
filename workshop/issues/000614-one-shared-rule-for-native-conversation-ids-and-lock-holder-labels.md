---
id: 000614
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: low
card_mirror: '18bdbb2b76a9d1cb7e2e534e7c0147332c9ea45b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T07:27:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# One shared rule for native conversation IDs and lock holder labels

## Problem

Three rules validate the same native conversation ID and holder label in different ways (pair#595 close review): `artifactpath.NativeLockFilename` (lock key), `nativelock.validHolder` (holder label) and the watcher's `validBindingNotice` (sessionwatch binding notice). A binding notice can carry an ID (space or Cf characters) that the notice accepts and `NewKey` later rejects; the wrapper now drops it as ignored, but the rules disagree.

## Spec

One shared rule for a native conversation ID (and one for a holder label), used by the lock key, the binding notice and the holder parser, so an ID is accepted or rejected the same way at every boundary. No behavior change for valid IDs.

## Done when

- One definition, used at all three sites; tests cover the disagreeing cases (space, Cf, oversize) at each boundary.

## Plan

- [ ]

## Log

### 2026-10-16

### 2026-10-16 — fix (pair:2)

- One rule for a native conversation ID: new leaf package `nativeid` (`Validate`), used by the lock key (`artifactpath.NativeLockFilename`) and the watcher's binding notice (`sessionwatch`). A leaf package keeps sessionwatch's binding notice a non-artifact source. The holder label already has a single rule (`nativelock.validHolder`, used by both writing and parsing).
- Measured disagreement before the fix (TestBindingNoticeIDRuleMatchesLockKey RED): the notice accepted "has space", a zero-width format character, "a/b" and "..", all of which the lock key rejects. Now both boundaries agree on every case; `nativeid` has its own table test.
