---
id: 000219
status: open
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
github_issue:
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
