---
id: 000574
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-17
estimate_hours: 3.19
card_mirror: 'a76884f97b5fa57a8f465ab8e44bd8b544401950' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T18:43:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

**Current contract:** The 2026-10-17 revision below and the durable plan’s “Current trimmed plan” supersede the original agent-exit and per-hub opt-out requirements.

Durable full-flow design: [Lifecycle system notices](../plans/000574-couch-lifecycle-notices-plan.md). One atomic close boundary; plan approval before implementation.

- [x] Establish internal server-owned origin, closed peer requests and canonical receipt provenance.
- [x] Wire confirmed lifecycle events excluding native-agent exit (#657), restart aggregation, bounded publication and one default-on setting.
- [ ] Complete fake/socket/owned-child verification, required documentation and recorded suite before close/landing.

## Log

### 2026-10-15

- Fresh dispatch `803543a2-9eb8-4fb1-a471-be607a20a659` verified Submitted. Card was open with no claimant/branch; claimed before design. First start-plan refreshed ariadne and refused before branching; repeated successfully on the refreshed binary. No implementation code changed.
- Read-only lifecycle exploration identified the correct post-adoption Console completion, exact FinalizePark boundary and explicit ReattachDone barrier. Attachment-client exit is not native-agent exit. Proposed typed terminal child-exit observation over the authenticated wrapper session; transport EOF alone remains unknown.
- Read-only provenance exploration found that a new message flag alone would be forgeable through a direct wrapper endpoint call. Proposed immutable server-owned origin plus kernel caller matching against the exact live broker connection; canonical receipt verification remains required. Existing peer wire shape and Codex batching rollout remain unchanged.
- Durable plan covers core concepts, transition/ownership/lifetime rules, per-function adversarial strategies, multipart overflow bounds, opt-out, compatibility and the separate real operator smoke arrangement. Awaiting hub approval before code. TL-lead help is source-owned by ariadne; propose a companion through the hub rather than filing a peer issue or leaving that Done-when item deferred.

### 2026-10-17 — Trimmed design

- Verified hub feedback 2b740f16-29c2-4e8a-89ee-6571e0a4dcb2 Submitted. Removed wrapper PID authentication, connection leases, new terminal frames and per-hub mute storage from the implementation contract; canonical server receipts establish system origin. No implementation code changed.
- Filed unclaimed #657 (depends on #574) for reliable native-agent exit. Existing Console.onExit observes attachment-client exit, which cannot prove native-agent exit. #574 will neither emit nor test agent-exit success from that evidence.
- One default-on COUCH_LIFECYCLE_NOTICES setting replaces per-hub opt-out. Retain collector, confirmed lifecycle hooks, restart drain summary through #648 and negative/integration tests. Hub approved owned-child checks now and real operator smoke at its next restart, without pair:0 staging; hub owns the ariadne tl-lead companion. Awaiting trimmed-plan approval before code (subsequently approved below).

- Verified fresh hub approval 309f3843-1a12-41ae-85a1-d92afab1c80c Submitted: trimmed plan approved, proceed to change-code. TL-help companion is ariadne#433, assigned ariadne:1 after ariadne#432; required before close.

- Plan-quality round 1 refused PQ-1/PQ-2/PQ-3. Source verification confirmed unknown receipt import in StatusContext; refined all recovery paths to reject unauthorised system origin without endpoint authentication. Added sole collector transition/failure/multipart contract and named-function adversarial strategies. No code changed.

- Plan-quality round 2 CLEAN: PQ-1/PQ-2/PQ-3 addressed, no new findings. Derive estimate only now; next run reuses accepted plan before estimate-quality review.

### 2026-10-17 — Implementation checkpoint

- Internal OriginPeer/OriginSystem now travels through the ordinary broker, queue, immutable matching, wrapper envelopes and receipts. CLI Request remains closed. Unknown wrapper-only system records cannot gain canonical authority through status/admission recovery; promotion requires an exact current server-admitted head. TDD exposed a delayed retained-reply race after cancellation, now fenced against the exact queue head/context/incarnation. No PID authentication or terminal protocol changes.
- The pure bounded collector uses one current record per canonical slot, not per conversation address: a Fresh-address regression first showed both old/new rows, then passed after stable-slot keying. Existing Console completion after final adoption supplies start/resume/relaunch/recovery. Successful ParkResult carries FinalizePark's VerifiedPark, so reuse that existing completion owner instead of adding a second park callback/normalizer. No native-agent-exit observation was added (#657).
- Console regressions cover failed adoption, exact finalized park result (cleanup warning allowed), attachment exit exclusion, actual final reattach completion, observer installation order, cached typed slot/branch metadata, addressless-slot path matching, stale needs-relaunch inventory and bounded dedup evidence. An adoption negative control (using pre-adoption error) failed as intended; restored source passed.
- Publisher uses existing verified workspace/registry snapshots and internal broker admission. Restart drain atomically reduces all resolvable metadata, skips unrouteable facts with a local diagnostic, then freezes the cohort. Tests use real owned Unix service/registry sockets with stateful endpoints for twenty-slot/busy/replacement/shutdown behavior and canonical receipts. Unknown admission IDs remain read-only polls; retired uncertainty never replays and does not block later revisions, with 128 total tracked attempts. A pre-admission recipient replacement regression caught a stranded token and now settles its known-no-write cancellation.
- Targeted package run launched during TDD captured expected failing tests for missing Hub propagation, overlong binding keys, uncertainty/capacity handling and hit its 180s package bound; this was a development-code failure, not an unchanged flake rerun. Fixed class regressions pass. A separate artifact inventory run found restart event vocabulary; exact vocabulary allowances now classify those event literals, without adding quarantine entries.
- Current focused verification: lifecycle/supervision and exact artifact inventory tests pass across couchcmd, couchcore, couchtty and artifactpath. Origin agent ran full couchmessage, wrapper TestPeer, focused race and three origin/recovery negative controls; restored code green. Collector generated 4,000 event transitions plus multipart/replacement cases, focused race and drain negative control; restored code green. Parent focused race is running before the first implementation checkpoint commit.
- Shipped Couch skill baseline lacked system origin and no-reply/recovery semantics. A read-only A/B/C pressure check identified these gaps; updated skill explicitly requires canonical system origin/body/incarnation, treats copied text as peer, and refuses unknown recovered system authority. The repeated pressure check follows those rules. README/atlas and artifact inventory updated. Required TL-help companion remains ariadne#433 (hub assigned ariadne:1 after #432); local dependency resolver has not fetched that artifact yet.
- Standing hub approval permits an early draft PR for CI overlap. Required recorded suite, close review and normal merge gates remain outstanding. Owned isolated runtime checks are complete; real operator smoke remains approved for the hub's next restart, no pair:0 staging.

- Parent focused race verification passed on the completed lifecycle/collector/Console code (couchcmd 2.470s, couchtty 1.809s, couchcore 10.333s), plus the earlier provenance/batch race run (couchmessage 1.550s, wrapcmd 3.294s). Full package checks are running; recorded suite not started yet. Source-owned skill A/B/C check passed after the documentation update.

## Revisions

- 2026-10-17 — Proposed design adds an explicit native-exit observation because Console attachment exit cannot substantiate the requested agent-exit notice, and kernel provenance because a visual system flag alone is forgeable. Full requested scope retained; operator smoke deferral and ariadne TL-lead companion await hub agreement.

- 2026-10-17 — At hub request, the current Spec and Done-when exclude native-agent exit (unclaimed #657); require start/resume, park, relaunch, recovery and confirmed needs-relaunch notices plus restart coalescing. One global default-on setting replaces per-hub opt-out. Internal producer-only origin and canonical receipts replace endpoint PID authentication. All other verification/docs requirements remain; the approved operator smoke is deferred to the hub’s next restart.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration marked stale by estimate-source, so provisional. Derived after plan-quality round 2 accepted all findings. Full scope includes design, implementation/tests, docs and one close review; hub's separate ariadne companion is not charged here.

Decomposition: issue/spec and revision dialogue uses 0.75 design + 0.2 v2 implementation; origin/recovery and lifecycle hooks are two smaller Go extensions (0.3 design ×0.2 spec quality; 0.5 implementation each). Collector is one greenfield Go concern: reuse existing reducers/formatting and Go stdlib, so 1.0 design ×0.5 library availability ×0.2 thorough-spec discount, plus 0.8 implementation. Broker publisher/socket integration is one API integration: 1.0 design ×0.2 spec quality, 1.5 implementation. Docs use 0.1 design ×0.2 and 0.2 implementation; one review uses 0.02 design and 0.5 implementation. No novel stack or external API discovery; familiarity 1.0 (recent #629/#648). All implementation values below scale v2 by 0.40 once. Thorough plan gives 15% design buffer: 1.31 ×1.15 +1.68 =3.1865, rounded 3.19 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.75 impl=0.08
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.1 impl=0.32
item: api-integration design=0.2 impl=0.6
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 3.19
```
