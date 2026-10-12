package webcast

import "testing"

func TestViewerStateSaysNothingWhileNotLive(t *testing.T) {
	tab := Tab{ID: "t", URL: "https://app/private", Title: "Inbox (3)"}
	for _, st := range []State{{Phase: PhaseIdle, Tab: tab, HasTab: true}, {Phase: PhaseEnded, Tab: tab, HasTab: true}} {
		if got := viewerState(st); got != (wireState{}) {
			t.Fatalf("%v: %+v", st.Phase, got)
		}
	}
	if got := viewerState(State{Phase: PhaseLive, Tab: tab, HasTab: true}); got != (wireState{Live: true, URL: tab.URL, Title: tab.Title}) {
		t.Fatalf("live: %+v", got)
	}
}
