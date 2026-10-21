---
id: 000766
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c18a5d0fde0fad0323cdcabfc8362cb58d0697cd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T04:13:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/scribecmd.TestRunTeesChildOutputToStdoutAndLog]
flow: {kind: quick, provenance: operator, spec: "0432b67b", done: "a0bbe98b"}
---

# flake: TestRunTeesChildOutputToStdoutAndLog

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/scribecmd.TestRunTeesChildOutputToStdoutAndLog: Same exact head 263cbba51998e2f853e2c1012f9c36c39858ad20: full recorded suite attempt 05d38e5d5280975a07e7e6ad51a2b0b4 failed only this test in 10m30s (stdout empty); all remaining packages completed. Failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/05d38e5d5280975a07e7e6ad51a2b0b4.log. Subsequent exact targeted go test -run ^TestRunTeesChildOutputToStdoutAndLog$ -count=1 -v passed in 0.055s on unchanged head. A separate count=100 reproduced two failures in 2.591s, confirming intermittent missing PTY child output. No scribe source is changed by #761. Root cause remains to be established; this is registration of the observed same-head fail/pass, not a whole-suite passing claim.

## Spec

Find and fix the actual PTY output-loss cause, pin it with a regression that rejects the original failure rate, and retire the flake registration through successful issue closure. No retries or timeout increases.

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ] Trace the PTY launch/read/EOF and test lifetime with native probes; distinguish discarded kernel output from wrapper/test races.
- [ ] Pin the confirmed cause, implement the smallest ownership/synchronization correction, and verify the original stress case.
- [ ] Run recorded verification, close review and landing; closure retires the flake registration.

## Log

### 2026-10-21

- Claimed and started from main; #757 remains preserved. Source already drains before cmd.Wait and has no waiter closing the master. Initial 100 + 2000 ordinary repetitions and 1000 race repetitions passed. Native standalone delayed-first-read probe reproduces empty output after child exit; isolating the exact terminal state/lifetime before choosing a fix. No implementation changes yet.

- Native diagnosis: original test count=10000 failed 10 times (each missing stdout, exit zero). An instrumented Run copy with GOMAXPROCS=1 failed 41/5000; every failure's first master read was n=0, EOF. Scribe's tee never received the bytes. Opening /dev/tty in the child before output passed 300 delayed-read probes. XNU last-slave-close can discard unread data without the child's S_CTTYREF; /dev/tty establishes its exit-time drain reference. Primary references: https://bugs.ruby-lang.org/issues/20682 and https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/tty_tty.c . A wait-before-read diagnostic can itself block in child exit-time ttywait; it is not a valid drain strategy.
- Production exposure audit (TL prerequisite): YES. docs/pair-scribe.md publicly admits arbitrary CMD arguments, cmd/pair-go/main.go routes directly into Run, so one-shot scribe commands and final shell exec can lose final output. Normal /bin/zsh hooks retain the shell's slave for ordinary subprocesses. pair wrap also admits arbitrary commands, and Couch launch failures need final diagnostics; those other paths are exposure candidates, not reproduced findings. There is no public generic pair run family. A separate unclaimed product issue is being filed; #766 stays fixture-only per TL approval.

## Revisions

- 2026-10-21: TL approved fixture-only repair after the production audit. The regression producer explicitly opens /dev/tty, writes a readiness marker, waits for the parent tee to acknowledge via real PTY input, then writes a final marker and exits. This pins terminal lifetime plus the bidirectional copy boundary without sleeps, retries or timeout changes. Add a bounded native stress pin and a local lint guard against restoring the bare printf fixture; map the rule in atlas. Production launch policy is deferred to the separately filed product issue.
