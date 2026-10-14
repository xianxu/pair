---
gate: plan-quality
issue: 522
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-13T17:58:44-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Non-blocking Missed treats every failure as "no agent running"; some failures leave a possibly-live target
          detail: 'continuation.go:256 (unresolved target start) and 382-389 (AbortStarted cleanup errors joined into the failure) fail while a target may still be alive; today the blocking Failed phase is what prevents a second actor on relaunch, switch or compaction. Split failures into confirmed-no-target and target-may-exist, name what guards against a duplicate start for the second class (cite the existing start-claim/incarnation guard, or keep a guard), and add a sequence test: an AbortStarted failure followed by relaunch must not produce two agents.'
          family: uncertain-outcome-collapsed
          round: 1
        - id: PQ-2
          severity: Minor
          title: Completion at helper registration doesn't confirm the agent consumed the prompt; state this as a deliberate non-goal
          detail: A prompt swallowed by a startup screen (e.g. Codex's trust prompt) would complete silently. Say this is intended and include a trust-prompt case in the live check. Also state that resume-with-prompt (offered in the Spec) is a non-goal, and why.
          family: unstated-non-goal
          round: 1
        - id: PQ-3
          severity: Minor
          title: Changes to checkpoint.Advance and the atomic clear-plus-Missed store method are not spelled out
          detail: Name the Advance changes in checkpoint/request.go (Registered completes the request; Fail/RetryAbsent/RetryObserve removed) and the ThreadStore method that clears the request and writes ContinuationMissed in one update.
          family: pure-transition-change-unstated
          round: 1
        - id: PQ-4
          severity: Minor
          title: '"Relaunch resumes the old conversation after park" has no file:line'
          detail: Cite the relaunch path that resumes the parked session after rollbackTrackedStart.
          family: unbacked-existing-behavior-claim
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T17:59:44-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Split into no-target/target-may-exist; guard is CommitStartClaim incarnation check (threadstore.go:606) + relaunch's exact park (relaunch.go:117-123), verified; sequence test added.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Non-goals stated (prompt consumption unconfirmed by design; resume-with-prompt YAGNI); trust-screen live case added.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Advance/Fail/Retry*/CheckDismissible changes and ThreadStore.MissContinuation named.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Relaunch/resume path cited and covered by a VerifiedPark-survives-rollback test.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-13T18:00:34-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
    - "n": 4
      timestamp: "2026-10-13T18:01:14-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
content_hash: ff4e65e2e23179d7662b8cd63e3108f0e245c7220eaa5003e79de311f006dec3
---

# Gate ledger — pair#522 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T17:58:44-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `uncertain-outcome-collapsed` Non-blocking Missed treats every failure as "no agent running"; some failures leave a possibly-live target
  continuation.go:256 (unresolved target start) and 382-389 (AbortStarted cleanup errors joined into the failure) fail while a target may still be alive; today the blocking Failed phase is what prevents a second actor on relaunch, switch or compaction. Split failures into confirmed-no-target and target-may-exist, name what guards against a duplicate start for the second class (cite the existing start-claim/incarnation guard, or keep a guard), and add a sequence test: an AbortStarted failure followed by relaunch must not produce two agents.
- **PQ-2** [Minor] `unstated-non-goal` Completion at helper registration doesn't confirm the agent consumed the prompt; state this as a deliberate non-goal
  A prompt swallowed by a startup screen (e.g. Codex's trust prompt) would complete silently. Say this is intended and include a trust-prompt case in the live check. Also state that resume-with-prompt (offered in the Spec) is a non-goal, and why.
- **PQ-3** [Minor] `pure-transition-change-unstated` Changes to checkpoint.Advance and the atomic clear-plus-Missed store method are not spelled out
  Name the Advance changes in checkpoint/request.go (Registered completes the request; Fail/RetryAbsent/RetryObserve removed) and the ThreadStore method that clears the request and writes ContinuationMissed in one update.
- **PQ-4** [Minor] `unbacked-existing-behavior-claim` "Relaunch resumes the old conversation after park" has no file:line
  Cite the relaunch path that resumes the parked session after rollbackTrackedStart.

## Round 2 — 2026-10-13T17:59:44-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Split into no-target/target-may-exist; guard is CommitStartClaim incarnation check (threadstore.go:606) + relaunch's exact park (relaunch.go:117-123), verified; sequence test added.
- PQ-2 — addressed — Non-goals stated (prompt consumption unconfirmed by design; resume-with-prompt YAGNI); trust-screen live case added.
- PQ-3 — addressed — Advance/Fail/Retry*/CheckDismissible changes and ThreadStore.MissContinuation named.
- PQ-4 — addressed — Relaunch/resume path cited and covered by a VerifiedPark-survives-rollback test.

## Round 3 — 2026-10-13T18:00:34-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 4 — 2026-10-13T18:01:14-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)
