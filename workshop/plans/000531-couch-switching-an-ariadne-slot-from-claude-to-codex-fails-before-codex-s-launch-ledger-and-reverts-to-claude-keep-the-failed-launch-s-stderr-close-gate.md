---
gate: boundary-review
issue: 531
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T13:45:57-07:00"
      dispatched: "2026-10-13T13:42:34-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Blocked-start setup errors discard evidence from an already-spawned helper
          detail: cmd/internal/couchcore/launch_existing.go:183-184 records all StartBlocked errors with a nil handle, although launchhelper.go:57-69 can fail after spawning and cleaning up the helper. Preserve its output, observed exit, and started status across this boundary; add a deterministic regression for post-spawn setup failure. ARCH-PURPOSE and ARCH-ORDER.
          family: partial-start-evidence-preservation
          round: 1
        - id: BR-2
          severity: Important
          title: Immediate failures omit attempt identity and persistence-error location
          detail: cmd/internal/couchcore/launchfailure_capture.go:21-43 and 94-98 omit the nonce and diagnostic path required by the Spec and failure protocol. Include bounded attempt correlation and wrap persistence errors with the store path while preserving errors.Is/As; test both successful and failed persistence.
          family: diagnostic-attempt-correlation
          round: 1
        - id: BR-3
          severity: Important
          title: Done-when criteria omit the authorized installation and recovery-message changes
          detail: The issue's Done when section at lines 37-43 remains diagnostic-only despite the scope additions at line 77. Add criteria for installed links following checkout rebuilds and immediate failures providing one sanitized explanation with exactly one recovery action.
          family: completion-criteria-scope-coverage
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T13:57:54-07:00"
      dispatched: "2026-10-13T13:53:01-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: launchhelper.go preserves post-spawn evidence through blockedStartError; launchfailure_capture.go consumes it. Independently disabling runner preservation and recorder consumption made their respective regression tests fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: launchfailure_capture.go:39 includes bounded attempt correlation and diagnostic location; :113 wraps persistence errors while preserving their chain. Removing correlation made both successful-storage and failed-storage regression cases fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The issue's Done when lines 46–47 now require one-action sanitized guidance and install links following rebuilds. These match launchfailure_capture.go:22–45 and Makefile.local:184–189; operator-line and temporary-prefix installer tests passed.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: c94678925d28caa379bcbdc52e3ef9c782ac0ea4
      blocked: false
---

# Gate ledger — pair#531 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T13:45:57-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `partial-start-evidence-preservation` Blocked-start setup errors discard evidence from an already-spawned helper
  cmd/internal/couchcore/launch_existing.go:183-184 records all StartBlocked errors with a nil handle, although launchhelper.go:57-69 can fail after spawning and cleaning up the helper. Preserve its output, observed exit, and started status across this boundary; add a deterministic regression for post-spawn setup failure. ARCH-PURPOSE and ARCH-ORDER.
- **BR-2** [Important] `diagnostic-attempt-correlation` Immediate failures omit attempt identity and persistence-error location
  cmd/internal/couchcore/launchfailure_capture.go:21-43 and 94-98 omit the nonce and diagnostic path required by the Spec and failure protocol. Include bounded attempt correlation and wrap persistence errors with the store path while preserving errors.Is/As; test both successful and failed persistence.
- **BR-3** [Important] `completion-criteria-scope-coverage` Done-when criteria omit the authorized installation and recovery-message changes
  The issue's Done when section at lines 37-43 remains diagnostic-only despite the scope additions at line 77. Add criteria for installed links following checkout rebuilds and immediate failures providing one sanitized explanation with exactly one recovery action.

## Round 2 — 2026-10-13T13:57:54-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — launchhelper.go preserves post-spawn evidence through blockedStartError; launchfailure_capture.go consumes it. Independently disabling runner preservation and recorder consumption made their respective regression tests fail.
- BR-2 — addressed — launchfailure_capture.go:39 includes bounded attempt correlation and diagnostic location; :113 wraps persistence errors while preserving their chain. Removing correlation made both successful-storage and failed-storage regression cases fail.
- BR-3 — addressed — The issue's Done when lines 46–47 now require one-action sanitized guidance and install links following rebuilds. These match launchfailure_capture.go:22–45 and Makefile.local:184–189; operator-line and temporary-prefix installer tests passed.

## Open findings

(none — every finding has been disposed)
