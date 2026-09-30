---
id: 000353
status: working
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-30
estimate_hours:
card_mirror: '8aad5da495291a2921bcf06e9c1c3a21cf40f3ae' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T13:47:42-07:00
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

### 2026-09-30 — operator design direction

Claimed and entered planning in pair:2. The repository is the durable source
of work, including future worksheets for multiple agents; Couch owns only
ephemeral scheduling and communication between live slots. Messages may address
an exact slot (`pair:1`) or a family (`pair`). The operator wants a Couch skill,
slot-owned delivery sequencing, immediate admission feedback, a bounded reply
convention, and clearly identified peer input. Human acceptance remains part of
completion: a quiet agent awaiting acceptance is not thereby free for more work.
Automated acceptance may replace selected human checks later without changing
this separation. Keep the first iteration small and revisable.

Research compared Gas Town's assignment/nudge/mail separation, Claude Code's
cross-session messaging, and orchestrator/worker systems. Relevant references:
[Gas Town messaging](https://github.com/gastownhall/gastown/blob/main/docs/design/mail-protocol.md),
[Gas Town input delivery](https://github.com/gastownhall/gastown/blob/main/internal/cmd/nudge.go),
[Claude cross-session messaging](https://code.claude.com/docs/en/cross-session-messaging),
and [AutoGen termination](https://microsoft.github.io/autogen/stable/user-guide/agentchat-user-guide/tutorial/termination.html).
These inform the design; they do not add a durable runtime queue or an
autonomous staffing/merge system to this issue.

The initial ticket read used `sdlc issue show`, which intentionally prints only
frontmatter and section headings. The full details were subsequently read from
the file; design work must use the file body.
