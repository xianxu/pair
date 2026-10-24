package couchcore

import "testing"

func TestHubAutomationExplicitAndIndependent(t *testing.T) {
	state, err := NewHubAutomation(map[string]bool{"/ops": false, "/bench": true})
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{"", "submit", "connect", "message", "tick", "restart"} {
		if _, err := AdvanceHubAutomation(state, "/ops", event); err == nil {
			t.Fatalf("non-explicit event %q changed automation", event)
		}
	}
	for _, action := range []string{"on", "on", "off", "off", "on"} {
		state, err = AdvanceHubAutomation(state, "/ops", action)
		if err != nil {
			t.Fatal(err)
		}
		status, err := AutomationStatus(state, "/ops")
		if err != nil || status.Enabled != (action == "on") {
			t.Fatalf("%s: %+v %v", action, status, err)
		}
		bench, err := AutomationStatus(state, "/bench")
		if err != nil || !bench.Enabled {
			t.Fatalf("cross-hub change: %+v %v", bench, err)
		}
	}
	if _, err := AdvanceHubAutomation(state, "/unknown", "on"); err == nil {
		t.Fatal("unconfigured hub accepted")
	}
}

func TestHubAutomationOwnsSnapshots(t *testing.T) {
	initial := map[string]bool{"/ops": false}
	state, err := NewHubAutomation(initial)
	if err != nil {
		t.Fatal(err)
	}
	initial["/ops"] = true
	next, err := AdvanceHubAutomation(state, "/ops", "on")
	if err != nil {
		t.Fatal(err)
	}
	old, _ := AutomationStatus(state, "/ops")
	current, _ := AutomationStatus(next, "/ops")
	if old.Enabled || !current.Enabled {
		t.Fatalf("aliased states: old=%+v current=%+v", old, current)
	}
}
