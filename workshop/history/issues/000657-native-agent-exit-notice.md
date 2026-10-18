---
id: 000657
status: done
deps: [pair#574]
github_issue:
created: 2026-10-17
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4398c38934d297a35c7215a612c7df805015134e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T07:05:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.13
---

# Couch reports native agent exit without mistaking attach disconnect

## Problem

#574 cannot report a native agent exit reliably from the existing Couch observations. Console.onExit observes the Pair attachment client, and registry EOF may be a reconnect or wrapper replacement. Neither proves that the native agent stopped.

## Spec

After #574 lands, extend its lifecycle notices to confirmed native agent exit. Establish evidence from the actual owned native child lifecycle, retaining exact session/incarnation fencing and compatibility. Do not infer agent exit from attach-client exit, EOF, failed inspection or an orphaned session. Choose the smallest reliable mechanism during design; this issue does not preselect a new terminal protocol frame.

## Done when

- A confirmed current native child exit produces one hub system notice through #574's internal producer.
- Stateful tests distinguish native exit from detach, reconnect, wrapper replacement and stale completion; no false exit notice in the negative cases.
- Compatibility and an owned-child conformance check cover the chosen evidence seam.

## Plan

Durable plan: `workshop/plans/000657-native-agent-exit-notice-plan.md` (a single pass, one close).

- [x] couchmessage `native-exit` op plus ID
- [x] lifecycle `exited` kind
- [x] Couch handler, with stateful negative cases
- [x] wrapper report after Wait (self-exit only) plus compatibility
- [x] owned-child conformance test
- [x] atlas, guards, suite, close

## Log




- 2026-10-18: closed — Post-close fix 137152db for the #665 Go lint catalog: native-exit wrapper test now waits on child stdout EOF instead of a fixed sleep, and the bound-only deadline test carries a reasoned lint:allow. go test -count=3 NativeExit/ClassifyChildExit in wrapcmd green; repolint TestTrackedGoFilesPassCatalog green. (fixed-then-ship); review verdict: SHIP
- 2026-10-18: closed — Recorded suite: sdlc-local/suite success on c6d4526d; review fixes since are tested: go test wrapcmd, couchcmd, couchtty lifecycle, artifactpath and testpair green; the merge reruns the suite. BR-1: afterChildReaped (run calls it unconditionally) is driven by the real owned-child conformance test: self exit 3 gives one report; SIGHUP, SIGUSR2, a wait error and a missing session give none. BR-2: report-to-hub chain test through the production wiring and a real Console gives one hub system notice "<slot>: exited (code 3)"; a duplicate gives none. It found and fixed the slot naming, now from the admitted binding.; review verdict: SHIP
- 2026-10-18: flow upgraded quick → full — 336 added lines in code files (limit 100); an earlier round of this close already ran the full review
- 2026-10-17: Filed unclaimed at the hub's request while simplifying #574. Relevant seams: cmd/internal/couchtty/console.go:onExit (attachment client); cmd/internal/wrapcmd/wrap.go native child group.Wait (actual child); couchmessage registry session currently has no terminal child-exit observation. No implementation or protocol change is included in #574.
- 2026-10-18: Built per the TL-approved plan.
  - **Commits:** the Couch side is b20efa96 and the wrapper side 554b7b35.
  - **Conformance found:** `ownedGroup.Wait` signals the group before reaping. `run` calls it only after master EOF, when the child has already exited, so the status holds. An agent that closes its PTY but keeps running would be stopped by Wait and reported as exited (code 143); it has left the pane, so that is accepted.
  - **couchtty write budget:** run directly (outside the isolated test environment), couchtty exceeds its disk-write budget even with this change's test removed, so it is environmental. The recorded suite decides.
- **Close review round 1 (FIX-THEN-SHIP):**
  - **BR-1:** run's post-Wait decision is now one helper, `afterChildReaped`, which `run` calls unconditionally. The owned-child conformance test drives it with a real self exit (one report), SIGHUP and SIGUSR2 stops (none), a wait error (none) and no registry session (none).
  - **BR-2:** the chain test `TestNativeExitReachesTheHubAsOneLifecycleNotice` runs a real report through the handler, the production wiring (`wireNativeExitNotices`), a real Console and the publisher, ending in one system notice at the hub "<slot>: exited (code 3)". A duplicate posts nothing.
  - **The chain test found a real bug:** the fact relied on the Console having a menu row and on workspace metadata for the slot name, and it came out as ":0". Couch now passes the caller's verified slot: the name from the admitted `Binding.Slot`, the roots from the workspace identity. The Console uses that target when it has no row.
