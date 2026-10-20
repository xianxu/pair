package couchcmd

import (
	"context"
	"errors"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func (s *messageService) authorizeDrainSteward(ctx context.Context, caller couchmessage.Binding) error {
	id, err := s.connectedWorkspace(ctx, caller)
	if err != nil {
		return err
	}
	main := s.authority.hub
	if s.authority.hubs != nil {
		main = s.authority.hubs.snapshot().Main()
	}
	if main == "" || id.WorktreeRoot != main {
		return errors.New("only the first command-line hub steward may drain or undrain; ask the steward")
	}
	_, err = s.authority.current(ctx, caller)
	return err
}

// drainBoardLocked reads only existing published observations. The flag mutex
// also orders console publication; message delivery never holds this lock.
func (s *messageService) drainBoardLocked() couchcore.DrainBoard {
	observations := map[string]couchcore.DrainObservation{}
	live, connected := s.liveness.Load(), s.connected.Load()
	activity := s.broker.ActivitySnapshot()
	if live != nil && connected != nil {
		for slot, l := range *live {
			if !(*connected)[l.Binding] {
				continue
			}
			observations[slot] = couchcore.DrainObservation{Connected: true, LastOutputAt: activity[l.Binding], Settled: l.Settled, Picker: l.Picker, Background: l.BackgroundWork}
		}
	}
	now := time.Now()
	if s.authority.now != nil {
		now = s.authority.now()
	}
	return couchcore.ProjectDrainBoard(s.drainFlags, observations, now)
}

func (s *messageService) handleDrain(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if r.Drain.Action == "status" {
		s.drainMu.Lock()
		board := s.drainBoardLocked()
		s.drainMu.Unlock()
		return couchmessage.Response{Code: "ok", Drain: &board}
	}
	caller, err := s.switchCaller(ctx, r)
	if err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	if err = s.authorizeDrainSteward(ctx, caller); err != nil {
		return couchmessage.Response{Code: "refused", Error: err.Error()}
	}
	body := couchcore.DrainNotice(r.Drain.Fast)
	var targets []string
	s.drainMu.Lock()
	if r.Drain.Action == "start" {
		for _, b := range s.broker.Connected() {
			if b.Slot != caller.Slot {
				targets = append(targets, b.Slot)
			}
		}
		next := couchcore.AddDrainFlags(s.drainFlags, targets, caller.Slot)
		if len(next) > couchmessage.MaxActors {
			s.drainMu.Unlock()
			return couchmessage.Response{Code: "refused", Error: "drain flag limit reached; inspect status and undrain before selecting another cohort"}
		}
		s.drainFlags = next
	} else {
		for slot := range s.drainFlags {
			targets = append(targets, slot)
		}
		s.drainFlags = couchcore.ClearDrainFlags(s.drainFlags, nil)
		// Validation already checked the note before any flag changes.
		body, _ = couchcore.UndrainNotice(r.Drain.Message)
	}
	board := s.drainBoardLocked()
	if s.authority.drainPublish != nil {
		s.authority.drainPublish(board)
	}
	s.drainMu.Unlock()
	// Admission is bounded as one command; each target gets ordinary receipts.
	admission, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result := couchmessage.Response{Code: "ok", Drain: &board}
	for _, target := range targets {
		delivery := couchmessage.DrainDelivery{Workspace: target}
		id, err := newRequestID()
		if err == nil {
			receipt, sendErr := s.broker.SendSystem(admission, caller, id, couchmessage.Route{Target: target}, body)
			err = sendErr
			if receipt.Message.ID != "" {
				delivery.Receipt = &receipt
			}
		}
		if err != nil {
			delivery.Error = err.Error()
		}
		result.DrainDeliveries = append(result.DrainDeliveries, delivery)
	}
	return result
}
