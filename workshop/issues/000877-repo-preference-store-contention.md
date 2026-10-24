---
id: 000877
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '7b516c567d63ab93ab83127ac557f54bf2a32924' # card fields mirrored from issue-cards; edit via sdlc
---

# Repository preference publication escapes typed store contention

## Problem

During #866 full local Go verification, TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents failed at repopreferences_store_test.go:123 with lock thread store: thread store busy; retry. The test starts16 independent writers (8 repositories x Claude/Codex) and expects all publications to succeed. The exact test also reproduced alone, failing after1.12s (package1.496s).

Repro: go test ./cmd/internal/couchcore -run ^TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents$ -count=1 -v -timeout=30s . The terminal period is prose punctuation, not an extra package argument.

rememberRepoConfigurationContext retries only unix.EWOULDBLOCK/EAGAIN at repopreferences_store.go:86-94. withUpdateLockContext uses the fixed1s storeReadLockWait for its nested store lock and can return typed ErrThreadStoreBusy instead. That error escapes unchanged. These source/test files are unchanged from main on #866 candidate b6f9b48d12d5c06eb09f9dbbe5d36a4ea00c3f35. Source log /Volumes/TestScratch/pair866-full-go-local-gr31bcx7/go.log (full run still finishing when filed).

pair:6 confirmed #854/PR590 does not cover this: it caps waits at a caller deadline, but context.Background has none, so the1s busy result remains intentional. #873 only changes startup acquisitions. TL266b90f8 directed separate unclaimed filing; do not fold this into Zellij startup readiness.

## Spec

Resolve the mismatch between concurrent repository-preference publication and the busy-store contract. Define the caller-budget/busy behavior explicitly and preserve context cancellation and the existing bounded store-wait policy. Do not silence the failed publication by deleting assertions or increasing a test timeout blindly.

## Done when

- Concurrent independent preference publications preserve unrelated repositories and per-agent arguments under the documented busy/caller-budget policy.
- Deterministic contention coverage distinguishes retryable contention, caller cancellation/deadline, and a genuine bounded busy refusal.
- The existing concurrent-instance test completes under that contract and targeted store-wait tests still pass.

## Plan

- [ ]

## Log

Filed unclaimed from #866; no fix made in that branch. Reported exact isolated reproduction exit1 and full-set failure. No flake classification claimed.
