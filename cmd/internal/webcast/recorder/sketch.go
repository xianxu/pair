package recorder

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

// WriteSketch acknowledges the settled actor and sink result, never enqueue
// acceptance. The event is owned before queueing. A canceled queued operation
// cannot execute; once running, acknowledged joins its outcome despite cancel.
func (r *Recorder) WriteSketch(ctx context.Context, in board.SketchWrite) (board.SketchWriteResult, error) {
	out := board.SketchWriteResult{Outcome: board.SketchRefused}
	failure := func(err error) (board.SketchWriteResult, error) { out.Err = err; return out, err }
	if err := ctx.Err(); err != nil {
		return failure(err)
	}
	owned, err := in.Clone()
	if err != nil {
		return failure(err)
	}
	in = owned
	if in.Record.Kind() == "sketch-begin" && r.opts.SyncEvents != nil {
		if err = r.opts.SyncEvents(ctx); err != nil {
			return failure(err)
		}
	}
	err = r.acknowledged(ctx, func() error {
		// The candidate source must include every earlier accepted record.
		// Flush first and treat any sticky failure as unknown persistence.
		r.write()
		if r.sink == nil || !r.opened {
			return board.ErrNotLive
		}
		if r.err != nil {
			duplicate, e := r.m.st.Sketches.Reconcile(in.Record)
			if e != nil {
				return e
			}
			if duplicate {
				out.Outcome = board.SketchUnknown
			}
			return r.err
		}
		prepared, duplicate, e := in.Preflight(r.m.st, r.m.lastKind, r.t(r.opts.Now()))
		out = prepared
		if e != nil {
			return e
		}
		if duplicate {
			return nil
		}
		if e = ctx.Err(); e != nil {
			out.Source = nil
			return e
		}
		if !r.m.Emit(in.Record.Time(), in.Record) {
			out.Source = nil
			return board.ErrShape
		}
		if e = r.write(); e != nil {
			out.Outcome = board.SketchUnknown
			return e
		}
		out.Outcome = board.SketchAccepted
		if in.Accepted != nil {
			in.Accepted()
		}
		return nil
	})
	if err != nil {
		return failure(err)
	}
	return out, nil
}

// ObserveSketch is a writer barrier with owned O(1) receipt/metadata queries.
// Positive reducer evidence is returned with a sticky error when persistence
// is unknown; absence from an unhealthy/closed writer is never proof of refusal.
func (r *Recorder) ObserveSketch(ctx context.Context, sketch, operation string) (board.SketchObservation, error) {
	out := board.SketchObservation{}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if r.opts.SyncEvents != nil {
		if err := r.opts.SyncEvents(ctx); err != nil {
			return out, err
		}
	}
	err := r.acknowledged(ctx, func() error {
		r.write()
		out = board.SketchObservationOf(r.m.st, sketch, operation)
		if r.sink == nil || !r.opened {
			return board.ErrNotLive
		}
		return r.err
	})
	return out, err
}
