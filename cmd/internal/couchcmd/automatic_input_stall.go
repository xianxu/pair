package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type automaticInputStallRecord struct {
	Binding  couchmessage.Binding
	Sequence uint64
}

// handleAutomaticInputStall grants only this typed, supervisor-composed notice.
// Reserve the sequence before any effects: losing an IO reply cannot replay it.
func (s *messageService) handleAutomaticInputStall(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	report, _ := couchmessage.ParseAutomaticInputStall(r.Body)
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if r.ID != couchmessage.AutomaticInputStallID(caller, report.Sequence) {
		return couchmessage.Response{Code: "invalid-request", Error: "automatic input stall ID does not match caller and sequence"}
	}
	if !s.isConnected(caller) {
		return couchmessage.Response{Code: "refused", Error: "automatic input stall caller disconnected"}
	}
	if err := ctx.Err(); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	s.stalls.Lock()
	// Recheck the in-memory incarnation under the reservation lock. Authority IO
	// happened immediately before this lock; no IO runs while it is held.
	if !s.isConnected(caller) {
		s.stalls.Unlock()
		return couchmessage.Response{Code: "refused", Error: "automatic input stall caller disconnected"}
	}
	previous, exists := s.stalls.records[caller.Slot]
	if exists && previous.Binding == caller && report.Sequence <= previous.Sequence {
		s.stalls.Unlock()
		return couchmessage.Response{Code: "ok"}
	}
	if !exists && len(s.stalls.records) >= couchmessage.MaxActors {
		s.stalls.Unlock()
		return couchmessage.Response{Code: "refused", Error: "automatic input stall occurrence capacity reached"}
	}
	if s.stalls.records == nil {
		s.stalls.records = map[string]automaticInputStallRecord{}
	}
	s.stalls.records[caller.Slot] = automaticInputStallRecord{Binding: caller, Sequence: report.Sequence}
	s.stalls.Unlock()
	// Bind the reserved occurrence before status/desktop effects can block.
	// Its verified owning hub and policy epoch come from memory only.
	permit := s.captureOwnerHubInputPermit(caller)
	outcome := "input still blocked"
	if report.Released {
		outcome = "lease released"
	}
	body := fmt.Sprintf("Slot %s: unexpected automatic-input stall; %s; cause %s; wait %d ms; fence %d, epoch %d. Inspect the slot's retained automatic-input-stall diagnostic (sequence %d).", caller.Slot, outcome, report.Cause, report.WaitMillis, report.Fence, report.Epoch, report.Sequence)
	if report.Cause == couchmessage.AutomaticInputStallUnsubmittedComposer {
		body = fmt.Sprintf("%s's composer holds an unsubmitted automatic message; %s. %s", caller.Slot, couchmessage.AutomaticInputRecoveryHint, body)
	}
	var failures []string
	if c := s.supervision.Load(); c != nil {
		if c.notice != nil {
			c.notice(body)
		}
		if permit.current(s) {
			if hub, ok := s.hubBindingFor(caller); ok {
				if hub.Slot != caller.Slot {
					if _, err := c.sendGenerated(ctx, caller, hub, r.ID, body, false, permit); err != nil && !errors.Is(err, errHubAutomationOff) {
						failures = append(failures, "hub notice: "+err.Error())
					}
				}
			} else {
				failures = append(failures, "owning hub unavailable")
			}
		}
	} else {
		failures = append(failures, "supervision output unavailable")
	}
	// Hub admission and desktop output are independent attempts. Never retry an
	// uncertain desktop write or discard duplicate evidence on disconnect.
	if write := s.desktopNotify.Load(); write != nil && *write != nil {
		if err := (*write)(ctx, "Couch automatic input", body); err != nil {
			failures = append(failures, "desktop notification: "+err.Error())
		}
	} else {
		failures = append(failures, "desktop notification unavailable")
	}
	if len(failures) > 0 {
		return couchmessage.Response{Code: "uncertain", Error: strings.Join(failures, "; ")}
	}
	return couchmessage.Response{Code: "ok"}
}
