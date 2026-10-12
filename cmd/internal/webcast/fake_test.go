package webcast

import (
	"errors"
	"testing"
)

func TestFakeBrowserOperatorFlow(t *testing.T) {
	f := NewFakeBrowser()
	if s := f.State(); s.Phase != PhaseIdle || !s.HasTab || s.Tab.URL != "about:blank" {
		t.Fatalf("start %+v", s)
	}
	if err := f.SetLive(true); err != nil || f.State().Phase != PhaseLive {
		t.Fatalf("SetLive: %v %+v", err, f.State())
	}
	f.ClickPageControl()
	if f.State().Phase != PhaseIdle {
		t.Fatalf("page control did not pause: %+v", f.State())
	}
	f.Navigate("http://localhost:3000/", "Shop")
	if tab := f.State().Tab; tab.URL != "http://localhost:3000/" || tab.Title != "Shop" {
		t.Fatalf("navigate %+v", tab)
	}
	first := f.State().Tab.ID
	bg := f.OpenTab("https://docs", "Docs", true)
	if f.State().Tab.ID != first {
		t.Fatalf("background tab %s took over", bg)
	}
	fg := f.OpenTab("about:blank", "", false)
	if f.State().Tab.ID != fg {
		t.Fatalf("foreground tab not active: %+v", f.State().Tab)
	}
	f.Activate(first)
	if f.State().Tab.ID != first {
		t.Fatalf("activate: %+v", f.State().Tab)
	}
	f.Activate(bg)
	if f.State().Tab.ID != bg {
		t.Fatalf("activate background tab: %+v", f.State().Tab)
	}
}

func TestFakeBrowserEnds(t *testing.T) {
	t.Run("quit by hand", func(t *testing.T) {
		f := NewFakeBrowser()
		states, cancel := f.Subscribe()
		defer cancel()
		f.QuitByHand()
		<-f.Done()
		if !errors.Is(f.Err(), ErrClosedByOperator) {
			t.Fatalf("err %v", f.Err())
		}
		var last State
		for s := range states {
			last = s
		}
		if last.Phase != PhaseEnded {
			t.Fatalf("last state %+v", last)
		}
		if err := f.SetLive(true); !errors.Is(err, ErrEnded) {
			t.Fatalf("SetLive after end: %v", err)
		}
	})
	t.Run("close", func(t *testing.T) {
		f := NewFakeBrowser()
		if f.Err() != nil {
			t.Fatal("err while running")
		}
		f.Close()
		f.Close()
		<-f.Done()
		if f.Err() != nil || f.Stats().Closes != 2 {
			t.Fatalf("err %v stats %+v", f.Err(), f.Stats())
		}
	})
	t.Run("last tab closed", func(t *testing.T) {
		f := NewFakeBrowser()
		f.CloseTab(f.State().Tab.ID)
		<-f.Done()
		if !errors.Is(f.Err(), ErrClosedByOperator) {
			t.Fatalf("err %v", f.Err())
		}
	})
}

func TestFakeBrowserFailNextSetLive(t *testing.T) {
	f := NewFakeBrowser()
	boom := errors.New("boom")
	f.FailNextSetLive(boom)
	if err := f.SetLive(true); !errors.Is(err, boom) || f.State().Phase != PhaseIdle {
		t.Fatalf("err %v state %+v", err, f.State())
	}
	if err := f.SetLive(true); err != nil || f.State().Phase != PhaseLive {
		t.Fatalf("second: %v %+v", err, f.State())
	}
}
