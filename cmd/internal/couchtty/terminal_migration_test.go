package couchtty

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Incomplete child output never owns the host parser. Chrome can update while
// the endpoint retains an unfinished sequence, and later cells remain correct.
func TestEndpointChromeDuringIncompleteChildSequence(t *testing.T) {
	host, child, con := newVTFixture(t, 8, 40)
	child.Feed([]byte("\x1b[38;2;"))
	con.setWarning("CHROME-DURING-PARTIAL")
	waitFor(t, "independent chrome", func() bool { return strings.Contains(host.row(8), AttentionGlyph+"1") })
	child.Feed([]byte("1;2;3mENDPOINT-COMPLETE"))
	waitFor(t, "completed endpoint", func() bool { return strings.Contains(host.childArea(), "ENDPOINT-COMPLETE") })
}

func TestEndpointChromeDoesNotUseChildCursorSave(t *testing.T) {
	for _, sequence := range []string{"\x1b7", "\x1b[?1049h\x1b[?1048h", "\x1b]title-without-terminator"} {
		t.Run(sequence, func(t *testing.T) {
			host, child, con := newVTFixture(t, 8, 40)
			child.Feed([]byte(sequence))
			con.setWarning("INDEPENDENT-STATUS")
			waitFor(t, "chrome without child cursor ownership", func() bool { return strings.Contains(host.row(8), AttentionGlyph+"1") })
		})
	}
}

// holdFirstPublish parks the first paint at site between its snapshot and its
// publication, and lets every other paint through. release lets it go on.
func holdFirstPublish(t *testing.T, con *Console, site string) (held <-chan struct{}, release func()) {
	t.Helper()
	parked, resume := make(chan struct{}), make(chan struct{})
	var first atomic.Bool
	con.mu.Lock()
	con.publishHook = func(at string) {
		if at == site && first.CompareAndSwap(false, true) {
			close(parked)
			<-resume
		}
	}
	con.mu.Unlock()
	var once sync.Once
	release = func() { once.Do(func() { close(resume) }) }
	t.Cleanup(release)
	return parked, release
}

func waitClosed(t *testing.T, what string, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-testwait.After(t):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// pair#484: a paint that snapshotted chrome before a warning was pushed must
// not publish after its badge update and leave stale chrome on screen.
func TestStaleChromeSnapshotCannotOverwriteAttentionBadge(t *testing.T) {
	host, _, con := newVTFixture(t, 8, 40)
	held, release := holdFirstPublish(t, con, "chrome")
	stale := make(chan struct{})
	go func() { defer close(stale); con.repaint() }()
	waitClosed(t, "the stale paint to snapshot", held)
	notice := make(chan struct{})
	go func() { defer close(notice); con.setWarning("AFTER-STALE-SNAPSHOT") }()
	// Observe publication's state commit while the older paint is held. This
	// forces the interleaving without assuming a goroutine ran after a sleep.
	waitFor(t, "notice committed behind held paint", func() bool { return con.menuSnapshot().LatestFeed == "AFTER-STALE-SNAPSHOT" })
	release()
	waitClosed(t, "the stale paint", stale)
	waitClosed(t, "the notice paint", notice)
	if got := host.row(8); !strings.Contains(got, AttentionGlyph+"1") {
		t.Fatalf("a stale status row replaced the warning badge: %q", got)
	}
}

// pair#484: the presenter's selection and the console's focus change as one
// step. A paint between them read the old (menu) focus and re-panelled over
// the endpoint just selected, leaving the menu up with focus on the actor.
func TestPaintDuringSelectionCannotRestoreTheMenu(t *testing.T) {
	host, child, con := newVTFixture(t, 8, 40)
	child.Feed([]byte("SELECTED-ENDPOINT"))
	waitFor(t, "endpoint content", func() bool { return strings.Contains(host.row(1), "SELECTED-ENDPOINT") })
	con.showMenu()
	waitFor(t, "the menu", func() bool { return !strings.Contains(host.row(1), "SELECTED-ENDPOINT") })
	held, release := holdFirstPublish(t, con, "select")
	selected := make(chan struct{})
	go func() {
		defer close(selected)
		if _, err := con.selectActor("c1", true, arrivalOrdinary); err != nil {
			t.Errorf("select: %v", err)
		}
	}()
	waitClosed(t, "the selection to publish", held)
	painted := make(chan struct{})
	go func() { defer close(painted); con.repaint() }()
	select {
	case <-painted:
	case <-time.After(50 * time.Millisecond):
	}
	release()
	waitClosed(t, "the selection", selected)
	waitClosed(t, "the repaint", painted)
	if got := host.row(1); !strings.Contains(got, "SELECTED-ENDPOINT") {
		t.Fatalf("the menu came back over the selected endpoint: %q", got)
	}
}

// pair#484: the switcher snapshot and publication share the publication owner.
// A notice concurrent with publication must appear in the final switcher banner,
// without being hidden by the older snapshot.
func TestNoticeDuringMenuPublicationIsShown(t *testing.T) {
	host, _, con := newVTFixture(t, 12, 40)
	con.mu.Lock()
	con.menu.Notice = MenuNotice{} // This parser fixture has no inventory provider.
	con.mu.Unlock()
	held, release := holdFirstPublish(t, con, "panel")
	menu := make(chan struct{})
	go func() { defer close(menu); con.showMenu() }()
	waitClosed(t, "the menu to snapshot", held)
	notice := make(chan struct{})
	go func() { defer close(notice); con.setNotice("NOTICE-UNDER-MENU") }()
	// Observe publication's state commit while the older paint is held. This
	// forces the interleaving without assuming a goroutine ran after a sleep.
	waitFor(t, "notice committed behind held paint", func() bool { return con.menuSnapshot().LatestFeed == "NOTICE-UNDER-MENU" })
	release()
	waitClosed(t, "the menu", menu)
	waitClosed(t, "the notice paint", notice)
	if got := host.childArea(); !strings.Contains(got, "NOTICE-UNDER-MENU") {
		t.Fatalf("the menu's older snapshot covered the notice: %q", got)
	}
}

// pair#484: the console's size and the presenter's geometry change as one
// step. A notice painted between them rendered a row of the new width against
// the old geometry, and the presenter's refusal ended couch.
func TestNoticeDuringResizeUsesOneGeometry(t *testing.T) {
	host, _, con := newVTFixture(t, 8, 40)
	held, release := holdFirstPublish(t, con, "resize")
	host.SetSize(ptychild.Size{Rows: 8, Cols: 50})
	waitClosed(t, "the resize to snapshot", held)
	notice := make(chan struct{})
	go func() { defer close(notice); con.setWarning("DURING-RESIZE") }()
	// Observe publication's state commit while the older paint is held. This
	// forces the interleaving without assuming a goroutine ran after a sleep.
	waitFor(t, "notice committed behind held paint", func() bool { return con.menuSnapshot().LatestFeed == "DURING-RESIZE" })
	release()
	waitClosed(t, "the notice paint", notice)
	con.mu.Lock()
	failure := con.terminalFailure
	con.mu.Unlock()
	if failure != nil {
		t.Fatalf("resize interleaving failed the terminal: %v", failure)
	}
	waitFor(t, "warning badge at the new width", func() bool { return strings.Contains(host.row(8), AttentionGlyph+"1") })
	if got := con.menuSnapshot().LatestFeed; got != "DURING-RESIZE" {
		t.Fatalf("resize warning lost retained text: %q", got)
	}
}

func TestEndpointSwitchRestoresFrameWithoutResizingChild(t *testing.T) {
	host, _, con := newVTFixture(t, 8, 40)
	incoming := newLaidOutFakeChild(t, con)
	incoming.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return con.Deliver(ctx, "c2", batch) })
	con.Attach("c2", "second", incoming)
	incoming.Feed([]byte("\x1b[?1049h\x1b[2J\x1b[1;1HRETAINED-ENDPOINT"))
	// Consume more than a capture ring without changing the cells. Restoration
	// must depend on the endpoint's frame, not retained raw startup output.
	for range 40 {
		incoming.Feed(bytes.Repeat([]byte("\x1b[0m"), 4096))
	}
	before := len(incoming.Resizes())
	con.Switch("c2")
	waitFor(t, "restored endpoint", func() bool { return strings.Contains(host.row(1), "RETAINED-ENDPOINT") })
	if got := len(incoming.Resizes()); got != before {
		t.Fatalf("switch resized child: before %d after %d", before, got)
	}
}

func TestConsoleConsumesHostResponsesBeforeOperatorPolicy(t *testing.T) {
	f := newFixture(t, 24, 80)
	waitFor(t, "initial frame", func() bool { return strings.Contains(f.host.Written(), "[brain]") })
	_, _ = f.stdin.Write([]byte("\x1b[?1;2c\x1b[12;34R\x1b]10;rgb:aaaa/bbbb/cccc\x1b\\typed"))
	waitFor(t, "operator suffix", func() bool { return strings.Contains(string(bytes.Join(f.child.Writes(), nil)), "typed") })
	if got := string(bytes.Join(f.child.Writes(), nil)); got != "typed" {
		t.Fatalf("host reply reached child: %q", got)
	}
}

func TestDeliverCancellationDoesNotWaitForStoppedConsumer(t *testing.T) {
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	con := New(host, bytes.NewReader(nil))
	defer con.Stop()
	defer con.release()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- con.Deliver(ctx, "not-started", ptychild.OutputBatch{}) }()
	waitFor(t, "queued delivery", func() bool { return len(con.chunks) == 1 })
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("delivery error %v", err)
		}
	case <-testwait.After(t):
		t.Fatal("canceled sink remained blocked")
	}
}

func TestEnhancedLifecycleKeysReachProductPolicy(t *testing.T) {
	for _, raw := range []string{"\x1b[100;3u", "\x1b[110;3u", "\x1b[110;3:2u"} {
		var decoder terminal.Decoder
		events, err := decoder.Feed([]byte(raw))
		if err != nil || len(events) != 1 {
			t.Fatalf("decode %q: %+v %v", raw, events, err)
		}
		var policy Interceptor
		_, hit, _ := policy.FeedHit(productKey(events[0]))
		if hit != HitDetach && hit != HitRelaunch {
			t.Fatalf("%q canonical=%q event=%#v hit=%v", raw, events[0].Canonical, events[0].Event, hit)
		}
	}
}

type joinedInputHost struct {
	*hostty.FakeHost
	reading       chan struct{}
	joined        chan struct{}
	releasedEarly chan struct{}
}

func (h *joinedInputHost) ReadContext(ctx context.Context, p []byte) (int, error) {
	close(h.reading)
	<-ctx.Done()
	close(h.joined)
	return 0, ctx.Err()
}
func (h *joinedInputHost) Read(p []byte) (int, error) { return h.ReadContext(context.Background(), p) }
func (h *joinedInputHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	if bytes.HasPrefix(p, []byte("\x18\x1b\\")) {
		select {
		case <-h.joined:
		default:
			close(h.releasedEarly)
		}
	}
	return h.FakeHost.WriteContext(ctx, p)
}
func TestConsoleJoinsContextualInputBeforeTerminalRelease(t *testing.T) {
	host := &joinedInputHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Cols: 80, Rows: 24}), reading: make(chan struct{}), joined: make(chan struct{}), releasedEarly: make(chan struct{})}
	con := New(host, host)
	child := ptychild.NewFakeChild(nil)
	defer child.Close()
	con.Attach("one", "one", child)
	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	select {
	case <-host.reading:
	case <-testwait.After(t):
		t.Fatal("reader never started")
	}
	con.Stop()
	select {
	case <-done:
	case <-testwait.After(t):
		t.Fatal("input did not join")
	}
	select {
	case <-host.releasedEarly:
		t.Fatal("release preceded input join")
	default:
	}
	if host.RawDepth() != 0 || !host.Closed() {
		t.Fatal("host was not restored")
	}
}

func TestConsoleSwitchCancelsDragBeforeNewActorInput(t *testing.T) {
	f := newFixture(t, 24, 80)
	b := newLaidOutFakeChild(t, f.con)
	b.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return f.con.Deliver(ctx, "c2", batch) })
	f.con.Attach("c2", "second", b)
	f.child.Feed([]byte("\x1b[?1002;1006h"))
	b.Feed([]byte("\x1b[?1002;1006h"))
	_, _ = f.stdin.Write([]byte("\x1b[<0;7;9M"))
	waitFor(t, "source press", func() bool { return string(bytes.Join(f.child.Writes(), nil)) == "\x1b[<0;7;9M" })
	f.con.Switch("c2")
	waitFor(t, "destination admission", func() bool { return f.con.presenter.View().Admitted == b.Endpoint().ID() })
	if got := string(bytes.Join(f.child.Writes(), nil)); got != "\x1b[<0;7;9M\x1b[<0;7;9m" {
		t.Fatalf("source cancellation %q", got)
	}
	_, _ = f.stdin.Write([]byte("\x1b[<32;8;9M\x1b[<0;8;9mx"))
	waitFor(t, "destination input barrier", func() bool { return strings.HasSuffix(string(bytes.Join(b.Writes(), nil)), "x") })
	if got := string(bytes.Join(b.Writes(), nil)); got != "x" {
		t.Fatalf("old gesture reached destination: %q", got)
	}
	_, _ = f.stdin.Write([]byte("\x1b[<0;2;3M\x1b[<0;2;3my"))
	waitFor(t, "fresh destination gesture", func() bool { return strings.HasSuffix(string(bytes.Join(b.Writes(), nil)), "y") })
	if got := string(bytes.Join(b.Writes(), nil)); got != "x\x1b[<0;2;3M\x1b[<0;2;3my" {
		t.Fatalf("fresh gesture was suppressed: %q", got)
	}
}

func TestConsolePanelResizeUpdatesEveryEndpoint(t *testing.T) {
	f := newFixture(t, 24, 80)
	b := newLaidOutFakeChild(t, f.con)
	f.con.Attach("c2", "second", b)
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "panel", func() bool { return f.con.presenter.View().Selected == "" })
	f.host.SetSize(ptychild.Size{Rows: 18, Cols: 60})
	for _, child := range []*ptychild.Child{f.child, b} {
		waitFor(t, "hidden geometry", func() bool { return child.Size() == (ptychild.Size{Rows: 17, Cols: 60}) })
		frame, err := child.Endpoint().Snapshot(time.Now())
		if err != nil || frame.Geometry != (terminal.Geometry{Rows: 17, Cols: 60}) {
			t.Fatalf("endpoint geometry %+v %v", frame.Geometry, err)
		}
	}
	f.con.Switch("c2")
	waitFor(t, "resized actor", func() bool { return f.con.presenter.View().Admitted == b.Endpoint().ID() })
	if !strings.Contains(f.screen.row(18), "[second]") {
		t.Fatalf("resized chrome: %q", f.screen.row(18))
	}
}
