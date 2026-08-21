package couchcore

import "testing"

func kinds(ms []Message) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Kind
	}
	return out
}

func TestEnqueueCollapsesKeepingTheLatestAtTheTail(t *testing.T) {
	// Asserting identity AND position: an Enqueue that silently dropped the
	// incoming message would leave the length unchanged and pass a
	// length-only check.
	q := []Message{{Kind: "status", Body: "old"}, {Kind: "note", Body: "a"}}
	out, dropped, ok := Enqueue(q, Message{Kind: "status", Body: "new"}, 8)
	if !ok || dropped.Kind != "" {
		t.Fatalf("ok=%v dropped=%+v; a collapse is not a drop", ok, dropped)
	}
	if got := kinds(out); len(got) != 2 || got[1] != "status" {
		t.Fatalf("kinds = %v, want the collapsed status at the tail", got)
	}
	if out[1].Body != "new" {
		t.Fatalf("body = %q, want the newest", out[1].Body)
	}
}

func TestEnqueueDropsOldestNonControlAtCapacity(t *testing.T) {
	q := []Message{{Kind: "stop", Control: true}, {Kind: "n1"}}
	out, dropped, ok := Enqueue(q, Message{Kind: "n2"}, 2)
	if ok {
		t.Fatal("a drop must report ok=false so the caller can be loud")
	}
	if dropped.Kind != "n1" {
		t.Fatalf("dropped = %q, want the oldest non-control entry", dropped.Kind)
	}
	if got := kinds(out); len(got) != 2 || got[0] != "stop" || got[1] != "n2" {
		t.Fatalf("kinds = %v; control must survive, newest must land", got)
	}
}

func TestEnqueueNeverDropsControlEvenOverCapacity(t *testing.T) {
	// Control carries stop and deadline. Dropping one to honour a capacity
	// number would trade a real obligation for a bookkeeping one.
	q := []Message{{Kind: "stop", Control: true}, {Kind: "deadline", Control: true}}
	out, dropped, ok := Enqueue(q, Message{Kind: "budget", Control: true}, 2)
	if len(out) != 3 {
		t.Fatalf("len = %d; control messages must not be dropped", len(out))
	}
	if dropped.Kind != "" {
		t.Fatalf("dropped = %q; nothing should have been dropped", dropped.Kind)
	}
	if ok {
		t.Fatal("exceeding capacity must still report ok=false -- it is a bug signal")
	}
}

func TestEnqueueUnderCapacityIsClean(t *testing.T) {
	out, dropped, ok := Enqueue(nil, Message{Kind: "n1"}, 4)
	if !ok || dropped.Kind != "" || len(out) != 1 {
		t.Fatalf("out=%v dropped=%+v ok=%v", kinds(out), dropped, ok)
	}
}

func TestEnqueueDoesNotMutateTheInputSlice(t *testing.T) {
	q := []Message{{Kind: "n1"}, {Kind: "n2"}}
	_, _, _ = Enqueue(q, Message{Kind: "n3"}, 2)
	if got := kinds(q); got[0] != "n1" || got[1] != "n2" {
		t.Fatalf("input mutated to %v", got)
	}
}
