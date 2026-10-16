---
id: 000595
status: working
created: 2026-10-15
updated: 2026-10-16
estimate_hours: 2.15
github_issue:
started: 2026-10-15T19:45:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
tracker:
    version: 1
---

# Replace the native-owner forensics (#556/#584) with a wrapper-held flock per conversation; one-time cutover; delete the scan and reservation store

## Problem

pair#556 (with #528's observer and #584's fixes) guards "no agent conversation has two live writers" by forensics over
historical state: a scan of every OS process, argv, wrapper readiness files, parent chains and birth times, launch
ledgers decoded for conversation IDs, `lsof` of held transcripts, observed twice and compared, plus a reservation
store. About 1,700 non-test lines and ~45 refusal reasons; #584 needed eight design deltas to make it work on this
machine (unreadable root processes, 18 of 21 older wrappers without agent_identity, unbound launches, a 21 MB ledger, a
15 s decode, a 5 s budget under load). The operator and TL agree it's over-engineered (ops QA
`2026-10-15-37-whats-in-pair-0.md`; fresh-context review). #556's plan never weighed a lock that dies with its owner.
