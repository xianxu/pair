---
id: 000658
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4a3ea94023ea2a2a32da21ba2058db3c10d61d34' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T21:32:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# pair test suites write ~20 MB/s (couchcmd.test): measure per package and cut, with a counted budget

## Problem

Full test suites are now nearly all of the fleet's disk writes (ops qa 2026-10-17-06): after the rolling relaunch the disk averaged 35.5 MB/s with suites running in 107 of 108 ten-minute samples, about 3.1 TB a day. The SSD reports 7% of rated life used at 89.4 TB written (rated near 1.3 PB), so this pace reaches rated wear in about a year. In a pair suite, `couchcmd.test` alone writes about 20 MB/s of its own (measured 10-16 23:13 with fs_usage-style per-process sampling).

## Spec

Measure first: per test package and per test, bytes written and to which paths (temp dirs, fixture trees, logs, SQLite or journal files, fsyncs). Then cut the largest writers without weakening what the tests prove: for example, smaller fixtures, sharing a fixture built once per package, in-memory fakes where the test is not about the filesystem, or fewer fsyncs in test-only stores. Defend the result with a counted check (bytes written per suite or per package, at a recorded budget), not timing.

Current proposed revision: target the measured full-suite scale using empty Git templates and preserved caller Go cache paths before the exact-ledger clone. Keep mutable repos independent and all real durability transitions. Add portable completed-fixture file-tree caps plus the supported native root-byte caps; do not mislabel these as complete device attribution. See the replacement design appended to the durable plan. Hub approval pending.

## Done when

- A per-package write measurement of the pair suite is recorded in the Log, before and after.
- `couchcmd.test` writes at least half less per run, with no test removed or weakened.
- A counted budget check fails if a package's writes grow past its recorded budget.

## Plan

Draft detailed plan: [suite write budget](../plans/000658-suite-write-budget-plan.md). The whole-Go-phase baseline is complete; await hub approval before change-code.

- [x] Record every measured Go package and large test/path; finalize exact proposed package caps.
- [ ] Obtain hub approval of the whole-suite replacement revision and pass change-code.
- [ ] Remove inert Git templates in shared fixture owners; enforce completed-fixture file/byte/hook caps and N/2N growth.
- [ ] Preserve caller Go cache resolution across isolated HOME with a counted stateful cache regression; keep cold-cache conformance.
- [ ] Preserve exact fixture bytes, logical sizes and independent files using fixture-local CoW templates with portable streaming fallback; add N/2N write-growth regression.
- [ ] Enforce own-process physical write budgets in the measured large-writer package TestMain paths; cover failed/unsupported counter evidence and over-budget refusal.
- [ ] Record matching after-measurement, run recorded suite/CI, close review and land.

## Log

### 2026-10-17

- Fresh dispatch4446225e-ba62-4e47-b3d4-225583aff47e verified Submitted; claimed658 unowned, then start-plan refreshed ariadne before its successful repeat. No implementation changes. Owned physical-counter measurement approved by hub23db90d8-1752-47fa-b99d-e020e65d1c0a (Submitted); the original fs_usage-style statement was a wording error, not privileged path evidence. Root fs_usage is unavailable; use kernel cumulative physical counters plus clearly sampled writable-path evidence first.
- Temporary diagnostic only: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-profile.yqKPI1 contains a small libproc probe, verbose-event sampler and one compiled couchcmd test/Pair binary. Calibration: a flushed4MiB regular-file write increments ri_diskio_byteswritten by4MiB. Root-process counters exclude descendants; per-test attribution samples serial verbose boundaries; open-path offsets are high-water evidence, not syscall write totals. First exploratory run retained some ambient Pair keys and will not be accepted as the clean baseline; corrected sampler removes all Pair/Couch/Zellij production session keys. Measurement artifacts and clean baseline follow before design/production code.

- Clean standalone couchcmd baseline (correct package cwd, all ambient Pair/Couch/Zellij session keys removed, supplied freshly built Pair, serial verbose attribution): exit0 in51.158s, own-process physical writes627,539,968 bytes. TestWorkbenchPerformanceMessageUnchangedEvidence accounts for600,489,984 bytes (95.7%); writable-FD evidence points to its twenty30,000,000-byte ledger-work-NN.jsonl files. Real PTY capture tests contribute9,474,048 and8,736,768 bytes; retain those independent stress surfaces. Exact per-path syscall totals remain unavailable, but the large writer is identified without privileged tracing.
- Read-only source sweep: shared workbenchperftest.NewFixture/writeLedger streams ASCII-space padding separately for each owner, then444-byte distinct launch/binding records. Ordinary-suite fixture payloads: couchcmd600,009,968; couchtty600,008,192; threadactivity30,014,336; titlepoller10,489,856; workbenchperftest90,000,444 logical bytes (1,330,522,796 combined). These are source-derived payload totals, not physical measurements. Tests intentionally decode real large JSONL; shrinking, NUL sparse holes, or shared hardlinks would weaken their contract.
- Owned temporary clonefile experiment: one30,000,000-byte all-space template plus twenty private clones with owner-specific tail patches produced600,000,000 logical ledger bytes, verified every byte,20 distinct inodes and20 distinct contents. Native own-process counters: template30,003,200 physical bytes; clones+patches81,920; total30,085,120. This is feasibility evidence, not a package after-measurement. Template and clones were removed by the temporary-directory owner. No repository code changed.
- Whole-Go-phase package baseline queued through the machine suite lock (diagnostic runner52309); preserves compiler/setup exclusion and root-only measurement limits. Await its package distribution before finalizing the plan/budgets or entering change-code. Preferred budget seam under investigation: shared TestMain wrapper around existing normal-run callbacks, using procutil.NativeResourceReader for exact before/after own-process counters; explicit unsupported-platform reporting rather than zero evidence.

### 2026-10-17 — Completed baseline and proposed plan

- Whole-Go-phase diagnostic completed exit0 in649.17s after804s in the machine suite queue. All116 package binaries passed; total1,640,353,792 own-process physical bytes. Every binary had successful counter samples and no final-read errors. This remains sampled final lower-bound observation, not child-process/compiler/shell/Lua or whole-machine writes. See [all package and leading test measurements](../plans/000658-write-baseline.md).
- Largest packages: couchcmd625,799,168; couchtty609,632,256; couchcore95,576,064; workbenchperftest90,030,080; sessionsync71,299,072; threadactivity30,089,216. Titlepoller10,592,256 is also a shared-fixture consumer. Seven-package cap proposal covers1,533,018,112 bytes (93.46% of observed writes), with explicit measured scope. Couchcore leading paths are private resource history stages, thread-store publications and launch-failure bounds; retain all those tests.
- Proposed caps: couchcmd96MiB, couchtty64MiB, workbenchperftest64MiB, threadactivity48MiB, titlepoller24MiB, sessionsync96MiB, couchcore128MiB. Native own-process counters are enforced around normal TestMain completion on Darwin+cgo; unsupported platforms report unavailable evidence and keep portable correctness tests. Existing procutil is the one native observer. Per-fixture templates end with t.TempDir; no global cache, no hardlink sharing, no smaller files or relaxed assertions. Default suite and matched serial after-measurements both required. Hub approval/change-code pending; repository source remains unchanged.

- Measurement and proposed plan sent to hub in receipt5d783d91-de3c-48fc-ab4a-7df5b0a19037, verified Submitted. Await approval of the seven-package scope and explicit native-counter/filesystem envelope before change-code; no implementation edits.


- Hub22e69495-dbd9-4776-b29a-c05e145c7163 verified Submitted: current1.64GB own-test-process baseline is not sufficient to identify the largest device-write share. Approval is withheld; no source edits. Next measure one full make test using physical disk0 IOBlockStorageDriver Bytes(Write), a30s quiet window before/after while holding both normal suite tokens, plus sampled identity-fenced owned descendant counters, Make phase timestamps and writable-FD paths. Normal cache settings/fresh shared Pair build remain in effect; no cache/machine configuration edits. Report device noise and process-sampling gaps explicitly.
- Read-only excluded-phase leads (not yet byte attribution): hermetic shell bootstrap preserves explicitly exported cache variables but may lose user go-env configuration when HOME changes, causing private cold caches in install/embedded-runtime fixtures. Runtimebundlegen clean-source test explicitly creates an empty GOCACHE. AgentScratch and workspace share physical SSD disk0s2; it is an APFS volume, not another device/RAM disk. Native process observation now includes descendants with PID-generation fencing; short-lived descendants still make sums lower bounds.

- Revised diagnostic is running/queued as exec session9158: temporary device_suite.py under the recorded diagnostic root. It acquires both ordinary suite tokens, exports the real holder PID for nested Make reentrancy, records30s quiet-before, runs normal-cache fresh-candidate make -k WF_TEST_WEIGHT=2 test, then records30s quiet-after. PID-generation-fenced descendant counters and sampled writable paths accompany physical disk0 counters and Make phase timestamps. It has not run yet; next in line after pair:4. Outputs device-suite/result.json, device.jsonl, phases.jsonl and make.log remain local. No repo source or machine/cache settings changed.
- Native descendant-probe calibration passed: owned child flushed4,194,304 bytes; fenced counter returned4,194,304 with a nonempty process start generation. Ambient30s sample while other suites ran:175,480,832 device bytes (5.843MB/s); this is context, not attribution to our suite.
- Cache resolution was reproduced without a build: parent has none of GOCACHE/GOMODCACHE/GOTMPDIR/GOENV exported; go env resolves AgentScratch/user configuration. With disposable HOME and empty environment, Go chooses private HOME/Library/Caches/go-build and HOME/go/pkg/mod, with empty GOTMPDIR. tests/lib/pair-test-bootstrap.sh forwards only explicit cache variables. Ordinary-suite real builders include tests/pair-go-install-layout-test.sh:55 make install and tests/pair-embedded-runtime-test.sh:18 runtimebundle-generate; runtimebundlegen/clean_source_test.go:20 intentionally uses a fresh compiler cache. These are source/probe leads, not measured shares yet.

### 2026-10-17 — whole-device full suite passed

- Full quiet-window diagnostic completed: make exit0, all116 final Go packages passed, 844.188s body, physical disk0 delta19,768,967,168 bytes (23.418MB/s). Final Go interval accounts for approximately15,554,056,192 bytes; preceding setup/shell/editor recipes4,214,910,976. See [whole-device baseline](../plans/000658-device-write-baseline.md) for exact counters, per-category lower bounds, APFS views, phase limits and local raw evidence.
- Owned descendant counters total2,288,795,648 sampled bytes across832 generations; short-process/final-sample gaps remain. Data APFS metadata rose5,216,157,696 bytes; views overlap and cannot be added into a device partition. VM/swap counters unchanged. Quiet slopes3.944/2.482MB/s are context, not exact background subtraction.
- Both suite tokens released after30s post-window. Pair:4 release receipt8fb8d8ac-baa0-4aea-8c94-0682b60d95e9 verified Submitted; their unchanged rerun may proceed. No remaining interleaving obligation.
- The original ~1.2GB ledger payload saving is only~6.1% of this raw device total. Continue bounded attribution of final-Go device bursts/file churn; do not choose compiler-cache or metadata optimization from incomplete process sums. No source changes, change-code or implementation approval.

### 2026-10-17 — partial couchcore churn diagnostic

- Hub18724da1-b2ea-4b45-94d5-f905718ce94e verified Submitted approved bounded couchcore attribution. Unprivileged owned-root file events observed23,465 distinct created file paths and18,253 directories, zero dropped/scan flags. Git sample hooks alone7,588 paths (14 names across542 copies); classify this as churn evidence, not device-byte attribution. Detailed source/path/count limits are appended to the whole-device baseline.
- The probe's8m serial cap was a harness mistake against the known634.448s passing serial baseline. Exit2 from that cap, no earlier FAIL lines; current TestWorktreeAddUsesWriteBudget had run2s. Preserve raw logs, do not call it a product flake/pass, and do not blindly rerun the whole package. Observer exited0; all measured owned PID generations were gone afterward. No code edits or change-code.
- Tiny owned Go interposition calibration proved libc-operation visibility; installed Go source shows File.Sync normally uses F_FULLFSYNC. A generic variadic fcntl forwarding shim is too broad for arbitrary product calls; do not use the calibration shim on the suite. A narrow safe flush-count diagnostic or the operator's60s trace remains the next attribution step.

### 2026-10-17 — final pre-code probe and replacement plan

- Hubf6308a17-cd1f-4897-9d24-63e72d8c47cf verified Submitted: one short owned F_FULLFSYNC probe, then stop measuring and plan. Completed four selected existing tests, exit0 in12.650s:274 root/private-TMPDIR F_FULLFSYNC attempts,265 in TestSlotPreferencesIndependentAcrossRestartResumeAndFresh. Real fcntl still ran; calibrated arm64 forwarding preserved arguments/stack/errno and skipped child logging. Temporary source/artifacts remain local; no tracer ships.
- Probe observed623 created files/349 directories, zero dropped/scan flags,23,777,280 last-sampled own bytes. Scope is four selected tests, not the package. No further pre-implementation measurement.
- Source-only reuse audit found mutable repository/worktree/origin/substrate state in measured shared fixtures; omit a package-wide repository cache initially. The replacement plan ranks measured payload versus unknown metadata savings, starts with cheap empty-template and Go-cache fixes, retains the ledger saving and proposes both completed-tree and native-byte caps. Await hub approval/change-code; no source edits.

## Revisions

- 2026-10-17 — Measured dominant fixture construction rather than assuming fsync/logging churn. Candidate design preserves full30MB files and decoder/authority checks by CoW cloning private all-space templates. Reject fixture shrinking, hardlinks and NUL sparse files because they weaken the actual coverage. A shared native-counter runner defends the whole normal-run callback in the large-fixture consumer packages; final numeric caps await the queued distribution and hub approval.

- 2026-10-17 — Hub required whole-device/full-suite attribution before choosing the headline optimization. Suspend the seven-package-cap/CoW-first proposal pending that measurement. The ledger saving remains proved feasibility, not the accepted target; re-plan against the biggest measured share and report revised numbers before code.

- 2026-10-17 — After the final flush probe, replace the suspended ledger-only task ordering with the whole-suite cheap-first design appended to the same durable plan. Preserve all prior decisions/evidence; explicitly omit unsafe/low-yield shared mutable repository reuse and distinguish fixture-tree budgets from physical root-byte budgets.
