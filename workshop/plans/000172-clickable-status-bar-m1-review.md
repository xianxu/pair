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
