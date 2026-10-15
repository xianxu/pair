---
id: 000565
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'cedfcc76282df8566ef7983c39e6754970f59fe0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T01:59:23-07:00
---

# Couch: a worker that asks the hub mid-turn can't receive the answer until its turn ends; add an in-turn ask/inbox

## Problem

Couch delivers a message to a worker slot only when the worker's current turn ends ("position N" queue). Workers
(Codex especially) often run one long turn (ariadne:1 on 10-14/15: one turn of ~5 h) and, inside it, ask the hub a
question and then wait for the answer, polling or idling *within the same turn*. The hub's answer queues behind the
turn and can't arrive until the worker gives up and ends the turn, so the worker re-asks ("please approve", three
times for ariadne#376's staging question; similar repeats from pair:3, pair:5, pair:7, parley.nvim:2). Combined with
"outcome uncertain" sends, the hub can't tell whether an answer was delivered, queued, or lost.

## Spec

- A worker that asks the hub and waits must be able to receive the answer within its turn: e.g. a blocking
  `couch --ask hub --message … [--timeout]` that returns the hub's reply (correlated by message ID), or a
  `couch --inbox` the worker polls inside its turn that returns queued messages addressed to it.
- The hub's reply to a question carries the question's ID (`--reply-to ID`), so the worker can match it.
- `--message-status` distinguishes "queued behind the recipient's turn" from "lost".
- The tl-worker rules say: ask with the blocking form; never wait on a queued message inside a turn.

## Done when

- A worker mid-turn asks the hub and receives the hub's reply in the same turn (an integration test with a fake hub
  and a long-running worker turn).
- `--message-status` reports "queued (recipient mid-turn)" for a queued message; a test covers it.
- tl-worker documents the ask pattern.

## Plan

- [ ]

## Log

### 2026-10-15
