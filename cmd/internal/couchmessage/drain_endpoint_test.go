package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"strings"
	"testing"
)

func TestDrainEndpointFailsClosed(t *testing.T) {
	b := protocolBinding("pair:1")
	for _, o := range []DrainObservation{{Instance: "instance", Idle: true, Active: true}, {Instance: "instance", Idle: true}, {Instance: "instance", ComposerEmpty: true}} {
		if err := o.Validate(); err == nil {
			t.Fatalf("impossible evidence accepted: %+v", o)
		}
	}
	for _, op := range []string{"drain-observe", "drain-interrupt", "drain-interrupt-status", "drain-revoke"} {
		r := EndpointRequest{Binding: b, Op: op}
		if op != "drain-observe" {
			r.ID = "drain-one"
		}
		if op == "drain-interrupt" {
			r.DrainObserved = &DrainObservation{Instance: "instance", Active: true}
		}
		if err := ValidateEndpointRequest(r); err != nil {
			t.Fatal(err)
		}
		r.Sequence = 1
		if err := ValidateEndpointRequest(r); err == nil {
			t.Fatalf("%s accepted arbitrary field", op)
		}
	}
	e := RemoteEndpoint{socket: transportSocket(t), Binding: b, transport: func(_ context.Context, _ string, _ any, response any) error { return nil }}
	if _, err := e.ObserveDrain(context.Background()); err == nil {
		t.Fatal("old wrapper accepted as capable")
	}
}
func TestDrainEndpointLostReplyDoesNotRetryInput(t *testing.T) {
	starts, status := 0, 0
	e := RemoteEndpoint{socket: transportSocket(t), Binding: protocolBinding("pair:1"), transport: func(_ context.Context, _ string, request any, response any) error {
		r := request.(EndpointRequest)
		if r.Op == "drain-interrupt" {
			starts++
			return errors.New("lost reply after write")
		}
		if r.Op == "drain-interrupt-status" {
			status++
			response.(*EndpointResponse).DrainInterrupt = &DrainInterruptResult{ID: r.ID, Instance: "instance", Outcome: "interrupted", Turn: 4}
		}
		return nil
	}}
	if _, err := e.InterruptDrain(context.Background(), "drain-one", DrainObservation{Instance: "instance", Turn: 4}); err == nil {
		t.Fatal("lost response reported success")
	}
	r, err := e.DrainInterruptStatus(context.Background(), "drain-one")
	if err != nil || r.Outcome != "interrupted" || starts != 1 || status != 1 {
		t.Fatalf("%+v %v starts=%d status=%d", r, err, starts, status)
	}
}

func FuzzDrainEndpointPayload(f *testing.F) {
	f.Add([]byte(`{"Op":"drain-interrupt","ID":"drain-one","DrainObserved":{"Instance":"wrapper","Turn":1},"Binding":{}}`))
	f.Add([]byte(`{"Op":"drain-observe","DrainObserved":{"Keys":["enter"]}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var r EndpointRequest
		if strictjson.Decode(raw, &r) != nil {
			return
		}
		if ValidateEndpointRequest(r) != nil {
			return
		}
		switch r.Op {
		case "drain-observe":
			if r.ID != "" || r.DrainObserved != nil {
				t.Fatal("read admitted input payload")
			}
		case "drain-interrupt":
			if r.DrainObserved == nil || r.DrainObserved.Validate() != nil {
				t.Fatal("unfenced interrupt admitted")
			}
		case "drain-interrupt-status", "drain-revoke":
			if r.DrainObserved != nil || r.Message != nil || r.Drive != nil {
				t.Fatal("non-input command admitted payload")
			}
		}
	})
}

func TestDrainRevokeLostReplyReconcilesWithoutPaste(t *testing.T) {
	calls := 0
	effect := false
	e := RemoteEndpoint{socket: transportSocket(t), Binding: protocolBinding("pair:1"), transport: func(_ context.Context, _ string, request any, response any) error {
		r := request.(EndpointRequest)
		if r.Op != "drain-revoke" {
			t.Fatalf("unexpected mutation %s", r.Op)
		}
		calls++
		if !effect {
			effect = true
			return errors.New("lost revoke reply")
		}
		response.(*EndpointResponse).DrainRevoke = &DrainRevokeResult{ID: r.ID, Instance: "instance", Outcome: "not-committed"}
		return nil
	}}
	if _, err := e.RevokeDrain(context.Background(), "notice-one"); err == nil {
		t.Fatal("lost reply accepted")
	}
	r, err := e.RevokeDrain(context.Background(), "notice-one")
	if err != nil || r.Outcome != "not-committed" || calls != 2 {
		t.Fatalf("%+v %v calls=%d", r, err, calls)
	}
}

func TestDrainInstanceWireAndRetainedIdentity(t *testing.T) {
	m := endpointMessage()
	m.Origin = OriginSystem
	m.DrainInstance = "wrapper-original"
	request := EndpointRequest{Op: "commit", Binding: m.To, ID: m.ID, Message: &m}
	if err := ValidateEndpointRequest(request); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var decoded EndpointRequest
	if err := strictjson.Decode(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Message.DrainInstance != m.DrainInstance {
		t.Fatal("wire dropped drain instance")
	}
	if !matchingReceipt(Receipt{Message: *decoded.Message}, m) {
		t.Fatal("wire instance no longer matches")
	}
	changed := *decoded.Message
	changed.DrainInstance = "wrapper-replacement"
	if matchingReceipt(Receipt{Message: changed}, m) {
		t.Fatal("retained outcome crossed wrapper instance")
	}
	if err := ValidateBatch([]Message{m}, m.To, m.ID); err != nil {
		t.Fatal(err)
	}
	m.Origin = OriginPeer
	if ValidateEndpointRequest(request) == nil {
		t.Fatal("peer admitted system-only instance fence")
	}
	if ValidateBatch([]Message{m}, m.To, m.ID) == nil {
		t.Fatal("batch peer admitted system-only instance fence")
	}
	m.DrainInstance = ""
	raw, err = json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "DrainInstance") {
		t.Fatal("ordinary peer wire gained drain field")
	}
}

func TestDrainRevokeRequiresWrapperInstance(t *testing.T) {
	for _, instance := range []string{"", "bad\x1binstance", "replacement"} {
		t.Run(instance, func(t *testing.T) {
			e := RemoteEndpoint{socket: transportSocket(t), Binding: protocolBinding("pair:1"), transport: func(_ context.Context, _ string, request any, response any) error {
				r := request.(EndpointRequest)
				response.(*EndpointResponse).DrainRevoke = &DrainRevokeResult{ID: r.ID, Instance: instance, Outcome: "not-committed"}
				return nil
			}}
			r, err := e.RevokeDrain(context.Background(), "notice-one")
			if instance != "replacement" {
				if err == nil {
					t.Fatal("unfenced revocation accepted")
				}
				return
			}
			if err != nil || r.Instance != "replacement" {
				t.Fatalf("%+v %v", r, err)
			}
		})
	}
}
