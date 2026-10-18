---
id: 000649
status: codecomplete
deps: ["#653"]
github_issue:
created: 2026-10-17
updated: 2026-10-18
estimate_hours: 3.207
collaboration-mode: collaboration
card_mirror: '846e4b085655101898f15c9c072b95aacbe4cf21' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T11:43:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 3.80
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
- BEFORE live measurements quantify poller RSS/CPU/tick/query rate and fleet totals. Per TL delivery 10172f8c-0c20-4200-8bfe-079e616b4400, AFTER fleet totals and actual hosted no-poller/title/fade/cmux checks belong to the hub at the operator next restart, after landing; no pair:0 staging. Pre-landing evidence is the recorded suite, OS-seam fixtures and close review.
- Required recorded suite and sole close boundary review pass; documentation explains hosted ownership and standalone behavior.

- Measured cause: the RSS growth per tick and the tick and query rate of one live poller over ~10 minutes, before the fix.
- The old per-poller 1,000-tick criterion is superseded by the approved replacement: fixed-fleet Couch updater test runs 1,000 ticks with post-GC retained heap at most warm baseline plus 4 MiB, cache cardinality bounded to current targets/panes, and zero extra warm history reads, stable-title effects or liveness calls. Measure live supervisor RSS separately; preserve standalone behavior.
- After the next restart and relaunch, the hub re-measures and records fleet `pair title` total RSS and CPU, plus hosted title/fade/cmux observations (explicitly deferred post-landing by TL).

## Plan

- [x] Complete live baseline and submit durable full-flow plan to TL.
- [x] After approval, pass change-code plan-quality and derive estimate.
- [x] Suppress hosted child; share exact target/title decisions and cancellable compact facts.
- [x] Wire one existing-schedule updater and shared activity/fade results; verify stateful failure sequences.
- [x] Integrate landed #653, make updater growth-recovery regression green, run the required recorded suite, and prepare PR/close evidence. Close review and landing follow as acceptance gates; hub performs deferred live checks and AFTER fleet measurement at next operator restart.

- [x] Verify BR-1/BR-2 corrections with the fresh recorded suite and prepare renewed close evidence.

## Log


- 2026-10-18: closed — Post-review recorded suite e870e668582e734772bf8b6d58f866e5 passed on aec1efb1b224fba130faa7ae9ec5e41cfaf863b5 (13m47s). BR-1 stale-native/recent-wrapper regression now drives frame, cmux heat and fading from one wrapper snapshot; Console delivery and targeted races pass. BR-2 actual two-target file-backed projections complete1000 ticks with50 transcript appends,10 ledger turnovers,940 zero-history-work stable ticks,2 current entries,62 scoped writes and retained heap below warm+4MiB. All affected packages, artifact/storage inventory and growth recovery tests pass. BEFORE fleet/poller RSS CPU tick/query evidence is recorded. TL explicitly deferred actual hosted no-poller/title/fade/cmux and AFTER fleet totals to hub next operator restart, with no pair:0 staging.; review verdict: SHIP
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

- 2026-10-17 — TL deliveries 2a740922-b8ca-4002-a3e4-7bee1ec191d2 and 750adbca-ede5-45bf-9fdf-799ce6db93c5 add #653 as a landing dependency. Do not edit the shared projection until synchronized; #653 lands first, #649 then merges landed main and verifies recovery through PollContext. Cancel the obsolete queued suite.

- 2026-10-17 — TL delivery 10172f8c-0c20-4200-8bfe-079e616b4400 supersedes pre-landing staging/live verification: pair:0 remains untouched; land on suite and close review. Hub owns post-landing no-poller/title/fade/cmux and AFTER fleet totals at the operator next restart.

- 2026-10-17 — TL revised investigation into replacement: remove hosted per-slot pollers in favor of a Couch-owned updater on existing listing/state/activity observations; preserve standalone behavior, cmux ownership and idle fading. Initial durable design captures that scope.

- 2026-10-17 — Plan-quality reconciliation: retain ten-minute pre-fix and fleet after-rollout measurement; transfer 1,000-tick retained-memory/work invariant to the approved Couch updater, with explicit deterministic limits. No change to standalone scope.

- 2026-10-18 — Close BR-1 corrects shared activity ordering: supply admitted wrapper work before both title and fade consumers. Close BR-2 replaces fake-facts resource acceptance with real file-backed projections, growth and turnover. These restore the approved contract; no runtime scope expansion.

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

## Implementation evidence

### 2026-10-17 — automated behavior

Full-flow change-code passed: plan-quality clean; estimate-quality INFO accepted (cancellation/integration/live verification allowances flagged for actuals). Shared facts, reducer and hosted admission committed in 59f7dfba3. Agent tasks supplied RED/GREEN evidence; parent reran wrapper title tests and title/sessioninventory/current-owner families successfully. Combined console/activity tests and targeted race tests for couchtty, couchcmd and sessioninventory passed. Broader affected-package checks remain running; no full-suite or live success is claimed yet.

Production-seam fixture reads a disposable Codex transcript, emits exact scoped rename argv and proves 20 unchanged updates read/decode zero native history and issue no extra title/liveness command. Missing selected pane/stale agent twin does not rename. Controlled command cancellation joins its group. Console tests exercise replacement cancellation and private-target snapshot copying. Isolated 1,000-tick workload holds two targets across ten ordinal turnovers: exactly 22 expected assignments, zero stable-tick writes, retained heap 1,027,592 -> 989,160 bytes. Raw post-GC heap samples were logged every 100 ticks.

### 2026-10-17 — completed large-poller baseline

Before deploying any fix, PID 26235 was measured for 600 seconds: RSS 205,856 -> 206,048 KiB (201.03 -> 201.22 MiB). The only observed RSS change was +192 KiB at elapsed 477.33 seconds; averaging over approximately ten ticks gives 19.2 KiB/tick, not evidence of a sustained linear leak. CPU 676.05 -> 679.26 seconds (+3.21 seconds, approximately 0.321 CPU-s/min). No historical growth cause is inferred from this window. Local raw series: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair649-tenminute-8mk6vyzy/measurement.json.

A separate 600-second high-frequency child observer on the same PID captured ten list-sessions --no-formatting commands and no list-clients commands (1.000 queries/minute). Tick intervals: mean 60.497 seconds, range 60.275–60.950 seconds. This reproduces the intended cadence, not over-polling. Observer windows overlap but do not share identical endpoints. Local raw query series: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair649-large-queries-600.json.

Fleet BEFORE remains the recorded 17 pollers / 1,055.44 MiB initial sample and 16 / 1,029.28 MiB later sample as one exited; survivors used 1.388 CPU-s/min. Fleet AFTER is pending an approved staged rollout; an old running wrapper retains its old poller and cannot count as upgraded. Hub request c8ee5589-0593-4894-aa2d-9a7f812c8cea is queued awaiting image input to clear. No :0 mutation or slot relaunch performed by this worker.

### 2026-10-17 — live verification deferral

TL delivery 10172f8c-0c20-4200-8bfe-079e616b4400 explicitly forbids pair:0 staging (shared restart binary), directs landing on the recorded suite and close review, and takes ownership of no-poller/title/fade/cmux plus AFTER fleet measurements at the operator next restart/relaunch. This supersedes the earlier queued staging request and pre-landing live-check ordering. No live result or measured fleet reduction is claimed. The standing per-process size, rather than a large observed per-tick leak, motivates the approved replacement.

### 2026-10-17 — affected packages and race verification

Broader checks passed: go test ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/titlepoller ./cmd/internal/sessioninventory ./cmd/internal/wrapcmd -count=1. Slowest package was couchcore at 571.950s under fleet IO load; no test failed. Targeted race families for couchtty/couchcmd/sessioninventory passed. Storage ManagedIOInventoryAnchors passed after adding the new existing-family publication edge. Recorded full suite remains queued under the two-suite machine gate on source commit 85d217e1e.

### 2026-10-17 — dependency and canceled suite

TL ordered #653 first. Pair:2 owns the projection change, narrowly treating same-identity admission growth and same-launch authorized root rebind as transient; identity/shrink/boundary/invalid-proof failures remain terminal. #649 has not edited title_projection.go since this coordination. Its existing PollContext transaction/cancellation and newest-agent fencing will be preserved when integrating the landed fix.

Canceled obsolete queued run 9a8db73ed1bffc246eddc65352d76333 on source 85d217e1e by matching the run inflight owner (PID 54033) and sending SIGINT. sdlc exited with suite lock interrupted: context canceled; owner/wait children gone. No tests in that recorded attempt ran. This was directed cancellation, not a flaky test failure; a new run is required after #653 integration.

New updater-level OS-seam test TestCouchTitlesOSRecoversGrowthDuringAdmission deterministically appends usage43k after the native boundary read, before final admission metadata check. It currently fails as expected (RED: no command ever emitted after three updater calls), while the existing stable source-to-command conformance test passes. Only couchcmd/titles_test.go changed; shared projection remains untouched. This test must pass after landed #653 is integrated.

### 2026-10-17 — final #653 classification before landing

Pair:2 delivery 27309298-d874-4e46-9f31-ee87617550cd supersedes its earlier transient classification. Same-launch rebind is a ledger conflict and remains terminal. Only an admission ErrArtifactChanged is retryable, and only when classifyAdmission proves every pre-read baseline artifact is unchanged or same-identity grown, with at least one growth. The baseline is collected before the first validation read (proof artifact list, or Observe+Select for proofless binding); this includes growth inside proof validation fallback on Darwin empty-generation-token files. Owner and tail errors remain terminal. #649 will preserve that exact classification inside next.poll(ctx), together with its cancellation rollback and newest-agent fencing, after confirmed #653 landing. At this observation #653 is codecomplete and not landed; no shared projection edit was made.

### 2026-10-18 — landed dependency integration

Hub and pair:2 confirmed #653 landed as PR 415, merge a5c7ae6f718c90172911fe6e39a5aba923b158b0; Git verified that commit and origin/main 3851b7338ff107d9b676d1db4eda7c7f21a2aa62. Integrated main with classifyAdmission/baseline-before-first-read inside next.poll(ctx), retaining cancellation rollback and newest-owner/agent fences. Console conflicts preserve incoming per-row probeTimeout, copy-key and lifecycle observations alongside the hosted combined pass.

TestCouchTitlesOSRecoversGrowthDuringAdmission is now GREEN. Focused TitleProjection/CurrentTitle/updater-growth tests passed in sessioninventory and couchcmd; Console TitleActivity/Activity/deadline/Lifecycle/CopyKey tests passed. Affected-package and race checks are running; recorded suite remains required on the pushed integrated head. No live rollout or AFTER fleet result is claimed.

### 2026-10-18 — integrated-head verification

Source 8641e428d2b85044173053dcfda33dcf273267eb is pushed; PR 424 opened. All six affected packages passed after integration: couchcore (308.783s), couchtty (45.624s), couchcmd (65.390s), titlepoller (1.279s), sessioninventory (6.490s), wrapcmd (72.766s). Targeted race families Title/CurrentTitle/OwnerLaunch/Activity passed for couchtty, couchcmd and sessioninventory. ManagedIOInventoryAnchors passed. Required recorded suite c0796f4dfe80368cf357507b001784e5 is queued on this source head; no recorded full-suite pass yet.

### 2026-10-18 — recorded-suite artifact inventory defect

Recorded suite c0796f4dfe80368cf357507b001784e5 on 8641e428d found TestProductionArtifactReferencesAreExactlyClassified failing: five new production files were absent from the exhaustive artifact inventory. Root cause was omitted source classification, not a runtime title failure. Added ResolvedConsumer entries for couchcmd/titles.go and couchcore/title_target.go, and NonArtifactSources entries for titles_wiring.go, title_context.go and titlepoller/update.go; no known-violation allowance added. Agent reproduced the five failures, then full artifactpath package passed (12.820s). The original suite continues to completion to retain all other results; replacement recorded suite is required after this source fix.

First recorded attempt completed: c0796f4dfe80368cf357507b001784e5, source 8641e428d, exit2 after 12m17s. The sole test failure was TestProductionArtifactReferencesAreExactlyClassified in artifactpath; every other shell/editor/Go test completed successfully. Retained failed log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/c0796f4dfe80368cf357507b001784e5.log (sha256 prefix f2eea6b976dc). Inventory fix is pushed as 876a43a07b36fd217d02518e8241315081531575; parent independently reran the failing test successfully (1.533s). Replacement recorded run 0619b0f0bcec5545314c1d0545bc00d8 is queued on that fixed source. This is a reproducible branch defect with a source fix, not a flaky unchanged retry.

### 2026-10-18 — required suite passed; close ready

Replacement recorded run 0619b0f0bcec5545314c1d0545bc00d8 passed on 876a43a07b36fd217d02518e8241315081531575: exit0, 10m52s, log sha256 prefix 4e8fde9764e4. Full log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/876a43a07b36fd217d02518e8241315081531575/suite.log. PR 424 is open. No newer main integration was chased after this pass. Next: mandatory close review and landing; real hosted no-poller/title/fade/cmux and AFTER fleet observations still belong to the hub at the next operator restart.

- 2026-10-18 — Checklist bookkeeping: separate future acceptance gates from verified implementation; no scope change.

### 2026-10-18 — close review REWORK

First close review found BR-1 Critical (shared-activity-consumer-consistency): titles used persisted facts before Console merged admitted wrapper work into fading. Confirmed by a stale-native/recent-wrapper regression: before the fix the returned fade timestamp stayed one hour old. The worker now snapshots wrapper evidence once before the pass; UpdateWithActivity merges it before frame eligibility and cmux heat, returning that same activity to fading. Focused updater and Console tests passed.

BR-2 Important (production-path-resource-verification): the prior 1,000-tick fixed-facts fake bounded reducer retention but never constructed TitleProjection, so its heap/count claims did not establish the specified combined production path. Replacing it with actual OS-backed compact projections, growing transcripts, ledger ordinal turnover and a stable neighbor, with retained heap/cache and per-tick native-work/command assertions. Review remains REWORK; required suite and close must rerun after these corrections. The earlier production-resource acceptance claim is superseded by this finding until the new workload passes.

BR-1 correction passed updater/Console focused checks and targeted races. BR-2 replacement passed in a fresh parent-run isolated child: 1,000 ticks, two real compact projections, 50 x128KiB appends (6,562,109-byte final transcript), ten physical ledger launch/binding turnovers (ordinal1→21), 940 stable ticks with zero native history read/decode delta. Exactly62 scoped commands (steady neighbor1), exactly2 current entries/one desired title each, neighbor projection identity/history counters unchanged. Parent heap samples: warm1,360,888 bytes; all ten turnover samples1,289,864–1,314,760; final1,169,328, within warm+4MiB. All sample/counter lines retained in test output. Affected couchcmd/couchtty/artifactpath package run passed (47.551s/40.412s/5.940s). Fresh full recorded suite remains required after the code/test fix.

Review corrections pushed as aec1efb1b224fba130faa7ae9ec5e41cfaf863b5. Required recorded suite e870e668582e734772bf8b6d58f866e5 is queued on that exact source. Targeted post-fix races passed couchcmd (10.651s) and couchtty (3.738s). No passing pre-fix suite is reused as verification of the review changes.

### 2026-10-18 — post-review required suite passed

Recorded suite e870e668582e734772bf8b6d58f866e5 passed on aec1efb1b224fba130faa7ae9ec5e41cfaf863b5: exit0 in13m47s, log sha256 prefix6537e3c5be5a. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/aec1efb1b224fba130faa7ae9ec5e41cfaf863b5/suite.log. This includes the stale-native/recent-wrapper title/fade regression, actual two-target 1,000-tick resource workload, all artifact classifications and the full remaining suite. Ready to repeat close review for BR-1/BR-2 disposition; live rollout/AFTER totals remain deferred to hub next restart.
