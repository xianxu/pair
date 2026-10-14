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

## Open findings

- **BR-1** [Critical] `generated-message-accounting` Generated wait notices still create session-file obligations
- **BR-2** [Critical] `asynchronous-outcome-reconciliation` Startup prompt failures after admission produce no required notice
- **BR-3** [Critical] `bounded-observation-overflow` Observation capacity exhaustion silently disables report and reply tracking
- **BR-4** [Important] `acceptance-scope-consistency` Primary Done-when criteria still require explicitly deferred work
