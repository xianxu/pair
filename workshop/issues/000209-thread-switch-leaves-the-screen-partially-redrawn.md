---
id: 000209
status: working
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-09
estimate_hours: 1.72
started: 2026-09-09T18:28:51-07:00
---

# thread switch leaves the screen partially redrawn

## Problem

Operator report: after switching threads in couch, the screen is sometimes not
fully redrawn. Clicking the right pane, or scrolling the agent pane a little,
brings it back.

**Both workarounds are the tell.** Neither adds information couch had — they make
*zellij* repaint the pane from its own complete buffer. That is the state couch's
reconstruction failed to reproduce.

### Mechanism

A switch reconstructs the screen from **bytes alone**. `switchTo`
(`couchtty/console.go:498`) calls:

```go
c.takeOverScreen(p.child.ReplayThrough(p.replayCutoff))
```

and `takeOverScreen` (`console.go:981-989`) is exactly two writes:

```go
io.WriteString(c.host, hostty.HomeAndClear)   // blank the screen
c.host.Write(body)                            // replay the retained tail
```

Nothing asks the child to repaint. The screen is correct **iff** the retained
tail happens to contain enough output to repaint it — and the tail is bounded:
`DefaultRingBytes = 128 * 1024` (`ptychild/ring.go:16`), whose own comment is
candid that this is an approximation: *"only the tail can repaint a screen — the
head is a screen nobody will ever see again."*

That assumption holds for a chatty child and breaks in at least four ways:

1. **The last full paint aged out.** A mostly-idle agent painted its frame more
   than 128 KiB ago. Clear-then-replay reproduces only the incremental updates
   since, over a blanked screen.
2. **Replay returns nothing at all.** `ReplayThrough` (`child.go:249-251`) returns
   `nil` when `cutoff < ringStart` — the cutoff is older than what the ring still
   holds. The screen is cleared and nothing is written.
3. **The tail begins mid-sequence.** `Ring` keeps the last N bytes and, as
   `replay.go` records, *"bisects whatever spans that boundary"* — so a replay can
   open inside an escape sequence, and `StripQueries` deliberately emits an
   unterminated escape verbatim.
4. **Mode state is not byte-replayable.** Alt-screen, scrolling region and cursor
   state are the product of the *whole* stream, not its tail. `#196` established
   this class for mouse modes: a bounded ring cannot re-derive a mode set before
   its window. Same defect, different mode.

Which of these dominates is unknown and the report's "sometimes" is consistent
with all four — see Plan step 1.

### It is a class, not a site (`ARCH-PURPOSE`)

`termcmd` does the identical thing for `pair term` tab switching
(`run.go:1021-1024`):

```go
func (m *terminalMux) redrawTab(replay []byte) {
	io.WriteString(m.stdout, hostty.HomeAndClear)
	m.stdout.Write(replay)
}
```

Same two writes, same bounded ring, same assumption. A fix that repairs only
couch leaves the same bug in the right pane's tabs. Both are consumers of the
shared `ptychild`/`hostty` split (`#146`), which is where the answer probably
belongs.

## Spec

**A switch must not depend on the retained byte tail being sufficient.** Ask the
child to repaint from its own state instead of reconstructing from history.

The mechanism already exists and is one call: `Child.Resize(Size)`
(`ptychild/child.go:195-196`) — *"The child gets SIGWINCH."* A resize nudge
(dimensions changed and restored) is the portable way to make a full-screen
child repaint from its authoritative model, and zellij — which is what couch's
children actually are — redraws its pane on SIGWINCH. That is the same thing the
operator's mouse click achieves, issued deliberately.

Byte replay stays as the immediate paint so the switch is not visibly blank
while the child responds; the nudge is what makes the result *correct* rather
than *probable*.

### To settle in the plan

1. **Which failure mode is actually firing.** The four above have different
   fixes, and a nudge only obviously fixes 1 and 2. Reproduce first — an idle
   thread left long enough to age its last paint out of 128 KiB is the cheapest
   candidate.
2. **Whether a nudge is safe here.** Resize is not free in this tree: the
   composer recognizers treat resize as *"a latched transaction: authorization
   stays closed from validation until a complete successful resize commits"*, and
   a spurious resize could interact with that latch or with a child mid-render.
   Check before adopting.
3. **Whether zellij offers a cleaner repaint request** than a synthetic resize.
   A dedicated action would avoid the reflow a resize implies.
4. **Where the fix lives.** Per the class above, prefer one answer used by both
   `couchtty` and `termcmd` over two.

Out of scope: enlarging the ring. It moves the threshold without removing the
assumption, and the memory cost is per-child across a fleet of 10+.

## Done when

- Switching to a thread whose last full paint has aged out of the ring produces
  a correct screen with no operator action — reproduced first, then fixed.
- The `cutoff < ringStart` path cannot present a cleared screen with nothing
  drawn.
- `pair term` tab switching gets the same guarantee, from the same code.
- `#196`'s reattach test still passes unmodified — the nudge must not perturb
  the mouse-mode belief on a path that shares this seam.
- A counted invariant lands in `#204`: a switch issues a repaint request, not
  only a replay write.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: pensive              design=0.15 impl=0.05
item: smaller-go-module    design=0.20 impl=0.24
item: smaller-go-module    design=0.10 impl=0.16
item: smaller-go-module    design=0.10 impl=0.16
item: smaller-go-module    design=0.05 impl=0.16
item: atlas-docs           design=0.05 impl=0.04
item: milestone-review     design=0.00 impl=0.20
design-buffer: 0.15
total: 1.72
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Rows: the reproduction (done) and its Log;
the shared `ptychild` repaint including `ReplaySafeStart`; wiring both consumers
plus the nudge; the regression suite and the `#196`/latch verification; atlas;
one close review. (`sdlc estimate-source` reports the calibration doc `[stale]`,
#127; the last three pair closes ran 0.89/1.01/0.69 against it.)

## Plan

- [x] **Reproduce deliberately — done, all four, deterministically** at the
      `ptychild` seam rather than as "sometimes" in a live session
      (`replay_insufficiency_test.go`). A live repro would have shown ONE mode;
      the seam shows that each is reachable, which is what the fix has to
      answer for. See `## Log` 2026-09-09.
- [x] **Settle items 2–4.** (3) zellij 0.44.3 has **no** repaint action —
      `clear` destroys buffers and `dump-screen` writes to a file — so a
      SIGWINCH nudge via `Child.Resize` is the mechanism. (4) The shared home is **`hostty`**, NOT
      `ptychild`: `hostty` imports `ptychild` and not the reverse
      (`hostty/fake.go:8`), and the composition emits host-side sequences, so
      `ptychild` cannot hold it without inverting the split `#146` drew
      (PQ-10). Both consumers already depend on `hostty`. (2)
      Nudge safety is checked against `#196`'s reattach test and the composer
      resize latch as its own plan row below.
- [x] **Correct the Problem section's citations first (PQ-3).** The quoted code
      and file:line cites were accurate when filed on 2026-09-06 and are not
      now: `takeOverScreen` is `console.go:992` and does more than two writes
      (it resets `hostScan`, writes, then FEEDS THE BODY BACK so the scanner
      re-learns the new child's modes), and `redrawTab` is `run.go:1658` and no
      longer writes at all — it enqueues `ptyChunk{takeover: true}`, consumed at
      `run.go:890`/`:982-983`. The mechanism the issue describes is unchanged;
      the code it points at is stale, and a plan that quotes it wrongly cannot
      be checked.
- [x] **Mode 4: an ORDERED composition, in `hostty`, over only the modes that
      can actually be restored (PQ-2, PQ-8, PQ-9, PQ-10, PQ-11).** Three
      successive versions of this row were wrong, each in a different way, and
      the constraints that survive are:

      1. **Order by side effect (PQ-8).** `?1049h`/`?1049l` switch buffers, so a
         paint belongs to whichever buffer was active when it was written.
         Buffer state is asserted BEFORE the clear; buffer-independent modes
         AFTER the tail, where they cannot be undone by it.
      2. **Cursor-save is NOT re-assertable and is dropped (PQ-11).** `\x1b7`
         and `\x1b[s` save the CURRENT cursor; no sequence injects a
         previously-saved position. `Screen.HoldsCursorSave()` is a belief about
         the child, not a state the host can restore. Listing it was a category
         error.
      3. **Never assert a mode from absence of evidence (PQ-9).** Only `mouse`
         carries an observed bit (`screen.go:47`); alt-screen and SGR-mouse do
         not, so "never witnessed" and "witnessed off" are the same value — and
         asserting `?1049l` from that would drop a child out of an alt screen it
         is really in. Add `altScreenObserved` / `sgrMouseObserved` mirroring
         `mouseObserved`, and assert a mode only on positive evidence.
      4. **Say why absence is admissible when it is.** `Screen` scans a child's
         whole stream from `Start`, so for a child we spawned an unobserved mode
         really is off. That is the exact opposite of the ring, where `#196`
         established absence proves nothing — and the two live two fields apart
         in the same struct, so the distinction is written down rather than
         assumed.

      Composition: assert buffer -> clear (or not, per intent) -> tail ->
      buffer-independent modes -> repaint request.
- [x] **Mode 3 is cosmetic once modes are asserted, and the plan says so rather
      than over-building.** A bisected tail prints an orphaned fragment as text.
      It cannot corrupt terminal STATE, because the mode assertion above runs
      after it and the repaint request overwrites the frame. Deterministically
      trimming it needs a boundary index `Screen` does not keep, and with the
      repaint landing that buys a few milliseconds of cleaner garbage. Recorded
      as an accepted limitation with its reason, not silently dropped.
- [x] **Mode 2: carry INTENT, do not infer it from an empty slice (PQ-1).**
      `m.redrawTab(nil)` at `run.go:845` is a DELIBERATE clear — it blanks the
      screen before releasing a new tab's startup output so the queued live copy
      is not duplicated (BR-9). "Do not clear when the replay is empty" would
      regress exactly that. The two callers want different things, so the seam
      states which: a repaint that has nothing to draw must NOT blank (a stale
      frame beats a blank one), while a deliberate clear still clears. Express
      it in the type — separate entry points, or an explicit field — so a future
      caller cannot get the default wrong.
- [x] **Mode 1: the repaint request, fully specified (PQ-4).** `Child.Resize`
      twice: to `Size{Cols: cols, Rows: rows - 1}` then back to the size the
      caller already owns — the console tracks `c.size`, the mux
      `childSizeLocked()`, so no new authority is introduced and there is no
      "restore size" to lose. Rows, not cols, because a column change reflows
      wrapped lines. **Failure path:** `Resize` returning an error is logged and
      dropped — the replay has already painted, so a failed nudge degrades to
      today's behaviour rather than to a blank screen; it must never propagate
      and abort a switch. **Rapid switches:** each switch nudges its own target
      once and nothing is coalesced, because a nudge is idempotent and two
      SIGWINCHes cost one extra repaint, not a wrong screen.
- [x] **The SIGWINCH assumption needs a double and a conformance check
      (PQ-5, `ARCH-MOCK`).** "zellij repaints its pane on SIGWINCH" carries the
      whole of mode 1 and is currently an assertion. `ptychild` already has
      `fake.go`; the fake models the behaviour we depend on — a resize produces
      a full repaint — so the repaint path is testable without a pty. Separately
      a live conformance check drives a real `zellij` child, nudges it, and
      asserts a full frame arrives; that is what tells us the model still
      matches the binary after an upgrade, and `#213` is a standing reminder
      that this zellij version's documented behaviour and actual behaviour
      diverge.
- [x] **Nudge safety (plan item 2).** `#196`'s reattach test passes UNMODIFIED,
      and the composer resize latch — "authorization stays closed from
      validation until a complete successful resize commits" — is untouched by a
      size-restoring nudge. If the latch is perturbed, that is a finding to
      report, not a workaround to route around.
- [x] **Tests, named, one strategy line per risky surface (PQ-6).**
      *Shared repaint:* the four `replay_insufficiency_test.go` reproductions
      become regression rows against the new path — each asserts the mode it
      demonstrates is now handled (2 and 4 fixed, 1 fixed given a repainting
      child, 3 documented). *Intent:* a table over the two call intents
      asserting a nothing-to-draw repaint emits no `HomeAndClear` while a
      deliberate clear still does. *Mode assertion:* the emitted bytes end with
      the modes `Screen` reports, over the cross-product
      {alt-screen, mouse, SGR-mouse, cursor-save} x {set, unset}. *Both
      consumers:* a differential row per `#220`'s lesson — assert the ANSWER,
      that `couchtty` and `termcmd` produce byte-identical repaint output for
      the same child state, not merely that both call the function.
- [x] **Counted invariant into `#204`:** a switch issues a repaint request, not
      only a replay write.
- [x] **Atlas:** the reconstruction rule — the replay is the immediate paint,
      the child is the authority — on the `ptychild`/`hostty` split, with the
      guarantee's dependence on what the child is.

## Log

### 2026-09-06

Filed from an operator report. The diagnosis is a code read, not a reproduction —
the mechanism is certain (byte-replay-only reconstruction over a bounded ring),
but *which* of the four failure modes produces the operator's "sometimes" is
not, which is why Plan step 1 is reproduction rather than a fix.

Worth recording that the workarounds diagnosed this: clicking the right pane and
scrolling the agent pane both cause zellij to repaint from its own complete
buffer. The operator's fix works because it routes around couch's reconstruction
entirely — so the bug is in trusting the reconstruction, not in the redraw path.

This is `#196`'s shape a second time: a bounded ring cannot re-derive state that
was established before its window, and the component acted as though absence of
evidence were evidence of absence. There it was mouse-mode tracking; here it is
the screen itself.

## Revisions

### 2026-09-09 — "same guarantee" qualified to "same mechanism"

**Reason.** Done-when says *"`pair term` tab switching gets the same guarantee,
from the same code."* The same code is right and stays. The same **guarantee**
is not achievable, and the reason is what the two children are: couch's child is
zellij, which holds a complete screen buffer and repaints all of it on SIGWINCH;
a `pair term` tab's child is a **shell**, which has no screen model at all. When
a shell's output has scrolled out of the ring, those bytes exist nowhere —
neither pair nor the shell can produce them. Nudging a bare prompt repaints the
prompt, not the history.

**Delta.** Done-when's `pair term` clause now reads: the same mechanism, with
modes 2–4 fixed for every child type and mode 1 fixed wherever the child has a
screen to repaint from (zellij, or a foreground TUI) and best-effort for a bare
shell. Operator chose this over building a per-tab VT emulator, which would fix
mode 1 everywhere at the cost of a screen model per tab across the fleet.

### 2026-09-09 — implementation, and the assumption measured

**The SIGWINCH assumption is confirmed against the real binary, not asserted.**
`probes/zellijrepaint` starts a throwaway zellij session in a pty, has its pane
print a marker ONCE and go quiet, drains what the pty saw, then resizes rows
only. Result on zellij 0.44.3 / macOS: **the resize produced 19,317 bytes and
the marker returned** while the child emitted nothing — so zellij re-rendered
the pane from its own buffer, which is exactly the property mode 1's fix rests
on. Worth having measured: `#213` established that this zellij version's
documented and actual behaviour diverge, and the whole fix would have been built
on a sentence otherwise.

That is the live half of `ARCH-MOCK`. The in-process half is
`ptychild`'s fake, which records resizes, so
`TestTabSwitchIssuesARepaintRequestAndRestoresTheSize` pins the nudge without a
pty — verified by mutation: dropping the request gives `resizes = []`.

**Three versions of the mode composition were wrong before this one**, each
caught by the plan gate rather than by me, and each for a different reason —
they are recorded in the `hostty.Repaint` doc comment because the ORDER is the
design: assert the buffer before the clear (asserting after discards the paint);
assert only what was OBSERVED (`Screen.AltScreenObserved` now sits beside
`mouseObserved`); and never list cursor-save, which no sequence can restore.

**`redrawTab(nil)` was a deliberate clear**, not an accident — it blanks a new
tab before releasing startup output so the queued live copy is not duplicated
(BR-9). "Do not clear when the replay is empty" would have regressed exactly
that, which is why intent is carried and `clearTab` is its own door.

**`#196`'s reattach test passes unmodified** — no test of it was touched, and the
composer deliberately does not assert mouse state, leaving that to the
authorities that already own it rather than adding a third writer.

**Deployment note.** These paths are compiled into long-lived processes, so
unlike `#220` a rebuild does NOT reach a running session: the couch half needs
`couch` restarted, the right-pane half needs that `pair term` restarted. And
because the failure needs a full frame to have aged out of a 128 KiB ring, a
freshly relaunched session cannot exercise it at all — absence of the symptom
right after a restart is close to no evidence either way.
