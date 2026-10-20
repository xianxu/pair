---
id: 000622
status: done
created: 2026-10-16
updated: 2026-10-20
estimate_hours: 3.55725
github_issue:
started: 2026-10-16T17:03:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-aa80e0ea3853
        repository: github.com/xianxu/pair
        reviewed_head: 90384d10782ab96507d6f735d4527f69b28becb1
        evidence_commit: ceb634941ff46c45ad8452ada7436ab7a143b2a4
        landed_commit: 787735804b56aa99515529f9346545d8d85459a2
---

# Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not

## Problem

Profiled by pair:2 for pair#618 (2026-10-16, macOS `sample` of the Couch server, 5 s): the Couch server burns about
one core continuously, and ~500 of its busy samples are terminal emulation: `ptychild.(*Child).ingest` →
`internal/terminal` + charmbracelet's vt `ansiparser`. Couch parses the full output stream of every hosted session
client (~27 `pair --couch-session-v1 resume` children) in a vt emulator, so every spinner redraw on every slot costs
Couch CPU all the time, whether or not anyone is looking at that slot.
