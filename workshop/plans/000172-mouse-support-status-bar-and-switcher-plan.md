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

## What already exists, and why this section is first

The first draft of this plan designed a mouse-mode scanner, an SGR parser and a
wheel policy that are all **already in the tree**. The plan-quality gate caught
it (PQ-1, Critical). The correction is not a smaller plan — it is a different
one, because reuse changes which milestone is hard.

| Already built | Where | What it means for this issue |
|---|---|---|
| child mouse-mode tracking, split-read safe | `ptychild.Screen.Mouse()` (`screen.go:104`, DECSET table at `:415`) | M2 does NOT write a scanner. `Screen`'s own comment says it absorbed `termcmd.updateMouseMode` *because* per-read scanning could not see a split sequence — the exact hazard the first draft made its headline test. |
| SGR report parsing, including prefix detection for split reads | `termcmd.parseSGRMousePress`, `findSGRMousePress`, `isSGRMousePrefix` (`run.go:590-635`) | M1 does NOT write a parser. It **promotes** these out of `termcmd` so two callers share one (`ARCH-DRY`). |
| button + wheel policy | `termcmd/run.go:445-465` | The first draft had none, which PQ-4 called a live hazard. The rules already decided there: a **release always forwards** (the child needs it to close a drag), and wheel (64/65) goes to zellij scroll only when the app is not in mouse mode. couch inherits these rather than inventing them. |
| child mode restored across a switch | `ptychild` replay (`replay.go:46` — DECSET `?1006h` deliberately survives) | couch must NOT add a second writer for the child's modes (PQ-5). couch owns **its own** mode only, and must not fight replay. |

**The consequence for milestone shape.** The first draft called mode ownership
"the deliverable". It is not — `Screen` and replay already own the child's half.
What is genuinely absent is narrower and sits in `couchtty`: recognising a mouse
report in the operator's input stream at all, deciding couch/forward/swallow, and
mapping a click to an actor.

## Non-goals

Stated because their absence was read as an oversight rather than a decision:

- **Drag, hover, motion tracking.** couch requests `?1000` (click) and never
  `?1002`/`?1003`; motion reports arrive at pointer-movement rates for a feature
  that needs human click rates (`ARCH-CONSTRAINTS`).
- **Right and middle click, and the wheel, on couch's own row.** Only button 0
  press acts. Everything else on couch's row is swallowed, not guessed at.
- **Text selection and copy on the status row.** Enabling `?1000` costs the
  terminal's native selection on that row; that is a real loss and it is
  accepted, not solved here.
- **The notice area.** Only actor chips are targets; the notice is not clickable.

## ARCH-SECURE — the report is input the component did not produce

A mouse report arrives from the terminal and a DECSET arrives from the child;
neither is trusted structure. Both are PARSED into a typed value at the boundary
and refused if they do not fit: coordinates outside the rendered geometry map to
no actor (`ColumnToActor`/`PointToActor` return false rather than clamping), an
unparseable report is swallowed rather than forwarded as raw bytes, and a
coordinate large enough to overflow is a refusal — which is the reason SGR is
required, since the legacy X10 encoding caps at 223 and fails *silently*.

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `ChipSpan` | `cmd/internal/couchtty/reserve.go` | new |
| `RenderedStatusRow` | `cmd/internal/couchtty/reserve.go` | new |
| `ColumnToActor` | `cmd/internal/couchtty/reserve.go` | new |
| `ActorExtent` | `cmd/internal/couchtty/menu_render.go` | new |
| `PointToActor` | `cmd/internal/couchtty/menu_render.go` | new |
| `MouseDisposition` | `cmd/internal/couchtty/mouse.go` | new |
| `RouteMouseReport` | `cmd/internal/couchtty/mouse.go` | new |
| `mouseinput.Event` / `mouseinput.FindSGR` | `cmd/internal/mouseinput/mouseinput.go` | new (promoted from `termcmd`) |
| `seqMouse` | `cmd/internal/couchtty/keys.go` | new |
| `RenderStatusRow` | `cmd/internal/couchtty/reserve.go` | modified |
| `Interceptor.FeedHit` | `cmd/internal/couchtty/keys.go` | modified |

- **ChipSpan / RenderedStatusRow** — the column range each actor occupies,
  returned *by the render*. `RenderStatusRow` returns
  `RenderedStatusRow{Body string, Chips []ChipSpan}`.
  - **DRY rationale:** the spans must come from the pass that already clips chips
    to width; a second derivation agrees at comfortable widths and disagrees at
    exactly the narrow ones where clipping happens.
  - **Cross-package caller:** `wrapcmd/codex_working_test.go:166` calls
    `RenderStatusRow` and breaks on the signature change. Compiler-caught, named
    here so it is a step rather than a surprise.

- **ColumnToActor / ActorExtent / PointToActor** — total functions; a coordinate
  in no target returns false. An actor occupies a VARIABLE number of switcher
  rows, so the map is point→**actor**, never point→line.

- **mouseinput.Event / FindSGR** — `termcmd`'s parser, promoted to a package both
  callers import. Moved rather than copied: `termcmd` keeps working through the
  new package, so there is one parser and one prefix rule
  (`ARCH-DRY`). The wheel/release policy moves with it as documented behaviour.

- **MouseDisposition / RouteMouseReport** — `couch | forward | swallow`, from
  `(event, hostRows, childWantsMouse)`. Three-way because "the child asked for
  this" and "the child must never see this" are the two cases this issue exists
  to separate; a bool collapses them.

- **seqMouse + `Interceptor.FeedHit`** — PQ-2, and it is why swallow is not free.
  The Interceptor splits operator input and forwards anything it does not
  recognise, so today a mouse report reaches the child as ordinary bytes. It must
  learn the SGR shape to be able to withhold one. Its partial-sequence rule
  already exists (`held`) and `isSGRMousePrefix` supplies the predicate.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Console.onMouse` | `cmd/internal/couchtty/console.go` | new | routing a decoded event to a switch |
| `hostty.MouseClickTracking` | `cmd/internal/hostty/control.go` | new | couch's own DECSET/DECRST |

- **Console.onMouse** — reads the child's mode from `Screen.Mouse()` (not from a
  new tracker), calls `RouteMouseReport`, and on `couch` maps the coordinate to an
  actor and takes the SAME switch path Return takes.
  - **PQ-3, and it changes the dispatch:** `hitHandlers()` is
    `map[InterceptorHit]func()` and cannot carry coordinates. `FeedHit` therefore
    returns the decoded event alongside the hit, and `processInput` passes it —
    so the table keeps its "every hit has a handler" guarantee while one hit
    carries a payload.

- **hostty.MouseClickTracking** — `?1000;?1006` on, and off, beside
  `ResetInteractiveModes` which already lists every mouse mode and remains the
  teardown authority.

## Chunk 1: M1 — geometry, and one parser instead of two

### Task 1: Promote the SGR parser out of termcmd

**Files:**
- Create: `cmd/internal/mouseinput/mouseinput.go`, `mouseinput_test.go`
- Modify: `cmd/internal/termcmd/run.go:575-640`
- Register: `cmd/internal/artifactpath/manifest.go`

- [ ] **Step 1:** Move `mousePressEvent`, `parseSGRMousePress`,
      `findSGRMousePress`, `isSGRMousePrefix` into `mouseinput` as exported
      names. MOVE, do not copy — `termcmd` imports them back, so there is one
      parser.
- [ ] **Step 2:** Run `go test ./cmd/internal/termcmd/` — its existing tests are
      the regression suite for the move and must pass UNCHANGED. If a test needs
      editing to pass, the move changed behaviour and is wrong.
- [ ] **Step 3:** Carry the wheel/release policy comment with it; it documents
      decisions couch now inherits.
- [ ] **Step 4: Commit.**

### Task 2: Chip spans out of the render pass

**Files:**
- Modify: `cmd/internal/couchtty/reserve.go:88`, `cmd/internal/wrapcmd/codex_working_test.go:166`
- Test: `cmd/internal/couchtty/reserve_test.go`

- [ ] **Step 1: Write the failing test** — spans agree with the drawn row at
      widths that CLIP and DROP chips:

```go
func TestChipSpansMatchTheDrawnRowWhenClipped(t *testing.T) {
	m := StatusModel{Actors: []StatusActor{
		{Label: "alpha"}, {Label: "beta"}, {Label: "gamma-with-a-long-name"},
	}}
	for _, width := range []int{80, 40, 24, 12, 6} {
		got := RenderStatusRow(width, m)
		for _, chip := range got.Chips {
			if chip.Start < 0 || chip.End > width || chip.Start >= chip.End {
				t.Fatalf("width %d: span %+v outside [0,%d)", width, chip, width)
			}
		}
		// A chip that did not render has NO span -- the count must track the
		// drawn row, not the model.
		if drawn := strings.Count(got.Body, "\u2502"); len(got.Chips) > len(m.Actors) {
			t.Fatalf("width %d: %d spans for %d actors (drawn %d)", width, len(got.Chips), len(m.Actors), drawn)
		}
	}
}
```

- [ ] **Step 2: Run — FAIL** (returns a string).
- [ ] **Step 3: Implement**, recording each span in the SAME loop that appends
      the chip text. Fix the `wrapcmd` caller in this commit.
- [ ] **Step 4: Run — PASS**, including `go test ./cmd/...`.
- [ ] **Step 5: Commit.**

### Task 3: Column to actor

**Files:** Modify `reserve.go`; test `reserve_test.go`.

- [ ] **Step 1:** Test: inside a chip → its actor; in the gap between chips →
      false; past the last chip → false; column 0 and width-1 both handled.
- [ ] **Step 2: Run — FAIL.**  **Step 3: Implement** `ColumnToActor`.
- [ ] **Step 4: Run — PASS.**  **Step 5: Commit.**

### Task 4: Actor extents and point-to-actor

**Files:** Modify `menu_render.go`; test `menu_render_test.go`.

- [ ] **Step 1:** Test a menu where one actor renders across several lines
      (primary + notification) and its neighbour across one: a click on the last
      line returns the SAME actor as a click on the first; a click on the
      neighbour returns the neighbour; a click below the last actor returns false.
- [ ] **Step 2: Run — FAIL.**  **Step 3: Implement** out of `RenderMenuView`.
- [ ] **Step 4: Run — PASS.** Add the scrolled case: mapping is against what is
      DRAWN, not the inventory index.
- [ ] **Step 5: Commit.**

### Task 5: Close M1

- [ ] Register this plan in `conceptPlans`
      (`couchtty/core_concepts_contract_test.go`) so its Core-concepts rows are
      pinned — PQ-6; unregistered, all twelve rows assert nothing. Rows for
      unshipped milestones carry status `planned`.
- [ ] `sdlc milestone-close --issue 172 --milestone M1`.

## Chunk 2: M2 — routing, and the swallow that is not free

### Task 6: The routing decision

**Files:** Create `cmd/internal/couchtty/mouse.go`, `mouse_test.go`.

- [ ] **Step 1:** Test the three-way table: couch's row → `couch`; elsewhere with
      no child mouse mode → `swallow`; elsewhere with child mouse mode →
      `forward`. Plus the inherited policy: a RELEASE always forwards, even on
      couch's row, because the child needs it to close a drag.
- [ ] **Step 2: Run — FAIL.**  **Step 3: Implement** `RouteMouseReport`, reading
      the child's mode from `Screen.Mouse()`.
- [ ] **Step 4: Run — PASS.**  **Step 5: Commit.**

### Task 7: The Interceptor learns the SGR shape

**Files:** Modify `cmd/internal/couchtty/keys.go`; test `keys_test.go`.

- [ ] **Step 1: Write the failing test** — a mouse report must not reach the
      child as ordinary bytes, and a report split across a read boundary must not
      leak its prefix:

```go
func TestMouseReportsAreNotForwardedAsOrdinaryBytes(t *testing.T) {
	const report = "\x1b[<0;10;24M"
	for split := 1; split < len(report); split++ {
		var it Interceptor
		before, hit, _ := it.FeedHit([]byte(report[:split]))
		if len(before) != 0 {
			t.Fatalf("split %d leaked %q to the child", split, before)
		}
		if hit != HitNone {
			t.Fatalf("split %d fired early", split)
		}
		before, hit, _ = it.FeedHit([]byte(report[split:]))
		if hit != HitMouse || len(before) != 0 {
			t.Fatalf("split %d: hit %v, before %q", split, hit, before)
		}
	}
}
```

- [ ] **Step 2: Run — FAIL** (`seqMouse` and `HitMouse` do not exist; the bytes
      are copied through).
- [ ] **Step 3: Implement** `seqMouse` via `mouseinput.IsSGRPrefix`, and widen
      `FeedHit` to return the decoded event. `hit()`/`intercepts()` stay derived
      from one switch.
- [ ] **Step 4: Run — PASS.** Confirm `AllInterceptorHits` and the
      handler-table test still hold with the payload-carrying hit.
- [ ] **Step 5: Commit.**

### Task 8: couch enables its own tracking, and only its own

**Files:** Modify `console.go`, `hostty/control.go`; test `console_mouse_test.go` (new).

- [ ] **Step 1:** Test that a child which never enabled tracking receives ZERO
      mouse bytes while couch's tracking is on — driven through the real input
      path, the Done-when's "asserted, not assumed".
- [ ] **Step 2: Run — FAIL.**
- [ ] **Step 3: Implement:** enable `?1000;?1006` on start, disable at teardown
      beside the existing `ResetInteractiveModes`. **Do not write the child's
      modes** — replay owns those (PQ-5); assert that by testing a switch between
      two children with different modes and checking couch emitted no mode bytes
      of its own.
- [ ] **Step 4: Run — PASS.**  **Step 5: Commit.**

### Task 9: Close M2

- [ ] `sdlc milestone-close --issue 172 --milestone M2`.

## Chunk 3: M3 — wiring and the manual-switch rule

### Task 10: Click routes into the existing switch

**Files:** Modify `console.go`; test `console_mouse_test.go`.

- [ ] **Step 1:** Test that a click on a chip takes the SAME path as
      `ctrl-space`+Return — asserted against the same handler, not a parallel one.
- [ ] **Step 2: Run — FAIL.**  **Step 3: Implement** `onMouse`.
- [ ] **Step 4: Run — PASS.**  **Step 5: Commit.**

### Task 11: A click is a MANUAL switch

**Files:** Test `console_mouse_test.go`.

- [ ] **Step 1:** Test that clicking re-pins `previous` so `ctrl+backspace`
      undoes it — INCLUDING a click on an actor showing a notification, which
      must NOT count as notification handling (`arrivalOrdinary`, not
      `arrivalNotification`).
- [ ] **Step 2:** If it passes unchanged, mutate the call to
      `arrivalNotification` and confirm it reddens. A test that passes because it
      asserts the default is not asserting the rule.
- [ ] **Step 3: Commit.**

### Task 12: Close M3

- [ ] **Step 1:** `menuControls` gains the mouse row so the README guard fires
      (scoped to the couch section); atlas gets the routing rule and the
      point-to-actor invariant.
- [ ] **Step 2: Manual verification, PQ-7** — the Done-when's "nvim selection and
      scroll still work inside an attached pair session" has no automatic test
      because it needs a real terminal, a real nvim and a real pointer. Steps:
      attach a pair session, drag-select in the draft, wheel-scroll the agent
      pane, confirm both behave as before couch enabled tracking; then click a
      chip and confirm the switch. Record the result in `## Log` as a
      measurement, not as "it worked".
- [ ] **Step 3:** Re-evaluate `pair#166` against what landed — close, fix, or
      re-punt WITH a reason.
- [ ] **Step 4:** Full `make test`; `sdlc milestone-close --issue 172 --milestone M3`.
