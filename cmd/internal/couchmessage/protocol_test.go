package couchmessage

import (
	"context"
	"testing"
	"time"
)

func protocolBinding(slot string) Binding {
	return Binding{Slot: slot, Repository: "/repo", Scope: "scope", Tag: slot, Session: "session", Nonce: "launch", Agent: "claude", Version: "1", PID: 123, Start: "start"}
}

func TestValidateRequestClosedShapes(t *testing.T) {
	caller := Request{Scope: "scope", Tag: "pair:0", Session: "session", Nonce: "launch"}
	valid := []Request{}
	for _, op := range []string{"actors", "status", "send"} {
		r := caller
		r.Op = op
		if op != "actors" {
			r.ID = "id"
		}
		if op == "send" {
			r.Target = "pair:1"
			r.Body = "work"
		}
		valid = append(valid, r)
	}
	binding := protocolBinding("pair:0")
	valid = append(valid, Request{Op: "register", Binding: &binding}, Request{Op: "operator-submit", Binding: &binding})
	for _, r := range valid {
		if err := ValidateRequest(r); err != nil {
			t.Errorf("%s: %v", r.Op, err)
		}
	}
	bad := []Request{{Op: "unknown"}, {Op: "actors"}, {Op: "register"}, {Op: "operator-submit"}}
	for _, r := range valid {
		r.Target = "unexpected"
		if r.Op == "send" {
			r.Binding = &binding
		}
		bad = append(bad, r)
	}
	r := caller
	r.Op = "send"
	r.ID = "id"
	r.Target = "pair:1"
	r.Body = "\x1bcommand"
	bad = append(bad, r)
	for _, r := range bad {
		if err := ValidateRequest(r); err == nil {
			t.Errorf("accepted invalid %#v", r)
		}
	}
}

func TestProtocolHandleRegistrationAndCaller(t *testing.T) {
	b := NewBroker(context.Background(), time.Now, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	binding := protocolBinding("pair:0")
	called := false
	r := Handle(context.Background(), b, Request{Op: "register", Binding: &binding}, func(_ context.Context, got Binding) error {
		called = true
		if got != binding {
			t.Fatal("binding changed")
		}
		return nil
	})
	if r.Code != "ok" || !called {
		t.Fatalf("register: %#v", r)
	}
	r = Handle(context.Background(), b, Request{Op: "actors", Scope: binding.Scope, Tag: binding.Tag, Session: binding.Session, Nonce: binding.Nonce}, nil)
	if r.Code != "unavailable" {
		t.Fatalf("unregistered caller accepted: %#v", r)
	}
	r = Handle(context.Background(), b, Request{Op: "operator-submit", Binding: &binding}, nil)
	if r.Code == "ok" {
		t.Fatal("unverified wrapper event accepted")
	}
}

func TestProtocolHandleSendStatusAndNoFree(t *testing.T) {
	b := NewBroker(context.Background(), time.Now, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	from, to := protocolBinding("brain:0"), protocolBinding("pair:1")
	for _, v := range []Binding{from, to} {
		if err := b.Register(v, newFakeEndpoint(time.Now().Add(-time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	request := Request{Op: "send", Scope: from.Scope, Tag: from.Tag, Session: from.Session, Nonce: from.Nonce, ID: "id", Target: to.Slot, Body: "work"}
	result := Handle(context.Background(), b, request, nil)
	if result.Code != "accepted" || result.Receipt == nil || result.Receipt.Message.To != to {
		t.Fatalf("send: %#v", result)
	}
	request.Op = "status"
	request.Target = ""
	request.Body = ""
	result = Handle(context.Background(), b, request, nil)
	if result.Code != "ok" || result.Receipt == nil || result.Receipt.Message.Body != "work" {
		t.Fatalf("status: %#v", result)
	}
	request.Op = "send"
	request.ID = "second"
	request.Target = to.Slot
	request.Body = "work"
	if result := Handle(context.Background(), b, request, nil); result.Code != "busy" {
		t.Fatalf("busy: %#v", result)
	}
	request.Target = "missing"
	if result := Handle(context.Background(), b, request, nil); result.Code != "not-dispatched" {
		t.Fatalf("no-free: %#v", result)
	}
	request.Nonce = "replacement"
	if result := Handle(context.Background(), b, request, nil); result.Code != "unavailable" {
		t.Fatalf("stale caller: %#v", result)
	}
}
