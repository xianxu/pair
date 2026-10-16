---
id: 000595
status: codecomplete
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
actual_hours: 3.23
tracker:
    version: 1
    completion:
        token: close-d3bc8645aaf5
        repository: github.com/xianxu/pair
        reviewed_head: c1a5e8b9c93b76608cd6507ae8618e38760d5458
        evidence_commit: 889006dd3a235d2b7e5b9a5d184537c200b03c69
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
