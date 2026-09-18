---
gate: plan-quality
issue: 282
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-18T11:35:07-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: DecodeOuterRecord, the only untrusted-input parser here, is pinned by six literal cases instead of a fuzz
          detail: |-
            The record is truncatable, hand-editable and cross-version; one FuzzDecodeOuterRecord seeded with the
            legacy one-line form plus the listed malformed strings, asserting no panic and Couch==true only for an
            exact two-line presenter=couch record, covers the class the table cannot. In the same paragraph, the
            ARCH-SECURE claim "a hand edit can never fabricate a Couch section" is false: a well-formed hand-written
            two-line record does exactly that. What strictness buys is visible degradation on a MALFORMED record.
          family: parser-fuzz-over-enumerated-cases
          round: 1
        - id: PQ-2
          severity: Minor
          title: Task 3 asserts hosting changes only Alt+d/Alt+n/Ctrl+Alt+n; compaction.go:90 also reroutes Alt+Shift+C
          detail: |-
            When hosted, runCompaction delegates the continuation to Couch instead of restarting in place. No help row
            is wrong, because Alt+Shift+C's wording comes from nvim's "compact session" desc and ChordAltShiftC is not
            a GlobalBinding, so HostedHelp structurally cannot carry it. Say that, rather than claiming sameness.
          family: unbacked-behavior-claim
          round: 1
        - id: PQ-3
          severity: Minor
          title: The doc sweep's line-numbered anchors are invalidated by the sweep's own earlier edits
          detail: |-
            Task 8 Step 2 replaces README 378-381 with a shorter passage before the implementer reaches the 496-497
            anchor, so the later citations drift. Content anchors survive edits; line numbers do not.
          family: stale-line-number-anchors
          round: 1
      blocked: false
content_hash: 2fc894f406d25ffb2797a6fa0217201da76a7e3127a1582a3529eb0ac4e9d9ce
---

# Gate ledger — pair#282 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T11:35:07-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `parser-fuzz-over-enumerated-cases` DecodeOuterRecord, the only untrusted-input parser here, is pinned by six literal cases instead of a fuzz
  The record is truncatable, hand-editable and cross-version; one FuzzDecodeOuterRecord seeded with the
  legacy one-line form plus the listed malformed strings, asserting no panic and Couch==true only for an
  exact two-line presenter=couch record, covers the class the table cannot. In the same paragraph, the
  ARCH-SECURE claim "a hand edit can never fabricate a Couch section" is false: a well-formed hand-written
  two-line record does exactly that. What strictness buys is visible degradation on a MALFORMED record.
- **PQ-2** [Minor] `unbacked-behavior-claim` Task 3 asserts hosting changes only Alt+d/Alt+n/Ctrl+Alt+n; compaction.go:90 also reroutes Alt+Shift+C
  When hosted, runCompaction delegates the continuation to Couch instead of restarting in place. No help row
  is wrong, because Alt+Shift+C's wording comes from nvim's "compact session" desc and ChordAltShiftC is not
  a GlobalBinding, so HostedHelp structurally cannot carry it. Say that, rather than claiming sameness.
- **PQ-3** [Minor] `stale-line-number-anchors` The doc sweep's line-numbered anchors are invalidated by the sweep's own earlier edits
  Task 8 Step 2 replaces README 378-381 with a shorter passage before the implementer reaches the 496-497
  anchor, so the later citations drift. Content anchors survive edits; line numbers do not.

## Open findings

- **PQ-1** [Minor] `parser-fuzz-over-enumerated-cases` DecodeOuterRecord, the only untrusted-input parser here, is pinned by six literal cases instead of a fuzz
- **PQ-2** [Minor] `unbacked-behavior-claim` Task 3 asserts hosting changes only Alt+d/Alt+n/Ctrl+Alt+n; compaction.go:90 also reroutes Alt+Shift+C
- **PQ-3** [Minor] `stale-line-number-anchors` The doc sweep's line-numbered anchors are invalidated by the sweep's own earlier edits
