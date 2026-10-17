---
id: 000649
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours: 3.207
collaboration-mode: collaboration
card_mirror: '4132487808148cbb588b7bb022661607ad1d67b8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T11:43:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# pair title pollers: 18 processes hold 1.6 GB RSS and ~5.5 CPU-s/min (one 52 CPU-min in 68 h) for a once-a-minute list-sessions + rename

## Problem

Each hosted Pair wrapper starts a resident title poller for occasional frame/context and cmux heat updates. These duplicate liveness queries across the fleet and retain a process per slot. The issue title records earlier resource observations; the measured baseline below supersedes them for this work.

## Spec

Per the TL's fresh dispatch on 2026-10-17, Couch-hosted Pair starts no title poller. One Couch-owned updater consumes existing inventory/state-change observations and the existing activity refresh schedule, issuing scoped rename-pane only when a title changes. Share compact context/activity projection with idle fading; retain existing cmux foreign-owner checks and heat thresholds. Standalone Pair keeps its wrapper lease and poller behavior.

Use exact scope/tag/session/start generation/launch ordinal and current pane birth evidence. Unknown or superseded authority produces no title effect. Run all optional IO outside console locks and input paths, cancel and join on supervisor shutdown, and cache only confirmed successful titles. Reuse shared rendering and cached current-launch resolution (ARCH-DRY, ARCH-PURE); no extra per-slot timer/process or liveness query (ARCH-CONSTRAINTS).

Full-flow implementation plan: [Couch-owned titles](../plans/000649-couch-owned-titles-plan.md). TL approved the design and 2-second fair pass budget; pass change-code plan-quality before code. Do not cost an unapproved plan.

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
- The old per-poller 1,000-tick criterion is superseded by the approved replacement: fixed-fleet Couch updater test runs 1,000 ticks with post-GC retained heap at most warm baseline plus 4 MiB, cache cardinality bounded to current targets/panes, and zero extra warm history reads, stable-title effects or liveness calls. Measure live supervisor RSS separately; preserve standalone behavior.
- After the next restart and relaunch, the fleet's `pair title` total RSS and CPU are re-measured and recorded.

## Plan

- [ ] Complete live baseline and submit durable full-flow plan to TL.
- [ ] After approval, pass change-code plan-quality and derive estimate.
- [ ] Suppress hosted child; share exact target/title decisions and cancellable compact facts.
- [ ] Wire one existing-schedule updater and shared activity/fade results; verify stateful failure sequences.
- [ ] Stage for hub live check, document, run recorded suite, close and land on SHIP.

## Log

### 2026-10-17 — approval, review and additional baseline

TL delivery 798a3ab4-b9da-465f-a516-34c291210e0a approved the full-flow replacement and 2-second fair display pass, requested a large-poller sample and fleet before/after totals. The fresh sample grew 128 KiB total over two ticks (64 KiB mean per tick), not 128 KiB per tick.

Large poller PID 26235 preliminary 130-second observation: RSS 205,856 KiB throughout (201.03 MiB), CPU 675.11 -> 675.73 seconds (+0.62). Two actual list-sessions calls at 9.134 and 69.514 seconds: 60.38-second interval, 0.994/minute; two calls/130-second observation = 0.923/minute. No RSS growth in this short sample; it does not rule out historical growth. Raw: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair649-large-bs_7d2th/measurement.json. A ten-minute pre-fix RSS/CPU series and query observer are running before any deployed fix.

Fresh PID 24228 child observation over 66 seconds captured one real zellij list-sessions call (0.91/minute finite-window rate). Another child was observed pre-exec and cannot be classified. First observer attempt misinterpreted macOS proc_listchildpids return units; controlled child proved it returns PID count, and the corrected attempt captured the query. Empty first attempt is not evidence. Raw corrected observation: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair649-query-observation-v2.json.

Plan-quality PQ-1 verified that original ten-minute and 1,000-tick obligations remained in the issue; retained ten-minute measurement and explicitly translated the obsolete per-poller regression to the approved replacement updater. PQ-2 verified procedural case lists; all tasks now use named functions and adversarial strategy table. No code begun.

### 2026-10-17 — recovery and design

Previous-screen lookup returned unavailable. Initial main-slot6 was clean, had no commits beyond main and no active issue branch; TL confirms #645 landed and nothing was in progress. Continuing only the fresh #649 dispatch. #636 remains held until #647 lands. Claim/start-plan completed; the first start-plan refreshed ariadne and required one explicit rerun before branching.

Measured live PID 24228 over 130 seconds: RSS 25,392 -> 25,520 KiB, CPU +0.04 seconds; two tick bursts about 60.35 seconds apart, RSS changes +112 and +16 KiB. Fleet first sample: 17 processes / 1,055.44 MiB; final fleet sample 121.06 seconds later: 16 / 1,029.28 MiB (one exited). The 16 surviving processes used 2.80 CPU-seconds = 1.388 CPU-s/min. Short sampling does not establish a leak. Local raw diagnostic path is recorded in the durable plan.

Read-only subagent exploration located the actual spawn in wrapcmd.startWrapperTitleProcess (launcher no longer spawns), existing single-flight activity schedule, missing exact title authority on display rows, and absent cancellation in TitleProjection.Poll. Plan includes narrow cached current-launch projection and PollContext rather than repeated full inventory scans or uncancelled timeout goroutines. No implementation started.

### 2026-10-17

## Revisions

- 2026-10-17 — TL revised investigation into replacement: remove hosted per-slot pollers in favor of a Couch-owned updater on existing listing/state/activity observations; preserve standalone behavior, cmux ownership and idle fading. Initial durable design captures that scope.

- 2026-10-17 — Plan-quality reconciliation: retain ten-minute pre-fix and fleet after-rollout measurement; transfer 1,000-tick retained-memory/work invariant to the approved Couch updater, with explicit deterministic limits. No change to standalone scope.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is flagged stale by sdlc estimate-source.

Decomposition after clean plan-quality: issue/spec; three smaller modules (hosted admission, exact-current launch, cancellation); shared decision module; title-effect API integration; console activity integration; docs; one close review; live discovery for Zellij and cmux. Existing projection, scheduler, parsers and Go context/exec APIs cover library availability: shared module design is halved for reuse. Thorough spec multiplies base design by 0.2; familiar Go integration uses familiarity 1.0; v3.1 impl is 40% of the v2.1 table. Representative choices are smaller module base 0.15 design/0.35 impl, greenfield base 1.25/0.55 with library discount, API base 2/1, TUI base 1.25/0.65. Docs and review use corresponding table midpoints. Two external live-discovery allowances use 0.45 x 0.4 each. This is a provisional approximately 3-hour estimate, derived from those primitives, not observed elapsed time.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.2 impl=0.08
item: smaller-go-module design=0.03 impl=0.14
item: smaller-go-module design=0.03 impl=0.14
item: smaller-go-module design=0.03 impl=0.14
item: greenfield-go-module design=0.125 impl=0.22
item: api-integration design=0.4 impl=0.4
item: tui-screen design=0.25 impl=0.26
item: atlas-docs design=0.025 impl=0.05
item: milestone-review design=0.02 impl=0.14
item: real-api-discovery design=0 impl=0.18
item: real-api-discovery design=0 impl=0.18
design-buffer: 0.15
total: 3.207
```
