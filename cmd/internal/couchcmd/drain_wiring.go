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
	board := couchcore.ProjectDrainBoard(s.drainFlags, observations, now)
	notices := s.drainNoticeSnapshotLocked()
	for i := range board.Rows {
		row := &board.Rows[i]
		notice := notices[row.Workspace]
		row.Notice, row.NoticeID, row.OperationID, row.NoticeDetail = notice.state, notice.id, notice.operation, notice.detail
	}
	return board
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
	defer s.drainMu.Unlock()
	if replay, held := s.drainReplayLocked(caller, r); held {
		return replay
	}
	next := s.drainFlags
	if r.Drain.Action == "start" {
		for _, b := range s.broker.Connected() {
			if b.Slot != caller.Slot {
				targets = append(targets, b.Slot)
			}
		}
		next = couchcore.AddDrainFlags(s.drainFlags, targets, caller.Slot)
		if len(next) > couchmessage.MaxActors {
			return couchmessage.Response{Code: "refused", Error: "drain flag limit reached; inspect status and undrain before selecting another cohort"}
		}
	} else {
		for slot := range s.drainFlags {
			targets = append(targets, slot)
		}
		next = couchcore.ClearDrainFlags(s.drainFlags, nil)
		body, _ = couchcore.UndrainNotice(r.Drain.Message)
	}
	response, created := s.registerDrainLocked(ctx, caller, r, targets, body)
	if !created {
		return response
	}
	s.drainFlags = next
	// Keep only current flagged-slot references; lift receipts live in their
	// registered parent rather than accumulating a second delivery history.
	for slot := range s.drainNotices {
		if !s.drainFlags[slot] {
			delete(s.drainNotices, slot)
		}
	}
	board := s.drainBoardLocked()
	if s.authority.drainPublish != nil {
		s.authority.drainPublish(board)
	}
	return response
}
