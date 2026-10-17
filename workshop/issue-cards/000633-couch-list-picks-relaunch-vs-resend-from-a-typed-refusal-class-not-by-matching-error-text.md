---
id: 000633
status: done
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T07:29:55-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 0.89
tracker:
    version: 1
    completion:
        token: close-f3dcc2b25c2b
        repository: github.com/xianxu/pair
        reviewed_head: b9071aba1f20096e3a37e08b35e77e52d82b2e2f
        evidence_commit: c187617de0f0f6cbb4199d71fe83baa27056d8cc
        landed_commit: d98b3289a765a79a41c561c5c010ae152587d4a1
---

# couch --list picks relaunch vs resend from a typed refusal class, not by matching error text

## Problem

`couch --list` shows a refused slot's receive state with one action (pair#618). `receiveStateText`
(`cmd/internal/couchcmd/list_refusals.go`) picks "relaunch" versus "send again" by matching five phrases ("does not
own", "conversation changed", "launch registration", "not positively listed", "no retained checkout") against the
refusal's free-text reason. Those phrases live in error strings in `message_service.go` and `registry.go`, so
rewording an error silently flips the advice from relaunch to resend. This is the second finding in the
duplicated-request-classification family (pair#618 close review, BR-14). The rule: classify a result once, at its
source, into a typed value, and have consumers read that value.
