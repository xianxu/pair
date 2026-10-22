package remotesession

import (
	"math/rand/v2"
	"testing"
)

func readyState(t *testing.T) remoteState {
	t.Helper()
	s, _ := ReduceRemote(remoteState{}, remoteEvent{kind: eventOn})
	if s.phase != phasePreparing {
		t.Fatal("start did not prepare")
	}
	s, effects := ReduceRemote(s, remoteEvent{kind: eventArmed, attempt: s.attempt})
	if s.phase != phaseOpening || len(effects) != 1 || effects[0].kind != effectOpen {
		t.Fatal("durable arm must precede open")
	}
	s, _ = ReduceRemote(s, remoteEvent{kind: eventOpened, attempt: s.attempt})
	if s.phase != phaseReady {
		t.Fatal("matching open not adopted")
	}
	return s
}

func TestRemoteStateKillFencesLateOpen(t *testing.T) {
	s, effects := ReduceRemote(remoteState{}, remoteEvent{kind: eventOn})
	if len(effects) != 1 || effects[0].kind != effectArm {
		t.Fatal("start must only arm")
	}
	old := s.attempt
	s, _ = ReduceRemote(s, remoteEvent{kind: eventKill})
	s, effects = ReduceRemote(s, remoteEvent{kind: eventArmed, attempt: old})
	if s.phase != phaseKilled || len(effects) != 0 {
		t.Fatal("late arm restored authority")
	}
	s, effects = ReduceRemote(s, remoteEvent{kind: eventOpened, attempt: old})
	if s.phase != phaseKilled || len(effects) != 1 || effects[0].kind != effectDispose {
		t.Fatal("late handle was not disposed")
	}
}

func TestRemoteStateStopCannotClearConcurrentKill(t *testing.T) {
	s := readyState(t)
	s, _ = ReduceRemote(s, remoteEvent{kind: eventOff})
	old := s.attempt
	s, _ = ReduceRemote(s, remoteEvent{kind: eventJoined, attempt: old})
	if s.phase != phaseStopping {
		t.Fatal("join alone cannot grant clean off")
	}
	s, _ = ReduceRemote(s, remoteEvent{kind: eventKill})
	s, _ = ReduceRemote(s, remoteEvent{kind: eventOffStored, attempt: old})
	if s.phase != phaseKilled {
		t.Fatal("ordinary off overwrote kill")
	}
}

func TestRemoteStateResetRequiresJoinedCleanup(t *testing.T) {
	s := readyState(t)
	s, _ = ReduceRemote(s, remoteEvent{kind: eventKill})
	s, effects := ReduceRemote(s, remoteEvent{kind: eventConsoleReset})
	if len(effects) != 0 || s.phase != phaseKilled {
		t.Fatal("reset before join")
	}
	s, _ = ReduceRemote(s, remoteEvent{kind: eventJoined, attempt: s.attempt})
	s, effects = ReduceRemote(s, remoteEvent{kind: eventConsoleReset})
	if len(effects) != 1 || effects[0].kind != effectReset || s.phase != phaseResetting {
		t.Fatal("joined reset did not publish")
	}
	s, _ = ReduceRemote(s, remoteEvent{kind: eventFailure, attempt: s.attempt})
	if s.phase == phaseOff || s.phase == phaseReady {
		t.Fatal("failed reset granted authority")
	}
}

func TestRemoteStateGeneratedKillInvariant(t *testing.T) {
	// The oracle is deliberately independent: only an acknowledged console reset
	// may lift a kill fence; a new start never grants until arm and open finish.
	rng := rand.New(rand.NewPCG(759, 731))
	for trial := 0; trial < 500; trial++ {
		var s remoteState
		killed := false
		resetting := false
		for step := 0; step < 100; step++ {
			kind := eventKind(rng.IntN(int(eventPoisoned) + 1))
			attempt := s.attempt
			if rng.IntN(4) == 0 && attempt > 0 {
				attempt--
			}
			if kind == eventKill {
				killed = true
				resetting = false
			}
			if kind == eventConsoleReset && s.phase == phaseKilled && s.joined {
				resetting = true
			}
			if kind == eventResetStored && s.phase == phaseResetting && attempt == s.attempt && resetting {
				killed = false
				resetting = false
			}
			prior := s
			next, effects := ReduceRemote(s, remoteEvent{kind: kind, attempt: attempt})
			if killed && (next.phase == phaseOff || next.phase == phaseReady || next.phase == phasePreparing || next.phase == phaseOpening) {
				t.Fatalf("kill fence lost: prior=%+v event=%d next=%+v", prior, kind, next)
			}
			if next.phase == phaseReady && prior.phase != phaseReady && !(prior.phase == phaseOpening && kind == eventOpened && attempt == prior.attempt) {
				t.Fatal("ready without current opening")
			}
			for _, e := range effects {
				if e.kind == effectOpen && !(prior.phase == phasePreparing && kind == eventArmed && attempt == prior.attempt) {
					t.Fatal("open before durable arm")
				}
			}
			s = next
		}
	}
}

func TestRemoteStateDuplicateCompletionKeepsOwnedHandle(t *testing.T) {
	s := readyState(t)
	for _, kind := range []eventKind{eventArmed, eventOpened} {
		next, effects := ReduceRemote(s, remoteEvent{kind: kind, attempt: s.attempt})
		if next != s || len(effects) != 0 {
			t.Fatalf("duplicate %d changed live owner: %+v", kind, effects)
		}
	}
}

func TestRemoteStateIdleKillStillRequiresDurableJoin(t *testing.T) {
	s := remoteState{phase: phaseOff, joined: true}
	s, _ = ReduceRemote(s, remoteEvent{kind: eventKill})
	s, effects := ReduceRemote(s, remoteEvent{kind: eventConsoleReset})
	if s.phase != phaseKilled || s.joined || len(effects) != 0 {
		t.Fatal("idle kill reset before its persistent write joined")
	}
}

func TestRemoteStateFailureDisposition(t *testing.T) {
	for _, tc := range []struct{ from, want remotePhase }{
		{phasePreparing, phaseBlocked}, {phaseOpening, phaseKilled}, {phaseReady, phaseKilled}, {phaseStopping, phaseBlocked}, {phaseResetting, phaseBlocked},
	} {
		next, effects := ReduceRemote(remoteState{phase: tc.from, attempt: 2}, remoteEvent{kind: eventFailure, attempt: 2})
		if next.phase != tc.want || len(effects) < 2 {
			t.Fatalf("failure from=%d next=%+v effects=%v", tc.from, next, effects)
		}
	}
}
