---
id: 000749
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '8163903d42f3543e343a99f1c41b3506f0c0d79c' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestCopyOnSelectShellFixture

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/clipcmd.TestCopyOnSelectShellFixture: First seen 2026-10-20 about18:27 PDT during pair#746. Same exact head a4f0b6ae0c9dd679c3829679485b43239f2de7a4: full suite attempt7562be4ed11412edecae702be7f460e4 failed only TestCopyOnSelectShellFixture at cli_admission_test.go:108 after3.54s: detached copy orchestrator did not finish. Complete failure log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7562be4ed11412edecae702be7f460e4.log hash6a8eb50de62d, run12m11s; every other test passed. Unchanged isolated detached checkout of that exact commit, make runtimebundle-generate then go test ./cmd/internal/clipcmd -run ^TestCopyOnSelectShellFixture$ -count=1 -v, passed3.82s/package4.022s exit0. Retained actual targeted output /Users/xianxu/workspace/pair/.git/manual-verification/pair746-clip-a4f0b6ae.log. Fixture tests/copy-on-select-test.sh:88 waits3s on select for detached completion. Load12.67/13.20/16.84 at observation; scheduling/deadline sensitivity is a hypothesis, not measured cause. No production, test or timeout edits. Independent unassigned follow-up; do not fold into746.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-20
