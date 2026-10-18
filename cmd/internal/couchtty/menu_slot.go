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
	if row.Target.Kind != couchcore.ThreadTargetSlot || row.Target.Validate() != nil {
		if row.Address.RepoScope == "" || row.Address.Tag == "" {
			state.Notice = errorMenuNotice("no exact conversation or valid slot target is available")
			return state, nil
		}
		// An invalid slot cannot supply a path; only its exact native address
		// remains usable. A validated slot can still recover without a record.
		row.Target = couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: row.Address}
		row.RowKey = couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: row.Address}
	}
	if view, owned := passViewOf(state.Reattach, row.Address); owned && view.State == PassFailed &&
		couchcore.ContinuityOutcome(row) == couchcore.Continue && (operation == "switch" || operation == "resume") {
		// A failed listed-live row may look live without a local client. Its
		// one-click retry attaches that exact retained address; it cannot cold
		// resume or start fresh if the listing changed since projection.
		effect := threadEffect("resume", row.Address)
		effect.Args["warm-only"] = "true"
		effect.RowKey = menuRowKey(row)
		next, effects := dispatchMenuOperation(state, effect, row.Address)
		return next, effects
	}
	// Non-live relaunch and starting-row details first observe recovery evidence.
	// They never dispatch a lifecycle operation merely by opening the preview.
	if operation == "resume-candidates" || menuRowNeedsRecoveryPreview(row, operation) {
		return openRecoveryPreview(state, row)
	}

	if row.Target.Kind == couchcore.ThreadTargetSlot && actorOperation(operation) {
		next, effects := dispatchMenuOperation(state, MenuEffect{Operation: operation, RowKey: menuRowKey(row), Args: couchcore.ActorOperationArgs(row, operation)}, row.Address)
		return next, effects
	}
	return dispatchThreadOperation(state, operation, row.Address)
}

// actorOperation names the operations whose arguments come from
// couchcore.ActorOperationArgs.
func actorOperation(operation string) bool {
	return operation == couchcore.OpRecoverPark || operation == "resume" || operation == "reboot" || operation == "reap" || operation == "recover"
}

// menuOperationReplacesAddress names an in-flight operation whose success may
// hand back a different address than the one it was sent for: anything keyed
// by row (a slot's record can be adopted or replaced), and reboot, which
// retires its record and starts a fresh one under a new tag (as recover may).
func menuOperationReplacesAddress(origin MenuOperationOrigin) bool {
	return origin.RowKey.Kind == couchcore.ThreadTargetSlot || origin.Operation == "recovery-confirmation" || origin.Operation == "reboot" || origin.Operation == "recover"
}

// menuFrameTargetsInFlight reports a frame bound to the row the in-flight
// operation was sent for: by row key when the operation is keyed by row, else
// by exact address.
func menuFrameTargetsInFlight(state MenuState, frame MenuFrame) bool {
	in := menuFrameRunningOrigin(state, frame)
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
	return menuFrameRunningOrigin(state, frame).Operation == frame.Action && menuFrameTargetsInFlight(state, frame)
}

func menuRowNeedsRecoveryPreview(row couchcore.ActionableThreadSummary, operation string) bool {
	return operation == "relaunch" && !row.Live()
}

func menuFrameRunningOrigin(state MenuState, frame MenuFrame) MenuOperationOrigin {
	for _, job := range state.Operations {
		origin := job.Origin
		if job.Phase != MenuOperationRunning {
			continue
		}
		if origin.RowKey.Kind == couchcore.ThreadTargetSlot {
			if origin.RowKey == frame.RowKey {
				return origin
			}
		} else if origin.Address == frame.Thread {
			return origin
		}
	}
	return MenuOperationOrigin{}
}
