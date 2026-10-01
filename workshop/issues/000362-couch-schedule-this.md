---
id: 000362
status: open
deps: [pair#365, ariadne#277, ariadne#279, ariadne#280]
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: 'cdcefb2e19829957bf093fa02ee07c8a8a51f3b5' # card fields mirrored from issue-cards; edit via sdlc
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

Implementation plan to be designed after issue claim and start-plan; these are requirements, not an approved implementation plan.

## Log

### 2026-10-01

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.

## Revisions

### 2026-10-01 — contextual scheduling scope

Expanded the existing blank scheduling issue into effect-verified coordination for cross-slot-work-scheduling; transport reliability is delegated to #365 and work admission/retry semantics to Ariadne SDLC.
