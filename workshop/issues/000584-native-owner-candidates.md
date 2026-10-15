---
id: 000584
status: working
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 1.3935
severity: critical
card_mirror: '7405628bc6bfb48c49df259a9ea7af407806f0ff' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T15:43:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

- Legacy records with a different exact owner-ledger ID do not block a resume; matching/unknown identities remain guarded.
- Exact legacy source kinship plus held writable owner transcript permits the existing source exemption; wrong parent, PID reuse, read-only or unrelated held files, changed generations and unavailable proof refuse.
- Legacy refusal names its exact owner (and slot when Couch can resolve it), with migration guidance; modern readiness still needs full identity.

## Plan

- [x] Add failing process-table regressions at the ownership observation boundary.
- [x] Filter candidates before argv certainty and retry unstable candidate observations once.
- [ ] Run targeted tests, host read-only observation, recorded suite and build; close and land on SHIP.

## Log

- 2026-10-15: TL authorized this fleet-blocking regression repair from main. Existing source at e1311121 and 6d65c4c4 is identical on the failing path. PID 32793 endpointsecurityd is live (exact identity 1792103969.915933), native argv EINVAL with successful identity; five other macOS services match. Eighteen kernel-confirmed zombies also exist. No production effects during diagnosis. ARCH-PURPOSE: certainty belongs to possible native owners, not every OS process; ARCH-ORDER: unknown owner evidence is not absence.
- 2026-10-15: Added regressions for projected unreadable root services, zombie wrapper and retained child, unreadable/identity-less possible Pair owners, settling and persistent candidate churn, unrelated churn, a newly visible colliding writer, and changed wrapper ancestry. All new cases fail against the original implementation (also checked using a Go source overlay); the native-owner suite passes after the fix. One changed-generation fixture now asserts the bounded retry before its later missing-child refusal. Retained-readiness changes retain their existing strict refusal.
- 2026-10-15: Read-only full host preflight advances past the original command-table refusal in 92–120 ms, then refuses `retained native child generation unknown`: legacy readiness repos/11873b98e33bf004/agent-ready-1-pair-7-codex.json has ordinal 71, PID 73724 and no agent_identity, while its node Codex process is still a child of live wrapper 73688. This separate #556 legacy-association blocker was reported to the hub (bc9cbbce-038e-4e6f-a942-43b444c6fdcc); no uncertainty bypass or production mutation was attempted.

- 2026-10-15: Final bounded-scheduling host observation passed five consecutive read-only checks under concurrent targeted/race runs: 1.579, 1.664, 1.562, 1.496 and 4.281 seconds, all inside the unchanged five-second budget (load 12.67/16.96/20.08 at the end). Typical margin is over three seconds; the slowest measured margin is 0.719 seconds, not a guarantee under arbitrary load. No launch, reservation, park, signal or production metadata writes were performed. Earlier serial checks under concurrent build load exhausted the budget and are retained above rather than omitted.
- 2026-10-15: Targeted affected-package run passed: couchcore 9.030s, launcher 0.725s, procutil 3.345s, sessioninventory 0.186s. The real twelve-owner/two-fence memo test and ordered worker suite pass under -race (five repetitions, 1.681s); worker-only race tests pass twenty repetitions. Sessionledger and strictjson package tests passed. Real-row differential corpus and adversarial differential fuzz pass; native ledger content is retained only locally.

## Revisions

### 2026-10-15 15:55 — approved narrow legacy association

TL approved extending #584 (32f38ca7, refined 61032867) after the host measurement found 18/21 legacy wrappers and migration's circular preflight. Preserve exact modern identities. For legacy readiness, prove current direct child/wrapper kinship from the kernel table and exact owner argv; use its exact readiness ordinal's ledger ID to exclude a different requested ID. For a matching ID, a source exemption additionally needs that owner ID among the native engine's held writable transcript paths, with generations/parent edges rechecked. Same-agent legacy uncertainty with no exclusion remains a refusal naming its owner and relaunch migration advice. Other agent kinds do not block. This is held-file/kinship proof, never cwd/tail inference. Codex needs its node shim's direct native engine child; arbitrary tool descendants are excluded. Durable design: workshop/plans/000584-native-owner-candidates-plan.md.


### 2026-10-15 — finite ledger snapshot reader

Read-only host validation reached a further #556 reader-only limit: pair:2's valid ledger is 21,466,623 bytes, exceeding the new guard's 8 MiB whole-file cap. #508 already removed this unsupported cap from sessioninventory. Native ownership now reads the finite observed ledger snapshot through the existing regular-file, no-symlink, cancellation and unchanged-generation reader, preserving readiness's small fixed bound; a >8 MiB valid-ledger regression went red on the old reader and green after the change. No writer policy or ledger schema changed.

### 2026-10-15 — unbound legacy owner, decision requested

The next host check advances in 783 ms to a genuine unbound legacy source: pair:2's readiness ordinal 65 has no requested_native_id and no binding record. Its exact live chain is wrapper 4374 -> node 4461 -> Codex 4470. Writable held rollouts contain one primary source=cli ID (01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd); other held metadata names that ID as its subagent parent. Sent the hub the remaining blocker and a proposed unique-primary held-header fallback; no implementation of that extra rule yet. Current guards preserve this uncertainty. This checkpoint's remaining host failure is expected and reported.


### 2026-10-15 — approved unbound legacy ID set

TL approved receipt 5d2dcac0 to use a complete positively held writable native transcript ID set only when the exact legacy launch is genuinely unbound, never conflicting or malformed. The same exact wrapper/child/engine births and parent edges, plus the sorted ID set, are compared at both observations. Matching held IDs remain collisions except the exact source exemption; missing/partial/unrecognized evidence remains unknown. New unbound/different-ID and source cases went red before implementation, and the native suite passes afterwards. Real lsof conformance also found a valid unnamed Darwin NPOLICY descriptor; its real-shaped regression went red and the parser now accepts that non-file shape without accepting unnamed REG files.


### 2026-10-15 — approved modern unbound ID set and host timing

TL approved fcda041b: the absent-only complete held-ID fallback also applies after a modern owner passes its full birth and ancestry checks. Malformed/conflicting IDs and identity mismatches still refuse. Both modern and legacy regression matrices pass. A per-admission SHA-256 projection memo retains fresh physical reads at both observations; a changed-byte mutation test proves that stale projections cannot admit changed bindings. The real five-second host check still times out: a separate read-only 15-second diagnostic reached the budget while decoding whole historical ledgers (20–33 MB each); physical reads took milliseconds. No slot effects occurred. Investigating codec cost before claiming the fleet blocker resolved; no deadline increase or weakened validation has been applied.

### 2026-10-15 — shared codec performance, approved

TL approved bc20aff9 the shared json.Valid/key-only duplicate scanner with differential proof against the old scanner, including real ledger rows and adversarial nesting, escaping, Unicode and trailing data. Private ledger field decoders can avoid redundant scans after the outer strict decoder, retaining all rejection semantics. The selective-generation shortcut and larger deadline are unnecessary and are not being implemented. Required host read-only evidence must show margin, beyond a single near-deadline pass.

### 2026-10-15 — approved bounded owner scheduling

TL approved dcac37b0 after serial observation remained marginal during a concurrent build. At most four independent owner proofs per phase; consume in stable wrapper order, cancel and join on failure, keep the memo lock off IO/decode, and retain both fences and the five-second budget. Differential scanner proof: 1,032 rows from the 21 largest retained ledgers, 326,709,546 non-newline bytes, all accepted by both scanners, SHA-256 df55dff5f68e4675c94dfed6ab77f584073e7ea7d842c5caf0b2988b591899fa; adversarial differential fuzz passed 812,261 cases. The corpus test is a local read-only overlay; native content is not committed.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md, Method A only (calibration marked stale by sdlc). Two smaller Go extensions cover candidate filtering/retry and legacy association; one bounded held-file IO module reuses os/exec and existing filename codecs. Thorough approved design discounts design by 0.2, standard-library availability halves the held-file module's design, implementation uses v3.1's 0.4 scale. Familiarity is 1.0. One review and one real-API discovery budget cover the boundary and host descriptor measurement; docs and issue authoring are separate primitives.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.15 impl=0.08
item: smaller-go-module design=0.02 impl=0.20
item: smaller-go-module design=0.02 impl=0.20
item: greenfield-go-module design=0.05 impl=0.20
item: atlas-docs design=0.05 impl=0.02
item: milestone-review design=0.0 impl=0.20
item: real-api-discovery design=0.0 impl=0.16
design-buffer: 0.15
total: 1.3935
```
