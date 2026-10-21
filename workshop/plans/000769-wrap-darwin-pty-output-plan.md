---
collaboration-mode: discussion
---

# Preserve wrap Darwin PTY output — design decision and implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. This is a design checkpoint, not implementation authorization. Do not run change-code until the TL resolves the termination-policy decision below.

**Goal:** Preserve exact final output without silently changing wrap's terminal completion, owned-group cleanup or launch publication.

**Architecture:** Existing masterPump owns output; ownedGroup owns stop and the sole reap, with the leader unreaped to fence its process-group identity. A Darwin-only retained slave is the smallest output-preserving candidate, but it changes a live closed-stdio target's completion boundary. The child-helper alternative has the same limitation. No implementation currently satisfies the original unchanged-termination requirement.

**Tech stack:** Go, creack/pty, native Darwin PTYs, existing ttyio and ownedGroup.

## Decision required before implementation

Both output protections retain a kernel slave reference. Consequently neither delivers the old last-slave EOF while a target closes all stdio and remains alive. Do not claim that opening and closing /dev/tty before exec solves this: the implicit controlling-terminal reference survives the descriptor close.

Recommendation, **only if the TL/operator accepts the policy change**: use a wrap-local Darwin parent slave lease, consistent with #768's ownership approach. A live closed-stdio target stays owned until its exit or an explicit existing wrapper stop; final output drains before group cleanup and reap. Keep ordinary target exit, inherited descendants, cancellation, restart and publication behavior otherwise unchanged. This is not approved scope yet.

If preserving current EOF-driven termination is mandatory, reject both candidates and keep code unchanged while seeking an independently justified mechanism. Do not add proc-fd polling, delays, idle deadlines, silent fallback, output retries or a heuristic EOF replacement to meet the wording.

| Option | Final output | Live target closes stdio | Decision |
|---|---|---|---|
| Existing pty.Start | Can lose unread bytes | EOF starts group stop | Cannot meet output goal |
| Parent retained slave | Preserved in native probe | No EOF until target exit/revoke | Recommended only with explicit policy approval |
| Child /dev/tty open/close before exec | Preserved in native probe | Same missing EOF | Extra helper and exec-error handshake without solving the policy conflict; reject |

## Evidence and coordination

Authenticated pair:2 handoff for #768: scribe-local startedPTY owns master and optional Darwin slave; release is idempotent, normal read completion releases before sole Wait, abort kills its exact child and closes descriptors before Wait. Its current branch is unlanded: inspect only, do not stack or copy its commits. There is no shared public adapter to reuse. Scribe already waits for a live closed-stdio direct child; wrap instead starts group termination at master EOF.

Pair:2's corrected native probe found both candidate protections suppress EOF for the live target. Pair:4 independently reproduced the counterexample on 2026-10-21: os.openpty; fork; child setsid/TIOCSCTTY, writes final-marker, closes 0/1/2, acknowledges a separate pipe, then stays alive on a private hold pipe. Parent closes its ordinary slave and reads only after the acknowledgement. Results:

| Variant | While child held alive | After releasing child | Status |
|---|---|---|---|
| Baseline | Empty bytes, EOF | Empty bytes | 7 |
| Child /dev/tty open/close | Exact final-marker, no EOF during observation | EOF | 7 |
| Parent slave lease | Exact final-marker, no EOF during observation | EOF | 7 |

The observation used two bounded readiness checks of 200 ms; that interval is a diagnostic observation, not a production deadline or proof of eventual noncompletion. Exact children were reaped and all descriptors closed. The hold/ack pipes establish ordering, not elapsed time. Source review and the native counterexample justify rejecting a drop-in claim.

Sources: [Ruby #20682](https://bugs.ruby-lang.org/issues/20682), Apple [controlling terminal reference](https://github.com/apple-oss-distributions/xnu/blob/aca3beaa3dfbd42498b42c5e5ce20a938e6554e5/bsd/kern/tty_tty.c), [exit drain/revoke](https://github.com/apple-oss-distributions/xnu/blob/aca3beaa3dfbd42498b42c5e5ce20a938e6554e5/bsd/kern/kern_exit.c). Native owner qualification remains necessary.

## Core concepts

Pure entities: no new parser or persisted model is proposed. Existing selector and exit-cause functions remain the authority. A policy revision, if approved, must be reflected in Spec and Done when before code.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Wrap PTY start/lease adapter, conditional on approval | cmd/internal/wrapcmd/pty_darwin.go, pty_other.go, pty.go | new | pty.Open, original exec.Cmd.Start, master/slave lifetime |
| Native lock/start boundary | cmd/internal/wrapcmd/native_lock.go | unchanged | Selector validation and lock acquisition using original argv |
| Output/cleanup owner | cmd/internal/wrapcmd/wrap.go | modified | masterPump, ttyio, initialization abort and final cleanup |
| Owned group | cmd/internal/wrapcmd/owned_group.go | unchanged | Existing stop, grace and sole reap; no early competing waiter |

ARCH-DRY: align lease semantics with #768 without inventing a cross-owner framework or importing another command package. Review the landed #768 implementation before writing the wrap adapter. Preserve original cmd.Path/Err/Args, env, cwd, ExtraFiles and SysProcAttr; fill only nil stdio and retain direct exec/start errors. Locks and all launch consumers see the real original command and PID. No helper executable or shell interpolation.

## Conditional lifetime contract

Actions: run, interact/resize, stop, restart, inspect failure, explicitly launch again.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Ordinary streaming | Existing terminal session | Interact/resize or stop | Existing interruption limits |
| Live child closes all stdio | **Needs approval:** session remains owned; no output | Stop, or let child exit | Unlike today, no automatic stop on descriptor closure |
| Direct leader exits, descendant holds slave/ignores HUP | Drain final output, then existing owned-group stop and reap | Inspect result | Descendant is stopped as today; no detached survival promise |
| Wrapper cancellation/restart | Existing group stop; terminal restored after join | Explicitly launch again, or existing restart | Interrupted output may be incomplete |
| PTY/open/exec/setup failure | Existing launch failure; no ready publication for failed launch | Inspect failure, explicitly launch again | No successful launch claimed |
| Read/write failure or abort | Preserve original failure, stop/join owned group, release handles | Inspect failure | Unread bytes may be abandoned on abort |

One owner retains and releases each descriptor exactly once, including setup failure. No early cmd.Wait: its unreaped leader fences group identity. Normal Darwin leader exit must drain/revoke and deliver EOF even with inherited slave holders; qualify rather than assume it. On abort, close master/lease before waiting so a child blocked in exit-time drain cannot deadlock cleanup. Existing group signal/grace/status policy remains authoritative.

## Conditional implementation tasks

These tasks become executable only after approval and reconciliation of the issue contract. One atomic review boundary; use full flow for the changed process-lifetime policy, even if the implementation fits the quick shell.

### 1. Native owner regression and policy pin

Files: cmd/internal/wrapcmd/pty_darwin_test.go, run_test.go, existing owned_group_test.go and native_exit_report_test.go.

- [ ] Write a test-only native child and out-of-band acknowledgement barrier. It writes a small exact marker, closes all child slave descriptors, then allows first production master read. Assert exact stdout/raw capture, normal EOF and status 0/7; old start path must fail byte assertion.
- [ ] Pin the approved live closed-stdio behavior with an independent hold pipe. No timing-based ordering. If unchanged EOF behavior remains required, do not proceed with retention.
- [ ] Hold an inherited slave in an ignored-HUP descendant. Verify final bytes, EOF, group cleanup and sole leader reap through the real owner. Include SIGTERM-resistant descendant and existing escalation policy.

### 2. Adapter and owner integration

Files: cmd/internal/wrapcmd/pty.go, pty_darwin.go, pty_other.go, wrap.go; tests alongside each seam.

- [ ] Add the smallest original-command starter with optional idempotent slave lease. Non-Darwin preserves upstream pty.Start. Stateful closeable fakes assert stable close errors and exact release counts, including allocation/start failure.
- [ ] Pass the starter through startNativePTY without rewriting selector argv. Keep publication after successful real target start. Test missing, non-executable and malformed target; argv/env/cwd/ExtraFiles; native lock refusals; no false ready record.
- [ ] Thread lease ownership through output drain, startup abort, signal stop, restart, stdout/read failure and deferred group cleanup. Assert exactly one reap/release, descriptor close before abort wait and no late worker after owner return.
- [ ] Run real resize/input, small delayed-first-read and concurrent large-output/backpressure tests. Run targeted package and race checks; no timeout bump or retry.

### 3. Enforcement, docs and landing

Files: atlas/terminal.md, atlas/architecture.md, atlas/index.md, atlas/go-lint-catalog.md if adding enforcement, individual affected-reader certificates for changed tests.

- [ ] Document approved lifecycle, Darwin limits and cleanup ownership; update every touched summary. Evaluate existing #768 lint/enforcement before adding a duplicate rule; an owner-start regression must reject an immediate slave-close mutation.
- [ ] Audit changed reader tests and production source dispositions. Run test-affected-inventory and targeted artifactpath/repolint preflights.
- [ ] Push final head and run the required recorded check. Arrange TL-required live qualification for changed operator terminal behavior, then close review, suite/CI gates and retained merge runner. Report landing once confirmed.

## Operating envelope and architectural checks

- ARCH-CONSTRAINTS: launch/shutdown path, one extra fd per live Darwin wrap, O(1) start/release, no additional per-byte/event scans or polling. Count 8 and 16 lease lifetimes: exactly one optional retained descriptor per successful start, exactly one release and zero live leases after join/failure. Streaming stays O(bytes) with existing buffer/backpressure; large-output tests read concurrently.
- ARCH-PURE/STATE: policy table above names observable completion changes; IO adapter owns resource effects, existing reducers/selectors remain pure. Test both cancellation-before-EOF and EOF-before-cancellation and duplicate cleanup.
- ARCH-MOCK: stateful descriptor/child lifecycle doubles cover failure ordering; native Darwin PTYs qualify actual kernel behavior and process-group cleanup. No new external service.
- ARCH-FUNERAL: lease creator is the starter, last consumer is output/abort cleanup; owner return releases it and joins owned children. No durable artifact, new daemon or retained background waiter.
- ARCH-SECURE: original argv and child identity, no shell wrapper or untrusted PID discovery.
- ARCH-EVOLVE: no persisted format changes.
- ARCH-PURPOSE: unchanged-termination is a real requirement; do not silently weaken it. This plan remains gated on a policy decision because neither investigated mechanism satisfies it.

## Revisions

- 2026-10-21: Initial design records pair:2's corrected helper counterexample and pair:4's independent native reproduction. No implementation selected or approved; request TL decision on the live closed-stdio contract before change-code.
