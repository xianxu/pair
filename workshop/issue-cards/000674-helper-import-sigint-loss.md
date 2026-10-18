---
id: 000674
status: codecomplete
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T12:30:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.06
tracker:
    version: 1
    completion:
        token: close-3147bdd7e287
        repository: github.com/xianxu/pair
        reviewed_head: 902e3b20a057a77c6375d270a655cdb285d971bb
        evidence_commit: 984f0d98b04677e053ece4fe90279a7de55fb779
---

# Startup SIGINT swallowed by importlib weakref callback before helper handler

## Problem

The pair#650 integration suite on aa8176700b0e50b230c72eeaf190c9b284342810 failed in LifetimeTests.test_cancellation_during_startup (SIGINT, delay 0.02 s) despite the pair#598 fixture normalization being present. The helper did not exit within the existing 3 s bound; captured stderr reports KeyboardInterrupt ignored in importlib._bootstrap module-lock weakref callback. The caller SIGINT was default_int_handler. This differs from the inherited SIG_IGN cause fixed in #598. No causal link to slug gating is shown: the helper and lifetime test files are unchanged by #650.
