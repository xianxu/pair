---
id: 000209
status: codecomplete
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-09
estimate_hours: 1.72
started: 2026-09-09T18:28:51-07:00
actual_hours: 2.63
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
- ~~The `cutoff < ringStart` path cannot present a cleared screen with nothing
  drawn.~~ **WITHDRAWN 2026-09-10** — see the Revisions entry. A takeover always
  blanks, so an empty replay IS a cleared screen with nothing drawn until the
  child's frame lands; the answer to mode 2 is the repaint request, not a
  carve-out in the composition. Best-effort for a child with no screen model,
  per the 2026-09-09 revision. Replaced by: *the `cutoff < ringStart` path asks
  the child for a frame rather than presenting a foreign one.*
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
         is really in. Add `altScreenObserved` mirroring `mouseObserved`, and
         assert a mode only on positive evidence. (This row originally also
         asked for `sgrMouseObserved`; `mouseObserved` already covers both mouse
         fields, as PQ-9 itself noted, so only `altScreenObserved` was added —
         corrected 2026-09-10, BR-9.)
      4. **Say why absence is admissible when it is.** `Screen` scans a child's
         whole stream from `Start`, so for a child we spawned an unobserved mode
         really is off. That is the exact opposite of the ring, where `#196`
         established absence proves nothing — and the two live two fields apart
         in the same struct, so the distinction is written down rather than
         assumed.

      Composition, as SHIPPED: clear (or not, per intent) -> tail -> repaint
      request. The buffer assertion at the head was withdrawn (`?1049` moves the
      cursor), and no buffer-independent modes are emitted at all — mouse would
      be a third writer for one terminal mode, which `hostty/repaint.go` states
      as the reason. Corrected 2026-09-10 (BR-9) from a line describing a
      composition the code never had.
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
      `fake.go`; the fake models what it CAN — a resize was requested, and the
      GEOMETRY the request reads (`Size`/`RequestRepaint` joined the conformance
      stimulus set, 2026-09-10) — so the repaint path is testable without a pty.
      It cannot model "a resize produces a full repaint", which is the binary's
      behaviour and not ours; that half is the live check below, and the row
      originally claimed the fake covered it. Separately
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
      demonstrates is now handled: **2 fixed** (an empty replay emits nothing
      rather than blanking), **4 tracked but NOT asserted on the wire** (the
      child's `Screen` still knows the buffer once the sequence ages out, and
      `?1049` was withdrawn from the composition because it moves the cursor —
      the `?1047` candidate is where it returns), **1 fixed given a repainting
      child**, **3 documented**. *Intent:* a table over the two call intents
      asserting a nothing-to-draw repaint emits no `HomeAndClear` while a
      deliberate clear still does. *Mode assertion:* WITHDRAWN with the
      assertion itself. No mode is emitted, so there is no cross-product;
      `hostty`'s `TestRepaintEmitsNoCursorMovingBufferAssertion` pins the
      withdrawal and `TestRepaintForReadsTheChildsObservedModesRatherThanAssuming`
      pins the mode READ that `?1047` will need. *Both consumers:* a
      differential row per `#220`'s lesson — assert the ANSWER, that `couchtty`
      and `termcmd` produce byte-identical repaint output for the same child
      state, not merely that both call the function.
- [x] **Counted invariant into `#204`:** a switch issues a repaint request, not
      only a replay write.
- [x] **Atlas:** the reconstruction rule — the replay is the immediate paint,
      the child is the authority — on the `ptychild`/`hostty` split, with the
      guarantee's dependence on what the child is.

## Log


- 2026-09-09: closed — make test green unsandboxed (exit 0, zero FAIL); go test -race -count=3 clean on ptychild/hostty/termcmd/couchtty; make test-zellij-repaint reports settle 20ms VERDICT REPAINTED. C-1 fixed as the RULE not the instance: the keep-stale branch had zero correct callers (enumerated all five takeover sites, incl. forceSwitch which returns from the panel), so the branch and the RepaintIntent enum it existed to carve exceptions out of are both deleted; pinned by TestATakeoverAlwaysBlanksEvenWithNothingToDraw, mutation-verified. I-1 fixed by releasing geom across the settle plus a generation counter, so a mandatory Resize never waits on the optional nudge and a resize during a settle supersedes the restore; mutation-verified (disabling the generation check leaves the child at 24x80 and fails). I-2: false Run-goroutine-only claim removed; the couch-wide typed-writer door filed as #224 rather than grown into this issue. I-3: NewFakeChild starts at FakeChildSize so the double models the state the fix reads. Also found and fixed by -race what the review judged harmless: only the fake branch of resizeLocked refused a dead child, so the nudge goroutine reached pty.Setsize during pump teardown - settle now cancellable on c.done, refusal moved ahead of the fake/real split, Close takes geom. Minors: EnterAltScreen unexported, stale mutation recipe corrected, race test now waits instead of sleeping, capacity hint and floor comment corrected.; review verdict: FIX-THEN-SHIP
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

### 2026-09-10 — the nudge was a coin flip, and the probe caught it

**The boundary review's BR-3 was right, and the number is worse than "unmeasured".**
It refused the conformance evidence on the grounds that
`cmd/probes/zellijrepaint` slept 1.5 s between the shrink and the restore while
production issues both `TIOCSWINSZ` ioctls back-to-back — signals do not queue,
so zellij can take a single `SIGWINCH`, read a winsize already restored to 24
rows, and re-render nothing. The probe now drives the production sequence, and
against the real binary (zellij 0.44.3 / macOS):

| settle between shrink and restore | repainted |
|---|---|
| **none — the sequence that shipped** | **6 of 12** |
| 1 ms | 5 of 5 |
| 2 ms | 5 of 5 |
| 5 ms | 8 of 8 |
| 20 ms | 3 of 3 |
| 50 ms | 3 of 3 |

So mode 1's fix worked about half the time, and every test in the tree was green
for it: the fake records that `Resize` was CALLED, which is not the property the
fix depends on. This is `ARCH-MOCK`'s point stated as a measurement — the
in-process double models the request, only the live check models the answer.

**Fixed with a measured settle, not a guessed one.** `ptychild.RepaintSettle` (then unexported) is
20 ms: the observed floor is under a millisecond, and 20 ms is about four of
this host's process wake-ups (`#204` measured 4.92 ms) of headroom. It is paid
once per switch keystroke, after the replay has already put a frame on screen,
and it BLOCKS on the goroutine that already serializes resizes — deliberately,
because a timer would reopen exactly the interleaving BR-7 closed. Pinned by
`TestRequestRepaintLeavesTheShrinkStandingLongEnoughToBeSeen`, verified by
mutation (dropping the sleep gives 666 ns and fails).

`PAIR_PROBE_SETTLE` now parameterises the probe, so the table above is
reproducible rather than a claim.

**The nudge cost is also now a number, not a prose assurance (BR-13).** The
back-to-back sequence re-renders **6.6 KB** on a single-pane session, against
19.3 KB for the sequence with a gap. Both are in `#204`'s table alongside the
counted invariant, which is where BR-9 said the invariant belonged and where it
now is.

**couch's half is pinned at last (BR-2).** The review measured that deleting
`RequestRepaint` from `switchTo` left this suite green, and couch is the
operator's report. `TestSwitchAsksTheIncomingChildToRepaint` now fails on that
deletion (verified: "timed out waiting for the incoming child to be asked to
repaint"), and `TestSwitchingToTheActiveThreadAsksForNoRepaint` pins the other
side — a landing on the actor already current pays no reflow.

**The `ChildModes` wiring could not be pinned where it was, so it moved.**
Both consumers built a `ChildModes` literal from their child, and neither could
be defended by a test: with the buffer assertion withdrawn, `ChildModes` has no
effect on the emitted bytes, so a correct literal and the zero value are
indistinguishable downstream. `hostty.RepaintFor(child, replay, intent)` is now
the one read, tested once in `hostty` against the VALUES rather than the bytes,
and it retires `Child.AltScreenObserved` (BR-12's dead exported surface) by
leaving `RepaintModes` as the only door.

**The differential row is honest about how it closes (BR-9).** Nothing in this
tree can drive both consoles in one test, so it closes transitively and each leg
asserts something real: `hostty`'s golden fixes the exact composition, and each
console asserts that what it WROTE equals `RepaintFor`'s output for that child
and replay (`TestSwitchWritesExactlyTheComposedRepaint`,
`TestTakeoverWritesExactlyTheComposedRepaint`). If either console adds or drops
a byte of its own, that console's test fails.

**Remaining findings.** BR-6's second half landed: `termcmd` fed the gate only
the replay while couch fed the composed bytes — a divergence in a shared
primitive, which is BR-77 and BR-82 both. Both now feed the composition. It
changes no behaviour today (the prefix is `HomeAndClear`, framing-complete and
mode-neutral) and is written now so the `?1047` candidate does not need either
console to remember. BR-10 had also RECURRED one function over: `RequestRepaint`
was inserted between `Child.Resize`'s doc comment and its func, so godoc
attributed the resize line to the nudge and left `Resize` bare. BR-11's
`'+chr(39)+'` escape artifact is gone.

### 2026-09-10 (second pass) — the settle was right and its two edges were not

The close review returned REWORK on two Criticals, and both are cases where
fixing the mechanism left something around it pointing the wrong way.

**C1 — the probe defaulted to the sequence its own table condemns.** Adding
`PAIR_PROBE_SETTLE` gave it a default of 0, which is exactly the back-to-back
sequence measured at 6-of-12, and `make test-smoke` runs every `probes/*/`
unattended with `|| exit 1` — so the standing conformance check would have
aborted the smoke suite about half the time, while its own comment still claimed
production issued no gap. A probe that restates the thing it verifies measures
itself. It now READS `ptychild.RepaintSettle`, which means it had to move to
`cmd/probes/` (Go forbids `probes/` from importing `cmd/internal/…`) with its
own target in the same commit — the branch of the rule `atlas/index.md` already
records, and the second time this repo has hit it. The move surfaced a second
defect for free: the probe resolved `config.kdl` by counting `..` from its own
directory, so it came back PROBE-INCONCLUSIVE at the new depth. Re-run through
`make test-zellij-repaint`: settle 20 ms, **REPAINTED**.

**C2 — the nudge's safety rule was a comment, and couch broke it in the same
round.** `RequestRepaint(size)` took the size to restore, so a stale restore was
EXPRESSIBLE, and correctness depended on every caller staying on the goroutine
that serialized its other resizes. `switchTo` is reached from the operationQueue
goroutine as well as the Run loop — the switcher's Enter and the status-chip
click, which is the operator's primary gesture. So the rule was already false
where it mattered most.

The fix is to delete the parameter rather than to document harder. `Child` owns
its geometry: `geom` guards every size change, the nudge holds it across the
whole shrink-settle-restore, and the size it restores is the one IT read under
that lock. A resize from any goroutine now either precedes the nudge (which
picks it up) or waits and applies after — the child ends at the newest size
either way, and no caller has an ordering obligation left to get wrong. Pinned
by `TestAResizeDuringANudgeWinsWhicheverGoroutineItComesFrom`, and worth
recording that the FIRST version of that test was green against the defect it
names: it read the size before the restore leg had run. A race test that does
not wait for the racing party is not a race test.

Removing the parameter also made the settle affordable asynchronously, which
retires the coalescing question rather than re-costing it: the nudge no longer
occupies the caller's event loop for 20 ms, and a second request while one is in
flight is dropped, so held key-repeat switching costs one settle rather than one
per keystroke.

**I2 — the takeover and the repaint request are now one act.** `removeTab` hands
the screen to a surviving tab and asked nobody to repaint, so closing a tab
could reproduce the issue's own symptom. The enumeration had been swept for
*intent* and not for the *request* — the same four sites, the same class, one
lens applied. Rather than add the call at the fourth site, the request rides
inside `applyTakeover`/`takeOverScreen`, so a site cannot compose a screen
without asking the child for the frame the ring may no longer hold. That also
deletes `ptyChunk.nudge`/`nudgeSize` (a parallel seam beside `onWriter`) and
carries `RepaintIntent` end-to-end instead of a bool converted back.

**I1 — three prior fixes reverted silently, and the rule is the deliverable.**
The review measured it: BR-4, BR-7 and BR-8 could each be reverted with the
suite green. A disposition of `addressed` needs a test that goes red on the
revert, and producing it is part of the fix. Done for BR-4
(`TestClosingATabBlanksTheDeadTabsScreenEvenWithNothingToDraw`) and BR-7 (now
subsumed by C2's goroutine-independence test). BR-8 is the honest exception and
is recorded as one rather than papered over: `RepaintReplace` and `RepaintClear`
emit identical bytes for any NON-empty body, and the panel always renders
something, so the intent is byte-inert exactly as BR-6 is. The test kept there
pins what it can — opening the panel blanks the child's frame — and says so.

### 2026-09-10 (third pass) — the empty-replay rule had a hidden premise

**C-1: "a stale frame beats a blank one" is only true when the stale frame is
the SAME CHILD's, and no takeover site is that case.** This is the third finding
in the `absent-data-is-not-intent` family, and the first two fixed instances:
PQ-1 gave `newTab` its own door, BR-4 gave `removeTab` a clear. Both were sites
opting OUT of the keep-stale rule — which should have been the tell. Enumerate
all five and the rule has no correct instance:

| site | what is on screen | wanted |
|---|---|---|
| couch `switchTo` | the previous thread, or the panel | blank |
| couch `showMenu` | a child | blank |
| termcmd `switchRelative` | the previous tab | blank |
| termcmd `removeTab` | the destroyed tab | blank |
| termcmd `newTab` | the previous tab | blank |

Even `forceSwitch` — the one candidate, since it repaints an actor that is
already active — is "returning from the panel, where the SCREEN changed but the
active actor did not", so the frame standing there is the panel's.

The failure it caused was on the primary flow: start a thread from the panel,
and the attach seeds `replayCutoff` from `ReplaySafeEnd()` before `Run` has
drained anything, so `ReplayThrough` returns nothing, the composition returns
zero bytes, and **the panel's own menu body stays on the terminal under the new
thread's label**. Before this issue touched it, that was `HomeAndClear` and
nothing: blank, which is honest. A foreign frame under the wrong label is the
misleading version of the same wrong.

So the branch is deleted rather than re-pointed, and `RepaintIntent` went with
it: the enum existed only to let three sites escape the rule, so with the rule
gone it distinguished nothing — and an intent parameter that changes no bytes is
a trap, because a site can pass the wrong one and nothing says so. `clearTab`
survives as a NAME (`redrawTab(nil, nil)`) because the call site reads better,
not as a second behaviour.

Worth stating plainly: mode 2's answer was never the empty-replay carve-out. It
is the repaint request, which is now measured to work. A blank frame for one
settle while the child redraws is the honest version.

**I-1: the optional nudge was gating the mandatory resize.** Holding `geom`
across the settle blocked a concurrent `Resize` for a measured 20.8 ms — and in
`termcmd` that resize runs on the writer goroutine, the sole writer of the pane,
so a SIGWINCH landing inside a switch stalled all output for a settle. `#204`
said the nudge costs the event loop zero, which was true only of the caller that
*requests* it. The nudge now releases the lock while it settles and carries a
generation: if a caller resized meanwhile, the restore leg SKIPS rather than
writing back a size nobody asked for. Optional work must never gate the
mandatory kind — and must never win a race against it either.

**I-2: `takeOverScreen` still claimed "Run-goroutine-only, like every other
writer", and this round's own test drives it from the operationQueue.** C2's
whole discovery was that `switchTo` has two calling goroutines; the nudge's half
became a mechanism and the writer's half stayed a sentence. couch's `c.host` is
a bare `io.Writer` with no serialization where `termcmd`'s `paneWriter` is
deliberately not one. The false sentence is gone and the divergence is filed as
`#224` — a couch-wide change `#209` has no business growing into. What `#209`
owed was not leaving the claim behind, because a claim like that is what lets
the next reader believe the rule is already kept.

**I-3: the fake and the real `Child` disagreed about the state the fix reads.**
`Start` records `opts.Size`, so a real child always has geometry; `NewFakeChild`
had none, so it declined every repaint request until a test remembered to resize
it — a green test for a path production would have nudged. Production cannot
reach the zero-geometry state the fake started in. Fakes now start at
`FakeChildSize`.

**And `-race` found the sub-point of I-2 that the review called harmless.** The
reviewer noted `go c.nudge()` has no tie to `c.done`, so the goroutine outlives
`Close` by a settle, and judged it harmless because "both branches of
`resizeLocked` error on a dead child". Only the FAKE branch did. Under `-race`:
`pty.Setsize` reading the fd while the pump's teardown destroyed it. Three
things now, and each is the general form rather than the instance — the settle
is cancellable on `c.done` (optional work must not outlive the thing it is
optional about), the dead-child refusal moved AHEAD of the fake/real split
(where the double was stricter than production, which is the direction that
hides bugs), and `Close` takes `geom` so an in-flight ioctl finishes before the
fd goes. Clean over `-count=3 -race` on all four packages.

Minors: `hostty.EnterAltScreen` is unexported (its only consumer is this
package's own withdrawal test — exported surface needs a consumer outside its
own package's tests); the mutation recipe in `console_test.go` named a signature
and call site that C2 deleted; the race test slept toward the nudge instead of
waiting for it, which is the same "did not wait for the other party" mistake its
own comment teaches; the capacity hint budgeted bytes the withdrawal guarantees
are never emitted; and `nudge`'s floor comment counted a reserved row that is
already subtracted from `c.size`.

### 2026-09-10 (fourth pass) — the sweep, and a guard that measured itself down

**The C-1 withdrawal reached the code and stopped there.** Seven prose sites
went on teaching the deleted rule, `atlas/architecture.md` among them — the
durable map, stating an intent parameter that no longer exists and a "stale
frame beats a blank one" rationale that C-1 refuted. Two were MUTATION RECIPES
naming a flag a reader cannot find, which reads as "this test is unpinned": the
precise failure a findings ledger exists to prevent. All seven swept.

**The rule behind it is real; the obvious guard for it is not, and that is
measured rather than assumed.** The review asked for a check that a symbol named
in a comment resolves — the repo already writes AST guards for this family. I
wrote it. It found 13 citations at HEAD and **7 of them are legitimate**:

> `pair#170 M4 deleted TestFleetPolicyResolverConformance…`
> `DELIBERATE CHANGE from the couchtty.PaintRow this replaces…`
> `Ported from termcmd's TestUpdateMouseMode, which this scanner absorbs…`
> `Was TestTerminalMuxChildUsesFullPaneHeight, which asserted the pre-#199 contract…`

Naming a symbol that no longer exists is how this repo records what it deleted
and why — a virtue, not a defect. A guard against it is a guard against the
house style, and at a ~70% false-positive rate it would be muted within a month
(`#204`'s own thesis about thresholds, arriving in a different form). So it does
not ship, and the reason is written here rather than discovered again.

What separates the two cases is TENSE, not syntax, and grammar-matching is what
`#199` M2 round 6 retired a guard for. **The idea worth keeping for whoever
tries next: resolve the cited name against git history.** A symbol that once
existed is a historical reference; one that never existed is a typo or a stale
rename. That distinguishes all 13 correctly. It costs a `git log -S` per symbol
and makes a test depend on repository history, which is why it is a note and not
this issue's work.

**The attempt paid for itself anyway** — it found three genuinely broken
pointers, all pre-existing and none from `#209`, each one a reader following a
citation into nothing:

| site | said | is |
|---|---|---|
| `couchcmd/run_test.go:1027` | `TestCLIAcceptsExactlyTheDeclaredOperations` | `TestTypedRegistryResolvesExactlyDeclaredOperations` (a doc naming a rename that never reached it — BR-10's family, missed by `stolenFrom` because the cited name is not declared in the file) |
| `couchtty/console_test.go:637` | `TestPanelIsNotPaintedOverByABackgroundChild` | no such test; the path is covered by this issue's own `TestOpeningThePanelBlanksTheChildsScreenDeliberately` |
| `launcher/args_test.go:246` | `TestParseListIsNative` | `TestParseLaunchArgsListIsNative` |

**BR-17 got the red-on-revert test it was owed.**
`TestFakeAndRealChildAgreeOnGeometryBeforeAnyoneResizes` puts geometry in the
conformance set beside the lifecycle stimuli, and deleting `size: fakeChildSize`
now fails with `Size() = {0 0}`. `replayChild` — a fixture that builds a `Child`
literal to get a small ring — carried the zero-geometry shape too, and now
carries the same default.

**And `Size()` stopped lying.** It reported the SHRUNK value for the length of a
settle, because the nudge's legs went through the same path that records intent.
They are not intent: they poke the pty and put it straight back. `setSizeLocked`
(writes the pty) is now split from `resizeLocked` (records what the child is
MEANT to be), so the one accessor a debugger would ask during a settle answers
correctly. `hostty.ChildModes` and `ptychild.fakeChildSize` are unexported for
the BR-18 rule — exported surface needs a consumer outside its own package's
tests.

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
`cmd/probes/zellijrepaint` starts a throwaway zellij session in a pty, has its pane
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

### 2026-09-10 — three plan rows corrected to what shipped (BR-1, BR-9)

**Reason.** The boundary review found three ticked rows describing a design the
code does not have. A plan that misdescribes the tree cannot be checked against
it, which is the whole point of ticking a row.

**Delta.**

1. *Tests row, mode-assertion cross-product* — asked for
   {alt-screen, mouse, SGR-mouse, cursor-save} × {set, unset}. The design
   asserts no mode at all since the `?1049` withdrawal, so there is no
   cross-product; the row now names the two tests that replaced it. This is
   BR-1, carried from PQ-12. **Claimed applied on 2026-09-10 and was not** —
   the edit script aborted on a later assertion before writing, so the entry
   described a change nobody had made, which the next review caught and is
   worse than the stale row it claimed to fix. Applied 2026-09-10 (second
   pass), together with the same row's "2 and 4 fixed" claim: mode 4 is
   tracked in the child's `Screen` but not asserted on the wire.
2. *Mode-4 item 3* — claimed `altScreenObserved` **and** `sgrMouseObserved` were
   added. Only `altScreenObserved` exists; `mouseObserved` already covers both
   mouse fields, as PQ-9 itself said.
3. *Mode-4 composition line* — read "assert buffer → clear → tail →
   buffer-independent modes → repaint request". No buffer-independent modes were
   ever emitted, and the buffer assertion is withdrawn. It now states the
   composition as shipped.

The two Tests rows the review found delivered-short are the exception and are
NOT rewritten, because they were delivered this round instead: the four
`replay_insufficiency_test.go` reproductions now carry their answers, and the
differential row exists (see the Log entry above for how it closes).

### 2026-09-10 (second pass) — the probe's home, the nudge's contract, the envelope

**Reason.** The close review returned REWORK; three of its recommendations are
plan-level rather than code-level.

**Delta.**

1. *Conformance-check row* — said the probe "drives the production sequence". It
   drove a 0 ms settle by default while production settled 20 ms. The probe now
   READS `ptychild.RepaintSettle` and lives at `cmd/probes/zellijrepaint` with
   its own `make test-zellij-repaint` target, because importing `cmd/internal/…`
   is forbidden from `probes/`. `PAIR_PROBE_SETTLE` stays as the override that
   re-measures the table.
2. *Nudge-safety row (plan item 2)* — asserted couch was safe because "both
   `onResize` and `switchTo` run on the Run goroutine". **That is false**:
   `switchTo` is also reached from the operationQueue goroutine via
   `ExecuteConsoleOperation`, which is the switcher's Enter and the status-chip
   click. The assumption is replaced by a mechanism — `Child` owns its geometry
   and `RequestRepaint` takes no size — so the row's claim is now about the type
   rather than about which goroutine a caller happens to be on.
3. *Nudge-coalescing row* — "two SIGWINCHes cost one extra repaint, not a wrong
   screen" was costed when the nudge was free, and a 20 ms settle is not free.
   Re-costed rather than restated: the nudge is asynchronous, so it costs the
   event loop nothing, and a request arriving while one is in flight is dropped.
   Held key-repeat switching therefore costs ONE settle for the whole burst, and
   the bound is one in-flight nudge per child.

### 2026-09-10 (third pass) — the empty-replay carve-out is withdrawn

**Reason.** `RepaintReplace`'s "emit nothing rather than blank" turned out to
have no correct call site (see `## Log`, C-1), so the plan rows that specify it
describe a behaviour the tree no longer has and should not get back.

**Delta.**

1. *Mode-2 row* — read "carry INTENT, do not infer it from an empty slice
   (PQ-1)", and specified that "a repaint that has nothing to draw must NOT
   blank (a stale frame beats a blank one)". **Withdrawn.** The premise was that
   the stale frame belongs to the child being repainted; none of the five
   takeover sites is that case. A takeover always blanks, and mode 2's real
   answer is the repaint request. `clearTab` remains as a name, not a second
   behaviour.
2. *Tests row, "Intent"* — asked for "a table over the two call intents". There
   is one intent now; the table asserts that a takeover blanks whatever it has
   to draw, which is the property that was got wrong twice.
3. *Nudge-envelope row* — the "costs the event loop zero" claim was true only
   for the requesting caller. Restated: the nudge releases the child's geometry
   lock while it settles, so a mandatory resize never waits on it, and a resize
   during a settle supersedes the restore rather than being undone by it.

### 2026-09-10 (fourth pass) — Done-when clause 2 withdrawn

**Reason.** C-1 reversed the empty-replay behaviour, and a reversed commitment
has to be revised everywhere it was written — Plan rows, `## Done when`, and the
atlas — not only in the rows a review happened to name. The third pass revised
three Plan rows and left the Done-when bullet standing, which is the criterion
the merge-time `specs` judge reads.

**Delta.**

1. *Done-when clause 2* — "The `cutoff < ringStart` path cannot present a
   cleared screen with nothing drawn" is **withdrawn**, struck in place with its
   reason. A takeover always blanks, so an empty replay is exactly that until
   the child's frame lands. It is replaced by the commitment the fix actually
   makes: that path ASKS THE CHILD for a frame rather than presenting a foreign
   one. Best-effort for a child with no screen model, per the 2026-09-09
   revision.
2. *ARCH-MOCK Plan row* — it claimed the fake "models the behaviour we depend
   on". Geometry is part of that behaviour and was not modelled; `Size` and
   `RequestRepaint` are now in the conformance stimulus set.
