---
id: 000367
status: open
deps: [pair#366, ariadne#277, ariadne#278, ariadne#279, ariadne#280]
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'cd654ed7c2bb20f18114d375092b6994454cfdb6' # card fields mirrored from issue-cards; edit via sdlc
---

# Recover local slots from durable issue ownership

## Problem

After restart, humans must remember which local slots owned which issues. Recovery needs to reconcile durable responsibility with local runtime and repository evidence without asserting a globally consistent live world.

## Spec

Project: pair/workshop/projects/cross-slot-work-scheduling.md. Captured for operator review; no implementation is authorized by this issue creation.

Provide a recovery skill consuming canonical read-only SDLC observations and Couch’s local inventory, including parked/stopped slots. Derive a reviewable recovery report: matching assignment and worktree, already running, stopped, deliberately parked, missing worktree, conflicting work, legacy/unattributed claim or foreign machine ownership.

At the operator’s request resume assigned local work in its existing slot (e.g. pair:4 owns #444), preserving branch, dirty files and usable conversation history. Inspect parked state without blindly waking intentionally parked work. Reclaim only under explicit operator direction, after out-of-band synchronization where needed; a crash does not release a claim. Keep observations in the owning binaries; the skill explains/applies them instead of implementing another state scanner. No remote-machine execution/proxy support.

## Done when

- A restart exercise reconstructs assignments and produces a clear report for every relevant local slot, with missing/unknown evidence retained.
- Operator-directed recovery resumes the matching issue/worktree and leaves already running work alone.
- Conflicting dirty work, parked intent, foreign-machine assignments and legacy missing owners do not trigger destructive takeover or automatic reclaim.
- The skill uses documented commands and is discoverable; its recovery steps are tested with stateful fixtures and a local restart acceptance case.

## Plan

Implementation plan to be designed after issue claim and start-plan; these are requirements, not an approved implementation plan.

## Log

### 2026-10-01

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.

## Revisions

### 2026-10-01 — recovery shape settled in operator discussion

Reason: operator review of the captured spec. Delta (still requirements, not a plan):

- **Slots are live at recovery time.** Couch will start a preallocated slot set (resuming parked slots) on restart, so recovery does not have to avoid waking stopped agents. Started slots come up idle; being started does not mean being assigned work.
- **Two steps.** (1) A central, read-only allocation report joins sdlc claims, Couch inventory and per-slot worktree state (branch, uncommitted files, whether the issue is still open) into one row per slot. (2) An operator-approved assignment step: Couch delivers "continue #N" to matched slots only. Conflicts, parked intent, foreign-machine and legacy-owner rows wait for the operator. Delivery goes through Couch, not peer-to-peer slot messaging.
- **Slot-side self-check.** Before working, an assigned slot checks branch → issue → "the claim names me". On a mismatch it reports and does not work. This is a local deterministic guard, not a messaging protocol.
- **Scan scope.** Slots are the repo's `:0` checkout plus `worktree/<repo>-slotN` worktrees; other worktrees (e.g. scratchpad detached-HEAD ones) are not slots. Sources: Couch inventory (switcher view minus archived), claims assigned to this machine, and `git worktree list` filtered by that rule. No filesystem crawl.
- **Lost-slot evidence.** Claim with no slot (no Couch row, or worktree gone) and worktree with no Couch row are both inspected and reported as evidence, naming which source is missing. Neither triggers an automatic reclaim or adoption.
