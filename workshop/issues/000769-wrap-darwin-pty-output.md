---
id: 000769
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 1.861
collaboration-mode: collaboration
card_mirror: '53af7dc451c1254e0abe8020fedd38c1bbfea6a4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T06:43:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Preserve wrap Darwin PTY final output without changing termination

## Problem

#768 audited wrapcmd.runProxy/startNativePTY and confirmed a generic-command PTY owner exposed to Darwin last-slave-close final-output loss. #768 fixes scribe only; this owner needs its own lifetime design.

Native planning probe from #768: openpty/fork, setsid/TIOCSCTTY, child writes final-marker and closes all slave stdio, then acknowledges out-of-band before parent first read. Baseline lost the bytes; child /dev/tty open/close and parent retained-slave variants each preserved them. Retained variant reached EOF and exited7. Additional descendant probe kept an inherited slave open in a descendant ignoring SIGHUP: Darwin still delivered exact marker and EOF on leader exit while descendant was alive. These are feasibility probes, not qualification of this owner. Sources: https://bugs.ruby-lang.org/issues/20682 and Apple XNU tty_tty.c/kern_exit.c.

Termination-semantics risk: Master EOF invokes ownedGroup.Wait and stops a still-live process group. Retaining the slave delays this transition when the target closes stdio but keeps running; a generic retention change could leave such a target running indefinitely.

## Spec

Preserve Darwin final output with a wrap-local parent slave lease aligned with #768. A live target that closes all stdio remains owned until the target exits or an explicit wrapper stop; descriptor EOF was a proxy, not the authoritative process end. Preserve ownedGroup stop and sole reap, direct argument semantics, original PID, start errors, publication, status/signals, exact output and cleanup ownership. No new polling, deadline tuning or retry after EOF. Document the changed live closed-stdio behavior in wrap help and atlas, enumerate every production wrap caller, and qualify their native normal-exit and daemonization behavior.

## Done when

- A native regression forces all child slave descriptors closed before first read and verifies exact final bytes, EOF and status.
- A live closed-stdio target remains owned until its own exit or explicit stop; inherited-slave descendants (including ignored SIGHUP), cancellation, restart and abort have native tested termination semantics.
- Every production wrap caller, including Couch agent launch, has an enumerated native test for normal agent exit and target daemonization; help and atlas explain the changed lifetime policy.
- Existing owner lifecycle consumers and executable startup errors remain correct; resources are reaped/released exactly once.

## Plan

Plan first; independently claim from main. Do not stack on #768.

Durable implementation plan: [wrap Darwin PTY output](../plans/000769-wrap-darwin-pty-output-plan.md). TL approved parent retention and the revised live closed-stdio lifetime contract on 2026-10-21. Full flow; reconcile the caller inventory and run change-code before implementation.

- [x] Pin final-output loss and approved lifetime with native child/descriptor barriers; enumerate and qualify every production caller.
- [x] Implement the wrap-local Darwin slave lease through normal drain and abort, preserving original command, locks, publication and ownedGroup stop/reap.
- [ ] Document help/atlas behavior; qualify lifecycle, inventory, recorded suite, close review and landing gates.

## Log

Filed unclaimed per TL approval of #768 on 2026-10-21. Probe provenance and audited owner paths are in workshop/plans/000768-darwin-pty-final-output-plan.md, planning commit b55089f72c233e0532918172a29b60291f798cfb. Retention must not be generalized without resolving the termination-semantics risk above.

### 2026-10-21 — Planning and native counterexample

Claimed and started from main in canonical pair-slot4; #757 remains parked awaiting operator approval. Coordinated with pair:2 through authenticated #768 handoff and corrected native probe. Read its scribe-only ownership design without importing unlanded code. Fresh-context read-only audit confirmed masterPump EOF drives ownedGroup.Wait, which starts group termination before its sole reap; preserve original selector argv, actual target PID and synchronous start-error publication.

Independently reproduced native openpty/fork/setsid/TIOCSCTTY child that writes final-marker, closes all stdio, acknowledges a pipe and blocks alive on a separate hold pipe. Baseline returned empty bytes and EOF; child /dev/tty open+close and parent-retained slave each returned exact marker but no EOF while held. After hold release, both protected variants returned EOF and status7. All children reaped and descriptors closed. Two bounded 200ms readiness observations are diagnostic only, never production policy. Both candidate protections therefore fail the unchanged live-target termination requirement. ARCH-PURPOSE/STATE: do not silently reinterpret it; plan presents conditional retention and requires TL/operator policy approval. No timeout change, polling mechanism, code change or change-code invocation.


### 2026-10-21 — Implementation and native qualification

Full-flow change-code passed with CLEAN plan review (all PQ-1/PQ-2/PQ-3 addressed) and INFO estimate review. The estimate is issue-wide, including already-incurred policy/spec work; review allocation includes active verification/landing work, not machine queue idle. Native caller qualification is the optimistic component noted by the reviewer.

Implemented wrap-local Darwin parent slave ownership, idempotent/stable-error release, instance-local runWithPTY starter, and release-before-sole-ownedGroup reap. Removed the pre-existing competing Process.Wait on transport initialization failure. Original command argv/PID/start failures and lock/publication paths remain intact. Non-Darwin delegates to upstream start. Help and atlas describe the approved live closed-stdio policy; extended the existing wrapper-child-release lint class to reject direct Process.Wait, with three initial RED examples then GREEN. Partial opener allocation failure also produced a native RED before cleanup correction.

The first deterministic native regression failed on upstream pty.Start for status0 and7: both final markers were empty after child stdio-close acknowledgement and before first master read. The protected adapter/production runner now preserves exact bytes/status and 128KiB concurrent output; causal request/ACK proves a closed-stdio target remains executing until release. Direct native TERM/USR2 tests assert drain, stop/restart flags, reaping and native lock lifetime. Resource tests cover32 concurrent release requests, stable close errors,8/16 simultaneous leases and start/open failures. No timeout bump, polling, retry or competing production waiter.

Caller coverage: public pair/pair-go, pair-wrap compatibility, emitted fresh replacement, and both actual KDL stanzas driven by production createflow all execute the shared candidate with normal, same-group ignored-HUP/TERM descendant, and escaped setsid daemon cases (18 rows). They assert exact raw marker/status7; same-group cleanup escalates, escaped daemon remains independently responsive after wrapper return. Shared pairlifecycletest fixture avoids duplicated protocols. Existing Couch route checks for new/cold resume/recovery/switch/continuation and warm exclusion passed5.587s, proving those callers converge on the qualified launcher/layout boundary.

Combined targeted race command passed wrapcmd33.079s and launcher11.670s; managed actual-Run race scenarios passed3.886s (normal, inherited-slave detached descendant, missing target, closedstdio SIGTERM, closedstdio SIGUSR2). Managed tests assert exact native PID/start/ordinal readiness, final stdout/raw bytes/status7 and one matching native-exit report; failed start publishes none, controlled stop/restart reports no self exit. Full repolint passed18.286s, artifactpath20.133s. Reader audit: launcher caller test reads actual KDL, so register its two tests in SOURCE_GUARDS and fixed_scanner certificate; managed/native owner tests read only private fixtures and are local_or_literal_fixture. No local-gate protected configuration changed. make test-affected-inventory passed after recording the three audited certificates. go vet passed for wrapcmd/launcher. An expanded optional vet run also inspected unchanged pairlifecycletest tests and flagged the intentionally discarded String snapshot at live_zellij_diagnostics_test.go:20; Git confirms the same line on HEAD, and its native race-snapshot unit test passes. No unrelated source edit or gate bypass.

## Revisions

- 2026-10-21: Added a gated design checkpoint after the native live-target counterexample. Existing Spec and Done when remain authoritative and unchanged pending TL decision; conditional plan is not implementation approval.

- 2026-10-21: TL approved the live closed-stdio behavior change (receipt 7829dd7f-d4d5-4cae-b69b-b06f40e2a7f2): keep target owned until exit/explicit stop. Reconciled Spec and Done when; require help/atlas documentation and exhaustive production-caller native normal-exit/daemonization pins. Preserve original argv/PID/start errors/publication and ownedGroup stop/reap; full flow, no new polling or deadlines.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md, Method A only (calibration reported stale). Existing creack/pty and #768's established pattern remove a greenfield/library assumption. Native PTY and group ownership are familiar from #766; use familiarity1.0. Three smaller modules cover adapter/lease, owner integration and caller qualification. Design uses thorough-plan discount0.2 plus15% buffer; implementation uses40% of v2 ranges. Native API discovery remains a separate primitive because kernel exit/revoke qualification matters. No cross-repo implementation.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.30 impl=0.12
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 1.861
```
