---
id: 000603
status: working
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T15:35:32-07:00
tracker:
    version: 1
    release:
        by:
            operator: Xian Xu
            machine: 4716879978a7b90f6b583da1716fd0e9
            machine_name: Xian’s MacBook Pro
            workspace: pair:3
            worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
            repository: github.com/xianxu/pair
        branch: 000603-native-shell-actor-timeout
        head: e0fdab4469efdd00a91a16aaa7b5b951ecf5f74d
---

# Native wrapped Zellij conformance times out finding shell actor

## Problem

GitHub macOS conformance intermittently stalls waiting for the real shell actor in TestNativeConsoleWrapperZellij/wrapped-zellij. This blocks unrelated PRs after their local full suite and review pass.

First observed here: 2026-10-16 06:27:17 UTC (2026-10-15 23:27:17 PDT), run 38860695250, attempt 1, head 550c434561059eeda72ed209041b5468916081f1, while landing #571. The subtest failed after 284.48 seconds at terminal_native_test.go:227: timed out waiting for real shell actor.

Run: https://github.com/xianxu/pair/actions/runs/38860695250/job/116842060496

Local diagnostic log: /tmp/pair571-ci-conformance-38860695250.log (ephemeral; the issue does not depend on its retention).
