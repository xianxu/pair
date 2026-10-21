---
id: 000757
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '759cd5f5efe753f8ced70aa4eebf89058e9fe0c2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T02:15:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# mock-to-real app template: per-screen real-or-mock dev router, refusing production build, fixtures contract tests (product-shaping)

## Problem

product-shaping (ops) grows the real app out of its mock screen by screen. In development, unbuilt screens should
still be clickable as mocks; nothing half-mocked may ship.

## Spec

A reusable template (Go) for mock-backed apps:
- a development router that serves each screen from real code if it exists, else from the mock, with a visible MOCK
  badge, and a progress count ("12 of 18 screens real") from the state list;
- the production build refuses while any listed screen is mock-backed (or, if a partial release is chosen
  explicitly, those screens render a "not yet" page, never the mock);
- contract tests: the real backend's responses are checked against the shape of `fixtures.json`.

## Done when

- A sample app with two real screens and one mock screen runs in development with the badge and count; the
  production build refuses until the third is real; the contract test catches a shape drift.

## Plan

- [ ] Deliver the approved template, sample, build refusal and fixture contracts described in workshop/plans/000757-mock-to-real-plan.md.
- [ ] Verify, close and land the single atomic issue.

## Log

### 2026-10-20

### 2026-10-21 — design proposal

Claimed and started after #755 landed as PR507. Read the product-shaping design and surveyed integration constraints. The host is Go-internal and owns its watcher only in the standalone serve lifecycle; direct template import cannot reuse it. Proposed a self-contained nested Go module with a thin development HTTP fallback to the existing host, one screen list and guarded production build; root integration test explicitly runs the nested module. Proposed strict build refusal without partial release, prototype-based fixture contracts, and bounded/owned effects. Detailed alternatives, state table and costs are in the durable plan. Awaiting hub/operator design approval; no implementation started and no estimate derived.
