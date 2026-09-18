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
    - "n": 3
      timestamp: "2026-09-17T23:01:19-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Composition is live-only (menu.go:1238) and menuLiveActions holds no archive; the guard-agreement test loops every declared RowAction (driven or exempt with a reason).
          round: 3
        - id: BR-2
          disposition: addressed
          note: Pure table now lists CheckDismissible (literal-request tests in request_dismiss_test.go); both IO entries sit in Integration points (plan lines 112-118).
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: Code, issue and atlas:158 fixed, but plan:83 still asserts "Both are bounded in time... Pending is picked up by the owner's scan", and atlas:165 says an in-flight request "keeps the restricted retry set" while a Pending row offers only name/describe (menu.go:1265, pinned by TestFailedContinuationComposesWithALiveRowsActions).
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: Seven sites wrapped, but recovery_execute.go:268 (AdmitRecoveryGeneration "...inspect or archive"), inside the same retained-request block, is not. Wrap at the block's return rather than per site.
          round: 3
        - id: BR-5
          disposition: addressed
          note: menuLiveActions now sits above menuActionItems' doc comment (menu.go:1205-1208).
          round: 3
        - id: BR-6
          disposition: addressed
          note: One checkpoint.Exits; writeRequestRecord serves advanceContinuation and DismissContinuation; the duplicate helpers are gone.
          round: 3
        - id: BR-7
          disposition: addressed
          note: The state-text test iterates checkpoint.AllPhases; sibling hand lists raised separately.
          round: 3
        - id: BR-8
          disposition: addressed
          note: Renamed operatorFacing; no bootstrap references remain.
          round: 3
        - id: BR-9
          disposition: addressed
          note: 'Dismissal now prunes the prompt, but the fix over-prunes: see the new Critical in the same family.'
          round: 3
        - id: BR-10
          disposition: addressed
          note: The doc names what is driven; SwitchAgent re-runs PrepareAgentSwitch (switchagent.go:256), which holds the guard.
          round: 3
      findings:
        - id: BR-11
          severity: Critical
          title: The continuation scan now deletes switch-agent's Copy orientation prompt within 500 ms (console_continuation.go:169)
          detail: 'This is the 2nd finding in family in-memory-state-follows-record. continuationAddresses() covers every pane, and ContinuationRequests returns no status for a record without a request, so line 169 deletes menu.Orientation for every hosted thread on each tick. That includes the entry watchOrientation writes (console_switchagent.go:37) and that finishOrientation deliberately keeps after an unconfirmed delivery, whose notice says "Copy orientation prompt is available in actions". Reproduced on HEAD in a scratch copy: the test fails, and passes once line 169 is removed. Rule (ARCH-ORDER): console state that mirrors a record fact carries the identity of the fact that produced it, and is pruned only when THAT fact vanishes. menu.Orientation has two producers and no provenance. Record the producer (for example, the continuation request ID) and let the scan prune only continuation-produced entries. Sweep every prune site under that rule: line 169 (new); line 117 (pre-existing, same defect for a record that retains a Complete request); line 222. Add a regression test with a switch-agent prompt that survives a scan tick.'
          family: in-memory-state-follows-record
          round: 3
        - id: BR-12
          severity: Important
          title: 'The claim that each exits-wrapped refusal site is driven is false: 3 of 7 sites are reached'
          detail: 'This is the 2nd finding in family doc-claims-match-test-reach. The test comment (continuation_guard_test.go:165), the plan''s BR-4 revision and the issue Log all claim every withContinuationExits site is driven. The test reaches only recovery_execute.go:179, detach.go:436 and continuation_recovery.go:34. Four sites are never reached: recovery_execute.go:266, :272 and :278, and detach.go:447. Rule: a claim of test reach over a set of sites is checked by the test against that set, or it names exactly the subset driven. Here, a table with one row per wrapped site, plus a source scan asserting that the count of withContinuationExits( call sites equals the table''s rows. The guard-agreement test''s "refused BY THE GUARD" has the same problem: its oracle phrase is shared with withContinuationExits. Match the guard''s own "continuation <id> is failed; " prefix.'
          family: doc-claims-match-test-reach
          round: 3
        - id: BR-13
          severity: Minor
          title: Continuation phases are still written out by hand in menu_action_sweep_test.go:88 and checkpoint/recovery_request_test.go:61
          detail: 'This is the 2nd finding in family vocabulary-enumerated-by-test. Rule: once a vocabulary has an All*() enumerator, no test restates it. Replace both lists with checkpoint.AllPhases(), and add a check that a []Phase{ literal appears only in AllPhases.'
          family: vocabulary-enumerated-by-test
          round: 3
        - id: BR-14
          severity: Minor
          title: pair continue --retry on a hosted thread names both of Failed's exits regardless of the request's phase (runcli.go:147)
          detail: 'This is the 2nd finding in family refusal-names-every-exit. Rule: a refusal names exactly the exits valid for the request''s actual phase, through checkpoint.Exits(phase, tag). A site that does not know the phase uses phase-neutral wording rather than assuming Failed. Here, dismiss is offered even when the hosted request may be pending or running, which CheckDismissible refuses.'
          family: refusal-names-every-exit
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-17T23:18:52-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: addressed
          note: Plan:83 now states the "bounded in time" claim is false; atlas:155-165 and menu.go:1258-1263 describe the per-phase sets as statements of which actions are offered, not time bounds, matching menuActionItems.
          round: 4
        - id: BR-4
          disposition: addressed
          note: All four non-guard checks wrap once where they return (RecoverThread:179, prepareAbsentContinuation:291 via admitRetainedRecovery, archiveContinuationVacant, validateContinuationWarm); unwrapping :291 in a scratch copy turns the row and the site scan red.
          round: 4
        - id: BR-11
          disposition: addressed
          note: Each prompt records its producer; the round-3 unconditional prune and a producer-blind prune each turn TestSwitchAgentOrientationPromptSurvivesContinuationScans red. A replacement-sequence gap in the same family is raised separately.
          round: 4
        - id: BR-12
          disposition: addressed
          note: One row per withContinuationExits call site (4), each driven, plus a productionCallsTo scan that the call sites equal the rows; the guard oracle matches the guard's own "continuation <id> is failed; " prefix and requires an unchanged revision.
          round: 4
        - id: BR-13
          disposition: addressed
          note: Both lists use AllPhases; restating the list in menu_action_sweep_test.go:88 turns TestPhaseListIsWrittenOnlyInAllPhases red.
          round: 4
        - id: BR-14
          disposition: addressed
          note: runcli.go:147 uses Exits("", tag), whose conditional dismiss wording TestExitsWithAnUnknownPhaseAreConditional pins; the call site itself is not pinned (acceptable for a message-only Minor).
          round: 4
      findings:
        - id: BR-15
          severity: Minor
          title: A request replaced before the scan sees it vanish strands its orientation prompt (console_continuation.go:145-147)
          detail: 'This is the 3rd finding in family in-memory-state-follows-record. When the scan sees a new request ID it replaces the watch without dropping the old request''s continuation-produced prompt; every later prune compares against the new producer, so Copy orientation prompt stays offered until a switch-agent launch or restart. Reproduced (A''s prompt, B seen, B complete, B vanished: prompt survives); base cleared it on any completion for the address. Window: A vanishes and B appears within one scan interval, or while a partial scan error disables the vanish loop. Rule: a view of a record fact lives inside, or is derived from, the object that tracks that fact''s identity, not pruned at an enumerated list of events (this family''s three findings are three missed events: dismissal, a scan with no request, replacement). Fix: keep the continuation''s prompt on continuationWatch, or reconcile continuation-produced entries against the current watch''s request ID in one place after every change to c.continuations. Add a replacement-sequence test.'
          family: in-memory-state-follows-record
          round: 4
        - id: BR-16
          severity: Minor
          title: admitRetainedRecovery was inserted under prepareAbsentContinuation's doc comment (recovery_execute.go:224-230)
          detail: 'This is the 2nd finding in family doc-comment-attachment. The combined comment starts "prepareAbsentContinuation snapshots..." but attaches to admitRetainedRecovery, and prepareAbsentContinuation has no doc. Rule: a Go doc comment begins with the name of the declaration it documents; enforce it with a package-level scan that fails when a function''s doc starts with the name of another declaration in the same file. Prevalence: 1 new instance in this diff (BR-5 was the first in the family).'
          family: doc-comment-attachment
          round: 4
      blocked: false
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

## Round 3 — 2026-09-17T23:01:19-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Composition is live-only (menu.go:1238) and menuLiveActions holds no archive; the guard-agreement test loops every declared RowAction (driven or exempt with a reason).
- BR-2 — addressed — Pure table now lists CheckDismissible (literal-request tests in request_dismiss_test.go); both IO entries sit in Integration points (plan lines 112-118).
- BR-3 — not-addressed — Code, issue and atlas:158 fixed, but plan:83 still asserts "Both are bounded in time... Pending is picked up by the owner's scan", and atlas:165 says an in-flight request "keeps the restricted retry set" while a Pending row offers only name/describe (menu.go:1265, pinned by TestFailedContinuationComposesWithALiveRowsActions).
- BR-4 — not-addressed — Seven sites wrapped, but recovery_execute.go:268 (AdmitRecoveryGeneration "...inspect or archive"), inside the same retained-request block, is not. Wrap at the block's return rather than per site.
- BR-5 — addressed — menuLiveActions now sits above menuActionItems' doc comment (menu.go:1205-1208).
- BR-6 — addressed — One checkpoint.Exits; writeRequestRecord serves advanceContinuation and DismissContinuation; the duplicate helpers are gone.
- BR-7 — addressed — The state-text test iterates checkpoint.AllPhases; sibling hand lists raised separately.
- BR-8 — addressed — Renamed operatorFacing; no bootstrap references remain.
- BR-9 — addressed — Dismissal now prunes the prompt, but the fix over-prunes: see the new Critical in the same family.
- BR-10 — addressed — The doc names what is driven; SwitchAgent re-runs PrepareAgentSwitch (switchagent.go:256), which holds the guard.

### Raised

- **BR-11** [Critical] `in-memory-state-follows-record` The continuation scan now deletes switch-agent's Copy orientation prompt within 500 ms (console_continuation.go:169)
  This is the 2nd finding in family in-memory-state-follows-record. continuationAddresses() covers every pane, and ContinuationRequests returns no status for a record without a request, so line 169 deletes menu.Orientation for every hosted thread on each tick. That includes the entry watchOrientation writes (console_switchagent.go:37) and that finishOrientation deliberately keeps after an unconfirmed delivery, whose notice says "Copy orientation prompt is available in actions". Reproduced on HEAD in a scratch copy: the test fails, and passes once line 169 is removed. Rule (ARCH-ORDER): console state that mirrors a record fact carries the identity of the fact that produced it, and is pruned only when THAT fact vanishes. menu.Orientation has two producers and no provenance. Record the producer (for example, the continuation request ID) and let the scan prune only continuation-produced entries. Sweep every prune site under that rule: line 169 (new); line 117 (pre-existing, same defect for a record that retains a Complete request); line 222. Add a regression test with a switch-agent prompt that survives a scan tick.
- **BR-12** [Important] `doc-claims-match-test-reach` The claim that each exits-wrapped refusal site is driven is false: 3 of 7 sites are reached
  This is the 2nd finding in family doc-claims-match-test-reach. The test comment (continuation_guard_test.go:165), the plan's BR-4 revision and the issue Log all claim every withContinuationExits site is driven. The test reaches only recovery_execute.go:179, detach.go:436 and continuation_recovery.go:34. Four sites are never reached: recovery_execute.go:266, :272 and :278, and detach.go:447. Rule: a claim of test reach over a set of sites is checked by the test against that set, or it names exactly the subset driven. Here, a table with one row per wrapped site, plus a source scan asserting that the count of withContinuationExits( call sites equals the table's rows. The guard-agreement test's "refused BY THE GUARD" has the same problem: its oracle phrase is shared with withContinuationExits. Match the guard's own "continuation <id> is failed; " prefix.
- **BR-13** [Minor] `vocabulary-enumerated-by-test` Continuation phases are still written out by hand in menu_action_sweep_test.go:88 and checkpoint/recovery_request_test.go:61
  This is the 2nd finding in family vocabulary-enumerated-by-test. Rule: once a vocabulary has an All*() enumerator, no test restates it. Replace both lists with checkpoint.AllPhases(), and add a check that a []Phase{ literal appears only in AllPhases.
- **BR-14** [Minor] `refusal-names-every-exit` pair continue --retry on a hosted thread names both of Failed's exits regardless of the request's phase (runcli.go:147)
  This is the 2nd finding in family refusal-names-every-exit. Rule: a refusal names exactly the exits valid for the request's actual phase, through checkpoint.Exits(phase, tag). A site that does not know the phase uses phase-neutral wording rather than assuming Failed. Here, dismiss is offered even when the hosted request may be pending or running, which CheckDismissible refuses.

## Round 4 — 2026-09-17T23:18:52-07:00 (claude) — passed

### Disposed

- BR-3 — addressed — Plan:83 now states the "bounded in time" claim is false; atlas:155-165 and menu.go:1258-1263 describe the per-phase sets as statements of which actions are offered, not time bounds, matching menuActionItems.
- BR-4 — addressed — All four non-guard checks wrap once where they return (RecoverThread:179, prepareAbsentContinuation:291 via admitRetainedRecovery, archiveContinuationVacant, validateContinuationWarm); unwrapping :291 in a scratch copy turns the row and the site scan red.
- BR-11 — addressed — Each prompt records its producer; the round-3 unconditional prune and a producer-blind prune each turn TestSwitchAgentOrientationPromptSurvivesContinuationScans red. A replacement-sequence gap in the same family is raised separately.
- BR-12 — addressed — One row per withContinuationExits call site (4), each driven, plus a productionCallsTo scan that the call sites equal the rows; the guard oracle matches the guard's own "continuation <id> is failed; " prefix and requires an unchanged revision.
- BR-13 — addressed — Both lists use AllPhases; restating the list in menu_action_sweep_test.go:88 turns TestPhaseListIsWrittenOnlyInAllPhases red.
- BR-14 — addressed — runcli.go:147 uses Exits("", tag), whose conditional dismiss wording TestExitsWithAnUnknownPhaseAreConditional pins; the call site itself is not pinned (acceptable for a message-only Minor).

### Raised

- **BR-15** [Minor] `in-memory-state-follows-record` A request replaced before the scan sees it vanish strands its orientation prompt (console_continuation.go:145-147)
  This is the 3rd finding in family in-memory-state-follows-record. When the scan sees a new request ID it replaces the watch without dropping the old request's continuation-produced prompt; every later prune compares against the new producer, so Copy orientation prompt stays offered until a switch-agent launch or restart. Reproduced (A's prompt, B seen, B complete, B vanished: prompt survives); base cleared it on any completion for the address. Window: A vanishes and B appears within one scan interval, or while a partial scan error disables the vanish loop. Rule: a view of a record fact lives inside, or is derived from, the object that tracks that fact's identity, not pruned at an enumerated list of events (this family's three findings are three missed events: dismissal, a scan with no request, replacement). Fix: keep the continuation's prompt on continuationWatch, or reconcile continuation-produced entries against the current watch's request ID in one place after every change to c.continuations. Add a replacement-sequence test.
- **BR-16** [Minor] `doc-comment-attachment` admitRetainedRecovery was inserted under prepareAbsentContinuation's doc comment (recovery_execute.go:224-230)
  This is the 2nd finding in family doc-comment-attachment. The combined comment starts "prepareAbsentContinuation snapshots..." but attaches to admitRetainedRecovery, and prepareAbsentContinuation has no doc. Rule: a Go doc comment begins with the name of the declaration it documents; enforce it with a package-level scan that fails when a function's doc starts with the name of another declaration in the same file. Prevalence: 1 new instance in this diff (BR-5 was the first in the family).

## Open findings

- **BR-15** [Minor] `in-memory-state-follows-record` A request replaced before the scan sees it vanish strands its orientation prompt (console_continuation.go:145-147)
- **BR-16** [Minor] `doc-comment-attachment` admitRetainedRecovery was inserted under prepareAbsentContinuation's doc comment (recovery_execute.go:224-230)
