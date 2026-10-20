package couchmessage

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDrainEndpointEpochFenceRequiredAndEchoed(t *testing.T) {
	fence := drainFenceFixture(t, 1, DrainInterruptControl)
	for _, op := range []string{"interrupt", "status", "revoke", "reserve", "release"} {
		t.Run(op, func(t *testing.T) {
			for _, mutate := range []func(*DrainFence){func(f *DrainFence) { f.Namespace = strings.Repeat("a", 64) }, func(f *DrainFence) { f.Generation = "00000000-0000-4000-8000-999999999999" }, func(f *DrainFence) { f.Epoch++ }, func(f *DrainFence) { f.Instance = "replacement" }, func(f *DrainFence) {
				if f.Kind == DrainInterruptControl {
					f.Kind = DrainNoticeControl
				} else {
					f.Kind = DrainInterruptControl
				}
			}} {
				calls := 0
				e := RemoteEndpoint{Namespace: "/test/couch", socket: transportSocket(t), Binding: protocolBinding("pair:1"), transport: func(_ context.Context, _ string, request any, response any) error {
					calls++
					r := request.(EndpointRequest)
					if r.DrainFence == nil {
						t.Fatal("request omitted typed fence")
					}
					wrong := *r.DrainFence
					mutate(&wrong)
					out := response.(*EndpointResponse)
					out.DrainFence = &wrong
					out.DrainInterrupt = &DrainInterruptResult{ID: r.ID, Fence: wrong, Outcome: "interrupted"}
					out.DrainRevoke = &DrainRevokeResult{ID: r.ID, Fence: wrong, Outcome: "not-committed"}
					return nil
				}}
				f := fence
				var err error
				switch op {
				case "interrupt":
					_, err = e.InterruptDrain(context.Background(), "id", f, DrainObservation{Protocol: 2, Namespace: f.Namespace, Instance: f.Instance})
				case "status":
					_, err = e.DrainInterruptStatus(context.Background(), "id", f)
				case "revoke":
					_, err = e.RevokeDrain(context.Background(), "id", f)
				case "reserve":
					f.Kind = DrainNoticeControl
					err = e.ReserveDrain(context.Background(), "id", 1, f)
				case "release":
					f.Kind = DrainNoticeControl
					err = e.ReleaseDrain(context.Background(), "id", f)
				}
				if err == nil || calls != 1 {
					t.Fatalf("wrong returned fence accepted or retried op=%s err=%v calls=%d", op, err, calls)
				}
			}
		})
	}
}
func TestDrainProtocolTwoNegotiationAndComparableMessage(t *testing.T) {
	f := drainFenceFixture(t, 1, DrainNoticeControl)
	e := RemoteEndpoint{Namespace: "/test/couch", socket: transportSocket(t), Binding: protocolBinding("pair:1"), transport: func(_ context.Context, _ string, _ any, response any) error {
		response.(*EndpointResponse).DrainObservation = &DrainObservation{Protocol: 1, Namespace: f.Namespace, Instance: f.Instance}
		return nil
	}}
	if _, err := e.ObserveDrain(context.Background()); err == nil {
		t.Fatal("protocol1 silently downgraded epoch authority")
	}
	m := endpointMessage()
	m.Deadline = m.Deadline.Round(0).UTC()
	m.Origin = OriginSystem
	m.DrainFence = f
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Message
	if err := json.Unmarshal(raw, &decoded); err != nil || decoded != m {
		t.Fatalf("fence content equality failed %v", err)
	}
	decoded.Origin = OriginPeer
	if ValidateDrainInstance(decoded) == nil {
		t.Fatal("peer adopted drain authority")
	}
	decoded.DrainFence = DrainFence{}
	raw, err = json.Marshal(decoded)
	if err != nil || strings.Contains(string(raw), "DrainFence") {
		t.Fatalf("optional fence changed ordinary wire %s %v", raw, err)
	}
}
