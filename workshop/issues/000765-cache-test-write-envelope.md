---
id: 000765
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '12df5e43cc22082eae5d8d0ae4d926bda4a26e40' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

Filed unclaimed per ops:0's 91039d1a-2f91-4fef-94cb-461c59be997b. Evidence: #739 close-review round4 (2026-10-21 02:21 PDT), artifact workshop/plans/000739-couch-start-up-safe-mode-no-automatic-injection-into-the-hub-restart-kind-reported-not-acted-on-one-generic-start-notice-per-worker-close-review.md. The required recorded #739 suite is still completing and uses -count=1; its only reported behavioral failure so far is the separately in-scope installed-command startup assertion.
