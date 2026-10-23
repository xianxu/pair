package vmcmd

import "testing"

func TestAdvanceCompletesOnlyAfterStopAndRelease(t *testing.T) {
	s := NewRunState()
	steps := []struct {
		event Event
		want  Effect
	}{
		{Event{Kind: Begin}, Acquire},
		{Event{Kind: Acquired}, Prepare},
		{Event{Kind: Prepared}, Start},
		{Event{Kind: Started}, AwaitGuest},
		{Event{Kind: GuestReady}, Execute},
		{Event{Kind: CommandEnded, Exit: 0}, Collect},
		{Event{Kind: Collected}, Stop},
		{Event{Kind: Stopped}, Release},
		{Event{Kind: Released}, Complete},
	}
	for _, step := range steps {
		var effect Effect
		var err error
		s, effect, err = Advance(s, step.event)
		if err != nil || effect != step.want {
			t.Fatalf("event %v: effect=%v err=%v, want %v", step.event.Kind, effect, err, step.want)
		}
		if step.want != Complete && s.Done() {
			t.Fatal("run completed before confirmed stop and resource release")
		}
	}
	if !s.Done() || s.ExitCode() != 0 {
		t.Fatalf("completed result: %+v", s)
	}
}

func TestAdvanceFailurePreservesCommandExitThroughCleanup(t *testing.T) {
	s := stateAt(t, Execute)
	s, effect, err := Advance(s, Event{Kind: CommandEnded, Exit: 23})
	if err != nil || effect != Collect {
		t.Fatalf("command result: %v %v", effect, err)
	}
	for _, want := range []Effect{Stop, Release, Complete} {
		s, effect, err = Advance(s, Event{Kind: Failed, Detail: "cleanup unavailable"})
		if err != nil || effect != want || s.ExitCode() != 23 {
			t.Fatalf("cleanup: result=%+v effect=%v err=%v", s, effect, err)
		}
	}
}

func TestAdvanceCancellationRetainsOwnershipUntilCleanup(t *testing.T) {
	for _, at := range []Effect{Acquire, Prepare, Start, AwaitGuest, Execute, Collect, Stop, Release} {
		t.Run(string(at), func(t *testing.T) {
			s := stateAt(t, at)
			next, effect, err := Advance(s, Event{Kind: Cancelled})
			if err != nil || next.ExitCode() != 130 {
				t.Fatalf("cancel: %+v %v", next, err)
			}
			if at == Acquire {
				if effect != Complete || !next.Done() {
					t.Fatal("unacquired position should finish without cleanup")
				}
			} else if next.Done() || effect == Release || effect == Complete {
				t.Fatal("cancellation skipped owned cleanup")
			}
		})
	}
}

func TestAdvanceRejectsOutOfOrderWithoutMutation(t *testing.T) {
	s := stateAt(t, Start)
	for _, kind := range []EventKind{Acquired, GuestReady, CommandEnded, Collected, Stopped, Released} {
		next, _, err := Advance(s, Event{Kind: kind})
		if err == nil || next != s {
			t.Fatalf("accepted or mutated out-of-order event %v: %+v %v", kind, next, err)
		}
	}
}

func FuzzAdvance(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8})
	f.Add([]byte{0, 1, 9, 6, 7, 8})
	f.Fuzz(func(t *testing.T, events []byte) {
		s := NewRunState()
		for _, b := range events {
			next, effect, err := Advance(s, Event{Kind: EventKind(b % 11), Exit: int(b % 32)})
			if err != nil {
				if next != s {
					t.Fatal("refused transition mutated state")
				}
				continue
			}
			if effect == Complete && s.phase != phaseIdle && s.phase != phaseAcquiring && s.phase != phaseReleasing {
				t.Fatalf("completion bypassed resource lifetime: %+v", s)
			}
			s = next
		}
	})
}

func stateAt(t *testing.T, want Effect) RunState {
	t.Helper()
	s := NewRunState()
	for _, event := range []Event{{Kind: Begin}, {Kind: Acquired}, {Kind: Prepared}, {Kind: Started}, {Kind: GuestReady}, {Kind: CommandEnded}, {Kind: Collected}, {Kind: Stopped}} {
		next, effect, err := Advance(s, event)
		if err != nil {
			t.Fatal(err)
		}
		s = next
		if effect == want {
			return s
		}
	}
	t.Fatalf("cannot reach %v", want)
	return s
}
