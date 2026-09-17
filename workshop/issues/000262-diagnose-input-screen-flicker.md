---
id: 000262
status: open
deps: [pair#255]
github_issue:
created: 2026-09-15
updated: 2026-09-16
estimate_hours:
---

# Diagnose input-triggered screen flicker

## Problem

The screen sometimes flickers subtly during fast Neovim input, without visible
corruption. Holding Delete and deleting one character at a time can trigger it;
running a program or producing heavy output in the right pane does not show the
same symptom.

## Spec

Instrument and reproduce the flicker at the terminal/input and compositor
boundaries. Compare rapid editor input with right-pane output, identify whether
extra redraws, synchronization, cursor updates, or scheduling cause the flash,
and fix the root cause without reducing input fidelity.

## Done when

- A bounded reproduction captures the input sequence and redraw/flush timing.
- The cause is corrected with no output corruption or lost input.
- Coverage distinguishes rapid editor input from high-volume right-pane output.
- Operator smoke testing confirms fast typing and held Delete no longer flicker.

## Revisions

### 2026-09-16 — the correlate is LOW activity, not fast input

An operator observation (recorded in `## Log` below) inverts this issue's framing.
Two things change, and the `## Problem` above is kept as filed rather than
rewritten:

1. **Confirmed, not accumulated session state.** The 2026-09-15 post-reload note
   treated the report as possibly specific to the pre-reload Pair session and said
   to keep observing before investing. That question is now answered — the flicker
   recurs, so instrumentation is justified.
2. **The trigger correlate is sparseness, not speed.** `## Problem` says rapid
   input triggers it and heavy right-pane output does not. Both new sightings are
   LOW-activity: a static agent pane with nothing but typing, and a quiet draft
   nvim while the agent works with minimal output. Held Delete is then not "fast
   input" as a cause — it is one instance of a screen with little else on it. Heavy
   output is therefore not a clean negative control; it may be MASKING the symptom.

Delta to `## Done when`: the coverage bullet distinguishing *rapid editor input
from high-volume right-pane output* is no longer the right axis. It should
distinguish a QUIET screen from a busy one, in either pane.

## Plan

- [ ] Build a focused reproduction and lightweight redraw/input timing trace.
- [ ] Compare editor input, held Delete, and high-volume right-pane output.
- [ ] Correct the responsible path and add regression/performance coverage.
- [ ] Run terminal/compositor tests and obtain operator smoke confirmation.

## Log

### 2026-09-15

Filed as a follow-up to #255 from operator smoke testing. Flicker is subtle and
non-corrupting, appears during rapid input, and has not appeared during similar
right-pane output.

### 2026-09-15 — post-reload observation

The symptom may be specific to the pre-reload Pair session or its accumulated
state. The brain thread flickered before reload, but the flicker was not
observable after reloading the Pair. Treat the report as unconfirmed for now;
continue observing before investing in instrumentation or a fix.

### 2026-09-16 — operator: it flickers when there is NOT much happening

Two regimes reported, both quiet:

- a **static agent pane**, with nothing happening but typing in the draft;
- a **quiet draft nvim** while the agent is working but putting little on screen.

Two hypotheses, and the cheap test that separates them. Both predict the quiet-screen
correlate, which is why heavy output was never a clean negative control: under
continuous output the artifact is immediately overdrawn, so only a quiet screen holds
it long enough to see.

**H1 — an escape sequence whose effect does not stop at the region boundary
(operator's guess, and the better-supported one).** This repo has hit this class
repeatedly and its defenses are known to be partial:

- `ptychild/screen.go:191` states the gap outright: *"DECSTBM restricts SCROLLING,
  not erasing -- so a full-screen child clearing the display on startup, which every
  one of them does, wipes the reserved row while the region is still perfectly
  intact."* So a scroll region does NOT contain an erase, which is exactly the
  containment the guess assumes is missing.
- `hostty/reserve.go:20` records that zellij honoring DECSTBM from a pane process is
  *measured, not assumed* — and that when it went wrong it went wrong big:
  *"pair#199 finding 5 -- 200 lines scrolled in the region while the reserved row
  held its paint."*
- `rowtext.go:9` and `couchtty/reserve.go:100` both defend against an unfiltered
  `\x1b[2J` reaching the terminal from text that should have been inert.
- **#241 is an open, confirmed leak of this family**: DEC private modes (`?1004h`,
  `?1h`, `?2004h`) survive a tab takeover and stay set on a pane that never asked
  for them.

The mechanism this suggests: a child erases the display, the reserved row (or a
neighbouring region) is wiped, and something repaints it. **The repaint is the
flicker** — blank, then restored. Continuous output hides the gap; a quiet screen
shows it as one discrete flash.

`ptychild.Screen.TakeRowDirty` (`screen.go:187`) already detects precisely this
event — *"the child did something that may have destroyed a reserved row: dropped
the scrolling region (DECSTBM, RIS, an alt-screen transition) or ERASED the
display."* So the first diagnostic is not new instrumentation: **log every
`TakeRowDirty` firing with a timestamp and see whether flicker events line up.**

**H2 — output batching (timing, not content).** The wrap loop runs two
unsynchronized tickers for the session's life: `captureTick` at 50ms
(`cmd/internal/wrapcmd/wrap.go:2825`) and `stdoutFlushTick` at
`defaultStdoutFlushInterval` = 100ms (`wrap.go:89`, ticker at `wrap.go:2839`). The
tick itself emits nothing when idle — `flushStdout` early-returns on an empty pump
(`wrap.go:513`) — but `stdoutPump` BATCHES, so under sparse output a small write can
wait a full interval and updates arrive quantized at ~10Hz.

**The discriminator, in order of cost:**

1. **Free, operator-side:** next time it flickers, note whether the flash stays
   INSIDE the pane. If it reaches another pane or the tab strip, it is H1 and H2 is
   dead.
2. **Cheap:** timestamp `TakeRowDirty` firings and check for coincidence with
   reported flickers.
3. **Only if both are inconclusive:** capture the raw byte stream pair writes to the
   terminal around a flicker. H1 leaves erase/scroll-region/cursor-positioning
   sequences in it; H2 leaves correct bytes with a gap at a tick boundary.

Unverified — no trace taken yet. Note that H1 and H2 are not exclusive: a late flush
would lengthen the window during which an erased region sits blank.

If H1 holds, the repair is containment or a repaint that closes the gap, not anything
in the batching path — a different fix from the one `## Spec` currently anticipates.

### 2026-09-16 — operator: the flicker is GLOBAL and very subtle

Ran the free discriminator from the entry above. The answer kills H2 and narrows H1
to one path.

**H2 (output batching) is dead.** `stdoutPump` feeds ONE pane's stream; it cannot
produce a whole-screen artifact.

**H1' — the reserved-row repaint is unsynchronized, and it runs on a 10Hz timer.**
Three verified facts compose into the symptom:

1. **A 100ms spinner timer drives repaints with no input and no output.**
   `cmd/internal/couchtty/console.go:681` — `spinnerTimer = time.NewTimer(100 *
   time.Millisecond)`, reset at `:683`, with `spinnerGlyph(phase)` at
   `couchtty/reserve.go:176`. While a spinner is up the strip repaints ~10×/sec
   whether or not anything else is happening.

2. **Every repaint emits a whole-screen state change, and an erase.**
   `hostty/reserve.go:124` — `ReserveAndPaint` = `SaveCursor` + `SetRegion(1,
   Rows-1)` + `drawRow` + `RestoreCursor`, where `drawRow` (`:144`) is `MoveTo(Rows,
   1)` + `ResetSGR` + `ClearLine` + text + `ResetSGR`. Two of those are not
   pane-local: DECSTBM is whole-screen scrolling state, and the code states its side
   effect outright — *"`SetRegion` (DECSTBM) HOMES THE CURSOR as a documented side
   effect"* (`:115`). So each frame blanks a full-width row and, on the
   `ReserveAndPaint` path, sends the cursor to 1,1 and back.

3. **Nothing is wrapped in synchronized output.** `grep -rn '2026h\|2026l\|?2026'`
   over `cmd/` returns NOTHING — DECSET 2026 (BSU/ESU) is not used anywhere in this
   repo. So the terminal is free to present intermediate frames: row erased but not
   yet redrawn, cursor homed but not yet restored.

Ten unsynchronized frames per second, each briefly blanking a row and moving the
cursor globally, on a screen where nothing else is changing, is a global very subtle
flicker. Heavy output masks it because the intermediate states are overdrawn before
the terminal presents them — which is why the original "heavy output does not show
it" control was misleading rather than informative.

**This also reframes the operator's own guess, and lands next to it.** The
intuition was *"adding some sequence so the effect of those printed escape sequences
stops at the boundary."* The missing containment is not spatial, it is TEMPORAL:
BSU/ESU bound which intermediate states get presented, not where they land.

**Next checks, cheapest first:**

1. **Free:** does the flicker stop when no spinner is running? If it tracks spinner
   activity, H1' is confirmed without instrumentation.
2. **Cheap:** confirm which painter the 100ms path actually reaches — `ReserveAndPaint`
   (DECSTBM + cursor homing every frame) or `Paint` (`reserve.go:160`, no region
   assert). Only the former explains a GLOBAL disturbance; if the timer path uses
   `Paint`, the whole-screen part needs another source.
3. Whether `termcmd`'s tab strip (`presentation.go:360` `paintStripLocked`, 8+ call
   sites) repaints on the same beat, compounding it.

**Likely repair, if H1' holds:** wrap each strip paint in DECSET 2026
(`\x1b[?2026h` … `\x1b[?2026l`) in `hostty/control.go` beside the other sequence
constants, applied in `drawRow` so both painters inherit it. Ghostty supports it. That
is a containment fix, not a rate fix — dropping the spinner to a slower beat would
only make the flashes rarer, not absent.

Still unverified: no trace taken, and no confirmation that a spinner was active during
either reported sighting. Check 1 settles that for free.
