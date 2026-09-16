package couchtty

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// A real child that wants focus reports sets DECSET 1004; vt emits \x1b[I only
// then. Without this seed the delivery tests below can observe nothing, because
// nothing is written whether the event is delivered or not.
var focusReporting = []byte("\x1b[?1004h")

func panelConsole(t *testing.T) (*Console, *ptychild.Child) {
	t.Helper()
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	child := ptychild.NewFakeChild(focusReporting)
	con.attachThreadActor("only", "actor", menuAddress("t"), "/w/t", "t", child)
	return con, child
}

func assertConsoleAlive(t *testing.T, con *Console, what string) {
	t.Helper()
	con.mu.Lock()
	failure := con.terminalFailure
	con.mu.Unlock()
	if failure != nil {
		t.Fatalf("%s latched a terminal failure: %v", what, failure)
	}
	select {
	case <-con.stop:
		t.Fatalf("%s stopped the console", what)
	default:
	}
}

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
			con, _ := panelConsole(t)
			con.showMenu()
			con.routeInputEvent(terminal.InputEvent{Event: tc.event, Raw: []byte("\x1b[32;5:3u")})
			assertConsoleAlive(t, con, "panel "+tc.name)
		})
	}
}

// The panel check governs every event kind that only means something to a
// child, not just the printable ones. Characterization: this passes before the
// fix too, because Presenter.Input refuses before reaching Endpoint.Send.
func TestPanelDropsChildOnlyEventsBeforeThePresenter(t *testing.T) {
	con, child := panelConsole(t)
	con.showMenu()
	before := len(child.Writes())
	con.routeInputEvent(terminal.InputEvent{Event: uv.FocusEvent{}, Raw: []byte("\x1b[I")})
	if got := len(child.Writes()); got != before {
		t.Fatalf("panel focus event reached the child: %d writes, want %d", got, before)
	}
}

// The mirror, and the one that can actually fail: with an actor focused the
// same event must still reach its child. Without it, "drop everything" passes
// the test above.
func TestFocusedActorStillReceivesChildOnlyEvents(t *testing.T) {
	con, child := panelConsole(t)
	con.switchTo("only", true, arrivalOrdinary)
	before := len(child.Writes())
	con.routeInputEvent(terminal.InputEvent{Event: uv.FocusEvent{}, Raw: []byte("\x1b[I")})
	if got := len(child.Writes()); got <= before {
		t.Fatalf("focused actor did not receive the focus event: %d writes, want more than %d", got, before)
	}
}

// Every kind the decoder can produce, with the panel focused, must leave the
// console alive. The list is makeInputEvent's closed set (terminal/input.go) --
// when a kind is added there, this test is where it declares its panel rule.
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
			con, _ := panelConsole(t)
			con.showMenu()
			con.routeInputEvent(terminal.InputEvent{Event: tc.event, Raw: []byte("x"), Canonical: []byte("x")})
			assertConsoleAlive(t, con, "panel "+tc.name)
		})
	}
}

// The durable disagreement: a foreground attach with no active pane sets
// c.focus to that actor WITHOUT selecting it (installObservedThreadActor), so
// the console believes an actor is focused while the presenter holds nothing.
// It persists until the operator switches. Every console path that talks to the
// presenter has to survive it (pair#265 BR-1, BR-3, BR-4).
func focusedButUnselected(t *testing.T) (*Console, *ptychild.Child) {
	t.Helper()
	con, child := panelConsole(t)
	con.mu.Lock()
	focus, selected := con.focus, con.active
	con.started = true
	con.mu.Unlock()
	if focus.IsPanel() || selected != "only" {
		t.Fatalf("fixture is not focused-but-unselected: focus=%v active=%q", focus, selected)
	}
	if v := con.presenter.View(); v.Admitted != "" {
		t.Fatalf("fixture presenter already holds an endpoint: %+v", v)
	}
	return con, child
}

// Red without the errors.Is block in paintNow: UpdateChrome refuses for want of
// an endpoint and the console exits.
func TestChromeRepaintWithNoEndpointDoesNotStopTheConsole(t *testing.T) {
	con, _ := focusedButUnselected(t)
	con.paintNow()
	assertConsoleAlive(t, con, "chrome repaint with no endpoint")
}

// Red without noDestination on resizeLayout: a terminal resize exits couch.
func TestResizeWithNoEndpointDoesNotStopTheConsole(t *testing.T) {
	con, _ := focusedButUnselected(t)
	con.onResize()
	assertConsoleAlive(t, con, "resize with no endpoint")
}

// Red if the pair#255 bypass is restored. The two arms are distinguishable at
// the trace: the panel check drops BEFORE the presenter is asked ("panel"),
// while a bypass reaches it and is classified there ("input"). Without this,
// deleting the panel check leaves every other assertion green -- the door is
// the AST guard's allowlisted callee either way.
func TestPanelDropsChildOnlyEventsWithoutAskingThePresenter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	con, _ := panelConsole(t)
	if err := con.SetEventTrace(path, time.Now()); err != nil {
		t.Fatal(err)
	}
	con.showMenu()

	con.routeInputEvent(terminal.InputEvent{Event: uv.FocusEvent{}, Raw: []byte("\x1b[I")})

	var dropped []string
	for _, line := range traceLines(t, path) {
		if strings.Contains(line, traceNoDestination) {
			dropped = append(dropped, line)
		}
	}
	if len(dropped) != 1 {
		t.Fatalf("want exactly one %s line, got %d: %q", traceNoDestination, len(dropped), dropped)
	}
	if !strings.Contains(dropped[0], "\tpanel") {
		t.Fatalf("panel drop was recorded as %q -- the presenter was asked, so the panel check is gone", dropped[0])
	}
}
