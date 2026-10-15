# Workbench counted performance invariants implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use bounded agents for the independent consumer tests; the SDLC gate owns fresh review. Steps use checkboxes for one atomic close, with no Mx boundaries.

**Status:** TL approved227c8394 with the ratchet addition recorded below; enter implementation only after change-code.

**Goal:** Catch repeat parsing and repeated process enumeration at the interactions that caused operator-visible stalls, using deterministic normal-suite counts.

**Architecture:** Reuse #559's distinct filesystem-read and decoder-input counters and #218's process-reader counting boundary. Exercise actual UI activity scheduling, real peer-message requests and selected Couch attachment. Keep counting support in test fixtures or instance-local observation dependencies, with native defaults and no global hooks, new cache policy, retry, background measurement service or timing gate.

**Tech Stack:** Go tests, existing Console and message-service fixtures, temporary Pair artifacts and stateful native-process/CLI doubles.

## Approved scope

TL receipt dcb3f338-41ac-45ae-a098-39ff395ae578 approves the three Couch invariants and closing #204 on them. Earlier #201/#202/#203/tier2 backlog is preserved in unassigned #566. Unassigned #567 owns lowering the other attach counts.

Selected Couch attach counts all full enumerations: parent0 plus helper1, through blocking handoff, excluding later client lifetime/cleanup and independent UI inventory. Targeted PID generation/liveness reads remain fresh and do not enumerate a table.

Add normal-suite ratchets for every enumerated sibling path: standalone<=2, continuation<=3, owner pass<=1+2N for N Probe+Revalidate callers. #218's initial capture count is not a total count: every native SessionServers fence is another full SnapshotNamed enumeration. Tests count these at the actual source seam; no safety fence is skipped and no budget is silently widened.

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

- [x] Extract only artifact building, mutation helpers and thread-safe counter snapshot/delta operations. Keep source construction and cache ownership in production code.
- [x] Retain separate read/decode counters and owner/path identity; cold controls must assert each relevant family is connected. Do not make zero assertions pass with empty maps or no callbacks.
- [x] Preserve #559's ledger mutation, transcript replacement, stale index and fresh PID regressions; reduce their fixture scale if a new consumer test replaces their 20-owner scale coverage, avoiding duplicate 600MB cold parses.
- [x] Verify existing tests unchanged in meaning before adding consumers. No independent event-state machine or broad performance library.

### 2. Exercise the actual idle UI pass and message request

Files: couchtty/console_activity_test.go or a colocated workbench invariant file; couchcmd/run.go, message_service.go, message_service_test.go and new counted invariant tests.

- [x] Share production activity-probe construction with the test (one persistent observed threadactivity runtime). Drive the actual Console activity request/advance/finish path with existing fixtures. Await generation completion through existing in-package scheduler/result seams, not sleeps or a shortened periodic ticker.
- [x] If package ownership prevents reuse of production probe construction, keep the adapter in its owning package and use an instance-local injected runtime plus the Console's existing public request installation path. Do not export a new product API solely to expose private menu state.
- [x] Share production recorded-evidence closure with serviceRig and add a pre-construction authority decorator. Use its real private sockets/mailbox/wrapper clients to send unique request IDs and await submitted delivery receipts.
- [x] Assert each warm operation independently is zero, not only a batch average. Assert cold work, one-owner mutation, the next zero pass, fresh authority and replacement refusal.
- [x] Counterfactual: recreate/bypass the relevant cache at the production adapter and confirm each consumer regression fails; remove the mutation/control connection and confirm cold/control assertions fail. Overlays stay temporary.

### 3. Defend selected Couch attachment's total scan budget

Files: launcher/osruntime.go, procutil/command.go as necessary; new launcher/workbench_attach_invariant_test.go and existing couchcore/warm_attach_readonly_test.go.

- [x] Reuse the native process-reader count boundary with an instance-local controlled source; cover Table, full/named snapshot enumeration and targeted identity reads distinctly. Never use a global counter or time output as the oracle.
- [x] Run real RunLaunch selected attach with production ancestry/skip-sweep logic and a stateful controlled process/listing/handoff fixture. Assert one actual ancestry enumeration at handoff, exact supplied terminal attached and zero creates.
- [x] Exercise public parent ResumeContextWith(WarmOnly) and StartInteractive listed attach. Count/refuse any native owner capture/server-source enumeration, retaining positive listing and PID-generation repair checks. Parent count must be zero, so combined parent/helper bound is <=1.
- [x] Include known-live, known-dead and unknown retained helpers where the existing public fixture models them; unknown cannot become absence to save work.
- [x] Add standalone attach<=2 and continuation fallthrough<=3 ratchets through the same source seam, asserting successful attachment and no creates. Extend the real owner-pass capture/fence fixture to count all tables for N callers, bounded by1+2N. Do not substitute high-level capture counts for actual full enumerations.
- [x] Add an extra enumeration in temporary counterfactuals and prove strict/ratchet budgets fail. Preserve effect-time fencing; lowering budgets is the separate redesign follow-up.

### 4. Publish the release net

Files: atlas/workbench-performance.md (new, link from atlas/index.md), relevant Couch acceptance runbook, issue Done-when/Plan/Revisions.

- [x] Document stable test names and one normal-suite/targeted command for all three invariant families, what counts, exclusions and how a fourth invariant supplies positive and counterfactual controls.
- [x] Keep the Revision's live CPU check in the acceptance runbook: 20-slot fixture, restart, attaches settled, average idle CPU <5% over60s. This is operator acceptance guidance, never a CI timing threshold; no operator restart/stage is needed for this test-only delivery unless TL requests it.
- [x] Run targeted normal/race tests, then the recorded full suite once on a pushed final source head. Finish any failed recipe's remainder with make -k and report every failure by name.
- [ ] Close through the binary-owned fresh review. On SHIP integrate main, run sdlc merge --yes, report real merge hash/PR to hub and stop.

## Constraints and ordering

ARCH-DRY: extract #559 fixtures/counters and production adapters instead of copying closures. ARCH-PURE: snapshots/deltas remain pure; filesystem/process/socket fixtures are explicitly integration. ARCH-PURPOSE: tests enter the actual consumer and prove successful behavior, not just zero work. ARCH-MOCK: controlled artifacts, existing message service and native/CLI doubles stay behind production seams. ARCH-CONSTRAINTS: fixed20 owners x30MB, bounded repeated warm operations and existing context deadlines; deterministic bytes/calls, no timing thresholds, burn loops or host-load manipulation. ARCH-SECURE: temporary fixture homes/namespaces only, fresh PID/readiness/generation checks retained. ARCH-ORDER: await completed generations and terminal receipts before measurement; mutate only after previous operation finishes; cancel/join fixture goroutines. ARCH-FUNERAL: all artifacts under testing.T.TempDir or existing cleanup-owned short socket directories; no persistent metrics or background process.

## Verification commands

- go test ./cmd/internal/threadactivity ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/launcher ./cmd/internal/couchcore ./cmd/internal/procutil -run 'TestWorkbenchPerformance|TestIdleActivityTick|TestMessageAuthorityUnchanged|TestSnapshotDiagnostics|TestSessionOwnerPass' -count=1
- Targeted -race rerun on changed packages, narrowed to the consumer/adapter cases.
- Temporary counterfactual overlays must fail the new named invariants without changing checked-in source.
- sdlc pr; sdlc local-check run --name suite; sdlc close with exact evidence; merge main; sdlc merge --yes.

There is one atomic review boundary for this suite addition. An incidental runtime defect uncovered by a strict budget is surfaced to the TL before changing scope; never relax counts or hide additional scans under a different label.

## Revisions

2026-10-15 — TL approved227c8394 via dcb3f338-41ac-45ae-a098-39ff395ae578. Replaced pending scope alternatives with the approved selected-Couch strict boundary and the additional standalone2/continuation3/owner1+2N ratchets. Earlier interaction/tier2 goals move to #566; scan reduction is filed separately. No timing gate or safety-fence change is authorized.

### 2026-10-15 — plan-quality PQ-1/PQ-2 refinement

PQ-1 concrete source ownership: add procutil.Source in cmd/internal/procutil/source.go. Its zero value uses nativeReader; NewSource accepts the exported low-level Reader below, adapts it to the existing private processReader, and each Snapshot method delegates to existing snapshotCapture (including #218 diagnostics, candidate selection and identity fencing). Package Snapshot/SnapshotNamed delegate to Source{}; no process observation is cached. A pointer-backed counted Reader owns the test topology and synchronized call totals; every copy of its Source shares the exact same Reader.

```go
type Reader interface {
    Table() ([]Process, error)
    Command(int) (CommandInfo, error)
    Identity(int) (string, error)
}
type Source struct { reader processReader }
func NewSource(Reader) Source
func (Source) Table() ([]Process, error)
func (Source) Snapshot(context.Context) ([]Observation, error)
func (Source) SnapshotNamed(context.Context, string) ([]Observation, error)
```

| Production owner | Injection path |
|---|---|
| launcher.OSRuntime.Processes | InZellijPane uses Source.Table; SweepOrphanNvim uses Source.Snapshot. ObserveSessionOwner/RevalidateSessionOwner pass this Source into SessionOwnerProbe. |
| launcher.OSServerStates.Processes | ServerStates uses Source.SnapshotNamed; initial capture counts at the Reader.Table seam. |
| launcher.SessionOwnerProbe.Processes | runtime() retains explicit IO precedence, otherwise builds osSessionOwnerIO with this Source. |
| osSessionOwnerIO | SessionServers constructs newOSSessionQuiescenceOps with that Source. |
| osSessionQuiescenceOps.processes | SessionServers uses Source.SnapshotNamed. Constructor accepts an optional Source, preserving zero-argument native defaults. |
| NewSessionOwnerPass | Nil initial source derives OSServerStates with probe.Processes, so capture and live fences cannot silently split to a host-default source. |
| Scoped checker fixture | Servers=OSServerStates{Processes:source}, OwnerProbe=SessionOwnerProbe{Processes:source}; both receive the same instance. |

No SessionOwnerCapture product type is introduced; that phrase describes the existing initial capture behavior. All native reads inside the measured adapters pass through the source above. Targeted identity/command operations remain distinguishable from table enumerations and never acquire fake evidence by incrementing a high-level method count.

PQ-2 named oracle functions and strategies: Counter.RecordRead(family,owner,path,n), Counter.RecordDecode(family,owner,path,n), Counter.Snapshot(), and Snapshot.Delta(before) live in workbenchperftest/counts.go. Counter.Observer(owner) adapts QueryWorkObserver without introducing a second accounting implementation. Test record/delta arithmetic against an independent serial reference over generated adversarial update sequences; verify snapshots own independent map copies; verify controlled concurrent update totals with joins under -race. Negative deltas are not silently clamped, so reset/disconnection is visible. Consumer cold controls assert all required families are present before checking zeros.

Source.Table/Source.Snapshot/Source.SnapshotNamed tests use a stateful injected Reader through actual snapshotCapture, proving exact enumeration counts, candidate/identity evidence preservation and cancellation/error propagation without host access. The launcher and owner-pass ratchets use that same low-level Reader; adding a second enumeration must be observed even when no argv enrichment occurs. These are integration tests, not declarations of a pure core.

### 2026-10-15 — activity adapter ownership

Implementation uses couchtty.ThreadActivityProbe(dataDir,runtime) for the row-to-thread adapter, called by both wireIdleFading and the Console consumer test. This keeps the thin UI adapter in the package owning ActivityProbe and avoids exposing menu state or copying the production closure across packages. The test is the scheduler's sole owner, drives the exact request/advance/finish events used by Run's ticker, and awaits each result/reducer application through private existing channels. No sleeps, ticker shortening, trace-dependent barrier or new product control API.
