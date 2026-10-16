---
id: 000609
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: medium
card_mirror: '94b3fed43c408adcae9f0712294c24d04827a149' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T09:57:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "88e45015", done: "36cf2873"}
---

# Launcher clean-exit cancellation test races TERM trap readiness

## Problem
The clean-exit handoff fixture cancels after 100 ms. Under load TERM can arrive before the shell installs its clean-exit trap, yielding -1,nil rather than 0,nil. The sibling escalation fixture already publishes trap readiness.

The recorded full suite for #589 at head ff788e65bb02408429d7e974389a2511d8492db2 failed TestCancelledHandoffReportsAChildsCleanExit in cmd/internal/launcher/osruntime_launch_test.go:85: handoff = -1, <nil>; want the child's own 0, nil. All other packages passed, including couchcmd, couchmessage and couchcore. Launcher source and tests have no diff from origin/main in #589.

The fixture sends cancellation after a fixed 100 ms without confirming that its shell installed the TERM trap. A delayed shell startup could therefore receive TERM before the trap is ready. This is a hypothesis, not a controlled reproduction. The neighboring TERM-ignoring test already waits for a trap-ready file before cancellation.

## Spec
Replace the timer with a test-owned trap-ready file written only after trap installation. Run the blocking handoff asynchronously, wait using testwait, cancel, and join before asserting the existing child exit contract. Cleanup cancels and joins even if readiness fails; the private directory owns the readiness file. Keep production, existing deadlines and assertions unchanged. ARCH-DRY: reuse the sibling readiness pattern and testwait; ARCH-ORDER/FUNERAL: readiness precedes cancellation and cleanup joins the child.

Measure the failure ordering with a controllable fixture. Synchronize cancellation with the child's actual trap readiness while preserving the intended contract: a child that exits cleanly after cancellation returns its own exit status. Do not hide signalled exits, add sleeps, or extend deadlines.

## Done when
- [x] The clean-exit test cancels only after actual trap readiness and retains the 0,nil assertion.
- [x] Controlled pre-trap cancellation reproduces the signal result; corrected ordering and repeated focused/race/package checks pass.
- [ ] Recorded full suite, CI and close review pass; land through the line.

- A controlled ordering reproduction identifies why the clean-exit test observed a signalled exit.
- Cancellation observes actual child readiness rather than elapsed time.
- Isolated repeated and full-suite verification pass; production cancellation semantics remain covered.

## Plan
- [x] Record baseline and a controlled pre-trap readiness failure.
- [x] Replace the fixed cancellation timer with ready-file ordering and bounded join/cleanup.
- [x] Qualify focused repeats, race and launcher package; run recorded suite. Close and landing remain tracked in Done when.

## Log

- 2026-10-16: closed — Controlled pre-trap cancellation reproduces -1,nil; ready-ordered control passes 30; corrected clean-exit test passes 100; cancellation group passes -race count=30; whole launcher passes; recorded suite on ae59d8e43e06d43577e31da0faa17cab3b0ff417 passes (14m56s, log 3718f2999a52), CI green. Test fixture only, no new architectural surface; TL owns project records.; review verdict: SHIP
2026-10-16: Verified TL dispatch adc64359-f62a-452c-9eab-77625699f148, small plan may proceed after posting. Claimed and started from main. start-plan refreshed the stale ariadne tools and requested one canonical retry; retry succeeded. Existing sibling uses trap-ready file and testwait. Production runBlockingHandoff correctly reports child status; this is fixture ordering only.

2026-10-16: Filed unassigned from pair:1 at the TL's direction, separate from #589. Failed recorded command: sdlc local-check run --name suite (make PAIR_GO_TRIMPATH=-trimpath test), head ff788e65bb02408429d7e974389a2511d8492db2. Full log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/515ad3a2617b9f0f7d1aa19c39691236.log (sha256 prefix 0c6257440f80). This was the only failing test in the completed run.

Qualification on the same unchanged head: go test -trimpath ./cmd/internal/launcher -run '^TestCancelledHandoffReportsAChildsCleanExit$' -count=30 passed in 4.425s. This establishes intermittent behavior; it does not prove the suspected trap-installation ordering. No launcher changes or production slot effects. The TL authorized one unchanged recorded-suite rerun for #589 after this filing.

2026-10-16: Plan receipt f993db5e-d36f-4bca-b30c-d72ecb2bf4e2 confirmed submitted to ops:0. Unchanged baseline: focused clean-exit test -trimpath -count=30 PASS (4.422s). Deterministic control uses an overlay-only test: the shell publishes startup then blocks on a FIFO before installing its TERM trap. Original 100ms cancellation while the gate remains closed FAILS with exactly handoff=-1,nil (0.14s test, 0.412s package). Releasing the FIFO, waiting for the post-trap ready file, then cancelling PASSES 30 repetitions (3.124s package). Both controls use the real killOnCancel/runBlockingHandoff. Scratch overlay: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair609.jvvft6st; control_test.go SHA256 e01fbc7150c52dbae2a1faa4066950b2f9d168826cd49a6b2f51075537a063f0. Run with go test -trimpath -overlay <scratch>/overlay.json ./cmd/internal/launcher -run '^Test609TrapReadinessControl$' -count=1 -timeout=30s; set PAIR609_READY=1 and -count=30 for corrected ordering. This settles the original hypothesis as fixture ordering, not production behavior.

2026-10-16: Implemented post-trap ready file, asynchronous handoff with close(done) synchronizing result reads, and LIFO cleanup cancelling/joining before TempDir removal. Early child exit fails immediately rather than waiting for readiness. Production and original child-status assertion unchanged; no new sleep or extended timeout. Focused clean-exit -trimpath -count=100 PASS (9.370s); cancellation group -trimpath -race -count=30 PASS (14.598s).

2026-10-16: Whole launcher package go test -trimpath ./cmd/internal/launcher -count=1 PASS (51.710s). git diff --check clean.

2026-10-16: Recorded suite PASS on ae59d8e43e06d43577e31da0faa17cab3b0ff417: sdlc local-check run --name suite (make PAIR_GO_TRIMPATH=-trimpath test), execution 14m56s after 755s queue wait. Full log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/ae59d8e43e06d43577e31da0faa17cab3b0ff417/suite.log, SHA256 prefix 3718f2999a52. Launcher 40.129s; couchcore 376.471s. GitHub merge-check and local-gates-guard both PASS. No failures. Close review and landing pending.

## Revisions
2026-10-16: Clarified the final plan checkbox to cover completed qualification; mechanical close review and landing stay pending in Done when until their gates finish. No scope or design change. Removed an inherited empty plan checkbox.
