---
id: 000262
status: working
deps: [pair#255]
github_issue:
created: 2026-09-15
updated: 2026-09-17
estimate_hours:
started: 2026-09-17T18:51:31-07:00
---

# Screen flicker: the compositor re-emits global terminal state every frame (#255)

## Problem

The screen sometimes flickers subtly during fast Neovim input, without visible
corruption. Holding Delete and deleting one character at a time can trigger it;
running a program or producing heavy output in the right pane does not show the
same symptom.

## Spec

**Leading cause** (evidence in `## Log`, 2026-09-16). #255 made pair a compositor:
child bytes are parsed into a cell grid and pair re-derives the parent update —
`paintPublication` (`cmd/internal/terminal/presenter.go:304`) → `Render(p.previous,
f)` (`:338`) → `p.write`. The flicker is NOT the diff tearing. The diff is
dirty-gated (`render.go:29-31`) and usually a few cells. It is the **fixed
preamble/postamble `Render` emits on every dirty frame**, constant-size and touching
WHOLE-SCREEN state no matter how small the diff:

```
preamble  render.go:35     ESC[?25l  ESC[?6l  ESC[r  ESC[?7l  ESC[0m  OSC8-close
postamble render.go:85-92  ESC[<N> q   ESC[?25h
```

That is why the effect is global while the change is one keystroke, and why it is
new: before #255 pair authored no frames, so it emitted no per-frame preamble.

**The prime suspect inside that preamble is DECSCUSR** (`ESC[<N> q`), re-issued
every frame carrying the blink bit (`if next.Cursor.Blink { code-- }`). Most
terminals RESET THE BLINK PHASE on receipt. On a sparse stream every frame restarts
the caret's blink timer — small, global, non-corrupting, and worse when quiet,
because under heavy output the cursor sits hidden between `?25l` and the next
`?25h` with no blink to disturb.

**Candidate repair: emit invariant parent state ON CHANGE, not per frame.** The
pattern is already in this file — `parentModeDelta` (`presenter.go:287`) returns
`""` when mouse tracking is unchanged, backed by `confirmedModes` + `modesKnown`.
The preamble does not participate.

**But "invariant across frames" is the wrong test on its own** (see the 2026-09-17
Log entry, from ariadne#232). Deltaing requires a SOUND BELIEF about the parent,
which needs two things the sequences do not share: a single writer, and a confirm
path. Both columns matter:

| sequence | invariant across frames | belief sound? |
|---|---|---|
| `ESC[?6l` | yes | needs exclusivity (M2) + confirm path |
| `ESC[r` | yes | **two writers today**; confirm is async-only |
| `ESC[?7l` | yes | needs exclusivity (M2) + confirm path |
| `ESC[0m` | yes | `Render` tracks `style` internally — belief is local |
| OSC8 close | yes | `Render` tracks `link` internally — belief is local |
| `ESC[<N> q` | yes | **single writer, idempotent — sound. Do this one first** |
| `ESC[?25l` / `?25h` | no | keep; legitimate per-frame |

The cursor hide/show pair stays: it exists so the caret is not seen crossing the
screen mid-paint. DECSCUSR is cleanly separable from it.

The right-hand column is the whole difference between M1 and M3. The two rows whose
belief is LOCAL to `Render` (`ESC[0m`, OSC8) are tractable; the mode rows are not,
for the reason in the prerequisite below.

**Invalidate and re-assert the full preamble on:** first paint (the existing
`!known` branch), partial or failed write, and resize. #255's plan already mandates
the second — *"on partial parent writes, retain the known accepted prefix and
invalidate the rendered-screen cache"* — and `paintPublication` already commits
`confirmedModes` only after a successful write. Same discipline, wider struct.

**PREREQUISITE — the exclusivity this rests on is false today.** The delta is only
safe if the presenter is the sole writer to the parent. #255's plan calls
`ParentPresenter` the *"exclusive typed parent-output door"* with *"no exported
generic raw-write door"*, but `hostty.Reservation` writes to the same terminal:
`SetRegion` and the DECSTBM reset. And `hostty/control.go:27` claims that sequence
*"lives here and only here"* while `terminal/render.go:35` and
`history_render.go:313` both emit it. So there are two writers, the stated invariant
is false, and `render.go`'s defensive re-assert is compensating for exactly that
rather than being paranoia.

Deltaing the preamble while a second writer can silently reset margins behind the
presenter would trade a subtle flicker for occasional real corruption — strictly
worse.

**And exclusivity is only half the blocker. The other half is confirmability.**
Even with M2 done, the presenter would be maintaining belief about state it cannot
synchronously read back: confirming a DEC mode or the scroll region means DECRQM /
DECRQSS, whose reply returns ASYNCHRONOUSLY through the input stream. So the belief
is permanently one round-trip stale — the async-confirm case, which sits nearer
write-only than controlled-proxy.

Under that classification **`render.go`'s convergent re-assert may be CORRECT for
the class, not waste to be eliminated.** A convergent write is the standard
treatment for state you cannot cheaply confirm. That reframes M3: its job is to
decide whether there is a job, not to finish one that was assumed.

Hence the milestone order below: the DECSCUSR fix needs neither exclusivity nor a
confirm path, so it ships first; the mode sequences wait on M2 and then on that
decision.

**Synchronized output (DECSET 2026) is a COMPLEMENT, not the fix.** It is
implemented at neither boundary (`grep -rn 2026 cmd/` is empty; the child's bit is
tracked and never read at `third_party/vt/mode.go:15`). Under BSU/ESU no
intermediate state is presented, which would also make the cursor hide/show pair
unnecessary. But it MASKS the per-frame churn rather than removing it, so it is
sequenced after the delta work, not instead of it.

**Constraints.**

- Never hold BSU open across an await, a blocking write, or the partial-write retry
  path `paintPublication` already has (`accepted`, `WriteFailure`). An ESU that
  never arrives is a frozen screen, worse than a tear. The ingest gate needs the
  same bounded timeout so a child that opens BSU and stalls cannot freeze the view.
- Verify 2026 nesting under zellij before shipping. Couch hosts a zellij client that
  may bracket to the real terminal itself; nesting is handled inconsistently across
  implementations. Do not assume counters.
- Advertise only what is implemented. #255's profile lists *"synchronized drawing"*
  as required, so that work closes a contract rather than adding a capability.

## Done when

- **The premise is confirmed before anything is fixed:** `Render` emits invariant
  global state on a frame that did not change it. If it does not, this Spec is wrong
  and the issue returns to diagnosis. How to check it is settled at implementation
  time — see the 2026-09-16 test-inference Log entry for what the candidate tests
  are worth.
- M1: DECSCUSR is emitted only when cursor shape or blink actually changed, asserted
  at the render seam — a one-cell diff with an unchanged cursor emits no cursor-style
  sequence. Operator smoke confirms both reported regimes.
- M1: **the comparison participates in the existing invalidation discipline.**
  `prev.Cursor` is what pair last RENDERED, which equals what the terminal holds
  only if that write landed. So the belief resets on partial write, write failure
  and resize — the way `modesKnown` already does — rather than trusting `prev`
  unconditionally. Tested for each. (Narrow and sound because cursor style has a
  single writer and the write is idempotent; it is still a belief.)
- M2: the parent has exactly ONE writer, or every other writer reports what it
  changed so the presenter's belief stays accurate. `hostty/control.go:27`'s
  *"lives here and only here"* claim is either made true or corrected.
- M3: each remaining preamble sequence is CLASSIFIED against the primitives that
  back it — single writer or not, confirm path synchronous, asynchronous, or absent
  — and the classification recorded. Deltaing a sequence whose belief cannot be
  kept sound is a finding, not a win; keeping a convergent re-assert because it is
  correct for the class is a valid outcome of M3 and closes it.
- M3: for any sequence that IS deltaed, re-assert is covered for first paint,
  partial write, write failure and resize, and no frame can leave the parent in a
  state the presenter does not believe it is in — pinned by a test diffing
  believed-vs-emitted across a frame sequence.
- M4: a frame captured while the child holds 2026 is not published until it is
  released or the bounded timeout fires; both branches tested. Nesting under zellij
  verified and the finding recorded whichever way it goes.
- Coverage distinguishes a QUIET screen from a busy one, in either pane — replacing
  the original rapid-input-vs-heavy-output axis, which the 2026-09-16 revision
  showed was measuring the masking rather than the bug.

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

### 2026-09-16 — cause identified; this issue turns from diagnosis to repair

`## Spec`, `## Done when` and `## Plan` are rewritten from "instrument and find the
cause" to the two-obligation repair above. Reason: the 2026-09-16 Log entries trace
the symptom to #255's compositor boundary and establish that DECSET 2026 is
implemented at neither end, which also explains why the flicker is NEW — the one
thing no earlier hypothesis accounted for.

Kept deliberately: `## Problem` stands as filed, and the Spec states the cause as a
leading hypothesis with a confirming step as the first `## Done when` bullet, rather
than as settled fact. Nothing here has been measured yet.

Superseded and recorded in `## Log` rather than deleted: H1' (spinner-driven
repaint — retracted, both spinner timers are gated to transient states that do not
hold in either regime) and H2 (stdout batching — killed by the flicker being
global, since `stdoutPump` feeds one pane).

The title changes to match. The filename slug does NOT, because it is the branch
name.


### 2026-09-16 — restructured around the per-frame preamble; M1 is now a one-condition fix

Second restructure today, prompted by the operator challenge *"if synchronization
was the issue, why is the effect global?"* — which was correct and changed the
diagnosis. The tearing story could not explain a global effect from a one-keystroke
diff. The per-frame preamble can, because it is constant-size and touches
whole-screen state.

Deltas:

- **Cause** narrows from "unsynchronized diff" to "per-frame re-emission of
  invariant global state", with DECSCUSR's blink-phase reset as prime suspect.
- **Fix** changes from "bracket the write in BSU/ESU" to "emit on change". BSU/ESU
  is demoted to a complement — it masks the churn instead of removing it.
- **Milestones** re-cut. The old M1 (bracket) and M2 (ingest gate) become M3's
  option and M4. The new M1 is a single condition on DECSCUSR that depends on
  nothing and plausibly fixes the whole reported symptom.
- **A prerequisite appears** that was not previously visible: the presenter is not
  the parent's exclusive writer, so most of the delta work is blocked on M2's
  reconciliation. M1 is carved out precisely because it escapes that.
- **Confirming test** changes from the bare-shell repro to `cursor-style-blink =
  false`, which is cheaper and more discriminating.

`## Problem` still stands as filed. The cause is still stated as leading rather than
settled, and the first `## Done when` bullet still exits to diagnosis if the test
fails.


## Plan

- [ ] M1 — Confirm the premise (see the test-inference Log entry). STOP and
      re-diagnose if it does not hold.
- [ ] M1 — Emit DECSCUSR only on shape/blink change, with the comparison hooked
      into the existing partial-write/failure/resize invalidation. Needs no
      exclusivity and no confirm path. Operator smoke in both quiet regimes.
- [ ] M2 — Reconcile the two parent writers: route `hostty.Reservation`'s paints
      through the presenter, or have them report their mutations. Make
      `hostty/control.go:27`'s claim true or correct it.
- [ ] M3 — Classify each remaining preamble sequence against its primitives
      (writer count, confirm path). Delta only what the primitives sustain;
      keeping a convergent re-assert is a valid outcome. Cover re-assert on first
      paint, partial write, failure and resize for anything deltaed.
- [ ] M3 — Optional, once the churn is gone: BSU/ESU around the write, gated on the
      parent advertising 2026, with no path able to emit BSU without ESU.
- [ ] M4 — Gate `capturePublication` on the child's tracked 2026 bit with a bounded
      timeout; verify nesting under zellij.

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

### 2026-09-16 — RETRACTION: the spinner is not the frequency source

The "does the flicker stop when no spinner is running" check in the entry above is
**void**, and H1's step 1 was wrong. Couch's spinner (`spinnerGlyph`, `◐◓◑◒`,
`couchtty/reserve.go:176`) is pair's own status-row animation, not the coding
agent's — and both timers that drive it are gated to narrow transient states:

- the 100ms `spinnerTimer` arms only when `focused && notice.Level ==
  MenuNoticeProgress && notice.Owner != (MenuProgressOwner{})` (`console.go`,
  `syncSpinner`) — the switcher panel focused AND showing a progress notice;
- `statusTimer` arms only while `c.menu.Reattach.Loading != (ThreadAddress{})`
  (`syncStatusTick`) — a thread mid-reattach.

Neither holds in either reported regime. There is no 10Hz repaint on a quiet
screen, so the frequency source is still open.

**What survives, and is still verified:** the paint itself is unsynchronized. No
DECSET 2026 anywhere in `cmd/` (grep returns nothing), and every strip paint erases
a full-width row, with the `ReserveAndPaint` path additionally asserting DECSTBM,
which homes the cursor (`hostty/reserve.go:115`, `:124`, `:144`). That remains a
real mechanism for a global subtle flash. It just needs a driver.

**Corrected candidate driver — the wake loop, not a timer.**
`Presenter.Present` (`cmd/internal/terminal/presenter.go:406`) does not paint: it
sets `p.dirty = e` and does a NON-BLOCKING send on `p.wake`, so a paint loop
elsewhere does the drawing and multiple events collapse into one wake. It is called
unconditionally on every child output event (`couchtty/console.go:1181`, outside the
`if changed` that guards `repaint()`).

That shape fits all three data points without a timer:

- **held Delete** (the original 2026-09-15 report) — many keystrokes, many wakes;
- **quiet agent, sparse output** — discrete individual wakes, each repaint
  individually perceptible;
- **heavy output** — wakes coalesce via the non-blocking send and successive
  repaints overdraw each other before the terminal presents them, masking it.

**Unverified, and the next thing to check:** what the wake loop actually emits per
frame — specifically whether it reaches `ReserveAndPaint` (DECSTBM + cursor homing
every frame, which would explain GLOBAL) or only `Paint`. That is the same question
listed as check 2 above; it is now the FIRST check, since the free spinner test is
void.

### 2026-09-16 — H3: #255 moved pair from passthrough to compositor, and synchronization is honored on INGEST but dropped on EMIT

Operator's framing — *"one difference compared to pre-#255 is we have proper
layers, so more processing"* — is right, and the mechanism is more specific than
cost. #255 changed the SHAPE of what pair writes to the parent terminal.

**Before #255:** the child's own bytes reached the parent. A modern TUI (nvim,
zellij) authors its own coherent update and, where the terminal supports it, wraps
that update in synchronized-output markers itself. The frame boundary was the
CHILD'S, and it survived to the terminal.

**After #255:** per `workshop/history/plans/000255-terminal-abstraction-plan.md` —
*"A shared compositor renders the selected terminal's cells and Couch/Pair chrome
into the parent terminal; raw child drawing commands never cross that composition
boundary."* Child bytes are parsed into a cell grid and pair RE-DERIVES the update:

`paintPublication` (`cmd/internal/terminal/presenter.go:304`) →
`Render(p.previous, f)` (`:338`) → `p.write(ctx, data, true)`.

`p.previous = f.Clone()` at `:360` confirms this is a frame DIFF, not a full
repaint — so the cost objection is answered, but the tearing one is not. A diff is a
multi-step mutation of the live screen: cursor positioning plus cell writes,
scattered across the pane.

**The gap, stated precisely: synchronization is honored on ingest and dropped on
emit.**

- On ingest, the design honors it — the plan commits that *"frames published during
  synchronized output remain unchanged until end-of-frame or a specified bounded
  timeout."* So a child's BSU/ESU is consumed by the parser and correctly prevents
  publishing a half-drawn frame.
- On emit, nothing replaces it. `Render`'s output is written straight to the parent
  with no wrapper, and `grep -rn '2026h\|2026l\|?2026' cmd/` returns NOTHING
  repo-wide. The parent mode delta (`desiredParentModes` / `parentModeDelta`,
  `:325-327`) carries persistent modes only; BSU/ESU is an inline bracket, not a
  persistent mode, so it is not propagated there either.

The child's frame boundary is therefore destroyed at the composition boundary and
not reconstructed. The parent terminal is free to present a partially-applied diff.

**This is the first hypothesis that explains every data point, including the one the
others could not — why it is NEW:**

| observation | explained by |
|---|---|
| global | the compositor owns the whole parent screen: pane cells + chrome |
| very subtle | intra-frame tearing of a diff, not a clear-and-repaint |
| worse when quiet | each frame is discrete and individually perceptible; under load the next diff lands before the eye resolves the last |
| held Delete (original report) | many small frames in quick succession |
| appeared around #255 | before it, the child's own synchronized update reached the terminal intact |

**Candidate repair:** wrap the rendered diff in `\x1b[?2026h` … `\x1b[?2026l` in
`paintPublication`, around the `Render` → `write` pair at `:336-341`, gated on the
parent terminal advertising the capability. Ghostty supports it. The plan already
lists *"synchronized drawing"* in the required initial profile, so this is closing a
contract #255 wrote rather than adding a feature.

**Verification before building:** confirm the parent write is genuinely unwrapped by
capturing what `p.write` emits for one frame, and confirm the terminal advertises
2026. Then the A/B is cheap and decisive — wrap it, and ask whether the flicker
goes.

Supersedes H1' (spinner-driven repaint, retracted above) and H2 (batching, killed by
the global scope). The unsynchronized-paint half of H1' survives here, with the
correct driver and the correct origin story.

### 2026-09-16 — correction: ingest is NOT honored either; the two brackets are independent obligations

**Correcting the entry above.** It said synchronized output is *"honored on ingest
and dropped on emit"*, sourced from the #255 plan's commitment that *"frames
published during synchronized output remain unchanged until end-of-frame."* That
commitment is **unbuilt**. In the vendored parser, `2026` appears exactly once —
`third_party/vt/mode.go:15`, `ansi.DECMode(2026): ansi.ModeReset`, an entry in
`resetModes`'s defaults table. The mode is RECOGNIZED and TRACKED; nothing reads it.
`Endpoint.capturePublication` (`cmd/internal/terminal/endpoint.go:242`) publishes
unconditionally. `grep -rn 2026 cmd/` is still empty.

So both ends are missing, and the failure compounds: pair may publish a frame
captured while the child is mid-update, then render an unsynchronized diff of it to
the parent. Two independent tears per flicker.

(Filed as an instance of the aspiration pattern: the plan documents this as
contracted behavior, which is why the first read of it was wrong. A plan sentence is
not evidence of an implementation.)

**Answering the design question: is this "preserve the child's bracket" or
"unwrap then re-wrap"? Neither.** A synchronized-output bracket is not payload
flowing down a pipeline — it is a framing assertion by WHOEVER IS WRITING to a
terminal. #255 created a second writer, so there are now two assertions to make, and
they do not correspond:

1. **Child → endpoint (ingest).** The child asserts *"my grid is incoherent until I
   say so."* The obligation is to gate `capturePublication` on the tracked 2026 bit,
   with a bounded timeout so a child that opens BSU and stalls cannot freeze the
   view. **Not implemented.**
2. **Presenter → parent (emit).** Pair asserts *"my diff is partially applied until
   I say so."* The obligation is to bracket the `Render` → `write` pair in
   `paintPublication` (`presenter.go:336-341`). **Not implemented.**

**Why a pass-through model is not merely unimplemented but ill-defined** — three
cases where no child bracket exists to forward:

- **Coalescing.** `Present` (`presenter.go:406`) does a NON-BLOCKING send on
  `p.wake`, so N child frames can collapse into one presenter write. One outgoing
  bracket for N incoming ones; there is no 1:1 to preserve.
- **Chrome.** A composed frame carries chrome cells (`Compose(pub.Frame, p.host,
  p.bottom)`, `presenter.go:372`) that no child authored. No child bracket covers
  them.
- **Un-bracketed children.** A plain shell, `cat`, a program that never emits 2026
  at all still produces presenter diffs that tear. Forwarding-what-was-wrapped
  leaves those broken permanently.

**That third case is a free test that discriminates the two models.** The flicker
should be reproducible with a child that never emits synchronized output — a bare
shell echoing keystrokes. If it flickers there, pass-through could never have fixed
it and the emit-side bracket is mandatory.

**Two constraints on the repair, both on the emit side:**

- Bracket only the synchronous render-and-write. Never hold BSU open across an
  await, a blocking write, or a partial-write retry — `paintPublication` already has
  a partial-write path (`accepted`, `WriteFailure`), and an ESU that never arrives is
  a frozen screen, which is worse than a tear.
- Check nesting under zellij. Couch hosts a zellij client that may itself bracket to
  the real terminal; 2026 nesting is handled inconsistently across implementations.
  Verify before shipping rather than assuming counters.

### 2026-09-16 — why GLOBAL: the per-frame preamble, not the diff

Operator challenge: *"if synchronization was the issue, why is the effect global?"*
Correct objection — a one-keystroke diff is a couple of cells, and tearing that
would be local and imperceptible. The answer is that the diff is not what carries
the global effect. `Render` (`cmd/internal/terminal/render.go:15`) wraps EVERY
frame in a constant-size preamble and postamble of WHOLE-SCREEN state, regardless of
how small the diff is.

Preamble, `render.go:35`, emitted on every dirty frame:

```
\x1b[?25l  \x1b[?6l  \x1b[r  \x1b[?7l  \x1b[0m  \x1b]8;;\x1b\\
```

Postamble, `render.go:85-92`:

```
\x1b[<N> q        (DECSCUSR — cursor shape AND blink; `if next.Cursor.Blink { code-- }`)
\x1b[?25h         (when the cursor is visible)
```

Three of those are global per-frame state changes that a small diff does not
justify:

1. **`\x1b[r` — DECSTBM reset to full screen, every frame.** Whole-screen scrolling
   state. Note this also contradicts `hostty/control.go:27`'s stated invariant that
   *"`\x1b[r` lives here and only here"* — `terminal/render.go:35` and
   `history_render.go:313` both emit it. Worth reconciling regardless of this issue.
2. **Cursor hidden and re-shown every frame** (`?25l` … `?25h`).
3. **DECSCUSR re-issued every frame**, carrying the blink bit. On most terminals
   re-issuing it RESETS THE BLINK PHASE.

(3) is the best fit for "global and very subtle", and it explains the quiet-screen
correlate better than tearing does: on a sparse stream each frame restarts the
caret's blink timer, so the cursor blinks irregularly — a small, global, non-
corrupting disturbance. Under heavy output frames are back-to-back, so the cursor
spends most of its time hidden between `?25l` and the next `?25h` and there is no
blink to disturb.

**Consequences for the Spec.**

- The framing "an unsynchronized diff tears" is DEMOTED. The diff is dirty-gated
  (`render.go:29-31` early-returns when nothing changed) and small. The global
  component is the fixed preamble/postamble.
- **Synchronized output remains a valid remedy** — BSU/ESU would stop every
  intermediate state above from being presented — but it is now the SECOND-choice
  fix, because it masks the symptom rather than removing the cause.
- **The better fix is to stop re-emitting invariant global state per frame.**
  `\x1b[r`, `?6l`, `?7l` and DECSCUSR are unchanged across almost every frame. The
  file already has the pattern: `parentModeDelta` (`presenter.go:287`) returns `""`
  when nothing changed. The preamble simply does not use it. Emit on change, not
  per frame.

**New free test, better than the bare-shell one:** disable cursor blink in Ghostty
(`cursor-style-blink = false`) and see whether the flicker stops or changes
character. If it does, (3) is implicated and the repair is the delta, not the
bracket. Costs one config line and a restart.

This supersedes the tearing mechanism in the 2026-09-16 H3 entry. The #255 origin
story is UNCHANGED and still holds — before #255 pair emitted no per-frame preamble
at all, because it was not authoring frames.

### 2026-09-16 — fix design: why "emit on change", and the exclusivity prerequisite

Recording the design discussion behind M1-M3 so the plan does not re-derive it.

**Why the per-frame re-assert exists.** It is deliberate, not sloppy.
`render.go:33-34` states the reason: *"Disable autowrap while painting the
lower-right cell, and reset origin and margins independently of whatever was on the
parent's screen before us."* The presenter declines to trust the parent's state and
re-establishes a known baseline every frame. That is a coherent position — it just
costs a constant global disturbance per frame.

**The argument against it is architectural, not stylistic.** #255's plan calls
`ParentPresenter` the *"exclusive typed parent-output door"* with *"no exported
generic raw-write door"* (plan lines 38, 283). If that holds, then after painting
frame N the presenter KNOWS the parent's margins, origin mode, autowrap, SGR and
link state, because it put them there. Re-asserting on frame N+1 defends against a
second writer the architecture says cannot exist. Either exclusivity holds and the
re-assert is dead weight, or it does not and considerably more than the preamble is
unsound.

**It does not hold.** `hostty.Reservation` writes `SetRegion` and the DECSTBM reset
to the same parent terminal, and `hostty/control.go:27` asserts that sequence
*"lives here and only here"* while `terminal/render.go:35` and
`history_render.go:313` both emit it. Two writers; the stated invariant is false.
This is a finding in its own right and may deserve its own issue — the flicker is
only how it surfaced.

**Which is why M1 is carved out the way it is.** Nothing except `Render` writes
cursor style, so making DECSCUSR conditional needs no exclusivity, no reconciliation
and no new state — it is one comparison against `prev.Cursor`, which `Render`
already holds as a parameter and already compares for dirtiness (`render.go:19`).
If the blink test implicates the cursor, this alone plausibly closes the reported
symptom. Everything else waits on M2 because deltaing margins while another writer
can reset them behind you trades a subtle flicker for occasional real corruption.

**Shape of the M3 state, when it comes.** Widen the existing pair — `confirmedModes`
+ `modesKnown` (`presenter.go:58,62`) — into a believed-parent-state struct covering
region, origin mode, autowrap, SGR, link and cursor style, with one `known` flag.
`parentModeDelta` (`:287`) is the template: return empty when unchanged, full
re-assert when `!known`. Invalidate `known` on first paint, partial write, write
failure and resize. The partial-write case is already contracted by #255's plan
(*"retain the known accepted prefix and invalidate the rendered-screen cache"*) and
`paintPublication` already commits `confirmedModes` only on success, so the
discipline exists and only needs widening.

**What stays per-frame.** The cursor hide/show pair (`?25l` … `?25h`). It exists so
the caret is not seen crossing the screen during the paint, which is a real job. It
becomes redundant only under BSU/ESU, where no intermediate state is presented at
all — one of the reasons 2026 remains worth doing after the churn is gone, as M3's
option rather than as the fix.

### 2026-09-16 — what the candidate confirming tests are actually worth

Recorded so implementation does not re-litigate it. Choosing among these is
deferred to when the work starts; this entry is only what each one can and cannot
establish.

**`cursor-style-blink = false` in Ghostty — asymmetric, weaker than first claimed.**

- Flicker stops or changes character → the cursor path is implicated, narrowing
  from seven preamble sequences to DECSCUSR. Informative.
- Nothing changes → proves NOTHING. At least three ways to get a null: Ghostty's
  `cursor-style-blink` sets a DEFAULT that an application's explicit DECSCUSR
  overrides, and pair sends the blink variant whenever the child asks for it
  (`if next.Cursor.Blink { code-- }`, `render.go:87`), so the config may never
  reach the code path at all; or the mechanism is the `?25l`/`?25h` pair rather
  than blink phase; or it is the DECSTBM reset and the cursor is innocent.

It was originally written into `## Done when` as the primary gate. It is not one —
it is a cheap shot at one suspect.

**The byte check on `Render` — this is the real gate.** `Render(prev, next)`
(`render.go:15`) is PURE and already takes `prev`. Feed it two frames differing by
one cell with an identical `Cursor` and read the output bytes: either the preamble
and DECSCUSR are there or they are not. Deterministic, no terminal, no operator.
It establishes the premise the entire Spec rests on — *invariant global state is
emitted on frames that did not change it* — and kills the Spec outright if it comes
back clean.

What it does NOT establish is causation: that this emission is what the operator
perceives. Only the A/B (make M1, smoke it) shows that.

**Judgment: do not over-instrument ahead of M1.** M1 is one comparison against
state `Render` already holds and already compares for dirtiness (`render.go:19`).
Confirmation machinery built ahead of a change that small can cost more than making
the change and smoking it.

**One assumption flagged, load-bearing and UNVERIFIED:** *"re-issuing DECSCUSR
resets the blink phase"* is a general belief about terminals and has NOT been
checked against Ghostty. The cursor suspicion rests on it; the byte check does not,
which is a further reason to lead with the byte check. If the premise confirms but
M1 does not fix the flicker, this assumption is the first place to look, and the
DECSTBM reset (`\x1b[r`, emitted every frame) becomes the next suspect among the
preamble items.

### 2026-09-17 — M3's premise is in question (from ariadne#232)

A design discussion on ariadne's ARCH-ORDER produced a classification that applies
directly here, and it is recorded in **ariadne#232** (a revision to ARCH-ORDER:
separate provenance from authority, and bound the modeled extent of external
state). Referenced, NOT a dependency — the principle revision does not gate this
fix.

**The rule:** a model of external state should be closed under the primitives that
state exposes. A modeled attribute needs both a single writer and a confirm path
before you may maintain belief about it instead of re-asserting it.

**Applied to the preamble:** confirming a DEC mode or the scroll region requires
DECRQM / DECRQSS, whose replies return asynchronously through the input stream.
Belief is therefore permanently one round-trip stale — the async-confirm case,
nearer write-only than controlled-proxy. Combined with the second writer
(`hostty.Reservation`), the mode sequences fail both tests.

**So `render.go`'s convergent re-assert may be correct for the class.** A convergent
write is the standard treatment for state you cannot cheaply confirm, and this
issue's earlier framing — that the re-assert is waste to be eliminated — was itself
the over-modeling the rule forbids. M3 is re-scoped accordingly: classify first,
and closing M3 by KEEPING the re-assert with the classification written down is a
legitimate outcome.

**M1 is unaffected in substance but gained a requirement.** Cursor style has a
single writer and an idempotent write, so the belief is sound. But it IS a belief:
`prev.Cursor` is what pair last rendered, not what the terminal holds, and
`paintPublication` has a partial-write path. M1 must therefore reset on partial
write, failure and resize like `modesKnown` does. That requirement was not written
down before and is exactly the kind of thing a one-line-looking change drops.

Two rows are unaffected by all of this: `ESC[0m` and the OSC8 close have beliefs
LOCAL to `Render` (it already tracks `style` and `link` through the paint), so they
need no external confirm path at all.

### 2026-09-17 — premise check: the hot path repaints the WHOLE SCREEN per frame; DECSCUSR is the minor term

Ran the byte check from the test-inference entry, which is the first `## Done when`
bullet. I ran it on BOTH renderers, because the Spec only ever read `Render` and that
is not the path child frames take.

**Which renderer runs.** `paintEndpoint` (`presenter.go:367`) always passes
`&pub.History`, so every child-pane frame goes through `RenderWithHistory` →
`HistoryRender.Emit` (`history_render.go:297`). `Render` is reached only through
`paint` with `history == nil`, i.e. `Presenter.Panel` (couch's switcher). Both
production presenters, couch (`couchtty/console.go:163`, writing straight to the
real terminal) and `pair term` (`termcmd/presentation.go:69`, writing into a zellij
pane), paint every child frame through `Emit`.

**Measurement.** 80×24 screen of styled text, then one `\b \b` keystroke, with
identical history (a scratch test, deleted afterwards):

| path | first paint | one-cell change |
|---|---|---|
| `RenderWithHistory` / `Emit` (normal and alt screen alike) | 2156 B | **2137 B**: 1× `ESC[23L`, 24× `ESC[2K`, every row repainted |
| `Render` (panels only) | n/a | 74 B: preamble, one cell, postamble |

So the premise holds, and it is much bigger than the Spec's premise. `Emit`'s
lower-row rebuild (`:359-400`) runs on every dirty frame. It resets the region,
inserts `height-1` blank lines at row 2 (the whole screen below row 1 goes blank),
erases every row, then repaints every row from `next.Cells`. It never consults
`previous` for cells. `previous` only decides `reset` and `sameHistoryFrame`. One
keystroke is a blank-then-restore of the entire parent screen.

**This fits every observation better than DECSCUSR does:**

- **Global:** couch's presenter owns the whole terminal (zellij client, both panes,
  tab strip, status row). A keystroke in any pane repaints all of it.
- **Subtle, sometimes:** the content is identical before and after. The terminal
  shows a flash only when its renderer snapshots between the erase and the repaint.
  A full screen is tens of KB, which the terminal's IO side consumes in several
  chunks, so that window is real but short. Nothing brackets it (no DECSET 2026).
- **Quiet screens:** each keystroke is one isolated blank-and-restore. Under heavy
  output the child pushes history every frame, so the screen genuinely scrolls and
  a repaint is indistinguishable from the content change.
- **New since #255:** `history_render.go` was created in #255 M3 (`f32bb4cf`).

DECSCUSR is still re-issued per frame on both paths (`render.go:89`,
`history_render.go:412`). The blink-phase assumption is still unverified against
Ghostty. It is now a secondary suspect, not the leading cause: M1 as planned would
change 6 bytes out of 2137 and leave the whole-screen erase in place.

**Why the full rebuild exists.** It is not convergence paranoia. The comment at
`:360` says it constructs *"canonical empty rows, then … their desired soft links"*.
A row's incoming soft-wrap flag can only be established by autowrapping through
the previous row, and zellij keeps a row's wrapped/canonical flag across `EL2`.
Only `IL` gives a pristine row, which is why the rebuild inserts lines rather than
erasing them. That constraint shapes the fix: the unit of repaint is a **soft-wrap
chain**, not a cell.

**The Spec's M2 premise is stale.** The Spec says `hostty.Reservation` is a live
second writer to the parent. In production it is not. `ReserveAndPaint` / `Paint`
/ `SetRegion` have exactly one caller, `cmd/probes/couchnestedrows`. Couch uses
`bottomReservation` only for `ChildRows` arithmetic (`console.go:1019`), and both
presenters paint their chrome through `UpdateChrome` (`console.go:1114`,
`termcmd/presentation.go:366`). #255 M3 (*"migrate Couch and Pair to owned terminal
state"*) moved the writes. The presenter IS the sole parent writer in both
production hosts. What survives of M2 is the stale *"lives here and only here"*
comment at `hostty/control.go:27`. That matters for the fix: trusting `previous`
at row granularity is the same belief `Render` already takes at cell granularity,
and the sole-writer precondition for it holds.

**Oracles available for the fix.** The xterm-headless oracle
(`tests/terminal-oracle`, node_modules present) runs locally, and
`TestHistoryWireIndependentOracle` and its siblings pass. The native zellij oracle
needs `PAIR_TERMINAL_NATIVE=1`. So a row-diff renderer can be checked end-state
against the full rebuild: same viewport, same wrap flags, same history, across
frame sequences.
