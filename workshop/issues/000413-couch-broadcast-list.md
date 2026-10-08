---
id: 000413
status: codecomplete
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: 'efd62d28d2615127361329bcc8f52e41173447a7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-08T09:47:46-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.67
---

# couch --broadcast-list: print the current broadcast session and its viewer count

## Problem

A couch broadcast (#395, `atlas/broadcast.md`) streams the screen view-only to
browser viewers through a token link. Ctrl+Alt+b starts and stops it, and the
status row shows LIVE. From a shell, though, nothing says whether a broadcast is
running or how many people are watching. To find out, the
operator has to look at the couch screen itself.

## Spec

`couch --broadcast-list` asks the running couch and prints its current
broadcast session, if any, and the number of connected viewers.

- **Routing.** A live-owner query answered by the running console over the
  broker socket. Unlike `--actors`, it needs no slot identity: the CLI finds
  the socket through the store directory, so it works from any shell. With no couch running it
  says so and exits non-zero. With no broadcast running it says
  "no broadcast" and exits zero.
- **Output.** One line per session (today there is at most one):
  - state (starting, live, stopping);
  - since when;
  - mode (tunnel or local-only);
  - the viewer count.

  `--json` gives the same fields for scripts.
- **The link is never printed** (operator decision, 2026-10-08: "don't show
  link for privacy reasons"). The token in the path is the broadcast's only
  access control, so the command prints neither the link nor the token, in
  text or `--json`, and has no flag to reveal it. The link stays where it is
  shown today: in couch, to the operator who started the broadcast.
- **Viewer count.** `broadcast.Hub` holds its subscribers inside its run
  goroutine and exposes no count. Add a read that answers through the hub's own
  loop (`h.do`), or a counter the hub updates as viewers join and leave, so the
  answer comes from memory with no per-viewer probe on the request path (see
  the "deterministic commands answer from memory" rule). Count viewers
  currently subscribed, including ones marked for resync.
- **No new trust boundary.** The socket is already owner-only; the query is
  read-only and starts, stops or changes nothing.

## Done when

- `couch --broadcast-list` prints the session state, start time, mode and live
  viewer count from the running couch, and "no broadcast" when there is none.
  `--json` prints the same fields.
- The viewer count is read from the hub's own state. A test joins and leaves
  viewers and checks the count, including a viewer that is resyncing.
- The output never contains the link or its token, in text or `--json`, and
  there is no flag to reveal it. A test checks both formats against a live
  session's token.
- With no couch running, the command refuses with a clear message and a
  non-zero exit.
- `atlas/broadcast.md` and the README (command list and broadcast section)
  document the command and its never-the-link rule.

## Plan

Durable plan: `workshop/plans/000413-couch-broadcast-list-plan.md`.

- [x] Hub viewer count and session status.
- [x] Console snapshot.
- [x] `broadcast-status` broker op and service route (no caller identity).
- [x] `--broadcast-list [--json]` CLI from any shell, with output and tests.
- [x] Atlas, verification, close.

## Log

### 2026-10-08
- 2026-10-08: closed — couch --broadcast-list [--json] plus close-review fixes: README command list and broadcast section document it (BR-1); older-couch identity refusal maps to the restart hint (tested); hub test asserts its resyncing precondition; local-only mode test. All broadcast/couchmessage/couchcmd/couchtty tests green unsandboxed; link/token never in text or JSON (tested against a real session); socket owner-only (0700 uid dir, 0600 socket). Full-suite residue pre-existing/environmental; side-quest fixed main's stacked-godoc lint.; review verdict: SHIP
- 2026-10-08: flow upgraded quick → full — 203 added lines in code files (limit 100); an earlier round of this close already ran the full review

Filed at the operator's request: "make a task to have a couch command, `couch
--broadcast-list` to print out current session, and how many viewers are
there." The details stay local until the operator asks to publish them.

### 2026-10-08: close review (FIX-THEN-SHIP) addressed

- **BR-1:** the README command list and broadcast section document
  `--broadcast-list`. Done-when now names the README.
- **Minors:**
  - The Spec's states read starting, live and stopping, matching the code.
  - An older couch answers with its identity refusal, so the CLI now
    recognises that and gives the restart hint (tested).
  - The hub test asserts its slow viewer really is resyncing.
  - A local-only mode test was added.
- **Context finding, no change:** `Hub.Viewers` is bounded because the hub
  loop never blocks, so the handler needs no extra context wiring.
