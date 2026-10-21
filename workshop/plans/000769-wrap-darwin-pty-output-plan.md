---
collaboration-mode: discussion
---

# Preserve wrap Darwin PTY output — design decision and implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. TL approved the parent lease and revised termination contract on 2026-10-21. Complete the production-caller inventory, then use full-flow change-code before implementation.

**Goal:** Preserve exact final output without silently changing wrap's terminal completion, owned-group cleanup or launch publication.

**Architecture:** Existing masterPump owns output; ownedGroup owns stop and the sole reap, with the leader unreaped to fence its process-group identity. A Darwin-only retained slave preserves output; the approved lifetime contract keeps a live closed-stdio target owned until exit or explicit stop. The child-helper alternative has the same EOF limitation and unnecessary launch machinery.

**Tech stack:** Go, creack/pty, native Darwin PTYs, existing ttyio and ownedGroup.

## Approved decision and alternatives

Both output protections retain a kernel slave reference. Consequently neither delivers the old last-slave EOF while a target closes all stdio and remains alive. Do not claim that opening and closing /dev/tty before exec solves this: the implicit controlling-terminal reference survives the descriptor close.

**Approved by TL (receipt 7829dd7f-d4d5-4cae-b69b-b06f40e2a7f2):** use a wrap-local Darwin parent slave lease, consistent with #768's ownership approach. A live closed-stdio target stays owned until its exit or an explicit existing wrapper stop; final output drains before group cleanup and reap. Keep ordinary target exit, inherited descendants, cancellation, restart and publication behavior otherwise unchanged. The Spec and Done when now record this approved scope.

The original unchanged-EOF requirement was superseded by explicit TL approval: process exit is the authoritative end. Do not add proc-fd polling, delays, idle deadlines, silent fallback, output retries or a heuristic EOF replacement to meet the wording.

| Option | Final output | Live target closes stdio | Decision |
|---|---|---|---|
| Existing pty.Start | Can lose unread bytes | EOF starts group stop | Cannot meet output goal |
| Parent retained slave | Preserved in native probe | No EOF until target exit/revoke | Approved |
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

### Production-caller inventory and native qualification

| Caller / production authority | Native qualification and expected normal exit / daemonization |
|---|---|
| Public pair wrap / pair-go wrap, cmd/pair-go/main.go:106 | Shared testpair binary with explicit argv: exact marker and leader status, wrapper completes after leader exit. Same-group inherited-slave descendants stop; detached-session daemon does not hold wrapper open and is cleaned by the fixture. |
| Pair workbench, launcher/createflow.go:824,975; zellij/layouts/main-2.kdl:44 and main-3.kdl:55 | Extend agent_command_acceptance_test.go to execute each actual KDL stanza and its production encoded command through the real Pair binary for normal exit and daemonization, verifying capture/status and group outcome. |
| Couch new, cold resume, prepared recovery, switch and continuation: couch.go:694, resume.go:632, recovery_prepared.go:540, switchagent.go:428, continuation.go:437 -> launchTrackedThread | Each caller converges launch_existing.go:96,175 -> pair --couch-session-v1 resume TAG/layout -> production launcher createflow -> the two KDL stanzas above. Keep existing fake route/launch-profile assertions for each caller, and add native qualification of their common encoded-command/layout boundary. This composition tests the real child lifetime, without pretending fake Couch registration is a live acceptance check. |
| In-pane fresh restart, wrap.go:2673 and Run | Execute freshAgentInvocation's emitted replacement argv with the shared binary for normal exit and daemonization; existing native TestSIGUSR2ReExecsWrapperWithoutReplacingPaneProcess covers signal-triggered old-group stop and replacement decision. |
| Compatibility basename pair-wrap, entrypoint/alias.go and mode.go | Symlink shared executable as pair-wrap, qualify the same explicit-command native cases. No separate standalone cmd/pair-wrap survives. |
| Development launch, bin/pair-dev:35 and bin/couch-dev:29 | Execs public pair/couch and converges above; no separate wrap start. Existing dev-rebuild/entrypoint tests plus the native public/layout qualification cover it. |
| Couch warm reattach | Does not create a wrapper: launch_existing.go:96 omits layout/profile authority. Existing warm/cold route tests pin this exclusion; no new native child should be launched by the warm branch. |

Native daemonization has two distinct outcomes: a descendant retaining slave stdio in the original owned group (including ignored HUP/TERM) is stopped by existing ownedGroup cleanup; a setsid descendant outside that group is not adopted or signalled, does not delay wrapper completion on Darwin leader revoke, and is explicitly killed/reaped by the test's independent owner. Do not broaden process ownership.

Pure entities: no new parser or persisted model is proposed. Existing selector and exit-cause functions remain the authority. The approved policy is reflected in Spec and Done when before code.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Wrap PTY start/lease adapter, approved | cmd/internal/wrapcmd/pty_darwin.go, pty_other.go, pty.go | new | pty.Open, original exec.Cmd.Start, master/slave lifetime |
| Native lock/start boundary | cmd/internal/wrapcmd/native_lock.go | unchanged | Selector validation and lock acquisition using original argv |
| Output/cleanup owner | cmd/internal/wrapcmd/wrap.go | modified | masterPump, ttyio, initialization abort and final cleanup |
| Owned group | cmd/internal/wrapcmd/owned_group.go | unchanged | Existing stop, grace and sole reap; no early competing waiter |

ARCH-DRY: align lease semantics with #768 without inventing a cross-owner framework or importing another command package. Review the landed #768 implementation before writing the wrap adapter. Preserve original cmd.Path/Err/Args, env, cwd, ExtraFiles and SysProcAttr; fill only nil stdio and retain direct exec/start errors. Locks and all launch consumers see the real original command and PID. No helper executable or shell interpolation.

## Lifetime contract

Actions: run, interact/resize, stop, restart, inspect failure, explicitly launch again.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Ordinary streaming | Existing terminal session | Interact/resize or stop | Existing interruption limits |
| Live child closes all stdio | Session remains owned; no further output | Stop, or let child exit | Unlike today, no automatic stop on descriptor closure |
| Direct leader exits, descendant holds slave/ignores HUP | Drain final output, then existing owned-group stop and reap | Inspect result | Descendant is stopped as today; no detached survival promise |
| Wrapper cancellation/restart | Existing group stop; terminal restored after join | Explicitly launch again, or existing restart | Interrupted output may be incomplete |
| PTY/open/exec/setup failure | Existing launch failure; no ready publication for failed launch | Inspect failure, explicitly launch again | No successful launch claimed |
| Read/write failure or abort | Preserve original failure, stop/join owned group, release handles | Inspect failure | Unread bytes may be abandoned on abort |

One owner retains and releases each descriptor exactly once, including setup failure. No early cmd.Wait: its unreaped leader fences group identity. Normal Darwin leader exit must drain/revoke and deliver EOF even with inherited slave holders; qualify rather than assume it. On abort, close master/lease before waiting so a child blocked in exit-time drain cannot deadlock cleanup. Existing group signal/grace/status policy remains authoritative.

Encapsulated lease model: startPTYWithOpen acquires master/slave in a local unstarted scope; open or original cmd.Start failure closes locally and returns no handle. Success transfers a startedPTY{master, optional slave, sync.Once release, stable releaseErr} to runWithPTY. Its only slave transition is Held -> Released through releaseSlave, enforced by sync.Once for EOF, abort and repeated/concurrent cleanup. Non-Darwin is an empty lease. No event may reacquire a released lease. The master remains with existing ttyio/masterPump ownership, whose close is idempotent at os.File/ttyio boundaries.

run delegates to runWithPTY(args, streams, startPTY), an instance-local starter seam. The existing startNativePTY callback captures the returned handle while keeping its original command/selector signature. Register group.Wait immediately for a started child, then defer slave release so it executes BEFORE the group's deferred Wait on every initialization failure. Normal masterPump completion releases the slave before explicit group.Wait; deferred release is a harmless idempotent check. Remove wrap.go:2982's direct Process.Wait and use the registered group's stop/reap instead: no direct Process.Wait or new waiter may compete with ownedGroup. Stop/restart may request StartStop concurrently, but never release or reap; EOF/abort owner does that. Sequence tests inject starter/closeable failures and barriers, not global mutable start hooks.

## Implementation tasks

TL approved these tasks and the reconciled issue contract. One atomic review boundary; use full flow for the changed process-lifetime policy, even if the implementation fits the quick shell.

### 1. Native owner regression and policy pin

Files: cmd/internal/wrapcmd/pty_darwin_test.go, run_test.go, existing owned_group_test.go and native_exit_report_test.go.

- [x] runWithPTY: barrier-controlled native child closes all stdio before first master read; compare exact stdout/raw capture, EOF and status0/7; original immediate-close start must fail. Hold a live closed-stdio child via independent pipe, then release or invoke stop/restart. Qualify inherited ignored-HUP/TERM group descendants and detached-session daemon; assert original ownership boundary and single leader reap.
- [x] Production caller composition (table above): actual public/alias/fresh argv and both production layout stanzas drive real wrapped children for normal and daemonized exit; assert exact capture, status, wrapper completion and independently witnessed descendant/group outcomes. Reuse existing Couch fake route assertions to prove every named caller reaches that qualified boundary.

### 2. Adapter and owner integration

Files: cmd/internal/wrapcmd/pty.go, pty_darwin.go, pty_other.go, wrap.go; tests alongside each seam.

- [x] startPTYWithOpen(cmd, open): adversarial allocation/start failure and original command configurations; stateful resource owner/native child assert no leaked descriptors/child and preserved Path/Err/Args/env/cwd/ExtraFiles/SysProcAttr. Missing/non-executable/malformed target fails synchronously, before readiness publication; native lock refusal never calls starter.
- [x] startedPTY.releaseSlave(): repeated and concurrent release, native revoked descriptor and reported closer failure; assert Held -> Released exactly once, stable error, no reacquisition. Count successful/failed leases at8/16 lifetimes.
- [x] runWithPTY cleanup and ownedGroup.Wait: inject initialization failure after successful Start, stdout/read failure, stop/restart before/after EOF and repeated cleanup. Assert master/slave close before the sole group reap, original error precedence and joined workers. Remove direct Process.Wait; no early waiter or group-identity fence violation.
- [x] Existing resize/input and large-output paths: real terminal input/size plus concurrent 128KiB output qualify unchanged backpressure, not a larger write behind the small delayed-read barrier. Run targeted package/race checks, no timeout bump or retry.

### 3. Enforcement, docs and landing

Files: atlas/terminal.md, atlas/architecture.md, atlas/index.md, atlas/go-lint-catalog.md if adding enforcement, individual affected-reader certificates for changed tests.

- [x] Document approved lifecycle, Darwin limits and cleanup ownership; update every touched summary. Evaluate existing #768 lint/enforcement before adding a duplicate rule; an owner-start regression must reject an immediate slave-close mutation.
- [x] Audit changed reader tests and production source dispositions. Run test-affected-inventory and targeted artifactpath/repolint preflights.
- [x] Push final head and enqueue the required recorded check.

The recorded suite pass, TL live-qualification decision, close review and merge/CI remain mandatory gates before landing; report outcome only after confirmed merge.

## Operating envelope and architectural checks

- ARCH-CONSTRAINTS: launch/shutdown path, one extra fd per live Darwin wrap, O(1) start/release, no additional per-byte/event scans or polling. Count 8 and 16 lease lifetimes: exactly one optional retained descriptor per successful start, exactly one release and zero live leases after join/failure. Streaming stays O(bytes) with existing buffer/backpressure; large-output tests read concurrently.
- ARCH-PURE/STATE: policy table above names observable completion changes; IO adapter owns resource effects, existing reducers/selectors remain pure. Test both cancellation-before-EOF and EOF-before-cancellation and duplicate cleanup.
- ARCH-MOCK: stateful descriptor/child lifecycle doubles cover failure ordering; native Darwin PTYs qualify actual kernel behavior and process-group cleanup. No new external service.
- ARCH-FUNERAL: lease creator is the starter, last consumer is output/abort cleanup; owner return releases it and joins owned children. No durable artifact, new daemon or retained background waiter.
- ARCH-SECURE: original argv and child identity, no shell wrapper or untrusted PID discovery.
- ARCH-EVOLVE: no persisted format changes.
- ARCH-PURPOSE: implement the explicitly approved process-lifetime policy and qualify all production callers rather than silently relying on the old EOF proxy.

## Revisions

- 2026-10-21: Initial design records pair:2's corrected helper counterexample and pair:4's independent native reproduction. No implementation selected or approved; request TL decision on the live closed-stdio contract before change-code.

- 2026-10-21: TL approved wrap-local parent retention, process-exit/explicit-stop lifetime, full flow and unchanged group/start/publication contracts. Reconciled issue Spec/Done when; add exhaustive caller inventory plus native normal-exit/daemonization coverage before code.
- 2026-10-21: Address plan gate PQ-1/PQ-2/PQ-3: enumerate all production caller families and native composition seams, define function-level adversarial strategies, and specify the encapsulated lease transition plus sole group-reap authority, including removal of the existing direct Process.Wait on initialization failure.

- 2026-10-21: Native implementation and targeted qualification complete. Keep future suite/review/landing gates as explicit prose rather than falsely completed implementation checkboxes; recorded suite is queued, no pass claimed.
