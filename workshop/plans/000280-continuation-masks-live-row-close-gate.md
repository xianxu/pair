---
gate: boundary-review
issue: 280
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-17T22:38:57-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Failed-row action composition filters only continuationGuard ops, but archive has its own continuation admission (archiveContinuationVacant)
          detail: |-
            2nd finding in this family. Composing "state actions minus ContinuationRefuses" newly offers archive on unusable Failed rows with no Recovery (menu.go:1255-1258), which archiveContinuationVacant (detach.go:430-450) may refuse, and TestContinuationFailedRowKeepsAnExplicitRetry (console_continuation_test.go:21), the test reused for PQ-2, asserts archive is absent there. Rule: an action is offered on a row with a retained request only if EVERY admission that reads record.Continuation allows it (continuationGuard sites, archiveContinuationVacant, validateContinuationWarm). The ContinuationRefuses table test should loop over every declared RowAction op, not a hand list, or composition should apply to live rows only.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: unbacked-existing-behavior-claim
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-17T22:38:57-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Critical
          title: Core concepts table lists DismissFailedContinuation and Couch.DismissContinuation as PURE, but both do store IO
          detail: Plan lines 36-37. Both read and write the file-backed thread store, and their tests need a temp-dir store (mutable filesystem). The dismissal rule is an inline closure no test can reach without IO. Add a Revisions entry moving both rows to Integration points, or extract the pure rule and name it (lessons.md, "PURE fixtures must be literal at their direct boundary").
          family: pure-row-must-be-io-free
          round: 2
        - id: BR-3
          severity: Important
          title: The stated reason Pending/Running may keep displacing the state (bounded in time) does not hold without a watching owner
          detail: The 30s deadline (continuation_recovery.go:196) starts only once Target is set, and runs only while the console scans the address (panes plus watches). A Running or Pending request left by a dead owner on a thread with no pane reads "continuing…" forever. Narrow the claim in menu_render.go:419, menu.go:1259, atlas/couch.md:163 and the issue Revisions, or compose these phases on rows that are not live.
          family: unbacked-existing-behavior-claim
          round: 2
        - id: BR-4
          severity: Important
          title: '"Every refusal of a failed request names both exits" is false for recovery, archive and warm-reattach refusals'
          detail: recovery_execute.go:179 and :266, archiveContinuationVacant (detach.go:436,447) and the validateContinuationWarm refusal (resume.go:479) are caused by a retained failed request but never mention dismiss. Route them through continuationExits(phase), or narrow the atlas (couch.md:160) and issue Log claim.
          family: refusal-names-every-exit
          round: 2
        - id: BR-5
          severity: Minor
          title: menuLiveActions was inserted between menuActionItems' doc comment and the function
          detail: menu.go:1205-1214. The doc now attaches to the var; move the var above the comment block.
          family: doc-comment-attachment
          round: 2
        - id: BR-6
          severity: Minor
          title: Exits for each phase are written twice (continuationExits and continuationExitsFor), and the stale-revision loop is copied from advanceContinuation
          detail: continuation.go:378-389 plus runcli.go:147. DismissContinuation's 8-try loop duplicates advanceContinuation (continuation.go:154-167); a shared retry-on-stale helper would serve both.
          family: single-source-per-fact
          round: 2
        - id: BR-7
          severity: Minor
          title: The state-text test lists continuation phases by hand, with no checkpoint.AllPhases()
          family: vocabulary-enumerated-by-test
          round: 2
        - id: BR-8
          severity: Minor
          title: The continuationArguments(bootstrap) parameter now also serves dismiss, which is not a bootstrap
          family: naming-matches-role
          round: 2
        - id: BR-9
          severity: Minor
          title: menu.Orientation survives a dismissal, so Copy orientation prompt stays on offer for a dropped handoff
          family: in-memory-state-follows-record
          round: 2
        - id: BR-10
          severity: Minor
          title: ContinuationRefuses' doc says a table test proves the list, but switch-agent, resume and start are not driven
          family: doc-claims-match-test-reach
          round: 2
      blocked: true
---

# Gate ledger — pair#280 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T22:38:57-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `unbacked-existing-behavior-claim` Failed-row action composition filters only continuationGuard ops, but archive has its own continuation admission (archiveContinuationVacant)
  2nd finding in this family. Composing "state actions minus ContinuationRefuses" newly offers archive on unusable Failed rows with no Recovery (menu.go:1255-1258), which archiveContinuationVacant (detach.go:430-450) may refuse, and TestContinuationFailedRowKeepsAnExplicitRetry (console_continuation_test.go:21), the test reused for PQ-2, asserts archive is absent there. Rule: an action is offered on a row with a retained request only if EVERY admission that reads record.Continuation allows it (continuationGuard sites, archiveContinuationVacant, validateContinuationWarm). The ContinuationRefuses table test should loop over every declared RowAction op, not a hand list, or composition should apply to live rows only.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-17T22:38:57-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Critical] `pure-row-must-be-io-free` Core concepts table lists DismissFailedContinuation and Couch.DismissContinuation as PURE, but both do store IO
  Plan lines 36-37. Both read and write the file-backed thread store, and their tests need a temp-dir store (mutable filesystem). The dismissal rule is an inline closure no test can reach without IO. Add a Revisions entry moving both rows to Integration points, or extract the pure rule and name it (lessons.md, "PURE fixtures must be literal at their direct boundary").
- **BR-3** [Important] `unbacked-existing-behavior-claim` The stated reason Pending/Running may keep displacing the state (bounded in time) does not hold without a watching owner
  The 30s deadline (continuation_recovery.go:196) starts only once Target is set, and runs only while the console scans the address (panes plus watches). A Running or Pending request left by a dead owner on a thread with no pane reads "continuing…" forever. Narrow the claim in menu_render.go:419, menu.go:1259, atlas/couch.md:163 and the issue Revisions, or compose these phases on rows that are not live.
- **BR-4** [Important] `refusal-names-every-exit` "Every refusal of a failed request names both exits" is false for recovery, archive and warm-reattach refusals
  recovery_execute.go:179 and :266, archiveContinuationVacant (detach.go:436,447) and the validateContinuationWarm refusal (resume.go:479) are caused by a retained failed request but never mention dismiss. Route them through continuationExits(phase), or narrow the atlas (couch.md:160) and issue Log claim.
- **BR-5** [Minor] `doc-comment-attachment` menuLiveActions was inserted between menuActionItems' doc comment and the function
  menu.go:1205-1214. The doc now attaches to the var; move the var above the comment block.
- **BR-6** [Minor] `single-source-per-fact` Exits for each phase are written twice (continuationExits and continuationExitsFor), and the stale-revision loop is copied from advanceContinuation
  continuation.go:378-389 plus runcli.go:147. DismissContinuation's 8-try loop duplicates advanceContinuation (continuation.go:154-167); a shared retry-on-stale helper would serve both.
- **BR-7** [Minor] `vocabulary-enumerated-by-test` The state-text test lists continuation phases by hand, with no checkpoint.AllPhases()
- **BR-8** [Minor] `naming-matches-role` The continuationArguments(bootstrap) parameter now also serves dismiss, which is not a bootstrap
- **BR-9** [Minor] `in-memory-state-follows-record` menu.Orientation survives a dismissal, so Copy orientation prompt stays on offer for a dropped handoff
- **BR-10** [Minor] `doc-claims-match-test-reach` ContinuationRefuses' doc says a table test proves the list, but switch-agent, resume and start are not driven

## Open findings

- **BR-1** [Minor] `unbacked-existing-behavior-claim` Failed-row action composition filters only continuationGuard ops, but archive has its own continuation admission (archiveContinuationVacant)
- **BR-2** [Critical] `pure-row-must-be-io-free` Core concepts table lists DismissFailedContinuation and Couch.DismissContinuation as PURE, but both do store IO
- **BR-3** [Important] `unbacked-existing-behavior-claim` The stated reason Pending/Running may keep displacing the state (bounded in time) does not hold without a watching owner
- **BR-4** [Important] `refusal-names-every-exit` "Every refusal of a failed request names both exits" is false for recovery, archive and warm-reattach refusals
- **BR-5** [Minor] `doc-comment-attachment` menuLiveActions was inserted between menuActionItems' doc comment and the function
- **BR-6** [Minor] `single-source-per-fact` Exits for each phase are written twice (continuationExits and continuationExitsFor), and the stale-revision loop is copied from advanceContinuation
- **BR-7** [Minor] `vocabulary-enumerated-by-test` The state-text test lists continuation phases by hand, with no checkpoint.AllPhases()
- **BR-8** [Minor] `naming-matches-role` The continuationArguments(bootstrap) parameter now also serves dismiss, which is not a bootstrap
- **BR-9** [Minor] `in-memory-state-follows-record` menu.Orientation survives a dismissal, so Copy orientation prompt stays on offer for a dropped handoff
- **BR-10** [Minor] `doc-claims-match-test-reach` ContinuationRefuses' doc says a table test proves the list, but switch-agent, resume and start are not driven
