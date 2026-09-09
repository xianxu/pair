---
type: continuation
slug: 199-tab-strip-test-under-couch
agent: claude
created: 2026-09-08T12:27:51
branch: 000199-pair-term-own-the-right-pane-tab-bar
issues: [199]
---

# pair#199 — the right pane's tab strip: state, and the one thing untested

## Where this stands

**M1 and M2 are closed. M3's code is complete and smoke-tested — but only in
`pair` alone, never under couch.** That gap is the next action, and it is the
plan's own open item 2.

- **M1** — `hostty.Reservation` (`Rows`, `Edge`; `ChildRows`/`Reserve`/`Release`/
  `Paint`/`ReserveAndPaint`). The row-reservation primitive lifted out of couch
  so both consumers share it. `EdgeTop` is representable and REFUSED: a top
  reservation is not the mirror of a bottom one, because the child still
  addresses absolute rows, so its row 1 *is* the strip.
- **M2** — one writer, one gate. Every byte to the pane goes through
  `copyActiveOutput`. `paneWriter` is deliberately NOT an `io.Writer`, so a new
  door is a compile error rather than a review finding.
- **M3** — `RenderStrip` (pure) plus the wiring. Operator confirms it works.

At handoff, `sdlc milestone-close --issue 199 --milestone M3 --actual 8.17` was
still running in the background. **Check its verdict first** — if it came back
REWORK, those findings are the real next action, ahead of anything below.

## NEXT ACTION

**Test the tab strip under couch**, in a real layout3 workbench. Everything so
far ran in `pair` standalone, which leaves the plan's open item 2 unanswered:
under couch there are TWO reserved rows — couch holds the HOST terminal's bottom
row, `pair term` holds its PANE's bottom row. Different terminals, each sized
from its own `Host.Size()`, so they *should* compose — but "should" is the word
that has already cost this milestone four defects, and no unit test can see it.
Watch for an off-by-one at the boundary and for either row eating the other's
content.

First, though: `sdlc milestone-close --issue 199 --milestone M3 --actual 8.17`
was still running at handoff. **Read its verdict before anything else** — a
REWORK there outranks everything below.

## Then

1. **(the couch test above)** Everything so far ran in `pair` standalone. Under couch
   there are TWO reserved rows: couch holds the HOST terminal's bottom row,
   `pair term` holds its PANE's bottom row. They are different terminals and
   each `Reservation` is computed from its own `Host.Size()`, so they *should*
   compose — but "should" is the word that has already cost this milestone four
   defects, and no unit test can see this. Watch for an off-by-one at the
   boundary and for either row eating the other's content.
2. **M3.7(c)**, still unexercised: a wide (`日本語`) and a long tab name, to see
   truncation and column alignment against a real terminal rather than the unit
   test's arithmetic.
3. **M4** — `borderless=true` on the terminal pane at all nine `name="terminal"`
   sites. **Edit `zellij/layouts/main-3.kdl`, the SOURCE.** The path under
   `cmd/internal/runtimebundle/assets/runtime/files/` is a generated mirror that
   `make test` regenerates before any test reads it, so an edit there is
   silently reverted.

## The thing worth carrying forward

**couch's reserved row is not easier by design, it is easier by CHILD.** couch's
child is zellij — a full-screen emulator that repaints from its own model and
addresses every cell absolutely. It never relies on the terminal remembering a
cursor, and any damage a paint does is overwritten within a frame. A shell
relies on all of it and repairs none of it.

That single fact explains the whole milestone. Two bugs latent in the shared
primitive since `#146` (`SetRegion` homing the cursor; an ERASE painting with
the current background) were invisible under couch and immediately visible under
a shell. And it is why the design was wrong at first: I copied couch's
MECHANISM faithfully and never read it for POLICY. couch does not paint on
row-dirty at all — it records a debt (`couchtty/console.go:1147`) — while M3
painted on every row-dirty batch, which for a shell means painting constantly,
and constantly at the moment the child is mid-prompt.

The operator found this by asking *"why is the tab in couch seems to be easy to
set up?"*. Worth remembering as a prompt: when a sibling does the same thing
more easily, ask what is different about its INPUTS before assuming its design.

## Decisions that are settled — do not relitigate

- **The strip belongs to `pair term`, not couch.** couch's child is the whole
  zellij session, so it cannot render into a pane zellij owns; and a tab bar in
  couch would leave standalone `pair` without one, which `couch must not degrade
  pair` forbids.
- **There is no second cursor-save slot.** `probes/cursorsaveslots` measured
  `CSI s`/`CSI u` failing to restore where told while `DECSC` succeeded under
  the identical harness. So the paint must not write while the child holds a
  save — that is what `Screen.HoldsCursorSave` and the two-condition gate are.
- **Ctrl-C not stopping `yes aaa` is NOT this issue's bug.** Five measurements:
  `cmd/probes/termctrlc` stops the child in 0.04s through a throttled reader;
  the scan M2 added runs at 30 GB/s; the channel cap is unchanged from before
  `#199`; input writes straight to the child rather than queueing; and a flood
  in a real zellij pane is no slower with a scroll region than without. If it
  resurfaces, build a pre-`#199` binary and A/B rather than re-deriving.
- **The tab MODEL already existed** before this issue. `Alt+t` was never a
  zellij tab — `config.kdl:114` forwards raw `ESC t` and `handleTerminalChord`
  catches it. This issue only ever added the DISPLAY.

## Working agreements observed this session

- The operator smoke-tests in the real workbench and reports precisely (often
  with screenshots). Treat those reports as primary evidence; several were
  right when my reasoning was wrong.
- **Measure, do not reason.** Repeatedly this session a confident derivation was
  wrong and a five-minute probe settled it — the child's row count, the scroll
  region's cost, the save slots, Ctrl-C. Probes live in `probes/` (no
  arguments; `make test-smoke` runs them all) or `cmd/probes/` with their own
  target when they need one.
- Mutation-check every new test. Several passed for accidental reasons — a
  replay of `"clean"` whose `c` is a valid CSI final byte; an assert-absent with
  no positive control; a test driving the injectable seam instead of the
  production path.

## Open, not blocking

- `#211` send truncation, `#210` perf-capture hardening, `#214` racing launch
  operations (the operator hit a restart race under couch this session and it
  may be a second instance — unconfirmed).
- `TestBareCouchInstalledCommand` and `scribecmd`'s
  `TestRunTeesChildOutputToStdoutAndLog` have each flaked once and passed on
  rerun. Real-process timing, unrelated to `#199`.
