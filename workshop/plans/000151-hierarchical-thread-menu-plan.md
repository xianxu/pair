# Hierarchical Work-Thread Menu Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Couch's transitional flat panel with a non-blocking hierarchical switcher over verified live and verified parked TTY work threads.

**Architecture:** A pure Couch-core projection joins durable thread records with exact live-owner TTY observations and exact parked-resume binding observations, then emits only actionable rows. A pure menu reducer owns frames, filtering, selection, forms, reconciliation, and bounded rendering; `couchtty.Console` remains the thin event/TTY/operation shell. Start preview and launch share an opaque resolution token, while inventory and preview I/O run through bounded single-flight/coalescing controllers.

**Tech Stack:** Go, Couch's existing `couchcore` operation table and stores, `couchtty` terminal emulator/console, stateful `FakeRunner`/`FakeHost` integration doubles, Go unit tests/benchmarks.

---

## Core concepts

### Pure entities

| Name | Lives in | Planned change | Delivery | Current at M3 boundary |
|------|----------|----------------|----------|---------------|
| `ActionableThreadSummary` | `cmd/internal/couchcore/actionableinventory.go` | new | M1/M3 | present |
| `LiveTTYObservation` | `cmd/internal/couchcore/actionableinventory.go` | new | M1 | present |
| `ParkedResumeObservation` | `cmd/internal/couchcore/actionableinventory.go` | new | M3 | present |
| `ProjectActionableThreads` | `cmd/internal/couchcore/actionableinventory.go` | new | M1/M3 | present |
| `StartResolution` | `cmd/internal/couchcore/startresolution.go` | new | M1 | present |
| `StartResolutionFingerprint` | `cmd/internal/couchcore/startresolution.go` | new | M1 | present |
| `ResolveStartResolution` | `cmd/internal/couchcore/startresolution.go` | new | M1 | present |
| `MenuState` | `cmd/internal/couchtty/menu.go` | new | M2 | present |
| `MenuFrame` | `cmd/internal/couchtty/menu.go` | new | M2 | present |
| `MenuEvent` | `cmd/internal/couchtty/menu.go` | new | M2 | present |
| `ReduceMenu` | `cmd/internal/couchtty/menu.go` | new | M2 | present |
| `MenuLayout` | `cmd/internal/couchtty/menu_render.go` | new | M2 | present |
| `AgeBand` | `cmd/internal/couchtty/menu_render.go` | new | M2 | present |
| `RenderMenu` | `cmd/internal/couchtty/menu_render.go` | new | M2 | present |
| `PreviewSchedule` | `cmd/internal/couchtty/menu_async.go` | new | M2 | present |
| `AdvancePreviewSchedule` | `cmd/internal/couchtty/menu_async.go` | new | M2 | present |
| `RefreshSchedule` | `cmd/internal/couchtty/menu_refresh.go` | new | M3 | present |
| `AdvanceRefreshSchedule` | `cmd/internal/couchtty/menu_refresh.go` | new | M3 | present |
| `PanelKey` | `cmd/internal/couchtty/panelkeys.go` | modified | M2 | modified, present |
| `DecodePanelKeys` | `cmd/internal/couchtty/panelkeys.go` | modified | M2 | modified, present |
| `PanelModel` | `cmd/internal/couchtty/panel.go` | deleted | M3 | deleted |
| `PanelModel.Filter` | `cmd/internal/couchtty/panel.go` | deleted | M3 | deleted |

- **`ActionableThreadSummary` / `LiveTTYObservation` / `ParkedResumeObservation` / `ProjectActionableThreads`** — the only interpretation that turns internal thread lifecycle into user-facing `live` or `parked` rows.
  - **Relationships:** N durable `ThreadRecord`s, N exact live-owner observations, and N exact parked-resume observations produce 0..N `ActionableThreadSummary` rows; each row retains one composite `ThreadAddress`. A live row requires one observation matching one durable incarnation's PID and process identity. A parked row requires exact `VerifiedPark`, no active `Park`, no occupied incarnation, and exactly one non-empty native binding observation whose agent matches the saved launch profile.
  - **DRY rationale:** terminal, future owner-routed clients, and tests consume one lifecycle projection rather than independently treating persisted `live` as TTY proof. Existing `ThreadSummary`/`BuildThreadInventory` remain unchanged, explicitly raw diagnostics for `couch list/show` and recovery tools; their persisted-incarnation `Live()` semantics never leak into the actionable type.
  - **Future extensions:** a cluster owner from #147 can supply equally strong remote observations without widening the state projection.

- **`StartResolution` / `StartResolutionFingerprint` / `ResolveStartResolution`** — immutable canonical path, selected agent, exact argv/source, candidate policy evidence, preference revision, default digest, and deterministic fingerprint shared by preview and launch.
  - **Relationships:** one start-form generation has at most one accepted resolution. The owner wraps that pure resolution in one random, expiring `StartGrant`; one grant authorizes at most one `start` attempt.
  - **DRY rationale:** factors the current `Couch.resolveLaunchProfile` behavior so preview and spawn cannot resolve through parallel algorithms or a check/use gap.
  - **Future extensions:** additional agent-specific defaults add fields to the resolution inputs and token material in this one unit.

- **`MenuState` / `MenuFrame` / `MenuEvent` / `ReduceMenu`** — immutable-by-copy hierarchical UI state and its total transition function.
  - **Relationships:** one menu owns one root frame plus a bounded stack of thread action/confirmation/input frames and at most one global start-form frame. Thread-bound frames capture one exact `ThreadAddress`; reconciliation walks root-to-leaf and drops the first invalid frame plus descendants.
  - **DRY rationale:** keyboard events, async completions, inventory refreshes, and operation results all use the same transition authority.
  - **Future extensions:** new thread actions add leaf descriptors and reducer events without adding Console state.

- **`MenuLayout` / `AgeBand` / `RenderMenu`** — bounded, terminal-independent render decision and ANSI output.
  - **Relationships:** one `MenuState` plus terminal dimensions and clock input produces one frame; frames reference inventory rows rather than copying them.
  - **DRY rationale:** wide/narrow placement, clipping, selection, age bands, controls, and resize behavior live outside Console I/O.
  - **Future extensions:** another terminal frontend can consume the layout model without duplicating menu transitions.

- **`PreviewSchedule` / `AdvancePreviewSchedule`** — pure one-running/one-coalesced-latest generation state.
  - **Relationships:** one menu lifetime owns one schedule and one monotonic identity sequence across every start form; edits replace its single pending request, and a pending submit is bound to exactly one identity.
  - **DRY rationale:** cancellation, stale completion, and armed-submit rules are tested once rather than encoded across goroutine branches.
  - **Future extensions:** none planned; this is deliberately not a generic job scheduler.

- **`RefreshSchedule` / `AdvanceRefreshSchedule`** — pure one-running/one-dirty-bit inventory refresh state.
  - **Relationships:** one Console owns one schedule; any number of attach, exit, and operation events coalesce into at most one follow-up refresh after the running generation terminates.
  - **DRY rationale:** initial-unavailable, last-good, stale completion, and dirty follow-up behavior use one transition authority rather than goroutine-local flags.
  - **Future extensions:** a remote owner may change the provider latency, but not the single-flight contract.

- **`PanelKey` / `DecodePanelKeys`** — existing framed terminal decoder widened with Tab plus four-direction CSI/SS3 cursor input.
  - **Relationships:** raw bytes map to semantic keys before the reducer sees them.
  - **DRY rationale:** all menu frame kinds consume shared semantic keys rather than terminal-mode-specific bytes.
  - **Future extensions:** add a semantic key only when a menu contract requires it.

- **Deleted `PanelModel` / resolver-driven `Filter`** — replaced by `MenuState`; ordinary filter keystrokes match the in-memory actionable rows and never call the durable resolver.
  - **Relationships:** existing compatibility tests move to `menu_test.go`; `ResolveThreadReference` remains for CLI/operation ref lookup, not keystroke filtering.
  - **DRY rationale:** removes the old competing flat panel state and the synchronous resolver I/O hot path.
  - **Future extensions:** N/A.

### Integration points

| Name | Lives in | Planned change | Delivery | Current at M3 boundary | Wraps |
|------|----------|----------------|----------|---------------|-------|
| `Couch.ActionableThreadInventory` | `cmd/internal/couchcore/actionableinventory.go`, `cmd/internal/couchcore/artifactcollision.go`, `cmd/internal/couchcore/couch.go`, `cmd/internal/couchcore/parktransaction.go`, `cmd/internal/couchcore/pathops.go`, `cmd/internal/couchcore/resume.go`, `cmd/internal/couchcore/threadstore.go`, `cmd/internal/launcher/agent_defaults.go` | new | M1/M3 | present | `ThreadStore.Snapshot`, `PathOps`, artifact/binding proof, valid agent profiles, and live-owner observations |
| `NativeBindingResolver` | `cmd/internal/couchcore/resume.go` | new | M3 | context-bearing exact parked-resume binding resolution present | session inventory exact established-root query |
| `SessionInventoryNativeBindingResolver` | `cmd/internal/couchcore/resume.go`, `cmd/internal/couchcore/couch.go`, `cmd/internal/sessioninventory/model.go`, `cmd/internal/sessioninventory/query.go`, `cmd/internal/sessioninventory/runtime.go` | new | M3 | context-bearing exact parked-resume binding resolution present | session inventory runtime, model, and exact established-root query |
| `Couch.PrepareStart` | `cmd/internal/couchcore/couch.go`, `cmd/internal/couchcore/startargs.go`, `cmd/internal/couchcore/startgrant.go` | new | M1 | present | start arguments and grant issuance |
| `Couch.SpawnPrepared` | `cmd/internal/couchcore/couch.go`, `cmd/internal/couchcore/registry.go`, `cmd/internal/couchcore/runner.go`, `cmd/internal/couchcore/startargs.go`, `cmd/internal/couchcore/startgrant.go`, `cmd/internal/couchcore/worktree.go` | new | M1 | present | grant consumption, worktree/registry state, and runner launch |
| `StartGrantStore` | `cmd/internal/couchcore/startgrant.go`, `cmd/internal/couchcore/clock.go` | new | M1 | present | owner-local clock, random issuance, TTL, and atomic consumption |
| `OperationCall` | `cmd/internal/couchcore/operationdispatch.go`, `cmd/internal/couchcore/ops.go` | modified | M1 | context dispatch present | declared owner operation and cancellation |
| `DispatchOperation` | `cmd/internal/couchcore/operationdispatch.go`, `cmd/internal/couchcore/ops.go` | modified | M1 | context dispatch present | owner operation declaration and executor routing |
| `Couch.AbortStarted` | `cmd/internal/couchcore/couch.go`, `cmd/internal/couchcore/naming.go`, `cmd/internal/couchcore/ops.go`, `cmd/internal/couchcore/registry.go`, `cmd/internal/couchcore/runner.go`, `cmd/internal/couchcore/store.go`, `cmd/internal/couchcore/worktree.go` | new | M1 | exact started-actor abort present | exact handle, registry/store, naming, and worktree cleanup |
| `Console` | `cmd/internal/couchtty/console.go`, `cmd/internal/couchcore/actorid.go`, `cmd/internal/couchcore/operationdispatch.go`, `cmd/internal/couchcore/ops.go`, `cmd/internal/couchcore/park.go`, `cmd/internal/couchcore/ptyrunner.go`, `cmd/internal/couchcore/starttransaction.go`, `cmd/internal/couchcore/thread.go`, `cmd/internal/couchcore/worktree.go`, `cmd/internal/couchtty/console_menu.go`, `cmd/internal/couchtty/focus.go`, `cmd/internal/couchtty/keys.go`, `cmd/internal/couchtty/menu.go`, `cmd/internal/couchtty/menu_async.go`, `cmd/internal/couchtty/menu_refresh.go`, `cmd/internal/couchtty/notice.go`, `cmd/internal/couchtty/panelkeys.go`, `cmd/internal/couchtty/reserve.go`, `cmd/internal/hostty/host.go`, `cmd/internal/ptychild/child.go`, `cmd/internal/ptychild/screen.go` | modified | M3 | hierarchical render, refresh, preview, action, and transactional attach controllers present | host input/output, pane observations, lifecycle identities, and bounded async workers |
| `wireResolver` | `cmd/internal/couchcmd/run.go`, `cmd/internal/couchcore/actionableinventory.go`, `cmd/internal/couchcore/couch.go`, `cmd/internal/couchcore/operationdispatch.go`, `cmd/internal/couchcore/ops.go`, `cmd/internal/couchtty/console.go` | modified | M3 | actionable refresh, shared action, attach-abort, and hierarchical render wiring present | Couch core providers, operation declarations, and Console dispatcher |
| `Runner` | `cmd/internal/couchcore/runner.go` | modified | M1 | context-bearing child lifecycle present | cancelable child lifecycle contract |
| `FakeRunner` | `cmd/internal/couchcore/runner_fake.go`, `cmd/internal/couchcore/runner.go`, `cmd/internal/ptychild/child.go`, `cmd/internal/ptychild/fake.go` | modified | M1 | stateful context-bearing double present | runner contract and fake terminal child lifecycle |
| `hostty.FakeHost` | `cmd/internal/hostty/fake.go`, `cmd/internal/ptychild/child.go` | modified | M3 | observable terminal double present | terminal sizing and emitted frames |
| `TestMenuTargetPerformance` | `cmd/internal/couchtty/menu_perf_test.go`, `cmd/internal/couchcore/actionableinventory.go`, `cmd/internal/couchtty/focus.go`, `cmd/internal/couchtty/menu.go`, `cmd/internal/couchtty/menu_refresh.go`, `cmd/internal/ptychild/child.go` | new | M3 | present | actionable rows, focus/input, refresh, terminal dimensions, clock samples, and deterministic four-worker CPU load |

- **`Couch.ActionableThreadInventory`** — snapshots durable records, resolves exact established native bindings for structurally eligible parked records, then calls the pure projector with live and parked proof observations.
  - **Injected into:** Console refresh worker; Console alone derives live observations from its registered panes and child identities, while `NativeBindingResolver` supplies context-bearing parked proof through session inventory.
  - **Future extensions:** #147 may inject remote owner observations with the same identity shape.

- **`NativeBindingResolver` / `SessionInventoryNativeBindingResolver`** — context-bearing dependency that converts session inventory's exact established-root query into one parked-resume observation; provisional, ambiguous, unbound, malformed, and canceled resolution never publishes actionable parked proof.

- **`Couch.PrepareStart` / `Couch.SpawnPrepared`** — performs canonical path/repository/default I/O once per preview and revalidates at submit before consuming the same accepted resolution in spawn.
  - **Injected into:** the shared live-owner operation executor; Console knows only operation calls/results.
  - **Future extensions:** remote operation routing can transport the JSON-safe resolution/token without transporting a terminal handle.

- **`StartGrantStore`** — bounded owner-local capability table joining an opaque random token to one immutable resolution/fingerprint.
  - **Injected into:** `Couch.PrepareStart` issues and `Couch.SpawnPrepared` atomically consumes. Capacity is 16 total issued-plus-consuming grants with a five-minute pre-claim TTL; full capacity refuses issuance rather than evicting authority, and Couch restart intentionally invalidates every grant.
  - **Future extensions:** #147 may move issuance/consumption behind the owner transport; it must not persist tokens as durable thread authority.

- **Shared operations and post-start cleanup** — add an `EffectAuthority` `prepare-start` owner operation, require the existing `start` operation to carry the accepted token, and carry a runtime-only `context.Context` through `OperationCall`. Expose `Couch.AbortStarted(StartResult, error)` as the failure half of owner-local attach; it delegates to the existing exact-handle quiesce/reconcile path before returning. Map UI `rename` to the existing shared `name` operation rather than declaring a duplicate verb.
  - **Injected into:** Console and future advisor clients through `DispatchOperation`.
  - **Future extensions:** none in #151; attach/detach remains explicitly out of scope.

- **`Console` menu controller** — owns event order, asynchronous refresh/preview workers, pane observation snapshots, dispatch, clear-and-replay switching, and the single terminal writer.
  - **Injected into:** existing `Console.Run` select loop; pure reducer/render functions receive values, not Console callbacks.
  - **Future extensions:** none; richer workspace panes belong outside Couch.

- **`wireResolver` composition** — replaces the old resolver-per-keystroke wiring with actionable inventory, start preparation, agent inventory, and operation dispatch wiring.
  - **Injected into:** the sole console startup path in `runConsole`.
  - **Future extensions:** #147 can replace local closures with owner transport.

- **Context-bearing runner and existing stateful doubles** — the blocked-start runner seam accepts the operation context so a canceled Console cannot strand a helper launch or registration wait. `FakeRunner` records the same before/after-ack cancellation states as `ExecRunner` and `PtyRunner`; `TestBlockedRunnerCancellationConformance` in `runner_contract_test.go` runs their shared trace on every core package test and M1 boundary. `hostty.FakeHost` retains terminal bytes/modes. Pair/Zellij suspend/resume continues through `PairLifecycleController` and its existing fake/live conformance suite (ARCH-MOCK).
  - **Injected into:** runner conformance, full-console, and current park/resume integration tests.
  - **Future extensions:** every new modeled runner transition extends the shared trace in the same change.

- **Target performance harness** — measures committed 100-row scripted interactions on M2 Max and optionally starts exactly four bounded CPU workers over fixed in-memory buffers.
  - **Injected into:** test-only measurement entrypoint; production contains no benchmark flags or load generator.
  - **Future extensions:** retain fixtures for regression; do not generalize into a benchmarking framework.

## Function-level test strategies

The Spec and Done-when own the behavioral cases. Implementation tests use these
compact strategies rather than copying that matrix back into prose:

| Risky function/seam | Adversarial class and mechanical guard |
|---|---|
| `ProjectActionableThreads` | Contradictory durable/observed lifecycle evidence → table/property checks require exact identity and fail-closed output. |
| `StartGrantStore.Issue/Claim/Finish` | Short entropy, collision, expiry, capacity, and concurrent claim traces → compare against a mutex-free reference state machine and run the race detector at M1. |
| `ResolveStartResolution` / fingerprint | Mutate every normalized input and argv element → equality/property checks require stable same-input and changed fingerprint for changed authority. |
| `PrepareStart` / `SpawnPrepared` / `ReconcileAdmissionPrepared` | Stateful fakes change policy/preference/default evidence between phases → effect logs prove refusal precedes allocation/fork and authorized values are not reread afterward. |
| `DispatchOperation` | Unknown schemas, absent capabilities, and implicit-argument injection → declaration-driven table requires exactly one executor and no UI-only verb. |
| `ClassifyThreadReferenceFields` / set filter | Arbitrary exact/fuzzy collisions and invalid bytes → shared classification oracle proves exact wins without store access. |
| `ReduceMenu` | Generated event traces over stale identities, UTF-8 bounds, and async outcomes → invariant checks bound stack/input, preserve stable identity, and forbid duplicate effects. |
| `RenderMenu` / layout / age band | Boundary dimensions, ages, viewport positions, and malformed/wide Unicode → goldens plus stripped-width/row bounds; no clock or terminal IO. |
| `DecodePanelKeys` | Every split point plus malformed/modified CSI-u and mouse noise → seeded fuzz requires framing, ordering, and fail-safe dropping. |
| `AdvancePreviewSchedule` / `AdvanceRefreshSchedule` | Duplicate, stale, canceled, and reordered completions → model traces enforce their one-running/coalesced bounds and one terminal retirement. |
| Console refresh/action/attach controllers | Blocking stateful providers and partial pane installation → deterministic barriers prove immediate repaint, exact rollback, context cancellation, and joined teardown. |
| `Runner.StartBlocked` cancellation contract | One shared trace drives `FakeRunner`, `ExecRunner`, and `PtyRunner`: fake transitions `blocked → canceled-before-ack → reaped` and `blocked → acknowledged → canceled-after-ack → reaped` must match real helper/PTY behavior. `TestBlockedRunnerCancellationConformance` runs on every `go test ./cmd/internal/couchcore` and therefore at each M1/boundary change, not on a separate manual cadence. |
| 100-row Console/render performance | Fixed generated inventory and generation-tagged input/repaint barriers → allocation ceilings and target-machine p50/p95/max checks enforce the operating envelope. |

## Chunk 1: Authoritative actionable inventory and token-bound start

### Task 1: Project durable records plus exact owner observations

**Files:**
- Create: `cmd/internal/couchcore/actionableinventory.go`
- Create: `cmd/internal/couchcore/actionableinventory_test.go`
- Modify: `cmd/internal/couchcore/threadinventory.go`
- Modify: `cmd/internal/couchcore/threadinventory_test.go`

- [x] **Step 1: Write failing pure projection tests**

Apply the `ProjectActionableThreads` contradictory-evidence strategy above; the
test oracle is the Spec's two actionable states and exact composite identity.

- [x] **Step 2: Run the focused test and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestProjectActionableThreads' -count=1`

Expected: FAIL because `LiveTTYObservation`, user-facing state, and projector do not exist.

- [x] **Step 3: Implement the pure projection**

Add the types named in Core concepts and the pure projector with fail-closed
state, defensive copies, deterministic sorting, and summary display methods.
Keep `ThreadSummary.Live()` and raw diagnostic inventory unchanged.

- [x] **Step 4: Write failing snapshot-wrapper tests**

Apply the `Couch.ActionableThreadInventory` IO-boundary strategy: injected
snapshot failure must remain distinguishable from empty actionable output and
all values must be defensively owned.

- [x] **Step 5: Run wrapper tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestActionableThreadInventory' -count=1`

Expected: FAIL because the wrapper is absent.

- [x] **Step 6: Implement the thin snapshot wrapper**

Implement the one-snapshot wrapper around `ProjectActionableThreads` without a
second lifecycle interpretation.

- [x] **Step 7: Run focused tests and verify GREEN**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'Test(ProjectActionableThreads|ActionableThreadInventory|BuildThreadInventory)' -count=1`

Expected: PASS.

- [x] **Step 8: Commit Task 1**

```bash
git add cmd/internal/couchcore/actionableinventory.go cmd/internal/couchcore/actionableinventory_test.go cmd/internal/couchcore/threadinventory.go cmd/internal/couchcore/threadinventory_test.go
git commit -m "#151 M1: project actionable Couch threads"
```

### Task 2: Issue bounded one-attempt start grants

**Files:**
- Create: `cmd/internal/couchcore/startgrant.go`
- Create: `cmd/internal/couchcore/startgrant_test.go`
- Modify: `cmd/internal/couchcore/couch.go`

- [x] **Step 1: Write failing grant authority tests**

Apply the `StartGrantStore.Issue/Claim/Finish` reference-state strategy to entropy
failure, collision, replay, and atomic claim authority. Pin 32-byte raw-URL tokens
and three total collision draws.

- [x] **Step 2: Write failing lifecycle/bound tests**

Use the same model to pin the 16-entry total bound, five-minute pre-claim TTL,
non-evictable consuming state, terminal removal, and restart invalidation.

- [x] **Step 3: Run tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestStartGrant' -count=1`

Expected: FAIL because `StartGrantStore` does not exist.

- [x] **Step 4: Implement minimal grant storage**

Implement the Core-concepts grant lifecycle behind one mutex with cloned
resolution ownership and existing clock/entropy injection; grants remain
owner-local and non-persistent.

- [x] **Step 5: Run focused and package tests**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestStartGrant' -count=1`

Expected: PASS, including concurrent claims and capacity while consuming.

- [x] **Step 6: Commit Task 2**

```bash
git add cmd/internal/couchcore/startgrant.go cmd/internal/couchcore/startgrant_test.go cmd/internal/couchcore/couch.go
git commit -m "#151 M1: add bounded start grants"
```

### Task 3: Prepare, revalidate, and consume one exact start resolution

**Files:**
- Create: `cmd/internal/couchcore/startresolution.go`
- Create: `cmd/internal/couchcore/startresolution_test.go`
- Modify: `cmd/internal/couchcore/couch.go`
- Modify: `cmd/internal/couchcore/couch_test.go`
- Modify: `cmd/internal/couchcore/admission.go`
- Modify: `cmd/internal/couchcore/admission_reconcile_test.go`

- [x] **Step 1: Write failing pure resolution/fingerprint tests**

Apply the `ResolveStartResolution` mutation strategy to every authority-bearing
input and malformed profile/policy data; reuse `ResolveLaunchProfile` as the
selection oracle.

- [x] **Step 2: Run pure tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestResolveStartResolution' -count=1`

Expected: FAIL because the resolution entity is absent.

- [x] **Step 3: Implement the pure resolution**

Implement the Core-concepts resolution/fingerprint with an explicit
length-delimited schema and defensive copies. Re-run Step 2 and expect PASS.

- [x] **Step 4: Write failing prepared-I/O and admission tests**

Apply the stateful evidence-change strategy to `PrepareStart`, `SpawnPrepared`,
and `ReconcileAdmissionPrepared`; effect logs are the mechanical guard against
check/use drift and unauthorized allocation/fork.

- [x] **Step 5: Run prepared-start tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'Test(PrepareStart|SpawnPrepared|ReconcileAdmissionPrepared)' -count=1`

Expected: FAIL because the prepared I/O/admission seam is absent.

- [x] **Step 6: Implement prepared start and candidate-policy admission**

Factor one resolution pipeline for prepare/revalidation, then feed its accepted
values directly into prepared admission and launch. Grant claim/finish brackets
the attempt; no authority input is reread after fingerprint acceptance.

- [x] **Step 7: Run focused core tests**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'Test(ResolveStartResolution|PrepareStart|SpawnPrepared|ReconcileAdmissionPrepared|StartGrant)' -count=1`

Expected: PASS.

- [x] **Step 8: Commit Task 3**

```bash
git add cmd/internal/couchcore/startresolution.go cmd/internal/couchcore/startresolution_test.go cmd/internal/couchcore/couch.go cmd/internal/couchcore/couch_test.go cmd/internal/couchcore/admission.go cmd/internal/couchcore/admission_reconcile_test.go
git commit -m "#151 M1: bind preparation to spawn"
```

### Task 4: Route preview and token-bound start through shared operations

**Files:**
- Modify: `cmd/internal/couchcore/ops.go`
- Modify: `cmd/internal/couchcore/ops_declarations_test.go`
- Modify: `cmd/internal/couchcore/operationdispatch.go`
- Modify: `cmd/internal/couchcore/operationdispatch_test.go`
- Modify: `cmd/internal/couchcore/plan_contract_test.go`
- Modify: `cmd/internal/couchcmd/run.go`
- Modify: `cmd/internal/couchcmd/run_test.go`
- Modify: `atlas/couch.md`

- [x] **Step 1: Write failing operation/CLI contract tests**

Apply the declaration-driven `DispatchOperation` strategy to the new
`prepare-start` result/effect and implicit token-bound `start`. Pin separately
that ownership acquisition does not imply Console/PTy creation.

- [x] **Step 2: Run tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(Operation|WantsConsole|ConsoleRunner|RunWithRuntime|RenderStartResolution)' -count=1`

Expected: FAIL because the declarations, result, and two-step CLI path are absent.

- [x] **Step 3: Implement operation and CLI wiring**

Add the declared authority/result surfaces and route the existing public start
command through prepare then token-bound start. Keep owner acquisition and
Console creation as separate decisions.

- [x] **Step 4: Run unfiltered changed-package tests**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`

Expected: PASS, including grant, admission, raw/actionable inventory, operation, and Console-routing tests.

- [x] **Step 5: Commit Task 4**

```bash
git add cmd/internal/couchcore/ops.go cmd/internal/couchcore/ops_declarations_test.go cmd/internal/couchcore/operationdispatch.go cmd/internal/couchcore/operationdispatch_test.go cmd/internal/couchcore/plan_contract_test.go cmd/internal/couchcmd/run.go cmd/internal/couchcmd/run_test.go
git commit -m "#151 M1: route token-bound start operations"
```

- [x] **Step 6: Update the Couch atlas for M1**

Document the proof-bearing actionable projection versus raw diagnostic
inventory, the 16-entry/256-bit/five-minute owner-local grant lifecycle, and
prepared candidate-policy admission in `atlas/couch.md`. Confirm
`atlas/index.md` already links `couch.md`; add no duplicate link.

```bash
git add atlas/couch.md
git commit -m "#151 M1: map switcher core contracts"
```

- [ ] **Step 7: Close M1 boundary**

Run focused package verification, then:

```bash
sdlc milestone-close --issue 151 --milestone M1 --verified 'actionable projection and token-bound start tests pass with -p 20'
```

Expected: fresh-context milestone review approves the core contracts; log the verdict in the issue.

## Chunk 2: Pure hierarchical menu, rendering, keys, and preview scheduling

### Task 5: Build the in-memory matcher and total menu reducer

**Files:**
- Create: `cmd/internal/couchtty/menu.go`
- Create: `cmd/internal/couchtty/menu_test.go`
- Modify: `cmd/internal/couchcore/threadmetadata.go`
- Modify: `cmd/internal/couchcore/threadmetadata_test.go`
- Modify: `cmd/internal/couchtty/panel.go`
- Modify: `cmd/internal/couchtty/panel_test.go`

- [x] **Step 1: Write failing shared matcher tests**

Apply the `ClassifyThreadReferenceFields` collision/invalid-byte strategy and
require both operation resolution and in-memory menu filtering to use the same
set-level exact-over-fuzzy helper.

- [x] **Step 2: Run matcher tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'Test(ThreadReferenceFields|ResolveThreadReference)' -count=1`

Expected: FAIL because the shared field predicate is absent.

- [x] **Step 3: Implement matcher extraction and verify GREEN**

Extract the pure classifier/set helper without store access, preserve CLI
resolution behavior, and rerun Step 2.

- [x] **Step 4: Write failing root/filter reducer tests**

Apply the `ReduceMenu` generated-trace strategy first to root/filter transitions;
stable identity and no-effect-on-zero-selection are the invariants.

- [x] **Step 5: Run root tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestReduceMenuRoot' -count=1`

Expected: FAIL because menu entities are absent.

- [x] **Step 6: Implement root entities/transitions and verify GREEN**

Add the Core-concepts menu types with fail-safe zero values, explicit frame
ownership, and root/filter transitions only; rerun Step 5.

- [x] **Step 7: Write failing action/confirmation traces**

Extend the reducer model traces across action/confirmation frames; captured
identity, current applicability, and ephemeral bell ownership are the guards.

- [x] **Step 8: Run action tests RED, implement, then GREEN**

Run before and after implementation: `go test -p 20 ./cmd/internal/couchtty -run 'TestReduceMenu(Action|Bell)' -count=1`

Expected: first FAIL, then PASS. Effects name existing shared operations (`name`,
not `rename`) and exact args but perform no I/O.

- [x] **Step 9: Write failing text/start form traces**

Extend the reducer trace generator across text/start forms with malformed UTF-8
boundaries, input caps, sticky agent choice, and exact originating-stack
restoration as invariants.

- [x] **Step 10: Run form tests RED, implement, then GREEN**

Run before and after implementation: `go test -p 20 ./cmd/internal/couchtty -run 'TestReduceMenu(Text|StartForm)' -count=1`

Expected: first FAIL, then PASS.

- [x] **Step 11: Write failing refresh/reconciliation traces**

Extend reducer model traces with reordered refresh/operation outcomes; the
mechanical guard reconciles root-to-leaf before restoration and forbids any
completion from redispatching an effect.

- [x] **Step 12: Run reconciliation tests RED, implement, then GREEN**

Run before and after implementation: `go test -p 20 ./cmd/internal/couchtty -run 'TestReduceMenu(Refresh|Reconcile|OperationResult)' -count=1`

Expected: first FAIL, then PASS. Keep maximum depth structural (root + one
thread action + one confirmation/input, or one global start overlay).

- [x] **Step 13: Keep the flat panel as a temporary compiling adapter**

Do not delete `PanelModel` yet. Adapt its construction/tests only as required by
the new actionable summary type so current `Console` and all package tests stay
buildable through M2. M3 migrates Console and deletes the adapter.

- [x] **Step 14: Run reducer and core matcher packages**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty -run 'Test(ThreadReference|ReduceMenu|Panel)' -count=1`

Expected: PASS.

- [x] **Step 15: Commit Task 5**

```bash
git add cmd/internal/couchcore/threadmetadata.go cmd/internal/couchcore/threadmetadata_test.go cmd/internal/couchtty/menu.go cmd/internal/couchtty/menu_test.go cmd/internal/couchtty/panel.go cmd/internal/couchtty/panel_test.go
git commit -m "#151 M2: reduce hierarchical menu state"
```

### Task 6: Render bounded wide/narrow menu frames

**Files:**
- Create: `cmd/internal/couchtty/menu_render.go`
- Create: `cmd/internal/couchtty/menu_render_test.go`

- [x] **Step 1: Write failing render/layout tests**

Apply the renderer boundary/Unicode strategy to `RenderMenu`,
`ChooseMenuLayout`, and `AgeBand`; stripped terminal width/height and visible
selection are the mechanical guards.

- [x] **Step 2: Run render tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(RenderMenu|ChooseMenuLayout|AgeBand|Clip)' -count=1`

Expected: FAIL because render/layout entities are absent.

- [x] **Step 3: Implement pure layout and rendering**

Implement the Core-concepts layout/renderer, reusing `reserve.go` sanitization
and column truncation. Viewports and child rectangles remain bounded; terminals
below 40x10 receive only the resize message.

- [x] **Step 4: Run render tests and verify GREEN**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(RenderMenu|ChooseMenuLayout|AgeBand|Clip)' -count=1`

Expected: PASS.

- [x] **Step 5: Commit Task 6**

```bash
git add cmd/internal/couchtty/menu_render.go cmd/internal/couchtty/menu_render_test.go
git commit -m "#151 M2: render bounded menu frames"
```

### Task 7: Decode Tab through the real terminal key seam

**Files:**
- Modify: `cmd/internal/couchtty/panelkeys.go`
- Modify: `cmd/internal/couchtty/panelkeys_test.go`

- [x] **Step 1: Write failing decoder tests**

Apply the seeded `DecodePanelKeys` framing/fuzz strategy to legacy HT and
unmodified Kitty CSI-u Tab encodings.

- [x] **Step 2: Run decoder tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestDecode.*Tab' -count=1`

Expected: FAIL because Tab is currently ignored.

- [x] **Step 3: Implement semantic Tab decoding**

Map only the two approved encodings to the semantic Tab key while preserving
partial framing and fail-safe dropping.

- [x] **Step 4: Run decoder tests and verify GREEN**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestDecode' -count=1`

Expected: PASS.

- [x] **Step 5: Commit Task 7**

```bash
git add cmd/internal/couchtty/panelkeys.go cmd/internal/couchtty/panelkeys_test.go
git commit -m "#151 M2: decode menu Tab encodings"
```

### Task 8: Model bounded preview scheduling and pending submit

**Files:**
- Create: `cmd/internal/couchtty/menu_async.go`
- Create: `cmd/internal/couchtty/menu_async_test.go`
- Modify: `cmd/internal/couchtty/menu.go`
- Modify: `cmd/internal/couchtty/menu_test.go`

- [x] **Step 1: Write failing scheduler traces**

Apply the `AdvancePreviewSchedule` reordered-completion model strategy; one
terminal outcome retires the matching generation and all work/submit bounds are
invariants.

- [x] **Step 2: Run scheduler tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(AdvancePreviewSchedule|ReduceMenu.*Preview)' -count=1`

Expected: FAIL because preview scheduling is absent.

- [x] **Step 3: Implement the pure scheduler and reducer events**

Implement the Core-concepts schedule as pure state/effects; cancellation is a
request and only a matching terminal outcome retires running work.

- [x] **Step 4: Run all Chunk 2 package tests**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1`

Expected: PASS with no store/process/terminal mocks in pure entity tests.

- [x] **Step 5: Commit Task 8**

```bash
git add cmd/internal/couchtty/menu_async.go cmd/internal/couchtty/menu_async_test.go cmd/internal/couchtty/menu.go cmd/internal/couchtty/menu_test.go
git commit -m "#151 M2: bound start preview scheduling"
```

### Task 9: Map and close the pure-menu boundary

**Files:**
- Modify: `atlas/couch.md`

- [x] **Step 1: Update the atlas**

Document the landed pure hierarchical frame/reducer model, shared in-memory matcher, wide/narrow renderer, Tab encodings, input bounds, reconciliation precedence, and one-running/one-latest preview scheduler. Explicitly state that the current Console still presents the flat compatibility panel until M3; do not describe the hierarchical UI as reachable yet.

- [x] **Step 2: Commit the M2 map**

```bash
git add atlas/couch.md
git commit -m "#151 M2: map hierarchical menu core"
```

- [ ] **Step 3: Close M2 boundary**

```bash
sdlc milestone-close --issue 151 --milestone M2 --verified 'pure reducer, renderer, key decoder, and bounded scheduler package tests pass with -p 20'
```

Expected: fresh-context milestone review approves the pure menu surface; log the verdict in the issue.

## Chunk 3: Console integration, performance evidence, and delivery

### Task 10: Feed exact pane observations through one asynchronous inventory refresh

**Files:**
- Create: `cmd/internal/couchtty/menu_refresh.go`
- Create: `cmd/internal/couchtty/menu_refresh_test.go`
- Create: `cmd/internal/couchtty/console_menu.go`
- Create: `cmd/internal/couchtty/console_menu_test.go`
- Modify: `cmd/internal/couchtty/console.go`
- Modify: `cmd/internal/couchtty/console_test.go`
- Modify: `cmd/internal/couchcmd/run.go`
- Modify: `cmd/internal/couchcmd/run_test.go`

- [x] **Step 1: Write failing pure refresh-schedule traces**

Apply the `AdvanceRefreshSchedule` reordered-completion model strategy and assert
the one-running/one-dirty bound.

- [x] **Step 2: Run scheduler tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestAdvanceRefreshSchedule' -count=1`

Expected: FAIL because `RefreshSchedule` is absent.

- [x] **Step 3: Implement the pure refresh schedule**

Return effects rather than spawning goroutines.

- [x] **Step 4: Run scheduler tests and verify GREEN**

Re-run Step 2. Expected: PASS.

- [x] **Step 5: Write failing observation/wiring tests**

Apply the Console refresh-controller strategy with exact pane identities and a
context-bearing actionable provider. Barriers prove opening/input use current
memory while blocked provider IO is cancelable and joined.

- [x] **Step 6: Run observation/wiring tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(Console.*Observation|Wire.*Actionable|MenuOpenDoesNotWait)' -count=1`

Expected: FAIL because pane identity and async actionable wiring are absent.

- [x] **Step 7: Implement the bounded inventory worker**

Wire one context-bound worker around `RefreshSchedule`; snapshot observations
under the Console mutex, perform provider IO outside it, and feed terminal
results only through `ReduceMenu`. Teardown cancels then joins.

- [x] **Step 8: Run focused and complete TTY/command tests**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`

Expected: PASS; blocked inventory tests prove opening/filtering/navigation do not wait or call the provider.

- [x] **Step 9: Commit Task 10**

```bash
git add cmd/internal/couchtty/menu_refresh.go cmd/internal/couchtty/menu_refresh_test.go cmd/internal/couchtty/console_menu.go cmd/internal/couchtty/console_menu_test.go cmd/internal/couchtty/console.go cmd/internal/couchtty/console_test.go cmd/internal/couchcmd/run.go cmd/internal/couchcmd/run_test.go
git commit -m "#151 M3: refresh actionable menu asynchronously"
```

### Task 11: Execute menu preview and actions without blocking input

**Files:**
- Modify: `cmd/internal/couchtty/operation_queue.go`
- Modify: `cmd/internal/couchtty/operation_queue_test.go`
- Modify: `cmd/internal/couchtty/park_latency_test.go`
- Modify: `cmd/internal/couchtty/console_menu.go`
- Modify: `cmd/internal/couchtty/console_menu_test.go`
- Modify: `cmd/internal/couchtty/console.go`
- Modify: `cmd/internal/couchtty/console_panel_regression_test.go`
- Modify: `cmd/internal/couchcore/operationdispatch.go`
- Modify: `cmd/internal/couchcore/operationdispatch_test.go`
- Modify: `cmd/internal/couchcore/startresolution.go`
- Modify: `cmd/internal/couchcore/startresolution_test.go`
- Modify: `cmd/internal/couchcore/couch.go`
- Modify: `cmd/internal/couchcore/couch_test.go`
- Modify: `cmd/internal/couchcore/launch_existing.go`
- Modify: `cmd/internal/couchcore/launchhelper.go`
- Modify: `cmd/internal/couchcore/launchhelper_test.go`
- Modify: `cmd/internal/couchcore/resume.go`
- Modify: `cmd/internal/couchcore/resume_launch_test.go`
- Modify: `cmd/internal/couchcore/runner.go`
- Modify: `cmd/internal/couchcore/runner_test.go`
- Create: `cmd/internal/couchcore/runner_contract_test.go`
- Modify: `cmd/internal/couchcore/runner_fake.go`
- Modify: `cmd/internal/couchcore/ptyrunner.go`
- Modify: `cmd/internal/couchcore/ptyrunner_test.go`
- Modify: `cmd/internal/couchcore/plan_contract_test.go`
- Modify: `cmd/internal/couchcore/starttransaction_integration_test.go`
- Modify: `cmd/probes/couchstartrecovery/main.go`
- Modify: `cmd/internal/couchcmd/run.go`
- Modify: `cmd/internal/couchcmd/run_test.go`

- [x] **Step 1: Write failing result-carrying operation queue tests**

Apply the Console action-controller strategy to every declared queued operation;
the model oracle guards typed results, duplicate coalescing, overload restoration,
single completion, and non-redispatch.

- [x] **Step 2: Run queue tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(OperationQueue|MenuOperationCompletion)' -count=1`

Expected: FAIL because completions currently discard results.

- [x] **Step 3: Implement result-carrying action execution**

Carry typed results through the existing bounded sequential queue. Accepted work
enters progress, overload reduces immediate failure, duplicates retain the
original progress, and returned lifecycle values re-enter the shared projector.

- [x] **Step 4: Run queue tests and verify GREEN**

Re-run Step 2. Expected: PASS, including the `park_latency_test.go` callback signature migration.

- [x] **Step 5: Write failing remaining lifecycle-context tests**

`OperationCall.Context`, `PrepareStart`, and `SpawnPrepared` landed in M1.
Apply the cancellation strategy to the remaining resume, Pair lifecycle,
leave, and runner consumers; executor barriers guard that supplied deadlines
reach each seam while a nil CLI context still means background.

- [x] **Step 6: Run operation-dispatch tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(DispatchOperationContext|PrepareStartCancellation|LifecycleOperationContext|ContextlessCLI)' -count=1`

Expected: existing dispatch/prepare context regressions PASS and the new
remaining lifecycle consumers FAIL because they still substitute background.

- [x] **Step 7: Implement operation/lifecycle context propagation**

Thread the already-carried operation context through resume admission, Pair
lifecycle calls, leave, and their operation wiring.

- [x] **Step 8: Run operation-dispatch tests and verify GREEN**

Re-run Step 6. Expected: PASS.

- [x] **Step 9: Write failing blocked-runner cancellation tests**

Add `TestBlockedRunnerCancellationConformance` using the shared fake/Exec/Pty
trace defined above, plus contract/compile checks for every `StartBlocked`
consumer. Registration timeout derives from the operation context.

- [x] **Step 10: Run blocked-runner tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/probes/couchstartrecovery -run 'Test(RunnerCancellation|LaunchHelperCancellation|TrackedLaunchCancellation|StartTransaction|Issue149BlockedRunnersDelegateToOneHandshakeAuthority)' -count=1`

Expected: FAIL at the contextless signature/behavior.

- [x] **Step 11: Implement blocked-runner cancellation**

Thread context through the runner/launch-helper seam and all consumers. Record
the fake transitions named in the shared conformance trace; reuse existing
exact post-ack cleanup.

- [x] **Step 12: Run blocked-runner tests and verify GREEN**

Re-run Step 10. Expected: PASS with every helper either transferred successfully or exactly canceled/reaped/reconciled.

- [x] **Step 13: Write failing Console action-lifetime tests**

Apply the Console action-controller teardown strategy to blocking lifecycle and
metadata dispatchers; cancellation observation, one completion, and a 250 ms
join bound are the guards.

- [x] **Step 14: Run Console action-lifetime tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestConsoleStopCancelsOperation' -count=1`

Expected: FAIL because accepted actions do not share one Console lifetime context.

- [x] **Step 15: Bind accepted actions to Console lifetime**

Create one Console lifetime context, bind every accepted action call to a child, and cancel it before joining the action worker.

- [x] **Step 16: Run Console action-lifetime tests and verify GREEN**

Re-run Step 14. Expected: PASS with one Console context owning every accepted action and worker join.

- [x] **Step 17: Write failing preview-worker cancellation tests**

Join the pure preview-schedule model to a blocking stateful dispatcher; generation
identity, one terminal completion, and cancel/join within 250 ms are the guards.

- [x] **Step 18: Run preview tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestConsole(StartPreview|PreviewCancellation|PendingSubmit)' -count=1`

Expected: FAIL because Console does not execute preview effects.

- [x] **Step 19: Implement preview/start effects**

Map preview/start schedule effects to context-bound declared operations and feed
typed results back through reducer/projector authority.

- [x] **Step 20: Run preview tests and verify GREEN**

Re-run Step 18. Expected: PASS.

- [x] **Step 21: Write failing exact-handle abort tests**

Apply the exact-handle cleanup strategy to `Couch.AbortStarted`; identity
validation and completed quiesce/reconciliation are the guards.

- [x] **Step 22: Run exact-handle abort tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestAbortStarted' -count=1`

Expected: FAIL because the owner method is absent.

- [x] **Step 23: Implement exact-handle abort**

Add the narrow owner method and reuse the existing cleanup authority without duplicating signals/reconciliation.

- [x] **Step 24: Run exact-handle abort tests and verify GREEN**

Re-run Step 22. Expected: PASS.

- [x] **Step 25: Write failing transactional pane-attach/Stop tests**

Apply the partial-install Console strategy to transactional attach and Stop;
exact identity removal, watcher join, routing restoration, terminal restoration,
and `AbortStarted` completion are the guards.

- [x] **Step 26: Run transactional attach tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(ConsoleAttachRollback|ConsoleStopDuringAttach|WireAttachAbort)' -count=1`

Expected: FAIL because attach has no transactional rollback/partial-Stop contract.

- [x] **Step 27: Implement transactional attach and partial-Stop cleanup**

Make attach commit/rollback one owner-local composition in `wireResolver`; cleanup finishes before reducer restoration.

- [x] **Step 28: Run transactional attach tests and verify GREEN**

Re-run Step 26. Expected: PASS.

- [x] **Step 29: Write semantic action/form restoration traces**

Drive Spec-defined operation outcomes through semantic events and the shared
Console/reducer model oracle. Exact dispatch identity, restoration only after
cleanup, frame reconciliation, and no redispatch are the guards.

- [x] **Step 30: Run restoration traces and verify RED**

Run:

`go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(DispatchOperationContext|PrepareStartCancellation|AbortStarted|Console(Menu|StartPreview|PreviewCancellation|PendingSubmit|Operation|Park|Resume|Bell|Refresh))' -count=1`

Expected: FAIL because the controller does not yet map every restoration case.

- [x] **Step 31: Implement remaining effect/restoration mappings**

Complete the thin mapping from reducer effects to declared operations, reusing
existing Alt+x and clear/replay ownership.

- [x] **Step 32: Run restoration traces and verify GREEN**

Re-run Step 30. Expected: PASS.

- [x] **Step 33: Commit Task 11**

```bash
git add cmd/internal/couchtty/operation_queue.go cmd/internal/couchtty/operation_queue_test.go cmd/internal/couchtty/park_latency_test.go cmd/internal/couchtty/console_menu.go cmd/internal/couchtty/console_menu_test.go cmd/internal/couchtty/console.go cmd/internal/couchtty/console_panel_regression_test.go cmd/internal/couchcore/operationdispatch.go cmd/internal/couchcore/operationdispatch_test.go cmd/internal/couchcore/startresolution.go cmd/internal/couchcore/startresolution_test.go cmd/internal/couchcore/couch.go cmd/internal/couchcore/couch_test.go cmd/internal/couchcore/launch_existing.go cmd/internal/couchcore/launchhelper.go cmd/internal/couchcore/launchhelper_test.go cmd/internal/couchcore/resume.go cmd/internal/couchcore/resume_launch_test.go cmd/internal/couchcore/runner.go cmd/internal/couchcore/runner_test.go cmd/internal/couchcore/runner_contract_test.go cmd/internal/couchcore/runner_fake.go cmd/internal/couchcore/ptyrunner.go cmd/internal/couchcore/ptyrunner_test.go cmd/internal/couchcore/plan_contract_test.go cmd/internal/couchcore/starttransaction_integration_test.go cmd/probes/couchstartrecovery/main.go cmd/internal/couchcmd/run.go cmd/internal/couchcmd/run_test.go
git commit -m "#151 M3: execute menu effects asynchronously"
```

### Task 12: Replace the flat compatibility panel with the hierarchical renderer

**Files:**
- Modify: `cmd/internal/couchtty/console.go`
- Modify: `cmd/internal/couchtty/console_test.go`
- Modify: `cmd/internal/couchtty/console_panel_regression_test.go`
- Modify: `cmd/internal/couchtty/core_concepts_contract_test.go`
- Delete: `cmd/internal/couchtty/panel.go`
- Delete: `cmd/internal/couchtty/panel_test.go`
- Modify: `cmd/internal/couchcmd/readme_test.go`
- Modify: `README.md`

- [x] **Step 1: Write failing real-loop/render replacement tests**

Drive the Spec's controls through raw `hostty.FakeHost` input and reuse the
decoder/reducer/controller strategies above. Generation-tagged repaint and host
mode snapshots guard input routing, resize, screen ownership, and teardown.
Add README expectation tests before editing prose.

- [x] **Step 2: Run replacement tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(ConsoleRun.*Menu|MenuControls|Readme.*Couch)' -count=1`

Expected: FAIL while Console still renders `PanelModel`.

- [x] **Step 3: Route all menu input/paint through reducer/renderer**

Make `MenuState`/`ReduceMenu`/`RenderMenu` the sole Console menu path and delete
the compatibility panel after references migrate. Update the concepts contract
to treat #151's table as the superseding status inventory without weakening
pure-source/direct-test enforcement.

- [x] **Step 4: Update user controls and migrate regressions**

Document the hierarchical menu and remove flat-panel wording in README. Move still-valid stable-selection, mouse, Escape, switch/replay, bell, park/resume, and terminal restoration assertions onto the new public behavior; delete tests only when their old symbol is deleted and their behavior is covered elsewhere.

- [x] **Step 5: Run complete changed packages**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`

Expected: PASS with no `PanelModel`, synchronous resolver-per-key path, or old prompt state remaining.

- [x] **Step 6: Commit Task 12**

```bash
git add cmd/internal/couchtty cmd/internal/couchcmd/readme_test.go README.md
git commit -m "#151 M3: switch Console to hierarchical menu"
```

### Task 13: Prove the operating envelope and close delivery

**Files:**
- Create: `cmd/internal/couchtty/menu_perf_test.go`
- Modify: `cmd/internal/couchtty/console_menu_test.go`
- Modify: `atlas/couch.md`
- Modify: `workshop/issues/000151-hierarchical-thread-menu.md`

- [x] **Step 1: Write the portable 100-row benchmark/bound tests**

Implement the Spec's fixed 100-row fixture and `BenchmarkMenu100`. Enforce the
declared allocation, worker/queue, input, and minimum-size bounds mechanically;
portable tests do not assert target-specific wall time.

- [x] **Step 2: Run portable tests and verify GREEN**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestMenu(100|Bounds|Feedback)' -count=1`

Expected: PASS with every allocation ceiling enforced; `go test -p 1 ./cmd/internal/couchtty -run '^$' -bench '^BenchmarkMenu100$' -benchmem -count=5` records allocation/ns-op evidence.

- [x] **Step 3: Implement and run the opt-in M2 Max protocol**

`TestMenuTargetPerformance` skips unless `PAIR_MENU_PERF_TARGET=m2-max`. Drive the committed Console/controller path with an instrumented `hostty.FakeHost`, generation-tagged frames, and a monotonic clock. For open and each filter/navigation event, start immediately before the raw input bytes enter the host input channel and stop when the sole host writer returns from the repaint carrying that generation. Measure render computation independently by starting immediately before the committed `RenderMenu(state, dimensions, now)` call and stopping when its ANSI frame bytes return, before host I/O. For refresh apply, start immediately before its completed result enters `Console.Run` and stop after the corresponding reconciled repaint returns. For blocked-operation feedback, start immediately before the confirming raw Enter enters the host and stop after the first repaint that visibly marks that exact operation in progress, while its dispatcher remains blocked. Fail a sample if no matching generation repaints or if another frame is mistaken for it.

Perform 20 warmups and 200 recorded samples per path, compute p50/p95/max, and run three trials: one baseline, then two with exactly four joined CPU goroutines hashing fixed in-memory buffers. Require p95 ≤50 ms for open input-to-repaint; ≤16 ms separately for each filter-byte input-to-repaint, navigation input-to-repaint, pure render computation, and refresh-result-to-repaint; and ≤100 ms for blocked-operation input-to-progress-repaint. Log OS/arch, target label, dimensions, counts, every statistic, the render call boundaries, and every input/repaint generation boundary.

Compile one optimized test executable on the named M2 Max with the default Go compiler settings—no race detector, coverage, or debug `gcflags`—so the production reducer/renderer/Console code has release optimization. Record `go version` and the binary SHA-256, then run it on AC power with Low Power Mode off:

```bash
go test -c -trimpath -o /tmp/couchtty-151-perf.test ./cmd/internal/couchtty
go version
shasum -a 256 /tmp/couchtty-151-perf.test
PAIR_MENU_PERF_TARGET=m2-max /tmp/couchtty-151-perf.test -test.run '^TestMenuTargetPerformance$' -test.count=1 -test.v
```

Expected: PASS in three trials. The helper always stops/joins four load goroutines; no process fan-out is introduced.

- [x] **Step 4: Run full automated verification with bounded parallelism**

Run: `go test -p 20 ./... -count=1`

Expected: PASS. Then run `git diff --check`; expected: no output.

- [x] **Step 5: Build and perform a clean-store live smoke**

Build: `go build -o /tmp/couch-151 ./cmd/couch`

Preflight in the target terminal. This protocol uses Claude explicitly so repository history/defaults cannot silently choose another harness:

```bash
command -v pair
command -v zellij
command -v claude
command -v rg
/tmp/couch-151 --help
```

Expected: every command succeeds. Then create a failure-safe, visibly named store and run Couch:

```bash
couch_smoke_store=$(mktemp -d "${TMPDIR%/}/couch-151-smoke.XXXXXX")
case "$couch_smoke_store" in
  "${TMPDIR%/}"/couch-151-smoke.*) ;;
  *) printf 'refusing unsafe smoke path: %s\n' "$couch_smoke_store" >&2; exit 1 ;;
esac
export COUCH_STORE_DIR="$couch_smoke_store"
export PAIR_AGENT=claude
printf '%s\n' "$couch_smoke_store"
/tmp/couch-151 start . --agent=claude
```

Execute and record this exact sequence:

1. In the root agent, send `reply only: root-ready` and wait for `root-ready`; this creates one exchange worth preserving. Open the switcher once and record the root row's opaque fallback tag as `smoke_root_tag` in the smoke notes.
2. Confirm that one root row is `live` and that open/filter/Up/Down are immediate, then press Ctrl-Space while the switcher is open. Submit path `.` with explicit `claude`; while start is blocked verify progress appears and keys still repaint. In the new agent send `reply only: child-ready` and wait for `child-ready`.
3. Press Ctrl-Space once to return from the child to home. Press Ctrl-Space once more to open the switcher and record the new child row's opaque fallback tag as `smoke_child_tag`. Select that fallback-tag row with Up/Down and press Enter exactly once; verify it switches from root to that child. Press Ctrl-Space once to return home, then once to reopen the switcher and verify both stable live-row identities remain. Do not press Ctrl-Space while the switcher is already open, because that intentionally opens the start form.
4. On the child row press Tab, run rename to `smoke-child`, then describe to `smoke description`; reopen the action frame and verify both survive refresh. Choose park, leave `cancel` selected and press Enter once, then choose the explicit `park smoke-child` row and confirm it becomes one ordinary `parked` row.
5. Press Enter on `smoke-child`; verify the exact thread resumes as `live` and retains name/description. Resize the terminal to 40 columns × 10 rows using the terminal emulator's size control, open the switcher, and record the visible single-column controls; shrink below 40×10 and record the resize request, then restore 120×40.
6. Return to the root actor, press Alt+x, confirm Leave Couch, and verify the shell returns only after both threads are parked. Move the mouse and type `printf 'tty-restored\n'`; expect no mouse escape bytes and exactly `tty-restored`.

On success, record the printed store path, both copied opaque tags, and all observations in the issue Log. On failure, record the failed numbered step and run `/tmp/couch-151 leave` against the same exported store if Couch no longer owns the terminal; if it still owns the terminal, use its Alt+x Leave flow first. After the shell returns, assign the two exact copied tags and mechanically refuse cleanup unless both diagnostic rows prove no agent is running:

```bash
printf 'recorded root tag: '
read -r smoke_root_tag
printf 'recorded child tag: '
read -r smoke_child_tag
case "$smoke_root_tag:$smoke_child_tag" in
  couch-*:couch-*) ;;
  *) printf 'refusing cleanup without two exact recorded tags\n' >&2; exit 1 ;;
esac
test "$smoke_root_tag" != "$smoke_child_tag" || {
  printf 'refusing cleanup: recorded tags are not distinct\n' >&2
  exit 1
}
smoke_list=$(/tmp/couch-151 list) || exit 1
printf '%s\n' "$smoke_list"
smoke_top_count=$(printf '%s\n' "$smoke_list" | awk 'NF && $0 !~ /^[[:space:]]/ {n++} END {print n+0}')
smoke_quiet_count=$(printf '%s\n' "$smoke_list" | rg -c '^  \(no agent running\)$' || true)
test "$smoke_top_count" -eq 2 && test "$smoke_quiet_count" -eq 2 || {
  printf 'refusing cleanup: store does not contain exactly two quiescent rows\n' >&2
  exit 1
}
smoke_labels=$(printf '%s\n' "$smoke_list" | awk 'NF && $0 !~ /^[[:space:]]/ {print $1}' | sort)
smoke_expected=$(printf '%s\n%s\n' "$smoke_root_tag" 'smoke-child' | sort)
test "$smoke_labels" = "$smoke_expected" || {
  printf 'refusing cleanup: diagnostic rows differ from recorded smoke rows\n' >&2
  exit 1
}
if printf '%s\n' "$smoke_list" | rg -q 'pid [0-9]+|^  (creating|live|unknown)$'; then
  printf 'refusing cleanup: store retains an occupied incarnation\n' >&2
  exit 1
fi
for smoke_tag in "$smoke_root_tag" "$smoke_child_tag"; do
  smoke_show=$(/tmp/couch-151 show "$smoke_tag") || exit 1
  printf '%s\n' "$smoke_show"
  printf '%s\n' "$smoke_show" | rg -q '^  \(no agent running\)$' || {
    printf 'refusing cleanup: %s is not quiescent\n' "$smoke_tag" >&2
    exit 1
  }
  if printf '%s\n' "$smoke_show" | rg -q 'pid [0-9]+|^  (creating|live|unknown)$'; then
    printf 'refusing cleanup: %s retains an occupied incarnation\n' "$smoke_tag" >&2
    exit 1
  fi
done
```

Only after both exact rows pass that state check, validate and remove the scoped directory:

```bash
test -n "$couch_smoke_store" && test -d "$couch_smoke_store"
case "$couch_smoke_store" in
  "${TMPDIR%/}"/couch-151-smoke.*) ;;
  *) printf 'refusing unsafe cleanup path: %s\n' "$couch_smoke_store" >&2; exit 1 ;;
esac
rm -rf -- "$couch_smoke_store"
```

Report the exact removed path in the Log; never substitute a workspace, home, or unresolved path.

- [x] **Step 6: Update atlas and issue evidence**

Update `atlas/couch.md` from “pure core planned integration” to current reachable hierarchical switcher, actionable/raw boundaries, async worker bounds, operation mapping, performance envelope, and suspend/resume-only application lifecycle. Confirm `atlas/index.md` still links it. Append target measurements/live-smoke evidence to the issue Log and tick implementation tasks only through SDLC gates.

- [x] **Step 7: Commit performance/docs evidence**

```bash
git add cmd/internal/couchtty/menu_perf_test.go cmd/internal/couchtty/console_menu_test.go atlas/couch.md workshop/issues/000151-hierarchical-thread-menu.md
git commit -m "#151 M3: verify hierarchical switcher envelope"
```

- [ ] **Step 8: Close M3 boundary**

```bash
sdlc milestone-close --issue 151 --milestone M3 --verified 'go test -p 20 ./... passes; M2 Max p95 protocol and clean-store live switch/park/resume/exit smoke pass'
```

Expected: fresh-context review approves M3 and the issue/project/atlas state is current.

- [ ] **Step 9: Close issue #151**

Run `sdlc actual --issue 151` to inspect measured attribution, then:

```bash
sdlc close --issue 151 --verified 'all three milestone reviews passed; full tests, target performance protocol, and clean-store operator workflow pass'
```

Expected: issue becomes done with measured actual time; do not type an estimated actual manually.

## Revisions

### 2026-08-30 — compress tests to function-level strategies and add runner conformance

**Reason:** the first `change-code` plan-quality gate found that the detailed
case/procedural inventories duplicated the Spec and executable tests, while the
new `Runner.StartBlocked` cancellation model lacked an explicit real/fake
conformance contract (`PQ-1`, `PQ-2`).

**Delta:** one function-level strategy table now names each risky function,
adversarial input class, and mechanical guard; task steps reference those
strategies instead of restating behavioral matrices. A shared always-run
`TestBlockedRunnerCancellationConformance` compares `FakeRunner`, `ExecRunner`,
and `PtyRunner`, including the fake's before/after-ack cancellation transitions
(ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-30 — M1 boundary review consumer and staging corrections

**Reason:** the first M1 boundary review found that the required-token schema
had not migrated the transitional Console consumer, malformed durable records
could enter the actionable projector, the exhaustive production-source
inventory omitted all three new files, and the atlas described future Console
wiring as current (`BR-1` through `BR-4`).

**Delta:** M1 now preserves the transitional Console by routing every start
through `prepare-start` then implicit token-bound `start`; actionable projection
validates the complete record before interpreting evidence; all new sources are
classified; and the atlas states that M1 exposes the authority while M3 adopts
it. Task 11 now treats `OperationCall.Context` plus prepare/spawn propagation as
already delivered and targets only the remaining lifecycle/runner consumers
(ARCH-DRY, ARCH-PURPOSE).

### 2026-08-30 — guard current-state staging across every durable summary

**Reason:** the second M1 boundary review found that correcting only the atlas
left the project milestone claiming that the ordinary switcher already consumed
the actionable projection (`BR-4`, `documentation-current-state-accuracy`).

**Delta:** M1 exposes the authority while M3 adopts it. The current source and
the atlas, project milestone, issue log, plan revisions, and README now form one
tested staging contract; M3 must migrate the consumer and update the complete
documentation class together (ARCH-PURPOSE).

### 2026-08-30 — qualify preview identity by menu lifetime and table state by boundary

**Reason:** the fourth M2 boundary review found that form-local generation one
could collide after Escape/reopen, and that the Core concepts tables described
final M3 files/deletions as current M2 state (`BR-16`, `BR-17`).

**Delta:** `MenuState` owns one monotonic preview sequence shared by every form
lifetime and edit; the composed reducer/scheduler trace proves a late old-form
completion cannot populate or submit the reopened form. Every Pure entities and
Integration points row now separates the planned final change, its delivery
milestone, and its actual status at the M2 boundary (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-30 — correlate every operation result to one dispatch attempt

**Reason:** the sixth M2 boundary review found that operation/address pairs
cannot distinguish a delayed duplicate from a newer identical dispatch
(`BR-18`, `operation-result-origin-correlation`).

**Delta:** `MenuState` allocates one monotonic nonzero operation-attempt
identity, propagated through every `MenuEffect`, captured
`MenuOperationOrigin`, and result `MenuEvent`. Exact attempt matching precedes
inventory acceptance or state mutation. The exhaustive operation × outcome ×
address-shape table now composes A completion, identical B dispatch, and stale
A replay for switch, resume, park, name, describe, and start; exhaustion
refuses dispatch (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-30 — identify frame instances independently from operation attempts

**Reason:** the seventh M2 boundary review found that kind/depth structural
coordinates can alias a replacement confirmation or draft after navigation
(`BR-19`, `operation-result-origin-correlation`).

**Delta:** every frame has a monotonic menu-lifetime instance identity captured
by its operation origin. Frame-local restoration requires that exact instance;
global successful park/resume/start restoration remains explicit. The complete
operation × outcome table replaces the origin frame before completion and
proves all frame-local actions preserve the replacement; frame-identity
exhaustion refuses navigation (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-31 — preserve unrelated later overlays during completion restoration

**Reason:** the eighth M2 boundary review found that exact origin identity
prevented a replacement frame from being mistaken for the origin, but global
park/resume/start restoration and park-failure restoration still sliced away a
start overlay legally opened after dispatch (`BR-20`,
`operation-result-origin-correlation`).

**Delta:** completion now transforms only the captured originating stack
prefix, then retains a distinct later global start overlay by frame-instance
identity unless target-invalid reconciliation has removed it. A real reducer
navigation table covers every operation across success and failure after the
later overlay opens (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-31 — advance the executable concept inventory through M3 Task 10

**Reason:** M2's boundary contract intentionally rejected future M3 files as
current. Landing the refresh scheduler and Console inventory controller changes
that filesystem truth while action execution and renderer replacement remain
pending.

**Delta:** the Core concepts tables and their executable contract now report
the refresh scheduler, context-bearing actionable provider, exact pane
observations, and bounded Console refresh worker as present after Task 10. They
continue to identify the flat panel, action/render migration, and performance
harness as unfinished (ARCH-PURPOSE, ARCH-CONSTRAINTS).

### 2026-08-31 — advance the executable concept inventory through M3 Task 11

**Reason:** Task 11 completed the context-bearing lifecycle, preview/action
controllers, exact started-actor abort, transactional terminal attach, and
semantic completion mapping that Task 10 still described as pending.

**Delta:** the Integration points table now records those controllers and
composition paths as present. The hierarchical renderer replacement and target
performance harness remain explicitly unfinished for Tasks 12 and 13
(`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — replace ancestor columns with breadcrumb single-surface layout

**Reason:** Task 12 live smoke made the planned wide/narrow multi-frame layout
misrepresent the global start form as a selected-thread child and spend most of
the switcher's width on inactive ancestors. The operator visually compared
collapsed rails, dimmed columns, and a breadcrumb single surface and approved
the latter.

**Delta:** Task 12's renderer replacement now emits exactly one left-anchored
active surface. Root threads and global start are separate level-zero surfaces;
nested frames derive a subdued breadcrumb from the retained reducer stack while
their parent bodies are hidden. Add pure layout/render tests for all frame kinds
and real-loop tests proving Ctrl-Space does not indent start, actions do not
retain the root body, Left/Escape back out, and Right follows Tab/Enter. Keep the
100-row computation budgets unchanged because the new renderer performs no IO
and renders no more content than the superseded layout (`ARCH-DRY`, `ARCH-PURE`,
`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — enumerate single-surface renderer and navigation acceptance

**Reason:** fresh-context review found that the approved plan revision did not
explicitly retire Task 6's child-rectangle behavior and described horizontal
aliases and breadcrumb ancestry too coarsely to provide deterministic test
oracles.

**Delta:** Task 12 supersedes Task 6's multi-body child rectangles and
`MenuLayoutWide`/`MenuLayoutNarrow` behavior while retaining its bounds,
clipping, selection, sanitation, and below-minimum guarantees. Its TDD table
covers root; global start opened from root, actions, and confirmation; actions;
park confirmation; rename; and describe at 120x40, 40x10, and below minimum.
Each supported-size case asserts one rectangle/body at X=0, exact breadcrumb
components, no parent body, bounded one-line breadcrumb clipping, and visible
selection or form focus.

Reducer and raw-host-loop tables cover the frame-specific Left/Right/Tab/Enter
matrix: root Right=Tab and Left=Escape; action/confirmation Right=Enter and
Left=Escape; text Left=Escape with Right/Tab no-op; start retains Tab field and
agent-field Left/Right semantics. Terminal cases include CSI/SS3 arrows and
HT/CSI-u Tab. Reconciliation cases invalidate or rename a target while nested
and beneath global start, then assert stale ancestry disappears, start remains
level zero, and Escape restores only the reconciled origin. Breadcrumb projection
runs after reconciliation, uses the current actionable display label, sanitizes
and clips to one row, and never consumes the minimum active-body viewport
(`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — cover direct leave confirmation as a distinct surface

**Reason:** follow-up plan review found that park confirmation does not cover
Alt+x leave, which has a root-plus-confirmation stack and must not synthesize an
actions breadcrumb component.

**Delta:** add Alt+x leave confirmation to Task 12's renderer table at 120x40,
40x10, and below minimum, with exact breadcrumb
`threads › <current actionable thread> › leave couch`, one body at X=0, and no
invented actions ancestor. Add it to the raw-key table with Left/Escape,
Right=Enter, Tab no-op, and Ctrl-Space/level-zero-start/Escape restoration cases
(`ARCH-DRY`, `ARCH-PURPOSE`).

### 2026-08-31 — add local message/cursor output and exact resume landing

**Reason:** live Task 12 smoke exposed three missing integration contracts:
non-root `MenuState` failures were not rendered, menu text fields inherited an
unrelated/hidden child cursor, and resume attached without exact presentation
landing while a queued exited pane could still trip duplicate checks.

**Delta:** extend Task 12 TDD before compatibility deletion. Add a pure rendered-
menu value containing bounded body plus optional cursor intent; keep string-only
rendering only as a compatibility wrapper until all tests migrate. Table-test
start path, rename/describe, and visible list filters with/without the approved
post-breadcrumb message banner at 120x40 and 40x10, including clipped and wide-
Unicode inputs. Test hidden cursor for non-text surfaces and resize. Add one
hostty-owned HideCursor constant and fake-host assertions that `showMenu` paints,
restores the Couch status row, then applies the pure cursor intent; actor replay
and teardown retain their existing cursor authority (`ARCH-DRY`, `ARCH-PURE`).

Give menu notices an explicit error vs informational/progress level. Render the
optional, sanitized, single-line banner directly below every breadcrumb; prefix
error-level text with `error:` and do not duplicate it into the agent-pane feed.
Reducer tables enumerate inventory, preview, operation, validation, stale-target,
no-selection, and resolving outcomes so no current notice producer silently
misses the shared banner (`ARCH-PURPOSE`).

Add stateful resume controller tests around `finishOperation`: a typed resume
result attaches first and force-switches the exact returned handle; a typed start
result still restores the switcher. Model the done-but-queued old pane in the
existing fake Console: duplicate attach and address switching ignore panes whose
child is done, while live duplicates refuse. Prove the resumed handle—not map
iteration order—receives replay, and that attach refusal invokes exact abort
before the local error banner. Run focused race plus the existing core/TTY/command
suites with `-p 20`; the keystroke path gains no IO or concurrency
(`ARCH-MOCK`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — exhaustively specify message, cursor, and completion matrices

**Reason:** fresh-context review found that the prior revision grouped message
producers, cursor states, and asynchronous outcomes. That left enumerable
siblings without deterministic oracles (`ARCH-PURPOSE`, `ARCH-PURE`).

**Delta:** Task 12 uses the following executable matrices in addition to the
normative Spec tables.

For every message-producer row in the Spec, reducer tests assert its exact typed
`info`/`error` value. One renderer table crosses each reachable result with
root; actions; park; leave; rename; describe; and start opened from root,
actions, and confirmation. Each runs at 120×40, 40×10, and 39×9. Supported
cases assert breadcrumb, optional banner, one separator, bounded active controls,
and no parent body. The 39×9 case asserts resize-only output, hidden cursor,
unchanged notice state, and the same banner after resize to 40×10. Ordinary
successful events clear notices; rejected stale completions preserve them.

The cursor table crosses every surface with message absent/present and all
three sizes. Root/action/park/leave filters cover empty, ASCII, combining,
double-width, and right-clipped input. Rename, describe, and selected start-path
cover the same inputs including empty; start-agent focus, selection-only,
unavailable, and resize always hide. For every visible case, tests strip ANSI,
locate the final field row, compute its clipped terminal-cell end with the
production width helper, and compare exact 1-based row/column intent. A banner
keeps the same column and adds exactly one row. A fake host asserts
`HideCursor → take-over/paint → status restore → MoveTo+ShowCursor` when visible,
the same prefix ending in `HideCursor` otherwise, actor return
`clear → replay → actor cursor`, and unconditional teardown `ShowCursor`
(`ARCH-DRY`, `ARCH-PURE`).

Stateful completion tests assert these exact traces:

- resume: `typed result → attach(exact handle) → reducer success/clear in-flight
  → force-switch(exact handle) → clear/replay → status/focus`; forbid abort and
  switcher repaint;
- start: `typed result → attach(exact handle) → reducer success/select returned
  address → refresh → switcher repaint`; forbid force-switch;
- attach refusal: `refusal → abort(exact actor, synchronously) → reducer failure
  → restore/reconcile origin → local error-banner paint`; forbid success, focus,
  and replay.

The done-pane table constructs old-done/new-live same-address handles and
asserts that admission skips done; a live duplicate refuses before mutation;
address lookup/switch never selects done; resume uses the returned handle rather
than map order; and delayed exit of the old handle cannot remove, activate, or
redirect the new handle. Run focused tests and race tests plus existing
core/TTY/command packages with `-p 20`; no command may create more than 20
package workers (`ARCH-CONSTRAINTS`).

### 2026-08-31 — require resume authority and animate pending operations

**Reason:** live smoke exposed two related trust/feedback gaps. A completed
shutdown without any established native transcript binding was projected as a
resumable `parked` row, and accepted operations could leave the switcher
visually static for their whole latency.

**Delta:** extend Task 12 before compatibility deletion with two TDD slices.

First, add `ParkedResumeObservation` as immutable proof input to the pure
`ProjectActionableThreads` projection. The Couch inventory shell snapshots
records, identifies structurally eligible verified parks, and asks the existing
`NativeBindingResolver` for the saved launch agent's exact binding outside the
Console and reducer. It contributes an observation only for
`BindingEstablished` plus a non-empty native ID; all other statuses/errors fail
closed for that row. Unit tests cover established, provisional, ambiguous,
unbound, empty-root, resolver error, absent/malformed launch profile, and mixed
live/parked input. A composition test proves the production actionable provider
uses the same resolver as `ResumeContext`. Performance tests prove menu opening
and key handling do not wait on a blocked resolver. The resolver continues to
use exact-ledger, proof-named incremental validation; no new cache or duplicate
binding authority is introduced (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`,
`ARCH-CONSTRAINTS`).

Second, add a fail-safe spinner phase to `MenuState` and one `MenuEventTick`.
Dispatch tests first assert each operation installs its exact progress notice
before emitting its effect, while preview resolution retains `resolving…`.
Pure reducer/render tests assert ticks advance only a current progress notice,
the four one-cell frames keep width stable, stale completion cannot stop a
newer attempt, and terminal completion/error/cancel clears or replaces progress.
The Console run-loop owns one 100 ms timer channel, armed only while the current
menu has progress and stopped/drained otherwise; each tick re-enters the reducer
and repaints only when the switcher is focused. Fake-clock/host tests assert the
first banner is painted before a blocked dispatcher runs, repeated ticks cause
no provider/dispatcher calls, keyboard events still reduce during the block,
and teardown leaves no timer worker. Run focused and race tests with `-p 20`
(`ARCH-PURE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — close fresh-review gaps in proof and progress lifecycles

**Reason:** fresh review found the prior delta could strand teardown behind a
non-contextual resolver, let queued ticks animate replacement work, and did not
define contradictory parked-proof or diagnostic-retention oracles.

**Delta:** before the first RED test, change the planned resume observation to
`ParkedResumeObservation{Address, Agent, NativeID}`. Pure projection groups by
address and admits a parked record only for exactly one total observation that
matches its valid saved profile agent and has a non-empty ID. Table tests add
zero, duplicate-identical, duplicate-conflicting, wrong-address, wrong-agent,
empty-ID, live-plus-stale-park-proof, and mixed-row cases. A paired command/core
table feeds provisional, ambiguous, unbound, malformed/unreadable, and resolver
error outcomes through the provider: each row is absent from actionable output
while the unchanged record remains in `ThreadInventory` and the declared
`list/show` path.

Make `NativeBindingResolver.ResolveEstablished` accept `context.Context` and
add the context-bearing exact-query path used by production and fakes. Check
cancellation before/after each exact-query stage and between parked records;
return cancellation without publishing partial rows. Local filesystem calls
remain synchronous and no detached goroutine is allowed. Extend the existing
blocked-provider Console test with a resolver fake that waits on `ctx.Done()`;
Stop must cancel and join the sole refresh worker within 250 ms.

Represent progress identity explicitly as preview generation or operation
attempt plus spinner phase. Preserve typed preview text `resolving`; typed
operation texts are `starting thread`, `resuming <label>`, `parking <label>`,
`leaving couch`, `renaming <label>`, and `saving <label> description`; the
renderer alone adds the current one-cell frame, one space, and `…`. Add a pure
event matrix for exact/stale tick, exact/stale completion, ordinary navigation,
overlay changes, inventory refresh, resize, preview replacement/edit/Escape,
operation Escape/focus loss, errors, supported-size return, and teardown.
Ticks carry the identity captured when the timer was armed.

The Run-loop test uses a controllable timer seam: dispatch paints phase zero
before the blocked operation begins; a tick advances without invoking provider
or dispatcher; filter/navigation still reduce; stop/drain/rearm followed by an
old queued tick leaves replacement progress unchanged; focus loss pauses the
timer without clearing progress; exact completion stops it; Stop leaves no
timer or worker. Keep every test invocation at `-p 20` (`ARCH-PURE`,
`ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — make concurrent notice precedence executable

**Reason:** the follow-up fresh review found that “preserve progress on refresh”
did not decide what happens when refresh or another reducer path produces a
required error during pending work.

**Delta:** give `MenuNotice` an optional exact progress owner (preview
generation or operation attempt) and add pure cross-product traces before
implementation. Successful refresh, ordinary navigation, and unrelated info
preserve owned progress. Refresh failure, reconciliation, validation/no-
selection, and identity-exhaustion errors replace it and stop the timer; there
is no hidden progress restoration. A later exact success clears only progress
owned by its same identity and therefore preserves an unrelated replacement
error. A matching failure replaces the banner with its work-owned error. Stale
ticks/results preserve both banner and timer state; newly accepted work replaces
either with new phase-zero progress. Table-test every producer against preview
and operation progress, then run the focused reducer and Console race suites at
`-p 20` (`ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — record M3 delivery and target evidence

**Reason:** Task 12 removed the reachable compatibility UI and Task 13 landed
the performance harness, so the executable Core concepts table's earlier
“render migration pending” and “absent” entries became false.

**Delta:** mark the Console renderer and command composition current, and mark
`menu_perf_test.go` present. The harness measures the 100-row Console
input/repaint, refresh/repaint, pure-render, and progress paths across one
baseline and two four-worker co-tenancy trials on the M2 Max. No production
benchmark mode or additional runtime scheduler was added (`ARCH-CONSTRAINTS`,
`ARCH-PURPOSE`).

### 2026-08-31 — close M3 operation, authority, and evidence gaps

**Reason:** the first M3 boundary review found that production operation
completions did not supply the inventories used by reducer tests, the declared
deleted flat-panel authority still compiled beside the menu, and target timing
bypassed `Console.Run`.

**Delta:** every successful switcher operation now follows one exhaustive
projection policy. `start`, `park`, `resume`, `name`, and `describe` mark the
current projection visibly pending until a successful actionable-provider
refresh; refresh failure preserves last-good rows, the pending marker, and a
local error. `switch` changes only focus and `leave` has no subsequent frame.
An operation result carrying an inventory applies it immediately and clears the
marker. Pure and Console tests cover all seven operations, including the
resume early-return path and failed refresh (`ARCH-PURE`, `ARCH-PURPOSE`).

Delete `PanelModel`, `panel.go`, its tests, the resolver/summary callbacks,
prompt fields, and old Console controller. The #151 current-boundary contract
requires `panel.go` absent, while the #146 concept contract treats its
historical compatibility row as retired and scans production sources for
returning authority (`ARCH-DRY`, `ARCH-PURPOSE`).

Target evidence now drives a running Console with raw input, resize, and typed
refresh results and completes a sample only when `hostty.FakeHost` observes the
corresponding emitted frame. Open, filter, navigation, repaint, refresh, and
first feedback all use this lifecycle boundary across 100 rows, 20 warmups,
200 samples, and the baseline plus two exactly-four-worker co-tenancy trials
(`ARCH-MOCK`, `ARCH-CONSTRAINTS`).

### 2026-08-31 — authorize projection by post-mutation refresh generation

**Reason:** the second M3 boundary review found that a refresh admitted before
a successful mutation could complete afterward and clear the visible pending
marker while the required dirty follow-up was still running.

**Delta:** a committed mutating operation captures the refresh scheduler's
current sequence as `ProjectionAfterGeneration`. Successful inventory results
still update last-good rows, but clear `ProjectionPending` only when their exact
generation is greater than that captured boundary. Thus a pre-mutation result
cannot authorize the projection; a post-mutation success can, and a
post-mutation failure keeps both the last-good rows and visible pending state.
The Console regression covers the full sequence: generation 1 running,
mutation success, dirty request, generation 1 success, generation 2 blocked,
then generation 2 failure. This rule applies to all five mutating operations
through the shared operation policy (`ARCH-PURE`, `ARCH-PURPOSE`).

The authoritative M3 checklist was swept against committed implementation and
evidence. Tasks 10, 11, 12, and Task 13 Steps 1–7 are checked; only the M3
boundary close and subsequent issue close remain open.

### 2026-08-31 — align boundary documentation with executable M3 semantics

**Reason:** the third M3 review found that final Core-concept rows retained
intermediate Task 10/11 column headings, README claimed root Escape exits when
no live actor exists, and the corrected M3 checklist had no executable guard.

**Delta:** both Core-concept tables now label their current-state column
`Current at M3 boundary`. README matches the reducer and Spec: root Escape
clears a filter or returns to an attached live actor; with none, the switcher
stays open and reports why. The document contract executes that reducer case
and requires the matching operator sentence. A separate plan contract parses
all 57 checklist steps across Tasks 10–13, requires Tasks 10–12 and Task 13
Steps 1–7 checked, and requires only Task 13 boundary/issue close open; a
delivered-as-unchecked mutation must fail (`ARCH-PURPOSE`).

### 2026-08-31 — derive the complete M3 architectural inventory

**Reason:** the fourth M3 review found that the final Core-concept inventory
omitted `ParkedResumeObservation` and its context-bearing native-binding/session-
inventory dependency, still described projection as live-observation-only, and
used one non-resolvable `existing test seams` location.

**Delta:** the projection row now includes live and parked proof inputs with
their current exact semantics. A new integration row records
`NativeBindingResolver` / `SessionInventoryNativeBindingResolver` and the exact
`cmd/internal/sessioninventory/query.go` dependency. Every Core-concept row now
uses complete repo-relative paths, including runner/fake/host seams. Six
delivered M3 architectural declarations carry source markers; the contract
discovers those markers, resolves each Go declaration, and requires its exact
source path in a Core-concept row. The complete table contract also compares
every row's path set exactly. Mutations removing a marked entity or the session-
inventory dependency path must fail (`ARCH-PURPOSE`, `ARCH-MOCK`).

### 2026-08-31 — close the complete M3 declaration set

**Reason:** the fifth M3 boundary review showed that concept markers alone were
opt-in: adding an unmarked declaration did not alter the six selected concepts,
so the documentation contract could not prove that its architectural inventory
was exhaustive (`BR-26`, `documentation-current-state-accuracy`).

**Delta:** the contract now derives every top-level declaration from the exact
net set of Go sources changed since the M2 boundary. The checked-in digest gives
each declaration one closed-set disposition: a `pair:m3-concept` marker is
architectural, every remaining declaration is implementation detail, and the
deleted flat-panel files are retired. The source catalog must exactly equal the
Git milestone diff, so a new file cannot evade classification. A mutation that
adds an unmarked exported `ReviewAddedM3Authority` changes the declaration
digest and fails before the Core concepts rows are accepted. Architectural
markers still require an exact entity and repo-relative path in the plan
(`ARCH-PURPOSE`).

### 2026-08-31 — pin and classify the M3 declaration oracle

**Reason:** the sixth M3 boundary review found that the declaration digest
covered membership but omitted the architectural/detail classification, so
removing a concept marker did not change it. It also read the current worktree
rather than a stable historical M3 snapshot (`BR-26`, `BR-27`).

**Delta:** M3 paths are derived only from `0c40a8d1..7ff7d8c4`, and every parsed
byte comes from the pinned `7ff7d8c4` Git object. Each declaration digest key
now includes `architectural` or `detail`; deleted panel files carry `retired`.
An exact name/path ledger is compared bidirectionally with all six architectural
markers, and those entries must each appear at their exact Core concepts path.
Mutation tests reject marker removal, architectural-to-detail and detail-to-
architectural reclassification, and an added unmarked declaration
(`ARCH-PURPOSE`).

### 2026-08-31 — make every Core concept one classified declaration

**Reason:** the seventh M3 boundary review showed that the six-entry marker
ledger was still a subset of the architectural table: for example,
`RefreshSchedule` and `AdvanceRefreshSchedule` were documented authorities but
were classified as detail. It also clarified that the immutable snapshot must
be separated from the commit that installs its oracle (`BR-26`, `BR-27`).

**Delta:** both Core concepts tables now contain exactly one declaration per
row. One typed ledger owns each declaration's Pure/Integration kind, delivery,
current status, source declaration path, complete dependency paths, and retired
paths. It covers all 37 current or retired declarations; plan parsing and pinned
source resolution both derive from it, and every row must occur exactly once.
The historical declaration digest uses this same complete ledger for
architectural/detail/retired classification rather than source markers, which
have been removed as a parallel authority. Classification and membership
mutations fail closed.

The runtime-plus-ledger snapshot is pinned at reviewed commit `d3ee08d5`. The
following oracle-installation commit changes only tests and documentation and
validates that immutable snapshot, avoiding a self-referential commit hash.
Future source changes therefore do not rewrite M3 history (`ARCH-DRY`,
`ARCH-PURPOSE`).

### 2026-08-31 — close the Integration dependency inventory

**Reason:** the eighth M3 boundary review found that one-entity rows still
listed declaration paths without separately enforcing the dependencies named by
their Integration contracts (`BR-26`).

**Delta:** an architectural dependency is a repo-owned seam directly named by
an Integration entity's signature/body or its `Wraps` contract; standard-library
types and private same-file helpers are implementation detail. Every Integration
entry was swept under that rule. Its exact path union now includes the declaring
source plus all such dependencies—for example `ThreadStore.Snapshot` and
`NativeBindingResolver` for actionable inventory, pure menu/scheduler/host seams
for Console, and runner/artifact/thread-store seams for abort cleanup. The typed
ledger, not plan prose, owns those unions. A separate stable digest covers name,
kind, delivery, current state, declaration source, dependencies, and retirement;
a mutation removing an enumerated dependency fails independently of the plan
(`ARCH-PURPOSE`).

### 2026-08-31 — derive Integration dependencies from pinned implementation

**Reason:** the ninth M3 boundary review disproved the closed ledger again:
`Couch.ActionableThreadInventoryContext` directly uses the `PathOps` seam, but
`pathops.go` was absent from the hand-maintained dependency set (`BR-26`).

**Delta:** the dependency contract now parses pinned Integration declarations,
their architectural method families, signatures, bodies, receiver fields, and
repo-owned selector/type references. It resolves those references through a
repository declaration index and requires the resulting exact path set to equal
each ledger entry. Every Integration row was regenerated from that mechanical
result. The actionable-inventory family now includes `PathOps` together with
thread/artifact/binding/profile dependencies. A mutation removes the pinned
`c.Path.Physical` branch and proves the derived set no longer matches the ledger;
this catches implementation-side omission rather than only edits to known
ledger entries (`ARCH-DRY`, `ARCH-PURPOSE`).
