---
id: 000658
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c1c0684152908d85c12b99d327b59e1bfa8001b8' # card fields mirrored from issue-cards; edit via sdlc
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
