package couchcore

// SlotInventoryObservation is filesystem inventory, never proof permitting a
// process launch. Empty Address means there is no readable current conversation.
type SlotInventoryObservation struct {
	Identity SlotIdentity
	Address  ThreadAddress
	Err      error
}

func projectSlotRows(rows []ActionableThreadSummary, slots []SlotInventoryObservation) []ActionableThreadSummary {
	for i := range rows {
		rows[i].Target = ThreadTarget{Kind: ThreadTargetOrdinary, Address: rows[i].Address}
		rows[i].RowKey, _ = rows[i].Target.RowKey()
	}
	for _, slot := range slots {
		target := ThreadTarget{Kind: ThreadTargetSlot, Slot: slot.Identity}
		key, err := target.RowKey()
		if err != nil {
			continue
		}
		index := -1
		if slot.Address != (ThreadAddress{}) {
			for i := range rows {
				if rows[i].Address == slot.Address {
					index = i
					break
				}
			}
		}
		if index < 0 {
			reason := ReasonNeverStarted
			if slot.Err != nil {
				reason = ReasonUnreadable
			}
			rows = append(rows, ActionableThreadSummary{Address: slot.Address, State: ThreadUnusable, Reason: reason, Layout: LayoutUnknown})
			index = len(rows) - 1
		}
		rows[index].Target, rows[index].RowKey = target, key
		rows[index].StartingPath = slot.Identity.WorktreeRoot
		rows[index].WorkingPath = slot.Identity.WorktreeRoot
	}
	return rows
}
