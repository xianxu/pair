---
id: 000631
status: codecomplete
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T01:16:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 0.72
tracker:
    version: 1
    completion:
        token: close-2a377f0443c3
        repository: github.com/xianxu/pair
        reviewed_head: 2e1ab24607e87b232bcb6846081e5dcce09acad3
        evidence_commit: 7fcba1f9f11bd578e8872f46822e26ac8261d41d
---

# Peer promotion holds broker lock during wrapper release RPC

## Problem

During #629 investigation, Broker.promote was found calling the wrapper endpoint's Release RPC while holding the shared broker mutex on its error path. A slow Release can delay independent recipients' Send and Status calls until the RPC timeout.

This is a separate static finding. No retained evidence ties this lock contention to the 2026-10-16 17:37–18:22 hub outage; #629's confirmed mechanism is serial pre-paste expiry.
