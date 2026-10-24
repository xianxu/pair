package couchtty

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// SetAutomationControl installs the memory-only automation owner. Actions are
// status, on, off, and toggle; toggle atomically changes the owner's state and
// returns its resulting value. The callback must never perform external IO.
// Console invokes it outside its mutex, only after an explicit switcher key.
func (c *Console) SetAutomationControl(fn func(hubRoot string, action string) (bool, error)) {
	c.mu.Lock()
	c.automationControl = fn
	c.mu.Unlock()
}

func selectedAutomationHub(state MenuState) (string, error) {
	if len(state.Frames) == 0 || len(state.HubOwnership.Hubs()) == 0 {
		return "", fmt.Errorf("no verified hub selected")
	}
	frame := state.CurrentFrame()
	key, address := state.Frames[0].SelectedKey, state.Frames[0].SelectedAddress
	if menuFrameBindsThread(frame) {
		key, address = frame.RowKey, frame.Thread
	}
	row, ok := menuThreadTarget(state, key, address)
	if !ok {
		return "", fmt.Errorf("no verified hub selected")
	}
	var slot couchcore.HubSlot
	if row.Target.Kind == couchcore.ThreadTargetSlot {
		if err := row.Target.Validate(); err != nil {
			return "", fmt.Errorf("selected slot is invalid: %w", err)
		}
		slot = couchcore.HubSlot{Repository: row.Target.Slot.PrimaryRoot, Number: row.Target.Slot.Number}
	} else {
		root := presentationRoot(row.StartingPath, row.Address.RepoScope)
		if root == "" {
			return "", fmt.Errorf("selected row has no verified repository")
		}
		if state.HubOwnership.IsHub(root) {
			return root, nil
		}
		slot = couchcore.HubSlot{Repository: root}
	}
	resolved := state.HubOwnership.Resolve(slot)
	if len(resolved.Conflicts) > 0 {
		return "", fmt.Errorf("selected row has conflicting hub ownership")
	}
	if !state.HubOwnership.IsHub(resolved.Hub) {
		return "", fmt.Errorf("selected row has no configured hub")
	}
	return resolved.Hub, nil
}

func (c *Console) toggleSelectedAutomation() {
	c.mu.Lock()
	hub, err := selectedAutomationHub(c.menu)
	control := c.automationControl
	c.mu.Unlock()
	if err == nil && control == nil {
		err = fmt.Errorf("control unavailable")
	}
	var enabled bool
	if err == nil {
		enabled, err = control(hub, "toggle")
	}
	text := ""
	if err != nil {
		text = fmt.Sprintf("hub automation: %v", err)
	} else {
		state := "off"
		if enabled {
			state = "on"
		}
		text = fmt.Sprintf("%s: hub automation %s", hub, state)
	}
	c.reduceMenu(MenuEvent{Kind: MenuEventNotice, Error: text, Success: err == nil})
}
