package notedraft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

// Save attaches frozen evidence through an acknowledged writer. A network
// request ending does not turn an uncertain write into an absent write.
func (s *Service) Save(ctx context.Context, owner Owner, request, draft string, a board.AnnotationAdmission, author board.Viewer, color int, text string, check func() error) (Status, error) {
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	if err := a.Check(); err != nil {
		return Status{}, err
	}
	if s.opts.Write == nil {
		return Status{}, errors.New("capture note writer unavailable")
	}
	payloadBytes, _ := json.Marshal(struct {
		Author board.Viewer
		Color  int
		Text   string
	}{author, color, text})
	payload := string(payloadBytes)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return Status{}, errors.New("cast has ended")
	}
	var r *captureRecord
	if strings.HasPrefix(draft, "u.") {
		decoded, err := s.untoken(owner, draft)
		if err != nil {
			s.mu.Unlock()
			return Status{}, err
		}
		r = s.records[decoded.ID]
		if r == nil {
			prepared, err := a.RestorePreparedNote(decoded.Prepared, decoded.Operator)
			if err != nil {
				s.mu.Unlock()
				return Status{}, err
			}
			decoded.Prepared = prepared
			decoded.admission = a
			decoded.Ephemeral = true
			r = decoded
		}
	} else {
		r = s.findLocked(owner, draft)
	}
	if r == nil || r.Owner != owner {
		s.mu.Unlock()
		return Status{}, errors.New("capture draft not found")
	}
	if r.SaveRequest != "" {
		if r.SaveRequest != request || r.SavePayload != payload {
			s.mu.Unlock()
			return Status{}, errors.New("save already started; retry Save note to check")
		}
		if r.State.Phase == Attaching {
			id := r.ID
			s.mu.Unlock()
			if err := s.reconcileWrite(ctx, id); err != nil {
				return Status{}, err
			}
			return s.Status(owner, id)
		}
		status := s.statusLocked(r)
		s.mu.Unlock()
		return status, nil
	}
	if r.State.Phase != Ready {
		s.mu.Unlock()
		return Status{}, fmt.Errorf("capture is %s", r.State.Phase)
	}
	if !s.opts.Now().Before(r.Expires) {
		s.mu.Unlock()
		return Status{}, errors.New("capture draft expired; draw the note again")
	}
	if request == "" || len(request) > 128 {
		s.mu.Unlock()
		return Status{}, errors.New("invalid save request")
	}
	reservation, err := a.ReservePreparedNote(r.Prepared, author, color, text, r.ID, r.Operator)
	if err != nil {
		s.mu.Unlock()
		return Status{}, err
	}
	reservation.Note.CropUnavailable = r.State.Unavailable
	r.NoteID = reservation.Note.ID
	r.Note = &reservation.Note
	r.SaveRequest = request
	r.SavePayload = payload
	r.Problem = ""
	if err = s.applyLocked(r, Event{Kind: Save}); err != nil {
		if s.opts.Release != nil {
			s.opts.Release(reservation.Note.ID)
		}
		r.NoteID = ""
		r.Note = nil
		r.SaveRequest = ""
		r.SavePayload = ""
		s.mu.Unlock()
		return Status{}, err
	}
	s.records[r.ID] = r
	id := r.ID
	s.mu.Unlock()
	opctx, cancel := context.WithTimeout(s.ctx, OperationBudget)
	settle := func(outcome board.OperatorOutcome, writeErr error) {
		cancel()
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.records[id]
		if current == nil || current.State.Phase != Attaching {
			return
		}
		switch outcome {
		case board.OperatorAccepted:
			current.Problem = ""
			_ = s.applyLocked(current, Event{Kind: WriteConfirmed})
		case board.OperatorRefused:
			if s.opts.Release != nil {
				s.opts.Release(current.NoteID)
			}
			problem := "note was not saved; retry Save note"
			if writeErr != nil {
				problem = writeErr.Error()
			}
			_ = s.absentWriteLocked(current, problem)
		case board.OperatorUnknown:
			current.Problem = "save not confirmed; retry Save note to check"
			if writeErr != nil {
				current.Problem = writeErr.Error() + "; " + current.Problem
			}
			// Persist the reason without converting an uncertain write to absence.
			_ = s.persistLocked(current)
		}
	}
	if err = s.opts.Write(opctx, a, reservation, check, settle); err != nil {
		settle(board.OperatorRefused, err)
		return Status{}, err
	}
	return s.Status(owner, id)
}

func (s *Service) reconcileWrite(ctx context.Context, id string) error {
	s.mu.Lock()
	r := s.records[id]
	if r == nil || r.State.Phase != Attaching {
		s.mu.Unlock()
		return nil
	}
	noteID := r.NoteID
	s.mu.Unlock()
	if s.opts.Observe == nil {
		return errors.New("save not confirmed; retry Save note to check")
	}
	observed, present, err := s.opts.Observe(ctx, noteID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r = s.records[id]
	if r == nil || r.State.Phase != Attaching {
		return nil
	}
	if present {
		if observed == nil || observed.ID != r.NoteID || observed.Capture != r.ID {
			return errors.New("recorded note differs from capture intent")
		}
		problem := r.Problem
		r.Problem = ""
		if err := s.applyLocked(r, Event{Kind: WriteConfirmed}); err != nil {
			r.Problem = problem
			return err
		}
		return nil
	}
	if s.opts.Release != nil {
		s.opts.Release(r.NoteID)
	}
	return s.absentWriteLocked(r, "note was not saved; retry Save note")
}

func (s *Service) absentWriteLocked(r *captureRecord, problem string) error {
	before := *r
	r.SaveRequest = ""
	r.SavePayload = ""
	r.NoteID = ""
	r.Note = nil
	r.Problem = problem
	if err := s.applyLocked(r, Event{Kind: WriteAbsent}); err != nil {
		*r = before
		return err
	}
	return nil
}

// Resolve is called only after an actual resolved/clear recording event. It
// never removes the durable submission bundle that may own these pixels.
func (s *Service) Resolve(noteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.records {
		if r.NoteID == noteID {
			return s.applyLocked(r, Event{Kind: Resolve})
		}
	}
	return nil
}
