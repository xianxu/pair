---
gate: plan-quality
issue: 421
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-09T22:59:56-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: reload-context signals via the pair-wrap pid file instead of the broker's verified Binding PID+Start
          detail: 'agentcmd.RunRestart (agentcmd/restart.go:20-52) kills whatever pid the file names, which is fine in-slot but not cross-process from Couch: a stale or rewritten pid file or a mismatched PAIR_DATA_DIR sends SIGUSR2 (default action: terminate) to the wrong process. The broker already holds Binding.PID+Start for the exact incarnation (wrapcmd/peer_runtime.go:136). Signal that PID after re-checking Start, and make the shared helper take a verified incarnation (ARCH-SECURE, ARCH-ORDER).'
          family: signal-target-by-verified-identity
          round: 1
        - id: PQ-2
          severity: Minor
          title: Adding PairRevision to couchmessage.Binding changes the broker actor map key and the wire contract unannounced
          detail: Binding is the key of Broker.actors (broker.go:68) and is validated in model.go:32. State how an old Couch and a new wrapper (and the reverse) behave, or carry the revision outside the identity struct.
          family: unstated-contract-change
          round: 1
        - id: PQ-3
          severity: Minor
          title: DecideBinaryFreshness compares vcs.revision only; vcs.modified builds and unknown build info have no stated rule
          detail: Two dirty builds at the same revision compare equal, so a real fix gets refused as stale-binary. Consider comparing the Go build ID or a content hash, and state the rule for unknown build info.
          family: freshness-rule-underspecified
          round: 1
        - id: PQ-4
          severity: Minor
          title: reload-context receipt reports succeeded on signal delivery, collapsing an unconfirmed outcome into success
          detail: The wrapper re-execs on SIGUSR2 and re-publishes its binding. Wait, with a time limit, for that evidence, or mark the receipt as unconfirmed.
          family: uncertain-outcome-collapsed
          round: 1
        - id: PQ-5
          severity: Minor
          title: M1 and M2 enumerate test cases in prose instead of one strategy line per risky function
          detail: 'For DecideLiveRestart and DecideBinaryFreshness: property or table tests over the fact space. For the settle timer: an injected clock with interleaved activity.'
          family: test-prose-enumeration
          round: 1
        - id: PQ-6
          severity: Minor
          title: Plan states no non-goals
          detail: 'For example: no fleet or batch verb, no automatic make build, no durable receipts, no new check of Settled between admission and park.'
          family: no-stated-non-goals
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-09T23:01:38-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Signals Binding.PID only after OSProcOps.Identity (procops.go:109) re-check equals Binding.Start (peer_runtime.go:136); pid file no longer used cross-process.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Binding (model.go:26) untouched; hello-v2 try-then-fallback relies on strictjson decode (session_protocol.go:70); four pairings tested.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Content-hash identity; modified and unknown-build rules stated.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Bounded 20s wait for a new Binding; otherwise unknown/unconfirmed.
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
        - id: PQ-6
          disposition: addressed
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-09T23:02:11-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          round: 3
        - id: PQ-2
          disposition: addressed
          round: 3
        - id: PQ-3
          disposition: addressed
          round: 3
        - id: PQ-4
          disposition: addressed
          round: 3
        - id: PQ-5
          disposition: addressed
          round: 3
        - id: PQ-6
          disposition: addressed
          round: 3
      findings:
        - id: PQ-7
          severity: Minor
          title: Milestones M1 and M3 still describe designs that the round-1 Revisions replaced
          detail: M1 still says Binding.PairRevision, which PQ-2 replaced with hello-v2 Build. M3 still says the receipt succeeds once the signal is delivered, which PQ-4 replaced with waiting for a new Binding. Add a one-line pointer to the superseding revision on each affected milestone row.
          family: revision-supersedes-body-unmarked
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-10-09T23:03:02-07:00"
      agent: claude
      dispose:
        - id: PQ-7
          disposition: not-addressed
          note: 'Plan file unchanged (lines 63, 103, 123 still carry PairRevision and signal-delivered success). Fix by the family rule: any body text a Revisions entry replaces gets a pointer. That covers the DecideBinaryFreshness bullet, the M1 row, the ReloadContext integration row and bullet (pid file, replaced by PQ-1), the M3 row and its test line, and the M1/M2 Tests prose (replaced by PQ-5).'
          round: 4
      blocked: false
content_hash: f2bdb21c694a68452add7aeb5d4d51a103b8297508e8e220cc5154b267ff9613
---

# Gate ledger — pair#421 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-09T22:59:56-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `signal-target-by-verified-identity` reload-context signals via the pair-wrap pid file instead of the broker's verified Binding PID+Start
  agentcmd.RunRestart (agentcmd/restart.go:20-52) kills whatever pid the file names, which is fine in-slot but not cross-process from Couch: a stale or rewritten pid file or a mismatched PAIR_DATA_DIR sends SIGUSR2 (default action: terminate) to the wrong process. The broker already holds Binding.PID+Start for the exact incarnation (wrapcmd/peer_runtime.go:136). Signal that PID after re-checking Start, and make the shared helper take a verified incarnation (ARCH-SECURE, ARCH-ORDER).
- **PQ-2** [Minor] `unstated-contract-change` Adding PairRevision to couchmessage.Binding changes the broker actor map key and the wire contract unannounced
  Binding is the key of Broker.actors (broker.go:68) and is validated in model.go:32. State how an old Couch and a new wrapper (and the reverse) behave, or carry the revision outside the identity struct.
- **PQ-3** [Minor] `freshness-rule-underspecified` DecideBinaryFreshness compares vcs.revision only; vcs.modified builds and unknown build info have no stated rule
  Two dirty builds at the same revision compare equal, so a real fix gets refused as stale-binary. Consider comparing the Go build ID or a content hash, and state the rule for unknown build info.
- **PQ-4** [Minor] `uncertain-outcome-collapsed` reload-context receipt reports succeeded on signal delivery, collapsing an unconfirmed outcome into success
  The wrapper re-execs on SIGUSR2 and re-publishes its binding. Wait, with a time limit, for that evidence, or mark the receipt as unconfirmed.
- **PQ-5** [Minor] `test-prose-enumeration` M1 and M2 enumerate test cases in prose instead of one strategy line per risky function
  For DecideLiveRestart and DecideBinaryFreshness: property or table tests over the fact space. For the settle timer: an injected clock with interleaved activity.
- **PQ-6** [Minor] `no-stated-non-goals` Plan states no non-goals
  For example: no fleet or batch verb, no automatic make build, no durable receipts, no new check of Settled between admission and park.

## Round 2 — 2026-10-09T23:01:38-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Signals Binding.PID only after OSProcOps.Identity (procops.go:109) re-check equals Binding.Start (peer_runtime.go:136); pid file no longer used cross-process.
- PQ-2 — addressed — Binding (model.go:26) untouched; hello-v2 try-then-fallback relies on strictjson decode (session_protocol.go:70); four pairings tested.
- PQ-3 — addressed — Content-hash identity; modified and unknown-build rules stated.
- PQ-4 — addressed — Bounded 20s wait for a new Binding; otherwise unknown/unconfirmed.
- PQ-5 — addressed
- PQ-6 — addressed

## Round 3 — 2026-10-09T23:02:11-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed
- PQ-2 — addressed
- PQ-3 — addressed
- PQ-4 — addressed
- PQ-5 — addressed
- PQ-6 — addressed

### Raised

- **PQ-7** [Minor] `revision-supersedes-body-unmarked` Milestones M1 and M3 still describe designs that the round-1 Revisions replaced
  M1 still says Binding.PairRevision, which PQ-2 replaced with hello-v2 Build. M3 still says the receipt succeeds once the signal is delivered, which PQ-4 replaced with waiting for a new Binding. Add a one-line pointer to the superseding revision on each affected milestone row.

## Round 4 — 2026-10-09T23:03:02-07:00 (claude) — passed

### Disposed

- PQ-7 — not-addressed — Plan file unchanged (lines 63, 103, 123 still carry PairRevision and signal-delivered success). Fix by the family rule: any body text a Revisions entry replaces gets a pointer. That covers the DecideBinaryFreshness bullet, the M1 row, the ReloadContext integration row and bullet (pid file, replaced by PQ-1), the M3 row and its test line, and the M1/M2 Tests prose (replaced by PQ-5).

## Open findings

- **PQ-7** [Minor] `revision-supersedes-body-unmarked` Milestones M1 and M3 still describe designs that the round-1 Revisions replaced
