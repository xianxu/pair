---
collaboration-mode: discussion
---

# Couch-owned title updates implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to choose subagent-driven development or execution in the current session. Steps use checkboxes for tracking.

**Status:** First close review returned REWORK (BR-1 activity consistency, BR-2 production resource coverage). Fixes and fresh verification in progress; landing remains pending. Live conformance deferred by TL to the next operator restart; no staging.

**Goal:** Couch-hosted Pair starts no per-slot title process; Couch updates titles from its existing inventory and activity refreshes, preserving standalone behavior, ownership and fading.

**Architecture:** Suppress the wrapper's title child for the shared Couch-hosted environment predicate. Add one supervisor-owned display updater, fed by accepted inventory observations and the existing single-flight activity schedule. Share compact invocation display facts and title decisions with the standalone poller, keeping external effects outside Console locks.

**Tech Stack:** Go, existing Couch console refresh scheduler, sessioninventory.TitleProjection, scoped Pair artifacts, Zellij and cmux command seams.

## Scope and alternatives

The TL's fresh 2026-10-17 dispatch revises #649 from investigating poller cost to replacing hosted pollers. No code before TL approval of this full-flow plan. #636 remains held for #647; this issue does not take over either.

Recommended: one updater on existing Couch refreshes. Keeping per-slot processes with lighter polling preserves their resident-memory floor and repeated liveness queries. A new standalone shared daemon or timer creates another lifecycle and independent observation schedule. Existing inventory plus activity scheduling already supplies the necessary observations and aging cadence.

This is one atomic delivery and one close review, without Mx tags. Use full flow explicitly; implementation spans more than the quick shell. No estimate until change-code's plan-quality gate passes.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| TitleTarget | cmd/internal/couchcore/title_target.go | new |
| TitleUpdateState and AdvanceTitleUpdate | cmd/internal/titlepoller/update.go | new |
| TitleLaunch | cmd/internal/sessioninventory/title_context.go | new |
| Shared frame/heat/ownership decisions | cmd/internal/titlepoller/titlepoller.go | modified |

TitleTarget is the observation-scoped tuple of thread address, agent, exact bound session, session/start generation and newest typed launch ordinal. It is projected from the same snapshot and resolved session evidence used by inventory; a live display row alone grants no target authority. Keep private authority fields out of JSON/display payloads. Include only hosted live sessions owned by this supervisor, never an unrelated standalone Pair.

TitleUpdateState records desired title, in-flight effect identity and last confirmed title per target/pane. One updater owns many bounded entries; full key includes scope, tag, session generation, launch ordinal, agent and pane id. Pane ids repeat across sessions. Extract existing frameTitle, prefixForAge, cmuxWorkspaceTitle and shouldClaimWorkspace for shared consumption, preserving their rules. Future formatting changes then have one owner (ARCH-DRY, ARCH-PURE).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Hosted wrapper title admission | cmd/internal/wrapcmd/title_process.go | modified | child pipe/process creation |
| Inventory title-target projection | cmd/internal/couchcore/actionableinventory.go and title_target.go | modified/new | existing thread snapshot, scoped binding and session presence evidence |
| Compact projection cancellation | cmd/internal/sessioninventory/title_projection.go, title_context.go and runtime_os.go | modified/new | context-aware bounded reads and SQLite child |
| Couch title updater | cmd/internal/couchcmd/titles.go | new | compact native display facts and scoped pane/owner metadata |
| Console display/activity pass | cmd/internal/couchtty/console_activity.go and console.go | modified | existing cancellable single-flight schedule |
| couchTitleIO | cmd/internal/couchcmd/titles.go | new | context-bounded Zellij/cmux effects and ownership publication |

Reuse sessioninventory.TitleProjection per exact invocation; its one poll yields activity and usage together, tails append-only native bytes and rejects supersession. Retain existing user-input activity and wrapper work when computing the final activity timestamp for both idle fade and titles. Replace the current separate native activity scan for hosted rows, rather than adding a second scan. Expose CurrentTitleLaunch as a narrow typed wrapper over sessioninventory.readCurrentLaunch(ctx, runtime, scope, tag, expectedAgent), using the shared OwnerLaunchCache. Reject agent conflicts and unavailable/latest-untyped launches; QueryResumeTargetContext is unsuitable because it may return an older established launch. Reuse OwnerLaunchCache-backed current-launch resolution; no full ledger/transcript parse every event. Awaiting/invalid ownership has no title effects.

Reuse FakeThreadArtifactCollisionChecker's session observations/counters. Extend the title Runtime fake into a stateful title world: sessions, panes, workspace owners, applied titles and injectable command failures. The production and fake updater share the same seam. Preserve the standalone poller's real child/lease tests. Store test state in private temporary directories; no production configuration required (ARCH-MOCK).

## Operating envelope and lifecycle

- Work class: optional background display maintenance. Never block input, rendering, inventory return, park or message delivery on title IO. No IO while holding Console.mu.
- One updater worker for the supervisor, coalescing refresh requests; no per-slot goroutine/process/timer. Reuse the existing 60-second activity timer and accepted listing/state changes. No added list-sessions or list-clients queries for ordinary title maintenance. Foreign-owner liveness uses the same host-wide listing; unresolved evidence defers ownership instead of probing per owner.
- Proposed initial budget: a 2-second bounded display pass, at most one command in flight, per-command cancellation within the remaining pass budget. Large fleets rotate the starting target fairly on the next existing activity pass; optional stale titles remain until then. Confirm budget with the TL against live measurements. Changed targets wake through existing refresh, stable titles issue zero renames. No new periodic native scan.
- Shared compact projection obeys existing framing and retained-state bounds; warm unchanged passes must read no transcript history. Measure total supervisor resident memory and decode work over repeated refreshes, not only removed process count (ARCH-CONSTRAINTS).
- State is in memory, capped to currently eligible targets/panes, pruned on successful inventory replacement and supersession. Failed inventory is unknown and cannot authorize effects or prove disappearance. Stop cancels and joins the sole worker and all command descendants. Restart rebuilds facts; idempotent title assignment may replay once. No new durable family. Existing cmux owner record retains storagegc publication/cleanup ownership; the updater does not acquire permission to kill legacy pollers (ARCH-FUNERAL).
- Validate session, scope, tag, launch and pane metadata through existing parsers. Title metadata cannot supply shell code; use argv. Never serialize binding nonces or native transcript text (ARCH-SECURE).

## Event model and invariants

AdvanceTitleUpdate is the only writer of updater state (ARCH-ORDER). Effect outcomes return with the exact target/generation and desired title that produced them.

| State / event | Transition and effects |
|---------------|------------------------|
| Unknown / accepted eligible target | Await facts; admit exact current launch and pane birth before effects |
| Ready / same desired title as confirmed | Retain state; no effect |
| Ready / changed title | Mark one effect in flight; issue rename with scoped session and pane |
| In flight / confirmed success for current generation | Commit confirmed cache; process coalesced latest desired state |
| In flight / failure or unconfirmed timeout | Do not cache success; retry idempotent assignment on next existing pass |
| Any / new generation, agent switch, pane replacement | Retire old entry/projection; late completion cannot update new state |
| Any / known disappearance | Retire entry and pending work; never rename the departed target |
| Any / unresolved listing or invalid authority | No effects; preserve uncertainty, re-observe on existing schedule |
| Any / shutdown | Cancel and join worker/effects; no work survives owner |

Birth gate: the exact current launch's agent pane sidecar must exist before any rename; never initiate Zellij work against a server still being created. Keep stale agent twins filtered. Validate current target again immediately before effects, using local binding/owner generation rather than a fresh host listing. A queued obsolete observation cannot retarget a successor. Title writes are idempotent; unknown outcomes allow later assignment, never destructive reconciliation.

cmux retains heat thresholds and foreign-owner semantics, including legacy tag-only owner records. A live foreign owner defers, proven absent may be reclaimed, unknown defers. Workspace publication and rename success remain separately observed; only confirmed rename updates the dedup cache. Preserve scoped owner path rules. Reuse the current inherited CMUX_WORKSPACE_ID; do not rename arbitrary desktop workspaces. Idle fading remains driven by shared activity, including wrapper work, through the existing minute scheduler even without operator input.

## Verification strategy

| Production function | Adversarial strategy and mechanical guard |
|---------------------|------------------------------------------|
| startWrapperTitleProcess | Valid hosted/standalone fixture environments; observe actual child and metadata effects through existing lease harness. |
| ProjectTitleTarget | Changing or ambiguous binding and launch evidence; stateful inventory fake asserts exact identity and fixed listing/client query counts. |
| AdvanceTitleUpdate | Generated event sequences with reordered/failed completions; independently assert current born targets only and confirmed-title deduplication. |
| CurrentTitleLaunch | Changing typed owner ledgers; shared-cache operation counts enforce latest-agent fencing without repeated history parsing. |
| TitleProjection.PollContext | Controlled cancellation barriers at IO/chunk boundaries; file-backed fake and blocked SQLite process prove joined cancellation and retry without poisoning valid state. |
| CouchTitleUpdater.Update | Changing fleets and ownership under injected failures; stateful pane/workspace world plus generation barriers enforce scoped effects, fair pass budgets and foreign-owner deferral. |
| Console.advanceActivity / finishActivity | Controlled refresh/completion ordering; fake clock and barriers enforce combined projection, fading, no IO under Console.mu, and joined shutdown. |
| CouchTitleUpdater.Update: 1,000 ticks | Fixed-fleet deterministic turnover workload; independently bound cache cardinality to current targets/panes, post-GC retained heap to warm baseline plus 4 MiB, and warm transcript-history decoding and extra liveness/title calls to zero. Use an isolated test process and retain raw heap samples. |

The old per-poller 1,000-tick acceptance is superseded by the TL-approved replacement: hosted pollers no longer exist. Preserve its retained-memory/work invariant on the single Couch updater, using the deterministic regression above. The 4 MiB allowance is a verification tolerance, not an OS RSS claim; live supervisor RSS is measured separately. Standalone behavior remains unchanged. Retain approximately ten minutes of live pre-fix RSS/tick/query measurement on one large poller; the initial 130-second fresh-poller sample is preliminary. The worker owns pre/post measurement and issue recording; the hub authorizes rollout and verifies visible behavior.

## Implementation tasks

### Task 1: Hosted child admission

Files: wrapcmd/title_process.go and title_process_test.go; stale launcher/runtime.go comment.

- [x] Write startWrapperTitleProcess regression using the strategy above; observe RED with `go test ./cmd/internal/wrapcmd -run 'Test.*Title' -count=1`.
- [x] Gate with launcher.CouchHostedEnv(os.Getenv) before allocation, verify GREEN with the same command, and commit with required trailers.

### Task 2: Shared title decisions and exact targets

Files under cmd/internal/: new titlepoller/update.go and update_test.go; titlepoller/titlepoller.go and run.go; new couchcore/title_target.go and title_target_test.go; actionableinventory.go/sessionevidence.go/artifactcollision.go where the real consumer needs binding/listing facts.

- [x] Write ProjectTitleTarget and AdvanceTitleUpdate tests using the strategies above; observe RED with `go test ./cmd/internal/titlepoller ./cmd/internal/couchcore -run 'Test.*(Title|SessionPresence)' -count=1`.
- [x] Extract shared decisions, observation-scoped targets and confirmed-only effect state. Verify GREEN, retain standalone compatibility and commit.

### Task 3: Cancellable compact queries

Files under cmd/internal/sessioninventory/: new title_context.go and title_projection_test.go; title_projection.go/title_projection_test.go, runtime_os.go/runtime.go and context runtime adapters.

- [x] Write CurrentTitleLaunch and PollContext regressions using the strategies above; observe RED with `go test ./cmd/internal/sessioninventory -run 'Test.*(Title|OwnerLaunch)' -count=1`.
- [x] Export the narrow cached current-launch projection. Add PollContext(ctx), retaining Poll() as standalone background wrapper; propagate context into owner resolution, admission, bounded tail reads and SQLite CommandContext. Cancellation leaves valid projection state retryable, authority invalidation stops it. No detached timeout goroutine. Verify GREEN and commit.

### Task 4: One combined Couch pass

Files under cmd/internal/: new couchcmd/titles.go and titles_test.go; couchcmd/run.go; couchtty/console_activity.go/console.go and console_titles_test.go; titlepoller runtime boundary where required.

- [x] Write CouchTitleUpdater.Update and Console activity regressions using the strategies above; observe RED with `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Test.*(Title|Activity)' -count=1`.
- [x] Feed accepted title targets into the existing single-flight activity worker; share compact facts, preserve user/wrapper activity and apply bounded effects. Return shared activity through MenuEventActivity. Verify GREEN and controlled worker race tests; commit.
- [x] Run the isolated 1,000-tick retained-memory/work regression and retain all samples/failures.

### Dependency checkpoint: #653

- [x] Wait for pair:2 to land #653 on main. No projection edit or unlanded stacking before that handoff.
- [x] Merge landed main into this branch, preserving #653 narrow transient recovery and #649 PollContext transaction/cancellation plus newest-owner fencing. Notify pair:2 before resolving the shared file.
- [x] Make TestCouchTitlesOSRecoversGrowthDuringAdmission GREEN, rerun affected projection/updater/console race families, push the new source head and run the recorded suite there. The canceled pre-dependency run provides no suite evidence.

### Task 5: Documentation, live evidence and landing

Files: README.md, atlas/couch.md, atlas/index.md as needed, this plan and #649 Log. TL owns project files.

- [x] Document hosted ownership and existing fade cadence; search stale per-session-poller claims.
- [x] Finish ten-minute large-poller pre-fix measurement; record fresh/large poller RSS, CPU, ticks, query rates and fleet totals in the issue. Baseline runs before any deployed fix.
- [x] Record TL-directed live deferral: do not stage pair:0 (shared restart binary). After landing, hub relaunches slots during the operator next restart; hub owns no-poller/title/fade/cmux observation and fleet/supervisor AFTER totals. Automated OS-seam tests and recorded suite plus close review are the pre-landing evidence. Live checks remain outstanding until that restart.
- [x] Run targeted/race verification, push issue head, run required recorded `sdlc local-check run --name suite`.
- [ ] Run the fresh recorded suite for BR-1/BR-2 corrections and record the result before re-closing.

Next acceptance gates: close with the recorded automated evidence and TL live deferral, resolve sole boundary review, update PR 424 and await SHIP. Then `sdlc merge --yes` and report Git-read hashes and PR to hub.

## Baseline evidence (2026-10-17)

A read-only 130-second ps sample of this slot's live poller PID 24228 observed RSS 25,392 -> 25,520 KiB and process CPU 0.08 -> 0.12 seconds. RSS changes at the two observed tick bursts were +112 KiB and +16 KiB (mean +64 KiB per tick); this short window does not establish a leak. Bursts occurred at approximately 23.86 and 84.21 seconds, consistent with 60.35 seconds/tick (0.994/minute). The first captured child was a rename of this slot's codex frame. No poller was stopped by this measurement.

Fleet at first sample: 17 title processes, 1,080,768 KiB RSS (1,055.44 MiB). At the last fleet sample, 121.06 seconds later: 16 processes, 1,053,984 KiB RSS (1,029.28 MiB); PID 45069 exited during sampling. The 16 processes present at both endpoints accrued 2.80 CPU-seconds, or 1.388 CPU-seconds/minute. The issue title's older 18 / 1.6 GB / 5.5 CPU-s/min figures are historical, not reproduced by this window. Sampler subprocess overhead is excluded from poller totals; host load and historical process versions remain confounders.

Raw local evidence: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair649-measure-iw4zuhto/measurement.json. It is a local diagnostic, not a required repository artifact. Short-lived query observation is recorded separately below when complete.

## Approval checkpoint

Full flow approved by TL delivery 798a3ab4-b9da-465f-a516-34c291210e0a; change-code passed with clean plan-quality and accepted estimate INFO. Source HEAD 85d217e1e is pushed. Next: recorded suite, close review, then landing. TL delivery 10172f8c-0c20-4200-8bfe-079e616b4400 defers actual live conformance to the operator next restart and forbids pair:0 staging.

## Revisions

- 2026-10-17 — PQ-1: retain ten-minute large-poller baseline and fleet before/after records; apply the original 1,000-tick retained-memory/work invariant to the approved replacement updater. Standalone behavior remains unchanged.
- 2026-10-17 — PQ-2: compress test-case prose across all tasks into named functions and adversarial strategies. Correct ARCH-ORDER. No runtime design delta.

- 2026-10-17 — Implementation placement: pure TitleLaunch result and CurrentTitleLaunch/context adapter share title_context.go; tests extend title_projection_test.go. Console environment wiring is titles_wiring.go. No runtime scope change.

- 2026-10-17 — TL delivery 10172f8c-0c20-4200-8bfe-079e616b4400: no pair:0 staging because it is the shared restart binary. Land after suite and close review; hub owns live title/fade/cmux checks, hosted no-poller observation and AFTER fleet totals at the operator next restart. This explicitly supersedes Task5 pre-landing staging/live ordering.

- 2026-10-17 — TL added #653 landing dependency and canceled the obsolete queued suite. Pair:2 owns narrow same-identity admission-growth/same-launch rebind recovery. #649 adds OS-seam updater regression, waits for landing, then merges and preserves both recovery and cancellation APIs before re-verifying.

- 2026-10-17 — Final #653 review semantics (pair:2 delivery 27309298-d874-4e46-9f31-ee87617550cd): supersedes earlier same-launch-rebind recovery. Ledger conflict/rebind, owner and tail failures stay terminal. Admission ErrArtifactChanged alone may re-admit when a pre-first-read artifact baseline proves all unchanged/only-grown and at least one grew. Integrate classifyAdmission/baseline into next.poll(ctx) after landing, retaining cancellation transactions; Darwin proof-fallback growth is part of the dependency test surface.

- 2026-10-18 — Integrated origin/main 3851b7338ff107d9b676d1db4eda7c7f21a2aa62 containing landed #653 a5c7ae6f718c90172911fe6e39a5aba923b158b0. classifyAdmission and pre-read baseline now run inside next.poll(ctx), preserving cancellation rollback and owner-wide newest-agent fencing. Kept incoming Console deadline, copy-key and lifecycle behavior alongside hosted activity passes. Updater growth regression and focused projection/current-owner tests passed; broader package/race and recorded suite remain pending.

- 2026-10-18 — Verification complete: recorded suite 0619b0f0bcec5545314c1d0545bc00d8 passed on 876a43a07b36fd217d02518e8241315081531575 in 10m52s, log sha256 prefix 4e8fde9764e4. The first suite found only missing artifact source inventory entries, now classified without adding exceptions. Separated the future close/landing gates from the implementation checklist so completion ticks do not claim gates that have not run; scope and runtime design unchanged.

- 2026-10-18 — Close BR-1 (shared-activity-consumer-consistency): snapshot admitted wrapper activity before the combined pass and merge it with persisted facts before both title and fade decisions; add stale-native/recent-wrapper regression for frame, cmux heat and fade, plus Console snapshot delivery. BR-2 (production-path-resource-verification): replace fake-facts 1,000-tick workload with real disposable compact projections, transcript growth and ordinal turnover, independently count warm history reads/decodes and commands, and retain heap/cache samples. Both restore already-approved acceptance; no new runtime scope. First suite pass predates these code/test corrections and must be replaced.
