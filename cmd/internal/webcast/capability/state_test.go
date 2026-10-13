package capability

import (
	"math"
	"testing"
)

func TestNextCapabilityTransitions(t *testing.T) {
	off := CapabilityState{Generation: 4, Attempt: 2}
	on := CapabilityState{Phase: On, Generation: 4, Attempt: 2}
	running := CapabilityState{Phase: Disabling, Generation: 4, Attempt: 2, Outcome: Running}
	retryable := running
	retryable.Outcome = Retryable
	broken := running
	broken.Outcome = Broken
	ended := CapabilityState{Phase: Ended, Generation: 4, Attempt: 2}
	states := []CapabilityState{off, on, running, retryable, broken, ended}
	events := []Event{{Kind: Enable}, {Kind: Disable}, {Kind: Retry}, {Kind: CleanupSucceeded, Attempt: 2}, {Kind: CleanupFailed, Attempt: 2, Outcome: Retryable}, {Kind: CleanupFailed, Attempt: 2, Outcome: Broken}, {Kind: End}}
	// Rows cover every phase/outcome, columns every valid event tag/outcome.
	// '=' is an accepted no-op; '!' refuses without changing the snapshot.
	wants := [][]string{
		{"on", "=", "!", "=", "=", "=", "ended"},
		{"=", "running", "!", "=", "=", "=", "ended"},
		{"!", "!", "!", "off", "retryable", "broken", "ended"},
		{"!", "!", "retry", "=", "=", "=", "ended"},
		{"!", "!", "!", "=", "=", "=", "ended"},
		{"!", "!", "!", "=", "=", "=", "="},
	}
	for i, s := range states {
		for j, e := range events {
			want := s
			result := wants[i][j]
			switch result {
			case "on":
				want.Phase = On
				want.Generation++
			case "running":
				want.Phase = Disabling
				want.Generation++
				want.Attempt++
				want.Outcome = Running
			case "retry":
				want.Attempt++
				want.Outcome = Running
			case "off":
				want.Phase = Off
				want.Outcome = NoCleanup
			case "retryable":
				want.Outcome = Retryable
			case "broken":
				want.Outcome = Broken
			case "ended":
				want.Phase = Ended
				want.Generation++
				want.Outcome = NoCleanup
			}
			got, err := NextCapability(s, e)
			if (err != nil) != (result == "!") || got != want {
				t.Errorf("state %+v event %+v: got %+v, %v; want %+v (%s)", s, e, got, err, want, result)
			}
		}
	}
}

func TestNextCapabilityRefusesMalformedValues(t *testing.T) {
	for _, s := range []CapabilityState{{Phase: 255}, {Outcome: Running}, {Phase: On, Outcome: Broken}, {Phase: Disabling}, {Phase: Disabling, Attempt: 1, Outcome: 255}, {Phase: Disabling, Outcome: Running}, {Phase: Ended, Outcome: Retryable}} {
		got, err := NextCapability(s, Event{Kind: End})
		if err == nil || got != s {
			t.Errorf("invalid state %+v: got %+v, %v", s, got, err)
		}
		if _, ok := s.Permit(); ok {
			t.Errorf("invalid state admitted: %+v", s)
		}
	}
	for _, e := range []Event{{Kind: 255}, {Kind: Enable, Attempt: 1}, {Kind: End, Outcome: Broken}, {Kind: CleanupSucceeded}, {Kind: CleanupSucceeded, Attempt: 1, Outcome: Retryable}, {Kind: CleanupFailed, Attempt: 1}, {Kind: CleanupFailed, Attempt: 1, Outcome: Running}, {Kind: CleanupFailed, Attempt: 1, Outcome: 255}} {
		s := CapabilityState{}
		got, err := NextCapability(s, e)
		if err == nil || got != s {
			t.Errorf("invalid event %+v: got %+v, %v", e, got, err)
		}
	}
}

func TestCleanupAttemptFencesLateCompletions(t *testing.T) {
	s := CapabilityState{}
	apply := func(e Event) {
		t.Helper()
		next, err := NextCapability(s, e)
		if err != nil {
			t.Fatal(err)
		}
		s = next
	}
	apply(Event{Kind: Enable})
	permit, ok := s.Permit()
	if !ok || !s.Admits(permit) {
		t.Fatal("On did not admit its permit")
	}
	apply(Event{Kind: Disable})
	if s.Admits(permit) {
		t.Fatal("Disable retained admission")
	}
	first := s.Attempt
	apply(Event{Kind: CleanupFailed, Attempt: first, Outcome: Retryable})
	apply(Event{Kind: Retry})
	second := s.Attempt
	apply(Event{Kind: CleanupFailed, Attempt: second, Outcome: Retryable})
	failed := s
	for _, attempt := range []uint64{first, second, second + 1} {
		apply(Event{Kind: CleanupSucceeded, Attempt: attempt})
		apply(Event{Kind: CleanupFailed, Attempt: attempt, Outcome: Broken})
		if s != failed {
			t.Fatal("late completion erased failure")
		}
	}
	apply(Event{Kind: Retry})
	apply(Event{Kind: CleanupSucceeded, Attempt: s.Attempt})
	if s.Phase != Off {
		t.Fatal("retry success must finish Off")
	}
	apply(Event{Kind: Enable})
	if s.Admits(permit) {
		t.Fatal("new On admitted old permit")
	}
	permit, _ = s.Permit()
	apply(Event{Kind: End})
	if s.Admits(permit) {
		t.Fatal("End retained admission")
	}
}

func TestCapabilityCountersNeverWrap(t *testing.T) {
	for _, tc := range []struct {
		s CapabilityState
		e Event
	}{
		{CapabilityState{Generation: math.MaxUint64}, Event{Kind: Enable}},
		{CapabilityState{Phase: On, Generation: math.MaxUint64}, Event{Kind: Disable}},
		{CapabilityState{Phase: On, Attempt: math.MaxUint64}, Event{Kind: Disable}},
		{CapabilityState{Phase: Disabling, Attempt: math.MaxUint64, Outcome: Retryable}, Event{Kind: Retry}},
		{CapabilityState{Generation: math.MaxUint64}, Event{Kind: End}},
	} {
		got, err := NextCapability(tc.s, tc.e)
		if err == nil || got != tc.s {
			t.Errorf("counter overflow: got %+v, %v", got, err)
		}
	}
}

func TestStaleCompletionsCannotSettleNewRunningAttempt(t *testing.T) {
	for _, retry := range []bool{true, false} {
		name := "after_new_disable"
		if retry {
			name = "after_retry"
		}
		t.Run(name, func(t *testing.T) {
			s := CapabilityState{}
			apply := func(e Event) {
				t.Helper()
				var err error
				s, err = NextCapability(s, e)
				if err != nil {
					t.Fatal(err)
				}
			}
			apply(Event{Kind: Enable})
			apply(Event{Kind: Disable})
			oldAttempt := s.Attempt
			if retry {
				apply(Event{Kind: CleanupFailed, Attempt: oldAttempt, Outcome: Retryable})
				apply(Event{Kind: Retry})
			} else {
				apply(Event{Kind: CleanupSucceeded, Attempt: oldAttempt})
				apply(Event{Kind: Enable})
				apply(Event{Kind: Disable})
			}
			if s.Outcome != Running || s.Attempt <= oldAttempt {
				t.Fatalf("expected new Running attempt, got %+v", s)
			}
			for _, event := range []Event{
				{Kind: CleanupSucceeded, Attempt: oldAttempt},
				{Kind: CleanupFailed, Attempt: oldAttempt, Outcome: Retryable},
				{Kind: CleanupFailed, Attempt: oldAttempt, Outcome: Broken},
			} {
				got, err := NextCapability(s, event)
				if err != nil || got != s {
					t.Errorf("stale completion %+v changed Running snapshot %+v to %+v, error %v", event, s, got, err)
				}
			}
		})
	}
}

func TestUnissuedPermitIsRefused(t *testing.T) {
	s := CapabilityState{Phase: On}
	if s.Admits(Permit{}) {
		t.Fatal("zero permit must not admit work")
	}
	p, ok := s.Permit()
	if !ok || !s.Admits(p) {
		t.Fatal("issued permit must admit work")
	}
}

func FuzzCapabilitySequences(f *testing.F) {
	f.Add([]byte{0, 2})                   // A refused retry preserves an already-On state.
	f.Add([]byte{0, 1, 4, 2, 11, 12, 28}) // Mismatched completions during a Running retry.
	f.Add([]byte{0, 1, 4, 2, 3, 0, 1, 5, 6})
	f.Add([]byte{0, 1, 4, 2, 4, 10, 2, 3, 0, 6})
	f.Fuzz(func(t *testing.T, sequence []byte) {
		if len(sequence) > 512 {
			sequence = sequence[:512]
		}
		s := CapabilityState{}
		var permits []Permit
		for _, b := range sequence {
			e := Event{Kind: EventKind(b % 8)}
			if e.Kind == CleanupSucceeded || e.Kind == CleanupFailed {
				e.Attempt = s.Attempt
				if b&8 != 0 {
					e.Attempt++
				}
				if e.Kind == CleanupFailed {
					e.Outcome = Retryable
					if b&16 != 0 {
						e.Outcome = Broken
					}
				}
			}
			next, err := NextCapability(s, e)
			if (e.Kind == CleanupSucceeded || e.Kind == CleanupFailed) && e.Attempt != s.Attempt && next != s {
				t.Fatal("mismatched completion changed state")
			}
			if err != nil && next != s {
				t.Fatal("error mutated state")
			}
			if next.Generation < s.Generation || next.Attempt < s.Attempt {
				t.Fatal("identity regressed")
			}
			if next.Phase == On && s.Phase != On && (s.Phase != Off || e.Kind != Enable) {
				t.Fatal("admission reopened without Enable")
			}
			if s.Phase == Ended && next != s {
				t.Fatal("ended resurrected")
			}
			if s.Outcome == Broken && e.Kind != End && next != s {
				t.Fatal("broken recovered without End")
			}
			if next.Attempt != s.Attempt && e.Kind != Disable && e.Kind != Retry {
				t.Fatal("completion changed attempt")
			}
			if e.Kind == Retry && err == nil && next.Phase == On {
				t.Fatal("retry enabled admission")
			}
			for _, p := range permits {
				if next.Admits(p) && (next.Phase != On || !s.Admits(p)) {
					t.Fatal("old permit resurrected")
				}
			}
			s = next
			if p, ok := s.Permit(); ok {
				if s.Phase != On || !s.Admits(p) {
					t.Fatal("invalid permit")
				}
				permits = append(permits, p)
			}
		}
	})
}
