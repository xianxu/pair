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
  - [x] One closed kind set: `artifactpath.NvimKinds()` = {draft, scrollback, review}, with a drift pin over
    every inline list (`paths.go:559`, `lifecycle.go:195,297`, `osruntime.go:947,973`, `gc.go:253`)
  - [x] Pure `ClassifyNvimReap(row)` — kill only when pid, kernel identity, the `nvim` short name, an exact
    `--embed` argv field and the expected artifact path all agree, and the owner is dead or ppid is 1
  - [x] The `ProcessTable` seam gains `Name` and a targeted `Command(pid)`; `ReapNvim` / `ReapNvimContext`
    signal through its identity fence. **`pkillF` and both `pkill -9 -f` patterns are deleted**
    (`osruntime.go:938,956-957`, `lifecycle_os.go:49-58`), so the reap path holds no subprocess seam
  - [x] The changelog viewer gets an **artifact-keyed** pid record — `<changelog-base>.nvim-pid`, a sibling
    of its existing `.openlock` — because (tag, kind) is not unique for per-(agent, session) viewers and
    neither the filename nor argv splits into a tag unambiguously
  - [x] The sweep decides per pid from one snapshot over three evidence sources: tag-keyed sidecars,
    tag-parseable argv, and changelog `.nvim-pid`/`.openlock` pairs (owner liveness = the opener pid)
  - [x] A clean exit removes its own sidecar (VimLeave in all four lua configs), pinned by
    `make test-nvim-pidfile` — 20 assertions over all four configs: published, removed on `qa!`, kept after a
    SIGKILL, and never removed when a later embed republished the same record
  - [x] Env-gated live conformance (`PAIR_LIVE_NVIM_REAP=1`): a real tagged embed is reaped on the kernel's own
    account (SIGKILL read back through `wait`), a live tag's embed is never decided, and a non-nvim process
    named by a stale record is never signalled. No owner is killed, because a measured host fact makes that
    impossible — see the plan's `## Revisions` (d): closing an embed's stdin ends it in ~31 ms and SIGKILLing
    the `nvim FILE` TUI parent removes its child, so owner death is supplied the way the sweep reads it (the
    tag absent from `liveTags`)
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

### 2026-10-16 — M1 Tasks 1-3 landed: one kind set, one decision, one identity-fenced reap

Commits `bf84b5edb` (kind set), `8280a2084` (pure decision), `a01db5a31` (the reaper). Both reap paths —
`OSRuntime.ReapNvim` and `OSRuntime.ReapNvimContext` — now run one core (`reapNvimSidecars`) that takes a
single snapshot for pid/ppid/identity/name, reads that pid's argv, decides with `ClassifyNvimReap`, and
re-reads the kernel start stamp immediately before `SIGKILL`. `pkillF` and both `pkill -9 -f` patterns are
gone, and with them the `exec.Command("kill", "-9", pid)` calls: the reap path holds no subprocess seam, so
nothing in it can reach a process by name.

**Attribution is three-way, and each way was measured rather than assumed.** `ps -eo pid,ppid,command` on
this host confirms `nvim FILE` forks a TUI parent plus an `nvim --embed` server child, and lua's
`vim.fn.getpid()` returns the *embed* pid, so every sidecar names an embed. The plan's single
"expected artifact path" predicate only fits the draft:

- *draft* argv carries its exact scope path (`nvim --embed -u <pairHome>/nvim/init.lua <scopeDir>/draft-<tag>.md`).
- *scrollback* is launched as `sh -c 'exec nvim -u "$1" "$2"' pair-viewer <lua> <ansi>`
  (`opener/runtime.go:164`), so argv carries `<scopeDir>/scrollback-<tag>-<agent>.ansi`. The obvious
  predicate — `Paths.ScrollbackPrefix()` — mis-attributes: `scrollback-work-` also matches
  `scrollback-work-2-claude.ansi`, so quitting tag `work` would kill tag `work-2`'s viewer. Attribution goes
  through `ScopePaths.TagFromNvimEmbedArgv`, the codebase's one parser, whose last-dash rule cuts the agent
  correctly; `TestReapNvimSidecarsAttributesScrollbackByTagNotPrefix` pins all four owner/ansi pairings.
- *review* argv carries no pair path at all (`nvim --embed -u <pairHome>/nvim/review.lua <any file in the
  user's tree>`), and `<pairHome>` is **checkout-relative** — a live embed outlives the checkout that
  spawned it, and this reaper runs from a different worktree than the sessions it reaps. So review matches
  the asset's **base name** (`review.lua`, from `reviewcmd.ReviewScript`), never its path.

`NvimReapRow` accordingly carries `ExpectedArgPath` / `ExpectArgvTag`+`ArgvTag` / `ExpectedAssetBase`; a
kind states as much as its argv allows, and a kind that cannot be attributed is **skipped** (`argv serves
another tag`, `argv runs another config`) rather than killed on `nvim --embed` alone.
`TestNvimSidecarsGiveEveryKindArgvEvidence` is the drift guard: a kind added to `NvimKinds()` without a
predicate fails it.

**Record rule.** Evidence that is merely missing keeps the sidecar for the next pass (owner live, identity
unreadable, argv unreadable); every *decided* outcome removes it — kill, malformed record, pid gone,
recycled identity, not-ours. A zombie embed (identity readable, argv not) therefore leaves its record to
storage GC, which already treats the family as collectable
(`GCClassifications["nvim-pid"] = GCCollectable`).

**SIGKILL only, no TERM escalation** — every caller has already established that the owner is gone or is
quitting, so no session is left to flush and quit is time-bounded. `outcome.Killed` is `Signal(...) == nil`,
so a pid that races away between the identity re-read and the signal is not reported as reaped; Task 5's
report reads that field.

**One derivation replaced two pinned spellings.** Task 1 pinned the quit path's inline spelling to
`editorPathsOf`; Task 3 deleted the inline spelling instead, because two spellings pinned to each other is
the exact shape that let `review` drift out of both. `editorPathsOf` →
`nvimSidecarsFor(paths, paths.Tag(), paths.ScopeDir())` is now the only place the kind→attribution mapping
exists. `TestNvimReapPathsDeriveEveryKindFromOneSource` went with it, redundant against
`TestNvimSidecarsForDerivesFromTheClosedKindSet` + `TestQuitPathAndReaperShareEditorPaths`.
Artifact-coverage consequence: `lifecycle.go` drops family `nvim-pid` / binding `scoped-nvim-pid`,
`osruntime.go` drops `draft` / `scoped-draft` / `scoped-nvim-pids`, `nvim_reap_os.go` is a ResolvedConsumer
with no bindings (it consumes members on a `Paths` handed to it and calls no resolver of its own), and the
pure `nvim_reap.go` joins `NonArtifactSources`. Task 2 had left `nvim_reap.go` out of the inventory
entirely — `TestProductionArtifactReferencesAreExactlyClassified` caught it as NEW, and `a01db5a31` pays it.

**Discovery, not folded in:** `KillTitlePollerContext` (`lifecycle_os.go:62-75`) still runs `kill <pid>`
straight from the title pidfile with no identity fence — the recycled-pid hazard just closed for the
editors, still open for the title poller. #583's contract is embedded nvim, so this is independent work;
raised with the hub rather than filed from this slot.

### 2026-10-16 — M1 Tasks 4-5 landed: the changelog viewer is identified, and the sweep decides per pid

Commits `33c6ab5a3` (changelog pid record) and `3d4df5ed6` (the sweep).

**Task 4 — the changelog viewer leaves the kind set.** PQ-1 measured that `(tag, kind)` cannot identify a
changelog embed: the viewers are per-(agent, session), and neither `changelog-work-codex-sid` nor its argv
splits into a tag unambiguously (`componentPattern` admits `-` in all three components and the session is
optional). So the record is **artifact-keyed** instead — `<changelog-base>.nvim-pid`, a sibling of the
`.openlock` the opener already writes (`opener/run.go:211-216` passes it as `PAIR_NVIM_PID_FILE`;
`nvim/changelog.lua` publishes `vim.fn.getpid()` at `VimEnter`, the `init.lua`/`scrollback.lua` spelling).
`NvimKinds()` stays `{draft, scrollback, review}`.

The family's filenames now have one construction: `changelogSiblings(base)` builds all seven from one
`changelogNvimPIDSuffix` const, and `ScopePaths.ChangelogSetFromNvimPID` recognizes a globbed record back
into the same set — so the constructor and the recognizer cannot drift on the spelling, which is the
failure mode Task 1's kind-set pin exists to prevent. The recognizer `filepath.Clean`s its input and
requires `filepath.Dir(clean) == p.scopeDir` before rebuilding from the base name alone;
`TestScopePathsRecognizeAChangelogNvimPIDRecord` pins the glob string, round-trips a sessionless and a
sessioned base, and rejects an outside-scope path, a `.md`, a kind sidecar, and a `../` traversal.
Retention follows (`gc.go`'s suffix list and the exact-match set both name `.nvim-pid`), so the record is
collectable like its siblings rather than a new unmanaged file.

**Task 5 — the sweep decides per pid, from one snapshot.** `SweepOrphanNvim` was a 44-line
`cands[tag] → ReapNvim(tag)` fan-out: a tag was the unit, so a tag that was live protected every embed
whose argv merely mentioned it, and one that was dead condemned every record under it. It is now three
evidence sources over **one** `Snapshot`, each producing `nvimSidecar` rows that the existing
`ClassifyNvimReap` + identity fence decide:

1. `nvim-pid-<tag>-<kind>` sidecars for tags absent from `liveTags` (`NvimPIDGlobs()` →
   `TagFromNvimPID` → `ResolveScoped` → `nvimSidecarsFor`, Task 3's one derivation).
2. rows named `nvim` whose argv carries a scope path — `TagFromNvimEmbedArgv` names the tag, and the row
   states the predicate its argv *already satisfied* (`ExpectedArgPath` for draft, `ExpectArgvTag` for
   scrollback) rather than a bare `nvim --embed` match, so the decision re-checks evidence instead of
   trusting the source that produced it.
3. `changelog-*.nvim-pid` records, whose `OwnerLive` is `openerLive` over the paired `.openlock`: an
   absent lock means the owner exited **cleanly** (`opener/run.go:189`'s `defer rt.Remove(openlock)`, and
   `RunViewer` is a held child, so the opener lives as long as its viewer) — absent is the reap case, not
   a skip.

`Owner` for source 3 is `set.Log`, not the base: the base is a filename fragment naming no process, while
`set.Log` is the artifact the viewer holds open and whose `.openlock` it pairs with.

**Two records naming one pid are decided once, signalled once.** `reapNvimRows` keeps `decided` as
`pid → reason`; the first record to name a pid drives the decision and the only `Signal`, and a later
record for that pid follows the same reason off disk without emitting a second outcome. They are two
pieces of evidence about one process — a stale sidecar beside a current one — and signalling twice would
have made the report claim a reap it did not perform. This was found by two failing launcher tests, not
by design: the first cut skipped later targets outright and left their records behind. Pinned by
"two sidecars naming the same embed" and "one embed named by both a sidecar and its argv".

Tests live in `nvim_reap_os_test.go` (15 cases in
`TestSweepOrphanNvimDecidesEachPidOnItsOwnEvidence`, over a real temp scope directory and a fake
`ProcessTable`; `fakeProcessTable` gained `snapshotErr`, and
`TestSweepOrphanNvimRefusesABrokenSnapshot` proves a failed snapshot is refused rather than read as
empty). The kept cases are the ones that matter: a live tag's draft embed, a live tag's scrollback embed
reached through a *dead* tag's stale record, a changelog viewer whose opener still holds the lock
(asserted with TUI pid 600 / embed pid 4242 so ppid ≠ 1), an argv-only embed of a live tag, and a bare
`nvim --embed` with no scope path at all — which is M2's classifier, not this sweep's business, and so
leaves the signal list empty.

**Artifact inventory:** enumerating the globs moved from `osruntime.go` into `nvim_reap_os.go`, so
`osruntime.go` loses family `nvim-pid` and binding `selected-nvim-pid-globs`, and `nvim_reap_os.go`
claims nothing — binding witnessing is per-file and one-directional, and a `Paths`/`ScopePaths` received
as a *parameter* is never tracked by `resolverResults`, so a consumer that reaches members through a
handed-down value cannot witness a binding there. Its header comment now says so.

Verified: `go test ./cmd/internal/launcher/` ok 55.1s, `./cmd/internal/artifactpath/` ok 8.8s,
`./cmd/internal/opener/` ok 0.35s, `./cmd/internal/storagegc/` ok 23.6s, `make test-changelog` 3×PASS,
`go build ./...` clean, `gofmt -l` clean for the touched packages.

### 2026-10-16 — M1 Tasks 6-7 landed: a clean exit drops its own record, and the reaper meets real nvim

Commits `0804a647b` (Task 6) and `74c0c832d` (Task 7).

**Task 6 — VimLeave in all four configs.** `init.lua`, `scrollback.lua`, `changelog.lua` and `review.lua` now
remove `$PAIR_NVIM_PID_FILE` on a clean exit, so a quit session does not hand the next sweep a pid that
stopped being ours. The removal is guarded by *"only a record that still names this process"*: `review.lua`
republishes per activation (its embed is restarted and rolled back), and a restarted embed for the same tag
writes the same path first, so an unguarded delete would leave the survivor unnamed. A SIGKILLed embed never
reaches VimLeave and keeps its record — that record is the reaper's only name for the process.

The four copies stay four copies (the shared `nvim/pidfile.lua` stays rejected), because the new
`tests/nvim-pidfile-lifecycle-test.sh` runs **all four configs** through the same four properties — published
at startup, removed on `qa!`, kept after a SIGKILL, never removed when a later embed republished it — 20
assertions, wired as `make test-nvim-pidfile` and into the aggregate `test:`. A copy that drifts now fails its
own case, which is what pinning the copies to each other could never prove. Two mechanics worth recording:
`-c 'qa!'` runs *before* `VimEnter`, so the autocmd under test would never register and the test would pass by
never publishing — the script quits through an ack file (`vim.wait` on `$PAIR_PIDFILE_ACK` inside a deferred
VimEnter callback, which fires both VimEnter and VimLeave); and the configs refuse to boot without a `pair`
binary on PATH (`retention.lua`: "cannot protect editor storage: … ENOENT … (cmd): 'pair'"), so the script
builds one via `tests/lib/pair-test-binary.sh`. `--remote-send ':qa!<CR>'` and `--remote-expr` were probed and
rejected (the former does not quit a headless init.lua boot, the latter only with the client already exit 2).

**Task 7 — live conformance, and three host facts that rewrote it.**
`cmd/internal/launcher/nvim_reap_live_test.go`, gated on `PAIR_LIVE_NVIM_REAP=1` (this package's live idiom,
not a flag: `launcher` gates on env + `t.Skip`, only `wrapcmd` uses flags). The plan said "kill the owner (the
spawning parent)"; measured, that leaves nothing to reap:

- closing an `nvim --embed --headless`'s stdin pipe ends it in **~31 ms**, exit status 1;
- SIGKILLing a `nvim FILE` TUI parent removes its `nvim --embed` child within 3 s (`nvim --embed draft.md`
  pid 69810, ppid 69809 → gone);
- the `sh -c '"$@" & exit 0'` disposable owner this test first tried never published a sidecar at all — the
  backgrounded nvim does not survive its shell, and two probes (held pipe, `/dev/null`, fifo) confirmed it.

So an orphan is a process whose stdin something long-lived still holds, and the test supplies exactly that: it
holds the pipe open, and states owner death the way the sweep reads it — **the tag absent from `liveTags`**,
which is the sweep's whole authorization for a tag kind (it never consults ppid there). The dropped case (an
orphaned embed of a *live* tag) became unreachable for the same reason; a live tag is never decided, which is
the second case, and `reapReasonReparented` stays pinned by the fake table's ppid-1 rows where it *is*
reachable.

Two things only a live run can catch, both caught:

1. A process that exited but was not waited on is a **zombie whose argv the kernel will not report**. The first
   gated run hit exactly that and the sweep said `argv unknown`, killed nothing and **kept the record** —
   real-kernel proof of the record rule (missing evidence is not a decision) and of the classifier's restraint.
2. `waitSignal` blocked in `cmd.Wait()` for the whole 5-minute go-test timeout when no kill landed. It is now
   deadline-bounded at 10 s. A mutation check (`if row.OwnerLive || row.Owner != ""` in `ClassifyNvimReap`)
   proves the assertions have teeth — a clean four-message failure in 10.04 s instead of a panic trace — and
   was reverted (`git status` clean apart from the new test).

Gated output on this machine (`PAIR_LIVE_NVIM_REAP=1 go test ./cmd/internal/launcher/ -run TestLiveNvimReap
-count=1 -v`):

```
=== RUN   TestLiveNvimReapRemovesADeadTagsEmbed
--- PASS: TestLiveNvimReapRemovesADeadTagsEmbed (0.03s)
=== RUN   TestLiveNvimReapLeavesALiveTagsEmbedAlone
--- PASS: TestLiveNvimReapLeavesALiveTagsEmbedAlone (0.03s)
=== RUN   TestLiveNvimReapLeavesANonNvimProcessAlone
--- PASS: TestLiveNvimReapLeavesANonNvimProcessAlone (0.01s)
PASS
ok  	github.com/xianxu/pair/cmd/internal/launcher	0.249s
```

Ungated, all three SKIP with "set PAIR_LIVE_NVIM_REAP=1 to reap a real embedded nvim" and the package passes
in 0.180 s, so `make test` never spawns or kills an editor. The live run also asserts what a real embed's argv
carries before it reaps (`--embed` and the exact draft path, read back through `ProcessTable.Command`), so a
later "not reaped" cannot be a silent argv mismatch — and it does that on a host with ~30 real pair embeds
running, none of which any sweep here can reach, because every kill is decided from evidence under one temp
scope directory.

Verified: `make runtimebundle-generate test-nvim-pidfile` ok 15.3s (20/20), `make test-review test-changelog
test-lifecycle-command test-retention` exit 0, `make test-lua` exit 0,
`go test ./cmd/internal/launcher/ -count=1` ok 31.1s, gated live run ok 0.249s, `go vet ./cmd/internal/launcher/`
clean, `gofmt -l cmd/internal/launcher/` clean.
