package webcast

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// drivenBrowser is a FakeBrowser that can be driven: its "recorder" is a
// fake injector following the stream of what was emitted, opened when the
// test says (as the real one opens at its first go-live).
type drivenBrowser struct {
	*FakeBrowser
	inj    *drive.FakeInjector
	stream *recording.MemSink
	opened atomic.Bool
	hold   chan struct{} // while set, a batch waits on it

	mu      sync.Mutex
	driving []string
}

func newDrivenBrowser(t *testing.T) *drivenBrowser {
	stream := &recording.MemSink{}
	page := drive.DrivenPage(100)
	inj, err := drive.NewFakeInjector(page, stream)
	if err != nil {
		t.Fatal(err)
	}
	return &drivenBrowser{FakeBrowser: NewFakeBrowser(), inj: inj, stream: stream}
}

func (d *drivenBrowser) Opened() bool { return d.opened.Load() }

func (d *drivenBrowser) Emit(r recording.Record, doc int) error {
	r.SetTime(d.inj.Page().T + 1)
	d.stream.Put(r)
	return d.inj.Put(r)
}

func (d *drivenBrowser) WithTab(ctx context.Context, by string, fn func(drive.Tab) error) error {
	if d.hold != nil {
		<-d.hold
	}
	return d.inj.Tabs.WithTab(ctx, by, fn)
}

func (d *drivenBrowser) SetDriving(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.driving = append(d.driving, name)
	return nil
}

func (d *drivenBrowser) lastDriving() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.driving) == 0 {
		return "?"
	}
	return d.driving[len(d.driving)-1]
}

func startDrivenCast(t *testing.T) (*Cast, *drivenBrowser) {
	t.Helper()
	d := newDrivenBrowser(t)
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}, ProbeTimeout: 2 * time.Second})
	c, err := StartCast(context.Background(), CastConfig{
		Launch:        func(context.Context, *RecordOptions) (Browser, error) { return d, nil },
		RecordingsDir: t.TempDir(),
		Host:          host,
		Ping:          time.Second,
		DriveGrace:    time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.End)
	return c, d
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("%s: timed out", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// A recorded cast over a drivable browser has a controller. Interleavings:
// before the recorder opens control changes are refused (even though the
// browser may publish live first), after it they land; Chrome is told who
// drives; a revoke returns while a batch is stuck; the controller ends
// with the cast.
func TestCastDriving(t *testing.T) {
	c, d := startDrivenCast(t)
	ctrl := c.Drive()
	if ctrl == nil {
		t.Fatal("a recorded, drivable cast has no controller")
	}
	seat, err := ctrl.Join("Ana")
	if err != nil {
		t.Fatal(err)
	}
	d.SetLive(true) // Chrome publishes live; the recorder hasn't opened yet
	waitUpdate(t, c, "live", func(u Update) bool { return u.State.Phase == PhaseLive })
	if err := ctrl.Request(seat.Key); !errors.Is(err, drive.ErrNotLive) {
		t.Fatalf("before the recorder opened: %v", err)
	}
	d.opened.Store(true)
	if err := ctrl.Request(seat.Key); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Grant(seat.Viewer.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "Chrome told Ana drives", func() bool { return d.lastDriving() == "Ana" })

	d.hold = make(chan struct{})
	s := d.inj.Page()
	q := drive.ElementByID(s, "q")
	box := s.Boxes[q.ID]
	batch := drive.Batch{Seg: s.Seg, Acts: []drive.Act{{A: drive.ActClick, ID: q.ID, X: box[0] + 1, Y: box[1] + 1, Button: drive.ButtonLeft, Count: 1}}}
	stuck := make(chan error, 1)
	go func() { stuck <- ctrl.Input(context.Background(), seat.Key, batch) }()
	time.Sleep(20 * time.Millisecond)
	revoked := make(chan error, 1)
	go func() { revoked <- ctrl.Revoke() }()
	select {
	case err := <-revoked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("revoke waited on a stuck batch")
	}
	close(d.hold)
	if err := <-stuck; !errors.Is(err, drive.ErrNotDriver) {
		t.Fatalf("the overtaken batch: %v", err)
	}
	eventually(t, "Chrome told nobody drives", func() bool { return d.lastDriving() == "" })
	recs := d.stream.Records()
	all := append(drive.DrivenPage(100), recs...)
	if _, err := recording.Reduce(all, all[len(all)-1].Time()); err != nil {
		t.Fatalf("the stream doesn't reduce: %v", err)
	}

	d.QuitByHand()
	<-c.Done()
	if _, err := ctrl.Join("Bo"); !errors.Is(err, drive.ErrClosed) {
		t.Fatalf("after the cast: %v", err)
	}
}

// A cast that can't be driven (not recorded) has no controller.
func TestCastWithoutDriving(t *testing.T) {
	c, _ := startCast(t, &broadcast.FakeTunnel{})
	if c.Drive() != nil {
		t.Fatal("an unrecorded cast has a controller")
	}
}
