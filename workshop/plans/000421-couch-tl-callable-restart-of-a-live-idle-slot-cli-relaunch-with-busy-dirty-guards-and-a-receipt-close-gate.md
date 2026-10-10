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

## Open findings

- **BR-6** [Minor] `derived-state-misses-source-transitions` A silent turn change does not advance the settle generation, so an in-flight check can re-assert Settled=true
