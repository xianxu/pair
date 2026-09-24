---
gate: boundary-review
issue: 313
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-23T19:58:28-07:00"
      agent: codex
      recipe: milestone-review
      blocked: false
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-09-23T20:48:45-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: 'The #313 boundary includes undeclared #315 launch-protocol changes'
          detail: 'The range adds launch_nonce profile schema and fresh claim-registration behavior in cmd/internal/launcher, contradicting #313''s explicit “no new operation schema” constraint. Remove the unrelated commits or formally split and review the expanded scope.'
          family: boundary-scope-strands-issue
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-23T20:54:52-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'The issue now explicitly includes #315, with a separate pinned SHIP review and acceptance contract covering reserved claims and launch nonces.'
          round: 3
      findings:
        - id: BR-2
          severity: Critical
          title: The pinned boundary still includes undeclared bootstrap and CI gateway changes
          detail: '`bootstrap.sh`, `.github/workflows/merge-check.yml`, `Makefile`, and `.gitignore` changed between the pinned base and head via `aff72f82`, while the issue explicitly says that gateway commit is outside this delivery. Remove/rebase those changes from this boundary, or formally split and review the expanded gateway scope.'
          family: boundary-scope-strands-issue
          round: 3
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-09-23T21:29:04-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'The #313 Spec explicitly incorporates #315, and the pinned code plus separate #315 issue/plan contain the revised ordinary-registration delivery.'
          round: 4
        - id: BR-2
          disposition: addressed
          note: The required bootstrap, CI gateway, Makefile, and gitignore paths have no diff in the pinned base-to-head range.
          round: 4
      findings:
        - id: BR-3
          severity: Important
          title: 'Combined delivery retains unresolved and superseded #315 review evidence'
          detail: 'workshop/plans/000315-fresh-slot-registration-close-review.md:27 still claims RegisterFreshCouchThread and nonce transport, while the active #315 contract removes them; its later re-review records REWORK with no subsequent clean disposition in this range. Reconcile the artifact and obtain clean review evidence before closing #313.'
          family: stale-boundary-review-artifact
          round: 4
      recipe: milestone-review
      blocked: false
    - "n": 5
      timestamp: "2026-09-23T21:39:56-07:00"
      agent: codex
      dispose:
        - id: BR-3
          disposition: addressed
          note: 'The #315 close-review artifact now labels the nonce/registration review as historical and superseded, and includes a later clean SHIP re-review for the ordinary-registration implementation.'
          round: 5
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#313 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T19:58:28-07:00 (codex) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-09-23T20:48:45-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `boundary-scope-strands-issue` The #313 boundary includes undeclared #315 launch-protocol changes
  The range adds launch_nonce profile schema and fresh claim-registration behavior in cmd/internal/launcher, contradicting #313's explicit “no new operation schema” constraint. Remove the unrelated commits or formally split and review the expanded scope.

## Round 3 — 2026-09-23T20:54:52-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The issue now explicitly includes #315, with a separate pinned SHIP review and acceptance contract covering reserved claims and launch nonces.

### Raised

- **BR-2** [Critical] `boundary-scope-strands-issue` The pinned boundary still includes undeclared bootstrap and CI gateway changes
  `bootstrap.sh`, `.github/workflows/merge-check.yml`, `Makefile`, and `.gitignore` changed between the pinned base and head via `aff72f82`, while the issue explicitly says that gateway commit is outside this delivery. Remove/rebase those changes from this boundary, or formally split and review the expanded gateway scope.

## Round 4 — 2026-09-23T21:29:04-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The #313 Spec explicitly incorporates #315, and the pinned code plus separate #315 issue/plan contain the revised ordinary-registration delivery.
- BR-2 — addressed — The required bootstrap, CI gateway, Makefile, and gitignore paths have no diff in the pinned base-to-head range.

### Raised

- **BR-3** [Important] `stale-boundary-review-artifact` Combined delivery retains unresolved and superseded #315 review evidence
  workshop/plans/000315-fresh-slot-registration-close-review.md:27 still claims RegisterFreshCouchThread and nonce transport, while the active #315 contract removes them; its later re-review records REWORK with no subsequent clean disposition in this range. Reconcile the artifact and obtain clean review evidence before closing #313.

## Round 5 — 2026-09-23T21:39:56-07:00 (codex) — passed

### Disposed

- BR-3 — addressed — The #315 close-review artifact now labels the nonce/registration review as historical and superseded, and includes a later clean SHIP re-review for the ordinary-registration implementation.

## Open findings

(none — every finding has been disposed)
