---
gate: boundary-review
issue: 363
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-02T23:43:23-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tasks 1.3/1.5/1.6/2.2/2.4 enumerate test cases in prose instead of one strategy line per risky function
          detail: |-
            Keep the derived-domain tables (DecideReboot, everyMenuRowShape, ResumeRebootAdvice parse); compress the named per-case bullets to the adversarial class + guard.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: test-prose-enumeration
          round: 1
        - id: BR-2
          severity: Minor
          title: The :0 reboot claim leak (death between Claim and journal) is asserted in prose but not pinned by a test
          detail: |-
            Add an assertion that a later start/reboot in the same path succeeds with a leaked start claim present, so "same window as AllocateThreadTag" stays an invariant.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: unconfirmed-outcome-untested
          round: 1
        - id: BR-3
          severity: Minor
          title: Busy-row "starting elsewhere" is patched at menu.go case "" rather than derived from the action-table busy phase
          detail: |-
            Have the root status and Enter notice read the menuPhaseBusy fact so the per-row authority stays single-sourced.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: single-action-authority
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-02T23:43:23-07:00"
      agent: claude
      findings:
        - id: BR-4
          severity: Minor
          title: withRebootAdvice tells the operator "Tab → reboot" while M1 declares reboot RowAction false
          detail: resume_route.go:89. Contradicts "each milestone leaves main releasable"; M2 Task 2.6 rewords it, or the M1 text should not name a missing switcher action.
          family: refusal-names-unoffered-action
          round: 2
        - id: BR-5
          severity: Minor
          title: OpenSlot refusals dropped the "choose Start fresh" exit while fresh-slot is still offered in M1
          detail: slotrecovery.go:494-500,530. The switcher's open-slot calls OpenSlot directly (no withRebootAdvice), so the M1 refusal names no next action.
          family: refusal-names-unoffered-action
          round: 2
        - id: BR-6
          severity: Minor
          title: Console continuation watch registers only for recover-thread/recover-checkpoint, not routed resume
          detail: console_continuation.go:230. resume can now return ContinuationResult from RecoverThread/RetryContinuation. The plan's Task 2.6 says these sites "become reboot"; they must become resume.
          family: result-consumer-keyed-by-op-name
          round: 2
        - id: BR-7
          severity: Minor
          title: rebootSlot ignores prepareRetirement's SessionNotStopped
          detail: reboot.go:241. Always false for a readable slot record today, but the RebootResult contract is silently narrower for slots.
          family: result-field-dropped
          round: 2
        - id: BR-8
          severity: Minor
          title: Routed resume adds a classifyForAction round per ordinary resume, not noted in ARCH-CONSTRAINTS
          detail: resume_route.go:162. One host-wide list-sessions per keypress, off the UI path; the plan Revision should state it, as it does for reboot.
          family: undeclared-runtime-cost
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 3
      timestamp: "2026-10-03T01:17:00-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: withdrawn
          note: 'Overtaken: tasks 1.x and 2.x are executed and their tests exist; compressing completed prose now has no value.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: TestRebootPrimaryCrashAfterJournalRecovers installs leakFirstClaim and asserts the next reboot starts past the leaked address (reboot_test.go:107-170).
          round: 3
        - id: BR-3
          disposition: addressed
          note: menuRowNotice reads menuPhaseBusy for both threadStateText and enterRefusalNotice; pinned by TestBusyRowSaysStartingElsewhere.
          round: 3
      findings:
        - id: BR-9
          severity: Minor
          title: A :0 path-missing row and its reboot confirmation say "add slot recreates it", which that row cannot reach
          detail: '3rd in family. Rule: a row''s next-step text must name an action reachable from that row''s kind (slot vs primary), chosen per kind next to the action table. Here a :0 never offers add-slot unless live, and add slot does not recreate a primary checkout (reboot_decision.go:47, menu_actions.go:170, menu.go:1316). Confirm the :0 wording with the operator.'
          family: refusal-names-unoffered-action
          round: 3
        - id: BR-10
          severity: Minor
          title: menu.go:1636 comment still says relaunch and archive keep their confirmation on failure
          detail: Archive was removed; reboot is now the action that keeps its confirmation. The Task 2.6 sweep grepped strings, not comments.
          family: removed-action-sweep-misses-comments
          round: 3
      boundary: M2
      recipe: milestone-review
      blocked: false
    - "n": 4
      timestamp: "2026-10-03T02:00:44-07:00"
      agent: claude
      findings:
        - id: BR-11
          severity: Minor
          title: The slot DirectoryMissing RebootCost text cannot appear, because that row offers no reboot
          detail: menuRowActions returns nil for a :1+ row whose directory is missing, so no reboot confirmation ever shows it. Remove the field value or comment why it is kept.
          family: dead-advice-text
          round: 4
        - id: BR-12
          severity: Minor
          title: A rolled-back :0 reboot refuses when a pre-363 co-tenant primary exists in the scope
          detail: reboot.go rolled-back branch calls spawnResolved, and the widened guard sees the other primary. This follows the rule and stops nothing, but it is untested and unmentioned.
          family: legacy-cotenant-guard-interaction
          round: 4
      boundary: M3
      recipe: milestone-review
      blocked: false
    - "n": 5
      timestamp: "2026-10-03T12:16:05-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: Reboot is a live action in the switcher now (M2/M3), so the "Tab → reboot" advice at resume_route.go:94 names an action the operator can reach.
          round: 5
        - id: BR-5
          disposition: addressed
          note: fresh-slot is gone; OpenSlot is reached only through slotstart.go:270 and resume_route.go:123, both of which wrap the error with withRebootAdvice.
          round: 5
        - id: BR-6
          disposition: addressed
          note: console_continuation.go:232 now registers the watch for the "resume" operation's ContinuationResult.
          round: 5
        - id: BR-7
          disposition: addressed
          note: rebootSlot and the :0 path both build their result through retiredResult (reboot.go:195), which carries SessionNotStopped.
          round: 5
        - id: BR-8
          disposition: addressed
          note: Plan Revision at line 972 states the extra classifyForAction round per ordinary resume.
          round: 5
        - id: BR-9
          disposition: addressed
          note: A :0 row now says RebootCheckoutMissing ("restore the checkout"); the add-slot text is used only for slot rows (menu_actions.go:189-198), and the advice test enforces reachability.
          round: 5
        - id: BR-10
          disposition: addressed
          note: menu.go:1635 comment now says relaunch and reboot keep their confirmation.
          round: 5
        - id: BR-11
          disposition: addressed
          note: Field removed in c8e5d395; menu_actions_test.go:451 now fails on any row that has reboot-confirmation text but offers no reboot.
          round: 5
        - id: BR-12
          disposition: addressed
          note: TestRebootOfARolledBackStartRefusesBesideALegacyPrimary pins the refusal, that nothing starts, and that the legacy primary stays live; ran green.
          round: 5
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#363 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-02T23:43:23-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-prose-enumeration` Tasks 1.3/1.5/1.6/2.2/2.4 enumerate test cases in prose instead of one strategy line per risky function
  Keep the derived-domain tables (DecideReboot, everyMenuRowShape, ResumeRebootAdvice parse); compress the named per-case bullets to the adversarial class + guard.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `unconfirmed-outcome-untested` The :0 reboot claim leak (death between Claim and journal) is asserted in prose but not pinned by a test
  Add an assertion that a later start/reboot in the same path succeeds with a leaked start claim present, so "same window as AllocateThreadTag" stays an invariant.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Minor] `single-action-authority` Busy-row "starting elsewhere" is patched at menu.go case "" rather than derived from the action-table busy phase
  Have the root status and Enter notice read the menuPhaseBusy fact so the per-row authority stays single-sourced.
  (carried from plan-quality PQ-3, deferred to the boundary review)

## Round 2 — 2026-10-02T23:43:23-07:00 (claude) — passed

### Raised

- **BR-4** [Minor] `refusal-names-unoffered-action` withRebootAdvice tells the operator "Tab → reboot" while M1 declares reboot RowAction false
  resume_route.go:89. Contradicts "each milestone leaves main releasable"; M2 Task 2.6 rewords it, or the M1 text should not name a missing switcher action.
- **BR-5** [Minor] `refusal-names-unoffered-action` OpenSlot refusals dropped the "choose Start fresh" exit while fresh-slot is still offered in M1
  slotrecovery.go:494-500,530. The switcher's open-slot calls OpenSlot directly (no withRebootAdvice), so the M1 refusal names no next action.
- **BR-6** [Minor] `result-consumer-keyed-by-op-name` Console continuation watch registers only for recover-thread/recover-checkpoint, not routed resume
  console_continuation.go:230. resume can now return ContinuationResult from RecoverThread/RetryContinuation. The plan's Task 2.6 says these sites "become reboot"; they must become resume.
- **BR-7** [Minor] `result-field-dropped` rebootSlot ignores prepareRetirement's SessionNotStopped
  reboot.go:241. Always false for a readable slot record today, but the RebootResult contract is silently narrower for slots.
- **BR-8** [Minor] `undeclared-runtime-cost` Routed resume adds a classifyForAction round per ordinary resume, not noted in ARCH-CONSTRAINTS
  resume_route.go:162. One host-wide list-sessions per keypress, off the UI path; the plan Revision should state it, as it does for reboot.

## Round 3 — 2026-10-03T01:17:00-07:00 (claude) — passed

### Disposed

- BR-1 — withdrawn — Overtaken: tasks 1.x and 2.x are executed and their tests exist; compressing completed prose now has no value.
- BR-2 — addressed — TestRebootPrimaryCrashAfterJournalRecovers installs leakFirstClaim and asserts the next reboot starts past the leaked address (reboot_test.go:107-170).
- BR-3 — addressed — menuRowNotice reads menuPhaseBusy for both threadStateText and enterRefusalNotice; pinned by TestBusyRowSaysStartingElsewhere.

### Raised

- **BR-9** [Minor] `refusal-names-unoffered-action` A :0 path-missing row and its reboot confirmation say "add slot recreates it", which that row cannot reach
  3rd in family. Rule: a row's next-step text must name an action reachable from that row's kind (slot vs primary), chosen per kind next to the action table. Here a :0 never offers add-slot unless live, and add slot does not recreate a primary checkout (reboot_decision.go:47, menu_actions.go:170, menu.go:1316). Confirm the :0 wording with the operator.
- **BR-10** [Minor] `removed-action-sweep-misses-comments` menu.go:1636 comment still says relaunch and archive keep their confirmation on failure
  Archive was removed; reboot is now the action that keeps its confirmation. The Task 2.6 sweep grepped strings, not comments.

## Round 4 — 2026-10-03T02:00:44-07:00 (claude) — passed

### Raised

- **BR-11** [Minor] `dead-advice-text` The slot DirectoryMissing RebootCost text cannot appear, because that row offers no reboot
  menuRowActions returns nil for a :1+ row whose directory is missing, so no reboot confirmation ever shows it. Remove the field value or comment why it is kept.
- **BR-12** [Minor] `legacy-cotenant-guard-interaction` A rolled-back :0 reboot refuses when a pre-363 co-tenant primary exists in the scope
  reboot.go rolled-back branch calls spawnResolved, and the widened guard sees the other primary. This follows the rule and stops nothing, but it is untested and unmentioned.

## Round 5 — 2026-10-03T12:16:05-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — Reboot is a live action in the switcher now (M2/M3), so the "Tab → reboot" advice at resume_route.go:94 names an action the operator can reach.
- BR-5 — addressed — fresh-slot is gone; OpenSlot is reached only through slotstart.go:270 and resume_route.go:123, both of which wrap the error with withRebootAdvice.
- BR-6 — addressed — console_continuation.go:232 now registers the watch for the "resume" operation's ContinuationResult.
- BR-7 — addressed — rebootSlot and the :0 path both build their result through retiredResult (reboot.go:195), which carries SessionNotStopped.
- BR-8 — addressed — Plan Revision at line 972 states the extra classifyForAction round per ordinary resume.
- BR-9 — addressed — A :0 row now says RebootCheckoutMissing ("restore the checkout"); the add-slot text is used only for slot rows (menu_actions.go:189-198), and the advice test enforces reachability.
- BR-10 — addressed — menu.go:1635 comment now says relaunch and reboot keep their confirmation.
- BR-11 — addressed — Field removed in c8e5d395; menu_actions_test.go:451 now fails on any row that has reboot-confirmation text but offers no reboot.
- BR-12 — addressed — TestRebootOfARolledBackStartRefusesBesideALegacyPrimary pins the refusal, that nothing starts, and that the legacy primary stays live; ran green.

## Open findings

(none — every finding has been disposed)
