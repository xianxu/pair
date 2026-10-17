---
id: 000649
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '31fc67d3784a9bc018c24b15e49f377573408618' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T11:43:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# pair title pollers: 18 processes hold 1.6 GB RSS and ~5.5 CPU-s/min (one 52 CPU-min in 68 h) for a once-a-minute list-sessions + rename

## Problem

Each hosted Pair wrapper starts a resident title poller for occasional frame/context and cmux heat updates. These duplicate liveness queries across the fleet and retain a process per slot. The issue title records earlier resource observations; the measured baseline below supersedes them for this work.

## Spec

Per the TL's fresh dispatch on 2026-10-17, Couch-hosted Pair starts no title poller. One Couch-owned updater consumes existing inventory/state-change observations and the existing activity refresh schedule, issuing scoped rename-pane only when a title changes. Share compact context/activity projection with idle fading; retain existing cmux foreign-owner checks and heat thresholds. Standalone Pair keeps its wrapper lease and poller behavior.

Use exact scope/tag/session/start generation/launch ordinal and current pane birth evidence. Unknown or superseded authority produces no title effect. Run all optional IO outside console locks and input paths, cancel and join on supervisor shutdown, and cache only confirmed successful titles. Reuse shared rendering and cached current-launch resolution (ARCH-DRY, ARCH-PURE); no extra per-slot timer/process or liveness query (ARCH-CONSTRAINTS).

Full-flow implementation plan: [Couch-owned titles](../plans/000649-couch-owned-titles-plan.md). Await TL approval before change-code/code. Do not cost an unapproved plan.

Measured by pair:2, 2026-10-17 ~10:50–11:00, across the live fleet (22 slots). The `pair title` pollers (`titlepoller`, which keeps each zellij frame title current on a 60 s tick) are:
- 18 processes holding **1.6 GB RSS** in total, the largest about 185 MB each;
- using about **5.5 CPU-seconds a minute** together;
- one, for 1-pair-6, has used **52 CPU-minutes over 68 hours**;
- 12 of the 18 are reparented to launchd. That's normal for a detached poller, but some may outlive their session.

The work per tick is one `zellij list-sessions --no-formatting`, sometimes a liveness read, and a `rename-pane` on change. That should cost a few MB and a few CPU-milliseconds a minute. The memory and CPU point to a leak (memory growing per tick) or over-polling (ticks or retries much faster than 60 s).

#617 fixed the pollers' lifetime with a lease. This issue is about their steady-state cost while alive.

## Done when

- Hosted wrapper fixtures (either Couch thread variable) create no title process; standalone positive controls retain the leased poller.
- One Couch-owned updater refreshes frame context and cmux heat from existing observations/schedule, with zero additional list-sessions/list-clients queries for normal maintenance and zero renames for stable confirmed titles.
- Activity and usage share one compact projection per exact invocation; warm unchanged passes do not reread transcript history. Existing idle fading includes user and wrapper work.
- Fake-backed sequence tests cover stale generations, repeated pane ids, missing pane birth, rename failure/retry, unknown liveness, foreign cmux owner, cancellation, bounded cache retirement and no UI blocking.
- Before/after live measurements quantify poller RSS/CPU/tick/query rate and fleet totals; the hub verifies staged hosted titles, fade and ownership on the real setup.
- Required recorded suite and sole close boundary review pass; documentation explains hosted ownership and standalone behavior.

- Measured cause: the RSS growth per tick and the tick and query rate of one live poller over ~10 minutes, before the fix.
- The fix keeps a poller's RSS flat over 1,000 simulated ticks, with a test, and its CPU at about one tick's work a minute.
- After the next restart and relaunch, the fleet's `pair title` total RSS and CPU are re-measured and recorded.

## Plan

- [ ] Complete live baseline and submit durable full-flow plan to TL.
- [ ] After approval, pass change-code plan-quality and derive estimate.
- [ ] Suppress hosted child; share exact target/title decisions and cancellable compact facts.
- [ ] Wire one existing-schedule updater and shared activity/fade results; verify stateful failure sequences.
- [ ] Stage for hub live check, document, run recorded suite, close and land on SHIP.

- [ ]

## Log

### 2026-10-17 — recovery and design

Previous-screen lookup returned unavailable. Initial main-slot6 was clean, had no commits beyond main and no active issue branch; TL confirms #645 landed and nothing was in progress. Continuing only the fresh #649 dispatch. #636 remains held until #647 lands. Claim/start-plan completed; the first start-plan refreshed ariadne and required one explicit rerun before branching.

Measured live PID 24228 over 130 seconds: RSS 25,392 -> 25,520 KiB, CPU +0.04 seconds; two tick bursts about 60.35 seconds apart, RSS changes +112 and +16 KiB. Fleet first sample: 17 processes / 1,055.44 MiB; final fleet sample 121.06 seconds later: 16 / 1,029.28 MiB (one exited). The 16 surviving processes used 2.80 CPU-seconds = 1.388 CPU-s/min. Short sampling does not establish a leak. Local raw diagnostic path is recorded in the durable plan.

Read-only subagent exploration located the actual spawn in wrapcmd.startWrapperTitleProcess (launcher no longer spawns), existing single-flight activity schedule, missing exact title authority on display rows, and absent cancellation in TitleProjection.Poll. Plan includes narrow cached current-launch projection and PollContext rather than repeated full inventory scans or uncancelled timeout goroutines. No implementation started.

### 2026-10-17

## Revisions

- 2026-10-17 — TL revised investigation into replacement: remove hosted per-slot pollers in favor of a Couch-owned updater on existing listing/state/activity observations; preserve standalone behavior, cmux ownership and idle fading. Initial durable design captures that scope.
