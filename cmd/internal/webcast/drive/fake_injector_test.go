package drive

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func newFakeInjector(t *testing.T) func(page []recording.Record) (Injector, recording.Sink, func() []recording.Record) {
	return func(page []recording.Record) (Injector, recording.Sink, func() []recording.Record) {
		sink := &recording.MemSink{}
		f, err := NewFakeInjector(page, sink)
		if err != nil {
			t.Fatal(err)
		}
		return f, f, sink.Records
	}
}

func TestFakeInjectorContract(t *testing.T) { RunInjectorContract(t, newFakeInjector(t)) }

func TestFakeInjectorWheelScrollsThePage(t *testing.T) {
	sink := &recording.MemSink{}
	f, err := NewFakeInjector(DrivenPage(drivenGrantedT), sink)
	if err != nil {
		t.Fatal(err)
	}
	s := f.Page()
	q := ElementByID(s, "q")
	b := Batch{Seg: s.Seg, Acts: []Act{{A: ActWheel, ID: q.ID, X: 20, Y: 20, DY: 120}, {A: ActWheel, ID: q.ID, X: 20, Y: 20, DY: -500}}}
	if err := f.Inject(context.Background(), drivenDriver, b); err != nil {
		t.Fatal(err)
	}
	recs := sink.Records()
	last := recs[len(recs)-1].(*recording.Scroll)
	if len(recs) != 2 || last.Y != 0 || last.By != drivenDriver {
		t.Fatalf("scrolls %+v", recs)
	}
}

func TestFakeInjectorRefusesAnUnattributableDriver(t *testing.T) {
	sink := &recording.MemSink{}
	f, _ := NewFakeInjector(DrivenPage(drivenGrantedT), sink)
	s := f.Page()
	q := ElementByID(s, "q")
	box := s.Boxes[q.ID]
	err := f.Inject(context.Background(), "v9", Batch{Seg: s.Seg, Acts: []Act{{A: ActClick, ID: q.ID, X: box[0] + 1, Y: box[1] + 1, Button: ButtonLeft}}})
	if !errors.Is(err, ErrNotDriver) || len(sink.Records()) != 0 {
		t.Fatalf("a viewer who isn't driving injected: %v, %d records", err, len(sink.Records()))
	}
}

// The fakes compose as the real pieces do: the controller's ctl records go
// into the stream the injector follows, so a granted driver injects and a
// revoked one is refused.
func TestFakesCompose(t *testing.T) {
	stream := &recording.MemSink{}
	page := DrivenPage(100) // live, nobody driving
	for _, r := range page {
		stream.Put(r)
	}
	inj, err := NewFakeInjector(page, stream)
	if err != nil {
		t.Fatal(err)
	}
	clock := func() int64 { return inj.Page().T + 1 }
	c := NewFakeController(recording.Tee(stream, inj), inj, clock, time.Minute)
	defer c.Close()
	seat, _ := c.Join("Ana")
	detach, _ := c.Attach(seat.Viewer.ID)
	defer detach()
	s := inj.Page()
	q := ElementByID(s, "q")
	box := s.Boxes[q.ID]
	b := Batch{Seg: s.Seg, Acts: []Act{{A: ActClick, ID: q.ID, X: box[0] + 1, Y: box[1] + 1, Button: ButtonLeft, Count: 1}, {A: ActText, V: "hi"}}}
	if err := c.Request(seat.Key); err != nil {
		t.Fatal(err)
	}
	if err := c.Grant(seat.Viewer.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.Input(context.Background(), seat.Key, b); err != nil {
		t.Fatalf("a granted driver: %v", err)
	}
	if err := c.Revoke(); err != nil {
		t.Fatal(err)
	}
	// Input checks the controller's state first; the injector's own check
	// is what refuses a batch already past it, so call it directly.
	if err := inj.Inject(context.Background(), seat.Viewer.ID, b); !errors.Is(err, ErrNotDriver) {
		t.Fatalf("a revoked driver at the injector: %v", err)
	}
	recs := stream.Records()
	if _, err := recording.Reduce(recs, recs[len(recs)-1].Time()); err != nil {
		t.Fatalf("the composed stream doesn't reduce: %v", err)
	}
}
