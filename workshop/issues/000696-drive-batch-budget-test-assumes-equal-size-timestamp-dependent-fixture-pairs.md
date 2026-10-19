---
id: 000696
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '6244d76107336aca07e08369f241b736812db07a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T00:10:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "8513c471", done: "9c9475b0"}
---

# Drive batch budget test assumes equal-size timestamp-dependent fixture pairs

## Problem

TestDriveMirrorBatchBudgetDefersCompletePairs assumes the second fixture pair fits a byte budget measured only from the first pair. During #689 label compatibility qualification it failed at the second pass with Verified=0, AlreadyRecorded=1, Deferred=1, Pending=1. This is a fixture defect, independent of the new codec: a method-free legacy label serialization equals the new label bytes exactly for both pairs.

## Spec

Make the fixture's per-pass budget admit either complete pair independently while still excluding their combined bytes. Preserve the assertions that one complete pair is deferred initially and the retry uploads exactly that pair with no new copy of the first. Include explicit unequal fractional mtimes so the fixture does not rely on random timestamp formatting.

## Done when

- The fixture deterministically exercises unequal pair sizes and both passes retain their current complete-pair/no-recopy assertions.
- The targeted test passes against production behavior without changing production limits, transfer semantics or retry policy.

## Plan

- [x] Give only this scenario explicit source mtimes (.1 and .123456789 seconds), and reproduce the second-pass failure with the original pair-0 budget.
- [x] Measure both complete encrypted bundle/label pairs, use their maximum as the per-pass budget, and assert unequal sizes and exclusion of the combined pair size. Keep the first-pass assertions and require exactly four total copies/two batches after retry (ARCH-PURPOSE); reuse the existing fixture and stateful rclone double (ARCH-DRY).
- [ ] Run targeted repetitions, the recorded suite, and quick-flow close/landing with fleet-blocking priority. No production or persisted-format changes; all files remain under the existing temporary fixture lifetime.

## Log


- 2026-10-19: closed — Controlled unequal fractional-mtime fixture failed with the old 3481-byte pair-0 budget; corrected measured-max budget passed 20 targeted repetitions (30.124s), retaining first-pass deferral and exact four-copy/two-batch retry assertions. Test-only change adds no architectural surface. Recorded full suite attempt 60b3024ec4f9c3c5d9ef89f0b35ecf67 is queued on pushed head 29094fc81; review overlap is explicitly permitted by tl-worker. The final Plan item is still unchecked because the full-suite pass and landing remain required, not claimed complete.; review verdict: SHIP
2026-10-18: discovered while #689 label compatibility tests ran. Deterministic temporary Go overlay sets pair 0 mtime fraction .1Z and pair 1 .123456789Z. Pair 0 bundle=3109, label=372, total=3481; pair 1 bundle=3621, label=388, total=4009. Time JSON trims fractional digits and tar PAX mtime representation changes serialized lengths. With budget=3481 the second pair cannot fit on the retry. Both labels compare byte-identical to legacy alias serialization. No tracked production/test change was made for this independent defect. The same failure was observed before this controlled stimulus with budget=4007. Location: cmd/internal/historyarchive/drive_external_test.go, TestDriveMirrorBatchBudgetDefersCompletePairs.

2026-10-19: #686 authoritative integration suite4031a2baeee56a290f445bc26edb6e43 failed only this test on main-derived head7e7358f0c6882e0eb3eb3bc73a1ef12c993ae318. Retained log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/4031a2baeee56a290f445bc26edb6e43.log (sha256 prefix72c05135cf21), exit2 in12m36s. Hub approved independent quick-flow repair and fleet-blocking landing priority; #686 is parked cleanly until this lands. Prefer measuring both actual pair sizes over forcing equal archive bytes: unequal sizes are valid production input and make the budget assertion meaningful.

2026-10-19 red/green: explicit fractional mtimes reproduced the original failure deterministically, with Verified0/AlreadyRecorded1/Deferred1/Pending1 and3481-byte budget. Measuring both pairs and choosing their maximum passed20 unchanged targeted repetitions in30.124s. The test asserts unequal pair sizes, budget below their combined bytes, original first-pass deferral, and exactly four copies/two batches after the retry. No production code, format, timeout or transfer assertion was weakened. All fixture writes remain in t.TempDir; optional mtimes leave other helper callers unchanged. Recorded full suite and close review follow.
