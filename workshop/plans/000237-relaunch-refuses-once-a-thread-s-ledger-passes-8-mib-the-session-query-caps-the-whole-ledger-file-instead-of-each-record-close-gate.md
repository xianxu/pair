---
gate: boundary-review
issue: 237
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-12T17:57:05-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: readJSONLArtifact duplicates the visitJSONLinesAt chunk loop; state the deliberate deltas (tolerant tail, no CR strip) or share a core
          detail: |-
            scan_helpers.go:162 vs scan_helpers.go:31. Acceptable for a bugfix, but the plan should say why two loops rather than one with a tolerant-tail option. ARCH-DRY.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: shared-helper-reuse
          round: 1
        - id: BR-2
          severity: Minor
          title: Plan removes the file cap without naming the new memory envelope or the polling caller that pays for unbounded ledger growth
          detail: |-
            The ledger is still read whole; titlepoller/runtime.go:117 calls QuerySession on a cadence. Name issue 238's row shrink as the bound and note the poller multiplier. ARCH-CONSTRAINTS.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: operating-envelope-unstated
          round: 1
        - id: BR-3
          severity: Minor
          title: No adversarial guard named for the new byte scanner; seed scan_fuzz_test.go with readJSONLArtifact
          detail: |-
            One strategy line: fuzz for equivalence to visitJSONLinesAt on terminated input and partial-tail passthrough on unterminated input, using FakeRuntime.ReadAt.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: risky-function-guard-missing
          round: 1
        - id: BR-4
          severity: Minor
          title: Caller roster omits titlepoller and contextcmd
          detail: |-
            cmd/internal/titlepoller/runtime.go:117 and cmd/internal/contextcmd/contextcmd.go:68 also call QuerySession; list them so the close review's sweep matches the code. ARCH-PURPOSE.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: consumer-enumeration-incomplete
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-12T17:57:05-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Minor
          title: Done-when "the operator's thread relaunches" is delegated to the operator, not verified in-range
          detail: The Log verifies the owner query resolves on the real 8,482,765-byte ledger (fixed vs control build), which is the mechanism, but the couch relaunch itself was not exercised. Say so in the close's --verified evidence rather than letting the Plan row read as fully delivered.
          family: done-when-operator-verified
          round: 2
        - id: BR-6
          severity: Minor
          title: OSRuntime.ReadAt opens the file once per 64 KiB chunk, so the ledger read now costs ~130 open/close cycles per query on the operator's ledger
          detail: 'runtime_os.go:163 re-opens per range. Pre-existing for transcripts, and #238 shrinks the file, but titlepoller multiplies it on a cadence. Note it in #238 or open the file once per frameJSONLArtifact call if it ever shows up in a profile. ARCH-CONSTRAINTS.'
          family: operating-envelope-unstated
          round: 2
        - id: BR-7
          severity: Minor
          title: TestQuerySessionEstablishesProofAfterCatalogLossWithoutTranscriptRead went from "ledger read exactly once" to "ledger read at least once"
          detail: query_test.go:63. The old ReadFile==1 pinned single-read; the new ReadAt!=0 would not catch a double read of the ledger. Asserting an exact chunk count (ceil(size/readChunkSize)+1) keeps the original claim.
          family: test-oracle-weakened
          round: 2
      blocked: false
---

# Gate ledger — pair#237 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T17:57:05-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `shared-helper-reuse` readJSONLArtifact duplicates the visitJSONLinesAt chunk loop; state the deliberate deltas (tolerant tail, no CR strip) or share a core
  scan_helpers.go:162 vs scan_helpers.go:31. Acceptable for a bugfix, but the plan should say why two loops rather than one with a tolerant-tail option. ARCH-DRY.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `operating-envelope-unstated` Plan removes the file cap without naming the new memory envelope or the polling caller that pays for unbounded ledger growth
  The ledger is still read whole; titlepoller/runtime.go:117 calls QuerySession on a cadence. Name issue 238's row shrink as the bound and note the poller multiplier. ARCH-CONSTRAINTS.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Minor] `risky-function-guard-missing` No adversarial guard named for the new byte scanner; seed scan_fuzz_test.go with readJSONLArtifact
  One strategy line: fuzz for equivalence to visitJSONLinesAt on terminated input and partial-tail passthrough on unterminated input, using FakeRuntime.ReadAt.
  (carried from plan-quality PQ-3, deferred to the boundary review)
- **BR-4** [Minor] `consumer-enumeration-incomplete` Caller roster omits titlepoller and contextcmd
  cmd/internal/titlepoller/runtime.go:117 and cmd/internal/contextcmd/contextcmd.go:68 also call QuerySession; list them so the close review's sweep matches the code. ARCH-PURPOSE.
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-09-12T17:57:05-07:00 (claude) — passed

### Raised

- **BR-5** [Minor] `done-when-operator-verified` Done-when "the operator's thread relaunches" is delegated to the operator, not verified in-range
  The Log verifies the owner query resolves on the real 8,482,765-byte ledger (fixed vs control build), which is the mechanism, but the couch relaunch itself was not exercised. Say so in the close's --verified evidence rather than letting the Plan row read as fully delivered.
- **BR-6** [Minor] `operating-envelope-unstated` OSRuntime.ReadAt opens the file once per 64 KiB chunk, so the ledger read now costs ~130 open/close cycles per query on the operator's ledger
  runtime_os.go:163 re-opens per range. Pre-existing for transcripts, and #238 shrinks the file, but titlepoller multiplies it on a cadence. Note it in #238 or open the file once per frameJSONLArtifact call if it ever shows up in a profile. ARCH-CONSTRAINTS.
- **BR-7** [Minor] `test-oracle-weakened` TestQuerySessionEstablishesProofAfterCatalogLossWithoutTranscriptRead went from "ledger read exactly once" to "ledger read at least once"
  query_test.go:63. The old ReadFile==1 pinned single-read; the new ReadAt!=0 would not catch a double read of the ledger. Asserting an exact chunk count (ceil(size/readChunkSize)+1) keeps the original claim.

## Open findings

- **BR-1** [Minor] `shared-helper-reuse` readJSONLArtifact duplicates the visitJSONLinesAt chunk loop; state the deliberate deltas (tolerant tail, no CR strip) or share a core
- **BR-2** [Minor] `operating-envelope-unstated` Plan removes the file cap without naming the new memory envelope or the polling caller that pays for unbounded ledger growth
- **BR-3** [Minor] `risky-function-guard-missing` No adversarial guard named for the new byte scanner; seed scan_fuzz_test.go with readJSONLArtifact
- **BR-4** [Minor] `consumer-enumeration-incomplete` Caller roster omits titlepoller and contextcmd
- **BR-5** [Minor] `done-when-operator-verified` Done-when "the operator's thread relaunches" is delegated to the operator, not verified in-range
- **BR-6** [Minor] `operating-envelope-unstated` OSRuntime.ReadAt opens the file once per 64 KiB chunk, so the ledger read now costs ~130 open/close cycles per query on the operator's ledger
- **BR-7** [Minor] `test-oracle-weakened` TestQuerySessionEstablishesProofAfterCatalogLossWithoutTranscriptRead went from "ledger read exactly once" to "ledger read at least once"
