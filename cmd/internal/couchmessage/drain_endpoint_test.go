package couchmessage

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func TestDrainEndpointFailsClosed(t *testing.T) {
	b := protocolBinding("pair:1")
	f := drainFenceFixture(t, 1, DrainInterruptControl)
	for _, o := range []DrainObservation{{Protocol: 2, Namespace: f.Namespace, Instance: f.Instance, Idle: true, Active: true}, {Protocol: 2, Namespace: f.Namespace, Instance: f.Instance, Idle: true}, {Protocol: 2, Namespace: f.Namespace, Instance: f.Instance, ComposerEmpty: true}} {
		if err := o.Validate(); err == nil {
			t.Fatalf("impossible evidence %+v", o)
		}
	}
	for _, op := range []string{"drain-observe", "drain-interrupt", "drain-interrupt-status", "drain-revoke", "drain-reserve", "drain-release"} {
		r := EndpointRequest{Binding: b, Op: op}
		fence := f
		if op != "drain-observe" {
			r.ID = "drain-one"
			r.DrainFence = &fence
		}
		if op == "drain-reserve" || op == "drain-release" {
			fence.Kind = DrainNoticeControl
		}
		if op == "drain-interrupt" {
			r.DrainObserved = &DrainObservation{Protocol: 2, Namespace: f.Namespace, Instance: f.Instance, Active: true}
		}
		if err := ValidateEndpointRequest(r); err != nil {
			t.Fatal(op, err)
		}
		r.Lines = 1
		if err := ValidateEndpointRequest(r); err == nil {
			t.Fatal("arbitrary field admitted", op)
		}
	}
	e := RemoteEndpoint{Namespace: "/test/couch", socket: transportSocket(t), Binding: b, transport: func(context.Context, string, any, any) error { return nil }}
	if _, err := e.ObserveDrain(context.Background()); err == nil {
		t.Fatal("old wrapper accepted as capable")
	}
}
func TestDrainEndpointLostReplyDoesNotRetryInput(t *testing.T) {
	starts, status := 0, 0
	f := drainFenceFixture(t, 1, DrainInterruptControl)
	e := RemoteEndpoint{Namespace: "/test/couch", socket: transportSocket(t), Binding: protocolBinding("pair:1"), transport: func(_ context.Context, _ string, request any, response any) error {
		r := request.(EndpointRequest)
		if r.Op == "drain-interrupt" {
			starts++
			return errors.New("lost reply after write")
		}
		if r.Op == "drain-interrupt-status" {
			status++
			response.(*EndpointResponse).DrainInterrupt = &DrainInterruptResult{ID: r.ID, Fence: f, Outcome: "interrupted", Turn: 4}
		}
		return nil
	}}
	if _, err := e.InterruptDrain(context.Background(), "drain-one", f, DrainObservation{Protocol: 2, Namespace: f.Namespace, Instance: f.Instance, Turn: 4}); err == nil {
		t.Fatal("lost response success")
	}
	r, err := e.DrainInterruptStatus(context.Background(), "drain-one", f)
	if err != nil || r.Outcome != "interrupted" || starts != 1 || status != 1 {
		t.Fatalf("%+v %v %d %d", r, err, starts, status)
	}
}
func FuzzDrainEndpointPayload(f *testing.F) {
	f.Add([]byte(`{"Op":"drain-interrupt","ID":"drain-one","DrainObserved":{"Instance":"wrapper","Turn":1},"Binding":{}}`))
	f.Add([]byte(`{"Op":"drain-observe","DrainObserved":{"Keys":["enter"]}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var r EndpointRequest
		if strictjson.Decode(raw, &r) != nil || ValidateEndpointRequest(r) != nil {
			return
		}
		switch r.Op {
		case "drain-observe":
			if r.ID != "" || r.DrainObserved != nil || r.DrainFence != nil {
				t.Fatal("read admitted mutation payload")
			}
		case "drain-interrupt":
			if r.DrainObserved == nil || r.DrainObserved.Validate() != nil || r.DrainFence == nil || r.DrainFence.Validate() != nil {
				t.Fatal("unfenced interrupt")
			}
		case "drain-interrupt-status", "drain-revoke", "drain-reserve", "drain-release":
			if r.DrainObserved != nil || r.Message != nil || r.Drive != nil || r.DrainFence == nil {
				t.Fatal("noninput admitted payload")
			}
		}
	})
}
func TestDrainRevokeLostReplyReconcilesWithoutPaste(t *testing.T) {
	calls := 0
	effect := false
	f := drainFenceFixture(t, 1, DrainNoticeControl)
	e := RemoteEndpoint{Namespace: "/test/couch", socket: transportSocket(t), Binding: protocolBinding("pair:1"), transport: func(_ context.Context, _ string, request any, response any) error {
		r := request.(EndpointRequest)
		if r.Op != "drain-revoke" {
			t.Fatal("unexpected mutation", r.Op)
		}
		calls++
		if !effect {
			effect = true
			return errors.New("lost reply")
		}
		response.(*EndpointResponse).DrainRevoke = &DrainRevokeResult{ID: r.ID, Fence: f, Outcome: "not-committed"}
		return nil
	}}
	if _, err := e.RevokeDrain(context.Background(), "notice-one", f); err == nil {
		t.Fatal("lost reply accepted")
	}
	r, err := e.RevokeDrain(context.Background(), "notice-one", f)
	if err != nil || r.Outcome != "not-committed" || calls != 2 {
		t.Fatalf("%+v %v %d", r, err, calls)
	}
}
func TestDrainInstanceWireAndRetainedIdentity(t *testing.T) {
	m := endpointMessage()
	m.Origin = OriginSystem
	m.DrainFence = drainFenceFixture(t, 1, DrainNoticeControl)
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
	if decoded.Message.DrainFence != m.DrainFence || !matchingReceipt(Receipt{Message: *decoded.Message}, m) {
		t.Fatal("wire dropped fence")
	}
	changed := *decoded.Message
	changed.DrainFence.Epoch++
	if matchingReceipt(Receipt{Message: changed}, m) {
		t.Fatal("retained outcome crossed epoch")
	}
	if err := ValidateBatch([]Message{m}, m.To, m.ID); err != nil {
		t.Fatal(err)
	}
	m.Origin = OriginPeer
	if ValidateEndpointRequest(request) == nil || ValidateBatch([]Message{m}, m.To, m.ID) == nil {
		t.Fatal("peer admitted system authority")
	}
	m.DrainFence = DrainFence{}
	raw, err = json.Marshal(m)
	if err != nil || strings.Contains(string(raw), "DrainFence") {
		t.Fatal("ordinary wire gained drain fence")
	}
}
func TestDrainRevokeRequiresWrapperInstance(t *testing.T) {
	for _, instance := range []string{"", "bad\x1binstance", "replacement"} {
		t.Run(instance, func(t *testing.T) {
			f := drainFenceFixture(t, 1, DrainNoticeControl)
			e := RemoteEndpoint{Namespace: "/test/couch", socket: transportSocket(t), Binding: protocolBinding("pair:1"), transport: func(_ context.Context, _ string, request any, response any) error {
				r := request.(EndpointRequest)
				bad := f
				bad.Instance = instance
				response.(*EndpointResponse).DrainRevoke = &DrainRevokeResult{ID: r.ID, Fence: bad, Outcome: "not-committed"}
				return nil
			}}
			if _, err := e.RevokeDrain(context.Background(), "notice-one", f); err == nil {
				t.Fatal("different/malformed instance accepted")
			}
		})
	}
}
