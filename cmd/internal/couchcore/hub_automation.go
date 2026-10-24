package couchcore

import "fmt"

// HubAutomation is volatile policy for configured canonical hub roots. Values
// never change through observations or user input; only explicit actions apply.
type HubAutomation struct{ hubs map[string]bool }

// HubAutomationStatus is the explicit control's view, not a lifecycle authority.
// Startup and MissingHubs describe the original startup observation, not current
// attachment or health; explicit recovery does not rewrite that history.
type HubAutomationStatus struct {
	Hub         string   `json:"hub"`
	Enabled     bool     `json:"enabled"`
	Startup     string   `json:"startup,omitempty"`
	MissingHubs []string `json:"missing_hubs,omitempty"`
}

func NewHubAutomation(initial map[string]bool) (HubAutomation, error) {
	if len(initial) == 0 || len(initial) > MaxHubSlots {
		return HubAutomation{}, fmt.Errorf("automation requires 1..%d configured hubs", MaxHubSlots)
	}

	hubs := make(map[string]bool, len(initial))
	for root, on := range initial {
		if root == "" {
			return HubAutomation{}, fmt.Errorf("automation hub is empty")
		}
		hubs[root] = on
	}
	return HubAutomation{hubs: hubs}, nil
}

// AdvanceHubAutomation is the sole policy transition. A returned value owns its
// map so concurrent readers can keep an earlier snapshot unchanged.
func AdvanceHubAutomation(state HubAutomation, hub, action string) (HubAutomation, error) {
	if _, err := AutomationStatus(state, hub); err != nil {
		return state, err
	}
	var on bool
	switch action {
	case "on":
		on = true
	case "off":
		on = false
	default:
		return state, fmt.Errorf("automation action must be on or off")
	}
	next := make(map[string]bool, len(state.hubs))
	for root, enabled := range state.hubs {
		next[root] = enabled
	}
	next[hub] = on
	return HubAutomation{hubs: next}, nil
}

func AutomationStatus(state HubAutomation, hub string) (HubAutomationStatus, error) {
	on, ok := state.hubs[hub]
	if !ok {
		return HubAutomationStatus{}, fmt.Errorf("automation hub is not configured")
	}
	return HubAutomationStatus{Hub: hub, Enabled: on}, nil
}
