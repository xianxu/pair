package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// Alerts are advisory and live only as long as this supervisor. Pending sends
// reuse their effect ID; the existing broker owns within-run deduplication.
type resourceAlertDispatcher struct {
	permit  func(couchcore.WorkspaceIdentity) (bool, uint64)
	resolve func(couchcore.WorkspaceIdentity) (couchmessage.Binding, bool)
	send    func(context.Context, couchmessage.Binding, string, string) (couchmessage.Receipt, error)
}

func newResourceAlertDispatcher(s *messageService) resourceAlertDispatcher {
	return resourceAlertDispatcher{
		permit: func(id couchcore.WorkspaceIdentity) (bool, uint64) {
			return s.hubAutomationPermit(s.authority.owner(id))
		},
		resolve: func(id couchcore.WorkspaceIdentity) (couchmessage.Binding, bool) {
			return s.hubBindingAt(s.authority.owner(id))
		},
		send: func(ctx context.Context, hub couchmessage.Binding, id, body string) (couchmessage.Receipt, error) {
			c := s.supervision.Load()
			if c == nil {
				return couchmessage.Receipt{}, couchmessage.ErrUnavailable
			}
			return c.sendGenerated(ctx, hub, hub, id, body, false)
		},
	}
}

func (d resourceAlertDispatcher) dispatch(ctx context.Context, state *couchcore.ResourceState) error {
	for i := range state.Episodes {
		e := &state.Episodes[i]
		if e.Phase != "pending" || e.Effect == nil {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		effect := e.Effect
		if d.permit != nil {
			enabled, epoch := d.permit(effect.Workspace)
			if !enabled || e.AutomationEpoch != nil && *e.AutomationEpoch != epoch {
				*e, _ = couchcore.AdvanceResourceEpisode(*e, couchcore.ResourceEpisodeEvent{Delivery: couchcore.ResourceDeliverySuppressed, EffectID: effect.ID})
				continue
			}
			if e.AutomationEpoch == nil {
				e.AutomationEpoch = &epoch
			}
		}
		outcome := couchcore.ResourceDeliveryUnknown
		hub, available := d.resolve(effect.Workspace)
		if available {
			body := resourceAlertBody(effect)
			receipt, err := d.send(ctx, hub, effect.ID, body)
			if err == nil && receipt.Message.ID == effect.ID && receipt.Message.From == hub && receipt.Message.To == hub && receipt.Message.Body == body {
				outcome = couchcore.ResourceDeliveryAdmitted
			} else if errors.Is(err, errHubAutomationOff) {
				outcome = couchcore.ResourceDeliverySuppressed
			} else if errors.Is(err, couchmessage.ErrRecipientBusy) || errors.Is(err, couchmessage.ErrBudgetExhausted) || errors.Is(err, couchmessage.ErrUnavailable) || errors.Is(err, couchmessage.ErrUnsupported) {
				outcome = couchcore.ResourceDeliveryRefused
			}
		}
		*e, _ = couchcore.AdvanceResourceEpisode(*e, couchcore.ResourceEpisodeEvent{Delivery: outcome, EffectID: effect.ID})
	}
	return ctx.Err()
}

// resourceAlertBody is the hub's whole view of one effect, so it carries the
// evidence and the next action. The orphan sweep is the only metric whose rows
// the monitor may end, and ending one always needs confirmation (#583); every
// other metric points at the report alone. Both bodies sanitize host-sourced
// strings: a process name or a cwd is not trusted text.
func resourceAlertBody(effect *couchcore.ResourceEffect) string {
	if effect.Metric == couchcore.ResourceMetricOrphanNvim {
		return fmt.Sprintf("Couch resource monitor: unattributed embedded nvim pid=%d age=%s in %s; no live slot owns it. Inspect with couch --resources; reap with couch --reap-orphans --confirm.",
			effect.Process.PID, effect.Age.Round(time.Minute), rowtext.Sanitize(effect.Root))
	}
	return fmt.Sprintf("Couch resource monitor: %s pid=%d name=%s %s %.3f for %s; sampled coverage. Inspect with couch --resources.",
		rowtext.Sanitize(effect.Slot), effect.Process.PID, rowtext.Sanitize(effect.Name), rowtext.Sanitize(effect.Metric), effect.Value, effect.Duration)
}
