package drive

import (
	"context"
	"sync"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// FakeInjector is the stateful fake of Injector: TabInjector, the rules the
// real injector runs, over a fake tab. The tab is a page reduced from
// records, hit-tested by its layout boxes (the smallest box holding the
// point), whose effects are records put into a sink, stamped by.
//   - a click: focus (on a form control, when it moves) and click;
//   - text: input on the focused input or textarea, its value extended (a
//     password's as #458's input {secret:true}, never the value);
//   - a wheel: scroll of the top document;
//   - down, up, move, key and ime: nothing a recording shows.
//
// It follows the record stream as a recording.Sink: Put applies each record
// (ctl included) to its page, so it knows who drives. Its own records are
// applied too, and go to its out sink. Time advances 1 ms per record from
// the page's last.
type FakeInjector struct {
	*TabInjector
	tab *fakeTab
}

type fakeTab struct {
	mu   sync.Mutex
	page *recording.State
	sink recording.Sink
	src  recording.Source // the running batch's by
}

// NewFakeInjector reduces page and puts what injection causes into sink.
func NewFakeInjector(page []recording.Record, sink recording.Sink) (*FakeInjector, error) {
	s := recording.NewState()
	for _, r := range page {
		if err := s.Apply(r); err != nil {
			return nil, err
		}
	}
	tab := &fakeTab{page: s, sink: sink}
	return &FakeInjector{TabInjector: NewTabInjector(tab), tab: tab}, nil
}

// Page is the fake's current page state.
func (f *FakeInjector) Page() *recording.State {
	f.tab.mu.Lock()
	defer f.tab.mu.Unlock()
	return f.tab.page
}

// Put follows the stream: one record of the cast, in order.
func (f *FakeInjector) Put(r recording.Record) error {
	f.tab.mu.Lock()
	defer f.tab.mu.Unlock()
	return f.tab.page.Apply(r)
}

// Close: the stream ended. Nothing to release.
func (f *FakeInjector) Close() error { return nil }

// WithTab runs fn holding the tab: one run at a time, as on the real
// stream's writer.
func (t *fakeTab) WithTab(ctx context.Context, by string, fn func(Tab) error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.src = recording.Source{By: by}
	defer func() { t.src = recording.Source{} }()
	return fn(t)
}

func (t *fakeTab) Page() *recording.State { return t.page }

// At is the smallest layout box holding the point.
func (t *fakeTab) At(x, y float64) (recording.NodeID, error) {
	var hit recording.NodeID
	var area float64
	for id, box := range t.page.Boxes {
		if x < box[0] || y < box[1] || x >= box[0]+box[2] || y >= box[1]+box[3] {
			continue
		}
		if t.page.Node(id) != nil && (hit == 0 || box[2]*box[3] < area) {
			hit, area = id, box[2]*box[3]
		}
	}
	return hit, nil
}

func (t *fakeTab) Dispatch(a Act) error {
	src := t.src
	switch a.A {
	case ActClick:
		if n := t.page.Node(a.ID); formControl(n) && t.page.Focus != a.ID {
			if err := t.emit(&recording.Focus{Source: src, ID: a.ID}); err != nil {
				return err
			}
		}
		return t.emit(&recording.Click{Source: src, ID: a.ID, X: a.X, Y: a.Y})
	case ActText:
		n := t.page.Node(t.page.Focus)
		if n == nil || (n.N != "input" && n.N != "textarea") {
			return nil // nothing focused takes text
		}
		if n.A["type"] == "password" {
			return t.emit(&recording.Input{Source: src, ID: n.ID, Secret: true})
		}
		v := a.V
		if in := t.page.Inputs[n.ID]; in != nil && in.V != nil {
			v = *in.V + v
		}
		return t.emit(&recording.Input{Source: src, ID: n.ID, V: recording.Str(v)})
	case ActWheel:
		top := t.page.Top.ID
		at := t.page.Scrolls[top]
		return t.emit(&recording.Scroll{Source: src, ID: top, X: max(0, at[0]+a.DX), Y: max(0, at[1]+a.DY)})
	}
	return nil
}

// emit applies r to the page, then puts it into the sink.
func (t *fakeTab) emit(r recording.Record) error {
	r.SetTime(t.page.T + 1)
	if err := t.page.Apply(r); err != nil {
		return err
	}
	return t.sink.Put(r)
}

func formControl(n *recording.DOMNode) bool {
	if n == nil {
		return false
	}
	switch n.N {
	case "input", "textarea", "select", "button":
		return true
	}
	return false
}
