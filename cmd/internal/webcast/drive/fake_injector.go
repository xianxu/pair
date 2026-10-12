package drive

import (
	"context"
	"fmt"
	"sync"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// FakeInjector is the stateful fake of Injector: a page reduced from
// records, hit-tested by its layout boxes (the smallest box holding the
// point), whose effects are records put into a sink, stamped by.
//   - a click: focus (on a form control, when it moves) and click;
//   - text: input on the focused input or textarea, its value extended;
//   - a wheel: scroll of the top document;
//   - down, up, move, key and ime: nothing a recording shows.
//
// It follows the record stream as a recording.Sink: Put applies each record
// (ctl included) to its page, so it knows who drives. Its own records are
// applied too, and go to its out sink. Time advances 1 ms per record from
// the page's last.
type FakeInjector struct {
	mu   sync.Mutex
	page *recording.State
	sink recording.Sink
}

// NewFakeInjector reduces page and puts what injection causes into sink.
func NewFakeInjector(page []recording.Record, sink recording.Sink) (*FakeInjector, error) {
	s := recording.NewState()
	for _, r := range page {
		if err := s.Apply(r); err != nil {
			return nil, err
		}
	}
	return &FakeInjector{page: s, sink: sink}, nil
}

// Page is the fake's current page state.
func (f *FakeInjector) Page() *recording.State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.page
}

// Put follows the stream: one record of the cast, in order.
func (f *FakeInjector) Put(r recording.Record) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.page.Apply(r)
}

// Close: the stream ended. Nothing to release.
func (f *FakeInjector) Close() error { return nil }

func (f *FakeInjector) Inject(ctx context.Context, by string, b Batch) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if by == "" || f.page.Control.Driving() != by {
		return ErrNotDriver
	}
	if !f.page.Live || b.Seg != f.page.Seg {
		return ErrNotLive
	}
	if err := confineAll(b); err != nil {
		return err
	}
	src := recording.Source{By: by}
	for _, a := range b.Acts {
		if err := ctx.Err(); err != nil {
			return err
		}
		if a.A == ActDown || a.A == ActClick || a.A == ActWheel {
			if !f.hits(a) {
				return ErrStale
			}
		}
		var err error
		switch a.A {
		case ActClick:
			if n := f.page.Node(a.ID); formControl(n) && f.page.Focus != a.ID {
				err = f.emit(&recording.Focus{Source: src, ID: a.ID})
			}
			if err == nil {
				err = f.emit(&recording.Click{Source: src, ID: a.ID, X: a.X, Y: a.Y})
			}
		case ActText:
			n := f.page.Node(f.page.Focus)
			if n == nil || (n.N != "input" && n.N != "textarea") {
				continue // nothing focused takes text
			}
			if n.A["type"] == "password" {
				// #458's rule: a password's value is never recorded
				// (input {secret:true}); the fake follows it once that
				// field exists, and until then refuses.
				return fmt.Errorf("%w: typing into a password field (pending #458's secret rule)", ErrRefused)
			}
			v := a.V
			if in := f.page.Inputs[n.ID]; in != nil && in.V != nil {
				v = *in.V + v
			}
			err = f.emit(&recording.Input{Source: src, ID: n.ID, V: recording.Str(v)})
		case ActWheel:
			top := f.page.Top.ID
			at := f.page.Scrolls[top]
			err = f.emit(&recording.Scroll{Source: src, ID: top, X: max(0, at[0]+a.DX), Y: max(0, at[1]+a.DY)})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// hits: the smallest layout box holding the point is a's element or inside it.
func (f *FakeInjector) hits(a Act) bool {
	var hit *recording.DOMNode
	var area float64
	for id, box := range f.page.Boxes {
		if a.X < box[0] || a.Y < box[1] || a.X >= box[0]+box[2] || a.Y >= box[1]+box[3] {
			continue
		}
		if n := f.page.Node(id); n != nil && (hit == nil || box[2]*box[3] < area) {
			hit, area = n, box[2]*box[3]
		}
	}
	for n := hit; n != nil; n = n.Parent {
		if n.ID == a.ID {
			return true
		}
	}
	return false
}

// emit applies r to the page, then puts it into the sink.
func (f *FakeInjector) emit(r recording.Record) error {
	r.SetTime(f.page.T + 1)
	if err := f.page.Apply(r); err != nil {
		return err
	}
	return f.sink.Put(r)
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
