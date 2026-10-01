---
gate: boundary-review
issue: 373
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-01T15:57:22-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Emulator teardown races with the reply drain
          detail: terminal_pressure_test.go:158,203 starts io.Copy on the emulator and closes it before joining the reader; Emulator.Read and Close access closed without synchronization. The focused race control failed at emulator.go:288,301. This shared pair is the only instance in the window and affects every trial. Use concurrency-safe draining/shutdown and verify with the race detector.
          family: concurrent-resource-shutdown
          round: 1
        - id: BR-2
          severity: Important
          title: Blocking trial operations escape the promised deadlines
          detail: 'terminal_pressure_test.go:152 uses cancellation without a deadline; input writes at 371,411,415 and synchronous observations at 333–396 can prevent timeout checks and cleanup from running. Recovery waits at 414,418,419,427 receive separate budgets rather than one five-second deadline. ARCH-PURPOSE: enforce an absolute deadline across these paths, release blocked operations on cancellation, join helpers and test stalled execution.'
          family: experiment-deadline-enforcement
          round: 1
        - id: BR-3
          severity: Important
          title: Recovery is reported without proving all PTY output arrived
          detail: terminal_pressure_test.go:419–447 treats side-channel producer completion plus queued-publication flush as output completion, although unread PTY bytes can remain. Final progress checks only child zero and accepts any sequence; emitted and ingested totals are only logged at 453. Require per-child terminal completion evidence and final presentation before reporting recovery, with a regression case delaying trailing PTY output.
          family: completion-evidence-before-success
          round: 1
        - id: BR-4
          severity: Important
          title: README omits the new pressure experiment invocation
          detail: atlas/couch.md documents PAIR_COUCH_PRESSURE=1 and TestCouchOutputPressure, but README.md is unchanged despite its existing terminal-testing section at 1076. Add the command and scope limitations there. This is the sole new operator-facing invocation family; PAIR_COUCH_PRESSURE_CHILD is an internal helper.
          family: readme-surface-discovery
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-01T16:06:11-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: terminal_pressure_test.go:230–251 closes the reply pipe and joins readers/writers before Emulator.Close. Focused race tests passed three repetitions; a scratch mutation restoring early Emulator.Close triggered race failures.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Operations use pressureAwait, recovery shares window-end-plus-five-second cancellation, and cleanup closes resources before joining workers. The stalled-operation regression passed; removing pressureAwait's cancellation branch made it time out.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: terminal_pressure_test.go:489–498 checks marker absence and releases trailing output before invoking recovery checks. Removing lines 500–524 in a scratch Go overlay left TestCouchPressureTrailingPTYOutput passing three repetitions, including one reporting ingested_raw=2061 instead of the required 2087 bytes. The implementation adds completion checks, but the required fail-without-fix regression is missing. Exercise the actual recovery operation while trailing output remains withheld and assert it cannot report success.
          round: 2
        - id: BR-4
          disposition: addressed
          note: README.md:1090–1101 now documents the opt-in command and limitations. The invocation matches TestCouchOutputPressure's environment guard and 2×4×3 matrix; atlas/couch.md:2207–2218 documents the same surface.
          round: 2
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — pair#373 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T15:57:22-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `concurrent-resource-shutdown` Emulator teardown races with the reply drain
  terminal_pressure_test.go:158,203 starts io.Copy on the emulator and closes it before joining the reader; Emulator.Read and Close access closed without synchronization. The focused race control failed at emulator.go:288,301. This shared pair is the only instance in the window and affects every trial. Use concurrency-safe draining/shutdown and verify with the race detector.
- **BR-2** [Important] `experiment-deadline-enforcement` Blocking trial operations escape the promised deadlines
  terminal_pressure_test.go:152 uses cancellation without a deadline; input writes at 371,411,415 and synchronous observations at 333–396 can prevent timeout checks and cleanup from running. Recovery waits at 414,418,419,427 receive separate budgets rather than one five-second deadline. ARCH-PURPOSE: enforce an absolute deadline across these paths, release blocked operations on cancellation, join helpers and test stalled execution.
- **BR-3** [Important] `completion-evidence-before-success` Recovery is reported without proving all PTY output arrived
  terminal_pressure_test.go:419–447 treats side-channel producer completion plus queued-publication flush as output completion, although unread PTY bytes can remain. Final progress checks only child zero and accepts any sequence; emitted and ingested totals are only logged at 453. Require per-child terminal completion evidence and final presentation before reporting recovery, with a regression case delaying trailing PTY output.
- **BR-4** [Important] `readme-surface-discovery` README omits the new pressure experiment invocation
  atlas/couch.md documents PAIR_COUCH_PRESSURE=1 and TestCouchOutputPressure, but README.md is unchanged despite its existing terminal-testing section at 1076. Add the command and scope limitations there. This is the sole new operator-facing invocation family; PAIR_COUCH_PRESSURE_CHILD is an internal helper.

## Round 2 — 2026-10-01T16:06:11-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — terminal_pressure_test.go:230–251 closes the reply pipe and joins readers/writers before Emulator.Close. Focused race tests passed three repetitions; a scratch mutation restoring early Emulator.Close triggered race failures.
- BR-2 — addressed — Operations use pressureAwait, recovery shares window-end-plus-five-second cancellation, and cleanup closes resources before joining workers. The stalled-operation regression passed; removing pressureAwait's cancellation branch made it time out.
- BR-3 — not-addressed — terminal_pressure_test.go:489–498 checks marker absence and releases trailing output before invoking recovery checks. Removing lines 500–524 in a scratch Go overlay left TestCouchPressureTrailingPTYOutput passing three repetitions, including one reporting ingested_raw=2061 instead of the required 2087 bytes. The implementation adds completion checks, but the required fail-without-fix regression is missing. Exercise the actual recovery operation while trailing output remains withheld and assert it cannot report success.
- BR-4 — addressed — README.md:1090–1101 now documents the opt-in command and limitations. The invocation matches TestCouchOutputPressure's environment guard and 2×4×3 matrix; atlas/couch.md:2207–2218 documents the same surface.

## Open findings

- **BR-3** [Important] `completion-evidence-before-success` Recovery is reported without proving all PTY output arrived
