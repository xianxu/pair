package recorder

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type preparedNoteWrite struct {
	ctx         context.Context
	admission   board.AnnotationAdmission
	reservation board.NoteReservation
	check       func() error
	settle      func(board.OperatorOutcome, error)
}

// WritePrepared queues an immutable reserved note. A returned error proves no
// dispatch and does not invoke settle. Otherwise settle runs exactly once on
// the recorder actor, including when shutdown drains the queue. It must not
// block or synchronously call the recorder. check is an optional actor-local
// identity check for the operator surface; it must not acquire a command lock.
func (r *Recorder) WritePrepared(ctx context.Context, admission board.AnnotationAdmission, reservation board.NoteReservation, check func() error, settle func(board.OperatorOutcome, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if settle == nil {
		return board.ErrShape
	}
	reservation.Note = *clonePreparedRecord(&reservation.Note)
	if !r.enqueue(preparedNoteWrite{ctx, admission, reservation, check, settle}, false) {
		return errClosed
	}
	return nil
}

func (r *Recorder) writePrepared(at time.Time, in preparedNoteWrite) {
	refuse := func(err error) { in.settle(board.OperatorRefused, err) }
	if err := in.ctx.Err(); err != nil {
		refuse(err)
		return
	}
	if r.sink == nil || !r.opened {
		refuse(board.ErrNotLive)
		return
	}
	if r.err != nil {
		refuse(r.err)
		return
	}
	if !r.m.st.OnDoc(in.reservation.Document) {
		err := board.ErrStale
		if r.m.st.Paused() && r.m.st.DocSeg == in.reservation.Document {
			err = board.ErrPaused
		}
		refuse(err)
		return
	}
	if !r.pageFeedbackReady(&in.reservation.Note, in.reservation.Document) {
		refuse(board.ErrStale)
		return
	}
	if err := in.admission.Check(); err != nil {
		refuse(err)
		return
	}
	if in.check != nil {
		if err := in.check(); err != nil {
			refuse(err)
			return
		}
	}
	if err := in.ctx.Err(); err != nil {
		refuse(err)
		return
	}
	if !r.m.Emit(r.t(at), &in.reservation.Note) {
		refuse(board.ErrShape)
		return
	}
	if err := r.write(); err != nil {
		in.settle(board.OperatorUnknown, err)
		return
	}
	in.settle(board.OperatorAccepted, nil)
}

// ObserveNote fences preceding CDP events and recorder writes. A healthy writer
// may prove absence; a failed/closed writer cannot. Results own their data, and
// cancellation never leaves the actor writing into the caller's stack.
func (r *Recorder) ObserveNote(ctx context.Context, id string) (*rec.Note, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if r.opts.SyncEvents != nil {
		if err := r.opts.SyncEvents(ctx); err != nil {
			return nil, false, err
		}
	}
	type result struct {
		note *rec.Note
		err  error
	}
	ch := make(chan result, 1)
	q := query{done: make(chan struct{}), fn: func() {
		if r.err != nil {
			ch <- result{err: r.err}
			return
		}
		if r.sink == nil || !r.opened {
			ch <- result{err: board.ErrNotLive}
			return
		}
		for _, n := range r.m.st.Notes {
			if n.Note.ID == id {
				ch <- result{note: clonePreparedRecord(n.Note)}
				return
			}
		}
		ch <- result{}
	}}
	if !r.enqueue(q, false) {
		return nil, false, errClosed
	}
	select {
	case v := <-ch:
		return v.note, v.note != nil, v.err
	case <-ctx.Done():
		return nil, false, ctx.Err()
	case <-r.done:
		return nil, false, errClosed
	}
}

func clonePreparedRecord(n *rec.Note) *rec.Note {
	v := *n
	v.At.Path = slices.Clone(n.At.Path)
	v.At.Classes = slices.Clone(n.At.Classes)
	v.At.Stable = maps.Clone(n.At.Stable)
	if n.At.Box != nil {
		box := *n.At.Box
		v.At.Box = &box
	}
	if n.Region != nil {
		box := *n.Region
		v.Region = &box
	}
	return &v
}
