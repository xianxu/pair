---
gate: plan-quality
issue: 262
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-17T19:18:17-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Task 4 corrects hostty/control.go:27, but the stale claim is at hostty/reserve.go:13-18
          detail: |-
            control.go:24-27 only says ResetRegion is written on teardown, which is still true for
            Reservation.Release. The false "\x1b[r lives here and only here" sentence is reserve.go:18,
            next to an equally stale "shared by two consumers -- couch ... pair term ..." claim (both
            migrated to UpdateChrome in #255 M3; only cmd/probes/couchnestedrows calls the painters).
            Retarget Task 4, the Spec sentence and the Done-when bullet, and fix both sentences.
          family: unverified-existing-code-claim
          round: 1
        - id: PQ-2
          severity: Minor
          title: The xterm oracle pinned at headless 5.5.0 does not implement mode 2026
          detail: |-
            grep 2026 over tests/terminal-oracle/node_modules/@xterm/headless returns nothing, so a
            green oracle run proves the ignoring-terminal path only. Say that in Task 2 Step 4, and run
            the existing native zellij oracle (PAIR_TERMINAL_NATIVE=1, TestHistoryWireNativeOracle) once
            as the model of pair term's real parent.
          family: oracle-lacks-behavior-under-test
          round: 1
        - id: PQ-3
          severity: Minor
          title: The failed-prefix sweep only covers a single-write frame, not the multi-write shapes
          detail: |-
            Alt-enter frames write syncBegin, then the 1049 packet, then the body; a >64 KiB frame writes
            several chunks. The "syncBegin already fully accepted, a later write fails" case is the one
            ARCH-ORDER reasons about and is not swept. Sweep cumulative offsets with an alt fixture.
          family: sequence-sweep-covers-one-shape
          round: 1
        - id: PQ-4
          severity: Minor
          title: '482 plan lines for a ~20-line change: full test bodies and a reproduced implementation'
          detail: |-
            The required content (functions under test plus one strategy line per risky function) is
            already present. The verbatim test bodies and the inline cursorEpilogue implementation will
            be rewritten within the hour. Noting the tension with superpowers-writing-plans' convention.
          family: plan-restates-the-diff
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-17T19:23:30-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Retargeted to reserve.go:13-18 in Task 4, Spec and Done-when; the sibling grep still misses couchtty/reserve.go:15-18 ("lives in one package only") and couchnestedrows/main.go:18 — sweep the claim, not just the symbols.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Integration-points note states xterm 5.5.0 lacks 2026; Task 2 adds the PAIR_TERMINAL_NATIVE=1 run and logs if zellij cannot run.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 3 cuts every write of single-write, alt-screen and multi-chunk frames via recordingParent over ttyio.Fake WriteStep.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Plan compressed to 314 lines of per-test specifications; verbatim bodies and inline implementation removed.
          round: 2
      blocked: false
content_hash: fb380918e87fc7fe46ee12aeceeb801bac667b63b702f0518a4d57133eb62ebe
---

# Gate ledger — pair#262 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T19:18:17-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unverified-existing-code-claim` Task 4 corrects hostty/control.go:27, but the stale claim is at hostty/reserve.go:13-18
  control.go:24-27 only says ResetRegion is written on teardown, which is still true for
  Reservation.Release. The false "\x1b[r lives here and only here" sentence is reserve.go:18,
  next to an equally stale "shared by two consumers -- couch ... pair term ..." claim (both
  migrated to UpdateChrome in #255 M3; only cmd/probes/couchnestedrows calls the painters).
  Retarget Task 4, the Spec sentence and the Done-when bullet, and fix both sentences.
- **PQ-2** [Minor] `oracle-lacks-behavior-under-test` The xterm oracle pinned at headless 5.5.0 does not implement mode 2026
  grep 2026 over tests/terminal-oracle/node_modules/@xterm/headless returns nothing, so a
  green oracle run proves the ignoring-terminal path only. Say that in Task 2 Step 4, and run
  the existing native zellij oracle (PAIR_TERMINAL_NATIVE=1, TestHistoryWireNativeOracle) once
  as the model of pair term's real parent.
- **PQ-3** [Minor] `sequence-sweep-covers-one-shape` The failed-prefix sweep only covers a single-write frame, not the multi-write shapes
  Alt-enter frames write syncBegin, then the 1049 packet, then the body; a >64 KiB frame writes
  several chunks. The "syncBegin already fully accepted, a later write fails" case is the one
  ARCH-ORDER reasons about and is not swept. Sweep cumulative offsets with an alt fixture.
- **PQ-4** [Minor] `plan-restates-the-diff` 482 plan lines for a ~20-line change: full test bodies and a reproduced implementation
  The required content (functions under test plus one strategy line per risky function) is
  already present. The verbatim test bodies and the inline cursorEpilogue implementation will
  be rewritten within the hour. Noting the tension with superpowers-writing-plans' convention.

## Round 2 — 2026-09-17T19:23:30-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Retargeted to reserve.go:13-18 in Task 4, Spec and Done-when; the sibling grep still misses couchtty/reserve.go:15-18 ("lives in one package only") and couchnestedrows/main.go:18 — sweep the claim, not just the symbols.
- PQ-2 — addressed — Integration-points note states xterm 5.5.0 lacks 2026; Task 2 adds the PAIR_TERMINAL_NATIVE=1 run and logs if zellij cannot run.
- PQ-3 — addressed — Task 3 cuts every write of single-write, alt-screen and multi-chunk frames via recordingParent over ttyio.Fake WriteStep.
- PQ-4 — addressed — Plan compressed to 314 lines of per-test specifications; verbatim bodies and inline implementation removed.

## Open findings

(none — every finding has been disposed)
