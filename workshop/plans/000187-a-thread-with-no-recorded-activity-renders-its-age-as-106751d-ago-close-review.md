# Boundary Review — pair#187 (whole-issue close)

| field | value |
|-------|-------|
| issue | 187 — A thread with no recorded activity renders its age as 106751d ago |
| repo | pair |
| issue file | workshop/issues/000187-a-thread-with-no-recorded-activity-renders-its-age-as-106751d-ago.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4d9123a7cd9fd8ccc7c01663ee37401cdf1453d8..dd55182a30135a12a12df0126d43ab2f8268b382 |
| command | sdlc close --issue 187 |
| reviewer | claude |
| timestamp | 2026-09-04T22:25:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diff fixes both defects the Spec names and fixes them in the right places: `hasRecordedActivity` is a single predicate asked by both readers (so the row text and the row colour cannot drift), `RetireIncarnation` takes the time as a parameter rather than reaching for a clock, and the clock read sits once outside the retry loop. I verified all three fixes by reverting them in a scratch copy: dropping the `withMenuAge` guard reproduces the operator's exact string (`"detached · 106751d ago"`), dropping the `AgeBandFor` guard makes absence paint as `AgeOld`, and dropping the `LastActiveAt` write leaves detach with no recorded activity — every one goes red. The shadow-sweep is genuinely complete: `startup.go`'s ranking is audited and documented instead of "fixed", `threadrecord/lifecycle.go:107` already guards zero, and no other surface renders `LastActiveAt`. What holds it back from SHIP is test coverage the Plan claims but does not have: bullet 4 is ticked `[x]` naming a test that `MonotonicLastActiveAt` still refuses a backward clock on the detach path, and no such test exists — I replaced the monotonic fold with a raw assignment and the whole `couchcore` suite stayed green. Both Important findings are about tests, not behaviour; the shipped code is correct.

Full suite: green except `ptychild`/`fork-exec ps: operation not permitted` failures across `couchcore`, `couchcmd`, `couchtty`, `cmd/couch` and `pair-go`, which are this environment blocking process spawning, not the diff. `gofmt -l` clean, `go vet ./cmd/...` clean.

### 1. Strengths

- **The predicate is shared, and that pays off observably.** `menu_render.go:73` is asked by both `AgeBandFor` (line 76) and `withMenuAge` (line 450), so the text and the colour cannot disagree about what "no age" means. This was the plan-quality gate's PQ-4 and it landed as specified (ARCH-DRY).
- **`ageColor(AgeUnknown)` returns `""` rather than a fourth dim escape** (`menu_render.go:472`). This is the difference between a guard and a *observable* guard — with `AgeUnknown` mapped to `240` the band would have been indistinguishable from `AgeOld` and its test could only have asserted the enum against itself. The finding was disposed correctly, not just marked addressed.
- **The adversarial pair is carried in every test.** `time.Time{}` ⇒ absent, `time.Unix(0,0)` ⇒ a real 19675-day age (`menu_age_test.go:53`, `:78`). A guard written as "very old implies absent" passes the reported bug and fails these. That is the right oracle.
- **`RetireIncarnation` takes `detachedAt` instead of reading a clock** (`threadstore.go:494`), keeping the store pure-ish and the clock read at the `Couch` boundary — and the read-once placement at `detach.go:113` is reasoned at the call site (ARCH-PURE).
- **The Log is honest about what could not be tested.** The retry-time test was written, found vacuous, and deleted with the reason recorded. I confirmed the claim independently: replacing `continue` with a `panic` at `detach.go:124` leaves every test — including `TestCouchDetachRetriesARevisionConflictAfterTeardown` — passing. The retry branch really is unreachable from the current fixtures, and the Log says so rather than shipping a green test that asserts nothing.
- **`startup.go:48` is a comment, not a change.** The ranking claim checks out: zero is `Before` everything so `After` is false, and `ProjectActionableThreads` does sort by `(RepoScope, Tag)` at `actionableinventory.go:222`. Correct, and now correct *for a written reason*.

### 2. Critical findings

None.

### 3. Important findings

**I-1 — `threadstore.go:513`: the monotonic fold on the detach path is unpinned, and the Plan ticks a bullet claiming its test.** Issue `## Plan` bullet 4 says "Test in `detach_test.go`: … `MonotonicLastActiveAt` still refuses a backward clock" and is checked. `TestDetachRecordsWhenTheThreadWasLastActive` only asserts that a fixture starting from zero ends at `Unix(100)` — which a raw `next.LastActiveAt = detachedAt` satisfies just as well. Verified by reverting: with the fold replaced by a bare assignment, `go test ./cmd/internal/couchcore/ -run 'Detach|Retire|Leave|Monotonic|LastActive'` is `ok`. The reachable scenario is park-at-T1 → relaunch → detach under a backward wall clock (NTP step), which would *reduce* recorded recency and mis-rank the row in `SelectResumableRoot`. Fix (~12 lines): a subtest that seeds `LastActiveAt = time.Unix(500,0).UTC()` via `UpdateExistingThread`, detaches with the fixture's `FixedClock{T: time.Unix(100,0)}`, and asserts the value is still `Unix(500)`.

**I-2 — `detach.go:114-127`: the retry loop has no seam to inject the ordering it exists to handle** (ARCH-ORDER). The loop is the only place the new `detachedAt` invariant lives, and no test can enter its conflict branch — confirmed by panic-mutation, which also shows `TestCouchDetachRetriesARevisionConflictAfterTeardown` (`detach_test.go:177`) does not exercise what its name says; the loop's re-read absorbs the injected edit before any CAS. A green run there is a sample of size one. The seam is cheap and the store already has the machinery: add `threadStoreHooks.AfterGetThread func(ThreadAddress)` fired in `GetThread` *after* `withLock` returns (`threadstore.go:215`, so the hook is not holding the lock), then a one-shot hook that bumps the revision makes the retry reachable. That single hook pins three things at once: the retry branch itself, the read-once `detachedAt` (with a sequence clock), and the true meaning of the misnamed test — which should be renamed to what it asserts.

### 4. Minor findings

- `menu_render.go:352` — with `ageColor` returning `""`, an unknown-age row renders at the terminal's default foreground, which is *brighter* than `AgeRecent`'s `250`; in a gradient that dims with age, "no age" now reads as "more recent than recent". Also emits a bare `\x1b[0m` with no preceding set (harmless — `styledMenuWidth` strips it). Consider making the wrap conditional on a non-empty escape.
- `menu_render.go:79` — only the exact zero is guarded. Any `LastActiveAt` older than ~292 years still saturates `now.Sub` and re-renders `106751d ago`; reachable only from a hand-edited or corrupt record, but a clamp in `relativeMenuAge` would make the class unrepresentable rather than the one value (ARCH-SECURE: the persisted record is input this process did not necessarily produce *at this version*).
- `menu_age_test.go:84-89` — the 24h/7d boundary assertions duplicate `TestAgeBandBoundaries` (`menu_render_test.go:109`), which already pins both sides of both boundaries. Harmless, but the new copy pins only the outer side.
- `detach.go:114` — `for {}` with `continue` on conflict has no `ctx.Err()` check and no attempt cap (pre-existing; the diff now adds state to it). ARCH-CONSTRAINTS: a persistently-conflicting writer spins unbounded on a path the operator triggers with `Alt+d`.
- `readme_test.go:221-228` — the comment's second-to-last line runs well past the file's prevailing width.

### 5. Test coverage notes

Three of the four behavioural changes are pinned by tests that fail without them (verified by revert, not by reading): the row-text guard, the colour guard, and the detach write. The row-level assertion (`rootStateText`, not just the helper) is the right altitude and is what the Spec asked for. Two gaps: the monotonic fold (I-1) and the retry ordering (I-2). Both are cheap. Nothing in the diff is a mock reasserting the implementation — `couchcore`'s tests run against a real file-backed store in a temp namespace with stateful `FakeProcOps` / `FakeThreadArtifactCollisionChecker` and an injected `FixedClock` (ARCH-MOCK: pass), and `couchtty`'s are pure with `now` as a parameter (ARCH-PURE: pass).

### 6. Architectural notes

- **ARCH-DRY** — pass. One predicate, two readers, no copy-paste. The only duplication is in tests (Minor above).
- **ARCH-PURE** — pass. The decision moved *up* out of the formatter (`withMenuAge` decides, `relativeMenuAge` stays a total formatter never handed a value it cannot format) — this deviation from the plan is the better shape and is recorded rather than ticked away.
- **ARCH-PURPOSE** — pass. Shadow-sweep: `rootStateText` ✔, `ageColor` ✔, `SelectResumableRoot` audited-and-documented ✔, `threadrecord/lifecycle.go:107` already guarded ✔, `launcher.FormatAge` is a different domain (file mtimes, never zero) ✔. No consumer left restating the model by hand.
- **ARCH-MOCK** — pass, with the seam gap folded into I-2.
- **ARCH-CONSTRAINTS** — pass on the render path (O(1) per row, no new allocation in the hot loop); flag on the unbounded retry (Minor).
- **ARCH-SECURE** — pass, with the out-of-range-timestamp Minor.
- **ARCH-ORDER** — flag, I-2. Worth carrying forward: `Couch.Threads` is a concrete `*ThreadStore`, so *no* couch operation's contention behaviour is currently injectable. `AfterGetThread` would unlock that for `MarkIncarnationUnknown`'s loop too, which has the same shape.

### 7. Plan revision recommendations

One `## Revisions` entry on the issue file. Either write the test in I-1 and leave the bullet ticked, or record that bullet 4's third sub-test (`MonotonicLastActiveAt` refuses a backward clock) was not written — currently the deleted retry-time sub-test is disclosed in the Log while this one is silently absent behind the same `[x]`. The bullet-2 deviation (`withMenuAge` vs `relativeMenuAge` returning `(string, bool)`) is already stated in prose and needs nothing further.

```findings
findings:
  - id: new
    severity: Important
    family: ticked-without-its-test
    title: |
      Plan bullet 4 is ticked claiming a monotonic-clock test on the detach path that does not exist
    detail: |
      threadstore.go:513 folds detachedAt through MonotonicLastActiveAt, but nothing pins it.
      Verified by reverting to a bare `next.LastActiveAt = detachedAt`: the whole couchcore
      suite stays green. Reachable via park-at-T1 then relaunch then detach under a backward
      wall clock, which would reduce recorded recency and mis-rank the row in SelectResumableRoot.
      Fix: seed LastActiveAt = Unix(500) via UpdateExistingThread, detach with the fixture's
      FixedClock at Unix(100), assert it is still Unix(500).
  - id: new
    severity: Important
    family: no-ordering-injection-seam
    title: |
      The detach retry loop carries the new read-once invariant and no test can enter it
    detail: |
      detach.go:114-127. Replacing `continue` with a panic leaves every test passing, including
      TestCouchDetachRetriesARevisionConflictAfterTeardown, whose name claims the branch it never
      reaches. Add threadStoreHooks.AfterGetThread fired after GetThread's withLock returns
      (threadstore.go:215); a one-shot hook that bumps the revision makes the retry reachable and
      pins the read-once detachedAt with a sequence clock. Rename the misnamed test to what it asserts.
  - id: new
    severity: Minor
    family: absence-rendered-as-a-value
    title: |
      An uncoloured unknown-age row is brighter than AgeRecent, so "no age" reads as "most recent"
    detail: |
      menu_render.go:352 wraps with a bare reset when ageColor returns "". Harmless to width
      accounting, but in a gradient that dims with age the default foreground is a claim of its own.
  - id: new
    severity: Minor
    family: saturating-duration-arithmetic
    title: |
      Only the exact zero time is guarded; anything older than ~292 years still saturates now.Sub
    detail: |
      menu_render.go:79 and :457. A hand-edited or corrupt last_active_at of e.g. year 1500 renders
      the same 106751d the issue exists to stop. A clamp in relativeMenuAge makes the class
      unrepresentable rather than the single value.
  - id: new
    severity: Minor
    family: unbounded-retry-loop
    title: |
      Detach's revision-conflict loop has no ctx check and no attempt cap
    detail: |
      detach.go:114. Pre-existing, but the diff now hangs state on it. A persistently-conflicting
      writer spins forever on a path the operator triggers with Alt+d.
  - id: new
    severity: Minor
    family: redundant-boundary-assertions
    title: |
      The new band-boundary assertions duplicate the pre-existing TestAgeBandBoundaries
    detail: |
      menu_age_test.go:84-89 pins 25h and 8d; menu_render_test.go:109 already pins both sides of
      both boundaries, which is the stronger version.
```

---

## Re-review — 2026-09-04T22:37:28-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 187 — A thread with no recorded activity renders its age as 106751d ago |
| repo | pair |
| issue file | workshop/issues/000187-a-thread-with-no-recorded-activity-renders-its-age-as-106751d-ago.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4d9123a7cd9fd8ccc7c01663ee37401cdf1453d8..86d845137cf6099335a38aca64ac6f7235073005 |
| command | sdlc close --issue 187 |
| reviewer | claude |
| timestamp | 2026-09-04T22:37:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both round-1 Importants are genuinely closed, and I verified each by mutation rather than by reading the commit message: reverting `MonotonicLastActiveAt` to a bare assignment reddens `TestDetachNeverMovesTheActivityTimeBackwards` (`LastActiveAt = 00:01:40, want 00:08:20`), replacing the retry `continue` with a panic now *reaches the panic* from `TestDetachRecordsOneActivityTimeHoweverManyAttemptsItTakes`, and moving `c.Clock.Now()` inside the loop reddens the same test (`want the FIRST clock reading`). The `AfterGetThread` seam is the right shape — it reuses the existing unexported `threadStoreHooks`, fires outside `withLock`, and is nil in production (`NewThreadStore` builds zero hooks; `newThreadStoreWithHooks` is test-only and already registered as such in `deadsymbols_test.go:37`). `gofmt -l` clean, `go vet ./cmd/...` clean, `-race` clean on the detach set; the only red in the suite is the known `ptychild: operation not permitted` environment class. What holds this back from SHIP is not the shipped behaviour: four round-1 Minors are still open with no recorded decision, and the new test — while correct today — cannot detect its own regression to vacuity, which is the exact failure mode this issue spent two rounds on.

### 1. Strengths

- **The seam answers the class, not the instance.** `threadStoreHooks.AfterGetThread` (`threadstore.go:88-96`, fired at `:224`) is the one window a caller's read-then-CAS is open, and it unlocks every `GetThread`-then-CAS loop in the store, not just detach's. Reusing the existing hook struct rather than inventing machinery is the ARCH-DRY answer.
- **The comment on the hook names why every *other* hook failed.** `threadstore.go:91-95` records that `AfterJournal`/`AfterTarget` fire inside the commit, after the revision check — so the next person doesn't re-derive it.
- **The misnamed test was renamed to what it asserts** (`detach_test.go:194`, `...AbsorbsAConcurrentWriteBeforeItsLoop`), with the old name preserved in the comment as the reason the branch went untested. That is the honest disposal, not a cosmetic one.
- **`Detach`'s precondition guard was widened with the dependency** (`detach.go:57`). A new `c.Clock` read without a declared precondition segfaults where the message says nothing; the guard now names it.
- **The shadow-sweep still holds.** I re-ran it independently: every `LastActiveAt` consumer is accounted for — `thread.go:102,155` copy, `startup.go:55` audited-and-commented, `actionableinventory.go:219` projects, `threadrecord/lifecycle.go:107` already guards zero, `menu_render.go:297,299,352` all route through `hasRecordedActivity`. No hand-maintained restatement left (ARCH-PURPOSE).

### 2. Critical findings

None.

### 3. Important findings

None. BR-1 and BR-2 are addressed and verified by revert.

### 4. Minor findings

- `detach_test.go:337` — the `!bumped` guard asserts the *hook fired*, not that the *retry ran*. Demonstrated: with one extra pre-loop `GetThread` added to `detach.go`, the test stays green **and** a panic planted on the retry branch is never reached. Pin the attempt count (`reads == 3`: one precondition read plus two loop reads) so a refactor that re-absorbs the conflict reddens instead of going quiet. New family `branch-entry-unasserted`.
- `workshop/lessons.md:3391-3397` and issue Log lines 219/224 — the superseded conclusion ("nothing can force the retry"; "it is not pinned, and a future change that moves it into the loop will not be caught") is retained as a present-tense factual claim. Both are now false and both are corrected only *elsewhere* in the same file. New family `superseded-conclusion-stated-as-fact`.
- BR-3 / BR-4 / BR-5 / BR-6 remain open and unfixed — see dispositions below.

### 5. Test coverage notes

All four behavioural changes now fail without their fix, verified by revert rather than by reading: the row-text guard, the colour guard, the detach write, the monotonic fold, and the read-once clock placement. `TestDetachRecordsOneActivityTimeHoweverManyAttemptsItTakes` is the strongest addition — one hook pins the retry branch, the read-once invariant, and (by renaming) the true meaning of the test that used to claim the branch. The one oracle weakness is above. Nothing in the diff is a mock reasserting the implementation.

### 6. Architectural notes

- **ARCH-DRY** — pass. One predicate, two readers; the seam reuses `threadStoreHooks` rather than adding a parallel mechanism.
- **ARCH-PURE** — pass. `RetireIncarnation` takes `detachedAt` instead of reaching for a clock; `relativeMenuAge` stays a total formatter and `withMenuAge` owns the decision; `couchtty` tests run with `now` as a parameter, no IO.
- **ARCH-PURPOSE** — pass. The round-2 fix is the class (a seam any conflict loop can use), not the site.
- **ARCH-MOCK** — pass. Real file-backed store in a temp namespace, stateful `FakeProcOps`/`FakeThreadArtifactCollisionChecker`, injected clocks; production and test flow share the same boundary.
- **ARCH-CONSTRAINTS** — flag, unchanged: BR-5's unbounded retry (no `ctx.Err()`, no attempt cap) on an `Alt+d` path.
- **ARCH-SECURE** — flag, unchanged: BR-4. The hook itself is safe — unexported type, unexported constructor, nil in production.
- **ARCH-ORDER** — largely resolved. The interleaving is now injectable and reproducible. Remaining: the oracle doesn't assert the interleaving occurred (Minor above), and `MarkIncarnationUnknown` (`threadstore.go:732`) still has the identical read-then-CAS loop with no test entering its conflict branch — the new hook makes that cheap, and it is the natural next consumer.

### 7. Plan revision recommendations

- The issue needs a `## Revisions` entry (or an edit in place) marking the "The original attempt…" paragraph's conclusions as **superseded**: line 219's "the retry branch in `detach.go` is unreachable from tests" and line 224's "it is not pinned" now read as current fact and are both false. The correcting paragraph above them doesn't neutralise a claim stated as fact below.
- `workshop/lessons.md:3391` needs the same treatment — its prescription ("Delete the test and record the testability gap") was overturned within this issue by the entry three bullets later.
- Plan bullets themselves now match the code; no other revision needed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestDetachNeverMovesTheActivityTimeBackwards reddens when the fold is reverted to a bare assignment — verified.
  - id: BR-2
    disposition: addressed
    note: |
      AfterGetThread seam added; panic on the retry branch is now reached, moving the clock into the loop reddens, test renamed.
  - id: BR-3
    disposition: not-addressed
    note: |
      menu_render.go:352 still wraps with a bare reset when ageColor returns ""; no change in this round.
  - id: BR-4
    disposition: not-addressed
    note: |
      relativeMenuAge still has no upper clamp; a pre-1678 last_active_at re-renders 106751d.
  - id: BR-5
    disposition: not-addressed
    note: |
      detach.go:118 loop still has no ctx check and no attempt cap. Defensibly out of scope, but undecided rather than declined.
  - id: BR-6
    disposition: not-addressed
    note: |
      menu_age_test.go:84-89 still duplicates TestAgeBandBoundaries.
findings:
  - id: new
    severity: Minor
    family: branch-entry-unasserted
    title: |
      The read-once test asserts the conflict hook fired, not that the retry branch ran
    detail: |
      detach_test.go:337. Its guard is `if !bumped`, which only proves the hook executed.
      Demonstrated by mutation: adding one extra pre-loop GetThread to Detach leaves the test
      GREEN while a panic planted on the retry branch is never reached — the loop re-read
      absorbs the bump exactly as it did before the seam existed. Assert the attempt count
      instead (reads == 3: one precondition read plus two loop reads), so the test reddens
      when it stops exercising the branch it is named for.
  - id: new
    severity: Minor
    family: superseded-conclusion-stated-as-fact
    title: |
      lessons.md and the issue Log still state the retry branch is untestable and the read-once placement unpinned
    detail: |
      workshop/lessons.md:3391-3397 prescribes deleting the test and recording a testability gap
      because "nothing can force the retry"; the issue Log at lines 219 and 224 says the branch is
      "unreachable from tests" and the placement "is not pinned". All three are now false, and each
      is corrected only elsewhere in the same file. A reader grepping either lands on the overturned
      claim. Mark them superseded where they are stated.
```
