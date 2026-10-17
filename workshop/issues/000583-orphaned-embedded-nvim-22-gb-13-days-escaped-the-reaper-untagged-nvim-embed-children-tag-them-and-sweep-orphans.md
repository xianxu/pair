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

- [x] M1 — every pair-spawned embed carries an identity and is reaped by exact identity
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
  - [x] Atlas updated (`architecture.md`'s Alt+x reap, startup sweep, tag-only family list and sidecar row).
    `atlas/storage-retention-io.md` needed nothing: it inventories retention-IO anchors, and the pid records
    are written by lua `writefile` outside the guard, as they were before #583 — `TestManagedIOInventoryAnchors`
    passes untouched
- [ ] M2 — untagged orphans are reported by couch and reaped only on confirmation
  - [x] Prove `procutil.Identity` is a wall clock; add `StartTimeFromIdentity` (renamed from the draft's
    `ParseIdentityTime`: it does IO on linux, and its argument is a token, not a pid)
  - [x] `procutil.Cwd` via `proc_pidinfo(PROC_PIDVNODEPATHINFO)` in the existing cgo libproc block (+ live conformance)
  - [x] Extract `UnattributedCandidate` from #587's `ClassifyUnattributed` so both sweeps share one rule
    (`TestResourceClassifyUnproven` stays green unchanged)
  - [x] Pure `ClassifyOrphanNvim` / `ManagedRoots` / `WithinManagedRoot` (managed roots come from
    `WorkspaceIdentity`; containment is positive evidence, so an incomplete inventory narrows the sweep)
  - [x] `ResourcePolicy.OrphanNvimAge` (default 6h, `COUCH_RESOURCE_ORPHAN_NVIM_AGE`); sampler probe gated on
    ppid 1 + name nvim + age ≥ threshold so the cwd cost stays inside #587's <0.2% budget
  - [x] Health line: `couch --resources` section + `--json`; one hub alert per episode on the existing
    `resource_episode`/`sendGenerated` seam (#574 is still open, so its notice is not available yet)
  - [x] `couch --reap-orphans --confirm`: CLI refusal without confirm, the confirmation gate in
    `couchmessage.ValidateRequest` (so a hand-written request cannot reach the classification), re-classify
    from a fresh pass the sampler's own worker ran, then SIGTERM→wait→SIGKILL over the whole tree, each signal
    identity-fenced via `launcher.ProcessTable` — not `ProcOps`, which has no `Snapshot` and so cannot drive
    `Reaper`'s same-moment pid/parent/identity read
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
- 2026-10-16: closed M1 — M1 round 5: BR-7 fixed by binding review attribution into the pid record at publish, which is what the family rule demands. review.lua writes the served file as the record second line out of its own argv (last element of vim.v.argv); the reaper reads that line beside the pid via sidecarRecord and never reads review-target state; a record that binds no file (legacy one-line, or empty field) is "attribution unknown", which skips and KEEPS the record. reviewcmd.TargetFileOf is deleted - nothing in production reads the mutable state any more, so a mismatch can no longer land on the side that deletes. New rows: sweep and quit paths for a reparented embed after a retarget (KILL), a cross-tag live pane holding a dead tag pid (skip), a legacy one-line record (kept), an empty second line (kept). Evidence: mutation check - restoring BR-6 semantics (state read overriding the record line) reddens exactly three rows, reap "a review embed whose pane retargeted after it was spawned", reap "a live tag review embed that reparented after a retarget", sweep "a review embed that reparented after its pane retargeted", and nothing else; then reverted and green. go test ./cmd/internal/launcher/ ./cmd/internal/reviewcmd/ ./cmd/internal/artifactpath/ -count=1 ok; make test-nvim-pidfile 21/21 across the four lua configs including the new "review: the record binds the file it serves"; PAIR_LIVE_NVIM_REAP=1 go test -run TestLiveNvimReap 3/3 PASS; whole go test ./cmd/internal/... -count=1 exit 0 with 0 FAIL lines; gofmt clean, go build ./... ok, go vet on the three touched packages clean. Prose: atlas/architecture.md review attribution now names the second line of the pid record and states that mutable state may corroborate but never be the subject; workshop/lessons.md gains the BR-7 rule (bind attribution at write time; check what a decided negative DELETES before calling a skip safe); the plan appends a round-4 revision and marks round-3 "residual gap, deliberate" as wrong and superseded. Residual, deliberate: two tags serving the same absolute file plus a pid wrap could still collide - strictly narrower than what BR-6 fired on (any retargeted pane); closing it needs a spawner-minted nonce in argv, a spawn-contract change, deferred out of M1.; review verdict: SHIP

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

### 2026-10-16 — M1 Task 8: the atlas, and a plan premise that did not survive reading the file

The plan's Task 8 named `atlas/storage-retention-io.md`'s "nvim-pid and changelog rows". Reading it first — as
the step itself asked — shows there is no nvim-pid row: that file inventories **retention-IO anchors**, the
paths that touch managed bytes under the retention guard, and its anchors are grep-checked by
`storagegc.TestManagedIOInventoryAnchors`. The pid records are not such a path. Lua writes them with
`writefile` outside the guard, exactly as it did before #583 (the changelog `.nvim-pid` follows its
`.openlock` sibling's precedent), and `storagegc` classifies both for collection. So nothing there was stale,
no row was invented to satisfy a plan sentence, and the guard passes untouched.

`atlas/architecture.md` is the only atlas file that mentions `nvim-pid`, and it needed four edits:

- **`:383` (cleanup on quit)** still described "a tag-scoped `pkill -f` fallback" — deleted in Task 3 — and
  two publishing configs. Now: three kinds (`artifactpath.NvimKinds()`), four configs, each removing its own
  record on a clean `VimLeave`, both the quit path and the sweep deciding through `ClassifyNvimReap` and
  signalling through `ProcessTable.Signal` behind the identity fence, so no reap can kill by name.
- **`:385` (startup orphan sweep)** still described collecting *candidate tags* and reaping *tags*. Now: the
  decision is per pid over one snapshot and three evidence sources, two records naming one pid are one
  decision and one signal, each candidate is attributed by the shape its kind allows (exact argv field /
  argv-tag attribution / pair asset base name), and a bare `nvim --embed` carrying none of them is invisible
  here by design — that untagged class is M2's report-and-confirm sweep.
- **`:1218`** the tag-only family list still said `nvim-pid-<tag>-{draft,scrollback}`; `NvimKinds()` and the
  rename/GC enumeration that iterates it (`gc.go`) say three, and `review` has been a kind since Task 1.
- **`:1256`** the sidecar row now names the three kinds, notes that the review record is resolved at its
  launch site (`reviewcmd/run.go`) rather than env-bound, states the "only a record that still names this
  process" removal guard, and carries the changelog's artifact-keyed `<base>.nvim-pid` with the reason
  (tag, kind) cannot serve it.

Verified: `go test ./cmd/internal/storagegc/ -run TestManagedIOInventoryAnchors -count=1` ok 0.176s,
`go test ./cmd/internal/artifactpath/ -count=1` ok 6.927s (its `TestArtifactDocsUseBindingsForCurrentConsumers`
is the other atlas grep guard, and the edited paragraphs touch neither its required nor its forbidden phrases).

### 2026-10-16 — M1 boundary review round 1: FIX-THEN-SHIP, and the closed pane was unreachable

`sdlc milestone-close --issue 583 --milestone M1` ran the boundary review at head `95db2c14b` and refused
to finalize: verdict **FIX-THEN-SHIP** with one open blocking finding
(`workshop/plans/000583-…-m1-review.md`).

**BR-1 (Important) was right, and the first fix it offered was the wrong one.** The rule "owner live but
reparented to launchd → kill" — Task 2's second killing row, Task 5's `OwnerLive = live[tag]` — could not
fire for a tag-keyed kind: both sweep sources dropped live tags and the quit path passes `false`, so
`reapReasonReparented` only ever reached changelog viewers and a hand-built test target. I drafted the
finding's other option (withdraw the claim) on the strength of Task 7's measurement that closing an
embed's stdin ends it in ~31 ms, then rejected my own draft: that measurement says an embed *usually* dies
with its parent, and this issue exists because one did not, for 13 days and 22 GB. The atlas already
carried the honest sentence — "the embed sometimes survives RPC-pipe EOF and gets reparented to launchd".
So the leak was real (a session lives for days after its editor pane closes) and the code changed instead
of the plan: `tagSidecars` (was `deadTagSidecars`) targets every tag the globs find with `live[tag]` as
`OwnerLive`, `nvimSidecarsFor` takes liveness from its caller, and `argvOnlyEmbeds` keeps live tags. The
guard did not move — ppid 1 plus the kind's full argv attribution plus the identity fence — and the cost
is near zero, because source 2 already read argv for every nvim row before it checked liveness. Two sweep
cases that had leaned on the ppid-1 `embed()` helper for a live tag now give it a live parent, so each
isolates what it names; two new cases pin one SIGKILL for a reparented live-tag embed (reached through its
sidecar, and through argv alone); `TestNvimSidecarsGiveEveryKindArgvEvidence` pins that a liveness a caller
states reaches every kind.

**BR-2 (Minor) fixed at its root, not as suggested.** `RunViewer` is a **held** child (`opener/runtime.go`
ends in `cmd.Wait()`) and the openlock goes by `defer` on its return, so PQ-5's "absent lock ⇒ dead owner"
stands; the defect was `opener/run.go` ignoring the `WriteFile` error, which left a viewer whose ownership
was never recorded reapable by any other session's sweep. `RunChangelog` now refuses to open one: exit 1,
cause named on stderr, no distiller launched, and no removal of a lock this opener never wrote. Pinned by
`TestRunChangelogRefusesAViewerItCannotClaim`, teeth proven by mutation (reverting to `_ = rt.WriteFile`
turns it RED on `code = 0, want 1`). The suggested grandparent-pid check would have added a second
ownership model beside the openlock; the scrollback lock at `run.go:109` stays unchecked on purpose, since
it guards re-entrancy, not ownership — a failed write there costs a duplicate read-only viewer.

**BR-3 (Minor) recorded** in the plan's `## Non-goals`: the quit path reads pid records only, so a second
scrollback viewer whose per-tag sidecar was overwritten waits for the next launch sweep. The deleted
`pkill -9 -f` pattern caught it, and everything else whose argv mentioned the path.

**BR-4 (Minor) half fixed, half disposed.** `byPID` is now built once per pass (`indexByPID`) and shared by
the openlock liveness check and the decisions. The second `Command(pid)` read stays, with the reason at the
read: `argvOnlyEmbeds` reads argv to *attribute*, `decideNvimSidecar` reads it to *authorize*, and the
authorizing read has to sit beside the identity fence — carrying an earlier read into the decision would
mix evidence from two moments, which is what the one-snapshot rule removed.

Verified: `go build ./...`; `go vet ./cmd/internal/launcher/ ./cmd/internal/opener/`;
`go test ./cmd/internal/launcher/ ./cmd/internal/opener/ -count=1` ok (30.7s / 0.2s);
`PAIR_LIVE_NVIM_REAP=1 go test ./cmd/internal/launcher/ -run TestLiveNvimReap -v -count=1` PASS 3/3
(0.176s), whose live-tag case now asserts exactly one decision, reason `owner live`, no kill, record
intact. Atlas `architecture.md`'s startup-sweep paragraph re-edited for the same reason as BR-1: its first
two evidence sources described dead tags only, and it now states the live-tag rule.

### 2026-10-16 — M1 boundary review round 2: BR-5, my own regression behind an inventory guard

Round 2 disposed all four round-1 findings as `addressed` and raised one new Critical. BR-5: the BR-2 fix
added `"pair-changelog-open: openlock: %v\n"` to `opener/run.go`, which `artifactpath`'s manifest classifies
as a `ResolvedConsumer` of the `changelog` family, and its
`TestProductionArtifactReferencesAreExactlyClassified` rejects a literal in such a file unless that exact
string is inventoried. The guard was red and my verification never ran it: I tested the two packages I
edited, and the guard lives in a third one that *reads* production sources rather than importing them.

The finding's suggested remedy was measured before being applied, and it does not work. Rewording the
message to `"pair-changelog-open: claim viewer: %v\n"` reproduces both violations, because the token that
fires is the `changelog` family token inside the command prefix — no tail wording escapes it, and every
other message this file prints is listed in the manifest for the same reason. Fixed the sanctioned way:
`"pair-changelog-open: write %s: %v\n"`, naming the record it could not write and matching
`reviewcmd/run.go`'s already-listed `"pair-review-definition: write %s: %v\n"`, plus its entry in the
file's existing changelog `goCallVocabularyValues` list. `TestRunChangelogRefusesAViewerItCannotClaim` now
asserts the refusal names that record path instead of a phrase, so a reword that drops the evidence turns
it red.

Rule added to `workshop/lessons.md` (#583 BR-5): verify a fix against every package's tests, including
source-parsing guards, and measure a reviewer's suggested remedy before adopting it.

Verified: `gofmt -l` clean on the three touched packages; the whole `go test ./cmd/internal/... -count=1`
passes with no failures (the run that had never been made); `make runtimebundle-generate
test-nvim-pidfile` 20/20 across the four lua configs.

### 2026-10-16 — M1 boundary review round 3: BR-6, attribution named the kind and not the owner

Round 3 disposed BR-5 as `addressed` (measured again in its own fenced checkout: the manifest entry is what
keeps `TestProductionArtifactReferencesAreExactlyClassified` green) and raised one new Important finding,
BR-6, family `kill-attribution-must-name-owner`. It is real, and it is worse than a gap in one kind: an
existing case in my own classifier table pinned the unsafe behaviour as intended ("review kind expects no
scope path" → kill).

The defect. `nvim_reap_os.go` gave the review kind `ExpectedAssetBase: "review.lua"` as its whole
attribution, and every tag's review pane runs that config. Pid records store no start stamp, so the
identity fence only catches a pid reuse that happens *after* the snapshot is taken. Put the two together:
tag A dies leaving `nvim-pid-A-review` behind, the kernel recycles that pid to tag B's live review embed,
and the next sweep reads owner A as dead, sees the `nvim` name, an exact `--embed` field and `review.lua`
in argv, and SIGKILLs B's pane. Every check passed and none of them said whose pane it was.

The fix is stated once, in the classifier, because that is where the rule belongs: **a recycled pid is safe
only for a kind whose argv names its owner.** `NvimReapRow` gained `ExpectArgvPath`, and
`ClassifyNvimReap` now requires at least one owner-naming form (`ExpectArgvPath` / `ExpectArgvTag`) to be
both stated and satisfied; a row stating neither returns the new reason `attribution unknown` and is never
killed. `ExpectedAssetBase` is demoted to corroboration — it still refuses `argv runs another config`, but
it names the kind, so it can no longer carry a kill by itself.

The review kind's owner-naming field is the file **its own tag's** `review-target-<tag>.json` names,
resolved in `decideOne` before the decision (`reviewAttribution`) and read by the new
`reviewcmd.TargetFileOf`, exported beside `targetJSON` and pinned to it by `TestTargetFileOf`. That state is
the only place recording the file: a review pane serves a path anywhere in the user's tree (`RunOpen` joins
`LogicalDir(file)` to its base), so its argv carries no scope path at all. A state that is gone, unreadable
or names no file leaves the field empty — missing evidence, so the candidate skips and keeps its record
(`reapKeepsRecord` gained `attribution unknown`), never a pass.

Two remedies the finding offered were measured and rejected. Storing a start stamp in the record:
`procutil.Identity` is a wall clock on darwin (`P_starttime`) but **ticks since boot** on linux, so
comparing one needs a per-OS parser and would either refuse every kill on linux or be inert there. Fencing
on the record's mtime measures the wrong quantity — `review.lua` republishes per activation, and a file's
mtime is not a process's start. Attribution through the owner's own state needs neither a new record format
nor OS-specific time parsing.

Tests. Five review cases replace the one that pinned the kill (attributed by its tag's file → kill; argv
serves another tag's document → `argv serves another path`; state names no file → `attribution unknown`; a
kind-wide asset name attributes no owner → `attribution unknown`; argv runs another config →
`argv runs another config`). The drop-one-predicate invariant gained an `attribution` mutation. At the reap
level the BR-6 shape is a case of its own, and at the sweep level too: A's record plus A's state, the pid
holding a pane that serves another file → no signal, and the stale record removed because it names a
process that is not this tag's. A sweep case with a state naming no file keeps the record.
`TestNvimSidecarsGiveEveryKindArgvEvidence`'s invariant is now "every kind carries owner-naming
attribution", which is the drift guard for a fifth kind arriving with a kind-wide predicate.

Round 3's two Minor ARCH-DRY notes are disposed in the plan's `## Revisions`, not dropped: the four lua
VimLeave copies stay copies (the lifecycle test runs all four through the same four properties, which
proves each correct rather than identical), and `gc.go`'s hand-listed changelog suffixes get a behavioural
guard — `TestGCChangelogSiblingsAreAllRecognized` fails if a suffix leaves `ownerCandidates`' list, teeth
proven by deleting `.nvim-pid` from it.

Verified: `gofmt -l` clean on the three touched packages; `go build ./...` and `go vet` clean;
`go test ./cmd/internal/launcher/ ./cmd/internal/reviewcmd/ ./cmd/internal/artifactpath/ -count=1` ok
(66.0s / 47.9s / 22.1s), with the two packages that gained tests after that run repeated in full
(13.7s / 9.2s). `PAIR_LIVE_NVIM_REAP=1 go test ./cmd/internal/launcher/ -run TestLiveNvimReap -count=1`
PASS 3/3 (0.305s). The whole `go test ./cmd/internal/... -count=1` (BR-5's rule) returned one failure,
`couchcore`'s `TestPublicWarmAttachCompletionNeedsListingNotLifecycleOwner`: it sets
`resumeRegistrationTimeout = 100ms` around a real `zellij list-sessions --short`, three other slots were
compiling with `-race` at the time, and it passes 3/3 on a re-run (`ok 1.885s`) with the real listing
answering in 0.14s — host load, not this diff. Teeth: the red was the new field (three `unknown field
ExpectArgvPath` sites), and the behaviour it replaces is documented by the case this fix *deleted* —
"review kind expects no scope path" asserted a kill on `nvim --embed` plus `review.lua` alone. Deleting
`.nvim-pid` from `ownerCandidates`' suffix list turns the new GC guard red on exactly that sibling
(reverted, `git diff` clean for `gc.go`).

### 2026-10-16 — M1 boundary review round 4: BR-7, my BR-6 fix reopened the leak it was fixing

Round 4 disposed BR-6 and raised one Critical in the same family (its 2nd finding, so the gate now names the
class, not the instance): *"attribution evidence must be bound to the pid at the moment the record is
written, and must not change for the pid's lifetime. Mutable session state may corroborate, but a mismatch
against it must keep the record, never drop it."*

BR-7 is right, and the plan's round-3 claim that the divergence was a safe deliberate residual was wrong on
both counts. Measured from source: `:PairReview` and an Alt+C activation of a live pane
(`nvim/review/restore_client.lua:137`) both rewrite `review-target-<tag>.json`, while the pane's argv keeps
naming the file it was spawned with — so the divergence is routine for any retargeted pane, not a one-pane
window. And the outcome is not a harmless skip: `reapReasonOtherPath` is a *decided negative*,
`reapKeepsRecord` drops the record on it, and review has no argv-only fallback (its argv carries no scope
path), so a retargeted pane's orphan loses the only name it ever had. The 22 GB class, reopened by the
previous round's fix.

Fixed the way the finding prescribes. `nvim/review.lua`'s publish now writes two lines: the pid, then
`vim.v.argv[#vim.v.argv]` — the file this embed was spawned with, out of its **own** argv, not the
activation's `file` parameter (which is exactly the value that drifts). `sidecarPID` → `sidecarRecord(raw)
(pid, served, ok)`; `nvimSidecar.ReviewState` → `ServedInRecord`, bound in `reapNvimRows` where the record is
read; `reviewAttribution` deleted, and `reviewcmd.TargetFileOf` + `TestTargetFileOf` with it. The launcher no
longer reads the review-target state at all (grep-verified: the only remaining `review-target` references in
the package are test fixtures). A record with no second line — a legacy one, or an empty argv tail — is
`attribution unknown` and is **kept**.

Why line 2 is sound where the state was not: it is written by the process it names, from that process's own
argv, which is fixed for its lifetime. A mismatch therefore proves the pid is not the process that wrote the
record, so dropping it is correct; a record that bound nothing proves nothing, so it stays. Residual,
strictly narrower than BR-6's: two tags serving the same absolute file plus a pid wrap onto the other's
record could still collide. Closing that needs a spawner-minted nonce in argv — a new spawn-shape contract,
not bought for a case that also needs a pid reuse.

Verified: `gofmt -l` clean; `go build ./...`; `go vet` + `go test ./cmd/internal/launcher/
./cmd/internal/reviewcmd/ ./cmd/internal/artifactpath/ -count=1` ok; the whole `go test ./cmd/internal/...
-count=1` (BR-5's rule); `make runtimebundle-generate test-nvim-pidfile` 21/21 across the four lua configs,
its new review assertion being that line 2 is byte-equal to the file the embed was launched with;
`PAIR_LIVE_NVIM_REAP=1 go test ./cmd/internal/launcher/ -run TestLiveNvimReap -count=1` PASS 3/3. Teeth
proven by mutation: restoring the state read (BR-6 semantics) reddens exactly the three retarget rows the
finding demanded — reap-level quit, reap-level live-tag reparent, sweep-level reparent — and nothing else;
reverted, green. The plan's round-3 "Residual gap, deliberate" paragraph is marked withdrawn rather than
deleted, so the false premise stays visible; `workshop/lessons.md` gains the BR-7 rule, including the half
that would have caught it before the judge did: check what a decided negative *deletes* before calling a
skip safe.

### 2026-10-16 — M2: the sweep classifies, reports, remembers and pages

**The classifier is pure, and measurement is what kept it that way.** The draft had `ClassifyOrphanNvim`
derive the process's age from its identity stamp; `procutil.StartTimeFromIdentity` switches on GOOS and
reads `/proc/stat` on every linux call, so that "pure" step would have done IO and its table test would
have needed a platform-shaped token. Instead the sampler — which already owns the IO and the sample
instant — derives the age from the stamp the process table already carries (no extra syscall) and hands
it over in a `ResourceOrphanProbe{Cwd, CwdKnown, Age, AgeKnown}`. A missing claim refuses the candidate;
it never widens it. `ClassifyOrphanNvim` is falsified one claim per row across 20 cases, including the
two that carry the safety claim: a cwd that merely *shares a string prefix* with a managed checkout
(`/fleet/pair-slot1/pair/cmd` vs `/fleet/pair`) is refused, and a nested checkout reports the most
specific root. Containment reuses `RelativeFamilyPath` — the fleet's existing segment-safe rule — rather
than a new prefix test. Mutation check: swapping in a string-prefix match and first-match-wins reddened
exactly those guards; reverted, green.

**The probe stays inside #587's measured budget.** A cwd read is paid only by a row that already proved
ppid 1 + name nvim + age ≥ `OrphanNvimAge` (default 6h, `COUCH_RESOURCE_ORPHAN_NVIM_AGE`, on the existing
1s..24h bound). Reading argv widened from "resident above the floor" to "old *or* above the floor",
because a small untagged orphan is still an orphan and the floor was only ever #587's reason to look.
`ResourceCost.CwdCalls` makes the new call attributable in the measurement instead of inferred from the
total, and it is printed in the sampler line beside the counters it joins.

**One row per process.** A swept nvim leaves the generic `Unattributed` section: the same pid under two
headings reads as two findings. `couch --resources` gains `Orphan nvim (untagged; reap needs --confirm):`
— the heading states the gate because those rows are what an operator confirms against — and prints
`none observed` when empty, since an absent section reads as "not checked". The wire projection gained a
sixth bounded detail section on the existing budget split, proven at maximum cardinality (8192 rows) with
an explicit `orphan nvim omitted` notice.

**The persisted ring keeps the offender.** Because the reduce suppresses a swept pid from
`v.Unattributed` and `ResourceFrame` stores exactly that slice, the 24h history would have stopped
recording the process it exists to record — a 22 GB untagged nvim sat in `frame.Unattributed` before this
issue and would have been nowhere after. `ResourceFrame` gained `OrphanNvim`, bounded at
`ResourceMaxProcesses` by a new `frameWithinBounds`. Additive at version 1: an upgrade reads old history
unchanged; a downgrade loses continuity into a fresh memory ring on the loud best-effort path
`resources_persist.go` already has — a cost `ResourceCost.CwdCalls` had already incurred, since the frame
persists `Cost`.

**One hub alert per episode, routed by the checkout.** The open design question was routing: every #587
effect carries its owning slot's workspace and the dispatcher resolves the hub from it, but an orphan's
slot is gone by definition, so a workspace-less effect would resolve to nothing and sit pending forever —
a silently dead alert, which is the failure mode this issue exists to end. The episode is addressed by the
managed checkout the sweep matched, taking that root's live slot's `WorkspaceIdentity` whole, so the
existing `hubBindingAt(owner(id))` path resolves it and there is still exactly one routing rule. The
metric is `orphan-nvim` (`couchcore.ResourceMetricOrphanNvim`), it requires `OrphanSamples` consecutive
passes so one racing probe cannot page the hub, and its episode clears when the pid is gone, a zombie, or
recycled into another generation — `nvim/` joined the process-keyed prefixes, now `processEpisodeKey`.
Body: `unattributed embedded nvim pid=… age=… in <root>; no live slot owns it. Inspect with couch
--resources; reap with couch --reap-orphans --confirm.` — every host-sourced string sanitized, because a
cwd is not trusted text and the body lands in the hub's terminal.
`ResourceEffect` gained `Age` and `Root` (omitempty, bound when the effect fires and never re-read, per
the BR-7 family rule), and `advance`'s `value float64` became a `ResourceEffect` so each metric carries
its own evidence rather than a float whose unit depends on the metric.
Proven end to end with nothing stubbed between the ends: `TestResourceAlertBrokerEndToEndFromTheSweep`
runs the real sampler over a real broker and asserts the delivered body, the silence on the first and
third samples, and that no `nvim/` episode survives the reap. Pre-existing gap noted, not fixed here:
nothing tests `newResourceAlertDispatcher`'s resolve — that needs a full `messageService` rig and belongs
to #587's seam.

**#574 note (as the plan records):** #574's hub lifecycle notice does not exist yet — its `## Plan` is
empty — so the orphan notice uses #587's alert seam. When #574 lands, this notice moves there; the
episode machinery and the effect ID stay, only the body's delivery changes.

Commits: `104279f61` classifier, `549eb3f07` sampler + policy, `f28f2d34f` health line + ring,
and this one for the alert. Evidence for each: `go build ./...`, `go vet`, targeted `go test -run`, and a
full `go test ./cmd/internal/couchcore/` (517s, exit 0) before the health-line commit.

### 2026-10-16 — M2 Task 16: `couch --reap-orphans --confirm`

**The plan's registry instruction was wrong, and its named fake cannot drive the reaper.** The draft said to
declare the verb in `couchcore/ops.go` with `ConfirmRequired` and route it through `operationdispatch.go` /
`slot_operations.go`. `TestDispatchTableIsIdenticalToTheDeclaredOperationSet` (`run_test.go:578`) audits
`Dispatch()` ≡ `couchcore.OperationNames()`, so declaring it obliges a dispatch entry — and the verb this one
is a sibling of, `--resources`, is declared nowhere: it is a `cliMessage` op parsed in `cli.go` and answered
by the supervisor over the broker socket. So `--reap-orphans` follows that shape (parse in `cli.go`, route in
`messages.go`, answer in `message_service.handle`) and the three registry files are untouched. The draft also
named `couchcore.FakeProcOps`, which implements `ProcOps` and has no `Snapshot`; `Reaper` plans from a
snapshot precisely because #399's BR-15 requires pid, parent and start identity to describe the same process
at the same moment, so no `ProcOps` fake can drive it. The reap is tested against a `launcher.ProcessTable`
fake instead. Both are recorded in the plan's `## Revisions`.

**Who classifies, and who kills.** The supervisor classifies: whether a process is unattributed is a fact
about the whole slot inventory, and under-claiming there means killing a process a live slot owns. The CLI
signals, on `peek`'s precedent — seconds of escalation must not sit on the console's serial queue, and
`couchmessage.AdmissionTimeout` is 2 s (`broker.go:13`), far short of a fresh pass plus a SIGTERM grace, so
the CLI holds its own 2-minute `orphanReapBudget`.

**The fresh pass belongs to the sampler's worker.** `resourceSampler`'s invariant is one worker owner with
handlers reading only the cache, and `pass` reads `s.state.Provenance`, so a throwaway second sampler was not
available. The worker now serves a `sweeps chan chan resourceSweep` inside `run`: a request hands it a reply
channel and the worker performs the pass, which *is* the loop's next pass rather than a second one. `pass`
returns whether it completed, because a pass the lifetime cut short leaves an older view cached and reporting
that as fresh is how a reap would kill from a minute-old decision; `sweep` turns `fresh == false` into an
error, and a sampler with no worker fails loudly instead of hanging to the deadline.
`TestReapOrphansSupervisorClassifiesFresh` proves the freshness on a sampler whose `Cadence` is an hour: the
cached report holds one process, the confirmed reap answers two, so the answer cannot have come from the
cache.

**The confirmation is the protocol's, not the handler's.** `couchmessage.ValidateRequest` refuses an
unconfirmed `reap-orphans` and refuses any second field, which is stronger than a registry flag would have
been: a hand-written request cannot reach the classification at all. The `resources` op's reflection audit —
set every `Request` field in turn, expect refusal — now covers the new one too.

**The cadence has one home.** `lifecycle.go:518`'s inline `Reaper` literal became `launcher.TreeReaper(table)`
(3 s SIGTERM, 2 s SIGKILL, 50 ms poll), so this second reaper inherits #399's measured fuse rather than
inventing a shorter one. The tree is reaped, not the pid: killing only the editor reparents its language
servers to launchd and manufactures the next #583.

**Outcomes, and what each costs the exit code.** Every row the plan named gets exactly one outcome
(`reaped`, `skipped`, `failed`, or `refused` when unconfirmed), because the operator confirmed each of them
and a row that vanished from the output is a row they cannot account for. A skip — the pid is gone, or
recycled into another generation — exits 0: that is the world having moved and the leak they confirmed
against is gone either way. A SIGKILL survivor or a refusal exits 1. The wire projection can truncate
`OrphanNvim` at its detail-limit budget, so the CLI prints `Resources.Errors` as `Coverage:` lines on stderr
*ahead* of the outcomes: a row this verb did not consider is said before the fact, not after it as an excuse.

**Mutation-checked, each reverted:** dropping the protocol gate (caught by the gate test *and* end to end by
`…SupervisorClassifiesFresh`: "an unconfirmed reap was answered ok"); answering from `sampler.view()` instead
of `sweep` (caught on process count); removing `Reaper`'s identity fences (caught by
`TestReapOrphanNvimFencesEveryRowOnItsIdentity` and by `TestReapOrphansCLIWithConfirmationReapsAndReports`,
which names the unsafe signal — "a recycled pid was signalled: [terminated]" — alongside #399's own
`TestReapNeverSignalsARecycledPid` / `TestReapRefusesAChangedServer`). Passing an empty identity from the new
code fails *closed*: every row is skipped and nothing is signalled, so the dangerous direction is the one the
launcher mutation measures.

Evidence: `gofmt -l` clean on every file this touched (`session_sync_test.go` stays as main left it),
`go vet` clean over couchcmd/couchmessage/launcher, `go test -race -run 'TestReapOrphan|TestResource'` green,
and full package runs green — couchcmd 54.7s, couchmessage 7.2s, launcher 35.1s, couchcore 255.4s.

### 2026-10-16 — M2 Task 17: the cost at maximum cardinality, and a live check that had to change shape

**Cost, measured at the widest gates the pass has.** `TestResourceSamplerNativeCost` now runs with
`OrphanNvimAge = 1s` (the floor `ParseResourcePolicy` accepts) and `OrphanRSSBytes = 0`, so no row is
filtered out by a threshold this host happens to sit under: the age floor admits every PID-1 nvim to the
argv read and the cwd probe, and the RSS floor — which is also #587's unattributed *report* floor — admits
every known-kind launchd child to the report and its serialization. Per-call native cost, measured
separately: cwd 8.609 µs, argv 26.107 µs, so #583's probe is the *cheaper* of the two per-row reads. At the
8192-row synthetic ceiling that extrapolates to ~70 ms of cwd probes on top of ~214 ms of argv reads #587
already paid.

**But the assertion is load-sensitive, and it went red mid-measurement.** The test's gate is mean < 120 ms
and max < 300 ms across three passes (#587's "below 0.2 % of one core at the default minute cadence"). Two
runs of the same code, hours apart, on the same host:

| host state | rows | mean | max | pass 1 | pass 2 | pass 3 | calls p1 / p2 | verdict |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| load ~14 | 1270 | **90.535 ms** | 142.409 ms | 142.409 ms | 63.952 ms | 65.244 ms | 1585 / 64 | PASS, 0.150892 % |
| load ~56 (other slots' full suites) | 1323 | 136.712 ms | 245.274 ms | 245.274 ms | 80.329 ms | 84.535 ms | 1581 / 66 | FAIL on the mean |

Three things fall out:

1. **It is not #583's cost.** Re-running with this task's widening removed, under the same load ~56, gave
   mean 138.438 ms / max 265.049 ms — *worse*, so the widening is inside noise. On the widened run the whole
   probe cost 2 argv reads + 2 cwd probes ≈ **69 µs** per pass.
2. **Steady state is inside the budget in both runs** (64-85 ms); only pass 1 moves, from 142 ms at rest to
   245-265 ms under load. Pass 1 makes ~1521 more native calls than a steady pass — one environment read per
   process generation, which is #606's detached-slot attribution, and which `atlas/couch.md` already
   documents as "about 0.3 s for 1,200 processes on Couch's first pass, then only new processes".
3. **The budget predates that scan.** #587 landed 2026-10-15 and qualified at 71.331 ms mean / 74.264 ms max
   over ~1007 rows and 58 native attempts; #606 landed the day after and roughly doubled the first pass. Its
   own plan records the TL ruling that "the binding CPU budget applies to steady sampling, not a rare
   maximum-cardinality admission burst" — and a three-pass mean now folds in a cold pass that is neither.

So the answer to Step 1 is: **0.150892 % of one core at the 60 s cadence, inside the approved <0.2 %**, and
#583's own contribution at maximum cardinality is ~69 µs. But the gate that says so is a coin flip on fleet
load, and nothing in its output distinguishes "the host was busy" from "the sampler regressed" — a
verification defect that belongs to #587 and #606, not here. The constant is #587's approved number and this
issue does not move it; filed separately and reported to the hub.

**What the PID-1 gate is worth.** Mutating `UnattributedShape`'s `p.PPID == 1` to `>= 1` — a one-character
relaxation — took the same pass from 2 to **122** argv+cwd probe pairs on a host running ~30 pair embeds.
That is the gate doing the cost containment, not the age threshold.

**A real orphan on this host, found while measuring.** `ps -eo pid,ppid,etime,rss,command` names pid 89403:
`nvim --embed -u <pair-slot6>/nvim/init.lua <scope>/draft-grok.md`, PPID 1, alive **8 days 10 h**, 5.6 MB
RSS, cwd `/Users/xianxu/workspace/pair` (`lsof -d cwd`). Read against the classifier's gates it is a
candidate on every one — `nvim` short name, PID 1, an exact `--embed` argv field, an age far past 6 h, and a
cwd inside the hub slot's managed root — so a supervisor built from this branch would name it in
`couch --resources` and offer it to `--reap-orphans`. It was *not* named by the measurement above, because
that test's inventory holds one synthetic slot whose root is a temp dir; the managed-root gate did exactly
what it is for and refused to reach outside it. Two things worth carrying out of this: the gate's
containment is real and observable, and its coverage limit is real too — `ManagedRoots` is the set of
**inventory** slots' `WorktreeRoot`, so an orphan whose slot has left the console's slot list entirely has no
root left to match and is not named. Reported to the hub; not fixed here, since #583's spec asks for
"cwd inside a pair-managed worktree" and this satisfies it. (It is also a *tagged* embed, so M1's launch
sweep should recover `grok` from its argv and reap it at slot 6's next launch — the operator's call, not
this slot's, and nothing here signalled it.)

**The plan's live check was not performable as written, so it changed shape.** Step 3 said to run
`couch --resources` and `couch --reap-orphans` on this machine. The supervisor serving this fleet was started
from a build that predates this branch, so neither the `reap-orphans` op nor the orphan report section exists
in it, and restarting the fleet's Couch is the operator's call, not a worker slot's. Restarting to satisfy a
step is the wrong trade: the substitution is stronger evidence anyway. `cmd/internal/couchcmd/
reap_orphans_live_test.go` (env-gated on `PAIR_LIVE_ORPHAN_REAP=1`, this repo's live-test idiom) asserts the
same three claims against the real host, and all three pass:

- a real reparented `nvim --embed` at PID 1, cwd inside a managed checkout, is named by the real sampler over
  the real table — identity, cwd, root and age all read natively, no synthetic field anywhere
  (`orphan_nvim=1`, `reason=unattributed embedded nvim`);
- an embed with a live parent, in the same checkout, under the same age gate, is **not** named
  (`orphan_nvim=0`) — PID 1 is the only difference between the two spawns, since both run the same script;
- `reapOrphanNvim` over `launcher.TreeReaper(launcher.OSProcessTable{})` ends the real orphan, proven by the
  kernel no longer reporting that generation for the pid rather than by a poll that went quiet.

The refusal half of Step 3 is covered where it lives: `TestReapOrphansCLIRefusesWithoutConfirmation`.
Mutation-checked, then reverted: relaxing the PID-1 gate makes the live-parent case fail on its own terms —
"the sweep named 16284, whose parent was still alive".

**An M1 measurement was wrong, and the reason is worth keeping.** M1's `launcher/nvim_reap_live_test.go`
header records that a disposable parent leaving the embed alive at PID 1 is something "this host does not
offer". It does; two constraints have to hold together, and the first two attempts each held one:

1. the embed's stdin must be **held open** — M1's own measurement, ~31 ms to exit once it closes;
2. it must be **redirected explicitly** — bash sends an asynchronous command's stdin from `/dev/null` in a
   non-interactive shell, so an `os.Pipe` handed to `cmd.Stdin` and inherited by the backgrounded embed
   reaches it as an immediate EOF. Measured 2026-10-16 with the write end open the whole time: the
   inherited-pipe embed was gone from the kernel table within 150 ms of its shell exiting (`STAT UN` at
   20 ms, absent at 180 ms, empty stderr — it was never an error, just EOF). The same embed behind `< fifo`,
   with this process holding the write end, sat at PID 1 in state `S` for the entire poll.

So `spawnEmbed` makes a FIFO in its own temp scope, opens the write end in a goroutine (a FIFO's two opens
rendezvous, so neither side races), and reads the pid from a sidecar file rather than the shell's stdout —
in the live-parent case the shell never exits, so there is no `Wait` to close a stdout copy against, and
reading a buffer a copy goroutine still owns is a race. The launcher header's stale claim is corrected with
the measurement and a pointer.

**Two ways the live-parent case could have passed for the wrong reason.** (a) `exec.CommandContext` with the
spawn function's own `defer cancel()` SIGKILLs the shell the moment that function returns — which reparents
the embed to PID 1 *before the sweep runs*, turning "an embed with a live parent is never touched" into the
orphan case and passing it vacuously. The spawn is plain `exec.Command` now, with the cleanup owning that
lifetime. (b) The verdict alone cannot distinguish "the live-parent rule held" from "the embed died, or was
reparented mid-run", so the test re-reads the kernel's table *after* the sweep and fails unless the embed is
still there, still holding the identity it was spawned with, and still not at PID 1. Both guards were added
after the first green run, and the mutation re-run above is from the guarded version.

**Docs.** `atlas/couch.md` gains an `### Unattributed embedded nvim (#583)` subsection (classification
evidence, the cwd probe, the report section, the persisted ring, the hub alert's addressing, the verb's two
modes, the identity fence, the outcome/exit-code table, owners and the live conformance) plus the
`COUCH_RESOURCE_ORPHAN_NVIM_AGE` row; its "does not kill anything" sentence now says the monitor does not,
and the one destructive verb is the operator's and confirmed. `README.md` gains the verb in the command list
with its `[seconds]` cost tag, the new alert trigger, and replaces "Monitoring never kills processes" with
what the two modes actually do.
