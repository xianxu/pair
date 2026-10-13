---
gate: boundary-review
issue: 494
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T23:27:05-07:00"
      dispatched: "2026-10-12T23:23:43-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Reconciliation reports another draft's save as successful
          detail: 'cmd/internal/webcast/operator.js:31–42: after draft A times out, saving a different draft B queries A and returns 204 when A was accepted, closing B without submitting it. Reproduced against production transport. Bind completion to its originating draft/action and test cancellation followed by a new draft (ARCH-ORDER).'
          family: operation-result-identity
          round: 1
        - id: BR-2
          severity: Critical
          title: Escape cannot turn annotation off while focus remains on the page
          detail: cmd/internal/webcast/web/annotate.js:591–593 registers Escape only on the operator ShadowRoot. Page-focused events do not reach it, including after Couch's focus-preserving mode command. Handle Escape at the document/window boundary and test this input path (ARCH-PURPOSE).
          family: keyboard-event-boundary
          round: 1
        - id: BR-3
          severity: Critical
          title: The plan labels a live-DOM helper PURE
          detail: workshop/plans/000494-operator-overlay-plan.md:23 classifies the recorded node path helper as PURE, but recorder/observer.js:53–69 reads live DOM structure and global document identity. Promote it to INTEGRATION and record the correction under Revisions, or extract a deterministic core over supplied data (ARCH-PURE).
          family: core-concept-purity-classification
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#494 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T23:27:05-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `operation-result-identity` Reconciliation reports another draft's save as successful
  cmd/internal/webcast/operator.js:31–42: after draft A times out, saving a different draft B queries A and returns 204 when A was accepted, closing B without submitting it. Reproduced against production transport. Bind completion to its originating draft/action and test cancellation followed by a new draft (ARCH-ORDER).
- **BR-2** [Critical] `keyboard-event-boundary` Escape cannot turn annotation off while focus remains on the page
  cmd/internal/webcast/web/annotate.js:591–593 registers Escape only on the operator ShadowRoot. Page-focused events do not reach it, including after Couch's focus-preserving mode command. Handle Escape at the document/window boundary and test this input path (ARCH-PURPOSE).
- **BR-3** [Critical] `core-concept-purity-classification` The plan labels a live-DOM helper PURE
  workshop/plans/000494-operator-overlay-plan.md:23 classifies the recorded node path helper as PURE, but recorder/observer.js:53–69 reads live DOM structure and global document identity. Promote it to INTEGRATION and record the correction under Revisions, or extract a deterministic core over supplied data (ARCH-PURE).

## Open findings

- **BR-1** [Critical] `operation-result-identity` Reconciliation reports another draft's save as successful
- **BR-2** [Critical] `keyboard-event-boundary` Escape cannot turn annotation off while focus remains on the page
- **BR-3** [Critical] `core-concept-purity-classification` The plan labels a live-DOM helper PURE
