# Boundary Review — pair#503 (whole-issue close)

| field | value |
|-------|-------|
| issue | 503 — broadcast: remove the 👽 remote-control placeholder from the terminal cast until it's built |
| repo | pair |
| issue file | workshop/issues/000503-broadcast-remove-the-remote-control-placeholder-from-the-terminal-cast-until-it-s-built.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3dc4c71edc050700ed9e18450a204c7266e46813..95065b5cded71b4b81325e5dd2b6e8be4e71cb21 |
| command | sdlc close --issue 503 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T17:08:14-07:00 |
| verdict | SHIP |

## Review

The viewer tests pass (71/71). The Go build needs a generated asset, so I'm looking for how the build makes it.

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** The change does what the issue asked. `👽` is gone from the terminal cast's status row, and so are `ControlLabel`, `RenderedStatusRow.Remote`, `Console.statusRemote`, `onRemoteClick` and its click branch. `StatusGuardCols` now measures `LIVE ⏸ 👆`, so it shrinks from 12 to 9 columns. The pointer and notes spans, the guard test, the shared pin-spot table and `TestNoteVisible` all follow the new width. README, `atlas/broadcast.md` and `atlas/webcast.md` are updated, and #407's Log records the removal and what bringing the control back involves. Nothing blocks shipping. One stale line in `workshop/lessons.md` is the only finding.

I ran the stat, name-status and full diff commands, then searched the tree for leftover references. Test results:
- **`couchtty` and `terminal`:** pass.
- **Viewer `.mjs` tests:** 71/71 pass.
- **`broadcast`:** one failure, `TestNamedTunnelServesFromPrivateSocket`. It also fails when I restore the base version of the package, so this change didn't cause it. It looks like the host refusing unix sockets.
- **Setup:** the Go tests only build after running the runtimebundle generate step first.

1. **Strengths**
   - `cmd/internal/broadcast/indicator.go:80`: `StatusGuardCols` is still computed from the labels, not hard-coded. `TestStatusGuardMatchesDrawnControls` now pins it to `r.Pointer.End`, the new last control.
   - `cmd/internal/broadcast/notes_test.go:331-341`: `TestNoteVisible` now derives its columns from `StatusGuardCols`. It covers the edges at `cols == guard` (visible) and `guard - 1` (off the grid), plus the last-row-inside-the-guard case.
   - `tests/broadcast-viewer/pinspots.json`: the cases moved with the guard (14→11, 12→9) and keep the same offsets relative to it. Both Go and the viewer read this one table, so neither side keeps its own copy of the number.
   - The guard width still reaches the web viewer over the wire (`event: notes` carries `guard`), so `notes.js`/`viewer.js` correctly needed no change. The `guard: 12` in `connect.test.mjs` is only an opaque value passed through, not a geometry assumption.
   - The neighbouring code is intact: removing the `👽` branch in `terminal_input.go` leaves the order of the pointer, notes and LIVE right-click checks unchanged, and `commitChrome` and the `Console` field comment were updated together.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `workshop/lessons.md:1079` still tells readers to "grep the README for the row's current spelling (`LIVE ⏸ 👆 👽`)". README no longer contains that string, so the lesson's grep finds nothing. This is the only stale spelling left outside issue/history files and the intentional historical mentions in the atlas and a test comment.
   - `indicator_test.go:70` (`TestCapabilityMarkerWidths`) and `render_oracle_test.go:302` (`TestOracleCapabilityMarkersAreWide`) now loop over one-element slices. Keeping the slices does make re-adding the control easy, so this is acceptable as is.

5. **Test coverage notes**
   - Both states in `## Done when` are covered:
     - `👽` absent: the exact-row tests in `reserve_broadcast_test.go:73,144,171` assert the full plain row.
     - Pointer and notes still work: `TestStatusRowPointerCells` covers off, on and clipped; `TestStatusRowNotesSpan` checks the notes span sits at `Pointer.End + 1`; the pointer click tests in `console_pointer_test.go` still run.
   - Removing `TestRemoteClickIsANoticeOnly` is correct because the behaviour it tested is gone.
   - The "operator sees it at the next smoke test" clause is manual and outside this review.

6. **Architectural notes**
   - **ARCH-DRY: pass.** No duplicated logic was added, and the guard width still has one source (`StatusGuardCols`), with the JSON table checked against it.
   - **ARCH-PURE: pass.** The change is all deletion. The pure `RenderStatusRow` and `NoteVisible` paths stay pure, and the IO handler was removed along with its routing.
   - **ARCH-PURPOSE: pass.** Every consumer of the removed model was swept: the label constant, row span, console field, click route, guard width, pin-spot table, anchor fixtures, oracle and width tests, README and both atlases. The only leftover is the lessons line above.
   - For #407: re-adding the control means adding a span after `👆`, widening the guard and updating `pinspots.json`. The #407 Log already says this.

7. **Plan revision recommendations:** none. The plan still matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: stale-row-spelling-in-docs
    title: |
      workshop/lessons.md:1079 still cites the status row as `LIVE ⏸ 👆 👽`
    detail: |
      The lesson says to grep the README for the row's current spelling, which is now `LIVE ⏸ 👆`, so that grep finds nothing. A tree-wide search (excluding issue/history files) shows this is the only stale site; atlas/broadcast.md:313 and reserve_broadcast_test.go:111 mention 👽 on purpose, as history.
```
