package couchmessage

import (
	"context"
	"errors"
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

// Legacy wrappers still send register/operator-submit every second; both are
// answered unsupported and register nothing (#365).
func TestProtocolHandleRefusesLegacyRegistration(t *testing.T) {
	b := NewBroker(context.Background(), time.Now, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	binding := protocolBinding("pair:0")
	for _, op := range []string{"register", "operator-submit"} {
		if r := Handle(context.Background(), b, Request{Op: op, Binding: &binding}); r.Code != "unsupported" {
			t.Fatalf("%s: %#v", op, r)
		}
	}
	r := Handle(context.Background(), b, Request{Op: "actors", Scope: binding.Scope, Tag: binding.Tag, Session: binding.Session, Nonce: binding.Nonce})
	if r.Code != "unavailable" {
		t.Fatalf("legacy registration made a caller: %#v", r)
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
	result := Handle(context.Background(), b, request)
	if result.Code != "accepted" || result.Receipt == nil || result.Receipt.Message.To != to {
		t.Fatalf("send: %#v", result)
	}
	request.Op = "status"
	request.Target = ""
	request.Body = ""
	result = Handle(context.Background(), b, request)
	if result.Code != "ok" || result.Receipt == nil || result.Receipt.Message.Body != "work" {
		t.Fatalf("status: %#v", result)
	}
	request.Op = "send"
	request.ID = "second"
	request.Target = to.Slot
	request.Body = "work"
	if result := Handle(context.Background(), b, request); result.Code != "busy" {
		t.Fatalf("busy: %#v", result)
	}
	request.Target = "missing"
	if result := Handle(context.Background(), b, request); result.Code != "not-dispatched" {
		t.Fatalf("no-free: %#v", result)
	}
	request.Nonce = "replacement"
	if result := Handle(context.Background(), b, request); result.Code != "unavailable" {
		t.Fatalf("stale caller: %#v", result)
	}
}

// TestValidateSlotOperationRequests pins the closed shapes of the slot
// operation requests (pair#367 M2), one strategy per malformation.
func TestValidateSlotOperationRequests(t *testing.T) {
	caller := Request{Scope: "scope", Tag: "pair:0", Session: "session", Nonce: "launch"}
	with := func(mutate func(*Request)) Request {
		r := caller
		mutate(&r)
		return r
	}
	valid := []Request{
		with(func(r *Request) { r.Op, r.ID, r.Target = "resume", "id", "pair:1" }),
		with(func(r *Request) { r.Op, r.ID, r.Target = "resume", "id", "pair:0" }),
		with(func(r *Request) { r.Op, r.ID, r.Target, r.Confirmed = "reboot", "id", "pair:1", true }),
		with(func(r *Request) { r.Op, r.ID, r.Target = "reboot", "id", "pair:1" }), // confirmation is the handler's check
		with(func(r *Request) { r.Op, r.ID = "operation-status", "id" }),
	}
	for _, r := range valid {
		if err := ValidateRequest(r); err != nil {
			t.Errorf("%s %q: %v", r.Op, r.Target, err)
		}
	}
	for _, c := range []struct {
		name string
		r    Request
		is   error
	}{
		{"resume without an ID", with(func(r *Request) { r.Op, r.Target = "resume", "pair:1" }), nil},
		{"resume with a family target", with(func(r *Request) { r.Op, r.ID, r.Target = "resume", "id", "pair" }), ErrInvalidTarget},
		{"reboot with a family target", with(func(r *Request) { r.Op, r.ID, r.Target = "reboot", "id", "pair" }), ErrInvalidTarget},
		{"resume with a malformed slot", with(func(r *Request) { r.Op, r.ID, r.Target = "resume", "id", "pair:01" }), ErrInvalidTarget},
		{"resume without a target", with(func(r *Request) { r.Op, r.ID = "resume", "id" }), ErrInvalidTarget},
		{"resume with a body", with(func(r *Request) { r.Op, r.ID, r.Target, r.Body = "resume", "id", "pair:1", "work" }), nil},
		{"resume with an agent", with(func(r *Request) { r.Op, r.ID, r.Target, r.Agent = "resume", "id", "pair:1", "claude" }), nil},
		{"resume without a caller", Request{Op: "resume", ID: "id", Target: "pair:1"}, nil},
		{"confirmation on a send", with(func(r *Request) { r.Op, r.ID, r.Target, r.Body, r.Confirmed = "send", "id", "pair:1", "work", true }), nil},
		{"confirmation on a status", with(func(r *Request) { r.Op, r.ID, r.Confirmed = "operation-status", "id", true }), nil},
		{"operation-status without an ID", with(func(r *Request) { r.Op = "operation-status" }), nil},
		{"operation-status with a target", with(func(r *Request) { r.Op, r.ID, r.Target = "operation-status", "id", "pair:1" }), nil},
		{"operation-status with a body", with(func(r *Request) { r.Op, r.ID, r.Body = "operation-status", "id", "x" }), nil},
	} {
		err := ValidateRequest(c.r)
		if err == nil || c.is != nil && !errors.Is(err, c.is) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// pair#421: the relaunch overrides are refused on any other operation.
func TestValidateRequestOverridesOnlyOnRelaunch(t *testing.T) {
	base := Request{Scope: "scope", Tag: "pair:0", Session: "session", Nonce: "launch",
		Op: "relaunch", ID: "id", Target: "pair:1", Confirmed: true, SameBinary: true, ForceUnknown: true}
	if err := ValidateRequest(base); err != nil {
		t.Fatalf("relaunch with overrides: %v", err)
	}
	for _, op := range []string{"resume", "reboot", "send", "reload-context"} {
		r := base
		r.Op = op
		if err := ValidateRequest(r); err == nil {
			t.Fatalf("%s accepted --same-binary", op)
		}
	}
	reload := base
	reload.Op, reload.SameBinary = "reload-context", false
	if err := ValidateRequest(reload); err != nil {
		t.Fatalf("reload-context with --force-unknown: %v", err)
	}
}
