package couchtty

import "github.com/xianxu/pair/cmd/internal/couchcore"

func menuRowKey(row couchcore.ActionableThreadSummary) couchcore.ThreadRowKey {
	if row.RowKey != (couchcore.ThreadRowKey{}) {
		return row.RowKey
	}
	return couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: row.Address}
}
func selectMenuRow(frame *MenuFrame, row couchcore.ActionableThreadSummary) {
	frame.SelectedAddress = row.Address
	frame.SelectedKey = menuRowKey(row)
}
func menuThreadTarget(state MenuState, key couchcore.ThreadRowKey, address couchcore.ThreadAddress) (couchcore.ActionableThreadSummary, bool) {
	if key.Kind == couchcore.ThreadTargetSlot {
		for _, row := range menuRows(state) {
			if menuRowKey(row) == key {
				return row, true
			}
		}
		return couchcore.ActionableThreadSummary{}, false
	}
	return menuThread(state, address)
}
func dispatchMenuRow(state MenuState, operation string, row couchcore.ActionableThreadSummary) (MenuState, []MenuEffect) {
	if operation == "open-slot" || operation == "fresh-slot" {
		next, effects := dispatchMenuOperation(state, MenuEffect{Operation: operation, Args: map[string]string{"path": row.Target.Slot.WorktreeRoot}}, row.Address)
		if len(effects) > 0 {
			next.InFlight.RowKey = menuRowKey(row)
		}
		return next, effects
	}
	return dispatchThreadOperation(state, operation, row.Address)
}
func slotFreshOffered(row couchcore.ActionableThreadSummary) bool {
	return row.Target.Kind == couchcore.ThreadTargetSlot && !row.Live() && !row.Detached() && row.State != couchcore.ThreadBusy
}
