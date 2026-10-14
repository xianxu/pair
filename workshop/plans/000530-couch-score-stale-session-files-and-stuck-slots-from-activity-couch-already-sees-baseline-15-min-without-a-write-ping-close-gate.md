---
gate: boundary-review
issue: 530
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T21:26:01-07:00"
      dispatched: "2026-10-13T21:24:09-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Generated wait notices still create session-file obligations
          detail: 'cmd/internal/couchcmd/wait_notice.go:397 bypasses system-provenance registration, so message_service.go:1058 records delivered wait notices as owed rows. ARCH-PURPOSE: apply the revised exclusion contract to all generated notices and add ledger regression coverage.'
          family: generated-message-accounting
          round: 1
        - id: BR-2
          severity: Critical
          title: Startup prompt failures after admission produce no required notice
          detail: 'cmd/internal/couchcmd/supervision.go:168 checks only admission errors, while lines 78–80 discard failed or unconfirmed delivery outcomes and queued expiry has no coordinator observer. ARCH-ORDER: reconcile terminal broker outcomes into one readable failure notice without regenerating the prompt; test cancellation, expiry, and uncertainty.'
          family: asynchronous-outcome-reconciliation
          round: 1
        - id: BR-3
          severity: Critical
          title: Observation capacity exhaustion silently disables report and reply tracking
          detail: 'cmd/internal/couchcmd/supervision.go:34–46 retains failed attempts alongside successful sends for two hours; once 256 entries accumulate, lines 65–66 discard new reports and reply provenance, and generated nudges also fail admission to this table. ARCH-CONSTRAINTS: preserve authoritative report/action observations under bounded overflow and test saturation with subsequent successful traffic.'
          family: bounded-observation-overflow
          round: 1
        - id: BR-4
          severity: Important
          title: Primary Done-when criteria still require explicitly deferred work
          detail: workshop/issues/000530-couch-score-stale-session-files-and-stuck-slots-from-activity-couch-already-sees-baseline-15-min-without-a-write-ping.md:35–37 requires the file-ping baseline, feature vectors, and sampling despite the approved rules-only revision. Update the primary section to the revised acceptance criteria and retain links to the companion issues.
          family: acceptance-scope-consistency
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T22:00:52-07:00"
      dispatched: "2026-10-13T21:56:45-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Registration is fixed, but supervision.go:128 removes provenance before an in-flight endpoint callback necessarily finishes. message_service.go:1060 then records generated traffic as owed. Controlled disconnect/reconcile/late-return tests fail for both generated-send directions in three repetitions. Preserve provenance through delivery accounting across the entire generated-message-accounting family.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Terminal prompt outcomes and queued cancellation now produce one actionable notice without resending. Relevant regressions pass at pinned head and fail with pre-fix production files restored; broker tests also verify queue expiry.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Failed attempts no longer consume a separate capped observation history. The regression exceeding MaxReceipts and then sending a successful report/reply passes at head and fails against the pre-fix implementation.
          round: 2
        - id: BR-4
          disposition: addressed
          note: The pinned Done when diff now requires rules, startup prompts, exclusions, and outcome coverage, explicitly assigning file pings to parley.nvim#321 and logging to pair#543. This matches the approved rules-only revision and implemented scope.
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-13T22:22:20-07:00"
      dispatched: "2026-10-13T22:19:03-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: supervision.go:124 retains provenance while delivery accounting remains active. TestSupervisionReviewLateDeliveryKeepsGeneratedProvenance passes at head; removing that guard fails startup, rules-nudge, and ordinary-wait cases with unwanted ledger obligations. The genuine-message control remains recorded.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Prior disposition retained. Terminal-prompt and queued-cancellation regressions pass; reconciliation produces one actionable notice without regenerating the prompt.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Prior disposition retained. Failed attempts no longer occupy a capped history; the regression exceeding MaxReceipts still tracks a subsequent report and confirmed reply.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Prior disposition retained. The issue's Done when explicitly assigns file pings to parley.nvim#321 and logging to pair#543, consistent with the revised plan and implemented rules-only scope.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 2b0b19c37c73169826ee96eaefa64ace3f681c30
      blocked: false
---

# Gate ledger — pair#530 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T21:26:01-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `generated-message-accounting` Generated wait notices still create session-file obligations
  cmd/internal/couchcmd/wait_notice.go:397 bypasses system-provenance registration, so message_service.go:1058 records delivered wait notices as owed rows. ARCH-PURPOSE: apply the revised exclusion contract to all generated notices and add ledger regression coverage.
- **BR-2** [Critical] `asynchronous-outcome-reconciliation` Startup prompt failures after admission produce no required notice
  cmd/internal/couchcmd/supervision.go:168 checks only admission errors, while lines 78–80 discard failed or unconfirmed delivery outcomes and queued expiry has no coordinator observer. ARCH-ORDER: reconcile terminal broker outcomes into one readable failure notice without regenerating the prompt; test cancellation, expiry, and uncertainty.
- **BR-3** [Critical] `bounded-observation-overflow` Observation capacity exhaustion silently disables report and reply tracking
  cmd/internal/couchcmd/supervision.go:34–46 retains failed attempts alongside successful sends for two hours; once 256 entries accumulate, lines 65–66 discard new reports and reply provenance, and generated nudges also fail admission to this table. ARCH-CONSTRAINTS: preserve authoritative report/action observations under bounded overflow and test saturation with subsequent successful traffic.
- **BR-4** [Important] `acceptance-scope-consistency` Primary Done-when criteria still require explicitly deferred work
  workshop/issues/000530-couch-score-stale-session-files-and-stuck-slots-from-activity-couch-already-sees-baseline-15-min-without-a-write-ping.md:35–37 requires the file-ping baseline, feature vectors, and sampling despite the approved rules-only revision. Update the primary section to the revised acceptance criteria and retain links to the companion issues.

## Round 2 — 2026-10-13T22:00:52-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — Registration is fixed, but supervision.go:128 removes provenance before an in-flight endpoint callback necessarily finishes. message_service.go:1060 then records generated traffic as owed. Controlled disconnect/reconcile/late-return tests fail for both generated-send directions in three repetitions. Preserve provenance through delivery accounting across the entire generated-message-accounting family.
- BR-2 — addressed — Terminal prompt outcomes and queued cancellation now produce one actionable notice without resending. Relevant regressions pass at pinned head and fail with pre-fix production files restored; broker tests also verify queue expiry.
- BR-3 — addressed — Failed attempts no longer consume a separate capped observation history. The regression exceeding MaxReceipts and then sending a successful report/reply passes at head and fails against the pre-fix implementation.
- BR-4 — addressed — The pinned Done when diff now requires rules, startup prompts, exclusions, and outcome coverage, explicitly assigning file pings to parley.nvim#321 and logging to pair#543. This matches the approved rules-only revision and implemented scope.

## Round 3 — 2026-10-13T22:22:20-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — supervision.go:124 retains provenance while delivery accounting remains active. TestSupervisionReviewLateDeliveryKeepsGeneratedProvenance passes at head; removing that guard fails startup, rules-nudge, and ordinary-wait cases with unwanted ledger obligations. The genuine-message control remains recorded.
- BR-2 — addressed — Prior disposition retained. Terminal-prompt and queued-cancellation regressions pass; reconciliation produces one actionable notice without regenerating the prompt.
- BR-3 — addressed — Prior disposition retained. Failed attempts no longer occupy a capped history; the regression exceeding MaxReceipts still tracks a subsequent report and confirmed reply.
- BR-4 — addressed — Prior disposition retained. The issue's Done when explicitly assigns file pings to parley.nvim#321 and logging to pair#543, consistent with the revised plan and implemented rules-only scope.

## Open findings

(none — every finding has been disposed)
