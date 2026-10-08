---
gate: plan-quality
issue: 409
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-07T21:56:27-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: M1 test relies on a "shortened WriteTimeout", but WriteTimeout is a const and stalledParent is unexported in package terminal
          detail: profile.go:23 declares WriteTimeout in a const block; presenter_stall_test.go:203-206 waits the real 5s+. Either accept a ~6s real-PTY test (and name which package hosts the console-to-crashreport test), or declare an injectable write budget as an explicit presenter seam change.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-2
          severity: Important
          title: changedPlainRows and the Emit fast path are tested by a hand-listed case list instead of a generative check against the full rebuild
          detail: 'Equality with the full rebuild is the whole correctness argument (ARCH-PURPOSE note). Replace the 2.1/2.2 case lists with one strategy line: seeded random frame pairs (cell mutations, wrap-flag flips, wide chars at the right edge, full-width rows) emitted both ways, with identical viewport, wrap flags and history under the xterm oracle.'
          family: enumerated-tests-instead-of-strategy
          round: 1
        - id: PQ-3
          severity: Minor
          title: 'The issue says the exit leaves "no message", but teardown already prints couch: terminal: err to errw'
          detail: console.go:982-984 writes the joined failure to stderr; it is lost on the broken terminal. The fix still stands, but the claim should be corrected.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-4
          severity: Minor
          title: ExitReason hardcodes "5s"; derive it from WriteTimeout
          family: single-source-wording
          round: 1
        - id: PQ-5
          severity: Minor
          title: Plan quality notes omit ARCH-ORDER and ARCH-FUNERAL
          detail: 'One line each: presenter failure is terminal, so no partial-paint state reaches the row diff (ORDER); the recorded exit reuses #397''s per-run file, rename and sweep, so nothing new is created (FUNERAL).'
          family: arch-lens-unaddressed
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-07T21:57:23-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Split into the existing real-5s stall test plus a timing-free couchcmd test of recordConsoleExit; presenter seam left unchanged on purpose.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Seeded generative TestHistoryRowDiffEqualsFullRebuild under runHistoryOracle; refusal branch must be byte-identical to the full rebuild.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Corrected with a console.go:982 citation; message goes to the stalled terminal and is lost.
          round: 2
        - id: PQ-4
          disposition: addressed
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-07T21:58:21-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
content_hash: 626e0ff80f75c56ebdf6979f46cbe17b527530aed5a51dc425de65051a694b1e
---

# Gate ledger — pair#409 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T21:56:27-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unbacked-existing-behavior-claim` M1 test relies on a "shortened WriteTimeout", but WriteTimeout is a const and stalledParent is unexported in package terminal
  profile.go:23 declares WriteTimeout in a const block; presenter_stall_test.go:203-206 waits the real 5s+. Either accept a ~6s real-PTY test (and name which package hosts the console-to-crashreport test), or declare an injectable write budget as an explicit presenter seam change.
- **PQ-2** [Important] `enumerated-tests-instead-of-strategy` changedPlainRows and the Emit fast path are tested by a hand-listed case list instead of a generative check against the full rebuild
  Equality with the full rebuild is the whole correctness argument (ARCH-PURPOSE note). Replace the 2.1/2.2 case lists with one strategy line: seeded random frame pairs (cell mutations, wrap-flag flips, wide chars at the right edge, full-width rows) emitted both ways, with identical viewport, wrap flags and history under the xterm oracle.
- **PQ-3** [Minor] `unbacked-existing-behavior-claim` The issue says the exit leaves "no message", but teardown already prints couch: terminal: err to errw
  console.go:982-984 writes the joined failure to stderr; it is lost on the broken terminal. The fix still stands, but the claim should be corrected.
- **PQ-4** [Minor] `single-source-wording` ExitReason hardcodes "5s"; derive it from WriteTimeout
- **PQ-5** [Minor] `arch-lens-unaddressed` Plan quality notes omit ARCH-ORDER and ARCH-FUNERAL
  One line each: presenter failure is terminal, so no partial-paint state reaches the row diff (ORDER); the recorded exit reuses #397's per-run file, rename and sweep, so nothing new is created (FUNERAL).

## Round 2 — 2026-10-07T21:57:23-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Split into the existing real-5s stall test plus a timing-free couchcmd test of recordConsoleExit; presenter seam left unchanged on purpose.
- PQ-2 — addressed — Seeded generative TestHistoryRowDiffEqualsFullRebuild under runHistoryOracle; refusal branch must be byte-identical to the full rebuild.
- PQ-3 — addressed — Corrected with a console.go:982 citation; message goes to the stalled terminal and is lost.
- PQ-4 — addressed
- PQ-5 — addressed

## Round 3 — 2026-10-07T21:58:21-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)
