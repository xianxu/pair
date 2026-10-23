package testcouchbridge

import "testing"

func TestLifecycleHappyPath(t *testing.T) {
	state := State{}
	for _, tt := range []struct {
		event  EventKind
		phase  Phase
		effect Effect
	}{{Begin, Preparing, Prepare}, {CandidateReady, Starting, Start}, {FleetReady, Checking, Check}, {EndpointReady, Publishing, Publish}, {ReadinessWritten, Held, None}, {EOF, Stopping, Stop}, {CleanupClean, Stopped, None}} {
		var effect Effect
		state, effect = Reduce(state, Event{Kind: tt.event})
		if state.Phase != tt.phase || effect != tt.effect {
			t.Fatalf("event %v: %+v effect %v", tt.event, state, effect)
		}
	}
}

func TestLifecycleGeneratedStopDominatesLateCompletions(t *testing.T) {
	completions := []EventKind{Begin, CandidateReady, FleetReady, EndpointReady, ReadinessWritten}
	for prefix := 0; prefix <= len(completions); prefix++ {
		for _, cause := range []EventKind{EOF, TERM, INT, Failed, Deadline, ProducerDied} {
			s := State{}
			for _, kind := range completions[:prefix] {
				s, _ = Reduce(s, Event{Kind: kind})
			}
			s, effect := Reduce(s, Event{Kind: cause, Failure: "original"})
			if s.Phase != Stopping || effect != Stop {
				t.Fatal("stop missing")
			}
			for _, late := range append(append([]EventKind{}, completions...), EOF, TERM, INT, Failed, Deadline, ProducerDied) {
				next, effect := Reduce(s, Event{Kind: late, Failure: "late"})
				if next != s || effect != None {
					t.Fatalf("late %v changed stopped admission: %+v -> %+v", late, s, next)
				}
			}
			s, _ = Reduce(s, Event{Kind: CleanupUncertain, Failure: "cleanup"})
			if s.Phase != Retained {
				t.Fatal("uncertain cleanup became clean")
			}
			next, effect := Reduce(s, Event{Kind: CleanupClean})
			if next != s || effect != None {
				t.Fatal("retained terminal revived")
			}
		}
	}
}

func TestLifecycleGeneratedSequencesNeverDuplicateEffects(t *testing.T) {
	// Exhaust all short input sequences, including reordering and duplicate
	// completion/failure events. The independent assertions count effects rather
	// than computing a second copy of the transition table.
	alphabet := []EventKind{Begin, CandidateReady, FleetReady, EndpointReady, ReadinessWritten, TERM, Failed, CleanupClean, CleanupUncertain}
	var visit func(State, [6]int, bool, int)
	visit = func(s State, counts [6]int, stopping bool, depth int) {
		if depth == 0 {
			return
		}
		for _, kind := range alphabet {
			next, effect := Reduce(s, Event{Kind: kind, Failure: "cause"})
			seen := counts
			if effect != None {
				seen[effect]++
				if seen[effect] > 1 {
					t.Fatalf("duplicate effect %v", effect)
				}
			}
			if stopping && effect != None {
				t.Fatalf("new effect %v after stop", effect)
			}
			if (s.Phase == Stopped || s.Phase == Retained) && next != s {
				t.Fatal("terminal changed")
			}
			if effect == Publish && seen[Prepare]+seen[Start]+seen[Check] != 3 {
				t.Fatal("publish before prerequisites")
			}
			visit(next, seen, stopping || effect == Stop, depth-1)
		}
	}
	visit(State{}, [6]int{}, false, 5)
}
