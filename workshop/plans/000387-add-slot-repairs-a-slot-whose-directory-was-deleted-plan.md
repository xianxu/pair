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

## Decisions this plan rests on (operator, 2026-10-05)

1. **Desired state = a working slot.** Derived state is repaired automatically, with no
   confirmation. Only the operator's data is preserved: uncommitted and untracked files,
   local-only commits, and the slot store (archive, preferences). There is no operator "rebuild"
   verb.
2. **Adopt a leftover `main-slotN`.** It is the resting branch by convention. Refuse only if it
   is checked out in a different worktree or its upstream config conflicts.
3. **Saved work lives in the slot store**, `<env>/.couch/saved-work/<id>/`, and is collected after
   storagegc's 60-day `RetentionPeriod`.
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
  removal would touch is saved first (rule 2), and only a broken dependency is ever removed.
  Save-before-remove is the only remaining guard. Test: a foreign, pre-populated env (a non-git
  `ariadne/` with files) is adopted, and the dependency is saved before removal. The intent
  becomes a legacy resource whose desired state is absent. Mixed binaries: an older Couch that
  crashed mid-creation and runs again after the new binary removed its intent will refuse;
  acceptable, since `make build` replaces the binary.
- **R2 — One save mechanism for every removal: a whole-tree tar.** The planning-session answer
  proposed a stash-shaped commit on `refs/couch/saved/slotN/<id>` for the host. Planning showed
  the host checkout is never removed: an absent host is re-added, and a mismatched one gets
  `git worktree repair`, otherwise a hand-off. Only a dependency clone is ever removed, and only
  when it is broken on **positive evidence**: no `.git`, or git answers "not a git repository".
  Any other git error is `unknown` (lessons: a failed probe is not evidence). A broken clone
  cannot be read through git, so the save is a whole-tree tar of the directory, `.git`
  included. That keeps local-only commits wherever git data survives, and needs no git. There
  is one mechanism and no new ref namespace. The issue's Done-when bullet is revised now
  (ARCH-DRY).
- **R5 — A failing recompile under a valid marker is remembered, not repeated every open.**
  Rule 4 keeps a working slot usable when its recompile fails. To keep each later open from
  re-running a failing `weave compile` (up to `SetupTimeout`), the reconciler writes
  `couch-setup-attempt.json` beside the marker. It records an inputs digest (host `HEAD` + the
  bytes of every read `construct/deps`) and the classified failure. Observation reads it: same
  digest → `setup` is `present-with-warning`, and the plan does not compile. A changed input, or
  an explicit `couch --reconcile`, compiles again. It is derived state, lives and dies with the
  registration's admin directory, and is overwritten in place (one file).
- **R3 — An invalid setup marker is derived state, not a conflict.** #367 treated an invalid
  `couch-setup-success.json` as "setup conflict, never advise reboot". Here it is a broken
  derived file: remove it and re-run setup. The rule "repair never makes a working slot worse"
  still holds for a *valid* marker whose recompile fails (Task 2.3).
- **R4 — Removal is a rename, then a delete.** A dependency clone is removed by an atomic
  `rename` into `<env>/.reconcile-trash/<id>`, and only then by `RemoveAll`. A crash mid-delete
  therefore never leaves a half-deleted clone that would read as "broken" and be saved again.
  The trash is swept at the start of every reconcile.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SlotLayout` (every slot-state location: env, host, store, saved-work, trash, setup lock, intent, resting branch, marker name) | `cmd/internal/couchcore/slotlayout.go` | new |
| `SlotResourceID` / `ResourceKind` / `SlotResourceSpec` / `SlotResources()` (the table and its edges) | `cmd/internal/couchcore/slotresource.go` | new |
| `ObservedState` (`present`/`absent`/`broken`/`unknown`) / `ResourceObservation` / `SlotObservation` | `cmd/internal/couchcore/slotresource.go` | new |
| `ConvergeStep` / `PlannedStep` / `SlotPlan` / `PlanSlot(SlotObservation) SlotPlan` | `cmd/internal/couchcore/slotplan.go` | new |
| `ReconcileFailure{Resource, Class: retryable\|handoff\|unknown, Cause}` / `ClassifyConvergeError` / `ReconcileAdvice(address, failure)` | `cmd/internal/couchcore/slotfailure.go` | new |
| `ParseSubstrateRows(content) ([]string, error)` | `cmd/internal/couchcore/slotdeps.go` | new |
| `SelectStartSlot` (a reconcilable candidate is reusable; an unknown one skips only its number) | `cmd/internal/couchcore/slotallocation.go` | modified |
| `NextHostAction` / `HostObservation` | `cmd/internal/couchcore/provision_host.go` | deleted (absorbed into `PlanSlot`) |
| `CreationIntent` / `validateCreationIntent` | `cmd/internal/couchcore/provision.go`, `provision_request.go` | deleted (R1; the legacy file is a resource with desired state absent) |

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ObserveSlot(ctx, io, layout) SlotObservation` | `cmd/internal/couchcore/slotobserve.go` | new | git (`worktree list`, `for-each-ref`, `config`, `rev-parse`, `status`), `sdlc workspace --json`, lstat, non-creating flock probe, `construct/deps` reads |
| `Converge(ctx, io, layout, step)` | `cmd/internal/couchcore/slotconverge.go` | new | git (`worktree add/repair/prune`, `fetch`, `update-ref`, `config --add`), mkdir, rename/RemoveAll, `weave compile`, marker write |
| `SaveWork(ctx, io, checkout, dest)` | `cmd/internal/couchcore/slotsave.go` | new | `git bundle`, `git diff --binary`, tar of untracked files |
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
| `trash` | reconcile trash | `<env>/.reconcile-trash/` | derived, transient | `env` | absent | present → `RemoveAll` (first step of every run) |
| `intent` | legacy creation intent | `<common>/couch-workspaces/N/creation.json` | derived, legacy (R1) | — | absent | present → remove, under the lease |
| `branch` | resting branch `main-slotN` | `refs/heads/main-slotN` in the shared repo | user data (adopted; never deleted) | — | present | absent → fetch remote `main`, `update-ref` create at the fetched OID; checked out in a worktree other than the host path → hand-off |
| `upstream` | `branch.main-slotN.{remote,merge}` | shared repo config | derived | `branch` | `<remote>` / `refs/heads/main` | absent → `config --add`; conflicting → hand-off |
| `registration` | git worktree registration | `<common>/worktrees/<name>/` for the host path | derived | `branch`, `env` | present, directory present | stale (directory gone, git reports `prunable` for **this** path) → `git worktree remove <hostPath>` (targeted; never `prune`, which drops every prunable registration in the shared repository); locked → hand-off; absent → handled by `host` |
| `host` | host checkout | `<env>/<repo>` | derived container; its dirty files are user data | `registration`, `env`, `upstream` | present and verified (`verifyHost`'s identity check) | absent → `git worktree add <host> main-slotN`; mismatched → `git worktree repair <host>` once, then hand-off; never removed |
| `deps` | dependency declaration | `construct/deps` of the host, then of each present substrate (transitive) | external (another repo's file) | `host` | parseable | unparseable/unreadable → unknown (stop at `deps` and `setup`) |
| `dep:<rel>` | one dependency clone | `<env>/<name>` per declared substrate | derived from a remote; dirty files and local-only commits are user data | `deps`, `env` | present, git-readable | absent → covered by `setup`; broken **on positive evidence only** (no `.git`, or `rev-parse` says "not a git repository"; any other git error is `unknown`) → tree-tar save → rename to trash → covered by `setup` |
| `setup` | weave setup, Couch's marker, the attempt memo (R5) | `<admin>/couch-setup-success.json`, `<admin>/couch-setup-attempt.json`; lock `<env>/.weave-setup.lock` | derived | `host`, every `dep:*` | valid marker and no `dep:*` absent | lock held → retryable stop; a memo whose digest matches → `present-with-warning`, no compile; otherwise `weave compile` (no lease held), then re-observe under the lease and write the marker (success) or the memo (failure under a valid marker); an invalid marker is removed first (R3) |
| `agent` | agent session + thread record | zellij session, `<env>/.couch/thread.json` | runtime | `host` | (observed only) | never converged by reconcile; a live, busy or unknown agent forbids any `dep:*` removal (hold `agent-live`), re-checked under `.weave-setup.lock` immediately before the rename; a record-less live agent is adopted by `OpenSlot`, which runs after reconcile |

**Outside the table (external, never touched; listed so the coverage audit can name them):**
the agent's transcript, Pair per-thread artifacts (`artifactpath`), sdlc claims, the global
manifest's `SlotRepositories` enrollment, and every other branch in the shared repository.

**Edges, in topological order:**
`env → {store, trash}`, `{intent}`, `branch → upstream`, `{branch, env} → registration →
host`, `upstream → host`, `host → deps → dep:* → setup`, `host → agent`. Converge runs in this
order. Removal (only `trash`, `intent`, a broken `dep:*`) runs before the converges of its
dependents.

### `PlanSlot` rules (pure; `SlotObservation → SlotPlan`)

1. **Unknown blocks.** An `unknown` resource contributes a `Stop{resource, reason}`. Every
   resource that transitively depends on it is skipped. Resources that do not depend on it
   still converge. Example: an unknown `agent` does not stop `branch`.
2. **User data is saved before removal.** A `RemoveDep` step is always immediately preceded by a
   `SaveWork` step for the same `dep:*`. The save is a whole-tree tar (R2). The size is measured
   only when a removal is planned, never on the healthy path. If the tree is over the cap
   (64 MB) or cannot be read, a `Stop` replaces both steps.
3. **No removal under a live agent.** If `agent` is `live`, `busy` or `unknown`, a broken `dep:*`
   gets `Stop{agent-live}`. Every non-removing step still runs. `RemoveDep` re-observes the agent
   sessions while holding `.weave-setup.lock`, immediately before the rename, and aborts on a
   change. A plain `pair` launch outside the operation queue can start one between observe and
   apply (lessons: revalidate authority before mutation).
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
sweep trash
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

### ARCH notes

- **ARCH-DRY.** One provisioner: `Ensure`'s body becomes `Reconcile`. Its callers are
  `slotlaunch.go:71`, `slotrecovery.go:240` (`selectedSlot`), `slotstart.go`
  (`spawnManagedResolution`) and `operationdispatch.go:154` (`provision-workspace`); Task 2.5
  changes *when* `selectedSlot` calls it. `NextHostAction` and the intent proofs are deleted
  rather than kept beside the new planner. One path authority (`SlotLayout`) replaces the slot
  path math, and the slot-path parsers become one `ParseSlotPath`. Task 1.1 enumerates the
  sites, and Task 1.2's token audit proves the list complete. One advice function. The deps
  parser is copied, with a conformance table (pair has no Go dependency on ariadne today). The
  alternative, importing ariadne's exported `layergraph.ParseRows`/`Walk`/`ReadDeclaration`, adds
  a module dependency pair→ariadne. That is a layer decision for the operator, raised at
  approval.
- **ARCH-PURE.** `PlanSlot`, `ClassifyConvergeError`, `ReconcileAdvice`, `ParseSubstrateRows` and
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
  - **Saves:** capped at 64 MB per entry and 16 entries per slot. Over the cap → `Stop` with the
    10 largest paths.
  - **Loop:** at most `len(SlotResources())+2` passes.
  - Other categories N/A.
- **ARCH-ORDER.** The reconciler holds no state between runs. Each run rebuilds its view from
  the world (level-triggered), so a crash at any point is handled by running it again.
  - **Inside a run:** git and filesystem resources are observed and converged under the host
    creation lease (nonblocking; busy → retryable). The lease is released for `weave compile`,
    and setup's marker is written only after a re-observation under the lease.
  - **Second actors:** another Couch operation on the same slot is serialized by the operation
    queue (same process) or by the lease (other processes). A concurrent `weave` holds
    `.weave-setup.lock` → retryable stop. A dependency removal probes that lock without
    `O_CREAT` and holds it across save + rename (`flock` on the existing file). If the file is
    absent, there is no setup to race.
  - **Stale observations:** each converge step re-checks its own precondition at execution time
    (e.g. `update-ref` with the zero-OID CAS, `worktree add` failing if the path appeared).
    The only destructive step, `RemoveDep`, re-observes agent sessions and the dependency's
    broken evidence under `.weave-setup.lock` immediately before its rename (rule 3). A test
    injects an agent appearing between observe and apply.
  - **Most likely to be mishandled:** a crash between `SaveWork` and the rename. On re-run the
    dependency is still broken, so it is saved again. Entries are content-addressed
    (`<id>` = digest of the saved files), so the second save finds the same id and writes
    nothing.
  - **Tests:** `SlotWorld` crash injection after every step reproduces each ordering
    deterministically.
- **ARCH-FUNERAL.**
  - **saved-work entries:** created by `SaveWork`. The last reader is the operator restoring
    work. Removed by the archive GC pass at 60 days (`storagegc.RetentionPeriod`, measured from
    the entry's `saved_at`). Bounded at 64 MB and 16 entries; at the cap, reconcile stops rather
    than evicting. The GC pass only visits registered stores (`CouchReferences.Snapshot`,
    `gcruntime/references.go:14`), so `SaveWork` registers the slot store through the same path
    `withRetentionWrite` uses (`archive_gc.go`) before writing. A store recreated lazily after
    an env reset is therefore still collected.
  - **The attempt memo (R5):** one file per registration, overwritten in place, removed with
    the admin directory.
  - **`.reconcile-trash/`:** removed within the same run, and swept at the start of the next.
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
  - **Tar members:** relative names only.
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
`recoverplan.go:883` (the `-slot` cut), `threadstore_layout.go:19` (`.couch`), and the
`slotForContainedPath`, `conventionalSlotFromPath` and `conventionalSlotOfMember` parsers.

- [ ] **Step 1: Failing tests.**
  - `TestSlotLayoutPaths`: for primary `/f/pair`, N=3, every method returns the documented path
    (env, host, store, saved-work root, trash, setup lock, intent, resting branch
    `main-slot3`, marker name).
  - `TestParseSlotPathRoundTrip`: for every layout path of N ∈ {1, 12} and for paths inside the
    host, `ParseSlotPath` returns (primary, N). Paths outside → false. N=0 → false.
    `primary-slotX` → false.
- [ ] **Step 2:** run `go test ./cmd/internal/couchcore -run 'SlotLayout|ParseSlotPath'`. Expect
  FAIL (undefined).
- [ ] **Step 3:** implement. Replace `slotDirectoryNumber`, `slotForContainedPath`,
  `conventionalSlotFromPath` and `conventionalSlotOfMember`'s parsing with `ParseSlotPath`.
- [ ] **Step 4:** `go test ./cmd/internal/couchcore` passes. Mutation: make `ParseSlotPath`
  accept N=0, and the round-trip test must fail.
- [ ] **Step 5:** commit `#387 M1: couchcore: SlotLayout is the one slot path authority`.

### Task 1.2: The resource table and its coverage audit

**Files:** create `slotresource.go`, `slotresource_test.go`, `slotresource_coverage_test.go`.

- [ ] **Step 1: Failing tests.**
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
      `couch-workspaces`, `creation.json`, `saved-work`, `.reconcile-trash`. Any other file
      fails the test, naming the file:line. The token list is the one hand-written piece: a
      token names a *kind* of state, so adding state without a token is caught by (b).
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
- [ ] **Step 2:** FAIL. The audit lists today's sites, i.e. Task 1.1's enumeration plus anything
  it missed. Record any miss in the Log.
- [ ] **Step 3:** implement the table exactly as in Core concepts, and move each listed site
  onto `SlotLayout` (for `threadstore_layout.go`, `newSlotThreadStore` takes `layout.Store()`).
- [ ] **Step 4:** PASS. Mutation (a): add `"x-slot2"` in `reboot.go`, and the audit must fail.
  Mutation (b): add `filepath.Join(slot.EnvironmentRoot, "x")` in `reboot.go`, and the audit
  must fail. Mutation (c): add an unused `SlotLayout.Foo()`, and the audit must fail. Revert
  each from a `cp` byte copy (lessons: never `git checkout <file>`).
- [ ] **Step 5:** commit `#387 M1: couchcore: the slot resource table, enforced by an AST audit`.

### Task 1.3: `ParseSubstrateRows` and the deps conformance table

**Files:** create `slotdeps.go`, `slotdeps_test.go`, and
`testdata/layergraph_deps_rows.txt` (copied from ariadne `pkg/layergraph/deps_test.go`'s
row table, with the ariadne revision in its header).

- [ ] **Step 1: Failing tests.**
  - `TestParseSubstrateRowsConformance`: for each copied row, substrate rows yield their path,
    `data` rows are ignored, and malformed rows are errors.
  - Paths are resolved like weave: relative to the declaring checkout's parent, canonicalized
    through `pwd -P` of the parent. An unresolvable parent is skipped (ariadne `walk.go`).
  - Inputs over 64 KB are errors. A row resolving outside the env makes the whole declaration
    unknown.
- [ ] **Step 2–4:** red → green. Mutation: accept `data` rows as substrates, and the
  conformance test must fail.
- [ ] **Step 5:** commit `#387 M1: couchcore: construct/deps substrate parser`. File the
  upstream ask in ariadne (`sdlc issue new` there, deps on pair#387): expose declared
  dependencies in `sdlc workspace --json` so this parser can be deleted. Record the id in the
  Log.

### Task 1.4: `ObserveSlot` (the shell) and its states

**Files:** create `slotobserve.go`, `slotobserve_test.go`. Modify `provision_git_test.go`
(`ProvisionFixture` gains helpers that break each resource) and `testdata/fakeweave` (stateful,
see ARCH-MOCK).

- [ ] **Step 1: Failing tests** on `ProvisionFixture` (real git). Each starts from a slot
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
    present. No user data is measured on observe (rule 2); the measure is taken by `SaveWork`.
- [ ] **Step 2:** FAIL. **Step 3:** implement. Observations use `SlotLayout` and the
  `ProvisionIO` seam only. The flock probe opens without `O_CREAT`, and a missing lock file
  means free.
- [ ] **Step 4:** PASS. Mutation: map a `worktree list` failure to `absent`, and the unknown
  case must fail.
- [ ] **Step 5:** commit `#387 M1: couchcore: observe every slot resource`.

### Task 1.5: `PlanSlot` (pure) and the derived domain

**Files:** create `slotplan.go`, `slotplan_test.go`.

- [ ] **Step 1: Failing tests.**
  - `TestPlanSlotDomain`: the domain is derived. Every resource from `SlotResources()` takes
    every `ObservedState` from `AllObservedStates()`, singly and in pairs (other resources
    present), with agent ∈ `AllAgentStates()` = {none, live, busy, unknown}. For each point the
    invariants hold:
    - I1: no step targets a resource that is converged;
    - I2: no step targets an `unknown` resource or anything depending on one;
    - I3: every `RemoveDep` is immediately preceded by `SaveWork` for the same dependency;
    - I4: no `RemoveDep` when agent is live, busy or unknown;
    - I5: `store` and `branch` never get a removing step;
    - I6: the step order respects `TopoOrder()`.
  - `TestPlanSlotNamedCases`: `tools:1`'s shape (host present, marker absent, `dep:ariadne`
    absent) → `[Compile]`. The deleted-slot shape (env absent, registration stale, branch
    present) → `[Mkdir env, Prune, WorktreeAdd, Compile]` across passes; asserted on the first
    pass's ready set. A valid marker with a missing dependency → `[Compile]` with
    `KeepOnFailure`.
  - `:0` → refusal.
- [ ] **Step 2–4:** red → green. Mutations: drop I3's save step, or let I4 pass under a live
  agent. Each must fail the domain test (verify the mutation applied before trusting the
  result).
- [ ] **Step 5:** commit `#387 M1: couchcore: PlanSlot over the derived domain`.

### Task 1.6: `couch --show` prints the slot's resources and plan

**Files:** `operationdispatch.go` (`show`: when the ref resolves to a slot address or a slot
path, also observe the slot and attach `SlotReport{Observation, Plan}`), `couchcmd/run.go`
(`render`), `run_test.go`. `--show repo:N` must work even when no thread record exists, as for a
deleted env.

- [ ] **Step 1: Failing tests.**
  - `--show tools:1`-shaped fixture prints one line per resource in topological order
    (`env present`, …, `dep:ariadne absent`, `setup absent (no marker)`), then
    `plan: compile (weave compile in <host>)`.
  - Deleted-slot fixture prints `env absent`, `registration stale`, and the plan.
  - A converged slot prints `plan: nothing to do`.
  - An unknown resource prints `<id> unknown: <reason>` and `plan: stops at <id>`.
- [ ] **Step 2–4:** red → green. Mutation: render from a hand-written resource list instead of
  `TopoOrder()`, and a test adding a resource must fail.
- [ ] **Step 5:** commit `#387 M1: couch --show reports slot resources and the plan`.

### Task 1.7: Docs and close M1

- [ ] `atlas/couch.md`: a "Slot resources" section with the table (linking `slotresource.go` as
  the source), observation states and the plan. Keep `atlas/index.md` valid.
- [ ] Full verification (Chunk 4). Then `sdlc milestone-close --issue 387 --milestone M1`.

---

## Chunk 2: M2 — converge: `Reconcile` replaces `Ensure`'s repair logic

### Task 2.1: `Converge` steps (the shell)

**Files:** create `slotconverge.go`, `slotconverge_test.go`.

- [ ] **Step 1: Failing tests** on `ProvisionFixture`, one per step, each run twice (second run a
  no-op or refused by its own precondition, never a duplicate effect):
  - `MkdirEnv`;
  - `SweepTrash`;
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
- [ ] **Step 2–4:** red → green.
- [ ] **Step 5:** commit `#387 M2: couchcore: idempotent converge steps`.

### Task 2.2: The loop, `SlotWorld`, and the twice-run property

**Files:** create `slotreconcile.go`, `slotreconcile_test.go`, `slotworld_fake_test.go`.

- [ ] **Step 1: Failing tests.**
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
    effects; content-addressed saves produce one entry.
  - `TestReconcileNoProgressStops`: a world whose `WorktreeAdd` reports success but changes
    nothing → failure `no progress at registration`, within the pass bound.
- [ ] **Step 2–4:** red → green. Mutations: remove the second-pass re-observation (use the first
  plan for every pass) → the crash test fails; make save ids random → the duplicate-entry
  assertion fails.
- [ ] **Step 5:** commit `#387 M2: couchcore: Reconcile, level-triggered and bounded`.

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
  - `TestSelectStartSlotReconcilable`: a reconcilable candidate is picked as the lowest free
    number (`Exists: true, Reconcile: true`). Any other `Err` skips its number only and the
    allocation carries a `Notice`. With the domain of candidate errors derived from the typed
    set, no candidate error refuses the whole repository.
  - Add slot on the deleted-slot fixture picks N, reconciles it through `selectedSlot`, and
    starts. On today's code it refuses (red).
  - Open on a fixture whose host is **verified** but whose marker is missing → reconcile
    compiles before the agent starts (red today: `selectedSlot` skips `Ensure`).
  - Open/resume on the `tools:1`-shaped fixture (the weave fake knows no source for `ariadne`)
    returns `ReconcileAdvice` handoff text, and no agent is started.
  - `rebootSlot` on a slot whose env is gone: reconcile recreates it, then a fresh start.
  - `DecideReboot`'s totality test: slot rows no longer produce `RebootDirectoryMissing`, and
    `:0` rows are unchanged.
  - `ActorActions`: a `:1+` `DirectoryMissing` row offers `reboot`.
- [ ] **Step 2–4:** red → green. Mutations: restore the repository-wide refusal in
  `SelectStartSlot` → the add-slot test fails; restore the `Verified` gate in `selectedSlot` →
  the verified-host open test fails.
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

## Chunk 3: M3 — save-then-remove, saved-work lifecycle, recovery report

### Task 3.1: `SaveWork` and the saved-work entry

**Files:** create `slotsave.go`, `slotsave_test.go`.

Entry layout: `<env>/.couch/saved-work/<id>/{manifest.json, tree.tar}` (one entry per removed
dependency; R2). `id` is a sha256 over a canonical listing of the tree: for each entry, sorted by
relative path, `(path, type, mode&0777, sha256(content) | symlink target)`. The tar is written
from the same sorted walk with canonical headers (uid/gid 0, empty names, fixed format), so
saving the same tree twice gives the same id and the same bytes. `manifest.json` holds
`saved_at`, the slot address, the dependency path, the size, and the restore command
(`mkdir -p <path> && tar -xf <entry>/tree.tar -C <path>`). It is written last via
`durablefile`, so an entry without a manifest is incomplete and is redone. Before writing,
`SaveWork` registers the slot store for GC (ARCH-FUNERAL).

- [ ] **Step 1: Failing tests** (real git):
  - a dependency with a local-only commit, a dirty tracked file and an untracked file, made
    broken by removing `.git/HEAD` → the tar restores into a temp directory. After restoring
    `.git/HEAD`, `git log` there shows the local-only commit, `git status` shows the dirty file,
    and the untracked file is present;
  - a non-git directory → the tar restores byte-identical;
  - over 64 MB → error listing the 10 largest paths, nothing written;
  - 16 existing entries → error `saved-work limit reached`;
  - the same tree saved twice (with mtimes touched between) → the same id, one entry,
    identical tar bytes;
  - symlinks are archived as links, never followed. Member names are relative, and an entry
    whose resolved path leaves the dependency directory is refused.
- [ ] **Step 2–4:** red → green.
- [ ] **Step 5:** commit `#387 M3: couchcore: SaveWork writes a content-addressed saved-work entry`.

### Task 3.2: `RemoveDep` under the setup lock, then setup re-clones

**Files:** `slotconverge.go`, `slotplan.go` (the save → remove pair already planned in M1;
now executed).

- [ ] **Step 1: Failing tests:**
  - a broken dirty dependency → saved, renamed to trash, trash swept, re-cloned by compile;
    the result lists the entry and its restore commands;
  - the same under a live agent → `Stop{agent-live}`, nothing saved or moved;
  - the setup lock held by the test during the run → retryable, nothing moved;
  - a symlinked dependency path → handoff, nothing moved.
- [ ] **Step 2–4:** red → green. Mutation: swap save and rename, and the dirty-dep test (which
  asserts the entry exists before the rename in the effect log) must fail.
- [ ] **Step 5:** commit `#387 M3: couchcore: a broken dependency is saved, removed and re-cloned`.

### Task 3.3: Saved-work collection

**Files:** `archive_gc.go`, the gcruntime pass that drives archive detach
(`gcruntime/references.go`), tests.

- [ ] **Step 1: Failing tests:** an entry with `saved_at` older than `storagegc.RetentionPeriod`
  is removed by the GC pass. A younger one stays. An entry without a manifest older than the
  period is removed as abandoned. The pass tolerates a slot with no `saved-work/`. Removal is
  under the store lock, through the same write path as archive detach.
  - A slot whose store was recreated lazily after an env reset, and whose only content is a
    saved-work entry, is visited by `CouchReferences.Snapshot` (because `SaveWork` registered
    it) and the old entry is collected. On today's registration path this is red.
- [ ] **Step 2–4:** red → green. **Step 5:** commit `#387 M3: saved-work entries expire with archive retention`.

### Task 3.4: The recovery report reads the reconciler's observations

**Files:** `recoverplan.go`, `recoverplan_source.go`, `recoverplan_test.go`.

- [ ] **Step 1: Failing tests.**
  - `RecoverPlanInput` carries one `SlotPlan` per present `:1+` candidate (gathered in the shell
    via `ObserveSlot`).
  - A slot whose plan is non-empty and has no `Stop` → class `reconcilable`, step `[reconcile]`
    (emitting `couch --reconcile repo:N`).
  - A plan with a handoff (from a recorded last failure, or a `Stop`) → class `slot-needs-:0`
    with `ReconcileAdvice`.
  - `directory-missing` for `:1+` maps to `reconcilable`.
  - The totality test's domain gains `SlotPlan` shapes, derived from Task 1.5's domain
    generator; class/hold/note coverage is re-derived.
  - `TestRowAdviceNamesOnlyReachableActions`'s sweep includes `reconcile`.
- [ ] **Step 2–4:** red → green. Mutation: classify a plan with a `Stop` as `reconcilable`, and
  the totality test must fail.
- [ ] **Step 5:** commit `#387 M3: recovery report reads slot plans`.

### Task 3.5: Acceptance (part 2) — the dirty slot

**Files:** `slotreconcile_acceptance_test.go`.

- [ ] A dirty slot whose broken dependency must be recreated. The host has a dirty file (it must
  be untouched: the host is never removed). The dependency has a local-only commit, a dirty file
  and an untracked file, and is made broken by removing `.git/HEAD` (git then answers "not a
  git repository": positive evidence). Reconcile saves the dependency first (entry verified as
  in Task 3.1, including the local-only commit after restore), recreates it via compile, and
  leaves the host's dirty file byte-identical. A second run is a no-op with no new entry.
- [ ] The same dependency with a held `index.lock` instead → `unknown`, nothing saved or moved,
  and the output names `dep:<rel>` as unobservable.
- [ ] A foreign, pre-populated env (R1): a non-git `ariadne/` with files at the env path before
  slot creation → adopted. The dependency is saved, then replaced by the clone, and the entry
  restores the foreign files.
- [ ] An agent session appearing between observe and `RemoveDep` (injected through the session
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
