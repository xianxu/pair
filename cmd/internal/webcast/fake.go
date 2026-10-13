package webcast

import (
	"fmt"
	"sync"
)

// FakeBrowser is a stateful Browser for tests across packages. It models a
// cast Chrome the operator uses: tabs opened, switched, navigated and closed,
// the in-page control clicked, Chrome quit by hand. The active tab comes from
// the same Tracker the real Chrome uses, so the rule isn't restated here. The
// conformance suite runs one scenario against this and against real Chrome.
type FakeBrowser struct {
	mu        sync.Mutex
	tracker   Tracker
	phase     Phase
	nextID    int
	failLive  error
	stats     FakeBrowserStats
	err       error
	pub       *publisher[State]
	done      chan struct{}
	endedOnce sync.Once
}

// FakeBrowserStats counts what was asked of the fake.
type FakeBrowserStats struct{ SetLives, Closes, Focuses int }

// NewFakeBrowser starts idle with one blank tab, as Chrome does.
func NewFakeBrowser() *FakeBrowser {
	f := &FakeBrowser{done: make(chan struct{})}
	id := f.newIDLocked()
	f.tracker.Apply(TargetCreated{ID: id, URL: "about:blank"})
	f.tracker.Apply(PageReport{ID: id, Visible: true, Focused: true})
	f.pub = newPublisher(f.stateLocked())
	return f
}

func (f *FakeBrowser) newIDLocked() string {
	f.nextID++
	return fmt.Sprintf("tab-%d", f.nextID)
}

func (f *FakeBrowser) stateLocked() State {
	tab, ok := f.tracker.Active()
	return State{Phase: f.phase, Tab: tab, HasTab: ok}
}

// change applies fn under the lock and publishes the result, unless ended.
func (f *FakeBrowser) change(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.phase == PhaseEnded {
		return
	}
	fn()
	f.pub.publish(f.stateLocked())
}

func (f *FakeBrowser) State() State                      { return f.pub.get() }
func (f *FakeBrowser) Subscribe() (<-chan State, func()) { return f.pub.subscribe() }
func (f *FakeBrowser) Done() <-chan struct{}             { return f.done }
func (f *FakeBrowser) Stats() FakeBrowserStats           { f.mu.Lock(); defer f.mu.Unlock(); return f.stats }
func (f *FakeBrowser) FailNextSetLive(err error)         { f.mu.Lock(); f.failLive = err; f.mu.Unlock() }

func (f *FakeBrowser) Err() error {
	select {
	case <-f.done:
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.err
	default:
		return nil
	}
}

func (f *FakeBrowser) SetLive(live bool) error {
	f.mu.Lock()
	f.stats.SetLives++
	if f.phase == PhaseEnded {
		f.mu.Unlock()
		return ErrEnded
	}
	if err := f.failLive; err != nil {
		f.failLive = nil
		f.mu.Unlock()
		return err
	}
	f.mu.Unlock()
	f.change(func() {
		f.phase = PhaseIdle
		if live {
			f.phase = PhaseLive
		}
	})
	return nil
}

func (f *FakeBrowser) Focus() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.phase == PhaseEnded {
		return ErrEnded
	}
	f.stats.Focuses++
	return nil
}

func (f *FakeBrowser) Close() error {
	f.mu.Lock()
	f.stats.Closes++
	f.mu.Unlock()
	f.end(nil)
	return nil
}

func (f *FakeBrowser) end(reason error) {
	f.endedOnce.Do(func() {
		f.mu.Lock()
		f.phase, f.err = PhaseEnded, reason
		final := f.stateLocked()
		f.mu.Unlock()
		f.pub.finish(final)
		close(f.done)
	})
}

// Navigate loads url, titled title, in the active tab.
func (f *FakeBrowser) Navigate(url, title string) {
	f.change(func() {
		if tab, ok := f.tracker.Active(); ok {
			f.tracker.Apply(TargetChanged{ID: tab.ID, URL: url, Title: title})
		}
	})
}

// OpenTab opens a tab, in the foreground unless background, and returns its id.
func (f *FakeBrowser) OpenTab(url, title string, background bool) string {
	var id string
	f.change(func() {
		prev, hadPrev := f.tracker.Active()
		id = f.newIDLocked()
		f.tracker.Apply(TargetCreated{ID: id, URL: url, Title: title})
		if background {
			f.tracker.Apply(PageReport{ID: id})
			return
		}
		if hadPrev {
			f.tracker.Apply(PageReport{ID: prev.ID})
		}
		f.tracker.Apply(PageReport{ID: id, Visible: true, Focused: true})
	})
	return id
}

// Activate switches to tab id in the same window.
func (f *FakeBrowser) Activate(id string) {
	f.change(func() {
		if prev, ok := f.tracker.Active(); ok && prev.ID != id {
			f.tracker.Apply(PageReport{ID: prev.ID})
		}
		f.tracker.Apply(PageReport{ID: id, Visible: true, Focused: true})
	})
}

// CloseTab closes tab id; closing the last one closes Chrome, as the real
// Browser does for a windowless Chrome.
func (f *FakeBrowser) CloseTab(id string) {
	var last bool
	f.change(func() {
		f.tracker.Apply(TargetDestroyed{ID: id})
		last = f.tracker.Pages() == 0
	})
	if last {
		f.end(ErrClosedByOperator)
	}
}

// ClickPageControl clicks the in-page Go live / Pause control.
func (f *FakeBrowser) ClickPageControl() {
	f.mu.Lock()
	live := f.phase == PhaseLive
	f.mu.Unlock()
	f.SetLive(!live)
}

// QuitByHand quits Chrome as the operator would.
func (f *FakeBrowser) QuitByHand() { f.end(ErrClosedByOperator) }

var _ Browser = (*FakeBrowser)(nil)
