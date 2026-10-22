---
id: 000801
status: working
deps: [pair#758]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 3.75
collaboration-mode: collaboration
card_mirror: '5fd338a4700fe725dea17a988673ba10ffb649bd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T07:20:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# External hold-open Couch conformance bridge

## Problem

parley.nvim#405 needs a real hold-open #758 editor endpoint producer for external Lua conformance. parley.nvim#408 exercised only public adopt-store/direct-store registry commands and did not start a daemon. Pair's cmd/internal/testcouch owns genuine disposable fleets but is a Go testing.TB API, with no external driver entrypoint. Reimplementing its lifetime and teardown in Parley would duplicate authority and risk live-state leakage.

## Spec

Provide a bounded Go testcouch owner/driver and thin external wrapper, reusing Build/New/Prepare/Start/Close/AssertCleaned. Build a candidate or accept an explicit candidate Couch executable, allocate a fresh private isolated root and private PATH from an empty environment, and start only the owned disposable Couch. No live roots, operator sessions, installed agents, network or default singleton authority.

Coordinate final argv/readiness/stop contract with parley.nvim:1 before code. Proposed external shape: explicit candidate binary plus owned scratch selection; one readiness JSON line with Version, canonical Namespace, IPC Socket and complete Environment map for clients (including private store/identity/Couch selection), diagnostics on stderr. Readiness must follow actual service admission, never mere socket creation. Bound readiness/output/startup and total lifetime. Stdin EOF or TERM requests the same idempotent joined stop, returning cleanly only after Close and AssertCleaned prove cleanup. On uncertain cleanup retain exact owned evidence/root and exit failure; no guessed PID cleanup or recursive removal before descendant absence. Parent owns launch timeout/process group; document crash recovery limits.

parley.nvim#405's real-producer acceptance depends on this bridge and #758. Per TL5e43c052, #405 may land its M1 on fakes and native unit evidence; only real-producer acceptance waits. This issue is independent work from main after its producer dependency lands, not added to #758's current review.

## Done when

- External driver can start a given/built producer, read a single confirmed readiness envelope, run Parley's isolated Lua driver in the emitted environment, and stop by EOF or TERM.
- Tests cover startup failure, early EOF/TERM, repeated stop, client/child cancellation and uncertain teardown without live-state access; genuine native conformance proves exact cleanup.
- Interface confirmed with parley.nvim:1, docs/entrypoint and inventories updated, recorded checks and normal whole-issue gates pass.

## Plan

Durable implementation plan: [external Couch bridge](../plans/000801-external-couch-bridge-plan.md). Full flow, one atomic close; design/consumer confirmation pending.

- [ ] Confirm executable/envelope/bounds with parley.nvim:1 and obtain hub design approval.
- [ ] Extract shared error-returning candidate/fleet owner seams; existing testing.TB wrappers retain behavior and cleanup.
- [ ] Implement pure protocol/lifecycle and bounded cancellable IO through the same stateful fake/native seam.
- [ ] Add the thin executable/atomic build target, genuine native bridge and Parley consumer conformance, inventories and documentation.
- [ ] Record checks, close review and land through sdlc.

## Log

2026-10-22: filed unclaimed by pair:6 per TLe632079d/5e43c052 after inspecting testcouch and #408's actual direct-store probe. parley.nvim:1 provided proposed envelope and stop interface in receipt59bd1da0. TL will dispatch to a free Pair slot; #758 fixes its own review findings first.


## Revisions

### 2026-10-22 — dispatched bridge contract and shared ownership

#758 and #773 are landed; start-plan now owns a fresh main-based #801 branch. Consumer receipt b1bdc65f pins Version/Namespace/Socket/complete Environment, private couch+nvim PATH, explicit candidate/scratch selection, real service roundtrip, stderr-only diagnostics and joined EOF/TERM cleanup; receipt1fb645cd confirms its existing driver will consume this bridge rather than add a Couch launcher. Proposed executable is bin/couch-conformance-bridge with --couch ABS and --scratch-parent ABS; omit --couch to build the candidate. Proposed extra Documents array exposes exactly two actual registered private fixture IDs/paths. Proposed absolute startup300s, active lifetime600s, joined-stop15s and64KiB readiness/diagnostic caps need consumer confirmation and hub approval before code. The durable plan names the existing testing.TB coupling, shared error-returning extraction, stateful fake, late-output/stop ordering, cleanup uncertainty and real conformance. No standalone testing.TB shim, duplicate launcher or installed Pair fallback.

### 2026-10-22 — planning catch-up seed recovery

First start-plan catch-up regenerated .github/workflows/merge-check.yml and correctly refused a dirty resting branch (ariadne#557). Followed authenticated hub26eccf46: saved the920-byte diff to /Volumes/TestScratch/tmp/pair-slot7/pair801-ariadne557-seeded-94ukwvc8.patch, restored only that seeded file, then re-ran bare start-plan successfully. No workflow change is part of #801; local-gates verification is never self-applied.


### 2026-10-22 — consumer contract confirmed and hub approval complete

Hub0cdb7c57 approves subject to consumer confirmation; canonical consumerd0f4eb15 confirms all proposed bounds/argv/envelope with the parent-plus-side-doc layout now specified in the durable plan Revisions. Ordered Documents are artifacts/first.md and artifacts/first.side/second.md (type: side-doc, parent: ../first.md), with distinct registry IDs in Namespace. Keep invocation-PATH-selected native nvim for separate0.11.7/0.12.5 qualification; parent allows630s, closes editors before EOF and retains uncertain evidence. No additional envelope fields or CLI verbs. Proceed to plan-quality/change-code; no implementation before it clears. #739 is separately pushed ready at96062db5b1a73fe327f171b1e93e7a28966d1068 and remains unstaged per hub; it is never stacked into this branch.


### 2026-10-22 — plan-quality round1 refinements, no implementation yet

Automatic compact-design inference selected quick flow; re-ran with full flow to match the approved refactor's known code size. Plan-quality returned PQ-1 (candidate build isolation/provenance) and PQ-2 (function-level adversarial test strategies). Appended the explicit shared testpair.PlaceCandidate extraction, strict offline/allowlisted build config, checkout provenance and deliberate shared Go cache boundary; no installed Pair or ambient PAIR_TEST_BINARY in the strict path. Added the canonical named-function strategy table superseding illustrative test inventories. These refine the approved same-owner/isolation contract without adding a resolver, cache or executable protocol. Restart prompt arrived during review, but the review completed normally; no lost runner or duplicate dispatch. Re-run the gate before estimating/coding.


## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md, using the v2/v2.1 primitive procedure. Method A only, derived after plan-quality CLEAN round2. estimate-source marks calibration stale, so these hour values are provisional. Familiarity1.0: existing Go startup/fixture/ownership stack, not a novel runtime. Thorough plan discounts implementation-component design by0.2 and uses15% design buffer; issue/spec design is counted once without that discount. Implementation values below are already40% of v2/v2.1 rates, not scaled a second time.

Breakdown in block order: issue/spec and consumer contract (raw0.8 design/0.2 implementation); two bounded shared refactors (candidate/testpair and Fleet ownership/reporting, each raw0.4/0.5); pure bridge model/protocol (raw1.0/0.8, design halved because existing standard-library JSON/context and shared owner cover the stack); runner/service/native IO integration (raw1.5/1.5); executable/atomic target (raw0.2/0.5); one native endpoint/conformance discovery surface (raw0/0.6, both nvim versions qualify the same API); atlas/docs (raw0.1/0.2); one close review (raw0/0.5). Tests are included in each implementation primitive, not charged again as a separate test phase. No external library search/dependency or cross-repo code change is needed; consumer coordination uses its existing driver and is included in issue/API work.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: cross-cutting-refactor design=0.08 impl=0.20
item: cross-cutting-refactor design=0.08 impl=0.20
item: greenfield-go-module design=0.10 impl=0.32
item: api-integration design=0.30 impl=0.60
item: smaller-go-module design=0.04 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 3.75
```

Design1.42h×1.15 + implementation2.12h =3.753h, rounded3.75h. Queue waits and operator-controlled #739 shutdown are not implementation work in this estimate.
