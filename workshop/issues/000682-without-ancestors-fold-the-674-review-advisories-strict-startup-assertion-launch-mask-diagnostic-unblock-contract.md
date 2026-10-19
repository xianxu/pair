---
id: 000682
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '6ef3dd084f1d2223e138847f8f416e84879f66af' # card fields mirrored from issue-cards; edit via sdlc
---

# without-ancestors: fold the #674 review advisories (strict startup assertion, launch-mask diagnostic, unblock contract)

## Problem

## Spec

Three Minor advisories from the #674 close review (SHIP, PR 438; sidecar workshop/history plan 000674-helper-import-sigint-loss-close-review.md), deferred to land #674 fast:

1. tests/without-ancestors-test.py test_cancellation_during_startup keeps a stale comment ("Python startup can itself translate SIGINT into exit 1") and only asserts a nonzero exit. With launch-time blocking, Popen returns after exec with the signals pending, so cancel always runs: assert code == 128 + cancellation.
2. The caller_blocked diagnostic is sampled after the launcher restores its mask, so it always reports []. Either sample the mask at launch time (inside the blocked window) or drop the field; only the gate bootstrap reports the child's own mask.
3. tests/without-ancestors.py unblocks SIGINT/SIGTERM unconditionally, and the other launchers (cmd/pair-go/main_test.go, tests/pair-embedded-runtime-test.sh) do not block before launch, so they keep the startup window. Document the contract in the helper comment (only a launcher that blocks across exec closes the window), and decide whether those launchers should block too. Also drop the no-op signal.signal(SIGINT, getsignal(SIGINT)) when ignore_in_caller is false.

## Done when

- test_cancellation_during_startup asserts 128 + signal, and the stale comment is gone.
- The caller_blocked diagnostic reports the launch-time mask, or is removed.
- The helper's unblock contract is documented; the other launchers block across exec or are explicitly noted as out of scope.
- without-ancestors-test passes, and test_cancellation_during_startup passes 25 runs in a row.

## Plan

- [ ]

## Log

### 2026-10-18
