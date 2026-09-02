# Resume Unique Parked Root Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Make interactive `couch [<repo>]` startup resume the sole exact actionable parked thread for that repository path, while preserving new-root startup for zero or ambiguous matches.

**Architecture:** Extend the existing actionable inventory to expose its already-verified parked rows with physical working paths, then pass those rows and the resolved startup target to one pure unique-candidate decision. A thin Couch composition-root method resolves the target, reads the authoritative inventory, and invokes exactly one existing effect path: `ResumeContext` for one candidate or the ordinary resolved start path otherwise; the CLI only selects this method for interactive launch.

**Tech Stack:** Go, `cmd/internal/couchcore`, `cmd/internal/couchcmd`, existing fake Path/Git/ThreadStore/native-binding/Runner seams, Go `testing`.

---

## Chunk 1: Startup decision and integration

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `SelectUniqueParkedRoot` | `cmd/internal/couchcore/startup.go` | new |

- **`SelectUniqueParkedRoot`** — returns an address only when exactly one actionable parked row has the requested repository scope and physical working path.
  - **Relationships:** N:1 actionable rows to one startup target; the returned address identifies at most one existing thread.
  - **DRY rationale:** consumes `ActionableThreadSummary`, so verified park and exact native-binding eligibility remain single-sourced in `ProjectActionableThreads` / `ActionableThreadInventoryContext` (`ARCH-DRY`, `ARCH-PURPOSE`).
  - **Future extensions:** intentionally none for ranking or prompting; a later UX must be a separately designed policy rather than widening this exact-uniqueness rule.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Couch.StartInteractive` | `cmd/internal/couchcore/startup.go` | new | existing path/git/policy, ThreadStore inventory, native-binding resolver, Resume, and start seams |
| `Couch.ActionableThreadInventoryContext` | `cmd/internal/couchcore/actionableinventory.go` | modified | ThreadStore snapshot, path physicalization, and native-binding inventory |
| interactive launch dispatch | `cmd/internal/couchcmd/run.go` | modified | terminal CLI startup and Console root installation |

- **`Couch.StartInteractive(ctx context.Context, args StartArgs) (StartResult, error)`** — resolves the requested target once, derives its `ThreadAddress.RepoScope` with `launcher.ResolveRepoScope(string(resolution.Worktree))`, obtains authoritative actionable inventory, and passes that scope plus `resolution.CanonicalPath` to the pure selector before calling either `ResumeContext` or the existing resolved new-start path.
  - **Injected into:** it is the thin IO shell around `SelectUniqueParkedRoot`; all dependencies already belong to the Couch composition root and tests use the existing stateful fakes (`ARCH-PURE`, `ARCH-MOCK`).
  - **Future extensions:** none within this issue; it deliberately returns the same `StartResult` consumed by Console attachment.
- **`Couch.ActionableThreadInventoryContext`** — preserves its conservative projection but writes the successful physical path into the snapshot copy supplied to the projector, so every downstream identity comparison uses the value whose existence was proved.
  - **Injected into:** `Couch.StartInteractive` and the existing Console actionable provider.
  - **Future extensions:** other consumers can rely on actionable `WorkingPath` being physical without repeating filesystem resolution.
- **interactive launch dispatch** — uses `StartInteractive` only for public terminal launch; typed `prepare-start`, `start`, and explicit `resume` operations retain their current contracts.
  - **Injected into:** existing `runConsole`, which installs either the resumed or newly-created handle as root/home through the same initial `attach` operation.
  - **Future extensions:** N/A.

### Architecture and operating envelope

- `ARCH-DRY`: eligibility stays entirely in actionable inventory and final authority stays entirely in `ResumeContext`; startup adds no second lifecycle or binding validator.
- `ARCH-PURE`: exact matching and cardinality are one deterministic function; path, ThreadStore, binding, launch, and Console effects stay in the Couch/CLI shell.
- `ARCH-PURPOSE`: the tests enumerate the complete startup class: sole match, zero, ambiguity, alias identity, excluded lifecycle/repository/path records, authoritative inventory failure, and post-selection Resume refusal.
- `ARCH-MOCK`: no external dependency is added. Integration tests reuse the production seams and stateful `FakeThreadArtifactCollisionChecker`, `FakeRunner`, fake Git, fake PathOps, and temp-backed ThreadStore.
- `ARCH-CONSTRAINTS`: this is an interactive startup path. It performs one existing target resolution and one existing local actionable snapshot/binding pass, with O(n) pure selection over the snapshot; no fleet scan, retry, prompt, goroutine fan-out, or additional asymptotic storage. Cancellation and existing binding-resolution bounds remain authoritative. CPU/network are N/A beyond existing local subprocess/filesystem seams; malformed per-record paths remain bounded omissions, while a whole snapshot failure terminates startup before child effects.

### Task 1: Pin the pure exact-uniqueness rule

**Files:**
- Create: `cmd/internal/couchcore/startup.go`
- Create: `cmd/internal/couchcore/startup_test.go`

- [x] **Step 1: Write failing table tests for `SelectUniqueParkedRoot`.** Strategy: vary arbitrary actionable row state/scope/path/cardinality inputs; the mechanical guard is returning an address only for cardinality exactly one after exact normalized identity filtering.
- [x] **Step 2: Run `go test ./cmd/internal/couchcore -run '^TestSelectUniqueParkedRoot' -count=1` and confirm it fails because the selector is absent.**
- [x] **Step 3: Implement the minimal pure selector.** Compare state, exact scope, and normalized/physical path strings; stop returning a candidate once a second exact match is observed.
- [x] **Step 4: Run the focused test and confirm it passes.**

### Task 2: Make actionable paths carry proved physical identity

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go`
- Modify: `cmd/internal/couchcore/actionableinventory_test.go`

- [x] **Step 1: Add failing tests for `ActionableThreadInventoryContext`.** Strategy: inject alias, per-record path failure, and whole-snapshot failure classes through existing stateful seams; the mechanical guards are physical-path projection, conservative row omission, and error propagation respectively.
- [x] **Step 2: Run `go test ./cmd/internal/couchcore -run 'TestActionableThreadInventory.*(Physical|Unavailable|Failure)' -count=1` and confirm the alias assertion fails.**
- [x] **Step 3: Physicalize eligible parked record paths in the context-bound inventory shell and project a copied record with that physical path.** Do not move path IO into `ProjectActionableThreads` or duplicate Resume eligibility checks.
- [x] **Step 4: Run `go test ./cmd/internal/couchcore -run 'Test(ActionableThreadInventory|ProjectActionableThreads)' -count=1` and confirm it passes.**

### Task 3: Choose Resume or new root before effects

**Files:**
- Modify: `cmd/internal/couchcore/startup.go`
- Modify: `cmd/internal/couchcore/startup_test.go`
- Reuse without changing authority: `cmd/internal/couchcore/resume.go`
- Reuse resolved start mechanics: `cmd/internal/couchcore/couch.go`

- [x] **Step 1: Add failing stateful integration tests for `Couch.StartInteractive`.** Strategy: vary actionable inventory cardinality/identity/authority and invalidate authority at the resume boundary through temp-backed ThreadStore plus production-seam fakes; the mechanical guard is exactly one terminal effect—resume for one exact candidate, new start otherwise, and no start after authoritative error/refusal.
- [x] **Step 2: Run `go test ./cmd/internal/couchcore -run '^TestStartInteractive' -count=1` and confirm it fails because the orchestration method is absent.**
- [x] **Step 3: Implement `StartInteractive(ctx context.Context, args StartArgs) (StartResult, error)`.** Reuse the same internal start resolution for the canonical physical path, and derive the target address scope with the same `launcher.ResolveRepoScope(string(resolution.Worktree))` call used by `spawnResolved`/thread allocation (do not substitute policy `RepoIdentity`). Query `ActionableThreadInventoryContext`, call `SelectUniqueParkedRoot(rows, scope.Key, resolution.CanonicalPath)`, invoke `ResumeContext` for one address, and otherwise invoke the existing resolved start path. Return a `StartResult` in both cases and never issue an unused start grant. Tests must use distinct policy identity and address-scope values so conflation fails.
- [x] **Step 4: Run the focused tests and then `go test ./cmd/internal/couchcore -count=1`; confirm both pass.**

### Task 4: Route public interactive startup through the decision

**Files:**
- Modify: `cmd/internal/couchcmd/run.go`
- Modify: `cmd/internal/couchcmd/run_test.go`

- [x] **Step 1: Add failing tests for the public interactive launch dispatch.** Strategy: vary `StartInteractive` outcomes through the stateful runtime; the mechanical guard is that its sole `StartResult` reaches existing initial Console attach while errors return nonzero before any fallback child effect, with typed operations remaining on their existing dispatcher.
- [x] **Step 2: Run `go test ./cmd/internal/couchcmd -run 'TestInteractiveLaunch.*(Resume|Root|Inventory|Refusal)' -count=1` and confirm the launch path still creates a root.**
- [x] **Step 3: Replace only the public launch's prepare/start sequence with `Couch.StartInteractive`; preserve the ordinary typed operation dispatcher and feed its `StartResult` into existing `runConsole` / `dispatchInitialAttach`.**
- [x] **Step 4: Run `go test ./cmd/internal/couchcmd -count=1` and confirm it passes.**

### Task 5: Restart-level acceptance and documentation

**Files:**
- Modify: `cmd/internal/couchcmd/run_test.go`
- Modify: `atlas/couch.md`
- Modify: `workshop/issues/000167-resume-unique-parked-root.md`

- [x] **Step 1: In `cmd/internal/couchcmd/run_test.go`, add a restart-level acceptance test using the existing `newRT` temp namespace/stateful runtime.** Strategy: transport production lifecycle ThreadStore and native-binding output across a fresh Couch construction; the mechanical guard is exact saved conversation and thread-address equality at initial Console root attach, without independently synthesizing either side.
- [x] **Step 2: Run the acceptance test alone and confirm it passes, then run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`.**
- [x] **Step 3: Update `atlas/couch.md` with the unique parked-root startup rule and pointers to the selector/orchestration. Update the issue checkboxes and Log with test evidence and the `ARCH-*` decisions.**
- [x] **Step 4: Run `go test ./... -count=1`, `make test`, and `git diff --check`.** Success means every command exits 0 with no failed Go/Lua/shell test and no whitespace errors. Record the exact passing commands in the issue Log before `sdlc close`.

## Revisions

### 2026-09-01 — Plan-quality gate PQ-1

Compressed enumerated prose cases across Tasks 1–5 into named-function adversarial input classes and mechanical guards. Concrete cases remain the responsibility of executable tests; the behavioral scope and architecture are unchanged.

### 2026-09-01 — Close review BR-1

The first acceptance test stopped at `dispatchInteractiveStart`, below the production `runTypedOperation` branch and `dispatchInitialAttach`. Reopened Task 5 steps 1–2: the replacement must traverse production interactive routing over reconstructed durable state and observe the resumed identity at initial Console attachment.
