# Boundary Review — pair#151 (milestone M2)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 66ae7eef502eb996f4b2d7f096e0ee73090204b2..8f6b5aad650e5f209698a6826981e7966759e6eb |
| command | sdlc milestone-close --issue 151 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-30T21:47:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 establishes a clean pure reducer/render/scheduler foundation, and the pinned package and race-focused tests pass. However, several Spec-required transitions are either missing or modeled incorrectly: path-history start resolution is overridden, root resume completions disappear, valid filtered frames are discarded during refresh, accepted previews can exhaust grant capacity, and two presentation contracts are incomplete. These require fixes and regression tests before the boundary closes.

1. Strengths

- `MenuState`, `ReduceMenu`, rendering, and preview scheduling remain deterministic and free of I/O.
- The shared exact-over-fuzzy matcher correctly consolidates CLI and in-memory filtering behavior.
- Stable composite thread identities are retained throughout frames and operation effects.
- Input sizes, stack depth, rendering bounds, and preview concurrency are explicitly bounded.
- Atlas documentation correctly says the hierarchical menu remains inert until M3. No README update is required yet because no reachable user-facing surface changed.

2. Critical findings

- [menu.go:387](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:387): the start form always sends its root-agent fallback as an explicit `prepare-start` argument. Consequently, path history can never select a different default agent. Accepted resolution provenance also is not retained/rendered, while `"name"` leaks into the UI instead of the specified `"rename"`. Separate display descriptors from operation identifiers, omit `agent` until the operator makes it sticky, and project the accepted agent plus argv source into the frame. Add a regression where root=`claude`, path history=`codex`, and the visible action says `rename` while dispatch remains `name`.

- [menu.go:493](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:493): preview deduplication checks only `PreviewPending`, not an already accepted generation. Repeatedly tabbing away from and back to an unchanged path therefore issues another grant for the same generation; sixteen repetitions can fill the bounded grant store until expiry. Reuse an accepted resolution for an unchanged generation or explicitly retire superseded grants, with a repeated-Tab regression.

- [menu.go:714](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:714): operation results are correlated only through `CurrentFrame().Thread`. Root frames never set that field, so a resume dispatched by root Enter has both success and failure completions silently ignored. Model the captured in-flight origin/identity and test root-resume success and failure, including returned inventory.

- [menu.go:685](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:685): reconciliation treats the current selection as frame applicability. An action filter with zero matches sets `SelectedItem=""`; an unchanged inventory refresh then discards the still-valid action frame. Validate frame/action applicability independently, then reconcile the filtered selection. Add an unchanged-refresh regression with a zero-match action filter.

- [menu.go:315](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:315): confirmation is a list frame, but unlike root and action lists it ignores printable input and Backspace. This violates the universal list-frame filtering contract. Drive the same filter/selection helper across every list-frame kind and test confirmation filtering.

- [menu_render.go:51](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_render.go:51): wide children always start at row zero, and narrow children are merely stacked into equal-height rectangles. Neither placement is anchored beside or below the selected parent row as the Spec requires. Carry parent-row geometry into layout and test exact child origins for wide and narrow terminals.

- [menu.go:678](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:678): hidden-target reconciliation reports only an opaque tag and omits the required diagnostic location; it also loses the previous human label after replacing inventory. Preserve the prior target presentation or carry explicit diagnostic context, then test a named thread becoming non-actionable.

3. Important findings

None beyond the blocking findings above.

4. Minor findings

None.

5. Test coverage notes

Verification performed:

- `go test -p 1 ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1` — passed.
- Focused race run over matcher, reducer, renderer, decoder, and scheduler tests — passed.
- `go test ./cmd/internal/artifactpath -count=1` — passed.
- `git diff --check` — passed.

An initial parallel full-package run transiently failed `TestParkCoordinatorConstructorDoesNotQueryPairSession`; the isolated rerun and subsequent serial full-package run passed. The blocking behaviors above currently have no regression tests.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass for the shared matcher and operation declarations. The shared-operation presentation mapping needs one descriptor source rather than leaking operation identifiers.
- `ARCH-PURE`: Pass. M2’s declared entities are pure and directly unit-tested without filesystem, process, terminal, or network mocks.
- `ARCH-PURPOSE`: Flagged. Start defaults/provenance, list filtering, hidden-target diagnostics, and parent-anchored layout under-deliver explicit Spec behavior.
- `ARCH-MOCK`: Pass/N/A for this boundary. No external dependency was introduced; stateful Console/provider integration remains explicitly scheduled for M3.
- `ARCH-CONSTRAINTS`: Mostly pass: input, rendering, stack, and concurrency bounds are enforced. Accepted-generation grant accumulation violates the bounded-work intent.

The Core concepts table matches the staged implementation: M2 entities exist at their declared paths; `RefreshSchedule` and `PanelModel` deletion remain explicitly assigned to M3.

7. Plan revision recommendations

Append a `## Revisions` entry recording that M2 review expanded Task 5/6/8 verification to cover:

- optional-versus-explicit start-agent semantics and resolution provenance;
- every operation origin, including root resume;
- frame applicability independent of filtered selection;
- filtering for every list-frame kind;
- parent-row-anchored child geometry;
- accepted-generation grant reuse;
- label plus diagnostic-location preservation for hidden targets.

```findings
findings:
  - id: new
    severity: Critical
    family: shared-operation-consumer-sweep
    title: |
      The menu changes shared-operation semantics instead of projecting them
    detail: |
      This is the 2nd finding in family `shared-operation-consumer-sweep`. At menu.go:387-500 the root fallback is always sent as an explicit agent, preventing prepare-start from consulting path history; menu.go:522-525 and menu_render.go:147-151 omit accepted provenance; menu.go:555 exposes the operation name `name` instead of the specified UI label `rename`. Do not patch only these sites: state the rule for every menu consumer—preserve optional versus explicit arguments, project required result fields, and separate presentation labels from operation identifiers—then sweep all declared menu operations. Add tests that fail without the corrected path default, provenance rendering, and rename-to-name mapping (ARCH-DRY, ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: preview-capability-single-generation
    title: |
      An accepted generation can repeatedly allocate start grants
    detail: |
      menu.go:493 deduplicates only a pending preview. After acceptance, unchanged Tab navigation requests the same generation again and can fill the 16-entry grant store until expiry. Reuse the accepted generation or retire superseded authority, and pin repeated unchanged navigation with a test that fails when another grant is issued (ARCH-CONSTRAINTS).
  - id: new
    severity: Critical
    family: operation-result-origin-correlation
    title: |
      Root-level resume completions are silently ignored
    detail: |
      menu.go:714-719 correlates completion only through CurrentFrame().Thread, but a root frame never owns that field. Both success and failure after root Enter resume therefore disappear. Model the exact in-flight origin and add root-resume success/failure tests with returned inventory (ARCH-PURE, ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: frame-validity-selection-independence
    title: |
      Refresh discards a valid action frame when its filter has zero matches
    detail: |
      menu.go:685-689 treats SelectedItem membership as frame applicability. A zero-match action filter clears SelectedItem, so an unchanged refresh incorrectly pops the actionable thread's frame. Validate the captured frame/action separately from filtered selection and add the missing unchanged-refresh regression (ARCH-PURE, ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: list-frame-filter-uniformity
    title: |
      Confirmation list frames do not implement the shared filtering contract
    detail: |
      menu.go:315-339 ignores printable and Backspace keys even though the Spec requires every list frame to own filter text and stable selection. Apply the list filtering rule to root, action, and confirmation frames and test the complete enumeration rather than only this instance (ARCH-DRY, ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: hierarchical-layout-parent-anchoring
    title: |
      Child frames are not positioned relative to their selected parent row
    detail: |
      menu_render.go:51-73 starts wide children at row zero and divides narrow height without reference to the parent selection. This does not deliver the specified beside-parent-row or below-parent-list hierarchy. Carry parent-row geometry into layout and pin exact wide and narrow child origins (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: hidden-target-diagnostic-completeness
    title: |
      Hidden-target transitions lose the human label and diagnostic location
    detail: |
      menu.go:678-682 replaces inventory before constructing the notice and reports only the tag. The Spec requires the target label and diagnostic location while preserving the hidden durable record. Preserve prior presentation or pass explicit diagnostic context, and test a named target becoming non-actionable (ARCH-PURPOSE).
```

---

## Re-review — 2026-08-30T22:28:13-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 66ae7eef502eb996f4b2d7f096e0ee73090204b2..9f8f3c1a3a8e8a03e9294b9b541c48e76e394f8b |
| command | sdlc milestone-close --issue 151 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-30T22:28:13-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2’s pure reducer, renderer, decoder, and scheduler are well-factored, and all seven prior findings are now regression-tested and mutation-verified. Two blocking outcome bugs remain: a failed start without a newly created address is ignored and permanently wedges operation dispatch, while a failed switch loses the thread’s notification before focus succeeds.

1. Strengths

- Prior fixes BR-5 through BR-11 are implemented and pinned by tests that fail when their corresponding corrections are removed.
- Reducer and scheduler logic remain deterministic and free of I/O ([menu.go:173](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:173), [menu_async.go:45](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_async.go:45)).
- Wide/narrow layout now derives child origins from parent geometry and remains bounded ([menu_render.go:40](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_render.go:40)).
- Atlas and README accurately preserve the M2/M3 staging distinction; no reachable user-facing menu surface changed in M2.

2. Critical findings

- [menu.go:833](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:833): failed starts are rejected because `menuOperationMatches` requires every start result to carry a nonzero created-thread address. A failed start naturally has no created address, so its diagnostic is ignored, the form is not restored, and `InFlight` remains set—silently blocking every later operation.

  **This is the 2nd finding in family `operation-result-origin-correlation`.** Earlier work fixed root resume only. Do not patch just failed start: state the rule that result-generated fields cannot be required to correlate failures, enumerate every declared operation across success/failure and address-presence outcomes, and test the entire table. A reviewer-only failed-start regression fails at the pinned head. This violates ARCH-PURE and ARCH-PURPOSE.

- [menu.go:255](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:255): live-thread Enter deletes the bell before the asynchronous `switch` succeeds. If focus fails, the inactive-thread notification is permanently lost even though no switch occurred. Defer notification clearing until a successful correlated switch result; preserve it on failure. Add inactive-target success/failure regressions. This violates ARCH-PURPOSE.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1` passed at the pinned head.
- Focused race tests for reducer, renderer, decoder, and scheduler passed.
- `git diff --check` passed.
- All seven prior-finding tests were mutation-verified red when their fixes were removed.
- Two reviewer-only regressions fail at HEAD: failed start without an address, and failed switch retaining its bell.
- The full repository run reached green for the changed packages but was not wholly executable in this sandbox: `cmd/pair-go` tests were denied `/bin/ps`.

6. Architectural notes for upcoming work

- ARCH-DRY: Pass. Shared matcher, item presentation mapping, and scheduling authority avoid parallel algorithms.
- ARCH-PURE: Flagged for incomplete outcome modeling: failure correlation currently depends on a success-only result field.
- ARCH-PURPOSE: Flagged by both findings; failed start restoration and notification clearing are explicit Spec behavior.
- ARCH-MOCK: Pass/N/A. M2 adds no external dependency; stateful Console integration remains M3 work.
- ARCH-CONSTRAINTS: Pass. Input, stack, viewport, and preview-work bounds are enforced.

7. Plan revision recommendations

Append a `## Revisions` entry recording:

- the repeated `operation-result-origin-correlation` family and the complete operation × outcome × address-presence enumeration;
- the rule that optimistic UI state changes such as notification clearing commit only after successful correlated operation completion.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      Optional-agent semantics, accepted provenance, and rename/name presentation are implemented; targeted mutations make each regression fail.
  - id: BR-6
    disposition: addressed
    note: |
      An unchanged accepted generation suppresses another preview request; removing the accepted-generation guard makes the repeated-Tab regression fail.
  - id: BR-7
    disposition: addressed
    note: |
      Root resume success and failure use captured operation origin and returned inventory; rejecting root origins makes both regressions fail.
  - id: BR-8
    disposition: addressed
    note: |
      Frame applicability is independent of filtered selection; restoring the selection-membership guard makes the zero-match refresh regression fail.
  - id: BR-9
    disposition: addressed
    note: |
      Confirmation frames filter displayed labels and handle Backspace; removing those transitions makes the confirmation regression fail.
  - id: BR-10
    disposition: addressed
    note: |
      Wide and narrow child origins derive from parent geometry; reverting placement makes both exact-origin regressions fail.
  - id: BR-11
    disposition: addressed
    note: |
      Hidden-target notices retain the prior human label and composite diagnostic address; reverting this projection makes its regression fail.
findings:
  - id: new
    severity: Critical
    family: operation-result-origin-correlation
    title: |
      Failed starts without a created address are ignored and wedge dispatch
    detail: |
      This is the 2nd finding in family `operation-result-origin-correlation`. menu.go:833-840 requires every start completion to carry a nonzero address, although failure has no created thread. The result is ignored and InFlight never clears. Do not patch only start failure: state the correlation rule, enumerate every operation across success/failure and address-presence outcomes, and sweep that table in one round (ARCH-PURE, ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: notification-success-commit
    title: |
      A failed switch permanently discards the target thread notification
    detail: |
      menu.go:255-260 deletes the bell when switch is dispatched rather than when correlated success arrives. A focus failure therefore loses notification state despite no switch occurring. Commit bell clearing on successful switch only and add inactive-target success/failure regressions (ARCH-PURPOSE).
```

---

## Re-review — 2026-08-30T22:37:45-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 66ae7eef502eb996f4b2d7f096e0ee73090204b2..34e080e19dc256159eb5731ce3c7ca0dbdd08961 |
| command | sdlc milestone-close --issue 151 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-30T22:37:45-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pure reducer architecture is strong, the full suite passes, and BR-12’s correction is mutation-proven. The boundary remains blocked because BR-13 lacks its required inactive-target regression, Left/Right agent selection is unreachable from real terminal input, and bounded row clipping can erase mandatory lifecycle and notification cues.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      Failed start results without an address now clear InFlight and restore the form; reverting the correlation change makes both the focused transition test and exhaustive outcome table fail.
  - id: BR-13
    disposition: not-addressed
    note: |
      Bell clearing moved to correlated switch success and its mutation is detected, but the only regression switches the active thread; the explicitly required inactive-target success/failure regression is absent.
findings:
  - id: new
    severity: Critical
    family: semantic-key-reachability
    title: |
      The start form's Left/Right agent selector is unreachable from terminal input
    detail: |
      menu.go handles KeyLeft and KeyRight, but panelkeys.go emits neither: decodeSequence recognizes only Up, Down, and CSI-u keys. Real CSI C/D and SS3 OC/OD input is dropped, so the specified agent selector cannot operate. Decode both terminal modes and add every-split tests that drive the decoded keys through ReduceMenu (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: bounded-render-semantic-cues
    title: |
      Row clipping removes required lifecycle, age, and bell information
    detail: |
      menu_render.go clips the complete row before adding the bell, so a long label or path consumes the 40-column budget and removes the trailing live/parked age; appending and reclipping then removes the bell too. Reserve a protected suffix for lifecycle and notification cues and clip variable label/path fields within the remaining columns (ARCH-PURPOSE, ARCH-CONSTRAINTS).
```

1. Strengths

- [menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:834) now defines an explicit outcome/address correlation rule, including addressless start failures.
- [menu_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_test.go:414) enumerates every declared operation across success, failure, missing-address, and wrong-address shapes.
- Reducer, renderer, scheduler, and matcher remain deterministic and IO-free; focused race tests pass.
- [atlas/couch.md](/Users/xianxu/workspace/pair/atlas/couch.md:39) accurately documents M2 as inert pure infrastructure pending M3 Console integration. No README change is required yet because no runnable surface became reachable.

2. Critical findings

- **BR-13 remains open:** [menu_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_test.go:129) sets `ActiveAddress` and switches that same thread. Change the fixture to select a distinct live inactive thread, then verify dispatch preserves its bell, failure retains it, and success clears it.
- **ARCH-PURPOSE — semantic key reachability:** [panelkeys.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/panelkeys.go:137) never emits the key kinds consumed by [menu.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu.go:462). Add CSI/SS3 Left and Right decoding plus split-boundary reducer tests.
- **ARCH-PURPOSE / ARCH-CONSTRAINTS — protected rendering cues:** [menu_render.go](/Users/xianxu/workspace/pair/cmd/internal/couchtty/menu_render.go:330) clips away required trailing semantics. Construct rows from a protected state/age/bell suffix and a separately clipped variable prefix; test long/wide labels and paths at 40 columns for live, parked, and bell states.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- `go test -p 20 ./... -count=1` passed.
- Focused `couchcore`/`couchtty` tests and reducer/renderer race tests passed.
- `git diff --check` passed.
- Reverting the BR-12/BR-13 implementation changes in an isolated archive made the corresponding tests fail.
- Missing coverage: inactive-target switching, raw Left/Right-to-reducer reachability, and mandatory row cues under worst-case clipping.

6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** matching and menu transitions have shared authorities.
- **ARCH-PURE — pass:** the M2 entities are directly testable without filesystem, process, terminal, or network IO.
- **ARCH-PURPOSE — flag:** real Left/Right input and bounded semantic rendering under-deliver committed behavior.
- **ARCH-MOCK — pass:** M2 introduces no external interaction; M3’s planned Console/stateful-fake seam remains appropriate.
- **ARCH-CONSTRAINTS — flag:** scheduling and geometry are bounded, but the declared 40-column envelope does not preserve required row information.

7. Plan revision recommendations

Append a `## Revisions` entry recording that M2 must:

- prove every reducer key through the real decoder seam, including legacy CSI and application-mode SS3 Left/Right;
- reserve bounded rendering space for lifecycle, age, and notification cues;
- pin BR-13 with an actually inactive switch target.

---

## Re-review — 2026-08-30T22:46:29-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 151 — couch: hierarchical work-thread menu |
| repo | pair |
| issue file | workshop/issues/000151-hierarchical-thread-menu.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 66ae7eef502eb996f4b2d7f096e0ee73090204b2..dea71d59226c63b434920be4e31d210d07d89619 |
| command | sdlc milestone-close --issue 151 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-08-30T22:46:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The three open findings are genuinely addressed and pinned by focused regressions, and the full repository suite passes. M2 still cannot close: preview generations collide across start-form lifetimes, allowing an old completion/token to satisfy and potentially launch from a newly opened form. The plan’s Core concepts table also reports future M3 entities as already new/deleted, contradicting the pinned tree.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Bell clearing now occurs only after correlated switch success; the inactive-target success/failure regression fails under dispatch-time clearing.
  - id: BR-14
    disposition: addressed
    note: |
      CSI and SS3 horizontal arrows reach ReduceMenu, with every-split decoder coverage and end-to-end agent-selection coverage.
  - id: BR-15
    disposition: addressed
    note: |
      Root rendering reserves lifecycle, age, and bell suffix width; the 40-column long-label regression fails under whole-row clipping.
findings:
  - id: new
    severity: Critical
    family: preview-capability-single-generation
    title: |
      Reopened start forms can accept and launch an earlier form's preview
    detail: |
      This is the 2nd finding in family `preview-capability-single-generation`. New start frames reset Generation to 1 at menu.go:427-429, the scheduler suppresses another request when its running generation is also 1 at menu_async.go:57-59, and menu.go:546-569 accepts the old generation-1 completion into the reopened form and can dispatch it when Enter is armed. State the class rule: preview identity must be unique across form lifetimes, not merely within one form. Carry a form/request epoch through frame, schedule, result, and submit authority—or use a menu-lifetime monotonic identity—and add a reducer-plus-scheduler Escape/reopen trace proving an old prepared token cannot populate or launch the new form.
  - id: new
    severity: Critical
    family: documentation-current-state-accuracy
    title: |
      The Core concepts inventory reports future M3 surface as current
    detail: |
      This is the 2nd finding in family `documentation-current-state-accuracy`. The plan lists menu_refresh.go as new and PanelModel as deleted at lines 24-26, and lists console_menu.go and menu_perf_test.go at lines 76 and 79, but the pinned head lacks all three new files and still contains panel.go. Apply one rule across the complete table: every row must distinguish current-boundary state from final planned state, such as with a delivery-milestone/current-status column; do not patch only these four rows. The atlas statement that Escape and stale preview results cannot reuse authority must also be corrected until the lifetime bug is fixed.
```

1. Strengths

- `menu.go:255-259,808-810` correctly delays bell clearing until switch success.
- `panelkeys.go:145-155` shares four-direction decoding across CSI and SS3 rather than duplicating horizontal paths.
- `menu_render.go:330-348` protects semantic suffixes while clipping only variable label/path content.
- The reducer, renderer, decoder, and scheduler remain pure and directly unit-tested without filesystem, terminal, process, or network I/O.
- Atlas coverage was added and remains linked from `atlas/index.md`; README correctly remains unchanged because M2 introduces no reachable user-facing controls.

2. Critical findings

- `menu.go:427-429,541-569`; `menu_async.go:57-59`: preview identity is not unique across form lifetimes. Fix with an epoch-qualified identity and a failing Escape/reopen/armed-submit integration trace.
- `workshop/plans/000151-hierarchical-thread-menu-plan.md:17-26,70-79`: Core concepts status contradicts the M2 tree. Sweep the full table and distinguish current delivery state from planned final state.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

`go test -p 20 ./... -count=1` passed, as did `git diff --check`. BR-13 through BR-15 each have reachable regressions that would fail without their fixes. The missing test is the composed reducer/scheduler trace across form cancellation and reopening.

6. Architectural notes for upcoming work

- `ARCH-DRY`: pass—shared matching, reducer authority, and common CSI/SS3 arrow decoding avoid parallel rules.
- `ARCH-PURE`: pass—the M2 entities are deterministic and tested without I/O.
- `ARCH-PURPOSE`: flag—the preview lifetime collision violates the exact-current-resolution guarantee.
- `ARCH-MOCK`: pass/not applicable—M2 introduces no external-service or binary interaction; those seams remain assigned to M3.
- `ARCH-CONSTRAINTS`: pass for rendering and scheduler capacity; 40×10 bounds and one-running/one-pending scheduling are mechanically covered. Identity safety must be fixed without widening those bounds.

7. Plan revision recommendations

Append a `## Revisions` entry recording:

- Preview authority is identified by form lifetime plus generation; Escape/reopen and late completion are part of the required enumeration.
- Core concepts tables distinguish current-boundary status from final planned status, with every pure entity and integration point swept under that rule.
