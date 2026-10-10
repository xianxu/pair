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
    - "n": 6
      timestamp: "2026-10-10T00:50:56-07:00"
      agent: claude
      findings:
        - id: BR-14
          severity: Important
          title: reload-context dispatch case and its not-live check are untested (deleting the executor case fails no reload-context test)
          detail: '3rd in family. Rule: every ExecuteLiveOwner operation in Operations() must be reachable through CouchLiveOwnerExecutor. Add a table test that dispatches each one via DispatchOperation and asserts no unknown-operation error, plus a ReloadContext not-live case (LiveRestartNotLive, no probe restart). Mutation run: with the case removed, couchcore''s only failures were sandbox pty denials and a git-less scratch tree.'
          family: safety-guard-wiring-untested
          round: 6
        - id: BR-15
          severity: Important
          title: Settled (busy) guard is checked at admission but not re-checked under the thread hold before SIGUSR2 or relaunch
          detail: The ReloadContext comment admits the queue runs between admission and effect, yet it re-checks only liveness. RestartConversation already holds before.Settled; refuse unless known true, apply the same check to Relaunch, and add a test where the snapshot flips to unsettled between admission and effect.
          family: guard-rechecked-at-effect
          round: 6
        - id: BR-16
          severity: Important
          title: Plan still places ReloadContext in reload_context.go and describes a shared SignalWrapper helper and helper-extraction tests
          detail: '3rd in family. Rule: each Revisions delta names the identifiers and paths it replaces, and its commit greps the plan body and marks every hit superseded (table rows, prose, milestone rows). Sweep reload_context.go, SignalWrapper/SignalVerifiedWrapper, and "helper extraction".'
          family: revision-supersedes-body-unmarked
          round: 6
        - id: BR-17
          severity: Minor
          title: A reload-context success receipt carries no Tag, unlike the documented "names the thread left running"
          family: typed-outcome-survives-boundary
          round: 6
        - id: BR-18
          severity: Minor
          title: --same-binary is silently accepted with --reload-context; refuse it instead
          family: override-accepted-without-effect
          round: 6
        - id: BR-19
          severity: Minor
          title: RestartConversation's cancelled-context path has no test
          family: safety-guard-wiring-untested
          round: 6
      boundary: M3
      recipe: milestone-review
      blocked: true
    - "n": 7
      timestamp: "2026-10-10T00:56:29-07:00"
      agent: claude
      dispose:
        - id: BR-14
          disposition: addressed
          note: TestEveryLiveOwnerOperationIsDispatched enumerates Operations(); mutation removing the OpReloadContext case goes red; TestReloadContextRefusesWhenNoLongerLive covers not-live.
          round: 7
        - id: BR-15
          disposition: addressed
          note: refuseKnownBusy in RestartConversation (under hold) and ConfirmNotBusy via require-settled for relaunch; mutation removing ConfirmNotBusy fails TestLiveRestartGuardRecheckedAtEffect.
          round: 7
        - id: BR-16
          disposition: addressed
          note: Plan lines 85, 94, 130 now mark reload_context.go, SignalWrapper and helper extraction superseded inline.
          round: 7
        - id: BR-17
          disposition: addressed
          note: ReloadContextResult.ReceiptTag plus notedResult passthrough; asserted in TestRestartEffectRechecksBusyAndCancellation.
          round: 7
        - id: BR-18
          disposition: addressed
          note: SlotOperationTakesSameBinary gates CLI and ValidateRequest; TestReloadContextRefusesSameBinary and protocol_test cover it.
          round: 7
        - id: BR-19
          disposition: addressed
          note: Cancelled-context path asserts ReloadUnconfirmed containing "cancelled".
          round: 7
      findings:
        - id: BR-20
          severity: Minor
          title: Deleted SlotOperationTakesOverrides still named in a code doc comment and in the M3 Revisions bullets
          detail: '4th in family. slot_operation.go:39 keeps the old function''s doc comment, and the plan''s M3 deltas still say SlotOperationTakesOverrides is the single list and --same-binary is accepted without effect. Rule: when a commit removes or renames an identifier, run git grep for it across code and plan in that commit, then delete or mark superseded every remaining hit.'
          family: revision-supersedes-body-unmarked
          round: 7
        - id: BR-21
          severity: Minor
          title: ConfirmNotBusy passes "unknown" at the effect even when admission saw a known-settled wrapper
          detail: '2nd in family. The comment assumes admission required --force-unknown, but a wrapper that disconnects between admission and effect turns known into unknown without operator consent (ARCH-ORDER: uncertainty collapsed into a pass). Rule: the effect re-check reads the same decision admission made. Carry whether admission was forced in require-settled (for example "known" or "forced") and refuse unknown unless it was forced.'
          family: guard-rechecked-at-effect
          round: 7
      boundary: M3
      recipe: milestone-review
      reviewed: e804157197c3868fa4a95604851f5ff4ef046afd
      blocked: false
    - "n": 8
      timestamp: "2026-10-10T01:22:59-07:00"
      agent: claude
      findings:
        - id: BR-22
          severity: Important
          title: Dispatch translation of require-settled into the probe's forced flag is untested; the fake's forced field has no reader
          detail: 'operationdispatch.go:415 and :425 map known/forced into a bool, and no test asserts it. Hard-coding false at :425 makes reload-context''s --force-unknown always refused, and every test stays green. Fifth in this family. Rule: a value admission writes into an implicit argument is tested from producer to consumer, one table through PrepareSlotOperation, CouchLiveOwnerExecutor and the probe, over verb x {known, forced, absent}. Add the rule to lessons.md.'
          family: safety-guard-wiring-untested
          round: 8
        - id: BR-23
          severity: Important
          title: M4 live-check deferral leaves the issue Done-when and plan M4 row unmarked
          detail: 'Issue Done-when line 65 still requires a live check; plan line 134 still promises it with no deferred marker. Fifth in this family. Rule: a scope-changing revision marks every line that states the scope (Done-when, issue Plan, plan body) in the same commit. Extend the existing lesson to cover Done-when.'
          family: revision-supersedes-body-unmarked
          round: 8
        - id: BR-24
          severity: Minor
          title: Admission recomputes the unknown predicate for the forced evidence instead of taking it from DecideLiveRestart
          detail: 'live_restart.go:239 duplicates :80, and the value is a free string. Rule: a decision''s outcome crosses the seam as the value the decision produced, not one recomputed from the inputs.'
          family: typed-outcome-survives-boundary
          round: 8
        - id: BR-25
          severity: Minor
          title: SKILL.md busy-unknown names only the old-wrapper cause, not the disconnect-after-admission refusal added here
          detail: 'Rule: each refusal code''s doc lists every site that emits it, and the fix for each.'
          family: docs-surface-lags-cli
          round: 8
      boundary: M4
      recipe: milestone-review
      blocked: true
    - "n": 9
      timestamp: "2026-10-10T01:26:03-07:00"
      agent: claude
      dispose:
        - id: BR-22
          disposition: addressed
          note: TestRequireSettledProducerToConsumer covers both verbs x known/forced/absent through PrepareSlotOperation, CouchLiveOwnerExecutor and the probe. A scratch mutation hard-coding false at the reload dispatch turns it red. The rule is in lessons.md.
          round: 9
        - id: BR-23
          disposition: addressed
          note: The issue Done-when (lines 65-68), the issue M4 row (line 101) and the plan M4 row (line 134) all mark the deferral. The lesson extends to Done-when.
          round: 9
        - id: BR-24
          disposition: addressed
          note: DecideLiveRestart returns Forced, and prepareLiveRestart writes d.Forced; the duplicate predicate is gone.
          round: 9
        - id: BR-25
          disposition: addressed
          note: SKILL.md busy-unknown now names the admission causes and the effect-time disconnect, each with its fix.
          round: 9
      boundary: M4
      recipe: milestone-review
      reviewed: 77b1f23cebdb4acfaeba1b258638faa4c9a9b9d7
      blocked: false
    - "n": 10
      timestamp: "2026-10-10T01:36:31-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: addressed
          note: armSettleLocked bumps settleGen on every source incl. lifecycleTurnChanged; settleFired rechecks seq after probe; TestSettleInFlightCheckInvalidatedBySilentTurn.
          round: 10
        - id: BR-13
          disposition: addressed
          note: TestWithAdmissionNoteOnSuccessAndFailure drives withAdmissionNote with a non-nil error and asserts the note survives.
          round: 10
        - id: BR-20
          disposition: addressed
          note: slot_operation.go:39 comment now names SlotOperationTakesForceUnknown; remaining plan hits are marked superseded inline.
          round: 10
        - id: BR-21
          disposition: addressed
          note: refuseUnlessIdle refuses unknown unless forced; require-settled carries known/forced; tested in TestRestartEffectRechecksBusyAndCancellation and the BR-22 dispatch matrix.
          round: 10
      recipe: milestone-review
      reviewed: e157ce9ae95b9f33820983f13e92c2056e4e61cf
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

## Round 6 — 2026-10-10T00:50:56-07:00 (claude) — BLOCKED

### Raised

- **BR-14** [Important] `safety-guard-wiring-untested` reload-context dispatch case and its not-live check are untested (deleting the executor case fails no reload-context test)
  3rd in family. Rule: every ExecuteLiveOwner operation in Operations() must be reachable through CouchLiveOwnerExecutor. Add a table test that dispatches each one via DispatchOperation and asserts no unknown-operation error, plus a ReloadContext not-live case (LiveRestartNotLive, no probe restart). Mutation run: with the case removed, couchcore's only failures were sandbox pty denials and a git-less scratch tree.
- **BR-15** [Important] `guard-rechecked-at-effect` Settled (busy) guard is checked at admission but not re-checked under the thread hold before SIGUSR2 or relaunch
  The ReloadContext comment admits the queue runs between admission and effect, yet it re-checks only liveness. RestartConversation already holds before.Settled; refuse unless known true, apply the same check to Relaunch, and add a test where the snapshot flips to unsettled between admission and effect.
- **BR-16** [Important] `revision-supersedes-body-unmarked` Plan still places ReloadContext in reload_context.go and describes a shared SignalWrapper helper and helper-extraction tests
  3rd in family. Rule: each Revisions delta names the identifiers and paths it replaces, and its commit greps the plan body and marks every hit superseded (table rows, prose, milestone rows). Sweep reload_context.go, SignalWrapper/SignalVerifiedWrapper, and "helper extraction".
- **BR-17** [Minor] `typed-outcome-survives-boundary` A reload-context success receipt carries no Tag, unlike the documented "names the thread left running"
- **BR-18** [Minor] `override-accepted-without-effect` --same-binary is silently accepted with --reload-context; refuse it instead
- **BR-19** [Minor] `safety-guard-wiring-untested` RestartConversation's cancelled-context path has no test

## Round 7 — 2026-10-10T00:56:29-07:00 (claude) — passed

### Disposed

- BR-14 — addressed — TestEveryLiveOwnerOperationIsDispatched enumerates Operations(); mutation removing the OpReloadContext case goes red; TestReloadContextRefusesWhenNoLongerLive covers not-live.
- BR-15 — addressed — refuseKnownBusy in RestartConversation (under hold) and ConfirmNotBusy via require-settled for relaunch; mutation removing ConfirmNotBusy fails TestLiveRestartGuardRecheckedAtEffect.
- BR-16 — addressed — Plan lines 85, 94, 130 now mark reload_context.go, SignalWrapper and helper extraction superseded inline.
- BR-17 — addressed — ReloadContextResult.ReceiptTag plus notedResult passthrough; asserted in TestRestartEffectRechecksBusyAndCancellation.
- BR-18 — addressed — SlotOperationTakesSameBinary gates CLI and ValidateRequest; TestReloadContextRefusesSameBinary and protocol_test cover it.
- BR-19 — addressed — Cancelled-context path asserts ReloadUnconfirmed containing "cancelled".

### Raised

- **BR-20** [Minor] `revision-supersedes-body-unmarked` Deleted SlotOperationTakesOverrides still named in a code doc comment and in the M3 Revisions bullets
  4th in family. slot_operation.go:39 keeps the old function's doc comment, and the plan's M3 deltas still say SlotOperationTakesOverrides is the single list and --same-binary is accepted without effect. Rule: when a commit removes or renames an identifier, run git grep for it across code and plan in that commit, then delete or mark superseded every remaining hit.
- **BR-21** [Minor] `guard-rechecked-at-effect` ConfirmNotBusy passes "unknown" at the effect even when admission saw a known-settled wrapper
  2nd in family. The comment assumes admission required --force-unknown, but a wrapper that disconnects between admission and effect turns known into unknown without operator consent (ARCH-ORDER: uncertainty collapsed into a pass). Rule: the effect re-check reads the same decision admission made. Carry whether admission was forced in require-settled (for example "known" or "forced") and refuse unknown unless it was forced.

## Round 8 — 2026-10-10T01:22:59-07:00 (claude) — BLOCKED

### Raised

- **BR-22** [Important] `safety-guard-wiring-untested` Dispatch translation of require-settled into the probe's forced flag is untested; the fake's forced field has no reader
  operationdispatch.go:415 and :425 map known/forced into a bool, and no test asserts it. Hard-coding false at :425 makes reload-context's --force-unknown always refused, and every test stays green. Fifth in this family. Rule: a value admission writes into an implicit argument is tested from producer to consumer, one table through PrepareSlotOperation, CouchLiveOwnerExecutor and the probe, over verb x {known, forced, absent}. Add the rule to lessons.md.
- **BR-23** [Important] `revision-supersedes-body-unmarked` M4 live-check deferral leaves the issue Done-when and plan M4 row unmarked
  Issue Done-when line 65 still requires a live check; plan line 134 still promises it with no deferred marker. Fifth in this family. Rule: a scope-changing revision marks every line that states the scope (Done-when, issue Plan, plan body) in the same commit. Extend the existing lesson to cover Done-when.
- **BR-24** [Minor] `typed-outcome-survives-boundary` Admission recomputes the unknown predicate for the forced evidence instead of taking it from DecideLiveRestart
  live_restart.go:239 duplicates :80, and the value is a free string. Rule: a decision's outcome crosses the seam as the value the decision produced, not one recomputed from the inputs.
- **BR-25** [Minor] `docs-surface-lags-cli` SKILL.md busy-unknown names only the old-wrapper cause, not the disconnect-after-admission refusal added here
  Rule: each refusal code's doc lists every site that emits it, and the fix for each.

## Round 9 — 2026-10-10T01:26:03-07:00 (claude) — passed

### Disposed

- BR-22 — addressed — TestRequireSettledProducerToConsumer covers both verbs x known/forced/absent through PrepareSlotOperation, CouchLiveOwnerExecutor and the probe. A scratch mutation hard-coding false at the reload dispatch turns it red. The rule is in lessons.md.
- BR-23 — addressed — The issue Done-when (lines 65-68), the issue M4 row (line 101) and the plan M4 row (line 134) all mark the deferral. The lesson extends to Done-when.
- BR-24 — addressed — DecideLiveRestart returns Forced, and prepareLiveRestart writes d.Forced; the duplicate predicate is gone.
- BR-25 — addressed — SKILL.md busy-unknown now names the admission causes and the effect-time disconnect, each with its fix.

## Round 10 — 2026-10-10T01:36:31-07:00 (claude) — passed

### Disposed

- BR-6 — addressed — armSettleLocked bumps settleGen on every source incl. lifecycleTurnChanged; settleFired rechecks seq after probe; TestSettleInFlightCheckInvalidatedBySilentTurn.
- BR-13 — addressed — TestWithAdmissionNoteOnSuccessAndFailure drives withAdmissionNote with a non-nil error and asserts the note survives.
- BR-20 — addressed — slot_operation.go:39 comment now names SlotOperationTakesForceUnknown; remaining plan hits are marked superseded inline.
- BR-21 — addressed — refuseUnlessIdle refuses unknown unless forced; require-settled carries known/forced; tested in TestRestartEffectRechecksBusyAndCancellation and the BR-22 dispatch matrix.

## Open findings

(none — every finding has been disposed)
