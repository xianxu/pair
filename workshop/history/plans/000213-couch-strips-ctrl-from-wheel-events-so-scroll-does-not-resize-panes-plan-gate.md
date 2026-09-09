---
gate: plan-quality
issue: 213
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-09T07:54:25-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Wheel predicate stated as "buttons 64/65" will never match ctrl+wheel; modifier decoding belongs in mouseinput
          detail: |-
            "wheel buttons 64/65 only" compiles most naturally to
            `Button == mouseinput.WheelUp || Button == mouseinput.WheelDown`, which
            never matches 80/81 and strips nothing. State the predicate on the
            modifier-masked base button, and put the mask alongside WheelUp/WheelDown
            in mouseinput (mouseinput.go:33-36) — which bit means ctrl is wire-format
            knowledge, and the Revisions split already committed format knowledge to
            that package (ARCH-DRY).
          family: format-knowledge-ownership
          round: 1
        - id: PQ-2
          severity: Important
          title: Nothing automated proves the strip is wired into onMouse's forward path
          detail: |-
            Both new functions are unit-tested pure, but a call whose results are
            discarded, or placed in the wrong branch, compiles and leaves every planned
            test green. console_mouse_test.go:218 TestForwardPreservesRawBytes is a
            ten-line template against an existing fixture: write \x1b[<80;7;9M to the
            host pipe, assert child.Writes() contains \x1b[<64;7;9M. Add it, so the
            Done-when is not carried by the manual step alone.
          family: wiring-unproven-by-test
          round: 1
        - id: PQ-3
          severity: Minor
          title: Tests bullet enumerates cases in prose; compress to one strategy line per risky function
          detail: |-
            The four couchtty cases restate Done-when and will be rewritten as code
            immediately. Replace with a strategy line each. For WithButton the risky
            class is arbitrary bytes, not the listed shapes: fuzz seeded with malformed
            forms, property = Parse(WithButton(raw,b)) equals Parse(raw) with only
            Button differing and all other bytes identical — which also covers absurd
            button arguments the enumeration is blind to.
          family: test-plan-enumeration
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-09T07:56:43-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Predicate now on BaseButton(event.Button); ModMask/ModCtrl placed beside WheelUp/WheelDown in mouseinput.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Wiring test added against the real newMouseFixture/onMouse path; the cited template at console_mouse_test.go:218 exists and fits.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Enumeration replaced by a fuzz property for WithButton and a modifier cross-product for the strip, with raw-byte assertions.
          round: 2
      blocked: false
content_hash: 41a0d62b9ac436a41d090e93e510b2e3ef2ace11ba1cd754122ac4f44c667805
---

# Gate ledger — pair#213 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T07:54:25-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `format-knowledge-ownership` Wheel predicate stated as "buttons 64/65" will never match ctrl+wheel; modifier decoding belongs in mouseinput
  "wheel buttons 64/65 only" compiles most naturally to
  `Button == mouseinput.WheelUp || Button == mouseinput.WheelDown`, which
  never matches 80/81 and strips nothing. State the predicate on the
  modifier-masked base button, and put the mask alongside WheelUp/WheelDown
  in mouseinput (mouseinput.go:33-36) — which bit means ctrl is wire-format
  knowledge, and the Revisions split already committed format knowledge to
  that package (ARCH-DRY).
- **PQ-2** [Important] `wiring-unproven-by-test` Nothing automated proves the strip is wired into onMouse's forward path
  Both new functions are unit-tested pure, but a call whose results are
  discarded, or placed in the wrong branch, compiles and leaves every planned
  test green. console_mouse_test.go:218 TestForwardPreservesRawBytes is a
  ten-line template against an existing fixture: write \x1b[<80;7;9M to the
  host pipe, assert child.Writes() contains \x1b[<64;7;9M. Add it, so the
  Done-when is not carried by the manual step alone.
- **PQ-3** [Minor] `test-plan-enumeration` Tests bullet enumerates cases in prose; compress to one strategy line per risky function
  The four couchtty cases restate Done-when and will be rewritten as code
  immediately. Replace with a strategy line each. For WithButton the risky
  class is arbitrary bytes, not the listed shapes: fuzz seeded with malformed
  forms, property = Parse(WithButton(raw,b)) equals Parse(raw) with only
  Button differing and all other bytes identical — which also covers absurd
  button arguments the enumeration is blind to.

## Round 2 — 2026-09-09T07:56:43-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Predicate now on BaseButton(event.Button); ModMask/ModCtrl placed beside WheelUp/WheelDown in mouseinput.
- PQ-2 — addressed — Wiring test added against the real newMouseFixture/onMouse path; the cited template at console_mouse_test.go:218 exists and fits.
- PQ-3 — addressed — Enumeration replaced by a fuzz property for WithButton and a modifier cross-product for the strip, with raw-byte assertions.

## Open findings

(none — every finding has been disposed)
