---
id: 000874
status: open
deps: [pair#737, ariadne#564]
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '39c9a6dc9db153e90827b7c19336f2291ed06e9e' # card fields mirrored from issue-cards; edit via sdlc
---

# Scheduler recovery wave-three gate

## Problem

The operator approved ariadne#564 with three recovery waves after a full shutdown: pending-work asleep slots, remaining asleep slots, then the scheduled-task system. Scheduled tasks belong to the one Couch (pair#737), while each hub can report only its own worker-wave completion. A hub enabling automation on the first operator message would resume scheduled work before the machine-wide recovery barrier.

## Spec

Couch/scheduler ownership follow-up in pair#737's area, requested by ops:0 on 2026-10-24 (submitted receipt 25d26f1d-9e80-483b-829f-7499e93e9f04).

After a full shutdown, hubs start in a quiet hold, automation off and workers asleep. Nothing resumes because a hub started, a flag disappeared, or an ordinary operator message arrived. Explicit operator-authorized worker recovery runs each hub's pending-work wave first and remaining-asleep wave second. ariadne#564 implements those bounded worker waves and reports an explicit handoff; it neither controls the scheduler nor stores a second machine-wide scheduler state.

The machine steward (currently ops:0, the Couch restart owner) resumes the scheduled-task system only after every hub has reported both worker waves complete, or the operator explicitly says go. No individual hub toggles the scheduled system alone. State the supported scheduler/status/control seam for this third wave and how the steward verifies completion or an uncertain control reply. Preserve the existing scheduler missed-run policy and ownership boundaries; do not infer success or replay missed work from a restart.

This is a contract to design with the owning TL/operator, not authorization to execute live recovery or to choose a new durable barrier mechanism. Existing per-hub couch --automation controls are the present public seam; ariadne's help will stop recommending enablement on the first operator message. Pair owns any missing scheduler-side capability or enforcement.

## Done when

- The scheduler's documented startup/resume control supports the machine-wide third-wave contract and identifies the machine steward's authority; per-hub completion alone cannot enable the scheduled system.
- Private stateful/disposable-Couch tests show that due notices and direct scheduled execution remain held during the quiet period and incomplete worker waves, and that the supported steward action after all handoffs (or explicit operator override) resumes according to existing missed-run semantics.
- Missing, failed or uncertain completion/control evidence stays visible and held, with one supported reconciliation action rather than an automatic resend.
- The producer seam and qualification evidence are linked to ariadne#564; no live fleet restart/wake is performed without separate operator authorization.

## Plan

- [ ] Design in pair#737's scheduler area with its owner; reuse existing Couch authority where sufficient.

## Log

2026-10-24 — Filed unclaimed from an existing resting Pair checkout; creation writes no checkout. The primary checkout was detached, so the first call refused before creation; no checkout was switched or edited. Read-only inspection found current per-hub automation authority in cmd/internal/couchcmd/automation.go and pair#737 still working/not landed in the tracker. Pair's root MEMORY.md is absent; AGENTS.local.md was read. ariadne#594 retains its existing merge waiter; no Pair source or runtime policy was changed.
