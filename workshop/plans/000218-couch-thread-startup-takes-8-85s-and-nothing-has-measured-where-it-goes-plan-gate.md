---
gate: plan-quality
issue: 218
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-14T15:08:14-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Targeted validation does not preserve fresh session-name exclusivity.
          detail: 'ARCH-ORDER/ARCH-SECURE: current Probe and Revalidate require exactly one server (cmd/internal/launcher/session_owner_os.go:86 and :110). Checking only the captured PID/generation/socket cannot detect a second claimant appearing after capture; specify fresh exclusivity evidence or explicitly justify the changed authorization contract, and verify it with controlled ordering.'
          family: fresh-authority-preserves-exclusivity
          round: 1
        - id: PQ-2
          severity: Important
          title: Define how registration discovers servers created after the pass snapshot.
          detail: 'ARCH-ORDER/ARCH-PURPOSE: M2 routes recoverySession through immutable initial evidence, but cold awaitResumeRegistration polls that function after pane birth (cmd/internal/couchcore/launch_existing.go:407 and :419). Specify a fresh discovery path or new observation scope for post-launch registration, including the missing-row branch, so legitimate startup can progress without stale absence authorizing an effect.'
          family: observation-scope-allows-lifecycle-progress
          round: 1
        - id: PQ-3
          severity: Important
          title: Compress test inventories into named functions and adversarial strategies.
          detail: M1 steps 4/6 and M2 steps 1/4 enumerate cases instead of the required compact test contract. Name the candidate selector, snapshotSelected, exact revalidator, SessionOwnerPass observation method, and affected lifecycle functions; give each risky function one strategy line naming its adversarial input or event class and mechanical guard.
          family: tests-name-functions-and-strategies
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-14T15:10:23-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Fresh narrowed enumeration preserves exactly-one-server authority; ordered competing-claimant tests verify it.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Registration bypasses pass evidence and discovers post-launch births; cold/create operations remain outside the pass.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: The superseding test contract names risky functions and gives adversarial strategies with mechanical guards.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-14T15:12:11-07:00"
      agent: codex
      blocked: false
      protocol_error: no valid findings block
    - "n": 4
      timestamp: "2026-10-14T15:12:55-07:00"
      agent: codex
      blocked: false
content_hash: 19af3f233d1dd25a19583bea3162dd2e7c46ea68a30648c2a19e487b1162a6a4
---

# Gate ledger — pair#218 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T15:08:14-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `fresh-authority-preserves-exclusivity` Targeted validation does not preserve fresh session-name exclusivity.
  ARCH-ORDER/ARCH-SECURE: current Probe and Revalidate require exactly one server (cmd/internal/launcher/session_owner_os.go:86 and :110). Checking only the captured PID/generation/socket cannot detect a second claimant appearing after capture; specify fresh exclusivity evidence or explicitly justify the changed authorization contract, and verify it with controlled ordering.
- **PQ-2** [Important] `observation-scope-allows-lifecycle-progress` Define how registration discovers servers created after the pass snapshot.
  ARCH-ORDER/ARCH-PURPOSE: M2 routes recoverySession through immutable initial evidence, but cold awaitResumeRegistration polls that function after pane birth (cmd/internal/couchcore/launch_existing.go:407 and :419). Specify a fresh discovery path or new observation scope for post-launch registration, including the missing-row branch, so legitimate startup can progress without stale absence authorizing an effect.
- **PQ-3** [Important] `tests-name-functions-and-strategies` Compress test inventories into named functions and adversarial strategies.
  M1 steps 4/6 and M2 steps 1/4 enumerate cases instead of the required compact test contract. Name the candidate selector, snapshotSelected, exact revalidator, SessionOwnerPass observation method, and affected lifecycle functions; give each risky function one strategy line naming its adversarial input or event class and mechanical guard.

## Round 2 — 2026-10-14T15:10:23-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Fresh narrowed enumeration preserves exactly-one-server authority; ordered competing-claimant tests verify it.
- PQ-2 — addressed — Registration bypasses pass evidence and discovers post-launch births; cold/create operations remain outside the pass.
- PQ-3 — addressed — The superseding test contract names risky functions and gives adversarial strategies with mechanical guards.

## Round 3 — 2026-10-14T15:12:11-07:00 (codex) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 4 — 2026-10-14T15:12:55-07:00 (codex) — passed

## Open findings

(none — every finding has been disposed)
