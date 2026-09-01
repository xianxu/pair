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
    - "n": 10
      timestamp: "2026-08-30T23:14:15-07:00"
      agent: codex
      dispose:
        - id: BR-18
          disposition: addressed
          note: Exact nonzero attempt identity is propagated through every operation effect, origin, and result; the stale-A-after-B regression covers all six operations and fails if attempt matching is removed.
          round: 10
      findings:
        - id: BR-19
          severity: Critical
          title: Operation completions can mutate a replacement frame at the same depth
          detail: This is the 4th finding in family operation-result-origin-correlation. menuOperationOriginFrame identifies an origin only by frame kind and depth, so navigation during in-flight work can replace park A's confirmation or name/describe A's draft with frame B at the same position; A's completion then pops B. State the class rule that frame-local completion effects require an exact menu-lifetime frame-instance identity, sweep every operation and outcome, and add mutation-sensitive replacement-frame regressions.
          family: operation-result-origin-correlation
          round: 10
      boundary: M2
      blocked: true
    - "n": 11
      timestamp: "2026-08-30T23:24:35-07:00"
      agent: codex
      dispose:
        - id: BR-19
          disposition: addressed
          note: Exact monotonic frame-instance identity now gates frame-local restoration, and the six-operation replacement-frame regression would fail against the prior kind/depth-only implementation.
          round: 11
      findings:
        - id: BR-20
          severity: Critical
          title: Operation restoration discards a global start overlay opened after dispatch
          detail: This is the 5th finding in family operation-result-origin-correlation. A park failure slices the stack to the originating action frame, while successful park or resume slices it to root; either path removes a legal global start overlay opened after dispatch, contrary to the Spec. State the class rule that completion may transform its captured origin but must preserve unrelated later frames, then sweep every operation, outcome, and legal post-dispatch navigation with mutation-sensitive regressions.
          family: operation-result-origin-correlation
          round: 11
      boundary: M2
      blocked: true
    - "n": 12
      timestamp: "2026-08-31T07:40:08-07:00"
      agent: codex
      dispose:
        - id: BR-20
          disposition: addressed
          note: Completion restoration now transforms only its captured prefix and preserves a distinct later global start frame; the all-operation, all-outcome regression fails when that preservation is removed.
          round: 12
      boundary: M2
      blocked: false
    - "n": 13
      timestamp: "2026-08-31T15:42:38-07:00"
      agent: codex
      findings:
        - id: BR-21
          severity: Critical
          title: Successful operation projections are exercised only in reducer tests and never supplied by production
          detail: 'This is the 3rd finding in family `shared-operation-consumer-sweep`. Earlier rounds fixed instances. Do NOT fix this instance alone: enumerate every operation result and state the shared rule for projecting or visibly deferring its committed state. `finishOperation` extracts only addresses from park/start results and never sets `InventorySet`; every production search hit for `InventorySet: true` is in tests. Start, park, name, and describe therefore repaint stale inventory before refresh, and a failed refresh can leave that stale view indefinitely rather than using returned state or remaining visibly refresh-pending.'
          family: shared-operation-consumer-sweep
          round: 13
        - id: BR-22
          severity: Critical
          title: The planned deleted flat-panel authority remains as a parallel implementation
          detail: The Core concepts row declares `PanelModel` deleted in M3 and its narrative says `MenuState` replaces it, but `panel.go`, legacy Console state, resolver/summaries callbacks, prompt state, and the old panel/action controller remain. The executable contract was changed to require this compatibility adapter rather than proving its removal. Delete the superseded authority and make the contract fail while any declared legacy symbols or production fields remain.
          family: superseded-ui-authority-retirement
          round: 13
        - id: BR-23
          severity: Important
          title: Target latency evidence bypasses the Console run-loop boundary it claims to measure
          detail: 'This is the 2nd finding in family `lifecycle-evidence-validation`. Earlier rounds fixed instances. Do NOT patch one timing label: state the evidence rule and apply it to every measured path. The harness directly calls `showMenu`, `onMenuInput`, and `finishMenuRefresh`, then times function return; it never sends raw bytes through the host input channel, runs `Console.Run`, or correlates a generation-specific repaint. Delays or misrouting in the actual select loop would not fail these measurements.'
          family: lifecycle-evidence-validation
          round: 13
      boundary: M3
      blocked: true
    - "n": 14
      timestamp: "2026-08-31T16:06:47-07:00"
      agent: codex
      dispose:
        - id: BR-21
          disposition: not-addressed
          note: A successful pre-operation refresh clears ProjectionPending before the dirty post-operation follow-up completes, so committed mutations can still expose stale rows without the required pending marker.
          round: 14
        - id: BR-22
          disposition: addressed
          note: The flat-panel source, controller, fields, callbacks, and tests are deleted, and the executable concept contract rejects their return.
          round: 14
        - id: BR-23
          disposition: addressed
          note: All target paths now traverse a running Console through raw input, resize, or typed result channels and wait for a uniquely correlated FakeHost frame.
          round: 14
      findings:
        - id: BR-24
          severity: Important
          title: Delivered M3 tasks remain unchecked in the authoritative plan
          detail: 'This is the 3rd finding in family `documentation-current-state-accuracy`. Earlier rounds fixed instances. Do NOT update only one checkbox: sweep every M3 checklist item against the committed implementation and evidence. Tasks 10, 12, and 13 remain largely or wholly unchecked even though later revisions and the issue log claim delivery; leave only the boundary-close and subsequent issue-close steps open.'
          family: documentation-current-state-accuracy
          round: 14
      boundary: M3
      blocked: true
    - "n": 15
      timestamp: "2026-08-31T16:16:41-07:00"
      agent: codex
      dispose:
        - id: BR-21
          disposition: addressed
          note: Production captures the latest admitted refresh generation for every projection-mutating success, and a Console-level regression proves a pre-mutation result cannot clear pending state while the dirty follow-up remains unresolved.
          round: 15
        - id: BR-24
          disposition: not-addressed
          note: The M3 boxes are corrected, but no executable test or contract fails when the delivered Task 10, Task 12, or Task 13 boxes are reverted to unchecked, as this review explicitly requires.
          round: 15
      findings:
        - id: BR-25
          severity: Critical
          title: Authoritative M3 documentation still contradicts the delivered boundary and root Escape contract
          detail: 'This is the 4th finding in family `documentation-current-state-accuracy`. The Core concepts columns still claim “Current after M3 Task 10” and “Current after M3 Task 11” while containing Task 12 deletion and Task 13 delivery, and README.md:369-370 says root Escape exits Couch without an actor although the Spec and menu.go:392-395 require the root to remain visible with an error. Do not patch only these phrases: state one rule covering current-boundary headings and user-facing key semantics, sweep that class, and make the documentation contract fail on semantic drift.'
          family: documentation-current-state-accuracy
          round: 15
      boundary: M3
      blocked: true
    - "n": 16
      timestamp: "2026-08-31T16:24:54-07:00"
      agent: codex
      dispose:
        - id: BR-24
          disposition: addressed
          note: All 57 M3 steps are checked or left open as required, and a mutation test rejects a delivered step being unchecked.
          round: 16
        - id: BR-25
          disposition: addressed
          note: Both tables name the M3 boundary; README root-Escape semantics match the reducer, with executable drift tests.
          round: 16
      findings:
        - id: BR-26
          severity: Critical
          title: Core concepts inventory still omits delivered parked-resume proof authority
          detail: 'This is the 5th finding in family `documentation-current-state-accuracy`. The table omits `ParkedResumeObservation` and its `NativeBindingResolver`/session-inventory dependency, while adjacent prose still says the projector uses only live-owner observations; one row also gives the non-resolvable location “existing test seams.” The hardcoded contract mirrors this incomplete inventory, so it cannot catch the omission. Do not patch one row: state and enforce the rule that every delivered architectural entity and dependency has exhaustive repo-relative locations and current semantics, then sweep every Core concepts row.'
          family: documentation-current-state-accuracy
          round: 16
      boundary: M3
      blocked: true
    - "n": 17
      timestamp: "2026-08-31T16:33:50-07:00"
      agent: codex
      dispose:
        - id: BR-26
          disposition: not-addressed
          note: 'This is the 6th finding in family `documentation-current-state-accuracy`: the known rows are fixed, but an unmarked architectural declaration still passes because the source-derived contract scans only opt-in markers and hardcodes six concepts.'
          round: 17
      boundary: M3
      blocked: true
    - "n": 18
      timestamp: "2026-08-31T16:44:11-07:00"
      agent: codex
      dispose:
        - id: BR-21
          disposition: addressed
          note: The shared generation-qualified projection policy and reachable Console regression remain present and pass.
          round: 18
        - id: BR-22
          disposition: addressed
          note: The flat-panel production authority and tests remain deleted, with executable retirement checks.
          round: 18
        - id: BR-23
          disposition: addressed
          note: The performance harness continues to drive the running Console and correlate semantic input with emitted frames.
          round: 18
        - id: BR-24
          disposition: addressed
          note: The complete M3 checklist state is pinned by a mutation test that rejects delivered work being unchecked.
          round: 18
        - id: BR-25
          disposition: addressed
          note: Boundary headings and README root-Escape behavior match the reducer and remain covered by drift tests.
          round: 18
        - id: BR-26
          disposition: not-addressed
          note: The digest inventories declaration names but omits their architectural/detail disposition; removing a concept marker leaves the digest unchanged and the marker validator accepts any remaining count, so source classification can contradict the plan without failing.
          round: 18
      findings:
        - id: BR-27
          severity: Important
          title: The M3 declaration oracle reads an unpinned diff and mutable worktree bytes
          detail: The source-set test runs git diff from the M2 base to the repository's current HEAD, while the digest parses current filesystem files. Any later Go change will rewrite this supposedly historical M3 boundary or fail unrelated CI. Pin both paths and bytes to the supplied M3 head object.
          family: historical-boundary-oracle-pinning
          round: 18
      boundary: M3
      blocked: true
    - "n": 19
      timestamp: "2026-08-31T16:55:47-07:00"
      agent: codex
      dispose:
        - id: BR-21
          disposition: addressed
          note: The generation-qualified projection policy remains production-reachable, and its Console regression fails if a pre-mutation refresh is allowed to clear pending state.
          round: 19
        - id: BR-22
          disposition: addressed
          note: The flat-panel implementation is deleted, and executable source scans reject restoration of PanelModel or its parallel authority.
          round: 19
        - id: BR-23
          disposition: addressed
          note: The target harness drives Console.Run through semantic input and correlated emitted frames; the complete M2 Max protocol passed in this review.
          round: 19
        - id: BR-24
          disposition: addressed
          note: The complete M3 checklist is pinned, including a mutation that rejects reverting a delivered step to unchecked.
          round: 19
        - id: BR-25
          disposition: addressed
          note: The plan headings and README root-Escape behavior match the reducer and remain protected by executable drift tests.
          round: 19
        - id: BR-26
          disposition: not-addressed
          note: The closed ledger calls unmarked declarations detail, yet RefreshSchedule and AdvanceRefreshSchedule are authoritative Core concepts without architectural markers; the validator compares only its hardcoded six-entry subset.
          round: 19
        - id: BR-27
          disposition: not-addressed
          note: Both source paths and bytes are pinned to 7ff7d8c4, not the supplied M3 head d3ee08d; reverting the path query to current HEAD would still pass because the changed Go path set is identical.
          round: 19
      boundary: M3
      blocked: true
    - "n": 20
      timestamp: "2026-08-31T17:14:48-07:00"
      agent: codex
      dispose:
        - id: BR-26
          disposition: not-addressed
          note: The typed ledger enumerates declarations but still omits exhaustive dependency locations and validates the plan against the same incomplete path slices; this is the 6th documentation-current-state-accuracy finding.
          round: 20
        - id: BR-27
          disposition: not-addressed
          note: The oracle pins d3ee08d5 rather than the supplied ccba4978 head, and reverting the source-set diff to moving HEAD leaves its test green because both ranges currently have the same Go path membership; this is the 2nd historical-boundary-oracle-pinning finding.
          round: 20
      boundary: M3
      blocked: true
    - "n": 21
      timestamp: "2026-08-31T17:26:34-07:00"
      agent: codex
      dispose:
        - id: BR-26
          disposition: not-addressed
          note: The typed ledger omits the PathOps seam directly used by Couch.ActionableThreadInventory, and its tests cannot discover dependencies omitted from the hand-maintained ledger itself.
          round: 21
        - id: BR-27
          disposition: addressed
          note: The source-set range is fixed and parsed bytes come from the immutable full M3 head object; regressions reject moving HEAD and mutable worktree bytes.
          round: 21
      boundary: M3
      blocked: true
    - "n": 22
      timestamp: "2026-08-31T17:40:18-07:00"
      agent: codex
      dispose:
        - id: BR-21
          disposition: addressed
          note: Production operation projections remain supplied through the shared completion and refresh policy.
          round: 22
        - id: BR-22
          disposition: addressed
          note: The obsolete panel implementation remains deleted in the pinned range.
          round: 22
        - id: BR-23
          disposition: addressed
          note: Target evidence continues to exercise the Console run-loop boundary.
          round: 22
        - id: BR-24
          disposition: addressed
          note: The authoritative M3 implementation steps are checked, leaving only boundary and issue closure open.
          round: 22
        - id: BR-25
          disposition: addressed
          note: Current documentation and the executable root-Escape contract remain aligned.
          round: 22
        - id: BR-26
          disposition: addressed
          note: The Core concepts ledger now includes parked-resume authority and mechanically derived Integration dependencies, with a PathOps-removal mutation test.
          round: 22
        - id: BR-27
          disposition: not-addressed
          note: The new dependency oracle parses its declaration/import/field index from mutable worktree files even though the analyzed declaration comes from the pinned M3 Git object.
          round: 22
      boundary: M3
      blocked: false
    - "n": 23
      timestamp: "2026-08-31T17:51:00-07:00"
      agent: codex
      dispose:
        - id: BR-21
          disposition: addressed
          note: Successful operation projections remain production-reachable and protected by the generation-qualified Console regression.
          round: 23
        - id: BR-22
          disposition: addressed
          note: The superseded flat-panel implementation is deleted and guarded against restoration.
          round: 23
        - id: BR-23
          disposition: addressed
          note: The performance harness measures the Console run-loop from semantic input to correlated emitted frames.
          round: 23
        - id: BR-24
          disposition: addressed
          note: The delivered M3 checklist is checked and protected by a mutation contract.
          round: 23
        - id: BR-25
          disposition: addressed
          note: README, atlas, plan, and executable root-Escape behavior remain aligned.
          round: 23
        - id: BR-26
          disposition: addressed
          note: The Core concepts ledger includes parked-resume authority and mechanically derived Integration dependencies.
          round: 23
        - id: BR-27
          disposition: addressed
          note: Every dependency-index path and byte now comes from one immutable M3 Git archive, and a post-snapshot declaration regression fails if mutable worktree indexing returns.
          round: 23
      boundary: M3
      blocked: false
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

## Round 10 — 2026-08-30T23:14:15-07:00 (codex) — BLOCKED

### Disposed

- BR-18 — addressed — Exact nonzero attempt identity is propagated through every operation effect, origin, and result; the stale-A-after-B regression covers all six operations and fails if attempt matching is removed.

### Raised

- **BR-19** [Critical] `operation-result-origin-correlation` Operation completions can mutate a replacement frame at the same depth
  This is the 4th finding in family operation-result-origin-correlation. menuOperationOriginFrame identifies an origin only by frame kind and depth, so navigation during in-flight work can replace park A's confirmation or name/describe A's draft with frame B at the same position; A's completion then pops B. State the class rule that frame-local completion effects require an exact menu-lifetime frame-instance identity, sweep every operation and outcome, and add mutation-sensitive replacement-frame regressions.

## Round 11 — 2026-08-30T23:24:35-07:00 (codex) — BLOCKED

### Disposed

- BR-19 — addressed — Exact monotonic frame-instance identity now gates frame-local restoration, and the six-operation replacement-frame regression would fail against the prior kind/depth-only implementation.

### Raised

- **BR-20** [Critical] `operation-result-origin-correlation` Operation restoration discards a global start overlay opened after dispatch
  This is the 5th finding in family operation-result-origin-correlation. A park failure slices the stack to the originating action frame, while successful park or resume slices it to root; either path removes a legal global start overlay opened after dispatch, contrary to the Spec. State the class rule that completion may transform its captured origin but must preserve unrelated later frames, then sweep every operation, outcome, and legal post-dispatch navigation with mutation-sensitive regressions.

## Round 12 — 2026-08-31T07:40:08-07:00 (codex) — passed

### Disposed

- BR-20 — addressed — Completion restoration now transforms only its captured prefix and preserves a distinct later global start frame; the all-operation, all-outcome regression fails when that preservation is removed.

## Round 13 — 2026-08-31T15:42:38-07:00 (codex) — BLOCKED

### Raised

- **BR-21** [Critical] `shared-operation-consumer-sweep` Successful operation projections are exercised only in reducer tests and never supplied by production
  This is the 3rd finding in family `shared-operation-consumer-sweep`. Earlier rounds fixed instances. Do NOT fix this instance alone: enumerate every operation result and state the shared rule for projecting or visibly deferring its committed state. `finishOperation` extracts only addresses from park/start results and never sets `InventorySet`; every production search hit for `InventorySet: true` is in tests. Start, park, name, and describe therefore repaint stale inventory before refresh, and a failed refresh can leave that stale view indefinitely rather than using returned state or remaining visibly refresh-pending.
- **BR-22** [Critical] `superseded-ui-authority-retirement` The planned deleted flat-panel authority remains as a parallel implementation
  The Core concepts row declares `PanelModel` deleted in M3 and its narrative says `MenuState` replaces it, but `panel.go`, legacy Console state, resolver/summaries callbacks, prompt state, and the old panel/action controller remain. The executable contract was changed to require this compatibility adapter rather than proving its removal. Delete the superseded authority and make the contract fail while any declared legacy symbols or production fields remain.
- **BR-23** [Important] `lifecycle-evidence-validation` Target latency evidence bypasses the Console run-loop boundary it claims to measure
  This is the 2nd finding in family `lifecycle-evidence-validation`. Earlier rounds fixed instances. Do NOT patch one timing label: state the evidence rule and apply it to every measured path. The harness directly calls `showMenu`, `onMenuInput`, and `finishMenuRefresh`, then times function return; it never sends raw bytes through the host input channel, runs `Console.Run`, or correlates a generation-specific repaint. Delays or misrouting in the actual select loop would not fail these measurements.

## Round 14 — 2026-08-31T16:06:47-07:00 (codex) — BLOCKED

### Disposed

- BR-21 — not-addressed — A successful pre-operation refresh clears ProjectionPending before the dirty post-operation follow-up completes, so committed mutations can still expose stale rows without the required pending marker.
- BR-22 — addressed — The flat-panel source, controller, fields, callbacks, and tests are deleted, and the executable concept contract rejects their return.
- BR-23 — addressed — All target paths now traverse a running Console through raw input, resize, or typed result channels and wait for a uniquely correlated FakeHost frame.

### Raised

- **BR-24** [Important] `documentation-current-state-accuracy` Delivered M3 tasks remain unchecked in the authoritative plan
  This is the 3rd finding in family `documentation-current-state-accuracy`. Earlier rounds fixed instances. Do NOT update only one checkbox: sweep every M3 checklist item against the committed implementation and evidence. Tasks 10, 12, and 13 remain largely or wholly unchecked even though later revisions and the issue log claim delivery; leave only the boundary-close and subsequent issue-close steps open.

## Round 15 — 2026-08-31T16:16:41-07:00 (codex) — BLOCKED

### Disposed

- BR-21 — addressed — Production captures the latest admitted refresh generation for every projection-mutating success, and a Console-level regression proves a pre-mutation result cannot clear pending state while the dirty follow-up remains unresolved.
- BR-24 — not-addressed — The M3 boxes are corrected, but no executable test or contract fails when the delivered Task 10, Task 12, or Task 13 boxes are reverted to unchecked, as this review explicitly requires.

### Raised

- **BR-25** [Critical] `documentation-current-state-accuracy` Authoritative M3 documentation still contradicts the delivered boundary and root Escape contract
  This is the 4th finding in family `documentation-current-state-accuracy`. The Core concepts columns still claim “Current after M3 Task 10” and “Current after M3 Task 11” while containing Task 12 deletion and Task 13 delivery, and README.md:369-370 says root Escape exits Couch without an actor although the Spec and menu.go:392-395 require the root to remain visible with an error. Do not patch only these phrases: state one rule covering current-boundary headings and user-facing key semantics, sweep that class, and make the documentation contract fail on semantic drift.

## Round 16 — 2026-08-31T16:24:54-07:00 (codex) — BLOCKED

### Disposed

- BR-24 — addressed — All 57 M3 steps are checked or left open as required, and a mutation test rejects a delivered step being unchecked.
- BR-25 — addressed — Both tables name the M3 boundary; README root-Escape semantics match the reducer, with executable drift tests.

### Raised

- **BR-26** [Critical] `documentation-current-state-accuracy` Core concepts inventory still omits delivered parked-resume proof authority
  This is the 5th finding in family `documentation-current-state-accuracy`. The table omits `ParkedResumeObservation` and its `NativeBindingResolver`/session-inventory dependency, while adjacent prose still says the projector uses only live-owner observations; one row also gives the non-resolvable location “existing test seams.” The hardcoded contract mirrors this incomplete inventory, so it cannot catch the omission. Do not patch one row: state and enforce the rule that every delivered architectural entity and dependency has exhaustive repo-relative locations and current semantics, then sweep every Core concepts row.

## Round 17 — 2026-08-31T16:33:50-07:00 (codex) — BLOCKED

### Disposed

- BR-26 — not-addressed — This is the 6th finding in family `documentation-current-state-accuracy`: the known rows are fixed, but an unmarked architectural declaration still passes because the source-derived contract scans only opt-in markers and hardcodes six concepts.

## Round 18 — 2026-08-31T16:44:11-07:00 (codex) — BLOCKED

### Disposed

- BR-21 — addressed — The shared generation-qualified projection policy and reachable Console regression remain present and pass.
- BR-22 — addressed — The flat-panel production authority and tests remain deleted, with executable retirement checks.
- BR-23 — addressed — The performance harness continues to drive the running Console and correlate semantic input with emitted frames.
- BR-24 — addressed — The complete M3 checklist state is pinned by a mutation test that rejects delivered work being unchecked.
- BR-25 — addressed — Boundary headings and README root-Escape behavior match the reducer and remain covered by drift tests.
- BR-26 — not-addressed — The digest inventories declaration names but omits their architectural/detail disposition; removing a concept marker leaves the digest unchanged and the marker validator accepts any remaining count, so source classification can contradict the plan without failing.

### Raised

- **BR-27** [Important] `historical-boundary-oracle-pinning` The M3 declaration oracle reads an unpinned diff and mutable worktree bytes
  The source-set test runs git diff from the M2 base to the repository's current HEAD, while the digest parses current filesystem files. Any later Go change will rewrite this supposedly historical M3 boundary or fail unrelated CI. Pin both paths and bytes to the supplied M3 head object.

## Round 19 — 2026-08-31T16:55:47-07:00 (codex) — BLOCKED

### Disposed

- BR-21 — addressed — The generation-qualified projection policy remains production-reachable, and its Console regression fails if a pre-mutation refresh is allowed to clear pending state.
- BR-22 — addressed — The flat-panel implementation is deleted, and executable source scans reject restoration of PanelModel or its parallel authority.
- BR-23 — addressed — The target harness drives Console.Run through semantic input and correlated emitted frames; the complete M2 Max protocol passed in this review.
- BR-24 — addressed — The complete M3 checklist is pinned, including a mutation that rejects reverting a delivered step to unchecked.
- BR-25 — addressed — The plan headings and README root-Escape behavior match the reducer and remain protected by executable drift tests.
- BR-26 — not-addressed — The closed ledger calls unmarked declarations detail, yet RefreshSchedule and AdvanceRefreshSchedule are authoritative Core concepts without architectural markers; the validator compares only its hardcoded six-entry subset.
- BR-27 — not-addressed — Both source paths and bytes are pinned to 7ff7d8c4, not the supplied M3 head d3ee08d; reverting the path query to current HEAD would still pass because the changed Go path set is identical.

## Round 20 — 2026-08-31T17:14:48-07:00 (codex) — BLOCKED

### Disposed

- BR-26 — not-addressed — The typed ledger enumerates declarations but still omits exhaustive dependency locations and validates the plan against the same incomplete path slices; this is the 6th documentation-current-state-accuracy finding.
- BR-27 — not-addressed — The oracle pins d3ee08d5 rather than the supplied ccba4978 head, and reverting the source-set diff to moving HEAD leaves its test green because both ranges currently have the same Go path membership; this is the 2nd historical-boundary-oracle-pinning finding.

## Round 21 — 2026-08-31T17:26:34-07:00 (codex) — BLOCKED

### Disposed

- BR-26 — not-addressed — The typed ledger omits the PathOps seam directly used by Couch.ActionableThreadInventory, and its tests cannot discover dependencies omitted from the hand-maintained ledger itself.
- BR-27 — addressed — The source-set range is fixed and parsed bytes come from the immutable full M3 head object; regressions reject moving HEAD and mutable worktree bytes.

## Round 22 — 2026-08-31T17:40:18-07:00 (codex) — passed

### Disposed

- BR-21 — addressed — Production operation projections remain supplied through the shared completion and refresh policy.
- BR-22 — addressed — The obsolete panel implementation remains deleted in the pinned range.
- BR-23 — addressed — Target evidence continues to exercise the Console run-loop boundary.
- BR-24 — addressed — The authoritative M3 implementation steps are checked, leaving only boundary and issue closure open.
- BR-25 — addressed — Current documentation and the executable root-Escape contract remain aligned.
- BR-26 — addressed — The Core concepts ledger now includes parked-resume authority and mechanically derived Integration dependencies, with a PathOps-removal mutation test.
- BR-27 — not-addressed — The new dependency oracle parses its declaration/import/field index from mutable worktree files even though the analyzed declaration comes from the pinned M3 Git object.

## Round 23 — 2026-08-31T17:51:00-07:00 (codex) — passed

### Disposed

- BR-21 — addressed — Successful operation projections remain production-reachable and protected by the generation-qualified Console regression.
- BR-22 — addressed — The superseded flat-panel implementation is deleted and guarded against restoration.
- BR-23 — addressed — The performance harness measures the Console run-loop from semantic input to correlated emitted frames.
- BR-24 — addressed — The delivered M3 checklist is checked and protected by a mutation contract.
- BR-25 — addressed — README, atlas, plan, and executable root-Escape behavior remain aligned.
- BR-26 — addressed — The Core concepts ledger includes parked-resume authority and mechanically derived Integration dependencies.
- BR-27 — addressed — Every dependency-index path and byte now comes from one immutable M3 Git archive, and a post-snapshot declaration regression fails if mutable worktree indexing returns.

## Open findings

(none — every finding has been disposed)
