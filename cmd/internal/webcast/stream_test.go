package webcast

import "testing"

func TestViewerStateSaysNothingWhileNotLive(t *testing.T) {
	tab := Tab{ID: "t", URL: "https://app/private", Title: "Inbox (3)"}
	for _, p := range []Phase{PhaseIdle, PhaseEnded} {
		if got := viewerState(State{Phase: p, Tab: tab, HasTab: true}); got != (wireState{}) {
			t.Fatalf("%v: %+v", p, got)
		}
	}
	if got := viewerState(State{Phase: PhaseLive, Tab: tab, HasTab: true}); got != (wireState{Live: true, URL: tab.URL, Title: tab.Title}) {
		t.Fatalf("live: %+v", got)
	}
}
