---
id: 000434
status: working
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '1d3cf9c20860c1eed29948274b335863349442f7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T23:14:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "4b07914a", done: "dd4e623d"}
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
- [ ] A refresh that leaves the checkout dirty (compile's `.gitattributes` block, pre-#436) stops before the reload with `refresh-dirty`, naming the paths (test).

## Plan

Durable plan: `workshop/plans/000434-couch-prepare-repo-n-one-command-brings-an-idle-slot-up-for-new-work-resting-branch-weave-refresh-context-reload-plan.md`.

- [ ] Admission rule (`not-resting`) and the admission-only `prepare` slot operation
- [ ] Receipt carries the slot path
- [ ] `RunPrepare` sequence with a fake weave over real git
- [ ] `couch --prepare` CLI over the two remote exchanges
- [ ] atlas, couch skill (TL flow: prepare, then send), CHANGELOG

## Log

### 2026-10-10
- Design: weave refresh runs in the CLI process, not on Couch's serial console queue (it would block every console operation for minutes). The remote half is an admission-only `prepare` slot operation plus the existing `reload-context`.

## Revisions

- 2026-10-10 — Spec: the surface is `couch --prepare repo:N --confirm [--force-unknown] [--json]`. It ends the agent's conversation, as reload-context does, so it takes the same `--confirm`. Added the `refresh-dirty` refusal for weave refresh dirtying its own checkout (ariadne#320's `.gitattributes` block until pair#436 lands); Couch never commits that file.
