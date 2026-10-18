---
collaboration-mode: discussion
---

# Pair suite disk writes implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox syntax for tracking.

**Goal:** Reduce couchcmd own-process physical writes by at least50% without changing its test scenarios, and defend the measured large-writer packages with counted byte budgets.

**Architecture:** Keep each ledger's exact contents, logical size and private inode, but initialize multi-owner fixtures from one private all-space copy-on-write template. A shared test-only runner measures cumulative physical bytes around each selected package's existing TestMain callback, separately from portable content/isolation and operation-count checks.

**Tech Stack:** Go testing; existing golang.org/x/sys/unix Clonefile; existing procutil.NativeResourceReader on Darwin+cgo. Original streaming fallback on unsupported filesystems/platforms.

**Approval status:** Baseline completed:116/116 packages passed,1,640,353,792 observed own-process physical bytes. Exact proposed caps are below. Await hub approval before change-code.

**Current design:** The earlier ledger-only candidate below is historical and suspended. The authoritative proposed replacement is [the whole-suite revision](#2026-10-17--whole-suite-replacement-design); hub approved it in receipt b835c6cc-0386-4592-8787-babadea817ba; change-code accepted it in d025e19ee.

## Measurement and operating envelope

Baseline/source analysis and the owned clone feasibility experiment are recorded in issue658 Log. Clean couchcmd:627,539,968 own-process physical bytes,600,489,984 in TestWorkbenchPerformanceMessageUnchangedEvidence. The clone experiment wrote30,085,120 bytes while retaining600,000,000 logical bytes across20 private files. Those are separate experiments, not a package before/after comparison.

The baseline/after comparison runs the same ordinary Go test packages, with a fresh supplied Pair executable and sanitized session environment. Compile/build bytes and child-process writes are excluded and labeled. Serial verbose sampling attributes approximate bytes to tests; writable-FD path sampling identifies likely writers but is not a syscall trace. Record every package, not only the five known fixture consumers and any other measured package at or above30,000,000 bytes. The seven selected packages account for1,533,018,112 observed bytes,93.46% of the baseline. Other packages remain in the before/after table; their existing stress tests are retained.

Workload: batch test setup; owner count N, ledger logical size L, distinct required record sizes S. Expected streaming work for cloned multi-owner ledgers is O(S*L + total record bytes), with one clone and one tail patch per owner. For the current fixed-size records, S=1. No latency ceiling or CPU timing assertion. Memory stays64KiB streaming buffer plus existing small record strings; no600MB shared buffer. No new concurrency. Each test owns its template/ledgers under t.TempDir; cleanup removes them on success/failure. Single-owner fixtures retain direct streaming (no saving from cloning one owner).

Physical budget envelope is Darwin+cgo on the clone-capable temp volume measured here; clone failure retains correctness through explicit streaming fallback, but cannot claim the APFS write reduction. Unsupported measurement is reported distinctly, never as zero. The ordinary package runner enforces its cap by default on Darwin+cgo. On other platforms it prints an explicit unsupported diagnostic and runs the portable correctness checks without claiming a measured budget. No silent read-error skip is permitted on Darwin+cgo. A non-clone-capable Darwin filesystem still gets correct streamed fixtures, but a whole-package cap can fail honestly because the write-saving operating envelope is unavailable; targeted portable fixture tests remain runnable. No new environment knob or Makefile/workflow edit is required. Numeric caps are specified below.

## Proposed package caps

Caps cover one ordinary test run, with explicit checked -count=N scaling. They are measured bytes, independent of elapsed time. For couchcmd/couchtty/fixture, headroom is calculated against a30MB template plus retained writes/new regression cases; the original per-owner streaming scheme exceeds the cap. For unchanged packages, bounded headroom above the baseline preserves their existing stress scenarios. Native filesystem allocation granularity and concurrent atomic publication can change physical counts, so these are meaningful ceilings rather than exact-counter equality assertions. The after-measurement must demonstrate the caps under the normal suite parallelism as well as the matched serial diagnostic.

| Package | Baseline bytes | Proposed cap bytes | Basis |
|---|---:|---:|---|
| couchcmd |625,799,168|100,663,296 (96MiB)|~30MB seed plus25MB retained work, new regressions and allocation headroom; below half baseline|
| couchtty |609,632,256|67,108,864 (64MiB)|~30MB seed plus9MB retained work and allocation headroom|
| workbenchperftest |90,030,080|67,108,864 (64MiB)|~30MB seed plus bounded content/isolation/fallback/N-2N regressions|
| threadactivity |30,089,216|50,331,648 (48MiB)|single-owner30MB control retained, plus bounded headroom|
| titlepoller |10,592,256|25,165,824 (24MiB)|10MiB logical-size control retained, plus bounded headroom|
| sessionsync |71,299,072|100,663,296 (96MiB)|publication/bounds/trim workloads retained; budget only|
| couchcore |95,576,064|134,217,728 (128MiB)|journal/resource-history/publication limits retained; budget only|

The optimization removes1,200,000,000 repeated padding bytes across the two20-owner and one3-owner30MB fixtures, before tail-page overhead; this predicts a large reduction but the real after-measurement is the acceptance evidence. The five fixture consumers keep all old file sizes and cold-read/decode assertions. Seven packages are guarded: every baseline package at or above30,000,000 own-process bytes, plus all shared-fixture consumers. Smaller independent packages have measured baselines, not a promised package cap in this issue.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| WriteBudgetVerdict | cmd/internal/testdisk/budget.go | new |

WriteBudgetVerdict compares before/after cumulative uint64 bytes with a positive per-run cap and explicit run count. It rejects a backwards counter, arithmetic overflow or excess, retaining an existing nonzero test result. Tests directly cover below/equal/above cap, counter regression, count scaling and overflow. One verdict owner avoids different per-package comparisons (ARCH-PURE, ARCH-DRY).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Fixture ledger writer | cmd/internal/workbenchperftest/fixture.go; ledger_clone_darwin.go; ledger_clone_other.go | modified | private regular files and native CoW cloning |
| RunWithWriteBudget | cmd/internal/testdisk/run.go | new | procutil.NativeResourceReader and package normal-run callback |
| Package budget callers | couchcmd/testmain_test.go; couchcore/launchhelper_test.go; couchtty/reattach_testmain_test.go; titlepoller/runcli_process_test.go; threadactivity/testmain_test.go; workbenchperftest/testmain_test.go; sessionsync/testmain_test.go | modified/new | existing package TestMain lifecycle |

File paths in the last row are below cmd/internal. The helper accepts func()int, preserving couchcmd's testpair.Run(m) cleanup, couchtty's LifecycleParallelism setup, couchcore's launch-helper dispatch, and titlepoller's explicit child-role dispatch. Only couchcore/titlepoller normal modes are budgeted. No change to testpair's global executable lifecycle.

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

- [x] Record whole-Go-phase package bytes, pass/failure names, and top test/path attribution in a compact checked-in measurement artifact. Preserve raw local diagnostics and identify their limitations.
- [x] Finalize the proposed budget table from the measured distribution and the proved clone saving. The couchcmd cap must be below312,899,584 bytes (half of its whole-Go-phase baseline), with bounded headroom for retained stress tests and new regression cases. Name every selected package and exact bytes.
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

Files: new cmd/internal/testdisk/budget.go, budget_test.go, run.go, run_test.go; seven TestMain callers above; no Makefile or workflow configuration change.

- [ ] Write failing pure tests for budget comparisons/count scaling/overflow and stateful-reader tests for supported, unsupported, failed before/after, missing identity and decreasing counters. Preserve nonzero callback exit and call it exactly once under the documented policy.
- [ ] Implement the narrow callback runner using existing procutil, with explicit own-process bytes/cap diagnostic. Handle -count=N without silently reusing a one-run allowance; when -test.bench is nonempty, report that the ordinary-suite cap does not cover the benchmark workload, and retain counted fixture-growth tests.
- [ ] Wire normal TestMain paths for the measured package set without affecting helper dispatch/cleanup/lifecycle setup. Run the pre-fix byte-producing control or synthetic over-budget child and confirm nonzero status. A deleted/no-op budget must make this negative control fail; avoid writing the full large baseline solely to prove refusal.
- [ ] Run targeted testdisk tests and the selected package suites. Record exact native bytes/caps, not elapsed time, as the regression evidence.
- [ ] Commit the enforcement and tests.

### 4. Compare, document, and land

- [ ] Repeat identical per-package measurement against the implemented head; retain every failed attempt and classify it under the flake rule. Compare all packages, exact target reduction and unchanged cold-read/decode assertions.
- [ ] Update atlas/index.md and a focused testing document with the byte-budget scope, supported counter/filesystem, invocation, unsupported behavior, repeated-count/benchmark policy, and how an intentional cap revision is justified. Package caps are ordinary-suite limits, not benchmark limits; native unsupported results are explicit and are never reported as zero.
- [ ] Open an early draft via sdlc pr for CI overlap if useful. Run the recorded suite on pushed head with sdlc local-check run --name suite, then sdlc close with measured evidence. The binary owns the fresh-context review; no duplicate boundary review.
- [ ] Resolve findings; sdlc merge --yes supplies integration validation. Report verified PR/merge to the hub. No pair:0 staging or operator restart is needed for test-only owned-file changes; ask if the eventual implementation changes that surface.

## Revisions

- 2026-10-17 — The broader baseline found sessionsync71,299,072 bytes and an active couchcore already above30,000,000 bytes. Budget selection now covers every package with baseline own-process writes at least30,000,000 bytes, plus all five shared-fixture consumers. This adds budget-only TestMain callers for sessionsync/couchcore without changing their scenarios. Final couchcore baseline95,576,064 bytes supports the proposed128MiB cap; all other packages remain measured in the comparison, with their retained stress tests unchanged.

- 2026-10-17 — Whole-Go-phase baseline completed successfully in649.17s after804s in the machine queue. All116 packages passed; total1,640,353,792 observed own-process bytes; every binary had successful native samples and no final-read errors. See [write baseline](000658-write-baseline.md). Exact proposed caps are now recorded; seven guarded packages cover93.46% of observed writes. The measured supported environment and explicit exclusion of child/compiler/shell-phase writes remain unchanged.

- 2026-10-17 — Hub receipt22e69495-dbd9-4776-b29a-c05e145c7163 rejected the partial accounting as a sufficient optimization target. This candidate is suspended pending full-suite physical-device bytes and rough excluded-phase attribution. No implementation approval/change-code has occurred. A revised plan will use measured full-suite scale; CoW may remain only as a cheap secondary item.


### 2026-10-17 — whole-suite replacement design

**Reason and delta:** Hub receipts22e69495-dbd9-4776-b29a-c05e145c7163 andf6308a17-cd1f-4897-9d24-63e72d8c47cf require the full device scale, fixture churn and cheap levers, rather than the original ledger-only headline. This section supersedes the earlier task ordering, operating-envelope exclusions and completion interpretation. Earlier text remains the design history. Hub approval/change-code still required.

**Goal:** Reduce ordinary-suite disk writes without changing any required file size, authority check, persistent transition or durability call. Meet the original couchcmd >=50% own-process reduction, remove inert Git template files from the measured fixture class, avoid accidental private cold Go caches in shell tests, and defend the result with both file-tree and actual-byte budgets. No pre-implementation measurement beyond the completed short flush probe.

**Evidence:** [Whole-device baseline](000658-device-write-baseline.md): full Make passes,116 final Go packages pass,19,768,967,168 device bytes in844.188s. Final Go interval15.554GB; earlier recipes4.215GB. Data metadata5.216GB is a non-additive counter view. Partial serial couchcore probe observed23,465 created files/18,253 directories, including7,588 sample hooks across542 repository copies, with no dropped-event flags. Its8m diagnostic cap was incorrect against the634s serial baseline; retain its failure and partial status. Final four-test owned root-only probe passed and counted274 real F_FULLFSYNC attempts (265 in the preferences integration test). It changed neither the fcntl arguments nor the call outcome. Preserve all those real durability paths.

#### Ranked levers and chosen order

| Lever | Estimated bytes/files affected | Cost and decision |
|---|---|---|
| Empty Git template in fixture owners |7,588 inert sample-hook files in the partial core sample; installed templates26,788 logical bytes/copy,14,519,096 across542 copies, about35.5MB allocated-size equivalent; metadata saving unknown | Cheap; do first because it attacks demonstrated small-file churn. No exact device saving claimed. Also removes unneeded default template files beyond hooks. |
| Preserve caller Go cache resolution across private HOME | Install builder alone280.3MB sampled own writes; install+embedded device intervals~1.35GB include required linking/background, so are not a saving forecast | Cheap; do second. Reuse compilation/module caches and configured Go temp path, while retaining fresh candidate outputs and intentional cold-cache conformance. Exact saved bytes remain after-verification evidence. |
| Clone exact large ledger fixtures |~1.2GB repeated payload eliminated; about6.1% of raw device baseline before amplification | Cheap, proved feasibility; do third. Still required to meet couchcmd's>=50% goal. Original content/isolation/counter design above applies. |
| Share completed read-only repositories or immutable seeds per package | No defensible byte estimate from this audit | Omit initially. Inspected fixtures mutate branches, refs, worktree membership, origin history and substrate commits. One read-only malformed-output consumer does not justify a new cache owner. A shared immutable seed still needs private topology/config and most metadata; keep the simpler independent real repos. |

The ranking separates estimated payload bytes from metadata exposure; it is not an additive allocation of19.77GB. Build output reuse does not mean reusing an old Pair executable. No global Git setting, Go configuration edit, RAM disk or operator path changes.

#### Additional core concepts

The original WriteBudgetVerdict and exact-ledger writer remain current. Add:

| Name | Lives in | Status |
|---|---|---|
| FixtureTreeBudget / FixtureTreeVerdict |cmd/internal/testgit/budget.go|new|

FixtureTreeBudget compares a completed newly owned fixture tree's regular-file count, directory count, hook count and logical file bytes to explicit caps. It is a pure comparison over measured totals. Exact counters are uint64 with checked arithmetic. This is a retained fixture-tree budget, not a claim of every transient Git syscall or device bytes. Independently changing a fixture file after setup does not re-run the construction budget. The normal package byte budget separately limits real root-process writes.

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| RunWithEmptyGitTemplate |cmd/internal/testgit/run.go|new|one empty invocation-owned directory and scoped GIT_TEMPLATE_DIR environment|
| AuditFixtureTree |cmd/internal/testgit/tree.go|new|bounded walk of a newly owned fixture root, without following symlinks|
| Captured Go tool paths |tests/lib/pair-test-go-env.sh|new|go env before disposable HOME/env-i|
| Fixture owners |cmd/internal/couchcore/launchhelper_test.go; provision_git_test.go; slot_prepare_test.go|modified|existing package runner and real Git constructors|
| Shell fixture boundary |tests/lib/pair-test-bootstrap.sh; tests/with-isolated-pair.sh|modified|existing canonical environment builder and cleanup|

RunWithEmptyGitTemplate creates one empty directory for the ordinary couchcore test invocation and exports GIT_TEMPLATE_DIR for its real Git subprocesses. Existing helper-mode dispatch stays before this runner. Each subprocess environment that explicitly replaces inheritance receives that same owned template path; enumerate those Git creation seams during the source sweep. No user-configured Git template enters this fixture envelope. The two primary constructors retain their independent mutable repository/remote/worktree state and existing Git commands. The shell bootstrap uses its already-owned root for an empty template; it does not add a persistent directory.

Captured Go tool paths resolve GOCACHE/GOMODCACHE/GOTMPDIR in the caller's environment, before HOME changes, only when they were not explicitly provided. Reuse the existing with-isolated-pair behavior through one POSIX-compatible helper; do not duplicate resolution. Preserve explicit values, spaces, Go's disabled-cache sentinel and empty configured GOTMPDIR. When Go is absent, preserve the existing missing-tool behavior rather than making unrelated shell fixtures require it. Propagate only these tool paths and the owned empty Git template through the existing environment boundary. Test Pair/Couch/Zellij production-authority exclusion remains unchanged. Intentional clean-source GOCACHE overrides still win.

#### Budgets and growth contract

Keep the seven normal-run own-process byte caps above. They protect the original fixture consumers and every package whose sampled own bytes exceeded30MB. They deliberately do not claim to include short-lived Git/compiler children or metadata; the full-device and tree evidence identify that gap openly.

Proposed completed-constructor caps:

| Constructor | Regular files | Directories | Default sample hooks | Logical file bytes |
|---|---:|---:|---:|---:|
| newProvisionFixture |24|48|0|262,144|
| newPrepareWorld |64|128|0|1,048,576|

These are ceilings with headroom for ordinary Git version differences, not exact equalities. Validate them against the real constructor outputs after implementation; any necessary revision is logged with the actual inventory before changing the cap. The hook bound is exactly zero. The budget executes once inside each shared constructor before returning, not only in a separate model test, so new callers derive it. The tree contains only that constructor's new state; no operator tree is walked. Transient Git lock names already removed before completion are outside this budget; do not call it a whole-system creation counter.

For N=8 and2N=16 real independent constructor instances, check total retained files/directories/logical bytes <=N times the per-fixture cap, zero sample hooks, private refs/worktree membership and mutation independence. Initialization is O(N) repositories and O(N) small metadata files. No repeated walk per file; one O(entries) audit at the constructor boundary. Byte-stream growth for the ledger keeps its earlier N/2N assertion: one padding seed per size, N private clones/patches, no N*large-padding streaming. Stateful fake Go builds across8/16 isolated homes must write one fixed cache payload total for one unchanged compiler key, while intentional distinct cache keys write distinct bounded payloads.

#### State, lifecycle and limits

Actions: run a package or fixture; inspect a refusal; fix code; intentionally revise a cap with evidence; run portable correctness checks.

| Internal state/failure | Visible result | Recovery | Possible loss |
|---|---|---|---|
| Owned empty template ready | normal real Git setup | run fixture | none |
| Template creation/env setup fails | explicit test setup failure | inspect/fix | no accepted fixture |
| Fixture tree exceeds files/bytes/hooks cap | inventory totals and cap failure | fix or evidence-backed cap revision | failed attempt retained |
| Native root counter unsupported | explicit unsupported evidence; portable fixture guards still run | run supported byte verification | no physical-byte claim on unsupported platform |
| Supported counter read fails/regresses | budget failure | inspect/fix | no success claim |
| Go tool cache resolution fails when required | bootstrap failure preserving tool status | inspect/fix | owned root cleanup still runs |
| Test or callback fails/panics | original unsuccessful result | inspect/fix | partial diagnostic only; process-owned temps remain scoped |

The package runner owns its empty template from before m.Run through all normal callbacks and cleanups; it restores the prior environment and removes only its own root afterward. A failed callback still cleans up. Shell bootstrap owns its template under its existing disposable root and cleanup trap. No repository cache or new background watcher ships. FSEvents and the interposer remain temporary diagnostics, not CI correctness or acceptance machinery. No production persistence call is removed.

ARCH-DRY: one template owner per invocation, one Go-cache resolver, one tree comparison and one native-byte runner. ARCH-PURE: comparisons/count scaling are pure; native counters, filesystem walking and subprocess environment capture are thin seams. ARCH-PURPOSE: include device-scale Git/build causes as well as the original couchcmd goal. ARCH-MOCK: reuse existing stateful provisioning fixture; a stateful fake Go command checks cache sharing across homes, and ordinary install/embedded tests retain real conformance. ARCH-CONSTRAINTS: explicit normal-run/native-filesystem envelope, N/2N counts, no timing ceilings or unbounded concurrency. ARCH-STATE: scoped runner owns setup/active/cleanup; callers never share mutable repository authority. ARCH-FUNERAL: every new template/cache test artifact has an invocation/test cleanup owner. ARCH-SECURE: no-follow owned-tree audits, exact environment allowlist and no new authority over operator paths.

#### Replacement implementation tasks — one close boundary

1. **Git template and file budgets** — cmd/internal/testgit/{run,budget,tree}.go and colocated tests; couchcore/launchhelper_test.go, provision_git_test.go, slot_prepare_test.go and new fixture_budget_test.go; shell bootstrap and tests/lib/pair-test-bootstrap-test.sh.
   - [ ] Write pure budget tests (equal/over cap, checked arithmetic, zero-hook refusal) and owned-tree tests (symlinks not followed, byte inventory, unreadable/error result).
   - [ ] Add a real constructor regression with a deliberately populated parent template; prove the old setup materializes sample hooks/exceeds the guard. Add N=8/16 and independent mutation/refs checks through the actual shared constructors.
   - [ ] Implement the scoped empty-template runner, preserve helper dispatch/cleanup, wire constructors' audits and explicitly stripped Git environments. Enumerate real Git init/clone fixture sites in couchcore and shell recipes and record which derive the scoped setting.
   - [ ] Verify go test ./cmd/internal/testgit ./cmd/internal/couchcore -run 'Test(.*Fixture.*|.*Prepare.*|.*SlotPreferences.*)' -count=1, plus the existing environment-bootstrap shell contract. Retain real init/clone/worktree semantics and existing assertions. Commit.
2. **Go cache preservation** — tests/lib/pair-test-go-env.sh and new tests/lib/pair-test-go-env-test.sh; tests/lib/pair-test-bootstrap.sh; tests/with-isolated-pair.sh; cmd/internal/testpair/environment_test.go for the shell-contract entrypoint.
   - [ ] Add a stateful fake Go whose configured cache lives outside each disposable HOME. Verify unset/explicit cache values, spaces, disabled cache, missing Go, resolution failure, and intentional cold-cache override. At8/16 child homes with the same build key, one fixed payload is written to the shared cache; the old private-HOME behavior fails this count.
   - [ ] Resolve before HOME/env-i and share one helper between both callers. Keep every existing authority/argv/status/cleanup assertion in pair-test-bootstrap-test.sh, pair-test-binary-test.sh and testpair environment guards.
   - [ ] Run focused bootstrap/cache/binary shell-contract tests and the real install-layout/embedded-runtime fixtures. Fresh output builds still occur. Commit.
3. **Exact ledgers and native bytes** — execute earlier ledger-content/isolation/fallback and N/2N tasks, then the seven-package native TestMain runner tasks. Keep all old logical sizes and real read/decode assertions. Add over-budget negative control; unsupported is explicit; fail supported read errors. Run focused fixture/budget tests and selected package suites. Commit.
4. **Verify/document/close** — update atlas/testing map with file-tree and byte scopes/limits. Record matched before/after116-package own-byte table, constructor inventories, cache count proof and failure retention. The couchcmd result must be below312,899,584 bytes; proposed96MiB cap is stricter. Observe full Make device bytes for the changed head during a coordinated quiet window (prefer the required recorded suite body where the runner supports it); label noise, queue/body endpoints and comparable scope. No fixed device percentage is promised from the partial attribution. Open early draft PR for CI overlap, run recorded pushed-head suite, close through the automatic fresh review, satisfy normal merge gates, land, report PR/merge. No pair:0 staging or machine configuration change.

**Approval request:** confirm this cheap-first scope, omission of a read-only repository cache, and the explicit distinction between completed fixture-tree budgets and normal-package own-process bytes before change-code. The plan does not treat the partial core diagnostic as a passing package or the proposed file budget as a complete device-write meter.


### 2026-10-17 — approved replacement; explicit post-change attribution

Hubb835c6cc-0386-4592-8787-babadea817ba approved the replacement, including cheap-first scope, no repository cache and the two-budget envelope. Added contract: repeat the same whole-suite device measurement after implementation, record the before/after results, and file an unclaimed same-repo follow-up naming the largest remaining unexplained interval/share. Most of19.8GB remains unattributed; the hub's rough~2.5GB lever estimate is not a complete partition or promised device saving.

Direct cache verification extends task2: use invocation-owned GOENV configuration and Go cache roots, leaving parent GOCACHE/GOMODCACHE/GOTMPDIR unexported where needed to reproduce the original boundary. A temporary owned Go launcher records the actual resolved cache paths and their inventories immediately after the real Go command and before shell-root cleanup. Prime an owned configured build cache once; compare the original private-HOME cold cache with the fixed reused cache on identical real install/embedded fixture commands. Record regular-file count, logical payload and allocated bytes separately, plus which fresh link/output writes remain mandatory. Keep shared module-cache inputs distinct from compiler-cache writes. Restrict inventory to owned roots, never walk the139GB operator build cache or edit user configuration. This is required implementation verification, not another open-ended pre-design investigation. Temporary diagnostics do not ship. Counted stateful-cache regression remains the portable drift guard.

Before implementation, run change-code and honor its returned flow and gates. Post-change full-device verification uses the same diagnostic method/body and records queue/body boundaries and quiet-window slopes; attach the exact comparison to the issue before filing the remaining-share follow-up. No attribution gap is silently reclassified as solved.
