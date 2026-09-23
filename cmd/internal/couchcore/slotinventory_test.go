package couchcore

import (
	"errors"
	"testing"
)

func TestSlotInventoryRetainsMissingAndCorruptRows(t *testing.T) {
	first := allocationCandidate(1).Identity
	second := allocationCandidate(2).Identity
	input := ThreadProjectionInput{Slots: []SlotInventoryObservation{{Identity: first}, {Identity: second, Err: errors.New("invalid JSON")}}}
	rows := ProjectActionableThreads(input)
	if len(rows) != 2 {
		t.Fatalf("rows: %+v", rows)
	}
	for i, row := range rows {
		if row.Target.Kind != ThreadTargetSlot || row.RowKey.SlotPath == "" || row.Address != (ThreadAddress{}) || row.State != ThreadUnusable {
			t.Fatalf("row %d: %+v", i, row)
		}
	}
	if rows[0].RowKey == rows[1].RowKey {
		t.Fatal("addressless slots collapsed")
	}
	diagnostic := BuildThreadInventory(input)
	if len(diagnostic) != 2 || diagnostic[0].RowKey != rows[0].RowKey {
		t.Fatalf("diagnostic differs: %+v", diagnostic)
	}
}

func TestSlotInventoryUsesOneStableRowAcrossConversations(t *testing.T) {
	slot := allocationCandidate(1).Identity
	var key ThreadRowKey
	for _, tag := range []ThreadTag{"old", "replacement"} {
		address := ThreadAddress{RepoScope: "scope", Tag: tag}
		input := ThreadProjectionInput{Unreadable: []ThreadAddress{address}, Slots: []SlotInventoryObservation{{Identity: slot, Address: address}}}
		rows := ProjectActionableThreads(input)
		if len(rows) != 1 || rows[0].Address != address {
			t.Fatalf("duplicate/lost conversation: %+v", rows)
		}
		if key != (ThreadRowKey{}) && rows[0].RowKey != key {
			t.Fatal("replacement moved selection")
		}
		key = rows[0].RowKey
	}
	ordinary := ProjectActionableThreads(ThreadProjectionInput{Unreadable: []ThreadAddress{{RepoScope: "scope", Tag: "ordinary"}}})
	if len(ordinary) != 1 || ordinary[0].Target.Kind != ThreadTargetOrdinary || ordinary[0].RowKey.Address != ordinary[0].Address {
		t.Fatalf("ordinary: %+v", ordinary)
	}
}
