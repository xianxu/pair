---
id: 000553
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
severity: critical
card_mirror: '8971d14d0e7f1e973cfa021cc1175f040fcebe5d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T11:23:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: operator, spec: "a91bf09c", done: "83122de6"}
---

# procutil: skip exited processes during native command reads

## Problem

The #547 native process inspector keeps kernel command-read failures for PIDs that exit after the table snapshot. Couch reattach and leave then abort on a vanished possible Zellij server with “process inspection unavailable … invalid argument”.

## Spec

A shared procutil predicate classifies wrapped EINVAL, ESRCH and ENOENT from individual process reads as gone. Snapshots, launcher discovery and later diagnostic environment reads omit those processes and zombies. Table-wide failures, EPERM and other inspection failures remain fail-closed. Retain start-identity fences; a positively changed identity drops only the old snapshot generation. No retry, subprocess fallback or process signaling is added (ARCH-DRY, ARCH-ORDER).

## Done when

- Exited-between-list-and-read and zombie cases are excluded; valid servers remain discoverable.
- Permission, malformed and unknown failures still block uncertain candidates; start fences remain.
- Recorded race tests, real native zombie test and full recorded suite pass; operator detach/restart repro verifies the clean staged build before landing.



## Plan

- [x] Add shared gone classification and apply it to snapshots and native process-read callers, with RED/GREEN regression tests.
- [x] Build/stage clean HEAD for operator verification and pass the recorded suite.

Run the local close review now. Landing remains held until the operator's real detach/restart repro is confirmed by the TL.



## Log

### 2026-10-14

TL scope approval receipt d4dbaf0a-01d5-4876-a060-9f69f8f2c390 verified; quick flow authorized. #544 checkpointed at aa6ef7d2ea9e6900751a8667d86433fd0bdc426c; pair:0 stage cleared. Discovery: Snapshot preserves per-row procargs failures, launcher treats possible Zellij error rows as hard uncertainty, and diagnostics has a second environment read with the same exit race. Preserve genuine inspection failures; use one error-classification source.

- RED: controlled Table→command exit and real Darwin zombie retained a possible-live error row with invalid argument; reused/missing rows also survived as uncertainty. Caller tests reproduced gone/zombie errors during discovery and later environment reads. Shared predicate and filters added; permission/unknown failures and identity signaling fences remain unchanged. One old zombie fixture used absent argv to model unreadable-live; it now explicitly supplies EPERM so that test continues to defend genuine uncertainty.

- GREEN: procutil full race package PASS5.276s; launcher full race package PASS18.748s; diagnosticlog full race package PASS23.112s. Focused caller/reauthorization races also pass. Tests cover wrapped gone errnos, controlled exit between table and argv, a real Darwin zombie after a stale live table, PID reuse, continued permission uncertainty and later environment disappearance. Preparing clean staged build and recorded suite; operator detach/restart acceptance remains pending.

- Recorded full suite: sdlc local-check run --name suite PASS exit0 in10m53s on ec7a395a609a57d75ba58785f5b762b2671b1bcf, status sdlc-local/suite; log sha256 prefix81e4bc6e7c6b. Complete shell+Go checks green (launcher32.449s, diagnostics84.401s, procutil0.327s). Same commit staged and built cleanly on pair:0, bin/couch vcs.modified=false. Await operator-requested real detach/restart acceptance before landing; no claim of that smoke yet.

## Revisions

- 2026-10-14 11:53 PDT — TL receipt fe7ecadd-51a0-466b-9a0b-c730d637d04d requests local close review while the operator is at lunch. Separate the completed build/suite checkpoint from the remaining landing acceptance: keep pair:0 staged at ec7a395a609a57d75ba58785f5b762b2671b1bcf, run close through review, and hold merge until the TL reports the real repro outcome. The behavior scope and pre-landing smoke requirement are unchanged.
