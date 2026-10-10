---
gate: boundary-review
issue: 421
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-09T23:23:28-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Milestones M1 and M3 still describe designs that the round-1 Revisions replaced
          detail: |-
            M1 still says Binding.PairRevision, which PQ-2 replaced with hello-v2 Build. M3 still says the receipt succeeds once the signal is delivered, which PQ-4 replaced with waiting for a new Binding. Add a one-line pointer to the superseding revision on each affected milestone row.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: revision-supersedes-body-unmarked
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-09T23:23:28-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: Settle check is not re-armed or unsettled on lifecycle turn transitions that emit no PTY bytes
          detail: Watchdog, grace and transcript completion clear Active without output, so an idle slot stays unsettled (false busy). Transcript start can open a turn while settled=true stands. processLifecycleObservation should unsettle on open and re-arm on close, with a test for each direction.
          family: derived-state-misses-source-transitions
          round: 2
        - id: BR-3
          severity: Minor
          title: peer_runtime.go hashes os.Executable() by path, so a rebuild between exec and the hash reports the new binary
          detail: It also shadows the executable parameter. This matters for M2's stale-binary rule.
          family: build-identity-from-path-not-image
          round: 2
        - id: BR-4
          severity: Minor
          title: An ack-read timeout is treated as errNoAck, so a slow new broker downgrades the connection to legacy hello
          family: negotiation-fallback-on-timeout
          round: 2
        - id: BR-5
          severity: Minor
          title: The full sha256 of the executable runs synchronously in startPeerRuntime on wrapper startup, with no measured budget
          family: startup-path-unmeasured-work
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-09T23:27:35-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: plan:104 and plan:124 now carry inline superseded pointers to PQ-2 / PQ-4.
          round: 3
        - id: BR-2
          disposition: addressed
          note: notification_lifecycle.go:246 Swap + lifecycleTurnChanged; TestSettleFollowsSilentTurnTransitions covers open and close.
          round: 3
        - id: BR-3
          disposition: addressed
          note: selfBuildIdentity runs first in run() (wrap.go:2693), before anything is spawned; the shadowing is removed.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Only EOF/UnexpectedEOF/ECONNRESET map to errNoAck; TestNegotiationTimeoutDoesNotFallBack would fail on the old code.
          round: 3
        - id: BR-5
          disposition: addressed
          note: The hash runs once at run() entry with a stated ~10ms cost; it is no longer in startPeerRuntime.
          round: 3
      findings:
        - id: BR-6
          severity: Minor
          title: A silent turn change does not advance the settle generation, so an in-flight check can re-assert Settled=true
          detail: 'This is the 2nd finding in this family. The rule: every source transition of Settled (output, input, lifecycle open/close) must both re-arm the check AND invalidate any check already in flight. lifecycleTurnChanged re-arms but leaves d.sequence alone, so a settleFired whose probe ran before turnActive.Swap still passes its seq check and publishes Settled=true during an open turn, for up to SettleInterval. Fix: give settle its own counter that every source bumps, and check it in settleFired.'
          family: derived-state-misses-source-transitions
          round: 3
      boundary: M1
      recipe: milestone-review
      reviewed: ef0e61f28c9909674768fce988a66a9412ecd63e
      blocked: false
    - "n": 4
      timestamp: "2026-10-10T00:05:59-07:00"
      agent: claude
      findings:
        - id: BR-7
          severity: Important
          title: Relaunch failure outcomes (park-incomplete vs park-ok-resume-failed) reach the receipt as untyped text, and the admission note is dropped on failure
          detail: slotOperationOutcome maps any err to failed with err text only; the plan promised the receipt maps RelaunchResult.Outcome, and notedResult wraps only when err == nil, so a forced-unknown failure no longer records --force-unknown. Set the receipt Code from the outcome, keep the note on failure, and test each outcome.
          family: typed-outcome-survives-boundary
          round: 4
        - id: BR-8
          severity: Important
          title: liveRestartProbe.LiveRestartFacts (settled/session/git mapping) and messageService.LivenessForThread have no tests
          detail: Only binaryFacts is tested. Settled nil→unknown, git error→GitKnown=false, and the scope/tag match are unpinned; a wrong match field silently makes every relaunch busy-unknown. Add a table test with a seeded liveness map and a fake git.
          family: safety-guard-wiring-untested
          round: 4
        - id: BR-9
          severity: Important
          title: README.md slot-operation verb list (lines 389-393) is missing couch --relaunch and its overrides
          family: docs-surface-lags-cli
          round: 4
        - id: BR-10
          severity: Important
          title: 'Core concepts table contradicts code: BinaryProbe location, SlotLiveness rename, DecideBinaryFreshness signature, PAIR_DEV skip absent from plan'
          detail: '2nd in family. Rule: every delta from a Core concepts row or milestone body lands as a dated Revisions entry in the same commit as the diverging code, and each milestone close diffs the table against the tree. M2 has no deltas entry.'
          family: revision-supersedes-body-unmarked
          round: 4
        - id: BR-11
          severity: Minor
          title: A relaunch sent to a pre-421 Couch prints the misleading "predates --resume/--reboot"; with overrides set, the strict-decode error is unmapped
          family: version-skew-refusal-names-fix
          round: 4
        - id: BR-12
          severity: Minor
          title: '"relaunch" literal restated in live_restart_probe.go:43 and cli.go:311, and the CLI verb lists do not read slotOperations'
          family: single-source-verb-list
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-10-10T00:19:02-07:00"
      agent: claude
      dispose:
        - id: BR-7
          disposition: addressed
          note: ReceiptCode carried through notedResult to slotOperationOutcome; TestRelaunchFailureOutcomeIsTyped fails without it. Note-on-failure in message_service.go:338 is unpinned (Minor).
          round: 5
        - id: BR-8
          disposition: addressed
          note: TestLiveRestartProbeFactsMapping covers settled/legacy/no-session/git-error/scope mismatch and the detached probe.
          round: 5
        - id: BR-9
          disposition: addressed
          note: README.md:394-396 lists --relaunch with --confirm, --same-binary, --force-unknown and its refusals.
          round: 5
        - id: BR-10
          disposition: addressed
          note: Plan core-concepts rows are marked superseded inline, plus a dated 2026-10-10 M2 deltas revision.
          round: 5
        - id: BR-11
          disposition: addressed
          note: messages.go:187 maps unknown-field and names the verb; covered by a new case in TestSlotOperationCLIPollLoop.
          round: 5
        - id: BR-12
          disposition: addressed
          note: OpRelaunch exported and used at live_restart_probe.go:43; cli.go reads IsSlotOperation/SlotOperationTakesOverrides.
          round: 5
      findings:
        - id: BR-13
          severity: Minor
          title: Admission note surviving a failed slot operation (message_service.go:338) has no regression test
          detail: '2nd finding in this family. Rule: every closure that adapts a queue result into a receipt needs a test driving it with err != nil. Restoring the `&& err == nil` guard leaves all tests green.'
          family: safety-guard-wiring-untested
          round: 5
      boundary: M2
      recipe: milestone-review
      reviewed: 2c3a0ad9d21a94272f7a19857f735c03ebd9e333
      blocked: false
---

# Gate ledger — pair#421 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-09T23:23:28-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `revision-supersedes-body-unmarked` Milestones M1 and M3 still describe designs that the round-1 Revisions replaced
  M1 still says Binding.PairRevision, which PQ-2 replaced with hello-v2 Build. M3 still says the receipt succeeds once the signal is delivered, which PQ-4 replaced with waiting for a new Binding. Add a one-line pointer to the superseding revision on each affected milestone row.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-10-09T23:23:28-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `derived-state-misses-source-transitions` Settle check is not re-armed or unsettled on lifecycle turn transitions that emit no PTY bytes
  Watchdog, grace and transcript completion clear Active without output, so an idle slot stays unsettled (false busy). Transcript start can open a turn while settled=true stands. processLifecycleObservation should unsettle on open and re-arm on close, with a test for each direction.
- **BR-3** [Minor] `build-identity-from-path-not-image` peer_runtime.go hashes os.Executable() by path, so a rebuild between exec and the hash reports the new binary
  It also shadows the executable parameter. This matters for M2's stale-binary rule.
- **BR-4** [Minor] `negotiation-fallback-on-timeout` An ack-read timeout is treated as errNoAck, so a slow new broker downgrades the connection to legacy hello
- **BR-5** [Minor] `startup-path-unmeasured-work` The full sha256 of the executable runs synchronously in startPeerRuntime on wrapper startup, with no measured budget

## Round 3 — 2026-10-09T23:27:35-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — plan:104 and plan:124 now carry inline superseded pointers to PQ-2 / PQ-4.
- BR-2 — addressed — notification_lifecycle.go:246 Swap + lifecycleTurnChanged; TestSettleFollowsSilentTurnTransitions covers open and close.
- BR-3 — addressed — selfBuildIdentity runs first in run() (wrap.go:2693), before anything is spawned; the shadowing is removed.
- BR-4 — addressed — Only EOF/UnexpectedEOF/ECONNRESET map to errNoAck; TestNegotiationTimeoutDoesNotFallBack would fail on the old code.
- BR-5 — addressed — The hash runs once at run() entry with a stated ~10ms cost; it is no longer in startPeerRuntime.

### Raised

- **BR-6** [Minor] `derived-state-misses-source-transitions` A silent turn change does not advance the settle generation, so an in-flight check can re-assert Settled=true
  This is the 2nd finding in this family. The rule: every source transition of Settled (output, input, lifecycle open/close) must both re-arm the check AND invalidate any check already in flight. lifecycleTurnChanged re-arms but leaves d.sequence alone, so a settleFired whose probe ran before turnActive.Swap still passes its seq check and publishes Settled=true during an open turn, for up to SettleInterval. Fix: give settle its own counter that every source bumps, and check it in settleFired.

## Round 4 — 2026-10-10T00:05:59-07:00 (claude) — BLOCKED

### Raised

- **BR-7** [Important] `typed-outcome-survives-boundary` Relaunch failure outcomes (park-incomplete vs park-ok-resume-failed) reach the receipt as untyped text, and the admission note is dropped on failure
  slotOperationOutcome maps any err to failed with err text only; the plan promised the receipt maps RelaunchResult.Outcome, and notedResult wraps only when err == nil, so a forced-unknown failure no longer records --force-unknown. Set the receipt Code from the outcome, keep the note on failure, and test each outcome.
- **BR-8** [Important] `safety-guard-wiring-untested` liveRestartProbe.LiveRestartFacts (settled/session/git mapping) and messageService.LivenessForThread have no tests
  Only binaryFacts is tested. Settled nil→unknown, git error→GitKnown=false, and the scope/tag match are unpinned; a wrong match field silently makes every relaunch busy-unknown. Add a table test with a seeded liveness map and a fake git.
- **BR-9** [Important] `docs-surface-lags-cli` README.md slot-operation verb list (lines 389-393) is missing couch --relaunch and its overrides
- **BR-10** [Important] `revision-supersedes-body-unmarked` Core concepts table contradicts code: BinaryProbe location, SlotLiveness rename, DecideBinaryFreshness signature, PAIR_DEV skip absent from plan
  2nd in family. Rule: every delta from a Core concepts row or milestone body lands as a dated Revisions entry in the same commit as the diverging code, and each milestone close diffs the table against the tree. M2 has no deltas entry.
- **BR-11** [Minor] `version-skew-refusal-names-fix` A relaunch sent to a pre-421 Couch prints the misleading "predates --resume/--reboot"; with overrides set, the strict-decode error is unmapped
- **BR-12** [Minor] `single-source-verb-list` "relaunch" literal restated in live_restart_probe.go:43 and cli.go:311, and the CLI verb lists do not read slotOperations

## Round 5 — 2026-10-10T00:19:02-07:00 (claude) — passed

### Disposed

- BR-7 — addressed — ReceiptCode carried through notedResult to slotOperationOutcome; TestRelaunchFailureOutcomeIsTyped fails without it. Note-on-failure in message_service.go:338 is unpinned (Minor).
- BR-8 — addressed — TestLiveRestartProbeFactsMapping covers settled/legacy/no-session/git-error/scope mismatch and the detached probe.
- BR-9 — addressed — README.md:394-396 lists --relaunch with --confirm, --same-binary, --force-unknown and its refusals.
- BR-10 — addressed — Plan core-concepts rows are marked superseded inline, plus a dated 2026-10-10 M2 deltas revision.
- BR-11 — addressed — messages.go:187 maps unknown-field and names the verb; covered by a new case in TestSlotOperationCLIPollLoop.
- BR-12 — addressed — OpRelaunch exported and used at live_restart_probe.go:43; cli.go reads IsSlotOperation/SlotOperationTakesOverrides.

### Raised

- **BR-13** [Minor] `safety-guard-wiring-untested` Admission note surviving a failed slot operation (message_service.go:338) has no regression test
  2nd finding in this family. Rule: every closure that adapts a queue result into a receipt needs a test driving it with err != nil. Restoring the `&& err == nil` guard leaves all tests green.

## Open findings

- **BR-6** [Minor] `derived-state-misses-source-transitions` A silent turn change does not advance the settle generation, so an in-flight check can re-assert Settled=true
- **BR-13** [Minor] `safety-guard-wiring-untested` Admission note surviving a failed slot operation (message_service.go:338) has no regression test
