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

## Open findings

- **BR-1** [Minor] `test-prose-enumeration` Tasks 1.3/1.5/1.6/2.2/2.4 enumerate test cases in prose instead of one strategy line per risky function
- **BR-2** [Minor] `unconfirmed-outcome-untested` The :0 reboot claim leak (death between Claim and journal) is asserted in prose but not pinned by a test
- **BR-3** [Minor] `single-action-authority` Busy-row "starting elsewhere" is patched at menu.go case "" rather than derived from the action-table busy phase
- **BR-4** [Minor] `refusal-names-unoffered-action` withRebootAdvice tells the operator "Tab → reboot" while M1 declares reboot RowAction false
- **BR-5** [Minor] `refusal-names-unoffered-action` OpenSlot refusals dropped the "choose Start fresh" exit while fresh-slot is still offered in M1
- **BR-6** [Minor] `result-consumer-keyed-by-op-name` Console continuation watch registers only for recover-thread/recover-checkpoint, not routed resume
- **BR-7** [Minor] `result-field-dropped` rebootSlot ignores prepareRetirement's SessionNotStopped
- **BR-8** [Minor] `undeclared-runtime-cost` Routed resume adds a classifyForAction round per ordinary resume, not noted in ARCH-CONSTRAINTS
