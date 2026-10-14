package couchmessage

import "testing"

func TestHubMoveProtocolAndReceiptIdentity(t *testing.T) {
	r := Request{Op: "move-slot", Scope: "s", Tag: "t", Session: "session", Nonce: "nonce", ID: "00000000-0000-4000-8000-000000000000", Target: "pair:7,pair:8", ToHub: "bench:0", Persist: true}
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Request{func() Request { x := r; x.Target = "pair"; return x }(), func() Request { x := r; x.ToHub = ""; return x }(), func() Request { x := r; x.Op = "actors"; return x }()} {
		if ValidateRequest(bad) == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	e := ReceiptEvent{Kind: ReceiptAdmit, ID: r.ID, Op: r.Op, Target: r.Target, ToHub: r.ToHub, Persist: true}
	got, err := ApplyReceiptEvent(OperationReceipt{}, e)
	if err != nil {
		t.Fatal(err)
	}
	e.Persist = false
	if _, err := ApplyReceiptEvent(got, e); err != ErrReceiptIDConflict {
		t.Fatalf("mode conflict %v", err)
	}
	e.Persist = true
	e.ToHub = "ops:0"
	if _, err := ApplyReceiptEvent(got, e); err != ErrReceiptIDConflict {
		t.Fatalf("target conflict %v", err)
	}
}
