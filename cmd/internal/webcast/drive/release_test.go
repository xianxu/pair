package drive

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// what is left pressed: the ups the tab got after the driver's acts.
func ups(acts []Act, from int) []string {
	var out []string
	for _, a := range acts[from:] {
		switch {
		case a.A == ActUp:
			out = append(out, "up")
		case a.A == ActKey && a.Type == KeyUp:
			out = append(out, "keyup "+a.Key)
		}
	}
	return out
}

func ctlRecord(t *testing.T, inj *FakeInjector, op string, who recording.Author) {
	t.Helper()
	next, err := recording.NextControl(inj.Page().Control, op, who)
	if err != nil {
		t.Fatal(err)
	}
	r := &recording.Ctl{Op: op, Who: who, Control: next}
	r.SetTime(inj.Page().T + 1)
	if err := inj.Put(r); err != nil {
		t.Fatal(err)
	}
}

// Every exit lets go of what the driver held (#467): a revoke (Release, as
// the controller calls it), another driver's first batch, a batch on
// another page; and nothing is let go twice.
func TestTabInjectorLetsGo(t *testing.T) {
	ctx := context.Background()
	newInj := func() (*FakeInjector, Batch) {
		inj, err := NewFakeInjector(DrivenPage(drivenGrantedT), &recording.MemSink{})
		if err != nil {
			t.Fatal(err)
		}
		s := inj.Page()
		q := ElementByID(s, "q")
		box := s.Boxes[q.ID]
		return inj, Batch{Seg: s.Seg, Acts: []Act{
			{A: ActDown, ID: q.ID, X: box[0] + 1, Y: box[1] + 1, Button: ButtonLeft},
			{A: ActMove, X: box[0] + 5, Y: box[1] + 1, Button: ButtonLeft},
			{A: ActKey, Type: KeyDown, Key: "End"},
		}}
	}
	bo := recording.Author{ID: "v2", Name: "Bo"}

	// A revoke: Release lets go of the button and both keys, last first.
	inj, held := newInj()
	held.Acts = held.Acts[:2]
	held.Acts = append(held.Acts, Act{A: ActKey, Type: KeyDown, Key: "ArrowLeft"})
	if err := inj.Inject(ctx, drivenDriver, held); err != nil {
		t.Fatal(err)
	}
	n := len(inj.Dispatched())
	ctlRecord(t, inj, recording.CtlRevoke, *inj.Page().Control.Driver)
	if err := inj.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ups(inj.Dispatched(), n); len(got) != 2 || got[0] != "up" || got[1] != "keyup ArrowLeft" {
		t.Fatalf("after a revoke: %v", got)
	}
	if up := inj.Dispatched()[n]; up.X != held.Acts[1].X {
		t.Errorf("the up is where the drag was: %+v", up)
	}
	n = len(inj.Dispatched())
	inj.Release(ctx)
	if got := ups(inj.Dispatched(), n); len(got) != 0 {
		t.Fatalf("let go twice: %v", got)
	}

	// Another driver's first batch: the old driver's hold goes first.
	inj, held = newInj()
	held.Acts = held.Acts[:1]
	if err := inj.Inject(ctx, drivenDriver, held); err != nil {
		t.Fatal(err)
	}
	ctlRecord(t, inj, recording.CtlRequest, bo)
	ctlRecord(t, inj, recording.CtlGrant, bo)
	n = len(inj.Dispatched())
	if err := inj.Inject(ctx, "v2", Batch{Seg: inj.Page().Seg, Acts: []Act{{A: ActText, V: "x"}}}); err != nil {
		t.Fatal(err)
	}
	if d := inj.Dispatched()[n:]; len(d) != 2 || d[0].A != ActUp || d[1].A != ActText {
		t.Fatalf("Bo's batch after Ana's press: %+v", d)
	}

	// The driver's own batch refused (a stale target): the shell drops its
	// queued ups, so the tab lets go here.
	inj, held = newInj()
	if err := inj.Inject(ctx, drivenDriver, Batch{Seg: held.Seg, Acts: held.Acts[:1]}); err != nil {
		t.Fatal(err)
	}
	n = len(inj.Dispatched())
	stale := held.Acts[0]
	stale.A, stale.ID = ActClick, ElementByID(inj.Page(), "save").ID
	if err := inj.Inject(ctx, drivenDriver, Batch{Seg: held.Seg, Acts: []Act{stale}}); err == nil {
		t.Fatal("a stale click went through")
	}
	if got := ups(inj.Dispatched(), n); len(got) != 1 || got[0] != "up" {
		t.Fatalf("after a refusal: %v", got)
	}

	// Another page: a key held on the old one goes up before the first
	// batch on the new one.
	inj, held = newInj()
	held.Acts = held.Acts[2:3] // End down
	if err := inj.Inject(ctx, drivenDriver, held); err != nil {
		t.Fatal(err)
	}
	seg := &recording.Seg{Seg: inj.Page().Seg + 1, Why: recording.SegNavigate, Tab: recording.Tab{ID: "tab1", URL: "http://app.local/next"}, VP: recording.Viewport{W: 1280, H: 800, DPR: 1}}
	seg.SetTime(inj.Page().T + 1)
	if err := inj.Put(seg); err != nil {
		t.Fatal(err)
	}
	n = len(inj.Dispatched())
	if err := inj.Inject(ctx, drivenDriver, Batch{Seg: seg.Seg, Acts: []Act{{A: ActText, V: "x"}}}); err != nil {
		t.Fatal(err)
	}
	if got := ups(inj.Dispatched(), n); len(got) != 1 || got[0] != "keyup End" {
		t.Fatalf("on the new page: %v", got)
	}
}

// The controller lets go when control changes: a revoke mid-drag ends with
// the button up in the tab, without anyone's next batch.
func TestControllerLetsGoOnRevoke(t *testing.T) {
	stream := &recording.MemSink{}
	page := DrivenPage(100)
	for _, r := range page {
		stream.Put(r)
	}
	inj, err := NewFakeInjector(page, stream)
	if err != nil {
		t.Fatal(err)
	}
	c := NewMemController(recording.Tee(stream, inj), inj, func() int64 { return inj.Page().T + 1 }, time.Minute)
	defer c.Close()
	seat, _ := c.Join("Ana")
	c.Request(seat.Key)
	c.Grant(seat.Viewer.ID)
	s := inj.Page()
	q := ElementByID(s, "q")
	box := s.Boxes[q.ID]
	if err := c.Input(context.Background(), seat.Key, Batch{Seg: s.Seg, Acts: []Act{{A: ActDown, ID: q.ID, X: box[0] + 1, Y: box[1] + 1, Button: ButtonLeft}}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Revoke(); err != nil {
		t.Fatal(err)
	}
	if !eventually(func() bool { d := inj.Dispatched(); return d[len(d)-1].A == ActUp }) {
		t.Fatalf("the press outlived the revoke: %+v", inj.Dispatched())
	}
}
