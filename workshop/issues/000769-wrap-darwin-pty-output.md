---
id: 000769
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '72ac9bef83a48057a5110cb8fb864d6fd5e4fd4d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T06:43:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Preserve wrap Darwin PTY final output without changing termination

## Problem

#768 audited wrapcmd.runProxy/startNativePTY and confirmed a generic-command PTY owner exposed to Darwin last-slave-close final-output loss. #768 fixes scribe only; this owner needs its own lifetime design.

Native planning probe from #768: openpty/fork, setsid/TIOCSCTTY, child writes final-marker and closes all slave stdio, then acknowledges out-of-band before parent first read. Baseline lost the bytes; child /dev/tty open/close and parent retained-slave variants each preserved them. Retained variant reached EOF and exited7. Additional descendant probe kept an inherited slave open in a descendant ignoring SIGHUP: Darwin still delivered exact marker and EOF on leader exit while descendant was alive. These are feasibility probes, not qualification of this owner. Sources: https://bugs.ruby-lang.org/issues/20682 and Apple XNU tty_tty.c/kern_exit.c.

Termination-semantics risk: Master EOF invokes ownedGroup.Wait and stops a still-live process group. Retaining the slave delays this transition when the target closes stdio but keeps running; a generic retention change could leave such a target running indefinitely.

## Spec

Design output preservation for this owner while explicitly preserving its EOF, termination and publication contract. Compare child controlling-tty helper versus retained-slave lifecycle. Preserve direct argument semantics, status/signals, exact output and cleanup ownership. No deadline tuning or retry after EOF.

## Done when

- A native regression forces all child slave descriptors closed before first read and verifies exact final bytes, EOF and status.
- Live closed-stdio targets, inherited-slave descendants (including ignored SIGHUP), cancellation and abort have explicit tested termination semantics.
- Existing owner lifecycle consumers and executable startup errors remain correct; resources are reaped/released exactly once.

## Plan

Plan first; independently claim from main. Do not stack on #768.

Durable design/conditional implementation plan: [wrap Darwin PTY output](../plans/000769-wrap-darwin-pty-output-plan.md). Await TL decision: both proposed protections suppress EOF for a still-live closed-stdio child. No change-code or production edits before an approved termination contract.

## Log

Filed unclaimed per TL approval of #768 on 2026-10-21. Probe provenance and audited owner paths are in workshop/plans/000768-darwin-pty-final-output-plan.md, planning commit b55089f72c233e0532918172a29b60291f798cfb. Retention must not be generalized without resolving the termination-semantics risk above.

### 2026-10-21 — Planning and native counterexample

Claimed and started from main in canonical pair-slot4; #757 remains parked awaiting operator approval. Coordinated with pair:2 through authenticated #768 handoff and corrected native probe. Read its scribe-only ownership design without importing unlanded code. Fresh-context read-only audit confirmed masterPump EOF drives ownedGroup.Wait, which starts group termination before its sole reap; preserve original selector argv, actual target PID and synchronous start-error publication.

Independently reproduced native openpty/fork/setsid/TIOCSCTTY child that writes final-marker, closes all stdio, acknowledges a pipe and blocks alive on a separate hold pipe. Baseline returned empty bytes and EOF; child /dev/tty open+close and parent-retained slave each returned exact marker but no EOF while held. After hold release, both protected variants returned EOF and status7. All children reaped and descriptors closed. Two bounded 200ms readiness observations are diagnostic only, never production policy. Both candidate protections therefore fail the unchanged live-target termination requirement. ARCH-PURPOSE/STATE: do not silently reinterpret it; plan presents conditional retention and requires TL/operator policy approval. No timeout change, polling mechanism, code change or change-code invocation.

## Revisions

- 2026-10-21: Added a gated design checkpoint after the native live-target counterexample. Existing Spec and Done when remain authoritative and unchanged pending TL decision; conditional plan is not implementation approval.
