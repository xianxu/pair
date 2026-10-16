package titlepoller

import "testing"

func TestTitleLifetimeTerminalOrders(t *testing.T) {
	events := []TitleEvent{TitleLeaseReady, TitleOwnerEOF, TitleSignalled, TitlePollStopped}
	var visit func(TitleLifetime, int)
	visit = func(state TitleLifetime, depth int) {
		if depth == 0 {
			return
		}
		for _, event := range events {
			next, effects := AdvanceTitleLifetime(state, event)
			if state.stopped() && effects&TitleStartPoll != 0 {
				t.Fatal("terminal owner restarted poll")
			}
			if state.phase == titleExited && next.phase != titleExited {
				t.Fatal("exited title reactivated")
			}
			visit(next, depth-1)
		}
	}
	visit(TitleLifetime{}, 5)
	active, effects := AdvanceTitleLifetime(TitleLifetime{}, TitleLeaseReady)
	if effects != TitleStartPoll {
		t.Fatalf("ready effects=%v", effects)
	}
	stopping, effects := AdvanceTitleLifetime(active, TitleOwnerEOF)
	if effects != TitleExit || stopping.phase != titleExited {
		t.Fatal("EOF did not exit immediately")
	}
}
