---
id: 000367
status: working
deps: [pair#366, ariadne#277, ariadne#278, ariadne#279, ariadne#280, ariadne#288, ariadne#289, pair#363]
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

(Rewritten 2026-10-03; see the Revision of that date. The original bullets are
superseded.)

- A read-only Couch report (`couch --slots --json`, name TBD) reconstructs, after
  a restart, one row per slot and per claimed-not-done issue of this machine,
  joining `sdlc fleet inventory` (claims, dangling claims, slot verdicts) with
  Couch's slot and thread state; each row carries git state, disk state, agent
  state and a suggested next step with its reason. Missing, stale or unknown
  evidence is shown as such, never as absence.
- Rows that are unsafe for automation (dirty or untracked files, an active Git
  operation, a dangling claim, a foreign or unattributed claim) suggest no
  automatic step and say why.
- `resume <slot>` and `reboot <slot>` (pair#363's operations) are callable through
  the running Couch's socket from a live Couch slot only; any other caller is
  refused; results and refusals are typed.
- Couch's skill (`couch --skill`) documents the recovery procedure an agent
  follows: read the report, review with the operator, resume/reboot per row,
  delegate disk fixes to the slot's own agent via `--send-to`, verify by
  re-reading; it never acts on a slot flagged for recovery.
- Tested with stateful fixtures covering every report row class and the caller
  rule, plus a restart acceptance case through the real report path.

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

### 2026-10-02 — scope settled after the surface survey

Reason: the survey (Log, same date) found three assumptions of the previous
revision unmet. Operator decisions, delta:

- **Slot-set start is pair#384.** Couch starting the enrolled slots (idle) on
  restart is its own issue; #367 depends on it and does not start slots.
- **Delivery is ordinary Couch peer messaging from a live coordinator slot.**
  A project-management thread (typically `:0`) is always live. After the report,
  the operator decides: "schedule #N", or ask that thread to schedule
  everything by dependency and free slots (the latter is pair#362's ground).
  No new Couch sender identity. Effects are verified through `sdlc issue show
  --json`, per #280, not through message receipts.
- **Bulk claim observation is ariadne#288.** The report consumes one read-only
  bulk claim query instead of a per-issue `issue show` loop; #367 depends on it.
- **The report is a deterministic `couch` subcommand** that owns the join
  (Couch slot inventory × claims × slot worktree state); the skill explains its
  rows and drives the operator-approved assignment, and reimplements no scan.

### 2026-10-02 — slot view absorbed from pair#384; pair#384 closed

Reason: operator design review of pair#384 (closed wontfix). Delta:

- **No automatic slot start.** Couch startup stays reattach-only; parked slots
  may be ready, stale or corrupted, so nothing resumes them unasked. The
  earlier revision's "slots are live at recovery time" no longer holds.
- **This issue owns the Couch slot view:** one row per slot joining ariadne#289's
  per-slot readiness verdict (from `sdlc fleet inventory`) with Couch's thread
  state (live / detached / parked / unusable) and ariadne#288's claims. Couch
  reimplements no git scan.
- **Recovery actions are pair#363's.** Bulk resume of slots holding work and
  bulk reboot of N ready slots fold into #363; this issue adds the claim join,
  the report, and the operator-approved "continue #N" step on top.
- deps: pair#384 dropped; ariadne#289 and pair#363 added.

### 2026-10-02 — owns bulk recovery and Couch-owned workspace shaping

Reason: operator design review; pair#363 is settled as actor-only. Delta:

- **Two layers.** pair#363's resume and reboot get a live agent into a slot
  without reshaping the worktree. This issue is automation: it also wants the
  worktree in a known shape, and composes per slot from ariadne#289's verdict:
  holds-work → #363 resume, then "continue #N"; ready and wanted for new work →
  shape the workspace, then #363 reboot for a fresh agent; needs-recovery /
  missing / unknown → report why and leave it.
- **Bulk commands live here:** resume the slots holding work; prepare N ready
  slots for new work (names TBD).
- **Couch owns workspace shaping**, because Couch owns the named slot
  worktrees: re-check the verdict at action time, switch every checkout of the
  slot to its resting branch (from `sdlc workspace`), run `weave refresh`
  (which refuses dirty trees and active Git operations). Never stash, reset or
  discard. sdlc supplies only the verdict and resting-branch name; weave the
  refresh.
- **Known constraint:** a second `couch` invocation cannot run operations while
  the console runs (owner routing deferred to #147); bulk commands need a route
  to the running Couch or must be triggered from inside it.
- A deleted slot directory is pair#387's repair, not this issue's.
- deps unchanged (pair#363 stays: resume and reboot are its primitives).

### 2026-10-03 — recovery as LLM-driven steps 1-3 over Couch primitives

Reason: operator design session after pair#363 landed. Delta:

- **Steps.** (1) Git: what this machine claims and has not finished. This is
  already `sdlc fleet inventory --json` (`rows[].claims`, `dangling_claims`,
  `slots[]`); no new sdlc work. (2) Git vs disk per slot, and (3) whether a live
  agent is there: a new read-only Couch report that also derives a recovery
  plan, the steps that make each slot ready for an agent to resume. (4) Driving
  agents to continue, the TL-in-`:0` workflow and notifications, is not this
  issue (pair#362 and later).
- **LLM-driven first.** An agent (typically the TL in `:0`) follows the report
  and a skill section, calling Couch primitives. A deterministic one-shot
  verb comes later, once the plan's row classes prove stable.
- **Primitives on the existing socket.** `resume <slot>` / `reboot <slot>` run
  in the running Couch through its operation queue. Callers are live
  registered Couch slots only (the `--send-to` rule); outside callers come later.
- **Disk fixes belong to the slot's own agent.** The TL resumes the slot, then
  asks its agent (existing `--send-to`) to restore its workspace per the plan.
  No agent acts in another slot's repository.
- **Removed from scope:** preparing N ready slots for new work (archive,
  resting branch, `weave refresh`, fresh agent) moves to step 4 / pair#362, and
  so does Couch-owned workspace shaping.
