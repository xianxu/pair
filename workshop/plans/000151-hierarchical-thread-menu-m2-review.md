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
