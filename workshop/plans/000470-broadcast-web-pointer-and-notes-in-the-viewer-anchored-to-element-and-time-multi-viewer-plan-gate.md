---
gate: plan-quality
issue: 470
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T03:00:49-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Board validates seg/node under its lock, then emits outside it; a navigation in between files the note into the new document
          detail: 'The reducer assigns a note the segment current when the note record is reduced (reduce.go:205), and recording.Note has no segment of its own (Header is T,K only, record.go:97). With emit outside the lock into the recorder queue, a Doc record ordered between check and emit makes a stale-node note live on the new page, so ErrStale is unenforceable. State the fix: the writer re-checks or drops at ordering time, or the note carries its checked segment and the reducer and viewers filter on it, or the race is accepted explicitly. Also state how pending is cleared when emit fails or the record is never seen back, or it counts toward MaxNotesPerColor until the board closes.'
          family: validate-then-emit-ordering
          round: 1
        - id: PQ-2
          severity: Minor
          title: Operator acts as author id "operator", which the viewer id regex also accepts, so viewers can impersonate the operator
          detail: Reserve "operator" (or a prefix that viewers can't use) in ParseNote/ParseNoteState/ParseStroke.
          family: identity-namespace-reservation
          round: 1
        - id: PQ-3
          severity: Minor
          title: M1 lists test cases in prose instead of one strategy line per risky function
          detail: Replace it with e.g. "Parse* over untrusted bodies → fuzz, seeded with oversize, NaN/Inf, invalid UTF-8 and unknown fields", and one line for the Board's concurrent color-limit check.
          family: test-plan-enumeration
          round: 1
        - id: PQ-4
          severity: Minor
          title: 'Frontmatter deps is empty although Done-when depends on #458 (the recorder) landing'
          family: undeclared-dep
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T03:01:45-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Emit carries the checked doc; the writer re-checks it with board.OnDoc when it writes; pending entries end on seen, emit error, or a new document; the race has its own test.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The id operator is reserved, and the parsers refuse it as a viewer id.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: M1 now gives one test-strategy line per risky function, with fuzz seeds and a -race test for the color limit.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: deps is now [000458].
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T03:02:49-07:00"
      agent: claude
      blocked: false
content_hash: c62f0ba778d72481f589c75a285c847bfd33a480ac4784182910e30beae452cd
---

# Gate ledger — pair#470 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T03:00:49-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `validate-then-emit-ordering` Board validates seg/node under its lock, then emits outside it; a navigation in between files the note into the new document
  The reducer assigns a note the segment current when the note record is reduced (reduce.go:205), and recording.Note has no segment of its own (Header is T,K only, record.go:97). With emit outside the lock into the recorder queue, a Doc record ordered between check and emit makes a stale-node note live on the new page, so ErrStale is unenforceable. State the fix: the writer re-checks or drops at ordering time, or the note carries its checked segment and the reducer and viewers filter on it, or the race is accepted explicitly. Also state how pending is cleared when emit fails or the record is never seen back, or it counts toward MaxNotesPerColor until the board closes.
- **PQ-2** [Minor] `identity-namespace-reservation` Operator acts as author id "operator", which the viewer id regex also accepts, so viewers can impersonate the operator
  Reserve "operator" (or a prefix that viewers can't use) in ParseNote/ParseNoteState/ParseStroke.
- **PQ-3** [Minor] `test-plan-enumeration` M1 lists test cases in prose instead of one strategy line per risky function
  Replace it with e.g. "Parse* over untrusted bodies → fuzz, seeded with oversize, NaN/Inf, invalid UTF-8 and unknown fields", and one line for the Board's concurrent color-limit check.
- **PQ-4** [Minor] `undeclared-dep` Frontmatter deps is empty although Done-when depends on #458 (the recorder) landing

## Round 2 — 2026-10-12T03:01:45-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Emit carries the checked doc; the writer re-checks it with board.OnDoc when it writes; pending entries end on seen, emit error, or a new document; the race has its own test.
- PQ-2 — addressed — The id operator is reserved, and the parsers refuse it as a viewer id.
- PQ-3 — addressed — M1 now gives one test-strategy line per risky function, with fuzz seeds and a -race test for the color limit.
- PQ-4 — addressed — deps is now [000458].

## Round 3 — 2026-10-12T03:02:49-07:00 (claude) — passed

## Open findings

(none — every finding has been disposed)
