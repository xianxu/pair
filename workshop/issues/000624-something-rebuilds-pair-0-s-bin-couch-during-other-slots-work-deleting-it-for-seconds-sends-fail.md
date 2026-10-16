---
id: 000624
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'd0b1f86105147d1b34743338e91876bdaddb372d' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

- Find what writes pair:0's `bin/` outside an explicit `make build` / `sdlc stage` in pair:0 (fs_usage or a guard).
- A slot's build, test or review preparation writes only inside its own worktree. pair:0's `bin/` changes only via an
  explicit build or stage there.
- Builds that replace a shared binary replace it atomically (write to a temp file, then rename), never delete first.

## Done when

- A test (or guard) shows a milestone-close / review prepare in a slot worktree leaves pair:0's `bin/` untouched.
- `make build` in pair:0 replaces binaries atomically (no window where `bin/couch` is missing).

## Plan

- [ ]

## Log

### 2026-10-16
