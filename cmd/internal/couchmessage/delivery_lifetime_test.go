package couchmessage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

type lifetimeEndpoint struct {
	*FakeDeliveryEndpoint
	contexts chan context.Context
}

func (e *lifetimeEndpoint) Deliver(ctx context.Context, m Message, progress func(Receipt)) (Receipt, error) {
	e.contexts <- ctx
	return e.FakeDeliveryEndpoint.Deliver(ctx, m, progress)
}

func assertDeliveryDeadline(t *testing.T, got, parent context.Context) {
	t.Helper()
	deadline, bounded := got.Deadline()
	want, parentBounded := parent.Deadline()
	if bounded != parentBounded || bounded && !deadline.Equal(want) {
		t.Errorf("accepted wait deadline=%v bounded=%v; parent=%v bounded=%v", deadline, bounded, want, parentBounded)
	}
}

func TestBrokerDeliveryLifetimeBelongsToReceiver(t *testing.T) {
	for _, boundedParent := range []bool{false, true} {
		t.Run(map[bool]string{false: "unbounded", true: "parent deadline"}[boundedParent], func(t *testing.T) {
			parent, cancel := context.WithCancel(context.Background())
			if boundedParent {
				cancel()
				parent, cancel = context.WithDeadline(context.Background(), time.Now().Add(24*time.Hour))
			}
			defer cancel()
			now := time.Now()
			b := NewBroker(parent, func() time.Time { return now }, nil)
			defer b.Close()
			from, to := brokerBinding("pair:1"), brokerBinding("ops:0")
			e := &lifetimeEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), contexts: make(chan context.Context, 1)}
			for binding, endpoint := range map[Binding]DeliveryEndpoint{from: newFakeEndpoint(now), to: e} {
				if err := b.Register(binding, endpoint); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := b.Send(context.Background(), from, "lifetime", Route{Target: to.Slot}, "report"); err != nil {
				t.Fatal(err)
			}
			var delivery context.Context
			select {
			case delivery = <-e.contexts:
			case <-testwait.After(t):
				t.Fatal("delivery never reached endpoint")
			}
			assertDeliveryDeadline(t, delivery, parent)
			b.Disconnect(to)
			select {
			case <-delivery.Done():
			case <-testwait.After(t):
				t.Fatal("receiver disconnect did not cancel delivery")
			}
		})
	}
}

func TestRemoteEndpointWaitInheritsOnlyParentLifetime(t *testing.T) {
	for _, initial := range []Status{Queued, Held} {
		for _, cancelWait := range []bool{false, true} {
			t.Run(string(initial)+map[bool]string{false: "/submit", true: "/cancel"}[cancelWait], func(t *testing.T) {
				m := endpointMessage()
				parent, cancel := context.WithCancel(context.Background())
				if cancelWait {
					cancel()
					parent, cancel = context.WithDeadline(context.Background(), time.Now().Add(24*time.Hour))
				}
				defer cancel()
				commits, polls := 0, 0
				e := RemoteEndpoint{Binding: m.To, socket: "stateful fake"}
				e.transport = func(ctx context.Context, _ string, request, response any) error {
					req := request.(EndpointRequest)
					status := initial
					switch req.Op {
					case "commit":
						commits++
						deadline, ok := ctx.Deadline()
						if !ok || !deadline.Equal(m.Deadline) {
							t.Errorf("commit lost admission deadline: %v %v", deadline, ok)
						}
					case "status":
						polls++
						assertDeliveryDeadline(t, ctx, parent)
						if cancelWait {
							cancel()
							<-ctx.Done()
							return ctx.Err()
						}
						status = Submitted
					default:
						t.Fatalf("unexpected operation %s", req.Op)
					}
					response.(*EndpointResponse).Receipt = &Receipt{Message: m, Status: status}
					return nil
				}
				r, err := e.Deliver(parent, m, nil)
				if cancelWait {
					if !errors.Is(err, context.Canceled) {
						t.Errorf("parent cancellation: %v", err)
					}
				} else if err != nil || r.Status != Submitted {
					t.Errorf("outcome=%+v error=%v", r, err)
				}
				if commits != 1 || polls != 1 {
					t.Errorf("commits=%d polls=%d", commits, polls)
				}
			})
		}
	}
}
