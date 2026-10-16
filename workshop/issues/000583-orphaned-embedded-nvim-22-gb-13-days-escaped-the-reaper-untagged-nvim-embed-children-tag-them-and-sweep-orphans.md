---
id: 000583
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours: 8.1
card_mirror: '03c264c1c8644d6eaa2aa644402b620cf408fa73' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T10:00:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=0.8  impl=0.1
item: cross-cutting-refactor  design=0.3  impl=0.22
item: smaller-go-module       design=0.15 impl=0.34
item: smaller-go-module       design=0.3  impl=0.34
item: lua-neovim              design=0.3  impl=0.3
item: real-api-discovery      design=0.0  impl=0.2
item: atlas-docs              design=0.15 impl=0.15
item: milestone-review        design=0.2  impl=0.29
item: milestone-review        design=0.1  impl=0.15
item: smaller-go-module       design=0.35 impl=0.4
item: greenfield-go-module    design=0.2  impl=0.3
item: smaller-go-module       design=0.35 impl=0.5
item: greenfield-go-module    design=0.4  impl=0.45
item: real-api-discovery      design=0.0  impl=0.2
design-buffer: 0.15
total: 8.1
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Derivation, item by item against the durable plan's 17 tasks (impl values are already the v3.1 40% scale):

| item | plan tasks | why these hours |
|---|---|---|
| `issue-spec` 0.8/0.1 | brainstorm, the 540-line durable plan, three plan-quality rounds | design already spent; this is why every other item's design takes the ×0.2 spec-quality discount |
| `cross-cutting-refactor` 0.3/0.22 | T1 (kind set across six consumers + drift pin), T11 (extract `UnattributedCandidate`) | both are behaviour-preserving multi-file rewires with a pin, low end of 0.2–1 / 0.2–0.5 |
| `smaller-go-module` 0.15/0.34 | T2 (`ClassifyNvimReap` + table test), T3 (`ProcessTable` seam, reaper rewrite, delete `pkillF`) | the plan states the decision rules and the seam verbatim; mid of 0.2–0.5 each |
| `smaller-go-module` 0.3/0.34 | T4 (changelog `.nvim-pid` record: artifactpath + gc + opener), T5 (three-source sweep) | T5 is the highest-risk logic in M1, so its design is above the discount floor |
| `lua-neovim` 0.3/0.3 | T4's pidfile block, T6's VimLeave in four configs + the headless lifecycle test | three-line copies of `init.lua:24-34`; the primitive's 1–3 design range collapses because the pattern exists |
| `real-api-discovery` 0.0/0.2 | T7 flag-gated live conformance against a real `nvim` fork | nvim is the external dependency whose fork/argv behaviour must be observed, not assumed |
| `atlas-docs` 0.15/0.15 | T8 (storage inventory + architecture), T17 (couch atlas, README, env knob) | high end of 0.05–0.2 twice: the inventory anchors are grep-checked |
| `milestone-review` 0.2/0.29 | M1 and M2 boundary reviews | two `milestone-close` rounds at 0.1/0.14 each, each priced as one clean pass |
| `milestone-review` 0.1/0.15 | one extra M1 round | M1 deletes `pkill -9 -f` from a kill path and adds the identity fence — the shape of diff that draws Important findings, and the gate allows up to 3 rounds |
| `smaller-go-module` 0.35/0.4 | T9 (`ParseIdentityTime` + live proof), T10 (`proc_pidinfo` cwd in the existing cgo block) | cgo is novel-but-bounded: it mirrors `resource_identity`/`resource_usage`, so familiarity stays 1.0, and the hours absorb the build/sandbox friction of live process proofs needing unsandboxed re-runs |
| `greenfield-go-module` 0.2/0.3 | T12 (`ClassifyOrphanNvim`, `ManagedRoots`, `WithinManagedRoot`) | a new file, but the plan fixes every signature; low end of 0.5–2 design |
| `smaller-go-module` 0.35/0.5 | T13 (policy knob + gated sampler probe), T14 (health line + projection), T15 (episode alert) | three existing-seam extensions, mid of 0.2–0.5 impl before scaling |
| `greenfield-go-module` 0.4/0.45 | T16 (`couch --reap-orphans --confirm` across cli, ops registry, dispatch) | a new operator verb spanning three layers plus a server-side second gate and per-row SIGTERM→wait→SIGKILL; the house confirmation pattern discounts design, not impl |
| `real-api-discovery` 0.0/0.2 | T17's cost re-measurement at maximum cardinality against #587's <0.2% budget, the full-suite runs, and the live check | measurement wall-clock the first pass left unpriced; `atlas-docs` covered only the docs half of T17 |

Σdesign 3.6 × 1.15 (thorough plan doc) + Σimpl 3.94 × 1.0 = 8.08 → **8.1 h**.

Revised from the first pass's 7.4 h after `change-code`'s estimate-quality judge returned INFO naming four
priced-out gaps (the second M1 review round, T17's unpriced measurement/suite/live-check wall-clock, T16's
thin impl, T9/T10's cgo friction). Its fifth point — the ×1.15 buffer also multiplying the already-spent
`issue-spec` design — is left as the model specifies; the buffer applies to Σdesign, not to design still ahead.

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

**Corrected later the same day — no mirror discovery to file.** I first wrote that
`cmd/internal/runtimebundle/assets/runtime/files/nvim/` is a tracked mirror gone stale against
`nvim/init.lua`, to be filed separately. That was wrong, and the correction is measured: `.gitignore:133`
ignores `/cmd/internal/runtimebundle/assets/` and `git ls-files` returns nothing for it, so it is
untracked build output; `Makefile.local:263-266` defines `runtimebundle-generate`, a prerequisite of
`build`, `test` and `$(RUNTIMEBUNDLE_ASSETS)`; and generating into a temp dir reproduces `nvim/init.lua`
byte-for-byte (`diff` exit 0), so the generator copies lua verbatim and a differing local copy only means
this checkout has not been built since those lua edits. Consequence for M1: the lua edits in Tasks 4 and 6
reach the embedded bundle on the next build with no mirror diff to commit, and the runtime path is
unchanged (`opener/runtime.go:163-165` runs `PairHome + "/nvim/<config>.lua"`).

### 2026-10-16 — change-code admitted the plan; estimate revised to 8.1 h

`sdlc change-code --issue 583` passed on the third plan-quality round (`VERDICT: CLEAN`, "no open blocking
findings after 3 round(s)"). Round 2 had disposed PQ-1..PQ-4 and raised one Minor advisory, PQ-5
(`unstated-decision-input`): Task 5's changelog source did not state the verdict when `<base>.nvim-pid`
exists but its `.openlock` sibling is gone. Folded in — an absent openlock means the opener exited cleanly
(`opener/run.go:189`), which *is* the root-cause orphan case, so absent reaps rather than skips; and the
absent/present/malformed rule is now stated for every evidence source, which the round-3 judge accepted as
closing the family.

`estimate-quality` returned INFO (advisory, gate passed). It confirmed the derivation is honest — all 12
items mapped to named plan tasks, arithmetic checked — but named four gaps it would have priced: the second
round a kill-path boundary review usually takes, T17's unpriced cost re-measurement / suite runs / live
check, T16's thin impl for a verb spanning three layers, and T9/T10's cgo and sandbox friction. Revised to
**8.1 h** (Σdesign 3.6 × 1.15 + Σimpl 3.94 = 8.08) and recorded on the card with
`sdlc issue set-estimate --issue 583 --hours 8.1`. Its fifth point — the buffer also multiplying the
already-spent `issue-spec` design — was left alone: v3.1 applies the buffer to Σdesign, and carving spent
design out of the multiplicand is a model change, not an issue-level choice.
