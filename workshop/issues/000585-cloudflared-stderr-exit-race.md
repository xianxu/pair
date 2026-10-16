---
id: 000585
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'f2eddca716cba0126330060b050fd46099f983c5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T18:08:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "af139ff2", done: "45fa3bd6"}
---

# Preserve cloudflared stderr on early process exit

## Problem

The full recorded suite intermittently loses the fake cloudflared process's stderr failure before reporting its exit. This is independent of #571's Codex adapter; no real tunnel or network is used by this test.

## Spec

Investigate process exit versus stderr-drain ordering in `cmd/internal/broadcast/cloudflared.go`. `StderrPipe`, `watchStderr`, and a concurrent `cmd.Wait` currently feed separate completion paths. The working hypothesis is that the exited branch reads `lastLine` before the reader records the error, or Wait closes the pipe before it drains. This is a hypothesis, not a measured root cause yet. Preserve the original cloudflared failure without accepting its API URL as a tunnel URL.

## Done when

- A deterministic fake-process regression demonstrates the exit/stderr ordering problem and passes after the lifecycle fix, without sleeps or a larger timeout as the fix.
- A fast failing fake retains its stderr error; API failure URLs still cannot become successful tunnel URLs.
- Process cleanup and the broadcast package tests pass, including repeated runs under load.

## Plan

Quick-flow scope: Cloudflared stderr ownership/completion only, expected under100added production lines; no tunnel URL admission changes or larger time budgets. Other flakes have separate seams and remain separate issues in the requested order.

- [x] Establish deterministic RED with the existing fake executable and real pipe: force child wait before draining and require the exact final API-failure line. Test shared stderr setup rather than a probabilistic repeat count; preserve rejection of api.trycloudflare.com as a tunnel URL.
- [x] Replace exec-owned StderrPipe with a parent-owned os.Pipe assigned to cmd.Stderr. Close the parent's write end immediately after successful Start and both ends on failed Start. Keep one scanner owner and close its reader on completion. Publish Exited only after both process wait and stderr drain finish (ARCH-ORDER/DRY).
- [x] Retain the existing five-second Close escalation. On that forced path, kill the existing owned groups and interrupt the owned reader before joining, so an inherited stderr writer cannot hold teardown forever. Normal early exit retains the complete final line; timeout/cancellation keeps bounded cleanup. No larger sleeps/timeouts (ARCH-CONSTRAINTS/FUNERAL).
- [ ] Add controlled drain/completion and forced-reader-release regressions, plus guarded and unguarded fast failures. Use real fake processes and channel-held reads, exact message assertions, and bounded test deadlines only as hang guards. Run targeted/race and repeated loaded broadcast checks, then make -k build and the recorded suite.
- [ ] Update the existing atlas ownership map if needed; close on SHIP and land before starting #586. Generated weave refresh ignore entries for xx-universal-review are bootstrap bookkeeping only.

## Log

### 2026-10-15 — first observed during #571 verification

- Test: `TestQuickTunnelAPIFailureIsNotAURL`, `cmd/internal/broadcast/cloudflared_test.go:434`.
- Exact tested head: `8bd4baa138c63cda357c3a447ddb2cba294f1df7`.
- Recorded suite run ID: `b2dd3d164d5caad0fd4f624661697ade`. At filing it is still finishing other packages; log: `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/b2dd3d164d5caad0fd4f624661697ade/run.log` (failed runs move under `test-stats/failed/<run-id>.log`).
- Failure after 0.13s: `err broadcast: cloudflared exited while opening: no output, want cloudflared's own failure`; broadcast package duration 59.332s.
- Unchanged isolated rerun: `go test ./cmd/internal/broadcast -run '^TestQuickTunnelAPIFailureIsNotAURL$' -count=10` passed, package duration 1.497s; log `/tmp/pair571-cloudflared-check.log`.
- Two earlier complete #571 suites on heads `12223899754159b8aba61548278c2aaacc74df6b` and `ba943894842c638e299efee8bd49be189c92d44d` passed broadcast. These are observations, not a calibrated failure rate.
- Filed unassigned at the TL's direction; keep its fix out of #571. The TL authorized an unchanged recorded-suite rerun on the same head after filing.

### 2026-10-15 — claimed triage and root cause

Claimed on pair:3 and started from landed main4c6fcc8d437ce52c708b0a6bdef09ce42b86917b. start-plan refreshed Ariadne frome055f2a3f7102135beb6bb966fe87f9da33fdd83 to4e5eed55fe801fdd216988d2f9c0c35f3f744018, rebuilt tools, then required rerun; second start-plan created the issue branch and delivered architecture principles. Refresh generated only two tracked ignore entries for newly supplied xx-universal-review skills. Saved their exact patch, temporarily removed only those own generated lines to satisfy clean-rest planning, then reapplied them on this issue branch; no unrelated edit was changed.

Source-confirmed contract violation: Open uses cmd.StderrPipe, starts watchStderr in one goroutine and cmd.Wait in another, then closes h.exited immediately after Wait. Local Go documentation explicitly says Wait closes StderrPipe after process exit and must not run before all reads complete. Thus both premature pipe closure and observing lastLine before the scanner has recorded it are possible. This establishes the ownership defect; a deterministic real-pipe RED is still required before the fix, rather than claiming the historical failure's exact schedule.

Alternatives considered: drain-before-Wait with exec-owned pipe is smaller but can stall process reaping/Close on inherited pipe writers unless additional forced-close coordination is added; cmd.Stderr as a streaming writer lets exec own copying but adds line framing and WaitDelay semantics. Recommend owned os.Pipe plus joined process/drain completion: explicit reader lifetime, unchanged scanner/URL parsing, deterministic wait-before-drain regression, and the existing Close budget supplies forced teardown. No real tunnel/network or live slot will be used. ARCH-PURE: existing URL/guard parsers remain pure; the change is thin process/IO lifecycle glue. ARCH-MOCK: existing fake executable and portable temp storage model the external process. ARCH-SECURE: group ownership and URL refusal remain unchanged. ARCH-PURPOSE: preserve the original stderr error, not just turn the flaky assertion green.

Read-only parallel triage: #586/#591 share identity.lua selection/wait and CLI's2sGit observation budget, but no demonstrated common cause; #589 has controlled evidence of a final asynchronous admission crossing its100ms fixture sleep and a measured channel-barrier candidate; #590 has a separate1sflock admission budget/10ms retry interval and17serialized fsync writers, with no retained trace proving the load contributor. Its lock already uses O_CLOEXEC and explicit LOCK_UN, so inherited lease retention is not established. Keep issues separate in requested order until evidence justifies combining a shared seam. No other issue claimed, source changed, test executed or production slot touched for this triage.

Prior #579 live acceptance is explicitly deferred by hub851b7681-9be7-4078-8664-8df039a458e4 to tomorrow's combined restage/restart (#544/#569/#587/#579); the hub will supply actual wrapper PIDs/build, then observe read-only and record before/after. No live measurement is being claimed now.

### Approved implementation boundary

Canonical hub approval7f23a2b8-bee4-4d38-a7e8-125be7d64f6d approves plan a3d8e90d and explicitly identifies this as a real production bug, not merely a flaky test: concurrent Wait while reading StderrPipe violates the Go contract. Preserve this distinction in close evidence. Approved owned os.Pipe, parent writer close after Start, exited publication after process wait and stderr drain, existing owned-group force teardown, and deterministic wait-before-drain RED. Keep the remaining batch order. No larger sleeps/timeouts or production slots.

### Deterministic RED checkpoint

The real fake-process regression forced cmd.Wait before any stderr read. With the exec-owned StderrPipe setup it failed with lastLine="no output" instead of the exact API failure (package0.334s). Switching only the shared setup to owned os.Pipe preserves that diagnostic. The second lifecycle regression delays the real scanner until the wait owner invokes its drain effect: baseline publication without invoking drain fails deterministically with drained=false last="no output" (package0.700s). This is a production ownership defect, not only a fixture flake. An intervening compile-only failure from adding the second test before its helper is not counted as RED evidence. Implementation is incomplete at this checkpoint: drain join and forced reader cleanup remain.

### Joined ownership and forced cleanup

Owned-pipe and exit/drain regressions passed together (0.365s). Forced-close regression retains a real writer while exercising both a running owned process group and a reaped command. A closed deadline channel triggers the existing escalation event without waiting for wall time; correctness depends on completed process/drain/Close events. Removing only forced reader close makes the reaped case fail its hang guard at10.09s; restoration passed all new focused tests and existing API refusal (0.876s). The test releases its retained writer during cleanup, so the negative control does not leave a process or drainer behind. Process wait completion is tracked separately from drain completion; Close avoids signalling the already-reaped command while waiting for its pipe drain. Parent-owned pipe ends are closed on setup/start failure, and the scanner owns reader disposal. No native slot or network used. Full package race and50focused repetitions are running concurrently as a loaded check; no results claimed yet.

### Targeted verification and recorded-suite admission

Source02b5ef94ee83e1274d7b4ec7912261620e5acfeb: full broadcast package with race detection passed (89.747s); focused ownership/drain/forced-close/guarded+direct fast failure and existing API refusal passed50repetitions (34.463s) concurrently with that package run. make -k -B build passed, rebuilding pair, couch and pair-launch-helper from this exact source. No live broadcast/conformance network test was enabled. Recorded sdlc local-check run --name suite is queued as run5cf351123b0b1e80ce3ea58061fd72b0 behind the machine-wide two-suite cap; it uses the pushed exact source head and no quota override.

## Revisions

### 2026-10-15 — hypothesis confirmed as a production ownership violation

The original Spec states a working hypothesis. Real-process wait-before-read RED now confirms loss of the exact failure through exec-owned StderrPipe, and controlled exit-before-drain RED confirms premature completion publication independently. Scope and Done when remain unchanged: own the pipe, join process and drain, preserve the exact error and API URL refusal, and force reader disposal at the existing teardown boundary. This fixes production lifecycle behavior as requested by the hub, rather than only stabilizing a test.
