# Boundary Review — pair#172 (whole-issue close)

| field | value |
|-------|-------|
| issue | 172 — Mouse support: click the status bar and the switcher |
| repo | pair |
| issue file | workshop/issues/000172-clickable-status-bar.md |
| boundary | whole-issue close |
| milestone | — |
| window | c15030df41c64086f1e669576034d22bcbb0ea28..a2ab2d2d2740e718dce136c3a58b8b385458c8b0 |
| command | sdlc close --issue 172 |
| reviewer | claude |
| timestamp | 2026-09-06T00:42:26-07:00 |
| verdict | REWORK |

## Review

I've completed the review, verifying claims by mutation and probe in a scratch copy of the pinned tree.

```verdict
verdict: REWORK
confidence: high
```

The pure geometry half of this issue is genuinely well built — chip spans come out of the clipping pass, `RouteMouseReport` is a real table with a property test over the whole event space, and the `#127` prefix-parking class is pinned at every split boundary this time. What blocks SHIP is the terminal half and the artifacts around it. I reproduced, at the pinned head, two mode-policy defects BR-26 named and one it implied: a child that enables `?1000h` without `?1006h` now receives SGR reports it cannot parse because couch sets `?1006` at startup and never releases it — behavior drift from the "a child that did enable tracking still receives its own events unchanged" Done-when, introduced by this diff; and a bare `?1000l` from the child never triggers a repaint, so the take-back BR-16's fix was supposed to deliver never fires on its own (probe: `host.Written()` stays empty). Separately, three claimed fixes are not pinned by any test — reverting the `if len(effects) > 0` guard, the `enterOperationFor` call in the click arm, and the column-vs-rune accounting in `RenderStatusRow` each leaves the entire `couchtty` suite green — and the atlas paragraph the *same commit* added still teaches the model the code refutes. Fixing the mode policy is a design change with its own test matrix, not a one-liner, which is why this is REWORK rather than FIX-THEN-SHIP.

## 1. Strengths

- `cmd/internal/couchtty/mouse.go:56` — `RouteMouseReport`'s three-way disposition with the full table in the doc block, and `mouse_test.go:53` `TestNoReportReachesAChildThatNeverAskedForMouse` asserting the zero-bytes rule as a *property* over button × release × every row, not a case list. That is the right shape for a Done-when phrased as "asserted, not assumed".
- `cmd/internal/couchtty/reserve.go:174-177` — spans recorded from inside the same `appendText` that clips (`used` accumulates `textwidth.Width`, so they are true columns, correct for a mouse X). `TestAChipClippedToOneColumnIsStillClickable` *constructs* the one-column case rather than sweeping widths and asserting whatever it finds — the fix BR-4 asked for, done properly.
- `cmd/internal/couchtty/keys.go:282-303` — the mouse arm placed before the fixed-string table with the `MaxReport` bound, pinned by `TestMouseReportsAreWithheldAndSurviveEverySplit` (every split index) and `TestAnUnterminatedMousePrefixDoesNotParkTheKeyboard`.
- `cmd/internal/couchtty/core_concepts_contract_test.go:414-421` — the `assertDirectTest` self-skip is a real guard fix, and it paid immediately: it exposed a second vacuous assertion (`endsItsOwnChild`), which got its own test at `menu_action_sweep_test.go:116`. A guard fix that finds a second instance on its first run is the good outcome.
- `console_menu.go:187-192` — extents captured at the single `RenderMenuView` host call site, so they cannot go stale against the drawn panel. I checked every `showMenu` caller and `takeOverScreen` writes `HomeAndClear` first, so body line *i* is screen row *i+1* and the `Y-1` conversion is right.

## 2. Critical findings

None newly raised. (BR-26's `?1006` half is behavior drift from a stated Done-when and would be Critical if raised fresh; it is disposed under its existing id rather than re-raised.)

## 3. Important findings

See the `findings` block. In brief: BR-5, BR-17, BR-26, BR-27 and BR-28 all remain open at the pinned head, each verified rather than read off a commit message; one new finding on the chip-span coordinate *unit* being unpinned.

## 4. Minor findings

- BR-9 — `go doc ChipSpan` still prints the whole `RenderStatusRow` rationale; `go doc RenderStatusRow` prints nothing. The "fix" added a meta-paragraph about the problem *inside the block that has the problem*.
- BR-29 — `mouseinput.go:51`'s `s == ""` is still unreachable.
- BR-30 — `console_mouse_test.go:146` still checks only `w[0] == 0x1b`.
- BR-31 — `manifest.go:549,557` still out of alphabetical order.

## 5. Test coverage notes

Verified by mutation in a scratch copy (`./cmd/internal/couchtty/` full package; the only pre-existing failure is `TestNotificationPTYConformance`, environmental — `ptychild: operation not permitted`):

| mutation | result |
|---|---|
| `index := len(lines)` → `start + i` in `renderRootMenuFrame` | **RED** (`TestPointToActorSpansEveryLineOfAnActor`) — BR-5's named mutation is now caught |
| `clampExtents` body → `return extents` | GREEN — and a panic probe confirms neither branch fires anywhere in the suite; the code now says so honestly |
| scroll-branch panic probe (`start > 0`) under the three extent tests | never entered — the 12-actor fixture selects actor 0, so it clips at the bottom but never scrolls |
| `if len(effects) > 0` guard removed (BR-17) | GREEN |
| `enterOperationFor(thread)` → `"switch"` in the click arm | GREEN |
| `used += textwidth.Width(clipped)` → `len([]rune(clipped))` | GREEN |

The last three are the fix-completeness rule: a fix is complete only when a test fails without it.

## 6. Architectural notes

- **ARCH-DRY — pass with one flag.** `enterOperationFor`, the single `mouseinput` parser, and spans-from-the-clipping-pass are all the right consolidation. The flag is that the *restatements* (atlas §356-361, the plan's disposition table) are hand-maintained copies of the model that have now diverged from it — BR-27 and BR-28.
- **ARCH-PURE — pass.** `ChipSpan`/`ActorExtent`/`ColumnToActor`/`PointToActor`/`RouteMouseReport` all run with no IO; `Console.onMouse` is the thin shell and does the one 1-based→0-based conversion.
- **ARCH-PURPOSE — flag.** Shadow-sweep on the plan-as-source: it is a restatement that no longer derives from the tree in at least six places with no `## Revisions` entry (BR-28), and the Done-when's "scrolled list" case is the deferred subset (BR-5).
- **ARCH-MOCK — flag, and it is the root of BR-26.** `ptychild.Screen` is the fake modelling the terminal's mouse state, and it models four DECSET modes as one bool. There is no live conformance check for the claim the whole policy rests on ("1000/1002/1003 replace one another; 1006 is orthogonal") — that belief lives only in a code comment. The fake and the dependency have already drifted; the `?1002h`+`?1006l` case is where.
- **ARCH-CONSTRAINTS — pass.** `?1000` not `?1002/1003` is declared and enforced by the constant; `MaxReport` bounds the held buffer; `paintNow`'s new `childWantsMouse()` call is outside the lock it re-takes (no deadlock, confirmed by the passing suite).
- **ARCH-SECURE — pass.** Labels are `sanitize`d before both the render and the spans; `Parse` rejects non-numeric and out-of-range fields rather than guessing.
- **ARCH-ORDER — flag, the main one.** BR-26 is an ordering finding, not a mode finding: the take-back has no *event* of its own. `Screen`'s DECSET handler sets `s.mouse` without `rowDirty`, and `onChunk` repaints only on `RowDirty`/`Bell`/attention, so the `(child releases mode) → (couch re-asserts)` cell has no trigger. `TestMouseModeTransitions` cannot see this because it calls `repaint()` by hand. That is the "test that can only observe one interleaving" case: it confirms the ordering the author supplied.

## 7. Plan revision recommendations

`workshop/plans/000172-mouse-support-status-bar-and-switcher-plan.md` needs one `## Revisions` entry covering, at minimum: `:41-56` the disposition table has no `couchOwnsScreen` dimension and its "anything, elsewhere | yes | forward verbatim" row is contradicted by the switcher case; `:155` `FindSGR` shipped as `Find`; `:161` the stated three-argument `RouteMouseReport` signature shipped with four; `:165`/`:394`/`:396` describe `seqMouse` as live work that `:118` already records as deleted; `:396` `mouseinput.IsSGRPrefix` shipped as `IsPrefix`; `:182` `DisableMouseClicks` has status `new` and exists nowhere in the tree; `:186-192` states `FeedHit` returns the event alongside the hit, where what shipped is the `Interceptor.Mouse()` side-channel; and all 42 checkboxes are unticked including M1's Tasks 1-5. Also stale in the same class: `core_concepts_contract_test.go:61-62`'s comment says "The M2/M3 rows carry status `planned` and are invisible here" immediately above the M2 rows it lists, and the file still speaks of M2/M3 after the milestones collapsed to one.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      Clamp half answered honestly (probe confirms unreachable); the scrolled
      case is still entered by no extent test -- panic probe on start>0.
  - id: BR-9
    disposition: not-addressed
    note: |
      go doc ChipSpan still prints the RenderStatusRow rationale; go doc
      RenderStatusRow prints nothing. Verified by running go doc.
  - id: BR-17
    disposition: not-addressed
    note: |
      enterOperationFor is shared (good), but both the resume branch and the
      len(effects)>0 guard survive reversion green; the notice divergence remains.
  - id: BR-25
    disposition: addressed
    note: |
      The named bullet and every rename residue are gone; the only remaining
      MouseClickTracking hits are the Revisions entries recording the rename.
      The class is carried by BR-28.
  - id: BR-26
    disposition: not-addressed
    note: |
      Untouched since round 8 (only a docs commit landed after). Both mechanisms
      reproduced at head; see the detail on the re-raise rule below.
  - id: BR-27
    disposition: not-addressed
    note: |
      atlas/couch.md:358-361 unchanged, and the diff shows the same commit ADDED
      that paragraph -- it is not stale prose, it is a new wrong claim.
  - id: BR-28
    disposition: not-addressed
    note: |
      Only :118 (seqMouse deleted) changed; five named sites plus two more remain
      and there is still no Revisions entry.
  - id: BR-29
    disposition: not-addressed
    note: |
      mouseinput.go:51 unchanged.
  - id: BR-30
    disposition: not-addressed
    note: |
      console_mouse_test.go:146 still w[0] == 0x1b.
  - id: BR-31
    disposition: not-addressed
    note: |
      manifest.go:549,557 unchanged.
findings:
  - id: new
    severity: Important
    family: unpinned-exported-shape
    title: |
      ChipSpan's documented column unit is unpinned -- swapping textwidth.Width for a rune count leaves the whole couchtty suite green
    detail: |
      This is the 4th finding in family `unpinned-exported-shape`. Earlier rounds
      fixed instances (BR-6 a missing test file, BR-13 a run shape, BR-30 a weak
      assertion). Do NOT fix this instance alone.

      The rule that covers all of them: a documented property of an exported shape
      is pinned only by a fixture that can DISCRIMINATE it. reserve.go:159
      accumulates `used += textwidth.Width(clipped)`, and ChipSpan's own doc calls
      Start/End "the column range" -- correct, because an SGR X is a terminal
      column. But every chip fixture is ASCII, where columns, runes and bytes all
      coincide, so replacing that line with `len([]rune(clipped))` leaves
      `./cmd/internal/couchtty/` fully green (verified by mutation). The same
      fixture-cannot-discriminate shape is why BR-30's first-byte check passes and
      why BR-5's clipped case needed constructing rather than sweeping. The
      enumeration this implies, and which should be swept in one round: for each
      exported geometry contract on this feature, name the property, then name the
      fixture value that violates it -- a CJK or emoji label for the column unit
      (labels are UNTRUSTED text and truncate()'s own doc calls out emoji), a
      scrolled window for the drawn-row base (BR-5), a report behind other bytes
      for the zero-bytes rule (BR-30). Measured prevalence: 3 of 3 geometry
      properties I mutated are unpinned by the shipped fixtures.
```
