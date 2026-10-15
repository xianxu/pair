---
id: 000564
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-15
estimate_hours:
card_mirror: 'c3a852f0a2d33e0f256d699a9a46ffe667533608' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T22:48:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "be817c06", done: "280bc332"}
actual_hours: 1.09
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
- [x] Run semantic retention, intentional timeout and broad couchcore verification.

## Log

### 2026-10-14

Filed and released as independent follow-up during #218 close. No code change or extra rerun was made for this issue. The observed review failure and three isolated passes are recorded separately from the successful required suite.

### 2026-10-15 — root cause and proposed quick flow
- 2026-10-15: closed — Retention-only fixture injects parent-linked cancellation through existing registrationDeadline; enclosing resume uses remaining test budget, with exact explicit/background retention, process-list and child-env assertions preserved. Deterministic pre-poll expiry reproduces ownership phase/0 polls/none. Parent-deadline regression failed under old 200ms timer; parent-detachment mutation fails cancellation regression. Three repetitions of semantic cases passed, targeted lifecycle/timeout -race x3 passed (22.454s), broad couchcore passed (258.789s), make -k -B build passed at clean 1212443fa837a747a724d52ae4836b31376cf8a5, required sdlc-local/suite passed on that head (10m50s; couchcore 326.501s). No atlas: test-only seam correction; production and intentional timeout fixtures unchanged.; review verdict: SHIP

- Claimed/started from main e85237e7e1951bc61ce3d9527ed7ad2dfb6e47d3; substrate refreshed to ariadne 72bf044e19b3be76bc22851008aaf9fbf2f30dc8. A temporary controlled-deadline probe spent the inherited 200ms registration budget at its arming boundary, before returning to the ownership loop. Resume deterministically failed with `ownership poll consumed the budget`, `0 ownership polls`, `last ownership check: none`; the session remained live. Its one later fake artifact callback belonged to failure cleanup, not a completed registration observation. No sleeps or workload inflation were used; temporary probe removed.
- Root cause: the retention-semantic case inherits a deadline intended for warm failure tests. The deadline begins after acknowledge and before the first registration observation; wall time can expire before that observer is scheduled. `FixedClock` only controls domain timestamps, while the registration factory defaults to context.WithTimeout.
- Proposed correction: a retention-only fixture uses the existing registrationDeadline seam with parent-linked context.WithCancel, so only the enclosing test budget/cancellation bounds registration. Keep warmDetachedThread's 200ms budget and all intentional timeout tests unchanged (ARCH-DRY, ARCH-ORDER). Preserve the real storage coordinator and both exact retention/environment assertions.
- Proposed regression: retain the event-spent-before-first-poll characterization; prove the retention fixture propagates its parent deadline rather than adding a 200ms deadline, and parent cancellation still terminates registration. Run semantic retention and intentional timeout/lifecycle regressions, broad couchcore (normal plus targeted race), make -k build and required sdlc-local suite before SHIP/landing. No production changes expected.

- 2026-10-15: Hub approved the design. The new deadline-propagation regression failed under the unchanged warm fixture: registration had a 200ms deadline while its enclosing test budget had about 590s remaining. Retention-only fixture now injects context.WithCancel through the existing registrationDeadline seam and uses the enclosing remaining test budget; warmDetachedThread and production code are untouched. All three deterministic cases plus the original semantic test passed three repetitions (9.338s). The original exact LastUse, process-list and PAIR_RETENTION_BACKGROUND assertions remain unchanged. Targeted lifecycle/race and broad couchcore checks are running.

- 2026-10-15: Full couchcore package passed on the implementation (258.789s); targeted semantic/timeout/lifecycle -race checks passed three repetitions (22.454s). make -k -B build passed; Pair and Couch embed clean revision 1212443fa837a747a724d52ae4836b31376cf8a5 with vcs.modified=false. Required sdlc-local suite remains in the machine queue. The pre-fix three isolated semantic passes (8.728s) remain distinct from the controlled budget-expiry reproduction and the inherited-deadline red regression.

- 2026-10-15: Required sdlc-local/suite passed on implementation 1212443fa837a747a724d52ae4836b31376cf8a5 (exit 0, 10m50s after 2933s queue); couchcore passed again in the full suite (326.501s). Full log SHA-256: cc615abac0cf4612730414bd3759206faadeaa4b9b5fc44639197f2617e7f273. A temporary parent-detachment mutation in the retention fixture made the cancellation regression fail deterministically (resume incorrectly succeeded); source restored byte-for-byte and git diff clean before close. This confirms the cancellation case is not merely observing final cleanup cancellation. No production, timeout-fixture or retention assertion changed.

## Revisions

### 2026-10-15 — integration with #528 public warm attachment

- Reason: main advanced after the first integrated suite. Merging #528 at e0db8d94a51ffd320dbaafb943c1dea432473da5 moved public warm resume from lifecycle ownership registration to listed-terminal attachment. The original retention-semantic test passes, but the three new regressions were observing the retired public route. The integrated suite failed exactly those three tests (9m50s); this is an actionable integration failure, not a flaky rerun to ignore.
- Delta proposed to hub: retain historical zero-ownership-poll expiry characterization directly through awaitResumeRegistration. Move the current resume deadline/cancellation observations to ListedSessionHook, preserving the parent-linked registrationDeadline factory and exact semantic assertions. No production changes or extra timer. Reopen working; obtain refreshed close review after targeted/broad/suite checks.
- Merge stopped with SIGINT while SDLC queued its upstream-baseline diagnostic, before any landing. Earlier integrated 0d87637085789339fb4c823ecac6b46ee9a73069 suite passed (10m10s after 3305s queue); original implementation suite and SHIP remain historical evidence, not validation of this revised integration.

### 2026-10-15 — approved revision checked against #528

The new listed-terminal route still arms resumeRegistrationTimeout through registrationDeadline (warm_attach.go); a deterministic spent-budget public-resume test passes, proving both attachment attempts fail with DeadlineExceeded before observation. The 200 ms fixture hazard therefore remains. Move parent deadline/cancellation observers to ListedSessionHook; check child cancellation inside the hook before cleanup can mask detached ancestry. Keep the direct zero-ownership regression because launch_existing.go still calls awaitResumeRegistration for cold/fresh resume. No production change.

- Revised red/green: public spent-budget attachment and direct production-used registration characterization pass. All four regressions and exact semantic cases pass x3 (2.855s). Restoring a 200ms timer fails the attachment parent-deadline assertion; detaching the parent fails cancellation inside the listed-session hook before cleanup. Both mutations restored byte-for-byte.
