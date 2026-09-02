package couchtty

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
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
	_, _ = f.stdin.Write([]byte("\x1bOC\x1bOC"))
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
		return strings.Contains(screen, "threads › leave couch") && strings.Contains(screen, "cancel")
	})
	if strings.Contains(f.host.Written(), "type yes") {
		t.Fatalf("Alt+x fell back to compatibility prompt: %q", f.host.Written())
	}
}

func TestConsoleRunConfirmedLeaveUsesArgumentFreeOperationAndExits(t *testing.T) {
	f := liveMenuFixture(t)
	called := make(chan struct{}, 1)
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		if name != "leave" {
			t.Fatalf("operation = %q, want leave", name)
		}
		if len(args) != 0 {
			t.Fatalf("leave args = %+v, want none", args)
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
	_, _ = f.stdin.Write([]byte("\x1b[C\x1b[C\x1b[B\r"))
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
