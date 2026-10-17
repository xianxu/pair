---
id: 000637
status: working
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T03:22:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# flake: couchtty TestDriveFromCouch hangs 20 min receiving from a nil capability-click channel

## Problem

`couchtty TestDriveFromCouch` hung for the whole `-timeout 20m` and failed the package. This happened during
`sdlc merge`'s pre-merge suite for pair#618 PR #387 on 2026-10-16, 21:03–21:23, on the PR branch head 9917e65f with
origin/main integrated, at a load average of about 40–50 with other suites running. The failure is intermittent: the
same test passed in pair#618's recorded suite on fb8333ae (sdlc-local/suite, 10m34s), and nothing in #618 touches
couchtty or the webcast capability code.

Evidence (log retained at `/Volumes/AgentScratch/tmp/pair-618-share/suite-ba41cd27-couchtty-hang.log`, sha256
41d11802d86d9ace…; the original is under `.git/sdlc/test-stats/failed/37bf2f3695ab73b2bc16b7317aef066b.log` in
`~/workspace/pair`):

```
panic: test timed out after 20m0s
	running tests:
		TestDriveFromCouch (19m51s)
goroutine 5667 [chan receive (nil chan), 19 minutes]:
github.com/xianxu/pair/cmd/internal/couchtty.TestDriveFromCouch(...)
	cmd/internal/couchtty/console_drive_test.go:314
```

Line 314 is `<-c.con.onWebCapabilityClick("drive", false)`. `onWebCapabilityClick` (`cast_control.go`) returns a nil
channel on several paths: cast not running, capability unsupported, an intent of "copy", or an unrecognized intent
(notice only). A receive from a nil channel blocks forever. So when the preceding `waitFor("drive disabled")` passes
while the capability's projected state still yields one of those intents (a state that is still settling), the test
hangs instead of failing.
