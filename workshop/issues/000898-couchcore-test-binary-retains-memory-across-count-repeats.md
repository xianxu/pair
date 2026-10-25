---
id: 000898
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'f5b2414deaaa22dd1e882e842c7d2fab2a1a9a34' # card fields mirrored from issue-cards; edit via sdlc
---

# couchcore test binary retains memory across -count repeats

## Problem

The couchcore test binary's memory grows with `-count`, so a repeated run cannot prove a change stable. Measured on 2026-10-25 by pair:10 while proving #885 M3, unsandboxed with `/usr/bin/time -l` on a prebuilt `couchcore.test`:

- main (`27793cd53` code): `-count=1` 677 MB maximum RSS, passes; `-count=2` reached 1.03 GB and timed out at `-test.timeout=20m` (1200 s).
- #885 branch: `-count=1` 677 MB; `-count=2` 1.25 GB.
- Ten sequential `-count=1` processes on the #885 branch: all pass, 0.51-0.73 GB each, so a single pass is bounded; the growth is retention across repeats within one process.

Same shape as #896 (couchtty: 1.37 GB at `-count=1`, 2.61 GB at `-count=2`, 9.6 GB at `-count=10`), a different package. #896's fixes are the likely pattern: Console fixture ownership (a test's fixture state released when the test ends), joined drainers (reader goroutines joined at cleanup instead of left running), and pointer reset (package-level references cleared between tests).

## Spec

Find which couchcore tests retain memory across repeats (heap profile across `-count=2`, or per-test RSS deltas) and release it at each test's cleanup, so `-count=N` peak RSS stays near `-count=1` and `-count=2` finishes well inside its timeout.

## Done when

- couchcore `-count=2` peak RSS is within about 20% of `-count=1`, and passes.
- `-count=10` completes without timing out, with its peak RSS recorded in the Log.
- Each retained-state source found has a cleanup fix in the owning fixture, not a global workaround such as forcing GC between tests.

## Plan

- [ ]

## Log

### 2026-10-25
