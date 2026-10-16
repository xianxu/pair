# #611 couchcore timing profile

## Isolated package baseline

Command: `/usr/bin/time -l go test -trimpath -json ./cmd/internal/couchcore -count=1`.

Code pin: `4a1245d43c052a63379252c43f0ada579be78a1b` (planning commits only above main). Go 1.27.1, darwin/arm64, 12 logical CPUs. Caller environment: `GOMAXPROCS=3`, `GOFLAGS=-p=3`; couchcore TestMain therefore defaults to nine parallel tests. Package-isolated: no other Go suite appeared in the initial process snapshot. OS load was 16.41/21.90/23.80, so this does not claim a fully idle machine.

- PASS, package execution **326.006 s**, command wall **328.25 s**.
- **1,376** discovered/executed top-level cases: **1,359 pass, 17 skip** (existing opt-in/platform dispositions).
- **379 sequential** cases; **997** cases call `t.Parallel`.
- First parallel continuation: **143.743848 s** after package start. Sequential reported elapsed sum: **143.34 s**.
- Remaining package interval: **182.262 s**. Parallel reported elapsed sum: **1,596.18 s**, overlapping across about nine active tests. Do not add overlapping per-test times to package wall time.
- Command user/system CPU: 199.81/239.01 s; maximum resident size reported by time: 594,591,744 bytes. This is command instrumentation, not a summed concurrent process memory profile.
- Raw JSON SHA-256: `e45f670a3506a5687ad440a12e4d278c1921a9f2fbe11bbf1b12d3bb1704827a`.
- Local raw evidence: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-611-profile.mIMPri/isolated.jsonl`, `isolated.time`, `isolated-host.json`, `isolated-summary.json`. These are diagnostic temporary files; this summary is the durable evidence.

| Slowest completed top-level test | Execution seconds | Scheduling |
|---|---:|---|
| TestReconcileAgreesWithRealGit | 46.26 | parallel |
| TestRecoverPlanStepsConverge | 42.55 | parallel |
| TestReAdoptionExitsAreTotalAndCoded | 28.75 | parallel |
| TestContinuationRequestsBoundedOwnedSet | 28.45 | parallel |
| TestHostJudgmentsReadOnlyHostFacts | 25.91 | parallel |
| TestDeriveRecoverPlanIsTotalOverTheEvidenceDomain | 24.92 | parallel |
| TestRecoverReconcileReadingIsMetamorphic | 24.67 | parallel |
| TestFleetInventoryLiveConformance | 24.61 | parallel |
| TestObserveEachBrokenResource | 24.29 | parallel |
| TestRepositoryNamesAtConversationAllocationBoundaries | 23.96 | parallel |
| TestSetAsideHoldsMatchThePlan | 22.97 | parallel |
| TestRecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates | 11.01 | sequential |
| TestAcceptanceReportBudgetTenSlots | 9.74 | sequential |
| TestInterruptedSwitchResumePreservesTargetAuthority | 9.06 | sequential |

Source inspection: real Git repositories/worktrees and index discovery dominate the identified setup paths; per-operation attribution has not been measured. SetAsideHoldsMatchThePlan constructs nine independent provisioned fixtures. RecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates constructs two separate six-slot worlds. Parent-killed hour-long helper sleeps are intentional lifetimes, not evidence of an hour-long test.

## Partition model (prediction, not verification)

Sort Go-discovered top-level names and distribute round-robin; keep total parallel budget nine. For three shards (459/459/458 names), measured sequential sums are 38.24/53.80/51.30 s and parallel elapsed sums 480.02/481.71/634.45 s. Dividing each parallel sum by three yields approximate busy times **198.25/214.37/262.78 s**. This omits changed contention, orchestration and long-tail scheduling; measured runs must qualify the design. Two shards predict a busiest lane of 300.75 s; four shards with two workers each predict 288.15 s. Three is the proposed smallest useful bounded configuration for the current envelope.

## Suite-equivalent baseline

In progress. A temporary Makefile includes the real root Makefile and overrides only the final Go recipe with `go test -json ./... -count=1 -timeout 20m`; all existing prerequisites, environment filtering and the machine-wide suite lock remain. The 20m is #610's current baseline, not an increased allowance. Command stdout and time metadata are in `suite-before.log` and `suite-before.time` beside the isolated evidence. Record queue wait, full wall and couchcore events separately before any split is implemented.


## Cost-cause audit (2026-10-16 revision)

All ten top-level tests already call t.Parallel. Their durations overlap; their sum is not removable package wall time. No deliberate sleeps appear in these ten test bodies. Real IO families keep real Git where it is the assertion, rather than replace a conformance proof with its own fake.

| Test (isolated package elapsed) | Observed cause | Proposed cheaper execution preserving the claim |
|---|---|---|
| ReconcileAgreesWithRealGit (46.26s) | 12 independent cases; each initializes primary/bare remote, provisions slot/dependency, perturbs, reconciles twice and observes. Command-count trace recorded 1,096 real Git invocations. Only 60 are the five-command primary setup repeated 12 times; repeated observations dominate. | Batch the fixture's four identity queries into one real Git process; keep all12 real scenarios, both passes, fake agreement and idempotence. Independent child cases can use existing t.Parallel budget; no shared mutable worktree. |
| RecoverPlanStepsConverge (42.55s) | Six real worktrees plus primary, OS catalog and five resume operations/reports. Trace: 931 Git calls, including144 branch,145 HEAD,150 top-level and270 common-dir queries. | Same fixture identity batching eliminates three spawns per identity read (~432 calls in this trace). Preserve real discovery, five resumes, untouched slot5 and replay refusal. |
| ReAdoptionExitsAreTotalAndCoded (28.75s) |110 attempted lifecycle shapes; each allocates a store and attempts durable creation; park cases add a separate metadata transaction. Invalid shapes still create directories/read store state before validation refuses. | Pure ValidateThreadRecord before store allocation, with the same invalid-name dispositions; put the existing name on the initial record to remove redundant metadata writes. Retain real write/cleanup/readback for every admitted shape and all lifecycle dimensions. |
| ContinuationRequestsBoundedOwnedSet (28.45s) |100 CreateThread +100 PublishContinuation transactions for ten requested unique rows;90 published payloads are unobserved. | Keep100 records,20 duplicate input addresses,ten actual requests, corrupt outsider, metadata secrecy and size assertions; publish only ten requested continuations. Prove full-scan mutant still fails. |
| HostJudgmentsReadOnlyHostFacts (25.91s) | Walks731,566,080 raw cells, discards all dependency variants except DepNone/Clean, then re-enumerates variants. | Generate only baseline dependency coordinates first; retain all102,204 bases and2,406,528 admitted variants, independent host judgments. |
| DeriveRecoverPlanIsTotalOverTheEvidenceDomain (24.92s) | Same731-million raw Cartesian traversal; only2,406,528 points reach classifier assertions. | Check existing physical consistency predicates at their earliest complete prefix. Factor predicates once; preserve the exact ordered admitted stream, no sampling. |
| RecoverReconcileReadingIsMetamorphic (24.67s) | Same full traversal, then stride23 of admitted points. | Reuse the pruned iterator while preserving accepted order, exact sample names/count/fingerprint and existing stride/variants. |
| FleetInventoryLiveConformance (24.61s) | One real installed sdlc inventories the operator fleet and contacts tracked-repo remotes; workload scales with unrelated local repos/network. Producer source warmRecords and LookupRepoClaims confirm this. | Invoke the same real producer against a small disposable real-Git fleet with controlled healthy/dirty/missing members and local remotes. Keep decoder/vocabulary assertions, nonempty rows/slots, and real CLI conformance; no new skip or fake producer. |
| ObserveEachBrokenResource (24.29s) | Ten independent cases each builds/provisions a primary and slot, breaks one resource and observes the complete healthy remainder. | Shared fixture identity batching, plus independent child t.Parallel under current aggregate limit. Keep all ten perturbations and every healthy-resource assertion; no stale Git-result cache. |
| RepositoryNamesAtConversationAllocationBoundaries (23.96s) | Five real repository fixtures (three new, two fresh); fresh creates linked worktree and discovers through OS catalog. Cases take3.67/5.70/4.11/6.58/3.90s. Fake registration is immediately established, not a polling delay. | Same identity batching and independent child scheduling. Preserve Unicode/punctuation/max-length names, both entry routes and exact tag/session/path assertions. |

### Diagnostic attribution evidence

A focused CPU profile of the three evidence-domain properties ran16.53s wall and sampled34.64s CPU: forEachEvidence cumulative34.07s (98.35%); its own flat15.66s plus consistent flat15.24s total89.20%. Actual classifier assertions accounted for little CPU. Raw profile files: properties.cpu/properties.test/properties.jsonl beside the initial JSON.

Git call tracing uses a temporary shell wrapper that records cwd/argv then execs the actual /usr/bin/git. Counts are evidence; wrapper timings are instrumentation-affected and not performance baselines. Reconcile trace completed61.608s and recovery trace37.068s while the representative suite was also active. Files: reconcile-git.tsv/recover-git.tsv and corresponding traced JSON. No repository implementation was modified.

A read-only independent source audit supplied case counts and fixture ownership; those static counts are cross-checked against Go event names during implementation. Do not share a mutable provisioned checkout across destructive cases, cache mutable Git observations, reduce the totality domain/stride, or substitute SlotWorld for the real-Git half.

Additional loaded-tail audit: TestSetAsideHoldsMatchThePlan is22.97s in the isolated profile and46.50s in the representative run so far. It constructs eight independent agent cases plus a recovered-checkout case, each with provisioned host/dependency Git state; the same identity batching applies. Independent child scheduling can remove the long serial table tail under the existing aggregate limit. The names-boundary trace completed with141 Git calls, including18 branch,23 HEAD,23 top-level and26 common-dir queries.


### Completed full-Make pre-change baseline

PASS, command wall900.89s (15m0.89s); couchcore execution539.310s. Same1,376 top-level dispositions (1,359 pass/17 skip), no lost cases. First parallel continuation321.993447s after core start; remaining217.317s. Core started15.799428s after the first Go event, which is package scheduling/build lag, not package execution. No machine-suite-lock wait was reported; do not conflate unobserved queue time with measured zero. During Go execution the focused CPU and Git-trace diagnostics ran concurrently; this is a loaded sample, not an idle comparison. All other packages and shell/editor prerequisites passed.

Slowest loaded core tests: real reconcile65.76s, SetAside holds46.50s, recovery convergence41.13s, bounded continuation38.49s, host judgments34.43s, metamorphic33.94s, readoption33.67s, observer32.88s, repository names32.43s, totality30.03s, live fleet27.50s. These elapsed durations overlap.

Raw full log SHA-256:9aa661a8edbdca473d6d326b8552c8a20667fc86c7706359ce2782489a254271. Files suite-before.log/suite-before.time/suite-before-summary.json in the diagnostic directory above. This is an instrumented planning baseline, not an sdlc-local/suite status on the branch.


## R2 post-change measurements (2026-10-16)

Same command: `/usr/bin/time -l go test -trimpath -json ./cmd/internal/couchcore -count=1`; same GOMAXPROCS=3 and effective GOFLAGS=-p=3. No timeout, skip, retry or increased parallel-worker limit added.

- Head `73cc1f2021f2927b3c958f73e133b5117766a5a3`: package278.599s, command286.19s, serial132.688168s, remaining145.911s. Diagnostic FAIL: one new fixture membership guard incorrectly rejected the existing separate-Git-directory enrollment test. This is not a qualified green performance result. All old names still ran; the failure was corrected before the next run. Initial load17-ish, with a targeted couchcmd run visible before launch; no competing full suite observed at that initial snapshot. Raw `after-1.jsonl` SHA256 b98c37e662dc58bd8f05c7aa7b284769d5f0069714fb1c173a3ae6646bd6987b.
- Corrected head `8b9977e09121c893686f3049068b63e8357bafe8`: PASS package340.356s, command346.97s, serial160.554945s, remaining179.801s. A second full suite started during this run: mid-run snapshot shows two couchcore binaries plus other Go package binaries. Therefore this is a **loaded correctness/profile sample**, not the corrected isolated qualification or the required recorded full-suite check. Start/mid/end load and process snapshots are retained next to raw data. Raw `after-corrected.jsonl` SHA256 761d92d424de99bd41c17315ed6b26a568732d35e5447aecfd95268a5a8ea698.
- Exact event comparison: every baseline top-level and subtest name has the same terminal disposition in the corrected run. No removed case, no changed skip. Five new top-level guards and16 new subtest names: seven malformed-identity cases and nine newly named set-aside cases. Total1,381 top-level cases =1,364 pass/17 pre-existing skips. All12 real-Git reconciliation cases, ten resource perturbations, five names,110 re-adoption shapes (74pass/36skip) and all2,406,528 accepted evidence points/order retained.

| Test | Before isolated seconds | After diagnostic seconds | Corrected loaded seconds |
|---|---:|---:|---:|
| ReconcileAgreesWithRealGit |46.26|sum children38.89; window105.56|sum children45.27; window114.14|
| RecoverPlanStepsConverge |42.55|18.67|19.16|
| ReAdoptionExitsAreTotalAndCoded |28.75|16.71|24.43|
| ContinuationRequestsBoundedOwnedSet |28.45|19.43|30.83|
| HostJudgmentsReadOnlyHostFacts |25.91|0.95|0.59|
| DeriveRecoverPlanIsTotalOverTheEvidenceDomain |24.92|0.93|0.73|
| RecoverReconcileReadingIsMetamorphic |24.67|0.30|0.21|
| FleetInventoryLiveConformance |24.61|4.08|5.16|
| ObserveEachBrokenResource |24.29|sum children18.39; window84.80|sum children22.24; window108.03|
| RepositoryNamesAtConversationAllocationBoundaries |23.96|sum children10.19; window138.70|sum children16.49; window172.70|
| SetAsideHoldsMatchThePlan |22.97|0.44|1.10|

Parallel child durations overlap: their sum is workload attribution, not elapsed wall savings. Go reports0.00s for these three parents after child parallelization, excluding their children. The inclusive parent window above adds run-to-pause and cont-to-terminal intervals; it includes child scheduling waits behind unrelated tests. Neither zero parent elapsed nor the larger scheduling window is an honest standalone execution comparison. Before/after samples also have different host contention, so make no qualified speedup/reliability claim.

Correctness evidence: combined changed properties/fixtures/tables race PASS20.379s; corrected separate-storage/identity race group PASS3.160s; existing durable/CAS/transaction group PASS and lifecycle/continuation race PASS10.561s. Domain drop/duplicate/reorder controls fail the ordered fingerprint. Durable outsider/full-scan/uncoded-refusal/partial-tombstone/unsafe-clear controls fail their actual assertions. Set-aside lost-move control fails its sentinel assertion; ignored-hold control fails on nil-error classification (never passes). One-process identity contract now includes separate Git storage, with an observed red before correction. No production files changed.

### Residual capacity proposal, not implementation

The diagnostic remains above200s even before the competing full suite, and the corrected loaded run passes at340s. A corrected quiet green measurement is still missing. Much of the remaining serial phase is distributed across small global-state/real-process tests: first diagnostic's largest sequential cases are InterruptedSwitchResumePreservesTargetAuthority9.41s, AcceptanceReportBudgetTenSlots9.07s, RecoverResumeRefusesConversationLiveInAnotherSlotAtBothGates6.73s; there is no single near-timeout case.

Using sorted round-robin top-level names, charging the three parallel-child tables by their child sums rather than zero parent elapsed, the diagnostic predicts three lanes of189.31/190.03/188.73s (461/460/460names). Applying the same model to the corrected loaded sample predicts228.98/231.42/236.78s. Model = measured sequential sum + parallel elapsed sum/3. It omits changed CPU/IO contention, compile/discovery and tail scheduling; these are proposals, not verified lane times. Retain #610 until the hub approves a residual plan and real qualification passes.


## Recorded integrated R2 suite (R5 landing acceptance)

PR371 head303e77d6d50f79d014a56c46ab9e5921a6e556cb: `sdlc local-check run --name suite` ran `make PAIR_GO_TRIMPATH=-trimpath test`, PASS exit0 in11m45s; couchcore341.494s and all prerequisites/packages passed. Server checks also pass. Log prefixSHA256d436b8aaf565; recorded as sdlc-local/suite, with log in the common Git directory under sdlc/local-checks/<head>/suite.log. Full suite wall is below the900.89s instrumented before sample, but differences in head/load/instrumentation prevent an isolated speedup claim. Integration includes #595's independent removal of two obsolete core top-level tests and one obsolete subcase; R2 removes no test cases. The allowance remains per hub's R5 decision. A quiet control on landed main, with every machine-suite lock checked free, remains pair:1's next obligation after landing; lane work needs its own item/review.
