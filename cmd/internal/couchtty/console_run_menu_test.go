package couchtty

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func liveMenuFixture(t *testing.T) *consoleFixture {
	t.Helper()
	f := newFixture(t, 24, 100)
	f.con.mu.Lock()
	root := f.con.panes[f.con.order[0]]
	address := root.thread
	root.process = couchcore.ProcessIdentity{PID: 42, Identity: "root-start"}
	f.con.menu.ActiveAddress = address
	f.con.mu.Unlock()
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{{
			Address: address, WorkingPath: "/repo", Name: "root", State: couchcore.ThreadLive,
			LastActiveAt: time.Now(),
		}}, nil
	})
	waitUpTo(t, 250*time.Millisecond, "live menu inventory", func() bool {
		return f.con.menuSnapshot().InventoryReady && len(f.con.menuSnapshot().Inventory) == 1
	})
	return f
}

func TestConsoleRunHierarchicalMenuControls(t *testing.T) {
	f := liveMenuFixture(t)
	f.host.Reset()
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "hierarchical root menu", func() bool {
		written := f.host.Written()
		return strings.Contains(written, "threads") && strings.Contains(written, "root")
	})
	if strings.Contains(f.host.Written(), "couch — actors") {
		t.Fatalf("live Console rendered compatibility panel: %q", f.host.Written())
	}

	f.host.Reset()
	_, _ = f.stdin.Write([]byte{'\t'})
	waitUpTo(t, 250*time.Millisecond, "thread action frame", func() bool {
		screen := lastConsoleScreen(f.host.Written())
		return strings.Contains(screen, "threads › root › actions") && strings.Contains(screen, "rename")
	})
	if screen := lastConsoleScreen(f.host.Written()); strings.Contains(screen, "/repo") {
		t.Fatalf("action surface retained the root body: %q", f.host.Written())
	}

	f.host.Reset()
	_, _ = f.stdin.Write([]byte("\x1b\x1b"))
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "start form", func() bool {
		return strings.Contains(f.host.Written(), "start thread") && strings.Contains(f.host.Written(), "path")
	})
	if screen := lastConsoleScreen(f.host.Written()); strings.Contains(screen, "threads") || strings.Contains(screen, "root") {
		t.Fatalf("global start rendered a false parent: %q", f.host.Written())
	}
}

func TestConsoleRunStartPathOwnsCursorOnHighlightedPathRow(t *testing.T) {
	f := liveMenuFixture(t)
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "hierarchical root menu", func() bool {
		return strings.Contains(f.host.Written(), "threads")
	})

	f.host.Reset()
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "start path cursor", func() bool {
		written := f.host.Written()
		return strings.Contains(written, "▸ path") && strings.Contains(written, hostty.MoveTo(3, 9)+hostty.ShowCursor)
	})
	written := f.host.Written()
	hide := strings.Index(written, "\x1b[?25l")
	clear := strings.Index(written, hostty.HomeAndClear)
	if hide < 0 || clear < 0 || hide > clear {
		t.Fatalf("switcher did not hide inherited cursor before takeover: %q", written)
	}
	if strings.Contains(written, hostty.MoveTo(1, 1)+hostty.ShowCursor) {
		t.Fatalf("cursor landed on start title instead of path row: %q", written)
	}
}

func TestConsoleRunStartPathCompletionNavigatesFakeFilesystem(t *testing.T) {
	f := liveMenuFixture(t)
	reader := &fakeDirectoryBatchReader{
		started: make(chan string, 1),
		release: make(chan struct{}),
		entries: map[string][]CompletionEntry{".": {
			{Name: "src", Directory: true}, {Name: "sample", Directory: true}, {Name: "notes.txt"},
		}},
	}
	f.con.SetDirectoryBatchReader(reader)
	close(reader.release)

	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "root menu", func() bool {
		return strings.Contains(lastConsoleScreen(f.host.Written()), "threads")
	})
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "start form", func() bool {
		return strings.Contains(lastConsoleScreen(f.host.Written()), "start thread")
	})
	f.host.Reset()
	_, _ = f.stdin.Write([]byte("s\t"))
	waitUpTo(t, 250*time.Millisecond, "directory candidates", func() bool {
		screen := lastConsoleScreen(f.host.Written())
		return strings.Contains(screen, "sample/") && strings.Contains(screen, "src/") && !strings.Contains(screen, "notes.txt")
	})
	select {
	case directory := <-reader.started:
		if directory != "." {
			t.Fatalf("completion base = %q, want Couch cwd", directory)
		}
	default:
		t.Fatal("completion did not reach filesystem seam")
	}
	_, _ = f.stdin.Write([]byte("\t\r"))
	waitUpTo(t, 250*time.Millisecond, "accepted cycled candidate", func() bool {
		return f.con.menuSnapshot().CurrentFrame().Path == "src/"
	})
}

func TestConsoleRunMenuOwnsInputAndBackgroundPainting(t *testing.T) {
	f := liveMenuFixture(t)
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "root menu", func() bool {
		return strings.Contains(lastConsoleScreen(f.host.Written()), "threads")
	})
	f.host.Reset()
	before := len(f.child.Writes())

	f.child.Feed([]byte("background output must stay hidden"))
	f.child.Feed([]byte("\x1b[2J"))
	waitUpTo(t, 250*time.Millisecond, "background output drain", func() bool {
		return f.con.PaneRowDirty("c1")
	})
	_, _ = f.stdin.Write([]byte("\x1b[<0;12;4M\x1b[<0;13;4Mz"))
	waitUpTo(t, 250*time.Millisecond, "local filter input", func() bool {
		return f.con.menuSnapshot().CurrentFrame().Filter == "z"
	})
	if strings.Contains(f.host.Written(), "background output must stay hidden") {
		t.Fatal("background actor painted over the switcher")
	}
	if got := len(f.child.Writes()); got != before {
		t.Fatalf("switcher input reached actor: writes %d -> %d", before, got)
	}
}

func TestConsoleRunRootEscapeClearsFilterThenReplaysActor(t *testing.T) {
	f := liveMenuFixture(t)
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "root menu", func() bool {
		return strings.Contains(lastConsoleScreen(f.host.Written()), "threads")
	})
	_, _ = f.stdin.Write([]byte("zz\x1b"))
	waitUpTo(t, 250*time.Millisecond, "root filter clear", func() bool {
		state := f.con.menuSnapshot()
		f.con.mu.Lock()
		focus := f.con.focus
		f.con.mu.Unlock()
		return state.CurrentFrame().Filter == "" && focus.IsPanel()
	})

	f.host.Reset()
	f.child.Feed([]byte("progress while switcher was open"))
	_, _ = f.stdin.Write([]byte("\x1b"))
	waitUpTo(t, 250*time.Millisecond, "actor replay", func() bool {
		f.con.mu.Lock()
		focus := f.con.focus
		f.con.mu.Unlock()
		return !focus.IsPanel() && strings.Contains(f.host.Written(), "progress while switcher was open")
	})
	if !strings.Contains(f.host.Written(), hostty.HomeAndClear) {
		t.Fatal("return from switcher skipped clear-and-replay")
	}
}

func lastConsoleScreen(written string) string {
	if index := strings.LastIndex(written, hostty.HomeAndClear); index >= 0 {
		return written[index+len(hostty.HomeAndClear):]
	}
	return written
}

func TestConsoleRunHorizontalArrowsNavigateSingleSurfaceHierarchy(t *testing.T) {
	f := liveMenuFixture(t)
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "thread root", func() bool {
		return strings.Contains(f.host.Written(), "threads")
	})

	f.host.Reset()
	_, _ = f.stdin.Write([]byte("\x1b[C"))
	waitUpTo(t, 250*time.Millisecond, "Right to actions", func() bool {
		return strings.Contains(f.host.Written(), "threads › root › actions")
	})
	f.host.Reset()
	_, _ = f.stdin.Write([]byte("\x1b[D"))
	waitUpTo(t, 250*time.Millisecond, "Left to root", func() bool {
		return strings.Contains(f.host.Written(), "root") && !strings.Contains(f.host.Written(), "actions")
	})

	f.host.Reset()
	// The action list is detach, relaunch, park, … so park is two Downs from
	// the top before descending into its confirmation.
	_, _ = f.stdin.Write([]byte("\x1bOC\x1b[B\x1b[B\x1bOC"))
	waitUpTo(t, 250*time.Millisecond, "SS3 Right to park confirmation", func() bool {
		return strings.Contains(f.host.Written(), "threads › root › park") && strings.Contains(f.host.Written(), "cancel")
	})
}

// Alt+x quits what you are looking at. On couch's own panel that is couch
// itself, which is how `leave` is reachable now that #170 deleted the root
// actor it used to hang off. Ctrl-space first, so the panel has focus.
func TestConsoleRunMenuAltXOnThePanelOpensLeaveConfirmation(t *testing.T) {
	f := liveMenuFixture(t)
	f.host.Reset()
	_, _ = f.stdin.Write([]byte("\x00\x1bx"))
	waitUpTo(t, 250*time.Millisecond, "leave confirmation", func() bool {
		screen := lastConsoleScreen(f.host.Written())
		return strings.Contains(screen, "threads › leave couch") && strings.Contains(screen, "cancel") &&
			// The destructive whole-couch action names its cost where the
			// operator actually reads it -- on the item, since the frame title
			// is overwritten by the breadcrumb.
			strings.Contains(screen, "leave couch, parking 1 live thread")
	})
	if strings.Contains(f.host.Written(), "type yes") {
		t.Fatalf("Alt+x fell back to compatibility prompt: %q", f.host.Written())
	}
}

func TestConsoleRunConfirmedLeaveCarriesItsDispositionAndExits(t *testing.T) {
	f := liveMenuFixture(t)
	called := make(chan struct{}, 1)
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		if name != "leave" {
			t.Fatalf("operation = %q, want leave", name)
		}
		// Alt+x is the park disposition. Its ONLY argument is that choice:
		// leaving addresses every thread, so it names none of them.
		if len(args) != 1 || args["mode"] != string(couchcore.LeavePark) {
			t.Fatalf("leave args = %+v, want only mode=park", args)
		}
		called <- struct{}{}
		return nil, nil
	})

	_, _ = f.stdin.Write([]byte("\x00\x1bx\x1b[B\r"))
	select {
	case <-called:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("confirmed leave was not dispatched")
	}
	select {
	case <-f.done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("successful leave did not exit Console.Run")
	}
}

func TestConsoleRunPaintsAndAnimatesProgressWhileOperationBlocks(t *testing.T) {
	f := liveMenuFixture(t)
	started := make(chan string, 1)
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		started <- lastConsoleScreen(f.host.Written())
		<-call.Context.Done()
		return nil, call.Context.Err()
	})

	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "root menu", func() bool {
		return strings.Contains(lastConsoleScreen(f.host.Written()), "threads")
	})
	f.host.Reset()
	_, _ = f.stdin.Write([]byte("\x1b[C\x1b[B\x1b[B\x1b[C\x1b[B\r"))
	select {
	case screen := <-started:
		if !strings.Contains(screen, "◐ parking root…") {
			t.Fatalf("dispatcher started before phase-zero progress paint: %q", screen)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("park operation did not start")
	}
	waitUpTo(t, 500*time.Millisecond, "animated operation progress", func() bool {
		screen := lastConsoleScreen(f.host.Written())
		return strings.Contains(screen, "◓ parking root…") || strings.Contains(screen, "◑ parking root…") || strings.Contains(screen, "◒ parking root…")
	})

	_, _ = f.stdin.Write([]byte("x"))
	waitUpTo(t, 250*time.Millisecond, "responsive navigation during operation", func() bool {
		screen := lastConsoleScreen(f.host.Written())
		return strings.Contains(screen, "filter: x") && strings.Contains(screen, "parking root…")
	})
}

// twoThreadMenuFixture is a live console hosting two actors, both projected as
// actionable, so a menu switch resolves to a real pane through the production
// dispatcher.
func twoThreadMenuFixture(t *testing.T) (*consoleFixture, couchcore.ThreadAddress, couchcore.ThreadAddress) {
	t.Helper()
	f := newFixture(t, 24, 100)
	second := ptychild.NewFakeChild([]byte("second screen"))
	second.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver("c2", batch) })
	f.con.Attach("c2", "worker", second)

	f.con.mu.Lock()
	one := f.con.panes["c1"].thread
	two := f.con.panes["c2"].thread
	f.con.panes["c1"].process = couchcore.ProcessIdentity{PID: 41, Identity: "one-start"}
	f.con.panes["c2"].process = couchcore.ProcessIdentity{PID: 42, Identity: "two-start"}
	f.con.menu.ActiveAddress = one
	f.con.mu.Unlock()

	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return []couchcore.ActionableThreadSummary{
			{Address: one, WorkingPath: "/repo/one", Name: "one", State: couchcore.ThreadLive, LastActiveAt: time.Now()},
			{Address: two, WorkingPath: "/repo/two", Name: "two", State: couchcore.ThreadLive, LastActiveAt: time.Now()},
		}, nil
	})
	waitUpTo(t, 250*time.Millisecond, "two-thread inventory", func() bool {
		return f.con.menuSnapshot().InventoryReady && len(f.con.menuSnapshot().Inventory) == 2
	})
	return f, one, two
}

// The headline behavior, driven end to end through the PRODUCTION input path:
// raw bytes into stdin, the real interceptor, the real dispatcher, the real
// arrival derivation. Every other test hand-feeds `arrival` into switchTo, so
// inverting the derivation at console.go's "switch" arm would ship the behavior
// backwards with a green suite -- and nothing proved HitPrevious reaches
// onPreviousHotkey rather than Run's default arm.
func TestConsoleRunNotificationHopThenPreviousReturnsHome(t *testing.T) {
	f, one, two := twoThreadMenuFixture(t)

	// Working in one.
	f.con.switchTo("c1", true, arrivalOrdinary)
	// two pages the operator.
	f.con.mu.Lock()
	f.con.attention.Mark(two, "review ready")
	f.con.syncAttentionLocked()
	f.con.mu.Unlock()

	// ctrl-space opens the switcher focused on the paged actor; Return switches.
	_, _ = f.stdin.Write([]byte("\x00"))
	waitUpTo(t, 250*time.Millisecond, "switcher focused on the paged actor", func() bool {
		return f.con.menuSnapshot().CurrentFrame().SelectedAddress == two
	})
	_, _ = f.stdin.Write([]byte("\r"))
	waitUpTo(t, time.Second, "the notification hop to land", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.active == "c2"
	})

	// ctrl+backspace must return to one -- which only holds if the arrival was
	// derived as a notification hop, so it never became `previous`.
	_, _ = f.stdin.Write([]byte("\x08"))
	waitUpTo(t, time.Second, "ctrl+backspace to return home", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.active == "c1"
	})

	f.con.mu.Lock()
	previous, ok := f.con.tracker.Previous()
	f.con.mu.Unlock()
	if !ok || previous != one {
		t.Fatalf("previous = (%+v, %v), want %+v -- home must stay pinned", previous, ok, one)
	}
}

// The negative twin: switching to an UNPAGED row is an ordinary arrival, so it
// does spend the previous slot. Without this, a derivation that always returned
// arrivalNotification would pass the test above.
func TestConsoleRunOrdinarySwitchAdvancesPrevious(t *testing.T) {
	f, one, two := twoThreadMenuFixture(t)
	f.con.switchTo("c1", true, arrivalOrdinary)

	// No notification on two: selecting it is an ordinary switch.
	_, _ = f.stdin.Write([]byte("\x00"))
	waitUpTo(t, 250*time.Millisecond, "the switcher", func() bool {
		return len(f.con.menuSnapshot().Frames) > 0
	})
	f.con.mu.Lock()
	f.con.menu.Frames[0].SelectedAddress = two
	f.con.mu.Unlock()
	_, _ = f.stdin.Write([]byte("\r"))
	waitUpTo(t, time.Second, "the ordinary switch to land", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.active == "c2"
	})

	f.con.mu.Lock()
	previous, ok := f.con.tracker.Previous()
	f.con.mu.Unlock()
	if !ok || previous != one {
		t.Fatalf("previous = (%+v, %v), want %+v -- an ordinary switch pins what it left",
			previous, ok, one)
	}
}

// alt+d dispatches detach for the attached thread, with NO confirmation --
// unlike alt+x, which confirms because park destroys the agent. Driven through
// the production input path, because a reducer that supports the operation
// proves nothing about the key reaching it.
func TestConsoleRunAltDDetachesWithoutConfirmation(t *testing.T) {
	f, _, _ := twoThreadMenuFixture(t)
	f.con.switchTo("c1", true, arrivalOrdinary)

	dispatched := make(chan string, 4)
	setTestOps(f.con, func(name string, _ map[string]string) (any, error) {
		dispatched <- name
		return nil, nil
	})

	for _, encoding := range workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltD) {
		_, _ = f.stdin.Write(encoding)
	}

	select {
	case name := <-dispatched:
		if name != "detach" {
			t.Fatalf("alt+d dispatched %q, want detach", name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("alt+d dispatched nothing")
	}
	if screen := lastConsoleScreen(f.host.Written()); strings.Contains(screen, "cancel") {
		t.Fatalf("alt+d rendered a confirmation: %q", screen)
	}
}

// The switcher IS couch, so Alt+d there applies to every live thread and then
// leaves -- with no confirmation, because detach destroys nothing at either
// scope. This is the operator's way out, and it used to be a refusal notice.
func TestConsoleRunAltDOnThePanelDetachesEveryThreadAndLeaves(t *testing.T) {
	f, _, _ := twoThreadMenuFixture(t)
	dispatched := make(chan map[string]string, 4)
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		if name == "leave" {
			dispatched <- args
		}
		return nil, nil
	})
	_, _ = f.stdin.Write([]byte("\x00"))
	waitUpTo(t, 250*time.Millisecond, "the switcher", func() bool {
		return strings.Contains(f.host.Written(), "threads")
	})

	f.host.Reset()
	_, _ = f.stdin.Write([]byte("\x1b[100;3u"))

	select {
	case args := <-dispatched:
		if len(args) != 1 || args["mode"] != string(couchcore.LeaveDetach) {
			t.Fatalf("leave args = %+v, want only mode=detach", args)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("alt+d in the switcher dispatched nothing; screen: %q", lastConsoleScreen(f.host.Written()))
	}
	if screen := lastConsoleScreen(f.host.Written()); strings.Contains(screen, "cancel") {
		t.Fatalf("the safe whole-couch gesture asked for confirmation: %q", screen)
	}
	select {
	case <-f.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("leave did not exit Console.Run")
	}
}

// The bug this grid replaced, pinned end to end: the last live actor is gone,
// the operator is sitting in the switcher, and the safe key still gets them out.
// Nothing here is live, so a whole-couch action with nothing to act on must
// still leave -- that conditionality was the trap.
func TestConsoleRunLeavesFromASwitcherWithNothingLive(t *testing.T) {
	f := newFixture(t, 24, 100)
	f.con.mu.Lock()
	root := f.con.panes[f.con.order[0]]
	address := root.thread
	root.process = couchcore.ProcessIdentity{PID: 42, Identity: "root-start"}
	f.con.menu.ActiveAddress = address
	f.con.mu.Unlock()

	var live atomic.Bool
	live.Store(true)
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		state := couchcore.ThreadDetached
		if live.Load() {
			state = couchcore.ThreadLive
		}
		return []couchcore.ActionableThreadSummary{{
			Address: address, WorkingPath: "/repo", Name: "root", State: state, LastActiveAt: time.Now(),
		}}, nil
	})
	waitUpTo(t, 250*time.Millisecond, "live inventory", func() bool {
		return f.con.menuSnapshot().InventoryReady && len(f.con.menuSnapshot().Inventory) == 1
	})

	// Open the switcher, then lose the last live actor behind it.
	_, _ = f.stdin.Write([]byte{0})
	waitUpTo(t, 250*time.Millisecond, "the switcher", func() bool {
		return strings.Contains(lastConsoleScreen(f.host.Written()), "threads")
	})
	live.Store(false)
	f.child.Exit(0)
	waitUpTo(t, 500*time.Millisecond, "the detached row", func() bool {
		snapshot := f.con.menuSnapshot()
		return len(snapshot.Inventory) == 1 && !snapshot.Inventory[0].Live()
	})
	select {
	case code := <-f.done:
		t.Fatalf("console exited on its own with %d", code)
	case <-time.After(100 * time.Millisecond):
	}

	dispatched := make(chan string, 4)
	setTestOps(f.con, func(name string, _ map[string]string) (any, error) {
		dispatched <- name
		return nil, nil
	})
	_, _ = f.stdin.Write([]byte("\x1b[100;3u"))
	select {
	case name := <-dispatched:
		if name != "leave" {
			t.Fatalf("alt+d in an empty switcher dispatched %q, want leave", name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("alt+d in an empty switcher dispatched nothing; screen: %q", lastConsoleScreen(f.host.Written()))
	}
	select {
	case <-f.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("leave did not exit Console.Run")
	}
}

// Detaching an ACTOR lands in the switcher rather than ending couch. The
// console exits with its final child only when an actor still owns the focus,
// so without this the safe gesture would quit couch out from under an operator
// who detached their last thread.
func TestConsoleRunAltDOnTheLastActorLandsInTheSwitcher(t *testing.T) {
	f := liveMenuFixture(t)
	dispatched := make(chan string, 4)
	setTestOps(f.con, func(name string, _ map[string]string) (any, error) {
		dispatched <- name
		if name == "detach" {
			f.child.Exit(0)
		}
		return nil, nil
	})

	_, _ = f.stdin.Write([]byte("\x1b[100;3u"))
	select {
	case name := <-dispatched:
		if name != "detach" {
			t.Fatalf("alt+d dispatched %q, want detach", name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("alt+d dispatched nothing")
	}
	waitUpTo(t, 500*time.Millisecond, "the switcher", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.focus.IsPanel()
	})
	select {
	case code := <-f.done:
		t.Fatalf("couch exited (%d) when its last actor detached", code)
	case <-time.After(200 * time.Millisecond):
	}
}
