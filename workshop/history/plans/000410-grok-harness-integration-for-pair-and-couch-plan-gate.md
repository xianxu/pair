---
gate: plan-quality
issue: 410
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-07T23:05:57-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: runGrok residue cleanup deletes a dir keyed by grok's JSON sessionId with no validation or path confinement
          detail: 'State: parse sessionId with uuidPattern, build only <grok-sessions>/<encode(cmd.Dir)>/<uuid>, refuse anything resolving outside it, and leave the residue (logged) on a missing or malformed ID. Task 9''s fake-binary test feeds a hostile sessionId.'
          family: untrusted-input-drives-destructive-op
          round: 1
        - id: PQ-2
          severity: Minor
          title: Task 1 known-gap message names switch-agent orientation, which the parity test does not probe
          family: known-gap-names-unprobed-sites
          round: 1
        - id: PQ-3
          severity: Minor
          title: M1 says Couch can launch grok, but ledgerRejectsAgent makes each Couch grok launch record malformed until M2
          detail: Verify Couch tolerates malformed ledger lines in the M1 live check, or defer Couch launches to M2.
          family: unverified-interim-claim
          round: 1
        - id: PQ-4
          severity: Minor
          title: persistStripFlags has no named home (resumeform is the shared one) and omits -s, which also bypasses shouldMintSessionID
          family: shared-table-home-unstated
          round: 1
        - id: PQ-5
          severity: Minor
          title: Tasks 2, 3 and 7 list test rows in prose; replace with one adversarial-strategy line per risky function
          family: enumerated-test-prose
          round: 1
        - id: PQ-6
          severity: Minor
          title: Unlisted ACP sessionUpdate kinds become near-miss drift; chunk-joining state location is unstated
          family: drift-signal-overbroad
          round: 1
        - id: PQ-7
          severity: Minor
          title: Scanner fixture cwd dir is -repo, but real grok dirs are URL-encoded (%2FUsers%2F...)
          family: fixture-shape-fidelity
          round: 1
        - id: PQ-8
          severity: Minor
          title: No ARCH-CONSTRAINTS envelope for updates.jsonl growth or grok -p slug latency
          family: operating-envelope-unstated
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-07T23:07:24-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: UUID-validated ID, target built from cmd.Dir, parent-equality + Lstat checks, residue left on refusal, hostile-ID fake-binary test.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Known-gap message now names only probed sites; orientation interim lives in Spec.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 6 adds a live Couch malformed-ledger check with a stated fallback (move registry row to M2).
          round: 2
        - id: PQ-4
          disposition: addressed
          note: resumeform gains SessionID/Continue groups; shouldMintSessionID reads HasSessionID so -s suppresses mint.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Tasks 2, 3, 7 now state one adversarial strategy per risky function.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: grokIgnoredKinds covers documented ACP kinds; chunk-join state lives in NativeEventsFromRecords with held-open flush rule.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Fixture dir is now %2Frepo.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Operating-envelope section added (shared framing, Request.timeout, measured ~2 s).
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-07T23:08:30-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
content_hash: f5a26ce1587cf3ace090fcf4137d3a97088d5fc048abf67ad1cb9138a2ddd6b6
---

# Gate ledger — pair#410 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T23:05:57-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `untrusted-input-drives-destructive-op` runGrok residue cleanup deletes a dir keyed by grok's JSON sessionId with no validation or path confinement
  State: parse sessionId with uuidPattern, build only <grok-sessions>/<encode(cmd.Dir)>/<uuid>, refuse anything resolving outside it, and leave the residue (logged) on a missing or malformed ID. Task 9's fake-binary test feeds a hostile sessionId.
- **PQ-2** [Minor] `known-gap-names-unprobed-sites` Task 1 known-gap message names switch-agent orientation, which the parity test does not probe
- **PQ-3** [Minor] `unverified-interim-claim` M1 says Couch can launch grok, but ledgerRejectsAgent makes each Couch grok launch record malformed until M2
  Verify Couch tolerates malformed ledger lines in the M1 live check, or defer Couch launches to M2.
- **PQ-4** [Minor] `shared-table-home-unstated` persistStripFlags has no named home (resumeform is the shared one) and omits -s, which also bypasses shouldMintSessionID
- **PQ-5** [Minor] `enumerated-test-prose` Tasks 2, 3 and 7 list test rows in prose; replace with one adversarial-strategy line per risky function
- **PQ-6** [Minor] `drift-signal-overbroad` Unlisted ACP sessionUpdate kinds become near-miss drift; chunk-joining state location is unstated
- **PQ-7** [Minor] `fixture-shape-fidelity` Scanner fixture cwd dir is -repo, but real grok dirs are URL-encoded (%2FUsers%2F...)
- **PQ-8** [Minor] `operating-envelope-unstated` No ARCH-CONSTRAINTS envelope for updates.jsonl growth or grok -p slug latency

## Round 2 — 2026-10-07T23:07:24-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — UUID-validated ID, target built from cmd.Dir, parent-equality + Lstat checks, residue left on refusal, hostile-ID fake-binary test.
- PQ-2 — addressed — Known-gap message now names only probed sites; orientation interim lives in Spec.
- PQ-3 — addressed — Task 6 adds a live Couch malformed-ledger check with a stated fallback (move registry row to M2).
- PQ-4 — addressed — resumeform gains SessionID/Continue groups; shouldMintSessionID reads HasSessionID so -s suppresses mint.
- PQ-5 — addressed — Tasks 2, 3, 7 now state one adversarial strategy per risky function.
- PQ-6 — addressed — grokIgnoredKinds covers documented ACP kinds; chunk-join state lives in NativeEventsFromRecords with held-open flush rule.
- PQ-7 — addressed — Fixture dir is now %2Frepo.
- PQ-8 — addressed — Operating-envelope section added (shared framing, Request.timeout, measured ~2 s).

## Round 3 — 2026-10-07T23:08:30-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)
