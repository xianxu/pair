---
id: 000434
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'd4208e4aef05abba085b30d9470d4f9220fd12ae' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --prepare repo:N: one command brings an idle slot up for new work (resting branch, weave refresh, context reload)

## Problem

Each slot builds `sdlc` and the other layer tools from its own substrate checkout (`worktree/<repo>-slotN/ariadne`),
which only moves when someone runs `weave refresh` there. On 10-10, every pair, parley.nvim, tools, parli and brain
slot ran an sdlc from 11:41 all evening, missing that day's fixes, including ariadne#183's `--fixed-then-ship`, so
pair#324 paid a full re-close; ariadne slots missed ariadne#322's landing-gate fix until they happened to merge main.
The TL has no single step that brings a slot up for new work, so freshness depends on luck.

## Spec

`couch --prepare repo:N [--json]`, run by the TL before dispatching an issue to an idle slot:
1. Refuse unless the slot is idle (no turn in flight) and its checkout is clean on its resting branch.
2. Fast-forward the resting branch to `origin/main`.
3. `weave refresh` (substrates, layer tools, compiled skills).
4. Reload the agent's context (as `--reload-context`) so the new skills and binaries are what it reads.
5. Report one line per step, and the sdlc/weave commits now live in the slot.

Steps 2–3 are weave's; Couch orchestrates and owns the idle check and the reload. A slot that refuses names the
reason and the fix. The TL's dispatch flow becomes prepare, then send.

## Done when

- [ ] `couch --prepare` on an idle, clean, stale slot leaves it at origin/main with refreshed substrates and a reloaded agent (test with a fake weave).
- [ ] It refuses a busy or dirty slot without changing anything (test).
- [ ] The report names the live sdlc commit (test).

## Plan

- [ ]

## Log

### 2026-10-10
