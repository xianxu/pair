---
id: 000660
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a1fb3b8cdcae2d92f6f7c81eab5cb1f92e5817a9' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T06:05:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Native direct Zellij session disappears after detach

## Problem

MacOS conformance intermittently loses the private direct-Zellij session at persistent reattach. First retained failure: PR422, run39105463917 attempt1, head7de3dec6684644b5114278ac92c53106d45345f2, 2026-10-18 06:57UTC. TestNativeConsoleWrapperZellij/direct-zellij-baseline waited285.86s at terminal_native_test.go:312 for the persistent frame after reattach. child.raw contained only "No session with the name ... found!"; receipts had BOOT/CPR/input and STATE=<same boot>:1 before detach. The wrapped subtest subsequently retained PID:nonce and advanced counter1 to2, before the package's five-minute alarm exhausted the remaining budget. Treat that alarm as a cascade from the first wait, not a separate causal finding.

Run: https://github.com/xianxu/pair/actions/runs/39105463917/job/117610220213

Same issue code passed conformance in runs39105228904 (d9c9bab5bb60d3cf68b753bbe079606e8139ac0f) and39105295728 (eeedca6731051353d4314ab056da6ba2b200d0f9); only documentation differs through the failed head. The terminal fixture is unchanged by #658. This establishes intermittency, not its cause.

This differs from #603's wrapped real-shell-actor discovery wait, and #361's initial layout/fixture mismatch. Here initial launch/input/state succeeded and reattach reported an absent session.

## Spec

Find why the direct fixture's server/session disappears between detach and attach. Preserve private-session ownership, actual same-process state verification and terminal assertions. Observe detach command result, original server/agent exit and socket/session liveness behind an owned seam; reproduce the ordering deterministically before fixing it. No blanket timeout bump or blind retry.

## Done when

- Retained bounded diagnostics identify why the session is gone after successful detach.
- A controlled ordering regression fails before the fix and passes after it while retaining original actor identity/state assertions.
- Native conformance passes with attempts and failure counts recorded.

## Plan

- [x] Classify the original failed job from its complete retained log and inspect the exact upstream 0.45.1 source before selecting a fix.
- [ ] Add bounded, fixture-owned detach/reattach observations and deterministic tests of diagnostic collection and early child-exit reporting; preserve all same-process/frame assertions.
- [ ] After hub approval and a native-lane grant, run one bounded private direct/wrapped conformance attempt with the observations. Reproduce the responsible ordering before proposing a behavioral fix; an unexplained green is not a fix.
- [ ] Implement the evidenced correction with a failing ordering regression, retain real native conformance, then run the normal recorded suite, close review, and landing gates.

## Log

### 2026-10-18

Filed unclaimed while verifying #658. Failed CI log and job metadata retained locally under pair658-profile.yqKPI1/ci-39105463917*; durable evidence is the public run and summary above. Two earlier greens on unchanged issue code establish the flake. #658 may make one recorded unchanged failed-job rerun under the fleet rule; any repeat is reported rather than retried indefinitely.

### 2026-10-18 — claimed investigation and corrected classification

Hub dispatch 5c39bbe0-8a21-4953-9949-255be98a93e4 verified submitted. Card was unowned; claimed pair:3 and ran start-plan from clean main. The first start-plan refreshed the ariadne tool layer and refused to branch with the old binary; the requested second invocation created this issue branch. No native lane used.

Fetched the complete original job log with `gh api repos/xianxu/pair/actions/jobs/117610220213/logs --allow-escape-sequences` into invocation-owned TMPDIR `pair660-evidence.iHo4cL/job-attempt1.log`. The older retained failed-step excerpt omitted the successful "Dump Zellij logs on failure" step. That full log materially changes the interpretation: the original server starts at 06:52:51.199, the client says Bye at 06:52:52.188, and the server emits its periodic PTY warnings at 06:53:51.312, 06:54:51.311, 06:55:51.314, and 06:56:51.312. Its plugin shutdown is 06:57:36.826, during direct-subtest cleanup after the failed wait. There is only one server start before that cleanup. Thus the attach error does not establish original-server death. The socket and agent were not independently sampled, so their exact state remains unknown. Attempt 2 of the same head passed; its retained metadata already exists. No additional retries have been run.

Read-only audit of the Pair fixture: detach command success is checked, then oldChild.Exited (after drain/reap) and endpoint disposal are awaited. Console disposal kills only the exited client's PID, not its process group. Explicit private-session kill is only in cleanup. Missing observations are the old client's wait code and exit-wire tail, the socket's state, and the original actor's exit reason. The fixture uses on_force_close=quit, but normal client Bye plus continued server activity does not support blaming that policy yet.

Exact local upstream tag `v0.45.1` (tag ref bebf50a75198fe6c434cbf34cb2ef981a1d959ca) supplies a concrete hypothesis: src/commands.rs performs session_exists and then attach_with_session_name, each probing the session socket. zellij-utils/src/sessions.rs:148-170 treats a missing/unexpected ConnStatus reply as absence. zellij-server/src/lib.rs reuses the lowest free ClientId, removes ConnStatus clients immediately, and route.rs queues another RemoveClient when its router exits. A late cleanup for probe/client N could therefore remove a replacement connection with the reused numeric ID. This resembles #640/#647's upstream client-ID family; it is source-supported, not a reproduction or proof of this incident. Do not weaken identity or retry attach based on it.

Proposed first stage is diagnostic, not a speculative behavioral fix (ARCH-PURPOSE, ARCH-ORDER): observe the owned boundary and retain unknown outcomes. Reuse Child.Wait/Exited and existing testpair environment/evidence helpers (ARCH-DRY); use deterministic fake child/process outcomes for the diagnostic policy (ARCH-PURE, ARCH-MOCK). Limit each diagnostic stream to a bounded tail, and all new logs to fixture scratch removed only after owned children join (ARCH-CONSTRAINTS, ARCH-FUNERAL). No user sessions or global process sweep, credentials, new runtime configuration, or production lifecycle changes (ARCH-SECURE). The native grant request will name exactly one sequential direct/wrapped family attempt, its five-minute test cap, and cleanup of only its private session. No native use before that grant.

## Revisions

### 2026-10-18 — investigate failed discovery before assuming session death

The original Problem/Spec/Done-when wording calls the session gone. Complete retained server logs show it stayed active until fixture cleanup. Preserve those original acceptance obligations, interpreting "why gone" as why attach's discovery returned absence; confirm actual socket/server/agent state before changing behavior. Await hub approval of this diagnostic-first plan before code.
