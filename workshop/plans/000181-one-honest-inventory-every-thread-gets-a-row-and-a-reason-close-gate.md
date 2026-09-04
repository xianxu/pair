---
gate: boundary-review
issue: 181
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-03T18:35:37-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
          detail: |-
            menu.go:388-401 sets operation="switch" for any non-Resumable row and
            menuActionItems returns {"resume","name","describe"} for any non-Live
            row, so a park-in-flight row gets both. Task 4 specifies only the
            unusable case. Downstream refusals catch it, but the row lies about what
            it offers.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: new-state-unhandled-at-consumers
          round: 1
        - id: BR-2
          severity: Minor
          title: Task 2's migration scope understates the affected files
          detail: |-
            ProjectActionableThreads and ActionableThreadInventory* appear in 14
            files (29 direct calls), not "36 call sites across six files"; the plan
            names four, one of which (artifactpath/deadsymbols_test.go) is outside
            the couch packages.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: unbacked-existing-behavior-claim
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-03T18:35:37-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
---

# Gate ledger — pair#181 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-03T18:35:37-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `new-state-unhandled-at-consumers` ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
  menu.go:388-401 sets operation="switch" for any non-Resumable row and
  menuActionItems returns {"resume","name","describe"} for any non-Live
  row, so a park-in-flight row gets both. Task 4 specifies only the
  unusable case. Downstream refusals catch it, but the row lies about what
  it offers.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-2** [Minor] `unbacked-existing-behavior-claim` Task 2's migration scope understates the affected files
  ProjectActionableThreads and ActionableThreadInventory* appear in 14
  files (29 direct calls), not "36 call sites across six files"; the plan
  names four, one of which (artifactpath/deadsymbols_test.go) is outside
  the couch packages.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-03T18:35:37-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

- **BR-1** [Minor] `new-state-unhandled-at-consumers` ThreadBusy rows reach Enter and menuActionItems, where !Live() offers switch and resume
- **BR-2** [Minor] `unbacked-existing-behavior-claim` Task 2's migration scope understates the affected files
