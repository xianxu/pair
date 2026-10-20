package couchmessage

import (
	"testing"
)

func TestDrainRequestClosedShape(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	r := Request{Op: "drain", ID: id, Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Drain: &DrainRequest{Action: "start", TimeoutMillis: 600000}}
	if err := ValidateRequest(r); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Request{
		{Op: "drain", Drain: &DrainRequest{Action: "start", TimeoutMillis: 600000}},
		{Op: "drain", Drain: &DrainRequest{Action: "status", Fast: true}},
		{Op: "drain", Drain: &DrainRequest{Action: "ack", ID: id}},
		{Op: "actors", Drain: &DrainRequest{Action: "status"}},
	} {
		if ValidateRequest(bad) == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	r.Body = "payload"
	if ValidateRequest(r) == nil {
		t.Fatal("accepted extraneous body")
	}
	if err := ValidateRequest(Request{Op: "drain", Drain: &DrainRequest{Action: "status"}}); err != nil {
		t.Fatal(err)
	}
}
