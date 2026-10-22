package remotesession

import "testing"

func TestTailnetLossRevokesBeforeCleanup(t *testing.T) {
	for _, phase := range []remotePhase{phaseOpening, phaseReady} {
		prior := remoteState{phase: phase, attempt: 7, epoch: 9}
		next, effects := ReduceRemote(prior, remoteEvent{kind: eventTailnetUnavailable, attempt: 7})
		if next.phase != phaseKilled || next.joined {
			t.Fatalf("phase %v: loss retained admission or reset authority: %+v", phase, next)
		}
		if len(effects) != 3 || effects[0].kind != effectRevoke || effects[1].kind != effectStoreKilled || effects[2].kind != effectClose {
			t.Fatalf("phase %v: revocation must precede persistence and cleanup: %+v", phase, effects)
		}
		late, effects := ReduceRemote(next, remoteEvent{kind: eventOpened, attempt: 7})
		if late.phase != phaseKilled || len(effects) != 1 || effects[0].kind != effectDispose {
			t.Fatalf("late readiness after auth loss was adopted: %+v %+v", late, effects)
		}
		reset, effects := ReduceRemote(late, remoteEvent{kind: eventConsoleReset})
		if reset.phase != phaseKilled || len(effects) != 0 {
			t.Fatal("auth loss allowed reset before exact cleanup joined")
		}
	}
}

func TestTailnetLossCannotPoisonNewEpochOrClearKill(t *testing.T) {
	for _, phase := range []remotePhase{phaseOff, phasePreparing, phaseOpening, phaseReady, phaseStopping, phaseKilled, phaseResetting, phaseBlocked} {
		prior := remoteState{phase: phase, attempt: 12, epoch: 8}
		next, effects := ReduceRemote(prior, remoteEvent{kind: eventTailnetUnavailable, attempt: 11})
		if next != prior || len(effects) != 0 {
			t.Fatalf("old watch event changed phase %v: %+v %+v", phase, next, effects)
		}
	}
	prior := remoteState{phase: phaseKilled, attempt: 12, epoch: 8, joined: true}
	next, effects := ReduceRemote(prior, remoteEvent{kind: eventTailnetUnavailable, attempt: 12})
	if next != prior || len(effects) != 0 {
		t.Fatal("repeated auth-loss event changed settled kill authority")
	}
}
