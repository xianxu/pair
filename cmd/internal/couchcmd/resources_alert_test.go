package couchcmd

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"testing"
)

func TestResourceAlertAdvisoryRetriesSameEffectUntilAdmission(t *testing.T) {
	hub := couchmessage.Binding{Slot: "ops:0"}
	state := couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{{Phase: "pending", Effect: &couchcore.ResourceEffect{ID: "same-effect", Slot: "pair:9"}}}}
	sends := 0
	var firstBody string
	d := resourceAlertDispatcher{resolve: func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) { return hub, true }, send: func(_ context.Context, to couchmessage.Binding, id, body string) (couchmessage.Receipt, error) {
		sends++
		if to != hub || id != "same-effect" {
			t.Fatal("changed effect or hub")
		}
		if sends == 1 {
			firstBody = body
			return couchmessage.Receipt{}, couchmessage.ErrUncertain
		}
		if body != firstBody {
			t.Fatal("effect payload changed")
		}
		return couchmessage.Receipt{Message: couchmessage.Message{ID: id, From: to, To: to, Body: body}, Status: couchmessage.Queued}, nil
	}}
	if err := d.dispatch(context.Background(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Episodes[0].Phase != "pending" || state.Episodes[0].Delivery != couchcore.ResourceDeliveryUnknown {
		t.Fatal(state.Episodes)
	}
	if err := d.dispatch(context.Background(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Episodes[0].Phase != "notified" || state.Episodes[0].Delivery != couchcore.ResourceDeliveryAdmitted {
		t.Fatal(state.Episodes)
	}
	_ = d.dispatch(context.Background(), &state)
	if sends != 2 {
		t.Fatal("admitted episode repeated")
	}
}
func TestResourceAlertUnavailableHubRemainsPending(t *testing.T) {
	state := couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{{Phase: "pending", Effect: &couchcore.ResourceEffect{ID: "one"}}}}
	d := resourceAlertDispatcher{resolve: func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) { return couchmessage.Binding{}, false }, send: func(context.Context, couchmessage.Binding, string, string) (couchmessage.Receipt, error) {
		t.Fatal("sent without owning hub")
		return couchmessage.Receipt{}, nil
	}}
	if err := d.dispatch(context.Background(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Episodes[0].Phase != "pending" || state.Episodes[0].Delivery != couchcore.ResourceDeliveryUnknown {
		t.Fatal(state.Episodes)
	}
}
func TestResourceAlertCancelledBeforeSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state := couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{{Phase: "pending", Effect: &couchcore.ResourceEffect{ID: "one"}}}}
	if err := (resourceAlertDispatcher{}).dispatch(ctx, &state); err != context.Canceled {
		t.Fatal(err)
	}
}
