# Couchcore test execution budget implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve every default-suite test while keeping each couchcore test process safely below Go's default ten-minute timeout, then remove #610's allowance.

**Architecture:** Keep the Go package and its private fixtures intact. A small Python runner uses ordinary Go discovery and `go test -json -run` to run three disjoint name partitions with an aggregate parallel budget. It also runs the remaining packages exactly once. Go owns compilation, TestMain, helper re-exec, seed execution, JSON conversion and timeout behavior.

**Tech stack:** Go test/list/JSON, Python standard library process orchestration and unittest, existing Makefile suite lock and environment sanitizer.

## Evidence and alternatives

Read `000611-couchcore-test-execution-budget-profile.md` first. The isolated baseline is 326.006s with a 143.744s serial phase; the longest individual test is 46.26s. The full suite baseline with JSON events is running before implementation.

1. **Recommend three process shards of the same package.** Preserve private coupling, test cwd and all self-exec helpers. Sorted round-robin requires no timing cache or handwritten test list. Its predicted worst lane is 262.78s under the measured nine-worker envelope; qualify with real runs.
2. A directory/package split would move heavily coupled fixtures (270 same-package test files, newTestEnv used across 68 files) or expose private production state. This would add unrelated API/ownership changes merely to obtain separate processes.
3. Generalizing ariadne's existing cmd/sdlc sharder requires a separate upstream change. Copying it locally would duplicate compilation/test2json/reporting policy. Reuse ordinary Go commands instead and own only Pair's small partition/coverage/process-lifetime adapter (ARCH-DRY).

No individual test body, assertion, fixture timeout, parallel marker or live-conformance gate changes. No retries, new skips, sleeps or larger deadlines.

## Core concepts

No production domain model changes.

| Pure entity | Kind | Lives in | Status |
|---|---|---|---|
| partition_names | PURE | scripts/test-couchcore.py | new |
| coverage_errors | PURE | scripts/test-couchcore.py | new |
| parallel_budget | PURE | scripts/test-couchcore.py | new |

- `partition_names`: one discovered name belongs to exactly one sorted round-robin partition. No persisted schedule or timing cache; newly listed names join automatically. Core rule: `parts = [sorted(names)[i::count] for i in range(count)]`, discarding empty parts only.
- `coverage_errors`: compare a Counter of terminal top-level events against every discovered Test/Fuzz/Example name; duplicates and omissions fail, including skip dispositions. Subtest events do not count as top-level coverage.
- `parallel_budget`: preserve the caller's positive GOMAXPROCS limit, with a conservative default ceiling of nine aggregate test workers when absent. Effective aggregate budget is at most nine and at most 3×the known CPU limit; divide by active shard count, never below one. Reduce shard count if necessary to avoid exceeding the budget. Explicit qualification options may reduce shard count/budget; they cannot silently enlarge the default envelope.

| Integration point | Kind | Lives in | Status | Wraps |
|---|---|---|---|---|
| Go suite runner | INTEGRATION | scripts/test-couchcore.py | new | go list/test, subprocess groups, temporary logs |
| default suite recipe | INTEGRATION | Makefile.local | modified | existing test prerequisites, machine suite lock, Go runner |
| runner conformance fixture | INTEGRATION | scripts/test-couchcore.test.py | new | disposable stateful fake Go executable + real child process trees |

The runner obtains package paths from `go list ./...`, discovers names with `go test -list .` and excludes couchcore only from the remaining-package command. Invoke each shard with `go test -json -count=1 -run=^(NAME|...)$ -parallel=N` on the unchanged package, retaining current caller GOFLAGS and GOMAXPROCS. Never pass a larger timeout: ordinary Go defaults apply. Anchor and validate discovered names before composing the expression. Tests and benchmarks are distinguished by Go's own listing; retain all listed Test/Fuzz/Example cases.

## Operating envelope and lifecycle

- **Requirement proposed to hub:** default three shards, current aggregate parallel budget nine; other-package build/run concurrency continues to honor GOFLAGS=-p=3. Machine suite weight stays one because test parallelism is bounded at the measured baseline, not multiplied per shard (ARCH-CONSTRAINTS).
- **Qualification proposal:** every shard below 300s in package-isolated qualification and below 420s under representative full-suite/co-tenant load, leaving at least 30% headroom to 600s. Full-suite wall must not materially regress against the same-envelope baseline. These are proposed acceptance budgets, subject to hub approval; do not turn estimates into claims.
- The real baseline was package-isolated, not a certified idle machine. Record host load and competing suites for each measurement. Do not change machine settings or kill unrelated work to manufacture quietness.
- Normal tests keep independent temporary roots. Shared process environment/cwd changes remain isolated per process; no new t.Parallel calls. TestMain helper dispatch and aliased executables remain untouched (ARCH-MOCK, ARCH-ORDER).
- Existing `make test-live` and focused native conformance remain their existing serial routes. The default runner must refuse explicit live-opt-in settings before starting parallel shards, with an actionable direct/live-target route, rather than run global-resource conformance concurrently. Enumerate actual gates while implementing; include every currently supported opt-in and test refusal. This deliberate default-suite boundary is part of the design for hub review.
- Runner states: discovery → partition → running → collecting → done. Discovery/build failure starts no shard. A test failure allows already selected jobs to finish, reports all failures and returns nonzero. Cancellation stops new admission, signals every owned process group, waits a bounded grace interval, kills unreaped groups and joins before returning; cancelled work never reports success. No asynchronous worker survives runner return (ARCH-ORDER).
- Temporary per-job logs belong to a TemporaryDirectory with an explicit nonempty TMPDIR path fallback; remove on every exit. Optional requested JSON output belongs to its caller (the SDLC suite record or profiling invocation). No new persistent cache, credentials or user-state writes (ARCH-FUNERAL, ARCH-SECURE).

## Implementation tasks (one atomic close boundary)

### Task 1: Finish measured baseline and approve design

Files: issue, this plan, profile artifact.

- [ ] Finish the real Make baseline; extract couchcore package/per-test timestamps, queue delay and full wall separately.
- [ ] Reconcile the proposed shard count/budget against both profiles; post the measured design to hub and wait for approval before any split.
- [ ] Commit the approved design. Run `sdlc change-code --issue 611 --flow full`; answer plan-quality review, then derive the required estimate from the approved scope.

### Task 2: Prove partition, coverage and process ownership before integration

Create `scripts/test-couchcore.py`, `scripts/test-couchcore.test.py`.

- [ ] Write failing tests for deterministic disjoint/exhaustive partitioning, shuffled discovery, added/removed names, empty discovery, Test/Fuzz/Example inclusion, duplicate/omitted terminal events, subtests, unchanged skip accounting and caller concurrency limits. Run the focused Python test command and observe the intended failures.
- [ ] Implement only those pure functions. Unit-test without IO. The expected Counter for coverage is `{name: 1 for name in discovered}`; reject every missing, extra or non-unit count.
- [ ] Add a stateful fake-Go fixture that lists packages/names, interprets anchored selection, emits Go-shaped JSON and can fail, omit, duplicate or hang a selected case. Assert remaining packages execute exactly once and never include couchcore; all shard selections use the normal package path and explicit bounded parallel value.
- [ ] Make discovery failure, malformed output, subprocess exit, failed subtest and timeout produce a nonzero result with retained actionable output. A known failure must not suppress other selected cases.
- [ ] Add cancellation/child-process-tree tests with an explicit readiness handshake. Observe red before adding process-group shutdown and joined cleanup. Assert no owned child remains, no new jobs start after cancellation and a cancelled run cannot pass.
- [ ] Implement the thin runner around ordinary subprocess Go commands. Keep stdout JSON separate from stderr diagnostics; report per-job execution elapsed independently of compile/discovery overhead. The optional merged JSON adds shard identity without conflating overlapping package events.
- [ ] Run all runner tests; use a real focused couchcore selection to verify helper re-exec and Go-shaped output before the full run. Commit.

### Task 3: Wire default suite and qualify preserved inventory

Modify `Makefile.local`, relevant workflow-contract tests only if their existing contract changes, `atlas/` and index.

- [ ] Add a targeted `test-couchcore` route using the same runner without other packages. Integrate the full runner at the existing final Go recipe; keep every prior prerequisite, environment sanitizer and machine lock. Add runner self-tests to the normal suite. Remove the #610 comment and explicit 20m timeout.
- [ ] Run Make workflow/environment contracts and negative tests for missing/duplicate coverage, omitted package, failing subtest and cancellation. Confirm default suite no longer supplies the temporary allowance.
- [ ] Run isolated sharded JSON measurement. Compare discovered names and terminal Counter against baseline's 1,376 cases; reconcile only actual platform/head differences, never waive a missing case. Require each shard's measured time below the approved isolated budget.
- [ ] Push PR with `sdlc pr`; run `sdlc local-check run --name suite` on the pushed head. Capture before/after full wall, queue, each shard and unchanged other-package times. Qualify representative load within the machine suite gate; do not launch artificial unbudgeted competing work. If representative co-tenancy is unavailable, report that gap to hub rather than claiming it.
- [ ] Run race verification through the same selection/inventory path with the caller race flags, retaining bounded concurrency. Verify each discovered test still executes once; preserve opt-in serial live routes separately.
- [ ] If any approved budget fails, stop and re-plan with the measured distribution; do not extend timeout, skip tests or raise fan-out opportunistically.
- [ ] Update atlas test-run guidance and index with commands, concurrency, accounting and budgets. Update issue profile/Log and lessons with observed results. Commit.

### Task 4: Close and land

- [ ] Reassess Done when, run `sdlc close --issue 611` with exact test and profile evidence (TL owns project record; use --no-project with rationale). The binary owns the fresh review.
- [ ] Address findings by class; run required verification for any changes. Report SHIP/evidence to hub and land through the line when authorized. Do not apply local-gates verification labels.

## Architecture checklist

ARCH-DRY: use Go's runner/build/JSON rather than clone the upstream specialized engine. ARCH-PURE: partition, coverage and budget rules separate from process IO. ARCH-PURPOSE: preserve the whole discovered inventory and every other package, not only fast fixtures. ARCH-MOCK: stateful fake-Go orchestration plus real Go/helper conformance. ARCH-CONSTRAINTS: measured aggregate budget and per-process headroom. ARCH-SECURE: only typed discovered names/argv; isolated temporary roots, no session identity or secret mutation. ARCH-ORDER: explicit job/cancellation lifecycle and joined workers. ARCH-FUNERAL: bounded temporary logs; no timing-cache family.


## Revisions

### 2026-10-16 — R2: cheaper tests before capacity (active plan)

**Reason:** TL receipt f85a0b66-f9d0-47d3-b5f0-51c3a2f54288 rejects the sharding-first proposal. The preceding runner design/tasks are retained as an unapproved historical proposal and are **not execution instructions**. This revision supersedes every earlier implementation task, proposed budget and live-opt-in refusal. No runner, new skip, changed fixture deadline or capacity increase is authorized. All ten causes are in the profile artifact.

**Active goal:** remove avoidable computation and setup/transaction/process work, preserve each test's claims, remeasure with the same GOMAXPROCS=3/GOFLAGS=-p=3 envelope. Only if the resulting isolated package still exceeds about200s will we propose three lanes with measured results. Remove #610's20m allowance once default600s has comfortable measured headroom.

**Core concepts delta:** retire partition_names, coverage_errors, parallel_budget and the custom runner entirely. New/modified test-only PURE entities: factored consistency predicates and ordered evidence iterator in recoverplan_test.go; bounded dependency-base enumeration. INTEGRATION entities: ProvisionFixture identity adapter in provision_git_test.go; existing lifecycle/continuation fixtures; disposable fleet conformance in recoverplan_fleet_test.go. All production algorithms and interfaces remain unchanged.

#### R2 task1 — preserve coverage and eliminate Cartesian waste

Files: recoverplan_test.go, optional focused test-only evidence enumeration file.

- [ ] Record the old iterator's accepted-order/count fingerprint and metamorphic stride fingerprint, plus host-base/variant counts. Retain All* vocabulary-derived dimensions. Regression/negative controls must detect a dropped, duplicated or reordered accepted point; do not retain a second expensive exhaustive reference pass in every normal run.
- [ ] Factor existing consistent into agent/offer physical facts, host Git/claim facts and dependency facts, with consistent delegating to those same predicates. Test existing impossible-evidence exclusions and boundary combinations before changing enumeration.
- [ ] Run agent/offer checks before entering branch/Git/claim dimensions, then complete host checks before dependency loops. Preserve lexical accepted order; do not duplicate pruning rules or invent narrower legal states. Keep the final consistency authority.
- [ ] Parameterize dependency coordinates so HostJudgments generates only DepNone/Clean bases initially, then checks every current dependency variant. Require old/new counts and fingerprints to match; keep stride23 unchanged in the metamorphic property.
- [ ] Verify original assertions and mutation controls; profile the three properties again and record CPU/time change. Commit.

#### R2 task2 — remove redundant fixture writes and identity subprocesses

Files: provision_git_test.go, sessionevidence_test.go, continuation_store_test.go, existing fixture conformance tests as needed.

- [ ] Add real-Git identity contract checks for primary/slot, branch change, detached HEAD, unsafe/corrupt membership and cancelled context before altering the fixture adapter. Observe the meaningful failure for any new semantic guard, then batch top-level/common-dir/HEAD/branch through one real Git invocation. Preserve fresh results and validate output shape; no observation cache and no production changes. Compare full identities before/after and count actual subprocesses in a focused diagnostic trace.
- [ ] In ReAdoption, build the same110 candidate shapes before allocating a store; use ValidateThreadRecord to avoid IO for exactly the shapes rejected today. Seed the same Name in the initial record, removing the extra metadata transition. Preserve per-cell names/dispositions, pure validator ownership, all admitted durable write/clear/readback assertions and coded-refusal/no-partial-tombstone negative controls.
- [ ] In ContinuationRequestsBoundedOwnedSet, leave all100 rows and corruption in place; publish only for the ten requested rows. Keep duplicate ownership, ten-output/4096-byte and redaction contracts. Force an outsider read/full scan to show the regression still fails. Run existing continuation CAS/transaction tests unchanged.
- [ ] Commit measured cost reductions and exact retained coverage counts.

#### R2 task3 — bound external conformance input and remove serial table bottlenecks

Files: recoverplan_fleet_test.go, slotreconcile_test.go, slotobserve_test.go, identity_repository_names_test.go, slotsave_test.go.

- [ ] Change live producer conformance's vantage from the operator fleet to a disposable real-Git fleet with nonempty ready/dirty/missing observations and local remotes. Keep the installed real sdlc CLI, schema/decoder/known-vocabulary assertions and existing missing-binary/short dispositions; introduce no opt-in gate or new skips. Demonstrate unsupported producer output still fails. A fake fleet alone cannot replace this proof.
- [ ] Replace SetAsideHoldsMatchThePlan's nine fully provisioned worlds with minimal independent real temp layouts: retain all eight agent states, actual set-aside/classification/no-residue behavior, a sentinel movement assertion for permitted states, and one minimal healthy real-Git checkout for recovered refusal. Existing full-checkout restoration/conformance tests stay unchanged. This removes unnecessary Git setup rather than parallelizing it.
- [ ] Add t.Parallel only to independent child cases of real-Git reconciliation, resource observation and repository-name tables. Each retains its own temp roots/refs/stores; no environment/cwd/global mutations. Parent and child cleanup ownership must remain intact. Existing TestMain's aggregate parallel limit is unchanged; verify race safety and all case results. Do not add parallelism to global-mutation tests or remove cases.
- [ ] Retain all12 real-Git scenarios, both reconciliation passes/model agreement/no-repeat effects, all ten broken-resource scenarios and all five name cases and all eight set-aside agent variants plus recovered checkout. Run focused race/negative controls. Commit.

#### R2 task4 — remeasure, decide, restore default timeout

Files: profile artifact, issue, Makefile.local, existing test documentation if needed.

- [ ] Finish and record current pre-change full-Make JSON baseline, including queue wait, package wall, serial/parallel intervals, inventory and co-tenancy (targeted diagnostics ran during its Go phase; label that load honestly).
- [ ] Repeat isolated full couchcore JSON with unchanged caller flags; compare every top-level/subtest disposition except explicitly added regression cases. Run focused race checks for changed fixtures. Record timings and command counts separately; never claim summed overlapping test durations as wall savings.
- [ ] If >about200s remains, stop and send a measured residual profile plus a three-lane proposal to hub. No runner work until new approval. If comfortably under, qualify representative full-suite load and remove the20m flag/comment from Makefile.local, retaining the default10m. Do not extend deadlines to make qualification pass.
- [ ] Push, record full suite with sdlc local-check run --name suite, complete required Make/environment contracts and CI. Record all failures, including unrelated ones. Update profile/Log/lessons; TL owns project records.
- [ ] Run the full-flow close review and land through the line on SHIP. Keep one atomic close boundary; no Mx tags.

**Approval checkpoint:** send this R2 plan and per-test causes to the hub, then wait before change-code/implementation. Full flow remains explicit. Estimate is derived only after the revised design clears plan-quality.

**Architecture:** ARCH-DRY shares physical predicates and existing fixtures, with no new scheduler. ARCH-PURE removes IO from rejected-shape setup and uses pure domain pruning. ARCH-PURPOSE retains all domain points, cases and write-path assertions. ARCH-MOCK keeps real Git/CLI conformance where that is the claim. ARCH-CONSTRAINTS keeps current concurrency and measures actual savings. ARCH-SECURE moves conformance from operator data into a controlled temporary fleet. ARCH-ORDER preserves accepted enumeration order and per-case ownership. ARCH-FUNERAL uses existing t.TempDir cleanup, with no new persistent cache or fixture template.


### 2026-10-16 — R3: measurable qualification and named test strategies

**Reason:** plan-quality PQ-1 requires numerical acceptance and a repeat rule; PQ-2 requests named functions instead of an identity-case inventory. This is a qualification clarification within approved R2; no scope, concurrency or deadline increase.

**Active acceptance:** same flags/head, two consecutive isolated couchcore runs at or below200s, plus one recorded real full suite whose couchcore execution is at or below420s (at least30% headroom to default600s), preserving inventory/dispositions and all assertions. The full suite's normal other-package workload is the representative load; record host/co-tenant activity honestly. Full-suite wall may not regress against the900.89s before sample. The approximately200s threshold separately triggers the hub's lane discussion; passing it once does not itself authorize removing the allowance. If either repeat or representative-load qualification is unavailable/fails, retain #610's allowance, report the actual result/gap to hub and re-plan; do not claim reliability, weaken checks or extend deadlines. Do not rerun the full suite gratuitously: the single required recorded suite supplies its loaded sample, with isolated repeat checks addressing measurement reliability.

**Named pure entities/strategies:**

- forEachEvidence / forEachEvidenceWithDeps: retain vocabulary-derived values; compare accepted ordered count/fingerprint and stride fingerprint against the pre-change stream, with dropped/duplicate/reordered negative controls.
- consistent / consistentActorEvidence / consistentHostEvidence / consistentDependencyEvidence: share the existing physical constraints once; impossible/unknown evidence classes remain rejected and the existing invariant tests retain their independent expectations.
- ProvisionFixture.Run: real Git identity must remain fresh and exact across mutable/unread state; compare full records through real Git and use a bounded command-count trace to verify the batched query, retaining cancellation/error propagation.

The R2 task2 identity-case enumeration is superseded by this function-level strategy. All risky source functions remain covered; the concrete executable cases are authored with the tests. The issue's original split/shard Done-when criterion is superseded by approved R2/R3: cheaper execution with preserved inventory is sufficient when these budgets pass; a split is conditional on the post-fix residual measurement and fresh hub approval.
