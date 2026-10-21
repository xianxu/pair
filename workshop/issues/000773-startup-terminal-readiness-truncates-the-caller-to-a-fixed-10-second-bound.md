---
id: 000773
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: '4a1287d65567e54da697a3127599430b522db604' # card fields mirrored from issue-cards; edit via sdlc
---

# startup: terminal readiness truncates the caller to a fixed 10-second bound

## Problem

Production terminal readiness imposes a fixed10s deadline even when its caller has a longer remaining deadline. A cold native agent launch under load can be refused before the caller budget expires. Discovered by #744 final native matrix, independently from #772's attach InputEnded failure. This is a startup-budget design defect; do not classify it away as a flake or increase a numeric timeout.

## Spec

Make terminal readiness honor a coherent caller-owned startup lifetime while preserving cancellation, bounded observation, fail-closed admission and generation-qualified cleanup. Design the production default for callers without deadlines explicitly; do not substitute a larger arbitrary constant. Preserve distinct precreation, readiness and postbirth native-query observation contracts and error evidence.

## Done when

- The actual production handoff path no longer truncates a longer caller startup deadline to the fixed birthBound10s; an earlier caller cancellation/deadline still stops promptly.
- A deterministic regression exercises the real production budget-selection boundary with readiness after10s but before a longer caller deadline, plus earlier cancellation/expiry and absent-readiness controls. No retry or blanket timeout bump.
- Cold native launch verification and required recorded checks pass, with original failure evidence retained and no claim that load/AMFI was the measured cause unless proved.

## Plan

- [ ]

## Log

2026-10-21: filed unclaimed by pair:5 per TL sequencing for #744. Trace: launcher/birthwatch.go birthBound10*time.Second; session_handoff_os.go passes it directly to coordinateTerminalHandoff; session_handoff.go applies context.WithTimeout(original caller,bound). Fleet caller allowed120s; separate LaunchOptions.BirthBound controls outer birth watch only; Couch registration default2m. CLI15s message is progress, not the budget.

Retained native failure on exact code db791e3b82bb23242ecd5e8fdbebd550c9da8e04: Claude fixture:3 (allocated thread tag1-fixture-4), nonce operation-9923c3b354cb3f27bbbb015f499d4a66; stage registration, Pair before/after exit1. Last output says terminal admission unverified: expected agent-ready-1-fixture-4-claude.json missing, then context deadline exceeded. Operation prepare877ms/operation15.732s; TestDisposableCouchFleet FAIL52.90s; whole command56.322s. Exact cleanup passed; no same-head rerun has occurred. Evidence in #744 PR509 at workshop/evidence/000744/final-outcome-native-fail.log, final-outcome-run.json, final-outcome-result.json; logSHA256355723305828834b91e4c8b91a2374d32d87e3f3c180bc00174f8c9f34058b42. Evidence commit51a07ef3936625a00011008e8d043db10f412efe.

Collision audit found shared scope/tag/agent resolver, unique tag-specific final/tmp paths and exact session/nonce validation; slot address3 versus tag ordinal4 is not a collision. Failed effective wrapper environment and sibling-file census were not retained, so incident inheritance remains unproved. Harness copies executables once per fleet; all workers reuse the same paths. Read-only macOS logs show contemporaneous Gatekeeper assessments with redacted paths; no event attributes this delay to the failed executable. AMFI, machine load, early wrapper failure and missing publication remain possible causes of this particular absent readiness; none is measured as its root cause. #744 is extending pre-cleanup evidence, without changing the production deadline.
