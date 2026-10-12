---
gate: boundary-review
issue: 458
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T01:05:32-07:00"
      dispatched: "2026-10-12T01:04:28-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: withAttr copies the attribute map on every attr record, even for elements that aren't inputs (reduce.go:306)
          detail: Put an n.N == "input" guard before building the copy; attr is the most frequent mutation kind when replaying.
          family: hot-path-allocation
          round: 1
        - id: BR-2
          severity: Minor
          title: query.UserAction leaves out the new Hover kind with no recorded decision
          detail: 'Leaving Ptr out is correct. Decide whether hover belongs in a note''s action context, and write the decision down when #461 consumes it.'
          family: consumer-kind-classification
          round: 1
        - id: BR-3
          severity: Minor
          title: TestValidateDirReportsStrayFrameFiles discards the errors from Put, PutFrame, Close and WriteFile
          family: test-ignores-setup-errors
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      reviewed: a005266576968cee74282ce83f5d768f166bd78d
      blocked: false
---

# Gate ledger — pair#458 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T01:05:32-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `hot-path-allocation` withAttr copies the attribute map on every attr record, even for elements that aren't inputs (reduce.go:306)
  Put an n.N == "input" guard before building the copy; attr is the most frequent mutation kind when replaying.
- **BR-2** [Minor] `consumer-kind-classification` query.UserAction leaves out the new Hover kind with no recorded decision
  Leaving Ptr out is correct. Decide whether hover belongs in a note's action context, and write the decision down when #461 consumes it.
- **BR-3** [Minor] `test-ignores-setup-errors` TestValidateDirReportsStrayFrameFiles discards the errors from Put, PutFrame, Close and WriteFile

## Open findings

- **BR-1** [Minor] `hot-path-allocation` withAttr copies the attribute map on every attr record, even for elements that aren't inputs (reduce.go:306)
- **BR-2** [Minor] `consumer-kind-classification` query.UserAction leaves out the new Hover kind with no recorded decision
- **BR-3** [Minor] `test-ignores-setup-errors` TestValidateDirReportsStrayFrameFiles discards the errors from Put, PutFrame, Close and WriteFile
