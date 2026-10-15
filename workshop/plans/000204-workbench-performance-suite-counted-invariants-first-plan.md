# Workbench counted performance invariants implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use bounded agents for the independent consumer tests; the SDLC gate owns fresh review. Steps use checkboxes for one atomic close, with no Mx boundaries.

**Status:** Proposed to the TL before implementation; the attach scope decision below must be settled first.

**Goal:** Catch repeat parsing and repeated process enumeration at the interactions that caused operator-visible stalls, using deterministic normal-suite counts.

**Architecture:** Reuse #559's distinct filesystem-read and decoder-input counters and #218's process-reader counting boundary. Exercise actual UI activity scheduling, real peer-message requests and selected Couch attachment. Keep counting support in test fixtures or instance-local observation dependencies, with native defaults and no global hooks, new cache policy, retry, background measurement service or timing gate.

**Tech Stack:** Go tests, existing Console and message-service fixtures, temporary Pair artifacts and stateful native-process/CLI doubles.

## Scope decision for the TL

Recommended delivery is the three Couch invariants in the 2026-10-14 Revision. Original #201/#202/#203 and nonblocking tier-2 timing capture are earlier, broader goals; they are not silently claimed complete by these tests. Before close, make the issue's current Done-when/Plan explicit under a Revision and carry the separate earlier goals in a released follow-up if the TL wants to preserve that backlog.

“One process-table scan per attach” must count all full enumerations, not only an initial owner capture. Main #528 now makes selected Couch attach feasible: parent listed attachment uses zero full tables; the Pair helper's ancestry check uses one. Targeted PID identity/liveness reads remain fresh and are not full-table scans. Proposed boundary: public selected Couch attach entry through the helper's blocking attach handoff, excluding the subsequent client lifetime, shutdown and independently scheduled UI inventory.

Standalone Pair attach currently has two full scans (ancestry plus orphan-editor sweep); continuation fallthrough can have three. #218's owner pass shares one initial capture but does two additional full-table SessionServers fences per caller (1+2N), so that existing test is not proof of a strict total <=1. If the requested invariant includes standalone/continuation or the owner-probe workflow, this is a runtime redesign as well as a counted suite; return to the TL before implementing that expansion. Recommend protecting the selected Couch path in this delivery, with the limitation explicit.

## Approaches

1. Extend lower-level #559 component tests only: cheapest, but production wiring can recreate a cache per tick or bypass it while those tests remain green.
2. **Recommended:** add consumer-level regressions using production adapters and existing scheduler/socket/attach fixtures. Share artifact builders and counters; count at actual read/decoder/process-source seams. Retain lower-level invalidation tests.
3. OS tracing or a separate timed benchmark runner: broader environmental dependencies and a second release path, unnecessary for these deterministic promises.

## Core concepts

No new product-domain entity or lifecycle is needed. Counter snapshots/deltas are pure values; fixture builders and observation adapters perform IO and are integration support.

| Entity | Kind | Location | Status |
|---|---|---|---|
| Count snapshots/deltas by artifact family and path | PURE | cmd/internal/workbenchperftest/counts.go | new, extracted from #559 tests |
| Twenty-owner artifact fixture | INTEGRATION | cmd/internal/workbenchperftest/fixture.go | new, extracted from existing idle fixture |
| Persistent idle activity adapter | INTEGRATION | cmd/internal/couchcmd/run.go | modified, shared construction helper |
| Recorded-message evidence adapter | INTEGRATION | cmd/internal/couchcmd/message_service.go | modified, shared production/fixture helper |
| Process-reader counted source | INTEGRATION | cmd/internal/procutil/command.go and launcher/osruntime.go | modified only as needed for instance-local counting |

The fixture package owns temporary files and counters, never consumer behavior. Existing QueryWorkObserver and SessionNameIndexCacheIO remain the only evidence counting seams. Reuse snapshotCallReader's actual table/command/identity accounting for the process source, rather than counting high-level methods or parsing duration text. Production and counted instances must use the same source boundary.

## Invariants and controls

| Consumer | Measured evidence | Warm bound | Positive control |
|---|---|---|---|
| Console idle activity pass | ledger, catalog and native transcript body bytes read and decoder input bytes | 0 separately for every unchanged pass over 20 owners x30,000,000-byte valid ledgers | cold pass positive; one owner mutation causes only that owner's work, next pass zero |
| Peer-message request | session index and any ledger/catalog/native evidence body reads/decoder inputs | 0 separately for every confirmed unchanged request with the same 20-owner scale | cold positive; one index rewrite causes only that index's work, next request zero |
| Selected Couch attach | full process-table enumerations in parent and helper before handoff | <=1 total, with successful exact attach and zero creates | ancestry source is actually invoked; an injected extra enumeration fails the count |

Fresh ready/PID files and targeted process-generation checks are security evidence, intentionally still observed. The zero-byte promise applies to unchanged historical/index/catalog/transcript bodies, not every filesystem read. Tests assert fresh security checks and stale-binding refusal separately, avoiding a false optimization by skipping authority.

## Implementation sequence

### 1. Reuse evidence fixtures and counters

Files: existing threadactivity/idle_decode_invariant_test.go, couchcmd/message_decode_invariant_test.go; new internal/workbenchperftest support and tests.

- [ ] Extract only artifact building, mutation helpers and thread-safe counter snapshot/delta operations. Keep source construction and cache ownership in production code.
- [ ] Retain separate read/decode counters and owner/path identity; cold controls must assert each relevant family is connected. Do not make zero assertions pass with empty maps or no callbacks.
- [ ] Preserve #559's ledger mutation, transcript replacement, stale index and fresh PID regressions; reduce their fixture scale if a new consumer test replaces their 20-owner scale coverage, avoiding duplicate 600MB cold parses.
- [ ] Verify existing tests unchanged in meaning before adding consumers. No independent event-state machine or broad performance library.

### 2. Exercise the actual idle UI pass and message request

Files: couchtty/console_activity_test.go or a colocated workbench invariant file; couchcmd/run.go, message_service.go, message_service_test.go and new counted invariant tests.

- [ ] Share production activity-probe construction with the test (one persistent observed threadactivity runtime). Drive the actual Console activity request/advance/finish path with existing fixtures. Await generation completion through existing in-package scheduler/result seams, not sleeps or a shortened periodic ticker.
- [ ] If package ownership prevents reuse of production probe construction, keep the adapter in its owning package and use an instance-local injected runtime plus the Console's existing public request installation path. Do not export a new product API solely to expose private menu state.
- [ ] Share production recorded-evidence closure with serviceRig and add a pre-construction authority decorator. Use its real private sockets/mailbox/wrapper clients to send unique request IDs and await submitted delivery receipts.
- [ ] Assert each warm operation independently is zero, not only a batch average. Assert cold work, one-owner mutation, the next zero pass, fresh authority and replacement refusal.
- [ ] Counterfactual: recreate/bypass the relevant cache at the production adapter and confirm each consumer regression fails; remove the mutation/control connection and confirm cold/control assertions fail. Overlays stay temporary.

### 3. Defend selected Couch attachment's total scan budget

Files: launcher/osruntime.go, procutil/command.go as necessary; new launcher/workbench_attach_invariant_test.go and existing couchcore/warm_attach_readonly_test.go.

- [ ] Reuse the native process-reader count boundary with an instance-local controlled source; cover Table, full/named snapshot enumeration and targeted identity reads distinctly. Never use a global counter or time output as the oracle.
- [ ] Run real RunLaunch selected attach with production ancestry/skip-sweep logic and a stateful controlled process/listing/handoff fixture. Assert one actual ancestry enumeration at handoff, exact supplied terminal attached and zero creates.
- [ ] Exercise public parent ResumeContextWith(WarmOnly) and StartInteractive listed attach. Count/refuse any native owner capture/server-source enumeration, retaining positive listing and PID-generation repair checks. Parent count must be zero, so combined parent/helper bound is <=1.
- [ ] Include known-live, known-dead and unknown retained helpers where the existing public fixture models them; unknown cannot become absence to save work.
- [ ] Add an extra source enumeration in a temporary counterfactual and prove the budget fails. Do not weaken effect-time generation fencing or broaden this into standalone launch changes without the TL decision.

### 4. Publish the release net

Files: atlas/workbench-performance.md (new, link from atlas/index.md), relevant Couch acceptance runbook, issue Done-when/Plan/Revisions.

- [ ] Document stable test names and one normal-suite/targeted command for all three invariant families, what counts, exclusions and how a fourth invariant supplies positive and counterfactual controls.
- [ ] Keep the Revision's live CPU check in the acceptance runbook: 20-slot fixture, restart, attaches settled, average idle CPU <5% over60s. This is operator acceptance guidance, never a CI timing threshold; no operator restart/stage is needed for this test-only delivery unless TL requests it.
- [ ] Run targeted normal/race tests, then the recorded full suite once on a pushed final source head. Finish any failed recipe's remainder with make -k and report every failure by name.
- [ ] Close through the binary-owned fresh review. On SHIP integrate main, run sdlc merge --yes, report real merge hash/PR to hub and stop.

## Constraints and ordering

ARCH-DRY: extract #559 fixtures/counters and production adapters instead of copying closures. ARCH-PURE: snapshots/deltas remain pure; filesystem/process/socket fixtures are explicitly integration. ARCH-PURPOSE: tests enter the actual consumer and prove successful behavior, not just zero work. ARCH-MOCK: controlled artifacts, existing message service and native/CLI doubles stay behind production seams. ARCH-CONSTRAINTS: fixed20 owners x30MB, bounded repeated warm operations and existing context deadlines; deterministic bytes/calls, no timing thresholds, burn loops or host-load manipulation. ARCH-SECURE: temporary fixture homes/namespaces only, fresh PID/readiness/generation checks retained. ARCH-ORDER: await completed generations and terminal receipts before measurement; mutate only after previous operation finishes; cancel/join fixture goroutines. ARCH-FUNERAL: all artifacts under testing.T.TempDir or existing cleanup-owned short socket directories; no persistent metrics or background process.

## Verification commands

- go test ./cmd/internal/threadactivity ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/launcher ./cmd/internal/couchcore ./cmd/internal/procutil -run 'TestWorkbenchPerformance|TestIdleActivityTick|TestMessageAuthorityUnchanged|TestSnapshotDiagnostics|TestSessionOwnerPass' -count=1
- Targeted -race rerun on changed packages, narrowed to the consumer/adapter cases.
- Temporary counterfactual overlays must fail the new named invariants without changing checked-in source.
- sdlc pr; sdlc local-check run --name suite; sdlc close with exact evidence; merge main; sdlc merge --yes.

There is one atomic review boundary for this suite addition. An incidental runtime defect uncovered by a strict budget is surfaced to the TL before changing scope; never relax counts or hide additional scans under a different label.
