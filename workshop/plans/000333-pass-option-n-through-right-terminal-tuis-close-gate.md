---
gate: boundary-review
issue: 333
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-26T22:16:12-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Hosted restart help contradicts the retained restart guard, and the help suite fails
          detail: cmd/internal/workbenchshortcut/shortcut.go:188 advertises draft reload for Couch-owned sessions reattached outside Couch, although CouchOwnsRestart still refuses it. Restore that refusal guidance alongside the right-terminal exception and update the standalone wording expectation in cmd/internal/keyscmd/keyscmd_test.go:159; TestPresenterAndHostingAreIndependent reproducibly fails in both modes. ARCH-PURPOSE.
          family: key-help-matches-runtime-policy
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-26T22:18:37-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Inspected the correction against the prior reviewed commit: cmd/internal/workbenchshortcut/shortcut.go:188 now says “does not reload a Couch thread; right terminal receives Alt+n”, matching CouchOwnsRestart in cmd/internal/launcher/outerrecord.go:72. The standalone expectation at cmd/internal/keyscmd/keyscmd_test.go:163 matches the new draft-scoped wording. Both the hosting/presenter regression and retained restart-guard regression pass uncached. This correction changes explanatory help text, not restart behavior.'
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#333 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-26T22:16:12-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `key-help-matches-runtime-policy` Hosted restart help contradicts the retained restart guard, and the help suite fails
  cmd/internal/workbenchshortcut/shortcut.go:188 advertises draft reload for Couch-owned sessions reattached outside Couch, although CouchOwnsRestart still refuses it. Restore that refusal guidance alongside the right-terminal exception and update the standalone wording expectation in cmd/internal/keyscmd/keyscmd_test.go:159; TestPresenterAndHostingAreIndependent reproducibly fails in both modes. ARCH-PURPOSE.

## Round 2 — 2026-09-26T22:18:37-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Inspected the correction against the prior reviewed commit: cmd/internal/workbenchshortcut/shortcut.go:188 now says “does not reload a Couch thread; right terminal receives Alt+n”, matching CouchOwnsRestart in cmd/internal/launcher/outerrecord.go:72. The standalone expectation at cmd/internal/keyscmd/keyscmd_test.go:163 matches the new draft-scoped wording. Both the hosting/presenter regression and retained restart-guard regression pass uncached. This correction changes explanatory help text, not restart behavior.

## Open findings

(none — every finding has been disposed)
