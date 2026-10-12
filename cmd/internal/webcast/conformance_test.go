package webcast

import (
	"errors"
	"net/url"
	"testing"
	"time"
)

// harness drives a Browser the way the operator would. The same scenario runs
// against FakeBrowser and real Chrome, so the fake is checked against the
// thing it models.
type harness interface {
	browser() Browser
	navigate(t *testing.T, title string)
	newTab(t *testing.T, title string) string
	activate(t *testing.T, id string)
	clickControl(t *testing.T)
	quitByHand(t *testing.T)
}

// page is a document whose title is title.
func page(title string) string {
	return "data:text/html," + url.PathEscape("<title>"+title+"</title><p>"+title)
}

// waitState waits for b's state to satisfy ok.
func waitState(t *testing.T, b Browser, what string, ok func(State) bool) State {
	t.Helper()
	states, cancel := b.Subscribe()
	defer cancel()
	deadline := time.After(15 * time.Second)
	last := b.State()
	for {
		if ok(last) {
			return last
		}
		select {
		case s, open := <-states:
			if !open {
				if ok(b.State()) {
					return b.State()
				}
				t.Fatalf("%s: ended at %+v", what, last)
			}
			last = s
		case <-deadline:
			t.Fatalf("%s: timed out at %+v", what, last)
		}
	}
}

func runConformance(t *testing.T, h harness) {
	b := h.browser()
	first := waitState(t, b, "start idle with a tab", func(s State) bool { return s.Phase == PhaseIdle && s.HasTab })
	if err := b.SetLive(true); err != nil {
		t.Fatal(err)
	}
	waitState(t, b, "live", func(s State) bool { return s.Phase == PhaseLive })
	h.navigate(t, "one")
	waitState(t, b, "navigated", func(s State) bool { return s.Tab.Title == "one" && s.Tab.ID == first.Tab.ID })
	two := h.newTab(t, "two")
	waitState(t, b, "new tab active", func(s State) bool { return s.Tab.ID == two && s.Tab.Title == "two" })
	h.activate(t, first.Tab.ID)
	waitState(t, b, "switched back", func(s State) bool { return s.Tab.ID == first.Tab.ID && s.Tab.Title == "one" })
	h.clickControl(t)
	waitState(t, b, "page control paused", func(s State) bool { return s.Phase == PhaseIdle })
	h.clickControl(t)
	waitState(t, b, "page control resumed", func(s State) bool { return s.Phase == PhaseLive })
	h.quitByHand(t)
	select {
	case <-b.Done():
	case <-time.After(15 * time.Second):
		t.Fatal("quit by hand: not done")
	}
	if !errors.Is(b.Err(), ErrClosedByOperator) {
		t.Fatalf("quit by hand: err %v", b.Err())
	}
	if b.State().Phase != PhaseEnded {
		t.Fatalf("final state %+v", b.State())
	}
	if err := b.SetLive(true); !errors.Is(err, ErrEnded) {
		t.Fatalf("SetLive after end: %v", err)
	}
}

type fakeHarness struct{ f *FakeBrowser }

func (h fakeHarness) browser() Browser                    { return h.f }
func (h fakeHarness) navigate(t *testing.T, title string) { h.f.Navigate(page(title), title) }
func (h fakeHarness) newTab(t *testing.T, title string) string {
	return h.f.OpenTab(page(title), title, false)
}
func (h fakeHarness) activate(t *testing.T, id string) { h.f.Activate(id) }
func (h fakeHarness) clickControl(t *testing.T)        { h.f.ClickPageControl() }
func (h fakeHarness) quitByHand(t *testing.T)          { h.f.QuitByHand() }

func TestConformanceFake(t *testing.T) {
	runConformance(t, fakeHarness{NewFakeBrowser()})
}
