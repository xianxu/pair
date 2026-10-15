---
id: 000564
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-15
estimate_hours:
card_mirror: '6339cacd8c8253b1ca30663bd4695ef3a2864b88' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T22:48:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e96f9574", done: "280bc332"}
---

# Retention resume test expires its registration budget under load

## Problem

The whole-issue review of #218 observed TestRetentionExplicitResumeTouchesButBackgroundReattachDoesNot/false fail in a broad couchcore package run: the fixture's 200 ms registration deadline expired before any completed ownership poll. The unchanged test passed three isolated repetitions. This suggests deadline sensitivity under load, but isolated passes do not establish the cause or erase the failed broad run.

Evidence is retained in [the close-review sidecar](https://github.com/xianxu/pair/blob/d0374e70fb689f559bbed3271d2981737b83b748/workshop/plans/000218-couch-thread-startup-takes-8-85s-and-nothing-has-measured-where-it-goes-close-review.md). The required sdlc-local/suite on b289539f9ce2211ce3b63d2d1bc71d4b5efef4cd passed, and the M2 reviewer independently passed the full couchcore package. The final reviewer returned SHIP with medium confidence and no findings, explicitly preserving this failure.

The test and warmDetachedThread helper are unchanged from main. warmDetachedThread sets resumeRegistrationTimeout to 200 ms; the retention test uses real coordinated storage and t.Parallel. #218's registration wrapper changes a context value to bypass cached absence; it preserves the parent's cancellation and deadline. The existing poll loop is otherwise unchanged. Which step consumes the budget in the failing run still needs measurement; do not infer a production lifecycle defect or dismiss it as unrelated solely from isolated passes.

## Spec

Reproduce the budget exhaustion through controlled scheduling/storage barriers or retained phase evidence. Separate semantic retention assertions from tests intentionally exercising short registration deadlines. Make the retention test reliable under a busy host without sleeps, retries, weakened retention checks or an unmeasured production timeout increase. Preserve the assertion that explicit resume updates last use while background reattach does not.

## Done when

- A deterministic regression or captured phase evidence identifies where the 200 ms budget expires and why no completed ownership observation occurs.
- The retention-semantic test preserves both explicit/background assertions while using a bounded test strategy that is not an incidental scheduler-speed requirement.
- Targeted timeout/lifecycle regressions and a broad couchcore run pass; the original failure remains documented rather than filtered out.

## Plan

- [x] Establish the deadline boundary and a meaningful red regression under controlled load/order.
- [x] Correct the test seam or diagnosed behavior while preserving production timeout and retention contracts.
- [ ] Run semantic retention, intentional timeout and broad couchcore verification.

## Log

### 2026-10-14

Filed and released as independent follow-up during #218 close. No code change or extra rerun was made for this issue. The observed review failure and three isolated passes are recorded separately from the successful required suite.

### 2026-10-15 — root cause and proposed quick flow

- Claimed/started from main e85237e7e1951bc61ce3d9527ed7ad2dfb6e47d3; substrate refreshed to ariadne 72bf044e19b3be76bc22851008aaf9fbf2f30dc8. A temporary controlled-deadline probe spent the inherited 200ms registration budget at its arming boundary, before returning to the ownership loop. Resume deterministically failed with `ownership poll consumed the budget`, `0 ownership polls`, `last ownership check: none`; the session remained live. Its one later fake artifact callback belonged to failure cleanup, not a completed registration observation. No sleeps or workload inflation were used; temporary probe removed.
- Root cause: the retention-semantic case inherits a deadline intended for warm failure tests. The deadline begins after acknowledge and before the first registration observation; wall time can expire before that observer is scheduled. `FixedClock` only controls domain timestamps, while the registration factory defaults to context.WithTimeout.
- Proposed correction: a retention-only fixture uses the existing registrationDeadline seam with parent-linked context.WithCancel, so only the enclosing test budget/cancellation bounds registration. Keep warmDetachedThread's 200ms budget and all intentional timeout tests unchanged (ARCH-DRY, ARCH-ORDER). Preserve the real storage coordinator and both exact retention/environment assertions.
- Proposed regression: retain the event-spent-before-first-poll characterization; prove the retention fixture propagates its parent deadline rather than adding a 200ms deadline, and parent cancellation still terminates registration. Run semantic retention and intentional timeout/lifecycle regressions, broad couchcore (normal plus targeted race), make -k build and required sdlc-local suite before SHIP/landing. No production changes expected.

- 2026-10-15: Hub approved the design. The new deadline-propagation regression failed under the unchanged warm fixture: registration had a 200ms deadline while its enclosing test budget had about 590s remaining. Retention-only fixture now injects context.WithCancel through the existing registrationDeadline seam and uses the enclosing remaining test budget; warmDetachedThread and production code are untouched. All three deterministic cases plus the original semantic test passed three repetitions (9.338s). The original exact LastUse, process-list and PAIR_RETENTION_BACKGROUND assertions remain unchanged. Targeted lifecycle/race and broad couchcore checks are running.
