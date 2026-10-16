---
id: 000583
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '756b6075455a7e8b4293cd148aa90d74d0a291bb' # card fields mirrored from issue-cards; edit via sdlc
---

# Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans

## Problem

2026-10-15: an orphaned `nvim --embed` (pid 15395, started 2026-10-02, cwd `worktree/ariadne-slot2/ariadne`, parent
`launchd`) had grown to **22 GB** over 13 days, contributing to 63 GB of swap. pair already reaps leftover embedded
nvims (`launcher/osruntime.go` ~L945–970: it scans for `nvim … --embed` and reaps those whose slot tag isn't live),
but it only recognises processes whose argv carries a slot tag (`scope.TagFromNvimEmbedArgv`). This one's argv was bare
`nvim --embed`, so it was invisible to the reaper. Whatever spawned it (a review pane, a GUI client or a test harness)
exited without closing it.

## Spec

- **Every pair-spawned embedded nvim carries an identity** the reaper can match (a tag in argv or env, plus the spawner
  pid), so it's reaped when its owner dies.
- **An orphan sweep** for untagged ones: `nvim --embed` with parent `launchd` (pid 1), cwd inside a pair-managed
  worktree, alive longer than N hours → reported by `couch` (a health line and the hub lifecycle notice from #574), and
  reaped with confirmation. Never touch an nvim with a live parent.
- Find the spawner of the untagged one (review pane? a test harness's `jobstart({…'--embed'…})`?) and make it tag and
  close its child.

## Done when

- A test spawns a tagged embedded nvim, kills its owner, and the reaper removes it; an untagged orphan in a worktree is
  reported (and reaped on confirmation); an nvim with a live parent is never touched.
- The spawner of bare `nvim --embed` children is identified and fixed.

## Plan

- [ ]

## Log

### 2026-10-15
