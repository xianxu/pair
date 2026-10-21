---
id: 000765
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f4690ab343144de2fb68457785009444266f5e5d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T03:09:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6341887b", done: "80fbe5c1"}
---

# Cache-enabled Go tests exceed physical write envelopes

## Problem

#739 close review observes different physical-write envelopes for plain cache-enabled go test and go test -count=1. At branch 1b948b3c040f6979eaafa77b1ad2da127f524da4, the reviewer measured couchcore 157257728 bytes against its 134217728 cap and couchtty 88002560 against its 67108864 cap using plain go test across five packages. The assertions pass; TestMain rejects the write counts. Both packages pass -count=1, as used by the required make suite. Earlier tty controls measured exactly 39800832 bytes with and without #739-added tests; new tests alone wrote zero.

The reviewer traced Go's cache-enabled filesystem-access logging as an invocation-dependent source of writes. Main-versus-branch cold/warm measurements are requested by the hub and pending; attribution is not yet proven. This issue is independent of #739's startup behavior and does not authorize any cap increase.

## Spec

Establish the supported invocation and physical-write accounting contract for ordinary package tests. Measure main and candidate in equivalent private cold and warm caches, distinguish test workload from cache instrumentation, and enforce the chosen contract consistently in test callers or accounting. Keep real workload writes bounded; do not silently waive a cache-enabled run or raise caps to conceal the difference. Coordinate the test-policy choice with the operator.

## Done when

- A controlled regression reproduces the invocation-dependent contribution and names whether it is pre-existing on main.
- Supported cache-enabled and uncached test modes have an explicit, enforced accounting contract.
- Real fixture-write regressions still fail the existing caps; no global cache clearing or cap increase is used as a fix.

## Plan

- [ ] Red native trace-heavy fixture and real go-test cache conformance, then suppress tracing only at the shared ordinary supported-budget boundary.
- [ ] Keep caps/repetition/failure semantics; qualify fresh cache-enabled and uncached core/tty runs and real excess refusal.
- [ ] Document result-cache policy, run recorded affected check, close and land.


## Log

Filed unclaimed per ops:0's 91039d1a-2f91-4fef-94cb-461c59be997b. Evidence: #739 close-review round4 (2026-10-21 02:21 PDT), artifact workshop/plans/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker-close-review.md. The required recorded #739 suite is still completing and uses -count=1; its only reported behavioral failure so far is the separately in-scope installed-command startup assertion.


## Revisions

### 2026-10-21 — invocation envelope owner (ARCH-DRY)

Measured main cold/warm core143224832/143204352 and tty105287680/104529920
already exceed134217728/67108864 caps; assertions pass. Retained full-core
control has137310208 physical bytes,130916787 of Go cache trace. Branch control
is lower, so #739 attribution was withdrawn. Full logs/hashes remain under
/Volumes/TestScratch/tmp/pair-slot7/pair739-cache.QFbUSM.

Choose no test-result caching for ordinary native-budgeted runs: the shared
RunWithWriteBudget clears registered test.testlogfile before m.Run and says so.
Go's testing.go2813 opens/traces only when nonempty; cmd/go test.go2174 returns
before cache.Put on a missing trace. Build caching is unaffected. Existing
-count/-cpu repetitions, real fixture accounting/caps and failure statuses stay.
Benchmarks/fuzz and unsupported native counters retain their current policy.
A real tiny-module go-test twice fixture proves fresh execution, not a forged
empty dependency log. Controlled trace-heavy reads must fit without their cache
instrumentation; a real256KiB payload still exceeds a4KiB cap. No production
surface or format changes. O(1) flag setup, no scan or per-operation wrapper.

Alternative subtraction would mix logical file length with physical IO and
could hide fixture writes. A new subprocess owner would add lifecycle and
cache identity machinery for an existing shared TestMain seam. Neither needed.
README/atlas describe the intentional result-cache tradeoff. Quick flow;
headless acceptance, no staging or machine/cache clearing.
