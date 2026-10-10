---
id: 000429
status: working
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '16a2e0197deca504c69510c5edafce1ea8520cbb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T13:37:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "aa538020", done: "07091856"}
---

# couch --peek fast path: tail first via the running Couch, transcripts only on request

## Problem

pair#425 put each slot's tail in wrapper memory so a TL can read "the last N lines of all slots" instantly. Live check (10-10, ops learnings kink 48): the tail reads `source live` with marks, but a single-slot `couch --peek` takes 3.1–3.4s, two slots 1.9s, four 2.4s, against #425's Done-when of well under a second. For comparison, `couch --message-status` (a socket round trip to the running Couch) takes 0.01s, and `couch --list` 0.9s.

Where the time goes (code read):
- `couchcore/peek.go:143` `PeekThread` always runs `SwitchContext.Resolve` (pair prompt log, native transcripts, session binding) **before** the tail read, even when the live tail succeeds. That's the "sent prompts / transcript / native session" footer.
- Each peek is a cold typed operation (`couchcmd/run.go` `runTypedOperationWithConsole`: prepare runtime, namespace, store), about 0.9s before any work.
- The single-reference path also resolves the workspace reference and repository scope, so one slot is slower than two.

## Spec

1. **Tail first and only by default.** Read the live tail; resolve transcripts only with `--transcripts`, or when the live tail fails and the recording fallback needs the agent.
2. **A fast path through the running Couch:** the CLI asks the live Couch over its socket for the slots' tails (as `--message-status` does), with no cold runtime build. Fall back to today's path when no Couch is running.
3. Single- and multi-slot peeks take the same path.

## Done when

- `couch --peek pair:1:2:3,ariadne:1:2:3 --lines 10` returns live tails for 6 slots in under 200ms (measured on the live workbench, binary not shell function), and a single slot in under 100ms.
- `--transcripts` restores today's footer; a failed live tail still falls back to the recording and says why.
- A test asserts the default peek does no transcript resolution.

## Plan

Durable plan: `workshop/plans/000429-couch-peek-fast-path-tail-first-via-the-running-couch-transcripts-only-on-request-plan.md`.

- [ ] Broker `tail` by slot reference (`Target`, resolved like `--send-to`), answering the thread's tag and agent.
- [ ] `PeekThread` transcript-free by default; `--transcripts` restores the footer; record agent for the recording fallback.
- [ ] CLI fast path: every `repo:N` ref read concurrently over the broker socket, no Couch build; anything else runs the typed path.
- [ ] Docs (atlas/couch.md, couch skill, README).

## Log

### 2026-10-10

- Design: `Binding.Slot` already names each connected wrapper's `repo:N`, so the broker maps a slot ref to its wrapper with no store read; the CLI fast path is one socket round trip per slot, like `--message-status`.
