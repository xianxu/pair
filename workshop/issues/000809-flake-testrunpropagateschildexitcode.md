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
flow: {kind: quick, provenance: inferred, spec: "4cca90d6", done: "a0bbe98b"}
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

**The errno arrives negated, which decides how the guard must be written.** The first classification harness opened `/dev/ptmx` with `unix.Open` rather than `syscall.Open`, so its step attribution was re-checked by replicating `creack/pty`'s darwin `open()` through *its own* call paths — `syscall.Open` for `/dev/ptmx`, `ioctlInner`'s raw `syscall.Syscall` for the three ioctls. That run gave 11/160,000 failures at 8 allocators, every one at the ptmx open, each a `syscall.Errno` whose `int` value is **-6**, rendering exactly `errno -6` — byte-identical to the production diagnostic below. All 11 recovered on the immediately following attempt.

So `errors.Is(err, syscall.ENXIO)` is **false** for the error this fix has to catch, because `syscall.ENXIO` is 6. The negation is specific to this failure and not to the platform: `syscall.Open` on an absent path returns `Errno` 2, renders "no such file or directory", and `errors.Is` matches it normally. A guard written as `errors.Is(err, syscall.ENXIO)` would compile, read correctly, pass review — and never fire.

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
2. The guard must match ENXIO under **both** sign conventions — the negated `Errno` this failure was measured to produce, and the positive one `syscall.ENXIO` names — or it does not fire at all.
3. When scribe *does* abort, the abort must be distinguishable from "the child exited 1" — by the caller and by the test — and the diagnostic must survive.
4. No retry loop in the test, no timeout extension, no assertion weakened.

### Ruling on `## Done when`'s "never a retry loop" (hub, ops:0)

Asked because the measured condition is transient — 52/52 recovered on the very next attempt at 8 allocators, 217/218 at 16 with one needing a third — which is the standard case *for* a narrow bounded retry, and is also what fixes production. The hub ruled: the ban covers test-side and blind retries; **a production-side bounded retry on a proven-transient errno is root-cause handling of an OS race.** Approved constraints: at most 3 attempts, ENXIO only, darwin `startPTY` only, the diagnostic still emitted when the bound is exhausted, and the measurement table cited in the code comment and here. Plan Option A (items 1–4) approved; the `io.Copy` side quest is out of scope and filed separately as #813.

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

Option A (recommended) is 1+2+3+4; Option B is 2+3+4 only and leaves the flake registered.

- [x] 1. **Transient allocation handling** (`ARCH-CONSTRAINTS`, under the ruling above): in `startPTY`, retry the pty allocation a small bounded number of times on ENXIO only, and still emit the diagnostic if the bound is exhausted. No timeout change, no general retry, no test-side retry. darwin path only — `pty_other.go` keeps its shape.
- [x] 1a. **The fixture allocates through the same primitive** (scope delta, see `## Revisions`): the retry is factored into `openPTYRetry`, exported to the package as `openPTY` (darwin: bounded; `pty_other.go`: pass-through to `pty.Open`), and `runWithPTYReply` calls `openPTY` instead of `pty.Open`. Without it the flake survives at a new line — the fixture allocates one pty per `Run` invocation and is exposed to the same race.
- [x] 2. **Oracle fix** (`ARCH-ORDER` — "attacks the oracle rather than the code"): `runWithPTYReply` captures Run's stderr instead of `io.Discard`, and a divergence reports the captured diagnostic. Add an assertion that separates "scribe aborted" from "child's code mis-propagated", so the class can never be unclassifiable again.
- [x] 3. **Injected-failure coverage** (`ARCH-DRY` — reuse the existing `start` seam and the `TestDarwinRunAbortsChildOnRawModeFailure` pattern): pin each `fatalf` site to exit 1 plus its diagnostic, including the ENXIO allocation failure measured here.
- [x] 4. **MISTAKE CLASSES** (tl-worker): the mechanical class is "a test asserts a wrapper's int exit code while passing `io.Discard` as the wrapper's diagnostic writer". Assessed against ariadne's `pkg/golint` catalog and **rejected as not mechanically detectable without false positives** (`io.Discard` is correct for most writers; telling a diagnostic sink from an output sink needs semantics the catalog's `Match` funcs lack). Recorded in `workshop/lessons.md` instead, with the rejection and its reason, plus the negated-errno trap. A real rule would need its own ariadne issue.
- [x] 5. Verify: the churn harness no longer diverges — 4500 child runs against 413 live allocation failures, **0 divergences**, PASS 118.53s, `go test` exit 0; `go test ./cmd/internal/scribecmd -count=1` clean — `ok … 16.101s`; `go vet` clean on darwin and under `GOOS=linux` (the fixture is untagged, so both platforms must provide `openPTY`); TDD red before the fix on both errno spellings, green after, 5/5; `make test-affected-inventory` green after auditing both changed readers per file.

Landing follows this close by construction — `sdlc close` is the local acceptance gate and flips the issue to `codecomplete`; `sdlc merge` is the publish gate that later flips it to `done`. Neither can be a checkbox ticked before close:

- The recorded check is queued on the pushed head `e395c093d`. `sdlc local-check run --name affected` escalated to the full suite ("full selection … running suite instead") and stood at position 17 in line for one of the 2 machine-wide full-suite slots at 14:12, so it will not land inside today's window. `HEAD` differs from it only by docs (`ace7eeff6`: atlas + this file).
- Report the outcome to the hub; on SHIP run `sdlc merge --yes` from this slot.

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

13:22 Design checkpointed as `b994c1b9e` and the plan sent to the hub (delivery `ed036c80-abf8-4e5a-a0b7-2361f8b147e5`), per the dispatch's "send me a plan before code". The message carries the root cause and evidence, the `## Done when` ruling request, the fleet-wide weave-drift refusal, and the two confounded failures marked unverified. **Holding: no source touched, waiting on the bounded-retry ruling and plan approval.** Well inside the 15:45 PT checkpoint.

13:2x Hub ruling received (delivery `4b2538ba-13ba-4028-bec9-b4ee72c56787`): production-side bounded retry on a proven-transient errno is root-cause handling, not a banned retry loop; at most 3 attempts, ENXIO only, darwin `startPTY` only, diagnostic still emitted when exhausted, measurement table cited in code and here. Plan A items 1–4 approved, starting with 2 and 3. The `io.Copy` side quest is out of scope and was filed unclaimed as **#813** with the `ARCH-ORDER` reasoning; the pressure-run observations stay unfiled as proposed. `sdlc change-code --issue 809` inferred the **quick** flow and accepted the design at `29a8b794d` (13:28).

13:3x–13:5x **Items 2 and 3 first, then 1.** Oracle fix: `runWithPTY`/`runWithPTYReply` return Run's stderr as a third value instead of passing `io.Discard`, and `TestRunPropagatesChildExitCode` asserts the diagnostic empty *before* the int, so an abort can never again be reported as a mis-propagated status. Injected-failure coverage: `TestRunFatalSitesExitOneWithDiagnostic` pins log-open, pty-start and `cmd.Wait` to exit 1 plus their diagnostic (MakeRaw and the slave-release diagnostic were already pinned in `pty_darwin_test.go`); `TestDarwinStartRidesOutTransientAllocationENXIO` covers both errno spellings, recovery on the last attempt, exhaustion, and an unrelated errno not being retried. Red before the fix — `raw spelling: startPTYWithOpen: errno -6` and `libSystem spelling: startPTYWithOpen: device not configured` — green after, 5/5.

**The negated errno changed the guard's shape.** Writing `errors.Is(err, syscall.ENXIO)` would have compiled, read correctly and never fired: the measured failure arrives as an `Errno` of int value **-6**. Verified by re-running the step classifier through `creack/pty`'s *own* call paths (`syscall.Open` for ptmx, `ioctlInner`'s raw `syscall.Syscall` for the ioctls) after finding the first harness had used `unix.Open` — libSystem, positive errno — so its step attribution had not really been exercised. 11/160,000 failures at 8 allocators, every one at the ptmx open, all rendering exactly `errno -6`, all 11 recovered on the next attempt. `isTransientPTYAlloc` normalises the sign. A constant `syscall.Errno(-int(syscall.ENXIO))` does not compile ("constant -6 overflows uintptr"), so the test builds the raw spelling through a variable.

13:53 **End-to-end verification of the fix.** The churn harness that originally failed at iter 15 — 16 goroutines churning pty open/close against the child-launch shape — ran 1500 iterations / 4500 child runs to completion: `650762 churned ptys, 413 churn allocation failures, 0 divergences`, PASS in 118.53s, `go test` exit 0. The 413 churn failures confirm the race was live throughout; the 0 divergences confirm `startPTY` rode all of them out. (The first attempt at this run failed for a plumbing reason — the harness's *own* `pty.Open` hit the race and `t.Fatalf`'d — and a `grep -E` in the pipeline hid it by taking the exit code; re-run with full output to `/Volumes/TestScratch/tmp/pair-slot8/pair809-churn-withfix.log`.) That plumbing failure is the evidence for item 1a below.

14:02 **Item 1a — the fixture was still exposed.** `runWithPTYReply` allocated its own pty with a bare `pty.Open` and `t.Fatalf`'d on any non-permission error, so the same race would fail the test at a new line with the fix in place and make unregistering the flake wrong. Factored the bounded retry into `openPTYRetry`, exposed it as `openPTY` (darwin: bounded; `pty_other.go`: pass-through), and pointed the fixture at it — `ARCH-DRY`, one production primitive, two callers, no test-side retry. Scope delta recorded in `## Revisions` and flagged to the hub.

14:06 `go vet ./cmd/internal/scribecmd` and `GOOS=linux go vet ./cmd/internal/scribecmd` both clean (the fixture is in an untagged test file, so the non-darwin path has to provide `openPTY` too). Full package `go test ./cmd/internal/scribecmd -count=1` → `ok … 16.101s`. Item 4: the `pkg/golint` proposal was assessed and rejected as undetectable without false positives; both rules — the discarded-diagnostic class and the negated-errno trap — are recorded in `workshop/lessons.md` with the rejection reason. Temporary diagnostic files removed from the tree; copies remain at `/Volumes/TestScratch/tmp/pair-slot8/pair809-scratch.lIOybI/`. Remaining: item 5's recorded suite on the pushed head, then item 6.

14:09 Hub **approved item 1a** (delivery `a5c156e8-fa5b-42a1-b4bb-7cf1e0bd8994`): the fixture's own `pty.Open` hits the same OS race, and routing it through the one production helper is not a test-side retry. Directed to queue the recorded suite and run the close review in parallel — the queue runs about an hour — and to checkpoint, report and stop if 15:45 arrives first.

14:10 `main` integrated first (`7da53f536`, #811/#812 details only, no conflicts) so the recorded check applies to the head that lands rather than being invalidated by the integration. `make test-affected-inventory` named both changed test files as changed readers; **audited before recording**, per the repo rule: `pty_darwin_test.go` takes only literal argv (`/usr/bin/true`) and injected allocators, and every path in `scribecmd_test.go` is `t.TempDir()`-rooted, so it follows `TMPDIR`/`GOTMPDIR` to the TestScratch RAM disk on its own (#468) with no hardcoded temp path. Neither displaces preflight or fallback ownership, so `local_or_literal_fixture` is the correct disposition — recorded per file, not in bulk (`e395c093d`), inventory green. Branch pushed at `e395c093d`; `sdlc local-check run --name affected` queued in the background (`.sdlc/affected.json` is present, so affected rather than the whole suite). Atlas updated: `architecture.md`'s `pair scribe` Darwin paragraph now carries the bounded allocation, the negated-errno trap and the rule that fixtures allocate through the same `openPTY`.

14:13 The queued check escalated from affected to the **full suite** ("full selection … running suite instead") and stood at **position 17 in line** for one of the 2 machine-wide full-suite slots, so it cannot land inside today's window. That gates `sdlc merge`, not `sdlc close`, so the close review runs in parallel with the queue as the hub directed.

14:15 Confirmed against source that retrying the allocation cannot leak: `creack/pty@v1.1.24/pty_darwin.go`'s `open()` returns `nil, nil, err` on all five error paths (`:16,28,32,36,41`) and closes the ptmx descriptor it already holds in a defer (`:20`) — the claim the code comment makes is checked, not assumed. First `sdlc close` attempt **refused** on the plan-check: items 5 and 6 were unticked. Both are post-close by construction (close flips to `codecomplete`; `sdlc merge` is the publish gate that flips to `done`), so a checklist can never hold them ticked at close time. Restructured the Plan — item 5 now carries only the verification that is actually complete and is ticked, and the landing steps moved out of the checkbox list into prose saying why they cannot be checkboxes. Not bypassed with `--no-plan-check`. Close also reported: flow **quick**, inside the shell at 60 added lines in code files and a 71-line design; measured actual **1.49h** adopted.

## Revisions

**2026-10-22 14:02 — Plan item 1a added; the ruling's "darwin `startPTY` only" gains one caller.** Reason: verification proved the exposure is not confined to `startPTY`. The churn run recorded 413 allocation failures across 650,762 allocations, and the harness's own fixture `pty.Open` was one of the victims — it aborted the run. `runWithPTYReply` allocates one pty per `Run` invocation, so with `startPTY` fixed the same race would fail `TestRunPropagatesChildExitCode` at `scribecmd_test.go:70` instead of `:106`, and closing the issue would unregister a test that is still flaky. Delta: the bounded retry is factored into `openPTYRetry` and surfaced as `openPTY`; darwin gets the bounded version, `pty_other.go` a two-line pass-through to `pty.Open` so the untagged fixture compiles; the fixture calls `openPTY`. The retry itself is unchanged and still exists only on darwin — same bound, same errno, same measurement. No new retry site, no test-side retry, no assertion weakened. Flagged to the hub for veto and **approved at 14:09** (delivery `a5c156e8`): the fixture's own `pty.Open` hits the same OS race, and routing it through the one production helper is not a test-side retry.
