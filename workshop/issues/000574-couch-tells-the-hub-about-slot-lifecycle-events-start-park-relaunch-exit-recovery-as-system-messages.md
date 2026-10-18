---
id: 000574
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-17
estimate_hours:
card_mirror: 'e161339e6f34ce44dbb258f65ab04306727b3e02' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T18:43:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Couch tells the hub about slot lifecycle events (start, park, relaunch, exit, recovery) as system messages

## Problem

The hub (TL) isn't told about slot lifecycle events. When the operator parks a slot (Alt+n), relaunches it, brings a
new slot up (parley.nvim:3 on 2026-10-15), or a slot's agent exits, the TL learns it only by polling `couch --actors`,
by peeking, or from a worker's own report. Tonight's evidence: stalled slots went unnoticed for hours, and messages to a
relaunched slot were lost. The reverse direction already exists: on connect, Couch sends a worker a "Couch restarted
and the hub is available…" notice, attributed to ops:0.

## Spec

- Couch sends the hub a **system lifecycle message** for each slot event: started (new or resumed), parked (Alt+n),
  relaunched, agent exited, recovered after a Couch restart (Continue / Resume / Start fresh), and needs-relaunch.
  One line each: slot, event, agent/model, branch and issue if known, and whether it was operator-initiated.
- Clearly marked as a system message (not from a peer slot), so the hub's rules treat it as information, never as a
  request; it's batched or coalesced so a Couch restart's many reattaches produce one summary, not twenty messages.
- Opt-out per hub (some operators may not want it), default on when a TL hub is configured.
- The worker-side "hub is available" notice should say it's from Couch, not "from ops:0".

## Done when

- Tests: park, relaunch, new slot, agent exit and post-restart recovery each produce the expected hub message; a
  restart's reattaches coalesce into one summary.
- `couch --skill` documents the messages; tl-lead says how the hub treats them (update Slots, no reply).

## Plan

Durable full-flow design: [Lifecycle system notices](../plans/000574-couch-lifecycle-notices-plan.md). One atomic close boundary; plan approval before implementation.

- [ ] Establish server-owned origin and protect system delivery against peer/body/direct-endpoint forgery.
- [ ] Wire confirmed lifecycle and native-exit events, restart pass aggregation, bounded multi-hub publication and opt-out.
- [ ] Complete fake/socket/owned-child verification, required documentation and recorded suite before close/landing.

## Log

### 2026-10-15

- Fresh dispatch `803543a2-9eb8-4fb1-a471-be607a20a659` verified Submitted. Card was open with no claimant/branch; claimed before design. First start-plan refreshed ariadne and refused before branching; repeated successfully on the refreshed binary. No implementation code changed.
- Read-only lifecycle exploration identified the correct post-adoption Console completion, exact FinalizePark boundary and explicit ReattachDone barrier. Attachment-client exit is not native-agent exit. Proposed typed terminal child-exit observation over the authenticated wrapper session; transport EOF alone remains unknown.
- Read-only provenance exploration found that a new message flag alone would be forgeable through a direct wrapper endpoint call. Proposed immutable server-owned origin plus kernel caller matching against the exact live broker connection; canonical receipt verification remains required. Existing peer wire shape and Codex batching rollout remain unchanged.
- Durable plan covers core concepts, transition/ownership/lifetime rules, per-function adversarial strategies, multipart overflow bounds, opt-out, compatibility and the separate real operator smoke arrangement. Awaiting hub approval before code. TL-lead help is source-owned by ariadne; propose a companion through the hub rather than filing a peer issue or leaving that Done-when item deferred.

## Revisions

- 2026-10-17 — Proposed design adds an explicit native-exit observation because Console attachment exit cannot substantiate the requested agent-exit notice, and kernel provenance because a visual system flag alone is forgeable. Full requested scope retained; operator smoke deferral and ariadne TL-lead companion await hub agreement.
