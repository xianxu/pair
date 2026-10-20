---
id: 000718
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '677fe069d5e99349c0beb6fe40c155d873279284' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T21:19:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# TestCouchLaunchPromptTransportHelper: TempDir cleanup finds a late write (directory not empty) under load

## Problem

`TestContinuationWriterPublishesExactCheckpointAcrossWorktrees/initial` (cmd/internal/couchcmd/continuation_acceptance_test.go:137) failed once in a full local suite on pair#716 head 657e99e1e. The production-transport subprocess it runs, launcher `TestCouchLaunchPromptTransportHelper`, failed only in `t.TempDir` cleanup:

```
testing.go:1617: TempDir RemoveAll cleanup: unlinkat /Volumes/TestScratch/tmp/sdlc-run-suite-20506-1989716327/TestCouchLaunchPromptTransportHelper1938472632/001: directory not empty
--- FAIL: TestCouchLaunchPromptTransportHelper (0.27s)
continuation_acceptance_test.go:146: continuation publication failure host: host os=darwin cpus=12 loadavg={ 27.10 56.40 78.06 }
```

The helper's assertions passed. Something still wrote into the temp directory (used as DataDir and Home for `run(t, opts, rt)`) after the test body returned, so RemoveAll found a new entry. Likely suspect: work started by the launch `run` that outlives it (a goroutine, sidecar or diagnostic writer) and that the fake runtime does not join. The machine was heavily loaded (load average 78), which widens the window. The pair#716 diff does not touch launcher or continuation code.

Evidence: suite log kept at /Users/xianxu/workspace/pair/.git/pair-parklive-evidence/suite-716-continuation-tempdir.log (copy of .git/sdlc/test-stats/failed/59087d99b62331e30d21a91c60bf6211.log).

## Spec

Find what writes into the helper's DataDir/Home after `run` returns, and make the launch join it, or the test wait on an ordering barrier, before cleanup. Get a deterministic red (a forced interleaving or barrier), not a timeout.

## Done when

- A test that deterministically reproduces the late write fails before the fix and passes after.
- No sleep or timeout is added.

## Plan

- [ ]

## Log

### 2026-10-19

- Already fixed by pair#715 (ad54eecde, landed 20:02): the released session-watch child wrote into the test temp dir after the wrapper returned. pair:3 measured it as 20 of 20 failing without the join and 0 of 20 with it. The failing head 657e99e1 predates #715. Evidence on main 73cc0403a, which contains #715: `go test ./cmd/internal/couchcmd -run '^TestContinuationWriterPublishesExactCheckpointAcrossWorktrees$' -count=20` passed 20 of 20, with the full #716 suite running alongside (load average 16 to 33). Log: /Users/xianxu/workspace/pair/.git/pair-parklive-evidence/718-main-count20.log. No second writer. Closed wontfix as fixed by #715 (the hub, ops:0, directed this check).
