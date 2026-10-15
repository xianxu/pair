---
id: 000218
status: codecomplete
created: 2026-09-08
updated: 2026-10-14
estimate_hours: 5.86
github_issue:
started: 2026-10-14T15:00:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 9.40
tracker:
    version: 1
    completion:
        token: close-7d4d3ea223ab
        repository: github.com/xianxu/pair
        reviewed_head: 36e8111e58ba6d7a4c9669bc083157cfcc8582f0
        evidence_commit: d0374e70fb689f559bbed3271d2981737b83b748
---

# couch thread startup takes 8.85s and nothing has measured where it goes

## Problem

`#215` raised couch's registration deadline from 5s to 15s against a **measured
8.85s** startup for `pair resume <tag> --layout3`, and made name assignment O(1)
zellij probes. Both were right, and neither explains the 8.85s. A deadline
chosen as 1.7x an unexplained number is a guess with a measurement attached.

The operator's felt symptom, after `#215` unblocked couch: *"startup works now.
I guess it's just slow?"*

### What the number is NOT — measured, so these are closed

| suspect | measured | how |
|---|---|---|
| nvim + pair's `init.lua` | **117 ms** | `nvim --startuptime`, full breakdown; slowest single step is markdown syntax at 3.5ms |
| `zellij list-sessions` | **43 ms** at 26 sessions | wall-clock, 5 runs, 42-45ms |
| one `ProbeSessionName` | **~45 ms**, creates no session | `zellij --session X action list-clients` against absent names |
| name assignment | **8 probes**, flat as the index grows | `#215`, was 52 |
| 6 orphaned zellij servers | **0.0% CPU** each | `ps`; they cost RSS and list-sessions entries, not time |

So roughly 0.5s of the 8.85s is accounted for. **~8.3s is unexplained.**

### What is known about the shape

- **The two waits are different.** The cold path (`awaitThreadRegistration`)
  polls the thread-claim file. The resume path (`awaitResumeRegistration`) waits
  for the zellij **session to be live** — so only resume pays for zellij bring-up,
  layout, three panes, nvim and the agent. The claim file for one long-lived
  thread was last written four days before it was next resumed, which is how we
  know the resume path never touches it.
- `pair` spawns `zellij` from **eight** call sites (`osruntime.go` x6,
  `session_quiescence.go` x2). Nobody has counted how many times a startup
  actually calls it.
