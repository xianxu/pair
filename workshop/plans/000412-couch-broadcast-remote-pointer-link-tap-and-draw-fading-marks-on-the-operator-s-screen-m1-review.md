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
