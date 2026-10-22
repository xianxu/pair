package couchmessage

import (
	"context"
	"testing"
	"time"
)

func TestForgetProtocolAcceptsOnlyCanonicalAbsoluteTarget(t *testing.T) {
	good := Request{Op: "session-forget", Target: "/private/session.md"}
	if err := ValidateRequest(good); err != nil {
		t.Fatalf("forget target rejected: %v", err)
	}
	for _, bad := range []Request{{Op: good.Op}, {Op: good.Op, Target: "relative.md"}, {Op: good.Op, Target: "/private/../session.md"}, {Op: good.Op, Target: good.Target, Scope: "worker"}, {Op: good.Op, Target: good.Target, Body: "hidden"}} {
		if err := ValidateRequest(bad); err == nil {
			t.Fatalf("open forget request accepted: %+v", bad)
		}
	}
}

func TestDocumentSettlementRetainsInflightCallbackAttribution(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	doc := "0123456789abcdef0123456789abcdef"
	if b.UnsettledDocument(doc) {
		t.Fatal("empty broker claims outstanding work")
	}
	b.admissions["admitting"] = &admission{provenance: messageProvenance{documentID: doc}}
	if !b.UnsettledDocument(doc) {
		t.Fatal("admission not counted")
	}
	delete(b.admissions, "admitting")
	b.receipts["recording"] = Receipt{Message: Message{ID: "recording", DocumentID: doc}, Status: Cancelled, RetainUntil: now.Add(-time.Second)}
	b.jobs["recording"] = &deliveryJob{}
	b.mu.Lock()
	b.sweep()
	b.mu.Unlock()
	if !b.UnsettledDocument(doc) {
		t.Fatal("active recording callback lost document attribution")
	}
	if _, ok, _ := b.ReceiptSnapshot("recording"); !ok {
		t.Fatal("receipt swept beneath callback")
	}
	delete(b.jobs, "recording")
	if b.UnsettledDocument(doc) {
		t.Fatal("definite completed callback remains unsettled")
	}
	b.receipts["recording"] = Receipt{Message: Message{ID: "recording", DocumentID: doc}, Status: Indeterminate}
	if !b.UnsettledDocument(doc) {
		t.Fatal("indeterminate outcome counted as settled")
	}
	if b.UnsettledDocument("fedcba9876543210fedcba9876543210") {
		t.Fatal("another document blocks exact forget")
	}
}
