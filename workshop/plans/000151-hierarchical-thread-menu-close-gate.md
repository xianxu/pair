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
    - "n": 5
      timestamp: "2026-08-30T22:28:13-07:00"
      agent: codex
      dispose:
        - id: BR-5
          disposition: addressed
          note: Optional-agent semantics, accepted provenance, and rename/name presentation are implemented; targeted mutations make each regression fail.
          round: 5
        - id: BR-6
          disposition: addressed
          note: An unchanged accepted generation suppresses another preview request; removing the accepted-generation guard makes the repeated-Tab regression fail.
          round: 5
        - id: BR-7
          disposition: addressed
          note: Root resume success and failure use captured operation origin and returned inventory; rejecting root origins makes both regressions fail.
          round: 5
        - id: BR-8
          disposition: addressed
          note: Frame applicability is independent of filtered selection; restoring the selection-membership guard makes the zero-match refresh regression fail.
          round: 5
        - id: BR-9
          disposition: addressed
          note: Confirmation frames filter displayed labels and handle Backspace; removing those transitions makes the confirmation regression fail.
          round: 5
        - id: BR-10
          disposition: addressed
          note: Wide and narrow child origins derive from parent geometry; reverting placement makes both exact-origin regressions fail.
          round: 5
        - id: BR-11
          disposition: addressed
          note: Hidden-target notices retain the prior human label and composite diagnostic address; reverting this projection makes its regression fail.
          round: 5
      findings:
        - id: BR-12
          severity: Critical
          title: Failed starts without a created address are ignored and wedge dispatch
          detail: 'This is the 2nd finding in family `operation-result-origin-correlation`. menu.go:833-840 requires every start completion to carry a nonzero address, although failure has no created thread. The result is ignored and InFlight never clears. Do not patch only start failure: state the correlation rule, enumerate every operation across success/failure and address-presence outcomes, and sweep that table in one round (ARCH-PURE, ARCH-PURPOSE).'
          family: operation-result-origin-correlation
          round: 5
        - id: BR-13
          severity: Critical
          title: A failed switch permanently discards the target thread notification
          detail: menu.go:255-260 deletes the bell when switch is dispatched rather than when correlated success arrives. A focus failure therefore loses notification state despite no switch occurring. Commit bell clearing on successful switch only and add inactive-target success/failure regressions (ARCH-PURPOSE).
          family: notification-success-commit
          round: 5
      boundary: M2
      blocked: true
    - "n": 6
      timestamp: "2026-08-30T22:37:45-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: addressed
          note: Failed start results without an address now clear InFlight and restore the form; reverting the correlation change makes both the focused transition test and exhaustive outcome table fail.
          round: 6
        - id: BR-13
          disposition: not-addressed
          note: Bell clearing moved to correlated switch success and its mutation is detected, but the only regression switches the active thread; the explicitly required inactive-target success/failure regression is absent.
          round: 6
      findings:
        - id: BR-14
          severity: Critical
          title: The start form's Left/Right agent selector is unreachable from terminal input
          detail: 'menu.go handles KeyLeft and KeyRight, but panelkeys.go emits neither: decodeSequence recognizes only Up, Down, and CSI-u keys. Real CSI C/D and SS3 OC/OD input is dropped, so the specified agent selector cannot operate. Decode both terminal modes and add every-split tests that drive the decoded keys through ReduceMenu (ARCH-PURPOSE).'
          family: semantic-key-reachability
          round: 6
        - id: BR-15
          severity: Critical
          title: Row clipping removes required lifecycle, age, and bell information
          detail: menu_render.go clips the complete row before adding the bell, so a long label or path consumes the 40-column budget and removes the trailing live/parked age; appending and reclipping then removes the bell too. Reserve a protected suffix for lifecycle and notification cues and clip variable label/path fields within the remaining columns (ARCH-PURPOSE, ARCH-CONSTRAINTS).
          family: bounded-render-semantic-cues
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-08-30T22:46:29-07:00"
      agent: codex
      dispose:
        - id: BR-13
          disposition: addressed
          note: Bell clearing now occurs only after correlated switch success; the inactive-target success/failure regression fails under dispatch-time clearing.
          round: 7
        - id: BR-14
          disposition: addressed
          note: CSI and SS3 horizontal arrows reach ReduceMenu, with every-split decoder coverage and end-to-end agent-selection coverage.
          round: 7
        - id: BR-15
          disposition: addressed
          note: Root rendering reserves lifecycle, age, and bell suffix width; the 40-column long-label regression fails under whole-row clipping.
          round: 7
      findings:
        - id: BR-16
          severity: Critical
          title: Reopened start forms can accept and launch an earlier form's preview
          detail: 'This is the 2nd finding in family `preview-capability-single-generation`. New start frames reset Generation to 1 at menu.go:427-429, the scheduler suppresses another request when its running generation is also 1 at menu_async.go:57-59, and menu.go:546-569 accepts the old generation-1 completion into the reopened form and can dispatch it when Enter is armed. State the class rule: preview identity must be unique across form lifetimes, not merely within one form. Carry a form/request epoch through frame, schedule, result, and submit authority—or use a menu-lifetime monotonic identity—and add a reducer-plus-scheduler Escape/reopen trace proving an old prepared token cannot populate or launch the new form.'
          family: preview-capability-single-generation
          round: 7
        - id: BR-17
          severity: Critical
          title: The Core concepts inventory reports future M3 surface as current
          detail: 'This is the 2nd finding in family `documentation-current-state-accuracy`. The plan lists menu_refresh.go as new and PanelModel as deleted at lines 24-26, and lists console_menu.go and menu_perf_test.go at lines 76 and 79, but the pinned head lacks all three new files and still contains panel.go. Apply one rule across the complete table: every row must distinguish current-boundary state from final planned state, such as with a delivery-milestone/current-status column; do not patch only these four rows. The atlas statement that Escape and stale preview results cannot reuse authority must also be corrected until the lifetime bug is fixed.'
          family: documentation-current-state-accuracy
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-08-30T22:56:44-07:00"
      agent: codex
      dispose:
        - id: BR-16
          disposition: addressed
          note: Menu-lifetime monotonic identity is reachable and mutation-verified by the reducer-plus-scheduler Escape/reopen regression.
          round: 8
        - id: BR-17
          disposition: not-addressed
          note: The tables now match the pinned tree, but no test fails when current-boundary statuses regress, as required by the claimed-fix contract.
          round: 8
      boundary: M2
      blocked: true
    - "n": 9
      timestamp: "2026-08-30T23:04:46-07:00"
      agent: codex
      dispose:
        - id: BR-17
          disposition: addressed
          note: The complete Core concepts inventory now distinguishes delivery from current M2 state, and TestIssue151CoreConceptsMatchM2Boundary validates every row plus a future-M3-as-current mutation.
          round: 9
      findings:
        - id: BR-18
          severity: Critical
          title: Delayed duplicate results can retire a newer identical operation
          detail: This is the 3rd finding in family operation-result-origin-correlation. MenuOperationOrigin and MenuEvent correlate only by operation and address, so a duplicate result from completed attempt A matches later attempt B for the same operation and target, clears B's InFlight state, and applies A's outcome. Add one menu-lifetime attempt identity and carry it through every operation effect and result; sweep switch, resume, park, name, describe, and start with a mutation-sensitive stale-A-after-B regression.
          family: operation-result-origin-correlation
          round: 9
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

## Round 5 — 2026-08-30T22:28:13-07:00 (codex) — BLOCKED

### Disposed

- BR-5 — addressed — Optional-agent semantics, accepted provenance, and rename/name presentation are implemented; targeted mutations make each regression fail.
- BR-6 — addressed — An unchanged accepted generation suppresses another preview request; removing the accepted-generation guard makes the repeated-Tab regression fail.
- BR-7 — addressed — Root resume success and failure use captured operation origin and returned inventory; rejecting root origins makes both regressions fail.
- BR-8 — addressed — Frame applicability is independent of filtered selection; restoring the selection-membership guard makes the zero-match refresh regression fail.
- BR-9 — addressed — Confirmation frames filter displayed labels and handle Backspace; removing those transitions makes the confirmation regression fail.
- BR-10 — addressed — Wide and narrow child origins derive from parent geometry; reverting placement makes both exact-origin regressions fail.
- BR-11 — addressed — Hidden-target notices retain the prior human label and composite diagnostic address; reverting this projection makes its regression fail.

### Raised

- **BR-12** [Critical] `operation-result-origin-correlation` Failed starts without a created address are ignored and wedge dispatch
  This is the 2nd finding in family `operation-result-origin-correlation`. menu.go:833-840 requires every start completion to carry a nonzero address, although failure has no created thread. The result is ignored and InFlight never clears. Do not patch only start failure: state the correlation rule, enumerate every operation across success/failure and address-presence outcomes, and sweep that table in one round (ARCH-PURE, ARCH-PURPOSE).
- **BR-13** [Critical] `notification-success-commit` A failed switch permanently discards the target thread notification
  menu.go:255-260 deletes the bell when switch is dispatched rather than when correlated success arrives. A focus failure therefore loses notification state despite no switch occurring. Commit bell clearing on successful switch only and add inactive-target success/failure regressions (ARCH-PURPOSE).

## Round 6 — 2026-08-30T22:37:45-07:00 (codex) — BLOCKED

### Disposed

- BR-12 — addressed — Failed start results without an address now clear InFlight and restore the form; reverting the correlation change makes both the focused transition test and exhaustive outcome table fail.
- BR-13 — not-addressed — Bell clearing moved to correlated switch success and its mutation is detected, but the only regression switches the active thread; the explicitly required inactive-target success/failure regression is absent.

### Raised

- **BR-14** [Critical] `semantic-key-reachability` The start form's Left/Right agent selector is unreachable from terminal input
  menu.go handles KeyLeft and KeyRight, but panelkeys.go emits neither: decodeSequence recognizes only Up, Down, and CSI-u keys. Real CSI C/D and SS3 OC/OD input is dropped, so the specified agent selector cannot operate. Decode both terminal modes and add every-split tests that drive the decoded keys through ReduceMenu (ARCH-PURPOSE).
- **BR-15** [Critical] `bounded-render-semantic-cues` Row clipping removes required lifecycle, age, and bell information
  menu_render.go clips the complete row before adding the bell, so a long label or path consumes the 40-column budget and removes the trailing live/parked age; appending and reclipping then removes the bell too. Reserve a protected suffix for lifecycle and notification cues and clip variable label/path fields within the remaining columns (ARCH-PURPOSE, ARCH-CONSTRAINTS).

## Round 7 — 2026-08-30T22:46:29-07:00 (codex) — BLOCKED

### Disposed

- BR-13 — addressed — Bell clearing now occurs only after correlated switch success; the inactive-target success/failure regression fails under dispatch-time clearing.
- BR-14 — addressed — CSI and SS3 horizontal arrows reach ReduceMenu, with every-split decoder coverage and end-to-end agent-selection coverage.
- BR-15 — addressed — Root rendering reserves lifecycle, age, and bell suffix width; the 40-column long-label regression fails under whole-row clipping.

### Raised

- **BR-16** [Critical] `preview-capability-single-generation` Reopened start forms can accept and launch an earlier form's preview
  This is the 2nd finding in family `preview-capability-single-generation`. New start frames reset Generation to 1 at menu.go:427-429, the scheduler suppresses another request when its running generation is also 1 at menu_async.go:57-59, and menu.go:546-569 accepts the old generation-1 completion into the reopened form and can dispatch it when Enter is armed. State the class rule: preview identity must be unique across form lifetimes, not merely within one form. Carry a form/request epoch through frame, schedule, result, and submit authority—or use a menu-lifetime monotonic identity—and add a reducer-plus-scheduler Escape/reopen trace proving an old prepared token cannot populate or launch the new form.
- **BR-17** [Critical] `documentation-current-state-accuracy` The Core concepts inventory reports future M3 surface as current
  This is the 2nd finding in family `documentation-current-state-accuracy`. The plan lists menu_refresh.go as new and PanelModel as deleted at lines 24-26, and lists console_menu.go and menu_perf_test.go at lines 76 and 79, but the pinned head lacks all three new files and still contains panel.go. Apply one rule across the complete table: every row must distinguish current-boundary state from final planned state, such as with a delivery-milestone/current-status column; do not patch only these four rows. The atlas statement that Escape and stale preview results cannot reuse authority must also be corrected until the lifetime bug is fixed.

## Round 8 — 2026-08-30T22:56:44-07:00 (codex) — BLOCKED

### Disposed

- BR-16 — addressed — Menu-lifetime monotonic identity is reachable and mutation-verified by the reducer-plus-scheduler Escape/reopen regression.
- BR-17 — not-addressed — The tables now match the pinned tree, but no test fails when current-boundary statuses regress, as required by the claimed-fix contract.

## Round 9 — 2026-08-30T23:04:46-07:00 (codex) — BLOCKED

### Disposed

- BR-17 — addressed — The complete Core concepts inventory now distinguishes delivery from current M2 state, and TestIssue151CoreConceptsMatchM2Boundary validates every row plus a future-M3-as-current mutation.

### Raised

- **BR-18** [Critical] `operation-result-origin-correlation` Delayed duplicate results can retire a newer identical operation
  This is the 3rd finding in family operation-result-origin-correlation. MenuOperationOrigin and MenuEvent correlate only by operation and address, so a duplicate result from completed attempt A matches later attempt B for the same operation and target, clears B's InFlight state, and applies A's outcome. Add one menu-lifetime attempt identity and carry it through every operation effect and result; sweep switch, resume, park, name, describe, and start with a mutation-sensitive stale-A-after-B regression.

## Open findings

- **BR-18** [Critical] `operation-result-origin-correlation` Delayed duplicate results can retire a newer identical operation
