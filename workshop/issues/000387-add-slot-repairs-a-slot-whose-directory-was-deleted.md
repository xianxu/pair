---
id: 000387
status: codecomplete
deps: [ariadne#294, ariadne#295]
github_issue:
created: 2026-10-02
updated: 2026-10-06
estimate_hours: 8.34
card_mirror: '8732981958a999481b4c8ec4930f3d87c4059b9a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-05T10:47:34-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 10.06
---

# Slot reconciler: reconcile a Couch slot's dispersed state

## Problem

A Couch `:1+` slot is not one thing. Its state is spread across systems that each
own a piece: the git worktree registration and `main-slotN` branch (git, in the
primary repository), the checkout directory, the slot's dependency clones and
generated files (weave), Couch's setup-success marker and creation intent, Couch's
per-slot store `<env>/.couch` (thread record, archive, preferences), the agent
session (zellij/Pair), the agent's conversation transcript, the operator's own
work in every checkout (uncommitted and untracked files, local-only commits), and
sdlc claims. Any of these can be missing, stale or broken independently, after a
crash, an interrupted setup, or a deleted directory.

Today each Couch operation (add slot, open, resume, reboot, archive) handles the
cases it happens to meet, so failures are partial, messages are unhelpful ("open
again to retry" when a retry cannot succeed), and some states have no repair at
all. Observed in pair#367's smoke test (2026-10-05): `tools:1` was nearly empty
(setup interrupted Sep 27; no `ariadne` clone, no generated files, no setup-success
marker) and reboot could only fail. A deleted slot directory leaves a git
registration and `main-slotN` that block add slot (this issue's original scope,
kept below). A scripted `couch --rebuild` (a phase journal, i.e. a saga) was drafted
in #367 and dropped after four review rounds kept finding uncovered paths: a global
state machine over these pieces grows exponentially.

## Spec

Captured for operator review; no implementation is authorized by this issue
creation. Principle: ariadne#291 (reconcile dispersed state; don't script it).

**Model a slot as resources and reconcile it.**
- **Resource table.** Each piece of slot state is a resource with an owner, a kind
  (Couch-internal, derived — recreatable from a source, user data — preserve
  only, runtime, external — never touched), a desired spec, an observe step that
  yields present / absent / broken / unknown, and an idempotent converge step.
  Starting point: the component table in #367's plan Revision g (C1–C15:
  `.couch` store, agent session, transcript, Pair per-thread artifacts, host
  worktree dir, git registration + setup-success marker, creation intent,
  `main-slotN`, other branches, dirty/untracked files, dependency clones,
  generated files + setup lock, claims, Couch enrollment, saved-work family).
  Verify each against the code.
- **Dependency graph.** Edges between resources (e.g. dependency clones and
  generated files need the checkout; the checkout needs the registration; the
  agent needs the setup marker and the thread record). Converge in topological
  order; tear down in reverse.
- **One reconcile operation.** Observe everything (plan), show the diff, then
  converge in order (apply). Run by the Couch server through the existing
  operation queue; idempotent, so recovery from any partial failure is "run it
  again", and creation (add slot), repair after a crash or interrupted setup,
  and rebuild are the same operation with different desired specs (rebuild =
  "derived resources reset to absent, then reconcile").
- **Guards.** User data is never converged over: it is preserved, or saved first
  (including ignored files, stashes, and local-only refs), with a size bound. An
  unknown observation stops the walk at that node, with the reason shown. A live
  or busy agent is never stopped by reconcile without the existing
  confirmations. `:0` is never recreated.
- **Errors fall out of the model:** "resource X did not converge: cause; fix",
  with the cause classified as permanent (recognized deterministic, e.g. a
  `construct/deps` substrate without a clone source), retryable (weave's "setup
  is active", timeout, cancellation) or unclassified (shown verbatim, may be
  transient). Never a futile "retry".
- **Observability.** `couch --show` and pair#367's recovery report read the
  reconciler's observations; "setup incomplete" and "leftovers block
  provisioning" are named states with their converge step.

**Carry in from #367 (decided or found in review):**
- Provisioning already re-runs `weave compile` for a present, unconfirmed host
  (`couchcore/provision.go:177-198`); a missing declared substrate must also
  trigger it (decided), but repair must never make a working slot worse: a valid
  marker with a failing compile falls back to reuse with a warning (decided).
- The weave setup lock is permanent by design and is not evidence; Couch's
  `couch-setup-success.json` is the "setup completed" signal. An invalid marker
  blocks provisioning ("setup conflict"); an unreadable one is unknown.
- No archive-only slot retirement exists (reboot's is `replaceSlotCurrent` with a
  successor); the `.couch` store lives inside the slot directory, so its archive
  and preferences must survive a directory reset and stay visible to `couch
  --archived` and `archive_gc`.
- The vacancy predicate (what add slot needs of a never-used number) must be
  derived inside `ensureHost`, which also refuses conflicting
  `branch.main-slotN.*` config and a `main-slotN` checked out elsewhere.
- Hold `.weave-setup.lock` across any delete of slot files (probe without
  O_CREAT); a saved-work family needs a real collector (`archive_gc` or the Couch
  store GC, not a storagegc family that does not exist).
- Pair-layer code must not call sdlc/weave (layer rule); couchcore provisioning
  may. Next-step text is single-sourced and names only actions reachable from the
  caller (switcher rows vs CLI).
- Full findings: #367's Log entries "M3 split" and "M3 dropped", and its plan
  Revisions g/h/i (`workshop/history/issues/000367-recover-owned-slots.md`,
  `workshop/history/plans/000367-recover-owned-slots-plan.md`).

## Done when

- A resource table and dependency graph for a `:1+` slot exist in code as the
  single source the reconciler, `couch --show` and the recovery report read; a
  test derives the resource list from the code paths that touch slot state (no
  hand-maintained second list).
- Reconcile is idempotent and converges from every partial state in a derived
  domain of per-resource starting states (present / absent / broken / unknown per
  resource), proved by a test that runs it twice and checks the second run is a
  no-op; unknown stops at its node; user data is never deleted unsaved.
- Real-git acceptance cases converge: a deleted slot directory with leftover
  registration and `main-slotN` (this issue's original case, which must no longer
  block add slot or allocation for the whole repository); an interrupted setup
  like `tools:1` (missing clone and marker); a missing dependency clone under a
  valid marker; a dirty slot whose broken checkout (a dependency clone, or the
  host when `git worktree repair` cannot fix it) must be recreated saves it first:
  the directory is moved whole into a saved-work entry in the slot store, the
  host is re-added on its recorded branch, and the result names how to restore
  it. A checkout that is merely stale is never touched.
- A failure names the resource that did not converge and the cause verbatim
  (weave's own `Error:` line for setup). Only a retryable cause (setup running
  elsewhere, timeout, cancellation) says to run it again; every other failure says
  to hand it to the repository's `:0` agent. The `tools:1` case shows weave's
  missing-substrate line and the `:0` hand-off, never "retry".
- Add slot, open, resume and reboot converge the workspace through the reconciler
  rather than their own repair logic, with no confirmation for routine repair;
  `couch --show repo:N` prints the observed resources and the plan, and
  `couch --reconcile repo:N` applies it explicitly.

## Estimate

Items in plan order. M1: SlotLayout, resource table + AST audit, ObserveSlot,
PlanSlot, layergraph import, `--show`. M2: converge steps, reconcile loop +
SlotWorld, Ensure replacement, failure classification, callers + outcome table,
`--reconcile` + acceptance. M3: SetAside, saved-work GC, recovery report,
dirty-slot acceptance. Then docs ×3 and milestone reviews ×3. Design is v2 ×0.2
(the durable plan pre-resolves decisions); impl is 40% of v2 (v3.1).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: cross-cutting-refactor design=0.2 impl=0.2
item: greenfield-go-module design=0.4 impl=0.32
item: greenfield-go-module design=0.4 impl=0.32
item: greenfield-go-module design=0.3 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.4 impl=0.32
item: greenfield-go-module design=0.4 impl=0.32
item: cross-cutting-refactor design=0.2 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: cross-cutting-refactor design=0.2 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: atlas-docs design=0.04 impl=0.08
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 8.34
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

Durable plan: `workshop/plans/000387-add-slot-repairs-a-slot-whose-directory-was-deleted-plan.md`
(fresh-context plan review: 5 rounds, approved 2026-10-05; awaiting operator
approval and the three sign-offs listed at its top).

- [x] M1 — resource model, `SlotLayout`, AST coverage audit, `ObserveSlot`, pure
  `PlanSlot` over the derived domain, `couch --show` resources and plan
- [x] M2 — `Reconcile` loop replaces `Ensure`'s repair logic; failures name the
  resource and cause and hand off to `:0`; add slot, open, resume and reboot
  converge through it; `couch --reconcile`; acceptance for deleted, interrupted
  and missing-dependency slots
- [x] M3 — save then remove a broken dependency, saved-work retention, the
  recovery report reads slot plans, dirty-slot acceptance

## Log



- 2026-10-06: closed — M1–M3 closed with boundary reviews (M3 SHIP at a7acac18). Close round 7 produced no parseable verdict (prose said FIX-THEN-SHIP, one Minor BR-21) — BR-21 fixed with a mutation-checked test (d439ada4). Operator live smoke test in pair:0 PASSED 2026-10-06: couch --show tools:1 / parli:1 reports, couch --reconcile parli:1 converged with ariadne#296's weave, hand-off text correct with the old weave. Smoke-test finding fixed: weave identity is a setup input of the remembered-failure digest (28486df7, TestRememberedSetupFailureRetriesWithAnUpgradedWeave, mutation-checked). Full suite unsandboxed after it: make -k test / test-changelog / go test ./... — every failure reproduces on origin/main 57ae1bed (spawn registration, continuation writer, cold-resume load flake, launcher x5, workbenchshortcut, gcruntime locator, embedded-runtime; artifactpath set identical).; review verdict: SHIP
### 2026-10-02

Filed at the operator's request from the #363/#367 design talk: a deleted slot
directory should be repaired by prune-then-add-slot, not left blocking.

### 2026-10-05
- 2026-10-05: closed M3 — M3 + round-5 fixes: set-aside entries reported by identity and the printed restore command executed in TestAcceptanceDirtySlot (shell-quoted, moves the recreated checkout aside first; a failed run still names it); saved work collected by the reconciler past retention (future saved_at not believed); recovery report decides needs-zero through SlotOutcome (TestRecoverSlotClassAgreesWithTheCallers, mutation: 56 wrongly held), recoverReason total over classes, degraded note, class renamed slot-needs-zero; metamorphic stride over the evidence domain; couch skill/README/atlas swept for the vocabulary; report budget 61 ms/slot over 10 real slots; real-git acceptance: dirty slot and foreign environment. Full suite unsandboxed: remaining failures reproduce on origin/main 57ae1bed (spawn registration, continuation writer, cold-resume load flake, launcher, workbenchshortcut, gcruntime locator, embedded-runtime; artifactpath set identical); godoc stacking fixed (a7acac18).; review verdict: SHIP
- 2026-10-05: closed M2 — M2 + round-3 fixes: Reconcile loop proved over every single+pair perturbation x 7 agent states (SlotWorld agrees with real git on 12 scenarios; crash after every step converges); Ensure is Reconcile; SetAside (refusals leave no residue, typed into the plan's class); failures retryable/handoff/unknown/hold with one advice text; OutcomeSeverity by observation; selectedSlot reconciles every open/resume/reboot/fresh start, reboot's first pass gets past the live-agent hold its advice names (TestRebootGetsPastAHoldItsAdviceNames); Discover unions env/registrations/main-slotN; couch --reconcile renders the report on failure; real-git acceptance: deleted slot dir, tools:1 hand-off, missing dep under valid marker, observe 68ms. Full suite unsandboxed: remaining failures reproduce on origin/main 57ae1bed (spawn registration, continuation writer, cold-resume load flake, launcher, workbenchshortcut, gcruntime locator, embedded-runtime; artifactpath set identical).; review verdict: FIX-THEN-SHIP
- 2026-10-05: closed M1 — M1 + boundary round-1 fixes: SlotLayout; resource table + AST coverage audit (3 mutations caught); DeclaredDepsOf via ariadne layergraph b9bc9f32; ObserveSlot on real git (single-break cases, failed probes unknown, gitfile dependency readable via --show-toplevel; mutations caught); PlanSlot over the derived single+pair domain x 7 agent states plus a hand-written I2 case; couch --show slot report through Discover's git identity (deleted-slot and symlinked-fleet cases; old construction reproduces BR-2's misreads); README documents --show repo:N. Full suite unsandboxed: remaining failures identical on origin/main 57ae1bed; TestColdResumeOfAParkedPrimary... passes 3/3 alone (load-timing flake).; review verdict: SHIP

Re-scoped at the operator's direction from "add slot repairs a deleted slot" to
the slot reconciler, after pair#367's M3 (`couch --rebuild` as a scripted saga) was
dropped. The deleted-slot repair stays as one acceptance case. Related:
ariadne#291 (the ARCH principle), pair#390 (wrapper messaging needs no sdlc).

Design decisions (operator, planning session):
- The desired state is a working slot. Past state matters only where it is the
  operator's: uncommitted and untracked files, local-only commits, the slot
  store's archive and preferences (the transcript is never touched). Everything
  else is derived and repaired automatically, without confirmation. There is no
  operator "rebuild" verb: a derived resource that cannot converge in place is
  saved (if it holds user data), removed and recreated.
- A leftover `main-slotN` without ownership proof is adopted (checked out again,
  its commits kept); refuse only if it is checked out elsewhere or its upstream
  config conflicts.
- Saved work: the host checkout's dirty and untracked files become a
  stash-shaped commit on a private ref `refs/couch/saved/slotN/<id>` (not the
  shared stash stack, which every worktree and session shares); dependency clones'
  dirty work and local-only commits go to `<env>/.couch/saved-work/<id>/`. Both
  are collected after storagegc's 60-day retention. Ignored files are derived and
  not saved.
- A live agent whose Couch record is lost or unreadable is adopted, never
  stopped. Reconcile never removes a checkout under a live agent; that case stops
  and reports.
- An unknown observation stops the walk at that resource.
- A converge step that fails is reported with its verbatim cause and handed to
  the repository's `:0` agent to debug; only retryable causes say run again.
  `tools:1` is this case: `tools/construct/deps` declares `substrate ../ariadne`
  without a URL (verified 2026-10-05; pair's line carries the URL), so a fresh
  `weave compile` fails identically. The fix belongs in tools.
- Surface: reconcile runs inside add slot, open, resume and reboot; `couch --show
  repo:N` shows resources and the plan; `couch --reconcile repo:N` applies it.

Code survey (2026-10-05): nothing in `cmd/` deletes a slot directory, its git
registration, `main-slotN` or its config, so any leftover blocks forever; a
leftover registration blocks add slot for the whole repository
(`SelectStartSlot`, `slotallocation.go`). Slot discovery has two disagreeing
predicates (`EnumerateSlotCandidates` filesystem-only vs `OSSlotCatalog.Discover`
plus `git worktree list`). `selectedSlot` already calls `Workspaces.Ensure` for an
unverified candidate, and `NextHostAction` is already observation-driven: the
reconciler generalizes `Ensure` rather than adding a second provisioner.

Plan review (fresh-context reviewer, 5 rounds, 2026-10-05). Each round's delta is in the
plan's Revisions. The design moves that came out of review: targeted `git worktree remove`
instead of `prune`; a dependency is broken only on positive evidence; `selectedSlot` always
reconciles; outcomes are blocking or degraded per caller (a live agent or a transient unknown
never blocks attaching); saved work is a set-aside rename (the 65 MB `ariadne` clone ruled out
a tar); a failed setup is memoized; add slot refuses with the hand-off instead of skipping
numbers, because `construct/deps` is shared through `main`.

Fleet audit (2026-10-05): only pair, parley.nvim, ducks and test-repo declare a
clone source on their `construct/deps` substrate row; 15 derivatives do not, so a
new slot of any of them fails setup the way `tools:1` does. Root cause: the
source migration (ariadne #239/#243) reached only pair and parley.nvim. Filed
ariadne#293 (record sources fleet-wide; weave flags a sourceless row). The
operator rejected a weave fallback that infers the source from a sibling
checkout's `origin` as fragile. #387 does not depend on it: reconcile reports the
failure and hands off to `:0`.
Filed ariadne#294 (export `layergraph.DeclaredSubstrates`, reporting absent
substrates that `Walk` present-skips); #387's Task 1.3 imports it, so it is in
`deps:`. Both ariadne filings' details are local in `~/workspace/ariadne`.

change-code (2026-10-05): full flow; plan-quality cleared after 3 rounds (PQ-1
host set-aside machinery fixed as a class; PQ-3, the prose test-case lists, carried
to close as Minor). Estimate 8.34 h (v3.1, Method A). The estimate-quality judge
returned INFO: the derivation is genuine but likely low (10–12 h). Task 3.2 has no
item of its own, and 2.5 and the acceptance tasks are costed light. The recorded
estimate is left as derived, so the actual calibrates against it.

M1 verification (2026-10-05). Unsandboxed, with the Pair session variables scrubbed:
`make -k test`, `test-changelog` (scratchpad TMPDIR, passes), `go test ./...`, and race on
couchcore. Every remaining failure also fails on `origin/main` (57ae1bed, checked in a scratch
worktree):
- `TestSpawnComposesProductionPairRegistrationBoundary` ("Couch thread claim missing");
- `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` ('origin' is not a repository);
- `test-pair-embedded-runtime` (inherited `PAIR_DATA_DIR` scope conflict);
- `artifactpath` `TestProductionArtifactReferencesAreExactlyClassified` (failure set byte-identical
  to main after the new files were classified).

The new audits found real issues, all fixed: the six new files were not classified in the
`artifactpath` inventory; three production symbols had only test consumers; one token match
lacked its boundary.

Resume point (2026-10-05, later): M1 closed (SHIP, c4637555); M2 closed (FIX-THEN-SHIP,
fd7da051). In M3, Tasks 3.1 (in M2), 3.2 (87a20de4) and 3.3 (8fb75941; owner collection, and
the hazardous RegisterStore hook removed) are done. Next: Task 3.4 (the recovery report reads
slot plans through RecoverSlotClass), then 3.5 (dirty-slot acceptance), 3.6 (docs, issue,
close). The live smoke test in pair:0 still awaits the operator's go-ahead to move the branch.

## Revisions

### 2026-10-05 — Done-when follows the planning-session decisions

Reason: operator decisions above (automatic repair, `:0` hand-off for failures
reconcile cannot fix, no rebuild verb). Delta: the acceptance bullet for a dirty
slot names where work is saved; the failure bullet drops the tools-specific fix
text and classified-fix wording for verbatim cause + retryable-only "run again" +
`:0` hand-off; the last bullet replaces "rebuild" with add slot/open/resume/reboot
plus `--show` and `--reconcile`.

### 2026-10-05 — re-scoped to the slot reconciler

Reason: operator: "without properly modeling the dispersed state across several
systems, you won't be able to patch out of the problem". Delta: the problem, spec
and Done-when were rewritten from the original deleted-slot repair (preserved in
git history and as one acceptance case) to a declarative resource model with a
dependency graph and idempotent per-resource convergence; the title changed via
`sdlc issue set-title`.

### 2026-10-05 (b) — Done-when follows plan refinement R2

Reason: planning showed the host checkout is never removed (absent → re-add,
mismatched → `git worktree repair` or hand-off), and a dependency is removed only
when git positively cannot read it, so a git-based save cannot run. Delta: the
dirty-slot acceptance saves a broken dependency as a whole-tree tar in
`<env>/.couch/saved-work/<id>/`; the stash-shaped host ref is dropped. See the
plan's R1–R5 and its plan-review revision.

### 2026-10-05 (c) — saved work is a set-aside directory, not a tar

Reason: plan review round 2 measured the real `ariadne` clone at 65 MB, above any
sensible tar cap. Delta: the dirty-slot bullet's save becomes an atomic rename of
the broken dependency into `<env>/.couch/saved-work/<id>/tree` (nothing lost, no
size cap); restore is a move.

### 2026-10-05 (d) — the host checkout follows the dependency rule

Reason: operator: a clean host is no different from a dependency; the earlier
"never removed" rested on the tar save, which the set-aside rename replaced.
Delta: the dirty-slot bullet covers a broken host (set aside after repair fails,
re-added on its recorded branch) and states that a stale checkout is never
touched. Saved-work retention follows `storagegc.RetentionPeriod`, one year since
pair#393 (was 60 days in the Log's decision).

### 2026-10-05 (e) — Spec deviations at implementation

Reason: findings during M2–M3, recorded in the plan's Revisions (s), (w) and (x). Delta against
the Spec:
- **Where reconcile runs.** It runs inside each caller (add slot, open, resume, reboot) through
  `Ensure`, in the process that calls it. `couch --reconcile repo:N` is a direct CLI operation
  rather than a request through the Couch server's queue: the socket's admission requires a thread
  row, and a slot with none (a deleted directory) must still reconcile. The host creation lease
  serializes concurrent runs.
- **Who collects saved work.** The reconciler that sets checkouts aside removes entries older than
  `storagegc.RetentionPeriod` at the start of every run on the slot (16 entries at most), not the
  archive GC pass. Registering slot stores with the storage coordinator was removed as unsafe.
- **The recovery report.** It reads the reconciler through one evidence dimension. "Setup
  incomplete" and "leftovers block provisioning" from the Spec appear as classes `reconcilable` and
  `slot-needs-:0` with hold `workspace-handoff`.
- **Severity.** Blocking versus degraded is decided by whether an agent can work in the slot (the
  host is present and setup completed once), not by which resource failed.

Done-when is met as revised in (b)–(d): every bullet has real-git or derived-domain evidence, listed
in the plan's Revisions (j)–(y).
