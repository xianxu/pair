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

---

## Re-review — 2026-09-05T13:44:27-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..f95da992f2faa2ef077c1f45998159585ee40033 |
| command | sdlc milestone-close --issue 172 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-05T13:44:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1's central claim — click geometry must fall out of the pass that already clips, never a second derivation — is genuinely delivered, and the parser promotion is a real move rather than a copy. What blocks the boundary is that the commit closing the prior round claims two Important findings fixed that mutation testing refutes: `clampExtents` is still dead to the suite (replacing its body with `return extents` stays green), the scrolled case is still unreachable (adding the scroll offset to every extent index stays green), and a truncated non-active chip can still lose its span silently. On top of that, M2 and M3 production code landed inside the M1 window with its headline behaviours unpinned — deleting `c.writeOwn(hostty.EnableMouseClicks)`, dropping the forward write to the child, and deleting `&& !origin.Manual` each leave the entire `couchtty` suite green. Every mutation below was run in the working tree and the tree restored; `git status` is clean.

## 1. Strengths

- **The DRY claim is enforced, not asserted.** `reserve.go:159-161` records each span inside the same `appendText` that clips, and the naive re-derivation reddens. Mutating `used > start` to require the full label width fails `reserve_mouse_test.go:60` with `"beta" is drawn in "alpha  [beta" but has no span`.
- **The notice re-base is at the right layer and pinned.** `menu_render.go:158-164` re-bases in `RenderMenuView` after the index-1 insert, and `menu_extent_test.go` runs the fixture both ways — the plan's predicted off-by-one is a real test, not a comment.
- **The parser move is byte-faithful and now has its own tests.** `mouseinput.Parse/ParsePrefix/Find/IsPrefix` are character-identical to the `termcmd` originals; `rename_input.go:182` was swept onto the shared `Terminators`; `termcmd`'s suite passed with only field-name edits. BR-6's coverage gap is closed properly.
- **The #127 bound is real.** `keys.go:301` gates the hold on `MaxReport`, `TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard` pins it, and `armInputEscape()` covers the stopped-typing case — the hazard is bounded twice.
- **Both maps are total and carry a `ThreadAddress`, not an index**, so a stale span degrades to "no longer attached" rather than switching to the wrong actor (`reserve.go:117-125`, `menu_render.go:67-75`).
- **`TestNoReportReachesAChildThatNeverAskedForMouse`** states the zero-bytes rule as a property over the whole event space rather than one case. That is the right shape.

## 2. Critical findings

**The M2/M3 behaviour that shipped in this window is unpinned across the board** (`console.go:528`, `:1503`, `:1455`). *This is the 2nd finding in family `unpinned-exported-shape`* — I am not asking for these three sites to be patched one at a time. The rule: **every behaviour this issue introduces is pinned by a test that reddens when the behaviour is removed, and the enumeration is already written — it is the plan's own "Done-when → Task map". The check is the mutation, not the green run.** Measured prevalence, all three verified in the working tree:

- deleting `c.writeOwn(hostty.EnableMouseClicks)` (`console.go:528`) — couch never asks the terminal for clicks, the feature is dead in any real terminal — suite green;
- replacing `_, _ = child.child.Write(hit.Raw)` (`console.go:1503`) with a discard — a child *with* tracking receives nothing, violating a Done-when bullet — suite green;
- deleting `&& !origin.Manual` (`console.go:1455`) — the manual-switch marker the commit message headlines does nothing — suite green. Plan Task 11 Step 2 says this exact mutation check "is not optional".

Walking the map, four of its rows have no test at all: `TestClickInTheSwitcherTakesTheReturnPath`, `TestClickIsAManualSwitch`, `TestForwardPreservesRawBytes`, `TestTeardownDisablesMouseTracking`. The switcher branch (`console.go:1516-1521`) has no fixture that focuses the panel, and no fixture ever puts a child into mouse mode.

## 3. Important findings

**M2 and M3 production code landed inside the M1 window** (`f95da992`, family `milestone-scope-overrun`). The plan tags three boundaries precisely because the middle one is hard; landing all three in one commit means M1's review is asked to bless code whose Done-when rows and tests belong to later gates, and — worse — once M1 closes at `f95da992`, M2's and M3's own mandatory reviews open on an empty window. The mechanism that was supposed to catch the Critical above is the gate this commit routes around. Either split the commit so M2/M3 code lands after M1 closes, or close M1/M2/M3 against this one window and accept the tests all three owe now.

**couch's own mouse mode is written once and never re-asserted** (`console.go:528`). *This is the 3rd finding in family `unspecified-event-policy`.* Do not fix the re-assert alone. The rule the family keeps asking for: **the disposition table must be crossed with the CHILD's mode-transition events, not only the operator's report events.** `?1000` is terminal-global, so a child writing `\x1b[?1000l` to the host stream — which nvim does on exit — turns couch's own clicks off for the whole terminal, and `EnableMouseClicks` has exactly one call site with no repaint path that re-emits it (`writeOwn` queues a *repaint*, not the bytes it dropped). `ptychild.Screen.Mouse()` already observes the transition, so couch has the signal and no policy. Enumeration the table needs: child enables; child disables; child exits with mouse on; switch between two children with different modes; replay re-asserting the child's modes over couch's.

**The click reducer is a parallel handler that restates and diverges from Enter's decision** (`menu.go:349-365` vs `:475-479`, family `parallel-handler-restates-decision`). The Done-when requires the switcher click be "asserted against the same handler, not a parallel one (ARCH-DRY)". Three divergences: the `switch`/`resume` choice is a second copy of `reduceRootKey`'s; a non-actionable row gets a notice from Enter and silence from a click; and `state.InFlight.Manual = true` (`menu.go:363`) is applied **unconditionally**, including when `dispatchMenuOperation` refused because another operation was already in flight (`menu.go:1542-1544`) — writing a flag onto a different operation's origin. Extract the actionable-thread → operation decision into one function both arms call, and set `Manual` only when the dispatch produced effects.

**Docs lag the surface that shipped in this window** (family `docs-lag-shipped-surface`). `atlas/couch.md` gained the M1 geometry (commit `1c895d4f`) but nothing for what `f95da992` shipped: the four-way disposition table, the narrowed release rule, couch's ownership of `?1000;?1006`, and the manual-switch classification. `menuControls` (`menu.go:19-33`) still has no mouse row, so `TestREADMEDocumentsEveryPanelControl` cannot fire and README does not mention that a click in the switcher selects-and-enters — a user-facing gesture that is live in the binary today. Task 12 defers these to M3's close, which is only defensible if the code also waits for M3.

## 4. Minor findings

- `menu.go:199-208`: `MenuEventNotice`'s three-line doc block is now attached to `MenuEventMouseSwitch`; `MenuEventNotice` has none. Same defect as BR-9, in a second file, introduced by the commit that was supposed to fix the first.
- `hostty/control.go:62`: `DisableMouseClicks` has zero call sites — a third unused symbol alongside `mousePressEvent` and `parseSGRMousePressPrefix`.
- `mouseinput.go:56`: `|| s == ""` is unreachable — `strings.HasPrefix(s, "\x1b[<")` already rejects the empty string.
- `console.go:528`: `writeOwn`'s contract is "refuse while mid-sequence, the next chunk pays the debt" — it pays a *repaint*, not arbitrary bytes, so a mode-set routed through it is silently droppable. Safe today only because `hostScan` is zero at `Run` start.
- `mouseinput.Parse` accepts negative coordinates via `%d`; both maps are total so it degrades to "no actor", but the invalid state is representable at the parse boundary (ARCH-SECURE prefers it not to be).

## 5. Test coverage notes

The split-at-every-boundary loop in `TestMouseReportsAreWithheldAndSurviveEverySplit` is the one genuine ordering seam in the diff and it is well built. Against that, the geometry tests are still one-directional in the way BR-4 named: `reserve_mouse_test.go:56-62` requires a span only when `strings.Contains(plain, label)` holds for the *whole* label, so a chip drawn as a strict prefix can lose its span undetected — I verified this with `(a.Active || used-start >= textwidth.Width(label))`, which leaves the suite green. `TestExtentsNeverPointPastTheDrawnMenu` asserts `PointToActor(extent.Start)` maps back to `extent.Thread`, which is tautological — both sides read `Extents` — and its fixture always selects `inventory[0]`, so `start` is always 0 and the scroll window is never entered. `clampExtents` cannot fire at all at the layout minimum (`rowBudget = height - 2` makes the block exactly `height` lines), so a test that reaches it needs either a sub-minimum height or a fixture with a filter row.

## 6. Architectural notes

- **ARCH-DRY — flag.** Pass on the parser move and on chip spans. Flag on `menu.go:356-358` restating `reduceRootKey`'s switch/resume rule, and on `termcmd/run.go:453,459` keeping the `64`/`65` literals while `mouseinput.WheelUp/WheelDown` exist (BR-10, re-raised).
- **ARCH-PURE — pass.** `RenderStatusRow`, `ColumnToActor`, `ActorExtent`, `PointToActor` and `RouteMouseReport` are pure, IO-free and unit-tested with no terminal; `onMouse` is the thin seam and does the 1-based → 0-based conversion in exactly one place.
- **ARCH-PURPOSE — flag.** The shadow-sweep across the plan's own Done-when → Task map leaves four rows with no delivery, and "same path as Return" is answered with a parallel restatement rather than a shared decision.
- **ARCH-MOCK — pass with a gap.** `hostty.FakeHost` and `ptychild.NewFakeChild` are stateful doubles behind the production seam, and `console_mouse_test.go` drives real bytes through the real input loop — the right shape. Gap: no fixture feeds `\x1b[?1000h` through the fake child, so the forward half of the seam is never exercised against it, and Task 12's live conformance step has not run.
- **ARCH-CONSTRAINTS — pass.** `?1000` not `?1002`/`?1003` is declared and implemented (`hostty/control.go:51-58`); `MaxReport` bounds the hold; no unbounded fan-out; one small slice per paint.
- **ARCH-SECURE — pass.** The report is parsed into a typed `Event` at the boundary and refused rather than guessed (`mouseinput_test.go:29-40` covers X10, non-numeric and unterminated); out-of-range coordinates fall out of two total maps as `false`; untrusted labels still go through `sanitize`.
- **ARCH-ORDER — flag.** Two carried states with no written transition set: couch's own mouse mode (one write, no re-assert, invalidated by a child DECRST) and `MenuOperationOrigin.Manual` (written after a dispatch that may have been refused). Both are exactly the shape the entry describes: state a component carries between events, whose legal combinations are unwritten.

## 7. Plan revision recommendations

The plan has **no `## Revisions` section at all**, despite six revising commits and BR-3 asking for one explicitly. Append:

- **Core-concepts correction.** `hostty.MouseClickTracking` (`:182`) names a symbol that exists nowhere and never will — M2 shipped `hostty.EnableMouseClicks` / `DisableMouseClicks` in this same window. Rename the row and, if M2/M3 close against this window, flip `MouseDisposition`, `RouteMouseReport`, `seqMouse`, `Interceptor.FeedHit` and `Console.onMouse` from `planned` to `new`/`modified` — the contract test skips `planned` rows, so six rows describing shipped code currently assert nothing.
- **Task 4 Step 4 is not delivered.** Record that the scrolled case and `clampExtents` remain unreachable from any test, with the two mutations that prove it, and state what fixture reaches each (a selection below the scroll window; a height or filter combination that makes the block exceed the clamp).
- **Milestone shape.** Record that M2 and M3 production code landed before M1's boundary, and say how the three boundaries are now being closed, so the later reviews are not opened on an empty window.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Disposition table is in the plan and in RouteMouseReport; release-to-a-no-mouse-child swallows, MaxReport bounds the hold, MouseHit.Raw carries the wire bytes.
  - id: BR-2
    disposition: addressed
    note: |
      Task 12 now names menu.go:19, README.md, atlas/couch.md and the readme_test guard.
  - id: BR-3
    disposition: not-addressed
    note: |
      Status flipped to `planned — M2`, but the symbol hostty.MouseClickTracking still exists nowhere and M2 shipped EnableMouseClicks/DisableMouseClicks in this same window; no Revisions entry was appended.
  - id: BR-4
    disposition: not-addressed
    note: |
      The named mutation reddens only because the fixture's `beta` is Active; `(a.Active || used-start >= textwidth.Width(label))` still leaves the whole suite green. Neither ask landed.
  - id: BR-5
    disposition: not-addressed
    note: |
      Verified: clampExtents -> `return extents` is green, and `index := len(lines) + start` is green. Neither the clipped nor the scrolled path is reachable.
  - id: BR-6
    disposition: addressed
    note: |
      mouseinput_test.go exists and covers Parse/ParsePrefix/Find/IsPrefix/MaxReport directly; the out-of-scope row remains pair#188's structural issue.
  - id: BR-7
    disposition: addressed
    note: |
      ChipSpan, ColumnToActor, ActorExtent and PointToActor all state ZERO-BASED and name the 1-based conversion; onMouse does it once.
  - id: BR-8
    disposition: addressed
    note: |
      000173 -> 000192 in the comment and all six entries; workshop/issues/000192-... exists.
  - id: BR-9
    disposition: not-addressed
    note: |
      reserve.go:93-94 still runs the RenderStatusRow doc block straight into ChipSpan's; RenderStatusRow (:133) has no doc comment. A second instance now exists at menu.go:199-208.
  - id: BR-10
    disposition: not-addressed
    note: |
      termcmd/run.go:453,459 still switch on the literals 64 and 65 while mouseinput.WheelUp/WheelDown exist.
  - id: BR-11
    disposition: not-addressed
    note: |
      mousePressEvent (run.go:579) and parseSGRMousePressPrefix (:585) still have zero callers.
  - id: BR-12
    disposition: addressed
    note: |
      keys.go:301 enforces MaxReport and TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard pins it.
  - id: BR-13
    disposition: not-addressed
    note: |
      Verified: deleting the run-merge at menu_render.go:463-467 leaves the suite green.
findings:
  - id: new
    severity: Critical
    family: unpinned-exported-shape
    title: |
      Three shipped M2/M3 behaviours are unpinned - removing each leaves the whole couchtty suite green
    detail: |
      This is the 2nd finding in family `unpinned-exported-shape`. Do NOT patch the three
      sites. State the rule: every behaviour this issue introduces is pinned by a test that
      reddens when the behaviour is removed, and the enumeration is already written - the
      plan's "Done-when to Task map". The check is the mutation, not the green run.
      Measured prevalence, all verified in the working tree and restored: deleting
      c.writeOwn(hostty.EnableMouseClicks) (console.go:528) - green; discarding
      child.child.Write(hit.Raw) (console.go:1503) - green; deleting `&& !origin.Manual`
      (console.go:1455) - green, which is the marker the commit message headlines and the
      one Plan Task 11 Step 2 calls not optional. Four map rows have no test at all:
      TestClickInTheSwitcherTakesTheReturnPath, TestClickIsAManualSwitch,
      TestForwardPreservesRawBytes, TestTeardownDisablesMouseTracking.
  - id: new
    severity: Important
    family: milestone-scope-overrun
    title: |
      M2 and M3 production code landed inside the M1 window, so their own boundary reviews open on an empty range
    detail: |
      f95da992 ships routing, the Interceptor's SGR arm, couch's mode enable, onMouse and the
      manual-switch marker - all tagged M2/M3 in the plan - inside the window this M1 gate
      reviews. Once M1 closes at f95da992, BASE_SHA for M2 and M3 is that commit, so the
      mandatory fresh-eyes review for the milestone the plan calls "the work" sees nothing.
      That is the gate that was supposed to catch the Critical above. Either split so M2/M3
      land after M1 closes, or close all three against this window and pay their tests now.
  - id: new
    severity: Important
    family: unspecified-event-policy
    title: |
      couch's own mouse mode is written once and never re-asserted, so a child's DECRST silently ends the feature
    detail: |
      This is the 3rd finding in family `unspecified-event-policy`. Do NOT add a re-assert
      and stop. The rule the family keeps asking for: the disposition table must be crossed
      with the CHILD's mode-transition events, not only the operator's report events.
      console.go:528 is the only write of EnableMouseClicks; ?1000 is terminal-global, so a
      child emitting \x1b[?1000l - nvim does this on exit - turns couch's clicks off for the
      whole terminal with nothing to restore them. ptychild.Screen.Mouse() already observes
      the transition, so couch has the signal and no policy. Enumeration: child enables;
      child disables; child exits with mouse on; switch between two children with different
      modes; replay re-asserting the child's modes over couch's.
  - id: new
    severity: Important
    family: parallel-handler-restates-decision
    title: |
      The click reducer restates Enter's switch/resume rule and diverges from it, and sets Manual on a refused dispatch
    detail: |
      menu.go:349-365 duplicates reduceRootKey's operation choice (menu.go:475-479); a
      non-actionable row gets a notice from Enter and silence from a click; and
      state.InFlight.Manual = true (menu.go:363) runs unconditionally, including when
      dispatchMenuOperation refused because another operation was in flight
      (menu.go:1542-1544) - writing the flag onto a different operation's origin. The
      Done-when requires the same handler, not a parallel one (ARCH-DRY). Extract the
      actionable-thread to operation decision into one function both arms call, and set
      Manual only when the dispatch produced effects.
  - id: new
    severity: Important
    family: docs-lag-shipped-surface
    title: |
      atlas and README cover M1's geometry but not the routing, ownership and click gesture that shipped in the same window
    detail: |
      atlas/couch.md gained the M1 geometry in 1c895d4f and nothing for f95da992: the
      four-way disposition table, the narrowed release rule, couch's ownership of
      ?1000;?1006, and the manual-switch classification. menuControls (menu.go:19-33) has no
      mouse row, so TestREADMEDocumentsEveryPanelControl cannot fire and README never
      mentions that a click in the switcher selects and enters - a user-facing gesture that
      is live in the binary today. Task 12 defers this to M3's close, which only works if
      the code also waits for M3.
  - id: new
    severity: Minor
    family: orphaned-doc-comment
    title: |
      MenuEventNotice's doc block is now attached to MenuEventMouseSwitch, a second instance in the commit that left the first
    detail: |
      This is the 2nd finding in family `orphaned-doc-comment`; BR-9 is the first and is
      re-raised not-addressed. Do NOT fix this instance alone. The rule: a symbol inserted
      into an existing file must go after the symbol an adjacent doc block documents, or the
      block must be re-anchored. Enumeration for this issue: reserve.go:93-94 (RenderStatusRow's
      block now reads as ChipSpan's) and menu.go:199-208 (MenuEventNotice's block now reads as
      MenuEventMouseSwitch's). Both are checkable with `go doc`.
  - id: new
    severity: Minor
    family: move-residue
    title: |
      hostty.DisableMouseClicks joins two other zero-call-site symbols left by the promotion
    detail: |
      This is the 2nd finding in family `move-residue`; BR-11 is the first and is re-raised
      not-addressed. Do NOT delete this one symbol. The rule: a symbol this issue introduces
      or leaves behind has a production call site by the milestone that introduces it, or it
      is deleted. Enumeration: termcmd/run.go:579 (mousePressEvent alias), :585
      (parseSGRMousePressPrefix), hostty/control.go:62 (DisableMouseClicks), and
      mouseinput.WheelUp/WheelDown which exist only in tests while run.go:453,459 keep the
      literals. deadSymbolScope is cmd/internal/couchcore, so no guard sees any of them.
```

---

## Re-review — 2026-09-05T16:25:23-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..15eaf1919d98badcaef169ec54de061d7ab0a6b6 |
| command | sdlc milestone-close --issue 172 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-05T16:25:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The geometry layer M1 owns is well built — the spans come out of the clipping pass, the extents come out of the drawn rows and are re-based where the notice shifts them, and the promoted `mouseinput` decoder is a genuine one-parser consolidation with its own tests. What blocks SHIP is not the design: it is that this round's *claimed fixes are unpinned by the same rule the round was opened to enforce*. I mutated every behaviour the commit message names and restored the tree each time. Three of the round's own fixes — BR-16's paint-time `EnableMouseClicks` re-assert, BR-17's `enterOperationFor` extraction, and BR-17's `len(effects) > 0` guard — can each be reverted with the whole `couchtty` suite staying green (only the pre-existing pty-sandbox failure remains). So can `clampExtents`, the extent run-merge, the scroll-offset re-base, and the chip span for a chip truncated below three columns. That is seven unpinned behaviours, three of them written *this round* in answer to a finding whose stated rule was "the check is the mutation, not the green run." Two claimed fixes are real and verified red (`child.child.Write(hit.Raw)`, `&& !origin.Manual`), and the docs gate is genuinely satisfied. The rest of the prior round is re-raised.

**1. Strengths**

- `cmd/internal/couchtty/reserve.go:164-172` — the span is recorded inside the same `appendText` that clips, exactly as the Spec demanded, so a dropped chip contributes no span by construction rather than by a parallel rule.
- `cmd/internal/couchtty/menu_render.go:160-167` — re-basing extents in `RenderMenuView` after the index-1 notice insert, with the reason written down, is the right seam; `renderRootMenuFrame` genuinely cannot know what its caller inserts above it. `TestPointToActorSpansEveryLineOfAnActor` runs both notice states.
- `cmd/internal/couchtty/mouse_test.go:52-67` — `TestNoReportReachesAChildThatNeverAskedForMouse` states the zero-bytes rule as a property over the whole (button × release × row) space instead of one case. That is the right shape for a Done-when bullet.
- `cmd/internal/couchtty/console_mouse_test.go:28-58` — the fixture drives real SGR bytes through `Run`'s input loop and records in-flight state *at dispatch*. The comment explaining why reading `InFlight` after completion reads a zero value is the kind of correction that stops the next author repeating it, and the `!origin.Manual` mutation does redden it.
- `cmd/internal/couchtty/keys.go:282-303` — the mouse arm sits before the fixed-string table with the bound applied in the same branch, and `TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard` reddens without it. #127's hazard is actually closed here.

**2. Critical findings** — see BR-14 and BR-3 in the block below (re-raised, not new).

**3. Important findings** — BR-4, BR-5, BR-15, BR-16, BR-17 re-raised. One new Minor only.

**4. Minor findings**

- `console.go:1557` — `HitMouse: func() {}` satisfies the enumeration guard with a stub the dispatcher never reaches (`processInput` special-cases `HitMouse` at `:647` *before* the table). Raised below.
- `console.go:528` vs `:1028` — the `Run` site is now redundant with the paint-time re-assert; deleting it alone leaves everything green. Not wrong, just untested and duplicative.
- `mouseinput.Parse` accepts negative coordinates into the typed value (`\x1b[<-5;-5;-5M` → `ok=true`). Harmless today because `ColumnToActor`/`PointToActor` are total, but it weakens "parse into a typed value at the boundary and refuse what does not fit" (ARCH-SECURE).

**5. Test coverage notes**

Measured, each mutation applied to the working tree, suite run, tree restored to `15eaf191` (verified clean):

| mutation | site | result |
|---|---|---|
| `clampExtents` → `return extents` | `menu_render.go:183` | **green** |
| `index := len(lines) + start` | `menu_render.go:455` | **green** |
| delete the run-merge | `menu_render.go:465-469` | **green** |
| span only when drawn ≥ 3 cols | `reserve.go:170` | **green** |
| delete paint-time `EnableMouseClicks` | `console.go:1028` | **green** |
| `enterOperationFor(thread)` → `"switch"` | `menu.go:357` | **green** |
| drop the `len(effects) > 0` guard | `menu.go:361` | **green** |
| delete `child.child.Write(hit.Raw)` | `console.go:1503` | red ✓ |
| drop `&& !origin.Manual` | `console.go:1461` | red ✓ |

Also probed: the scrolled root list *is* reached by the suite (`TestRenderMenuKeepsSelectedRowVisibleAndBounded` panics on an injected `start > 0` probe) — so the fixture exists, but no test reads `Extents` there. And `clampExtents` looks not merely untested but unreachable: `rowBudget` already subtracts the header, filter and notice rows, so `len(lines)` lands at exactly `height` and the clamp never bites.

**6. Architectural notes**

- **ARCH-DRY — flag.** The `mouseinput` promotion is the good half. The bad half: `mousePressEvent` (`run.go:579`) and `parseSGRMousePressPrefix` (`run.go:585`) have zero references anywhere, `hostty.DisableMouseClicks` has zero, and `mouseinput.WheelUp/WheelDown` exist only in tests while `run.go:453,459` keep the `64`/`65` literals. The promotion added a source of truth without retiring one. (BR-10/BR-11/BR-20.)
- **ARCH-PURE — pass.** Geometry and routing are pure and unit-tested with no terminal; the contract test enforces no IO imports on the PURE rows; IO stays in `console.go`/`keys.go`.
- **ARCH-PURPOSE — flag.** The plan's Core-concepts table is the declared single source and six of its rows still say `planned` for code live in this window, so the guard skips them (BR-3). The Done-when→Task map is satisfied by test *names* rather than by tests that discriminate: `TestClickInTheSwitcherTakesTheReturnPath` only exercises a live thread, where Enter and click agree, so "same handler, not a parallel one" is unasserted.
- **ARCH-MOCK — pass.** `hostty.FakeHost` and `ptychild.NewFakeChild` are stateful and sit on the production seam; `TestForwardPreservesRawBytes` feeds a real `\x1b[?1000h` through `Screen` rather than setting a flag. The missing piece is the live conformance leg — Task 12's manual nvim verification is the substitute and no `## Log` entry records it.
- **ARCH-CONSTRAINTS — pass.** `?1000` over `?1002/1003` is reasoned from click-vs-motion rates; the Interceptor hold is bounded at 32 bytes; the per-paint 14-byte DECSET is negligible beside the row repaint it accompanies.
- **ARCH-SECURE — pass.** Coordinate overflow is refused (`\x1b[<0;99999999999999999999;1M` → `ok=false`), untrusted labels are still sanitized, and out-of-range coordinates degrade to "nobody" rather than clamping.
- **ARCH-ORDER — flag.** `ptychild.Screen` collapses `1000/1002/1003/1006` into **one bool** (`screen.go:39,415`), and `RouteMouseReport` reads that bool as "the child wants mouse". Two reachable consequences: (a) couch re-asserts `?1006` globally on every paint, so a child that requested `?1000` alone now receives SGR-encoded reports it did not ask for — forwarded *raw*, which is the "unchanged" Done-when broken by couch's own DECSET; (b) a child DECRSTing only `?1006` flips `Mouse()` false and couch starts swallowing that child's own clicks. Both are rows the BR-16 enumeration would have produced. Separately, `Interceptor.mouse` is a payload valid only between one `FeedHit` and the next, kept safe by a hand-written special case rather than by the type.
- **Boundary shape.** After M1 closes at `15eaf191`, `M2` (which now absorbs M3) opens on an empty range — the Revisions entry merged the milestones but the merge does not create a diff for M2 to review, and Task 12's manual verification and the `pair#166` re-evaluation still ride on that boundary.

**7. Plan revision recommendations**

The plan has no `## Revisions` section at all; Chunk 3's heading was edited in place, which is what AGENTS.md §1 forbids. It needs:

- **`## Revisions` — 2026-09-05, M3 folded into M2.** Reason + delta, replacing the parenthetical edit at line 427.
- **`## Revisions` — 2026-09-05, Core-concepts rows flipped to shipped.** `MouseDisposition`, `RouteMouseReport`, `seqMouse`, `Interceptor.FeedHit` (lines 115-120), `Console.onMouse` (181) → `new`/`modified`, and line 182's `hostty.MouseClickTracking` → `hostty.EnableMouseClicks` / `DisableMouseClicks` with its milestone tag corrected. Flipping them will make `conceptInventory` report six unexpected rows until they are added to `core_concepts_contract_test.go` — that report is the signal, not a failure.
- **`## Revisions` — the disposition table gains child-mode-transition rows.** The table at "The complete disposition table" is crossed only with the operator's report events; add the child's: enables, disables, exits with mouse on, requests `?1000` without `?1006`, disables `?1006` alone, and replay re-asserting over couch's.

```findings
dispose:
  - id: BR-3
    disposition: not-addressed
    note: |
      Status flipped, but no plan Revisions entry, the row still names a symbol that exists nowhere, and six rows now claim `planned` for shipped code.
  - id: BR-4
    disposition: not-addressed
    note: |
      Verified: dropping the span for any chip truncated below 3 columns leaves the whole suite green.
  - id: BR-5
    disposition: not-addressed
    note: |
      Verified twice: `return extents` and a scroll-offset re-base both stay green; the new fixture enters neither path.
  - id: BR-9
    disposition: not-addressed
    note: |
      `go doc ChipSpan` still prints RenderStatusRow's sanitize rationale; RenderStatusRow still has no doc.
  - id: BR-10
    disposition: not-addressed
    note: |
      run.go:453,459 still switch on the 64/65 literals; WheelUp/WheelDown remain test-only.
  - id: BR-11
    disposition: not-addressed
    note: |
      run.go:579 and :585 still have zero references anywhere in the tree.
  - id: BR-13
    disposition: not-addressed
    note: |
      Verified: deleting the run-merge at menu_render.go:465-469 leaves the suite green.
  - id: BR-14
    disposition: not-addressed
    note: |
      Two of the three named sites are now pinned, but three of THIS round's own fixes are not; seven unpinned behaviours measured.
  - id: BR-15
    disposition: not-addressed
    note: |
      Merging M3 into M2 does not give M2 a diff; its close still opens on an empty range, carrying Task 12's manual check and pair#166 with it.
  - id: BR-16
    disposition: not-addressed
    note: |
      The re-assert can be deleted with the suite green, and the child-mode-transition enumeration was never written.
  - id: BR-17
    disposition: not-addressed
    note: |
      Both fixes are unpinned by mutation, and a non-actionable row still gets a notice from Enter and silence from a click.
  - id: BR-18
    disposition: addressed
    note: |
      atlas/couch.md, README and the menuControls row all landed; the README guard fires on the new row.
  - id: BR-19
    disposition: not-addressed
    note: |
      menu.go:200-209 unchanged; MenuEventNotice's block still reads as MenuEventMouseSwitch's.
  - id: BR-20
    disposition: not-addressed
    note: |
      DisableMouseClicks, mousePressEvent, parseSGRMousePressPrefix and WheelUp/WheelDown all still have zero production call sites.
findings:
  - id: new
    severity: Minor
    family: guard-not-registered
    title: |
      The handler-table entry for HitMouse is a stub the dispatcher never reaches, so the enumeration guard proves nothing for it
    detail: |
      This is the 2nd finding in family `guard-not-registered`. Do NOT just delete
      or fill in this one entry. The rule: an enumeration guard is satisfied only
      by the thing it guards -- registering a value the production path never
      reads converts the guard into a formality that reports coverage it does not
      have. console.go:647 special-cases HitMouse BEFORE consulting the table, so
      hitHandlers()[HitMouse] (console.go:1557) is a `func() {}` with no caller;
      AllInterceptorHits still counts it as proven. The same shape is one edit
      away for any future payload-carrying hit. Either widen the table's value to
      carry the payload so every hit really does route through it, or have the
      guard assert reachability rather than presence. Related: Interceptor.mouse
      is a payload valid only between one FeedHit and the next, kept correct by a
      hand-written ordering rather than by the type (ARCH-ORDER).
```

---

## Re-review — 2026-09-05T16:47:31-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..73294518efda855f24e15dc7bc3c9b1ec33c3ca6 |
| command | sdlc milestone-close --issue 172 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-05T16:47:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pure layer M1 actually claims is in good shape, and this round genuinely paid down the biggest debt from the last one: BR-14's three unpinned behaviours all redden under mutation now, BR-4's clipped chip and BR-13's extent run-merge are pinned by constructed cases rather than sweeps, and BR-17's duplicated switch/resume rule collapsed into one `enterOperationFor` both arms call. What blocks SHIP is the fix for BR-16. couch now re-asserts `\x1b[?1000;1006h` on *every* paint; mouse-tracking modes 1000/1002/1003 are a single mutually-exclusive state in every terminal pair targets, so that re-assert silently demotes a child that enabled `?1002`/`?1003` (nvim drag-select, tmux, htop) to click-only — while the code comment and `atlas/couch.md` both assert the opposite, and `ptychild.Screen.Mouse()` still reports `true` so couch keeps forwarding press/release into a child that will never see the motion that ends its drag. That is a direct regression of the Done-when "a child that did enable tracking still receives its own events unchanged", and deleting the re-assert leaves the whole suite green, so nothing observes it either way. Alongside it, four prior findings whose "fix" was a comment describing the fix (BR-9, BR-19), a coverage half that stayed unreachable (BR-5's scrolled window), and the guard formality BR-21 named, are all re-raised not-addressed.

Verification note: the pty-backed tests (`TestNotificationPTYConformance`, `ptychild`, `couchcmd` launch tests) cannot run in this environment — `operation not permitted` on pty allocation. Every failure in `go test ./...` at HEAD is of that shape; no non-pty failure surfaced. All mutation results below were run against the working tree at `73294518` and restored (tree verified clean afterwards).

## 1. Strengths

- **`reserve.go:159-175`** — spans recorded inside the same `appendText` pass that clips. Mutating the guard to `used-start >= textwidth.Width(label)` reddens *two* tests now (`reserve_mouse_test.go:60` and `:124`), which is exactly what BR-4 asked for and what the width-sweep alone could not do.
- **`menu.go:594-607`** — `enterOperationFor` is the right extraction: one authority for "what does landing on a row do", called by both `reduceRootKey`'s Enter arm and the `MenuEventMouseSwitch` arm (ARCH-DRY).
- **`menu.go:358-363`** — gating `state.InFlight.Manual = true` on `len(effects) > 0` is the precise fix for the refused-dispatch half of BR-17, and mutating `&& !origin.Manual` off `console.go:1461` reddens `TestClickIsAManualSwitch`.
- **`console_mouse_test.go:28-59`** — the fixture drives real bytes through `Run`'s input loop rather than calling `onMouse`, and captures `InFlight` *at dispatch* because it is cleared on completion. That comment is the difference between a test that passes for the right reason and one that reads a zero value.
- **`keys.go:297`** — the `MaxReport` bound releases correctly under *incremental* feeds too, not just the single-chunk case the test covers (verified: `\x1b[<` + 8×5 digits releases at feed 7 and typing behind it arrives).
- **`menu_extent_test.go:150-155`** — declining to fabricate a test for `clampExtents` and saying why, rather than manufacturing one, is the honest answer; the unreachability claim checks out (`menuMinHeight = 10` ⇒ `rowBudget = height-2-filter-notice ≥ 6`, so `lines[:height]` never fires while extents exist).

## 2. Critical findings

**`cmd/internal/couchtty/console.go:1024-1030` — the per-paint `?1000h` re-assert clobbers a child's `?1002`/`?1003` tracking.**

`paintNow` writes `hostty.EnableMouseClicks` (`\x1b[?1000;1006h`) unconditionally. The comment claims "DECSET is additive and idempotent, so this cannot clobber a mode the child enabled for itself"; that is false for this family of modes. xterm keeps one `screen->send_mouse_pos`; Alacritty explicitly does `self.mode.remove(TermMode::MOUSE_MODE)` before inserting the requested one; kitty, Ghostty and iTerm2 all hold a single mouse-tracking enum. Setting 1000 while 1002 is active *replaces* button-event tracking with press/release-only.

Failure scenario: nvim in an attached couch actor enables `?1002h`. `Screen.Mouse()` → true, so `RouteMouseReport` forwards. The operator starts a drag-select; nvim redraws, `ch.batch.RowDirty` fires, `repaint()` → `paintNow()` → `\x1b[?1000;1006h`. The terminal drops to VT200 tracking, motion reports stop, and nvim sits in a mouse drag with no further events — the exact "stuck in visual selection" symptom `mouseinput.go:19-23` documents. `paintNow` is on the attention/bell/row-dirty path, so this fires continuously under a full-screen child, not once.

Fix sketch: do not re-assert unconditionally. Re-assert only when the child has *not* claimed the mode (`!child.Mouse()`), or re-assert in response to the observed DECRST transition rather than on every paint. Then pin it: mutating the re-assert out must redden a test that first has a child enable `?1002h`, paints, and asserts couch did not emit `?1000h`.

- Family: `unspecified-event-policy` — **this is the 3rd finding in that family** (BR-16 was the 2nd). Do NOT patch this one site. The rule the family has now asked for twice: **the disposition table must be crossed with the CHILD's mode-transition events, and every cell must state both what couch writes and what mode the child is left holding.** The enumeration is already written in BR-16 and is unchanged: child enables 1000; child enables 1002/1003; child disables; child exits with mouse on; switch between two children with different modes; replay re-asserting the child's modes over couch's. Measured prevalence this round: 1 of 6 cells implemented (the "child disables" cell), 0 of 6 pinned — deleting the paintNow re-assert leaves the entire couchtty suite green (verified), so even the cell that was implemented is unobserved. `RouteMouseReport`'s signature already takes `childWantsMouse`; the missing thing is that couch's *write* policy does not consult it at all.
- ARCH-ORDER, ARCH-MOCK. The claim the re-assert rests on is a fact about an external dependency (the terminal) with no fake modelling it and no conformance check; the plan's only instrument is Task 12 Step 2's manual nvim check, which has not run.

## 3. Important findings

**a. `menu_render.go:465-479` — the scrolled window is still unreachable from any test (BR-5, re-raised).** Replacing `index := len(lines)` with `index := start + ri + 2` leaves the whole suite green, because every extent test selects `inventory[0]` so `start` is always 0. Production is correct in the scrolled case (verified by a scratch fixture with `Frames[0].SelectedAddress = inventory[11]`: extents `h..l` at rows 2..12), but nothing pins it. Plan Task 4 Step 4 ("Add the scrolled case: mapping is against what is DRAWN, not the inventory index") and the Done-when both name it. Fix: set `state.Frames[0].SelectedAddress` to a thread below the window in `TestExtentsNeverPointPastTheDrawnMenu`, and assert the selected row maps to the selected thread.

**b. `core_concepts_contract_test.go:377-394` — `assertDirectTest` is satisfied by the contract's own file.** It globs `*_test.go` in the package and greps for the symbol; `core_concepts_contract_test.go` is in that glob and contains every symbol name verbatim inside the `conceptInventory` literal, so the "PURE row has a direct test" assertion passes for every row unconditionally. `seqMouse` is the demonstration: `grep -rn seqMouse cmd/internal/couchtty/*_test.go` returns exactly one hit — line 73 of the contract itself. Fix: skip `core_concepts_contract_test.go` in the glob (or require the match from a file other than the contract), then give whatever goes red a real test.
- Family: `guard-not-registered` — **this is the 3rd finding in that family** (BR-21 is the 2nd and is re-raised not-addressed). Do NOT fix this one guard. The rule, which now covers both instances: **an enumeration guard is satisfied only by an artifact the production path reads.** `hitHandlers()[HitMouse]` is a `func(){}` the dispatcher skips (`console.go:647`); `assertDirectTest` matches on the inventory that declares the requirement. Both report coverage they do not have, and both are one edit away for the next row or hit. Measured prevalence: 2 of 2 enumeration guards touched by this issue are self-satisfying.

**c. `menu.go:349-352` — the click arm is still silent where Enter explains (BR-17, partial).** `reduceRootKey`'s Enter arm sets `errorMenuNotice(thread.Label() + ": " + unusableThreadNotice(thread))` for a non-actionable row, with the comment "Silence is what the operator reports as a bug." The `MenuEventMouseSwitch` arm returns `next, nil`. Spec B says a click is "identical to pressing Return on that actor". Fold the refusal into `enterOperationFor`'s neighbourhood so both arms get the same answer, refusal included.

## 4. Minor findings

- `reserve.go:86-105` — `go doc ChipSpan` still prints "RenderStatusRow lays the model out in width columns" plus the untrusted-text rationale, and `go doc RenderStatusRow` prints nothing. The added meta-comment describes the fix instead of being it (BR-9, re-raised).
- `menu.go:200-211` — same shape, unchanged: `go doc MenuEventMouseSwitch` prints `MenuEventNotice`'s doc block, and `MenuEventNotice` has none (BR-19, re-raised).
- `run.go:581` — `type mousePressEvent = mouseinput.Event` has zero callers, three lines below a comment asserting "The move-time aliases that kept the diff small are gone" (BR-11, re-raised).
- `hostty/control.go:62` — `DisableMouseClicks` has zero production call sites; teardown uses `ResetInteractiveModes` (BR-20, re-raised).
- `keys.go:52,75` — `seqMouse` is never produced: `sequenceAt` only returns kinds from `knownSequences`, and its own comment says it cannot be there. Deleting `case seqMouse: return HitMouse` leaves the suite green. It exists to give the plan's Core-concepts table a row (BR-20's enumeration, re-raised).
- `termcmd/rename_input.go:175-186` — `sgrMouseSize` derives `Terminators` from `mouseinput` but restates the report-length rule (`idx + 3 + 1`), which `mouseinput.ParsePrefix` already answers as `len(raw)`. Second site knowing "where does this sequence end" — the thing the package doc says it exists to unify. **2nd in family `promoted-constant-with-surviving-literal`**; the rule: a promoted source is finished when every consumer derives the *rule*, not just the constant the diff happened to touch.
- `plan …-plan.md:193` — the Integration-points prose bullet still reads `hostty.MouseClickTracking`, seven lines below a Revisions entry recording that it became `EnableMouseClicks`/`DisableMouseClicks`. **2nd in family `plan-table-claims-unshipped-code`**; the rule: a rename in an artifact is finished when `grep <old-name>` returns zero, not when the machine-parsed row is fixed.
- `console.go:1024` — the re-assert is a second `writeOwn` per paint where it could be prefixed onto the existing `Reserve+PaintRow` string; one extra host write per child-output batch (ARCH-CONSTRAINTS, low cost, noted only because the paint path is per-batch).

## 5. Test coverage notes

Mutation results, all run and restored against `73294518`:

| mutation | result |
|---|---|
| `reserve.go:175` guard → `used-start >= Width(label)` | **RED** (2 tests) |
| delete extent run-merge (`menu_render.go:475-478`) | **RED** |
| `console.go:1461` drop `&& !origin.Manual` | **RED** |
| `console.go:1508` discard `child.child.Write(hit.Raw)` | **RED** |
| delete both `writeOwn(EnableMouseClicks)` | **RED** (2 tests) |
| delete **only** the paintNow re-assert | **GREEN** ← the BR-16 fix is unpinned |
| `clampExtents` → `return extents` | GREEN (documented, accepted) |
| extent index `len(lines)` → `start + ri + 2` | **GREEN** ← scrolled window untested |
| delete `case seqMouse: return HitMouse` | **GREEN** ← dead |

BR-14's four map rows now have tests and all reach production. `TestForwardPreservesRawBytes` feeds `\x1b[?1000h` through the fake child's real `Screen`, which is the right seam (ARCH-MOCK). The gap is the mode-*write* side: no test observes what couch emits to the host as a function of the child's mode.

## 6. Architectural notes

- **ARCH-DRY** — pass on the two headline extractions (`mouseinput`, `enterOperationFor`); flagged at `sgrMouseSize` (Minor).
- **ARCH-PURE** — pass. `ChipSpan`/`ColumnToActor`/`ActorExtent`/`PointToActor`/`RouteMouseReport` are total, terminal-free and directly tested; `onMouse` is the one place two coordinate bases meet and it is thin.
- **ARCH-PURPOSE** — flag. Shadow sweep of the promoted parser: `termcmd/run.go` ✓, `couchtty/keys.go` ✓, `couchtty/mouse.go` ✓, `termcmd/rename_input.go` ✗ (derives the constant, restates the rule). And the "child that did enable tracking still receives its own events unchanged" Done-when is under-delivered, not deferred — see the Critical.
- **ARCH-MOCK** — flag. The fact the re-assert depends on (terminal mouse-mode exclusivity) is external-dependency behaviour with no fake and no conformance check; Task 12 Step 2's manual nvim check is the only instrument and has not run. It should be run *before* the M2 close, and as written it will fail.
- **ARCH-CONSTRAINTS** — pass on the mode choice (`?1000` not `?1002`/`?1003`, with the reason recorded at `control.go:51-57`) and on the `MaxReport` liveness bound. Minor note on the extra per-paint write.
- **ARCH-SECURE** — pass. Reports are parsed to a typed value at the boundary and refused rather than clamped; negative and >223 coordinates land in no target (checked); an unparseable report degrades to literal input under a bound; untrusted labels are still stripped before spans are recorded.
- **ARCH-ORDER** — flag, twice: the child's mode transitions are still an unwritten state machine (the Critical), and `Interceptor.mouse` remains a payload valid only between one `FeedHit` and the next, kept correct by `processInput` calling `route(before)` before `it.Mouse()` — a hand-written ordering the type does not enforce.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md` covering:

1. **The disposition table gains a second axis.** It currently enumerates operator report events only. Add the child mode-transition rows (child enables 1000 / enables 1002 or 1003 / disables / exits with mouse on / switch between two children with different modes / replay re-asserts over couch's), each stating what couch writes and what mode the child ends up holding. Record that mouse-tracking modes are mutually exclusive, which is the fact the current "additive and idempotent" note gets wrong.
2. **`hostty.MouseClickTracking` at line 193.** The prose bullet still names it; the earlier Revisions entry only fixed the parsed row.
3. **Task 4 Step 4's scrolled case.** The Done-when → Task map claims it under `TestPointToActorSpansEveryLineOfAnActor`; that test does not scroll. Either record the delivery or move it to an explicit step.
4. **M2's review window.** The M2 production code sits inside the window this M1 gate reviewed. Folding M3 into M2 reduced the count of empty reviews but did not remove one — say plainly that M2's routing/ownership/wiring was reviewed at this boundary, so its own close is recorded as covered rather than treated as a fresh boundary.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      Row is now hostty.EnableMouseClicks/DisableMouseClicks and both symbols exist; the prose bullet at plan:193 still says MouseClickTracking, raised separately as Minor.
  - id: BR-4
    disposition: addressed
    note: |
      Verified: mutating reserve.go:175 to require the full label width reddens reserve_mouse_test.go:60 and :124.
  - id: BR-5
    disposition: not-addressed
    note: |
      Clamp half honestly dispositioned; the SCROLLED half is still unreachable -- the index mutation stays green because every extent test selects inventory[0].
  - id: BR-9
    disposition: not-addressed
    note: |
      go doc ChipSpan still prints RenderStatusRow's rationale and go doc RenderStatusRow prints nothing; the added comment describes the fix instead of being it.
  - id: BR-10
    disposition: addressed
    note: |
      run.go:453,459 now switch on mouseinput.WheelUp/WheelDown; no 64/65 literals remain.
  - id: BR-11
    disposition: not-addressed
    note: |
      parseSGRMousePressPrefix is gone but the mousePressEvent alias at run.go:581 still has zero callers, under a comment claiming the aliases were removed.
  - id: BR-13
    disposition: addressed
    note: |
      Verified: deleting the run-merge reddens TestClampAndRunMergeAreLoadBearing.
  - id: BR-14
    disposition: addressed
    note: |
      All three sites verified RED under mutation, and the four unmapped Done-when rows now have tests that reach production.
  - id: BR-15
    disposition: addressed
    note: |
      The tests were paid inside this window and this M1 review covers the M2/M3 code; residual is that M2's own close now opens on a near-empty range and should be recorded as covered here.
  - id: BR-16
    disposition: not-addressed
    note: |
      The re-assert landed but is unpinned (deleting it leaves the suite green) and it introduces a regression -- see the new Critical; the crossed enumeration was never written.
  - id: BR-17
    disposition: not-addressed
    note: |
      enterOperationFor and the effects-gated Manual flag both landed and mutate RED; the third half survives -- a non-actionable row still gets a notice from Enter and silence from a click.
  - id: BR-19
    disposition: not-addressed
    note: |
      Unchanged: go doc MenuEventMouseSwitch prints MenuEventNotice's block and MenuEventNotice has none.
  - id: BR-20
    disposition: not-addressed
    note: |
      WheelUp/WheelDown got production callers, but DisableMouseClicks and the mousePressEvent alias still have none, and seqMouse joins the enumeration as a new zero-reachability symbol.
  - id: BR-21
    disposition: not-addressed
    note: |
      console.go:647 still special-cases HitMouse before the table and hitHandlers()[HitMouse] is still an uncalled func(){} counted by the guard.
findings:
  - id: new
    severity: Critical
    family: unspecified-event-policy
    title: |
      The per-paint ?1000h re-assert demotes a child that enabled ?1002/?1003, breaking the "child keeps its own events unchanged" Done-when
    detail: |
      console.go:1024-1030 writes hostty.EnableMouseClicks on every paint. Mouse-tracking
      modes 1000/1002/1003 are one mutually-exclusive state in xterm (send_mouse_pos),
      Alacritty (removes MOUSE_MODE before inserting), kitty, Ghostty and iTerm2, so
      setting 1000 while a child holds 1002 replaces button-event tracking with
      press/release only. Screen.Mouse() still reports true, so couch keeps forwarding
      presses into a child that will never receive the motion that closes its drag --
      the "nvim stuck in visual selection" symptom mouseinput.go:19-23 documents.
      paintNow is on the rowDirty/bell/attention path, so a full-screen child triggers
      it continuously. The code comment and atlas/couch.md both assert the opposite.
      This is the 3rd finding in family unspecified-event-policy. Do NOT patch this one
      site. The rule the family keeps asking for: the disposition table must be crossed
      with the CHILD's mode-transition events, and each cell must state what couch
      writes AND what mode the child is left holding. Enumeration (unchanged from
      BR-16): child enables 1000; child enables 1002/1003; child disables; child exits
      with mouse on; switch between two children with different modes; replay
      re-asserting over couch's. Measured prevalence: 1 of 6 cells implemented, 0 of 6
      pinned -- deleting the paintNow re-assert leaves the whole couchtty suite green.
  - id: new
    severity: Important
    family: guard-not-registered
    title: |
      assertDirectTest is satisfied by the contract's own conceptInventory literal, so every PURE row's coverage assertion is vacuous
    detail: |
      core_concepts_contract_test.go:377-394 globs *_test.go in the package and greps
      for the row's symbol; the contract file is in that glob and holds every symbol
      name verbatim in conceptInventory, so the match always succeeds. seqMouse is the
      demonstration -- its only occurrence in any _test.go is line 73 of the contract
      itself, and seqMouse is dead code (sequenceAt can never return it; deleting
      `case seqMouse: return HitMouse` leaves the suite green).
      This is the 3rd finding in family guard-not-registered. Do NOT fix this one
      guard. The rule covering both instances: an enumeration guard is satisfied only
      by an artifact the production path reads. hitHandlers()[HitMouse] is a func(){}
      the dispatcher skips; assertDirectTest matches on the inventory that declares the
      requirement. Measured prevalence: 2 of 2 enumeration guards this issue touches
      are self-satisfying. Fix: exclude the contract file from the glob, then give
      whatever goes red a real test.
  - id: new
    severity: Minor
    family: promoted-constant-with-surviving-literal
    title: |
      sgrMouseSize derives Terminators from mouseinput but restates the report-length rule
    detail: |
      termcmd/rename_input.go:175-186 computes the report size as idx+3+1 after
      scanning for mouseinput.Terminators. mouseinput.ParsePrefix already answers that
      question as len(raw), and the package doc says it exists so there is ONE answer
      to "where does this sequence end". This is the 2nd finding in family
      promoted-constant-with-surviving-literal. Do NOT just rewrite this call. The
      rule: a promotion is finished when every consumer derives the RULE, not only the
      constant the diff happened to touch. Enumeration for this issue:
      termcmd/run.go:453,459 (fixed this round), rename_input.go:183 (constant derived,
      length rule restated). Note the two differ on malformed input -- ParsePrefix
      validates the numbers and sgrMouseSize does not -- so the sweep must state which
      behaviour wins.
  - id: new
    severity: Minor
    family: plan-table-claims-unshipped-code
    title: |
      The plan's Integration-points prose bullet still names hostty.MouseClickTracking after the Revisions entry renamed it
    detail: |
      workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md:193 reads
      "**hostty.MouseClickTracking**", seven lines below a Revisions entry recording
      that it became EnableMouseClicks/DisableMouseClicks. A reader grepping the plan
      for the symbol finds a name that exists nowhere in the tree.
      This is the 2nd finding in family plan-table-claims-unshipped-code. Do NOT edit
      only this line. The rule: a rename inside an artifact is finished when
      grep <old-name> over the artifact returns zero, not when the machine-parsed row
      is fixed -- the contract test reads the table, so the table is the only part the
      last fix was forced to get right.
```

---

## Re-review — 2026-09-05T20:53:23-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..a43d71e23ab69f8fd6d8e705a2a92c73b43c9efb |
| command | sdlc milestone-close --issue 172 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-05T20:53:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The geometry M1 owns is genuinely good and this round's Critical fix is real: re-asserting `?1000h` unconditionally now reddens `TestCouchDoesNotDemoteAChildsTrackingMode` (verified by mutation), and the `mouseinput` promotion, the bounded Interceptor hold, the raw-byte forward and `!origin.Manual` are all pinned. What blocks the boundary is that three of this round's claimed fixes are unpinned by the very rule the round was opened to enforce — deleting the whole paint-time re-assert block (`console.go:1040`), removing the `len(effects) > 0` guard (`menu.go:362`), and hardcoding `"switch"` at the click's call site (`menu.go:357`) each leave the entire `couchtty` suite green — and that `atlas/couch.md:356-361` still states the exact claim BR-22 refuted ("couch RE-ASSERTS its own on every paint … DECSET is additive and idempotent, so this cannot clobber a mode the child set for itself"), which is the sentence that produced the Critical in the first place. Every mutation below was applied to the working tree and reverted; `git status` is clean at `a43d71e2`, and the only test failures in `go test ./...` are the pre-existing pty "operation not permitted" ones.

## 1. Strengths

- **`cmd/internal/couchtty/console.go:1024-1042`** — the BR-22 fix is correct *and* argued. The comment states both facts that have to hold together (a child's DECRST kills couch's clicks; 1000/1002/1003 are one mutually-exclusive state) and names the trade it chose. Mutation: replacing the guarded write with an unconditional one → `TestCouchDoesNotDemoteAChildsTrackingMode` **RED**.
- **`cmd/internal/couchtty/reserve_mouse_test.go:104-133`** — `TestAChipClippedToOneColumnIsStillClickable` is a *constructed* case (width 8 chosen so "beta" draws exactly one column), not a sweep. This is the right answer to the previous round's one-directional-assertion finding, and `menu_extent_test.go:126` does the same for the run-merge: deleting the merge → **RED**.
- **`cmd/internal/couchtty/core_concepts_contract_test.go:414-424`** — excluding the contract file from its own glob is a real fix to a guard that was matching its own inventory, and it was followed through: the three types it unmasked (`RenderedStatusRow`, `ActorExtent`, `endsItsOwnChild` — the last shipped two issues ago) all got direct tests, and dead `seqMouse` was deleted rather than papered over.
- **`cmd/internal/mouseinput/`** — a genuine move, not a copy: `termcmd` imports back, its own suite is unedited except for renames, and the new package carries tests `termcmd` never had (the >223-coordinate case that is the whole reason SGR is required).
- **`cmd/internal/couchtty/keys.go:282-306` + `keys_test.go:551-601`** — the `#127` bound is pinned directly: `\x1b[<` + 200 digits releases as literal input and the keystrokes behind it still arrive.

## 2. Critical findings

None open. BR-22's code defect is fixed and verified red.

## 3. Important findings

**BR-16 re-raised — the re-assert this round added is unpinned.** `cmd/internal/couchtty/console.go:1039-1041`. Deleting `if !c.childWantsMouse() { c.writeOwn(hostty.EnableMouseClicks) }` outright leaves the whole suite green. Only the *negative* half has a test; nothing drives the transition the finding is about — child writes `\x1b[?1000l`, `Screen.Mouse()` goes false, next paint re-asserts. The family's enumeration is still unwritten: measured, 2 of 6 cells are pinned (child enables 1002/1003 → don't demote; child holds 1000 → forward). Child disables, child exits with mouse on, switch between two children with different modes, and replay re-asserting over couch's have neither a cell nor a test. `TestCouchDoesNotDemoteAChildsTrackingMode` already shows the fixture is cheap — `child.Feed([]byte("\x1b[?1000l"))`, `host.Reset()`, `con.repaint()`, assert `EnableMouseClicks` reappears.

**BR-17 re-raised — two of its three parts survive, and the third is unpinned.** `cmd/internal/couchtty/menu.go:350-365`.
- Replacing `enterOperationFor(thread)` with the literal `"switch"` at line 357 leaves the suite green. The extraction landed, but no test clicks a *resumable* row, so the shared authority is unpinned at the one call site the finding was about. (`TestReduceRootKeyEnterRoutesByRowState` covers the Enter arm only.)
- `state.InFlight.Manual = true` set unconditionally leaves the suite green — the `len(effects) > 0` guard has no test.
- The notice divergence is untouched: `menu.go:351-353` returns silently for a non-actionable row, while the Enter arm at `menu.go:466-471` sets `errorMenuNotice(thread.Label() + ": " + unusableThreadNotice(thread))` under a comment that says "Silence is what the operator reports as a bug."

**BR-5 re-raised — the scrolled half.** `cmd/internal/couchtty/menu_render.go:465-479`. The `clampExtents` half is answered honestly (documented unreachable, measured across heights 3..16, with the test saying why it is absent) and I accept it. The scrolled half is not. A probe (`panic("SCROLLED")` at `menu_render.go:465`) confirms `TestExtentsNeverPointPastTheDrawnMenu` *does* render a scrolled list — but changing `index := len(lines)` to `index := len(lines) + start` still leaves the suite green, because `clampExtents` silently absorbs the damage and the test only checks in-range plus self-consistency (`PointToActor(extent.Start)` maps back to `extent.Thread`, which is true by construction). The Done-when names "on a scrolled or clipped list" explicitly; assert a scrolled row against the *drawn text* the way `TestPointToActorSpansEveryLineOfAnActor` does.

**BR-22 re-raised — the atlas still asserts what the code no longer does.** `atlas/couch.md:356-361`. The finding named `atlas/couch.md` alongside the code comment; the comment was corrected and the atlas was not. It now says couch "RE-ASSERTS its own on every paint" (false since `console.go:1039`) and repeats "DECSET is additive and idempotent, so this cannot clobber a mode the child set for itself" — the sentence the finding refuted with five terminal implementations. This is the durable artifact the next implementor reads before touching mouse mode.

## 4. Minor findings

- **BR-9 re-raised.** `go doc ChipSpan` still prints RenderStatusRow's untrusted-text rationale; `go doc RenderStatusRow` prints nothing. The fix prepended a paragraph *to the same misattributed block* (`reserve.go:86-90`) explaining that the block used to be misattributed — it still is. Verified with `go doc`.
- **BR-19 re-raised.** `go doc MenuEventMouseSwitch` still leads with `MenuEventNotice`'s three lines (`menu.go:199-207`); `MenuEventNotice` now has no doc at all. The rule both instances need is checkable: `go doc <symbol>` for every symbol this diff adds.
- **BR-11 / BR-20 re-raised.** `parseSGRMousePressPrefix`, `seqMouse` and the `WheelUp`/`WheelDown` literals are done. Still at zero references: `type mousePressEvent = mouseinput.Event` (`termcmd/run.go:581`), sitting directly beneath a comment that says "The move-time aliases that kept the diff small are gone", and `hostty.DisableMouseClicks` (`hostty/control.go:62`). The rule is stated but unenforced — `deadSymbolScope` (`artifactpath/deadsymbols_test.go:19`) is still the single const `"cmd/internal/couchcore"`, so no guard can see either. Widening it to a list covering `couchtty`/`hostty`/`termcmd`/`mouseinput` is the fix that ends the family.
- **BR-21 re-raised.** `console.go:647` still special-cases `HitMouse` before the table and `hitHandlers()[HitMouse]` is still `func() {}`. Both halves measured: deleting the entry reddens `TestEveryInterceptedChordHasAHandler` (the guard proves *presence*), while forcing dispatch through the table (`hit == HitMouse && false`) reddens four click tests (the registered value is provably never read).
- **BR-24 re-raised.** `termcmd/rename_input.go:186` still computes the report length as `idx + 3 + 1` after deriving only the terminator set from `mouseinput`. `ParsePrefix` already answers it as `len(raw)`, and the two still differ on malformed input.
- **BR-25 re-raised.** `plan:193` still reads `hostty.MouseClickTracking`, and the same rule has a second symbol: `seqMouse` is struck through in the table (`:118`) but named as shipped in the prose at `:165`, `:394` and `:396`.

## 5. Test coverage notes

Mutation results against `./cmd/internal/couchtty` + `./cmd/internal/termcmd` + `./cmd/internal/mouseinput` (pty failures excluded; tree restored after each):

| mutation | site | result |
|---|---|---|
| delete the guarded paint-time re-assert | `console.go:1039` | **green** |
| re-assert unconditionally | `console.go:1039` | RED (1) |
| set `Manual` unconditionally | `menu.go:362` | **green** |
| hardcode `"switch"` at the click | `menu.go:357` | **green** |
| drop `&& !origin.Manual` | `console.go:1474` | RED (1) |
| `enterOperationFor` → always `"switch"` | `menu.go:604` | RED (5, Enter arm only) |
| extent index += scroll `start` | `menu_render.go:466` | **green** |
| `clampExtents` → identity | `menu_render.go:200` | **green** (documented unreachable) |
| drop the extent run-merge | `menu_render.go:475` | RED (1) |
| remove `HitMouse` from the table | `console.go:1577` | RED (1, presence only) |
| bypass the `HitMouse` special case | `console.go:647` | RED (4) |

Gaps worth closing beyond the findings: no test clicks a **parked or detached** row (the `resume` branch of the click path is entirely uncovered), and no test exercises a click while another operation is in flight (the `len(effects) > 0` path).

## 6. Architectural notes

- **ARCH-DRY — flag.** One parser is real. Three restatements remain: `sgrMouseSize`'s length rule (BR-24), the click arm's actionable/notice decision against the Enter arm's (BR-17), and two zero-reference symbols the promotion left (BR-11/BR-20).
- **ARCH-PURE — pass.** `reserve.go`, `menu_render.go` and `mouse.go` import no IO seam; `assertPureSource` enforces it and the new rows are inside its scope. `Console.onMouse` is a thin caller over `RouteMouseReport` + two total hit-tests.
- **ARCH-PURPOSE — flag.** Three families recurred again this round, each answered at the instance: `orphaned-doc-comment` (BR-9 fixed *in place* without moving the block; BR-19 untouched), `move-residue` (3 of 5 symbols swept, rule unenforced), `unspecified-event-policy` (one cell implemented, enumeration unwritten). The enumerations these findings ask for are cheap and none has been written down.
- **ARCH-MOCK — pass, with one note.** `hostty.FakeHost` and `ptychild.NewFakeChild` sit on the same seam production uses, and `child.Feed([]byte("\x1b[?1002h"))` drives the real DECSET scanner rather than stubbing `Mouse()`. What has no check at any cadence is the claim the Critical fix rests on — that real terminals *replace* rather than union 1000/1002/1003. Task 12 Step 2's manual verification is the declared answer and has not been recorded in `## Log` yet.
- **ARCH-CONSTRAINTS — pass.** `?1000` not `?1002`/`?1003`; `MaxReport` bounds the hold; the per-paint mode write is 12 bytes and now gated.
- **ARCH-SECURE — pass.** The report is parsed into a typed value at the boundary and refused (not clamped) when it does not fit; `ColumnToActor`/`PointToActor` are total; label sanitisation survives the signature change.
- **ARCH-ORDER — flag.** `Interceptor.mouse` (`keys.go:236`) is a payload valid only between one `FeedHit` and the next, kept correct by hand-written ordering rather than by the type — which is what the plan's own PQ-3 bullet promised to avoid by widening `FeedHit`'s return. And couch's mouse state across child mode-transitions is still a pair of booleans read at paint time rather than a written `(state, event) → (state, effects)` table (BR-16).
- **For M2's close:** M2's production code landed inside this window, so its boundary review will open on a near-empty range. The substance is reviewed here, but three M2 Done-when items are at risk of never being checked at any gate — the manual nvim selection/scroll verification, the `pair#166` re-evaluation, and the "child exits with mouse on" recovery. Worth carrying them into the M2 close explicitly rather than relying on its window.

## 7. Plan revision recommendations

A `## Revisions` entry in `workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md` covering:

- **`:193`** — `hostty.MouseClickTracking` → `hostty.EnableMouseClicks` / `DisableMouseClicks`, and `:165`, `:394`, `:396` — `seqMouse` was deleted, not shipped. The rule the last fix missed: the rename is finished when `grep <old-name>` over the artifact returns zero, not when the machine-parsed row is fixed.
- **`:186-191`** — the PQ-3 bullet claims "`FeedHit` therefore returns the decoded event alongside the hit, and `processInput` passes it". It does not: `FeedHit`'s signature is unchanged and the payload is read from the mutable `Interceptor.mouse` via `Mouse()`. Record what shipped and why, since that choice is what BR-21 is about.
- **Task 4 Step 4** ("Add the scrolled case: mapping is against what is DRAWN, not the inventory index") — either deliver it or record that it is deferred with a reason; today the step is unticked and the behaviour is unpinned.
- **Core concepts** — `RenderedMenu` gained an exported field (`Extents`) in this window and has no row; add it as `modified`, or state that `PointToActor`'s row covers it.

---

## Re-review — 2026-09-05T22:22:18-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..079a646156b6830342f14c5d5582b880ea06aad1 |
| command | sdlc milestone-close --issue 172 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-05T22:22:18-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Critical from last round is genuinely fixed and I verified it in **both** directions by mutation in a scratch checkout of the pinned head: making the per-paint write unconditional reddens 5 subtests (`TestCouchDoesNotDemoteAChildsTrackingMode`, three `TestMouseModeTransitions` cells, `TestSwitchingToAChildWithoutTrackingReturnsTheTerminalToCouch`), and deleting it reddens 4 the other way. BR-21 and BR-23 are likewise real: no-op'ing the `HitMouse` table entry reddens three console tests, and removing `mouse_test.go` now correctly reddens the `MouseDisposition`/`RouteMouseReport` contract rows. What holds SHIP back is smaller and cheap: the **scrolled** switcher list is still entered by no test at all (`index := len(lines) + start*1000` leaves the whole package green — `start` is 0 in every fixture), which is a Done-when bullet and Plan Task 4 Step 4; the BR-9 doc fix was written *about* rather than *performed* (`go doc ChipSpan` still prints `RenderStatusRow`'s rationale, and `RenderStatusRow` has no doc at all); BR-17's `enterOperationFor` extraction and its `len(effects) > 0` guard are both unpinned (reverting either leaves the suite green); and `atlas/couch.md:356-361` still teaches the exact refuted model — "DECSET is additive and idempotent, so this cannot clobber a mode the child set for itself" — that produced BR-22.

One window note: the repo is at `7c444884`, one commit past the pinned head `079a6461`. That commit fixes BR-19 (menu.go) and adds two lessons; I reviewed the pinned range and checked `7c444884` separately where a disposition depended on it.

## 1. Strengths

- **`TestMouseModeTransitions` (`console_mouse_test.go:319-368`) is the right shape for the family that kept re-firing.** One rule (the child's mode wins whenever it has one) producing six cells as a table, driven through the real fixture, with the "child exits" case going through `onExit` rather than deleting the map entry. Mutation-verified both directions.
- **`TestClickIsAManualSwitch` observes the capture *at dispatch*** (`console_mouse_test.go:150-205`), which is the only moment it exists — the comment explaining why reading `InFlight` after the dispatch would pass for the wrong reason is exactly right, and removing `&& !origin.Manual` from `console.go:1478` reddens it.
- **The `assertDirectTest` self-exclusion (`core_concepts_contract_test.go:417-424`) is a real repair, not a formality.** I removed `mouse_test.go` and the guard reported both PURE rows uncovered. Unmasking it also found and fixed three genuinely untested types, one from `pair#182`.
- **`TestAChipClippedToOneColumnIsStillClickable`** and **`TestClampAndRunMergeAreLoadBearing`** correctly answer the "sweeps assert what they find" critique by *constructing* the interesting case; deleting the run-merge reddens the latter.
- **The `mouseinput` promotion is a clean single source.** `WheelUp`/`WheelDown` now have production call sites (`run.go:453,459`), `MaxReport` bounds the Interceptor hold (`keys.go:297`), `sgrMouseSize` derives the length from `ParsePrefix` instead of restating the framing rule, and `DisableMouseClicks` / the `mousePressEvent` alias / `parseSGRMousePressPrefix` are gone rather than left as surface.

## 2. Critical findings

None.

## 3. Important findings

**A. `cmd/internal/couchtty/menu_render.go:467` — the scrolled list is entered by no test, so a scroll-offset bug in the extents ships green.** Measured: replacing `index := len(lines)` with `index := len(lines) + start` (identity when unscrolled) leaves the *entire* `couchtty` package green; even `+ start*1000` passes. `TestExtentsNeverPointPastTheDrawnMenu`'s 12-actor fixture never scrolls, because the scroll window follows the selection and the selection stays on row 0. The Done-when requires "a scrolled or clipped list" and Plan Task 4 Step 4 says "Add the scrolled case: mapping is against what is DRAWN, not the inventory index." Fix sketch — the state is cheap to construct and I confirmed the extents are *correct* today, so this is purely coverage: build 20 live actors, drive 18 `MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyDown}}` through `ReduceMenu`, render at height 12 (window opens on `j`), then assert each drawn line's text maps to the actor named in it — the same text-matching assertion `TestPointToActorSpansEveryLineOfAnActor` already uses.

**B. `cmd/internal/couchtty/menu.go:352` and `:357` — BR-17's fix is unpinned in both halves, and the notice divergence it named is still there.** Reverting `enterOperationFor(thread)` to a hardcoded `"switch"` in the click arm leaves the whole suite green (the switcher fixture has only `ThreadLive` rows, so `Resumable()` is never true through a click); and making `state.InFlight.Manual = true` unconditional also leaves it green, so the "only mark a dispatch that HAPPENED" guard has no test. Separately, a non-actionable row still gets a notice from Enter (`menu.go:468-471`) and silence from a click (`menu.go:352-354`) — the third item BR-17 named. Fix sketch: add a parked/detached row to the switcher fixture and assert a click dispatches `resume`; assert a click while another operation is in flight leaves `Manual` false; and route the click's non-actionable case through the same `errorMenuNotice(thread.Label() + ": " + unusableThreadNotice(thread))` Enter uses.

**C. `atlas/couch.md:356-361` — the atlas documents the policy that was replaced and the terminal fact that was refuted.** It reads "couch RE-ASSERTS its own on every paint … DECSET is additive and idempotent, so this cannot clobber a mode the child set for itself." Both halves are now false: `console.go:1040-1045` re-asserts only while `!c.childWantsMouse()`, precisely *because* 1000/1002/1003 replace one another. `22de4ceb` and `079a6461` touched `console.go`, tests, `lessons.md` and the gate ledgers, and left the atlas alone. **This is the 2nd finding in family `docs-lag-shipped-surface`** (BR-18 was the first). Do not patch only this paragraph — the rule the family is asking for: *when a boundary changes a behaviour the atlas already describes, the same commit rewrites the contradicted paragraph, not just appends the new surface; the check is `grep` the atlas for the mechanism the diff changed, not "did I add a section".* Measured prevalence this round: 1 of 1 atlas paragraph that the mode-policy change contradicted was left standing, while a brand-new section for the same feature was added in the same window.

**D. `cmd/internal/couchtty/console.go:530` and `hostty/control.go:58` — couch's mode ownership has three writers and the stand-back policy governs one of them.** `?1006` (SGR encoding) is set together with `?1000` and is *never* stood back from: when the policy hands the terminal to a child, `?1006h` stays on, so a child that enables `?1002h` alone — expecting X10 reports — receives SGR reports it did not ask for and cannot parse, against the "receives its own events unchanged" Done-when. The startup write at `:530` is gated by neither the policy nor `childWantsMouse()`, and `release()` writes unconditionally *before* `c.started = false` (`console.go:794-806`), so a paint racing teardown can re-enable tracking in the operator's shell. Also: the rationale comment at `console.go:1036-1038` ("couch loses its own clicks for as long as that child is attached") is not true — `RouteMouseReport` still routes button-0 on the last row to `MouseCouch`, and the reports still arrive because `?1006` is still on. **This is the 4th finding in family `unspecified-event-policy`.** Do not add a fourth site fix. The rule: *couch's mode ownership is one gated writer over the full set of modes it touches — the tracking mode and the encoding mode — and every cell of (child mode transition × couch write) states what couch writes AND what mode-and-encoding the child is left holding.* Measured prevalence: 3 writers, 1 governed by the policy; 2 modes written, 1 governed.

**E. `workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md:41-56, :161, :182` — the plan artifact disagrees with the tree in three more places, with no `## Revisions` entry.** (i) The Integration-points row at `:182` declares `hostty.EnableMouseClicks` / **`DisableMouseClicks`** status `new`; `DisableMouseClicks` was deleted in `079a6461` and exists nowhere. The declared path is `cmd/internal/hostty/`, outside `conceptPackage`, so the contract test cannot see it — the same blind spot BR-3 exploited. (ii) The section titled "**The complete** disposition table" (`:41-56`) has no `couchOwnsScreen` dimension, so it is missing the two switcher rows the code and `mouse.go`'s own doc carry, and it still says "anything, elsewhere | yes | forward verbatim" which the switcher case contradicts. (iii) `:161` states the signature as `(event, hostRows, childWantsMouse)`; the shipped one takes a fourth argument. Also: all 43 plan checkboxes are `- [ ]`, including M1's Tasks 1-5 whose work demonstrably shipped. **This is the 3rd finding in family `plan-table-claims-unshipped-code`.** The rule, covering this and BR-25: *before the boundary tick, diff the plan against the tree in both directions — every backticked symbol in the artifact resolves in the tree, and every signature/table the artifact calls "complete" is re-derived from the code — and each delta gets a `## Revisions` entry. Fixing the machine-parsed row alone is fixing the part the test forces.*

## 4. Minor findings

- `reserve.go:86-105` — the BR-9 fix added a paragraph *explaining* the defect above the still-misplaced block; `go doc ChipSpan` prints "The untrusted-text rationale below belongs to RenderStatusRow… RenderStatusRow lays the model out in width columns…", and `go doc RenderStatusRow` prints nothing. `lessons.md:3479` now records this as fixed in the past tense.
- `menu_render.go:181-195` — `clampExtents` is honestly documented as unreachable, but it is still a branch no test and no production input can enter; consider deleting it and keeping the invariant as an assertion inside the loop that emits extents.
- `artifactpath/manifest.go:549,557` — `cmd/internal/mouseinput/mouseinput.go` is inserted inside the `couchcore` block and `couchtty/mouse.go` before `couchtty/menu.go`, breaking the list's alphabetical order.
- `termcmd/run.go:580,584` — `findSGRMousePress` and `isSGRMousePrefix` are now one-line pass-throughs; they have call sites, but they add a second name for one rule.
- `mouseinput.go:50` — `s == ""` is unreachable after `strings.HasPrefix(s, "\x1b[<")` succeeds.
- `console_mouse_test.go:141` — `TestChildWithoutTrackingReceivesNoMouseBytes` inspects only `w[0] == 0x1b`, so a report concatenated behind other bytes would pass; the split test covers this today, but the assertion is weaker than its name.

## 5. Test coverage notes

Everything I could mutate, I mutated against a scratch checkout of `079a6461`. Confirmed load-bearing: the per-paint mode guard (both directions), the manual-switch capture suppression, the `HitMouse` table entry, the extent run-merge, the `assertDirectTest` self-exclusion, the chip-span clip behaviour. Confirmed **not** load-bearing: the scrolled extent index (finding A), `enterOperationFor` in the click arm and the `len(effects)` guard (finding B), `clampExtents` (acknowledged). The pty-dependent suites (`TestNotificationPTYConformance`, `ptychild`, `couchcore` pty subtests, `hostty` OSHost, `keyscmd` shim) fail with "operation not permitted" in this environment — that is the known sandbox limitation, not this diff; every non-pty package is green at the pinned head.

## 6. Architectural notes

- **ARCH-DRY — pass with one flag.** One parser, one operation-choice function, spans from the clipping pass. Flagged in finding B: the extraction is a convention, not an invariant, because nothing reddens when the click arm restates it.
- **ARCH-PURE — pass.** `reserve.go`, `menu_render.go`, `mouse.go`, `keys.go` import no IO seam and their tests run without one; `Console.onMouse` is a genuinely thin adapter that converts 1-based→0-based once and delegates. The contract test enforces the import ban.
- **ARCH-PURPOSE — flag.** The `mouseinput` shadow-sweep passes: every consumer (`run.go`, `rename_input.go`, `keys.go`) derives from the package, and the surviving literals are gone. The remaining hand-maintained restatements of the model are the plan (finding E) and the atlas (finding C). Finding A is the Done-when's own words deferred.
- **ARCH-MOCK — flag (Minor).** `hostty.FakeHost` and `ptychild.FakeChild` are real stateful doubles on the same seam production uses, and `TestFakeChildConformsToRealChildLifecycle` exists. But the fact the whole mode policy rests on — that setting `?1000h` *replaces* a child's `?1002h` — is modelled nowhere executable: `FakeChild.Mouse()` is a bool, `FakeHost` records bytes, so the tests assert "couch did not write" rather than "the child's mode survived", and there is no live conformance check against a real terminal. **2nd finding in family `doc-ahead-of-enforcement`.** The rule: *a dependency behaviour the design depends on is modelled in the fake so a test can exhibit its violation, or covered by a scheduled live conformance check; a comment and a lesson entry asserting it are neither.*
- **ARCH-CONSTRAINTS — pass.** `?1000` not `?1002/1003` is honoured and documented; the `MaxReport` bound on the Interceptor hold is enforced and tested; the per-paint mode write is 12 bytes and a mutex, negligible against the row repaint it rides on.
- **ARCH-SECURE — pass.** The report is parsed into a typed `mouseinput.Event` at the boundary and refused rather than clamped; `ColumnToActor`/`PointToActor` are total and return false for negative, gap and past-the-end columns (I checked negative and far-right explicitly in `TestColumnToActorIsTotal`); an unparseable report degrades to ordinary input rather than a fabricated coordinate. Untrusted labels are still stripped before layout.
- **ARCH-ORDER — flag.** The mouse payload now lives in *two* mutable slots — `Interceptor.mouse` (`keys.go:236`) and `Console.mouseHit` (`console.go:135`) — each valid only between one `FeedHit` and the next, correct by hand-written ordering rather than by the type. This is BR-21's second half, unaddressed by design; the durable answer is still to widen `hitHandlers()`'s value to `func(MouseHit)` so the payload travels with the dispatch. Also under this lens: `menuExtents` is a snapshot of the last *menu* paint but is guarded only by `focus.IsPanel()`, so "panel focused" and "panel drawn" are two facts kept in agreement by convention. And see finding D's teardown race — the mode write is now on the paint path, which is not lifecycle-guarded.

Process note (not a new finding — BR-15 covered it and was disposed): all of M2's production code is inside this M1 window, so M2's own boundary review will open on a docs-and-log-only range. The `## Revisions` entry merged M3 into M2 but the M1/M2 split still has the shape BR-15 named.

## 7. Plan revision recommendations

- `### 2026-09-05 — the disposition table gained a dimension it does not show.` Record that `RouteMouseReport` takes a fourth argument, `couchOwnsScreen`, and add the two switcher rows (button-0 press → couch; anything else → swallow) to the table at `:41-56`, correcting "anything, elsewhere | yes | forward verbatim" to say it holds only while a child is displayed. Say why: the panel branch was unreachable until a test clicked a row, which `mouse.go:59-63` records.
- `### 2026-09-05 — DisableMouseClicks was deleted, not shipped.` Change the Integration-points row at `:182` to `hostty.EnableMouseClicks` only, and note that the row's declared path is outside `conceptPackage` so no guard can catch a wrong symbol there — the third artifact row this issue has had wrong for that reason.
- `### 2026-09-05 — the seqMouse rename sweep.` `:165`, `:394`, `:396` still name `seqMouse` as shipped work and `:396` names `mouseinput.IsSGRPrefix`, which is `IsPrefix`; `:155` names `mouseinput.FindSGR`, which is `Find`. Finish the rule BR-25 stated: `grep` the artifact for each old name until it returns zero.
- `### 2026-09-05 — M1's tasks shipped unticked.` Tick Tasks 1-5's steps, or state in the entry which ones were superseded (Task 5's `conceptPlans` registration landed; Task 4 Step 4's scrolled case did not — see finding A).
- Add a line under **Non-goals** or the ARCH-SECURE section recording that couch's `?1006` is not stood back from when the child holds a tracking mode, and what a child that enabled `?1000h` without `?1006h` is expected to see.

---

## Re-review — 2026-09-05T22:40:16-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..7c444884162cdeb2373974424f7f39f9203cbdfa |
| command | sdlc milestone-close --issue 172 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-05T22:40:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The mouse-mode Critical from the last two rounds is genuinely fixed and I mutation-verified it in both directions in a scratch checkout of the pinned head (making the per-paint write unconditional reddens 4 subtests; deleting it reddens 4 different ones), and `BR-21`/`BR-23`/`BR-20`/`BR-24`/`BR-11`/`BR-19` all hold up under the same treatment. What keeps this off SHIP is that four prior findings were answered without a test that fails without the answer, and one new behaviour gap survives the mode fix: `BR-5`'s **scrolled** switcher list is still entered by no test (`index := len(lines) + start*1000` leaves the entire `couchtty` package green), `BR-17`'s `enterOperationFor` extraction *and* its `len(effects) > 0` guard are both revertible with no red, `BR-9` was answered by writing a paragraph *about* the misplaced doc block rather than moving it (`go doc ChipSpan` still prints `RenderStatusRow`'s rationale), and couch's mode ownership writes two modes but governs one — I confirmed by probe that a child holding `?1002h` which clears only the *encoding* (`?1006l`) gets demoted on the next paint, the exact BR-22 mechanism reached through the observation instead of the write. None of these is a crash or a broad regression, so the gate can be crossed once they are disposed.

## 1. Strengths

- **`TestMouseModeTransitions` (`console_mouse_test.go:305-368`) is the right answer to the family that kept re-firing** — one rule ("the child's mode wins whenever it has one") expanded into six cells as a table, with the child-exit case driven through `onExit` rather than by deleting a map entry. I reverted the guard both ways and it reddens correctly in both directions.
- **`assertDirectTest`'s self-exclusion (`core_concepts_contract_test.go:417-424`) is a real repair.** I removed `mouse_test.go` and both `MouseDisposition` and `RouteMouseReport` reported uncovered — the guard now fails for the reason it exists.
- **`HitMouse` really routes through `hitHandlers()` now** (`console.go:1584`). Replacing the entry with `func() {}` reddens four console tests, so `AllInterceptorHits` counts a hit the dispatcher actually reaches.
- **The `mouseinput` promotion is a clean single source.** `WheelUp`/`WheelDown` have production call sites (`run.go:453,459`), `MaxReport` bounds the Interceptor hold (`keys.go:297`) and is tested directly, `sgrMouseSize` derives the report length from `ParsePrefix` instead of restating the framing rule, and every move-time alias is deleted rather than left as surface.
- **`TestAChipClippedToOneColumnIsStillClickable` and `TestClampAndRunMergeAreLoadBearing` construct the interesting case** instead of sweeping and asserting whatever turns up — deleting the extent run-merge reddens the latter.

## 2. Critical findings

None.

## 3. Important findings

**A. `cmd/internal/couchtty/console.go:528,1040` + `cmd/internal/ptychild/screen.go:415` — couch writes two mouse modes and governs one, and the fact it reads conflates them.** Two probes against the pinned head, both confirmed:

- A child that does `\x1b[?1002h` then `\x1b[?1006l` (clears only the SGR encoding) flips `Screen.Mouse()` to false — `screen.go:415` maps `1000/1002/1003/1006` onto one `s.mouse` bool — so the next paint writes `?1000;1006h` and demotes it to press/release. That is BR-22's symptom reached through the *observation* rather than the write.
- A child that does `\x1b[?1006h` alone (encoding, no tracking) makes couch stand back permanently for a child that enabled no tracking at all.

Two more halves of the same gap: `?1006` is never stood back from, so a child that enables `?1000h` *without* `?1006h` receives SGR reports it did not ask for and cannot parse — against the "receives its own events unchanged" Done-when; and the startup write at `:528` is gated by neither `childWantsMouse()` nor anything else. Separately, the take-back is evaluated **only when something else repaints**: `onChunk` repaints on `RowDirty`/`Bell`/attention, and the DECSET handler sets `rowDirty` only for alt-screen modes, so a child's bare `?1000l` leaves couch's clicks off for an unbounded time. `TestMouseModeTransitions` cannot see this because it calls `con.repaint()` by hand and its `FakeChild` has a nil sink.

**This is the 4th finding in family `unspecified-event-policy`.** Do not add a fifth site fix. The rule: *couch's mouse ownership is ONE gated writer over the full set of modes it touches — tracking AND encoding — driven by an observation that keeps them as separate state, and every cell of (child mode transition × couch write) states what couch writes, what the child is left holding, and what EVENT causes the cell to be re-evaluated.* Measured prevalence: 3 writers (`:528` startup, `:1040` per-paint, `release()`), 1 governed; 2 modes written, 1 governed; 1 observation collapsing 4 modes into 1 bool; 6 pinned cells, 0 of which pin the trigger. Fix sketch: split `ptychild.Screen`'s mouse state into `Tracking (none|1000|1002|1003)` and `SGREncoding bool` (`termcmd.appMouseMode()` is the other consumer and wants the tracking half), model the same split in `FakeChild` so a test can exhibit a demote, and make the transition drive a repaint rather than riding on one.

**B. `cmd/internal/couchtty/menu_render.go:467` — the scrolled switcher list is entered by no test, so a scroll-offset bug in the extents ships green.** Measured: `index := len(lines) + start*1000` leaves the *entire* `couchtty` package green. `TestExtentsNeverPointPastTheDrawnMenu`'s 12-actor fixture never scrolls, because the scroll window follows the selection and the selection stays on row 0, so `start` is 0 in every fixture in the package. The Done-when requires "a scrolled or clipped list" and Plan Task 4 Step 4 says "Add the scrolled case: mapping is against what is DRAWN, not the inventory index" — this is M1's own unticked step. The extents are *correct* today; this is purely coverage. Fix sketch: 20 live actors, drive ~18 `MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyDown}}` through `ReduceMenu` so the window opens, render at height 12, then assert each drawn line's *text* maps to the actor named in it — the same text-matching assertion `TestPointToActorSpansEveryLineOfAnActor` already uses. (Disposed as `BR-5` not-addressed.)

**C. `cmd/internal/couchtty/menu.go:352,357` — BR-17's fix is unpinned in both halves and the third divergence it named is still open.** Reverting `enterOperationFor(thread)` to a hardcoded `"switch"` in the click arm leaves the whole suite green — the switcher fixture holds only `ThreadLive` rows, so `Resumable()` is never true through a click; and making `state.InFlight.Manual = true` unconditional also leaves it green, so "only mark a dispatch that HAPPENED" has no test. A non-actionable row still gets `errorMenuNotice` from Enter (`menu.go:468-471`) and silence from a click (`:352-354`), which is not "exactly the path Return takes". Fix sketch: add a parked/detached row to the switcher fixture and assert a click dispatches `resume`; assert a click while another operation is in flight leaves `Manual` false; route the click's non-actionable case through the same notice Enter uses. (Disposed as `BR-17` not-addressed.)

**D. `atlas/couch.md:356-361` — the atlas teaches the model that produced the Critical.** It still reads "couch RE-ASSERTS its own on every paint … DECSET is additive and idempotent, so this cannot clobber a mode the child set for itself." Both halves are false at this head: `console.go:1040` re-asserts only while `!c.childWantsMouse()`, precisely *because* 1000/1002/1003 replace one another. `22de4ceb` and `079a6461` touched `console.go`, tests, `lessons.md` and the gate ledgers and left the atlas alone, while a brand-new atlas section for the same feature landed in the same window. **This is the 2nd finding in family `docs-lag-shipped-surface`** (BR-18 was the first). Do not patch only this paragraph — the rule: *when a boundary changes a behaviour the atlas already describes, the same commit rewrites the contradicted paragraph; the check is `grep` the atlas for the mechanism the diff changed, not "did I add a section".* Measured prevalence: 1 of 1 contradicted paragraph left standing.

**E. `workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md` — the plan artifact disagrees with the tree in six places and has no `## Revisions` entry for any of them.** (i) `:182` declares `hostty.EnableMouseClicks` / **`DisableMouseClicks`** status `new`; `DisableMouseClicks` was deleted in `079a6461` and exists nowhere — and the row's declared path is outside `conceptPackage`, so the contract test cannot see it, the same blind spot BR-3 exploited. (ii) `:118`, `:165`, `:394`, `:396` still describe `seqMouse` as shipped work. (iii) `:396` names `mouseinput.IsSGRPrefix` and `:155` names `mouseinput.FindSGR`; the shipped names are `IsPrefix` and `Find`. (iv) The section titled "**The complete** disposition table" (`:41-56`) has no `couchOwnsScreen` dimension, so it is missing the two switcher rows the code and `mouse.go`'s own doc carry, and its "anything, elsewhere | yes | forward verbatim" row is contradicted by the switcher case. (v) `:161` states the signature as `(event, hostRows, childWantsMouse)`; the shipped one takes four arguments. (vi) All 43 checkboxes are `- [ ]`, including M1's Tasks 1-5 whose work demonstrably shipped. **This is the 3rd finding in family `plan-table-claims-unshipped-code`.** The rule, covering this and BR-25: *before the boundary tick, diff the plan against the tree in BOTH directions — every backticked symbol in the artifact resolves in the tree, and every signature or table the artifact calls "complete" is re-derived from the code — and each delta gets a `## Revisions` entry. Fixing the machine-parsed row alone is fixing the part the test forces.* I have marked this Important rather than Critical despite the review checklist's table-vs-code rule: the code is right and the guard is blind to that row, so the damage is documentation drift, not the silent coverage hole BR-3 was.

## 4. Minor findings

- `reserve.go:86-105` — `go doc ChipSpan` still prints "The untrusted-text rationale below belongs to RenderStatusRow… RenderStatusRow lays the model out in width columns…", and `go doc RenderStatusRow` prints nothing. The block was explained, not moved; `lessons.md:3477-3483` records it in the past tense. (`BR-9`, not-addressed.)
- `artifactpath/manifest.go:549,557` — `cmd/internal/mouseinput/mouseinput.go` is inserted inside the `couchcore` block and `couchtty/mouse.go` before `couchtty/menu.go`, breaking the list's alphabetical order.
- `console_mouse_test.go:145-149` — `TestChildWithoutTrackingReceivesNoMouseBytes` inspects only `w[0] == 0x1b`, so a report concatenated behind other bytes passes. The split test covers this today, but the assertion is weaker than the name.
- `mouseinput.go:50` — `s == ""` is unreachable once `strings.HasPrefix(s, "\x1b[<")` has succeeded.
- `menu_render.go:181-195` — `clampExtents` is now honestly documented as unreachable, which is the right disclosure, but it remains a branch no input can enter; consider collapsing it into an assertion inside the emit loop.

## 5. Test coverage notes

Everything I claim above about a fix being load-bearing, I mutated in a scratch checkout of the pinned head. **Confirmed load-bearing:** the per-paint mode guard (unconditional → 4 red; deleted → 4 red, different ones), the `HitMouse` handler-table entry (→ 4 red), `assertDirectTest`'s self-exclusion (removing `mouse_test.go` → 2 red), the extent run-merge. **Confirmed NOT load-bearing:** the scrolled extent index (finding B), `enterOperationFor` in the click arm and the `len(effects)` guard (finding C), `clampExtents` (acknowledged in-source). **Confirmed reachable defects by probe:** the encoding-clear demote and the encoding-only stand-back (finding A). The pty-dependent suites across `couchtty`, `couchcore`, `couchcmd`, `termcmd`, `hostty`, `keyscmd` and `ptychild` fail with "operation not permitted" in this environment — that is the known sandbox limitation, not this diff; every non-pty package is green at the pinned head.

## 6. Architectural notes

- **ARCH-DRY — pass with one flag.** One parser, spans from the clipping pass, one `enterOperationFor`. Flagged in C: the extraction is a convention rather than an invariant, because nothing reddens when the click arm restates it. `findSGRMousePress`/`isSGRMousePrefix` (`run.go:580,584`) are one-line pass-throughs, but they have live call sites, so BR-20's rule is satisfied and I am not raising them.
- **ARCH-PURE — pass.** `reserve.go`, `menu_render.go`, `mouse.go`, `keys.go` and `mouseinput` import no IO seam and their tests run without one; `Console.onMouse` is a genuinely thin adapter that converts 1-based→0-based once and delegates.
- **ARCH-PURPOSE — flag.** The `mouseinput` shadow-sweep passes: `run.go`, `rename_input.go` and `keys.go` all derive from the package and the surviving literals are gone. The remaining hand-maintained restatements of the model are the plan (E) and the atlas (D), and finding B is the Done-when's own words deferred.
- **ARCH-MOCK — flag.** `hostty.FakeHost` and `ptychild.FakeChild` are real stateful doubles on the same seam production uses. But the dependency fact the entire policy rests on — that `?1000h` *replaces* a child's `?1002h` — is modelled nowhere executable: `FakeChild.Mouse()` is a single bool, so every test asserts "couch did not write" rather than "the child's mode survived", and there is no live conformance check against a real terminal. This is the mechanism behind finding A and the fix for both is the same split.
- **ARCH-CONSTRAINTS — pass.** `?1000` not `?1002`/`?1003` is honoured and documented; the `MaxReport` bound on the Interceptor hold is enforced and directly tested; the per-paint mode write is 12 bytes and one mutex against a row repaint that was already happening.
- **ARCH-SECURE — pass.** The report is parsed into a typed `mouseinput.Event` at the boundary and refused rather than clamped; `ColumnToActor`/`PointToActor` are total and return false for negative, gap and past-the-end coordinates; an over-long prefix degrades to ordinary input rather than a fabricated coordinate or a parked keyboard. Untrusted labels are still stripped before layout.
- **ARCH-ORDER — flag.** Two things. The mouse payload now lives in *two* mutable slots — `Interceptor.mouse` (`keys.go:236`) and `Console.mouseHit` (`console.go:135`) — each valid only between one `FeedHit` and the next; correct today because both live on the Run goroutine, but kept correct by hand-written ordering rather than by the type. The durable answer is still widening `hitHandlers()`'s value to `func(MouseHit)`. And the mode policy has no explicit trigger: the `(child mode, couch write)` table is enumerated but the *event* that re-evaluates it is an incidental paint (finding A), which is exactly the "no seam to inject ordering" shape this lens exists to flag. Also under this lens: `menuExtents` is a snapshot of the last *menu* paint but is gated by `focus.IsPanel()`, so "panel focused" and "panel drawn" are two facts held in agreement by convention.

## 7. Plan revision recommendations

- `### 2026-09-05 — the disposition table gained a dimension it does not show.` `RouteMouseReport` takes a fourth argument, `couchOwnsScreen`; add the two switcher rows (button-0 press → couch, anything else → swallow) at `:41-56` and correct "anything, elsewhere | yes | forward verbatim" to say it holds only while a child is displayed. Record why: the panel branch was unreachable until a test clicked a row, which `mouse.go:59-63` already records.
- `### 2026-09-05 — DisableMouseClicks was deleted, not shipped.` Change `:182` to `hostty.EnableMouseClicks` only, and note that the row's declared path is outside `conceptPackage`, so no guard can catch a wrong symbol there — the third artifact row this issue has had wrong for that reason.
- `### 2026-09-05 — the seqMouse / IsSGRPrefix / FindSGR rename sweep.` Finish the rule BR-25 stated: `grep` the artifact for each old name until it returns zero outside the Revisions entries themselves.
- `### 2026-09-05 — M1's tasks shipped unticked.` Tick Tasks 1-5's steps, or state which were superseded — Task 5's `conceptPlans` registration landed; Task 4 Step 4's scrolled case did not (finding B).
- Add a line under **Non-goals** or the ARCH-SECURE section stating that couch's `?1006` is not stood back from when the child holds a tracking mode, and what a child that enabled `?1000h` without `?1006h` is expected to see (finding A).

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      Run-merge and clamp bounds now pinned, but the scrolled list still enters no test: index := len(lines) + start*1000 leaves the whole package green.
  - id: BR-9
    disposition: not-addressed
    note: |
      go doc ChipSpan still prints RenderStatusRow's rationale and go doc RenderStatusRow prints nothing; the block was explained, not moved.
  - id: BR-11
    disposition: addressed
    note: |
      mousePressEvent and parseSGRMousePressPrefix are deleted; the two surviving wrappers have live call sites at run.go:418,471.
  - id: BR-16
    disposition: addressed
    note: |
      Mutation-verified both ways: deleting the per-paint re-assert reddens 4 subtests, making it unconditional reddens 4 others.
  - id: BR-17
    disposition: not-addressed
    note: |
      Both halves revert green (enterOperationFor -> "switch"; Manual set unconditionally), and the non-actionable notice divergence remains.
  - id: BR-19
    disposition: addressed
    note: |
      MenuEventMouseSwitch now carries its own doc and MenuEventNotice's block is re-anchored.
  - id: BR-20
    disposition: addressed
    note: |
      All four enumerated symbols swept: DisableMouseClicks and the alias deleted, WheelUp/WheelDown now used at run.go:453,459.
  - id: BR-21
    disposition: addressed
    note: |
      hitHandlers()[HitMouse] is a real handler; no-oping it reddens four console tests.
  - id: BR-23
    disposition: addressed
    note: |
      Verified by deleting mouse_test.go — both PURE rows now report uncovered instead of matching the inventory literal.
  - id: BR-22
    disposition: addressed
    note: |
      The paintNow site is fixed and pinned; the same mechanism survives through Screen.Mouse()'s mode conflation, raised separately.
  - id: BR-24
    disposition: addressed
    note: |
      sgrMouseSize derives the length from ParsePrefix and the malformed-input winner is stated.
  - id: BR-25
    disposition: not-addressed
    note: |
      The named bullet is fixed; the class is not — seqMouse, IsSGRPrefix, FindSGR and DisableMouseClicks still name symbols absent from the tree.
findings:
  - id: new
    severity: Important
    family: unspecified-event-policy
    title: |
      couch writes two mouse modes and governs one, and the state it reads collapses tracking and encoding into a single bool
    detail: |
      This is the 4th finding in family `unspecified-event-policy`. Do NOT add a fifth
      site fix. Probed and confirmed at the pinned head: a child doing `\x1b[?1002h`
      then `\x1b[?1006l` flips Screen.Mouse() false (screen.go:415 maps 1000/1002/1003
      /1006 onto one bool), so the next paint writes `?1000;1006h` and demotes it —
      BR-22's mechanism reached through the observation instead of the write; and a
      child doing `\x1b[?1006h` alone makes couch stand back for a child holding no
      tracking at all. Two more halves: `?1006` is never stood back from, so a child
      enabling `?1000h` without `?1006h` receives SGR reports it cannot parse (against
      the "receives its own events unchanged" Done-when), and the startup write at
      console.go:528 is gated by nothing. The take-back also has no trigger of its own:
      onChunk repaints only on RowDirty/Bell/attention and the DECSET handler sets
      rowDirty only for alt-screen, so a bare `?1000l` leaves couch's clicks off
      indefinitely; TestMouseModeTransitions cannot see this because it calls repaint()
      by hand and its FakeChild has a nil sink. The rule: couch's mouse ownership is ONE
      gated writer over the full set of modes it touches — tracking AND encoding —
      driven by an observation that keeps them as separate state, and every cell of
      (child mode transition x couch write) states what couch writes, what mode AND
      encoding the child is left holding, and what EVENT re-evaluates the cell.
      Measured prevalence: 3 writers, 1 governed; 2 modes written, 1 governed; 1
      observation collapsing 4 modes into 1 bool; 6 cells pinned, 0 pinning the trigger.
  - id: new
    severity: Important
    family: docs-lag-shipped-surface
    title: |
      atlas/couch.md:356-361 still teaches the refuted model that produced BR-22
    detail: |
      This is the 2nd finding in family `docs-lag-shipped-surface` (BR-18 was the
      first). Do NOT patch only this paragraph. It reads "couch RE-ASSERTS its own on
      every paint ... DECSET is additive and idempotent, so this cannot clobber a mode
      the child set for itself"; both halves are false at this head, since console.go
      :1040 re-asserts only while !childWantsMouse() precisely because 1000/1002/1003
      replace one another. The rule: when a boundary changes a behaviour the atlas
      already describes, the same commit rewrites the contradicted paragraph — the
      check is grepping the atlas for the mechanism the diff changed, not "did I add a
      section". Measured prevalence: 1 of 1 contradicted paragraph left standing while
      a new section for the same feature landed in the same window.
  - id: new
    severity: Important
    family: plan-table-claims-unshipped-code
    title: |
      The plan artifact disagrees with the tree in six places, with no Revisions entry
    detail: |
      This is the 3rd finding in family `plan-table-claims-unshipped-code`. Do NOT fix
      only the row a test reads. Sites: :182 declares DisableMouseClicks status `new`
      and it exists nowhere (declared path is outside conceptPackage, so the contract
      test is blind to it, the same gap BR-3 exploited); :118, :165, :394, :396
      describe seqMouse as shipped work; :396 names mouseinput.IsSGRPrefix and :155
      names mouseinput.FindSGR, which shipped as IsPrefix and Find; the section titled
      "The complete disposition table" (:41-56) has no couchOwnsScreen dimension and so
      omits the two switcher rows the code carries, and its "anything, elsewhere | yes
      | forward verbatim" row is contradicted by the switcher case; :161 states a
      three-argument signature where four shipped; and all 43 checkboxes are unticked
      including M1's Tasks 1-5, whose work demonstrably landed. The rule, covering this
      and BR-25: before the boundary tick, diff the plan against the tree in BOTH
      directions — every backticked symbol resolves in the tree, and every table or
      signature the artifact calls "complete" is re-derived from the code — and each
      delta gets a `## Revisions` entry. Important rather than Critical: the code is
      right and the guard is blind to the row, so this is documentation drift, not the
      silent coverage hole BR-3 was.
  - id: new
    severity: Minor
    family: move-residue
    title: |
      mouseinput.go:50's `s == ""` is unreachable after the HasPrefix check succeeds
    detail: |
      Carried over verbatim from termcmd's parseSGRMousePress in the promotion. Dead
      condition, no behaviour change either way.
  - id: new
    severity: Minor
    family: unpinned-exported-shape
    title: |
      TestChildWithoutTrackingReceivesNoMouseBytes only inspects the first byte of each write
    detail: |
      console_mouse_test.go:145-149 checks w[0] == 0x1b, so a report concatenated
      behind other bytes would pass. The split test covers this case today, but the
      assertion is weaker than its name and would not catch a batching change.
  - id: new
    severity: Minor
    family: docs-lag-shipped-surface
    title: |
      artifactpath/manifest.go:549,557 break NonArtifactSources' alphabetical order
    detail: |
      cmd/internal/mouseinput/mouseinput.go is inserted inside the couchcore block and
      cmd/internal/couchtty/mouse.go sits before couchtty/menu.go. Nothing enforces the
      order, which is why it drifted.
```
