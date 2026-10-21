---
id: 000749
status: done
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/clipcmd.TestCopyOnSelectShellFixture]
started: 2026-10-20T18:48:49-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 0.23
tracker:
    version: 1
    completion:
        token: close-006e19a0a20a
        repository: github.com/xianxu/pair
        reviewed_head: 0f7a69d46759f31e0b3944ae30149f05dbb932ec
        evidence_commit: be4923ced56cd324d3f0cc4dd32a6a7543a206cf
        landed_commit: b1735a52b255f28e41b888007dc744c8a3c737a7
---

# flake: TestCopyOnSelectShellFixture

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/clipcmd.TestCopyOnSelectShellFixture: First seen 2026-10-20 about18:27 PDT during pair#746. Same exact head a4f0b6ae0c9dd679c3829679485b43239f2de7a4: full suite attempt7562be4ed11412edecae702be7f460e4 failed only TestCopyOnSelectShellFixture at cli_admission_test.go:108 after3.54s: detached copy orchestrator did not finish. Complete failure log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7562be4ed11412edecae702be7f460e4.log hash6a8eb50de62d, run12m11s; every other test passed. Unchanged isolated detached checkout of that exact commit, make runtimebundle-generate then go test ./cmd/internal/clipcmd -run ^TestCopyOnSelectShellFixture$ -count=1 -v, passed3.82s/package4.022s exit0. Retained actual targeted output /Users/xianxu/workspace/pair/.git/manual-verification/pair746-clip-a4f0b6ae.log. Fixture tests/copy-on-select-test.sh:88 waits3s on select for detached completion. Load12.67/13.20/16.84 at observation; scheduling/deadline sensitivity is a hypothesis, not measured cause. No production, test or timeout edits. Independent unassigned follow-up; do not fold into746.
