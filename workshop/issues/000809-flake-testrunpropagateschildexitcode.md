---
id: 000809
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0ac1eb55d9c3c2662ac0bbd3404c99115133282f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T12:41:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/scribecmd.TestRunPropagatesChildExitCode]
---

# flake: TestRunPropagatesChildExitCode

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/scribecmd.TestRunPropagatesChildExitCode: pair#797 recorded suite attempt 5df63acf2f4b4b91b3ec17108300a892 on exact ceda3da54a7bd9577dfd529da9d8f89995715053 failed only TestRunPropagatesChildExitCode: exit 42 propagated as 1 at scribecmd_test.go:94; every other package and all script phases finished. Suite duration 12m5s, raw log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/5df63acf2f4b4b91b3ec17108300a892.log SHA256 e2caee9abf54beb8f1ea404ccad0245b5f0bc7828612d57b2f50489ce9027d35. Exact-commit detached checkout qualification go test ./cmd/internal/scribecmd -run ^TestRunPropagatesChildExitCode$ -count=100 -json then passed 100/100, package 4.067s; raw /Volumes/TestScratch/tmp/pair797-scribe-qualification.yhdzFn/qualification.jsonl SHA256 9fcf974492cf9205203ed137b05d66f0d910a89cb931a47398f746e3e4358807. No scribe changes in #797; root cause unclassified, original helper discards Run stderr, so failing fatal error unavailable. An additional code-identical close-head qualification passed100/100 in1.658s but exact-commit qualification above is authoritative. This first unrelated intermittent suite failure is registered per TL-worker #394; no source, timeout or assertion changed.

## Spec

### Root cause: measured and reproduced

`run` obtains its child pty through `startPTY` → `creack/pty.Open()` → `open("/dev/ptmx", O_RDWR|O_CLOEXEC)`. On macOS/XNU that open intermittently returns **ENXIO (errno 6, "device not configured")**, and it needs *concurrent* pty churn to happen: it is a race in XNU's ptmx index allocator, not resource exhaustion.

Replicating `creack/pty`'s darwin `open()` step by step so a failure names its own syscall, on this machine (darwin/arm64, `kern.tty.ptmx_max` 511):

| concurrent allocators | allocations | `open(/dev/ptmx)` ENXIO |
|---|---|---|
| 1 | 20,000 | 0 |
| 2 | 40,000 | 0 |
| 4 | 80,000 | 2 |
| 8 | 160,000 | 46 |
| 8 | 160,000 | 52 |
| 16 | 320,000 | 218 |

The rate rises monotonically with allocator concurrency — the signature of a race, not of a limit. Every failure was at `open(/dev/ptmx)`; `TIOCPTYGNAME`, `TIOCPTYGRANT`, `TIOCPTYUNLK` and the slave open never failed.

**Retry recovery, which sizes any bound:** at 8 allocators all 52 failures recovered on the immediately following attempt. At 16 allocators, 217 of 218 recovered on the next attempt and **1 needed a third attempt**. So a bound of one retry is measurably insufficient; the bound has to be at least three attempts.

`run` maps that failure to `fatalf` → **exit code 1** (`scribecmd.go:80` with `:206-209`). The child never started, so no exit status was ever collected or propagated. `TestRunPropagatesChildExitCode` asserts `code == want` (`scribecmd_test.go:93-95`) and reports "child exit 42 propagated as 1" — a misdiagnosis — and the helper passes `io.Discard` as Run's stderr (`scribecmd_test.go:79`), destroying the only diagnostic that names the site. That is exactly why the incident was recorded as "root cause unclassified".

End-to-end reproduction of the reported symptom — 8 goroutines churning pty open/close while the same child-launch shape runs:

```
iter 15: child exit 5 propagated as 1; stderr="scribe: pty.Start: errno -6"
```

This also explains the recorded evidence. The exact-commit qualification (`-run ^TestRunPropagatesChildExitCode$ -count=100`) is a **single sequential allocator**; the 1- and 2-allocator rows above are 0 failures, so 100/100 is the expected result and says nothing about the suite path. The suite runs ~12 packages in parallel, several churning ptys, so the ≥4-allocator regime is live for its whole 12 minutes.

### Production impact

Not test-only. `zshrc`'s `exec pair scribe -log "$LOG" -- /bin/zsh` runs the same `startPTY`; inside that window scribe exits 1 instead of starting the user's shell.

### Ruled out by measurement

Each was a live hypothesis before measurement:

- **pty exhaustion** — peak 230 concurrent `/dev/ttys*` of `kern.tty.ptmx_max` 511, sampled at 4 Hz for 5 min while 11 pty-heavy packages ran under `-p 12`.
- **fd / proc exhaustion** — fd soft limit 1048576; `kern.maxprocperuid` 10666; 6,000 child runs under a 6-goroutine fork storm gave 0 divergences.
- **disk / memory** — TestScratch 9.1 GiB free, 0% inodes; 96 GiB RAM, swap 0.
- **exit-status collection or propagation** (the dispatch's hypothesis) — Go 1.27.1 `os/exec.Cmd.Wait()` (`$GOROOT/src/os/exec/exec.go:936-972`) returns a bare `*ExitError` on this path: no Context, so `ctxResult` is nil, and Stdin/Stdout/Stderr are all `*os.File`, so there are no copying goroutines. `werr.(*exec.ExitError)` therefore cannot miss, and a signal-killed child yields `ExitCode() == -1`, not 1. **Exit code 1 is reachable only from `run`'s four `fatalf` sites** — log open (`:71`), `pty.Start` (`:80`), `MakeRaw` (`:157`), `cmd.Wait` (`:201`). There is no race in how the status is collected; the status is never reached.
- **cross-reaping → `cmd.Wait()` ECHILD** — no `wait4(-1)`, `waitid(P_ALL)` or SIGCHLD reaper anywhere in the tree.
- **in-process test ordering** — 25 full-package runs (every `pty_darwin_test.go` test included) clean.
- **controlling-terminal collision** — `creack/pty` opens the slave with `O_NOCTTY`, so the parent never adopts it and the child's `TIOCSCTTY` cannot collide.
- **the failing run's 0.07s duration is not evidence** — warm isolated runs measure 0.03–0.05s and the recorded qualification 0.017–0.041s per run, so a loaded 0.07s cannot discriminate which site fired.

### Required behavior

1. A transient pty-allocation ENXIO must not prevent scribe from starting the child.
2. When scribe *does* abort, the abort must be distinguishable from "the child exited 1" — by the caller and by the test — and the diagnostic must survive.
3. No retry loop in the test, no timeout extension, no assertion weakened.

### Open design decision (needs the hub's ruling)

`## Done when` forbids "a retry loop". The measured condition is transient — 52/52 recovered on the very next attempt at 8 allocators, 217/218 at 16 with one needing a third — which is the standard case *for* a narrow bounded retry, and it is also what fixes production. Whether the ban targets test-side retries (the usual anti-pattern) or also a production-side bounded retry on a proven-transient errno is the hub's/operator's call, not mine. Option A below assumes the former, and the measurement sets its floor: **three attempts, not one retry.**

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

Option A (recommended) is 1+2+3+4; Option B is 2+3+4 only and leaves the flake registered.

- [ ] 1. **Transient allocation handling** (`ARCH-CONSTRAINTS`, pending the ruling above): in `startPTY`, retry the pty allocation a small bounded number of times on ENXIO only, and still emit the diagnostic if the bound is exhausted. No timeout change, no general retry, no test-side retry. darwin path only — `pty_other.go` keeps its shape.
- [ ] 2. **Oracle fix** (`ARCH-ORDER` — "attacks the oracle rather than the code"): `runWithPTYReply` captures Run's stderr instead of `io.Discard`, and a divergence reports the captured diagnostic. Add an assertion that separates "scribe aborted" from "child's code mis-propagated", so the class can never be unclassifiable again.
- [ ] 3. **Injected-failure coverage** (`ARCH-DRY` — reuse the existing `start` seam and the `TestDarwinRunAbortsChildOnRawModeFailure` pattern): pin each `fatalf` site to exit 1 plus its diagnostic, including the ENXIO allocation failure measured here.
- [ ] 4. **MISTAKE CLASSES** (tl-worker): the mechanical class is "a test asserts a wrapper's int exit code while passing `io.Discard` as the wrapper's diagnostic writer". Propose the rule to ariadne's `pkg/golint` catalog citing this incident; if it is not mechanically detectable without false positives, record it in `workshop/lessons.md` instead and say so.
- [ ] 5. Verify: the churn harness no longer diverges; `go test ./cmd/internal/scribecmd -count=1` clean; recorded suite via `sdlc local-check run --name suite` on the pushed head.
- [ ] 6. Report the outcome to the hub; on SHIP run `sdlc merge --yes` from this slot.

Side quest (not proven causal — `poll.FD` refcounting already makes use-after-close return `ErrFileClosing`): the `io.Copy(ptmx, stdin)` goroutine (`scribecmd.go:164-166`) is the only goroutine in `run` whose extent is not lexically bounded; winch (`:119-123`) and pause (`:145-149`) are both joined, with a comment naming the fd-recycling hazard. `ARCH-ORDER` wants extent bounded. In scope only if the hub wants it.

## Log

### 2026-10-22

12:41 Slot restarted fresh; previous screen showed the prior session cancelled at startup, so no work was in flight and there was no unanswered hub report to resend. Hub dispatched pair#809 (ops:0) with a 15:45 PT checkpoint inside a window ending 16:00 PT.

12:44 `sdlc claim --issue 809` — claimed by pair:8; slot labelled.

12:47–12:49 `sdlc start-plan --issue 809` first refused: its own catch-up ran `weave refresh` after bumping ariadne `2dc46703` → `5785330c`, which regenerated `.github/workflows/merge-check.yml` (a SIGPIPE/drain fix in the compiler-provenance step) on the resting branch. Not my change and not #809's; saved the patch to `/Volumes/TestScratch/tmp/pair-slot8/pair809-drift.yf8woY/merge-check-weave-drift.patch`, restored the file, re-ran. **Fleet-wide: every slot that refreshes against the new ariadne will hit this — raised with the hub in the plan report below.** Branch `000809-flake-testrunpropagateschildexitcode` created at main.

12:52–13:03 Elimination by measurement, all negative: 25 full-package runs clean; 6,000 child launches under a 6-goroutine fork storm, 0 divergences; peak 230 of 511 ptys under 11 pty-heavy packages at `-p 12`; fd/proc/disk/memory limits all far from binding; no reaper in the tree; `creack/pty` uses `O_NOCTTY`. Read Go 1.27.1 `os/exec.Cmd.Wait()` to prove the `*ExitError` type assertion cannot miss and that exit 1 comes only from the four `fatalf` sites — this falsified the dispatch's "race in exit-status collection" hypothesis.

13:03 Two unrelated failures surfaced in that 11-package run while my own load harnesses were hammering the machine concurrently: `wrapcmd.TestWrapperTitleLeaseReplacement` ("leased title readiness incomplete", plus a `notify;pair;Conversation lock discovery unavailable` in its log) and `couchcore` hitting the 10m test timeout. **Confounded by my own added load — not verified as real, not claimed as findings.**

13:06 **Reproduced the reported symptom.** 8 goroutines churning pty open/close alongside the child-launch shape: `iter 15: child exit 5 propagated as 1; stderr="scribe: pty.Start: errno -6"`. Same message shape and same assertion line as the incident.

13:09–13:13 **Pinned the syscall and the transience.** Step-by-step replication of `creack/pty`'s darwin `open()`: 46/160,000 then 52/160,000 failures, every one at `open(/dev/ptmx)` with ENXIO, and all 52 recovered on the immediately following attempt. Concurrency sweep: 0 failures at 1 allocator (20k) and 2 (40k), 2 at 4 (80k), ~0.03% at 8 (160k), 218 at 16 (320k) — monotonic in concurrency, so a race rather than a limit. At 16 allocators 217/218 recovered on the next attempt but **1 needed a third**, which sets the floor for any retry bound.

13:19 Diagnostic instrumentation for this investigation lived in two untracked scratch files (`cmd/internal/scribecmd/zz_scratch809_test.go`, `zz_scratch809b_test.go`). They are **not** part of the fix: removed from the tree before this commit, copies kept at `/Volumes/TestScratch/tmp/pair-slot8/pair809-scratch.lIOybI/` so Plan item 5 can re-run the same harness. The permanent equivalents are Plan items 2 and 3.
