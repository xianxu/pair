# Panel-Safe Input Routing Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A key release, focus or blur event arriving while couch's panel is focused must not exit couch.

**Architecture:** Two changes, one structural and one about severity. Structurally, every couch input event reaches `Presenter.Input` through a single door that knows whether a child exists; today three event kinds bypass the panel check that every other kind honours. For severity, the presenter gains a typed "I have no destination for this" answer, distinct from a terminal failure, so a routing-domain answer can never tear down terminal ownership — in `pair term` as well as couch, which has the same escalation.

**Tech Stack:** Go; `cmd/internal/terminal` (presenter/view), `cmd/internal/couchtty` (console), `cmd/internal/termcmd` (pair's mux); `hostty.NewFakeHost` + `ptychild.NewFakeChild` as the existing fakes.

---

## Background

Reproduced deterministically before planning (scratch test, 3/3 subtests): with the panel focused, each of `uv.KeyReleaseEvent`, `uv.FocusEvent`, `uv.BlurEvent` stops the console with `terminalFailure = terminal: no admitted endpoint`. `Console.teardown` then prints `"couch: terminal: %v"`, producing the operator's exact `couch: terminal: terminal: no admitted endpoint`.

Two facts make this a design bug rather than a missing nil check:

1. **The panel's "no admitted endpoint" is correct, not broken.** `Presenter.Panel` sets `p.selected = nil`, and `SelectView` with an empty `EndpointID` leaves `Selected` empty through `PresentView`, so the panel's healthy state is `State=Ready, Admitted="", selected=nil` — exactly the guard triple at `presenter.go:470`.
2. **`Console.terminalError` is two-valued.** Any non-nil error becomes `terminalFailure` and calls `Stop()`. There is no way for the presenter to answer "not applicable" — so a legitimate domain answer is escalated to a terminal-ownership failure.

Introduced by `f32bb4cf` (`#255 M3`); `git log -S` gives that arm exactly one commit.

The plan-quality gate then found the same shape a second time, on a path with no input in it: `Console.paintNow` feeds any `UpdateChrome` error to `terminalError`, and `UpdateChrome` refuses with a plain error when `p.selected == nil`. `showMenu` clears the endpoint (`presenter.Panel`, `console_menu.go:220`) before it flips focus (`:225-228`), so a repaint landing between those two reads "actor focused, no endpoint" and exits couch. That is the same defect, and it is why the fix is a typed answer in the terminal package rather than a panel check in the router alone.

The record-versus-world half of the brain diagnosis (a park wedged in `ThreadBusy`) is **out of scope here** and filed as `pair#271`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ErrNoDestination` | `cmd/internal/terminal/destination.go` | new |
| `noDestination` | `cmd/internal/terminal/destination.go` | new |

- **ErrNoDestination** — the sentinel for "the presenter has no endpoint to deliver this input to". Its own file, not `presenter.go`, so the purity boundary is visible from outside: `destination_test.go` runs with no writer, no host and no endpoint.
  - **Relationships:** 1:N — one sentinel, wrapped by every refusal site in `Presenter` (`Input`'s non-mouse arm and `mouseInput`'s unpresented arm). N callers classify with `errors.Is`.
  - **DRY rationale:** Today two sites hand-build two different unrelated strings for the same answer (`"terminal: no admitted endpoint"`, `"terminal: no presented mouse destination"`), and no caller can tell either from a write failure. One sentinel, one message shape (ARCH-DRY).
  - **Future extensions:** If a third "cannot deliver" condition appears (a retired origin, a released endpoint), it wraps the same sentinel and every existing caller classifies it correctly with no edit.

- **noDestination** — builds the wrapped error, carrying the reason and the `View` that produced it. Pure: `(reason string, v View) -> error`.
  - **Relationships:** 1:1 with `ErrNoDestination`; the only constructor.
  - **DRY rationale:** The issue's Done-when requires the surfaced condition to name the endpoint/view state. One constructor means the two refusal sites cannot describe the same state two ways.
  - **Future extensions:** Widens to carry the event kind if a caller ever needs to distinguish key from mouse without string matching.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Console.deliverPresenterInput` | `cmd/internal/couchtty/terminal_input.go` | new | `Presenter.Input` |
| `Console.deliverChildInput` | `cmd/internal/couchtty/terminal_input.go` | new | `deliverPresenterInput` |
| `Console.routeInputEvent` | `cmd/internal/couchtty/terminal_input.go` | modified | operator keystrokes |
| `Console.routeMouseEvent` | `cmd/internal/couchtty/terminal_input.go` | modified | operator mouse reports |
| `Presenter.Input` / `Presenter.mouseInput` / `Presenter.UpdateChrome` | `cmd/internal/terminal/presenter.go` | modified | parent terminal writes |
| `Console.paintNow` | `cmd/internal/couchtty/console.go` | modified | chrome repaint |
| `terminalMux.writeEvents` | `cmd/internal/termcmd/presentation.go` | modified | `pair term` input |

- **deliverPresenterInput** — the ONE door to `Presenter.Input` in couchtty. Classifies the answer: `ErrNoDestination` is returned to the caller, anything else goes to `terminalError`.
  - **Injected into:** nothing; it *is* the seam. The pure decisions it defends (`Focus.IsPanel`, `Transition`) are already pure and already unit-tested.
  - **Future extensions:** if couch grows a third input origin (the #121 control relay), it enters through this door and inherits the classification.

- **deliverChildInput** — the panel-aware wrapper for events that only mean something to a child. Drops on the panel; on an actor, a no-destination answer is anomalous and publishes a notice.
  - **Injected into:** `routeInputEvent`'s `route` closure and its release/focus/blur arm.
  - **Future extensions:** the natural home for per-event-kind panel semantics if a focus event ever needs to mean something to the panel.

- **routeMouseEvent** — unchanged in policy, rewired to the door. Mouse release and motion must still reach the presenter on the panel, because its gesture owner cancels and clips them; that is precisely why the door, not the wrapper, is the general primitive.

**Test surface.** `destination_test.go` colocated and IO-free. Console behaviour is tested through `hostty.NewFakeHost` + `ptychild.NewFakeChild` — the same fakes production flow uses (`ptychild.Child` is the concrete type on both sides, per `TerminalHandle`'s comment), so no new fake is introduced (ARCH-MOCK). `Child.Writes()` is the observation point for "the child actually received it".

### ARCH-ORDER — the input router's transitions

The router holds no durable state of its own; the state it reads is `Focus` (console) and `View` (presenter). The table is `(focus, event) -> (destination, effects)`:

| focus | event kind | destination | on no-destination |
|---|---|---|---|
| panel | printable / ESC / chord | menu reducer | n/a |
| panel | key release, focus, blur | **dropped** (was: presenter) | n/a |
| panel | mouse release / motion | presenter (gesture owner must clip) | ignore, non-fatal |
| actor | printable / ESC / paste | presenter | ignore, non-fatal |
| actor | key release, focus, blur | presenter | ignore, non-fatal |
| actor | mouse | presenter | ignore, non-fatal |

Events the caller cannot block, and their governing rule:

- **A key release whose press was consumed by chrome.** `ctrl+space` press opens the switcher, so the release straddles a focus change and arrives with the panel up. Governed by **ignore**; no rollback, because a release whose press never reached a child has nothing to undo. This is the event most likely to be mishandled and it is the reported one.
- **Focus/blur while the panel is up.** Couch itself asks for these (`\x1b[?1004h` in `parentModeDelta`'s `!known` branch, written on the *first* paint, panel or not). Governed by **ignore**.
- **Mouse release while the panel is up.** Must still be delivered — the presenter's `PendingMouseRelease`/`CancelMouse` machinery owns it. Governed by **deliver**, with a non-fatal no-destination answer.
- **Presenter released or failed concurrently with an input** (shutdown). Already handled: `p.call` returns `presenter unavailable`/`presenter released`, and `shutdownCancellation` filters the lifetime-cancelled case. Unchanged, still fatal-classified — correct, because that *is* a terminal-ownership failure.

Not applicable: cross-process ordering, and a second actor on the same state. This change adds no concurrency of its own. It is worth being exact about what it does share, because the obvious claim is wrong: `routeInputEvent` runs on the Run goroutine, but `c.focus` is **not** written only from there — `installObservedThreadActor` (`console.go:422`) writes it from the operation-queue goroutine (`console.go:576`), and `selectActorContext` (`terminal.go:145`) writes it via `runTerminalCommand` onto Run. Every read and write holds `c.mu`, so there is no race; and the one interleaving that matters — focus flipping to panel between the capture and the re-read in `deliverChildInput` — turns a deliver into a drop, which is the panel's own semantics and needs no rollback. Structural enforcement is the AST guard in Task 4: `Presenter.Input` is unreachable from couchtty except through the door.

### ARCH-CONSTRAINTS

Keystroke path. Per event the change adds one already-taken mutex read of `c.focus` and one `errors.Is`; panel-focused releases/focus/blur now skip a full presenter FIFO round-trip entirely, so the path gets strictly cheaper, not dearer. No new allocation per event (the sentinel is package-level; `noDestination` allocates only on the refusal path, which is off the happy path). No budget change to declare.

An earlier draft published a notice on a refused actor-focused keystroke; it was removed, and cost is the second reason. A notice repaints (`publishNotice` → `paintNow` → `UpdateChrome` → `Snapshot` + `Compose` + `paintEndpoint`), so a burst of refusals would add a presenter round-trip **per keystroke** — raising queue pressure exactly when refusals cluster, on a path where `ErrBackpressure` stays fatal by deliberate choice. The correctness reason is in Task 3, Step 3b.

### ARCH-SECURE

Input is already parsed into typed `uv.Event` values by `terminal.Decoder` at the boundary before the router sees it; this change adds no new parsing and no new external input. No credentials touched. The one widening to check is the error text: `noDestination` embeds `View.Selected`/`View.Admitted`, which are endpoint ids couch minted (`couch-pty-N`), not operator data — safe to surface.

### ARCH-FUNERAL

Creates nothing durable: no file, row, cache, handle or background process. The sentinel is a package-level value with process lifetime; every other value dies with its event.

### ARCH-PURPOSE

The finding named one instance (couchtty's input arm). The class is **"a caller that escalates a presenter routing answer to a fatal failure"**, and it has two enumerations, because the answer reaches callers two ways.

*By input.* `grep -rn '\.Input(' cmd --include='*.go'` gives three families: couchtty (4 sites, Tasks 3–4), `termcmd/presentation.go:206` (Task 5, same escalation via `stopLocked`), and `terminalqualify/presenter_cases.go` (a qualification oracle that only tests `!= nil`, correct as-is and verified in Task 5).

*By repaint.* The plan-quality gate found the enumeration above was the wrong shape — it enumerated the *call site* and missed the *answer*. `Presenter.UpdateChrome` returns the same "I hold no endpoint" refusal, and `Console.paintNow` escalates it identically, reachable with no input at all. Task 2 gives that site the typed answer and Task 3 Step 3b classifies it. Before starting, re-run the real enumeration — every refusal in `cmd/internal/terminal` that reports absence of an endpoint rather than a failed write — and confirm it is exactly these three: `Input`, `mouseInput`, `UpdateChrome`. A fourth would belong in Task 2.

Tasks 2, 3b and 5 exist so the class is swept in this round, not the next.

---

## Chunk 1: Tasks

### Task 1: The typed no-destination answer

**Files:**
- Create: `cmd/internal/terminal/destination.go`
- Test: `cmd/internal/terminal/destination_test.go`

- [ ] **Step 1: Write the failing test**

```go
package terminal

import (
	"errors"
	"strings"
	"testing"
)

func TestNoDestinationIsClassifiableAndNamesTheView(t *testing.T) {
	err := noDestination("no admitted endpoint", View{State: Presenting, Selected: "couch-pty-3"})
	if !errors.Is(err, ErrNoDestination) {
		t.Fatalf("errors.Is(%v, ErrNoDestination) = false", err)
	}
	for _, want := range []string{"no admitted endpoint", "couch-pty-3", "state=1"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %q", err, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/internal/terminal/ -run TestNoDestination -v`
Expected: FAIL — `undefined: noDestination`, `undefined: ErrNoDestination`

- [ ] **Step 3: Write minimal implementation**

```go
package terminal

import (
	"errors"
	"fmt"
)

// ErrNoDestination is the presenter's answer when it holds no endpoint to
// deliver an input event to.
//
// It is a ROUTING answer, not a terminal failure, and the difference is the
// whole point. The compositor's panel is a legitimate no-destination state:
// Panel sets selected to nil and leaves Admitted empty, which is the healthy
// shape of a console showing its own screen. A caller that cannot tell this
// from a failed write tears down terminal ownership over a question it merely
// asked at the wrong moment -- which is what took couch down in pair#265.
//
// Physical failure keeps its own separate channel: Presenter.fail latches the
// View into Failed and closes Failed(). Nothing here touches that.
var ErrNoDestination = errors.New("terminal: no destination for input")

// noDestination is the ONE constructor, so the two refusal sites cannot
// describe the same condition two different ways. The View travels with the
// error because "which endpoint, in which state" is what makes the condition
// diagnosable when a caller does choose to surface it.
func noDestination(reason string, v View) error {
	return fmt.Errorf("%w: %s (state=%d selected=%q admitted=%q)",
		ErrNoDestination, reason, v.State, v.Selected, v.Admitted)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/internal/terminal/ -run TestNoDestination -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/terminal/destination.go cmd/internal/terminal/destination_test.go
git commit -m "#265: terminal: a typed answer for input with no destination"
```

---

### Task 2: The presenter returns it from both refusal sites

**Files:**
- Modify: `cmd/internal/terminal/presenter.go:470` (`Input`), `cmd/internal/terminal/presenter.go:478` (`mouseInput`)
- Test: `cmd/internal/terminal/presenter_test.go`

- [ ] **Step 1: Write the failing test**

Append to `presenter_test.go`:

```go
func TestPresenterRefusesInputWithoutADestinationClassifiably(t *testing.T) {
	p := NewPresenter(ttyio.NewFake(), CouchAnyMotion)
	t.Cleanup(func() { _ = p.Release(context.Background()) })
	err := p.Input(context.Background(), uv.KeyPressEvent{Code: 'x', Text: "x"})
	if !errors.Is(err, ErrNoDestination) {
		t.Fatalf("Input with no endpoint = %v, want ErrNoDestination", err)
	}
}
```

`ttyio.NewFake()` is the package's writer double — the same one `presenterFixture` (`presenter_test.go:15-21`) builds. Reuse it rather than adding another (ARCH-DRY).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/internal/terminal/ -run TestPresenterRefusesInputWithoutADestinationClassifiably -v`
Expected: FAIL — the returned error is a bare `errors.New`, so `errors.Is` is false

- [ ] **Step 3: Write minimal implementation**

In `Presenter.Input`, replace:

```go
		if v.State != Ready || v.Admitted == "" || p.selected == nil {
			return errors.New("terminal: no admitted endpoint")
		}
```

with:

```go
		if v.State != Ready || v.Admitted == "" || p.selected == nil {
			return noDestination("no admitted endpoint", v)
		}
```

In `Presenter.mouseInput`, replace:

```go
	if v.State != Ready {
		return errors.New("terminal: no presented mouse destination")
	}
```

with:

```go
	if v.State != Ready {
		return noDestination("no presented mouse destination", v)
	}
```

In `Presenter.UpdateChrome` (`presenter.go:782`), replace:

```go
		if p.selected == nil {
			return errors.New("terminal: chrome requires selected endpoint")
		}
```

with:

```go
		if p.selected == nil {
			return noDestination("chrome requires selected endpoint", p.View())
		}
```

**This third site was not in the first draft and is the one that matters most.** It is the same answer — "I hold no endpoint for this" — and `paintNow` escalates it to `terminalError` exactly as the input path did. Task 3 Step 3b closes that half; without this line, that step has nothing to classify. Found by the plan-quality gate (PQ-1).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/internal/terminal/`
Expected: PASS (whole package). No existing test asserts any of the three messages — verified with `grep -rn "no admitted endpoint\|no presented mouse destination\|chrome requires selected endpoint" cmd --include='*.go'`, which matches only the production sites. Re-run that grep before editing, in case a test landed since.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/terminal/presenter.go cmd/internal/terminal/presenter_test.go
git commit -m "#265: terminal: every refusal site answers with ErrNoDestination"
```

---

### Task 3: One panel-aware door for every input

**Files:**
- Modify: `cmd/internal/couchtty/terminal_input.go`
- Test: `cmd/internal/couchtty/panel_input_routing_test.go` (create)

This is the task that stops the operator's crash. It makes two changes together — the door classifies the answer, and the panel check starts governing key release, focus and blur — because only the pair is observable. Splitting them was tried and rejected: on the panel `Presenter.Input` refuses *before* reaching `Endpoint.Send`, so "did the panel stop asking" writes nothing to the child either way and has no red oracle of its own (plan review, second pass, measured). The non-fatality test below is the red one; the actor-delivery test is what stops the fix from over-dropping.

- [ ] **Step 1: Write the failing test**

```go
package couchtty

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// pair#265: the panel is a legitimate no-admitted-endpoint state, so an event
// that only means something to a child must not take the console down.
func TestPanelInputWithNoChildDoesNotStopTheConsole(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event uv.Event
	}{
		{"key-release", uv.KeyReleaseEvent{Code: uv.KeySpace, Mod: uv.ModCtrl}},
		{"focus", uv.FocusEvent{}},
		{"blur", uv.BlurEvent{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			t.Cleanup(con.Stop)
			con.attachThreadActor("only", "actor", menuAddress("t"), "/w/t", "t", ptychild.NewFakeChild(nil))
			con.showMenu()

			con.routeInputEvent(terminal.InputEvent{Event: tc.event, Raw: []byte("\x1b[32;5:3u")})

			con.mu.Lock()
			failure := con.terminalFailure
			con.mu.Unlock()
			if failure != nil {
				t.Fatalf("panel input latched a terminal failure: %v", failure)
			}
			select {
			case <-con.stop:
				t.Fatal("panel input stopped the console")
			default:
			}
		})
	}
}
```

Then, in the same file, the delivery pair. The first is a characterization test — it passes today and must keep passing; the second is the oracle that goes red if the fix over-drops.

```go
// A real child that wants focus reports sets DECSET 1004; vt emits \x1b[I only
// then. Without this seed neither test below can observe anything, because
// nothing is written whether the event is delivered or not.
var focusReporting = []byte("\x1b[?1004h")

// The panel check governs every event kind that only means something to a child,
// not just the printable ones.
func TestPanelDropsChildOnlyEventsBeforeThePresenter(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	child := ptychild.NewFakeChild(focusReporting)
	con.attachThreadActor("only", "actor", menuAddress("t"), "/w/t", "t", child)
	con.showMenu()

	before := len(child.Writes())
	con.routeInputEvent(terminal.InputEvent{Event: uv.FocusEvent{}, Raw: []byte("\x1b[I")})
	if got := len(child.Writes()); got != before {
		t.Fatalf("panel focus event reached the child: %d writes, want %d", got, before)
	}
}

// The mirror, and the one that can actually fail: with an actor focused the same
// event must still reach its child. Without it, "drop everything" passes above.
func TestFocusedActorStillReceivesChildOnlyEvents(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	child := ptychild.NewFakeChild(focusReporting)
	con.attachThreadActor("only", "actor", menuAddress("t"), "/w/t", "t", child)
	con.switchTo("only", true, arrivalOrdinary)

	before := len(child.Writes())
	con.routeInputEvent(terminal.InputEvent{Event: uv.FocusEvent{}, Raw: []byte("\x1b[I")})
	if got := len(child.Writes()); got <= before {
		t.Fatalf("focused actor did not receive the focus event: %d writes, want more than %d", got, before)
	}
}
```

**The seed is load-bearing, not decoration.** `Endpoint.Send` calls `backend.Focus()` (`terminal/endpoint.go:314`), and `third_party/vt/focus.go:21-29` emits `\x1b[I` **only if the child has set DECSET 1004**. A bare `ptychild.NewFakeChild(nil)` has not, so `child.Writes()` stays at 0 even with a properly admitted endpoint: the positive test would fail and the negative would pass for the wrong reason. Measured both ways — seeded, writes go 0 → 1 with content `"\x1b[I"`; unseeded, 0 → 0.

**Do not expect `TestPanelDropsChildOnlyEventsBeforeThePresenter` to be red.** On the panel `Presenter.Input` refuses before reaching `Endpoint.Send`, so the child receives nothing with or without the panel check; it passes on HEAD and is here to stay green. The red oracles in this task are `TestPanelInputWithNoChildDoesNotStopTheConsole` (3/3) and — against an over-dropping fix — `TestFocusedActorStillReceivesChildOnlyEvents`.

Only `uv.FocusEvent` is exercised at the child here. `KeyReleaseEvent` would need its own seed (`\x1b[>3u`) for the same reason, and all three kinds are already covered for non-fatality by the first test.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/internal/couchtty/ -run TestPanelInputWithNoChildDoesNotStopTheConsole -v`
Expected: FAIL on all three subtests — `panel input latched a terminal failure: terminal: no destination for input: no admitted endpoint (state=0 selected="" admitted="")`

Run: `go test ./cmd/internal/couchtty/ -run 'TestPanelDropsChildOnlyEvents|TestFocusedActorStillReceives' -v`
Expected: both PASS already. Run the positive one first and watch it pass before trusting the negative one.

- [ ] **Step 3: Write minimal implementation**

Add to `terminal_input.go` (imports gain `errors` and `cmd/internal/terminal` — the latter is already imported):

```go
// deliverPresenterInput is the ONE door from the console to Presenter.Input.
//
// It exists because the two are separate state machines that can legitimately
// disagree: the console tracks Focus, the presenter tracks View, and "I have no
// endpoint for this" is a correct answer from the second, not a failure of it.
// terminalError means "terminal ownership is lost"; routing a domain answer
// into it is what exited couch on a keystroke (pair#265). Every call site goes
// through here, and TestConsoleReachesPresenterInputOnlyThroughItsDoor pins it.
func (c *Console) deliverPresenterInput(event uv.Event) error {
	err := c.presenter.Input(c.lifetime, event)
	if errors.Is(err, terminal.ErrNoDestination) {
		return err
	}
	c.terminalError(err)
	return nil
}

// deliverChildInput routes an event that only means something to a child.
//
// On the panel there is no child, and a key release, focus or blur has no panel
// meaning, so it is dropped.
//
// A no-destination answer with an actor focused is ignored too, deliberately.
// The obvious move is to surface it as a notice -- and that is a trap. setNotice
// is publishNotice (console.go:1945), documented as "push and paint are
// therefore one operation", so it repaints through paintNow, which hands ANY
// UpdateChrome error to terminalError. That is the very exit this issue exists
// to remove, reached by a longer path. See Step 3b for the window that makes it
// reachable.
func (c *Console) deliverChildInput(event uv.Event) {
	c.mu.Lock()
	panel := c.focus.IsPanel()
	c.mu.Unlock()
	if panel {
		return
	}
	_ = c.deliverPresenterInput(event)
}
```


The doc comment above names `TestConsoleReachesPresenterInputOnlyThroughItsDoor`, which does not exist until Task 4. That is a deliberate forward reference, not an oversight — the two tasks land together, and a comment naming the guard that enforces it is worth more than the ordering purity. If Task 4 is ever dropped, delete the sentence with it.

Then replace all four `c.terminalError(c.presenter.Input(c.lifetime, event.Event))` call sites:

- `terminal_input.go:34` — `routeInputEvent`'s `route` closure, the `else` arm → `c.deliverChildInput(event.Event)`. Routing it through the wrapper rather than the door gives this arm the same panel rule as every other child-bound event, in one place rather than two (ARCH-DRY). The closure captured `panel` earlier and `deliverChildInput` re-reads it; that is safe, but **not** because everything runs on one goroutine — it does not. `installObservedThreadActor` writes `c.focus` at `console.go:422` from the operation-queue goroutine (`console.go:576`), while `selectActorContext`'s write at `terminal.go:145` does funnel onto Run. It is safe because both the capture and the re-read hold `c.mu`, so there is no data race, and the only reachable disagreement — a flip to panel between them — turns a deliver into a drop, which is the panel's own semantics.
- `terminal_input.go:42` — the `case uv.KeyReleaseEvent, uv.FocusEvent, uv.BlurEvent:` arm → `c.deliverChildInput(event.Event)`, with this comment, which is the point of the whole issue:

```go
	// These three reach the console because couch ASKS for them: the first mode
	// delta writes \x1b[?1004h (focus reporting) and \x1b[>3u (kitty flags 1|2,
	// whose flag 2 is "report event types", i.e. key release) on the very first
	// paint -- panel or not. They carry no panel meaning, so the panel check
	// governs them exactly as it governs printable keys and paste (pair#265).
```
- `terminal_input.go:71` — `routeMouseEvent`'s release arm → `_ = c.deliverPresenterInput(event.Event)`
- `terminal_input.go:93` — `routeMouseEvent`'s tail call → `_ = c.deliverPresenterInput(event.Event)`

Mouse stays on the door, not the wrapper: release and motion must reach the presenter **even on the panel** so its gesture owner can cancel or clip them.

- [ ] **Step 3b: Close the repaint half of the same escalation**

`paintNow` (`console.go:1101-1121`) is the other caller that turns a no-destination answer into an exit, and it is reachable **without any input at all**:

```go
	row, cells, err := c.chrome()
	if err == nil {
		err = c.presenter.UpdateChrome(c.lifetime, cells)
	}
	if err != nil {
		c.terminalError(err)
		return
	}
```

The window is real and is this issue's own premise inverted. `showMenu` calls `presenter.Panel` — which sets `p.selected = nil` — at `console_menu.go:220`, and only *then* takes `c.mu` to set `c.focus = FocusPanel()` at `:225-228`. Between those two, `focus` says actor while the presenter holds nothing. Any repaint landing there reads `panel == false`, calls `UpdateChrome`, gets `chrome requires selected endpoint`, and exits couch. Change it to:

```go
	row, cells, err := c.chrome()
	if err == nil {
		err = c.presenter.UpdateChrome(c.lifetime, cells)
	}
	if errors.Is(err, terminal.ErrNoDestination) {
		// Mid-transition: Panel has cleared the endpoint and focus has not
		// caught up yet. The paint that follows the flip is the authoritative
		// one, so skipping this one loses nothing.
		return
	}
	if err != nil {
		c.terminalError(err)
		return
	}
```

This is what makes "non-fatal" true rather than aspirational, and it is why `deliverChildInput` can safely ignore a refusal instead of announcing it.

- [ ] **Step 3c: Cover the closed enumeration, not three instances**

`makeInputEvent` (`terminal/input.go:292-300`) emits a **closed** set: `KeyPressEvent`, `KeyReleaseEvent`, the four mouse kinds, `FocusEvent`, `BlurEvent`, `PasteEvent`, and `Reply` for everything else. Testing the three kinds the operator happened to hit is the same hand-enumeration this issue's own lesson warns about, and Task 4's AST guard pins the *door*, not the *panel rule* — a future arm calling `deliverPresenterInput` directly would pass it. So drive the whole set (ARCH-PURPOSE: the class, not the instances):

```go
// Every kind the decoder can produce, with the panel focused, must leave the
// console alive. The list is makeInputEvent's closed set -- when a kind is added
// there, this test is where it declares its panel rule.
func TestNoDecodedEventKindCanStopThePanelConsole(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event uv.Event
	}{
		{"key-press", uv.KeyPressEvent{Code: 'x', Text: "x"}},
		{"key-release", uv.KeyReleaseEvent{Code: uv.KeySpace, Mod: uv.ModCtrl}},
		{"mouse-click", uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}},
		{"mouse-release", uv.MouseReleaseEvent{X: 1, Y: 1, Button: uv.MouseLeft}},
		{"mouse-motion", uv.MouseMotionEvent{X: 1, Y: 1}},
		{"mouse-wheel", uv.MouseWheelEvent{X: 1, Y: 1, Button: uv.MouseWheelUp}},
		{"focus", uv.FocusEvent{}},
		{"blur", uv.BlurEvent{}},
		{"paste", uv.PasteEvent{Content: "x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			t.Cleanup(con.Stop)
			con.attachThreadActor("only", "actor", menuAddress("t"), "/w/t", "t", ptychild.NewFakeChild(focusReporting))
			con.showMenu()

			con.routeInputEvent(terminal.InputEvent{Event: tc.event, Raw: []byte("x"), Canonical: []byte("x")})

			con.mu.Lock()
			failure := con.terminalFailure
			con.mu.Unlock()
			if failure != nil {
				t.Fatalf("panel %s latched a terminal failure: %v", tc.name, failure)
			}
			select {
			case <-con.stop:
				t.Fatalf("panel %s stopped the console", tc.name)
			default:
			}
		})
	}
}
```

Check the exact `uv` constructor names against `terminal/input_test.go` before pasting — that file already builds most of these kinds. If a mouse subtest needs a specific `Raw` to decode, take it from `mouseinput`'s tests rather than inventing one.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/internal/couchtty/ -run 'TestPanelInputWithNoChild|TestPanelDropsChildOnlyEvents|TestFocusedActorStillReceives|TestNoDecodedEventKindCanStop' -v`
Expected: PASS — 3 subtests, both delivery tests, and 9 enumeration subtests

Run: `go test ./cmd/internal/couchtty/`
Expected: PASS (whole package)

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/couchtty/terminal_input.go cmd/internal/couchtty/console.go cmd/internal/couchtty/panel_input_routing_test.go
git commit -m "#265: couch: a routing answer is not a terminal failure"
```

---

### Task 4: Pin the door

**Files:**
- Create: `cmd/internal/couchtty/input_door_guard_test.go`

The regression was one call site written without the panel check. A comment would not have stopped it; the package already solves this shape mechanically. Model this test on `TestMenuCodeReadsTheInventoryOnlyThroughTheViewedLookups` in `menu_inventory_guard_test.go` — couchtty's only guard test — and read it first. (`artifactpath/deadsymbols_test.go` is the same idiom but scoped to `couchcore`, so it does not see couchtty; that also means Task 3's intermediate state, with `deliverChildInput` defined and not yet called, trips no guard.)

- [ ] **Step 1: Write the failing test**

```go
package couchtty

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Console code reaches Presenter.Input ONLY through its door (pair#265).
//
// The crash this pins was one call site, added with the #255 migration, that
// forwarded key release, focus and blur straight to the presenter without the
// panel check every neighbouring arm honours. The presenter answered correctly
// -- the panel holds no endpoint -- and the console turned that answer into a
// fatal exit. A second such site is a one-line edit away, so the rule is a test
// rather than a comment.
var presenterInputCallersAllowed = map[string]bool{
	"deliverPresenterInput": true, // the door: classifies the answer
}

func TestConsoleReachesPresenterInputOnlyThroughItsDoor(t *testing.T) {
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var violations []string
	fset := token.NewFileSet()
	for _, path := range sources {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, path, raw, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || presenterInputCallersAllowed[fn.Name.Name] {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Input" {
					return true
				}
				inner, ok := selector.X.(*ast.SelectorExpr)
				if !ok || inner.Sel.Name != "presenter" {
					return true
				}
				violations = append(violations, fset.Position(call.Pos()).String()+" in "+fn.Name.Name)
				return true
			})
		}
	}
	if len(violations) > 0 {
		t.Fatalf("Presenter.Input called outside its door:\n  %s", strings.Join(violations, "\n  "))
	}
}
```

- [ ] **Step 2: Run it against a deliberate violation to prove the oracle**

A guard test that has never been red reports nothing. Temporarily add `_ = c.presenter.Input(c.lifetime, nil)` inside `routeMouseEvent`, run the test, confirm it FAILs and names `routeMouseEvent`, then remove the line.

Run: `go test ./cmd/internal/couchtty/ -run TestConsoleReachesPresenterInputOnlyThroughItsDoor -v`
Expected with the violation present: FAIL naming `routeMouseEvent`. Expected after removing it: PASS.

- [ ] **Step 3: Run the package**

Run: `go test ./cmd/internal/couchtty/`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/internal/couchtty/input_door_guard_test.go
git commit -m "#265: couch: pin Presenter.Input to its one door"
```

---

### Task 5: Sweep the class — `pair term` has the same escalation

**Files:**
- Modify: `cmd/internal/termcmd/presentation.go:206`
- Test: `cmd/internal/termcmd/presentation_test.go` (or the package's existing mux test file — read first and follow it)

`termcmd/presentation.go:206` is the same shape: `if err := m.presenter.Input(...); err != nil { m.stopLocked(err); return }`. It guards on `m.activeTabLocked() == nil`, which is `pair term`'s own tab model — the same two-models-can-disagree problem, so a tab that exists while the presenter holds no admitted endpoint kills `pair term`. Fixing only couch would be the instance, not the class (ARCH-PURPOSE).

- [ ] **Step 1: Confirm the enumeration is complete**

Run: `grep -rn '\.Input(' cmd --include='*.go' | grep -v _test`
Expected: three families only — `couchtty` (now one door), `termcmd/presentation.go:206`, and `terminalqualify/presenter_cases.go`. Confirm `terminalqualify` needs no change: its sites either run against an admitted endpoint or record `p.Input(...) != nil` as an oracle, which stays true.

- [ ] **Step 2: Write the failing test**

A mux whose presenter has no admitted endpoint must not latch a failure.

Construction matters: do **not** use `addPresentationTab` (`presentation_test.go:29`), which calls `m.admitTab` and therefore admits. This test needs a tab present while the presenter holds nothing, so build the struct directly in the style of `passthrough_test.go:25`:

```go
func TestRoutingAnswerDoesNotStopTheMux(t *testing.T) {
	p := terminal.NewPresenter(ttyio.NewFake(), terminal.ChildRequested)
	t.Cleanup(func() { _ = p.Release(context.Background()) })
	tab := &terminalTab{id: 1, child: ptychild.NewFakeChild(nil)}
	// Deliberately NOT admitted: a tab exists while the presenter holds nothing,
	// which is the disagreement this test is about.
	m := &terminalMux{tabs: []*terminalTab{tab}, active: 0, presenter: p, done: make(chan struct{})}
	m.writeEvents([]terminal.InputEvent{{Event: uv.KeyPressEvent{Code: 'x', Text: "x"}}})
	if m.failure != nil {
		t.Fatalf("a routing answer latched a mux failure: %v", m.failure)
	}
	select {
	case <-m.done:
		t.Fatal("a routing answer stopped the mux")
	default:
	}
}
```

`terminal.ChildRequested` matches `newTerminalMux` (`presentation.go:69`). Do **not** use `addPresentationTab` (`presentation_test.go:29`) — it calls `m.admitTab`, which admits, and this test needs the opposite.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./cmd/internal/termcmd/ -run TestRoutingAnswerDoesNotStopTheMux -v`
Expected: FAIL — `a routing answer latched a mux failure: terminal: no destination for input: no admitted endpoint (…)`

- [ ] **Step 4: Write minimal implementation**

```go
		if err := m.presenter.Input(context.Background(), event.Event); err != nil {
			// A routing answer is not a failure of the terminal: the presenter
			// simply holds no endpoint for this event right now. Stopping the
			// mux over it is the pair#265 escalation.
			if errors.Is(err, terminal.ErrNoDestination) {
				continue
			}
			m.stopLocked(err)
			return
		}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/internal/termcmd/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/internal/termcmd/
git commit -m "#265: term: a routing answer does not stop the mux"
```

---

### Task 6: Atlas and lessons

**Files:**
- Modify: `atlas/couch.md` — under its existing *Navigation* heading (there is no "input-routing" heading; check the current headings before editing rather than inventing one)
- Modify: `atlas/terminal.md` — `ErrNoDestination` is new **exported** surface in `cmd/internal/terminal` and belongs on that map too
- Modify: `workshop/lessons.md`
- Modify: `workshop/issues/000265-*.md`

- [ ] **Step 1: Atlas**

Record the door and why it exists: couch and the presenter are separate state machines; `Focus` decides *whether* there is a child, `View` decides *whether the presenter holds one*, and `ErrNoDestination` is how the second says "not applicable" without the first treating it as terminal loss. Note that couch requests focus reporting and kitty event types on its first paint, which is why these events exist at all.

- [ ] **Step 2: Lessons**

Append:

```markdown
- A cross-layer answer channel needs a third value. `Presenter.Input` could only
  answer "fine" or "error", and the console's `terminalError` could only hear
  "fine" or "fatal" -- so "I hold no endpoint for this", which is the panel's
  normal state, exited couch on a keystroke. When one component asks another
  about state the second owns, "not applicable" is an answer, not a failure;
  give it a type. couchcore already models this as `ProofStatus`
  (`ProofUnresolved` = "never asked, or asking failed"); couchtty did not.
  (#265, 2026-09-16)

- When a routing decision has an allowlist of event kinds, the kinds NOT in the
  list are the bug surface. `routeInputEvent` checked the panel for printable
  keys, ESC and paste, and forwarded key release, focus and blur straight
  through -- for events couch itself had asked the terminal to send. Enumerate
  the kinds the decoder can produce and state the rule for each; a `switch` with
  an unexamined arm is an unwritten rule. (#265, 2026-09-16)
```

- [ ] **Step 3: Issue log**

Record the outcome in `## Log` and tick `## Plan`.

- [ ] **Step 4: Full verification**

Per `workshop/lessons.md`, `make test` run from inside a live pair session needs the retention-owner group scrubbed and a real `TMPDIR`:

```bash
env -u PAIR_DATA_DIR -u PAIR_TAG -u PAIR_RETENTION_PROTOCOL \
    -u PAIR_RETENTION_BACKGROUND -u PAIR_RETENTION_START_ID \
    TMPDIR=/private/tmp/pair-test make test
```

Expected: PASS. `parley_harness_golden` 7/7 is a known pre-existing failure — confirm it is unchanged rather than treating it as a regression.

- [ ] **Step 5: Commit**

```bash
git add atlas/couch.md atlas/terminal.md workshop/lessons.md workshop/issues/000265-*.md
git commit -m "#265: atlas + lessons for the input door"
```

---

## Operator verification

The repro is a code test, but the reported symptom is physical. After Task 3, ask the operator to confirm in a real couch: open the switcher with `ctrl+space` and leave it open, click away to another window and back (focus/blur), and confirm couch stays up. The pre-`#255` behaviour is the baseline.

Per `memory: feedback_pair_dogfood_and_agnostic` — ASK the operator to smoke test before calling this done. Fixing the crash in tests is not the same as fixing it in the operator's terminal, and the last time that shortcut was taken it shipped broken (#209).

## Out of scope

- The blank-viewport/hang between couch's startup in `brain` and the crash, and the raw escape sequence reported at the top-left. Not yet attributed; needs one `COUCH_INPUT_TRACE`/`COUCH_TRACE` run. `pair` has no resumable thread either and couch works there, so the fresh-spawn path is not broken in general — which narrows this considerably. Issue 265's Spec and Done-when were written before the diagnosis and still claim this half; they are narrowed to the routing half in that issue's `## Revisions` (2026-09-16), which also records that this symptom needs its own issue once the trace attributes it. **Do not close 265 against the original wording.**
- The wedged park and the record-versus-world reconciliation: `pair#271`.
- Orphaned live agents (liveness proved from the launcher pid): `pair#272`.
- **`ErrBackpressure` is an adjacent sibling this issue does NOT fix.** `p.call` returns it (`presenter.go:91`) when the presenter's request queue is full; it is not `ErrNoDestination`, so it still reaches `terminalError` and exits couch. It belongs to a neighbouring class — a *capacity* answer rather than a *routing* answer — and dropping input under backpressure is almost certainly better than exiting, but that is a policy decision with its own ARCH-CONSTRAINTS argument and should not ride along here. Raised by the plan review's second pass; record it in the `## Log` at close so it is deferred rather than lost.
