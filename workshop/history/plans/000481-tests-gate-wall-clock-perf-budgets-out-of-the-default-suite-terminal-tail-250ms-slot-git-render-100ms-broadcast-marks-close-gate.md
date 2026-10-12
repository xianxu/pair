---
gate: boundary-review
issue: 481
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T08:36:43-07:00"
      dispatched: "2026-10-12T08:34:06-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: atlas/couch.md:2579 enumerates only the couchcore budgets for make test-perf-budgets
          detail: The tail budget (wrapcmd TestTailOverEndpointSocketIsFast) and the shared cmd/internal/perfbudget switch (pair#481) are missing. Point the atlas at perfbudget.Check and the Makefile target instead of naming tests. It is the only stale enumeration in this window.
          family: docs-enumeration-drifts-from-source
          round: 1
        - id: BR-2
          severity: Minor
          title: perfbudget.Enforced is exported but has no caller outside the package
          detail: Keep it as deliberate API or unexport it to keep Check the single entry point.
          family: unused-exported-surface
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T08:40:44-07:00"
      dispatched: "2026-10-12T08:37:52-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'atlas/couch.md:2579-2585 now cites perfbudget.Check, cmd/internal/perfbudget and make test-perf-budgets as the list, and includes the wrapcmd tail. Remaining nit: the bracketed list at :2584 repeats the Makefile recipe; dropping it removes the drift risk. Rule for the family: docs point at the single source and do not restate it.'
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: 'perfbudget.go:20 Enforced is still exported with no caller outside the package. Env at :17 is the same, used only by perfbudget_test.go, which is in-package. Rule: export only what has an out-of-package caller. Unexport both (enforced, env) or record that the API is deliberate.'
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: c3496125f19c3f5e9b7d4a67c5a2f0d96c89cdaf
      blocked: false
---

# Gate ledger — pair#481 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T08:36:43-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-enumeration-drifts-from-source` atlas/couch.md:2579 enumerates only the couchcore budgets for make test-perf-budgets
  The tail budget (wrapcmd TestTailOverEndpointSocketIsFast) and the shared cmd/internal/perfbudget switch (pair#481) are missing. Point the atlas at perfbudget.Check and the Makefile target instead of naming tests. It is the only stale enumeration in this window.
- **BR-2** [Minor] `unused-exported-surface` perfbudget.Enforced is exported but has no caller outside the package
  Keep it as deliberate API or unexport it to keep Check the single entry point.

## Round 2 — 2026-10-12T08:40:44-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — atlas/couch.md:2579-2585 now cites perfbudget.Check, cmd/internal/perfbudget and make test-perf-budgets as the list, and includes the wrapcmd tail. Remaining nit: the bracketed list at :2584 repeats the Makefile recipe; dropping it removes the drift risk. Rule for the family: docs point at the single source and do not restate it.
- BR-2 — not-addressed — perfbudget.go:20 Enforced is still exported with no caller outside the package. Env at :17 is the same, used only by perfbudget_test.go, which is in-package. Rule: export only what has an out-of-package caller. Unexport both (enforced, env) or record that the API is deliberate.

## Open findings

- **BR-2** [Minor] `unused-exported-surface` perfbudget.Enforced is exported but has no caller outside the package
