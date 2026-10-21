---
collaboration-mode: discussion
---

# Preserve scribe Darwin PTY final output — implementation plan

> **For agentic workers:** Consult AGENTS.md Section3. Keep this lifetime correction in one implementation context. Await TL approval and change-code before implementation.

**Goal:** Preserve exact final stdout/log bytes when a scribe child closes its slave descriptors before the first master read.

**Architecture:** Retain one parent-side slave on Darwin through the existing master drain, release it before reaping, and preserve direct exec. Other platforms retain upstream behavior. Other production PTY owners are audited below and require separate lifetime designs.

**Tech stack:** Go, pinned creack/pty v1.1.24, native Darwin PTYs, existing scribe raw-mode/signal handling.

**Flow:** Full flow recommended for launch/lifetime risk, even if the final patch fits the quick shell; TL approval requested for this choice. One atomic issue boundary, no Mx split. After approval run change-code --flow full; derive estimate only after plan-quality passes.

## Evidence and alternatives

Scribe already drains before cmd.Wait. Loss of the last slave reference can discard bytes before the first read; waiting longer cannot recover them.

Native planning probe: openpty/fork; child setsid/TIOCSCTTY, writes final-marker, closes0/1/2, acknowledges via a separate pipe, then exits7. Parent first reads only after acknowledgement. Baseline was empty. Child /dev/tty open/close preserved output. Parent-retained slave also preserved output, reached EOF while held, and returned exit7. Separate slave TIOCNOTTY probe returned errno25 and preserved bytes/EOF/status. Feasibility only: automated acceptance still required.

Sources: [Ruby20682](https://bugs.ruby-lang.org/issues/20682) documents loss and the workaround wait-before-read hang; its final closure was test stabilization, so the workaround is not claimed shipped. Apple [cttyopen](https://github.com/apple-oss-distributions/xnu/blob/aca3beaa3dfbd42498b42c5e5ce20a938e6554e5/bsd/kern/tty_tty.c) retains the session tty reference; [proc_exit](https://github.com/apple-oss-distributions/xnu/blob/aca3beaa3dfbd42498b42c5e5ce20a938e6554e5/bsd/kern/kern_exit.c) closes descriptors before session-leader drain/revoke. Native tests must prove EOF rather than infer it from those sources.

| Option | Benefit | Cost / decision |
|---|---|---|
| Darwin retained slave, scribe only | Preserves PID, argv, PATH/env/cwd, synchronous exec errors and drain-before-Wait | One fd per live scribe, explicit drain/abort release. Recommended |
| Child /dev/tty exec helper | Establishes child/session reference before target exec | New helper route/dependency and startup-error handshake; Couch/test binaries cannot assume their executable supports Pair internal modes. Excess machinery for scribe |
| Delays or retry after EOF | Small patch | Cannot recover discarded bytes; rejected |

If native EOF/cleanup qualification fails, stop and return evidence to TL. No new drain deadline, silent fallback or deadline tuning.

## Scope and remaining exposure

| Production owner | Existing lifecycle | This issue |
|---|---|---|
| scribecmd.Run, public pair scribe | Master drain then direct child Wait | Fix/qualify |
| wrapcmd.runProxy/startNativePTY | EOF invokes ownedGroup.Wait and stops a live group | Audited, unchanged: retaining slave changes a target closing0/1/2 while remaining alive |
| ptychild.Start, Couch PtyRunner and termcmd | EOF ends terminal input then reaps/publishes completion | Audited, unchanged: retention changes observable input-ended boundary |

Both remaining owners admit generic commands and remain exposed. After scope approval, file unclaimed follow-up issue(s) and send hub references; do not silently generalize #768. Non-PTY ExecRunner, probes and test-only hosts are unchanged. No Couch acknowledgement protocol changes.

## Core concepts

Pure core: no new user lifecycle/parser/persisted format. A compile-time Darwin retention choice is the only new policy; do not build a generic framework.

| Integration | Location | Status | Wraps |
|---|---|---|---|
| Native start adapter | cmd/internal/scribecmd/pty_darwin.go; pty_other.go | new | pty.Open and original exec.Cmd.Start |
| Slave lease | cmd/internal/scribecmd/pty.go | new | optional descriptor, idempotent close |
| Child/output owner | cmd/internal/scribecmd/scribecmd.go | modified | byte pump, signals/raw mode, sole child wait |

Internal shape:

```go
type startedPTY struct { master *os.File; slave io.Closer }
func startPTY(cmd *exec.Cmd) (*startedPTY, error)
```

The handle owns the optional slave; release is idempotent and retains its error. Non-Darwin delegates to pty.Start without retention. Darwin reuses pty.Open, fills only nil stdio, preserves SysProcAttr fields while setting Setsid/Setctty, and invokes the original cmd.Start. Preserve cmd.Path/Err/Args, argv[0], cwd/env, ExtraFiles and synchronous missing/non-executable errors. Failed setup/start closes all acquired fds.

Inject an internal starter into the production run body for tests. A test closure invokes the real starter, waits for the child out-of-band stdio-close acknowledgement, then returns the handle. Thus no master read can precede the barrier. Public Run always supplies the native starter: no global test hooks, sleeps, flags or production environment knobs.

## Lifetime and visible outcomes

| Event | Required transition |
|---|---|
| Open/start failure | Close acquired fds; existing error; no live child |
| Started | Run owns master, optional slave and sole wait authority |
| Read returns bytes and error | Forward n>0 to stdout/unpaused log before handling error |
| Terminal EOF/read failure | Release slave, then one cmd.Wait; restore raw mode and stop/join signal watchers |
| Initialization abort/controlled failure | Signal child as needed; CLOSE MASTER and release slave BEFORE joining Wait, so exit-drain cannot wait on unread master |
| Repeated cleanup | Idempotent release, exactly one reap, no competing waiter |

Close can follow kernel revoke: test it and report cleanup errors without replacing child exit status. Preserve exec.ExitError.ExitCode(), including signal deaths. No new waiter, exit polling, retry or timer. Pre-existing arbitrary-io.Reader stdin-copy cancellation is outside this output fix; add no unjoined worker. New resources end on every controlled return. Wrapper SIGKILL cannot promise complete logging; kernel fd cleanup remains.

Actions: run, type/resize, interrupt/stop, inspect, explicitly run again.

| Failure/state | Visible state | Recovery | Possible loss |
|---|---|---|---|
| Streaming/final drain | Same output/log, completion after drain/reap | Type/resize or stop | No normal final-close loss |
| Open/exec/setup failure | Existing error/status, target not started or owned child cleaned up | Inspect; explicitly run again | No successful result claimed |
| Nonzero/signal exit | Existing child result after drain | Inspect | Same failure/interruption |
| Parent/transport abort | Existing failure/stop behavior | Inspect; explicitly run again | Unread output may be abandoned as before |
| Cleanup error | Diagnostic, child status stays attributable | Inspect | No silent unfinished cleanup |

Scribe already waits for a live target after it closes stdio and master EOF occurs. Retention delays EOF but must not newly stop/detach that target. Pin this case to distinguish scribe from wrap.

## Constraints and principles

ARCH-CONSTRAINTS: one extra fd per live Darwin scribe, O(1) startup/cleanup, no extra operation per byte. Keep4096-byte buffer, pause/resume and output writes. At8 and16 lease lifetimes count one retained descriptor per successful start and exactly one release, zero after join and zero on failed starts. Large-output backpressure uses concurrent reading, separate from the small delayed-first-read oracle.

ARCH-DRY/PURE: scribe-local adapter/lease, no premature all-Pair framework. Stateful closeable fakes retain open/closed resource state; native PTYs qualify kernel semantics. ARCH-STATE/FUNERAL: child scope creates lease; drain/abort releases it; one owner reaps. No durable state/collector. ARCH-SECURE: preserve argv slices and inherited fd numbering, no shell interpolation. ARCH-EVOLVE: no format changes. ARCH-PURPOSE: repair documented arbitrary-command scribe and record remaining owners.

## Tasks (single atomic boundary)

### 1. Pin failure and implement lease

Files: pty.go, pty_darwin.go, pty_other.go and corresponding tests under cmd/internal/scribecmd/; native writer remains test-only.

- [ ] Add non-terminal-aware writer: small exact payload, close0/1/2, acknowledge out-of-band; no /dev/tty or protective extra slave.
- [ ] Gate starter return on acknowledgement. Old upstream start must fail exact stdout/log on Darwin (RED). Safety timeout diagnoses/cleans only; it establishes no ordering.
- [ ] Implement minimal adapter/lease. Require exact stdout AND unpaused log bytes, EOF and exit0/7. Mutate to immediate slave close in a disposable copy and require RED; restore source.
- [ ] Cover failed open/start, nil/provided stdio, attrs, close-after-revoke, repeated/concurrent release and N/2N resource counts using stateful resources and native tests.

### 2. Integrate normal and abort ownership

Files: scribecmd.go, scribecmd_test.go, new Darwin lifecycle tests.

- [ ] Add initialization-failure/abort tests while child exit drains: master closes before wait/join, no unreaped child/slave.
- [ ] Adopt starter without public CLI change; release slave after output loop, before sole wait. Every post-start return owns cleanup/reaping.
- [ ] Preserve pause/resume, resize/raw-mode restore, argv empty/spaces/metacharacters, PATH/env and existing exit/signal results.
- [ ] Native cases: interactive input/resize/terminal interrupt, SIGTERM/SIGHUP, leader exit with inherited-slave descendant including ignored SIGHUP, empty output, closed-stdio-but-live target, concurrent large backpressure. Clean exact owned processes/fds.
- [ ] Run go test ./cmd/internal/scribecmd -count=1, targeted race and vet. Never demand a larger-than-PTY-buffer write complete before reading.

### 3. Document, qualify public route and land

Files: docs/pair-scribe.md, README/atlas, changed/new test-reader certificates.

- [ ] Document Darwin ownership, controlled-abort limits, unchanged other-platform behavior and remaining exposures. Correct touched stale standalone-scribe packaging comment.
- [ ] Public built-pair scribe smoke: short commands, exact stdout/log and status. Distinguish it from the internal deterministic barrier. Coordinate staging/live check with hub before touching pair:0.
- [ ] Audit changed/new repository readers, register scanners, regenerate individual certificates; run make test-affected-inventory.
- [ ] Push, recorded suite, SDLC close/review, merge with required integrated evidence/CI, report landing. No blind flaky rerun or timeout adjustment.

## Acceptance clarification requiring approval

Original Done when says child exits before first read. A protected Darwin child may be inside exit waiting for drain; waiting for waitpid before releasing the read barrier deadlocks. Assert the destructive prerequisite: completed writes and closure of ALL child-owned slave descriptors before first master read; then drain, observe EOF and reap with correct status. This is deterministic coverage of the loss, not a weakened byte check or new deadline.

## Revisions

- 2026-10-21: [Initial shared-owner draft](000768-darwin-pty-final-output-plan.md.side/shared-owner-draft.md) proposed all three owners. Audit found wrap's live-target stdio-close cleanup would change; ptychild has its own input-ended boundary. Current plan supersedes that draft: retain #768 scribe scope, report other exposures for separate designs, require master-close-before-wait on abort, separate small-output/backpressure test protocols.

## Descendant-held slave contract

On Darwin, exit of scribe's session-leader child drains and revokes its controlling terminal. The master must therefore return terminal EOF after final bytes even if a descendant retains a slave and ignores SIGHUP. Scribe releases its parent lease and reaps only its direct child, returning that status; it does not wait for, adopt or kill detached descendants. No new timer or descendant wait is introduced. Native planning probe confirmed exact final-marker, EOF while the descendant remained alive, and direct-child exit7 with the parent slave still held. Qualification must exercise this through the production run path, with an out-of-band descendant-ready barrier; keep the descendant alive until after scribe returns, then clean the exact owned descendant in test cleanup. Failure to reach EOF is a design failure: stop, report evidence and re-plan rather than introducing a deadline.

## Approval checkpoint

TL approved retained-slave choice, scribe-only scope, acceptance clarification and full flow on 2026-10-21 (receipt a711dfa3-61f8-4c22-847e-25caf4a69a63). Required descendant-held-slave contract is explicit above and will be tested. Proceed to change-code; no production/test edits yet.

- 2026-10-21: TL approved full flow and required deterministic completion with descendant-held slave. Added explicit Darwin revoke/EOF contract and native feasibility result; no child-wait-before-read.
