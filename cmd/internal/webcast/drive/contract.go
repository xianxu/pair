package drive

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Pages of recording.DrivenRecording the contracts start from: viewer v1
// ("Ana") was just granted control on a live page with a search input #q, a
// button #save and a link, each with a layout box; and the same cast paused.
const (
	drivenGrantedT = 1500
	drivenPausedT  = 3500
	drivenDriver   = "v1"
)

// DrivenPage is DrivenRecording up to t.
func DrivenPage(t int64) []recording.Record {
	var out []recording.Record
	for _, r := range recording.DrivenRecording() {
		if r.Time() <= t {
			out = append(out, r)
		}
	}
	return out
}

// ElementByID is the live element with that id attribute, or nil.
func ElementByID(s *recording.State, id string) *recording.DOMNode {
	var walk func(n *recording.DOMNode) *recording.DOMNode
	walk = func(n *recording.DOMNode) *recording.DOMNode {
		if n.Y == recording.YEl && n.A["id"] == id {
			return n
		}
		for _, k := range n.Kids {
			if m := walk(k); m != nil {
				return m
			}
		}
		return nil
	}
	if s.Top == nil {
		return nil
	}
	return walk(s.Top)
}

// RunInjectorContract checks an Injector. newInjector loads page (records
// of recording.DrivenRecording: build the tab from them) and returns the
// injector, the sink that feeds it later records of the stream it follows,
// and what it has caused so far, as records.
func RunInjectorContract(t recording.TB, newInjector func(page []recording.Record) (Injector, recording.Sink, func() []recording.Record)) {
	t.Helper()
	ctx := context.Background()
	page := DrivenPage(drivenGrantedT)
	s, err := recording.Reduce(page, drivenGrantedT)
	if err != nil {
		t.Fatalf("the contract page doesn't reduce: %v", err)
	}
	q, save := ElementByID(s, "q"), ElementByID(s, "save")
	qBox, saveBox := s.Boxes[q.ID], s.Boxes[save.ID]
	at := func(b recording.Box) (float64, float64) { return b[0] + b[2]/2, b[1] + b[3]/2 }
	qx, qy := at(qBox)
	sx, sy := at(saveBox)
	click := func(id recording.NodeID, x, y float64) Act {
		return Act{A: ActClick, ID: id, X: x, Y: y, Button: ButtonLeft, Count: 1}
	}

	// Paused: nothing reaches the page.
	{
		inj, _, observed := newInjector(DrivenPage(drivenPausedT))
		if err := inj.Inject(ctx, drivenDriver, Batch{Seg: 1, Acts: []Act{click(q.ID, qx, qy)}}); !errors.Is(err, ErrNotLive) {
			t.Errorf("paused: got %v, want ErrNotLive", err)
		}
		if n := len(observed()); n != 0 {
			t.Errorf("paused: %d records caused", n)
		}
	}

	inj, follow, observed := newInjector(page)
	// Only the driver injects.
	if err := inj.Inject(ctx, "v2", Batch{Seg: s.Seg, Acts: []Act{click(q.ID, qx, qy)}}); !errors.Is(err, ErrNotDriver) {
		t.Errorf("a viewer who doesn't drive: got %v, want ErrNotDriver", err)
	}
	// A batch aimed at another segment's document.
	if err := inj.Inject(ctx, drivenDriver, Batch{Seg: s.Seg + 1, Acts: []Act{click(q.ID, qx, qy)}}); !errors.Is(err, ErrNotLive) {
		t.Errorf("another segment: got %v, want ErrNotLive", err)
	}
	// One refused act refuses the batch, the acts before it included.
	refused := Batch{Seg: s.Seg, Acts: []Act{click(q.ID, qx, qy), {A: ActKey, Type: KeyDown, Key: "v", Mods: []string{ModMeta}}}}
	if err := inj.Inject(ctx, drivenDriver, refused); !errors.Is(err, ErrRefused) {
		t.Errorf("paste: got %v, want ErrRefused", err)
	}
	if n := len(observed()); n != 0 {
		t.Fatalf("a refused batch caused %d records", n)
	}
	// Aiming at #save where #q is: stale.
	if err := inj.Inject(ctx, drivenDriver, Batch{Seg: s.Seg, Acts: []Act{click(save.ID, qx, qy)}}); !errors.Is(err, ErrStale) {
		t.Errorf("stale target: got %v, want ErrStale", err)
	}
	if n := len(observed()); n != 0 {
		t.Fatalf("a stale click caused %d records", n)
	}
	// The driver clicks #q and types: attributed click and value.
	ok := Batch{Seg: s.Seg, Acts: []Act{click(q.ID, qx, qy), {A: ActText, V: "inv"}, {A: ActText, V: "oice"}}}
	if err := inj.Inject(ctx, drivenDriver, ok); err != nil {
		t.Fatalf("click and type: %v", err)
	}
	var clicked bool
	var value string
	for _, r := range observed() {
		switch r := r.(type) {
		case *recording.Click:
			clicked = clicked || (r.ID == q.ID && r.By == drivenDriver)
		case *recording.Input:
			if r.ID == q.ID && r.V != nil && r.By == drivenDriver {
				value = *r.V
			}
		}
	}
	if !clicked || value != "invoice" {
		t.Errorf("after click and typing: clicked by the driver %v, value by the driver %q", clicked, value)
	}
	// A click on #save after it still lands: the earlier acts stood.
	if err := inj.Inject(ctx, drivenDriver, Batch{Seg: s.Seg, Acts: []Act{click(save.ID, sx, sy)}}); err != nil {
		t.Errorf("click #save: %v", err)
	}
	// Everything caused reduces after the page: by is the driver's.
	all := append(append([]recording.Record(nil), page...), observed()...)
	end := all[len(all)-1].Time()
	if _, err := recording.Reduce(all, end); err != nil {
		t.Errorf("the caused records don't reduce after the page: %v", err)
	}
	// A revoke in the stream: the old driver's next batch is refused where
	// it is dispatched, and causes nothing.
	revoke := &recording.Ctl{Op: recording.CtlRevoke, Who: *s.Control.Driver, Control: recording.Control{}}
	revoke.SetTime(end + 1)
	if err := follow.Put(revoke); err != nil {
		t.Fatalf("following a revoke: %v", err)
	}
	n := len(observed())
	if err := inj.Inject(ctx, drivenDriver, Batch{Seg: s.Seg, Acts: []Act{click(save.ID, sx, sy)}}); !errors.Is(err, ErrNotDriver) {
		t.Errorf("after a revoke: got %v, want ErrNotDriver", err)
	}
	if len(observed()) != n {
		t.Errorf("a revoked driver's batch caused records")
	}
}
