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

## The complete disposition table

Every mouse report reaching couch gets exactly one of four answers. Written as a
table because the first draft had a rule per prose paragraph, and the gate found
two of them contradicting each other (PQ-4, PQ-12).

| report | child has mouse mode | disposition |
|---|---|---|
| press, button 0, on couch's row | either | **couch** — map to an actor |
| press, any other button, on couch's row | no | **swallow** |
| press, any other button, on couch's row | yes | **forward** |
| release, on couch's row | no | **swallow** |
| release, on couch's row | yes | **forward** — the child needs it to close a drag |
| anything, elsewhere | no | **swallow** |
| anything, elsewhere | yes | **forward** verbatim |
| unparseable / over-long prefix | either | **release as literal input** — see below |

**The release rule is NARROWER than `termcmd`'s, deliberately.** `termcmd`
forwards every release unconditionally (`run.go:449-452`), which is correct where
the child is already receiving presses. In couch a child that never enabled
tracking must receive NOTHING — the Done-when says zero bytes — and a release it
never saw a press for is an unpaired event as well as a contradiction. So the
release forwards only when the child has mouse mode. Recorded here because the
two rules differing is a decision, not an oversight.

**The wheel** (buttons 64/65) is not couch's: on couch's row it swallows or
forwards by the same rule as any other non-zero button. couch does not translate
it to zellij scroll — that is `termcmd`'s job in `termcmd`'s context, and a
second translator would be two policies for one gesture.

**The unterminated prefix has a bound, and this is the #127 hazard.** The
Interceptor's `seqPartial` arm holds bytes with no limit and returns `rest=nil`,
so a stray or pasted `\x1b[<` with no terminator would park every following
keystroke — the dead keyboard `run.go:575-580` records having shipped once. An
SGR report is bounded (`\x1b[<` + three numbers + one of `Mm`, ~20 bytes), so:
a held mouse prefix longer than `maxSGRReport` is released as ordinary input,
and the existing escape-ambiguity timer already covers the "operator stopped
typing mid-sequence" case. The bound is what releases the hold.

## Non-goals

Stated because their absence was read as an oversight rather than a decision:

- **Drag, hover, motion tracking.** couch requests `?1000` (click) and never
  `?1002`/`?1003`; motion reports arrive at pointer-movement rates for a feature
  that needs human click rates (`ARCH-CONSTRAINTS`).
- **Right and middle click, and the wheel, as couch gestures.** Only button 0
  press acts. See the table for what happens to the rest.
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

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ChipSpan` | `cmd/internal/couchtty/reserve.go` | new |
| `RenderedStatusRow` | `cmd/internal/couchtty/reserve.go` | new |
| `ColumnToActor` | `cmd/internal/couchtty/reserve.go` | new |
| `ActorExtent` | `cmd/internal/couchtty/menu_render.go` | new |
| `PointToActor` | `cmd/internal/couchtty/menu_render.go` | new |
| `MouseDisposition` | `cmd/internal/couchtty/mouse.go` | planned — M2 |
| `RouteMouseReport` | `cmd/internal/couchtty/mouse.go` | planned — M2 |
| `mouseinput.Event` / `mouseinput.Find` | `cmd/internal/mouseinput/mouseinput.go` | new |
| `seqMouse` | `cmd/internal/couchtty/keys.go` | planned — M2 |
| `RenderStatusRow` | `cmd/internal/couchtty/reserve.go` | modified |
| `Interceptor.FeedHit` | `cmd/internal/couchtty/keys.go` | planned — M2 |

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
  - **Extents are DERIVED from what already exists** (PQ-8), not invented:
    `renderRootMenuFrame` already builds `rootLine{actorStart: true}`
    (`menu_render.go:336,360`) and already scroll-windows on actor boundaries
    (`:374`), so multi-line actors are testable today without `#173`.
  - **The rows shift after that, and the extents must be re-based.**
    `RenderMenuView` replaces `lines[0]` with the breadcrumb and inserts the
    notice at index 1 (`:100-127`), moving every actor row down. Extents are
    therefore emitted from `RenderMenuView`, after the shift — computing them in
    `renderRootMenuFrame` would be right by one line and wrong by one, which is
    the least visible way to be wrong.

- **The switcher's input path (PQ-11): there is no new one.** A click does not
  enter through `panelkeys.go`, and no `PanelKey` kind is added. The `Interceptor`
  sees ALL operator input before focus is considered, so `onMouse` is reached the
  same way whether an actor or the panel has focus, and decides there: a report on
  the last row maps through `ColumnToActor`; any other row, with the panel
  focused, maps through `PointToActor`; any other row with an actor focused
  forwards or swallows. Named explicitly because "no change needed" is a claim
  that has to be checked, not an omission.

- **mouseinput.Event / FindSGR** — `termcmd`'s parser, promoted to a package both
  callers import. Moved rather than copied: `termcmd` keeps working through the
  new package, so there is one parser and one prefix rule
  (`ARCH-DRY`). The wheel/release policy moves with it as documented behaviour.

- **MouseDisposition / RouteMouseReport** — `couch | forward | swallow`, from
  `(event, hostRows, childWantsMouse)`. Three-way because "the child asked for
  this" and "the child must never see this" are the two cases this issue exists
  to separate; a bool collapses them.

- **seqMouse + `Interceptor.FeedHit`** — carries the RAW bytes alongside the
  decoded event, never the event alone. The `forward` disposition writes the
  child the exact bytes the terminal sent; re-encoding them from the parsed
  fields would be a second source of truth for the wire format and would differ on any
  form the encoder did not reproduce (`ARCH-DRY`, PQ-12). `findSGRMousePress`
  already returns raw, so this is a matter of not discarding it.
  PQ-2, and it is why swallow is not free.
  The Interceptor splits operator input and forwards anything it does not
  recognise, so today a mouse report reaches the child as ordinary bytes. It must
  learn the SGR shape to be able to withhold one. Its partial-sequence rule
  already exists (`held`) and `isSGRMousePrefix` supplies the predicate.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Console.onMouse` | `cmd/internal/couchtty/console.go` | planned — M3 | routing a decoded event to a switch |
| `hostty.MouseClickTracking` | `cmd/internal/hostty/control.go` | planned — M2 | couch's own DECSET/DECRST |

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

## "Same path as Return" — what Return decides that a click must not

The Done-when says a click takes Return's path AND that a click is always a
manual switch. Those contradict on exactly one input, and the gate found it
(PQ-13, Critical): **a click on an actor that is currently paging.**

The mechanism, site by site:

- `reduceActionKey`/`reduceRootKey` dispatch the declared `switch` operation
  (`menu.go:445-449`).
- `runMenuOperation` captures pending attention for a `switch`
  (`console.go:1434-1436` → `attention.Capture`, `attention.go:75-79`).
- `ExecuteConsoleOperation` reads that capture and sets
  `how = arrivalNotification` when it is nonzero (`console.go:1625-1628`).
- `#170`'s rule: only `arrivalNotification` is NON-pinning.

So a click reusing Return's path unmodified is classified as a notification hop
whenever the clicked actor was paging — which is the one thing the Done-when
forbids.

**Resolution: same OPERATION, one declared difference.** The click dispatches the
same `switch` through the same declared surface — no private verb — and marks its
origin manual so the capture at `console.go:1434` is skipped. `arrivalOrdinary`
then falls out of the existing derivation at `:1625` with no second rule.

Rejected: calling `switchTo(id, true, arrivalOrdinary)` directly. It would be a
parallel path that skips the operation queue, the projection refresh and the
notice bookkeeping — the "reuse the path" requirement exists to prevent exactly
that drift, and satisfying it by bypassing the path is not satisfying it.

**And the rule the finding actually asks for, applied to every task below:** each
task's Files list names every production site its steps assert, and any "reuse
what X does" claim states what X decides where the new caller differs.

## Done-when → Task map

PQ-11 asked for the RULE, not the instance: every Done-when bullet maps to a
named task with a named test, and writing the map is what catches a bullet with
no delivery. Two bullets had none — the switcher had no input path, and nvim
scroll had no step — and both were found by building this table rather than by
re-reading the plan.

| Done-when bullet | Task | Test |
|---|---|---|
| chip click attaches; empty row does nothing | 3, 10 | `TestColumnToActor*`, `TestClickOnAChipTakesTheSwitchPath` |
| switcher single click enters, same path as Return | 4, 10 | `TestClickInTheSwitcherTakesTheReturnPath` |
| click on a multi-line actor's notification/description line selects it | 4 | `TestPointToActorSpansEveryLineOfAnActor` |
| click is a MANUAL switch, re-pins `previous` | 11 | `TestClickIsAManualSwitch` (mutation-checked) |
| point-to-actor is pure, unit-tested with no terminal | 4 | same |
| column-to-actor tested against the same render pass, clipped and dropped | 2, 3 | `TestChipSpansMatchTheDrawnRowWhenClipped` |
| child with no tracking receives ZERO mouse bytes | 6, 7, 8 | `TestMouseReportsAreNotForwardedAsOrdinaryBytes`, `TestChildWithoutTrackingReceivesNoMouseBytes` |
| child WITH tracking still gets its events unchanged | 6, 8 | `TestForwardPreservesRawBytes` |
| teardown leaves mouse reporting off | 8 | `TestTeardownDisablesMouseTracking` |
| nvim selection and scroll still work in an attached session | 12 | **manual** — no automatic test; steps in Task 12 |
| `pair#166` re-evaluated | 12 | n/a — a written answer, not a test |

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
      **Include a state with a NOTICE**, so the index-1 insert is exercised — an
      extent computed before that shift is right by one line and wrong by one.
- [ ] **Step 2: Run — FAIL.**  **Step 3: Implement** by deriving from the
      existing `rootLine.actorStart` and emitting extents from `RenderMenuView`
      AFTER the breadcrumb replacement and notice insert.
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

**Files:**
- Create: `cmd/internal/couchtty/mouse.go`, `mouse_test.go`
- Register: `cmd/internal/artifactpath/manifest.go` — `NonArtifactSources`
  (`:482-560`) lists every `couchtty/*.go`, and `make test` fails on an
  unregistered one. **The rule, not the instance:** every task that creates a
  production file registers it in the same task, because a guard whose input is
  a hand-maintained list is one the next addition skips. Task 1 does this for
  `mouseinput`; this is the second instance.

- [ ] **Step 1:** Test EVERY row of the disposition table above, including the
      two that contradict a naive reading: a release to a child with NO mouse
      mode is **swallowed** (forwarding it would break the zero-bytes Done-when
      and hand the child an unpaired event), and a non-zero button on couch's own
      row is swallowed or forwarded rather than acted on. Table-driven, one case
      per row, so a row added later without a case is visible.
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
      `FeedHit` to return the decoded event AND the raw bytes — `forward` writes
      the wire form, never a re-encoding.
- [ ] **Step 3b: Bound the hold.** A held mouse prefix longer than
      `maxSGRReport` is released as ordinary input. Test it directly: feed
      `\x1b[<` followed by 100 digits and assert the keystrokes after it still
      reach the child. Without this a stray prefix parks the keyboard, which is
      #127 and it has shipped once already.
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

**Files:**
- Modify: `cmd/internal/couchtty/console.go` — `onMouse` (new),
  `processInput`'s `HitMouse` arm, `runMenuOperation:1434` (skip the capture for
  a manual origin), `MenuOperationOrigin` in `menu.go` (the manual marker)
- Test: `cmd/internal/couchtty/console_mouse_test.go`

**What Return decides that this caller differs on:** the attention capture. See
the section above; everything else — the declared `switch` operation, the
operation queue, the projection refresh, the notice bookkeeping — is identical
and must stay identical.

- [ ] **Step 1:** Test that a click on a chip dispatches the same declared
      `switch` operation `ctrl-space`+Return dispatches, asserted through the
      operation dispatcher rather than a parallel handler.
- [ ] **Step 2: Run — FAIL.**  **Step 3: Implement** `onMouse` and the manual
      marker.
- [ ] **Step 4: Run — PASS.**  **Step 5: Commit.**

### Task 11: A click is a MANUAL switch

**Files:**
- Assert against: `cmd/internal/couchtty/console.go:1434` (capture skipped),
  `:1625` (arrival derived), `cmd/internal/couchtty/attention.go:75`
  (`Capture`), `SwitchTracker` (`previous` pinning)
- Test: `cmd/internal/couchtty/console_mouse_test.go`

- [ ] **Step 1:** Test that clicking re-pins `previous` so `ctrl+backspace`
      undoes it — with the clicked actor **PAGING**, which is the only input
      where click and Return legitimately differ. Assert BOTH halves: the arrival
      is ordinary, and the bell is still cleared (every landing clears it; only
      the pinning differs).
- [ ] **Step 2:** Mutate the manual marker off and confirm it reddens. A test
      that passes because it asserts the default is not asserting the rule — and
      here the default is the wrong answer, so this step is not optional.
- [ ] **Step 3: Commit.**

### Task 12: Close M3

**Files:**
- Modify: `cmd/internal/couchtty/menu.go:19` (`menuControls`), `README.md`
  (the couch section the guard is scoped to), `atlas/couch.md`
- Assert against: `cmd/internal/couchcmd/readme_test.go`
  (`TestREADMEDocumentsEveryPanelControl`, scoped to the couch section)

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
