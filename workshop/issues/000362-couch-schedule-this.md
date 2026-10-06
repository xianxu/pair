---
id: 000362
status: working
deps: [pair#365, ariadne#277, ariadne#279, ariadne#280]
github_issue:
created: 2026-09-30
updated: 2026-10-06
estimate_hours:
card_mirror: '0d817bf9641db727524d7a205425fa505d621620' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-06T11:40:57-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Teach Couch skill to schedule contextual work

## Problem

Contextual cross-slot scheduling needs more than sending a prompt: the coordinator must observe whether the requested SDLC effects happen and recover safely from missing or delayed delivery.

## Spec

Project: pair/workshop/projects/cross-slot-work-scheduling.md. Captured for operator review; no implementation is authorized by this issue creation.

Teach the Couch scheduling skill the cross-slot-work-scheduling contract. Send ordinary task instructions through local messaging; SDLC centrally owns claim-before-new-work and owner-checked continuation. Allow agents to use all available tools to query SDLC directly in another local worktree. Do not force queries through the slot actor or introduce multi-machine routing.

Distinguish queue acceptance, observed composer text, submitted user turn in the agent transcript and domain effects such as claim/gate completion. Continued TTY/transcript observation can resolve uncertain delivery; absence after an interval is not proof of loss. Observe effects after a reasonable interval (30 seconds is an example, not a fixed failure deadline) and decide to wait, follow up or retry using the operation’s recovery contract. Duplicated instructions must not start a second worker on someone else’s issue.

A successful claim is not proof work is progressing. Use owner, worktree/branch, activity and gate evidence together. Do not build a new centralized scheduler or global consistency engine; support agent reasoning with reliable commands.

## Done when

- Two slots receiving duplicate instructions produce one claim winner; other recipients report observed ownership instead of beginning duplicate work.
- A lost/delayed message exercise uses TTY/transcript and authoritative SDLC evidence to resolve progress or safely retry.
- Queries work with a recipient agent idle/unresponsive; read failures stay unknown and do not authorize takeover.
- Skill examples separate acceptance, submission, claim, progress and completion, and refer to SDLC recovery contracts rather than duplicating workflow rules.

## Plan

Durable plan: `workshop/plans/000362-couch-schedule-this-plan.md`.

- [ ] Task 1: `RenderLines` extracted from the scrollback renderer (byte-identical output)
- [ ] Task 2: `Couch.PeekSlot`: recent terminal tail, transcript paths, explicit unavailable reasons
- [ ] Task 3: `peek` operation and `couch --peek repo:N [--lines N] [--json]`
- [ ] Task 4: Couch skill scheduling section with the evidence ladder
- [ ] Task 5: full suite, then live exercises (duplicate claim, delayed delivery, idle recipient) recorded in the Log

## Log

### 2026-10-01

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.

### 2026-10-06 — design decisions

- **Reading another slot is in scope.** A coordinator may inspect another slot's
  recent terminal output or transcript, read-only, instead of asking the other actor
  "did my message arrive?". The operator's reasoning: Couch centralizes the context,
  and every actor can already read its own transcript, so a read-only look at a peer
  is acceptable. This deliberately departs from strict encapsulation, to keep the
  protocol simple and to get the benefit of the central view.
- **No combined couch+sdlc view for now.** If it is ever needed, it lives in couch,
  which calls `sdlc issue show`: couch → sdlc is the allowed direction, and sdlc
  never depends on couch.
- **Existing building blocks.**
  - Sending: `couch --send-to`, `--message-status`.
  - Liveness: `couch --actors --json`.
  - Work state, including milestone checkpoints and verdicts:
    `sdlc issue show N --json` (ariadne#279).
  - Retry semantics: `sdlc help recovery` (ariadne#280).
  - Pair's agent-agnostic recorded scrollback (`scrollback-<tag>-<agent>.raw` plus
    `.events.jsonl`) is the likely base for the read command; being traced.

## Revisions

### 2026-10-06 — narrowed to mechanisms; orchestration is a follow-up

The operator narrowed #362 to the mechanisms of cross-slot scheduling:
- sending a scheduling message to another local slot;
- knowing where and how to pull the state of scheduled work.

Milestone and gate state are named explicitly as observable: a closed milestone (its
`closed Mx` log line and `Review-Verdict` trailer) and a closed issue are progress
evidence the coordinator reads, alongside owner, branch and activity.

Out of scope, and going to a follow-up issue (being brainstormed):
- the TL ↔ worker protocol for whether a task may close autonomously or needs the
  operator's smoke test;
- a "ready for smoke test" state, since `codecomplete` blurs "review passed" and
  "operator verified";
- tracking which assigned work awaits a smoke test;
- where the TL sits;
- the end-to-end test of driving a project from one TL slot.
Ergonomics and scale are also left to later issues. The Done-when bullets above are
unchanged: they already describe mechanisms.

### 2026-10-01 — contextual scheduling scope

Expanded the existing blank scheduling issue into effect-verified coordination for cross-slot-work-scheduling; transport reliability is delegated to #365 and work admission/retry semantics to Ariadne SDLC.
