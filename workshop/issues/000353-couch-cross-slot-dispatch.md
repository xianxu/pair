---
id: 000353
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '43b82c291057168cd309e010eda1ee731df5ae6b' # card fields mirrored from issue-cards; edit via sdlc
---

# Live cross-slot dispatch between couch slots

## Problem

Using several Couch slots concurrently still costs operator attention at the
start: the operator must go to each slot and kick work off. An agent in one
slot (say ariadne:1) that has discussed follow-up work with the operator
cannot hand it to an idle live slot (say pair:2) so that work is already
under way when the operator arrives. Slots are bounded, inspectable
concurrency (unlike subagents, the operator has console access), but only if
dispatch is cheap.

## Spec

- **Live, not queued on disk.** Dispatch targets a live, idle Couch slot like
  asking an online coworker. No free slot is the status quo, not an error: the
  issue exists and simply isn't picked up yet. No disk queue (that's Argos,
  ariadne#229).
- **Durable subject, ephemeral transport.** A dispatch names an issue, not a
  free-form brief. The sender files it in the target repo (the sender has the
  operator's context): `sdlc issue new` in a free slot of that repo, then
  `sdlc issue move-detail` so it is claimable. Losing the Couch message loses
  nothing durable.
- **Only concurrent work.** Dispatch only work not blocked by the sender's
  current task (ariadne#272: never start on an unlanded base).
- **Couch routes, pair sequences.** Couch holds a per-slot in-memory queue of
  machine messages; pair interleaves them with human input and injects only
  at a safe point. Pair marks injected lines (e.g. `[couch:ariadne:1] …`);
  the marker's authority comes from pair applying it, never from text an
  agent reads, so a typed or tool-output line starting `[couch:` is not a
  peer message. The skill teaches the format and that a peer message is a
  request, not operator authority.
- **Safe insertion point is the hard part.** Pair's return/alt-return work
  already detects some agent-pane states (e.g. a selection menu accepting only
  certain input), but it cannot see everything: an image pasted from the
  draft leaves an `[Image 1]` placeholder in the agent's input box until sent,
  and injecting then would intermingle the messages. Candidate rule: inject
  only when the agent's input box is visible and empty; fall back to a
  recent-user-activity heuristic where that can't be detected.
- **Discovery:** an environment marker plus the Couch skill tell agents they
  run inside Couch and how to address `repo:N` or `repo` (any free slot).
- Provenance UI is secondary; the operator can ask the slot's agent.

## Done when

- From one slot, an agent dispatches an issue to an idle slot (explicit
  `repo:N` or any free slot of `repo`); the target starts work on it.
- Injection never happens while the target's input box is hidden or
  non-empty; tests cover a menu state and a pending image placeholder.
- A line not injected by pair is never treated as a peer message.
- No free slot leaves the filed issue unclaimed, with a clear message.

## Plan

- [ ] Design the queue, the safe-insertion detector, and the message marker.

## Log

### 2026-09-29

Filed from ariadne#272's brainstorm. Companion to pair#352 (`couch --notify`).
