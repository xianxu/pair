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
    - "n": 2
      timestamp: "2026-10-13T18:22:37-07:00"
      dispatched: "2026-10-13T18:18:59-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: operator.js:93 supplies the original resolver through recorder/operator_original.go:13 and chrome_operator_original.go:66. Removing that resolver in a disposable copy made five adapter regressions fail, including common-text originals and pages exceeding the scan cap; the pinned implementation passes.
          round: 2
        - id: BR-2
          disposition: addressed
          note: workshop/plans/000538-anchored-notes-draw-capture-plan.md:209 explicitly supersedes the stale tables. The replacement inventory matches the inspected definitions, classifies captureRecord in service.go as integration state, and correctly identifies board/wire.go and the capture-engine files as reused unchanged.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 4ad125006c605beedf3bae51ca96bf65c09b7be4
      blocked: false
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

## Round 2 — 2026-10-13T18:22:37-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — operator.js:93 supplies the original resolver through recorder/operator_original.go:13 and chrome_operator_original.go:66. Removing that resolver in a disposable copy made five adapter regressions fail, including common-text originals and pages exceeding the scan cap; the pinned implementation passes.
- BR-2 — addressed — workshop/plans/000538-anchored-notes-draw-capture-plan.md:209 explicitly supersedes the stale tables. The replacement inventory matches the inspected definitions, classifies captureRecord in service.go as integration state, and correctly identifies board/wire.go and the capture-engine files as reused unchanged.

## Open findings

(none — every finding has been disposed)
