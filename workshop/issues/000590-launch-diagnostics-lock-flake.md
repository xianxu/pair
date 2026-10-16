---
id: 000590
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '7f50530700a1eacce487bc39f6a0e2916229fd8d' # card fields mirrored from issue-cards; edit via sdlc
---

# Diagnose concurrent launch diagnostics lock timeout under load

## Problem

`TestLaunchFailureStoreReopenConcurrent` intermittently fails under full-suite load when one of 17 concurrent diagnostics Save calls times out acquiring the store lock. The later retention assertion also fails because that failure record was not saved. This is independent of #571's Codex adapter; the diagnostics store and its test are unchanged by that branch.

## Spec

Find and reproduce why the concurrent diagnostics writer exceeds its bounded lock budget under load. Save currently retries nonblocking flock with 10ms pauses against a one-second budget, then performs the read/upsert/atomic write while holding the lease. Possible contributors include aggregate serialized IO/scheduling and inherited flock retention during another fork; neither has been measured here. Distinguish actual writer contention, scheduling, and inherited lease retention before choosing a fix. Do not merely enlarge the timeout or weaken the retention assertion.

## Done when

- A controlled regression establishes the contention/lifecycle cause and preserves the intended bounded admission behavior.
- Concurrent valid diagnostics writes retain all required history, while the explicit lock-timeout test still refuses a genuinely held lease.
- Repeated isolated and loaded runs report before/after failure counts without a timing-only fix.

## Plan

## Log

### 2026-10-15 — first seen during #571 corrected-code full suite

- Exact tested head: `8d79c161e1ba68c07b5884fefb657e13272b145f`.
- Test `TestLaunchFailureStoreReopenConcurrent`, `cmd/internal/couchcore/launchfailure_store_test.go:32,42`, failed after 1.10s: `launch diagnostics lock timeout: resource temporarily unavailable`; retention then lacked a required record.
- Recorded full-suite log: `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/fe1008b1860377ef1d9a72d918548b57.log`; SHA256 prefix `73f371f8e9a9`; runtime12m57s, couchcore481.625s. Every recipe/package completed. Its other failure was #571's separately corrected protocol-vocabulary inventory classification; broadcast and review restoration passed.
- One targeted unchanged repeat: `go test ./cmd/internal/couchcore -run '^TestLaunchFailureStoreReopenConcurrent$' -count=10` passed all10 iterations, package5.041s. Log `/tmp/pair571-launchfailure-lock-check.log`.
- No matching active issue found. Filed unassigned at the TL's direction. Do not fold a diagnostics-store fix into #571; its already queued corrected-head suite may proceed, and any further unrelated flake stops retries for a suite-wide test-economy review.
