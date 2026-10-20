---
id: 000744
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ddc133aa1d7d8437a3650f1a07ef0167983a38d1' # card fields mirrored from issue-cards; edit via sdlc
---

# testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup)

## Problem

Live tests each build their own disposable Couch: pair#722's three-slot drain matrix, pair#726's held-cold resumes,
pair#740's global park, pair#741's hub-resume conformance, and pair#743's conformance in a VM. Each reinvents
isolation (state folder, socket namespace, HOME, queue root) and cleanup, and a mistake in any of them can touch the
live fleet. Operator, 10-20 (ops side doc `2026-10-20-19-conformance-structure.md`, decision 1): one shared harness.

## Spec

One pair test helper package for a disposable Couch fleet:
- private `COUCH_ISOLATED_ROOT`, store, identity, Pair data, HOME, XDG, Zellij socket/config/runtime and sdlc queue
  roots, under TestScratch when present (TMPDIR otherwise); refuses empty, live or inherited paths before starting;
- a real supervisor plus N throwaway worker slots in real disposable checkouts, launched with genuine caller
  identity (no synthesized bindings);
- agent fixtures for **each agent kind** (Claude and Codex, Qoder when available) and a deterministic fake agent;
- exact cleanup: joins every child and removes only the processes, sessions and sockets it created;
- a guard test proving it can never see the live Couch's socket, store or HOME.
Existing live tests migrate to it as they're next touched; new ones must use it.

## Done when

- The helper exists with the guard test, and pair#741 (or the first live test to land next) uses it.
- `atlas/` documents it as the one way to write a live test.

## Plan

- [ ]

## Log

### 2026-10-20
