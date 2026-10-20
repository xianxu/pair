package couchcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func flagRequest(b couchmessage.Binding, action string) couchmessage.Request {
	r := callerRequest(b, "drain")
	r.Drain = &couchmessage.DrainRequest{Action: action}
	if action != "status" {
		r.ID = "00000000-0000-4000-8000-000000000001"
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
	if got.Code != "ok" || got.Drain == nil || len(got.Drain.Rows) != len(r.callers) || len(got.DrainDeliveries) != len(r.callers) {
		t.Fatalf("drain: %+v", got)
	}
	for _, row := range got.Drain.Rows {
		if row.Workspace == r.callers[0].Slot {
			t.Fatal("issuing steward flagged")
		}
	}
	for _, delivery := range got.DrainDeliveries {
		if delivery.Error != "" || delivery.Receipt == nil || delivery.Receipt.Message.Origin != couchmessage.OriginSystem {
			t.Fatalf("not ordinary authenticated system delivery: %+v", delivery)
		}
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
	for _, delivery := range got.DrainDeliveries {
		id := delivery.Receipt.Message.ID
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
	if got.Code != "ok" || len(got.Drain.Rows) != 0 || len(got.DrainDeliveries) != len(r.callers) {
		t.Fatalf("lift failed to clear before refused delivery: %+v", got)
	}
	got = r.s.handle(ctx, flagRequest(r.callers[0], "undrain"))
	if len(got.DrainDeliveries) != 0 {
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
