# Boundary Review — pair#412 (milestone M1)

| field | value |
|-------|-------|
| issue | 412 — Couch broadcast: remote pointer link (tap and draw fading marks on the operator's screen) |
| repo | pair |
| issue file | workshop/issues/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 58c6dadacbf4a039d42c73cd362a8791b3b506cf..034c056a201cd77f220ebab93fb40297cddebd48 |
| command | sdlc milestone-close --issue 412 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-08T09:41:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the plan claims. The emoji-width check found a real mismatch, and it was fixed in the right place: the Unicode 11 add-on is vendored for the viewer and pinned in the oracle. `Marks` is a pure model with a fixed-clock table of tests. The Presenter overlay hook is ordered through `p.call`, applied before the diff, the tap and `p.previous`, and kept out of scrollback; an oracle test proves the scrollback part. The new tests pass: `broadcast` passes in full under `-race`, and every new `terminal` test passes. The other `terminal` failures are the sandbox's pty `operation not permitted`. The `artifactpath` failures come from unrelated inventory entries that exist on main, not from `marks.go`.

Two problems in `Marks` should be fixed before M2/M3 connect it to untrusted input:
- **Cap eviction is quadratic.** It runs under the leaf lock that the paint path takes.
- **Status-row exclusion is checked in the wrong place.** It is checked against the batch's grid, not against the frame actually being painted.

**1. Strengths**
- `cmd/internal/terminal/presenter.go:784-789`: the overlay is applied after `Validate`, and the result is checked again. `p.previous` and the tap both get the overlaid frame, so the diff and the viewers agree. The test showing an identity overlay leaves the bytes unchanged pins the "no marks, no cost" claim.
- `cmd/internal/terminal/overlay_test.go:732`: `TestOverlayNeverReachesScrollback` uses the independent oracle. It also guards against passing vacuously ("no scrollback produced; the test proves nothing").
- `cmd/internal/broadcast/marks.go:246-269`: `Overlay` returns the frame itself, uncopied, when nothing is live. It handles wide-character continuation cells, and tests pin both behaviors.
- The width mismatch went through the plan's decision point and is recorded under `## Revisions`. `VENDOR.md` records the integrity hash; I checked it against the file's SHA-256 (`72353b51…`). I also confirmed by reading the code that the add-on's UMD export matches `globalThis.Unicode11Addon.Unicode11Addon`.

**2. Critical findings:** none.

**3. Important findings**
- **Eviction cost (ARCH-CONSTRAINTS).** `cmd/internal/broadcast/marks.go:154-165`: for every cell over the cap, the loop rescans the whole map.
  - I re-ran the algorithm standalone on a 200×60 grid with every cell marked: 850 ms. One legal 64-point zigzag batch marks about 6,400 cells and costs about 250 ms.
  - M3 runs `Add` under `marksMu`. The overlay takes that same lock inside every paint, so at the planned 30 requests/s one pointer holder can freeze the operator's screen.
  - The tie-break is also wrong. All cells from one batch share a timestamp, so eviction drops them by row and column, not by stroke order.
  - Fix: one pass that collects the cells and sorts them by (time, sequence), then deletes the excess, with a monotonic per-cell sequence number. Add a test or benchmark that bounds the worst-case batch.
- **Status-row exclusion on resize.** `cmd/internal/broadcast/marks.go:254`: `Overlay` clips only to the frame's own size. The status-row exclusion is enforced only in `Add`, using the batch's row count.
  - Marks that survive a resize (they last up to 3 s) can tint the new last row. A tinted `LIVE` label fails `IndicatorShown`, so the hub drops frames and the fail-safe can fire. The plan's trust section says marks can't trip the fail-safes; this breaks that.
  - The strongest tint is colour 214, the same as `PointerSGR`, so in M3 a stray mark could also hide or fake the `PointerShown` check.
  - Fix: skip `c.row >= rows-1` in `Overlay` as well, with a test that adds marks on a 10-row grid and overlays a 9-row frame.

**4. Minor findings**
- `marks_test.go:453`: `var _ = terminal.FramePrivate` is filler and asserts nothing.
- The viewer's add-on loading (`viewer.js:584-588`, `index.html` script order) isn't covered by any test. A wrong global would break all #395 viewing. Correct by inspection; the M4 smoke will catch it.
- `Refresh` on a panel repaints with `selection=true`, which bumps the view token on every fade step. That matches `Panel` itself, but every fade step now counts as a re-selection; note it for later.

**5. Test coverage notes**
- Each Task 1.2 and 1.3 test named in the plan is present and passes.
- Missing: a cost bound on `Add`, and a test that overlays marks on a frame whose geometry differs from the batch's (Important #2).

**6. Architectural notes for upcoming work**
- ARCH-DRY: pass. One place decides mark geometry and fading; the marker constants live alongside the existing `LiveLabel`/`LiveSGR`.
- ARCH-PURE: pass. `Marks` takes the clock as an argument and the overlay is a plain function value.
- ARCH-PURPOSE: pass for M1's scope.
- ARCH-MOCK: N/A; no new external dependency apart from the vendored add-on, which the oracle runs.
- ARCH-CONSTRAINTS: flagged (Important #1).
- ARCH-SECURE: the vendor integrity is recorded, and `allowProposedApi` only widens the API inside the page. The paint-time status-row check is defense in depth for untrusted pointer input (Important #2).
- ARCH-ORDER: pass. `SetOverlay` and `Refresh` are serialized with paints.
- ARCH-FUNERAL: pass. Everything is in memory; expired cells are pruned only on `Add`, but the cap bounds them.
- For M3: keep `marksMu` a true leaf, and run `Add` outside the paint-critical lock or keep it cheap.

**7. Plan revision recommendations**
- The Core concepts table lists `Marks (Add, Clear, Overlay, Expired)`, but the code has `Live` and `NextChange`, and `Add` takes `(points, cols, rows, now)` rather than a batch. Add a `## Revisions` entry recording the real surface.

```findings
findings:
  - id: new
    severity: Important
    family: untrusted-input-work-unbounded
    title: |
      Marks.Add cap eviction is O(excess x n), up to ~250-850ms per batch under the paint-path lock
    detail: |
      marks.go:154-165 rescans the whole map for each evicted cell. Measured 850ms on a full 200x60 grid; a legal 64-point zigzag batch marks about 6400 cells, about 250ms. M3 holds marksMu, which the overlay takes on every paint, so this stalls the operator's screen. Same-batch cells are also evicted by row/col, not stroke order. Fix: sort once by (time, sequence) and delete the excess; add a worst-case bound test.
  - id: new
    severity: Important
    family: invariant-enforced-at-paint-time
    title: |
      Marks.Overlay can tint the status row after a resize, failing IndicatorShown
    detail: |
      The status-row drop is enforced only in Add, against the batch's rows. Overlay clips only to the frame size (marks.go:254), so marks surviving a shrink tint the new last row. That breaks IndicatorShown (the hub drops frames and the fail-safe can fire), and the tint, colour 214, equals PointerSGR. Skip row >= rows-1 in Overlay and test with a smaller frame.
  - id: new
    severity: Minor
    family: plan-table-matches-code
    title: |
      Plan Core concepts lists Marks.Expired; the code has Live/NextChange and Add(points, cols, rows, now)
  - id: new
    severity: Minor
    family: test-filler
    title: |
      marks_test.go ends with var _ = terminal.FramePrivate, which asserts nothing
  - id: new
    severity: Minor
    family: untested-page-wiring
    title: |
      No test covers the viewer loading the Unicode 11 add-on; a wrong global would break all viewing
```

---

## Re-review — 2026-10-08T09:43:51-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 412 — Couch broadcast: remote pointer link (tap and draw fading marks on the operator's screen) |
| repo | pair |
| issue file | workshop/issues/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 58c6dadacbf4a039d42c73cd362a8791b3b506cf..864ce607aef81b908546882cf6a823eed77be927 |
| command | sdlc milestone-close --issue 412 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-08T09:43:51-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 delivers what the plan promises. It adds the 👆/👽 marker constants with width checks in all three measurers (`textwidth`, `ansi`, the vt emulator), and the Unicode 11 add-on in both the viewer and the oracle. It adds a pure `Marks` model and the Presenter's `Overlay`/`SetOverlay`/`Refresh` hook. Every test the plan names for Tasks 1.1–1.3 exists and passes. The two Important findings from the earlier rounds are fixed, and I checked each fix by mutation, not by reading the commit message:
- **BR-3:** I reverted the overlay's `rows-1` guard to `rows` in a scratch copy, and `TestMarksOverlayNeverTintsStatusRow` failed ("status-row cell 0 tinted after a resize").
- **BR-2:** I put the pre-fix `Add` back (from `aba2176b`), and `TestMarksAddWorstCaseIsCheap` failed at 200ms per batch. With the fix it takes about 0.3ms, so the 20ms bound has roughly 60× headroom.

The four Minor findings still open are bookkeeping and don't block. I raise one new Minor below, about a precondition.

**1. Strengths**
- `marks.go:41-82`: `Add` now cuts the batch down to the cap before inserting, and evicts with a single sort. It is bounded at O(n log n), where n ≤ cols·rows/8. Because the batch is cut first and earlier cells are older, cells from the same stroke are evicted only when two `Add` calls share a timestamp.
- `marks.go:170-174`: the status-row guard now applies where the tint is drawn, not only where input enters. The new lesson in `lessons.md` states the general rule.
- `tap.go:27-46`: `Overlay` and `Refresh` both go through `p.call`, so they are ordered with paints. `Panel` clones its frame before keeping it in `p.panel` (`presenter.go:461`), so `Refresh` can't repaint a frame the caller has since changed. `Select` resets `p.panel`.
- `presenter.go:330-335`: the overlaid frame is checked with `Validate` again before painting. `TestIdentityOverlayIsByteIdentical` and `TestOverlayNeverReachesScrollback` check, through the oracle, that the overlay adds nothing to the bytes and never reaches scrollback.
- The marker widths are checked in every measurer (`TestCapabilityMarkerWidths`, `TestOracleCapabilityMarkersAreWide`). That is the right way to catch drift between Couch and the viewer.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- **BR-1 (not addressed):** the plan's task prose still restates the test cases.
- **BR-4 (not addressed):** the plan's Core concepts table still lists `Expired`, and there is no Revisions entry.
- **BR-5 (not addressed):** `marks_test.go` still has `var _ = terminal.FramePrivate`.
- **BR-6 (not addressed):** no test covers the viewer loading the add-on. The M4 smoke test is the intended backstop.
- **New, Minor — a precondition that isn't enforced:** `Add`'s doc comment says its cost is bounded, but the line walk (`line`) runs once per cell between the two raw coordinates, before the off-grid filter. A point like `[1e9, 0]` therefore costs about 10⁹ iterations under the paint-path lock. The plan relies on M2's parser rejecting out-of-range points, but `Add`'s own comment says off-grid points are "dropped".

**5. Test coverage notes**
- Every M1 test listed in the plan is present and passing.
- The BR-2 and BR-3 regression tests are confirmed to fail without their fixes.
- The worst-case timing test uses a wall-clock bound. Its headroom is large, so it isn't flaky in practice.

**6. Architectural notes**
- **ARCH-DRY: pass.** One model decides mark geometry, fading and the cap. The marker constants sit next to `LiveLabel`.
- **ARCH-PURE: pass.** The clock is passed in as an argument, and there is no IO in `Marks`.
- **ARCH-PURPOSE: pass** for M1's scope.
- **ARCH-MOCK: N/A.** No new external call; the vendored add-on is run by the oracle.
- **ARCH-CONSTRAINTS: pass.** BR-2 is fixed and tested at the worst case. The new Minor above is a precondition on that envelope.
- **ARCH-SECURE: pass with one note.** The status row is now protected where it is drawn. Bounding the cost against raw coordinates is left to M2. When M2 lands, make sure the coordinate validation actually runs before `Add`.
- **ARCH-ORDER: pass.** `SetOverlay` and `Refresh` are serialized on the Presenter goroutine.
- **ARCH-FUNERAL: pass.** Everything is in memory and capped. Expired cells are removed on `Add`, and `Live`/`Overlay` ignore them.
- **For M3:** keep `marksMu` a leaf lock. A `Refresh` on a panel bumps the view token on every fade step; keep that in mind if anything keys off re-selection.

**7. Plan revision recommendations**
- Add a `## Revisions` entry (BR-4): the real `Marks` surface is `Add(points, cols, rows, now)`, `Clear`, `Overlay(frame, now)`, `Live(now)` and `NextChange(now)`; `Expired` was never built. Note there that `Overlay` also drops the last row (BR-3).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Task prose in the plan unchanged; still Minor, non-blocking.
  - id: BR-2
    disposition: addressed
    note: |
      Add now truncates batch to cap and evicts via one sort; restoring aba2176b's Add makes TestMarksAddWorstCaseIsCheap fail at 200ms/batch, fixed runs ~0.3ms.
  - id: BR-3
    disposition: addressed
    note: |
      Overlay skips row >= rows-1 (marks.go:174); mutating to rows makes TestMarksOverlayNeverTintsStatusRow fail; test also asserts IndicatorShown.
  - id: BR-4
    disposition: not-addressed
    note: |
      Plan Core concepts still lists Expired; no Revisions entry records Live/NextChange or Add's signature.
  - id: BR-5
    disposition: not-addressed
    note: |
      var _ = terminal.FramePrivate still present in marks_test.go.
  - id: BR-6
    disposition: not-addressed
    note: |
      No test of viewer add-on wiring; M4 smoke remains the intended backstop.
findings:
  - id: new
    severity: Minor
    family: untrusted-input-work-unbounded
    title: |
      Marks.Add walks Bresenham over raw coordinates before the off-grid filter, so its cost bound holds only if M2's parser validates range
    detail: |
      This is the 2nd finding in family untrusted-input-work-unbounded. Rule: work derived from untrusted input is bounded at the entity doing the work, not by an upstream validator. A point like [1e9,0] costs ~1e9 iterations under the paint lock despite the doc's "cost is bounded". Fix at the rule: in Add, drop (or clip to the grid) any off-grid point before line(), and add the case to TestMarksAddWorstCaseIsCheap.
```
