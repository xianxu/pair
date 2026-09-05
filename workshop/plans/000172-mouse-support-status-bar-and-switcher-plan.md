# Mouse support: click the status bar and the switcher — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clicking an actor — its chip on the reserved row, or anywhere in its
extent in the switcher — switches to it, without the child ever receiving a
mouse byte it did not ask for.

**Architecture:** Three layers, and the middle one is the work. A PURE layer maps
a click to an actor, deriving its coordinates from the same render pass that drew
them. An OWNERSHIP layer tracks which mouse modes the *child* enabled by scanning
its output, because mouse reporting is a terminal-global mode and couch cannot
enable it for itself without deciding what the child sees. A thin WIRING layer
feeds the resulting actor into the switch path `ctrl-space`+Return already uses.

**Tech Stack:** Go; `couchtty` (render + console), `hostty` (terminal control),
existing `ptychild.Screen` framing for stream scanning.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `ChipSpan` | `cmd/internal/couchtty/reserve.go` | new |
| `RenderedStatusRow` | `cmd/internal/couchtty/reserve.go` | new |
| `ColumnToActor` | `cmd/internal/couchtty/reserve.go` | new |
| `ActorExtent` | `cmd/internal/couchtty/menu_render.go` | new |
| `PointToActor` | `cmd/internal/couchtty/menu_render.go` | new |
| `MouseReport` | `cmd/internal/couchtty/mouse.go` | new |
| `ParseMouseReport` | `cmd/internal/couchtty/mouse.go` | new |
| `MouseModes` | `cmd/internal/couchtty/mouse.go` | new |
| `ScanMouseModes` | `cmd/internal/couchtty/mouse.go` | new |
| `MouseDisposition` | `cmd/internal/couchtty/mouse.go` | new |
| `RouteMouseReport` | `cmd/internal/couchtty/mouse.go` | new |
| `RenderStatusRow` | `cmd/internal/couchtty/reserve.go` | modified |

- **ChipSpan / RenderedStatusRow** — the column range each actor occupies, returned
  *by the render*. `RenderStatusRow` currently returns a string; it will return
  `RenderedStatusRow{Body string, Chips []ChipSpan}`.
  - **DRY rationale:** the spans must come from the pass that already clips chips
    to width. A second derivation would agree at comfortable widths and disagree
    at exactly the narrow ones where clipping happens — which is where a
    mis-mapped click is least testable by eye (`ARCH-DRY`).
  - **Future extensions:** the same span list is what a hover or a tooltip would
    need; nothing else has to change to add one.

- **ColumnToActor** — `(RenderedStatusRow, column) -> (ActorID, bool)`. Total: a
  column in no chip returns false, which is the "clicking bare row does nothing"
  rule expressed as a value rather than a branch at the call site.

- **ActorExtent / PointToActor** — an actor occupies a VARIABLE number of rows in
  the switcher (primary line, notification line, description line once `#173`
  lands), so the map is point→**actor**, never point→line.
  - **Relationships:** 1:1 with a rendered row group; N per rendered menu.
  - **DRY rationale:** same as ChipSpan — extents come out of `RenderMenuView`,
    which already computes the line layout and already knows about clipping and
    scrolling.

- **MouseReport / ParseMouseReport** — one decoded SGR report:
  `(button, col, row, press|release)`. SGR (`?1006`) only; the legacy X10
  encoding caps coordinates at 223 and fails SILENTLY on a wide or tall
  terminal, which is a wrong answer rather than a refused one.

- **MouseModes / ScanMouseModes** — the set of DEC private mouse modes the CHILD
  has turned on, folded from its output stream: `ScanMouseModes(state, chunk) ->
  MouseModes`. Pure, so the whole forward-versus-swallow rule is testable with
  no terminal and no child.
  - **Relationships:** one `MouseModes` per pane, not one per console — actors
    differ, and switching between them is the interesting transition.
  - **Future extensions:** the same scan answers `pair#166` (park/resume losing
    the child's mouse mode), which is why that issue should be re-evaluated
    rather than left punted.

- **MouseDisposition / RouteMouseReport** — `couch | forward | swallow`, decided
  from `(report, hostRows, childModes)`. The three-way answer is the entity: a
  bool would collapse "the child asked for this" and "the child must never see
  this" into one branch, and they are the two cases this issue exists to
  separate.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Console.mouse` | `cmd/internal/couchtty/console.go` | new | per-pane mode state + host writes |
| `hostty.MouseTracking` | `cmd/internal/hostty/control.go` | new | DECSET/DECRST strings |

- **Console.mouse** — holds the per-pane `MouseModes`, feeds child output through
  `ScanMouseModes` on the existing `writeChild` path, and applies the mode the
  terminal should be in on attach, switch, detach and teardown.
  - **Injected into:** nothing — it is the shell. Every decision it makes is a
    call into the pure functions above, so its own tests are about ORDERING, not
    about mapping.

- **hostty.MouseTracking** — the enable/disable strings, beside
  `ResetInteractiveModes` which already lists every mouse mode and is the
  teardown authority today.

**Test surface.** Every PURE row gets a colocated unit test that runs with no
terminal and no child. The INTEGRATION rows get tests that drive the console's
real input and output paths — per `workshop/lessons.md`, a guard tested by
calling it directly can be correct and unreachable.

---

## Ordering and interleaving (ARCH-ORDER)

This component holds state across events it does not control, so the transitions
are part of the design.

**State:** per pane, the `MouseModes` the child enabled. **Events:** child output
carrying DECSET/DECRST; operator clicks; attach; switch; detach; child death;
teardown.

`(state, event) -> (state, effects)`:

| state | event | next | effects |
|---|---|---|---|
| any | child writes `?1000h`/`?1002h`/`?1003h`/`?1006h` | + that mode | terminal already in it — no write |
| any | child writes the matching `l` | − that mode | if couch still needs tracking, re-assert couch's |
| child has none | click on couch's row | unchanged | act |
| child has none | click elsewhere | unchanged | **swallow** |
| child has some | click on couch's row | unchanged | act |
| child has some | click elsewhere | unchanged | **forward verbatim** |
| A active, A has modes | switch to B | B's modes | write B's, not A's |
| any | child dies / detach | drop that pane's | re-assert couch's own |
| any | teardown | — | `ResetInteractiveModes` (already the rule) |

**The unblockable events that apply here**, targeted rather than swept:

- **A second actor on the same terminal.** This is the one most likely to be
  mishandled: modes are terminal-global but state is per-pane, so a switch from an
  actor that enabled motion tracking to one that did not must actively *disable*
  it. Governed by **preempt** — the arriving actor's modes win — with no rollback,
  because the terminal has exactly one true state.
- **Process death mid-transition.** A child that dies with tracking on leaves the
  terminal in a mode nobody owns. Governed by **preempt**: couch re-asserts its
  own on the next paint, and teardown resets unconditionally (already true).
- **Completions arriving out of order.** A mouse report emitted while A was
  attached can be read after a switch to B. Governed by **ignore**: a report is
  evaluated against the focus at READ time, and a click that lands on a
  now-different actor is a mis-switch costing one `ctrl+backspace` — which is
  `pair#170`'s own switch-rule bargain, not a new one.
- **Not applicable, with reasons:** *retry of an already-applied step* — mode
  writes are idempotent DECSET strings, so a duplicate is a no-op; *connection
  loss* — the terminal is local and its loss is teardown, covered above.

**Where nondeterminism enters:** the chunk boundary. A child's `\x1b[?1002h` can
split across two reads, so `ScanMouseModes` must carry a partial-sequence tail —
the same hazard `ptychild.Screen`/`MidSequence` already handles for framing, and
the same one that made `Interceptor.held` necessary. **How a failing ordering is
reproduced:** feed the byte sequence one byte at a time; a scanner that only
works on whole chunks fails immediately and deterministically.

---

## Chunk 1: M1 — the pure layer

### Task 1: Chip spans out of the render pass

**Files:**
- Modify: `cmd/internal/couchtty/reserve.go:88`
- Test: `cmd/internal/couchtty/reserve_test.go`

- [ ] **Step 1: Write the failing test** — spans agree with the drawn row at a
      width that CLIPS:

```go
func TestChipSpansMatchTheDrawnRowWhenClipped(t *testing.T) {
	m := StatusModel{Actors: []StatusActor{
		{Label: "alpha"}, {Label: "beta"}, {Label: "gamma-with-a-long-name"},
	}}
	for _, width := range []int{80, 40, 24, 12, 6} {
		got := RenderStatusRow(width, m)
		for _, chip := range got.Chips {
			// Every span must be inside the row it came from, and the text at
			// that span must be the actor's label.
			if chip.Start < 0 || chip.End > width || chip.Start >= chip.End {
				t.Fatalf("width %d: span %+v outside [0,%d)", width, chip, width)
			}
		}
	}
}
```

- [ ] **Step 2: Run — FAIL** (`RenderStatusRow` returns a string; `.Chips` does
      not exist).
- [ ] **Step 3: Implement** `ChipSpan`, `RenderedStatusRow`, and record each span
      in the SAME loop that appends the chip text.
- [ ] **Step 4: Run — PASS.** Then add the dropped-chip case: a width so narrow a
      chip does not render must produce NO span for it.
- [ ] **Step 5: Commit.**

### Task 2: Column to actor

**Files:**
- Modify: `cmd/internal/couchtty/reserve.go`
- Test: `cmd/internal/couchtty/reserve_test.go`

- [ ] **Step 1:** Test: a column inside a chip returns its actor; a column in the
      gap between chips returns false; a column past the last chip returns false;
      column 0 and column width-1 are both handled (off-by-one at both ends).
- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement** `ColumnToActor`.
- [ ] **Step 4: Run — PASS.**
- [ ] **Step 5: Commit.**

### Task 3: Actor extents and point-to-actor

**Files:**
- Modify: `cmd/internal/couchtty/menu_render.go`
- Test: `cmd/internal/couchtty/menu_render_test.go`

- [ ] **Step 1:** Test a menu where one actor renders across THREE lines
      (primary + notification) and its neighbour across one. Assert a click on
      the third line returns the same actor as a click on the first; assert a
      click on the neighbour returns the neighbour; assert a click below the last
      actor returns false.
- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement** `ActorExtent` out of `RenderMenuView` and
      `PointToActor`.
- [ ] **Step 4: Run — PASS.** Add the scrolled case: with a list taller than the
      viewport, a click maps against what is DRAWN, not against the inventory
      index.
- [ ] **Step 5: Commit.**

### Task 4: Parse an SGR mouse report

**Files:**
- Create: `cmd/internal/couchtty/mouse.go`, `mouse_test.go`

- [ ] **Step 1:** Test `\x1b[<0;12;34M` (press) and `...m` (release) decode to
      button 0 at col 12 row 34; a legacy X10 report is REFUSED rather than
      guessed at; a truncated report returns false; coordinates above 223 decode
      correctly (the reason SGR is required).
- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement** `MouseReport` + `ParseMouseReport`.
- [ ] **Step 4: Run — PASS.**
- [ ] **Step 5: Commit.**

---

## Chunk 2: M2 — mode ownership

### Task 5: Scan the child's mouse modes

**Files:**
- Modify: `cmd/internal/couchtty/mouse.go`
- Test: `cmd/internal/couchtty/mouse_test.go`

- [ ] **Step 1: Write the failing test, byte-at-a-time** — this is the ordering
      hazard named above, so it is the FIRST assertion, not an afterthought:

```go
func TestScanMouseModesSurvivesAChunkBoundaryAnywhere(t *testing.T) {
	const stream = "hello\x1b[?1002h world\x1b[?1006h!"
	for split := 0; split <= len(stream); split++ {
		var state MouseModes
		state = ScanMouseModes(state, []byte(stream[:split]))
		state = ScanMouseModes(state, []byte(stream[split:]))
		if !state.Has(1002) || !state.Has(1006) {
			t.Fatalf("split at %d lost a mode: %+v", split, state)
		}
	}
}
```

- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement** `MouseModes` + `ScanMouseModes` with a held tail.
- [ ] **Step 4: Run — PASS.** Add DECRST (`l`) removing a mode, and a
      multi-parameter `\x1b[?1000;1006h` setting both.
- [ ] **Step 5: Commit.**

### Task 6: The routing decision

**Files:**
- Modify: `cmd/internal/couchtty/mouse.go`
- Test: `cmd/internal/couchtty/mouse_test.go`

- [ ] **Step 1:** Test the three-way table above: couch's row → `couch`;
      elsewhere with no child modes → `swallow`; elsewhere with child modes →
      `forward`. The swallow case is the one the issue's Done-when names as
      "asserted, not assumed".
- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement** `MouseDisposition` + `RouteMouseReport`.
- [ ] **Step 4: Run — PASS.**
- [ ] **Step 5: Commit.**

### Task 7: The console owns the terminal's mode

**Files:**
- Modify: `cmd/internal/couchtty/console.go`, `cmd/internal/hostty/control.go`
- Test: `cmd/internal/couchtty/console_mouse_test.go` (new)

- [ ] **Step 1:** Test the transition that matters — attach A (which enables
      `?1002`), switch to B (which enabled nothing), and assert the host received
      the DISABLE. Drive it through the real output path, not by calling the
      scanner.
- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement:** feed `writeChild` through `ScanMouseModes` into the
      pane's state; apply the arriving pane's modes on switch; re-assert couch's
      own on child death and detach.
- [ ] **Step 4: Run — PASS.** Add: a child that never enabled tracking receives
      ZERO mouse bytes while couch's tracking is on.
- [ ] **Step 5: Commit.**

---

## Chunk 3: M3 — wiring

### Task 8: Click routes into the existing switch

**Files:**
- Modify: `cmd/internal/couchtty/console.go`
- Test: `cmd/internal/couchtty/console_mouse_test.go`

- [ ] **Step 1:** Test that a click on a chip takes the SAME path as
      `ctrl-space`+Return — asserted against the same handler, so a parallel
      implementation cannot drift (`ARCH-DRY`).
- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement** the `HitMouse` arm, dispatching through
      `hitHandlers()` like every other intercepted input.
- [ ] **Step 4: Run — PASS.**
- [ ] **Step 5: Commit.**

### Task 9: A click is a MANUAL switch

**Files:**
- Test: `cmd/internal/couchtty/console_mouse_test.go`

- [ ] **Step 1:** Test that clicking an actor re-pins `previous`, so
      `ctrl+backspace` undoes it — INCLUDING a click on an actor that is showing a
      notification, which must NOT count as notification handling
      (`pair#170`'s `entered_via_notification` rule).
- [ ] **Step 2: Run.** If it passes without a change, mutate the switch call to
      the notification form and confirm it reddens — otherwise the test is
      asserting the default rather than the rule.
- [ ] **Step 3: Commit.**

### Task 10: Close M3

- [ ] **Step 1:** `menuControls` gains the mouse row so the README guard fires;
      atlas gets the mode-ownership rule and the point-to-actor invariant.
- [ ] **Step 2:** Re-evaluate `pair#166` against the new tracking — close, fix,
      or re-punt WITH a reason. The issue's Done-when requires an answer, not a
      default.
- [ ] **Step 3:** Full `make test`; `sdlc milestone-close --issue 172 --milestone M3`.
