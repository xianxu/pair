---
id: 000792
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours: 4.033
collaboration-mode: collaboration
card_mirror: '8cf2318f6c80190ae2470cc546af388fc2abc050' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T00:59:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

Fix: `--reap-orphans --confirm` (run by the owning hub, or the machine steward) also ends **proven orphans of a
previous incarnation** of a live slot, the set `--resources` already proves, each signal fenced on the process
identity the sweep recorded, never touching the live incarnation's processes. Report what was ended.

### Proposed implementation contract

Keep the existing argv and preview/confirmation distinction. Reuse the monitor's `SelectResourceProcesses` proof; freeze the admitted exact process keys and never expand their descendants. A CLI exact-set coordinator uses the existing TERM/3 s/KILL/2 s cadence. Each short supervisor-side effect request rechecks the frozen target against current process/inventory proof, authenticates an owning hub when caller credentials are supplied, and rechecks a nonempty start identity before signalling one PID. Unknown, stale or current ownership refuses; lost replies remain uncertain and do not authorize automatic resend/escalation.

Hub pin 941a9381-a841-44ca-98b0-7e10a90936b3 approves authenticated owning hubs only (workers and other hubs refused), with entirely unbound local CLI requests using the existing owner-only socket as the operator boundary. There is no new machine-steward role. The [durable plan](../plans/000792-proven-orphan-reap-plan.md) describes the protocol, state outcomes, counted scan cost, overlap handling, tests and live acceptance; the complete design awaits approval before change-code.

## Done when

- With a live slot plus a proven orphan set from its prior incarnation (fake process table), `--reap-orphans --confirm`
  ends exactly the orphans and none of the live processes (test).
- A pid reused since the sweep is not signalled (identity fence test).
- A current descendant beneath an old root and newly discovered/unproven children are not added to the confirmed exact set.
- Caller replacement, hub reassignment, live ownership transfer and unknown evidence before either TERM or KILL veto that effect; canceled/lost-reply/failed signals retain explicit partial outcomes.
- Preview and confirmed text/JSON report each admitted process once, including legacy overlap, and preserve the existing families. Public disposable-Couch conformance keeps the current Claude/Codex fixture slot alive and responsive.

## Plan

- [x] Approve the durable plan and enter change-code through the full-flow gate.
- [x] Add pure exact-target proof and strict protocol shapes; verify the negative authority/identity cases first.
- [ ] Implement the guarded exact executor and supervisor effect owner, with stateful transition and counted-cost tests.
- [ ] Wire preview/confirmed CLI reporting, prove disposable public-command behavior, document, verify and land through SDLC.

## Log

### 2026-10-21

### 2026-10-22 — claim and plan

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
