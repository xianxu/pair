package capability

import "testing"

func TestProjectionHasOneActionAndNoFailureEnables(t *testing.T) {
	for _, test := range []struct {
		s             CapabilityState
		state, action string
		on            bool
	}{
		{CapabilityState{}, "off", "enable", false},
		{CapabilityState{Phase: On}, "on", "disable", true},
		{CapabilityState{Phase: Disabling, Attempt: 1, Outcome: Running}, "turning-off", "wait", false},
		{CapabilityState{Phase: Disabling, Attempt: 1, Outcome: Retryable}, "retry", "retry", false},
		{CapabilityState{Phase: Disabling, Attempt: 1, Outcome: Broken}, "broken", "end-cast", false},
		{CapabilityState{Phase: Ended}, "ended", "start-cast", false},
		{CapabilityState{Phase: 99}, "broken", "end-cast", false},
	} {
		p := Project(test.s)
		if p.State != test.state || p.Action != test.action || p.Enabled != test.on {
			t.Fatalf("%+v -> %+v", test.s, p)
		}
	}
}
