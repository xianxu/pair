---
id: 000624
status: codecomplete
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T20:02:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 3.44
tracker:
    version: 1
    completion:
        token: close-b83b7ae3455b
        repository: github.com/xianxu/pair
        reviewed_head: 8e88e3629694568666a4b6a5141c5801c65e2935
        evidence_commit: 27cf285cb135c51558dd25b0c7406d554fe67169
---

# Something rebuilds pair:0's bin/couch during other slots' work, deleting it for seconds (sends fail)

## Problem

On 2026-10-16, `~/workspace/pair/bin/couch` (pair:0, the machine's installed Couch CLI that every slot and the TL use)
**disappeared for several seconds, twice**, and came back rebuilt at the same revision (9fc035e7, vcs.modified=false):
around 15:40 and at 16:04–16:06. During the gap every `couch --send-to` failed with "no such file or directory".
Nobody asked for a pair:0 rebuild. At 16:04:17 pair:4 (Qoder) started `sdlc milestone-close --issue 583 --milestone M1`
from its slot; the 16:06 rebuild followed. Hypothesis: a review prepare step (pair#549-era "prepare runtime assets
before boundary reviews") or a dev alias builds into pair:0's `bin/` instead of the worktree's own. The same mechanism
would clobber a staged branch build on pair:0.
