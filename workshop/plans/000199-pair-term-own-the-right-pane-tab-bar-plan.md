# `pair term` Tab Strip Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `pair term` draws its own tab strip in a reserved row of its pane, replacing `zellij rename-pane` as the way tab state reaches the operator, and the right pane goes frameless.

**Architecture:** The row-reservation primitive moves from `couchtty` into `hostty` as a value that carries *which edge* it reserves, so couch (bottom of the host) and `pair term` (bottom of its pane) share one implementation of `\x1b[r`. `termcmd` becomes single-writer first — a prerequisite, not cleanup — then renders the strip from its existing tab model, with display-column spans so `#200` has nothing to retrofit.

**Tech Stack:** Go; `hostty` (host half), `ptychild` (child half, unchanged), `termcmd` (the new consumer), `couchtty` (repointed), zellij layout KDL.

---

## What the probes established, and what each one changed

Every scope decision below rests on a measurement or an operator decision, not
on the Spec's assumptions. Findings 1-4 are from 2026-09-06; 5 and 6 are from
the 2026-09-07 feasibility probe that reopened this issue. Four of the six
contradict the Spec.

**1. The pane frame is mouse-dead** (measured; `## Log`). A shell in the right
pane with `?1000`+`?1006` on reported clicks inside the text area and **nothing**
for the top or bottom border, with inside-clicks bracketing the frame clicks so
silence is meaningful. Consequence: the strip is the **only** surface that can
ever carry the interaction, so `#200` depends on this issue structurally, and
frameless costs nothing we could otherwise have had.

**2. The operator does not split the right pane.** Tabs are the multi-window
mechanism. The divider job that caused the 2026-07-27 frameless revert is
retired.

**3. Scroll position is NOT ours** (contradicts the Spec). The Spec argued
`pair term` "already holds its own scroll position without asking zellij
anything". `run.go:453-463` forwards every wheel tick to
`RunZellijAction("scroll-up"/"scroll-down")`, and `ptychild` exposes a **replay
ring** (`Snapshot`/`Replay`/`ReplayThrough`) with no viewport or offset
anywhere. The operator dropped the readout rather than build one, which is what
moved frameless into this issue — see `## Revisions`.

**4. `termcmd` writes to the host from two goroutines** (not in the Spec).
`copyActiveOutput` (`run.go:702`) and `redrawTab` (`run.go:1023`, from three
tab-switch sites). `atlas/couch.md` records why a reserved row cannot survive
that: *"a pty read boundary falls wherever the kernel puts it, so a paint
written between two chunks can land inside one of the child's escape
sequences."* M2 exists because of this.

**5. zellij HONORS a pane process's DECSTBM** (measured 2026-09-07; the probe
that reopened this issue). This is the load-bearing fact and it had never been
tested: couch's reserved row works on the HOST terminal, where couch writes
straight to the tty, but the right pane's writes pass through zellij's emulator
instead. If zellij ignored the scroll region, the strip would have needed full
compositing -- a different and far larger project -- and this plan's M1/M3 would
have been built on sand.

Method and result: a throwaway zellij session (`--layout`, repo config, so no
startup-tips pane steals focus) ran a pane process that set
`ESC[1;<rows-1>r`, painted the bottom row, then printed 200 lines to force
scrolling. Read from the pty -- what zellij actually rendered to the host:

```
marker last painted at row 23
last scroll line at row 21
earliest scroll line still on screen: false
RESULT: DECSTBM HONORED — 200 lines scrolled in rows 1..21
        while the reserved row 23 held its paint.
```

So `couchtty.Reserve`/`PaintRow` transfer to the pane unchanged, which is
exactly what M1 assumes. Probe kept at `scratchpad/199-probe/`.

Two cautions carried forward from running it. Three earlier runs printed
"NOT HONORED" and every one was the probe's own session failing to start
(zellij refuses to nest, and `--session` with `--layout` means *attach*, not
create) -- the probe now refuses to emit a verdict when its session is absent,
per `#208`'s rule that a reading whose precondition failed is `n/a` and not a
result. And `dump-screen` proved useless here: it returns pane content without
saying where zellij placed it, so the verdict reads cursor positions out of the
pty frame instead.

**6. The tab MODEL already exists** (verified 2026-09-07; corrects the operator's
and the Spec's framing). `Alt+t` is already not a zellij tab: `config.kdl:114`
unbinds it and forwards raw `ESC t` to the focused pane, and
`handleTerminalChord` (`run.go:491`) catches `ChordAltT` into `mux.newTab()`.
`terminalMux` carries `tabs []*terminalTab`, `active`, `previousTab`/`nextTab`
and `closeActive`. Multiple terminals switched on demand is **built and
shipping**; the only missing piece is the display, whose sole surface today is
`setPaneTitle` -> `rename-pane` packing the whole tab set into one string. This
issue is therefore a rendering change, not an architecture change, which is why
its milestones touch no tab-lifecycle code.

## Non-goals

- **Clickable tabs.** `#200`, and deliberately so: its substance is
  consolidating a mouse-mode belief that has been wrong four consecutive times
  (BR-16/22/26/33), and `termcmd` already carries a second implementation of it
  (`appMouseMode()`, `run.go:815`). What this issue *does* do is emit
  display-column spans so `#200` retrofits nothing.
- **A scrollback viewport in `pair term`.** See finding 3. It is a feature, not
  a readout, and nothing here needs it.
- **Splits, or any pane management.** `pair term` owns tabs *within* a pane;
  zellij keeps panes. A split still yields two panes each running `pair term`,
  each drawing its own strip.
- **Changing the agent pane's frame.** `pane_frames true` stays global; only the
  right pane opts out. Whether the agent pane still needs frames is a separate
  question this issue deliberately does not answer.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Edge` | `cmd/internal/hostty/reserve.go` | new |
| `Reservation` | `cmd/internal/hostty/reserve.go` | new |
| `TabChip` | `cmd/internal/termcmd/strip.go` | new |
| `StripModel` | `cmd/internal/termcmd/strip.go` | new |
| `RenderedStrip` | `cmd/internal/termcmd/strip.go` | new |
| `RenderStrip` | `cmd/internal/termcmd/strip.go` | new |
| `couchtty.ChildRows` / `Reserve` / `Release` / `PaintRow` | `cmd/internal/couchtty/reserve.go` | deleted |
| `couchtty.RenderStatusRow` (+ `StatusModel`, `ChipSpan`) | `cmd/internal/couchtty/reserve.go` | unchanged |

- **Edge / Reservation** — which edge of a terminal is reserved, and how tall
  the terminal is. `Reservation{Rows, Edge}` answers `ChildRows()`,
  `Reserve()`, `Release()`, `Paint(text)`.
  - **DRY rationale:** today three `couchtty` functions each hardcode the bottom
    row. `hostty` already owns `SetRegion`/`MoveTo`/`ResetRegion` and the atlas
    states the split this follows: *"`\x1b[r` lives here and only here; it was
    about to exist in two packages."* Same argument, next mechanism.
  - **Why a value and not three functions with an edge argument:** the edge is
    the same for every call in a consumer's lifetime, and passing it three times
    invites the two calls that disagree. `ChildRows` and `Paint` disagreeing is
    exactly the off-by-one that makes a child overwrite the row.
  - **The edge is NOT symmetric, and the type must not pretend otherwise.** A
    bottom reservation works because the child is handed `N-1` rows and
    addresses `1..N-1`, which cannot reach row `N`. A **top** reservation puts
    the child's row 1 *on* the strip, so every full-screen app draws over it —
    correct only under origin mode (`\x1b[?6h`), which nothing tracks
    (`Screen` handles `1049/1047/47`, `1000/1002/1003`, `1006`). `EdgeTop` is
    therefore **not implemented** in M1; the type carries the axis so the
    asymmetry is visible, and the constructor refuses it. See ARCH-ORDER.
  - **Future extensions:** `EdgeTop` when someone does the DECOM arbitration;
    multi-row reservations if a strip ever needs two lines.

- **StripModel → RenderedStrip** — the tab set and which is active, rendered to
  a row plus the display-column span of each tab.
  - **Relationships:** 1 `StripModel` per `terminalMux`; N `TabChip` per model,
    one per `terminalTab`.
  - **DRY rationale:** mirrors `couchtty.RenderStatusRow`'s shape
    (`RenderedStatusRow{Body, Chips}` + `ColumnToActor`) rather than inventing a
    second one. The two renderers stay separate — what a row *says* is
    per-consumer policy — but they agree on the contract a clickable row needs.
  - **Spans are display columns, not rune counts** (BR-32, `#172`). A rune count
    puts every span after a wide character one column left of what was drawn,
    and an all-ASCII suite stays green while it does. `textwidth.Width` is the
    existing answer; the test pins it with a wide-character tab name.
  - **ARCH-PURE:** takes a model and a width, returns a row and spans. No IO, no
    clock, no zellij. Unit-tested with hand-built models.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| single host writer | `cmd/internal/termcmd/run.go` | modified | the operator's tty |
| paint gate | `cmd/internal/termcmd/run.go` | new | `ptychild.Screen.MidSequence` |
| strip repaint trigger | `cmd/internal/termcmd/run.go` | new | `ptychild.Child.TakeRowDirty` |
| degraded `rename-pane` | `cmd/internal/termcmd/run.go` | modified | `zellij action` |
| right pane chrome | `.../zellij/layouts/main-3.kdl` | modified | zellij layout |

- **single host writer** — one goroutine owns `m.stdout`; `redrawTab` becomes an
  event on that loop rather than a direct write from the tab-switch path.
  - **Why this is a prerequisite and not a follow-up:** with two writers, a
    strip paint can land between two chunks of the child's output — inside an
    escape sequence. couch names this as learned the expensive way and answered
    it by making `Console.Run` the only writer. Building the strip first would
    reproduce a bug this repo has already paid for once.

- **paint gate** — a `ptychild.Screen` fed **child bytes only**, consulted via
  `MidSequence()` before any strip paint; a deferred paint is owed and flushed
  on the next chunk that ends on a boundary.
  - **Injected:** the gate is a value the writer loop owns; tests drive it by
    feeding chunk boundaries, no pty required.
  - **Both halves of couch's lesson carry over verbatim** and must not be
    re-derived: ask about *the stream the writer writes*, not the child's own
    scanner (which is fed ahead of the consumer), and feed it **child bytes
    only** — feeding our own escapes in let it frame our bytes with the child's
    partial and report "safe" precisely when it was not.
  - **ARCH-DRY note, deliberately not acted on yet:** couch's defer/owe policy
    (`console.go:1098,1122,1172`) is the same shape, but it is entangled with
    notification batching `termcmd` has no analogue for. M2 implements the rule
    against the already-shared `MidSequence()`; if the two policies turn out
    identical once both exist, lifting is a follow-up with two real callers
    rather than a guess with one.

- **degraded `rename-pane`** — plan item 1, decided: **it survives**, writing a
  short title (active tab name) rather than the packed multi-tab string.
  - The zellij pane title is still the only label visible when the pane is not
    focused, and `#118`'s tab-strip titles and `#123`'s registry read it
    (`run.go:229`). Dropping it silently breaks consumers this issue never
    looked at. What goes away is `paneTitleLocked` packing the whole tab set
    into one rename argument — the strip carries that now.

- **right pane chrome** — `borderless=true` on the terminal pane.
  - **Nine sites, not one** (`main-3.kdl`: `:83, :125, :138, :139, :152, :165,
    :166, :191, :192`) — the default layout plus every swap rung. The draft pane
    is the precedent: it carries `borderless=true` in every rung for the same
    reason. Missing one leaves a rung that reframes the pane on `Alt+Up`.
    `main-2.kdl` has no terminal pane and is untouched.

## ARCH-CONSTRAINTS — operating envelope

- **Interaction path: keystroke.** This is the one that matters — the strip
  repaints on the same loop that echoes the operator's typing.
- **Budget: no paint on the common path.** A repaint happens on tab change,
  resize, and `TakeRowDirty`; **not** per output chunk. Basis: couch's row works
  this way today and typing in a couch-hosted pane is not laggy. Exceeded → the
  strip is repainting per chunk and the operator feels it as input lag, which is
  the failure the operator already reported once for other reasons (`#201`).
- **Budget: no subprocess per tab change.** Replacing `rename-pane`-per-title
  with a strip paint should *reduce* process spawns; the degraded title keeps
  one spawn on tab switch only, not on every render.
- **Scale:** tab counts are single digit. Linear rendering is correct.
- **Concurrency:** exactly one goroutine writes the host after M2 — that is the
  envelope, and `TestOnlyOneGoroutineWritesTheHost` is how it is enforced rather
  than asserted.
- **Latency/overload:** N/A beyond the above; no network, no fan-out.

## ARCH-ORDER — states, events, and the ones the caller cannot block

The reserved row is state carried across events from a child this process does
not control. Three transitions, and the third is the one that bites.

| event | state | -> effect |
|---|---|---|
| tab created / closed / switched | strip model changes | repaint (gated) |
| host resize | rows change | re-`Reserve`, re-`Paint` |
| child clears display / resets margins / RIS / alt-screen | row wiped, region may be gone | `TakeRowDirty` → re-`Reserve` **and** repaint |
| child writes mid-sequence | paint would corrupt | defer, owe, flush on the next boundary |
| child exits | tab removed | repaint; last tab exit tears down the reservation |

Events the caller cannot block, named rather than swept:

- **A child that sets its own DECSTBM.** `nvim` does. It clobbers our region,
  and only `TakeRowDirty` (which counts margin reset) brings it back. The
  repaint must therefore **re-`Reserve`, not just re-`Paint`** — repainting into
  a region the child replaced puts the strip inside the child's scroll area,
  where the next scroll eats it. This is the single most likely thing to get
  wrong here.
- **A pty read boundary splitting an escape sequence.** The paint gate, above.
- **Process death between `Reserve` and `Release`.** The operator's shell is
  left scrolling in a box. `Release` on every exit path, including panic —
  couch's `Release` exists for exactly this.
- **Resize racing a paint.** Both are events on the single writer loop after M2,
  so they serialize by construction rather than by luck.
- **Concurrent extent:** no goroutine outlives `Run`; the writer loop drains and
  exits on `m.done`, which it already does.

Nondeterminism enters through pty chunk boundaries. It is reproducible because
the gate is fed bytes by the test, not by a kernel: a test can split an escape
sequence at any index and assert the paint deferred.

## ARCH-SECURE

`N/A for secrets` — this issue touches no credential. It does read one untrusted
input: **the child's byte stream**, which `ptychild.Screen` already parses and
which this issue does not extend. Tab *names* reach the row and come from the
operator (`rename`), so `RenderStrip` sanitizes and truncates exactly as
`couchtty.RenderStatusRow` does (`sanitize`, `truncate`) rather than growing a
second policy — a name containing `\x1b` must not become an escape in our row.

## ARCH-MOCK

No new external dependency. zellij is already behind `Runtime.RunZellijAction`
with a fake that records calls; the tty is already behind `hostty.FakeHost`,
which records writes. The strip's assertions run against `FakeHost.Written()`,
so production and test share the boundary. The one *new* fake surface is the
paint gate, which takes bytes directly and needs no double at all.

## ARCH-PURPOSE

The purpose is that tab state reaches the operator without the zellij frame.
The deliverable therefore includes the frame coming off (M4) and the
`rename-pane` consumers still working — not just a row being drawn. Two
enumerations are written out rather than left as "sweep": the **nine**
`name="terminal"` sites in `main-3.kdl`, and the `rename-pane` consumers at
`run.go:229`, `#118`, `#123`.

---

## M1 — lift the reservation into `hostty`

**Files:**
- Create: `cmd/internal/hostty/reserve.go`, `cmd/internal/hostty/reserve_test.go`
- Modify: `cmd/internal/couchtty/reserve.go` (delete the four functions), `cmd/internal/couchtty/console.go` (repoint)

- [ ] **M1.1: Write the failing tests** — including the asymmetry:

```go
func TestBottomReservationKeepsTheChildOffTheRow(t *testing.T) {
	r := hostty.Reservation{Rows: 24, Edge: hostty.EdgeBottom}
	if got := r.ChildRows(); got != 23 {
		t.Fatalf("ChildRows = %d; want 23", got)
	}
	// The region ends one row above the reserved row, so a child scrolling at
	// the bottom of its own screen cannot walk onto it.
	if got := r.Reserve(); got != "\x1b[1;23r" {
		t.Fatalf("Reserve = %q; want the region to stop at 23", got)
	}
}

// A terminal too short to reserve from gives the child everything and draws
// nothing -- a zero-row pty is not a thing.
func TestDegenerateHeightsNeverProduceAZeroRowChild(t *testing.T) {
	for _, rows := range []uint16{0, 1} {
		r := hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom}
		if r.ChildRows() == 0 {
			t.Fatalf("rows=%d produced a zero-row child", rows)
		}
		if r.Reserve() != "" {
			t.Fatalf("rows=%d reserved from a terminal with no room", rows)
		}
	}
}

// EdgeTop is representable but refused, because a top reservation is NOT the
// mirror of a bottom one: the child addresses 1..N-1, so its row 1 IS the
// strip. Correct only under origin mode, which nothing tracks.
func TestTopEdgeIsRefusedUntilOriginModeExists(t *testing.T) {
	if _, err := hostty.NewReservation(24, hostty.EdgeTop); err == nil {
		t.Fatal("EdgeTop accepted; a top strip needs DECOM arbitration first")
	}
}

func TestPaintBracketsWithCursorSaveRestore(t *testing.T) {
	// without save/restore the caret jumps to the row on every repaint
}
```

- [ ] **M1.2: Run to verify they fail.** `go test ./cmd/internal/hostty/ -run Reserv -count=1`
- [ ] **M1.3: Implement** `Edge`, `Reservation`, `NewReservation`, and the four methods, composing from `hostty`'s existing `SetRegion`/`MoveTo`/`ClearLine`/`Save`/`RestoreCursor` constants. Do not spell an escape twice.
- [ ] **M1.4: Repoint couch** — delete `couchtty.ChildRows/Reserve/Release/PaintRow`, construct `Reservation{Edge: EdgeBottom}` in `console.go`. `RenderStatusRow` and friends stay put: what the row *says* is policy.
- [ ] **M1.5: Prove it was a MOVE, not a rewrite.** `go test ./cmd/internal/couchtty/ -count=1` green with **no couch test edited** — `git diff --stat cmd/internal/couchtty/*_test.go` must be empty. This is the Done-when's "a regression there means it was a rewrite", made checkable.
- [ ] **M1.6:** `grep -rn "SetRegion\|\\\\x1b\\[.*r\"" cmd/ --include=*.go | grep -v hostty/` returns nothing — one implementation, per the atlas.
- [ ] **M1.7: Commit**, then `sdlc milestone-close --issue 199 --milestone M1`.

## M2 — one writer, one gate

**Files:**
- Modify: `cmd/internal/termcmd/run.go`
- Test: `cmd/internal/termcmd/writer_test.go` (new)

- [ ] **M2.1: Write the failing tests**

```go
// The envelope, enforced rather than asserted: after this milestone exactly one
// goroutine may write the host. A second writer is how a paint lands inside a
// child's escape sequence.
func TestOnlyOneGoroutineWritesTheHost(t *testing.T) {
	// stdout wrapped in a writer that records the goroutine id of each Write;
	// drive a tab switch (redrawTab's old path) concurrently with output
	// chunks; assert exactly one distinct writer.
}

// couch's lesson, restated at the new consumer: a write issued while the
// child's stream is mid-sequence must be deferred, and OWED -- dropping it
// leaves a stale row that nothing will repaint.
func TestPaintDefersMidSequenceAndIsOwed(t *testing.T) {
	// feed a chunk ending inside "\x1b[3" ; request a paint; assert nothing
	// written; feed the completing "m"; assert the paint lands exactly once.
}

// And the half couch got wrong first: the gate is fed CHILD bytes only. Feeding
// our own escapes in lets it frame our bytes with the child's partial and
// report safe precisely when it is not.
func TestGateIsNotFedOurOwnWrites(t *testing.T) {
	// paint twice while the child stream is mid-sequence; assert the gate's
	// MidSequence state is unchanged by our writes
}
```

- [ ] **M2.2: Run to verify they fail.**
- [ ] **M2.3: Implement.** Route `redrawTab`'s replay through the writer loop as an event. Add the gate: a `ptychild.Screen` fed child chunks before they are written, consulted before any console-originated write, with a deferred-paint slot flushed on the next boundary-ending chunk.
- [ ] **M2.4:** `go test ./cmd/internal/termcmd/ -count=1 -race` — the race detector is the point, not decoration.
- [ ] **M2.5: Manual** — switch tabs rapidly under load (`yes` in one tab) and confirm no corruption. Record what was observed in `## Log`, not "it worked".
- [ ] **M2.6: Commit**, `sdlc milestone-close --issue 199 --milestone M2`.

## M3 — the strip

**Files:**
- Create: `cmd/internal/termcmd/strip.go`, `strip_test.go`
- Modify: `cmd/internal/termcmd/run.go`

- [ ] **M3.1: Write the failing tests**

```go
func TestActiveTabIsDistinguishable(t *testing.T) { /* not by colour alone */ }

// BR-32, carried forward from #172: spans are DISPLAY COLUMNS. A rune count
// puts every span after a wide character one column left of what was drawn --
// and an all-ASCII suite stays green while it does, which is why this fixture
// is not ASCII.
func TestSpansAreDisplayColumnsNotRuneCounts(t *testing.T) {
	m := StripModel{Tabs: []TabChip{{Name: "日本語"}, {Name: "build"}}, Active: 1}
	r := RenderStrip(40, m)
	if got := r.Spans[1].Start; got != textwidth.Width(r.Body[:idxOfBuild]) {
		t.Fatalf("span start %d is a rune count, not a column", got)
	}
}

// Tab names come from the operator; an escape in one must not become an escape
// in our row. Same policy as couchtty, not a second one.
func TestATabNameCannotInjectEscapes(t *testing.T) {
	m := StripModel{Tabs: []TabChip{{Name: "a\x1b[31mred"}}}
	if strings.Contains(RenderStrip(40, m).Body, "\x1b") {
		t.Fatal("a tab name reached the row as an escape sequence")
	}
}

func TestNarrowPaneTruncatesWithoutLosingTheActiveTab(t *testing.T) {}
```

- [ ] **M3.2: Run to verify they fail.**
- [ ] **M3.3: Implement `RenderStrip`** — pure, returning `RenderedStrip{Body, Spans}`, sanitizing and truncating via the same helpers `couchtty` uses.
- [ ] **M3.4: Wire it.** `Reservation{Edge: EdgeBottom}` sized from the pane; child pty gets `ChildRows()`; repaint on tab change, resize, and `TakeRowDirty`.
- [ ] **M3.5: The re-`Reserve` rule** (ARCH-ORDER's most-likely-wrong): on `TakeRowDirty`, re-`Reserve` *before* repainting. Test: simulate a child emitting `\x1b[r` (margin reset), assert the next repaint re-emits the region and not only the row.
- [ ] **M3.6: Degrade `rename-pane`** to the active tab name; assert `RunZellijAction` still receives a rename on tab switch (the `#118`/`#123` consumers) and that it is no longer the packed multi-tab string.
- [ ] **M3.7:** `go test ./cmd/... -count=1`, then **manual in a real layout3 pane**: run `nvim`, confirm the strip survives its startup clear and its own margin changes; quit; confirm the shell is not left scrolling in a box.
- [ ] **M3.8: Commit**, `sdlc milestone-close --issue 199 --milestone M3`.

## M4 — take the frame off

**Files:**
- Modify: `cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main-3.kdl`
- Test: `cmd/internal/runtimebundle/` layout assertion

- [ ] **M4.1: Write the failing test.** Every `name="terminal"` pane in
      `main-3.kdl` carries `borderless=true` — the enumeration, so a rung added
      later cannot quietly reframe the pane:

```go
func TestEveryTerminalPaneRungIsBorderless(t *testing.T) {
	// parse main-3.kdl; for each pane named "terminal", assert borderless=true.
	// Nine sites today (:83,:125,:138,:139,:152,:165,:166,:191,:192).
}
```

- [ ] **M4.2:** Add `borderless=true` at all nine sites.
- [ ] **M4.3: Manual, and this is the acceptance** — a real layout3 workbench:
      the pane has no frame; the strip is legible and identifies the pane; the
      layout rungs (`Alt+Up`/`Alt+Down`) do not reframe it; `nvim` in the pane
      still behaves. Record what was seen.
- [ ] **M4.4:** Update `atlas/architecture.md` — the row primitive is shared
      host-half structure (alongside the existing `\x1b[r` note), and
      `config.kdl`'s comment that frames are global *for the scroll indicator*
      is now wrong for the right pane; correct it there too.
- [ ] **M4.5: Commit**, then `sdlc close --issue 199 --verified '...'`.

## Rollback

M4 is the only step with a chrome consequence, and it is one attribute at nine
sites — reverting is a single commit and costs nothing. M2 is the risky one: it
changes how output reaches the terminal. If corruption appears after M2 and the
cause is not obvious, revert M2 and stop, because M3 depends on it and building
the strip over a suspected-broken writer would confuse both.


## Revisions

### 2026-09-07 — reopened from `punt`; feasibility established

The issue was punted 2026-09-06 to prioritise the performance work (`#208`),
with the cost/benefit recorded in the issue Log. Reopened at operator request
after a probe answered the one question that made the whole design speculative.

**Delta to the plan:**

1. **Findings 5 and 6 added** to `## What the probes established`. Finding 5
   (zellij honors a pane's DECSTBM, measured) is the fact M1 and M3 rest on and
   was previously untested. Finding 6 (the tab model already exists) narrows
   what this issue is: a rendering change over a shipped multiplexer, not new
   tab machinery.
2. **The strip's owner was re-confirmed as `pair term`, not couch.** The
   operator initially framed this as couch owning the tab bar. Two reasons it
   cannot: couch's child is the whole zellij session, so it has no way to render
   into a pane zellij owns; and `couch must not degrade pair` -- a tab bar
   living in couch would leave standalone pair without one. Decided in
   discussion 2026-09-07; the plan's existing design is unchanged by it.
3. **No milestone content changed.** M1-M4 stand as written and as previously
   plan-gated.
