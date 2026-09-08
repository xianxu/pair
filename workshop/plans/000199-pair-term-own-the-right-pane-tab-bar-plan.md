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

**7. `Child.TakeRowDirty()` is the WRONG repaint trigger for `termcmd`, and would
have read false forever** (PQ-3, verified 2026-09-07). `Child.readLoop` already
DRAINS it into `batch.RowDirty` on every batch whenever a `Sink` is set
(`ptychild/child.go:174-177`), and `Screen.TakeRowDirty` clears the flag as it
reads (`screen.go:138-142`). `termcmd` sets a Sink (`run.go:660`), so a separate
`child.TakeRowDirty()` call in the strip's repaint path is consuming an
already-drained flag: always false, and a strip that never repaints after a
full-screen child's startup clear -- the one failure mode M3 exists to survive.
**M3 reads `batch.RowDirty` inside the Sink callback**, which is where the
signal actually arrives.

**8. Every writer to the pane's tty, DERIVED** (PQ-4, PQ-10). The Spec named
two; there are four in-process across two descriptors, plus a fifth that is a
**different process entirely**. Derivation:

```sh
grep -n "stdout\|stderr" cmd/internal/termcmd/run.go | grep -i "write\|fprint"
grep -n "RunZellijAction" cmd/internal/termcmd/run.go   # subprocess writers
```

| writer | fd | goroutine |
|---|---|---|
| `copyActiveOutput` (`run.go:702`) | stdout | pty pump |
| `redrawTab` (`run.go:1022-1023`) | stdout | Run, from three tab-switch sites |
| `restoreTerminal` (`run.go:1036`) | stdout | Run, via `defer` |
| `fmt.Fprintf(stderr, "term: …")` (`run.go:53,62,66,232,242`) | stderr | Run |
| **`RunZellijAction`** (`run.go:1086`) | **`os.Stdout`, from a SUBPROCESS** | `zellij action` |

The fifth is the one that matters and the one a goroutine-id test can never see:
`RunZellijAction` hands `os.Stdout` to the child process (`runZellij(cmdArgs,
os.Stdout)`), so **every wheel tick** (`scroll-up`/`scroll-down`, `run.go:457,463`)
and **every tab rename** (`run.go:961,963`) spawns a process writing straight
into the pane, outside any in-process envelope. `RunZellijActionQuiet` already
exists and passes `io.Discard` (`run.go:1091`).

**Six writers, and `Quiet` closes only one of the two subprocess ones.**
`RunZellijActionQuiet` swaps stdout for `io.Discard`, but `runZellij` hardwires
`cmd.Stderr = os.Stderr` for BOTH methods (`run.go:1100-1105`) — so a *failing*
`zellij action scroll-up` writes into the pane on every wheel tick, outside the
writer loop and outside the paint gate, and after M4 there is no frame to
absorb it. Routing stdout alone would have left the noisiest failure path open,
which is exactly this issue's recurring defect: a correction applied at the site
the finding named rather than to the class.

**M2's envelope covers all six.** The in-process four are serialised through one
writer. Both subprocess descriptors are captured — `runZellij` takes a
`stderr io.Writer` alongside its `stdout`, and the captured text is logged,
never written to the pane's fd. **Named exceptions: none.**

**9. The `rename-pane` consumer set, DERIVED** (PQ-1, PQ-10). The Spec asserted
it from memory; my first answer grepped one file and still missed a consumer.
The set is derived, and **the derivation is recorded so it can be re-run** rather
than re-remembered — that is the rule PQ-10 names, and the reason this entry
carries a command instead of a list:

```sh
grep -rn "\.Title" cmd --include="*.go" | grep -v _test.go   # every read of a pane title
```

Three matcher forms in two files, and they do NOT agree:

| site | patterns | command fallback |
|---|---|---|
| `launcher/layoutflow.go:62` | `Title == "terminal"`, `HasPrefix(Title, "[terminal")` | `Contains(command, "pair term")` |
| `workbenchshortcut/shortcut.go:189` | `title == "terminal"`, `HasPrefix(title, "terminal ")` — **lowercased** | `Contains(cmd, "pair term")` |
| `launcher/layoutflow.go:56,59` | `draft`, `terminal-filler` | nvim / `tail -f` |

The disagreement is the finding. `paneTitleLocked` emits `[tab1] tab2`, which
matches layoutflow's `"[terminal"` arm only when the first tab happens to be
named `terminal…`, and matches shortcut.go's `"terminal "` arm **never** — it
starts with `[`. So today `RoleForPane` classifies the right pane by its
COMMAND, not its title, and the title arm is already dead for the packed format.

That is what makes M3's degraded title safe, and it is now a derivation rather
than a hope: **every** arm has a `pair term` command fallback, so no consumer
depends on the title alone. M3.6 asserts exactly this — `RoleForPane` and
`ClassifyLiveLayout` fed the degraded title, including the
`TerminalCommand == ""` case `zellijpane.paneFrom` admits
(`zellijpane.go:79-84`), where the fallback is unavailable and the title is all
there is.

**4. `termcmd` writes to the host from two goroutines** (superseded by finding 8, which derives FIVE writers across two descriptors plus a subprocess; kept because it is the fact the Spec was missing).
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
exactly what M1 assumes. Probe: `probes/zellijscrollregion`, run by `make
test-smoke`.

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

### The rule behind findings 8 and 9 (PQ-10, family `consumer-set-not-derived`)

Three findings in this issue have been the same mistake: **a consumer set stated
from memory instead of derived from the code.** The Spec did it for
`rename-pane`, the plan did it for the writers, and my first correction did it
again — grepping one file and calling the result exhaustive.

**The rule:** a plan that names a consumer set records the COMMAND that produces
it, not the answer. A list is a snapshot that rots silently as consumers are
added; a derivation can be re-run by the next reader and by the review. Where a
milestone asserts on a consumer set, its test enumerates by the same derivation
rather than by a hand-copied list.

Applied here: findings 8 and 9 each carry their `grep`, and both turned up a
member the hand-written version had missed — a subprocess writing to the pane's
stdout, and a third title matcher that disagrees with the other two.

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
| `rowtext.Sanitize` / `rowtext.Fit` | `cmd/internal/rowtext/rowtext.go` | planned — M3 |
| `Edge` | `cmd/internal/hostty/reserve.go` | new |
| `Reservation` | `cmd/internal/hostty/reserve.go` | new |
| `TabChip` | `cmd/internal/termcmd/strip.go` | planned — M3 |
| `StripModel` | `cmd/internal/termcmd/strip.go` | planned — M3 |
| `RenderedStrip` | `cmd/internal/termcmd/strip.go` | planned — M3 |
| `RenderStrip` | `cmd/internal/termcmd/strip.go` | planned — M3 |
| `couchtty.ChildRows` / `Reserve` / `Release` / `PaintRow` | `cmd/internal/couchtty/reserve.go` | deleted |
| `couchtty.RenderStatusRow` (+ `StatusModel`, `ChipSpan`) | `cmd/internal/couchtty/reserve.go` | unchanged |

- **`rowtext.Sanitize` / `rowtext.Fit`** (`cmd/internal/rowtext/rowtext.go`, new
  — PQ-7) — make untrusted text safe and fitted for a one-row strip.
  - **Why a package and not a copy:** `couchtty`'s `sanitize` and `truncate`
    (`reserve.go:148,161`) are exactly this, and they are **unexported in
    another package** — `RenderStrip` cannot call them, so the plan's "reuse
    them" was not implementable. Both consumers put agent-published text on a
    reserved row: a label containing `\x1b[2J` clears the operator's screen from
    the strip, and a double-width rune must not wrap onto the child's area.
  - **Relationships:** N:1 — every reserved-row renderer depends on it;
    `couchtty.RenderStatusRow` and `termcmd.RenderStrip` are the two today.
  - **DRY rationale:** the alternative is a second copy of a security-relevant
    strip, in a package whose tests do not cover it.
  - **Future extensions:** a right-truncating variant if a strip ever needs the
    tail rather than the head.

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
| strip repaint trigger | `cmd/internal/termcmd/run.go` | new | `ptychild.OutputBatch.RowDirty` (read in the Sink — see finding 7; `Child.TakeRowDirty` is already drained there) |
| degraded `rename-pane` | `cmd/internal/termcmd/run.go` | modified | `zellij action` |
| right pane chrome | `.../zellij/layouts/main-3.kdl` | modified | zellij layout |

- **single host writer** — one goroutine owns `m.stdout`; `redrawTab` becomes an
  event on that loop rather than a direct write from the tab-switch path.
  - **Why this is a prerequisite and not a follow-up:** with five writers (finding 8), a
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
    (`layoutflow.go:56-62` and `workbenchshortcut/shortcut.go:189` — the derived
    set, finding 9). Dropping it silently breaks consumers this issue never
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
  resize, and `batch.RowDirty`; **not** per output chunk. Basis: couch's row works
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
| child clears display / resets margins / RIS / alt-screen | row wiped, region may be gone | `batch.RowDirty` → re-`Reserve` **and** repaint |
| child writes mid-sequence | paint would corrupt | defer, owe, flush on the next boundary |
| child exits | tab removed | repaint; last tab exit tears down the reservation |

Events the caller cannot block, named rather than swept:

- **A child that sets its own DECSTBM.** `nvim` does. It clobbers our region,
  and only `batch.RowDirty` (which counts margin reset) brings it back. The
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
operator (`rename`), so `RenderStrip` sanitizes and truncates via the shared
`rowtext.Sanitize`/`rowtext.Fit` (PQ-7: `couchtty`'s own `sanitize`/`truncate`
are unexported and unreachable from `termcmd`) rather than growing a
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
`name="terminal"` sites in `main-3.kdl`, and the `rename-pane` consumers derived
in finding 9 (`layoutflow.go:56,59,62` + `workbenchshortcut/shortcut.go:189`).

---

## M1 — lift the reservation into `hostty`

**Files:**
- Create: `cmd/internal/hostty/reserve.go`, `cmd/internal/hostty/reserve_test.go`
- Modify: `cmd/internal/couchtty/reserve.go` (delete the four functions), `cmd/internal/couchtty/console.go` (repoint)

- [x] **M1.1: Write the failing tests** — including the asymmetry:

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

- [x] **M1.2: Run to verify they fail.** `go test ./cmd/internal/hostty/ -run Reserv -count=1`
- [x] **M1.3: Implement** `Edge`, `Reservation`, `NewReservation`, and the four methods, composing from `hostty`'s existing `SetRegion`/`MoveTo`/`ClearLine`/`Save`/`RestoreCursor` constants. Do not spell an escape twice.
- [x] **M1.4: Repoint couch** — delete `couchtty.ChildRows/Reserve/Release/PaintRow`, construct `Reservation{Edge: EdgeBottom}` in `console.go`. `RenderStatusRow` and friends stay put: what the row *says* is policy.
- [x] **M1.5: Prove it was a MOVE, not a rewrite.** `go test ./cmd/internal/couchtty/ -count=1` green with **no couch test edited** — `git diff --stat cmd/internal/couchtty/*_test.go` must be empty. This is the Done-when's "a regression there means it was a rewrite", made checkable.
- [x] **M1.6:** `grep -rn "SetRegion\|\\\\x1b\\[.*r\"" cmd --include=*.go | grep -v _test.go | grep -v hostty/` returns nothing — one implementation, per the atlas. (The `_test.go` filter is load-bearing and was missing when this was first ticked: without it the command returns 35 lines, every one a test fixture. The invariant held; the stated command did not check it.)
- [x] **M1.7: Commit**, then `sdlc milestone-close --issue 199 --milestone M1`.

## M2 — one writer, one gate

**Files:**
- Modify: `cmd/internal/termcmd/run.go`
- Test: `cmd/internal/termcmd/writer_test.go` (new)

**Couch's gate has THREE rules, not two** (PQ-5). M2 restates the first two --
one writer, and defer-plus-owe while the child's stream is mid-sequence. The
third is the takeover reset (`couchtty/console.go:992-995`): when the screen is
taken over wholesale, `hostScan` is reset to a zero `ptychild.Screen{}` and
`paintPending` cleared, because whatever partial sequence the old child left is
no longer on screen to be corrupted -- without it the gate stays stuck owing a
paint against a stream that no longer exists. `termcmd`'s equivalent moment is
`redrawTab`, which issues `HomeAndClear` and replays: the same wholesale
takeover, and it must reset the same two pieces of state.

- [x] **M2.1: Write the failing tests**

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

- [x] **M2.2: Run to verify they fail.**
- [x] **M2.3: Implement.** Four pieces, one per writer class from finding 8:
      1. Route `redrawTab`'s replay through the writer loop as an event, and
         reset `hostScan` + the deferred slot there — it is `termcmd`'s wholesale
         takeover, so it needs couch's third gate rule (see above), not just the
         first two.
      2. Add the gate: a `ptychild.Screen` fed child chunks before they are
         written, consulted before any console-originated write, with a
         deferred-paint slot flushed on the next boundary-ending chunk.
      3. **Make the RUNTIME incapable, rather than fixing call sites.** The
         first form of this step routed `termcmd`'s five `RunZellijAction` calls
         through `Quiet`. Wrong altitude: `termcmd` hands its Runtime to
         `layoutcmd` and `draftroute` (`run.go:181,194,198,514`), which hold five
         more sites between them, and a call-site rule covers neither those nor
         the next one added. So **neither** `OSRuntime` method gives a
         subprocess the pane's descriptors, and the call sites are left alone.
      4. **Diagnostics reach the operator without touching the pane's fd.**
         `runZellijCaptured` folds the subprocess's stderr into the returned
         error, and `terminalMux.reportError` puts it on the pane through the
         writer loop. Keeping the pane clean must not mean DESTROYING the
         report: discarding both descriptors while the wheel-tick callers also
         drop the error (`_ = rt.RunZellijAction("scroll-up")`) makes a failure
         completely silent, which is worse than the noise it replaced.
      5. **Diagnostics queue; paints coalesce.** Same gate, different deferral
         policy. A paint renders current state, so a later one supersedes an
         earlier one and only the freshest should land. An error is an EVENT:
         every one lands, and it survives a takeover, because unlike a paint it
         is not made obsolete by the screen being replaced.
      6. **The resize goroutine** (`run.go:261-266`) calls `inheritSize`, which
         becomes a writer in M3. It joins the writer loop here, not in M3 —
         ARCH-ORDER already asserts resize and paint "serialize by construction"
         on that loop, and that claim is false until this piece lands.
- [x] **M2.3b: Prove the subprocess routing by the FD, not the method name.** A
      `Runtime` fake records which method each call site used — but a
      `Quiet`-only refactor passes that assertion while `cmd.Stderr` still points
      at the pane, which is how BR-4 survived three rounds. So also drive
      `runZellij` directly with recording writers and assert the subprocess's
      stdout AND stderr both land there rather than on the pane's descriptors.
      This is what `TestOnlyOneGoroutineWritesTheHost` structurally cannot see.
- [x] **M2.4:** `go test ./cmd/internal/termcmd/ -count=1 -race` — the race detector is the point, not decoration.
- [x] **M2.5: Manual** — switch tabs rapidly under load (`yes` in one tab) and confirm no corruption. Record what was observed in `## Log`, not "it worked".
- [x] **M2.6: Commit**, `sdlc milestone-close --issue 199 --milestone M2`.

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
- [ ] **M3.3: Implement `RenderStrip`** — pure, returning `RenderedStrip{Body, Spans}`, sanitizing and truncating via `rowtext.Sanitize`/`rowtext.Fit` (the shared package M3 extracts; `couchtty`'s `sanitize`/`truncate` are unexported and unreachable — PQ-7).
- [ ] **M3.4: Wire it.** `Reservation{Edge: EdgeBottom}` sized from the pane; child pty gets `ChildRows()`; repaint on tab change, resize, and `batch.RowDirty` **read inside the Sink callback** (finding 7).
- [ ] **M3.5: The re-`Reserve` rule** (ARCH-ORDER's most-likely-wrong): on a `batch.RowDirty` batch, re-`Reserve` *before* repainting. Test: simulate a child emitting `\x1b[r` (margin reset), assert the next repaint re-emits the region and not only the row.
- [ ] **M3.6: Degrade `rename-pane`** to the active tab name; assert `RunZellijAction` still receives a rename on tab switch (the `#118`/`#123` consumers) and that it is no longer the packed multi-tab string.
- [ ] **M3.7:** `go test ./cmd/... -count=1`, then **manual in a real layout3 pane**: run `nvim`, confirm the strip survives its startup clear and its own margin changes; quit; confirm the shell is not left scrolling in a box.
- [ ] **M3.8: Commit**, `sdlc milestone-close --issue 199 --milestone M3`.

## M4 — take the frame off

**Files:**
- Modify: `zellij/layouts/main-3.kdl` — **the SOURCE**. The path under
  `cmd/internal/runtimebundle/assets/runtime/files/` is a GENERATED MIRROR
  (`artifactpath.GeneratedMirror`), regenerated by `runtimebundle-generate`,
  which `make test` runs FIRST — so an edit there is silently reverted before a
  single test observes it (PQ-6).
- Modify: `zellij/config.kdl` — source, same reason; its scroll-indicator
  rationale is the comment M4.4 corrects.
- Regenerate: `make runtimebundle-generate`, and commit the mirror it produces.
- Test: `cmd/internal/runtimebundle/` layout assertion, reading the SOURCE.

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

### 2026-09-07 — M2 boundary review (REWORK → addressed)

**BR-27 is the one that mattered: I kept the pane clean by destroying the
report.** Sending both subprocess descriptors to `io.Discard` looked like the
clean answer, but the wheel-tick callers also drop the error
(`_ = rt.RunZellijAction("scroll-up")`), so a failing action became
*completely silent* — strictly worse than the noise it replaced, and exactly
the failure-reported-as-nothing shape `#208` spent fourteen rounds on.
`runZellijCaptured` now folds the subprocess's stderr into the returned error,
where a caller can report it; the pane's fd is still never handed over.

The same finding's second half: `reportError` shared the coalescing `owed` slot
with paints, so a routine repaint could swallow an error. They share the gate
but not the deferral policy now — a paint renders current state and the freshest
wins; a diagnostic is an EVENT, so every one lands and it survives a takeover,
which a paint deliberately does not.

**BR-26: two assert-absent tests could not fail.** The takeover test asserted
only that `STALE` was absent, which passes if nothing was written at all — a
broken writer, a dropped event, a recorder on the wrong stream. It now checks
the takeover's own replay arrived first. The subprocess test had the same hole
at a different layer: if `captureFD` silently failed, every arm passed. It now
writes a control string through the captured descriptor and fails if the capture
cannot see it. Both are positive controls, and the rule generalises: *an
assertion that something is ABSENT is worthless without an assertion that the
apparatus can see it PRESENT.*

**BR-28 was the superseded-facts class again — and the test I added at M1 caught
it.** M2.3's steps 3-5 still described routing call sites through `Quiet`, an
implementation the code stopped having when I moved the fix to the Runtime. The
guard did not know those tokens; adding them made it red immediately, and
mutation-checking confirmed it. That is the first time this class was caught by
something other than a reviewer.

### 2026-09-07 — M1 close (FIX-THEN-SHIP), and the sweep becomes a test

**BR-17 was raised three rounds running, and the third round's point was not
"three more instances".** It was that I wrote the rule — correct the class, not
the site — shipped an acceptance grep with it, recorded *"All swept."*, and then
did not run the grep. Each round found live superseded prose in this file, once
in the very commit that moved the probe. A sweep that depends on remembering to
sweep is the same defect as a consumer set that depends on remembering to update
it, which is the family this issue has now hit six times.

So it is a test: `tests/plan-superseded-facts-test.sh`, wired into `make test`.
It pairs each superseded token with its replacement and bounds the check to the
plan body, so `## Revisions` may still quote a dead fact while narrating the
correction. What stays hand-written is *what counts as superseded* — a judgement
made when a finding lands. What is no longer hand-run is the CHECKING.
Mutation-verified: restoring "the same helpers `couchtty` uses" turns it red.

Live sites it caught and fixed: finding 5 still said `scratchpad/199-probe/`
(after the probe moved), M3.3 and ARCH-SECURE still promised `couchtty`'s
unexported `sanitize`/`truncate` that PQ-7 established are unreachable, and the
probe's own doc comment still printed the old `cmd/probes/` path.

**BR-4, third raise, and the diagnosis was right both times.** Finding 8 named
the stderr hole correctly while M2.3's steps routed only stdout — so the plan
*described* a closed envelope and *specified* an open one, and M2.3b's assertion
("no site calls `RunZellijAction`") would pass a `Quiet`-only refactor with
`cmd.Stderr` still pointing at the pane. M2.3 now has pieces 5 and 6:
`runZellij(args, stdout, stderr io.Writer)`, and the resize goroutine
(`run.go:261-266`) joining the writer loop — ARCH-ORDER already claims resize
and paint serialize on that loop, which is false until it does. M2.3b asserts
the fd handed to the subprocess, not the method name.

**BR-21's class: the atlas stated the rule without its exception.**
`atlas/index.md` said probes live in `probes/`; `cmd/probes/couchstartrecovery`
is a second home, reachable only through its own target because it takes an
argument the wholesale loop cannot supply. A rule with an unstated exception is
one a reader can follow into the wrong place — which is what I did. The atlas
now carries both, and the test for which home applies: *does it run with no
arguments?*

**BR-20's residual.** The panic was fixed but `line000` stayed an unused printed
aside. It is now a checked precondition: if the earliest line is still on screen
the region never scrolled, and "the marker sits below the last line" would be
true for a reason that proves nothing — the probe would report HONORED without
having tested anything. That case now exits PROBE-INCONCLUSIVE.

### 2026-09-07 — M1 boundary review (FIX-THEN-SHIP)

**BR-16 was the sharp one, and it was mine.** Finding 5 is this plan's
load-bearing measurement — "M1/M3 would have been built on sand" — and it closed
with *"Probe kept at `scratchpad/199-probe/`"*, a path that exists on no one
else's machine and in no commit, while `atlas/architecture.md` states the result
as settled fact for every future reader. A one-time manual reading with no
reproducible apparatus is not evidence anyone can re-check. The probe is now
`probes/zellijscrollregion` (the repo's probe home is
`probes/`, which `make test-smoke` runs wholesale), self-locating so it runs from anywhere, and it
reproduces: `DECSTBM HONORED — 200 lines scrolled in rows 1..21 while the
reserved row 23 held its paint`. Its doc comment records the three ways it
produced a CONFIDENT WRONG ANSWER before being fixed — zellij refusing to nest,
`--session`+`--layout` meaning attach rather than create, and `dump-screen`
being unable to answer a positional question — because each of those looked
exactly like "not honored".

**BR-21: I put the probe in the wrong home, and the atlas already said where.**
`atlas/index.md:17` states it plainly — `probes/` at the repo root, where
`make test-smoke` runs **every** directory *"so a new probe is covered by
existing it rather than by remembering to add a line."* I created
`cmd/probes/zellijscrollregion` instead, next to the one probe that lives there
for its own reasons, and then hand-added it to two lists to compensate — the
exact remembering the convention exists to abolish. Moved to
`probes/zellijscrollregion`; `make test-smoke` picks it up with no list at all,
verified. The Makefile comment on that target cites `#146 M1 BR-16`, which is
this same finding at its first occurrence: a probe quoted as evidence must be
re-runnable by anyone, which means a target rather than a remembered `go run`.

**BR-19 and BR-20 were live bugs in the probe I had just landed as evidence.**
Its reader goroutine wrote a `strings.Builder` that the verdict read with no
synchronisation — a data race across a goroutine that never stops, on the single
artifact the verdict is computed from. And `frame[len(frame)-3000:]` panics
whenever the pty produced under 3000 bytes, which is exactly the failed-session
path the probe must survive to report: a crash there replaces a diagnosis with a
stack trace. Both fixed; `go run -race ./probes/zellijscrollregion` is clean and
still reproduces.

**BR-17 is this issue's recurring defect, applied to the plan itself.** The six
plan-quality corrections each landed at one site and left the superseded facts
standing everywhere else: `TakeRowDirty` still named as the repaint trigger in
five places after finding 7 replaced it, "two writers" after finding 8 derived
six, `run.go:229` as the rename-pane consumer after finding 9 derived the real
set. All swept.

**BR-4 was half-fixed, which is worse than not fixed.** Routing every
`RunZellijAction` through `RunZellijActionQuiet` closes stdout — but `runZellij`
hardwires `cmd.Stderr = os.Stderr` for *both* methods, so a failing `zellij
action scroll-up` still writes into the pane per wheel tick. I had written "M2's
envelope covers all five" while the noisiest failure path stayed open. Six
writers now, and M2.3 gives `runZellij` a `stderr io.Writer`.

Two Minors, both real: M1.6's grep was ticked as "returns nothing" when as
written it returns 35 lines (the invariant held; the command didn't check it —
it needed `| grep -v _test.go`), and `NewReservation` validated only the edge,
so the "validating door" admitted `rows: 0` and handed back a Reservation whose
every method silently no-ops.

**And landing the probe hit the same class again**: two more hand-maintained
lists (`artifactpath`'s inventory, `.gitignore`'s main-package binaries) that a
new file falls out of silently. That is five such lists in one milestone.

### 2026-09-07 — M1 landed, with two deviations worth recording

**M1.5's acceptance was restated, as the plan gate (PQ-2) predicted.** "No couch
test edited" is unsatisfiable: `couchtty/reserve_test.go` calls the four moved
functions directly, so the package cannot compile until those tests move with
them. The invariant that actually carries the meaning -- *this was a move, not a
rewrite* -- is that the BEHAVIOURAL couch tests are untouched, and they are:
`git diff --stat cmd/internal/couchtty/{console_live,vtscreen}_test.go` is
empty, and those are the tests that exercise the reserved row end to end.

**Three consumer sets needed the move**, none of them in the plan's file list:
`core_concepts_contract_test.go`'s `conceptPlans` and `conceptInventory` (the
gate named the first, not the second), `#146`'s Core concepts row (now marked
deleted so the contract asserts the symbols are ABSENT from couchtty), and
`artifactpath`'s exhaustive production inventory. All three are hand-maintained
lists that a new or moved file silently falls out of -- the class `#188` tracks.

**One real bug, caught by the suite.** The first lift gave `Console` a
`reservation()` method that read `c.size` under `c.mu`. Two call sites --
`ChildSize` and `applyLayout` -- already hold that lock, so the package
deadlocked and the suite hung for four minutes rather than failing. Replaced
with a plain `bottomReservation(rows uint16)`: taking rows as an argument makes
every call site obviously safe and leaves reading `c.size` where the locking
discipline already lives.

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
