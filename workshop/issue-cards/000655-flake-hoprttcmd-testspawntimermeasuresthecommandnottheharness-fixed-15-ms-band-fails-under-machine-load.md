---
id: 000655
status: done
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T17:35:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 0.36
tracker:
    version: 1
    completion:
        token: close-d9add8175590
        repository: github.com/xianxu/pair
        reviewed_head: 2481722ff4e7b7ddd282c0d54b375289679f65f5
        evidence_commit: fb3a506055d7c662c716d8dc1e00367fa2d40e64
        landed_commit: f16ecfe6c009be187ed7a1cdc03b9e90328e367f
---

# Flake: hoprttcmd TestSpawnTimerMeasuresTheCommandNotTheHarness fixed 15 ms band fails under machine load

## Problem

Flake in the recorded pre-merge suite, 2026-10-17: `cmd/internal/hoprttcmd` `TestSpawnTimerMeasuresTheCommandNotTheHarness` failed on #620's integrated head 7c96d30f (PR #407):

    hoprtt_test.go:69: fork+exec median 16.935 ms is outside the plausible band; the harness is measuring itself, not the command

- The test gives a fork+exec of /usr/bin/true a fixed 15 ms ceiling. Its comment budgets for the suite's own spawn load (about 4.5 ms), not for whole-machine load. At the time the machine's load average was about 10 now, 11.5 over 5 min and 18 over 15 min: two full suites plus fleet slots running.
- #620 doesn't touch hoprttcmd, and the main baseline passed, so this is the band failing under machine load, not a code change.
- Log retained: /Volumes/AgentScratch/tmp/pair-618-share/suite-7c96d30f-hoprtt-flake.log (sha256 prefix f1d958641097; original at .git/sdlc/test-stats/failed/e9a569fffa42f6450a9b09fb08569c4b.log).
