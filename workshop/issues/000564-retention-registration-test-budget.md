---
id: 000564
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'f74e3ce7a21858f8abf5fc896430fbaf3b010a88' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T22:48:29-07:00
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

- [ ] Establish the deadline boundary and a meaningful red regression under controlled load/order.
- [ ] Correct the test seam or diagnosed behavior while preserving production timeout and retention contracts.
- [ ] Run semantic retention, intentional timeout and broad couchcore verification.

## Log

### 2026-10-14

Filed and released as independent follow-up during #218 close. No code change or extra rerun was made for this issue. The observed review failure and three isolated passes are recorded separately from the successful required suite.
