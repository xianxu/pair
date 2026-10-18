---
collaboration-mode: discussion
---

# Pair suite disk writes implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** Reduce couchcmd own-process physical writes by at least50% without changing its test scenarios, and defend the measured large-writer packages with counted byte budgets.

**Architecture:** Keep each ledger's exact contents, logical size and private inode, but initialize multi-owner fixtures from one private all-space copy-on-write template. A shared test-only runner measures cumulative physical bytes around each selected package's existing TestMain callback, separately from portable content/isolation and operation-count checks.

**Tech Stack:** Go testing; existing golang.org/x/sys/unix Clonefile; existing procutil.NativeResourceReader on Darwin+cgo. Original streaming fallback on unsupported filesystems/platforms.

**Approval status:** Draft measurement/design; whole-Go-phase baseline is queued. Do not enter change-code until the baseline is recorded, exact package budgets are set and the hub approves.

## Measurement and operating envelope

Baseline/source analysis and the owned clone feasibility experiment are recorded in issue658 Log. Clean couchcmd:627,539,968 own-process physical bytes,600,489,984 in TestWorkbenchPerformanceMessageUnchangedEvidence. The clone experiment wrote30,085,120 bytes while retaining600,000,000 logical bytes across20 private files. Those are separate experiments, not a package before/after comparison.

The baseline/after comparison runs the same ordinary Go test packages, with a fresh supplied Pair executable and sanitized session environment. Compile/build bytes and child-process writes are excluded and labeled. Serial verbose sampling attributes approximate bytes to tests; writable-FD path sampling identifies likely writers but is not a syscall trace. Record every package, not only the five known fixture consumers. If the baseline identifies another writer large enough to prevent the goal, revise this plan before coding.

Workload: batch test setup; owner count N, ledger logical size L, distinct required record sizes S. Expected streaming work for cloned multi-owner ledgers is O(S*L + total record bytes), with one clone and one tail patch per owner. For the current fixed-size records, S=1. No latency ceiling or CPU timing assertion. Memory stays64KiB streaming buffer plus existing small record strings; no600MB shared buffer. No new concurrency. Each test owns its template/ledgers under t.TempDir; cleanup removes them on success/failure. Single-owner fixtures retain direct streaming (no saving from cloning one owner).

Physical budget envelope is Darwin+cgo on the clone-capable temp volume measured here; clone failure retains correctness through explicit streaming fallback, but cannot claim the APFS write reduction. Unsupported measurement is reported distinctly, never as zero. The ordinary package runner enforces its cap by default on Darwin+cgo. On other platforms it prints an explicit unsupported diagnostic and runs the portable correctness checks without claiming a measured budget. No silent read-error skip is permitted on Darwin+cgo. A non-clone-capable Darwin filesystem still gets correct streamed fixtures, but a whole-package cap can fail honestly because the write-saving operating envelope is unavailable; targeted portable fixture tests remain runnable. No new environment knob or Makefile/workflow edit is required. Numeric caps are finalized with the baseline before approval.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| WriteBudgetVerdict | cmd/internal/testdisk/budget.go | new |

WriteBudgetVerdict compares before/after cumulative uint64 bytes with a positive per-run cap and explicit run count. It rejects a backwards counter, arithmetic overflow or excess, retaining an existing nonzero test result. Tests directly cover below/equal/above cap, counter regression, count scaling and overflow. One verdict owner avoids five subtly different comparisons (ARCH-PURE, ARCH-DRY).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Fixture ledger writer | cmd/internal/workbenchperftest/fixture.go; ledger_clone_darwin.go; ledger_clone_other.go | modified | private regular files and native CoW cloning |
| RunWithWriteBudget | cmd/internal/testdisk/run.go | new | procutil.NativeResourceReader and package normal-run callback |
| Package budget callers | couchcmd/testmain_test.go; couchtty/reattach_testmain_test.go; titlepoller/runcli_process_test.go; threadactivity/testmain_test.go; workbenchperftest/testmain_test.go | modified/new | existing package TestMain lifecycle |

File paths in the last row are below cmd/internal. The helper accepts func()int, preserving couchcmd's testpair.Run(m) cleanup, couchtty's LifecycleParallelism setup, and titlepoller's explicit child-role dispatch. Only titlepoller's normal mode is budgeted. No change to testpair's global executable lifecycle.

The fixture writer owns a template per actual size max(requested,len(records)); no global cache or shared mutable inode. The first owner's records must never become padding for another owner. Templates contain ASCII spaces, not sparse NUL bytes. Darwin uses existing unix.Clonefile; other platforms use the original writer. Classify known unsupported clone errors for fallback; real permissions/storage errors fail. A stateful in-memory file backend or owned temp files behind the same narrow writer seam exercise clone/fallback, partial failure and patch isolation. Native conformance checks exact bytes, metadata/inode separation and modifications to one owner against siblings/template. No external service is added (ARCH-MOCK).

The budget reader locates the exact current process once through Table and reuses its identity-fenced Usage before/after the callback. Supported-platform observation failures and missing self never turn into a passing zero. A stateful fake advances cumulative counters through run, failure and regression. A real small flushed file calibrates physical counter changes without a speed assertion. No new native ABI binding duplicates procutil (ARCH-DRY).

## State and failures

Actions available to automation: run the package; run portable correctness checks; rerun an interrupted exact head; inspect failure and change code or explicitly revise a cap with evidence.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Counter available; tests and budget pass | counted bytes and limit; success | run next package | none |
| Tests return nonzero | original test failure plus measured bytes | inspect/fix then run package | failed attempt retained |
| Counter fails/regresses or cap exceeded | explicit measurement/budget failure | inspect/fix or evidence-backed cap revision | none; no success claim |
| Counter unsupported on non-Darwin/cgo build | explicit unsupported, no byte claim | run portable correctness checks | no physical evidence on that platform |
| Native clone unsupported | original streaming semantics; physical budget may fail | run targeted correctness or use clone-capable temp volume for the package budget | write saving unavailable |
| Process killed/panics during test | unsuccessful suite, no completed measurement | rerun interrupted exact head | incomplete measurement; owned temp cleanup follows Go/test owner |

No durable application state or background worker is introduced. Template ownership lasts only the fixture scope; completed samples last only the test process. Test process death cannot mutate operator paths. No retries of a half-patched ledger; report the setup error and let the owner clean its temp tree (ARCH-STATE, ARCH-FUNERAL, ARCH-SECURE).

## Alternatives and architectural decisions

- Smaller fixtures or in-memory ledger fakes would lose real600MB cold read/decode coverage. Hardlinks would lose mutation independence; sparse holes are invalid JSON whitespace. Reject all three.
- A persistent shared fixture cache would add cross-test invalidation, ownership and cleanup state. Keep templates fixture-local.
- An external go-test wrapper cannot reliably sample a short-lived process's final cumulative counter without cooperation. TestMain is the precise existing completion boundary; no sampling race in the regression check.
- Protect the shared large-fixture consumer class and record every package's measured distribution; do not rewrite unrelated PTY stress tests merely to lower a number (ARCH-PURPOSE).
- Supported temp filesystem and own-process accounting are explicit constraints, not universal SSD wear claims (ARCH-CONSTRAINTS).

## Tasks (one close boundary)

### 1. Finish and publish baseline/spec

- [ ] Record whole-Go-phase package bytes, pass/failure names, and top test/path attribution in a compact checked-in measurement artifact. Preserve raw local diagnostics and identify their limitations.
- [ ] Finalize the budget table from the measured distribution and the proved clone saving. The couchcmd cap must be below313,769,984 bytes (half of the clean baseline), with bounded headroom for retained stress tests and new regression cases. Name every selected package and exact bytes.
- [ ] Post measurement, proposed mechanism, unsupported-platform behavior, package scope and no-real-environment-change live arrangement to the hub. Commit design and wait for approval. Run sdlc change-code and satisfy its review/estimate gate before source edits.

### 2. Defend the byte/content contract and implement the shared writer

Files: cmd/internal/workbenchperftest/fixture.go, fixture_test.go, ledger_clone_darwin.go, ledger_clone_other.go; new focused ledger_writer_test.go if needed.

- [ ] Add failing tests for identical expected bytes including all padding; independent inode and tail mutation; unchanged required-record minimum size; owner record-length variations; clone unsupported fallback; fatal clone/storage failures; partial patch failure cleanup. Keep current three30MB decode test intact.
- [ ] Add a counted N=8 and2N=16 test at a modest fixed ledger size: one seed per size, exactly N clones/patches, streaming bytes bounded by one seed plus required records. Doubling owner count must not double large padding writes. Assert the same backend path NewFixture actually uses, not an unused model.
- [ ] Run go test ./cmd/internal/workbenchperftest -run 'Test(Fixture|Ledger)' -count=1; demonstrate the new counted regression fails with the original per-owner streaming construction.
- [ ] Implement fixture-local all-space templates, private native clone plus tail WriteAt, original streaming fallback, scoped close/error handling and t.TempDir cleanup. No changed logical sizes or consumer assertions.
- [ ] Run the focused tests, then all five fixture-consumer packages with their existing read/decode/authority tests. Check no new hidden large fixture cost in the regression itself.
- [ ] Commit the atomic writer and its tests with Agent/Co-Authored-By trailers.

### 3. Enforce physical package budgets

Files: new cmd/internal/testdisk/budget.go, budget_test.go, run.go, run_test.go; five TestMain callers above; no Makefile or workflow configuration change.

- [ ] Write failing pure tests for budget comparisons/count scaling/overflow and stateful-reader tests for supported, unsupported, failed before/after, missing identity and decreasing counters. Preserve nonzero callback exit and call it exactly once under the documented policy.
- [ ] Implement the narrow callback runner using existing procutil, with explicit own-process bytes/cap diagnostic. Handle -count=N without silently reusing a one-run allowance; when -test.bench is nonempty, report that the ordinary-suite cap does not cover the benchmark workload, and retain counted fixture-growth tests.
- [ ] Wire normal TestMain paths for the measured package set without affecting helper dispatch/cleanup/lifecycle setup. Run the pre-fix byte-producing control or synthetic over-budget child and confirm nonzero status. A deleted/no-op budget must make this negative control fail; avoid writing the full large baseline solely to prove refusal.
- [ ] Run targeted testdisk tests and the five package suites. Record exact native bytes/caps, not elapsed time, as the regression evidence.
- [ ] Commit the enforcement and tests.

### 4. Compare, document, and land

- [ ] Repeat identical per-package measurement against the implemented head; retain every failed attempt and classify it under the flake rule. Compare all packages, exact target reduction and unchanged cold-read/decode assertions.
- [ ] Update atlas/index.md and a focused testing document with the byte-budget scope, supported counter/filesystem, invocation, unsupported behavior, repeated-count/benchmark policy, and how an intentional cap revision is justified. Package caps are ordinary-suite limits, not benchmark limits; native unsupported results are explicit and are never reported as zero.
- [ ] Open an early draft via sdlc pr for CI overlap if useful. Run the recorded suite on pushed head with sdlc local-check run --name suite, then sdlc close with measured evidence. The binary owns the fresh-context review; no duplicate boundary review.
- [ ] Resolve findings; sdlc merge --yes supplies integration validation. Report verified PR/merge to the hub. No pair:0 staging or operator restart is needed for test-only owned-file changes; ask if the eventual implementation changes that surface.
