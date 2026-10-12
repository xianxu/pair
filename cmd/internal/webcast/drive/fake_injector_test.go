package drive

import (
	"context"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func newFakeInjector(t *testing.T) func(page []recording.Record) (Injector, func() []recording.Record) {
	return func(page []recording.Record) (Injector, func() []recording.Record) {
		sink := &recording.MemSink{}
		f, err := NewFakeInjector(page, sink)
		if err != nil {
			t.Fatal(err)
		}
		return f, sink.Records
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
	if err == nil || len(sink.Records()) != 0 {
		t.Fatalf("a viewer who isn't driving injected: %v, %d records", err, len(sink.Records()))
	}
}
