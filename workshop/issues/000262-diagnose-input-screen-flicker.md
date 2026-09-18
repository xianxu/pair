---
id: 000262
status: working
deps: [pair#255]
github_issue:
created: 2026-09-15
updated: 2026-09-17
estimate_hours: 2.45
started: 2026-09-17T18:51:31-07:00
---

# Screen flicker: the compositor re-emits global terminal state every frame (#255)

## Problem

The screen sometimes flickers subtly during fast Neovim input, without visible
corruption. Holding Delete and deleting one character at a time can trigger it;
running a program or producing heavy output in the right pane does not show the
same symptom.

## Spec

**Cause** (evidence in `## Log`, 2026-09-17 premise check). #255 made pair a
compositor. Child bytes are parsed into a cell grid, and the presenter re-derives
every parent write from grid state. The child-pane path, `paintEndpoint` →
`RenderWithHistory` → `HistoryRender.Emit` (`history_render.go:297`), **erases and
repaints the whole parent screen on every dirty frame**. On an 80×24 screen that
is 2137 bytes for one keystroke, against 2156 for a first paint: `ESC[23L` blanks
every row below the first, each row is erased, then every row is repainted with
content that did not change. Under couch the parent is the real terminal, and the
grid is the whole window (zellij client, both panes, tab strip, status row). Ghostty
draws on its own thread from whatever state it holds when a display frame is due.
A frame that lands mid-repaint shows a blank or half-blank window for one display
frame. That is the global, fast, uniform, quiet-screen flicker.

**Fix: synchronized output (DECSET 2026) on the emit side.** Every frame the
presenter paints is bracketed `ESC[?2026h` … `ESC[?2026l`. While the mode is set,
the terminal keeps parsing into its grid and defers drawing. At the end marker it
draws the resulting state, so the identical-content erase-and-redraw never reaches
the screen. This is the contract any program that repaints a terminal is expected
to keep (nvim, tmux and zellij bracket their own frames). #255 dropped the child's
markers at the composition boundary and did not add pair's own.

Design decisions:

- **The bracket lives in the two pure renderers, not the presenter.** `Render`
  wraps its single buffer. `Emit` adds the begin marker as its first byte and the
  end marker as its last. An alt-screen switch stays inside the bracket, so opening
  or leaving nvim is atomic too. Write boundaries do not change, except that on an
  alt-screen switch the begin marker is flushed as its own write ahead of the
  `ESC[?1049h`/`l` packet. That packet must stay whole: `Presenter.write` tracks
  alt-screen ownership by exact match (`presenter.go:196`). A frame with nothing to
  draw (`Render` returns `nil`, `Emit` with `!dirty`) emits no bracket. (ARCH-PURE)
- **Unconditional, not gated on a capability query.** A terminal that does not
  implement mode 2026 ignores the DECSET, as it does any unrecognised private
  mode. Gating would need a DECRQM round-trip, whose reply arrives asynchronously:
  a permanently one-round-trip-stale belief, the class the 2026-09-17 ariadne#232
  entry says not to model. The bracket keeps no state between frames, so there is
  nothing to believe.
- **An unmatched begin marker is closed on release.** A frame write that fails
  after the begin marker was accepted leaves sync open at the parent, and the
  presenter goes `Failed` (`p.fail`). `parentReleaseControls` gains `ESC[?2026l`
  right after its CAN/ST abort, so release always closes it. That is the same
  discipline the release path already applies to margins, modes and cursor.
  Between a failure and release, the terminal's own sync timeout bounds the freeze.
- **Sync is never open across an idle period.** The bracket opens and closes inside
  one synchronous `Render`+write or `Emit` call. Each write is bounded by
  `WriteTimeout` (2s), and the presenter fails rather than retrying.
- **Shared cursor epilogue.** Both renderers end with the same CUP + DECSCUSR +
  `?25h` sequence (`render.go:80-92`, `history_render.go:403-415`), duplicated
  today. M1 extracts it into one helper, alongside the bracket constants, so M2's
  DECSCUSR decision has a single site. Behaviour unchanged. (ARCH-DRY)

**Out of M1, deliberately:**

- **The row diff.** Bracketing makes the full repaint invisible, but it does not
  make it cheaper: a keystroke still ships a full screen of bytes, which a local
  terminal parses far faster than a person types. The row diff becomes worth
  building if pair's output starts crossing a network (#120, remote terminal
  stream) or a measurement shows the bytes matter. The chain-granular design is
  recorded in the 2026-09-17 Log entry.
- **DECSCUSR on change.** Inside the bracket, a re-issued DECSCUSR cannot be seen
  mid-frame. Whether re-issuing it resets the caret's blink phase is still
  unverified against Ghostty. It is M2's question, decided on smoke evidence.
- **Ingest is already honoured, so it is not a milestone.** `Endpoint` withholds
  publication while the child holds 2026 (`endpoint.go:126-134`), recovers after
  `SyncTimeout` = 150ms (`:258-262`, `profile.go:25`), and gives the presenter that
  deadline through `NextPublication` (`:404`). The endpoint answers the child's
  DECRQM for 2026 (`terminalqualify/input_cases.go:43`). Landed in #255 M2
  (`d44ff360`), pinned by `TestEndpointSyncWithholdsThenRecoversAndCopies`. With
  M1, the pipeline is atomic end to end: pair publishes only frames the child has
  finished, and presents each as one bracketed frame. Nesting under zellij is
  moot: the endpoint CONSUMES the child's bracket, so the parent only ever sees
  pair's single bracket per frame.

**The Spec's former M2 premise is retracted.** `hostty.Reservation`'s painters have
one caller, `cmd/probes/couchnestedrows`. The presenter is the sole parent writer
in both production hosts. What remains is stale prose. `hostty/reserve.go:13-18`
says the painters are *"shared by two consumers"* (couch and `pair term`) and that
*"`\x1b[r` lives here and only here"*, and `atlas/architecture.md` describes the
deleted pre-#255 console-write door as live. Both are corrected in M1. (The
earlier `control.go:27` citation was wrong: that comment is true as written.)

## Done when

- M1: every frame either renderer emits is bracketed: the first byte of the frame
  is `ESC[?2026h`, the last is `ESC[?2026l`, and no frame byte falls outside the
  bracket. This is pinned for `Render`, `Emit` (normal, alt enter, alt leave,
  history push) and the presenter's written stream. A no-op frame emits nothing.
- M1: for every accepted-prefix length of a failed frame write, the presenter's
  release output closes sync: after the last accepted `ESC[?2026h` there is an
  `ESC[?2026l`. Tested across failure positions, not one.
- M1: the xterm oracle suites still pass, since the bracket must not change the
  terminal end state. Existing byte-exact expectations are updated with the
  bracket.
- M1: prose presenting the pre-#255 reserved-row painters as live is retired:
  `hostty/reserve.go:13-18` and the matching `atlas/architecture.md` paragraphs,
  with a sweep recorded in `## Log`.
- M1: operator smoke under couch in both quiet regimes (static agent pane with
  typing in the draft; quiet draft while the agent works): the global flicker is
  gone. `pair term` under plain zellij is smoked too, and whether zellij honours
  2026 from a pane is recorded whichever way it goes.
- M2: each remaining per-frame sequence, DECSCUSR first, is classified against its
  primitives (writer count; confirm path synchronous, asynchronous or absent).
  Deltaing only what the primitives sustain, and keeping a convergent re-assert,
  are both valid outcomes. The row-diff trigger is recorded with it.
- ~~M3: ingest gate on the child's 2026~~: already satisfied by #255 M2 (see
  `## Spec`); no milestone.
- Coverage distinguishes a QUIET screen from a busy one, in either pane.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional. Covers the whole issue as re-cut
(M1 + M2), counting only work inside the measured window, which starts at the
claim. Line by line:

- issue-spec: the in-window re-diagnosis, discussion and plan. The pre-claim
  diagnosis is sunk.
- smaller-go-module ×2 (M1): renderers and release; the cut-sweep harness. ×0.2
  design, since the plan pre-resolves them.
- real-api-discovery: the native zellij oracle.
- atlas-docs (M1): the stale-prose sweep. ×0.2, since the plan lists the sites.
- smaller-go-module (M2): mostly classification, with one open decision
  (DECSCUSR), so ×0.5 design.
- atlas-docs (M2).
- milestone-review ×2.
- real-api-discovery: the operator's Ghostty smoke, a conformance check.

Buffer +15%. The undiscounted design is in-window and small, and the M1 plan is
thorough.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.50 impl=0.08
item: smaller-go-module design=0.06 impl=0.14
item: smaller-go-module design=0.06 impl=0.14
item: real-api-discovery design=0.00 impl=0.18
item: atlas-docs design=0.03 impl=0.06
item: smaller-go-module design=0.15 impl=0.14
item: atlas-docs design=0.10 impl=0.04
item: milestone-review design=0.10 impl=0.14
item: milestone-review design=0.10 impl=0.14
item: real-api-discovery design=0.00 impl=0.12
design-buffer: 0.15
total: 2.45
```

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

### 2026-09-17 — premise check re-diagnosed the cause; M1 becomes the 2026 emit bracket

The first `## Done when` bullet (confirm the premise) ran. It found the child-pane
path repaints the WHOLE SCREEN per dirty frame (`## Log`, 2026-09-17), which
dwarfs the per-frame preamble the Spec had been built around. Operator discussion
the same day settled the fix. The erase-and-identical-redraw is invisible under
DECSET 2026, because the terminal draws only the end state. That is what the mode
is for. Synchronized output moves from "complement, after the delta work" to the
fix.

Deltas:

- **Cause:** "per-frame re-emission of invariant global state, DECSCUSR prime
  suspect" becomes "whole-screen erase and repaint per frame, unbracketed".
  DECSCUSR is demoted to a secondary suspect.
- **Fix:** "emit on change" becomes "bracket every frame in 2026", unconditional,
  in the pure renderers, with the end marker guaranteed on release.
- **Considered and deferred, with the operator:** a chain-granular row diff in
  `Emit`. It removes the bytes rather than hiding them, at the cost of new
  soft-wrap-sensitive render code. It is recorded as a performance item with a
  named trigger (#120 or a measurement), not built.
- **Milestones re-cut.** The old M2 (reconcile two writers) is retracted: its
  premise is false in production. Only a stale comment survives, folded into M1.
  The old M3 (classify the preamble) becomes M2, now including DECSCUSR. The old
  M4 (ingest gate) becomes M3.
- **The earlier "gate on the parent advertising 2026" constraint is dropped**, with
  the reason in `## Spec`: unknown modes are ignored, and a DECRQM confirm would be
  an async-stale belief.

`## Problem` still stands as filed.

### 2026-09-17 — ingest is already honoured; the planned M3 is retracted

Correcting the 2026-09-16 Log entry *"correction: ingest is NOT honored either"*,
and the M3 this morning's re-cut carried over from it. Ingest-side sync landed in
#255 M2 (`d44ff360`, 2026-09-15), the day BEFORE that entry: `Endpoint` tracks the
child's 2026 hold (`syncState`), withholds publication, and recovers after a 150ms
`SyncTimeout`. The entry's greps for `2026h|2026l|?2026` cannot match
`ansi.DECMode(2026)`, and the one bare `grep -rn 2026` it cites would have shown
`endpoint.go`. The claim was wrong when written. The same entry's nesting worry
assumed brackets flow through pair; they do not, because the endpoint consumes
them. Delta: the planned M3 is dropped, and the issue is M1 (emit bracket) + M2
(classification).

### 2026-09-17 — estimate revised 4.41 → 2.45 before any code

The estimate-quality check (INFO) found about 1.35h of design booked for diagnosis
that mostly happened before the claim anchor `sdlc actual` measures from. Of the
two items that booked it, `issue-spec` and `scope-pivot`, the second
double-counted the first. It also found the smoke booked as a UX iteration, M2
priced as full design although it is mostly classification, and the M1 atlas item
undiscounted. All four were corrected before implementation started, so the
estimate prices only the measured window.

## Plan

Durable plan: `workshop/plans/000262-sync-output-emit-bracket-plan.md` (M1).

- [ ] M1 — Bracket every emitted frame in DECSET 2026 inside `Render` and
      `HistoryRender.Emit`, sharing one cursor epilogue. Close sync in
      `parentReleaseControls`. Test across alt transitions, history push, no-op
      frames and a cut in any write of the frame. Oracle suites green. Retire
      the stale reserved-row prose (`reserve.go:13-18`, atlas). Operator smoke
      under couch and `pair term`.
- [ ] M2 — Classify the remaining per-frame sequences (DECSCUSR first) against their
      primitives, using M1's smoke evidence. Delta only what they sustain; keeping a
      convergent re-assert is a valid outcome. Record the row-diff trigger.

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

### 2026-09-17 — operator discussion: why the terminal should do the "diff"

Operator question: does pair keep the child's delta format? No. Since #255,
child bytes (delta or full repaint) end at the endpoint's parser, and the
presenter writes from grid state. Today a 3-byte child delta becomes a
full-screen parent write. Passing the child's bytes through would undo #255's
composition boundary, so that is not an option.

Two ways to stop the visible churn were weighed:

1. **Row diff in `Emit`: fewer bytes.** The cell diff is cheap, and `Render`
   already does it for panels. The hard part is soft-wrap provenance. A row's
   incoming wrap flag is set only by autowrapping through the row above, zellij
   keeps it across `EL2`, and only `IL` yields a pristine row. So the repaint unit
   is a wrap CHAIN (a logical line), taken as the union of the chains in the
   previous and next frames, so both ends of a run are chain heads in both frames.
   A multi-row run gets pristine rows from a confined `IL` (DECSTBM around the
   run). A single-row run is a chain head in both frames and needs only `EL2`. Row
   0 keeps the full path's `ECH` treatment when it continues from history. Anything
   with a reset, alt transition or history push takes the full rebuild. The end
   state is checkable against the full rebuild with the xterm oracle. A cheaper
   fast path handles only rows that are unwrapped in both frames: `CUP` + `EL2` +
   repaint, falling back to the full rebuild for everything else.
2. **Synchronized output: the same bytes, never shown half-applied.** The terminal
   parses into its grid under 2026 and draws the end state at the end marker. That
   IS the diff, done where it is cheapest, and it is the mode's purpose.

Operator: *"it's easier for ghostty to do this 'diff', and the whole point of the
sync instruction."* Agreed, and chosen: 2026 is M1, and the row diff is recorded
here with its trigger (network transport, #120, or a measured byte cost) instead
of being built.

Open for the M1 smoke: whether the operator's sightings were under couch. The
whole-window mechanism is couch's, because couch's presenter owns the real
terminal. Under plain pair, zellij is outermost and redraws only changed rows,
and `pair term`'s full repaint is confined to its own pane.

### 2026-09-17 — M1 implementation: oracles, failure sweep, stale-prose sweep

**Tasks 1–3 landed** (`05c0c26e`, `9c453a83`, `9ba2b30d`). Both renderers bracket
each frame. `parentReleaseControls` closes sync right after its CAN/ST abort. The
cursor epilogue is one helper (`cursorEpilogue`, `render.go`).

**Oracles.** The xterm-headless suites pass with `PAIR_TERMINAL_ORACLE=1`. xterm
5.5.0 does not implement 2026, so this proves the *ignoring-terminal* path: the
bracket leaves the end state unchanged. The native zellij oracle
(`PAIR_TERMINAL_NATIVE=1`, zellij 0.45.1, sandbox off) also passes:
`TestHistoryWireNativeOracle`, `TestHistoryED2NativeBlankAndSpaceRows`,
`TestErasedBackgroundSoftGapNativeCopyOracle` and
`TestPresenterHistoryEvictionNativeAppendAndRebuild`. The whole
`cmd/internal/terminal` package is green with both oracles on; only the unrelated
`TestTerminalResourceProbe` skips. So bracketed wire leaves `pair term`'s real
parent in the same end state. Whether zellij *honours* 2026 from a pane is for
the smoke.

**Failure sweep.** `TestPresenterReleaseClosesSyncAfterAnyCutWrite` cuts writes
in three frame layouts, measured by a probe:

- single-write `[delta 58 | 181]`;
- alt-switch `[58 | 8 syncBegin | 8 ?1049h | 165]`;
- multi-chunk `[58 | 8 | 8 | 65527 65534 65531 65529 59312]`.

Every offset is cut for the small layouts. For the 321 KB one, the cuts are
boundaries ±2, the first and last 32 bytes, and a 4093 stride. Before the release
fix, all three failed at cut 8, the first offset after a complete `syncBegin`.
After it, all pass, in 1.3s.

**Stale-prose sweep (Task 4).** The class is *prose presenting the pre-#255
reserved-row machinery as live*. The sweep matched the claim's vocabulary, not
only symbols: `SafeToPaint`, `TakeRowDirty`, `ReserveAndPaint`, `paneWriter`,
`writeOwn`, `flushOwed`, `owe-and-flush`, `only here`, `one package only`,
`shared by two consumers`, `clear the reserved row`. Fixed:

- `hostty/reserve.go:13-18`;
- `couchtty/reserve.go:12-18`;
- `cmd/probes/couchnestedrows/main.go:18`;
- a status note on `ptychild.Screen`, which covers its console-facing method docs;
- `atlas/architecture.md`: six paragraphs describing the deleted console-write
  door (including a `stripmutation_test.go` that no longer exists) condensed into
  one pre-#255 note;
- `atlas/couch.md` teardown: *"clear the reserved row"* was false (release keeps
  the pixels); the sentence now names `parentReleaseControls`, which since this
  milestone really does close synchronized output.

The final sweep returns only correctly framed text: the historical note, and
`reserve.go`'s description of its own API.

**Filed pair#281** to dispose of the machinery itself: `ptychild.Screen` has no
production consumer, and `hostty.Reservation`'s painters serve only a probe.
Deleting code is separable from the flicker fix, so #262 corrected only the
prose.

**Atlas:** `atlas/terminal.md` records the emit bracket and its end-to-end pairing
with ingest-side sync.

**Full suite, 2026-09-17.** `make -k test`, with the five-variable retention
scrub and the sandbox off, had one failure: `test-changelog`, the known
pre-existing failure (`viewer: process target is outside selected owner
directory`; see memory, reproduced on `origin/main` during #256). Everything else
passed. Because a prerequisite failed, make skipped the `test` target's own
`go test ./...` recipe, so the Go suite was run directly:
`go test ./... -count=1` with the same scrub. It exited 0, with 71 packages ok.
`make build` rebuilt `bin/pair` and `bin/couch` from this branch for the smoke.

### 2026-09-17 — M1 operator smoke under couch: no flicker

Operator smoke, ~20:05, recorded from a brain advisor session.

**Build under test, verified before the smoke rather than assumed** (a couch
older than its own binary had already produced one wrong conclusion that day):
`bin/pair` built 19:37 and `bin/couch` 20:02, both after the last M1 code commit
`9ba2b30d` (19:29). couch pid 76239 started 20:02:28; the thread's `pair wrap`
pid 77655 started 20:02:47. Both renderers M1 touches were the new ones.

**Result: no flicker observed.** Operator's words: *"I don't see any flicker
anymore, including pressing key for a long time, while you are changing agent
pane display"*, with the right-hand pane running `top`.

That puts three writers on screen at once:

- sustained key-repeat in the draft;
- the agent pane repainting continuously (an agent streaming output);
- `top` in the right pane — a full-screen child under `pair term` on the
  **alternate screen**, refreshing at a steady low rate. That exercises the alt
  enter path M1 explicitly bracketed, and its periodic whole-screen refresh is
  close to the original complaint's regime: the flicker was worst when *little*
  was happening.

**What this does NOT settle — the zellij half of the smoke item.** The Done-when
asks for `pair term` under **plain zellij**, and whether zellij honours 2026
from a pane, recorded whichever way it goes. Under couch that question is
masked: `pair term` emits into its zellij pane, zellij composites, and couch's
own renderer then wraps the result in its *own* 2026 bracket before it reaches
the terminal. A zellij that silently dropped the pane's bracket would still look
flicker-free here, because the outer bracket absorbs it. So this smoke proves
the couch path end-to-end and says nothing about zellij on its own.

**Remaining for M1:** smoke `pair term` directly in a plain zellij session
(outside couch) under the same quiet regimes, and record the zellij answer.
Threads whose `pair wrap` predates the 19:37 build still carry the old renderer
until relaunched (Alt+n), so judge those only after a relaunch.

### 2026-09-17 — operator smoke: couch flicker gone

The operator killed the pre-M1 couch (pid 80116, started 17:41, before the 19:37
build) and relaunched `couch`, which rebuilt from this branch. After reloading the
brain thread: *"I think your fix is successful."* That covers the couch regime,
where the whole-window flicker lived. Still open for the M1 close: `pair term`
under plain zellij (a fresh Alt+Shift+d split, since running `pair term`
processes still run the old binary), and whether the caret blinks regularly,
which is M2's DECSCUSR input.
