---
id: 000687
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 2.015
collaboration-mode: collaboration
card_mirror: '9b71ef9a7f95444c2378a533f5eb52a698ebd4c5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T21:13:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Recognize Couch supervisor retention registration

## Problem

During #677's real diagnostic preview, the running Couch supervisor PID45023 was classified as an ownerless legacy runtime and blocked253 current diagnostic rows containing23,462,164,824bytes. Those live current generations were correctly retained; no process intervention was performed. Closed unheld old generations now have separate reader-only proof.

Couch already calls storagegc.Coordinator.RegisterRuntime in couchcore/couch.go, but diagnosticlog.OSInspection/VerifyWriters recognizes PAIR_RETENTION_PROTOCOL=1 or the particular log's writer registrations. The supervisor has no per-thread PAIR_DATA_DIR/tag and its runtime lease is not consumed by that proof, so being registered in the coordinator does not establish protocol awareness there. Missing protocol means unrecognized registration, not an old start time.

## Spec

Make the supervisor's retention protocol participation recognizable through the existing lifecycle/identity-fenced registration boundary, as its clients' participation is. Do not broadly exempt processes by executable name or weaken unknown/open-holder protection. Decide whether to consume the exact registered runtime lease or publish a protocol capability through the existing launch boundary. Preserve birth identity and cancellation fences.

Proposed for hub approval: publish explicit diagnostic protocol version1 on the existing coordinator runtime registration before Couch admits its coordinated store. The selected-root inspection adapter consumes only a matching Couch-role PID/birth and supported capability; it shares the same bounded sweep observation used by rooted writer maintenance and GC. Old/missing/future capabilities grant no recognition. Actual open holders, uncertain identity/metadata and cancellation retain their existing protection. Runtime-registry reads accept and surviving-record writes preserve unknown envelope, registration and nested process members (ARCH-EVOLVE), without relaxing unrelated retention schemas or changing comparable ProcessIdentity.

## Done when

- A stateful regression in the real supervisor shape (no per-thread tag or PAIR_DATA_DIR, valid runtime registration) is recognized as protocol-aware without a global current-log block.
- A missing/replaced/unreadable registration stays blocked; a supervisor actually holding a diagnostic payload still protects it.
- The next operator-coordinated Couch restart verifies the supervisor is registered; no process is stopped or relaunched as part of filing this follow-up.
- Unknown JSON members at the registry, registration and nested process levels survive a capability upgrade, new registration and positively-dead pruning; malformed known fields, ambiguous JSON and unsafe metadata remain blocked.
- Counted N=8/16 sweeps retain one runtime-registry read and one native snapshot pair; each later sweep refreshes lease evidence.

## Plan

See [implementation plan](../plans/000687-couch-retention-registration-plan.md). Hub approved the plan; enter implementation through change-code.

- [x] Preserve unknown runtime-registry members through bounded validated reads and surviving-record rewrites; add the explicit diagnostic capability to the existing lease.
- [x] Consume exact identity-fenced participation from the selected root through shared writer/GC inspection, with stateful sequence and counted sweep regressions.
- [x] Publish from Couch, qualify owned/native boundaries and document the operator restart check.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; the source is flagged stale, so calibration is provisional. Scope is one repo including qualification, docs and close review.

Two single-concern modules cover the codec and rooted inspection adapter: each starts at 1.0 design/0.8 implementation hours. Existing encoding/json/strictjson and SweepSnapshot/ProcessProbe cover the library seams, halving design; the resolved plan applies the 0.2 spec discount, yielding 0.1 design each. Implementation values are scaled once by 0.4. The smaller-module row covers publication/composition and constructor tests (0.3 design x0.2, 0.5 implementation x0.4). Issue/spec uses 1.0 design x0.2 and 0.2 implementation x0.4. Owned native qualification takes the real-API-discovery upper bound 0.6 x0.4; no new external API. Atlas and one close review use their documented ranges. Familiarity is 1.0 from #677; the thorough-plan design buffer is 0.15. No cross-repo overhead.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.2 impl=0.08
item: greenfield-go-module design=0.1 impl=0.32
item: greenfield-go-module design=0.1 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 2.015
```

## Log

Filed unclaimed at the hub's request in receipt42a427eb-c28b-4124-9ef2-8c09ed6b56cd. #677 PR442 retains the current-log block and does not fold this supervisor lifecycle change into its retention policy implementation.


### 2026-10-18 — claim, source evidence and proposed plan

Verified fresh hub assignment c1eb775b-6d04-44d9-93e5-013b5940f0bd, claimed #687 and ran start-plan. The first start-plan refreshed the ariadne substrate and explicitly refused branching under the old binary; its requested rerun created the issue branch. No product code or live data mutation has occurred. start-plan currently lists ariadne#454 in flight, so ARCH-EVOLVE is an explicit hub requirement even though the delivered principle registry has not added it yet.

Couch constructor already registers the exact current PID/birth as couch-runtime before its coordinated thread store. The lease producer originated in #239, predating the diagnostic protocol, so treating every existing lease as that later capability is unsafe. Proposed explicit versioned capability on that same record, consumed from already-selected roots rather than environment/HOME guessing. Current runtime reads reject additive fields, and RegisterRuntime rewrites through a lossy typed projection; a private wire codec preserves unknown envelope/record/process members without making shared ProcessIdentity non-comparable or relaxing every retention schema. A bounded read-only helper confirmed the existing strictjson duplicate-key primitive and these three preservation depths.

The durable plan names all writer/GC consumers, unchanged open-holder protection, PID reuse and cancellation, one-read/one-native-snapshot N/2N budgets, alternatives, and a proposed owned/live arrangement. Plan approval and the change-code gate are still outstanding; no implementation tests or builds beyond the tool's required substrate refresh have run.

### 2026-10-18 — hub approval

Hub receipt34b6b719-dab8-4751-aeff-2a87883e7d88 approves the explicit diagnostic_protocol v1 capability, one selected-root adapter, unchanged exact-holder protection, and unknown-member preservation with strict known checks. The owned child conformance/read-only preview arrangement is approved; the existing supervisor remains blocked until the operator restart publishes its capability. No pair:0 staging or live apply. Proceeding to the full-flow change-code gate.

### 2026-10-18 — plan-quality round 1

PQ-1 requested named function-level test strategies instead of prose inventories. Compressed the durable plan's codec, registry, predicate, adapter and composition test rows accordingly; no architecture/scope change. Re-running the remembered plan gate before implementation.

### 2026-10-18 — implementation and owned/read-only qualification

change-code admitted the full flow after PQ-1 was addressed and the 2.015-hour v3.1 estimate was recorded. Estimate-quality was INFO: integration/qualification and shipping allowances are tight. The two module rows include their adversarial unit/integration tests, the smaller-module row covers wiring and constructor coverage, and the review row includes overlapping recorded checks/CI/landing supervision; elapsed checks may exceed that allowance and will be captured by actuals rather than hidden.

TDD RED: TestDiagnosticSupervisorLeaseRecognizesOnlyPublishedCapability failed because a lease with diagnostic_protocol1 still left the supervisor as legacy; TestCoordinatedCouchRegistersActualRootRuntime failed with DiagnosticProtocol0. Both now pass after selected-root consumption and explicit constructor publication. Codec work independently observed RED for unknown-member rewrites, malformed known fields and missing capability, then GREEN. Unknown envelope/row/process members survive changed append, upgrade and dead pruning; cancellation/persist failures preserve previous bytes.

Affected full packages storagegc (38.863s), diagnosticlog (89.050s) and gcruntime (35.929s) passed. Artifactpath initially found three obsolete ignored runtime-bundle Lua mirrors removed from tracked sources by #664; make runtimebundle-generate reconciled local assets, then the whole artifactpath package passed15.798s. The whole-tree Godoc guard passed0.479s. Full couchcore run is still active. N8/16 tests hold one registry read, one native process/open-file snapshot pair and one matching liveness probe; next-sweep lease removal is observed. Owned-child native FD/PID/birth conformance passed3.33s with the 2s native bound unchanged.

Read-only real preview completed4.463s, exit0, no stderr:519 rows /33,335,609,869bytes, zero eligible. Existing supervisor45023 has its original couch-runtime lease with no diagnostic_protocol and correctly blocks253 current rows /23,917,067,119bytes. No live apply or staging. runtimes.json remained byte-identical, SHA256 ac43fa7db16e4f5e22e59fcb7bf4a209953fdfab89d91cc642ba692d482ff426. Retained evidence: /Users/xianxu/workspace/pair/.git/pair-687-evidence (before/after registry, preview JSON/stderr and SHA256SUMS); manifest SHA25669932b22f2bb863b9208b87ac90f1489ca324393238af17988ec07e96ec564e0. Hub report197c66ca-bd69-41c4-84a1-026271ac5a46 was submitted. Real supervisor participation remains the approved next-restart check.

Restart check for hub/operator: after restarting onto this landing, read the selected root's .retention/runtimes.json and match the new supervisor PID/birth, couch-runtime role and diagnostic_protocol1. Run pair gc --diagnostics-only --json against that root; its current-log rows must no longer name the supervisor as an unrecognized runtime. Actual open holders, old clients, changed generations and other blockers remain protected. A new preview and separate apply approval precede any deletion.

### 2026-10-18 — draft PR and recorded suite

Opened early PR447 through sdlc pr, then converted it to draft for CI overlap (sdlc pr has no draft option). All four server checks passed on a508b7ef803197b892f0d28f6ae4333165b07a0f: macOS conformance2m12s, merge-check1m11s and both local-gates checks. Recorded suite attempt a4b9bb7cbcb08c0e833fc1a05cfa062f is queued from this slot on that head; its retained inflight log is .git/sdlc/test-stats/inflight/a4b9bb7cbcb08c0e833fc1a05cfa062f/run.log. Full couchcore package run remains active. Implementation checkboxes are complete; suite completion remains a landing gate. Starting the binary-owned close review while the recorded suite waits, per tl-worker's approved overlap.

## Revisions

### 2026-10-18 — dispatched capability and record-evolution design

The fresh hub assignment chooses consumption of the existing coordinator lease and requires ARCH-EVOLVE. The proposed explicit capability avoids retrospectively trusting #239-era producers. Added corresponding Spec/Done-when detail, unknown-member round-trip checks and the counted shared-observation acceptance. Awaiting the hub's plan/live-arrangement approval.
