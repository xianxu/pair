# Boundary Review — pair#199 (milestone M1)

| field | value |
|-------|-------|
| issue | 199 — pair term: own the right pane tab bar |
| repo | pair |
| issue file | workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c61f5ca0619d59e3ba795b77d6e12ad18e5517f5..0aa2b6afcb133353349942bb286ef56ff21ed5ee |
| command | sdlc milestone-close --issue 199 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T16:30:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 is a clean, genuinely faithful lift: `Reserve`/`Release`/`PaintRow`/`ChildRows` become `hostty.Reservation` methods, couch is repointed at four call sites, `grep` confirms one implementation of `\x1b[r`, and the move is proved by `couchtty/vtscreen_test.go` — a real VT-emulator end-to-end oracle over the reserved row — passing **unedited**. `go test ./cmd/internal/hostty/ ./cmd/internal/couchtty/ ./cmd/internal/artifactpath/ -count=1` is green (the two failures under the agent sandbox are the known pty restriction; both pass unsandboxed), `go vet` clean, atlas updated. Nothing here blocks correctness. What I'd fix before crossing: one deliberate behaviour change at the degenerate height that is *undeclared and unpinned* (mutation-verified — restoring the old behaviour keeps the whole suite green), the plan's Core-concepts table claiming four entities at a file that doesn't exist, and the plan-quality gate still sitting `blocked: true` with all nine PQ findings open and `estimate_hours:` empty while M1's code has already landed.

## 1. Strengths

- **The move is proved by an unedited oracle, not by assertion.** `cmd/internal/couchtty/vtscreen_test.go:135-171` drives a real VT screen: 40 scrolled lines don't touch row 8, `\x1b[r` from the child triggers re-assertion of `\x1b[1;7r`, teardown leaves a full-height scrollable screen. `git diff --stat` over that file is empty. That is exactly the "a regression there means it was a rewrite" Done-when, made checkable by something the implementor couldn't tune.
- **The deadlock was found, fixed at the right level, and explained.** `couchtty/console.go:914-922` — `bottomReservation(rows uint16)` as a plain function rather than a `c.mu`-reading method. Taking rows as an argument makes `ChildSize` and `applyLayout` obviously safe and leaves the locking discipline where it already lives (ARCH-ORDER: the illegal state is now unrepresentable at the call site, not merely avoided).
- **`Edge`'s asymmetry is named rather than assumed, and fails closed.** `hostty/reserve.go:31-38,58-67` plus `TestAnUnvalidatedTopEdgeFailsClosed`. I mutated `Paint`'s guard back to a bare `Rows == 0` and that test went red immediately — the fail-closed path is a real oracle, not decoration.
- **The hand-maintained-list sweep was complete.** All five consumers of the moved symbols were found and updated: `console.go` (4 sites), `conceptInventory`, `conceptPlans`, `#146`'s Core-concepts row (flipped to `deleted` so the contract now asserts *absence*), and `artifactpath.NonArtifactSources`. That is the `#188` class caught rather than shipped (ARCH-PURPOSE shadow-sweep: pass).
- **`hostty/reserve.go` imports only `fmt`.** Pure by construction; its tests need no fake, no fixture, no IO (ARCH-PURE: pass).

## 2. Critical findings

None.

## 3. Important findings

**I1 — `Paint` silently changed behaviour at `Rows == 1`, and no test pins the change** (`cmd/internal/hostty/reserve.go:102-105`)

`PaintRow` guarded only `hostRows == 0`, so on a 1-row host it drew the status row over the child's only line. `Paint` now guards `!usable()` (`Rows > 1`), so it draws nothing. That is the *right* change — it is precisely what `#146`'s close gate re-raised and never got fixed ("on a 1-row host `PaintRow` still draws the row, so 'simply not drawn' is wrong at `hostRows==1`") and it makes `ChildRows`' doc clause true for the first time. But it is undeclared as a deviation from the "this is a MOVE" claim, and it is unpinned.

Verified by mutation: restoring `if r.Edge != EdgeBottom || r.Rows == 0` — old height behaviour, new edge behaviour — leaves `./cmd/internal/hostty/` and `./cmd/internal/couchtty/` fully green. `TestDegenerateHeightsNeverProduceAZeroRowChild` exercises `ChildRows` and `Reserve` at rows 0 and 1 but never `Paint`.

Fix: add to that loop — `if r.Paint("x") != "" { t.Fatalf("rows=%d drew a row it has no room for", rows) }` — and record the deviation in the plan's `## Revisions` (the lift deliberately fixed the residual half of `#146`'s BR-32, rather than being byte-faithful).

**I2 — the plan's Core-concepts table declares four entities `new` at a file that does not exist** (`workshop/plans/000199-…-plan.md:114-117`)

`TabChip`, `StripModel`, `RenderedStrip`, `RenderStrip` are listed `new` at `cmd/internal/termcmd/strip.go`; `ls cmd/internal/termcmd/` confirms no `strip.go`. The contract test documents the convention this violates in its own comments: *"Rows for milestones that have not shipped carry a `planned` status, which the row loop skips — so the status column doubles as the build tracker."* This milestone is what enrolled `#199` into `conceptPlans` (`core_concepts_contract_test.go:257`), so the table is now a contract input. It passes today only because the package filter drops non-`couchtty` rows — the guard is silent, not satisfied.

Fix: flip those four rows to `planned — M3`. `Edge` and `Reservation` correctly stay `new`.

**I3 — M1's code landed while the plan-quality gate is still blocked and no estimate was recorded** (`workshop/plans/000199-…-plan-gate.md:110`, issue frontmatter line 8)

The ledger's only round is `blocked: true` with all nine PQ findings in `## Open findings`, and `estimate_hours:` is empty with no `## Estimate` block in the issue. Per AGENTS.md §2, `change-code` runs plan-quality first and only then asks for the estimate, so neither appears to have run. PQ-2 was M1-scoped and its remediation *did* ship here (the `#146` row flipped to `deleted`, `#199` registered in `conceptPlans`, both test files listed as M1 work in the plan's Revisions) — but the ledger still reports it open, which makes the remaining open set wrong right when M2 starts. The other eight are live against M2/M3/M4: PQ-3 in particular ("`TakeRowDirty` is always false in `termcmd`; the sink drops it at `run.go:661`") invalidates a row of the plan's own Integration-points table before M3 is written.

Fix: re-run the plan gate so PQ-2 disposes `addressed` and the rest carry forward accurately, and land the estimate. Not a defect in M1's code.

## 4. Minor findings

- `hostty/reserve.go:49` — `NewReservation` has **zero production callers**; couch builds the struct directly (`console.go:921`). The validating door is bypassed by the only consumer; the fail-closed methods are the real (and tested) protection. Either construct through it at the couch site or drop it and let `usable()` carry the whole contract.
- `atlas/couch.md:228` "The reserved row: a reservation, not compositing" still reads as couch-owned mechanism with no pointer to `hostty.Reservation`. That section is where a reader lands for this subject; one cross-reference line.
- `hostty/reserve.go:97-101` — `Paint`'s doc never states the caller's obligation to sanitize and column-clamp `text`. Faithful to `PaintRow`, but the mechanism/policy split just moved that obligation across a package boundary, and M3's consumer feeds it operator-supplied tab names (the plan's own ARCH-SECURE section). One doc line at the new door.
- `hostty/reserve.go:95` — `Release()` ignores its receiver entirely. Correct (teardown is unconditional by design) but reads as if it depends on the reservation; say so in the comment.
- `core_concepts_contract_test.go:45` and `:88` now assert the same three symbols absent from the same file via two rows in two plans. Harmless, explained in the comment, but it is two rows for one fact.

## 5. Test coverage notes

- The new `hostty` tests are genuine oracles — I mutated the `Paint` guard and got a red test with a precise message. They run with no IO and no double (ARCH-PURE, ARCH-MOCK: pass).
- The one real gap is I1: `Paint` at `Rows ∈ {0,1}`. That dimension of the guard is currently free.
- Not covered, and probably shouldn't be at this milestone: the `bottomReservation` deadlock class. It surfaced as a four-minute hang, not a failure. A package-level `-timeout` in CI is the cheap catch; a test isn't worth it.
- `Release()` on an unusable reservation is unpinned. Trivial, unconditional by design; noting for completeness only.

## 6. Architectural notes for upcoming work

- **ARCH-DRY: pass.** One `\x1b[r`, verified — every `SetRegion`/`ResetRegion` outside `hostty` is a qualified use of the constant, and the remaining literals are test assertions. `termcmd/run.go:1036` (`restoreTerminal` writing `hostty.ResetRegion` bare) is the one site that should become `res.Release()` when M3 wires the reservation in; today it's correct because `termcmd` doesn't reserve.
- **ARCH-CONSTRAINTS: pass for M1** — pure string building on a path that already concatenated two strings per paint. The plan's envelope claims all target M2/M3.
- **ARCH-ORDER: pass** — `Reservation` is a value carrying no state between events. The plan's transition table stands; note PQ-3 says its `TakeRowDirty` row names a seam that doesn't deliver in `termcmd`, so that row needs correcting before M3 is implemented, not during.
- **Nothing pins the new `hostty` entities.** The concept contract is package-scoped to `couchtty`, so `Edge`/`Reservation` are declared in the plan and asserted by no guard. Not worth a second contract for two symbols, but if `hostty` keeps growing shared mechanism it becomes the same drift `#146` fought.
- **The prose about this mechanism now exists in four hand-maintained places** (hostty package doc, `couchtty/reserve.go` header, `atlas/architecture.md`, `atlas/couch.md`). All four agree today. `atlas/couch.md` is the one that will go stale first.

## 7. Plan revision recommendations

Add one `## Revisions` entry to `workshop/plans/000199-pair-term-own-the-right-pane-tab-bar-plan.md`:

- **The lift was not byte-faithful at `Rows == 1`, deliberately.** `PaintRow(1, text)` drew the row over a 1-row child's only line; `Reservation.Paint` draws nothing, which is what `ChildRows`' doc has always claimed and what `#146`'s close gate flagged as false and never fixed. Recorded as a fix, not drift, and pinned by extending `TestDegenerateHeightsNeverProduceAZeroRowChild` to cover `Paint`.
- **Core-concepts table:** `TabChip` / `StripModel` / `RenderedStrip` / `RenderStrip` move from `new` to `planned — M3`. `cmd/internal/termcmd/strip.go` does not exist and the contract's status column is the build tracker.

```findings
findings:
  - id: new
    severity: Important
    family: uncovered-negative-assertion
    title: |
      Paint's new 1-row behaviour is an undeclared, unpinned change from the moved PaintRow
    detail: |
      cmd/internal/hostty/reserve.go:102 guards on !usable() (Rows > 1) where
      PaintRow guarded only hostRows == 0, so a 1-row host now draws nothing
      instead of drawing over the child's only line. This is the right change --
      it is the residual half of pair#146's BR-32 -- but the milestone claims a
      MOVE and nothing pins it. Mutation-verified: restoring the old height
      behaviour keeps ./cmd/internal/hostty and ./cmd/internal/couchtty fully
      green. TestDegenerateHeightsNeverProduceAZeroRowChild covers ChildRows and
      Reserve at rows 0 and 1 but never Paint. Add the Paint assertion to that
      loop and record the deviation in the plan's Revisions.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      Four Core-concepts rows claim `new` at cmd/internal/termcmd/strip.go, which does not exist
    detail: |
      plan lines 114-117 declare TabChip, StripModel, RenderedStrip and
      RenderStrip as `new` at a file that is not in the tree; they are M3 work.
      core_concepts_contract_test.go documents the convention this breaks --
      unshipped rows carry `planned` so the status column doubles as the build
      tracker -- and this milestone is what registered #199 in conceptPlans. The
      contract passes only because the couchtty package filter drops those rows,
      so the guard is silent rather than satisfied. Flip the four rows to
      `planned — M3`.
  - id: new
    severity: Important
    family: gate-not-cleared-before-code
    title: |
      M1 code landed with the plan-quality gate still blocked and no estimate recorded
    detail: |
      The plan-gate ledger's only round is `blocked: true` with all nine PQ
      findings open, and the issue's estimate_hours is empty with no Estimate
      block -- so change-code appears not to have cleared. PQ-2 was the only
      M1-scoped finding and its remediation did ship here, but the ledger still
      reports it open, which makes the open set wrong entering M2. PQ-3 in
      particular contradicts the plan's own Integration-points row before M3 is
      written. Re-run the gate so PQ-2 disposes `addressed`, and land the
      estimate. Not a defect in M1's code.
  - id: new
    severity: Minor
    family: validating-door-bypassed
    title: |
      NewReservation has zero production callers; couch constructs the struct directly
    detail: |
      hostty/reserve.go:49 is the validating constructor, but couchtty/console.go:921
      builds Reservation{} literally. The exported struct makes the door optional
      and the fail-closed methods are what actually protect. Either construct
      through NewReservation at the couch site or drop it and let usable() carry
      the contract alone.
  - id: new
    severity: Minor
    family: atlas-points-at-old-home
    title: |
      atlas/couch.md's "The reserved row" section still reads as couch-owned mechanism
    detail: |
      atlas/couch.md:228 describes the reservation with no pointer to
      hostty.Reservation, while architecture.md carries the new note. That
      section is where a reader lands for this subject; add one cross-reference.
  - id: new
    severity: Minor
    family: caller-obligation-undocumented
    title: |
      Paint's doc does not state the caller's obligation to sanitize and clamp its text
    detail: |
      hostty/reserve.go:97 writes caller text verbatim between SaveCursor and
      RestoreCursor. Faithful to PaintRow, and couchtty.RenderStatusRow sanitizes
      upstream, but the mechanism/policy split just moved that obligation across
      a package boundary and M3 adds a consumer whose input is operator-supplied
      tab names. One doc line at the new door (ARCH-SECURE).
```
