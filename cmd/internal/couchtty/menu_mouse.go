package couchtty

import "github.com/xianxu/pair/cmd/internal/couchcore"

// Selection acknowledges a press; only a selected, visible root row activates.
// Chips may reveal their current target, while stale list hits never may.
func reduceSwitcherRowClick(next MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	thread, ok := menuThreadTarget(next, event.RowKey, event.Address)
	if !ok || len(next.Frames) == 0 || enterOperationFor(thread) == "" || !menuRowSelectable(next, thread.Address) {
		return next, nil
	}
	root := &next.Frames[0]
	visible := menuMouseRowVisible(next, *root, thread)
	selected := root.SelectedKey == menuRowKey(thread)
	activate := len(next.Frames) == 1 && visible && selected
	if !visible && !event.RevealTarget {
		return next, nil
	}
	if event.RevealTarget && !visible {
		// Remove only restrictions that hide this target, preserving the rest.
		if !menuMouseRowVisible(next, MenuFrame{GroupScope: root.GroupScope}, thread) {
			root.GroupScope = nil
		}
		if !menuMouseRowVisible(next, MenuFrame{Filter: root.Filter}, thread) {
			root.Filter = ""
		}
		if !menuMouseRowVisible(next, *root, thread) {
			root.View = MenuViewNormal
		}
	}
	if activate {
		return activateMouseMenuRow(next, event)
	}
	next.Frames = next.Frames[:1]
	selectMenuRow(&next.Frames[0], thread)
	return next, nil
}

func menuMouseRowVisible(state MenuState, frame MenuFrame, target couchcore.ActionableThreadSummary) bool {
	for _, row := range visibleMenuRows(state, frame) {
		if menuRowKey(row) == menuRowKey(target) {
			return true
		}
	}
	return false
}
