---
id: 000614
status: done
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: low
started: 2026-10-16T07:27:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.04
tracker:
    version: 1
    completion:
        token: close-d117b6e53bf1
        repository: github.com/xianxu/pair
        reviewed_head: 61066dc88570e5b010925faf1ef02fcefc028231
        evidence_commit: aac5cfdc3a8cc25e9f22ba5c741ac0e7735dd59f
        landed_commit: 768c2ed403baed67b8a35045e6d39798a51771ab
---

# One shared rule for native conversation IDs and lock holder labels

## Problem

Three rules validate the same native conversation ID and holder label in different ways (pair#595 close review): `artifactpath.NativeLockFilename` (lock key), `nativelock.validHolder` (holder label) and the watcher's `validBindingNotice` (sessionwatch binding notice). A binding notice can carry an ID (space or Cf characters) that the notice accepts and `NewKey` later rejects; the wrapper now drops it as ignored, but the rules disagree.
