---
collaboration-mode: discussion
---

# Preserve Darwin PTY final output — implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Use bounded subagents for independent exploration; keep the shared launch/lifetime change in one implementation context. Do not implement before TL approval and change-code.

**Goal:** Preserve bytes already written by a short-lived PTY child when its own slave descriptors close before the parent first reads.

**Architecture:** Retain one parent-side slave descriptor on Darwin until the existing output reader finishes, then release it before reaping. A shared launch adapter preserves creack/pty's command/session/geometry semantics and returns the ordinary master plus an independently owned slave lease. All three production launch owners adopt it; non-Darwin keeps the upstream immediate-slave-close behavior.

**Tech stack:** Go, pinned creack/pty v1.1.24, Darwin PTYs, existing ttyio/terminal adapters, native subprocess fixtures.

**Flow:** Recommend full flow because the shared starter plus three lifetime integrations will exceed 100 production lines. One atomic issue boundary, no Mx split. TL approval of this plan includes that flow choice; run change-code with --flow full after approval. Do not estimate until its plan-quality gate admits the design.

## Evidence and decision

The issue's original failure is not a reader retry or cmd.Wait ordering defect: scribe already reads to EOF before waiting. The actual invalidating event is loss of the last slave reference before that first read.

A native planning probe on this host used openpty/fork, child setsid/TIOCSCTTY, a direct write of final-marker, closure of descriptors0/1/2, then an out-of-band pipe acknowledgement. Only after the acknowledgement did the parent read the master. Results: baseline empty; child-side /dev/tty open/close preserved the marker; retained parent slave also preserved the marker, reached EOF while that slave was still held, and returned child exit7. A second probe attempted TIOCNOTTY on the slave: Darwin refused it (errno25), and retained-slave output/EOF/exit still completed. These are feasibility observations, not the final automated regression.

Primary sources: [Ruby20682](https://bugs.ruby-lang.org/issues/20682) describes output loss and the child-tty workaround's wait-before-read hang. Apple [cttyopen](https://github.com/apple-oss-distributions/xnu/blob/aca3beaa3dfbd42498b42c5e5ce20a938e6554e5/bsd/kern/tty_tty.c) records the session reference, and [proc_exit](https://github.com/apple-oss-distributions/xnu/blob/aca3beaa3dfbd42498b42c5e5ce20a938e6554e5/bsd/kern/kern_exit.c) closes ordinary descriptors before the session-leader terminal drain/revoke. Retaining the slave keeps the queued output available through that sequence. The automated test must verify EOF rather than assume it from this source reading.

Options considered:

| Option | Benefit | Cost / reason |
|---|---|---|
| Darwin retained slave (recommended) | Preserves direct exec PID, argv, PATH lookup, startup errors, environment, inherited descriptors and native-lock inspection | Adds one fd per live Darwin PTY, with mandatory drain/abort release; requires native EOF and teardown qualification |
| Child /dev/tty exec trampoline | Establishes the kernel's child-session reference before target exec | New executable route or helper dependency; arbitrary callers include Couch/test binaries, so os.Executable alone is insufficient; preserving synchronous exec failure needs another startup handshake; cannot unconditionally change Couch's helper because non-PTY ExecRunner shares it |
| Read/wait delays or retry after EOF | Small patch | Does not preserve the reference; rejected as incapable of recovering discarded bytes |

If the native retained-descriptor tests cannot prove EOF and cleanup for the cases below, stop and return the evidence to the TL. Do not patch around a hang with a new drain deadline or silently fall back to the trampoline.

## Core concepts

### Pure core

| Name | Lives in | Status |
|---|---|---|
| Platform slave-retention policy | cmd/internal/ptystart/policy_darwin.go; policy_other.go | new compile-time policy: retain only on Darwin |

No new user lifecycle, protocol, durable record or parser is introduced. The existing terminal/output and child-exit values remain authoritative. Policy tests assert the platform branch; no new policy framework is needed.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Start(cmd, size) | cmd/internal/ptystart/start.go | new | pty.Open, pty.Setsize and exec.Cmd.Start |
| SlaveLease.Close | cmd/internal/ptystart/lease.go | new | one retained slave descriptor, idempotent release |
| Scribe output/child owner | cmd/internal/scribecmd/scribecmd.go | modified | stdout/log byte pump and direct child wait |
| Wrapper child owner | cmd/internal/wrapcmd/wrap.go | modified | existing startNativePTY admission, master pump and ownedGroup |
| Terminal child owner | cmd/internal/ptychild/child.go | modified | existing transport, pump, publication and reaping |

Proposed internal API:

```go
func Start(cmd *exec.Cmd, size *pty.Winsize) (master *os.File, lease *SlaveLease, err error)
func (l *SlaveLease) Close() error
```

The lease owns only the extra slave; existing transport/caller ownership of the master stays intact. Close is concurrency-safe and idempotent, retaining its error. Non-Darwin returns a settled no-op lease after closing the parent slave at successful start as upstream does. No lease descriptor is added to ExtraFiles or inherited by the target beyond its ordinary stdio.

Start follows the pinned upstream sequence: open master/slave; apply initial size before spawn; fill only nil stdio fields; retain existing SysProcAttr fields while setting Setsid/Setctty; call the original cmd.Start. Preserve cmd.Path/Err/Args, cwd, environment and ExtraFiles unchanged. Failed open/resize/start closes all acquired descriptors and returns the original wrapped error without a live child. Reuse pty.Open and Setsize; do not copy platform PTY allocation code.

## Ownership and transitions

| Owner state / event | Effect and next state |
|---|---|
| Unopened -> open failure | Return error, no descriptors or child |
| Master+slave opened -> resize/start failure | Close both, no live child; return error |
| Successful Darwin start | Existing owner holds master and one slave lease; reader drains continuously |
| Successful non-Darwin start | Parent slave closes immediately; existing reader behavior retained |
| Reader observes terminal EOF / terminal read failure | Preserve n>0 bytes first; release slave, then existing owner reaps exactly once |
| Initialization abort, cancellation or explicit Close | Stop/join the owned child as existing caller requires; close/cancel master to unblock kernel drain, release slave exactly once, join pump/wait before owner returns |
| Repeated cleanup or concurrent pump/Close | Lease close is idempotent; retain existing one-reaper authority, no extra waiter |

One resource, held by the existing child owner for its lifetime, prevents premature last-slave-close. There is no exit polling, additional wait goroutine, retry, new timeout or background collector. Scribe must also join a successfully started child on a later initialization error; use one wait authority on both normal drain and abort, rather than adding a competing cmd.Wait. Its pre-existing stdin-copy behavior is outside this output-lifetime change; do not redesign arbitrary io.Reader cancellation here.

### User-visible state table

Actions: run a command; type/resize; interrupt/stop; inspect an error; explicitly run again.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Start succeeds; streaming or final drain | Same terminal/stdout output and log; command still active until drained/reaped | Type/resize or interrupt/stop | None from ordinary final close |
| Open/resize/exec fails before target starts | Existing launch error and status | Inspect error, explicitly run again | No target work ran |
| Parent/transport aborts while target runs | Existing stop/failure behavior | Inspect, explicitly run again | Unread output can be abandoned by the intentional abort, as before |
| Child returns nonzero or dies from a signal | Existing caller-specific exit result after output drain | Inspect result | Same command failure/interruption as before |
| Slave release reports an error | Existing cleanup diagnostic / Close error surface; child status remains attributable to the child | Inspect error | No silent success claim for unfinished cleanup |

No automatic replay is safe or added. Abrupt SIGKILL of the wrapper cannot promise a completed log; the kernel closes its owned descriptors. No new user-facing choices are required.

## Coverage audit and constraints

Production launch owners are exactly scribecmd.Run, wrapcmd.runProxy/startNativePTY, and ptychild.Start. The latter covers Couch PtyRunner (including acknowledgement-gated launch-helper chains) and termcmd. All three are in this plan, including generic one-shot wrap/terminal commands and final shell exec. Couch non-PTY ExecRunner is unchanged. Probe programs, test-only PTY hosts and pairlifecycletest are not production owners and are not claimed migrated.

ARCH-DRY: one start adapter and slave lease for all three production owners. Wrap must retain native-selector inspection on original cmd.Args before its injected starter runs; use a closure to retain the returned lease rather than rewriting arguments or changing the selector protocol. ARCH-PURE/MOCK: lifecycle/lease tests use stateful closeable resources that retain open/closed ownership state; real PTYs qualify kernel behavior. Existing Child fake and publication tests continue to qualify consumer state.

ARCH-CONSTRAINTS: one additional fd per live Darwin PTY, O(1) launch/cleanup work and no work added per output byte. Existing 4096-byte pumping and backpressure stay intact. At N=8 and2N=16 launches, count one retained slave per successful Darwin launch and exactly one release per owner, zero after join; failed starts retain zero. Count resources rather than compare timings. No new command deadline or output-size bound.

ARCH-STATE/FUNERAL: the slave lease is created at start, its child owner is its last consumer, and drain/abort closes it. Master/process cleanup remains with existing owners. No durable artifact or orphan worker is created. ARCH-SECURE: preserve argv as a slice, environment/cwd and ExtraFiles positions; no shell interpolation, executable lookup changes or new inherited fd. ARCH-EVOLVE: no persisted format changes. ARCH-PURPOSE: cover every audited production PTY starter; document any newly found owner before expanding scope.

## Tasks (single issue boundary)

### 1. Pin the native failure and shared lifetime

Files: new cmd/internal/ptystart/start.go, lease.go, policy_darwin.go, policy_other.go and corresponding tests; native writer fixture under that package's testdata if an executable fixture is needed.

- [ ] Add a native fixture that writes exact bytes, closes its own slave stdio, and acknowledges that closure through an out-of-band pipe before the first master read is allowed. It must not open /dev/tty or inherit a protective extra slave. No elapsed sleep establishes ordering.
- [ ] Force the old upstream start path through that barrier on Darwin; require the output-loss regression to fail. The test's safety timeout only collects diagnostics and cleans up its own PID/master; it is not the oracle.
- [ ] Add retained-lease tests: exact bytes, EOF while held, drain-before-wait, exit0/nonzero, signal death, empty output, output larger than the PTY queue, argv boundaries, cwd/env, initial geometry, unchanged ExtraFiles numbering and synchronous missing/non-executable errors.
- [ ] Implement the minimal starter and lease; cover failed open/resize/start and repeated/concurrent close with stateful resource fakes and native conformance. Establish no master/slave/process leaks at N/2N.
- [ ] Run go test ./cmd/internal/ptystart -count=1 and targeted race tests. Restore immediate slave close in a disposable mutation and require the final-byte oracle RED; restore normal source.

### 2. Adopt all production owners without replacing their lifecycles

Files: cmd/internal/scribecmd/scribecmd.go and tests; cmd/internal/wrapcmd/wrap.go and owned/native-lock tests; cmd/internal/ptychild/child.go and child/publication tests.

- [ ] Add owner-level regressions that gate the first master read through the same native writer. Scribe asserts exact stdout and log, not substring presence. Wrap asserts final capture bytes; ptychild asserts endpoint/raw publication before Exited/Wait completes.
- [ ] Adopt the starter in scribe; release the slave after the read loop and before its single wait. Ensure post-start initialization failure closes the master/lease and joins the child. Keep pause/resume, resize, raw-mode restoration and child status semantics.
- [ ] Adopt via startNativePTY's existing callback after native admission; retain lease cleanup in the existing wrapper scope, with master close and ownedGroup joining on every early return. Preserve watcher/PID/session ownership and prompt argv.
- [ ] Adopt in ptychild; release on pump completion before reaping and integrate idempotently with Close/initialization-failure cleanup. Keep publication flush and endpoint completion ordering.
- [ ] Run focused output, exit, interactive input/resize, terminal-generated interrupt, SIGTERM/SIGHUP, descendant cleanup and startup-failure tests. Assert no new waiter, no sleep-based retry and no double reap. Retain Couch helper acknowledgement/no-ack/timeout and non-PTY ExecRunner tests.

### 3. Prevent drift and qualify public behavior

Files: docs/pair-scribe.md, relevant README/atlas terminal ownership entries, internal/repolint local catalog if a direct-start guard is appropriate, scripts/test-affected.py and per-reader certificates for new/changed scanners.

- [ ] Add a Pair-local golint catalog rule, using the existing local-catalog composition, that rejects direct production creack/pty Start/StartWithSize/StartWithAttrs outside the shared owner; enumerate probe/test exclusions explicitly and prove a seeded bypass fails. Do not edit upstream Ariadne on this issue branch.
- [ ] Audit/register any new repository-reading scanner and regenerate only its explicit certificate with make affected-readers. Keep every fixed scanner registered in SOURCE_GUARDS.
- [ ] Document Darwin final-output preservation, lease ownership, covered routes, abort limits and unchanged non-Darwin behavior. Correct stale standalone scribe/wrap binary comments encountered at these touched launch sites.
- [ ] Run targeted package/race/vet and make test-affected-inventory. Exercise the public built pair scribe command with the deterministic native writer, plus normal interactive shell input/resize/interrupt and cleanup. Coordinate staging/live check with the hub before touching pair:0.
- [ ] Push PR, run the recorded full suite on the pushed final code, close through SDLC's review, then enter merge. Any integrated code needs its required exact-head suite; never waive CI. Report landing to the hub.

## Acceptance clarification

The initial Done when says the child exits before the first read. A protected Darwin child can be inside exit waiting for the parent to drain; waiting for waitpid before releasing the read barrier would create the documented deadlock. Use the actual failing prerequisite instead: prove the child finished its writes and closed all its own slave descriptors before the first master read, then drain and reap. Preserve the exit code and prove EOF. This clarification is part of the requested TL approval, not an implementation relaxation or new timeout.

## Approval checkpoint

Plan only. No production code or tests have been edited. Await TL approval of the retained-slave choice, all-three-owner scope, acceptance clarification and full-flow choice before change-code.
