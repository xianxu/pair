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
    - "n": 2
      timestamp: "2026-10-12T23:36:38-07:00"
      dispatched: "2026-10-12T23:31:47-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: operator.js:34–42 binds reconciliation to the originating action/body. Reverting this fix in a disposable checkout failed all three transport regressions; restored HEAD passed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: annotate.js:594 registers operator Escape on window capture. Reverting this fix failed the page-focused Escape regression; restored HEAD passed.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Plan line 48 now classifies the recorded node path helper as INTEGRATION, with the correction recorded at lines 417–419. observer.js:53–69 confirms its live-DOM and document dependencies.
          round: 2
      findings:
        - id: BR-4
          severity: Critical
          title: A lost save reply blocks future saves after switching away and back
          detail: 'operator.js:31–43 retains unsettled across projection/document changes at lines 100–104, while recorder/operator.go:156–161 replaces OperatorSession when the recording document changes. Reproduced an accepted save with its reply lost, followed by A→B→A: the recorder returns “result is no longer retained,” and every new save sends status for the old request forever. ARCH-ORDER, ARCH-PURPOSE. This is the 2nd finding in family operation-result-identity. Do NOT fix only this instance: define operation ownership across session/context/document/action, enumerate the lifecycle boundaries, and retire old-scope transport state without resubmitting old writes or accepting late replies into a new scope.'
          family: operation-result-identity
          round: 2
        - id: BR-5
          severity: Important
          title: Ordinary record updates replace color buttons during interaction
          detail: annotate.js:587 calls sync for every record; transition at line 392 calls renderRail, whose line 403 invokes renderSwatches. collab.js:127 replaces every color button. Confirmed on both viewer and operator mounts that an unchanged-model scroll record detaches the pressed button; updates between pointer-down and pointer-up can therefore cancel color selection. Preserve button identity and update attributes in place, or avoid rebuilding on unchanged state. Add an interleaved-record click regression. ARCH-ORDER, ARCH-CONSTRAINTS.
          family: interactive-dom-identity
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T23:45:07-07:00"
      dispatched: "2026-10-12T23:42:00-07:00"
      agent: codex
      dispose:
        - id: BR-4
          disposition: addressed
          note: operator.js:24 retires document-owned waiters, timers and reconciliation state; line 56 rejects stale continuations. operator-scope.test.mjs:77 reproduces lost-reply switch/return and fails without the fix. Recorder tests independently verify ledger retirement and fresh saves.
          round: 3
        - id: BR-5
          disposition: addressed
          note: annotate.js:402 preserves swatches when color is unchanged. Both interleaved-record regressions in operator.test.mjs:229 pass at head and fail with detached-button assertions when the fix is removed.
          round: 3
        - id: BR-1
          disposition: addressed
          note: Prior disposition retained. Action/body matching and identity-specific settlement remain in operator.js:48–59; both different-draft reconciliation regressions pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Prior disposition retained. Operator Escape remains captured at the window boundary with unavailable-state guards; the page-focus event regression passes.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Prior disposition retained. The plan lists the recorded-node path helper under Integration points, matching observer.js's live DOM ancestry traversal.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: bc69501b32cec152e616d1d25e3475a1962fec87
      blocked: false
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

## Round 2 — 2026-10-12T23:36:38-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — operator.js:34–42 binds reconciliation to the originating action/body. Reverting this fix in a disposable checkout failed all three transport regressions; restored HEAD passed.
- BR-2 — addressed — annotate.js:594 registers operator Escape on window capture. Reverting this fix failed the page-focused Escape regression; restored HEAD passed.
- BR-3 — addressed — Plan line 48 now classifies the recorded node path helper as INTEGRATION, with the correction recorded at lines 417–419. observer.js:53–69 confirms its live-DOM and document dependencies.

### Raised

- **BR-4** [Critical] `operation-result-identity` A lost save reply blocks future saves after switching away and back
  operator.js:31–43 retains unsettled across projection/document changes at lines 100–104, while recorder/operator.go:156–161 replaces OperatorSession when the recording document changes. Reproduced an accepted save with its reply lost, followed by A→B→A: the recorder returns “result is no longer retained,” and every new save sends status for the old request forever. ARCH-ORDER, ARCH-PURPOSE. This is the 2nd finding in family operation-result-identity. Do NOT fix only this instance: define operation ownership across session/context/document/action, enumerate the lifecycle boundaries, and retire old-scope transport state without resubmitting old writes or accepting late replies into a new scope.
- **BR-5** [Important] `interactive-dom-identity` Ordinary record updates replace color buttons during interaction
  annotate.js:587 calls sync for every record; transition at line 392 calls renderRail, whose line 403 invokes renderSwatches. collab.js:127 replaces every color button. Confirmed on both viewer and operator mounts that an unchanged-model scroll record detaches the pressed button; updates between pointer-down and pointer-up can therefore cancel color selection. Preserve button identity and update attributes in place, or avoid rebuilding on unchanged state. Add an interleaved-record click regression. ARCH-ORDER, ARCH-CONSTRAINTS.

## Round 3 — 2026-10-12T23:45:07-07:00 (codex) — passed

### Disposed

- BR-4 — addressed — operator.js:24 retires document-owned waiters, timers and reconciliation state; line 56 rejects stale continuations. operator-scope.test.mjs:77 reproduces lost-reply switch/return and fails without the fix. Recorder tests independently verify ledger retirement and fresh saves.
- BR-5 — addressed — annotate.js:402 preserves swatches when color is unchanged. Both interleaved-record regressions in operator.test.mjs:229 pass at head and fail with detached-button assertions when the fix is removed.
- BR-1 — addressed — Prior disposition retained. Action/body matching and identity-specific settlement remain in operator.js:48–59; both different-draft reconciliation regressions pass.
- BR-2 — addressed — Prior disposition retained. Operator Escape remains captured at the window boundary with unavailable-state guards; the page-focus event regression passes.
- BR-3 — addressed — Prior disposition retained. The plan lists the recorded-node path helper under Integration points, matching observer.js's live DOM ancestry traversal.

## Open findings

(none — every finding has been disposed)
