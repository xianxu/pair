package couchmessage

import (
	"errors"
	"fmt"
	"testing"
)

func TestRecentDeliveriesBoundedOldestOut(t *testing.T) {
	var r RecentDeliveries
	for i := 0; i < MaxRecentDeliveries+10; i++ {
		r.Add(Receipt{Message: Message{ID: fmt.Sprint(i)}, Status: Submitted})
	}
	if _, ok := r.Get("0"); ok {
		t.Fatal("kept more than the bound")
	}
	if got, ok := r.Get(fmt.Sprint(MaxRecentDeliveries + 9)); !ok || got.Status != Submitted {
		t.Fatalf("newest %+v %v", got, ok)
	}
	if len(r.byID) != MaxRecentDeliveries || len(r.order) != MaxRecentDeliveries {
		t.Fatalf("size %d/%d", len(r.byID), len(r.order))
	}
	// Re-adding an ID updates it in place.
	r.Add(Receipt{Message: Message{ID: "70"}, Status: Indeterminate})
	if got, _ := r.Get("70"); got.Status != Indeterminate || len(r.order) != MaxRecentDeliveries {
		t.Fatalf("update %+v", got)
	}
	err := error(&AlreadyCommittedError{Receipt: Receipt{Status: Submitted}})
	if !errors.Is(err, ErrAlreadyCommitted) {
		t.Fatal("typed error does not match its sentinel")
	}
}
