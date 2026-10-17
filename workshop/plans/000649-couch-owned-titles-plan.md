---
collaboration-mode: discussion
---

# Couch-owned title updates implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to choose subagent-driven development or execution in the current session. Steps use checkboxes for tracking.

**Status:** Draft for TL approval; no implementation.

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
| CurrentTitleLaunch | cmd/internal/sessioninventory/title_launch.go | new |
| Shared frame/heat/ownership decisions | cmd/internal/titlepoller/titlepoller.go | modified |

TitleTarget is the observation-scoped tuple of thread address, agent, exact bound session, session/start generation and newest typed launch ordinal. It is projected from the same snapshot and resolved session evidence used by inventory; a live display row alone grants no target authority. Keep private authority fields out of JSON/display payloads. Include only hosted live sessions owned by this supervisor, never an unrelated standalone Pair.

TitleUpdateState records desired title, in-flight effect identity and last confirmed title per target/pane. One updater owns many bounded entries; full key includes scope, tag, session generation, launch ordinal, agent and pane id. Pane ids repeat across sessions. Extract existing frameTitle, prefixForAge, cmuxWorkspaceTitle and shouldClaimWorkspace for shared consumption, preserving their rules. Future formatting changes then have one owner (ARCH-DRY, ARCH-PURE).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Hosted wrapper title admission | cmd/internal/wrapcmd/title_process.go | modified | child pipe/process creation |
| Inventory title-target projection | cmd/internal/couchcore/actionableinventory.go and title_target.go | modified/new | existing thread snapshot, scoped binding and session presence evidence |
| Compact projection cancellation | cmd/internal/sessioninventory/title_projection.go, title_launch.go and runtime_os.go | modified/new | context-aware bounded reads and SQLite child |
| Couch title updater | cmd/internal/couchcmd/titles.go | new | compact native display facts and scoped pane/owner metadata |
| Console display/activity pass | cmd/internal/couchtty/console_activity.go and console.go | modified | existing cancellable single-flight schedule |
| Title effect runtime | cmd/internal/titlepoller/runtime.go and update.go | modified/new | context-bounded Zellij/cmux effects and ownership publication |

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

AdvanceTitleUpdate is the only writer of updater state (ARCH-STATE). Effect outcomes return with the exact target/generation and desired title that produced them.

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

## Implementation tasks

### Task 1: Lock down spawn admission

Files: modify cmd/internal/wrapcmd/title_process.go; tests cmd/internal/wrapcmd/title_process_test.go; fix stale SpawnTitlePoller comment in cmd/internal/launcher/runtime.go.

- [ ] Add hosted cases for scope only, tag only and both using valid wrapper fixtures. Assert no title child/pipe/PID publication; retain a standalone positive control.
- [ ] Run `go test ./cmd/internal/wrapcmd -run 'Test.*Title' -count=1`; new hosted cases fail before change.
- [ ] Gate startWrapperTitleProcess with launcher.CouchHostedEnv(os.Getenv) before allocation. Do not add a user switch or disable standalone title leases.
- [ ] Rerun targeted tests and commit with #649, Agent and Co-Authored-By trailers.

### Task 2: Shared decision model and exact target observations

Files: create cmd/internal/titlepoller/update.go and update_test.go; modify titlepoller.go/run.go; create cmd/internal/couchcore/title_target.go and title_target_test.go; modify actionableinventory.go/sessionevidence.go/artifactcollision.go only where the real consumer requires the existing binding/listing facts.

- [ ] Write unit tables for shared formatting/ownership and event sequences: repeated title, repeated pane id across sessions, failed rename then retry, late completion after replacement, unknown presence, pane birth and stale agent twins.
- [ ] Add inventory tests with FakeThreadArtifactCollisionChecker: exact bound session and newest typed launch; missing/ambiguous binding and superseded owner produce no target. Assert one SessionPresence query independent of fleet size and zero new client queries.
- [ ] Run `go test ./cmd/internal/titlepoller ./cmd/internal/couchcore -run 'Test.*(Title|SessionPresence)' -count=1`; observe new tests fail.
- [ ] Extract shared pure decisions and introduce the observation-scoped target. Use existing cached current-launch resolution, with bounded invalidation on source changes; never copy continuation's full-file reader into a tick path.
- [ ] Run the targeted packages and commit. Standalone Run retains cadence, lease, frame/fade thresholds and ownership behavior.

### Task 3: Make compact display queries cancellable

Files: cmd/internal/sessioninventory/title_launch.go and title_launch_test.go (new), title_projection.go/title_projection_test.go, runtime_os.go and runtime.go.

- [ ] Add tests showing cached exact-current resolution avoids ledger history reads, rejects a newer agent/conflicting generation and does not fall back to an older established launch.
- [ ] Add cancellation tests at owner lookup, native admission, tail chunks and a blocked SQLite child. Cancellation must not permanently poison a reusable projection; an invalid authority still does.
- [ ] Run `go test ./cmd/internal/sessioninventory -run 'Test.*(Title|OwnerLaunch)' -count=1`; confirm new tests fail.
- [ ] Export the narrow cached exact-launch projection. Add TitleProjection.PollContext(ctx), keeping Poll() as the standalone background-compatible wrapper; replace owner()'s context.Background use and check cancellation between bounded chunks. Route subprocess work through a context-aware runtime seam (SQLite CommandContext) and preserve existing native proof validation. No detached goroutine timeout wrapper is acceptable.
- [ ] Reuse the existing sessioninventory file-backed/stateful runtime fake to pin cancellation and resumption interleavings. Run targeted tests and commit.

### Task 4: One combined Couch display pass

Files: create cmd/internal/couchcmd/titles.go and titles_test.go; modify cmd/internal/couchcmd/run.go, cmd/internal/couchtty/console_activity.go/console.go; create console_titles_test.go; extend titlepoller runtime and tests as needed for context and explicit workspace targeting.

- [ ] Write stateful fake integration tests for accepted listing -> combined facts -> rename -> same listing/no rename; changed usage; idle transition; foreign live owner; proven stale owner; unknown owner; multiple scopes; shutdown during blocked command.
- [ ] Add controlled ordering tests for a listing replacement during the old projection/effect, coalesced requests, deadline fairness and no IO under Console.mu. Count one combined projection poll per admitted target and no duplicate activity scan.
- [ ] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Test.*(Title|Activity)' -count=1`; confirm failures.
- [ ] Feed accepted target observations through one supervisor-owned worker on the existing activity schedule. Return shared activity to existing MenuEventActivity, preserving wrapper/user-input contributions. Run context-bounded effects behind the stateful seam and commit caches only on confirmed success.
- [ ] Cancel/join on console exit; prune exact-generation projections on supersession/disappearance. Use bounded native IO and work-observer assertions to prevent historical redecoding. The preceding query task supplies PollContext and cancellable subprocess IO; preserve its cancellation classification throughout the pass.
- [ ] Rerun targeted title/activity tests, existing sessioninventory title projection tests, and race tests for new worker sequences; commit.

### Task 5: Documentation, live conformance and close

Files: README.md (hosted title behavior), atlas/couch.md, atlas/index.md as needed, this plan and #649 Log. The TL owns project updates.

- [ ] Document hosted title ownership, existing minute fade cadence and standalone operation. Search for stale claims of one poller per session.
- [ ] Repeat baseline resource measurement on the same class of live slot after staging; compare process count, RSS, CPU delta, tick/query rate and supervisor growth. Warm repeated fake refreshes assert zero extra liveness queries and no transcript-history reads. Retain every failed attempt with classification.
- [ ] Ask hub to authorize live staging/smoke; use `sdlc stage --build` from this slot, report printed revisions. Hub checks a hosted start (no title child), changed context title, unchanged dedup, idle fade and cmux ownership on actual setup. Standalone fixture/live disposable control still starts its lease child. Do not relaunch others or kill old pollers without authorization.
- [ ] Run targeted packages with race where relevant, push issue head, then `sdlc local-check run --name suite` on that pushed head. Use the declared suite and record evidence, not a manual main integration.
- [ ] Run `sdlc close --issue 649 --verified '<specific automated and live evidence>'` for the sole mandatory boundary review. Resolve blocking findings; log result. Open/update PR and await SHIP; then `sdlc merge --yes`, report PR and Git-read hashes to hub.

## Baseline evidence (2026-10-17)

A read-only 130-second ps sample of this slot's live poller PID 24228 observed RSS 25,392 -> 25,520 KiB and process CPU 0.08 -> 0.12 seconds. RSS changes at the two observed tick bursts were +112 KiB and +16 KiB (mean +64 KiB per tick); this short window does not establish a leak. Bursts occurred at approximately 23.86 and 84.21 seconds, consistent with 60.35 seconds/tick (0.994/minute). The first captured child was a rename of this slot's codex frame. No poller was stopped by this measurement.

Fleet at first sample: 17 title processes, 1,080,768 KiB RSS (1,055.44 MiB). At the last fleet sample, 121.06 seconds later: 16 processes, 1,053,984 KiB RSS (1,029.28 MiB); PID 45069 exited during sampling. The 16 processes present at both endpoints accrued 2.80 CPU-seconds, or 1.388 CPU-seconds/minute. The issue title's older 18 / 1.6 GB / 5.5 CPU-s/min figures are historical, not reproduced by this window. Sampler subprocess overhead is excluded from poller totals; host load and historical process versions remain confounders.

Raw local evidence: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair649-measure-iw4zuhto/measurement.json. It is a local diagnostic, not a required repository artifact. Short-lived query observation is recorded separately below when complete.

## Approval checkpoint

Full-flow plan awaiting TL approval. Next action after approval: `sdlc change-code --issue 649 --flow full`; respond to plan-quality findings before estimating or implementation. No implementation has begun.
