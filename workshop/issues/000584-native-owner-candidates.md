---
id: 000584
status: working
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: '811482386c5355c0c904353332c9bbefeb6ad2e3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T15:43:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "61cdc71d", done: "2f5bc31e"}
---

# Native owner scan blocks fleet resume on unrelated processes

## Problem

#556 native-owner admission blocks Resume from disk fleet-wide on macOS. It requires command evidence for every process before filtering Pair wrappers. Real live root services return EINVAL for kern.procargs2 and are omitted from the command snapshot; zombies and process-table churn also leave metadata without command evidence. Two read-only probes reproduced the exact refusal with 205–207 missing entries. ariadne:5 is parked with a proven native binding but cannot resume.

## Spec

Filter native-owner candidates from kernel metadata before requiring argv; unrelated processes and kernel-proven zombies do not block. Missing metadata or unreadable argv for a possible Pair owner remains uncertainty, never proof of absence. Fence candidate generation/parent changes across observations, with one bounded re-observation and refusal if still unstable. Preserve orphan/native-ID collision checks and exact-ID admission.

## Done when

- An unreadable unrelated root service and kernel-proven zombies cannot prevent proven Resume from disk.
- A possible Pair owner's missing argv or identity still refuses with actionable process evidence.
- Candidate process churn gets one bounded retry; stable re-observation succeeds and persistent churn refuses.
- Fake process-table regressions prove these behaviors and existing native-owner collision protections; required recorded suite and build pass, close review is SHIP.

## Plan

- Add failing process-table regressions at the ownership observation boundary.
- Filter candidates before argv certainty and retry unstable candidate observations once.
- Run targeted tests, host read-only observation, recorded suite and build; close and land on SHIP.

## Log

- 2026-10-15: TL authorized this fleet-blocking regression repair from main. Existing source at e1311121 and 6d65c4c4 is identical on the failing path. PID 32793 endpointsecurityd is live (exact identity 1792103969.915933), native argv EINVAL with successful identity; five other macOS services match. Eighteen kernel-confirmed zombies also exist. No production effects during diagnosis. ARCH-PURPOSE: certainty belongs to possible native owners, not every OS process; ARCH-ORDER: unknown owner evidence is not absence.
- 2026-10-15: Added regressions for projected unreadable root services, zombie wrapper and retained child, unreadable/identity-less possible Pair owners, settling and persistent candidate churn, unrelated churn, a newly visible colliding writer, and changed wrapper ancestry. All new cases fail against the original implementation (also checked using a Go source overlay); the native-owner suite passes after the fix. One changed-generation fixture now asserts the bounded retry before its later missing-child refusal. Retained-readiness changes retain their existing strict refusal.
- 2026-10-15: Read-only full host preflight advances past the original command-table refusal in 92–120 ms, then refuses `retained native child generation unknown`: legacy readiness repos/11873b98e33bf004/agent-ready-1-pair-7-codex.json has ordinal 71, PID 73724 and no agent_identity, while its node Codex process is still a child of live wrapper 73688. This separate #556 legacy-association blocker was reported to the hub (bc9cbbce-038e-4e6f-a942-43b444c6fdcc); no uncertainty bypass or production mutation was attempted.

## Revisions

### 2026-10-15 15:55 — approved narrow legacy association

TL approved extending #584 (32f38ca7, refined 61032867) after the host measurement found 18/21 legacy wrappers and migration's circular preflight. Preserve exact modern identities. For legacy readiness, prove current direct child/wrapper kinship from the kernel table and exact owner argv; use its exact readiness ordinal's ledger ID to exclude a different requested ID. For a matching ID, a source exemption additionally needs that owner ID among the native engine's held writable transcript paths, with generations/parent edges rechecked. Same-agent legacy uncertainty with no exclusion remains a refusal naming its owner and relaunch migration advice. Other agent kinds do not block. This is held-file/kinship proof, never cwd/tail inference. Codex needs its node shim's direct native engine child; arbitrary tool descendants are excluded. Durable design: workshop/plans/000584-native-owner-candidates-plan.md.

## Additional done when

- Legacy records with a different exact owner-ledger ID do not block a resume; matching/unknown identities remain guarded.
- Exact legacy source kinship plus held writable owner transcript permits the existing source exemption; wrong parent, PID reuse, read-only or unrelated held files, changed generations and unavailable proof refuse.
- Legacy refusal names its exact owner (and slot when Couch can resolve it), with migration guidance; modern readiness still needs full identity.
