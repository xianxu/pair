# Boundary Review — pair#151 (milestone M3)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..542c5d58411e6db126048360676c479cad841df6 |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T15:42:38-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The hierarchical switcher is reachable, well-separated, documented, and broadly tested, but M3 cannot close yet. Production never supplies the operation-result inventory that the reducer tests assume, leaving successful mutations temporarily—or after refresh failure, indefinitely—represented by stale rows. The superseded flat-panel authority also remains despite the plan declaring its deletion. Finally, the latency harness passes but bypasses the real run-loop boundary it claims to measure.

```findings
findings:
  - id: new
    severity: Critical
    family: shared-operation-consumer-sweep
    title: |
      Successful operation projections are exercised only in reducer tests and never supplied by production
    detail: |
      This is the 3rd finding in family `shared-operation-consumer-sweep`. Earlier rounds fixed instances. Do NOT fix this instance alone: enumerate every operation result and state the shared rule for projecting or visibly deferring its committed state. `finishOperation` extracts only addresses from park/start results and never sets `InventorySet`; every production search hit for `InventorySet: true` is in tests. Start, park, name, and describe therefore repaint stale inventory before refresh, and a failed refresh can leave that stale view indefinitely rather than using returned state or remaining visibly refresh-pending.
  - id: new
    severity: Critical
    family: superseded-ui-authority-retirement
    title: |
      The planned deleted flat-panel authority remains as a parallel implementation
    detail: |
      The Core concepts row declares `PanelModel` deleted in M3 and its narrative says `MenuState` replaces it, but `panel.go`, legacy Console state, resolver/summaries callbacks, prompt state, and the old panel/action controller remain. The executable contract was changed to require this compatibility adapter rather than proving its removal. Delete the superseded authority and make the contract fail while any declared legacy symbols or production fields remain.
  - id: new
    severity: Important
    family: lifecycle-evidence-validation
    title: |
      Target latency evidence bypasses the Console run-loop boundary it claims to measure
    detail: |
      This is the 2nd finding in family `lifecycle-evidence-validation`. Earlier rounds fixed instances. Do NOT patch one timing label: state the evidence rule and apply it to every measured path. The harness directly calls `showMenu`, `onMenuInput`, and `finishMenuRefresh`, then times function return; it never sends raw bytes through the host input channel, runs `Console.Run`, or correlates a generation-specific repaint. Delays or misrouting in the actual select loop would not fail these measurements.
```

### Strengths

- `ProjectActionableThreads` fails closed on exact live-process evidence and exact resumable parked evidence ([actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:67)).
- Refresh and preview controllers enforce bounded single-flight/coalescing behavior with cancellation and joined workers ([console_menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console_menu.go:79)).
- Operation results are correlated to exact attempt, address, frame instance, and depth before reducer mutation ([menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:1017)).
- Transactional attach failure routes through exact-actor cleanup rather than leaking a started process ([run.go](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:442)).
- README and atlas changes accurately describe the reachable hierarchical controls and actionable/raw inventory distinction.

### Critical findings

1. ARCH-PURPOSE: Production does not consume operation-returned state.

   At [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:1449), `finishOperation` derives addresses but constructs no projected inventory. The reducer’s `InventorySet` path is tested extensively, yet has zero production setters. Establish one exhaustive result-consumption rule covering start, park, resume, name, describe, switch, and leave. Add Console-level tests proving each returned mutation is reflected without waiting for refresh and that refresh failure cannot leave an unmarked stale view.

2. ARCH-DRY / ARCH-PURPOSE: The compatibility panel was not retired.

   The plan declares deletion, while [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:77), [panel.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/panel.go:60), and the legacy controller beginning at [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:1059) retain the competing state and behavior. Remove the full old surface and invert the concept contract so retention fails tests.

### Important findings

1. ARCH-CONSTRAINTS: The target performance harness does not exercise its declared boundary.

   [menu_perf_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_perf_test.go:143) constructs a Console but directly invokes internal methods; [measureMenuPath](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_perf_test.go:208) merely times those calls. Drive the running Console through raw fake-host input/result channels and stop only on the matching generation-tagged repaint.

### Minor findings

None.

### Test coverage notes

- `go test -p 20 ./... -count=1`: PASS.
- `git diff --check <base> <head>`: PASS.
- `BenchmarkMenu100`: PASS; approximately 51–61 µs/op across paths on the current M2 Max.
- Opt-in target test: PASS, but its boundary is invalid for the reason above.
- No prior findings required disposition in this first M3 review round.

### Architectural notes

- ARCH-DRY: Flagged—the legacy panel duplicates the new menu authority.
- ARCH-PURE: Pass—the reducer, renderer, and schedulers remain pure; Console is the thin asynchronous shell.
- ARCH-PURPOSE: Flagged—production omits the reducer’s operation-result projection contract, and planned compatibility retirement is incomplete.
- ARCH-MOCK: Pass—the runner, terminal host, lifecycle, and resolver paths use shared injected/stateful seams.
- ARCH-CONSTRAINTS: Flagged—the implementation is bounded, but the target measurement does not validate the declared real interaction path.

### Plan revision recommendations

Append revisions that:

- Record complete retirement of `PanelModel`, legacy Console fields/controllers, and compatibility contract expectations.
- Enumerate every operation-result consumer and define the shared projection-or-visible-pending rule, including refresh failure.
- Replace the latency evidence with raw-host, running-loop, generation-correlated measurements and update the issue evidence after rerunning it.

---

## Re-review — 2026-08-31T16:06:47-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..d0e53c9e09b981491aaf8f189a7c013314c71cbc |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T16:06:47-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The flat-panel authority is genuinely retired, and the corrected performance harness now validates the real Console lifecycle well within its declared budgets. However, BR-21 remains blocking: a refresh started before a successful mutation can complete afterward and incorrectly clear `ProjectionPending`, exposing stale rows as current while the required follow-up refresh is still running. The plan also leaves delivered M3 work unchecked, contradicting its revisions and implementation.

```findings
dispose:
  - id: BR-21
    disposition: not-addressed
    note: |
      A successful pre-operation refresh clears ProjectionPending before the dirty post-operation follow-up completes, so committed mutations can still expose stale rows without the required pending marker.
  - id: BR-22
    disposition: addressed
    note: |
      The flat-panel source, controller, fields, callbacks, and tests are deleted, and the executable concept contract rejects their return.
  - id: BR-23
    disposition: addressed
    note: |
      All target paths now traverse a running Console through raw input, resize, or typed result channels and wait for a uniquely correlated FakeHost frame.
findings:
  - id: new
    severity: Important
    family: documentation-current-state-accuracy
    title: |
      Delivered M3 tasks remain unchecked in the authoritative plan
    detail: |
      This is the 3rd finding in family `documentation-current-state-accuracy`. Earlier rounds fixed instances. Do NOT update only one checkbox: sweep every M3 checklist item against the committed implementation and evidence. Tasks 10, 12, and 13 remain largely or wholly unchecked even though later revisions and the issue log claim delivery; leave only the boundary-close and subsequent issue-close steps open.
```

### Strengths

- BR-22 is addressed comprehensively: `panel.go` and `panel_test.go` are deleted, legacy Console fields/controllers are gone, and [core_concepts_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/core_concepts_contract_test.go:112) enforces retirement.
- The seven-operation projection policy is explicit and fail-safe for unknown future operations in [menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:1006).
- BR-23’s replacement harness drives the running Console and correlates each sample with unique emitted content in [menu_perf_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_perf_test.go:121).
- README and atlas accurately describe the reachable hierarchical switcher and its actionable-versus-diagnostic inventory distinction.

### Critical findings

1. **BR-21 — ARCH-PURPOSE: refresh provenance is insufficient.**

   [menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:265) clears `ProjectionPending` on every successful inventory result. But [console_menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console_menu.go:116) does not distinguish a generation started before the mutation from the dirty follow-up requested afterward.

   A scratch regression initialized generation 1 as pre-operation and dirty, then completed it successfully while generation 2 remained blocked. It failed because the state became `RefreshPending:true, ProjectionPending:false`.

   Track the minimum refresh generation or mutation epoch capable of clearing each committed projection. Apply that rule to all five mutating operations. Add a Console-level regression covering:

   `pre-operation refresh running → mutation succeeds → old refresh succeeds → follow-up blocks/fails`

   The pending marker must remain until a post-mutation snapshot succeeds.

### Important findings

1. **ARCH-PURPOSE: authoritative plan status contradicts delivery.**

   M3 Task 10 begins unchecked at [the plan](/Users/xianxu/workspace/pair/workshop/plans/000151-hierarchical-thread-menu-plan.md:606), Task 12 at line 877, and Task 13 at line 922, while the revision at line 1458 claims those surfaces are delivered. Sweep the entire M3 checklist rather than repairing isolated boxes.

### Minor findings

None.

### Test coverage notes

- `git diff --check`: PASS.
- `go test -p 20 ./... -count=1`: PASS.
- `go test -race ./cmd/internal/couchtty -count=1`: PASS.
- Target lifecycle performance protocol: PASS on Darwin/arm64; worst observed p95 was approximately 272 µs, comfortably within all budgets.
- The scratch refresh-order regression fails on pinned HEAD, confirming the remaining BR-21 behavior is reachable and currently untested.

### Architectural notes for upcoming work

- **ARCH-DRY:** Pass. The competing panel authority was removed.
- **ARCH-PURE:** Pass. Reducer, renderer, and schedules remain pure; Console owns the thin asynchronous shell.
- **ARCH-PURPOSE:** Flagged. Pending-state clearance does not yet cover the full refresh-generation lifecycle.
- **ARCH-MOCK:** Pass. Production and tests share the stateful host/provider seams.
- **ARCH-CONSTRAINTS:** Pass. The corrected lifecycle harness exercises the declared environment, workload, co-tenancy, and latency boundaries.

### Plan revision recommendations

Append a `## Revisions` entry that:

- Defines which refresh generation is authorized to clear a committed mutation’s pending state.
- Adds the pre-operation-success/follow-up-blocked-or-failed regression.
- Records a complete M3 checklist sweep, checking delivered implementation/evidence steps while leaving M3 close and issue close open until their gates succeed.

---

## Re-review — 2026-08-31T16:16:41-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..44c3271e09eaf064e496bc9f0dd68da05c3d4240 |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T16:16:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation is technically strong: BR-21 is fixed through a shared generation rule with reachable Console-level coverage, the flat-panel authority is gone, the real Console lifecycle is measured, and the full test suite passes. The boundary still cannot ship because BR-24 lacks the required executable regression, and the authoritative Core concepts headers plus README still contradict the delivered M3 boundary and reducer behavior.

```findings
dispose:
  - id: BR-21
    disposition: addressed
    note: |
      Production captures the latest admitted refresh generation for every projection-mutating success, and a Console-level regression proves a pre-mutation result cannot clear pending state while the dirty follow-up remains unresolved.
  - id: BR-24
    disposition: not-addressed
    note: |
      The M3 boxes are corrected, but no executable test or contract fails when the delivered Task 10, Task 12, or Task 13 boxes are reverted to unchecked, as this review explicitly requires.
findings:
  - id: new
    severity: Critical
    family: documentation-current-state-accuracy
    title: |
      Authoritative M3 documentation still contradicts the delivered boundary and root Escape contract
    detail: |
      This is the 4th finding in family `documentation-current-state-accuracy`. The Core concepts columns still claim “Current after M3 Task 10” and “Current after M3 Task 11” while containing Task 12 deletion and Task 13 delivery, and README.md:369-370 says root Escape exits Couch without an actor although the Spec and menu.go:392-395 require the root to remain visible with an error. Do not patch only these phrases: state one rule covering current-boundary headings and user-facing key semantics, sweep that class, and make the documentation contract fail on semantic drift.
```

### Strengths

- BR-21 is fixed at the production seam: [console.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console.go:1063) captures refresh provenance, while [menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:277) accepts only a strictly later generation.
- [console_menu_operation_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/console_menu_operation_test.go:200) exercises the real pre-mutation/follow-up ordering. Removing the production generation capture would make it fail.
- The seven-operation projection policy is centralized and fail-safe in [menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:1017).
- BR-22 remains genuinely disposed: `panel.go` and `panel_test.go` are deleted, and [core_concepts_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/core_concepts_contract_test.go:112) rejects their return.
- BR-23 remains disposed: [menu_perf_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_perf_test.go:121) drives raw events through a running Console and waits for correlated emitted frames.

### Critical findings

- **Documentation authority contradicts code — ARCH-PURPOSE.** The plan headers at lines 17 and 70 describe older Task 10/11 boundaries while their rows include later delivery. Separately, [README.md](/Users/xianxu/workspace/pair/README.md:369) says Escape exits Couch when no actor remains, but [menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:392) keeps the root visible and reports `no live thread can receive focus`, matching the Spec. Correct the full documentation class and extend the contract beyond merely checking that an `Escape` token appears.

### Important findings

- **BR-24 remains open.** The checklist now accurately leaves only M3 boundary close and issue close unchecked, but no test pins that state. Add a plan-contract test that enumerates the expected M3 checked/open steps and demonstrate that reverting a delivered checkbox makes it fail.

### Minor findings

None.

### Test coverage notes

Executed successfully:

- Focused `couchcore`, `couchtty`, and `couchcmd` packages.
- BR-21 projection-generation and production Console regressions.
- Core-concepts, performance-bound, and operation-policy tests.
- `go test -p 20 ./... -count=1`.
- `git diff --check`.

The opt-in M2 Max timing protocol was inspected but not rerun; its committed evidence reports the required target execution.

### Architectural notes

- **ARCH-DRY — pass:** one operation projection policy and one reducer authority; obsolete panel implementation removed.
- **ARCH-PURE — pass:** projection, reducer, refresh scheduler, and rendering remain pure; Console owns the thin asynchronous boundary.
- **ARCH-PURPOSE — flag:** implementation fulfills the switcher purpose, but authoritative documentation does not consistently describe it.
- **ARCH-MOCK — pass:** production and tests share the actionable-provider, runner, host, and Console boundaries; stateful fakes cover lifecycle interactions.
- **ARCH-CONSTRAINTS — pass:** single-flight/dirty scheduling, bounded queues, 100-row tests, allocation bounds, and target lifecycle measurements enforce the declared envelope.

### Plan revision recommendation

Append a revision titled `2026-08-31 — align boundary documentation with executable M3 semantics`:

- **Reason:** the Core concepts headers and README Escape semantics remained stale despite the M3 delivery sweep.
- **Delta:** rename both current-state columns to the actual M3 boundary, correct root Escape behavior, enumerate all current-boundary/key-semantic documentation assertions, and extend the contract so reverting either checklist completion or documented behavior fails.
