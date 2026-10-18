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

## Done when

- A per-package write measurement of the pair suite is recorded in the Log, before and after.
- `couchcmd.test` writes at least half less per run, with no test removed or weakened.
- A counted budget check fails if a package's writes grow past its recorded budget.

## Plan

- [ ]

## Log

### 2026-10-17

- Fresh dispatch4446225e-ba62-4e47-b3d4-225583aff47e verified Submitted; claimed658 unowned, then start-plan refreshed ariadne before its successful repeat. No implementation changes. Owned physical-counter measurement approved by hub23db90d8-1752-47fa-b99d-e020e65d1c0a (Submitted); the original fs_usage-style statement was a wording error, not privileged path evidence. Root fs_usage is unavailable; use kernel cumulative physical counters plus clearly sampled writable-path evidence first.
- Temporary diagnostic only: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-profile.yqKPI1 contains a small libproc probe, verbose-event sampler and one compiled couchcmd test/Pair binary. Calibration: a flushed4MiB regular-file write increments ri_diskio_byteswritten by4MiB. Root-process counters exclude descendants; per-test attribution samples serial verbose boundaries; open-path offsets are high-water evidence, not syscall write totals. First exploratory run retained some ambient Pair keys and will not be accepted as the clean baseline; corrected sampler removes all Pair/Couch/Zellij production session keys. Measurement artifacts and clean baseline follow before design/production code.

- Clean standalone couchcmd baseline (correct package cwd, all ambient Pair/Couch/Zellij session keys removed, supplied freshly built Pair, serial verbose attribution): exit0 in51.158s, own-process physical writes627,539,968 bytes. TestWorkbenchPerformanceMessageUnchangedEvidence accounts for600,489,984 bytes (95.7%); writable-FD evidence points to its twenty30,000,000-byte ledger-work-NN.jsonl files. Real PTY capture tests contribute9,474,048 and8,736,768 bytes; retain those independent stress surfaces. Exact per-path syscall totals remain unavailable, but the large writer is identified without privileged tracing.
- Read-only source sweep: shared workbenchperftest.NewFixture/writeLedger streams ASCII-space padding separately for each owner, then444-byte distinct launch/binding records. Ordinary-suite fixture payloads: couchcmd600,009,968; couchtty600,008,192; threadactivity30,014,336; titlepoller10,489,856; workbenchperftest90,000,444 logical bytes (1,330,522,796 combined). These are source-derived payload totals, not physical measurements. Tests intentionally decode real large JSONL; shrinking, NUL sparse holes, or shared hardlinks would weaken their contract.
- Owned temporary clonefile experiment: one30,000,000-byte all-space template plus twenty private clones with owner-specific tail patches produced600,000,000 logical ledger bytes, verified every byte,20 distinct inodes and20 distinct contents. Native own-process counters: template30,003,200 physical bytes; clones+patches81,920; total30,085,120. This is feasibility evidence, not a package after-measurement. Template and clones were removed by the temporary-directory owner. No repository code changed.
- Whole-Go-phase package baseline queued through the machine suite lock (diagnostic runner52309); preserves compiler/setup exclusion and root-only measurement limits. Await its package distribution before finalizing the plan/budgets or entering change-code. Preferred budget seam under investigation: shared TestMain wrapper around existing normal-run callbacks, using procutil.NativeResourceReader for exact before/after own-process counters; explicit unsupported-platform reporting rather than zero evidence.
