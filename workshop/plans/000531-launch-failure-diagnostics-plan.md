# Durable Couch launch failures Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve failed target launches and their enclosing switch errors so the next intermittent failure can be diagnosed after rollback or restart.

**Architecture:** The shared launch path samples bounded runner output and observed exit state before cleanup, saves an attempt record, then enriches that same record with cleanup and the enclosing switch error. A separate bounded namespace store owns this evidence; thread lifecycle records and launch admission remain unchanged. Existing PTY capture, atomic file writes and terminal-text sanitization remain their respective authorities (ARCH-DRY).

**Tech Stack:** Go, existing Runner/FakeRunner, ptychild.Ring, namespace-local JSON and file locks, couch --show.

---

## Approved scope and open approval

The operator's successful ariadne:5 switch with exactly `--sandbox danger-full-access` supersedes the requested live reproduction. No further live repro, startup-policy changes, retries, or work on #529. This diagnostic design awaits hub approval before change-code. One atomic close boundary; tasks below are implementation steps, not milestones.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| LaunchFailure | cmd/internal/couchcore/launchfailure.go | new |
| LaunchFailureHistory | cmd/internal/couchcore/launchfailure.go | new |
| ShowResult | cmd/internal/couchcore/slotreport.go | modified |

**LaunchFailure** identifies one attempt by scope, tag and nonce, with starting path, target agent, operation, first failure time, stage, raw bounded output and accurately labeled stream. It carries launch error, final operation error, and separate exit observations before/after cleanup. An absent exit code means unobserved, never success. Output relates 1:1 to the exact launched handle, not to whichever handle currently occupies the address. Future extensions add evidence fields here, not to ThreadRecord.

**LaunchFailureHistory** owns at most 16 attempts, sorted by first failure time and stable attempt key. Upsert merges only matching keys and retains the original failure time. Late updates cannot displace newer attempts; an already-evicted attempt is subject to the same oldest-first policy. Output is a 16 KiB byte tail, each of launch/final/cleanup error is bounded to 4 KiB with an explicit truncation marker. Metadata fields are validated/bounded; reject oversized identity rather than silently change its key. Encode output as bytes so invalid UTF-8 is preserved. Serialized history cannot exceed 1 MiB; reject an oversized encoding without replacing the previous file. Bounds and merge rules have pure tests without IO mocks.

**ShowResult** adds matching diagnostic records and a diagnostic-read-error field independent of thread/slot lifecycle observations. Exact scope/tag and canonical starting path associate evidence with the requested thread or slot; never match tags globally across repositories. Slot inspection must work after provisional thread rollback. A successful launch leaves failure history intact.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| LaunchOutputHandle | cmd/internal/couchcore/launchoutput.go | new | bounded process output snapshots |
| ExecRunner and acknowledgedHandle | cmd/internal/couchcore/runner.go; cmd/internal/couchcore/launchhelper.go | modified | ordinary child stderr and blocked-start wrapper |
| PtyRunner and FakeRunner | cmd/internal/couchcore/ptyrunner.go; cmd/internal/couchcore/runner_fake.go | modified | existing terminal snapshots and stateful fake child |
| LaunchFailureStore | cmd/internal/couchcore/launchfailure_store.go | new | namespace JSON, lock and atomic replacement |
| shared failure recorder | cmd/internal/couchcore/launch_existing.go; cmd/internal/couchcore/switchagent.go | modified | launch cleanup and enclosing switch error |
| diagnostic presentation | cmd/internal/couchcore/operationdispatch.go; cmd/internal/couchcmd/run.go | modified | show command and readable terminal output |

The optional output interface returns a cloned bounded snapshot and stream label. Nonimplementing handles explicitly report output unavailable. The acknowledged wrapper forwards this capability; it must not hide it. Ordinary stderr is forwarded while a mutex-protected ring retains its tail. Stdout/stdin retain their current routing. PTY output stays combined and is read from the existing Child.Snapshot rather than a second pump. Tests must measure ordinary stderr forwarding and any tty-detection change: a tee uses a pipe for stderr, so document this limitation and verify supported launch behavior rather than claim identical file-descriptor semantics.

LaunchFailureStore is injected at Couch composition in cmd/internal/couchcore/couch.go, with a stateful in-memory fake for launch tests. Production storage is `<namespace>/launch-diagnostics/failures.json`, mode 0600 under a 0700 directory. Use a separate lock in that directory, existing locking/atomic-write primitives, bounded reads, regular-file/no-symlink checks and atomic replacement. Never hold a thread-store lock while taking the diagnostic lock. Lock acquisition retries nonblocking with a bounded independent one-second context so a canceled operation can still record but cannot hang shutdown indefinitely. Corruption/write failure preserves the old file and surfaces alongside the original error. Normal restart reads the saved file; no background queue, global registry or lifecycle-schema change.

The stateful fake keeps history across recorder construction, supports injected read/write failure, and captures write ordering against cleanup. Existing FakeRunner supplies actual child state transitions and output; no function-call mock of a successful launch.

## Failure protocol

1. Once an accepted operation has a valid address, allocate/retain its attempt identity. For failures before a process exists, record the stage and error with helper-not-started and no fabricated output or exit. Invalid unaccepted previews remain ordinary validation errors, not launch attempts.
2. On a shared launch failure, snapshot the exact handle. Poll Exited without blocking; only call Wait after it is closed. Save the failure before destructive cleanup. The existing error chain remains reachable through errors.Is/As.
3. Run existing cleanup unchanged. Sample again and enrich the same record: an exit first observed after cleanup is labeled after-cleanup, not asserted to be caused by the target. Retain the original pre-cleanup evidence.
4. The enclosing SwitchAgent error enriches that attempt with the final text including the source-park/recovery explanation. A typed internal error carries the attempt key through wrappers so no lookup by latest failure is needed. Persist errors join the original error and include the diagnostic path/attempt key; failed persistence must not turn failure into success or suppress cleanup.
5. Immediate error text gives sanitized bounded evidence (maximum 2 KiB excerpt) and the diagnostic location. Show returns the retained records, newest first; human output sanitizes each line using rowtext.Sanitize, preserving safe line breaks. JSON retains structured fields and normal JSON escaping.

This preserves failures that Couch observes, including graceful cancellation. It cannot manufacture stderr from an unstarted helper, observe an exit not yet reported by the runner, or guarantee saving after abrupt process kill/power loss before the failure handler runs. Do not diagnose the historical Ariadne failure from absence of evidence.

## Task 1: Bounded evidence and storage

**Files:** create `cmd/internal/couchcore/launchfailure.go`, `launchfailure_test.go`, `launchfailure_store.go`, `launchfailure_store_test.go`, `launchfailure_store_fake.go`; modify `cmd/internal/couchcore/couch.go`.

- [ ] Write failing pure tests for byte-tail bounds/invalid UTF-8, separate pre/post-cleanup exit observations, duplicate enrichment, reverse-order concurrent attempts, eviction and encoded-size ceiling. Use 17 attempts to prove retention independently of implementation constants.
- [ ] Run `go test -p 1 ./cmd/internal/couchcore -run '^TestLaunchFailure' -count=1`; expect missing types/behavior.
- [ ] Implement the record and pure history transformation with the contracts above.
- [ ] Write store integration tests for fresh-store round trip, reopened store after rollback, concurrent independent writers, interrupted atomic replacement preserving the old record, corrupt/oversized/nonregular files, and lock timeout. Inject failures through the store seam; use temporary directories only.
- [ ] Implement store and fake, initialize the production default in Couch.New, and rerun the same targeted command until green.
- [ ] Commit `#531: retain bounded attempt-scoped launch evidence` with model co-author trailer.

## Task 2: Runner output and shared failure capture

**Files:** create `cmd/internal/couchcore/launchoutput.go`, `launchoutput_test.go`, `launchfailure_capture.go`, `launchfailure_capture_test.go`; modify `runner.go`, `ptyrunner.go`, `launchhelper.go`, `runner_fake.go`, `launch_existing.go`, `switchagent.go`, `switchagent_test.go` in that directory.

- [ ] Write real subprocess tests: print known stderr then exit 23; ordinary stderr is forwarded and retained, PTY is labeled combined, blocked acknowledgement preserves the capability, and a child with no observed exit never invokes blocking Wait. Assert truncation after output beyond 16 KiB. Use short-lived test helpers, no visible terminals or live agents.
- [ ] Run `go test -p 1 ./cmd/internal/couchcore -run 'Test(LaunchOutput|LaunchFailureCapture)' -count=1`; expect missing capture behavior.
- [ ] Add the optional output interface and wrapper forwarding, reusing ptychild.Ring/Child.Snapshot. Cover output draining and child reaping; no new unbounded channel or whole-session transcript.
- [ ] Add stateful fake tests for spawn refusal, acknowledgement failure, registration child exit, timeout/cancellation while alive, cancellation during cleanup, cleanup error, store failure, and successful registration. Assert pre-cleanup write occurs before cancellation and the source remains parked/default preferences unchanged on switch failure.
- [ ] Implement shared capture and typed attempt error, then enrich with SwitchAgent's final error. Audit every return in launchTrackedThread and accepted SwitchAgent execution so pre-spawn failures are not omitted. Keep PrepareAgentSwitch side-effect free.
- [ ] Run `go test -p 1 ./cmd/internal/couchcore -run 'Test(LaunchOutput|LaunchFailure|SwitchAgent|UncertainSwitch|ExecRunner|PtyRunner|RealRunners)' -count=1`; expect PASS, exact final switch error and child exit 23 survive a new store instance.
- [ ] Commit `#531: capture launch output and preserve final switch errors` with model co-author trailer.

## Task 3: Inspection and verification

**Files:** modify `cmd/internal/couchcore/slotreport.go`, `operationdispatch.go`; create `cmd/internal/couchcore/launchfailure_show_test.go`; modify `cmd/internal/couchcmd/run.go`; create `cmd/internal/couchcmd/launchfailure_test.go`; update `README.md`, `atlas/couch.md`, `atlas/index.md` and the issue Log.

- [ ] Write failing show tests for exact address, scope collision, slot without a remaining thread, later successful launch retaining old evidence, and store-read error distinct from missing evidence. Rendering tests cover ANSI, OSC, C1 and invalid UTF-8, exit-unobserved wording, PTY stream label and bounded excerpts.
- [ ] Run `go test -p 1 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'TestLaunchFailure' -count=1`; expect missing show evidence.
- [ ] Implement diagnostic-only resolution and rendering; reuse rowtext sanitization. Avoid changing the authoritative thread-reference resolver to make a historical failed attempt look like a live thread.
- [ ] Update README inspection instructions and atlas storage/flow mapping, including retention, stderr versus PTY labels, normal success retention, tty-detection limitation and abrupt-kill limits.
- [ ] Run the targeted command and `go test -p 1 ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'Test(LaunchFailure|LaunchOutput|SwitchAgent)' -race -count=1`; expect PASS. Check `git diff --check`.
- [ ] Record tests, operator non-reproduction and actual limits in issue Log. Check machine load before any full suite; while load is >=40, report the pending full check to hub and do not start it. When below 40 use the repository's gated full-suite command, at most two machine-wide. No live smoke slot changes without fresh hub authorization.
- [ ] Commit final code/docs; merge current main and resolve only this scope. Run required checks, then `sdlc close --issue 531 --verified '<measured evidence>'`; the gate owns the fresh-eyes review. Address findings, create PR via sdlc, land on SHIP via `sdlc merge --yes` with validation enabled, report real commit/PR to hub and stop.

## Revisions

### 2026-10-13 — initial diagnostic-only plan

The issue Log retains the investigation and successful operator repro. This plan starts after the scope narrowing; there is no live-reproduction task or speculative Codex argument change.
