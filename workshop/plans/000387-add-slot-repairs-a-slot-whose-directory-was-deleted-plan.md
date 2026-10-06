# Slot Reconciler Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Model a Couch `:1+` slot as a table of resources with a dependency graph, and make one
level-triggered reconcile operation (observe → plan → converge, repeated until the plan is empty)
the only way any caller brings a slot to "working". Add slot, open, resume and reboot use it, and
so do `couch --show` and the recovery report.

**Architecture:** A pure core (`slotresource.go`, `slotplan.go`) declares the resources and their
edges, and turns a `SlotObservation` into a `SlotPlan` of converge steps. A thin shell
(`slotobserve.go`, `slotconverge.go`) observes git, the filesystem and weave through the existing
`ProvisionIO` seam, and executes steps. `Reconcile` loops observe → plan → apply until the plan is
empty, a step fails, or no progress is made. It keeps no state between runs (no journal). It
replaces the body of `WorkspaceProvisioner.Ensure`, which is already observation-driven
(`NextHostAction`), so there is still exactly one provisioner.

**Tech Stack:** Go; real git in tests (`ProvisionFixture`); a stateful fake `weave` script on
`PATH`; an in-memory `SlotWorld` fake for the convergence domain test.

**Source:** issue `workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md`
(Spec, Done-when, Log "Design decisions (operator, planning session)"). Principle: ariadne#291
(reconcile dispersed state; don't script it). History: pair#367's dropped M3 and its Revisions
g/h/i (`workshop/history/plans/000367-recover-owned-slots-plan.md`).

---

## Operator sign-offs (2026-10-05)

1. **R1: accepted, as reworded.** Couch stops refusing slots it did not itself create. Couch is
   a singleton, so "not created by Couch" realistically means one of three things: a slot from
   before creation tracking, or from an interrupted setup (`tools:1`); the leftovers of a slot
   Couch lost track of (this issue's original case); or something done by hand at that path.
   The right move in each case is to adopt what is there. The worst that can happen to such
   content is a broken dependency being set aside intact.
2. **Deps parser: depend on ariadne, don't copy.** pair imports
   `github.com/xianxu/ariadne/pkg/layergraph` (stdlib-only), pinned with
   `go get github.com/xianxu/ariadne@<sha>`, with no `replace` to a local tree. The Homebrew
   build fetches the pinned module, and the live ariadne tree never leaks into pair's build.
   `layergraph.Walk` silently skips an *absent* substrate (`walk.go`, "present-skip"), which is
   exactly what reconcile must detect. So ariadne first exports
   `DeclaredSubstrates(fs FS, root string) ([]DeclaredSubstrate, error)`: every substrate
   declared transitively over the *present* layers, with `{Path (physical), Owner (declaring
   repo), Source, Present}`, built on the same unexported `substrateTargets` that `Walk` uses,
   so the two cannot diverge. That is its own ariadne issue (pair#387 `deps:` it). Task 1.3
   waits for it to land; Tasks 1.1–1.2 do not.
3. **Done-when wording (dirty slot):** pending the explanation of "host checkout".

## Non-goals

- Freshness: reconcile never fetches, pulls or recreates a checkout for being behind origin
  (sdlc's resting-branch handling owns that).
- Thread records: an unreadable conversation still refuses add slot; the agent's transcript,
  Pair per-thread artifacts, sdlc claims and the manifest enrollment are never touched.
- `:0`: never planned or converged.
- Guaranteeing add slot always succeeds: a hand-off on any slot (shared setup inputs, or a
  slot-local conflict) refuses it, deliberately; only reconcilable leftovers stop blocking.
- Sourceless `construct/deps` rows: fixed fleet-wide by ariadne#293, not worked around here.

## Decisions this plan rests on (operator, 2026-10-05)

1. **Desired state = a working slot.** Derived state is repaired automatically, with no
   confirmation. Only the operator's data is preserved: uncommitted and untracked files,
   local-only commits, and the slot store (archive, preferences). There is no operator "rebuild"
   verb.
2. **Adopt a leftover `main-slotN`.** It is the resting branch by convention. Refuse only if it
   is checked out in a different worktree or its upstream config conflicts.
3. **Saved work lives in the slot store**, `<env>/.couch/saved-work/<id>/`, and is collected after
   storagegc's `RetentionPeriod` (one year since pair#393).
4. **A live agent is never stopped by reconcile.** A record-less live agent is adopted (the
   existing `OpenSlot` route). Removing a checkout under a live agent stops and reports.
5. **Unknown stops the walk** at that resource and its dependents.
6. **A failed converge is reported verbatim and handed to the repository's `:0` agent.** Only a
   retryable cause (setup running elsewhere, timeout, cancellation) says "run it again".
   `tools:1` is such a hand-off: `tools/construct/deps` declares `substrate ../ariadne` without a
   URL.
7. **Surface.** Reconcile runs inside add slot, open, resume and reboot. `couch --show repo:N`
   prints resources and the plan; `couch --reconcile repo:N` applies it.

### Design refinements made while planning (flagged for operator review)

- **R1 — The creation intent retires, and protection against foreign actors goes with it
  (needs operator sign-off).** `<common>/couch-workspaces/N/creation.json`, its env-inode check
  ("environment was replaced") and the refusal "environment already exists without ownership
  evidence" proved that a half-created `main-slotN`/env was Couch's own. The host creation lease
  (`provision_lock_unix.go:23`) only serializes Couch against Couch. So what is dropped is
  protection against *non-Couch* actors (the operator, sdlc, a sync tool) having put something
  at the slot's conventional path. Under decisions 1–2 that content is adopted. Anything a
  removal would touch is saved first (rule 2), and only a checkout broken on positive evidence is ever set aside.
  Save-before-remove is the only remaining guard. Test: a foreign, pre-populated env (a non-git
  `ariadne/` with files) is adopted, and the dependency is saved before removal. The intent
  becomes a legacy resource whose desired state is absent. Mixed binaries: an older Couch that
  crashed mid-creation and runs again after the new binary removed its intent will refuse;
  acceptable, since `make build` replaces the binary.
- **R2 — One save mechanism for every removal: set the directory aside.** The planning-session
  answer proposed a stash-shaped commit on `refs/couch/saved/slotN/<id>` for the host. Any
  checkout, the host or a dependency clone, is set aside only when it is broken on **positive
  evidence**: no `.git`, or git answers "not a git repository" (for the host: after `git
  worktree repair` was tried and the evidence remains). A stale checkout (behind origin) is not
  broken and is never touched; re-adding would check out the same commit anyway.
  Any other git error is `unknown` (lessons: a failed probe is not evidence). The save is an
  atomic `rename` of the whole broken directory into `<env>/.couch/saved-work/<id>/tree`. Env
  and store share the env root, so it is the same filesystem. Nothing is lost: `.git`, ignored
  files and mtimes all move. There is no size cap, no tar and no git dependency, and the save
  *is* the removal: the dependency is absent afterwards, so it cannot be saved twice. Restore is
  `mv`. Plan review round 2 showed a tar cannot work: the real `ariadne` clone measures 65 MB.
  There is one mechanism and no new ref namespace (ARCH-DRY).
- **R5 — A failed setup is remembered, not repeated on every open or add slot.** Rule 4 keeps a
  working slot usable when its recompile fails, and a slot without a marker refuses (blocking).
  In both cases, re-running a failing `weave compile` (up to `SetupTimeout`) on every open or
  add slot is waste. So the reconciler writes `couch-setup-attempt.json` in the registration's
  admin directory after any setup failure, **with or without a valid marker**. It records an
  inputs digest (host `HEAD` + the bytes of every read `construct/deps`) and the failure,
  **handoff class only**: a retryable failure (timeout, cancellation, a held lock) is never
  memoized. Observation reads it. Same digest → `setup` is `present-with-warning` under a valid
  marker, or `failed-known` without one; in both cases the plan does not compile. A changed
  input, or an explicit `couch --reconcile` (which ignores the memo), compiles again. The digest
  cannot see fixes outside the host (network restored, credentials, a remote created), so the
  advice names `couch --reconcile repo:N` as the way to force another attempt. It is derived
  state, lives and dies with the admin directory, and is overwritten in place (one file).
- **R3 — An invalid setup marker is derived state, not a conflict.** #367 treated an invalid
  `couch-setup-success.json` as "setup conflict, never advise reboot". Here it is a broken
  derived file: remove it and re-run setup. The rule "repair never makes a working slot worse"
  still holds for a *valid* marker whose recompile fails (Task 2.3).
- **R4 — Removal is the rename (superseded by R2's set-aside).** There is no trash directory
  and no `RemoveAll` of a dependency. The only deletes reconcile performs are the legacy intent
  file and a stale registration (`git worktree remove`, whose directory is already gone).
  Collection of set-aside trees is the GC's, after `storagegc.RetentionPeriod` (one year).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SlotLayout` (every slot-state location: env, host, store, saved-work, setup lock, intent, resting branch, marker and memo names) | `cmd/internal/couchcore/slotlayout.go` | new |
| `SlotResourceID` / `ResourceKind` / `SlotResourceSpec` / `SlotResources()` (the table and its edges) | `cmd/internal/couchcore/slotresource.go` | new |
| `ObservedState` (`present`/`absent`/`broken`/`unknown`) / `ResourceObservation` / `SlotObservation` | `cmd/internal/couchcore/slotresource.go` | new |
| `ConvergeStep` / `PlannedStep` / `SlotPlan` / `PlanSlot(SlotObservation) SlotPlan` | `cmd/internal/couchcore/slotplan.go` | new |
| `ReconcileFailure{Resource, Class: retryable\|handoff\|unknown, Cause}` / `ClassifyConvergeError` / `ReconcileAdvice(address, failure)` / `OutcomeSeverity(resource, state, stopReason, marker)` (blocking\|degraded) | `cmd/internal/couchcore/slotfailure.go` | new |
| `DeclaredDeps(host)` over `layergraph.DeclaredSubstrates` (ariadne, imported) | `cmd/internal/couchcore/slotdeps.go` | new |
| `SelectStartSlot` (a reconcilable candidate is reusable; an unknown one skips only its number) | `cmd/internal/couchcore/slotallocation.go` | modified |
| `NextHostAction` / `HostObservation` | `cmd/internal/couchcore/provision_host.go` | deleted (absorbed into `PlanSlot`) |
| `CreationIntent` / `validateCreationIntent` | `cmd/internal/couchcore/provision.go`, `provision_request.go` | deleted (R1; the legacy file is a resource with desired state absent) |

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ObserveSlot(ctx, io, layout) SlotObservation` | `cmd/internal/couchcore/slotobserve.go` | new | git (`worktree list`, `for-each-ref`, `config`, `rev-parse`, `status`), `sdlc workspace --json`, lstat, non-creating flock probe, `construct/deps` reads |
| `Converge(ctx, io, layout, step)` | `cmd/internal/couchcore/slotconverge.go` | new | git (`worktree add/repair/remove`, `fetch`, `update-ref`, `config --add`), mkdir, `rename`, `weave compile`, marker write |
| `SetAside(ctx, layout, checkout, sessions)` (one step for the host and every `dep:*`) | `cmd/internal/couchcore/slotsave.go` | new | mkdir, the session re-check, `rename`, manifest write via `durablefile`, `RegisterStore` |
| `Reconciler.Reconcile(ctx, slot) (ReconcileResult, error)` (the loop) | `cmd/internal/couchcore/slotreconcile.go` | new | the two above, the host creation lease |
| `WorkspaceProvisioner.Ensure` (body becomes `Reconcile`; signature kept for its four callers) | `cmd/internal/couchcore/provision.go` | modified | — |
| `OSSlotCatalog.Discover` (candidates carry their `SlotObservation`; numbers come from env dirs ∪ registrations ∪ `main-slotN` refs) | `cmd/internal/couchcore/slotcatalog.go` | modified | git, filesystem |
| `saved-work` collection in the archive GC pass | `cmd/internal/couchcore/archive_gc.go` + `cmd/internal/gcruntime` | modified | `<env>/.couch/saved-work/` |
| Operation `reconcile` (`ExecuteLiveOwner`, `ConfirmNone`, args `path`, `repo-scope`) and the `show` slot section | `cmd/internal/couchcore/ops.go`, `operationdispatch.go`, `cmd/internal/couchcmd/run.go` | new / modified | stdout |
| Fake `weave` (stateful: clones declared substrates from a source table, honors the setup lock, prints weave's real error lines) | `cmd/internal/couchcore/testdata/fakeweave` + `provision_git_test.go` `ProvisionFixture` | modified | — |
| `SlotWorld` (in-memory stateful world implementing observe + converge, for the domain test) | `cmd/internal/couchcore/slotworld_fake_test.go` | new | — |

### The resource table (`SlotResources()`, verified against the code on 2026-10-05)

| ID | Resource | Location (`SlotLayout`) | Kind | Depends on | Desired | Converge (per observed state) |
|---|---|---|---|---|---|---|
| `env` | env directory | `<fleet>/worktree/<repo>-slotN` | derived | — | present (real dir, not a symlink) | absent → `mkdir 0700`; broken (a file or symlink) → hand-off |
| `store` | Couch slot store | `<env>/.couch` | internal, preserved | `env` | (observed only) | never converged or removed by reconcile; created lazily by the thread store |
| `intent` | legacy creation intent | `<common>/couch-workspaces/N/creation.json` | derived, legacy (R1) | — | absent | present → remove, under the lease |
| `branch` | resting branch `main-slotN` | `refs/heads/main-slotN` in the shared repo | user data (adopted; never deleted) | — | present | absent → fetch remote `main`, `update-ref` create at the fetched OID; checked out in a worktree other than the host path → hand-off |
| `upstream` | `branch.main-slotN.{remote,merge}` | shared repo config | derived | `branch` | `<remote>` / `refs/heads/main` | absent → `config --add`; conflicting → hand-off |
| `registration` | git worktree registration | `<common>/worktrees/<name>/` for the host path | derived | `branch`, `env` | present, directory present | stale (directory gone, git reports `prunable` for **this** path) → `git worktree remove <hostPath>` (targeted; never `prune`, which drops every prunable registration in the shared repository); locked → hand-off; absent → handled by `host` |
| `host` | host checkout | `<env>/<repo>` | derived container; its dirty files are user data | `registration`, `env`, `upstream` | present and verified (`verifyHost`'s identity check) | absent → `git worktree add <host> main-slotN`; mismatched → `git worktree repair <host>` once; still broken on positive evidence (R2) → set aside (rename into saved-work), then `git worktree remove` the registration and re-add on the branch the registration recorded (`main-slotN` if that branch is gone; the result says which); never under a live agent; stale (behind origin) is converged |
| `deps` | dependency declaration | `construct/deps` of the host, then of each present substrate (transitive) | external (another repo's file) | `host` | parseable | unparseable/unreadable → unknown (stop at `deps` and `setup`) |
| `dep:<rel>` | one dependency clone | `<env>/<name>` per declared substrate | derived from a remote; dirty files and local-only commits are user data | `deps`, `env` | present, git-readable | absent → covered by `setup`; broken **on positive evidence only** (no `.git`, or `rev-parse` says "not a git repository"; any other git error is `unknown`) → set aside (rename into saved-work, R2) → absent → covered by `setup` |
| `setup` | weave setup, Couch's marker, the attempt memo (R5) | `<admin>/couch-setup-success.json`, `<admin>/couch-setup-attempt.json`; lock `<env>/.weave-setup.lock` | derived | `host`, every `dep:*` | valid marker and no `dep:*` absent | lock held → retryable stop; a memo whose digest matches → `present-with-warning`, no compile; otherwise `weave compile` (no lease held), then re-observe under the lease and write the marker (success) or the memo (failure under a valid marker); an invalid marker is removed first (R3) |
| `agent` | agent session + thread record | zellij session, `<env>/.couch/thread.json` | runtime | `host` | (observed only) | never converged by reconcile; a live, busy or unknown agent forbids setting aside any checkout, the host or a `dep:*` (hold `agent-live`), re-checked immediately before the rename; a record-less live agent is adopted by `OpenSlot`, which runs after reconcile |

**Outside the table (external, never touched; listed so the coverage audit can name them):**
the agent's transcript, Pair per-thread artifacts (`artifactpath`), sdlc claims, the global
manifest's `SlotRepositories` enrollment, and every other branch in the shared repository.

**Edges, in topological order:**
`env → store`, `{intent}`, `branch → upstream`, `{branch, env} → registration →
host`, `upstream → host`, `host → deps → dep:* → setup`, `host → agent`. Converge runs in this
order. Removal (only `intent`, a stale `registration`, a broken checkout set aside) runs before the converges of its
dependents.

### `PlanSlot` rules (pure; `SlotObservation → SlotPlan`)

1. **Unknown blocks.** An `unknown` resource contributes a `Stop{resource, reason}`. Every
   resource that transitively depends on it is skipped. Resources that do not depend on it
   still converge. Example: an unknown `agent` does not stop `branch`.
2. **User data is never deleted.** A checkout broken on positive evidence (R2), either the
   `host` after its `RepairHost` step left the evidence in place or a `dep:*`, gets one step,
   `SetAside(checkout)`, which moves the whole directory into a saved-work entry atomically.
   For the host, the entry's manifest records the branch the registration's admin `HEAD` named,
   read before the rename. The following passes run `RemoveRegistration` (now stale) and then
   `WorktreeAdd` on that recorded branch. `WorktreeAdd`'s branch is chosen in this order: the
   stale registration's `HEAD`, then the newest host saved-work manifest's `branch`, then
   `main-slotN`. A branch that is gone, or checked out elsewhere, falls through to the next. If
   the entry limit (16 per slot) is reached, the step is replaced by a `Stop{saved-work-full}`.
   No size is measured on any path.
3. **No removal under a live agent.** If `agent` is `live`, `busy` or `unknown`, a broken
   checkout, host or `dep:*`, gets `Stop{agent-live}`. Every non-removing step still runs.
   `SetAside` re-observes the agent sessions immediately before the rename and aborts on a
   change. A plain `pair` launch outside the operation queue can start one between observe and
   apply (lessons: revalidate authority before mutation). A stop's severity comes from its
   **reason**, not its resource: `agent-live` is always degraded, because a live agent is
   attached as-is and reboot is what clears the hold. This holds even on a host. Fresh starts
   only run after the agent is gone (reboot's second pass), where the stop no longer arises.
4. **Repair never makes a working slot worse.** With a valid marker and an absent `dep:*`, the
   plan is `Compile` with `KeepOnFailure`. If that compile fails, the run **ends** with a
   warning (it never re-plans the compile in a later pass), the slot stays usable, and the
   memo (R5) keeps later opens from recompiling until an input changes (decision carried from
   #367 Revision i). With no valid marker, a failed compile is an error.
5. **`:0` is never planned.** `PlanSlot` refuses a slot number of 0 before anything else.
6. **Idempotence by construction.** Every step's precondition is the observed state it fixes, so
   a converged resource never yields a step. The loop stops when the plan is empty.

### The loop (`Reconciler.Reconcile`)

```
var previous *SlotObservation
for i := 0; i < len(SlotResources())+2; i++ {   // bound: each pass converges ≥1 resource or stops
    obs  := ObserveSlot(...)            // under the host creation lease for git resources
    if previous != nil && obs.Equal(*previous) {
        return failure("no progress", PlanSlot(obs))   // guards a converge that reports success but changes nothing
    }
    previous = &obs
    plan := PlanSlot(obs)
    if plan.Empty() { return converged(obs, warnings) }
    if plan.OnlyStops() { return stopped(plan) }
    for step in plan.Ready() {          // steps whose dependencies are all converged
        err := Converge(step)
        if err != nil && step.KeepOnFailure { return converged(obs, warning(ClassifyConvergeError(step, err))) }
        if err != nil { return failure(ClassifyConvergeError(step, err)) }
    }
}
return failure("pass bound exceeded", PlanSlot(ObserveSlot(...)))
```

`Compile` releases the lease before running weave, as `Ensure` does today. The marker is written
only after a fresh observation under the lease shows the same admin directory (today's check).

### Failures and advice (`slotfailure.go`)

`ClassifyConvergeError(step, err) ReconcileFailure`:
- **retryable:** the weave line `environment setup is active in <env>; retry after the active
  setup finishes`; the setup lock observed held; `ErrHostCreationBusy`;
  `context.DeadlineExceeded`; `context.Canceled`.
- **handoff:** every other error. The cause is weave's last `Error:` line (matched as a
  substring, since weave may prefix the owner), or the git/IO error text, shown verbatim.
- **unknown:** a `Stop` from rule 1, with its reason.

`ReconcileAdvice(address, failure)` is the single source of operator text:
- retryable: `slot <addr>: <resource> did not converge: <cause>; run it again when that finishes`
- handoff: `slot <addr>: <resource> did not converge: <cause>. Reconcile cannot fix this; ask the
  <repo>:0 agent to investigate (couch --show <addr> lists every resource)`
- unknown: `slot <addr>: <resource> could not be observed: <reason>; nothing depending on it was
  changed`

The text never contains a bare "retry" for handoff, and never "fix it". `withRebootAdvice`, the
start form, `--reconcile`, `--show` and the recovery report all call this function.

### What a reconcile outcome means to each caller (`ReconcileResult.Severity`)

`Reconcile` itself never decides whether an agent may start. It returns a `ReconcileResult` whose
non-converged resources are each tagged **blocking** or **degraded** by a pure function,
`OutcomeSeverity(resource, state, stopReason, marker)` (the reason decides first: `agent-live` and `saved-work-full` are degraded whatever the resource):
- **blocking:** `env`, `branch`, `upstream`, `registration` or `host` not converged, or `setup`
  without a valid marker (setup never completed: an agent would start in an unprepared
  checkout, today's refusal).
- **degraded:** anything under `deps`/`dep:*`, `setup` under a valid marker (including R5's
  `present-with-warning`), the `agent-live` and `saved-work-full` holds, and `agent` unknown.
  The checkout exists and was once set up, so an agent can work in it.

| Caller | blocking | degraded |
|---|---|---|
| open / resume (`selectedSlot` → `OpenSlot`, incl. adopting a live agent) | refuse with `ReconcileAdvice` | proceed; the advice is a warning on the result |
| reboot, first `selectedSlot` (before `prepareRetirement`) | refuse; nothing stopped | proceed; `agent-live` is expected here, since reboot stops the agent next |
| reboot / fresh start, `selectedSlot` inside `startFreshSlot` (agent already stopped) | refuse; the old record is already archived, as with today's launch failure | proceed with a warning; a broken dependency was set aside and recompiled in this pass, because the agent is gone |
| add slot (reuse route) | refuse with `ReconcileAdvice` naming that slot (the `:0` hand-off). The refusing run writes the memo (R5) before returning, so a repeat add slot refuses at once without recompiling | proceed with a warning |
| `couch --reconcile` | report, exit 1 | report, exit 0 with warnings |
| `couch --show`, recovery report | display only | display only |

**Add slot does not skip to another number.** A blocking setup failure comes from inputs every
slot of the repository shares (`construct/deps` reaches each slot through `main`), so a fresh
number would fail identically. Add slot therefore allocates as usual and refuses with the `:0`
hand-off. The memo makes repeat attempts refuse immediately with the same advice, until an input
changes or `couch --reconcile` forces a compile. The original deleted-directory case is
different: reconcile can repair it, so it no longer blocks the repository (Done-when bullet 3).
This is the `tools:1` case: one compile, then an immediate hand-off on every later attempt.
A failure local to one slot also refuses add slot for the repository, because allocation always
picks the lowest free number. Examples: a leftover `main-slotN` checked out in another worktree,
a locked registration, conflicting upstream config. That is deliberate: the hand-off names the
slot, and `:0` fixes it. This plan does not claim that nothing ever blocks the repository, only
that reconcilable leftovers no longer do.

So a held `index.lock` in a dependency (an agent running git) never blocks attaching to that
agent. A broken dependency on a live slot is repaired by reboot, which is the operation that
clears the hold.

### ARCH notes

- **ARCH-DRY.** One provisioner: `Ensure`'s body becomes `Reconcile`. Its callers are
  `slotlaunch.go:71`, `slotrecovery.go:240` (`selectedSlot`), `slotstart.go`
  (`spawnManagedResolution`) and `operationdispatch.go:154` (`provision-workspace`); Task 2.5
  changes *when* `selectedSlot` calls it. `NextHostAction` and the intent proofs are deleted
  rather than kept beside the new planner. One path authority (`SlotLayout`) replaces the slot
  path math, and the slot-path parsers become one `ParseSlotPath`. Task 1.1 enumerates the
  sites, and Task 1.2's token audit proves the list complete. One advice function. The deps
  declaration is read through ariadne's `layergraph` (sign-off 2), so weave and pair share one
  parser and one resolver.
- **ARCH-PURE.** `PlanSlot`, `ClassifyConvergeError`, `ReconcileAdvice`, `DeclaredDeps`' path checks and
  `SelectStartSlot` are pure. Observe, converge and save are the shell. The loop is thin glue,
  tested against `SlotWorld` and real git.
- **ARCH-PURPOSE.** All four Done-when consumers derive from the table: the reconciler,
  `--show`, the recovery report and add-slot allocation. The coverage audit (Task 1.2) makes the
  table the enforced source rather than documentation.
- **ARCH-MOCK.**
  - **Seam:** `ProvisionIO` (git, sdlc, weave) plus the filesystem.
  - **Fakes:** `ProvisionFixture` is real git. Today its weave fake is in-process in
    `ProvisionFixture.Run` (`provision_git_test.go:99`, a call counter plus `FailWeave`). It is
    extended in place into a stateful model: a source table, cloning declared substrates whose
    source it knows, honoring `.weave-setup.lock`, and printing weave's real error lines through
    the same `ProvisionIO` result. A separate `testdata/fakeweave` script (new) is used only by
    `TestSetupFailureThroughRealSeam`, which needs a real process behind `OSProvisionIO`.
    `SlotWorld` is the in-memory model for the pair domain. It must reach the same terminal class
    as real git on every single perturbation (Task 2.2), not only on the acceptance cases.
  - **Live conformance:** `TestWeaveConformance` runs only when `weave` is on PATH. It checks a
    held lock → retryable, and a sourceless substrate → handoff with the captured line.
- **ARCH-CONSTRAINTS.**
  - **Interaction path:** startup of open/resume/add slot, with the operator waiting.
  - **Healthy-slot observe budget:** ≤ 150 ms over today's reuse path. Today's path costs
    `sdlc workspace --json` + 2 `rev-parse` + a marker read. Added: `worktree list`,
    `for-each-ref`, 2 `config --get-all`, a non-creating flock probe, deps file reads and one
    `stat` + `rev-parse` per dependency. No user-data measurement (rule 2). Basis: assumption.
    Measured in Task 2.6, which logs the observe time of the healthy fixture and fails above
    500 ms.
  - **Discover stays cheap; `ObserveSlot` is lazy.** `Discover` runs from at least nine sites
    (`slotcontext.go`, `slotstart.go`, `slotrecovery.go`). It gains only the number union (one
    `readdir`, one `worktree list`, one `for-each-ref`: three calls per repository, not per
    slot) and keeps today's per-candidate verification. A full `ObserveSlot` runs only for the
    one slot being opened, resumed, rebooted, allocated or shown. Per-Discover budget: today's
    cost + ≤ 30 ms, measured with 10 slots in Task 2.5. The recovery report observes every
    `:1+` slot, which is a batch path already paying sdlc's ~6 s; budget ≤ 150 ms per slot,
    measured with 10 slots in Task 3.4.
  - **Compile:** bounded by the existing 20 min `SetupTimeout`.
  - **Saves:** a rename, O(1) and with no copy. Capped at 16 entries per slot (`Stop` at the
    limit); disk use is the set-aside trees themselves, bounded by `RetentionPeriod` collection.
  - **Loop:** at most `len(SlotResources())+2` passes.
  - Other categories N/A.
- **ARCH-ORDER.** The reconciler holds no state between runs. Each run rebuilds its view from
  the world (level-triggered), so a crash at any point is handled by running it again.
  - **Inside a run:** git and filesystem resources are observed and converged under the host
    creation lease (nonblocking; busy → retryable). The lease is released for `weave compile`,
    and setup's marker is written only after a re-observation under the lease.
  - **Second actors:** another Couch operation on the same slot is serialized by the operation
    queue (same process) or by the lease (other processes). A concurrent `weave` holds
    `.weave-setup.lock` → retryable stop. `SetAside` probes that lock without `O_CREAT` and
    holds it across the re-check and the rename (`flock` on the existing file). If the file is
    absent, there is no setup to race.
  - **Stale observations:** each converge step re-checks its own precondition at execution time
    (e.g. `update-ref` with the zero-OID CAS, `worktree add` failing if the path appeared).
    The only removing step, `SetAside`, re-observes agent sessions and the checkout's broken
    evidence immediately before its rename (rule 3). A test injects an agent appearing
    between observe and apply. Pair never takes `.weave-setup.lock`, so the re-check narrows the
    window but cannot close it. A race costs nothing, because the rename preserves every byte and
    a running agent keeps its open files.
  - **Most likely to be mishandled:** a crash between creating the entry directory and the
    rename, or between the rename and the manifest write. In the first case the dependency is
    still broken, so the next run sets it aside under a new id, and the empty entry is collected
    as abandoned. In the second the tree is safe and the entry's id (`<dep>-<UTC timestamp>`)
    names it. A manifest-less entry is listed by `--show` and collected by directory age like
    any other.
  - **Tests:** `SlotWorld` crash injection after every step reproduces each ordering
    deterministically.
- **ARCH-FUNERAL.**
  - **saved-work entries:** created by `SetAside`. The last reader is the operator restoring
    work. Removed by the archive GC pass after `storagegc.RetentionPeriod` (one year since pair#393; measured from
    the entry's `saved_at`, else its directory mtime). Bounded at 16 entries, so disk use per
    slot is at most 16 × the largest dependency; at the cap,
    reconcile stops rather than evicting. The GC pass only visits registered stores
    (`CouchReferences.Snapshot`, `gcruntime/references.go:14`), so `SetAside` registers the
    slot store with `Coordinator.RegisterStore` (`storagegc/stores.go:122`) before the rename. A store recreated lazily after
    an env reset is therefore still collected.
  - **The attempt memo (R5):** one file per registration, overwritten in place, removed with
    the admin directory.
  - **The legacy intent family:** removed by reconcile (R1). Nothing creates it any more.
  - **The marker:** lives in the registration's admin dir and dies with it.
  - **Nothing else new is durable.**
- **ARCH-SECURE.**
  - **Paths:** every path comes from `SlotLayout` (repo name + number from `conventionalSlot`),
    never from a request.
  - **Removal:** each removal target passes `provisionSafePath`, must be a direct child of the
    env, and is refused if it is a symlink.
  - **`construct/deps`:** another repository's file, so untrusted. It is read with a 64 KB
    bound, and a row whose path resolves outside the env is `unknown` (never cloned or removed).
  - **Set-aside:** a rename within the env; the target is built by `SlotLayout`, and the source must be a direct, non-symlink child of the env.
  - **The marker:** validated as today. Read errors → unknown.
  - No credentials are touched.

---

## Chunk 1: M1 — the resource model, observation and `--show`

No convergence changes in M1: `Ensure` keeps its behavior. M1 makes the model real, observed and
visible, and enforces it as the single source.

### Task 1.1: `SlotLayout` — one path authority

**Files:** create `slotlayout.go`, `slotlayout_test.go`. Modify every site that builds or parses
slot state (from the review's survey; Task 1.2's token audit proves the list complete):
`slotcatalog.go` (`conventionalSlot`, `slotDirectoryNumber`), `provision.go` (`slotHostPath`,
the intent path, the marker path built from `admin` in `readSuccess` and `Ensure`),
`provision_lock_unix.go:29` (`couch-workspaces`), `slot.go:40` (`validateLocation`),
`workspace_identity.go:96,131` (the `main-slot` parse), `slotgit.go:37` (`RestingBranch`),
`recoverplan.go:883` (the `-slot` cut), `recoverplan_fake.go:111`, `couchsingleton/inspect.go:223`,
`threadstore_layout.go:19` (`.couch`), and the
`slotForContainedPath`, `conventionalSlotFromPath` and `conventionalSlotOfMember` parsers.

- [x] **Step 1: Failing tests.**
  - `TestSlotLayoutPaths`: for primary `/f/pair`, N=3, every method returns the documented path
    (env, host, store, saved-work root, setup lock, intent, resting branch
    `main-slot3`, marker name).
  - `TestParseSlotPathRoundTrip`: for every layout path of N ∈ {1, 12} and for paths inside the
    host, `ParseSlotPath` returns (primary, N). Paths outside → false. N=0 → false.
    `primary-slotX` → false.
- [x] **Step 2:** run `go test ./cmd/internal/couchcore -run 'SlotLayout|ParseSlotPath'`. Expect
  FAIL (undefined).
- [x] **Step 3:** implement. Replace `slotDirectoryNumber`, `slotForContainedPath`,
  `conventionalSlotFromPath` and `conventionalSlotOfMember`'s parsing with `ParseSlotPath`.
- [x] **Step 4:** `go test ./cmd/internal/couchcore` passes. Mutation: make `ParseSlotPath`
  accept N=0, and the round-trip test must fail.
- [x] **Step 5:** commit `#387 M1: couchcore: SlotLayout is the one slot path authority`.

### Task 1.2: The resource table and its coverage audit

**Files:** create `slotresource.go`, `slotresource_test.go`, `slotresource_coverage_test.go`.

- [x] **Step 1: Failing tests.**
  - `TestSlotResourcesFormADAG`: every `DependsOn` names a declared resource. There are no
    cycles. `TopoOrder()` is stable and puts every resource after its dependencies.
  - `TestSlotResourceKinds`: `store` and `branch` are never removable (`Removable()` false).
    Every removable resource is `derived`. Every resource with a user-data measure has a save
    rule.
  - `TestEverySlotStateSiteIsAResource` (the derived list, satisfying Done-when bullet 1). It
    parses every non-test `.go` file under `cmd/` with `go/parser`, as
    `artifactpath/coverage_test.go` does. The oracle is the **code's own string literals**, not
    a list of sites:
    - (a) **Token audit.** Every string literal (and every constant it reaches) containing a slot
      state token appears only in `slotlayout.go`. Tokens: `-slot`, `main-slot`, `.couch`,
      `couch-setup-success.json`, `couch-setup-attempt.json`, `.weave-setup.lock`,
      `couch-workspaces`, `creation.json`, `saved-work`. Any other file
      fails the test, naming the file:line. Tokens match at a path or ref boundary (`/`, start,
      end, or `N` digits for `-slot`), not as substrings: `"unknown-slot"`, `"reuse-slot"`,
      `"add-slot"` and `".couch-provision-*"` are not slot state. Each remaining exception is an
      allowlist entry with a reason string, asserted non-stale. Non-test fakes in production
      files (`recoverplan_fake.go:111`) move their literals to `SlotLayout` like any site. The
      token list is the one hand-written piece: a token names a *kind* of state, so adding state
      without a token is caught by (b).
    - (b) **Path-root audit.** Any `filepath.Join`/`+` outside `slotlayout.go` whose first
      operand is a `SlotIdentity`/`WorkspaceIdentity` field (`EnvironmentRoot`,
      `WorktreeRoot`, `FleetRoot`, `PrimaryRoot`, `RepoIdentity`) or a `SlotLayout` result
      fails. The allowlist holds only paths that are not slot state (e.g.
      `RelativeFamilyPath`), each with a reason string.
    - (c) **Layout↔table bijection.** Every `SlotLayout` method (from its `reflect` method set)
      is the `Location` of exactly one resource, or is on the documented external list. This
      half is consistency, not discovery; (a) and (b) are the discovery.
    - (d) Every git invocation whose argv contains `worktree`, `update-ref` or a `branch.`
      config key, or whose args are built from `SlotLayout.RestingBranch()`, lives in
      `slotobserve.go`, `slotconverge.go`, `slotcatalog.go` or `slotgit.go`. It matches the
      built expression, not a `refs/heads/main-slot` literal, which never appears.
- [x] **Step 2:** FAIL. The audit lists today's sites, i.e. Task 1.1's enumeration plus anything
  it missed. Record any miss in the Log.
- [x] **Step 3:** implement the table exactly as in Core concepts, and move each listed site
  onto `SlotLayout` (for `threadstore_layout.go`, `newSlotThreadStore` takes `layout.Store()`).
- [x] **Step 4:** PASS. Mutation (a): add `"x-slot2"` in `reboot.go`, and the audit must fail.
  Mutation (b): add `filepath.Join(slot.EnvironmentRoot, "x")` in `reboot.go`, and the audit
  must fail. Mutation (c): add an unused `SlotLayout.Foo()`, and the audit must fail. Revert
  each from a `cp` byte copy (lessons: never `git checkout <file>`).
- [x] **Step 5:** commit `#387 M1: couchcore: the slot resource table, enforced by an AST audit`.

### Task 1.3: Declared dependencies through ariadne's `layergraph`

**Prerequisite (met):** ariadne#294 (`DeclaredSubstrates`) and ariadne#295 (typed
`NotLayerError`, returned by `declaredGraph`; the walk still stops there) are on ariadne
`main`; pin `b9bc9f32` (the #295 merge) or later.

`DeclaredSubstrates` errors for the whole walk when a *present* substrate has no
`construct/base.manifest` ("present but not a compilable layer", an untyped `fmt.Errorf`).
A half-cloned or gutted dependency is exactly that, so an untyped error would make `deps`
unknown and stop the repair. Pair must not match error text. Resolution (operator choice,
2026-10-05): a typed `*layergraph.NotLayerError{Path, Owner}` from ariadne#295,
read with `errors.As` → that `dep:<path>` is a set-aside candidate, judged by R2's git evidence.

**Files:** `go.mod`/`go.sum` (`go get github.com/xianxu/ariadne@b9bc9f32`; no `replace`);
create `slotdeps.go` (an `osFS` adapter implementing `layergraph.FS`, with reads bounded by
`layergraph.ReadDeclaration`'s 64 KB limit; `DeclaredDeps(host) ([]DeclaredSubstrate,
error)`) and `slotdeps_test.go`.

- [x] **Step 1: Failing tests** (temp directories, no git needed):
  - host declares `substrate ../a src`, `a` declares `substrate ../b`, and `b` is absent →
    `a` present, `b` absent with owner `a`;
  - a `data` row is not a dependency;
  - a malformed row → error (mapped to `deps` unknown);
  - a present substrate without a base.manifest → that `dep:<path>` is broken-candidate, and the
    others are still reported (via the typed error);
  - a row resolving outside the env → `deps` unknown (a pair-side check on the returned paths);
  - an absent host `construct/deps` → no dependencies.
- [x] **Step 2–4:** red → green.
- [x] **Step 5:** commit `#387 M1: couchcore: declared dependencies from ariadne layergraph`.
  Verify `make build` and the Homebrew-style module fetch: `GOFLAGS=-mod=mod go build ./...`
  in a clean `GOMODCACHE`.

### Task 1.4: `ObserveSlot` (the shell) and its states

**Files:** create `slotobserve.go`, `slotobserve_test.go`. Modify `provision_git_test.go`
(`ProvisionFixture` gains helpers that break each resource) and `testdata/fakeweave` (stateful,
see ARCH-MOCK).

- [x] **Step 1: Failing tests** on `ProvisionFixture` (real git). Each starts from a slot
  provisioned by today's `Ensure`, breaks exactly one resource, and asserts that resource's
  observation (and that every other resource stays `present`):
  - env deleted (whole directory) → `env` absent, `registration` stale, `branch` present,
    `host` absent;
  - host deleted only → `host` absent, `registration` stale;
  - marker deleted → `setup` absent;
  - marker edited to another slot → `setup` broken;
  - a dependency directory deleted → `dep:<rel>` absent, `setup` absent (desired includes every
    dep);
  - a dependency replaced by a plain directory (no `.git`) → `dep:<rel>` broken;
  - a dependency whose `.git` is corrupt (`rev-parse` says "not a git repository") →
    `dep:<rel>` broken;
  - a dependency with a held `index.lock`, or a git script failing `rev-parse` with any other
    error → `dep:<rel>` **unknown**, never broken (positive evidence only, R2);
  - a valid marker plus an attempt memo whose digest matches → `setup` `present-with-warning`;
    a changed host `HEAD` → `setup` absent (memo stale);
  - `construct/deps` malformed → `deps` unknown, and no `dep:*` observed;
  - `.weave-setup.lock` flock-held by the test → `setup` carries `lock-held`;
  - `main-slotN` checked out in another worktree → `branch` broken;
  - `branch.main-slotN.merge` set to `refs/heads/other` → `upstream` broken;
  - a stale `creation.json` → `intent` present;
  - `git` replaced by a script failing `worktree list` → `registration` unknown (never absent;
    lessons: a failed probe is not absence);
  - a dirty dependency (modified tracked file + untracked file + local-only commit) → `dep`
    present. No user data is measured anywhere (rule 2);
  - a host whose `.git` file points at a missing admin directory and whose registration
    `git worktree repair` cannot restore → `host` broken (positive evidence), carrying the
    branch from the registration's admin `HEAD` when one is still readable.
- [x] **Step 2:** FAIL. **Step 3:** implement. Observations use `SlotLayout` and the
  `ProvisionIO` seam only. The flock probe opens without `O_CREAT`, and a missing lock file
  means free.
- [x] **Step 4:** PASS. Mutation: map a `worktree list` failure to `absent`, and the unknown
  case must fail.
- [x] **Step 5:** commit `#387 M1: couchcore: observe every slot resource`.

### Task 1.5: `PlanSlot` (pure) and the derived domain

**Files:** create `slotplan.go`, `slotplan_test.go`.

- [x] **Step 1: Failing tests.**
  - `TestPlanSlotDomain`: the domain is derived. Every resource from `SlotResources()` takes
    every state from `AllObservedStates()` plus that resource's declared sub-states (for
    `setup`: `present-with-warning`, `failed-known`, `lock-held`; enumerated by
    `SlotResourceSpec.SubStates`, so the domain cannot skip one), singly and in pairs (other resources
    present), with agent ∈ `AllAgentStates()` = {none, live, busy, unknown}. For each point the
    invariants hold:
    - I1: no step targets a resource that is converged;
    - I2: no step targets an `unknown` resource or anything depending on one;
    - I3: the only step that makes a `host` or `dep:*` absent is `SetAside`; no step deletes user data;
    - I4: no `SetAside` when agent is live, busy or unknown, for the host and every `dep:*`;
    - I5: `store` and `branch` never get a removing step;
    - I6: the step order respects `TopoOrder()`.
  - `TestPlanSlotNamedCases`: `tools:1`'s shape (host present, marker absent, `dep:ariadne`
    absent) → `[Compile]`. The deleted-slot shape (env absent, registration stale, branch
    present) → `[MkdirEnv, RemoveRegistration, WorktreeAdd, Compile]` across passes; asserted on the first
    pass's ready set. A valid marker with a missing dependency → `[Compile]` with
    `KeepOnFailure`.
  - `:0` → refusal.
- [x] **Step 2–4:** red → green. Mutations: drop I3's save step, or let I4 pass under a live
  agent. Each must fail the domain test (verify the mutation applied before trusting the
  result).
- [x] **Step 5:** commit `#387 M1: couchcore: PlanSlot over the derived domain`.

### Task 1.6: `couch --show` prints the slot's resources and plan

**Files:** `operationdispatch.go` (`show`: when the ref resolves to a slot address or a slot
path, also observe the slot and attach `SlotReport{Observation, Plan}`), `couchcmd/run.go`
(`render`), `run_test.go`. `--show repo:N` must work even when no thread record exists, as for a
deleted env.

- [x] **Step 1: Failing tests.**
  - `--show tools:1`-shaped fixture prints one line per resource in topological order
    (`env present`, …, `dep:ariadne absent`, `setup absent (no marker)`), then
    `plan: compile (weave compile in <host>)`.
  - Deleted-slot fixture prints `env absent`, `registration stale`, and the plan.
  - A converged slot prints `plan: nothing to do`.
  - An unknown resource prints `<id> unknown: <reason>` and `plan: stops at <id>`.
- [x] **Step 2–4:** red → green. Mutation: render from a hand-written resource list instead of
  `TopoOrder()`, and a test adding a resource must fail.
- [x] **Step 5:** commit `#387 M1: couch --show reports slot resources and the plan`.

### Task 1.7: Docs and close M1

- [ ] `atlas/couch.md`: a "Slot resources" section with the table (linking `slotresource.go` as
  the source), observation states and the plan. Keep `atlas/index.md` valid.
- [ ] Full verification (Chunk 4). Then `sdlc milestone-close --issue 387 --milestone M1`.

---

## Chunk 2: M2 — converge: `Reconcile` replaces `Ensure`'s repair logic

### Task 2.1: `Converge` steps (the shell)

**Files:** create `slotconverge.go`, `slotconverge_test.go`.

- [x] **Step 1: Failing tests** on `ProvisionFixture`, one per step, each run twice (second run a
  no-op or refused by its own precondition, never a duplicate effect):
  - `MkdirEnv`;
  - `RemoveIntent`;
  - `CreateBranch`: fetch, then `update-ref` with the zero-OID CAS. A concurrently created
    branch → the CAS fails → re-observe adopts it;
  - `SetUpstream`;
  - `RemoveRegistration`: `git worktree remove <hostPath>` for this slot's stale registration
    only. Assert that a **second stale registration** (another slot's, and a non-slot worktree
    whose directory is missing) survives with its admin directory and marker, and that a locked
    registration is a hand-off, not removed. A real-git conformance row pins the behavior: git
    removes only the named stale registration and exits 0 (verified on git 2.54 in review);
  - `WorktreeAdd`;
  - `RepairHost` (`git worktree repair`);
  - `Compile`, then `WriteMarker`.
- [x] **Step 2–4:** red → green.
- [x] **Step 5:** commit `#387 M2: couchcore: idempotent converge steps`.

### Task 2.2: The loop, `SlotWorld`, and the twice-run property

**Files:** create `slotreconcile.go`, `slotreconcile_test.go`, `slotworld_fake_test.go`.

- [x] **Step 1: Failing tests.**
  - `TestReconcileConvergesFromEverySinglePerturbationRealGit` (Done-when bullet 2, on the real
    boundary). It reuses Task 1.4's perturbation helpers: one per (resource, state) the fixture
    can produce, derived from `SlotResources()` × `AllObservedStates()`. A pair the fixture
    cannot produce is listed with a reason, and the list is asserted to shrink, never to grow
    silently. On real git, run Reconcile, then run it again. The first run reaches either
    `converged` or a `Stop`/handoff whose resource is the perturbed one (or depends on it). The
    second run's plan is empty or the same stop, and its effect log (git argv + filesystem
    mutations recorded by the `ProvisionIO` seam) is empty. The same perturbation in `SlotWorld`
    must reach the same terminal class (fake/real agreement on the whole single domain).
  - `TestReconcileConvergesFromEveryPairPerturbation`: the pair domain, run in `SlotWorld` only
    (it is too large for real git), with the same assertions. It is trusted because the single
    domain proved `SlotWorld` agrees with git.
  - `TestReconcileCrashAfterEveryStep`: inject a panic-equivalent abort after each executed step
    of the deleted-slot and dirty-broken-dep scenarios. Re-running converges with no duplicate
    effects. A crash after the rename produces exactly one entry holding the tree.
  - `TestReconcileNoProgressStops`: a world whose `WorktreeAdd` reports success but changes
    nothing → failure `no progress at registration`, within the pass bound.
- [x] **Step 2–4:** red → green. Mutations: remove the second-pass re-observation (use the first
  plan for every pass) → the crash test fails; make `SetAside` copy then delete instead of
  rename → the crash-mid-step assertion (no partial tree anywhere) fails.
- [x] **Step 5:** commit `#387 M2: couchcore: Reconcile, level-triggered and bounded`.

### Task 2.3: `Ensure` is `Reconcile`; intents and `NextHostAction` retire

**Files:** `provision.go`, `provision_host.go` (delete), `provision_request.go`,
`provision_git_test.go`, `provision_recovery_test.go`.

- [ ] **Step 1: Update tests to the new contract** (the deliberate behavior changes, each named):
  - `TestProvisionHostRefusesForeignPathOrBranch` splits. A foreign `main-slotN` is now adopted
    (decision 2). A non-directory at the env path still refuses (handoff).
  - The intent tests become: a stale `creation.json` is removed and creation proceeds.
  - New: a valid marker with a missing dependency and a compile that fails as handoff → `Ensure`
    returns `reused` plus a `Warning` carrying `ReconcileAdvice` (decision carried from #367
    Revision i). Without a marker → error.
  - `ProvisionResult` gains `Warning string` and `Saved []SavedWork`.
- [ ] **Step 2:** FAIL. **Step 3:** replace `Ensure`'s body with `Reconcile`, keeping the request
  validation and the primary identity check. Delete `NextHostAction`, `HostObservation`,
  `CreationIntent`, `validateCreationIntent`, `cleanupIntent` and `provisionDirIdentity` (if now
  unused). Sweep: `grep -rn "NextHostAction\|CreationIntent\|open again to retry" cmd/` must be
  empty.
- [ ] **Step 4:** PASS, including every existing provisioning test that is not a named change.
- [ ] **Step 5:** commit `#387 M2: couchcore: Ensure converges through the reconciler`.

### Task 2.4: Failures carry their resource and cause

**Files:** create `slotfailure.go`, `slotfailure_test.go`. Modify `resume_route.go`
(`withRebootAdvice` routes `*ReconcileError` through `ReconcileAdvice`), `reboot.go`, and the
start form's error path.

- [ ] **Step 1: Failing tests.**
  - `TestClassifyConvergeError` table, using weave texts captured from ariadne's source
    (revision recorded in a fixture header):
    - retryable: `environment setup is active…`, `ErrHostCreationBusy`, deadline, cancel;
    - handoff: `missing substrate … record its source in construct/deps`,
      `missing repository …`, a git auth failure, no `Error:` line;
    - a recognized line embedded behind an owner prefix still matches (substring).
  - `TestSetupFailureThroughRealSeam`: `OSProvisionIO` running the fake weave script yields the
    same class as the table.
  - `TestReconcileAdviceText`: handoff contains `ask the <repo>:0 agent` and neither `retry` nor
    `fix it`. Retryable contains `run it again`. Every class is covered (`AllFailureClasses`).
  - `TestWeaveConformance`: skipped without `weave` on PATH (see ARCH-MOCK).
- [ ] **Step 2–4:** red → green. Mutation: make handoff retryable, and the no-retry assertion
  must fail.
- [ ] **Step 5:** commit `#387 M2: reconcile failures name resource, cause and the :0 hand-off`.

### Task 2.5: Callers converge through it — add slot, open, resume, reboot

**Files:** `slotallocation.go`, `slotcatalog.go` (`Discover`), `slotstart.go`
(`resolveManagedStart`'s `StartCreate` block, `spawnManagedResolution`, `checkSlotCreation`),
`slotrecovery.go` (`selectedSlot`), `reboot.go` (`rebootSlot`), `reboot_decision.go`,
`actor_actions.go`, `recoverplan.go` (directory-missing class and hold).

**What changes, call site by call site:**
- **`selectedSlot`** (the funnel for open, resume, reboot and a reused-slot fresh start) calls
  `Ensure` (= `Reconcile`) **always**, not only when `!candidate.Verified || candidate.Err !=
  nil` (`slotrecovery.go:233`). `verifyHost` checks identity, not the marker or the clones, so
  today `tools:1` and a missing clone under a valid marker go straight to the agent. On a
  healthy slot this costs one `ObserveSlot` with an empty plan (the 150 ms budget). Its
  "not an existing conventional candidate" refusal stays only for numbers no source knows.
- **`selectedSlot` details.** `ReserveRepositoryFamily` (today inside the `!Verified` branch,
  `slotrecovery.go:233-237`) runs always, before reconcile. It is an idempotent family write
  that returns the existing family. `selectedSlot` returns the `ReconcileResult`, and each
  caller applies the outcome table above. Reboot observes twice (both `selectedSlot` calls); the
  second has an empty plan and is counted in the budget (≤ 2 × 150 ms).
- **`EnumerateSlotCandidates`** (`slotcatalog.go:66-68`, `checkSlotDirectory`) fails the whole
  inventory when an env path is a file or a symlink. It becomes a per-candidate `Err` (`env`
  broken → handoff for that number only), so no single slot refuses the whole repository.
- **`Discover`** gains the number union (env dirs ∪ registrations ∪ `main-slotN` refs; three
  calls per repository) and keeps today's cheap per-candidate verification. A number known only
  from a registration or branch becomes a candidate with `Err = ErrSlotNeedsReconcile` (typed),
  not "incomplete slot host".
- **Add slot.** `resolveManagedStart`'s `StartCreate` block (`slotstart.go:86-125`) has three
  repository-wide refusals. Each one's fate:
  - `SelectStartSlot`'s "slot N needs attention" for any candidate with `Err`: a candidate whose
    `Err` is `ErrSlotNeedsReconcile` (or that is unverified) is **reusable**, picked as the lowest
    free number with `Exists: true, Reconcile: true`. Any other `Err` (an identity mismatch, a
    probe error) skips that number only and adds a `Notice`.
  - "repository slot %d needs attention before creating another slot" (snapshot slot `Err`,
    `slotstart.go:106-109`): narrowed the same way. It skips that number and adds a notice.
  - "repository has an unreadable conversation": unchanged. It is about a thread record, not
    workspace state, and stays out of scope.
  
  `checkSlotCreation` (`slotstart.go:244`) refuses on unreadable/unknown rows; it is unchanged,
  for the same reason. A reusable allocation goes through the existing reuse route
  (`resolution.ReuseSlot` → `startFreshSlot(..., true, ...)` → `selectedSlot`), so it never
  reaches `spawnManagedResolution`'s create branch and its "number already exists →
  `ErrStartResolutionChanged`" check (`slotstart.go:278-284`). That check stays correct for the
  genuinely new number.
- **Reboot.** `rebootSlot`'s early refusal when the directory is missing (`reboot.go:203-210`)
  goes away. It calls `selectedSlot` (which reconciles), then observes the slot record, as today.
  In `DecideReboot` (`reboot_decision.go:57-100`), for `Slot: true`, the `!DirectoryPresent`
  branches become unreachable, because reconcile runs first and makes the directory present or
  fails with `ReconcileAdvice`. They are removed for slots and kept for `:0`
  (`RebootCheckoutMissing`). `RebootDirectoryMissing` is deleted. `ActorActions`
  (`actor_actions.go:59-66`) offers `reboot` for a `:1+` row with `DirectoryMissing`.
- **Recovery report.** `RecoverDirectoryMissing` / hold `directory-missing` for `:1+` becomes
  `reconcilable` in Task 3.4; until then it gains the step `[reboot]`, since reboot now
  reconciles.

- [ ] **Step 1: Failing tests.**
  - `TestDiscoverUnionsNumberSources`: numbers come from env dirs ∪ registrations ∪ `main-slotN`
    refs; registration/branch-only numbers carry `ErrSlotNeedsReconcile`. A 10-slot fixture
    measures Discover at ≤ today + 30 ms.
  - Add slot in a `tools`-shaped repository (no source for `ariadne`): the first attempt compiles
    once, refuses with the hand-off and writes the memo. A second attempt refuses with the same
    text and zero weave calls. Changing the root `construct/deps` makes the next attempt compile
    again.
  - `TestSelectStartSlotReconcilable`: a reconcilable candidate is picked as the lowest free
    number (`Exists: true, Reconcile: true`). Any other `Err` skips its number only and the
    allocation carries a `Notice`. With the domain of candidate errors derived from the typed
    set, no candidate error refuses the whole repository.
  - Add slot on the deleted-slot fixture picks N, reconciles it through `selectedSlot`, and
    starts. On today's code it refuses (red).
  - Open on a fixture whose host is **verified** but whose marker is missing → reconcile
    compiles before the agent starts (red today: `selectedSlot` skips `Ensure`).
  - Open/resume on the `tools:1`-shaped fixture (the weave fake knows no source for `ariadne`)
    returns `ReconcileAdvice` handoff text, and no agent is started (blocking: no valid marker).
  - `TestReconcileOutcomeTable`: for every (caller, severity) cell, a fixture produces it and
    asserts the caller's behavior. The severity domain is derived from `OutcomeSeverity` over
    Task 1.5's state domain. Named cells:
    - open with a held `index.lock` in a dependency of a live slot → attaches, with a warning;
    - open with a broken dependency under a live agent → attaches, with the `agent-live`
      warning, and nothing moved;
    - reboot of a live slot with a broken dependency → the agent is stopped, the dependency is
      set aside and re-cloned, and a fresh agent starts;
    - add slot whose lowest reusable number is blocking → refused with the hand-off; no other
      number is tried;
    - a leftover `main-slotN` checked out in another worktree → add slot refuses with the
      hand-off naming that slot.
  - An env path that is a file → only that number is skipped.
  - `rebootSlot` on a slot whose env is gone: reconcile recreates it, then a fresh start.
  - `DecideReboot`'s totality test: slot rows no longer produce `RebootDirectoryMissing`, and
    `:0` rows are unchanged.
  - `ActorActions`: a `:1+` `DirectoryMissing` row offers `reboot`.
- [ ] **Step 2–4:** red → green. Mutations: restore the repository-wide refusal in
  `SelectStartSlot` → the add-slot test fails; restore the `Verified` gate in `selectedSlot` →
  the verified-host open test fails; make every non-converged resource blocking → the
  `index.lock` attach test fails.
- [ ] **Step 5:** commit `#387 M2: add slot, open, resume and reboot converge through the reconciler`.

### Task 2.6: `couch --reconcile repo:N` and real-git acceptance (part 1)

**Files:** `ops.go` (`reconcile`: `ExecuteLiveOwner`, `EffectProcess`, `ConfirmNone`,
`ResultConsole`, args `path`, `repo-scope` implicit, `ref` positional), `operationdispatch.go`,
`couchcmd/cli.go`, `couchcmd/slot_operations.go` (socket admission like `reboot`), `slot_operation.go`,
`run.go` (render the `ReconcileResult`), and `slotreconcile_acceptance_test.go`.

- [ ] **Step 1: Failing acceptance tests**, using real git + fake weave through
  `CouchLiveOwnerExecutor`:
  1. Deleted slot directory with leftover registration and `main-slotN` (the issue's original
     case). Add slot for the repository succeeds, at N. `--reconcile` on N is then a no-op.
  2. Interrupted setup like `tools:1` (no clone, no marker). With a known source the run
     converges and the marker is valid. With no source the output carries weave's
     missing-substrate line and the `:0` hand-off; a second run gives the same output and has
     no effects.
  3. A missing dependency clone under a valid marker: it converges (recompile). With a failing
     source the slot stays usable, with the warning.
  - Each case: `SlotWorld` given the same starting perturbation reaches the same terminal
    class (fake/real agreement).
  - Healthy-fixture observe time is logged; the test fails above 500 ms (ARCH-CONSTRAINTS).
- [ ] **Step 2–4:** red → green.
- [ ] **Step 5:** commit `#387 M2: couch --reconcile and acceptance for deleted, interrupted and missing-dep slots`.

### Task 2.7: Docs and close M2

- [ ] README (Couch, slot recovery), `atlas/couch.md` (reconcile loop, failure classes,
  `--reconcile`), and the couch skill (`cmd/internal/couchcmd/skills/couch/SKILL.md`): a reconcile handoff goes to the
  `:0` agent. Sweep retired vocabulary in prose and strings: `grep -rn "add slot recreates\|open
  again\|creation intent\|setup conflict" cmd/ skills/ README.md atlas/`.
- [ ] Full verification (Chunk 4). Ask the operator to smoke-test live (pair:0 rebuilt per
  AGENTS.local.md): `couch --show tools:1`, then `couch --reconcile tools:1` showing weave's line
  and the hand-off. Then fix `tools/construct/deps` (add the URL; tools' own workflow) and
  reconcile again.
- [ ] `sdlc milestone-close --issue 387 --milestone M2`.

---

## Chunk 3: M3 — set aside a broken dependency, saved-work lifecycle, recovery report

### Task 3.1: `SetAside` and the saved-work entry

**Files:** create `slotsave.go`, `slotsave_test.go`.

Entry layout: `<env>/.couch/saved-work/<id>/{tree/, manifest.json}`, with `<id>` =
`<checkout-name>-<UTC timestamp>` (the host's name is the repository's). The steps, in order:
1. `RegisterStore` (GC visibility).
2. Refuse at 16 entries.
3. `mkdir` the entry, then write `manifest.json` via `durablefile` with `state: pending`,
   `saved_at`, the slot address, the checkout path, its kind (`host`/`dep`), for the host the
   recorded `branch`, and the restore command `mv <entry>/tree <path>` (after moving the
   recreated checkout aside).
4. Open the existing `.weave-setup.lock` without `O_CREAT` and take a nonblocking `flock`;
   held → retryable. Hold it through steps 5–6; an absent lock file means no setup to race.
5. Re-check agent sessions and the broken evidence; a change aborts.
6. `rename(checkout, entry/tree)` through the injected `SetAsideIO.Rename` seam (the crash-injection
   point for tests and `SlotWorld`).
7. Rewrite the manifest with `state: complete`.

A crash at any point leaves a manifest naming the checkout. `pending` with a `tree/` means
the move happened; `pending` without one means it did not. `--show` lists both, and GC
collects by `saved_at`. Disk use per slot is bounded by 16 × the largest dependency
(ARCH-FUNERAL).

Every path comes from `SlotLayout`. The checkout must be a direct child of the env and not
a symlink (`provisionSafePath`).

- [x] **Step 1: Failing tests** (real git):
  - a dependency with a local-only commit, a dirty tracked file, an untracked file and an
    ignored file, made broken by removing `.git/HEAD` → after `SetAside`, the dependency
    path is absent and `entry/tree` holds every file byte-identical, with mtimes kept. After
    restoring `.git/HEAD` in the tree, `git log` shows the local-only commit;
  - 16 existing entries → `Stop{saved-work-full}`, nothing moved;
  - a symlinked dependency path → handoff, nothing moved;
  - the lock held by the test → retryable, nothing moved;
  - an agent session appearing at step 5 (injected through the session-probe seam) →
    `agent-live`, nothing moved; the empty entry is later collected as abandoned;
  - the store's registration is visible to `CouchReferences.Snapshot` after the call;
  - the host, broken on positive evidence and on an issue branch: the manifest records that
    branch, read from the admin `HEAD` before the rename;
  - crash injection at the `Rename` seam for both kinds: no partial tree anywhere, and the
    manifest names the checkout.
- [x] **Step 2–4:** red → green. Mutation: implement step 6 as copy+delete through the same
  seam with an abort injected between them, and the test that no partial tree exists anywhere
  must fail.
- [x] **Step 5:** commit `#387 M3: couchcore: SetAside moves a broken checkout into saved work`.

### Task 3.2: Reconcile sets aside, then recreates (host re-add, dependency re-clone)

**Files:** `slotconverge.go`, `slotreconcile.go` (the step was planned in M1; now executed).

- [ ] **Step 1: Failing tests:**
  - a broken dirty dependency → set aside, then re-cloned by compile. The result lists the entry
    and its restore command;
  - the same under a live agent → `Stop{agent-live}`, which callers treat as degraded (Task
    2.5's outcome table), and nothing moved;
  - a broken host on an issue branch → set aside, the registration removed, re-added on the
    same issue branch (asserted from git, and named in the result); the branch's commits are
    intact in the shared repository;
  - the recorded branch checked out in another worktree → re-added on `main-slotN`, with the
    result saying so;
  - a broken host under a live agent → `Stop{agent-live}` (degraded): open attaches, and nothing
    moves;
  - a second run → no-op.
- [ ] **Step 2–4:** red → green.
- [ ] **Step 5:** commit `#387 M3: reconcile repairs a broken dependency without losing work`.

### Task 3.3: Saved-work collection

**Files:** `archive_gc.go`, the gcruntime pass that drives archive detach
(`gcruntime/references.go`), tests.

- [ ] **Step 1: Failing tests:** an entry with `saved_at` older than `storagegc.RetentionPeriod`
  is removed by the GC pass. A younger one stays. An entry without a manifest older than the
  period is removed as abandoned. The pass tolerates a slot with no `saved-work/`. Removal is
  under the store lock, through the same write path as archive detach.
  - A slot whose store was recreated lazily after an env reset, and whose only content is a
    saved-work entry, is visited by `CouchReferences.Snapshot` (because `SetAside` registered
    it) and the old entry is collected. On today's registration path this is red.
- [ ] **Step 2–4:** red → green. **Step 5:** commit `#387 M3: saved-work entries expire with archive retention`.

### Task 3.4: The recovery report reads the reconciler's observations

**Files:** `recoverplan.go`, `recoverplan_source.go`, `recoverplan_test.go`.

- [ ] **Step 1: Failing tests.**
  - `RecoverPlanInput` carries one `SlotPlan` per present `:1+` candidate (gathered in the shell
    via `ObserveSlot`).
  - The class is derived from the same source as the callers, `OutcomeSeverity` plus the stop
    reason, through one pure `RecoverSlotClass(SlotPlan)`:
    - a blocking resource, a handoff failure or a `failed-known` memo → class `slot-needs-:0`,
      with `ReconcileAdvice`;
    - `Stop{agent-live}` → step `[reboot]` (reboot repairs after stopping the agent);
    - degraded `unknown` (e.g. a dependency's `index.lock`) → a note, with no step;
    - a non-empty plan with no stop → class `reconcilable`, step `[reconcile]` (emitting
      `couch --reconcile repo:N`);
    - an empty plan → unchanged.
  - `directory-missing` for `:1+` maps to `reconcilable`.
  - The totality test's domain is `RecoverSlotClass` over Task 1.5's state domain × agent states
    (derived); class/hold/note coverage is re-derived.
  - `TestRowAdviceNamesOnlyReachableActions`'s sweep includes `reconcile`.
- [ ] **Step 2–4:** red → green. Mutations: map `agent-live` to `slot-needs-:0` → the totality
  test fails; map degraded unknown to `reconcilable` → it fails.
- [ ] **Step 5:** commit `#387 M3: recovery report reads slot plans`.

### Task 3.5: Acceptance (part 2) — the dirty slot

**Files:** `slotreconcile_acceptance_test.go`.

- [ ] A dirty slot whose broken dependency must be recreated. The host has a dirty file (it must
  be untouched: a readable host is never set aside). The dependency has a local-only commit, a dirty file
  and an untracked file, and is made broken by removing `.git/HEAD` (git then answers "not a
  git repository": positive evidence). Reconcile saves the dependency first (entry verified as
  in Task 3.1, including the local-only commit after restore), recreates it via compile, and
  leaves the host's dirty file byte-identical. A second run is a no-op with no new entry.
- [ ] A host made unreadable (its `.git` file points nowhere and `git worktree repair` cannot fix
  it), checked out on an issue branch with a dirty file → set aside whole, re-added on the same
  issue branch; the set-aside tree holds the dirty file. Under a live agent → `agent-live`,
  nothing moved.
- [ ] The same dependency with a held `index.lock` instead → `unknown`, nothing saved or moved,
  and the output names `dep:<rel>` as unobservable.
- [ ] A foreign, pre-populated env (R1): a non-git `ariadne/` with files at the env path before
  slot creation → adopted. The dependency is saved, then replaced by the clone, and the entry
  restores the foreign files.
- [ ] An agent session appearing between observe and `SetAside` (injected through the session
  probe seam) → the rename aborts with `agent-live`; nothing is moved, and the saved entry stays.

### Task 3.6: Docs, issue revision, close

- [ ] Confirm the issue's Done-when still matches what shipped (the R2 revision landed with plan
  approval); append a Revision for any delta.
- [ ] README, `atlas/couch.md` (saved-work entry, restore commands, retention), and the skill
  (restoring saved work). Run `sdlc issue sync --issue 387`.
- [ ] Full verification (Chunk 4); operator live smoke on a scratch slot (break a dependency, make
  it dirty, `couch --reconcile`, restore from the entry).
- [ ] `sdlc milestone-close --issue 387 --milestone M3`, then `sdlc close --issue 387 --verified '…'`.

---

## Chunk 4: Verification recipe (every milestone close)

Per memory "Run full make test before sdlc close": run each into a file, never piped to `head`.

```bash
S=$TMPDIR/387-verify; mkdir -p $S
make -k test > $S/make-test.log 2>&1; echo "make: $?"
TMPDIR=$S make test-changelog > $S/changelog.log 2>&1; echo "changelog: $?"
go test ./... > $S/go-test.log 2>&1; echo "go: $?"
go test -race ./cmd/internal/couchcore/... > $S/race.log 2>&1; echo "race: $?"
```

Expected: all 0, except the three Go tests known to fail on main (compare the failing set against
`main` before claiming a regression). Pty-child tests need the sandbox off.

---

## Revisions

### 2026-10-05 — plan review round 1 (fresh-context reviewer, 9 blocking)

Delta, one line per finding:
1. Registration: targeted `git worktree remove <hostPath>`, never repository-wide `prune`; a
   test proves a second stale registration survives (table, Task 2.1).
2. A dependency is broken only on positive evidence (no `.git` / "not a git repository"); other
   git errors are unknown. Save is always a deterministic whole-tree tar with a
   content-addressed id (R2, rule 2, Tasks 1.4, 3.1, 3.5).
3. `selectedSlot` reconciles always, not only for unverified candidates; the add-slot route and
   the fate of each repository-wide refusal are named; reboot, `DecideReboot` and
   `ActorActions` changes are spelled out (Task 2.5).
4. Agent liveness is re-checked under `.weave-setup.lock` immediately before the rename, with an
   injected-ordering test (rule 3, ARCH-ORDER, Task 3.5).
5. R1 states plainly that protection against non-Couch actors is dropped, asks for operator
   sign-off, and adds a foreign-env test.
6. The coverage audit's oracle is the code's string literals and path-root expressions, with
   the layout↔table bijection kept only as a consistency check; Task 1.1 enumerates the extra
   sites.
7. Discover gains only the per-repository number union; `ObserveSlot` is lazy, with
   per-Discover and recovery-report budgets measured at 10 slots.
8. The twice-run convergence property runs on real git for the whole single-perturbation domain,
   with `SlotWorld` agreement; pairs stay in `SlotWorld`.
9. `SaveWork` registers the slot store so GC visits it; tested on a lazily recreated store.

Non-blocking items adopted: the `KeepOnFailure` loop ending plus the attempt memo (R5);
Done-when revised now (issue Revision b); the weave fake is extended in place, plus one new
script for the real seam; the skill path; the full `Ensure` caller list; corrected loop
pseudocode; `busy` in the agent domain; no user-data measurement on the healthy path; the deps
parser import-vs-copy question raised for the operator.

### 2026-10-05 (b) — plan review round 2 (2 blocking)

1. A stop in reconcile would have blocked open, resume and reboot of a running slot (a held
   `index.lock` in a dependency, or `agent-live`). Added `OutcomeSeverity` (blocking for
   checkout resources and never-completed setup; degraded otherwise) and a per-caller outcome
   table. Reboot's first pass tolerates `agent-live`; its post-stop pass repairs. Tests per cell
   (Task 2.5).
2. The whole-tree tar with a 64 MB cap could not save the real `ariadne` clone (65 MB). Replaced
   by `SetAside`: an atomic rename into `<env>/.couch/saved-work/<id>/tree`, with no size
   cap, no tar, no trash directory and no content addressing (R2, R4, rule 2, Tasks 3.1–3.2).
   The issue's Done-when follows in its own Revision.

Non-blocking items adopted: R5 memoizes handoff failures only and names `--reconcile` as the
override; the token audit matches at boundaries, with a reasoned allowlist; Task 1.1 lists
`recoverplan_fake.go:111` and `couchsingleton/inspect.go:223`; `EnumerateSlotCandidates`
downgrades a file or symlink env to a per-candidate error; `ReserveRepositoryFamily` runs
always; reboot's double observation is counted in the budget; the agent re-check is stated to
narrow, not close, the race (harmless under rename).

### 2026-10-05 (c) — plan review round 3 (2 blocking)

1. Add slot's "next number" had no mechanism. Allocation is decided at preview, so add slot now
   refuses a blocking slot with the hand-off. R5's memo now also covers slots without a marker
   (`failed-known`), and `SelectStartSlot` excludes such numbers at preview, so the next add
   slot takes a fresh number through the existing create branch. There is no retry loop.
2. The recovery report's class is derived from `OutcomeSeverity` via `RecoverSlotClass`:
   `agent-live` → `[reboot]`, degraded unknown → a note, blocking/handoff → `slot-needs-:0`.

Non-blocking items adopted: the tar/trash leftovers are removed (`Converge` wraps, ARCH-SECURE,
`SweepTrash`); the manifest is written `pending` before the rename and `complete` after; the lock
is held across the re-check and the rename; the crash seam is named (`SetAsideIO.Rename`); the
disk bound is stated.

### 2026-10-05 (d) — plan review round 4 (1 blocking)

The preview exclusion missed `tools:1`, a *verified* candidate, and could not apply the
memo's digest from one file read. The reviewer's bounding note then showed that the exclusion
was wrong in principle: `construct/deps` reaches every slot through `main`, so a new number
fails identically. The exclusion is removed. Add slot refuses with the hand-off, and the memo
(written before the refusal returns) makes repeat attempts immediate. `failed-known` and the
other `setup` sub-states are enumerated through `SlotResourceSpec.SubStates`, so the derived
domains include them.

### 2026-10-05 (e) — plan review round 5: approved

No blocking issues. Adopted: the add-slot paragraph states that a slot-local hand-off still
refuses add slot for the repository (deliberately), with an outcome-table cell for a
`main-slotN` checked out elsewhere; the operator sign-offs are listed at the top.

### 2026-10-05 (f) — operator sign-offs: R1 reworded, depend on ariadne

R1 is accepted with the wording "Couch stops refusing slots it did not itself create" (Couch is
a singleton; the non-Couch content is legacy, interrupted, lost-track or hand-made). The deps
parser is imported from ariadne `layergraph`, pinned with no `replace`. Because `Walk`
present-skips absent substrates, ariadne first exports `DeclaredSubstrates`, as its own issue;
Task 1.3 is rewritten around it, and the copied parser and conformance table are dropped.

### 2026-10-05 (g) — host set-aside, one-year retention, ariadne#294 landed

- The host checkout follows the same rule as a dependency (operator): set aside only on
  positive evidence after `git worktree repair`, then re-added on its recorded branch; a stale
  checkout is never touched. Table, R2 and Task 3.5 updated.
- pair#393 raised `storagegc.RetentionPeriod` to one year; saved-work collection follows it.
- ariadne#294 landed (`e76aac11`); Task 1.3 pins it. Its error for a present substrate without
  base.manifest is untyped, so a typed error is proposed as a small ariadne follow-up.

### 2026-10-05 (h) — plan-quality gate PQ-1 (Important)

Revision g put the host set-aside into the table and Done-when, but not into the machinery. The
class is fixed: one `SetAside(checkout)` step for the host and every `dep:*` (rule 2, I3, I4,
R1, ARCH-ORDER, Tasks 1.4, 3.1, 3.2). The host's branch is recorded in the manifest before the
rename, and `WorktreeAdd` picks the stale registration's `HEAD`, then the manifest's branch,
then `main-slotN`. A stop's severity follows its reason, so `agent-live` is degraded even on a
host. Minors: Task 1.5's `Prune` → `RemoveRegistration`; a Non-goals section. The test-case
lists in Tasks 1.4/2.5/3.1 stay as written: each is a derived perturbation or a named
acceptance cell that the Done-when calls for, not free-form enumeration.

### 2026-10-05 (i) — plan-quality advisory: the old subject swept

Every occurrence of the dependency-only hold now names the host as well: the agent row of the
table, and `OutcomeSeverity`, which gains `stopReason` so `agent-live` is degraded on any
resource.

### 2026-10-05 (j) — Task 1.3 implementation reconciliation

- `layergraph.OSFS` already reads declarations safely (no-follow, ordinary file, 1 MiB
  `DeclarationLimit`), so there is no pair-side 64 KB adapter; `DeclaredDepsOf(env, host)`
  calls `DeclaredSubstrates(layergraph.OSFS{}, host)` directly.
- Deviation, decided at implementation: a dependency that does not live directly in the slot
  environment is reported with `Outside: true` (not slot state, weave's to manage, never
  converged or set aside by reconcile) instead of making the whole declaration unknown. The plan's
  rule would have left any repository with such a row permanently unreconcilable. A present
  non-layer is `DeclaredDeps.NotLayer` (from ariadne#295's typed error); it carries `Outside` too,
  so an outside non-layer is never set aside.
- Pinned `github.com/xianxu/ariadne v0.0.0-20261005230324-b9bc9f32f5ae` (no `replace`);
  `make build` and a clean-`GOMODCACHE` `go build ./...` pass.

### 2026-10-05 (k) — Task 1.4 implementation reconciliation

- The agent input reuses the recovery report's `EvidenceAgent`, rather than a new enum;
  `AgentRunning(EvidenceAgent) (running, known)` is the one reading for reconcile (detached =
  running; parked/none = not; unusable = unknown). `AllAgentStates()` in Task 1.5 is therefore
  `AllEvidenceAgents()`.
- A resource whose dependency is not present is `StatePending` (the zero state, "waits for
  X"): neither a step nor a stop. It is not in `AllObservedStates()`; the planner skips it.
- The host is verified by git (`--absolute-git-dir` under `Registrations()`, common directory
  match), not `sdlc workspace --json`, so observation spends no sdlc call. Setup reads the
  marker through the new `ValidSetupMarker` (which `readSuccess` now shares), and a held weave
  lock outranks every marker reading.
- Observation needs no weave, so the stateful weave fake moves to M2 (Task 2.1/2.6), where
  compile runs. The attempt memo (R5) is observed when M2 starts writing it.

### 2026-10-05 (l) — Task 1.5 implementation reconciliation

- `RemoveRegistration` is replaced by `WorktreeAdd{Force: true, Branch: <the stale registration's
  branch>}`. `git worktree add --force` overrides "missing but already registered" in one step, so
  the branch the registration records is used before it can be lost. With a separate remove, the
  next pass would no longer see it, and a deleted slot on an issue branch would silently come back
  on `main-slotN`. A broken host's set-aside leaves its registration stale with the branch, so the
  same re-add serves it, and the host manifest no longer needs to carry the branch for the re-add
  (it still records it for the operator). Task 2.1's `RemoveRegistration` row becomes the forced
  re-add, plus a conformance row: git re-adds over the stale registration only, and a second stale
  registration survives.
- An invalid marker needs no separate remove: `Compile`'s marker write replaces it.
- `RepairHost` runs at most once per run (`PlanInput.Attempted`). A host still unreadable after it
  is set aside; a mismatched one is a hand-off.
- The table gained `DesiredAbsent` (the intent) and `BrokenSubs` (the broken readings each
  resource can produce), so the domain test is generated from it: 7 agent states × singles and
  pairs. Mutations caught: dropping the agent gate (210 I4 violations), setting aside a
  git-readable non-layer (I3), ignoring unknown blocking (I2). One mutation first "passed" only
  because it did not compile; it was re-run in a compiling form.

### 2026-10-05 (m) — Task 1.6 implementation reconciliation

- `show` now returns one type, `ShowResult{Threads, Slot *SlotReport}`, rather than a slice
  of threads; the slot section is present when the reference names a :1+ slot (a `repo:N`, a path
  at or inside a slot, or the one slot all matched threads start in). A slot with no thread (a
  deleted directory) no longer fails `show`: `ErrThreadReferenceNotFound` is tolerated exactly when
  the reference resolved to a slot.
- `Couch.SlotIO` (wired to `OSProvisionIO` in `couchcmd`) is the observation seam.
  `slotAgentEvidence` reads the rows through `agentEvidence`, extracted from the recovery
  report's `agentOf` so both read rows one way.
- `SlotPlanSummary` is the one plan text (shared with M2's reconcile result).
- Added `TestObserveSlotFollowsTheResourceOrder`: `ObserveSlot`'s order is pinned to
  `SlotResourceOrder()`, so the renderer, which prints in observation order, follows the
  table. Mutation (observe intent after host) caught.

### 2026-10-05 (n) — M1 verification reconciliation

- The repository audits shaped the surface: `artifactpath`'s exhaustive inventory now lists the six
  new files as non-artifact sources. The dead-symbol audit removed `SavedWork`, `SetupAttemptPath`
  and `SlotPlan.OnlyStops`, which M2/M3 re-add with their consumers. `AllObservedStates` is
  allowlisted as the plan domain's vocabulary (the `AllEvidence*` precedent). `SlotResource`,
  `Desired` and `SlotResourceOrder` gained production consumers: `PlanSlot` retires the intent
  through `Desired`, and sorts its steps by `SlotResourceOrder`, so I6 holds by construction.
- The stop reason's value is `live-agent` (constant `StopReasonAgentLive`); `agent-…` literals
  trip `artifactpath`'s agent-family vocabulary check. Prose keeps calling the hold "agent-live".
- The coverage token for saved work matches at a path boundary, so the stop reason
  `saved-work-full` is not slot state.

### 2026-10-05 (o) — M1 boundary review round 1 (FIX-THEN-SHIP)

- BR-2 (Important): `--show` now takes the slot's identity from git (`Discover`'s candidate, else
  the conventional location with git's common directory) after resolving the reference through
  symlinks (`retainedPhysicalPath`). Building it from the `<primary>/.git` guess and the unresolved
  path misread a healthy slot reached through a symlinked fleet as registration absent, host
  mismatched and branch elsewhere. A test now reproduces exactly that against the old construction.
  `ObserveSlot`'s doc states the precondition: a git-resolved layout.
- BR-3 (Important): a dependency is judged by `git rev-parse --show-toplevel == path`, not by where
  its git directory lives, so a gitfile-backed clone (a linked worktree, `--separate-git-dir`) is a
  readable checkout rather than "unreadable" (which would set it aside). Test added; the old check
  fails it.
- BR-4 (Important): the README describes `--show repo:N`, the resource lines and the plan line.
- Minors taken:
  - a held weave lock stops only when setup has work pending;
  - an unknown agent holds with its own reason, `unknown-agent`;
  - `--show repo:N` of a missing slot keeps `WorkspaceReferencePath`'s own error;
  - a hand-written I2 case (an unknown host blocks exactly deps/clone/setup) is independent of
    `SlotResourceDependents`.
- Amends Revision (j): when a present dependency is not a layer, the walk stops there, so the other
  declared dependencies are not reported until it is repaired. Task 1.3's "the others are still
  reported" does not hold (ariadne#295 chose the typed error, not partial results). The
  consequence is bounded: the not-layer dependency is a stop or set-aside, so setup never reads as
  converged past it.
- Task 1.7's atlas section lives in `atlas/workspace-provisioning.md` (the numbered-slot page), not
  `atlas/couch.md`.
- For M2 (Task 2.2): the loop's no-progress guard compares observations on state and reading only.
  `Reason` text from git can vary between runs, which would defeat the guard.

### 2026-10-05 (p) — Task 2.1 implementation reconciliation

- The steps are methods of `slotConverger`, which wraps the one `WorkspaceProvisioner` and reuses
  its git, `selectRemote`, `branchOID`, `configValue` and fetch-baseline helpers (ARCH-DRY).
  `RemoveRegistration` is gone (Revision l). The stale-registration row is
  `TestConvergeWorktreeAddOverAStaleRegistration`: the host comes back on the registration's
  branch, and slot 2's stale registration survives.
- Measured: `git worktree repair <host>` rewrites a broken `.git` file but still exits 1, reporting
  what it found (git 2.54). `RepairHost` is therefore an attempt whose result the next observation
  judges; only a cancelled context fails it.
- `compileSetup` runs weave without the lease, then retakes it through a caller-supplied `relock`,
  checks that the admin directory is unchanged, and writes the marker with the resting branch's
  current commit as its baseline.

### 2026-10-05 (q) — Task 2.2 implementation reconciliation; SetAside pulled into M2

- The loop (`reconcileLoop`) runs over a `slotWorld` interface (lock, observe, apply). Production is
  `osSlotWorld` (git and the filesystem under the host creation lease, released for weave
  compile); tests use `SlotWorld`.
- No-progress is detected precisely, not by comparing observations. `PlanSlot` reports a step the
  observation still calls for after this run executed it as `SlotPlan.Retried`, and the loop fails
  with no progress at it. An observation-equality guard would have misfired on `RepairHost`, whose
  effect is judged by the next pass. The `SameShape` idea from Revision (o) is therefore not used.
  `SlotPlan.Empty` counts `Retried`, so a lying converge is never read as converged.
- Domain evidence (Done-when 2):
  - `TestReconcileConvergesFromEveryPerturbation` runs every single and pair perturbation under
    all 7 agent states, twice each (>5000 runs).
  - `TestReconcileAgreesWithRealGit` gives the same terminal class on real git for all 12
    producible scenarios.
  - `TestReconcileCrashAfterEveryStep` models a crash as process death (a panic), not a returned
    error.
  - Mutations caught: the agent gate, ignoring `Retried`, planning from the first observation only.
- The fixture's weave is now stateful (`DepSources`): it clones a known missing dependency as a
  layer, and fails with weave's missing-substrate line on an unknown one.
- Task 3.1 (`SetAside`) moves into M2. The planner already plans it, and M2 wires reconcile into
  open and reboot, so a slot with an unreadable clone would otherwise fail in production. It
  follows Task 3.1's order. `registerStore` and `agentNow` are hooks Couch supplies; `rename` is
  the crash seam. `checkoutEvidence` is the one positive-evidence reading, shared by the observer
  and the pre-rename re-check (ARCH-DRY). M3 keeps the GC (3.3), the report (3.4) and the
  acceptance work (3.5).
