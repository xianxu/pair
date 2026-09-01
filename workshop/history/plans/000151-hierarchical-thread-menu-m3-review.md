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

---

## Re-review — 2026-08-31T16:24:54-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..069acfc68b35d68e10005d6c6dbf82a1f796540c |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T16:24:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Implementation and verification are strong, and BR-24/BR-25 are addressed. However, the authoritative Core concepts inventory still omits the delivered parked-resume proof architecture. Because this is the fifth recurrence of documentation-current-state drift and contradicts the code, it blocks M3 closure.

```findings
dispose:
  - id: BR-24
    disposition: addressed
    note: |
      All 57 M3 steps are checked or left open as required, and a mutation test rejects a delivered step being unchecked.
  - id: BR-25
    disposition: addressed
    note: |
      Both tables name the M3 boundary; README root-Escape semantics match the reducer, with executable drift tests.
findings:
  - id: new
    severity: Critical
    family: documentation-current-state-accuracy
    title: |
      Core concepts inventory still omits delivered parked-resume proof authority
    detail: |
      This is the 5th finding in family `documentation-current-state-accuracy`. The table omits `ParkedResumeObservation` and its `NativeBindingResolver`/session-inventory dependency, while adjacent prose still says the projector uses only live-owner observations; one row also gives the non-resolvable location “existing test seams.” The hardcoded contract mirrors this incomplete inventory, so it cannot catch the omission. Do not patch one row: state and enforce the rule that every delivered architectural entity and dependency has exhaustive repo-relative locations and current semantics, then sweep every Core concepts row.
```

### Strengths

- BR-21 remains substantively addressed: one exhaustive projection policy covers all seven operations, including start/resume early-return paths and generation-qualified refresh completion.
- BR-22 remains addressed: `panel.go` and `panel_test.go` are deleted, with production-source absence checks.
- BR-23 remains addressed: the target harness measures semantic input through a running `Console` to a correlated emitted frame.
- The pure reducer/render/scheduler core is cleanly separated from Console I/O, and stateful `FakeHost`/runner seams exercise production boundaries.

### Critical findings

- [Plan lines 19–30](/Users/xianxu/workspace/pair/workshop/plans/000151-hierarchical-thread-menu-plan.md:19), [lines 72–81](/Users/xianxu/workspace/pair/workshop/plans/000151-hierarchical-thread-menu-plan.md:72), [contract line 814](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:814): the final inventory omits `ParkedResumeObservation`, binding resolution, and accurate paths. Fix by deriving an exhaustive inventory from delivered architectural declarations, updating every row and explanatory paragraph, and adding mutation tests for entity, dependency, and path omission.

### Important findings

None.

### Minor findings

None.

### Test coverage notes

Fresh verification passed:

- `go test -p 20 ./...`
- `go test -race -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd`
- M2 Max lifecycle performance protocol: all baseline and four-worker trials passed; worst observed p95 was 301.083µs, comfortably inside the declared budgets.
- `git diff --check` passed.

### Architectural notes

- `ARCH-DRY`: Pass — shared projection, reducer, operation policy, and retired panel authority.
- `ARCH-PURE`: Pass — business logic remains pure; Console is the thin I/O shell.
- `ARCH-PURPOSE`: Flag — the authoritative architectural inventory under-reports the delivered resume-proof design.
- `ARCH-MOCK`: Pass — production flows share stateful fake seams.
- `ARCH-CONSTRAINTS`: Pass — representative 100-row lifecycle measurements enforce the declared envelope.

### Plan revision recommendations

Append a revision titled “derive the complete M3 architectural inventory” recording:

- Every delivered Core concept, including `ParkedResumeObservation`, its resolver dependency, and current semantics.
- Explicit repo-relative paths for every row.
- A source-derived contract that fails when an architectural entity, dependency, path, or current-state description is omitted.

---

## Re-review — 2026-08-31T16:33:50-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..2b04cc6fc174ec285468a4d4613651a16ff6c4cc |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T16:33:50-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The named BR-26 omissions are documented correctly, and all automated suites pass. However, BR-26 remains open: the new contract derives only six opt-in `pair:m3-concept` declarations and hardcodes `found != 6`. A scratch mutation adding an unmarked architectural declaration passed both issue-151 concept contracts, proving the contract still cannot detect the class of omission it claims to prevent.

```findings
dispose:
  - id: BR-26
    disposition: not-addressed
    note: |
      This is the 6th finding in family `documentation-current-state-accuracy`: the known rows are fixed, but an unmarked architectural declaration still passes because the source-derived contract scans only opt-in markers and hardcodes six concepts.
```

### Strengths

- `ParkedResumeObservation` now has accurate semantics and a resolvable location in [actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:28).
- Parked projection correctly requires one matching, non-empty native binding and rejects ambiguous proof at [actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:110).
- Production obtains parked proof through the context-bearing resolver before invoking the pure projector at [actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:147).
- BR-21 through BR-25 remain substantively disposed: generation-qualified projection refresh, flat-panel retirement, run-loop performance coverage, checklist enforcement, and README/root-Escape contracts remain present.

### Critical findings

- **BR-26 — ARCH-PURPOSE: the inventory remains opt-in, not exhaustive.** [plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:799) scans only declarations already marked `pair:m3-concept`, then requires exactly six at line 858. It has no fail-closed disposition for unmarked M3 declarations. In a scratch archive, adding `ReviewAddedM3Authority` without a marker left both `TestIssue151M3SourceConceptsAppearAtExactPlanPaths` and `TestIssue151CoreConceptsMatchCurrentBoundary` green.

  Fix the rule, not another row: derive the complete M3 production declaration set and require every declaration to receive an architectural/detail/retired disposition. Architectural entries must supply kind, exact repository paths, dependencies, and current semantics. Add a mutation test proving an unmarked declaration fails.

### Important findings

None.

### Minor findings

None.

### Test coverage notes

Passed:

- `git diff --check <base> <head>`
- `go test -p 20 ./... -count=1`
- Race tests for `couchcore`, `couchtty`, `couchcmd`, and `sessioninventory`
- Existing omission mutations for `ParkedResumeObservation` and the session-inventory path

The scratch unmarked-authority mutation unexpectedly passed, which is the decisive BR-26 red/green evidence.

### Architectural notes

- **ARCH-DRY:** Pass. The old panel authority is removed and actionable projection remains centralized.
- **ARCH-PURE:** Pass. Projection, reducer, rendering, and scheduling remain pure; resolver and terminal work stay at integration boundaries.
- **ARCH-PURPOSE:** Flag. The contract protects six selected instances rather than the exhaustive architectural-inventory purpose.
- **ARCH-MOCK:** Pass. Production and tests share the resolver, runner, host, and Console seams with stateful doubles.
- **ARCH-CONSTRAINTS:** Pass. Refresh and preview concurrency are bounded, and performance evidence traverses the real Console loop at the declared workload.

### Plan revision recommendation

Append a `## Revisions` entry recording that the six-marker contract was not exhaustive. Its delta should define a closed-set declaration-disposition rule, require exact paths and dependencies for every architectural entry, and add a mutation proving a newly introduced unmarked authority fails.

---

## Re-review — 2026-08-31T16:44:11-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..7ff7d8c43e3ce85fcfe3febc776dd521dd73adca |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T16:44:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation and broader verification remain strong, and BR-21 through BR-25 are substantively addressed. M3 still cannot close: BR-26’s new “closed-set disposition” contract records declaration names but not their architectural/detail classification, so removing a `pair:m3-concept` marker silently reclassifies authority while leaving the plan and all tests green. The same contract also derives a historical milestone from mutable current-worktree state rather than pinned base/head objects.

```findings
dispose:
  - id: BR-21
    disposition: addressed
    note: |
      The shared generation-qualified projection policy and reachable Console regression remain present and pass.
  - id: BR-22
    disposition: addressed
    note: |
      The flat-panel production authority and tests remain deleted, with executable retirement checks.
  - id: BR-23
    disposition: addressed
    note: |
      The performance harness continues to drive the running Console and correlate semantic input with emitted frames.
  - id: BR-24
    disposition: addressed
    note: |
      The complete M3 checklist state is pinned by a mutation test that rejects delivered work being unchecked.
  - id: BR-25
    disposition: addressed
    note: |
      Boundary headings and README root-Escape behavior match the reducer and remain covered by drift tests.
  - id: BR-26
    disposition: not-addressed
    note: |
      The digest inventories declaration names but omits their architectural/detail disposition; removing a concept marker leaves the digest unchanged and the marker validator accepts any remaining count, so source classification can contradict the plan without failing.
findings:
  - id: new
    severity: Important
    family: historical-boundary-oracle-pinning
    title: |
      The M3 declaration oracle reads an unpinned diff and mutable worktree bytes
    detail: |
      The source-set test runs git diff from the M2 base to the repository's current HEAD, while the digest parses current filesystem files. Any later Go change will rewrite this supposedly historical M3 boundary or fail unrelated CI. Pin both paths and bytes to the supplied M3 head object.
```

### 1. Strengths

- The Core concepts table now accurately includes `ParkedResumeObservation`, `NativeBindingResolver`, the session-inventory dependency, and resolvable repository paths ([plan](/Users/xianxu/workspace/pair/workshop/plans/000151-hierarchical-thread-menu-plan.md:19)).
- Parked authority is modeled explicitly and separately from live-owner proof ([actionableinventory.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:28)).
- Production resolver integration remains context-bearing and shares the session-inventory seam ([resume.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/resume.go:140)).
- README and atlas both changed in the boundary, satisfying the documentation-surface gate.
- The obsolete panel implementation is deleted rather than retained as parallel authority.

### 2. Critical findings

- **BR-26 — ARCH-PURPOSE:** [plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:923) hashes only `path | declaration kind | receiver | name`. It never includes the `pair:m3-concept` disposition. Meanwhile, the marker validator at [line 986](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:986) no longer requires an expected marker set or count.

  Removing the marker from `ParkedResumeObservation`, for example, changes neither the declaration digest nor the plan. The validator simply stops checking that declaration, leaving the source classification (“implementation detail”) inconsistent with the Core concepts table (“architectural”) while all contracts pass.

  Fix the class by making the closed-set ledger encode each declaration’s explicit disposition and validating architectural entries bidirectionally against the plan. Add mutations for:

  - removing an existing architectural marker;
  - changing architectural → detail while leaving the plan row;
  - changing detail → architectural without adding the complete plan entry;
  - adding a declaration without an explicit ledger disposition.

### 3. Important findings

- **Historical boundary oracle is not pinned:** [plan_contract_test.go:883](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:883) runs `git diff <base>` without an M3 head, and [line 929](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:929) reads current filesystem bytes. Pin an `issue151M3Head`, derive paths with `git diff <base> <head>`, and parse bytes from that same head object. Deleted-source checks should inspect the pinned tree, not current `os.Stat`.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed on the pinned head:

- Focused issue-151 concept, declaration, and checklist contracts.
- `git diff --check`.
- `go test -p 20 ./... -count=1`.
- Race tests for `couchcore`, `couchtty`, `couchcmd`, and `sessioninventory`.

The target-hardware performance protocol was inspected through its committed harness/evidence but not rerun.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** actionable projection is centralized and the superseded panel authority is removed.
- **ARCH-PURE — pass:** projection, reducer, renderer, and schedulers remain pure; terminal, resolver, and Git inspection stay at integration/test edges.
- **ARCH-PURPOSE — flag:** BR-26’s contract protects declaration membership, not the promised exhaustive architectural classification.
- **ARCH-MOCK — pass:** production and tests share context-bearing resolver, runner, host, and Console seams with stateful doubles.
- **ARCH-CONSTRAINTS — pass:** refresh/preview work remains bounded, race suites pass, and the performance harness exercises the real Console lifecycle at the declared workload.

### 7. Plan revision recommendations

Append a `## Revisions` entry stating:

- The declaration ledger includes an explicit disposition for every declaration, and its digest covers that disposition.
- Source markers and Core concepts rows are validated bidirectionally.
- Marker removal/reclassification mutations must fail.
- Historical M3 paths and bytes are both read from the pinned `0c40a8d… → 7ff7d8c…` range.

---

## Re-review — 2026-08-31T16:55:47-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..d3ee08d548aebb38eb8a6f15bea78cf71c2dafc8 |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T16:55:47-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M3 runtime implementation is well-tested and the previously accepted production fixes remain sound. However, BR-26 and BR-27 are still not addressed: the declaration ledger contradicts the plan’s architectural inventory, and the historical oracle pins the parent commit rather than the supplied M3 head. Both require another review round.

```findings
dispose:
  - id: BR-21
    disposition: addressed
    note: |
      The generation-qualified projection policy remains production-reachable, and its Console regression fails if a pre-mutation refresh is allowed to clear pending state.
  - id: BR-22
    disposition: addressed
    note: |
      The flat-panel implementation is deleted, and executable source scans reject restoration of PanelModel or its parallel authority.
  - id: BR-23
    disposition: addressed
    note: |
      The target harness drives Console.Run through semantic input and correlated emitted frames; the complete M2 Max protocol passed in this review.
  - id: BR-24
    disposition: addressed
    note: |
      The complete M3 checklist is pinned, including a mutation that rejects reverting a delivered step to unchecked.
  - id: BR-25
    disposition: addressed
    note: |
      The plan headings and README root-Escape behavior match the reducer and remain protected by executable drift tests.
  - id: BR-26
    disposition: not-addressed
    note: |
      The closed ledger calls unmarked declarations detail, yet RefreshSchedule and AdvanceRefreshSchedule are authoritative Core concepts without architectural markers; the validator compares only its hardcoded six-entry subset.
  - id: BR-27
    disposition: not-addressed
    note: |
      Both source paths and bytes are pinned to 7ff7d8c4, not the supplied M3 head d3ee08d; reverting the path query to current HEAD would still pass because the changed Go path set is identical.
```

1. Strengths

- BR-21 is correctly enforced through the production Console lifecycle and `TestConsolePreMutationRefreshCannotAuthorizeCommittedProjection`.
- BR-22’s obsolete `panel.go` authority is deleted and guarded against reintroduction.
- BR-23’s performance harness exercises raw input, refresh results, and correlated repaint output through the running Console. All baseline and four-worker-load trials passed.
- README and atlas updates cover the new hierarchical switcher surface and root-Escape semantics.
- The pure reducer/scheduler core remains cleanly separated from injected provider, runner, and host I/O.

2. Critical findings

- **BR-26 — `documentation-current-state-accuracy`, `ARCH-DRY`, `ARCH-PURPOSE`: architectural classification remains internally contradictory.** The plan declares `RefreshSchedule` and `AdvanceRefreshSchedule` as delivered Core concepts at `workshop/plans/000151-hierarchical-thread-menu-plan.md:24`, and they exist at `cmd/internal/couchtty/menu_refresh.go:5` and `:38`. Neither carries an M3 architectural marker. Consequently, `issue151M3Disposition` classifies them as detail at `cmd/internal/couchcore/plan_contract_test.go:1019-1037`, while the six-entry architectural ledger at `:110-117` and validator at `:1082-1093` never compare the full Core-concept set bidirectionally.

  This repeated family needs a class-level fix: enumerate every exact entity in both Core-concept tables and require one matching source disposition, path, kind, and current status; then require every architectural source declaration to occur exactly once in those tables. Do not merely add markers for this example.

3. Important findings

- **BR-27 — `historical-boundary-oracle-pinning`: the oracle does not represent the reviewed head.** `issue151M3Head` is `7ff7d8c4` at `cmd/internal/couchcore/plan_contract_test.go:31`, while this review’s supplied head is `d3ee08d`. Both the path query at `:893` and source-byte reader at `:1041` therefore omit the final boundary commit’s version of `plan_contract_test.go`.

  Use an immutable source/blob manifest derived from the supplied boundary or otherwise separate final oracle installation from the snapshot it validates. Add mutations proving that replacing pinned paths with `HEAD` and pinned bytes with worktree reads both fail. The current path test would still pass after reverting to `HEAD` because no new Go pathname was added between `7ff7d8c4` and `d3ee08d`.

4. Minor findings

None.

5. Test coverage notes

Passed:

- `go test -p 20 ./... -count=1`
- `go test -race -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`
- `PAIR_MENU_PERF_TARGET=m2-max go test -p 20 ./cmd/internal/couchtty -run '^TestMenuTargetPerformance$' -count=1 -v`
- `git diff --check` for the pinned review range

The open oracle defects pass the suite because their baselines encode the incorrect six-entry classification and parent-commit snapshot.

6. Architectural notes for upcoming work

- **ARCH-DRY — flag:** the hardcoded Core-concept contract and six-entry source ledger are parallel authorities and already disagree.
- **ARCH-PURE — pass:** reducer, renderer, refresh scheduling, and projection logic remain directly testable without I/O.
- **ARCH-PURPOSE — flag:** the claimed exhaustive architectural inventory and supplied-head pinning are not actually exhaustive or pinned to the reviewed boundary.
- **ARCH-MOCK — pass:** production runner/provider/host flows share injected seams with stateful fakes; no new direct external runtime dependency was introduced.
- **ARCH-CONSTRAINTS — pass:** bounded scheduling, 100-row workload, four-worker co-tenancy, and semantic-input-to-frame latency were exercised successfully.

7. Plan revision recommendations

Append—not overwrite—two `## Revisions` entries:

- Record that the six-entry declaration ledger contradicted the complete Core-concept tables, and define the bidirectional entity/path/kind/status rule used by the replacement contract.
- Record that `7ff7d8c4` was the parent rather than the reviewed M3 head, then describe the immutable blob/snapshot mechanism and the mutations proving independence from `HEAD` and worktree bytes.

---

## Re-review — 2026-08-31T17:14:48-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..ccba497882fab0a10a33e1dd2fc515896a79ce43 |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T17:14:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation and full test suite are healthy, and the latest commits materially improve the declaration ledger. However, both open findings remain: the Core concepts inventory still validates its own incomplete dependency list, and the historical oracle is pinned to `d3ee08d5`, not the supplied M3 head `ccba4978`. BR-26 is Critical, so the boundary cannot close.

1. Strengths

- The flat-panel authority is genuinely removed; `panel.go` and `panel_test.go` are deleted.
- Pure reducer/projection logic remains separated from Console, filesystem, and process I/O.
- The new typed declaration ledger gives each listed concept an individual lifecycle and source entry.
- README and atlas changes accompany the user-facing menu surface.
- `go test ./... -count=1 -p 20` passed across the repository.

2. Critical findings

- **BR-26 — not addressed:** [plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:115) treats each ledger entry’s `present` slice as the complete location set, but many entries contain only their declaration file. For example, the plan says `Couch.ActionableThreadInventory` wraps `ThreadStore.Snapshot`, live-owner observations, and `NativeBindingResolver`, while its ledger entry at line 138 lists only `actionableinventory.go`. Similar omitted dependency locations remain for Console and other integration rows. The validator at lines 1129–1166 compares the plan against that same manually incomplete ledger, so both can agree while dependencies are absent.

  **This is the 6th finding in family `documentation-current-state-accuracy`.** Do not patch another row. Define what counts as an architectural dependency, enumerate those dependencies for every integration entity with exhaustive repo-relative paths, and make a mutation removing any enumerated dependency fail independently of the plan text. `ARCH-PURPOSE` is flagged.

3. Important findings

- **BR-27 — not addressed:** [plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:31) pins `d3ee08d5`, while the supplied M3 head is `ccba4978`. The range `d3ee08d5..ccba4978` changes four Go files, including production sources. Moreover, the base-to-`d3ee08d5` and base-to-HEAD Go path sets are currently identical, so reverting line 929 to an unpinned diff would still pass. The test at lines 952–966 checks the Git-object helper, not that the source-set oracle itself goes red when its head argument is removed.

  **This is the 2nd finding in family `historical-boundary-oracle-pinning`.** Pin the complete oracle to the boundary-supplied snapshot through an immutable boundary datum/manifest, and mutation-test the actual source-set command with a fixture where moving HEAD changes membership. `ARCH-PURPOSE` is flagged.

4. Minor findings

None.

5. Test coverage notes

- Passed: `go test ./... -count=1 -p 20`.
- Passed focused packages: `couchcore`, `couchtty`, and `couchcmd`.
- The target M2 Max performance protocol is opt-in and was skipped without `PAIR_MENU_PERF_TARGET=m2-max`; prior recorded evidence was not rerun here.
- BR-27’s claimed source-set fix lacks a test that fails when that specific fix is reverted.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. The typed ledger replaces the earlier parallel marker authority.
- `ARCH-PURE`: Pass. Projection/reducer/scheduling logic remains pure, with Console as the I/O shell.
- `ARCH-PURPOSE`: Flag. The documentation authority and historical oracle still under-deliver their stated exhaustive/pinned contracts.
- `ARCH-MOCK`: Pass. Production boundaries use stateful `FakeRunner` and `FakeHost` seams.
- `ARCH-CONSTRAINTS`: Pass. The 100-row envelope, bounded scheduling, allocation checks, and Console lifecycle harness are present.

7. Plan revision recommendations

Append a `## Revisions` entry that:

- replaces the false claim that the current ledger exhaustively owns dependencies with the precise dependency-enumeration rule and completed sweep;
- records the actual immutable boundary snapshot mechanism;
- names the mutation tests proving dependency removal and moving-head source-set reads fail.

```findings
dispose:
  - id: BR-26
    disposition: not-addressed
    note: |
      The typed ledger enumerates declarations but still omits exhaustive dependency locations and validates the plan against the same incomplete path slices; this is the 6th documentation-current-state-accuracy finding.
  - id: BR-27
    disposition: not-addressed
    note: |
      The oracle pins d3ee08d5 rather than the supplied ccba4978 head, and reverting the source-set diff to moving HEAD leaves its test green because both ranges currently have the same Go path membership; this is the 2nd historical-boundary-oracle-pinning finding.
```

---

## Re-review — 2026-08-31T17:26:34-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..90999ea472ec063effb2f2e6c4dfa13368cb6568 |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T17:26:34-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation and full test suite are healthy, and BR-27 is addressed. BR-26 remains blocking: the architectural ledger is internally consistent but still cannot detect dependencies omitted from both the ledger and plan. `Couch.ActionableThreadInventory` directly invokes the `PathOps` seam, yet its dependency inventory omits `pathops.go`, demonstrating that the claimed exhaustive sweep is not enforced.

### Strengths

- Historical M3 source bytes are read from the immutable full commit ID in `issue151M3SourceAtRef` ([plan_contract_test.go:1128](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:1128)).
- The Core concepts tables now use one entity per row and correctly include `ParkedResumeObservation`, `NativeBindingResolver`, and `SessionInventoryNativeBindingResolver`.
- The declaration oracle classifies all changed Go sources and verifies retired panel files.
- README and atlas changes cover the new hierarchical-menu surface.
- `go test -p 20 ./...` and the focused issue-151 contract tests pass.

### Critical findings

- **BR-26 remains not addressed — ARCH-PURPOSE.** This is now the sixth observed instance in family `documentation-current-state-accuracy`. `Couch.ActionableThreadInventory` calls `c.Path.Physical` ([actionableinventory.go:165](/Users/xianxu/workspace/pair/cmd/internal/couchcore/actionableinventory.go:165)), whose repo-owned seam is declared in `cmd/internal/couchcore/pathops.go`, but its ledger entry omits that path ([plan_contract_test.go:139](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:139)). The mutation test only removes an already-enumerated dependency from the hand-maintained map ([plan_contract_test.go:953](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:953)); it cannot detect an omitted dependency.

  Fix the class: mechanically derive repo-owned seams referenced by each Integration declaration’s signature/body and compare that exact set with the typed ledger. Then sweep all Integration rows, including `PathOps`, and mutation-test removal of a dependency reference from pinned implementation—not merely removal from the ledger.

### Important findings

None.

### Minor findings

None.

### Test coverage notes

BR-27 has a meaningful regression: moving `HEAD` and worktree bytes are explicitly rejected. BR-26 lacks the required failing-without-the-fix proof because implementation-to-ledger dependency completeness is not tested.

### Architectural notes

- `ARCH-DRY`: Pass—the typed ledger is the single table authority.
- `ARCH-PURE`: Pass—reducers, projection, scheduling, and rendering remain pure; Console owns the injected IO boundary.
- `ARCH-PURPOSE`: Flag—dependency exhaustiveness remains asserted rather than enforced.
- `ARCH-MOCK`: Pass—stateful `FakeRunner` and `FakeHost` exercise production seams.
- `ARCH-CONSTRAINTS`: Pass—bounded workers and the target performance harness cover the declared operating envelope.

### Plan revision recommendations

Append a `## Revisions` entry stating that the prior “complete Integration dependency inventory” claim was disproven by the omitted `PathOps` seam. Record the mechanically enforced dependency rule, the complete Integration-row resweep, and the mutation proving an implementation dependency omission fails.

```findings
dispose:
  - id: BR-26
    disposition: not-addressed
    note: |
      The typed ledger omits the PathOps seam directly used by Couch.ActionableThreadInventory, and its tests cannot discover dependencies omitted from the hand-maintained ledger itself.
  - id: BR-27
    disposition: addressed
    note: |
      The source-set range is fixed and parsed bytes come from the immutable full M3 head object; regressions reject moving HEAD and mutable worktree bytes.
```

---

## Re-review — 2026-08-31T17:40:18-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..f7de2f202f6eb96e72a1a9d4d9d56522cac69947 |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T17:40:18-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The delivered M3 behavior and BR-26 inventory fix are well tested, and all affected package tests pass. However, BR-27 is no longer fully addressed: the new dependency oracle combines pinned source bytes with a declaration index parsed from the mutable worktree. This makes the supposedly historical M3 dependency inventory dependent on later checkout state.

## Strengths

- BR-26’s missing entities and dependencies now appear in the Core concepts table, including `ParkedResumeObservation`, `NativeBindingResolver`, and session-inventory paths.
- The `PathOps` fix has a meaningful mutation test: [plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:990) removes the actual production reference and verifies the derived dependency set stops matching.
- The architectural ledger compares mechanically derived Integration dependencies bidirectionally with documented paths.
- Both [README.md](/Users/xianxu/workspace/pair/README.md) and [atlas/couch.md](/Users/xianxu/workspace/pair/atlas/couch.md) changed in the boundary range.

## Critical findings

- **BR-27 remains open — historical dependency resolution reads mutable worktree state.** [plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:1035) builds the declaration index with `filepath.WalkDir`, and line 1049 parses files directly from disk. Only the entity being analyzed is loaded from the pinned Git object at line 1136. Consequently, later declaration moves, additions, imports, or receiver-field changes can alter the dependency result for the historical M3 snapshot. The existing worktree-rejection test at line 1259 exercises the declaration digest, not this new dependency-index path.

  This is the 2nd finding in family `historical-boundary-oracle-pinning`. Do not patch only `PathOps`: state and enforce the rule that every byte and path participating in a historical oracle—including indexes used to resolve references—comes from the same pinned Git object. Build the index from a deterministic pinned path set using `issue151M3SourceAtHead`, and add a mutation proving changed worktree-only declaration/index bytes cannot affect the result.

## Important findings

None.

## Minor findings

None.

## Test coverage notes

- Focused M3 declaration, dependency, mutation, ledger, and historical-oracle tests passed.
- Affected-package suite passed for `couchcore`, `couchtty`, `couchcmd`, `artifactpath`, `sessioninventory`, `sessionwatch`, `hostty`, and `textwidth`.
- `git diff --check` passed.
- The target-machine performance run and race suite were not repeated in this review.

## Architectural notes

- **ARCH-DRY:** Pass. One typed ledger and shared dependency derivation replace parallel hand-maintained inventories.
- **ARCH-PURE:** Pass. Pure menu/projection authorities remain separated from Console and provider I/O.
- **ARCH-PURPOSE:** Flagged by BR-27. A historical oracle is not fulfilled while one of its resolution inputs remains mutable.
- **ARCH-MOCK:** Pass. Production integration continues through the established runner, host, and lifecycle seams with stateful doubles.
- **ARCH-CONSTRAINTS:** Pass. The bounded single-flight scheduling and 100-row Console performance harness remain present.

## Plan revision recommendations

Append a `## Revisions` entry recording that the dependency oracle’s declaration/import/field index must also be built entirely from pinned Git-object paths and bytes, with a worktree-divergence regression.

```findings
dispose:
  - id: BR-21
    disposition: addressed
    note: |
      Production operation projections remain supplied through the shared completion and refresh policy.
  - id: BR-22
    disposition: addressed
    note: |
      The obsolete panel implementation remains deleted in the pinned range.
  - id: BR-23
    disposition: addressed
    note: |
      Target evidence continues to exercise the Console run-loop boundary.
  - id: BR-24
    disposition: addressed
    note: |
      The authoritative M3 implementation steps are checked, leaving only boundary and issue closure open.
  - id: BR-25
    disposition: addressed
    note: |
      Current documentation and the executable root-Escape contract remain aligned.
  - id: BR-26
    disposition: addressed
    note: |
      The Core concepts ledger now includes parked-resume authority and mechanically derived Integration dependencies, with a PathOps-removal mutation test.
  - id: BR-27
    disposition: not-addressed
    note: |
      The new dependency oracle parses its declaration/import/field index from mutable worktree files even though the analyzed declaration comes from the pinned M3 Git object.
```

---

## Re-review — 2026-08-31T17:51:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0c40a8d1880b49a9cac1a7f4d8cd24a2c713dba7..c6dab1a6af6fde762f69504aa1c39383f750ec28 |
| command | sdlc milestone-close --issue 151 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-08-31T17:51:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The M3 boundary is ready. BR-27 is addressed: every dependency-index path and byte now comes from the same immutable Git archive, and the regression would fail if worktree indexing returned. The focused contracts, full repository suite, race suite, documentation gate, and prior-finding anchors all pass inspection.

### 1. Strengths

- The dependency index is built entirely from `git archive` at the pinned M3 snapshot ([plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:1041)).
- The regression proves a post-snapshot worktree declaration cannot enter that index ([plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:965)).
- Integration dependencies are mechanically compared with the closed architectural ledger, including a production-reference removal mutation.
- The obsolete flat-panel authority is deleted and guarded against reintroduction.
- README and atlas accurately document the reachable hierarchical switcher and its operating envelope.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed:

- Focused M3 historical-oracle, dependency, checklist, and ledger contracts.
- `go test -p 20 ./... -count=1`
- `go test -race -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`
- `git diff --check` for the pinned range.

The target-specific M2 Max performance protocol was not repeated; portable bounds and the production Console lifecycle harness passed.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** one typed ledger and shared dependency derivation govern the architectural inventory.
- **ARCH-PURE — pass:** projection, reducers, rendering, and schedulers remain pure; Console owns the thin I/O shell.
- **ARCH-PURPOSE — pass:** the historical oracle now pins every participating path and byte, including its resolution index.
- **ARCH-MOCK — pass:** runner, host, provider, and lifecycle tests use the same injected seams as production with stateful doubles.
- **ARCH-CONSTRAINTS — pass:** scheduling is bounded, the 100-row envelope is tested, and the Console performance harness exercises semantic input through correlated output frames.

### 7. Plan revision recommendations

None. The plan already records the pinned dependency-index rule and its regression.

```findings
dispose:
  - id: BR-21
    disposition: addressed
    note: |
      Successful operation projections remain production-reachable and protected by the generation-qualified Console regression.
  - id: BR-22
    disposition: addressed
    note: |
      The superseded flat-panel implementation is deleted and guarded against restoration.
  - id: BR-23
    disposition: addressed
    note: |
      The performance harness measures the Console run-loop from semantic input to correlated emitted frames.
  - id: BR-24
    disposition: addressed
    note: |
      The delivered M3 checklist is checked and protected by a mutation contract.
  - id: BR-25
    disposition: addressed
    note: |
      README, atlas, plan, and executable root-Escape behavior remain aligned.
  - id: BR-26
    disposition: addressed
    note: |
      The Core concepts ledger includes parked-resume authority and mechanically derived Integration dependencies.
  - id: BR-27
    disposition: addressed
    note: |
      Every dependency-index path and byte now comes from one immutable M3 Git archive, and a post-snapshot declaration regression fails if mutable worktree indexing returns.
```
