---
id: 000778
status: done
created: 2026-10-21
updated: 2026-10-22
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand]
started: 2026-10-21T18:23:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 4.64
tracker:
    version: 1
    completion:
        token: close-4cedc78d67c5
        repository: github.com/xianxu/pair
        reviewed_head: 5dc34bb27415a64474ef501fc08bf27831507d86
        evidence_commit: d55ae7709826499d01e696e6e37821e011a87941
        landed_commit: 8296e90e5628d3fa1cf547a0bd375291c87fce3b
---

# flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub: Exact source3279e1e9ec6ccbddba0b50193f5c8f72c11f4b22: train238 suite lease-1293856745 failed cmd/couch TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub after15s with live/contested terminal 📁1-1; log .git/sdlc/test-stats/inflight/de3d7e12cd48155a874dbd8db94ac3f6/run.log. On the unchanged source, env -u PAIR_TEST_BINARY go test ./cmd/couch -run ^TestBareCouchInstalledCommand$ -count=1 -timeout=90s -v passed11.305s (PTY2.12s). This fixture has private socket roots but fixed terminal name; SessionServers filters only session basename, a lead not yet a forced reproduction. Production scribe is unchanged. Track this intermittent fixture refusal separately from #768.
