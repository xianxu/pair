---
id: 000219
status: open
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
---

# Collapse NotificationLifecycle's boolean constellation into a tagged turn state

## Problem

`NotificationLifecycle` carries five booleans — `Active`, `Completed`,
`ActivitySeen`, `GracePending`, `IdleNotified` — which declare 32 states and
mean roughly four. `Completed` is redundant with `!Active` except to
distinguish "never opened", and the guards disagree across cases as a result:
`ObservationNativeCompletion` tests `Active && !Completed`,
`ObservationBareReturn` tests `!Active || Completed`, and
`ObservationStopped` tests bare `Active`. Nothing writes down which
combinations are legal, so each new case re-derives the predicate and the
reader cannot tell an intentional difference from an oversight.

Raised as BR-9 in pair#171's boundary review (`ARCH-ORDER`), where the idle
floor added the fifth boolean. Pre-existing; #171 extended it rather than
introduced it, and the refactor was deliberately not bundled into that issue
because it rewrites every case in `Reduce` and the tests that inspect the
flags directly — separable work with its own risk.

## Spec

Collapse the representation to a tagged state — `turnState{none, open,
closed}` — with the open-substate flags (`ActivitySeen`, `GracePending`,
`IdleNotified`) hanging off `open`, so illegal combinations become
unrepresentable rather than merely unwritten. Every guard in `Reduce` then
reads the tag, and a difference between two cases is visible as a different
tag rather than a different hand-written boolean expression.

Behaviour must not change: the reducer's existing unit rows and
`FuzzNotificationLifecycleAtMostOncePerGeneration` are the contract, and both
should pass untouched apart from mechanical field renames.

## Done when

- `NotificationLifecycle` exposes a tagged turn state; no caller reads a
  redundant `Completed`/`Active` pair to infer the same fact.
- Every `Reduce` case guards on the tag; no two cases express the same
  precondition differently.
- The existing reducer tests and the fuzz pass with no assertion weakened.

## Plan

- [ ] Enumerate the legal states and map each current boolean combination onto
      one, recording any combination currently reachable that the tag forbids.
- [ ] Introduce the tag, migrate `Reduce` case by case, and delete the
      redundant booleans.
- [ ] Sweep readers outside the reducer (`syncLifecycleTimer`,
      `syncIdleTimer`, tests) onto the tag.

## Log

### 2026-09-09

Filed from pair#171's boundary review finding BR-9.
