---
id: 000697
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f1f446f6cccc8aaef9d90695bd2a5071065024a9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T06:54:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcmd.TestHubStartupLaterFailureLeavesFirstUsable]
flow: {kind: quick, provenance: inferred, spec: "6e340b9c", done: "a0bbe98b"}
---

# flake: TestHubStartupLaterFailureLeavesFirstUsable

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestHubStartupLaterFailureLeavesFirstUsable: failed in attempt 4fed2c456da3773658b2bd4328454d6a, passed on the same head in attempt 5bafd7c76410f108c1e6ee72e3e78952

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

Cause (2026-10-19, pair:2). Failed attempt 4fed2c45: `hubs_startup_test.go:202: second launch did not report its injected refusal: <nil>`. The test arms `FakeRunner.FailNextStart`, which fails whichever child starts next. The rig deliberately runs two independent starters: the explicit hub queue (the second hub, `bench`) and the warm-only background pass (ordinary slot `ops:1`). Their order is not defined in production. When the warm worker's start comes first, it takes the injected failure, the second hub starts cleanly, and `start-hub` reports nil. Production ordering is correct here: hubs and warm workers are independent by design. The defect is the fixture's injection, which targets an order rather than the start it means to fail.

- [ ] Deterministic red: a variant that holds `start-hub` at the dispatcher until the warm worker child has started (the barrier pattern TestHubStartupDetachedExtraLaunchesWithoutDuplicateWarmAttempt already uses) reproduces the failure every run.
- [ ] Fix: `FakeRunner.FailNextStartIn(dir, err)` fails the next start in that directory only, so the injected refusal is the second hub's whatever the order. The test arms it for `paths[1]`. Its assertions are unchanged: start-hub reports the refusal, the visible retry line, the first hub usable, and the warm worker started (two children).
- [ ] The held-order variant stays as a regression for both orders. Verify with `-count=50` under CPU load and the full suite.

## Log

### 2026-10-19
