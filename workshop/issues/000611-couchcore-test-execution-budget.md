---
id: 000611
status: codecomplete
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours: 2.21
card_mirror: '008577335058d38adb27c8ad5cd88d9324272afd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T06:20:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.28
---

# Bring couchcore under a measured test execution budget

## Problem

Couchcore consumes nearly Go's default ten-minute per-package timeout: #591's integrated run abe0a5ed883f06ffe58305cb45a29c09 timed out at 600.245 s, while its automatic main comparison 2064bde22de3d40f5d15091fe41ef443 passed with couchcore at 580.585 s. #589 is affected too. #610 temporarily gives the broad suite an explicit 20m timeout to unblock landings; it does not solve the accumulated execution cost. This is part of ops/test-economy, whose project is maintained by the hub.

## Spec

Land the measured R2 reductions to couchcore's test computation, Git subprocesses and durable fixture setup while preserving every existing assertion, domain point and case disposition. Record per-test before/after evidence and validate the integrated change through the full suite and full-flow review. Keep #610's temporary20m allowance unchanged. Quiet post-landing measurement and any conditional lane work are separate from this landing, as directed by the hub in R5 below. No production changes, new skips, sleeps, retries or longer fixture deadlines.

## Done when

- [x] Measured profiles attribute the expensive tests and record R2 before/after per-test evidence with host/co-tenant and scheduling limits explicit.
- [x] R2 preserves all existing case dispositions and the ordered2,406,528-point domain, with focused race verification and semantic mutation controls.
- [x] The integrated R2 head passes the recorded full suite; mandatory full-flow close review and landing gates govern publication.
- [x] #610's allowance remains unchanged; pair:1 owns one quiet control on landed main after checking no suite lock is held. R4 is deferred and needs separate review only if that control remains above about200s.

## Plan

- [x] Profile and identify causes; obtain the approved R2 cost-first plan and full-flow change-code checkpoint.
- [x] Implement the approved test-only reductions; prove domain/case preservation and focused race/mutation checks; report the complete measured profile.
- [x] Record the full suite on the integrated head, update verification guidance and prepare R2 for mandatory close review and landing, retaining #610's allowance.

## Log

### 2026-10-16 — deferred performance work
- 2026-10-16: closed — R5 hub decision 898fac62: land only R2 and retain #610; TL owns ops/test-economy project. Recorded integrated suite PASS on 303e77d6d50f79d014a56c46ab9e5921a6e556cb (11m45s, core341.494s), CI green; ordered2406528-point domain and every R2 baseline disposition preserved, semantic mutations fail, targeted race20.379s and separate-storage race3.160s PASS. Atlas updated. Quiet main control is explicitly post-landing; no200s qualification or lane implementation claimed.; review verdict: SHIP

Filed unassigned at the hub's request (decision 173f498b-9796-4fd6-a777-07e28ce4107c). #610 is the minimal unblock and lands first, followed by unchanged #591 and #589 retries. The timeout failure alone does not identify a particular slow test; this follow-up owns profiling before choosing the package/shard boundary.

2026-10-16: Dispatched by ops:0 (verified receipt 34a5f4e7-0d39-4ada-9556-d23a2e357561). Claimed and entered planning from fresh main after #612 landed. No test-structure changes authorized before measured profile and hub plan review. A read-only independent audit will map test/helper boundaries while the main slot records an isolated JSON timing run. ARCH-CONSTRAINTS: preserve the 10-minute default with measured headroom and bounded concurrency; profile before choosing mechanism.

2026-10-16: Static audit: 270 same-package test files, 1,369 source top-level tests plus 8 fuzz functions, 317 free helpers; newTestEnv has 208 references across 68 files and many tests use package-private state. TestMain dispatches self-exec aliases/helpers before normal tests and defaults parallelism to 3×GOMAXPROCS unless explicitly set. Existing upstream sharder is specific to ariadne cmd/sdlc, so evaluate reusing ordinary Go discovery/JSON/run semantics rather than duplicating its compile/conversion stack. Raw isolated profile and host metadata: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-611-profile.mIMPri/. The package run has no competing Go suite at its initial process snapshot, but OS load averages were 16.41/21.90/23.80 on 12 logical CPUs; label this package-isolated, not an idle-machine guarantee. First parallel continuation is 143.744s after package start, proving a substantial sequential execution phase before parallel-queue release. No split chosen or code changed.

2026-10-16: Isolated JSON profile PASS: 326.006s package, 328.25s command wall; 1,376 top-level cases (1,359 pass, 17 existing skips). Sequential phase 143.743848s; remaining parallel phase 182.262s. Critical environment correction: inherited GOMAXPROCS=3 and GOFLAGS=-p=3 make TestMain's actual parallel default nine, not 36. Source CPU-count inference alone would have overprovisioned the shards. Slowest test 46.26s; no individual near the package timeout. Durable profile: workshop/plans/000611-couchcore-test-execution-budget-profile.md; raw JSON SHA-256 e45f670a3506a5687ad440a12e4d278c1921a9f2fbe11bbf1b12d3bb1704827a.

Full-flow design draft: workshop/plans/000611-couchcore-test-execution-budget-plan.md. Recommend three ordinary Go-command process shards, sorted round-robin from Go-discovered names, aggregate parallel budget at most nine and honoring smaller caller CPU limits. Predict worst lane 262.78s; proposed qualification <300s isolated and <420s representative load, then remove #610's allowance. Preserve package-private fixtures, TestMain and self-exec helpers. Counter-check every discovered Test/Fuzz/Example exactly once, including existing skip dispositions; every other package executes once. Refuse explicit native/live opt-ins in the default parallel runner and retain their existing serial routes; this boundary is explicit for hub review. No implementation or split yet. Real Make suite baseline with only the final Go command instrumented as JSON is running, holding the existing machine lock. Finish it and reconcile measured loaded distribution before implementation.

## Revisions

2026-10-16 — Full-flow design proposal: use process selection of the intact package rather than a directory split or a fork of the specialized ariadne sharder. Concurrency is based on measured caller GOMAXPROCS=3, not host logical CPU count. Proposed thresholds and live-route handling require hub design review before change-code.

2026-10-16 — R2 direction (receipt f85a0b66-f9d0-47d3-b5f0-51c3a2f54288): hub rejects sharding-first. Make the slowest tests cheaper, remeasure, and propose lanes only if isolated time still exceeds about200s. The previous runner proposal is unapproved history, not an implementation contract. Revised full-flow plan and per-test cause table are appended to the canonical plan/profile. No source implementation changes made.

Diagnostic causes: three domain properties each generate731,566,080 raw cells for2,406,528 admitted points; CPU profile attributes98.35% cumulative samples to iterator work. HostJudgments then discards most dependency coordinates before re-expanding them. Git traces count1,096 calls in12-scenario real reconciliation and931 in recovery convergence; fake SDLC identity uses four real queries per observation, allowing fresh batched evidence without caching mutable state. Bounded continuation test performs100 publications for ten requested rows. Lifecycle shape test allocates stores before rejecting invalid shapes and writes metadata twice. Default producer conformance walks the operator fleet/remotes rather than a bounded test fleet. The loaded-tail SetAside hold-policy test creates nine fully provisioned Git worlds although it chiefly needs real directories/locks and one minimal healthy checkout. Preserve all domain/case assertions and real conformance where it is the claim. No savings claim until repeat measurements.

2026-10-16: Instrumented full-Make before baseline PASS,900.89s wall; couchcore539.310s (serial321.993447s, parallel217.317s), same1,376 top-level dispositions. Go package scheduling/build lag15.799428s separately; no machine-lock wait reported. Targeted CPU/Git-trace diagnostics overlapped its Go phase, explicitly recorded as loaded co-tenancy. Raw full log SHA-2569aa661a8edbdca473d6d326b8552c8a20667fc86c7706359ce2782489a254271. Profile artifact contains the completed baseline. R2 cause-based plan delivered to hub (40482e40-e68a-4f06-8b98-59f16f44943b submitted); waiting for full-flow design response before change-code.

2026-10-16: R2 approved by verified TL receipt d0ece531-7c7e-4a8b-957b-977d70a46574. First full-flow plan review found PQ-1 (numerical repeat/loaded acceptance missing) and minor PQ-2 (named helper strategies). R3 appends concrete qualification: two same-head isolated passes<=200s and one recorded full suite with couchcore<=420s, preserving all cases; report/retain allowance if unqualified. Named consistency predicates and identity contract strategy now explicit. This clarifies comfortable headroom within approved cost-first scope; original mandatory-split clause is superseded by the approved conditional split policy. No implementation yet.


## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale by estimate-source; values are provisional.

Primitive decomposition: issue-spec covers profiling/design/roundtrips at v2's low design0.5h and impl0.1h×0.4. Four smaller-go-module items separately cover ordered-domain pruning, real-Git identity batching, durable lifecycle/continuation fixtures, and bounded filesystem/real-producer conformance; each uses upper0.3h design×0.2 for the thorough approved spec and0.5h impl×0.4. Cross-cutting-refactor covers case scheduling/inventory and default Make timeout wiring beyond those four units (design0.2×0.2, impl0.5×0.4). Real-api-discovery budgets the one Git CLI batching/identity contract at0.3×0.4 implementation. Atlas-docs uses low0.05×0.2 design/0.05×0.4 impl; one milestone-review uses0.3×0.4 impl. No novel library required: reuse Go testing, existing stateful fixtures and standard library. Familiar territory×1.0; thorough-plan design buffer15%. No scheduler work included.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.50 impl=0.04
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.04 impl=0.20
item: real-api-discovery design=0.00 impl=0.12
item: atlas-docs design=0.01 impl=0.02
item: milestone-review design=0.00 impl=0.12
design-buffer: 0.15
total: 2.21
```

2026-10-16: Plan-quality round2 CLEAN: PQ-1/PQ-2 addressed; no new findings. Estimate derived only after plan cleared, with design subtotal0.79×1.15 + implementation1.30 =2.2085h, recorded as2.21h on the card.


2026-10-16 — Implementation checkpoint after approved R2/change-code: test-only changes batch fixture Git identity into one fresh query (with cancellation, lease and malformed/foreign identity contracts), prune only inconsistent evidence prefixes, project host-property dependency bases directly, remove 90 unobserved continuation publications, validate lifecycle shapes before IO and seed the same named revision, parallelize independent real-Git cases, and bound live real-producer/set-aside fixtures to their observed inputs. No production source or scheduler changes. All original 2,406,528 accepted evidence points and their ordering are pinned against pre-change SHA-256 fingerprints, including stride-23 and reconstructed host variants. Drop/duplicate/reorder iterator mutations each fail the fingerprint guard semantically. Focused property tests PASS (0.57–1.47s, fingerprint 1.88s); durable lifecycle/continuation checks, race run and five refusal/redaction mutation controls completed; identity contracts and representative provisioning group PASS. Bounded live sdlc fleet now exercises ready/holds-work/missing/needs-recovery and PASS 2.77s; minimal real set-aside policy PASS 0.29s. These targeted timings are diagnostics under co-tenancy, not final before/after qualification. Next: commit this checkpoint, run the combined targeted race checks, then two consecutive isolated same-head JSON runs with original flags/environment. Full-suite loaded qualification, timeout removal, review and landing remain pending.


2026-10-16 — First full post-change diagnostic (73cc1f2021f2927b3c958f73e133b5117766a5a3): 278.599s package /286.19s command versus326.006s baseline; qualification FAIL (over200s and one fixture regression). Same flags/GOMAXPROCS/GOFLAGS. All1,376 old top-level names present, five new regression tests; unchanged17 top-level skips. TestSeparateGitDirectoryEnrollmentPreservesStorageAuthority failed because the fake's new membership check assumed primary/.git was a directory. Corrected it to resolve the primary Git-file authority through the filesystem (including relative pointers), preserving one fresh Git process and foreign-repository refusal. Extended the one-process contract to a separate-Git-directory primary: RED before fix, identity+existing separate-storage race group GREEN3.160s. No product regression or assertion weakened. Combined changed-fixture/property/table race group PASS20.379s. Set-aside negative controls reject ignored holds (nil-error classification panic) and lost movement (sentinel absent). Keep #610 allowance; no shard implementation approved. Record one corrected-head full isolated run to verify inventory and provide the residual profile for the hub's conditional lane decision; no second qualification repeat if still over200s.


2026-10-16 — R2 residual report: corrected head8b9977e09121c893686f3049068b63e8357bafe8 PASS340.356s package/346.97s command under a competing full suite that began mid-run. Every old top-level/subtest name has identical terminal disposition; five new top-level guards,17 unchanged existing top-level skips. The prior package diagnostic278.599s is over200s but failed the subsequently fixed separate-storage fixture, so it is not a qualified green isolated control. Full per-test table, hashes, queue accounting and gaps are now durable in the profile. Cost-only acceptance not met; #610 retained. R4 proposes three intact-package lanes with aggregate CPU3/workers9/build jobs3 and models190s diagnostic/237s loaded worst lane; no runner code written. Await hub decision on this proposal versus first scheduling a corrected quiet control, then a new full-flow plan-quality checkpoint before any structure change. No full-suite/SHIP/landing claim.


### 2026-10-16 — R5: land R2 independently of the residual capacity question

Verified hub decision898fac62-3c98-445e-9bf3-0bd7b9f78de2 explicitly authorizes closing/landing the existing reductions now while retaining #610. This supersedes R3's200s/repeat and420s/full-wall requirements as gates for this landing; no claim is made that those performance budgets passed. R4 is deferred to a separate item/review, conditional on one quiet isolated control after landing. Pair:1 owns that control on fresh main, first checking the machine-wide suite locks are all free and observing co-tenants during the run; the hub owns any resulting project/capacity decision. Full recorded correctness suite and mandatory close review still apply. Active Spec/Done when/Plan above are reconciled to this scope. Prior criteria are preserved below as historical scope rather than erased or silently checked complete.

#### Superseded pre-R5 scope

> ## Spec
>
> Profile couchcore's slowest tests using recorded per-test execution data under both isolated and suite-equivalent load. Distinguish real IO/setup cost, serialization and scheduling waits. Choose a measured package split or deterministic sharding strategy that preserves test inventory, fixture ownership, assertions and cancellation. Set and defend a practical execution budget with headroom, and remove #610's temporary 20m allowance when the measured suite is reliably within that budget. No retries, skips or longer sleeps as a performance fix.
>
> ## Done when
>
> - [ ] Recorded profiles identify the slowest tests and quantify execution separately from suite queue wait, under idle and representative load.
> - [ ] A justified package split or sharding design preserves every test and its isolation contract, with deterministic coverage and bounded concurrency.
> - [ ] Before/after full-suite measurements demonstrate the agreed execution budget with headroom; fixture deadlines and assertions remain intact.
> - [ ] The temporary 20m allowance from #610 is removed or replaced by the measured, documented steady-state budget.
>
> ## Plan
>
> - [x] Measure an isolated couchcore run with Go JSON events; distinguish execution, paused parallel tests, compilation/setup, and machine co-tenancy.
> - [ ] Compare suite-equivalent timings, audit fixture/package boundaries, and send a measured full-flow design to the hub before any split.
> - [ ] After design approval, implement and qualify the agreed test organization, inventory preservation and execution budget; remove the temporary allowance; full-flow review and landing.


2026-10-16 — R5 integrated correctness qualification: integrated current main (including #595, which independently removed two obsolete core top-level tests and one obsolete subcase), pushed PR371, and recorded sdlc-local/suite PASS on303e77d6d50f79d014a56c46ab9e5921a6e556cb. Command make PAIR_GO_TRIMPATH=-trimpath test completed exit0 in11m45s, couchcore341.494s, wrapcmd78.388s, all shell/editor and other Go packages PASS. Full log SHA256 prefixd436b8aaf565; saved under the common Git sdlc/local-checks/303e77d6d50f79d014a56c46ab9e5921a6e556cb/suite.log. This is full-suite loaded correctness evidence; no quiet200s claim. Server conformance/merge-check/local-gates checks PASS. Atlas describes bounded real fixtures and ordered-domain guard; TL owns project. Ready for the binary-owned full-flow close review under R5, preserving the allowance and explicit pair:1 post-landing quiet-control obligation.
