---
id: 000637
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: '66623a42c3b775407114f088767f886cf964b277' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T03:22:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ca144426", done: "bd7ad23b"}
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

- `TestDriveFromCouch` rejects a nil click completion immediately with the named action; accepted clicks retain the existing five-second wait bound.
- Both gated regression orderings reject premature readiness and pass after completion, and the original test passes 20 times under `-race -count=20` while another suite applies CPU load.

## Plan

- [x] Force the premature-disabled observation with a gated fake tab release; observe the fixture regression red.
- [x] Wait for Off plus the disable completion notice, and reject nil completion channels before bounded awaits; keep the real mouse path and existing assertions.
- [ ] Run targeted/race checks and 20 repetitions under concurrent suite load, then recorded suite, close review and landing.

## Log

### 2026-10-16

### 2026-10-17 — root cause and quick plan

The old predicate treats Disabling/Running as disabled (`Project.Enabled == false`). Cleanup resets control and notifies Chrome before releasing held input and publishing Off. Thus all three preceding waits can pass while the capability click correctly means wait and returns nil. A second window exists after Off: Cast publishes capabilities before clearing its running slot. The console's `drive capability off.` notice is set only after the result has arrived (the slot is already released), on its terminal command queue. Wait for that completion evidence and Off before the new click. No product change is required: refusing a click during cleanup is deliberate and already displayed as turning off.

Use the existing stateful FakeDriven browser, with a test-local gated WithTab release, to hold the problematic ordering. Also hold the terminal command queue after authority reaches Off to prove phase alone is insufficient. Reuse awaitDrive's five-second bound and add an immediate non-nil assertion (ARCH-DRY); no new sleeps or longer deadlines. This changes fixture synchronization only (ARCH-PURPOSE, ARCH-ORDER); no new production model, external trust boundary, durable artifact or resource budget. The gate is released on all test exits and fixture teardown joins the console and cast (ARCH-FUNERAL). Plan posted to the hub, proceeding under quick-flow authorization.

### 2026-10-17 — deterministic red/green

Introduced a shared fixture readiness predicate with the original `!Project.Enabled` body, then ran `go test ./cmd/internal/couchtty -run '^TestDriveDisableCompletionOrdering$' -count=1 -v`. Both controlled orderings failed in 0.12 s: `fixture accepted disabled projection before cleanup completed` and `fixture accepted Off before terminal operation completion`. The first holds the fake browser's input-release return after control reset; it proves all original setup observations (no driver, Chrome told nobody, disabled projection) and the intentional nil result while Action=wait. The second holds the console command queue while authority publishes Off. Both gates release on test exit; the terminal barrier is explicitly joined.

Changed only the fixture readiness body and replaced the unbounded receive with awaitDrive, which now immediately rejects nil with the named action. The disable/enable operation bounds stay five seconds. Targeted drive/capability family passed in 0.923 s, including both regression orderings and the original TestDriveFromCouch. Existing terminal-presenter teardown diagnostics occurred in passing sibling tests; no test failure. Race/count=20 qualification is running under concurrent ariadne sharded suite CPU load (sdlc.test workers observed; host load 70.33/46.76/39.52 at launch). Effective GOFLAGS=-p=3; no flag/deadline changes.


### 2026-10-17 — load qualification

go test -race ./cmd/internal/couchtty -run '^(TestDriveDisableCompletionOrdering|TestDriveFromCouch|TestWebCapability.*|TestDriveChip.*)$' -count=20 passed in 119.773 s, no race reports. This includes twenty executions of the original integration test and each deterministic interleaving, plus capability recovery, stale completion, and chip-click siblings. During execution the host load reached 71.28/52.98/42.64 and a concurrent ariadne sharded suite's sdlc.test workers were observed consuming CPU. No artificial load jobs, deadline extensions or retries were used. Read the retained original hang log too: it confirms the nil-channel receive in TestDriveFromCouch. PR #396 is open; recorded suite on b7f361ec1a8a097d500baa1241e9b88bf35a3b41 is queued behind the existing machine test line.

### 2026-10-17 — recorded suite failure classified; one retry approved

First recorded suite on b7f361ec1a8a097d500baa1241e9b88bf35a3b41 acquired its slot after 1466 s, then failed after 27 s before Go tests: nvim/changelog_test.lua, reload content and newest-entry cursor assertions. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7e7efa0d33e0f88288f8746ce6e1d8d3.log, reported SHA256 prefix 2ae07bbeb087 (full hash in #642).

Classified as pre-existing fixture collision and filed unassigned #642 with original-source proof. Original baseline ce783cd4a474d939c174bb079bc687efc2cc457a (git archive of every loaded nvim module and test), shared private TMPDIR, two unmodified test invocations: A paused after real writefile, B completes and removes the same fixed reload filename, then A fails exactly both recorded assertions. Forty unsynchronized concurrent controls passed; an isolated current test passed. The original failed suite's particular remover remains unknown. Evidence /tmp/pair637-ed1erwu8 includes wrapper, exact original source, all logs and classification.json. #637 changes no nvim code.

The failed recorded head was followed while queued by qualification-log-only commit 4b40b8953aa2 (no source difference). Hub explicitly approved the one unchanged retry recorded on current tip 4b40b895 rather than manufacturing another ref at b7f361ec; receipt a7fa88db-97d5-420e-83d7-b8e13b0005ea verified submitted and exact. Retry command unchanged: sdlc local-check run --name suite. If the same collision recurs, stop and report; do not retry again. All server CI checks currently green. No separate make -k started yet: the approved full retry can cover the remaining targets if the prerequisite passes; additional failures will be named and classified under the worker rule.
