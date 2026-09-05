---
gate: boundary-review
issue: 187
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-04T22:25:45-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Plan bullet 4 is ticked claiming a monotonic-clock test on the detach path that does not exist
          detail: |-
            threadstore.go:513 folds detachedAt through MonotonicLastActiveAt, but nothing pins it.
            Verified by reverting to a bare `next.LastActiveAt = detachedAt`: the whole couchcore
            suite stays green. Reachable via park-at-T1 then relaunch then detach under a backward
            wall clock, which would reduce recorded recency and mis-rank the row in SelectResumableRoot.
            Fix: seed LastActiveAt = Unix(500) via UpdateExistingThread, detach with the fixture's
            FixedClock at Unix(100), assert it is still Unix(500).
          family: ticked-without-its-test
          round: 1
        - id: BR-2
          severity: Important
          title: The detach retry loop carries the new read-once invariant and no test can enter it
          detail: |-
            detach.go:114-127. Replacing `continue` with a panic leaves every test passing, including
            TestCouchDetachRetriesARevisionConflictAfterTeardown, whose name claims the branch it never
            reaches. Add threadStoreHooks.AfterGetThread fired after GetThread's withLock returns
            (threadstore.go:215); a one-shot hook that bumps the revision makes the retry reachable and
            pins the read-once detachedAt with a sequence clock. Rename the misnamed test to what it asserts.
          family: no-ordering-injection-seam
          round: 1
        - id: BR-3
          severity: Minor
          title: An uncoloured unknown-age row is brighter than AgeRecent, so "no age" reads as "most recent"
          detail: |-
            menu_render.go:352 wraps with a bare reset when ageColor returns "". Harmless to width
            accounting, but in a gradient that dims with age the default foreground is a claim of its own.
          family: absence-rendered-as-a-value
          round: 1
        - id: BR-4
          severity: Minor
          title: Only the exact zero time is guarded; anything older than ~292 years still saturates now.Sub
          detail: |-
            menu_render.go:79 and :457. A hand-edited or corrupt last_active_at of e.g. year 1500 renders
            the same 106751d the issue exists to stop. A clamp in relativeMenuAge makes the class
            unrepresentable rather than the single value.
          family: saturating-duration-arithmetic
          round: 1
        - id: BR-5
          severity: Minor
          title: Detach's revision-conflict loop has no ctx check and no attempt cap
          detail: |-
            detach.go:114. Pre-existing, but the diff now hangs state on it. A persistently-conflicting
            writer spins forever on a path the operator triggers with Alt+d.
          family: unbounded-retry-loop
          round: 1
        - id: BR-6
          severity: Minor
          title: The new band-boundary assertions duplicate the pre-existing TestAgeBandBoundaries
          detail: |-
            menu_age_test.go:84-89 pins 25h and 8d; menu_render_test.go:109 already pins both sides of
            both boundaries, which is the stronger version.
          family: redundant-boundary-assertions
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-04T22:37:28-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestDetachNeverMovesTheActivityTimeBackwards reddens when the fold is reverted to a bare assignment — verified.
          round: 2
        - id: BR-2
          disposition: addressed
          note: AfterGetThread seam added; panic on the retry branch is now reached, moving the clock into the loop reddens, test renamed.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: menu_render.go:352 still wraps with a bare reset when ageColor returns ""; no change in this round.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: relativeMenuAge still has no upper clamp; a pre-1678 last_active_at re-renders 106751d.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: detach.go:118 loop still has no ctx check and no attempt cap. Defensibly out of scope, but undecided rather than declined.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: menu_age_test.go:84-89 still duplicates TestAgeBandBoundaries.
          round: 2
      findings:
        - id: BR-7
          severity: Minor
          title: The read-once test asserts the conflict hook fired, not that the retry branch ran
          detail: |-
            detach_test.go:337. Its guard is `if !bumped`, which only proves the hook executed.
            Demonstrated by mutation: adding one extra pre-loop GetThread to Detach leaves the test
            GREEN while a panic planted on the retry branch is never reached — the loop re-read
            absorbs the bump exactly as it did before the seam existed. Assert the attempt count
            instead (reads == 3: one precondition read plus two loop reads), so the test reddens
            when it stops exercising the branch it is named for.
          family: branch-entry-unasserted
          round: 2
        - id: BR-8
          severity: Minor
          title: lessons.md and the issue Log still state the retry branch is untestable and the read-once placement unpinned
          detail: |-
            workshop/lessons.md:3391-3397 prescribes deleting the test and recording a testability gap
            because "nothing can force the retry"; the issue Log at lines 219 and 224 says the branch is
            "unreachable from tests" and the placement "is not pinned". All three are now false, and each
            is corrected only elsewhere in the same file. A reader grepping either lands on the overturned
            claim. Mark them superseded where they are stated.
          family: superseded-conclusion-stated-as-fact
          round: 2
      blocked: false
---

# Gate ledger — pair#187 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T22:25:45-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `ticked-without-its-test` Plan bullet 4 is ticked claiming a monotonic-clock test on the detach path that does not exist
  threadstore.go:513 folds detachedAt through MonotonicLastActiveAt, but nothing pins it.
  Verified by reverting to a bare `next.LastActiveAt = detachedAt`: the whole couchcore
  suite stays green. Reachable via park-at-T1 then relaunch then detach under a backward
  wall clock, which would reduce recorded recency and mis-rank the row in SelectResumableRoot.
  Fix: seed LastActiveAt = Unix(500) via UpdateExistingThread, detach with the fixture's
  FixedClock at Unix(100), assert it is still Unix(500).
- **BR-2** [Important] `no-ordering-injection-seam` The detach retry loop carries the new read-once invariant and no test can enter it
  detach.go:114-127. Replacing `continue` with a panic leaves every test passing, including
  TestCouchDetachRetriesARevisionConflictAfterTeardown, whose name claims the branch it never
  reaches. Add threadStoreHooks.AfterGetThread fired after GetThread's withLock returns
  (threadstore.go:215); a one-shot hook that bumps the revision makes the retry reachable and
  pins the read-once detachedAt with a sequence clock. Rename the misnamed test to what it asserts.
- **BR-3** [Minor] `absence-rendered-as-a-value` An uncoloured unknown-age row is brighter than AgeRecent, so "no age" reads as "most recent"
  menu_render.go:352 wraps with a bare reset when ageColor returns "". Harmless to width
  accounting, but in a gradient that dims with age the default foreground is a claim of its own.
- **BR-4** [Minor] `saturating-duration-arithmetic` Only the exact zero time is guarded; anything older than ~292 years still saturates now.Sub
  menu_render.go:79 and :457. A hand-edited or corrupt last_active_at of e.g. year 1500 renders
  the same 106751d the issue exists to stop. A clamp in relativeMenuAge makes the class
  unrepresentable rather than the single value.
- **BR-5** [Minor] `unbounded-retry-loop` Detach's revision-conflict loop has no ctx check and no attempt cap
  detach.go:114. Pre-existing, but the diff now hangs state on it. A persistently-conflicting
  writer spins forever on a path the operator triggers with Alt+d.
- **BR-6** [Minor] `redundant-boundary-assertions` The new band-boundary assertions duplicate the pre-existing TestAgeBandBoundaries
  menu_age_test.go:84-89 pins 25h and 8d; menu_render_test.go:109 already pins both sides of
  both boundaries, which is the stronger version.

## Round 2 — 2026-09-04T22:37:28-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TestDetachNeverMovesTheActivityTimeBackwards reddens when the fold is reverted to a bare assignment — verified.
- BR-2 — addressed — AfterGetThread seam added; panic on the retry branch is now reached, moving the clock into the loop reddens, test renamed.
- BR-3 — not-addressed — menu_render.go:352 still wraps with a bare reset when ageColor returns ""; no change in this round.
- BR-4 — not-addressed — relativeMenuAge still has no upper clamp; a pre-1678 last_active_at re-renders 106751d.
- BR-5 — not-addressed — detach.go:118 loop still has no ctx check and no attempt cap. Defensibly out of scope, but undecided rather than declined.
- BR-6 — not-addressed — menu_age_test.go:84-89 still duplicates TestAgeBandBoundaries.

### Raised

- **BR-7** [Minor] `branch-entry-unasserted` The read-once test asserts the conflict hook fired, not that the retry branch ran
  detach_test.go:337. Its guard is `if !bumped`, which only proves the hook executed.
  Demonstrated by mutation: adding one extra pre-loop GetThread to Detach leaves the test
  GREEN while a panic planted on the retry branch is never reached — the loop re-read
  absorbs the bump exactly as it did before the seam existed. Assert the attempt count
  instead (reads == 3: one precondition read plus two loop reads), so the test reddens
  when it stops exercising the branch it is named for.
- **BR-8** [Minor] `superseded-conclusion-stated-as-fact` lessons.md and the issue Log still state the retry branch is untestable and the read-once placement unpinned
  workshop/lessons.md:3391-3397 prescribes deleting the test and recording a testability gap
  because "nothing can force the retry"; the issue Log at lines 219 and 224 says the branch is
  "unreachable from tests" and the placement "is not pinned". All three are now false, and each
  is corrected only elsewhere in the same file. A reader grepping either lands on the overturned
  claim. Mark them superseded where they are stated.

## Open findings

- **BR-3** [Minor] `absence-rendered-as-a-value` An uncoloured unknown-age row is brighter than AgeRecent, so "no age" reads as "most recent"
- **BR-4** [Minor] `saturating-duration-arithmetic` Only the exact zero time is guarded; anything older than ~292 years still saturates now.Sub
- **BR-5** [Minor] `unbounded-retry-loop` Detach's revision-conflict loop has no ctx check and no attempt cap
- **BR-6** [Minor] `redundant-boundary-assertions` The new band-boundary assertions duplicate the pre-existing TestAgeBandBoundaries
- **BR-7** [Minor] `branch-entry-unasserted` The read-once test asserts the conflict hook fired, not that the retry branch ran
- **BR-8** [Minor] `superseded-conclusion-stated-as-fact` lessons.md and the issue Log still state the retry branch is untestable and the read-once placement unpinned
