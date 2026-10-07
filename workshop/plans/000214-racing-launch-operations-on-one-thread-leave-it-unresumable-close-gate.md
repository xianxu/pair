---
gate: boundary-review
issue: 214
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-07T15:38:53-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: Evidence pass drops the resolution on a typed refusal, so no-turn and unconfirmed are never produced
          detail: 'Real and fake resolvers return (resolution, ResumeRefusal) together; actionableinventory.go:1017-1023 records only Ambiguous on that branch and the default-branch provenBindingRefusal is unreachable for refusals. Scratch test at head: FreshRequired refusal projects session-gone; calling provenBindingRefusal(ResumeDiagnosticOf(resolveErr), binding) there yields no-turn.'
          family: refusal-drops-resolution
          round: 1
        - id: BR-2
          severity: Important
          title: Named reasons are tested only via hand-built ThreadEvidence, never through resolver to evidence to classify
          detail: Add inventory-level tests using a resolver returning resolution plus refusal (the real contract) for no-turn and unconfirmed; the existing shapes stayed green over the dead branch.
          family: test-bypasses-production-seam
          round: 1
        - id: BR-3
          severity: Important
          title: unconfirmed label says retry after a turn, but it is proven only by an incomplete storage listing
          detail: provenBindingRefusal names provisional only when ObservationIncomplete is set; the resolver message says retry when storage can be listed, and slot actions offer reboot only. Reword the Label and menu notice, or change the proof condition.
          family: label-matches-proof
          round: 1
        - id: BR-4
          severity: Minor
          title: Binding-code wording now lives in three tables (Label, unusableThreadNotice, bindingRefusalDiagnostic)
          family: wording-single-source
          round: 1
        - id: BR-5
          severity: Minor
          title: Query-level fallback test uses a simplified ledger; the 2026-09-08 26/29/31 shape exists only in the pure table
          family: done-when-test-shape
          round: 1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#214 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T15:38:53-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `refusal-drops-resolution` Evidence pass drops the resolution on a typed refusal, so no-turn and unconfirmed are never produced
  Real and fake resolvers return (resolution, ResumeRefusal) together; actionableinventory.go:1017-1023 records only Ambiguous on that branch and the default-branch provenBindingRefusal is unreachable for refusals. Scratch test at head: FreshRequired refusal projects session-gone; calling provenBindingRefusal(ResumeDiagnosticOf(resolveErr), binding) there yields no-turn.
- **BR-2** [Important] `test-bypasses-production-seam` Named reasons are tested only via hand-built ThreadEvidence, never through resolver to evidence to classify
  Add inventory-level tests using a resolver returning resolution plus refusal (the real contract) for no-turn and unconfirmed; the existing shapes stayed green over the dead branch.
- **BR-3** [Important] `label-matches-proof` unconfirmed label says retry after a turn, but it is proven only by an incomplete storage listing
  provenBindingRefusal names provisional only when ObservationIncomplete is set; the resolver message says retry when storage can be listed, and slot actions offer reboot only. Reword the Label and menu notice, or change the proof condition.
- **BR-4** [Minor] `wording-single-source` Binding-code wording now lives in three tables (Label, unusableThreadNotice, bindingRefusalDiagnostic)
- **BR-5** [Minor] `done-when-test-shape` Query-level fallback test uses a simplified ledger; the 2026-09-08 26/29/31 shape exists only in the pure table

## Open findings

- **BR-1** [Critical] `refusal-drops-resolution` Evidence pass drops the resolution on a typed refusal, so no-turn and unconfirmed are never produced
- **BR-2** [Important] `test-bypasses-production-seam` Named reasons are tested only via hand-built ThreadEvidence, never through resolver to evidence to classify
- **BR-3** [Important] `label-matches-proof` unconfirmed label says retry after a turn, but it is proven only by an incomplete storage listing
- **BR-4** [Minor] `wording-single-source` Binding-code wording now lives in three tables (Label, unusableThreadNotice, bindingRefusalDiagnostic)
- **BR-5** [Minor] `done-when-test-shape` Query-level fallback test uses a simplified ledger; the 2026-09-08 26/29/31 shape exists only in the pure table
