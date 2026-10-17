---
id: 000637
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '1aa684524c53bdc5de2e3d0d2301355eda9fc5c2' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

- The test never blocks on a possibly nil channel. It asserts that the click was accepted (the channel is non-nil),
  and waits on it with a deadline.
- Find the state race that made the click a no-op under load, and fix it at its source (in the test's wait, or in
  the console if a real click there could also be dropped).

## Done when

- `TestDriveFromCouch` fails fast with a named message instead of hanging when the click is a no-op.
- The test passes 20 times in a row under `-count=20` while CPU load is applied, e.g. a parallel suite.

## Plan

- [ ]

## Log

### 2026-10-16
