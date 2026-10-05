---
gate: plan-quality
issue: 367
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-04T00:28:11-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: Tasks 2.4/2.5 enumerate individual test cases in prose instead of one strategy line per risky function
          detail: The poll loop over transport faults (table-driven fault injection) and the --resume/--reboot argv parser (fuzz or generated malformed argv) should each be one strategy line; the long case lists will be rewritten as code and go stale.
          family: plan-enumerates-test-cases
          round: 1
        - id: PQ-2
          severity: Minor
          title: The live sdlc conformance test can be skipped at every close; nothing makes it run on a schedule
          detail: Chunk 3 step 4 allows accepting the skip. Record at each milestone close whether it ran unsandboxed, so drift between FakeFleetSDLC and real sdlc is detected (ARCH-MOCK).
          family: live-conformance-cadence-unstated
          round: 1
        - id: PQ-3
          severity: Minor
          title: A remote resume is recognized by Operation/Attempt/ContinuationID shape rather than an explicit origin
          detail: Task 2.3 clears reattach-failure for any resume with Attempt 0 and no ContinuationID. A future origin with that shape would match silently. The plan pins both sides with a test; recheck at the close review (ARCH-ORDER).
          family: origin-recognized-by-field-shape
          round: 1
      blocked: false
    - "n": 2
      timestamp: "2026-10-04T00:29:00-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Tasks 2.4/2.5 still enumerate cases; Minor, carried to close review.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Skip is recorded but no milestone requires a live run; close review should check --verified for one.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Deliberate shape-based recognition pinned by a two-sided test; recheck at close review.
          round: 2
      blocked: false
content_hash: a739865e7407cdec1b00cee2f35f03a6eb89a444e7ddf2d21c6c85844db2f7f1
---

# Gate ledger — pair#367 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-04T00:28:11-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `plan-enumerates-test-cases` Tasks 2.4/2.5 enumerate individual test cases in prose instead of one strategy line per risky function
  The poll loop over transport faults (table-driven fault injection) and the --resume/--reboot argv parser (fuzz or generated malformed argv) should each be one strategy line; the long case lists will be rewritten as code and go stale.
- **PQ-2** [Minor] `live-conformance-cadence-unstated` The live sdlc conformance test can be skipped at every close; nothing makes it run on a schedule
  Chunk 3 step 4 allows accepting the skip. Record at each milestone close whether it ran unsandboxed, so drift between FakeFleetSDLC and real sdlc is detected (ARCH-MOCK).
- **PQ-3** [Minor] `origin-recognized-by-field-shape` A remote resume is recognized by Operation/Attempt/ContinuationID shape rather than an explicit origin
  Task 2.3 clears reattach-failure for any resume with Attempt 0 and no ContinuationID. A future origin with that shape would match silently. The plan pins both sides with a test; recheck at the close review (ARCH-ORDER).

## Round 2 — 2026-10-04T00:29:00-07:00 (claude) — passed

### Disposed

- PQ-1 — not-addressed — Tasks 2.4/2.5 still enumerate cases; Minor, carried to close review.
- PQ-2 — not-addressed — Skip is recorded but no milestone requires a live run; close review should check --verified for one.
- PQ-3 — not-addressed — Deliberate shape-based recognition pinned by a two-sided test; recheck at close review.

## Open findings

- **PQ-1** [Minor] `plan-enumerates-test-cases` Tasks 2.4/2.5 enumerate individual test cases in prose instead of one strategy line per risky function
- **PQ-2** [Minor] `live-conformance-cadence-unstated` The live sdlc conformance test can be skipped at every close; nothing makes it run on a schedule
- **PQ-3** [Minor] `origin-recognized-by-field-shape` A remote resume is recognized by Operation/Attempt/ContinuationID shape rather than an explicit origin
