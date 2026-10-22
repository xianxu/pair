---
id: 000810
status: working
deps: [791]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5d6dd2f9b8a3197e888f087f520abe57257a7870' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T13:41:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Couch delivery: recover resize waits without typed input

## Problem

Recurring Couch deliveries stall until operator input, before/after pair#735, perhaps twice daily. Static Claude auto/banner and bypass/banner frames and the Codex blank-line/one-warning frame replay as empty on current main; banner alone cannot explain the intermittent fault.

## Spec

Recover queued peer delivery when a real terminal resize invalidates cursor visibility and the idle agent emits no repaint. After a short bounded wait, request one harmless native redraw through the wrapper's owned child group, then re-observe. Never type recovery input, invent cursor visibility, relax composer/frame/input/picker/automatic guards, or deliver before a qualified native empty-composer repaint. Target safe recovery within5 seconds when the installed agent honors the redraw request; ignored or failed requests remain explicitly unconfirmed and safely queued.

Use the controlled resize-then-silence reproduction and sanitized native transition from #791. Its recordings cannot prove historical causation because resize events lack timestamps; TL accepts the controlled mechanism plus the operator's visible-slot/tab-wrap/keystroke evidence as sufficient to plan recovery. This replaces the earlier requirement to wait exclusively for a first failing capture. Implementation waits for #791 landing, on this independent main-based branch.

## Done when

- Controlled resize then silence recovers within5 seconds without operator bytes, for Claude and Codex stateful native fixtures and Claude batch admission; ignored/delayed/error responses preserve safe refusal.
- Installed Claude and Codex isolated conformance proves the chosen redraw mechanism produces a fresh qualified empty composer, with versions/times and sanitized provenance. If it does not, stop and revise the plan before choosing another mechanism.
- At most one request per actual resize, including repeated polls/head changes/no-op SIGWINCH; drafts, pickers, work, incomplete frames, pending human input and ended child lifetimes remain protected.
- Real process-group lifetime tests and counted8/16 growth tests prove no redraw after owner revocation/reap, no discovered PID authority, and bounded work/state. Final recorded checks and mandatory review precede landing.

## Plan

[Durable plan](../plans/000810-resize-repaint-plan.md), draft for TL approval before code.

- [ ] TL approves recovery design; parent #791 lands, then implementation gate.
- [ ] Behavioral red and native capability check through existing shared harnesses.
- [ ] Pure bounded recovery transition, child lifetime signal fence and safe re-observation.
- [ ] Safety/growth/concurrency/conformance tests, atlas, recorded verification and close/landing.

## Log

2026-10-22: Filed by pair:1 under ops:0 approval of diagnostic-only pair#791. Unclaimed because it waits for a future real capture; no active implementation assigned.


### 2026-10-22 — claimed; recovery plan from main

Authenticated TL9a7378bb-68de-4a95-bb1c-129f827a79c1 directs claim and plan now, based on the operator's observations plus the controlled resize-then-silence mechanism. Claimed pair:1; start-plan created independent branch000810-captured-idle-stall at fresh main651e30092e7ca245e5537ba1450b7ea7eb36ddae. No parent commits imported. Parent #791 is SHIP but its recorded suite is still queued: lease-1905342880/session96595, last position14. Poll it at natural pauses; checkpoint this plan and land #791 first when green. #807 remains independently checkpointed/pushed, implementation not started.

First start-plan refreshed ariadne ffd859eb5d7298b0844bc1c6876edad9b7574fd2→050fa89d520394e538938c04e18325e4c2fa299e, then refused the seeded merge-check workflow diff (ariadne#557). Per TL334838d5-215d-4e30-8be5-1d23196b1556, saved exact diff at /Volumes/TestScratch/tmp/pair810-seeded-workflow-4qhmdf3n/merge-check.patch, restored only that file and reran successfully. No local-gates change included.

Read-only discovery confirms signalChildWinch/ownedGroup/existing stdin peer poll are reusable seams. The unreaped leader is the lifetime fence; add serialized signal admission at that owner rather than PID liveness scans. Shared agentfixture honors every size notification, but real Claude2.1.308/Codex0.163.0 unchanged-geometry SIGWINCH redraw is unverified. Codex's JS launcher does not forward WINCH, so conformance must cover group signaling. No native agent launch, signal or #810 source edit has occurred. Durable plan covers bounded state, recovery/loss, one-request growth, stateful fakes and native capability stop condition. Await TL approval before code.

## Revisions

### 2026-10-22 — from capture wait to bounded repaint recovery

Original Spec waited for the first real failed-admission capture, then a recognition fix; original Done when required that first capture and its exact refusing condition. TL now accepts the controlled actual-resize→silence→cursor-hidden reproduction plus operator observations as enough to act. Current Spec/Done when replace that exclusive wait with harmless redraw and re-observation, retaining native capability proof and all input-safety guards. This is a separate recovery mechanism, not a footer recognizer change. Parent #791 dependency and land-first sequencing are explicit; no stacked development.
