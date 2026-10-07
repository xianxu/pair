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

// dispatchMenuRow sends a row's operation. Resume and reboot on a slot row
// address the slot by its host checkout: its record may be missing, unreadable
// or replaced under the row, so the tag is not what names it. Its in-flight
// operation is then keyed by row, not by an address the operation replaces.
func dispatchMenuRow(state MenuState, operation string, row couchcore.ActionableThreadSummary) (MenuState, []MenuEffect) {
	if row.Target.Kind == couchcore.ThreadTargetSlot && actorOperation(operation) {
		next, effects := dispatchMenuOperation(state, MenuEffect{Operation: operation, Args: couchcore.ActorOperationArgs(row, operation)}, row.Address)
		if len(effects) > 0 {
			next.InFlight.RowKey = menuRowKey(row)
		}
		return next, effects
	}
	return dispatchThreadOperation(state, operation, row.Address)
}

// actorOperation names the operations whose arguments come from
// couchcore.ActorOperationArgs.
func actorOperation(operation string) bool {
	return operation == "resume" || operation == "reboot" || operation == "reap"
}

// menuOperationReplacesAddress names an in-flight operation whose success may
// hand back a different address than the one it was sent for: anything keyed
// by row (a slot's record can be adopted or replaced), and reboot, which
// retires its record and starts a fresh one under a new tag (as recover may).
func menuOperationReplacesAddress(origin MenuOperationOrigin) bool {
	return origin.RowKey.Kind == couchcore.ThreadTargetSlot || origin.Operation == "reboot" || origin.Operation == "recover"
}

// menuFrameTargetsInFlight reports a frame bound to the row the in-flight
// operation was sent for: by row key when the operation is keyed by row, else
// by exact address.
func menuFrameTargetsInFlight(state MenuState, frame MenuFrame) bool {
	in := state.InFlight
	if in.Operation == "" {
		return false
	}
	if in.RowKey.Kind == couchcore.ThreadTargetSlot {
		return in.RowKey == frame.RowKey
	}
	return in.Address == frame.Thread
}

// menuFrameOperationInFlight is the frame whose OWN operation is running:
// same target and same action.
func menuFrameOperationInFlight(state MenuState, frame MenuFrame) bool {
	return state.InFlight.Operation == frame.Action && menuFrameTargetsInFlight(state, frame)
}
