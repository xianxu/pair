package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type resourceAlertAttempt struct {
	ID    string               `json:"id"`
	From  couchmessage.Binding `json:"from"`
	To    couchmessage.Binding `json:"to"`
	Body  string               `json:"body"`
	State string               `json:"state"` // attempting, refused, unknown, admitted
}
type resourceCheckpoint struct {
	Version  int                     `json:"version"`
	State    couchcore.ResourceState `json:"state"`
	Attempts []resourceAlertAttempt  `json:"attempts"`
}

func validateResourceCheckpoint(raw []byte) error {
	var cp resourceCheckpoint
	if err := strictjson.Decode(raw, &cp); err != nil {
		return err
	}
	if cp.Version != 1 || len(cp.Attempts) > couchcore.ResourceMaxProcesses*2+couchcore.ResourceMaxSlots*2 {
		return errors.New("invalid resource checkpoint schema/count")
	}
	if err := couchcore.ValidateResourceState(cp.State); err != nil {
		return err
	}
	active := map[string]bool{}
	for _, e := range cp.State.Episodes {
		if e.Effect != nil && (e.Phase == "pending" || e.Phase == "notified") {
			active[e.Effect.ID] = true
		}
	}
	seen := map[string]bool{}
	for _, a := range cp.Attempts {
		if a.ID == "" || len(a.ID) > 128 || seen[a.ID] || !active[a.ID] || len(a.Body) > 8192 || !strings.HasPrefix(a.Body, "Couch resource monitor:") || a.From != a.To {
			return errors.New("invalid resource alert envelope")
		}
		if err := couchmessage.ValidateBody(a.Body); err != nil {
			return err
		}
		if err := a.From.Validate(); err != nil {
			return err
		}
		switch a.State {
		case "attempting", "refused", "unknown", "admitted":
		default:
			return errors.New("invalid resource delivery state")
		}
		seen[a.ID] = true
	}
	return nil
}

// The persisted immutable envelope precedes every send. Unknown admission is
// never retried merely because receipt retention ended, including after restart.
type resourceAlertDispatcher struct {
	resolve   func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool)
	connected func(couchmessage.Binding) bool
	send      func(context.Context, resourceAlertAttempt) (couchmessage.Receipt, error)
	lookup    func(context.Context, resourceAlertAttempt) (couchmessage.Receipt, bool)
}

func newResourceAlertDispatcher(s *messageService) resourceAlertDispatcher {
	return resourceAlertDispatcher{
		resolve: func(id couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) {
			return s.hubBindingAt(s.authority.owner(id))
		},
		connected: s.isConnected,
		send: func(ctx context.Context, a resourceAlertAttempt) (couchmessage.Receipt, error) {
			c := s.supervision.Load()
			if c == nil {
				return couchmessage.Receipt{}, couchmessage.ErrUnavailable
			}
			return c.sendGenerated(ctx, a.From, a.To, a.ID, a.Body, false)
		},
		lookup: func(ctx context.Context, a resourceAlertAttempt) (couchmessage.Receipt, bool) {
			if r, ok, _ := s.broker.ReceiptSnapshot(a.ID); ok {
				return r, true
			}
			if !s.isConnected(a.From) {
				return couchmessage.Receipt{}, false
			}
			r, err := s.broker.StatusContext(ctx, a.From, a.ID)
			return r, err == nil
		},
	}
}
func resourceReceiptMatches(r couchmessage.Receipt, a resourceAlertAttempt) bool {
	return r.Message.ID == a.ID && r.Message.From == a.From && r.Message.To == a.To && r.Message.Body == a.Body
}
func (d resourceAlertDispatcher) dispatch(ctx context.Context, cp *resourceCheckpoint, save func(resourceCheckpoint) error) error {
	active := map[string]bool{}
	for _, e := range cp.State.Episodes {
		if e.Effect != nil && (e.Phase == "pending" || e.Phase == "notified") {
			active[e.Effect.ID] = true
		}
	}
	kept := make([]resourceAlertAttempt, 0, len(cp.Attempts))
	for _, a := range cp.Attempts {
		if active[a.ID] {
			kept = append(kept, a)
		}
	}
	cp.Attempts = kept
	for i := range cp.State.Episodes {
		e := &cp.State.Episodes[i]
		if e.Phase != "pending" || e.Effect == nil {
			continue
		}
		delivery := func(outcome couchcore.ResourceDeliveryOutcome) {
			*e, _ = couchcore.AdvanceResourceEpisode(*e, couchcore.ResourceEpisodeEvent{Delivery: outcome, EffectID: e.Effect.ID})
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		at := -1
		for j := range cp.Attempts {
			if cp.Attempts[j].ID == e.Effect.ID {
				at = j
				break
			}
		}
		fresh := at < 0
		if fresh {
			hub, ok := d.resolve(e.Effect.Workspace)
			if !ok {
				delivery(couchcore.ResourceDeliveryUnknown)
				continue
			}
			body := fmt.Sprintf("Couch resource monitor: %s pid=%d name=%s %s %.3f for %s; sampled coverage. Inspect with couch --resources.", rowtext.Sanitize(e.Effect.Slot), e.Effect.Process.PID, rowtext.Sanitize(e.Effect.Name), rowtext.Sanitize(e.Effect.Metric), e.Effect.Value, e.Effect.Duration)
			cp.Attempts = append(cp.Attempts, resourceAlertAttempt{ID: e.Effect.ID, From: hub, To: hub, Body: body, State: "attempting"})
			at = len(cp.Attempts) - 1
		}
		a := &cp.Attempts[at]
		if a.State == "admitted" {
			delivery(couchcore.ResourceDeliveryAdmitted)
			continue
		}
		if !fresh && a.State != "refused" {
			if d.lookup != nil {
				if r, ok := d.lookup(ctx, *a); ok && resourceReceiptMatches(r, *a) {
					a.State = "admitted"
					delivery(couchcore.ResourceDeliveryAdmitted)
					continue
				}
			}
			a.State = "unknown"
			delivery(couchcore.ResourceDeliveryUnknown)
			continue
		}
		currentHub, currentOwnerKnown := d.resolve(e.Effect.Workspace)
		if !fresh && (!currentOwnerKnown || currentHub != a.From || d.connected == nil || !d.connected(a.From)) {
			a.State = "unknown"
			delivery(couchcore.ResourceDeliveryUnknown)
			continue
		}
		a.State = "attempting"
		if err := save(*cp); err != nil {
			a.State = "refused"
			delivery(couchcore.ResourceDeliveryRefused) // No broker call occurred; retry the same envelope after storage recovers.
			return err
		}
		r, err := d.send(ctx, *a)
		if err == nil && resourceReceiptMatches(r, *a) {
			a.State = "admitted"
			delivery(couchcore.ResourceDeliveryAdmitted)
		} else {
			a.State = "unknown"
			delivery(couchcore.ResourceDeliveryUnknown)
			if errors.Is(err, couchmessage.ErrRecipientBusy) || errors.Is(err, couchmessage.ErrBudgetExhausted) || errors.Is(err, couchmessage.ErrUnavailable) || errors.Is(err, couchmessage.ErrUnsupported) {
				a.State = "refused"
				delivery(couchcore.ResourceDeliveryRefused)
			}
		}
		if err := save(*cp); err != nil {
			return err
		}
	}
	return nil
}

func encodeResourceCheckpoint(cp resourceCheckpoint) ([]byte, error) {
	raw, err := json.Marshal(cp)
	if err != nil {
		return nil, err
	}
	if len(raw) > couchcore.ResourceCheckpointLimit {
		return nil, errors.New("resource checkpoint exceeds byte cap")
	}
	return raw, validateResourceCheckpoint(raw)
}
