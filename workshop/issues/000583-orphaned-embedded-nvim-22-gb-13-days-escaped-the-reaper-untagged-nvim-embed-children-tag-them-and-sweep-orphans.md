---
id: 000583
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: 'b021eaa2904d3113cb4d659d03fdea950a99cd78' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T10:00:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
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

Durable plan: `workshop/plans/000583-orphaned-embedded-nvim-22-gb-13-days-escaped-the-reaper-untagged-nvim-embed-children-tag-them-and-sweep-orphans-plan.md`
(two chunks = the two milestones below).

- [ ] M1 — every pair-spawned embed carries an identity and is reaped by exact identity
  - [ ] One closed kind set: `artifactpath.NvimKinds()` = {draft, scrollback, review}, with a drift pin over
    every inline list (`paths.go:559`, `lifecycle.go:195,297`, `osruntime.go:947,973`, `gc.go:253`)
  - [ ] Pure `ClassifyNvimReap(row)` — kill only when pid, kernel identity, the `nvim` short name, an exact
    `--embed` argv field and the expected artifact path all agree, and the owner is dead or ppid is 1
  - [ ] The `ProcessTable` seam gains `Name` and a targeted `Command(pid)`; `ReapNvim` / `ReapNvimContext`
    signal through its identity fence. **`pkillF` and both `pkill -9 -f` patterns are deleted**
    (`osruntime.go:938,956-957`, `lifecycle_os.go:49-58`), so the reap path holds no subprocess seam
  - [ ] The changelog viewer gets an **artifact-keyed** pid record — `<changelog-base>.nvim-pid`, a sibling
    of its existing `.openlock` — because (tag, kind) is not unique for per-(agent, session) viewers and
    neither the filename nor argv splits into a tag unambiguously
  - [ ] The sweep decides per pid from one snapshot over three evidence sources: tag-keyed sidecars,
    tag-parseable argv, and changelog `.nvim-pid`/`.openlock` pairs (owner liveness = the opener pid)
  - [ ] A clean exit removes its own sidecar (VimLeave in all four lua configs)
  - [ ] Flag-gated live conformance test: spawn a tagged embed, kill its owner, assert the reaper removed it
  - [ ] Atlas + `atlas/storage-retention-io.md` inventory updated (`TestManagedIOInventoryAnchors` is the guard)
- [ ] M2 — untagged orphans are reported by couch and reaped only on confirmation
  - [ ] Prove `procutil.Identity` is a wall clock; add `ParseIdentityTime`
  - [ ] `procutil.Cwd` via `proc_pidinfo(PROC_PIDVNODEPATHINFO)` in the existing cgo libproc block (+ live conformance)
  - [ ] Extract `UnattributedCandidate` from #587's `ClassifyUnattributed` so both sweeps share one rule
    (`TestResourceClassifyUnproven` stays green unchanged)
  - [ ] Pure `ClassifyOrphanNvim` / `ManagedRoots` / `WithinManagedRoot` (managed roots come from
    `WorkspaceIdentity`; containment is positive evidence, so an incomplete inventory narrows the sweep)
  - [ ] `ResourcePolicy.OrphanNvimAge` (default 6h, `COUCH_RESOURCE_ORPHAN_NVIM_AGE`); sampler probe gated on
    ppid 1 + name nvim + age ≥ threshold so the cwd cost stays inside #587's <0.2% budget
  - [ ] Health line: `couch --resources` section + `--json`; one hub alert per episode on the existing
    `resource_episode`/`sendGenerated` seam (#574 is still open, so its notice is not available yet)
  - [ ] `couch --reap-orphans --confirm`: CLI refusal without confirm, server-side `confirmation-required`,
    re-classify from a fresh sample, SIGTERM→wait→SIGKILL each identity-fenced via `ProcOps`
  - [ ] Cost re-measured at max cardinality, docs updated, live check, full suite, close

## Log

### 2026-10-15

### 2026-10-16 — claimed in pair:4, root cause identified

Claimed and branched (`sdlc claim`, `sdlc start-plan --issue 583`). Durable plan written via
`superpowers-writing-plans`: `workshop/plans/000583-orphan-nvim-reaper-plan.md`.

**Spawner identified (Done-when #2).** Two classes, not one:

1. *pair-spawned but untagged or unreaped.* `nvim --embed` in a bare argv is nvim's own TUI/embed fork
   (documented at `nvim/init.lua:16-23`): the TUI process re-execs an embedded server, and when the pane dies the
   server is reparented to launchd. pair's reaper only recognises embeds whose argv carries a slot tag
   (`TagFromNvimEmbedArgv`), and its kind lists are drift-prone: `gc.go:253` knows {draft, scrollback, review} while
   `osruntime.go:947,973`, `lifecycle.go:195,297` and `paths.go:559` know only {draft, scrollback}, and nothing
   anywhere knows `changelog` — the changelog viewer passes no `PAIR_NVIM_PID_FILE` (`opener/run.go:211-217`) and
   `nvim/changelog.lua` has no pidfile block. So a review or changelog embed survives its owner.
2. *not pair-spawned.* A bare interactive `nvim` typed into a pane has no tag pair could ever have written.

**Live evidence that shaped the safety pivot.** Right now this machine has bare `nvim --embed` rows *with live
parents* (pids 38555, 46819, 51229), and pid 38555's cwd is `/Users/xianxu/workspace/worktree/ariadne-slot2/ariadne`
— the 22 GB orphan's exact cwd. So neither name, nor cwd, nor age discriminates an orphan from a healthy embed;
**ppid 1 is the pivot**, and every kill must be identity-fenced (pid + kernel start stamp) because a pid recycles.
Consequence: the two name-based `pkill -9 -f` calls (`pkillF`, `osruntime.go:938,956-957`, and the pair at
`lifecycle_os.go:49-58`) cannot be repaired — they are deleted, per the dispatch's "never by name alone".

**Design decisions (ARCH markers in the plan).** Reuse the existing `nvim-pid-<tag>-<kind>` sidecar channel rather
than inventing an env probe (PPID comes free from the same one-read `kern.proc.all` table). Reuse
`session_reap.go`'s identity fence and `couchcore.ProcOps`/`FakeProcOps` instead of new kill machinery. Share #587's
unattributed predicate by extracting it *inside* couchcore (launcher cannot import couchcore — cycle). Add
`proc_pidinfo` to the existing cgo libproc block rather than an `lsof` subprocess per row, and gate it behind
ppid/name/age so the sampler cost stays under #587's approved <0.2%.

**Four points raised with the hub in the plan post:** the Done-when #2 reading (pair owns tagging + reaping what it
spawns and detecting + reporting the rest; it cannot make nvim's forked embed die with its TUI), reporting through
#587's alert seam because #574 is still open, a 6h default orphan age, and the verb name
`couch --reap-orphans --confirm`. None of them blocks M1, so M1 proceeds per the dispatch ("post a short plan … then
build"); a hub reply lands as a scope revision here.

### 2026-10-16 — plan-quality gate round 1: changelog identity redesigned (PQ-1)

`sdlc change-code --issue 583` recorded 1 Important + 3 Minor findings
(`workshop/plans/000583-…-plan-gate.md`). PQ-1 was correct and I verified each claim in the tree:
`ChangelogArtifactSet.Log` is `base + ".md"` (`paths.go:765`, my plan said `.log`); `componentPattern`
admits `-` in tag, agent and session (`paths.go:14`) with the session optional (`paths.go:738-749`); and a
changelog viewer is per (tag, agent, session), so one `nvim-pid-<tag>-changelog` sidecar collides.

Both remedies the finding offered were measured and rejected. Keying the sidecar per agent/session still
cannot recover the tag from the filename. Matching argv against a tag's expected paths crosses tags: a
`changelog-work-*` glob also claims `work-x`'s file, so quitting tag `work` could kill a live `work-x`
viewer — unacceptable in a kill path.

The changelog surface therefore leaves the kind set and gets an **artifact-keyed** record:
`<changelog-base>.nvim-pid`, a sibling of the `.openlock` the opener already writes per
(tag, agent, session) (`opener/run.go:180-189`). Unique by construction, needs no parsing, and owner
liveness comes from the openlock's opener pid rather than from a tag — a mis-read errs toward skipping.
`NvimKinds()` is `{draft, scrollback, review}`, which is still the whole drift fix (the reaper and the
sweep were missing `review`). PQ-2: `TagLive` → `OwnerLive`, explicit quit passes `false`. PQ-3: test
tasks now carry one strategy line each. PQ-4: the seam is stated — kills move onto `ProcessTable.Signal`,
so the reap path holds no subprocess seam and `pkill` is unreachable, not merely unasserted. Plan
`## Revisions` records the delta; task order moved (changelog record is now Task 4, before the sweep).

**Discovery filed separately:** `cmd/internal/runtimebundle/assets/runtime/files/nvim/` is a generated
mirror that is stale against `nvim/init.lua` (missing the `fs_watch.lua` dofile and the focus-pad
refactor) and `nvim/review.lua`, and no Makefile target regenerates it. Out of scope here — regenerating
would import unrelated drift into a kill-path diff — so M1 edits only `nvim/`, which is what the runtime
loads (`opener/runtime.go:163-165` runs `PairHome + "/nvim/<config>.lua"`).
