---
id: 000684
status: working
deps: [pair#683]
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '991e2d200aedeebccdbec3e2a6a038388bd64160' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T19:49:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# fresh fallback after a failed resume archives the old thread without tearing down its actor and session: two live actors, Couch unusable

## Problem

On 2026-10-18 the operator relaunched parley.nvim:0 (codex thread 1-parley-nvim-37, zellij session 📁1-188 since Oct 11). The park worked (18:21:47, parked-scrollback-1-parley-nvim-37-20261018T182147.raw saved, agent stopped), but the resume did not, and Couch fell back to start fresh (18:21:57, mode fresh, selected_key fresh:1-parley-nvim-37:143), archived thread 37 (18:21:58) and started 1-parley-nvim-70 in a new session 📁1-311 (18:22:01).

Archiving thread 37 did not tear down what held it: the Couch actor couch-393ac172 (pair resume 1-parley-nvim-37, pid 60305) stayed attached, and the 📁1-188 zellij server (pid 6463, with its nvim 6467/6471 and a pair title 17468 from Oct 14) kept running. Couch then had two live actors on parley.nvim:0; both tabs showed raw thread names instead of :0, the operator could not switch to any regular slot, and had to kill Couch to recover. Investigation by the operator's pair-dev session; full notes in ops qa 2026-10-18-18-couch-hang-bug.md.

## Spec

A thread is archived only after its actor and terminal session are proven gone (the same proof-of-death standard #651 uses for a stranded park). The start-fresh fallback after a failed resume must run the same teardown as a normal relaunch before starting the new thread; if teardown cannot be proven, refuse with a visible reason instead of starting a second live actor on the slot. A slot must never present two live actors; if it ever observes two, it reports it rather than mislabelling tabs. Related: #683 (failure reasons must persist; this resume's failure left no launch-diagnostics entry either), #681.

## Done when

- A regression reproduces: resume fails, fresh fallback, old actor and session still alive; today two actors result, after the fix teardown completes first or the fresh start refuses.
- Archive refuses while the actor or session is alive.
- The operator's slot switcher keeps working through a failed relaunch (live check on a disposable slot).

## Plan

Durable plan: [workshop/plans/000684-…-plan.md](../plans/000684-fresh-fallback-after-a-failed-resume-archives-the-old-thread-without-tearing-down-its-actor-and-session-two-live-actors-couch-unusable-plan.md). This issue is the vehicle for the ops project `slot-lifecycle` (operator direction 2026-10-18): one mechanism (`Converge`) over three primitives (proven teardown via a per-slot lifetime lease; bind by the transcript ID stored on the record; a persisted failure record at the dispatcher).

Prerequisite: #683 lands P3 (failure persistence) first; #684's code starts from main after it (no stacking).

- [ ] M1 — Transcript binding on the record; Resume binds by one stat (P2)
- [ ] M2 — Lifetime lease (pair processes only), agent tty witness, `Teardown` + `TeardownProof`; archive/replace require proof (P1)
- [ ] M3 — `Converge`; paths 1, 2, 3, 5, 6, 7, 8, 9, 10 onto it (fixes this issue's incident)
- [ ] M4 — Paths 4, 11, 14, 15, 16; three-outcome classification with the two-actor anomaly
- [ ] M5 — Lifecycle drill on a disposable slot with aged state, hub-triggered; 7 green nights

## Log

### 2026-10-18

- Claimed as the slot-lifecycle design vehicle (hub delivery 011ca7a4). Read the project file, audit.md (16 paths, D1–D20), targets slot-continuity and slot-recovery, #683. Spot-checked the seams the design rests on: `DispatchOperation` is the single op entry (run.go:616/632/982, recover_action.go:176); `ObserveSlotSessions` is the slot-wide witness; the LaunchFailure ring is 16 entries; strictjson makes a new record field forward-compatible only; the wrapper spawns the agent and session-watch via `exec.Command` (ExtraFiles available). Key decision: a per-slot lifetime lease (shared flock held by every pair process of the tree, inherited by the agent and helpers) as the single resource proving teardown (tl-worker PLANNING rule), with lsof naming survivors. Plan written; awaiting operator approval via the hub before any code.
- Hub review (delivery 13cead0d): the lease must not reach the agent's pty child, since everything the agent runs would inherit it. The sweep could then kill an in-flight landing or a daemon, and a setsid'd holder would wedge teardown. Revised P1 into three witnesses: the lease for pair's own processes (O_CLOEXEC, passed only to pair children); the agent's controlling tty for the agent and its in-flight tool commands; zellij absence for the terminal. Agent children that left the session are named in the proof, never killed or required gone. Hub leanings folded in: M1 runs as #683 first; no launchd, the drill is hub-triggered; the drill CODEX_HOME symlinks auth.json.
- Operator approved the design with five changes (hub delivery 90271deb), folded in as a plan Revision: Shift+Alt+N untouched; plain pair unchanged without COUCH_* env (new test); drill aged state = last 2 weeks of session files; LeftRunning persisted to an orphan registry surfaced by couch --resources / --reap-orphans; old-build refusal accepted. Open question raised to the hub: a crash after an in-pane context reset would resume the pre-reset conversation; proposed one exact read of the tag's ledger file in Bind.

- 2026-10-18 22:30 — pair:5 lifecycle evidence handoff received (delivery 566735e7): it confirms #683's retained shapes (success completions, owners and sessions absent, archived while awaiting_completion at revs 686/547) and #684's timeline as reported. Note for M3's regression: the surviving session 1-188 is operator-owned, so build the regression from fakes, never from that session. The original park error text is unrecoverable, so the design must not depend on knowing it.

## Revisions

- 2026-10-18 20:30 — scope widened from the Fresh-fallback fix to the slot-lifecycle mechanism, on the hub's dispatch (operator direction). The original Done-when items stay and are delivered by M3 (archive refuses while alive) and M4 (the regression; the switcher through a failed relaunch). Added: every audit path composed of the three primitives (M1–M5), and the nightly drill (M6).
- 2026-10-18 21:00 — P1 revised on the hub's review (lease limited to pair processes; agent proven by its pty; detached agent children named, not killed). Failure persistence moved out to #683 as a prerequisite; milestones renumbered M1–M5; drill trigger is the hub, not launchd.
- 2026-10-18 21:40 — operator approval with five changes (see the plan's Revisions): path 12 dropped from M4; Couch-only activation of lease/tty; bounded drill aged state; orphan registry. Code waits for #683.
- 2026-10-18 22:10 — operator: additive record compatibility instead of old-build refusal; plan §2a (tolerant + preserving decoders, no version bump, reasons stated).
- 2026-10-18 23:00 — introduce-team scope addition (hub delivery da27249b): the lifecycle address has two kinds, slot (repo:N) and team hub (team folder with a committed .hub, no sdlc claims). Primitives and Converge take `LifecycleAddress`; type and tests land in M1. Type only: no milestone grows, hub addressing/startup/switcher stay in pair#694.
