---
id: 000367
status: working
deps: [pair#366, ariadne#277, ariadne#278, ariadne#279, ariadne#280]
github_issue:
created: 2026-10-01
updated: 2026-10-02
estimate_hours:
card_mirror: 'eb186c2373aeebbe7a59503fb223d080066f8af6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-02T11:50:33-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
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

### 2026-10-02 — surface survey after claim (sdlc 3516e1bf, ariadne#277–280 landed)

What exists:
- `sdlc issue show N --json` (schema v1) is the per-issue observation: claimant
  {operator, machine (hashed), machine_name, workspace "pair:N", worktree,
  repository}, `relation` (this-workspace / other-workspace / unattributed /
  unknown, judged from the cwd checkout), `claimant_worktree` (holds-branch /
  elsewhere / missing / other-machine / unknown), `workspaces.holding[]` with
  dirty/ahead counts and `is_claimant`, and card status/revision. Foreign
  machine = other-workspace + other-machine; legacy = unattributed (5 such
  working issues today: #121, #207, #253, #278, #292).
- `sdlc fleet inventory --json`: every worktree's git facts plus the issue its
  branch prefix names; no claimant, no slot address.
- `sdlc workspace [addr] --json`: resolves `:0` / `pair:N` to kind, path, branch.
- `sdlc help recovery` (#280): claim/reclaim convergent-retry, issue show
  read-only; verify effects via `issue show --json`, never via message receipts.
- `sdlc reclaim`: inspect-then-`--expect REV` CAS, operator-directed only.
- Couch: per-slot `.couch/thread.json` + global records; states live, detached,
  parked, busy, unusable(reason), archived. `OSSlotCatalog.Discover` already
  enumerates slots by the `worktree/<repo>-slotN/<repo>` rule joined with
  `git worktree list`, each verified through `sdlc workspace --json`.

Gaps against the Revisions' assumptions:
1. Couch does not start a preallocated slot set on restart; it only reattaches
   detached/unknown agents. Parked slots stay parked.
2. No operator/Couch-originated delivery: `couch` peer messaging requires a
   live registered slot as sender; there is no way to inject "continue #N"
   from outside a slot.
3. No machine-readable Couch inventory outside a slot (`--list` is text;
   `--actors --json` is live-only, in-slot only).
4. No bulk "claims naming this machine" query; `issue list` has no `--json`.
   Per-issue `issue show --json` costs a tracker read each.
5. Deliberate park vs crash-park is only implied by `verified_park`.

Scope decisions requested from the operator before the durable plan.

## Revisions

### 2026-10-01 — recovery shape settled in operator discussion

Reason: operator review of the captured spec. Delta (still requirements, not a plan):

- **Slots are live at recovery time.** Couch will start a preallocated slot set (resuming parked slots) on restart, so recovery does not have to avoid waking stopped agents. Started slots come up idle; being started does not mean being assigned work.
- **Two steps.** (1) A central, read-only allocation report joins sdlc claims, Couch inventory and per-slot worktree state (branch, uncommitted files, whether the issue is still open) into one row per slot. (2) An operator-approved assignment step: Couch delivers "continue #N" to matched slots only. Conflicts, parked intent, foreign-machine and legacy-owner rows wait for the operator. Delivery goes through Couch, not peer-to-peer slot messaging.
- **Slot-side self-check.** Before working, an assigned slot checks branch → issue → "the claim names me". On a mismatch it reports and does not work. This is a local deterministic guard, not a messaging protocol.
- **Scan scope.** Slots are the repo's `:0` checkout plus `worktree/<repo>-slotN` worktrees; other worktrees (e.g. scratchpad detached-HEAD ones) are not slots. Sources: Couch inventory (switcher view minus archived), claims assigned to this machine, and `git worktree list` filtered by that rule. No filesystem crawl.
- **Lost-slot evidence.** Claim with no slot (no Couch row, or worktree gone) and worktree with no Couch row are both inspected and reported as evidence, naming which source is missing. Neither triggers an automatic reclaim or adoption.
