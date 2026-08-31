---
gate: boundary-review
issue: 151
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-30T20:57:48-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Required start tokens break the existing Console start consumer
          detail: start now requires an implicit token, but Console still dispatches start with only path; both existing Console start regressions time out. Sweep every shared-operation consumer and make Console perform prepare-start followed by token-bound start, or stage the schema change atomically.
          family: shared-operation-consumer-sweep
          round: 1
        - id: BR-2
          severity: Critical
          title: The actionable projector accepts corrupt verified-park evidence
          detail: ProjectActionableThreads checks only that VerifiedPark is non-nil and never validates the durable record, so a zero identity/attempt and other malformed records can become actionable despite the fail-closed Spec. Validate the record and add malformed live and parked regressions using valid positive fixtures.
          family: lifecycle-evidence-validation
          round: 1
        - id: BR-3
          severity: Important
          title: New production files are absent from the exhaustive source inventory
          detail: TestProductionArtifactReferencesAreExactlyClassified rejects actionableinventory.go, startgrant.go, and startresolution.go. Classify all three in the artifact authority inventory.
          family: exhaustive-production-source-inventory
          round: 1
        - id: BR-4
          severity: Important
          title: Atlas claims the ordinary switcher already consumes the actionable inventory
          detail: The atlas describes ActionableThreadInventory as the current switcher source, while run.go still wires ThreadInventory into the transitional panel. Document the milestone staging accurately or complete the wiring.
          family: documentation-current-state-accuracy
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-08-30T21:10:24-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Both production consumers now perform prepare-start then token-bound start; reverting the Console fix makes both panel start regressions fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Projection validates the complete durable record; reverting that validation makes malformed live and parked regressions fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: All three sources are classified; reverting the entries makes the exhaustive production-source inventory test fail.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: The atlas was corrected, but the project milestone still claims the ordinary switcher consumes actionable inventory while run.go wires raw ThreadInventory, and no regression guards this staging claim.
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-30T21:18:18-07:00"
      agent: codex
      dispose:
        - id: BR-4
          disposition: addressed
          note: Source, atlas, project, issue, plan, and README now consistently distinguish M1 authority from M3 consumer adoption; mutating the project claim makes the class-level regression fail.
          round: 3
      boundary: M1
      blocked: false
    - "n": 4
      timestamp: "2026-08-30T21:47:52-07:00"
      agent: codex
      findings:
        - id: BR-5
          severity: Critical
          title: The menu changes shared-operation semantics instead of projecting them
          detail: 'This is the 2nd finding in family `shared-operation-consumer-sweep`. At menu.go:387-500 the root fallback is always sent as an explicit agent, preventing prepare-start from consulting path history; menu.go:522-525 and menu_render.go:147-151 omit accepted provenance; menu.go:555 exposes the operation name `name` instead of the specified UI label `rename`. Do not patch only these sites: state the rule for every menu consumer—preserve optional versus explicit arguments, project required result fields, and separate presentation labels from operation identifiers—then sweep all declared menu operations. Add tests that fail without the corrected path default, provenance rendering, and rename-to-name mapping (ARCH-DRY, ARCH-PURPOSE).'
          family: shared-operation-consumer-sweep
          round: 4
        - id: BR-6
          severity: Critical
          title: An accepted generation can repeatedly allocate start grants
          detail: menu.go:493 deduplicates only a pending preview. After acceptance, unchanged Tab navigation requests the same generation again and can fill the 16-entry grant store until expiry. Reuse the accepted generation or retire superseded authority, and pin repeated unchanged navigation with a test that fails when another grant is issued (ARCH-CONSTRAINTS).
          family: preview-capability-single-generation
          round: 4
        - id: BR-7
          severity: Critical
          title: Root-level resume completions are silently ignored
          detail: menu.go:714-719 correlates completion only through CurrentFrame().Thread, but a root frame never owns that field. Both success and failure after root Enter resume therefore disappear. Model the exact in-flight origin and add root-resume success/failure tests with returned inventory (ARCH-PURE, ARCH-PURPOSE).
          family: operation-result-origin-correlation
          round: 4
        - id: BR-8
          severity: Critical
          title: Refresh discards a valid action frame when its filter has zero matches
          detail: menu.go:685-689 treats SelectedItem membership as frame applicability. A zero-match action filter clears SelectedItem, so an unchanged refresh incorrectly pops the actionable thread's frame. Validate the captured frame/action separately from filtered selection and add the missing unchanged-refresh regression (ARCH-PURE, ARCH-PURPOSE).
          family: frame-validity-selection-independence
          round: 4
        - id: BR-9
          severity: Critical
          title: Confirmation list frames do not implement the shared filtering contract
          detail: menu.go:315-339 ignores printable and Backspace keys even though the Spec requires every list frame to own filter text and stable selection. Apply the list filtering rule to root, action, and confirmation frames and test the complete enumeration rather than only this instance (ARCH-DRY, ARCH-PURPOSE).
          family: list-frame-filter-uniformity
          round: 4
        - id: BR-10
          severity: Critical
          title: Child frames are not positioned relative to their selected parent row
          detail: menu_render.go:51-73 starts wide children at row zero and divides narrow height without reference to the parent selection. This does not deliver the specified beside-parent-row or below-parent-list hierarchy. Carry parent-row geometry into layout and pin exact wide and narrow child origins (ARCH-PURPOSE).
          family: hierarchical-layout-parent-anchoring
          round: 4
        - id: BR-11
          severity: Critical
          title: Hidden-target transitions lose the human label and diagnostic location
          detail: menu.go:678-682 replaces inventory before constructing the notice and reports only the tag. The Spec requires the target label and diagnostic location while preserving the hidden durable record. Preserve prior presentation or pass explicit diagnostic context, and test a named target becoming non-actionable (ARCH-PURPOSE).
          family: hidden-target-diagnostic-completeness
          round: 4
      boundary: M2
      blocked: true
---

# Gate ledger — pair#151 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-30T20:57:48-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `shared-operation-consumer-sweep` Required start tokens break the existing Console start consumer
  start now requires an implicit token, but Console still dispatches start with only path; both existing Console start regressions time out. Sweep every shared-operation consumer and make Console perform prepare-start followed by token-bound start, or stage the schema change atomically.
- **BR-2** [Critical] `lifecycle-evidence-validation` The actionable projector accepts corrupt verified-park evidence
  ProjectActionableThreads checks only that VerifiedPark is non-nil and never validates the durable record, so a zero identity/attempt and other malformed records can become actionable despite the fail-closed Spec. Validate the record and add malformed live and parked regressions using valid positive fixtures.
- **BR-3** [Important] `exhaustive-production-source-inventory` New production files are absent from the exhaustive source inventory
  TestProductionArtifactReferencesAreExactlyClassified rejects actionableinventory.go, startgrant.go, and startresolution.go. Classify all three in the artifact authority inventory.
- **BR-4** [Important] `documentation-current-state-accuracy` Atlas claims the ordinary switcher already consumes the actionable inventory
  The atlas describes ActionableThreadInventory as the current switcher source, while run.go still wires ThreadInventory into the transitional panel. Document the milestone staging accurately or complete the wiring.

## Round 2 — 2026-08-30T21:10:24-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Both production consumers now perform prepare-start then token-bound start; reverting the Console fix makes both panel start regressions fail.
- BR-2 — addressed — Projection validates the complete durable record; reverting that validation makes malformed live and parked regressions fail.
- BR-3 — addressed — All three sources are classified; reverting the entries makes the exhaustive production-source inventory test fail.
- BR-4 — not-addressed — The atlas was corrected, but the project milestone still claims the ordinary switcher consumes actionable inventory while run.go wires raw ThreadInventory, and no regression guards this staging claim.

## Round 3 — 2026-08-30T21:18:18-07:00 (codex) — passed

### Disposed

- BR-4 — addressed — Source, atlas, project, issue, plan, and README now consistently distinguish M1 authority from M3 consumer adoption; mutating the project claim makes the class-level regression fail.

## Round 4 — 2026-08-30T21:47:52-07:00 (codex) — BLOCKED

### Raised

- **BR-5** [Critical] `shared-operation-consumer-sweep` The menu changes shared-operation semantics instead of projecting them
  This is the 2nd finding in family `shared-operation-consumer-sweep`. At menu.go:387-500 the root fallback is always sent as an explicit agent, preventing prepare-start from consulting path history; menu.go:522-525 and menu_render.go:147-151 omit accepted provenance; menu.go:555 exposes the operation name `name` instead of the specified UI label `rename`. Do not patch only these sites: state the rule for every menu consumer—preserve optional versus explicit arguments, project required result fields, and separate presentation labels from operation identifiers—then sweep all declared menu operations. Add tests that fail without the corrected path default, provenance rendering, and rename-to-name mapping (ARCH-DRY, ARCH-PURPOSE).
- **BR-6** [Critical] `preview-capability-single-generation` An accepted generation can repeatedly allocate start grants
  menu.go:493 deduplicates only a pending preview. After acceptance, unchanged Tab navigation requests the same generation again and can fill the 16-entry grant store until expiry. Reuse the accepted generation or retire superseded authority, and pin repeated unchanged navigation with a test that fails when another grant is issued (ARCH-CONSTRAINTS).
- **BR-7** [Critical] `operation-result-origin-correlation` Root-level resume completions are silently ignored
  menu.go:714-719 correlates completion only through CurrentFrame().Thread, but a root frame never owns that field. Both success and failure after root Enter resume therefore disappear. Model the exact in-flight origin and add root-resume success/failure tests with returned inventory (ARCH-PURE, ARCH-PURPOSE).
- **BR-8** [Critical] `frame-validity-selection-independence` Refresh discards a valid action frame when its filter has zero matches
  menu.go:685-689 treats SelectedItem membership as frame applicability. A zero-match action filter clears SelectedItem, so an unchanged refresh incorrectly pops the actionable thread's frame. Validate the captured frame/action separately from filtered selection and add the missing unchanged-refresh regression (ARCH-PURE, ARCH-PURPOSE).
- **BR-9** [Critical] `list-frame-filter-uniformity` Confirmation list frames do not implement the shared filtering contract
  menu.go:315-339 ignores printable and Backspace keys even though the Spec requires every list frame to own filter text and stable selection. Apply the list filtering rule to root, action, and confirmation frames and test the complete enumeration rather than only this instance (ARCH-DRY, ARCH-PURPOSE).
- **BR-10** [Critical] `hierarchical-layout-parent-anchoring` Child frames are not positioned relative to their selected parent row
  menu_render.go:51-73 starts wide children at row zero and divides narrow height without reference to the parent selection. This does not deliver the specified beside-parent-row or below-parent-list hierarchy. Carry parent-row geometry into layout and pin exact wide and narrow child origins (ARCH-PURPOSE).
- **BR-11** [Critical] `hidden-target-diagnostic-completeness` Hidden-target transitions lose the human label and diagnostic location
  menu.go:678-682 replaces inventory before constructing the notice and reports only the tag. The Spec requires the target label and diagnostic location while preserving the hidden durable record. Preserve prior presentation or pass explicit diagnostic context, and test a named target becoming non-actionable (ARCH-PURPOSE).

## Open findings

- **BR-5** [Critical] `shared-operation-consumer-sweep` The menu changes shared-operation semantics instead of projecting them
- **BR-6** [Critical] `preview-capability-single-generation` An accepted generation can repeatedly allocate start grants
- **BR-7** [Critical] `operation-result-origin-correlation` Root-level resume completions are silently ignored
- **BR-8** [Critical] `frame-validity-selection-independence` Refresh discards a valid action frame when its filter has zero matches
- **BR-9** [Critical] `list-frame-filter-uniformity` Confirmation list frames do not implement the shared filtering contract
- **BR-10** [Critical] `hierarchical-layout-parent-anchoring` Child frames are not positioned relative to their selected parent row
- **BR-11** [Critical] `hidden-target-diagnostic-completeness` Hidden-target transitions lose the human label and diagnostic location
