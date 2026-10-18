# Native agent exit notice Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a slot's current native agent actually exits on its own, the hub gets exactly one lifecycle notice. Attach-client exit, detach, reconnect, wrapper replacement, a SIGUSR2 restart, a park or teardown stop, and a stale or duplicate report never produce one.

**Architecture:** The wrapper owns the native child, so it is the only process that can prove the child exited.
- After `group.Wait()` reaps a child that exited by itself, and while its registry session is still open, the wrapper sends one authenticated broker RPC, `native-exit`.
- The design copies the shape of the existing `automatic-input-stall` RPC: the same caller resolution, the same `authority.current` check, the same connected-session recheck, and an ID derived from the exact binding.
- Couch fences the report to the current child by launch ordinal, dedupes it, and feeds one `exited` fact to #574's lifecycle publisher.
- No new terminal frame and no hello version are needed.

**Tech Stack:** Go (`wrapcmd`, `couchmessage`, `couchcmd`, `couchcore`).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `NativeExitReport` / `NativeExitID` | `cmd/internal/couchmessage/native_exit.go` | new |
| `ValidateRequest` case `native-exit` | `cmd/internal/couchmessage/protocol.go` | modified |
| `LifecycleExited` kind | `cmd/internal/couchcore/lifecycle_notice.go` | modified |
| `childExitCause` (self / restart / stopped) | `cmd/internal/wrapcmd/native_exit_report.go` | new |

- **NativeExitReport** has the fields `{LaunchOrdinal uint64, ExitCode int, AgentIdentity string}`, and `NativeExitID(binding, ordinal)` is a sha256 over the exact binding plus the ordinal.
  - **DRY:** it mirrors `AutomaticInputStallID`, so one helper derives the ID; it doesn't add a second scheme.
  - **Growth:** a signal number or core-dump flag can be added later as fields.
- **childExitCause** is pure. Its inputs are `restartFresh`, the `stoppedByWrapper` flag and the wait result, and its output is `self`, `restart` or `stopped`. Only `self` reports.
  - **ARCH-PURPOSE:** a park or teardown signal and a SIGUSR2 restart are wrapper decisions, and the hub already hears about them through `parked` and `relaunched`. The new notice is for the agent leaving on its own.
- **LifecycleExited** is one more closed `LifecycleNoticeKind`. Its text is "the agent exited (code N)".
  - **Coalescing:** keyed per slot as today, so a later fact replaces an unsent earlier one. `validLifecycleNoticeFact` gains the kind.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| wrapper `reportNativeExit` | `cmd/internal/wrapcmd/native_exit_report.go` | new | `couchmessage.Call` (broker socket) |
| `stoppedByWrapper` flag | `cmd/internal/wrapcmd/native_lock.go` (`stopForSignal`) | modified | signal handling |
| Couch `nativeExit` handler | `cmd/internal/couchcmd/native_exit.go` | new | broker caller, `messageAuthority`, lifecycle publisher |
| `messageService.handle` dispatch | `cmd/internal/couchcmd/message_service.go` | modified | — |

- **Wrapper (ARCH-ORDER):** in `run`, right after `werr := group.Wait()` (`wrap.go:3089`) and before the deferred `peerRun.Close()`:
  1. If `childExitCause(...) == self` and a peer runtime exists (a registry session is open), call `reportNativeExit` once, with a 2 s deadline.
  2. Errors are traced (`traceWrap("native-exit-report", …)`) and never change the wrapper's exit code or delay it beyond the deadline.
  3. **Compatibility:** an old Couch answers `invalid-request` ("unknown message operation"), and the wrapper treats that as "not supported" and stops.
- **stoppedByWrapper:** `stopForSignal` sets an atomic flag before it stops the group (SIGTERM, SIGINT, SIGHUP, SIGQUIT and USR2). A child that exits before any stop leaves it false.
- **Couch handler**, in this order (**ARCH-SECURE**, ARCH-ORDER):
  1. Resolve the caller with `broker.Caller(scope, tag, session, nonce)`, then `authority.current(caller)`. This is the kernel-PID-authenticated, admitted wrapper; the same path the stall uses (`switchCaller`).
  2. Require `ID == NativeExitID(binding, ordinal)`.
  3. Require `LaunchOrdinal` to equal the slot's current launch ordinal (the ready record that `messageAuthority` reads), and `AgentIdentity` to equal the record's identity. A stale child, or the child from before a SIGUSR2 restart, is refused.
  4. Recheck `isConnected(caller)` under the lock. A replaced or closed session is refused.
  5. Dedupe per `(binding, ordinal)`. A repeat answers `ok` (idempotent) and posts nothing.
  6. Feed `LifecycleExited` to the lifecycle publisher with `Incarnation` set to the binding key plus the ordinal, so the publisher's routing, hub fencing and restart barrier apply unchanged.
  7. Gated by `COUCH_LIFECYCLE_NOTICES`, like every lifecycle notice.

  The reply codes are `ok`, `stale` (ordinal or identity mismatch), `not-current` (the caller isn't current or connected) and `invalid-request`.
- **Scope limit (stated):** agents that aren't peer receivers (agy, muse, grok) have no registry session, so they have no authenticated caller and produce no exit notice. The issue's purpose is not to infer an exit, so they stay silent rather than guessed at. Covering them is a follow-up if wanted.

**Test surface:**
- **Pure:** `childExitCause` table; `NativeExitID` determinism and binding sensitivity; the lifecycle reducer with the new kind (coalescing, validation).
- **Integration, stateful fakes** (the `messageWorld`/`serviceRig` rig and its `controlledEndpoint`, no mocks):
  - A current report produces one hub notice.
  - Duplicates post nothing more.
  - A stale ordinal is refused.
  - After wrapper replacement (a new token), the old report is refused.
  - A report from a closed session is refused.
  - Detach, reattach and Couch restart produce no notice.
  - An attach-client exit produces no notice.
- **Compatibility:** a new wrapper against an old Couch (`invalid-request`) exits with the child's code and no delay past the deadline. An old wrapper produces no notice and no error.
- **Owned-child conformance:** a real wrapper `run` with a real native child (`sh -c 'exit 3'`), a real broker socket, and a recording Couch handler.
  - The RPC arrives after the child is reaped, carrying exit code 3 and the ordinal, while the session is still connected.
  - A SIGHUP-stopped child, and a SIGUSR2 fresh restart, send none.

## Chunk 1: single pass (one close)

- [ ] Wire `NativeExitReport`, `NativeExitID` and the `native-exit` op into `couchmessage`, with protocol validation tests.
- [ ] Add the `LifecycleExited` kind to the reducer and validation, with reducer tests.
- [ ] Couch `nativeExit` handler plus dispatch, with stateful rig tests: current, duplicate, stale ordinal, replaced, closed, detach/reattach/restart, attach exit.
- [ ] Wrapper: `childExitCause`, the `stoppedByWrapper` flag and `reportNativeExit` after `group.Wait()`, with unit tests and the compatibility test against an old broker.
- [ ] Owned-child conformance test: real wrapper, real child exit 3, real broker socket. Self-exit sends one report; SIGHUP and SIGUSR2 send none.
- [ ] Atlas (`atlas/couch.md` lifecycle notices; wrapper section), the artifactpath inventory for new files, run the guard tests, the recorded suite, `sdlc close`.

## Decisions for the TL

1. **Self-exit only.** A child stopped by the wrapper (park SIGHUP, teardown SIGTERM, SIGUSR2 restart) produces no exit notice; those already have `parked` and `relaunched`.
2. **Non-peer agents are silent.** agy, muse and grok have no registry session, so they get no authenticated caller and no notice.
3. **Fencing by launch ordinal plus agent identity.** It is checked against the ready record, because the binding survives a SIGUSR2 exec.
4. **Coalescing:** the notice goes through #574's publisher, so an exit coalesces with an unsent earlier fact for the same slot, as the other kinds do.
