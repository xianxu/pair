package couchcore

import "testing"

func targetTestSlot(number int) SlotIdentity {
	return SlotIdentity{Repo: "pair", RepoIdentity: "/src/pair/.git", PrimaryRoot: "/src/pair", EnvironmentRoot: "/src/worktree/pair-slot1", WorktreeRoot: "/src/worktree/pair-slot1/pair", Number: number}
}

func TestThreadTargetStableSlotKey(t *testing.T) {
	slot := targetTestSlot(1)
	target := ThreadTarget{Kind: ThreadTargetSlot, Slot: slot}
	key, err := target.RowKey()
	if err != nil {
		t.Fatal(err)
	}
	// Native conversation replacement never enters the slot selection key.
	for _, tag := range []ThreadTag{"old", "new"} {
		ordinary := ThreadTarget{Kind: ThreadTargetOrdinary, Address: ThreadAddress{RepoScope: "scope", Tag: tag}}
		ordinaryKey, err := ordinary.RowKey()
		if err != nil || ordinaryKey == key {
			t.Fatalf("native address aliases slot: %v", err)
		}
	}
	again, err := target.RowKey()
	if err != nil || again != key || key.SlotPath != slot.WorktreeRoot {
		t.Fatalf("unstable key: %+v %v", again, err)
	}
	keys := map[ThreadRowKey]bool{key: true}
	if !keys[again] {
		t.Fatal("key must be comparable")
	}
}

func TestThreadTargetRejectsAmbiguousOrInvalidChoice(t *testing.T) {
	address := ThreadAddress{RepoScope: "scope", Tag: "native"}
	for _, target := range []ThreadTarget{
		{}, {Kind: ThreadTargetKind("unknown")},
		{Kind: ThreadTargetOrdinary},
		{Kind: ThreadTargetOrdinary, Address: address, Slot: targetTestSlot(1)},
		{Kind: ThreadTargetSlot, Address: address, Slot: targetTestSlot(1)},
		{Kind: ThreadTargetSlot, Slot: targetTestSlot(0)},
	} {
		if _, err := target.RowKey(); err == nil {
			t.Fatalf("accepted %+v", target)
		}
	}
}
