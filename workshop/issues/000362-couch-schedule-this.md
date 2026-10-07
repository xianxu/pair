---
id: 000362
status: codecomplete
deps: [pair#365, ariadne#277, ariadne#279, ariadne#280]
github_issue:
created: 2026-09-30
updated: 2026-10-06
estimate_hours:
card_mirror: '107baa9371b8d4ed8207232f457e369aacd0319e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-06T11:40:57-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 2.62
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

- [x] Task 1: `RenderLines` extracted from the scrollback renderer (byte-identical output)
- [x] Task 2: `Couch.PeekSlot`: recent terminal tail, transcript paths, explicit unavailable reasons
- [x] Task 3: `peek` operation and `couch --peek repo:N [--lines N] [--json]`
- [x] Task 4: Couch skill scheduling section with the evidence ladder
- [x] Task 5: full suite, then live exercises (duplicate claim, delayed delivery, idle recipient) recorded in the Log

## Log

### 2026-10-01

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.

### 2026-10-06 — live exercises (Task 5)
- 2026-10-06: closed — Round 1's verdict was lost behind the reviewer's final message; recovered from its transcript: FIX-THEN-SHIP, no Critical, one Important (plan Core concepts drift) fixed as plan revision (c), minors fixed (json encode error printed, shadowed import, peek recorded as CLI-only, atlas wrap) in 59b077bb. Tasks 1-5 done: RenderLines byte-identical (TestRenderLinesMatchesTheRenderedFile); RenderOwnedLines under the retention lease (mutation caught); PeekThread tests; CLI/declaration/arity/atlas audits. Live exercises (issue Log): duplicate claim on throwaway pair#401 -> one winner pair:5, pair:6 reported the owner back to the sender and did not start; held delivery -> peek showed the occupied composer, receipt expired with reason after 30s; parked pair:0 and idle pair:6 -> peek and sdlc issue show answered, unreadable sources named. Full suite unsandboxed: only failures match origin/main (spawn registration, continuation writer, cold-resume flake, launcher x5, workbenchshortcut, gcruntime locator, embedded-runtime; artifactpath set identical).; review verdict: SHIP
- 2026-10-06: flow upgraded quick → full — 371 added lines in code files (limit 100); an earlier round of this close already ran the full review

The exercises ran from `pair:1` with the branch's `couch` build, against two
scratch slots (`pair:5`, `pair:6`, both claude) and the throwaway pair#401.

1. **Duplicate work.** "Please work on pair#401" went to `pair:5` and `pair:6` back
   to back.
   - Both receipts reached `submitted`, and peek showed `pair:5` running
     `sdlc claim --issue 401`.
   - `sdlc issue show 401 --json` showed card `working`, with `assignment.claimant`
     at workspace `pair:5`.
   - `pair:6` reported "The claim lost: pair:5 owns it" and started nothing.
   - Its reply to the sender was blocked by the operator's `couch` dev shell
     function: it runs `go build ./cmd/couch` in the slot's own checkout, which
     fails in a fresh slot because the embedded runtime assets are produced by
     `make`. The agent fell back to `~/workspace/pair/bin/couch`. This is an
     environment finding, not a Couch defect.
2. **Held delivery.** A message went to `pair:5`, whose composer held text
   (`unclaim 401`, apparently Claude Code's prompt suggestion).
   - Peek showed the composer occupied and no envelope; the receipt read
     `delivering`.
   - After 30 seconds: `expired | delivery deadline elapsed: waiting for previous
     automatic input to clear`.
   - Finding: the receipt's `Detail` is empty while the message is held, and the
     reason only arrives with the outcome. The skill text was corrected to say so,
     and to direct the coordinator to peek for the live reason.
3. **Idle or stopped recipient.**
   - `couch --peek pair:0`, a parked codex thread whose agent is not running,
     showed its last screen from the recording and the sent-prompt log. It named
     the unreadable transcript ("native session has no exact established outgoing
     binding") instead of returning nothing.
   - `sdlc issue show 401 --json` answered with `pair:6` idle: card, assignment
     and landing present; workspaces, branch, checkpoints and completion `absent`
     (read, none recorded), none `unknown`.

Bug found during setup: with `pair:0` parked, the switcher offered no add slot.
Filed as pair#402, a separate branch.

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
