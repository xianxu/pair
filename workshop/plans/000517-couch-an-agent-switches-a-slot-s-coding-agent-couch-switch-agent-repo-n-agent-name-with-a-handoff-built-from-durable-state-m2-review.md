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
