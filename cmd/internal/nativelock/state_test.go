package nativelock

import "testing"

func step(t *testing.T, s State, e Event) (State, Effects) {
	t.Helper()
	next, effects, err := Advance(s, e)
	if err != nil {
		t.Fatal(err)
	}
	return next, effects
}
func TestLifecycleKnownSetAndFailure(t *testing.T) {
	var initial State
	k := testKey(t, "one")
	other := testKey(t, "two")
	s, fx := step(t, initial, Event{Kind: Discovered, Key: k})
	if s.Phase() != Acquiring || fx.Acquire == nil || *fx.Acquire != k || initial.Phase() != Unknown {
		t.Fatal(s, fx)
	}
	duplicate, fx := step(t, s, Event{Kind: Discovered, Key: k})
	if fx != (Effects{}) || duplicate.Phase() != Acquiring {
		t.Fatal(duplicate, fx)
	}
	held, fx := step(t, s, Event{Kind: Acquired, Key: k})
	if held.Phase() != Held || !held.Known(k) || fx != (Effects{}) || s.Known(k) {
		t.Fatal(held, fx)
	}
	s, fx = step(t, held, Event{Kind: Discovered, Key: k})
	if fx != (Effects{}) {
		t.Fatal(fx)
	}
	s, fx = step(t, s, Event{Kind: Discovered, Key: other})
	if fx.Acquire == nil || !s.Known(k) || held.Phase() != Held {
		t.Fatal(s, fx)
	}
	s, fx = step(t, s, Event{Kind: Failed, Key: other})
	if s.Phase() != Stopping || !fx.Stop || fx.Release || !s.Known(k) {
		t.Fatal(s, fx)
	}
	s, fx = step(t, s, Event{Kind: CleanupComplete})
	if s.Phase() != Exited || !fx.Release || s.Known(k) {
		t.Fatal(s, fx)
	}
	_, fx = step(t, s, Event{Kind: CleanupComplete})
	if fx != (Effects{}) {
		t.Fatal(fx)
	}
}
func TestStopRejectsDiscoveryAndRetainsLateAcquire(t *testing.T) {
	for _, cleanupFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "acquire-first", true: "cleanup-first"}[cleanupFirst], func(t *testing.T) {
			k := testKey(t, "pending")
			s, _ := step(t, State{}, Event{Kind: Discovered, Key: k})
			s, fx := step(t, s, Event{Kind: Stop})
			if !fx.Stop || fx.Release {
				t.Fatal(fx)
			}
			if _, _, err := Advance(s, Event{Kind: Discovered, Key: testKey(t, "late")}); err == nil {
				t.Fatal("late discovery accepted")
			}
			if cleanupFirst {
				s, fx = step(t, s, Event{Kind: CleanupComplete})
				if !fx.Release {
					t.Fatal(fx)
				}
			}
			s, fx = step(t, s, Event{Kind: Acquired, Key: k})
			if cleanupFirst {
				if s.Phase() != Exited || !fx.Release || s.Known(k) {
					t.Fatal(s, fx)
				}
			} else {
				if s.Phase() != Stopping || fx.Release || !s.Known(k) {
					t.Fatal(s, fx)
				}
				s, fx = step(t, s, Event{Kind: CleanupComplete})
				if !fx.Release || s.Phase() != Exited {
					t.Fatal(s, fx)
				}
			}
		})
	}
}
func TestLifecycleInvalidEventsAndImmutability(t *testing.T) {
	k := testKey(t, "pending")
	s, _ := step(t, State{}, Event{Kind: Discovered, Key: k})
	for _, event := range []Event{{Kind: EventKind(99)}, {Kind: Acquired, Key: testKey(t, "unrequested")}, {Kind: Discovered}, {Kind: CleanupComplete}} {
		if _, _, err := Advance(s, event); err == nil {
			t.Fatalf("accepted %+v", event)
		}
	}
	held, _ := step(t, s, Event{Kind: Acquired, Key: k})
	// Advance must copy its private sets: branching a state cannot mutate its input.
	stopped, _ := step(t, held, Event{Kind: Stop})
	step(t, stopped, Event{Kind: CleanupComplete})
	if !held.Known(k) || s.Known(k) {
		t.Fatal("input state mutated")
	}
}
