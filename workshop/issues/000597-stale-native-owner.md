---
id: 000597
status: working
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: 'f83e7afcbe689cb5e7ae098f53cca30fc2a348d2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T20:32:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "2aadeb9b", done: "105f9900"}
---

# Reject stale legacy native owners

## Problem

Tonight's main Couch restart parked ariadne:4, then refused its resume because legacy readiness named reused PID 1111, now an unrelated system service born before the wrapper. ariadne:5 also refuses on lsof timeouts for unrelated legacy owners under load (TL receipt 368e95f0).

## Spec

Couch Resume from disk / Relaunch retire their own attributed processes, then launch the proven transcript without global native-owner/lsof/ledger forensics or reservation admission. Cross-slot checking uses only Couch's recorded conversation identities and positively live slot registry: an exact collision refuses with its slot; unknown IDs do not refuse. Fresh never runs ownership admission and advances only its validated exact reserved thread-address marker. Failed attempts remain startable through existing Fresh replacement/archiving, preserving evidence. Ordinary standalone Pair is unchanged until #595.

## Done when

- Headless tests show stale unrelated process/readiness evidence cannot block Couch recovery; same recorded conversation in another live slot refuses, own/dead/unknown identities do not.
- Failed native relaunch followed by Fresh consumes a validated reserved address and remains retryable; busy/kill-not-confirmed safety remains.
- Targeted/race checks, build, required recorded suite and mandatory SHIP review pass; report the supported patched-build recovery command and landing hash to hub, then return to #595.

## Plan

- Remove forensic/store admission from Couch recovery only, reuse existing own-process retirement and validate registry collisions at both recovery gates.
- Keep the proven Fresh reservation fix; test exact identity, retry, false-positive controls and no forbidden scan calls.
- Verify, close and land; TL owns restage and live recovery. Deletion of dead guard code remains #595.

## Log

### 2026-10-15

Urgent TL dispatch 368e95f0 verified submitted. #595 checkpointed at eb6528a5; this independent hotfix starts from main, with no stacked code. No live process or slot changes authorized or performed.

## Revisions

- 2026-10-15, TL 52593bb1: include stale established registration after park-ok-resume-failed and later Fresh refusal. Read-only diagnosis first; no live mutations. Keep existing claim authority and rollback, no new recovery mechanism (ARCH-DRY). Scope remains a narrow correction while #595 replaces the guard.

### 2026-10-15 — operator redirect (TL f4341d43)

Supersedes the legacy PID/lsof patch: resume/relaunch retire the slot's own attributed processes and restart the proven transcript. Remove global native-owner/store admission calls from Couch recovery launch paths; deletion stays #595. The only cross-slot refusal is another positively live Couch registry entry recording the exact same agent/conversation ID, naming its slot. Unknown recorded IDs do not manufacture collisions. Fresh skips native ownership completely and consumes its validated reserved address; stale failed-attempt evidence is preserved/archived through existing Fresh replacement. Refusals are busy, own-process retirement unconfirmed, or exact conversation live elsewhere. Build/test this narrow path correction, no live mutation; TL restages and recovers ariadne:4/5.
