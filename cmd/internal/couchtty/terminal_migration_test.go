package couchtty

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"strings"
	"testing"
	"time"
)

// Incomplete child output never owns the host parser. Chrome can update while
// the endpoint retains an unfinished sequence, and later cells remain correct.
func TestEndpointChromeDuringIncompleteChildSequence(t *testing.T) {
	host, child, con := newVTFixture(t, 8, 40)
	child.Feed([]byte("\x1b[38;2;"))
	con.setNotice("CHROME-DURING-PARTIAL")
	waitFor(t, "independent chrome", func() bool { return strings.Contains(host.row(8), "CHROME-DURING-PARTIAL") })
	child.Feed([]byte("1;2;3mENDPOINT-COMPLETE"))
	waitFor(t, "completed endpoint", func() bool { return strings.Contains(host.childArea(), "ENDPOINT-COMPLETE") })
}

func TestEndpointChromeDoesNotUseChildCursorSave(t *testing.T) {
	for _, sequence := range []string{"\x1b7", "\x1b[?1049h\x1b[?1048h", "\x1b]title-without-terminator"} {
		t.Run(sequence, func(t *testing.T) {
			host, child, con := newVTFixture(t, 8, 40)
			child.Feed([]byte(sequence))
			con.setNotice("INDEPENDENT-STATUS")
			waitFor(t, "chrome without child cursor ownership", func() bool { return strings.Contains(host.row(8), "INDEPENDENT-STATUS") })
		})
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
	case <-time.After(time.Second):
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
	case <-time.After(time.Second):
		t.Fatal("reader never started")
	}
	con.Stop()
	select {
	case <-done:
	case <-time.After(time.Second):
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
