---
id: 000542
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: 'ba1a6136f8f80c810967fa5f46b947dace49ab18' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T18:00:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ce386477", done: "92c14d9a"}
---

# PTY canceled-before-ack reports a live process group after the child is reaped

## Problem

TestBlockedRunnerCancellationConformance/pty/canceled-before-ack failed in /tmp/pair536-make-k.log at runner_contract_test.go:59: canceled blocked child remained alive. It passed three consecutive isolated reruns (/tmp/pair536-runner-rerun.log).

## Spec

Measure the cancellation/observation race without relaxing timing. Cancel closes the acknowledgement writer and terminal.Close kills/reaps the direct child; Wait observes that child, while Alive probes the whole process group. Before acknowledgement the launch helper cannot exec the target or create target descendants. Preserve process-group ownership guarantees.

This is independent of #519: it uses couchcore runner/helper code, not notifytransport or the wrapper output pump. A Darwin group-teardown/probe race is a hypothesis, not a proven cause. Capture PID, exit status, direct-child and group liveness, and group members on failure.

## Done when

- Root cause reproduced with retained process evidence and fixed without a sleep or weakened descendant check.
- Cancellation conformance passes repeatedly alone and in the full suite.
- #511 survivor conformance passes 50 times both at low load and under natural parallel host load, with readiness established before parent exit.

## Plan

- [x] Retain failure diagnostics and establish the failing observation sequence.
- [x] Fix ownership or observation at its source; add regression coverage and verify.

## Log

### 2026-10-13
- 2026-10-13: closed — Reproduced original PTY cancellation false positive: reaped PID ESRCH, group EPERM, no remaining group member. Deterministic injected-EPERM/real-kernel regression failed before fix, passes after. Cancellation, ambiguity, uncertainty and survivor regressions passed 50 times at load 35 to 34; #511 survivor fixture also passed 50 under load 67 and 50 at load 7 to 6.7. Race checks and Linux cross-compile pass. Full make -k test exit 0 (couchcore 377.409s), /tmp/pair542-make-k-final.log; then merged unrelated #529 main cleanly, runner code unchanged. No timing relaxation, no verification exceptions.; review verdict: SHIP

- Filed from #519 investigation at TL request. Read-only source analysis confirms a separate subsystem; no live-slot changes or additional cancellation loops run.

- Started #542 after #519 landed. Unchanged cancellation conformance passed 20 repetitions at load 5.10 (0.618s); retain diagnostics before further reproduction. ARCH-ORDER: distinguish child reaping from whole-group absence; ARCH-PURPOSE: preserve descendant ownership. ARCH-DRY/ARCH-PURE: use existing runner and process seams, keep probes at the OS boundary. ARCH-MOCK: existing fake/exec/PTY conformance remains the oracle. ARCH-CONSTRAINTS: bounded sequential reproduction, no load generation or changed deadlines. ARCH-SECURE/ARCH-FUNERAL: only fixture-owned children and temporary directories, cleaned by each test; no live-slot state or secrets.

- Reproduced unchanged PTY cancellation under bounded sequential runs (one failure in 2,000; 10.278s, /tmp/pair542-probe2.log): PID 30984, Wait=-1, direct kill(0)=ESRCH, group kill(0)=EPERM, no matching PID/PGID in the immediate process snapshot. Darwin Go syscall.Kill uses the POSIX flag; Apple [killpg1](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/kern_sig.c) returns EPERM when an existing group has no signalable members. Investigating a deterministic empty-foreground-group fixture before selecting the fix. No deadlines changed.

- Deterministic recorded-outcome regression failed before the fix (/tmp/pair542-red.log). With the fix, original cancellation conformance and all new group tests passed 20 times (4.908s; /tmp/pair542-green20.log). Load rose from 37.31 to 48.68 during qualification; no timing threshold was changed. Darwin membership queries only run for EPERM; group descendants and snapshot failures remain owned.

- #511 readiness fixture passed 50 repetitions under existing parallel host load (3.849s; load 67.32; /tmp/pair542-descendant50.log). Added explicit successful parent-exit confirmation and the same ambiguous-probe path against its surviving descendant. Artifact-source classification passes.

- Final focused qualification: cancellation conformance, ambiguous empty/live groups, unknown-membership safety, and the handshaked survivor fixture all passed 50 consecutive repetitions (8.615s; /tmp/pair542-qualification50.log), load 35.06 → 34.09 (/tmp/pair542-qualification-load.log). Race-enabled same cases passed (1.634s; /tmp/pair542-race.log).

- Full make -k test passed (exit 0, /tmp/pair542-make-k-final.log); couchcore passed in 377.409s. Linux/amd64 couchcore test binary cross-compiled successfully (/tmp/pair542-linux-build.log). #511 passed another 50 repetitions at load 7.19 → 6.70 in 2.393s (/tmp/pair542-survivor-idle50.log). No verification exceptions.
- Merged main after the stable full-suite run: unrelated #529 start-form inventory initialization plus documentation, with no code conflict or change to the verified runner implementation. #511's implementation, qualification and review are covered by this issue; its card will receive the landing pointer after merge.

## Revisions

### 2026-10-13 — measured ambiguous Darwin group probe

The cause is treating Darwin EPERM as positive group-membership evidence. The observed post-Wait direct ESRCH/group EPERM sequence and member-free process snapshot demonstrate the false positive; the exact kernel scheduling is not claimed. Published XNU also returns EPERM for zero signalable members, but differs in details of group removal, so a permanent tty-reference explanation was investigated and rejected.

Keep the common signal probe. On EPERM only, Darwin queries kern.proc.pgrp; a successful empty snapshot means absent, any member or query error retains ownership. Other platforms preserve the existing conservative behavior. No retries, sleeps, deadlines or descendant-check weakening. A regression injects the recorded EPERM outcome while retaining a real reaped child and real kernel membership query; live membership and query-failure regressions protect conservative ownership.

### 2026-10-13 — include #511's survivor readiness fixture

TL approved folding #511 into #542 and closing it with a landing pointer. The existing survivor test failed with the old liveness behavior (2/5 PTY repetitions, raw group ESRCH, /tmp/pair542-descendant-baseline.log). Its parent exits concurrently with the child's installation of SIGHUP ignore. Add a FIFO readiness handshake after the trap, before parent exit, with no changed deadline. Redirect survivor stdio away from the PTY so only group membership, not retained terminal descriptors, defines this fixture. Qualify #511's 50 repetitions under natural machine load; do not generate load. The initial full suite was stopped while still queued, before it ran, to preserve a stable test snapshot.
