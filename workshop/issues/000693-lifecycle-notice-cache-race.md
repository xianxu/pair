---
id: 000693
status: codecomplete
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
severity: medium
collaboration-mode: collaboration
card_mirror: '2bad04a108fbd77d2ed4f3a373f529c60c51ff1f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T03:57:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "1e1a958b", done: "86779369"}
actual_hours: 0.50
---

# flake: lifecycle primary-slot notice can race verified workspace publication

## Problem

The #681 integrated suite on 807115355d853aa5ef041dd212fd90f0142d4994 failed TestLifecyclePublisherRegisteredPrimarySlotIsNotOrdinaryContent after1139.55s with "no lifecycle message" (lifecycle_notices_test.go:556). The package reached its20m deadline as the next, unrelated TestMessageAdmissionRejectsMismatchesThenGoesDormant/root began (0s). All other Go packages and prior script/editor checks completed successfully.

## Spec

Reproduce and repair the lifecycle publisher/fixture readiness race without relaxing timeout, ownership or notice-delivery authority. A connected wrapper does not yet guarantee its verified workspace cache has been published. Make the fixture wait on the actual cached-identity evidence it consumes; separately evaluate whether ordinary-target production notices may be dropped in the same interval.

## Done when

- A controlled stateful regression exercises connected publication before workspace-cache publication and identifies whether the missing notice is fixture-only or production behavior.
- The primary-slot lifecycle test observes verified workspace readiness before sending its ordinary-target fact, without sleeps or retries as readiness.
- A bounded diagnostic names missing publisher/cache state if the notice never arrives.
- Required verification and normal review/landing pass.

## Plan

Diagnose the exact lifecycle publisher interleaving, add a red deterministic case, then fix the responsible seam. Do not change #681 admission/receipt code unless new evidence makes that code reachable.

### Diagnosis (2026-10-19, pair:2)

The failing interleaving: `messageService.loop` step publishes `connected` (and liveness) before `execute(EffectConnect)` writes `s.workspaces[b]`. The fixture's `waitConnected(worker)` can return in that window; `p.refresh()` then snapshots a cache without the worker, and `p.observe(ordinary fact)` reaches `observeFactLocked` with no identity. The `!ok && Target.Kind == ordinary` branch returns, dropping the fact for good. The publisher's later refreshes (the loop signals it after every step) do cache the worker, but nothing remains to resolve, so the notice never comes and `lifecycleNextMessage` waits out the package deadline (testwait.After).

That window is not only a fixture gap. In production the same drop is the normal case: the Console emits `started` (or relaunched, resumed, recovered) when the child launches, while the wrapper's message session connects and its workspace is verified seconds later. An ordinary-target fact (a row the slot inventory does not project as a slot, such as a registered primary checkout) therefore usually arrives before its identity is cached, and is lost. Untargeted facts (`Target` empty) already wait in `unresolved` and resolve on the next refresh; ordinary facts are the one class dropped outright. So the root cause is the publisher's drop, not the loop's publish order: reordering `connected` after the workspace write would close only the microsecond test window (and the loop's publish-first order is deliberate, see its comment), while the seconds-long production gap would keep losing notices.

### Fix

- [x] Red first, deterministic, no timing: a publisher regression that observes the ordinary-target fact before the worker's wrapper connects at all (the widest form of the connected-before-cache window), then connects it and expects the notice. Today it fails (the fact is dropped). A second case keeps the existing order: cached first, fact after.
- [x] `observeFactLocked`: an ordinary target without a cached identity waits in `unresolved`, like an untargeted one, instead of returning (ARCH-PURPOSE: the class is "a fact that arrives before its identity", not the one test). `reducePendingLocked` already resolves it on the next refresh, which the message loop triggers after each registry step, so the notice follows the workspace publication without a poll or retry.
- [x] Bound (ARCH-FUNERAL): `unresolved` stays keyed by path, newest observation wins, and is cleared at restart drain (with its diagnostic) and on ownership change. At its 128 cap a new path evicts the entry with the oldest observation rather than being refused, so threads that never verify (non-slot directories) cannot crowd out a resolvable one. Per-path residue is one observation.
- [x] Fixture readiness: the primary-slot test waits for the worker's verified workspace in the message service cache (a `waitWorkspace` rig helper over `s.workspaces`, the evidence `refresh` consumes), not `connected`. No sleeps as readiness.
- [x] Bounded diagnostic: `lifecycleNextMessage` takes the publisher and, on timeout, fails with its state: unresolved paths and observations, cached identity roots, hub roots, ready tokens, attempts, plus the service's cached workspaces. Capped in size.
- [x] Verify: the regression fails before the fix and passes after; `go test ./cmd/internal/couchcmd -run Lifecycle -count=50` under load (a concurrent CPU burner), and the full suite.

ARCH-ORDER: the publisher's states for a fact are dropped, unresolved, or confirmed. This change moves ordinary facts lacking identity from dropped to unresolved. The events that can interleave are fact-before-identity, identity-before-fact, drain, and ownership change; each now has a defined outcome, and the tests cover the first two orders. ARCH-CONSTRAINTS: reduction stays in memory under the publisher lock, bounded at 128 entries; no discovery or subprocess is added. ARCH-SECURE: no new inputs; a cached identity remains the only slot authority (an unresolved fact still cannot claim a slot). #681 admission and receipt code are unchanged.

## Log


- 2026-10-19: closed — sdlc-local/suite green on fdac39f3 (15m21s); round-1 fixes in 877401b2 re-verified with the full couchcmd package (66s). TestLifecyclePublisherOrdinaryFactWaitsForItsWorkspace forces fact-before-workspace and failed before the fix with the state diagnostic (identity cached, nothing unresolved); passes after. TestLifecycleDrainReportsOnlyUntargetedFacts (red when drain counts ordinary facts), TestLifecycleOrdinaryFactExpires, TestLifecycleUnresolvedFactsAreBoundedOldestFirst. -run TestLifecycle -count=50 passed under one yes burner per CPU. No timeout changed; #681 code untouched.; review verdict: SHIP
2026-10-18 — Observed in integrated suite66983be006023c8bc5127ecedfc3ad3c, source807115355d853aa5ef041dd212fd90f0142d4994, exit2, wall1752.639382458s, no outer suite timeout. Full retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/66983be006023c8bc5127ecedfc3ad3c.log, SHA256 c442bf6703ef51ab2aea6e38b61dadbeea3d8d67c2ec7a7bc4ca821112f50c87. Read-only1s native sample retained at /tmp/pair681-integrated-couchcmd-1709.sample; no process was signalled or lock changed.

The #681 branch suite passed in13m6s. On this exact integrated head, TMPDIR=/tmp go test ./cmd/internal/couchcmd -run '^TestLifecyclePublisherRegisteredPrimarySlotIsNotOrdinaryContent$' -count=20 -timeout30s passed0.754s. One observed failure plus passing repetitions does not prove the initiating interleaving.

Source audit: messageService.loop publishes connected before EffectConnect populates s.workspaces. The fixture's waitConnected only observes connected; its p.refresh can therefore miss the worker identity. observeFactLocked drops an ordinary target lacking cached identity rather than retaining it for later refresh. This is a strong pre-existing readiness-race diagnosis; the failed attempt did not retain publisher state, so the initiating cause is not conclusively reconstructed. Lifecycle files are unchanged since #574; this rig has slotOps nil, so #681's authorizePark/new preparePark path is unreachable. No slotOperations, cached-inventory selection or receipt lock wait is in the timeout stack.

testwait.After uses the package deadline minus10s. Thus the notice wait consumed nearly the full remaining package budget; its eventual failure was followed by the next test reaching the global alarm.

Retained primary failure and complete package goroutine dump:

```text
--- FAIL: TestLifecyclePublisherRegisteredPrimarySlotIsNotOrdinaryContent (1139.55s)
    lifecycle_notices_test.go:556: no lifecycle message
panic: test timed out after 20m0s
	running tests:
		TestMessageAdmissionRejectsMismatchesThenGoesDormant (0s)
		TestMessageAdmissionRejectsMismatchesThenGoesDormant/root (0s)

goroutine 3609 [running]:
testing.(*M).startAlarm.func1()
	testing/testing.go:2959 +0x2c4
created by time.goFunc
	time/sleep.go:182 +0x38

goroutine 1 [chan receive]:
testing.(*T).Run(0xd6dbba72248, {0x1056f1ecd?, 0xd6dbbe75738?}, 0x1064ca0b8)
	testing/testing.go:2266 +0x3cc
testing.runTests.func1(0xd6dbba72248)
	testing/testing.go:2742 +0x38
testing.tRunner(0xd6dbba72248, 0xd6dbbe75868)
	testing/testing.go:2193 +0xc4
testing.runTests({0x1056ac24e, 0x16}, {0x1056e21c3, 0x2c}, 0xd6dbba24120, {0x106578df8, 0x27a, 0x27a}, {0xbbe76000?, 0xd6dbbd3fcb0?, ...})
	testing/testing.go:2740 +0x400
testing.(*M).Run(0xd6dbbcfe320)
	testing/testing.go:2600 +0x578
github.com/xianxu/pair/cmd/internal/testpair.Run(0x0?)
	github.com/xianxu/pair/cmd/internal/testpair/binary.go:22 +0x1c
github.com/xianxu/pair/cmd/internal/couchcmd.TestMain.func1()
	github.com/xianxu/pair/cmd/internal/couchcmd/testmain_test.go:12 +0x20
github.com/xianxu/pair/cmd/internal/testdisk.runMeasured({{0x105693326, 0x8}, 0x6000000, 0x1, 0x1, {0x0, 0x0}}, 0xd6dbbc33d88, 0xd6dbbe75df8, {0x1064be130, ...})
	github.com/xianxu/pair/cmd/internal/testdisk/run.go:35 +0x88
github.com/xianxu/pair/cmd/internal/testdisk.RunWithWriteBudget({0x105693326, 0x8}, 0x6000000, 0xd6dbbb77df8)
	github.com/xianxu/pair/cmd/internal/testdisk/run.go:87 +0x3c4
github.com/xianxu/pair/cmd/internal/couchcmd.TestMain(0xd6dbbcfe320)
	github.com/xianxu/pair/cmd/internal/couchcmd/testmain_test.go:16 +0xb8
main.main()
	_testmain.go:1322 +0x88

goroutine 3570 [chan receive]:
testing.(*T).Run(0xd6dbc55fb08, {0x10568dce5?, 0x1c428fde7?}, 0xd6dbbe3bcb0)
	testing/testing.go:2266 +0x3cc
github.com/xianxu/pair/cmd/internal/couchcmd.TestMessageAdmissionRejectsMismatchesThenGoesDormant(0xd6dbc55fb08)
	github.com/xianxu/pair/cmd/internal/couchcmd/message_service_test.go:393 +0x88
testing.tRunner(0xd6dbc55fb08, 0x1064ca0b8)
	testing/testing.go:2193 +0xc4
created by testing.(*T).Run in goroutine 1
	testing/testing.go:2258 +0x3b8

goroutine 2547 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/terminal.(*Presenter).run(0xd6dbbd14e08)
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:152 +0x118
created by github.com/xianxu/pair/cmd/internal/terminal.NewPresenter in goroutine 2481
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:89 +0x1cc

goroutine 1245 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/terminal.(*Presenter).run(0xd6dbb9faa88)
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:152 +0x118
created by github.com/xianxu/pair/cmd/internal/terminal.NewPresenter in goroutine 1298
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:89 +0x1cc

goroutine 3605 [IO wait]:
internal/poll.runtime_pollWait(0x153d4d800, 0x72)
	runtime/netpoll.go:351 +0xa0
internal/poll.(*pollDesc).wait(0xd6dbbc71e00?, 0xd6dbbc37388?, 0x0)
	internal/poll/fd_poll_runtime.go:84 +0x28
internal/poll.(*pollDesc).waitRead(...)
	internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0xd6dbbc71e00, {0xd6dbbc37388, 0x4, 0x4})
	internal/poll/fd_unix.go:170 +0x22c
net.(*netFD).Read(0xd6dbbc71e00, {0xd6dbbc37388?, 0x106583fc0?, 0x1?})
	net/fd_posix.go:68 +0x28
net.(*conn).Read(0xd6dbc3d2700, {0xd6dbbc37388?, 0xd6dbc577bb8?, 0x1052615f0?})
	net/net.go:196 +0x34
io.ReadAtLeast({0x153d33820, 0xd6dbc3d2700}, {0xd6dbbc37388, 0x4, 0x4}, 0x4)
	io/io.go:335 +0x90
io.ReadFull(...)
	io/io.go:354
github.com/xianxu/pair/cmd/internal/couchmessage.transportReadFrameLimit({0x153d33820, 0xd6dbc3d2700}, 0x1400)
	github.com/xianxu/pair/cmd/internal/couchmessage/transport.go:315 +0x50
github.com/xianxu/pair/cmd/internal/couchmessage.readSessionFrame({0x1064c8da8?, 0xd6dbc3d2700?})
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:154 +0x68
github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionServer).serve(0xd6dbc0534a0, 0xd6dbc3d2700, 0x1, 0x0)
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:137 +0x648
created by github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionServer).run in goroutine 3584
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:93 +0x250

goroutine 1312 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/terminal.(*Presenter).run(0xd6dbb9fa008)
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:152 +0x118
created by github.com/xianxu/pair/cmd/internal/terminal.NewPresenter in goroutine 1297
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:89 +0x1cc

goroutine 3604 [chan receive]:
github.com/xianxu/pair/cmd/internal/couchmessage.(*Server).run.func1()
	github.com/xianxu/pair/cmd/internal/couchmessage/transport.go:145 +0x5c
created by github.com/xianxu/pair/cmd/internal/couchmessage.(*Server).run in goroutine 3583
	github.com/xianxu/pair/cmd/internal/couchmessage/transport.go:143 +0x80

goroutine 1616 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/terminal.(*Presenter).run(0xd6dbb9fb888)
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:152 +0x118
created by github.com/xianxu/pair/cmd/internal/terminal.NewPresenter in goroutine 1615
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:89 +0x1cc

goroutine 2487 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbc1347e0)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 2481
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 3582 [select]:
github.com/xianxu/pair/cmd/internal/couchcmd.(*messageService).loop(0xd6dbb9fe960)
	github.com/xianxu/pair/cmd/internal/couchcmd/message_service.go:751 +0x1c4
created by github.com/xianxu/pair/cmd/internal/couchcmd.newMessageService in goroutine 3581
	github.com/xianxu/pair/cmd/internal/couchcmd/message_service.go:629 +0x71c

goroutine 2480 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbbe1b9e0)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 2536
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 3618 [IO wait]:
internal/poll.runtime_pollWait(0x153d33400, 0x72)
	runtime/netpoll.go:351 +0xa0
internal/poll.(*pollDesc).wait(0xd6dbc062d80?, 0xd6dbbc37340?, 0x0)
	internal/poll/fd_poll_runtime.go:84 +0x28
internal/poll.(*pollDesc).waitRead(...)
	internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0xd6dbc062d80, {0xd6dbbc37340, 0x1, 0x1})
	internal/poll/fd_unix.go:170 +0x22c
net.(*netFD).Read(0xd6dbc062d80, {0xd6dbbc37340?, 0xd6dbc13e768?, 0x1052635e8?})
	net/fd_posix.go:68 +0x28
net.(*conn).Read(0xd6dbb993460, {0xd6dbbc37340?, 0xd6dbc13e5e8?, 0xd6dbc13e5d8?})
	net/net.go:196 +0x34
github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionClient).session.func1()
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:368 +0x7c
created by github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionClient).session in goroutine 3585
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:365 +0x57c

goroutine 3581 [sleep]:
time.Sleep(0x5f5e100)
	runtime/time.go:368 +0x150
github.com/xianxu/pair/cmd/internal/couchcmd.TestMessageAdmissionRejectsMismatchesThenGoesDormant.func1(0xd6dbbd4a6c8)
	github.com/xianxu/pair/cmd/internal/couchcmd/message_service_test.go:420 +0x1fc
testing.tRunner(0xd6dbbd4a6c8, 0xd6dbbe3bcb0)
	testing/testing.go:2193 +0xc4
created by testing.(*T).Run in goroutine 3570
	testing/testing.go:2258 +0x3b8

goroutine 3590 [chan receive]:
github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionServer).run.func1()
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:68 +0x38
created by github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionServer).run in goroutine 3584
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:67 +0x64

goroutine 2538 [chan receive, 19 minutes]:
github.com/xianxu/pair/cmd/internal/hostty.(*OSHost).watch(0xd6dbb9aa770)
	github.com/xianxu/pair/cmd/internal/hostty/os.go:69 +0x98
created by github.com/xianxu/pair/cmd/internal/hostty.NewOSHost in goroutine 2537
	github.com/xianxu/pair/cmd/internal/hostty/os.go:53 +0x21c

goroutine 1574 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbbd49440)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 1519
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 2539 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/terminal.(*Presenter).run(0xd6dbbd64008)
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:152 +0x118
created by github.com/xianxu/pair/cmd/internal/terminal.NewPresenter in goroutine 2537
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:89 +0x1cc

goroutine 1455 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbb912cc0)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 1485
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 1555 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbc132480)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 1550
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 2477 [chan receive, 19 minutes]:
github.com/xianxu/pair/cmd/internal/hostty.(*OSHost).watch(0xd6dbbb21a40)
	github.com/xianxu/pair/cmd/internal/hostty/os.go:69 +0x98
created by github.com/xianxu/pair/cmd/internal/hostty.NewOSHost in goroutine 2536
	github.com/xianxu/pair/cmd/internal/hostty/os.go:53 +0x21c

goroutine 1614 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/terminal.(*Presenter).run(0xd6dbb9fb188)
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:152 +0x118
created by github.com/xianxu/pair/cmd/internal/terminal.NewPresenter in goroutine 1613
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:89 +0x1cc

goroutine 3583 [IO wait]:
internal/poll.runtime_pollWait(0x153d32c00, 0x72)
	runtime/netpoll.go:351 +0xa0
internal/poll.(*pollDesc).wait(0xd6dbc062a00?, 0x1050a0158?, 0x0)
	internal/poll/fd_poll_runtime.go:84 +0x28
internal/poll.(*pollDesc).waitRead(...)
	internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Accept(0xd6dbc062a00)
	internal/poll/fd_unix.go:618 +0x210
net.(*netFD).accept(0xd6dbc062a00)
	net/fd_unix.go:149 +0x28
net.(*UnixListener).accept(0xd6dbbb4e738?)
	net/unixsock_posix.go:172 +0x20
net.(*UnixListener).AcceptUnix(0xd6dbc059350)
	net/unixsock.go:262 +0x2c
github.com/xianxu/pair/cmd/internal/couchmessage.(*Server).run(0xd6dbc062b80)
	github.com/xianxu/pair/cmd/internal/couchmessage/transport.go:154 +0x8c
created by github.com/xianxu/pair/cmd/internal/couchmessage.StartServer in goroutine 3581
	github.com/xianxu/pair/cmd/internal/couchmessage/transport.go:77 +0x178

goroutine 3584 [IO wait]:
internal/poll.runtime_pollWait(0x153d4d400, 0x72)
	runtime/netpoll.go:351 +0xa0
internal/poll.(*pollDesc).wait(0xd6dbc062c00?, 0x1050b0138?, 0x0)
	internal/poll/fd_poll_runtime.go:84 +0x28
internal/poll.(*pollDesc).waitRead(...)
	internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Accept(0xd6dbc062c00)
	internal/poll/fd_unix.go:618 +0x210
net.(*netFD).accept(0xd6dbc062c00)
	net/fd_unix.go:149 +0x28
net.(*UnixListener).accept(0xd6dbbc87f48?)
	net/unixsock_posix.go:172 +0x20
net.(*UnixListener).AcceptUnix(0xd6dbc0593e0)
	net/unixsock.go:262 +0x2c
github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionServer).run(0xd6dbc0534a0)
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:77 +0x70
created by github.com/xianxu/pair/cmd/internal/couchmessage.StartSessionServer in goroutine 3581
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:62 +0x1d4

goroutine 2546 [chan receive, 19 minutes]:
github.com/xianxu/pair/cmd/internal/hostty.(*OSHost).watch(0xd6dbbb2aee0)
	github.com/xianxu/pair/cmd/internal/hostty/os.go:69 +0x98
created by github.com/xianxu/pair/cmd/internal/hostty.NewOSHost in goroutine 2481
	github.com/xianxu/pair/cmd/internal/hostty/os.go:53 +0x21c

goroutine 2541 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbc1b0540)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 2537
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 1248 [syscall, 19 minutes]:
os/signal.signal_recv()
	runtime/sigqueue.go:149 +0x104
os/signal.loop()
	os/signal/signal_unix.go:23 +0x1c
created by os/signal.Notify.func2.1 in goroutine 1301
	os/signal/signal.go:164 +0x28

goroutine 2478 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/terminal.(*Presenter).run(0xd6dbbd14708)
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:152 +0x118
created by github.com/xianxu/pair/cmd/internal/terminal.NewPresenter in goroutine 2536
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:89 +0x1cc

goroutine 1434 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbbb5c7e0)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 1459
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 1368 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/terminal.(*Presenter).run(0xd6dbba41c08)
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:152 +0x118
created by github.com/xianxu/pair/cmd/internal/terminal.NewPresenter in goroutine 1349
	github.com/xianxu/pair/cmd/internal/terminal/presenter.go:89 +0x1cc

goroutine 1543 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbbb5c3c0)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 1509
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 1603 [select, 19 minutes]:
github.com/xianxu/pair/cmd/internal/ptychild.(*publication).run(0xd6dbbe1a0c0)
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:96 +0x150
created by github.com/xianxu/pair/cmd/internal/ptychild.newPublication in goroutine 1601
	github.com/xianxu/pair/cmd/internal/ptychild/publication.go:35 +0x108

goroutine 3585 [select]:
github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionClient).session(0xd6dbc1b4780, {0x1064c5e20, 0xd6dbba74dc0}, {0xd6dbc05ec80, 0x4d}, {{0xd6dbc3a73a0, 0x6}, {0x10569646a, 0xa}, {0x10568fb4c, ...}, ...}, ...)
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:376 +0x6d8
github.com/xianxu/pair/cmd/internal/couchmessage.(*SessionClient).Run(0xd6dbc1b4780, {0x1064c5e20, 0xd6dbba74dc0}, {0xd6dbc05ec80, 0x4d}, {{0xd6dbc3a73a0, 0x6}, {0x10569646a, 0xa}, {0x10568fb4c, ...}, ...}, ...)
	github.com/xianxu/pair/cmd/internal/couchmessage/session_transport.go:271 +0xd4
github.com/xianxu/pair/cmd/internal/couchcmd.(*serviceRig).wrapper.func1()
	github.com/xianxu/pair/cmd/internal/couchcmd/message_service_test.go:261 +0xa8
created by github.com/xianxu/pair/cmd/internal/couchcmd.(*serviceRig).wrapper in goroutine 3581
	github.com/xianxu/pair/cmd/internal/couchcmd/message_service_test.go:259 +0x1fc
```

2026-10-19 (pair:2) — Built. TestLifecyclePublisherOrdinaryFactWaitsForItsWorkspace observes the ordinary fact before the worker connects, so the window is forced, not hoped for. Before the fix it failed with the new diagnostic naming the cause: `unresolved=[] identities=[/repo1 /repo2] roots=[] ready=0 attempts=0 workspaces=[pair:1=/repo1 pair:2=/repo2]`. The identity was cached and nothing was waiting, so the fact had been dropped (production behavior, not fixture-only). After the fix it passes. TestLifecycleUnresolvedFactsAreBoundedOldestFirst covers the cap. The primary-slot test now waits on the cached workspace (waitWorkspace). `go test ./cmd/internal/couchcmd -run TestLifecycle -count=50` passed in 59s with one `yes` burner per CPU. Atlas couch.md lifecycle section names the unresolved set.

Close round 1 (FIX-THEN-SHIP). BR-1: the drain diagnostic now counts only untargeted facts; ordinary ones clear silently (TestLifecycleDrainReportsOnlyUntargetedFacts, red when the count includes ordinary facts). BR-2: lifecycleNextMessage now requires the publisher, so all 14 call sites carry the state diagnostic. BR-3: an ordinary fact expires after lifecycleOrdinaryWait (2 min; the gap it covers is seconds, and admission itself is capped at 10 s), pruned on refresh (TestLifecycleOrdinaryFactExpires); untargeted facts still wait for the drain. Atlas states what drain reports and the expiry.
