# Boundary Review — pair#172 (milestone M1)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..1c895d4fbdd2f4e05e146ed6dc9d9cf3bfbe7936 |
| command | sdlc milestone-close --issue 172 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-05T13:19:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1's core claim — that click geometry must come out of the pass that already clips, not a second derivation — is genuinely delivered and I proved it by mutation: replacing the chip span with a model-derived `start + Width(label)` reddens `TestChipSpansMatchTheDrawnRowAtEveryWidth` at width 24, and deleting the notice-insert re-base reddens `TestPointToActorSpansEveryLineOfAnActor` on exactly the off-by-one the plan predicted. The SGR parser move is byte-faithful (I diffed all four functions line-for-line against the base) and termcmd's own suite passed unedited, which is the regression the plan asked for. What holds SHIP back is the other half of the same Done-when bullets: the *clipped* and *scrolled* cases are asserted one-directionally or not at all — a mutation that silently drops the span of every truncated chip, and one that computes extents from the inventory index instead of the drawn line, both pass the entire suite — and `clampExtents` (the whole clipped-list path) is dead to the test suite. Plus the plan's Core-concepts table now claims `hostty.MouseClickTracking` is `new` when it does not exist in any file.

## 1. Strengths

- **The DRY claim is real, not asserted.** `reserve.go:158-160` records the span inside the same `appendText` that clips, and `reserve_mouse_test.go:27-30` catches the naive re-derivation. Mutation-verified in a scratch copy.
- **The re-base lives at the right layer and is pinned.** `menu_render.go:154-160` re-bases extents in `RenderMenuView` after the index-1 notice insert, and `menu_extent_test.go` runs the fixture both with and without a notice. Removing the loop fails with `row 5 ("    second message") mapped to two, want one` — the precise failure the plan named.
- **The parser move is a move.** `mouseinput.Parse`/`ParsePrefix`/`Find`/`IsPrefix` are character-identical to the originals; `rename_input.go:182` was swept to the shared `Terminators` constant, closing the drift `#127` came from.
- **Both maps are total by construction, returning a value rather than pushing a branch to the call site** (`reserve.go:114-121`, `menu_render.go:67-74`) — and `ChipSpan` carries a `ThreadAddress`, not an index, so a stale span degrades to "no longer attached" instead of switching to the wrong actor.
- **The plan registration actually asserts.** `go test -run Concept -v` shows six new `TestCoreConceptsContract/PURE/...` subtests for #172 rows, so PQ-6's fix is not decorative.

## 2. Critical findings

**`workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md:182` — Core-concepts row claims a symbol that exists nowhere.** The Integration points table declares `hostty.MouseClickTracking` at `cmd/internal/hostty/control.go` with status **`new`**. `grep -rn Mouse cmd/internal/hostty/` returns nothing — the whole package has no mouse surface; this is M2 Task 8's work. The plan's own Task 5 sets the convention ("Rows for unshipped milestones carry status `planned`") and the contract test's `planned` skip (`core_concepts_contract_test.go:299-305`) is what makes the status column the build tracker. Because the row's declared path is outside `conceptPackage`, nothing catches it. Fix: change the status to `planned — M2`, and add a `## Revisions` entry.

## 3. Important findings

**`cmd/internal/couchtty/reserve_mouse_test.go:46-49` — the clipped case is asserted in one direction only.** The test bounds spans from above (`chip.End > width` → fail) but the only lower-bound assertion is `len(got.Chips) > len(m.Actors)`, which with three distinct threads can never fire. I mutated `reserve.go:159` to `used > start && used-start >= textwidth.Width(label)` — i.e. every *truncated* chip silently loses its span and becomes unclickable — and the full couchtty suite stayed green. The Done-when asks for "a width narrow enough to clip chips and one narrow enough to drop them"; the drop half is covered incidentally by the `Start >= End` guard, the clip half is not. Fix: assert the expected span count per width (e.g. at width 24 all three actors still draw ⇒ three spans; at width 6 exactly one), and assert `ColumnToActor` inside a *truncated* chip's columns.

**`cmd/internal/couchtty/menu_render.go:173-186` — `clampExtents` and the scrolled list are dead to the test suite.** Replacing the whole function body with `return extents` leaves the package green (only the pre-existing pty failure remains), and re-indexing extents off the inventory row instead of `len(lines)` also passes. The fixture is 2 actors × 14 rows, so neither `start > 0` nor `len(lines) < extent.End` is ever reached. Plan Task 4 Step 4 says "Add the scrolled case: mapping is against what is DRAWN, not the inventory index" and the Done-when says "on a scrolled or clipped list" — neither is delivered. Fix: one fixture with enough actors/attention lines to force `rowBudget` scrolling, and one with `height` small enough that the last actor is cut, asserting the cut rows map to nobody.

**`cmd/internal/mouseinput/` — new package ships with no test file.** Plan Task 1's Files list names `mouseinput_test.go`; it does not exist (`go test` reports `[no test files]`). termcmd's suite is a fair regression for the *moved* code, but it only reaches `Parse` (`run_test.go:626`) and `Find`/`IsPrefix` indirectly; `ParsePrefix`, `MaxReport`, `WheelUp`, `WheelDown` and `Terminators` have zero direct coverage. The plan's Core-concepts row `mouseinput.Event / mouseinput.Find | new | PURE` would have been caught by the contract's `assertDirectTest`, but the row is filtered out by `conceptPackage = "cmd/internal/couchtty/"` — so registering the plan pinned 6 of the 8 non-`planned` rows and silently dropped the two living outside couchtty. Same rule as PQ-6/PQ-14 (`guard-not-registered`): the guard's input is scoped, and Task 5's claim that "its Core-concepts rows are pinned" is true of six of them.

**`cmd/internal/couchtty/menu_render.go:40-46, 57-58` vs `:130-131` — two coordinate conventions on one struct, one documented.** `RenderMenuView`'s doc says "Cursor coordinates are 1-based terminal cells in Body"; the new `Extents` field on the same struct is 0-based rows of Body, and neither `ActorExtent` nor `PointToActor` says so. `ChipSpan` has the same gap: `Start`/`End` are 0-based display columns, while the SGR report that will index them (`mouseinput.Event.X`) is 1-based, and the row is painted at terminal column 1 (`reserve.go` → `hostty.MoveTo(rows, 1)`). This is a newly-introduced internal surface that M2/M3 consume; an off-by-one here is the least visible way to be wrong, which is the plan's own phrase. Fix: state the base and the origin in the `ChipSpan`, `ActorExtent` and `PointToActor` doc comments.

**`cmd/internal/artifactpath/deadsymbols_test.go:47` — the renumber sweep left a sibling reference in the same comment block.** Six map entries moved `pair#173 → pair#192`, but the path two lines above still reads `workshop/issues/000173-disposition-six-production-symbols-reachable-only-from-tests.md`, which does not exist (the file is `000192-…`). The comment's own next sentence is "A deferral whose ticket does not exist is not deferred, it is exempted" — the exact invariant the stale path breaks. `grep -rn "000173-disposition"` returns this one site, so the enumeration is a one-line sweep.

## 4. Minor findings

- `cmd/internal/couchtty/reserve.go:86-93` — the `RenderStatusRow` doc block (untrusted labels, why control bytes are stripped) is now orphaned onto `ChipSpan`; `RenderStatusRow` at `:126` has no doc comment at all, and godoc will attribute the sanitize rationale to the wrong type.
- `cmd/internal/mouseinput/mouseinput.go:36-37` — `WheelUp`/`WheelDown` have zero call sites while `termcmd/run.go:451,457` still tests `event.Button == 64` / `== 65`. The promotion created a second source of truth for the wheel codes instead of removing one (ARCH-DRY).
- `cmd/internal/termcmd/run.go:579,585` — `mousePressEvent` (the alias) and `parseSGRMousePressPrefix` have no callers after the move; `parseSGRMousePressPrefix` was only ever reached through the old `findSGRMousePress`. Residue of a move, and `deadSymbolScope` is `cmd/internal/couchcore` so the guard cannot see it.
- `cmd/internal/mouseinput/mouseinput.go:31` — `MaxReport` is declared with the comment that a holder "must bound the wait", and nothing bounds anything yet (M2 Task 7 Step 3b owns it). Fine as a landing spot; worth a `// consumed in M2` marker so it doesn't read as an enforced invariant.
- `atlas/couch.md` — "A caller that holds on `IsPrefix` must bound the wait (`MaxReport`)" is written in the present tense for a rule no code enforces at this boundary.
- `menu_render.go:437-441` — the extent-run merge is unobservable through `PointToActor` (removing it leaves the suite green, since per-line extents still resolve correctly). The "one extent per actor" shape of the exported `Extents` field is therefore unpinned; a consumer that highlights an actor's whole block would break silently.

## 5. Test coverage notes

Verified by mutation in a scratch copy (repo left clean): notice re-base **caught**; model-derived spans **caught**; all-spans-dropped **caught** (`TestColumnToActorIsTotal`). Not caught: truncated-chip span dropped, `clampExtents` neutered, extents indexed off the inventory row, run-merge removed. Environment note — `TestNotificationPTYConformance`, `termcmd`'s two mux tests and `wrapcmd`'s harness-TTY tests fail with `operation not permitted` from `pty.Open()` in this shell; those are the known pty-child environment failures, not regressions from this window, and I confirmed they fail identically with the diff's changes neutered.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — pass on the headline (one render pass, one parser, mutation-verified), flag on `WheelUp`/`WheelDown` vs the surviving literals and on the two dead shims.
- **ARCH-PURE** — pass. Both entities are pure, `assertPureSource` holds, and every new test runs without a terminal, a child or a clock.
- **ARCH-PURPOSE** — flag. Three M1 checklist commitments landed as their easy subset: the scrolled case (Task 4 Step 4), `mouseinput_test.go` (Task 1), and the renumber sweep's last sibling. Each names a class the fix stopped short of.
- **ARCH-MOCK** — N/A this window; no external binary or service is touched, and the terminal surface is reached only through pure string functions.
- **ARCH-CONSTRAINTS** — pass. Both hit-tests are O(actors) linear scans on a human-click path, and nothing was added to the keystroke path. `paintNow` now allocates a `[]ChipSpan` per repaint and throws it away, which is negligible but points at the next item.
- **ARCH-SECURE** — pass. Spans are measured *after* `sanitize`, so untrusted label text cannot shift the geometry; `Parse` fails closed on a missing terminator and on numeric overflow (`Sscanf` range error). It does accept negative coordinates the wire format cannot emit — both maps are total so that degrades to "nobody", and it is faithful to the pre-move behavior, so it is not drift. Worth parsing to an unsigned typed value when M2 puts this on the routing path.
- **ARCH-ORDER** — the one to carry into M2/M3. The rendered geometry is state held between two external events (paint, then click), and today `console.go:1007` discards `.Chips` entirely. M3 must decide explicitly whether a click resolves against the row *currently on screen* or a fresh render, and what happens when the model changed in between (actor vanished, chips reflowed on a resize, the terminal narrowed). The `ThreadAddress`-not-index choice already makes the failure mode benign; say so in the plan and name resize as the event most likely to be mishandled, rather than leaving it to fall out.

## 7. Plan revision recommendations

- `### 2026-09-05 — M1 close: hostty.MouseClickTracking is not shipped`. Flip the Integration points row (`:182`) to `planned — M2`, and record that the row was outside `conceptPackage` so the contract could not catch the wrong status — the `guard-not-registered` family, third instance.
- `### 2026-09-05 — M1 close: Task 1 and Task 4 delivered a subset`. Record that `mouseinput_test.go` was not created and that Task 4 Step 4's scrolled case was not added, so the Done-when's "on a scrolled or clipped list" and "a width narrow enough to clip chips" remain unasserted; say which milestone now owns them.
- In the Done-when → Task map (`:246`), the row names `TestChipSpansMatchTheDrawnRowWhenClipped`; the shipped test is `TestChipSpansMatchTheDrawnRowAtEveryWidth`. Update the name so the map stays greppable.

```findings
findings:
  - id: new
    severity: Critical
    family: plan-table-claims-unshipped-code
    title: |
      Core-concepts row declares hostty.MouseClickTracking `new` when it exists nowhere
    detail: |
      workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md:182 lists
      `hostty.MouseClickTracking` at cmd/internal/hostty/control.go with status `new`.
      grep -rn Mouse cmd/internal/hostty/ returns nothing; it is M2 Task 8's work. The
      plan's own Task 5 requires unshipped rows to carry `planned`, and the contract
      test's planned-skip is what makes the status column the build tracker. The row's
      path is outside conceptPackage, so nothing catches the wrong status. Flip it to
      `planned — M2` plus a Revisions entry.
  - id: new
    severity: Important
    family: onedirectional-geometry-assertion
    title: |
      The clipped-chip case is asserted only from above, so a lost span passes
    detail: |
      reserve_mouse_test.go:46-49 bounds spans with chip.End > width but its only
      lower bound is len(Chips) > len(Actors), which cannot fire with three distinct
      threads. Mutating reserve.go:159 to `used > start && used-start >=
      textwidth.Width(label)` makes every truncated chip silently unclickable and the
      whole couchtty suite stays green. Assert the expected span count per width, and
      assert ColumnToActor inside a truncated chip's columns.
  - id: new
    severity: Important
    family: onedirectional-geometry-assertion
    title: |
      clampExtents and the scrolled list are unreachable from any test
    detail: |
      Replacing clampExtents' body with `return extents` leaves menu_render green, and
      so does computing the extent index from the inventory row instead of len(lines).
      The fixture is 2 actors in 14 rows, so neither rowBudget scrolling nor the height
      clamp is ever entered. Plan Task 4 Step 4 and the Done-when both require the
      scrolled and clipped cases.
  - id: new
    severity: Important
    family: guard-not-registered
    title: |
      mouseinput ships with no test file, and its Core-concepts row is out of the guard's scope
    detail: |
      Plan Task 1 names cmd/internal/mouseinput/mouseinput_test.go; it does not exist
      (go test reports "no test files"). ParsePrefix, MaxReport, WheelUp, WheelDown and
      Terminators have no direct coverage. The plan's `mouseinput.Event / mouseinput.Find`
      row would have been caught by assertDirectTest, but conceptPackage filters it out,
      so Task 5 pinned 6 of the 8 non-planned rows. Third instance of the family PQ-6
      and PQ-14 named.
  - id: new
    severity: Important
    family: coordinate-base-unstated-at-seam
    title: |
      ChipSpan and ActorExtent state no coordinate base, on a struct that already carries a 1-based one
    detail: |
      RenderMenuView documents Cursor as "1-based terminal cells in Body"; the new
      Extents field on the same struct is 0-based rows of Body and says so nowhere.
      ChipSpan.Start/End are 0-based display columns while mouseinput.Event.X is 1-based
      and the row is painted from terminal column 1. M2/M3 consume this surface; state
      the base and origin in the ChipSpan, ActorExtent and PointToActor doc comments.
  - id: new
    severity: Important
    family: stale-issue-reference
    title: |
      The 173-to-192 renumber left the issue path in the same comment block stale
    detail: |
      deadsymbols_test.go:47 still reads workshop/issues/000173-disposition-six-production-
      symbols-reachable-only-from-tests.md, a file that no longer exists, two lines above
      the six entries the sweep updated to pair#192. The comment's own next sentence makes
      the ticket's existence load-bearing. grep -rn "000173-disposition" returns this one
      site.
  - id: new
    severity: Minor
    family: orphaned-doc-comment
    title: |
      RenderStatusRow's doc block is now attached to ChipSpan
    detail: |
      reserve.go:86-93 explains untrusted labels and control-byte stripping, but the new
      ChipSpan type was inserted between it and RenderStatusRow (:126), which now has no
      doc comment. godoc attributes the sanitize rationale to the wrong type.
  - id: new
    severity: Minor
    family: promoted-constant-with-surviving-literal
    title: |
      WheelUp/WheelDown have zero call sites while termcmd keeps the 64/65 literals
    detail: |
      mouseinput.go:36-37 introduces the named constants; termcmd/run.go:451,457 still
      switch on event.Button == 64 / == 65. The promotion added a second source of truth
      for the wheel codes rather than removing one (ARCH-DRY).
  - id: new
    severity: Minor
    family: move-residue
    title: |
      mousePressEvent alias and parseSGRMousePressPrefix have no callers after the move
    detail: |
      termcmd/run.go:579 and :585. parseSGRMousePressPrefix was only ever reached through
      the old findSGRMousePress, which now delegates to mouseinput.Find. deadSymbolScope
      is cmd/internal/couchcore, so the orphan guard cannot see termcmd.
  - id: new
    severity: Minor
    family: doc-ahead-of-enforcement
    title: |
      MaxReport and the atlas both state a bound that no code enforces yet
    detail: |
      mouseinput.go:31 and atlas/couch.md read as shipped invariants; the bound is M2
      Task 7 Step 3b. Mark the constant as consumed in M2 so it does not read as enforced.
  - id: new
    severity: Minor
    family: unpinned-exported-shape
    title: |
      The one-extent-per-actor run shape is unobservable through PointToActor
    detail: |
      Removing the run-merge at menu_render.go:437-441 leaves the suite green, because
      per-line extents still resolve to the right thread. The exported Extents field's
      documented shape ("where each actor was DRAWN") is therefore unpinned for any
      consumer that reads it directly rather than through PointToActor.
```
