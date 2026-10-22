---
id: 000792
status: codecomplete
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours: 4.033
collaboration-mode: collaboration
card_mirror: '5a486f262949a815585a8a92ece81449c14c6213' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T00:59:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 8.71
---

# Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans

## Problem

The resource monitor has positive prior-incarnation provenance, but the confirmed orphan command does not consume that group. Reusing its expanding tree reaper for an old root could also signal a current live descendant. The fix must preserve exact process scope and put the current authority/ownership veto beside each signal.

## Spec

10-21 ~20:15: the operator restarted echo:2 (hub ops-echo). `couch --resources` then listed **proven orphans** of its
previous incarnation: zellij 66561, pair 66575 and 66576 (`pair wrap`, `pair term`), nvim 66577 and 66583, and Claude
66695 (~350 MB RSS). No verb ends them:
- `couch --reap echo:2 --confirm` refuses `not-offered: echo:2 is live` (it would target the live slot);
- `couch --reap-orphans --confirm` covers only unattributed embedded nvim;
- from a different hub, `--reap` refuses with "slot belongs to hub …" (correct).

Fix: `--reap-orphans --confirm` (run by the authenticated owning hub, or the unbound local operator) also ends **proven orphans of a
previous incarnation** of a live slot, the set `--resources` already proves, each signal fenced on the process
identity the sweep recorded, never touching the live incarnation's processes. Report what was ended.

### Proposed implementation contract

Keep the existing argv and preview/confirmation distinction. Reuse the monitor's `SelectResourceProcesses` proof; freeze the admitted exact process keys and never expand their descendants. A CLI exact-set coordinator uses the existing TERM/3 s/KILL/2 s cadence. Each short supervisor-side effect request rechecks the frozen target against current process/inventory proof, authenticates an owning hub when caller credentials are supplied, and rechecks a nonempty start identity before signalling one PID. Unknown, stale or current ownership refuses; lost replies remain uncertain and do not authorize automatic resend/escalation.

Hub pin 941a9381-a841-44ca-98b0-7e10a90936b3 approves authenticated owning hubs only (workers and other hubs refused), with entirely unbound local CLI requests using the existing owner-only socket as the operator boundary. There is no new machine-steward role. The [durable plan](../plans/000792-proven-orphan-reap-plan.md) describes the protocol, state outcomes, counted scan cost, overlap handling, tests and live acceptance; the complete design and authoritative refinements were approved and admitted through change-code.

## Done when

- With a live slot plus a proven orphan set from its prior incarnation (fake process table), `--reap-orphans --confirm`
  ends exactly the orphans and none of the live processes (test).
- A pid reused since the sweep is not signalled (identity fence test).
- A current descendant beneath an old root and newly discovered/unproven children are not added to the confirmed exact set.
- Caller replacement, hub reassignment, live ownership transfer and unknown evidence before either TERM or KILL veto that effect; canceled/lost-reply/failed signals retain explicit partial outcomes.
- Preview and confirmed text/JSON report each admitted process once, including legacy overlap. Existing legacy families remain available only until this Couch process first observes incomplete provenance; after that they refuse for its lifetime. Public disposable-Couch conformance keeps the current Claude/Codex fixture slot alive and responsive.

## Plan

- [x] Approve the durable plan and enter change-code through the full-flow gate.
- [x] Add pure exact-target proof and strict protocol shapes; verify the negative authority/identity cases first.
- [x] Implement the guarded exact executor and supervisor effect owner, with stateful transition and counted-cost tests.
- [x] Wire preview/confirmed CLI reporting, prove disposable public-command behavior, document and run the required recorded verification before the SDLC close review.

## Log

### 2026-10-21

### 2026-10-22 — claim and plan
- 2026-10-22: closed — Completed full suite c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934 ran10m13s and found only two deterministic comment lint defects; retained attempt9e8598c190908d7948cc12875b4deb6f and full SHA logged. Three comment-only corrections, executable tokens/tests unchanged. Targeted Godoc0.528s/catalog4.579s; full repolint13.921s/termcmd7.314s preflight and reader/whitespace audits PASS. Prior final-source native Claude/Codex exact6/current delivery/cleanup remains applicable to unchanged behavior. Required replacement affected/full suite on pushed e6535a0c2c671bfeb3b1065ed279c269886829b0, lease-2020441815 queued. --no-plan-check only for worker-authorized review overlap; final Plan row remains pending recorded pass, which still precedes landing. Original merge waiter retained. (fixed-then-ship); review verdict: SHIP
- 2026-10-22: closed — Final source c1ca44e7eddb0e5a0ba10108e284bb4f6e049321: combined targeted race couchcmd3.344s core3.209s message1.160s; reader audit and artifact classification pass; three native nvim checks pass5.896s; separate public-command Claude44.188s and Codex42.797s pass exact6/current delivery/native cleanup under96MiB cap, strict permitted legacy-only partial. Recorded affected selected full suite on pushed c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934, lease-440357 queued; final Plan row remains pending recorded pass. --no-plan-check acknowledges only TL-worker-authorized review overlap; recorded final-source pass still required before landing. BR2 revision routes all legacy effects through current supervisor proof and one sticky gate; rootless survivors refuse.; review verdict: SHIP

Dispatched by ops:0 after #777 and the #768 manual archive repair landed. PR #540 is merged; both retained repair runners exited successfully and the workspace returned cleanly to its resting branch. Claimed #792 before design. start-plan's automatic dependency refresh generated an unrelated merge-check workflow change in an initially clean tree; restored only that generated diff and reran start-plan successfully, keeping protected gate configuration outside this issue.

Code inspection: ResourceView.Orphans already contains exact positive provenance from SelectResourceProcesses. The existing CLI consumes only OrphanNvim and LeftRunning and performs tree-expanded signals locally. The protocol has confirmation but no caller credentials. A bounded read-only helper audit confirmed TreeReaper expands every descendant, while teardown stopExact is unexported and lacks an effect guard; neither is safe to reuse unchanged for this live-slot case. Thread/lifetime leases cannot freeze an old process set while keeping the current incarnation live.

Asked the hub to pin the authority boundary in receipt eda16b29-ce04-4710-a5b2-4df4e17c03c8. Its submitted reply 941a9381-a841-44ca-98b0-7e10a90936b3 confirms owning authenticated hubs, no worker/other-hub permission, and the owner-only unbound operator CLI. The proposed full-flow plan keeps waits in the CLI and sends each guarded signal in the supervisor, with no new durable operation state. ARCH-DRY and ARCH-PURE preserve the existing provenance source and cadence; ARCH-SECURE and ARCH-STATE require current per-effect vetoes and explicit uncertainty. No implementation code changed and no estimate derived.

Hub approved the complete plan in submitted receipt cea188f7-9b5c-4999-879d-59404fd1971a and directed change-code. Disposable public-command proof with Claude and Codex is sufficient; no operator stage is needed. The first real use after landing is ops-echo's cleanup of echo:2's still-proven 10-21 leftovers. Enter the full-flow gate before implementation; estimate remains deferred until plan-quality clears.

Plan-quality round 1 raised PQ-1 complete detached ownership, PQ-2 enforced transition model, and PQ-3 named production-function test strategies. Inspection confirms DetachedSlotRoots enriches a live background suite before SelectResourceProcesses; the raw roots alone can misclassify it (resource_detached_test.go's real reparenting regression). The plan refinement therefore shares the sampler's complete ownership stage on its existing worker, includes generation-cached gated environment evidence, and refuses unknown relevant enrichment. It also makes AdvanceExactReap the only progress/effect writer with tagged invocation states and generated sequence/interpreter checks, and replaces the active test inventories with function-level strategies. No code changed; request the hub's ownership/cost clarification approval and rerun plan-quality before deriving an estimate.

### 2026-10-22 — implementation checkpoint

Authenticated refinement approval 044b0f32-5645-4870-80af-61eb967f7369 accepts the shared worker proof, detached unknown veto, unchanged bounds and pure invocation transitions. Plan quality cleared PQ-1/2/3 in round 2; the estimate retry ran round 3 CLEAN again. Derived 4.033 hours from the prescribed stale v3.1 calibration; estimate quality returned INFO, not a blocking finding. Its integration bucket includes the shared collection/channel/cache/authority tests; the conformance and close buckets also absorb inventory checks, recorded verification, landing and review repairs. The tight reducer allowance is a calibration risk to compare with measured actuals, not a reason to omit verification.

The full-flow change-code gate admitted implementation. Test-first work now adds positive exact provenance selection, strict optional caller and closed TERM/KILL protocol shapes, the pure AdvanceExactReap model and stateful interpreter, worker-owned collectOwnership, per-effect hub/identity guards, and CLI exact-group reporting without local signal fallback. Known-empty environment evidence uses a requested PATH observability witness without retaining its value; empty/withheld, partial or failed reads remain unknown for effects, including cached unknown generations. All process signals in the new group remain supervisor-owned and exact.

Verification so far: targeted core/protocol/launcher/service/CLI and legacy reap selections pass. Launcher generated/exhaustive model tests and stateful executor race tests pass. The combined new service/CLI proof selection passes under race detection, including current or unknown detached ownership before KILL after acknowledged TERM. The full artifact inventory first refused the two new shared retained-fixture helpers, as expected; they are being registered, without a quarantine. The reader audit is pending final source registration. Shared disposable-Couch native acceptance is running; no recorded suite or landing runner has started, and no :0 stage was made.

### 2026-10-22 — native qualification and independent restart refusal

PR #541 opened; its first head passed all four CI checks. The broader couchcmd, couchcore and launcher package runs passed in62.312s,380.455s and64.216s. Full artifactpath passed in16.032s after registering the shared fixture sources. Reader inputs were audited: pure/fake proof tests are literal fixtures; native Build/New consumes real runtime inputs and conservatively uses non_go_full_fallback. Their per-reader certificates pass the audit.

Native setup first failed at missing wrapper-ready publication, before retained descendants existed; subsequent starts reached the fixture. Cause remains unclassified, not a passing-retry/flake claim; #773 records the related missing-ready evidence class. A diagnostic run then exposed a test-only resource-envelope oracle error, followed by the expected two-step live reboot confirmation contract. Corrected those assertions without product changes. The completed native attempt then proved all six ancestry and reparent barriers but the acknowledged public reboot failed with unknown pane owner before #792 effects. Its retained log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-confirmation.kamQRO/native.log hashes to794f8c0dce57ec187b5ae7eda2c90e80fd35a9974b280c4036e96eb8b40f4d0e; exact disposable cleanup passed.

Authenticated hub approval d2356d3f-461c-4576-9ceb-a71acb4179f6 approves the fixture-only exact-generation crash of the already-proven old worker cohort, excluding the frozen six and the steward, followed by public reboot on the dead session, preview, confirmed reap and current-worker delivery for Claude/Codex. Filed the separate live-reboot bug as unclaimed #797 with the concrete receipt and log evidence; no recovery validator change. The plan Revisions records this acceptance refinement.

Recorded affected selection on d2f4b5e9 expanded to full suite and queued lease-2652748101. Canceled only that owned queued lease via sdlc before capacity, because native-test oracle repairs changed the source requiring a new head; the runner exited with suite-lock context canceled. No suite started or evidence was discarded. No recorded check or landing runner is active now. Requeue on the final qualified source.

### 2026-10-22 — final native harness and sampler lifetime checkpoint

The approved fixture-crash path now intersects only the previously monitor-observed old worker set with the shared harness's positive private ownership proof. It protects retained descendants, steward and supervisor, skips only positively gone/reused generations, and rechecks identity for every single-PID fixture signal. No recovery validator or production bounds changed. The fixture explicitly launches its hub through the public directory argv, and waits for the exact replacement broker binding, not merely agent readiness.

One native run reached successful exact six-process reaping and proved the replacement agent generation survived; final delivery qualification remains pending the complete native run. A later ancillary full-resource read ended with JSON EOF; its oracle decoded before checking the typed truncation flag, so that run did not retain conclusive flag/length evidence. Source inspection and bounded payload tests establish that the production frame can exceed the fixture's32KiB generic CLI capture; this is an observation-limit mismatch, not a reason to expand generic limits. Kept every frame/output/deadline bound unchanged: a closed read-only fixture observer calls the existing resource RPC in the actor's inherited isolated namespace and returns only observed key/slot/name/orphan flags, refusing oversized output. Public reboot, preview, confirmed reap and delivery stay on the genuine hub's CLI.

Source inspection then found an implementation lifetime defect: the worker request's context could outlive canceled sampler ownership. TestOwnershipProofJoinsWhenSamplerLifetimeEnds reproduced it (red: sampler proof outlived its lifetime). Collection now inherits both caller and worker cancellation without another worker; the proof, legacy, service and CLI race selection passes1.806s and ordinary selection passes0.412s. The shared observer/startup/crash race selection passes (testcouch1.255s, agentfixture1.214s); full artifactpath passes23.389s. Re-audited changed reader inputs; the new resource-proof tests read only literal/stateful fake fixtures.

No recorded suite is active. Checkpoint this source, run the complete native acceptance without further source edits, then push and queue the required recorded check. Any new concrete failure is classified before another run; no deadline tuning or validator weakening.

### 2026-10-22 — refusal checkpoint and split native acceptance

The committed-head39625e2c9 combined native invocation failed, rather than supplying a passing matrix. Claude passed its complete six-key reap, current-generation public delivery and exact cleanup (confirmed reap128.707ms). Codex missed replacement wrapper readiness after14.446s, before reap effects; cause remains unclassified alongside #773. The combined process also exceeded the unchanged96MiB ordinary write cap (162344960bytes). Retained log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-final.U2rlOV/native.log SHA2567fbd45732e2905afdd4bd72a53030ad92938cc621bb80c0ce29300012cc26ff0.

A new deterministic regression reproduced an incomplete orphan witness being excluded from exact admission and then reaching the legacy LeftRunning tree for the same PID. Admission now protects every orphan-row PID against legacy fallback, reports incomplete/ambiguous witnesses as explicit refusals, and deduplicates preview through the same admission decision. TestReapProvenUnqualifiedWitnessCannotFallBackToLegacyTree failed red with a TERM signal, then the corrected reap CLI race selection passed1.530s. No target expansion or production bound changed.

Authenticated hub approval8c0e8477-dcd4-495d-a6b9-ea4f95285957 permits one focused Go process per personality on this final source, preserving96MiB and every existing time/frame/output limit. Keep the startup miss unclassified; do not register the safety test as flaky. Checkpoint first, then final Claude/Codex runs, required recorded suite and close. No suite or landing runner is active.

### 2026-10-22 — focused native acceptance passes

Both approved focused commands passed on76cc79deb00a92833f0a6ddf5ee13f7ee9a6c50c. Each reaped exactly six old identities, preserved the current agent, accepted public delivery and completed exact-generation/direct-child/IO/root cleanup. Claude completed49.684s, confirmed reap152.182ms, writes81387520bytes; Codex43.914s, confirmed157.283ms, writes81444864bytes. Both are below the unchanged100663296-byte cap. Commands used PAIR_LIVE_PROVEN_REAP=1 with go test ./cmd/internal/couchcmd -run '^TestLiveProvenOrphanReap$/claude$' (or /codex$), -count=1 -v. Logs: /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-claude.GrB3z3/native.log SHA256f5767dcaad9e46458903fadb103dc75d0772d05b88a26a24ebe85360648b84df; /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-codex.891kPz/native.log SHA256d63a282fb735b06c1ae47692d9a43051396447656d26523f2bc0c5b9b5939b90. Prior startup failures remain unclassified evidence, not erased by these passes.

Pre-close traceability audit found missing explicit between-TERM/KILL caller/hub-change regressions and mixed-own/foreign CLI composition coverage. Add these approved-contract tests before recorded verification; no production change is required. The new caller/hub mutation race test passes1.549s; independently varying K/P/S at8/16 passes1.453s with exactly2K proof table/inventory observations. The plan's measurement clarification distinguishes deterministic operation counters from native outcome/time/write evidence.

Mixed-owner real service composition now covers all four preview/confirmed and text/JSON cases with the actual classification worker, actual effect handler and real temporary legacy registry overlap. Its race selection passes1.596s; the combined reap/proof/count race selection passes1.857s. Reader audit and diff whitespace check pass. These additions change tests/documentation only; native and product sources remain unchanged from76cc79deb. Checkpoint final coverage before another pair of final-head focused native commands and the recorded check.

### 2026-10-22 — final source qualified; recorded suite next

Both focused native commands passed again on final coverage commit7181ed56f2a4940c5b50f24242702559944ee22d, with no source edits during qualification. Claude39.471s, confirmed126.425ms, writes81539072bytes; Codex33.934s, confirmed103.023ms, writes81543168bytes. Each proved six exact old outcomes, preservation and public delivery to the current worker, and complete owned-generation/direct-child/IO/private-root cleanup. All bounds remain unchanged. Final Claude log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-final-claude.NzmOKl/native.log SHA256f75bbf3d98dd20931127901c23bd277173c52a394b0bf491dd95dc7ad4d36b3f; final Codex log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-final-codex.Ovv7Wq/native.log SHA256b048e75b7811d33f61327aab369efd4262ec3040076f615f5507f666bb5b94b3.

Native acceptance, negative authority/identity cases, composed CLI reporting, operation counters, documentation, artifact inventory and reader audit are complete. The last issue Plan row remains unchecked solely because required recorded verification has not yet passed; queue it after this durable evidence checkpoint. Any permitted close-review overlap must preserve that pending status and cannot authorize landing before the exact-source recorded pass.

### 2026-10-22 — close round1 REWORK; projection boundary

Boundary review round1 returned Critical BR-1, family incomplete-evidence-must-not-widen-authority: independent response projection can omit an orphan row while retaining its overlapping legacy row, bypassing exact admission. Verified the dataflow: the sampler projects its cached view, then the confirmed service response projects again; a service-only filter would therefore be too late. Root fix belongs before the first lossy projection and covers both legacy families against all full-view orphan PIDs, including incomplete witnesses. The durable plan adds this refinement without new state/protocol/bounds. Mandatory review is not finalized; issue stays working.

Canceled only owned queued suite lease-3281852616 ondfd00c60 via sdlc after the source became superseded. The retained runner exited context-canceled before capacity; no suite executed. There is no recorded passing suite and no active landing runner. Reproduce BR-1 with a projection-to-CLI red test, then repair, requalify and queue the corrected head.

BR-1 red evidence: all four projected CLI cases (LeftRunning/OrphanNvim, complete/incomplete omitted witness) signalled both the old PID11 and current descendantPID12, then falsely reported reaped. The correction filters both legacy sections using every complete-view orphan PID at the common projection entry, before either section is bounded; the CLI shares that protected-PID helper. This covers both cached sampler and final-response projections. No frame/deadline/output cap changed. The transport-capacity fixture now uses genuinely unrelated nvim PIDs so it still proves legacy section retention.

Projection-to-CLI regressions, repeated projection, conservative incomplete/changed-identity overlap, unrelated legacy retention and source immutability all pass in the combined race selection2.364s. Reader audit and diff check pass. Re-run final focused native acceptance and the required recorded suite on this corrected source, then full close (round1 REWORK was not a finalized close).

### 2026-10-22 — BR-1 corrected-source live acceptance

Both focused native commands pass on48c6d6584e6744c6303f1efdfd13558140fcefd3 after the production projection fix. Claude49.967s, confirmed152.024ms, writes81539072bytes; Codex38.157s, confirmed77.554ms, writes81453056bytes. Both reap exactly six prior identities, preserve the current worker and accept public delivery, then prove every owned generation absent with joined direct child/IO and removed private root. The96MiB cap and all time/frame/output bounds remain unchanged. Logs: /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-br1-claude.oIM6DB/native.log SHA256b1b44eba3ae5f89a8904dd82038f4b72a5f75b6501a9c0dd3a48f8a7d3ac6553; /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-br1-codex.aKr40D/native.log SHA25644aba4b77a53c38da64d15702a9cff7a15ecdf2e3fcebf1d33551e7170d8d1c8. Broader sampler/projection/reap selection also passes0.573s.

Checkpoint this evidence, queue the corrected-source affected check (full selection expected), and rerun full close because round1 was REWORK. The required recorded pass remains pending and the final Plan checkbox stays unchecked until it completes. No continuation/restart while the owned check is queued or running.

### 2026-10-22 — round2 REWORK and approved omission-class re-plan

Round2 verified BR-1 addressed but reproduced Critical BR-2 upstream selection/provenance loss. Real sampler/service/CLI red cases cover selection overflow, overflow then healthy, ownership conflict then healthy, and incomplete census: each reached legacy tree signalling of foreign oldPID11 plus current descendantPID12 and falsely reported success. Healthy recovery cases had no remaining coverage error. Important BR-3 independently reproduced the RPC fixture socket exceeding103bytes under long TMPDIR; a short private TestScratch-preferred fixture root fixes it, long-TMPDIR race pass1.229s.

Canceled only queued owned suite lease-2781851290 via sdlc; it exited context-canceled before execution. No recorded suite or landing runner remains. Stopped production edits for the repeated omission family and asked the hub to pin the compatibility/state tradeoff. Authenticated08398b83-dc96-4884-8c39-48c93a17f479 prioritizes safety: one in-memory sticky flag, cleared only by a fresh Couch process, refuses legacy tree admission after any incomplete provenance. Exact proven reaping stays available; nothing is persisted. The durable plan now specifies the whole-class decision, typed refusal, compatibility and sequence/cost tests before implementation. No estimate is refitted.

The first sticky implementation passes seven real uncertainty/recovery scenarios, exact-after-refusal and fresh-service legacy cleanup (race2.511s). Pure completeness mutation/growth tests pass race1.353s; broader existing sampler/reap selection passes0.315s; full artifact inventory passes6.632s. The flag lives only on the process-owned sampler; worker replies carry a refusal fact, not a second authority. Confirmed responses use provenance-incomplete and remove actionable legacy rows, so older CLIs refuse. The current CLI still runs guarded exact rows and reports legacy_refusal/nonzero.

Integration inspection found the initial inventory defaults unknown while its seed races the sampler. Asked the hub before changing startup ordering: authenticated dc2b6d09 approves one owned seed-completion barrier, no new waits/deadlines. Unobserved cold startup is distinct from a failed/incomplete seed; the sampler starts after the seed attempt completes. The corresponding plan revision and healthy/failed/canceled startup tests are in progress. Filed unclaimed ariadne#549 for the shared socket-fixture lint class, because Pair imports the peer catalog; its dependency/landing disposition is with the hub.

### 2026-10-22 — sticky refusal and seed ordering implemented

The approved omission-class fix is now implemented: ResourceOwnershipComplete is pure, and the shared collection worker sets one process-lifetime boolean on any incomplete/ambiguous ownership observation, including proof-only requests. No later healthy sample clears it. The sweep reply carries that refusal to legacy admission; a provenance-incomplete response contains exact rows and zero actionable legacy rows, so older CLIs fail closed. The new CLI can finish healthy guarded exact rows while reporting legacy_refusal/nonzero, and never executes either legacy family after that refusal. No state or refusal flag enters ResourceState, ResourceView, cache or persistence.

Seven actual service/CLI sequence regressions cover overflow, repeated overflow then healthy, conflicts, incomplete census/inventory and unknown detached evidence; independent legacy nvim and registry rows stay untouched. Exact-after-sticky remains usable; a fresh complete service permits both normal legacy families. A proof-only incomplete observation also fences later legacy admission without disabling later healthy exact proof.

The startup barrier releases sampling only after the existing inventory seed attempt completes, remains owned by the two existing service workers and cancels with their lifetime. Failed seed cannot inherit the console's cached known flag. Healthy, failed, delayed and canceled seed tests pass. Final combined reap/proof/ownership/sampler/inventory race selection passes couchcmd3.218s and core1.718s; focused inventory race passes1.621s. BR-3 long-TMPDIR race passes1.229s with short private root/cleanup. Full artifact guard passes6.632s, reader audit and whitespace check pass. The shared mechanical lint follow-up is unclaimed ariadne#549, reported to the hub with submitted receipt e5d1505f-b0b7-4db0-8202-0e4c40a19a8e.

Checkpoint this source, qualify both focused native personalities with all bounds unchanged, then push/record the final check and full close. No recorded check or landing runner is active.

### 2026-10-22 — current ownership edge and native partial oracle

Claude on7eb4418e joined65.592s exit1: the CLI reported all six exact rows reaped alongside the newly approved legacy_refusal, but the old exit0 test assertion stopped before current delivery/remaining native assertions. Cleanup passed and writes81297408bytes stayed under96MiB. Keep this as failed acceptance, not a full pass. Log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-br2-claude.lhJ4Go/native.log SHA256a7eb06008053ff19561c9b896f1416e42397f19283870d5f8bf207cc869db1ce. Its first incomplete observation was not retained; do not invent a cause. Codex was not started.

A new actual-handler red test reproduced TERM despite unknown current slot ownership after old-root death. Monitor positive root-death evidence was insufficient for destructive proof. The worker effect reply now reuses ResourceOwnershipComplete for its current proof, independently of the historical sticky flag. Direct TERM and between-TERM/KILL unknown-owner regressions pass; combined affected proof/reap/oracle race selection passes2.912s (prior broad selection3.651s). This enforces the existing complete-current-proof contract without more state/protocol/bounds.

The native oracle now recognizes only the exact documented typed legacy partial, while preserving six unique exact reaped outcomes, native absence, current generation, delivery, joined completion and cleanup. Unknown refusal, transport loss, truncation or uncertain cleanup still fail. Small oracle regressions pass0.335s. Record updated reader certificates, checkpoint, then repeat both focused final-source acceptances and required recorded verification. No native/suite/landing runner remains.

### 2026-10-22 — diagnose current proof refusal before changing eligibility

Claude on534166a6 failed an exact target with complete current ownership proof unavailable, beyond the permitted legacy-only partial. The test stopped before current delivery; exact cleanup passed and writes81485824bytes remained under96MiB. No Codex run or blind retry. Log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-proof-claude.osb808/native.log SHA256f10eb9d13c8d3caca11c44cac733928ae87f929e76adfdbcf0440719d782060e.

The global current-completeness reuse may veto unrelated unknown evidence, but this log does not establish which condition failed. Add a bounded pure current refusal reason, carried only in the ephemeral proof and existing effect detail. It preserves the exact eligibility decision, sticky flag, all bounds and no retained history. Diagnose that observation before modifying eligibility; do not invent a root cause or register a safety flake. No native/suite/landing runner is active.

### 2026-10-22 — diagnosed global detached veto and restored target relevance

The single diagnostic Claude run on356cf510 failed with the bounded reason detached ownership unknown. Cleanup passed, writes81469440bytes stayed under96MiB; current delivery was not reached. Log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-diagnostic-claude.FWzOzN/native.log SHA256af795d6e3b69a102054535485d707e8c6c1cfb1d3ffe14b9e8194b0478cf3fe7. Codex was not started.

Verified the composition defect rather than weakening the assertion: the global completeness helper applied to exact effects was treating unrelated detached uncertainty as relevant. A new real-handler test with unrelated unknown sdlcPID99 failed red with zero allowed target signals. Exact effects now keep full current census/slot completeness while passing the full detached-unknown set to the existing target-and-ancestor veto. The process-lifetime legacy fence remains global and unchanged. Relevant detached ancestors and unknown current owner after old-root death still veto TERM/KILL. Combined proof/reap/ownership/seed race selection passes couchcmd3.324s/core1.663s. This restores the already approved relevance contract, not a deadline or frame/output tolerance.

### 2026-10-22 — final scoped-proof native acceptance passes

Both focused native commands pass on924ea8b3a913fe193e34023c0ac929d6674a7931. Each accepts only the documented joined legacy-only partial, then proves all six unique exact outcomes reaped, independent native absence, unchanged current generation, accepted public delivery and complete owned-generation/direct-child/IO/private-root cleanup. Claude38.839s, confirmed102.468ms, writes81362944bytes; Codex42.083s, confirmed128.377ms, writes81559552bytes. All96MiB/time/frame/output bounds remain unchanged.

Logs: /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-scoped-claude.sHXxz3/native.log SHA2563763275b6ce4b1143548663f257ade67ab38b0fb07e2620cfe04c25141b808ef; /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-scoped-codex.qqImeX/native.log SHA2561f342c503c547cdf5650a518bf7718a0ec753de32dd768e5b7acf1d66183168d. Source stayed frozen through both runs and reader audit passes. Required recorded suite remains pending; checkpoint this evidence, queue the final check and full close round3. The sole final issue Plan checkbox stays unchecked until its recorded verification passes. No native or landing runner remains.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. The source is marked stale by estimate-source; this is a provisional calibration, not a fitted total.

The approved plan resolves the design: apply ×0.2 to design, familiarity 1.0 for existing Go/protocol/process seams, 40% of table implementation hours, and the thorough-plan 15% design buffer. In order: issue/design checkpoint (base 1.0/0.2); pure target proof (1.0/0.8); exact transition model/executor (1.5/0.8); worker proof and guarded signal integration (2.0/1.5); CLI/reporting extension (0.3/0.5); disposable public-command conformance (1.0/1.5); docs (0.2/0.2); one close review (0.1/0.5). Each pair is base design/implementation hours before those discounts. Reuse the existing native reader, classifier, messaging transport and disposable-Couch harness: no novel platform library or external API needs discovery, and no library can replace the application-specific authority proof or reducer. No cross-repo propagation.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.2 impl=0.08
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.3 impl=0.32
item: api-integration design=0.4 impl=0.6
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.2 impl=0.6
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 4.033
```

## Revisions

- 2026-10-22 — Design inspection found descendant expansion and a missing caller authority boundary. Preserve the original incident and two acceptance requirements; add the approved hub/operator contract and adversarial acceptance cases, and link the full implementation plan. No relaxation of identity or live-incarnation protection.
- 2026-10-22 — PQ-1/2/3 refinement adds the missing detached-root evidence owner, a structurally enforced invocation transition model and production-function test strategies. The complete proof may require existing generation-cached environment reads; the earlier zero-environment-read claim is superseded. No new durable state, widened target set, role or deadline.

- 2026-10-22 — Reconcile current Spec with the approved owning-hub/unbound-operator boundary (no machine-steward role) and completed design gate. The last Plan row now names pre-close acceptance; close, landing and the outcome report remain mandatory subsequent SDLC actions, recorded in the Log. No implementation scope or acceptance is relaxed.

- 2026-10-22 — Hub pin08398b83 approves the BR-2 re-plan: one non-persisted, process-lifetime sticky legacy refusal after any incomplete provenance. This explicitly supersedes unchanged legacy execution in pathological incomplete/cap cases. Exact proven reaping remains available under full per-effect proof; no deadline/cap increase or persisted state.

### 2026-10-22 — Close round three and approved effect-gate revision

Close round three remained REWORK on BR-2: a healthy legacy admission could continue CLI-local signals after an exact proof collection set the sticky refusal. BR-1 and BR-3 remain addressed. Canceled the superseded queued lease-3110944572 before execution; its runner exited context-canceled and produced no suite evidence. No suite or merge runner remains.

Hub approval 80b806ab-7c5f-4a69-a3f5-019fd92fc7f3 authorizes the appended durable re-plan: every legacy single-PID signal gets fresh supervisor root/ancestry proof and a sticky gate read lock through identity rereads and signal; flag transitions take the write lock. CLI trees become frozen exact executor inputs with server callbacks. Root death/reparenting yields conservative partial/refused outcomes. Next legacy-window finding must go to TL for retirement of legacy signalling, not another patch. Final verification remains pending.

### 2026-10-22 — Supervisor legacy effects implemented; verification pending

All legacy TERM/KILL effects now use reap-legacy-signal. The CLI freezes one bounded forest and deduplicates generations across both families; old local signalling helpers were removed. The worker returns full current proof with that same collection's legacy classification, and its sticky setter takes the gate write lock. The handler holds hub and sticky read locks through the final current proof decision, exact root/member rereads and one signal. The pure decision excludes protected ancestors and descendants and refuses surviving children after root death/reparenting. No retained permits or persisted state.

Red evidence: fresh sweep omitted proof; strict wire rejected the new operation; actual CLI/service late-loss regression could not reach a supervisor legacy gate. Planner tests exposed an invalid empty-identity root incorrectly skipped, now refused. Targeted race tests passed for core/wire, planner, and actual service/CLI revocation sequences. A new exact-wait fixture initially hooked Identity although the executor polls Snapshot; the combined race run correctly failed its missing-injection assertion. Corrected the seam to actual first wait census after acknowledged TERM; targeted race now passes2.550s. This was a test defect, not a registered flake or a passing retry over unexplained failure.

The existing three real nvim fixtures all executed and passed5.896s, including fresh supervisor proof and a real native signal with independent generation absence. The destructive fixture's census is closed to its exact real kernel-derived subtree; broader host classification remains exercised by final public-command native qualification. Artifact source classification passed2.663s. Final combined race, certificate audit, pinned-head native Claude/Codex, recorded verification and close remain pending.

### 2026-10-22 — Final supervisor-gated source verification

Source c1ca44e7eddb0e5a0ba10108e284bb4f6e049321 stayed unchanged through both separately invoked approved public-command native runs. Combined final race selection passed couchcmd3.344s, couchcore3.209s, couchmessage1.160s; reader certificate audit passed. Both native personalities proved exactly six unique targets reaped, independent native absence, current generation preserved, public accepted-body delivery and complete owned cleanup (direct children and I/O joined). Both returned only the strictly permitted legacy-only partial refusal; other refusal, uncertainty, truncation or unrelated outcomes still fail acceptance. No bound changed.

- Claude: Go44.188s/subtest35.14s, confirmed202.871291ms, writes81518592 of100663296 bytes. Current4677/1792666878.630476 remained live through delivery. Log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-finallegacy-claude.K4h79C/native.log; SHA2569bf6d3a0fac0bced766889afb4d47f9f2a96890489b22dff9855ab1fb64a3d1e.
- Codex: Go42.797s/subtest33.59s, confirmed153.228167ms, writes81534976 of100663296 bytes. Current61438/1792666941.388852 remained live through delivery. Log /Volumes/TestScratch/tmp/pair-slot2/pair-792-native-finallegacy-codex.63OTUg/native.log; SHA256bd9837315dbe66a350a27b56ddff23fd14be5753ab2d740a968d35516e54c255.

Both runners joined; no native runner or stage remains. Required recorded verification and full close remain pending. The final Plan checkbox stays unchecked until that recorded check passes.

### 2026-10-22 — Recorded full-suite result and comment-only lint correction

Retained lease-440357 completed on c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934 after10m13s of execution. It ran to completion and reported exactly two failing checks: cmd/internal/termcmd.TestNoDeclarationCarriesTwoStackedGodocs (orphanReapReport and resourceSweep doc paragraphs displaced by new declarations), and internal/repolint.TestTrackedGoFilesPassCatalog (an unmatched tmp-literal suppression on the short socket-root fixture). These are reproducible branch defects, not flakes or infrastructure; no registration or resume is appropriate. Saved attempt9e8598c190908d7948cc12875b4deb6f, log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/9e8598c190908d7948cc12875b4deb6f.log, SHA25617eef1ee34adbd7155c0e44e00aa9399f7a6fa26c443e4cc943cf12c6a60dd44. Other packages completed, including couchcmd51.434s, couchcore153.181s, couchmessage25.017s and launcher66.763s.

Moved both doc paragraphs onto their declarations and converted the unmatched suppression into an ordinary explanatory comment. Only comments changed; executable tokens and tests are unchanged, so the pinned native acceptance remains applicable. The existing lint rules already cover both mechanical classes. Targeted Godoc check passed0.528s and catalog check4.579s; full preflight packages passed repolint13.921s and termcmd7.314s. Reader certificate audit and whitespace check passed. I missed these cheap preflights before the first queue; the lesson now requires them together before recorded verification.

Round-four close is SHIP with zero new findings and all previous findings addressed. The original merge waiter remains retained, safely gated on required verification. The final Plan checkbox stays pending the replacement recorded check; narrow post-close review is required for this comment-only correction.

### 2026-10-22 — Required recorded verification passed

Replacement lease-2020441815 completed successfully on e6535a0c2c671bfeb3b1065ed279c269886829b0: exit0 in10m16s, recorded sdlc-local/suite. Both prior deterministic lint failures now pass (termcmd2.452s, repolint6.895s), with all remaining packages and shell/editor checks complete. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/e6535a0c2c671bfeb3b1065ed279c269886829b0/suite.log; SHA256558727feaedf25ed7760ac48b3951f09d73ef6e2b080f751b3eb04f602e96865. No flake registration, blind retry or suite waiver. The immutable earlier failed record remains preserved.

Final issue verification checklist is complete. The worker-authorized review/suite overlap is now resolved by the passing record before landing. Full close and narrow comment-only re-close both returned SHIP; native Claude/Codex proofs remain applicable to unchanged executable behavior. The original merge runner remains responsible for current-main integration, authoritative checks and landing.

### 2026-10-22 — Landing integration and atlas conflict resolution

The original merge runner exited1 while integrating origin/main at5df9efb7d0df1a627f8d929895181fcf7b40ef73 after #779 landed. SDLC preserved the branch and directed manual integration, required verification, fixed-then-ship re-close and re-entry. Repeated that merge as directed; its sole conflict is atlas/index.md, where both features extended the same Couch summary line. Kept the wrapped tab-bar description (#779) and exact previous-incarnation cleanup description (#792) together. All code and reader certificates merged automatically; no reap behavior changed in this resolution.

The earlier e6535a0c recorded pass remains historical proof of that source. Final verification is pending again for the integrated source, and the final issue checkbox is reopened accordingly. Run the cheap catalog/Godoc/artifact/reader preflights, then the required integrated recorded check and narrow re-close. No merge or recorded-check runner remains active in this slot; re-enter only after those gates.

The index resolution initially retained both feature clauses. The final index entry now matches landed main exactly: Couch was already linked, so #792 needs no per-feature clause on that shared line. Its detailed behavior and ownership map remain in atlas/couch.md. This removes our unnecessary index delta and lets later main additions merge without another index-only conflict; all guide links are preserved.

Integrated-source preflights passed: internal/repolint13.422s, termcmd5.562s, artifactpath15.595s, reader inventory and whitespace audits. The PR has no .sdlc/.github delta against integrated main; those incoming policy changes are already on main. Reap handlers, executor, protocol and resource-collection source paths are unchanged from the recorded e6535a0c source; prior native acceptance remains applicable, while a fresh complete recorded check is still required for the combined code. This checkpoint precedes that long check.

### 2026-10-22 — Integrated recorded verification passed

Retained lease-1170667131 completed exit0 in10m42s on pushed integrated source5bf644a97583c63168a0059701a8c4c3cac967b9, recorded as sdlc-local/suite. All packages and shell/editor checks completed, including couchcmd51.149s, couchcore167.812s, couchmessage26.001s, launcher72.054s, termcmd2.357s and repolint6.369s. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/5bf644a97583c63168a0059701a8c4c3cac967b9/suite.log; SHA256743004136e89fec9e80bc0772db0243ef415631235e02c6ad90abafa8f4c6f08. No failure, flake rerun or waiver. The final verification checkbox is complete again; earlier native Claude/Codex acceptance applies to unchanged reap executable behavior. Narrow fixed-then-ship review of the atlas-only integration resolution is next, followed promptly by a fresh merge runner because the original runner exited on that conflict. TL granted two-hour priority; report landing immediately so it can be revoked.
