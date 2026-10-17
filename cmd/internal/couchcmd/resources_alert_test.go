package couchcmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// orphanNvimAlertState is one pending orphan-nvim episode as the reduce hands it
// to the dispatcher: the effect is immutable evidence bound when it fired (#583).
func orphanNvimAlertState(at time.Time, root string) couchcore.ResourceState {
	e, _ := couchcore.AdvanceResourceEpisode(
		couchcore.ResourceEpisode{Key: "nvim/4242/1760000000.5", Metric: "orphan-nvim"},
		couchcore.ResourceEpisodeEvent{At: at, Known: true, High: true, RequiredSamples: 1, Effect: couchcore.ResourceEffect{
			Name:    "/opt/homebrew/bin/nvim",
			Process: couchcore.ResourceProcessKey{PID: 4242, Identity: "1760000000.5"},
			Age:     13*24*time.Hour + 5*time.Minute + 20*time.Second,
			Root:    root,
		}})
	return couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{e}}
}

// The root is a path read off the host, so it reaches the hub's terminal through
// the same sanitizer as every other label.
func TestResourceAlertSanitizesTheOrphanRoot(t *testing.T) {
	hub := couchmessage.Binding{Slot: "ops:0"}
	state := orphanNvimAlertState(time.Now(), "unsafe\x1b[2J\r\nforged")
	var body string
	d := resourceAlertDispatcher{resolve: func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) { return hub, true }, send: func(_ context.Context, to couchmessage.Binding, id, text string) (couchmessage.Receipt, error) {
		body = text
		return couchmessage.Receipt{Message: couchmessage.Message{ID: id, From: to, To: to, Body: text}, Status: couchmessage.Queued}, nil
	}}
	if err := d.dispatch(context.Background(), &state); err != nil {
		t.Fatal(err)
	}
	if body == "" || strings.ContainsAny(body, "\x1b\r") || strings.Contains(body, "\nforged") {
		t.Fatalf("terminal injection: %q", body)
	}
	if !strings.Contains(body, "pid=4242") || !strings.Contains(body, "couch --reap-orphans --confirm") {
		t.Fatalf("body lost the evidence or the gate: %q", body)
	}
	if state.Episodes[0].Phase != "notified" {
		t.Fatal(state.Episodes[0])
	}
}

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
