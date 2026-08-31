# Hierarchical Work-Thread Menu Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Couch's transitional flat panel with a non-blocking hierarchical switcher over verified live and verified parked TTY work threads.

**Architecture:** A pure Couch-core projection joins durable thread records with exact live-owner TTY observations and emits only actionable rows. A pure menu reducer owns frames, filtering, selection, forms, reconciliation, and bounded rendering; `couchtty.Console` remains the thin event/TTY/operation shell. Start preview and launch share an opaque resolution token, while inventory and preview I/O run through bounded single-flight/coalescing controllers.

**Tech Stack:** Go, Couch's existing `couchcore` operation table and stores, `couchtty` terminal emulator/console, stateful `FakeRunner`/`FakeHost` integration doubles, Go unit tests/benchmarks.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ActionableThreadSummary` / `LiveTTYObservation` / `ProjectActionableThreads` | `cmd/internal/couchcore/actionableinventory.go` | new |
| `StartResolution` / `StartResolutionFingerprint` / `ResolveStartResolution` | `cmd/internal/couchcore/startresolution.go` | new |
| `MenuState` / `MenuFrame` / `MenuEvent` / `ReduceMenu` | `cmd/internal/couchtty/menu.go` | new |
| `MenuLayout` / `AgeBand` / `RenderMenu` | `cmd/internal/couchtty/menu_render.go` | new |
| `PreviewSchedule` / `AdvancePreviewSchedule` | `cmd/internal/couchtty/menu_async.go` | new |
| `RefreshSchedule` / `AdvanceRefreshSchedule` | `cmd/internal/couchtty/menu_refresh.go` | new |
| `PanelKey` / `DecodePanelKeys` | `cmd/internal/couchtty/panelkeys.go` | modified |
| `PanelModel` / resolver-driven `Filter` | `cmd/internal/couchtty/panel.go` | deleted |

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
  - **Relationships:** one start form owns one schedule; edits replace its single pending request, and a pending submit is bound to exactly one generation.
  - **DRY rationale:** cancellation, stale completion, and armed-submit rules are tested once rather than encoded across goroutine branches.
  - **Future extensions:** none planned; this is deliberately not a generic job scheduler.

- **`RefreshSchedule` / `AdvanceRefreshSchedule`** — pure one-running/one-dirty-bit inventory refresh state.
  - **Relationships:** one Console owns one schedule; any number of attach, exit, and operation events coalesce into at most one follow-up refresh after the running generation terminates.
  - **DRY rationale:** initial-unavailable, last-good, stale completion, and dirty follow-up behavior use one transition authority rather than goroutine-local flags.
  - **Future extensions:** a remote owner may change the provider latency, but not the single-flight contract.

- **`PanelKey` / `DecodePanelKeys`** — existing framed terminal decoder widened with Tab in both legacy HT and Kitty CSI-u encodings.
  - **Relationships:** raw bytes map to semantic keys before the reducer sees them.
  - **DRY rationale:** all menu frame kinds consume one decoded Tab event.
  - **Future extensions:** add a semantic key only when a menu contract requires it.

- **Deleted `PanelModel` / resolver-driven `Filter`** — replaced by `MenuState`; ordinary filter keystrokes match the in-memory actionable rows and never call the durable resolver.
  - **Relationships:** existing compatibility tests move to `menu_test.go`; `ResolveThreadReference` remains for CLI/operation ref lookup, not keystroke filtering.
  - **DRY rationale:** removes the old competing flat panel state and the synchronous resolver I/O hot path.
  - **Future extensions:** N/A.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Couch.ActionableThreadInventory` | `cmd/internal/couchcore/actionableinventory.go` | new | `ThreadStore.Snapshot` plus live-owner observations |
| `Couch.PrepareStart` / `Couch.SpawnPrepared` | `cmd/internal/couchcore/startresolution.go`, `couch.go` | new | path, policy, preference/default reads, runner launch |
| `StartGrantStore` | `cmd/internal/couchcore/startgrant.go` | new | owner-local random issuance, TTL, and atomic consumption |
| context-bearing shared operations and post-start cleanup | `cmd/internal/couchcore/ops.go`, `operationdispatch.go`, `couch.go` | modified | owner operation dispatch, cancellation, exact-handle cleanup |
| `Console` menu controller | `cmd/internal/couchtty/console_menu.go`, `console.go` | modified | host input/output, pane observations, bounded async workers |
| `wireResolver` composition | `cmd/internal/couchcmd/run.go` | modified | Couch core providers and Console operation dispatcher |
| context-bearing `Runner` / `FakeRunner` / `hostty.FakeHost` | existing test seams | modified | cancelable child lifecycle and terminal behavior |
| target performance harness | `cmd/internal/couchtty/menu_perf_test.go` | new | clock samples and deterministic four-worker CPU load |

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

- **Context-bearing runner and existing stateful doubles** — the blocked-start runner seam accepts the operation context so a canceled Console cannot strand a helper launch or registration wait. `FakeRunner` retains fork/registration/exit state and `hostty.FakeHost` retains terminal bytes/modes; no new external binary is introduced. Pair/Zellij suspend/resume continues through `PairLifecycleController` and its existing fake/live conformance suite (ARCH-MOCK).
  - **Injected into:** full-console tests and current park/resume integration tests.
  - **Future extensions:** add modeled calls only when #151 consumes a new lifecycle behavior.

- **Target performance harness** — measures committed 100-row scripted interactions on M2 Max and optionally starts exactly four bounded CPU workers over fixed in-memory buffers.
  - **Injected into:** test-only measurement entrypoint; production contains no benchmark flags or load generator.
  - **Future extensions:** retain fixtures for regression; do not generalize into a benchmarking framework.

## Chunk 1: Authoritative actionable inventory and token-bound start

### Task 1: Project durable records plus exact owner observations

**Files:**
- Create: `cmd/internal/couchcore/actionableinventory.go`
- Create: `cmd/internal/couchcore/actionableinventory_test.go`
- Modify: `cmd/internal/couchcore/threadinventory.go`
- Modify: `cmd/internal/couchcore/threadinventory_test.go`

- [ ] **Step 1: Write failing pure projection tests**

Add table tests constructing `ThreadRecord`s for: matching observed live incarnation, persisted-live without observation, mismatched PID/identity observation, exact verified park, active park transaction, creating/unknown incarnation, abandoned/tombstoned history, corrupt simultaneous live+verified park, and two threads at the same path. Assert only the matching live and exact verified-park rows survive, retain composite identity, carry `LastActiveAt`, and sort deterministically.

```go
got := ProjectActionableThreads(records, []LiveTTYObservation{{
    Address: live.Address,
    Process: ProcessIdentity{PID: 42, Identity: "pair-live"},
}})
if diff := cmp.Diff([]ActionableThreadSummary{
    {Address: live.Address, State: ThreadLive, LastActiveAt: live.LastActiveAt},
    {Address: parked.Address, State: ThreadParked, LastActiveAt: parked.LastActiveAt},
}, got, summaryCmpOptions...); diff != "" {
    t.Fatalf("actionable inventory mismatch (-want +got):\n%s", diff)
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestProjectActionableThreads' -count=1`

Expected: FAIL because `LiveTTYObservation`, user-facing state, and projector do not exist.

- [ ] **Step 3: Implement the pure projection**

Define fail-closed zero-valued state and defensive copies:

```go
type ActionableThreadState uint8
const (
    ThreadActionUnknown ActionableThreadState = iota
    ThreadLive
    ThreadParked
)

type LiveTTYObservation struct {
    Address ThreadAddress
    Process ProcessIdentity
}

type ActionableThreadSummary struct {
    Address ThreadAddress
    StartingPath, WorkingPath, Name, Description, PublishedSummary string
    State ActionableThreadState
    LastActiveAt time.Time
}

func ProjectActionableThreads(records []ThreadRecord, observations []LiveTTYObservation) []ActionableThreadSummary
```

Give `ActionableThreadSummary` its own `Live()`, `Label()`, and `DisplaySummary()` methods. Do not modify `ThreadSummary.Live()`: raw `BuildThreadInventory` continues to report persisted diagnostic incarnation state for `couch list/show`. Rename comments and add a cross-type regression proving the raw row remains visible/diagnostic while the same unobserved record is absent from the actionable projection.

- [ ] **Step 4: Write failing snapshot-wrapper tests**

Test caller mutation cannot alter records/observations/results, a failed snapshot
is returned as an error rather than an empty list, and raw diagnostic
`ThreadInventory()` still retains hidden records for `couch list/show`.

- [ ] **Step 5: Run wrapper tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestActionableThreadInventory' -count=1`

Expected: FAIL because the wrapper is absent.

- [ ] **Step 6: Implement the thin snapshot wrapper**

Implement:

```go
func (c *Couch) ActionableThreadInventory(obs []LiveTTYObservation) ([]ActionableThreadSummary, error) {
    snapshot, err := c.Threads.Snapshot()
    if err != nil { return nil, err }
    return ProjectActionableThreads(snapshot.Records, obs), nil
}
```

- [ ] **Step 7: Run focused tests and verify GREEN**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'Test(ProjectActionableThreads|ActionableThreadInventory|BuildThreadInventory)' -count=1`

Expected: PASS.

- [ ] **Step 8: Commit Task 1**

```bash
git add cmd/internal/couchcore/actionableinventory.go cmd/internal/couchcore/actionableinventory_test.go cmd/internal/couchcore/threadinventory.go cmd/internal/couchcore/threadinventory_test.go
git commit -m "#151 M1: project actionable Couch threads"
```

### Task 2: Issue bounded one-attempt start grants

**Files:**
- Create: `cmd/internal/couchcore/startgrant.go`
- Create: `cmd/internal/couchcore/startgrant_test.go`
- Modify: `cmd/internal/couchcore/couch.go`

- [ ] **Step 1: Write failing grant authority tests**

Define `StartGrantToken` and `StartGrant`. Require exactly 32 bytes (256 bits) from injected cryptographic entropy, encoded with `base64.RawURLEncoding`. Test `io.ReadFull` short-read/error issuance leaves the store unchanged; token collision permits three total complete 32-byte draws (initial plus two retries), then refuses without overwriting the existing grant.

- [ ] **Step 2: Write failing lifecycle/bound tests**

Test a mutex-owned `issued → consuming → removed` lifecycle. Issued plus consuming entries count toward a hard capacity of 16. Issuance first prunes expired issued entries, but never evicts consuming authority; sixteen consuming grants make the seventeenth issuance fail. TTL is five minutes and checked atomically before claim (`now >= ExpiresAt` rejects/removes); once claimed, expiry does not abort the in-flight attempt. Every attempt outcome removes the consuming entry, and replay returns unknown/consumed. A new store rejects all pre-restart tokens.

- [ ] **Step 3: Run tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestStartGrant' -count=1`

Expected: FAIL because `StartGrantStore` does not exist.

- [ ] **Step 4: Implement minimal grant storage**

Use one mutex for the map, lifecycle, capacity, and collision checks. `Issue` clones `StartResolution`; `Claim` returns one cloned resolution and marks it consuming; `Finish` requires the consuming token and deletes it. Add the store to `Couch` with existing `Clock`/`Entropy` dependencies. Do not persist grants or reissue a token after failure.

- [ ] **Step 5: Run focused and package tests**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestStartGrant' -count=1`

Expected: PASS, including concurrent claims and capacity while consuming.

- [ ] **Step 6: Commit Task 2**

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

- [ ] **Step 1: Write failing pure resolution/fingerprint tests**

Cover canonical path, agent precedence, same-agent path argv, repository-default argv, full normalized `PolicyResult`, preference revision, semantic repository-default digest, defensive copies, deterministic fingerprint equality, and a fingerprint change for every field/argv element. Add negative cases for unsupported agents, absent source, wrong-agent defaults, malformed args, and invalid policy evidence.

- [ ] **Step 2: Run pure tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestResolveStartResolution' -count=1`

Expected: FAIL because the resolution entity is absent.

- [ ] **Step 3: Implement the pure resolution**

Define `StartResolution` with canonical path, worktree, full policy, launch profile/sources, preference revision, default digest, and `StartResolutionFingerprint`. Reuse `ResolveLaunchProfile`; hash an explicit schema plus length-delimited normalized fields. Re-run Step 2 and expect PASS.

- [ ] **Step 4: Write failing prepared-I/O and admission tests**

Assert `PrepareStart` performs no allocation/fork, resolves candidate policy once, reads preference/default evidence, and issues one grant. Assert `SpawnPrepared` claims once, reruns that identical resolution pipeline once, compares fingerprints, and uses those returned values. Mutating any path/agent/policy/default/preference evidence refuses before allocation/fork and consumes the grant. Add mismatch tests proving call args cannot override the grant resolution.

Add `ReconcileAdmissionPrepared(ctx, ..., candidatePolicy)` tests proving it never resolves the candidate but retains current snapshot retries, incumbent-policy refresh, capacity decisions, and rollback. Verify the factored spawn path never re-canonicalizes or re-reads policy/default/preference after authorization, and only successful Pair registration updates path preference.

- [ ] **Step 5: Run prepared-start tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'Test(PrepareStart|SpawnPrepared|ReconcileAdmissionPrepared)' -count=1`

Expected: FAIL because the prepared I/O/admission seam is absent.

- [ ] **Step 6: Implement prepared start and candidate-policy admission**

`PrepareStart` canonicalizes path/tree, resolves/validates candidate policy, keys preference from that repo identity/canonical path, reads the selected agent default, computes its semantic digest, calls the pure resolver, and issues a grant. `SpawnPrepared` atomically claims, recomputes without issuing, compares fingerprints, and always defers `Finish`; stale/failure requires a new preview. Pass the recomputed resolution directly into allocation, `ReconcileAdmissionPrepared`, and launch. Never re-resolve candidate policy/profile/path after the comparison; incumbent policy reconciliation remains unchanged.

- [ ] **Step 7: Run focused core tests**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'Test(ResolveStartResolution|PrepareStart|SpawnPrepared|ReconcileAdmissionPrepared|StartGrant)' -count=1`

Expected: PASS.

- [ ] **Step 8: Commit Task 3**

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

- [ ] **Step 1: Write failing operation/CLI contract tests**

Require `ResultStartResolution` and a new nonzero `EffectAuthority`; `prepare-start` as `ExecuteLiveOwner`/`EffectAuthority`; and `start` with required implicit `resolution-token`. Pin validation, defensive result copies, and absence of a duplicate `rename` operation. Require `operationOwnsLive("prepare-start")`, while a new `operationStartsConsole`/`WantsConsole` decision remains true only for `start`/`resume`; `consoleRunnerFor("prepare-start", ...)` must return no Console and an `ExecRunner` even on a terminal.

Require direct `couch prepare-start` rendering and normal `couch start` internally dispatching preview then token-bound start without accepting a token from argv. Pin owner-routing refusal when the live-owner capability is absent.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(Operation|WantsConsole|ConsoleRunner|RunWithRuntime|RenderStartResolution)' -count=1`

Expected: FAIL because the declarations, result, and two-step CLI path are absent.

- [ ] **Step 3: Implement operation and CLI wiring**

Add `EffectAuthority`, declare/dispatch `prepare-start`, and make token-bound `start` the sole production spawn route. Let direct preview acquire the singleton lease but never a PTY/Console. `RunWithRuntime` keeps public `couch start [path] --agent=...`; after constructing the owner it invokes `prepare-start`, injects the token as trusted implicit data, and dispatches `start`. Render direct preview fields/token. Console/advisor clients must explicitly make the same two calls.

- [ ] **Step 4: Run unfiltered changed-package tests**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`

Expected: PASS, including grant, admission, raw/actionable inventory, operation, and Console-routing tests.

- [ ] **Step 5: Commit Task 4**

```bash
git add cmd/internal/couchcore/ops.go cmd/internal/couchcore/ops_declarations_test.go cmd/internal/couchcore/operationdispatch.go cmd/internal/couchcore/operationdispatch_test.go cmd/internal/couchcore/plan_contract_test.go cmd/internal/couchcmd/run.go cmd/internal/couchcmd/run_test.go
git commit -m "#151 M1: route token-bound start operations"
```

- [ ] **Step 6: Update the Couch atlas for M1**

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

- [ ] **Step 1: Write failing shared matcher tests**

Extract the existing pure matching rule from `ResolveThreadReference` into `ClassifyThreadReferenceFields`, returning `MatchNone`, `MatchFuzzy`, or `MatchExact`. Pin exact tag classification, case-insensitive name/path fuzzy classification, repo-scope filtering when supplied, empty-query behavior for menu filtering, and NUL refusal for operation resolution. A second pure helper filters a complete candidate set by the highest present class, so any exact tag suppresses fuzzy siblings. Assert `ResolveThreadReference` and actionable-row filtering use these helpers rather than restating fields.

- [ ] **Step 2: Run matcher tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'Test(ThreadReferenceFields|ResolveThreadReference)' -count=1`

Expected: FAIL because the shared field predicate is absent.

- [ ] **Step 3: Implement matcher extraction and verify GREEN**

Add the pure exported classification/set helpers with no store access, preserve resolver ambiguity/sorting behavior, and rerun Step 2. Menu filtering passes its current in-memory `ActionableThreadSummary` fields and never invokes `Couch.ResolveThreadReference`.

- [ ] **Step 4: Write failing root/filter reducer tests**

Cover stable identity selection, exact-over-fuzzy filtering, first-match fallback,
zero-match Enter/Tab notices, clamped Up/Down, root Escape clear/return effects,
and immutable-by-copy inputs.

- [ ] **Step 5: Run root tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestReduceMenuRoot' -count=1`

Expected: FAIL because menu entities are absent.

- [ ] **Step 6: Implement root entities/transitions and verify GREEN**

Define fail-safe zero values and explicit frame ownership:

```go
type MenuFrameKind uint8
const (
    FrameUnknown MenuFrameKind = iota
    FrameThreads
    FrameActions
    FrameParkConfirm
    FrameTextInput
    FrameStart
)

type MenuFrame struct {
    Kind MenuFrameKind
    Target couchcore.ThreadAddress
    Filter string
    SelectedID string
    Input TextInputState
    Start StartFormState
}

type MenuState struct {
    Inventory []couchcore.ActionableThreadSummary
    Frames []MenuFrame
    BellThreads []couchcore.ThreadAddress
    Notice string
    RefreshPending bool
}

func ReduceMenu(state MenuState, event MenuEvent) (MenuState, []MenuEffect)
```

Implement only root/filter transitions, then rerun Step 5 and expect PASS.

- [ ] **Step 7: Write failing action/confirmation traces**

Cover root Tab → captured thread action frame, quiet leaf Tab, nested Escape,
live `park/name/describe`, parked `resume/name/describe`, cancel-default park
confirmation, shared operation descriptors, stale identity/applicability
refusal, and switch clearing the exact thread's ephemeral bell. Bell join/clear
uses a defensively copied address set and never mutates core/durable rows.

- [ ] **Step 8: Run action tests RED, implement, then GREEN**

Run before and after implementation: `go test -p 20 ./cmd/internal/couchtty -run 'TestReduceMenu(Action|Bell)' -count=1`

Expected: first FAIL, then PASS. Effects name existing shared operations (`name`,
not `rename`) and exact args but perform no I/O.

- [ ] **Step 9: Write failing text/start form traces**

Cover name/describe editing and UTF-8 Backspace, cancel/success/failure
preservation, 1 KiB name/filter and 4 KiB path/description byte bounds without
splitting UTF-8, Ctrl-Space from every list frame, no-op in text input, start
Tab/Left/Right, sticky agent, and exact originating-stack restoration.

- [ ] **Step 10: Run form tests RED, implement, then GREEN**

Run before and after implementation: `go test -p 20 ./cmd/internal/couchtty -run 'TestReduceMenu(Text|StartForm)' -count=1`

Expected: first FAIL, then PASS.

- [ ] **Step 11: Write failing refresh/reconciliation traces**

Pin all total outcomes: failure before any successful inventory yields
unavailable, later failure preserves complete last-good inventory/frames/drafts,
and successful refresh reconciles root-to-leaf before restoration. A hidden
target discards the first invalid frame/descendants/draft with notice; a global
start form survives with only its origin reconciled. Operation success applies
its returned affected-row projection when supplied; without one it retains
last-good data as refresh-pending. Neither path redispatches. Also pin
switch/resume/start failures as notice-only without presentation/process effects.

- [ ] **Step 12: Run reconciliation tests RED, implement, then GREEN**

Run before and after implementation: `go test -p 20 ./cmd/internal/couchtty -run 'TestReduceMenu(Refresh|Reconcile|OperationResult)' -count=1`

Expected: first FAIL, then PASS. Keep maximum depth structural (root + one
thread action + one confirmation/input, or one global start overlay).

- [ ] **Step 13: Keep the flat panel as a temporary compiling adapter**

Do not delete `PanelModel` yet. Adapt its construction/tests only as required by
the new actionable summary type so current `Console` and all package tests stay
buildable through M2. M3 migrates Console and deletes the adapter.

- [ ] **Step 14: Run reducer and core matcher packages**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty -run 'Test(ThreadReference|ReduceMenu|Panel)' -count=1`

Expected: PASS.

- [ ] **Step 15: Commit Task 5**

```bash
git add cmd/internal/couchcore/threadmetadata.go cmd/internal/couchcore/threadmetadata_test.go cmd/internal/couchtty/menu.go cmd/internal/couchtty/menu_test.go cmd/internal/couchtty/panel.go cmd/internal/couchtty/panel_test.go
git commit -m "#151 M2: reduce hierarchical menu state"
```

### Task 6: Render bounded wide/narrow menu frames

**Files:**
- Create: `cmd/internal/couchtty/menu_render.go`
- Create: `cmd/internal/couchtty/menu_render_test.go`

- [ ] **Step 1: Write failing render/layout tests**

Use golden string/table tests for 120x40 wide child placement, 40x10 single-column placement, and below-minimum resize message. Cover `▸` plus reverse-video selection, textual live/parked state, path, bell, description, parked relative age and three gray bands (`<24h`, `1–7d`, `>7d`), no color as sole state cue, controls by frame, zero matches, unavailable/refresh-pending notices, confirmation, text input, start fields/source, and safely clipped wide/combining Unicode.

Pin age boundaries exactly at 24h and 7d and use injected `now`; no renderer reads the clock or terminal. Strip ANSI in assertions and require every line's `textwidth.Width` ≤ columns and total lines ≤ rows. With 100 rows in 40x10, assert deterministic vertical viewport adjustment keeps selection visible at first/middle/last. Clamp a wide child beside the selected parent near both top and bottom; narrow child placement must remain within rows.

- [ ] **Step 2: Run render tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(RenderMenu|ChooseMenuLayout|AgeBand|Clip)' -count=1`

Expected: FAIL because render/layout entities are absent.

- [ ] **Step 3: Implement pure layout and rendering**

Define `MenuLayout` with frame rectangles/viewports and `RenderMenu(state, cols, rows, now) string`. Reuse existing `sanitize` and terminal-column `truncate` from `reserve.go`; do not add a rune-count clipping implementation. Compute a selection-containing vertical viewport before output and clamp child rectangles to the screen. Choose wide only when both parent/child minimum widths fit; otherwise stack below. Return only the resize message below 40x10.

- [ ] **Step 4: Run render tests and verify GREEN**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(RenderMenu|ChooseMenuLayout|AgeBand|Clip)' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit Task 6**

```bash
git add cmd/internal/couchtty/menu_render.go cmd/internal/couchtty/menu_render_test.go
git commit -m "#151 M2: render bounded menu frames"
```

### Task 7: Decode Tab through the real terminal key seam

**Files:**
- Modify: `cmd/internal/couchtty/panelkeys.go`
- Modify: `cmd/internal/couchtty/panelkeys_test.go`

- [ ] **Step 1: Write failing decoder tests**

Add `KeyTab` expectations for legacy HT (`0x09`) and Kitty CSI-u (`ESC [ 9 u`, including explicit no-modifier `;1`). Include every split point for CSI-u, reject modified Tab chords, and prove adjacent printable bytes remain ordered.

- [ ] **Step 2: Run decoder tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestDecode.*Tab' -count=1`

Expected: FAIL because Tab is currently ignored.

- [ ] **Step 3: Implement semantic Tab decoding**

Add the fail-safe enum member and map only HT and unmodified codepoint 9. Preserve mouse/unknown-sequence dropping and partial framing.

- [ ] **Step 4: Run decoder tests and verify GREEN**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestDecode' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit Task 7**

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

- [ ] **Step 1: Write failing scheduler traces**

Table-test edit bursts, completion order, cancellation supported/unsupported, failures, Escape, and Enter while unresolved. Assert one running request plus one replaceable latest pending generation, stale completions never update displayed profile, and Enter arms one submit for the current generation. `CancelPreview` only requests cancellation: it never retires `Running` or starts pending work. Exactly one terminal `PreviewFinished{generation, outcome}` event (success, failure, or canceled acknowledgment) retires the matching running request and starts the coalesced latest; duplicate/racing completions are ignored. Editing/Escape cancels the arm; only an accepted unchanged generation emits token-bound start.

- [ ] **Step 2: Run scheduler tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(AdvancePreviewSchedule|ReduceMenu.*Preview)' -count=1`

Expected: FAIL because preview scheduling is absent.

- [ ] **Step 3: Implement the pure scheduler and reducer events**

Model generation, running with `CancelRequested`, optional latest pending, optional armed-submit generation, and accepted resolution/token. Return effects `CancelPreview`, `StartPreview`, and `SubmitStart`; never call context cancellation directly. A worker is obligated to emit one terminal outcome even when cancellation wins, so pending work cannot strand and two previews never overlap. Bound retained text to the approved path/name limits before scheduling.

- [ ] **Step 4: Run all Chunk 2 package tests**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1`

Expected: PASS with no store/process/terminal mocks in pure entity tests.

- [ ] **Step 5: Commit Task 8**

```bash
git add cmd/internal/couchtty/menu_async.go cmd/internal/couchtty/menu_async_test.go cmd/internal/couchtty/menu.go cmd/internal/couchtty/menu_test.go
git commit -m "#151 M2: bound start preview scheduling"
```

### Task 9: Map and close the pure-menu boundary

**Files:**
- Modify: `atlas/couch.md`

- [ ] **Step 1: Update the atlas**

Document the landed pure hierarchical frame/reducer model, shared in-memory matcher, wide/narrow renderer, Tab encodings, input bounds, reconciliation precedence, and one-running/one-latest preview scheduler. Explicitly state that the current Console still presents the flat compatibility panel until M3; do not describe the hierarchical UI as reachable yet.

- [ ] **Step 2: Commit the M2 map**

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

Pin idle request → one start, requests while running → one dirty bit, completion/failure → reducer event plus exactly one follow-up when dirty, and stale/duplicate completion rejection. A refresh has one generation and the schedule can never report two running jobs.

- [ ] **Step 2: Run scheduler tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'TestAdvanceRefreshSchedule' -count=1`

Expected: FAIL because `RefreshSchedule` is absent.

- [ ] **Step 3: Implement the pure refresh schedule**

Return effects rather than spawning goroutines.

- [ ] **Step 4: Run scheduler tests and verify GREEN**

Re-run Step 2. Expected: PASS.

- [ ] **Step 5: Write failing observation/wiring tests**

Extend `pane` to retain the exact `ProcessIdentity` from the attached `StartResult.Handle`. Test observation snapshots include only registered, non-done children and carry exact address/PID/identity; duplicate incarnations remain separate observations. Test initial attach, resumed attach, exit, and switch never synthesize proof from persisted state.

Require production wiring to inject `func(context.Context, []LiveTTYObservation) ([]ActionableThreadSummary, error)` using `Couch.ActionableThreadInventory`; remove the old resolver-per-keystroke provider. Pin that Console construction/first attach requests background inventory, but opening the menu renders the current in-memory unavailable/last-good state before the provider is released. A blocking stateful provider must observe Console cancellation, emit exactly one canceled terminal completion, and let `Console.Stop` join within 250 ms.

- [ ] **Step 6: Run observation/wiring tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(Console.*Observation|Wire.*Actionable|MenuOpenDoesNotWait)' -count=1`

Expected: FAIL because pane identity and async actionable wiring are absent.

- [ ] **Step 7: Implement the bounded inventory worker**

Add one result channel to `Console.Run`; snapshots of pane observations occur under the Console mutex, while provider I/O occurs outside it. One worker derived from the Console lifetime context executes only effects emitted by `RefreshSchedule`, passes that context into the provider, and always sends one success/failure/canceled completion. Apply completion through `ReduceMenu`; never rebuild state directly. Start one refresh after initial attach and coalesce attach/exit/operation-triggered refreshes. Stop cancels and joins the worker during existing Console teardown; no provider may be injected without the context parameter.

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
- Modify: `cmd/internal/couchcore/runner_fake.go`
- Modify: `cmd/internal/couchcore/ptyrunner.go`
- Modify: `cmd/internal/couchcore/ptyrunner_test.go`
- Modify: `cmd/internal/couchcore/plan_contract_test.go`
- Modify: `cmd/internal/couchcore/starttransaction_integration_test.go`
- Modify: `cmd/probes/couchstartrecovery/main.go`
- Modify: `cmd/internal/couchcmd/run.go`
- Modify: `cmd/internal/couchcmd/run_test.go`

- [ ] **Step 1: Write failing result-carrying operation queue tests**

Change operation requests to return `(any, error)` and completions to carry the exact typed result plus stable request identity. Pin existing duplicate coalescing/capacity, one sequential action worker, panic-safe terminal completion, stop/join, and result defensive ownership. Add a table with `prepare-start`, token-bound `start`, `park`, `resume`, `name`, and `describe`, each initialized from its real form/frame/draft/selection state. For each operation prove: accepted enqueue enters its exact progress state and repaints; exact duplicate preserves that original progress and emits no second completion; overload immediately restores the initiating form/frame/draft/selection through its typed failure event, never enters progress, and expects no later completion. Test a successful mutation completion is reduced once even when its following refresh fails; no completion path reruns the request.

- [ ] **Step 2: Run queue tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty -run 'Test(OperationQueue|MenuOperationCompletion)' -count=1`

Expected: FAIL because completions currently discard results.

- [ ] **Step 3: Implement result-carrying action execution**

Keep local `switch` synchronous because it is an in-memory routing operation. Queue `park`, `resume`, `name`, `describe`, `prepare-start`, and token-bound `start`; only an accepted non-blocking enqueue emits progress state, then repaints immediately. Reduce queue-overload rejection synchronously through the requested operation's ordinary failure event; an exact duplicate is already represented by the original in-progress state and is a no-op. Convert returned `ThreadRecord`, `ParkResult`, or attached `StartResult` through the shared actionable projector plus current observations when possible; otherwise reduce success with no row and request refresh.

- [ ] **Step 4: Run queue tests and verify GREEN**

Re-run Step 2. Expected: PASS, including the `park_latency_test.go` callback signature migration.

- [ ] **Step 5: Write failing operation-dispatch context tests**

Add a runtime-only `Context context.Context` to `OperationCall`. Test that `DispatchOperation` substitutes `context.Background()` only when callers leave it nil, preserves a supplied canceled/deadline context, and delivers it to direct-store/live-owner executors. Test `CouchLiveOwnerExecutor` passes it to `PrepareStart`, Pair park/retry/recover, resume admission, and leave rather than manufacturing a background context; a blocking stateful policy resolver must observe cancellation. Preserve contextless public CLI behavior at `DispatchOperation`.

- [ ] **Step 6: Run operation-dispatch tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(DispatchOperationContext|PrepareStartCancellation|LifecycleOperationContext|ContextlessCLI)' -count=1`

Expected: FAIL because operation and lifecycle seams are contextless.

- [ ] **Step 7: Implement operation/lifecycle context propagation**

Thread context through `PrepareStart`, `SpawnPrepared`, resume admission, Pair lifecycle calls, leave, and their operation wiring.

- [ ] **Step 8: Run operation-dispatch tests and verify GREEN**

Re-run Step 6. Expected: PASS.

- [ ] **Step 9: Write failing blocked-runner cancellation tests**

Change the planned `Runner.StartBlocked` signature to accept `context.Context` and update every implementation/direct caller, including the recovery probe. Test cancellation before helper creation, after helper creation but before acknowledgement, and after acknowledgement during exact Pair registration. Before acknowledgement the exact channel closes and the helper is reaped; after acknowledgement the existing post-ack quiesce/reconcile path owns the exact handle. Registration timeout must derive from `context.WithTimeout(call.Context, registrationTimeout)`, never background. Cover ExecRunner, PtyRunner, FakeRunner, `launchTrackedThread`, the launch helper authority, the start transaction integration test, probe compilation, and `plan_contract_test.go`'s exact delegation strings.

- [ ] **Step 10: Run blocked-runner tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore ./cmd/probes/couchstartrecovery -run 'Test(RunnerCancellation|LaunchHelperCancellation|TrackedLaunchCancellation|StartTransaction|Issue149BlockedRunnersDelegateToOneHandshakeAuthority)' -count=1`

Expected: FAIL at the contextless signature/behavior.

- [ ] **Step 11: Implement blocked-runner cancellation**

Update the interface, every direct caller/implementation, helper cancellation, registration-derived timeout, exact cleanup, and recovery probe.

- [ ] **Step 12: Run blocked-runner tests and verify GREEN**

Re-run Step 10. Expected: PASS with every helper either transferred successfully or exactly canceled/reaped/reconciled.

- [ ] **Step 13: Write failing Console action-lifetime tests**

Require every accepted queued call to receive a child of the Console lifetime context. Table-test `Console.Stop` while token-bound `start` before/after acknowledgement, `park`, and `resume` are blocked; each observes cancellation, emits one terminal completion, and lets the action worker join within 250 ms. Also block `name`/`describe` at the injected dispatcher to prove the queue never omits cancellation, while production direct-store operations remain short local writes.

- [ ] **Step 14: Run Console action-lifetime tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestConsoleStopCancelsOperation' -count=1`

Expected: FAIL because accepted actions do not share one Console lifetime context.

- [ ] **Step 15: Bind accepted actions to Console lifetime**

Create one Console lifetime context, bind every accepted action call to a child, and cancel it before joining the action worker.

- [ ] **Step 16: Run Console action-lifetime tests and verify GREEN**

Re-run Step 14. Expected: PASS with one Console context owning every accepted action and worker join.

- [ ] **Step 17: Write failing preview-worker cancellation tests**

Use a blocking fake dispatcher and `context.WithCancel` to prove exactly one preview call runs, edits replace one latest pending generation, cancellation only requests stop, and the worker always sends one terminal success/failure/canceled completion. Race cancellation with normal completion and assert the reducer accepts one, ignores the duplicate, then starts only the coalesced latest. Block `prepare-start` and prove `Console.Stop` cancels it and joins the preview worker within 250 ms.

- [ ] **Step 18: Run preview tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestConsole(StartPreview|PreviewCancellation|PendingSubmit)' -count=1`

Expected: FAIL because Console does not execute preview effects.

- [ ] **Step 19: Implement preview/start effects**

Execute `prepare-start` with the current generation's path/agent and cancelable call context. Reduce its `StartResolution`/grant token only for the matching completion. An armed unchanged submit queues token-bound `start`; edits/Escape cancel the arm. After `StartResult`, dispatch existing owner-local attach, wait for exact registration as current start already does, take a fresh observation, and reduce the live row. Any start failure consumes the grant and schedules a new preview while preserving the form.

Do not run the tests in this step.

- [ ] **Step 20: Run preview tests and verify GREEN**

Re-run Step 18. Expected: PASS.

- [ ] **Step 21: Write failing exact-handle abort tests**

Test `Couch.AbortStarted(start StartResult, cause error) error` validates that the non-nil handle's PID/identity equal the returned record, delegates to `failPostAckStart(start.Record.Thread, start.Handle, cause)`, refuses nil or identity-mismatched payloads without touching another handle, and returns only after exact signal/reap and durable reconciliation.

- [ ] **Step 22: Run exact-handle abort tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchcore -run 'TestAbortStarted' -count=1`

Expected: FAIL because the owner method is absent.

- [ ] **Step 23: Implement exact-handle abort**

Add the narrow owner method and reuse the existing cleanup authority without duplicating signals/reconciliation.

- [ ] **Step 24: Run exact-handle abort tests and verify GREEN**

Re-run Step 22. Expected: PASS.

- [ ] **Step 25: Write failing transactional pane-attach/Stop tests**

Give each pane an exact process identity plus a pane-local watcher cancel/done pair. Specify a shared `removeExactPane(handle ID, thread address, process identity)` helper that cancels and joins that pane's reader/waiter, removes only its pane/order/observation entries, and restores the prior active/root/focus routing snapshot. Inject failures after pane insertion and after watcher start, plus `Console.Stop` while a partially installed pane exists. Assert all three paths remove the exact pane, observation, queued bytes, bell, selection, reader/waiter, and routing target; abort the exact started handle; restore terminal modes; and join before returning.

- [ ] **Step 26: Run transactional attach tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(ConsoleAttachRollback|ConsoleStopDuringAttach|WireAttachAbort)' -count=1`

Expected: FAIL because attach has no transactional rollback/partial-Stop contract.

- [ ] **Step 27: Implement transactional attach and partial-Stop cleanup**

Make attach commit/rollback one owner-local composition in `wireResolver`; cleanup finishes before reducer restoration.

- [ ] **Step 28: Run transactional attach tests and verify GREEN**

Re-run Step 26. Expected: PASS.

- [ ] **Step 29: Write semantic action/form restoration traces**

Drive semantic `MenuEvent`s through the controller/effect mapper and prove each operation is dispatched once with exact identity/arguments while input-state events continue to reduce and repaint during blocked work. Cover this failure matrix explicitly:

| Operation/path | Failure restoration and cleanup |
|---|---|
| `prepare-start` | Preserve the complete start form and originating stack, retain no token, and show a notice. |
| token-bound `start` refusal, stale grant, or fork failure | Treat the grant as consumed, preserve form/origin, request a fresh preview, and never attach. |
| `start` launches a helper but exact Pair registration fails | The context-bearing tracked-launch path quiesces/reconciles the exact handle before returning failure; preserve form/origin, consume the grant, and request fresh preview. |
| `start` returns `StartResult` but Console attach fails | Call `Couch.AbortStarted(StartResult, cause)`, which delegates to `failPostAckStart`; do not restore the form or return the error until the partial pane is rolled back and that exact handle is quiesced/reconciled. |
| `park` | If refreshed/returned projection still contains the live target, retain its action frame and notice; if hidden, reconcile to root. |
| `resume` refusal/fork failure | Preserve root selection and the parked row when still actionable; never attach. |
| `resume` launches a helper but exact Pair registration fails | The resume launch path quiesces and rolls back/marks unknown from exact evidence before returning; preserve root selection and reconcile the parked row from the returned projection/refresh. |
| `resume` returns `StartResult` but Console attach fails | Roll back the partial pane, abort the exact returned handle, preserve root selection, and retain the parked row after reconciliation when it remains actionable. |
| `name` / `describe` | Preserve the text-input frame and complete draft with a notice. |
| `switch` | Preserve root/selection and do not clear that thread's bell. |
| queue overload before acceptance | Emit the same typed failure event immediately, restoring the frame/form/draft for that operation; do not enter progress and expect no later completion. |
| exact duplicate while accepted request is running | Keep the original progress state, do not dispatch again, and expect only the original terminal completion. |
| successful mutation followed by refresh failure | Apply the returned row projection when present, otherwise keep last-good with refresh pending; never repeat the operation. |

The attach failure cases use the already-tested transactional cleanup from Steps 25–28. Only after cleanup completes does reducer restoration run: failed start restores the complete start form/originating stack, while failed resume restores the root with the same parked identity selected when it remains actionable. This is the failure half of declared `attach`, not a new menu verb.

- [ ] **Step 30: Run restoration traces and verify RED**

Run:

`go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(DispatchOperationContext|PrepareStartCancellation|AbortStarted|Console(Menu|StartPreview|PreviewCancellation|PendingSubmit|Operation|Park|Resume|Bell|Refresh))' -count=1`

Expected: FAIL because the controller does not yet map every restoration case.

- [ ] **Step 31: Implement remaining effect/restoration mappings**

Map reducer effects to exact shared operation calls and the restoration matrix above. Preserve current Alt+x leave/park ownership outside menu semantics and current clear-and-replay switch path.

- [ ] **Step 32: Run restoration traces and verify GREEN**

Re-run Step 30. Expected: PASS.

- [ ] **Step 33: Commit Task 11**

```bash
git add cmd/internal/couchtty/operation_queue.go cmd/internal/couchtty/operation_queue_test.go cmd/internal/couchtty/park_latency_test.go cmd/internal/couchtty/console_menu.go cmd/internal/couchtty/console_menu_test.go cmd/internal/couchtty/console.go cmd/internal/couchtty/console_panel_regression_test.go cmd/internal/couchcore/operationdispatch.go cmd/internal/couchcore/operationdispatch_test.go cmd/internal/couchcore/startresolution.go cmd/internal/couchcore/startresolution_test.go cmd/internal/couchcore/couch.go cmd/internal/couchcore/couch_test.go cmd/internal/couchcore/launch_existing.go cmd/internal/couchcore/launchhelper.go cmd/internal/couchcore/launchhelper_test.go cmd/internal/couchcore/resume.go cmd/internal/couchcore/resume_launch_test.go cmd/internal/couchcore/runner.go cmd/internal/couchcore/runner_test.go cmd/internal/couchcore/runner_fake.go cmd/internal/couchcore/ptyrunner.go cmd/internal/couchcore/ptyrunner_test.go cmd/internal/couchcore/plan_contract_test.go cmd/internal/couchcore/starttransaction_integration_test.go cmd/probes/couchstartrecovery/main.go cmd/internal/couchcmd/run.go cmd/internal/couchcmd/run_test.go
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

Drive `Console.Run` through raw `hostty.FakeHost` input at 120x40, 40x10, and below minimum. Feed both legacy HT and Kitty CSI-u Tab plus every control/action/form path: live switch + bell clear; parked resume; action menu; cancel/default and confirmed park; rename UI → shared `name`; describe; start path/agent/source preview; pending Enter; Escape/back; and zero selection. Assert every decoded semantic key reaches `ReduceMenu`, resize recomputes layout without state loss, mouse/focus/unknown escapes remain dropped, background child output never paints over menu ownership, and Escape/switch/teardown restore terminal modes/cursor/mouse exactly. This is the raw-input coverage; Task 11 tests the same behavior below the decoder through semantic events.

Update the control inventory test to require typeahead, arrows, Enter switch/resume, Tab actions, Ctrl-Space start, Alt+x park/leave, and Escape clear/back. Add README expectation tests before editing prose.

- [ ] **Step 2: Run replacement tests and verify RED**

Run: `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test(ConsoleRun.*Menu|MenuControls|Readme.*Couch)' -count=1`

Expected: FAIL while Console still renders `PanelModel`.

- [ ] **Step 3: Route all menu input/paint through reducer/renderer**

Replace `panel`, `query`, prompt, resolver, `rebuildPanel`, `showPanel`, and panel-key branches with `MenuState`, `ReduceMenu`, effect execution, and `RenderMenu`. Keep `Console.Run` the sole host writer. Delete the compatibility panel only after no production/test references remain.

Update `core_concepts_contract_test.go` to parse #151's current Core concepts table as a superseding inventory alongside #146: the latest status marks `PanelModel` deleted and requires direct tests for every new pure entity/integration point. Do not weaken the existing pure-source/import and direct-unit-test checks.

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

Commit rows `thread-000`…`thread-099`, 120x40 dimensions, filter bytes to `thread-09`, twenty alternating Down/Up events, selection-preserving refresh, and one blocked lifecycle feedback event. Add `BenchmarkMenu100` and `testing.AllocsPerRun` guards: navigation reducer ≤8 allocations/event, one filter-byte reducer ≤16, completed-refresh reconciliation ≤32, and full 100-row 120x40 render ≤256. Pin exactly one inventory job, one preview plus one pending, one sequential action worker with queue capacity 16, the 1 KiB filter/name and 4 KiB path/description caps, and below-minimum rendering. Portable tests assert these numeric bounds/complexity, not target-specific wall time.

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
