# Release-aware workspace lease retry Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. This is one atomic review boundary; use the executing-plans skill for session-warm implementation.

**Goal:** Let a workspace creator wait out a released lease's fork-to-exec copies without weakening live producer exclusion.

**Architecture:** Keep the existing close-only flock as the sole admission authority. Store one advisory byte in the existing lock inode: held after acquisition, released before Close. Only released evidence permits a longer bounded backoff, with one absolute deadline; unknown/legacy or active ownership retains the current short refusal.

**Tech Stack:** Go, x/sys/unix flock, standalone C test fixture for a real paused fork.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| creationLeaseRetry | cmd/internal/couchcore/provision_lock_unix.go | new |

A small value computes whether to retry and the next delay from elapsed time and the currently observed release hint. The short budget remains 100ms; released evidence permits up to 2s total from the initial attempt. Backoff begins at 2ms and caps at 20ms, clamped to the remaining budget. A released observation never grants the lock, and observing a new held/unknown marker returns to the short budget. The absolute start never changes. Pure tests cover exact boundary, active/unknown refusal, release then held, and repeated release observations. No persistent cache or queue exists (ARCH-PURE, ARCH-CONSTRAINTS).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| HostCreationLease | cmd/internal/couchcore/provision_lock_unix.go | modified | existing secure inode open, flock, advisory byte, descriptor Close |
| paused fork fixture | cmd/internal/couchcore/testdata/creation_fork.c | new | native fork, pipes, CLOEXEC, exec and waitpid |

The lease owns the descriptor. WriteAt held after successful flock; a failed write closes and returns the error. Close writes released before closing and joins both errors. ReadAt reads exactly one byte after a busy flock; missing, malformed or unreadable hints are unknown. The flock is always retried before proceeding, even after a released observation. No LOCK_UN is introduced. No fsync: the hint is live retry advice, not durable authority. The existing lock file is permanent and grows by at most one byte; no sidecar or cleanup lifecycle is added (ARCH-DRY, ARCH-FUNERAL).

The native fixture receives the lease as fd3, sets FD_CLOEXEC, forks a child paused on a control pipe, and closes its own fd3. Readiness proves the child is paused and parent copy closed. Go closes its lease, attempts acquisition while the child remains paused beyond the former 100ms budget, then tells the child to exec /usr/bin/true. The helper reaps the child and exits; test cleanup closes pipes and kills/reaps the helper on failure. Compile once per test process with cc in a temporary directory; skip this conformance fixture if cc is unavailable, while ordinary Go policy/descriptor tests always run. No fork occurs inside the Go runtime (ARCH-MOCK: real kernel conformance; pure retry model supplies deterministic boundary coverage).

## State and safety contract

| State/event | Effect |
|---|---|
| flock acquired | mark held; return lease only after successful mark |
| busy + held/unknown | retry only within the original 100ms budget, then Busy |
| busy + released | back off within the original 2s budget, then Busy |
| marker changes during retry | reread on each busy result; never reset start/deadline |
| owner Close | mark released, close descriptor even on write error |
| owner dies without Close | kernel descriptors remain authoritative; no new dead-owner heuristic |
| live inherited producer after Close/death | never admitted until actual flock succeeds; Busy at bound |
| stale released hint during new owner's mark | may add bounded latency; cannot admit a second creator |

There is no task queue, reservation, renumbering, lock stealing, or background retry. The caller gets the existing Busy error if the bound expires. The 2s release budget is a bounded UX allowance for scheduling under load, not a guarantee against arbitrary process suspension. Natural failure rates and forced paused-child rates must be reported separately. Earlier 2–9ms observations cannot establish the worst case (ARCH-STATE, ARCH-PURPOSE).

## Tasks

- [ ] Add pure retry boundary tests and real-descriptor marker tests in provision_lock_test.go. Retain alias, inheritance and parent-death tests. Replace the old transient-copy test's private 30s override with the production policy. Cover release then held, EOF/malformed marker, held-write/release-write errors, and released live producer exhaustion.
- [ ] Add the native paused-fork fixture and regression. Run against unchanged production code first to demonstrate failure beyond 100ms. Keep its kernel ordering distinct from the pure clock boundary oracle.
- [ ] Implement the one-byte lifecycle and bounded retry in provision_lock_unix.go; remove the test-only settle override if unused. Run targeted tests, including provision cancellation and recovery-marker tests, then commit.
- [ ] Build baseline and fixed test binaries using a scratch overlay for the baseline source. Interleave at least 50 repetitions per variant of the eight historically affected cases alongside a full make -k test run. Also record at least 50 forced paused-child attempts per variant. Preserve complete logs, exact test/subtest denominators, load samples and failures. Run binaries from cmd/internal/couchcore so relative inventory tests work. Do not claim an improved natural failure rate if baseline natural runs remain green.
- [ ] Update atlas/workspace-provisioning.md's contention contract, issue evidence and plan checkboxes. Run make -k test to completion, cite known failures, close through SDLC review, merge main, create PR and land with sdlc merge --yes after SHIP/validation; report hub and stop.

## Scope and principles

The descriptor-open sweep found creation already uses O_CLOEXEC. Go os.OpenFile opens are close-on-exec too. Raw missing flags in diagnosticlog.openRegular and notifytransport.lockNamespace are independent descriptor hygiene; both lock paths explicitly unlock. Report them to TL rather than broaden this ownership fix. No slot attach/resume state or UI changes. ARCH-DRY keeps metadata in the existing inode; ARCH-PURE isolates retry decisions; ARCH-MOCK uses real kernel conformance; ARCH-CONSTRAINTS bounds all retry and fixture lifetimes; ARCH-STATE specifies owner turnover and errors; ARCH-FUNERAL creates no growing record; ARCH-PURPOSE requires forced and natural-load evidence. No new abstract helper process is introduced in production.

## Revisions

### 2026-10-14 — fixture cleanup and retry representation

Compile the tiny native fixture per invocation in t.TempDir instead of sharing
a process-global temporary executable: this gives it an existing explicit cleanup
owner without editing the package TestMain. The extra compile cost is small and
reported separately from acquisition. Cancellation kills the helper process group
so a child retaining stdout cannot strand readiness. creationLeaseRetry is a pure
function of elapsed time, attempt count and hint rather than a mutable value.
