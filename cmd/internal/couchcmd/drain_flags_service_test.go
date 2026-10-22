package couchcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

var drainTestRequestSequence atomic.Uint64

func flagRequest(b couchmessage.Binding, action string) couchmessage.Request {
	r := callerRequest(b, "drain")
	r.Drain = &couchmessage.DrainRequest{Action: action}
	if action != "status" {
		r.ID = fmt.Sprintf("%032x", drainTestRequestSequence.Add(1))
	}
	return r
}

func TestDrainFlagsExcludeStewardAndKeepOrdinaryDelivery(t *testing.T) {
	r := newHubMoveRig(t)
	ctx := context.Background()
	for _, b := range []couchmessage.Binding{r.callers[1], r.worker} {
		if got := r.s.handle(ctx, flagRequest(b, "start")); got.Code == "ok" {
			t.Fatalf("non-steward %s mutated flags", b.Slot)
		}
	}
	got := r.s.handle(ctx, flagRequest(r.callers[0], "start"))
	if got.Code != "accepted" || got.Operation == nil || got.Operation.Drain == nil || len(got.Operation.Drain.Children) != len(r.callers) {
		t.Fatalf("drain: %+v", got)
	}
	for _, row := range r.s.handle(ctx, flagRequest(r.callers[0], "status")).Drain.Rows {
		if row.Workspace == r.callers[0].Slot {
			t.Fatal("issuing steward flagged")
		}
	}
	children := got.Operation.Drain.Children
	for _, child := range children {
		testwait.Until(t, "ordinary authenticated system notice admitted", func() bool {
			receipt, err := r.s.broker.Status(r.callers[0], child.ID)
			return err == nil && receipt.Message.Origin == couchmessage.OriginSystem
		})
	}

	msg := callerRequest(r.callers[0], "send")
	msg.ID, msg.Target, msg.Body = "00000000-0000-4000-8000-000000000002", r.worker.Slot, "ordinary work remains deliverable"
	if got := r.s.handle(ctx, msg); got.Code != "accepted" {
		t.Fatalf("flag held ordinary message: %+v", got)
	}
	testwait.Until(t, "ordinary message submitted while flagged", func() bool {
		receipt, err := r.s.broker.Status(r.callers[0], msg.ID)
		return err == nil && receipt.Status == couchmessage.Submitted
	})
	for _, child := range children {
		id := child.ID
		testwait.Until(t, "drain notice submitted", func() bool {
			receipt, err := r.s.broker.Status(r.callers[0], id)
			return err == nil && receipt.Status == couchmessage.Submitted
		})
	}
	before := r.world.counts()
	for i := 0; i < 10; i++ {
		if got := r.s.handle(ctx, flagRequest(r.worker, "status")); got.Code != "ok" || len(got.Drain.Rows) != len(r.callers) {
			t.Fatalf("status: %+v", got)
		}
	}
	if got := r.world.counts(); got != before {
		t.Fatalf("status ran authority IO: %v -> %v", before, got)
	}
	r.disconnect(r.worker)
	got = r.s.handle(ctx, flagRequest(r.callers[0], "undrain"))
	if got.Code != "accepted" || got.Operation == nil || len(got.Operation.Drain.Children) != len(r.callers) {
		t.Fatalf("lift failed to clear before refused delivery: %+v", got)
	}
	got = r.s.handle(ctx, flagRequest(r.callers[0], "undrain"))
	if got.Operation == nil || len(got.Operation.Drain.Children) != 0 {
		t.Fatal("empty lift sent unsolicited continuation")
	}
}

func TestDrainFlagsBoundBeforeAnyEffect(t *testing.T) {
	r := newHubMoveRig(t)
	r.s.drainFlags = make(couchcore.DrainFlags)
	for i := 0; i < couchmessage.MaxActors; i++ {
		r.s.drainFlags[fmt.Sprintf("gone:%d", i)] = true
	}
	got := r.s.handle(context.Background(), flagRequest(r.callers[0], "start"))
	if got.Code != "refused" || len(r.s.drainFlags) != couchmessage.MaxActors || len(got.DrainDeliveries) != 0 {
		t.Fatalf("unbounded cohort or effects before refusal: flags=%d result=%+v", len(r.s.drainFlags), got)
	}
}

func TestDrainFlagsRestartIgnoresLegacyFile(t *testing.T) {
	r := newServiceRig(t)
	r.s.drainFlags = couchcore.DrainFlags{"pair:1": true}
	path := filepath.Join(filepath.Dir(r.brokerSock), "drain.json")
	want := []byte(`{"State":"old persisted drain must stay untouched"}`)
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	r.restart(nil)
	got := r.s.handle(context.Background(), couchmessage.Request{Op: "drain", Drain: &couchmessage.DrainRequest{Action: "status"}})
	if got.Code != "ok" || len(got.Drain.Rows) != 0 {
		t.Fatalf("restart restored flags: %+v", got)
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != string(want) {
		t.Fatalf("legacy file changed: %q %v", bytes, err)
	}
}

type countedFlagEndpoint struct {
	serviceEndpointFake
	reserves, deliveries *atomic.Int32
}

func (e countedFlagEndpoint) Reserve(ctx context.Context, _ string, _ uint64) error {
	e.reserves.Add(1)
	return ctx.Err()
}

func (e countedFlagEndpoint) Deliver(ctx context.Context, _ couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.deliveries.Add(1)
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}

func TestDrainFlagFanoutCountsAt8And16(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			r := newServiceRig(t)
			steward := r.connect(0)
			r.s.authority.hub = "/repo0"
			var reserves, deliveries atomic.Int32
			for i := 1; i <= n; i++ {
				b := r.world.add(i)
				r.world.mu.Lock()
				r.world.endpoints[b] = countedFlagEndpoint{reserves: &reserves, deliveries: &deliveries}
				r.world.mu.Unlock()
				r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("fanout-%d", i)))
				r.wrapper(b)
				r.waitConnected(b, true)
			}
			before := r.world.counts()
			got := r.s.handle(context.Background(), flagRequest(steward, "start"))
			if got.Code != "accepted" || got.Operation == nil || len(got.Operation.Drain.Children) != n {
				t.Fatalf("fanout: %+v", got)
			}
			for _, child := range got.Operation.Drain.Children {
				id := child.ID
				testwait.Until(t, "counted notice submitted", func() bool {
					receipt, err := r.s.broker.Status(steward, id)
					return err == nil && receipt.Status == couchmessage.Submitted
				})
			}

			if reserves.Load() != int32(n) || deliveries.Load() != int32(n) {
				t.Fatalf("N=%d reserve=%d deliver=%d", n, reserves.Load(), deliveries.Load())
			}
			after := r.world.counts()
			if after[0] != before[0] || after[1] != before[1] || after[2] != before[2] || after[3]-before[3] > 6*n+4 {
				t.Fatalf("fanout repeated expensive census: N=%d before=%v after=%v", n, before, after)
			}
			for i := 0; i < n; i++ {
				r.s.handle(context.Background(), flagRequest(steward, "status"))
			}
			if r.world.counts() != after {
				t.Fatal("status performed authority IO")
			}
		})
	}
}
