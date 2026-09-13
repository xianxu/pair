---
gate: plan-quality
issue: 227
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-13T09:25:00-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: 'Escape-chord decision recorded on a wrong enumeration: M-k focus-left is also lost, with no global fallback'
          detail: Plan says only create/close need leaving the app; every role-scoped chord passes through, including M-k, the sole keyboard bridge back to the left stack (shortcut.go:192, no global equivalent in shortcut.go:145-167, zellij MoveFocus defaults unbound in zellij/config.kdl:93-119). Enumerate the full lost set, name M-k's surviving fallbacks, then record the decision.
          family: decision-premise-enumeration
          round: 1
        - id: PQ-2
          severity: Minor
          title: Task 3 Step 1 is a placeholder duplicating Task 4's table test
          detail: altScreenMux builds a real terminalMux with unverified fields and is unused; the loop body is `_ = chord`. Drop it and let Task 4 own the table.
          family: test-plan-compression
          round: 1
        - id: PQ-3
          severity: Minor
          title: Issue Done-when and Plan rows still describe the superseded ShortcutInput design
          detail: The Revisions entry in Task 5 should also rewrite the "arrives as an input field" bullet and the Plan rows, or the close review fails the issue against its own text.
          family: issue-text-tracks-design
          round: 1
        - id: PQ-4
          severity: Minor
          title: Task 5 Step 1 points at RoleBinding.Help for a note that belongs in keyhelp's groupTerminal heading
          detail: catalog.go:10 holds the heading; entries hold no wording (catalog.go:16). Name the one site.
          family: name-the-edit-site
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-13T09:29:24-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Full lost set enumerated, M-k excluded with rationale, fallbacks named, escape-chord decision recorded against the corrected premise.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Task 3 Step 1 is now the seam description; Task 4 owns the table.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 5 Step 3 rewrites the Done-when bullet and Plan rows in place; issue file catches up when that step runs.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Names the groupTerminal heading at catalog.go:10 as the one site.
          round: 2
      findings:
        - id: PQ-5
          severity: Minor
          title: Task 4 table oracle says "iff !IsGlobalChord", contradicting the M-k exclusion
          detail: M-k is non-global yet must not be written to the child. The oracle should be RightTerminalChordPassesThrough itself, so the test cannot diverge from the exclusion set.
          family: test-oracle-derives-from-classifier
          round: 2
      blocked: false
content_hash: c434151b19d11cb1e4dedf201c0d043d3de08f0d217555a51521c01e70525ea8
---

# Gate ledger — pair#227 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T09:25:00-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `decision-premise-enumeration` Escape-chord decision recorded on a wrong enumeration: M-k focus-left is also lost, with no global fallback
  Plan says only create/close need leaving the app; every role-scoped chord passes through, including M-k, the sole keyboard bridge back to the left stack (shortcut.go:192, no global equivalent in shortcut.go:145-167, zellij MoveFocus defaults unbound in zellij/config.kdl:93-119). Enumerate the full lost set, name M-k's surviving fallbacks, then record the decision.
- **PQ-2** [Minor] `test-plan-compression` Task 3 Step 1 is a placeholder duplicating Task 4's table test
  altScreenMux builds a real terminalMux with unverified fields and is unused; the loop body is `_ = chord`. Drop it and let Task 4 own the table.
- **PQ-3** [Minor] `issue-text-tracks-design` Issue Done-when and Plan rows still describe the superseded ShortcutInput design
  The Revisions entry in Task 5 should also rewrite the "arrives as an input field" bullet and the Plan rows, or the close review fails the issue against its own text.
- **PQ-4** [Minor] `name-the-edit-site` Task 5 Step 1 points at RoleBinding.Help for a note that belongs in keyhelp's groupTerminal heading
  catalog.go:10 holds the heading; entries hold no wording (catalog.go:16). Name the one site.

## Round 2 — 2026-09-13T09:29:24-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Full lost set enumerated, M-k excluded with rationale, fallbacks named, escape-chord decision recorded against the corrected premise.
- PQ-2 — addressed — Task 3 Step 1 is now the seam description; Task 4 owns the table.
- PQ-3 — addressed — Task 5 Step 3 rewrites the Done-when bullet and Plan rows in place; issue file catches up when that step runs.
- PQ-4 — addressed — Names the groupTerminal heading at catalog.go:10 as the one site.

### Raised

- **PQ-5** [Minor] `test-oracle-derives-from-classifier` Task 4 table oracle says "iff !IsGlobalChord", contradicting the M-k exclusion
  M-k is non-global yet must not be written to the child. The oracle should be RightTerminalChordPassesThrough itself, so the test cannot diverge from the exclusion set.

## Open findings

- **PQ-5** [Minor] `test-oracle-derives-from-classifier` Task 4 table oracle says "iff !IsGlobalChord", contradicting the M-k exclusion
