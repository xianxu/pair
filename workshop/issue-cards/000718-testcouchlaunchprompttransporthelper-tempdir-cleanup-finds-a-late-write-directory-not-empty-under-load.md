---
id: 000718
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
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
