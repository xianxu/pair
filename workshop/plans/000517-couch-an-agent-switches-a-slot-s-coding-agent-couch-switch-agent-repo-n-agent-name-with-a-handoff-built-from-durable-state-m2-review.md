# Boundary Review — pair#517 (milestone M2)

| field | value |
|-------|-------|
| issue | 517 — couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state |
| repo | pair |
| issue file | workshop/issues/000517-couch-an-agent-switches-a-slot-s-coding-agent-couch-switch-agent-repo-n-agent-name-with-a-handoff-built-from-durable-state.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 192c68f08ee46edc34fdd46596cbd94af49fa400..3a6fe6e4b2a8a85d7d9f22745e397ecc40368985 |
| command | sdlc milestone-close --issue 517 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T23:01:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The revised lean switch is largely implemented: authenticated admission, saved configuration, prompt-free readiness, and tty-only previous evidence have passing tests. Shipping is blocked by an interrupted-start recovery bug and an explicit mismatch in the active core-concepts table. The pinned checkout remains unchanged; the disposable test copy was removed.

1. **Strengths**

   - Remote admission rechecks source identity, settled state, clean Git state, and accepted configuration before effects.
   - Fresh registration no longer depends on orientation delivery; targeted readiness and transport acceptance tests pass.
   - Previous-session persistence validates legacy records while dropping obsolete context on rewrite. Exact-capture selection and missing-evidence tests pass.
   - README and atlas updates cover the new command and reduced previous-session contract.

2. **Critical findings**

   - **Interrupted target identity can be discarded during resume — `ARCH-ORDER`, `ARCH-PURPOSE`.**  
     [resume.go:459](/Users/xianxu/Library/Caches/sdlc/review-checkouts/4818fa2d7f450191/cmd/internal/couchcore/resume.go:459) checks only `incarnation.LaunchProfile`. Before registration or `StartRecoveredUnknown`, the replacement profile lives in `incarnation.Start.LaunchProfile`.

     I reproduced: park Claude, record a Codex target helper, lose the supervisor/helper while the target terminal survives, then run startup reconciliation and resume. Reconciliation correctly retained the pending start. Resume subsequently erased it: revision **7 → 10**, no incarnations remaining, old source binding restored, and `resume-session-gone` returned despite the surviving target.

     **Fix:** recognize replacement authority in every pending/completed start representation before clearing lifecycle state. Preserve target identity until exact absence or confirmed registration permits reconciliation. Extend sequence tests across these interruption points; the current test always advances through `StartRecoveredUnknown` first.

   - **Active core-concepts table claims an absent resolver change — `ARCH-PURE` traceability.**  
     [plan:872](/Users/xianxu/Library/Caches/sdlc/review-checkouts/4818fa2d7f450191/workshop/plans/000517-agent-switch-plan.md:872) says `LaunchProfileResolution` was modified to expose configured/missing explicitly. [launchprofile.go:52](/Users/xianxu/Library/Caches/sdlc/review-checkouts/4818fa2d7f450191/cmd/internal/couchcore/launchprofile.go:52) remains unchanged; missing configuration is instead checked in [switchagent.go:241](/Users/xianxu/Library/Caches/sdlc/review-checkouts/4818fa2d7f450191/cmd/internal/couchcore/switchagent.go:241).

     **This is the 2nd finding in family `core-concepts-match-implementation`.** Earlier rounds fixed instances. Do not repair only this row: state and apply the rule that every active row identifies its actual owner, classification, and change status against a named baseline. Add a `## Revisions` entry. One concrete entity contradiction is verified here; the supplied review contract makes it Critical.

3. **Important findings**

   None separately; the missing interrupted-start coverage belongs to the recovery finding above.

4. **Minor findings**

   - [atlas/couch.md:1330](/Users/xianxu/Library/Caches/sdlc/review-checkouts/4818fa2d7f450191/atlas/couch.md:1330) describes pruning through `dropOrientationLocked`, which this range deletes. Reference `reconcileContinuationOrientationLocked` instead.

5. **Test coverage notes**

   - Required stat/name-status inspections and patch checks completed.
   - Targeted switch, previous/peek, capture/replay, persistence, protocol, and readiness selections passed across seven packages.
   - Switch-focused race selections passed for core, CLI/service, and TUI.
   - The additional interrupted-start regression failed as described above.
   - CLI/TUI tests required generating the ignored runtime bundle in the disposable copy. Full `make test` and live operator acceptance were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** existing configuration, queue, receipt, and lifecycle owners are reused.
   - **ARCH-PURE — Flag:** active plan classification/ownership claims need reconciliation with implementation.
   - **ARCH-PURPOSE — Flag:** interrupted replacement recovery does not yet meet the revised contract.
   - **ARCH-MOCK — Pass:** stateful process/session fakes and portable stores share production seams; scheduled lifecycle conformance exists.
   - **ARCH-CONSTRAINTS — Pass:** existing operation bounds remain; full replay stays outside the supervisor.
   - **ARCH-SECURE — Pass:** authentication, server-selected argv, source correlation, and persisted validation are present.
   - **ARCH-ORDER — Flag:** resume bypasses pending replacement authority.
   - **ARCH-FUNERAL — Pass:** previous evidence has supersession/thread-deletion ownership; receipts retain existing bounded expiry.

7. **Plan revision recommendations**

   - Record the actual configuration-policy owner and reconcile every active core-concepts row.
   - Enumerate recovery behavior before and after registration/unknown promotion, requiring surviving target evidence to remain intact.

```findings
findings:
  - id: new
    severity: Critical
    family: replacement-recovery-preserves-unconfirmed-effects
    title: |
      Resume can discard a surviving replacement's pending start identity
    detail: |
      ARCH-ORDER / ARCH-PURPOSE: cmd/internal/couchcore/resume.go:459 checks incarnation.LaunchProfile but misses incarnation.Start.LaunchProfile before registration or unknown promotion. An isolated regression with dead supervisor/helper and a surviving target terminal remained pending after startup reconciliation, then resume erased the target claim (revision 7 to 10, incarnations empty) and returned resume-session-gone. Cover every pending/completed replacement representation before lifecycle cleanup and add interruption-sequence tests asserting retained identity and permitted effects.
  - id: new
    severity: Critical
    family: core-concepts-match-implementation
    title: |
      The active core-concepts table claims a resolver change absent from the implementation
    detail: |
      ARCH-PURE traceability: workshop/plans/000517-agent-switch-plan.md:872 claims LaunchProfileResolution gained an explicit configured/missing result, but cmd/internal/couchcore/launchprofile.go:52 is unchanged and the check lives in switchagent.go:241. This is the 2nd finding in family core-concepts-match-implementation. Earlier rounds fixed instances; do not fix only this row. Apply a rule requiring every active row's owner, classification and change status to match the implementation against a named baseline, sweep the table, and document the correction under Revisions. One concrete entity contradiction was verified.
  - id: new
    severity: Minor
    family: documentation-referents-match-implementation
    title: |
      Atlas still names a deleted orientation-pruning helper
    detail: |
      atlas/couch.md:1330 refers to dropOrientationLocked, removed from cmd/internal/couchtty/console_continuation.go in this range. Update the passage to describe reconcileContinuationOrientationLocked.
```

---

## Re-review — 2026-10-12T23:34:55-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 517 — couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state |
| repo | pair |
| issue file | workshop/issues/000517-couch-an-agent-switches-a-slot-s-coding-agent-couch-switch-agent-repo-n-agent-name-with-a-handoff-built-from-durable-state.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 192c68f08ee46edc34fdd46596cbd94af49fa400..e829fa4c4b4ff65fdea419d769a633903a356dee |
| command | sdlc milestone-close --issue 517 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T23:34:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation matches the revised M2 scope. All three prior findings are addressed, including BR-6’s executable regression evidence. No new correctness defect was found. One inexpensive testing gap remains: pure tty-selection logic lacks direct tests without IO.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      resume.go checks pending profile/binding authority and rechecks before cleanup. Three isolated mutations each fail TestInterruptedSwitchResumePreservesTargetAuthority; restored tests pass.
  - id: BR-7
    disposition: addressed
    note: |
      Plan lines 1020–1088 replace the active inventory with a baseline-specific ownership/classification rule and complete table. Checked owners and unchanged dependencies against M1 37707f73b; launchprofile.go remains correctly classified unchanged.
  - id: BR-8
    disposition: addressed
    note: |
      atlas/couch.md:1330 now names reconcileContinuationOrientationLocked, matching the implementation at console_continuation.go:152 and removal of dropOrientationLocked.
findings:
  - id: new
    severity: Important
    family: pure-core-tested-without-io
    title: |
      Pure previous-tty projection and selection lack direct tests without IO
    detail: |
      ARCH-PURE: previous_session_test.go:62 constructs filesystem-backed Couch state through newTestEnv/createParkedThreadInCouch; peek_previous_test.go exercises SelectPeekSource only through integration paths. Add value-only tests for previousSessionEvidence and SelectPeekSource covering missing park/capture, exact source selection, malformed evidence, and defensive copying. The functions themselves are pure; retain that classification.
```

1. **Strengths**
   - Recovery tests cover pending, reconciled, promoted and registered replacement identities, including a target appearing between probes.
   - Remote admission authenticates the hub and rechecks source identity, settled state and configuration before effects.
   - Fresh registration works independently of orientation delivery.
   - README and atlas document the revised command, recovery behavior and tty-only compatibility.

2. **Critical findings:** None.

3. **Important findings:** Add direct pure-core tests described above.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused core, protocol, CLI, console, launcher, previous-session and replay tests passed.
   - Removing pending-profile handling reproduced the original revision `7→10` identity loss. Removing pending-binding handling or the pre-cleanup recheck also failed regressions.
   - Generated runtime assets and mutations were confined to a disposable copy, subsequently removed. Reviewed checkout remains clean.
   - Full `make test` and live operator acceptance were not independently rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** existing resolver, receipts, queue and lifecycle reused.
   - **ARCH-PURE — flag:** direct pure tests missing as detailed above.
   - **ARCH-PURPOSE — pass:** implements the documented scope revisions.
   - **ARCH-MOCK — pass:** stateful process/session doubles and existing conformance seams retained.
   - **ARCH-CONSTRAINTS — pass:** existing queue, receipt and replay bounds retained.
   - **ARCH-SECURE — pass:** authenticated callers, server-selected configuration and validated persisted evidence.
   - **ARCH-ORDER — pass:** uncertainty preserves replacement authority; mutation-tested interruption sequences.
   - **ARCH-FUNERAL — pass:** bounded receipts and existing thread/capture cleanup own retained state.

7. **Plan revision recommendations:** No ownership correction needed. Record the direct pure-test coverage when added.

---

## Re-review — 2026-10-12T23:44:59-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 517 — couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state |
| repo | pair |
| issue file | workshop/issues/000517-couch-an-agent-switches-a-slot-s-coding-agent-couch-switch-agent-repo-n-agent-name-with-a-handoff-built-from-durable-state.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 192c68f08ee46edc34fdd46596cbd94af49fa400..902ecdf391aa4d28136c7f0cb63273a4027ffed9 |
| command | sdlc milestone-close --issue 517 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T23:44:59-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised M2 contract: authenticated switching with saved configuration, prompt-free startup, and exact previous tty evidence. BR-9 is addressed with direct pure tests and independently verified mutation failures. No new blocking findings.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      previous_session_pure_test.go:13 and peek_selection_test.go:29,52,80 use value-only fixtures covering missing evidence, exact selection, malformed evidence and defensive copying. Removing capture copying, binding copying or validation independently fails these tests; restored code passes.
  - id: BR-6
    disposition: addressed
    note: |
      Prior disposition retained. resume.go checks pending replacement authority and rechecks absence before cleanup; interrupted-switch recovery sequence tests pass.
  - id: BR-7
    disposition: addressed
    note: |
      Prior disposition retained. The active plan inventory identifies ResolveLaunchProfile as unchanged and known-configuration admission as integration code in switchagent.go; comparison against the named M1 baseline confirms the unchanged dependencies.
  - id: BR-8
    disposition: addressed
    note: |
      Prior disposition retained. atlas/couch.md now names reconcileContinuationOrientationLocked, which exists in console_continuation.go and owns continuation prompt pruning.
```

1. **Strengths**
   - Source identity, settled state and Git cleanliness are checked at admission and before effects (`agent_switch_service.go:28`).
   - Saved configuration distinguishes missing settings from explicitly empty argv and is fingerprinted before execution (`switchagent.go:228`).
   - BR-9 tests exercise production functions without filesystem fixtures or mocks.
   - README and atlas document the revised command, recovery semantics and tty-only previous reads.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None requiring action.

5. **Test coverage notes**

   Targeted tests passed across couchcore, couchcmd, couchmessage, couchtty, launcher, threadrecord and scrollbackcmd, including interrupted replacement recovery. Three independent mutations failed the new pure tests; restoration passed.

   Socket/console tests required generating runtime assets in a disposable pinned-source copy, which was removed afterward. The full `make test` suite and live harness conformance were not rerun.

6. **Architectural notes**
   - **ARCH-DRY — Pass:** Reuses profile resolution, operation receipts, queue and lifecycle machinery.
   - **ARCH-PURE — Pass:** Evidence projection and selection now have direct value-only tests.
   - **ARCH-PURPOSE — Pass:** Implements the explicitly revised 22:17/22:19 scope.
   - **ARCH-MOCK — Pass:** Stateful lifecycle and queue doubles exercise production seams; existing live conformance targets remain available.
   - **ARCH-CONSTRAINTS — Pass:** Existing bounded queue, receipt retention and replay limits remain applicable.
   - **ARCH-SECURE — Pass:** Authenticated hub admission, server-selected argv and validated persisted evidence preserve trust boundaries.
   - **ARCH-ORDER — Pass:** Existing transition machinery retains uncertain replacement identity; recovery tests exercise interrupted stages and changing observations.
   - **ARCH-FUNERAL — Pass:** Previous evidence replaces one descriptor; receipts expire; existing lifecycle and retention owners handle residue.

7. **Plan revision recommendations:** None.
