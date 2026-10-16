package couchcmd

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestResourceAlertFreezesAndDoesNotRepeatUncertainAttempt(t *testing.T) {
	hub := couchmessage.Binding{Slot: "ops:0"}
	sends, writes := 0, 0
	d := resourceAlertDispatcher{
		resolve:   func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) { return hub, true },
		connected: func(couchmessage.Binding) bool { return true },
		send: func(_ context.Context, a resourceAlertAttempt) (couchmessage.Receipt, error) {
			sends++
			if writes == 0 {
				t.Fatal("sent before checkpoint")
			}
			return couchmessage.Receipt{}, couchmessage.ErrUncertain
		},
		lookup: func(context.Context, resourceAlertAttempt) (couchmessage.Receipt, bool) {
			return couchmessage.Receipt{}, false
		},
	}
	s := couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{{Key: "rss", Phase: "pending", Effect: &couchcore.ResourceEffect{ID: "one", Metric: "process-rss", Slot: "pair:1", Value: 9}}}}
	cp := resourceCheckpoint{Version: 1, State: s}
	persist := func(resourceCheckpoint) error { writes++; return nil }
	if err := d.dispatch(context.Background(), &cp, persist); err != nil {
		t.Fatal(err)
	}
	if sends != 1 || len(cp.Attempts) != 1 || cp.Attempts[0].State != "unknown" {
		t.Fatalf("%d %+v", sends, cp)
	}
	old := cp.Attempts[0]
	hub.Slot = "other:0"
	cp.State.Episodes[0].Effect.Value = 12
	if err := d.dispatch(context.Background(), &cp, persist); err != nil {
		t.Fatal(err)
	}
	if sends != 1 || cp.Attempts[0] != old {
		t.Fatalf("uncertain envelope changed or resent: %d %+v", sends, cp.Attempts)
	}
}

func TestResourceAlertCheckpointFailurePreventsSend(t *testing.T) {
	d := resourceAlertDispatcher{resolve: func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) {
		return couchmessage.Binding{Slot: "ops:0"}, true
	}, send: func(context.Context, resourceAlertAttempt) (couchmessage.Receipt, error) {
		t.Fatal("sent without durable intent")
		return couchmessage.Receipt{}, nil
	}}
	cp := resourceCheckpoint{Version: 1, State: couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{{Phase: "pending", Effect: &couchcore.ResourceEffect{ID: "one"}}}}}
	if err := d.dispatch(context.Background(), &cp, func(resourceCheckpoint) error { return errors.New("disk full") }); err == nil {
		t.Fatal("ignored failed intent persistence")
	}
}
func TestResourceAlertDispatcherProjectsPureDeliveryOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, attempt string
		sendErr       error
		receipt       bool
		want          couchcore.ResourceDeliveryOutcome
		phase         string
	}{{"admission", "", nil, true, couchcore.ResourceDeliveryAdmitted, "notified"}, {"refusal", "", couchmessage.ErrUnavailable, false, couchcore.ResourceDeliveryRefused, "pending"}, {"uncertainty", "", couchmessage.ErrUncertain, false, couchcore.ResourceDeliveryUnknown, "pending"}, {"restart receipt", "attempting", nil, true, couchcore.ResourceDeliveryAdmitted, "notified"}, {"restart missing receipt", "attempting", nil, false, couchcore.ResourceDeliveryUnknown, "pending"}, {"persisted admission", "admitted", nil, false, couchcore.ResourceDeliveryAdmitted, "notified"}} {
		t.Run(tc.name, func(t *testing.T) {
			hub := couchmessage.Binding{Slot: "ops:0"}
			cp := resourceCheckpoint{Version: 1, State: couchcore.ResourceState{Episodes: []couchcore.ResourceEpisode{{Phase: "pending", Effect: &couchcore.ResourceEffect{ID: "effect"}}}}}
			if tc.attempt != "" {
				cp.Attempts = []resourceAlertAttempt{{ID: "effect", From: hub, To: hub, Body: "Couch resource monitor: fixed", State: tc.attempt}}
			}
			receipt := func(a resourceAlertAttempt) couchmessage.Receipt {
				return couchmessage.Receipt{Message: couchmessage.Message{ID: a.ID, From: a.From, To: a.To, Body: a.Body}, Status: couchmessage.Queued}
			}
			d := resourceAlertDispatcher{resolve: func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) { return hub, true }, connected: func(couchmessage.Binding) bool { return true }, send: func(_ context.Context, a resourceAlertAttempt) (couchmessage.Receipt, error) {
				if tc.attempt != "" {
					t.Fatal("reconciliation sent again")
				}
				if tc.receipt {
					return receipt(a), nil
				}
				return couchmessage.Receipt{}, tc.sendErr
			}, lookup: func(_ context.Context, a resourceAlertAttempt) (couchmessage.Receipt, bool) {
				return receipt(a), tc.receipt
			}}
			if err := d.dispatch(context.Background(), &cp, func(resourceCheckpoint) error { return nil }); err != nil {
				t.Fatal(err)
			}
			e := cp.State.Episodes[0]
			if e.Phase != tc.phase || e.Delivery != tc.want {
				t.Fatalf("delivery outcome not owned by reducer: %+v", e)
			}
		})
	}
}
