---
id: 000696
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'bc41999383f8d01f9fc43ed02c9503987bd0fe58' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

2026-10-18: discovered while #689 label compatibility tests ran. Deterministic temporary Go overlay sets pair 0 mtime fraction .1Z and pair 1 .123456789Z. Pair 0 bundle=3109, label=372, total=3481; pair 1 bundle=3621, label=388, total=4009. Time JSON trims fractional digits and tar PAX mtime representation changes serialized lengths. With budget=3481 the second pair cannot fit on the retry. Both labels compare byte-identical to legacy alias serialization. No tracked production/test change was made for this independent defect. The same failure was observed before this controlled stimulus with budget=4007. Location: cmd/internal/historyarchive/drive_external_test.go, TestDriveMirrorBatchBudgetDefersCompletePairs.
