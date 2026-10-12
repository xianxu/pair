package webcast

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// FakeDriven is a FakeBrowser a viewer can drive (#467): Driven's stateful
// fake. Its recorder is the fake injector over a page: what is emitted (the
// controller's ctl records) is stamped after the page, applied to the
// injector's page and put into out, as the recorder's queue does; the
// injector's records go to out too. Open stands for the recorder's first
// go-live. It records nothing else.
type FakeDriven struct {
	*FakeBrowser
	inj    *drive.FakeInjector
	out    recording.Sink
	opened atomic.Bool

	mu      sync.Mutex
	hold    chan struct{}
	driving []string
	failing error // Emit refuses with it, as a stopped recorder would
}

// FailEmits makes every later Emit fail with err (nil: succeed again).
func (d *FakeDriven) FailEmits(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failing = err
}

// NewFakeDriven drives f's tab, showing page; what the cast records goes
// to out.
func NewFakeDriven(f *FakeBrowser, page []recording.Record, out recording.Sink) (*FakeDriven, error) {
	d := &FakeDriven{FakeBrowser: f, out: out}
	inj, err := drive.NewFakeInjector(page, &lockedSink{mu: &d.mu, sink: out})
	if err != nil {
		return nil, err
	}
	d.inj = inj
	return d, nil
}

// Open is the recorder's first go-live: control records are taken after.
func (d *FakeDriven) Open()        { d.opened.Store(true) }
func (d *FakeDriven) Opened() bool { return d.opened.Load() }

// Injector is the fake injector the tab is: its page and what it got.
func (d *FakeDriven) Injector() *drive.FakeInjector { return d.inj }

func (d *FakeDriven) Emit(r recording.Record, _ int) error {
	d.mu.Lock()
	failing := d.failing
	d.mu.Unlock()
	if failing != nil {
		return failing
	}
	r.SetTime(d.inj.Page().T + 1)
	if err := d.inj.Put(r); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.out.Put(r)
}

func (d *FakeDriven) WithTab(ctx context.Context, by string, fn func(drive.Tab) error) error {
	d.mu.Lock()
	hold := d.hold
	d.mu.Unlock()
	if hold != nil && by != "" { // a driver's batch; a release isn't held
		<-hold
	}
	return d.inj.Tabs.WithTab(ctx, by, fn)
}

// HoldBatches makes drivers' batches wait until release is called (a
// dispatch stuck in the tab).
func (d *FakeDriven) HoldBatches() (release func()) {
	hold := make(chan struct{})
	d.mu.Lock()
	d.hold = hold
	d.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			d.mu.Lock()
			d.hold = nil
			d.mu.Unlock()
			close(hold)
		})
	}
}

func (d *FakeDriven) SetDriving(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.driving = append(d.driving, name)
	return nil
}

// Driving is who Chrome was last told drives ("" nobody; "?" never told).
func (d *FakeDriven) Driving() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.driving) == 0 {
		return "?"
	}
	return d.driving[len(d.driving)-1]
}

// lockedSink serializes puts into a sink shared by two writers.
type lockedSink struct {
	mu   *sync.Mutex
	sink recording.Sink
}

func (l *lockedSink) Put(r recording.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.sink.Put(r)
}
func (l *lockedSink) Close() error { return nil }

var _ Driven = (*FakeDriven)(nil)
