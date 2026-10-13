package recorder

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// driveTab is a fake cast tab's answers to a driven run, as observer.js
// gives them: the hold returns what was pending (the operator's); the take
// returns nothing and sends what the dispatch caused through the binding,
// tagged with the hold's by (send). Also the node at every point, and the
// input it got.
type driveTab struct {
	mu       sync.Mutex
	held     []map[string]any // pending at the hold (the operator's)
	taken    []map[string]any // what the dispatch caused (the driver's)
	heldBy   string
	send     func(payload string) // a bindingCalled from the page
	hit      int              // backendNodeId at any point
	hitErr   error            // what the hit test fails with, if set
	hitAt    [][2]float64     // the points hit-tested
	takeFail bool
	block    chan struct{} // while set, input calls wait on it
	inputs   []string
	evals    []string
}

func (d *driveTab) more(session, method string, params map[string]any, p *fakePage) (any, error, bool) {
	d.mu.Lock()
	switch {
	case method == "Runtime.evaluate":
		expr, _ := params["expression"].(string)
		var items []map[string]any
		switch {
		case strings.Contains(expr, "__couchcastHold"):
			d.evals = append(d.evals, "hold")
			items, d.held = d.held, nil
			if i := strings.Index(expr, `__couchcastHold("`); i >= 0 {
				rest := expr[i+len(`__couchcastHold("`):]
				d.heldBy = rest[:strings.Index(rest, `"`)]
			}
		case strings.Contains(expr, "__couchcastTake"):
			d.evals = append(d.evals, "take")
			if d.takeFail {
				d.mu.Unlock()
				return nil, errors.New("Cannot find context with specified id"), true
			}
			if len(d.taken) > 0 && d.send != nil {
				payload, _ := jsonString(map[string]any{"t": "obs", "by": d.heldBy, "items": d.taken})
				d.send(payload)
			}
			d.taken = nil
		default:
			d.mu.Unlock()
			return nil, nil, false
		}
		d.mu.Unlock()
		if items == nil {
			items = []map[string]any{}
		}
		b, _ := jsonString(items)
		return map[string]any{"result": map[string]any{"value": b}}, nil, true
	case method == "DOM.getNodeForLocation":
		hit, err := d.hit, d.hitErr
		num := func(v any) float64 {
			switch n := v.(type) {
			case int:
				return float64(n)
			case float64:
				return n
			}
			return -1
		}
		d.hitAt = append(d.hitAt, [2]float64{num(params["x"]), num(params["y"])})
		d.mu.Unlock()
		if err != nil {
			return nil, err, true
		}
		return map[string]any{"backendNodeId": hit}, nil, true
	case strings.HasPrefix(method, "Input."):
		d.inputs = append(d.inputs, method)
		block := d.block
		d.mu.Unlock()
		if block != nil {
			<-block
		}
		return map[string]any{}, nil, true
	}
	d.mu.Unlock()
	return nil, nil, false
}

func (d *driveTab) log() (inputs, evals []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.inputs...), append([]string(nil), d.evals...)
}

// drivenHarness: page A live with a button, an observer context, and v1
// ("Ana") granted control through the recorder's queue.
func drivenHarness(t *testing.T, dialog chan struct{}) (*harness, *driveTab, *fakePage, *fnode) {
	t.Helper()
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	d := &driveTab{}
	h.fc.more = d.more
	opts := Options{ID: "t", Call: h.fc, ShotEvery: -1,
		Out: func() (Opened, error) { h.out.opened++; return Opened{Sink: &h.out.sink}, nil }}
	if dialog != nil {
		opts.Dialog = func() <-chan struct{} { return dialog }
	}
	h.r = Start(opts)
	t.Cleanup(func() { h.r.Close() })
	d.send = func(payload string) {
		h.r.Send(Event{Session: "A", Method: "Runtime.bindingCalled", Params: raw(map[string]any{"name": "__couchcast", "payload": payload, "executionContextId": 7})})
	}
	p := h.page("A", "http://a/", "L1")
	btn := p.el("button")
	p.insert(p.body(), nil, btn)
	p.take()
	h.ctxCreated("A", p.frameID, 7)
	if h.r.Opened() {
		t.Fatal("opened before the first go-live")
	}
	h.r.Send(Live{Tab: rec.Tab{ID: p.frameID}, Session: "A"})
	h.settle()
	if !h.r.Opened() {
		t.Fatal("not opened after go-live")
	}
	ctl(h, rec.CtlRequest, rec.CtlGrant)
	d.hit = btn.backend
	return h, d, p, btn
}

var ana = rec.Author{ID: "v1", Name: "Ana"}

// ctl puts control changes for Ana through the queue, as the controller does.
func ctl(h *harness, ops ...string) {
	h.t.Helper()
	c := rec.Control{}
	if s := h.out.sink.Records(); len(s) > 0 {
		st, err := rec.Reduce(s, s[len(s)-1].Time())
		if err == nil {
			c = st.Control
		}
	}
	for _, op := range ops {
		next, err := rec.NextControl(c, op, ana)
		if err != nil {
			h.t.Fatal(err)
		}
		h.r.Send(Emit{Record: &rec.Ctl{Op: op, Who: ana, Control: next}})
		c = next
	}
	h.settle()
}

func clickOn(h *harness, id rec.NodeID) drive.Batch {
	st := h.mustValidate()
	return drive.Batch{Seg: st.Seg, Acts: []drive.Act{{A: drive.ActClick, ID: id, X: 5, Y: 5, Button: drive.ButtonLeft, Count: 1}}}
}

// The driver's click: what the observer had pending before is recorded
// without by, what the dispatch caused (sent tagged through the binding)
// with the driver's by, in that order, and the input reached the tab.
func TestWithTabAttributesExactly(t *testing.T) {
	h, d, _, btn := drivenHarness(t, nil)
	d.held = []map[string]any{{"k": "hover", "p": []int{1, 2, 0}}}
	d.taken = []map[string]any{{"k": "click", "p": []int{1, 2, 0}, "x": 5, "y": 5}}
	inj := drive.NewTabInjector(h.r)
	if err := inj.Inject(context.Background(), "v1", clickOn(h, h.id("A", btn.backend))); err != nil {
		t.Fatal(err)
	}
	h.settle()
	hover, click := h.last("hover").(*rec.Hover), h.last("click").(*rec.Click)
	if hover.By != "" || click.By != "v1" || hover.Time() > click.Time() {
		t.Fatalf("hover by %q, click by %q", hover.By, click.By)
	}
	inputs, evals := d.log()
	if len(inputs) != 3 || strings.Join(evals, " ") != "hold take" {
		t.Fatalf("inputs %v, evals %v", inputs, evals)
	}
	h.mustValidate()
}

// A revoke queued before the run is seen by it: nothing is dispatched.
func TestWithTabSeesARevokeQueuedFirst(t *testing.T) {
	h, d, _, btn := drivenHarness(t, nil)
	b := clickOn(h, h.id("A", btn.backend))
	ctl(h, rec.CtlRevoke)
	err := drive.NewTabInjector(h.r).Inject(context.Background(), "v1", b)
	if !errors.Is(err, drive.ErrNotDriver) {
		t.Fatalf("after a revoke: %v", err)
	}
	if inputs, _ := d.log(); len(inputs) != 0 {
		t.Fatalf("dispatched %v", inputs)
	}
}

// Every exit takes: a stale target (the rules stop the run) still releases
// the hold; a take that fails leaves the recorder working, and the next
// ordinary report is written.
func TestWithTabReleasesOnEveryExit(t *testing.T) {
	h, d, p, btn := drivenHarness(t, nil)
	other := p.el("i")
	p.insert(p.body(), btn, other)
	h.events("A", p)
	h.settle()
	d.hit = other.backend
	err := drive.NewTabInjector(h.r).Inject(context.Background(), "v1", clickOn(h, h.id("A", btn.backend)))
	if !errors.Is(err, drive.ErrStale) {
		t.Fatalf("stale: %v", err)
	}
	if _, evals := d.log(); strings.Join(evals, " ") != "hold take" {
		t.Fatalf("a stale run didn't take: %v", evals)
	}
	d.mu.Lock()
	d.takeFail, d.hit = true, btn.backend
	d.mu.Unlock()
	if err := drive.NewTabInjector(h.r).Inject(context.Background(), "v1", clickOn(h, h.id("A", btn.backend))); err != nil {
		t.Fatalf("a failed take fails the batch: %v", err)
	}
	before := h.count("click")
	payload, _ := jsonString(map[string]any{"t": "obs", "items": []any{map[string]any{"k": "click", "p": []int{1, 2, 0}, "x": 1, "y": 1}}})
	h.send("A", "Runtime.bindingCalled", map[string]any{"name": "__couchcast", "payload": payload, "executionContextId": 7})
	h.settle()
	if h.count("click") != before+1 || h.last("click").(*rec.Click).By != "" {
		t.Fatalf("the operator's next click: %d → %d", before, h.count("click"))
	}
}

// A dialog that opens mid-dispatch gives the driver ErrDialog at once,
// without waiting for the stopped input call; the take is skipped (it
// would wait on the dialog too) and the recorder goes on.
func TestWithTabGivesUpOnADialog(t *testing.T) {
	dialog := make(chan struct{})
	h, d, _, btn := drivenHarness(t, dialog)
	block := make(chan struct{})
	defer close(block)
	d.mu.Lock()
	d.block = block
	d.mu.Unlock()
	done := make(chan error, 1)
	go func() {
		done <- drive.NewTabInjector(h.r).Inject(context.Background(), "v1", clickOn(h, h.id("A", btn.backend)))
	}()
	for {
		if inputs, _ := d.log(); len(inputs) > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	close(dialog) // the page opened an alert
	select {
	case err := <-done:
		if !errors.Is(err, drive.ErrDialog) {
			t.Fatalf("got %v", err)
		}
		if code, say := drive.Answer(err); code != 409 || !strings.Contains(say, "dialog") {
			t.Fatalf("answer %d %q", code, say)
		}
	case <-time.After(callTimeout / 2):
		t.Fatal("the dispatch waited on the dialog")
	}
	if _, evals := d.log(); strings.Join(evals, " ") != "hold" {
		t.Fatalf("evals %v: the take must be skipped while a dialog is open", evals)
	}
	h.settle() // the loop is free
}

func jsonString(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// A held observer's tagged report (a click sent at once, before its
// document goes) is recorded with its by (#467); one ordered after control
// moved (a revoke between the batch and its reports) keeps the act, without
// the attribution.
func TestHeldDocumentSendsAttributedItems(t *testing.T) {
	h, _, _, _ := drivenHarness(t, nil)
	send := func(by string) {
		payload, _ := jsonString(map[string]any{"t": "obs", "by": by, "items": []any{map[string]any{"k": "click", "p": []int{1, 2, 0}, "x": 1, "y": 1}}})
		h.send("A", "Runtime.bindingCalled", map[string]any{"name": "__couchcast", "payload": payload, "executionContextId": 7})
		h.settle()
	}
	send("v1")
	if c := h.last("click").(*rec.Click); c.By != "v1" {
		t.Fatalf("the navigating click: by %q", c.By)
	}
	n := h.count("click")
	ctl(h, rec.CtlRevoke)
	send("v1")
	if h.count("click") != n+1 || h.last("click").(*rec.Click).By != "" {
		t.Fatalf("a click reported after the revoke: %d clicks, by %q", h.count("click"), h.last("click").(*rec.Click).By)
	}
}

// pair#505: the hit test takes the act's top-page coordinates as they are,
// at any scroll (measured: Chrome's DOM.getNodeForLocation hit-tests page
// px; subtracting the scroll made every click after a scroll stale). A
// "No node found" is nothing there (stale); any other failure is the tab's
// (ErrInput, logged for the operator), not "the page changed".
func TestCdpTabHitTestsPageCoordinates(t *testing.T) {
	h, d, _, btn := drivenHarness(t, nil)
	payload, _ := json.Marshal(map[string]any{"t": "obs", "items": []any{map[string]any{"k": "scroll", "p": []int{}, "x": 0, "y": 300}}})
	h.send("A", "Runtime.bindingCalled", map[string]any{"name": "__couchcast", "payload": string(payload), "executionContextId": 7})
	h.settle()
	b := clickOn(h, h.id("A", btn.backend))
	b.Acts[0].X, b.Acts[0].Y = 40, 420 // top-page px, below the 300 scroll
	if err := drive.NewTabInjector(h.r).Inject(context.Background(), ana.ID, b); err != nil {
		t.Fatalf("click after a scroll: %v", err)
	}
	d.mu.Lock()
	at := d.hitAt[len(d.hitAt)-1]
	d.mu.Unlock()
	if at != [2]float64{40, 420} {
		t.Fatalf("hit-tested at %v, want the page point (40, 420)", at)
	}

	for _, c := range []struct {
		name string
		err  error
		want error
	}{
		{"nothing there", &cdp.Error{Code: -32000, Message: "No node found at given location"}, drive.ErrStale},
		{"the session went", errors.New("cdp: connection closed"), drive.ErrInput},
	} {
		d.mu.Lock()
		d.hitErr = c.err
		d.mu.Unlock()
		if err := drive.NewTabInjector(h.r).Inject(context.Background(), ana.ID, clickOn(h, h.id("A", btn.backend))); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
}
