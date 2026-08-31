# Hierarchical Work-Thread Menu Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Couch's transitional flat panel with a non-blocking hierarchical switcher over verified live and verified parked TTY work threads.

**Architecture:** A pure Couch-core projection joins durable thread records with exact live-owner TTY observations and emits only actionable rows. A pure menu reducer owns frames, filtering, selection, forms, reconciliation, and bounded rendering; `couchtty.Console` remains the thin event/TTY/operation shell. Start preview and launch share an opaque resolution token, while inventory and preview I/O run through bounded single-flight/coalescing controllers.

**Tech Stack:** Go, Couch's existing `couchcore` operation table and stores, `couchtty` terminal emulator/console, stateful `FakeRunner`/`FakeHost` integration doubles, Go unit tests/benchmarks.

---

## Core concepts

### Pure entities

| Name | Lives in | Planned change | Delivery | Current at M2 |
|------|----------|----------------|----------|---------------|
| `ActionableThreadSummary` / `LiveTTYObservation` / `ProjectActionableThreads` | `cmd/internal/couchcore/actionableinventory.go` | new | M1 | present |
| `StartResolution` / `StartResolutionFingerprint` / `ResolveStartResolution` | `cmd/internal/couchcore/startresolution.go` | new | M1 | present |
| `MenuState` / `MenuFrame` / `MenuEvent` / `ReduceMenu` | `cmd/internal/couchtty/menu.go` | new | M2 | present |
| `MenuLayout` / `AgeBand` / `RenderMenu` | `cmd/internal/couchtty/menu_render.go` | new | M2 | present |
| `PreviewSchedule` / `AdvancePreviewSchedule` | `cmd/internal/couchtty/menu_async.go` | new | M2 | present |
| `RefreshSchedule` / `AdvanceRefreshSchedule` | `cmd/internal/couchtty/menu_refresh.go` | new | M3 | absent |
| `PanelKey` / `DecodePanelKeys` | `cmd/internal/couchtty/panelkeys.go` | modified | M2 | modified, present |
| `PanelModel` / resolver-driven `Filter` | `cmd/internal/couchtty/panel.go` | deleted | M3 | present compatibility adapter |

- **`ActionableThreadSummary` / `LiveTTYObservation` / `ProjectActionableThreads`** — the only interpretation that turns internal thread lifecycle into user-facing `live` or `parked` rows.
  - **Relationships:** N durable `ThreadRecord`s and N exact owner observations produce 0..N `ActionableThreadSummary` rows; each row retains one composite `ThreadAddress`. A live row requires one observation matching one durable incarnation's PID and process identity; a parked row requires exact `VerifiedPark`, no active `Park`, and no occupied incarnation.
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

| Name | Lives in | Planned change | Delivery | Current at M2 | Wraps |
|------|----------|----------------|----------|---------------|-------|
| `Couch.ActionableThreadInventory` | `cmd/internal/couchcore/actionableinventory.go` | new | M1 | present | `ThreadStore.Snapshot` plus live-owner observations |
| `Couch.PrepareStart` / `Couch.SpawnPrepared` | `cmd/internal/couchcore/startresolution.go`, `couch.go` | new | M1 | present | path, policy, preference/default reads, runner launch |
| `StartGrantStore` | `cmd/internal/couchcore/startgrant.go` | new | M1 | present | owner-local random issuance, TTL, and atomic consumption |
| context-bearing shared operations and post-start cleanup | `cmd/internal/couchcore/ops.go`, `operationdispatch.go`, `couch.go` | modified | M1 | modified, present | owner operation dispatch, cancellation, exact-handle cleanup |
| `Console` menu controller | `cmd/internal/couchtty/console_menu.go`, `console.go` | modified | M3 | `console_menu.go` absent; flat Console present | host input/output, pane observations, bounded async workers |
| `wireResolver` composition | `cmd/internal/couchcmd/run.go` | modified | M3 | M1 start authority present; menu wiring pending | Couch core providers and Console operation dispatcher |
| context-bearing `Runner` / `FakeRunner` / `hostty.FakeHost` | existing test seams | modified | M1 | modified, present | cancelable child lifecycle and terminal behavior |
| target performance harness | `cmd/internal/couchtty/menu_perf_test.go` | new | M3 | absent | clock samples and deterministic four-worker CPU load |

- **`Couch.ActionableThreadInventory`** — snapshots durable records, then calls the pure projector with caller-supplied exact observations.
  - **Injected into:** Console refresh worker; Console alone derives observations from its registered panes and child identities.
  - **Future extensions:** #147 may inject remote owner observations with the same identity shape.

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

- [ ] **Step 1: Write failing pure refresh-schedule traces**

Apply the `AdvanceRefreshSchedule` reordered-completion model strategy and assert
the one-running/one-dirty bound.

- [ ] **Step 2: Run scheduler tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestAdvanceRefreshSchedule' -count=1`

Expected: FAIL because `RefreshSchedule` is absent.

- [ ] **Step 3: Implement the pure refresh schedule**

Return effects rather than spawning goroutines.

- [ ] **Step 4: Run scheduler tests and verify GREEN**

Re-run Step 2. Expected: PASS.

- [ ] **Step 5: Write failing observation/wiring tests**

Apply the Console refresh-controller strategy with exact pane identities and a
context-bearing actionable provider. Barriers prove opening/input use current
memory while blocked provider IO is cancelable and joined.

- [ ] **Step 6: Run observation/wiring tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(Console.*Observation|Wire.*Actionable|MenuOpenDoesNotWait)' -count=1`

Expected: FAIL because pane identity and async actionable wiring are absent.

- [ ] **Step 7: Implement the bounded inventory worker**

Wire one context-bound worker around `RefreshSchedule`; snapshot observations
under the Console mutex, perform provider IO outside it, and feed terminal
results only through `ReduceMenu`. Teardown cancels then joins.

- [ ] **Step 8: Run focused and complete TTY/command tests**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`

Expected: PASS; blocked inventory tests prove opening/filtering/navigation do not wait or call the provider.

- [ ] **Step 9: Commit Task 10**

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

- [ ] **Step 1: Write failing result-carrying operation queue tests**

Apply the Console action-controller strategy to every declared queued operation;
the model oracle guards typed results, duplicate coalescing, overload restoration,
single completion, and non-redispatch.

- [ ] **Step 2: Run queue tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(OperationQueue|MenuOperationCompletion)' -count=1`

Expected: FAIL because completions currently discard results.

- [ ] **Step 3: Implement result-carrying action execution**

Carry typed results through the existing bounded sequential queue. Accepted work
enters progress, overload reduces immediate failure, duplicates retain the
original progress, and returned lifecycle values re-enter the shared projector.

- [ ] **Step 4: Run queue tests and verify GREEN**

Re-run Step 2. Expected: PASS, including the `park_latency_test.go` callback signature migration.

- [ ] **Step 5: Write failing remaining lifecycle-context tests**

`OperationCall.Context`, `PrepareStart`, and `SpawnPrepared` landed in M1.
Apply the cancellation strategy to the remaining resume, Pair lifecycle,
leave, and runner consumers; executor barriers guard that supplied deadlines
reach each seam while a nil CLI context still means background.

- [ ] **Step 6: Run operation-dispatch tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(DispatchOperationContext|PrepareStartCancellation|LifecycleOperationContext|ContextlessCLI)' -count=1`

Expected: existing dispatch/prepare context regressions PASS and the new
remaining lifecycle consumers FAIL because they still substitute background.

- [ ] **Step 7: Implement operation/lifecycle context propagation**

Thread the already-carried operation context through resume admission, Pair
lifecycle calls, leave, and their operation wiring.

- [ ] **Step 8: Run operation-dispatch tests and verify GREEN**

Re-run Step 6. Expected: PASS.

- [ ] **Step 9: Write failing blocked-runner cancellation tests**

Add `TestBlockedRunnerCancellationConformance` using the shared fake/Exec/Pty
trace defined above, plus contract/compile checks for every `StartBlocked`
consumer. Registration timeout derives from the operation context.

- [ ] **Step 10: Run blocked-runner tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/probes/couchstartrecovery -run 'Test(RunnerCancellation|LaunchHelperCancellation|TrackedLaunchCancellation|StartTransaction|Issue149BlockedRunnersDelegateToOneHandshakeAuthority)' -count=1`

Expected: FAIL at the contextless signature/behavior.

- [ ] **Step 11: Implement blocked-runner cancellation**

Thread context through the runner/launch-helper seam and all consumers. Record
the fake transitions named in the shared conformance trace; reuse existing
exact post-ack cleanup.

- [ ] **Step 12: Run blocked-runner tests and verify GREEN**

Re-run Step 10. Expected: PASS with every helper either transferred successfully or exactly canceled/reaped/reconciled.

- [ ] **Step 13: Write failing Console action-lifetime tests**

Apply the Console action-controller teardown strategy to blocking lifecycle and
metadata dispatchers; cancellation observation, one completion, and a 250 ms
join bound are the guards.

- [ ] **Step 14: Run Console action-lifetime tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestConsoleStopCancelsOperation' -count=1`

Expected: FAIL because accepted actions do not share one Console lifetime context.

- [ ] **Step 15: Bind accepted actions to Console lifetime**

Create one Console lifetime context, bind every accepted action call to a child, and cancel it before joining the action worker.

- [ ] **Step 16: Run Console action-lifetime tests and verify GREEN**

Re-run Step 14. Expected: PASS with one Console context owning every accepted action and worker join.

- [ ] **Step 17: Write failing preview-worker cancellation tests**

Join the pure preview-schedule model to a blocking stateful dispatcher; generation
identity, one terminal completion, and cancel/join within 250 ms are the guards.

- [ ] **Step 18: Run preview tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestConsole(StartPreview|PreviewCancellation|PendingSubmit)' -count=1`

Expected: FAIL because Console does not execute preview effects.

- [ ] **Step 19: Implement preview/start effects**

Map preview/start schedule effects to context-bound declared operations and feed
typed results back through reducer/projector authority.

- [ ] **Step 20: Run preview tests and verify GREEN**

Re-run Step 18. Expected: PASS.

- [ ] **Step 21: Write failing exact-handle abort tests**

Apply the exact-handle cleanup strategy to `Couch.AbortStarted`; identity
validation and completed quiesce/reconciliation are the guards.

- [ ] **Step 22: Run exact-handle abort tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestAbortStarted' -count=1`

Expected: FAIL because the owner method is absent.

- [ ] **Step 23: Implement exact-handle abort**

Add the narrow owner method and reuse the existing cleanup authority without duplicating signals/reconciliation.

- [ ] **Step 24: Run exact-handle abort tests and verify GREEN**

Re-run Step 22. Expected: PASS.

- [ ] **Step 25: Write failing transactional pane-attach/Stop tests**

Apply the partial-install Console strategy to transactional attach and Stop;
exact identity removal, watcher join, routing restoration, terminal restoration,
and `AbortStarted` completion are the guards.

- [ ] **Step 26: Run transactional attach tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(ConsoleAttachRollback|ConsoleStopDuringAttach|WireAttachAbort)' -count=1`

Expected: FAIL because attach has no transactional rollback/partial-Stop contract.

- [ ] **Step 27: Implement transactional attach and partial-Stop cleanup**

Make attach commit/rollback one owner-local composition in `wireResolver`; cleanup finishes before reducer restoration.

- [ ] **Step 28: Run transactional attach tests and verify GREEN**

Re-run Step 26. Expected: PASS.

- [ ] **Step 29: Write semantic action/form restoration traces**

Drive Spec-defined operation outcomes through semantic events and the shared
Console/reducer model oracle. Exact dispatch identity, restoration only after
cleanup, frame reconciliation, and no redispatch are the guards.

- [ ] **Step 30: Run restoration traces and verify RED**

Run:

`go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(DispatchOperationContext|PrepareStartCancellation|AbortStarted|Console(Menu|StartPreview|PreviewCancellation|PendingSubmit|Operation|Park|Resume|Bell|Refresh))' -count=1`

Expected: FAIL because the controller does not yet map every restoration case.

- [ ] **Step 31: Implement remaining effect/restoration mappings**

Complete the thin mapping from reducer effects to declared operations, reusing
existing Alt+x and clear/replay ownership.

- [ ] **Step 32: Run restoration traces and verify GREEN**

Re-run Step 30. Expected: PASS.

- [ ] **Step 33: Commit Task 11**

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

- [ ] **Step 1: Write failing real-loop/render replacement tests**

Drive the Spec's controls through raw `hostty.FakeHost` input and reuse the
decoder/reducer/controller strategies above. Generation-tagged repaint and host
mode snapshots guard input routing, resize, screen ownership, and teardown.
Add README expectation tests before editing prose.

- [ ] **Step 2: Run replacement tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(ConsoleRun.*Menu|MenuControls|Readme.*Couch)' -count=1`

Expected: FAIL while Console still renders `PanelModel`.

- [ ] **Step 3: Route all menu input/paint through reducer/renderer**

Make `MenuState`/`ReduceMenu`/`RenderMenu` the sole Console menu path and delete
the compatibility panel after references migrate. Update the concepts contract
to treat #151's table as the superseding status inventory without weakening
pure-source/direct-test enforcement.

- [ ] **Step 4: Update user controls and migrate regressions**

Document the hierarchical menu and remove flat-panel wording in README. Move still-valid stable-selection, mouse, Escape, switch/replay, bell, park/resume, and terminal restoration assertions onto the new public behavior; delete tests only when their old symbol is deleted and their behavior is covered elsewhere.

- [ ] **Step 5: Run complete changed packages**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`

Expected: PASS with no `PanelModel`, synchronous resolver-per-key path, or old prompt state remaining.

- [ ] **Step 6: Commit Task 12**

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

- [ ] **Step 1: Write the portable 100-row benchmark/bound tests**

Implement the Spec's fixed 100-row fixture and `BenchmarkMenu100`. Enforce the
declared allocation, worker/queue, input, and minimum-size bounds mechanically;
portable tests do not assert target-specific wall time.

- [ ] **Step 2: Run portable tests and verify GREEN**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestMenu(100|Bounds|Feedback)' -count=1`

Expected: PASS with every allocation ceiling enforced; `go test -p 1 ./cmd/internal/couchtty -run '^$' -bench '^BenchmarkMenu100$' -benchmem -count=5` records allocation/ns-op evidence.

- [ ] **Step 3: Implement and run the opt-in M2 Max protocol**

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

- [ ] **Step 4: Run full automated verification with bounded parallelism**

Run: `go test -p 20 ./... -count=1`

Expected: PASS. Then run `git diff --check`; expected: no output.

- [ ] **Step 5: Build and perform a clean-store live smoke**

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

- [ ] **Step 6: Update atlas and issue evidence**

Update `atlas/couch.md` from “pure core planned integration” to current reachable hierarchical switcher, actionable/raw boundaries, async worker bounds, operation mapping, performance envelope, and suspend/resume-only application lifecycle. Confirm `atlas/index.md` still links it. Append target measurements/live-smoke evidence to the issue Log and tick implementation tasks only through SDLC gates.

- [ ] **Step 7: Commit performance/docs evidence**

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
