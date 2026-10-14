---
gate: boundary-review
issue: 538
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T18:09:58-07:00"
      dispatched: "2026-10-13T18:05:52-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Operator placement never supplies original-node identity
          detail: cmd/internal/webcast/operator.js:88 omits createPlacement's original resolver, forcing unchanged connected anchors through replacement scoring. Production-function probes show common-text anchors and anchors on pages above the 2,000-node scan limit disappear only on the operator surface. Supply trusted document-scoped original-node identity and test both cases through the adapter (ARCH-PURPOSE).
          family: connected-anchor-authority
          round: 1
        - id: BR-2
          severity: Critical
          title: The core-concept table does not describe the pinned implementation
          detail: workshop/plans/000538-anchored-notes-draw-capture-plan.md:23-44 lists nonexistent AnchorEvidence/MatchDecision entities, places CaptureRecord in model.go instead of service.go, and marks unchanged board/wire.go and capture-engine files modified. Correct names, paths, pure/integration classifications and statuses in a Revisions entry; the required core-concept gate treats these contradictions as Critical (ARCH-PURE).
          family: core-concept-inventory-consistency
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#538 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T18:09:58-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `connected-anchor-authority` Operator placement never supplies original-node identity
  cmd/internal/webcast/operator.js:88 omits createPlacement's original resolver, forcing unchanged connected anchors through replacement scoring. Production-function probes show common-text anchors and anchors on pages above the 2,000-node scan limit disappear only on the operator surface. Supply trusted document-scoped original-node identity and test both cases through the adapter (ARCH-PURPOSE).
- **BR-2** [Critical] `core-concept-inventory-consistency` The core-concept table does not describe the pinned implementation
  workshop/plans/000538-anchored-notes-draw-capture-plan.md:23-44 lists nonexistent AnchorEvidence/MatchDecision entities, places CaptureRecord in model.go instead of service.go, and marks unchanged board/wire.go and capture-engine files modified. Correct names, paths, pure/integration classifications and statuses in a Revisions entry; the required core-concept gate treats these contradictions as Critical (ARCH-PURE).

## Open findings

- **BR-1** [Critical] `connected-anchor-authority` Operator placement never supplies original-node identity
- **BR-2** [Critical] `core-concept-inventory-consistency` The core-concept table does not describe the pinned implementation
