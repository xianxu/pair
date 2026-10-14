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

- [ ] Add shared gone classification and apply it to snapshots and native process-read callers, with RED/GREEN regression tests.
- [ ] Build/stage clean HEAD for operator verification; run recorded suite, close review and land.



## Log

### 2026-10-14

TL scope approval receipt d4dbaf0a-01d5-4876-a060-9f69f8f2c390 verified; quick flow authorized. #544 checkpointed at aa6ef7d2ea9e6900751a8667d86433fd0bdc426c; pair:0 stage cleared. Discovery: Snapshot preserves per-row procargs failures, launcher treats possible Zellij error rows as hard uncertainty, and diagnostics has a second environment read with the same exit race. Preserve genuine inspection failures; use one error-classification source.
